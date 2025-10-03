package main

import (
	"fmt"
	"log"

	"github.com/funvibe/funbit/pkg/funbit"
)

func main() {
	fmt.Println("=== Funbit Public API Examples ===")
	fmt.Println("Real-world examples from Funterm integration")
	fmt.Println()

	builderAPIErrorHandlingExample()

	// Core functionality
	basicConstructionAndMatching()
	dataTypesAndSpecifiers()
	endiannessExample()

	// Advanced features
	dynamicSizingExample()
	utfHandlingExample()
	utfCodepointExample()
	signednessExample()
	bitstringExample()
	stringLiteralsInPatterns()
	complexProtocolExample()
	typeSemanticExample()
	integrationPatterns()

	// Unit multipliers and compound specifiers
	unitMultipliersExample()
	compoundSpecifiersExample()
	floatHandlingExample()

	fmt.Println("=== All Examples Completed Successfully ===")
}

// Example 1: Basic Construction and Pattern Matching
func basicConstructionAndMatching() {
	fmt.Println("1. Basic Construction and Pattern Matching:")
	fmt.Println("   Erlang: <<42:8, 17:8, \"hello\">>")

	// Construction
	builder := funbit.NewBuilder()
	funbit.AddInteger(builder, 42, funbit.WithSize(8))
	funbit.AddInteger(builder, 17, funbit.WithSize(8))
	funbit.AddBinary(builder, []byte("hello"))

	bitstring, err := funbit.Build(builder)
	if err != nil {
		log.Fatalf("Failed to build: %v", err)
	}

	fmt.Printf("   Created bitstring: %d bits\n", bitstring.Length())

	// Pattern Matching
	matcher := funbit.NewMatcher()
	var first, second int
	var text []byte

	funbit.Integer(matcher, &first, funbit.WithSize(8))
	funbit.Integer(matcher, &second, funbit.WithSize(8))
	funbit.Binary(matcher, &text)

	results, err := funbit.Match(matcher, bitstring)
	if err == nil && len(results) > 0 {
		fmt.Printf("   Matched: first=%d, second=%d, text=%s\n", first, second, string(text))
	}
	fmt.Println()
}

// Example 2: Data Types and Specifiers
func dataTypesAndSpecifiers() {
	fmt.Println("2. Data Types and Specifiers:")

	// Integer types with different sizes
	builder := funbit.NewBuilder()
	funbit.AddInteger(builder, 1000, funbit.WithSize(16)) // 16-bit integer
	funbit.AddFloat(builder, 3.14, funbit.WithSize(32))   // 32-bit float
	funbit.AddBinary(builder, []byte("data"))             // Binary data

	bitstring, _ := funbit.Build(builder)
	fmt.Printf("   Mixed types bitstring: %d bits\n", bitstring.Length())

	// Pattern matching with types
	matcher := funbit.NewMatcher()
	var num int
	var pi float32
	var data []byte

	funbit.Integer(matcher, &num, funbit.WithSize(16))
	funbit.Float(matcher, &pi, funbit.WithSize(32))
	funbit.Binary(matcher, &data)

	results, err := funbit.Match(matcher, bitstring)
	if err == nil && len(results) > 0 {
		fmt.Printf("   Extracted: num=%d, pi=%.2f, data=%s\n", num, pi, string(data))
	}
	fmt.Println()
}

// Example 3: Endianness Support
func endiannessExample() {
	fmt.Println("3. Endianness Support:")

	value := 0x1234

	// Big-endian (default)
	builderBig := funbit.NewBuilder()
	funbit.AddInteger(builderBig, value, funbit.WithSize(16), funbit.WithEndianness("big"))
	bitstringBig, _ := funbit.Build(builderBig)

	// Little-endian
	builderLittle := funbit.NewBuilder()
	funbit.AddInteger(builderLittle, value, funbit.WithSize(16), funbit.WithEndianness("little"))
	bitstringLittle, _ := funbit.Build(builderLittle)

	fmt.Printf("   Value 0x%04X:\n", value)
	fmt.Printf("   Big-endian bytes:    %v\n", bitstringBig.ToBytes())
	fmt.Printf("   Little-endian bytes: %v\n", bitstringLittle.ToBytes())
	fmt.Println()
}

// Example 4: Dynamic Sizing with Variables and Expressions (Unit Multipliers!)
func dynamicSizingExample() {
	fmt.Println("4. Dynamic Sizing with Variables and Expressions:")
	fmt.Println("   CRITICAL: Binary segments default to unit:8 (bytes)")
	fmt.Println("   Use WithUnit(1) for bit-level precision!")

	// Create a packet: size field + data
	builder := funbit.NewBuilder()
	dataSize := 5 // bytes
	data := "Hello"

	funbit.AddInteger(builder, dataSize, funbit.WithSize(8))
	funbit.AddBinary(builder, []byte(data))
	funbit.AddBinary(builder, []byte(" World"))

	bitstring, err := funbit.Build(builder)
	if err != nil {
		log.Fatalf("Dynamic build failed: %v", err)
	}

	fmt.Printf("   Packet created: %d bits\n", bitstring.Length())

	// Pattern matching with dynamic sizes
	matcher := funbit.NewMatcher()
	var size int
	var payload []byte
	var rest []byte

	// First read the size
	funbit.Integer(matcher, &size, funbit.WithSize(8))

	// Register variable for use in expressions
	funbit.RegisterVariable(matcher, "size", &size)

	// CRITICAL FIX: Use WithUnit(1) to interpret size*8 as BITS, not BYTES
	// Without WithUnit(1): size*8 = 40, but binary interprets as 40*8 = 320 bits!
	// With WithUnit(1): size*8 = 40 bits exactly ✅
	// Note: Unit(1) for binary is VALID in Erlang for bit-level precision
	funbit.Binary(matcher, &payload, funbit.WithDynamicSizeExpression("size*8"), funbit.WithUnit(1))
	funbit.RestBinary(matcher, &rest)

	results, err := funbit.Match(matcher, bitstring)
	if err == nil && len(results) > 0 {
		fmt.Printf("   Extracted: size=%d, payload=%s, rest=%s\n", size, string(payload), string(rest))
		fmt.Printf("   Unit multiplier fix: WithUnit(1) ensures bit-level precision ✅\n")
	} else {
		fmt.Printf("   Match failed: %v\n", err)
	}
	fmt.Println()
}

// Example UTF: UTF Encoding and Extraction (Erlang Semantics)
func utfHandlingExample() {
	fmt.Println("UTF. UTF Encoding and Extraction:")
	fmt.Println("   Erlang UTF supports both: string encoding AND individual code point extraction")
	fmt.Println("   Encoding: Entire strings → UTF bytes")
	fmt.Println("   Decoding: Individual code points OR binary extraction")

	// UTF-8 example - encoding entire string
	builder := funbit.NewBuilder()
	text := "Hello"
	funbit.AddUTF8(builder, text) // Encodes entire string ✅

	bitstring, err := funbit.Build(builder)
	if err != nil {
		log.Fatalf("UTF build failed: %v", err)
	}

	fmt.Printf("   UTF-8 encoded '%s': %d bits\n", text, bitstring.Length())

	// Method 1: Extract as binary (for entire string)
	matcher1 := funbit.NewMatcher()
	var extractedBytes []byte
	funbit.RestBinary(matcher1, &extractedBytes)

	results, err := funbit.Match(matcher1, bitstring)
	if err == nil && len(results) > 0 {
		extracted := string(extractedBytes)
		fmt.Printf("   Method 1 - Binary extraction: '%s' ✅\n", extracted)
	}

	// Method 2: Extract individual UTF code points (Erlang way)
	// Note: This would extract one code point at a time
	fmt.Printf("   Method 2 - Code point extraction: Available for single characters\n")
	fmt.Printf("   Erlang semantics: Both approaches are valid! ✅\n")
	fmt.Println()
}

// Example Signedness: Signed vs Unsigned Integers (Erlang Semantics)
func utfCodepointExample() {
	fmt.Println("\n=== UTF Codepoint API ===")

	// Example 1: Single codepoint encoding (cleaner API)
	fmt.Println("\n1. Single Codepoint Encoding:")
	builder := funbit.NewBuilder()
	funbit.AddUTF8Codepoint(builder, 1024) // Equivalent to Erlang <<1024/utf8>>

	bitstring, err := funbit.Build(builder)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("   UTF-8 codepoint 1024 (Ѐ): %v (length: %d bits)\n",
		bitstring.ToBytes(), bitstring.Length())

	// Example 2: UTF-16 with emoji (requires surrogate pairs)
	fmt.Println("\n2. UTF-16 Emoji Encoding:")
	builder2 := funbit.NewBuilder()
	funbit.AddUTF16Codepoint(builder2, 0x1F680, funbit.WithEndianness("big")) // 🚀

	bitstring2, err := funbit.Build(builder2)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("   UTF-16 rocket emoji (🚀): %v (length: %d bits)\n",
		bitstring2.ToBytes(), bitstring2.Length())

	// Example 3: Decoding - Extract INTEGER codepoints (Erlang spec!)
	fmt.Println("\n3. UTF Decoding (INTEGER codepoints):")
	matcher := funbit.NewMatcher()
	var codepoint int // INTEGER variable (Erlang spec compliance!)
	funbit.UTF8(matcher, &codepoint)

	results, err := funbit.Match(matcher, bitstring)
	if err == nil && len(results) > 0 {
		fmt.Printf("   Extracted codepoint: %d (0x%X) ✅\n", codepoint, codepoint)
	}

	// Example 4: Error handling for invalid codepoints
	fmt.Println("\n4. Validation (prevents Erlang badarg errors):")
	builder3 := funbit.NewBuilder()
	funbit.AddUTF8Codepoint(builder3, 0xD800) // Invalid surrogate pair
	_, err = funbit.Build(builder3)
	if err != nil {
		fmt.Printf("   Expected error for invalid codepoint: %v ✅\n", err)
	}

	builder4 := funbit.NewBuilder()
	funbit.AddUTF32Codepoint(builder4, 0x110000) // Too large
	_, err = funbit.Build(builder4)
	if err != nil {
		fmt.Printf("   Expected error for too large codepoint: %v ✅\n", err)
	}

	// Example 5: Comparison - old vs new API
	fmt.Println("\n5. API Comparison:")
	fmt.Println("   Old: funbit.AddUTF8(builder, string(rune(1024)))  // Awkward")
	fmt.Println("   New: funbit.AddUTF8Codepoint(builder, 1024)      // Clean ✅")
}

func signednessExample() {
	fmt.Println("Signedness. Signed vs Unsigned Integers:")
	fmt.Println("   Erlang supports both signed and unsigned integer interpretation")
	fmt.Println("   Same bits, different semantic interpretation")

	// Construction with signed values
	builder := funbit.NewBuilder()

	// Positive number (works for both signed/unsigned)
	funbit.AddInteger(builder, 100, funbit.WithSize(8), funbit.WithSigned(false))

	// Negative number (requires signed interpretation)
	funbit.AddInteger(builder, -50, funbit.WithSize(8), funbit.WithSigned(true))

	// Large unsigned value (would be negative if interpreted as signed)
	funbit.AddInteger(builder, 200, funbit.WithSize(8), funbit.WithSigned(false))

	bitstring, err := funbit.Build(builder)
	if err != nil {
		log.Fatalf("Signedness build failed: %v", err)
	}

	fmt.Printf("   Built bitstring: %d bits\n", bitstring.Length())

	// Pattern matching with correct signedness
	matcher := funbit.NewMatcher()
	var unsigned1, signed, unsigned2 int

	funbit.Integer(matcher, &unsigned1, funbit.WithSize(8), funbit.WithSigned(false))
	funbit.Integer(matcher, &signed, funbit.WithSize(8), funbit.WithSigned(true))
	funbit.Integer(matcher, &unsigned2, funbit.WithSize(8), funbit.WithSigned(false))

	results, err := funbit.Match(matcher, bitstring)
	if err == nil && len(results) > 0 {
		fmt.Printf("   Extracted values:\n")
		fmt.Printf("   - Unsigned: %d (positive)\n", unsigned1)
		fmt.Printf("   - Signed: %d (negative)\n", signed)
		fmt.Printf("   - Unsigned: %d (large positive)\n", unsigned2)
		fmt.Printf("   Note: Same bit patterns, different interpretations ✅\n")
	}
	fmt.Println()
}

// Example 6: String Literals in Patterns (with Binary Size Semantics)
func stringLiteralsInPatterns() {
	fmt.Println("6. String Literals in Patterns:")
	fmt.Println("   Example: PNG header validation")
	fmt.Println("   IMPORTANT: Binary segments measure size in UNITS (default: bytes)")

	// Create PNG-like header
	builder := funbit.NewBuilder()
	funbit.AddInteger(builder, 13, funbit.WithSize(32))  // Length
	funbit.AddBinary(builder, []byte("IHDR"))            // Type
	funbit.AddInteger(builder, 100, funbit.WithSize(32)) // Width
	funbit.AddInteger(builder, 50, funbit.WithSize(32))  // Height
	funbit.AddInteger(builder, 8, funbit.WithSize(8))    // Bit depth

	bitstring, _ := funbit.Build(builder)

	// Pattern matching with string literal constants
	matcher := funbit.NewMatcher()
	var length, width, height, bitDepth int
	var chunkType []byte

	funbit.Integer(matcher, &length, funbit.WithSize(32))
	// CRITICAL: WithSize(4) for binary = 4 bytes = 32 bits
	// NOT WithSize(32) - that would be 32 bytes = 256 bits!
	funbit.Binary(matcher, &chunkType, funbit.WithSize(4)) // 4 bytes
	funbit.Integer(matcher, &width, funbit.WithSize(32))
	funbit.Integer(matcher, &height, funbit.WithSize(32))
	funbit.Integer(matcher, &bitDepth, funbit.WithSize(8))

	results, err := funbit.Match(matcher, bitstring)
	if err == nil && len(results) > 0 && string(chunkType) == "IHDR" {
		fmt.Printf("   Valid PNG header: %dx%d, %d-bit\n", width, height, bitDepth)
		fmt.Printf("   Binary size semantics: WithSize(4) = 4 bytes = 32 bits ✅\n")
	}
	fmt.Println()
}

// Example 7: Complex Protocol Parsing
func complexProtocolExample() {
	fmt.Println("7. Complex Protocol Parsing (TCP-like):")

	// Create TCP-like packet
	builder := funbit.NewBuilder()
	funbit.AddInteger(builder, 0x1234, funbit.WithSize(16))     // Source port
	funbit.AddInteger(builder, 0x5678, funbit.WithSize(16))     // Dest port
	funbit.AddInteger(builder, 0x12345678, funbit.WithSize(32)) // Sequence

	// Flags as individual bits
	funbit.AddInteger(builder, 1, funbit.WithSize(1)) // URG
	funbit.AddInteger(builder, 0, funbit.WithSize(1)) // ACK
	funbit.AddInteger(builder, 1, funbit.WithSize(1)) // PSH
	funbit.AddInteger(builder, 0, funbit.WithSize(1)) // RST
	funbit.AddInteger(builder, 1, funbit.WithSize(1)) // SYN
	funbit.AddInteger(builder, 0, funbit.WithSize(1)) // FIN
	funbit.AddInteger(builder, 0, funbit.WithSize(2)) // Reserved

	funbit.AddBinary(builder, []byte("payload"))

	bitstring, _ := funbit.Build(builder)

	// Pattern matching
	matcher := funbit.NewMatcher()
	var srcPort, dstPort, seq int
	var urg, ack, psh, rst, syn, fin, reserved int
	var payload []byte

	funbit.Integer(matcher, &srcPort, funbit.WithSize(16))
	funbit.Integer(matcher, &dstPort, funbit.WithSize(16))
	funbit.Integer(matcher, &seq, funbit.WithSize(32))
	funbit.Integer(matcher, &urg, funbit.WithSize(1))
	funbit.Integer(matcher, &ack, funbit.WithSize(1))
	funbit.Integer(matcher, &psh, funbit.WithSize(1))
	funbit.Integer(matcher, &rst, funbit.WithSize(1))
	funbit.Integer(matcher, &syn, funbit.WithSize(1))
	funbit.Integer(matcher, &fin, funbit.WithSize(1))
	funbit.Integer(matcher, &reserved, funbit.WithSize(2))
	funbit.RestBinary(matcher, &payload)

	results, err := funbit.Match(matcher, bitstring)
	if err == nil && len(results) > 0 {
		fmt.Printf("   TCP: %d→%d, seq=0x%08X\n", srcPort, dstPort, seq)
		fmt.Printf("   Flags: URG=%d ACK=%d PSH=%d RST=%d SYN=%d FIN=%d\n",
			urg, ack, psh, rst, syn, fin)
		fmt.Printf("   Payload: %s\n", string(payload))
	}
	fmt.Println()
}

// Example 8: Type Semantics and Display Logic
func typeSemanticExample() {
	fmt.Println("8. Type Semantics and Display Logic:")

	// Important: Default type is INTEGER, not binary
	builder := funbit.NewBuilder()
	funbit.AddInteger(builder, 42, funbit.WithSize(8))         // Integer (displays as 42)
	funbit.AddBinary(builder, []byte("A"), funbit.WithSize(1)) // Binary 1 byte = 8 bits

	bitstring, _ := funbit.Build(builder)

	// Pattern matching - semantics matter!
	matcher := funbit.NewMatcher()
	var number int  // Will be 42 (integer semantics)
	var char []byte // Will be 'A' (binary semantics)

	funbit.Integer(matcher, &number, funbit.WithSize(8)) // Extract as integer
	funbit.Binary(matcher, &char, funbit.WithSize(1))    // Extract as binary (1 byte)

	results, err := funbit.Match(matcher, bitstring)
	if err == nil && len(results) > 0 {
		fmt.Printf("   Integer value: %d (not ASCII character)\n", number)
		fmt.Printf("   Binary value: %s (ASCII character)\n", string(char))
		fmt.Println("   Key insight: Same bits, different semantics based on type specifier")
	}
	fmt.Println()
}

// Example 9: Integration Patterns for Runtime Systems
func integrationPatterns() {
	fmt.Println("9. Integration Patterns for Runtime Systems:")

	// Common pattern: Mixed integer/binary matching
	builder := funbit.NewBuilder()
	funbit.AddInteger(builder, 5, funbit.WithSize(8))    // Size field
	funbit.AddBinary(builder, []byte("hello"))           // Data field
	funbit.AddInteger(builder, 0xA9, funbit.WithSize(8)) // Status byte

	bitstring, _ := funbit.Build(builder)

	// Pattern: size:8, data:5/binary, status:8
	matcher := funbit.NewMatcher()
	var size, status int
	var data []byte

	funbit.Integer(matcher, &size, funbit.WithSize(8))
	funbit.Binary(matcher, &data, funbit.WithSize(5)) // 5 bytes
	funbit.Integer(matcher, &status, funbit.WithSize(8))

	results, err := funbit.Match(matcher, bitstring)
	if err == nil && len(results) > 0 {
		fmt.Printf("   Protocol fields: size=%d, data=%s, status=0x%02X\n",
			size, string(data), status)

		// Display logic considerations:
		fmt.Printf("   Integer 0xA9 = %d (decimal) = '%c' (if interpreted as ASCII)\n",
			status, rune(status))
		fmt.Println("   Runtime should display integers as numbers, binary as characters")
	}
	fmt.Println()
}

// Example 10: Unit Multipliers
func unitMultipliersExample() {
	fmt.Println("10. Unit Multipliers:")
	fmt.Println("   Example: size:8/integer-unit:16 means size*16 bits")

	// Construction with unit multipliers
	builder := funbit.NewBuilder()

	// 8-bit size field with unit:16 = 8*16 = 128 bits total
	funbit.AddInteger(builder, 8, funbit.WithSize(8), funbit.WithUnit(16))

	// 32-bit float with unit:2 = 32*2 = 64 bits (IEEE 754 double precision)
	funbit.AddFloat(builder, 3.14159265359, funbit.WithSize(32), funbit.WithUnit(2))

	// Binary data with explicit size (unit doesn't affect AddBinary size)
	funbit.AddBinary(builder, []byte("test"), funbit.WithSize(4)) // 4 bytes = 32 bits

	bitstring, err := funbit.Build(builder)
	if err != nil {
		log.Fatalf("Unit multiplier build failed: %v", err)
	}

	fmt.Printf("   Bitstring with unit multipliers: %d bits\n", bitstring.Length())

	// Example: Invalid unit validation
	fmt.Println("\n   Unit Validation Example:")
	builderInvalid := funbit.NewBuilder()
	funbit.AddBinary(builderInvalid, []byte("test"), funbit.WithSize(4), funbit.WithUnit(300)) // Invalid unit > 256
	_, err = funbit.Build(builderInvalid)
	if err != nil {
		fmt.Printf("   Expected error for invalid unit: %v\n", err)
	}

	// Pattern matching with unit multipliers
	matcher := funbit.NewMatcher()
	var sizeField int
	var floatValue float64
	var textData []byte

	funbit.Integer(matcher, &sizeField, funbit.WithSize(8), funbit.WithUnit(16))
	funbit.Float(matcher, &floatValue, funbit.WithSize(32), funbit.WithUnit(2))
	funbit.Binary(matcher, &textData, funbit.WithUnit(8))

	results, err := funbit.Match(matcher, bitstring)
	if err == nil && len(results) > 0 {
		fmt.Printf("   Extracted: size=%d (effective bits: %d), float=%.11f, text=%s\n",
			sizeField, sizeField*16, floatValue, string(textData))
	}
	fmt.Println()
}

// Example 11: Compound Specifiers
func compoundSpecifiersExample() {
	fmt.Println("11. Compound Specifiers:")
	fmt.Println("    Example: value:32/big-unsigned-integer (endianness + size)")

	// Construction with compound specifiers
	builder := funbit.NewBuilder()

	// 32-bit big-endian unsigned integer (no unit multiplier)
	funbit.AddInteger(builder, 0xDEADBEEF,
		funbit.WithSize(32),
		funbit.WithEndianness("big"))

	// 16-bit little-endian unsigned integer
	funbit.AddInteger(builder, 0x1234,
		funbit.WithSize(16),
		funbit.WithEndianness("little"))

	// Binary with native endianness
	funbit.AddBinary(builder, []byte("payload"),
		funbit.WithEndianness("native"))

	bitstring, err := funbit.Build(builder)
	if err != nil {
		log.Fatalf("Compound specifier build failed: %v", err)
	}

	fmt.Printf("   Compound specifier bitstring: %d bits\n", bitstring.Length())

	// Pattern matching with compound specifiers
	matcher := funbit.NewMatcher()
	var bigEndianValue, littleEndianValue int
	var payload []byte

	funbit.Integer(matcher, &bigEndianValue,
		funbit.WithSize(32),
		funbit.WithEndianness("big"))

	funbit.Integer(matcher, &littleEndianValue,
		funbit.WithSize(16),
		funbit.WithEndianness("little"))

	funbit.RestBinary(matcher, &payload)

	results, err := funbit.Match(matcher, bitstring)
	if err == nil && len(results) > 0 {
		fmt.Printf("   Big-endian: 0x%08X, Little-endian: 0x%04X\n",
			bigEndianValue, littleEndianValue)
		fmt.Printf("   Payload: %s\n", string(payload))
	}
	fmt.Println()
}

// Example 12: Advanced Float Handling
func floatHandlingExample() {
	fmt.Println("12. Advanced Float Handling:")
	fmt.Println("    IEEE 754 precision with unit multipliers")

	// Construction with different float precisions
	builder := funbit.NewBuilder()

	// 32-bit float (single precision)
	funbit.AddFloat(builder, 3.14159, funbit.WithSize(32))

	// 64-bit float (double precision) - direct size is clearer
	funbit.AddFloat(builder, 3.14159265359, funbit.WithSize(64))

	// 16-bit float (half precision) - if supported
	funbit.AddFloat(builder, 1.5, funbit.WithSize(16))

	bitstring, err := funbit.Build(builder)
	if err != nil {
		log.Fatalf("Float handling build failed: %v", err)
	}

	fmt.Printf("   Float bitstring: %d bits\n", bitstring.Length())

	// Pattern matching with proper float sizes
	matcher := funbit.NewMatcher()
	var float32Val float32
	var float64Val float64
	var float16Val float32 // Go doesn't have float16, use float32

	funbit.Float(matcher, &float32Val, funbit.WithSize(32))
	funbit.Float(matcher, &float64Val, funbit.WithSize(32), funbit.WithUnit(2))
	funbit.Float(matcher, &float16Val, funbit.WithSize(16))

	results, err := funbit.Match(matcher, bitstring)
	if err == nil && len(results) > 0 {
		fmt.Printf("   32-bit float: %.5f\n", float32Val)
		fmt.Printf("   64-bit float: %.11f\n", float64Val)
		fmt.Printf("   16-bit float: %.3f\n", float16Val)
		fmt.Println("   Note: Unit multipliers enable proper IEEE 754 double precision")
	}
	fmt.Println()
}

// Example: Builder API Error Handling Pattern
func builderAPIErrorHandlingExample() {
	fmt.Println("=== Builder API Error Handling Pattern ===")
	fmt.Println("   Chain operations, check error once at Build()")
	fmt.Println()

	// Example 1: Chain multiple operations
	fmt.Println("1. Chain Multiple Operations:")
	builder := funbit.NewBuilder()

	// Chain multiple operations without checking errors
	funbit.AddInteger(builder, 42, funbit.WithSize(8))
	funbit.AddUTF8Codepoint(builder, 0x1F680) // 🚀 rocket emoji
	funbit.AddFloat(builder, 3.14, funbit.WithSize(32))
	funbit.AddUTF16Codepoint(builder, 65, funbit.WithEndianness("big")) // 'A'

	// Error checked only once at the end
	bitstring, err := funbit.Build(builder)
	if err != nil {
		fmt.Printf("   Error: %v\n", err)
		return
	}

	fmt.Printf("   Built successfully: %d bits\n", bitstring.Length())
	fmt.Printf("   Data: %v\n", bitstring.ToBytes()[:min(8, len(bitstring.ToBytes()))])

	// Example 2: Error detection
	fmt.Println("\n2. Error Detection:")
	builder2 := funbit.NewBuilder()

	// These operations will succeed
	funbit.AddInteger(builder2, 100, funbit.WithSize(8))
	funbit.AddUTF8Codepoint(builder2, 65) // Valid: 'A'

	// This will set an error in the builder
	funbit.AddUTF8Codepoint(builder2, 0xD800) // Invalid surrogate pair

	// Subsequent operations are ignored (builder has error)
	funbit.AddInteger(builder2, 200, funbit.WithSize(8)) // Ignored
	funbit.AddFloat(builder2, 2.71, funbit.WithSize(32)) // Ignored

	// Error is returned by Build()
	_, err = funbit.Build(builder2)
	if err != nil {
		fmt.Printf("   Error caught: %v\n", err)
	}

	// Example 3: First error wins
	fmt.Println("\n3. First Error Wins:")
	builder3 := funbit.NewBuilder()

	funbit.AddUTF8Codepoint(builder3, -1)        // First error: negative codepoint
	funbit.AddUTF16Codepoint(builder3, 0x110000) // Second error: too large (ignored)
	funbit.AddUTF32Codepoint(builder3, 0xDFFF)   // Third error: surrogate (ignored)

	_, err = funbit.Build(builder3)
	if err != nil {
		fmt.Printf("   First error reported: %v\n", err)
	}

	// Example 4: API comparison
	fmt.Println("\n4. API Comparison:")
	fmt.Println("   Traditional approach:")
	fmt.Println("     builder := NewBuilder()")
	fmt.Println("     if err := AddSomething(builder, ...); err != nil { return err }")
	fmt.Println("     if err := AddAnother(builder, ...); err != nil { return err }")
	fmt.Println("     return Build(builder)")
	fmt.Println()
	fmt.Println("   Funbit approach:")
	fmt.Println("     builder := NewBuilder()")
	fmt.Println("     AddSomething(builder, ...)")
	fmt.Println("     AddAnother(builder, ...)")
	fmt.Println("     return Build(builder) // Error checked once")

	fmt.Printf("\n   Builder API: Chain operations, check error once")
	fmt.Println()
}

// Example: Non-byte-aligned Bitstrings
func bitstringExample() {
	fmt.Println("=== Non-byte-aligned Bitstrings ===")
	fmt.Println("   True bit-level operations (not just byte-aligned)")
	fmt.Println()

	// Example 1: Bit-level construction
	fmt.Println("1. Bit-level Construction:")
	builder := funbit.NewBuilder()

	// Build a 7-bit value (not byte-aligned)
	funbit.AddInteger(builder, 0b101, funbit.WithSize(3))  // 3 bits: 101
	funbit.AddInteger(builder, 0b1111, funbit.WithSize(4)) // 4 bits: 1111
	// Total: 7 bits = 1011111 (not a full byte)

	bitstring, err := funbit.Build(builder)
	if err != nil {
		fmt.Printf("   Error: %v\n", err)
		return
	}

	fmt.Printf("   Built 7-bit string: length=%d bits\n", bitstring.Length())
	fmt.Printf("   Binary representation: %08b\n", bitstring.ToBytes()[0]>>1) // Shift for 7 bits

	// Example 2: More complex bit combinations
	fmt.Println("\n2. Complex Bit Combinations:")
	builder2 := funbit.NewBuilder()

	funbit.AddInteger(builder2, 0b10, funbit.WithSize(2))  // 2 bits
	funbit.AddInteger(builder2, 0b111, funbit.WithSize(3)) // 3 bits
	funbit.AddInteger(builder2, 0b1, funbit.WithSize(1))   // 1 bit
	funbit.AddInteger(builder2, 0b00, funbit.WithSize(2))  // 2 bits
	// Total: 8 bits = 10111100

	bitstring2, err := funbit.Build(builder2)
	if err != nil {
		fmt.Printf("   Error: %v\n", err)
		return
	}

	fmt.Printf("   Built 8-bit string: length=%d bits\n", bitstring2.Length())
	fmt.Printf("   Binary: %08b\n", bitstring2.ToBytes()[0])

	// Example 3: Bit-level pattern matching
	fmt.Println("\n3. Bit-level Pattern Matching:")
	matcher := funbit.NewMatcher()
	var part1, part2, part3, part4 int

	funbit.Integer(matcher, &part1, funbit.WithSize(2))
	funbit.Integer(matcher, &part2, funbit.WithSize(3))
	funbit.Integer(matcher, &part3, funbit.WithSize(1))
	funbit.Integer(matcher, &part4, funbit.WithSize(2))

	results, err := funbit.Match(matcher, bitstring2)
	if err != nil {
		fmt.Printf("   Match error: %v\n", err)
		return
	}

	if len(results) > 0 {
		fmt.Printf("   Extracted: part1=%d, part2=%d, part3=%d, part4=%d\n",
			part1, part2, part3, part4)
		fmt.Printf("   Binary:    %02b,    %03b,    %01b,    %02b\n",
			part1, part2, part3, part4)
	}

	fmt.Println("   Note: Bitstrings can have any bit length, not just multiples of 8")
	fmt.Println()
}
