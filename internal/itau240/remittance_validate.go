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
		v.add("Rail", fmt.Errorf("%w: transfers are not supported yet", core.ErrUnsupportedRail))
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
