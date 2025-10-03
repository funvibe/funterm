package builder

import (
	"testing"

	"github.com/funvibe/funbit/internal/bitstring"
)

// TestBuilder_encodeUTF_MissingNativeEndianness tests native endianness paths in encodeUTF
func TestBuilder_encodeUTF_MissingNativeEndianness(t *testing.T) {
	t.Run("Encode UTF8 with native endianness", func(t *testing.T) {
		w := newBitWriter()
		segment := &bitstring.Segment{
			Value:      65, // 'A'
			Type:       "utf8",
			Endianness: bitstring.EndiannessNative,
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

	t.Run("Encode UTF16 with native endianness", func(t *testing.T) {
		w := newBitWriter()
		segment := &bitstring.Segment{
			Value:      0x03A9, // Omega symbol
			Type:       "utf16",
			Endianness: bitstring.EndiannessNative,
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

		// The actual byte order depends on the native endianness
		t.Logf("Native endianness UTF16 result: %v", data)
	})

	t.Run("Encode UTF32 with native endianness", func(t *testing.T) {
		w := newBitWriter()
		segment := &bitstring.Segment{
			Value:      0x1F600, // Grinning face emoji
			Type:       "utf32",
			Endianness: bitstring.EndiannessNative,
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

		// The actual byte order depends on the native endianness
		t.Logf("Native endianness UTF32 result: %v", data)
	})

	t.Run("Encode UTF with uint16 value type", func(t *testing.T) {
		w := newBitWriter()
		segment := &bitstring.Segment{
			Value:      uint16(0x03A9), // Omega symbol as uint16
			Type:       "utf16",
			Endianness: bitstring.EndiannessBig,
		}

		err := encodeUTF(w, segment)
		if err == nil {
			t.Error("Expected error for uint16 value type, got nil")
		} else if err.Error() != "unsupported value type for UTF: uint16" {
			t.Errorf("Expected 'unsupported value type for UTF: uint16', got %v", err)
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

	t.Run("Encode UTF with uint64 value type", func(t *testing.T) {
		w := newBitWriter()
		segment := &bitstring.Segment{
			Value:      uint64(0x1F600), // Grinning face emoji as uint64
			Type:       "utf32",
			Endianness: bitstring.EndiannessBig,
		}

		err := encodeUTF(w, segment)
		if err != nil {
			t.Errorf("Expected no error for uint64 value, got %v", err)
		}

		data, totalBits := w.final()
		if totalBits != 32 {
			t.Errorf("Expected totalBits 32, got %d", totalBits)
		}

		if len(data) != 4 {
			t.Errorf("Expected 4 bytes, got %d", len(data))
		}
	})

	t.Run("Encode UTF with int32 value type", func(t *testing.T) {
		w := newBitWriter()
		segment := &bitstring.Segment{
			Value:      int32(65), // 'A' as int32
			Type:       "utf8",
			Endianness: bitstring.EndiannessBig,
		}

		err := encodeUTF(w, segment)
		if err != nil {
			t.Errorf("Expected no error for int32 value, got %v", err)
		}

		data, totalBits := w.final()
		if totalBits != 8 {
			t.Errorf("Expected totalBits 8, got %d", totalBits)
		}

		if len(data) != 1 || data[0] != 65 {
			t.Errorf("Expected byte [65], got %v", data)
		}
	})

	t.Run("Encode UTF with int64 value type", func(t *testing.T) {
		w := newBitWriter()
		segment := &bitstring.Segment{
			Value:      int64(0x1F600), // Grinning face emoji as int64
			Type:       "utf32",
			Endianness: bitstring.EndiannessBig,
		}

		err := encodeUTF(w, segment)
		if err != nil {
			t.Errorf("Expected no error for int64 value, got %v", err)
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
