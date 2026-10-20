package taxid

import "testing"

func TestValidate_Valid(t *testing.T) {
	// Synthetic IDs with valid check digits.
	for _, s := range []string{"00000100064", "52998224725", "00000090000110", "11222333000181"} {
		if err := Validate(s); err != nil {
			t.Errorf("Validate(%q) = %v, want nil", s, err)
		}
	}
}

func TestValidate_Invalid(t *testing.T) {
	cases := map[string]string{
		"wrong CPF check digit":  "52998224724",
		"wrong CNPJ check digit": "11222333000180",
		"all same digit":         "11111111111",
		"non-digit":              "5299822472X",
		"wrong length":           "123",
		"empty":                  "",
	}
	for name, s := range cases {
		if err := Validate(s); err == nil {
			t.Errorf("%s: Validate(%q) = nil, want error", name, s)
		}
	}
}
