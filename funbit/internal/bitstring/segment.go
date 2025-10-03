package bitstring

import "fmt"

// This file contains additional segment-related utilities and constants

// Segment types constants
const (
	TypeInteger       = "integer"
	TypeFloat         = "float"
	TypeBinary        = "binary"
	TypeBitstring     = "bitstring"
	TypeUTF           = "utf"   // Generic UTF type
	TypeUTF8          = "utf8"  // UTF-8 specific
	TypeUTF16         = "utf16" // UTF-16 specific
	TypeUTF32         = "utf32" // UTF-32 specific
	TypeRestBinary    = "rest_binary"
	TypeRestBitstring = "rest_bitstring"
)

// Endianness constants
const (
	EndiannessBig    = "big"
	EndiannessLittle = "little"
	EndiannessNative = "native"
)

// Signedness constants
const (
	Signed   = true
	Unsigned = false
)

// Default unit values for different types
const (
	DefaultUnitInteger   = 1
	DefaultUnitFloat     = 1
	DefaultUnitBinary    = 8
	DefaultUnitBitstring = 1
	DefaultUnitUTF       = 1
)

// Default size values for different types (in bits)
const (
	DefaultSizeInteger = 8
	DefaultSizeFloat   = 64
)

// SegmentOption is a function type for configuring segments
type SegmentOption func(*Segment)

// WithSize sets the size for a segment
func WithSize(size uint) SegmentOption {
	return func(s *Segment) {
		s.Size = size
		s.SizeSpecified = true
	}
}

// WithType sets the type for a segment
func WithType(segmentType string) SegmentOption {
	return func(s *Segment) {
		s.Type = segmentType
	}
}

// WithSigned sets the signedness for a segment
func WithSigned(signed bool) SegmentOption {
	return func(s *Segment) {
		s.Signed = signed
	}
}

// WithEndianness sets the endianness for a segment
func WithEndianness(endianness string) SegmentOption {
	return func(s *Segment) {
		s.Endianness = endianness
	}
}

// WithUnit sets the unit for a segment
func WithUnit(unit uint) SegmentOption {
	return func(s *Segment) {
		s.Unit = unit
		s.UnitSpecified = true
	}
}

// WithDynamicSize sets the size for a segment using a variable reference
func WithDynamicSize(sizeVar *uint) SegmentOption {
	return func(s *Segment) {
		s.DynamicSize = sizeVar
		s.IsDynamic = true
		s.SizeSpecified = false // Dynamic size overrides explicit size
	}
}

// WithDynamicSizeExpression sets the size for a segment using an expression
func WithDynamicSizeExpression(expr string) SegmentOption {
	return func(s *Segment) {
		s.DynamicExpr = expr
		s.IsDynamic = true
		s.SizeSpecified = false // Dynamic size overrides explicit size
	}
}

// NewSegment creates a new segment with the given value and options
func NewSegment(value interface{}, options ...SegmentOption) *Segment {
	segment := &Segment{
		Value:      value,
		Type:       TypeInteger,   // default type
		Signed:     Unsigned,      // default signedness
		Endianness: EndiannessBig, // default endianness
		Unit:       0,             // start with 0 to detect if unit was set
		IsDynamic:  false,         // default to static size
	}

	for _, option := range options {
		option(segment)
	}

	// Set default size based on type if not specified
	if !segment.SizeSpecified {
		segment.Size = getDefaultSizeForType(segment.Type)
		// For UTF types, we should not mark size as specified because UTF cannot have size
		if segment.Type == TypeUTF8 || segment.Type == TypeUTF16 || segment.Type == TypeUTF32 {
			segment.SizeSpecified = false
		} else {
			segment.SizeSpecified = true
		}
	}

	return segment
}

// getDefaultSizeForType returns the default size for a given type
func getDefaultSizeForType(segmentType string) uint {
	switch segmentType {
	case TypeInteger:
		return DefaultSizeInteger
	case TypeFloat:
		return DefaultSizeFloat
	case TypeUTF8, TypeUTF16, TypeUTF32:
		return 0 // UTF types should not have default size - they are variable length
	default:
		return 0 // no default size for binary/bitstring types
	}
}

// getDefaultUnitForType returns the default unit for a given type
func getDefaultUnitForType(segmentType string) uint {
	switch segmentType {
	case TypeInteger, TypeFloat, TypeBitstring:
		return DefaultUnitInteger
	case TypeBinary:
		return DefaultUnitBinary
	case TypeUTF, TypeUTF8, TypeUTF16, TypeUTF32:
		return DefaultUnitUTF
	default:
		return DefaultUnitInteger // default for unknown types
	}
}

// ValidateSegment checks if a segment has valid configuration
func ValidateSegment(segment *Segment) error {
	if segment == nil {
		return NewBitStringError(CodeInvalidSegment, "segment cannot be nil")
	}

	// Validate type
	if segment.Type == "" {
		segment.Type = TypeInteger // default to integer
	}

	// Size 0 is allowed in Erlang bit syntax for all types
	// It simply contributes 0 bits to the bitstring
	// No validation needed for size 0

	// Validate unit - check if explicitly set to invalid value
	if segment.UnitSpecified {
		if segment.Unit < 1 || segment.Unit > 256 {
			return NewBitStringError(CodeInvalidUnit, "unit must be between 1 and 256")
		}
	}

	// Validate endianness
	if segment.Endianness == "" {
		segment.Endianness = EndiannessBig // default to big
	}

	if segment.Endianness != EndiannessBig &&
		segment.Endianness != EndiannessLittle &&
		segment.Endianness != EndiannessNative {
		return NewBitStringError(CodeInvalidEndianness, "endianness must be 'big', 'little', or 'native'")
	}

	// Type-specific validations
	switch segment.Type {
	case TypeFloat:
		if segment.SizeSpecified && (segment.Size != 16 && segment.Size != 32 && segment.Size != 64) {
			return NewBitStringError(CodeInvalidFloatSize, "float size must be 16, 32, or 64 bits")
		}
	case TypeUTF8, TypeUTF16, TypeUTF32:
		// First validate Unicode code point
		if segment.Value != nil {
			if codePoint, ok := segment.Value.(int); ok {
				if codePoint < 0 || codePoint > 0x10FFFF {
					return NewBitStringError(CodeInvalidUnicodeCodepoint, "invalid Unicode code point")
				}
			}
		}

		// UTF types cannot have size specified according to Erlang spec
		if segment.SizeSpecified {
			return NewBitStringError(CodeUTFSizeSpecified, "UTF types cannot have size specified")
		}

		// For UTF types, unit can only be set to the default value (1)
		// but only if it was explicitly specified
		if segment.UnitSpecified && segment.Unit != getDefaultUnitForType(segment.Type) {
			return NewBitStringError(CodeUTFUnitModified, "UTF types cannot have unit modified from default value")
		}
	case TypeBinary, TypeBitstring:
		// Binary and bitstring segments must have size specified, unless using dynamic sizing
		if !segment.SizeSpecified {
			// Allow dynamic sizing for matcher (size will be determined during matching)
			return nil
		}

		// Size 0 is allowed for dynamic sizing (will be determined during matching)
		if segment.Size == 0 {
			return nil
		}

		if segment.Value != nil {
			// For binary type, validate that the value is []byte or *[]uint8
			if segment.Type == TypeBinary {
				if _, ok := segment.Value.([]byte); !ok {
					// Try to handle *[]uint8 which is essentially the same as []byte
					if ptr, ok := segment.Value.(*[]uint8); ok {
						// Convert *[]uint8 to []byte
						*ptr = []byte(*ptr)
					} else {
						return NewBitStringErrorWithContext(CodeInvalidBinaryData,
							fmt.Sprintf("binary segment expects []byte, got %T", segment.Value),
							segment.Value)
					}
				}
			}
			// For bitstring type, validate that the value is *BitString
			if segment.Type == TypeBitstring {
				if _, ok := segment.Value.(*BitString); !ok {
					// Also check for **BitString (double pointer)
					if _, ok := segment.Value.(**BitString); ok {
						// Double pointer is valid in some contexts
					} else {
						return NewBitStringErrorWithContext(CodeInvalidBitstringData,
							fmt.Sprintf("bitstring segment expects *BitString, got %T", segment.Value),
							segment.Value)
					}
				}
			}
		}
	}

	// Validate that the type is supported
	switch segment.Type {
	case TypeInteger, TypeFloat, TypeBinary, TypeBitstring, TypeUTF, TypeUTF8, TypeUTF16, TypeUTF32, TypeRestBinary, TypeRestBitstring:
		// Valid types
	default:
		return NewBitStringError(CodeInvalidType, fmt.Sprintf("unsupported segment type: %s", segment.Type))
	}

	return nil
}

// BitStringError represents an error in bitstring operations
type BitStringError struct {
	Code    string
	Message string
	Context interface{}
}

func (e *BitStringError) Error() string {
	if e.Context != nil {
		switch v := e.Context.(type) {
		case string:
			return e.Message + " (context: " + v + ")"
		case int:
			return e.Message + " (context: " + fmt.Sprintf("%d", v) + ")"
		case uint:
			return e.Message + " (context: " + fmt.Sprintf("%d", v) + ")"
		case map[string]interface{}:
			return e.Message + " (context: " + fmt.Sprintf("%v", v) + ")"
		default:
			return e.Message + " (context: " + fmt.Sprintf("%v", v) + ")"
		}
	}
	return e.Message
}

// Error codes for bitstring operations
const (
	// Overflow errors
	CodeOverflow       = "OVERFLOW"
	CodeSignedOverflow = "SIGNED_OVERFLOW"

	// Insufficient data errors
	CodeInsufficientBits = "INSUFFICIENT_BITS"

	// Invalid segment configuration errors
	CodeInvalidSize       = "INVALID_SIZE"
	CodeInvalidType       = "INVALID_TYPE"
	CodeInvalidEndianness = "INVALID_ENDIANNESS"

	// Binary-specific errors
	CodeBinarySizeRequired = "BINARY_SIZE_REQUIRED"
	CodeBinarySizeMismatch = "BINARY_SIZE_MISMATCH"
	CodeInvalidBinaryData  = "INVALID_BINARY_DATA"

	// Bitstring-specific errors
	CodeInvalidBitstringData = "INVALID_BITSTRING_DATA"

	// UTF-specific errors
	CodeUTFSizeSpecified        = "UTF_SIZE_SPECIFIED"
	CodeInvalidUnicodeCodepoint = "INVALID_UNICODE_CODEPOINT"

	// Type mismatch errors
	CodeTypeMismatch = "TYPE_MISMATCH"

	// General validation errors
	CodeInvalidSegment   = "INVALID_SEGMENT"
	CodeInvalidUnit      = "INVALID_UNIT"
	CodeInvalidFloatSize = "INVALID_FLOAT_SIZE"
	CodeUTFUnitModified  = "UTF_UNIT_MODIFIED"
)

// NewBitStringError creates a new BitStringError with the given code and message
func NewBitStringError(code, message string) *BitStringError {
	return &BitStringError{
		Code:    code,
		Message: message,
	}
}

// NewBitStringErrorWithContext creates a new BitStringError with the given code, message, and context
func NewBitStringErrorWithContext(code, message string, context interface{}) *BitStringError {
	return &BitStringError{
		Code:    code,
		Message: message,
		Context: context,
	}
}
