package core

// OccurrenceHandler is invoked for every canonical occurrence emitted while
// parsing, as soon as it is fully parsed. Returning an error aborts
// parsing.
type OccurrenceHandler func(Occurrence) error

// RecordErrorHandler is invoked for every recoverable record-level error
// encountered while parsing. Returning an error aborts parsing. If nil is
// passed to a parser, record errors are ignored and parsing continues.
type RecordErrorHandler func(RecordError) error
