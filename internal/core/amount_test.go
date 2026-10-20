package core

import (
	"math"
	"testing"
)

func TestParseAmount_ValidInput_ReturnsCents(t *testing.T) {
	tests := []struct {
		in   string
		want Amount
	}{
		{"123.45", 12345},
		{"7.5", 750},
		{"10", 1000},
		{"0", 0},
		{"0.05", 5},
		{"0.00", 0},
		{"007.10", 710},
		{"-1.50", -150},
		{"92233720368547757.07", 9223372036854775707},
	}
	for _, tt := range tests {
		got, err := ParseAmount(tt.in)
		if err != nil {
			t.Errorf("ParseAmount(%q) error = %v, want nil", tt.in, err)
			continue
		}
		if got != tt.want {
			t.Errorf("ParseAmount(%q) = %d, want %d", tt.in, got, tt.want)
		}
	}
}

func TestParseAmount_InvalidInput_ReturnsError(t *testing.T) {
	for _, in := range []string{
		"", "-", ".", ".5", "5.", "1.234", "1,50", "1.000,00", "+1.00", "--1", "1e3", " 1.00", "1.00 ", "abc",
		"99999999999999999999",
	} {
		if got, err := ParseAmount(in); err == nil {
			t.Errorf("ParseAmount(%q) = %d, nil; want an error", in, got)
		}
	}
}

func TestString_Amount_FormatsTwoDecimalPlaces(t *testing.T) {
	tests := []struct {
		in   Amount
		want string
	}{
		{12345, "123.45"},
		{5, "0.05"},
		{0, "0.00"},
		{100, "1.00"},
		{-150, "-1.50"},
		{-5, "-0.05"},
		{math.MaxInt64, "92233720368547758.07"},
		{math.MinInt64, "-92233720368547758.08"},
	}
	for _, tt := range tests {
		if got := tt.in.String(); got != tt.want {
			t.Errorf("Amount(%d).String() = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestParseAmount_RoundTripsString(t *testing.T) {
	for _, a := range []Amount{0, 1, 99, 100, 12345, -150} {
		got, err := ParseAmount(a.String())
		if err != nil || got != a {
			t.Errorf("ParseAmount(%q) = %d, %v; want %d, nil", a.String(), got, err, a)
		}
	}
}
