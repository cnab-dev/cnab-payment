package core

import "time"

// RemittanceClass identifies which bank-mandated partition a
// RemittanceFile belongs to. Values are defined by each layout (Itaú:
// "pix" or "other"); they are lowercase ASCII and safe in a file name.
type RemittanceClass string

// RemittanceFile is one remittance file produced from a list of payments.
type RemittanceFile struct {
	// Class is the partition this file holds. Every payment in it has
	// this Class.
	Class RemittanceClass

	// PaymentIDs holds the IDs of the payments in this file, sorted
	// lexicographically.
	PaymentIDs []string

	// Content is the file itself, ready to send to the bank.
	Content []byte
}

// GenerateOptions configures a remittance generation run.
type GenerateOptions struct {
	// CreatedAt is written to the file header. Optional: zero means
	// time.Now(). Layouts convert it to their own local time (Itaú:
	// Brasília) before formatting, so any time zone may be passed.
	CreatedAt time.Time
}
