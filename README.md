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

# Show per-diagram repair summary
wire-fix --verbose docs/architecture.md

# Recursively scan a directory and report defects (read-only)
wire-fix --scan docs/

# Recursively scan a directory and repair defects in-place
wire-fix --fix docs/

# Print version
wire-fix --version
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

## Known Limitations

- **Strict containment**: When a child box's right column exactly equals the parent box's right column, the nesting check fails and both boxes become roots — the diagram passes through unrepaired. **Fix**: widen the outer box by at least one column so it fully contains the inner box.
- **Connector snap window**: Free-line connectors (`│` on non-frame lines) are only snapped when they are within ±2 columns of the expected position. Connectors drifted more than 2 columns will not be repaired.

## Library Usage

### Install

```bash
go get github.com/shapestone/flow-wire-diagram@latest
```

### Repair a file

```go
import (
    "fmt"
    "log"
    "os"
    wirediagram "github.com/shapestone/flow-wire-diagram"
)

input, err := os.ReadFile("docs/architecture.md")
if err != nil {
    log.Fatal(err)
}

output, result, err := wirediagram.RepairFile(input, wirediagram.Options{})
if err != nil {
    log.Fatal(err)
}

// result.DiagramsFound    — total diagrams parsed from the file
// result.DiagramsRepaired — diagrams that had alignment defects (now fixed)
// result.DiagramsOK       — diagrams that were already correct
// result.Warnings         — human-readable defect descriptions

if result.DiagramsRepaired > 0 {
    if err := os.WriteFile("docs/architecture.md", output, 0644); err != nil {
        log.Fatal(err)
    }
    fmt.Printf("repaired %d of %d diagrams\n", result.DiagramsRepaired, result.DiagramsFound)
}
```

### Verify without modifying

```go
result, err := wirediagram.VerifyFile(input)
if err != nil {
    log.Fatal(err)
}
if result.DiagramsRepaired > 0 {
    for _, w := range result.Warnings {
        fmt.Println(w)
    }
}
```

### Terminal-safe ASCII output

Pass `Options{ASCII: true}` to convert box-drawing Unicode (`┌│─┘`) to plain ASCII (`+|-`) in the output:

```go
output, result, err := wirediagram.RepairFile(input, wirediagram.Options{ASCII: true})
```

### Detect wide characters

```go
blocks := wirediagram.ExtractBlocks(string(input))
for _, block := range blocks {
    if block.Kind != wirediagram.BlockDiagram {
        continue
    }
    for _, line := range block.Lines {
        if wide := wirediagram.DetectWideChars(line); len(wide) > 0 {
            fmt.Printf("wide chars in: %q\n", line)
        }
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
| 1 | `-v` or `--scan` found defects |
| 2 | File read/write error |

## Troubleshooting

| Symptom | Cause | Resolution |
|---------|-------|------------|
| File unchanged after running `wire-fix` | No defects detected, or diagram uses `+`/`-`/`\|` (ASCII) instead of `┌`/`│` box-drawing chars | Run `-d` to preview; if no diff, run `-v` to see defect count. If using ASCII, omit `-a`. |
| `-v` exits 1 on a diagram that looks correct | A connector is misaligned by ≤2 columns but the frame is fine | Run `-d` to preview the exact repair |
| `--scan` reports a file as FAIL | The file has at least one diagram with defects | Run `wire-fix -v <file>` for detail, or `wire-fix <file>` to repair |
| `--fix` leaves a file unchanged | The file had no defects | Expected — it will show as PASS |
| Wide characters reported by `-w` | Emoji or CJK glyphs inside diagram blocks have visual width 2 | Remove wide characters from inside boxes, or annotate them outside |

## CI/CD Integration

GitHub Actions step:

```yaml
- name: Verify wire diagrams
  run: wire-fix -v docs/architecture.md
```

Pre-commit hook one-liner:

```bash
wire-fix -v "$1" || { echo "Diagram defects found — run wire-fix to repair"; exit 1; }
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

Apache 2.0 — see [LICENSE](LICENSE).
