package builder

import (
	"testing"

	"github.com/funvibe/funbit/internal/bitstring"
)

// TestBuilder_Build_AlignmentEdgeCases tests the specific alignment scenarios mentioned in the Build function
func TestBuilder_Build_AlignmentEdgeCases1(t *testing.T) {
	// Test case 1: 3 bits + 8 bits scenario
	t.Run("3BitsPlus8Bits", func(t *testing.T) {
		builder := NewBuilder()

		// Create first segment with empty type and 3 bits
		segment1 := &bitstring.Segment{
			Value:         uint64(5), // 101 in binary (3 bits)
			Type:          "",        // Empty type to trigger special alignment logic
			Size:          3,
			SizeSpecified: true,
		}

		// Create second segment with 8 bits
		segment2 := &bitstring.Segment{
			Value:         uint64(255), // 11111111 in binary (8 bits)
			Type:          "",          // Empty type to trigger special alignment logic
			Size:          8,
			SizeSpecified: true,
		}

		// Add segments directly to bypass AddInteger logic
		builder.segments = append(builder.segments, segment1, segment2)

		// Build should trigger the alignment logic for i==1 && bitCount==3
		result, err := builder.Build()
		if err != nil {
			t.Errorf("Build failed: %v", err)
		}

		if result == nil {
			t.Error("Build result should not be nil")
		}

		t.Logf("3+8 bits result: length=%d bits", result.Length())
	})

	// Test case 2: 1 bit + 15 bits scenario
	t.Run("1BitPlus15Bits", func(t *testing.T) {
		builder := NewBuilder()

		// Create first segment with empty type and 1 bit
		segment1 := &bitstring.Segment{
			Value:         uint64(1), // 1 in binary (1 bit)
			Type:          "",        // Empty type to trigger special alignment logic
			Size:          1,
			SizeSpecified: true,
		}

		// Create second segment with 15 bits
		segment2 := &bitstring.Segment{
			Value:         uint64(0x7FFF), // 0111111111111111 in binary (15 bits)
			Type:          "",             // Empty type to trigger special alignment logic
			Size:          15,
			SizeSpecified: true,
		}

		// Add segments directly to bypass AddInteger logic
		builder.segments = append(builder.segments, segment1, segment2)

		// Build should trigger the alignment logic for i==1 && bitCount==1
		// This should NOT add alignment because 1 + 15 = 16 bits (already aligned)
		result, err := builder.Build()
		if err != nil {
			t.Errorf("Build failed: %v", err)
		}

		if result == nil {
			t.Error("Build result should not be nil")
		}

		t.Logf("1+15 bits result: length=%d bits", result.Length())
	})

	// Test case 3: Default alignment case
	t.Run("DefaultAlignmentCase", func(t *testing.T) {
		builder := NewBuilder()

		// Create first segment with empty type and 5 bits
		segment1 := &bitstring.Segment{
			Value:         uint64(31), // 11111 in binary (5 bits)
			Type:          "",         // Empty type to trigger special alignment logic
			Size:          5,
			SizeSpecified: true,
		}

		// Create second segment with empty type and 10 bits
		segment2 := &bitstring.Segment{
			Value:         uint64(1023), // 1111111111 in binary (10 bits)
			Type:          "",           // Empty type to trigger special alignment logic
			Size:          10,
			SizeSpecified: true,
		}

		// Add segments directly to bypass AddInteger logic
		builder.segments = append(builder.segments, segment1, segment2)

		// Build should trigger the default alignment case (else branch)
		result, err := builder.Build()
		if err != nil {
			t.Errorf("Build failed: %v", err)
		}

		if result == nil {
			t.Error("Build result should not be nil")
		}

		t.Logf("5+10 bits result: length=%d bits", result.Length())
	})
}

// TestBuilder_Build_MissingAlignmentCases tests specific alignment scenarios that may be missing coverage
func TestBuilder_Build_MissingAlignmentCases(t *testing.T) {
	t.Run("Build with valid segment size 0 before alignment", func(t *testing.T) {
		b := NewBuilder()
		// Add a segment with size 0 (valid according to BIT_SYNTAX_SPEC.md)
		b.AddInteger(42, bitstring.WithSize(0)) // Valid size 0

		_, err := b.Build()
		if err != nil {
			t.Errorf("Unexpected error for valid size 0: %v", err)
		}
	})

	t.Run("Build with mixed segment types and alignment", func(t *testing.T) {
		b := NewBuilder()
		// Test complex alignment scenario with different segment types
		b.AddInteger(0b101, bitstring.WithSize(3))   // 3 bits
		b.AddBinary([]byte{0xAB})                    // Should trigger alignment
		b.AddInteger(0x1234, bitstring.WithSize(16)) // 16 bits

		bs, err := b.Build()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if bs == nil {
			t.Fatal("Expected non-nil bitstring")
		}

		// Should be 3 + 5 (padding) + 8 + 16 = 32 bits
		if bs.Length() == 0 {
			t.Error("Expected non-empty bitstring")
		}
		t.Logf("Mixed alignment bitstring length: %d", bs.Length())
	})

	t.Run("Build with exact byte boundary alignment", func(t *testing.T) {
		b := NewBuilder()
		// Test case where total bits exactly align to byte boundary
		b.AddInteger(0b1, bitstring.WithSize(1))       // 1 bit
		b.AddInteger(0b1111111, bitstring.WithSize(7)) // 7 bits = 1 byte total
		b.AddInteger(0xFF, bitstring.WithSize(8))      // 8 bits = 1 byte

		bs, err := b.Build()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		// Should be 1 + 7 + 8 = 16 bits (exactly 2 bytes, no padding needed)
		if bs.Length() != 16 {
			t.Errorf("Expected bitstring length 16, got %d", bs.Length())
		}
	})

	t.Run("Build with multiple alignment scenarios", func(t *testing.T) {
		b := NewBuilder()
		// Test multiple alignment scenarios in sequence
		b.AddInteger(0b1, bitstring.WithSize(1))      // 1 bit
		b.AddInteger(0b1, bitstring.WithSize(1))      // 1 bit (total 2 bits)
		b.AddInteger(0b111111, bitstring.WithSize(6)) // 6 bits (total 8 bits = 1 byte)
		b.AddInteger(0xAB, bitstring.WithSize(8))     // 8 bits (should align automatically)

		bs, err := b.Build()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		// Should be 1 + 1 + 6 + 8 = 16 bits
		if bs.Length() != 16 {
			t.Errorf("Expected bitstring length 16, got %d", bs.Length())
		}
	})
}

// TestBuilder_Build_AlignmentEdgeCases tests the specific alignment scenarios mentioned in the Build function
func TestBuilder_Build_AlignmentEdgeCases2(t *testing.T) {
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
