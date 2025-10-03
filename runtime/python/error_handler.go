package python

import (
	"fmt"
	"regexp"
	"strings"
	"funterm/errors"
)

// PythonErrorHandler provides enhanced error handling for Python runtime
type PythonErrorHandler struct {
	errorPatterns map[string]ErrorSuggestion
}

// ErrorSuggestion provides context and suggestions for common errors
type ErrorSuggestion struct {
	Description string
	Suggestion  string
	Example     string
	Category    string
}

// ExecutionContext provides context for error analysis
type ExecutionContext struct {
	Command        string
	LineNumber     int
	LocalVariables map[string]interface{}
	CallStack      []string
}

// NewPythonErrorHandler creates a new error handler with predefined patterns
func NewPythonErrorHandler() *PythonErrorHandler {
	handler := &PythonErrorHandler{
		errorPatterns: make(map[string]ErrorSuggestion),
	}
	handler.initializeErrorPatterns()
	return handler
}

// initializeErrorPatterns sets up common error patterns and suggestions
func (h *PythonErrorHandler) initializeErrorPatterns() {
	h.errorPatterns = map[string]ErrorSuggestion{
		"NameError.*name '(.*)' is not defined": {
			Description: "Variable or function name not found",
			Suggestion:  "Check if the variable is defined or imported correctly",
			Example:     "Define the variable first: variable_name = value",
			Category:    "UNDEFINED_VARIABLE",
		},
		"TypeError.*": {
			Description: "Invalid operation between incompatible types",
			Suggestion:  "Convert variables to compatible types before operation",
			Example:     "str(number) + string or number + int(string)",
			Category:    "TYPE_MISMATCH",
		},
		"IndentationError.*": {
			Description: "Incorrect indentation in Python code",
			Suggestion:  "Ensure consistent use of spaces or tabs for indentation",
			Example:     "Use 4 spaces for each indentation level",
			Category:    "SYNTAX_ERROR",
		},
		"AttributeError.*'(.*)' object has no attribute '(.*)'": {
			Description: "Trying to access non-existent attribute or method",
			Suggestion:  "Check available attributes with dir(object) or help(object)",
			Example:     "Use correct method name or check object type",
			Category:    "ATTRIBUTE_ERROR",
		},
		"KeyError.*": {
			Description: "Dictionary key not found",
			Suggestion:  "Check if key exists using 'in' operator or get() method",
			Example:     "dict.get('key', default_value) or 'key' in dict",
			Category:    "KEY_ERROR",
		},
		"IndexError.*list index out of range": {
			Description: "Trying to access list index that doesn't exist",
			Suggestion:  "Check list length with len() before accessing",
			Example:     "if len(list) > index: value = list[index]",
			Category:    "INDEX_ERROR",
		},
		"ImportError.*No module named '(.*)'": {
			Description: "Python module not found or not installed",
			Suggestion:  "Install the module or check the import path",
			Example:     "pip install module_name or check PYTHONPATH",
			Category:    "IMPORT_ERROR",
		},
		"SyntaxError.*": {
			Description: "Invalid Python syntax",
			Suggestion:  "Check for missing parentheses, quotes, or colons",
			Example:     "if condition: (not if condition)",
			Category:    "SYNTAX_ERROR",
		},
		"ValueError.*": {
			Description: "Invalid value for operation or function",
			Suggestion:  "Check the value type and range before using",
			Example:     "int('123') works, int('abc') raises ValueError",
			Category:    "VALUE_ERROR",
		},
		"ZeroDivisionError.*": {
			Description: "Division by zero attempted",
			Suggestion:  "Check divisor is not zero before division",
			Example:     "if divisor != 0: result = dividend / divisor",
			Category:    "ZERO_DIVISION_ERROR",
		},
		"FileNotFoundError.*": {
			Description: "File or directory not found",
			Suggestion:  "Check file path and existence before accessing",
			Example:     "import os; if os.path.exists(file_path): ...",
			Category:    "FILE_NOT_FOUND_ERROR",
		},
		"PermissionError.*": {
			Description: "Insufficient permissions for operation",
			Suggestion:  "Check file permissions and user rights",
			Example:     "chmod 755 file.py or run with appropriate permissions",
			Category:    "PERMISSION_ERROR",
		},
		"MemoryError.*": {
			Description: "Insufficient memory for operation",
			Suggestion:  "Optimize memory usage or process data in chunks",
			Example:     "Use generators or process data in smaller batches",
			Category:    "MEMORY_ERROR",
		},
		"OverflowError.*": {
			Description: "Numeric operation result too large",
			Suggestion:  "Use appropriate data types or check bounds",
			Example:     "Use decimal module for large numbers or check ranges",
			Category:    "OVERFLOW_ERROR",
		},
		"RuntimeError.*": {
			Description: "Runtime error occurred during execution",
			Suggestion:  "Check runtime conditions and input values",
			Example:     "Add proper error handling and validation",
			Category:    "RUNTIME_ERROR",
		},
		"OSError.*": {
			Description: "Operating system related error",
			Suggestion:  "Check system resources and file operations",
			Example:     "Ensure sufficient disk space and proper file handles",
			Category:    "OS_ERROR",
		},
	}
}

// EnhanceError takes a Python error and returns an enhanced error with context
func (h *PythonErrorHandler) EnhanceError(originalError string, context *ExecutionContext) *errors.ExecutionError {
	enhanced := h.analyzeError(originalError)

	errorCode := enhanced.Category
	if errorCode == "" {
		errorCode = "PYTHON_RUNTIME_ERROR"
	}

	message := h.buildEnhancedMessage(originalError, enhanced, context)

	runtimeError := errors.NewRuntimeError("python", errorCode, message)

	// Add additional context if available
	if context != nil {
		runtimeError = runtimeError.WithContext("command", context.Command)
		if context.LineNumber > 0 {
			runtimeError = runtimeError.WithContext("line_number", context.LineNumber)
		}
		if len(context.LocalVariables) > 0 {
			runtimeError = runtimeError.WithContext("variables", context.LocalVariables)
		}
		if len(context.CallStack) > 0 {
			runtimeError = runtimeError.WithContext("call_stack", context.CallStack)
		}
	}

	return runtimeError
}

// analyzeError matches error against known patterns
func (h *PythonErrorHandler) analyzeError(errorStr string) ErrorSuggestion {
	for pattern, suggestion := range h.errorPatterns {
		if matched, _ := regexp.MatchString(pattern, errorStr); matched {
			return suggestion
		}
	}

	// Try to extract more generic error type from the error string
	if strings.Contains(errorStr, "Error:") {
		errorType := strings.Split(errorStr, ":")[0]
		if strings.HasSuffix(errorType, "Error") {
			return ErrorSuggestion{
				Description: fmt.Sprintf("Python %s occurred", errorType),
				Suggestion:  "Check the error context and review your code logic",
				Example:     "Add proper error handling and input validation",
				Category:    errorType,
			}
		}
	}

	// Default suggestion for unrecognized errors
	return ErrorSuggestion{
		Description: "Python runtime error occurred",
		Suggestion:  "Check Python syntax and variable definitions",
		Example:     "Refer to Python documentation for correct syntax",
		Category:    "UNKNOWN_ERROR",
	}
}

// buildEnhancedMessage creates a comprehensive error message
func (h *PythonErrorHandler) buildEnhancedMessage(originalError string, suggestion ErrorSuggestion, context *ExecutionContext) string {
	var parts []string

	parts = append(parts, fmt.Sprintf("Python Error: %s", originalError))
	parts = append(parts, fmt.Sprintf("Description: %s", suggestion.Description))
	parts = append(parts, fmt.Sprintf("Suggestion: %s", suggestion.Suggestion))

	if suggestion.Category != "" {
		parts = append(parts, fmt.Sprintf("Category: %s", suggestion.Category))
	}

	if suggestion.Example != "" {
		parts = append(parts, fmt.Sprintf("Example: %s", suggestion.Example))
	}

	if context != nil && context.Command != "" {
		parts = append(parts, fmt.Sprintf("Command: %s", context.Command))
	}

	if context != nil && context.LineNumber > 0 {
		parts = append(parts, fmt.Sprintf("Line: %d", context.LineNumber))
	}

	return strings.Join(parts, "\n")
}

// GetSuggestionForErrorType returns suggestion for specific error category
func (h *PythonErrorHandler) GetSuggestionForErrorType(category string) string {
	for _, suggestion := range h.errorPatterns {
		if suggestion.Category == category {
			return suggestion.Suggestion
		}
	}
	return "Check Python documentation for more information"
}

// GetSupportedErrorCategories returns all supported error categories
func (h *PythonErrorHandler) GetSupportedErrorCategories() []string {
	categories := make(map[string]bool)
	for _, suggestion := range h.errorPatterns {
		categories[suggestion.Category] = true
	}

	var result []string
	for category := range categories {
		result = append(result, category)
	}
	return result
}

// AddCustomErrorPattern allows adding custom error patterns at runtime
func (h *PythonErrorHandler) AddCustomErrorPattern(pattern string, suggestion ErrorSuggestion) {
	h.errorPatterns[pattern] = suggestion
}

// RemoveCustomErrorPattern allows removing custom error patterns
func (h *PythonErrorHandler) RemoveCustomErrorPattern(pattern string) {
	delete(h.errorPatterns, pattern)
}

// GetErrorPatternCount returns the number of error patterns
func (h *PythonErrorHandler) GetErrorPatternCount() int {
	return len(h.errorPatterns)
}

// IsErrorRecognized checks if an error matches any known pattern
func (h *PythonErrorHandler) IsErrorRecognized(errorStr string) bool {
	for pattern := range h.errorPatterns {
		if matched, _ := regexp.MatchString(pattern, errorStr); matched {
			return true
		}
	}
	return false
}
