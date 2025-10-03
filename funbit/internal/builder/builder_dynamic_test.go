package builder

import (
	"fmt"
	"testing"

	"github.com/funvibe/funbit/internal/bitstring"
)

// TestDynamic_AppendToBitString_EdgeCases tests edge cases for dynamic bitstring operations
func TestDynamic_AppendToBitString_EdgeCases(t *testing.T) {
	t.Run("Append with build error", func(t *testing.T) {
		target := bitstring.NewBitStringFromBits([]byte{0xAB}, 8)
		// Create a segment that will fail to build
		segment := bitstring.NewSegment("invalid", bitstring.WithType(bitstring.TypeBinary))
		segment.Size = 1

		_, err := AppendToBitString(target, *segment)
		if err == nil {
			t.Error("Expected error for invalid segment")
		}
	})
}

// TestBuilder_extractBitAtPosition tests the extractBitAtPosition method
func TestBuilder_extractBitAtPosition(t *testing.T) {
	tests := []struct {
		name     string
		byteVal  byte
		bitIndex uint
		expected byte
	}{
		{"Extract MSB", 0b10000000, 0, 1},
		{"Extract LSB", 0b00000001, 7, 1},
		{"Extract middle bit", 0b00100000, 2, 1},
		{"Extract zero bit", 0b01010101, 1, 1},
		{"All ones", 0b11111111, 3, 1},
		{"All zeros", 0b00000000, 4, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractBitAtPosition(tt.byteVal, tt.bitIndex)
			if result != tt.expected {
				t.Errorf("Expected %d, got %d", tt.expected, result)
			}
		})
	}
}

// TestBuilder_setDefaultBitstringProperties tests setting default bitstring properties
func TestBuilder_setDefaultBitstringProperties(t *testing.T) {
	t.Run("Default unit", func(t *testing.T) {
		b := NewBuilder()
		bs := bitstring.NewBitStringFromBits([]byte{0xAB}, 8)
		segment := &bitstring.Segment{}

		b.setDefaultBitstringProperties(segment, bs, []bitstring.SegmentOption{})

		if segment.Unit != 1 {
			t.Errorf("Expected default unit 1, got %d", segment.Unit)
		}
	})

	t.Run("Auto-set size", func(t *testing.T) {
		b := NewBuilder()
		bs := bitstring.NewBitStringFromBits([]byte{0xAB, 0xCD}, 16)
		segment := &bitstring.Segment{}

		b.setDefaultBitstringProperties(segment, bs, []bitstring.SegmentOption{})

		if segment.Size != 16 {
			t.Errorf("Expected auto-set size 16, got %d", segment.Size)
		}

		if !segment.SizeSpecified {
			t.Error("Expected SizeSpecified to be true")
		}
	})

	t.Run("Preserve explicit unit", func(t *testing.T) {
		b := NewBuilder()
		bs := bitstring.NewBitStringFromBits([]byte{0xAB}, 8)
		segment := &bitstring.Segment{Unit: 4}

		b.setDefaultBitstringProperties(segment, bs, []bitstring.SegmentOption{})

		if segment.Unit != 4 {
			t.Errorf("Expected preserved unit 4, got %d", segment.Unit)
		}
	})
}

// TestBuilder_isSizeExplicitlySet tests checking if size is explicitly set
func TestBuilder_isSizeExplicitlySet(t *testing.T) {
	b := &Builder{}

	t.Run("Size explicitly set", func(t *testing.T) {
		options := []bitstring.SegmentOption{bitstring.WithSize(16)}
		result := b.isSizeExplicitlySet(options)

		if !result {
			t.Error("Expected isSizeExplicitlySet to return true for explicit size")
		}
	})

	t.Run("Size not set", func(t *testing.T) {
		options := []bitstring.SegmentOption{bitstring.WithType(bitstring.TypeBinary)}
		result := b.isSizeExplicitlySet(options)

		if result {
			t.Error("Expected isSizeExplicitlySet to return false for no size option")
		}
	})

	t.Run("Empty options", func(t *testing.T) {
		options := []bitstring.SegmentOption{}
		result := b.isSizeExplicitlySet(options)

		if result {
			t.Error("Expected isSizeExplicitlySet to return false for empty options")
		}
	})

	t.Run("Multiple options with size", func(t *testing.T) {
		options := []bitstring.SegmentOption{
			bitstring.WithType(bitstring.TypeBinary),
			bitstring.WithSize(32),
			bitstring.WithEndianness(bitstring.EndiannessLittle),
		}
		result := b.isSizeExplicitlySet(options)

		if !result {
			t.Error("Expected isSizeExplicitlySet to return true for multiple options with size")
		}
	})
}

// TestDynamic_BuildBitStringDynamically tests dynamic bitstring building
func TestDynamic_BuildBitStringDynamically(t *testing.T) {
	t.Run("Valid generator", func(t *testing.T) {
		generator := func() ([]bitstring.Segment, error) {
			return []bitstring.Segment{
				*bitstring.NewSegment(42, bitstring.WithSize(8)),
				*bitstring.NewSegment(17, bitstring.WithSize(8)),
			}, nil
		}

		bs, err := BuildBitStringDynamically(generator)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if bs == nil {
			t.Fatal("Expected non-nil bitstring")
		}

		if bs.Length() != 16 {
			t.Errorf("Expected bitstring length 16, got %d", bs.Length())
		}
	})

	t.Run("Nil generator", func(t *testing.T) {
		bs, err := BuildBitStringDynamically(nil)
		if err == nil {
			t.Error("Expected error for nil generator")
		}

		if bs != nil {
			t.Error("Expected nil bitstring on error")
		}

		if err.Error() != "generator function cannot be nil" {
			t.Errorf("Expected 'generator function cannot be nil', got %v", err)
		}
	})

	t.Run("Generator returns error", func(t *testing.T) {
		generator := func() ([]bitstring.Segment, error) {
			return nil, fmt.Errorf("generator error")
		}

		bs, err := BuildBitStringDynamically(generator)
		if err == nil {
			t.Error("Expected error from generator")
		}

		if bs != nil {
			t.Error("Expected nil bitstring on error")
		}

		if err.Error() != "generator error" {
			t.Errorf("Expected 'generator error', got %v", err)
		}
	})

	t.Run("Empty segments", func(t *testing.T) {
		generator := func() ([]bitstring.Segment, error) {
			return []bitstring.Segment{}, nil
		}

		bs, err := BuildBitStringDynamically(generator)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if bs == nil {
			t.Fatal("Expected non-nil bitstring")
		}

		if bs.Length() != 0 {
			t.Errorf("Expected empty bitstring, got length %d", bs.Length())
		}
	})
}

// TestDynamic_BuildConditionalBitString tests conditional bitstring building
func TestDynamic_BuildConditionalBitString(t *testing.T) {
	trueSegments := []bitstring.Segment{
		*bitstring.NewSegment(1, bitstring.WithSize(8)),
	}

	falseSegments := []bitstring.Segment{
		*bitstring.NewSegment(0, bitstring.WithSize(8)),
	}

	t.Run("True condition", func(t *testing.T) {
		bs, err := BuildConditionalBitString(true, trueSegments, falseSegments)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if bs == nil {
			t.Fatal("Expected non-nil bitstring")
		}

		if bs.Length() != 8 {
			t.Errorf("Expected bitstring length 8, got %d", bs.Length())
		}
	})

	t.Run("False condition", func(t *testing.T) {
		bs, err := BuildConditionalBitString(false, trueSegments, falseSegments)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if bs == nil {
			t.Fatal("Expected non-nil bitstring")
		}

		if bs.Length() != 8 {
			t.Errorf("Expected bitstring length 8, got %d", bs.Length())
		}
	})

	t.Run("Empty segments", func(t *testing.T) {
		bs, err := BuildConditionalBitString(true, []bitstring.Segment{}, []bitstring.Segment{})
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if bs == nil {
			t.Fatal("Expected non-nil bitstring")
		}

		if bs.Length() != 0 {
			t.Errorf("Expected empty bitstring, got length %d", bs.Length())
		}
	})
}

// TestDynamic_AppendToBitString tests appending to bitstring
func TestDynamic_AppendToBitString(t *testing.T) {
	t.Run("Append to existing bitstring", func(t *testing.T) {
		target := bitstring.NewBitStringFromBits([]byte{0xAB}, 8)
		segments := []bitstring.Segment{
			*bitstring.NewSegment(0xCD, bitstring.WithSize(8)),
		}

		result, err := AppendToBitString(target, segments...)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if result == nil {
			t.Fatal("Expected non-nil bitstring")
		}

		if result.Length() != 16 {
			t.Errorf("Expected bitstring length 16, got %d", result.Length())
		}

		bytes := result.ToBytes()
		if len(bytes) != 2 || bytes[0] != 0xAB || bytes[1] != 0xCD {
			t.Errorf("Expected bytes [0xAB, 0xCD], got %v", bytes)
		}
	})

	t.Run("Nil target", func(t *testing.T) {
		segments := []bitstring.Segment{
			*bitstring.NewSegment(0xAB, bitstring.WithSize(8)),
		}

		result, err := AppendToBitString(nil, segments...)
		if err == nil {
			t.Error("Expected error for nil target")
		}

		if result != nil {
			t.Error("Expected nil bitstring on error")
		}

		if err.Error() != "target bitstring cannot be nil" {
			t.Errorf("Expected 'target bitstring cannot be nil', got %v", err)
		}
	})

	t.Run("Empty segments", func(t *testing.T) {
		target := bitstring.NewBitStringFromBits([]byte{0xAB}, 8)

		result, err := AppendToBitString(target)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if result == nil {
			t.Fatal("Expected non-nil bitstring")
		}

		if result.Length() != 8 {
			t.Errorf("Expected bitstring length 8, got %d", result.Length())
		}

		// Should be a clone, not the same instance
		if result == target {
			t.Error("Expected cloned bitstring, not same instance")
		}
	})
}
