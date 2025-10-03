package matcher

import (
	"testing"

	bitstringpkg "github.com/funvibe/funbit/internal/bitstring"
)

func TestMatcher_RestBitstringBugNotAligned(t *testing.T) {
	// Create bitstring: <<1:3, 2:5, 3:8, 4:4>> = 3+5+8+4 = 20 bits total
	var a, b uint
	var rest *bitstringpkg.BitString

	matcher := NewMatcher().
		Integer(&a, bitstringpkg.WithSize(3)).
		Integer(&b, bitstringpkg.WithSize(5)).
		RestBitstring(&rest)

	// Create test bitstring manually to match the pattern
	bitstring := bitstringpkg.NewBitStringFromBits([]byte{0b00100010, 0b00000011, 0b00000100}, 20)

	_, err := matcher.Match(bitstring)
	if err != nil {
		t.Errorf("Failed to match pattern: %v", err)
	}

	if a != 1 {
		t.Errorf("Expected a=1, got %d", a)
	}
	if b != 2 {
		t.Errorf("Expected b=2, got %d", b)
	}
	if rest == nil {
		t.Fatal("Expected 12 bits in rest, got nothing")
	}
	if rest.Length() != 12 {
		t.Errorf("Expected 12 bits in rest, got %d", rest.Length())
	}
}
