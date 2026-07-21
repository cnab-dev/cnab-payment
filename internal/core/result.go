package core

// ParseResult reports statistics about a parse run, whether it completed
// successfully or was aborted. It is deliberately minimal and expected to
// grow as the parser gains capabilities.
type ParseResult struct {
	// RecordsRead is the number of input records the parser consumed.
	RecordsRead int

	// OccurrencesEmitted is the number of Occurrence values successfully
	// emitted to the OccurrenceHandler.
	OccurrencesEmitted int
}
