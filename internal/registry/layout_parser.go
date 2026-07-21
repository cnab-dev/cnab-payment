// Package registry holds the LayoutParser extension-point interface and
// the Parser that resolves and delegates to one, kept separate from
// internal/core so that layout parser implementations only ever need to
// depend on the shared domain model, never on this orchestration layer.
package registry

import "github.com/cnab-dev/cnab-payment/internal/core"

// LayoutParser implements the layout-specific knowledge required to
// recognize a CNAB file from its first record and parse the remainder of
// the stream into canonical occurrences.
//
// Custom implementations are supported: pass one or more to New to
// use them instead of the built-in layout parsers.
type LayoutParser interface {
	// Detect reports whether first, the file's first physical record,
	// belongs to the layout this parser understands.
	Detect(first core.Record) bool

	// Parse consumes the remainder of s, starting from the record right
	// after first, and emits canonical occurrences to onOccurrence as
	// soon as they are fully parsed. first is provided because it was
	// already read from s to resolve the layout parser and must not be
	// read again.
	//
	// Recoverable record-level errors are reported to onRecordError, if
	// non-nil; returning an error from either callback aborts parsing.
	Parse(
		s core.Scanner,
		first core.Record,
		onOccurrence core.OccurrenceHandler,
		onRecordError core.RecordErrorHandler,
	) (core.ParseResult, error)
}
