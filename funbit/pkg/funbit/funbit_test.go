package funbit

import (
	"testing"
)

func TestPublicAPIBasicConstruction(t *testing.T) {
	// Test basic bitstring construction using public API
	builder := NewBuilder()
	AddInteger(builder, 42)
	AddInteger(builder, 17, WithSize(8))
	AddBinary(builder, []byte("hello"))

	bs, err := Build(builder)
	if err != nil {
		t.Fatalf("Failed to build bitstring: %v", err)
	}

	if bs.Length() == 0 {
		t.Error("Expected non-empty bitstring")
	}

	if !bs.IsBinary() {
		t.Error("Expected binary-aligned bitstring")
	}

	data := bs.ToBytes()
	if len(data) == 0 {
		t.Error("Expected non-empty byte data")
	}
}

func TestPublicAPIBasicMatching(t *testing.T) {
	// Create a test bitstring
	builder := NewBuilder()
	AddInteger(builder, 42)
	AddInteger(builder, 17, WithSize(8))
	AddBinary(builder, []byte("hello"))

	bs, err := Build(builder)
	if err != nil {
		t.Fatalf("Failed to build bitstring: %v", err)
	}

	// Test pattern matching using public API
	var a, b int
	var c []byte

	matcher := NewMatcher()
	Integer(matcher, &a)
	Integer(matcher, &b, WithSize(8))
	Binary(matcher, &c)

	results, err := Match(matcher, bs)
	if err != nil {
		t.Fatalf("Failed to match pattern: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}

	if a != 42 {
		t.Errorf("Expected a=42, got %d", a)
	}

	if b != 17 {
		t.Errorf("Expected b=17, got %d", b)
	}

	if string(c) != "hello" {
		t.Errorf("Expected c='hello', got '%s'", string(c))
	}
}

func TestPublicAPIUtilityFunctions(t *testing.T) {
	// Test bit manipulation utilities
	data := []byte{0xFF, 0x00, 0xF0}

	// Test ExtractBits
	extracted, err := ExtractBits(data, 4, 8)
	if err != nil {
		t.Fatalf("Failed to extract bits: %v", err)
	}
	if len(extracted) != 1 { // 8 bits = 1 byte
		t.Errorf("Expected 1 byte for extracted data, got %d", len(extracted))
	}

	// Test CountBits
	count := CountBits(data)
	expectedCount := uint(12) // 0xFF has 8 bits, 0xF0 has 4 bits
	if count != expectedCount {
		t.Errorf("Expected %d set bits, got %d", expectedCount, count)
	}

	// Test GetBitValue
	value, err := GetBitValue(data, 0)
	if err != nil {
		t.Fatalf("Failed to get bit value: %v", err)
	}
	if !value { // First bit of 0xFF should be 1 (MSB first)
		t.Error("Expected first bit to be true (1)")
	}

	// Test conversion functions
	bits, err := IntToBits(42, 16, false)
	if err != nil {
		t.Fatalf("Failed to convert int to bits: %v", err)
	}
	if len(bits) != 2 { // 16 bits = 2 bytes
		t.Errorf("Expected 2 bytes for 16-bit int, got %d", len(bits))
	}

	converted, err := BitsToInt(bits, false)
	if err != nil {
		t.Fatalf("Failed to convert bits to int: %v", err)
	}
	if converted != 42 {
		t.Errorf("Expected converted value 42, got %d", converted)
	}
}

func TestPublicAPIUTFFunctions(t *testing.T) {
	// Test UTF encoding/decoding
	text := "Hello, 世界!"

	// Test UTF-8 encoding
	encoded, err := EncodeUTF8(text)
	if err != nil {
		t.Fatalf("Failed to encode UTF-8: %v", err)
	}

	// Test UTF-8 decoding
	decoded, err := DecodeUTF8(encoded)
	if err != nil {
		t.Fatalf("Failed to decode UTF-8: %v", err)
	}

	if decoded != text {
		t.Errorf("Expected decoded text '%s', got '%s'", text, decoded)
	}

	// Test UTF-16 encoding/decoding
	encoded16, err := EncodeUTF16(text, "big")
	if err != nil {
		t.Fatalf("Failed to encode UTF-16: %v", err)
	}

	decoded16, err := DecodeUTF16(encoded16, "big")
	if err != nil {
		t.Fatalf("Failed to decode UTF-16: %v", err)
	}

	if decoded16 != text {
		t.Errorf("Expected decoded UTF-16 text '%s', got '%s'", text, decoded16)
	}

	// Test Unicode validation
	if !IsValidUnicodeCodePoint(0x1F600) { // Grinning face emoji
		t.Error("Expected 0x1F600 to be valid Unicode code point")
	}

	if IsValidUnicodeCodePoint(0xD800) { // Invalid surrogate
		t.Error("Expected 0xD800 to be invalid Unicode code point")
	}
}

func TestPublicAPIEndianness(t *testing.T) {
	// Test endianness functions
	native := GetNativeEndianness()
	if native != "big" && native != "little" {
		t.Errorf("Expected native endianness to be 'big' or 'little', got '%s'", native)
	}

	data := []byte{0x12, 0x34, 0x56, 0x78}

	// Test endianness conversion
	converted, err := ConvertEndianness(data, "big", "little", 32)
	if err != nil {
		t.Fatalf("Failed to convert endianness: %v", err)
	}

	expected := []byte{0x78, 0x56, 0x34, 0x12}
	if len(converted) != len(expected) {
		t.Errorf("Expected converted data length %d, got %d", len(expected), len(converted))
	}

	for i := range expected {
		if converted[i] != expected[i] {
			t.Errorf("At position %d: expected 0x%02X, got 0x%02X", i, expected[i], converted[i])
		}
	}
}

func TestPublicAPIFormatting(t *testing.T) {
	// Create a test bitstring
	builder := NewBuilder()
	AddInteger(builder, 0x12)
	AddInteger(builder, 0x34)
	AddInteger(builder, 0x56)

	bs, err := Build(builder)
	if err != nil {
		t.Fatalf("Failed to build bitstring: %v", err)
	}

	// Test ToHexDump
	hexDump := ToHexDump(bs)
	if hexDump == "" {
		t.Error("Expected non-empty hex dump")
	}

	// Test ToBinaryString
	binaryStr := ToBinaryString(bs)
	if binaryStr == "" {
		t.Error("Expected non-empty binary string")
	}

	// Test ToErlangFormat
	erlangFmt := ToErlangFormat(bs)
	if erlangFmt == "" {
		t.Error("Expected non-empty Erlang format")
	}

	// Should be in format <<18,52,86>> for byte-aligned data
	if erlangFmt[0] != '<' || erlangFmt[len(erlangFmt)-1] != '>' {
		t.Errorf("Expected Erlang format to be enclosed in <<>>, got '%s'", erlangFmt)
	}
}

func TestPublicAPIErrorHandling(t *testing.T) {
	// Test error creation
	err := NewBitStringError(ErrInvalidSize, "size must be positive")
	if err == nil {
		t.Error("Expected non-nil error")
	}

	if err.Code != ErrInvalidSize {
		t.Errorf("Expected error code '%s', got '%s'", ErrInvalidSize, err.Code)
	}

	// Test error with context
	errWithContext := NewBitStringErrorWithContext(ErrInvalidSize, "size too large", map[string]interface{}{"size": 100})
	if errWithContext == nil {
		t.Error("Expected non-nil error with context")
	}

	if errWithContext.Code != ErrInvalidSize {
		t.Errorf("Expected error code '%s', got '%s'", ErrInvalidSize, errWithContext.Code)
	}

	// Test validation functions - size 0 is now valid in Erlang spec
	validationErr := ValidateSize(0, 1)
	if validationErr != nil {
		t.Errorf("Unexpected validation error for size 0: %v", validationErr)
	}

	unicodeErr := ValidateUnicodeCodePoint(0x110000) // Beyond Unicode range
	if unicodeErr == nil {
		t.Error("Expected validation error for invalid Unicode code point")
	}
}

func TestPublicAPIConstants(t *testing.T) {
	// Test that all constants are properly defined

	// Segment types
	if TypeInteger != "integer" {
		t.Errorf("Expected TypeInteger to be 'integer', got '%s'", TypeInteger)
	}
	if TypeFloat != "float" {
		t.Errorf("Expected TypeFloat to be 'float', got '%s'", TypeFloat)
	}

	// Endianness
	if EndiannessBig != "big" {
		t.Errorf("Expected EndiannessBig to be 'big', got '%s'", EndiannessBig)
	}
	if EndiannessLittle != "little" {
		t.Errorf("Expected EndiannessLittle to be 'little', got '%s'", EndiannessLittle)
	}

	// Error codes
	if ErrOverflow != "OVERFLOW" {
		t.Errorf("Expected ErrOverflow to be 'OVERFLOW', got '%s'", ErrOverflow)
	}
	if ErrInvalidSize != "INVALID_SIZE" {
		t.Errorf("Expected ErrInvalidSize to be 'INVALID_SIZE', got '%s'", ErrInvalidSize)
	}

	// Default values
	if DefaultSizeInteger != 8 {
		t.Errorf("Expected DefaultSizeInteger to be 8, got %d", DefaultSizeInteger)
	}
	if DefaultSizeFloat != 64 {
		t.Errorf("Expected DefaultSizeFloat to be 64, got %d", DefaultSizeFloat)
	}
}

func TestPublicAPIFactoryFunctions(t *testing.T) {
	// Test BitString factory functions
	empty := NewBitString()
	if empty == nil || empty.Length() != 0 {
		t.Error("Expected empty bitstring with length 0")
	}

	fromBytes := NewBitStringFromBytes([]byte{1, 2, 3})
	if fromBytes == nil || fromBytes.Length() != 24 {
		t.Error("Expected bitstring with length 24 (3 bytes)")
	}

	fromBits := NewBitStringFromBits([]byte{0xFF}, 4)
	if fromBits == nil || fromBits.Length() != 4 {
		t.Error("Expected bitstring with length 4 bits")
	}

	// Test Builder and Matcher factory functions
	builder := NewBuilder()
	if builder == nil {
		t.Error("Expected non-nil builder")
	}

	matcher := NewMatcher()
	if matcher == nil {
		t.Error("Expected non-nil matcher")
	}
}

func TestPublicAPISegmentOptions(t *testing.T) {
	// Test segment options
	segment := NewSegment(42,
		WithSize(16),
		WithSigned(true),
		WithEndianness("little"),
		WithUnit(2),
	)

	if segment.Size != 16 {
		t.Errorf("Expected segment size 16, got %d", segment.Size)
	}

	if !segment.Signed {
		t.Error("Expected segment to be signed")
	}

	if segment.Endianness != "little" {
		t.Errorf("Expected endianness 'little', got '%s'", segment.Endianness)
	}

	if segment.Unit != 2 {
		t.Errorf("Expected unit 2, got %d", segment.Unit)
	}

	// Test segment validation
	err := ValidateSegment(segment)
	if err != nil {
		t.Errorf("Expected segment to be valid, got error: %v", err)
	}

	// Test valid segment with size 0 (allowed in Erlang spec)
	validSegment := NewSegment(42, WithSize(0)) // Size 0 is now valid
	err = ValidateSegment(validSegment)
	if err != nil {
		t.Errorf("Unexpected validation error for segment with size 0: %v", err)
	}
}
