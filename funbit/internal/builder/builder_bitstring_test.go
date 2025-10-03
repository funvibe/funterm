package builder

import (
	"testing"

	"github.com/funvibe/funbit/internal/bitstring"
)

// TestBuilder_encodeInteger_BitstringTypeEdgeCases tests the bitstring type edge cases in encodeInteger
func TestBuilder_encodeInteger_BitstringTypeEdgeCases(t *testing.T) {
	// Test case 1: Bitstring type with integer value and size > 8 (should trigger insufficient bits error)
	t.Run("BitstringTypeIntegerValueSizeTooLarge", func(t *testing.T) {
		builder := NewBuilder()

		// Create a segment with bitstring type, integer value, and size > 8
		// This should trigger the "size too large for data" error on line 425
		segment := &bitstring.Segment{
			Value:         0, // Integer value
			Type:          bitstring.TypeBitstring,
			Size:          16, // Size > 8, should trigger error
			SizeSpecified: true,
		}

		builder.segments = append(builder.segments, segment)

		_, err := builder.Build()
		if err == nil {
			t.Error("Build should have failed with insufficient bits error")
		} else {
			t.Logf("Expected error occurred: %v", err)
		}
	})

	// Test case 2: Bitstring type with []byte value and insufficient data
	t.Run("BitstringTypeByteValueInsufficientData", func(t *testing.T) {
		builder := NewBuilder()

		// Create a segment with bitstring type, []byte value, and size > available bits
		// This should trigger the "size too large for data" error on line 417
		segment := &bitstring.Segment{
			Value:         []byte{0xFF}, // 1 byte = 8 bits
			Type:          bitstring.TypeBitstring,
			Size:          16, // Request 16 bits, only 8 available
			SizeSpecified: true,
		}

		builder.segments = append(builder.segments, segment)

		_, err := builder.Build()
		if err == nil {
			t.Error("Build should have failed with insufficient bits error")
		} else {
			t.Logf("Expected error occurred: %v", err)
		}
	})

	// Test case 3: Bitstring type with *BitString value and sufficient data
	t.Run("BitstringTypeBitStringValueSufficientData", func(t *testing.T) {
		builder := NewBuilder()

		// Create a proper *BitString value with sufficient data
		bs := bitstring.NewBitStringFromBits([]byte{0xFF, 0xFF}, 16)
		segment := &bitstring.Segment{
			Value:         bs, // *BitString value
			Type:          bitstring.TypeBitstring,
			Size:          16, // Request 16 bits, 16 available
			SizeSpecified: true,
		}

		builder.segments = append(builder.segments, segment)

		_, err := builder.Build()
		if err != nil {
			t.Errorf("Build should have succeeded: %v", err)
		} else {
			t.Log("Build succeeded with sufficient *BitString data")
		}
	})

	// Test case 4: Bitstring type with *BitString value and insufficient data
	t.Run("BitstringTypeBitStringValueInsufficientData", func(t *testing.T) {
		builder := NewBuilder()

		// Create a proper *BitString value with insufficient data
		bs := bitstring.NewBitStringFromBits([]byte{0xFF}, 8) // Only 8 bits available
		segment := &bitstring.Segment{
			Value:         bs, // *BitString value
			Type:          bitstring.TypeBitstring,
			Size:          16, // Request 16 bits, only 8 available
			SizeSpecified: true,
		}

		builder.segments = append(builder.segments, segment)

		_, err := builder.Build()
		if err == nil {
			t.Error("Build should have failed with insufficient bits error")
		} else {
			t.Logf("Expected error occurred: %v", err)
		}
	})
}
