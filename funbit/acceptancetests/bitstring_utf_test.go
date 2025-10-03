package acceptancetests

import (
	"testing"

	"github.com/funvibe/funbit/pkg/funbit"
)

func TestBitstringUTF(t *testing.T) {
	// Тесты для UTF кодирования (UTF-8, UTF-16, UTF-32)
	testCases := []struct {
		name        string
		value       interface{}
		utfType     string
		endianness  string
		expected    []byte
		expectError bool
	}{
		{
			name:     "UTF-8 ASCII character",
			value:    'A',
			utfType:  "utf8",
			expected: []byte{65}, // 'A' in UTF-8
		},
		{
			name:     "UTF-8 ASCII string",
			value:    "Hello",
			utfType:  "utf8",
			expected: []byte{72, 101, 108, 108, 111}, // "Hello" in UTF-8
		},
		{
			name:     "UTF-8 Unicode character (2 bytes)",
			value:    '©', // copyright symbol
			utfType:  "utf8",
			expected: []byte{194, 169}, // U+00A9 in UTF-8
		},
		{
			name:     "UTF-8 Unicode character (3 bytes)",
			value:    '€', // euro symbol
			utfType:  "utf8",
			expected: []byte{226, 130, 172}, // U+20AC in UTF-8
		},
		{
			name:     "UTF-8 Emoji (4 bytes)",
			value:    '🚀', // rocket emoji
			utfType:  "utf8",
			expected: []byte{240, 159, 154, 128}, // U+1F680 in UTF-8
		},
		{
			name:       "UTF-16 ASCII character",
			value:      'A',
			utfType:    "utf16",
			endianness: "big",
			expected:   []byte{0, 65}, // 'A' in UTF-16 BE
		},
		{
			name:       "UTF-16 ASCII character little-endian",
			value:      'A',
			utfType:    "utf16",
			endianness: "little",
			expected:   []byte{65, 0}, // 'A' in UTF-16 LE
		},
		{
			name:       "UTF-16 Unicode character (BMP)",
			value:      '©', // copyright symbol (U+00A9)
			utfType:    "utf16",
			endianness: "big",
			expected:   []byte{0, 169}, // U+00A9 in UTF-16 BE
		},
		{
			name:       "UTF-16 Unicode character (supplementary plane)",
			value:      '🚀', // rocket emoji (U+1F680)
			utfType:    "utf16",
			endianness: "big",
			expected:   []byte{216, 61, 222, 128}, // U+1F680 as surrogate pair in UTF-16 BE
		},
		{
			name:       "UTF-32 ASCII character",
			value:      'A',
			utfType:    "utf32",
			endianness: "big",
			expected:   []byte{0, 0, 0, 65}, // 'A' in UTF-32 BE
		},
		{
			name:       "UTF-32 ASCII character little-endian",
			value:      'A',
			utfType:    "utf32",
			endianness: "little",
			expected:   []byte{65, 0, 0, 0}, // 'A' in UTF-32 LE
		},
		{
			name:       "UTF-32 Unicode character",
			value:      '€', // euro symbol (U+20AC)
			utfType:    "utf32",
			endianness: "big",
			expected:   []byte{0, 0, 32, 172}, // U+20AC in UTF-32 BE
		},
		{
			name:       "UTF-32 Emoji",
			value:      '🚀', // rocket emoji (U+1F680)
			utfType:    "utf32",
			endianness: "big",
			expected:   []byte{0, 1, 246, 128}, // U+1F680 in UTF-32 BE
		},
		{
			name:        "Invalid Unicode code point (too low)",
			value:       0xD800, // surrogate low
			utfType:     "utf8",
			expectError: true,
		},
		{
			name:        "Invalid Unicode code point (too high)",
			value:       0x110000, // beyond Unicode range
			utfType:     "utf8",
			expectError: true,
		},
		{
			name:        "UTF-8 with size specified (should fail)",
			value:       'A',
			utfType:     "utf8",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			builder := funbit.NewBuilder()

			// Build options based on test case
			options := []funbit.SegmentOption{funbit.WithType(tc.utfType)}

			if tc.endianness != "" {
				options = append(options, funbit.WithEndianness(tc.endianness))
			}

			// For UTF types, size should not be specified according to spec
			// But we need to test that specifying size fails
			if tc.name == "UTF-8 with size specified (should fail)" {
				options = append(options, funbit.WithSize(8))
			}

			if strVal, ok := tc.value.(string); ok {
				// String value - add each character separately
				for _, char := range strVal {
					builder.AddInteger(char, options...)
				}
			} else {
				// Single character/rune value
				builder.AddInteger(tc.value, options...)
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

func TestBitstringUTFMatching(t *testing.T) {
	// Тесты для pattern matching с UTF кодированием
	testCases := []struct {
		name        string
		data        []byte
		patternType string
		endianness  string
		expected    interface{}
		expectError bool
	}{
		{
			name:        "Match UTF-8 ASCII",
			data:        []byte{65}, // 'A'
			patternType: "utf8",
			expected:    int64(65),
		},
		{
			name:        "Match UTF-8 Unicode (2 bytes)",
			data:        []byte{194, 169}, // ©
			patternType: "utf8",
			expected:    int64(0xA9), // U+00A9
		},
		{
			name:        "Match UTF-8 Emoji (4 bytes)",
			data:        []byte{240, 159, 154, 128}, // 🚀
			patternType: "utf8",
			expected:    int64(0x1F680), // U+1F680
		},
		{
			name:        "Match UTF-16 BE ASCII",
			data:        []byte{0, 65}, // 'A'
			patternType: "utf16",
			endianness:  "big",
			expected:    int64(65),
		},
		{
			name:        "Match UTF-16 LE ASCII",
			data:        []byte{65, 0}, // 'A'
			patternType: "utf16",
			endianness:  "little",
			expected:    int64(65),
		},
		{
			name:        "Match UTF-16 BE Unicode",
			data:        []byte{0, 169}, // ©
			patternType: "utf16",
			endianness:  "big",
			expected:    int64(0xA9),
		},
		{
			name:        "Match UTF-16 BE surrogate pair",
			data:        []byte{216, 61, 222, 128}, // 🚀
			patternType: "utf16",
			endianness:  "big",
			expected:    int64(0x1F680),
		},
		{
			name:        "Match UTF-32 BE ASCII",
			data:        []byte{0, 0, 0, 65}, // 'A'
			patternType: "utf32",
			endianness:  "big",
			expected:    int64(65),
		},
		{
			name:        "Match UTF-32 LE ASCII",
			data:        []byte{65, 0, 0, 0}, // 'A'
			patternType: "utf32",
			endianness:  "little",
			expected:    int64(65),
		},
		{
			name:        "Match UTF-32 BE Unicode",
			data:        []byte{0, 0, 32, 172}, // €
			patternType: "utf32",
			endianness:  "big",
			expected:    int64(0x20AC),
		},
		{
			name:        "Match UTF-32 BE Emoji",
			data:        []byte{0, 1, 246, 128}, // 🚀
			patternType: "utf32",
			endianness:  "big",
			expected:    int64(0x1F680),
		},
		{
			name:        "Invalid UTF-8 sequence",
			data:        []byte{255, 255}, // Invalid UTF-8
			patternType: "utf8",
			expectError: true,
		},
		{
			name:        "Invalid UTF-16 surrogate pair",
			data:        []byte{216, 255, 216, 128}, // Invalid surrogate (high + high)
			patternType: "utf16",
			endianness:  "big",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			bs := funbit.NewBitStringFromBytes(tc.data)

			var result string
			matcher := funbit.NewMatcher()

			options := []funbit.SegmentOption{funbit.WithType(tc.patternType)}
			if tc.endianness != "" {
				options = append(options, funbit.WithEndianness(tc.endianness))
			}

			funbit.UTF(matcher, &result, options...)

			results, err := matcher.Match(bs)

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

			if len(results) == 0 || !results[0].Matched {
				t.Errorf("Pattern did not match")
				return
			}

			expectedRune := rune(tc.expected.(int64))
			if result != string(expectedRune) {
				t.Errorf("Expected %s (%d), got %s", string(expectedRune), tc.expected.(int64), result)
			}
		})
	}
}

func TestBitstringUTFStringConstruction(t *testing.T) {
	// Тесты для построения UTF строк из строковых литералов
	testCases := []struct {
		name     string
		input    string
		utfType  string
		expected []byte
	}{
		{
			name:     "String literal UTF-8 ASCII",
			input:    "Hello",
			utfType:  "utf8",
			expected: []byte{72, 101, 108, 108, 111},
		},
		{
			name:     "String literal UTF-8 Unicode",
			input:    "Привет",
			utfType:  "utf8",
			expected: []byte{208, 159, 209, 128, 208, 184, 208, 178, 208, 181, 209, 130},
		},
		{
			name:     "String literal UTF-8 Emoji",
			input:    "🚀",
			utfType:  "utf8",
			expected: []byte{240, 159, 154, 128},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// This test will fail until we implement string literal support
			// For now, we'll simulate by adding each character separately
			builder := funbit.NewBuilder()

			for _, char := range tc.input {
				builder.AddInteger(char, funbit.WithType(tc.utfType))
			}

			bs, err := builder.Build()

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
