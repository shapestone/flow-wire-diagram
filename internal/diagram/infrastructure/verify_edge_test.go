package infrastructure

import (
	"strings"
	"testing"

	"github.com/shapestone/flow-wire-diagram/internal/diagram/domain"
)

func TestCheckFrameEdgeCases(t *testing.T) {
	t.Run("correct top frame", func(t *testing.T) {
		errs := checkFrame("┌──────────────────┐", 0, 0, 19, true)
		if len(errs) != 0 {
			t.Errorf("expected no errors for correct top frame, got: %v", errs)
		}
	})

	t.Run("correct bottom frame", func(t *testing.T) {
		errs := checkFrame("└──────────────────┘", 0, 0, 19, false)
		if len(errs) != 0 {
			t.Errorf("expected no errors for correct bottom frame, got: %v", errs)
		}
	})

	t.Run("wrong left corner top frame", func(t *testing.T) {
		errs := checkFrame("X──────────────────┐", 0, 0, 19, true)
		if len(errs) == 0 {
			t.Error("expected error for wrong left corner in top frame")
		}
	})

	t.Run("wrong right corner top frame", func(t *testing.T) {
		errs := checkFrame("┌──────────────────X", 0, 0, 19, true)
		if len(errs) == 0 {
			t.Error("expected error for wrong right corner in top frame")
		}
	})

	t.Run("wrong corners bottom frame", func(t *testing.T) {
		errs := checkFrame("X──────────────────X", 0, 0, 19, false)
		if len(errs) == 0 {
			t.Error("expected errors for wrong corners in bottom frame")
		}
	})

	t.Run("line too short for leftCol", func(t *testing.T) {
		errs := checkFrame("", 0, 5, 10, true)
		found := false
		for _, e := range errs {
			if strings.Contains(e.Message, "line too short") {
				found = true
			}
		}
		if !found {
			t.Errorf("expected 'line too short' error for empty line, got: %v", errs)
		}
	})

	t.Run("line too short for rightCol only", func(t *testing.T) {
		// "┌───┐" has 5 runes (cols 0-4); rightCol=10 is out of range.
		errs := checkFrame("┌───┐", 0, 0, 10, true)
		found := false
		for _, e := range errs {
			if strings.Contains(e.Message, "line too short") {
				found = true
			}
		}
		if !found {
			t.Errorf("expected 'line too short' error for rightCol out of range, got: %v", errs)
		}
	})
}

func TestVerifyDiagramEdgeCases(t *testing.T) {
	t.Run("correct simple box", func(t *testing.T) {
		lines := []string{
			"┌──────────────┐",
			"│  some text   │",
			"└──────────────┘",
		}
		boxes, _, err := ParseDiagram(lines)
		if err != nil {
			t.Fatalf("ParseDiagram: %v", err)
		}
		errs := VerifyDiagram(lines, boxes)
		if len(errs) != 0 {
			t.Errorf("expected no errors for correct box, got: %v", errs)
		}
	})

	t.Run("content line too short at left col", func(t *testing.T) {
		// Box: LeftCol=5, RightCol=10. Content line only 2 chars → LeftCol out of range.
		b := &domain.Box{TopLine: 0, BottomLine: 2, LeftCol: 5, RightCol: 10, Width: 6}
		lines := []string{
			"     ┌────┐",
			"ab",
			"     └────┘",
		}
		errs := VerifyDiagram(lines, []*domain.Box{b})
		found := false
		for _, e := range errs {
			if e.Line == 1 && strings.Contains(e.Message, "line too short") {
				found = true
			}
		}
		if !found {
			t.Errorf("expected 'line too short' on line 1 for left col, got: %v", errs)
		}
	})

	t.Run("content line too short at right col", func(t *testing.T) {
		// Box: LeftCol=0, RightCol=10. Content line has │ at col 0 but only 4 chars.
		b := &domain.Box{TopLine: 0, BottomLine: 2, LeftCol: 0, RightCol: 10, Width: 11}
		lines := []string{
			"┌─────────┐",
			"│abc",
			"└─────────┘",
		}
		errs := VerifyDiagram(lines, []*domain.Box{b})
		found := false
		for _, e := range errs {
			if e.Line == 1 && strings.Contains(e.Message, "line too short") {
				found = true
			}
		}
		if !found {
			t.Errorf("expected 'line too short' on line 1 for right col, got: %v", errs)
		}
	})

	t.Run("content line wrong left edge char", func(t *testing.T) {
		lines := []string{
			"┌──────────────┐",
			"│  some text   │",
			"└──────────────┘",
		}
		boxes, _, err := ParseDiagram(lines)
		if err != nil {
			t.Fatalf("ParseDiagram: %v", err)
		}
		broken := []string{
			"┌──────────────┐",
			"X  some text   │", // wrong char at left edge
			"└──────────────┘",
		}
		errs := VerifyDiagram(broken, boxes)
		found := false
		for _, e := range errs {
			if e.Line == 1 && strings.Contains(e.Message, "expected │ at col 0") {
				found = true
			}
		}
		if !found {
			t.Errorf("expected error for wrong left edge char, got: %v", errs)
		}
	})

	t.Run("content line with trailing text no width error", func(t *testing.T) {
		// A content line with trailing text should not produce a spurious width error.
		lines := []string{
			"┌──────────────┐",
			"│  API Layer   │",
			"└──────────────┘",
		}
		boxes, _, err := ParseDiagram(lines)
		if err != nil {
			t.Fatalf("ParseDiagram: %v", err)
		}
		// Add trailing text to the content line.
		withTrailing := []string{
			"┌──────────────┐",
			"│  API Layer   │ <- trailing comment",
			"└──────────────┘",
		}
		errs := VerifyDiagram(withTrailing, boxes)
		for _, e := range errs {
			if e.Line == 1 && strings.Contains(e.Message, "width") {
				t.Errorf("unexpected width error on content line with trailing text: %v", e)
			}
		}
	})

	t.Run("box topline beyond lines slice", func(t *testing.T) {
		// Box with TopLine beyond the slice should not panic.
		b := &domain.Box{TopLine: 10, BottomLine: 12, LeftCol: 0, RightCol: 5, Width: 6}
		lines := []string{"short file"}
		// Should not panic.
		_ = VerifyDiagram(lines, []*domain.Box{b})
	})
}

func TestVerifyDiagramContentLineWidthError(t *testing.T) {
	// A content line with the correct │ positions but extra trailing spaces
	// (findTrailingText returns "" for all-space tails) should trigger the
	// width != expected error on line 71-73.
	b := &domain.Box{TopLine: 0, BottomLine: 2, LeftCol: 0, RightCol: 5, Width: 6, Parent: nil}
	lines := []string{
		"┌────┐",
		"│txt │     ", // │ at col 5 followed by spaces → trailing="" but lineWidth>6
		"└────┘",
	}
	errs := VerifyDiagram(lines, []*domain.Box{b})
	found := false
	for _, e := range errs {
		if e.Line == 1 && strings.Contains(e.Message, "width") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected width error on content line with trailing spaces, got: %v", errs)
	}
}

func TestVerifyDiagramWideCharsInBox(t *testing.T) {
	// A box where a content line contains a wide character (emoji).
	lines := []string{
		"┌──────────────────┐",
		"│  Hello 🌍        │",
		"└──────────────────┘",
	}
	boxes, _, err := ParseDiagram(lines)
	if err != nil {
		t.Fatalf("ParseDiagram: %v", err)
	}
	errs := VerifyDiagram(lines, boxes)
	found := false
	for _, e := range errs {
		if strings.Contains(e.Message, "wide character") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected wide character error, got: %v", errs)
	}
}

func TestRuneAtColBeyondEnd(t *testing.T) {
	got := runeAtCol("abc", 10)
	if got != 0 {
		t.Errorf("runeAtCol beyond line end: got %q, want 0", got)
	}
}
