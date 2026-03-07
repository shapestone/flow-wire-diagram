package infrastructure

import (
	"strings"
	"testing"

	"github.com/shapestone/flow-wire-diagram/internal/diagram/domain"
)

func TestRepairFreeLineEmptyPrev(t *testing.T) {
	// When prevRepaired is empty, repairFreeLine must return original unchanged.
	original := "   │"
	got := repairFreeLine(original, "")
	if got != original {
		t.Errorf("repairFreeLine(empty prev): got %q, want %q", got, original)
	}
}

func TestRepairFreeLineNoConnectors(t *testing.T) {
	// A free line with no │ or ▼ chars is returned unchanged.
	original := "plain text no connectors"
	got := repairFreeLine(original, "│ something │")
	if got != original {
		t.Errorf("repairFreeLine(no connectors): got %q, want %q", got, original)
	}
}

func TestRepairFreeLineNoSources(t *testing.T) {
	// A free line where prevRepaired has no source chars is returned unchanged.
	original := "   │"
	got := repairFreeLine(original, "plain text no source chars")
	if got != original {
		t.Errorf("repairFreeLine(no sources): got %q, want %q", got, original)
	}
}

func TestRepairLinesDefaultRole(t *testing.T) {
	// A DiagramLine with an unknown Role hits the default case and passes through.
	dl := domain.DiagramLine{
		Index:    0,
		Original: "some line",
		Role:     domain.LineRole(99),
	}
	result, err := RepairLines([]domain.DiagramLine{dl}, nil)
	if err != nil {
		t.Fatalf("RepairLines: %v", err)
	}
	if result[0] != "some line" {
		t.Errorf("default role: got %q, want %q", result[0], "some line")
	}
}

func TestRepairFrameLineNoBoxes(t *testing.T) {
	// A frame line with no active boxes passes through unchanged.
	dl := domain.DiagramLine{
		Index:       0,
		Original:    "┌──┐",
		Role:        domain.RoleTopFrame,
		ActiveBoxes: nil,
	}
	result, err := RepairLines([]domain.DiagramLine{dl}, nil)
	if err != nil {
		t.Fatalf("RepairLines: %v", err)
	}
	if result[0] != "┌──┐" {
		t.Errorf("frame no boxes: got %q, want %q", result[0], "┌──┐")
	}
}

func TestRepairFrameLineMultipleRoots(t *testing.T) {
	// A frame line with multiple root boxes (no parent) passes through unchanged.
	box1 := &domain.Box{TopLine: 0, BottomLine: 2, LeftCol: 0, RightCol: 5}
	box2 := &domain.Box{TopLine: 0, BottomLine: 2, LeftCol: 5, RightCol: 10}
	original := "┌────┌────┐"
	dl := domain.DiagramLine{
		Index:       0,
		Original:    original,
		Role:        domain.RoleTopFrame,
		ActiveBoxes: []*domain.Box{box1, box2},
	}
	result, err := RepairLines([]domain.DiagramLine{dl}, nil)
	if err != nil {
		t.Fatalf("RepairLines: %v", err)
	}
	if result[0] != original {
		t.Errorf("frame multiple roots: got %q, want %q", result[0], original)
	}
}

func TestRepairContentLineNoBoxes(t *testing.T) {
	// A content line with no active boxes passes through unchanged.
	original := "│ text │"
	dl := domain.DiagramLine{
		Index:       0,
		Original:    original,
		Role:        domain.RoleContent,
		ActiveBoxes: nil,
	}
	result, err := RepairLines([]domain.DiagramLine{dl}, nil)
	if err != nil {
		t.Fatalf("RepairLines: %v", err)
	}
	if result[0] != original {
		t.Errorf("content no boxes: got %q, want %q", result[0], original)
	}
}

func TestRepairContentLineMultipleRoots(t *testing.T) {
	// A content line with multiple root boxes (no parent) passes through unchanged.
	box1 := &domain.Box{TopLine: 0, BottomLine: 2, LeftCol: 0, RightCol: 5}
	box2 := &domain.Box{TopLine: 0, BottomLine: 2, LeftCol: 5, RightCol: 10}
	original := "│ ab │ cd │"
	dl := domain.DiagramLine{
		Index:       0,
		Original:    original,
		Role:        domain.RoleContent,
		ActiveBoxes: []*domain.Box{box1, box2},
	}
	result, err := RepairLines([]domain.DiagramLine{dl}, nil)
	if err != nil {
		t.Fatalf("RepairLines: %v", err)
	}
	if result[0] != original {
		t.Errorf("content multiple roots: got %q, want %q", result[0], original)
	}
}

func TestRepairFrameLineWithTrailingText(t *testing.T) {
	// A frame DiagramLine with TrailingText set should include the trailing
	// text in the repaired output (covers the dl.TrailingText != "" branch
	// in repairFrameLine).
	b := &domain.Box{TopLine: 0, BottomLine: 2, LeftCol: 0, RightCol: 5, Width: 6}
	trailing := "  <- comment"
	dl := domain.DiagramLine{
		Index:        0,
		Original:     "┌────┐" + trailing,
		Role:         domain.RoleTopFrame,
		ActiveBoxes:  []*domain.Box{b},
		TrailingText: trailing,
	}
	result, err := RepairLines([]domain.DiagramLine{dl}, nil)
	if err != nil {
		t.Fatalf("RepairLines: %v", err)
	}
	if !strings.HasSuffix(result[0], trailing) {
		t.Errorf("frame with trailing text: got %q, want suffix %q", result[0], trailing)
	}
}

func TestRepairContentLineAdjacentPipes(t *testing.T) {
	// A box where LeftCol and RightCol are adjacent (width 2) produces
	// segWidth = rightTarget - leftTarget - 1 = 0, triggering the `continue`.
	b := &domain.Box{TopLine: 0, BottomLine: 2, LeftCol: 5, RightCol: 6, Width: 2}
	original := "     ││"
	dl := domain.DiagramLine{
		Index:       0,
		Original:    original,
		Role:        domain.RoleContent,
		ActiveBoxes: []*domain.Box{b},
		TargetWidth: 7,
	}
	result, err := RepairLines([]domain.DiagramLine{dl}, nil)
	if err != nil {
		t.Fatalf("RepairLines: %v", err)
	}
	_ = result[0] // just verify no panic
}

// TestRepairContentLineTooWideNoSlack verifies that when a content line is
// wider than the target box and the text fills the segment with no trailing
// space, repairContentLine returns the original line unchanged rather than
// silently truncating the last character.
func TestRepairContentLineTooWideNoSlack(t *testing.T) {
	// Box: LeftCol=0, RightCol=10 (width 11). segWidth = 9.
	// Line: "│ abcdefghi│" — 12 chars wide, │ at col 11 (off by 1).
	// Content between walls: " abcdefghi" (10 chars, no trailing space).
	// 10 > segWidth(9) → must return original unchanged.
	b := &domain.Box{TopLine: 0, BottomLine: 2, LeftCol: 0, RightCol: 10, Width: 11}
	original := "│ abcdefghi│"
	dl := domain.DiagramLine{
		Index:       0,
		Original:    original,
		Role:        domain.RoleContent,
		ActiveBoxes: []*domain.Box{b},
		TargetWidth: 11,
	}
	result, err := RepairLines([]domain.DiagramLine{dl}, nil)
	if err != nil {
		t.Fatalf("RepairLines: %v", err)
	}
	if result[0] != original {
		t.Errorf("content too wide, no slack: got %q, want original %q", result[0], original)
	}
}

func TestRepairContentLineDegenerateBox(t *testing.T) {
	// A degenerate box where LeftCol == RightCol produces only 1 unique pipe column.
	// When actualPipes count != pipeCols count and pipeCols < 2, returns dl.Original.
	b := &domain.Box{TopLine: 0, BottomLine: 2, LeftCol: 5, RightCol: 5, Width: 1}
	// Two │ chars in the line (at cols 5 and 10) but only 1 unique pipe column in box.
	original := "     │text│"
	dl := domain.DiagramLine{
		Index:       0,
		Original:    original,
		Role:        domain.RoleContent,
		ActiveBoxes: []*domain.Box{b},
		TargetWidth: 6,
	}
	result, err := RepairLines([]domain.DiagramLine{dl}, nil)
	if err != nil {
		t.Fatalf("RepairLines: %v", err)
	}
	// Result should be set (no panic) — either original or repaired.
	_ = result[0]
}
