package infrastructure

import (
	"strings"

	"github.com/shapestone/flow-wire-diagram/internal/diagram/domain"
)

// containsBoxDrawing returns true if the lines look like a box diagram.
// A box diagram must contain ┌ (top-left corner) and ┐ (top-right corner),
// plus at least one │ (vertical bar). This distinguishes box diagrams from
// tree diagrams (which use └ and │ but never ┌ or ┐).
func containsBoxDrawing(lines []string) bool {
	hasTopLeft := false  // ┌
	hasTopRight := false // ┐
	hasVertical := false // │
	for _, line := range lines {
		for _, r := range line {
			if r == '┌' {
				hasTopLeft = true
			}
			if r == '┐' {
				hasTopRight = true
			}
			if r == '│' {
				hasVertical = true
			}
		}
	}
	return hasTopLeft && hasTopRight && hasVertical
}

// isFenceLine returns true if the line is a markdown code fence.
func isFenceLine(line string) bool {
	t := strings.TrimLeft(line, " \t")
	return strings.HasPrefix(t, "```") || strings.HasPrefix(t, "~~~")
}

// ExtractBlocks splits markdown content into passthrough and diagram blocks.
func ExtractBlocks(content string) []domain.Block {
	lines := strings.Split(content, "\n")
	var blocks []domain.Block
	var curPassthrough []string

	flushPassthrough := func() {
		if len(curPassthrough) > 0 {
			blocks = append(blocks, domain.Block{
				Kind:  domain.BlockPassthrough,
				Lines: append([]string{}, curPassthrough...),
			})
			curPassthrough = nil
		}
	}

	i := 0
	for i < len(lines) {
		if isFenceLine(lines[i]) {
			flushPassthrough()
			fenceOpen := lines[i]
			var contentLines []string
			j := i + 1
			for j < len(lines) && !isFenceLine(lines[j]) {
				contentLines = append(contentLines, lines[j])
				j++
			}
			fenceClose := ""
			if j < len(lines) {
				fenceClose = lines[j]
				j++
			}

			kind := domain.BlockPassthrough
			if containsBoxDrawing(contentLines) {
				kind = domain.BlockDiagram
			}

			blocks = append(blocks, domain.Block{
				Kind:       kind,
				Lines:      contentLines,
				FenceOpen:  fenceOpen,
				FenceClose: fenceClose,
			})
			i = j
		} else {
			curPassthrough = append(curPassthrough, lines[i])
			i++
		}
	}

	flushPassthrough()
	return blocks
}

// ReconstructContent rebuilds full markdown from (possibly modified) blocks.
func ReconstructContent(blocks []domain.Block) string {
	var allLines []string
	for _, block := range blocks {
		if block.FenceOpen == "" && block.FenceClose == "" {
			allLines = append(allLines, block.Lines...)
		} else {
			if block.FenceOpen != "" {
				allLines = append(allLines, block.FenceOpen)
			}
			allLines = append(allLines, block.Lines...)
			if block.FenceClose != "" {
				allLines = append(allLines, block.FenceClose)
			}
		}
	}
	return strings.Join(allLines, "\n")
}
