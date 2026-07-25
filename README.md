# cnab-payment

`cnab-payment` is a Go library for streaming **CNAB payment return files**
into a canonical, bank-agnostic stream of payment occurrences.

CNAB ("Centro Nacional de Automação Bancária") is a family of fixed-width
file formats used by Brazilian banks. Every bank implements its own layout,
which forces integrators to write bespoke parsing code per bank. This
library provides a single streaming entry point that reads a file's first
record, resolves the matching layout, and hands the rest of the stream off
to a pluggable **layout parser** — while exposing one small, stable API
regardless of which bank produced the file.

```go
import "github.com/cnab-dev/cnab-payment"

p := cnabpayment.NewParser(cnabpayment.NewItau240LayoutParser())

result, err := p.Parse(r, func(o cnabpayment.Occurrence) error {
    fmt.Println(o.PaymentID, o.Type, o.RawType, o.CreatedAt, o.Format)
    return nil
}, nil)
```

## Status

**Early foundation, first layout parser implemented.** The public API,
streaming infrastructure, and the first layout parser — Itaú CNAB240
payment return files — are in place. Other banks and CNAB400 are not
supported yet.

The current milestone is scoped strictly to **payment return parsing**.
Remittance file generation (exporting) is out of scope.

## A single public package

`github.com/cnab-dev/cnab-payment` (package `cnabpayment`) is the **only**
importable package. Everything else — the domain model, the scanner, the
`LayoutParser`/`Parser` orchestration, and every built-in layout parser's
actual implementation — lives under `internal/` and is not part of the
public API:

```
internal/core        canonical domain model + physical-record scanner:
                       Occurrence, ParseResult, RecordError,
                       OccurrenceHandler/RecordErrorHandler, Scanner, Record
                       (re-exported at root as type aliases)
internal/registry     the LayoutParser interface + the Parser that resolves
                        and delegates to one; imports only internal/core
                        (re-exported at root as type aliases)
internal/itau240        the built-in Itaú CNAB240 layout parser: all Itaú-
                         and CNAB240-specific knowledge (record grammar,
                         detail grouping, field extraction) lives here,
                         hidden from the public API
```

The root package re-exports `Occurrence`, `OccurrenceType`, `ParseResult`,
`RecordError`, `OccurrenceHandler`, `RecordErrorHandler`, `Scanner`,
`Record`, `LayoutParser`, and `Parser` from `internal/core`/`internal/registry`
via Go type aliases — they're the same types, just reachable without an
extra import. This is also what lets a custom, external `LayoutParser`
implementation satisfy the interface without importing anything beyond the
root package, and what lets `internal/itau240` satisfy `LayoutParser`
structurally without ever importing `internal/registry` or the root
package — keeping the whole module's internal dependency graph a one-way
DAG (root → `internal/registry`/`internal/itau240` → `internal/core`),
with no cycle possible.

## `NewParser`: explicit, always

```go
func NewParser(first LayoutParser, rest ...LayoutParser) *Parser
```

There is no default and no auto-loading — `NewParser()` doesn't compile;
at least one `LayoutParser` is required. Built-in layout parsers are
regular values you construct explicitly, just like custom ones, so they
compose freely in the same `Parser`:

```go
cnabpayment.NewParser(cnabpayment.NewItau240LayoutParser())

cnabpayment.NewParser(myCustomLayoutParser)

cnabpayment.NewParser(myCustomLayoutParser, cnabpayment.NewItau240LayoutParser())
```

## Architecture

```
io.Reader
    │
    ▼
Scanner            reads one physical record at a time
    │
    ▼
first Record
    │
    ▼
Parser             resolves a LayoutParser via Detect(first)
    │
    ▼
LayoutParser        takes over from the scanner's current position
    │
    ▼
Occurrences
```

- **Streaming, not buffering.** The scanner reads one physical record at a
  time from an `io.Reader`; nothing peeks, rewinds, or requires an
  `io.Seeker`. The first record is read exactly once, then handed to the
  resolved layout parser so it isn't read again.
- **Constant memory.** No file header, batch, record set, or trailer is
  held in memory as a whole; occurrences are emitted as soon as they're
  fully parsed.
- **Callback model.** `OccurrenceHandler` receives every occurrence as soon
  as it's ready; `RecordErrorHandler` receives recoverable, record-level
  errors. Returning an error from either callback aborts parsing. A `nil`
  `RecordErrorHandler` means such errors are ignored and parsing continues.
  `error` returned from `Parse` itself is reserved for fatal failures.
- **Extensible through layout parsers.** All bank-specific knowledge lives
  behind the `LayoutParser` interface. The root package and its `internal/`
  infrastructure contain no bank-specific logic.

## Public API

```go
package cnabpayment

func NewParser(first LayoutParser, rest ...LayoutParser) *Parser

func (p *Parser) Parse(
    r io.Reader,
    onOccurrence OccurrenceHandler,
    onRecordError RecordErrorHandler,
) (ParseResult, error)

type LayoutParser interface {
    Detect(first Record) bool

    Parse(
        s Scanner,
        first Record,
        onOccurrence OccurrenceHandler,
        onRecordError RecordErrorHandler,
    ) (ParseResult, error)
}
```

## Built-in layout parsers

### Itaú CNAB240

`NewItau240LayoutParser()` constructs it; the actual implementation lives
in `internal/itau240` and is not importable directly. It detects and
streams Itaú CNAB240 payment return files, extracting per logical payment
a `PaymentID` and an occurrence `RawType` from the
batch's mandatory segment (A for crédito em conta, J for boleto, or O for
boleto de outro banco), plus an optional `ConfirmationID` and `ExternalID`
from a Segment Z detail record when one follows it — `ConfirmationID` is a
bank-issued confirmation identifier, `ExternalID` a bank-generated
identifier, both empty when no Segment Z is present. `CreatedAt` is the
return file's own
generation date/time, taken from the file header (assumed to be Brasília
time, UTC-3) and attached to every occurrence in the file.

`Type` is a canonical `OccurrenceType` (`REJECTED`, `SCHEDULED`,
`SETTLED`, or `UNKNOWN`) derived from the first two characters of
`RawType` — e.g. `RJ` → `REJECTED`, `BD` → `SCHEDULED`, `00` → `SETTLED`.
Any other prefix maps to `UNKNOWN`; this mapping table, like all other
Itaú-specific knowledge, lives inside `internal/itau240` — not in the
public API.

`Format` is a `ReceiptFormat` lookup key, `"BBB-S-TT-FF"`: the bank
compensation code, the mandatory segment (`A`/`J`/`O`/`N`), and the batch
header's "Tipo de Serviço"/"Forma de Lançamento" codes (e.g.
`341-A-20-41` for a TED). It is populated on every occurrence, not just
settled ones — receipts only make sense for `SETTLED` occurrences, but the
key itself just describes the batch/segment. This library does not ship
receipt templates; `Format` only tells a caller which one of *its own*
templates applies.

Consecutive detail records that share a batch sequence number are treated
as one logical payment: each segment's fields are merged into the same
occurrence, and the group is only resolved once the sequence number
changes or the batch trailer is reached.

## License

Apache-2.0. See [LICENSE](LICENSE).
