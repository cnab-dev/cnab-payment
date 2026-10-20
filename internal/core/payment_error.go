package core

import (
	"errors"
	"fmt"
)

// ErrLayoutMismatch is wrapped by a PaymentError when a payment can't be
// written by the chosen layout at all, e.g. its payer account belongs to
// another bank.
var ErrLayoutMismatch = errors.New("cnab-payment: payment does not match the layout")

// ErrUnsupportedRail is wrapped by a PaymentError when a transfer's Rail
// and Destination combination can't be written by the layout, e.g. a TED
// to an account at the payer's own bank.
var ErrUnsupportedRail = errors.New("cnab-payment: rail not supported for this destination")

// ErrInvalidField is wrapped by a PaymentError when a payment field is
// missing or malformed.
var ErrInvalidField = errors.New("cnab-payment: invalid field")

// PaymentError describes why a single payment can't be written. A payment
// may be reported by more than one PaymentError, one per offending field.
type PaymentError struct {
	// Index is the payment's position in the slice passed to Generate. It
	// is only meaningful for errors returned by Generate.
	Index int

	// PaymentID is the offending payment's ID, which may be empty.
	PaymentID string

	// Field names the offending field in this package's model, e.g.
	// "Destination.BankCode". It is empty when the error isn't about a
	// single field.
	Field string

	// Err is the underlying cause, usually wrapping one of the sentinel
	// errors.
	Err error
}

func (e *PaymentError) Error() string {
	where := fmt.Sprintf("payment[%d]", e.Index)
	if e.PaymentID != "" {
		where += fmt.Sprintf(" (id %q)", e.PaymentID)
	}
	if e.Field != "" {
		return fmt.Sprintf("cnab-payment: %s: %s: %v", where, e.Field, e.Err)
	}
	return fmt.Sprintf("cnab-payment: %s: %v", where, e.Err)
}

func (e *PaymentError) Unwrap() error {
	return e.Err
}
