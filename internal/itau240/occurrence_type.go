package itau240

import "github.com/cnab-dev/cnab-payment/internal/core"

// rawTypePrefixes maps the first two characters of a mandatory segment's
// raw occurrence code (RawType) to a canonical core.OccurrenceType.
// Prefixes not present here classify as core.OccurrenceTypeUnknown.
// Supporting another prefix is a matter of adding an entry here.
var rawTypePrefixes = map[string]core.OccurrenceType{
	"RJ": core.OccurrenceTypeRejected,
	"BD": core.OccurrenceTypeScheduled,
	"00": core.OccurrenceTypeSettled,
}

// classifyRawType derives an occurrence's canonical Type from its RawType.
func classifyRawType(rawType string) core.OccurrenceType {
	if len(rawType) < 2 {
		return core.OccurrenceTypeUnknown
	}
	if t, ok := rawTypePrefixes[rawType[:2]]; ok {
		return t
	}
	return core.OccurrenceTypeUnknown
}
