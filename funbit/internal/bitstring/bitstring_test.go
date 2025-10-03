package bitstring

import (
	"reflect"
	"testing"
)

func TestBitString_NewBitString(t *testing.T) {
	bs := NewBitString()

	if bs == nil {
		t.Fatal("Expected NewBitString() to return non-nil")
	}

	if bs.Length() != 0 {
		t.Errorf("Expected empty bitstring length 0, got %d", bs.Length())
	}

	if !bs.IsEmpty() {
		t.Error("Expected empty bitstring to be empty")
	}

	if !bs.IsBinary() {
		t.Error("Expected empty bitstring to be binary (0 is multiple of 8)")
	}
}

func TestBitString_NewBitStringFromBytes(t *testing.T) {
	tests := []struct {
		name           string
		data           []byte
		expectedLen    uint
		expectedBinary bool
		expectedBytes  []byte
	}{
		{
			name:           "empty bytes",
			data:           []byte{},
			expectedLen:    0,
			expectedBinary: true,
			expectedBytes:  []byte{},
		},
		{
			name:           "single byte",
			data:           []byte{42},
			expectedLen:    8,
			expectedBinary: true,
			expectedBytes:  []byte{42},
		},
		{
			name:           "multiple bytes",
			data:           []byte{1, 2, 3, 4},
			expectedLen:    32,
			expectedBinary: true,
			expectedBytes:  []byte{1, 2, 3, 4},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bs := NewBitStringFromBytes(tt.data)

			if bs == nil {
				t.Fatal("Expected NewBitStringFromBytes() to return non-nil")
			}

			if bs.Length() != tt.expectedLen {
				t.Errorf("Expected length %d, got %d", tt.expectedLen, bs.Length())
			}

			if bs.IsEmpty() != (tt.expectedLen == 0) {
				t.Errorf("Expected IsEmpty() to be %v for length %d", tt.expectedLen == 0, tt.expectedLen)
			}

			if bs.IsBinary() != tt.expectedBinary {
				t.Errorf("Expected IsBinary() to be %v", tt.expectedBinary)
			}

			bytes := bs.ToBytes()
			if !reflect.DeepEqual(bytes, tt.expectedBytes) {
				t.Errorf("Expected bytes %v, got %v", tt.expectedBytes, bytes)
			}
		})
	}
}

func TestBitString_NewBitStringFromBits(t *testing.T) {
	tests := []struct {
		name           string
		data           []byte
		length         uint
		expectedLen    uint
		expectedBinary bool
		wantErr        bool
	}{
		{
			name:           "zero bits",
			data:           []byte{},
			length:         0,
			expectedLen:    0,
			expectedBinary: true,
			wantErr:        false,
		},
		{
			name:           "4 bits (half byte)",
			data:           []byte{0b10110000},
			length:         4,
			expectedLen:    4,
			expectedBinary: false,
			wantErr:        false,
		},
		{
			name:           "12 bits (1.5 bytes)",
			data:           []byte{0xFF, 0xF0},
			length:         12,
			expectedLen:    12,
			expectedBinary: false,
			wantErr:        false,
		},
		{
			name:           "16 bits (exactly 2 bytes)",
			data:           []byte{0xFF, 0xAA},
			length:         16,
			expectedLen:    16,
			expectedBinary: true,
			wantErr:        false,
		},
		{
			name:           "insufficient data",
			data:           []byte{0xFF},
			length:         16,
			expectedLen:    0,
			expectedBinary: false,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bs := NewBitStringFromBits(tt.data, tt.length)

			if tt.wantErr {
				if bs != nil {
					t.Error("Expected NewBitStringFromBits() to return nil on error")
				}
				return
			}

			if bs == nil {
				t.Fatal("Expected NewBitStringFromBits() to return non-nil")
			}

			if bs.Length() != tt.expectedLen {
				t.Errorf("Expected length %d, got %d", tt.expectedLen, bs.Length())
			}

			if bs.IsEmpty() != (tt.expectedLen == 0) {
				t.Errorf("Expected IsEmpty() to be %v for length %d", tt.expectedLen == 0, tt.expectedLen)
			}

			if bs.IsBinary() != tt.expectedBinary {
				t.Errorf("Expected IsBinary() to be %v", tt.expectedBinary)
			}
		})
	}
}

func TestBitString_Clone(t *testing.T) {
	original := NewBitStringFromBytes([]byte{1, 2, 3})

	cloned := original.Clone()

	if cloned == nil {
		t.Fatal("Expected Clone() to return non-nil")
	}

	if cloned.Length() != original.Length() {
		t.Errorf("Expected clone to have same length %d, got %d", original.Length(), cloned.Length())
	}

	if cloned.IsEmpty() != original.IsEmpty() {
		t.Errorf("Expected clone to have same IsEmpty() value %v", original.IsEmpty())
	}

	if cloned.IsBinary() != original.IsBinary() {
		t.Errorf("Expected clone to have same IsBinary() value %v", original.IsBinary())
	}

	originalBytes := original.ToBytes()
	clonedBytes := cloned.ToBytes()

	if !reflect.DeepEqual(originalBytes, clonedBytes) {
		t.Errorf("Expected clone to have same bytes %v, got %v", originalBytes, clonedBytes)
	}
}

func TestBitString_EmptyString(t *testing.T) {
	bs := NewBitString()

	if !bs.IsEmpty() {
		t.Error("Expected empty bitstring to be empty")
	}

	if bs.Length() != 0 {
		t.Errorf("Expected empty bitstring length 0, got %d", bs.Length())
	}

	if !bs.IsBinary() {
		t.Error("Expected empty bitstring to be binary (0 is multiple of 8)")
	}

	bytes := bs.ToBytes()
	if len(bytes) != 0 {
		t.Errorf("Expected empty bytes, got %v", bytes)
	}
}

func TestSegmentOptions(t *testing.T) {
	// Test WithSize
	t.Run("WithSize", func(t *testing.T) {
		segment := NewSegment(42, WithSize(16))
		if segment.Size != 16 {
			t.Errorf("Expected size 16, got %d", segment.Size)
		}
		if !segment.SizeSpecified {
			t.Error("Expected SizeSpecified to be true")
		}
	})

	// Test WithType
	t.Run("WithType", func(t *testing.T) {
		segment := NewSegment(42, WithType(TypeFloat))
		if segment.Type != TypeFloat {
			t.Errorf("Expected type %s, got %s", TypeFloat, segment.Type)
		}
	})

	// Test WithSigned
	t.Run("WithSigned", func(t *testing.T) {
		segment := NewSegment(42, WithSigned(Signed))
		if segment.Signed != Signed {
			t.Errorf("Expected signed %v, got %v", Signed, segment.Signed)
		}
	})

	// Test WithEndianness
	t.Run("WithEndianness", func(t *testing.T) {
		segment := NewSegment(42, WithEndianness(EndiannessLittle))
		if segment.Endianness != EndiannessLittle {
			t.Errorf("Expected endianness %s, got %s", EndiannessLittle, segment.Endianness)
		}
	})

	// Test WithUnit
	t.Run("WithUnit", func(t *testing.T) {
		segment := NewSegment(42, WithUnit(8))
		if segment.Unit != 8 {
			t.Errorf("Expected unit 8, got %d", segment.Unit)
		}
		if !segment.UnitSpecified {
			t.Error("Expected UnitSpecified to be true")
		}
	})

	// Test WithDynamicSize
	t.Run("WithDynamicSize", func(t *testing.T) {
		sizeVar := uint(32)
		segment := NewSegment(42, WithDynamicSize(&sizeVar))
		if segment.DynamicSize != &sizeVar {
			t.Error("Expected DynamicSize to be set")
		}
		if !segment.IsDynamic {
			t.Error("Expected IsDynamic to be true")
		}
		// Note: NewSegment sets default size for integer type, so SizeSpecified will be true
		// but IsDynamic overrides this behavior
		if !segment.IsDynamic {
			t.Error("Expected IsDynamic to override SizeSpecified")
		}
	})

	// Test WithDynamicSizeExpression
	t.Run("WithDynamicSizeExpression", func(t *testing.T) {
		expr := "size + 8"
		segment := NewSegment(42, WithDynamicSizeExpression(expr))
		if segment.DynamicExpr != expr {
			t.Errorf("Expected DynamicExpr %s, got %s", expr, segment.DynamicExpr)
		}
		if !segment.IsDynamic {
			t.Error("Expected IsDynamic to be true")
		}
		// Note: NewSegment sets default size for integer type, so SizeSpecified will be true
		// but IsDynamic overrides this behavior
		if !segment.IsDynamic {
			t.Error("Expected IsDynamic to override SizeSpecified")
		}
	})

	// Test combined options
	t.Run("CombinedOptions", func(t *testing.T) {
		segment := NewSegment(42,
			WithSize(32),
			WithType(TypeInteger),
			WithSigned(Signed),
			WithEndianness(EndiannessLittle),
			WithUnit(4),
		)

		if segment.Size != 32 {
			t.Errorf("Expected size 32, got %d", segment.Size)
		}
		if segment.Type != TypeInteger {
			t.Errorf("Expected type %s, got %s", TypeInteger, segment.Type)
		}
		if segment.Signed != Signed {
			t.Errorf("Expected signed %v, got %v", Signed, segment.Signed)
		}
		if segment.Endianness != EndiannessLittle {
			t.Errorf("Expected endianness %s, got %s", EndiannessLittle, segment.Endianness)
		}
		if segment.Unit != 4 {
			t.Errorf("Expected unit 4, got %d", segment.Unit)
		}
	})
}

func TestBitStringError(t *testing.T) {
	// Test NewBitStringError
	t.Run("NewBitStringError", func(t *testing.T) {
		err := NewBitStringError(CodeOverflow, "overflow error")
		if err.Code != CodeOverflow {
			t.Errorf("Expected code %s, got %s", CodeOverflow, err.Code)
		}
		if err.Message != "overflow error" {
			t.Errorf("Expected message 'overflow error', got %s", err.Message)
		}
		if err.Context != nil {
			t.Error("Expected context to be nil")
		}
	})

	// Test NewBitStringErrorWithContext
	t.Run("NewBitStringErrorWithContext", func(t *testing.T) {
		context := map[string]interface{}{"size": 16, "value": 42}
		err := NewBitStringErrorWithContext(CodeInvalidSize, "invalid size", context)
		if err.Code != CodeInvalidSize {
			t.Errorf("Expected code %s, got %s", CodeInvalidSize, err.Code)
		}
		if err.Message != "invalid size" {
			t.Errorf("Expected message 'invalid size', got %s", err.Message)
		}
		if err.Context == nil {
			t.Error("Expected context to be set")
		}
	})

	// Test Error() method without context
	t.Run("ErrorWithoutContext", func(t *testing.T) {
		err := NewBitStringError(CodeOverflow, "overflow error")
		expected := "overflow error"
		if err.Error() != expected {
			t.Errorf("Expected error message '%s', got '%s'", expected, err.Error())
		}
	})

	// Test Error() method with string context
	t.Run("ErrorWithStringContext", func(t *testing.T) {
		err := NewBitStringErrorWithContext(CodeInvalidSize, "invalid size", "additional info")
		expected := "invalid size (context: additional info)"
		if err.Error() != expected {
			t.Errorf("Expected error message '%s', got '%s'", expected, err.Error())
		}
	})

	// Test Error() method with int context
	t.Run("ErrorWithIntContext", func(t *testing.T) {
		err := NewBitStringErrorWithContext(CodeInvalidSize, "invalid size", 42)
		expected := "invalid size (context: 42)"
		if err.Error() != expected {
			t.Errorf("Expected error message '%s', got '%s'", expected, err.Error())
		}
	})

	// Test Error() method with uint context
	t.Run("ErrorWithUintContext", func(t *testing.T) {
		err := NewBitStringErrorWithContext(CodeInvalidSize, "invalid size", uint(16))
		expected := "invalid size (context: 16)"
		if err.Error() != expected {
			t.Errorf("Expected error message '%s', got '%s'", expected, err.Error())
		}
	})

	// Test Error() method with map context
	t.Run("ErrorWithMapContext", func(t *testing.T) {
		context := map[string]interface{}{"size": 16, "value": 42}
		err := NewBitStringErrorWithContext(CodeInvalidSize, "invalid size", context)
		expected := "invalid size (context: map[size:16 value:42])"
		if err.Error() != expected {
			t.Errorf("Expected error message '%s', got '%s'", expected, err.Error())
		}
	})

	// Test Error() method with unknown context type
	t.Run("ErrorWithUnknownContext", func(t *testing.T) {
		type customType struct{ field string }
		context := customType{field: "value"}
		err := NewBitStringErrorWithContext(CodeInvalidSize, "invalid size", context)
		expected := "invalid size (context: {value})"
		if err.Error() != expected {
			t.Errorf("Expected error message '%s', got '%s'", expected, err.Error())
		}
	})
}

func TestValidateSegment(t *testing.T) {
	// Test nil segment
	t.Run("NilSegment", func(t *testing.T) {
		err := ValidateSegment(nil)
		if err == nil {
			t.Error("Expected error for nil segment")
		}
		bsErr, ok := err.(*BitStringError)
		if !ok {
			t.Fatalf("Expected *BitStringError, got %T", err)
		}
		if bsErr.Code != CodeInvalidSegment {
			t.Errorf("Expected code %s, got %s", CodeInvalidSegment, bsErr.Code)
		}
	})

	// Test default type assignment
	t.Run("DefaultType", func(t *testing.T) {
		segment := &Segment{Type: ""}
		err := ValidateSegment(segment)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if segment.Type != TypeInteger {
			t.Errorf("Expected default type %s, got %s", TypeInteger, segment.Type)
		}
	})

	// Test valid size 0 (allowed in BIT_SYNTAX_SPEC.md)
	t.Run("ValidSizeZero", func(t *testing.T) {
		segment := &Segment{Type: TypeInteger, Size: 0, SizeSpecified: true}
		err := ValidateSegment(segment)
		if err != nil {
			t.Errorf("Unexpected error for valid size 0: %v", err)
		}
	})

	// Test valid size 0 for binary
	t.Run("ValidSizeZeroForBinary", func(t *testing.T) {
		segment := &Segment{Type: TypeBinary, Size: 0, SizeSpecified: true}
		err := ValidateSegment(segment)
		if err != nil {
			t.Errorf("Expected no error for binary with size 0, got %v", err)
		}
	})

	// Test invalid unit
	t.Run("InvalidUnit", func(t *testing.T) {
		segment := &Segment{Type: TypeInteger, Unit: 0, UnitSpecified: true}
		err := ValidateSegment(segment)
		if err == nil {
			t.Error("Expected error for invalid unit")
		}
		bsErr, ok := err.(*BitStringError)
		if !ok {
			t.Fatalf("Expected *BitStringError, got %T", err)
		}
		if bsErr.Code != CodeInvalidUnit {
			t.Errorf("Expected code %s, got %s", CodeInvalidUnit, bsErr.Code)
		}
	})

	// Test invalid unit too large
	t.Run("InvalidUnitTooLarge", func(t *testing.T) {
		segment := &Segment{Type: TypeInteger, Unit: 257, UnitSpecified: true}
		err := ValidateSegment(segment)
		if err == nil {
			t.Error("Expected error for invalid unit")
		}
		bsErr, ok := err.(*BitStringError)
		if !ok {
			t.Fatalf("Expected *BitStringError, got %T", err)
		}
		if bsErr.Code != CodeInvalidUnit {
			t.Errorf("Expected code %s, got %s", CodeInvalidUnit, bsErr.Code)
		}
	})

	// Test default endianness
	t.Run("DefaultEndianness", func(t *testing.T) {
		segment := &Segment{Type: TypeInteger, Endianness: ""}
		err := ValidateSegment(segment)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if segment.Endianness != EndiannessBig {
			t.Errorf("Expected default endianness %s, got %s", EndiannessBig, segment.Endianness)
		}
	})

	// Test invalid endianness
	t.Run("InvalidEndianness", func(t *testing.T) {
		segment := &Segment{Type: TypeInteger, Endianness: "invalid"}
		err := ValidateSegment(segment)
		if err == nil {
			t.Error("Expected error for invalid endianness")
		}
		bsErr, ok := err.(*BitStringError)
		if !ok {
			t.Fatalf("Expected *BitStringError, got %T", err)
		}
		if bsErr.Code != CodeInvalidEndianness {
			t.Errorf("Expected code %s, got %s", CodeInvalidEndianness, bsErr.Code)
		}
	})

	// Test invalid float size
	t.Run("InvalidFloatSize", func(t *testing.T) {
		segment := &Segment{Type: TypeFloat, Size: 24, SizeSpecified: true}
		err := ValidateSegment(segment)
		if err == nil {
			t.Error("Expected error for invalid float size")
		}
		bsErr, ok := err.(*BitStringError)
		if !ok {
			t.Fatalf("Expected *BitStringError, got %T", err)
		}
		if bsErr.Code != CodeInvalidFloatSize {
			t.Errorf("Expected code %s, got %s", CodeInvalidFloatSize, bsErr.Code)
		}
	})

	// Test valid float sizes
	t.Run("ValidFloatSizes", func(t *testing.T) {
		sizes := []uint{16, 32, 64}
		for _, size := range sizes {
			segment := &Segment{Type: TypeFloat, Size: size, SizeSpecified: true}
			err := ValidateSegment(segment)
			if err != nil {
				t.Errorf("Expected no error for float size %d, got %v", size, err)
			}
		}
	})

	// Test UTF size specified error
	t.Run("UTFSizeSpecified", func(t *testing.T) {
		segment := &Segment{Type: TypeUTF8, Size: 8, SizeSpecified: true}
		err := ValidateSegment(segment)
		if err == nil {
			t.Error("Expected error for UTF with size specified")
		}
		bsErr, ok := err.(*BitStringError)
		if !ok {
			t.Fatalf("Expected *BitStringError, got %T", err)
		}
		if bsErr.Code != CodeUTFSizeSpecified {
			t.Errorf("Expected code %s, got %s", CodeUTFSizeSpecified, bsErr.Code)
		}
	})

	// Test UTF unit modified error
	t.Run("UTFUnitModified", func(t *testing.T) {
		segment := &Segment{Type: TypeUTF8, Unit: 8, UnitSpecified: true}
		err := ValidateSegment(segment)
		if err == nil {
			t.Error("Expected error for UTF with modified unit")
		}
		bsErr, ok := err.(*BitStringError)
		if !ok {
			t.Fatalf("Expected *BitStringError, got %T", err)
		}
		if bsErr.Code != CodeUTFUnitModified {
			t.Errorf("Expected code %s, got %s", CodeUTFUnitModified, bsErr.Code)
		}
	})

	// Test invalid Unicode code point
	t.Run("InvalidUnicodeCodePoint", func(t *testing.T) {
		segment := &Segment{Type: TypeUTF8, Value: int(0x110000)}
		err := ValidateSegment(segment)
		if err == nil {
			t.Error("Expected error for invalid Unicode code point")
		}
		bsErr, ok := err.(*BitStringError)
		if !ok {
			t.Fatalf("Expected *BitStringError, got %T", err)
		}
		if bsErr.Code != CodeInvalidUnicodeCodepoint {
			t.Errorf("Expected code %s, got %s", CodeInvalidUnicodeCodepoint, bsErr.Code)
		}
	})

	// Test valid Unicode code point
	t.Run("ValidUnicodeCodePoint", func(t *testing.T) {
		segment := &Segment{Type: TypeUTF8, Value: int(0x10FFFF)}
		err := ValidateSegment(segment)
		if err != nil {
			t.Errorf("Expected no error for valid Unicode code point, got %v", err)
		}
	})

	// Test invalid binary data
	t.Run("InvalidBinaryData", func(t *testing.T) {
		segment := &Segment{Type: TypeBinary, Value: "invalid", Size: 8, SizeSpecified: true}
		err := ValidateSegment(segment)
		if err == nil {
			t.Error("Expected error for invalid binary data")
		}
		bsErr, ok := err.(*BitStringError)
		if !ok {
			t.Fatalf("Expected *BitStringError, got %T", err)
		}
		if bsErr.Code != CodeInvalidBinaryData {
			t.Errorf("Expected code %s, got %s", CodeInvalidBinaryData, bsErr.Code)
		}
	})

	// Test valid binary data
	t.Run("ValidBinaryData", func(t *testing.T) {
		segment := &Segment{Type: TypeBinary, Value: []byte{0x01, 0x02}, Size: 16, SizeSpecified: true}
		err := ValidateSegment(segment)
		if err != nil {
			t.Errorf("Expected no error for valid binary data, got %v", err)
		}
	})

	// Test invalid bitstring data
	t.Run("InvalidBitstringData", func(t *testing.T) {
		segment := &Segment{Type: TypeBitstring, Value: "invalid", Size: 8, SizeSpecified: true}
		err := ValidateSegment(segment)
		if err == nil {
			t.Error("Expected error for invalid bitstring data")
		}
		bsErr, ok := err.(*BitStringError)
		if !ok {
			t.Fatalf("Expected *BitStringError, got %T", err)
		}
		if bsErr.Code != CodeInvalidBitstringData {
			t.Errorf("Expected code %s, got %s", CodeInvalidBitstringData, bsErr.Code)
		}
	})

	// Test valid bitstring data
	t.Run("ValidBitstringData", func(t *testing.T) {
		bs := NewBitStringFromBytes([]byte{0x01, 0x02})
		segment := &Segment{Type: TypeBitstring, Value: bs, Size: 16, SizeSpecified: true}
		err := ValidateSegment(segment)
		if err != nil {
			t.Errorf("Expected no error for valid bitstring data, got %v", err)
		}
	})

	// Test unsupported type
	t.Run("UnsupportedType", func(t *testing.T) {
		segment := &Segment{Type: "unsupported"}
		err := ValidateSegment(segment)
		if err == nil {
			t.Error("Expected error for unsupported type")
		}
		bsErr, ok := err.(*BitStringError)
		if !ok {
			t.Fatalf("Expected *BitStringError, got %T", err)
		}
		if bsErr.Code != CodeInvalidType {
			t.Errorf("Expected code %s, got %s", CodeInvalidType, bsErr.Code)
		}
	})

	// Test valid types
	t.Run("ValidTypes", func(t *testing.T) {
		validTypes := []string{TypeInteger, TypeFloat, TypeBinary, TypeBitstring, TypeUTF, TypeUTF8, TypeUTF16, TypeUTF32, TypeRestBinary, TypeRestBitstring}
		for _, validType := range validTypes {
			segment := &Segment{Type: validType}
			err := ValidateSegment(segment)
			if err != nil {
				t.Errorf("Expected no error for valid type %s, got %v", validType, err)
			}
		}
	})
}

// Additional tests to improve coverage

func TestBitString_NewBitStringFromBytes_Error(t *testing.T) {
	// Test with nil data to trigger error path
	bs := NewBitStringFromBytes(nil)
	if bs == nil {
		t.Fatal("Expected NewBitStringFromBytes() to return non-nil even with nil data")
	}
	// Should create an empty bitstring
	if bs.Length() != 0 {
		t.Errorf("Expected empty bitstring length 0, got %d", bs.Length())
	}
}

func TestBitString_NewBitStringFromBits_Error(t *testing.T) {
	// Test with nil data to trigger error path
	bs := NewBitStringFromBits(nil, 8)
	if bs != nil {
		t.Error("Expected NewBitStringFromBits() to return nil with nil data")
	}
}

func TestBitString_ToBytes_ZeroLength(t *testing.T) {
	bs := NewBitString()
	bytes := bs.ToBytes()
	if len(bytes) != 0 {
		t.Errorf("Expected empty bytes for zero-length bitstring, got %v", bytes)
	}
}

func TestBitString_Clone_ZeroLength(t *testing.T) {
	original := NewBitString()
	cloned := original.Clone()

	if cloned == nil {
		t.Fatal("Expected Clone() to return non-nil")
	}

	if cloned.Length() != 0 {
		t.Errorf("Expected clone to have zero length, got %d", cloned.Length())
	}

	if !cloned.IsEmpty() {
		t.Error("Expected clone to be empty")
	}
}

func TestNewSegment_NoOptions(t *testing.T) {
	// Test NewSegment without any options to cover default values path
	segment := NewSegment(42)

	if segment == nil {
		t.Fatal("Expected NewSegment() to return non-nil")
	}

	// Check default values
	if segment.Value != 42 {
		t.Errorf("Expected value 42, got %v", segment.Value)
	}

	if segment.Type != TypeInteger {
		t.Errorf("Expected default type %s, got %s", TypeInteger, segment.Type)
	}

	if segment.Size != 8 { // Default size for integer is DefaultSizeInteger = 8
		t.Errorf("Expected default size 8, got %d", segment.Size)
	}

	if segment.Signed != Unsigned { // Default signedness is Unsigned = false
		t.Errorf("Expected default signed %v, got %v", Unsigned, segment.Signed)
	}

	if segment.Endianness != EndiannessBig {
		t.Errorf("Expected default endianness %s, got %s", EndiannessBig, segment.Endianness)
	}

	if segment.Unit != 0 { // NewSegment starts with Unit = 0 to detect if unit was set
		t.Errorf("Expected default unit 0, got %d", segment.Unit)
	}
}

func TestGetDefaultSizeForType(t *testing.T) {
	testCases := []struct {
		typeName string
		expected uint
	}{
		{TypeInteger, 8},       // DefaultSizeInteger = 8
		{TypeFloat, 64},        // DefaultSizeFloat = 64
		{TypeBinary, 0},        // No default size
		{TypeBitstring, 0},     // No default size
		{TypeUTF8, 0},          // UTF types should not have default size
		{TypeUTF16, 0},         // UTF types should not have default size
		{TypeUTF32, 0},         // UTF types should not have default size
		{TypeUTF, 0},           // UTF types should not have default size
		{TypeRestBinary, 0},    // No default size
		{TypeRestBitstring, 0}, // No default size
		{"unknown", 0},         // Should return 0 for unknown types
	}

	for _, tc := range testCases {
		t.Run(tc.typeName, func(t *testing.T) {
			result := getDefaultSizeForType(tc.typeName)
			if result != tc.expected {
				t.Errorf("Expected default size %d for type %s, got %d", tc.expected, tc.typeName, result)
			}
		})
	}
}

func TestGetDefaultUnitForType(t *testing.T) {
	testCases := []struct {
		typeName string
		expected uint
	}{
		{TypeInteger, 1},       // DefaultUnitInteger = 1
		{TypeFloat, 1},         // DefaultUnitFloat = 1
		{TypeBinary, 8},        // DefaultUnitBinary = 8
		{TypeBitstring, 1},     // DefaultUnitInteger = 1
		{TypeUTF8, 1},          // DefaultUnitUTF = 1
		{TypeUTF16, 1},         // DefaultUnitUTF = 1
		{TypeUTF32, 1},         // DefaultUnitUTF = 1
		{TypeUTF, 1},           // DefaultUnitUTF = 1
		{TypeRestBinary, 1},    // DefaultUnitInteger = 1
		{TypeRestBitstring, 1}, // DefaultUnitInteger = 1
		{"unknown", 1},         // DefaultUnitInteger = 1 for unknown types
	}

	for _, tc := range testCases {
		t.Run(tc.typeName, func(t *testing.T) {
			result := getDefaultUnitForType(tc.typeName)
			if result != tc.expected {
				t.Errorf("Expected default unit %d for type %s, got %d", tc.expected, tc.typeName, result)
			}
		})
	}
}

// Additional tests to reach 95%+ coverage

func TestBitString_ToBytes_NonBinary(t *testing.T) {
	// Test ToBytes with non-binary bitstring (not multiple of 8 bits)
	bs := NewBitStringFromBits([]byte{0b10110000}, 4) // 4 bits

	bytes := bs.ToBytes()
	if len(bytes) != 1 {
		t.Errorf("Expected 1 byte for 4 bits, got %d", len(bytes))
	}
	// Should be 0b10110000 (the first 4 bits are preserved, rest are as-is)
	if bytes[0] != 0b10110000 {
		t.Errorf("Expected byte 0b10110000, got 0b%08b", bytes[0])
	}
}

func TestBitString_Clone_NonBinary(t *testing.T) {
	// Test Clone with non-binary bitstring
	original := NewBitStringFromBits([]byte{0b10110000, 0xFF}, 12) // 12 bits (1.5 bytes)

	cloned := original.Clone()

	if cloned == nil {
		t.Fatal("Expected Clone() to return non-nil")
	}

	if cloned.Length() != original.Length() {
		t.Errorf("Expected clone to have same length %d, got %d", original.Length(), cloned.Length())
	}

	if cloned.IsBinary() != original.IsBinary() {
		t.Errorf("Expected clone to have same IsBinary() value %v", original.IsBinary())
	}

	originalBytes := original.ToBytes()
	clonedBytes := cloned.ToBytes()

	if !reflect.DeepEqual(originalBytes, clonedBytes) {
		t.Errorf("Expected clone to have same bytes %v, got %v", originalBytes, clonedBytes)
	}
}

func TestNewSegment_UTFTypes(t *testing.T) {
	// Test NewSegment with UTF types to cover the path where SizeSpecified is set to false
	segment := NewSegment(65, WithType(TypeUTF8)) // 'A' character

	if segment == nil {
		t.Fatal("Expected NewSegment() to return non-nil")
	}

	if segment.Type != TypeUTF8 {
		t.Errorf("Expected type %s, got %s", TypeUTF8, segment.Type)
	}

	// For UTF types, SizeSpecified should be false
	if segment.SizeSpecified {
		t.Error("Expected SizeSpecified to be false for UTF types")
	}
}

func TestValidateSegment_EdgeCases(t *testing.T) {
	// Test case for binary segment with size 0 (should be allowed for dynamic sizing)
	t.Run("BinarySizeZero", func(t *testing.T) {
		segment := &Segment{Type: TypeBinary, Size: 0, SizeSpecified: true}
		err := ValidateSegment(segment)
		if err != nil {
			t.Errorf("Expected no error for binary with size 0, got %v", err)
		}
	})

	// Test case for bitstring segment with size 0 (should be allowed only if not SizeSpecified)
	t.Run("BitstringSizeZero", func(t *testing.T) {
		segment := &Segment{Type: TypeBitstring, Size: 0, SizeSpecified: false}
		err := ValidateSegment(segment)
		if err != nil {
			t.Errorf("Expected no error for bitstring with size 0 and SizeSpecified=false, got %v", err)
		}
	})

	// Test case for UTF with unit explicitly set to default value (should be allowed)
	t.Run("UTFDefaultUnit", func(t *testing.T) {
		segment := &Segment{Type: TypeUTF8, Unit: 1, UnitSpecified: true}
		err := ValidateSegment(segment)
		if err != nil {
			t.Errorf("Expected no error for UTF with default unit, got %v", err)
		}
	})
}
