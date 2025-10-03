package builder

import (
	"testing"

	"github.com/funvibe/funbit/internal/bitstring"
)

// TestBuilder_AddFloat_SizeSpecifiedFalse tests scenarios where SizeSpecified might be false
func TestBuilder_AddFloat_SizeSpecifiedFalse(t *testing.T) {
	t.Run("Float with custom type that results in SizeSpecified=false", func(t *testing.T) {
		b := NewBuilder()

		// Create a custom type that might result in SizeSpecified=false
		type CustomFloat struct {
			val float64
		}

		customValue := CustomFloat{}
		result := b.AddFloat(customValue)

		if result != b {
			t.Error("Expected AddFloat() to return the same builder instance")
		}

		// Verify segment was added
		if len(b.segments) != 1 {
			t.Error("Expected 1 segment to be added")
		}
		if b.segments[0].Type != bitstring.TypeFloat {
			t.Error("Expected segment type to be float")
		}

		// Log the actual SizeSpecified value to understand behavior
		t.Logf("CustomFloat - Size: %d, SizeSpecified: %v", b.segments[0].Size, b.segments[0].SizeSpecified)

		// If SizeSpecified is false, then the default size should be set
		if !b.segments[0].SizeSpecified {
			if b.segments[0].Size != bitstring.DefaultSizeFloat {
				t.Errorf("Expected default size %d, got %d", bitstring.DefaultSizeFloat, b.segments[0].Size)
			}
		}
	})

	t.Run("Float with map value to test unusual type handling", func(t *testing.T) {
		b := NewBuilder()

		// Test with map value (unusual type that might result in SizeSpecified=false)
		value := map[string]interface{}{"value": 3.14}
		result := b.AddFloat(value)

		if result != b {
			t.Error("Expected AddFloat() to return the same builder instance")
		}

		// Verify segment was added
		if len(b.segments) != 1 {
			t.Error("Expected 1 segment to be added")
		}
		if b.segments[0].Type != bitstring.TypeFloat {
			t.Error("Expected segment type to be float")
		}

		// Log the actual behavior
		t.Logf("Map value - Size: %d, SizeSpecified: %v", b.segments[0].Size, b.segments[0].SizeSpecified)
	})

	t.Run("Float with slice value to test array type handling", func(t *testing.T) {
		b := NewBuilder()

		// Test with slice value
		value := []float64{3.14, 2.718}
		result := b.AddFloat(value)

		if result != b {
			t.Error("Expected AddFloat() to return the same builder instance")
		}

		// Verify segment was added
		if len(b.segments) != 1 {
			t.Error("Expected 1 segment to be added")
		}
		if b.segments[0].Type != bitstring.TypeFloat {
			t.Error("Expected segment type to be float")
		}

		// Log the actual behavior
		t.Logf("Slice value - Size: %d, SizeSpecified: %v", b.segments[0].Size, b.segments[0].SizeSpecified)
	})

	t.Run("Float with function value to test function type handling", func(t *testing.T) {
		b := NewBuilder()

		// Test with function value
		value := func() float64 { return 3.14 }
		result := b.AddFloat(value)

		if result != b {
			t.Error("Expected AddFloat() to return the same builder instance")
		}

		// Verify segment was added
		if len(b.segments) != 1 {
			t.Error("Expected 1 segment to be added")
		}
		if b.segments[0].Type != bitstring.TypeFloat {
			t.Error("Expected segment type to be float")
		}

		// Log the actual behavior
		t.Logf("Function value - Size: %d, SizeSpecified: %v", b.segments[0].Size, b.segments[0].SizeSpecified)
	})

	t.Run("Float with channel value to test channel type handling", func(t *testing.T) {
		b := NewBuilder()

		// Test with channel value
		value := make(chan float64)
		result := b.AddFloat(value)

		if result != b {
			t.Error("Expected AddFloat() to return the same builder instance")
		}

		// Verify segment was added
		if len(b.segments) != 1 {
			t.Error("Expected 1 segment to be added")
		}
		if b.segments[0].Type != bitstring.TypeFloat {
			t.Error("Expected segment type to be float")
		}

		// Log the actual behavior
		t.Logf("Channel value - Size: %d, SizeSpecified: %v", b.segments[0].Size, b.segments[0].SizeSpecified)
	})

	t.Run("Float with pointer to function to test pointer type handling", func(t *testing.T) {
		b := NewBuilder()

		// Test with pointer to function
		funcPtr := func() float64 { return 3.14 }
		value := &funcPtr
		result := b.AddFloat(value)

		if result != b {
			t.Error("Expected AddFloat() to return the same builder instance")
		}

		// Verify segment was added
		if len(b.segments) != 1 {
			t.Error("Expected 1 segment to be added")
		}
		if b.segments[0].Type != bitstring.TypeFloat {
			t.Error("Expected segment type to be float")
		}

		// Log the actual behavior
		t.Logf("Pointer to function - Size: %d, SizeSpecified: %v", b.segments[0].Size, b.segments[0].SizeSpecified)
	})

	t.Run("Float with array value to test array type handling", func(t *testing.T) {
		b := NewBuilder()

		// Test with array value
		value := [2]float64{3.14, 2.718}
		result := b.AddFloat(value)

		if result != b {
			t.Error("Expected AddFloat() to return the same builder instance")
		}

		// Verify segment was added
		if len(b.segments) != 1 {
			t.Error("Expected 1 segment to be added")
		}
		if b.segments[0].Type != bitstring.TypeFloat {
			t.Error("Expected segment type to be float")
		}

		// Log the actual behavior
		t.Logf("Array value - Size: %d, SizeSpecified: %v", b.segments[0].Size, b.segments[0].SizeSpecified)
	})
}

// TestBuilder_AddFloat_SizeAlreadySpecified tests the scenario where size is already specified in options
func TestBuilder_AddFloat_SizeAlreadySpecified(t *testing.T) {
	builder := NewBuilder()

	// Test case where size is already specified in options
	result := builder.AddFloat(3.14, bitstring.WithSize(64))

	// Verify that the segment was added
	if len(result.segments) != 1 {
		t.Errorf("Expected 1 segment, got %d", len(result.segments))
	}

	segment := result.segments[0]
	if segment.Type != bitstring.TypeFloat {
		t.Errorf("Expected segment type to be float, got %s", segment.Type)
	}

	// When size is specified in options, SizeSpecified should be true
	// and Size should be the specified value (not default)
	if segment.Size != 64 {
		t.Errorf("Expected segment size to be 64, got %d", segment.Size)
	}

	if !segment.SizeSpecified {
		t.Errorf("Expected SizeSpecified to be true when size is provided in options")
	}

	// Test with different size
	builder2 := NewBuilder()
	result2 := builder2.AddFloat(2.71, bitstring.WithSize(32))

	if len(result2.segments) != 1 {
		t.Errorf("Expected 1 segment, got %d", len(result2.segments))
	}

	segment2 := result2.segments[0]
	if segment2.Size != 32 {
		t.Errorf("Expected segment size to be 32, got %d", segment2.Size)
	}

	if !segment2.SizeSpecified {
		t.Errorf("Expected SizeSpecified to be true when size is provided in options")
	}
}

// TestBuilder_AddFloat_SizeNotSpecified tests the case where size is not specified
func TestBuilder_AddFloat_SizeNotSpecified(t *testing.T) {
	t.Run("Float32 without size specified", func(t *testing.T) {
		b := NewBuilder()
		value := float32(3.14159)
		result := b.AddFloat(value) // No size specified

		if result != b {
			t.Error("Expected AddFloat() to return the same builder instance")
		}

		if len(b.segments) != 1 {
			t.Fatalf("Expected 1 segment, got %d", len(b.segments))
		}

		segment := b.segments[0]
		if segment.Type != bitstring.TypeFloat {
			t.Errorf("Expected segment type 'float', got '%s'", segment.Type)
		}

		// When size is not specified, it should use default size
		if segment.Size != 32 {
			t.Errorf("Expected segment size 32, got %d", segment.Size)
		}

		// SizeSpecified should be false when using default
		if segment.SizeSpecified {
			t.Error("Expected SizeSpecified to be false when using default size")
		}
	})

	t.Run("Float64 without size specified", func(t *testing.T) {
		b := NewBuilder()
		value := float64(2.718281828459045)
		result := b.AddFloat(value) // No size specified

		if result != b {
			t.Error("Expected AddFloat() to return the same builder instance")
		}

		segment := b.segments[0]
		if segment.Type != bitstring.TypeFloat {
			t.Errorf("Expected segment type 'float', got '%s'", segment.Type)
		}

		// When size is not specified, it should use default size for float64
		if segment.Size != 64 {
			t.Errorf("Expected segment size 64, got %d", segment.Size)
		}

		// SizeSpecified should be false when using default
		if segment.SizeSpecified {
			t.Error("Expected SizeSpecified to be false when using default size")
		}
	})

	t.Run("Float with interface value without size specified", func(t *testing.T) {
		b := NewBuilder()
		var value interface{} = float32(1.618)
		result := b.AddFloat(value) // No size specified

		if result != b {
			t.Error("Expected AddFloat() to return the same builder instance")
		}

		segment := b.segments[0]
		if segment.Type != bitstring.TypeFloat {
			t.Errorf("Expected segment type 'float', got '%s'", segment.Type)
		}

		// When size is not specified, it should use default size for float32
		if segment.Size != 32 {
			t.Errorf("Expected segment size 32, got %d", segment.Size)
		}

		// SizeSpecified should be false when using default
		if segment.SizeSpecified {
			t.Error("Expected SizeSpecified to be false when using default size")
		}
	})
}

// TestBuilder_AddFloat_AdditionalCoverage tests additional scenarios for AddFloat
func TestBuilder_AddFloat_AdditionalCoverage(t *testing.T) {
	t.Run("Float32 with explicit size and type", func(t *testing.T) {
		b := NewBuilder()
		value := float32(3.14159)
		result := b.AddFloat(value, bitstring.WithSize(32), bitstring.WithType("custom_float"))

		if result != b {
			t.Error("Expected AddFloat() to return the same builder instance")
		}

		if len(b.segments) != 1 {
			t.Fatalf("Expected 1 segment, got %d", len(b.segments))
		}

		segment := b.segments[0]
		if segment.Type != "custom_float" {
			t.Errorf("Expected segment type 'custom_float', got '%s'", segment.Type)
		}
		if segment.Size != 32 {
			t.Errorf("Expected segment size 32, got %d", segment.Size)
		}
		if !segment.SizeSpecified {
			t.Error("Expected SizeSpecified to be true")
		}
	})

	t.Run("Float64 with little endianness", func(t *testing.T) {
		b := NewBuilder()
		value := float64(2.718281828459045)
		result := b.AddFloat(value, bitstring.WithSize(64), bitstring.WithEndianness(bitstring.EndiannessLittle))

		if result != b {
			t.Error("Expected AddFloat() to return the same builder instance")
		}

		segment := b.segments[0]
		if segment.Endianness != bitstring.EndiannessLittle {
			t.Errorf("Expected segment endianness %s, got %s", bitstring.EndiannessLittle, segment.Endianness)
		}
	})

	t.Run("Float with unit", func(t *testing.T) {
		b := NewBuilder()
		value := float32(1.618)
		result := b.AddFloat(value, bitstring.WithSize(32), bitstring.WithUnit(16))

		if result != b {
			t.Error("Expected AddFloat() to return the same builder instance")
		}

		segment := b.segments[0]
		if segment.Unit != 16 {
			t.Errorf("Expected segment unit 16, got %d", segment.Unit)
		}
	})

	t.Run("Float with all options", func(t *testing.T) {
		b := NewBuilder()
		value := float64(123.456)
		result := b.AddFloat(value,
			bitstring.WithSize(64),
			bitstring.WithType("double"),
			bitstring.WithEndianness(bitstring.EndiannessBig),
			bitstring.WithUnit(32))

		if result != b {
			t.Error("Expected AddFloat() to return the same builder instance")
		}

		segment := b.segments[0]
		if segment.Type != "double" {
			t.Errorf("Expected segment type 'double', got '%s'", segment.Type)
		}
		if segment.Size != 64 {
			t.Errorf("Expected segment size 64, got %d", segment.Size)
		}
		if segment.Endianness != bitstring.EndiannessBig {
			t.Errorf("Expected segment endianness %s, got %s", bitstring.EndiannessBig, segment.Endianness)
		}
		if segment.Unit != 32 {
			t.Errorf("Expected segment unit 32, got %d", segment.Unit)
		}
		if !segment.SizeSpecified {
			t.Error("Expected SizeSpecified to be true")
		}
	})

	t.Run("Zero float value", func(t *testing.T) {
		b := NewBuilder()
		value := float32(0.0)
		result := b.AddFloat(value)

		if result != b {
			t.Error("Expected AddFloat() to return the same builder instance")
		}

		segment := b.segments[0]
		if segment.Value != value {
			t.Errorf("Expected segment value %v, got %v", value, segment.Value)
		}
	})

	t.Run("Negative float value", func(t *testing.T) {
		b := NewBuilder()
		value := float32(-3.14159)
		result := b.AddFloat(value)

		if result != b {
			t.Error("Expected AddFloat() to return the same builder instance")
		}

		segment := b.segments[0]
		if segment.Value != value {
			t.Errorf("Expected segment value %v, got %v", value, segment.Value)
		}
	})

	t.Run("Very large float64 value", func(t *testing.T) {
		b := NewBuilder()
		value := float64(1.7976931348623157e+308) // Max float64
		result := b.AddFloat(value, bitstring.WithSize(64))

		if result != b {
			t.Error("Expected AddFloat() to return the same builder instance")
		}

		segment := b.segments[0]
		if segment.Value != value {
			t.Errorf("Expected segment value %v, got %v", value, segment.Value)
		}
		if segment.Size != 64 {
			t.Errorf("Expected segment size 64, got %d", segment.Size)
		}
	})

	t.Run("Very small float64 value", func(t *testing.T) {
		b := NewBuilder()
		value := float64(2.2250738585072014e-308) // Min positive float64
		result := b.AddFloat(value, bitstring.WithSize(64))

		if result != b {
			t.Error("Expected AddFloat() to return the same builder instance")
		}

		segment := b.segments[0]
		if segment.Value != value {
			t.Errorf("Expected segment value %v, got %v", value, segment.Value)
		}
	})
}

// TestBuilder_encodeFloat_MissingNativeEndianness tests native endianness paths in encodeFloat
func TestBuilder_encodeFloat_MissingNativeEndianness(t *testing.T) {
	t.Run("Encode float32 with native endianness", func(t *testing.T) {
		w := newBitWriter()
		value := float32(3.14159)
		segment := &bitstring.Segment{
			Value:         value,
			Type:          bitstring.TypeFloat,
			Size:          32,
			SizeSpecified: true,
			Endianness:    bitstring.EndiannessNative,
		}

		err := encodeFloat(w, segment)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		data, totalBits := w.final()
		if totalBits != 32 {
			t.Errorf("Expected totalBits 32, got %d", totalBits)
		}

		if len(data) != 4 {
			t.Errorf("Expected 4 bytes, got %d", len(data))
		}

		// The actual byte order depends on the native endianness
		t.Logf("Native endianness float32 result: %v", data)
	})

	t.Run("Encode float64 with native endianness", func(t *testing.T) {
		w := newBitWriter()
		value := float64(2.718281828459045)
		segment := &bitstring.Segment{
			Value:         value,
			Type:          bitstring.TypeFloat,
			Size:          64,
			SizeSpecified: true,
			Endianness:    bitstring.EndiannessNative,
		}

		err := encodeFloat(w, segment)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		data, totalBits := w.final()
		if totalBits != 64 {
			t.Errorf("Expected totalBits 64, got %d", totalBits)
		}

		if len(data) != 8 {
			t.Errorf("Expected 8 bytes, got %d", len(data))
		}

		// The actual byte order depends on the native endianness
		t.Logf("Native endianness float64 result: %v", data)
	})

	t.Run("Encode float with int value type conversion", func(t *testing.T) {
		w := newBitWriter()
		segment := &bitstring.Segment{
			Value:         42, // int value
			Type:          bitstring.TypeFloat,
			Size:          32,
			SizeSpecified: true,
		}

		err := encodeFloat(w, segment)
		// This should fail because int is not a supported float type
		if err == nil {
			t.Error("Expected error for int value type")
		}

		if bitStringErr, ok := err.(*bitstring.BitStringError); ok {
			if bitStringErr.Code != bitstring.CodeTypeMismatch {
				t.Errorf("Expected error code %s, got %s", bitstring.CodeTypeMismatch, bitStringErr.Code)
			}
		}
	})

	t.Run("Encode float with string value type", func(t *testing.T) {
		w := newBitWriter()
		segment := &bitstring.Segment{
			Value:         "3.14", // string value
			Type:          bitstring.TypeFloat,
			Size:          32,
			SizeSpecified: true,
		}

		err := encodeFloat(w, segment)
		// This should fail because string is not a supported float type
		if err == nil {
			t.Error("Expected error for string value type")
		}

		if bitStringErr, ok := err.(*bitstring.BitStringError); ok {
			if bitStringErr.Code != bitstring.CodeTypeMismatch {
				t.Errorf("Expected error code %s, got %s", bitstring.CodeTypeMismatch, bitStringErr.Code)
			}
		}
	})
}
