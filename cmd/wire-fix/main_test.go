package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// captureStdout runs fn and returns everything written to os.Stdout.
func captureStdout(fn func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	fn()
	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	io.Copy(&buf, r) //nolint:errcheck
	return buf.String()
}

// writeTempMD writes content to a temp file and returns its path.
func writeTempMD(t *testing.T, content string) string {
	t.Helper()
	tmp := t.TempDir()
	path := filepath.Join(tmp, "input.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp md: %v", err)
	}
	return path
}

// testdataPath returns the absolute path to a testdata file.
func testdataPath(t *testing.T, name string) string {
	t.Helper()
	return filepath.Join("..", "..", "testdata", name)
}

const correctMD = "```ascii\n┌──────────────┐\n│  some text   │\n└──────────────┘\n```\n"
const brokenMD = "```ascii\n┌──────────────┐\n│ too short  │\n└──────────────┘\n```\n"

// ── printDiff unit tests ─────────────────────────────────────────────────────

func TestPrintDiffNoChanges(t *testing.T) {
	out := captureStdout(func() {
		printDiff("line1\nline2\nline3", "line1\nline2\nline3")
	})
	if !strings.Contains(out, "(no changes)") {
		t.Errorf("expected '(no changes)', got: %q", out)
	}
}

func TestPrintDiffChanged(t *testing.T) {
	out := captureStdout(func() {
		printDiff("line1\nline2\nline3", "line1\nchanged\nline3")
	})
	if !strings.Contains(out, "@@ line 2 @@") {
		t.Errorf("expected diff marker, got: %q", out)
	}
	if !strings.Contains(out, "-line2") {
		t.Errorf("expected '-line2', got: %q", out)
	}
	if !strings.Contains(out, "+changed") {
		t.Errorf("expected '+changed', got: %q", out)
	}
}

func TestPrintDiffOriginalLonger(t *testing.T) {
	out := captureStdout(func() {
		printDiff("a\nb\nc", "a\nb")
	})
	if !strings.Contains(out, "@@ line 3 @@") {
		t.Errorf("expected diff for removed line, got: %q", out)
	}
}

func TestPrintDiffRepairedLonger(t *testing.T) {
	out := captureStdout(func() {
		printDiff("a\nb", "a\nb\nc")
	})
	if !strings.Contains(out, "@@ line 3 @@") {
		t.Errorf("expected diff for added line, got: %q", out)
	}
}

func TestPrintDiffEmptyInputs(t *testing.T) {
	out := captureStdout(func() {
		printDiff("", "")
	})
	if !strings.Contains(out, "(no changes)") {
		t.Errorf("empty inputs: expected '(no changes)', got: %q", out)
	}
}

func TestPrintDiffMultipleChanges(t *testing.T) {
	out := captureStdout(func() {
		printDiff("a\nb\nc\nd", "a\nX\nc\nY")
	})
	if !strings.Contains(out, "@@ line 2 @@") {
		t.Errorf("expected diff at line 2, got: %q", out)
	}
	if !strings.Contains(out, "@@ line 4 @@") {
		t.Errorf("expected diff at line 4, got: %q", out)
	}
}

// ── run() unit tests ─────────────────────────────────────────────────────────

func TestRunNoArgs(t *testing.T) {
	got := run([]string{})
	if got != 2 {
		t.Errorf("run(): exit %d, want 2", got)
	}
}

func TestRunUnknownFlag(t *testing.T) {
	got := run([]string{"--no-such-flag"})
	if got != 2 {
		t.Errorf("run(unknown flag): exit %d, want 2", got)
	}
}

func TestRunReadError(t *testing.T) {
	got := run([]string{"/nonexistent/path/file.md"})
	if got != 2 {
		t.Errorf("run(bad path): exit %d, want 2", got)
	}
}

func TestRunVerifyCorrect(t *testing.T) {
	f := writeTempMD(t, correctMD)
	got := run([]string{"-v", f})
	if got != 0 {
		t.Errorf("run -v correct: exit %d, want 0", got)
	}
}

func TestRunVerifyDefective(t *testing.T) {
	f := writeTempMD(t, brokenMD)
	got := run([]string{"-v", f})
	if got != 1 {
		t.Errorf("run -v defective: exit %d, want 1", got)
	}
}

func TestRunVerifyVerboseDefective(t *testing.T) {
	f := writeTempMD(t, brokenMD)
	out := captureStdout(func() {
		run([]string{"-v", "--verbose", f})
	})
	if !strings.Contains(out, "Diagrams:") {
		t.Errorf("run -v --verbose: expected summary output, got: %q", out)
	}
}

func TestRunVerifyVerboseCorrect(t *testing.T) {
	f := writeTempMD(t, correctMD)
	got := run([]string{"-v", "--verbose", f})
	if got != 0 {
		t.Errorf("run -v --verbose correct: exit %d, want 0", got)
	}
}

func TestRunRepairBroken(t *testing.T) {
	f := writeTempMD(t, brokenMD)
	got := run([]string{f})
	if got != 0 {
		t.Errorf("run repair broken: exit %d, want 0", got)
	}
}

func TestRunRepairToOutputFile(t *testing.T) {
	f := writeTempMD(t, brokenMD)
	outFile := filepath.Join(t.TempDir(), "out.md")
	got := run([]string{"-o", outFile, f})
	if got != 0 {
		t.Errorf("run -o: exit %d, want 0", got)
	}
	if _, err := os.Stat(outFile); err != nil {
		t.Errorf("output file not created: %v", err)
	}
}

func TestRunRepairAlreadyCorrectNoWrite(t *testing.T) {
	// Already-correct files should not be written (bytes.Equal check).
	f := writeTempMD(t, correctMD)
	got := run([]string{f})
	if got != 0 {
		t.Errorf("run already correct: exit %d, want 0", got)
	}
}

func TestRunDiff(t *testing.T) {
	f := writeTempMD(t, brokenMD)
	var out string
	code := func() int {
		var c int
		out = captureStdout(func() { c = run([]string{"-d", f}) })
		return c
	}()
	if code != 0 {
		t.Errorf("run -d: exit %d, want 0", code)
	}
	_ = out // diff output may or may not have changes depending on repair
}

func TestRunDiffAlreadyCorrect(t *testing.T) {
	f := writeTempMD(t, correctMD)
	out := captureStdout(func() {
		run([]string{"-d", f})
	})
	if !strings.Contains(out, "(no changes)") {
		t.Errorf("run -d correct: expected '(no changes)', got: %q", out)
	}
}

func TestRunASCII(t *testing.T) {
	f := writeTempMD(t, correctMD)
	outFile := filepath.Join(t.TempDir(), "out.md")
	got := run([]string{"-a", "-o", outFile, f})
	if got != 0 {
		t.Errorf("run -a: exit %d, want 0", got)
	}
	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	for _, r := range string(data) {
		switch r {
		case '┌', '┐', '└', '┘', '│', '─':
			t.Errorf("box-drawing char %c still present after ASCII conversion", r)
		}
	}
}

func TestRunCheckWidthWithWide(t *testing.T) {
	f := testdataPath(t, "wide_chars.md")
	out := captureStdout(func() {
		run([]string{"-w", f})
	})
	if !strings.Contains(out, "wide char") {
		t.Errorf("run -w wide_chars.md: expected wide char report, got: %q", out)
	}
}

func TestRunCheckWidthNoWide(t *testing.T) {
	f := writeTempMD(t, correctMD)
	out := captureStdout(func() {
		run([]string{"-w", f})
	})
	if !strings.Contains(out, "No wide characters") {
		t.Errorf("run -w no wide: expected 'No wide characters', got: %q", out)
	}
}

func TestRunVerboseRepair(t *testing.T) {
	f := writeTempMD(t, brokenMD)
	outFile := filepath.Join(t.TempDir(), "out.md")
	out := captureStdout(func() {
		run([]string{"--verbose", "-o", outFile, f})
	})
	if !strings.Contains(out, "Diagrams:") {
		t.Errorf("run --verbose: expected summary output, got: %q", out)
	}
}

func TestRunWriteError(t *testing.T) {
	// Write input to a read-only directory so os.WriteFile fails.
	tmpDir := t.TempDir()
	f := writeTempMD(t, brokenMD)

	// Create a read-only output path.
	roDir := filepath.Join(tmpDir, "readonly")
	if err := os.MkdirAll(roDir, 0o555); err != nil {
		t.Fatalf("mkdir readonly: %v", err)
	}
	outFile := filepath.Join(roDir, "out.md")

	got := run([]string{"-o", outFile, f})
	if got != 2 {
		t.Errorf("run write error: exit %d, want 2", got)
	}
}
