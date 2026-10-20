package itau240

import (
	"github.com/cnab-dev/cnab-payment/internal/cnab240"
	"github.com/cnab-dev/cnab-payment/internal/core"
	"github.com/cnab-dev/cnab-payment/internal/pix"
	"github.com/cnab-dev/cnab-payment/internal/taxid"
)

// Field positions and notes ("nota N") refer to the Itaú SISPAG CNAB240
// manual.

const (
	fileLayoutVersion  = "081"
	batchLayoutVersion = "083"
)

func buildFileHeader(ctx cnab240.FileContext) string {
	a := ctx.PayerAccount
	return bankCode +
		cnab240.Zeros(4) +
		"0" +
		cnab240.Spaces(6) +
		fileLayoutVersion +
		taxIDWithType14(ctx.Payer) +
		cnab240.Spaces(20) +
		cnab240.PadL(a.Branch, 5, '0') +
		" " +
		cnab240.PadL(a.Number, 12, '0') +
		" " +
		cnab240.PadR(a.CheckDigit, 1, ' ') +
		cnab240.PadR(ctx.Payer.Name, 30, ' ') +
		cnab240.PadR("ITAU UNIBANCO SA", 30, ' ') +
		cnab240.Spaces(10) +
		string(fileKindRemessa) +
		cnab240.FmtDate(ctx.CreatedAt) +
		cnab240.FmtTime(ctx.CreatedAt) +
		cnab240.Zeros(9) +
		cnab240.Zeros(5) +
		cnab240.Spaces(69)
}

func buildFileTrailer(totals cnab240.FileTotals) string {
	return bankCode +
		"9999" +
		"9" +
		cnab240.Spaces(9) +
		cnab240.FmtInt(totals.BatchCount, 6) +
		cnab240.FmtInt(totals.RecordCount+2, 6) + // + file header and trailer
		cnab240.Spaces(211)
}

func buildBatchHeader(state *cnab240.BuildState, paymentType, paymentMethod string) string {
	a, payer := state.PayerAccount, state.Payer
	addr := addressOf(payer)
	return bankCode +
		cnab240.FmtInt(state.BatchSequence, 4) +
		"1" +
		"C" +
		paymentType +
		paymentMethod +
		batchLayoutVersion +
		" " +
		taxIDWithType14(payer) +
		cnab240.Spaces(20) +
		cnab240.PadL(a.Branch, 5, '0') +
		" " +
		cnab240.PadL(a.Number, 12, '0') +
		" " +
		cnab240.PadR(a.CheckDigit, 1, ' ') +
		cnab240.PadR(payer.Name, 30, ' ') +
		cnab240.Spaces(30) +
		cnab240.Spaces(10) +
		cnab240.PadR(addr.Street+" "+addr.Number+" "+addr.Complement, 30, ' ') +
		cnab240.Zeros(5) +
		cnab240.Spaces(15) +
		cnab240.PadR(addr.City, 20, ' ') +
		cnab240.PadL(addr.ZipCode, 8, '0') +
		cnab240.PadR(addr.State, 2, ' ') +
		cnab240.Spaces(8) +
		cnab240.Spaces(10)
}

func buildBatchTrailer(batch int, totals cnab240.BatchTotals, zeroPad, spacePad int) string {
	return bankCode +
		cnab240.FmtInt(batch, 4) +
		"5" +
		cnab240.Spaces(9) +
		cnab240.FmtInt(totals.RecordCount+1, 6) + // + batch trailer
		cnab240.FmtAmount(totals.TotalAmount, 18) +
		cnab240.Zeros(zeroPad) +
		cnab240.Spaces(spacePad) +
		cnab240.Spaces(10)
}

func trailerAJ(batch int, t cnab240.BatchTotals) string { return buildBatchTrailer(batch, t, 18, 171) }
func trailerO(batch int, t cnab240.BatchTotals) string  { return buildBatchTrailer(batch, t, 15, 174) }

// segmentPrefix builds positions 1-17 of a detail record: bank, batch,
// record type, sequence number, segment code and movement type.
func segmentPrefix(batch, seq int, segment byte) string {
	return bankCode + cnab240.FmtInt(batch, 4) + "3" + cnab240.FmtInt(seq, 5) + string(segment) + "000"
}

func detailJ(p core.Payment, batch, seq int) ([]string, cnab240.BatchTotals) {
	bp := p.(core.BarcodePayment)

	segJ := segmentJ(bp.PaymentBase, cnab240.PadR(bp.Barcode, 44, ' '), batch, seq)
	segJ52 := segmentJ52Prefix(bp.PaymentBase, batch, seq) +
		"0" +
		cnab240.Zeros(15) +
		cnab240.Spaces(40) +
		cnab240.Spaces(53)

	return []string{segJ, segJ52}, cnab240.BatchTotals{RecordCount: 2, PaymentCount: 1, TotalAmount: bp.Amount}
}

// detailJPix builds segment J with the barcode zeroed (nota 18) and
// segment J-52 PIX.
func detailJPix(p core.Payment, batch, seq int) ([]string, cnab240.BatchTotals) {
	qp := p.(core.QRCodePayment)
	q, _ := pix.Parse(qp.Payload) // already checked by validatePayment

	txid := q.TxID
	if qp.TxID != "" {
		txid = qp.TxID
	}
	if len(txid) > 32 {
		txid = "" // nota 38: optional; the field holds 32 characters
	}

	segJ := segmentJ(qp.PaymentBase, cnab240.Zeros(44), batch, seq)
	segJ52 := segmentJ52Prefix(qp.PaymentBase, batch, seq) +
		cnab240.PadR(q.PaymentKey(), 77, ' ') +
		cnab240.PadR(txid, 32, ' ')

	return []string{segJ, segJ52}, cnab240.BatchTotals{RecordCount: 2, PaymentCount: 1, TotalAmount: qp.Amount}
}

func segmentJ(b core.PaymentBase, barcode string, batch, seq int) string {
	return segmentPrefix(batch, seq, 'J') +
		barcode +
		cnab240.PadR(b.Payee.Name, 30, ' ') +
		cnab240.FmtDate(b.DueDate) +
		cnab240.FmtAmount(b.Amount, 15) +
		cnab240.Zeros(15) +
		cnab240.Zeros(15) +
		cnab240.FmtDate(b.SettlementDate) +
		cnab240.FmtAmount(b.Amount, 15) +
		cnab240.Zeros(15) +
		cnab240.PadR(b.ID, 20, ' ') +
		cnab240.Spaces(13) +
		cnab240.Spaces(15) +
		cnab240.Spaces(10)
}

// segmentJ52Prefix builds positions 1-131 of segment J-52, the debtor
// (sacado) and the beneficiary (cedente), shared by J-52 and J-52 PIX.
func segmentJ52Prefix(b core.PaymentBase, batch, seq int) string {
	debtor := b.Payer
	if b.Debtor != nil {
		debtor = *b.Debtor
	}
	return segmentPrefix(batch, seq, 'J') +
		"52" +
		taxIDWithType15(debtor) +
		cnab240.PadR(debtor.Name, 40, ' ') +
		taxIDWithType15(b.Payee) +
		cnab240.PadR(b.Payee.Name, 40, ' ')
}

func detailO(p core.Payment, batch, seq int) ([]string, cnab240.BatchTotals) {
	bp := p.(core.BarcodePayment)

	s := segmentPrefix(batch, seq, 'O') +
		collectionLine(bp.Barcode) +
		cnab240.Spaces(30) +
		cnab240.FmtDate(bp.SettlementDate) +
		"REA" +
		cnab240.Zeros(15) +
		cnab240.FmtAmount(bp.Amount, 15) +
		cnab240.FmtDate(bp.SettlementDate) +
		cnab240.Zeros(15) +
		cnab240.Spaces(3) +
		cnab240.Zeros(9) +
		cnab240.Spaces(3) +
		cnab240.PadR(bp.ID, 20, ' ') +
		cnab240.Spaces(21) +
		cnab240.Spaces(15) +
		cnab240.Spaces(10)

	return []string{s}, cnab240.BatchTotals{RecordCount: 1, PaymentCount: 1, TotalAmount: bp.Amount}
}

// taxIDType is the registration type: 1 for CPF, 2 for CNPJ.
func taxIDType(id string) string {
	if taxid.IsCNPJ(id) {
		return "2"
	}
	return "1"
}

// taxIDWithType14 is the registration type followed by the tax ID in 14
// digits, as in the file and batch headers.
func taxIDWithType14(p core.Party) string {
	return taxIDType(p.TaxID) + cnab240.PadL(p.TaxID, 14, '0')
}

// taxIDWithType15 is the registration type followed by the tax ID in 15
// digits (nota 36).
func taxIDWithType15(p core.Party) string {
	return taxIDType(p.TaxID) + cnab240.PadL(p.TaxID, 15, '0')
}

func addressOf(p core.Party) core.Address {
	if p.Address == nil {
		return core.Address{}
	}
	return *p.Address
}
