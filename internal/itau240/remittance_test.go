package itau240

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/cnab-dev/cnab-payment/internal/core"
)

// pos returns the 1-indexed, inclusive [from, to] slice of a record, as in
// the manual.
func pos(record string, from, to int) string { return record[from-1 : to] }

// qrURL is the location of the synthetic dynamic QR code below. Its path
// has lowercase letters, which must survive normalization (nota 41).
const qrURL = "pix.example.com/api/v2/cobv/0a1b2c3d4e5f60718293a4b5c6d7e8f9"

// qrPayload is a synthetic dynamic Pix QR code: location URL, no amount,
// txid "***".
var qrPayload = func() string {
	tlv := func(id, v string) string { return fmt.Sprintf("%s%02d%s", id, len(v), v) }
	body := tlv("00", "01") + tlv("01", "12") +
		tlv("26", tlv("00", "br.gov.bcb.pix")+tlv("25", qrURL)) +
		tlv("52", "0000") + tlv("53", "986") + tlv("58", "BR") +
		tlv("59", "EXEMPLO RECEBEDOR SA") + tlv("60", "Brasilia") +
		tlv("62", tlv("05", "***")) + "6304"
	return body + crc16(body)
}()

// crc16 is CRC16-CCITT (poly 0x1021, init 0xFFFF), as the BR Code uses.
func crc16(s string) string {
	crc := uint16(0xFFFF)
	for i := 0; i < len(s); i++ {
		crc ^= uint16(s[i]) << 8
		for range 8 {
			if crc&0x8000 != 0 {
				crc = crc<<1 ^ 0x1021
			} else {
				crc <<= 1
			}
		}
	}
	return fmt.Sprintf("%04X", crc)
}

func qrCodePayment() core.QRCodePayment {
	return core.QRCodePayment{
		PaymentBase: core.PaymentBase{
			ID:    "AB12CD34EF56GH78IJ90",
			Payer: core.Party{Name: "Empresa Pagadora Ltda", TaxID: "11222333000181"},
			PayerAccount: core.BankAccount{
				BankCode: "341", Branch: "0610", Number: "20445", CheckDigit: "5", Type: core.AccountTypeChecking,
			},
			Payee:          core.Party{Name: "EXEMPLO RECEBEDOR SA", TaxID: "00000090000110"},
			Amount:         340513,
			DueDate:        time.Date(2026, 9, 18, 0, 0, 0, 0, time.UTC),
			SettlementDate: time.Date(2026, 9, 17, 0, 0, 0, 0, time.UTC),
		},
		Payload: qrPayload,
	}
}

func generate(t *testing.T, payments ...core.Payment) []string {
	t.Helper()
	for _, p := range payments {
		if err := validatePayment(p); err != nil {
			t.Fatalf("validatePayment: %v", err)
		}
	}
	out, err := NewGenerator().Generate(payments, core.GenerateOptions{
		CreatedAt: time.Date(2026, 9, 16, 13, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	records := strings.Split(strings.TrimSuffix(string(out), "\r\n"), "\r\n")
	for i, r := range records {
		if len(r) != recordLength {
			t.Fatalf("record %d has %d characters, want %d", i+1, len(r), recordLength)
		}
	}
	return records
}

func TestGenerate_PixQRCode(t *testing.T) {
	p := qrCodePayment()
	records := generate(t, p)
	if len(records) != 6 { // file header, batch header, J, J-52, batch trailer, file trailer
		t.Fatalf("got %d records, want 6", len(records))
	}
	batch, j, j52 := records[1], records[2], records[3]

	checks := []struct {
		name, record string
		from, to     int
		want         string
	}{
		{"batch tipo/forma", batch, 10, 13, "2047"},
		{"J segment", j, 14, 14, "J"},
		{"J barcode zeroed (nota 18)", j, 18, 61, strings.Repeat("0", 44)},
		{"J beneficiary", j, 62, 91, "EXEMPLO RECEBEDOR SA          "},
		{"J due date", j, 92, 99, "18092026"},
		{"J amount", j, 100, 114, "000000000340513"},
		{"J payment date", j, 145, 152, "17092026"},
		{"J payment amount", j, 153, 167, "000000000340513"},
		{"J seu número", j, 183, 202, "AB12CD34EF56GH78IJ90"},
		{"J-52 segment", j52, 14, 14, "J"},
		{"J-52 code", j52, 18, 19, "52"},
		{"J-52 sacado type", j52, 20, 20, "2"},
		{"J-52 sacado doc", j52, 21, 35, "011222333000181"},
		{"J-52 sacado name, upper-cased", j52, 36, 75, "EMPRESA PAGADORA LTDA" + strings.Repeat(" ", 19)},
		{"J-52 cedente type", j52, 76, 76, "2"},
		{"J-52 cedente doc", j52, 77, 91, "000000090000110"},
		{"J-52 cedente name", j52, 92, 131, "EXEMPLO RECEBEDOR SA" + strings.Repeat(" ", 20)},
		{"J-52 PIX URL keeps its case (nota 41)", j52, 132, 208, qrURL + strings.Repeat(" ", 77-len(qrURL))},
		{"J-52 PIX txid empty (***)", j52, 209, 240, strings.Repeat(" ", 32)},
	}
	for _, c := range checks {
		if got := pos(c.record, c.from, c.to); got != c.want {
			t.Errorf("%s [%03d-%03d]: got %q, want %q", c.name, c.from, c.to, got, c.want)
		}
	}
}

func TestGenerate_SacadoIsDebtorWhenSet(t *testing.T) {
	p := qrCodePayment()
	p.Debtor = &core.Party{Name: "Cliente Devedor", TaxID: "52998224725"}
	records := generate(t, p)
	if got := pos(records[3], 20, 75); got != "1000052998224725CLIENTE DEVEDOR"+strings.Repeat(" ", 25) {
		t.Errorf("sacado = %q", got)
	}
}

func TestGenerate_TxID(t *testing.T) {
	p := qrCodePayment()

	p.TxID = "PEDIDO123"
	if got := pos(generate(t, p)[3], 209, 240); got != "PEDIDO123"+strings.Repeat(" ", 23) {
		t.Errorf("txid = %q", got)
	}

	p.TxID = strings.Repeat("T", 33)
	if got := pos(generate(t, p)[3], 209, 240); got != strings.Repeat(" ", 32) {
		t.Errorf("txid > 32 should be blank, got %q", got)
	}
}

func TestGenerate_AccentedNamesKeepRecordWidth(t *testing.T) {
	p := qrCodePayment()
	p.Payee.Name = "João da Conceição"
	p.Debtor = &core.Party{Name: "Ação Nº 1 Ltda", TaxID: "52998224725"}
	records := generate(t, p)
	if got := pos(records[2], 62, 91); got != "JOAO DA CONCEICAO"+strings.Repeat(" ", 13) {
		t.Errorf("J beneficiary = %q", got)
	}
	if got := pos(records[3], 36, 75); got != "ACAO N  1 LTDA"+strings.Repeat(" ", 26) {
		t.Errorf("J-52 sacado = %q", got)
	}
}

func TestGenerate_HeaderCreatedAtInBrasiliaTime(t *testing.T) {
	// 13:00 UTC is 10:00 in Brasília.
	records := generate(t, qrCodePayment())
	if got := pos(records[0], 143, 157); got != "1"+"16092026"+"100000" {
		t.Errorf("file kind and creation date/time = %q", got)
	}
}

func TestValidate_ReportsEveryProblem(t *testing.T) {
	p := qrCodePayment()
	p.PayerAccount.BankCode = "237"
	p.PayerAccount.CheckDigit = ""
	p.Payee.TaxID = ""
	p.DueDate = time.Time{}
	p.Amount = 0
	p.Payload = qrPayload[:len(qrPayload)-4] + "0000"

	err := NewGenerator().Validate(p)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, core.ErrLayoutMismatch) || !errors.Is(err, core.ErrInvalidField) {
		t.Errorf("error should wrap ErrLayoutMismatch and ErrInvalidField: %v", err)
	}
	for _, field := range []string{"PayerAccount.BankCode", "PayerAccount.CheckDigit", "Payee.TaxID", "DueDate", "Amount", "Payload"} {
		if !hasField(err, field) {
			t.Errorf("no PaymentError for %s in %v", field, err)
		}
	}
}

func TestValidate_QRCodeAmountMustMatch(t *testing.T) {
	tlv := func(id, v string) string { return fmt.Sprintf("%s%02d%s", id, len(v), v) }
	body := tlv("00", "01") + tlv("26", tlv("00", "br.gov.bcb.pix")+tlv("01", "fulano@example.com")) +
		tlv("54", "10.50") + "6304"

	p := qrCodePayment()
	p.Payload = body + crc16(body)
	if err := validatePayment(p); !hasField(err, "Amount") {
		t.Errorf("expected an Amount error, got %v", err)
	}
	p.Amount = 1050
	if err := validatePayment(p); err != nil {
		t.Errorf("matching amount: %v", err)
	}
}

func TestValidate_Barcode(t *testing.T) {
	base := qrCodePayment().PaymentBase
	cases := map[string]string{
		"too short":                 "123",
		"not digits":                strings.Repeat("A", 44),
		"collection, not DARF":      "8" + strings.Repeat("1", 43),
		"collection, wrong agrmt":   "85" + strings.Repeat("0", 42),
		"boleto with a letter":      "34191" + strings.Repeat("0", 38) + "X",
		"collection, 43 digits":     "85" + strings.Repeat("0", 13) + "0385" + strings.Repeat("0", 24),
		"collection, with a letter": "85" + strings.Repeat("0", 13) + "0385" + strings.Repeat("0", 24) + "X",
	}
	for name, barcode := range cases {
		err := validatePayment(core.BarcodePayment{PaymentBase: base, Barcode: barcode})
		if !hasField(err, "Barcode") {
			t.Errorf("%s: expected a Barcode error, got %v", name, err)
		}
	}

	darf := "85" + strings.Repeat("0", 13) + "0385" + strings.Repeat("0", 25)
	if err := validatePayment(core.BarcodePayment{PaymentBase: base, Barcode: darf}); err != nil {
		t.Errorf("DARF barcode: %v", err)
	}
}

func TestValidate_TransfersNotSupportedYet(t *testing.T) {
	p := core.TransferPayment{PaymentBase: qrCodePayment().PaymentBase, Rail: core.RailTEF}
	if err := validatePayment(p); !errors.Is(err, core.ErrUnsupportedRail) {
		t.Errorf("expected ErrUnsupportedRail, got %v", err)
	}
}

func TestClass(t *testing.T) {
	base := qrCodePayment().PaymentBase
	cases := []struct {
		p    core.Payment
		want core.RemittanceClass
	}{
		{qrCodePayment(), "pix"},
		{core.TransferPayment{PaymentBase: base, Rail: core.RailPix}, "pix"},
		{core.TransferPayment{PaymentBase: base, Rail: core.RailTED}, "other"},
		{core.BarcodePayment{PaymentBase: base}, "other"},
	}
	for _, c := range cases {
		if got := NewGenerator().Class(c.p); got != c.want {
			t.Errorf("Class(%T) = %q, want %q", c.p, got, c.want)
		}
	}
}

func TestNewPaymentID(t *testing.T) {
	re := regexp.MustCompile(`^[A-Z0-9]{20}$`)
	seen := map[string]bool{}
	for range 1000 {
		id, err := NewGenerator().NewPaymentID()
		if err != nil {
			t.Fatal(err)
		}
		if !re.MatchString(id) {
			t.Fatalf("id %q is not 20 uppercase alphanumeric characters", id)
		}
		if seen[id] {
			t.Fatalf("duplicate id %q", id)
		}
		seen[id] = true
	}
}

func TestCollectionLine(t *testing.T) {
	// Each 11-digit block gets its modulo 11 check digit appended.
	barcode := "85890000001" + "23450385000" + "00000000000" + "12345678901"
	line := collectionLine(barcode)
	if len(line) != 48 {
		t.Fatalf("line has %d digits, want 48", len(line))
	}
	for i := 0; i < 4; i++ {
		block := line[i*12 : i*12+11]
		if block != barcode[i*11:i*11+11] {
			t.Errorf("block %d = %q, want %q", i+1, block, barcode[i*11:i*11+11])
		}
		if dv := line[i*12+11 : i*12+12]; dv != collectionCheckDigit(block) {
			t.Errorf("block %d check digit = %q", i+1, dv)
		}
	}
	if got := collectionCheckDigit("00000000000"); got != "0" {
		t.Errorf("check digit of zeros = %q, want 0", got)
	}
}

func hasField(err error, field string) bool {
	var joined interface{ Unwrap() []error }
	if !errors.As(err, &joined) {
		var pe *core.PaymentError
		return errors.As(err, &pe) && pe.Field == field
	}
	for _, e := range joined.Unwrap() {
		var pe *core.PaymentError
		if errors.As(e, &pe) && pe.Field == field {
			return true
		}
	}
	return false
}
