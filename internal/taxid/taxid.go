// Package taxid validates Brazilian taxpayer IDs: CPF (individuals, 11
// digits) and CNPJ (companies, 14 digits), both digits only.
package taxid

import (
	"errors"
	"fmt"
)

// Validate reports whether s is a valid CPF or CNPJ, by length and check
// digits.
func Validate(s string) error {
	switch len(s) {
	case 11:
		if err := validateCPF(s); err != nil {
			return fmt.Errorf("CPF %s: %w", s, err)
		}
		return nil
	case 14:
		if err := validateCNPJ(s); err != nil {
			return fmt.Errorf("CNPJ %s: %w", s, err)
		}
		return nil
	default:
		return fmt.Errorf("must be 11 digits (CPF) or 14 digits (CNPJ), got %d", len(s))
	}
}

// IsCNPJ reports whether s has a CNPJ's length. It doesn't validate s.
func IsCNPJ(s string) bool { return len(s) == 14 }

func validateCPF(s string) error {
	if !isDigits(s) {
		return errors.New("must be 11 digits")
	}
	if allSameDigit(s) {
		return errors.New("invalid")
	}
	if checkDigit(s[:9], 10) != s[9] || checkDigit(s[:10], 11) != s[10] {
		return errors.New("invalid check digit")
	}
	return nil
}

func validateCNPJ(s string) error {
	if !isDigits(s) {
		return errors.New("must be 14 digits")
	}
	if allSameDigit(s) {
		return errors.New("invalid")
	}
	if cnpjCheckDigit(s[:12]) != s[12] || cnpjCheckDigit(s[:13]) != s[13] {
		return errors.New("invalid check digit")
	}
	return nil
}

// checkDigit computes a CPF check digit over base, with weights descending
// from firstWeight.
func checkDigit(base string, firstWeight int) byte {
	sum := 0
	for i := 0; i < len(base); i++ {
		sum += int(base[i]-'0') * (firstWeight - i)
	}
	return mod11Digit(sum)
}

// cnpjCheckDigit computes a CNPJ check digit over base, with weights
// cycling 2..9 from the right.
func cnpjCheckDigit(base string) byte {
	sum, weight := 0, 2
	for i := len(base) - 1; i >= 0; i-- {
		sum += int(base[i]-'0') * weight
		if weight++; weight > 9 {
			weight = 2
		}
	}
	return mod11Digit(sum)
}

func mod11Digit(sum int) byte {
	if rem := sum % 11; rem >= 2 {
		return byte('0' + 11 - rem)
	}
	return '0'
}

func isDigits(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return len(s) > 0
}

func allSameDigit(s string) bool {
	for i := 1; i < len(s); i++ {
		if s[i] != s[0] {
			return false
		}
	}
	return true
}
