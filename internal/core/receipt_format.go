package core

import "fmt"

// ReceiptFormat identifies which receipt template a settled Occurrence
// corresponds to, as "BBB-S-TT-FF": bank compensation code, batch segment,
// and the Febraban "Tipo de Serviço"/"Forma de Lançamento" codes from the
// batch header. It is a lookup key only — consumers own the actual receipt
// templates.
type ReceiptFormat string

// NewReceiptFormat builds a ReceiptFormat from its four components, so
// every LayoutParser assembles the key identically.
func NewReceiptFormat(bankCode string, segment byte, paymentType, paymentMethod string) ReceiptFormat {
	return ReceiptFormat(fmt.Sprintf("%s-%c-%s-%s", bankCode, segment, paymentType, paymentMethod))
}
