package errors

import (
	"fmt"
	"runtime"
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
	if e.Language != "" {
		return fmt.Sprintf("[%s][%s][%s] %s", e.Language, e.Type, e.Code, e.Message)
	}
	return fmt.Sprintf("[%s][%s] %s", e.Type, e.Code, e.Message)
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
