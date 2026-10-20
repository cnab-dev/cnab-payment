// Package pix parses Pix QR code payloads ("Pix copia e cola"), the
// Central Bank of Brazil's BR Code: an EMV merchant-presented TLV sequence
// protected by a CRC16. It knows nothing about any bank layout.
package pix

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/cnab-dev/cnab-payment/internal/core"
)

// QRCode is the parsed content of a Pix QR code payload.
type QRCode struct {
	Dynamic      bool        // the payload carries a location URL (tag 26/25)
	URL          string      // dynamic: location URL without its scheme
	Key          string      // static: Pix key (tag 26/01)
	TxID         string      // tag 62/05; empty when absent or "***"
	MerchantName string      // tag 59
	City         string      // tag 60
	Amount       core.Amount // tag 54; zero when absent
}

// PaymentKey returns what identifies the charge to the bank: the URL for
// dynamic codes, the Pix key for static ones.
func (q QRCode) PaymentKey() string {
	if q.Dynamic {
		return q.URL
	}
	return q.Key
}

const gui = "br.gov.bcb.pix"

// Parse parses and validates a Pix QR code payload: TLV structure, CRC16
// (tag 63) and the Pix merchant account (tag 26) with either a URL
// (dynamic) or a Pix key (static).
func Parse(payload string) (QRCode, error) {
	var q QRCode
	if err := checkCRC(payload); err != nil {
		return q, err
	}
	fields, err := parseTLV(payload)
	if err != nil {
		return q, err
	}
	if fields["00"] != "01" {
		return q, errors.New("payload format indicator (tag 00) must be 01")
	}

	mai, ok := fields["26"]
	if !ok {
		return q, errors.New("missing Pix merchant account (tag 26)")
	}
	acc, err := parseTLV(mai)
	if err != nil {
		return q, fmt.Errorf("tag 26: %w", err)
	}
	if !strings.EqualFold(acc["00"], gui) {
		return q, fmt.Errorf("tag 26: GUI must be %s", gui)
	}
	if u := acc["25"]; u != "" {
		q.Dynamic = true
		q.URL = stripScheme(u)
	} else if k := acc["01"]; k != "" {
		q.Key = k
	} else {
		return q, errors.New("tag 26: missing URL (25) or Pix key (01)")
	}

	if add, ok := fields["62"]; ok {
		adf, err := parseTLV(add)
		if err != nil {
			return q, fmt.Errorf("tag 62: %w", err)
		}
		if t := adf["05"]; t != "***" {
			q.TxID = t
		}
	}
	if a, ok := fields["54"]; ok {
		amount, err := core.ParseAmount(a)
		if err != nil || amount <= 0 {
			return q, fmt.Errorf("tag 54: invalid amount %q", a)
		}
		q.Amount = amount
	}
	q.MerchantName = fields["59"]
	q.City = fields["60"]
	return q, nil
}

// parseTLV splits an EMV TLV sequence (2-digit id, 2-digit length in
// characters, value) into a map.
func parseTLV(s string) (map[string]string, error) {
	r := []rune(s)
	out := map[string]string{}
	for i := 0; i < len(r); {
		if i+4 > len(r) {
			return nil, fmt.Errorf("truncated field at position %d", i)
		}
		id := string(r[i : i+2])
		n, err := strconv.Atoi(string(r[i+2 : i+4]))
		if err != nil || n < 0 || !isDigits(id) {
			return nil, fmt.Errorf("invalid field header at position %d", i)
		}
		if i+4+n > len(r) {
			return nil, fmt.Errorf("field %s: truncated value", id)
		}
		out[id] = string(r[i+4 : i+4+n])
		i += 4 + n
	}
	return out, nil
}

// checkCRC verifies that the payload ends with tag 63 (length 04) holding
// the CRC16-CCITT (poly 0x1021, init 0xFFFF) of everything up to and
// including "6304".
func checkCRC(payload string) error {
	if len(payload) < 8 || payload[len(payload)-8:len(payload)-4] != "6304" {
		return errors.New("missing CRC (tag 63) at the end of the payload")
	}
	want := strings.ToUpper(payload[len(payload)-4:])
	got := fmt.Sprintf("%04X", crc16CCITT([]byte(payload[:len(payload)-4])))
	if got != want {
		return fmt.Errorf("invalid CRC: got %s, payload says %s", got, want)
	}
	return nil
}

func crc16CCITT(data []byte) uint16 {
	crc := uint16(0xFFFF)
	for _, b := range data {
		crc ^= uint16(b) << 8
		for range 8 {
			if crc&0x8000 != 0 {
				crc = crc<<1 ^ 0x1021
			} else {
				crc <<= 1
			}
		}
	}
	return crc
}

func stripScheme(u string) string {
	for _, p := range []string{"https://", "http://"} {
		if len(u) >= len(p) && strings.EqualFold(u[:len(p)], p) {
			return u[len(p):]
		}
	}
	return u
}

func isDigits(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return len(s) > 0
}
