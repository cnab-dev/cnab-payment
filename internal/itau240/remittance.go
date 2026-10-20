package itau240

import (
	"crypto/rand"

	"github.com/cnab-dev/cnab-payment/internal/cnab240"
	"github.com/cnab-dev/cnab-payment/internal/core"
)

// Remittance classes. The Itaú SISPAG manual requires Pix batches to be
// sent in a file separate from every other payment method (p. 7).
const (
	classPix   core.RemittanceClass = "pix"
	classOther core.RemittanceClass = "other"
)

// LayoutGenerator implements the LayoutGenerator contract for Itaú SISPAG
// CNAB240 remittance files, file layout 081 and batch layout 083. It is
// pinned to that version. It holds no state, so a single instance can be
// reused, including across concurrent calls.
type LayoutGenerator struct{}

// NewGenerator creates an Itaú CNAB240 remittance layout generator.
func NewGenerator() *LayoutGenerator {
	return &LayoutGenerator{}
}

// Validate reports whether p can be written to an Itaú remittance file.
// See validatePayment for the rules.
func (*LayoutGenerator) Validate(p core.Payment) error {
	return validatePayment(p)
}

// Class returns classPix for Pix QR code payments and Pix transfers, and
// classOther for everything else.
func (*LayoutGenerator) Class(p core.Payment) core.RemittanceClass {
	switch v := p.(type) {
	case core.QRCodePayment:
		return classPix
	case core.TransferPayment:
		if v.Rail == core.RailPix {
			return classPix
		}
	}
	return classOther
}

// paymentIDAlphabet and paymentIDLength define Itaú payment IDs ("seu
// número"): 20 random uppercase alphanumeric characters.
const (
	paymentIDAlphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	paymentIDLength   = 20
)

// NewPaymentID returns 20 random uppercase alphanumeric characters.
func (*LayoutGenerator) NewPaymentID() (string, error) {
	// Reject bytes past the largest multiple of the alphabet size, so every
	// character is equally likely.
	const limit = 256 - 256%len(paymentIDAlphabet)
	id := make([]byte, 0, paymentIDLength)
	buf := make([]byte, paymentIDLength)
	for len(id) < paymentIDLength {
		if _, err := rand.Read(buf); err != nil {
			return "", err
		}
		for _, b := range buf {
			if int(b) < limit && len(id) < paymentIDLength {
				id = append(id, paymentIDAlphabet[int(b)%len(paymentIDAlphabet)])
			}
		}
	}
	return string(id), nil
}

// Generate writes payments into a single remittance file. The header's
// creation date and time are written in Brasília time.
func (*LayoutGenerator) Generate(payments []core.Payment, opts core.GenerateOptions) ([]byte, error) {
	f := cnab240.Build(remittanceLayout{}, payments, opts.CreatedAt.In(brasiliaLocation))
	return cnab240.Write(f), nil
}
