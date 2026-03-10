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
// widenInnerBoxesFromTopFrames adjusts the RightCol of the rightmost inner
// box on each top-frame line, but only when that box has content that actually
// overflows its current width.  The widening preserves the trailing gap seen
// in the original line between the inner box's ┐ and the outer │.
func widenInnerBoxesFromTopFrames(diagLines []domain.DiagramLine) {
	for _, dl := range diagLines {
		if dl.Role != domain.RoleTopFrame || len(dl.ActiveBoxes) == 0 || countRootBoxes(dl) != 1 {
			continue
		}
		outermost := dl.ActiveBoxes[0]

		// Find the rightmost inner box that owns this top frame line.
		var rightmostOwn *domain.Box
		for _, b := range dl.ActiveBoxes {
			if b == outermost || dl.Index != b.TopLine {
				continue
			}
			if rightmostOwn == nil || b.RightCol > rightmostOwn.RightCol {
				rightmostOwn = b
			}
		}
		if rightmostOwn == nil {
			continue
		}

		// Only apply widening when the inner box has content that overflows
		// its current segment width.  Without overflow, the existing repair
		// logic already produces correct output and no widening is needed.
		innerSegWidth := rightmostOwn.RightCol - rightmostOwn.LeftCol - 1
		hasOverflow := false
		for _, cdl := range diagLines {
			if cdl.Role != domain.RoleContent {
				continue
			}
			if cdl.Index <= rightmostOwn.TopLine || cdl.Index >= rightmostOwn.BottomLine {
				continue
			}
			actualPipes := findActualPipes(cdl.Original)
			actLeft := closestPipe(actualPipes, rightmostOwn.LeftCol)
			actRight := closestPipe(actualPipes, rightmostOwn.RightCol)
			if actLeft < 0 || actRight <= actLeft {
				continue
			}
			content := extractBetweenCols(cdl.Original, actLeft+1, actRight)
			content = strings.TrimRight(content, " ")
			if StringWidth(content) > innerSegWidth {
				hasOverflow = true
				break
			}
		}
		if !hasOverflow {
			continue
		}

		// Scan the original top-frame line for the rightmost ┐ and the
		// rightmost │ (the displaced outer wall).
		origInnerRight := -1
		origOuterCol := -1
		col := 0
		for _, r := range dl.Original {
			if r == '┐' {
				origInnerRight = col
			} else if r == '│' {
				origOuterCol = col
			}
			col += RuneWidthOf(r)
		}

		if origInnerRight < 0 || origOuterCol < 0 || origOuterCol <= origInnerRight {
			continue
		}

		// Widen the inner box by the outer wall's displacement, which
		// preserves the original trailing gap at the target outer position.
		trailingGap := origOuterCol - origInnerRight - 1
		newInnerRight := outermost.RightCol - trailingGap - 1
		if newInnerRight > rightmostOwn.RightCol {
			rightmostOwn.RightCol = newInnerRight
			rightmostOwn.Width = newInnerRight - rightmostOwn.LeftCol + 1
		}
	}
}

// widenRootBoxes widens free-standing (root) boxes when their content lines
// overflow the current frame width.  This pre-processing pass runs before
// frame / content repair so that the correct (widened) RightCol is used
// when rendering each box.
func widenRootBoxes(diagLines []domain.DiagramLine) {
	for _, dl := range diagLines {
		if dl.Role != domain.RoleContent {
			continue
		}
		// Single-root widening is handled by the main repair loop; only
		// multi-root lines need pre-processing here.
		if countRootBoxes(dl) <= 1 {
			continue
		}
		actualPipes := findActualPipes(dl.Original)
		for _, b := range dl.ActiveBoxes {
			if b.Parent != nil {
				continue
			}
			actLeft := closestPipe(actualPipes, b.LeftCol)
			actRight := closestPipe(actualPipes, b.RightCol)
			if actLeft < 0 || actRight <= actLeft {
				continue
			}
			// If the actual right │ sits beyond the frame's ┐, the content
			// overflows the current box width — widen to match.
			if actRight > b.RightCol {
				b.RightCol = actRight
				b.Width = actRight - b.LeftCol + 1
			}
		}
	}
}

func RepairLines(diagLines []domain.DiagramLine, boxes []*domain.Box) ([]string, error) {
	result := make([]string, len(diagLines))

	// Widen root boxes and inner boxes before the main loop so that all
	// subsequent frame and content repairs see the correct (widened) dimensions.
	widenRootBoxes(diagLines)
	widenInnerBoxesFromTopFrames(diagLines)

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

	// Snap lone ▼/▲ connectors to align with ┬/┴ in the preceding line.
	for i := 1; i < len(result); i++ {
		result[i] = snapConnectorToTee(result[i], result[i-1])
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

// repairMultiRootFrameLine renders each root box's own frame corners and
// dashes independently, leaving the gaps between boxes untouched.
func repairMultiRootFrameLine(dl domain.DiagramLine) string {
	buf := []rune(dl.Original)
	for _, b := range dl.ActiveBoxes {
		if b.Parent != nil {
			continue
		}
		isOwnFrame := (dl.Role == domain.RoleTopFrame && dl.Index == b.TopLine) ||
			(dl.Role == domain.RoleBottomFrame && dl.Index == b.BottomLine)
		if !isOwnFrame {
			continue
		}
		for len(buf) <= b.RightCol {
			buf = append(buf, ' ')
		}
		if dl.Role == domain.RoleTopFrame {
			buf[b.LeftCol] = '┌'
			buf[b.RightCol] = '┐'
		} else {
			buf[b.LeftCol] = '└'
			buf[b.RightCol] = '┘'
		}
		for c := b.LeftCol + 1; c < b.RightCol; c++ {
			buf[c] = '─'
		}
	}
	return strings.TrimRight(string(buf), " ")
}

// repairMultiRootContentLine repairs each root box's content segment
// independently, preserving the gaps between sibling boxes.
func repairMultiRootContentLine(dl domain.DiagramLine) string {
	buf := []rune(dl.Original)
	actualPipes := findActualPipes(dl.Original)
	for _, b := range dl.ActiveBoxes {
		if b.Parent != nil {
			continue
		}
		actLeft := closestPipe(actualPipes, b.LeftCol)
		actRight := closestPipe(actualPipes, b.RightCol)
		if actLeft < 0 || actRight <= actLeft {
			continue
		}
		for len(buf) <= b.RightCol {
			buf = append(buf, ' ')
		}
		// Place walls at correct columns.
		if actLeft < len(buf) {
			buf[actLeft] = ' '
		}
		buf[b.LeftCol] = '│'
		if actRight < len(buf) {
			buf[actRight] = ' '
		}
		buf[b.RightCol] = '│'
		// Fill content between walls.
		segWidth := b.RightCol - b.LeftCol - 1
		content := extractBetweenCols(dl.Original, actLeft+1, actRight)
		content = strings.TrimRight(content, " ")
		padded := []rune(VisualPad(content, segWidth))
		for j, r := range padded {
			pos := b.LeftCol + 1 + j
			if pos < b.RightCol && pos < len(buf) {
				buf[pos] = r
			}
		}
	}
	return strings.TrimRight(string(buf), " ")
}

// repairFrameLine fixes a top (┌─┐) or bottom (└─┘) frame line.
func repairFrameLine(dl domain.DiagramLine) string {
	if len(dl.ActiveBoxes) == 0 {
		return dl.Original
	}
	if countRootBoxes(dl) > 1 {
		return repairMultiRootFrameLine(dl)
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
					// The outermost right wall may be displaced by more columns
					// than connectorAlignWindow; use a wider window so that a
					// shifted outer wall is not mistakenly preserved as a free
					// connector in the trailing gap.
					rightWindow := connectorAlignWindow
					if b == outermost {
						rightWindow = 4
					}
					if absInt(col-b.LeftCol) <= connectorAlignWindow ||
						absInt(col-b.RightCol) <= rightWindow {
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
		return repairMultiRootContentLine(dl)
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

			var padded []rune
			// Extend connector dashes up to the wall gap rather than padding with spaces.
			if strings.HasSuffix(content, "─") && StringWidth(content) < segWidth {
				cr := []rune(content)
				for StringWidth(string(cr)) < segWidth-1 {
					cr = append(cr, '─')
				}
				if StringWidth(string(cr)) < segWidth {
					cr = append(cr, ' ')
				}
				padded = cr
			} else {
				padded = []rune(VisualPad(content, segWidth))
			}
			for j, r := range padded {
				pos := leftTarget + 1 + j
				if pos < rightTarget && pos < targetWidth {
					buf[pos] = r
				}
			}
		}
	} else {
		// Actual │ count differs from expected box-wall count (e.g. extra
		// connector │ inside the box, or shifted walls).  Process all segments
		// using closestPipe to map each expected boundary to the nearest actual
		// pipe, preserving content in every segment including free-form connectors.
		for seg := 0; seg < len(pipeCols)-1; seg++ {
			leftTarget := pipeCols[seg]
			rightTarget := pipeCols[seg+1]
			segWidth := rightTarget - leftTarget - 1
			if segWidth <= 0 {
				continue
			}
			actLeft := closestPipe(actualPipes, leftTarget)
			actRight := closestPipe(actualPipes, rightTarget)
			if actLeft < 0 || actRight <= actLeft {
				continue
			}
			content := extractBetweenCols(dl.Original, actLeft+1, actRight)
			content = strings.TrimRight(content, " ")
			padded := []rune(VisualPad(content, segWidth))
			for j, r := range padded {
				pos := leftTarget + 1 + j
				if pos < rightTarget && pos < targetWidth {
					buf[pos] = r
				}
			}
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

// snapConnectorToTee repositions a lone ▼ (or ▲) in connLine to the column of ┬ (or
// ┴) found in teeLine, if such a tee exists and differs from the connector's current
// position.  Lines with multiple ▼/▲ symbols (branch splits) are left unchanged.
func snapConnectorToTee(connLine, teeLine string) string {
	// Count ▼/▲; only snap lone connectors.
	count := 0
	connCol := -1
	connRune := rune(0)
	col := 0
	for _, r := range connLine {
		if r == '▼' || r == '▲' {
			count++
			if count == 1 {
				connCol = col
				connRune = r
			}
		}
		col += RuneWidthOf(r)
	}
	if count != 1 || connCol < 0 {
		return connLine
	}
	// Find ┬/┴ position in teeLine
	teeCol := -1
	col = 0
	for _, r := range teeLine {
		if r == '┬' || r == '┴' {
			teeCol = col
			break
		}
		col += RuneWidthOf(r)
	}
	if teeCol < 0 || teeCol == connCol {
		return connLine
	}
	buf := []rune(connLine)
	if teeCol >= len(buf) || connCol >= len(buf) {
		return connLine
	}
	buf[connCol] = ' '
	buf[teeCol] = connRune
	return string(buf)
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
