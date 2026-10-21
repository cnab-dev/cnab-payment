# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.4.0] - 2026-10-20

### Added

- Remittance file generation. `NewGenerator(layout)` returns a `Generator`
  whose `Generate` validates every payment (a single invalid payment fails
  the whole call), partitions them by the layout's `RemittanceClass`,
  sorts each partition by ID and returns one `RemittanceFile` per class.
  `Validate` checks a single payment ahead of time, and `NewPaymentID`
  returns an ID in the layout's format.
- `LayoutGenerator`, the extension point for remittance layouts.
- `NewItau240LayoutGenerator`, writing Itaú SISPAG CNAB240 remittances
  (file layout 081, batch layout 083): transfers by `tef`, `ted` and `pix`
  (to a Pix key or a bank account), boletos, DARF (DCTFWeb) collection
  barcodes and Pix QR codes, with Pix in a separate file.
- The payment model: `Payment` (sealed), `PaymentBase`, `TransferPayment`,
  `BarcodePayment`, `QRCodePayment`, `Rail`, `Destination` (`BankAccount`
  or `PixKey`), `AccountType`, `PixKeyType`, `Party`, `Address`, `Amount`
  (cents) and `ParseAmount`.
- `PaymentError` and the sentinels `ErrNoPayments`, `ErrLayoutMismatch`,
  `ErrUnsupportedRail` and `ErrInvalidField`.

### Changed

- The Itaú CNAB240 return parser ignores segment J-52 when extracting
  occurrences. Defensive: J-52 has no payment ID or occurrence fields, and
  the bank isn't known to echo it in return files.

## [0.3.0] - 2026-07-25

### Added

- `Occurrence.ExternalID`, a bank-generated identifier read from Segment Z
  (columns 104-118), empty when no Segment Z is present.

## [0.2.0] - 2026-07-25

### Added

- `Occurrence.Format`, a `ReceiptFormat` lookup key (`"BBB-S-TT-FF"`: bank
  compensation code, batch segment, "Tipo de Serviço", "Forma de
  Lançamento") that callers can use to pick which receipt template applies
  to a settled payment. The library does not ship template content.

## [0.1.1] - 2026-07-23

### Removed

- Unused indirect `github.com/google/go-cmp` dependency.
- Roadmap section from README.

## [0.1.0] - 2026-07-21

### Added

- Initial release with Itaú CNAB 240 parser.
