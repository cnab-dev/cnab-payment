package registry

import "github.com/cnab-dev/cnab-payment/internal/core"

// LayoutGenerator implements the layout-specific knowledge required to
// write payments into remittance files for one bank layout. Generator
// handles everything layout-independent — validating the whole list,
// partitioning by Class, sorting by ID — so implementations only decide
// classes and encode records.
//
// Custom implementations are supported: pass one to NewGenerator.
type LayoutGenerator interface {
	// Validate reports whether p can be written by this layout, returning
	// nil if so. Errors should be *core.PaymentError values (joined with
	// errors.Join when there are several) wrapping one of the core
	// sentinels; Generator fills in Index and PaymentID. A payment for
	// which Validate returns nil must not make Generate fail.
	Validate(p core.Payment) error

	// Class returns the bank-mandated partition p belongs to. It must
	// depend only on p, never on other payments.
	Class(p core.Payment) core.RemittanceClass

	// NewPaymentID returns a fresh payment ID in the format this layout
	// writes into the remittance. It is stateless; uniqueness across calls
	// is the caller's responsibility.
	NewPaymentID() (string, error)

	// Generate encodes payments into a single remittance file. Generator
	// only calls it with a non-empty list of already validated payments
	// sharing one Class, sorted by ID, and with a non-zero opts.CreatedAt.
	// CreatedAt is passed as the caller's instant: converting it to the
	// layout's local time before formatting is the layout's job. The same
	// payments and opts must always produce the same bytes.
	Generate(payments []core.Payment, opts core.GenerateOptions) ([]byte, error)
}
