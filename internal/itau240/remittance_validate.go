package itau240

import (
	"errors"
	"fmt"
	"strings"

	"github.com/cnab-dev/cnab-payment/internal/core"
	"github.com/cnab-dev/cnab-payment/internal/pix"
	"github.com/cnab-dev/cnab-payment/internal/taxid"
)

// validatePayment checks everything the Itaú layout needs to write p: the
// payer account is an Itaú account, dates and amount are present, tax IDs
// are valid, and the kind-specific fields can be encoded. Every problem is
// reported, one *core.PaymentError per field, joined.
func validatePayment(p core.Payment) error {
	v := &validator{}
	b := p.Base()

	if b.Amount <= 0 {
		v.invalid("Amount", "must be positive")
	}
	if b.DueDate.IsZero() {
		v.invalid("DueDate", "required")
	}
	if b.SettlementDate.IsZero() {
		v.invalid("SettlementDate", "required")
	}
	v.payerAccount(b.PayerAccount)
	v.optionalTaxID("Payer.TaxID", b.Payer.TaxID)
	v.optionalTaxID("Payee.TaxID", b.Payee.TaxID)
	if b.Debtor != nil {
		v.optionalTaxID("Debtor.TaxID", b.Debtor.TaxID)
	}

	switch p := p.(type) {
	case core.BarcodePayment:
		v.barcode(p.Barcode)
	case core.QRCodePayment:
		v.qrCode(p)
	case core.TransferPayment:
		v.transfer(p)
	}
	return errors.Join(v.errs...)
}

type validator struct {
	errs []error
}

func (v *validator) add(field string, err error) {
	v.errs = append(v.errs, &core.PaymentError{Field: field, Err: err})
}

func (v *validator) invalid(field, format string, args ...any) {
	v.add(field, fmt.Errorf("%w: %s", core.ErrInvalidField, fmt.Sprintf(format, args...)))
}

func (v *validator) payerAccount(a core.BankAccount) {
	if a.BankCode != bankCode {
		v.add("PayerAccount.BankCode", fmt.Errorf("%w: payer account is at bank %q, the layout is for bank %s",
			core.ErrLayoutMismatch, a.BankCode, bankCode))
	}
	if !isAllDigits(a.Branch) {
		v.invalid("PayerAccount.Branch", "must be non-empty digits")
	}
	if !isAllDigits(a.Number) {
		v.invalid("PayerAccount.Number", "must be non-empty digits")
	}
	if len(a.CheckDigit) != 1 {
		v.invalid("PayerAccount.CheckDigit", "must be exactly one character")
	}
}

func (v *validator) optionalTaxID(field, id string) {
	if id == "" {
		return
	}
	if err := taxid.Validate(id); err != nil {
		v.invalid(field, "%v", err)
	}
}

func (v *validator) barcode(barcode string) {
	switch {
	case len(barcode) != 44 || !isAllDigits(barcode):
		v.invalid("Barcode", "must be exactly 44 digits")
	case strings.HasPrefix(barcode, "8") && !isDARFDCTFWebBarcode(barcode):
		v.invalid("Barcode", "only DARF (DCTFWeb) collection barcodes are supported")
	}
}

func (v *validator) qrCode(p core.QRCodePayment) {
	if p.Payee.TaxID == "" {
		v.invalid("Payee.TaxID", "required for QR code payments (beneficiary in segment J-52)")
	}
	q, err := pix.Parse(p.Payload)
	if err != nil {
		v.invalid("Payload", "invalid Pix QR code: %v", err)
		return
	}
	if q.Amount != 0 && q.Amount != p.Amount {
		v.invalid("Amount", "%s differs from the QR code amount %s", p.Amount, q.Amount)
	}
}

// transfer enforces the rail and destination translation table:
//
//	tef + Itaú checking, savings or salary account  → 20/01, 20/05, 30/01
//	ted + account at another bank (any type)         → 20/41
//	pix + Pix key                                    → 20/45, A + B
//	pix + checking, savings or payment account       → 20/45, A
//
// Anything else is ErrUnsupportedRail.
func (v *validator) transfer(p core.TransferPayment) {
	unsupported := func(format string, args ...any) {
		v.add("Rail", fmt.Errorf("%w: %s", core.ErrUnsupportedRail, fmt.Sprintf(format, args...)))
	}

	switch d := p.Destination.(type) {
	case nil:
		v.invalid("Destination", "required")
	case core.PixKey:
		if p.Rail != core.RailPix {
			unsupported("a Pix key destination requires rail pix, got %q", p.Rail)
		}
		v.pixKey(d)
	case core.BankAccount:
		switch p.Rail {
		case core.RailTEF:
			if d.BankCode != bankCode {
				unsupported("tef requires an account at bank %s; use ted for another bank", bankCode)
			}
			if d.Type == core.AccountTypePayment {
				unsupported("tef doesn't support payment accounts")
			}
		case core.RailTED:
			if d.BankCode == bankCode {
				unsupported("use tef for an account at bank %s", bankCode)
			}
		case core.RailPix:
			if d.Type == core.AccountTypeSalary {
				unsupported("pix doesn't support salary accounts")
			}
		default:
			unsupported("rail %q is not supported", p.Rail)
		}
		v.destinationAccount(d, p.Rail)
	}
}

func (v *validator) destinationAccount(a core.BankAccount, rail core.Rail) {
	switch a.Type {
	case core.AccountTypeChecking, core.AccountTypeSavings, core.AccountTypeSalary, core.AccountTypePayment:
	default:
		v.invalid("Destination.Type", "unknown account type %q", a.Type)
	}
	if len(a.BankCode) != 3 || !isAllDigits(a.BankCode) {
		v.invalid("Destination.BankCode", "must be exactly 3 digits")
	}
	if !isAllDigits(a.Branch) {
		v.invalid("Destination.Branch", "must be non-empty digits")
	}
	if !isAllDigits(a.Number) {
		v.invalid("Destination.Number", "must be non-empty digits")
	}

	// Field widths of the account formats in nota 11.
	branchMax, numberMax, digitMax := 5, 12, 2
	switch {
	case a.BankCode == bankCode:
		branchMax, numberMax, digitMax = 4, 6, 1
	case rail == core.RailPix && a.Type == core.AccountTypePayment:
		branchMax, numberMax, digitMax = 5, 20-len(a.CheckDigit), 2
	}
	if len(strings.TrimLeft(a.Branch, "0")) > branchMax {
		v.invalid("Destination.Branch", "must have at most %d significant digits", branchMax)
	}
	if len(strings.TrimLeft(a.Number, "0")) > numberMax {
		v.invalid("Destination.Number", "must have at most %d significant digits", numberMax)
	}
	if n := len(a.CheckDigit); n == 0 || n > digitMax {
		v.invalid("Destination.CheckDigit", "must have 1 to %d characters", digitMax)
	}
}

func (v *validator) pixKey(k core.PixKey) {
	if _, ok := pixKeyTypes[k.Type]; !ok {
		v.invalid("Destination.Type", "unknown Pix key type %q", k.Type)
	}
	switch {
	case k.Value == "":
		v.invalid("Destination.Value", "required")
	case len(k.Value) > 100:
		v.invalid("Destination.Value", "must have at most 100 characters")
	case k.Type == core.PixKeyTypeTaxID:
		if err := taxid.Validate(k.Value); err != nil {
			v.invalid("Destination.Value", "%v", err)
		}
	}
}
