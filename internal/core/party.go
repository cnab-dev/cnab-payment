package core

// Party is an individual or a company taking part in a payment.
type Party struct {
	Name string

	// TaxID holds digits only: 11 for a CPF, 14 for a CNPJ.
	TaxID string

	// Address is optional.
	Address *Address
}

// Address is a postal address.
type Address struct {
	Street     string
	Number     string
	Complement string
	City       string
	State      string
	ZipCode    string
}
