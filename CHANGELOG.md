# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.5.2] - 2026-03-07

### Fixed
- `RepairLines`: when a content line's text fills right up to the wrong right wall (no trailing-space slack), the outermost box's `RightCol` is now widened to fit the content and all lines from the box's top frame down to the current line are retroactively re-rendered with the new width. Previously those lines were left unchanged.

### Changed
- Removed "content lines too wide with no trailing space" from Known Limitations (the widening strategy now handles it fully)

## [0.5.1] - 2026-03-06

### Fixed
- `repairContentLine`: when a content line is wider than the target box and the text fills the segment with no trailing space, the line is now left unchanged instead of silently truncating the last character. Lines that do have trailing-space slack are still repaired normally.

### Added
- `TestRepairContentLineTooWideNoSlack`: unit test that exercises the no-trailing-space edge case
- `TestContentTooWideNoSlack`: integration test confirming no characters are dropped from the repaired output

## [0.5.0] - 2026-03-06

### Added
- Test fixture: `content_too_wide.md` — box where content lines have the outer right wall 1 or 2 columns too far right
- `TestContentTooWide`: verifies VerifyFile detects the defect, RepairFile fixes it, and repair is idempotent

## [0.4.0] - 2026-03-06

### Added
- `--scan` and `--fix` now skip common non-content directories: `node_modules`, `.git`, `vendor`, `.cache`, `dist`, `.next`, `.nuxt`
- Test fixtures: `connector_offby1.md` (free-line connector off by 1 col), `outer_wall_offby1.md` (outer right wall short by 1 col)

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

[Unreleased]: https://github.com/shapestone/flow-wire-diagram/compare/v0.5.2...HEAD
[0.5.2]: https://github.com/shapestone/flow-wire-diagram/compare/v0.5.1...v0.5.2
[0.5.1]: https://github.com/shapestone/flow-wire-diagram/compare/v0.5.0...v0.5.1
[0.5.0]: https://github.com/shapestone/flow-wire-diagram/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/shapestone/flow-wire-diagram/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/shapestone/flow-wire-diagram/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/shapestone/flow-wire-diagram/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/shapestone/flow-wire-diagram/releases/tag/v0.1.0
