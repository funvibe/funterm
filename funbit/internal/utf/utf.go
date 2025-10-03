package utf

import (
	"errors"
	"unicode/utf8"
)

// Unicode constants
const (
	// Unicode valid ranges (excluding surrogate pairs)
	MinUnicodeCodePoint = 0x0000
	MaxUnicodeCodePoint = 0x10FFFF

	// Surrogate code points (invalid)
	SurrogateHighStart = 0xD800
	SurrogateHighEnd   = 0xDBFF
	SurrogateLowStart  = 0xDC00
	SurrogateLowEnd    = 0xDFFF

	// Valid Unicode ranges
	ValidRange1Start = 0x0000
	ValidRange1End   = 0xD7FF
	ValidRange2Start = 0xE000
	ValidRange2End   = 0x10FFFF
)

// Errors
var (
	ErrInvalidUnicodeCodePoint = errors.New("invalid unicode code point")
	ErrInvalidUTF8Sequence     = errors.New("invalid utf-8 sequence")
	ErrInvalidUTF16Sequence    = errors.New("invalid utf-16 sequence")
	ErrInvalidUTF32Sequence    = errors.New("invalid utf-32 sequence")
	ErrSizeSpecifiedForUTF     = errors.New("size cannot be specified for utf types")
)

// ValidateUnicodeCodePoint validates if a code point is valid Unicode
func ValidateUnicodeCodePoint(codePoint int) error {
	if codePoint < MinUnicodeCodePoint || codePoint > MaxUnicodeCodePoint {
		return ErrInvalidUnicodeCodePoint
	}

	// Check if it's in surrogate range (invalid)
	if (codePoint >= SurrogateHighStart && codePoint <= SurrogateHighEnd) ||
		(codePoint >= SurrogateLowStart && codePoint <= SurrogateLowEnd) {
		return ErrInvalidUnicodeCodePoint
	}

	return nil
}

// IsValidUnicodeCodePoint checks if a code point is valid Unicode
func IsValidUnicodeCodePoint(codePoint int) bool {
	return ValidateUnicodeCodePoint(codePoint) == nil
}

// UTF8Encoder handles UTF-8 encoding and decoding
type UTF8Encoder struct{}

// NewUTF8Encoder creates a new UTF-8 encoder
func NewUTF8Encoder() *UTF8Encoder {
	return &UTF8Encoder{}
}

// Encode encodes a Unicode code point to UTF-8
func (e *UTF8Encoder) Encode(codePoint int) ([]byte, error) {
	if err := ValidateUnicodeCodePoint(codePoint); err != nil {
		return nil, err
	}

	buf := make([]byte, 4)
	n := utf8.EncodeRune(buf, rune(codePoint))
	return buf[:n], nil
}

// Decode decodes UTF-8 bytes to a Unicode code point
func (e *UTF8Encoder) Decode(data []byte) (int, error) {
	if len(data) == 0 {
		return 0, ErrInvalidUTF8Sequence
	}

	r, _ := utf8.DecodeRune(data)
	if r == utf8.RuneError {
		return 0, ErrInvalidUTF8Sequence
	}

	// Validate the decoded rune
	if err := ValidateUnicodeCodePoint(int(r)); err != nil {
		return 0, err
	}

	return int(r), nil
}

// RequiredBytes returns the number of bytes needed to encode a code point in UTF-8
func (e *UTF8Encoder) RequiredBytes(codePoint int) int {
	if !IsValidUnicodeCodePoint(codePoint) {
		return 0
	}

	if codePoint <= 0x7F {
		return 1
	} else if codePoint <= 0x7FF {
		return 2
	} else if codePoint <= 0xFFFF {
		return 3
	} else {
		return 4
	}
}

// UTF16Encoder handles UTF-16 encoding and decoding
type UTF16Encoder struct{}

// NewUTF16Encoder creates a new UTF-16 encoder
func NewUTF16Encoder() *UTF16Encoder {
	return &UTF16Encoder{}
}

// Encode encodes a Unicode code point to UTF-16
func (e *UTF16Encoder) Encode(codePoint int, endianness string) ([]byte, error) {
	if err := ValidateUnicodeCodePoint(codePoint); err != nil {
		return nil, err
	}

	if codePoint <= 0xFFFF {
		// BMP character - single 16-bit code unit
		return e.encodeUint16(uint16(codePoint), endianness), nil
	} else {
		// Supplementary plane character - surrogate pair
		codePoint -= 0x10000
		highSurrogate := uint16((codePoint >> 10) + 0xD800)
		lowSurrogate := uint16((codePoint & 0x3FF) + 0xDC00)

		result := make([]byte, 4)
		copy(result[0:2], e.encodeUint16(highSurrogate, endianness))
		copy(result[2:4], e.encodeUint16(lowSurrogate, endianness))
		return result, nil
	}
}

// Decode decodes UTF-16 bytes to a Unicode code point
func (e *UTF16Encoder) Decode(data []byte, endianness string) (int, error) {
	if len(data) < 2 {
		return 0, ErrInvalidUTF16Sequence
	}

	first := e.decodeUint16(data[0:2], endianness)

	if first < 0xD800 || first > 0xDFFF {
		// BMP character
		return int(first), nil
	}

	if len(data) < 4 {
		return 0, ErrInvalidUTF16Sequence
	}

	// Check for valid surrogate pair
	if first >= 0xD800 && first <= 0xDBFF {
		second := e.decodeUint16(data[2:4], endianness)
		if second >= 0xDC00 && second <= 0xDFFF {
			// Valid surrogate pair - calculate the code point
			codePoint := int((first-0xD800)<<10) + int(second-0xDC00) + 0x10000

			// Validate the resulting code point is within valid Unicode range
			if codePoint < 0 || codePoint > 0x10FFFF {
				return 0, ErrInvalidUTF16Sequence
			}

			if err := ValidateUnicodeCodePoint(codePoint); err != nil {
				return 0, err
			}
			return codePoint, nil
		}
	}

	return 0, ErrInvalidUTF16Sequence
}

// RequiredBytes returns the number of bytes needed to encode a code point in UTF-16
func (e *UTF16Encoder) RequiredBytes(codePoint int) int {
	if !IsValidUnicodeCodePoint(codePoint) {
		return 0
	}

	if codePoint <= 0xFFFF {
		return 2
	} else {
		return 4
	}
}

// encodeUint16 encodes a 16-bit value with specified endianness
func (e *UTF16Encoder) encodeUint16(value uint16, endianness string) []byte {
	result := make([]byte, 2)
	if endianness == "little" {
		result[0] = byte(value & 0xFF)
		result[1] = byte(value >> 8)
	} else {
		// big-endian (default)
		result[0] = byte(value >> 8)
		result[1] = byte(value & 0xFF)
	}
	return result
}

// decodeUint16 decodes a 16-bit value with specified endianness
func (e *UTF16Encoder) decodeUint16(data []byte, endianness string) uint16 {
	if endianness == "little" {
		return uint16(data[0]) | (uint16(data[1]) << 8)
	} else {
		// big-endian (default)
		return (uint16(data[0]) << 8) | uint16(data[1])
	}
}

// UTF32Encoder handles UTF-32 encoding and decoding
type UTF32Encoder struct{}

// NewUTF32Encoder creates a new UTF-32 encoder
func NewUTF32Encoder() *UTF32Encoder {
	return &UTF32Encoder{}
}

// Encode encodes a Unicode code point to UTF-32
func (e *UTF32Encoder) Encode(codePoint int, endianness string) ([]byte, error) {
	if err := ValidateUnicodeCodePoint(codePoint); err != nil {
		return nil, err
	}

	return e.encodeUint32(uint32(codePoint), endianness), nil
}

// Decode decodes UTF-32 bytes to a Unicode code point
func (e *UTF32Encoder) Decode(data []byte, endianness string) (int, error) {
	if len(data) < 4 {
		return 0, ErrInvalidUTF32Sequence
	}

	codePoint := int(e.decodeUint32(data[0:4], endianness))
	if err := ValidateUnicodeCodePoint(codePoint); err != nil {
		return 0, err
	}

	return codePoint, nil
}

// RequiredBytes returns the number of bytes needed to encode a code point in UTF-32
func (e *UTF32Encoder) RequiredBytes(codePoint int) int {
	if !IsValidUnicodeCodePoint(codePoint) {
		return 0
	}
	return 4
}

// encodeUint32 encodes a 32-bit value with specified endianness
func (e *UTF32Encoder) encodeUint32(value uint32, endianness string) []byte {
	result := make([]byte, 4)
	if endianness == "little" {
		result[0] = byte(value & 0xFF)
		result[1] = byte((value >> 8) & 0xFF)
		result[2] = byte((value >> 16) & 0xFF)
		result[3] = byte((value >> 24) & 0xFF)
	} else {
		// big-endian (default)
		result[0] = byte((value >> 24) & 0xFF)
		result[1] = byte((value >> 16) & 0xFF)
		result[2] = byte((value >> 8) & 0xFF)
		result[3] = byte(value & 0xFF)
	}
	return result
}

// decodeUint32 decodes a 32-bit value with specified endianness
func (e *UTF32Encoder) decodeUint32(data []byte, endianness string) uint32 {
	if endianness == "little" {
		return uint32(data[0]) | (uint32(data[1]) << 8) | (uint32(data[2]) << 16) | (uint32(data[3]) << 24)
	} else {
		// big-endian (default)
		return (uint32(data[0]) << 24) | (uint32(data[1]) << 16) | (uint32(data[2]) << 8) | uint32(data[3])
	}
}

// GetEncoder returns the appropriate encoder for the given UTF type
func GetEncoder(utfType string) interface{} {
	switch utfType {
	case "utf8":
		return NewUTF8Encoder()
	case "utf16":
		return NewUTF16Encoder()
	case "utf32":
		return NewUTF32Encoder()
	default:
		return nil
	}
}
