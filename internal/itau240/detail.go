package itau240

import (
	"fmt"
	"strings"

	"github.com/cnab-dev/cnab-payment/internal/core"
)

// detail holds the fields common to every detail record (registro
// detalhe, tipo 3), decoded from its fixed envelope. Segment-specific
// fields are pulled separately, from Raw, by a segmentExtractor.
type detail struct {
	BatchNumber string

	// Sequence is the "Número do Registro no Lote" field (columns 9-13).
	// Consecutive detail records that share this value belong to the same
	// logical payment; see flushCurrentDetail.
	Sequence string

	// Segment is the "Código de Segmento" byte (column 14), e.g. 'A'.
	Segment byte

	Raw  []byte
	Line int
}

func decodeDetail(rec core.Record) (detail, error) {
	raw := rec.Raw
	if len(raw) != recordLength {
		return detail{}, fmt.Errorf("itau240: detail record must be %d bytes, got %d", recordLength, len(raw))
	}
	if recordType(raw) != recordTypeDetail {
		return detail{}, fmt.Errorf("itau240: expected detail record type %q, got %q", recordTypeDetail, recordType(raw))
	}
	return detail{
		BatchNumber: batchNumber(raw),
		Sequence:    string(raw[8:13]),
		Segment:     raw[13],
		Raw:         raw,
		Line:        rec.Line,
	}, nil
}

// isSegmentLetter reports whether b is a plausible "Código de Segmento"
// value. It does not mean the segment is implemented, only that it's
// grammatically valid CNAB240 — used to tell a genuinely corrupt detail
// record apart from one this parser simply doesn't extract fields from
// yet.
func isSegmentLetter(b byte) bool {
	return b >= 'A' && b <= 'Z'
}

// occurrenceFields holds the values a segmentExtractor contributes toward
// the canonical occurrence for a logical payment. A single logical
// payment's fields are typically assembled from more than one segment
// extractor; see flushCurrentDetail, which merges them field by field
// rather than letting one segment's contribution replace another's.
type occurrenceFields struct {
	PaymentID      string
	RawType        string
	ConfirmationID string

	// Segment is the mandatory segment (A, J, O, or N) that contributed
	// PaymentID/RawType, used as the "S" component of ReceiptFormat.
	// Segment Z, being complementary, never sets this.
	Segment byte
}

// segmentExtractor pulls the occurrence-relevant fields out of a single
// decoded detail record. Not every segment carries them; an extractor
// returns ok=false when it has nothing to contribute (e.g. a
// complementary segment for a payment whose identifying fields live in
// its Segment A). An extractor should leave a field zero-valued in its
// result when it has nothing to say about that particular field, so the
// merge in flushCurrentDetail doesn't clobber another segment's
// contribution.
type segmentExtractor func(d detail) (fields occurrenceFields, ok bool)

// segmentExtractors maps each supported segment code to the function that
// knows how to read occurrence fields out of it. Segments A, J, and O are
// each a batch's mandatory segment, depending on payment method (crédito
// em conta, boleto, and boleto de outro banco, respectively) — exactly
// one of them carries the payment identifier and occurrence code for a
// given logical payment. Segment Z is optional and, when present, follows
// the mandatory segment within the same batch sequence number, carrying a
// bank-issued confirmation identifier. Supporting another detail record
// format is a matter of writing its extractor and registering it here —
// no other part of this parser needs to change.
var segmentExtractors = map[byte]segmentExtractor{
	'A': extractSegmentA,
	'J': extractSegmentJ,
	'O': extractSegmentO,
	'N': extractSegmentN,
	'Z': extractSegmentZ,
}

// occurrenceCodeStart/End mark the "Código de ocorrências para retorno"
// field. It sits at the same position across every mandatory segment
// (A, J, O).
const occurrenceCodeStart, occurrenceCodeEnd = 230, 240

// Field positions below follow the Febraban CNAB240 layout as commonly
// implemented for payment return files; verify against Itaú's official
// manual before relying on them in production.
const (
	segmentAPaymentIDStart, segmentAPaymentIDEnd = 73, 93 // "Número do documento atribuído pela empresa"
	segmentJPaymentIDStart, segmentJPaymentIDEnd = 182, 202
	segmentOPaymentIDStart, segmentOPaymentIDEnd = 174, 194
	segmentNPaymentIDStart, segmentNPaymentIDEnd = 195, 215
)

func extractSegmentA(d detail) (occurrenceFields, bool) {
	return occurrenceFields{
		PaymentID: strings.TrimSpace(string(d.Raw[segmentAPaymentIDStart:segmentAPaymentIDEnd])),
		RawType:   strings.TrimSpace(string(d.Raw[occurrenceCodeStart:occurrenceCodeEnd])),
		Segment:   d.Segment,
	}, true
}

func extractSegmentJ(d detail) (occurrenceFields, bool) {
	return occurrenceFields{
		PaymentID: strings.TrimSpace(string(d.Raw[segmentJPaymentIDStart:segmentJPaymentIDEnd])),
		RawType:   strings.TrimSpace(string(d.Raw[occurrenceCodeStart:occurrenceCodeEnd])),
		Segment:   d.Segment,
	}, true
}

func extractSegmentO(d detail) (occurrenceFields, bool) {
	return occurrenceFields{
		PaymentID: strings.TrimSpace(string(d.Raw[segmentOPaymentIDStart:segmentOPaymentIDEnd])),
		RawType:   strings.TrimSpace(string(d.Raw[occurrenceCodeStart:occurrenceCodeEnd])),
		Segment:   d.Segment,
	}, true
}

func extractSegmentN(d detail) (occurrenceFields, bool) {
	return occurrenceFields{
		PaymentID: strings.TrimSpace(string(d.Raw[segmentNPaymentIDStart:segmentNPaymentIDEnd])),
		RawType:   strings.TrimSpace(string(d.Raw[occurrenceCodeStart:occurrenceCodeEnd])),
		Segment:   d.Segment,
	}, true
}

// segmentZConfirmationIDStart/End mark the confirmation identifier field
// within Segment Z. Segment Z's layout is not part of the well-known
// Febraban Segment A/B/C set this parser otherwise follows, so this
// position is a placeholder assumption — verify against Itaú's official
// manual before relying on it in production.
const segmentZConfirmationIDStart, segmentZConfirmationIDEnd = 14, 78

func extractSegmentZ(d detail) (occurrenceFields, bool) {
	return occurrenceFields{
		ConfirmationID: strings.TrimSpace(string(d.Raw[segmentZConfirmationIDStart:segmentZConfirmationIDEnd])),
	}, true
}
