# cnab-payment

`cnab-payment` is a Go library for **CNAB payment files**: it streams
payment return files into a canonical, bank-agnostic stream of payment
occurrences, and writes payments into remittance files.

CNAB ("Centro Nacional de Automação Bancária") is a family of fixed-width
file formats used by Brazilian banks. Every bank implements its own layout,
which forces integrators to write bespoke code per bank. This library
exposes one small, stable API in both directions, with all bank-specific
knowledge behind pluggable layouts:

- **Reading** — a single streaming entry point reads a return file's first
  record, resolves the matching **layout parser**, and hands it the rest
  of the stream.
- **Writing** — a **generator** validates a list of payments and writes
  them into the remittance files a **layout generator** requires.

```go
import "github.com/cnab-dev/cnab-payment"

p := cnabpayment.NewParser(cnabpayment.NewItau240LayoutParser())

result, err := p.Parse(r, func(o cnabpayment.Occurrence) error {
    fmt.Println(o.PaymentID, o.Type, o.RawType, o.CreatedAt, o.Format)
    return nil
}, nil)
```

```go
g := cnabpayment.NewGenerator(cnabpayment.NewItau240LayoutGenerator())

files, err := g.Generate(payments, cnabpayment.GenerateOptions{})
for _, f := range files {
    save(f.Class, f.PaymentIDs, f.Content) // one file per bank-required partition
}
```

## Status

**Early, one bank in both directions.** The public API, the streaming and
writing infrastructure, and the Itaú CNAB240 layouts are in place: payment
return files (retorno) and SISPAG payment remittance files (remessa). Other
banks and CNAB400 are not supported yet.

## A single public package

`github.com/cnab-dev/cnab-payment` (package `cnabpayment`) is the **only**
importable package. Everything else — the domain model, the scanner, the
orchestration, and every built-in layout's actual implementation — lives
under `internal/` and is not part of the public API:

```
internal/core        canonical domain model + physical-record scanner:
                       Occurrence, ParseResult, RecordError, Scanner,
                       Record, Payment and its types, Amount,
                       RemittanceFile, PaymentError
                       (re-exported at root as type aliases)
internal/registry     the LayoutParser and LayoutGenerator interfaces, the
                        Parser that resolves and delegates to a layout
                        parser, and the Generator that validates, partitions
                        and sorts payments for a layout generator; imports
                        only internal/core (re-exported at root)
internal/cnab240      bank-agnostic CNAB240 writing engine: field
                        formatting, accent stripping, file and batch
                        assembly
internal/pix          Pix QR code ("copia e cola") payload parser
internal/taxid        CPF/CNPJ validation
internal/itau240      the built-in Itaú CNAB240 layouts: all Itaú-specific
                        knowledge (record grammar, detail grouping, field
                        extraction, remittance record layouts, validation)
                        lives here, hidden from the public API
```

The root package re-exports the types from `internal/core` and
`internal/registry` via Go type aliases — they're the same types, just
reachable without an extra import. This is also what lets a custom,
external `LayoutParser` or `LayoutGenerator` satisfy its interface without
importing anything beyond the root package, and what lets
`internal/itau240` satisfy both structurally without ever importing
`internal/registry` or the root package — keeping the module's internal
dependency graph a one-way DAG, with no cycle possible.

The module has no third-party dependencies.

## Explicit layouts, always

```go
func NewParser(first LayoutParser, rest ...LayoutParser) *Parser
func NewGenerator(layout LayoutGenerator) *Generator
```

There is no default and no auto-loading in either direction. Built-in
layouts are regular values you construct explicitly, just like custom ones.

A `Parser` takes one or more layout parsers and picks one per file, because
a return file identifies its own layout in its first record:

```go
cnabpayment.NewParser(cnabpayment.NewItau240LayoutParser())

cnabpayment.NewParser(myCustomLayoutParser, cnabpayment.NewItau240LayoutParser())
```

A `Generator` takes exactly one layout generator: nothing in a payment says
which layout to write it with, so the caller decides, and the generator
never infers one.

## Reading return files

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

## Writing remittance files

```
[]Payment
    │
    ▼
Generator          validates every payment (all or nothing),
    │                partitions them by Class, sorts each partition by ID
    ▼
LayoutGenerator    encodes one partition into one file
    │
    ▼
[]RemittanceFile   {Class, PaymentIDs, Content}
```

- **One layout per call.** Every payment must fit the generator's layout
  (e.g. its payer account must be at the layout's bank).
- **All or nothing.** If any payment is invalid, no file is produced, and
  the error joins one `*PaymentError` per problem, each naming the
  payment's index, ID and offending field and wrapping a sentinel:
  `ErrLayoutMismatch`, `ErrUnsupportedRail` or `ErrInvalidField`.
- **Validate ahead of time.** `Generator.Validate` checks a single payment,
  e.g. when it's created. A payment it accepts never makes `Generate` fail.
- **Bank-required partitions.** Some banks require certain payment methods
  in separate files. The layout assigns each payment a `RemittanceClass`
  that depends only on that payment, and each class becomes one file.
  Splitting payments for any other reason (per payer, per customer) is up
  to the caller.
- **Stable output.** The same payments always produce the same partitions,
  with payments in the same order; with the same non-zero
  `GenerateOptions.CreatedAt`, the files are byte-for-byte identical. A zero
  `CreatedAt` means now.
- **Explicit rail.** A `TransferPayment` carries its `Rail` (`tef`, `ted` or
  `pix`) and exactly one `Destination` (a `BankAccount` or a `PixKey`); the
  layout translates them and never infers either. A combination it can't
  write is `ErrUnsupportedRail`.
- **Amounts in cents.** `Amount` is an `int64` number of cents;
  `ParseAmount("123.45")` returns `12345` and rejects a third decimal place.
  Due and settlement dates are calendar dates, written as given.

## Public API

```go
package cnabpayment

// Reading
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

// Writing
func NewGenerator(layout LayoutGenerator) *Generator

func (g *Generator) Generate(payments []Payment, opts GenerateOptions) ([]RemittanceFile, error)
func (g *Generator) Validate(p Payment) error
func (g *Generator) NewPaymentID() (string, error)

type LayoutGenerator interface {
    Validate(p Payment) error
    Class(p Payment) RemittanceClass
    NewPaymentID() (string, error)
    Generate(payments []Payment, opts GenerateOptions) ([]byte, error)
}
```

`Payment` is a sealed interface implemented by `TransferPayment`,
`BarcodePayment` (boletos and collection barcodes) and `QRCodePayment`
(Pix "copia e cola"), all embedding `PaymentBase`. Its `ID` is written into
the remittance and comes back as `Occurrence.PaymentID` in the return
file; `NewPaymentID` produces one in the format the layout writes.

## Built-in layouts

### Itaú CNAB240: reading return files

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

### Itaú CNAB240: writing remittance files

`NewItau240LayoutGenerator()` writes Itaú SISPAG CNAB240 remittance files,
file layout 081 and batch layout 083. It is pinned to that version: a
future layout version will get its own constructor or option, never
silently replace this one.

| Payment | Batch (tipo/forma) | Segments |
|---|---|---|
| `tef` to an Itaú checking / savings / salary account | 20/01, 20/05, 30/01 | A |
| `ted` to an account at another bank | 20/41 | A |
| `pix` to a Pix key | 20/45 | A + B |
| `pix` to a checking / savings / payment account | 20/45 | A |
| Boleto issued by Itaú / another bank | 20/30, 20/31 | J + J-52 |
| DARF (DCTFWeb) collection barcode | 22/91 | O |
| Pix QR code | 20/47 | J + J-52 PIX |

Any other rail and destination combination — e.g. `ted` to an Itaú
account (use `tef`), or `pix` to a salary account — is
`ErrUnsupportedRail`.

- **Classes:** `"pix"` (Pix transfers and Pix QR codes, which the bank
  requires in a file of their own) and `"other"`.
- **Payment IDs:** 20 random uppercase alphanumeric characters.
- **Not yet validated with the bank:** Pix to a bank account, and the
  account format for other banks with more than 6 significant digits,
  follow the manual but haven't been confirmed against the bank yet.

The expected files in `internal/itau240/testdata/remittance` (fictitious
data) are byte-for-byte golden tests, and a round-trip test parses each
generated remittance back as a return file.

## License

Apache-2.0. See [LICENSE](LICENSE).
