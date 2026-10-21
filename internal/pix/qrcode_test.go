package pix

import (
	"fmt"
	"strings"
	"testing"

	"github.com/cnab-dev/cnab-payment/internal/core"
)

// tlv encodes one EMV field.
func tlv(id, value string) string { return fmt.Sprintf("%s%02d%s", id, len([]rune(value)), value) }

// withCRC appends tag 63 with the payload's CRC16.
func withCRC(body string) string {
	body += "6304"
	return body + fmt.Sprintf("%04X", crc16CCITT([]byte(body)))
}

// dynamicPayload is a synthetic dynamic code: location URL, no amount,
// txid "***".
var dynamicPayload = withCRC(
	tlv("00", "01") + tlv("01", "12") +
		tlv("26", tlv("00", "br.gov.bcb.pix")+tlv("25", "pix.example.com/api/v2/cobv/0a1b2c3d4e5f60718293a4b5c6d7e8f9")) +
		tlv("52", "0000") + tlv("53", "986") + tlv("58", "BR") +
		tlv("59", "EXEMPLO RECEBEDOR SA") + tlv("60", "Brasilia") +
		tlv("62", tlv("05", "***")),
)

// staticPayload is a static code with a random key, amount and txid.
const staticPayload = "00020126580014br.gov.bcb.pix0136123e4567-e12b-12d1-a456-426614174000520400005303986540510.50" +
	"5802BR5913FULANO DE TAL6009SAO PAULO62130509PEDIDO12363048C31"

// longTxIDPayload is a dynamic code with "https://" in the URL and a
// 33-character txid.
const longTxIDPayload = "00020101021226550014br.gov.bcb.pix2533https://pix.example.com/qr/v2/abc5204000053039865802BR5901X6001Y" +
	"62370533TTTTTTTTTTTTTTTTTTTTTTTTTTTTTTTTT63049399"

func TestParse_Dynamic(t *testing.T) {
	q, err := Parse(dynamicPayload)
	if err != nil {
		t.Fatal(err)
	}
	if !q.Dynamic {
		t.Error("Dynamic = false, want true")
	}
	if want := "pix.example.com/api/v2/cobv/0a1b2c3d4e5f60718293a4b5c6d7e8f9"; q.URL != want || q.PaymentKey() != want {
		t.Errorf("URL = %q, PaymentKey = %q, want %q", q.URL, q.PaymentKey(), want)
	}
	if q.TxID != "" {
		t.Errorf("TxID = %q, want empty for ***", q.TxID)
	}
	if q.Amount != 0 {
		t.Errorf("Amount = %s, want zero", q.Amount)
	}
	if q.MerchantName != "EXEMPLO RECEBEDOR SA" || q.City != "Brasilia" {
		t.Errorf("merchant = %q / %q", q.MerchantName, q.City)
	}
}

func TestParse_Static(t *testing.T) {
	q, err := Parse(staticPayload)
	if err != nil {
		t.Fatal(err)
	}
	if q.Dynamic || q.URL != "" {
		t.Errorf("Dynamic = %v, URL = %q, want static", q.Dynamic, q.URL)
	}
	if want := "123e4567-e12b-12d1-a456-426614174000"; q.Key != want || q.PaymentKey() != want {
		t.Errorf("Key = %q, PaymentKey = %q", q.Key, q.PaymentKey())
	}
	if q.TxID != "PEDIDO123" {
		t.Errorf("TxID = %q", q.TxID)
	}
	if q.Amount != core.Amount(1050) {
		t.Errorf("Amount = %s, want 10.50", q.Amount)
	}
}

func TestParse_StripsScheme(t *testing.T) {
	q, err := Parse(longTxIDPayload)
	if err != nil {
		t.Fatal(err)
	}
	if q.URL != "pix.example.com/qr/v2/abc" {
		t.Errorf("URL = %q", q.URL)
	}
	if len(q.TxID) != 33 {
		t.Errorf("TxID length = %d, want 33", len(q.TxID))
	}
}

func TestParse_Invalid(t *testing.T) {
	cases := map[string]string{
		"empty":     "",
		"bad CRC":   dynamicPayload[:len(dynamicPayload)-4] + "0000",
		"no CRC":    dynamicPayload[:len(dynamicPayload)-8],
		"truncated": dynamicPayload[:60],
		"not EMV":   "34191790010104351004791020150008291070026000",
		"no tag 26": withCRC(tlv("00", "01") + tlv("59", "X")),
		"wrong GUI": withCRC(tlv("00", "01") + tlv("26", tlv("00", "br.example")+tlv("01", "k"))),
		"bad amount": withCRC(tlv("00", "01") + tlv("26", tlv("00", "br.gov.bcb.pix")+tlv("01", "k")) +
			tlv("54", "10.505")),
	}
	for name, payload := range cases {
		if _, err := Parse(payload); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}
}

func TestParse_LowercaseCRCAccepted(t *testing.T) {
	p := dynamicPayload[:len(dynamicPayload)-4] + strings.ToLower(dynamicPayload[len(dynamicPayload)-4:])
	if _, err := Parse(p); err != nil {
		t.Fatal(err)
	}
}
