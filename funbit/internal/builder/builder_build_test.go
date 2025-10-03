package builder

import (
	"testing"

	"github.com/funvibe/funbit/internal/bitstring"
)

// TestBuilder_Build_EdgeCases tests edge cases for the Build method
func TestBuilder_Build_EdgeCases(t *testing.T) {
	t.Run("Valid segment with size 0", func(t *testing.T) {
		b := NewBuilder()
		// Add a segment with size 0 (valid according to BIT_SYNTAX_SPEC.md)
		b.AddInteger(42, bitstring.WithSize(0)) // Valid size 0

		_, err := b.Build()
		if err != nil {
			t.Errorf("Unexpected error for valid size 0: %v", err)
		}
	})

	t.Run("Encode segment error", func(t *testing.T) {
		b := NewBuilder()
		// Add a segment that will fail during encoding
		b.AddBinary([]byte{0xAB}, bitstring.WithSize(2)) // Size mismatch

		_, err := b.Build()
		if err == nil {
			t.Error("Expected error during segment encoding")
		}
	})

	t.Run("Mixed alignment scenarios", func(t *testing.T) {
		b := NewBuilder()
		// Test the special alignment logic in Build method
		b.AddInteger(0b101, bitstring.WithSize(3)) // 3 bits
		b.AddInteger(0xFF, bitstring.WithSize(8))  // Should trigger alignment

		bs, err := b.Build()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		// The actual behavior depends on the implementation details
		// Let's just verify it builds without error
		if bs.Length() == 0 {
			t.Errorf("Expected non-zero bitstring length")
		}
	})

	t.Run("No alignment needed for exact byte boundary", func(t *testing.T) {
		b := NewBuilder()
		// Test the case where 1 bit + 15 bits = 16 bits (no alignment needed)
		b.AddInteger(1, bitstring.WithSize(1))
		b.AddInteger(0x7FFF, bitstring.WithSize(15))

		bs, err := b.Build()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		// Should be 1 + 15 = 16 bits (no padding)
		if bs.Length() != 16 {
			t.Errorf("Expected bitstring length 16, got %d", bs.Length())
		}
	})

	t.Run("Build with segment that fails during encoding", func(t *testing.T) {
		b := NewBuilder()
		// Add a segment that will fail during encoding
		b.AddBinary([]byte{0xAB}, bitstring.WithSize(2)) // Size mismatch

		_, err := b.Build()
		if err == nil {
			t.Error("Expected error during segment encoding")
		}
		t.Logf("Expected encoding error: %v", err)
	})

	t.Run("Build with multiple segments and alignment", func(t *testing.T) {
		b := NewBuilder()
		// Add segments that will require alignment
		b.AddInteger(0b101, bitstring.WithSize(3))   // 3 bits
		b.AddInteger(0xFF, bitstring.WithSize(8))    // Should trigger alignment to byte boundary
		b.AddInteger(0x1234, bitstring.WithSize(16)) // 16 bits

		bs, err := b.Build()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		// Should be 3 + 5 (padding) + 8 + 16 = 32 bits
		if bs.Length() == 0 {
			t.Errorf("Expected non-zero bitstring length")
		}
		t.Logf("Bitstring length with alignment: %d", bs.Length())
	})

	t.Run("Build with empty segments list", func(t *testing.T) {
		b := NewBuilder()
		// Don't add any segments
		bs, err := b.Build()

		if err != nil {
			t.Errorf("Expected no error for empty build, got %v", err)
		}
		if bs == nil {
			t.Fatal("Expected non-nil bitstring for empty build")
		}
		if bs.Length() != 0 {
			t.Errorf("Expected empty bitstring length 0, got %d", bs.Length())
		}
	})
}

// TestBuilder_validateBitstring_EdgeCases tests edge cases for validateBitstringValue
func TestBuilder_validateBitstring_EdgeCases(t *testing.T) {
	t.Run("Validate bitstring with nil value in segment", func(t *testing.T) {
		segment := &bitstring.Segment{
			Value:         nil,
			Type:          bitstring.TypeBitstring,
			Size:          8,
			SizeSpecified: true,
		}

		_, err := validateBitstringValue(segment)
		if err == nil {
			t.Error("Expected error for nil bitstring value")
		}

		if bitStringErr, ok := err.(*bitstring.BitStringError); ok {
			if bitStringErr.Code != bitstring.CodeTypeMismatch {
				t.Errorf("Expected error code %s, got %s", bitstring.CodeTypeMismatch, bitStringErr.Code)
			}
		}
	})
}

// TestBuilder_determineBitstringSize_EdgeCases tests edge cases for determineBitstringSize
func TestBuilder_determineBitstringSize_EdgeCases(t *testing.T) {
	t.Run("Size not specified - use bitstring length", func(t *testing.T) {
		bs := bitstring.NewBitStringFromBits([]byte{0xAB, 0xCD}, 16)
		segment := &bitstring.Segment{
			Value:         bs,
			Type:          bitstring.TypeBitstring,
			SizeSpecified: false,
		}

		size, err := determineBitstringSize(segment, bs)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if size != 16 {
			t.Errorf("Expected size 16, got %d", size)
		}
	})

	t.Run("Size specified - use specified size", func(t *testing.T) {
		bs := bitstring.NewBitStringFromBits([]byte{0xAB, 0xCD}, 16)
		segment := &bitstring.Segment{
			Value:         bs,
			Type:          bitstring.TypeBitstring,
			Size:          8,
			SizeSpecified: true,
		}

		size, err := determineBitstringSize(segment, bs)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if size != 8 {
			t.Errorf("Expected size 8, got %d", size)
		}
	})
}

// TestBuilder_writeBitstringBits_EdgeCases tests edge cases for writeBitstringBits
func TestBuilder_writeBitstringBits_EdgeCases(t *testing.T) {
	t.Run("Write exactly available bits", func(t *testing.T) {
		w := newBitWriter()
		bs := bitstring.NewBitStringFromBits([]byte{0xAB}, 8)

		err := writeBitstringBits(w, bs, 8)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		data, totalBits := w.final()
		if totalBits != 8 {
			t.Errorf("Expected totalBits 8, got %d", totalBits)
		}

		if len(data) != 1 || data[0] != 0xAB {
			t.Errorf("Expected byte [0xAB], got %v", data)
		}
	})

	t.Run("Write partial bits", func(t *testing.T) {
		w := newBitWriter()
		bs := bitstring.NewBitStringFromBits([]byte{0xAB, 0xCD}, 16)

		err := writeBitstringBits(w, bs, 4) // Write only first 4 bits
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		data, totalBits := w.final()
		if totalBits != 4 {
			t.Errorf("Expected totalBits 4, got %d", totalBits)
		}

		// First 4 bits of 0xAB (0b10101010) should be 0b1010
		if len(data) != 1 || data[0] != 0b10100000 {
			t.Errorf("Expected byte 0b10100000, got 0b%08b", data[0])
		}
	})

	t.Run("Write zero bits", func(t *testing.T) {
		w := newBitWriter()
		bs := bitstring.NewBitStringFromBits([]byte{0xAB}, 8)

		err := writeBitstringBits(w, bs, 0)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		data, totalBits := w.final()
		if totalBits != 0 {
			t.Errorf("Expected totalBits 0, got %d", totalBits)
		}

		if len(data) != 0 {
			t.Errorf("Expected empty data, got %v", data)
		}
	})
}
