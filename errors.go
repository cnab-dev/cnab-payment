package cnabpayment

import (
	"github.com/cnab-dev/cnab-payment/internal/core"
	"github.com/cnab-dev/cnab-payment/internal/registry"
)

// ErrEmptyInput is returned by Parse when the input contains no records at
// all, so no layout can be resolved.
var ErrEmptyInput = registry.ErrEmptyInput

// ErrNoLayoutParser is returned by Parse when no configured layout parser
// recognizes the input file's first record.
var ErrNoLayoutParser = registry.ErrNoLayoutParser

// ErrNoPayments is returned by Generate when called with an empty list of
// payments.
var ErrNoPayments = registry.ErrNoPayments

// ErrLayoutMismatch is wrapped by a PaymentError when a payment can't be
// written by the chosen layout at all, e.g. its payer account belongs to
// another bank.
var ErrLayoutMismatch = core.ErrLayoutMismatch

// ErrUnsupportedRail is wrapped by a PaymentError when a transfer's Rail
// and Destination combination can't be written by the layout, e.g. a TED
// to an account at the payer's own bank.
var ErrUnsupportedRail = core.ErrUnsupportedRail

// ErrInvalidField is wrapped by a PaymentError when a payment field is
// missing or malformed.
var ErrInvalidField = core.ErrInvalidField
