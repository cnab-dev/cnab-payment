package cnab240

import (
	"testing"
	"time"

	"github.com/cnab-dev/cnab-payment/internal/core"
)

func TestPadR_CountsCharactersNotBytes(t *testing.T) {
	cases := []struct{ in, want string }{
		{"JOÃO DA CONCEIÇÃO", "JOAO DA CONCEICAO   "},
		{"Rua X, Nº 10", "Rua X, N  10        "},
		{"AÇÃO", "ACAO                "},
		{"ação", "acao                "},
		{"Ångström", "Angstrom            "},
	}
	for _, c := range cases {
		if got := PadR(c.in, 20, ' '); got != c.want {
			t.Errorf("PadR(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestPadR_Truncates(t *testing.T) {
	if got := PadR("ABCDEFGH", 5, ' '); got != "ABCDE" {
		t.Errorf("PadR = %q, want %q", got, "ABCDE")
	}
}

func TestPadL(t *testing.T) {
	cases := []struct {
		in   string
		n    int
		want string
	}{
		{"Ç1", 4, "00C1"},
		{"123", 5, "00123"},
		{"123456", 4, "1234"},
		{"", 3, "000"},
	}
	for _, c := range cases {
		if got := PadL(c.in, c.n, '0'); got != c.want {
			t.Errorf("PadL(%q, %d) = %q, want %q", c.in, c.n, got, c.want)
		}
	}
}

func TestStripAccents_DropsCombiningMarks(t *testing.T) {
	// "e" + COMBINING ACUTE ACCENT, "c" + COMBINING CEDILLA
	if got := StripAccents("café ç"); got != "cafe c" {
		t.Errorf("StripAccents = %q, want %q", got, "cafe c")
	}
}

func TestStripAccents_KeepsRunesWithoutASCIIBase(t *testing.T) {
	// º has no canonical decomposition: kept here, blanked by padding.
	if got := StripAccents("Nº"); got != "Nº" {
		t.Errorf("StripAccents = %q, want %q", got, "Nº")
	}
}

func TestStripAccentsUpper(t *testing.T) {
	if got := StripAccentsUpper("São Paulo"); got != "SAO PAULO" {
		t.Errorf("StripAccentsUpper = %q, want %q", got, "SAO PAULO")
	}
}

func TestZerosAndSpaces(t *testing.T) {
	if Zeros(3) != "000" || Spaces(2) != "  " || Zeros(0) != "" {
		t.Errorf("Zeros/Spaces produced unexpected output")
	}
}

func TestFmtAmount(t *testing.T) {
	cases := []struct {
		a     core.Amount
		width int
		want  string
	}{
		{12345, 15, "000000000012345"},
		{0, 5, "00000"},
		{100000000, 15, "000000100000000"},
	}
	for _, c := range cases {
		if got := FmtAmount(c.a, c.width); got != c.want {
			t.Errorf("FmtAmount(%d, %d) = %q, want %q", c.a, c.width, got, c.want)
		}
	}
}

func TestFmtInt(t *testing.T) {
	if got := FmtInt(7, 4); got != "0007" {
		t.Errorf("FmtInt = %q, want %q", got, "0007")
	}
}

func TestFmtDateTime_UsesTheValuesOwnLocation(t *testing.T) {
	// 00:30 UTC on the 20th is still the 19th in UTC-3; FmtDate must not
	// convert, so a calendar date is written as given.
	utc := time.Date(2026, 4, 20, 0, 30, 15, 0, time.UTC)
	if got := FmtDate(utc); got != "20042026" {
		t.Errorf("FmtDate = %q, want %q", got, "20042026")
	}
	if got := FmtTime(utc); got != "003015" {
		t.Errorf("FmtTime = %q, want %q", got, "003015")
	}

	brt := utc.In(time.FixedZone("-03:00", -3*60*60))
	if got := FmtDate(brt); got != "19042026" {
		t.Errorf("FmtDate(in UTC-3) = %q, want %q", got, "19042026")
	}
}
