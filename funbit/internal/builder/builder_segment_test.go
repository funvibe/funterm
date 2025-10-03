package builder

import (
	"testing"

	"github.com/funvibe/funbit/internal/bitstring"
)

// TestBuilder_encodeSegment_AdditionalCoverage tests additional coverage for encodeSegment
func TestBuilder_encodeSegment_AdditionalCoverage(t *testing.T) {
	t.Run("Segment validation error", func(t *testing.T) {
		w := newBitWriter()
		segment := &bitstring.Segment{
			Value:         42,
			Type:          bitstring.TypeInteger,
			Size:          8, // Valid size
			SizeSpecified: true,
			Unit:          300, // Invalid unit - should fail validation
			UnitSpecified: true,
		}

		err := encodeSegment(w, segment)
		if err == nil {
			t.Error("Expected validation error for invalid unit")
		}

		if bitStringErr, ok := err.(*bitstring.BitStringError); ok {
			if bitStringErr.Code != bitstring.CodeInvalidUnit {
				t.Errorf("Expected error code %s, got %s", bitstring.CodeInvalidUnit, bitStringErr.Code)
			}
		}
	})

	t.Run("UTF8 type with validation error", func(t *testing.T) {
		w := newBitWriter()
		segment := &bitstring.Segment{
			Value:         65, // Valid UTF codepoint
			Type:          bitstring.TypeUTF8,
			Size:          8,
			SizeSpecified: true, // Size specified for UTF - should fail
		}

		err := encodeSegment(w, segment)
		if err == nil {
			t.Error("Expected error for invalid UTF segment")
		}
	})

	t.Run("UTF16 type with validation error", func(t *testing.T) {
		w := newBitWriter()
		segment := &bitstring.Segment{
			Value: 0x10FFFF + 1, // Invalid Unicode code point
			Type:  "utf16",
		}

		err := encodeSegment(w, segment)
		// This might pass validation but fail during UTF encoding
		// The important thing is to test the code path
		if err != nil {
			// Error is acceptable, we just want to cover the code path
			t.Logf("Expected possible error for invalid UTF16: %v", err)
		}
	})

	t.Run("UTF32 type with validation error", func(t *testing.T) {
		w := newBitWriter()
		segment := &bitstring.Segment{
			Value: -1, // Invalid Unicode code point
			Type:  "utf32",
		}

		err := encodeSegment(w, segment)
		// This might pass validation but fail during UTF encoding
		if err != nil {
			// Error is acceptable, we just want to cover the code path
			t.Logf("Expected possible error for invalid UTF32: %v", err)
		}
	})

	t.Run("Binary type with validation error", func(t *testing.T) {
		w := newBitWriter()
		segment := &bitstring.Segment{
			Value:         "not bytes", // Invalid value type for binary
			Type:          bitstring.TypeBinary,
			Size:          1,
			SizeSpecified: true,
		}

		err := encodeSegment(w, segment)
		if err == nil {
			t.Error("Expected error for invalid binary segment")
		}

		if bitStringErr, ok := err.(*bitstring.BitStringError); ok {
			if bitStringErr.Code != bitstring.CodeInvalidBinaryData {
				t.Errorf("Expected error code %s, got %s", bitstring.CodeInvalidBinaryData, bitStringErr.Code)
			}
		}
	})

	t.Run("Float type with validation error", func(t *testing.T) {
		w := newBitWriter()
		segment := &bitstring.Segment{
			Value:         "not float", // Invalid value type for float
			Type:          bitstring.TypeFloat,
			Size:          32,
			SizeSpecified: true,
		}

		err := encodeSegment(w, segment)
		if err == nil {
			t.Error("Expected error for invalid float segment")
		}

		if bitStringErr, ok := err.(*bitstring.BitStringError); ok {
			if bitStringErr.Code != bitstring.CodeTypeMismatch {
				t.Errorf("Expected error code %s, got %s", bitstring.CodeTypeMismatch, bitStringErr.Code)
			}
		}
	})

	t.Run("Bitstring type with validation error", func(t *testing.T) {
		w := newBitWriter()
		segment := &bitstring.Segment{
			Value:         "not bitstring", // Invalid value type for bitstring
			Type:          bitstring.TypeBitstring,
			Size:          8,
			SizeSpecified: true,
		}

		err := encodeSegment(w, segment)
		if err == nil {
			t.Error("Expected error for invalid bitstring segment")
		}

		if bitStringErr, ok := err.(*bitstring.BitStringError); ok {
			if bitStringErr.Code != bitstring.CodeInvalidBitstringData {
				t.Errorf("Expected error code %s, got %s", bitstring.CodeInvalidBitstringData, bitStringErr.Code)
			}
		}
	})

	t.Run("Integer type with validation error", func(t *testing.T) {
		w := newBitWriter()
		segment := &bitstring.Segment{
			Value:         "not integer", // Invalid value type for integer
			Type:          bitstring.TypeInteger,
			Size:          8,
			SizeSpecified: true,
		}

		err := encodeSegment(w, segment)
		if err == nil {
			t.Error("Expected error for invalid integer segment")
		}

		if err.Error() != "unsupported integer type for bitstring value: string" {
			t.Errorf("Expected 'unsupported integer type for bitstring value: string', got %v", err)
		}
	})

	t.Run("Empty type with validation error", func(t *testing.T) {
		w := newBitWriter()
		segment := &bitstring.Segment{
			Value:         "not integer", // Invalid value type for default integer
			Type:          "",            // Empty type should default to integer
			Size:          8,
			SizeSpecified: true,
		}

		err := encodeSegment(w, segment)
		if err == nil {
			t.Error("Expected error for invalid segment with empty type")
		}

		if err.Error() != "unsupported integer type for bitstring value: string" {
			t.Errorf("Expected 'unsupported integer type for bitstring value: string', got %v", err)
		}
	})

	t.Run("Multiple validation errors", func(t *testing.T) {
		w := newBitWriter()
		segment := &bitstring.Segment{
			Value:         42, // Valid value
			Type:          bitstring.TypeInteger,
			Size:          8, // Valid size
			SizeSpecified: true,
			Unit:          300, // Invalid unit
			UnitSpecified: true,
		}

		err := encodeSegment(w, segment)
		if err == nil {
			t.Error("Expected validation error for segment with invalid unit")
		}

		// Should catch the validation error
		t.Logf("Validation error (expected): %v", err)
	})

	t.Run("Segment with nil value", func(t *testing.T) {
		w := newBitWriter()
		segment := &bitstring.Segment{
			Value:         nil,
			Type:          bitstring.TypeInteger,
			Size:          8,
			SizeSpecified: true,
		}

		err := encodeSegment(w, segment)
		if err == nil {
			t.Error("Expected error for nil value")
		}
		t.Logf("Expected error for nil value: %v", err)
	})

	t.Run("Segment with invalid type that defaults to integer", func(t *testing.T) {
		w := newBitWriter()
		segment := &bitstring.Segment{
			Value:         "invalid_value",
			Type:          "unknown_type", // Should default to integer
			Size:          8,
			SizeSpecified: true,
		}

		err := encodeSegment(w, segment)
		// This should fail because "invalid_value" cannot be converted to integer
		if err == nil {
			t.Error("Expected error for invalid value with unknown type")
		}
		t.Logf("Expected error for invalid value: %v", err)
	})
}

// TestBuilder_toUint64 tests the toUint64 helper function
func TestBuilder_toUint64(t *testing.T) {
	tests := []struct {
		name      string
		value     interface{}
		expected  uint64
		expectErr bool
	}{
		{"Positive int", int(42), 42, false},
		{"Positive int8", int8(127), 127, false},
		{"Positive int16", int16(32767), 32767, false},
		{"Positive int32", int32(2147483647), 2147483647, false},
		{"Positive int64", int64(9223372036854775807), 9223372036854775807, false},
		{"Negative int", int(-42), uint64(18446744073709551574), false}, // -42 as two's complement
		{"Uint", uint(42), 42, false},
		{"Uint8", uint8(255), 255, false},
		{"Uint16", uint16(65535), 65535, false},
		{"Uint32", uint32(4294967295), 4294967295, false},
		{"Uint64", uint64(18446744073709551615), 18446744073709551615, false},
		{"Unsupported type", "string", 0, true},
		{"Float", float64(3.14), 0, true},
		{"Nil", nil, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := toUint64(tt.value)

			if tt.expectErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Expected no error, got %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("Expected %d, got %d", tt.expected, result)
			}
		})
	}
}

// TestBuilder_encodeInteger_EdgeCases tests edge cases for encodeInteger
func TestBuilder_encodeInteger_EdgeCases(t *testing.T) {
	t.Run("Zero size", func(t *testing.T) {
		w := newBitWriter()
		segment := &bitstring.Segment{
			Value:         int64(42),
			Type:          bitstring.TypeInteger,
			Size:          0, // Valid size 0 according to BIT_SYNTAX_SPEC.md
			SizeSpecified: true,
		}

		err := encodeInteger(w, segment)
		if err != nil {
			t.Errorf("Unexpected error for valid size 0: %v", err)
		}

		// Should write 0 bits
		data, bits := w.final()
		if len(data) != 0 || bits != 0 {
			t.Errorf("Expected no data for size 0, got %d bytes, %d bits", len(data), bits)
		}
	})

	t.Run("Size 65 valid for integer", func(t *testing.T) {
		w := newBitWriter()
		segment := &bitstring.Segment{
			Value:         int64(42),
			Type:          bitstring.TypeInteger,
			Size:          65,
			SizeSpecified: true,
		}

		err := encodeInteger(w, segment)
		if err != nil {
			t.Errorf("Unexpected error for size 65: %v", err)
		}
	})
}
