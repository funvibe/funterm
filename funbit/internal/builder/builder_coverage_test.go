package builder

import (
	"testing"

	"github.com/funvibe/funbit/internal/bitstring"
)

// TestBuilder_validateBitstringValue_MissingCases tests missing cases in validateBitstringValue
func TestBuilder_validateBitstringValue_MissingCases(t *testing.T) {
	t.Run("Validate bitstring with non-bitstring pointer", func(t *testing.T) {
		segment := &bitstring.Segment{
			Value:         "not a bitstring",
			Type:          bitstring.TypeBitstring,
			Size:          8,
			SizeSpecified: true,
		}

		_, err := validateBitstringValue(segment)
		if err == nil {
			t.Error("Expected error for non-bitstring value")
		}

		if bitStringErr, ok := err.(*bitstring.BitStringError); ok {
			if bitStringErr.Code != bitstring.CodeTypeMismatch {
				t.Errorf("Expected error code %s, got %s", bitstring.CodeTypeMismatch, bitStringErr.Code)
			}
		}
	})

	t.Run("Validate bitstring with nil pointer", func(t *testing.T) {
		var bs *bitstring.BitString = nil
		segment := &bitstring.Segment{
			Value:         bs,
			Type:          bitstring.TypeBitstring,
			Size:          8,
			SizeSpecified: true,
		}

		_, err := validateBitstringValue(segment)
		if err == nil {
			t.Error("Expected error for nil bitstring pointer")
		}

		if bitStringErr, ok := err.(*bitstring.BitStringError); ok {
			if bitStringErr.Code != bitstring.CodeInvalidSegment {
				t.Errorf("Expected error code %s, got %s", bitstring.CodeInvalidSegment, bitStringErr.Code)
			}
		}
	})

	t.Run("Validate bitstring with integer value", func(t *testing.T) {
		segment := &bitstring.Segment{
			Value:         42,
			Type:          bitstring.TypeBitstring,
			Size:          8,
			SizeSpecified: true,
		}

		_, err := validateBitstringValue(segment)
		if err == nil {
			t.Error("Expected error for integer value")
		}

		if bitStringErr, ok := err.(*bitstring.BitStringError); ok {
			if bitStringErr.Code != bitstring.CodeTypeMismatch {
				t.Errorf("Expected error code %s, got %s", bitstring.CodeTypeMismatch, bitStringErr.Code)
			}
		}
	})

	t.Run("Validate bitstring with slice value", func(t *testing.T) {
		segment := &bitstring.Segment{
			Value:         []byte{0xAB},
			Type:          bitstring.TypeBitstring,
			Size:          8,
			SizeSpecified: true,
		}

		_, err := validateBitstringValue(segment)
		if err == nil {
			t.Error("Expected error for slice value")
		}

		if bitStringErr, ok := err.(*bitstring.BitStringError); ok {
			if bitStringErr.Code != bitstring.CodeTypeMismatch {
				t.Errorf("Expected error code %s, got %s", bitstring.CodeTypeMismatch, bitStringErr.Code)
			}
		}
	})
}

// TestBuilder_writeBitstringBits_MissingBoundaryConditions tests boundary conditions in writeBitstringBits
func TestBuilder_writeBitstringBits_MissingBoundaryConditions(t *testing.T) {
	t.Run("Write bitstring bits with size exactly matching bitstring length", func(t *testing.T) {
		w := newBitWriter()
		bs := bitstring.NewBitStringFromBits([]byte{0xAB, 0xCD}, 16)

		err := writeBitstringBits(w, bs, 16) // Exact match
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		data, totalBits := w.final()
		if totalBits != 16 {
			t.Errorf("Expected totalBits 16, got %d", totalBits)
		}

		if len(data) != 2 || data[0] != 0xAB || data[1] != 0xCD {
			t.Errorf("Expected bytes [0xAB, 0xCD], got %v", data)
		}
	})

	t.Run("Write bitstring bits with size one less than bitstring length", func(t *testing.T) {
		w := newBitWriter()
		bs := bitstring.NewBitStringFromBits([]byte{0xAB, 0xCD}, 16)

		err := writeBitstringBits(w, bs, 15) // One less than available
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		data, totalBits := w.final()
		if totalBits != 15 {
			t.Errorf("Expected totalBits 15, got %d", totalBits)
		}

		// Should have first 15 bits of 0xABCD
		if len(data) != 2 {
			t.Errorf("Expected 2 bytes, got %d", len(data))
		}
	})

	t.Run("Write bitstring bits with single byte boundary", func(t *testing.T) {
		w := newBitWriter()
		bs := bitstring.NewBitStringFromBits([]byte{0xAB, 0xCD}, 16)

		err := writeBitstringBits(w, bs, 8) // Exactly one byte
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

	t.Run("Write bitstring bits with size crossing byte boundary", func(t *testing.T) {
		w := newBitWriter()
		bs := bitstring.NewBitStringFromBits([]byte{0xAB, 0xCD}, 16)

		err := writeBitstringBits(w, bs, 12) // Crosses byte boundary
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		data, totalBits := w.final()
		if totalBits != 12 {
			t.Errorf("Expected totalBits 12, got %d", totalBits)
		}

		if len(data) != 2 {
			t.Errorf("Expected 2 bytes, got %d", len(data))
		}
	})
}

// TestBuilder_encodeSegment_MissingErrorPaths tests error paths in encodeSegment that may be missing coverage
func TestBuilder_encodeSegment_MissingErrorPaths(t *testing.T) {
	t.Run("Encode segment with nil value", func(t *testing.T) {
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

	t.Run("Encode segment with invalid UTF type", func(t *testing.T) {
		w := newBitWriter()
		segment := &bitstring.Segment{
			Value:      65,
			Type:       "utf64", // Invalid UTF type
			Endianness: "big",
		}

		err := encodeSegment(w, segment)
		if err == nil {
			t.Error("Expected error for invalid UTF type")
		}

		if err.Error() != "unsupported segment type: utf64" {
			t.Errorf("Expected 'unsupported segment type: utf64', got %v", err)
		}
	})

	t.Run("Encode segment with float type and invalid size", func(t *testing.T) {
		w := newBitWriter()
		segment := &bitstring.Segment{
			Value:         float32(3.14),
			Type:          bitstring.TypeFloat,
			Size:          24, // Invalid float size (not 16, 32, or 64)
			SizeSpecified: true,
		}

		err := encodeSegment(w, segment)
		if err == nil {
			t.Error("Expected error for invalid float size")
		}

		if bitStringErr, ok := err.(*bitstring.BitStringError); ok {
			if bitStringErr.Code != bitstring.CodeInvalidFloatSize {
				t.Errorf("Expected error code %s, got %s", bitstring.CodeInvalidFloatSize, bitStringErr.Code)
			}
		}
	})

	t.Run("Encode segment with binary type and size not specified", func(t *testing.T) {
		w := newBitWriter()
		segment := &bitstring.Segment{
			Value:         []byte{0xAB},
			Type:          bitstring.TypeBinary,
			SizeSpecified: false, // Size not specified for binary
		}

		err := encodeSegment(w, segment)
		if err == nil {
			t.Error("Expected error for binary with size not specified")
		}

		if bitStringErr, ok := err.(*bitstring.BitStringError); ok {
			if bitStringErr.Code != bitstring.CodeBinarySizeRequired {
				t.Errorf("Expected error code %s, got %s", bitstring.CodeBinarySizeRequired, bitStringErr.Code)
			}
		}
	})
}

// TestBuilder_encodeInteger_MissingBitstringPaths tests bitstring type paths in encodeInteger
func TestBuilder_encodeInteger_MissingBitstringPaths(t *testing.T) {
	t.Run("Encode integer with bitstring type and exact size match", func(t *testing.T) {
		w := newBitWriter()
		segment := &bitstring.Segment{
			Value:         int64(0xAB),
			Type:          bitstring.TypeBitstring,
			Size:          8, // Exact match for integer value
			SizeSpecified: true,
		}

		err := encodeInteger(w, segment)
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

	t.Run("Encode integer with bitstring type and slice value", func(t *testing.T) {
		w := newBitWriter()
		segment := &bitstring.Segment{
			Value:         []byte{0xAB, 0xCD},
			Type:          bitstring.TypeBitstring,
			Size:          16, // Exact match for slice length
			SizeSpecified: true,
		}

		err := encodeInteger(w, segment)
		if err == nil {
			t.Error("Expected error for unsupported integer type, got nil")
		} else if err.Error() != "unsupported integer type for bitstring value: []uint8" {
			t.Errorf("Expected 'unsupported integer type for bitstring value: []uint8', got %v", err)
		}

		// Verify that no data was written due to the error
		data, totalBits := w.final()
		if totalBits != 0 {
			t.Errorf("Expected totalBits 0 due to error, got %d", totalBits)
		}

		if len(data) != 0 {
			t.Errorf("Expected 0 bytes due to error, got %d", len(data))
		}
	})

	t.Run("Encode integer with bitstring type and insufficient slice data", func(t *testing.T) {
		w := newBitWriter()
		segment := &bitstring.Segment{
			Value:         []byte{0xAB}, // Only 1 byte (8 bits)
			Type:          bitstring.TypeBitstring,
			Size:          16, // Requesting 16 bits
			SizeSpecified: true,
		}

		err := encodeInteger(w, segment)
		if err == nil {
			t.Error("Expected error for insufficient bits")
		}

		if bitStringErr, ok := err.(*bitstring.BitStringError); ok {
			if bitStringErr.Code != bitstring.CodeInsufficientBits {
				t.Errorf("Expected error code %s, got %s", bitstring.CodeInsufficientBits, bitStringErr.Code)
			}
		}
	})

	t.Run("Encode integer with bitstring type and non-byte slice", func(t *testing.T) {
		w := newBitWriter()
		// Create a slice that's not []byte to test different reflection path
		intSlice := []int{0xAB, 0xCD}
		segment := &bitstring.Segment{
			Value:         intSlice,
			Type:          bitstring.TypeBitstring,
			Size:          16,
			SizeSpecified: true,
		}

		err := encodeInteger(w, segment)
		// This should fail because it's not a []byte slice
		if err == nil {
			t.Error("Expected error for non-byte slice")
		}
		t.Logf("Expected error for non-byte slice: %v", err)
	})
}

// TestBuilder_encodeBinary_MissingSizeValidation tests size validation edge cases in encodeBinary
func TestBuilder_encodeBinary_MissingSizeValidation(t *testing.T) {
	t.Run("Encode binary with size specified as zero", func(t *testing.T) {
		w := newBitWriter()
		segment := &bitstring.Segment{
			Value:         []byte{0xAB},
			Type:          bitstring.TypeBinary,
			Size:          0, // Zero size
			SizeSpecified: true,
		}

		err := encodeBinary(w, segment)
		if err == nil {
			t.Error("Expected error for zero size")
		}

		if bitStringErr, ok := err.(*bitstring.BitStringError); ok {
			if bitStringErr.Code != bitstring.CodeInvalidSize {
				t.Errorf("Expected error code %s, got %s", bitstring.CodeInvalidSize, bitStringErr.Code)
			}
		}
	})

	t.Run("Encode binary with size larger than data", func(t *testing.T) {
		w := newBitWriter()
		segment := &bitstring.Segment{
			Value:         []byte{0xAB}, // 1 byte
			Type:          bitstring.TypeBinary,
			Size:          2, // Requesting 2 bytes
			SizeSpecified: true,
		}

		err := encodeBinary(w, segment)
		if err == nil {
			t.Error("Expected error for size mismatch")
		}

		if bitStringErr, ok := err.(*bitstring.BitStringError); ok {
			if bitStringErr.Code != bitstring.CodeBinarySizeMismatch {
				t.Errorf("Expected error code %s, got %s", bitstring.CodeBinarySizeMismatch, bitStringErr.Code)
			}
		}
	})

	t.Run("Encode binary with size smaller than data", func(t *testing.T) {
		w := newBitWriter()
		segment := &bitstring.Segment{
			Value:         []byte{0xAB, 0xCD}, // 2 bytes
			Type:          bitstring.TypeBinary,
			Size:          1, // Requesting 1 byte (should truncate)
			SizeSpecified: true,
		}

		err := encodeBinary(w, segment)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		// Should have truncated to 1 byte
		data, _ := w.final()
		if len(data) != 1 {
			t.Errorf("Expected 1 byte, got %d", len(data))
		}
		if data[0] != 0xAB {
			t.Errorf("Expected 0xAB, got 0x%02X", data[0])
		}
	})

	t.Run("Encode binary with empty data and size specified", func(t *testing.T) {
		w := newBitWriter()
		segment := &bitstring.Segment{
			Value:         []byte{}, // Empty data
			Type:          bitstring.TypeBinary,
			Size:          1, // Size specified but no data
			SizeSpecified: true,
		}

		err := encodeBinary(w, segment)
		if err == nil {
			t.Error("Expected error for size mismatch with empty data")
		}

		if bitStringErr, ok := err.(*bitstring.BitStringError); ok {
			if bitStringErr.Code != bitstring.CodeBinarySizeMismatch {
				t.Errorf("Expected error code %s, got %s", bitstring.CodeBinarySizeMismatch, bitStringErr.Code)
			}
		}
	})
}
