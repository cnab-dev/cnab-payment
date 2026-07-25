package itau240

import (
	"fmt"
	"strings"
)

// batchHeaderPaymentTypeStart/End and batchHeaderPaymentMethodStart/End mark
// the "Tipo de Serviço" and "Forma de Lançamento" fields (columns 10-11 and
// 12-13), per the Febraban CNAB240 Header de Lote layout, shared by
// remessa and retorno files; verify against Itaú's official manual before
// relying on them in production.
const (
	batchHeaderPaymentTypeStart, batchHeaderPaymentTypeEnd     = 9, 11
	batchHeaderPaymentMethodStart, batchHeaderPaymentMethodEnd = 11, 13
)

// batchHeader holds the fields extracted from a batch header (registro
// header de lote, tipo 1).
type batchHeader struct {
	// BatchNumber is the "lote de serviço" field (columns 4-7), used to
	// correlate this batch's header, detail records, and trailer.
	BatchNumber string

	// PaymentType is the "Tipo de Serviço" field (columns 10-11).
	PaymentType string

	// PaymentMethod is the "Forma de Lançamento" field (columns 12-13).
	PaymentMethod string
}

func decodeBatchHeader(raw []byte) (batchHeader, error) {
	if len(raw) != recordLength {
		return batchHeader{}, fmt.Errorf("itau240: batch header must be %d bytes, got %d", recordLength, len(raw))
	}
	if recordType(raw) != recordTypeBatchHeader {
		return batchHeader{}, fmt.Errorf("itau240: expected batch header record type %q, got %q", recordTypeBatchHeader, recordType(raw))
	}
	return batchHeader{
		BatchNumber:   batchNumber(raw),
		PaymentType:   strings.TrimSpace(string(raw[batchHeaderPaymentTypeStart:batchHeaderPaymentTypeEnd])),
		PaymentMethod: strings.TrimSpace(string(raw[batchHeaderPaymentMethodStart:batchHeaderPaymentMethodEnd])),
	}, nil
}

// batchTrailer holds the fields extracted from a batch trailer (registro
// trailer de lote, tipo 5).
type batchTrailer struct {
	BatchNumber string
}

func decodeBatchTrailer(raw []byte) (batchTrailer, error) {
	if len(raw) != recordLength {
		return batchTrailer{}, fmt.Errorf("itau240: batch trailer must be %d bytes, got %d", recordLength, len(raw))
	}
	if recordType(raw) != recordTypeBatchTrailer {
		return batchTrailer{}, fmt.Errorf("itau240: expected batch trailer record type %q, got %q", recordTypeBatchTrailer, recordType(raw))
	}
	return batchTrailer{BatchNumber: batchNumber(raw)}, nil
}
