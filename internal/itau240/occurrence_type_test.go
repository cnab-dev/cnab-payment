package itau240

import (
	"testing"

	"github.com/cnab-dev/cnab-payment/internal/core"
)

func TestClassifyRawType(t *testing.T) {
	tests := []struct {
		name    string
		rawType string
		want    core.OccurrenceType
	}{
		{"rejectedPrefixWithSuffix_returnsRejected", "RJNR", core.OccurrenceTypeRejected},
		{"rejectedPrefixExact_returnsRejected", "RJ", core.OccurrenceTypeRejected},
		{"scheduledPrefixExact_returnsScheduled", "BD", core.OccurrenceTypeScheduled},
		{"scheduledPrefixWithSuffix_returnsScheduled", "BDXX", core.OccurrenceTypeScheduled},
		{"settledPrefixExact_returnsSettled", "00", core.OccurrenceTypeSettled},
		{"settledPrefixWithSuffix_returnsSettled", "0000000000", core.OccurrenceTypeSettled},
		{"unrecognizedNumericPrefix_returnsUnknown", "02", core.OccurrenceTypeUnknown},
		{"unrecognizedAlphaPrefix_returnsUnknown", "XX", core.OccurrenceTypeUnknown},
		{"emptyString_returnsUnknown", "", core.OccurrenceTypeUnknown},
		{"singleCharacterTooShort_returnsUnknown", "R", core.OccurrenceTypeUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := classifyRawType(tt.rawType); got != tt.want {
				t.Errorf("classifyRawType(%q) = %v, want %v", tt.rawType, got, tt.want)
			}
		})
	}
}
