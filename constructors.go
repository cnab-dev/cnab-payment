package cnabpayment

import "github.com/cnab-dev/cnab-payment/internal/itau240"

// NewItau240LayoutParser creates the built-in LayoutParser for Itaú
// CNAB240 payment return files. Pass it to NewParser explicitly — alone,
// or alongside your own custom LayoutParser implementations.
func NewItau240LayoutParser() LayoutParser {
	return itau240.New()
}
