// Package cnab240 is the bank-agnostic engine for writing CNAB240
// remittance files: fixed-width field formatting, text normalization, and
// the assembly of a file from batches of detail records. Everything that
// varies by bank — record layouts, batch grouping, which segments a
// payment produces — is supplied by a Layout.
package cnab240

import (
	"strings"
	"unicode"
)

// StripAccents removes diacritics from s: a precomposed letter becomes its
// ASCII base letter (e.g. "Ç" → "C") and standalone nonspacing marks are
// dropped. Any other rune is kept as is.
func StripAccents(s string) string {
	return strings.Map(func(r rune) rune {
		if r <= unicode.MaxASCII {
			return r
		}
		if b, ok := asciiFold[r]; ok {
			return rune(b)
		}
		if unicode.Is(unicode.Mn, r) {
			return -1
		}
		return r
	}, s)
}

// StripAccentsUpper is StripAccents followed by upper-casing, the default
// normalization applied to every record.
func StripAccentsUpper(s string) string {
	return strings.ToUpper(StripAccents(s))
}

// toASCII strips accents and replaces any other non-ASCII rune with a
// blank, so that padding and truncation count characters, not bytes.
func toASCII(s string) string {
	return strings.Map(func(r rune) rune {
		if r > unicode.MaxASCII {
			return ' '
		}
		return r
	}, StripAccents(s))
}

// PadL converts s to ASCII, left-pads it with ch to width n, then
// truncates it to n.
func PadL(s string, n int, ch byte) string {
	s = toASCII(s)
	if len(s) < n {
		s = strings.Repeat(string(ch), n-len(s)) + s
	}
	return s[:n]
}

// PadR converts s to ASCII, right-pads it with ch to width n, then
// truncates it to n.
func PadR(s string, n int, ch byte) string {
	s = toASCII(s)
	if len(s) < n {
		s += strings.Repeat(string(ch), n-len(s))
	}
	return s[:n]
}

// Zeros returns n '0' characters.
func Zeros(n int) string { return strings.Repeat("0", n) }

// Spaces returns n blanks.
func Spaces(n int) string { return strings.Repeat(" ", n) }
