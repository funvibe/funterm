# Funbit - Erlang/OTP Bit Syntax Library for Go

Funbit is a comprehensive Go library that provides **full Erlang/OTP bit syntax compatibility** for working with bitstrings and binary data. It offers a fluent interface for both constructing and pattern matching bitstrings with advanced features.

## ✨ Key Features

- **🎯 True Erlang Compatibility**: Direct 1:1 mapping of Erlang bit syntax to Go API
- **⚡ True Bit-Level Operations**: Operates as a genuine bit stream, not byte-aligned segments  
- **🔢 Rich Data Types**: Integer, float (16/32/64-bit), binary, bitstring, UTF-8/16/32
- **📐 Dynamic & Expression-Based Sizing**: Variables and arithmetic expressions (`total-6`)
- **🔢 Unit Multipliers**: Size multiplication with `unit:N` (e.g., `32/float-unit:2` = 64-bit double)
- **🌐 Full Endianness Support**: Big, little, and native byte ordering
- **⚙️ Compound Specifiers**: Complex combinations (`32/big-unsigned-integer-unit:8`)
- **🎭 String Literals in Patterns**: Constants like `"IHDR":4/binary` for protocol validation
- **🏗️ Builder API Pattern**: Chain operations, check errors once
- **🧠 Proper Type Semantics**: Integers display as numbers, binary as characters
- **🛠️ Protocol-Ready**: Built for parsing real-world protocols (IPv4, TCP, PNG, etc.)
- **🚀 High Performance**: Optimized for both construction and pattern matching

## ⚠️ **CRITICAL SEMANTICS - READ THIS FIRST!**

### 🎯 **Binary vs Integer Size Semantics**
```go
// CRITICAL DIFFERENCE:
funbit.Integer(m, &val, funbit.WithSize(32))  // 32 BITS
funbit.Binary(m, &data, funbit.WithSize(32))  // 32 BYTES = 256 BITS!

// For binary segments: WithSize(N) means N UNITS (default: bytes)
funbit.Binary(m, &data, funbit.WithSize(4))   // 4 bytes ✅
funbit.Binary(m, &data, funbit.WithSize(32))  // 32 bytes ❌
```

### 🔧 **Unit Multipliers - Essential for Dynamic Sizing**
```go
// WITHOUT WithUnit(1): size*8 interpreted as BYTES, multiplied by 8!
funbit.Binary(m, &data, funbit.WithDynamicSizeExpression("size*8"))
// size=5 → 5*8=40, but binary interprets as 40*8=320 bits!

// WITH WithUnit(1): size*8 interpreted as exact BITS
funbit.Binary(m, &data, funbit.WithDynamicSizeExpression("size*8"), funbit.WithUnit(1))
// size=5 → 5*8=40 bits exactly ✅
```

### 🔤 **UTF Extraction - Erlang Semantics**
```go
// Erlang UTF supports BOTH approaches:

// 1. String encoding (entire strings)
funbit.AddUTF8(builder, "Hello")  // Encodes full string ✅

// 2. Code point extraction (individual characters)
funbit.UTF8(matcher, &codepoint)  // Single code point

// 3. Binary extraction (for full strings)
var bytes []byte
funbit.RestBinary(matcher, &bytes)
text := string(bytes)  // Full string ✅
```

## 🚀 Quick Start

### 🏗️ **Builder API Pattern**

Funbit uses a builder pattern for error handling:

```go
// Chain operations, check error once
builder := funbit.NewBuilder()
funbit.AddInteger(builder, 42, funbit.WithSize(8))
funbit.AddUTF8Codepoint(builder, 0x1F680) // 🚀 rocket emoji  
funbit.AddFloat(builder, 3.14, funbit.WithSize(32))

bitstring, err := funbit.Build(builder) // Error checked once
if err != nil {
    return err
}

// Traditional approach:
// if err := AddInteger(...); err != nil { return err }
// if err := AddUTF8Codepoint(...); err != nil { return err } 
// if err := AddFloat(...); err != nil { return err }
```

**Key Benefits:**
- **Chainable**: No error returns from `Add*` functions
- **Efficient**: First error stops processing, subsequent calls ignored
- **Clean**: Single error check at `Build()` time
- **Consistent**: Same pattern throughout the API

### Installation

```bash
go get github.com/funvibe/funbit
```

### Basic Construction and Matching

```go
package main

import (
    "fmt"
    "github.com/funvibe/funbit/pkg/funbit"
)

func main() {
    // Construction: <<42:8, "hello":5/binary>>
    builder := funbit.NewBuilder()
    funbit.AddInteger(builder, 42, funbit.WithSize(8))
    funbit.AddBinary(builder, []byte("hello"))
    
    bitstring, _ := funbit.Build(builder)
    
    // Pattern Matching: <<value:8, text:5/binary>>
    matcher := funbit.NewMatcher()
    var value int
    var text []byte
    
    funbit.Integer(matcher, &value, funbit.WithSize(8))
    // CRITICAL: Binary size in BYTES! WithSize(5) = 5 bytes = 40 bits
    funbit.Binary(matcher, &text, funbit.WithSize(5)) // 5 bytes ✅
    
    results, err := funbit.Match(matcher, bitstring)
    if err == nil && len(results) > 0 {
        fmt.Printf("Value: %d, Text: %s\n", value, string(text))
        // Output: Value: 42, Text: hello
    }
}
```

## 🏗️ Core Concepts

### Construction vs Pattern Matching

**Construction** builds bitstrings from values:
```go
builder := funbit.NewBuilder()
funbit.AddInteger(builder, 1000, funbit.WithSize(16))
funbit.AddFloat(builder, 3.14, funbit.WithSize(32))
bitstring, _ := funbit.Build(builder)
```

**Pattern Matching** extracts values from bitstrings:
```go
matcher := funbit.NewMatcher()
var num int
var pi float32
funbit.Integer(matcher, &num, funbit.WithSize(16))
funbit.Float(matcher, &pi, funbit.WithSize(32))
results, _ := funbit.Match(matcher, bitstring)
```

### Type Semantics (Important!)

Funbit follows Erlang semantics where **default type is integer**:

```go
// Construction
funbit.AddInteger(builder, 42, funbit.WithSize(8))     // Integer: displays as 42
funbit.AddBinary(builder, []byte("A"), funbit.WithSize(1))  // Binary: displays as 'A' (1 byte)

// Pattern Matching
funbit.Integer(matcher, &num, funbit.WithSize(8))      // Extract as number: 42
funbit.Binary(matcher, &char, funbit.WithSize(1))      // Extract as character: 'A' (1 byte)
```

**Key Rule**: Same bits, different semantics based on type specifier.

## 🔧 Advanced Features

### Non-byte-aligned Bitstrings

True bit-level operations (not just byte-aligned):

```go
// Build 7-bit value (not byte-aligned)
builder := funbit.NewBuilder()
funbit.AddInteger(builder, 0b101, funbit.WithSize(3))   // 3 bits
funbit.AddInteger(builder, 0b1111, funbit.WithSize(4))  // 4 bits
// Total: 7 bits (not a full byte)

bitstring, _ := funbit.Build(builder)

// Pattern matching bit-level
matcher := funbit.NewMatcher()
var part1, part2 int

funbit.Integer(matcher, &part1, funbit.WithSize(3))  // Extract 3 bits
funbit.Integer(matcher, &part2, funbit.WithSize(4))  // Extract 4 bits

results, _ := funbit.Match(matcher, bitstring)
// part1 = 5 (0b101), part2 = 15 (0b1111)
```

### UTF Codepoint API

Clean API for single codepoints:

```go
// Encoding single codepoints
builder := funbit.NewBuilder()
funbit.AddUTF8Codepoint(builder, 0x1F680)  // 🚀 emoji
funbit.AddUTF16Codepoint(builder, 0x1F31F, funbit.WithEndianness("big"))  // 🌟
funbit.AddUTF32Codepoint(builder, 65)  // 'A'

bitstring, err := funbit.Build(builder)
if err != nil {
    // Handles invalid codepoints (e.g., surrogate pairs)
    fmt.Printf("Error: %v\n", err)
}

// Extract as INTEGER (Erlang spec!)
matcher := funbit.NewMatcher()
var codepoint int
funbit.UTF8(matcher, &codepoint)  // Returns 0x1F680 (integer)

results, _ := funbit.Match(matcher, bitstring)
// codepoint = 128640 (0x1F680)
```

### Signed vs Unsigned Integers

```go
// Signed interpretation
builder := funbit.NewBuilder()
funbit.AddInteger(builder, -50, funbit.WithSize(8), funbit.WithSigned(true))

// Unsigned interpretation (default)
funbit.AddInteger(builder, 200, funbit.WithSize(8), funbit.WithSigned(false))

bitstring, _ := funbit.Build(builder)

// Pattern matching with signedness
matcher := funbit.NewMatcher()
var signedVal, unsignedVal int

funbit.Integer(matcher, &signedVal, funbit.WithSize(8), funbit.WithSigned(true))
funbit.Integer(matcher, &unsignedVal, funbit.WithSize(8), funbit.WithSigned(false))

results, _ := funbit.Match(matcher, bitstring)
// signedVal = -50, unsignedVal = 200
```

### Dynamic Sizing with Expressions

```go
// Create matcher and register variables
matcher := funbit.NewMatcher()
var headerSize int = 32
var total int = 96
funbit.RegisterVariable(matcher, "headerSize", &headerSize)
funbit.RegisterVariable(matcher, "total", &total)

// Use in expressions
funbit.AddBinary(builder, data, funbit.WithDynamicSizeExpression("total-headerSize"))
```

### String Literals in Patterns

Perfect for protocol validation:

```go
// Validate PNG header: expect exactly "IHDR"
matcher := funbit.NewMatcher()
var length int
expectedType := "IHDR"

funbit.Integer(matcher, &length, funbit.WithSize(32))
funbit.Binary(matcher, &expectedType, funbit.WithSize(4)) // Must match "IHDR" (4 bytes)
```

### Endianness Control

```go
// Big-endian (default)
funbit.AddInteger(builder, 0x1234, funbit.WithSize(16), funbit.WithEndianness("big"))

// Little-endian  
funbit.AddInteger(builder, 0x1234, funbit.WithSize(16), funbit.WithEndianness("little"))
```

### Unit Multipliers

Unit multipliers allow size multiplication for precise control:

```go
// 32-bit float with unit:2 = 64-bit IEEE 754 double precision
funbit.AddFloat(builder, 3.14159265359, funbit.WithSize(32), funbit.WithUnit(2))

// 8-bit size with unit:16 = 128 effective bits
funbit.AddInteger(builder, 8, funbit.WithSize(8), funbit.WithUnit(16))

// Pattern matching with unit multipliers
funbit.Float(matcher, &doubleValue, funbit.WithSize(32), funbit.WithUnit(2))
```

### Compound Specifiers

Combine multiple specifiers for complex data layouts:

```go
// 32/big-unsigned-integer-unit:8
funbit.AddInteger(builder, 0xDEADBEEF,
    funbit.WithSize(32),
    funbit.WithEndianness("big"),
    funbit.WithUnit(8))

// 16/little-unsigned-integer
funbit.AddInteger(builder, 0x1234,
    funbit.WithSize(16),
    funbit.WithEndianness("little"))
```

### Bit-Level Precision

```go
// Individual flag bits
funbit.AddInteger(builder, 1, funbit.WithSize(1))  // Single bit
funbit.AddInteger(builder, 0, funbit.WithSize(1))  // Another bit
funbit.AddInteger(builder, 3, funbit.WithSize(2))  // 2-bit value
```

## 🌟 Real-World Examples

### TCP Header Parsing

```go
builder := funbit.NewBuilder()
funbit.AddInteger(builder, 0x1234, funbit.WithSize(16))    // Source port
funbit.AddInteger(builder, 0x5678, funbit.WithSize(16))    // Dest port
funbit.AddInteger(builder, 1, funbit.WithSize(1))          // URG flag
funbit.AddInteger(builder, 0, funbit.WithSize(1))          // ACK flag
// ... more flags
funbit.AddBinary(builder, []byte("payload"))

// Pattern matching extracts all fields with proper types
```

### PNG Header Validation

```go
// Pattern: <<length:32, "IHDR":4/binary, width:32, height:32>>
matcher := funbit.NewMatcher()
var length, width, height int
expectedChunk := "IHDR"

funbit.Integer(matcher, &length, funbit.WithSize(32))
funbit.Binary(matcher, &expectedChunk, funbit.WithSize(4))  // Validates "IHDR" (4 bytes)
funbit.Integer(matcher, &width, funbit.WithSize(32))
funbit.Integer(matcher, &height, funbit.WithSize(32))
```

## 🎯 Best Practices

### 1. Understand Type Semantics
- Use `funbit.Integer()` for numeric values (displays as numbers)
- Use `funbit.Binary()` for text/character data (displays as characters)
- Default type is **integer**, not binary

### 2. Handle Dynamic Sizes Properly
```go
// Register all variables before use
funbit.RegisterVariable("size", 32)
funbit.RegisterVariable("total", 128)

// Use expressions for complex sizing
funbit.WithDynamicSizeExpression("total-size-8")
```

### 3. Validate Protocol Constants
```go
// Use string literals to validate protocol headers
expectedType := "IHDR"
funbit.Binary(matcher, &expectedType, funbit.WithSize(4))  // 4 bytes = "IHDR"
```

### 4. Use Unit Multipliers for Precision
```go
// For IEEE 754 double precision floats
funbit.AddFloat(builder, value, funbit.WithSize(32), funbit.WithUnit(2))  // 64-bit

// For size fields that represent bit counts
funbit.AddInteger(builder, 8, funbit.WithSize(8), funbit.WithUnit(16))    // 8*16 bits
```

### 5. Combine Specifiers for Complex Layouts
```go
// Full compound specifier
funbit.AddInteger(builder, value,
    funbit.WithSize(32),
    funbit.WithEndianness("big"),
    funbit.WithUnit(8))
```

### 6. Endianness Format (Important!)
```go
// ✅ CORRECT: Use short forms
funbit.WithEndianness("big")     // Supported
funbit.WithEndianness("little")  // Supported  
funbit.WithEndianness("native")  // Supported
```

### 7. Pattern Size Validation
Funbit automatically validates pattern sizes unless:
- Pattern contains rest patterns (`funbit.RestBinary()`)
- Pattern contains dynamic sizes
- Pattern contains string literals (for validation)
- Pattern contains unit multipliers (calculated dynamically)

## 📋 Erlang ↔ Funbit Syntax Reference

| Erlang Syntax | Funbit Equivalent | Description |
|---------------|-------------------|-------------|
| `<<42:8>>` | `funbit.AddInteger(b, 42, funbit.WithSize(8))` | 8-bit integer |
| `<<42:8/big>>` | `funbit.AddInteger(b, 42, funbit.WithSize(8), funbit.WithEndianness("big"))` | Big-endian integer |
| `<<3.14:32/float>>` | `funbit.AddFloat(b, 3.14, funbit.WithSize(32))` | 32-bit float |
| `<<"hello world"/binary>>` | `funbit.AddBinary(b, []byte("hello world"))` | Binary data (full) |
| `<<"hello world":5/binary>>` | `funbit.AddBinary(b, []byte("hello world"), funbit.WithSize(5))` | Binary data (truncated to 5 bytes: "hello") |
| `<<Size:8, Data:Size/binary>>` | `funbit.Integer(m, &size, funbit.WithSize(8))`<br>`funbit.Binary(m, &data, funbit.WithDynamicSizeExpression("size"))` | Dynamic sizing |
| `<<Value:16/unit:8>>` | `funbit.AddInteger(b, value, funbit.WithSize(16), funbit.WithUnit(8))` | Unit multiplier |
| `<<Codepoint/utf8>>` | `funbit.AddUTF8Codepoint(b, codepoint)` | UTF-8 codepoint |
| `<<"text"/utf8>>` | `funbit.AddUTF8(b, "text")` | UTF-8 string |
| `<<-50:8/signed>>` | `funbit.AddInteger(b, -50, funbit.WithSize(8), funbit.WithSigned(true))` | Signed integer |
| `<<Rest/binary>>` | `funbit.RestBinary(m, &rest)` | Rest pattern |
| `<<1:3, 15:4>>` | `funbit.AddInteger(b, 1, funbit.WithSize(3))`<br>`funbit.AddInteger(b, 15, funbit.WithSize(4))` | Non-byte-aligned |

**Key Differences:**
- Funbit uses explicit builder pattern vs Erlang's literal syntax
- Funbit requires variable registration for dynamic sizes
- Funbit supports method chaining and error accumulation
- Funbit provides stronger type safety with Go's type system

## 📊 Performance Notes

- **Construction**: O(n) where n is number of segments
- **Pattern Matching**: O(n) where n is number of segments  
- **Memory**: Bitstrings are immutable and memory-efficient
- **Threading**: Thread-safe for concurrent reads, builders are not thread-safe

## 🔍 Integration with Runtime Systems

When integrating with language runtimes (like Lua, JavaScript):

```go
// For integers extracted from patterns
if intValue >= 0 && intValue <= 255 {
    // In mixed binary context, might display as character
    // In pure integer context, display as number
}

// For binary data
string(binaryData) // Always display as characters
```

## 📚 More Examples

See `funbit/examples/public_api_example.go` for comprehensive examples covering:
- Basic construction and matching
- Data types and specifiers  
- Endianness support
- Dynamic sizing
- String literals in patterns
- Complex protocol parsing
- Unit multipliers
- Compound specifiers
- Advanced float handling
- Type semantics
- Integration patterns

## 🤝 Contributing

Contributions welcome!

## 📄 License

MIT License - see LICENSE file for details.