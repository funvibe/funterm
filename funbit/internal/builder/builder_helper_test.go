package builder

import (
	"bytes"
	"testing"
)

// TestBuilder_alignToByte tests the alignToByte helper function
func TestBuilder_alignToByte(t *testing.T) {
	t.Run("No alignment needed", func(t *testing.T) {
		w := newBitWriter()
		w.alignToByte()

		if w.bitCount != 0 {
			t.Errorf("Expected bitCount 0, got %d", w.bitCount)
		}

		if w.acc != 0 {
			t.Errorf("Expected acc 0, got %d", w.acc)
		}
	})

	t.Run("Alignment needed", func(t *testing.T) {
		w := newBitWriter()
		w.writeBits(0b101, 3) // Write 3 bits
		w.alignToByte()

		if w.bitCount != 0 {
			t.Errorf("Expected bitCount 0 after alignment, got %d", w.bitCount)
		}

		if w.acc != 0 {
			t.Errorf("Expected acc 0 after alignment, got %d", w.acc)
		}

		// Check that buffer has one byte with 3 bits padded to 8
		bytes := w.buf.Bytes()
		if len(bytes) != 1 {
			t.Errorf("Expected 1 byte in buffer, got %d", len(bytes))
		}

		// 0b101 padded with 5 zeros becomes 0b10100000
		if bytes[0] != 0b10100000 {
			t.Errorf("Expected byte 0b10100000, got 0b%08b", bytes[0])
		}
	})

	t.Run("Already aligned", func(t *testing.T) {
		w := newBitWriter()
		w.writeBits(0xFF, 8) // Write full byte
		w.alignToByte()

		if w.bitCount != 0 {
			t.Errorf("Expected bitCount 0, got %d", w.bitCount)
		}

		// Should have one byte in buffer, no additional padding
		bufBytes := w.buf.Bytes()
		if len(bufBytes) != 1 {
			t.Errorf("Expected 1 byte in buffer, got %d", len(bufBytes))
		}
	})
}

func TestBuilder_writeBytes(t *testing.T) {
	t.Run("Write bytes to empty writer", func(t *testing.T) {
		w := newBitWriter()
		data := []byte{0xAB, 0xCD, 0xEF}
		n, err := w.writeBytes(data)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if n != len(data) {
			t.Errorf("Expected to write %d bytes, wrote %d", len(data), n)
		}

		if w.bitCount != 0 {
			t.Errorf("Expected bitCount 0 after writeBytes, got %d", w.bitCount)
		}

		bufBytes := w.buf.Bytes()
		if !bytes.Equal(bufBytes, data) {
			t.Errorf("Expected bytes %v, got %v", data, bufBytes)
		}
	})

	t.Run("Write bytes after partial byte", func(t *testing.T) {
		w := newBitWriter()
		w.writeBits(0b101, 3) // Write 3 bits first
		data := []byte{0xAB, 0xCD}
		n, err := w.writeBytes(data)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if n != len(data) {
			t.Errorf("Expected to write %d bytes, wrote %d", len(data), n)
		}

		if w.bitCount != 0 {
			t.Errorf("Expected bitCount 0 after writeBytes, got %d", w.bitCount)
		}

		// Should have 3 bytes: padded partial byte + 2 data bytes
		bytes := w.buf.Bytes()
		if len(bytes) != 3 {
			t.Errorf("Expected 3 bytes in buffer, got %d", len(bytes))
		}
	})

	t.Run("Write empty data", func(t *testing.T) {
		w := newBitWriter()
		data := []byte{}
		n, err := w.writeBytes(data)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if n != 0 {
			t.Errorf("Expected to write 0 bytes, wrote %d", n)
		}

		if w.bitCount != 0 {
			t.Errorf("Expected bitCount 0, got %d", w.bitCount)
		}
	})
}

func TestBuilder_final(t *testing.T) {
	t.Run("Empty writer", func(t *testing.T) {
		w := newBitWriter()
		data, totalBits := w.final()

		if len(data) != 0 {
			t.Errorf("Expected empty data, got %v", data)
		}

		if totalBits != 0 {
			t.Errorf("Expected totalBits 0, got %d", totalBits)
		}
	})

	t.Run("Writer with full bytes", func(t *testing.T) {
		w := newBitWriter()
		w.writeBits(0xAB, 8)
		w.writeBits(0xCD, 8)
		data, totalBits := w.final()

		if len(data) != 2 {
			t.Errorf("Expected 2 bytes, got %d", len(data))
		}

		if totalBits != 16 {
			t.Errorf("Expected totalBits 16, got %d", totalBits)
		}

		if data[0] != 0xAB || data[1] != 0xCD {
			t.Errorf("Expected bytes [0xAB, 0xCD], got %v", data)
		}
	})

	t.Run("Writer with partial byte", func(t *testing.T) {
		w := newBitWriter()
		w.writeBits(0b101, 3)
		data, totalBits := w.final()

		if len(data) != 1 {
			t.Errorf("Expected 1 byte, got %d", len(data))
		}

		if totalBits != 3 {
			t.Errorf("Expected totalBits 3, got %d", totalBits)
		}

		// 0b101 should be shifted to MSB: 0b10100000
		if data[0] != 0b10100000 {
			t.Errorf("Expected byte 0b10100000, got 0b%08b", data[0])
		}
	})
}
