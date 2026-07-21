package core

import (
	"errors"
	"testing"
)

func TestError_WrapsLineAndCause_ReturnsFormattedMessage(t *testing.T) {
	cause := errors.New("bad field")
	e := &RecordError{Line: 42, Raw: []byte("raw"), Err: cause}

	want := "cnab-payment: record error at line 42: bad field"
	if got := e.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestUnwrap_RecordError_ReturnsUnderlyingCause(t *testing.T) {
	cause := errors.New("bad field")
	e := &RecordError{Line: 1, Err: cause}

	if got := e.Unwrap(); got != cause {
		t.Errorf("Unwrap() = %v, want %v", got, cause)
	}
	if !errors.Is(e, cause) {
		t.Errorf("errors.Is(e, cause) = false, want true")
	}
}
