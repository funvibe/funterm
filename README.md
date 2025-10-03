# 🚀 funterm: Multi-Language REPL & Binary Data Processor

**Seamlessly blend Python, Lua, JavaScript, and Go with advanced bitstring pattern matching, cross-language pipes, and sophisticated data processing capabilities.**

funterm integrates Python's ecosystem, Lua's speed, JavaScript's versatility, and Go's performance into a unified scripting environment with Erlang-style bitstring operations and pattern matching.

## ✨ Key Features

- 🛠️ **Multi-Language Support**: Python, Lua, JavaScript (Node.js), Go
- 🔥 **Bitstring Pattern Matching**: Erlang-style binary parsing with `<< >>` syntax
- ⚡ **Inplace Pattern Matching**: Direct variable extraction with `<<pattern>> = value`
- 🔄 **Cross-Language Pipes**: Chain functions between languages with `|`
- ⚡ **Background Execution**: Non-blocking tasks with `&` operator
- 🎯 **Advanced Pattern Matching**: Destructure data with guards and wildcards
- 🔧 **String Concatenation**: `++` operator with automatic type conversion
- 📦 **Controlled Variable Persistence**: Explicit state management with `runtime (vars) { ... }`

## 🏃‍♂️ Quick Start

### Download Pre-built Binaries
Download the latest release for your platform from [GitHub Releases](../../releases):
- **Linux**: `funterm-linux-amd64.tar.gz` or `funterm-linux-arm64.tar.gz`
- **macOS**: `funterm-darwin-amd64.tar.gz` or `funterm-darwin-arm64.tar.gz`
- **Windows**: `funterm-windows-amd64.zip` or `funterm-windows-arm64.zip`
- **FreeBSD/OpenBSD**: `funterm-freebsd-amd64.tar.gz` or `funterm-openbsd-amd64.tar.gz`

### Build from Source
```bash
# Clone and build
git clone https://github.com/funvibe/funterm.git
cd funterm
go build -o funterm main.go config.go batch.go

# Run interactive mode
./funterm

# Execute a script
./funterm -exec script.su
```

## 🎯 Real-World Examples

### 🔥 Bitstring Pattern Matching
```erlang
# Binary protocol parsing
lua.packet = <<0xDEADBEEF:32, 256:16/little, "payload"/binary, 0x1234:16>>

match lua.packet {
    <<header:32, size:16/little, data/binary, checksum:16>> -> {
        lua.print("Header:", lua.string.format("0x%08X", header))
        lua.print("Size:", size, "bytes")
        lua.print("Data:", data)
    }
}

# UTF-8 string processing
lua.utf8_data = <<"Hello 🚀"/utf8>>
match lua.utf8_data {
    <<"Hello", emoji:32/utf8>> -> lua.print("Found emoji:", emoji)
}
```

### ⚡ Inplace Pattern Matching
```erlang
# Direct pattern assignment (Erlang-style)
lua.source = "Hello"

# Extract variables directly from pattern
if (<<h/utf8, e/utf8, l1/utf8, l2/utf8, o/utf8>> = lua.source) {
    lua.print("Characters:", h, e, l1, l2, o)  # Output: 72 101 108 108 111
}

# Use qualified variables to persist results
lua.data = "Hi"
<<lua.first/utf8, lua.second/utf8>> = lua.data
lua.print("Qualified pattern:", lua.first, lua.second)  # Output: 72 105

# Binary protocol parsing
lua.packet = <<0xAA:8, 12:4, 0x55:8>>
<<header:8, id:4, footer:8>> = lua.packet
lua.print("Header:", header, "ID:", id, "Footer:", footer)
```

### 🔄 Cross-Language Pipes
```python
# Multi-language data processing pipeline
py (process_data) {
    def process_data(text):
        return text.upper().strip()
}

lua (add_prefix) {
    function add_prefix(text)
        return "PROCESSED: " .. text
    end
}

js (up) {
    function up(s) {
        return s.toUpperCase();
    }
}

# Chain operations across languages
py.greeting = "hello world"
lua.result = js.up(py.greeting)
lua.print(lua.result)  # Output: HELLO WORLD
```

### 🔧 String Concatenation
```python
# String concatenation with automatic type conversion
py.name = "Alice"
py.age = 30
py.score = 85.5

# Number + String = String
lua.message = py.age ++ " years old"
lua.print(lua.message)  # Output: "30 years old"

# String + Number = String  
lua.info = py.name ++ " scored " ++ py.score
lua.print(lua.info)  # Output: "Alice scored 85.5"

# Float formatting uses %g (42.0 becomes "42")
py.value = 42.0
lua.formatted = py.value ++ " points"
lua.print(lua.formatted)  # Output: "42 points"
```

### ⚡ Background Execution
```python
# Non-blocking tasks
py (background_task) {
    def background_task():
        import time
        time.sleep(2)
        print("Background task completed")
}

py.background_task() &  # Runs in background
py.print("This executes immediately")
```

### 📦 Controlled Variable Persistence
```python
# Variables are isolated by default
py {
    temp_var = "not saved"
    print(temp_var)
}
# py.temp_var  # Error: variable not accessible

# Explicit variable preservation
py (greeting, calculate) {
    def greeting(name):
        return f"Hello, {name}!"
    
    def calculate(x, y):
        return x * y + 10
}

# Functions persist and can be used
lua.print(py.greeting("World"))  # Output: Hello, World!
lua.print(py.calculate(5, 3))    # Output: 25
```

## 🛠️ Language Runtimes

### Python Runtime - Full Ecosystem
```python
py (fetch_data) {
    import requests, json
    
    def fetch_data(url):
        response = requests.get(url)
        return response.json()
}
```

### Lua Runtime - High Performance
```lua
lua (fast_filter) {
    function fast_filter(arr, predicate)
        local result = {}
        for i, v in ipairs(arr) do
            if predicate(v) then
                table.insert(result, v)
            end
        end
        return result
    end
}
```

### JavaScript Runtime - Node.js Ecosystem
```javascript
js (processFiles) {
    const fs = require('fs');
    const path = require('path');
    
    function processFiles(directory) {
        const files = fs.readdirSync(directory);
        return files.filter(f => path.extname(f) === '.js');
    }
}
```

### Go Runtime - System Operations
```go
# High-performance utilities (stateless)
lua.result = go.md5("hello world")      # Cryptography
lua.timestamp = go.timestamp()          # Time functions
lua.encoded = go.base64_encode("data")  # Encoding
lua.files = go.list_dir("/tmp")         # File operations
```

## 📁 Project Structure

```
funterm/
├── main.go                    # Entry point
├── config.go                  # Configuration system
├── batch.go                   # Batch execution mode
├── engine/                    # Execution engine
├── go-parser/                 # Parser with bitstring support
├── runtime/                   # Language runtimes
│   ├── python/               # Python runtime (external process)
│   ├── lua/                  # Lua runtime (embedded)
│   ├── node/                 # Node.js runtime (external)
│   └── go/                   # Go runtime (embedded)
├── funbit/                   # Bitstring processing library
├── examples/                 # Usage examples
└── tests/                    # Test scenarios (regression tests)
    ├── 016_*.su              # Bitstring tests
    ├── 012_*.su              # Pipe expression tests
    ├── 013_*.su              # Background execution tests
    └── 018_*.su              # Inplace pattern matching tests
```

## ⚙️ Configuration

### Default Behavior (No Config Required)
```bash
# Works out-of-the-box with smart defaults
./funterm
```

### Optional Configuration
```yaml
# ~/.funterm/config.yaml
engine:
  max_execution_time_seconds: 60  # Custom timeout
  verbose: true                   # Debug output

languages:
  disabled: ["go"]  # Disable specific languages
  runtimes:
    python:
      path: "/usr/local/bin/python3.11"  # Custom Python path
```

## 📚 Examples

Check out the `examples/` directory for real-world use cases:
- `001_welcome.su` - Basic multi-language usage
- `006_bitcoin.su` - Bitcoin transaction parsing
- `008_iot.su` - IoT protocol handling

The `tests/` directory contains comprehensive test scenarios that demonstrate all features.

## 🤝 Contributing

Open-source project welcoming contributions for new features, optimizations, and documentation.

**License**: MIT

---

**Ready to orchestrate multiple languages with advanced binary processing?** 🚀  
`./funterm`