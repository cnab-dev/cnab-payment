package cnabpayment

import "github.com/cnab-dev/cnab-payment/internal/registry"

// ErrEmptyInput is returned by Parse when the input contains no records at
// all, so no layout can be resolved.
var ErrEmptyInput = registry.ErrEmptyInput

// ErrNoLayoutParser is returned by Parse when no configured layout parser
// recognizes the input file's first record.
var ErrNoLayoutParser = registry.ErrNoLayoutParser
