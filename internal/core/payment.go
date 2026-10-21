package core

import "time"

// Payment is a single payment instruction to be written into a remittance
// file. It is sealed: only TransferPayment, BarcodePayment and
// QRCodePayment implement it, so a type switch over a Payment is
// exhaustive.
type Payment interface {
	// Base returns the fields shared by every kind of payment.
	Base() PaymentBase

	isPayment()
}

// PaymentBase holds the fields shared by every kind of payment.
type PaymentBase struct {
	// ID identifies the payment to the bank ("seu número"). It is written
	// into the remittance and comes back as Occurrence.PaymentID in the
	// return file.
	ID string

	// Payer is the holder of PayerAccount.
	Payer Party

	// PayerAccount is the account the payment is debited from.
	PayerAccount BankAccount

	// Debtor is who owes the bill, when it isn't Payer. Optional; nil
	// means Payer.
	Debtor *Party

	// Payee is who receives the payment.
	Payee Party

	// Amount is the payment amount.
	Amount Amount

	// DueDate is a calendar date: only its year, month and day are used,
	// as given, with no time zone conversion.
	DueDate time.Time

	// SettlementDate is the date the payment should be made. Like DueDate,
	// it is a calendar date.
	SettlementDate time.Time
}

// Base returns b itself, so every payment type embedding PaymentBase
// satisfies Payment's Base method.
func (b PaymentBase) Base() PaymentBase { return b }

// Rail is the path a transfer takes to reach its destination.
type Rail string

const (
	// RailTEF is a transfer between accounts of the same bank.
	RailTEF Rail = "tef"

	// RailTED is a TED transfer to an account at another bank.
	RailTED Rail = "ted"

	// RailPix is a Pix transfer, to a Pix key or to a bank account.
	RailPix Rail = "pix"
)

// TransferPayment moves money to a Destination through an explicit Rail.
// Layouts never infer the rail; a Rail and Destination combination a
// layout can't write is reported as ErrUnsupportedRail.
type TransferPayment struct {
	PaymentBase

	Rail        Rail
	Destination Destination
}

// BarcodePayment pays a boleto or a utility/tax collection bill
// (arrecadação); which one is determined by the barcode itself.
type BarcodePayment struct {
	PaymentBase

	// Barcode is the 44-digit barcode.
	Barcode string
}

// QRCodePayment pays a Pix QR code from its "copia e cola" payload.
type QRCodePayment struct {
	PaymentBase

	// Payload is the Pix "copia e cola" (EMV) payload.
	Payload string

	// TxID overrides the payload's transaction ID when set. Optional.
	TxID string
}

func (TransferPayment) isPayment() {}
func (BarcodePayment) isPayment()  {}
func (QRCodePayment) isPayment()   {}
