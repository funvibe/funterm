package acceptancetests

import (
	"testing"

	bitstringpkg "github.com/funvibe/funbit/internal/bitstring"
	"github.com/funvibe/funbit/internal/builder"
	"github.com/funvibe/funbit/internal/matcher"
)

// TestBasicBitStringConstruction tests basic bitstring construction functionality
func TestBasicBitStringConstruction(t *testing.T) {
	// Test empty bitstring
	empty := bitstringpkg.NewBitString()
	if empty.Length() != 0 {
		t.Errorf("Expected empty bitstring length 0, got %d", empty.Length())
	}
	if !empty.IsEmpty() {
		t.Error("Expected empty bitstring to be empty")
	}

	// Test single byte bitstring
	single := bitstringpkg.NewBitStringFromBytes([]byte{42})
	if single.Length() != 8 {
		t.Errorf("Expected single byte bitstring length 8, got %d", single.Length())
	}
	if single.IsEmpty() {
		t.Error("Expected single byte bitstring to not be empty")
	}
	if !single.IsBinary() {
		t.Error("Expected single byte bitstring to be binary")
	}

	// Test three bytes bitstring
	three := bitstringpkg.NewBitStringFromBytes([]byte{1, 2, 3})
	if three.Length() != 24 {
		t.Errorf("Expected three bytes bitstring length 24, got %d", three.Length())
	}
	if three.IsEmpty() {
		t.Error("Expected three bytes bitstring to not be empty")
	}
	if !three.IsBinary() {
		t.Error("Expected three bytes bitstring to be binary")
	}

	// Test from string
	fromString := bitstringpkg.NewBitStringFromBytes([]byte("hello"))
	if fromString.Length() != 40 { // 5 bytes * 8 bits = 40 bits
		t.Errorf("Expected string bitstring length 40, got %d", fromString.Length())
	}
	if !fromString.IsBinary() {
		t.Error("Expected string bitstring to be binary")
	}

	// Test ToBytes conversion
	bytes := three.ToBytes()
	if len(bytes) != 3 || bytes[0] != 1 || bytes[1] != 2 || bytes[2] != 3 {
		t.Errorf("Expected [1, 2, 3], got %v", bytes)
	}

	// Test Clone
	cloned := three.Clone()
	if cloned.Length() != three.Length() {
		t.Errorf("Expected clone to have same length %d, got %d", three.Length(), cloned.Length())
	}
	clonedBytes := cloned.ToBytes()
	if len(clonedBytes) != 3 || clonedBytes[0] != 1 || clonedBytes[1] != 2 || clonedBytes[2] != 3 {
		t.Errorf("Expected clone to be [1, 2, 3], got %v", clonedBytes)
	}
}

// TestBasicBuilderConstruction tests basic builder functionality
func TestBasicBuilderConstruction(t *testing.T) {
	// Test simple construction
	bs, err := builder.NewBuilder().
		AddInteger(1, bitstringpkg.WithSize(8)).
		AddInteger(17, bitstringpkg.WithSize(8)).
		AddInteger(42, bitstringpkg.WithSize(8)).
		Build()

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if bs.Length() != 24 { // 3 integers * 8 bits each = 24 bits
		t.Errorf("Expected bitstring length 24, got %d", bs.Length())
	}

	if !bs.IsBinary() {
		t.Error("Expected bitstring to be binary")
	}

	bytes := bs.ToBytes()
	if len(bytes) != 3 || bytes[0] != 1 || bytes[1] != 17 || bytes[2] != 42 {
		t.Errorf("Expected [1, 17, 42], got %v", bytes)
	}
}

// TestBasicPatternMatching tests basic pattern matching functionality
func TestBasicPatternMatching(t *testing.T) {
	// Create test bitstring
	bs := bitstringpkg.NewBitStringFromBytes([]byte{1, 17, 42})

	var a, b, c int

	// Test simple matching
	results, err := matcher.NewMatcher().
		Integer(&a, bitstringpkg.WithSize(8)).
		Integer(&b, bitstringpkg.WithSize(8)).
		Integer(&c, bitstringpkg.WithSize(8)).
		Match(bs)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("Expected 3 results, got %d", len(results))
	}

	if a != 1 || b != 17 || c != 42 {
		t.Errorf("Expected a=1, b=17, c=42, got a=%d, b=%d, c=%d", a, b, c)
	}

	// Test that all results are matched
	for i, result := range results {
		if !result.Matched {
			t.Errorf("Expected result %d to be matched", i)
		}
	}
}

// TestBasicBitstringFromBits tests bitstring creation from bits with specific length
func TestBasicBitstringFromBits(t *testing.T) {
	// Test with 4 bits (half byte)
	data := []byte{0b10110000} // Only first 4 bits matter
	bs := bitstringpkg.NewBitStringFromBits(data, 4)

	if bs.Length() != 4 {
		t.Errorf("Expected bitstring length 4, got %d", bs.Length())
	}

	if bs.IsBinary() {
		t.Error("Expected 4-bit bitstring to not be binary (not multiple of 8)")
	}

	// Test with exact byte boundary
	bs2 := bitstringpkg.NewBitStringFromBits([]byte{0xFF, 0xAA}, 16)
	if bs2.Length() != 16 {
		t.Errorf("Expected bitstring length 16, got %d", bs2.Length())
	}

	if !bs2.IsBinary() {
		t.Error("Expected 16-bit bitstring to be binary")
	}
}
