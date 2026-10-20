package core

// Destination is where a TransferPayment sends money. It is sealed: only
// BankAccount and PixKey implement it, so a transfer always has exactly
// one kind of destination.
type Destination interface {
	isDestination()
}

// BankAccount is a bank account, used both as a payer account and as a
// transfer destination.
type BankAccount struct {
	// BankCode is the 3-digit bank compensation code.
	BankCode string

	Branch     string
	Number     string
	CheckDigit string

	Type AccountType
}

// AccountType describes a BankAccount. It says nothing about how money
// reaches the account; that's the transfer's Rail.
type AccountType string

const (
	AccountTypeChecking AccountType = "checking"
	AccountTypeSavings  AccountType = "savings"
	AccountTypeSalary   AccountType = "salary"
	AccountTypePayment  AccountType = "payment"
)

// PixKey is a Pix key registered in the DICT.
type PixKey struct {
	Type  PixKeyType
	Value string
}

// PixKeyType is the kind of a PixKey. It is provided by the caller; layouts
// never infer it from the key's value.
type PixKeyType string

const (
	// PixKeyTypeTaxID is a CPF or CNPJ key.
	PixKeyTypeTaxID  PixKeyType = "tax_id"
	PixKeyTypeEmail  PixKeyType = "email"
	PixKeyTypePhone  PixKeyType = "phone"
	PixKeyTypeRandom PixKeyType = "random"
)

func (BankAccount) isDestination() {}
func (PixKey) isDestination()      {}
