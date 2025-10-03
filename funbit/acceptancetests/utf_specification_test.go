package acceptancetests

import (
	"fmt"
	"strings"
	"testing"

	"github.com/funvibe/funbit/pkg/funbit"
)

// Helper function for byte slice comparison
func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestUTFSpecificationCompliance tests UTF encoding/decoding according to Erlang BIT_SYNTAX_SPEC.md
//
// From BIT_SYNTAX_SPEC.md lines 224-240:
// - When constructing: Value must be integer codepoint OR string (syntactic sugar)
// - When matching: Results in INTEGER codepoint (not string!)
// - <<"abc"/utf8>> is syntactic sugar for <<$a/utf8,$b/utf8,$c/utf8>>
func TestUTFSpecificationCompliance(t *testing.T) {

	t.Run("UTF8_Single_Codepoint_Construction", func(t *testing.T) {
		// Erlang: <<1024/utf8>> should produce [208,128]
		builder := funbit.NewBuilder()
		funbit.AddUTF8(builder, string(rune(1024))) // Codepoint 1024 = 'Ѐ'

		bitstring, err := funbit.Build(builder)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		expected := []byte{208, 128} // UTF-8 encoding of codepoint 1024
		if !bytesEqual(expected, bitstring.ToBytes()) {
			t.Errorf("Expected %v, got %v", expected, bitstring.ToBytes())
		}
		if bitstring.Length() != uint(16) {
			t.Errorf("Expected length 16, got %d", bitstring.Length())
		}
	})

	t.Run("UTF8_String_Construction", func(t *testing.T) {
		// Erlang: <<"abc"/utf8>> should produce [97,98,99]
		builder := funbit.NewBuilder()
		funbit.AddUTF8(builder, "abc")

		bitstring, err := funbit.Build(builder)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		expected := []byte{97, 98, 99} // ASCII 'a', 'b', 'c'
		if !bytesEqual(expected, bitstring.ToBytes()) {
			t.Errorf("Expected %v, got %v", expected, bitstring.ToBytes())
		}
		if bitstring.Length() != uint(24) {
			t.Errorf("Expected length 24, got %d", bitstring.Length())
		}
	})

	t.Run("UTF8_Matching_Integer_Codepoint", func(t *testing.T) {
		// CRITICAL: According to BIT_SYNTAX_SPEC.md line 237:
		// "A successful match of a segment of a utf type, results in an INTEGER"

		// Create bitstring with codepoint 1024
		builder := funbit.NewBuilder()
		funbit.AddUTF8(builder, string(rune(1024))) // 'Ѐ'
		funbit.AddUTF8(builder, "abc")              // Additional data

		bitstring, err := funbit.Build(builder)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Match: should extract INTEGER codepoint, not string
		matcher := funbit.NewMatcher()
		var codepoint int
		var rest []byte

		funbit.UTF8(matcher, &codepoint) // Extract as INTEGER (Erlang spec!)
		funbit.RestBinary(matcher, &rest)

		results, err := funbit.Match(matcher, bitstring)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if len(results) == 0 {
			t.Fatalf("Expected non-empty results")
		}

		// Verify: extracted codepoint should be 1024 (integer)
		if codepoint != 1024 {
			t.Errorf("Expected 1024, got %d", codepoint)
		}
		if !bytesEqual([]byte{97, 98, 99}, rest) {
			t.Errorf("Expected %v, got %v", []byte{97, 98, 99}, rest)
		}
	})

	t.Run("UTF8_Matching_Rune_Codepoint", func(t *testing.T) {
		// Test with rune type (Go's native Unicode type)
		builder := funbit.NewBuilder()
		funbit.AddUTF8(builder, "🚀") // Emoji: U+1F680

		bitstring, err := funbit.Build(builder)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		matcher := funbit.NewMatcher()
		var rocket rune

		funbit.UTF8(matcher, &rocket)

		results, err := funbit.Match(matcher, bitstring)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if len(results) == 0 {
			t.Fatalf("Expected non-empty results")
		}

		if rocket != rune(0x1F680) {
			t.Errorf("Expected %v, got %v", rune(0x1F680), rocket)
		}
	})

	t.Run("UTF8_Backward_Compatibility_String", func(t *testing.T) {
		// Maintain backward compatibility: string variables should still work
		// but return the CHARACTER, not the codepoint
		builder := funbit.NewBuilder()
		funbit.AddUTF8(builder, "Ѐ") // Codepoint 1024

		bitstring, err := funbit.Build(builder)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		matcher := funbit.NewMatcher()
		var character string

		funbit.UTF8(matcher, &character)

		results, err := funbit.Match(matcher, bitstring)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if len(results) == 0 {
			t.Fatalf("Expected non-empty results")
		}

		if character != "Ѐ" {
			t.Errorf("Expected %v, got %v", "Ѐ", character)
		}
		if len(character) != 2 {
			t.Errorf("Expected length 2, got %d", len(character))
		}
	})

	t.Run("UTF16_Codepoint_Matching", func(t *testing.T) {
		// Test UTF-16 with integer extraction
		builder := funbit.NewBuilder()
		funbit.AddUTF16(builder, "A", funbit.WithEndianness("big")) // Simple ASCII

		bitstring, err := funbit.Build(builder)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		matcher := funbit.NewMatcher()
		var codepoint int

		funbit.UTF16(matcher, &codepoint, funbit.WithEndianness("big"))

		results, err := funbit.Match(matcher, bitstring)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if len(results) == 0 {
			t.Fatalf("Expected non-empty results")
		}

		if codepoint != 65 {
			t.Errorf("Expected 65, got %d", codepoint)
		}
	})

	t.Run("UTF32_Codepoint_Matching", func(t *testing.T) {
		// Test UTF-32 with integer extraction
		builder := funbit.NewBuilder()
		funbit.AddUTF32(builder, "🌟") // Star emoji: U+1F31F

		bitstring, err := funbit.Build(builder)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		matcher := funbit.NewMatcher()
		var codepoint int

		funbit.UTF32(matcher, &codepoint)

		results, err := funbit.Match(matcher, bitstring)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if len(results) == 0 {
			t.Fatalf("Expected non-empty results")
		}

		if codepoint != 0x1F31F {
			t.Errorf("Expected %d, got %d", 0x1F31F, codepoint)
		}
	})

	t.Run("UTF8_Multiple_Codepoints", func(t *testing.T) {
		// Test extracting multiple UTF codepoints in sequence
		// Erlang: <<A/utf8, B/utf8, Rest/binary>> = <<"Hello">>
		builder := funbit.NewBuilder()
		funbit.AddUTF8(builder, "Hi!")

		bitstring, err := funbit.Build(builder)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		matcher := funbit.NewMatcher()
		var h, i, exclamation int

		funbit.UTF8(matcher, &h)
		funbit.UTF8(matcher, &i)
		funbit.UTF8(matcher, &exclamation)

		results, err := funbit.Match(matcher, bitstring)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if len(results) == 0 {
			t.Fatalf("Expected non-empty results")
		}

		if h != int('H') {
			t.Errorf("Expected %d, got %d", int('H'), h)
		}
		if i != int('i') {
			t.Errorf("Expected %d, got %d", int('i'), i)
		}
		if exclamation != int('!') {
			t.Errorf("Expected %d, got %d", int('!'), exclamation)
		}
	})

	t.Run("UTF8_Invalid_Codepoint_Range", func(t *testing.T) {
		// Test validation of codepoint ranges
		// BIT_SYNTAX_SPEC.md: Value must be in range 0-0xD7FF or 0xE000-0x10FFFF
		builder := funbit.NewBuilder()

		// This should work - valid range
		funbit.AddUTF8(builder, string(rune(0xD7FF))) // Last valid before surrogate range

		bitstring, err := funbit.Build(builder)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		matcher := funbit.NewMatcher()
		var codepoint int
		funbit.UTF8(matcher, &codepoint)

		results, err := funbit.Match(matcher, bitstring)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if len(results) == 0 {
			t.Fatalf("Expected non-empty results")
		}

		if codepoint != 0xD7FF {
			t.Errorf("Expected %d, got %d", 0xD7FF, codepoint)
		}
	})

	t.Run("UTF8_Error_Handling", func(t *testing.T) {
		// Test error handling for invalid UTF-8 sequences
		invalidUTF8 := []byte{0xFF, 0xFE} // Invalid UTF-8
		bitstring := funbit.NewBitStringFromBytes(invalidUTF8)

		matcher := funbit.NewMatcher()
		var codepoint int
		funbit.UTF8(matcher, &codepoint)

		_, err := funbit.Match(matcher, bitstring)
		if err == nil {
			t.Error("Expected error, got nil")
		}
		if !strings.Contains(err.Error(), "invalid UTF-8") {
			t.Errorf("Expected error to contain 'invalid UTF-8', got %v", err)
		}
	})
}

// TestUTFEncodingEdgeCases tests edge cases and boundary conditions
func TestUTFEncodingEdgeCases(t *testing.T) {

	t.Run("UTF8_Boundary_Codepoints", func(t *testing.T) {
		// Test boundary codepoints for UTF-8 encoding lengths
		testCases := []struct {
			name      string
			codepoint rune
			expected  []byte
		}{
			{"1-byte UTF-8 max", 0x7F, []byte{0x7F}},
			{"2-byte UTF-8 min", 0x80, []byte{0xC2, 0x80}},
			{"2-byte UTF-8 max", 0x7FF, []byte{0xDF, 0xBF}},
			{"3-byte UTF-8 min", 0x800, []byte{0xE0, 0xA0, 0x80}},
			{"3-byte UTF-8 max", 0xFFFF, []byte{0xEF, 0xBF, 0xBF}},
			{"4-byte UTF-8 min", 0x10000, []byte{0xF0, 0x90, 0x80, 0x80}},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				builder := funbit.NewBuilder()
				funbit.AddUTF8(builder, string(tc.codepoint))

				bitstring, err := funbit.Build(builder)
				if err != nil {
					t.Fatalf("Expected no error, got %v", err)
				}

				if !bytesEqual(tc.expected, bitstring.ToBytes()) {
					t.Errorf("Expected %v, got %v", tc.expected, bitstring.ToBytes())
				}

				// Test round-trip: encode then decode
				matcher := funbit.NewMatcher()
				var decoded int
				funbit.UTF8(matcher, &decoded)

				results, err := funbit.Match(matcher, bitstring)
				if err != nil {
					t.Fatalf("Expected no error, got %v", err)
				}
				if len(results) == 0 {
					t.Fatalf("Expected non-empty results")
				}

				if decoded != int(tc.codepoint) {
					t.Errorf("Expected %d, got %d", int(tc.codepoint), decoded)
				}
			})
		}
	})
}

// TestUTFCodepointAPI tests the new codepoint-specific API functions
func TestUTFCodepointAPI(t *testing.T) {

	t.Run("UTF8_Codepoint_Functions", func(t *testing.T) {
		builder := funbit.NewBuilder()

		// Test AddUTF8Codepoint - equivalent to Erlang <<1024/utf8>>
		funbit.AddUTF8Codepoint(builder, 1024)

		bitstring, err := funbit.Build(builder)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		expected := []byte{208, 128} // UTF-8 encoding of codepoint 1024
		if !bytesEqual(expected, bitstring.ToBytes()) {
			t.Errorf("Expected %v, got %v", expected, bitstring.ToBytes())
		}

		// Test round-trip
		matcher := funbit.NewMatcher()
		var decoded int
		funbit.UTF8(matcher, &decoded)

		results, err := funbit.Match(matcher, bitstring)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if len(results) == 0 {
			t.Fatalf("Expected non-empty results")
		}

		if decoded != 1024 {
			t.Errorf("Expected 1024, got %d", decoded)
		}
	})

	t.Run("UTF16_Codepoint_Functions", func(t *testing.T) {
		builder := funbit.NewBuilder()

		// Test AddUTF16Codepoint - equivalent to Erlang <<65/utf16>>
		funbit.AddUTF16Codepoint(builder, 65, funbit.WithEndianness("big"))

		bitstring, err := funbit.Build(builder)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		expected := []byte{0x00, 0x41} // UTF-16 BE encoding of 'A'
		if !bytesEqual(expected, bitstring.ToBytes()) {
			t.Errorf("Expected %v, got %v", expected, bitstring.ToBytes())
		}

		// Test round-trip
		matcher := funbit.NewMatcher()
		var decoded int
		funbit.UTF16(matcher, &decoded, funbit.WithEndianness("big"))

		results, err := funbit.Match(matcher, bitstring)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if len(results) == 0 {
			t.Fatalf("Expected non-empty results")
		}

		if decoded != 65 {
			t.Errorf("Expected 65, got %d", decoded)
		}
	})

	t.Run("UTF32_Codepoint_Functions", func(t *testing.T) {
		builder := funbit.NewBuilder()

		// Test AddUTF32Codepoint - equivalent to Erlang <<0x1F680/utf32>>
		funbit.AddUTF32Codepoint(builder, 0x1F680) // 🚀 rocket emoji

		bitstring, err := funbit.Build(builder)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		expected := []byte{0x00, 0x01, 0xF6, 0x80} // UTF-32 BE encoding
		if !bytesEqual(expected, bitstring.ToBytes()) {
			t.Errorf("Expected %v, got %v", expected, bitstring.ToBytes())
		}

		// Test round-trip
		matcher := funbit.NewMatcher()
		var decoded int
		funbit.UTF32(matcher, &decoded)

		results, err := funbit.Match(matcher, bitstring)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if len(results) == 0 {
			t.Fatalf("Expected non-empty results")
		}

		if decoded != 0x1F680 {
			t.Errorf("Expected %d, got %d", 0x1F680, decoded)
		}
	})
}

// TestUTFInvalidCodepoints tests validation of invalid Unicode codepoints
func TestUTFInvalidCodepoints(t *testing.T) {

	t.Run("Invalid_Surrogate_Pairs", func(t *testing.T) {
		// Test surrogate pair range 0xD800-0xDFFF (invalid for UTF-8/32)
		invalidCodepoints := []int{0xD800, 0xDBFF, 0xDC00, 0xDFFF}

		for _, cp := range invalidCodepoints {
			t.Run(fmt.Sprintf("Codepoint_0x%X", cp), func(t *testing.T) {
				// UTF-8
				builder1 := funbit.NewBuilder()
				funbit.AddUTF8Codepoint(builder1, cp)
				_, err := funbit.Build(builder1)
				if err == nil {
					t.Error("Expected error, got nil")
				}
				if !strings.Contains(err.Error(), "invalid Unicode codepoint") {
					t.Errorf("Expected error to contain 'invalid Unicode codepoint', got %v", err)
				}

				// UTF-16
				builder2 := funbit.NewBuilder()
				funbit.AddUTF16Codepoint(builder2, cp)
				_, err = funbit.Build(builder2)
				if err == nil {
					t.Error("Expected error, got nil")
				}
				if !strings.Contains(err.Error(), "invalid Unicode codepoint") {
					t.Errorf("Expected error to contain 'invalid Unicode codepoint', got %v", err)
				}

				// UTF-32
				builder3 := funbit.NewBuilder()
				funbit.AddUTF32Codepoint(builder3, cp)
				_, err = funbit.Build(builder3)
				if err == nil {
					t.Error("Expected error, got nil")
				}
				if !strings.Contains(err.Error(), "invalid Unicode codepoint") {
					t.Errorf("Expected error to contain 'invalid Unicode codepoint', got %v", err)
				}
			})
		}
	})

	t.Run("Codepoint_Too_Large", func(t *testing.T) {
		// Test codepoint larger than maximum Unicode (0x10FFFF)
		builder1 := funbit.NewBuilder()
		funbit.AddUTF8Codepoint(builder1, 0x110000)
		_, err := funbit.Build(builder1)
		if err == nil {
			t.Error("Expected error, got nil")
		}
		if !strings.Contains(err.Error(), "invalid Unicode codepoint") {
			t.Errorf("Expected error to contain 'invalid Unicode codepoint', got %v", err)
		}

		builder2 := funbit.NewBuilder()
		funbit.AddUTF16Codepoint(builder2, 0x110000)
		_, err = funbit.Build(builder2)
		if err == nil {
			t.Error("Expected error, got nil")
		}
		if !strings.Contains(err.Error(), "invalid Unicode codepoint") {
			t.Errorf("Expected error to contain 'invalid Unicode codepoint', got %v", err)
		}

		builder3 := funbit.NewBuilder()
		funbit.AddUTF32Codepoint(builder3, 0x110000)
		_, err = funbit.Build(builder3)
		if err == nil {
			t.Error("Expected error, got nil")
		}
		if !strings.Contains(err.Error(), "invalid Unicode codepoint") {
			t.Errorf("Expected error to contain 'invalid Unicode codepoint', got %v", err)
		}
	})

	t.Run("Negative_Codepoint", func(t *testing.T) {
		// Test negative codepoint
		builder1 := funbit.NewBuilder()
		funbit.AddUTF8Codepoint(builder1, -1)
		_, err := funbit.Build(builder1)
		if err == nil {
			t.Error("Expected error, got nil")
		}
		if !strings.Contains(err.Error(), "invalid Unicode codepoint") {
			t.Errorf("Expected error to contain 'invalid Unicode codepoint', got %v", err)
		}

		builder2 := funbit.NewBuilder()
		funbit.AddUTF16Codepoint(builder2, -1)
		_, err = funbit.Build(builder2)
		if err == nil {
			t.Error("Expected error, got nil")
		}
		if !strings.Contains(err.Error(), "invalid Unicode codepoint") {
			t.Errorf("Expected error to contain 'invalid Unicode codepoint', got %v", err)
		}

		builder3 := funbit.NewBuilder()
		funbit.AddUTF32Codepoint(builder3, -1)
		_, err = funbit.Build(builder3)
		if err == nil {
			t.Error("Expected error, got nil")
		}
		if !strings.Contains(err.Error(), "invalid Unicode codepoint") {
			t.Errorf("Expected error to contain 'invalid Unicode codepoint', got %v", err)
		}
	})
}

// TestUTF16SurrogatePairs tests UTF-16 surrogate pair handling
func TestUTF16SurrogatePairs(t *testing.T) {

	t.Run("UTF16_Surrogate_Pair_Encoding", func(t *testing.T) {
		// Test characters outside BMP that require surrogate pairs
		testCases := []struct {
			name      string
			char      string
			codepoint int
		}{
			{"Musical Symbol", "𝄞", 0x1D11E}, // U+1D11E
			{"Rocket Emoji", "🚀", 0x1F680},   // U+1F680
			{"Star Emoji", "🌟", 0x1F31F},     // U+1F31F
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Test string encoding
				builder := funbit.NewBuilder()
				funbit.AddUTF16(builder, tc.char, funbit.WithEndianness("big"))

				bitstring, err := funbit.Build(builder)
				if err != nil {
					t.Fatalf("Expected no error, got %v", err)
				}

				// Should be 4 bytes (surrogate pair)
				if bitstring.Length() != uint(32) {
					t.Errorf("Expected length 32, got %d", bitstring.Length())
				}

				// Test codepoint encoding
				builder2 := funbit.NewBuilder()
				funbit.AddUTF16Codepoint(builder2, tc.codepoint, funbit.WithEndianness("big"))

				bitstring2, err := funbit.Build(builder2)
				if err != nil {
					t.Fatalf("Expected no error, got %v", err)
				}

				// Both should produce the same result
				if !bytesEqual(bitstring.ToBytes(), bitstring2.ToBytes()) {
					t.Errorf("Expected %v, got %v", bitstring.ToBytes(), bitstring2.ToBytes())
				}

				// Test round-trip decoding
				matcher := funbit.NewMatcher()
				var decoded int
				funbit.UTF16(matcher, &decoded, funbit.WithEndianness("big"))

				results, err := funbit.Match(matcher, bitstring)
				if err != nil {
					t.Fatalf("Expected no error, got %v", err)
				}
				if len(results) == 0 {
					t.Fatalf("Expected non-empty results")
				}

				if decoded != tc.codepoint {
					t.Errorf("Expected %d, got %d", tc.codepoint, decoded)
				}
			})
		}
	})
}
