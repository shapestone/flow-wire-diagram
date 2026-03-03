package domain

import "fmt"

// Box represents a rectangular region bounded by ┌┐└┘│─
type Box struct {
	TopLine    int
	BottomLine int
	LeftCol    int // visual column of ┌/└
	RightCol   int // visual column of ┐/┘
	Width      int // RightCol - LeftCol + 1
	Parent     *Box
	Children   []*Box
}

// LineRole classifies what a line does within the diagram.
type LineRole int

const (
	RoleFree        LineRole = iota // not inside any box
	RoleTopFrame                    // ┌───┐ line
	RoleBottomFrame                 // └───┘ line
	RoleContent                     // │ content │ line
)

// DiagramLine holds analysis of a single line.
type DiagramLine struct {
	Index        int
	Original     string
	ActiveBoxes  []*Box   // outermost first
	Role         LineRole
	TrailingText string // text past outermost box's right edge
	TargetWidth  int    // expected visual width (from outermost box)
}

// BoxFrame records a top (┌┐) or bottom (└┘) frame found on a line.
type BoxFrame struct {
	Line     int
	LeftCol  int
	RightCol int
	IsTop    bool
}

// BlockKind identifies what kind of markdown text block we have.
type BlockKind int

const (
	BlockPassthrough BlockKind = iota
	BlockDiagram
)

// Block represents a contiguous section of the markdown.
type Block struct {
	Kind       BlockKind
	Lines      []string // content lines (diagrams: without fences; passthrough: all lines)
	FenceOpen  string   // opening fence line (e.g. "```" or "```text"), empty for passthrough
	FenceClose string   // closing fence line, empty for passthrough
}

// VerifyError describes a single alignment defect found during verification.
type VerifyError struct {
	Line    int
	Message string
}

func (e VerifyError) Error() string {
	return fmt.Sprintf("line %d: %s", e.Line, e.Message)
}
