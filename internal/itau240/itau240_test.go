package itau240

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"github.com/cnab-dev/cnab-payment/internal/core"
)

func blank() []byte {
	return bytes.Repeat([]byte(" "), recordLength)
}

func setField(raw []byte, start, end int, value string) {
	copy(raw[start:end], value)
}

// fileHeaderBytes builds a file header record. dateTime must be a 14-char
// DDMMAAAAHHMMSS value (columns 144-157).
func fileHeaderBytes(bank string, kind byte, dateTime string) []byte {
	raw := blank()
	setField(raw, 0, 3, bank)
	raw[7] = recordTypeFileHeader
	raw[142] = kind
	setField(raw, 143, 157, dateTime)
	return raw
}

func batchHeaderBytes(batch string) []byte {
	raw := blank()
	setField(raw, 3, 7, batch)
	raw[7] = recordTypeBatchHeader
	return raw
}

// batchHeaderBytesWithPayment builds a batch header record like
// batchHeaderBytes, additionally setting the "Tipo de Serviço"/"Forma de
// Lançamento" fields — used to exercise ReceiptFormat construction.
func batchHeaderBytesWithPayment(batch, paymentType, paymentMethod string) []byte {
	raw := batchHeaderBytes(batch)
	setField(raw, batchHeaderPaymentTypeStart, batchHeaderPaymentTypeEnd, paymentType)
	setField(raw, batchHeaderPaymentMethodStart, batchHeaderPaymentMethodEnd, paymentMethod)
	return raw
}

func detailBytes(batch, seq string, segment byte, paymentID, rawType string) []byte {
	raw := blank()
	setField(raw, 3, 7, batch)
	raw[7] = recordTypeDetail
	setField(raw, 8, 13, seq)
	raw[13] = segment
	setField(raw, segmentAPaymentIDStart, segmentAPaymentIDEnd, paymentID)
	setField(raw, occurrenceCodeStart, occurrenceCodeEnd, rawType)
	return raw
}

// detailBytesAt builds a detail record for segment, placing paymentID at
// the given field bounds instead of assuming Segment A's — used to
// exercise the other mandatory segments' own extractors.
func detailBytesAt(batch, seq string, segment byte, idStart, idEnd int, paymentID, rawType string) []byte {
	raw := blank()
	setField(raw, 3, 7, batch)
	raw[7] = recordTypeDetail
	setField(raw, 8, 13, seq)
	raw[13] = segment
	setField(raw, idStart, idEnd, paymentID)
	setField(raw, occurrenceCodeStart, occurrenceCodeEnd, rawType)
	return raw
}

func segmentZBytes(batch, seq, confirmationID string) []byte {
	raw := blank()
	setField(raw, 3, 7, batch)
	raw[7] = recordTypeDetail
	setField(raw, 8, 13, seq)
	raw[13] = 'Z'
	setField(raw, segmentZConfirmationIDStart, segmentZConfirmationIDEnd, confirmationID)
	return raw
}

func batchTrailerBytes(batch string) []byte {
	raw := blank()
	setField(raw, 3, 7, batch)
	raw[7] = recordTypeBatchTrailer
	return raw
}

func fileTrailerBytes() []byte {
	raw := blank()
	raw[7] = recordTypeFileTrailer
	return raw
}

// setupScanner writes lines newline-joined into a scanner, reads the
// first record (mirroring what parser.Parser does before delegating to a
// LayoutParser), and returns it alongside the still-open scanner.
func setupScanner(t *testing.T, lines [][]byte) (core.Record, core.Scanner) {
	t.Helper()
	var buf bytes.Buffer
	for _, l := range lines {
		buf.Write(l)
		buf.WriteByte('\n')
	}
	s := core.NewScanner(&buf)
	if !s.Next() {
		t.Fatalf("scanner had no records")
	}
	return s.Record(), s
}

// sliceScanner is a core.Scanner test double that replays a fixed list of
// records and always reports err from Err, regardless of position —
// including right after successfully returning the last record. It exists
// to exercise run()'s scanner-error checks, which the real line-based
// scanner can't be made to hit once a record has been returned
// successfully.
type sliceScanner struct {
	recs []core.Record
	idx  int
	err  error
}

func (s *sliceScanner) Next() bool {
	if s.idx >= len(s.recs) {
		return false
	}
	s.idx++
	return true
}

func (s *sliceScanner) Record() core.Record { return s.recs[s.idx-1] }

func (s *sliceScanner) Err() error { return s.err }

func recsFromLines(lines [][]byte) []core.Record {
	recs := make([]core.Record, len(lines))
	for i, l := range lines {
		recs[i] = core.Record{Raw: l, Line: i + 2} // +2: line 1 is the file header, passed as `first`
	}
	return recs
}

func TestDetect(t *testing.T) {
	tests := []struct {
		name string
		raw  []byte
		want bool
	}{
		{"validItauRetornoHeader_returnsTrue", fileHeaderBytes(bankCode, fileKindRetorno, "19072026120000"), true},
		{"wrongBankCode_returnsFalse", fileHeaderBytes("237", fileKindRetorno, "19072026120000"), false},
		{"remessaFileKind_returnsFalse", fileHeaderBytes(bankCode, fileKindRemessa, "19072026120000"), false},
		{"wrongRecordType_returnsFalse", func() []byte { r := fileHeaderBytes(bankCode, fileKindRetorno, "19072026120000"); r[7] = '1'; return r }(), false},
		{"invalidGenerationDateTime_returnsFalse", fileHeaderBytes(bankCode, fileKindRetorno, "not-a-date!!!!"), false},
		{"shortRecord_returnsFalse", []byte("too short"), false},
	}

	lp := New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := lp.Detect(core.Record{Raw: tt.raw, Line: 1}); got != tt.want {
				t.Errorf("Detect() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParse_SingleSegmentADetail_EmitsOccurrenceWithStats(t *testing.T) {
	lines := [][]byte{
		fileHeaderBytes(bankCode, fileKindRetorno, "19072026120000"),
		batchHeaderBytes("0001"),
		detailBytes("0001", "00002", 'A', "PAY-1", "02"),
		batchTrailerBytes("0001"),
		fileTrailerBytes(),
	}
	first, s := setupScanner(t, lines)

	var occurrences []core.Occurrence
	result, err := New().Parse(s, first, func(o core.Occurrence) error {
		occurrences = append(occurrences, o)
		return nil
	}, nil)
	if err != nil {
		t.Fatalf("Parse() error = %v, want nil", err)
	}
	if len(occurrences) != 1 {
		t.Fatalf("got %d occurrences, want 1", len(occurrences))
	}
	if occurrences[0].PaymentID != "PAY-1" || occurrences[0].RawType != "02" {
		t.Errorf("occurrence = %+v, want PaymentID=PAY-1 RawType=02", occurrences[0])
	}
	wantCreatedAt := time.Date(2026, time.July, 19, 12, 0, 0, 0, brasiliaLocation)
	if !occurrences[0].CreatedAt.Equal(wantCreatedAt) {
		t.Errorf("CreatedAt = %v, want %v", occurrences[0].CreatedAt, wantCreatedAt)
	}
	if result.RecordsRead != 5 {
		t.Errorf("RecordsRead = %d, want 5", result.RecordsRead)
	}
	if result.OccurrencesEmitted != 1 {
		t.Errorf("OccurrencesEmitted = %d, want 1", result.OccurrencesEmitted)
	}
}

func TestParse_ClassifiesRawTypeIntoOccurrenceType(t *testing.T) {
	tests := []struct {
		name    string
		rawType string
		want    core.OccurrenceType
	}{
		{"rjPrefix_setsTypeRejected", "RJNR", core.OccurrenceTypeRejected},
		{"bdPrefix_setsTypeScheduled", "BD  ", core.OccurrenceTypeScheduled},
		{"zeroZeroPrefix_setsTypeSettled", "00", core.OccurrenceTypeSettled},
		{"unrecognizedPrefix_setsTypeUnknown", "02", core.OccurrenceTypeUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lines := [][]byte{
				fileHeaderBytes(bankCode, fileKindRetorno, "19072026120000"),
				batchHeaderBytes("0001"),
				detailBytes("0001", "00002", 'A', "PAY-1", tt.rawType),
				batchTrailerBytes("0001"),
				fileTrailerBytes(),
			}
			first, s := setupScanner(t, lines)

			var occurrences []core.Occurrence
			_, err := New().Parse(s, first, func(o core.Occurrence) error {
				occurrences = append(occurrences, o)
				return nil
			}, nil)
			if err != nil {
				t.Fatalf("Parse() error = %v, want nil", err)
			}
			if len(occurrences) != 1 {
				t.Fatalf("got %d occurrences, want 1", len(occurrences))
			}
			if got := occurrences[0].Type; got != tt.want {
				t.Errorf("Type = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParse_MandatorySegmentVariants_ExtractsPaymentIDAndRawType(t *testing.T) {
	tests := []struct {
		name    string
		segment byte
		idStart int
		idEnd   int
	}{
		{"segmentA_extractsPaymentIDAndRawType", 'A', segmentAPaymentIDStart, segmentAPaymentIDEnd},
		{"segmentJ_extractsPaymentIDAndRawType", 'J', segmentJPaymentIDStart, segmentJPaymentIDEnd},
		{"segmentO_extractsPaymentIDAndRawType", 'O', segmentOPaymentIDStart, segmentOPaymentIDEnd},
		{"segmentN_extractsPaymentIDAndRawType", 'N', segmentNPaymentIDStart, segmentNPaymentIDEnd},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lines := [][]byte{
				fileHeaderBytes(bankCode, fileKindRetorno, "19072026120000"),
				batchHeaderBytes("0001"),
				detailBytesAt("0001", "00002", tt.segment, tt.idStart, tt.idEnd, "PAY-1", "02"),
				batchTrailerBytes("0001"),
				fileTrailerBytes(),
			}
			first, s := setupScanner(t, lines)

			var occurrences []core.Occurrence
			_, err := New().Parse(s, first, func(o core.Occurrence) error {
				occurrences = append(occurrences, o)
				return nil
			}, nil)
			if err != nil {
				t.Fatalf("Parse() error = %v, want nil", err)
			}
			if len(occurrences) != 1 {
				t.Fatalf("got %d occurrences, want 1", len(occurrences))
			}
			if occurrences[0].PaymentID != "PAY-1" || occurrences[0].RawType != "02" {
				t.Errorf("occurrence = %+v, want PaymentID=PAY-1 RawType=02", occurrences[0])
			}
		})
	}
}

// TestParse_BuildsReceiptFormat covers real Tipo de Serviço/Forma de
// Lançamento combinations (mirroring conapag-cnab's loteSpecs table) to
// document the expected BBB-S-TT-FF key per payment kind and catch
// regressions in batch header field positions.
func TestParse_BuildsReceiptFormat(t *testing.T) {
	tests := []struct {
		name          string
		segment       byte
		idStart       int
		idEnd         int
		paymentType   string
		paymentMethod string
		want          core.ReceiptFormat
	}{
		{"transferChecking_ted", 'A', segmentAPaymentIDStart, segmentAPaymentIDEnd, "20", "41", "341-A-20-41"},
		{"transferPix", 'A', segmentAPaymentIDStart, segmentAPaymentIDEnd, "20", "45", "341-A-20-45"},
		{"transferCorrente", 'A', segmentAPaymentIDStart, segmentAPaymentIDEnd, "20", "01", "341-A-20-01"},
		{"transferPoupanca", 'A', segmentAPaymentIDStart, segmentAPaymentIDEnd, "20", "05", "341-A-20-05"},
		{"transferSalario", 'A', segmentAPaymentIDStart, segmentAPaymentIDEnd, "30", "01", "341-A-30-01"},
		{"boletoItau", 'J', segmentJPaymentIDStart, segmentJPaymentIDEnd, "20", "30", "341-J-20-30"},
		{"boletoOutrosBancos", 'J', segmentJPaymentIDStart, segmentJPaymentIDEnd, "20", "31", "341-J-20-31"},
		{"arrecadacao", 'O', segmentOPaymentIDStart, segmentOPaymentIDEnd, "22", "91", "341-O-22-91"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lines := [][]byte{
				fileHeaderBytes(bankCode, fileKindRetorno, "19072026120000"),
				batchHeaderBytesWithPayment("0001", tt.paymentType, tt.paymentMethod),
				detailBytesAt("0001", "00002", tt.segment, tt.idStart, tt.idEnd, "PAY-1", "02"),
				batchTrailerBytes("0001"),
				fileTrailerBytes(),
			}
			first, s := setupScanner(t, lines)

			var occurrences []core.Occurrence
			_, err := New().Parse(s, first, func(o core.Occurrence) error {
				occurrences = append(occurrences, o)
				return nil
			}, nil)
			if err != nil {
				t.Fatalf("Parse() error = %v, want nil", err)
			}
			if len(occurrences) != 1 {
				t.Fatalf("got %d occurrences, want 1", len(occurrences))
			}
			if got := occurrences[0].Format; got != tt.want {
				t.Errorf("Format = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParse_ConsecutiveDetailsShareSequence_GroupsIntoSingleOccurrence(t *testing.T) {
	lines := [][]byte{
		fileHeaderBytes(bankCode, fileKindRetorno, "19072026120000"),
		batchHeaderBytes("0001"),
		detailBytes("0001", "00002", 'A', "PAY-1", "02"),
		detailBytes("0001", "00002", 'B', "", ""), // same sequence, unimplemented segment
		batchTrailerBytes("0001"),
		fileTrailerBytes(),
	}
	first, s := setupScanner(t, lines)

	var occurrences []core.Occurrence
	var recordErrs []core.RecordError
	result, err := New().Parse(s, first,
		func(o core.Occurrence) error { occurrences = append(occurrences, o); return nil },
		func(re core.RecordError) error { recordErrs = append(recordErrs, re); return nil },
	)
	if err != nil {
		t.Fatalf("Parse() error = %v, want nil", err)
	}
	if len(occurrences) != 1 {
		t.Fatalf("got %d occurrences, want 1 (both records should be grouped into a single logical payment)", len(occurrences))
	}
	if len(recordErrs) != 0 {
		t.Fatalf("got %d record errors, want 0: %v", len(recordErrs), recordErrs)
	}
	if result.RecordsRead != 6 {
		t.Errorf("RecordsRead = %d, want 6", result.RecordsRead)
	}
}

func TestParse_SegmentZPresent_AddsConfirmationID(t *testing.T) {
	lines := [][]byte{
		fileHeaderBytes(bankCode, fileKindRetorno, "19072026120000"),
		batchHeaderBytes("0001"),
		detailBytes("0001", "00002", 'A', "PAY-1", "02"),
		segmentZBytes("0001", "00002", "CONF-123"),
		batchTrailerBytes("0001"),
		fileTrailerBytes(),
	}
	first, s := setupScanner(t, lines)

	var occurrences []core.Occurrence
	var recordErrs []core.RecordError
	_, err := New().Parse(s, first,
		func(o core.Occurrence) error { occurrences = append(occurrences, o); return nil },
		func(re core.RecordError) error { recordErrs = append(recordErrs, re); return nil },
	)
	if err != nil {
		t.Fatalf("Parse() error = %v, want nil", err)
	}
	if len(recordErrs) != 0 {
		t.Fatalf("got %d record errors, want 0: %v", len(recordErrs), recordErrs)
	}
	if len(occurrences) != 1 {
		t.Fatalf("got %d occurrences, want 1", len(occurrences))
	}
	got := occurrences[0]
	if got.PaymentID != "PAY-1" || got.RawType != "02" || got.ConfirmationID != "CONF-123" {
		t.Errorf("occurrence = %+v, want PaymentID=PAY-1 RawType=02 ConfirmationID=CONF-123", got)
	}
}

func TestParse_NoSegmentZPresent_LeavesConfirmationIDEmpty(t *testing.T) {
	lines := [][]byte{
		fileHeaderBytes(bankCode, fileKindRetorno, "19072026120000"),
		batchHeaderBytes("0001"),
		detailBytes("0001", "00002", 'A', "PAY-1", "02"),
		batchTrailerBytes("0001"),
		fileTrailerBytes(),
	}
	first, s := setupScanner(t, lines)

	var occurrences []core.Occurrence
	_, err := New().Parse(s, first, func(o core.Occurrence) error {
		occurrences = append(occurrences, o)
		return nil
	}, nil)
	if err != nil {
		t.Fatalf("Parse() error = %v, want nil", err)
	}
	if len(occurrences) != 1 {
		t.Fatalf("got %d occurrences, want 1", len(occurrences))
	}
	if got := occurrences[0].ConfirmationID; got != "" {
		t.Errorf("ConfirmationID = %q, want empty when no Segment Z is present", got)
	}
}

func TestParse_SequenceNumberChanges_StartsNewOccurrence(t *testing.T) {
	lines := [][]byte{
		fileHeaderBytes(bankCode, fileKindRetorno, "19072026120000"),
		batchHeaderBytes("0001"),
		detailBytes("0001", "00002", 'A', "PAY-1", "02"),
		detailBytes("0001", "00003", 'A', "PAY-2", "03"),
		batchTrailerBytes("0001"),
		fileTrailerBytes(),
	}
	first, s := setupScanner(t, lines)

	var occurrences []core.Occurrence
	_, err := New().Parse(s, first, func(o core.Occurrence) error {
		occurrences = append(occurrences, o)
		return nil
	}, nil)
	if err != nil {
		t.Fatalf("Parse() error = %v, want nil", err)
	}
	if len(occurrences) != 2 {
		t.Fatalf("got %d occurrences, want 2", len(occurrences))
	}
	if occurrences[0].PaymentID != "PAY-1" || occurrences[1].PaymentID != "PAY-2" {
		t.Errorf("occurrences = %+v, want PAY-1 then PAY-2", occurrences)
	}
}

func TestParse_DetailRecordWithNoOpenBatch_ReturnsErrUnexpectedRecord(t *testing.T) {
	lines := [][]byte{
		fileHeaderBytes(bankCode, fileKindRetorno, "19072026120000"),
		detailBytes("0001", "00002", 'A', "PAY-1", "02"),
	}
	first, s := setupScanner(t, lines)

	_, err := New().Parse(s, first, func(core.Occurrence) error { return nil }, nil)
	if !errors.Is(err, ErrUnexpectedRecord) {
		t.Fatalf("Parse() error = %v, want ErrUnexpectedRecord", err)
	}
}

func TestParse_InputEndsWithoutFileTrailer_ReturnsErrMissingFileTrailer(t *testing.T) {
	lines := [][]byte{
		fileHeaderBytes(bankCode, fileKindRetorno, "19072026120000"),
		batchHeaderBytes("0001"),
		batchTrailerBytes("0001"),
	}
	first, s := setupScanner(t, lines)

	_, err := New().Parse(s, first, func(core.Occurrence) error { return nil }, nil)
	if !errors.Is(err, ErrMissingFileTrailer) {
		t.Fatalf("Parse() error = %v, want ErrMissingFileTrailer", err)
	}
}

func TestParse_UnrecognizedRecordType_ReportsRecordErrorAndContinues(t *testing.T) {
	garbage := blank()
	garbage[7] = '7'

	lines := [][]byte{
		fileHeaderBytes(bankCode, fileKindRetorno, "19072026120000"),
		batchHeaderBytes("0001"),
		garbage,
		batchTrailerBytes("0001"),
		fileTrailerBytes(),
	}
	first, s := setupScanner(t, lines)

	var recordErrs []core.RecordError
	result, err := New().Parse(s, first, func(core.Occurrence) error { return nil }, func(re core.RecordError) error {
		recordErrs = append(recordErrs, re)
		return nil
	})
	if err != nil {
		t.Fatalf("Parse() error = %v, want nil", err)
	}
	if len(recordErrs) != 1 {
		t.Fatalf("got %d record errors, want 1", len(recordErrs))
	}
	if recordErrs[0].Line != 3 {
		t.Errorf("record error line = %d, want 3", recordErrs[0].Line)
	}
	if result.RecordsRead != 5 {
		t.Errorf("RecordsRead = %d, want 5", result.RecordsRead)
	}
}

func TestParse_NilRecordErrorHandler_IgnoresRecoverableErrorsAndContinues(t *testing.T) {
	garbage := blank()
	garbage[7] = '7'

	lines := [][]byte{
		fileHeaderBytes(bankCode, fileKindRetorno, "19072026120000"),
		batchHeaderBytes("0001"),
		garbage,
		detailBytes("0001", "00002", 'A', "PAY-1", "02"),
		batchTrailerBytes("0001"),
		fileTrailerBytes(),
	}
	first, s := setupScanner(t, lines)

	var occurrences []core.Occurrence
	_, err := New().Parse(s, first, func(o core.Occurrence) error {
		occurrences = append(occurrences, o)
		return nil
	}, nil)
	if err != nil {
		t.Fatalf("Parse() error = %v, want nil", err)
	}
	if len(occurrences) != 1 {
		t.Fatalf("got %d occurrences, want 1", len(occurrences))
	}
}

func TestParse_RecordErrorHandlerReturnsError_AbortsParsing(t *testing.T) {
	garbage := blank()
	garbage[7] = '7'
	wantErr := errors.New("stop")

	lines := [][]byte{
		fileHeaderBytes(bankCode, fileKindRetorno, "19072026120000"),
		batchHeaderBytes("0001"),
		garbage,
		batchTrailerBytes("0001"),
		fileTrailerBytes(),
	}
	first, s := setupScanner(t, lines)

	_, err := New().Parse(s, first, func(core.Occurrence) error { return nil }, func(core.RecordError) error {
		return wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Parse() error = %v, want %v", err, wantErr)
	}
}

func TestParse_OccurrenceHandlerReturnsError_AbortsParsing(t *testing.T) {
	wantErr := errors.New("stop")
	lines := [][]byte{
		fileHeaderBytes(bankCode, fileKindRetorno, "19072026120000"),
		batchHeaderBytes("0001"),
		detailBytes("0001", "00002", 'A', "PAY-1", "02"),
		batchTrailerBytes("0001"),
		fileTrailerBytes(),
	}
	first, s := setupScanner(t, lines)

	_, err := New().Parse(s, first, func(core.Occurrence) error { return wantErr }, nil)
	if !errors.Is(err, wantErr) {
		t.Fatalf("Parse() error = %v, want %v", err, wantErr)
	}
}

func TestParse_InvalidFileHeaderAsFirstRecord_ReturnsError(t *testing.T) {
	first := core.Record{Raw: blank(), Line: 1} // record type ' ', not '0'

	_, err := New().Parse(&sliceScanner{}, first, func(core.Occurrence) error { return nil }, nil)
	if err == nil {
		t.Fatalf("Parse() error = nil, want a decode error")
	}
}

func TestParse_UnexpectedFileHeaderMidStream_ReturnsErrUnexpectedRecord(t *testing.T) {
	lines := [][]byte{
		fileHeaderBytes(bankCode, fileKindRetorno, "19072026120000"),
		fileHeaderBytes(bankCode, fileKindRetorno, "19072026120000"),
	}
	first, s := setupScanner(t, lines)

	_, err := New().Parse(s, first, func(core.Occurrence) error { return nil }, nil)
	if !errors.Is(err, ErrUnexpectedRecord) {
		t.Fatalf("Parse() error = %v, want ErrUnexpectedRecord", err)
	}
}

func TestParse_BatchHeaderWhileBatchAlreadyOpen_ReturnsErrUnexpectedRecord(t *testing.T) {
	lines := [][]byte{
		fileHeaderBytes(bankCode, fileKindRetorno, "19072026120000"),
		batchHeaderBytes("0001"),
		batchHeaderBytes("0002"),
	}
	first, s := setupScanner(t, lines)

	_, err := New().Parse(s, first, func(core.Occurrence) error { return nil }, nil)
	if !errors.Is(err, ErrUnexpectedRecord) {
		t.Fatalf("Parse() error = %v, want ErrUnexpectedRecord", err)
	}
}

func TestParse_BatchTrailerWithNoOpenBatch_ReturnsErrUnexpectedRecord(t *testing.T) {
	lines := [][]byte{
		fileHeaderBytes(bankCode, fileKindRetorno, "19072026120000"),
		batchTrailerBytes("0001"),
	}
	first, s := setupScanner(t, lines)

	_, err := New().Parse(s, first, func(core.Occurrence) error { return nil }, nil)
	if !errors.Is(err, ErrUnexpectedRecord) {
		t.Fatalf("Parse() error = %v, want ErrUnexpectedRecord", err)
	}
}

func TestParse_BatchTrailerBatchNumberMismatch_ReturnsErrUnexpectedRecord(t *testing.T) {
	lines := [][]byte{
		fileHeaderBytes(bankCode, fileKindRetorno, "19072026120000"),
		batchHeaderBytes("0001"),
		batchTrailerBytes("0002"),
	}
	first, s := setupScanner(t, lines)

	_, err := New().Parse(s, first, func(core.Occurrence) error { return nil }, nil)
	if !errors.Is(err, ErrUnexpectedRecord) {
		t.Fatalf("Parse() error = %v, want ErrUnexpectedRecord", err)
	}
}

func TestParse_DetailBatchNumberMismatch_ReturnsErrUnexpectedRecord(t *testing.T) {
	lines := [][]byte{
		fileHeaderBytes(bankCode, fileKindRetorno, "19072026120000"),
		batchHeaderBytes("0001"),
		detailBytes("0002", "00002", 'A', "PAY-1", "02"),
	}
	first, s := setupScanner(t, lines)

	_, err := New().Parse(s, first, func(core.Occurrence) error { return nil }, nil)
	if !errors.Is(err, ErrUnexpectedRecord) {
		t.Fatalf("Parse() error = %v, want ErrUnexpectedRecord", err)
	}
}

func TestParse_FileTrailerWhileBatchStillOpen_ReturnsErrUnexpectedRecord(t *testing.T) {
	lines := [][]byte{
		fileHeaderBytes(bankCode, fileKindRetorno, "19072026120000"),
		batchHeaderBytes("0001"),
		fileTrailerBytes(),
	}
	first, s := setupScanner(t, lines)

	_, err := New().Parse(s, first, func(core.Occurrence) error { return nil }, nil)
	if !errors.Is(err, ErrUnexpectedRecord) {
		t.Fatalf("Parse() error = %v, want ErrUnexpectedRecord", err)
	}
}

func TestParse_RecordWithWrongLength_ReportsRecordErrorAndContinues(t *testing.T) {
	lines := [][]byte{
		fileHeaderBytes(bankCode, fileKindRetorno, "19072026120000"),
		batchHeaderBytes("0001"),
		[]byte("too short"),
		batchTrailerBytes("0001"),
		fileTrailerBytes(),
	}
	first, s := setupScanner(t, lines)

	var recordErrs []core.RecordError
	_, err := New().Parse(s, first, func(core.Occurrence) error { return nil }, func(re core.RecordError) error {
		recordErrs = append(recordErrs, re)
		return nil
	})
	if err != nil {
		t.Fatalf("Parse() error = %v, want nil", err)
	}
	if len(recordErrs) != 1 {
		t.Fatalf("got %d record errors, want 1", len(recordErrs))
	}
}

func TestParse_RecordWithWrongLengthHandlerReturnsError_AbortsParsing(t *testing.T) {
	wantErr := errors.New("stop")
	lines := [][]byte{
		fileHeaderBytes(bankCode, fileKindRetorno, "19072026120000"),
		batchHeaderBytes("0001"),
		[]byte("too short"),
		batchTrailerBytes("0001"),
		fileTrailerBytes(),
	}
	first, s := setupScanner(t, lines)

	_, err := New().Parse(s, first, func(core.Occurrence) error { return nil }, func(core.RecordError) error {
		return wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Parse() error = %v, want %v", err, wantErr)
	}
}

func TestParse_InvalidSegmentCodeHandlerReturnsError_AbortsParsing(t *testing.T) {
	wantErr := errors.New("stop")
	lines := [][]byte{
		fileHeaderBytes(bankCode, fileKindRetorno, "19072026120000"),
		batchHeaderBytes("0001"),
		detailBytes("0001", "00002", '#', "", ""),
		batchTrailerBytes("0001"),
		fileTrailerBytes(),
	}
	first, s := setupScanner(t, lines)

	_, err := New().Parse(s, first, func(core.Occurrence) error { return nil }, func(core.RecordError) error {
		return wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Parse() error = %v, want %v", err, wantErr)
	}
}

func TestParse_InvalidSegmentCodeAlongsideValidSegment_ReportsRecordErrorAndStillEmitsOccurrence(t *testing.T) {
	lines := [][]byte{
		fileHeaderBytes(bankCode, fileKindRetorno, "19072026120000"),
		batchHeaderBytes("0001"),
		detailBytes("0001", "00002", '#', "", ""),
		detailBytes("0001", "00002", 'A', "PAY-1", "02"),
		batchTrailerBytes("0001"),
		fileTrailerBytes(),
	}
	first, s := setupScanner(t, lines)

	var occurrences []core.Occurrence
	var recordErrs []core.RecordError
	_, err := New().Parse(s, first,
		func(o core.Occurrence) error { occurrences = append(occurrences, o); return nil },
		func(re core.RecordError) error { recordErrs = append(recordErrs, re); return nil },
	)
	if err != nil {
		t.Fatalf("Parse() error = %v, want nil", err)
	}
	if len(recordErrs) != 1 {
		t.Fatalf("got %d record errors, want 1", len(recordErrs))
	}
	if len(occurrences) != 1 || occurrences[0].PaymentID != "PAY-1" {
		t.Errorf("occurrences = %+v, want a single occurrence with PaymentID=PAY-1", occurrences)
	}
}

func TestParse_GroupWithOnlyUnimplementedSegment_ReportsCouldNotExtractAndContinues(t *testing.T) {
	lines := [][]byte{
		fileHeaderBytes(bankCode, fileKindRetorno, "19072026120000"),
		batchHeaderBytes("0001"),
		detailBytes("0001", "00002", 'B', "", ""),
		batchTrailerBytes("0001"),
		fileTrailerBytes(),
	}
	first, s := setupScanner(t, lines)

	var occurrences []core.Occurrence
	var recordErrs []core.RecordError
	_, err := New().Parse(s, first,
		func(o core.Occurrence) error { occurrences = append(occurrences, o); return nil },
		func(re core.RecordError) error { recordErrs = append(recordErrs, re); return nil },
	)
	if err != nil {
		t.Fatalf("Parse() error = %v, want nil", err)
	}
	if len(occurrences) != 0 {
		t.Fatalf("got %d occurrences, want 0", len(occurrences))
	}
	if len(recordErrs) != 1 {
		t.Fatalf("got %d record errors, want 1", len(recordErrs))
	}
}

func TestParse_SegmentZWithoutMandatorySegment_ReportsCouldNotExtract(t *testing.T) {
	lines := [][]byte{
		fileHeaderBytes(bankCode, fileKindRetorno, "19072026120000"),
		batchHeaderBytes("0001"),
		segmentZBytes("0001", "00002", "CONF-1"),
		batchTrailerBytes("0001"),
		fileTrailerBytes(),
	}
	first, s := setupScanner(t, lines)

	var occurrences []core.Occurrence
	var recordErrs []core.RecordError
	_, err := New().Parse(s, first,
		func(o core.Occurrence) error { occurrences = append(occurrences, o); return nil },
		func(re core.RecordError) error { recordErrs = append(recordErrs, re); return nil },
	)
	if err != nil {
		t.Fatalf("Parse() error = %v, want nil", err)
	}
	if len(occurrences) != 0 {
		t.Fatalf("got %d occurrences, want 0", len(occurrences))
	}
	if len(recordErrs) != 1 {
		t.Fatalf("got %d record errors, want 1", len(recordErrs))
	}
}

func TestParse_CouldNotExtractOccurrenceAtBatchTrailer_AbortsParsing(t *testing.T) {
	wantErr := errors.New("stop")
	lines := [][]byte{
		fileHeaderBytes(bankCode, fileKindRetorno, "19072026120000"),
		batchHeaderBytes("0001"),
		detailBytes("0001", "00002", 'B', "", ""),
		batchTrailerBytes("0001"),
		fileTrailerBytes(),
	}
	first, s := setupScanner(t, lines)

	_, err := New().Parse(s, first, func(core.Occurrence) error { return nil }, func(core.RecordError) error {
		return wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Parse() error = %v, want %v", err, wantErr)
	}
}

func TestParse_FlushErrorOnSequenceChange_AbortsParsing(t *testing.T) {
	wantErr := errors.New("stop")
	lines := [][]byte{
		fileHeaderBytes(bankCode, fileKindRetorno, "19072026120000"),
		batchHeaderBytes("0001"),
		detailBytes("0001", "00002", 'B', "", ""),
		detailBytes("0001", "00003", 'A', "PAY-2", "02"),
		batchTrailerBytes("0001"),
		fileTrailerBytes(),
	}
	first, s := setupScanner(t, lines)

	_, err := New().Parse(s, first, func(core.Occurrence) error { return nil }, func(core.RecordError) error {
		return wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Parse() error = %v, want %v", err, wantErr)
	}
}

func TestParse_ScannerErrorAfterLoopWithoutFileTrailer_ReturnsScannerError(t *testing.T) {
	wantErr := errors.New("read failed")
	first := core.Record{Raw: fileHeaderBytes(bankCode, fileKindRetorno, "19072026120000"), Line: 1}
	lines := [][]byte{
		batchHeaderBytes("0001"),
		batchTrailerBytes("0001"),
	}
	s := &sliceScanner{recs: recsFromLines(lines), err: wantErr}

	_, err := New().Parse(s, first, func(core.Occurrence) error { return nil }, nil)
	if !errors.Is(err, wantErr) {
		t.Fatalf("Parse() error = %v, want %v", err, wantErr)
	}
}

func TestParse_ScannerErrorAfterFileTrailer_ReturnsScannerError(t *testing.T) {
	wantErr := errors.New("read failed")
	first := core.Record{Raw: fileHeaderBytes(bankCode, fileKindRetorno, "19072026120000"), Line: 1}
	lines := [][]byte{
		batchHeaderBytes("0001"),
		detailBytes("0001", "00002", 'A', "PAY-1", "02"),
		batchTrailerBytes("0001"),
		fileTrailerBytes(),
	}
	s := &sliceScanner{recs: recsFromLines(lines), err: wantErr}

	_, err := New().Parse(s, first, func(core.Occurrence) error { return nil }, nil)
	if !errors.Is(err, wantErr) {
		t.Fatalf("Parse() error = %v, want %v", err, wantErr)
	}
}
