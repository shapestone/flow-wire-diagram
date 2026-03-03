# flow-wire-diagram

A CLI tool and Go library for detecting and repairing alignment defects in ASCII box diagrams embedded in Markdown files.

![Build Status](https://github.com/shapestone/flow-wire-diagram/actions/workflows/ci.yml/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/shapestone/flow-wire-diagram)](https://goreportcard.com/report/github.com/shapestone/flow-wire-diagram)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![codecov](https://codecov.io/gh/shapestone/flow-wire-diagram/branch/main/graph/badge.svg)](https://codecov.io/gh/shapestone/flow-wire-diagram)
![Go Version](https://img.shields.io/github/go-mod/go-version/shapestone/flow-wire-diagram)
![Latest Release](https://img.shields.io/github/v/release/shapestone/flow-wire-diagram)
[![GoDoc](https://pkg.go.dev/badge/github.com/shapestone/flow-wire-diagram.svg)](https://pkg.go.dev/github.com/shapestone/flow-wire-diagram)

[![CodeQL](https://github.com/shapestone/flow-wire-diagram/actions/workflows/codeql.yml/badge.svg)](https://github.com/shapestone/flow-wire-diagram/actions/workflows/codeql.yml)
[![OpenSSF Scorecard](https://api.securityscorecards.dev/projects/github.com/shapestone/flow-wire-diagram/badge)](https://securityscorecards.dev/viewer/?uri=github.com/shapestone/flow-wire-diagram)
[![Security Policy](https://img.shields.io/badge/Security-Policy-brightgreen)](SECURITY.md)

## The Problem

LLMs and humans produce broken ASCII box diagrams in Markdown. The most common defect is misaligned `│` characters, the right wall of a box drifts left or right across lines because content isn't padded to a fixed width.

Before — `│` characters drift, lines are different widths:

```
+----------------------+
|  +--------------+  |      <- outer | at wrong column
|  | Component A  |  |
|  | short      | |         <- inner | shifted left
|  +--------------+  |
+----------------------+
```

After — every `│` snapped to its correct column:

```ascii
┌──────────────────────┐
│  ┌──────────────┐    │
│  │ Component A  │    │
│  │ short        │    │
│  └──────────────┘    │
└──────────────────────┘
```

## Installation

```bash
go install github.com/shapestone/flow-wire-diagram/cmd/wire-fix@latest
```

Or build from source:

```bash
git clone https://github.com/shapestone/flow-wire-diagram.git
cd flow-wire-diagram
make build
# Binary: bin/wire-fix
```

## Usage

```bash
# Repair in-place
wire-fix docs/architecture.md

# Verify only (exit 1 if defects found)
wire-fix -v docs/architecture.md

# Preview changes without writing
wire-fix -d docs/architecture.md

# Write to a different file
wire-fix -o fixed.md docs/architecture.md

# Convert box-drawing Unicode to safe ASCII
wire-fix -a docs/architecture.md

# Scan for characters with visual width != 1
wire-fix -w docs/architecture.md
```

## Supported Diagram Types

| Type | Example |
|------|---------|
| Simple box | `┌──┐ / │ │ / └──┘` |
| Nested boxes | Box inside a box |
| Side-by-side | Sibling boxes within a parent |
| Multi-cell | Separate boxes connected by `│`/`▼` |
| Trailing text | `│ content │ - annotation` |

Tree diagrams (`├──`, `└──`) are detected and passed through unchanged.

## Library Usage

```go
import wirediagram "github.com/shapestone/flow-wire-diagram"

// Repair all diagrams in a Markdown file
output, result, err := wirediagram.RepairFile(input, wirediagram.Options{})

// Verify without modifying
result, err := wirediagram.VerifyFile(input)

// Detect wide characters (emoji, CJK) inside diagram blocks
blocks := wirediagram.ExtractBlocks(content)
for _, block := range blocks {
    for _, line := range block.Lines {
        wide := wirediagram.DetectWideChars(line)
    }
}
```

## How It Works

1. **Extract** — find fenced code blocks containing `┌` and `┐`
2. **Parse** — build a nesting tree of boxes with exact column positions
3. **Repair** — realign `│` characters to match the frame, snap free-line connectors
4. **Verify** — confirm no defects remain and no text content was altered
5. **Write** — reassemble Markdown, skip writing if unchanged

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | `-v` found defects |
| 2 | File read/write error |

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

Apache 2.0 — see [LICENSE](LICENSE).
