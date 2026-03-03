package infrastructure

import (
	"testing"

	"github.com/shapestone/flow-wire-diagram/internal/diagram/domain"
)

func TestContainsBoxDrawing(t *testing.T) {
	tests := []struct {
		name  string
		lines []string
		want  bool
	}{
		{
			name:  "complete box diagram",
			lines: []string{"┌──┐", "│x │", "└──┘"},
			want:  true,
		},
		{
			name:  "tree diagram no corners",
			lines: []string{"└── item", "│   child"},
			want:  false,
		},
		{
			name:  "empty",
			lines: []string{},
			want:  false,
		},
		{
			name:  "corners but no vertical bar",
			lines: []string{"┌──┐"},
			want:  false,
		},
		{
			name:  "vertical bars but no corners",
			lines: []string{"│text│"},
			want:  false,
		},
		{
			name:  "has top-left and vertical but no top-right",
			lines: []string{"┌──", "│x "},
			want:  false,
		},
		{
			name:  "corners and vertical scattered across lines",
			lines: []string{"┌──x", "│ y", "x──┐"},
			want:  true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := containsBoxDrawing(tc.lines)
			if got != tc.want {
				t.Errorf("containsBoxDrawing(%v) = %v, want %v", tc.lines, got, tc.want)
			}
		})
	}
}

func TestIsFenceLine(t *testing.T) {
	tests := []struct {
		line string
		want bool
	}{
		{"```", true},
		{"```go", true},
		{"```ascii", true},
		{"~~~", true},
		{"~~~python", true},
		{"  ```", true},
		{"\t```", true},
		{"   ~~~text", true},
		{"normal line", false},
		{"", false},
		{"not```fence", false},
		{"text~~~", false},
	}
	for _, tc := range tests {
		got := isFenceLine(tc.line)
		if got != tc.want {
			t.Errorf("isFenceLine(%q) = %v, want %v", tc.line, got, tc.want)
		}
	}
}

func TestExtractBlocks(t *testing.T) {
	t.Run("empty content", func(t *testing.T) {
		blocks := ExtractBlocks("")
		if len(blocks) == 0 {
			t.Error("expected at least one block for empty content")
		}
		for _, b := range blocks {
			if b.Kind != domain.BlockPassthrough {
				t.Errorf("expected passthrough, got %v", b.Kind)
			}
		}
	})

	t.Run("passthrough only", func(t *testing.T) {
		content := "# Heading\nSome text\n"
		blocks := ExtractBlocks(content)
		for _, b := range blocks {
			if b.Kind != domain.BlockPassthrough {
				t.Errorf("expected passthrough for plain text, got %v", b.Kind)
			}
		}
	})

	t.Run("diagram block backtick fence", func(t *testing.T) {
		content := "```ascii\n┌──┐\n│x │\n└──┘\n```"
		blocks := ExtractBlocks(content)
		hasDiagram := false
		for _, b := range blocks {
			if b.Kind == domain.BlockDiagram {
				hasDiagram = true
				if b.FenceOpen != "```ascii" {
					t.Errorf("FenceOpen: want %q, got %q", "```ascii", b.FenceOpen)
				}
				if b.FenceClose != "```" {
					t.Errorf("FenceClose: want %q, got %q", "```", b.FenceClose)
				}
				if len(b.Lines) != 3 {
					t.Errorf("Lines count: want 3, got %d", len(b.Lines))
				}
			}
		}
		if !hasDiagram {
			t.Error("expected at least one diagram block")
		}
	})

	t.Run("non-diagram fenced block", func(t *testing.T) {
		content := "```go\nfunc main() {}\n```"
		blocks := ExtractBlocks(content)
		for _, b := range blocks {
			if b.Kind == domain.BlockDiagram {
				t.Error("expected no diagram block for go code without box chars")
			}
		}
	})

	t.Run("unclosed fence", func(t *testing.T) {
		content := "```ascii\n┌──┐\n│x │\n└──┘"
		blocks := ExtractBlocks(content)
		hasDiagram := false
		for _, b := range blocks {
			if b.Kind == domain.BlockDiagram {
				hasDiagram = true
				if b.FenceClose != "" {
					t.Errorf("unclosed fence FenceClose should be empty, got %q", b.FenceClose)
				}
			}
		}
		if !hasDiagram {
			t.Error("expected diagram block even with unclosed fence")
		}
	})

	t.Run("mixed content", func(t *testing.T) {
		content := "# Heading\n\n```ascii\n┌──┐\n│x │\n└──┘\n```\n\nMore prose"
		blocks := ExtractBlocks(content)
		nDiagram := 0
		for _, b := range blocks {
			if b.Kind == domain.BlockDiagram {
				nDiagram++
			}
		}
		if nDiagram != 1 {
			t.Errorf("expected 1 diagram block, got %d", nDiagram)
		}
		if len(blocks) < 3 {
			t.Errorf("expected at least 3 blocks, got %d", len(blocks))
		}
	})

	t.Run("tilde fence", func(t *testing.T) {
		content := "~~~ascii\n┌──┐\n│x │\n└──┘\n~~~"
		blocks := ExtractBlocks(content)
		hasDiagram := false
		for _, b := range blocks {
			if b.Kind == domain.BlockDiagram {
				hasDiagram = true
			}
		}
		if !hasDiagram {
			t.Error("expected diagram block with tilde fence")
		}
	})
}

func TestReconstructContent(t *testing.T) {
	t.Run("roundtrip diagram block", func(t *testing.T) {
		content := "# Heading\n\n```ascii\n┌──┐\n│x │\n└──┘\n```\n\nMore prose"
		blocks := ExtractBlocks(content)
		got := ReconstructContent(blocks)
		if got != content {
			t.Errorf("roundtrip failed:\n  want: %q\n   got: %q", content, got)
		}
	})

	t.Run("passthrough only blocks", func(t *testing.T) {
		blocks := []domain.Block{
			{Kind: domain.BlockPassthrough, Lines: []string{"line1", "line2"}},
		}
		got := ReconstructContent(blocks)
		want := "line1\nline2"
		if got != want {
			t.Errorf("passthrough only: got %q, want %q", got, want)
		}
	})

	t.Run("nil blocks", func(t *testing.T) {
		got := ReconstructContent(nil)
		if got != "" {
			t.Errorf("nil blocks: got %q, want %q", got, "")
		}
	})

	t.Run("block with open fence only", func(t *testing.T) {
		blocks := []domain.Block{
			{
				Kind:      domain.BlockDiagram,
				FenceOpen: "```ascii",
				Lines:     []string{"┌──┐", "│x │", "└──┘"},
				// FenceClose is empty (unclosed fence)
			},
		}
		got := ReconstructContent(blocks)
		want := "```ascii\n┌──┐\n│x │\n└──┘"
		if got != want {
			t.Errorf("open fence only: got %q, want %q", got, want)
		}
	})

	t.Run("multiple blocks roundtrip", func(t *testing.T) {
		content := "before\n\n```ascii\n┌──────┐\n│ box  │\n└──────┘\n```\n\nafter"
		blocks := ExtractBlocks(content)
		got := ReconstructContent(blocks)
		if got != content {
			t.Errorf("multiple blocks roundtrip:\n  want: %q\n   got: %q", content, got)
		}
	})
}
