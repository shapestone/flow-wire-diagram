package infrastructure

import (
	"strings"
	"testing"
)

// ─────────────────────────────────────────────────────────────────────────────
// Deterministic diagram builders
//
// Each function returns a structurally perfect diagram as a slice of lines.
// These are the ground-truth inputs that manglers break and repair must fix.
// ─────────────────────────────────────────────────────────────────────────────

// buildSimpleBox returns a correct single-box diagram.
// innerWidth is the visual column count between the │ chars.
func buildSimpleBox(innerWidth int, rows []string) []string {
	top := "┌" + strings.Repeat("─", innerWidth) + "┐"
	bot := "└" + strings.Repeat("─", innerWidth) + "┘"
	out := []string{top}
	for _, r := range rows {
		out = append(out, "│"+VisualPad(r, innerWidth)+"│")
	}
	return append(out, bot)
}

// buildNestedBox returns a correct outer box containing a single inner box.
// outerInnerWidth is the content width of the outer box (between outer │ chars).
// pad is the spaces between the outer │ and the inner box corners on each side.
func buildNestedBox(outerInnerWidth, pad int, innerRows []string) []string {
	innerInnerWidth := outerInnerWidth - 2*pad - 2
	sp := strings.Repeat(" ", pad)
	outerTop := "┌" + strings.Repeat("─", outerInnerWidth) + "┐"
	outerBot := "└" + strings.Repeat("─", outerInnerWidth) + "┘"
	innerTop := "│" + sp + "┌" + strings.Repeat("─", innerInnerWidth) + "┐" + sp + "│"
	innerBot := "│" + sp + "└" + strings.Repeat("─", innerInnerWidth) + "┘" + sp + "│"
	out := []string{outerTop, innerTop}
	for _, r := range innerRows {
		out = append(out, "│"+sp+"│"+VisualPad(r, innerInnerWidth)+"│"+sp+"│")
	}
	return append(out, innerBot, outerBot)
}

// buildSideBySide returns a correct outer box containing two side-by-side inner boxes.
// pad is the spaces from the outer │ to the inner box corners on each side.
// gap is the spaces between the two inner boxes.
// cellInnerWidth is the content width of each inner box.
func buildSideBySide(pad, gap, cellInnerWidth int, leftRows, rightRows []string) []string {
	sp := strings.Repeat(" ", pad)
	gp := strings.Repeat(" ", gap)
	outerInnerWidth := pad + (cellInnerWidth + 2) + gap + (cellInnerWidth + 2) + pad
	dash := strings.Repeat("─", cellInnerWidth)
	outerTop := "┌" + strings.Repeat("─", outerInnerWidth) + "┐"
	outerBot := "└" + strings.Repeat("─", outerInnerWidth) + "┘"
	innerTops := "│" + sp + "┌" + dash + "┐" + gp + "┌" + dash + "┐" + sp + "│"
	innerBots := "│" + sp + "└" + dash + "┘" + gp + "└" + dash + "┘" + sp + "│"
	maxRows := len(leftRows)
	if len(rightRows) > maxRows {
		maxRows = len(rightRows)
	}
	out := []string{outerTop, innerTops}
	for i := 0; i < maxRows; i++ {
		l, r := "", ""
		if i < len(leftRows) {
			l = leftRows[i]
		}
		if i < len(rightRows) {
			r = rightRows[i]
		}
		out = append(out, "│"+sp+"│"+VisualPad(l, cellInnerWidth)+"│"+gp+"│"+VisualPad(r, cellInnerWidth)+"│"+sp+"│")
	}
	return append(out, innerBots, outerBot)
}

// buildBoxWithSideConnector returns a box that has a vertical │ connector
// running alongside it (at connCol) on every line — including the frame lines.
// The connector is outside the box's column range and must survive repair
// unchanged.  connCol must be < boxLeft; boxLeft = connCol + 1 + pad.
// innerWidth is the content width of the box.
func buildBoxWithSideConnector(connCol, pad, innerWidth int, rows []string) []string {
	connSp := strings.Repeat(" ", connCol)
	between := strings.Repeat(" ", pad)
	topFrame := connSp + "│" + between + "┌" + strings.Repeat("─", innerWidth) + "┐"
	botFrame := connSp + "│" + between + "└" + strings.Repeat("─", innerWidth) + "┘"
	out := []string{topFrame}
	for _, r := range rows {
		out = append(out, connSp+"│"+between+"│"+VisualPad(r, innerWidth)+"│")
	}
	return append(out, botFrame)
}

// buildConnectedBoxes returns two independent boxes stacked vertically and
// joined by a ┬/▼ connector arrow at connOffset columns from the left edge.
// innerWidth is the content width of each box.
func buildConnectedBoxes(innerWidth, connOffset int, topRows, botRows []string) []string {
	frame := strings.Repeat("─", innerWidth)
	topBot := "└" + strings.Repeat("─", connOffset) + "┬" + strings.Repeat("─", innerWidth-connOffset-1) + "┘"
	connLine := strings.Repeat(" ", connOffset+1) + "│"
	botTop := "┌" + strings.Repeat("─", connOffset) + "▼" + strings.Repeat("─", innerWidth-connOffset-1) + "┐"
	out := []string{"┌" + frame + "┐"}
	for _, r := range topRows {
		out = append(out, "│"+VisualPad(r, innerWidth)+"│")
	}
	out = append(out, topBot, connLine, botTop)
	for _, r := range botRows {
		out = append(out, "│"+VisualPad(r, innerWidth)+"│")
	}
	return append(out, "└"+frame+"┘")
}

// ─────────────────────────────────────────────────────────────────────────────
// Manglers
//
// Each mangler introduces a specific defect into an otherwise correct diagram.
// Only content-line mangles are included: shifting the right │ left or right
// by removing or adding padding spaces. These are the defects repair is
// designed to fix.
//
// Frame-only mangles are omitted: changing only one frame breaks the parser's
// corner-matching (both frames must have equal LeftCol/RightCol), so the box
// is not detected and no repair occurs — not a useful repair scenario.
// ─────────────────────────────────────────────────────────────────────────────

// mangleContentShort removes n trailing spaces before the rightmost │ on
// lineIdx, shifting the pipe leftward by n positions.
func mangleContentShort(lines []string, lineIdx, n int) []string {
	out := make([]string, len(lines))
	copy(out, lines)
	runes := []rune(out[lineIdx])
	rightPipe := -1
	for i := len(runes) - 1; i >= 0; i-- {
		if runes[i] == '│' {
			rightPipe = i
			break
		}
	}
	if rightPipe < 1 {
		return out
	}
	removed := 0
	for i := rightPipe - 1; i >= 0 && runes[i] == ' ' && removed < n; i-- {
		removed++
	}
	if removed == 0 {
		return out
	}
	out[lineIdx] = string(runes[:rightPipe-removed]) + string(runes[rightPipe:])
	return out
}

// mangleTab replaces the first interior space on lineIdx with a tab,
// simulating a user manually editing a diagram and introducing a tab.
func mangleTab(lines []string, lineIdx int) []string {
	out := make([]string, len(lines))
	copy(out, lines)
	runes := []rune(out[lineIdx])
	for i, r := range runes {
		if r == ' ' && i > 0 && i < len(runes)-1 && runes[i-1] != '│' {
			runes[i] = '\t'
			out[lineIdx] = string(runes)
			return out
		}
	}
	return out
}

// mangleContentLong inserts n extra spaces before the rightmost │ on lineIdx,
// shifting the pipe rightward by n positions.
func mangleContentLong(lines []string, lineIdx, n int) []string {
	out := make([]string, len(lines))
	copy(out, lines)
	runes := []rune(out[lineIdx])
	rightPipe := -1
	for i := len(runes) - 1; i >= 0; i-- {
		if runes[i] == '│' {
			rightPipe = i
			break
		}
	}
	if rightPipe < 0 {
		return out
	}
	extra := strings.Repeat(" ", n)
	out[lineIdx] = string(runes[:rightPipe]) + extra + string(runes[rightPipe:])
	return out
}

// mangleConnectorOffset shifts the first │ or ▼ on lineIdx by delta columns
// (positive = right, negative = left).  The surrounding spaces are
// adjusted so the rest of the line stays unchanged.  If the shift is
// impossible (no connector found, not enough leading spaces for leftward
// shift) the slice is returned unchanged.
func mangleConnectorOffset(lines []string, lineIdx, delta int) []string {
	out := make([]string, len(lines))
	copy(out, lines)
	runes := []rune(out[lineIdx])

	// Find first │ or ▼.
	pipeIdx := -1
	for i, r := range runes {
		if r == '│' || r == '▼' {
			pipeIdx = i
			break
		}
	}
	if pipeIdx < 0 {
		return out
	}

	ch := runes[pipeIdx]

	if delta > 0 {
		// Shift right: insert delta spaces before the connector.
		newRunes := make([]rune, 0, len(runes)+delta)
		newRunes = append(newRunes, runes[:pipeIdx]...)
		for i := 0; i < delta; i++ {
			newRunes = append(newRunes, ' ')
		}
		newRunes = append(newRunes, ch)
		newRunes = append(newRunes, runes[pipeIdx+1:]...)
		out[lineIdx] = string(newRunes)
	} else {
		// Shift left: remove -delta spaces immediately before the connector.
		shift := -delta
		if pipeIdx < shift {
			return out
		}
		canShift := 0
		for i := pipeIdx - 1; i >= 0 && runes[i] == ' '; i-- {
			canShift++
		}
		if canShift < shift {
			return out
		}
		newRunes := make([]rune, 0, len(runes))
		newRunes = append(newRunes, runes[:pipeIdx-shift]...)
		newRunes = append(newRunes, ch)
		newRunes = append(newRunes, runes[pipeIdx+1:]...)
		out[lineIdx] = string(newRunes)
	}
	return out
}

// ─────────────────────────────────────────────────────────────────────────────
// Invariant assertions
// ─────────────────────────────────────────────────────────────────────────────

// assertRepairInvariants checks the three post-repair invariants:
//  1. Text content (non-structural chars) is unchanged.
//  2. VerifyDiagram reports no defects.
//  3. A second repair pass produces identical output (idempotency).
func assertRepairInvariants(t *testing.T, label string, original, repaired []string) {
	t.Helper()

	// 1. Text content must not change.
	origText := TextContent(original)
	repText := TextContent(repaired)
	if origText != repText {
		t.Errorf("%s: text content changed:\n  before: %q\n   after: %q", label, origText, repText)
	}

	// 2. VerifyDiagram must pass on the repaired output.
	boxes, _, err := ParseDiagram(repaired)
	if err != nil {
		t.Errorf("%s: ParseDiagram after repair: %v", label, err)
		return
	}
	for _, e := range VerifyDiagram(repaired, boxes) {
		t.Errorf("%s: VerifyDiagram: %v", label, e)
	}

	// 3. Repair must be idempotent: repair(repair(x)) == repair(x).
	_, diagLines2, err := ParseDiagram(repaired)
	if err != nil {
		t.Errorf("%s: ParseDiagram (idempotency): %v", label, err)
		return
	}
	repaired2, err := RepairLines(diagLines2, nil)
	if err != nil {
		t.Errorf("%s: RepairLines (idempotency): %v", label, err)
		return
	}
	for i, line := range repaired {
		got := ""
		if i < len(repaired2) {
			got = repaired2[i]
		}
		if line != got {
			t.Errorf("%s: not idempotent at line %d:\n  first:  %q\n  second: %q", label, i, line, got)
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Property tests
//
// For every diagram type × every content line × every mangler:
//   build perfect diagram → mangle → repair → assert all three invariants.
// ─────────────────────────────────────────────────────────────────────────────

func TestRepairProperties(t *testing.T) {
	type diagramCase struct {
		name         string
		lines        []string
		contentLines []int // indices of lines that can be mangled
	}

	diagrams := []diagramCase{
		{
			name:         "simple",
			lines:        buildSimpleBox(14, []string{" Hello World  ", " second line  "}),
			contentLines: []int{1, 2},
		},
		{
			name:         "nested",
			lines:        buildNestedBox(18, 2, []string{" inner text ", " more inner "}),
			contentLines: []int{2, 3},
		},
		{
			name:         "side-by-side",
			lines:        buildSideBySide(2, 2, 6, []string{" Left "}, []string{"Right "}),
			contentLines: []int{2},
		},
		{
			name:         "connected",
			lines:        buildConnectedBoxes(14, 6, []string{" Box A        "}, []string{" Box B        "}),
			contentLines: []int{1, 5},
		},
		{
			// Box with a vertical │ connector at col 2 running alongside it.
			// connCol=2, pad=4, innerWidth=12: box occupies cols 7-20.
			// The │ at col 2 on every line is outside [7,20] and must survive repair.
			name:         "side-connector",
			lines:        buildBoxWithSideConnector(2, 4, 12, []string{" row one    ", " row two    "}),
			contentLines: []int{1, 2},
		},
	}

	type mangleCase struct {
		name  string
		apply func(lines []string, lineIdx int) []string
	}

	mangles := []mangleCase{
		{"short-1", func(l []string, i int) []string { return mangleContentShort(l, i, 1) }},
		{"short-2", func(l []string, i int) []string { return mangleContentShort(l, i, 2) }},
		{"long-1", func(l []string, i int) []string { return mangleContentLong(l, i, 1) }},
		{"long-2", func(l []string, i int) []string { return mangleContentLong(l, i, 2) }},
		{"tab", func(l []string, i int) []string { return mangleTab(l, i) }},
	}

	for _, d := range diagrams {
		for _, m := range mangles {
			for _, lineIdx := range d.contentLines {
				d, m, lineIdx := d, m, lineIdx
				name := d.name + "/" + m.name + "/line-" + itoa(lineIdx)
				t.Run(name, func(t *testing.T) {
					mangled := m.apply(d.lines, lineIdx)
					// Expand tabs before repair, mirroring the application layer.
					expanded := ExpandTabs(mangled)
					repaired := repairDiagram(t, expanded)
					assertRepairInvariants(t, name, expanded, repaired)
				})
			}
		}
	}

	// ── Connector-offset cases ────────────────────────────────────────────
	// These apply only to the free connector line in the "connected" diagram,
	// not to content lines.  A separate loop keeps them orthogonal to the
	// content-line mangler matrix above.
	connectorMangles := []struct {
		name  string
		apply func([]string, int) []string
	}{
		{"connector-right-1", func(l []string, i int) []string { return mangleConnectorOffset(l, i, +1) }},
		{"connector-left-1", func(l []string, i int) []string { return mangleConnectorOffset(l, i, -1) }},
	}

	// Line 3 is the free connector line in buildConnectedBoxes(14,6,...).
	connLine := 3
	for _, m := range connectorMangles {
		m := m
		t.Run("connected/"+m.name, func(t *testing.T) {
			base := buildConnectedBoxes(14, 6, []string{" Box A        "}, []string{" Box B        "})
			mangled := m.apply(base, connLine)
			expanded := ExpandTabs(mangled)
			repaired := repairDiagram(t, expanded)
			assertRepairInvariants(t, "connected/"+m.name, expanded, repaired)
		})
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Round-trip tests
//
// Content-only mangles are fully reversible: the frame defines the target
// width, so repair must restore the line to exactly the original.
// ─────────────────────────────────────────────────────────────────────────────

func TestRepairRoundTrip(t *testing.T) {
	original := buildSimpleBox(14, []string{" Hello World  ", " second line  "})

	cases := []struct {
		name   string
		mangle func([]string) []string
	}{
		{"short-1 line 1", func(l []string) []string { return mangleContentShort(l, 1, 1) }},
		{"short-2 line 1", func(l []string) []string { return mangleContentShort(l, 1, 2) }},
		{"short-1 line 2", func(l []string) []string { return mangleContentShort(l, 2, 1) }},
		{"long-1 line 1", func(l []string) []string { return mangleContentLong(l, 1, 1) }},
		{"long-2 line 1", func(l []string) []string { return mangleContentLong(l, 1, 2) }},
		{"both lines short", func(l []string) []string {
			l = mangleContentShort(l, 1, 1)
			return mangleContentShort(l, 2, 2)
		}},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			mangled := tc.mangle(original)
			repaired := repairDiagram(t, mangled)
			for i, want := range original {
				got := ""
				if i < len(repaired) {
					got = repaired[i]
				}
				if got != want {
					t.Errorf("line %d not restored:\n  want: %q\n   got: %q", i, want, got)
				}
			}
		})
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// TextContent unit tests
// ─────────────────────────────────────────────────────────────────────────────

func TestTextContent(t *testing.T) {
	cases := []struct {
		name  string
		lines []string
		want  string
	}{
		{
			name:  "strips box-drawing chars and spaces",
			lines: []string{"┌──────┐", "│ hello│", "└──────┘"},
			want:  "hello",
		},
		{
			name:  "strips interior spaces",
			lines: []string{"│  foo  bar  │"},
			want:  "foobar",
		},
		{
			name:  "preserves non-structural unicode",
			lines: []string{"│ héllo │"},
			want:  "héllo",
		},
		{
			name:  "preserves numbers and punctuation",
			lines: []string{"│ v1.2.3: ok │"},
			want:  "v1.2.3:ok",
		},
		{
			name:  "empty box is empty",
			lines: []string{"┌──┐", "│  │", "└──┘"},
			want:  "",
		},
		{
			name:  "all structural chars stripped",
			lines: []string{"─│┌┐└┘├┤┬┴┼▼   "},
			want:  "",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := TextContent(tc.lines)
			if got != tc.want {
				t.Errorf("TextContent: got %q, want %q", got, tc.want)
			}
		})
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Fuzz test
//
// Seeds the fuzzer with all four diagram types. On each iteration the fuzzer
// mutates the input and we assert:
//   - no panic
//   - text content is preserved by repair
//   - repair is idempotent
// ─────────────────────────────────────────────────────────────────────────────

func FuzzRepair(f *testing.F) {
	seeds := [][]string{
		buildSimpleBox(14, []string{" hello world "," another row "}),
		buildSimpleBox(14, []string{" Hello World  ", " second line  "}),
		buildNestedBox(18, 2, []string{" inner text  "}),
		buildSideBySide(2, 2, 6, []string{" Left "}, []string{"Right "}),
		buildConnectedBoxes(14, 6, []string{" Box A       "}, []string{" Box B       "}),
		buildBoxWithSideConnector(2, 4, 12, []string{" row one    ", " row two    "}),
	}
	for _, s := range seeds {
		f.Add(strings.Join(s, "\n"))
	}

	f.Fuzz(func(t *testing.T, input string) {
		lines := ExpandTabs(strings.Split(input, "\n"))

		_, diagLines, err := ParseDiagram(lines)
		if err != nil {
			return
		}
		repaired, err := RepairLines(diagLines, nil)
		if err != nil {
			return
		}

		// Text content must be preserved.
		if TextContent(lines) != TextContent(repaired) {
			t.Errorf("content changed:\n  before: %q\n   after: %q",
				TextContent(lines), TextContent(repaired))
		}

		// Repair must be idempotent.
		_, diagLines2, _ := ParseDiagram(repaired)
		repaired2, _ := RepairLines(diagLines2, nil)
		if strings.Join(repaired, "\n") != strings.Join(repaired2, "\n") {
			t.Errorf("repair not idempotent:\n  first:  %q\n  second: %q",
				strings.Join(repaired, "\n"), strings.Join(repaired2, "\n"))
		}
	})
}

// itoa converts an int to a string without importing strconv or fmt.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	digits := make([]byte, 0, 10)
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	if neg {
		digits = append([]byte{'-'}, digits...)
	}
	return string(digits)
}
