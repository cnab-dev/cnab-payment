# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
