package itau240

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/cnab-dev/cnab-payment/internal/core"
	"github.com/cnab-dev/cnab-payment/internal/registry"
)

func account(bank, branch, number, digit string, typ core.AccountType) core.BankAccount {
	return core.BankAccount{BankCode: bank, Branch: branch, Number: number, CheckDigit: digit, Type: typ}
}

func transfer(rail core.Rail, dest core.Destination) core.TransferPayment {
	return core.TransferPayment{
		PaymentBase: core.PaymentBase{
			ID:             "TR0001",
			Payer:          core.Party{Name: "EMPRESA PAGADORA LTDA", TaxID: "11222333000181"},
			PayerAccount:   account("341", "0610", "20445", "5", core.AccountTypeChecking),
			Payee:          core.Party{Name: "ANA EXEMPLO DE SOUZA", TaxID: "52998224725"},
			Amount:         12345,
			DueDate:        time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC),
			SettlementDate: time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC),
		},
		Rail:        rail,
		Destination: dest,
	}
}

func TestValidate_RailDestinationMatrix(t *testing.T) {
	itauChecking := account("341", "0567", "01234", "9", core.AccountTypeChecking)
	cases := []struct {
		name    string
		rail    core.Rail
		dest    core.Destination
		wantErr error
	}{
		{"tef checking", core.RailTEF, itauChecking, nil},
		{"tef savings", core.RailTEF, account("341", "0567", "01234", "9", core.AccountTypeSavings), nil},
		{"tef salary", core.RailTEF, account("341", "0567", "01234", "9", core.AccountTypeSalary), nil},
		{"tef payment account", core.RailTEF, account("341", "0567", "01234", "9", core.AccountTypePayment), core.ErrUnsupportedRail},
		{"tef other bank", core.RailTEF, account("237", "0321", "4567", "3", core.AccountTypeChecking), core.ErrUnsupportedRail},
		{"tef pix key", core.RailTEF, core.PixKey{Type: core.PixKeyTypeEmail, Value: "a@example.com"}, core.ErrUnsupportedRail},

		{"ted other bank checking", core.RailTED, account("237", "0321", "4567", "3", core.AccountTypeChecking), nil},
		{"ted other bank payment account", core.RailTED, account("260", "0001", "12345678", "9", core.AccountTypePayment), nil},
		{"ted same bank", core.RailTED, itauChecking, core.ErrUnsupportedRail},

		{"pix key", core.RailPix, core.PixKey{Type: core.PixKeyTypeTaxID, Value: "52998224725"}, nil},
		{"pix checking", core.RailPix, account("260", "0001", "12345678", "9", core.AccountTypeChecking), nil},
		{"pix savings", core.RailPix, account("104", "1234", "000012345678", "0", core.AccountTypeSavings), nil},
		{"pix payment account", core.RailPix, account("323", "0001", "123456789", "0", core.AccountTypePayment), nil},
		{"pix Itaú checking", core.RailPix, itauChecking, nil},
		{"pix salary", core.RailPix, account("341", "0567", "01234", "9", core.AccountTypeSalary), core.ErrUnsupportedRail},

		{"doc", core.Rail("doc"), account("237", "0321", "4567", "3", core.AccountTypeChecking), core.ErrUnsupportedRail},
		{"no rail", "", itauChecking, core.ErrUnsupportedRail},
		{"no destination", core.RailTEF, nil, core.ErrInvalidField},
	}
	for _, c := range cases {
		err := validatePayment(transfer(c.rail, c.dest))
		switch {
		case c.wantErr == nil && err != nil:
			t.Errorf("%s: unexpected error %v", c.name, err)
		case c.wantErr != nil && !errors.Is(err, c.wantErr):
			t.Errorf("%s: error = %v, want %v", c.name, err, c.wantErr)
		}
	}
}

func TestValidate_DestinationAccountWidths(t *testing.T) {
	cases := []struct {
		name  string
		rail  core.Rail
		dest  core.BankAccount
		field string
	}{
		{"Itaú branch over 4 digits", core.RailTEF, account("341", "12345", "01234", "9", core.AccountTypeChecking), "Destination.Branch"},
		{"Itaú account over 6 digits", core.RailTEF, account("341", "0567", "1234567", "9", core.AccountTypeChecking), "Destination.Number"},
		{"Itaú two-character DAC", core.RailTEF, account("341", "0567", "01234", "9X", core.AccountTypeChecking), "Destination.CheckDigit"},
		{"other bank account over 12 digits", core.RailTED, account("237", "0321", "1234567890123", "3", core.AccountTypeChecking), "Destination.Number"},
		{"missing DAC", core.RailTED, account("237", "0321", "4567", "", core.AccountTypeChecking), "Destination.CheckDigit"},
		{"bank code not 3 digits", core.RailTED, account("23", "0321", "4567", "3", core.AccountTypeChecking), "Destination.BankCode"},
		{"unknown account type", core.RailTED, account("237", "0321", "4567", "3", "loan"), "Destination.Type"},
	}
	for _, c := range cases {
		if err := validatePayment(transfer(c.rail, c.dest)); !hasField(err, c.field) {
			t.Errorf("%s: expected a %s error, got %v", c.name, c.field, err)
		}
	}

	// Leading zeros don't count towards the width.
	if err := validatePayment(transfer(core.RailTEF, account("341", "0567", "0001234", "9", core.AccountTypeChecking))); err != nil {
		t.Errorf("leading zeros: %v", err)
	}
}

func TestValidate_PixKey(t *testing.T) {
	cases := map[string]core.PixKey{
		"unknown type":   {Type: "cpf", Value: "52998224725"},
		"empty value":    {Type: core.PixKeyTypeEmail},
		"too long":       {Type: core.PixKeyTypeEmail, Value: strings.Repeat("a", 101)},
		"invalid tax ID": {Type: core.PixKeyTypeTaxID, Value: "52998224724"},
	}
	for name, key := range cases {
		err := validatePayment(transfer(core.RailPix, key))
		if !errors.Is(err, core.ErrInvalidField) {
			t.Errorf("%s: expected ErrInvalidField, got %v", name, err)
		}
	}
}

func TestGenerate_SegmentA(t *testing.T) {
	cases := []struct {
		name            string
		rail            core.Rail
		dest            core.Destination
		batch           string // tipo/forma, positions 10-13 of the batch header
		chamber, bank   string // 18-20, 21-23
		account         string // 24-43
		transferType    string // 113-114
		wantSegmentB    bool
		wantSegmentBKey string // B 15-16
	}{
		{
			name: "tef checking", rail: core.RailTEF, dest: account("341", "0567", "01234", "9", core.AccountTypeChecking),
			batch: "2001", chamber: "000", bank: "341", account: "00567 000000001234 9", transferType: "00",
		},
		{
			name: "tef salary", rail: core.RailTEF, dest: account("341", "0567", "01234", "9", core.AccountTypeSalary),
			batch: "3001", chamber: "000", bank: "341", account: "00567 000000001234 9", transferType: "00",
		},
		{
			name: "ted with a 12-digit account and two-character DAC (nota 11b)", rail: core.RailTED,
			dest:  account("104", "1234", "123456789012", "0X", core.AccountTypeChecking),
			batch: "2041", chamber: "000", bank: "104", account: "01234 1234567890120X", transferType: "00",
		},
		{
			name: "ted to a payment account goes as a checking account", rail: core.RailTED,
			dest:  account("260", "0001", "12345678", "9", core.AccountTypePayment),
			batch: "2041", chamber: "000", bank: "260", account: "00001 000012345678 9", transferType: "00",
		},
		{
			name: "pix to a checking account at another bank", rail: core.RailPix,
			dest:  account("260", "0001", "12345678", "9", core.AccountTypeChecking),
			batch: "2045", chamber: "009", bank: "260", account: "00001 000012345678 9", transferType: "01",
		},
		{
			name: "pix to a savings account", rail: core.RailPix,
			dest:  account("104", "1234", "000012345678", "0", core.AccountTypeSavings),
			batch: "2045", chamber: "009", bank: "104", account: "01234 000012345678 0", transferType: "03",
		},
		{
			name: "pix to a payment account", rail: core.RailPix,
			dest:  account("323", "0001", "123456789", "0", core.AccountTypePayment),
			batch: "2045", chamber: "009", bank: "323", account: "00000000001234567890", transferType: "PG",
		},
		{
			name: "pix to an Itaú checking account", rail: core.RailPix,
			dest:  account("341", "0567", "01234", "9", core.AccountTypeChecking),
			batch: "2045", chamber: "009", bank: "341", account: "00567 000000001234 9", transferType: "01",
		},
		{
			name: "pix key", rail: core.RailPix,
			dest:  core.PixKey{Type: core.PixKeyTypeRandom, Value: "123e4567-e12b-12d1-a456-426614174000"},
			batch: "2045", chamber: "009", bank: "000", account: strings.Repeat(" ", 20), transferType: "04",
			wantSegmentB: true, wantSegmentBKey: "04",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			records := generate(t, transfer(c.rail, c.dest))
			wantRecords := 5 // file header, batch header, A, batch trailer, file trailer
			if c.wantSegmentB {
				wantRecords = 6
			}
			if len(records) != wantRecords {
				t.Fatalf("got %d records, want %d", len(records), wantRecords)
			}
			a := records[2]
			checks := []struct {
				name   string
				record string
				from   int
				to     int
				want   string
			}{
				{"batch tipo/forma", records[1], 10, 13, c.batch},
				{"segment", a, 14, 14, "A"},
				{"chamber", a, 18, 20, c.chamber},
				{"bank", a, 21, 23, c.bank},
				{"account (nota 11)", a, 24, 43, c.account},
				{"payee name", a, 44, 73, "ANA EXEMPLO DE SOUZA          "},
				{"seu número", a, 74, 93, "TR0001              "},
				{"payment date", a, 94, 101, "20042026"},
				{"currency", a, 102, 104, "REA"},
				{"transfer type (nota 36)", a, 113, 114, c.transferType},
				{"amount", a, 120, 134, "000000000012345"},
				{"payee CPF, left-aligned", a, 204, 217, "52998224725   "},
			}
			if c.wantSegmentB {
				b := records[3]
				checks = append(checks,
					struct {
						name   string
						record string
						from   int
						to     int
						want   string
					}{"B key type (nota 37)", b, 15, 16, c.wantSegmentBKey},
					struct {
						name   string
						record string
						from   int
						to     int
						want   string
					}{
						// Upper-cased like the rest of the record, as the reference
						// implementation does; to be confirmed with the bank for
						// random keys.
						"B key", b, 128, 227,
						strings.ToUpper(c.dest.(core.PixKey).Value) + strings.Repeat(" ", 100-len(c.dest.(core.PixKey).Value)),
					},
				)
			}
			for _, ck := range checks {
				if got := pos(ck.record, ck.from, ck.to); got != ck.want {
					t.Errorf("%s [%03d-%03d]: got %q, want %q", ck.name, ck.from, ck.to, got, ck.want)
				}
			}
		})
	}
}

func TestGenerate_PixAndOtherPaymentsGoToSeparateFiles(t *testing.T) {
	pixKey := transfer(core.RailPix, core.PixKey{Type: core.PixKeyTypeEmail, Value: "ana@example.com"})
	pixKey.ID = "B2"
	tef := transfer(core.RailTEF, account("341", "0567", "01234", "9", core.AccountTypeChecking))
	tef.ID = "A1"
	qr := qrCodePayment()
	qr.ID = "C3"

	files, err := registry.NewGenerator(NewGenerator()).Generate(
		[]core.Payment{qr, pixKey, tef}, core.GenerateOptions{CreatedAt: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Fatalf("got %d files, want 2", len(files))
	}
	if files[0].Class != "other" || strings.Join(files[0].PaymentIDs, ",") != "A1" {
		t.Errorf("files[0] = %s %v, want other [A1]", files[0].Class, files[0].PaymentIDs)
	}
	if files[1].Class != "pix" || strings.Join(files[1].PaymentIDs, ",") != "B2,C3" {
		t.Errorf("files[1] = %s %v, want pix [B2 C3]", files[1].Class, files[1].PaymentIDs)
	}
}
