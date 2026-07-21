// Package itau240 implements the LayoutParser contract required by
// github.com/cnab-dev/cnab-payment (via internal/registry), satisfied
// structurally without importing that package — see internal/core for the
// shared types this depends on instead. All Itaú-specific knowledge —
// record navigation, detail grouping, identifier extraction, and
// occurrence-code handling — is owned here; the generic streaming
// infrastructure remains unaware of any of it.
//
// Only payment return files (retorno) are in scope. Remittance files and
// CNAB400 are not supported.
package itau240

import "github.com/cnab-dev/cnab-payment/internal/core"

// LayoutParser implements the LayoutParser contract for Itaú CNAB240
// payment return files. It holds no per-parse state, so a single instance
// can be reused, including across concurrent Parse calls.
type LayoutParser struct{}

// New creates an Itaú CNAB240 layout parser.
func New() *LayoutParser {
	return &LayoutParser{}
}

// Detect reports whether first is the file header of an Itaú CNAB240
// payment return file.
func (lp *LayoutParser) Detect(first core.Record) bool {
	_, err := decodeFileHeader(first.Raw)
	return err == nil
}

// Parse consumes the remainder of s, starting right after first, and
// streams canonical occurrences to onOccurrence as soon as each logical
// payment is fully parsed. Detail records are grouped by their batch
// sequence number: consecutive detail records that share a sequence
// number belong to the same logical payment, which is only resolved into
// occurrence(s) once the sequence number changes or the batch trailer is
// reached. See the root cnabpayment package for the callback and error
// semantics this follows.
func (lp *LayoutParser) Parse(
	s core.Scanner,
	first core.Record,
	onOccurrence core.OccurrenceHandler,
	onRecordError core.RecordErrorHandler,
) (core.ParseResult, error) {
	ps := &parseState{
		s:             s,
		onOccurrence:  onOccurrence,
		onRecordError: onRecordError,
	}
	return ps.run(first)
}
