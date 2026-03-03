package infrastructure

import (
	"strings"
	"testing"
)

// repairDiagram is a helper: parse + repair a slice of diagram lines.
func repairDiagram(t *testing.T, lines []string) []string {
	t.Helper()
	_, diagLines, err := ParseDiagram(lines)
	if err != nil {
		t.Fatalf("ParseDiagram: %v", err)
	}
	repaired, err := RepairLines(diagLines, nil)
	if err != nil {
		t.Fatalf("RepairLines: %v", err)
	}
	return repaired
}

func TestRepairSingleBox(t *testing.T) {
	lines := []string{
		"┌──────────────┐", // frame: RightCol=15, width=16
		"│ too short   │",  // broken: │ at col 14 instead of 15
		"└──────────────┘",
	}
	repaired := repairDiagram(t, lines)

	got := repaired[1]
	if StringWidth(got) != 16 {
		t.Errorf("repaired width: want 16, got %d  (line: %q)", StringWidth(got), got)
	}
	if !strings.Contains(got, "too short") {
		t.Errorf("content lost: got %q, want to contain 'too short'", got)
	}
	runes := []rune(got)
	if runes[0] != '│' {
		t.Errorf("left pipe: got %c", runes[0])
	}
	if runes[15] != '│' {
		t.Errorf("right pipe: got %c at col 15", runes[15])
	}
}

func TestRepairNestedBox(t *testing.T) {
	lines := []string{
		"┌────────────────────┐",  // line 0: outer top (22)
		"│  ┌──────────────┐  │", // line 1: inner top (22)
		"│  │ inner box     │  │", // line 2: inner content (22)
		"│  │ inner short │ │",   // line 3: broken (20)
		"│  └──────────────┘  │", // line 4: inner bottom (22)
		"└────────────────────┘",  // line 5: outer bottom (22)
	}
	repaired := repairDiagram(t, lines)

	got := repaired[3]
	w := StringWidth(got)
	if w != 22 {
		t.Errorf("repaired line 3 width: want 22, got %d  (line: %q)", w, got)
	}
	runes := []rune(got)
	for _, col := range []int{0, 3, 18, 21} {
		if col >= len(runes) || runes[col] != '│' {
			t.Errorf("expected │ at col %d, got %c (line: %q)", col, safeRune(runes, col), got)
		}
	}
}

func TestRepairSideBySide(t *testing.T) {
	lines := []string{
		"┌──────────────────────┐",  // outer top (24)
		"│  ┌──────┐  ┌──────┐  │", // inner tops (24)
		"│  │  A   │  │  B   │  │", // content (24) -- correct
		"│  │  A2  │  │  B2  │ │",  // content broken (23)
		"│  └──────┘  └──────┘  │", // inner bottoms (24)
		"└──────────────────────┘",  // outer bottom (24)
	}
	repaired := repairDiagram(t, lines)
	got := repaired[3]
	w := StringWidth(got)
	if w != 24 {
		t.Errorf("repaired line 3 width: want 24, got %d  (line: %q)", w, got)
	}
}

func TestRepairTrailingText(t *testing.T) {
	lines := []string{
		"┌──────────────┐",
		"│ API Layer    │ - HTTP handlers",
		"└──────────────┘",
	}
	repaired := repairDiagram(t, lines)

	got := repaired[1]
	if !strings.Contains(got, "API Layer") {
		t.Errorf("content lost: %q", got)
	}
	if !strings.Contains(got, "- HTTP handlers") {
		t.Errorf("trailing text lost: %q", got)
	}
}

func TestRepairNoOp(t *testing.T) {
	lines := []string{
		"┌──────────────┐",
		"│  some text   │",
		"│  more text   │",
		"└──────────────┘",
	}
	repaired := repairDiagram(t, lines)
	for i, orig := range lines {
		if repaired[i] != orig {
			t.Errorf("line %d changed:\n  orig: %q\n   got: %q", i, orig, repaired[i])
		}
	}
}

func TestRepairContentPreservation(t *testing.T) {
	lines := []string{
		"┌──────────────┐",
		"│ Hello World │",
		"└──────────────┘",
	}
	repaired := repairDiagram(t, lines)
	got := repaired[1]

	origContent := strings.Trim(strings.ReplaceAll(lines[1], "│", ""), " ")
	gotContent := strings.Trim(strings.ReplaceAll(got, "│", ""), " ")
	if origContent != gotContent {
		t.Errorf("content changed:\n  orig: %q\n   got: %q", origContent, gotContent)
	}
}

// TestRepairPreservesOutsideBoxConnector verifies that a │ running alongside a
// box (at a column outside the box's [LeftCol, RightCol] span) is preserved on
// every line after repair — even when the content line itself is misaligned.
func TestRepairPreservesOutsideBoxConnector(t *testing.T) {
	// connCol=2, pad=4, innerWidth=12 → box at cols 7-20, │ at col 2.
	base := buildBoxWithSideConnector(2, 4, 12, []string{" row one    ", " row two    "})

	// Mangle both content lines by shortening them, forcing repair to rebuild.
	mangled := mangleContentShort(base, 1, 2)
	mangled = mangleContentShort(mangled, 2, 1)

	repaired := repairDiagram(t, mangled)

	// The │ at col 2 must survive on every line (frame and content alike).
	for i, line := range repaired {
		col := 0
		found := false
		for _, r := range line {
			if col == 2 {
				if r != '│' {
					t.Errorf("line %d: expected │ at col 2 after repair, got %c (line: %q)", i, r, line)
				}
				found = true
				break
			}
			col += RuneWidthOf(r)
		}
		if !found {
			t.Errorf("line %d: line too short to have │ at col 2 (line: %q)", i, line)
		}
	}
}

// TestRepairConnectorMisalignment verifies that a │ on a free (between-boxes)
// line that is shifted by 1 column relative to the ┬ on the frame above is
// moved back to the correct column during repair.
func TestRepairConnectorMisalignment(t *testing.T) {
	base := buildConnectedBoxes(14, 6, []string{" Box A        "}, []string{" Box B        "})
	// Line 3 is the free connector line with │ at col 7.
	// Shift │ right by 1 → col 8.
	base[3] = "        │" // 8 spaces + │

	repaired := repairDiagram(t, base)

	// The connector │ should be back at col 7 after repair.
	got := repaired[3]
	col := 0
	found := false
	for _, r := range got {
		if r == '│' {
			if col != 7 {
				t.Errorf("connector │ at col %d after repair, want col 7 (line: %q)", col, got)
			}
			found = true
			break
		}
		col += RuneWidthOf(r)
	}
	if !found {
		t.Errorf("connector │ not found in repaired line: %q", got)
	}
}

// TestVerifyDetectsMisalignedConnector verifies that VerifyDiagram reports an
// error when a free-line connector is off by 1 column relative to the source
// char on the previous line.
func TestVerifyDetectsMisalignedConnector(t *testing.T) {
	base := buildConnectedBoxes(14, 6, []string{" Box A        "}, []string{" Box B        "})
	// Shift connector │ right by 1 on line 3 (should be at col 7).
	base[3] = "        │" // 8 spaces + │

	boxes, _, err := ParseDiagram(base)
	if err != nil {
		t.Fatalf("ParseDiagram: %v", err)
	}
	errs := VerifyDiagram(base, boxes)

	found := false
	for _, e := range errs {
		if e.Line == 3 {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("VerifyDiagram should report error on line 3 (misaligned connector), got: %v", errs)
	}
}

// safeRune returns the rune at index i or '?' if out of bounds.
func safeRune(runes []rune, i int) rune {
	if i < 0 || i >= len(runes) {
		return '?'
	}
	return runes[i]
}
