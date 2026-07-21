package registry

import (
	"io"

	"github.com/cnab-dev/cnab-payment/internal/core"
)

// Parser resolves the layout of a CNAB file and streams it into canonical
// occurrences. A Parser holds no per-parse state, so a single instance can
// be reused, including across concurrent Parse calls.
type Parser struct {
	layoutParsers []LayoutParser
}

// New creates a Parser that can resolve any of the given layout
// parsers from a file's first record. At least one is required — there is
// no default; construct built-in layout parsers explicitly (e.g.
// NewItau240LayoutParser) and pass them in, alongside any custom
// LayoutParser implementations you need.
func New(first LayoutParser, rest ...LayoutParser) *Parser {
	return &Parser{layoutParsers: append([]LayoutParser{first}, rest...)}
}

// Parse reads a CNAB file from r as a stream, emitting each fully parsed
// occurrence to onOccurrence. Recoverable record-level errors are reported
// to onRecordError; if onRecordError is nil, such errors are ignored and
// parsing continues. Returning an error from either callback aborts
// parsing and Parse returns that error.
//
// error is reserved for fatal parsing failures: an unreadable input, an
// empty input, or a file whose first record no registered layout parser
// recognizes. ParseResult reflects whatever progress the resolved layout
// parser made before returning.
func (p *Parser) Parse(
	r io.Reader,
	onOccurrence core.OccurrenceHandler,
	onRecordError core.RecordErrorHandler,
) (core.ParseResult, error) {
	s := core.NewScanner(r)

	if !s.Next() {
		if err := s.Err(); err != nil {
			return core.ParseResult{}, err
		}
		return core.ParseResult{}, ErrEmptyInput
	}
	first := s.Record()

	lp := p.resolve(first)
	if lp == nil {
		return core.ParseResult{}, ErrNoLayoutParser
	}

	return lp.Parse(s, first, onOccurrence, onRecordError)
}

// resolve returns the first configured layout parser whose Detect method
// recognizes first, or nil if none match.
func (p *Parser) resolve(first core.Record) LayoutParser {
	for _, lp := range p.layoutParsers {
		if lp.Detect(first) {
			return lp
		}
	}
	return nil
}
