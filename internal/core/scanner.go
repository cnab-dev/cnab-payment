package core

import (
	"bufio"
	"bytes"
	"io"
)

// Record is a single physical record read from the input, exactly as found
// on disk, with its record separator stripped.
type Record struct {
	// Raw holds the record's bytes, excluding any trailing line
	// terminator.
	Raw []byte

	// Line is the 1-based physical line number this record was read from.
	Line int
}

// Scanner reads physical records from an input sequentially. Callers must
// check Next before calling Record, and must check Err after Next returns
// false to distinguish a clean end of input from a read failure.
//
// A custom LayoutParser's Parse method receives a Scanner already
// positioned right after the file's first record, and continues reading
// from it.
type Scanner interface {
	// Next advances the scanner to the next physical record. It returns
	// false when there are no more records, either because the input was
	// fully consumed or because a read error occurred; call Err to tell
	// the two apart.
	Next() bool

	// Record returns the record most recently made available by Next.
	// Its result is undefined before the first call to Next or after
	// Next returns false.
	Record() Record

	// Err returns the first non-EOF error encountered while reading, if
	// any.
	Err() error
}

// NewScanner creates a Scanner that reads physical records from r, one
// line at a time, split on '\n' with any trailing '\r' stripped. Unlike
// bufio.Scanner, which is tuned for line-oriented text, this is
// purpose-built for fixed-width banking files: it never peeks, rewinds,
// or requires an io.Seeker, and hands back the raw bytes alongside the
// physical line number.
func NewScanner(r io.Reader) Scanner {
	return &lineScanner{br: bufio.NewReader(r)}
}

type lineScanner struct {
	br      *bufio.Reader
	line    int
	current Record
	err     error
	done    bool
}

func (s *lineScanner) Next() bool {
	if s.done {
		return false
	}

	raw, err := s.br.ReadBytes('\n')
	if len(raw) == 0 {
		s.done = true
		if err != nil && err != io.EOF {
			s.err = err
		}
		return false
	}
	if err != nil && err != io.EOF {
		s.done = true
		s.err = err
		return false
	}
	if err == io.EOF {
		// This was the final record; the next call to Next reports false
		// with no error.
		s.done = true
	}

	s.line++
	s.current = Record{
		Raw:  bytes.TrimRight(raw, "\r\n"),
		Line: s.line,
	}
	return true
}

func (s *lineScanner) Record() Record { return s.current }

func (s *lineScanner) Err() error { return s.err }
