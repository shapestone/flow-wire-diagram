# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2026-03-03

### Added
- CLI tool `wire-fix` with `-v`, `-d`, `-o`, `-a`, `-w`, `-verbose` flags
- Go library (`wirediagram` package) with `RepairFile`, `VerifyFile`, `ExtractBlocks`, `DetectWideChars`
- Hexagonal architecture: domain / application / infrastructure layers
- Support for simple, nested, side-by-side, and multi-cell (connected) box diagrams
- Trailing text preservation — text after the right box edge is kept unchanged
- Tab expansion before parsing (tabs → 2 spaces)
- Free-line connector alignment: snaps `│`/`▼` to nearest source char within ±2 columns
- Outside-box connector preservation: `│` columns running alongside a box survive repair
- Content preservation safety check: repair aborts if text content would be altered
- Property-based tests covering all diagram types with 50+ mangled cases
- Go fuzz test (`FuzzRepair`) seeded with all diagram types
- Tree diagram passthrough (diagrams using `├──`/`└──` are not modified)
- Wide character detection (`DetectWideChars`, `-w` flag) for emoji and CJK characters
- ASCII conversion mode (`-a` flag) for terminals that don't render box-drawing Unicode

[Unreleased]: https://github.com/shapestone/flow-wire-diagram/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/shapestone/flow-wire-diagram/releases/tag/v0.1.0
