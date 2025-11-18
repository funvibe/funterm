package repl

import (
	"bufio"
	"bytes"
	"fmt"
	"funterm/engine"
	"funterm/errors"
	"funterm/factory"
	"funterm/jobmanager"
	"funterm/shared"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/chzyer/readline"
)

// History settings are now configurable via REPLConfig

// REPL represents the Read-Eval-Print Loop
type REPL struct {
	engine               *engine.ExecutionEngine
	prompt               string
	continuePrompt       string // Continuation prompt for multiline
	running              bool
	history              []string
	historyFile          string // Path to history file
	historySize          int    // Maximum history size
	showWelcome          bool
	registry             *factory.RuntimeRegistry
	performanceOptimizer *PerformanceOptimizer
	optimizer            *PerformanceOptimizer // Alias for backward compatibility
	advancedCommands     *AdvancedCommands
	errorHandler         interface{}                     // Placeholder for error handler integration
	jobNotifications     chan jobmanager.JobNotification // Channel for job notifications
	verbose              bool                            // Enable verbose/debug output
	enableMultiline      bool                            // Enable multiline input
	enableColors         bool                            // Enable color output
	buffer               *MultiLineBuffer                // Buffer for multiline input
	displayManager       *DisplayManager                 // Display manager for formatting
}

// NewREPL creates a new REPL instance
func NewREPL() *REPL {
	return NewREPLWithRegistry(nil)
}

// NewREPLWithRegistry creates a new REPL instance with a custom runtime registry
func NewREPLWithRegistry(registry *factory.RuntimeRegistry) *REPL {
	return NewREPLWithConfig(REPLConfig{
		Registry: registry,
	})
}

// REPLConfig contains configuration for the REPL
type REPLConfig struct {
	Registry        *factory.RuntimeRegistry
	Verbose         bool
	EnableMultiline bool // Enable multiline input
	EnableColors    bool
	Prompt          string // Main prompt (default: "> ")
	ContinuePrompt  string // Continuation prompt for multiline (default: "... ")
	HistoryFile     string // History file path (default: "/tmp/funterm_history")
	HistorySize     int    // Maximum history size (default: 1000)
}

// NewREPLWithConfig creates a new REPL instance with configuration
func NewREPLWithConfig(config REPLConfig) *REPL {
	// Create execution engine
	eng, err := engine.NewExecutionEngineWithConfig(engine.ExecutionEngineConfig{
		RuntimeRegistry: config.Registry,
		Verbose:         config.Verbose,
	})
	if err != nil {
		panic(errors.NewSystemError("ENGINE_CREATION_FAILED", fmt.Sprintf("Failed to create execution engine: %v", err)).Error())
	}

	// If no registry is provided, create a default one
	registry := config.Registry
	if registry == nil {
		registry = factory.DefaultRuntimeRegistryWithConfig(factory.RuntimeRegistryConfig{})
	}

	// Set default prompts if not provided
	prompt := config.Prompt
	if prompt == "" {
		prompt = "> "
	}
	continuePrompt := config.ContinuePrompt
	if continuePrompt == "" {
		continuePrompt = "... "
	}

	// Set default history settings if not provided
	historyFile := config.HistoryFile
	if historyFile == "" {
		historyFile = "/tmp/funterm_history"
	}
	historySize := config.HistorySize
	if historySize == 0 {
		historySize = 1000
	}

	// Initialize performance optimizer
	perfOptimizer := NewPerformanceOptimizer(true)
	replInstance := &REPL{
		engine:               eng,
		prompt:               prompt,
		continuePrompt:       continuePrompt,
		running:              false,
		history:              make([]string, 0),
		historyFile:          historyFile,
		historySize:          historySize,
		showWelcome:          true,
		registry:             registry,
		performanceOptimizer: perfOptimizer,
		optimizer:            perfOptimizer,                              // Alias for backward compatibility
		advancedCommands:     nil,                                        // Will be set after creating the REPL instance
		errorHandler:         nil,                                        // Placeholder for error handler integration
		jobNotifications:     make(chan jobmanager.JobNotification, 100), // Buffered channel for job notifications
		verbose:              config.Verbose,
		enableMultiline:      config.EnableMultiline,
		enableColors:         config.EnableColors,
		buffer:               NewMultiLineBuffer(),
		displayManager:       NewDisplayManager(config.EnableColors, config.Verbose),
	}

	// Initialize advanced commands with the REPL instance
	replInstance.advancedCommands = NewAdvancedCommands(replInstance)

	// Register all advanced commands
	if replInstance.advancedCommands != nil {
		replInstance.advancedCommands.RegisterCommands()
	}

	return replInstance
}

// IsVerbose returns whether verbose mode is enabled
func (r *REPL) IsVerbose() bool {
	return r.verbose
}

// startJobNotificationListener starts a goroutine to listen for job notifications
func (r *REPL) startJobNotificationListener() {
	go func() {
		for notification := range r.engine.GetJobNotificationChannel() {
			// Forward the notification to our internal channel
			select {
			case r.jobNotifications <- notification:
				// Notification forwarded successfully
			default:
				// Channel is full, log a warning (in a real implementation, use proper logging)
				fmt.Printf("Warning: Job notification channel full, dropping notification for job %d\n", notification.JobID)
			}
		}
	}()
}

// isInteractive checks if the input is interactive (terminal) or piped
func (r *REPL) isInteractive() bool {
	fileInfo, _ := os.Stdin.Stat()
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

// Run starts the REPL loop
func (r *REPL) Run() error {
	r.running = true

	if r.showWelcome {
		r.printWelcome()
	}

	// Initialize runtimes (in future phases, this will register actual language runtimes)
	if err := r.initializeRuntimes(); err != nil {
		return errors.NewSystemError("RUNTIME_INITIALIZATION_FAILED", fmt.Sprintf("failed to initialize runtimes: %v", err))
	}

	// Start the job notification listener
	r.startJobNotificationListener()

	// Check if we're running in interactive mode or piped mode
	if r.isInteractive() {
		// Use interactive mode with readline for arrow key navigation
		return r.runInteractive()
	} else {
		return r.runPiped()
	}
}

// runInteractive runs the REPL in interactive mode with readline and multiline support
func (r *REPL) runInteractive() error {
	// Create runtime completer for autocompletion with fallback
	runtimeManager := r.engine.GetRuntimeManager()
	completer := NewFallbackCompleter(runtimeManager)

	// Create readline instance
	rl, err := readline.NewEx(&readline.Config{
		Prompt:          r.prompt,
		HistoryFile:     r.historyFile,
		HistoryLimit:    r.historySize,
		InterruptPrompt: "^C",
		EOFPrompt:       ":exit",
		AutoComplete:    completer,
	})
	if err != nil {
		return errors.NewSystemError("READLINE_INIT_FAILED", fmt.Sprintf("failed to initialize readline: %v", err))
	}
	defer func() {
		if err := rl.Close(); err != nil {
			// Log the error but continue
			fmt.Printf("Warning: Failed to close readline: %v\n", err)
		}
	}()

	fmt.Println("Multi-line mode is always enabled:")
	fmt.Println("  Enter      - execute the code")
	fmt.Println("  \\ at end   - add a line to the buffer (like Shift+Enter)")
	fmt.Println("  :help ml   - more detailed")
	fmt.Println()

	buffer := NewMultiLineBuffer()

	// Start the main REPL loop
	for r.running {
		// Show prompt based on buffer state
		if buffer.IsActive() {
			rl.SetPrompt(r.continuePrompt)
		} else {
			rl.SetPrompt(r.prompt)
		}

		input, err := rl.Readline()
		if err != nil {
			if err == readline.ErrInterrupt {
				if len(input) == 0 {
					// Empty input + Ctrl+C means exit
					fmt.Println("\nGoodbye!")
					break
				}
				// Ctrl+C with input, just clear the line and reset buffer
				buffer.Clear()
				continue
			}
			if err == io.EOF {
				fmt.Println("\nGoodbye!")
				break
			}
			return errors.NewSystemError("READ_ERROR", fmt.Sprintf("read error: %v", err))
		}

		// Remove \n at the end (readline includes it)
		line := strings.TrimSpace(input)

		// Handle special commands
		if strings.HasPrefix(line, ":") {
			if r.handleSpecialCommandsSimple(line, buffer) {
				continue
			}
		}

		// Handle terminal commands with $ prefix - execute immediately even in multiline mode
		if strings.HasPrefix(line, "$") {
			if strings.HasPrefix(line, "<$") {
				// <$ command - execute terminal command and pass output to funterm
				r.handleDollarCommand(line, buffer)
			} else {
				// $ command - execute terminal command and print output
				r.handleSingleDollarCommand(line)
			}
			continue
		}

		// Process input
		if isShiftEnter(line) {
			// Shift+Enter - add to buffer (remove trailing \)
			cleanLine := strings.TrimSuffix(line, "\r")     // Remove \r for Windows
			cleanLine = strings.TrimSuffix(cleanLine, "\\") // Remove trailing \ for continuation
			if strings.TrimSpace(cleanLine) != "" {
				buffer.AddLine(cleanLine)
			}
		} else {
			// Regular Enter
			if buffer.IsActive() {
				if line != "" {
					buffer.AddLine(line)
				}

				// Execute buffer and clear it
				if !buffer.IsEmpty() {
					r.executeBufferSimple(buffer)
				}
				buffer.Clear()
			} else {
				// Regular mode - execute line immediately
				if line != "" {
					r.processSingleLine(line)
				}
			}
		}

		// Check for job notifications and print them
		r.checkAndPrintJobNotifications()
	}

	// Cleanup
	if err := r.engine.CleanupRuntimes(); err != nil {
		return errors.NewSystemError("CLEANUP_ERROR", fmt.Sprintf("cleanup error: %v", err))
	}

	return nil
}

// runPiped runs the REPL in piped mode, reading from stdin
func (r *REPL) runPiped() error {
	scanner := bufio.NewScanner(os.Stdin)

	// Read all lines from stdin
	for scanner.Scan() {
		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		// Process the command
		if err := r.processCommand(input); err != nil {
			r.displayError(err)
			return err
		}
	}

	if err := scanner.Err(); err != nil {
		return errors.NewSystemError("STDIN_READ_ERROR", fmt.Sprintf("error reading from stdin: %v", err))
	}

	// Cleanup
	if err := r.engine.CleanupRuntimes(); err != nil {
		return errors.NewSystemError("CLEANUP_ERROR", fmt.Sprintf("cleanup error: %v", err))
	}

	return nil
}

// printWelcome displays the welcome message
func (r *REPL) printWelcome() {
	fmt.Println("Welcome to funterm - Multi-Language REPL")
	fmt.Println("Type ':help' for available commands or ':quit' to exit")
	fmt.Println("Available languages: go, js, lua, python")
	fmt.Println()
}

// initializeRuntimes initializes the language runtimes
func (r *REPL) initializeRuntimes() error {
	// Get all supported languages from all factories
	supportedLanguages := r.registry.GetSupportedLanguages()

	// Track which factories we've already processed to avoid duplicates
	processedFactories := make(map[string]bool)

	// Create and register runtimes for each supported language
	for _, language := range supportedLanguages {
		// Get the factory for this language
		factory, err := r.registry.GetFactoryForLanguage(language)
		if err != nil {
			return errors.NewSystemError("FACTORY_RETRIEVAL_FAILED", fmt.Sprintf("failed to get factory for language '%s': %v", language, err))
		}

		// Check if we've already processed this factory
		factoryName := factory.GetName()
		if processedFactories[factoryName] {
			continue
		}
		processedFactories[factoryName] = true

		// Create runtime from the factory
		runtime, err := factory.CreateRuntime()
		if err != nil {
			return errors.NewSystemError("RUNTIME_CREATION_FAILED", fmt.Sprintf("failed to create %s runtime: %v", factoryName, err))
		}

		if err := r.engine.RegisterRuntime(runtime); err != nil {
			return errors.NewSystemError("RUNTIME_REGISTRATION_FAILED", fmt.Sprintf("failed to register %s runtime: %v", factoryName, err))
		}
	}

	// Initialize all registered runtimes
	if err := r.engine.InitializeRuntimes(); err != nil {
		return errors.NewSystemError("RUNTIME_INITIALIZATION_FAILED", fmt.Sprintf("failed to initialize runtimes: %v", err))
	}

	return nil
}

// processCommand processes a single command
func (r *REPL) processCommand(input string) error {
	// Handle terminal commands with $ prefix
	if strings.HasPrefix(input, "$") {
		if strings.HasPrefix(input, "<$") {
			// Execute terminal command and parse result
			result, err := r.executeAndParseTerminalCommand(input)
			if err != nil {
				return err
			}
			if result != nil {
				fmt.Printf("=> %v\n", r.formatResult(result))
			}
			return nil
		} else {
			// Execute terminal command without parsing
			return r.executeTerminalCommand(input)
		}
	}

	// Handle terminal commands with <$ prefix (standalone)
	if strings.HasPrefix(input, "<$") {
		// Remove the <$ prefix
		cmd := strings.TrimSpace(input[2:])
		if cmd == "" {
			return errors.NewUserError("INVALID_COMMAND", "no command provided after <$")
		}

		// Check if this is an echo command with funterm code
		if strings.HasPrefix(cmd, "echo ") {
			// Extract the funterm code from echo
			funtermCode := strings.TrimSpace(cmd[5:]) // Remove "echo "
			// Remove quotes if present
			if len(funtermCode) >= 2 && (funtermCode[0] == '\'' && funtermCode[len(funtermCode)-1] == '\'' ||
				funtermCode[0] == '"' && funtermCode[len(funtermCode)-1] == '"') {
				funtermCode = funtermCode[1 : len(funtermCode)-1]
			}

		// Execute the funterm code directly
		result, isPrint, hasResult, err := r.engine.Execute(funtermCode)
		if err != nil {
			return err
		}
		// Don't print the result if the command already produced output (isPrint flag)
		// Only show result if hasResult is true (even if result is nil)
		if hasResult && !isPrint {
			fmt.Printf("=> %v\n", r.formatResult(result))
		}
			return nil
		}

		// Execute regular terminal command and parse result
		result, err := r.executeAndParseTerminalCommand(input)
		if err != nil {
			return err
		}
		if result != nil {
			fmt.Printf("=> %v\n", r.formatResult(result))
		}
		return nil
	}

	// Handle built-in commands
	if strings.HasPrefix(input, ":") {
		return r.handleBuiltInCommand(input)
	}

	// Check if we have a cached result for this command
	if cachedResult, cacheErr, found := r.performanceOptimizer.GetCachedCommand(input); found && cacheErr == nil {
		fmt.Printf("=> (cached) %v\n", r.formatResult(cachedResult))
		return nil
	}

	// Pre-parse and optimize the command (for future use, not currently used in execution)
	_ = r.performanceOptimizer.PreParseCommand(input)

	// Execute the command
	result, isPrint, hasResult, err := r.engine.Execute(input)
	if err != nil {
		return err
	}

	// Cache the result for future use, but only for safe operations
	// Most runtime commands can have side effects or change state, so we disable caching
	// Only cache simple operations that are guaranteed to be pure functions
	// For now, disable caching entirely for safety
	// TODO: Implement fine-grained caching for truly pure operations only
	if false {
		r.performanceOptimizer.CacheCommand(input, result, nil)
	}

	// Print the result
	if hasResult {
		// Check if this is a job ID (background task)
		if jobID, ok := result.(jobmanager.JobID); ok {
			fmt.Printf("[%d] %s\n", jobID, r.history[len(r.history)-1])
		} else {
			// Don't print the result if the command already produced output (isPrint flag)
			if !isPrint {
				fmt.Printf("=> %v\n", r.formatResult(result))
			}
		}
	} else {
		// Show execution indicator for commands without results (like lua.print)
		// But only if the command doesn't contain print functions (they already produced output)
		if !isPrint {
			fmt.Printf("✓ Executed\n")
		}
	}

	return nil
}

// handleBuiltInCommand handles built-in REPL commands
func (r *REPL) handleBuiltInCommand(input string) error {
	cmd := strings.TrimSpace(input[1:]) // Remove the ':'
	parts := strings.Fields(cmd)
	command := parts[0]

	// First try to handle with advanced commands
	if r.advancedCommands != nil {
		// Check if this is an advanced command and handle it
		switch cmd {
		case "debug", "breakpoint", "step", "continue", "inspect", "stack",
			"profile", "benchmark", "memory", "gc",
			"save", "load", "reset", "snapshot",
			"runtimes", "isolate", "pool",
			"analyze", "dependencies", "performance":
			// Parse the command and arguments
			parts := strings.Fields(cmd)
			args := []string{}
			if len(parts) > 1 {
				args = parts[1:]
			}

			result, err := r.handleAdvancedCommand(parts[0], args)
			if err != nil {
				return err
			}
			if result != nil {
				fmt.Printf("=> %v\n", r.formatResult(result))
			}
			return nil
		}
	}

	// Fall back to basic commands if not handled by advanced commands
	switch command {
	case "help", "h":
		r.printHelp()
	case "quit", "q", "exit", "e":
		r.running = false
	case "languages", "l":
		r.printAvailableLanguages()
	case "history", "hist":
		r.printHistory()
	case "clear", "c":
		r.clearScreen()
	case "version", "v":
		r.printVersion()
	// case "run", "r":
	// 	if len(parts) < 3 {
	// 		return errors.NewUserError("INVALID_COMMAND", "usage: :run <language> <file-path>")
	// 	}
	// 	language := parts[1]
	// 	filePath := parts[2]
	// 	return r.executeFile(language, filePath)
	case "run", "r":
		if len(parts) < 2 {
			return errors.NewUserError("INVALID_COMMAND", "usage: :run <file-path>")
		}
		filePath := parts[1]
		return r.executeMixedFile(filePath)
	case "jobs":
		return r.printJobs()
	default:
		// Check if this is a language command (lua, python, js, etc.)
		if r.isLanguageCommand(command) {
			if len(parts) == 1 {
				// Command like ":lua" - show language info
				return r.printLanguageInfo(command)
			} else if len(parts) == 2 {
				// Command like ":lua math" - show module functions
				moduleName := parts[1]
				return r.printModuleFunctions(command, moduleName)
			} else {
				return errors.NewUserError("INVALID_COMMAND", fmt.Sprintf("usage: :%s [module]", command))
			}
		}
		return errors.NewUserError("UNKNOWN_COMMAND", fmt.Sprintf("unknown command: :%s", command))
	}

	return nil
}

// printHelp displays help information
func (r *REPL) printHelp() {
	fmt.Println("Available commands:")
	fmt.Println("  :help, :h               - Show this help message")
	fmt.Println("  :help ml, :h ml         - Show multiline help message")
	fmt.Println("  :quit, :q, :exit, :e    - Exit the REPL")
	fmt.Println("  :languages, :l          - List available languages")
	fmt.Println("  :history, :hist         - Show command history")
	fmt.Println("  :clear, :c              - Clear the screen")
	fmt.Println("  :version, :v            - Show version information")
	// fmt.Println("  :run <lang> <file> - Execute code from file in specified language")
	// fmt.Println("  :mixed <file>      - Execute mixed language code from file")
	fmt.Println("  :run <file>, r: <file>  - Execute mixed language code from file")
	fmt.Println("  :jobs                   - List background jobs and their status")
	fmt.Println()

	fmt.Println("Terminal commands:")
	fmt.Println("  $ command          - Execute terminal command without parsing result")
	fmt.Println("  <$ command         - Execute terminal command and parse result")
	fmt.Println("  Example: $ echo \"Hello World\"")
	fmt.Println("  Example: <$ ls -la")
	fmt.Println()

	fmt.Println("Language exploration commands:")
	fmt.Println("  :lua              - Show available Lua modules and functions")
	fmt.Println("  :python, :py      - Show available Python modules and functions")
	fmt.Println("  :js, :node        - Show available JavaScript/Node.js modules and functions")
	fmt.Println("  :go               - Show available Go modules and functions")
	fmt.Println("  :lua math         - Show functions in Lua math module")
	fmt.Println("  :python json      - Show functions in Python json module")
	fmt.Println("  :js fs            - Show functions in JavaScript fs module")
	fmt.Println()

	fmt.Println("Language call syntax:")
	fmt.Println("  language.function(arg1, arg2, ...)")
	fmt.Println("  Example: lua.print('Hello')")
	fmt.Println("  Example: python.math.sin(3.14)")
	fmt.Println("  Example: python.str.upper('hello')")
	fmt.Println()
}

// printAvailableLanguages displays available languages
func (r *REPL) printAvailableLanguages() {
	languages := r.engine.ListAvailableLanguages()
	if len(languages) == 0 {
		fmt.Println("No language runtimes available")
		return
	} else if len(languages) > 1 {
		sort.Strings(languages)
	}

	fmt.Println("Available languages:")
	for _, lang := range languages {
		ready := "✓"
		if !r.engine.IsLanguageAvailable(lang) {
			ready = "✗"
		}
		description := ""
		skip := false

		// Add descriptions for built-in languages
		switch lang {
		case "js", "py":
			skip = true
		case "node":
			description = " (also available as js)"
		case "python":
			description = " (also available as py)"
		}
		if skip {
			continue
		}

		fmt.Printf("  %s %s%s\n", ready, lang, description)
	}
}

// printHistory displays command history
func (r *REPL) printHistory() {
	if len(r.history) == 0 {
		fmt.Println("No command history")
		return
	}

	fmt.Println("Command history:")
	for i, cmd := range r.history {
		fmt.Printf("  %d: %s\n", i+1, cmd)
	}
}

// clearScreen clears the terminal screen
func (r *REPL) clearScreen() {
	fmt.Print("\033[H\033[2J") // ANSI escape codes to clear screen
}

// printVersion displays version information
func (r *REPL) printVersion() {
	fmt.Println("funterm v0.1.0 - Multi-Language REPL")
}

// formatResult formats the result for display
func (r *REPL) formatResult(result interface{}) string {
	if result == nil {
		return "nil"
	}
	// For pre-formatted results (like from print function), return as-is
	if preFormatted, ok := result.(*shared.PreFormattedResult); ok {
		return preFormatted.Value
	}
	// For strings, add quotes like the old formatResult did, but not for multi-line output (likely from print functions)
	if str, ok := result.(string); ok {
		if strings.Contains(str, "\n") {
			// Multi-line string, likely from print functions - return as-is
			return str
		}
		return fmt.Sprintf("\"%s\"", str)
	}
	// For all other types, use the shared formatter
	return shared.FormatValueForDisplay(result)
}

// GetEngine returns the execution engine (useful for testing)
func (r *REPL) GetEngine() *engine.ExecutionEngine {
	return r.engine
}

// GetHistory returns the command history
func (r *REPL) GetHistory() []string {
	return r.history
}

// SetPrompt sets the REPL prompt
func (r *REPL) SetPrompt(prompt string) {
	r.prompt = prompt
}

// SetWelcomeMessage controls whether to show the welcome message
func (r *REPL) SetWelcomeMessage(show bool) {
	r.showWelcome = show
}

// GetRegistry returns the runtime registry used by this REPL
func (r *REPL) GetRegistry() *factory.RuntimeRegistry {
	return r.registry
}

// InitializeRuntimes initializes all language runtimes
func (r *REPL) InitializeRuntimes() error {
	return r.initializeRuntimes()
}

// executeFile выполняет код из файла для указанного языка
func (r *REPL) executeFile(language, filePath string) error {
	// Проверяем, доступен ли язык
	if !r.engine.IsLanguageAvailable(language) {
		return errors.NewUserError("LANGUAGE_NOT_AVAILABLE", fmt.Sprintf("language '%s' is not available", language))
	}

	// Читаем содержимое файла
	content, err := os.ReadFile(filePath)
	if err != nil {
		return errors.NewSystemError("FILE_READ_ERROR", fmt.Sprintf("failed to read file: %v", err))
	}

	// Проверяем, что runtime для языка существует
	if !r.engine.IsLanguageAvailable(language) {
		return errors.NewSystemError("RUNTIME_NOT_FOUND", fmt.Sprintf("runtime for language '%s' not found", language))
	}

	// Выполняем код построчно
	lines := strings.Split(string(content), "\n")
	var result interface{}

	fmt.Printf("Executing %s file: %s (%d lines)\n", language, filePath, len(lines))

	for i, line := range lines {
		// Пропускаем пустые строки и комментарии
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") || strings.HasPrefix(line, "--") {
			continue
		}

		// Формируем команду для выполнения
		cmd := fmt.Sprintf("%s.eval(\"%s\")", language, escapeString(line))

	// Выполняем команду
	result, _, _, err = r.engine.Execute(cmd)
		if err != nil {
			return errors.NewSystemError("EXECUTION_ERROR", fmt.Sprintf("error at line %d: %v", i+1, err))
		}
	}

	fmt.Println("File executed successfully")

	// Если есть результат последней команды, выводим его
	if result != nil {
		fmt.Printf("=> %v\n", r.formatResult(result))
	}

	return nil
}

// executeMixedFile выполняет смешанный код из файла
func (r *REPL) executeMixedFile(filePath string) error {
	// Читаем содержимое файла
	content, err := os.ReadFile(filePath)
	if err != nil {
		return errors.NewSystemError("FILE_READ_ERROR", fmt.Sprintf("failed to read file: %v", err))
	}

	fileContent := string(content)

	if r.verbose {
		fmt.Printf("Executing mixed language file: %s (%d characters)\n", filePath, len(fileContent))
	}

	// Выполняем весь файл как единое целое через ExecutionEngine
	// Это позволяет правильно обрабатывать многострочные конструкции как блоки кода
	result, _, _, err := r.engine.Execute(fileContent)
	if err != nil {
		return errors.NewSystemError("EXECUTION_ERROR", fmt.Sprintf("error executing file: %v", err))
	}

	// Выводим результат выполнения, если он не пустой
	if result != nil && result != "" {
		fmt.Printf("=> %v\n", r.formatResult(result))
	}

	if r.verbose {
		fmt.Println("Mixed file executed successfully")
	}
	return nil
}

// handleAdvancedCommand handles advanced commands using the AdvancedCommands instance
func (r *REPL) handleAdvancedCommand(command string, args []string) (interface{}, error) {
	if r.advancedCommands == nil {
		return nil, fmt.Errorf("advanced commands not initialized")
	}

	// Map command names to their handlers
	switch command {
	case "debug":
		return r.advancedCommands.HandleDebugCommand(args)
	case "breakpoint":
		return r.advancedCommands.HandleBreakpointCommand(args)
	case "step":
		return r.advancedCommands.HandleStepCommand(args)
	case "continue":
		return r.advancedCommands.HandleContinueCommand(args)
	case "inspect":
		return r.advancedCommands.HandleInspectCommand(args)
	case "stack":
		return r.advancedCommands.HandleStackCommand(args)
	case "profile":
		return r.advancedCommands.HandleProfileCommand(args)
	case "benchmark":
		return r.advancedCommands.HandleBenchmarkCommand(args)
	case "memory":
		return r.advancedCommands.HandleMemoryCommand(args)
	case "gc":
		return r.advancedCommands.HandleGCCommand(args)
	case "save":
		return r.advancedCommands.HandleSaveStateCommand(args)
	case "load":
		return r.advancedCommands.HandleLoadStateCommand(args)
	case "reset":
		return r.advancedCommands.HandleResetCommand(args)
	case "snapshot":
		return r.advancedCommands.HandleSnapshotCommand(args)
	case "runtimes":
		return r.advancedCommands.HandleRuntimesCommand(args)
	case "isolate":
		return r.advancedCommands.HandleIsolateCommand(args)
	case "pool":
		return r.advancedCommands.HandlePoolCommand(args)
	case "analyze":
		return r.advancedCommands.HandleAnalyzeCommand(args)
	case "dependencies":
		return r.advancedCommands.HandleDependenciesCommand(args)
	case "performance":
		return r.advancedCommands.HandlePerformanceStatsCommand(args)
	default:
		return nil, fmt.Errorf("unknown advanced command: %s", command)
	}
}

// escapeString экранирует строку для безопасного использования в строковом литерале
func escapeString(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	s = strings.ReplaceAll(s, "\t", "\\t")
	return s
}

// EnableOptimization enables or disables performance optimization
func (r *REPL) EnableOptimization(enabled bool) {
	if r.performanceOptimizer != nil {
		r.performanceOptimizer.SetEnabled(enabled)
	}
	if r.optimizer != nil {
		r.optimizer.SetEnabled(enabled)
	}
}

// ExecuteCommand executes a single command (for backward compatibility)
func (r *REPL) ExecuteCommand(command string) (interface{}, error) {
	// Check if we have a cached result for this command
	if cachedResult, cacheErr, found := r.performanceOptimizer.GetCachedCommand(command); found && cacheErr == nil {
		return cachedResult, nil
	}

	// Pre-parse and optimize the command (for future use, not currently used in execution)
	_ = r.performanceOptimizer.PreParseCommand(command)

	// Execute the command
	result, _, _, err := r.engine.Execute(command)
	if err != nil {
		return nil, err
	}

	// Cache the result for future use, but only for safe operations
	// Most runtime commands can have side effects or change state, so we disable caching
	// Only cache simple operations that are guaranteed to be pure functions
	// For now, disable caching entirely for safety
	// TODO: Implement fine-grained caching for truly pure operations only
	if false {
		r.performanceOptimizer.CacheCommand(command, result, nil)
	}

	return result, nil
}

// checkAndPrintJobNotifications checks for pending job notifications and prints them
func (r *REPL) checkAndPrintJobNotifications() {
	// Check for job notifications without blocking
	select {
	case notification := <-r.jobNotifications:
		r.printJobNotification(notification)

		// Process any additional notifications that might be available
		for {
			select {
			case additionalNotification := <-r.jobNotifications:
				r.printJobNotification(additionalNotification)
			default:
				// No more notifications available
				return
			}
		}
	default:
		// No notifications available
		return
	}
}

// printJobNotification prints a job notification in a user-friendly format
func (r *REPL) printJobNotification(notification jobmanager.JobNotification) {
	switch notification.Status {
	case jobmanager.StatusCompleted:
		fmt.Printf("[%d]+ Done %s\n", notification.JobID, notification.Result)
	case jobmanager.StatusFailed:
		fmt.Printf("[%d]+ Error: %v\n", notification.JobID, notification.Error)
	default:
		// Should not happen for job completion notifications
		fmt.Printf("[%d]+ Status: %s\n", notification.JobID, notification.Status)
	}
}

// printJobs displays all background jobs and their status
func (r *REPL) printJobs() error {
	jobs := r.engine.ListJobs()
	if len(jobs) == 0 {
		fmt.Println("No background jobs")
		return nil
	}

	fmt.Println("Background jobs:")
	for _, job := range jobs {
		duration := job.GetDuration().String()

		switch job.GetStatus() {
		case jobmanager.StatusRunning:
			fmt.Printf("  [%d] %s - Running (%s)\n", job.ID, job.Command, duration)
		case jobmanager.StatusCompleted:
			fmt.Printf("  [%d] %s - Done (%s)\n", job.ID, job.Command, duration)
		case jobmanager.StatusFailed:
			fmt.Printf("  [%d] %s - Failed (%s) - %v\n", job.ID, job.Command, duration, job.GetError())
		}
	}

	return nil
}

// executeTerminalCommand executes a terminal command without parsing the result
func (r *REPL) executeTerminalCommand(command string) error {
	// Remove the $ prefix
	cmd := strings.TrimSpace(command[1:])
	if cmd == "" {
		return errors.NewUserError("INVALID_COMMAND", "no command provided after $")
	}

	// Use shell to properly handle environment variables, pipes, and special characters
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/bash" // fallback to bash
	}

	// Create the command with shell
	execCmd := exec.Command(shell, "-c", cmd)

	// Capture output to check if it ends with newline
	var stdout, stderr bytes.Buffer
	execCmd.Stdout = &stdout
	execCmd.Stderr = &stderr

	// Execute the command
	if err := execCmd.Run(); err != nil {
		// Print stderr if there was an error
		if stderr.Len() > 0 {
			os.Stderr.Write(stderr.Bytes())
			// Add newline if stderr doesn't end with one
			if stderrBytes := stderr.Bytes(); len(stderrBytes) > 0 && stderrBytes[len(stderrBytes)-1] != '\n' {
				fmt.Fprint(os.Stderr, "\n")
			}
		}
		return errors.NewSystemError("COMMAND_EXECUTION_FAILED", fmt.Sprintf("failed to execute command: %v", err))
	}

	// Print stdout
	if stdout.Len() > 0 {
		os.Stdout.Write(stdout.Bytes())
		// Add newline only if stdout doesn't end with one
		if stdoutBytes := stdout.Bytes(); len(stdoutBytes) > 0 && stdoutBytes[len(stdoutBytes)-1] != '\n' {
			fmt.Print("\n")
		}
	}

	// Print stderr
	if stderr.Len() > 0 {
		os.Stderr.Write(stderr.Bytes())
		// Add newline only if stderr doesn't end with one
		if stderrBytes := stderr.Bytes(); len(stderrBytes) > 0 && stderrBytes[len(stderrBytes)-1] != '\n' {
			fmt.Fprint(os.Stderr, "\n")
		}
	}

	return nil
}

// executeAndParseTerminalCommand executes a terminal command and parses the result
func (r *REPL) executeAndParseTerminalCommand(command string) (interface{}, error) {
	// Remove the <$ prefix
	cmd := strings.TrimSpace(command[2:])
	if cmd == "" {
		return nil, errors.NewUserError("INVALID_COMMAND", "no command provided after <$")
	}

	// Use shell to properly handle environment variables, pipes, and special characters
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/bash" // fallback to bash
	}

	// Create the command with shell
	execCmd := exec.Command(shell, "-c", cmd)

	// Capture stdout and stderr
	var stdout, stderr strings.Builder
	execCmd.Stdout = &stdout
	execCmd.Stderr = &stderr

	// Execute the command with timeout
	timeout := 30 * time.Second
	done := make(chan error, 1)

	go func() {
		done <- execCmd.Run()
	}()

	select {
	case err := <-done:
		if err != nil {
			// Return error as string for parsing
			errorOutput := stderr.String()
			if errorOutput == "" {
				errorOutput = err.Error()
			}
			return errorOutput, nil
		}

		// Return stdout as the result
		result := strings.TrimSpace(stdout.String())
		if result == "" {
			return "✓ Executed", nil
		}
		return result, nil

	case <-time.After(timeout):
		// Try to kill the process if it's still running
		if execCmd.Process != nil {
			if err := execCmd.Process.Kill(); err != nil {
				// Log the error but continue
				fmt.Printf("Warning: Failed to kill process: %v\n", err)
			}
		}
		return nil, errors.NewSystemError("COMMAND_TIMEOUT", "command execution timed out")
	}
}

// isLanguageCommand checks if a command is a supported language name
func (r *REPL) isLanguageCommand(command string) bool {
	// Get all supported languages from the engine
	languages := r.engine.ListAvailableLanguages()

	// Check if the command matches any supported language
	for _, lang := range languages {
		if command == lang {
			return true
		}
	}

	// Check common aliases
	aliases := map[string]bool{
		"py": true,
		"js": true,
	}

	return aliases[command]
}

// printLanguageInfo displays available modules and functions for a specific language
func (r *REPL) printLanguageInfo(language string) error {
	// Get the runtime for the specified language
	var runtimeName string

	// Handle aliases
	switch language {
	case "py":
		runtimeName = "python"
	case "js":
		runtimeName = "node"
	default:
		runtimeName = language
	}

	// Check if the language is available
	if !r.engine.IsLanguageAvailable(runtimeName) {
		return errors.NewUserError("LANGUAGE_NOT_AVAILABLE", fmt.Sprintf("language '%s' is not available", language))
	}

	// Get the runtime through runtime manager
	runtimeManager := r.engine.GetRuntimeManager()
	runtime, err := runtimeManager.GetRuntime(runtimeName)
	if err != nil {
		return errors.NewUserError("RUNTIME_ERROR", fmt.Sprintf("failed to get runtime for '%s': %v", language, err))
	}

	// Get modules (already sorted in the runtime)
	modules := runtime.GetModules()
	if len(modules) > 0 {
		fmt.Printf("Available modules:\n  ")
		for i, module := range modules {
			if i > 0 {
				fmt.Print(", ")
			}
			fmt.Print(module)
		}
		fmt.Println()
	}

	// Get built-in functions (global variables that are functions)
	globalVars := runtime.GetGlobalVariables()
	var functions []string
	for _, variable := range globalVars {
		// Filter out internal variables and keep likely function names
		if strings.HasPrefix(variable, "__") {
			functions = append(functions, variable)
		}
	}

	// Remove duplicates and filter
	if len(functions) > 0 {
		// Simple deduplication
		uniqueFunctions := make(map[string]bool)
		var filteredFunctions []string
		for _, fn := range functions {
			if !uniqueFunctions[fn] {
				uniqueFunctions[fn] = true
				// Skip common non-function global variables
				if fn != "true" && fn != "false" && fn != "null" && fn != "undefined" &&
					!strings.Contains(fn, ".") && len(fn) > 1 {
					filteredFunctions = append(filteredFunctions, fn)
				}
			}
		}

		if len(filteredFunctions) > 0 {
			fmt.Println("\nAvailable functions:")
			for i, fn := range filteredFunctions {
				fmt.Printf("  %s", fn)
				// Add line breaks for better formatting (same as module functions)
				if (i+1)%6 == 0 {
					fmt.Println()
				} else if i < len(filteredFunctions)-1 {
					fmt.Print(", ")
				}
			}
			fmt.Println()
		}
	}

	return nil
}

// printModuleFunctions displays available functions for a specific module
func (r *REPL) printModuleFunctions(language, module string) error {
	// Get the runtime for the specified language
	var runtimeName string

	// Handle aliases
	switch language {
	case "py":
		runtimeName = "python"
	case "js":
		runtimeName = "node"
	default:
		runtimeName = language
	}

	// Check if the language is available
	if !r.engine.IsLanguageAvailable(runtimeName) {
		return errors.NewUserError("LANGUAGE_NOT_AVAILABLE", fmt.Sprintf("language '%s' is not available", language))
	}

	// Get the runtime through runtime manager
	runtimeManager := r.engine.GetRuntimeManager()
	runtime, err := runtimeManager.GetRuntime(runtimeName)
	if err != nil {
		return errors.NewUserError("RUNTIME_ERROR", fmt.Sprintf("failed to get runtime for '%s': %v", language, err))
	}

	// Get functions for the module (already sorted in the runtime)
	functions := runtime.GetModuleFunctions(module)
	if len(functions) == 0 {
		fmt.Printf("No functions found in module '%s' for language '%s'\n", module, language)
		return nil
	}

	fmt.Printf("Available functions in %s module:\n", module)
	for i, fn := range functions {
		fmt.Printf("  %s", fn)
		// Add line breaks for better formatting
		if (i+1)%6 == 0 {
			fmt.Println()
		} else if i < len(functions)-1 {
			fmt.Print(", ")
		}
	}
	fmt.Println()

	return nil
}

// isShiftEnter determines if multiline input should continue (heuristic)
func isShiftEnter(line string) bool {
	trimmed := strings.TrimSpace(line)

	// If line is empty, it's definitely not a continuation
	if trimmed == "" {
		return false
	}

	// Check if line ends with \ for continuation (Shift+Enter simulation)
	return strings.HasSuffix(trimmed, "\\")
}

// executeBufferSimple executes the contents of the multiline buffer
func (r *REPL) executeBufferSimple(buffer *MultiLineBuffer) {
	content := buffer.GetContent()
	if strings.TrimSpace(content) == "" {
		return
	}

	// Show that we're executing the buffer
	fmt.Printf("Executing a buffer (%d lines):\n", buffer.GetLineCount())

	// Execute the code
	result, isPrint, hasResult, err := r.engine.Execute(content)

	// Show the result
	if err != nil {
		r.displayError(err)
	} else if hasResult {
		fmt.Printf("=> %v\n", r.formatResult(result))
	} else if !isPrint {
		fmt.Println("✓ Executed")
	}
}

// handleSpecialCommandsSimple handles special commands for buffer management
// This function now delegates to handleBuiltInCommand to avoid duplication
func (r *REPL) handleSpecialCommandsSimple(line string, buffer *MultiLineBuffer) bool {
	// Handle buffer-specific commands first
	switch line {
	case ":reset", ":rb":
		buffer.Clear()
		fmt.Println("The buffer has been reset")
		return true
	case ":buffer", ":b":
		if buffer.IsActive() {
			fmt.Printf("The buffer contains %d lines:\n", buffer.GetLineCount())
			for i, line := range buffer.GetLines() {
				fmt.Printf("%2d: %s\n", i+1, line)
			}
		} else {
			fmt.Println("The buffer is empty")
		}
		return true
	case ":multiline", ":ml":
		buffer.SetActive(true)
		fmt.Println("Multiline mode activated. Use :reset to clear buffer.")
		return true
	case ":help ml", ":h ml":
		fmt.Println("Multi-line Funterm Mode:")
		fmt.Println("  Enter            - execute the code (buffer or line)")
		fmt.Println("  \\ at end         - add a line to the buffer")
		fmt.Println("  :reset, :rb      - reset the buffer")
		fmt.Println("  :buffer, :b      - show buffer contents")
		fmt.Println("  :help ml, :h ml  - this help")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  >>> python.def add(a, b):\\")
		fmt.Println("  ...     return a + b")
		fmt.Println("  ... ")
		fmt.Println("  ✓ Executed")
		return true
	}

	// For all other commands, delegate to handleBuiltInCommand
	// Convert the error return to bool (true if no error)
	if err := r.handleBuiltInCommand(line); err != nil {
		// If there was an error, the command wasn't handled
		return false
	}
	return true
}

// processSingleLine processes a single line of input
func (r *REPL) processSingleLine(line string) {
	// Process the command
	if err := r.processCommand(line); err != nil {
		r.displayError(err)
	}
}

// IsMultilineEnabled returns whether multiline mode is enabled
func (r *REPL) IsMultilineEnabled() bool {
	return r.enableMultiline
}

// IsColorEnabled returns whether color output is enabled
func (r *REPL) IsColorEnabled() bool {
	return r.enableColors
}

// GetBuffer returns the multiline buffer
func (r *REPL) GetBuffer() *MultiLineBuffer {
	return r.buffer
}

// GetDisplayManager returns the display manager
func (r *REPL) GetDisplayManager() *DisplayManager {
	return r.displayManager
}

// ClearBuffer clears the multiline buffer
func (r *REPL) ClearBuffer() {
	if r.buffer != nil {
		r.buffer.Clear()
	}
}

// hasIncompleteSyntax always returns false in the simple implementation
func (r *REPL) hasIncompleteSyntax(line string) bool {
	// In the simple implementation, we don't do automatic syntax detection
	// Multiline mode is controlled manually by the user
	return false
}

// handleSingleDollarCommand handles $ command - execute terminal command and print output
func (r *REPL) handleSingleDollarCommand(command string) {
	// Remove the $ prefix
	cmd := strings.TrimSpace(command[1:])
	if cmd == "" {
		fmt.Println("Error: no command provided after $")
		return
	}

	// Use shell to properly handle environment variables, pipes, and special characters
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/bash" // fallback to bash
	}

	// Create the command with shell
	execCmd := exec.Command(shell, "-c", cmd)

	// Capture output to check if it ends with newline
	var stdout, stderr bytes.Buffer
	execCmd.Stdout = &stdout
	execCmd.Stderr = &stderr

	// Execute the command
	if err := execCmd.Run(); err != nil {
		// Print stderr if there was an error
		if stderr.Len() > 0 {
			os.Stderr.Write(stderr.Bytes())
			// Add newline if stderr doesn't end with one
			if stderrBytes := stderr.Bytes(); len(stderrBytes) > 0 && stderrBytes[len(stderrBytes)-1] != '\n' {
				fmt.Fprint(os.Stderr, "\n")
			}
		}
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Print stdout
	if stdout.Len() > 0 {
		os.Stdout.Write(stdout.Bytes())
		// Add newline only if stdout doesn't end with one
		if stdoutBytes := stdout.Bytes(); len(stdoutBytes) > 0 && stdoutBytes[len(stdoutBytes)-1] != '\n' {
			fmt.Print("\n")
		}
	}

	// Print stderr
	if stderr.Len() > 0 {
		os.Stderr.Write(stderr.Bytes())
		// Add newline only if stderr doesn't end with one
		if stderrBytes := stderr.Bytes(); len(stderrBytes) > 0 && stderrBytes[len(stderrBytes)-1] != '\n' {
			fmt.Fprint(os.Stderr, "\n")
		}
	}
}

// handleDollarCommand handles <$ command - execute terminal command and pass output to funterm
func (r *REPL) handleDollarCommand(command string, buffer *MultiLineBuffer) {
	// Remove the <$ prefix
	cmd := strings.TrimSpace(command[2:])
	if cmd == "" {
		fmt.Println("Error: no command provided after <$")
		return
	}

	// Use shell to properly handle environment variables, pipes, and special characters
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/bash" // fallback to bash
	}

	// Create the command with shell
	execCmd := exec.Command(shell, "-c", cmd)

	// Capture stdout and stderr
	var stdout, stderr strings.Builder
	execCmd.Stdout = &stdout
	execCmd.Stderr = &stderr

	// Execute the command
	if err := execCmd.Run(); err != nil {
		errorOutput := stderr.String()
		if errorOutput == "" {
			errorOutput = err.Error()
		}
		fmt.Printf("Error: %s\n", errorOutput)
		return
	}

	// Get the output and pass it to funterm
	output := strings.TrimSpace(stdout.String())
	if output == "" {
		return
	}

	// Execute the output as a funterm command
	result, isPrint, hasResult, err := r.engine.Execute(output)
	if err != nil {
		r.displayError(err)
	} else if hasResult {
		fmt.Printf("=> %v\n", r.formatResult(result))
	} else if !isPrint {
		fmt.Println("✓ Executed")
	}
}

// displayError displays an error with position information if available
func (r *REPL) displayError(err error) {
	// Check if this is an ExecutionError with position information
	if execErr, ok := err.(*errors.ExecutionError); ok {
		if execErr.Line > 0 && execErr.Col >= 0 {
			fmt.Printf("Error at line %d, col %d: %s\n", execErr.Line, execErr.Col, execErr.Message)
		} else if execErr.Line > 0 {
			fmt.Printf("Error at line %d: %s\n", execErr.Line, execErr.Message)
		} else {
			fmt.Printf("Error: %s\n", execErr.Message)
		}
	} else {
		// For other error types, just display the error
		fmt.Printf("Error: %v\n", err)
	}
}
