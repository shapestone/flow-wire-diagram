package infrastructure

import (
	"testing"

	"github.com/shapestone/flow-wire-diagram/internal/diagram/domain"
)

func TestParseSimpleBox(t *testing.T) {
	lines := []string{
		"┌──────────────┐",
		"│  some text   │",
		"└──────────────┘",
	}
	boxes, diagLines, err := ParseDiagram(lines)
	if err != nil {
		t.Fatalf("ParseDiagram error: %v", err)
	}
	if len(boxes) != 1 {
		t.Fatalf("expected 1 root box, got %d", len(boxes))
	}
	b := boxes[0]
	if b.LeftCol != 0 {
		t.Errorf("LeftCol: want 0, got %d", b.LeftCol)
	}
	if b.RightCol != 15 {
		t.Errorf("RightCol: want 15, got %d", b.RightCol)
	}
	if b.TopLine != 0 {
		t.Errorf("TopLine: want 0, got %d", b.TopLine)
	}
	if b.BottomLine != 2 {
		t.Errorf("BottomLine: want 2, got %d", b.BottomLine)
	}
	if b.Width != 16 {
		t.Errorf("Width: want 16, got %d", b.Width)
	}
	if b.Parent != nil {
		t.Errorf("Parent: want nil, got non-nil")
	}
	if diagLines[0].Role != domain.RoleTopFrame {
		t.Errorf("line 0 role: want RoleTopFrame, got %v", diagLines[0].Role)
	}
	if diagLines[1].Role != domain.RoleContent {
		t.Errorf("line 1 role: want RoleContent, got %v", diagLines[1].Role)
	}
	if diagLines[2].Role != domain.RoleBottomFrame {
		t.Errorf("line 2 role: want RoleBottomFrame, got %v", diagLines[2].Role)
	}
}

func TestParseNestedBoxes(t *testing.T) {
	lines := []string{
		"┌────────────────────┐", // line 0: outer top
		"│  ┌──────────────┐  │", // line 1: inner top
		"│  │ inner box     │  │", // line 2: inner content
		"│  └──────────────┘  │", // line 3: inner bottom
		"└────────────────────┘", // line 4: outer bottom
	}
	boxes, _, err := ParseDiagram(lines)
	if err != nil {
		t.Fatalf("ParseDiagram error: %v", err)
	}
	allBoxes := collectAllBoxes(boxes)
	if len(allBoxes) != 2 {
		t.Fatalf("expected 2 boxes total, got %d", len(allBoxes))
	}
	if len(boxes) != 1 {
		t.Fatalf("expected 1 root box, got %d", len(boxes))
	}
	outer := boxes[0]
	if outer.LeftCol != 0 || outer.RightCol != 21 {
		t.Errorf("outer: LeftCol=%d RightCol=%d, want 0 21", outer.LeftCol, outer.RightCol)
	}
	if len(outer.Children) != 1 {
		t.Fatalf("outer children: want 1, got %d", len(outer.Children))
	}
	inner := outer.Children[0]
	if inner.LeftCol != 3 || inner.RightCol != 18 {
		t.Errorf("inner: LeftCol=%d RightCol=%d, want 3 18", inner.LeftCol, inner.RightCol)
	}
	if inner.Parent != outer {
		t.Errorf("inner.Parent should be outer")
	}
}

func TestParseSideBySide(t *testing.T) {
	lines := []string{
		"┌──────────────────────┐",  // line 0: outer top
		"│  ┌──────┐  ┌──────┐  │", // line 1: two inner tops
		"│  │  A   │  │  B   │  │", // line 2: content
		"│  └──────┘  └──────┘  │", // line 3: two inner bottoms
		"└──────────────────────┘",  // line 4: outer bottom
	}
	boxes, _, err := ParseDiagram(lines)
	if err != nil {
		t.Fatalf("ParseDiagram error: %v", err)
	}
	all := collectAllBoxes(boxes)
	if len(all) != 3 {
		t.Fatalf("expected 3 boxes, got %d", len(all))
	}
	if len(boxes) != 1 {
		t.Fatalf("expected 1 root box, got %d", len(boxes))
	}
	outer := boxes[0]
	if len(outer.Children) != 2 {
		t.Errorf("outer children: want 2, got %d", len(outer.Children))
	}
}

func TestParseMultiCell(t *testing.T) {
	lines := []string{
		"┌──────────────┐", // line 0: box A top
		"│ Box A        │", // line 1
		"└──────┬───────┘", // line 2: box A bottom
		"       │",         // line 3: connector
		"┌──────▼───────┐", // line 4: box B top
		"│ Box B        │", // line 5
		"└──────────────┘", // line 6: box B bottom
	}
	boxes, diagLines, err := ParseDiagram(lines)
	if err != nil {
		t.Fatalf("ParseDiagram error: %v", err)
	}
	if len(boxes) != 2 {
		t.Fatalf("expected 2 root boxes, got %d", len(boxes))
	}
	if diagLines[3].Role != domain.RoleFree {
		t.Errorf("connector line 3 role: want RoleFree, got %v", diagLines[3].Role)
	}
	for i, b := range boxes {
		if b.LeftCol != 0 || b.RightCol != 15 {
			t.Errorf("box %d: LeftCol=%d RightCol=%d, want 0 15", i, b.LeftCol, b.RightCol)
		}
	}
}

// collectAllBoxes flattens a tree of boxes into a slice.
func collectAllBoxes(roots []*domain.Box) []*domain.Box {
	var all []*domain.Box
	var walk func(b *domain.Box)
	walk = func(b *domain.Box) {
		all = append(all, b)
		for _, c := range b.Children {
			walk(c)
		}
	}
	for _, r := range roots {
		walk(r)
	}
	return all
}
