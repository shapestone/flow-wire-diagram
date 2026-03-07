# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.3.0] - 2026-03-06

### Added
- `--fix <dir>` flag: recursively scan a directory for `.md` files and repair diagram defects in-place; reports `FIXED`, `PASS`, or `SKIP` per file

## [0.2.0] - 2026-03-06

### Added
- `--scan <dir>` flag: recursively scan a directory for `.md` files and report diagram defects (read-only, no files modified)
- `--version` flag: print the version string and exit
- Version injected at build time via `ldflags` from the git tag (`make build` / `make install`)

### Changed
- Long flags now follow Unix/POSIX convention: `--verbose`, `--scan`, `--version` (double dash); single-character flags unchanged (`-v`, `-a`, `-d`, `-o`, `-w`)

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

[Unreleased]: https://github.com/shapestone/flow-wire-diagram/compare/v0.3.0...HEAD
[0.3.0]: https://github.com/shapestone/flow-wire-diagram/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/shapestone/flow-wire-diagram/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/shapestone/flow-wire-diagram/releases/tag/v0.1.0
