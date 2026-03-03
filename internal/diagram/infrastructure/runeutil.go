package infrastructure

import (
	"strings"

	"github.com/mattn/go-runewidth"
)

// StringWidth returns the visual monospace width of s.
func StringWidth(s string) int {
	return runewidth.StringWidth(s)
}

// VisualPad pads s with trailing spaces until its visual width equals target.
// If s is already >= target, returns s unchanged.
func VisualPad(s string, target int) string {
	current := runewidth.StringWidth(s)
	if current >= target {
		return s
	}
	return s + strings.Repeat(" ", target-current)
}

// RuneWidthOf returns the visual width of a single rune.
func RuneWidthOf(r rune) int {
	return runewidth.RuneWidth(r)
}

// ExpandTabs replaces each tab character with two spaces in every line.
// Tabs have no defined visual width in diagrams, so a fixed 2-space
// expansion normalises them before alignment repair.
func ExpandTabs(lines []string) []string {
	out := make([]string, len(lines))
	for i, line := range lines {
		out[i] = strings.ReplaceAll(line, "\t", "  ")
	}
	return out
}

// DetectWideChars returns all runes in s where RuneWidth != 1.
func DetectWideChars(s string) []rune {
	var wide []rune
	for _, r := range s {
		if runewidth.RuneWidth(r) != 1 {
			wide = append(wide, r)
		}
	}
	return wide
}

// lineRunes returns the runes, visual column positions, and total visual width of a line.
func lineRunes(line string) (runes []rune, cols []int, totalWidth int) {
	runes = []rune(line)
	cols = make([]int, len(runes))
	col := 0
	for i, r := range runes {
		cols[i] = col
		col += RuneWidthOf(r)
	}
	totalWidth = col
	return
}

// extractBetweenCols returns the substring of line covering visual columns [startCol, endCol).
func extractBetweenCols(line string, startCol, endCol int) string {
	var sb strings.Builder
	col := 0
	for _, r := range line {
		w := RuneWidthOf(r)
		if col >= endCol {
			break
		}
		if col >= startCol {
			sb.WriteRune(r)
		}
		col += w
	}
	return sb.String()
}
