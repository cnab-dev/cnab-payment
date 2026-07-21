package cnabpayment

import (
	"strings"
	"testing"
)

// fakeLayoutParser is a minimal LayoutParser test double used to verify
// that NewParser wires its arguments through to a working *Parser, without
// depending on any real layout implementation.
type fakeLayoutParser struct{ matches bool }

func (f fakeLayoutParser) Detect(first Record) bool { return f.matches }

func (f fakeLayoutParser) Parse(
	s Scanner,
	first Record,
	onOccurrence OccurrenceHandler,
	onRecordError RecordErrorHandler,
) (ParseResult, error) {
	if err := onOccurrence(Occurrence{PaymentID: string(first.Raw)}); err != nil {
		return ParseResult{}, err
	}
	return ParseResult{RecordsRead: 1, OccurrencesEmitted: 1}, nil
}

func TestNewScanner_ValidReader_ReturnsScannerThatReadsRecords(t *testing.T) {
	s := NewScanner(strings.NewReader("PAY-1\n"))

	if !s.Next() {
		t.Fatalf("Next() = false, want true")
	}
	if got := string(s.Record().Raw); got != "PAY-1" {
		t.Errorf("Record().Raw = %q, want %q", got, "PAY-1")
	}
}

func TestNewParser_LayoutParserMatches_DelegatesParsing(t *testing.T) {
	p := NewParser(fakeLayoutParser{matches: false}, fakeLayoutParser{matches: true})

	var occurrences []Occurrence
	result, err := p.Parse(strings.NewReader("PAY-1\n"), func(o Occurrence) error {
		occurrences = append(occurrences, o)
		return nil
	}, nil)
	if err != nil {
		t.Fatalf("Parse() error = %v, want nil", err)
	}
	if len(occurrences) != 1 || occurrences[0].PaymentID != "PAY-1" {
		t.Errorf("occurrences = %+v, want a single occurrence with PaymentID=PAY-1", occurrences)
	}
	if result.RecordsRead != 1 || result.OccurrencesEmitted != 1 {
		t.Errorf("result = %+v, want RecordsRead=1 OccurrencesEmitted=1", result)
	}
}
