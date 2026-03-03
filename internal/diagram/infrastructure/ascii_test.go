package infrastructure

import "testing"

func TestDetectWideChars(t *testing.T) {
	t.Run("no wide chars", func(t *testing.T) {
		got := DetectWideChars("plain ASCII text")
		if len(got) != 0 {
			t.Errorf("expected no wide chars, got: %v", got)
		}
	})

	t.Run("empty string", func(t *testing.T) {
		got := DetectWideChars("")
		if len(got) != 0 {
			t.Errorf("expected no wide chars for empty string, got: %v", got)
		}
	})

	t.Run("with emoji", func(t *testing.T) {
		got := DetectWideChars("Hello 🌍 World")
		if len(got) != 1 || got[0] != '🌍' {
			t.Errorf("expected [🌍], got: %v", got)
		}
	})

	t.Run("with CJK char", func(t *testing.T) {
		got := DetectWideChars("中文")
		if len(got) != 2 {
			t.Errorf("expected 2 wide chars, got: %d", len(got))
		}
	})
}

func TestConvertToASCII(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"┌──┐", "+--+"},
		{"│x │", "|x |"},
		{"└──┘", "+--+"},
		{"plain text", "plain text"},
		{"▼→•", "v>*"},
		{"┬┴├┤┼", "+++++"},
		{"▶◀▲←", "><^<"},
		{"mixed: ┌box┐", "mixed: +box+"},
		{"", ""},
	}
	for _, tc := range tests {
		got := ConvertToASCII(tc.in)
		if got != tc.want {
			t.Errorf("ConvertToASCII(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestIsASCIISafe(t *testing.T) {
	tests := []struct {
		line string
		want bool
	}{
		{"plain text", true},
		{"", true},
		{"+--+", true},
		{"|x |", true},
		{"┌──┐", false},
		{"│ text │", false},
		{"└──┘", false},
		{"text with ─ dash", false},
		{"▼ arrow", false},
		{"→ right", false},
	}
	for _, tc := range tests {
		got := IsASCIISafe(tc.line)
		if got != tc.want {
			t.Errorf("IsASCIISafe(%q) = %v, want %v", tc.line, got, tc.want)
		}
	}
}
