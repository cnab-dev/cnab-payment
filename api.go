// Package cnabpayment streams Brazilian CNAB payment return files into a
// canonical, bank-agnostic sequence of payment occurrences.
//
// This is the only public package in the module: Occurrence, ParseResult,
// RecordError, the OccurrenceHandler/RecordErrorHandler callbacks,
// Scanner/Record, and the LayoutParser abstraction all live here. Built-in
// layout parsers' actual implementations (currently Itaú CNAB240) are
// under internal/ and are not importable on their own — each is exposed
// at this package's root through a constructor, e.g.
// NewItau240LayoutParser, so it can be passed to NewParser explicitly,
// alone or alongside your own custom LayoutParser implementations.
//
//	p := cnabpayment.NewParser(cnabpayment.NewItau240LayoutParser())
//
//	result, err := p.Parse(r, func(o cnabpayment.Occurrence) error {
//		fmt.Println(o.PaymentID, o.Type, o.RawType, o.CreatedAt)
//		return nil
//	}, nil)
//
// Parsing begins immediately: the parser reads the first physical record,
// uses it to resolve the appropriate LayoutParser, and delegates the rest
// of the stream to that layout parser from the parser's current position
// onward. It never peeks, rewinds, or requires an io.Seeker, and the first
// record is never read twice.
//
// NewParser requires at least one LayoutParser — there is no default.
//
// See the repository README for project philosophy, current status, and
// roadmap.
package cnabpayment

import (
	"io"

	"github.com/cnab-dev/cnab-payment/internal/core"
	"github.com/cnab-dev/cnab-payment/internal/registry"
)

// Occurrence is the canonical representation of a single payment occurrence
// extracted from a CNAB return file, independent of which bank or layout
// produced it. Its field set is expected to grow as more layout parsers
// are added and more of a return file's business information becomes
// worth extracting.
type Occurrence = core.Occurrence

// OccurrenceType classifies an Occurrence's canonical outcome, independent
// of any bank-specific raw code. Layout parsers derive it from RawType.
type OccurrenceType = core.OccurrenceType

const (
	// OccurrenceTypeRejected means the payment was rejected.
	OccurrenceTypeRejected = core.OccurrenceTypeRejected

	// OccurrenceTypeScheduled means the payment was accepted and
	// scheduled, but not yet settled.
	OccurrenceTypeScheduled = core.OccurrenceTypeScheduled

	// OccurrenceTypeSettled means the payment was settled.
	OccurrenceTypeSettled = core.OccurrenceTypeSettled

	// OccurrenceTypeUnknown means the layout parser could not classify
	// RawType into one of the other OccurrenceType values.
	OccurrenceTypeUnknown = core.OccurrenceTypeUnknown
)

// ParseResult reports statistics about a parse run, whether it completed
// successfully or was aborted. It is deliberately minimal and expected to
// grow as the parser gains capabilities.
type ParseResult = core.ParseResult

// RecordError describes a recoverable failure encountered while parsing a
// single record. Unlike a fatal parse error, a RecordError does not by
// itself stop parsing; it is reported to a RecordErrorHandler, which
// decides whether to continue.
type RecordError = core.RecordError

// OccurrenceHandler is invoked for every canonical occurrence emitted while
// parsing, as soon as it is fully parsed. Returning an error aborts
// parsing.
type OccurrenceHandler = core.OccurrenceHandler

// RecordErrorHandler is invoked for every recoverable record-level error
// encountered while parsing. Returning an error aborts parsing. If nil is
// passed to a parser, record errors are ignored and parsing continues.
type RecordErrorHandler = core.RecordErrorHandler

// Record is a single physical record read from the input, exactly as found
// on disk, with its record separator stripped.
type Record = core.Record

// Scanner reads physical records from an input sequentially. Callers must
// check Next before calling Record, and must check Err after Next returns
// false to distinguish a clean end of input from a read failure.
//
// A custom LayoutParser's Parse method receives a Scanner already
// positioned right after the file's first record, and continues reading
// from it.
type Scanner = core.Scanner

// NewScanner creates a Scanner that reads physical records from r, one
// line at a time, split on '\n' with any trailing '\r' stripped.
func NewScanner(r io.Reader) Scanner {
	return core.NewScanner(r)
}

// LayoutParser implements the layout-specific knowledge required to
// recognize a CNAB file from its first record and parse the remainder of
// the stream into canonical occurrences.
//
// Custom implementations are supported: pass one or more to NewParser to
// use them instead of the built-in layout parsers.
type LayoutParser = registry.LayoutParser

// Parser resolves the layout of a CNAB file and streams it into canonical
// occurrences. A Parser holds no per-parse state, so a single instance can
// be reused, including across concurrent Parse calls.
type Parser = registry.Parser

// NewParser creates a Parser that can resolve any of the given layout
// parsers from a file's first record. At least one is required — there is
// no default; construct built-in layout parsers explicitly (e.g.
// NewItau240LayoutParser) and pass them in, alongside any custom
// LayoutParser implementations you need.
func NewParser(first LayoutParser, rest ...LayoutParser) *Parser {
	return registry.New(first, rest...)
}
