package registry

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/cnab-dev/cnab-payment/internal/core"
)

// fakeLayoutGenerator is a test double for LayoutGenerator: payments whose
// ID is in invalid fail Validate with the mapped error; Class is the ID's
// prefix up to '-'; Generate writes the IDs it receives, one per line, and
// records every call so tests can assert on what Generator passed in.
type fakeLayoutGenerator struct {
	invalid     map[string]error
	generateErr error

	calls []generateCall
}

type generateCall struct {
	ids  []string
	opts core.GenerateOptions
}

func (f *fakeLayoutGenerator) Validate(p core.Payment) error {
	return f.invalid[p.Base().ID]
}

func (f *fakeLayoutGenerator) Class(p core.Payment) core.RemittanceClass {
	class, _, _ := strings.Cut(p.Base().ID, "-")
	return core.RemittanceClass(class)
}

func (f *fakeLayoutGenerator) NewPaymentID() (string, error) { return "NEW-ID", nil }

func (f *fakeLayoutGenerator) Generate(payments []core.Payment, opts core.GenerateOptions) ([]byte, error) {
	ids := idsOf(payments)
	f.calls = append(f.calls, generateCall{ids: ids, opts: opts})
	if f.generateErr != nil {
		return nil, f.generateErr
	}
	return []byte(strings.Join(ids, "\n")), nil
}

func transfer(id string) core.Payment {
	return core.TransferPayment{PaymentBase: core.PaymentBase{ID: id}}
}

func idsOf(payments []core.Payment) []string {
	ids := make([]string, len(payments))
	for i, p := range payments {
		ids[i] = p.Base().ID
	}
	return ids
}

func paymentErrorsOf(t *testing.T, err error) []*core.PaymentError {
	t.Helper()
	joined, ok := err.(interface{ Unwrap() []error })
	if !ok {
		t.Fatalf("error %v is not a joined error", err)
	}
	var out []*core.PaymentError
	for _, e := range joined.Unwrap() {
		if inner, ok := e.(interface{ Unwrap() []error }); ok {
			for _, ie := range inner.Unwrap() {
				out = append(out, ie.(*core.PaymentError))
			}
			continue
		}
		out = append(out, e.(*core.PaymentError))
	}
	return out
}

var fixedTime = time.Date(2026, 4, 20, 10, 30, 0, 0, time.UTC)

func TestGenerate_EmptyInput_ReturnsErrNoPayments(t *testing.T) {
	g := NewGenerator(&fakeLayoutGenerator{})

	files, err := g.Generate(nil, core.GenerateOptions{})
	if !errors.Is(err, ErrNoPayments) {
		t.Fatalf("Generate() error = %v, want ErrNoPayments", err)
	}
	if files != nil {
		t.Errorf("Generate() files = %v, want nil", files)
	}
}

func TestGenerate_OneInvalidPayment_ReturnsNoFilesAndNeverGenerates(t *testing.T) {
	layout := &fakeLayoutGenerator{invalid: map[string]error{"pix-2": core.ErrUnsupportedRail}}
	g := NewGenerator(layout)

	files, err := g.Generate([]core.Payment{transfer("pix-1"), transfer("pix-2"), transfer("other-3")}, core.GenerateOptions{})
	if !errors.Is(err, core.ErrUnsupportedRail) {
		t.Fatalf("Generate() error = %v, want ErrUnsupportedRail", err)
	}
	if files != nil {
		t.Errorf("Generate() files = %v, want nil", files)
	}
	if len(layout.calls) != 0 {
		t.Errorf("layout Generate called %d times, want 0", len(layout.calls))
	}
}

func TestGenerate_SeveralInvalidPayments_ReportsEachWithIndexAndID(t *testing.T) {
	layout := &fakeLayoutGenerator{invalid: map[string]error{
		// plain error: Generator wraps it in a PaymentError
		"a-1": core.ErrLayoutMismatch,
		// PaymentError without Index/ID, as a layout would return it
		"a-3": &core.PaymentError{Field: "Amount", Err: core.ErrInvalidField},
		// several field errors for the same payment
		"a-4": errors.Join(
			&core.PaymentError{Field: "Payee.Name", Err: core.ErrInvalidField},
			&core.PaymentError{Field: "Destination", Err: core.ErrUnsupportedRail},
		),
	}}
	g := NewGenerator(layout)

	_, err := g.Generate([]core.Payment{transfer("a-1"), transfer("a-2"), transfer("a-3"), transfer("a-4")}, core.GenerateOptions{})

	got := paymentErrorsOf(t, err)
	want := []core.PaymentError{
		{Index: 0, PaymentID: "a-1", Err: core.ErrLayoutMismatch},
		{Index: 2, PaymentID: "a-3", Field: "Amount", Err: core.ErrInvalidField},
		{Index: 3, PaymentID: "a-4", Field: "Payee.Name", Err: core.ErrInvalidField},
		{Index: 3, PaymentID: "a-4", Field: "Destination", Err: core.ErrUnsupportedRail},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d payment errors, want %d: %v", len(got), len(want), err)
	}
	for i := range want {
		if *got[i] != want[i] {
			t.Errorf("payment error %d = %+v, want %+v", i, *got[i], want[i])
		}
	}
}

func TestGenerate_LayoutPaymentError_IsNotMutated(t *testing.T) {
	layoutErr := &core.PaymentError{Field: "Amount", Err: core.ErrInvalidField}
	g := NewGenerator(&fakeLayoutGenerator{invalid: map[string]error{"a-1": layoutErr}})

	_, _ = g.Generate([]core.Payment{transfer("a-0"), transfer("a-1")}, core.GenerateOptions{})

	if layoutErr.Index != 0 || layoutErr.PaymentID != "" {
		t.Errorf("layout error mutated to %+v", *layoutErr)
	}
}

func TestGenerate_NilPayment_ReportedAsInvalidField(t *testing.T) {
	g := NewGenerator(&fakeLayoutGenerator{})

	_, err := g.Generate([]core.Payment{transfer("a-1"), nil}, core.GenerateOptions{})

	var pe *core.PaymentError
	if !errors.As(err, &pe) || pe.Index != 1 || !errors.Is(pe, core.ErrInvalidField) {
		t.Errorf("Generate() error = %v, want a PaymentError at index 1 wrapping ErrInvalidField", err)
	}
}

func TestGenerate_ValidPayments_PartitionsByClassSortedByIDAndClass(t *testing.T) {
	layout := &fakeLayoutGenerator{}
	g := NewGenerator(layout)

	files, err := g.Generate([]core.Payment{
		transfer("pix-3"), transfer("other-2"), transfer("pix-1"), transfer("other-9"), transfer("other-10"),
	}, core.GenerateOptions{CreatedAt: fixedTime})
	if err != nil {
		t.Fatalf("Generate() error = %v, want nil", err)
	}

	want := []core.RemittanceFile{
		{Class: "other", PaymentIDs: []string{"other-10", "other-2", "other-9"}, Content: []byte("other-10\nother-2\nother-9")},
		{Class: "pix", PaymentIDs: []string{"pix-1", "pix-3"}, Content: []byte("pix-1\npix-3")},
	}
	if !reflect.DeepEqual(files, want) {
		t.Errorf("Generate() files = %+v, want %+v", files, want)
	}
}

func TestGenerate_DoesNotReorderCallersSlice(t *testing.T) {
	g := NewGenerator(&fakeLayoutGenerator{})
	payments := []core.Payment{transfer("a-2"), transfer("a-1")}

	if _, err := g.Generate(payments, core.GenerateOptions{CreatedAt: fixedTime}); err != nil {
		t.Fatalf("Generate() error = %v, want nil", err)
	}
	if got := idsOf(payments); !reflect.DeepEqual(got, []string{"a-2", "a-1"}) {
		t.Errorf("caller's slice reordered to %v", got)
	}
}

func TestGenerate_SameInputInAnyOrder_ProducesSameFiles(t *testing.T) {
	g := NewGenerator(&fakeLayoutGenerator{})
	opts := core.GenerateOptions{CreatedAt: fixedTime}

	first, err := g.Generate([]core.Payment{transfer("pix-2"), transfer("other-1"), transfer("pix-1")}, opts)
	if err != nil {
		t.Fatalf("Generate() error = %v, want nil", err)
	}
	second, err := g.Generate([]core.Payment{transfer("pix-1"), transfer("pix-2"), transfer("other-1")}, opts)
	if err != nil {
		t.Fatalf("Generate() error = %v, want nil", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Errorf("Generate() not stable: %+v vs %+v", first, second)
	}
}

func TestGenerate_CreatedAtSet_PassedThroughUnchanged(t *testing.T) {
	layout := &fakeLayoutGenerator{}
	g := NewGenerator(layout)
	brt := time.FixedZone("-03:00", -3*60*60)
	createdAt := time.Date(2026, 4, 20, 7, 30, 0, 0, brt)

	if _, err := g.Generate([]core.Payment{transfer("a-1")}, core.GenerateOptions{CreatedAt: createdAt}); err != nil {
		t.Fatalf("Generate() error = %v, want nil", err)
	}
	if got := layout.calls[0].opts.CreatedAt; got != createdAt {
		t.Errorf("layout got CreatedAt %v, want %v unchanged", got, createdAt)
	}
}

func TestGenerate_CreatedAtZero_DefaultsToNow(t *testing.T) {
	layout := &fakeLayoutGenerator{}
	g := NewGenerator(layout)

	before := time.Now()
	if _, err := g.Generate([]core.Payment{transfer("a-1"), transfer("b-1")}, core.GenerateOptions{}); err != nil {
		t.Fatalf("Generate() error = %v, want nil", err)
	}
	after := time.Now()

	first := layout.calls[0].opts.CreatedAt
	if first.Before(before) || first.After(after) {
		t.Errorf("layout got CreatedAt %v, want between %v and %v", first, before, after)
	}
	if second := layout.calls[1].opts.CreatedAt; second != first {
		t.Errorf("partitions got different CreatedAt: %v and %v", first, second)
	}
}

func TestGenerate_LayoutGenerateFails_ReturnsItsErrorAndNoFiles(t *testing.T) {
	wantErr := errors.New("boom")
	g := NewGenerator(&fakeLayoutGenerator{generateErr: wantErr})

	files, err := g.Generate([]core.Payment{transfer("a-1")}, core.GenerateOptions{})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Generate() error = %v, want %v", err, wantErr)
	}
	if files != nil {
		t.Errorf("Generate() files = %v, want nil", files)
	}
}

func TestValidate_InvalidPayment_ReturnsPaymentErrorWithID(t *testing.T) {
	g := NewGenerator(&fakeLayoutGenerator{invalid: map[string]error{"a-1": core.ErrLayoutMismatch}})

	err := g.Validate(transfer("a-1"))

	var pe *core.PaymentError
	if !errors.As(err, &pe) || pe.PaymentID != "a-1" || !errors.Is(err, core.ErrLayoutMismatch) {
		t.Errorf("Validate() error = %v, want a PaymentError for a-1 wrapping ErrLayoutMismatch", err)
	}
}

func TestValidate_ValidPayment_ReturnsNil(t *testing.T) {
	g := NewGenerator(&fakeLayoutGenerator{})

	if err := g.Validate(transfer("a-1")); err != nil {
		t.Errorf("Validate() error = %v, want nil", err)
	}
}

func TestNewPaymentID_DelegatesToLayout(t *testing.T) {
	g := NewGenerator(&fakeLayoutGenerator{})

	id, err := g.NewPaymentID()
	if err != nil || id != "NEW-ID" {
		t.Errorf("NewPaymentID() = %q, %v, want %q, nil", id, err, "NEW-ID")
	}
}
