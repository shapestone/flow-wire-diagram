package infrastructure

import (
	"fmt"
	"strings"

	"github.com/shapestone/flow-wire-diagram/internal/diagram/domain"
)

// isStructuralRune returns true for characters that repair is allowed to
// add, remove, or change: box-drawing chars and spaces.
func isStructuralRune(r rune) bool {
	switch r {
	case '─', '│', '┌', '┐', '└', '┘', '├', '┤', '┬', '┴', '┼', '▼', ' ', '\t':
		return true
	}
	return false
}

// TextContent strips structural characters from lines and returns only the
// text content that repair must never alter.
func TextContent(lines []string) string {
	var sb strings.Builder
	for _, line := range lines {
		for _, r := range line {
			if !isStructuralRune(r) {
				sb.WriteRune(r)
			}
		}
	}
	return sb.String()
}

// VerifyDiagram checks repaired lines for alignment correctness.
// Returns a list of defects found. Empty list means the diagram is correct.
func VerifyDiagram(lines []string, boxes []*domain.Box) []domain.VerifyError {
	var errs []domain.VerifyError

	for _, b := range boxes {
		// 1. Check top frame.
		if b.TopLine < len(lines) {
			errs = append(errs, checkFrame(lines[b.TopLine], b.TopLine, b.LeftCol, b.RightCol, true)...)
		}

		// 2. Check bottom frame.
		if b.BottomLine < len(lines) {
			errs = append(errs, checkFrame(lines[b.BottomLine], b.BottomLine, b.LeftCol, b.RightCol, false)...)
		}

		// 3. Check content lines: vertical char at LeftCol and RightCol.
		for lineIdx := b.TopLine + 1; lineIdx < b.BottomLine; lineIdx++ {
			if lineIdx >= len(lines) {
				break
			}
			line := lines[lineIdx]
			runes := []rune(line)

			// Check left edge — accepts │ ┌ └ ├
			if b.LeftCol < len(runes) {
				ch := runes[b.LeftCol]
				if ch != '│' && ch != '┌' && ch != '└' && ch != '├' {
					errs = append(errs, domain.VerifyError{Line: lineIdx, Message: fmt.Sprintf("expected │ at col %d (box left)", b.LeftCol)})
				}
			} else {
				errs = append(errs, domain.VerifyError{Line: lineIdx, Message: fmt.Sprintf("line too short: expected │ at col %d", b.LeftCol)})
			}

			// Check right edge — accepts │ ┐ ┘ ┤
			if b.RightCol < len(runes) {
				ch := runes[b.RightCol]
				if ch != '│' && ch != '┐' && ch != '┘' && ch != '┤' {
					errs = append(errs, domain.VerifyError{Line: lineIdx, Message: fmt.Sprintf("expected │ at col %d (box right)", b.RightCol)})
				}
			} else {
				errs = append(errs, domain.VerifyError{Line: lineIdx, Message: fmt.Sprintf("line too short: expected │ at col %d", b.RightCol)})
			}

			// 4. Check line width for outermost boxes.
			if b.Parent == nil {
				lineWidth := StringWidth(line)
				trailing := findTrailingText(line, b.RightCol)
				if trailing != "" {
					lineWidth = StringWidth(line) - StringWidth(trailing)
				}
				expected := b.RightCol + 1
				if lineWidth != expected {
					errs = append(errs, domain.VerifyError{Line: lineIdx, Message: fmt.Sprintf("width %d, expected %d", lineWidth, expected)})
				}
			}
		}

		// 5. Check for wide characters.
		for lineIdx := b.TopLine; lineIdx <= b.BottomLine && lineIdx < len(lines); lineIdx++ {
			wide := DetectWideChars(lines[lineIdx])
			for _, r := range wide {
				errs = append(errs, domain.VerifyError{Line: lineIdx, Message: fmt.Sprintf("wide character U+%04X (%c) in diagram", r, r)})
			}
		}
	}

	errs = append(errs, checkConnectorAlignment(lines, boxes)...)

	return errs
}

// runeAtCol returns the rune at the given visual column in line, or 0 if the
// column is out of range.
func runeAtCol(line string, col int) rune {
	c := 0
	for _, r := range line {
		if c == col {
			return r
		}
		c += RuneWidthOf(r)
	}
	return 0
}

// checkConnectorAlignment verifies that every │/▼ on a free (between-boxes)
// line is aligned with a source char on the line immediately above.
// An error is reported when exactly one source falls within ±connectorAlignWindow
// at a different column — meaning the position is unambiguously wrong.
func checkConnectorAlignment(lines []string, boxes []*domain.Box) []domain.VerifyError {
	insideBox := make(map[int]bool)
	for _, b := range boxes {
		for i := b.TopLine; i <= b.BottomLine; i++ {
			insideBox[i] = true
		}
	}

	var errs []domain.VerifyError
	for i := 1; i < len(lines); i++ {
		if insideBox[i] {
			continue
		}
		connCols := findConnectorChars(lines[i])
		if len(connCols) == 0 {
			continue
		}
		sources := findSourceChars(lines[i-1])
		for _, x := range connCols {
			if _, ok := sources[x]; ok {
				continue // already aligned
			}
			var candidates []int
			for srcCol := range sources {
				if srcCol >= x-connectorAlignWindow && srcCol <= x+connectorAlignWindow {
					candidates = append(candidates, srcCol)
				}
			}
			if len(candidates) == 1 {
				errs = append(errs, domain.VerifyError{
					Line: i,
					Message: fmt.Sprintf("connector %c at col %d should be at col %d",
						runeAtCol(lines[i], x), x, candidates[0]),
				})
			}
		}
	}
	return errs
}

// checkFrame verifies a top or bottom frame line.
func checkFrame(line string, lineIdx, leftCol, rightCol int, isTop bool) []domain.VerifyError {
	var errs []domain.VerifyError
	runes := []rune(line)

	var leftExpected, rightExpected rune
	if isTop {
		leftExpected, rightExpected = '┌', '┐'
	} else {
		leftExpected, rightExpected = '└', '┘'
	}

	if leftCol < len(runes) {
		ch := runes[leftCol]
		if ch != leftExpected {
			errs = append(errs, domain.VerifyError{Line: lineIdx, Message: fmt.Sprintf("expected %c at col %d, got %c", leftExpected, leftCol, ch)})
		}
	} else {
		errs = append(errs, domain.VerifyError{Line: lineIdx, Message: fmt.Sprintf("line too short for frame at col %d", leftCol)})
	}

	if rightCol < len(runes) {
		ch := runes[rightCol]
		if ch != rightExpected {
			errs = append(errs, domain.VerifyError{Line: lineIdx, Message: fmt.Sprintf("expected %c at col %d, got %c", rightExpected, rightCol, ch)})
		}
	} else {
		errs = append(errs, domain.VerifyError{Line: lineIdx, Message: fmt.Sprintf("line too short for frame at col %d", rightCol)})
	}

	return errs
}
