// Package wirediagram provides tools for detecting and repairing alignment
// defects in ASCII box diagrams embedded in Markdown files.
package wirediagram

import (
	"github.com/shapestone/flow-wire-diagram/internal/diagram/application"
	"github.com/shapestone/flow-wire-diagram/internal/diagram/domain"
	"github.com/shapestone/flow-wire-diagram/internal/diagram/infrastructure"
)

// Options controls the behaviour of RepairFile.
type Options = application.Options

// Result summarises what RepairFile or VerifyFile found.
type Result = application.Result

// Block represents a contiguous section of the markdown.
type Block = domain.Block

// BlockKind identifies the kind of markdown block.
type BlockKind = domain.BlockKind

// Block kind constants.
const (
	BlockPassthrough = domain.BlockPassthrough
	BlockDiagram     = domain.BlockDiagram
)

// RepairFile takes markdown bytes, repairs all box diagrams, and returns
// the fixed bytes together with a summary.
func RepairFile(input []byte, opts Options) ([]byte, Result, error) {
	return application.RepairFile(input, opts)
}

// VerifyFile checks all box diagrams in the markdown without modifying anything.
// Result.DiagramsRepaired counts diagrams that have defects.
func VerifyFile(input []byte) (Result, error) {
	return application.VerifyFile(input)
}

// ExtractBlocks splits markdown content into passthrough and diagram blocks.
func ExtractBlocks(content string) []Block {
	return infrastructure.ExtractBlocks(content)
}

// DetectWideChars returns all runes in s where visual width != 1.
func DetectWideChars(s string) []rune {
	return infrastructure.DetectWideChars(s)
}

// RuneWidthOf returns the visual width of a single rune.
func RuneWidthOf(r rune) int {
	return infrastructure.RuneWidthOf(r)
}
