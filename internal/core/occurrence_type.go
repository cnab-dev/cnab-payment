package core

// OccurrenceType classifies an Occurrence's canonical outcome, independent
// of any bank-specific raw code. Layout parsers derive it from RawType.
type OccurrenceType string

const (
	// OccurrenceTypeRejected means the payment was rejected.
	OccurrenceTypeRejected OccurrenceType = "REJECTED"

	// OccurrenceTypeScheduled means the payment was accepted and
	// scheduled, but not yet settled.
	OccurrenceTypeScheduled OccurrenceType = "SCHEDULED"

	// OccurrenceTypeSettled means the payment was settled.
	OccurrenceTypeSettled OccurrenceType = "SETTLED"

	// OccurrenceTypeUnknown means the layout parser could not classify
	// RawType into one of the other OccurrenceType values.
	OccurrenceTypeUnknown OccurrenceType = "UNKNOWN"
)
