package registry

import (
	"errors"
	"strings"
	"testing"

	"github.com/cnab-dev/cnab-payment/internal/core"
)

// fakeLayoutParser is a test double for LayoutParser: matches controls
// Detect's result, and result/err are returned verbatim by Parse, so tests
// can assert on delegation without depending on any real layout.
type fakeLayoutParser struct {
	matches bool
	result  core.ParseResult
	err     error

	called   bool
	gotFirst core.Record
}

func (f *fakeLayoutParser) Detect(first core.Record) bool { return f.matches }

func (f *fakeLayoutParser) Parse(
	s core.Scanner,
	first core.Record,
	onOccurrence core.OccurrenceHandler,
	onRecordError core.RecordErrorHandler,
) (core.ParseResult, error) {
	f.called = true
	f.gotFirst = first
	return f.result, f.err
}

// erroringReader always fails with err, without ever returning data.
type erroringReader struct{ err error }

func (r erroringReader) Read(p []byte) (int, error) { return 0, r.err }

func TestParse_EmptyInput_ReturnsErrEmptyInput(t *testing.T) {
	p := New(&fakeLayoutParser{matches: true})

	_, err := p.Parse(strings.NewReader(""), nil, nil)
	if !errors.Is(err, ErrEmptyInput) {
		t.Fatalf("Parse() error = %v, want ErrEmptyInput", err)
	}
}

func TestParse_ScannerFailsBeforeFirstRecord_ReturnsScannerErr(t *testing.T) {
	wantErr := errors.New("read failed")
	p := New(&fakeLayoutParser{matches: true})

	_, err := p.Parse(erroringReader{err: wantErr}, nil, nil)
	if !errors.Is(err, wantErr) {
		t.Fatalf("Parse() error = %v, want %v", err, wantErr)
	}
}

func TestParse_NoLayoutParserMatches_ReturnsErrNoLayoutParser(t *testing.T) {
	p := New(&fakeLayoutParser{matches: false}, &fakeLayoutParser{matches: false})

	_, err := p.Parse(strings.NewReader("some record\n"), nil, nil)
	if !errors.Is(err, ErrNoLayoutParser) {
		t.Fatalf("Parse() error = %v, want ErrNoLayoutParser", err)
	}
}

func TestParse_FirstConfiguredLayoutParserMatches_DelegatesToIt(t *testing.T) {
	first := &fakeLayoutParser{matches: true, result: core.ParseResult{RecordsRead: 3}}
	second := &fakeLayoutParser{matches: true}
	p := New(first, second)

	got, err := p.Parse(strings.NewReader("some record\n"), nil, nil)
	if err != nil {
		t.Fatalf("Parse() error = %v, want nil", err)
	}
	if !first.called {
		t.Errorf("first layout parser was not called")
	}
	if second.called {
		t.Errorf("second layout parser was called, want it skipped since the first already matched")
	}
	if got != first.result {
		t.Errorf("Parse() = %+v, want %+v", got, first.result)
	}
	if string(first.gotFirst.Raw) != "some record" {
		t.Errorf("first record passed to LayoutParser.Parse = %q, want %q", first.gotFirst.Raw, "some record")
	}
}

func TestParse_OnlyLaterLayoutParserMatches_ResolvesPastNonMatchingOnes(t *testing.T) {
	first := &fakeLayoutParser{matches: false}
	second := &fakeLayoutParser{matches: true}
	p := New(first, second)

	_, err := p.Parse(strings.NewReader("some record\n"), nil, nil)
	if err != nil {
		t.Fatalf("Parse() error = %v, want nil", err)
	}
	if first.called {
		t.Errorf("non-matching layout parser was called")
	}
	if !second.called {
		t.Errorf("matching layout parser was not called")
	}
}

func TestParse_LayoutParserReturnsError_PropagatesItAndItsResult(t *testing.T) {
	wantErr := errors.New("boom")
	wantResult := core.ParseResult{RecordsRead: 1, OccurrencesEmitted: 0}
	p := New(&fakeLayoutParser{matches: true, result: wantResult, err: wantErr})

	got, err := p.Parse(strings.NewReader("some record\n"), nil, nil)
	if !errors.Is(err, wantErr) {
		t.Fatalf("Parse() error = %v, want %v", err, wantErr)
	}
	if got != wantResult {
		t.Errorf("Parse() = %+v, want %+v", got, wantResult)
	}
}
