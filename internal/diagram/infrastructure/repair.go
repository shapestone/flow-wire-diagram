package infrastructure

import (
	"sort"
	"strings"

	"github.com/shapestone/flow-wire-diagram/internal/diagram/domain"
)

// connectorAlignWindow is the maximum column distance (±) within which a
// source char on the previous line is considered the anchor for a free-line
// connector char.
const connectorAlignWindow = 2

// isConnectorSourceRune returns true for chars that can serve as the
// attachment point for a vertical connector on the line above.
func isConnectorSourceRune(r rune) bool {
	switch r {
	case '│', '▼', '┐', '┬', '┼', '┤', '├':
		return true
	}
	return false
}

// findConnectorChars returns the visual column positions of all │ and ▼
// chars in line.
func findConnectorChars(line string) []int {
	var cols []int
	col := 0
	for _, r := range line {
		if r == '│' || r == '▼' {
			cols = append(cols, col)
		}
		col += RuneWidthOf(r)
	}
	return cols
}

// findSourceChars returns a map of visual column → rune for every
// isConnectorSourceRune char in line.
func findSourceChars(line string) map[int]rune {
	sources := make(map[int]rune)
	col := 0
	for _, r := range line {
		if isConnectorSourceRune(r) {
			sources[col] = r
		}
		col += RuneWidthOf(r)
	}
	return sources
}

// repairFreeLine aligns connector chars (│ ▼) on a free (between-boxes) line
// by snapping each one to the nearest source char on the already-repaired line
// above.  The move is applied only when exactly one source falls within
// ±connectorAlignWindow columns and it is at a different column.  Ambiguous
// (0 or 2+ sources) or already-aligned connectors are left unchanged.
func repairFreeLine(original, prevRepaired string) string {
	if prevRepaired == "" {
		return original
	}
	connCols := findConnectorChars(original)
	if len(connCols) == 0 {
		return original
	}
	sources := findSourceChars(prevRepaired)
	if len(sources) == 0 {
		return original
	}

	// Build rune buffer; since all box-drawing/space chars have width 1,
	// visual column == rune index throughout.
	buf := []rune(original)

	type move struct{ from, to int }
	var moves []move

	for _, x := range connCols {
		// Already aligned with a source — nothing to do.
		if _, ok := sources[x]; ok {
			continue
		}
		var candidates []int
		for srcCol := range sources {
			if srcCol >= x-connectorAlignWindow && srcCol <= x+connectorAlignWindow {
				candidates = append(candidates, srcCol)
			}
		}
		if len(candidates) == 1 {
			moves = append(moves, move{from: x, to: candidates[0]})
		}
	}

	if len(moves) == 0 {
		return original
	}

	for _, m := range moves {
		if m.to < 0 {
			continue
		}
		// Extend buffer with spaces if the target column is past the end.
		for len(buf) <= m.to {
			buf = append(buf, ' ')
		}
		ch := buf[m.from]
		buf[m.from] = ' '
		buf[m.to] = ch
	}

	return string(buf)
}

// RepairLines repairs diagram lines based on their classified roles.
// Returns the repaired lines in order.
//
// When a content line's text is wider than the current box frame allows,
// RepairLines widens the outermost box's RightCol to fit and then
// retroactively re-renders every line from the box's top frame down to
// (and including) the current line before continuing.
func RepairLines(diagLines []domain.DiagramLine, boxes []*domain.Box) ([]string, error) {
	result := make([]string, len(diagLines))

	// Index all diagram lines so we can look them up by line number when
	// retroactively re-rendering after a box is widened.
	byIndex := make(map[int]domain.DiagramLine, len(diagLines))
	for _, dl := range diagLines {
		byIndex[dl.Index] = dl
	}

	for _, dl := range diagLines {
		switch dl.Role {
		case domain.RoleFree:
			prev := ""
			if dl.Index > 0 {
				prev = result[dl.Index-1]
			}
			result[dl.Index] = repairFreeLine(dl.Original, prev)
		case domain.RoleTopFrame, domain.RoleBottomFrame:
			result[dl.Index] = repairFrameLine(dl)
		case domain.RoleContent:
			// Check whether the outermost box needs to be widened to
			// accommodate this line's content without truncation.
			if len(dl.ActiveBoxes) > 0 && countRootBoxes(dl) == 1 {
				outermost := dl.ActiveBoxes[0]
				if needed := neededRightCol(dl); needed > outermost.RightCol {
					outermost.RightCol = needed
					outermost.Width = needed - outermost.LeftCol + 1
					// Re-render every line from the box's top frame up to
					// (not including) the current line with the new width.
					for i := outermost.TopLine; i < dl.Index; i++ {
						prev, ok := byIndex[i]
						if !ok {
							continue
						}
						switch prev.Role {
						case domain.RoleTopFrame, domain.RoleBottomFrame:
							result[i] = repairFrameLine(prev)
						case domain.RoleContent:
							result[i] = repairContentLine(prev)
						}
					}
				}
			}
			result[dl.Index] = repairContentLine(dl)
		default:
			result[dl.Index] = dl.Original
		}
	}
	return result, nil
}

// neededRightCol returns the minimum RightCol required so that every segment
// of dl's content fits without truncation.  Returns the box's current RightCol
// when the content already fits or when the pipe count is ambiguous.
func neededRightCol(dl domain.DiagramLine) int {
	if len(dl.ActiveBoxes) == 0 || countRootBoxes(dl) != 1 {
		return dl.ActiveBoxes[0].RightCol
	}
	outermost := dl.ActiveBoxes[0]

	pipeSet := make(map[int]bool)
	for _, b := range dl.ActiveBoxes {
		pipeSet[b.LeftCol] = true
		pipeSet[b.RightCol] = true
	}
	var pipeCols []int
	for c := range pipeSet {
		pipeCols = append(pipeCols, c)
	}
	sort.Ints(pipeCols)

	actualPipes := findActualPipes(dl.Original)
	if len(actualPipes) != len(pipeCols) {
		return outermost.RightCol
	}

	needed := outermost.RightCol
	for seg := 0; seg < len(pipeCols)-1; seg++ {
		leftActual := actualPipes[seg]
		rightActual := actualPipes[seg+1]
		leftTarget := pipeCols[seg]
		rightTarget := pipeCols[seg+1]
		if rightTarget != outermost.RightCol {
			continue // only consider the outermost right wall
		}
		segWidth := rightTarget - leftTarget - 1
		if segWidth <= 0 {
			continue
		}
		content := extractBetweenCols(dl.Original, leftActual+1, rightActual)
		content = strings.TrimRight(content, " ")
		if w := StringWidth(content); w > segWidth {
			if n := leftTarget + w + 1; n > needed {
				needed = n
			}
		}
	}
	return needed
}

// countRootBoxes returns the number of active boxes with no parent.
func countRootBoxes(dl domain.DiagramLine) int {
	n := 0
	for _, b := range dl.ActiveBoxes {
		if b.Parent == nil {
			n++
		}
	}
	return n
}

// repairFrameLine fixes a top (┌─┐) or bottom (└─┘) frame line.
func repairFrameLine(dl domain.DiagramLine) string {
	if len(dl.ActiveBoxes) == 0 {
		return dl.Original
	}
	if countRootBoxes(dl) > 1 {
		return dl.Original
	}

	outermost := dl.ActiveBoxes[0]
	targetWidth := outermost.RightCol + 1

	buf := make([]rune, targetWidth)
	for i := range buf {
		buf[i] = ' '
	}

	for _, b := range dl.ActiveBoxes {
		isOwnFrame := (dl.Role == domain.RoleTopFrame && dl.Index == b.TopLine) ||
			(dl.Role == domain.RoleBottomFrame && dl.Index == b.BottomLine)

		if isOwnFrame {
			var leftCh, rightCh rune
			if dl.Role == domain.RoleTopFrame {
				leftCh, rightCh = '┌', '┐'
			} else {
				leftCh, rightCh = '└', '┘'
			}
			if b.LeftCol < targetWidth {
				buf[b.LeftCol] = leftCh
			}
			if b.RightCol < targetWidth {
				buf[b.RightCol] = rightCh
			}
			for c := b.LeftCol + 1; c < b.RightCol && c < targetWidth; c++ {
				if buf[c] == ' ' {
					buf[c] = '─'
				}
			}
		} else {
			if b.LeftCol < targetWidth {
				buf[b.LeftCol] = '│'
			}
			if b.RightCol < targetWidth {
				buf[b.RightCol] = '│'
			}
		}
	}

	// Preserve connector characters from the original line.
	origRunes, origCols, _ := lineRunes(dl.Original)
	for i, r := range origRunes {
		switch r {
		case '┬', '▼', '┴', '┼', '├', '┤':
			c := origCols[i]
			if c >= 0 && c < targetWidth {
				buf[c] = r
			}
		}
	}

	// Preserve any non-space chars from the original that lie outside all
	// active box column ranges (e.g. a vertical connector alongside the box).
	copyOutsideBoxChars(buf, dl.Original, dl.ActiveBoxes)

	// Preserve non-structural text that sits inside the outermost box but
	// outside every inner box's column range.  This covers annotations or
	// labels placed beside an inner box's top/bottom frame on the same line
	// (e.g. "│  ┌──┐   some label   │").  copyOutsideBoxChars skips these
	// because they are inside the outermost box; without this step the text
	// would be lost and the safety check would abort the repair.
	if len(dl.ActiveBoxes) > 1 {
		col := 0
		for _, r := range dl.Original {
			w := RuneWidthOf(r)
			if col > outermost.LeftCol && col < outermost.RightCol && col < len(buf) {
				// A │ connector that is far from any expected box wall column
				// should be preserved just like non-structural text (e.g. the
				// vertical leg of an elbow connector on a bottom-frame line).
				// │ chars within connectorAlignWindow of a box wall are treated
				// as shifted walls and must not be preserved here.
				nearBoxWall := false
				for _, b := range dl.ActiveBoxes {
					if absInt(col-b.LeftCol) <= connectorAlignWindow ||
						absInt(col-b.RightCol) <= connectorAlignWindow {
						nearBoxWall = true
						break
					}
				}
				if (!isStructuralRune(r) || (r == '│' && !nearBoxWall)) && buf[col] == ' ' {
					inInner := false
					for _, b := range dl.ActiveBoxes {
						if b != outermost && col >= b.LeftCol && col <= b.RightCol {
							inInner = true
							break
						}
					}
					if !inInner {
						buf[col] = r
					}
				}
			}
			col += w
		}
	}

	result := string(buf)
	if dl.TrailingText != "" {
		result += dl.TrailingText
	}
	return result
}

// repairContentLine fixes a content line (│ ... │) so │ chars are at exact columns.
func repairContentLine(dl domain.DiagramLine) string {
	if len(dl.ActiveBoxes) == 0 {
		return dl.Original
	}
	if countRootBoxes(dl) > 1 {
		return dl.Original
	}

	outermost := dl.ActiveBoxes[0]
	targetWidth := outermost.RightCol + 1

	pipeSet := make(map[int]bool)
	for _, b := range dl.ActiveBoxes {
		pipeSet[b.LeftCol] = true
		pipeSet[b.RightCol] = true
	}

	var pipeCols []int
	for c := range pipeSet {
		pipeCols = append(pipeCols, c)
	}
	sort.Ints(pipeCols)

	actualPipes := findActualPipes(dl.Original)

	buf := make([]rune, targetWidth)
	for i := range buf {
		buf[i] = ' '
	}

	for _, c := range pipeCols {
		if c < targetWidth {
			buf[c] = '│'
		}
	}

	if len(actualPipes) == len(pipeCols) {
		for seg := 0; seg < len(pipeCols)-1; seg++ {
			leftActual := actualPipes[seg]
			rightActual := actualPipes[seg+1]
			leftTarget := pipeCols[seg]
			rightTarget := pipeCols[seg+1]
			segWidth := rightTarget - leftTarget - 1

			if segWidth <= 0 {
				continue
			}

			content := extractBetweenCols(dl.Original, leftActual+1, rightActual)
			content = strings.TrimRight(content, " ")

			padded := []rune(VisualPad(content, segWidth))
			for j, r := range padded {
				pos := leftTarget + 1 + j
				if pos < rightTarget && pos < targetWidth {
					buf[pos] = r
				}
			}
		}
	} else {
		if len(pipeCols) >= 2 {
			// Actual │ count differs from expected box-wall count (e.g. extra
			// connector │ inside the box, or shifted walls).  Map the last two
			// expected pipe positions to the closest actual pipe positions so we
			// extract content from the right region, not from the whole span.
			innerLeft := pipeCols[len(pipeCols)-2]
			innerRight := pipeCols[len(pipeCols)-1]
			segWidth := innerRight - innerLeft - 1
			if segWidth > 0 {
				actLeft := closestPipe(actualPipes, innerLeft)
				actRight := closestPipe(actualPipes, innerRight)
				if actLeft >= 0 && actRight > actLeft {
					content := extractBetweenCols(dl.Original, actLeft+1, actRight)
					content = strings.TrimRight(content, " ")
					padded := []rune(VisualPad(content, segWidth))
					for j, r := range padded {
						pos := innerLeft + 1 + j
						if pos < innerRight && pos < targetWidth {
							buf[pos] = r
						}
					}
				}
			}
		} else {
			return dl.Original
		}
	}

	// Preserve any non-space chars from the original that lie outside all
	// active box column ranges (e.g. a vertical connector alongside the box).
	copyOutsideBoxChars(buf, dl.Original, dl.ActiveBoxes)

	result := string(buf)
	if dl.TrailingText != "" {
		result += dl.TrailingText
	}
	return result
}

// copyOutsideBoxChars copies non-space runes from the original line into buf
// at columns that fall outside every active box's [LeftCol, RightCol] span.
// This preserves visual elements (e.g. a vertical connector │ running alongside
// a box) that are not part of the box structure being repaired.
func copyOutsideBoxChars(buf []rune, original string, activeBoxes []*domain.Box) {
	col := 0
	for _, r := range original {
		if r != ' ' && col < len(buf) {
			outside := true
			for _, b := range activeBoxes {
				if col >= b.LeftCol && col <= b.RightCol {
					outside = false
					break
				}
			}
			if outside {
				buf[col] = r
			}
		}
		col += RuneWidthOf(r)
	}
}

// closestPipe returns the element of pipes that minimises |pipes[i] - target|.
// Returns -1 if pipes is empty.
func closestPipe(pipes []int, target int) int {
	if len(pipes) == 0 {
		return -1
	}
	best := pipes[0]
	bestDist := absInt(pipes[0] - target)
	for _, p := range pipes[1:] {
		if d := absInt(p - target); d < bestDist {
			best, bestDist = p, d
		}
	}
	return best
}

func absInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// findActualPipes returns the visual column positions of all │ chars in a line.
func findActualPipes(line string) []int {
	var pipes []int
	col := 0
	for _, r := range line {
		if r == '│' {
			pipes = append(pipes, col)
		}
		col += RuneWidthOf(r)
	}
	return pipes
}
