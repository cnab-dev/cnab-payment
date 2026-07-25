package itau240

import (
	"fmt"
	"time"

	"github.com/cnab-dev/cnab-payment/internal/core"
)

// parseState holds the mutable state of a single Parse call: the current
// batch context and the detail records pending a flush. LayoutParser
// itself stays stateless so a single instance can be reused, including
// across concurrent Parse calls; a fresh parseState is created per call.
type parseState struct {
	s             core.Scanner
	onOccurrence  core.OccurrenceHandler
	onRecordError core.RecordErrorHandler

	result core.ParseResult

	createdAt time.Time

	inBatch              bool
	currentBatch         string
	currentPaymentType   string
	currentPaymentMethod string
	pending              []detail // consecutive detail records sharing a sequence number
}

func (p *parseState) run(first core.Record) (core.ParseResult, error) {
	if err := p.parseFileHeader(first); err != nil {
		return p.result, err
	}
	p.result.RecordsRead++

	for p.s.Next() {
		rec := p.s.Record()
		p.result.RecordsRead++

		if len(rec.Raw) != recordLength {
			if err := p.reportRecordError(rec, fmt.Errorf("itau240: record must be %d bytes, got %d", recordLength, len(rec.Raw))); err != nil {
				return p.result, err
			}
			continue
		}

		switch recordType(rec.Raw) {
		case recordTypeBatchHeader:
			if err := p.parseBatchHeader(rec); err != nil {
				return p.result, err
			}
		case recordTypeDetail:
			if err := p.parseDetail(rec); err != nil {
				return p.result, err
			}
		case recordTypeBatchTrailer:
			if err := p.parseBatchTrailer(rec); err != nil {
				return p.result, err
			}
		case recordTypeFileTrailer:
			if err := p.parseFileTrailer(rec); err != nil {
				return p.result, err
			}
			if err := p.s.Err(); err != nil {
				return p.result, err
			}
			return p.result, nil
		case recordTypeFileHeader:
			return p.result, fmt.Errorf("%w: unexpected file header at line %d", ErrUnexpectedRecord, rec.Line)
		default:
			if err := p.reportRecordError(rec, fmt.Errorf("itau240: unrecognized record type %q", recordType(rec.Raw))); err != nil {
				return p.result, err
			}
		}
	}

	if err := p.s.Err(); err != nil {
		return p.result, err
	}
	return p.result, ErrMissingFileTrailer
}

func (p *parseState) parseFileHeader(rec core.Record) error {
	fh, err := decodeFileHeader(rec.Raw)
	if err != nil {
		return err
	}
	p.createdAt = fh.CreatedAt
	return nil
}

func (p *parseState) parseBatchHeader(rec core.Record) error {
	if p.inBatch {
		return fmt.Errorf("%w: batch header at line %d while batch %s is still open", ErrUnexpectedRecord, rec.Line, p.currentBatch)
	}
	bh, err := decodeBatchHeader(rec.Raw)
	if err != nil {
		return err
	}
	p.inBatch = true
	p.currentBatch = bh.BatchNumber
	p.currentPaymentType = bh.PaymentType
	p.currentPaymentMethod = bh.PaymentMethod
	return nil
}

func (p *parseState) parseDetail(rec core.Record) error {
	if !p.inBatch {
		return fmt.Errorf("%w: detail record at line %d with no open batch", ErrUnexpectedRecord, rec.Line)
	}
	d, err := decodeDetail(rec)
	if err != nil {
		return err
	}
	if d.BatchNumber != p.currentBatch {
		return fmt.Errorf("%w: detail record at line %d belongs to batch %s, want %s", ErrUnexpectedRecord, rec.Line, d.BatchNumber, p.currentBatch)
	}

	if len(p.pending) > 0 && p.pending[0].Sequence != d.Sequence {
		if err := p.flushCurrentDetail(); err != nil {
			return err
		}
	}
	p.pending = append(p.pending, d)
	return nil
}

func (p *parseState) parseBatchTrailer(rec core.Record) error {
	if !p.inBatch {
		return fmt.Errorf("%w: batch trailer at line %d with no open batch", ErrUnexpectedRecord, rec.Line)
	}
	bt, err := decodeBatchTrailer(rec.Raw)
	if err != nil {
		return err
	}
	if bt.BatchNumber != p.currentBatch {
		return fmt.Errorf("%w: batch trailer at line %d closes batch %s, want %s", ErrUnexpectedRecord, rec.Line, bt.BatchNumber, p.currentBatch)
	}

	if err := p.flushCurrentDetail(); err != nil {
		return err
	}

	p.inBatch = false
	p.currentBatch = ""
	p.currentPaymentType = ""
	p.currentPaymentMethod = ""
	return nil
}

func (p *parseState) parseFileTrailer(rec core.Record) error {
	if err := decodeFileTrailer(rec.Raw); err != nil {
		return err
	}
	if p.inBatch {
		return fmt.Errorf("%w: file trailer at line %d while batch %s is still open", ErrUnexpectedRecord, rec.Line, p.currentBatch)
	}
	return nil
}

// flushCurrentDetail builds a canonical occurrence out of the detail
// records accumulated so far for the current logical payment, emits it,
// and clears the pending group. It is called whenever the sequence number
// changes and when the batch trailer is reached; it is a no-op if nothing
// is pending.
//
// A logical payment's fields are typically spread across more than one
// segment (e.g. the mandatory segment carries PaymentID/RawType, an
// optional following Segment Z carries ConfirmationID), so each segment's
// contribution is merged field by field rather than one segment's result
// replacing another's.
func (p *parseState) flushCurrentDetail() error {
	if len(p.pending) == 0 {
		return nil
	}
	group := p.pending
	p.pending = nil

	var fields occurrenceFields
	var found bool
	for _, d := range group {
		extract, ok := segmentExtractors[d.Segment]
		if !ok {
			if !isSegmentLetter(d.Segment) {
				if err := p.reportRecordError(core.Record{Raw: d.Raw, Line: d.Line}, fmt.Errorf("itau240: invalid segment code %q", d.Segment)); err != nil {
					return err
				}
			}
			continue
		}
		f, ok := extract(d)
		if !ok {
			continue
		}
		if f.PaymentID != "" {
			fields.PaymentID = f.PaymentID
		}
		if f.RawType != "" {
			fields.RawType = f.RawType
		}
		if f.ConfirmationID != "" {
			fields.ConfirmationID = f.ConfirmationID
		}
		if f.ExternalID != "" {
			fields.ExternalID = f.ExternalID
		}
		if f.Segment != 0 {
			fields.Segment = f.Segment
		}
		found = true
	}

	if !found || fields.PaymentID == "" || fields.RawType == "" {
		anchor := group[0]
		return p.reportRecordError(core.Record{Raw: anchor.Raw, Line: anchor.Line}, fmt.Errorf("itau240: could not extract an occurrence from the detail group starting at line %d", anchor.Line))
	}

	if err := p.onOccurrence(core.Occurrence{
		PaymentID:      fields.PaymentID,
		RawType:        fields.RawType,
		Type:           classifyRawType(fields.RawType),
		ConfirmationID: fields.ConfirmationID,
		ExternalID:     fields.ExternalID,
		CreatedAt:      p.createdAt,
		Format:         core.NewReceiptFormat(bankCode, fields.Segment, p.currentPaymentType, p.currentPaymentMethod),
	}); err != nil {
		return err
	}
	p.result.OccurrencesEmitted++
	return nil
}

func (p *parseState) reportRecordError(rec core.Record, cause error) error {
	if p.onRecordError == nil {
		return nil
	}
	return p.onRecordError(core.RecordError{
		Line: rec.Line,
		Raw:  rec.Raw,
		Err:  cause,
	})
}
