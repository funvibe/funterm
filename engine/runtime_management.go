package engine

import (
	"fmt"
	"strings"

	"funterm/errors"
	"funterm/factory"
	"funterm/runtime"
	"funterm/runtime/python"
)

// GetRuntimeManager returns the runtime manager
func (e *ExecutionEngine) GetRuntimeManager() *runtime.RuntimeManager {
	return e.runtimeManager
}

// GetRuntimeRegistry returns the runtime registry
func (e *ExecutionEngine) GetRuntimeRegistry() *factory.RuntimeRegistry {
	return e.runtimeRegistry
}

// RegisterRuntime registers a language runtime
func (e *ExecutionEngine) RegisterRuntime(rt runtime.LanguageRuntime) error {
	return e.runtimeManager.RegisterRuntime(rt)
}

// InitializeRuntimes initializes all registered runtimes
func (e *ExecutionEngine) InitializeRuntimes() error {
	if err := e.runtimeManager.InitializeAll(); err != nil {
		return err
	}

	// Set verbose mode for Python runtimes after initialization
	return e.setVerboseForPythonRuntimes()
}

// setVerboseForPythonRuntimes sets verbose mode for all Python runtimes
func (e *ExecutionEngine) setVerboseForPythonRuntimes() error {
	for _, runtime := range e.runtimeManager.GetAllRuntimes() {
		if pythonRuntime, ok := runtime.(*python.PythonRuntime); ok {
			pythonRuntime.SetVerbose(e.verbose)
		}
	}
	return nil
}

// CleanupRuntimes cleans up all registered runtimes
func (e *ExecutionEngine) CleanupRuntimes() error {
	return e.runtimeManager.CleanupAll()
}

// ListAvailableLanguages returns the names of available languages
func (e *ExecutionEngine) ListAvailableLanguages() []string {
	// Get languages from runtime manager
	languages := e.runtimeManager.ListRuntimes()

	// Also get languages from runtime registry
	if e.runtimeRegistry != nil {
		registryLanguages := e.runtimeRegistry.GetSupportedLanguages()
		// Merge and deduplicate
		languageMap := make(map[string]bool)
		for _, lang := range languages {
			languageMap[lang] = true
		}
		for _, lang := range registryLanguages {
			languageMap[lang] = true
		}

		// Convert back to slice
		languages = make([]string, 0, len(languageMap))
		for lang := range languageMap {
			languages = append(languages, lang)
		}
	}

	return languages
}

// IsLanguageAvailable checks if a language runtime is available
func (e *ExecutionEngine) IsLanguageAvailable(language string) bool {
	// Check runtime manager first
	if e.runtimeManager.IsRuntimeReady(language) {
		return true
	}

	// Check runtime registry
	if e.runtimeRegistry != nil {
		// Check if there's a factory for this language
		if _, err := e.runtimeRegistry.GetFactoryForLanguage(language); err == nil {
			return true
		}
	}

	return false
}

// getRuntimeByName gets or creates a runtime by name
func (e *ExecutionEngine) getRuntimeByName(runtimeName string) (runtime.LanguageRuntime, error) {
	// Handle alias 'js' for 'node'
	if runtimeName == "js" {
		runtimeName = "node"
	}

	// Try to get the runtime from the runtime manager first
	rt, err := e.runtimeManager.GetRuntime(runtimeName)
	if err == nil {
		// Runtime found in manager, use it
		if !rt.IsReady() {
			return nil, errors.NewSystemError("RUNTIME_NOT_READY", fmt.Sprintf("%s runtime is not ready", runtimeName))
		}
		return rt, nil
	}

	// Try to get or create runtime from cache
	if e.runtimeRegistry != nil {
		runtime, err := e.GetOrCreateRuntime(runtimeName)
		if err == nil {
			return runtime, nil
		}
	}

	return nil, errors.NewUserError("UNSUPPORTED_LANGUAGE", fmt.Sprintf("unsupported language '%s'", runtimeName))
}

// getRuntimeForLanguage gets or creates a runtime for the specified language
func (e *ExecutionEngine) getRuntimeForLanguage(language string) (runtime.LanguageRuntime, error) {
	// Handle alias 'js' for 'node'
	if language == "js" {
		language = "node"
	}

	// Try to get from runtime manager first
	rt, err := e.runtimeManager.GetRuntime(language)
	if err == nil {
		return rt, nil
	}

	// Try to get or create from cache
	if e.runtimeRegistry != nil {
		runtime, err := e.GetOrCreateRuntime(language)
		if err == nil {
			return runtime, nil
		}
	}

	return nil, errors.NewUserError("UNSUPPORTED_LANGUAGE", fmt.Sprintf("unsupported language '%s'", language))
}

// tryFixEmptyFunctionCall attempts to fix empty function calls by adding a dummy argument
func (e *ExecutionEngine) tryFixEmptyFunctionCall(input string) (string, bool) {
	// Check if input matches pattern "language.function()"
	if len(input) < 4 || !strings.HasSuffix(input, "()") {
		return input, false
	}

	// Find the last dot before the parentheses
	lastDot := strings.LastIndex(input[:len(input)-2], ".")
	if lastDot == -1 {
		return input, false
	}

	// Extract language part
	languagePart := input[:lastDot]
	if !e.IsLanguageAvailable(languagePart) {
		return input, false
	}

	// Add a dummy argument
	return input[:len(input)-1] + "dummy" + ")", true
}

// extractLanguageAndFunction extracts language and function name from input like "language.function()"
func (e *ExecutionEngine) extractLanguageAndFunction(input string) (string, string) {
	// Remove trailing parentheses
	if strings.HasSuffix(input, "()") {
		input = input[:len(input)-2]
	}

	// Split by last dot to separate language from function
	lastDot := strings.LastIndex(input, ".")
	if lastDot == -1 {
		return "", ""
	}

	language := input[:lastDot]
	function := input[lastDot+1:]

	// Validate that language is available
	if !e.IsLanguageAvailable(language) {
		return "", ""
	}

	return language, function
}

// convertValueToString converts a value to its string representation
func (e *ExecutionEngine) convertValueToString(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", v)
	case float32, float64:
		return fmt.Sprintf("%v", v)
	case bool:
		return fmt.Sprintf("%t", v)
	case nil:
		return ""
	case []interface{}:
		// For arrays, join elements with space (like print usually does)
		var elements []string
		for _, elem := range v {
			elements = append(elements, e.convertValueToString(elem))
		}
		return strings.Join(elements, " ")
	case map[string]interface{}:
		// For objects, convert to a simple string representation
		return fmt.Sprintf("%v", v)
	default:
		// For other types, use default string conversion
		return fmt.Sprintf("%v", v)
	}
}

// isStatementWithoutReturnValue checks if the given code is a statement that shouldn't return a value
func (e *ExecutionEngine) isStatementWithoutReturnValue(code string, runtimeName string) bool {
	trimmedCode := strings.TrimSpace(code)

	switch runtimeName {
	case "python":
		// Python statements that don't return values
		return strings.HasPrefix(trimmedCode, "def ") || // function definition
			strings.HasPrefix(trimmedCode, "class ") || // class definition
			strings.HasPrefix(trimmedCode, "if ") && strings.Contains(code, ":") || // if statement
			strings.HasPrefix(trimmedCode, "for ") && strings.Contains(code, ":") || // for loop
			strings.HasPrefix(trimmedCode, "while ") && strings.Contains(code, ":") || // while loop
			strings.HasPrefix(trimmedCode, "try:") || // try block
			strings.HasPrefix(trimmedCode, "with ") || // with statement
			strings.HasPrefix(trimmedCode, "import ") || // import statement
			strings.HasPrefix(trimmedCode, "from ") // from import statement

	case "lua":
		// Lua statements that don't return values
		return strings.HasPrefix(trimmedCode, "function ") || // function definition
			strings.HasPrefix(trimmedCode, "local function ") || // local function definition
			strings.HasPrefix(trimmedCode, "if ") && strings.Contains(code, " then") || // if statement
			strings.HasPrefix(trimmedCode, "for ") && strings.Contains(code, " do") || // for loop
			strings.HasPrefix(trimmedCode, "while ") && strings.Contains(code, " do") || // while loop
			(strings.Contains(trimmedCode, " = ") && !strings.Contains(trimmedCode, "return ") && !strings.Contains(trimmedCode, "print(")) // assignment without return or print

	default:
		return false
	}
}
