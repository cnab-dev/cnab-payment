// Package core defines the canonical domain model and physical-record
// scanner shared by the root package and every layout parser, independent
// of any bank-specific CNAB layout.
package core

import "time"

// Occurrence is the canonical representation of a single payment occurrence
// extracted from a CNAB return file, independent of which bank or layout
// produced it. Its field set is expected to grow as more layout parsers
// are added and more of a return file's business information becomes
// worth extracting.
type Occurrence struct {
	// PaymentID identifies the payment this occurrence refers to, as
	// assigned by whoever generated the original payment instruction.
	PaymentID string

	// RawType is the occurrence/reason code reported by the bank for this
	// payment, exactly as it appears in the return file.
	RawType string

	// Type is the canonical outcome this occurrence represents, derived
	// from RawType by the layout parser. It is OccurrenceTypeUnknown when
	// the layout parser doesn't recognize RawType's value.
	Type OccurrenceType

	// ConfirmationID is a bank-issued confirmation identifier for this
	// payment, when the return file provides one. It is empty when no
	// such identifier was reported.
	ConfirmationID string

	// CreatedAt is the return file's generation date and time, as
	// reported by its file header.
	CreatedAt time.Time
}
