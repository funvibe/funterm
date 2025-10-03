package funbit

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	bitstringpkg "github.com/funvibe/funbit/internal/bitstring"
	"github.com/funvibe/funbit/internal/builder"
	"github.com/funvibe/funbit/internal/endianness"
	"github.com/funvibe/funbit/internal/matcher"
	"github.com/funvibe/funbit/internal/utf"
)

// BitString represents a sequence of bits
type BitString = bitstringpkg.BitString

// Segment represents a segment in bitstring operations
type Segment = bitstringpkg.Segment

// SegmentOption represents an option for configuring segments
type SegmentOption = bitstringpkg.SegmentOption

// SegmentResult represents the result of a matching operation
type SegmentResult = bitstringpkg.SegmentResult

// Builder represents a bitstring builder
type Builder = builder.Builder

// Matcher represents a pattern matcher
type Matcher = matcher.Matcher

// BitStringError represents a bitstring error
type BitStringError = bitstringpkg.BitStringError

// Type constants
const (
	TypeBitstring = bitstringpkg.TypeBitstring
	TypeBinary    = bitstringpkg.TypeBinary
	TypeInteger   = bitstringpkg.TypeInteger
	TypeFloat     = bitstringpkg.TypeFloat
	TypeUTF       = bitstringpkg.TypeUTF
	TypeUTF8      = bitstringpkg.TypeUTF8
	TypeUTF16     = bitstringpkg.TypeUTF16
	TypeUTF32     = bitstringpkg.TypeUTF32
)

// Endianness constants
const (
	EndiannessBig    = bitstringpkg.EndiannessBig
	EndiannessLittle = bitstringpkg.EndiannessLittle
	EndiannessNative = bitstringpkg.EndiannessNative
)

// Error code constants
const (
	ErrOverflow                = bitstringpkg.CodeOverflow
	ErrSignedOverflow          = bitstringpkg.CodeSignedOverflow
	ErrInsufficientBits        = bitstringpkg.CodeInsufficientBits
	ErrInvalidSize             = bitstringpkg.CodeInvalidSize
	ErrInvalidType             = bitstringpkg.CodeInvalidType
	ErrInvalidEndianness       = bitstringpkg.CodeInvalidEndianness
	ErrBinarySizeRequired      = bitstringpkg.CodeBinarySizeRequired
	ErrBinarySizeMismatch      = bitstringpkg.CodeBinarySizeMismatch
	ErrInvalidBinaryData       = bitstringpkg.CodeInvalidBinaryData
	ErrInvalidBitstringData    = bitstringpkg.CodeInvalidBitstringData
	ErrUTFSizeSpecified        = bitstringpkg.CodeUTFSizeSpecified
	ErrInvalidUnicodeCodepoint = bitstringpkg.CodeInvalidUnicodeCodepoint
	ErrTypeMismatch            = bitstringpkg.CodeTypeMismatch
	ErrInvalidSegment          = bitstringpkg.CodeInvalidSegment
	ErrInvalidUnit             = bitstringpkg.CodeInvalidUnit
	ErrInvalidFloatSize        = bitstringpkg.CodeInvalidFloatSize
	ErrUTFUnitModified         = bitstringpkg.CodeUTFUnitModified
)

// Default size constants
const (
	DefaultSizeInteger = bitstringpkg.DefaultSizeInteger
	DefaultSizeFloat   = bitstringpkg.DefaultSizeFloat
)

// NewBitString creates an empty bitstring
func NewBitString() *BitString {
	return bitstringpkg.NewBitString()
}

// NewBitStringFromBytes creates a bitstring from byte slice
func NewBitStringFromBytes(data []byte) *BitString {
	return bitstringpkg.NewBitStringFromBytes(data)
}

// NewBitStringFromBits creates a bitstring from bits with specific length
func NewBitStringFromBits(data []byte, length uint) *BitString {
	return bitstringpkg.NewBitStringFromBits(data, length)
}

// NewBuilder creates a new builder instance
func NewBuilder() *Builder {
	return builder.NewBuilder()
}

// NewMatcher creates a new matcher instance
func NewMatcher() *Matcher {
	return matcher.NewMatcher()
}

// Builder functions

// AddInteger adds an integer to the builder
func AddInteger(b *Builder, value interface{}, options ...SegmentOption) {
	b.AddInteger(value, options...)
}

// AddFloat adds a float to the builder
func AddFloat(b *Builder, value interface{}, options ...SegmentOption) {
	b.AddFloat(value, options...)
}

// AddBinary adds binary data to the builder
func AddBinary(b *Builder, value []byte, options ...SegmentOption) {
	b.AddBinary(value, options...)
}

// AddBitstring adds a bitstring to the builder
func AddBitstring(b *Builder, value *BitString, options ...SegmentOption) {
	b.AddBitstring(value, options...)
}

// AddUTF adds UTF-encoded string data to the builder
func AddUTF(b *Builder, value string, options ...SegmentOption) {
	segment := bitstringpkg.NewSegment(value, options...)
	// Set default UTF type if not specified in options
	if segment.Type == "" {
		segment.Type = bitstringpkg.TypeUTF
	}
	b.AddSegment(*segment)
}

// AddUTF8 adds UTF-8 encoded string data to the builder
func AddUTF8(b *Builder, value string, options ...SegmentOption) {
	// For UTF types, we need to create the segment with the correct type first
	allOptions := append([]SegmentOption{bitstringpkg.WithType(bitstringpkg.TypeUTF8)}, options...)
	segment := bitstringpkg.NewSegment(value, allOptions...)
	b.AddSegment(*segment)
}

// AddUTF16 adds UTF-16 encoded string data to the builder
func AddUTF16(b *Builder, value string, options ...SegmentOption) {
	// For UTF types, we need to create the segment with the correct type first
	allOptions := append([]SegmentOption{bitstringpkg.WithType(bitstringpkg.TypeUTF16)}, options...)
	segment := bitstringpkg.NewSegment(value, allOptions...)
	b.AddSegment(*segment)
}

// AddUTF32 adds UTF-32 encoded string data to the builder
func AddUTF32(b *Builder, value string, options ...SegmentOption) {
	// For UTF types, we need to create the segment with the correct type first
	allOptions := append([]SegmentOption{bitstringpkg.WithType(bitstringpkg.TypeUTF32)}, options...)
	segment := bitstringpkg.NewSegment(value, allOptions...)
	b.AddSegment(*segment)
}

// AddUTF8Codepoint adds a single UTF-8 encoded codepoint to the builder
// This is equivalent to Erlang's <<Codepoint/utf8>>
func AddUTF8Codepoint(b *Builder, codepoint int, options ...SegmentOption) {
	// Validate codepoint range according to Unicode standard
	if !IsValidUnicodeCodePoint(codepoint) {
		// Set error in builder (will be returned by Build())
		b.SetError(NewBitStringError(ErrInvalidUnicodeCodepoint,
			fmt.Sprintf("invalid Unicode codepoint: 0x%X", codepoint)))
		return
	}

	allOptions := append([]SegmentOption{bitstringpkg.WithType(bitstringpkg.TypeUTF8)}, options...)
	segment := bitstringpkg.NewSegment(string(rune(codepoint)), allOptions...)
	b.AddSegment(*segment)
}

// AddUTF16Codepoint adds a single UTF-16 encoded codepoint to the builder
// This is equivalent to Erlang's <<Codepoint/utf16>>
func AddUTF16Codepoint(b *Builder, codepoint int, options ...SegmentOption) {
	// Validate codepoint range according to Unicode standard
	if !IsValidUnicodeCodePoint(codepoint) {
		// Set error in builder (will be returned by Build())
		b.SetError(NewBitStringError(ErrInvalidUnicodeCodepoint,
			fmt.Sprintf("invalid Unicode codepoint: 0x%X", codepoint)))
		return
	}

	allOptions := append([]SegmentOption{bitstringpkg.WithType(bitstringpkg.TypeUTF16)}, options...)
	segment := bitstringpkg.NewSegment(string(rune(codepoint)), allOptions...)
	b.AddSegment(*segment)
}

// AddUTF32Codepoint adds a single UTF-32 encoded codepoint to the builder
// This is equivalent to Erlang's <<Codepoint/utf32>>
func AddUTF32Codepoint(b *Builder, codepoint int, options ...SegmentOption) {
	// Validate codepoint range according to Unicode standard
	if !IsValidUnicodeCodePoint(codepoint) {
		// Set error in builder (will be returned by Build())
		b.SetError(NewBitStringError(ErrInvalidUnicodeCodepoint,
			fmt.Sprintf("invalid Unicode codepoint: 0x%X", codepoint)))
		return
	}

	allOptions := append([]SegmentOption{bitstringpkg.WithType(bitstringpkg.TypeUTF32)}, options...)
	segment := bitstringpkg.NewSegment(string(rune(codepoint)), allOptions...)
	b.AddSegment(*segment)
}

// Build builds the bitstring from the builder
func Build(b *Builder) (*BitString, error) {
	return b.Build()
}

// Matcher functions

// Integer adds an integer segment to the matcher
func Integer(m *Matcher, variable interface{}, options ...SegmentOption) {
	m.Integer(variable, options...)
}

// Float adds a float segment to the matcher
func Float(m *Matcher, variable interface{}, options ...SegmentOption) {
	m.Float(variable, options...)
}

// Binary adds a binary segment to the matcher
func Binary(m *Matcher, variable interface{}, options ...SegmentOption) {
	m.Binary(variable, options...)
}

// Bitstring adds a bitstring segment to the matcher
func Bitstring(m *Matcher, variable interface{}, options ...SegmentOption) {
	m.Bitstring(variable, options...)
}

// UTF adds a UTF segment to the matcher
func UTF(m *Matcher, variable interface{}, options ...SegmentOption) {
	m.UTF(variable, options...)
}

// UTF8 adds a UTF-8 segment to the matcher
func UTF8(m *Matcher, variable interface{}, options ...SegmentOption) {
	// Combine the UTF8 type with other options
	allOptions := append([]SegmentOption{bitstringpkg.WithType(bitstringpkg.TypeUTF8)}, options...)
	m.UTF(variable, allOptions...)
}

// UTF16 adds a UTF-16 segment to the matcher
func UTF16(m *Matcher, variable interface{}, options ...SegmentOption) {
	// Combine the UTF16 type with other options
	allOptions := append([]SegmentOption{bitstringpkg.WithType(bitstringpkg.TypeUTF16)}, options...)
	m.UTF(variable, allOptions...)
}

// UTF32 adds a UTF-32 segment to the matcher
func UTF32(m *Matcher, variable interface{}, options ...SegmentOption) {
	// Combine the UTF32 type with other options
	allOptions := append([]SegmentOption{bitstringpkg.WithType(bitstringpkg.TypeUTF32)}, options...)
	m.UTF(variable, allOptions...)
}

// RestBinary adds a rest binary segment to the matcher
func RestBinary(m *Matcher, variable interface{}) {
	m.RestBinary(variable)
}

// RestBitstring adds a rest bitstring segment to the matcher
func RestBitstring(m *Matcher, variable interface{}) {
	m.RestBitstring(variable)
}

// RegisterVariable registers a variable for dynamic size expressions
func RegisterVariable(m *Matcher, name string, variable interface{}) {
	m.RegisterVariable(name, variable)
}

// Match performs the pattern matching
func Match(m *Matcher, bs *BitString) ([]SegmentResult, error) {
	return m.Match(bs)
}

// Segment options

// WithSize sets the size for a segment
func WithSize(size uint) SegmentOption {
	return bitstringpkg.WithSize(size)
}

// WithUnit sets the unit for a segment
func WithUnit(unit uint) SegmentOption {
	return bitstringpkg.WithUnit(unit)
}

// WithSigned sets the signed flag for a segment
func WithSigned(signed bool) SegmentOption {
	return bitstringpkg.WithSigned(signed)
}

// WithEndianness sets the endianness for a segment
func WithEndianness(endianness string) SegmentOption {
	return bitstringpkg.WithEndianness(endianness)
}

// WithDynamicSizeExpression sets a dynamic size expression for a segment
func WithDynamicSizeExpression(expression string) SegmentOption {
	return bitstringpkg.WithDynamicSizeExpression(expression)
}

// WithType sets the type for a segment
func WithType(typeStr string) SegmentOption {
	return bitstringpkg.WithType(typeStr)
}

// Utility functions

// ToHexDump converts a bitstring to hex dump format
func ToHexDump(bs *BitString) string {
	if bs == nil || bs.IsEmpty() {
		return ""
	}
	data := bs.ToBytes()
	result := ""
	for i, b := range data {
		if i > 0 {
			result += " "
		}
		result += fmt.Sprintf("%02X", b)
	}
	return result
}

// ToErlangFormat converts a bitstring to Erlang format
func ToErlangFormat(bs *BitString) string {
	if bs == nil || bs.IsEmpty() {
		return "<<>>"
	}
	data := bs.ToBytes()
	if bs.IsBinary() {
		// Byte-aligned data
		result := "<<"
		for i, b := range data {
			if i > 0 {
				result += ","
			}
			result += fmt.Sprintf("%d", b)
		}
		result += ">>"
		return result
	} else {
		// Bit-aligned data
		return fmt.Sprintf("<<%v:%d/binary>>", data, bs.Length())
	}
}

// ToFunbitFormat converts a bitstring to Funterm format with smart display
func ToFunbitFormat(bs *BitString) string {
	if bs == nil || bs.IsEmpty() {
		return "<<>>"
	}

	data := bs.ToBytes()

	// Try to decode as UTF-8 string
	if utf8.Valid(data) {
		str := string(data)
		chars := make([]string, 0, len(str))
		for _, r := range str {
			if unicode.IsPrint(r) || unicode.In(r, unicode.Cyrillic) {
				chars = append(chars, string(r))
			} else {
				chars = append(chars, fmt.Sprintf("%d", r))
			}
		}
		return strings.Join(chars, "")
	}

	// Fallback to numeric representation
	result := ""
	for i, b := range data {
		if i > 0 {
			result += " "
		}
		result += fmt.Sprintf("%d", b)
	}
	return result
}

// formatByteSmart formats a byte intelligently:
// - Printable ASCII characters are shown as characters
// - Non-printable bytes are shown as numbers
func formatByteSmart(b byte) string {
	if b >= 32 && b <= 126 {
		// Printable ASCII
		return fmt.Sprintf("'%c'", b)
	}
	return fmt.Sprintf("%d", b)
}

// ToBinaryString converts a bitstring to binary string representation
func ToBinaryString(bs *BitString) string {
	if bs == nil || bs.IsEmpty() {
		return ""
	}
	data := bs.ToBytes()
	result := ""
	for _, b := range data {
		result += fmt.Sprintf("%08b ", b)
	}
	return result[:len(result)-1] // Remove trailing space
}

// CountBits counts the number of set bits in data
func CountBits(data []byte) uint {
	count := uint(0)
	for _, b := range data {
		for i := 0; i < 8; i++ {
			if (b & (1 << i)) != 0 {
				count++
			}
		}
	}
	return count
}

// GetBitValue gets the value of a specific bit
func GetBitValue(data []byte, bitIndex uint) (bool, error) {
	if bitIndex >= uint(len(data)*8) {
		return false, fmt.Errorf("bit index %d out of range", bitIndex)
	}
	byteIndex := bitIndex / 8
	bitInByte := bitIndex % 8
	mask := byte(1 << (7 - bitInByte))
	return (data[byteIndex] & mask) != 0, nil
}

// ExtractBits extracts a range of bits from data
func ExtractBits(data []byte, startBit, numBits uint) ([]byte, error) {
	if startBit+numBits > uint(len(data)*8) {
		return nil, fmt.Errorf("bit range out of range")
	}

	result := make([]byte, (numBits+7)/8)
	for i := uint(0); i < numBits; i++ {
		bitValue, err := GetBitValue(data, startBit+i)
		if err != nil {
			return nil, err
		}
		if bitValue {
			byteIndex := i / 8
			bitInByte := i % 8
			result[byteIndex] |= 1 << (7 - bitInByte)
		}
	}
	return result, nil
}

// IntToBits converts an integer to bits
func IntToBits(value int64, size uint, signed bool) ([]byte, error) {
	if size > 64 {
		return nil, fmt.Errorf("size %d too large", size)
	}

	bytesNeeded := (size + 7) / 8
	result := make([]byte, bytesNeeded)

	for i := uint(0); i < size && i < 64; i++ {
		if (value & (1 << i)) != 0 {
			byteIndex := i / 8
			bitInByte := i % 8
			if byteIndex < uint(len(result)) {
				result[byteIndex] |= 1 << bitInByte
			}
		}
	}

	return result, nil
}

// BitsToInt converts bits to an integer
func BitsToInt(bits []byte, signed bool) (int64, error) {
	var result int64

	for i, b := range bits {
		result |= int64(b) << (uint(i) * 8)
	}

	return result, nil
}

// GetNativeEndianness returns the native endianness
func GetNativeEndianness() string {
	return endianness.GetNativeEndianness()
}

// ConvertEndianness converts data between different endianness
func ConvertEndianness(data []byte, from, to string, size uint) ([]byte, error) {
	return endianness.ConvertEndianness(data, from, to, size)
}

// UTF encoding/decoding functions

// EncodeUTF8 encodes a string to UTF-8 bytes
func EncodeUTF8(text string) ([]byte, error) {
	encoder := utf.NewUTF8Encoder()
	result := []byte{}
	for _, r := range text {
		encoded, err := encoder.Encode(int(r))
		if err != nil {
			return nil, err
		}
		result = append(result, encoded...)
	}
	return result, nil
}

// DecodeUTF8 decodes UTF-8 bytes to a string
func DecodeUTF8(data []byte) (string, error) {
	result := ""
	offset := 0

	for offset < len(data) {
		r, size := utf8.DecodeRune(data[offset:])
		if r == utf8.RuneError {
			return "", utf.ErrInvalidUTF8Sequence
		}
		result += string(r)
		offset += size
	}

	return result, nil
}

// EncodeUTF16 encodes a string to UTF-16 bytes
func EncodeUTF16(text string, endianness string) ([]byte, error) {
	encoder := utf.NewUTF16Encoder()
	result := []byte{}
	for _, r := range text {
		encoded, err := encoder.Encode(int(r), endianness)
		if err != nil {
			return nil, err
		}
		result = append(result, encoded...)
	}
	return result, nil
}

// DecodeUTF16 decodes UTF-16 bytes to a string
func DecodeUTF16(data []byte, endianness string) (string, error) {
	encoder := utf.NewUTF16Encoder()
	result := ""
	offset := 0

	for offset < len(data) {
		if offset+2 > len(data) {
			break
		}

		codePoint, err := encoder.Decode(data[offset:], endianness)
		if err != nil {
			return "", err
		}
		result += string(rune(codePoint))

		// Move offset based on whether this was a surrogate pair
		if codePoint <= 0xFFFF {
			offset += 2
		} else {
			offset += 4
		}
	}

	return result, nil
}

// IsValidUnicodeCodePoint checks if a code point is valid Unicode
func IsValidUnicodeCodePoint(codePoint int) bool {
	return utf.IsValidUnicodeCodePoint(codePoint)
}

// Error handling functions

// NewBitStringError creates a new BitStringError with the given code and message
func NewBitStringError(code, message string) *BitStringError {
	return bitstringpkg.NewBitStringError(code, message)
}

// NewBitStringErrorWithContext creates a new BitStringError with the given code, message, and context
func NewBitStringErrorWithContext(code, message string, context interface{}) *BitStringError {
	return bitstringpkg.NewBitStringErrorWithContext(code, message, context)
}

// Validation functions

// ValidateSegment validates a segment
func ValidateSegment(segment interface{}) error {
	switch s := segment.(type) {
	case *bitstringpkg.Segment:
		return bitstringpkg.ValidateSegment(s)
	default:
		return fmt.Errorf("invalid segment type")
	}
}

// ValidateUnicodeCodePoint validates a Unicode code point
func ValidateUnicodeCodePoint(codePoint int) error {
	return utf.ValidateUnicodeCodePoint(codePoint)
}

// ValidateSize validates the size and unit for a segment
func ValidateSize(size uint, unit uint) error {
	return bitstringpkg.ValidateSize(size, unit)
}

// NewSegment creates a new segment
func NewSegment(value interface{}, options ...SegmentOption) *Segment {
	return bitstringpkg.NewSegment(value, options...)
}
