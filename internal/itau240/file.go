package itau240

import (
	"fmt"
	"time"
)

// fileHeaderDateTimeLayout parses the file header's generation date and
// time field (columns 144-157, DDMMAAAAHHMMSS) using Go's reference time.
const fileHeaderDateTimeLayout = "02012006150405"

// brasiliaLocation is the timezone assumed for the file header's
// generation date/time, since the field itself carries no timezone
// information and Itaú is a Brazilian bank. It's a fixed UTC-3 offset
// rather than a loaded "America/Sao_Paulo" time.Location so decoding
// doesn't depend on the runtime having an IANA timezone database
// available (e.g. minimal container images) — safe because Brazil has
// observed a fixed UTC-3 offset nationwide, with no daylight saving time,
// since 2019.
var brasiliaLocation = time.FixedZone("-03:00", -3*60*60)

// fileHeader holds the fields extracted from the file header (registro
// header de arquivo, tipo 0). Only what's immediately useful is kept; the
// rest of the record is available in its raw bytes if a future field
// becomes worth extracting.
type fileHeader struct {
	// CreatedAt is the file generation date and time, decoded from
	// columns 144-157 (DDMMAAAAHHMMSS).
	CreatedAt time.Time
}

// decodeFileHeader decodes and validates raw as an Itaú CNAB240 payment
// return file header. It also doubles as the sole source of truth for
// what makes a file an Itaú CNAB240 retorno file: LayoutParser.Detect
// simply checks whether this succeeds.
func decodeFileHeader(raw []byte) (fileHeader, error) {
	if len(raw) != recordLength {
		return fileHeader{}, fmt.Errorf("itau240: file header must be %d bytes, got %d", recordLength, len(raw))
	}
	if recordType(raw) != recordTypeFileHeader {
		return fileHeader{}, fmt.Errorf("itau240: expected file header record type %q, got %q", recordTypeFileHeader, recordType(raw))
	}
	if got := string(raw[0:3]); got != bankCode {
		return fileHeader{}, fmt.Errorf("itau240: unexpected bank code %q, want %q", got, bankCode)
	}
	if got := raw[142]; got != fileKindRetorno {
		return fileHeader{}, fmt.Errorf("itau240: expected a retorno file (kind %q), got %q", fileKindRetorno, got)
	}
	createdAt, err := time.ParseInLocation(fileHeaderDateTimeLayout, string(raw[143:157]), brasiliaLocation)
	if err != nil {
		return fileHeader{}, fmt.Errorf("itau240: invalid file header generation date/time: %w", err)
	}
	return fileHeader{CreatedAt: createdAt}, nil
}

// decodeFileTrailer validates raw as an Itaú CNAB240 file trailer
// (registro trailer de arquivo, tipo 9). It carries no fields this parser
// currently needs.
func decodeFileTrailer(raw []byte) error {
	if len(raw) != recordLength {
		return fmt.Errorf("itau240: file trailer must be %d bytes, got %d", recordLength, len(raw))
	}
	if recordType(raw) != recordTypeFileTrailer {
		return fmt.Errorf("itau240: expected file trailer record type %q, got %q", recordTypeFileTrailer, recordType(raw))
	}
	return nil
}
