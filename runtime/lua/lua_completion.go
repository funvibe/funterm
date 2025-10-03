package lua

import (
	"fmt"
	"sort"
	"strings"

	"funterm/errors"

	lua "github.com/yuin/gopher-lua"
)

// Completion interface implementation

// GetModules returns available modules for the Lua runtime
func (lr *LuaRuntime) GetModules() []string {
	if !lr.ready {
		return []string{}
	}

	// Get standard Lua modules
	modules := []string{
		"math", "string", "table", "io", "os",
		"debug", "package", "coroutine", "utf8", "ffi",
	}

	// Add built-in modules
	builtinModules := lr.moduleManager.ListModules()
	modules = append(modules, builtinModules...)

	// Sort modules alphabetically
	sort.Strings(modules)
	return modules
}

// GetModuleFunctions returns available functions for a specific Lua module
func (lr *LuaRuntime) GetModuleFunctions(module string) []string {
	if !lr.ready {
		return []string{}
	}

	switch module {
	case "math":
		functions := []string{
			"abs", "acos", "asin", "atan", "atan2", "ceil", "cos", "cosh", "deg",
			"exp", "floor", "fmod", "frexp", "huge", "ldexp", "log", "log10",
			"max", "maxinteger", "min", "mininteger", "modf", "pi", "pow",
			"rad", "random", "randomseed", "sin", "sinh", "sqrt", "tan", "tanh",
			"tointeger", "type", "ult",
		}
		sort.Strings(functions)
		return functions
	case "string":
		functions := []string{
			"byte", "char", "dump", "find", "format", "gmatch", "gsub",
			"len", "lower", "match", "pack", "packsize", "rep", "reverse",
			"sub", "unpack", "upper",
		}
		sort.Strings(functions)
		return functions
	case "table":
		functions := []string{
			"concat", "insert", "move", "pack", "remove", "sort", "unpack",
		}
		sort.Strings(functions)
		return functions
	case "io":
		functions := []string{
			"close", "flush", "input", "lines", "open", "output", "popen",
			"read", "tmpfile", "type", "write",
		}
		sort.Strings(functions)
		return functions
	case "os":
		functions := []string{
			"clock", "date", "difftime", "execute", "exit", "getenv",
			"remove", "rename", "setlocale", "time", "tmpname",
		}
		sort.Strings(functions)
		return functions
	case "debug":
		functions := []string{
			"gethook", "getinfo", "getlocal", "getmetatable", "getregistry",
			"getupvalue", "sethook", "setlocal", "setmetatable", "setupvalue",
			"traceback",
		}
		sort.Strings(functions)
		return functions
	case "package":
		functions := []string{
			"config", "cpath", "loaded", "loadlib", "path", "preload",
			"searchers", "searchpath",
		}
		sort.Strings(functions)
		return functions
	case "coroutine":
		functions := []string{
			"create", "isyieldable", "resume", "running", "status", "wrap", "yield",
		}
		sort.Strings(functions)
		return functions
	case "utf8":
		functions := []string{
			"char", "charpattern", "codes", "codepoint", "len", "offset",
		}
		sort.Strings(functions)
		return functions
	case "ffi":
		functions := []string{
			"cdef", "C", "cast", "typeof", "new", "copy", "fill", "sizeof",
			"alignof", "offsetof", "istype", "metatype", "gc", "string",
		}
		sort.Strings(functions)
		return functions
	case "json":
		functions := []string{
			"encode", "decode",
		}
		sort.Strings(functions)
		return functions
	case "http":
		functions := []string{
			"get", "post",
		}
		sort.Strings(functions)
		return functions
	case "fs":
		functions := []string{
			"exists", "read", "write", "list",
		}
		sort.Strings(functions)
		return functions
	default:
		return []string{}
	}
}

// GetFunctionSignature returns the signature of a function in a module
func (lr *LuaRuntime) GetFunctionSignature(module, function string) (string, error) {
	if !lr.ready {
		return "", errors.NewRuntimeError("lua", "LUA_RUNTIME_NOT_INITIALIZED", "runtime is not initialized")
	}

	// Define function signatures for common Lua functions
	signatures := map[string]map[string]string{
		"math": {
			"abs":    "abs(x) -> number",
			"acos":   "acos(x) -> number",
			"asin":   "asin(x) -> number",
			"atan":   "atan(x) -> number",
			"atan2":  "atan2(y, x) -> number",
			"ceil":   "ceil(x) -> number",
			"cos":    "cos(x) -> number",
			"cosh":   "cosh(x) -> number",
			"deg":    "deg(x) -> number",
			"exp":    "exp(x) -> number",
			"floor":  "floor(x) -> number",
			"fmod":   "fmod(x, y) -> number",
			"log":    "log(x) -> number",
			"log10":  "log10(x) -> number",
			"max":    "max(x, ...) -> number",
			"min":    "min(x, ...) -> number",
			"pow":    "pow(x, y) -> number",
			"rad":    "rad(x) -> number",
			"random": "random([m, n]) -> number",
			"sin":    "sin(x) -> number",
			"sinh":   "sinh(x) -> number",
			"sqrt":   "sqrt(x) -> number",
			"tan":    "tan(x) -> number",
			"tanh":   "tanh(x) -> number",
		},
		"string": {
			"byte":    "byte(s [, i [, j]]) -> number, ...",
			"char":    "char(...) -> string",
			"find":    "find(s, pattern [, init [, plain]]) -> start, end",
			"format":  "format(formatstring, ...) -> string",
			"gmatch":  "gmatch(s, pattern) -> function",
			"gsub":    "gsub(s, pattern, repl [, n]) -> string",
			"len":     "len(s) -> number",
			"lower":   "lower(s) -> string",
			"match":   "match(s, pattern [, init]) -> string",
			"rep":     "rep(s, n [, sep]) -> string",
			"reverse": "reverse(s) -> string",
			"sub":     "sub(s, i [, j]) -> string",
			"upper":   "upper(s) -> string",
		},
		"table": {
			"concat": "concat(table [, sep [, i [, j]]]) -> string",
			"insert": "insert(table, [pos,] value)",
			"move":   "move(a1, f, e, t [,a2]) -> a2",
			"remove": "remove(table [, pos]) -> value",
			"sort":   "sort(table [, comp])",
			"unpack": "unpack(table [, i [, j]]) -> value1, value2, ...",
		},
		"json": {
			"encode": "encode(table) -> string",
			"decode": "decode(string) -> table",
		},
		"http": {
			"get":  "get(url) -> table, error",
			"post": "post(url, data) -> table, error",
		},
		"fs": {
			"exists": "exists(path) -> boolean, error",
			"read":   "read(path) -> string, error",
			"write":  "write(path, data) -> boolean, error",
			"list":   "list(path) -> table, error",
		},
	}

	if moduleSignatures, ok := signatures[module]; ok {
		if signature, ok := moduleSignatures[function]; ok {
			return signature, nil
		}
	}

	return "", errors.NewRuntimeError("lua", "LUA_FUNCTION_NOT_FOUND", fmt.Sprintf("function '%s.%s' not found", module, function))
}

// GetGlobalVariables returns available global variables
func (lr *LuaRuntime) GetGlobalVariables() []string {
	if !lr.ready {
		return []string{}
	}

	// Get the global table
	globalTable := lr.state.GetGlobal("_G")
	if globalTable.Type() != lua.LTTable {
		return []string{}
	}

	var variables []string
	globalTable.(*lua.LTable).ForEach(func(key, value lua.LValue) {
		if key.Type() == lua.LTString {
			keyStr := key.String()
			// Include important global variables like "_G" and "_VERSION"
			// Skip other internal Lua variables, modules, and internal funterm functions
			if (keyStr == "_G" || keyStr == "_VERSION" || !strings.HasPrefix(keyStr, "_")) &&
				!lr.isModule(keyStr) &&
				!lr.isInternalFunction(keyStr) &&
				keyStr != "package" &&
				keyStr != "string" &&
				keyStr != "table" &&
				keyStr != "math" &&
				keyStr != "io" &&
				keyStr != "os" &&
				keyStr != "debug" &&
				keyStr != "coroutine" &&
				keyStr != "utf8" {
				variables = append(variables, keyStr)
			}
		}
	})

	return variables
}

// GetCompletionSuggestions returns completion suggestions for a given input
func (lr *LuaRuntime) GetCompletionSuggestions(input string) []string {
	if !lr.ready {
		return []string{}
	}

	input = strings.TrimSpace(input)
	if input == "" {
		// Return all modules and global variables
		suggestions := lr.GetModules()
		suggestions = append(suggestions, lr.GetGlobalVariables()...)
		return suggestions
	}

	// Check if input contains a dot (module.function)
	if strings.Contains(input, ".") {
		parts := strings.Split(input, ".")
		if len(parts) == 2 {
			module := parts[0]
			prefix := parts[1]

			// Check if it's a valid module
			if lr.isModule(module) {
				functions := lr.GetModuleFunctions(module)
				var suggestions []string
				for _, fn := range functions {
					if strings.HasPrefix(fn, prefix) {
						suggestions = append(suggestions, fmt.Sprintf("%s.%s", module, fn))
					}
				}
				return suggestions
			}
		}
	} else {
		// Check for module or global variable completion
		var suggestions []string

		// Check modules
		modules := lr.GetModules()
		for _, module := range modules {
			if strings.HasPrefix(module, input) {
				suggestions = append(suggestions, module+".")
			}
		}

		// Check global variables
		variables := lr.GetGlobalVariables()
		for _, variable := range variables {
			if strings.HasPrefix(variable, input) {
				suggestions = append(suggestions, variable)
			}
		}

		return suggestions
	}

	return []string{}
}

// isModule checks if a name is a valid Lua module
func (lr *LuaRuntime) isModule(name string) bool {
	modules := lr.GetModules()
	for _, module := range modules {
		if module == name {
			return true
		}
	}
	return false
}

// isInternalFunction checks if a name is an internal funterm function that should not appear in autocompletion
func (lr *LuaRuntime) isInternalFunction(name string) bool {
	internalFunctions := []string{
		"matchesBitstringPattern", // Internal bitstring pattern matching function
		"execution_engine",        // Internal execution engine reference
		"print_funterm",           // Internal print function
	}

	for _, internal := range internalFunctions {
		if name == internal {
			return true
		}
	}
	return false
}
