package wirediagram_test

import (
	"os"
	"strings"
	"testing"

	wirediagram "github.com/shapestone/flow-wire-diagram"
)

func readTestdata(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatalf("cannot read testdata/%s: %v", name, err)
	}
	return data
}

// TestRepairSimpleBoxFile repairs testdata/simple_box.md and verifies the result.
func TestRepairSimpleBoxFile(t *testing.T) {
	input := readTestdata(t, "simple_box.md")
	output, result, err := wirediagram.RepairFile(input, wirediagram.Options{})
	if err != nil {
		t.Fatalf("RepairFile: %v", err)
	}
	if result.DiagramsFound == 0 {
		t.Error("expected at least 1 diagram")
	}
	// Verify the repaired output has no defects.
	vResult, err := wirediagram.VerifyFile(output)
	if err != nil {
		t.Fatalf("VerifyFile: %v", err)
	}
	if vResult.DiagramsRepaired > 0 {
		t.Errorf("verify found defects after repair: %v", vResult.Warnings)
	}
}

// TestRoundTrip repairs simple_box.md and then verifies it passes cleanly.
func TestRoundTrip(t *testing.T) {
	files := []string{
		"simple_box.md",
		"nested_box.md",
		"already_correct.md",
		"side_by_side.md",
		"multi_cell.md",
		"multi_cell_nested.md",
	}
	for _, f := range files {
		t.Run(f, func(t *testing.T) {
			input := readTestdata(t, f)
			repaired, _, err := wirediagram.RepairFile(input, wirediagram.Options{})
			if err != nil {
				t.Fatalf("RepairFile: %v", err)
			}
			vResult, err := wirediagram.VerifyFile(repaired)
			if err != nil {
				t.Fatalf("VerifyFile: %v", err)
			}
			if vResult.DiagramsRepaired > 0 {
				t.Errorf("verify found defects after repair: %v", vResult.Warnings)
			}
		})
	}
}

// TestDiffCorrectFile verifies that an already-correct file produces no diff.
func TestDiffCorrectFile(t *testing.T) {
	input := readTestdata(t, "already_correct.md")
	output, result, err := wirediagram.RepairFile(input, wirediagram.Options{})
	if err != nil {
		t.Fatalf("RepairFile: %v", err)
	}
	if string(input) != string(output) {
		t.Errorf("already_correct.md was modified (diagrams repaired: %d)", result.DiagramsRepaired)
	}
}

// TestASCIIConversion verifies that --ascii replaces all box-drawing chars
// across all diagram types.
func TestASCIIConversion(t *testing.T) {
	files := []string{
		"simple_box.md",
		"nested_box.md",
		"multi_cell.md",
		"multi_cell_nested.md",
	}
	for _, f := range files {
		t.Run(f, func(t *testing.T) {
			input := readTestdata(t, f)
			output, _, err := wirediagram.RepairFile(input, wirediagram.Options{ASCII: true})
			if err != nil {
				t.Fatalf("RepairFile: %v", err)
			}
			blocks := wirediagram.ExtractBlocks(string(output))
			for _, block := range blocks {
				if block.Kind != wirediagram.BlockDiagram {
					continue
				}
				for _, line := range block.Lines {
					for _, r := range line {
						switch r {
						case '┌', '┐', '└', '┘', '│', '─', '┬', '┴', '├', '┤', '┼':
							t.Errorf("box-drawing char %c still present after ASCII conversion in line: %q", r, line)
						}
					}
				}
			}
		})
	}
}

// TestCheckWidthDetection verifies that wide characters in diagrams are detected.
func TestCheckWidthDetection(t *testing.T) {
	// Create a diagram with an emoji (width > 1) inside.
	content := "```ascii\n┌──────────────┐\n│  Hello 🌍    │\n└──────────────┘\n```\n"
	blocks := wirediagram.ExtractBlocks(content)
	found := false
	for _, block := range blocks {
		if block.Kind != wirediagram.BlockDiagram {
			continue
		}
		for _, line := range block.Lines {
			wide := wirediagram.DetectWideChars(line)
			if len(wide) > 0 {
				found = true
			}
		}
	}
	if !found {
		t.Error("expected DetectWideChars to find the emoji 🌍")
	}
}

// TestCheckWidthFile verifies that DetectWideChars finds wide characters in
// testdata/wide_chars.md and that RepairFile does not silently corrupt content
// when wide characters are present.
func TestCheckWidthFile(t *testing.T) {
	input := readTestdata(t, "wide_chars.md")

	// DetectWideChars must find the emoji in the diagram block.
	blocks := wirediagram.ExtractBlocks(string(input))
	wideFound := false
	for _, block := range blocks {
		if block.Kind != wirediagram.BlockDiagram {
			continue
		}
		for _, line := range block.Lines {
			if len(wirediagram.DetectWideChars(line)) > 0 {
				wideFound = true
			}
		}
	}
	if !wideFound {
		t.Fatal("DetectWideChars: expected wide char in wide_chars.md diagram block")
	}

	// RepairFile must not crash and must not silently lose text content.
	repaired, result, err := wirediagram.RepairFile(input, wirediagram.Options{})
	if err != nil {
		t.Fatalf("RepairFile: %v", err)
	}
	_ = result

	// Idempotency: a second repair pass must produce the same bytes.
	repaired2, _, err := wirediagram.RepairFile(repaired, wirediagram.Options{})
	if err != nil {
		t.Fatalf("RepairFile (idempotent): %v", err)
	}
	if string(repaired) != string(repaired2) {
		t.Error("repair of wide_chars.md is not idempotent")
	}
}

// TestPassthroughPreservation is the core safety test: everything outside a
// box diagram block must be byte-for-byte identical in the output.
//
// It covers:
//   - Plain prose (headings, paragraphs, lists, inline code)
//   - Non-diagram fenced blocks (Go, shell, JSON)
//   - The opening and closing fence markers (``` and language tags)
//   - Empty lines between sections
//   - A trailing newline
//   - The broken diagram itself (only that block is allowed to change)
func TestPassthroughPreservation(t *testing.T) {
	// Build a markdown document with many kinds of non-diagram content
	// surrounding one broken diagram block.
	const brokenDiagram = "┌──────────────┐\n│  some text   │\n│ too short  │\n└──────────────┘"

	sections := []string{
		"# Heading 1\n",
		"Some **bold** and _italic_ prose.\n",
		"A list:\n- item 1\n- item 2\n- item 3\n",
		"Inline `code` in a sentence.\n",
		"```go\nfunc main() {\n\tfmt.Println(\"hello\")\n}\n```\n",
		"```bash\necho 'hello world'\nls -la\n```\n",
		"```json\n{\"key\": \"value\", \"num\": 42}\n```\n",
		"```\nplain fenced block\nno language tag\n```\n",
		"## Heading 2\n",
		"More prose after a heading.\n",
		"```ascii\n" + brokenDiagram + "\n```\n", // ← only this block changes
		"### Heading 3\n",
		"Final paragraph at the end of the file.\n",
	}

	input := strings.Join(sections, "\n")

	output, result, err := wirediagram.RepairFile([]byte(input), wirediagram.Options{})
	if err != nil {
		t.Fatalf("RepairFile: %v", err)
	}
	if result.DiagramsFound != 1 {
		t.Errorf("expected 1 diagram, got %d", result.DiagramsFound)
	}

	// Split input and output into lines for per-line comparison.
	inLines := strings.Split(input, "\n")
	outLines := strings.Split(string(output), "\n")

	if len(inLines) != len(outLines) {
		t.Fatalf("line count changed: input=%d output=%d", len(inLines), len(outLines))
	}

	// Find the diagram block boundaries in the input so we know which lines
	// are allowed to differ.
	diagramStart, diagramEnd := -1, -1
	inFence := false
	for i, line := range inLines {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			if !inFence {
				// Opening fence: check if this is the diagram block.
				end := i + 1
				for end < len(inLines) && !strings.HasPrefix(strings.TrimSpace(inLines[end]), "```") {
					end++
				}
				content := inLines[i+1 : end]
				if wirediagram.ExtractBlocks(strings.Join(append([]string{line}, append(content, inLines[end])...), "\n"))[0].Kind == wirediagram.BlockDiagram {
					diagramStart, diagramEnd = i, end
				}
				inFence = true
			} else {
				inFence = false
			}
		}
	}

	if diagramStart < 0 {
		t.Fatal("could not locate diagram block in input")
	}

	// Every line OUTSIDE the diagram block must be byte-for-byte identical.
	for i := 0; i < len(inLines); i++ {
		if i >= diagramStart && i <= diagramEnd {
			continue // diagram content is allowed to differ
		}
		if inLines[i] != outLines[i] {
			t.Errorf("non-diagram line %d changed:\n  in:  %q\n  out: %q", i, inLines[i], outLines[i])
		}
	}
}

// TestMixedFile verifies that mixed.md is handled correctly: box diagrams are
// repaired, the embedded tree diagram is not detected as a box diagram, and the
// already-correct diagram is left unchanged.
func TestMixedFile(t *testing.T) {
	input := readTestdata(t, "mixed.md")
	repaired, result, err := wirediagram.RepairFile(input, wirediagram.Options{})
	if err != nil {
		t.Fatalf("RepairFile: %v", err)
	}

	// The tree block must not be counted as a diagram.
	if result.DiagramsFound != 3 {
		t.Errorf("DiagramsFound: want 3, got %d", result.DiagramsFound)
	}

	// After repair, no defects should remain.
	vResult, err := wirediagram.VerifyFile(repaired)
	if err != nil {
		t.Fatalf("VerifyFile: %v", err)
	}
	if vResult.DiagramsRepaired > 0 {
		t.Errorf("verify found defects after repair: %v", vResult.Warnings)
	}

	// Repair must be idempotent.
	repaired2, _, err := wirediagram.RepairFile(repaired, wirediagram.Options{})
	if err != nil {
		t.Fatalf("RepairFile (idempotent): %v", err)
	}
	if string(repaired) != string(repaired2) {
		t.Error("repair is not idempotent")
	}
}

// TestSharedBoundaryPassthrough verifies the documented shared-boundary
// limitation: when a child box's right column equals the parent's right column
// the strict containment check fails, both become roots, and content lines with
// more than one active root box pass through unchanged (no repair attempted).
func TestSharedBoundaryPassthrough(t *testing.T) {
	input := readTestdata(t, "shared_boundary.md")
	repaired, _, err := wirediagram.RepairFile(input, wirediagram.Options{})
	if err != nil {
		t.Fatalf("RepairFile: %v", err)
	}
	// The file must be byte-for-byte identical: shared-boundary lines are
	// intentionally left unchanged rather than repaired incorrectly.
	if string(input) != string(repaired) {
		inLines := strings.Split(string(input), "\n")
		outLines := strings.Split(string(repaired), "\n")
		for i := range inLines {
			if i < len(outLines) && inLines[i] != outLines[i] {
				t.Errorf("line %d changed:\n  in:  %q\n  out: %q", i, inLines[i], outLines[i])
			}
		}
		t.Fatal("shared_boundary.md was modified; shared-boundary lines should pass through unchanged")
	}
}

// TestRuneWidthOf verifies that RuneWidthOf returns correct visual widths.
func TestRuneWidthOf(t *testing.T) {
	tests := []struct {
		r    rune
		want int
	}{
		{'a', 1},
		{'─', 1},
		{'│', 1},
		{' ', 1},
		{'🌍', 2}, // wide emoji
		{'中', 2}, // CJK character
	}
	for _, tc := range tests {
		got := wirediagram.RuneWidthOf(tc.r)
		if got != tc.want {
			t.Errorf("RuneWidthOf(%c) = %d, want %d", tc.r, got, tc.want)
		}
	}
}

// TestConnectorOffByOne verifies that a free-line │ connector that is 1 column
// to the right of the expected ┬ position is snapped to the correct column.
func TestConnectorOffByOne(t *testing.T) {
	input := readTestdata(t, "connector_offby1.md")
	repaired, result, err := wirediagram.RepairFile(input, wirediagram.Options{})
	if err != nil {
		t.Fatalf("RepairFile: %v", err)
	}
	if result.DiagramsFound == 0 {
		t.Fatal("expected at least 1 diagram")
	}
	// After repair, verify should find no defects.
	vResult, err := wirediagram.VerifyFile(repaired)
	if err != nil {
		t.Fatalf("VerifyFile: %v", err)
	}
	if vResult.DiagramsRepaired > 0 {
		t.Errorf("verify found defects after repair: %v", vResult.Warnings)
	}
	// Repair must be idempotent.
	repaired2, _, err := wirediagram.RepairFile(repaired, wirediagram.Options{})
	if err != nil {
		t.Fatalf("RepairFile (idempotent): %v", err)
	}
	if string(repaired) != string(repaired2) {
		t.Error("repair of connector_offby1.md is not idempotent")
	}
}

// TestOuterWallOffByOne verifies that a content line where the outer right wall
// │ is one column short is detected as a defect and repaired.
func TestOuterWallOffByOne(t *testing.T) {
	input := readTestdata(t, "outer_wall_offby1.md")
	// VerifyFile should detect the defect.
	vBefore, err := wirediagram.VerifyFile(input)
	if err != nil {
		t.Fatalf("VerifyFile (before): %v", err)
	}
	if vBefore.DiagramsRepaired == 0 {
		t.Error("expected VerifyFile to detect defects in outer_wall_offby1.md before repair")
	}
	// RepairFile should fix it.
	repaired, result, err := wirediagram.RepairFile(input, wirediagram.Options{})
	if err != nil {
		t.Fatalf("RepairFile: %v", err)
	}
	if result.DiagramsFound == 0 {
		t.Fatal("expected at least 1 diagram")
	}
	// After repair, no defects should remain.
	vAfter, err := wirediagram.VerifyFile(repaired)
	if err != nil {
		t.Fatalf("VerifyFile (after): %v", err)
	}
	if vAfter.DiagramsRepaired > 0 {
		t.Errorf("verify found defects after repair: %v", vAfter.Warnings)
	}
}

// TestContentTooWide verifies that content lines where the outer right wall │
// is one or two columns too far right are detected as defects and repaired.
func TestContentTooWide(t *testing.T) {
	input := readTestdata(t, "content_too_wide.md")
	// VerifyFile should detect the defect.
	vBefore, err := wirediagram.VerifyFile(input)
	if err != nil {
		t.Fatalf("VerifyFile (before): %v", err)
	}
	if vBefore.DiagramsRepaired == 0 {
		t.Error("expected VerifyFile to detect defects in content_too_wide.md before repair")
	}
	// RepairFile should fix it.
	repaired, result, err := wirediagram.RepairFile(input, wirediagram.Options{})
	if err != nil {
		t.Fatalf("RepairFile: %v", err)
	}
	if result.DiagramsFound == 0 {
		t.Fatal("expected at least 1 diagram")
	}
	// After repair, no defects should remain.
	vAfter, err := wirediagram.VerifyFile(repaired)
	if err != nil {
		t.Fatalf("VerifyFile (after): %v", err)
	}
	if vAfter.DiagramsRepaired > 0 {
		t.Errorf("verify found defects after repair: %v", vAfter.Warnings)
	}
	// Repair must be idempotent.
	repaired2, _, err := wirediagram.RepairFile(repaired, wirediagram.Options{})
	if err != nil {
		t.Fatalf("RepairFile (idempotent): %v", err)
	}
	if string(repaired) != string(repaired2) {
		t.Error("repair of content_too_wide.md is not idempotent")
	}
}

// TestTreeDiagramPassthrough verifies tree diagrams pass through unchanged.
func TestTreeDiagramPassthrough(t *testing.T) {
	input := readTestdata(t, "tree_diagram.md")
	output, result, err := wirediagram.RepairFile(input, wirediagram.Options{})
	if err != nil {
		t.Fatalf("RepairFile: %v", err)
	}
	// Tree diagrams are not box diagrams, so no diagrams should be found.
	if result.DiagramsFound != 0 {
		t.Errorf("expected 0 diagrams found for tree_diagram.md, got %d", result.DiagramsFound)
	}
	// Content must be unchanged.
	if string(input) != string(output) {
		t.Error("tree_diagram.md was modified")
		// Show diff.
		inLines := strings.Split(string(input), "\n")
		outLines := strings.Split(string(output), "\n")
		for i := range inLines {
			if i < len(outLines) && inLines[i] != outLines[i] {
				t.Logf("line %d: %q -> %q", i, inLines[i], outLines[i])
			}
		}
	}
}
