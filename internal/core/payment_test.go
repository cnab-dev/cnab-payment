package core

import (
	"errors"
	"testing"
)

func TestBase_EmbeddedPaymentBase_ReturnsSharedFields(t *testing.T) {
	base := PaymentBase{ID: "PAY-1", Amount: 100}

	for _, p := range []Payment{
		TransferPayment{PaymentBase: base},
		BarcodePayment{PaymentBase: base},
		QRCodePayment{PaymentBase: base},
	} {
		if got := p.Base(); got != base {
			t.Errorf("%T.Base() = %+v, want %+v", p, got, base)
		}
	}
}

// describe exercises the intended use of the sealed interfaces: an
// exhaustive type switch over Payment and Destination.
func describe(p Payment) string {
	switch v := p.(type) {
	case TransferPayment:
		switch v.Destination.(type) {
		case BankAccount:
			return "transfer to account"
		case PixKey:
			return "transfer to pix key"
		}
		return "transfer without destination"
	case BarcodePayment:
		return "barcode"
	case QRCodePayment:
		return "qr code"
	}
	return "unknown"
}

func TestSealedInterfaces_TypeSwitch_CoversEveryImplementation(t *testing.T) {
	tests := []struct {
		p    Payment
		want string
	}{
		{TransferPayment{Rail: RailTEF, Destination: BankAccount{BankCode: "341", Type: AccountTypeChecking}}, "transfer to account"},
		{TransferPayment{Rail: RailPix, Destination: PixKey{Type: PixKeyTypeEmail, Value: "a@b.c"}}, "transfer to pix key"},
		{TransferPayment{Rail: RailTED}, "transfer without destination"},
		{BarcodePayment{}, "barcode"},
		{QRCodePayment{}, "qr code"},
	}
	for _, tt := range tests {
		if got := describe(tt.p); got != tt.want {
			t.Errorf("describe(%T) = %q, want %q", tt.p, got, tt.want)
		}
	}
}

func TestError_PaymentError_FormatsWhereAndCause(t *testing.T) {
	tests := []struct {
		e    PaymentError
		want string
	}{
		{PaymentError{Index: 3, PaymentID: "X1", Field: "Amount", Err: ErrInvalidField}, `cnab-payment: payment[3] (id "X1"): Amount: cnab-payment: invalid field`},
		{PaymentError{Index: 0, Err: ErrLayoutMismatch}, `cnab-payment: payment[0]: cnab-payment: payment does not match the layout`},
	}
	for _, tt := range tests {
		if got := tt.e.Error(); got != tt.want {
			t.Errorf("Error() = %q, want %q", got, tt.want)
		}
	}
}

func TestUnwrap_PaymentError_ReturnsUnderlyingCause(t *testing.T) {
	e := &PaymentError{Err: ErrUnsupportedRail}

	if !errors.Is(e, ErrUnsupportedRail) {
		t.Errorf("errors.Is(%v, ErrUnsupportedRail) = false, want true", e)
	}
}
