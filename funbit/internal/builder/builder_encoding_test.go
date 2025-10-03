package builder

import (
	"testing"

	"github.com/funvibe/funbit/internal/bitstring"
)

// TestBuilder_encodeInteger tests the encodeInteger function
func TestBuilder_encodeInteger(t *testing.T) {
	t.Run("Encode unsigned integer", func(t *testing.T) {
		w := newBitWriter()
		segment := &bitstring.Segment{
			Value:         uint64(42),
			Type:          bitstring.TypeInteger,
			Size:          8,
			SizeSpecified: true,
			Signed:        false,
		}

		err := encodeInteger(w, segment)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		data, totalBits := w.final()
		if totalBits != 8 {
			t.Errorf("Expected totalBits 8, got %d", totalBits)
		}

		if len(data) != 1 || data[0] != 42 {
			t.Errorf("Expected byte [42], got %v", data)
		}
	})

	t.Run("Encode signed integer", func(t *testing.T) {
		w := newBitWriter()
		segment := &bitstring.Segment{
			Value:         int64(-42),
			Type:          bitstring.TypeInteger,
			Size:          8,
			SizeSpecified: true,
			Signed:        true,
		}

		err := encodeInteger(w, segment)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		data, totalBits := w.final()
		if totalBits != 8 {
			t.Errorf("Expected totalBits 8, got %d", totalBits)
		}

		if len(data) != 1 {
			t.Errorf("Expected 1 byte, got %d", len(data))
		}
	})

	t.Run("Encode with little endian", func(t *testing.T) {
		w := newBitWriter()
		segment := &bitstring.Segment{
			Value:         uint64(0x1234),
			Type:          bitstring.TypeInteger,
			Size:          16,
			SizeSpecified: true,
			Endianness:    bitstring.EndiannessLittle,
		}

		err := encodeInteger(w, segment)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		data, totalBits := w.final()
		if totalBits != 16 {
			t.Errorf("Expected totalBits 16, got %d", totalBits)
		}

		if len(data) != 2 || data[0] != 0x34 || data[1] != 0x12 {
			t.Errorf("Expected bytes [0x34, 0x12], got %v", data)
		}
	})
}

// TestBuilder_encodeFloat tests the encodeFloat function
func TestBuilder_encodeFloat(t *testing.T) {
	t.Run("Encode float32", func(t *testing.T) {
		w := newBitWriter()
		segment := &bitstring.Segment{
			Value:         float32(3.14),
			Type:          bitstring.TypeFloat,
			Size:          32,
			SizeSpecified: true,
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
	})

	t.Run("Encode float64", func(t *testing.T) {
		w := newBitWriter()
		segment := &bitstring.Segment{
			Value:         float64(2.718281828459045),
			Type:          bitstring.TypeFloat,
			Size:          64,
			SizeSpecified: true,
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
	})

	t.Run("Encode with little endian", func(t *testing.T) {
		w := newBitWriter()
		segment := &bitstring.Segment{
			Value:         float32(3.14),
			Type:          bitstring.TypeFloat,
			Size:          32,
			SizeSpecified: true,
			Endianness:    bitstring.EndiannessLittle,
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
	})
}

// TestBuilder_encodeBinary tests the encodeBinary function
func TestBuilder_encodeBinary(t *testing.T) {
	t.Run("Encode binary data", func(t *testing.T) {
		w := newBitWriter()
		segment := &bitstring.Segment{
			Value:         []byte{0xAB, 0xCD, 0xEF},
			Type:          bitstring.TypeBinary,
			Size:          3,
			SizeSpecified: true,
		}

		err := encodeBinary(w, segment)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		data, totalBits := w.final()
		if totalBits != 24 {
			t.Errorf("Expected totalBits 24, got %d", totalBits)
		}

		if len(data) != 3 || data[0] != 0xAB || data[1] != 0xCD || data[2] != 0xEF {
			t.Errorf("Expected bytes [0xAB, 0xCD, 0xEF], got %v", data)
		}
	})

	t.Run("Encode empty binary data", func(t *testing.T) {
		w := newBitWriter()
		segment := &bitstring.Segment{
			Value:         []byte{},
			Type:          bitstring.TypeBinary,
			Size:          0,
			SizeSpecified: true,
		}

		err := encodeBinary(w, segment)
		if err == nil {
			t.Error("Expected error for zero size binary data")
		} else {
			t.Logf("Expected error for zero size binary data: %v", err)
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
}

// TestBuilder_encodeUTF tests the encodeUTF function
func TestBuilder_encodeUTF(t *testing.T) {
	t.Run("Encode UTF8", func(t *testing.T) {
		w := newBitWriter()
		segment := &bitstring.Segment{
			Value:      65, // 'A'
			Type:       "utf8",
			Endianness: bitstring.EndiannessBig,
		}

		err := encodeUTF(w, segment)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		data, totalBits := w.final()
		if totalBits != 8 {
			t.Errorf("Expected totalBits 8, got %d", totalBits)
		}

		if len(data) != 1 || data[0] != 65 {
			t.Errorf("Expected byte [65], got %v", data)
		}
	})

	t.Run("Encode UTF16", func(t *testing.T) {
		w := newBitWriter()
		segment := &bitstring.Segment{
			Value:      0x03A9, // Omega symbol
			Type:       "utf16",
			Endianness: bitstring.EndiannessBig,
		}

		err := encodeUTF(w, segment)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		data, totalBits := w.final()
		if totalBits != 16 {
			t.Errorf("Expected totalBits 16, got %d", totalBits)
		}

		if len(data) != 2 {
			t.Errorf("Expected 2 bytes, got %d", len(data))
		}
	})

	t.Run("Encode UTF32", func(t *testing.T) {
		w := newBitWriter()
		segment := &bitstring.Segment{
			Value:      0x1F600, // Grinning face emoji
			Type:       "utf32",
			Endianness: bitstring.EndiannessBig,
		}

		err := encodeUTF(w, segment)
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
	})
}

// TestBuilder_encodeBitstring tests the encodeBitstring function
func TestBuilder_encodeBitstring(t *testing.T) {
	t.Run("Encode bitstring", func(t *testing.T) {
		w := newBitWriter()
		bs := bitstring.NewBitStringFromBits([]byte{0xAB, 0xCD}, 16)
		segment := &bitstring.Segment{
			Value:         bs,
			Type:          bitstring.TypeBitstring,
			Size:          16,
			SizeSpecified: true,
		}

		err := encodeBitstring(w, segment)
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

	t.Run("Encode empty bitstring", func(t *testing.T) {
		w := newBitWriter()
		bs := bitstring.NewBitStringFromBits([]byte{}, 0)
		segment := &bitstring.Segment{
			Value:         bs,
			Type:          bitstring.TypeBitstring,
			Size:          0,
			SizeSpecified: true,
		}

		err := encodeBitstring(w, segment)
		if err == nil {
			t.Error("Expected error for zero size bitstring")
		} else {
			t.Logf("Expected error for zero size bitstring: %v", err)
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
}

// TestBuilder_encodeSegment tests the encodeSegment function
func TestBuilder_encodeSegment(t *testing.T) {
	t.Run("Encode integer segment", func(t *testing.T) {
		w := newBitWriter()
		segment := &bitstring.Segment{
			Value:         uint64(42),
			Type:          bitstring.TypeInteger,
			Size:          8,
			SizeSpecified: true,
		}

		err := encodeSegment(w, segment)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		data, totalBits := w.final()
		if totalBits != 8 {
			t.Errorf("Expected totalBits 8, got %d", totalBits)
		}

		if len(data) != 1 || data[0] != 42 {
			t.Errorf("Expected byte [42], got %v", data)
		}
	})

	t.Run("Encode float segment", func(t *testing.T) {
		w := newBitWriter()
		segment := &bitstring.Segment{
			Value:         float32(3.14),
			Type:          bitstring.TypeFloat,
			Size:          32,
			SizeSpecified: true,
		}

		err := encodeSegment(w, segment)
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
	})

	t.Run("Encode binary segment", func(t *testing.T) {
		w := newBitWriter()
		segment := &bitstring.Segment{
			Value:         []byte{0xAB, 0xCD},
			Type:          bitstring.TypeBinary,
			Size:          2,
			SizeSpecified: true,
		}

		err := encodeSegment(w, segment)
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

	t.Run("Encode UTF8 segment", func(t *testing.T) {
		w := newBitWriter()
		segment := &bitstring.Segment{
			Value:      65, // 'A'
			Type:       "utf8",
			Endianness: bitstring.EndiannessBig,
		}

		err := encodeSegment(w, segment)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		data, totalBits := w.final()
		if totalBits != 8 {
			t.Errorf("Expected totalBits 8, got %d", totalBits)
		}

		if len(data) != 1 || data[0] != 65 {
			t.Errorf("Expected byte [65], got %v", data)
		}
	})

	t.Run("Encode bitstring segment", func(t *testing.T) {
		w := newBitWriter()
		bs := bitstring.NewBitStringFromBits([]byte{0xFF}, 8)
		segment := &bitstring.Segment{
			Value:         bs,
			Type:          bitstring.TypeBitstring,
			Size:          8,
			SizeSpecified: true,
		}

		err := encodeSegment(w, segment)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		data, totalBits := w.final()
		if totalBits != 8 {
			t.Errorf("Expected totalBits 8, got %d", totalBits)
		}

		if len(data) != 1 || data[0] != 0xFF {
			t.Errorf("Expected byte [0xFF], got %v", data)
		}
	})
}
