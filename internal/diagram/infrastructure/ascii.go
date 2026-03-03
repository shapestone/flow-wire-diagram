package infrastructure

import "strings"

// asciiMap maps Unicode box-drawing characters to safe ASCII equivalents.
var asciiMap = map[rune]rune{
	'┌': '+',
	'┐': '+',
	'└': '+',
	'┘': '+',
	'─': '-',
	'│': '|',
	'•': '*',
	'→': '>',
	'▼': 'v',
	'┬': '+',
	'┴': '+',
	'├': '+',
	'┤': '+',
	'┼': '+',
	'▶': '>',
	'◀': '<',
	'▲': '^',
	'←': '<',
}

// ConvertToASCII replaces Unicode box-drawing and arrow characters with safe ASCII.
func ConvertToASCII(line string) string {
	var sb strings.Builder
	for _, r := range line {
		if ascii, ok := asciiMap[r]; ok {
			sb.WriteRune(ascii)
		} else {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

// IsASCIISafe returns true if the line contains no box-drawing Unicode characters.
func IsASCIISafe(line string) bool {
	for r := range asciiMap {
		if strings.ContainsRune(line, r) {
			return false
		}
	}
	return true
}
