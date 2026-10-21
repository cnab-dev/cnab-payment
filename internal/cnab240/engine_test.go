package cnab240

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/cnab-dev/cnab-payment/internal/core"
)

// fakeLayout writes short, readable records so tests can assert on
// structure: batches are keyed by payer account number, and each payment
// produces segmentsPerPayment detail records.
type fakeLayout struct {
	segmentsPerPayment int
}

func (fakeLayout) GroupKey(p core.Payment) string { return p.Base().PayerAccount.Number }

func (fakeLayout) BuildFileHeader(ctx FileContext) string {
	return fmt.Sprintf("FH %s %s %s", ctx.Payer.Name, ctx.PayerAccount.Number, FmtDate(ctx.CreatedAt))
}

func (fakeLayout) BuildFileTrailer(t FileTotals) string {
	return fmt.Sprintf("FT batches=%d records=%d", t.BatchCount, t.RecordCount)
}

func (fakeLayout) BuildBatchHeader(s *BuildState, key string) string {
	return fmt.Sprintf("BH %d %s %s", s.BatchSequence, key, s.Payer.Name)
}

func (fakeLayout) BuildBatchTrailer(s *BuildState, key string, t BatchTotals) string {
	return fmt.Sprintf("BT %d records=%d payments=%d total=%d", s.BatchSequence, t.RecordCount, t.PaymentCount, t.TotalAmount)
}

func (l fakeLayout) BuildDetail(s *BuildState, p core.Payment) ([]string, BatchTotals) {
	n := l.segmentsPerPayment
	if n == 0 {
		n = 1
	}
	var lines []string
	for i := 0; i < n; i++ {
		lines = append(lines, fmt.Sprintf("D %d.%d %s", s.BatchSequence, s.PaymentSeq, p.Base().ID))
	}
	return lines, BatchTotals{RecordCount: n, PaymentCount: 1, TotalAmount: p.Base().Amount}
}

type lowercaseKeepingLayout struct{ fakeLayout }

func (lowercaseKeepingLayout) NormalizeRecord(r string) string { return r }

func payment(id, payerName, account string, amount core.Amount) core.Payment {
	return core.BarcodePayment{PaymentBase: core.PaymentBase{
		ID:           id,
		Payer:        core.Party{Name: payerName},
		PayerAccount: core.BankAccount{Number: account},
		Amount:       amount,
	}}
}

func lines(f *File) []string {
	out := []string{f.Header}
	for _, b := range f.Batches {
		out = append(out, b.Header)
		out = append(out, b.Details...)
		out = append(out, b.Trailer)
	}
	return append(out, f.Trailer)
}

func TestBuild_GroupsByKeyInFirstAppearanceOrder(t *testing.T) {
	payments := []core.Payment{
		payment("A1", "Payer X", "111", 100),
		payment("B2", "Payer Y", "222", 200),
		payment("C3", "Payer X", "111", 300),
	}

	got := lines(Build(fakeLayout{}, payments, time.Date(2026, 4, 20, 10, 0, 0, 0, time.UTC)))
	want := []string{
		"FH PAYER X 111 20042026",
		"BH 1 111 PAYER X",
		"D 1.1 A1",
		"D 1.2 C3",
		"BT 1 RECORDS=3 PAYMENTS=2 TOTAL=400",
		"BH 2 222 PAYER Y",
		"D 2.1 B2",
		"BT 2 RECORDS=2 PAYMENTS=1 TOTAL=200",
		"FT BATCHES=2 RECORDS=7",
	}
	assertLines(t, got, want)
}

func TestBuild_CountsEveryDetailRecord(t *testing.T) {
	payments := []core.Payment{
		payment("A1", "Payer", "111", 100),
		payment("A2", "Payer", "111", 100),
	}

	got := lines(Build(fakeLayout{segmentsPerPayment: 2}, payments, time.Time{}))
	if trailer := got[len(got)-2]; trailer != "BT 1 RECORDS=5 PAYMENTS=2 TOTAL=200" {
		t.Errorf("batch trailer = %q", trailer)
	}
	if trailer := got[len(got)-1]; trailer != "FT BATCHES=1 RECORDS=6" {
		t.Errorf("file trailer = %q", trailer)
	}
}

func TestBuild_NormalizesEveryRecordByDefault(t *testing.T) {
	f := Build(fakeLayout{}, []core.Payment{payment("a1", "João", "111", 1)}, time.Time{})
	for _, l := range lines(f) {
		if l != strings.ToUpper(l) || strings.Contains(l, "Ã") {
			t.Errorf("record not normalized: %q", l)
		}
	}
}

func TestBuild_UsesLayoutNormalizerWhenProvided(t *testing.T) {
	f := Build(lowercaseKeepingLayout{}, []core.Payment{payment("a1", "João", "111", 1)}, time.Time{})
	if f.Batches[0].Details[0] != "D 1.1 a1" {
		t.Errorf("detail = %q, want the layout's normalizer to keep it as is", f.Batches[0].Details[0])
	}
}

func TestWrite_TerminatesEveryRecordWithCRLF(t *testing.T) {
	f := &File{
		Header:  "H",
		Batches: []Batch{{Header: "BH", Details: []string{"D1", "D2"}, Trailer: "BT"}},
		Trailer: "T",
	}
	if got, want := string(Write(f)), "H\r\nBH\r\nD1\r\nD2\r\nBT\r\nT\r\n"; got != want {
		t.Errorf("Write = %q, want %q", got, want)
	}
}

func assertLines(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %d records, want %d:\n%s", len(got), len(want), strings.Join(got, "\n"))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("record %d = %q, want %q", i, got[i], want[i])
		}
	}
}
