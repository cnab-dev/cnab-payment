package core

import (
	"bytes"
	"errors"
	"testing"
)

// chunkReader is an io.Reader test double that returns each chunk on a
// successive Read call, then err forever after.
type chunkReader struct {
	chunks [][]byte
	err    error
	i      int
}

func (r *chunkReader) Read(p []byte) (int, error) {
	if r.i < len(r.chunks) {
		n := copy(p, r.chunks[r.i])
		r.i++
		return n, nil
	}
	return 0, r.err
}

func TestNewScanner_ValidReader_ReturnsScannerThatReadsRecords(t *testing.T) {
	s := NewScanner(bytes.NewBufferString("hello\n"))

	if !s.Next() {
		t.Fatalf("Next() = false, want true")
	}
	if got := string(s.Record().Raw); got != "hello" {
		t.Errorf("Record().Raw = %q, want %q", got, "hello")
	}
}

func TestNext_MultipleCleanLines_ReturnsEachRecordInOrderWithLineNumbers(t *testing.T) {
	s := NewScanner(bytes.NewBufferString("first\nsecond\nthird\n"))

	var got []Record
	for s.Next() {
		got = append(got, s.Record())
	}
	if err := s.Err(); err != nil {
		t.Fatalf("Err() = %v, want nil", err)
	}

	want := []Record{
		{Raw: []byte("first"), Line: 1},
		{Raw: []byte("second"), Line: 2},
		{Raw: []byte("third"), Line: 3},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d records, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if string(got[i].Raw) != string(want[i].Raw) || got[i].Line != want[i].Line {
			t.Errorf("record[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestNext_TrailingCarriageReturn_StripsItFromRaw(t *testing.T) {
	s := NewScanner(bytes.NewBufferString("windows-style\r\n"))

	if !s.Next() {
		t.Fatalf("Next() = false, want true")
	}
	if got := string(s.Record().Raw); got != "windows-style" {
		t.Errorf("Record().Raw = %q, want %q", got, "windows-style")
	}
}

func TestNext_FinalLineWithoutTrailingNewline_ReturnsItAsALastRecord(t *testing.T) {
	s := NewScanner(bytes.NewBufferString("only line, no newline"))

	if !s.Next() {
		t.Fatalf("Next() = false, want true")
	}
	if got := string(s.Record().Raw); got != "only line, no newline" {
		t.Errorf("Record().Raw = %q, want %q", got, "only line, no newline")
	}
	if s.Next() {
		t.Fatalf("Next() = true on second call, want false")
	}
	if err := s.Err(); err != nil {
		t.Errorf("Err() = %v, want nil", err)
	}
}

func TestNext_EmptyInput_ReturnsFalseWithNilErr(t *testing.T) {
	s := NewScanner(bytes.NewBufferString(""))

	if s.Next() {
		t.Fatalf("Next() = true, want false")
	}
	if err := s.Err(); err != nil {
		t.Errorf("Err() = %v, want nil", err)
	}
}

func TestNext_CalledAfterScannerIsDone_ReturnsFalseWithoutReading(t *testing.T) {
	s := NewScanner(bytes.NewBufferString("line\n"))

	if !s.Next() {
		t.Fatalf("Next() = false on first call, want true")
	}
	if s.Next() {
		t.Fatalf("Next() = true on second call, want false")
	}
	if s.Next() {
		t.Fatalf("Next() = true on third call, want false")
	}
}

func TestNext_ReaderFailsBeforeAnyDataIsRead_ReturnsFalseWithErr(t *testing.T) {
	wantErr := errors.New("boom")
	s := NewScanner(&chunkReader{err: wantErr})

	if s.Next() {
		t.Fatalf("Next() = true, want false")
	}
	if err := s.Err(); !errors.Is(err, wantErr) {
		t.Errorf("Err() = %v, want %v", err, wantErr)
	}
}

func TestNext_ReaderFailsAfterPartialData_ReturnsFalseWithErr(t *testing.T) {
	wantErr := errors.New("boom")
	s := NewScanner(&chunkReader{chunks: [][]byte{[]byte("partial")}, err: wantErr})

	if s.Next() {
		t.Fatalf("Next() = true, want false")
	}
	if err := s.Err(); !errors.Is(err, wantErr) {
		t.Errorf("Err() = %v, want %v", err, wantErr)
	}
}
