package matcher

import (
	"encoding/binary"
	"math"
	"testing"

	"github.com/funvibe/funbit/internal/bitstring"
	bitstringpkg "github.com/funvibe/funbit/internal/bitstring"
)

func TestMatcher_NewMatcher(t *testing.T) {
	m := NewMatcher()

	if m == nil {
		t.Fatal("Expected NewMatcher() to return non-nil")
	}
}

func TestMatcher_Integer(t *testing.T) {
	m := NewMatcher()

	var x int

	// Test that Integer returns the matcher for chaining
	result := m.Integer(&x)
	if result != m {
		t.Error("Expected Integer() to return the same matcher instance")
	}

	// Test multiple additions for chaining
	var a, b, c int
	m2 := m.
		Integer(&a).
		Integer(&b).
		Integer(&c)

	if m2 != m {
		t.Error("Expected chaining to work correctly")
	}
}

func TestMatcher_Match(t *testing.T) {
	// This test will fail until we implement the bitstring package
	// For now, we'll test the basic structure

	var a, b, c int

	m := NewMatcher().
		Integer(&a).
		Integer(&b).
		Integer(&c)

	// This should fail because we don't have a bitstring yet
	// but we can test the method exists and returns appropriate error types
	results, err := m.Match(nil)

	if err == nil {
		t.Error("Expected error when matching nil bitstring")
	}

	if results != nil {
		t.Error("Expected nil results when matching nil bitstring")
	}
}

func TestMatcher_MatchVariables(t *testing.T) {
	// Test that variables are properly bound
	var x, y, z int

	m := NewMatcher().
		Integer(&x).
		Integer(&y).
		Integer(&z)

	// Initial values should be zero
	if x != 0 || y != 0 || z != 0 {
		t.Errorf("Expected initial values to be zero, got x=%d, y=%d, z=%d", x, y, z)
	}

	// After matching with nil, values should still be zero (no change on error)
	_, err := m.Match(nil)
	if err == nil {
		t.Error("Expected error when matching nil bitstring")
	}

	if x != 0 || y != 0 || z != 0 {
		t.Errorf("Expected values to remain zero after failed match, got x=%d, y=%d, z=%d", x, y, z)
	}
}

func TestMatcher_EmptyMatcher(t *testing.T) {
	m := NewMatcher()

	// Empty matcher should match empty bitstring
	// This will fail until we implement bitstring package
	results, err := m.Match(nil)

	if err == nil {
		t.Error("Expected error when matching nil bitstring with empty matcher")
	}

	if results != nil {
		t.Error("Expected nil results when matching nil bitstring")
	}
}

func TestMatcher_Chaining(t *testing.T) {
	var a, b, c int

	// Test that chaining works properly
	m := NewMatcher()
	result1 := m.Integer(&a)
	result2 := result1.Integer(&b)
	result3 := result2.Integer(&c)

	// All should return the same instance
	if result1 != m || result2 != m || result3 != m {
		t.Error("Expected all chained methods to return the same matcher instance")
	}

	// Variables should be properly set up
	if a != 0 || b != 0 || c != 0 {
		t.Errorf("Expected initial values to be zero, got a=%d, b=%d, c=%d", a, b, c)
	}
}

func TestMatcher_MultipleVariables(t *testing.T) {
	// Test different types of variables (for future extensibility)
	var intVar int
	var int8Var int8
	var int16Var int16
	var int32Var int32
	var int64Var int64
	var uintVar uint
	var uint8Var uint8
	var uint16Var uint16
	var uint32Var uint32
	var uint64Var uint64

	m := NewMatcher().
		Integer(&intVar).
		Integer(&int8Var).
		Integer(&int16Var).
		Integer(&int32Var).
		Integer(&int64Var).
		Integer(&uintVar).
		Integer(&uint8Var).
		Integer(&uint16Var).
		Integer(&uint32Var).
		Integer(&uint64Var)

	// Should be able to create matcher with different integer types
	if m == nil {
		t.Fatal("Expected matcher to be created successfully")
	}

	// Test with nil bitstring - should error
	_, err := m.Match(nil)
	if err == nil {
		t.Error("Expected error when matching nil bitstring")
	}
}

func TestMatcher_MatchFunctions(t *testing.T) {
	t.Run("Match with simple integer pattern", func(t *testing.T) {
		var result int
		m := NewMatcher()
		m.Integer(&result, bitstring.WithSize(16))

		// Create a bitstring with 16-bit value 0x1234
		bs := bitstringpkg.NewBitStringFromBytes([]byte{0x12, 0x34})

		matcherResults, err := m.Match(bs)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if len(matcherResults) != 1 {
			t.Errorf("Expected 1 result, got %d", len(matcherResults))
		}

		if !matcherResults[0].Matched {
			t.Error("Expected match to be successful")
		}

		if result != 0x1234 {
			t.Errorf("Expected result 0x1234, got 0x%X", result)
		}
	})

	t.Run("Match with pattern chaining", func(t *testing.T) {
		var result1, result2 int
		m := NewMatcher()
		m.Integer(&result1, bitstring.WithSize(8)).
			Integer(&result2, bitstring.WithSize(8))

		// Create a bitstring with two 8-bit values
		bs := bitstringpkg.NewBitStringFromBytes([]byte{0xAB, 0xCD})

		matcherResults, err := m.Match(bs)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if len(matcherResults) != 2 {
			t.Errorf("Expected 2 results, got %d", len(matcherResults))
		}

		if !matcherResults[0].Matched || !matcherResults[1].Matched {
			t.Error("Expected all matches to be successful")
		}

		if result1 != 0xAB || result2 != 0xCD {
			t.Errorf("Expected results 0xAB, 0xCD, got 0x%X, 0x%X", result1, result2)
		}
	})

	t.Run("Match with insufficient data", func(t *testing.T) {
		var result int
		m := NewMatcher()
		m.Integer(&result, bitstring.WithSize(16))

		// Create a bitstring with only 8 bits
		bs := bitstringpkg.NewBitStringFromBytes([]byte{0x12})

		_, err := m.Match(bs)

		if err == nil {
			t.Error("Expected error for insufficient data")
		}
	})

	t.Run("Match with float", func(t *testing.T) {
		var result float64
		m := NewMatcher()
		m.Float(&result, bitstring.WithSize(32))

		// Create a bitstring with 32-bit float value 1.5
		bytes := make([]byte, 4)
		binary.BigEndian.PutUint32(bytes, math.Float32bits(1.5))
		bs := bitstringpkg.NewBitStringFromBytes(bytes)

		matcherResults, err := m.Match(bs)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if len(matcherResults) != 1 {
			t.Errorf("Expected 1 result, got %d", len(matcherResults))
		}

		if !matcherResults[0].Matched {
			t.Error("Expected match to be successful")
		}

		if math.Abs(result-1.5) > 0.001 {
			t.Errorf("Expected result 1.5, got %f", result)
		}
	})

	t.Run("Match with binary", func(t *testing.T) {
		var result []byte
		m := NewMatcher()
		m.Binary(&result, bitstring.WithSize(2)) // 2 bytes

		// Create a bitstring with 16-bit binary data (2 bytes)
		bs := bitstringpkg.NewBitStringFromBytes([]byte{0xAB, 0xCD})

		matcherResults, err := m.Match(bs)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if len(matcherResults) != 1 {
			t.Errorf("Expected 1 result, got %d", len(matcherResults))
		}

		if !matcherResults[0].Matched {
			t.Error("Expected match to be successful")
		}

		if len(result) != 2 || result[0] != 0xAB || result[1] != 0xCD {
			t.Errorf("Expected result [0xAB, 0xCD], got %v", result)
		}
	})

	t.Run("Match with UTF-8", func(t *testing.T) {
		var result string
		m := NewMatcher()
		m.UTF(&result, bitstring.WithType("utf8")) // Use type instead of endianness for UTF

		// Create a bitstring with UTF-8 character 'A'
		bs := bitstringpkg.NewBitStringFromBytes([]byte{'A'})

		matcherResults, err := m.Match(bs)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if len(matcherResults) != 1 {
			t.Errorf("Expected 1 result, got %d", len(matcherResults))
		}

		if !matcherResults[0].Matched {
			t.Error("Expected match to be successful")
		}

		if result != "A" {
			t.Errorf("Expected result 'A', got '%s'", result)
		}
	})

	t.Run("Match with nil bitstring", func(t *testing.T) {
		var result int
		m := NewMatcher()
		m.Integer(&result)

		_, err := m.Match(nil)

		if err == nil {
			t.Error("Expected error for nil bitstring")
		}
	})

	t.Run("Match with rest binary", func(t *testing.T) {
		var result []byte
		m := NewMatcher()
		m.RestBinary(&result)

		// Create a bitstring with some data
		bs := bitstringpkg.NewBitStringFromBytes([]byte{0xAB, 0xCD, 0xEF})

		matcherResults, err := m.Match(bs)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if len(matcherResults) != 1 {
			t.Errorf("Expected 1 result, got %d", len(matcherResults))
		}

		if !matcherResults[0].Matched {
			t.Error("Expected match to be successful")
		}

		expected := []byte{0xAB, 0xCD, 0xEF}
		if len(result) != len(expected) || !bytesEqual(result, expected) {
			t.Errorf("Expected result %v, got %v", expected, result)
		}
	})

	t.Run("Match with rest bitstring", func(t *testing.T) {
		var result *bitstringpkg.BitString
		m := NewMatcher()
		m.RestBitstring(&result)

		// Create a bitstring with some data
		bs := bitstringpkg.NewBitStringFromBytes([]byte{0xAB, 0xCD, 0xEF})

		matcherResults, err := m.Match(bs)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if len(matcherResults) != 1 {
			t.Errorf("Expected 1 result, got %d", len(matcherResults))
		}

		if !matcherResults[0].Matched {
			t.Error("Expected match to be successful")
		}

		if result == nil || result.Length() != 24 {
			t.Errorf("Expected result with 24 bits, got %d bits", result.Length())
		}
	})
}
