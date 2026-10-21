package cnabpayment

import "github.com/cnab-dev/cnab-payment/internal/itau240"

// NewItau240LayoutParser creates the built-in LayoutParser for Itaú
// CNAB240 payment return files. Pass it to NewParser explicitly — alone,
// or alongside your own custom LayoutParser implementations.
func NewItau240LayoutParser() LayoutParser {
	return itau240.New()
}

// NewItau240LayoutGenerator creates the built-in LayoutGenerator for Itaú
// SISPAG CNAB240 remittance files, file layout 081 and batch layout 083.
// It is pinned to that version: a future layout version will get its own
// constructor or option, never silently replace this one. Pass it to
// NewGenerator.
//
// Its classes are "pix" (Pix QR codes and Pix transfers, which the bank
// requires in a separate file) and "other".
func NewItau240LayoutGenerator() LayoutGenerator {
	return itau240.NewGenerator()
}
