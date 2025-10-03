package acceptancetests

import (
	"math"
	"testing"

	"github.com/funvibe/funbit/internal/bitstring"
	"github.com/funvibe/funbit/internal/builder"
	"github.com/funvibe/funbit/internal/matcher"
)

// TestBitstringTypes_IntegerType tests explicit integer type specification
func TestBitstringTypes_IntegerType(t *testing.T) {
	// <<1:8/integer, 2:16/integer, 3:32/integer>>
	intValue1 := 1
	intValue2 := 2
	intValue3 := 3

	bs, err := builder.NewBuilder().
		AddInteger(intValue1, bitstring.WithSize(8), bitstring.WithType("integer")).
		AddInteger(intValue2, bitstring.WithSize(16), bitstring.WithType("integer")).
		AddInteger(intValue3, bitstring.WithSize(32), bitstring.WithType("integer")).
		Build()

	if err != nil {
		t.Fatalf("Expected to build bitstring, got error: %v", err)
	}

	var a, b, c int
	_, err = matcher.NewMatcher().
		Integer(&a, bitstring.WithSize(8), bitstring.WithType("integer")).
		Integer(&b, bitstring.WithSize(16), bitstring.WithType("integer")).
		Integer(&c, bitstring.WithSize(32), bitstring.WithType("integer")).
		Match(bs)

	if err != nil {
		t.Fatalf("Expected to match pattern, got error: %v", err)
	}

	if a != 1 || b != 2 || c != 3 {
		t.Errorf("Expected integers 1, 2, 3, got %d, %d, %d", a, b, c)
	}
}

// TestBitstringTypes_Float32 tests 32-bit float values
func TestBitstringTypes_Float32(t *testing.T) {
	// <<3.14:32/float>>
	floatValue := float32(3.14)

	bs, err := builder.NewBuilder().
		AddFloat(floatValue, bitstring.WithSize(32), bitstring.WithType("float")).
		Build()

	if err != nil {
		t.Fatalf("Expected to build bitstring, got error: %v", err)
	}

	var result float32
	_, err = matcher.NewMatcher().
		Float(&result, bitstring.WithSize(32), bitstring.WithType("float")).
		Match(bs)

	if err != nil {
		t.Fatalf("Expected to match pattern, got error: %v", err)
	}

	// Use acceptable tolerance for float comparison
	if math.Abs(float64(result-3.14)) > 0.001 {
		t.Errorf("Expected float ~3.14, got %f", result)
	}
}

// TestBitstringTypes_Float64 tests 64-bit float values
func TestBitstringTypes_Float64(t *testing.T) {
	// <<2.718:64/float>>
	floatValue := 2.718

	bs, err := builder.NewBuilder().
		AddFloat(floatValue, bitstring.WithSize(64), bitstring.WithType("float")).
		Build()

	if err != nil {
		t.Fatalf("Expected to build bitstring, got error: %v", err)
	}

	var result float64
	_, err = matcher.NewMatcher().
		Float(&result, bitstring.WithSize(64), bitstring.WithType("float")).
		Match(bs)

	if err != nil {
		t.Fatalf("Expected to match pattern, got error: %v", err)
	}

	// Use acceptable tolerance for float comparison
	if math.Abs(result-2.718) > 0.0001 {
		t.Errorf("Expected float ~2.718, got %f", result)
	}
}

// TestBitstringTypes_Binary tests binary type (byte-aligned)
func TestBitstringTypes_Binary(t *testing.T) {
	// <<"hello">>
	data := "hello"

	bs, err := builder.NewBuilder().
		AddBinary([]byte(data), bitstring.WithType("binary")).
		Build()

	if err != nil {
		t.Fatalf("Expected to build bitstring, got error: %v", err)
	}

	var result []byte
	_, err = matcher.NewMatcher().
		Binary(&result, bitstring.WithType("binary")).
		Match(bs)

	if err != nil {
		t.Fatalf("Expected to match pattern, got error: %v", err)
	}

	if string(result) != "hello" {
		t.Errorf("Expected 'hello', got '%s'", string(result))
	}
}

// TestBitstringTypes_CombinedBinary tests combining binary data
func TestBitstringTypes_CombinedBinary(t *testing.T) {
	// <<data/binary, " world"/binary>> where data = <<"hello">>
	data := "hello"
	world := " world"

	bs, err := builder.NewBuilder().
		AddBinary([]byte(data), bitstring.WithType("binary")).
		AddBinary([]byte(world), bitstring.WithType("binary")).
		Build()

	if err != nil {
		t.Fatalf("Expected to build bitstring, got error: %v", err)
	}

	var result []byte
	_, err = matcher.NewMatcher().
		Binary(&result, bitstring.WithSize(11), bitstring.WithType("binary")). // 5 + 6 = 11 bytes
		Match(bs)

	if err != nil {
		t.Fatalf("Expected to match pattern, got error: %v", err)
	}

	if string(result) != "hello world" {
		t.Errorf("Expected 'hello world', got '%s'", string(result))
	}
}

// TestBitstringTypes_Bitstring tests bitstring type (not byte-aligned)
func TestBitstringTypes_Bitstring(t *testing.T) {
	// <<1:1, 0:1, 1:1, 1:1, 0:4>>
	// Use integer type for individual bits, as bitstring is now intended for nested structures
	bs, err := builder.NewBuilder().
		AddInteger(1, bitstring.WithSize(1)).
		AddInteger(0, bitstring.WithSize(1)).
		AddInteger(1, bitstring.WithSize(1)).
		AddInteger(1, bitstring.WithSize(1)).
		AddInteger(0, bitstring.WithSize(4)).
		Build()

	if err != nil {
		t.Fatalf("Expected to build bitstring, got error: %v", err)
	}

	// Should result in 8 bits (1 byte) with value 10110000 in binary = 0xB0
	if bs.Length() != 8 {
		t.Errorf("Expected bitstring length 8, got %d", bs.Length())
	}

	bytes := bs.ToBytes()
	if len(bytes) != 1 || bytes[0] != 0xB0 {
		t.Errorf("Expected byte 0xB0, got len=%d, bytes[0]=0x%02X", len(bytes), bytes[0])
	}

	// Match each bit separately
	var bit1, bit2, bit3, bit4, padding int
	_, err = matcher.NewMatcher().
		Integer(&bit1, bitstring.WithSize(1)).
		Integer(&bit2, bitstring.WithSize(1)).
		Integer(&bit3, bitstring.WithSize(1)).
		Integer(&bit4, bitstring.WithSize(1)).
		Integer(&padding, bitstring.WithSize(4)).
		Match(bs)

	if err != nil {
		t.Fatalf("Expected to match pattern, got error: %v", err)
	}

	if bit1 != 1 || bit2 != 0 || bit3 != 1 || bit4 != 1 || padding != 0 {
		t.Errorf("Expected bits 1,0,1,1,0, got %d,%d,%d,%d,%d", bit1, bit2, bit3, bit4, padding)
	}
}

// TestBitstringTypes_MultipleTypeSegments tests mixed types in one bitstring
func TestBitstringTypes_MultipleTypeSegments(t *testing.T) {
	// Combined bitstring with different types
	// Use integer instead of bitstring for individual bits, as bitstring is now intended for nested structures
	bs, err := builder.NewBuilder().
		AddInteger(42, bitstring.WithSize(8), bitstring.WithType("integer")). // integer
		AddFloat(3.14, bitstring.WithSize(32), bitstring.WithType("float")).  // float 32
		AddBinary([]byte("test"), bitstring.WithType("binary")).              // binary
		AddInteger(1, bitstring.WithSize(1)).                                 // integer (was bitstring)
		AddInteger(0, bitstring.WithSize(7)).                                 // padding to byte (was bitstring)
		Build()

	if err != nil {
		t.Fatalf("Expected to build bitstring, got error: %v", err)
	}

	var intValue int
	var floatValue float32
	var binaryData []byte
	var bit1, padding int

	_, err = matcher.NewMatcher().
		Integer(&intValue, bitstring.WithSize(8), bitstring.WithType("integer")).
		Float(&floatValue, bitstring.WithSize(32), bitstring.WithType("float")).
		Binary(&binaryData, bitstring.WithSize(4), bitstring.WithType("binary")).
		Integer(&bit1, bitstring.WithSize(1)).    // integer (was bitstring)
		Integer(&padding, bitstring.WithSize(7)). // padding to byte (was bitstring)
		Match(bs)

	if err != nil {
		t.Fatalf("Expected to match pattern, got error: %v", err)
	}

	if intValue != 42 {
		t.Errorf("Expected integer 42, got %d", intValue)
	}
	if math.Abs(float64(floatValue-3.14)) > 0.001 {
		t.Errorf("Expected float ~3.14, got %f", floatValue)
	}
	if string(binaryData) != "test" {
		t.Errorf("Expected binary 'test', got '%s'", string(binaryData))
	}
	if bit1 != 1 || padding != 0 {
		t.Errorf("Expected bits 1 and 0, got %d and %d", bit1, padding)
	}
}

// TestBitstringTypes_FloatSpecialValues tests special float values
func TestBitstringTypes_FloatSpecialValues(t *testing.T) {
	// Test NaN, Infinity, -Infinity
	testCases := []struct {
		name    string
		value   float64
		size    uint
		isNaN   bool
		isInf   bool
		infSign int // 1 for +Inf, -1 for -Inf
	}{
		{"PositiveInfinity", math.Inf(1), 32, false, true, 1},
		{"NegativeInfinity", math.Inf(-1), 32, false, true, -1},
		{"NaN", math.NaN(), 32, true, false, 0},
		{"PositiveInfinity64", math.Inf(1), 64, false, true, 1},
		{"NegativeInfinity64", math.Inf(-1), 64, false, true, -1},
		{"NaN64", math.NaN(), 64, true, false, 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			bs, err := builder.NewBuilder().
				AddFloat(tc.value, bitstring.WithSize(tc.size), bitstring.WithType("float")).
				Build()

			if err != nil {
				t.Fatalf("Expected to build bitstring, got error: %v", err)
			}

			var result float64
			_, err = matcher.NewMatcher().
				Float(&result, bitstring.WithSize(tc.size), bitstring.WithType("float")).
				Match(bs)

			if err != nil {
				t.Fatalf("Expected to match pattern, got error: %v", err)
			}

			if tc.isNaN {
				if !math.IsNaN(result) {
					t.Errorf("Expected NaN, got %f", result)
				}
			} else if tc.isInf {
				if !math.IsInf(result, tc.infSign) {
					sign := '+'
					if tc.infSign < 0 {
						sign = '-'
					}
					t.Errorf("Expected %cInf, got %f", sign, result)
				}
			}
		})
	}
}

// TestBitstringTypes_BinaryAlignment tests binary data alignment
func TestBitstringTypes_BinaryAlignment(t *testing.T) {
	// Test: binary data should be byte-aligned
	bs, err := builder.NewBuilder().
		AddInteger(1, bitstring.WithSize(4)).                  // 4 bits
		AddBinary([]byte("AB"), bitstring.WithType("binary")). // 16 bits (2 bytes)
		AddInteger(2, bitstring.WithSize(4)).                  // 4 bits
		Build()

	if err != nil {
		t.Fatalf("Expected to build bitstring, got error: %v", err)
	}

	// Total length: 4 + 16 + 4 = 24 bits = 3 bytes
	if bs.Length() != 24 {
		t.Errorf("Expected bitstring length 24, got %d", bs.Length())
	}

	var firstPart, lastPart int
	var binaryData []byte

	_, err = matcher.NewMatcher().
		Integer(&firstPart, bitstring.WithSize(4)).
		Binary(&binaryData, bitstring.WithSize(2), bitstring.WithType("binary")).
		Integer(&lastPart, bitstring.WithSize(4)).
		Match(bs)

	if err != nil {
		t.Fatalf("Expected to match pattern, got error: %v", err)
	}

	if firstPart != 1 || lastPart != 2 {
		t.Errorf("Expected first=1, last=2, got %d and %d", firstPart, lastPart)
	}
	if string(binaryData) != "AB" {
		t.Errorf("Expected binary 'AB', got '%s'", string(binaryData))
	}
}

// TestBitstringTypes_BitstringAlignment tests unaligned bitstring data
func TestBitstringTypes_BitstringAlignment(t *testing.T) {
	// Test: bitstring data doesn't require byte alignment
	// Use integer instead of bitstring for individual bits, as bitstring is now intended for nested structures
	bs, err := builder.NewBuilder().
		AddInteger(1, bitstring.WithSize(3)). // 3 bits
		AddInteger(2, bitstring.WithSize(5)). // 5 bits
		AddInteger(3, bitstring.WithSize(7)). // 7 bits
		AddInteger(1, bitstring.WithSize(1)). // 1 bit (value 1, not 4)
		Build()

	if err != nil {
		t.Fatalf("Expected to build bitstring, got error: %v", err)
	}

	// Total length: 3 + 5 + 7 + 1 = 16 bits = 2 bytes
	if bs.Length() != 16 {
		t.Errorf("Expected bitstring length 16, got %d", bs.Length())
	}

	var a, b, c, d int
	_, err = matcher.NewMatcher().
		Integer(&a, bitstring.WithSize(3)).
		Integer(&b, bitstring.WithSize(5)).
		Integer(&c, bitstring.WithSize(7)).
		Integer(&d, bitstring.WithSize(1)).
		Match(bs)

	if err != nil {
		t.Fatalf("Expected to match pattern, got error: %v", err)
	}

	if a != 1 || b != 2 || c != 3 || d != 1 {
		t.Errorf("Expected values 1,2,3,1, got %d,%d,%d,%d", a, b, c, d)
	}
}
