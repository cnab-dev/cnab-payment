package cnabpayment

import (
	"errors"
	"testing"
)

func TestNewItau240LayoutParser_GarbageFirstRecord_DetectReturnsFalse(t *testing.T) {
	lp := NewItau240LayoutParser()

	if lp.Detect(Record{Raw: []byte("not a cnab240 record"), Line: 1}) {
		t.Errorf("Detect() = true for a non-Itaú-240 record, want false")
	}
}

func TestNewItau240LayoutGenerator_NewPaymentID(t *testing.T) {
	g := NewGenerator(NewItau240LayoutGenerator())

	id, err := g.NewPaymentID()
	if err != nil {
		t.Fatalf("NewPaymentID() error = %v", err)
	}
	if len(id) != 20 {
		t.Errorf("NewPaymentID() = %q, want 20 characters", id)
	}
}

func TestNewItau240LayoutGenerator_PayerAtAnotherBank_ReturnsErrLayoutMismatch(t *testing.T) {
	g := NewGenerator(NewItau240LayoutGenerator())

	err := g.Validate(BarcodePayment{PaymentBase: PaymentBase{PayerAccount: BankAccount{BankCode: "237"}}})
	if !errors.Is(err, ErrLayoutMismatch) {
		t.Errorf("Validate() error = %v, want ErrLayoutMismatch", err)
	}
}
