package cnabpayment

import "testing"

func TestNewItau240LayoutParser_GarbageFirstRecord_DetectReturnsFalse(t *testing.T) {
	lp := NewItau240LayoutParser()

	if lp.Detect(Record{Raw: []byte("not a cnab240 record"), Line: 1}) {
		t.Errorf("Detect() = true for a non-Itaú-240 record, want false")
	}
}
