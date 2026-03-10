package infrastructure

import (
	"sort"
	"strings"

	"github.com/shapestone/flow-wire-diagram/internal/diagram/domain"
)

// ParseDiagram parses box structures from diagram content lines.
// Returns root boxes, classified lines, and any error.
func ParseDiagram(lines []string) ([]*domain.Box, []domain.DiagramLine, error) {
	frames := findAllFrames(lines)
	boxes := matchFrames(frames)
	roots := buildNestingTree(boxes)
	diagLines := classifyLines(lines, boxes)
	return roots, diagLines, nil
}

// findAllFrames scans all lines for ┌...┐ and └...┘ patterns.
func findAllFrames(lines []string) []domain.BoxFrame {
	var frames []domain.BoxFrame
	for i, line := range lines {
		frames = append(frames, findFramesOnLine(line, i)...)
	}
	return frames
}

// findFramesOnLine finds all box frame segments on a single line.
func findFramesOnLine(line string, lineIdx int) []domain.BoxFrame {
	runes, cols, _ := lineRunes(line)
	var frames []domain.BoxFrame

	i := 0
	for i < len(runes) {
		r := runes[i]

		if r == '┌' {
			startCol := cols[i]
			j := i + 1
			for j < len(runes) {
				jr := runes[j]
				if jr == '─' || jr == '┬' || jr == '▼' || jr == '┼' {
					j++
				} else {
					break
				}
			}
			if j < len(runes) && runes[j] == '┐' {
				rightCol := cols[j]
				frames = append(frames, domain.BoxFrame{
					Line:     lineIdx,
					LeftCol:  startCol,
					RightCol: rightCol,
					IsTop:    true,
				})
				i = j + 1
				continue
			}
		} else if r == '└' {
			startCol := cols[i]
			j := i + 1
			for j < len(runes) {
				jr := runes[j]
				if jr == '─' || jr == '┬' || jr == '┴' || jr == '▼' || jr == '┼' {
					j++
				} else {
					break
				}
			}
			if j < len(runes) && runes[j] == '┘' {
				rightCol := cols[j]
				frames = append(frames, domain.BoxFrame{
					Line:     lineIdx,
					LeftCol:  startCol,
					RightCol: rightCol,
					IsTop:    false,
				})
				i = j + 1
				continue
			}
		}
		i++
	}
	return frames
}

// matchFrames pairs top frames with matching bottom frames to create Boxes.
func matchFrames(frames []domain.BoxFrame) []*domain.Box {
	var tops, bottoms []domain.BoxFrame
	for _, f := range frames {
		if f.IsTop {
			tops = append(tops, f)
		} else {
			bottoms = append(bottoms, f)
		}
	}

	var boxes []*domain.Box
	usedBottom := make(map[int]bool)

	for _, top := range tops {
		bestLine := -1
		bestIdx := -1
		bestRightCol := top.RightCol
		for idx, bot := range bottoms {
			if usedBottom[idx] {
				continue
			}
			rightDiff := bot.RightCol - top.RightCol
			if rightDiff < 0 {
				rightDiff = -rightDiff
			}
			if bot.LeftCol == top.LeftCol && rightDiff <= 1 && bot.Line > top.Line {
				if bestLine == -1 || bot.Line < bestLine {
					bestLine = bot.Line
					bestIdx = idx
					if bot.RightCol > top.RightCol {
						bestRightCol = bot.RightCol
					} else {
						bestRightCol = top.RightCol
					}
				}
			}
		}
		if bestIdx >= 0 {
			usedBottom[bestIdx] = true
			bot := bottoms[bestIdx]
			boxes = append(boxes, &domain.Box{
				TopLine:    top.Line,
				BottomLine: bot.Line,
				LeftCol:    top.LeftCol,
				RightCol:   bestRightCol,
				Width:      bestRightCol - top.LeftCol + 1,
			})
		}
	}
	return boxes
}

// boxArea returns the visual area of a box (used for sorting).
func boxArea(b *domain.Box) int {
	return (b.RightCol - b.LeftCol) * (b.BottomLine - b.TopLine)
}

// buildNestingTree assigns parent-child relationships to boxes.
// Returns the root boxes (those with no parent).
func buildNestingTree(boxes []*domain.Box) []*domain.Box {
	sort.Slice(boxes, func(i, j int) bool {
		return boxArea(boxes[i]) > boxArea(boxes[j])
	})

	for _, b := range boxes {
		b.Parent = nil
		b.Children = nil
	}

	for _, b := range boxes {
		var parent *domain.Box
		for _, other := range boxes {
			if other == b {
				continue
			}
			if other.LeftCol < b.LeftCol && other.RightCol > b.RightCol &&
				other.TopLine < b.TopLine && other.BottomLine > b.BottomLine {
				if parent == nil || boxArea(other) < boxArea(parent) {
					parent = other
				}
			}
		}
		b.Parent = parent
		if parent != nil {
			parent.Children = append(parent.Children, b)
		}
	}

	var roots []*domain.Box
	for _, b := range boxes {
		if b.Parent == nil {
			roots = append(roots, b)
		}
	}
	return roots
}

// classifyLines assigns a role and active boxes to each diagram line.
func classifyLines(lines []string, boxes []*domain.Box) []domain.DiagramLine {
	diagLines := make([]domain.DiagramLine, len(lines))

	for i, line := range lines {
		dl := domain.DiagramLine{
			Index:    i,
			Original: line,
		}

		var active []*domain.Box
		for _, b := range boxes {
			if i >= b.TopLine && i <= b.BottomLine {
				active = append(active, b)
			}
		}

		sort.Slice(active, func(a, b int) bool {
			return boxArea(active[a]) > boxArea(active[b])
		})

		dl.ActiveBoxes = active

		if len(active) == 0 {
			dl.Role = domain.RoleFree
		} else {
			outermost := active[0]
			dl.TargetWidth = outermost.RightCol + 1

			isTopFrame := false
			isBottomFrame := false
			for _, b := range active {
				if i == b.TopLine {
					isTopFrame = true
				}
				if i == b.BottomLine {
					isBottomFrame = true
				}
			}

			switch {
			case isTopFrame:
				dl.Role = domain.RoleTopFrame
			case isBottomFrame:
				dl.Role = domain.RoleBottomFrame
			default:
				dl.Role = domain.RoleContent
				dl.TrailingText = findTrailingText(line, outermost.RightCol)
			}
		}

		diagLines[i] = dl
	}
	return diagLines
}

// findTrailingText returns any non-space text past the outermost right pipe column.
func findTrailingText(line string, rightCol int) string {
	runes, cols, _ := lineRunes(line)
	for i, r := range runes {
		if r == '│' && cols[i] == rightCol {
			rest := string(runes[i+1:])
			if strings.TrimSpace(rest) != "" {
				return rest
			}
			return ""
		}
	}
	return ""
}
