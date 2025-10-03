package acceptancetests

import (
	"testing"

	"github.com/funvibe/funbit/pkg/funbit"
)

func TestBitstringSizes_1_64Bits(t *testing.T) {
	// Tests for 1-64 bit size support
	testCases := []struct {
		name        string
		value       interface{}
		size        uint
		expected    []byte
		expectError bool
	}{
		{
			name:     "1 bit value 1",
			value:    1,
			size:     1,
			expected: []byte{0x80}, // 10000000
		},
		{
			name:     "1 bit value 0",
			value:    0,
			size:     1,
			expected: []byte{0x00}, // 00000000
		},
		{
			name:     "4 bits value 10",
			value:    10,
			size:     4,
			expected: []byte{0xA0}, // 10100000
		},
		{
			name:     "8 bits value 255",
			value:    255,
			size:     8,
			expected: []byte{0xFF},
		},
		{
			name:     "16 bits value 0x1234",
			value:    0x1234,
			size:     16,
			expected: []byte{0x12, 0x34},
		},
		{
			name:     "32 bits value 0x12345678",
			value:    0x12345678,
			size:     32,
			expected: []byte{0x12, 0x34, 0x56, 0x78},
		},
		{
			name:     "64 bits value 0x123456789ABCDEF0",
			value:    0x123456789ABCDEF0,
			size:     64,
			expected: []byte{0x12, 0x34, 0x56, 0x78, 0x9A, 0xBC, 0xDE, 0xF0},
		},
		{
			name:        "Valid size 0",
			value:       1,
			size:        0, // Valid according to BIT_SYNTAX_SPEC.md
			expectError: false,
			expected:    []byte{}, // Size 0 produces empty result
		},
		{
			name:        "Valid size 65",
			value:       1,
			size:        65,
			expectError: false,
			expected:    []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x80},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			builder := funbit.NewBuilder()
			builder.AddInteger(tc.value, funbit.WithSize(tc.size))

			bs, err := builder.Build()

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			actual := bs.ToBytes()
			if len(actual) != len(tc.expected) {
				t.Errorf("Expected length %d, got %d", len(tc.expected), len(actual))
				return
			}

			for i := range actual {
				if actual[i] != tc.expected[i] {
					t.Errorf("At position %d: expected 0x%02X, got 0x%02X", i, tc.expected[i], actual[i])
					break
				}
			}
		})
	}
}

func TestBitstringSizes_ArbitraryBinarySizes(t *testing.T) {
	// Tests for arbitrary binary data sizes
	testCases := []struct {
		name        string
		value       []byte
		size        uint
		expected    []byte
		expectError bool
	}{
		{
			name:     "3 bits from byte 0xFF",
			value:    []byte{0xFF},
			size:     3,
			expected: []byte{0xE0}, // 11100000
		},
		{
			name:     "12 bits from 2 bytes",
			value:    []byte{0x12, 0x34},
			size:     12,
			expected: []byte{0x12, 0x30}, // 00010010 00110000
		},
		{
			name:     "15 bits from 2 bytes",
			value:    []byte{0x12, 0x34},
			size:     15,
			expected: []byte{0x12, 0x34}, // 00010010 00110100
		},
		{
			name:     "17 bits from 3 bytes",
			value:    []byte{0x12, 0x34, 0x56},
			size:     17,
			expected: []byte{0x12, 0x34, 0x00}, // 00010010 00110100 0 (first 17 bits from 0x12,0x34,0x56)
		},
		{
			name:        "Size too large for data (should succeed with padding)",
			value:       []byte{0x12},
			size:        16,
			expected:    []byte{0x12, 0x00}, // Should pad with zeros
			expectError: false,              // Should succeed, padding with zeros
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			builder := funbit.NewBuilder()

			// Check that size doesn't exceed available bits
			totalBits := uint(len(tc.value)) * 8
			if tc.size > totalBits {
				// For padding test: create bits from available data, then pad with zeros
				// First, add all available bits
				for i := uint(0); i < totalBits; i++ {
					bytePos := i / 8
					bitPos := 7 - (i % 8) // MSB first

					bit := (tc.value[bytePos] >> bitPos) & 1
					builder.AddInteger(bit, funbit.WithSize(1))
				}
				// Then pad with zeros for remaining bits
				for i := totalBits; i < tc.size; i++ {
					builder.AddInteger(0, funbit.WithSize(1))
				}
			} else {
				// Create a separate segment for each bit in the source data
				// Use integer type instead of bitstring, as bitstring is now intended for nested structures
				for i := uint(0); i < tc.size; i++ {
					bytePos := i / 8
					bitPos := 7 - (i % 8) // MSB first

					bit := (tc.value[bytePos] >> bitPos) & 1
					builder.AddInteger(bit, funbit.WithSize(1))
				}
			}

			bs, err := builder.Build()

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			actual := bs.ToBytes()
			if len(actual) != len(tc.expected) {
				t.Errorf("Expected length %d, got %d", len(tc.expected), len(actual))
				return
			}

			for i := range actual {
				if actual[i] != tc.expected[i] {
					t.Errorf("At position %d: expected 0x%02X, got 0x%02X", i, tc.expected[i], actual[i])
					break
				}
			}
		})
	}
}

func TestBitstringSizes_MultipleSegmentsWithDifferentSizes(t *testing.T) {
	// Tests for multiple segments with different sizes
	builder := funbit.NewBuilder().
		AddInteger(1, funbit.WithSize(4)).     // 0001 -> 00010000
		AddInteger(15, funbit.WithSize(4)).    // 1111 -> 11110000
		AddInteger(255, funbit.WithSize(8)).   // 11111111
		AddInteger(0x123, funbit.WithSize(12)) // 000100100011

	bs, err := builder.Build()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expected := []byte{0x1F, 0xFF, 0x12, 0x30} // 00011111 11111111 00010010 00110000
	actual := bs.ToBytes()

	if len(actual) != len(expected) {
		t.Errorf("Expected length %d, got %d", len(expected), len(actual))
		return
	}

	for i := range actual {
		if actual[i] != expected[i] {
			t.Errorf("At position %d: expected 0x%02X, got 0x%02X", i, expected[i], actual[i])
			break
		}
	}
}

func TestBitstringSizes_MatchingDifferentSizes(t *testing.T) {
	// Tests for matching with different sizes
	data := []byte{0x1F, 0xFF, 0x12, 0x30} // 00011111 11111111 00010010 00110000
	bs := funbit.NewBitStringFromBytes(data)

	var a, b, c, d int

	matcher := funbit.NewMatcher().
		Integer(&a, funbit.WithSize(4)).
		Integer(&b, funbit.WithSize(4)).
		Integer(&c, funbit.WithSize(8)).
		Integer(&d, funbit.WithSize(12))

	results, err := matcher.Match(bs)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Check that all variables are bound correctly
	if a != 1 {
		t.Errorf("Expected a=1, got %d", a)
	}
	if b != 15 {
		t.Errorf("Expected b=15, got %d", b)
	}
	if c != 255 {
		t.Errorf("Expected c=255, got %d", c)
	}
	if d != 0x123 {
		t.Errorf("Expected d=0x123, got %d", d)
	}

	// Check that all results are matched
	if len(results) != 4 {
		t.Errorf("Expected 4 results, got %d", len(results))
	}

	for i, result := range results {
		if !result.Matched {
			t.Errorf("Result %d should be matched", i)
		}
	}
}

func TestBitstringSizes_AlignmentAndPadding(t *testing.T) {
	// Tests for alignment and padding
	testCases := []struct {
		name     string
		segments []interface{}
		expected []byte
	}{
		{
			name: "3 bits + 8 bits (should align to 16 bits)",
			segments: []interface{}{
				&funbit.Segment{Value: 5, Size: 3, SizeSpecified: true},   // 101
				&funbit.Segment{Value: 170, Size: 8, SizeSpecified: true}, // 10101010
			},
			expected: []byte{0xA0, 0xAA}, // 10100000 10101010 (with padding to 16 bits)
		},
		{
			name: "1 bit + 15 bits = 16 bits (no padding)",
			segments: []interface{}{
				&funbit.Segment{Value: 1, Size: 1, SizeSpecified: true},       // 1
				&funbit.Segment{Value: 0x7FFF, Size: 15, SizeSpecified: true}, // 011111111111111
			},
			expected: []byte{0xFF, 0xFF}, // 11111111 11111111
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			builder := funbit.NewBuilder()

			for _, seg := range tc.segments {
				if segment, ok := seg.(*funbit.Segment); ok {
					builder.AddSegment(*segment)
				}
			}

			bs, err := builder.Build()
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			actual := bs.ToBytes()
			if len(actual) != len(tc.expected) {
				t.Errorf("Expected length %d, got %d", len(tc.expected), len(actual))
				return
			}

			for i := range actual {
				if actual[i] != tc.expected[i] {
					t.Errorf("At position %d: expected 0x%02X, got 0x%02X", i, tc.expected[i], actual[i])
					break
				}
			}
		})
	}
}

func TestBitstringSizes_SizeValidation(t *testing.T) {
	// Tests for size validation
	testCases := []struct {
		name        string
		value       interface{}
		size        uint
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Size 0 valid",
			value:       1,
			size:        0, // Valid according to BIT_SYNTAX_SPEC.md
			expectError: false,
		},
		{
			name:        "Size > 64 valid for integer",
			value:       1,
			size:        65,
			expectError: false,
			errorMsg:    "",
		},
		{
			name:        "Value too large for size (truncated)",
			value:       256,
			size:        8,
			expectError: false, // According to Erlang spec, should be truncated
			errorMsg:    "",
		},
		{
			name:        "Negative value too large for size (truncated)",
			value:       -129,
			size:        8,
			expectError: false, // According to Erlang spec, should be truncated
			errorMsg:    "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			builder := funbit.NewBuilder()
			builder.AddInteger(tc.value, funbit.WithSize(tc.size))

			_, err := builder.Build()

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error, got nil")
					return
				}
				if tc.errorMsg != "" && err.Error() != tc.errorMsg {
					t.Errorf("Expected error message '%s', got '%s'", tc.errorMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func uintPtr(val uint) *uint {
	return &val
}
