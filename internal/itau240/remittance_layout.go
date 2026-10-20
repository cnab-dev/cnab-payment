package itau240

import (
	"strconv"
	"strings"

	"github.com/cnab-dev/cnab-payment/internal/cnab240"
	"github.com/cnab-dev/cnab-payment/internal/core"
)

// batchKind is the kind of batch (lote) a payment lands in. Each kind has
// its own Febraban "Tipo de Serviço"/"Forma de Lançamento" pair and
// detail-record shape.
type batchKind int

const (
	kindCollection  batchKind = iota // arrecadação with barcode (segment O)
	kindBoletoItau                   // boleto issued by Itaú (J + J-52)
	kindBoletoOther                  // boleto issued by another bank (J + J-52)
	kindPixQRCode                    // Pix QR code (J + J-52 PIX)
)

type detailFn func(p core.Payment, batch, seq int) ([]string, cnab240.BatchTotals)
type trailerFn func(batch int, totals cnab240.BatchTotals) string

type batchSpec struct {
	paymentType   string // Tipo de Serviço
	paymentMethod string // Forma de Lançamento
	detail        detailFn
	trailer       trailerFn
}

var batchSpecs = map[batchKind]batchSpec{
	kindCollection:  {"22", "91", detailO, trailerO},
	kindBoletoItau:  {"20", "30", detailJ, trailerAJ},
	kindBoletoOther: {"20", "31", detailJ, trailerAJ},
	kindPixQRCode:   {"20", "47", detailJPix, trailerAJ},
}

// paymentKind classifies a payment already accepted by validatePayment.
func paymentKind(p core.Payment) batchKind {
	switch v := p.(type) {
	case core.BarcodePayment:
		switch {
		case strings.HasPrefix(v.Barcode, "8"):
			return kindCollection
		case strings.HasPrefix(v.Barcode, bankCode):
			return kindBoletoItau
		default:
			return kindBoletoOther
		}
	case core.QRCodePayment:
		return kindPixQRCode
	}
	panic("itau240: payment kind not accepted by Validate")
}

// remittanceLayout implements cnab240.Layout for Itaú SISPAG.
type remittanceLayout struct{}

// GroupKey puts payments with the same payer account and batch kind in the
// same batch.
func (remittanceLayout) GroupKey(p core.Payment) string {
	a := p.Base().PayerAccount
	return a.BankCode + a.Branch + a.Number + ":" + strconv.Itoa(int(paymentKind(p)))
}

func (remittanceLayout) BuildFileHeader(ctx cnab240.FileContext) string {
	return buildFileHeader(ctx)
}

func (remittanceLayout) BuildFileTrailer(totals cnab240.FileTotals) string {
	return buildFileTrailer(totals)
}

func (remittanceLayout) BuildBatchHeader(state *cnab240.BuildState, batchKey string) string {
	spec := batchSpecs[kindFromKey(batchKey)]
	return buildBatchHeader(state, spec.paymentType, spec.paymentMethod)
}

func (remittanceLayout) BuildBatchTrailer(state *cnab240.BuildState, batchKey string, totals cnab240.BatchTotals) string {
	return batchSpecs[kindFromKey(batchKey)].trailer(state.BatchSequence, totals)
}

func (remittanceLayout) BuildDetail(state *cnab240.BuildState, p core.Payment) ([]string, cnab240.BatchTotals) {
	return batchSpecs[paymentKind(p)].detail(p, state.BatchSequence, state.PaymentSeq)
}

// NormalizeRecord strips accents and upper-cases the record, except
// positions 132-240 of segment J-52, which J-52 PIX uses for the QR code
// URL or key and the TXID: the URL path is case-sensitive and must go as
// captured (nota 41). On a boleto's J-52 they are zeros and blanks.
func (remittanceLayout) NormalizeRecord(record string) string {
	if len(record) == recordLength && record[13] == 'J' && record[17:19] == "52" {
		return cnab240.StripAccentsUpper(record[:131]) + cnab240.StripAccents(record[131:])
	}
	return cnab240.StripAccentsUpper(record)
}

func kindFromKey(key string) batchKind {
	n, _ := strconv.Atoi(key[strings.LastIndex(key, ":")+1:])
	return batchKind(n)
}
