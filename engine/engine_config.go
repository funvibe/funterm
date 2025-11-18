package engine

import (
	"fmt"
	"sync"

	"funterm/container"
	"funterm/errors"
	"funterm/factory"
	"funterm/jobmanager"
	"funterm/runtime"
	"go-parser/pkg/parser"
	sharedparser "go-parser/pkg/shared"
)

// Sentinel errors for control flow
var (
	ErrBreak    = errors.NewSystemError("BREAK", "break statement")
	ErrContinue = errors.NewSystemError("CONTINUE", "continue statement")
)

// ExecutionEngine handles the execution of parsed language calls
type ExecutionEngine struct {
	parser          *parser.UnifiedParser
	runtimeManager  *runtime.RuntimeManager
	runtimeRegistry *factory.RuntimeRegistry
	container       container.Container
	jobManager      *jobmanager.JobManager // Job manager for background tasks
	// Общее хранилище переменных для всех языковых окружений
	sharedVariables map[string]map[string]interface{} // language -> variable -> value
	variablesMutex  sync.RWMutex                      // для потокобезопасности
	// Глобальные неквалифицированные переменные (доступны во всех runtimes)
	globalVariables  map[string]*sharedparser.VariableInfo // name -> VariableInfo
	globalMutex      sync.RWMutex                          // для потокобезопасности глобальных переменных
	verbose          bool                                  // Enable verbose/debug output
	jobFinished      chan struct{}
	localScope       *sharedparser.Scope   // Local scope for variables
	scopeStack       []*sharedparser.Scope // Stack of nested scopes
	backgroundOutput string                // Output from completed background jobs
	// Кэш рантаймов для переиспользования
	runtimeCache      map[string]runtime.LanguageRuntime // language -> runtime instance
	runtimeCacheMutex sync.RWMutex                       // для потокобезопасности кэша
	// Кэш для отслеживания последней синхронизированной версии глобальных переменных
	lastSyncedGlobals map[string]interface{} // name -> value
	syncedGlobalMutex sync.RWMutex           // для потокобезопасности кэша синхронизации
}

// NewExecutionEngine creates a new execution engine with default dependencies
func NewExecutionEngine() (*ExecutionEngine, error) {
	return NewExecutionEngineWithContainer(nil)
}

// NewExecutionEngineWithContainer creates a new execution engine with a custom DI container
func NewExecutionEngineWithContainer(c container.Container) (*ExecutionEngine, error) {
	return NewExecutionEngineWithConfig(ExecutionEngineConfig{
		Container: c,
	})
}

// ExecutionEngineConfig contains configuration for the execution engine
type ExecutionEngineConfig struct {
	Container       container.Container
	RuntimeRegistry *factory.RuntimeRegistry
	JobManager      *jobmanager.JobManager // Optional: if nil, a default one will be created
	Verbose         bool                   // Enable verbose/debug output
}

// NewExecutionEngineWithConfig creates a new execution engine with configuration
func NewExecutionEngineWithConfig(config ExecutionEngineConfig) (*ExecutionEngine, error) {
	// If no container is provided, create a default one
	var diContainer container.Container
	if config.Container == nil {
		diContainer = container.NewDIContainer()
	} else {
		diContainer = config.Container
	}

	// Register default dependencies if not already registered
	if !diContainer.IsRegistered("parser") {
		err := diContainer.Register("parser", func() (interface{}, error) {
			return parser.NewUnifiedParserWithVerbose(config.Verbose), nil
		}, container.Transient)
		if err != nil {
			return nil, errors.NewSystemError("PARSER_REGISTRATION_FAILED", fmt.Sprintf("failed to register parser: %v", err))
		}
	}

	if !diContainer.IsRegistered("runtimeManager") {
		err := diContainer.Register("runtimeManager", func() (interface{}, error) {
			return runtime.NewRuntimeManager(), nil
		}, container.Singleton)
		if err != nil {
			return nil, errors.NewSystemError("RUNTIME_MANAGER_REGISTRATION_FAILED", fmt.Sprintf("failed to register runtime manager: %v", err))
		}
	}

	// Register runtime registry if not already registered
	if !diContainer.IsRegistered("runtimeRegistry") {
		var registry *factory.RuntimeRegistry
		if config.RuntimeRegistry != nil {
			registry = config.RuntimeRegistry
		} else {
			registry = factory.DefaultRuntimeRegistryWithConfig(factory.RuntimeRegistryConfig{})
		}

		err := diContainer.Register("runtimeRegistry", func() (interface{}, error) {
			return registry, nil
		}, container.Singleton)
		if err != nil {
			return nil, errors.NewSystemError("RUNTIME_REGISTRY_REGISTRATION_FAILED", fmt.Sprintf("failed to register runtime registry: %v", err))
		}
	}

	// Resolve dependencies
	parserInstance, err := diContainer.Resolve("parser")
	if err != nil {
		return nil, errors.NewSystemError("PARSER_RESOLUTION_FAILED", fmt.Sprintf("failed to resolve parser: %v", err))
	}

	runtimeManagerInstance, err := diContainer.Resolve("runtimeManager")
	if err != nil {
		return nil, errors.NewSystemError("RUNTIME_MANAGER_RESOLUTION_FAILED", fmt.Sprintf("failed to resolve runtime manager: %v", err))
	}

	runtimeRegistryInstance, err := diContainer.Resolve("runtimeRegistry")
	if err != nil {
		return nil, errors.NewSystemError("RUNTIME_REGISTRY_RESOLUTION_FAILED", fmt.Sprintf("failed to resolve runtime registry: %v", err))
	}

	p, ok := parserInstance.(*parser.UnifiedParser)
	if !ok {
		return nil, errors.NewSystemError("INVALID_PARSER_TYPE", "resolved parser is not of correct type")
	}

	rm, ok := runtimeManagerInstance.(*runtime.RuntimeManager)
	if !ok {
		return nil, errors.NewSystemError("INVALID_RUNTIME_MANAGER_TYPE", "resolved runtime manager is not of correct type")
	}

	rr, ok := runtimeRegistryInstance.(*factory.RuntimeRegistry)
	if !ok {
		return nil, errors.NewSystemError("INVALID_RUNTIME_REGISTRY_TYPE", "resolved runtime registry is not of correct type")
	}

	// Create or use provided JobManager
	var jm *jobmanager.JobManager
	if config.JobManager != nil {
		jm = config.JobManager
	} else {
		// Create a default JobManager with reasonable concurrency limit
		jm = jobmanager.NewJobManager(5)
	}

	// Create a single root scope
	rootScope := sharedparser.NewScope(nil)

	engine := &ExecutionEngine{
		parser:            p,
		runtimeManager:    rm,
		runtimeRegistry:   rr,
		container:         diContainer,
		jobManager:        jm,
		sharedVariables:   make(map[string]map[string]interface{}),
		globalVariables:   make(map[string]*sharedparser.VariableInfo), // Initialize global variables
		verbose:           config.Verbose,
		jobFinished:       make(chan struct{}),
		localScope:        rootScope,                                // Use the same root scope
		scopeStack:        []*sharedparser.Scope{rootScope},         // Initialize scope stack with the same root scope
		runtimeCache:      make(map[string]runtime.LanguageRuntime), // Initialize runtime cache
		lastSyncedGlobals: make(map[string]interface{}),             // Initialize sync cache
	}

	return engine, nil
}

// GetOrCreateRuntime получает рантайм из кэша или создает новый, если его нет
func (e *ExecutionEngine) GetOrCreateRuntime(language string) (runtime.LanguageRuntime, error) {
	// Сначала проверяем кэш
	e.runtimeCacheMutex.RLock()
	if cachedRuntime, exists := e.runtimeCache[language]; exists {
		e.runtimeCacheMutex.RUnlock()
		if e.verbose {
			fmt.Printf("DEBUG: Using cached runtime for language '%s'\n", language)
		}
		return cachedRuntime, nil
	}
	e.runtimeCacheMutex.RUnlock()

	// Если в кэше нет, создаем новый
	e.runtimeCacheMutex.Lock()
	defer e.runtimeCacheMutex.Unlock()

	// Двойная проверка после получения блокировки записи
	if cachedRuntime, exists := e.runtimeCache[language]; exists {
		if e.verbose {
			fmt.Printf("DEBUG: Using cached runtime for language '%s' (double-check)\n", language)
		}
		return cachedRuntime, nil
	}

	if e.verbose {
		fmt.Printf("DEBUG: Creating new runtime for language '%s'\n", language)
	}

	// Создаем новый рантайм
	newRuntime, err := e.runtimeRegistry.CreateRuntimeForLanguage(language)
	if err != nil {
		return nil, err
	}

	// Инициализируем рантайм
	if err := newRuntime.Initialize(); err != nil {
		return nil, fmt.Errorf("failed to initialize runtime for language '%s': %w", language, err)
	}

	// Сохраняем в кэш
	e.runtimeCache[language] = newRuntime

	if e.verbose {
		fmt.Printf("DEBUG: Successfully created and cached runtime for language '%s'\n", language)
	}

	return newRuntime, nil
}
