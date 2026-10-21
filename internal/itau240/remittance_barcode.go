package itau240

import (
	"strconv"
	"strings"
)

// isDARFDCTFWebBarcode reports whether barcode is a 44-digit collection
// barcode for a DARF issued by DCTFWeb: segment "85" (government, value in
// reais) and agreement "0385" at positions 16-19. It is the only kind of
// collection barcode segment O is written for.
func isDARFDCTFWebBarcode(barcode string) bool {
	return len(barcode) == 44 && isAllDigits(barcode) && barcode[:2] == "85" && barcode[15:19] == "0385"
}

// collectionLine expands a 44-digit collection barcode into the 48-digit
// line segment O expects: four 11-digit blocks, each followed by its
// modulo 11 check digit.
func collectionLine(barcode string) string {
	var line strings.Builder
	for i := 0; i < 44; i += 11 {
		block := barcode[i : i+11]
		line.WriteString(block + collectionCheckDigit(block))
	}
	return line.String()
}

// collectionCheckDigit is the modulo 11 check digit of a collection
// barcode block, weights 2..9 from the right; remainders 0 and 1 give 0.
func collectionCheckDigit(block string) string {
	sum, weight := 0, 2
	for i := len(block) - 1; i >= 0; i-- {
		sum += int(block[i]-'0') * weight
		if weight++; weight > 9 {
			weight = 2
		}
	}
	rem := sum % 11
	if rem == 0 || rem == 1 {
		return "0"
	}
	return strconv.Itoa(11 - rem)
}

func isAllDigits(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return len(s) > 0
}
