package runtime

import (
	"fmt"

	"funterm/errors"
)

// LanguageRuntime defines the interface for all language runtimes
type LanguageRuntime interface {
	// Initialize sets up the language runtime
	Initialize() error

	// ExecuteFunction calls a function in the language runtime
	ExecuteFunction(name string, args []interface{}) (interface{}, error)

	// ExecuteFunctionMultiple calls a function in the language runtime and returns multiple values
	ExecuteFunctionMultiple(functionName string, args ...interface{}) ([]interface{}, error)

	// Eval выполняет произвольный код в языковом окружении
	Eval(code string) (interface{}, error)

	// ExecuteBatch выполняет код в пакетном режиме, отображая весь вывод
	ExecuteBatch(code string) error

	// ExecuteCodeBlockWithVariables выполняет код с сохранением указанных переменных
	ExecuteCodeBlockWithVariables(code string, variables []string) (interface{}, error)

	// SetVariable sets a variable in the language runtime
	SetVariable(name string, value interface{}) error

	// GetVariable retrieves a variable from the language runtime
	GetVariable(name string) (interface{}, error)

	// Isolate creates an isolated state for the runtime
	Isolate() error

	// Cleanup releases resources used by the runtime
	Cleanup() error

	// GetSupportedTypes returns the types supported by this runtime
	GetSupportedTypes() []string

	// GetName returns the name of the language runtime
	GetName() string

	// IsReady checks if the runtime is ready for execution
	IsReady() bool

	// Completion interface methods

	// GetModules returns available modules for the runtime
	GetModules() []string

	// GetModuleFunctions returns available functions for a specific module
	GetModuleFunctions(module string) []string

	// GetFunctionSignature returns the signature of a function in a module
	GetFunctionSignature(module, function string) (string, error)

	// GetGlobalVariables returns available global variables
	GetGlobalVariables() []string

	// GetCompletionSuggestions returns completion suggestions for a given input
	GetCompletionSuggestions(input string) []string

	// Dynamic introspection methods for runtime completion

	// GetUserDefinedFunctions returns functions defined by the user during the session
	GetUserDefinedFunctions() []string

	// GetImportedModules returns modules that have been imported during the session
	GetImportedModules() []string

	// GetDynamicCompletions returns completions based on current runtime state
	GetDynamicCompletions(input string) ([]string, error)

	// GetObjectProperties returns properties and methods of a runtime object
	GetObjectProperties(objectName string) ([]string, error)

	// GetFunctionParameters returns parameter names and types for a function
	GetFunctionParameters(functionName string) ([]FunctionParameter, error)

	// UpdateCompletionContext updates the completion context after code execution
	UpdateCompletionContext(executedCode string, result interface{}) error

	// RefreshRuntimeState refreshes the runtime state for completion
	RefreshRuntimeState() error

	// GetRuntimeObjects returns all objects currently available in the runtime
	GetRuntimeObjects() map[string]interface{}

	// State management methods

	// SetStateManager sets the state manager for this runtime
	// State management removed for simplified beta version

	// All state management methods removed for simplified beta version
}

// FunctionParameter represents a function parameter with name and type
type FunctionParameter struct {
	Name string
	Type string
}

// RuntimeError represents an error from a language runtime
type RuntimeError struct {
	Language string
	Message  string
	Code     string
	Context  map[string]interface{}
}

// Error returns the error message
func (e *RuntimeError) Error() string {
	return fmt.Sprintf("[%s][RUNTIME] %s", e.Language, e.Message)
}

// NewRuntimeError creates a new runtime error
func NewRuntimeError(language, message string) *RuntimeError {
	return &RuntimeError{
		Language: language,
		Message:  message,
		Context:  make(map[string]interface{}),
	}
}

// WithContext adds context to the error
func (e *RuntimeError) WithContext(key string, value interface{}) *RuntimeError {
	e.Context[key] = value
	return e
}

// ToExecutionError converts the runtime error to an execution error
func (e *RuntimeError) ToExecutionError() *errors.ExecutionError {
	execErr := errors.NewExecutionError("RUNTIME_ERROR", e.Message)
	_ = execErr.WithLanguage(e.Language)
	for k, v := range e.Context {
		_ = execErr.WithContext(k, v)
	}
	return execErr
}

// RuntimeManager manages multiple language runtimes
type RuntimeManager struct {
	runtimes map[string]LanguageRuntime
	aliases  map[string]string // Алиасы языков (alias -> language)
}

// NewRuntimeManager creates a new runtime manager
func NewRuntimeManager() *RuntimeManager {
	rm := &RuntimeManager{
		runtimes: make(map[string]LanguageRuntime),
		aliases:  make(map[string]string),
	}

	// Регистрируем стандартные алиасы
	rm.RegisterAlias("py", "python")

	return rm
}

// RegisterAlias регистрирует алиас для языка
func (rm *RuntimeManager) RegisterAlias(alias, language string) {
	rm.aliases[alias] = language
}

// GetAlias возвращает полное имя языка по алиасу
func (rm *RuntimeManager) GetAlias(alias string) (string, bool) {
	language, exists := rm.aliases[alias]
	return language, exists
}

// RegisterRuntime registers a language runtime
func (rm *RuntimeManager) RegisterRuntime(runtime LanguageRuntime) error {
	name := runtime.GetName()
	if _, exists := rm.runtimes[name]; exists {
		return fmt.Errorf("runtime '%s' is already registered", name)
	}
	rm.runtimes[name] = runtime
	return nil
}

// GetRuntime returns a language runtime by name
func (rm *RuntimeManager) GetRuntime(name string) (LanguageRuntime, error) {
	// Проверяем, не является ли имя алиасом
	if language, isAlias := rm.GetAlias(name); isAlias {
		name = language
	}

	runtime, exists := rm.runtimes[name]
	if !exists {
		return nil, fmt.Errorf("runtime '%s' is not registered", name)
	}
	return runtime, nil
}

// InitializeAll initializes all registered runtimes
func (rm *RuntimeManager) InitializeAll() error {
	for name, runtime := range rm.runtimes {
		if err := runtime.Initialize(); err != nil {
			return fmt.Errorf("failed to initialize runtime '%s': %v", name, err)
		}
	}
	return nil
}

// CleanupAll cleans up all registered runtimes
func (rm *RuntimeManager) CleanupAll() error {
	var lastErr error
	for name, runtime := range rm.runtimes {
		if err := runtime.Cleanup(); err != nil {
			lastErr = fmt.Errorf("failed to cleanup runtime '%s': %v", name, err)
		}
	}
	return lastErr
}

// ListRuntimes returns the names of all registered runtimes
func (rm *RuntimeManager) ListRuntimes() []string {
	names := make([]string, 0, len(rm.runtimes))
	for name := range rm.runtimes {
		names = append(names, name)
	}
	return names
}

// GetAllRuntimes returns all registered runtime instances
func (rm *RuntimeManager) GetAllRuntimes() []LanguageRuntime {
	runtimes := make([]LanguageRuntime, 0, len(rm.runtimes))
	for _, runtime := range rm.runtimes {
		runtimes = append(runtimes, runtime)
	}
	return runtimes
}

// IsRuntimeReady checks if a runtime is ready
func (rm *RuntimeManager) IsRuntimeReady(name string) bool {
	// Проверяем, не является ли имя алиасом
	if language, isAlias := rm.GetAlias(name); isAlias {
		name = language
	}

	runtime, exists := rm.runtimes[name]
	if !exists {
		return false
	}
	return runtime.IsReady()
}
