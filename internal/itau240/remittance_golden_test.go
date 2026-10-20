package itau240

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cnab-dev/cnab-payment/internal/core"
	"github.com/cnab-dev/cnab-payment/internal/registry"
)

// The golden files in testdata/remittance were produced by a reference
// implementation of this layout; scenarios.json holds their inputs in this
// package's model. See testdata/remittance/README.md.

type fixtures struct {
	CreatedAt time.Time `json:"createdAt"`
	Scenarios []struct {
		File     string           `json:"file"`
		Payments []fixturePayment `json:"payments"`
	} `json:"scenarios"`
}

type fixturePayment struct {
	ID             string              `json:"id"`
	Kind           string              `json:"kind"`
	Payer          fixtureParty        `json:"payer"`
	PayerAccount   fixtureBankAccount  `json:"payerAccount"`
	Payee          fixtureParty        `json:"payee"`
	Amount         string              `json:"amount"`
	DueDate        string              `json:"dueDate"`
	SettlementDate string              `json:"settlementDate"`
	Barcode        string              `json:"barcode"`
	Rail           string              `json:"rail"`
	BankAccount    *fixtureBankAccount `json:"bankAccount"`
	PixKey         *struct {
		Type  string `json:"type"`
		Value string `json:"value"`
	} `json:"pixKey"`
}

type fixtureParty struct {
	Name    string        `json:"name"`
	TaxID   string        `json:"taxId"`
	Address *core.Address `json:"address"`
}

type fixtureBankAccount struct {
	BankCode   string `json:"bankCode"`
	Branch     string `json:"branch"`
	Number     string `json:"number"`
	CheckDigit string `json:"checkDigit"`
	Type       string `json:"type"`
}

func (a fixtureBankAccount) toCore() core.BankAccount {
	return core.BankAccount{
		BankCode:   a.BankCode,
		Branch:     a.Branch,
		Number:     a.Number,
		CheckDigit: a.CheckDigit,
		Type:       core.AccountType(a.Type),
	}
}

func (p fixtureParty) toCore() core.Party {
	return core.Party{Name: p.Name, TaxID: p.TaxID, Address: p.Address}
}

func (f fixturePayment) toCore(t *testing.T) core.Payment {
	t.Helper()
	amount, err := core.ParseAmount(f.Amount)
	if err != nil {
		t.Fatalf("%s: %v", f.ID, err)
	}
	base := core.PaymentBase{
		ID:             f.ID,
		Payer:          f.Payer.toCore(),
		PayerAccount:   f.PayerAccount.toCore(),
		Payee:          f.Payee.toCore(),
		Amount:         amount,
		DueDate:        date(t, f.DueDate),
		SettlementDate: date(t, f.SettlementDate),
	}
	switch f.Kind {
	case "barcode":
		return core.BarcodePayment{PaymentBase: base, Barcode: f.Barcode}
	case "transfer":
		tp := core.TransferPayment{PaymentBase: base, Rail: core.Rail(f.Rail)}
		switch {
		case f.BankAccount != nil:
			tp.Destination = f.BankAccount.toCore()
		case f.PixKey != nil:
			tp.Destination = core.PixKey{Type: core.PixKeyType(f.PixKey.Type), Value: f.PixKey.Value}
		}
		return tp
	}
	t.Fatalf("%s: unknown kind %q", f.ID, f.Kind)
	return nil
}

func date(t *testing.T, s string) time.Time {
	t.Helper()
	d, err := time.Parse("2006-01-02", s)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func loadFixtures(t *testing.T) fixtures {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "remittance", "scenarios.json"))
	if err != nil {
		t.Fatal(err)
	}
	var f fixtures
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatal(err)
	}
	return f
}

func TestGenerate_Golden(t *testing.T) {
	f := loadFixtures(t)
	g := registry.NewGenerator(NewGenerator())

	for _, sc := range f.Scenarios {
		t.Run(sc.File, func(t *testing.T) {
			var payments []core.Payment
			for _, fp := range sc.Payments {
				payments = append(payments, fp.toCore(t))
			}

			files, err := g.Generate(payments, core.GenerateOptions{CreatedAt: f.CreatedAt})
			if err != nil {
				t.Fatal(err)
			}
			if len(files) != 1 {
				t.Fatalf("got %d files, want 1", len(files))
			}
			if len(files[0].PaymentIDs) != len(payments) {
				t.Errorf("file lists %d payment IDs, want %d", len(files[0].PaymentIDs), len(payments))
			}

			want, err := os.ReadFile(filepath.Join("testdata", "remittance", sc.File))
			if err != nil {
				t.Fatal(err)
			}
			assertSameRecords(t, files[0].Content, want)
		})
	}
}

// assertSameRecords compares two remittances byte for byte, reporting the
// first differing records.
func assertSameRecords(t *testing.T, got, want []byte) {
	t.Helper()
	if bytes.Equal(got, want) {
		return
	}
	g, w := bytes.Split(got, []byte("\r\n")), bytes.Split(want, []byte("\r\n"))
	if len(g) != len(w) {
		t.Errorf("got %d records, want %d", len(g)-1, len(w)-1)
	}
	reported := 0
	for i := 0; i < len(g) && i < len(w) && reported < 5; i++ {
		if !bytes.Equal(g[i], w[i]) {
			t.Errorf("record %d:\n got %q\nwant %q", i+1, g[i], w[i])
			reported++
		}
	}
	if reported == 0 {
		t.Error("contents differ")
	}
}
