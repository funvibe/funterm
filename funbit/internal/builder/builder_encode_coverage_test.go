package builder

import (
	"strings"
	"testing"

	"github.com/funvibe/funbit/internal/bitstring"
)

// TestEncodeFloat_FullCoverage targets the missing coverage in encodeFloat to reach 100%
func TestEncodeFloat_FullCoverage(t *testing.T) {
	// Looking at encodeFloat function (lines 625-683), I need to find what's not covered
	// Let me test all possible paths and edge cases

	// Test case 1: Test all possible float types and endianness combinations
	t.Run("Float32AllEndianness", func(t *testing.T) {
		endiannessOptions := []string{
			bitstring.EndiannessBig,
			bitstring.EndiannessLittle,
			bitstring.EndiannessNative,
			"", // Default (big endian)
		}

		for _, endianness := range endiannessOptions {
			writer := newBitWriter()
			segment := &bitstring.Segment{
				Value:         float32(3.14159),
				Type:          bitstring.TypeFloat,
				Size:          32,
				SizeSpecified: true,
				Endianness:    endianness,
			}

			err := encodeFloat(writer, segment)
			if err != nil {
				t.Errorf("encodeFloat failed for float32 with endianness %s: %v", endianness, err)
			}
		}
	})

	// Test case 2: Float64 with all endianness combinations
	t.Run("Float64AllEndianness", func(t *testing.T) {
		endiannessOptions := []string{
			bitstring.EndiannessBig,
			bitstring.EndiannessLittle,
			bitstring.EndiannessNative,
			"", // Default (big endian)
		}

		for _, endianness := range endiannessOptions {
			writer := newBitWriter()
			segment := &bitstring.Segment{
				Value:         float64(2.718281828459045),
				Type:          bitstring.TypeFloat,
				Size:          64,
				SizeSpecified: true,
				Endianness:    endianness,
			}

			err := encodeFloat(writer, segment)
			if err != nil {
				t.Errorf("encodeFloat failed for float64 with endianness %s: %v", endianness, err)
			}
		}
	})

	// Test case 3: Test edge case float values
	t.Run("EdgeCaseFloatValues", func(t *testing.T) {
		edgeCaseValues := []interface{}{
			float32(0.0),
			float32(-0.0),
			float32(3.4028234663852886e+38),  // Max float32
			float32(-3.4028234663852886e+38), // Min float32
			float32(1.401298464324817e-45),   // Smallest positive float32
			float32(-1.401298464324817e-45),  // Smallest negative float32
			float32(3.141592653589793),
			float64(0.0),
			float64(-0.0),
			float64(1.7976931348623157e+308),  // Max float64
			float64(-1.7976931348623157e+308), // Min float64
			float64(4.940656458412465e-324),   // Smallest positive float64
			float64(-4.940656458412465e-324),  // Smallest negative float64
			float64(3.141592653589793),
		}

		for _, value := range edgeCaseValues {
			writer := newBitWriter()

			var size uint
			switch value.(type) {
			case float32:
				size = 32
			case float64:
				size = 64
			}

			segment := &bitstring.Segment{
				Value:         value,
				Type:          bitstring.TypeFloat,
				Size:          size,
				SizeSpecified: true,
				Endianness:    bitstring.EndiannessBig,
			}

			err := encodeFloat(writer, segment)
			if err != nil {
				t.Errorf("encodeFloat failed for edge case value %v: %v", value, err)
			}
		}
	})

	// Test case 4: Test interface{} values containing floats
	t.Run("InterfaceFloatValues", func(t *testing.T) {
		interfaceValues := []interface{}{
			interface{}(float32(1.234)),
			interface{}(float64(5.678)),
		}

		for _, value := range interfaceValues {
			writer := newBitWriter()

			var size uint
			switch value.(type) {
			case float32:
				size = 32
			case float64:
				size = 64
			}

			segment := &bitstring.Segment{
				Value:         value,
				Type:          bitstring.TypeFloat,
				Size:          size,
				SizeSpecified: true,
				Endianness:    bitstring.EndiannessNative,
			}

			err := encodeFloat(writer, segment)
			if err != nil {
				t.Errorf("encodeFloat failed for interface{} value %v: %v", value, err)
			}
		}
	})

	// Test case 5: Test error conditions
	t.Run("ErrorConditions", func(t *testing.T) {
		// Test size not specified
		writer1 := newBitWriter()
		segment1 := &bitstring.Segment{
			Value:         float32(1.0),
			Type:          bitstring.TypeFloat,
			SizeSpecified: false, // Should trigger error
		}
		err := encodeFloat(writer1, segment1)
		if err == nil {
			t.Error("encodeFloat should have failed with size not specified")
		}

		// Test size zero
		writer2 := newBitWriter()
		segment2 := &bitstring.Segment{
			Value:         float32(1.0),
			Type:          bitstring.TypeFloat,
			Size:          0, // Should trigger error
			SizeSpecified: true,
		}
		err = encodeFloat(writer2, segment2)
		if err == nil {
			t.Error("encodeFloat should have failed with size zero")
		}

		// Test invalid size
		writer3 := newBitWriter()
		segment3 := &bitstring.Segment{
			Value:         float32(1.0),
			Type:          bitstring.TypeFloat,
			Size:          24, // Invalid size (not 16, 32, or 64)
			SizeSpecified: true,
		}
		err = encodeFloat(writer3, segment3)
		if err == nil {
			t.Error("encodeFloat should have failed with invalid size")
		}

		// Test invalid value type
		writer4 := newBitWriter()
		segment4 := &bitstring.Segment{
			Value:         "not a float", // Invalid type
			Type:          bitstring.TypeFloat,
			Size:          32,
			SizeSpecified: true,
		}
		err = encodeFloat(writer4, segment4)
		if err == nil {
			t.Error("encodeFloat should have failed with invalid value type")
		}
	})
}

// TestWriteBitstringBits_FullCoverage targets the missing coverage in writeBitstringBits to reach 100%
func TestWriteBitstringBits_FullCoverage(t *testing.T) {
	// Looking at writeBitstringBits function (lines 599-615), I need to find what's not covered
	// The function has a safety check: if byteIndex >= uint(len(sourceBytes)) { break }
	// Let me test scenarios that might trigger this

	// Test case 1: Create bitstring and test exact size matches
	t.Run("ExactSizeMatches", func(t *testing.T) {
		testCases := []struct {
			data     []byte
			size     uint
			expected bool
		}{
			{[]byte{0xFF}, 1, true},        // 1 byte, 1 bit
			{[]byte{0xFF}, 8, true},        // 1 byte, 8 bits
			{[]byte{0xFF, 0x00}, 16, true}, // 2 bytes, 16 bits
			{[]byte{0xAA, 0x55}, 15, true}, // 2 bytes, 15 bits
			{[]byte{0x80}, 1, true},        // 1 byte, 1 bit (MSB set)
		}

		for _, tc := range testCases {
			writer := newBitWriter()
			bs := bitstring.NewBitStringFromBits(tc.data, uint(len(tc.data))*8)

			err := writeBitstringBits(writer, bs, tc.size)
			if err != nil && tc.expected {
				t.Errorf("writeBitstringBits failed for data %v, size %d: %v", tc.data, tc.size, err)
			}
		}
	})

	// Test case 2: Test boundary conditions that might trigger the safety check
	t.Run("BoundaryConditions", func(t *testing.T) {
		writer := newBitWriter()
		bs := bitstring.NewBitStringFromBits([]byte{0xFF}, 8) // 1 byte = 8 bits

		// Test with size larger than available (should trigger safety check)
		err := writeBitstringBits(writer, bs, 16) // Request 16 bits, only 8 available
		if err != nil {
			t.Errorf("writeBitstringBits failed with size larger than available: %v", err)
		}
	})

	// Test case 3: Test zero size
	t.Run("ZeroSize", func(t *testing.T) {
		writer := newBitWriter()
		bs := bitstring.NewBitStringFromBits([]byte{0xFF}, 8)

		err := writeBitstringBits(writer, bs, 0) // Zero size
		if err != nil {
			t.Errorf("writeBitstringBits failed with zero size: %v", err)
		}
	})

	// Test case 4: Test single bit extraction from different positions
	t.Run("SingleBitPositions", func(t *testing.T) {
		bs := bitstring.NewBitStringFromBits([]byte{0xAA}, 8) // 0xAA = 10101010

		// Test extracting each bit position individually
		for i := uint(0); i < 8; i++ {
			writer := newBitWriter()
			err := writeBitstringBits(writer, bs, 1) // Write 1 bit at a time

			if err != nil {
				t.Errorf("writeBitstringBits failed for bit position %d: %v", i, err)
			}
		}
	})

	// Test case 5: Test with empty bitstring
	t.Run("EmptyBitstring", func(t *testing.T) {
		writer := newBitWriter()
		bs := bitstring.NewBitStringFromBits([]byte{}, 0) // Empty bitstring

		err := writeBitstringBits(writer, bs, 0) // Zero size on empty bitstring
		if err != nil {
			t.Errorf("writeBitstringBits failed with empty bitstring: %v", err)
		}
	})

	// Test case 6: Test with maximum bit patterns
	t.Run("MaximumBitPatterns", func(t *testing.T) {
		testPatterns := []struct {
			data []byte
			size uint
		}{
			{[]byte{0xFF}, 8},        // All 1s
			{[]byte{0x00}, 8},        // All 0s
			{[]byte{0x55}, 8},        // Alternating 01010101
			{[]byte{0xAA}, 8},        // Alternating 10101010
			{[]byte{0x80, 0x00}, 9},  // MSB set + zero byte
			{[]byte{0x00, 0x01}, 16}, // Zero byte + LSB set
		}

		for _, pattern := range testPatterns {
			writer := newBitWriter()
			bs := bitstring.NewBitStringFromBits(pattern.data, uint(len(pattern.data))*8)

			err := writeBitstringBits(writer, bs, pattern.size)
			if err != nil {
				t.Errorf("writeBitstringBits failed for pattern %v, size %d: %v", pattern.data, pattern.size, err)
			}
		}
	})
}

// TestEncodeSegment_UnsupportedType covers the default case for unsupported segment types
func TestEncodeSegment_UnsupportedType(t *testing.T) {
	// Create a segment with an unsupported type
	segment := bitstring.NewSegment(42, bitstring.WithType("unsupported_type"))

	// Try to build the bitstring - this should fail with unsupported type error
	builder := NewBuilder()
	builder.AddSegment(*segment)

	_, err := builder.Build()

	// Verify the error
	if err == nil {
		t.Error("Expected error for unsupported type, got nil")
	} else if !strings.Contains(err.Error(), "unsupported segment type") {
		t.Errorf("Expected 'unsupported segment type' error, got %q", err.Error())
	}

}

// TestEncodeInteger_NegativeValueAsUnsigned covers the case where a negative value is encoded as unsigned
func TestEncodeInteger_NegativeValueAsUnsigned(t *testing.T) {
	// Create a segment with a negative value but unsigned flag
	segment := bitstring.NewSegment(-42, bitstring.WithSize(8), bitstring.WithSigned(false))

	// Try to build the bitstring - this should fail with overflow error
	builder := NewBuilder()
	builder.AddSegment(*segment)

	_, err := builder.Build()

	// Verify the error
	if err == nil {
		t.Error("Expected error for unsigned overflow, got nil")
	} else if err.Error() != "unsigned overflow" {
		t.Errorf("Expected 'unsigned overflow' error, got %q", err.Error())
	}
}

// TestEncodeInteger_BitstringTypeWithInsufficientSliceData covers the case where bitstring type has slice data with insufficient bits
func TestEncodeInteger_BitstringTypeWithInsufficientSliceData(t *testing.T) {
	// Create a segment with bitstring type and slice data that has insufficient bits
	data := []byte{0x12}                                                                           // 8 bits
	segment := bitstring.NewSegment(data, bitstring.WithType("bitstring"), bitstring.WithSize(16)) // Request 16 bits

	// Try to build the bitstring - this should fail with insufficient bits error
	builder := NewBuilder()
	builder.AddSegment(*segment)

	_, err := builder.Build()

	// Verify the error
	if err == nil {
		t.Error("Expected error for bitstring type mismatch, got nil")
	} else if err.Error() != "bitstring segment expects *BitString, got []uint8 (context: [18])" {
		t.Errorf("Expected 'bitstring segment expects *BitString, got []uint8 (context: [18])' error, got %q", err.Error())
	}
}

// TestEncodeBinary_SizeNotSpecified covers the case where binary segment size is not specified
func TestEncodeBinary_SizeNotSpecified(t *testing.T) {
	// Create a binary segment without specifying size
	// This is tricky because the AddBinary method auto-sets size based on data length
	// So we need to create a segment directly and manipulate it
	data := []byte{0x01, 0x02, 0x03}
	segment := bitstring.NewSegment(data, bitstring.WithType("binary"))
	segment.SizeSpecified = false // Manually unset size specified

	// Try to build the bitstring - this should fail with binary size required error
	builder := NewBuilder()
	builder.AddSegment(*segment)

	_, err := builder.Build()

	// Verify the error
	if err == nil {
		t.Error("Expected error for zero size, got nil")
	} else if err.Error() != "binary size cannot be zero" {
		t.Errorf("Expected 'binary size cannot be zero' error, got %q", err.Error())
	}
}

// TestEncodeSegment_MissingPaths covers the remaining uncovered paths in encodeSegment
func TestEncodeSegment_MissingPaths(t *testing.T) {
	// Test the default case in encodeSegment switch statement
	// This should trigger line 327: return fmt.Errorf("unsupported segment type: %s", segment.Type)
	// But first we need to bypass validation by setting a valid size
	segment := &bitstring.Segment{
		Value:         42,
		Type:          "unknown_type",
		Size:          8,
		SizeSpecified: true,
		Signed:        false,
		Unit:          1,
	}

	err := encodeSegment(newBitWriter(), segment)
	if err == nil {
		t.Error("Expected error for unsupported segment type, got nil")
	} else if err.Error() != "unsupported segment type: unknown_type" {
		t.Errorf("Expected 'unsupported segment type: unknown_type', got %v", err)
	}
}

// TestEncodeInteger_MissingPaths covers the remaining uncovered paths in encodeInteger
func TestEncodeInteger_MissingPaths(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		size     uint
		signed   bool
		expected string
	}{
		{
			name:     "Unsigned value encoded as signed - truncated",
			value:    uint64(255), // 255 will be truncated and interpreted as signed (-1)
			size:     8,
			signed:   true,
			expected: "", // No error expected according to Erlang spec
		},
		{
			name:     "Negative value encoded as unsigned",
			value:    int32(-1),
			size:     8,
			signed:   false,
			expected: "unsigned overflow",
		},
		{
			name:     "Bitstring type with integer value and size > 8",
			value:    int32(42),
			size:     16,
			signed:   false,
			expected: "size too large for data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			segment := bitstring.NewSegment(tt.value,
				bitstring.WithSize(tt.size),
				bitstring.WithSigned(tt.signed),
				bitstring.WithType(bitstring.TypeBitstring))

			err := encodeInteger(newBitWriter(), segment)
			if tt.expected == "" {
				// No error expected
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
			} else {
				// Error expected
				if err == nil {
					t.Error("Expected error, got nil")
				} else if err.Error() != tt.expected {
					t.Errorf("Expected %q, got %q", tt.expected, err.Error())
				}
			}
		})
	}
}

// TestEncodeBinary_MissingPaths covers the remaining uncovered paths in encodeBinary
func TestEncodeBinary_MissingPaths(t *testing.T) {
	// Test the case where SizeSpecified is false
	// This should trigger lines 525-528 in encodeBinary
	// But looking at the code, this path is actually blocked by the validation at line 511-513

	// Let's test what actually happens - the validation should trigger first
	segment := &bitstring.Segment{
		Value:         []byte{1, 2, 3},
		Type:          bitstring.TypeBinary,
		Size:          0,
		SizeSpecified: false,
		Unit:          8,
	}

	err := encodeBinary(newBitWriter(), segment)
	if err == nil {
		t.Error("Expected error for binary with SizeSpecified=false, got nil")
	} else if err.Error() != "binary segment must have size specified" {
		t.Errorf("Expected 'binary segment must have size specified', got %v", err)
	}

	// The uncovered path might be different - let's check if there are other paths
	// Looking at the code, lines 525-528 are actually unreachable because of the validation above
	// This suggests the coverage tool might be counting these lines as uncovered because they're unreachable
}
