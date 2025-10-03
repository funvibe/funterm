package acceptancetests

import (
	"encoding/binary"
	"math"
	"testing"
	"unsafe"

	"github.com/funvibe/funbit/internal/bitstring"
	"github.com/funvibe/funbit/internal/builder"
	"github.com/funvibe/funbit/internal/matcher"
)

func TestBitstringEndianness_BigEndian(t *testing.T) {
	// Test basic big-endian encoding (default)
	bs, err := builder.NewBuilder().
		AddInteger(0x1234, bitstring.WithSize(16)).
		Build()

	if err != nil {
		t.Fatalf("Expected to build bitstring, got error: %v", err)
	}

	// Big-endian: 0x12 0x34
	expected := []byte{0x12, 0x34}
	actual := bs.ToBytes()
	if len(actual) != len(expected) {
		t.Errorf("Expected length %d, got %d", len(expected), len(actual))
	} else {
		for i := range expected {
			if actual[i] != expected[i] {
				t.Errorf("At position %d: expected 0x%02X, got 0x%02X", i, expected[i], actual[i])
			}
		}
	}
}

func TestBitstringEndianness_BigEndianExplicit(t *testing.T) {
	// Test explicit big-endian specification
	bs, err := builder.NewBuilder().
		AddInteger(0x1234, bitstring.WithSize(16), bitstring.WithEndianness("big")).
		Build()

	if err != nil {
		t.Fatalf("Expected to build bitstring, got error: %v", err)
	}

	// Big-endian: 0x12 0x34
	expected := []byte{0x12, 0x34}
	actual := bs.ToBytes()
	if len(actual) != len(expected) {
		t.Errorf("Expected length %d, got %d", len(expected), len(actual))
	} else {
		for i := range expected {
			if actual[i] != expected[i] {
				t.Errorf("At position %d: expected 0x%02X, got 0x%02X", i, expected[i], actual[i])
			}
		}
	}
}

func TestBitstringEndianness_LittleEndian(t *testing.T) {
	// Test little-endian encoding
	bs, err := builder.NewBuilder().
		AddInteger(0x1234, bitstring.WithSize(16), bitstring.WithEndianness("little")).
		Build()

	if err != nil {
		t.Fatalf("Expected to build bitstring, got error: %v", err)
	}

	// Little-endian: 0x34 0x12
	expected := []byte{0x34, 0x12}
	actual := bs.ToBytes()
	if len(actual) != len(expected) {
		t.Errorf("Expected length %d, got %d", len(expected), len(actual))
	} else {
		for i := range expected {
			if actual[i] != expected[i] {
				t.Errorf("At position %d: expected 0x%02X, got 0x%02X", i, expected[i], actual[i])
			}
		}
	}
}

func TestBitstringEndianness_NativeEndian(t *testing.T) {
	// Test native-endian encoding (depends on system)
	bs, err := builder.NewBuilder().
		AddInteger(0x1234, bitstring.WithSize(16), bitstring.WithEndianness("native")).
		Build()

	if err != nil {
		t.Fatalf("Expected to build bitstring, got error: %v", err)
	}

	// Native should match system endianness
	var expected []byte
	if isLittleEndian() {
		expected = []byte{0x34, 0x12} // little-endian
	} else {
		expected = []byte{0x12, 0x34} // big-endian
	}
	actual := bs.ToBytes()
	if len(actual) != len(expected) {
		t.Errorf("Expected length %d, got %d", len(expected), len(actual))
	} else {
		for i := range expected {
			if actual[i] != expected[i] {
				t.Errorf("At position %d: expected 0x%02X, got 0x%02X", i, expected[i], actual[i])
			}
		}
	}
}

func TestBitstringEndianness_NativeEndianMatchesSystem(t *testing.T) {
	// Test that native endian matches system endianness
	bs, err := builder.NewBuilder().
		AddInteger(0x12345678, bitstring.WithSize(32), bitstring.WithEndianness("native")).
		Build()

	if err != nil {
		t.Fatalf("Expected to build bitstring, got error: %v", err)
	}

	// Compare with system's native encoding
	var buf [4]byte
	if isLittleEndian() {
		binary.LittleEndian.PutUint32(buf[:], 0x12345678)
	} else {
		binary.BigEndian.PutUint32(buf[:], 0x12345678)
	}

	expected := buf[:]
	actual := bs.ToBytes()
	if len(actual) != len(expected) {
		t.Errorf("Expected length %d, got %d", len(expected), len(actual))
	} else {
		for i := range expected {
			if actual[i] != expected[i] {
				t.Errorf("At position %d: expected 0x%02X, got 0x%02X", i, expected[i], actual[i])
			}
		}
	}
}

func TestBitstringEndianness_MixedEndianness(t *testing.T) {
	// Test mixing different endianness in same bitstring
	bs, err := builder.NewBuilder().
		AddInteger(0x1234, bitstring.WithSize(16), bitstring.WithEndianness("big")).
		AddInteger(0x5678, bitstring.WithSize(16), bitstring.WithEndianness("little")).
		Build()

	if err != nil {
		t.Fatalf("Expected to build bitstring, got error: %v", err)
	}

	// Big-endian 0x1234: 0x12 0x34
	// Little-endian 0x5678: 0x78 0x56
	expected := []byte{0x12, 0x34, 0x78, 0x56}
	actual := bs.ToBytes()
	if len(actual) != len(expected) {
		t.Errorf("Expected length %d, got %d", len(expected), len(actual))
	} else {
		for i := range expected {
			if actual[i] != expected[i] {
				t.Errorf("At position %d: expected 0x%02X, got 0x%02X", i, expected[i], actual[i])
			}
		}
	}
}

func TestBitstringEndianness_FloatBigEndian(t *testing.T) {
	// Test float with big-endian
	value := float32(3.14)
	bs, err := builder.NewBuilder().
		AddFloat(value, bitstring.WithSize(32), bitstring.WithEndianness("big")).
		Build()

	if err != nil {
		t.Fatalf("Expected to build bitstring, got error: %v", err)
	}

	// Compare with Go's big-endian encoding
	var buf [4]byte
	binary.BigEndian.PutUint32(buf[:], math.Float32bits(value))
	expected := buf[:]
	actual := bs.ToBytes()
	if len(actual) != len(expected) {
		t.Errorf("Expected length %d, got %d", len(expected), len(actual))
	} else {
		for i := range expected {
			if actual[i] != expected[i] {
				t.Errorf("At position %d: expected 0x%02X, got 0x%02X", i, expected[i], actual[i])
			}
		}
	}
}

func TestBitstringEndianness_FloatLittleEndian(t *testing.T) {
	// Test float with little-endian
	value := float32(3.14)
	bs, err := builder.NewBuilder().
		AddFloat(value, bitstring.WithSize(32), bitstring.WithEndianness("little")).
		Build()

	if err != nil {
		t.Fatalf("Expected to build bitstring, got error: %v", err)
	}

	// Compare with Go's little-endian encoding
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], math.Float32bits(value))
	expected := buf[:]
	actual := bs.ToBytes()
	if len(actual) != len(expected) {
		t.Errorf("Expected length %d, got %d", len(expected), len(actual))
	} else {
		for i := range expected {
			if actual[i] != expected[i] {
				t.Errorf("At position %d: expected 0x%02X, got 0x%02X", i, expected[i], actual[i])
			}
		}
	}
}

func TestBitstringEndianness_MatchingBigEndian(t *testing.T) {
	// Test matching big-endian data
	data := []byte{0x12, 0x34} // 0x1234 in big-endian
	bs := bitstring.NewBitStringFromBytes(data)

	var result int
	_, err := matcher.NewMatcher().
		Integer(&result, bitstring.WithSize(16), bitstring.WithEndianness("big")).
		Match(bs)

	if err != nil {
		t.Fatalf("Expected to match pattern, got error: %v", err)
	}

	if result != 0x1234 {
		t.Errorf("Expected 0x1234, got 0x%04X", result)
	}
}

func TestBitstringEndianness_MatchingLittleEndian(t *testing.T) {
	// Test matching little-endian data
	data := []byte{0x34, 0x12} // 0x1234 in little-endian
	bs := bitstring.NewBitStringFromBytes(data)

	var result int
	_, err := matcher.NewMatcher().
		Integer(&result, bitstring.WithSize(16), bitstring.WithEndianness("little")).
		Match(bs)

	if err != nil {
		t.Fatalf("Expected to match pattern, got error: %v", err)
	}

	if result != 0x1234 {
		t.Errorf("Expected 0x1234, got 0x%04X", result)
	}
}

func TestBitstringEndianness_MatchingNativeEndian(t *testing.T) {
	// Test matching native-endian data
	var data []byte
	if isLittleEndian() {
		data = []byte{0x34, 0x12} // little-endian representation
	} else {
		data = []byte{0x12, 0x34} // big-endian representation
	}

	bs := bitstring.NewBitStringFromBytes(data)

	var result int
	_, err := matcher.NewMatcher().
		Integer(&result, bitstring.WithSize(16), bitstring.WithEndianness("native")).
		Match(bs)

	if err != nil {
		t.Fatalf("Expected to match pattern, got error: %v", err)
	}

	if result != 0x1234 {
		t.Errorf("Expected 0x1234, got 0x%04X", result)
	}
}

func TestBitstringEndianness_DifferentSizes(t *testing.T) {
	// Test endianness with different sizes
	bs, err := builder.NewBuilder().
		AddInteger(0x12345678, bitstring.WithSize(32), bitstring.WithEndianness("big")).
		AddInteger(0xABCD, bitstring.WithSize(16), bitstring.WithEndianness("little")).
		AddInteger(0xEF, bitstring.WithSize(8)). // 8-bit, endianness doesn't matter
		Build()

	if err != nil {
		t.Fatalf("Expected to build bitstring, got error: %v", err)
	}

	// 32-bit big-endian 0x12345678: 0x12 0x34 0x56 0x78
	// 16-bit little-endian 0xABCD: 0xCD 0xAB
	// 8-bit 0xEF: 0xEF
	expected := []byte{0x12, 0x34, 0x56, 0x78, 0xCD, 0xAB, 0xEF}
	actual := bs.ToBytes()
	if len(actual) != len(expected) {
		t.Errorf("Expected length %d, got %d", len(expected), len(actual))
	} else {
		for i := range expected {
			if actual[i] != expected[i] {
				t.Errorf("At position %d: expected 0x%02X, got 0x%02X", i, expected[i], actual[i])
			}
		}
	}
}

func TestBitstringEndianness_64BitValues(t *testing.T) {
	// Test endianness with 64-bit values
	bs, err := builder.NewBuilder().
		AddInteger(0x123456789ABCDEF0, bitstring.WithSize(64), bitstring.WithEndianness("big")).
		Build()

	if err != nil {
		t.Fatalf("Expected to build bitstring, got error: %v", err)
	}

	// 64-bit big-endian: 0x12 0x34 0x56 0x78 0x9A 0xBC 0xDE 0xF0
	expected := []byte{0x12, 0x34, 0x56, 0x78, 0x9A, 0xBC, 0xDE, 0xF0}
	actual := bs.ToBytes()
	if len(actual) != len(expected) {
		t.Errorf("Expected length %d, got %d", len(expected), len(actual))
	} else {
		for i := range expected {
			if actual[i] != expected[i] {
				t.Errorf("At position %d: expected 0x%02X, got 0x%02X", i, expected[i], actual[i])
			}
		}
	}
}

func TestBitstringEndianness_64BitValuesLittleEndian(t *testing.T) {
	// Test endianness with 64-bit values in little-endian
	bs, err := builder.NewBuilder().
		AddInteger(0x123456789ABCDEF0, bitstring.WithSize(64), bitstring.WithEndianness("little")).
		Build()

	if err != nil {
		t.Fatalf("Expected to build bitstring, got error: %v", err)
	}

	// 64-bit little-endian: 0xF0 0xDE 0xBC 0x9A 0x78 0x56 0x34 0x12
	expected := []byte{0xF0, 0xDE, 0xBC, 0x9A, 0x78, 0x56, 0x34, 0x12}
	actual := bs.ToBytes()
	if len(actual) != len(expected) {
		t.Errorf("Expected length %d, got %d", len(expected), len(actual))
	} else {
		for i := range expected {
			if actual[i] != expected[i] {
				t.Errorf("At position %d: expected 0x%02X, got 0x%02X", i, expected[i], actual[i])
			}
		}
	}
}

// Helper function to determine system endianness
func isLittleEndian() bool {
	var i uint32 = 0x01020304
	return *(*byte)(unsafe.Pointer(&i)) == 0x04
}
