package application_test

import (
	"strings"
	"testing"

	"github.com/shapestone/flow-wire-diagram/internal/diagram/application"
)

const correctDiagram = "```ascii\n┌──────────────┐\n│  some text   │\n└──────────────┘\n```\n"
const brokenDiagram = "```ascii\n┌──────────────┐\n│ too short   │\n└──────────────┘\n```\n"

func TestRepairFileEmpty(t *testing.T) {
	output, result, err := application.RepairFile([]byte(""), application.Options{})
	if err != nil {
		t.Fatalf("RepairFile empty: %v", err)
	}
	if result.DiagramsFound != 0 {
		t.Errorf("DiagramsFound: want 0, got %d", result.DiagramsFound)
	}
	if string(output) != "" {
		t.Errorf("output: want empty, got %q", string(output))
	}
}

func TestRepairFileNoDiagrams(t *testing.T) {
	input := "# Heading\nSome text without diagrams.\n"
	output, result, err := application.RepairFile([]byte(input), application.Options{})
	if err != nil {
		t.Fatalf("RepairFile no diagrams: %v", err)
	}
	if result.DiagramsFound != 0 {
		t.Errorf("DiagramsFound: want 0, got %d", result.DiagramsFound)
	}
	if string(output) != input {
		t.Errorf("passthrough content changed")
	}
}

func TestRepairFileAlreadyCorrect(t *testing.T) {
	_, result, err := application.RepairFile([]byte(correctDiagram), application.Options{})
	if err != nil {
		t.Fatalf("RepairFile: %v", err)
	}
	if result.DiagramsFound != 1 {
		t.Errorf("DiagramsFound: want 1, got %d", result.DiagramsFound)
	}
	if result.DiagramsOK != 1 {
		t.Errorf("DiagramsOK: want 1, got %d", result.DiagramsOK)
	}
	if result.DiagramsRepaired != 0 {
		t.Errorf("DiagramsRepaired: want 0, got %d", result.DiagramsRepaired)
	}
}

func TestRepairFileBrokenDiagram(t *testing.T) {
	_, result, err := application.RepairFile([]byte(brokenDiagram), application.Options{})
	if err != nil {
		t.Fatalf("RepairFile: %v", err)
	}
	if result.DiagramsFound != 1 {
		t.Errorf("DiagramsFound: want 1, got %d", result.DiagramsFound)
	}
	if result.DiagramsRepaired != 1 {
		t.Errorf("DiagramsRepaired: want 1, got %d", result.DiagramsRepaired)
	}
}

func TestRepairFileASCIIOption(t *testing.T) {
	output, _, err := application.RepairFile([]byte(correctDiagram), application.Options{ASCII: true})
	if err != nil {
		t.Fatalf("RepairFile ASCII: %v", err)
	}
	for _, r := range string(output) {
		switch r {
		case '┌', '┐', '└', '┘', '│', '─':
			t.Errorf("box-drawing char %c still present after ASCII conversion", r)
		}
	}
}

func TestRepairFileMultipleDiagrams(t *testing.T) {
	input := correctDiagram + "\n" + brokenDiagram
	_, result, err := application.RepairFile([]byte(input), application.Options{})
	if err != nil {
		t.Fatalf("RepairFile multiple: %v", err)
	}
	if result.DiagramsFound != 2 {
		t.Errorf("DiagramsFound: want 2, got %d", result.DiagramsFound)
	}
	if result.DiagramsOK != 1 {
		t.Errorf("DiagramsOK: want 1, got %d", result.DiagramsOK)
	}
	if result.DiagramsRepaired != 1 {
		t.Errorf("DiagramsRepaired: want 1, got %d", result.DiagramsRepaired)
	}
}

func TestRepairFileWithTabs(t *testing.T) {
	input := "```ascii\n┌──────────────┐\n│\ttext       │\n└──────────────┘\n```\n"
	_, result, err := application.RepairFile([]byte(input), application.Options{})
	if err != nil {
		t.Fatalf("RepairFile with tabs: %v", err)
	}
	if result.DiagramsFound != 1 {
		t.Errorf("DiagramsFound: want 1, got %d", result.DiagramsFound)
	}
}

func TestRepairFileWithMixedContent(t *testing.T) {
	input := "# Title\n\n```go\nfunc main() {}\n```\n\n" + correctDiagram + "\nMore text.\n"
	output, result, err := application.RepairFile([]byte(input), application.Options{})
	if err != nil {
		t.Fatalf("RepairFile mixed: %v", err)
	}
	if result.DiagramsFound != 1 {
		t.Errorf("DiagramsFound: want 1, got %d", result.DiagramsFound)
	}
	// Passthrough content must be preserved.
	if !strings.Contains(string(output), "func main()") {
		t.Error("go code block content was lost")
	}
	if !strings.Contains(string(output), "# Title") {
		t.Error("heading was lost")
	}
}

func TestVerifyFileEmpty(t *testing.T) {
	result, err := application.VerifyFile([]byte(""))
	if err != nil {
		t.Fatalf("VerifyFile empty: %v", err)
	}
	if result.DiagramsFound != 0 {
		t.Errorf("DiagramsFound: want 0, got %d", result.DiagramsFound)
	}
}

func TestVerifyFileNoDiagrams(t *testing.T) {
	result, err := application.VerifyFile([]byte("# Heading\nSome text.\n"))
	if err != nil {
		t.Fatalf("VerifyFile no diagrams: %v", err)
	}
	if result.DiagramsFound != 0 {
		t.Errorf("DiagramsFound: want 0, got %d", result.DiagramsFound)
	}
}

func TestVerifyFileCorrect(t *testing.T) {
	result, err := application.VerifyFile([]byte(correctDiagram))
	if err != nil {
		t.Fatalf("VerifyFile: %v", err)
	}
	if result.DiagramsFound != 1 {
		t.Errorf("DiagramsFound: want 1, got %d", result.DiagramsFound)
	}
	if result.DiagramsOK != 1 {
		t.Errorf("DiagramsOK: want 1, got %d", result.DiagramsOK)
	}
	if result.DiagramsRepaired != 0 {
		t.Errorf("DiagramsRepaired (defects): want 0, got %d; warnings: %v",
			result.DiagramsRepaired, result.Warnings)
	}
}

func TestVerifyFileDefective(t *testing.T) {
	result, err := application.VerifyFile([]byte(brokenDiagram))
	if err != nil {
		t.Fatalf("VerifyFile: %v", err)
	}
	if result.DiagramsFound != 1 {
		t.Errorf("DiagramsFound: want 1, got %d", result.DiagramsFound)
	}
	if result.DiagramsRepaired != 1 {
		t.Errorf("DiagramsRepaired (defects): want 1, got %d", result.DiagramsRepaired)
	}
	if len(result.Warnings) == 0 {
		t.Error("expected warnings for defective diagram")
	}
}

func TestVerifyFileMultipleDiagrams(t *testing.T) {
	input := correctDiagram + "\n" + brokenDiagram
	result, err := application.VerifyFile([]byte(input))
	if err != nil {
		t.Fatalf("VerifyFile multiple: %v", err)
	}
	if result.DiagramsFound != 2 {
		t.Errorf("DiagramsFound: want 2, got %d", result.DiagramsFound)
	}
	if result.DiagramsOK != 1 {
		t.Errorf("DiagramsOK: want 1, got %d", result.DiagramsOK)
	}
	if result.DiagramsRepaired != 1 {
		t.Errorf("DiagramsRepaired: want 1, got %d", result.DiagramsRepaired)
	}
}
