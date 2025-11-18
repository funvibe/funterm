# FunTerm

A REPL and scripting environment with advanced bitstring operations inspired by Erlang/OTP.

**Status:** Beta v0.2.0

## Table of Contents

- [Installation](#installation)
- [Quick Reference](#quick-reference)
- [Quick Start](#quick-start)
- [Best Practices](#best-practices)
- [Bitstring Operations](#bitstring-operations)
- [Real-World Example](#real-world-example)
- [More Examples](#more-examples)
- [Advanced Features](#advanced-features)
- [Language Integration](#language-integration)
- [Use Cases](#use-cases)
- [Examples & Resources](#examples--resources)

## Installation and Downloads

```bash
git clone https://github.com/funvibe/funterm
cd funterm
go build -o funterm main.go config.go batch.go
```

or download app for your OS:
[Funterm latest](https://github.com/funvibe/funterm/releases/latest)

then run it:
```bash
./funterm
```

**Requirements:**
- Go 1.20+
- Python 3.9+ (optional, for Python integration)
- Node.js 14+ (optional, for JavaScript integration)
- Lua 5.1+ (built-in, no installation needed)

## Quick Reference

### Data Types

| Type | Example | Description |
|------|---------|-------------|
| Number | `42`, `3.14`, `-5` | Integers and floats (supports big.Int) |
| String | `"hello"`, `'world'` | UTF-8 strings |
| Boolean | `true`, `false` | Logical values |
| Array | `[1, "a", true]` | Ordered collections |
| Map | `{"x": 10, "y": 20}` | Key-value dictionaries |
| Bitstring | `<<0xFF, 0x00>>` | Binary data |
| Nil | `nil` | Null/empty value |

### Operators (by precedence)

| Precedence | Operators | Description |
|------------|-----------|-------------|
| 1 (highest) | `()`, `[]`, `{}` | Grouping, indexing, map access |
| 2 | `@` | Size operator (bitstrings only) |
| 3 | `-`, `!`, `~` | Unary operators (negate, not, bitwise-not) |
| 4 | `**` | Power/exponentiation |
| 5 | `*`, `/`, `%` | Multiply, divide, modulo |
| 6 | `+`, `-` | Addition, subtraction |
| 7 | `++` | String/bitstring concatenation |
| 8 | `&` | Bitwise AND |
| 9 | `|` | Bitwise OR |
| 10 | `^` | Bitwise XOR |
| 11 | `<<`, `>>` | Bitwise shift (also used for bitstring literals) |
| 12 | `<`, `<=`, `>`, `>=` | Comparison |
| 13 | `==`, `!=` | Equality |
| 14 | `&&` | Logical AND |
| 15 | `\|\|` | Logical OR |
| 16 | `?:`, `?` | Elvis operator, ternary operator |
| 17 (lowest) | `=` | Assignment (all variables are mutable) |

### Built-in Functions

| Function | Usage | Returns | Example |
|----------|-------|---------|---------|
| `print()` | `print(value, ...)` | nil (prints to stdout) | `print("Hello", 42)` |
| `len()` | `len(array/string/map)` | number | `len("hello")` → `5` |
| `concat()` | `concat(array, array, ...)` | array | `concat([1,2], [3,4])` → `[1,2,3,4]` |
| `id()` | `id(value)` | value (identity function) | `id(42)` → `42` |
| `@` | `@bitstring` | number (size in bytes) | `@<<0xFF>>` → `1` |

### Bitstring Limits

| Limit | Value | Notes |
|-------|-------|-------|
| Max binary size | 1 MB | Hard limit for binary segments |
| Max integer bits | ~8M bits | For integer segments |
| Min segment size | 1 bit | Allows individual bit packing |
| Supported types | binary, integer, float, utf8 | See Advanced Features |

### Language Qualifiers

| Syntax | Language | Execution |
|--------|----------|-----------|
| `py.` | Python | External process (IPC) |
| `lua.` | Lua | Built-in runtime (fast) |
| `js.` | JavaScript | External Node.js process |
| `go.` | Go | Direct function calls |
| Plain | FunTerm | Native execution |

## Quick Start

### Basic Syntax

```python
# Variables and literals
x = 42                                          # Output: 42
y = 3.14                                        # Output: 3.14

# Arrays and dictionaries (collections)
data = [1, "a", true, nil, {"k": "v"}]        # Output: [1, a, true, nil, {k: v}]
config = {"host": "localhost", "port": 8080}  # Output: {host: localhost, port: 8080}

# Arithmetic operations (standard precedence)
result = (10 + 5) * 2                          # Output: 30
power = 2 ** 8                                 # Output: 256

# Bitwise operations (work on integers)
flags = 0x01 | 0x04 | 0x10                    # Output: 21 (binary: 0001 0101)
masked = 0xFF & 0x0F                          # Output: 15 (binary: 0000 1111)

# String concatenation
greeting = "Hello" ++ ", " ++ "World"         # Output: Hello, World
```

### Control Flow

```python
# Conditionals
if x > 10 {
    print("large")
} else if x > 0 {
    print("positive")
} else {
    print("non-positive")
}

# Numeric loop with optional step
# Note: iterates from 0 to 90 (10 iterations), since condition is i <= end
for i = 0, 100, 10 {
    print(i)  # Output: 0, 10, 20, 30, 40, 50, 60, 70, 80, 90
}

# Iterate over collection
for item in data {
    print(item)
}

# While loop
while counter < limit {
    counter = counter + 1
}

# C-style loop
for (i = 0; i < 10; i = i + 1) {
    print(i)
}
```

### Pattern Matching

```python
# Match on values
match status {
    200 -> print("OK"),
    404 -> print("Not Found"),
    500 -> print("Server Error"),
    _ -> print("Unknown")
}

# Match arrays
match list {
    [] -> print("empty"),
    [single] -> print("one element"),
    [first, second] -> print("two elements"),
    _ -> print("multiple elements")
}
```

## Bitstring Operations

### Construction

```python
# Basic bitstring creation
packet = <<0xFF, 0x00, 0xAB, 0xCD>>
message = <<"Hello, World!">>

# Specify sizes and types
header = <<
    0xDEADBEEF:32/big,      # 32-bit big-endian
    256:16/little,          # 16-bit little-endian
    3.14:32/float,          # 32-bit float
    "DATA"/binary           # Binary string
>>

# Individual bits
flags = <<1:1, 0:1, 1:1, 1:1, 0:4>>  # 8 bits total
```

### Pattern Matching on Bitstrings

```python
# Extract fixed-size fields
data = <<1, 2, 3, 4>>
match data {
    <<a:8, b:8, c:8, d:8>> -> {
        print("Values:", a, b, c, d)
    }
}

# Variable-sized fields
packet = <<5:8, "Hello":5/binary, " World">>
match packet {
    <<len:8, content:len/binary, rest/binary>> -> {
        print("Length:", len)
        print("Content:", content)
        print("Rest:", rest)
    }
}
```

### In-place Pattern Matching

Direct extraction without match blocks - a powerful feature for concise code:

```python
# Simple extraction
response = <<0x200:16, 13:16, "Hello, World!">>
<<status:16, length:16, body:length/binary>> = response
print("Status:", status, "Body:", body)

# Conditional in-place matching
if (<<0x89, "PNG", 0x0D, 0x0A, 0x1A, 0x0A, rest/binary>> = header) {
    print("Valid PNG file detected")
    process_png(rest)
}

# Extract UTF-8 codepoints directly
text = "Hello"
<<h/utf8, e/utf8, l1/utf8, l2/utf8, o/utf8>> = text
print("Codepoints:", h, e, l1, l2, o)

# With qualified variables
packet = <<0xDEADBEEF:32, 42:8, "data">>
<<lua.magic:32, lua.version:8, lua.payload/binary>> = packet
```

## Real-World Example: DNS Query Implementation

Complete working DNS client that queries real DNS servers:

```python
print("=== Real-World Example: DNS Query over UDP ===")

lua (encode_qname) {
    function encode_qname(domain)
        local parts = {}
        for part in string.gmatch(domain, "[^.]+") do
            table.insert(parts, string.char(#part) .. part)
        end
        return table.concat(parts) .. string.char(0)
    end
}

py (send_udp_query) {
    import socket

    def send_udp_query(dns_server, query_bytes):
        dns_port = 53
        sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        sock.settimeout(5)

        try:
            sock.sendto(query_bytes, (dns_server, dns_port))
            response_bytes, _ = sock.recvfrom(1024)
            return response_bytes
        except socket.timeout:
            return None
        finally:
            sock.close()
}

lua.domain_to_resolve = "lag.me"
print("Attempting to resolve domain:", lua.domain_to_resolve)

lua.qname = lua.encode_qname(lua.domain_to_resolve)

lua.dns_query_packet = <<
    0xDB42:16,
    0x0100:16,
    1:16,
    0:16,
    0:16,
    0:16,
    lua.qname,
    1:16,
    1:16
>>

lua.dns_response = py.send_udp_query("8.8.8.8", lua.dns_query_packet)

print("Parsing DNS response...")

match lua.dns_response {
    <<
        tid:16/big, flags:16/big, questions:16/big, answers:16/big, auth:16/big, add:16/big,
        qname_len:8, qname:qname_len/binary, tld_len:8, tld:tld_len/binary, 0:8,
        qtype:16/big, qclass:16/big,
        name_ptr:16/big,
        rtype:16/big, rclass:16/big, ttl:32/big, rdlength:16/big,
        ip1:8, ip2:8, ip3:8, ip4:8
    >> -> {
        print("DNS Response successfully parsed!")
        print("")
        print("Header:")
        print("  Transaction ID:", tid)
        print("  Flags:", flags)
        print("  Questions:", questions)
        print("  Answers:", answers)
        print("")
        print("Question:")
        print("  QTYPE:", qtype, "(A record)")
        print("  QCLASS:", qclass, "(IN - Internet)")
        print("")
        print("Answer:")
        print("  Name pointer: ", lua.string.format("0x%04X", name_ptr))
        print("  TYPE:", rtype, "(A record)")
        print("  CLASS:", rclass, "(IN - Internet)")
        print("  TTL:", ttl, "seconds")
        print("  RDLENGTH:", rdlength, "(should be 4 for A record)")
        print("  Domain:", lua.string.format("%s.%s", qname, tld))
        print("  IP Address:", lua.string.format("%d.%d.%d.%d", ip1, ip2, ip3, ip4))
        print("")
    }
}
```

## More Examples

### TCP Header Parsing

```python
tcp_header = <<
    0x1234:16,         # Source port
    0x5678:16,         # Destination port
    0x12345678:32,     # Sequence number
    0x87654321:32,     # Acknowledgment number
    5:4, 0:6,          # Header length and reserved
    0x18:8,            # Flags
    0x1000:16,         # Window size
    0x0000:16,         # Checksum
    0x0000:16          # Urgent pointer
>>

match tcp_header {
    <<src:16, dst:16, seq:32, ack:32,
      hlen:4, _:6, flags:8,
      window:16, checksum:16, urgent:16>> -> {
        print("TCP", src, "->", dst)
        print("Seq:", seq, "Ack:", ack)
        print("Flags:", (flags & 0x02) ? "SYN" : "",
                       (flags & 0x10) ? "ACK" : "")
    }
}
```

### File Format Detection

```python
# First, define a helper function to read file headers in Python
py (read_file_header) {
    def read_file_header(filename, num_bytes):
        try:
            with open(filename, 'rb') as f:
                return f.read(num_bytes)
        except Exception as e:
            return None
}

# Then use pattern matching to detect file type
filename = "example.bin"
header = py.read_file_header(filename, 16)

match header {
    <<0x89, "PNG", 0x0D, 0x0A, 0x1A, 0x0A, rest/binary>> -> {
        print("PNG image")
    },
    <<0xFF, 0xD8, 0xFF, rest/binary>> -> {
        print("JPEG image")
    },
    <<"PK", 0x03, 0x04, rest/binary>> -> {
        print("ZIP archive")
    },
    <<0x7F, "ELF", rest/binary>> -> {
        print("ELF executable")
    },
    _ -> print("Unknown format")
}
```

### Protocol Message Parsing

```python
stream = receive_data()
while @stream > 0 {
    <<msg_type:8, length:16/big, payload:length/binary, rest/binary>> = stream
    match msg_type {
        1 -> process_heartbeat(payload),
        2 -> process_data(payload),
        3 -> process_control(payload),
        _ -> print("Unknown message type:", msg_type)
    }
    stream = rest
}
```

## Advanced Features

### Endianness and Types

```python
# Endianness control
value = 0x1234
big = <<value:16/big>>        # Network byte order
little = <<value:16/little>>  # x86 byte order

# Signed vs unsigned
data = <<255:8/unsigned, -1:8/signed>>

# Float types
measurements = <<3.14:32/float, 2.718:64/float>>

# UTF-8 strings
text = <<"Hello, 世界"/utf8>>
```

### Size Operator

```python
data = <<"Hello World">>
size = @data  # Returns 11 bytes

if @packet > 1024 {
    print("Packet too large")
}
```

### String and Array Operations

```python
# String concatenation
message = "Error: " ++ error_code ++ " at line " ++ line_number

# Array concatenation
combined = concat([1, 2], [3, 4], [5, 6])
# Result: [1, 2, 3, 4, 5, 6]
```

### Big Integer Support

```python
# Arbitrary precision arithmetic
huge = 999999999999999999999999999999999999
result = huge * 2

# Works seamlessly in bitstrings
packet = <<huge:256/big>>

# Complex expressions in pattern matching
data = <<10:8, payload/binary>>
match data {
    <<size:8, content:(size * 8)/binary>> -> {
        print("Extracted", size, "bytes")
    }
}
```

## Language Integration

FunTerm can embed Lua, Python, and JavaScript code when needed:

```python
# Use Python for complex calculations
py(hash_data) {
    import hashlib
    def hash_data(data):
        return hashlib.sha256(data).hexdigest()
}

# Use Lua for string manipulation
lua(format_hex) {
    function format_hex(value)
        return string.format("0x%08X", value)
    end
}

# Call from FunTerm
packet = <<"test">>
status = 200
hash = py.hash_data(packet.bytes)
formatted = lua.format_hex(status)
```

## Use Cases

### Educational Purposes
- Learn binary protocols and data structures
- Understand network packet structure
- Explore file formats interactively
- Practice bitwise operations and low-level programming

### Protocol Development and Prototyping
- Rapid protocol design and testing
- Debug network communications
- Parse and analyze binary protocols
- Create protocol test harnesses
- Reverse engineer unknown protocols

### Automation and Scripting
- System administration tasks with binary data
- IoT device communication scripts
- Binary file manipulation and conversion
- Network monitoring and analysis tools
- Custom data extraction pipelines

## Examples & Resources

### Run Examples

```bash
# Run script
./funterm examples/001_dns_query.su

# Interactive REPL
./funterm
```

## License

MIT
