package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	wirediagram "github.com/shapestone/flow-wire-diagram"
)

// version is set at build time via -ldflags "-X main.version=<tag>".
var version = "dev"

// skipDirs is the set of directory names to skip during recursive walks.
var skipDirs = map[string]bool{
	"node_modules": true,
	".git":         true,
	"vendor":       true,
	".cache":       true,
	"dist":         true,
	".next":        true,
	".nuxt":        true,
}

// run executes the wire-fix logic and returns an exit code.
// Separating it from main() makes the logic directly testable.
func run(argv []string) int {
	fs := flag.NewFlagSet("wire-fix", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	var (
		output      = fs.String("o", "", "write to a different output file (default: in-place)")
		verify      = fs.Bool("v", false, "verify only, don't modify (exit 0=ok, 1=broken)")
		diff        = fs.Bool("d", false, "show diff of changes")
		ascii       = fs.Bool("a", false, "convert box-drawing Unicode to safe ASCII equivalents")
		checkWidth  = fs.Bool("w", false, "scan for chars where visual width != 1 (report only)")
		verbose     = fs.Bool("verbose", false, "show per-line repair details")
		scan        = fs.String("scan", "", "recursively scan `dir` for .md files and report diagram defects (read-only)")
		fix         = fs.String("fix", "", "recursively scan `dir` for .md files and repair diagram defects in-place")
		showVersion = fs.Bool("version", false, "print version and exit")
	)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: wire-fix [options] <input.md>")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Options:")
		fmt.Fprintln(os.Stderr, "  -a              convert box-drawing Unicode to safe ASCII equivalents")
		fmt.Fprintln(os.Stderr, "  -d              show diff of changes")
		fmt.Fprintln(os.Stderr, "  -o string       write to a different output file (default: in-place)")
		fmt.Fprintln(os.Stderr, "  -v              verify only, don't modify (exit 0=ok, 1=broken)")
		fmt.Fprintln(os.Stderr, "  -w              scan for chars where visual width != 1 (report only)")
		fmt.Fprintln(os.Stderr, "  --scan dir      recursively scan dir for .md files and report diagram defects (read-only)")
		fmt.Fprintln(os.Stderr, "  --fix dir       recursively scan dir for .md files and repair diagram defects in-place")
		fmt.Fprintln(os.Stderr, "  --verbose       show per-line repair details")
		fmt.Fprintln(os.Stderr, "  --version       print version and exit")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Exit codes:")
		fmt.Fprintln(os.Stderr, "  0  Success (or -v found no issues)")
		fmt.Fprintln(os.Stderr, "  1  -v or --scan found defects")
		fmt.Fprintln(os.Stderr, "  2  Error reading/writing files")
	}

	if err := fs.Parse(argv); err != nil {
		return 2
	}

	if *showVersion {
		fmt.Println(version)
		return 0
	}

	if *scan != "" {
		return runScan(*scan, *verbose)
	}

	if *fix != "" {
		return runFix(*fix, *verbose)
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

// runScan walks dir recursively, verifies every .md file, and prints a report.
// It never modifies any file. Returns 0 (all OK), 1 (defects found), 2 (I/O error).
func runScan(dir string, verbose bool) int {
	info, err := os.Stat(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: cannot access %s: %v\n", dir, err)
		return 2
	}
	if !info.IsDir() {
		fmt.Fprintf(os.Stderr, "error: %s is not a directory\n", dir)
		return 2
	}

	fmt.Printf("Scanning: %s\n\n", dir)

	type fileResult struct {
		path     string
		found    int
		ok       int
		defects  int
		warnings []string
		readErr  error
	}

	var results []fileResult

	walkErr := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if skipDirs[d.Name()] {
				return fs.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
			return nil
		}
		input, readErr := os.ReadFile(path)
		if readErr != nil {
			results = append(results, fileResult{path: path, readErr: readErr})
			return nil
		}
		result, _ := wirediagram.VerifyFile(input)
		results = append(results, fileResult{
			path:     path,
			found:    result.DiagramsFound,
			ok:       result.DiagramsOK,
			defects:  result.DiagramsRepaired,
			warnings: result.Warnings,
		})
		return nil
	})

	if walkErr != nil {
		fmt.Fprintf(os.Stderr, "error: scan failed: %v\n", walkErr)
		return 2
	}

	var passCount, failCount, skipCount, errCount int
	hasDefects := false

	for _, r := range results {
		switch {
		case r.readErr != nil:
			errCount++
			fmt.Printf("ERROR %s (%v)\n", r.path, r.readErr)
		case r.found == 0:
			skipCount++
			if verbose {
				fmt.Printf("SKIP  %s (no diagrams)\n", r.path)
			}
		case r.defects > 0:
			failCount++
			hasDefects = true
			fmt.Printf("FAIL  %s (%d diagram(s), %d with defects)\n", r.path, r.found, r.defects)
		default:
			passCount++
			if verbose {
				fmt.Printf("PASS  %s (%d diagram(s), %d OK)\n", r.path, r.found, r.ok)
			}
		}
	}

	fmt.Printf("\n%s\n", strings.Repeat("─", 62))
	fmt.Printf("Files scanned: %d  │  PASS: %d  │  FAIL: %d  │  No diagrams: %d",
		len(results), passCount, failCount, skipCount)
	if errCount > 0 {
		fmt.Printf("  │  Errors: %d", errCount)
	}
	fmt.Println()

	if hasDefects || errCount > 0 {
		return 1
	}
	return 0
}

// runFix walks dir recursively, repairs every .md file that has defects,
// and writes the result back in-place. Returns 0 (all OK), 2 (I/O error).
func runFix(dir string, verbose bool) int {
	info, err := os.Stat(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: cannot access %s: %v\n", dir, err)
		return 2
	}
	if !info.IsDir() {
		fmt.Fprintf(os.Stderr, "error: %s is not a directory\n", dir)
		return 2
	}

	fmt.Printf("Fixing: %s\n\n", dir)

	type fileResult struct {
		path    string
		fixed   int
		ok      int
		found   int
		readErr error
	}

	var results []fileResult

	walkErr := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if skipDirs[d.Name()] {
				return fs.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
			return nil
		}
		input, readErr := os.ReadFile(path)
		if readErr != nil {
			results = append(results, fileResult{path: path, readErr: readErr})
			return nil
		}
		output, result, _ := wirediagram.RepairFile(input, wirediagram.Options{})
		if result.DiagramsRepaired > 0 {
			if writeErr := os.WriteFile(path, output, 0o644); writeErr != nil {
				fmt.Fprintf(os.Stderr, "error: cannot write %s: %v\n", path, writeErr)
				results = append(results, fileResult{path: path, readErr: writeErr})
				return nil
			}
		}
		results = append(results, fileResult{
			path:  path,
			found: result.DiagramsFound,
			fixed: result.DiagramsRepaired,
			ok:    result.DiagramsOK,
		})
		return nil
	})

	if walkErr != nil {
		fmt.Fprintf(os.Stderr, "error: fix failed: %v\n", walkErr)
		return 2
	}

	var fixedCount, passCount, skipCount, errCount int

	for _, r := range results {
		switch {
		case r.readErr != nil:
			errCount++
			fmt.Printf("ERROR %s (%v)\n", r.path, r.readErr)
		case r.found == 0:
			skipCount++
			if verbose {
				fmt.Printf("SKIP  %s (no diagrams)\n", r.path)
			}
		case r.fixed > 0:
			fixedCount++
			fmt.Printf("FIXED %s (%d diagram(s) repaired)\n", r.path, r.fixed)
		default:
			passCount++
			if verbose {
				fmt.Printf("PASS  %s (%d diagram(s), %d OK)\n", r.path, r.found, r.ok)
			}
		}
	}

	fmt.Printf("\n%s\n", strings.Repeat("─", 62))
	fmt.Printf("Files scanned: %d  │  PASS: %d  │  FIXED: %d  │  No diagrams: %d",
		len(results), passCount, fixedCount, skipCount)
	if errCount > 0 {
		fmt.Printf("  │  Errors: %d", errCount)
	}
	fmt.Println()

	if errCount > 0 {
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
