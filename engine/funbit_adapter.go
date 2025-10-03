package engine

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"funterm/shared"
	"go-parser/pkg/ast"

	"github.com/funvibe/funbit/pkg/funbit"
)

// FunbitBitstringSpecifiers represents parsed bitstring specifiers for the funbit adapter
type FunbitBitstringSpecifiers struct {
	Type       string
	Signed     bool
	Endianness string
	Unit       uint
}

// FunbitAdapter provides a bridge between funterm AST and funbit API
type FunbitAdapter struct {
	engine          *ExecutionEngine
	verbose         bool                   // Verbose flag for debug output
	variables       map[string]interface{} // Registered variables for dynamic sizing
	constantStorage map[string]*int        // Storage for constant values during pattern matching
}

// NewFunbitAdapter creates a new FunbitAdapter instance
func NewFunbitAdapter() *FunbitAdapter {
	return &FunbitAdapter{
		variables:       make(map[string]interface{}),
		constantStorage: make(map[string]*int),
	}
}

// NewFunbitAdapterWithEngine creates a new FunbitAdapter instance with ExecutionEngine
func NewFunbitAdapterWithEngine(engine *ExecutionEngine) *FunbitAdapter {
	return &FunbitAdapter{
		engine:          engine,
		verbose:         engine.verbose, // Передаем verbose flag из engine
		variables:       make(map[string]interface{}),
		constantStorage: make(map[string]*int),
	}
}

// addIntegerWithOverflowHandling safely adds an integer to the builder, handling overflow by truncating
func (fa *FunbitAdapter) addIntegerWithOverflowHandling(builder *funbit.Builder, value interface{}, options ...funbit.SegmentOption) error {
	// Convert value to int64 for potential truncation
	var intValue int64
	switch v := value.(type) {
	case int:
		intValue = int64(v)
	case int64:
		intValue = v
	case float64:
		intValue = int64(v)
	default:
		// For non-integer types, just pass through
		funbit.AddInteger(builder, value, options...)
		return nil
	}

	// Extract size from options to determine if truncation is needed
	var segmentSize uint = 8 // Default size for integers

	// Create a temporary segment to extract size from options
	tempSegment := &funbit.Segment{}
	for _, option := range options {
		option(tempSegment)
	}

	// Use the size from options if specified
	if tempSegment.SizeSpecified {
		segmentSize = tempSegment.Size
	}

	// Check if value fits in the specified size
	if segmentSize > 0 && segmentSize <= 64 {
		maxValue := int64(1) << segmentSize
		minValue := int64(0)

		// For signed integers, adjust range
		if tempSegment.Signed {
			maxValue = maxValue / 2
			minValue = -maxValue
		}

		// Truncate if value doesn't fit
		if intValue >= maxValue || intValue < minValue {
			if tempSegment.Signed {
				// For signed, use two's complement wrapping
				mask := (int64(1) << segmentSize) - 1
				intValue = intValue & mask
				// Convert to signed if MSB is set
				if intValue >= (int64(1) << (segmentSize - 1)) {
					intValue = intValue - (int64(1) << segmentSize)
				}
			} else {
				// For unsigned, simple modulo
				mask := (int64(1) << segmentSize) - 1
				intValue = intValue & mask
			}
		}
	}

	// Add the (possibly truncated) value
	funbit.AddInteger(builder, intValue, options...)
	return nil
}

// ExecuteBitstringExpression executes a bitstring expression and returns a BitstringObject
func (fa *FunbitAdapter) ExecuteBitstringExpression(expr *ast.BitstringExpression) (*shared.BitstringObject, error) {
	if fa.verbose {
		fmt.Printf("DEBUG: ExecuteBitstringExpression - ENTRY with %d segments (ptr=%p)\n", len(expr.Segments), expr)
	}
	// Handle empty bitstring specially
	if len(expr.Segments) == 0 {
		// Create empty bitstring using funbit.NewBitString()
		emptyBitstring := funbit.NewBitString()
		bitstringObject := &shared.BitstringObject{
			BitString: emptyBitstring,
		}
		return bitstringObject, nil
	}

	builder := funbit.NewBuilder()
	totalBits := uint(0)

	// Process each segment in the bitstring expression
	for _, segment := range expr.Segments {
		bitsAdded, err := fa.addSegment(builder, &segment)
		if err != nil {
			return nil, fmt.Errorf("failed to add segment: %v", err)
		}
		totalBits += bitsAdded
	}

	// Build the bitstring
	bitstring, err := funbit.Build(builder)
	if err != nil {
		return nil, fmt.Errorf("failed to build bitstring: %v", err)
	}

	// Create BitstringObject
	bitstringObject := &shared.BitstringObject{
		BitString: bitstring,
	}

	return bitstringObject, nil
}

// addSegment adds a single segment to the funbit builder and returns bits added
func (fa *FunbitAdapter) addSegment(builder *funbit.Builder, segment *ast.BitstringSegment) (uint, error) {
	if fa.verbose {
		fmt.Printf("DEBUG: addSegment - processing segment with value type %T, value: %v\n", segment.Value, segment.Value)
	}
	bitsAdded := uint(0)

	// Convert the value from AST expression to Go interface{}
	value, err := fa.convertValue(segment.Value)
	if err != nil {
		return 0, fmt.Errorf("failed to convert value: %v", err)
	}

	// Check if this is a zero-size segment (padding/no-op)
	if segment.Size != nil {
		sizeValue, err := fa.convertValue(segment.Size)
		if err != nil {
			return 0, fmt.Errorf("failed to convert size: %v", err)
		}

		// Convert size to uint to check if it's zero
		var size uint
		switch v := sizeValue.(type) {
		case int64:
			size = uint(v)
		case float64:
			size = uint(v)
		case int:
			size = uint(v)
		default:
			return 0, fmt.Errorf("unsupported size type: %T", sizeValue)
		}

		// Skip zero-size segments entirely (padding/no-op)
		if size == 0 {
			return 0, nil
		}
	}

	// Parse specifiers if present
	var options []funbit.SegmentOption
	var specs FunbitBitstringSpecifiers
	if len(segment.Specifiers) > 0 {
		var err error
		specs, err = fa.parseSpecifiers(segment.Specifiers)
		if err != nil {
			return 0, fmt.Errorf("failed to parse specifiers: %v", err)
		}

		// Apply specifiers as options
		if specs.Type != "" {
			options = append(options, funbit.WithType(specs.Type))
		}
		// Note: For integer and binary types, unit multiplier is handled separately in their cases
		// to avoid double application (once in effectiveSize calculation, once by funbit)
		if specs.Unit > 0 && specs.Type != "integer" && specs.Type != "binary" && specs.Type != "float" {
			if fa.verbose {
				fmt.Printf("DEBUG: AddSegment - adding WithUnit(%d) for type '%s'\n", specs.Unit, specs.Type)
			}
			options = append(options, funbit.WithUnit(specs.Unit))
		} else if specs.Unit > 0 && (specs.Type == "integer" || specs.Type == "binary" || specs.Type == "float") {
			if fa.verbose {
				fmt.Printf("DEBUG: AddSegment - skipping WithUnit(%d) for %s type (handled in case)\n", specs.Unit, specs.Type)
			}
		}
		if specs.Signed {
			options = append(options, funbit.WithSigned(specs.Signed))
		}
		if specs.Endianness != "" {
			if fa.verbose {
				fmt.Printf("DEBUG: AddSegment - applying endianness: %s\n", specs.Endianness)
			}
			options = append(options, funbit.WithEndianness(specs.Endianness))
		}
	}

	// Handle size if present (but not for UTF types) - declare size at function scope
	var size uint
	if segment.Size != nil {
		// Size is specified - convert it
		// Check if this is a UTF type - UTF types cannot have size specified
		isUTFType := specs.Type == "utf8" || specs.Type == "utf16" || specs.Type == "utf32" || specs.Type == "utf"

		if !isUTFType {
			var err error

			if segment.IsDynamicSize && segment.SizeExpression != nil {
				// Dynamic size - resolve at runtime
				size, err = fa.resolveDynamicSize(segment.SizeExpression, nil)
				if err != nil {
					return 0, fmt.Errorf("failed to resolve dynamic size: %v", err)
				}
			} else {
				// Static size - convert directly
				sizeValue, err := fa.convertValue(segment.Size)
				if err != nil {
					return 0, fmt.Errorf("failed to convert size: %v", err)
				}

				// Convert size to uint
				switch v := sizeValue.(type) {
				case int64:
					size = uint(v)
				case float64:
					size = uint(v)
				case int:
					size = uint(v)
				default:
					return 0, fmt.Errorf("unsupported size type: %T", sizeValue)
				}
			}

			// Skip zero-size segments (padding/no-op)
			if size > 0 {
				// For binary type, size is in bytes, bitsAdded is in bits
				if specs.Type == "binary" || specs.Type == "bytes" {
					bitsAdded = size * 8
					options = append(options, funbit.WithSize(size))
				} else {
					bitsAdded = size
					options = append(options, funbit.WithSize(size))
				}
			}
		}
	}

	// Determine the type and add the segment accordingly
	if len(segment.Specifiers) > 0 || segment.Size != nil {
		specs, _ := fa.parseSpecifiers(segment.Specifiers) // We already validated this above

		// Handle the case where type is not specified (defaults to "")
		if specs.Type == "" {
			// Calculate effective size for unit specifiers
			var effectiveSize uint
			if segment.Size != nil && specs.Unit > 0 {
				// Manually calculate size * unit
				effectiveSize = size * uint(specs.Unit)
			} else if segment.Size != nil {
				effectiveSize = size
			} else if specs.Unit > 0 {
				effectiveSize = uint(specs.Unit)
			} else {
				effectiveSize = 8 // Default size
			}

			// Create options without WithUnit to avoid conflicts
			sizeOptions := []funbit.SegmentOption{funbit.WithSize(effectiveSize)}
			if specs.Signed {
				sizeOptions = append(sizeOptions, funbit.WithSigned(specs.Signed))
			}
			if specs.Endianness != "" {
				sizeOptions = append(sizeOptions, funbit.WithEndianness(specs.Endianness))
			}

			switch v := value.(type) {
			case int, int64:
				err := fa.addIntegerWithOverflowHandling(builder, value, sizeOptions...)
				if err != nil {
					return 0, fmt.Errorf("failed to add integer: %v", err)
				}
				bitsAdded = effectiveSize
			case float64:
				// Check if the float value is actually a whole number
				if v == float64(int(v)) {
					err := fa.addIntegerWithOverflowHandling(builder, int(v), sizeOptions...)
					if err != nil {
						return 0, fmt.Errorf("failed to add integer: %v", err)
					}
					bitsAdded = effectiveSize
				} else {
					funbit.AddFloat(builder, v, sizeOptions...)
					bitsAdded = effectiveSize
				}
			case string:
				// Check if this is a UTF type based on specifiers
				isUTFType := false
				for _, spec := range segment.Specifiers {
					if spec == "utf8" || spec == "utf16" || spec == "utf32" || spec == "utf" {
						isUTFType = true
						break
					}
				}

				if isUTFType {
					// For UTF types, use AddUTF
					funbit.AddUTF(builder, v, sizeOptions...)
					// UTF strings have variable bit length, calculate based on UTF-8 encoding
					bitsAdded = uint(len([]byte(v))) * 8
				} else {
					// Convert string to bytes and add as binary
					bytes := []byte(v)
					bitsAdded = uint(len(bytes)) * 8
					stringOptions := []funbit.SegmentOption{funbit.WithSize(bitsAdded)}
					funbit.AddBinary(builder, bytes, stringOptions...)
				}
			case []byte:
				funbit.AddBinary(builder, v, options...)
			case *funbit.BitString:
				bitsAdded = uint(v.Length())
				// Skip empty bitstrings - they cause "size must be positive" error in funbit.Build
				if v.Length() > 0 {
					// Don't pass size options when adding existing bitstring
					funbit.AddBitstring(builder, v)
				}
			case *shared.BitstringObject:
				bitsAdded = uint(v.Len())
				// Skip empty bitstrings - they cause "size must be positive" error in funbit.Build
				if v.Len() > 0 {
					// Don't pass size options when adding existing bitstring
					funbit.AddBitstring(builder, v.BitString)
				}
			default:
				return 0, fmt.Errorf("unsupported value type for auto-detection: %T", value)
			}
		} else {
			switch specs.Type {
			case "integer":
				// Calculate effective size for this case
				var effectiveSize uint = 8 // Default size
				if size > 0 {
					effectiveSize = size
					// Apply unit multiplier if specified
					if specs.Unit > 1 {
						effectiveSize = size * uint(specs.Unit)
						if fa.verbose {
							fmt.Printf("DEBUG: integer case - size=%d, unit=%d, effectiveSize=%d\n", size, specs.Unit, effectiveSize)
						}
					}
				} else if specs.Unit > 1 {
					// If no size specified but unit is given, use unit as size
					effectiveSize = uint(specs.Unit)
					if fa.verbose {
						fmt.Printf("DEBUG: integer case - no size, unit=%d, effectiveSize=%d\n", specs.Unit, effectiveSize)
					}
				}
				if fa.verbose {
					fmt.Printf("DEBUG: integer case - final effectiveSize=%d bits for value=%v\n", effectiveSize, value)
				}

				// Create options for this context
				integerOptions := []funbit.SegmentOption{funbit.WithSize(effectiveSize)}

				// Add other options (endianness, signedness, etc.)
				if specs.Signed {
					integerOptions = append(integerOptions, funbit.WithSigned(specs.Signed))
				}
				if specs.Endianness != "" {
					integerOptions = append(integerOptions, funbit.WithEndianness(specs.Endianness))
				}
				// Note: We don't pass Unit to funbit.AddInteger because we already applied it to effectiveSize
				// funbit expects either Size OR (Size * Unit), not both

				// Convert float64 to int if needed
				if floatVal, ok := value.(float64); ok {
					if floatVal == float64(int(floatVal)) {
						// It's a whole number, convert to int
						intVal := int(floatVal)
						err := fa.addIntegerWithOverflowHandling(builder, intVal, integerOptions...)
						if err != nil {
							return 0, fmt.Errorf("failed to add integer: %v", err)
						}
					} else {
						return 0, fmt.Errorf("integer type requires whole number, got float %f", floatVal)
					}
				} else {
					err := fa.addIntegerWithOverflowHandling(builder, value, integerOptions...)
					if err != nil {
						return 0, fmt.Errorf("failed to add integer: %v", err)
					}
				}
			case "bitstring":
				// Add bitstring values using AddBitstring
				if bitstring, ok := value.(*funbit.BitString); ok {
					// Skip empty bitstrings - they cause "size must be positive" error in funbit.Build
					if bitstring.Length() > 0 {
						// Don't pass size options when adding existing bitstring
						funbit.AddBitstring(builder, bitstring)
					}
				} else if bitstringObj, ok := value.(*shared.BitstringObject); ok {
					// Skip empty bitstrings - they cause "size must be positive" error in funbit.Build
					if bitstringObj.Len() > 0 {
						// Don't pass size options when adding existing bitstring
						funbit.AddBitstring(builder, bitstringObj.BitString)
					}
				} else {
					return 0, fmt.Errorf("bitstring type requires *funbit.BitString or *shared.BitstringObject value, got %T", value)
				}
			case "float":
				// Calculate effective size for float type
				var effectiveSize uint = 32 // Default float size
				if segment.Size != nil {
					sizeValue, err := fa.convertValue(segment.Size)
					if err == nil {
						if size, ok := sizeValue.(int); ok {
							effectiveSize = uint(size)
							if specs.Unit > 0 {
								effectiveSize = uint(size) * specs.Unit
							}
						}
					}
				}

				// Validate float size - funbit supports 16, 32, or 64 bits
				if effectiveSize != 16 && effectiveSize != 32 && effectiveSize != 64 {
					return 0, fmt.Errorf("float type requires 16, 32, or 64 bits, got %d", effectiveSize)
				}

				// Create float options with calculated size (not unit)
				floatOptions := []funbit.SegmentOption{funbit.WithSize(effectiveSize)}
				if specs.Endianness != "" {
					floatOptions = append(floatOptions, funbit.WithEndianness(specs.Endianness))
				}
				// Note: We don't pass Unit to funbit.AddFloat because we already applied it to effectiveSize

				funbit.AddFloat(builder, value, floatOptions...)
			case "binary", "bytes":
				// Calculate effective size for binary type
				// For binary: Size * Unit gives total BITS, then convert to bytes
				var effectiveSizeBits uint = size * 8 // Default: size in bytes -> bits
				if size > 0 && specs.Unit > 0 {
					// Apply unit multiplier: size * unit gives total bits
					effectiveSizeBits = size * uint(specs.Unit)
				}
				// Convert bits to bytes (round up)
				var effectiveSize uint = (effectiveSizeBits + 7) / 8

				// Create clean options for binary (no size or unit, as we handle size manually)
				binaryOptions := []funbit.SegmentOption{}
				if specs.Endianness != "" {
					binaryOptions = append(binaryOptions, funbit.WithEndianness(specs.Endianness))
				}
				if specs.Signed {
					binaryOptions = append(binaryOptions, funbit.WithSigned(specs.Signed))
				}
				if specs.Type != "" {
					binaryOptions = append(binaryOptions, funbit.WithType(specs.Type))
				}

				if bytes, ok := value.([]byte); ok {
					// If we have a size specification, truncate or pad the data
					if effectiveSize > 0 {
						if uint(len(bytes)) > effectiveSize {
							// Truncate to specified size
							bytes = bytes[:effectiveSize]
						} else if uint(len(bytes)) < effectiveSize {
							// Pad with zeros to reach specified size
							padded := make([]byte, effectiveSize)
							copy(padded, bytes)
							bytes = padded
						}
					}
					funbit.AddBinary(builder, bytes, binaryOptions...)
				} else if str, ok := value.(string); ok {
					// Convert string to bytes for binary type
					bytes := []byte(str)
					// If we have a size specification, truncate or pad the data
					if effectiveSize > 0 {
						if uint(len(bytes)) > effectiveSize {
							// Truncate to specified size
							bytes = bytes[:effectiveSize]
						} else if uint(len(bytes)) < effectiveSize {
							// Pad with zeros to reach specified size
							padded := make([]byte, effectiveSize)
							copy(padded, bytes)
							bytes = padded
						}
					}
					funbit.AddBinary(builder, bytes, binaryOptions...)
				} else if bitstringObj, ok := value.(*shared.BitstringObject); ok {
					// Add BitstringObject as bitstring, not as binary bytes
					// Skip empty bitstrings - they cause "size must be positive" error in funbit.Build
					if bitstringObj.Len() > 0 {
						// Don't pass size options when adding existing bitstring
						funbit.AddBitstring(builder, bitstringObj.BitString)
					}
				} else if intValue, ok := value.(int); ok {
					// For binary type, provide exactly effectiveSize bytes
					binarySize := int(effectiveSize)
					if binarySize == 0 {
						binarySize = 4 // Default to 4 bytes if no size specified
					}
					bytes := make([]byte, binarySize)
					// Put the value in big-endian format
					for i := 0; i < binarySize; i++ {
						bytes[binarySize-1-i] = byte(intValue >> (i * 8))
					}
					funbit.AddBinary(builder, bytes, binaryOptions...)
				} else if floatValue, ok := value.(float64); ok {
					// Convert number to bytes for binary type
					if floatValue == float64(int(floatValue)) {
						intValue := int(floatValue)
						// For binary type, provide exactly effectiveSize bytes
						binarySize := int(effectiveSize)
						if binarySize == 0 {
							binarySize = 4 // Default to 4 bytes if no size specified
						}
						bytes := make([]byte, binarySize)
						// Put the value in big-endian format
						for i := 0; i < binarySize; i++ {
							bytes[binarySize-1-i] = byte(intValue >> (i * 8))
						}
						funbit.AddBinary(builder, bytes, binaryOptions...)
					} else {
						return 0, fmt.Errorf("binary type requires integer values, got float %f", floatValue)
					}
				} else {
					return 0, fmt.Errorf("binary type requires []byte, string, numeric, or BitstringObject value, got %T", value)
				}
			case "bits":
				if bitstring, ok := value.(*funbit.BitString); ok {
					funbit.AddBitstring(builder, bitstring, options...)
				} else if bitstringObj, ok := value.(*shared.BitstringObject); ok {
					funbit.AddBitstring(builder, bitstringObj.BitString, options...)
				} else {
					return 0, fmt.Errorf("bitstring type requires *funbit.BitString or *shared.BitstringObject value, got %T", value)
				}
			case "utf8", "utf16", "utf32", "utf":
				// For UTF types, convert value to string first
				var str string
				switch v := value.(type) {
				case string:
					str = v
				case int, int64:
					// Convert number to Unicode codepoint string
					var codepoint int
					switch val := v.(type) {
					case int:
						codepoint = val
					case int64:
						codepoint = int(val)
					}
					str = string(rune(codepoint))
				case float64:
					// Convert float to int then to Unicode codepoint string
					str = string(rune(int(v)))
				default:
					return 0, fmt.Errorf("UTF type requires string or numeric value, got %T", value)
				}
				// For UTF types, use AddUTF instead of AddBinary
				funbit.AddUTF(builder, str, options...)
			default:
				return 0, fmt.Errorf("unsupported type: %s", specs.Type)
			}
		}
	} else {
		// No specifiers - use default behavior based on value type
		// Calculate the effective size (segment size * unit)
		var effectiveSize uint = 8 // Default size
		if segment.Size != nil {
			sizeValue, err := fa.convertValue(segment.Size)
			if err != nil {
				return 0, fmt.Errorf("failed to convert segment size: %v", err)
			}
			var segmentSize uint
			switch v := sizeValue.(type) {
			case int64:
				segmentSize = uint(v)
			case float64:
				segmentSize = uint(v)
			case int:
				segmentSize = uint(v)
			default:
				return 0, fmt.Errorf("unsupported segment size type: %T", sizeValue)
			}

			if specs.Unit > 0 {
				effectiveSize = segmentSize * uint(specs.Unit)
			} else {
				effectiveSize = segmentSize
			}
		} else if specs.Unit > 0 {
			effectiveSize = uint(specs.Unit)
		}

		// Create options for this context
		defaultOptions := []funbit.SegmentOption{funbit.WithSize(effectiveSize)}

		switch v := value.(type) {
		case int, int64:
			err := fa.addIntegerWithOverflowHandling(builder, value, defaultOptions...)
			if err != nil {
				return 0, fmt.Errorf("failed to add integer: %v", err)
			}
			bitsAdded = effectiveSize
		case float64:
			// Check if the float value is actually a whole number
			if v == float64(int(v)) {
				// It's a whole number, treat as integer
				err := fa.addIntegerWithOverflowHandling(builder, int(v), defaultOptions...)
				if err != nil {
					return 0, fmt.Errorf("failed to add integer: %v", err)
				}
				bitsAdded = effectiveSize
			} else {
				// It's a real float, use float type
				funbit.AddFloat(builder, v, funbit.WithSize(effectiveSize))
				bitsAdded = effectiveSize
			}
		case string:
			// Convert string to bytes and add as binary
			bytes := []byte(v)
			funbit.AddBinary(builder, bytes)
			bitsAdded = uint(len(bytes)) * 8
		case *funbit.BitString:
			if v.Length() > 0 {
				funbit.AddBitstring(builder, v)
				bitsAdded = uint(v.Length())
			}
		case *shared.BitstringObject:
			if v.Len() > 0 {
				funbit.AddBitstring(builder, v.BitString)
				bitsAdded = uint(v.Len())
			}
		default:
			return 0, fmt.Errorf("unsupported value type without specifiers: %T", value)
		}
	}

	return bitsAdded, nil
}

// convertValue converts an AST expression to a Go interface{} value
func (fa *FunbitAdapter) convertValue(expr ast.Expression) (interface{}, error) {
	switch e := expr.(type) {
	case *ast.NumberLiteral:
		// Check if the number is an integer (no fractional part)
		if e.Value == float64(int(e.Value)) {
			return int(e.Value), nil
		}
		return e.Value, nil
	case *ast.StringLiteral:
		return e.Value, nil
	case *ast.BooleanLiteral:
		return e.Value, nil
	case *ast.Identifier:
		// If we have an ExecutionEngine, use it to resolve the identifier
		if fa.engine != nil {
			value, err := fa.engine.convertExpressionToValue(e)
			if err != nil {
				return nil, err
			}

			// Convert float64 to int if it's a whole number (common case for runtime variables)
			if floatValue, ok := value.(float64); ok {
				if floatValue == float64(int(floatValue)) {
					return int(floatValue), nil
				}
			}

			return value, nil
		}
		// Otherwise, return the identifier name as a string (fallback for backward compatibility)
		return e.Name, nil
	case *ast.UnaryExpression:
		// Handle unary expressions like -42
		value, err := fa.convertValue(e.Right)
		if err != nil {
			return nil, err
		}
		switch e.Operator {
		case "-":
			switch v := value.(type) {
			case int64:
				return -v, nil
			case float64:
				return -v, nil
			case int:
				return -v, nil
			default:
				return nil, fmt.Errorf("cannot apply unary minus to type %T", value)
			}
		case "+":
			return value, nil
		default:
			return nil, fmt.Errorf("unsupported unary operator: %s", e.Operator)
		}
	case *ast.BinaryExpression:
		// Handle binary expressions (like arithmetic operations)
		if fa.engine == nil {
			return nil, fmt.Errorf("cannot evaluate binary expressions without ExecutionEngine")
		}
		return fa.engine.convertExpressionToValue(e)
	case *ast.BitstringExpression:
		// Handle nested bitstring expressions
		return fa.ExecuteBitstringExpression(e)
	case *ast.TernaryExpression:
		// Handle ternary expressions (condition ? trueValue : falseValue)
		if fa.engine == nil {
			return nil, fmt.Errorf("cannot evaluate ternary expressions without ExecutionEngine")
		}
		return fa.engine.convertExpressionToValue(e)
	default:
		return nil, fmt.Errorf("unsupported expression type: %T", expr)
	}
}

// parseSpecifiers parses a list of specifier strings into FunbitBitstringSpecifiers
func (fa *FunbitAdapter) parseSpecifiers(specifiers []string) (FunbitBitstringSpecifiers, error) {
	result := FunbitBitstringSpecifiers{
		Type:       "",    // No default type - let it be determined by value
		Signed:     false, // Default unsigned
		Endianness: "big", // Default big-endian
		Unit:       1,     // Default unit
	}

	// Check if binary type is specified and set default unit accordingly
	for _, spec := range specifiers {
		if spec == "binary" || spec == "bytes" {
			result.Unit = 8 // Default unit for binary is 8
			break
		}
	}

	for _, spec := range specifiers {
		// Handle specifiers with parameters (e.g., "unit:8", "integer-unit:1", "little-signed-integer-unit:8")
		if strings.Contains(spec, ":") {
			parts := strings.Split(spec, ":")
			if len(parts) != 2 {
				return result, fmt.Errorf("invalid specifier format: %s", spec)
			}

			leftPart := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			// Check if this is a compound specifier with multiple components
			if strings.Contains(leftPart, "-") {
				compoundParts := strings.Split(leftPart, "-")

				// Handle multi-component specifiers like "little-signed-integer-unit"
				// Format: [endianness]-[signedness]-[type]-unit:value
				// Or simpler formats like "type-unit:value"

				if len(compoundParts) >= 2 {
					// Check if the last part is "unit"
					lastPart := compoundParts[len(compoundParts)-1]
					if lastPart == "unit" {
						// Parse unit value
						unit, err := strconv.ParseUint(value, 10, 32)
						if err != nil {
							return result, fmt.Errorf("invalid unit value: %s", value)
						}
						result.Unit = uint(unit)

						// Parse the remaining parts (excluding "unit")
						remainingParts := compoundParts[:len(compoundParts)-1]

						// Special handling for endianness compounds in remaining parts
						remainingSpec := strings.Join(remainingParts, "-")
						if remainingSpec == "big-endian" {
							result.Endianness = "big"
						} else if remainingSpec == "little-endian" {
							result.Endianness = "little"
						} else if remainingSpec == "native-endian" {
							result.Endianness = funbit.GetNativeEndianness()
						} else {
							// Parse each component individually
							for _, part := range remainingParts {
								switch part {
								// Endianness
								case "big":
									result.Endianness = "big"
								case "little":
									result.Endianness = "little"
								case "native":
									result.Endianness = funbit.GetNativeEndianness()
								// Signedness
								case "signed":
									result.Signed = true
								case "unsigned":
									result.Signed = false
								// Types
								case "integer":
									result.Type = "integer"
								case "binary":
									result.Type = "binary"
								case "float":
									result.Type = "float"
								case "bitstring", "bits":
									result.Type = "bitstring"
								case "utf8":
									result.Type = "utf8"
								case "utf16":
									result.Type = "utf16"
								case "utf32":
									result.Type = "utf32"
								case "utf":
									result.Type = "utf"
								default:
									return result, fmt.Errorf("unknown component in compound specifier: %s", part)
								}
							}
						}
					} else {
						return result, fmt.Errorf("invalid compound specifier format: expected 'unit' as last component, got '%s' in %s", lastPart, spec)
					}
				} else {
					return result, fmt.Errorf("invalid compound specifier format: %s", spec)
				}
			} else {
				// Handle simple key:value specifiers
				switch leftPart {
				case "unit":
					unit, err := strconv.ParseUint(value, 10, 32)
					if err != nil {
						return result, fmt.Errorf("invalid unit value: %s", value)
					}
					result.Unit = uint(unit)
				default:
					return result, fmt.Errorf("unknown specifier parameter: %s", leftPart)
				}
			}
		} else {
			// Handle simple specifiers and compound specifiers without unit
			if strings.Contains(spec, "-") {
				// Handle compound specifiers without unit (e.g., "little-signed-integer", "big-endian")

				// Special handling for endianness compounds
				if spec == "big-endian" {
					result.Endianness = "big"
				} else if spec == "little-endian" {
					result.Endianness = "little"
				} else if spec == "native-endian" {
					result.Endianness = funbit.GetNativeEndianness()
				} else {
					// Handle other compound specifiers
					compoundParts := strings.Split(spec, "-")

					// Parse each component
					for _, part := range compoundParts {
						switch part {
						// Endianness
						case "big":
							result.Endianness = "big"
						case "little":
							result.Endianness = "little"
						case "native":
							result.Endianness = funbit.GetNativeEndianness()
						// Signedness
						case "signed":
							result.Signed = true
						case "unsigned":
							result.Signed = false
						// Types
						case "integer":
							result.Type = "integer"
						case "binary":
							result.Type = "binary"
						case "float":
							result.Type = "float"
						case "bitstring", "bits":
							result.Type = "bitstring"
						case "utf8":
							result.Type = "utf8"
						case "utf16":
							result.Type = "utf16"
						case "utf32":
							result.Type = "utf32"
						case "utf":
							result.Type = "utf"
						default:
							return result, fmt.Errorf("unknown component in compound specifier: %s", part)
						}
					}
				}
			} else {
				// Handle simple specifiers
				switch spec {
				case "signed":
					result.Signed = true
				case "unsigned":
					result.Signed = false
				case "big", "big-endian":
					result.Endianness = "big"
				case "little", "little-endian":
					result.Endianness = "little"
				case "native", "native-endian":
					result.Endianness = funbit.GetNativeEndianness()
				case "integer":
					result.Type = "integer"
				case "float":
					result.Type = "float"
				case "binary", "bytes":
					result.Type = "binary"
				case "bitstring", "bits":
					result.Type = "bitstring"
				case "utf8":
					result.Type = "utf8"
				case "utf16":
					result.Type = "utf16"
				case "utf32":
					result.Type = "utf32"
				case "utf":
					result.Type = "utf"
				default:
					return result, fmt.Errorf("unknown specifier: %s", spec)
				}
			}
		}
	}

	return result, nil
}

// calculatePatternSize calculates the expected size of a pattern in bits
func (fa *FunbitAdapter) calculatePatternSize(patternExpr *ast.BitstringExpression) uint {
	totalSize := uint(0)

	for _, segment := range patternExpr.Segments {
		// Skip rest patterns - they don't contribute to fixed size
		if ident, ok := segment.Value.(*ast.Identifier); ok && ident.Name == "rest" {
			continue
		}
		// Skip binary segments without size - they are also rest patterns
		if segment.Size == nil && len(segment.Specifiers) > 0 {
			isBinaryRest := false
			for _, spec := range segment.Specifiers {
				if spec == "binary" {
					isBinaryRest = true
					break
				}
			}
			if isBinaryRest {
				continue
			}
		}

		// Calculate segment size
		segmentSize := uint(8) // Default size
		if segment.Size != nil {
			if sizeValue, err := fa.convertValue(segment.Size); err == nil {
				switch v := sizeValue.(type) {
				case int64:
					segmentSize = uint(v)
				case float64:
					segmentSize = uint(v)
				case int:
					segmentSize = uint(v)
				}
			}
		}

		// Apply unit multiplier if present and handle UTF types
		if len(segment.Specifiers) > 0 {
			if specs, err := fa.parseSpecifiers(segment.Specifiers); err == nil {
				// For UTF types, skip size calculation - they are dynamic
				if specs.Type == "utf8" || specs.Type == "utf16" || specs.Type == "utf32" || specs.Type == "utf" {
					if segment.Size == nil {
						// UTF without explicit size is dynamic - skip in size calculation
						continue
					}
					// UTF with explicit size - use the specified size
				} else if specs.Unit > 0 {
					segmentSize *= uint(specs.Unit)
				}
			}
		}

		totalSize += segmentSize
	}

	return totalSize
}

// hasRestPattern checks if the pattern contains any rest patterns
func (fa *FunbitAdapter) hasRestPattern(patternExpr *ast.BitstringExpression) bool {
	for _, segment := range patternExpr.Segments {
		// Explicit rest pattern
		if ident, ok := segment.Value.(*ast.Identifier); ok && ident.Name == "rest" {
			return true
		}
		// Binary segment without size is also a rest pattern
		if segment.Size == nil && len(segment.Specifiers) > 0 {
			for _, spec := range segment.Specifiers {
				if spec == "binary" {
					return true
				}
			}
		}
	}
	return false
}

// hasDynamicSizes checks if the pattern contains any dynamic size expressions
func (fa *FunbitAdapter) hasDynamicSizes(patternExpr *ast.BitstringExpression) bool {
	for _, segment := range patternExpr.Segments {
		if segment.IsDynamicSize || segment.SizeExpression != nil {
			return true
		}

		// UTF patterns without explicit size are dynamic (variable length)
		if len(segment.Specifiers) > 0 {
			for _, spec := range segment.Specifiers {
				if spec == "utf8" || spec == "utf16" || spec == "utf32" || spec == "utf" {
					// UTF patterns are dynamic unless they have an explicit size
					if segment.Size == nil {
						return true
					}
				}
			}
		}
	}
	return false
}

// patternHasBinarySegments checks if the pattern contains any binary segments
func (fa *FunbitAdapter) patternHasBinarySegments(patternExpr *ast.BitstringExpression) bool {
	for _, segment := range patternExpr.Segments {
		if len(segment.Specifiers) > 0 {
			for _, spec := range segment.Specifiers {
				if spec == "binary" {
					return true
				}
			}
		}
	}
	return false
}

// MatchBitstringWithFunbit performs pattern matching using funbit API
// This function takes AST pattern and data bitstring, converts pattern to matcher, and returns variable bindings
func (fa *FunbitAdapter) MatchBitstringWithFunbit(patternExpr *ast.BitstringExpression, data *shared.BitstringObject) (map[string]interface{}, error) {
	if fa.verbose {
		fmt.Printf("DEBUG: MatchBitstringWithFunbit - input data size: %d bits, data: %s\n", data.BitString.Length(), funbit.ToBinaryString(data.BitString))
		fmt.Printf("DEBUG: MatchBitstringWithFunbit - pattern has %d segments\n", len(patternExpr.Segments))
	}

	// Convert AST pattern to funbit matcher
	matcher, variableNames, err := fa.convertASTPatternToMatcher(patternExpr)
	if err != nil {
		return nil, fmt.Errorf("failed to convert pattern to matcher: %v", err)
	}

	// Register variables for dynamic sizing if any are available
	if len(fa.variables) > 0 {
		if err := fa.registerVariables(matcher, fa.variables); err != nil {
			return nil, fmt.Errorf("failed to register variables for dynamic sizing: %v", err)
		}
	}

	// Calculate expected pattern size for exact matching
	expectedSize := fa.calculatePatternSize(patternExpr)

	// Execute the match
	results, err := funbit.Match(matcher, data.BitString)
	if err != nil {
		return nil, fmt.Errorf("pattern matching failed: %v", err)
	}

	// Check if pattern size matches data size exactly (unless there are rest patterns or dynamic sizes)
	if expectedSize > 0 && !fa.hasRestPattern(patternExpr) && !fa.hasDynamicSizes(patternExpr) {
		if uint(data.Len()) != expectedSize {
			return nil, fmt.Errorf("pattern size mismatch: expected %d bits, got %d bits", expectedSize, data.Len())
		}
	}

	// Validate constant values before creating bindings
	err = fa.validateConstantValues(patternExpr, results, variableNames)
	if err != nil {
		return nil, fmt.Errorf("constant validation failed: %v", err)
	}

	// Convert results to bindings map with proper type handling
	resultBindings := make(map[string]interface{})
	for i, result := range results {
		if i < len(variableNames) && variableNames[i] != "" && result.Matched {
			// Get the specifiers for this segment to determine type conversion
			if i < len(patternExpr.Segments) {
				segment := &patternExpr.Segments[i]
				if len(segment.Specifiers) > 0 {
					specs, _ := fa.parseSpecifiers(segment.Specifiers)

					// Apply type-specific conversions for UTF types
					if specs.Type == "utf8" || specs.Type == "utf16" || specs.Type == "utf32" || specs.Type == "utf" {
						var str string
						if strPtr, ok := result.Value.(*string); ok {
							str = *strPtr
						} else if s, ok := result.Value.(string); ok {
							str = s
						} else if bytes, ok := result.Value.([]byte); ok {
							str = string(bytes)
						} else {
							resultBindings[variableNames[i]] = result.Value
							continue
						}

						if len(str) > 0 {
							if segment.Size != nil {
								// For sized UTF, return codepoint
								r, _ := utf8.DecodeRuneInString(str)
								resultBindings[variableNames[i]] = int(r)
							} else {
								// For unsized UTF, return the string
								resultBindings[variableNames[i]] = str
							}
						} else {
							resultBindings[variableNames[i]] = ""
						}
					} else if specs.Type == "binary" {
						// Convert byte slice to string for binary types
						if bytes, ok := result.Value.([]byte); ok {
							resultBindings[variableNames[i]] = string(bytes)
						} else if bytes, ok := result.Value.([]uint8); ok {
							// Handle []uint8 from funbit rest patterns
							resultBindings[variableNames[i]] = string(bytes)
						} else {
							resultBindings[variableNames[i]] = result.Value
						}
					} else if specs.Type == "float" {
						// Use float values as-is without special formatting
						if floatValue, ok := result.Value.(float64); ok {
							resultBindings[variableNames[i]] = floatValue
						} else {
							resultBindings[variableNames[i]] = result.Value
						}
					} else if specs.Type == "bitstring" {
						// Convert *funbit.BitString to *shared.BitstringObject for proper Lua handling
						if bitstring, ok := result.Value.(*funbit.BitString); ok {
							resultBindings[variableNames[i]] = &shared.BitstringObject{BitString: bitstring}
						} else {
							resultBindings[variableNames[i]] = result.Value
						}
					} else if specs.Type == "integer" {
						// For explicit integer types, always treat as numbers, never as BitstringByte
						resultBindings[variableNames[i]] = result.Value
					} else {
						resultBindings[variableNames[i]] = result.Value
					}
				} else {
					// No specifiers - handle common types
					if bytes, ok := result.Value.([]uint8); ok {
						// Handle []uint8 from funbit rest patterns without explicit type
						resultBindings[variableNames[i]] = string(bytes)
					} else {
						// For values without explicit type specifiers, use as-is
						// BitstringByte conversion should only apply to explicit binary contexts
						resultBindings[variableNames[i]] = result.Value
					}
				}
			} else {
				resultBindings[variableNames[i]] = result.Value
			}
		}
	}

	return resultBindings, nil
}

// convertASTPatternToMatcher converts AST BitstringExpression to funbit matcher with variable names
func (fa *FunbitAdapter) convertASTPatternToMatcher(patternExpr *ast.BitstringExpression) (*funbit.Matcher, []string, error) {
	if fa.verbose {
		fmt.Printf("DEBUG: convertASTPatternToMatcher - processing %d segments\n", len(patternExpr.Segments))
	}
	matcher := funbit.NewMatcher()
	variableNames := make([]string, 0)

	// Clear and initialize storage for constant values that need to be validated
	fa.constantStorage = make(map[string]*int)

	// First pass: collect all variables that will be used for dynamic sizing
	dynamicSizeVars := make(map[string]*uint)
	for _, segment := range patternExpr.Segments {
		if segment.IsDynamicSize && segment.SizeExpression != nil {
			if segment.SizeExpression.ExprType == "variable" {
				// Simple variable reference
				varName := segment.SizeExpression.Variable
				if _, exists := dynamicSizeVars[varName]; !exists {
					// Create a variable to hold the dynamic size value
					dynamicSizeVars[varName] = new(uint)
					funbit.RegisterVariable(matcher, varName, dynamicSizeVars[varName])
				}
			} else if segment.SizeExpression.ExprType == "expression" && segment.SizeExpression.Variable != "" {
				// Expression - extract variable names (simple heuristic for "var-num" or "var+num")
				expr := segment.SizeExpression.Variable
				// Extract variables from expressions like "total-6", "size+1", etc.
				vars := fa.extractVariablesFromExpression(expr)
				for _, varName := range vars {
					if _, exists := dynamicSizeVars[varName]; !exists {
						// Create a variable to hold the dynamic size value
						dynamicSizeVars[varName] = new(uint)
						funbit.RegisterVariable(matcher, varName, dynamicSizeVars[varName])
					}
				}
			}
		}
	}

	// Second pass: process each segment in the pattern
	for _, segment := range patternExpr.Segments {
		varName, err := fa.addPatternSegmentWithSharedVars(matcher, &segment, dynamicSizeVars)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to add pattern segment: %v", err)
		}
		variableNames = append(variableNames, varName)
	}

	return matcher, variableNames, nil
}

// addPatternSegmentWithSharedVars adds a single pattern segment to the funbit matcher using shared variables
func (fa *FunbitAdapter) addPatternSegmentWithSharedVars(matcher *funbit.Matcher, segment *ast.BitstringSegment, dynamicSizeVars map[string]*uint) (string, error) {
	// Parse specifiers
	specs, err := fa.parseSpecifiers(segment.Specifiers)
	if err != nil {
		return "", fmt.Errorf("failed to parse specifiers: %v", err)
	}

	// Build options
	options := []funbit.SegmentOption{}
	var size uint
	if specs.Type != "" {
		options = append(options, funbit.WithType(specs.Type))
	}
	if specs.Signed {
		options = append(options, funbit.WithSigned(specs.Signed))
	}
	if specs.Endianness != "" {
		options = append(options, funbit.WithEndianness(specs.Endianness))
	}

	// Set default unit for binary types if not specified
	unit := specs.Unit
	if specs.Type == "binary" || specs.Type == "bytes" {
		if unit == 0 {
			unit = 8 // Default unit for binary is 8 (bytes)
		}
	} else if specs.Type == "utf8" || specs.Type == "utf16" || specs.Type == "utf32" || specs.Type == "utf" {
		if unit == 0 {
			unit = 1 // Default unit for UTF types is 1 (codepoint)
		}
	}
	if unit > 0 {
		options = append(options, funbit.WithUnit(unit))
	}

	// Handle size if present
	if segment.Size != nil {
		if segment.IsDynamicSize && segment.SizeExpression != nil {
			// Dynamic size - use funbit's WithDynamicSizeExpression
			if segment.SizeExpression.ExprType == "variable" {
				// Simple variable reference - use WithDynamicSizeExpression
				options = append(options, funbit.WithDynamicSizeExpression(segment.SizeExpression.Variable))
				// Size will be determined dynamically during matching
				size = 0 // Placeholder - actual size determined by funbit
			} else if segment.SizeExpression.ExprType == "expression" && segment.SizeExpression.Variable != "" {
				// String expression from parentheses (e.g., "total-6")
				options = append(options, funbit.WithDynamicSizeExpression(segment.SizeExpression.Variable))
				// Size will be determined dynamically during matching
				size = 0 // Placeholder - actual size determined by funbit
			} else {
				// Complex expressions - for now, try to resolve statically
				if s, err := fa.resolveDynamicSize(segment.SizeExpression, nil); err == nil {
					size = s
				} else {
					return "", fmt.Errorf("cannot resolve complex dynamic size in pattern matching: %v", err)
				}
			}
		} else {
			// Static size
			sizeValue, err := fa.convertValue(segment.Size)
			if err != nil {
				return "", fmt.Errorf("failed to convert size: %v", err)
			}
			switch v := sizeValue.(type) {
			case int64:
				size = uint(v)
			case float64:
				size = uint(v)
			case int:
				size = uint(v)
			default:
				return "", fmt.Errorf("unsupported size type: %T", sizeValue)
			}
		}
	}

	// Determine variable name and add appropriate pattern
	var varName string
	var varNameForDynamicSizing string // Имя для поиска в dynamicSizeVars
	switch value := segment.Value.(type) {
	case *ast.Identifier:
		if value.Qualified {
			// Квалифицированная переменная (например, lua.h)
			varName = value.Language + "." + value.Name
			varNameForDynamicSizing = value.Name // Для динамического размера используем только имя переменной
		} else {
			// Простая переменная
			varName = value.Name
			varNameForDynamicSizing = value.Name
		}

		// Handle different types based on specifiers
		switch specs.Type {
		case "integer", "":
			if segment.Size != nil && !segment.IsDynamicSize {
				options = append(options, funbit.WithSize(size))
			}

			// Check if this variable will be used for dynamic sizing
			var varValue uint
			var varPtr *uint
			if sharedVar, exists := dynamicSizeVars[varNameForDynamicSizing]; exists {
				// Use the shared variable for both result and dynamic sizing
				varPtr = sharedVar
			} else {
				// Use a local variable
				varPtr = &varValue
			}
			funbit.Integer(matcher, varPtr, options...)
		case "float":
			if segment.Size != nil && !segment.IsDynamicSize {
				options = append(options, funbit.WithSize(size))
			}
			var varValue float64
			funbit.Float(matcher, &varValue, options...)
		case "binary", "bytes":
			if segment.Size != nil && !segment.IsDynamicSize {
				// Static size - add WithSize option
				options = append(options, funbit.WithSize(size))
			}
			// For dynamic size, WithDynamicSizeExpression is already added above

			if segment.Size != nil {
				if fa.verbose {
					fmt.Printf("DEBUG: Adding Binary pattern segment - size: %d, options: %+v\n", size, options)
				}
				var varValue []byte
				funbit.Binary(matcher, &varValue, options...)
			} else {
				if fa.verbose {
					fmt.Printf("DEBUG: Adding RestBinary pattern segment - options: %+v\n", options)
				}
				var varValue []byte
				funbit.RestBinary(matcher, &varValue)
			}
		case "utf8", "utf16", "utf32", "utf":
			// For UTF types, extract as int (codepoint) following Erlang spec
			if fa.verbose {
				fmt.Printf("DEBUG: Adding UTF pattern segment - type: %s\n", specs.Type)
			}
			var varValue int // Extract as int (codepoint), not string

			// Use specific UTF functions for better compatibility
			switch specs.Type {
			case "utf8":
				funbit.UTF8(matcher, &varValue)
			case "utf16":
				funbit.UTF16(matcher, &varValue)
			case "utf32":
				funbit.UTF32(matcher, &varValue)
			default:
				funbit.UTF8(matcher, &varValue) // Default to UTF8
			}
		case "bitstring", "bits":
			if segment.Size != nil && !segment.IsDynamicSize {
				options = append(options, funbit.WithSize(size))
			}

			if segment.Size != nil {
				var varValue *funbit.BitString
				funbit.Bitstring(matcher, &varValue, options...)
			} else {
				// Rest bitstring - takes all remaining bits
				var varValue *funbit.BitString
				funbit.RestBitstring(matcher, &varValue)
			}
		default:
			return "", fmt.Errorf("unsupported type for pattern matching: %s", specs.Type)
		}

	case *ast.NumberLiteral:
		// For literal values, extract to variable and validate after matching
		varName = fmt.Sprintf("__const_%d_%p", int(value.Value), value) // Unique name for constant
		extractedValue := new(int)                                      // Create persistent storage
		fa.constantStorage[varName] = extractedValue

		// Add size option for constants (was missing!)
		if segment.Size != nil && !segment.IsDynamicSize {
			options = append(options, funbit.WithSize(size))
		}

		funbit.Integer(matcher, extractedValue, options...)

	case *ast.StringLiteral:
		// For literal strings, we don't need to bind to a variable
		varName = ""
		var literalValue []byte
		// Force binary type for string literals
		specs.Type = "binary"
		// Set size to match the entire string in bytes for binary type
		stringValue := segment.Value.(*ast.StringLiteral).Value
		stringSize := len(stringValue) // number of bytes

		// For string literals, use CLEAN options - only size
		literalOptions := []funbit.SegmentOption{funbit.WithSize(uint(stringSize))}
		if fa.verbose {
			fmt.Printf("DEBUG: Adding StringLiteral pattern segment - value: %q, size: %d bytes, CLEAN options: %+v\n", stringValue, stringSize, literalOptions)
		}
		funbit.Binary(matcher, &literalValue, literalOptions...)

	default:
		return "", fmt.Errorf("unsupported pattern value type: %T", value)
	}

	return varName, nil
}

// addPatternSegment adds a single pattern segment to the funbit matcher (legacy wrapper)
func (fa *FunbitAdapter) addPatternSegment(matcher *funbit.Matcher, segment *ast.BitstringSegment) (string, error) {
	// Call the new function with empty shared variables map
	return fa.addPatternSegmentWithSharedVars(matcher, segment, make(map[string]*uint))
}

// extractVariablesFromExpression extracts variable names from expressions like "total-6", "size+1"
func (fa *FunbitAdapter) extractVariablesFromExpression(expr string) []string {
	var variables []string

	// Simple regex-based extraction for common patterns
	// This is a heuristic approach - funbit will handle the actual evaluation

	// Remove spaces
	expr = strings.ReplaceAll(expr, " ", "")

	// Split by common operators: +, -, *, /, (, )
	parts := strings.FieldsFunc(expr, func(r rune) bool {
		return r == '+' || r == '-' || r == '*' || r == '/' || r == '(' || r == ')'
	})

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Check if it's a variable (starts with letter or underscore)
		if len(part) > 0 && (part[0] >= 'a' && part[0] <= 'z' || part[0] >= 'A' && part[0] <= 'Z' || part[0] == '_') {
			// It's likely a variable name
			variables = append(variables, part)
		}
		// Numbers are ignored - they're not variables
	}

	return variables
}

// registerVariables registers variables with the funbit matcher for dynamic sizing
func (fa *FunbitAdapter) registerVariables(matcher *funbit.Matcher, variables map[string]interface{}) error {
	for name, value := range variables {
		funbit.RegisterVariable(matcher, name, value)
	}
	return nil
}

// RegisterVariable registers a single variable for dynamic sizing
func (fa *FunbitAdapter) RegisterVariable(name string, value interface{}) {
	fa.variables[name] = value
}

// validateConstantValues validates that extracted values match expected constants
func (fa *FunbitAdapter) validateConstantValues(patternExpr *ast.BitstringExpression, results []funbit.SegmentResult, variableNames []string) error {
	for i, segment := range patternExpr.Segments {
		// Check if this segment is a constant (NumberLiteral)
		if numLit, ok := segment.Value.(*ast.NumberLiteral); ok {
			expectedValue := int(numLit.Value)

			// Find the corresponding variable name and stored value
			if i < len(variableNames) && variableNames[i] != "" {
				varName := variableNames[i]
				if extractedPtr, exists := fa.constantStorage[varName]; exists {
					actualValue := *extractedPtr

					// Validate that the extracted value matches the expected constant
					if actualValue != expectedValue {
						return fmt.Errorf("segment %d: expected constant %d, got %d", i, expectedValue, actualValue)
					}
				} else {
					return fmt.Errorf("segment %d: constant storage not found for variable %s", i, varName)
				}
			}
		}
	}
	return nil
}

// RegisterVariables registers multiple variables for dynamic sizing
func (fa *FunbitAdapter) RegisterVariables(variables map[string]interface{}) {
	for name, value := range variables {
		fa.variables[name] = value
	}
}

// ClearVariables clears all registered variables
func (fa *FunbitAdapter) ClearVariables() {
	fa.variables = make(map[string]interface{})
}

// resolveDynamicSize resolves a dynamic size expression at runtime
func (fa *FunbitAdapter) resolveDynamicSize(sizeExpr *ast.SizeExpression, bindings map[string]interface{}) (uint, error) {
	switch sizeExpr.ExprType {
	case "variable":
		// Simple variable reference
		if value, exists := fa.variables[sizeExpr.Variable]; exists {
			return fa.convertToUint(value)
		}
		if value, exists := bindings[sizeExpr.Variable]; exists {
			return fa.convertToUint(value)
		}
		return 0, fmt.Errorf("variable %s not found in registered variables or bindings", sizeExpr.Variable)

	case "expression":
		// Complex expression - need to evaluate it
		if fa.engine == nil {
			return 0, fmt.Errorf("cannot evaluate expressions without ExecutionEngine")
		}
		value, err := fa.engine.convertExpressionToValue(sizeExpr.Expression)
		if err != nil {
			return 0, fmt.Errorf("failed to evaluate size expression: %v", err)
		}
		return fa.convertToUint(value)

	case "literal":
		// Literal value
		value, err := fa.convertValue(sizeExpr.Literal)
		if err != nil {
			return 0, err
		}
		return fa.convertToUint(value)

	default:
		return 0, fmt.Errorf("unknown size expression type: %s", sizeExpr.ExprType)
	}
}

// convertToUint converts various types to uint for size values
func (fa *FunbitAdapter) convertToUint(value interface{}) (uint, error) {
	switch v := value.(type) {
	case int:
		if v < 0 {
			return 0, fmt.Errorf("size cannot be negative: %d", v)
		}
		return uint(v), nil
	case int64:
		if v < 0 {
			return 0, fmt.Errorf("size cannot be negative: %d", v)
		}
		return uint(v), nil
	case float64:
		if v < 0 {
			return 0, fmt.Errorf("size cannot be negative: %f", v)
		}
		if v != float64(int(v)) {
			return 0, fmt.Errorf("size must be an integer, got: %f", v)
		}
		return uint(v), nil
	case uint:
		return v, nil
	case uint64:
		return uint(v), nil
	default:
		return 0, fmt.Errorf("cannot convert %T to uint for size", value)
	}
}
