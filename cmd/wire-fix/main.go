package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"strings"

	wirediagram "github.com/shapestone/flow-wire-diagram"
)

// run executes the wire-fix logic and returns an exit code.
// Separating it from main() makes the logic directly testable.
func run(argv []string) int {
	fs := flag.NewFlagSet("wire-fix", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	var (
		output     = fs.String("o", "", "write to a different output file (default: in-place)")
		verify     = fs.Bool("v", false, "verify only, don't modify (exit 0=ok, 1=broken)")
		diff       = fs.Bool("d", false, "show diff of changes")
		ascii      = fs.Bool("a", false, "convert box-drawing Unicode to safe ASCII equivalents")
		checkWidth = fs.Bool("w", false, "scan for chars where visual width != 1 (report only)")
		verbose    = fs.Bool("verbose", false, "show per-line repair details")
	)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: wire-fix [options] <input.md>")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Options:")
		fs.PrintDefaults()
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Exit codes:")
		fmt.Fprintln(os.Stderr, "  0  Success (or --verify found no issues)")
		fmt.Fprintln(os.Stderr, "  1  --verify found defects")
		fmt.Fprintln(os.Stderr, "  2  Error reading/writing files")
	}

	if err := fs.Parse(argv); err != nil {
		return 2
	}

	args := fs.Args()
	if len(args) == 0 {
		fs.Usage()
		return 2
	}

	inputFile := args[0]
	input, err := os.ReadFile(inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: cannot read %s: %v\n", inputFile, err)
		return 2
	}

	// --check-width: scan for problematic characters and report.
	if *checkWidth {
		blocks := wirediagram.ExtractBlocks(string(input))
		found := false
		for _, block := range blocks {
			if block.Kind != wirediagram.BlockDiagram {
				continue
			}
			for lineIdx, line := range block.Lines {
				wide := wirediagram.DetectWideChars(line)
				if len(wide) > 0 {
					found = true
					for _, r := range wide {
						fmt.Printf("line %d: wide char U+%04X (%c) width=%d\n",
							lineIdx, r, r, wirediagram.RuneWidthOf(r))
					}
				}
			}
		}
		if !found {
			fmt.Println("No wide characters found in diagram blocks.")
		}
		return 0
	}

	opts := wirediagram.Options{
		ASCII: *ascii,
	}

	// --verify mode.
	if *verify {
		result, err := wirediagram.VerifyFile(input)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			return 2
		}
		if *verbose || result.DiagramsRepaired > 0 {
			fmt.Printf("Diagrams: %d found, %d OK, %d with defects\n",
				result.DiagramsFound, result.DiagramsOK, result.DiagramsRepaired)
		}
		for _, w := range result.Warnings {
			fmt.Println(w)
		}
		if result.DiagramsRepaired > 0 {
			return 1
		}
		return 0
	}

	// Repair mode.
	outputBytes, result, err := wirediagram.RepairFile(input, opts)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 2
	}

	if *verbose {
		fmt.Printf("Diagrams: %d found, %d repaired, %d already OK\n",
			result.DiagramsFound, result.DiagramsRepaired, result.DiagramsOK)
	}

	for _, w := range result.Warnings {
		fmt.Fprintln(os.Stderr, "warning:", w)
	}

	// --diff: print line-by-line diff and exit without writing.
	if *diff {
		printDiff(string(input), string(outputBytes))
		return 0
	}

	// Write output.
	outFile := inputFile
	if *output != "" {
		outFile = *output
	}

	if bytes.Equal(input, outputBytes) {
		return 0
	}

	if err := os.WriteFile(outFile, outputBytes, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "error: cannot write %s: %v\n", outFile, err)
		return 2
	}
	return 0
}

func main() {
	os.Exit(run(os.Args[1:]))
}

// printDiff prints a simple line-by-line unified-style diff.
func printDiff(original, repaired string) {
	origLines := strings.Split(original, "\n")
	repLines := strings.Split(repaired, "\n")

	maxLen := len(origLines)
	if len(repLines) > maxLen {
		maxLen = len(repLines)
	}

	hasDiff := false
	for i := 0; i < maxLen; i++ {
		o, r := "", ""
		if i < len(origLines) {
			o = origLines[i]
		}
		if i < len(repLines) {
			r = repLines[i]
		}
		if o != r {
			hasDiff = true
			fmt.Printf("@@ line %d @@\n", i+1)
			fmt.Printf("-%s\n", o)
			fmt.Printf("+%s\n", r)
		}
	}
	if !hasDiff {
		fmt.Println("(no changes)")
	}
}
