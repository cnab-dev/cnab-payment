package itau240

import "errors"

// ErrUnexpectedRecord is returned when a record appears where the Itaú
// CNAB240 grammar does not allow it (e.g. a detail record outside any
// batch, or a batch trailer with no matching header). It leaves the
// parser unable to safely determine which batch, or file, a subsequent
// record belongs to, so it is a structural failure that aborts parsing.
var ErrUnexpectedRecord = errors.New("itau240: record not valid in the current position")

// ErrMissingFileTrailer is returned when the input ends without a file
// trailer record.
var ErrMissingFileTrailer = errors.New("itau240: input ended before a file trailer was found")
