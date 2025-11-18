package lua

import (
	"fmt"
	"reflect"
	"strings"
	"sync"

	"funterm/shared"
	"go-parser/pkg/ast"

	"github.com/funvibe/funbit/pkg/funbit"
	lua "github.com/yuin/gopher-lua"
)

// LuaFunctionWrapper wraps a Lua function to preserve it when converting to Go
type LuaFunctionWrapper struct {
	Function lua.LValue
}

// LuaRuntime implements the LanguageRuntime interface for Lua
type LuaRuntime struct {
	state *lua.LState
	ready bool
	// stateManager removed for simplified beta
	userDefinedFunctions []string
	importedModules      []string
	executionHistory     []string
	runtimeObjects       map[string]interface{}
	mu                   sync.Mutex       // Для потокобезопасности
	outputCapture        *strings.Builder // Для перехвата вывода
	ffiEnhancer          *FFIEnhancer     // Enhanced FFI support
	moduleManager        *ModuleManager   // Built-in modules manager
	verbose              bool             // Флаг для вывода отладочной информации
}

// NewLuaRuntime creates a new Lua runtime instance
func NewLuaRuntime() *LuaRuntime {
	return &LuaRuntime{
		state:                nil,
		ready:                false,
		userDefinedFunctions: make([]string, 0),
		importedModules:      make([]string, 0),
		executionHistory:     make([]string, 0),
		runtimeObjects:       make(map[string]interface{}),
		outputCapture:        nil,
		ffiEnhancer:          NewFFIEnhancer(),
		moduleManager:        NewModuleManager(),
		verbose:              false,
	}
}

// Initialize sets up the Lua runtime
func (lr *LuaRuntime) Initialize() error {
	// Create new Lua state
	lr.state = lua.NewState()

	// Open standard libraries
	lr.state.OpenLibs()

	// Initialize FFI support
	if err := lr.initializeFFI(); err != nil {
		lr.state.Close()
		return fmt.Errorf("failed to initialize FFI: %w", err)
	}

	// Register custom functions for Go-Lua integration
	lr.registerGoFunctions()

	// Register bitstring utility functions
	lr.registerBitstringFunctions()

	// Create a fresh module manager to avoid registration conflicts
	lr.moduleManager = NewModuleManager()

	// Register built-in modules
	if err := lr.registerBuiltinModules(); err != nil {
		lr.state.Close()
		return fmt.Errorf("failed to register built-in modules: %w", err)
	}

	lr.ready = true
	return nil
}

// registerBitstringFunctions registers utility functions for working with bitstrings
func (lr *LuaRuntime) registerBitstringFunctions() {
	// Register len function for bitstrings and byte arrays (lua.len(bitstring))
	lr.state.SetGlobal("len", lr.state.NewFunction(func(L *lua.LState) int {
		if L.GetTop() < 1 {
			L.RaiseError("len() requires one argument")
			return 0
		}

		arg := L.Get(1)

		// Handle regular Lua strings first
		if str, ok := arg.(lua.LString); ok {
			L.Push(lua.LNumber(len(string(str))))
			return 1
		}

		if userData, ok := arg.(*lua.LUserData); ok {
			if bs, ok := userData.Value.(*shared.BitstringObject); ok {
				// For bitstrings, return byte length (bits / 8)
				L.Push(lua.LNumber(bs.Len() / 8))
				return 1
			}
			// Handle []uint8 (byte arrays from rest patterns)
			if bytes, ok := userData.Value.([]uint8); ok {
				L.Push(lua.LNumber(len(bytes)))
				return 1
			}
			// Handle []byte (byte arrays)
			if bytes, ok := userData.Value.([]byte); ok {
				L.Push(lua.LNumber(len(bytes)))
				return 1
			}
		}

		// Fallback to standard Lua behavior
		L.Push(lua.LNumber(0))
		return 1
	}))

	// Register bit_size function for bitstrings (lua.bit_size(bitstring))
	lr.state.SetGlobal("bit_size", lr.state.NewFunction(func(L *lua.LState) int {
		if L.GetTop() < 1 {
			L.RaiseError("bit_size() requires one argument")
			return 0
		}

		arg := L.Get(1)
		if userData, ok := arg.(*lua.LUserData); ok {
			if bs, ok := userData.Value.(*shared.BitstringObject); ok {
				// Return bit length
				L.Push(lua.LNumber(bs.Len()))
				return 1
			}
			// Handle *funbit.BitString directly (from rest patterns)
			if bs, ok := userData.Value.(*funbit.BitString); ok {
				L.Push(lua.LNumber(bs.Length()))
				return 1
			}
		}

		// For non-bitstring values, return 0
		L.Push(lua.LNumber(0))
		return 1
	}))
}

// initializeFFI sets up enhanced FFI support
func (lr *LuaRuntime) initializeFFI() error {
	// Try to load FFI library - it may not be available in standard gopher-lua
	if err := lr.state.DoString(`
		local ffi_loaded, ffi = pcall(require, "ffi")
		if ffi_loaded then
			-- Make FFI available globally
			_G.ffi = ffi
			
			-- Initialize basic C types
			ffi.cdef[[
				typedef int int32_t;
				typedef unsigned int uint32_t;
				typedef long long int64_t;
				typedef unsigned long long uint64_t;
				typedef float float32_t;
				typedef double float64_t;
				typedef void* void_ptr;
			]]
		else
			-- FFI not available, create a placeholder that indicates FFI is not supported
			_G.ffi = {
				cdef = function()
					error("FFI not available in this Lua implementation")
				end,
				C = {},
				new = function()
					error("FFI not available in this Lua implementation")
				end,
				cast = function()
					error("FFI not available in this Lua implementation")
				end,
				typeof = function()
					error("FFI not available in this Lua implementation")
				end,
			}
		end
	`); err != nil {
		return fmt.Errorf("failed to initialize FFI: %w", err)
	}

	// Apply enhanced FFI capabilities
	if err := lr.ffiEnhancer.EnhanceRuntime(lr.state); err != nil {
		return fmt.Errorf("failed to enhance FFI: %w", err)
	}

	return nil
}

// parseLuaTableToBitstringExpression parses a Lua table into a BitstringExpression AST
func parseLuaTableToBitstringExpression(table *lua.LTable) *ast.BitstringExpression {
	expr := &ast.BitstringExpression{
		Segments: make([]ast.BitstringSegment, 0),
	}

	segmentsTable := table.RawGetString("segments")
	if segmentsTable.Type() != lua.LTTable {
		return expr
	}

	segmentsLTable := segmentsTable.(*lua.LTable)
	segmentsLTable.ForEach(func(key, value lua.LValue) {
		if segmentTable, ok := value.(*lua.LTable); ok {
			segment := ast.BitstringSegment{}

			// Parse value
			if value := segmentTable.RawGetString("value"); value.Type() == lua.LTString {
				segment.Value = &ast.StringLiteral{Value: value.String()}
			} else if value.Type() == lua.LTNil {
				// Variable, leave as nil
			}

			// Parse size
			if size := segmentTable.RawGetString("size"); size.Type() == lua.LTNumber {
				sizeValue := float64(size.(lua.LNumber))
				segment.Size = &ast.NumberLiteral{
					FloatValue: sizeValue,
					IsInt:      false,
					Pos:        ast.Position{},
				}
			}

			// Parse specifiers
			if specifiers := segmentTable.RawGetString("specifiers"); specifiers.Type() == lua.LTTable {
				specsTable := specifiers.(*lua.LTable)
				segment.Specifiers = make([]string, 0)
				specsTable.ForEach(func(_, spec lua.LValue) {
					if spec.Type() == lua.LTString {
						segment.Specifiers = append(segment.Specifiers, spec.String())
					}
				})
			}

			expr.Segments = append(expr.Segments, segment)
		}
	})

	return expr
}

// Execute executes Lua code and returns captured output
func (lr *LuaRuntime) Execute(code string) (string, error) {
	// Note: engine should be set in registry before calling this
	result, err := lr.Eval(code)
	if err != nil {
		return "", err
	}
	if str, ok := result.(string); ok {
		return str, nil
	}
	return "", nil
}

// GetState returns the Lua state
func (lr *LuaRuntime) GetState() *lua.LState {
	return lr.state
}

// registerGoFunctions registers Go functions that can be called from Lua
func (lr *LuaRuntime) registerGoFunctions() {
	// Register custom print function that captures output
	lr.state.SetGlobal("print", lr.state.NewFunction(func(L *lua.LState) int {
		// Get all arguments
		args := make([]interface{}, L.GetTop())
		for i := 1; i <= L.GetTop(); i++ {
			args[i-1] = lr.luaToGo(L.Get(i))
		}

		// Convert arguments to strings using centralized formatter
		// Skip nil values to prevent outputting "<nil>"
		var strArgs []string
		for _, arg := range args {
			if arg != nil {
				strArgs = append(strArgs, shared.FormatValueForDisplay(arg))
			}
		}

		// Only output if we have non-nil arguments
		if len(strArgs) == 0 {
			return 0 // No return values, no output
		}

		// Join with spaces and print to output capture
		output := strings.Join(strArgs, " ")

		// Debug output
		if lr.verbose {
			fmt.Printf("DEBUG: Lua print called with output: '%s', outputCapture: %v\n", output, lr.outputCapture != nil)
		}

		// Capture output if outputCapture is available
		if lr.outputCapture != nil {
			lr.outputCapture.WriteString(output + "\n")
			if lr.verbose {
				fmt.Printf("DEBUG: Captured output, current buffer: '%s'\n", lr.outputCapture.String())
			}
		}

		// Don't print to console here - let the engine handle output display
		// fmt.Println(output)

		return 0 // No return values
	}))

	// Register matchesBitstringPattern function for pattern matching
	lr.state.SetGlobal("matchesBitstringPattern", lr.state.NewFunction(func(L *lua.LState) int {
		// Get data and pattern arguments
		if L.GetTop() < 2 {
			L.Push(lua.LNil)
			L.RaiseError("matchesBitstringPattern() requires data and pattern arguments")
			return 0
		}

		// Get engine from global
		engineValue := L.GetGlobal("execution_engine")
		if engineValue.Type() != lua.LTUserData {
			L.Push(lua.LBool(false))
			return 1
		}
		engineUd := engineValue.(*lua.LUserData)
		// Assume it's ExecutionEngine and has MatchesBitstringPattern method
		engine := engineUd.Value

		// Convert arguments from Lua to Go
		dataArg := L.Get(1)
		patternArg := L.Get(2)

		// Convert data
		var bitstringData *shared.BitstringObject
		if ud, ok := dataArg.(*lua.LUserData); ok {
			if bs, ok := ud.Value.(*shared.BitstringObject); ok {
				bitstringData = bs
			} else {
				L.Push(lua.LBool(false))
				return 1
			}
		} else {
			L.Push(lua.LBool(false))
			return 1
		}

		// Convert pattern
		var patternExpr *ast.BitstringExpression
		if table, ok := patternArg.(*lua.LTable); ok {
			patternExpr = parseLuaTableToBitstringExpression(table)
		} else if ud, ok := patternArg.(*lua.LUserData); ok {
			if pe, ok := ud.Value.(*ast.BitstringExpression); ok {
				patternExpr = pe
			} else {
				L.Push(lua.LBool(false))
				return 1
			}
		} else {
			L.Push(lua.LBool(false))
			return 1
		}

		// Call the engine's MatchesBitstringPattern method using reflection
		engineReflectValue := reflect.ValueOf(engine)
		method := engineReflectValue.MethodByName("MatchesBitstringPattern")
		if !method.IsValid() {
			L.Push(lua.LBool(false))
			return 1
		}
		results := method.Call([]reflect.Value{
			reflect.ValueOf(patternExpr),
			reflect.ValueOf(bitstringData),
		})
		matches := results[0].Bool()

		// Push the result
		L.Push(lua.LBool(matches))
		return 1
	}))

	// Register eval function for funterm compatibility
	lr.state.SetGlobal("eval", lr.state.NewFunction(func(L *lua.LState) int {
		// Get the code argument
		if L.GetTop() < 1 {
			L.Push(lua.LNil)
			L.RaiseError("eval() requires at least one argument")
			return 0
		}

		codeArg := L.Get(1)
		if codeArg.Type() != lua.LTString {
			L.Push(lua.LNil)
			L.RaiseError("eval() requires a string argument")
			return 0
		}

		code := codeArg.String()

		// Use the existing Eval method to execute the code
		result, err := lr.Eval(code)
		if err != nil {
			L.Push(lua.LNil)
			L.RaiseError("eval error: %s", err.Error())
			return 0
		}

		// Convert the result to Lua and push it
		luaResult, err := lr.GoToLua(result)
		if err != nil {
			L.Push(lua.LNil)
			L.RaiseError("result conversion error")
			return 0
		}

		L.Push(luaResult)
		return 1
	}))
}

// registerBuiltinModules registers the built-in Lua modules
func (lr *LuaRuntime) registerBuiltinModules() error {
	// Register JSON module
	jsonModule := &JSONModule{}
	if err := lr.moduleManager.RegisterModule(jsonModule); err != nil {
		return fmt.Errorf("failed to register JSON module: %w", err)
	}

	// Register HTTP module
	httpModule := &HTTPModule{}
	if err := lr.moduleManager.RegisterModule(httpModule); err != nil {
		return fmt.Errorf("failed to register HTTP module: %w", err)
	}

	// Register filesystem module (restricted to current directory)
	fsModule := NewFSModule(".") // Current directory for security
	if err := lr.moduleManager.RegisterModule(fsModule); err != nil {
		return fmt.Errorf("failed to register filesystem module: %w", err)
	}

	// Register all modules in package.preload for require() support
	if err := lr.moduleManager.RegisterAllModules(lr.state); err != nil {
		return fmt.Errorf("failed to register modules in package.preload: %w", err)
	}

	return nil
}

// GetSupportedTypes returns the types supported by this runtime
func (lr *LuaRuntime) GetSupportedTypes() []string {
	return []string{
		"string", "number", "boolean", "nil", "table", "function",
		"ffi.cdef", "ffi.C", "ffi.typeof", "ffi.cast", "ffi.new",
	}
}

// GetName returns the name of the language runtime
func (lr *LuaRuntime) GetName() string {
	return "lua"
}

// IsReady checks if the runtime is ready for execution
func (lr *LuaRuntime) IsReady() bool {
	return lr.ready && lr.state != nil
}

// GetVariableFromRuntimeObject получает переменную из runtimeObjects
func (lr *LuaRuntime) GetVariableFromRuntimeObject(name string) (interface{}, error) {
	lr.mu.Lock()
	defer lr.mu.Unlock()

	if value, exists := lr.runtimeObjects[name]; exists {
		return value, nil
	}
	return nil, fmt.Errorf("variable '%s' not found in runtime objects", name)
}

// GoToLua converts a Go value to a Lua value
func (lr *LuaRuntime) GoToLua(value interface{}) (lua.LValue, error) {
	switch v := value.(type) {
	case string:
		return lua.LString(v), nil
	case int:
		return lua.LNumber(v), nil
	case int8:
		return lua.LNumber(v), nil
	case int16:
		return lua.LNumber(v), nil
	case int32:
		return lua.LNumber(v), nil
	case int64:
		return lua.LNumber(v), nil
	case uint:
		return lua.LNumber(v), nil
	case uint8:
		return lua.LNumber(v), nil
	case uint16:
		return lua.LNumber(v), nil
	case uint32:
		return lua.LNumber(v), nil
	case uint64:
		return lua.LNumber(v), nil
	case float32, float64:
		return lua.LNumber(v.(float64)), nil
	case bool:
		return lua.LBool(v), nil
	case nil:
		return lua.LNil, nil
	case []interface{}:
		table := lr.state.NewTable()
		for i, item := range v {
			luaItem, err := lr.GoToLua(item)
			if err != nil {
				return nil, err
			}
			table.RawSetInt(i+1, luaItem) // Lua arrays are 1-based
		}
		return table, nil
	case map[string]interface{}:
		table := lr.state.NewTable()
		for key, item := range v {
			luaItem, err := lr.GoToLua(item)
			if err != nil {
				return nil, err
			}
			table.RawSetString(key, luaItem)
		}
		return table, nil
	case *ast.BitstringExpression:
		// Create a Lua userdata object for AST
		userData := lr.state.NewUserData()
		userData.Value = value
		return userData, nil
	case *shared.BitstringObject:
		// Create a Lua userdata object for bitstring
		userData := lr.state.NewUserData()
		userData.Value = v
		// Add metatable for __tostring to return ToFunbitFormat and bytes method
		metaTable := lr.state.NewTable()
		metaTable.RawSetString("__tostring", lr.state.NewFunction(func(L *lua.LState) int {
			userData := L.CheckUserData(1)
			if bs, ok := userData.Value.(*shared.BitstringObject); ok {
				L.Push(lua.LString(funbit.ToFunbitFormat(bs.BitString)))
				return 1
			}
			L.Push(lua.LString("<<>>"))
			return 1
		}))
		// Add bytes method to get byte representation
		metaTable.RawSetString("bytes", lr.state.NewFunction(func(L *lua.LState) int {
			userData := L.CheckUserData(1)
			if bs, ok := userData.Value.(*shared.BitstringObject); ok {
				bytes := bs.BitString.ToBytes()
				// Convert []byte to Lua string (which can be used as bytes)
				L.Push(lua.LString(string(bytes)))
				return 1
			}
			L.Push(lua.LNil)
			return 1
		}))
		userData.Metatable = metaTable
		return userData, nil
	case *funbit.BitString:
		// Create a Lua userdata object for funbit.BitString (from rest patterns)
		userData := lr.state.NewUserData()
		userData.Value = v
		// Add metatable for __tostring to return ToFunbitFormat and bytes method
		metaTable := lr.state.NewTable()
		metaTable.RawSetString("__tostring", lr.state.NewFunction(func(L *lua.LState) int {
			userData := L.CheckUserData(1)
			if bs, ok := userData.Value.(*funbit.BitString); ok {
				L.Push(lua.LString(funbit.ToFunbitFormat(bs)))
				return 1
			}
			L.Push(lua.LString("<<>>"))
			return 1
		}))
		// Add bytes method to get byte representation
		metaTable.RawSetString("bytes", lr.state.NewFunction(func(L *lua.LState) int {
			userData := L.CheckUserData(1)
			if bs, ok := userData.Value.(*funbit.BitString); ok {
				bytes := bs.ToBytes()
				// Convert []byte to Lua string (which can be used as bytes)
				L.Push(lua.LString(string(bytes)))
				return 1
			}
			L.Push(lua.LNil)
			return 1
		}))
		userData.Metatable = metaTable
		return userData, nil
	case []byte:
		// Convert []byte to a Lua table of numbers
		table := lr.state.NewTable()
		for i, b := range v {
			table.RawSetInt(i+1, lua.LNumber(b)) // Lua arrays are 1-based
		}
		return table, nil
	case shared.BitstringByte:
		// Convert BitstringByte to special userdata that print can recognize
		userData := lr.state.NewUserData()
		userData.Value = v
		// Set metatable to make it behave like a number but be recognizable as BitstringByte
		metaTable := lr.state.NewTable()
		metaTable.RawSetString("__tostring", lr.state.NewFunction(func(L *lua.LState) int {
			// When converted to string, show as ASCII if printable
			if v.Value >= 32 && v.Value <= 126 {
				L.Push(lua.LString(string(rune(v.Value))))
			} else {
				L.Push(lua.LString(fmt.Sprintf("%d", v.Value)))
			}
			return 1
		}))
		userData.Metatable = metaTable
		return userData, nil
	case *LuaFunctionWrapper:
		// Return the original Lua function
		return v.Function, nil
	default:
		return nil, fmt.Errorf("unsupported Go type: %T", value)
	}
}

// luaToGo converts a Lua value to a Go value
func (lr *LuaRuntime) luaToGo(value lua.LValue) interface{} {
	return lr.luaToGoWithVisited(value, make(map[uintptr]bool))
}

// luaToGoWithVisited converts a Lua value to a Go value with cycle detection
func (lr *LuaRuntime) luaToGoWithVisited(value lua.LValue, visited map[uintptr]bool) interface{} {
	switch value.Type() {
	case lua.LTString:
		return value.String()
	case lua.LTNumber:
		num := float64(value.(lua.LNumber))
		// Check if it's a whole number within int64 range
		if num == float64(int64(num)) && num >= -9223372036854775808 && num <= 9223372036854775807 {
			return int64(num)
		}
		return num
	case lua.LTBool:
		return bool(value.(lua.LBool))
	case lua.LTNil:
		return nil
	case lua.LTUserData:
		userData := value.(*lua.LUserData)
		if bs, ok := userData.Value.(*shared.BitstringObject); ok {
			return bs
		}
		// Check if it's a BitstringByte
		if bb, ok := userData.Value.(shared.BitstringByte); ok {
			return bb
		}
		// Return the userdata value as is for other types
		return userData.Value
	case lua.LTTable:
		// Check for circular references
		ptr := uintptr(0)
		if value != nil {
			ptr = reflect.ValueOf(value).Pointer()
		}

		if ptr != 0 && visited[ptr] {
			return "<circular_reference>"
		}

		// Mark this table as visited
		if ptr != 0 {
			visited[ptr] = true
		}

		// Try to determine if it's an array or map
		table := value.(*lua.LTable)

		// Check if it's an array-like table
		isArray := true
		maxIndex := 0
		table.ForEach(func(key, val lua.LValue) {
			if key.Type() == lua.LTNumber {
				index := int(key.(lua.LNumber))
				if index > maxIndex {
					maxIndex = index
				}
			} else {
				isArray = false
			}
		})

		var result interface{}
		if isArray && maxIndex > 0 {
			// Convert to array
			result = make([]interface{}, maxIndex)
			for i := 1; i <= maxIndex; i++ {
				val := table.RawGetInt(i)
				result.([]interface{})[i-1] = lr.luaToGoWithVisited(val, visited)
			}
		} else {
			// Convert to map
			result = make(map[string]interface{})
			table.ForEach(func(key, val lua.LValue) {
				var keyStr string
				switch key.Type() {
				case lua.LTString:
					keyStr = key.String()
				case lua.LTNumber:
					num := float64(key.(lua.LNumber))
					if num == float64(int64(num)) && num >= -9223372036854775808 && num <= 9223372036854775807 {
						keyStr = fmt.Sprintf("%.0f", num)
					} else {
						keyStr = fmt.Sprintf("%v", num)
					}
				default:
					keyStr = fmt.Sprintf("%v", key)
				}
				result.(map[string]interface{})[keyStr] = lr.luaToGoWithVisited(val, visited)
			})
		}

		// Unmark this table
		if ptr != 0 {
			delete(visited, ptr)
		}

		return result
	case lua.LTFunction:
		// For functions, we need to preserve the actual Lua function object
		// Store it as a special wrapper that can be converted back
		return &LuaFunctionWrapper{Function: value}
	default:
		return fmt.Sprintf("<%s: %v>", value.Type().String(), value)
	}
}

// convertLuaValueToGo converts a Lua value to a Go value (helper method for external use)
func (lr *LuaRuntime) convertLuaValueToGo(value lua.LValue) interface{} {
	return lr.luaToGo(value)
}

// convertGoValueToLua converts a Go value to a Lua value (helper method for external use)
func (lr *LuaRuntime) convertGoValueToLua(value interface{}) (lua.LValue, error) {
	return lr.GoToLua(value)
}

// Cleanup releases resources used by the runtime
func (lr *LuaRuntime) Cleanup() error {
	lr.mu.Lock()
	defer lr.mu.Unlock()

	if lr.state != nil {
		lr.state.Close()
		lr.state = nil
	}
	lr.ready = false
	return nil
}

// Isolate creates an isolated state for the runtime
// In Lua, isolation preserves the current state (unlike Python which clears it)
func (lr *LuaRuntime) Isolate() error {
	lr.mu.Lock()
	defer lr.mu.Unlock()

	if !lr.ready {
		return fmt.Errorf("Lua runtime is not initialized")
	}

	// For Lua, isolation means creating a snapshot of the current state
	// We don't actually need to do anything special here since Lua state
	// is already isolated by design. Just return success.
	return nil
}

// RefreshRuntimeState method already exists in lua_introspection.go
