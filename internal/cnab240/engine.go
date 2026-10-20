package cnab240

import (
	"time"

	"github.com/cnab-dev/cnab-payment/internal/core"
)

// Layout is the bank-specific half of a CNAB240 remittance: how payments
// are grouped into batches and how each record is laid out. Build drives
// it; a Layout never sees more than one batch at a time.
//
// Records returned by a Layout are normalized by Build (see
// RecordNormalizer), so fields only need to be padded to width.
type Layout interface {
	// GroupKey returns the batch a payment belongs to. Payments with the
	// same key share a batch; batches are written in the order their
	// first payment appears.
	GroupKey(p core.Payment) string

	BuildFileHeader(ctx FileContext) string
	BuildFileTrailer(totals FileTotals) string
	BuildBatchHeader(state *BuildState, batchKey string) string
	BuildBatchTrailer(state *BuildState, batchKey string, totals BatchTotals) string

	// BuildDetail returns the detail records (segments) for p and the
	// totals they contribute to the batch.
	BuildDetail(state *BuildState, p core.Payment) ([]string, BatchTotals)
}

// RecordNormalizer is optionally implemented by a Layout that must keep
// part of a record as is (e.g. a case-sensitive URL). Without it, every
// record is normalized with StripAccentsUpper.
type RecordNormalizer interface {
	NormalizeRecord(record string) string
}

// FileContext is what the file header is built from: the payer of the
// first payment, and the file's creation instant.
type FileContext struct {
	Payer        core.Party
	PayerAccount core.BankAccount
	CreatedAt    time.Time
}

// BuildState is the position of the record being built, plus the payer of
// the current batch (taken from the batch's first payment).
type BuildState struct {
	BatchSequence int // 1-based batch number within the file
	PaymentSeq    int // 1-based payment number within the batch
	Payer         core.Party
	PayerAccount  core.BankAccount
}

// BatchTotals accumulates what a batch trailer reports.
type BatchTotals struct {
	RecordCount  int
	PaymentCount int
	TotalAmount  core.Amount
}

// Add returns the sum of bt and other.
func (bt BatchTotals) Add(other BatchTotals) BatchTotals {
	return BatchTotals{
		RecordCount:  bt.RecordCount + other.RecordCount,
		PaymentCount: bt.PaymentCount + other.PaymentCount,
		TotalAmount:  bt.TotalAmount + other.TotalAmount,
	}
}

// FileTotals accumulates what the file trailer reports. RecordCount covers
// batch headers, details and batch trailers, but not the file header and
// trailer themselves.
type FileTotals struct {
	BatchCount  int
	RecordCount int
}

// File is an assembled remittance, with every record already normalized.
type File struct {
	Header  string
	Batches []Batch
	Trailer string
}

// Batch is one batch (lote) of a File.
type Batch struct {
	Header  string
	Details []string
	Trailer string
}

// Build assembles payments into a File using l. Payments are grouped into
// batches by l.GroupKey, preserving input order both across and within
// batches. payments must have already been validated by the layout and
// must not be empty.
func Build(l Layout, payments []core.Payment, createdAt time.Time) *File {
	normalize := StripAccentsUpper
	if n, ok := l.(RecordNormalizer); ok {
		normalize = n.NormalizeRecord
	}

	var ctx FileContext
	if len(payments) > 0 {
		first := payments[0].Base()
		ctx = FileContext{Payer: first.Payer, PayerAccount: first.PayerAccount}
	}
	ctx.CreatedAt = createdAt

	state := &BuildState{}
	var batches []Batch
	var fileTotals FileTotals

	for _, g := range groupPayments(payments, l.GroupKey) {
		first := g.payments[0].Base()
		state.BatchSequence++
		state.PaymentSeq = 0
		state.Payer = first.Payer
		state.PayerAccount = first.PayerAccount

		batch := Batch{Header: normalize(l.BuildBatchHeader(state, g.key))}
		batchTotals := BatchTotals{RecordCount: 1} // batch header

		for _, p := range g.payments {
			state.PaymentSeq++
			lines, dt := l.BuildDetail(state, p)
			for _, line := range lines {
				batch.Details = append(batch.Details, normalize(line))
			}
			batchTotals = batchTotals.Add(dt)
		}

		batch.Trailer = normalize(l.BuildBatchTrailer(state, g.key, batchTotals))
		batches = append(batches, batch)

		fileTotals.BatchCount++
		fileTotals.RecordCount += batchTotals.RecordCount + 1 // + batch trailer
	}

	return &File{
		Header:  normalize(l.BuildFileHeader(ctx)),
		Batches: batches,
		Trailer: normalize(l.BuildFileTrailer(fileTotals)),
	}
}

// Write serializes f, terminating every record with CRLF.
func Write(f *File) []byte {
	const sep = "\r\n"
	var buf []byte
	buf = append(buf, f.Header+sep...)
	for _, b := range f.Batches {
		buf = append(buf, b.Header+sep...)
		for _, d := range b.Details {
			buf = append(buf, d+sep...)
		}
		buf = append(buf, b.Trailer+sep...)
	}
	buf = append(buf, f.Trailer+sep...)
	return buf
}

type paymentGroup struct {
	key      string
	payments []core.Payment
}

func groupPayments(payments []core.Payment, keyFn func(core.Payment) string) []paymentGroup {
	index := map[string]int{}
	var groups []paymentGroup
	for _, p := range payments {
		key := keyFn(p)
		i, ok := index[key]
		if !ok {
			i = len(groups)
			index[key] = i
			groups = append(groups, paymentGroup{key: key})
		}
		groups[i].payments = append(groups[i].payments, p)
	}
	return groups
}
