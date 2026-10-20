package core

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Amount is a monetary value in cents (centavos). CNAB files only
// represent whole cents, so a value with a third decimal place can't be
// expressed.
type Amount int64

// ParseAmount parses a decimal string with a '.' separator and at most two
// decimal places, e.g. "123.45" → 12345, "7.5" → 750, "10" → 1000. An
// optional leading '-' is accepted. Thousands separators, a ',' decimal
// separator, a leading '+', and a missing integer part (".5") are
// rejected.
func ParseAmount(s string) (Amount, error) {
	orig := s
	negative := strings.HasPrefix(s, "-")
	if negative {
		s = s[1:]
	}

	intPart, fracPart, hasDot := strings.Cut(s, ".")
	if intPart == "" || !isDigits(intPart) {
		return 0, fmt.Errorf("cnab-payment: invalid amount %q", orig)
	}
	if hasDot {
		if fracPart == "" || !isDigits(fracPart) {
			return 0, fmt.Errorf("cnab-payment: invalid amount %q", orig)
		}
		if len(fracPart) > 2 {
			return 0, fmt.Errorf("cnab-payment: invalid amount %q: more than two decimal places", orig)
		}
	}
	fracPart += strings.Repeat("0", 2-len(fracPart))

	units, err := strconv.ParseInt(intPart, 10, 64)
	if err != nil || units > (math.MaxInt64-99)/100 {
		return 0, fmt.Errorf("cnab-payment: invalid amount %q: %w", orig, errAmountOutOfRange)
	}
	cents, _ := strconv.ParseInt(fracPart, 10, 64)

	a := Amount(units*100 + cents)
	if negative {
		a = -a
	}
	return a, nil
}

var errAmountOutOfRange = errors.New("out of range")

// String formats a with exactly two decimal places, e.g. 12345 → "123.45",
// 5 → "0.05", -150 → "-1.50".
func (a Amount) String() string {
	sign := ""
	u := uint64(a)
	if a < 0 {
		sign = "-"
		u = uint64(-a) // math.MinInt64 wraps to its own magnitude as uint64
	}
	return fmt.Sprintf("%s%d.%02d", sign, u/100, u%100)
}

func isDigits(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}
