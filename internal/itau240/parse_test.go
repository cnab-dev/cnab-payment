package itau240

import (
	"testing"

	"github.com/cnab-dev/cnab-payment/internal/core"
)

// These call parseState's unexported methods directly, bypassing run()'s
// dispatch. run() only ever routes a record to one of these methods after
// already checking its length and record-type byte, so the decode failure
// each method guards against can't be reached by driving Parse end to end
// — it can only be exercised by calling the method directly with a record
// that violates that guarantee.

func TestParseBatchHeader_DecodeFails_ReturnsError(t *testing.T) {
	p := &parseState{}

	err := p.parseBatchHeader(core.Record{Raw: []byte("too short"), Line: 1})
	if err == nil {
		t.Fatalf("parseBatchHeader() error = nil, want a decode error")
	}
}

func TestParseDetail_DecodeFails_ReturnsError(t *testing.T) {
	p := &parseState{inBatch: true, currentBatch: "0001"}

	err := p.parseDetail(core.Record{Raw: []byte("too short"), Line: 1})
	if err == nil {
		t.Fatalf("parseDetail() error = nil, want a decode error")
	}
}

func TestParseBatchTrailer_DecodeFails_ReturnsError(t *testing.T) {
	p := &parseState{inBatch: true, currentBatch: "0001"}

	err := p.parseBatchTrailer(core.Record{Raw: []byte("too short"), Line: 1})
	if err == nil {
		t.Fatalf("parseBatchTrailer() error = nil, want a decode error")
	}
}

func TestParseFileTrailer_DecodeFails_ReturnsError(t *testing.T) {
	p := &parseState{}

	err := p.parseFileTrailer(core.Record{Raw: []byte("too short"), Line: 1})
	if err == nil {
		t.Fatalf("parseFileTrailer() error = nil, want a decode error")
	}
}
