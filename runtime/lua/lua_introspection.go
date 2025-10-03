package lua

import (
	"fmt"
	"regexp"
	"strings"

	"funterm/runtime"

	lua "github.com/yuin/gopher-lua"
)

// Dynamic introspection methods implementation

// GetUserDefinedFunctions returns functions defined by the user during the session
func (lr *LuaRuntime) GetUserDefinedFunctions() []string {
	if !lr.ready {
		return []string{}
	}

	// Get the global table
	globalTable := lr.state.GetGlobal("_G")
	if globalTable.Type() != lua.LTTable {
		return lr.userDefinedFunctions
	}

	var functions []string
	globalTable.(*lua.LTable).ForEach(func(key, value lua.LValue) {
		if key.Type() == lua.LTString {
			keyStr := key.String()
			// Check if it's a function and not a built-in module or internal function
			if value.Type() == lua.LTFunction &&
				!lr.isModule(keyStr) &&
				!lr.isInternalFunction(keyStr) &&
				!strings.HasPrefix(keyStr, "_") {
				functions = append(functions, keyStr)
			}
		}
	})

	// Add any user-defined functions tracked during execution
	functions = append(functions, lr.userDefinedFunctions...)

	return lr.deduplicateStrings(functions)
}

// GetImportedModules returns modules that have been imported during the session
func (lr *LuaRuntime) GetImportedModules() []string {
	if !lr.ready {
		return []string{}
	}

	// Combine static modules with dynamically imported ones
	allModules := lr.GetModules()
	allModules = append(allModules, lr.importedModules...)

	return lr.deduplicateStrings(allModules)
}

// GetDynamicCompletions returns completions based on current runtime state
func (lr *LuaRuntime) GetDynamicCompletions(input string) ([]string, error) {
	if !lr.ready {
		return []string{}, nil
	}

	input = strings.TrimSpace(input)
	var completions []string

	// Check for object property access (object.property)
	if strings.Contains(input, ".") {
		parts := strings.Split(input, ".")
		if len(parts) == 2 {
			objectName := parts[0]
			prefix := parts[1]

			// Get properties of the object
			properties, err := lr.GetObjectProperties(objectName)
			if err == nil {
				for _, prop := range properties {
					if strings.HasPrefix(prop, prefix) {
						completions = append(completions, fmt.Sprintf("%s.%s", objectName, prop))
					}
				}
			}
		}
	} else {
		// Regular completion - combine modules, functions, and variables
		completions = append(completions, lr.GetModules()...)
		completions = append(completions, lr.GetUserDefinedFunctions()...)
		completions = append(completions, lr.GetGlobalVariables()...)

		// Filter by input prefix
		if input != "" {
			var filtered []string
			for _, item := range completions {
				if strings.HasPrefix(item, input) {
					filtered = append(filtered, item)
				}
			}
			completions = filtered
		}
	}

	return lr.deduplicateStrings(completions), nil
}

// GetObjectProperties returns properties and methods of a runtime object
func (lr *LuaRuntime) GetObjectProperties(objectName string) ([]string, error) {
	if !lr.ready {
		return []string{}, nil
	}

	// Get the object from Lua state
	obj := lr.state.GetGlobal(objectName)
	if obj.Type() == lua.LTNil {
		return []string{}, fmt.Errorf("object '%s' not found", objectName)
	}

	var properties []string

	if obj.Type() == lua.LTTable {
		// Extract table keys/properties
		obj.(*lua.LTable).ForEach(func(key, value lua.LValue) {
			if key.Type() == lua.LTString {
				properties = append(properties, key.String())
			}
		})
	}

	return properties, nil
}

// GetFunctionParameters returns parameter names and types for a function
func (lr *LuaRuntime) GetFunctionParameters(functionName string) ([]runtime.FunctionParameter, error) {
	if !lr.ready {
		return []runtime.FunctionParameter{}, nil
	}

	// Try to get function signature from predefined signatures first
	signature, err := lr.GetFunctionSignature("", functionName)
	if err == nil && signature != "" {
		// Parse signature to extract parameters
		return lr.parseFunctionSignature(signature), nil
	}

	// For user-defined functions, we can't easily extract parameters
	// Return a placeholder
	return []runtime.FunctionParameter{
		{Name: "...", Type: "any"},
	}, nil
}

// UpdateCompletionContext updates the completion context after code execution
func (lr *LuaRuntime) UpdateCompletionContext(executedCode string, result interface{}) error {
	if !lr.ready {
		return nil
	}

	// Add to execution history
	lr.executionHistory = append(lr.executionHistory, executedCode)

	// Track user-defined functions
	if lr.isFunctionDefinition(executedCode) {
		funcName := lr.extractFunctionName(executedCode)
		if funcName != "" && !lr.containsString(lr.userDefinedFunctions, funcName) {
			lr.userDefinedFunctions = append(lr.userDefinedFunctions, funcName)
		}
	}

	// Track imported modules
	if lr.isModuleImport(executedCode) {
		moduleName := lr.extractModuleName(executedCode)
		if moduleName != "" && !lr.containsString(lr.importedModules, moduleName) {
			lr.importedModules = append(lr.importedModules, moduleName)
		}
	}

	return nil
}

// RefreshRuntimeState refreshes the runtime state for completion
func (lr *LuaRuntime) RefreshRuntimeState() error {
	if !lr.ready {
		return nil
	}

	// Clear cached data and rebuild from current runtime state
	lr.userDefinedFunctions = lr.getUserDefinedFunctionsFromRuntime()
	lr.importedModules = lr.getImportedModulesFromRuntime()
	lr.runtimeObjects = lr.getRuntimeObjectsFromRuntime()

	return nil
}

// GetRuntimeObjects returns all objects currently available in the runtime
func (lr *LuaRuntime) GetRuntimeObjects() map[string]interface{} {
	if !lr.ready {
		return make(map[string]interface{})
	}

	// Refresh runtime objects
	lr.runtimeObjects = lr.getRuntimeObjectsFromRuntime()
	return lr.runtimeObjects
}

// Helper methods for dynamic introspection

// deduplicateStrings removes duplicate strings from a slice
func (lr *LuaRuntime) deduplicateStrings(slice []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0)
	for _, item := range slice {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}
	return result
}

// containsString checks if a string slice contains a specific string
func (lr *LuaRuntime) containsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// isFunctionDefinition checks if code is a function definition
func (lr *LuaRuntime) isFunctionDefinition(code string) bool {
	patterns := []string{
		`^function\s+\w+\s*\(.*\)`,
		`^\w+\s*=\s*function\s*\(.*\)`,
		`^local\s+function\s+\w+\s*\(.*\)`,
		`^local\s+\w+\s*=\s*function\s*\(.*\)`,
	}

	for _, pattern := range patterns {
		matched, _ := regexp.MatchString(pattern, strings.TrimSpace(code))
		if matched {
			return true
		}
	}
	return false
}

// extractFunctionName extracts function name from function definition code
func (lr *LuaRuntime) extractFunctionName(code string) string {
	code = strings.TrimSpace(code)

	// Handle "function name(...)" pattern
	if strings.HasPrefix(code, "function") {
		parts := strings.Fields(code)
		if len(parts) >= 2 {
			// Extract function name (might have parentheses attached)
			funcName := parts[1]
			if strings.Contains(funcName, "(") {
				funcName = strings.Split(funcName, "(")[0]
			}
			return funcName
		}
	}

	// Handle "name = function(...)" pattern
	if strings.Contains(code, "=") && strings.Contains(code, "function") {
		parts := strings.Split(code, "=")
		if len(parts) >= 1 {
			funcName := strings.TrimSpace(parts[0])
			// Remove "local" keyword if present
			if strings.HasPrefix(funcName, "local") {
				funcName = strings.TrimSpace(strings.TrimPrefix(funcName, "local"))
			}
			return funcName
		}
	}

	return ""
}

// isModuleImport checks if code is a module import
func (lr *LuaRuntime) isModuleImport(code string) bool {
	return strings.Contains(strings.TrimSpace(code), "require")
}

// extractModuleName extracts module name from require statement
func (lr *LuaRuntime) extractModuleName(code string) string {
	code = strings.TrimSpace(code)
	if strings.Contains(code, "require") {
		// Extract string inside require() or require ""
		re := regexp.MustCompile(`require\s*["\']([^"\']*)["\']`)
		matches := re.FindStringSubmatch(code)
		if len(matches) > 1 {
			return matches[1]
		}
	}
	return ""
}

// parseFunctionSignature parses a function signature to extract parameters
func (lr *LuaRuntime) parseFunctionSignature(signature string) []runtime.FunctionParameter {
	// Simple parsing for Lua function signatures
	// Format: "function_name(param1, param2) -> return_type"

	params := make([]runtime.FunctionParameter, 0)

	// Extract parameters part
	openParen := strings.Index(signature, "(")
	closeParen := strings.Index(signature, ")")

	if openParen != -1 && closeParen != -1 && closeParen > openParen {
		paramsStr := signature[openParen+1 : closeParen]
		if paramsStr != "" {
			paramList := strings.Split(paramsStr, ",")
			for _, param := range paramList {
				param = strings.TrimSpace(param)
				if param != "" {
					// Assume all parameters are "any" type for Lua
					params = append(params, runtime.FunctionParameter{
						Name: param,
						Type: "any",
					})
				}
			}
		}
	}

	return params
}

// getUserDefinedFunctionsFromRuntime extracts user-defined functions from current runtime
func (lr *LuaRuntime) getUserDefinedFunctionsFromRuntime() []string {
	var functions []string

	globalTable := lr.state.GetGlobal("_G")
	if globalTable.Type() == lua.LTTable {
		globalTable.(*lua.LTable).ForEach(func(key, value lua.LValue) {
			if key.Type() == lua.LTString {
				keyStr := key.String()
				if value.Type() == lua.LTFunction &&
					!lr.isModule(keyStr) &&
					!strings.HasPrefix(keyStr, "_") {
					functions = append(functions, keyStr)
				}
			}
		})
	}

	return functions
}

// getImportedModulesFromRuntime extracts imported modules from current runtime
func (lr *LuaRuntime) getImportedModulesFromRuntime() []string {
	// For Lua, check package.loaded for imported modules
	var modules []string

	packageTable := lr.state.GetGlobal("package")
	if packageTable.Type() == lua.LTTable {
		loadedTable := packageTable.(*lua.LTable).RawGetString("loaded")
		if loadedTable.Type() == lua.LTTable {
			loadedTable.(*lua.LTable).ForEach(func(key, value lua.LValue) {
				if key.Type() == lua.LTString {
					moduleName := key.String()
					// Skip internal modules
					if !strings.HasPrefix(moduleName, "_") && moduleName != "" {
						modules = append(modules, moduleName)
					}
				}
			})
		}
	}

	return modules
}

// getRuntimeObjectsFromRuntime extracts all objects from current runtime
func (lr *LuaRuntime) getRuntimeObjectsFromRuntime() map[string]interface{} {
	objects := make(map[string]interface{})

	globalTable := lr.state.GetGlobal("_G")
	if globalTable.Type() == lua.LTTable {
		globalTable.(*lua.LTable).ForEach(func(key, value lua.LValue) {
			if key.Type() == lua.LTString {
				keyStr := key.String()
				// Skip internal variables and functions
				if !strings.HasPrefix(keyStr, "_") &&
					value.Type() != lua.LTFunction &&
					!lr.isModule(keyStr) {
					objects[keyStr] = lr.luaToGo(value)
				}
			}
		})
	}

	return objects
}
