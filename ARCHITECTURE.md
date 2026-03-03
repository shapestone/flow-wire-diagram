# Architecture

`flow-wire-diagram` uses a hexagonal (ports and adapters) architecture with three internal layers and a thin public facade.

## Layer Overview

```
wirediagram.go          ← public API facade (RepairFile, VerifyFile, ExtractBlocks, DetectWideChars)
internal/diagram/
  application/          ← use-case orchestration
  domain/               ← pure types, no I/O
  infrastructure/       ← parsing, repair, verification, ASCII conversion
```

### `domain/` — Pure types

`types.go` defines all shared data structures. No logic, no I/O.

| Type | Purpose |
|------|---------|
| `Box` | A rectangular region bounded by `┌┐└┘│─`, with `TopLine`, `BottomLine`, `LeftCol`, `RightCol`, `Parent`, `Children` |
| `LineRole` | `RoleFree`, `RoleTopFrame`, `RoleBottomFrame`, `RoleContent` |
| `DiagramLine` | One line of a diagram with its role, active boxes, and trailing text |
| `Block` | A contiguous section of Markdown — either `BlockDiagram` or `BlockPassthrough` |
| `VerifyError` | A post-repair defect with line number and description |

### `infrastructure/` — I/O and algorithms

| File | Responsibility |
|------|---------------|
| `extract.go` | Phase 1: scan Markdown for fenced blocks containing `┌` and `┐` |
| `parse.go` | Phase 2: detect box frames, build nesting tree, classify every line |
| `repair.go` | Phase 3: realign `│` chars, snap free-line connectors, preserve outside-box chars |
| `verify.go` | Phase 4: confirm width consistency, pipe alignment, connector alignment |
| `ascii.go` | Optional: convert box-drawing Unicode to `+`/`-`/`|` equivalents |
| `runeutil.go` | Unicode helpers wrapping `go-runewidth` (`RuneWidthOf`, `StringWidth`, `VisualPad`) |

### `application/` — Use-case orchestration

`service.go` owns the two public use cases:

- **`RepairFile`** — runs the full pipeline (Extract → Expand tabs → Parse → Repair → Safety check → ASCII conversion → Reconstruct) across all diagram blocks in a Markdown file
- **`VerifyFile`** — runs Extract → Parse → Verify without modifying anything

The content preservation safety check lives here: if `TextContent(original) != TextContent(repaired)` for any diagram, the repair is aborted for that diagram and a warning is emitted. This prevents silent data loss.

## Data Flow

```
Input bytes (Markdown)
        │
        ▼
  ExtractBlocks          extract.go     find fenced blocks with ┌ and ┐
        │
        ▼
   ExpandTabs            runeutil.go    tabs → 2 spaces
        │
        ▼
  ParseDiagram           parse.go       detect frames, build Box tree, classify lines
        │
        ▼
   RepairLines           repair.go      realign │, snap connectors, preserve outside chars
        │
        ▼
 TextContent check       service.go     abort if text content would change
        │
        ▼
  VerifyDiagram          verify.go      confirm no defects remain
        │
        ▼
ReconstructContent       extract.go     reassemble Markdown with repaired blocks
        │
        ▼
Output bytes (Markdown)
```

## Key Design Decisions

**Why require both `┌` and `┐` for detection?**
Tree diagrams (`├──`, `└──`) use `└` and `│` but never `┌` or `┐`. Requiring both top corners correctly excludes tree diagrams while matching every box diagram type.

**Why strict containment (`<` not `<=`) for the nesting tree?**
When a child box's right column equals the parent's right column, the containment check fails and both boxes become roots. Lines with multiple active root boxes pass through unchanged rather than being repaired incorrectly. This is a documented limitation — widen the outer box to fully contain the inner box to fix such diagrams.

**Why visual column rather than rune index?**
Unicode box-drawing characters are "Ambiguous" width under UAX #11, but render as 1 cell in Western locales (terminals, GitHub, VS Code). Under `go-runewidth` defaults (`EastAsianWidth=false`), visual column equals rune index for all diagram content. This allows `buf[col]` indexing, which simplifies the repair logic significantly.

**Why is the content safety check in `application/` rather than `infrastructure/`?**
The check is a policy decision (abort on mismatch), not a structural algorithm. Infrastructure returns the repaired lines; application decides whether to accept them.

**Why `connectorAlignWindow = 2`?**
A window of ±2 columns catches the most common off-by-one and off-by-two connector drift from LLM output while being narrow enough to avoid false positives when multiple connectors are close together. The constant is named for easy tuning.
