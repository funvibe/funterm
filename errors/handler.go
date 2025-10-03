package errors

import (
	"context"
	"fmt"
)

// ErrorOption is a function that modifies an ExecutionError
type ErrorOption func(*ExecutionError)

// WithLanguageOption sets the language for the error
func WithLanguageOption(language string) ErrorOption {
	return func(e *ExecutionError) {
		e.Language = language
	}
}

// WithSeverityOption sets the severity level for the error
func WithSeverityOption(severity ErrorSeverity) ErrorOption {
	return func(e *ExecutionError) {
		e.Severity = severity
	}
}

// WithTypeOption sets the error type
func WithTypeOption(errorType ErrorType) ErrorOption {
	return func(e *ExecutionError) {
		e.Type = errorType
	}
}

// WithContextOption adds context information to the error
func WithContextOption(key string, value interface{}) ErrorOption {
	return func(e *ExecutionError) {
		if e.Context == nil {
			e.Context = make(map[string]interface{})
		}
		e.Context[key] = value
	}
}

// RecoveryAction represents the type of recovery action
type RecoveryAction string

const (
	RecoveryActionNone     RecoveryAction = "NONE"
	RecoveryActionRetry    RecoveryAction = "RETRY"
	RecoveryActionFallback RecoveryAction = "FALLBACK"
	RecoveryActionAbort    RecoveryAction = "ABORT"
	RecoveryActionLog      RecoveryAction = "LOG"
)

// RecoveryStrategy represents a strategy for recovering from an error
type RecoveryStrategy struct {
	Action      RecoveryAction `json:"action"`
	Message     string         `json:"message"`
	RetryCount  int            `json:"retry_count"`
	RetryDelay  string         `json:"retry_delay"`
	Fallback    interface{}    `json:"fallback,omitempty"`
	ShouldRetry bool           `json:"should_retry"`
}

// ErrorHandler defines the interface for handling errors
type ErrorHandler interface {
	// Handle processes an error and returns a potentially modified error
	Handle(ctx context.Context, err error) error

	// Recover attempts to recover from an error and returns a recovery strategy
	Recover(ctx context.Context, err error) (RecoveryStrategy, error)

	// Wrap wraps an error with additional context
	Wrap(ctx context.Context, err error, code, message string, options ...ErrorOption) *ExecutionError
}

// DefaultErrorHandler is the default implementation of ErrorHandler
type DefaultErrorHandler struct {
	recoveryPolicies map[ErrorType]RecoveryPolicy
}

// RecoveryPolicy defines how to handle errors of a specific type
type RecoveryPolicy struct {
	MaxRetries    int
	RetryDelay    string
	DefaultAction RecoveryAction
}

// NewDefaultErrorHandler creates a new default error handler
func NewDefaultErrorHandler() *DefaultErrorHandler {
	return &DefaultErrorHandler{
		recoveryPolicies: map[ErrorType]RecoveryPolicy{
			ErrorTypeRuntime: {
				MaxRetries:    3,
				RetryDelay:    "1s",
				DefaultAction: RecoveryActionRetry,
			},
			ErrorTypeValidation: {
				MaxRetries:    0,
				RetryDelay:    "0s",
				DefaultAction: RecoveryActionLog,
			},
			ErrorTypeSystem: {
				MaxRetries:    1,
				RetryDelay:    "5s",
				DefaultAction: RecoveryActionRetry,
			},
			ErrorTypeNetwork: {
				MaxRetries:    5,
				RetryDelay:    "3s",
				DefaultAction: RecoveryActionRetry,
			},
			ErrorTypeUser: {
				MaxRetries:    0,
				RetryDelay:    "0s",
				DefaultAction: RecoveryActionLog,
			},
		},
	}
}

// Handle processes an error and returns a potentially modified error
func (h *DefaultErrorHandler) Handle(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}

	// If it's already an ExecutionError, return it as-is
	if execErr, ok := err.(*ExecutionError); ok {
		return execErr
	}

	// Convert standard error to ExecutionError
	execErr := NewExecutionError("UNKNOWN_ERROR", err.Error())
	_ = execErr.WithStackTrace()

	return execErr
}

// Recover attempts to recover from an error and returns a recovery strategy
func (h *DefaultErrorHandler) Recover(ctx context.Context, err error) (RecoveryStrategy, error) {
	execErr, ok := err.(*ExecutionError)
	if !ok {
		// Convert to ExecutionError if it's not already
		execErr = NewExecutionError("UNKNOWN_ERROR", err.Error())
	}

	policy, exists := h.recoveryPolicies[execErr.Type]
	if !exists {
		policy = RecoveryPolicy{
			MaxRetries:    0,
			RetryDelay:    "0s",
			DefaultAction: RecoveryActionLog,
		}
	}

	strategy := RecoveryStrategy{
		Action:      policy.DefaultAction,
		Message:     fmt.Sprintf("Recovery strategy for %s error: %s", execErr.Type, execErr.Message),
		RetryCount:  policy.MaxRetries,
		RetryDelay:  policy.RetryDelay,
		ShouldRetry: policy.MaxRetries > 0,
	}

	return strategy, nil
}

// Wrap wraps an error with additional context
func (h *DefaultErrorHandler) Wrap(ctx context.Context, err error, code, message string, options ...ErrorOption) *ExecutionError {
	if err == nil {
		return nil
	}

	execErr := NewExecutionError(code, message)
	_ = execErr.Wrap(err)

	// Apply options
	for _, option := range options {
		option(execErr)
	}

	// Add context from the context.Context if available
	if ctx != nil {
		if requestID := ctx.Value("request_id"); requestID != nil {
			_ = execErr.WithContext("request_id", requestID)
		}
		if userID := ctx.Value("user_id"); userID != nil {
			_ = execErr.WithContext("user_id", userID)
		}
		if traceID := ctx.Value("trace_id"); traceID != nil {
			_ = execErr.WithContext("trace_id", traceID)
		}
	}

	return execErr
}

// ErrorHandlerRegistry manages multiple error handlers
type ErrorHandlerRegistry struct {
	handlers       map[string]ErrorHandler
	defaultHandler ErrorHandler
}

// NewErrorHandlerRegistry creates a new error handler registry
func NewErrorHandlerRegistry() *ErrorHandlerRegistry {
	return &ErrorHandlerRegistry{
		handlers:       make(map[string]ErrorHandler),
		defaultHandler: NewDefaultErrorHandler(),
	}
}

// RegisterHandler registers an error handler for a specific component
func (r *ErrorHandlerRegistry) RegisterHandler(component string, handler ErrorHandler) {
	r.handlers[component] = handler
}

// GetHandler returns the error handler for a specific component
func (r *ErrorHandlerRegistry) GetHandler(component string) ErrorHandler {
	if handler, exists := r.handlers[component]; exists {
		return handler
	}
	return r.defaultHandler
}

// HandleError handles an error using the appropriate handler
func (r *ErrorHandlerRegistry) HandleError(ctx context.Context, component string, err error) error {
	handler := r.GetHandler(component)
	return handler.Handle(ctx, err)
}

// RecoverError attempts to recover from an error using the appropriate handler
func (r *ErrorHandlerRegistry) RecoverError(ctx context.Context, component string, err error) (RecoveryStrategy, error) {
	handler := r.GetHandler(component)
	return handler.Recover(ctx, err)
}

// WrapError wraps an error using the appropriate handler
func (r *ErrorHandlerRegistry) WrapError(ctx context.Context, component string, err error, code, message string, options ...ErrorOption) *ExecutionError {
	handler := r.GetHandler(component)
	return handler.Wrap(ctx, err, code, message, options...)
}
