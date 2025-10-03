package factory

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"funterm/errors"
	"funterm/runtime"
	go_runtime "funterm/runtime/go"
	"funterm/runtime/lua"
	"funterm/runtime/node"
	"funterm/runtime/python"
)

// RuntimeFactory defines the interface for creating language runtimes
type RuntimeFactory interface {
	// CreateRuntime creates a new language runtime instance
	CreateRuntime() (runtime.LanguageRuntime, error)

	// GetSupportedLanguages returns the list of languages supported by this factory
	GetSupportedLanguages() []string

	// ValidateEnvironment checks if the environment is suitable for this runtime
	ValidateEnvironment() error

	// GetName returns the name of the runtime factory
	GetName() string
}

// RuntimeRegistry manages multiple runtime factories
type RuntimeRegistry struct {
	factories map[string]RuntimeFactory
	mutex     sync.RWMutex
}

// RuntimeRegistryConfig contains configuration for the runtime registry
type RuntimeRegistryConfig struct {
}

// NewRuntimeRegistry creates a new runtime registry
func NewRuntimeRegistry() *RuntimeRegistry {
	return NewRuntimeRegistryWithConfig(RuntimeRegistryConfig{})
}

// NewRuntimeRegistryWithConfig creates a new runtime registry with configuration
func NewRuntimeRegistryWithConfig(config RuntimeRegistryConfig) *RuntimeRegistry {
	return &RuntimeRegistry{
		factories: make(map[string]RuntimeFactory),
	}
}

// RegisterFactory registers a runtime factory
func (rr *RuntimeRegistry) RegisterFactory(factory RuntimeFactory) error {
	if factory == nil {
		return errors.NewValidationError("NIL_FACTORY", "factory cannot be nil")
	}

	name := factory.GetName()
	if name == "" {
		return errors.NewValidationError("EMPTY_FACTORY_NAME", "factory name cannot be empty")
	}

	rr.mutex.Lock()
	defer rr.mutex.Unlock()

	if _, exists := rr.factories[name]; exists {
		return errors.NewSystemError("FACTORY_ALREADY_REGISTERED", fmt.Sprintf("factory '%s' is already registered", name))
	}

	rr.factories[name] = factory
	return nil
}

// GetFactory returns a registered factory by name
func (rr *RuntimeRegistry) GetFactory(name string) (RuntimeFactory, error) {
	rr.mutex.RLock()
	defer rr.mutex.RUnlock()

	factory, exists := rr.factories[name]
	if !exists {
		return nil, errors.NewSystemError("FACTORY_NOT_REGISTERED", fmt.Sprintf("factory '%s' is not registered", name))
	}

	return factory, nil
}

// CreateRuntime creates a runtime using the specified factory
func (rr *RuntimeRegistry) CreateRuntime(factoryName string) (runtime.LanguageRuntime, error) {
	factory, err := rr.GetFactory(factoryName)
	if err != nil {
		return nil, errors.NewSystemError("FACTORY_RETRIEVAL_FAILED", fmt.Sprintf("failed to get factory '%s': %v", factoryName, err))
	}

	return factory.CreateRuntime()
}

// LuaRuntimeFactory creates Lua runtime instances
type LuaRuntimeFactory struct{}

// NewLuaRuntimeFactory creates a new Lua runtime factory
func NewLuaRuntimeFactory() *LuaRuntimeFactory {
	return &LuaRuntimeFactory{}
}

// CreateRuntime creates a new Lua runtime instance
func (lf *LuaRuntimeFactory) CreateRuntime() (runtime.LanguageRuntime, error) {
	return lua.NewLuaRuntime(), nil
}

// GetSupportedLanguages returns the languages supported by this factory
func (lf *LuaRuntimeFactory) GetSupportedLanguages() []string {
	return []string{"lua"}
}

// ValidateEnvironment checks if Lua environment is available
func (lf *LuaRuntimeFactory) ValidateEnvironment() error {
	// For Lua, we assume it's available since we're using gopher-lua
	// In a real implementation, we might check for specific Lua versions or libraries
	return nil
}

// ListFactories returns the names of all registered factories
func (rr *RuntimeRegistry) ListFactories() []string {
	rr.mutex.RLock()
	defer rr.mutex.RUnlock()

	names := make([]string, 0, len(rr.factories))
	for name := range rr.factories {
		names = append(names, name)
	}

	return names
}

// GetSupportedLanguages returns all supported languages from all factories
func (rr *RuntimeRegistry) GetSupportedLanguages() []string {
	rr.mutex.RLock()
	defer rr.mutex.RUnlock()

	languages := make(map[string]bool)
	for _, factory := range rr.factories {
		for _, lang := range factory.GetSupportedLanguages() {
			languages[lang] = true
		}
	}

	result := make([]string, 0, len(languages))
	for lang := range languages {
		result = append(result, lang)
	}

	return result
}

// ValidateAllEnvironments validates environments for all registered factories
func (rr *RuntimeRegistry) ValidateAllEnvironments() map[string]error {
	rr.mutex.RLock()
	defer rr.mutex.RUnlock()

	errors := make(map[string]error)
	for name, factory := range rr.factories {
		if err := factory.ValidateEnvironment(); err != nil {
			errors[name] = err
		}
	}

	return errors
}

// Clear removes all registered factories
func (rr *RuntimeRegistry) Clear() {
	rr.mutex.Lock()
	defer rr.mutex.Unlock()

	rr.factories = make(map[string]RuntimeFactory)
}

// GetName returns the name of the runtime factory
func (lf *LuaRuntimeFactory) GetName() string {
	return "lua"
}

// PythonRuntimeFactory creates Python runtime instances
type PythonRuntimeFactory struct {
	pythonPath       string
	verbose          bool
	executionTimeout time.Duration
}

// NewPythonRuntimeFactory creates a new Python runtime factory
func NewPythonRuntimeFactory() *PythonRuntimeFactory {
	return &PythonRuntimeFactory{
		pythonPath:       "python3",        // default
		verbose:          false,            // default
		executionTimeout: 30 * time.Second, // default
	}
}

// NewPythonRuntimeFactoryWithConfig creates a new Python runtime factory with configuration
func NewPythonRuntimeFactoryWithConfig(pythonPath string, verbose bool, executionTimeout time.Duration) *PythonRuntimeFactory {
	if pythonPath == "" {
		pythonPath = "python3" // fallback to default
	}
	if executionTimeout <= 0 {
		executionTimeout = 30 * time.Second // fallback to default
	}
	return &PythonRuntimeFactory{
		pythonPath:       pythonPath,
		verbose:          verbose,
		executionTimeout: executionTimeout,
	}
}

// CreateRuntime creates a new Python runtime instance
func (pf *PythonRuntimeFactory) CreateRuntime() (runtime.LanguageRuntime, error) {
	// Check if we're running in test mode
	if isTestMode() {
		return python.GetSharedTestRuntime(), nil
	}

	// Create new runtime and initialize with configuration
	runtime := python.NewPythonRuntime()
	if err := runtime.InitializeWithConfig(pf.pythonPath, pf.verbose); err != nil {
		return nil, err
	}

	// Set execution timeout
	runtime.SetExecutionTimeout(pf.executionTimeout)

	return runtime, nil
}

// isTestMode checks if we're running in test environment
func isTestMode() bool {
	// Check if we're running under 'go test'
	for _, arg := range os.Args {
		if strings.Contains(arg, "test") || strings.HasSuffix(arg, ".test") {
			return true
		}
	}
	// Check for test environment variable
	return os.Getenv("FUNTERM_TEST_MODE") == "true"
}

// GetSupportedLanguages returns the languages supported by this factory
func (pf *PythonRuntimeFactory) GetSupportedLanguages() []string {
	return []string{"python", "py"}
}

// ValidateEnvironment checks if Python environment is available
func (pf *PythonRuntimeFactory) ValidateEnvironment() error {
	// Check if Python is available
	cmd := exec.Command("python3", "--version")
	if err := cmd.Run(); err != nil {
		// Try python as fallback
		cmd = exec.Command("python", "--version")
		if err := cmd.Run(); err != nil {
			return errors.NewSystemError("PYTHON_NOT_FOUND", "neither python3 nor python found in PATH")
		}
	}
	return nil
}

// GetName returns the name of the runtime factory
func (pf *PythonRuntimeFactory) GetName() string {
	return "python"
}

// NodeRuntimeFactory creates Node.js runtime instances
type NodeRuntimeFactory struct{}

// NewNodeRuntimeFactory creates a new Node.js runtime factory
func NewNodeRuntimeFactory() *NodeRuntimeFactory {
	return &NodeRuntimeFactory{}
}

// CreateRuntime creates a new Node.js runtime instance
func (nf *NodeRuntimeFactory) CreateRuntime() (runtime.LanguageRuntime, error) {
	return node.NewNodeRuntime(), nil
}

// GetSupportedLanguages returns the languages supported by this factory
func (nf *NodeRuntimeFactory) GetSupportedLanguages() []string {
	return []string{"node", "js"} // Assuming js can alias to node for now
}

// ValidateEnvironment checks if Node.js environment is available
func (nf *NodeRuntimeFactory) ValidateEnvironment() error {
	// The actual check is now inside the runtime's Initialize method
	// to allow for "soft" disabling. So we just return nil here.
	return nil
}

// GetName returns the name of the runtime factory
func (nf *NodeRuntimeFactory) GetName() string {
	return "node"
}

// GoRuntimeFactory creates Go runtime instances
type GoRuntimeFactory struct{}

// NewGoRuntimeFactory creates a new Go runtime factory
func NewGoRuntimeFactory() *GoRuntimeFactory {
	return &GoRuntimeFactory{}
}

// CreateRuntime creates a new Go runtime instance
func (gf *GoRuntimeFactory) CreateRuntime() (runtime.LanguageRuntime, error) {
	return go_runtime.NewGoRuntime(), nil
}

// GetSupportedLanguages returns the languages supported by this factory
func (gf *GoRuntimeFactory) GetSupportedLanguages() []string {
	return []string{"go"}
}

// ValidateEnvironment checks if Go environment is available
func (gf *GoRuntimeFactory) ValidateEnvironment() error {
	// Go runtime doesn't require external dependencies since it uses Go stdlib
	return nil
}

// GetName returns the name of the runtime factory
func (gf *GoRuntimeFactory) GetName() string {
	return "go"
}

// DefaultRuntimeRegistry creates a runtime registry with default factories
func DefaultRuntimeRegistry() *RuntimeRegistry {
	return DefaultRuntimeRegistryWithConfig(RuntimeRegistryConfig{})
}

// DefaultRuntimeRegistryWithConfig creates a runtime registry with default factories and configuration
func DefaultRuntimeRegistryWithConfig(config RuntimeRegistryConfig) *RuntimeRegistry {
	registry := NewRuntimeRegistryWithConfig(config)

	// Register default factories
	luaFactory := NewLuaRuntimeFactory()
	pythonFactory := NewPythonRuntimeFactory()
	goFactory := NewGoRuntimeFactory()
	nodeFactory := NewNodeRuntimeFactory()

	if err := registry.RegisterFactory(luaFactory); err != nil {
		// Log error but continue with other factories
	}
	if err := registry.RegisterFactory(pythonFactory); err != nil {
		// Log error but continue with other factories
	}
	if err := registry.RegisterFactory(goFactory); err != nil {
		// Log error but continue with other factories
	}
	if err := registry.RegisterFactory(nodeFactory); err != nil {
		// Log error but continue with other factories
	}

	return registry
}

// GetFactoryForLanguage returns the appropriate factory for a given language
func (rr *RuntimeRegistry) GetFactoryForLanguage(language string) (RuntimeFactory, error) {
	rr.mutex.RLock()
	defer rr.mutex.RUnlock()

	for _, factory := range rr.factories {
		for _, lang := range factory.GetSupportedLanguages() {
			if strings.EqualFold(lang, language) {
				return factory, nil
			}
		}
	}

	return nil, errors.NewSystemError("FACTORY_NOT_FOUND", fmt.Sprintf("no factory found for language '%s'", language))
}

// CreateRuntimeForLanguage creates a runtime for the specified language
func (rr *RuntimeRegistry) CreateRuntimeForLanguage(language string) (runtime.LanguageRuntime, error) {
	factory, err := rr.GetFactoryForLanguage(language)
	if err != nil {
		return nil, errors.NewSystemError("FACTORY_RETRIEVAL_FAILED", fmt.Sprintf("failed to get factory for language '%s': %v", language, err))
	}

	return factory.CreateRuntime()
}
