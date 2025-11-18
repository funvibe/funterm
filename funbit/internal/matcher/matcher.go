package matcher

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"reflect"
	"unicode/utf8"

	bitstringpkg "github.com/funvibe/funbit/internal/bitstring"
	"github.com/funvibe/funbit/internal/endianness"
)

// Matcher provides a fluent interface for pattern matching against bitstrings
type Matcher struct {
	pattern   []*bitstringpkg.Segment
	variables map[string]interface{} // Map to store variable names and their pointers
}

// NewMatcher creates a new matcher instance
func NewMatcher() *Matcher {
	return &Matcher{
		pattern:   []*bitstringpkg.Segment{},
		variables: make(map[string]interface{}),
	}
}

// Integer adds an integer segment to the matching pattern
func (m *Matcher) Integer(variable interface{}, options ...bitstringpkg.SegmentOption) *Matcher {
	segment := bitstringpkg.NewSegment(variable, options...)
	segment.Type = bitstringpkg.TypeInteger

	// Set default size if not specified
	if !segment.SizeSpecified {
		segment.Size = bitstringpkg.DefaultSizeInteger
		segment.SizeSpecified = false
	}

	// Set default unit only if not explicitly specified
	if !segment.UnitSpecified {
		segment.Unit = bitstringpkg.DefaultUnitInteger
	}

	m.pattern = append(m.pattern, segment)
	return m
}

// Float adds a float segment to the matching pattern
func (m *Matcher) Float(variable interface{}, options ...bitstringpkg.SegmentOption) *Matcher {
	segment := bitstringpkg.NewSegment(variable, options...)
	segment.Type = bitstringpkg.TypeFloat

	// Set default size if not specified
	if !segment.SizeSpecified {
		segment.Size = bitstringpkg.DefaultSizeFloat
		segment.SizeSpecified = false
	}

	// Set default unit only if not explicitly set
	if segment.Unit == 0 {
		segment.Unit = bitstringpkg.DefaultUnitFloat
	}

	m.pattern = append(m.pattern, segment)
	return m
}

// Binary adds a binary segment to the matching pattern
func (m *Matcher) Binary(variable interface{}, options ...bitstringpkg.SegmentOption) *Matcher {
	// Create segment with binary type from the beginning
	optionsWithBinary := append([]bitstringpkg.SegmentOption{
		bitstringpkg.WithType(bitstringpkg.TypeBinary),
	}, options...)
	segment := bitstringpkg.NewSegment(variable, optionsWithBinary...)

	// Set default unit only if not explicitly specified
	if !segment.UnitSpecified {
		segment.Unit = bitstringpkg.DefaultUnitBinary // Use default unit for binary
	}

	// For binary segments, we need to ensure size is specified for validation
	// But allow dynamic sizing for specific test cases
	if !segment.SizeSpecified {
		// Check if variable is []byte to determine size dynamically
		if data, ok := variable.([]byte); ok {
			segment.Size = uint(len(data))
			segment.SizeSpecified = true
		} else {
			// For non-byte variables, use a reasonable default or mark as dynamic
			segment.Size = 0 // Will be handled as dynamic size in matchBinary
			segment.SizeSpecified = false
		}
	}

	m.pattern = append(m.pattern, segment)
	return m
}

// UTF adds a UTF segment to the matching pattern
func (m *Matcher) UTF(variable interface{}, options ...bitstringpkg.SegmentOption) *Matcher {
	segment := bitstringpkg.NewSegment(variable, options...)

	// Only set TypeUTF if no specific UTF type was already set in options
	if segment.Type == "" || segment.Type == bitstringpkg.TypeUTF {
		segment.Type = bitstringpkg.TypeUTF
	}

	// Set default size if not specified (UTF-8 by default)
	if !segment.SizeSpecified {
		segment.Size = 8 // Default to UTF-8
		segment.SizeSpecified = false
	}

	// Set default unit only if not explicitly set
	if segment.Unit == 0 {
		segment.Unit = bitstringpkg.DefaultUnitUTF
	}

	m.pattern = append(m.pattern, segment)
	return m
}

// Bitstring adds a bitstring segment to the matching pattern
func (m *Matcher) Bitstring(variable interface{}, options ...bitstringpkg.SegmentOption) *Matcher {
	// Create segment with bitstring type from the beginning
	optionsWithBitstring := append([]bitstringpkg.SegmentOption{
		bitstringpkg.WithType(bitstringpkg.TypeBitstring),
	}, options...)
	segment := bitstringpkg.NewSegment(variable, optionsWithBitstring...)

	// Set default unit only if not explicitly specified
	if !segment.UnitSpecified {
		segment.Unit = bitstringpkg.DefaultUnitBitstring // Use default unit for bitstring
	}

	// For bitstring segments, size should be specified for validation
	// But allow dynamic sizing for specific cases
	if !segment.SizeSpecified {
		// For bitstring without explicit size, use a reasonable default
		segment.Size = 0 // Will be handled as dynamic size in matchBitstring
		segment.SizeSpecified = false
	}

	m.pattern = append(m.pattern, segment)
	return m
}

// RegisterVariable registers a variable with a specific name for dynamic size usage
func (m *Matcher) RegisterVariable(name string, variable interface{}) *Matcher {
	m.variables[name] = variable
	return m
}

// RestBinary adds a rest binary segment to the matching pattern (must be byte-aligned)
func (m *Matcher) RestBinary(variable interface{}) *Matcher {
	segment := bitstringpkg.NewSegment(variable)
	segment.Type = bitstringpkg.TypeRestBinary
	m.pattern = append(m.pattern, segment)
	return m
}

// RestBitstring adds a rest bitstring segment to the matching pattern (any bit length)
func (m *Matcher) RestBitstring(variable interface{}) *Matcher {
	segment := bitstringpkg.NewSegment(variable)
	segment.Type = bitstringpkg.TypeRestBitstring
	m.pattern = append(m.pattern, segment)
	return m
}

// Match attempts to match the pattern against the provided bitstring
func (m *Matcher) Match(bitstring *bitstringpkg.BitString) ([]bitstringpkg.SegmentResult, error) {
	if bitstring == nil {
		return nil, bitstringpkg.NewBitStringError(bitstringpkg.CodeInvalidSegment, "bitstring cannot be nil")
	}

	results := make([]bitstringpkg.SegmentResult, len(m.pattern))
	currentOffset := uint(0)
	context := NewDynamicSizeContext()

	// Initialize context with registered variables
	m.initializeContextFromRegisteredVariables(context)

	for i, segment := range m.pattern {
		if err := bitstringpkg.ValidateSegment(segment); err != nil {
			return nil, bitstringpkg.NewBitStringErrorWithContext(bitstringpkg.CodeInvalidSegment,
				fmt.Sprintf("invalid segment %d: %v", i, err), i)
		}

		result, newOffset, err := m.matchSegmentWithContext(segment, bitstring, currentOffset, context, results)
		if err != nil {
			// If the underlying error is BitStringError, preserve its code
			if bitstringErr, ok := err.(*bitstringpkg.BitStringError); ok {
				return nil, bitstringpkg.NewBitStringErrorWithContext(bitstringErr.Code,
					fmt.Sprintf("failed to match segment %d: %v", i, err), i)
			}
			return nil, err
		}

		results[i] = *result
		currentOffset = newOffset

		// Update context with the matched variable value
		m.updateContextWithResult(context, segment, result)
	}

	return results, nil
}

// matchSegment matches a single segment against the bitstring at the given offset
func (m *Matcher) matchSegment(segment *bitstringpkg.Segment, bs *bitstringpkg.BitString, offset uint) (*bitstringpkg.SegmentResult, uint, error) {
	return m.matchSegmentWithContext(segment, bs, offset, NewDynamicSizeContext(), nil)
}

// matchSegmentWithContext matches a single segment with dynamic size context
func (m *Matcher) matchSegmentWithContext(segment *bitstringpkg.Segment, bs *bitstringpkg.BitString, offset uint, context *DynamicSizeContext, previousResults []bitstringpkg.SegmentResult) (*bitstringpkg.SegmentResult, uint, error) {
	// Evaluate dynamic size if needed
	if segment.IsDynamic {
		evaluatedSize, err := m.EvaluateDynamicSize(segment, context)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to evaluate dynamic size: %v", err)
		}
		// Create a copy of the segment with the evaluated size
		segmentCopy := *segment
		segmentCopy.Size = evaluatedSize
		segmentCopy.SizeSpecified = true

		// For binary type, dynamic size is already in bytes, no conversion needed
		// The unit multiplication will be handled in matchBinary

		segment = &segmentCopy
	}

	switch segment.Type {
	case bitstringpkg.TypeInteger:
		return m.matchInteger(segment, bs, offset)
	case bitstringpkg.TypeFloat:
		return m.matchFloat(segment, bs, offset)
	case bitstringpkg.TypeBinary:
		return m.matchBinary(segment, bs, offset)
	case bitstringpkg.TypeBitstring:
		return m.matchBitstring(segment, bs, offset)
	case bitstringpkg.TypeUTF, bitstringpkg.TypeUTF8, bitstringpkg.TypeUTF16, bitstringpkg.TypeUTF32:
		return m.matchUTF(segment, bs, offset)
	case bitstringpkg.TypeRestBinary:
		return m.matchRestBinary(segment, bs, offset)
	case bitstringpkg.TypeRestBitstring:
		return m.matchRestBitstring(segment, bs, offset)
	default:
		return nil, 0, fmt.Errorf("unsupported segment type: %s", segment.Type)
	}
}

// updateContextWithResult updates the dynamic size context with a matched result
func (m *Matcher) updateContextWithResult(context *DynamicSizeContext, segment *bitstringpkg.Segment, result *bitstringpkg.SegmentResult) {
	if !result.Matched {
		return
	}

	// Extract variable name and value
	varName := m.getVariableNameFromSegment(segment)
	if varName == "" {
		return
	}

	// Convert result value to uint
	var value uint
	switch v := result.Value.(type) {
	case int:
		value = uint(v)
	case int8:
		value = uint(v)
	case int16:
		value = uint(v)
	case int32:
		value = uint(v)
	case int64:
		value = uint(v)
	case uint:
		value = v
	case uint8:
		value = uint(v)
	case uint16:
		value = uint(v)
	case uint32:
		value = uint(v)
	case uint64:
		value = uint(v)
	default:
		// Skip non-integer types
		return
	}

	context.AddVariable(varName, value)
}

// getVariableNameFromSegment extracts variable name from segment value
func (m *Matcher) getVariableNameFromSegment(segment *bitstringpkg.Segment) string {
	if segment.Value == nil {
		return ""
	}

	// Look up the variable name in the registered variables map
	for name, variable := range m.variables {
		if variable == segment.Value {
			return name
		}
	}

	// If not found, try to extract from DynamicSize field
	if segment.DynamicSize != nil {
		// Look for the variable pointer in the registered variables
		for name, variable := range m.variables {
			if ptr, ok := variable.(*uint); ok && ptr == segment.DynamicSize {
				return name
			}
		}
	}

	// If still not found, return empty string
	return ""
}

// matchInteger matches an integer segment against the bitstring
func (m *Matcher) matchInteger(segment *bitstringpkg.Segment, bs *bitstringpkg.BitString, offset uint) (*bitstringpkg.SegmentResult, uint, error) {
	// Use default size if not specified
	var size uint
	if !segment.SizeSpecified {
		size = bitstringpkg.DefaultSizeInteger
	} else {
		size = segment.Size
	}

	// Calculate effective size using unit
	effectiveSize := size * segment.Unit

	if effectiveSize == 0 || effectiveSize > 64 {
		return nil, 0, bitstringpkg.NewBitStringErrorWithContext(bitstringpkg.CodeInvalidSize,
			fmt.Sprintf("invalid integer size: %d bits (size=%d, unit=%d)", effectiveSize, size, segment.Unit),
			map[string]interface{}{"effective_size": effectiveSize, "size": size, "unit": segment.Unit})
	}

	// Handle alignment for unit-based segments
	alignedOffset := offset
	if segment.Unit > 1 && segment.Unit%8 == 0 {
		// For byte-aligned units (8, 16, 24, etc.), align to byte boundary
		byteOffset := (offset + 7) / 8 * 8 // Round up to next byte boundary
		if byteOffset <= bs.Length() {
			alignedOffset = byteOffset
		}
	}

	// Check if we have enough bits remaining
	if alignedOffset+effectiveSize > bs.Length() {
		return nil, 0, bitstringpkg.NewBitStringErrorWithContext(bitstringpkg.CodeInsufficientBits,
			fmt.Sprintf("insufficient bits: need %d, have %d", effectiveSize, bs.Length()-alignedOffset),
			map[string]interface{}{"needed": effectiveSize, "available": bs.Length() - alignedOffset})
	}

	// Extract the integer value
	value, err := m.extractInteger(bs, alignedOffset, effectiveSize, segment.Endianness, segment.Signed)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to extract integer: %v", err)
	}

	// Bind the value to the variable
	if err := m.bindValue(segment.Value, value); err != nil {
		return nil, 0, fmt.Errorf("failed to bind value: %v", err)
	}

	// Create remaining bitstring
	var remaining *bitstringpkg.BitString
	if offset+effectiveSize < bs.Length() {
		// Extract remaining bits
		remainingData := bs.ToBytes()
		remainingOffset := (offset + effectiveSize) / 8
		remainingBitOffset := (offset + effectiveSize) % 8

		if remainingBitOffset == 0 {
			// Aligned to byte boundary
			remaining = bitstringpkg.NewBitStringFromBytes(remainingData[remainingOffset:])
		} else {
			// Not aligned - need bit-level extraction
			remaining = m.extractRemainingBits(bs, offset+effectiveSize)
		}
	} else {
		// No remaining bits
		remaining = bitstringpkg.NewBitString()
	}

	result := &bitstringpkg.SegmentResult{
		Value:     value,
		Matched:   true,
		Remaining: remaining,
	}

	return result, alignedOffset + effectiveSize, nil
}

// matchFloat matches a float segment against the bitstring
func (m *Matcher) matchFloat(segment *bitstringpkg.Segment, bs *bitstringpkg.BitString, offset uint) (*bitstringpkg.SegmentResult, uint, error) {
	if !segment.SizeSpecified {
		return nil, 0, bitstringpkg.NewBitStringError(bitstringpkg.CodeInvalidSize, "float segment must have size specified")
	}

	size := segment.Size
	effectiveSize := size * segment.Unit

	if effectiveSize != 16 && effectiveSize != 32 && effectiveSize != 64 {
		return nil, 0, bitstringpkg.NewBitStringErrorWithContext(bitstringpkg.CodeInvalidFloatSize,
			fmt.Sprintf("invalid float size: %d bits (size=%d, unit=%d, must be 16, 32, or 64)", effectiveSize, size, segment.Unit),
			map[string]interface{}{"effective_size": effectiveSize, "size": size, "unit": segment.Unit})
	}

	// Check if we have enough bits remaining
	if offset+effectiveSize > bs.Length() {
		return nil, 0, bitstringpkg.NewBitStringErrorWithContext(bitstringpkg.CodeInsufficientBits,
			fmt.Sprintf("insufficient bits: need %d, have %d", effectiveSize, bs.Length()-offset),
			map[string]interface{}{"needed": effectiveSize, "available": bs.Length() - offset})
	}

	// Extract the float value
	value, err := m.extractFloat(bs, offset, effectiveSize, segment.Endianness)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to extract float: %v", err)
	}

	// Bind the value to the variable
	if err := m.bindFloatValue(segment.Value, value); err != nil {
		return nil, 0, fmt.Errorf("failed to bind float value: %v", err)
	}

	// Create remaining bitstring
	var remaining *bitstringpkg.BitString
	if offset+effectiveSize < bs.Length() {
		remaining = m.extractRemainingBits(bs, offset+effectiveSize)
	} else {
		remaining = bitstringpkg.NewBitString()
	}

	result := &bitstringpkg.SegmentResult{
		Value:     value,
		Matched:   true,
		Remaining: remaining,
	}

	return result, offset + effectiveSize, nil
}

// matchBinary matches a binary segment against the bitstring
func (m *Matcher) matchBinary(segment *bitstringpkg.Segment, bs *bitstringpkg.BitString, offset uint) (*bitstringpkg.SegmentResult, uint, error) {
	// Determine size if not specified
	var size uint
	if !segment.SizeSpecified {
		// For binary without explicit size, use available bytes (dynamic sizing)
		remainingBits := bs.Length() - offset
		size = remainingBits / 8 // Convert bits to bytes
		if size == 0 {
			return nil, 0, bitstringpkg.NewBitStringError(bitstringpkg.CodeInsufficientBits, "no bytes available for binary match")
		}
	} else {
		size = segment.Size
		if size == 0 {
			// If size is explicitly set to 0, use remaining bytes (dynamic sizing)
			remainingBits := bs.Length() - offset
			size = remainingBits / 8 // Convert bits to bytes
			if size == 0 {
				return nil, 0, bitstringpkg.NewBitStringError(bitstringpkg.CodeInsufficientBits, "no bytes available for binary match")
			}
		}
	}

	// For binary type, size is in bytes, convert to bits for extraction
	// Total bits = size * unit
	effectiveSize := size * segment.Unit

	// Check if we have enough bits remaining
	if offset+effectiveSize > bs.Length() {
		return nil, 0, bitstringpkg.NewBitStringErrorWithContext(bitstringpkg.CodeInsufficientBits,
			fmt.Sprintf("insufficient bits: need %d, have %d", effectiveSize, bs.Length()-offset),
			map[string]interface{}{"needed": effectiveSize, "available": bs.Length() - offset})
	}

	// Extract the binary data
	value, err := m.extractBinary(bs, offset, effectiveSize)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to extract binary: %v", err)
	}

	// Bind the value to the variable
	if err := m.bindBinaryValue(segment.Value, value); err != nil {
		return nil, 0, fmt.Errorf("failed to bind binary value: %v", err)
	}

	// Create remaining bitstring
	var remaining *bitstringpkg.BitString
	if offset+effectiveSize < bs.Length() {
		remaining = m.extractRemainingBits(bs, offset+effectiveSize)
	} else {
		remaining = bitstringpkg.NewBitString()
	}

	result := &bitstringpkg.SegmentResult{
		Value:     value,
		Matched:   true,
		Remaining: remaining,
	}

	return result, offset + effectiveSize, nil
}

// matchBitstring matches a bitstring segment against the bitstring
func (m *Matcher) matchBitstring(segment *bitstringpkg.Segment, bs *bitstringpkg.BitString, offset uint) (*bitstringpkg.SegmentResult, uint, error) {
	effectiveSize, err := m.calculateBitstringEffectiveSize(segment, bs, offset)
	if err != nil {
		return nil, 0, err
	}

	value, err := m.extractNestedBitstring(bs, offset, effectiveSize)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to extract nested bitstring: %v", err)
	}

	if err := m.bindBitstringValue(segment.Value, value); err != nil {
		return nil, 0, fmt.Errorf("failed to bind nested bitstring value: %v", err)
	}

	result := m.createBitstringMatchResult(value, bs, offset, effectiveSize)
	return result, offset + effectiveSize, nil
}

// calculateBitstringEffectiveSize calculates the effective size for bitstring matching
func (m *Matcher) calculateBitstringEffectiveSize(segment *bitstringpkg.Segment, bs *bitstringpkg.BitString, offset uint) (uint, error) {
	size, err := m.determineBitstringMatchSize(segment, bs, offset)
	if err != nil {
		return 0, err
	}

	effectiveSize := size * segment.Unit
	if effectiveSize == 0 {
		return 0, bitstringpkg.NewBitStringErrorWithContext(bitstringpkg.CodeInvalidSize,
			fmt.Sprintf("invalid bitstring size: %d bits (size=%d, unit=%d)", effectiveSize, size, segment.Unit),
			map[string]interface{}{"effective_size": effectiveSize, "size": size, "unit": segment.Unit})
	}

	if offset+effectiveSize > bs.Length() {
		return 0, bitstringpkg.NewBitStringErrorWithContext(bitstringpkg.CodeInsufficientBits,
			fmt.Sprintf("insufficient bits: need %d, have %d", effectiveSize, bs.Length()-offset),
			map[string]interface{}{"needed": effectiveSize, "available": bs.Length() - offset})
	}

	return effectiveSize, nil
}

// determineBitstringMatchSize determines the size for bitstring matching
func (m *Matcher) determineBitstringMatchSize(segment *bitstringpkg.Segment, bs *bitstringpkg.BitString, offset uint) (uint, error) {
	if !segment.SizeSpecified || segment.Size == 0 {
		// Use remaining bits (dynamic sizing)
		remainingBits := bs.Length() - offset
		if remainingBits == 0 {
			return 0, bitstringpkg.NewBitStringError(bitstringpkg.CodeInsufficientBits, "no bits available for bitstring match")
		}
		return remainingBits, nil
	}

	return segment.Size, nil
}

// createBitstringMatchResult creates a segment result for bitstring matching
func (m *Matcher) createBitstringMatchResult(value *bitstringpkg.BitString, bs *bitstringpkg.BitString, offset, effectiveSize uint) *bitstringpkg.SegmentResult {
	var remaining *bitstringpkg.BitString
	if offset+effectiveSize < bs.Length() {
		remaining = m.extractRemainingBits(bs, offset+effectiveSize)
	} else {
		remaining = bitstringpkg.NewBitString()
	}

	return &bitstringpkg.SegmentResult{
		Value:     value,
		Matched:   true,
		Remaining: remaining,
	}
}

// matchUTF matches a UTF segment against the bitstring
func (m *Matcher) matchUTF(segment *bitstringpkg.Segment, bs *bitstringpkg.BitString, offset uint) (*bitstringpkg.SegmentResult, uint, error) {
	// Determine UTF encoding type
	var utfType string

	switch segment.Type {
	case bitstringpkg.TypeUTF8:
		utfType = "utf8"
	case bitstringpkg.TypeUTF16:
		utfType = "utf16"
	case bitstringpkg.TypeUTF32:
		utfType = "utf32"
	case bitstringpkg.TypeUTF:
		// Generic UTF, default to UTF-8
		utfType = "utf8"
	default:
		return nil, 0, fmt.Errorf("unsupported UTF type: %s", segment.Type)
	}

	// For UTF types, unit must always be 1 (this is validated in ValidateSegment)
	if segment.Unit != 1 {
		return nil, 0, fmt.Errorf("UTF types must have unit=1, but got unit=%d", segment.Unit)
	}

	// For UTF, we need to extract the encoded data and decode it
	// UTF encoding is variable-length, so we need to parse until we get a valid character
	value, bytesConsumed, err := m.extractUTF(bs, offset, utfType, segment.Endianness)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to extract UTF: %v", err)
	}

	// Bind the value to the variable
	if err := m.bindUTFValue(segment.Value, value); err != nil {
		return nil, 0, fmt.Errorf("failed to bind UTF value: %v", err)
	}

	// Create remaining bitstring
	var remaining *bitstringpkg.BitString
	bitsConsumed := bytesConsumed * 8
	if offset+bitsConsumed < bs.Length() {
		remaining = m.extractRemainingBits(bs, offset+bitsConsumed)
	} else {
		remaining = bitstringpkg.NewBitString()
	}

	result := &bitstringpkg.SegmentResult{
		Value:     value, // The extracted value
		Matched:   true,
		Remaining: remaining,
	}

	return result, offset + bitsConsumed, nil
}

// extractFloat extracts a float value from the bitstring
func (m *Matcher) extractFloat(bs *bitstringpkg.BitString, offset, size uint, endiannessStr string) (float64, error) {
	data := bs.ToBytes()
	byteOffset := offset / 8
	bitOffset := offset % 8

	// For now, only support byte-aligned floats
	if bitOffset != 0 {
		return 0, fmt.Errorf("non-byte-aligned floats not supported yet")
	}

	bytesNeeded := size / 8
	if byteOffset+bytesNeeded > uint(len(data)) {
		return 0, fmt.Errorf("insufficient data for float extraction")
	}

	extractedData := data[byteOffset : byteOffset+bytesNeeded]

	switch size {
	case 16:
		// 16-bit float (half precision) - proper IEEE 754 conversion
		var bits uint16
		switch endiannessStr {
		case bitstringpkg.EndiannessBig, "":
			bits = binary.BigEndian.Uint16(extractedData)
		case bitstringpkg.EndiannessLittle:
			bits = binary.LittleEndian.Uint16(extractedData)
		case bitstringpkg.EndiannessNative:
			if endianness.GetNativeEndianness() == "little" {
				bits = binary.LittleEndian.Uint16(extractedData)
			} else {
				bits = binary.BigEndian.Uint16(extractedData)
			}
		default:
			return 0, fmt.Errorf("unsupported endianness: %s", endiannessStr)
		}

		// Proper IEEE 754 half precision to single precision conversion
		sign16 := (bits >> 15) & 1
		exp16 := (bits >> 10) & 0x1F
		mant16 := bits & 0x3FF

		var sign32, exp32, mant32 uint32
		sign32 = uint32(sign16)

		if exp16 == 0x1F { // Inf or NaN
			exp32 = 0xFF
			if mant16 != 0 {
				mant32 = 1 // NaN
			}
		} else if exp16 == 0 { // zero or denormal
			exp32 = 0
			mant32 = uint32(mant16)
		} else {
			// Convert exponent: float16 bias 15, float32 bias 127
			exp32 = uint32(exp16 - 15 + 127)
			// Convert mantissa: float16 has 10 bits, float32 has 23
			mant32 = uint32(mant16) << 13
		}

		float32Bits := sign32<<31 | exp32<<23 | mant32
		return float64(math.Float32frombits(float32Bits)), nil
	case 32:
		var bits uint32
		switch endiannessStr {
		case bitstringpkg.EndiannessBig, "":
			bits = binary.BigEndian.Uint32(extractedData)
		case bitstringpkg.EndiannessLittle:
			bits = binary.LittleEndian.Uint32(extractedData)
		case bitstringpkg.EndiannessNative:
			if endianness.GetNativeEndianness() == "little" {
				bits = binary.LittleEndian.Uint32(extractedData)
			} else {
				bits = binary.BigEndian.Uint32(extractedData)
			}
		default:
			return 0, fmt.Errorf("unsupported endianness: %s", endiannessStr)
		}
		return float64(math.Float32frombits(bits)), nil
	case 64:
		var bits uint64
		switch endiannessStr {
		case bitstringpkg.EndiannessBig, "":
			bits = binary.BigEndian.Uint64(extractedData)
		case bitstringpkg.EndiannessLittle:
			bits = binary.LittleEndian.Uint64(extractedData)
		case bitstringpkg.EndiannessNative:
			if endianness.GetNativeEndianness() == "little" {
				bits = binary.LittleEndian.Uint64(extractedData)
			} else {
				bits = binary.BigEndian.Uint64(extractedData)
			}
		default:
			return 0, fmt.Errorf("unsupported endianness: %s", endiannessStr)
		}
		return math.Float64frombits(bits), nil
	default:
		return 0, fmt.Errorf("unsupported float size: %d", size)
	}
}

// extractBinary extracts binary data from the bitstring
func (m *Matcher) extractBinary(bs *bitstringpkg.BitString, offset, size uint) ([]byte, error) {
	data := bs.ToBytes()
	byteOffset := offset / 8
	bitOffset := offset % 8

	// Handle bit-level extraction
	if bitOffset != 0 || size%8 != 0 {
		return m.extractBinaryBits(data, offset, size)
	}

	// Handle byte-aligned extraction
	bytesNeeded := size / 8
	if byteOffset+bytesNeeded > uint(len(data)) {
		return nil, fmt.Errorf("insufficient data for binary extraction")
	}

	// Return a copy of the extracted data
	result := make([]byte, bytesNeeded)
	copy(result, data[byteOffset:byteOffset+bytesNeeded])
	return result, nil
}

// extractBinaryBits extracts binary data with proper bit alignment for binary type
func (m *Matcher) extractBinaryBits(data []byte, start, length uint) ([]byte, error) {
	if start >= uint(len(data))*8 {
		return nil, fmt.Errorf("start position %d is beyond data length", start)
	}

	if length == 0 {
		return []byte{}, nil
	}

	if start+length > uint(len(data))*8 {
		return nil, fmt.Errorf("cannot extract %d bits from position %d", length, start)
	}

	// For binary data, we need to extract bits and pack them into bytes
	resultBytes := make([]byte, (length+7)/8)

	for i := uint(0); i < length; i++ {
		currentBitPos := start + i
		bytePos := currentBitPos / 8
		bitInByte := 7 - (currentBitPos % 8) // 0 is MSB

		bit := (data[bytePos] >> bitInByte) & 1
		resultBytePos := i / 8
		bitInResult := 7 - (i % 8)

		if bit != 0 {
			resultBytes[resultBytePos] |= (1 << bitInResult)
		}
	}

	return resultBytes, nil
}

// extractNestedBitstring extracts a nested bitstring from the source bitstring
func (m *Matcher) extractNestedBitstring(bs *bitstringpkg.BitString, offset, size uint) (*bitstringpkg.BitString, error) {
	if err := validateExtractionBounds(bs, offset, size); err != nil {
		return nil, err
	}

	nestedBs := m.extractRemainingBits(bs, offset)

	if nestedBs.Length() == size {
		return nestedBs, nil
	}

	return m.truncateBitstring(nestedBs, size)
}

// validateExtractionBounds validates the bounds for bitstring extraction
func validateExtractionBounds(bs *bitstringpkg.BitString, offset, size uint) error {
	if offset+size > bs.Length() {
		return fmt.Errorf("cannot extract %d bits from position %d", size, offset)
	}
	return nil
}

// truncateBitstring truncates a bitstring to the specified size
func (m *Matcher) truncateBitstring(bs *bitstringpkg.BitString, size uint) (*bitstringpkg.BitString, error) {
	if bs.Length() <= size {
		return bs, nil
	}

	if size == 0 {
		return bitstringpkg.NewBitString(), nil
	}

	nestedData := bs.ToBytes()
	resultData := make([]byte, (size+7)/8)

	err := m.copyBitsWithTruncation(nestedData, resultData, size)
	if err != nil {
		return nil, err
	}

	return bitstringpkg.NewBitStringFromBits(resultData, size), nil
}

// copyBitsWithTruncation copies bits from source to destination with proper truncation
func (m *Matcher) copyBitsWithTruncation(source, dest []byte, totalBits uint) error {
	dataIndex := uint(0)
	resultIndex := uint(0)
	bitsRemaining := totalBits

	for bitsRemaining > 0 && dataIndex < uint(len(source)) {
		bitsToExtract := bitsRemaining
		if bitsToExtract > 8 {
			bitsToExtract = 8
		}

		extractedByte, err := m.extractBitsFromByte(source[dataIndex], bitsToExtract)
		if err != nil {
			return err
		}

		dest[resultIndex] = extractedByte
		dataIndex++
		resultIndex++
		bitsRemaining -= bitsToExtract
	}

	return nil
}

// extractBitsFromByte extracts the specified number of bits from a byte
func (m *Matcher) extractBitsFromByte(byteVal byte, bitsToExtract uint) (byte, error) {
	if bitsToExtract > 8 {
		return 0, fmt.Errorf("cannot extract more than 8 bits from a byte")
	}

	var extractedByte byte
	for i := uint(0); i < bitsToExtract; i++ {
		bit := (byteVal >> (7 - i)) & 1
		extractedByte = (extractedByte << 1) | bit
	}

	// Shift the extracted byte to the left if we didn't use all 8 bits
	if bitsToExtract < 8 {
		extractedByte <<= (8 - bitsToExtract)
	}

	return extractedByte, nil
}

// bindFloatValue binds the extracted float value to the variable
func (m *Matcher) bindFloatValue(variable interface{}, value float64) error {
	if variable == nil {
		return errors.New("variable cannot be nil")
	}

	// Use reflection to set the value
	val := reflect.ValueOf(variable)

	// Check if it's a pointer
	if val.Kind() != reflect.Ptr {
		return errors.New("variable must be a pointer")
	}

	// Dereference the pointer
	val = val.Elem()

	// Check if it's settable
	if !val.CanSet() {
		return errors.New("variable is not settable")
	}

	// Set the value based on the type
	switch val.Kind() {
	case reflect.Float32:
		val.SetFloat(value)
	case reflect.Float64:
		val.SetFloat(value)
	default:
		return fmt.Errorf("unsupported float variable type: %v", val.Kind())
	}

	return nil
}

// bindBinaryValue binds the extracted binary value to the variable
func (m *Matcher) bindBinaryValue(variable interface{}, value []byte) error {
	if variable == nil {
		return errors.New("variable cannot be nil")
	}

	// Use reflection to set the value
	val := reflect.ValueOf(variable)

	// Check if it's a pointer
	if val.Kind() != reflect.Ptr {
		return errors.New("variable must be a pointer")
	}

	// Dereference the pointer
	val = val.Elem()

	// Check if it's settable
	if !val.CanSet() {
		return errors.New("variable is not settable")
	}

	// Set the value based on the type
	switch val.Kind() {
	case reflect.Slice:
		if val.Type().Elem().Kind() == reflect.Uint8 {
			// []byte
			val.SetBytes(value)
		} else {
			return fmt.Errorf("unsupported slice type: %v", val.Type())
		}
	case reflect.String:
		val.SetString(string(value))
	default:
		return fmt.Errorf("unsupported binary variable type: %v", val.Kind())
	}

	return nil
}

// extractInteger extracts an integer value from the bitstring
func (m *Matcher) extractInteger(bs *bitstringpkg.BitString, offset, size uint, endiannessStr string, signed bool) (int64, error) {
	data := bs.ToBytes()
	byteOffset := offset / 8
	bitOffset := offset % 8

	// Handle bit-level extraction
	if bitOffset != 0 || size%8 != 0 {
		return extractIntegerBits(data, offset, size, signed)
	}

	// Handle byte-aligned extraction
	bytesNeeded := size / 8
	if byteOffset+bytesNeeded > uint(len(data)) {
		return 0, fmt.Errorf("insufficient data for extraction")
	}

	extractedData := data[byteOffset : byteOffset+bytesNeeded]

	switch endiannessStr {
	case bitstringpkg.EndiannessBig, "":
		return m.bytesToInt64BigEndian(extractedData, signed, size)
	case bitstringpkg.EndiannessLittle:
		return m.bytesToInt64LittleEndian(extractedData, signed, size)
	case bitstringpkg.EndiannessNative:
		return m.bytesToInt64Native(extractedData, signed, size)
	default:
		return 0, fmt.Errorf("unsupported endianness: %s", endiannessStr)
	}
}

// extractIntegerBits extracts an integer value from a non-byte-aligned position.
func extractIntegerBits(data []byte, startBit, numBits uint, signed bool) (int64, error) {
	if startBit+numBits > uint(len(data))*8 {
		return 0, fmt.Errorf("cannot extract %d bits starting from bit %d", numBits, startBit)
	}
	if numBits == 0 {
		return 0, nil
	}
	if numBits > 64 {
		return 0, fmt.Errorf("cannot extract more than 64 bits into an int64")
	}

	var value uint64
	for i := uint(0); i < numBits; i++ {
		currentBitPos := startBit + i
		bytePos := currentBitPos / 8
		bitInByte := 7 - (currentBitPos % 8) // 0 is MSB

		bit := (data[bytePos] >> bitInByte) & 1
		value = (value << 1) | uint64(bit)
	}

	// Handle signed interpretation
	if signed && numBits > 0 {
		// Check if the MSB is set (indicating a negative number in two's complement)
		msb := uint64(1) << (numBits - 1)
		if value&msb != 0 {
			// Sign extend: set all bits above the MSB to 1
			mask := ^(msb - 1)
			value |= mask
		}
	}

	return int64(value), nil
}

// bytesToInt64BigEndian converts bytes to int64 in big-endian format
func (m *Matcher) bytesToInt64BigEndian(data []byte, signed bool, size uint) (int64, error) {
	var result uint64 = 0

	for _, b := range data {
		result = (result << 8) | uint64(b)
	}

	// Handle signed interpretation
	if signed && size > 0 {
		// Check if the MSB is set (indicating a negative number in two's complement)
		msb := uint64(1) << (size - 1)
		if result&msb != 0 {
			// Sign extend: set all bits above the MSB to 1
			mask := ^(msb - 1)
			result |= mask
		}
	}

	return int64(result), nil
}

// bytesToInt64LittleEndian converts bytes to int64 in little-endian format
func (m *Matcher) bytesToInt64LittleEndian(data []byte, signed bool, size uint) (int64, error) {
	var result uint64 = 0

	for i := len(data) - 1; i >= 0; i-- {
		result = (result << 8) | uint64(data[i])
	}

	// Handle signed interpretation
	if signed && size > 0 {
		// Check if the MSB is set (indicating a negative number in two's complement)
		msb := uint64(1) << (size - 1)
		if result&msb != 0 {
			// Sign extend: set all bits above the MSB to 1
			mask := ^(msb - 1)
			result |= mask
		}
	}

	return int64(result), nil
}

// bytesToInt64Native converts bytes to int64 in native endianness format
func (m *Matcher) bytesToInt64Native(data []byte, signed bool, size uint) (int64, error) {
	if endianness.GetNativeEndianness() == "little" {
		switch len(data) {
		case 1:
			result := uint64(data[0])
			if signed && size > 0 {
				msb := uint64(1) << (size - 1)
				if result&msb != 0 {
					mask := ^(msb - 1)
					result |= mask
				}
			}
			return int64(result), nil
		case 2:
			result := uint64(binary.LittleEndian.Uint16(data))
			if signed && size > 0 {
				msb := uint64(1) << (size - 1)
				if result&msb != 0 {
					mask := ^(msb - 1)
					result |= mask
				}
			}
			return int64(result), nil
		case 4:
			result := uint64(binary.LittleEndian.Uint32(data))
			if signed && size > 0 {
				msb := uint64(1) << (size - 1)
				if result&msb != 0 {
					mask := ^(msb - 1)
					result |= mask
				}
			}
			return int64(result), nil
		case 8:
			result := binary.LittleEndian.Uint64(data)
			if signed && size > 0 {
				msb := uint64(1) << (size - 1)
				if result&msb != 0 {
					mask := ^(msb - 1)
					result |= mask
				}
			}
			return int64(result), nil
		default:
			// Fall back to little-endian for unusual sizes
			return m.bytesToInt64LittleEndian(data, signed, size)
		}
	} else {
		// Big-endian system
		switch len(data) {
		case 1:
			result := uint64(data[0])
			if signed && size > 0 {
				msb := uint64(1) << (size - 1)
				if result&msb != 0 {
					mask := ^(msb - 1)
					result |= mask
				}
			}
			return int64(result), nil
		case 2:
			result := uint64(binary.BigEndian.Uint16(data))
			if signed && size > 0 {
				msb := uint64(1) << (size - 1)
				if result&msb != 0 {
					mask := ^(msb - 1)
					result |= mask
				}
			}
			return int64(result), nil
		case 4:
			result := uint64(binary.BigEndian.Uint32(data))
			if signed && size > 0 {
				msb := uint64(1) << (size - 1)
				if result&msb != 0 {
					mask := ^(msb - 1)
					result |= mask
				}
			}
			return int64(result), nil
		case 8:
			result := binary.BigEndian.Uint64(data)
			if signed && size > 0 {
				msb := uint64(1) << (size - 1)
				if result&msb != 0 {
					mask := ^(msb - 1)
					result |= mask
				}
			}
			return int64(result), nil
		default:
			// Fall back to big-endian for unusual sizes
			return m.bytesToInt64BigEndian(data, signed, size)
		}
	}
}

// bindValue binds the extracted value to the variable
func (m *Matcher) bindValue(variable interface{}, value int64) error {
	if variable == nil {
		return bitstringpkg.NewBitStringError(bitstringpkg.CodeTypeMismatch, "variable cannot be nil")
	}

	// Use reflection to set the value
	val := reflect.ValueOf(variable)

	// Check if it's a pointer
	if val.Kind() != reflect.Ptr {
		return bitstringpkg.NewBitStringError(bitstringpkg.CodeTypeMismatch, "variable must be a pointer")
	}

	// Dereference the pointer
	val = val.Elem()

	// Check if it's settable
	if !val.CanSet() {
		return bitstringpkg.NewBitStringError(bitstringpkg.CodeTypeMismatch, "variable is not settable")
	}

	// Check if the variable type is compatible with integer values
	switch val.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		// Compatible integer types
	default:
		return bitstringpkg.NewBitStringErrorWithContext(bitstringpkg.CodeTypeMismatch,
			fmt.Sprintf("cannot bind integer value to variable of type %v", val.Kind()),
			val.Kind().String())
	}

	// Set the value based on the type
	switch val.Kind() {
	case reflect.Int:
		val.SetInt(value)
	case reflect.Int8:
		val.SetInt(value)
	case reflect.Int16:
		val.SetInt(value)
	case reflect.Int32:
		val.SetInt(value)
	case reflect.Int64:
		val.SetInt(value)
	case reflect.Uint:
		val.SetUint(uint64(value))
	case reflect.Uint8:
		val.SetUint(uint64(value))
	case reflect.Uint16:
		val.SetUint(uint64(value))
	case reflect.Uint32:
		val.SetUint(uint64(value))
	case reflect.Uint64:
		val.SetUint(uint64(value))
	}

	return nil
}

// extractRemainingBits extracts the remaining bits from a bitstring after a given offset
func (m *Matcher) extractRemainingBits(bs *bitstringpkg.BitString, offset uint) *bitstringpkg.BitString {
	if offset >= bs.Length() {
		return bitstringpkg.NewBitString()
	}

	remainingSize := bs.Length() - offset
	data := bs.ToBytes()
	byteOffset := offset / 8
	bitOffset := offset % 8
	bitRemainingSize := remainingSize % 8

	if bitOffset == 0 && bitRemainingSize == 0 {
		// Aligned case
		return bitstringpkg.NewBitStringFromBytes(data[byteOffset:])
	}

	// Non-aligned case - need to shift bits
	resultData := make([]byte, (remainingSize+7)/8)
	dataIndex := byteOffset
	resultIndex := uint(0)
	remainingBits := remainingSize

	// Handle first partial byte
	if bitOffset > 0 && remainingBits > 0 {
		bitsFromFirstByte := 8 - bitOffset
		if bitsFromFirstByte > remainingBits {
			bitsFromFirstByte = remainingBits
		}

		firstByte := data[dataIndex]
		mask := byte((1 << bitsFromFirstByte) - 1)
		extractedBits := (firstByte >> (8 - bitOffset - bitsFromFirstByte)) & mask

		resultData[resultIndex] = extractedBits << (8 - bitsFromFirstByte)

		dataIndex++
		remainingBits -= bitsFromFirstByte

		if remainingBits > 0 {
			resultIndex++
		}
	}

	// Handle full bytes
	for remainingBits >= 8 && dataIndex < uint(len(data)) {
		resultData[resultIndex] = data[dataIndex]
		dataIndex++
		resultIndex++
		remainingBits -= 8
	}

	// Handle last partial byte
	if remainingBits > 0 && dataIndex < uint(len(data)) {
		lastByte := data[dataIndex]
		mask := byte((1 << remainingBits) - 1)
		extractedBits := (lastByte >> (8 - remainingBits)) & mask

		if resultIndex < uint(len(resultData)) {
			resultData[resultIndex] = extractedBits << (8 - remainingBits)
		}
	}

	return bitstringpkg.NewBitStringFromBits(resultData, remainingSize)
}

// extractUTF extracts a UTF-encoded codepoint from the bitstring
func (m *Matcher) extractUTF(bs *bitstringpkg.BitString, offset uint, utfType, endianness string) (int, uint, error) {
	data := bs.ToBytes()
	byteOffset := offset / 8
	bitOffset := offset % 8

	// For UTF, we need byte-aligned data
	if bitOffset != 0 {
		return 0, 0, fmt.Errorf("UTF data must be byte-aligned")
	}

	if byteOffset >= uint(len(data)) {
		return 0, 0, fmt.Errorf("insufficient data for UTF extraction")
	}

	remainingData := data[byteOffset:]

	switch utfType {
	case "utf8":
		return m.extractUTF8(remainingData)
	case "utf16":
		return m.extractUTF16(remainingData, endianness)
	case "utf32":
		return m.extractUTF32(remainingData, endianness)
	default:
		return 0, 0, fmt.Errorf("unsupported UTF type: %s", utfType)
	}
}

// extractUTF8 extracts a UTF-8 encoded codepoint from the data
func (m *Matcher) extractUTF8(data []byte) (int, uint, error) {
	if len(data) == 0 {
		return 0, 0, fmt.Errorf("no data for UTF-8 extraction")
	}

	// Use Go's utf8.DecodeRune to properly decode UTF-8 sequences
	rune, size := utf8.DecodeRune(data)

	if rune == utf8.RuneError && size == 1 {
		// Check if it's a real error or just an incomplete sequence
		if !utf8.FullRune(data) {
			return 0, 0, fmt.Errorf("incomplete UTF-8 sequence")
		}
		return 0, 0, fmt.Errorf("invalid UTF-8 sequence")
	}

	if rune == utf8.RuneError {
		return 0, 0, fmt.Errorf("invalid UTF-8 sequence")
	}

	// Return the codepoint as integer (Erlang spec compliance)
	return int(rune), uint(size), nil
}

// extractUTF16 extracts a UTF-16 encoded codepoint from the data
func (m *Matcher) extractUTF16(data []byte, endiannessStr string) (int, uint, error) {
	if len(data) < 2 {
		return 0, 0, fmt.Errorf("insufficient data for UTF-16 extraction")
	}

	// UTF-16 uses 2 bytes per code unit
	bytesNeeded := 2
	if len(data) < bytesNeeded {
		return 0, 0, fmt.Errorf("insufficient data for UTF-16 extraction")
	}

	// Extract the 16-bit value
	var codeUnit uint16
	switch endiannessStr {
	case bitstringpkg.EndiannessBig, "":
		codeUnit = binary.BigEndian.Uint16(data[:2])
	case bitstringpkg.EndiannessLittle:
		codeUnit = binary.LittleEndian.Uint16(data[:2])
	case bitstringpkg.EndiannessNative:
		if endianness.GetNativeEndianness() == "little" {
			codeUnit = binary.LittleEndian.Uint16(data[:2])
		} else {
			codeUnit = binary.BigEndian.Uint16(data[:2])
		}
	default:
		return 0, 0, fmt.Errorf("unsupported endianness: %s", endiannessStr)
	}

	// Convert UTF-16 code unit to rune
	// For now, handle only BMP (Basic Multilingual Plane) characters
	if codeUnit >= 0xD800 && codeUnit <= 0xDFFF {
		// Surrogate pair - need additional 2 bytes
		if len(data) < 4 {
			return 0, 0, fmt.Errorf("incomplete surrogate pair in UTF-16")
		}

		var codeUnit2 uint16
		switch endiannessStr {
		case bitstringpkg.EndiannessBig, "":
			codeUnit2 = binary.BigEndian.Uint16(data[2:4])
		case bitstringpkg.EndiannessLittle:
			codeUnit2 = binary.LittleEndian.Uint16(data[2:4])
		case bitstringpkg.EndiannessNative:
			if endianness.GetNativeEndianness() == "little" {
				codeUnit2 = binary.LittleEndian.Uint16(data[2:4])
			} else {
				codeUnit2 = binary.BigEndian.Uint16(data[2:4])
			}
		}

		// Convert surrogate pair to code point
		if codeUnit >= 0xD800 && codeUnit <= 0xDBFF &&
			codeUnit2 >= 0xDC00 && codeUnit2 <= 0xDFFF {
			high := uint32(codeUnit - 0xD800)
			low := uint32(codeUnit2 - 0xDC00)
			codePoint := (high << 10) + low + 0x10000

			// Validate the resulting code point is within valid Unicode range
			if codePoint > 0x10FFFF || !utf8.ValidRune(rune(codePoint)) {
				return 0, 0, fmt.Errorf("invalid Unicode code point: %x", codePoint)
			}

			return int(codePoint), 4, nil
		}

		return 0, 0, fmt.Errorf("invalid surrogate pair in UTF-16")
	}

	// Single code unit from BMP
	if !utf8.ValidRune(rune(codeUnit)) {
		return 0, 0, fmt.Errorf("invalid Unicode code point: %x", codeUnit)
	}

	return int(codeUnit), 2, nil
}

// extractUTF32 extracts a UTF-32 encoded codepoint from the data
func (m *Matcher) extractUTF32(data []byte, endiannessStr string) (int, uint, error) {
	if len(data) < 4 {
		return 0, 0, fmt.Errorf("insufficient data for UTF-32 extraction")
	}

	// UTF-32 uses 4 bytes per code point
	var codePoint uint32
	switch endiannessStr {
	case bitstringpkg.EndiannessBig, "":
		codePoint = binary.BigEndian.Uint32(data[:4])
	case bitstringpkg.EndiannessLittle:
		codePoint = binary.LittleEndian.Uint32(data[:4])
	case bitstringpkg.EndiannessNative:
		if endianness.GetNativeEndianness() == "little" {
			codePoint = binary.LittleEndian.Uint32(data[:4])
		} else {
			codePoint = binary.BigEndian.Uint32(data[:4])
		}
	default:
		return 0, 0, fmt.Errorf("unsupported endianness: %s", endiannessStr)
	}

	// Validate the code point
	if codePoint > 0x10FFFF || (codePoint >= 0xD800 && codePoint <= 0xDFFF) {
		return 0, 0, fmt.Errorf("invalid Unicode code point: %x", codePoint)
	}

	if !utf8.ValidRune(rune(codePoint)) {
		return 0, 0, fmt.Errorf("invalid Unicode code point: %x", codePoint)
	}

	return int(codePoint), 4, nil
}

// bindUTFValue binds the extracted UTF codepoint to the variable
func (m *Matcher) bindUTFValue(variable interface{}, codepoint int) error {
	if variable == nil {
		return errors.New("variable cannot be nil")
	}

	// Use reflection to set the value
	val := reflect.ValueOf(variable)

	// Check if it's a pointer
	if val.Kind() != reflect.Ptr {
		return errors.New("variable must be a pointer")
	}

	// Dereference the pointer
	val = val.Elem()

	// Check if it's settable
	if !val.CanSet() {
		return errors.New("variable is not settable")
	}

	// Set the value based on the type
	switch val.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// For integer types: set the codepoint directly (Erlang spec!)
		val.SetInt(int64(codepoint))
	case reflect.String:
		// For string types: convert codepoint to character (backward compatibility)
		val.SetString(string(rune(codepoint)))
	case reflect.Slice:
		// For byte slices, convert string to bytes (backward compatibility)
		if val.Type().Elem().Kind() == reflect.Uint8 {
			val.Set(reflect.ValueOf([]byte(string(rune(codepoint)))))
		} else {
			return fmt.Errorf("unsupported UTF variable slice type: %v", val.Type())
		}
	default:
		return fmt.Errorf("unsupported UTF variable type: %v", val.Kind())
	}

	return nil
}

// createRestResult creates a common result structure for rest patterns
func (m *Matcher) createRestResult(value interface{}, offset uint, remainingBits uint) (*bitstringpkg.SegmentResult, uint) {
	// Create remaining bitstring (should be empty for rest patterns)
	remaining := bitstringpkg.NewBitString()

	result := &bitstringpkg.SegmentResult{
		Value:     value,
		Matched:   true,
		Remaining: remaining,
	}

	return result, offset + remainingBits
}

// matchRestBinary matches the rest of the bitstring as binary (must be byte-aligned)
func (m *Matcher) matchRestBinary(segment *bitstringpkg.Segment, bs *bitstringpkg.BitString, offset uint) (*bitstringpkg.SegmentResult, uint, error) {
	remainingBits := bs.Length() - offset

	// Check if remaining bits are byte-aligned
	if remainingBits%8 != 0 {
		return nil, 0, fmt.Errorf("rest binary requires byte-aligned data, but %d bits remain (not divisible by 8)", remainingBits)
	}

	// Extract remaining data as binary
	value, err := m.extractBinary(bs, offset, remainingBits)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to extract rest binary data: %v", err)
	}

	// Bind the value to the variable
	if err := m.bindBinaryValue(segment.Value, value); err != nil {
		return nil, 0, fmt.Errorf("failed to bind rest binary value: %v", err)
	}

	result, newOffset := m.createRestResult(value, offset, remainingBits)
	return result, newOffset, nil
}

// matchRestBitstring matches the rest of the bitstring as bitstring (any bit length)
func (m *Matcher) matchRestBitstring(segment *bitstringpkg.Segment, bs *bitstringpkg.BitString, offset uint) (*bitstringpkg.SegmentResult, uint, error) {
	remainingBits := bs.Length() - offset

	// Extract remaining data as bitstring
	value := m.extractRemainingBits(bs, offset)

	// Verify that the extracted bitstring has the expected length
	if value.Length() != remainingBits {
		return nil, 0, fmt.Errorf("extracted bitstring length %d doesn't match expected remaining bits %d", value.Length(), remainingBits)
	}

	// Bind the value to the variable
	if err := m.bindBitstringValue(segment.Value, value); err != nil {
		return nil, 0, fmt.Errorf("failed to bind rest bitstring value: %v", err)
	}

	result, newOffset := m.createRestResult(value, offset, remainingBits)
	return result, newOffset, nil
}

// bindBitstringValue binds the extracted bitstring value to the variable
func (m *Matcher) bindBitstringValue(variable interface{}, value *bitstringpkg.BitString) error {
	if variable == nil {
		return errors.New("variable cannot be nil")
	}

	// Use reflection to set the value
	val := reflect.ValueOf(variable)

	// Check if it's a pointer
	if val.Kind() != reflect.Ptr {
		return errors.New("variable must be a pointer")
	}

	// Dereference the pointer
	val = val.Elem()

	// Check if it's settable
	if !val.CanSet() {
		return errors.New("variable is not settable")
	}

	// Set the value based on the type
	switch val.Kind() {
	case reflect.Ptr:
		// For *BitString type
		if val.Type() == reflect.TypeOf(&bitstringpkg.BitString{}) {
			val.Set(reflect.ValueOf(value))
		} else {
			return fmt.Errorf("unsupported bitstring variable type: %v", val.Type())
		}
	default:
		return fmt.Errorf("unsupported bitstring variable type: %v", val.Kind())
	}

	return nil
}

// initializeContextFromRegisteredVariables initializes the dynamic size context with registered variables
func (m *Matcher) initializeContextFromRegisteredVariables(context *DynamicSizeContext) {
	for name, variable := range m.variables {
		// Extract the current value from the variable pointer
		value := m.extractValueFromVariable(variable)
		context.AddVariable(name, value)
	}
}

// extractValueFromVariable extracts the current uint value from a variable pointer
func (m *Matcher) extractValueFromVariable(variable interface{}) uint {
	val := reflect.ValueOf(variable)

	// Check if it's a pointer
	if val.Kind() != reflect.Ptr {
		return 0
	}

	// Dereference the pointer
	val = val.Elem()

	// Check if it's settable
	if !val.IsValid() {
		return 0
	}

	// Convert to uint based on type
	switch val.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return uint(val.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return uint(val.Uint())
	default:
		return 0
	}
}
