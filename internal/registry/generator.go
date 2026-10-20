package registry

import (
	"errors"
	"sort"
	"time"

	"github.com/cnab-dev/cnab-payment/internal/core"
)

// Generator turns a list of payments into the remittance files a single
// layout requires. The caller chooses the layout; Generator never infers
// one. A Generator holds no per-call state, so a single instance can be
// reused, including across concurrent calls.
type Generator struct {
	layout LayoutGenerator
}

// NewGenerator creates a Generator that writes payments with layout.
func NewGenerator(layout LayoutGenerator) *Generator {
	return &Generator{layout: layout}
}

// Generate validates every payment, partitions them by Class, sorts each
// partition by ID and returns one RemittanceFile per partition, ordered by
// Class.
//
// If any payment is invalid, no file is returned and the error joins one
// or more *core.PaymentError per offending payment. An empty list returns
// ErrNoPayments. A failure while encoding a partition is returned as is.
//
// The same payments always produce the same partitions, with payments in
// the same order; with the same non-zero opts.CreatedAt, the files are
// byte-identical. A zero opts.CreatedAt means time.Now().
func (g *Generator) Generate(payments []core.Payment, opts core.GenerateOptions) ([]core.RemittanceFile, error) {
	if len(payments) == 0 {
		return nil, ErrNoPayments
	}

	var errs []error
	for i, p := range payments {
		if err := g.validate(i, p); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}

	if opts.CreatedAt.IsZero() {
		opts.CreatedAt = time.Now()
	}

	partitions := map[core.RemittanceClass][]core.Payment{}
	for _, p := range payments {
		c := g.layout.Class(p)
		partitions[c] = append(partitions[c], p)
	}

	classes := make([]core.RemittanceClass, 0, len(partitions))
	for c := range partitions {
		classes = append(classes, c)
	}
	sort.Slice(classes, func(i, j int) bool { return classes[i] < classes[j] })

	files := make([]core.RemittanceFile, 0, len(classes))
	for _, c := range classes {
		ps := partitions[c]
		sort.SliceStable(ps, func(i, j int) bool { return ps[i].Base().ID < ps[j].Base().ID })

		content, err := g.layout.Generate(ps, opts)
		if err != nil {
			return nil, err
		}

		ids := make([]string, len(ps))
		for i, p := range ps {
			ids[i] = p.Base().ID
		}
		files = append(files, core.RemittanceFile{Class: c, PaymentIDs: ids, Content: content})
	}
	return files, nil
}

// Validate reports whether p can be written by g's layout. If Validate
// returns nil for every payment, Generate won't fail because of them.
// Returned *core.PaymentError values carry p's ID; their Index is 0.
func (g *Generator) Validate(p core.Payment) error {
	return g.validate(0, p)
}

// NewPaymentID returns a fresh payment ID in the format g's layout writes
// into the remittance.
func (g *Generator) NewPaymentID() (string, error) {
	return g.layout.NewPaymentID()
}

func (g *Generator) validate(index int, p core.Payment) error {
	if p == nil {
		return &core.PaymentError{Index: index, Err: core.ErrInvalidField}
	}
	err := g.layout.Validate(p)
	if err == nil {
		return nil
	}
	return errors.Join(paymentErrors(err, index, p.Base().ID)...)
}

// paymentErrors flattens err, as returned by LayoutGenerator.Validate, into
// *core.PaymentError values carrying index and id. Errors that aren't
// PaymentErrors are wrapped in one, so callers can always rely on
// errors.As. The layout's own error values are never mutated.
func paymentErrors(err error, index int, id string) []error {
	if joined, ok := err.(interface{ Unwrap() []error }); ok {
		var out []error
		for _, e := range joined.Unwrap() {
			out = append(out, paymentErrors(e, index, id)...)
		}
		return out
	}

	pe := core.PaymentError{Err: err}
	if layoutErr, ok := err.(*core.PaymentError); ok {
		pe = *layoutErr
	}
	pe.Index = index
	if pe.PaymentID == "" {
		pe.PaymentID = id
	}
	return []error{&pe}
}
