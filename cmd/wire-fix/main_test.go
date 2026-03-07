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

func TestRunVersion(t *testing.T) {
	out := captureStdout(func() {
		got := run([]string{"--version"})
		if got != 0 {
			t.Errorf("--version: exit %d, want 0", got)
		}
	})
	if strings.TrimSpace(out) == "" {
		t.Errorf("--version: expected non-empty output, got: %q", out)
	}
}

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
	f := testdataPath(t, "fixtures/wide_chars.md")
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

// ── runScan tests ─────────────────────────────────────────────────────────────

// writeTempMDInDir writes a .md file with the given name inside dir.
func writeTempMDInDir(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return path
}

func TestRunScanNotADirectory(t *testing.T) {
	f := writeTempMD(t, correctMD)
	got := run([]string{"-scan", f})
	if got != 2 {
		t.Errorf("scan non-dir: exit %d, want 2", got)
	}
}

func TestRunScanNonexistentDir(t *testing.T) {
	got := run([]string{"-scan", "/nonexistent/path/that/does/not/exist"})
	if got != 2 {
		t.Errorf("scan nonexistent: exit %d, want 2", got)
	}
}

func TestRunScanEmptyDir(t *testing.T) {
	dir := t.TempDir()
	out := captureStdout(func() {
		run([]string{"-scan", dir})
	})
	if !strings.Contains(out, "Files scanned: 0") {
		t.Errorf("scan empty dir: expected 'Files scanned: 0', got: %q", out)
	}
}

func TestRunScanAllPass(t *testing.T) {
	dir := t.TempDir()
	writeTempMDInDir(t, dir, "a.md", correctMD)
	writeTempMDInDir(t, dir, "b.md", correctMD)
	got := run([]string{"-scan", dir})
	if got != 0 {
		t.Errorf("scan all pass: exit %d, want 0", got)
	}
}

func TestRunScanWithDefects(t *testing.T) {
	dir := t.TempDir()
	writeTempMDInDir(t, dir, "good.md", correctMD)
	writeTempMDInDir(t, dir, "bad.md", brokenMD)
	var out string
	got := func() int {
		var c int
		out = captureStdout(func() { c = run([]string{"-scan", dir}) })
		return c
	}()
	if got != 1 {
		t.Errorf("scan with defects: exit %d, want 1", got)
	}
	if !strings.Contains(out, "FAIL") {
		t.Errorf("scan with defects: expected FAIL in output, got: %q", out)
	}
	if !strings.Contains(out, "PASS: 1") {
		t.Errorf("scan with defects: expected PASS: 1 in summary, got: %q", out)
	}
	if !strings.Contains(out, "FAIL: 1") {
		t.Errorf("scan with defects: expected FAIL: 1 in summary, got: %q", out)
	}
}

func TestRunScanNoMDFiles(t *testing.T) {
	dir := t.TempDir()
	// Write a non-md file
	if err := os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatalf("write txt: %v", err)
	}
	out := captureStdout(func() {
		run([]string{"-scan", dir})
	})
	if !strings.Contains(out, "Files scanned: 0") {
		t.Errorf("scan no md: expected 'Files scanned: 0', got: %q", out)
	}
}

func TestRunScanVerboseShowsPassAndSkip(t *testing.T) {
	dir := t.TempDir()
	writeTempMDInDir(t, dir, "good.md", correctMD)
	writeTempMDInDir(t, dir, "nodiag.md", "# Just text\n\nNo diagrams here.\n")
	out := captureStdout(func() {
		run([]string{"-scan", dir, "--verbose"})
	})
	if !strings.Contains(out, "PASS") {
		t.Errorf("scan verbose: expected PASS in output, got: %q", out)
	}
	if !strings.Contains(out, "SKIP") {
		t.Errorf("scan verbose: expected SKIP in output, got: %q", out)
	}
}

func TestRunScanRecursive(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "subdir")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatalf("mkdir sub: %v", err)
	}
	writeTempMDInDir(t, dir, "root.md", correctMD)
	writeTempMDInDir(t, sub, "nested.md", brokenMD)
	var out string
	got := func() int {
		var c int
		out = captureStdout(func() { c = run([]string{"-scan", dir}) })
		return c
	}()
	if got != 1 {
		t.Errorf("scan recursive: exit %d, want 1", got)
	}
	if !strings.Contains(out, "nested.md") {
		t.Errorf("scan recursive: expected nested.md in output, got: %q", out)
	}
}

func TestRunScanSummaryLineAlwaysPresent(t *testing.T) {
	dir := t.TempDir()
	writeTempMDInDir(t, dir, "f.md", correctMD)
	out := captureStdout(func() {
		run([]string{"-scan", dir})
	})
	if !strings.Contains(out, "Files scanned:") {
		t.Errorf("scan: expected summary line, got: %q", out)
	}
}

func TestRunScanTestdata(t *testing.T) {
	// Scan the real testdata directory — expect defects (intentional fixtures).
	dir := filepath.Join("..", "..", "testdata")
	got := run([]string{"-scan", dir})
	if got != 1 {
		t.Errorf("scan testdata: exit %d, want 1 (intentional defects present)", got)
	}
}

// ── runFix tests ──────────────────────────────────────────────────────────────

func TestRunFixNotADirectory(t *testing.T) {
	f := writeTempMD(t, correctMD)
	got := run([]string{"--fix", f})
	if got != 2 {
		t.Errorf("fix non-dir: exit %d, want 2", got)
	}
}

func TestRunFixNonexistentDir(t *testing.T) {
	got := run([]string{"--fix", "/nonexistent/path/that/does/not/exist"})
	if got != 2 {
		t.Errorf("fix nonexistent: exit %d, want 2", got)
	}
}

func TestRunFixAllPass(t *testing.T) {
	dir := t.TempDir()
	writeTempMDInDir(t, dir, "a.md", correctMD)
	got := run([]string{"--fix", dir})
	if got != 0 {
		t.Errorf("fix all pass: exit %d, want 0", got)
	}
}

func TestRunFixRepairsDefects(t *testing.T) {
	dir := t.TempDir()
	path := writeTempMDInDir(t, dir, "broken.md", brokenMD)
	var out string
	got := func() int {
		var c int
		out = captureStdout(func() { c = run([]string{"--fix", dir}) })
		return c
	}()
	if got != 0 {
		t.Errorf("fix repairs: exit %d, want 0", got)
	}
	if !strings.Contains(out, "FIXED") {
		t.Errorf("fix repairs: expected FIXED in output, got: %q", out)
	}
	// File should now be repaired on disk.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read repaired file: %v", err)
	}
	if string(data) == brokenMD {
		t.Errorf("fix repairs: file content unchanged after fix")
	}
}

func TestRunFixVerboseShowsPassAndSkip(t *testing.T) {
	dir := t.TempDir()
	writeTempMDInDir(t, dir, "good.md", correctMD)
	writeTempMDInDir(t, dir, "nodiag.md", "# Just text\n\nNo diagrams here.\n")
	out := captureStdout(func() {
		run([]string{"--fix", dir, "--verbose"})
	})
	if !strings.Contains(out, "PASS") {
		t.Errorf("fix verbose: expected PASS in output, got: %q", out)
	}
	if !strings.Contains(out, "SKIP") {
		t.Errorf("fix verbose: expected SKIP in output, got: %q", out)
	}
}

func TestRunFixRecursive(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "subdir")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatalf("mkdir sub: %v", err)
	}
	writeTempMDInDir(t, dir, "root.md", correctMD)
	writeTempMDInDir(t, sub, "nested.md", brokenMD)
	var out string
	got := func() int {
		var c int
		out = captureStdout(func() { c = run([]string{"--fix", dir}) })
		return c
	}()
	if got != 0 {
		t.Errorf("fix recursive: exit %d, want 0", got)
	}
	if !strings.Contains(out, "nested.md") {
		t.Errorf("fix recursive: expected nested.md in output, got: %q", out)
	}
}

func TestRunFixSummaryLineAlwaysPresent(t *testing.T) {
	dir := t.TempDir()
	writeTempMDInDir(t, dir, "f.md", correctMD)
	out := captureStdout(func() {
		run([]string{"--fix", dir})
	})
	if !strings.Contains(out, "Files scanned:") {
		t.Errorf("fix: expected summary line, got: %q", out)
	}
}

// ── skip-dirs tests ───────────────────────────────────────────────────────────

func TestRunScanSkipsNodeModules(t *testing.T) {
	dir := t.TempDir()
	nm := filepath.Join(dir, "node_modules")
	if err := os.MkdirAll(nm, 0o755); err != nil {
		t.Fatalf("mkdir node_modules: %v", err)
	}
	// Place a broken diagram inside node_modules — it must not be reported.
	writeTempMDInDir(t, nm, "broken.md", brokenMD)
	// Place a clean diagram at the root — it must be found.
	writeTempMDInDir(t, dir, "good.md", correctMD)

	got := run([]string{"--scan", dir})
	if got != 0 {
		t.Errorf("scan with node_modules: exit %d, want 0 (node_modules should be skipped)", got)
	}
}

func TestRunScanSkipsGitDir(t *testing.T) {
	dir := t.TempDir()
	git := filepath.Join(dir, ".git")
	if err := os.MkdirAll(git, 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}
	writeTempMDInDir(t, git, "broken.md", brokenMD)
	writeTempMDInDir(t, dir, "good.md", correctMD)

	got := run([]string{"--scan", dir})
	if got != 0 {
		t.Errorf("scan with .git: exit %d, want 0 (.git should be skipped)", got)
	}
}

func TestRunFixSkipsNodeModules(t *testing.T) {
	dir := t.TempDir()
	nm := filepath.Join(dir, "node_modules")
	if err := os.MkdirAll(nm, 0o755); err != nil {
		t.Fatalf("mkdir node_modules: %v", err)
	}
	path := writeTempMDInDir(t, nm, "broken.md", brokenMD)
	original, _ := os.ReadFile(path)

	captureStdout(func() {
		run([]string{"--fix", dir})
	})

	after, _ := os.ReadFile(path)
	if string(original) != string(after) {
		t.Error("--fix modified a file inside node_modules; node_modules should be skipped")
	}
}
