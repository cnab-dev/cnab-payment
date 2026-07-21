package registry

import "errors"

// ErrEmptyInput is returned by Parse when the input contains no records at
// all, so no layout can be resolved.
var ErrEmptyInput = errors.New("cnab-payment: input contains no records")

// ErrNoLayoutParser is returned by Parse when no configured layout parser
// recognizes the input file's first record.
var ErrNoLayoutParser = errors.New("cnab-payment: no layout parser recognizes this file")
