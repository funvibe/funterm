package errors

import (
	"fmt"
	"go-parser/pkg/ast"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"time"
)

// ErrorType represents the type of error
type ErrorType string

const (
	ErrorTypeRuntime    ErrorType = "RUNTIME"
	ErrorTypeValidation ErrorType = "VALIDATION"
	ErrorTypeSystem     ErrorType = "SYSTEM"
	ErrorTypeNetwork    ErrorType = "NETWORK"
	ErrorTypeUser       ErrorType = "USER"
)

// ErrorSeverity represents the severity level of an error
type ErrorSeverity string

const (
	SeverityDebug   ErrorSeverity = "DEBUG"
	SeverityInfo    ErrorSeverity = "INFO"
	SeverityWarning ErrorSeverity = "WARNING"
	SeverityError   ErrorSeverity = "ERROR"
	SeverityFatal   ErrorSeverity = "FATAL"
)

// ExecutionError represents a structured error with detailed information
type ExecutionError struct {
	Code       string                 `json:"code"`
	Message    string                 `json:"message"`
	Language   string                 `json:"language,omitempty"`
	Line       int                    `json:"line,omitempty"`
	Col        int                    `json:"col,omitempty"`
	Positions  []ast.Position         `json:"positions,omitempty"`
	StackTrace string                 `json:"stack_trace,omitempty"`
	Context    map[string]interface{} `json:"context,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
	Severity   ErrorSeverity          `json:"severity"`
	Type       ErrorType              `json:"type"`
	Cause      error                  `json:"-"`
	Wrapped    []error                `json:"-"`
}

// Error implements the error interface
func (e *ExecutionError) Error() string {
	var builder strings.Builder

	// Format: [TYPE][CODE] message
	builder.WriteString(fmt.Sprintf("[%s][%s] %s", e.Type, e.Code, e.Message))

	// Handle multiple positions without duplicates
	if len(e.Positions) > 0 {
		// Create a map to track unique positions (line:column)
		uniquePositions := make(map[string]bool)
		var uniquePosList []ast.Position

		for _, pos := range e.Positions {
			posKey := fmt.Sprintf("%d:%d", pos.Line, pos.Column)
			if !uniquePositions[posKey] {
				uniquePositions[posKey] = true
				uniquePosList = append(uniquePosList, pos)
			}
		}

		if len(uniquePosList) > 0 {
			// For backward compatibility, if there's only one position, use the old format
			if len(uniquePosList) == 1 {
				pos := uniquePosList[0]
				if pos.Column > 0 {
					builder.WriteString(fmt.Sprintf(" line %d col %d", pos.Line, pos.Column))
				} else {
					builder.WriteString(fmt.Sprintf(" line %d col 0", pos.Line))
				}
			} else {
				// For multiple positions, use the new format with "at" and "and"
				builder.WriteString(" at")
				for i, pos := range uniquePosList {
					if i > 0 {
						builder.WriteString(" and")
					}
					builder.WriteString(fmt.Sprintf(" line %d col %d", pos.Line, pos.Column))
				}
			}
		}
	} else {
		// Fallback to single Line/Col for backward compatibility
		var posInfo string
		if e.Line > 0 && e.Col > 0 {
			posInfo = fmt.Sprintf(" line %d col %d", e.Line, e.Col)
		} else if e.Line > 0 {
			posInfo = fmt.Sprintf(" line %d col 0", e.Line)
		}
		if posInfo != "" {
			builder.WriteString(posInfo)
		}
	}

	return builder.String()
}

// Unwrap returns the underlying error
func (e *ExecutionError) Unwrap() error {
	return e.Cause
}

// Is checks if the error matches the target
func (e *ExecutionError) Is(target error) bool {
	if other, ok := target.(*ExecutionError); ok {
		return e.Code == other.Code && e.Type == other.Type
	}
	return false
}

// WithContext adds context information to the error
func (e *ExecutionError) WithContext(key string, value interface{}) *ExecutionError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// WithLanguage sets the language for the error
func (e *ExecutionError) WithLanguage(language string) *ExecutionError {
	e.Language = language
	return e
}

// WithSeverity sets the severity level for the error
func (e *ExecutionError) WithSeverity(severity ErrorSeverity) *ExecutionError {
	e.Severity = severity
	return e
}

// WithType sets the error type
func (e *ExecutionError) WithType(errorType ErrorType) *ExecutionError {
	e.Type = errorType
	return e
}

// WithPosition sets the line and column for the error
func (e *ExecutionError) WithPosition(line, col int) *ExecutionError {
	e.Line = line
	e.Col = col
	return e
}

// WithStackTrace captures and adds stack trace information
func (e *ExecutionError) WithStackTrace() *ExecutionError {
	buf := make([]byte, 4096)
	n := runtime.Stack(buf, false)
	e.StackTrace = string(buf[:n])
	return e
}

// Wrap wraps another error
func (e *ExecutionError) Wrap(err error) *ExecutionError {
	e.Cause = err
	if e.Wrapped == nil {
		e.Wrapped = make([]error, 0)
	}
	e.Wrapped = append(e.Wrapped, err)
	return e
}

// NewExecutionError creates a new execution error
func NewExecutionError(code, message string) *ExecutionError {
	return &ExecutionError{
		Code:      code,
		Message:   message,
		Timestamp: time.Now(),
		Severity:  SeverityError,
		Type:      ErrorTypeSystem,
		Context:   make(map[string]interface{}),
	}
}

// NewRuntimeError creates a new runtime error
func NewRuntimeError(language, code, message string) *ExecutionError {
	return &ExecutionError{
		Code:      code,
		Message:   message,
		Language:  language,
		Timestamp: time.Now(),
		Severity:  SeverityError,
		Type:      ErrorTypeRuntime,
		Context:   make(map[string]interface{}),
	}
}

// NewValidationError creates a new validation error
func NewValidationError(code, message string) *ExecutionError {
	return &ExecutionError{
		Code:      code,
		Message:   message,
		Timestamp: time.Now(),
		Severity:  SeverityWarning,
		Type:      ErrorTypeValidation,
		Context:   make(map[string]interface{}),
	}
}

// NewSystemError creates a new system error
func NewSystemError(code, message string) *ExecutionError {
	return &ExecutionError{
		Code:      code,
		Message:   message,
		Timestamp: time.Now(),
		Severity:  SeverityError,
		Type:      ErrorTypeSystem,
		Context:   make(map[string]interface{}),
	}
}

// NewUserError creates a new user error
func NewUserError(code, message string) *ExecutionError {
	return &ExecutionError{
		Code:      code,
		Message:   message,
		Timestamp: time.Now(),
		Severity:  SeverityInfo,
		Type:      ErrorTypeUser,
		Context:   make(map[string]interface{}),
	}
}

// NewUserErrorWithPosition creates a new user error with line and column information
func NewUserErrorWithPosition(code, message string, line, col int) *ExecutionError {
	return &ExecutionError{
		Code:      code,
		Message:   message,
		Line:      line,
		Col:       col,
		Timestamp: time.Now(),
		Severity:  SeverityInfo,
		Type:      ErrorTypeUser,
		Context:   make(map[string]interface{}),
	}
}

// NewRuntimeErrorWithPosition creates a new runtime error with line and column information
func NewRuntimeErrorWithPosition(language, code, message string, line, col int) *ExecutionError {
	return &ExecutionError{
		Code:      code,
		Message:   message,
		Language:  language,
		Line:      line,
		Col:       col,
		Timestamp: time.Now(),
		Severity:  SeverityError,
		Type:      ErrorTypeRuntime,
		Context:   make(map[string]interface{}),
	}
}

// NewSystemErrorWithPosition creates a new system error with line and column information
func NewSystemErrorWithPosition(code, message string, line, col int) *ExecutionError {
	return &ExecutionError{
		Code:      code,
		Message:   message,
		Line:      line,
		Col:       col,
		Timestamp: time.Now(),
		Severity:  SeverityError,
		Type:      ErrorTypeSystem,
		Context:   make(map[string]interface{}),
	}
}

// NewNetworkError creates a new network error
func NewNetworkError(code, message string) *ExecutionError {
	return &ExecutionError{
		Code:      code,
		Message:   message,
		Timestamp: time.Now(),
		Severity:  SeverityError,
		Type:      ErrorTypeNetwork,
		Context:   make(map[string]interface{}),
	}
}

// NewErrorWithPos creates a new error with AST position information
func NewErrorWithPos(code, message string, positions ...interface{}) *ExecutionError {
	execErr := &ExecutionError{
		Code:      code,
		Message:   message,
		Timestamp: time.Now(),
		Severity:  SeverityError,
		Type:      ErrorTypeSystem,
		Context:   make(map[string]interface{}),
	}

	// Process all positions and deduplicate them
	var allPositions []ast.Position

	for _, pos := range positions {
		if pos == nil {
			continue
		}

		var extractedPos ast.Position
		var valid bool

		// Handle different position types
		switch p := pos.(type) {
		case ast.Position:
			if p.Line > 0 {
				extractedPos = p
				valid = true
			}
		case interface{ GetLine() int }:
			if line := p.GetLine(); line > 0 {
				extractedPos.Line = line
				if colGetter, ok := pos.(interface{ GetColumn() int }); ok {
					extractedPos.Column = colGetter.GetColumn()
				}
				valid = true
			}
		case struct {
			Line   int
			Column int
			Offset int
		}:
			if p.Line > 0 {
				extractedPos.Line = p.Line
				extractedPos.Column = p.Column
				extractedPos.Offset = p.Offset
				valid = true
			}
		case map[string]interface{}:
			if line, ok := p["line"].(int); ok && line > 0 {
				extractedPos.Line = line
				if col, ok := p["column"].(int); ok {
					extractedPos.Column = col
				}
				if offset, ok := p["offset"].(int); ok {
					extractedPos.Offset = offset
				}
				valid = true
			}
		default:
			// Try to handle as a struct with Line, Column, Offset fields (like ast.Position)
			if posVal := reflect.ValueOf(pos); posVal.Kind() == reflect.Struct {
				if lineField := posVal.FieldByName("Line"); lineField.IsValid() && lineField.Kind() == reflect.Int {
					if line := int(lineField.Int()); line > 0 {
						extractedPos.Line = line
						if colField := posVal.FieldByName("Column"); colField.IsValid() && colField.Kind() == reflect.Int {
							extractedPos.Column = int(colField.Int())
						}
						if offsetField := posVal.FieldByName("Offset"); offsetField.IsValid() && offsetField.Kind() == reflect.Int {
							extractedPos.Offset = int(offsetField.Int())
						}
						valid = true
					}
				}
			}
			// Try to handle as a struct with Position() method
			// This is a fallback for unknown position types
			if posWithMethod, ok := pos.(interface {
				Position() interface {
					GetLine() int
					GetColumn() int
				}
			}); ok {
				position := posWithMethod.Position()
				if line := position.GetLine(); line > 0 {
					extractedPos.Line = line
					extractedPos.Column = position.GetColumn()
					valid = true
				}
			} else if posWithMethod, ok := pos.(interface {
				Position() struct {
					Line   int
					Column int
					Offset int
				}
			}); ok {
				position := posWithMethod.Position()
				if position.Line > 0 {
					extractedPos.Line = position.Line
					extractedPos.Column = position.Column
					extractedPos.Offset = position.Offset
					valid = true
				}
			}
		}

		if valid {
			allPositions = append(allPositions, extractedPos)
		}
	}

	// Deduplicate positions
	if len(allPositions) > 0 {
		uniquePositions := deduplicatePositions(allPositions)
		execErr.Positions = uniquePositions

		// Set the primary Line/Col for backward compatibility
		if len(uniquePositions) > 0 {
			execErr.Line = uniquePositions[0].Line
			execErr.Col = uniquePositions[0].Column
		}
	}

	return execErr
}

// deduplicatePositions removes duplicate positions from a slice
func deduplicatePositions(positions []ast.Position) []ast.Position {
	if len(positions) <= 1 {
		return positions
	}

	// Sort positions by Line, then Column
	sort.Slice(positions, func(i, j int) bool {
		if positions[i].Line != positions[j].Line {
			return positions[i].Line < positions[j].Line
		}
		return positions[i].Column < positions[j].Column
	})

	// Remove duplicates
	unique := make([]ast.Position, 0, len(positions))
	seen := make(map[string]bool)

	for _, pos := range positions {
		key := fmt.Sprintf("%d:%d", pos.Line, pos.Column)
		if !seen[key] {
			seen[key] = true
			unique = append(unique, pos)
		}
	}

	return unique
}

// NewUserErrorWithASTPos creates a new user error with AST position
func NewUserErrorWithASTPos(code, message string, pos interface{}) *ExecutionError {
	execErr := NewErrorWithPos(code, message, pos)
	execErr.Type = ErrorTypeUser
	execErr.Severity = SeverityInfo
	return execErr
}

// NewRuntimeErrorWithASTPos creates a new runtime error with AST position
func NewRuntimeErrorWithASTPos(language, code, message string, pos interface{}) *ExecutionError {
	execErr := NewErrorWithPos(code, message, pos)
	execErr.Language = language
	execErr.Type = ErrorTypeRuntime
	execErr.Severity = SeverityError
	return execErr
}

// NewValidationErrorWithASTPos creates a new validation error with AST position
func NewValidationErrorWithASTPos(code, message string, pos interface{}) *ExecutionError {
	execErr := NewErrorWithPos(code, message, pos)
	execErr.Type = ErrorTypeValidation
	execErr.Severity = SeverityWarning
	return execErr
}

// WrapError wraps an existing error into an ExecutionError
func WrapError(err error, code, message string) *ExecutionError {
	execErr := NewExecutionError(code, message)
	_ = execErr.Wrap(err)
	return execErr
}

// IsExecutionError checks if an error is an ExecutionError
func IsExecutionError(err error) bool {
	_, ok := err.(*ExecutionError)
	return ok
}

// AsExecutionError converts an error to ExecutionError if possible
func AsExecutionError(err error) (*ExecutionError, bool) {
	if execErr, ok := err.(*ExecutionError); ok {
		return execErr, true
	}
	return nil, false
}

// GetErrorChain returns the chain of errors
func GetErrorChain(err error) []error {
	var chain []error
	for err != nil {
		chain = append(chain, err)
		if execErr, ok := err.(*ExecutionError); ok {
			if len(execErr.Wrapped) > 0 {
				for _, wrapped := range execErr.Wrapped {
					chain = append(chain, wrapped)
				}
			}
		}
		err = unwrapError(err)
	}
	return chain
}

// unwrapError unwraps an error to get the underlying cause
func unwrapError(err error) error {
	if err == nil {
		return nil
	}
	if wrapper, ok := err.(interface{ Unwrap() error }); ok {
		return wrapper.Unwrap()
	}
	return nil
}

// Debug integration functions - these will be linked to config package
// to avoid circular imports

// IsGlobalDebugEnabled returns whether global debug mode is enabled
// This function should be overridden by the config package
var IsGlobalDebugEnabled = func() bool {
	return false
}

// ShouldShowStackTraces returns whether stack traces should be shown
// This function should be overridden by the config package
var ShouldShowStackTraces = func() bool {
	return false
}
