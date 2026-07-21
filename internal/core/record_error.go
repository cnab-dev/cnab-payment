package core

import "fmt"

// RecordError describes a recoverable failure encountered while parsing a
// single record. Unlike a fatal parse error, a RecordError does not by
// itself stop parsing; it is reported to a RecordErrorHandler, which
// decides whether to continue.
type RecordError struct {
	// Line is the 1-based line (or record) number where the error occurred.
	Line int

	// Raw holds the raw bytes of the offending record, as read from the
	// input, for diagnostics or reprocessing.
	Raw []byte

	// Err is the underlying cause of the error.
	Err error
}

func (e *RecordError) Error() string {
	return fmt.Sprintf("cnab-payment: record error at line %d: %v", e.Line, e.Err)
}

func (e *RecordError) Unwrap() error {
	return e.Err
}
