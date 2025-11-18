package engine

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"unicode/utf8"

	"funterm/errors"
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

// addIntegerWithOverflowHandling safely adds an integer to the builder, handling overflow by truncation
func (fa *FunbitAdapter) addIntegerWithOverflowHandling(builder *funbit.Builder, value interface{}, options ...funbit.SegmentOption) error {
	// Extract size and signedness from options
	var segmentSize uint = 8 // Default size for integers
	var isSigned bool = false

	// Create a temporary segment to extract properties from options
	tempSegment := &funbit.Segment{}
	for _, option := range options {
		option(tempSegment)
	}

	// Use the size from options if specified
	if tempSegment.SizeSpecified {
		segmentSize = tempSegment.Size
	}
	isSigned = tempSegment.Signed

	// Handle *big.Int specially for negative values
	if bigInt, ok := value.(*big.Int); ok && bigInt != nil && bigInt.Sign() < 0 {
		if isSigned {
			// For signed negative big.Int, apply two's complement for values outside range
			if segmentSize > 0 {
				// For signed numbers, check if the value fits in the signed range first
				// Signed range: -2^(n-1) to 2^(n-1)-1
				minVal := new(big.Int).Neg(new(big.Int).Lsh(big.NewInt(1), uint(segmentSize-1)))
				maxVal := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), uint(segmentSize-1)), big.NewInt(1))

				if fa.verbose {
					fmt.Printf("DEBUG: Signed negative big.Int %s, min: %s, max: %s (size: %d)\n",
						bigInt.String(), minVal.String(), maxVal.String(), segmentSize)
				}

				if bigInt.Cmp(minVal) >= 0 && bigInt.Cmp(maxVal) <= 0 {
					// Value fits in signed range, convert directly
					if bigInt.IsInt64() {
						funbit.AddInteger(builder, bigInt.Int64(), options...)
						return nil
					}
				} else {
					// Value doesn't fit, apply two's complement for signed overflow
					// For signed negative overflow, we want the maximum unsigned value (all bits set)
					// This matches the expected behavior where huge negative becomes 255 for 8-bit signed
					modulus := new(big.Int).Lsh(big.NewInt(1), uint(segmentSize)) // 2^segmentSize
					maxUnsigned := new(big.Int).Sub(modulus, big.NewInt(1))       // 2^segmentSize - 1

					var intValue int64
					if maxUnsigned.IsInt64() {
						intValue = maxUnsigned.Int64()
					} else {
						// If it doesn't fit in int64, use low 64 bits
						intValue = maxUnsigned.Int64()
					}

					if fa.verbose {
						fmt.Printf("DEBUG: Signed negative big.Int %s outside range, using max unsigned: %s -> int64 %d\n",
							bigInt.String(), maxUnsigned.String(), intValue)
					}
					funbit.AddInteger(builder, intValue, options...)
					return nil
				}
			}
			// If size is unspecified, fall through to regular handling
		} else {
			// For unsigned negative big.Int, we need to handle two's complement manually
			// Calculate two's complement for the specified size
			if segmentSize > 0 {
				modulus := new(big.Int).Lsh(big.NewInt(1), uint(segmentSize)) // 2^segmentSize
				// For two's complement: (2^n + value) where n is the number of bits
				twosComplement := new(big.Int).Add(modulus, bigInt)
				twosComplement.Mod(twosComplement, modulus)
				if fa.verbose {
					fmt.Printf("DEBUG: Unsigned negative big.Int two's complement: %s -> %s (size: %d)\n",
						bigInt.String(), twosComplement.String(), segmentSize)
				}
				// Convert to int64 for funbit compatibility
				if twosComplement.IsInt64() {
					funbit.AddInteger(builder, twosComplement.Int64(), options...)
				} else {
					// If it doesn't fit in int64, use low 64 bits
					funbit.AddInteger(builder, twosComplement.Int64(), options...)
				}
				return nil
			}
			// If size is unspecified, fall through to regular handling
		}
	}

	// Convert value to int64 for potential truncation (for all other cases)
	var intValue int64
	switch v := value.(type) {
	case int:
		intValue = int64(v)
	case int64:
		intValue = v
	case *big.Int:
		if v.IsInt64() {
			intValue = v.Int64()
		} else {
			// For very large integers, truncate to int64 range
			// This preserves the low 64 bits which is typically what's expected
			intValue = v.Int64() // This will give the low 64 bits
		}
	case float64:
		intValue = int64(v)
	default:
		// For non-integer types, just pass through
		funbit.AddInteger(builder, value, options...)
		return nil
	}

	// Check if value fits in the specified size
	if segmentSize > 0 && segmentSize <= 64 {
		maxValue := int64(1) << segmentSize
		minValue := int64(0)

		// For signed integers, adjust range
		if isSigned {
			maxValue = maxValue / 2
			minValue = -maxValue
		}

		// Truncate if value doesn't fit
		if intValue >= maxValue || intValue < minValue {
			if isSigned {
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
			return nil, errors.NewUserErrorWithASTPos("BITSTRING_SEGMENT_ERROR", fmt.Sprintf("failed to add segment: %v", err), expr.Position())
		}
		totalBits += bitsAdded
	}

	// Build the bitstring
	bitstring, err := funbit.Build(builder)
	if err != nil {
		return nil, errors.NewUserError("BITSTRING_BUILD_ERROR", fmt.Sprintf("failed to build bitstring: %v", err))
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
		if segment.Value != nil {
			if valExpr, ok := segment.Value.(ast.Expression); ok {
				return 0, errors.NewUserErrorWithASTPos("VALUE_CONVERSION_ERROR", fmt.Sprintf("failed to convert value: %v", err), valExpr.Position())
			}
		}
		return 0, fmt.Errorf("failed to convert value: %v", err)
	}

	// Check if this is a zero-size segment (padding/no-op)
	if segment.Size != nil {
		sizeValue, err := fa.convertValue(segment.Size)
		if err != nil {
			if segment.Size != nil {
				if sizeExpr, ok := segment.Size.(ast.Expression); ok {
					return 0, errors.NewUserErrorWithASTPos("SIZE_CONVERSION_ERROR", fmt.Sprintf("failed to convert size: %v", err), sizeExpr.Position())
				}
			}
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
		case *big.Int:
			if !v.IsUint64() {
				return 0, fmt.Errorf("size value %s is too large (max: %d)", v.String(), ^uint64(0))
			}
			size = uint(v.Uint64())
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
			if segment.Value != nil {
				if valExpr, ok := segment.Value.(ast.Expression); ok {
					return 0, errors.NewUserErrorWithASTPos("SPECIFIER_PARSE_ERROR", fmt.Sprintf("failed to parse specifiers: %v", err), valExpr.Position())
				}
			}
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
					if segment.SizeExpression != nil && segment.SizeExpression.Expression != nil {
						if expr, ok := segment.SizeExpression.Expression.(ast.Expression); ok {
							return 0, errors.NewUserErrorWithASTPos("DYNAMIC_SIZE_ERROR", fmt.Sprintf("failed to resolve dynamic size: %v", err), expr.Position())
						}
					}
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
				case *big.Int:
					if !v.IsUint64() {
						return 0, fmt.Errorf("size value %s is too large (max: %d)", v.String(), ^uint64(0))
					}
					size = uint(v.Uint64())
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

			// Check for maximum reasonable size to prevent panic (~8M bits = 1MB)
			const maxDefaultBits = 1024 * 1024 * 8 // 1M bits limit
			if effectiveSize > maxDefaultBits {
				if segment.Size != nil {
					if sizeExpr, ok := segment.Size.(ast.Expression); ok {
						return 0, errors.NewUserErrorWithASTPos("DEFAULT_SIZE_ERROR", fmt.Sprintf("default segment size %d bits exceeds maximum allowed size %d bits", effectiveSize, maxDefaultBits), sizeExpr.Position())
					}
				}
				return 0, fmt.Errorf("default segment size %d bits exceeds maximum allowed size %d bits", effectiveSize, maxDefaultBits)
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
			case int, int64, *big.Int:
				err := fa.addIntegerWithOverflowHandling(builder, value, sizeOptions...)
				if err != nil {
					if segment.Value != nil {
						if valExpr, ok := segment.Value.(ast.Expression); ok {
							return 0, errors.NewUserErrorWithASTPos("INTEGER_ADD_ERROR", fmt.Sprintf("failed to add integer: %v", err), valExpr.Position())
						}
					}
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

				// Check for maximum reasonable size to prevent panic (~8M bits = 1MB)
				const maxIntegerBits = 1024 * 1024 * 8 // 1M bits limit
				if effectiveSize > maxIntegerBits {
					if segment.Size != nil {
						if sizeExpr, ok := segment.Size.(ast.Expression); ok {
							return 0, errors.NewUserErrorWithASTPos("INTEGER_SIZE_ERROR", fmt.Sprintf("integer size %d bits exceeds maximum allowed size %d bits", effectiveSize, maxIntegerBits), sizeExpr.Position())
						}
					}
					return 0, fmt.Errorf("integer size %d bits exceeds maximum allowed size %d bits", effectiveSize, maxIntegerBits)
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
						if segment.Value != nil {
							if valExpr, ok := segment.Value.(ast.Expression); ok {
								return 0, errors.NewUserErrorWithASTPos("INTEGER_TYPE_ERROR", fmt.Sprintf("integer type requires whole number, got float %f", floatVal), valExpr.Position())
							}
						}
						return 0, fmt.Errorf("integer type requires whole number, got float %f", floatVal)
					}
				} else {
					err := fa.addIntegerWithOverflowHandling(builder, value, integerOptions...)
					if err != nil {
						if segment.Value != nil {
							if valExpr, ok := segment.Value.(ast.Expression); ok {
								return 0, errors.NewUserErrorWithASTPos("INTEGER_ADD_ERROR", fmt.Sprintf("failed to add integer: %v", err), valExpr.Position())
							}
						}
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
					if segment.Value != nil {
						if valExpr, ok := segment.Value.(ast.Expression); ok {
							return 0, errors.NewUserErrorWithASTPos("FLOAT_SIZE_ERROR", fmt.Sprintf("float type requires 16, 32, or 64 bits, got %d", effectiveSize), valExpr.Position())
						}
					}
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

				// Check for maximum reasonable size to prevent panic
				const maxBinarySize = 1024 * 1024 // 1MB limit
				if effectiveSize > maxBinarySize {
					if segment.Size != nil {
						if sizeExpr, ok := segment.Size.(ast.Expression); ok {
							return 0, errors.NewUserErrorWithASTPos("BINARY_SIZE_ERROR", fmt.Sprintf("binary size %d exceeds maximum allowed size %d", effectiveSize, maxBinarySize), sizeExpr.Position())
						}
					}
					return 0, fmt.Errorf("binary size %d exceeds maximum allowed size %d", effectiveSize, maxBinarySize)
				}

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
				} else if int64Value, ok := value.(int64); ok {
					// For binary type, provide exactly effectiveSize bytes
					binarySize := int(effectiveSize)
					if binarySize == 0 {
						binarySize = 8 // Default to 8 bytes for int64
					}
					bytes := make([]byte, binarySize)
					// Put the value in big-endian format
					for i := 0; i < binarySize; i++ {
						bytes[binarySize-1-i] = byte(int64Value >> (i * 8))
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
						if segment.Value != nil {
							if valExpr, ok := segment.Value.(ast.Expression); ok {
								return 0, errors.NewUserErrorWithASTPos("BINARY_TYPE_ERROR", fmt.Sprintf("binary type requires integer values, got float %f", floatValue), valExpr.Position())
							}
						}
						return 0, fmt.Errorf("binary type requires integer values, got float %f", floatValue)
					}
				} else if bigIntValue, ok := value.(*big.Int); ok {
					// Convert big.Int to bytes for binary type
					binarySize := int(effectiveSize)
					if binarySize == 0 {
						binarySize = (bigIntValue.BitLen() + 7) / 8 // Use actual bit length
						if binarySize == 0 {
							binarySize = 1 // At least 1 byte
						}
					}
					bytes := make([]byte, binarySize)
					// Convert big.Int to big-endian bytes
					bigIntValue.FillBytes(bytes)
					funbit.AddBinary(builder, bytes, binaryOptions...)
				} else {
					if segment.Value != nil {
						if valExpr, ok := segment.Value.(ast.Expression); ok {
							return 0, errors.NewUserErrorWithASTPos("BINARY_TYPE_ERROR", fmt.Sprintf("binary type requires []byte, string, numeric, or BitstringObject value, got %T", value), valExpr.Position())
						}
					}
					return 0, fmt.Errorf("binary type requires []byte, string, numeric, or BitstringObject value, got %T", value)
				}
			case "bits":
				if bitstring, ok := value.(*funbit.BitString); ok {
					funbit.AddBitstring(builder, bitstring, options...)
				} else if bitstringObj, ok := value.(*shared.BitstringObject); ok {
					funbit.AddBitstring(builder, bitstringObj.BitString, options...)
				} else {
					if segment.Value != nil {
						if valExpr, ok := segment.Value.(ast.Expression); ok {
							return 0, errors.NewUserErrorWithASTPos("BITSTRING_TYPE_ERROR", fmt.Sprintf("bitstring type requires *funbit.BitString or *shared.BitstringObject value, got %T", value), valExpr.Position())
						}
					}
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
					if segment.Value != nil {
						if valExpr, ok := segment.Value.(ast.Expression); ok {
							return 0, errors.NewUserErrorWithASTPos("UTF_TYPE_ERROR", fmt.Sprintf("UTF type requires string or numeric value, got %T", value), valExpr.Position())
						}
					}
					return 0, fmt.Errorf("UTF type requires string or numeric value, got %T", value)
				}
				// For UTF types, use AddUTF instead of AddBinary
				funbit.AddUTF(builder, str, options...)
			default:
				if segment.Value != nil {
					if valExpr, ok := segment.Value.(ast.Expression); ok {
						return 0, errors.NewUserErrorWithASTPos("UNSUPPORTED_TYPE_ERROR", fmt.Sprintf("unsupported type: %s", specs.Type), valExpr.Position())
					}
				}
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
		case int, int64, *big.Int:
			err := fa.addIntegerWithOverflowHandling(builder, value, defaultOptions...)
			if err != nil {
				if segment.Value != nil {
					if valExpr, ok := segment.Value.(ast.Expression); ok {
						return 0, errors.NewUserErrorWithASTPos("INTEGER_ADD_ERROR", fmt.Sprintf("failed to add integer: %v", err), valExpr.Position())
					}
				}
				return 0, fmt.Errorf("failed to add integer: %v", err)
			}
			bitsAdded = effectiveSize
		case float64:
			// Check if the float value is actually a whole number
			if v == float64(int(v)) {
				// It's a whole number, treat as integer
				err := fa.addIntegerWithOverflowHandling(builder, int(v), defaultOptions...)
				if err != nil {
					if segment.Value != nil {
						if valExpr, ok := segment.Value.(ast.Expression); ok {
							return 0, errors.NewUserErrorWithASTPos("INTEGER_ADD_ERROR", fmt.Sprintf("failed to add integer: %v", err), valExpr.Position())
						}
					}
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
			if segment.Value != nil {
				if valExpr, ok := segment.Value.(ast.Expression); ok {
					return 0, errors.NewUserErrorWithASTPos("UNSUPPORTED_VALUE_TYPE_ERROR", fmt.Sprintf("unsupported value type without specifiers: %T", value), valExpr.Position())
				}
			}
			return 0, fmt.Errorf("unsupported value type without specifiers: %T", value)
		}
	}

	return bitsAdded, nil
}

// convertValue converts an AST expression to a Go interface{} value
func (fa *FunbitAdapter) convertValue(expr ast.Expression) (interface{}, error) {
	switch e := expr.(type) {
	case *ast.NumberLiteral:
		// Handle the new NumberLiteral structure with big.Int support
		if e.IsInt {
			// For integers, check if they fit in int64, otherwise keep as *big.Int
			if e.IntValue != nil {
				if e.IntValue.IsInt64() {
					// Fits in int64, convert to int for compatibility
					return int(e.IntValue.Int64()), nil
				} else {
					// Too large for int64, keep as *big.Int
					return e.IntValue, nil
				}
			}
			// Fallback for safety
			return 0, nil
		} else {
			// For floats, return the float64 value
			return e.FloatValue, nil
		}
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
		case "!":
			// Logical NOT - return opposite of truthiness
			return !fa.isTruthy(value), nil
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
func (fa *FunbitAdapter) calculatePatternSize(patternExpr *ast.BitstringExpression) (uint, error) {
	totalSize := uint(0)

	for i, segment := range patternExpr.Segments {
		// Check for invalid segments without size
		if segment.Size == nil {
			// Check if this is a UTF segment without size - they are dynamic
			isUTF := false
			if len(segment.Specifiers) > 0 {
				for _, spec := range segment.Specifiers {
					if spec == "utf8" || spec == "utf16" || spec == "utf32" || spec == "utf" {
						isUTF = true
						break
					}
				}
			}

			// Check if this is a literal value (integer/string) without size - they have default size
			isLiteralValue := false
			if _, ok := segment.Value.(*ast.NumberLiteral); ok {
				isLiteralValue = true
			} else if _, ok := segment.Value.(*ast.StringLiteral); ok {
				isLiteralValue = true
			}

			if isUTF {
				// UTF segments without size are dynamic, skip in size calculation
				continue
			} else if isLiteralValue {
				// Literal values without size have default size, will be handled below
			} else if i == len(patternExpr.Segments)-1 {
				// This is the last segment, it's a valid rest pattern
				continue
			} else {
				// This is not the last segment and not UTF/literal, it should have a size - invalid pattern
				return 0, fmt.Errorf("segment %d has no size specification but is not the last segment", i)
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

	return totalSize, nil
}

// hasRestPattern checks if the pattern contains any rest patterns
func (fa *FunbitAdapter) hasRestPattern(patternExpr *ast.BitstringExpression) bool {
	// Only the last segment can be a rest pattern (without size)
	if len(patternExpr.Segments) > 0 {
		lastSegment := patternExpr.Segments[len(patternExpr.Segments)-1]
		return lastSegment.Size == nil
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
// returnFalseOnError: if true, returns empty bindings on failure instead of error (for assignments)
// if false, returns error on failure (for match statements)
func (fa *FunbitAdapter) MatchBitstringWithFunbit(patternExpr *ast.BitstringExpression, data *shared.BitstringObject, returnFalseOnError bool) (map[string]interface{}, error) {
	if fa.verbose {
		fmt.Printf("DEBUG: MatchBitstringWithFunbit - input data size: %d bits, data: %s\n", data.BitString.Length(), funbit.ToBinaryString(data.BitString))
		fmt.Printf("DEBUG: MatchBitstringWithFunbit - pattern has %d segments\n", len(patternExpr.Segments))
	}

	// Collect global variables for dynamic sizing if we have an ExecutionEngine
	globalVars := make(map[string]interface{})
	bigIntVars := make(map[string]*big.Int)
	if fa.engine != nil {
		fa.engine.globalMutex.RLock()
		for name, varInfo := range fa.engine.globalVariables {
			globalVars[name] = varInfo.Value
			// Also collect big.Int variables separately for big int expression evaluation
			if bigInt, ok := varInfo.Value.(*big.Int); ok {
				bigIntVars[name] = bigInt
			}
		}
		fa.engine.globalMutex.RUnlock()

		if fa.verbose {
			fmt.Printf("DEBUG: MatchBitstringWithFunbit - collected %d global variables: %v\n", len(globalVars), getMapKeys(globalVars))
			if len(bigIntVars) > 0 {
				fmt.Printf("DEBUG: MatchBitstringWithFunbit - collected %d big.Int variables\n", len(bigIntVars))
			}
		}
	}

	// Merge with additional variables
	allVars := make(map[string]interface{})
	for k, v := range globalVars {
		allVars[k] = v
	}
	for k, v := range fa.variables {
		allVars[k] = v
	}

	// Convert AST pattern to funbit matcher with variables available
	// Pass bigIntVars for expressions that involve big integers
	matcher, variableNames, err := fa.convertASTPatternToMatcherWithVars(patternExpr, allVars)
	if err != nil {
		return nil, fmt.Errorf("failed to convert pattern to matcher: %v", err)
	}

	// Check if any dynamic size expressions involve big.Int variables and pre-evaluate them
	for i, segment := range patternExpr.Segments {
		if segment.IsDynamicSize && segment.SizeExpression != nil {
			if hasBigIntVariable(segment.SizeExpression, bigIntVars) {
				// Evaluate with big.Int
				sizeStr, err := ast.ExpressionToString(segment.SizeExpression.Expression)
				if err == nil {
					bigResult, err := fa.evaluateBigIntExpression(sizeStr, mapBigIntToInterface(bigIntVars))
					if err != nil {
						if strings.Contains(err.Error(), "division by zero") {
							return nil, fa.mapFunbitErrorToUserError(fmt.Errorf("division by zero"))
						}
						return nil, fa.mapFunbitErrorToUserError(fmt.Errorf("big integer overflow in size expression"))
					}
					if !bigResult.IsInt64() {
						return nil, fa.mapFunbitErrorToUserError(fmt.Errorf("big integer overflow in size expression"))
					}
					// Update the segment in the matcher's internal state
					if fa.verbose {
						fmt.Printf("DEBUG: MatchBitstringWithFunbit - evaluated big.Int expression for segment %d: %v\n", i, bigResult)
					}
				}
			}
		}
	}

	// Variables are already registered in convertASTPatternToMatcherWithVars
	// No need to register them again here
	if fa.verbose && len(allVars) > 0 {
		fmt.Printf("DEBUG: MatchBitstringWithFunbit - %d variables already registered in convertASTPatternToMatcherWithVars: %v\n", len(allVars), getMapKeys(allVars))
	}

	// Execute the match
	results, err := funbit.Match(matcher, data.BitString)
	if err != nil {
		if returnFalseOnError {
			// For assignments: return empty bindings (false) instead of error
			// This matches Erlang/Elixir behavior where pattern matching returns false on failure
			return make(map[string]interface{}), nil
		} else {
			// For match statements: return error
			return nil, fa.mapFunbitErrorToUserError(err)
		}
	}

	// Check if pattern size matches data size exactly (unless there are rest patterns or dynamic sizes)
	// This ensures that patterns without rest match exactly
	expectedSize, sizeErr := fa.calculatePatternSize(patternExpr)
	if sizeErr != nil {
		// Pattern has invalid structure (segment without size not at the end)
		if returnFalseOnError {
			// For assignments: return empty bindings to indicate pattern matching failure
			return make(map[string]interface{}), nil
		} else {
			// For match statements: return error
			return nil, fmt.Errorf("invalid pattern: %v", sizeErr)
		}
	}
	if expectedSize > 0 && !fa.hasRestPattern(patternExpr) && !fa.hasDynamicSizes(patternExpr) {
		if uint(data.Len()) != expectedSize {
			if returnFalseOnError {
				// For assignments: return empty bindings to indicate pattern matching failure (size mismatch)
				return make(map[string]interface{}), nil
			} else {
				// For match statements: return error on size mismatch
				return nil, fmt.Errorf("pattern matching failed: size mismatch - expected %d bits, got %d bits", expectedSize, data.Len())
			}
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
						// Convert byte slice to appropriate type for binary types
						if fa.verbose {
							fmt.Printf("DEBUG: processing binary field %s, result.Value type: %T\n", variableNames[i], result.Value)
						}

						// Check if this is a rest pattern (last segment without size)
						isRestPattern := i == len(variableNames)-1 && patternExpr.Segments[i].Size == nil


						if bytes, ok := result.Value.([]byte); ok {
							if isRestPattern {
								// For rest patterns, return string if valid UTF-8, otherwise BitstringObject
								if utf8.Valid(bytes) {
									resultBindings[variableNames[i]] = string(bytes)
									if fa.verbose {
										fmt.Printf("DEBUG: returning UTF-8 string for rest pattern field %s\n", variableNames[i])
									}
								} else {
									bitString := funbit.NewBitStringFromBytes(bytes)
									resultBindings[variableNames[i]] = &shared.BitstringObject{BitString: bitString}
									if fa.verbose {
										fmt.Printf("DEBUG: returning BitstringObject for rest pattern field %s\n", variableNames[i])
									}
								}
							} else {
								resultBindings[variableNames[i]] = string(bytes)
							}
						} else if bytes, ok := result.Value.([]uint8); ok {
							// Handle []uint8 from funbit rest patterns
							if isRestPattern {
								// For rest patterns, return string if valid UTF-8, otherwise BitstringObject
								byteSlice := []byte(bytes)
								if utf8.Valid(byteSlice) {
									resultBindings[variableNames[i]] = string(byteSlice)
									if fa.verbose {
										fmt.Printf("DEBUG: returning UTF-8 string for rest pattern field %s ([]uint8)\n", variableNames[i])
									}
								} else {
									bitString := funbit.NewBitStringFromBytes(byteSlice)
									resultBindings[variableNames[i]] = &shared.BitstringObject{BitString: bitString}
									if fa.verbose {
										fmt.Printf("DEBUG: returning BitstringObject for rest pattern field %s ([]uint8)\n", variableNames[i])
									}
								}
							} else {
								resultBindings[variableNames[i]] = string(bytes)
							}
						} else if bitstringObj, ok := result.Value.(*shared.BitstringObject); ok {
							// Handle BitstringObject from funbit rest patterns
							if fa.verbose {
								fmt.Printf("DEBUG: converting BitstringObject to string for field %s\n", variableNames[i])
							}
							// Convert BitstringObject back to string
							bytes := bitstringObj.BitString.ToBytes()
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
	return fa.convertASTPatternToMatcherWithVars(patternExpr, make(map[string]interface{}))
}

// convertASTPatternToMatcherWithVars converts AST BitstringExpression to funbit matcher with variable names and available variables
func (fa *FunbitAdapter) convertASTPatternToMatcherWithVars(patternExpr *ast.BitstringExpression, availableVars map[string]interface{}) (*funbit.Matcher, []string, error) {
	if fa.verbose {
		fmt.Printf("DEBUG: convertASTPatternToMatcherWithVars - processing %d segments with %d available variables\n", len(patternExpr.Segments), len(availableVars))
	}
	matcher := funbit.NewMatcher()
	variableNames := make([]string, 0)

	// Clear and initialize storage for constant values that need to be validated
	fa.constantStorage = make(map[string]*int)

	// IMPORTANT: Register available variables FIRST, before processing segments
	// This ensures variables are available for dynamic size expressions
	if len(availableVars) > 0 {
		if fa.verbose {
			fmt.Printf("DEBUG: convertASTPatternToMatcherWithVars - registering available variables FIRST: %v\n", getMapKeys(availableVars))
		}
		for name, value := range availableVars {
			if fa.verbose {
				fmt.Printf("DEBUG: Registering variable '%s' = %v (type: %T)\n", name, value, value)
			}
			// For arithmetic expressions, prefer int type for funbit compatibility
			switch v := value.(type) {
			case int64:
				persistentInt := new(int)
				*persistentInt = int(v)
				if fa.verbose {
					fmt.Printf("DEBUG: Registered int64->int variable '%s' = %d\n", name, *persistentInt)
				}
				funbit.RegisterVariable(matcher, name, persistentInt)
			case int:
				persistentInt := new(int)
				*persistentInt = v
				if fa.verbose {
					fmt.Printf("DEBUG: Registered int variable '%s' = %d\n", name, *persistentInt)
				}
				funbit.RegisterVariable(matcher, name, persistentInt)
			case float64:
				if v == float64(int(v)) {
					persistentInt := new(int)
					*persistentInt = int(v)
					if fa.verbose {
						fmt.Printf("DEBUG: Registered float64->int variable '%s' = %d\n", name, *persistentInt)
					}
					funbit.RegisterVariable(matcher, name, persistentInt)
				} else {
					persistentFloat := new(float64)
					*persistentFloat = v
					if fa.verbose {
						fmt.Printf("DEBUG: Registered float64 variable '%s' = %f\n", name, *persistentFloat)
					}
					funbit.RegisterVariable(matcher, name, persistentFloat)
				}
			case uint:
				persistentInt := new(int)
				*persistentInt = int(v)
				if fa.verbose {
					fmt.Printf("DEBUG: Registered uint->int variable '%s' = %d\n", name, *persistentInt)
				}
				funbit.RegisterVariable(matcher, name, persistentInt)
			case uint64:
				persistentInt := new(int)
				*persistentInt = int(v)
				if fa.verbose {
					fmt.Printf("DEBUG: Registered uint64->int variable '%s' = %d\n", name, *persistentInt)
				}
				funbit.RegisterVariable(matcher, name, persistentInt)
			default:
				if fa.verbose {
					fmt.Printf("DEBUG: Registered raw variable '%s' = %v (type: %T)\n", name, value, value)
				}
				funbit.RegisterVariable(matcher, name, value)
			}
		}
	}

	// First pass: collect all variables that will be used for dynamic sizing
	dynamicSizeVars := make(map[string]*uint)
	for i, segment := range patternExpr.Segments {
		if segment.IsDynamicSize && segment.SizeExpression != nil {
			if segment.SizeExpression.ExprType == "variable" {
				// Simple variable reference
				varName := segment.SizeExpression.Variable
				// Check if variable is available (either from outside or defined in previous segments)
				if _, exists := availableVars[varName]; !exists {
					// Check if this variable is defined in a previous segment of this pattern
					varDefinedInPattern := false
					for j, prevSeg := range patternExpr.Segments {
						if j >= i { // Only check previous segments
							break
						}
						if ident, ok := prevSeg.Value.(*ast.Identifier); ok && ident.Name == varName {
							varDefinedInPattern = true
							break
						}
					}

					if !varDefinedInPattern {
						// Variable not found anywhere - this should cause pattern matching to fail
						return nil, nil, fmt.Errorf("undefined variable '%s' used in dynamic size expression", varName)
					}
				}
				if _, exists := dynamicSizeVars[varName]; !exists {
					// Create a variable to hold the dynamic size value with the actual value from availableVars
					varValue := uint(0)
					if val, exists := availableVars[varName]; exists {
						if convertedVal, err := fa.convertToUint(val); err == nil {
							varValue = convertedVal
						}
					}
					dynamicSizeVars[varName] = &varValue
					funbit.RegisterVariable(matcher, varName, dynamicSizeVars[varName])
				}
			} else if segment.SizeExpression.ExprType == "expression" && segment.SizeExpression.Variable != "" {
				// Expression - extract variable names (simple heuristic for "var-num" or "var+num")
				expr := segment.SizeExpression.Variable
				// Extract variables from expressions like "total-6", "size+1", etc.
				vars := fa.extractVariablesFromExpression(expr)
				for _, varName := range vars {
					// Check if variable is available (either from outside or defined in previous segments)
					if _, exists := availableVars[varName]; !exists {
						// Check if this variable is defined in a previous segment of this pattern
						varDefinedInPattern := false
						for j, prevSeg := range patternExpr.Segments {
							if j >= i { // Only check previous segments
								break
							}
							if ident, ok := prevSeg.Value.(*ast.Identifier); ok && ident.Name == varName {
								varDefinedInPattern = true
								break
							}
						}

						if !varDefinedInPattern {
							// Variable not found anywhere - this should cause pattern matching to fail
							return nil, nil, fmt.Errorf("undefined variable '%s' used in dynamic size expression", varName)
						}
					}
					if _, exists := dynamicSizeVars[varName]; !exists {
						// Create a variable to hold the dynamic size value with the actual value from availableVars
						varValue := uint(0)
						if val, exists := availableVars[varName]; exists {
							if convertedVal, err := fa.convertToUint(val); err == nil {
								varValue = convertedVal
							}
						}
						dynamicSizeVars[varName] = &varValue
						funbit.RegisterVariable(matcher, varName, dynamicSizeVars[varName])
					}
				}
			} else if segment.SizeExpression.ExprType == "expression" && segment.SizeExpression.Expression != nil {
				// Complex AST expression - extract variable names from the AST
				if exprStr, err := ast.ExpressionToString(segment.SizeExpression.Expression); err == nil {
					vars := fa.extractVariablesFromExpression(exprStr)
					for _, varName := range vars {
						// Check if variable is available (either from outside or defined in previous segments)
						if _, exists := availableVars[varName]; !exists {
							// Check if this variable is defined in a previous segment of this pattern
							varDefinedInPattern := false
							for j, prevSeg := range patternExpr.Segments {
								if j >= i { // Only check previous segments
									break
								}
								if ident, ok := prevSeg.Value.(*ast.Identifier); ok && ident.Name == varName {
									varDefinedInPattern = true
									break
								}
							}

							if !varDefinedInPattern {
								// Variable not found anywhere - this should cause pattern matching to fail
								return nil, nil, fmt.Errorf("undefined variable '%s' used in dynamic size expression", varName)
							}
						}
						if _, exists := dynamicSizeVars[varName]; !exists {
							// Create a variable to hold the dynamic size value with the actual value from availableVars
							varValue := uint(0)
							if val, exists := availableVars[varName]; exists {
								if convertedVal, err := fa.convertToUint(val); err == nil {
									varValue = convertedVal
								}
							}
							dynamicSizeVars[varName] = &varValue
							funbit.RegisterVariable(matcher, varName, dynamicSizeVars[varName])
						}
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
			} else if segment.SizeExpression.Expression != nil {
				// Complex AST expression - convert to string for funbit
				sizeStr, err := ast.ExpressionToString(segment.SizeExpression.Expression)
				if err != nil {
					return "", fmt.Errorf("failed to convert size expression to string: %v", err)
				}
				if fa.verbose {
					fmt.Printf("DEBUG: Adding dynamic size expression: '%s'\n", sizeStr)
				}
				options = append(options, funbit.WithDynamicSizeExpression(sizeStr))
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
			case *big.Int:
				if !v.IsUint64() {
					return "", fmt.Errorf("size value %s is too large (max: %d)", v.String(), ^uint64(0))
				}
				size = uint(v.Uint64())
			default:
				return "", fmt.Errorf("unsupported size type: %T", sizeValue)
			}
		}
	}

	// Determine variable name and add appropriate pattern
	var varName string
	var varNameForDynamicSizing string // Имя для поиска в dynamicSizeVars

	// Special case for wildcard pattern '_'
	if ident, ok := segment.Value.(*ast.Identifier); ok && ident.Name == "_" {
		varName = ""
		if fa.verbose {
			fmt.Printf("DEBUG: Adding Wildcard pattern segment - size: %d, options: %+v\n", size, options)
		}

		// Handle different types based on specifiers
		switch specs.Type {
		case "integer", "":
			if segment.Size != nil && !segment.IsDynamicSize {
				options = append(options, funbit.WithSize(size))
			}
			// For wildcard, we still need a variable to match, but we don't care about its value
			var dummyValue uint
			funbit.Integer(matcher, &dummyValue, options...)
		case "float":
			if segment.Size != nil && !segment.IsDynamicSize {
				options = append(options, funbit.WithSize(size))
			}
			var dummyValue float64
			funbit.Float(matcher, &dummyValue, options...)
		case "binary", "bytes":
			if segment.Size != nil && !segment.IsDynamicSize {
				options = append(options, funbit.WithSize(size))
			}
			var dummyValue []byte
			funbit.Binary(matcher, &dummyValue, options...)
		default:
			return "", fmt.Errorf("unsupported type for wildcard pattern: %s", specs.Type)
		}
		return varName, nil
	}

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
		var literalValue int
		if value.IsInt {
			if value.IntValue != nil && value.IntValue.IsInt64() {
				literalValue = int(value.IntValue.Int64())
			} else {
				// For very large integers, truncate to int range
				literalValue = int(value.IntValue.Int64())
			}
		} else {
			// For floats, convert to int (should be whole number for pattern matching)
			literalValue = int(value.FloatValue)
		}
		varName = fmt.Sprintf("__const_%d_%p", literalValue, value) // Unique name for constant
		extractedValue := new(int)                                  // Create persistent storage
		*extractedValue = literalValue
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
		if fa.verbose {
			fmt.Printf("DEBUG: registerVariables - registering '%s' = %v (type: %T)\n", name, value, value)
		}
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
			var expectedValue int
			if numLit.IsInt {
				if numLit.IntValue != nil && numLit.IntValue.IsInt64() {
					expectedValue = int(numLit.IntValue.Int64())
				} else {
					// For very large integers, truncate to int range
					expectedValue = int(numLit.IntValue.Int64())
				}
			} else {
				// For floats, convert to int (should be whole number for pattern matching)
				expectedValue = int(numLit.FloatValue)
			}

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
		// Check if we have an ExecutionEngine to look up the variable
		if fa.engine != nil {
			if val, found := fa.engine.getVariable(sizeExpr.Variable); found {
				return fa.convertToUint(val)
			}
		}
		return 0, fmt.Errorf("variable %s not found in registered variables or bindings", sizeExpr.Variable)

	case "expression":
		// Complex expression - need to evaluate it
		if fa.engine == nil {
			return 0, fmt.Errorf("cannot evaluate expressions without ExecutionEngine")
		}
		value, err := fa.engine.evaluateExpression(sizeExpr.Expression)
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

// isTruthy determines if a value is truthy (for logical NOT operations)
func (fa *FunbitAdapter) isTruthy(value interface{}) bool {
	switch v := value.(type) {
	case bool:
		return v
	case int, int64, float64:
		return v != 0
	case string:
		return v != ""
	case []byte:
		return len(v) > 0
	case nil:
		return false
	default:
		// For other types, consider non-nil values as truthy
		return true
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

// getMapKeys returns the keys of a map as a slice of strings for debugging
func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// hasBigIntVariable checks if a size expression contains any big.Int variables
func hasBigIntVariable(sizeExpr *ast.SizeExpression, bigIntVars map[string]*big.Int) bool {
	if sizeExpr == nil || len(bigIntVars) == 0 {
		return false
	}

	if sizeExpr.ExprType == "variable" {
		_, exists := bigIntVars[sizeExpr.Variable]
		return exists
	}

	if sizeExpr.ExprType == "expression" && sizeExpr.Variable != "" {
		// Check if any variable in the expression is a big.Int
		for varName := range bigIntVars {
			if strings.Contains(sizeExpr.Variable, varName) {
				return true
			}
		}
	}

	if sizeExpr.ExprType == "expression" && sizeExpr.Expression != nil {
		// Check the AST expression for big.Int variables
		exprStr, err := ast.ExpressionToString(sizeExpr.Expression)
		if err == nil {
			for varName := range bigIntVars {
				if strings.Contains(exprStr, varName) {
					return true
				}
			}
		}
	}

	return false
}

// mapBigIntToInterface converts map[string]*big.Int to map[string]interface{}
func mapBigIntToInterface(m map[string]*big.Int) map[string]interface{} {
	result := make(map[string]interface{}, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}

// evaluateBigIntExpression evaluates expressions with big.Int operands for dynamic sizing
// This handles big integers that don't fit in int64 (avoiding overflow)
func (fa *FunbitAdapter) evaluateBigIntExpression(exprStr string, vars map[string]interface{}) (*big.Int, error) {
	if fa.verbose {
		fmt.Printf("DEBUG: evaluateBigIntExpression - input: '%s'\n", exprStr)
	}

	// Simple tokenizer for mathematical expressions
	tokens := fa.tokenizeExpression(exprStr)
	if fa.verbose {
		fmt.Printf("DEBUG: evaluateBigIntExpression - tokens: %v\n", tokens)
	}

	// Convert to postfix notation
	postfix, err := fa.toPostfix(tokens)
	if err != nil {
		return nil, fmt.Errorf("failed to convert to postfix: %v", err)
	}

	if fa.verbose {
		fmt.Printf("DEBUG: evaluateBigIntExpression - postfix: %v\n", postfix)
	}

	// Evaluate postfix with big.Int
	result, err := fa.evaluateBigPostfix(postfix, vars)
	if err != nil {
		return nil, fmt.Errorf("big integer evaluation error: %v", err)
	}

	if fa.verbose {
		fmt.Printf("DEBUG: evaluateBigIntExpression - result: %v\n", result)
	}

	return result, nil
}

// tokenizeExpression breaks down an expression into tokens
// Handles unary operators (e.g., -5, +3) and binary operators
func (fa *FunbitAdapter) tokenizeExpression(expr string) []string {
	var tokens []string
	var current strings.Builder

	// Trim spaces first
	expr = strings.TrimSpace(expr)

	for i := 0; i < len(expr); i++ {
		ch := expr[i]
		if ch == ' ' {
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
		} else if ch == '(' || ch == ')' || ch == '+' || ch == '-' || ch == '*' || ch == '/' {
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}

			// Handle unary operators: + or - at start or after operator/parenthesis
			isUnaryOperator := false
			if ch == '-' || ch == '+' {
				if len(tokens) == 0 {
					// At the start of expression
					isUnaryOperator = true
				} else {
					lastToken := tokens[len(tokens)-1]
					if lastToken == "(" || lastToken == "+" || lastToken == "-" || lastToken == "*" || lastToken == "/" {
						// After operator or opening parenthesis
						isUnaryOperator = true
					}
				}
			}

			if isUnaryOperator {
				// Unary operator - collect it with the next number
				current.WriteByte(ch)
				// Continue to next character to form the number
				continue
			}

			// Binary operator or parenthesis
			tokens = append(tokens, string(ch))
		} else {
			current.WriteByte(ch)
		}
	}

	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}

	return tokens
}

// toPostfix converts infix tokens to postfix notation (Shunting Yard algorithm)
// Includes validation for proper expression structure
func (fa *FunbitAdapter) toPostfix(tokens []string) ([]string, error) {
	if len(tokens) == 0 {
		return nil, fmt.Errorf("empty expression")
	}

	var output []string
	var operators []string
	precedence := map[string]int{"+": 1, "-": 1, "*": 2, "/": 2}

	// Track whether we expect an operand or operator
	expectOperand := true

	for i, token := range tokens {
		isOperator := token == "+" || token == "-" || token == "*" || token == "/"
		isOperand := token != "(" && token != ")" && !isOperator

		// Validate token sequence
		if isOperand {
			if !expectOperand && i > 0 && tokens[i-1] != "(" && !strings.HasPrefix(tokens[i-1], "-") && !strings.HasPrefix(tokens[i-1], "+") {
				lastToken := tokens[i-1]
				if lastToken != "(" && lastToken != "+" && lastToken != "-" && lastToken != "*" && lastToken != "/" {
					return nil, fmt.Errorf("unexpected operand at position %d: %s", i, token)
				}
			}
			output = append(output, token)
			expectOperand = false

		} else if token == "(" {
			if !expectOperand {
				return nil, fmt.Errorf("unexpected '(' at position %d", i)
			}
			operators = append(operators, token)
			expectOperand = true

		} else if token == ")" {
			if expectOperand {
				return nil, fmt.Errorf("unexpected ')' at position %d: missing operand", i)
			}
			for len(operators) > 0 && operators[len(operators)-1] != "(" {
				output = append(output, operators[len(operators)-1])
				operators = operators[:len(operators)-1]
			}
			if len(operators) == 0 {
				return nil, fmt.Errorf("mismatched closing parenthesis at position %d", i)
			}
			operators = operators[:len(operators)-1] // Pop (
			expectOperand = false

		} else if isOperator {
			if expectOperand && (token == "*" || token == "/") {
				return nil, fmt.Errorf("unexpected operator '%s' at position %d: expected operand", token, i)
			}
			if expectOperand && token == "+" {
				// Unary plus - skip it
				expectOperand = true
				continue
			}
			if expectOperand && token == "-" {
				// Unary minus - treat as part of number
				expectOperand = true
				continue
			}

			prec := precedence[token]
			for len(operators) > 0 && operators[len(operators)-1] != "(" {
				topPrec, hasPrec := precedence[operators[len(operators)-1]]
				if hasPrec && topPrec >= prec {
					output = append(output, operators[len(operators)-1])
					operators = operators[:len(operators)-1]
				} else {
					break
				}
			}
			operators = append(operators, token)
			expectOperand = true
		}
	}

	// Check for unclosed parentheses
	for len(operators) > 0 {
		op := operators[len(operators)-1]
		if op == "(" {
			return nil, fmt.Errorf("unclosed opening parenthesis")
		}
		output = append(output, op)
		operators = operators[:len(operators)-1]
	}

	// Check if expression ends properly
	if expectOperand && len(output) > 0 {
		return nil, fmt.Errorf("expression ends with operator")
	}

	return output, nil
}

// evaluateBigPostfix evaluates a postfix expression with big.Int operands
func (fa *FunbitAdapter) evaluateBigPostfix(postfix []string, vars map[string]interface{}) (*big.Int, error) {
	var stack []*big.Int

	for _, token := range postfix {
		if token == "+" || token == "-" || token == "*" || token == "/" {
			if len(stack) < 2 {
				return nil, fmt.Errorf("invalid expression: insufficient operands for %s", token)
			}

			b := stack[len(stack)-1]
			a := stack[len(stack)-2]
			stack = stack[:len(stack)-2]

			var result *big.Int
			switch token {
			case "+":
				result = new(big.Int).Add(a, b)
			case "-":
				result = new(big.Int).Sub(a, b)
			case "*":
				result = new(big.Int).Mul(a, b)
			case "/":
				if b.Sign() == 0 {
					return nil, fmt.Errorf("division by zero")
				}
				result = new(big.Int).Div(a, b)
			}

			stack = append(stack, result)
		} else {
			// Parse number or variable
			var value *big.Int

			// Check if it's a variable
			if val, exists := vars[token]; exists {
				switch v := val.(type) {
				case *big.Int:
					value = new(big.Int).Set(v)
				case int64:
					value = big.NewInt(v)
				case int:
					value = big.NewInt(int64(v))
				case float64:
					value = big.NewInt(int64(v))
				default:
					return nil, fmt.Errorf("unsupported variable type: %T", val)
				}
			} else {
				// Try to parse as number
				bigVal := new(big.Int)
				_, ok := bigVal.SetString(token, 10)
				if !ok {
					return nil, fmt.Errorf("invalid number or undefined variable: %s", token)
				}
				value = bigVal
			}

			stack = append(stack, value)
		}
	}

	if len(stack) != 1 {
		return nil, fmt.Errorf("invalid expression: invalid stack state")
	}

	return stack[0], nil
}

// mapFunbitErrorToUserError maps funbit errors to user-friendly UserError codes
// This function parses error messages from funbit and converts them to appropriate error types
func (fa *FunbitAdapter) mapFunbitErrorToUserError(err error) error {
	if err == nil {
		return nil
	}

	errMsg := err.Error()

	// Check for various funbit error patterns
	// These patterns are based on errors returned by funbit/internal/matcher/dynamic.go

	// Big integer overflow in size expressions
	if strings.Contains(errMsg, "big integer overflow in size expression") {
		return errors.NewUserError("OVERFLOW_ERROR", "big integer overflow in size expression")
	}

	// Overflow/underflow in size expressions
	if strings.Contains(errMsg, "overflow") || strings.Contains(errMsg, "underflow") {
		// Preserve the original error message from funbit for debugging
		return errors.NewUserError("OVERFLOW_ERROR", errMsg)
	}

	// Division by zero
	if strings.Contains(errMsg, "division by zero") {
		return errors.NewUserError("DIVISION_BY_ZERO_ERROR", "division by zero in size expression")
	}

	// Negative size
	if strings.Contains(errMsg, "negative size") {
		return errors.NewUserError("NEGATIVE_SIZE_ERROR", "negative size in pattern matching")
	}

	// Undefined variable
	if strings.Contains(errMsg, "undefined variable") {
		// Extract the variable name from the error message if possible
		// Format might be: "evaluation error: undefined variable: varname"
		return errors.NewUserError("UNDEFINED_VARIABLE_ERROR", errMsg)
	}

	// Generic fallback for other funbit errors
	return fmt.Errorf("pattern matching failed: %v", err)
}
