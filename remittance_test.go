package cnabpayment

import (
	"errors"
	"testing"
	"time"
)

// fakeLayoutGenerator is a minimal LayoutGenerator test double used to
// verify that NewGenerator wires its argument through to a working
// *Generator, and that a custom layout can be written against the root
// package alone.
type fakeLayoutGenerator struct{}

func (fakeLayoutGenerator) Validate(p Payment) error {
	if p.Base().Amount <= 0 {
		return &PaymentError{Field: "Amount", Err: ErrInvalidField}
	}
	return nil
}

func (fakeLayoutGenerator) Class(p Payment) RemittanceClass {
	if t, ok := p.(TransferPayment); ok && t.Rail == RailPix {
		return "pix"
	}
	return "other"
}

func (fakeLayoutGenerator) NewPaymentID() (string, error) { return "ID-1", nil }

func (fakeLayoutGenerator) Generate(payments []Payment, opts GenerateOptions) ([]byte, error) {
	return []byte(opts.CreatedAt.Format(time.RFC3339)), nil
}

func TestNewGenerator_ValidPayments_GeneratesOneFilePerClass(t *testing.T) {
	g := NewGenerator(fakeLayoutGenerator{})
	createdAt := time.Date(2026, 4, 20, 10, 0, 0, 0, time.UTC)

	files, err := g.Generate([]Payment{
		TransferPayment{PaymentBase: PaymentBase{ID: "B", Amount: 100}, Rail: RailPix, Destination: PixKey{Type: PixKeyTypeEmail, Value: "a@b.c"}},
		BarcodePayment{PaymentBase: PaymentBase{ID: "A", Amount: 100}},
	}, GenerateOptions{CreatedAt: createdAt})
	if err != nil {
		t.Fatalf("Generate() error = %v, want nil", err)
	}
	if len(files) != 2 || files[0].Class != "other" || files[1].Class != "pix" {
		t.Fatalf("files = %+v, want classes [other pix]", files)
	}
	if got := string(files[1].Content); got != "2026-04-20T10:00:00Z" {
		t.Errorf("Content = %q, want the CreatedAt passed in", got)
	}
}

func TestNewGenerator_InvalidPayment_ReturnsPaymentError(t *testing.T) {
	g := NewGenerator(fakeLayoutGenerator{})

	_, err := g.Generate([]Payment{BarcodePayment{PaymentBase: PaymentBase{ID: "A"}}}, GenerateOptions{})

	var pe *PaymentError
	if !errors.As(err, &pe) || pe.PaymentID != "A" || pe.Field != "Amount" || !errors.Is(err, ErrInvalidField) {
		t.Errorf("Generate() error = %v, want a PaymentError for A on Amount wrapping ErrInvalidField", err)
	}
}

func TestNewGenerator_EmptyInput_ReturnsErrNoPayments(t *testing.T) {
	g := NewGenerator(fakeLayoutGenerator{})

	if _, err := g.Generate(nil, GenerateOptions{}); !errors.Is(err, ErrNoPayments) {
		t.Errorf("Generate() error = %v, want ErrNoPayments", err)
	}
}

func TestParseAmount_ValidInput_ReturnsCents(t *testing.T) {
	a, err := ParseAmount("123.45")
	if err != nil || a != 12345 {
		t.Errorf("ParseAmount() = %d, %v, want 12345, nil", a, err)
	}
}
