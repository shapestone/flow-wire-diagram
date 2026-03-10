// Package application provides the use-case orchestrators for diagram repair.
package application

import (
	"fmt"

	"github.com/shapestone/flow-wire-diagram/internal/diagram/domain"
	"github.com/shapestone/flow-wire-diagram/internal/diagram/infrastructure"
)

// Options controls the behaviour of RepairFile.
type Options struct {
	ASCII bool // convert box-drawing Unicode to safe ASCII
}

// Result summarises what RepairFile or VerifyFile found.
type Result struct {
	DiagramsFound    int
	DiagramsRepaired int // repaired (RepairFile) or has defects (VerifyFile)
	DiagramsOK       int
	Warnings         []string
}

// RepairFile takes markdown bytes, repairs all box diagrams, and returns
// the fixed bytes together with a summary.
func RepairFile(input []byte, opts Options) ([]byte, Result, error) {
	content := string(input)
	blocks := infrastructure.ExtractBlocks(content)

	var result Result

	for i, block := range blocks {
		if block.Kind != domain.BlockDiagram {
			continue
		}
		result.DiagramsFound++

		// Normalise tabs to 2 spaces before parsing. Tabs have no defined
		// visual width in diagrams and are treated as erroneous whitespace.
		lines := infrastructure.ExpandTabs(block.Lines)

		_, diagLines, err := infrastructure.ParseDiagram(lines)
		if err != nil {
			result.Warnings = append(result.Warnings, "parse error: "+err.Error())
			continue
		}

		repairedLines, err := infrastructure.RepairLines(diagLines, nil)
		if err != nil {
			result.Warnings = append(result.Warnings, "repair error: "+err.Error())
			continue
		}

		// Per-line safety: if repair would alter a line's text content, revert
		// that line to the original.  This allows the rest of the block to be
		// repaired even when individual lines cannot be safely reformatted.
		for j := range repairedLines {
			if j < len(lines) {
				if infrastructure.TextContent([]string{lines[j]}) != infrastructure.TextContent([]string{repairedLines[j]}) {
					repairedLines[j] = lines[j]
				}
			}
		}

		// Block-level safety check: should always pass after per-line safety.
		if infrastructure.TextContent(lines) != infrastructure.TextContent(repairedLines) {
			result.Warnings = append(result.Warnings, fmt.Sprintf("diagram %d: repair aborted, text content would be altered", result.DiagramsFound))
			result.DiagramsOK++
			continue
		}

		changed := false
		for j, line := range repairedLines {
			if j < len(lines) && line != lines[j] {
				changed = true
				break
			}
		}

		if changed {
			result.DiagramsRepaired++
		} else {
			result.DiagramsOK++
		}

		if opts.ASCII {
			for j, line := range repairedLines {
				repairedLines[j] = infrastructure.ConvertToASCII(line)
			}
			changed = true
		}

		blocks[i].Lines = repairedLines
	}

	output := infrastructure.ReconstructContent(blocks)
	return []byte(output), result, nil
}

// VerifyFile checks all box diagrams in the markdown without modifying anything.
// Result.DiagramsRepaired counts diagrams that have defects.
func VerifyFile(input []byte) (Result, error) {
	content := string(input)
	blocks := infrastructure.ExtractBlocks(content)

	var result Result

	for _, block := range blocks {
		if block.Kind != domain.BlockDiagram {
			continue
		}
		result.DiagramsFound++

		boxes, _, err := infrastructure.ParseDiagram(block.Lines)
		if err != nil {
			result.Warnings = append(result.Warnings, "parse error: "+err.Error())
			continue
		}

		errs := infrastructure.VerifyDiagram(block.Lines, boxes)
		if len(errs) > 0 {
			result.DiagramsRepaired++
			for _, e := range errs {
				result.Warnings = append(result.Warnings, e.Error())
			}
		} else {
			result.DiagramsOK++
		}
	}

	return result, nil
}
