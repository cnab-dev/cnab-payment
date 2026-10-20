package cnabpayment

import (
	"github.com/cnab-dev/cnab-payment/internal/core"
	"github.com/cnab-dev/cnab-payment/internal/registry"
)

// Payment is a single payment instruction to be written into a remittance
// file. It is sealed: only TransferPayment, BarcodePayment and
// QRCodePayment implement it, so a type switch over a Payment is
// exhaustive.
type Payment = core.Payment

// PaymentBase holds the fields shared by every kind of payment. Its ID is
// written into the remittance and comes back as Occurrence.PaymentID in
// the return file.
type PaymentBase = core.PaymentBase

// TransferPayment moves money to a Destination through an explicit Rail.
// Layouts never infer the rail; a Rail and Destination combination a
// layout can't write is reported as ErrUnsupportedRail.
type TransferPayment = core.TransferPayment

// BarcodePayment pays a boleto or a utility/tax collection bill
// (arrecadação); which one is determined by the barcode itself.
type BarcodePayment = core.BarcodePayment

// QRCodePayment pays a Pix QR code from its "copia e cola" payload.
type QRCodePayment = core.QRCodePayment

// Rail is the path a transfer takes to reach its destination.
type Rail = core.Rail

const (
	// RailTEF is a transfer between accounts of the same bank.
	RailTEF = core.RailTEF

	// RailTED is a TED transfer to an account at another bank.
	RailTED = core.RailTED

	// RailPix is a Pix transfer, to a Pix key or to a bank account.
	RailPix = core.RailPix
)

// Destination is where a TransferPayment sends money. It is sealed: only
// BankAccount and PixKey implement it.
type Destination = core.Destination

// BankAccount is a bank account, used both as a payer account and as a
// transfer destination.
type BankAccount = core.BankAccount

// AccountType describes a BankAccount. It says nothing about how money
// reaches the account; that's the transfer's Rail.
type AccountType = core.AccountType

const (
	AccountTypeChecking = core.AccountTypeChecking
	AccountTypeSavings  = core.AccountTypeSavings
	AccountTypeSalary   = core.AccountTypeSalary
	AccountTypePayment  = core.AccountTypePayment
)

// PixKey is a Pix key registered in the DICT.
type PixKey = core.PixKey

// PixKeyType is the kind of a PixKey. It is provided by the caller; layouts
// never infer it from the key's value.
type PixKeyType = core.PixKeyType

const (
	// PixKeyTypeTaxID is a CPF or CNPJ key.
	PixKeyTypeTaxID  = core.PixKeyTypeTaxID
	PixKeyTypeEmail  = core.PixKeyTypeEmail
	PixKeyTypePhone  = core.PixKeyTypePhone
	PixKeyTypeRandom = core.PixKeyTypeRandom
)

// Party is an individual or a company taking part in a payment.
type Party = core.Party

// Address is a postal address.
type Address = core.Address

// Amount is a monetary value in cents (centavos). CNAB files only
// represent whole cents, so a value with a third decimal place can't be
// expressed.
type Amount = core.Amount

// ParseAmount parses a decimal string with a '.' separator and at most two
// decimal places, e.g. "123.45" → 12345. An optional leading '-' is
// accepted; anything else, including a third decimal place, is an error.
func ParseAmount(s string) (Amount, error) {
	return core.ParseAmount(s)
}

// RemittanceClass identifies which bank-mandated partition a
// RemittanceFile belongs to. Values are defined by each layout (Itaú:
// "pix" or "other"); they are lowercase ASCII and safe in a file name.
type RemittanceClass = core.RemittanceClass

// RemittanceFile is one remittance file produced from a list of payments:
// its Class, the IDs of the payments in it, and its Content.
type RemittanceFile = core.RemittanceFile

// GenerateOptions configures a remittance generation run. A zero CreatedAt
// means time.Now().
type GenerateOptions = core.GenerateOptions

// PaymentError describes why a single payment can't be written. A payment
// may be reported by more than one PaymentError, one per offending field.
type PaymentError = core.PaymentError

// LayoutGenerator implements the layout-specific knowledge required to
// write payments into remittance files for one bank layout.
//
// Custom implementations are supported: pass one to NewGenerator.
type LayoutGenerator = registry.LayoutGenerator

// Generator turns a list of payments into the remittance files a single
// layout requires. A Generator holds no per-call state, so a single
// instance can be reused, including across concurrent calls.
type Generator = registry.Generator

// NewGenerator creates a Generator that writes payments with layout. The
// layout is always chosen explicitly — there is no default, and Generator
// never infers one from the payments.
func NewGenerator(layout LayoutGenerator) *Generator {
	return registry.NewGenerator(layout)
}
