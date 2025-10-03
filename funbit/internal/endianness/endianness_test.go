package endianness

import (
	"testing"
)

func TestGetNativeEndianness(t *testing.T) {
	result := GetNativeEndianness()

	if result != "big" && result != "little" {
		t.Errorf("Expected 'big' or 'little', got '%s'", result)
	}

	// The result should be consistent
	result2 := GetNativeEndianness()
	if result != result2 {
		t.Errorf("Inconsistent results: '%s' and '%s'", result, result2)
	}
}

func TestConvertEndianness(t *testing.T) {
	t.Run("Same endianness", func(t *testing.T) {
		data := []byte{0x12, 0x34}
		result, err := ConvertEndianness(data, "big", "big", 16)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if !bytesEqual(result, data) {
			t.Errorf("Expected same data %v, got %v", data, result)
		}
	})

	t.Run("Big to little", func(t *testing.T) {
		data := []byte{0x12, 0x34}
		result, err := ConvertEndianness(data, "big", "little", 16)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		expected := []byte{0x34, 0x12}
		if !bytesEqual(result, expected) {
			t.Errorf("Expected %v, got %v", expected, result)
		}
	})

	t.Run("Little to big", func(t *testing.T) {
		data := []byte{0x34, 0x12}
		result, err := ConvertEndianness(data, "little", "big", 16)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		expected := []byte{0x12, 0x34}
		if !bytesEqual(result, expected) {
			t.Errorf("Expected %v, got %v", expected, result)
		}
	})

	t.Run("Native endianness resolution", func(t *testing.T) {
		data := []byte{0x12, 0x34}
		native := GetNativeEndianness()
		opposite := "little"
		if native == "little" {
			opposite = "big"
		}

		// Test native to opposite
		result, err := ConvertEndianness(data, "native", opposite, 16)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		// Test opposite to native
		result2, err := ConvertEndianness(result, opposite, "native", 16)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if !bytesEqual(result2, data) {
			t.Errorf("Expected round-trip conversion to return original data %v, got %v", data, result2)
		}
	})

	t.Run("Multiple bytes", func(t *testing.T) {
		data := []byte{0x12, 0x34, 0x56, 0x78}
		result, err := ConvertEndianness(data, "big", "little", 32)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		expected := []byte{0x78, 0x56, 0x34, 0x12}
		if !bytesEqual(result, expected) {
			t.Errorf("Expected %v, got %v", expected, result)
		}
	})
}

func TestEncodeWithEndianness(t *testing.T) {
	t.Run("Uint64 16-bit big endian", func(t *testing.T) {
		value := uint64(0x1234)
		result, err := EncodeWithEndianness(value, 16, "big")

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		expected := []byte{0x12, 0x34}
		if !bytesEqual(result, expected) {
			t.Errorf("Expected %v, got %v", expected, result)
		}
	})

	t.Run("Uint64 16-bit little endian", func(t *testing.T) {
		value := uint64(0x1234)
		result, err := EncodeWithEndianness(value, 16, "little")

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		expected := []byte{0x34, 0x12}
		if !bytesEqual(result, expected) {
			t.Errorf("Expected %v, got %v", expected, result)
		}
	})

	t.Run("Uint64 32-bit big endian", func(t *testing.T) {
		value := uint64(0x12345678)
		result, err := EncodeWithEndianness(value, 32, "big")

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		expected := []byte{0x12, 0x34, 0x56, 0x78}
		if !bytesEqual(result, expected) {
			t.Errorf("Expected %v, got %v", expected, result)
		}
	})

	t.Run("Uint64 32-bit little endian", func(t *testing.T) {
		value := uint64(0x12345678)
		result, err := EncodeWithEndianness(value, 32, "little")

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		expected := []byte{0x78, 0x56, 0x34, 0x12}
		if !bytesEqual(result, expected) {
			t.Errorf("Expected %v, got %v", expected, result)
		}
	})

	t.Run("Uint64 64-bit big endian", func(t *testing.T) {
		value := uint64(0x123456789ABCDEF0)
		result, err := EncodeWithEndianness(value, 64, "big")

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		expected := []byte{0x12, 0x34, 0x56, 0x78, 0x9A, 0xBC, 0xDE, 0xF0}
		if !bytesEqual(result, expected) {
			t.Errorf("Expected %v, got %v", expected, result)
		}
	})

	t.Run("Uint64 64-bit little endian", func(t *testing.T) {
		value := uint64(0x123456789ABCDEF0)
		result, err := EncodeWithEndianness(value, 64, "little")

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		expected := []byte{0xF0, 0xDE, 0xBC, 0x9A, 0x78, 0x56, 0x34, 0x12}
		if !bytesEqual(result, expected) {
			t.Errorf("Expected %v, got %v", expected, result)
		}
	})

	t.Run("Int64 16-bit big endian", func(t *testing.T) {
		value := int64(0x1234)
		result, err := EncodeWithEndianness(value, 16, "big")

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		expected := []byte{0x12, 0x34}
		if !bytesEqual(result, expected) {
			t.Errorf("Expected %v, got %v", expected, result)
		}
	})

	t.Run("Int64 32-bit little endian", func(t *testing.T) {
		value := int64(0x12345678)
		result, err := EncodeWithEndianness(value, 32, "little")

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		expected := []byte{0x78, 0x56, 0x34, 0x12}
		if !bytesEqual(result, expected) {
			t.Errorf("Expected %v, got %v", expected, result)
		}
	})

	t.Run("Native endianness", func(t *testing.T) {
		value := uint64(0x1234)
		result, err := EncodeWithEndianness(value, 16, "native")

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if len(result) != 2 {
			t.Errorf("Expected 2 bytes, got %d", len(result))
		}
	})

	t.Run("Unsupported size", func(t *testing.T) {
		value := uint64(0x1234)
		_, err := EncodeWithEndianness(value, 8, "big")

		if err == nil {
			t.Error("Expected error for unsupported size")
		}

		if endiannessErr, ok := err.(*EndiannessError); ok {
			if endiannessErr.Code != "UNSUPPORTED_SIZE" {
				t.Errorf("Expected error code 'UNSUPPORTED_SIZE', got '%s'", endiannessErr.Code)
			}
			if endiannessErr.Size != 8 {
				t.Errorf("Expected error size 8, got %d", endiannessErr.Size)
			}
		} else {
			t.Errorf("Expected EndiannessError, got %T", err)
		}
	})

	t.Run("Unsupported type", func(t *testing.T) {
		value := "not a number"
		_, err := EncodeWithEndianness(value, 16, "big")

		if err == nil {
			t.Error("Expected error for unsupported type")
		}

		if endiannessErr, ok := err.(*EndiannessError); ok {
			if endiannessErr.Code != "UNSUPPORTED_TYPE" {
				t.Errorf("Expected error code 'UNSUPPORTED_TYPE', got '%s'", endiannessErr.Code)
			}
			if endiannessErr.Type != value {
				t.Errorf("Expected error type %v, got %v", value, endiannessErr.Type)
			}
		} else {
			t.Errorf("Expected EndiannessError, got %T", err)
		}
	})
}

func TestDecodeWithEndianness(t *testing.T) {
	t.Run("16-bit big endian", func(t *testing.T) {
		data := []byte{0x12, 0x34}
		result, err := DecodeWithEndianness(data, 16, "big")

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if result != uint64(0x1234) {
			t.Errorf("Expected 0x1234, got 0x%X", result)
		}
	})

	t.Run("16-bit little endian", func(t *testing.T) {
		data := []byte{0x34, 0x12}
		result, err := DecodeWithEndianness(data, 16, "little")

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if result != uint64(0x1234) {
			t.Errorf("Expected 0x1234, got 0x%X", result)
		}
	})

	t.Run("32-bit big endian", func(t *testing.T) {
		data := []byte{0x12, 0x34, 0x56, 0x78}
		result, err := DecodeWithEndianness(data, 32, "big")

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if result != uint64(0x12345678) {
			t.Errorf("Expected 0x12345678, got 0x%X", result)
		}
	})

	t.Run("32-bit little endian", func(t *testing.T) {
		data := []byte{0x78, 0x56, 0x34, 0x12}
		result, err := DecodeWithEndianness(data, 32, "little")

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if result != uint64(0x12345678) {
			t.Errorf("Expected 0x12345678, got 0x%X", result)
		}
	})

	t.Run("64-bit big endian", func(t *testing.T) {
		data := []byte{0x12, 0x34, 0x56, 0x78, 0x9A, 0xBC, 0xDE, 0xF0}
		result, err := DecodeWithEndianness(data, 64, "big")

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if result != uint64(0x123456789ABCDEF0) {
			t.Errorf("Expected 0x123456789ABCDEF0, got 0x%X", result)
		}
	})

	t.Run("64-bit little endian", func(t *testing.T) {
		data := []byte{0xF0, 0xDE, 0xBC, 0x9A, 0x78, 0x56, 0x34, 0x12}
		result, err := DecodeWithEndianness(data, 64, "little")

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if result != uint64(0x123456789ABCDEF0) {
			t.Errorf("Expected 0x123456789ABCDEF0, got 0x%X", result)
		}
	})

	t.Run("Native endianness", func(t *testing.T) {
		data := []byte{0x12, 0x34}
		result, err := DecodeWithEndianness(data, 16, "native")

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if result != uint64(0x1234) && result != uint64(0x3412) {
			t.Errorf("Expected valid result, got 0x%X", result)
		}
	})

	t.Run("Unsupported size", func(t *testing.T) {
		data := []byte{0x12}
		_, err := DecodeWithEndianness(data, 8, "big")

		if err == nil {
			t.Error("Expected error for unsupported size")
		}

		if endiannessErr, ok := err.(*EndiannessError); ok {
			if endiannessErr.Code != "UNSUPPORTED_SIZE" {
				t.Errorf("Expected error code 'UNSUPPORTED_SIZE', got '%s'", endiannessErr.Code)
			}
			if endiannessErr.Size != 8 {
				t.Errorf("Expected error size 8, got %d", endiannessErr.Size)
			}
		} else {
			t.Errorf("Expected EndiannessError, got %T", err)
		}
	})
}

func TestReverseBytes(t *testing.T) {
	t.Run("Empty slice", func(t *testing.T) {
		data := []byte{}
		reverseBytes(data)
		// Should not panic and data should remain empty
		if len(data) != 0 {
			t.Errorf("Expected empty slice, got %v", data)
		}
	})

	t.Run("Single byte", func(t *testing.T) {
		data := []byte{0xAB}
		original := make([]byte, len(data))
		copy(original, data)
		reverseBytes(data)
		if !bytesEqual(data, original) {
			t.Errorf("Single byte should not change: %v", data)
		}
	})

	t.Run("Two bytes", func(t *testing.T) {
		data := []byte{0x12, 0x34}
		reverseBytes(data)
		expected := []byte{0x34, 0x12}
		if !bytesEqual(data, expected) {
			t.Errorf("Expected %v, got %v", expected, data)
		}
	})

	t.Run("Three bytes", func(t *testing.T) {
		data := []byte{0x12, 0x34, 0x56}
		reverseBytes(data)
		expected := []byte{0x56, 0x34, 0x12}
		if !bytesEqual(data, expected) {
			t.Errorf("Expected %v, got %v", expected, data)
		}
	})

	t.Run("Four bytes", func(t *testing.T) {
		data := []byte{0x12, 0x34, 0x56, 0x78}
		reverseBytes(data)
		expected := []byte{0x78, 0x56, 0x34, 0x12}
		if !bytesEqual(data, expected) {
			t.Errorf("Expected %v, got %v", expected, data)
		}
	})

	t.Run("Eight bytes", func(t *testing.T) {
		data := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
		reverseBytes(data)
		expected := []byte{0x08, 0x07, 0x06, 0x05, 0x04, 0x03, 0x02, 0x01}
		if !bytesEqual(data, expected) {
			t.Errorf("Expected %v, got %v", expected, data)
		}
	})
}

func TestIsLittleEndian(t *testing.T) {
	result := isLittleEndian()

	// Should return either true or false
	if result != true && result != false {
		t.Errorf("Expected true or false, got %v", result)
	}

	// Should be consistent across calls
	result2 := isLittleEndian()
	if result != result2 {
		t.Errorf("Inconsistent results: %v and %v", result, result2)
	}

	// Should match GetNativeEndianness
	native := GetNativeEndianness()
	expected := (native == "little")
	if result != expected {
		t.Errorf("isLittleEndian() = %v but GetNativeEndianness() = '%s'", result, native)
	}
}

func TestEndiannessError(t *testing.T) {
	t.Run("Error with size", func(t *testing.T) {
		err := unsupportedSizeError(16)

		if endiannessErr, ok := err.(*EndiannessError); ok {
			if endiannessErr.Code != "UNSUPPORTED_SIZE" {
				t.Errorf("Expected code 'UNSUPPORTED_SIZE', got '%s'", endiannessErr.Code)
			}
			if endiannessErr.Size != 16 {
				t.Errorf("Expected size 16, got %d", endiannessErr.Size)
			}
			if endiannessErr.Type != nil {
				t.Errorf("Expected nil type, got %v", endiannessErr.Type)
			}
			expectedMsg := "unsupported size for endianness conversion (size: " + string(rune('0'+16)) + ")"
			if err.Error() != expectedMsg {
				t.Errorf("Expected message '%s', got '%s'", expectedMsg, err.Error())
			}
		} else {
			t.Errorf("Expected EndiannessError, got %T", err)
		}
	})

	t.Run("Error with type", func(t *testing.T) {
		value := "test_type"
		err := unsupportedTypeError(value)

		if endiannessErr, ok := err.(*EndiannessError); ok {
			if endiannessErr.Code != "UNSUPPORTED_TYPE" {
				t.Errorf("Expected code 'UNSUPPORTED_TYPE', got '%s'", endiannessErr.Code)
			}
			if endiannessErr.Size != 0 {
				t.Errorf("Expected size 0, got %d", endiannessErr.Size)
			}
			if endiannessErr.Type != value {
				t.Errorf("Expected type %v, got %v", value, endiannessErr.Type)
			}
			expectedMsg := "unsupported type for endianness conversion (type: " + value + ")"
			if err.Error() != expectedMsg {
				t.Errorf("Expected message '%s', got '%s'", expectedMsg, err.Error())
			}
		} else {
			t.Errorf("Expected EndiannessError, got %T", err)
		}
	})

	t.Run("Error without size or type", func(t *testing.T) {
		err := &EndiannessError{
			Code:    "TEST_ERROR",
			Message: "test message",
		}

		if err.Error() != "test message" {
			t.Errorf("Expected 'test message', got '%s'", err.Error())
		}
	})
}

// Helper function to compare byte slices
func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestEndianness_EdgeCases(t *testing.T) {
	t.Run("ConvertEndianness empty data", func(t *testing.T) {
		data := []byte{}
		result, err := ConvertEndianness(data, "big", "little", 0)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if len(result) != 0 {
			t.Errorf("Expected empty result, got %v", result)
		}
	})

	t.Run("ConvertEndianness native to native", func(t *testing.T) {
		data := []byte{0x12, 0x34}
		result, err := ConvertEndianness(data, "native", "native", 16)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if !bytesEqual(result, data) {
			t.Errorf("Expected same data %v, got %v", data, result)
		}
	})

	t.Run("GetNativeEndianness coverage", func(t *testing.T) {
		// Call multiple times to ensure both branches are covered
		result1 := GetNativeEndianness()
		result2 := GetNativeEndianness()
		result3 := GetNativeEndianness()

		if result1 != result2 || result2 != result3 {
			t.Errorf("Inconsistent results: %s, %s, %s", result1, result2, result3)
		}

		// Ensure it's a valid result
		if result1 != "big" && result1 != "little" {
			t.Errorf("Invalid result: %s", result1)
		}
	})

	t.Run("EncodeWithEndianness coverage edge cases", func(t *testing.T) {
		// Test int64 negative values
		value := interface{}(int64(-1))
		result, err := EncodeWithEndianness(value, 16, "big")

		if err != nil {
			t.Errorf("Expected no error for negative int64, got %v", err)
		}

		if len(result) != 2 {
			t.Errorf("Expected 2 bytes, got %d", len(result))
		}

		// Test uint64 maximum value
		value = interface{}(uint64(0xFFFFFFFFFFFFFFFF))
		result, err = EncodeWithEndianness(value, 64, "little")

		if err != nil {
			t.Errorf("Expected no error for max uint64, got %v", err)
		}

		if len(result) != 8 {
			t.Errorf("Expected 8 bytes, got %d", len(result))
		}
	})

	t.Run("DecodeWithEndianness coverage edge cases", func(t *testing.T) {
		// Test with maximum 16-bit value
		data := []byte{0xFF, 0xFF}
		result, err := DecodeWithEndianness(data, 16, "big")

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if result != uint64(0xFFFF) {
			t.Errorf("Expected 0xFFFF, got 0x%X", result)
		}

		// Test with maximum 32-bit value
		data = []byte{0xFF, 0xFF, 0xFF, 0xFF}
		result, err = DecodeWithEndianness(data, 32, "little")

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if result != uint64(0xFFFFFFFF) {
			t.Errorf("Expected 0xFFFFFFFF, got 0x%X", result)
		}

		// Test with maximum 64-bit value
		data = []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}
		result, err = DecodeWithEndianness(data, 64, "big")

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if result != uint64(0xFFFFFFFFFFFFFFFF) {
			t.Errorf("Expected 0xFFFFFFFFFFFFFFFF, got 0x%X", result)
		}
	})

	t.Run("ReverseBytes odd length", func(t *testing.T) {
		data := []byte{0x01, 0x02, 0x03, 0x04, 0x05}
		original := make([]byte, len(data))
		copy(original, data)

		reverseBytes(data)
		reverseBytes(data) // Reverse twice should get back original

		if !bytesEqual(data, original) {
			t.Errorf("Double reverse should return original: got %v, expected %v", data, original)
		}
	})

	t.Run("EndiannessError message formatting", func(t *testing.T) {
		// Test error with size 0 (should not include size in message)
		err := &EndiannessError{
			Code:    "TEST",
			Message: "test message",
			Size:    0,
		}

		msg := err.Error()
		if msg != "test message" {
			t.Errorf("Expected 'test message', got '%s'", msg)
		}

		// Test error with non-zero size
		err = &EndiannessError{
			Code:    "TEST",
			Message: "test message",
			Size:    32,
		}

		msg = err.Error()
		expected := "test message (size: " + string(rune('0'+32)) + ")"
		if msg != expected {
			t.Errorf("Expected '%s', got '%s'", expected, msg)
		}

		// Test error with string type
		err = &EndiannessError{
			Code:    "TEST",
			Message: "test message",
			Type:    "string_type",
		}

		msg = err.Error()
		expected = "test message (type: string_type)"
		if msg != expected {
			t.Errorf("Expected '%s', got '%s'", expected, msg)
		}
	})
}

func TestEndianness_MissingCoverage(t *testing.T) {
	t.Run("EncodeWithEndianness unsupported size for uint64", func(t *testing.T) {
		value := interface{}(uint64(123))
		_, err := EncodeWithEndianness(value, 8, "big") // 8 bits is not supported

		if err == nil {
			t.Error("Expected error for unsupported size")
		}

		if err.Error() != "unsupported size for endianness conversion (size: "+string(rune('0'+8))+")" {
			t.Errorf("Expected unsupported size error, got: %v", err)
		}
	})

	t.Run("EncodeWithEndianness unsupported size for int64", func(t *testing.T) {
		value := interface{}(int64(-123))
		_, err := EncodeWithEndianness(value, 24, "little") // 24 bits is not supported

		if err == nil {
			t.Error("Expected error for unsupported size")
		}

		if err.Error() != "unsupported size for endianness conversion (size: "+string(rune('0'+24))+")" {
			t.Errorf("Expected unsupported size error, got: %v", err)
		}
	})

	t.Run("EncodeWithEndianness unsupported type", func(t *testing.T) {
		value := interface{}("string_value")
		_, err := EncodeWithEndianness(value, 16, "big")

		if err == nil {
			t.Error("Expected error for unsupported type")
		}

		if err.Error() != "unsupported type for endianness conversion (type: string_value)" {
			t.Errorf("Expected unsupported type error, got: %v", err)
		}
	})

	t.Run("ConvertEndianness native resolution to same endianness", func(t *testing.T) {
		data := []byte{0x12, 0x34}
		native := GetNativeEndianness()

		// Test when both from and to resolve to native (and are the same)
		result, err := ConvertEndianness(data, "native", native, 16)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if !bytesEqual(result, data) {
			t.Errorf("Expected same data %v, got %v", data, result)
		}
	})

	t.Run("ConvertEndianness different native resolution", func(t *testing.T) {
		data := []byte{0x12, 0x34}
		native := GetNativeEndianness()
		opposite := "big"
		if native == "big" {
			opposite = "little"
		}

		// Test when from is native and to is opposite
		result, err := ConvertEndianness(data, "native", opposite, 16)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		// The result should be different from original (reversed)
		if bytesEqual(result, data) {
			t.Error("Expected reversed data, got same data")
		}

		// Verify it's actually reversed
		expected := []byte{data[1], data[0]}
		if !bytesEqual(result, expected) {
			t.Errorf("Expected reversed data %v, got %v", expected, result)
		}
	})

	t.Run("EndiannessError with both size and type", func(t *testing.T) {
		// Test error with both size and type (size should take precedence)
		err := &EndiannessError{
			Code:    "TEST",
			Message: "test message",
			Size:    16,
			Type:    "some_type",
		}

		msg := err.Error()
		expected := "test message (size: " + string(rune('0'+16)) + ")"
		if msg != expected {
			t.Errorf("Expected '%s', got '%s'", expected, msg)
		}
	})
}
