package cnabpayment_test

import (
	"errors"
	"fmt"
	"strings"
	"time"

	cnabpayment "github.com/cnab-dev/cnab-payment"
)

func ExampleGenerator_Generate() {
	g := cnabpayment.NewGenerator(cnabpayment.NewItau240LayoutGenerator())

	payer := cnabpayment.Party{Name: "EMPRESA EXEMPLO LTDA", TaxID: "11222333000181"}
	payerAccount := cnabpayment.BankAccount{
		BankCode: "341", Branch: "0610", Number: "20445", CheckDigit: "5",
		Type: cnabpayment.AccountTypeChecking,
	}
	base := func(id string, amount cnabpayment.Amount) cnabpayment.PaymentBase {
		day := time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC)
		return cnabpayment.PaymentBase{
			ID:             id,
			Payer:          payer,
			PayerAccount:   payerAccount,
			Payee:          cnabpayment.Party{Name: "ANA EXEMPLO", TaxID: "52998224725"},
			Amount:         amount,
			DueDate:        day,
			SettlementDate: day,
		}
	}

	payments := []cnabpayment.Payment{
		cnabpayment.TransferPayment{
			PaymentBase: base("PAY02", 15000),
			Rail:        cnabpayment.RailPix,
			Destination: cnabpayment.PixKey{Type: cnabpayment.PixKeyTypeEmail, Value: "ana@example.com"},
		},
		cnabpayment.TransferPayment{
			PaymentBase: base("PAY01", 52000),
			Rail:        cnabpayment.RailTED,
			Destination: cnabpayment.BankAccount{
				BankCode: "237", Branch: "0321", Number: "4567", CheckDigit: "3",
				Type: cnabpayment.AccountTypeChecking,
			},
		},
	}

	files, err := g.Generate(payments, cnabpayment.GenerateOptions{
		CreatedAt: time.Date(2026, 4, 20, 10, 0, 0, 0, time.FixedZone("BRT", -3*60*60)),
	})
	if err != nil {
		fmt.Println(err)
		return
	}
	for _, f := range files {
		fmt.Println(f.Class, f.PaymentIDs, strings.Count(string(f.Content), "\r\n"), "records")
	}
	// Output:
	// other [PAY01] 5 records
	// pix [PAY02] 6 records
}

func ExampleGenerator_Validate() {
	g := cnabpayment.NewGenerator(cnabpayment.NewItau240LayoutGenerator())

	ted := cnabpayment.TransferPayment{
		PaymentBase: cnabpayment.PaymentBase{
			ID:             "PAY01",
			PayerAccount:   cnabpayment.BankAccount{BankCode: "341", Branch: "0610", Number: "20445", CheckDigit: "5"},
			Amount:         1000,
			DueDate:        time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC),
			SettlementDate: time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC),
		},
		Rail: cnabpayment.RailTED,
		Destination: cnabpayment.BankAccount{
			BankCode: "341", Branch: "0567", Number: "1234", CheckDigit: "9",
			Type: cnabpayment.AccountTypeChecking,
		},
	}

	err := g.Validate(ted)
	fmt.Println(errors.Is(err, cnabpayment.ErrUnsupportedRail))
	// Output:
	// true
}
