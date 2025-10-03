package acceptancetests

import (
	"errors"
	"testing"

	bitstringpkg "github.com/funvibe/funbit/internal/bitstring"
	"github.com/funvibe/funbit/internal/builder"
	"github.com/funvibe/funbit/internal/matcher"
)

// TestPatternMatchingSizeMismatch tests pattern matching with incorrect sizes
func TestPatternMatchingSizeMismatch(t *testing.T) {
	// Create test bitstring with 3 bytes
	data := bitstringpkg.NewBitStringFromBytes([]byte{1, 2, 3})

	var a, b, c, d int

	// Test matching with too many segments (should fail)
	_, err := matcher.NewMatcher().
		Integer(&a, bitstringpkg.WithSize(8)).
		Integer(&b, bitstringpkg.WithSize(8)).
		Integer(&c, bitstringpkg.WithSize(8)).
		Integer(&d, bitstringpkg.WithSize(8)). // This should fail - only 3 bytes available
		Match(data)

	if err == nil {
		t.Fatal("Expected error for too many segments, got nil")
	}

	// Check if it's the right type of error
	var bitstringErr *bitstringpkg.BitStringError
	if errors.As(err, &bitstringErr) {
		if bitstringErr.Code != "INSUFFICIENT_BITS" {
			t.Errorf("Expected INSUFFICIENT_BITS error, got %s", bitstringErr.Code)
		}
	} else {
		t.Errorf("Expected BitStringError, got %T", err)
	}

	// Test matching with too few segments (should succeed but leave remaining data)
	_, err = matcher.NewMatcher().
		Integer(&a, bitstringpkg.WithSize(8)).
		Integer(&b, bitstringpkg.WithSize(8)). // Only matching 2 bytes, leaving 1 byte unmatched
		Match(data)

	if err != nil {
		t.Fatalf("Expected no error for too few segments, got %v", err)
	}

	if a != 1 || b != 2 {
		t.Errorf("Expected a=1, b=2, got a=%d, b=%d", a, b)
	}

	// Test correct matching (should succeed)
	_, err = matcher.NewMatcher().
		Integer(&a, bitstringpkg.WithSize(8)).
		Integer(&b, bitstringpkg.WithSize(8)).
		Integer(&c, bitstringpkg.WithSize(8)).
		Match(data)

	if err != nil {
		t.Fatalf("Expected no error for correct matching, got %v", err)
	}

	if a != 1 || b != 2 || c != 3 {
		t.Errorf("Expected a=1, b=2, c=3, got a=%d, b=%d, c=%d", a, b, c)
	}
}

// TestBuilderOverflowError tests truncation behavior according to Erlang spec
func TestBuilderOverflowError(t *testing.T) {
	// According to Erlang spec, 256 in 8 bits should be truncated to 0
	bs, err := builder.NewBuilder().
		AddInteger(256, bitstringpkg.WithSize(8)). // 256 truncated to 8 bits = 0
		Build()

	// According to Erlang spec, should succeed with truncation
	if err != nil {
		t.Fatalf("Expected no error according to Erlang spec, got %v", err)
	}

	if bs == nil {
		t.Fatal("Expected valid bitstring, got nil")
	}

	// Verify the value was truncated to 0
	data := bs.ToBytes()
	if len(data) != 1 || data[0] != 0 {
		t.Errorf("Expected [0] (256 truncated), got %v", data)
	}

	// Test negative truncation for signed integers (-129 should be truncated)
	bs2, err := builder.NewBuilder().
		AddInteger(-129, bitstringpkg.WithSize(8), bitstringpkg.WithSigned(true)). // -129 truncated
		Build()

	// According to Erlang spec, should succeed with truncation
	if err != nil {
		t.Fatalf("Expected no error for signed truncation according to Erlang spec, got %v", err)
	}

	if bs2 == nil {
		t.Fatal("Expected valid bitstring for signed truncation, got nil")
	}
}

// TestEmptyBitstringMatching tests matching with empty bitstrings
func TestEmptyBitstringMatching(t *testing.T) {
	empty := bitstringpkg.NewBitString()

	// Test matching empty pattern
	_, err := matcher.NewMatcher().
		Match(empty)

	if err != nil {
		t.Fatalf("Expected no error for empty pattern matching, got %v", err)
	}

	// Test matching non-empty pattern with empty bitstring (should fail)
	var value int
	_, err = matcher.NewMatcher().
		Integer(&value, bitstringpkg.WithSize(8)).
		Match(empty)

	if err == nil {
		t.Fatal("Expected error for matching non-empty pattern with empty bitstring, got nil")
	}

	var bitstringErr *bitstringpkg.BitStringError
	if errors.As(err, &bitstringErr) {
		if bitstringErr.Code != "INSUFFICIENT_BITS" {
			t.Errorf("Expected INSUFFICIENT_BITS error, got %s", bitstringErr.Code)
		}
	}
}

// TestUnalignedBitstringMatching tests matching with unaligned bitstrings
func TestUnalignedBitstringMatching(t *testing.T) {
	// Create 7-bit unaligned bitstring
	data := []byte{0b10110000} // Only first 7 bits used
	bs := bitstringpkg.NewBitStringFromBits(data, 7)

	if bs.Length() != 7 {
		t.Errorf("Expected 7-bit bitstring, got %d bits", bs.Length())
	}

	// Test matching 7 bits (should succeed)
	var value int
	_, err := matcher.NewMatcher().
		Integer(&value, bitstringpkg.WithSize(7)).
		Match(bs)

	if err != nil {
		t.Fatalf("Expected no error for matching 7 bits, got %v", err)
	}

	expected := 0b1011000 // First 7 bits of 0b10110000
	if value != expected {
		t.Errorf("Expected value %d, got %d", expected, value)
	}

	// Test matching 8 bits with 7-bit data (should fail)
	_, err = matcher.NewMatcher().
		Integer(&value, bitstringpkg.WithSize(8)).
		Match(bs)

	if err == nil {
		t.Fatal("Expected error for matching 8 bits with 7-bit data, got nil")
	}

	var bitstringErr *bitstringpkg.BitStringError
	if errors.As(err, &bitstringErr) {
		if bitstringErr.Code != "INSUFFICIENT_BITS" {
			t.Errorf("Expected INSUFFICIENT_BITS error, got %s", bitstringErr.Code)
		}
	}
}

// TestInvalidSegmentValidation tests validation of invalid segments
func TestInvalidSegmentValidation(t *testing.T) {
	// Test segment with invalid size
	_, err := builder.NewBuilder().
		AddSegment(bitstringpkg.Segment{
			Value:         42,
			Size:          8, // Valid size
			Type:          "integer",
			Unit:          300, // Invalid unit
			UnitSpecified: true,
		}).
		Build()

	if err == nil {
		t.Fatal("Expected error for invalid unit, got nil")
	}

	var bitstringErr *bitstringpkg.BitStringError
	if errors.As(err, &bitstringErr) {
		if bitstringErr.Code != "INVALID_UNIT" {
			t.Errorf("Expected INVALID_UNIT error, got %s", bitstringErr.Code)
		}
	}

	// Test segment with invalid type
	_, err = builder.NewBuilder().
		AddSegment(bitstringpkg.Segment{
			Value: 42,
			Size:  8,
			Type:  "invalid_type", // Invalid type
		}).
		Build()

	if err == nil {
		t.Fatal("Expected error for invalid type, got nil")
	}

	var bitstringErr2 *bitstringpkg.BitStringError
	if errors.As(err, &bitstringErr2) {
		if bitstringErr2.Code != "INVALID_TYPE" {
			t.Errorf("Expected INVALID_TYPE error, got %s", bitstringErr2.Code)
		}
	}

	// Test segment with invalid endianness
	_, err = builder.NewBuilder().
		AddSegment(bitstringpkg.Segment{
			Value:      42,
			Size:       16,
			Type:       "integer",
			Endianness: "invalid_endian", // Invalid endianness
		}).
		Build()

	if err == nil {
		t.Fatal("Expected error for invalid endianness, got nil")
	}

	var bitstringErr3 *bitstringpkg.BitStringError
	if errors.As(err, &bitstringErr3) {
		if bitstringErr3.Code != "INVALID_ENDIANNESS" {
			t.Errorf("Expected INVALID_ENDIANNESS error, got %s", bitstringErr3.Code)
		}
	}
}

// TestBinarySegmentErrors tests errors specific to binary segments
func TestBinarySegmentErrors(t *testing.T) {
	// Test binary segment with explicit zero size (should fail)
	_, err := builder.NewBuilder().
		AddBinary([]byte("test"), bitstringpkg.WithSize(0)).
		Build()

	if err == nil {
		t.Fatal("Expected error for binary segment with zero size, got nil")
	}

	var bitstringErr *bitstringpkg.BitStringError
	if errors.As(err, &bitstringErr) {
		if bitstringErr.Code != "INVALID_SIZE" {
			t.Errorf("Expected INVALID_SIZE error, got %s", bitstringErr.Code)
		}
	}

	// Test binary segment with size larger than data
	_, err = builder.NewBuilder().
		AddBinary([]byte("test"), bitstringpkg.WithSize(10)). // Only 4 bytes available
		Build()

	if err == nil {
		t.Fatal("Expected error for binary size larger than data, got nil")
	}

	var bitstringErr2 *bitstringpkg.BitStringError
	if errors.As(err, &bitstringErr2) {
		if bitstringErr2.Code != "BINARY_SIZE_MISMATCH" {
			t.Errorf("Expected BINARY_SIZE_MISMATCH error, got %s", bitstringErr2.Code)
		}
	}

	// Test binary segment with non-byte data
	_, err = builder.NewBuilder().
		AddSegment(bitstringpkg.Segment{
			Value: 42, // Not []byte
			Size:  4,
			Type:  "binary",
		}).
		Build()

	if err == nil {
		t.Fatal("Expected error for binary segment with non-byte data, got nil")
	}

	var bitstringErr3 *bitstringpkg.BitStringError
	if errors.As(err, &bitstringErr3) {
		if bitstringErr3.Code != "INVALID_BINARY_DATA" {
			t.Errorf("Expected INVALID_BINARY_DATA error, got %s", bitstringErr3.Code)
		}
	}
}

// TestUTFErrors tests UTF-specific errors
func TestUTFErrors(t *testing.T) {
	// Test UTF segment with size specified (should fail)
	_, err := builder.NewBuilder().
		AddSegment(bitstringpkg.Segment{
			Value:         65, // UTF codepoint for 'A'
			Size:          8,  // Size should not be specified for UTF
			SizeSpecified: true,
			Type:          "utf8",
		}).
		Build()

	if err == nil {
		t.Fatal("Expected error for UTF segment with size, got nil")
	}

	var bitstringErr *bitstringpkg.BitStringError
	if errors.As(err, &bitstringErr) {
		if bitstringErr.Code != "UTF_SIZE_SPECIFIED" {
			t.Errorf("Expected UTF_SIZE_SPECIFIED error, got %s", bitstringErr.Code)
		}
	}

	// Test UTF with invalid code point - create segment with proper UTF type without size
	segment := bitstringpkg.NewSegment(0x110000) // Beyond Unicode max (0x10FFFF)
	segment.Type = "utf8"
	// Don't set Size - UTF should not have size specified

	_, err = builder.NewBuilder().
		AddSegment(*segment).
		Build()

	if err == nil {
		t.Fatal("Expected error for invalid Unicode code point, got nil")
	}

	var bitstringErr2 *bitstringpkg.BitStringError
	if errors.As(err, &bitstringErr2) {
		if bitstringErr2.Code != "INVALID_UNICODE_CODEPOINT" {
			t.Errorf("Expected INVALID_UNICODE_CODEPOINT error, got %s", bitstringErr2.Code)
		}
	}
}

// TestMatcherPatternMismatch tests pattern mismatch scenarios
func TestMatcherPatternMismatch(t *testing.T) {
	// Create bitstring with specific pattern
	bs, err := builder.NewBuilder().
		AddInteger(0x42, bitstringpkg.WithSize(8)).
		AddInteger(0x13, bitstringpkg.WithSize(8)).
		Build()

	if err != nil {
		t.Fatalf("Failed to create test bitstring: %v", err)
	}

	// Test mismatch with different values
	var a, b int
	_, err = matcher.NewMatcher().
		Integer(&a, bitstringpkg.WithSize(8)).
		Integer(&b, bitstringpkg.WithSize(8)).
		Match(bs)

	if err != nil {
		t.Fatalf("Expected no error for correct pattern, got %v", err)
	}

	if a != 0x42 || b != 0x13 {
		t.Errorf("Expected a=0x42, b=0x13, got a=0x%02x, b=0x%02x", a, b)
	}

	// Test with different type mismatch - try to bind integer value to float variable
	// Create bitstring with 32 bits of data
	bs32, err := builder.NewBuilder().
		AddInteger(0x12345678, bitstringpkg.WithSize(32)).
		Build()

	if err != nil {
		t.Fatalf("Failed to create 32-bit test bitstring: %v", err)
	}

	var c float64
	_, err = matcher.NewMatcher().
		Integer(&c, bitstringpkg.WithSize(16)). // Try to bind integer result to float variable
		Match(bs32)

	if err == nil {
		t.Fatal("Expected error for type mismatch, got nil")
	}

	var bitstringErr *bitstringpkg.BitStringError
	if errors.As(err, &bitstringErr) {
		if bitstringErr.Code != "TYPE_MISMATCH" {
			t.Errorf("Expected TYPE_MISMATCH error, got %s", bitstringErr.Code)
		}
	}
}
