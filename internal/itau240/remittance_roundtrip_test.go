package itau240

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/cnab-dev/cnab-payment/internal/core"
	"github.com/cnab-dev/cnab-payment/internal/registry"
)

// toReturn turns a remittance into the return file the bank would send
// back for it, as far as the parser is concerned: the file header's kind
// becomes retorno, and every payment's mandatory segment (A, J or O)
// carries the occurrence code given for its payment ID in positions
// 231-240. Like the observed return files, segment B is echoed unchanged
// and segment J-52 is not echoed at all.
func toReturn(t *testing.T, remittance []byte, codes map[string]string) []byte {
	t.Helper()
	var records [][]byte
	for _, r := range bytes.Split(bytes.TrimSuffix(remittance, []byte("\r\n")), []byte("\r\n")) {
		if r[7] == recordTypeDetail && r[13] == 'J' && string(r[17:19]) == "52" {
			continue
		}
		r = bytes.Clone(r)
		records = append(records, r)
		switch r[7] {
		case recordTypeFileHeader:
			r[142] = fileKindRetorno
		case recordTypeDetail:
			var id string
			switch {
			case r[13] == 'A':
				id = string(r[73:93])
			case r[13] == 'J' && string(r[17:19]) != "52":
				id = string(r[182:202])
			case r[13] == 'O':
				id = string(r[174:194])
			default:
				continue
			}
			id = strings.TrimSpace(id)
			code, ok := codes[id]
			if !ok {
				t.Fatalf("no occurrence code for payment %q", id)
			}
			copy(r[230:240], fmt.Sprintf("%-10s", code))
		}
	}
	return append(bytes.Join(records, []byte("\r\n")), "\r\n"...)
}

// wantFormat is the receipt format key a payment's occurrence must carry:
// the mandatory segment and the batch's tipo/forma, per the translation
// table, derived here independently of the layout code.
func wantFormat(p core.Payment) core.ReceiptFormat {
	switch v := p.(type) {
	case core.QRCodePayment:
		return "341-J-20-47"
	case core.BarcodePayment:
		switch {
		case strings.HasPrefix(v.Barcode, "8"):
			return "341-O-22-91"
		case strings.HasPrefix(v.Barcode, "341"):
			return "341-J-20-30"
		default:
			return "341-J-20-31"
		}
	case core.TransferPayment:
		switch v.Rail {
		case core.RailPix:
			return "341-A-20-45"
		case core.RailTED:
			return "341-A-20-41"
		}
		switch v.Destination.(core.BankAccount).Type {
		case core.AccountTypeSavings:
			return "341-A-20-05"
		case core.AccountTypeSalary:
			return "341-A-30-01"
		default:
			return "341-A-20-01"
		}
	}
	return ""
}

// TestRoundTrip generates the remittance for every golden scenario plus a
// Pix QR code payment, turns each file into a return with a distinct
// occurrence per payment, parses it back, and checks that every payment
// comes back exactly once with its own ID, occurrence and receipt format.
func TestRoundTrip(t *testing.T) {
	f := loadFixtures(t)

	var scenarios [][]core.Payment
	for _, sc := range f.Scenarios {
		var ps []core.Payment
		for _, fp := range sc.Payments {
			ps = append(ps, fp.toCore(t))
		}
		scenarios = append(scenarios, ps)
	}
	qr := qrCodePayment()
	qr.TxID = "PEDIDO123"
	scenarios = append(scenarios, []core.Payment{qr})

	gen := registry.NewGenerator(NewGenerator())
	parser := registry.New(New())
	// Cycle through codes of every canonical type, including a rejection
	// with a reason.
	codes := []string{"00", "BD", "RJAN", "00"}
	wantTypes := map[string]core.OccurrenceType{
		"00": core.OccurrenceTypeSettled, "BD": core.OccurrenceTypeScheduled, "RJAN": core.OccurrenceTypeRejected,
	}

	for i, payments := range scenarios {
		files, err := gen.Generate(payments, core.GenerateOptions{CreatedAt: f.CreatedAt})
		if err != nil {
			t.Fatalf("scenario %d: %v", i, err)
		}

		byID := map[string]core.Payment{}
		codeByID := map[string]string{}
		for j, p := range payments {
			byID[p.Base().ID] = p
			codeByID[p.Base().ID] = codes[j%len(codes)]
		}

		seen := map[string]bool{}
		for _, file := range files {
			ret := toReturn(t, file.Content, codeByID)
			var recordErrs []string
			_, err := parser.Parse(bytes.NewReader(ret), func(o core.Occurrence) error {
				p, ok := byID[o.PaymentID]
				if !ok {
					t.Errorf("scenario %d: occurrence for unknown payment ID %q", i, o.PaymentID)
					return nil
				}
				if seen[o.PaymentID] {
					t.Errorf("scenario %d: payment %s reported twice", i, o.PaymentID)
				}
				seen[o.PaymentID] = true
				if o.RawType != codeByID[o.PaymentID] {
					t.Errorf("payment %s: RawType = %q, want %q", o.PaymentID, o.RawType, codeByID[o.PaymentID])
				}
				if want := wantTypes[codeByID[o.PaymentID]]; o.Type != want {
					t.Errorf("payment %s: Type = %s, want %s", o.PaymentID, o.Type, want)
				}
				if want := wantFormat(p); o.Format != want {
					t.Errorf("payment %s: Format = %s, want %s", o.PaymentID, o.Format, want)
				}
				if !o.CreatedAt.Equal(f.CreatedAt) {
					t.Errorf("payment %s: CreatedAt = %s, want %s", o.PaymentID, o.CreatedAt, f.CreatedAt)
				}
				return nil
			}, func(re core.RecordError) error {
				recordErrs = append(recordErrs, re.Error())
				return nil
			})
			if err != nil {
				t.Errorf("scenario %d, file %s: Parse: %v", i, file.Class, err)
			}
			if len(recordErrs) > 0 {
				t.Errorf("scenario %d, file %s: record errors: %v", i, file.Class, recordErrs)
			}
		}
		for id := range byID {
			if !seen[id] {
				t.Errorf("scenario %d: payment %s missing from the return", i, id)
			}
		}
	}
}
