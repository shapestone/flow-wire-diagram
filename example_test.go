package wirediagram_test

import (
	"fmt"
	"log"

	wirediagram "github.com/shapestone/flow-wire-diagram"
)

// ExampleRepairFile demonstrates reading markdown bytes, repairing box diagrams,
// and inspecting the result summary.
func ExampleRepairFile() {
	input := []byte("```ascii\n┌──────────┐\n│  Hello   │\n└──────────┘\n```\n")
	_, result, err := wirediagram.RepairFile(input, wirediagram.Options{})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("found: %d, repaired: %d, ok: %d\n",
		result.DiagramsFound, result.DiagramsRepaired, result.DiagramsOK)
	// Output:
	// found: 1, repaired: 0, ok: 1
}

// ExampleVerifyFile demonstrates checking diagrams for defects without modifying
// the input. result.DiagramsRepaired counts diagrams that have defects.
func ExampleVerifyFile() {
	input := []byte("```ascii\n┌──────────┐\n│  Hello   │\n└──────────┘\n```\n")
	result, err := wirediagram.VerifyFile(input)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("defects: %d\n", result.DiagramsRepaired)
	// Output:
	// defects: 0
}

// ExampleRepairFile_ascii demonstrates the ASCII option, which converts
// box-drawing Unicode characters (┌│─┘) to plain ASCII (+, -, |) in the output.
func ExampleRepairFile_ascii() {
	input := []byte("```ascii\n┌──────────┐\n│  Hello   │\n└──────────┘\n```\n")
	_, result, err := wirediagram.RepairFile(input, wirediagram.Options{ASCII: true})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("found: %d\n", result.DiagramsFound)
	// Output:
	// found: 1
}
