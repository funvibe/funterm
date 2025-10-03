package lua

import (
	"fmt"
	"regexp"
	"strings"
	"sync"

	lua "github.com/yuin/gopher-lua"
)

// FFIEnhancer provides enhanced FFI capabilities for Lua within gopher-lua limitations
type FFIEnhancer struct {
	mu            sync.Mutex
	loadedLibs    map[string]interface{}
	functionCache map[string]*FFIFunction
	typeDefs      map[string]string
	enhancedTypes map[string]FFIType
}

// FFIFunction represents an FFI function with metadata
type FFIFunction struct {
	Name        string
	Library     string
	ReturnType  string
	ParamTypes  []string
	Description string
}

// FFIType represents an FFI type definition
type FFIType struct {
	Name        string
	Size        int
	Alignment   int
	Description string
	Fields      []FFIField
}

// FFIField represents a field in a struct type
type FFIField struct {
	Name     string
	Type     string
	Offset   int
	Size     int
	Comments string
}

// NewFFIEnhancer creates a new FFI enhancer instance
func NewFFIEnhancer() *FFIEnhancer {
	enhancer := &FFIEnhancer{
		loadedLibs:    make(map[string]interface{}),
		functionCache: make(map[string]*FFIFunction),
		typeDefs:      make(map[string]string),
		enhancedTypes: make(map[string]FFIType),
	}

	enhancer.initializeBasicTypes()
	enhancer.initializeCommonFunctions()

	return enhancer
}

// initializeBasicTypes sets up enhanced type definitions
func (fe *FFIEnhancer) initializeBasicTypes() {
	fe.enhancedTypes = map[string]FFIType{
		"int": {
			Name:        "int",
			Size:        4,
			Alignment:   4,
			Description: "Signed integer",
		},
		"uint32_t": {
			Name:        "uint32_t",
			Size:        4,
			Alignment:   4,
			Description: "Unsigned 32-bit integer",
		},
		"int32_t": {
			Name:        "int32_t",
			Size:        4,
			Alignment:   4,
			Description: "Signed 32-bit integer",
		},
		"int64_t": {
			Name:        "int64_t",
			Size:        8,
			Alignment:   8,
			Description: "Signed 64-bit integer",
		},
		"uint64_t": {
			Name:        "uint64_t",
			Size:        8,
			Alignment:   8,
			Description: "Unsigned 64-bit integer",
		},
		"float": {
			Name:        "float",
			Size:        4,
			Alignment:   4,
			Description: "Single precision floating point",
		},
		"double": {
			Name:        "double",
			Size:        8,
			Alignment:   8,
			Description: "Double precision floating point",
		},
		"void*": {
			Name:        "void*",
			Size:        8,
			Alignment:   8,
			Description: "Void pointer",
		},
		"char": {
			Name:        "char",
			Size:        1,
			Alignment:   1,
			Description: "Character",
		},
	}
}

// initializeCommonFunctions sets up common FFI functions
func (fe *FFIEnhancer) initializeCommonFunctions() {
	// Common C library functions
	fe.functionCache["strlen"] = &FFIFunction{
		Name:        "strlen",
		Library:     "libc",
		ReturnType:  "size_t",
		ParamTypes:  []string{"const char*"},
		Description: "Get string length",
	}

	fe.functionCache["strcpy"] = &FFIFunction{
		Name:        "strcpy",
		Library:     "libc",
		ReturnType:  "char*",
		ParamTypes:  []string{"char*", "const char*"},
		Description: "Copy string",
	}

	fe.functionCache["malloc"] = &FFIFunction{
		Name:        "malloc",
		Library:     "libc",
		ReturnType:  "void*",
		ParamTypes:  []string{"size_t"},
		Description: "Allocate memory",
	}

	fe.functionCache["free"] = &FFIFunction{
		Name:        "free",
		Library:     "libc",
		ReturnType:  "void",
		ParamTypes:  []string{"void*"},
		Description: "Free memory",
	}

	fe.functionCache["printf"] = &FFIFunction{
		Name:        "printf",
		Library:     "libc",
		ReturnType:  "int",
		ParamTypes:  []string{"const char*", "..."},
		Description: "Print formatted output",
	}

	// Math functions
	fe.functionCache["sin"] = &FFIFunction{
		Name:        "sin",
		Library:     "libm",
		ReturnType:  "double",
		ParamTypes:  []string{"double"},
		Description: "Sine function",
	}

	fe.functionCache["cos"] = &FFIFunction{
		Name:        "cos",
		Library:     "libm",
		ReturnType:  "double",
		ParamTypes:  []string{"double"},
		Description: "Cosine function",
	}

	fe.functionCache["sqrt"] = &FFIFunction{
		Name:        "sqrt",
		Library:     "libm",
		ReturnType:  "double",
		ParamTypes:  []string{"double"},
		Description: "Square root function",
	}
}

// EnhanceRuntime enhances Lua runtime with FFI capabilities
func (fe *FFIEnhancer) EnhanceRuntime(L *lua.LState) error {
	fe.mu.Lock()
	defer fe.mu.Unlock()

	// Create enhanced FFI object
	ffiTable := L.NewTable()

	// Add basic types
	typesTable := L.NewTable()
	for typeName, typeInfo := range fe.enhancedTypes {
		typeTable := L.NewTable()
		L.SetField(typeTable, "name", lua.LString(typeInfo.Name))
		L.SetField(typeTable, "size", lua.LNumber(typeInfo.Size))
		L.SetField(typeTable, "alignment", lua.LNumber(typeInfo.Alignment))
		L.SetField(typeTable, "description", lua.LString(typeInfo.Description))
		L.SetField(typesTable, typeName, typeTable)
	}
	L.SetField(ffiTable, "types", typesTable)

	// Add common functions
	funcsTable := L.NewTable()
	for funcName, funcInfo := range fe.functionCache {
		funcTable := L.NewTable()
		L.SetField(funcTable, "name", lua.LString(funcInfo.Name))
		L.SetField(funcTable, "library", lua.LString(funcInfo.Library))
		L.SetField(funcTable, "return_type", lua.LString(funcInfo.ReturnType))
		L.SetField(funcTable, "description", lua.LString(funcInfo.Description))

		paramsTable := L.NewTable()
		for i, paramType := range funcInfo.ParamTypes {
			L.SetField(paramsTable, fmt.Sprintf("%d", i+1), lua.LString(paramType))
		}
		L.SetField(funcTable, "parameters", paramsTable)

		L.SetField(funcsTable, funcName, funcTable)
	}
	L.SetField(ffiTable, "functions", funcsTable)

	// Add enhanced cdef function
	L.SetField(ffiTable, "cdef", L.NewFunction(fe.enhancedCDef))
	L.SetField(ffiTable, "typeof", L.NewFunction(fe.enhancedTypeOf))
	L.SetField(ffiTable, "cast", L.NewFunction(fe.enhancedCast))
	L.SetField(ffiTable, "new", L.NewFunction(fe.enhancedNew))
	L.SetField(ffiTable, "load_library", L.NewFunction(fe.enhancedLoadLibrary))

	// Replace existing ffi global with enhanced version
	L.SetGlobal("ffi", ffiTable)

	return nil
}

// enhancedCDef provides enhanced cdef functionality
func (fe *FFIEnhancer) enhancedCDef(L *lua.LState) int {
	definition := L.CheckString(1)

	// Parse and store the definition
	fe.parseAndStoreDefinition(definition)

	// In real implementation, this would actually define the C types
	// For now, we'll just acknowledge it
	L.Push(lua.LString("FFI definition processed: " + definition))
	return 1
}

// enhancedTypeOf provides enhanced typeof functionality
func (fe *FFIEnhancer) enhancedTypeOf(L *lua.LState) int {
	typeName := L.CheckString(1)

	if typeInfo, exists := fe.enhancedTypes[typeName]; exists {
		result := L.NewTable()
		L.SetField(result, "name", lua.LString(typeInfo.Name))
		L.SetField(result, "size", lua.LNumber(typeInfo.Size))
		L.SetField(result, "alignment", lua.LNumber(typeInfo.Alignment))
		L.SetField(result, "description", lua.LString(typeInfo.Description))
		L.Push(result)
	} else {
		L.Push(lua.LNil)
	}
	return 1
}

// enhancedCast provides enhanced cast functionality
func (fe *FFIEnhancer) enhancedCast(L *lua.LState) int {
	targetType := L.CheckString(1)
	value := L.CheckAny(2)

	// In real implementation, this would perform actual type casting
	// For now, return the value with type information
	result := L.NewTable()
	L.SetField(result, "type", lua.LString(targetType))
	L.SetField(result, "value", value)
	L.SetField(result, "note", lua.LString("Cast operation simulated"))
	L.Push(result)
	return 1
}

// enhancedNew provides enhanced new functionality
func (fe *FFIEnhancer) enhancedNew(L *lua.LState) int {
	typeName := L.CheckString(1)

	// In real implementation, this would create a new C object
	// For now, return a simulated object
	result := L.NewTable()
	L.SetField(result, "type", lua.LString(typeName))
	L.SetField(result, "address", lua.LString("0x0")) // Simulated address
	L.SetField(result, "note", lua.LString("New operation simulated"))
	L.Push(result)
	return 1
}

// enhancedLoadLibrary provides enhanced library loading
func (fe *FFIEnhancer) enhancedLoadLibrary(L *lua.LState) int {
	libName := L.CheckString(1)

	// In real implementation, this would load the actual library
	// For now, return a simulated library object
	result := L.NewTable()
	L.SetField(result, "name", lua.LString(libName))
	L.SetField(result, "loaded", lua.LBool(true))
	L.SetField(result, "note", lua.LString("Library loading simulated"))
	L.Push(result)
	return 1
}

// parseAndStoreDefinition parses FFI definitions and stores them
func (fe *FFIEnhancer) parseAndStoreDefinition(definition string) {
	// Simple parsing for common patterns
	if strings.Contains(definition, "typedef") {
		fe.parseTypeDef(definition)
	} else if strings.Contains(definition, "(") && strings.Contains(definition, ")") {
		fe.parseFunctionDef(definition)
	}
}

// parseTypeDef parses type definitions
func (fe *FFIEnhancer) parseTypeDef(definition string) {
	// Simple regex-based parsing for type definitions
	typedefPattern := `typedef\s+(struct\s+\w+\s*\{[^}]*\}\s*|\w+)\s+(\w+)\s*;`
	re := regexp.MustCompile(typedefPattern)
	matches := re.FindStringSubmatch(definition)

	if len(matches) == 3 {
		typeName := matches[2]
		fe.typeDefs[typeName] = definition
		return
	}

	// Fallback for simple typedefs like "typedef int my_int;"
	simplePattern := `typedef\s+(\w+)\s+(\w+)\s*;`
	simpleRe := regexp.MustCompile(simplePattern)
	simpleMatches := simpleRe.FindStringSubmatch(definition)

	if len(simpleMatches) == 3 {
		typeName := simpleMatches[2]
		fe.typeDefs[typeName] = definition
	}
}

// parseFunctionDef parses function definitions
func (fe *FFIEnhancer) parseFunctionDef(definition string) {
	// Simple regex-based parsing for function definitions
	funcPattern := `(\w+)\s+(\w+)\s*\(([^)]*)\);`
	re := regexp.MustCompile(funcPattern)
	matches := re.FindStringSubmatch(definition)

	if len(matches) == 4 {
		returnType := matches[1]
		funcName := matches[2]
		params := strings.Split(matches[3], ",")

		paramTypes := make([]string, 0)
		for _, param := range params {
			param = strings.TrimSpace(param)
			if param != "" {
				// Extract type (ignore parameter name)
				parts := strings.Fields(param)
				if len(parts) > 0 {
					paramTypes = append(paramTypes, parts[0])
				}
			}
		}

		fe.functionCache[funcName] = &FFIFunction{
			Name:       funcName,
			Library:    "user_defined",
			ReturnType: returnType,
			ParamTypes: paramTypes,
		}
	}
}

// GetFunctionInfo returns information about a function
func (fe *FFIEnhancer) GetFunctionInfo(funcName string) (*FFIFunction, bool) {
	fe.mu.Lock()
	defer fe.mu.Unlock()

	funcInfo, exists := fe.functionCache[funcName]
	return funcInfo, exists
}

// GetTypeInfo returns information about a type
func (fe *FFIEnhancer) GetTypeInfo(typeName string) (*FFIType, bool) {
	fe.mu.Lock()
	defer fe.mu.Unlock()

	typeInfo, exists := fe.enhancedTypes[typeName]
	if !exists {
		return nil, false
	}
	// Return a copy to avoid race conditions
	result := typeInfo
	return &result, exists
}

// ListFunctions returns all available functions
func (fe *FFIEnhancer) ListFunctions() map[string]*FFIFunction {
	fe.mu.Lock()
	defer fe.mu.Unlock()

	result := make(map[string]*FFIFunction)
	for name, funcInfo := range fe.functionCache {
		result[name] = funcInfo
	}
	return result
}

// ListTypes returns all available types
func (fe *FFIEnhancer) ListTypes() map[string]FFIType {
	fe.mu.Lock()
	defer fe.mu.Unlock()

	result := make(map[string]FFIType)
	for name, typeInfo := range fe.enhancedTypes {
		result[name] = typeInfo
	}
	return result
}

// AddCustomFunction adds a custom function definition
func (fe *FFIEnhancer) AddCustomFunction(funcName, library, returnType string, paramTypes []string, description string) {
	fe.mu.Lock()
	defer fe.mu.Unlock()

	fe.functionCache[funcName] = &FFIFunction{
		Name:        funcName,
		Library:     library,
		ReturnType:  returnType,
		ParamTypes:  paramTypes,
		Description: description,
	}
}

// AddCustomType adds a custom type definition
func (fe *FFIEnhancer) AddCustomType(name string, size, alignment int, description string) {
	fe.mu.Lock()
	defer fe.mu.Unlock()

	fe.enhancedTypes[name] = FFIType{
		Name:        name,
		Size:        size,
		Alignment:   alignment,
		Description: description,
	}
}

// GetCapabilities returns the capabilities of the FFI enhancer
func (fe *FFIEnhancer) GetCapabilities() map[string]interface{} {
	return map[string]interface{}{
		"enhanced_types":   len(fe.enhancedTypes),
		"function_cache":   len(fe.functionCache),
		"type_definitions": len(fe.typeDefs),
		"loaded_libraries": len(fe.loadedLibs),
		"simulated_ffi":    true, // Using simulated FFI due to gopher-lua limitations
	}
}

// ValidateDefinition validates an FFI definition
func (fe *FFIEnhancer) ValidateDefinition(definition string) error {
	// Basic validation
	if strings.TrimSpace(definition) == "" {
		return fmt.Errorf("empty definition")
	}

	// Check for basic syntax
	if strings.Contains(definition, "typedef") {
		if !strings.Contains(definition, ";") {
			return fmt.Errorf("typedef missing semicolon")
		}
	} else if strings.Contains(definition, "(") {
		if !strings.Contains(definition, ")") || !strings.Contains(definition, ";") {
			return fmt.Errorf("function definition missing closing parenthesis or semicolon")
		}
	}

	return nil
}

// GenerateBindings generates Lua bindings for C functions
func (fe *FFIEnhancer) GenerateBindings() string {
	var bindings strings.Builder

	bindings.WriteString("-- Enhanced FFI Bindings Generated Automatically\n")
	bindings.WriteString("-- This provides simulated FFI functionality within gopher-lua limitations\n\n")

	// Generate type definitions
	bindings.WriteString("-- Type Definitions\n")
	for typeName, typeInfo := range fe.enhancedTypes {
		bindings.WriteString(fmt.Sprintf("-- %s: %s (%d bytes, alignment %d)\n",
			typeName, typeInfo.Description, typeInfo.Size, typeInfo.Alignment))
	}
	bindings.WriteString("\n")

	// Generate function wrappers
	bindings.WriteString("-- Function Wrappers\n")
	for funcName, funcInfo := range fe.functionCache {
		bindings.WriteString(fmt.Sprintf("-- %s from %s: %s\n", funcName, funcInfo.Library, funcInfo.Description))
		bindings.WriteString(fmt.Sprintf("-- Returns: %s\n", funcInfo.ReturnType))
		if len(funcInfo.ParamTypes) > 0 {
			bindings.WriteString("-- Parameters: " + strings.Join(funcInfo.ParamTypes, ", ") + "\n")
		}
		bindings.WriteString("\n")
	}

	return bindings.String()
}

// GetDocumentation returns comprehensive FFI documentation
func (fe *FFIEnhancer) GetDocumentation() string {
	var doc strings.Builder

	doc.WriteString("# Enhanced FFI Documentation\n\n")
	doc.WriteString("This FFI enhancer provides extended Foreign Function Interface capabilities\n")
	doc.WriteString("within the limitations of the gopher-lua implementation.\n\n")

	doc.WriteString("## Available Types\n\n")
	for typeName, typeInfo := range fe.enhancedTypes {
		doc.WriteString(fmt.Sprintf("%s:\n", typeName))
		doc.WriteString(fmt.Sprintf("- **Size**: %d bytes\n", typeInfo.Size))
		doc.WriteString(fmt.Sprintf("- **Alignment**: %d bytes\n", typeInfo.Alignment))
		doc.WriteString(fmt.Sprintf("- **Description**: %s\n\n", typeInfo.Description))
	}

	doc.WriteString("## Available Functions\n\n")
	for funcName, funcInfo := range fe.functionCache {
		doc.WriteString(fmt.Sprintf("### %s\n", funcName))
		doc.WriteString(fmt.Sprintf("- **Library**: %s\n", funcInfo.Library))
		doc.WriteString(fmt.Sprintf("- **Return Type**: %s\n", funcInfo.ReturnType))
		doc.WriteString(fmt.Sprintf("- **Parameters**: %s\n", strings.Join(funcInfo.ParamTypes, ", ")))
		doc.WriteString(fmt.Sprintf("- **Description**: %s\n\n", funcInfo.Description))
	}

	doc.WriteString("## Usage Examples\n\n")
	doc.WriteString("```lua\n")
	doc.WriteString("-- Define a custom type\n")
	doc.WriteString("ffi.cdef[[\n")
	doc.WriteString("typedef struct {\n")
	doc.WriteString("    int x;\n")
	doc.WriteString("    int y;\n")
	doc.WriteString("} Point;\n")
	doc.WriteString("]]\n\n")
	doc.WriteString("-- Get type information\n")
	doc.WriteString("point_type = ffi.typeof(\"Point\")\n")
	doc.WriteString("print(point_type.name, point_type.size)\n\n")
	doc.WriteString("-- Create a new instance (simulated)\n")
	doc.WriteString("point = ffi.new(\"Point\")\n")
	doc.WriteString("```")

	return doc.String()
}
