package python

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"funterm/runtime"
)

// Dynamic introspection methods implementation

// GetUserDefinedFunctions returns functions defined by the user during the session
func (pr *PythonRuntime) GetUserDefinedFunctions() []string { return []string{} }

// GetImportedModules returns modules that have been imported during the session
func (pr *PythonRuntime) GetImportedModules() []string { return []string{} }

// GetDynamicCompletions returns completions based on current runtime state
func (pr *PythonRuntime) GetDynamicCompletions(input string) ([]string, error) {
	if !pr.ready {
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
			properties, err := pr.GetObjectProperties(objectName)
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
		completions = append(completions, pr.GetModules()...)
		completions = append(completions, pr.GetGlobalVariables()...)

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

	return pr.deduplicateStrings(completions), nil
}

// GetObjectProperties returns properties and methods of a runtime object
func (pr *PythonRuntime) GetObjectProperties(objectName string) ([]string, error) {
	if !pr.ready {
		return []string{}, nil
	}

	// Get object properties from Python runtime
	code := fmt.Sprintf(`
import json

obj = globals().get('%s')
if obj is None:
    print(json.dumps([]))
else:
    properties = []
    # Get attributes and methods
    for attr in dir(obj):
        if not attr.startswith('_'):
            properties.append(attr)
    print(json.dumps(properties))
`, objectName)

	output, err := pr.sendAndAwait(code)
	if err != nil {
		return []string{}, err
	}

	var properties []string
	if err := json.Unmarshal([]byte(output), &properties); err == nil {
		return properties, nil
	}

	return []string{}, nil
}

// GetFunctionParameters returns parameter names and types for a function
func (pr *PythonRuntime) GetFunctionParameters(functionName string) ([]runtime.FunctionParameter, error) {
	if !pr.ready {
		return []runtime.FunctionParameter{}, nil
	}

	// Try to get function signature from predefined signatures first
	signature, err := pr.GetFunctionSignature("", functionName)
	if err == nil && signature != "" {
		// Parse signature to extract parameters
		return pr.parseFunctionSignature(signature), nil
	}

	// Get function parameters from Python runtime
	code := fmt.Sprintf(`
import inspect
import json

func = globals().get('%s')
if func and callable(func):
    try:
        sig = inspect.signature(func)
        params = []
        for name, param in sig.parameters.items():
            param_type = str(param.annotation) if param.annotation != inspect.Parameter.empty else "any"
            params.append({"name": name, "type": param_type})
        print(json.dumps(params))
    except:
        print(json.dumps([{"name": "...", "type": "any"}]))
else:
    print(json.dumps([{"name": "...", "type": "any"}]))
`, functionName)

	output, err := pr.sendAndAwait(code)
	if err != nil {
		return []runtime.FunctionParameter{
			{Name: "...", Type: "any"},
		}, nil
	}

	var params []map[string]interface{}
	if err := json.Unmarshal([]byte(output), &params); err == nil {
		var result []runtime.FunctionParameter
		for _, param := range params {
			if name, ok := param["name"].(string); ok {
				paramType := "any"
				if t, ok := param["type"].(string); ok {
					paramType = t
				}
				result = append(result, runtime.FunctionParameter{
					Name: name,
					Type: paramType,
				})
			}
		}
		return result, nil
	}

	return []runtime.FunctionParameter{
		{Name: "...", Type: "any"},
	}, nil
}

// UpdateCompletionContext updates the completion context after code execution
func (pr *PythonRuntime) UpdateCompletionContext(executedCode string, result interface{}) error {
	return nil
}

// RefreshRuntimeState refreshes the runtime state for completion
func (pr *PythonRuntime) RefreshRuntimeState() error { return nil }

// GetRuntimeObjects returns all objects currently available in the runtime
func (pr *PythonRuntime) GetRuntimeObjects() map[string]interface{} {
	return make(map[string]interface{})
}

// Helper methods for dynamic introspection

// deduplicateStrings removes duplicate strings from a slice
func (pr *PythonRuntime) deduplicateStrings(slice []string) []string {
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
func (pr *PythonRuntime) containsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// isFunctionDefinition checks if code is a function definition
func (pr *PythonRuntime) isFunctionDefinition(code string) bool {
	patterns := []string{
		`^def\s+\w+\s*\(.*\)`,
		`^\w+\s*=\s*lambda\s*`,
		`^@\w+.*\ndef\s+\w+\s*\(.*\)`,
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
func (pr *PythonRuntime) extractFunctionName(code string) string {
	code = strings.TrimSpace(code)

	// Handle "def name(...)" pattern
	if strings.HasPrefix(code, "def") {
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

	// Handle "name = lambda" pattern
	if strings.Contains(code, "=") && strings.Contains(code, "lambda") {
		parts := strings.Split(code, "=")
		if len(parts) >= 1 {
			return strings.TrimSpace(parts[0])
		}
	}

	return ""
}

// isModuleImport checks if code is a module import
func (pr *PythonRuntime) isModuleImport(code string) bool {
	code = strings.TrimSpace(code)
	return strings.HasPrefix(code, "import ") || strings.HasPrefix(code, "from ")
}

// extractModuleName extracts module name from import statement
func (pr *PythonRuntime) extractModuleName(code string) string {
	code = strings.TrimSpace(code)

	if strings.HasPrefix(code, "import ") {
		// Handle "import module" or "import module as alias" or "import module1, module2"
		parts := strings.Fields(code)
		if len(parts) >= 2 {
			// Remove "import" keyword
			modules := strings.Join(parts[1:], " ")
			// Split by comma for multiple imports
			moduleList := strings.Split(modules, ",")
			if len(moduleList) > 0 {
				// Return first module name (remove "as alias" if present)
				firstModule := strings.TrimSpace(moduleList[0])
				if strings.Contains(firstModule, " as ") {
					return strings.Split(firstModule, " as ")[0]
				}
				return firstModule
			}
		}
	} else if strings.HasPrefix(code, "from ") {
		// Handle "from module import ..."
		parts := strings.Fields(code)
		if len(parts) >= 2 {
			return parts[1]
		}
	}

	return ""
}

// parseFunctionSignature parses a function signature to extract parameters
func (pr *PythonRuntime) parseFunctionSignature(signature string) []runtime.FunctionParameter {
	// Simple parsing for Python function signatures
	// Format: "function_name(param1: type1, param2: type2) -> return_type"

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
					// Handle parameter with type annotation
					if strings.Contains(param, ":") {
						paramParts := strings.Split(param, ":")
						name := strings.TrimSpace(paramParts[0])
						paramType := "any"
						if len(paramParts) > 1 {
							paramType = strings.TrimSpace(paramParts[1])
						}
						params = append(params, runtime.FunctionParameter{
							Name: name,
							Type: paramType,
						})
					} else {
						// Parameter without type annotation
						params = append(params, runtime.FunctionParameter{
							Name: param,
							Type: "any",
						})
					}
				}
			}
		}
	}

	return params
}

// getUserDefinedFunctionsFromRuntime extracts user-defined functions from current runtime
func (pr *PythonRuntime) getUserDefinedFunctionsFromRuntime() []string { return []string{} }

// getImportedModulesFromRuntime extracts imported modules from current runtime
func (pr *PythonRuntime) getImportedModulesFromRuntime() []string { return []string{} }

// getRuntimeObjectsFromRuntime extracts all objects from current runtime
func (pr *PythonRuntime) getRuntimeObjectsFromRuntime() map[string]interface{} {
	return make(map[string]interface{})
}

// getCurrentLineNumber returns the current line number (approximation)
func (pr *PythonRuntime) getCurrentLineNumber() int {
	// This is no longer possible to track accurately.
	return 0
}

// getLocalVariables returns a copy of current local variables
func (pr *PythonRuntime) getLocalVariables() map[string]interface{} {
	pr.mutex.RLock()
	defer pr.mutex.RUnlock()

	variables := make(map[string]interface{})
	for k, v := range pr.variables {
		variables[k] = v
	}
	return variables
}
