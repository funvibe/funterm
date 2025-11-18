package main

import (
	"flag"
	"fmt"
	"funterm/factory"
	"funterm/repl"
	"funterm/runtime/python"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func main() {

	// Parse command line flags
	var (
		configPath  = flag.String("config", "", "Path to configuration file")
		showVersion = flag.Bool("version", false, "Show version information")
		showHelp    = flag.Bool("help", false, "Show help information")
		execFile    = flag.String("exec", "", "Execute file in batch mode")
		language    = flag.String("lang", "", "Language for file execution (lua, python, go, mixed)")

		// Package management flags
		packages      = flag.String("packages", "", "Python package management (list, install, check)")
		packageTarget = flag.String("package-name", "", "Target package for install/check operations")

		// Module management flags
		modules      = flag.String("modules", "", "Lua module management (list, info, test)")
		moduleTarget = flag.String("module-name", "", "Target module for info/test operations")

		// Diagnostic flags
		doctor  = flag.Bool("doctor", false, "Run system diagnostics")
		verbose = flag.Bool("verbose", false, "Enable verbose output")
		envInfo = flag.Bool("env-info", false, "Show Python environment information")
	)
	flag.Parse()

	// Handle shebang execution (when script is run as ./script.su)
	args := flag.Args()
	if len(args) > 0 && *execFile == "" {
		// Check if the argument is a .su file
		filePath := args[0]
		if strings.HasSuffix(filePath, ".su") {
			// Parse additional arguments that might be passed to the script
			shebangVerbose := *verbose
			shebangLanguage := *language
			shebangConfigPath := *configPath

			// Check if there are additional arguments after the filename
			for i := 1; i < len(args); i++ {
				switch args[i] {
				case "--verbose", "-v":
					shebangVerbose = true
				case "--lang":
					if i+1 < len(args) {
						shebangLanguage = args[i+1]
						i++ // Skip next arg
					}
				case "--config":
					if i+1 < len(args) {
						shebangConfigPath = args[i+1]
						i++ // Skip next arg
					}
				}
			}

			// Automatically execute .su files in batch mode
			if err := BatchMode(filePath, shebangLanguage, shebangConfigPath, shebangVerbose); err != nil {
				fmt.Printf("Error executing script: %v\n", err)
				os.Exit(1)
			}
			os.Exit(0)
		}
	}

	// Handle version flag with verbose support
	if *showVersion {
		if *verbose {
			fmt.Println("funterm v0.1.0 - Multi-Language REPL")
			fmt.Println("Build: development")
			fmt.Println("Go Version: runtime.Version()")
			fmt.Println("Supported Languages: Go, JS, Lua, Python")
		} else {
			fmt.Println("funterm v0.1.0 - Multi-Language REPL")
		}
		os.Exit(0)
	}

	// Handle help flag
	if *showHelp {
		printHelp()
		os.Exit(0)
	}

	// Handle package management commands
	if *packages != "" {
		// Check if there are additional arguments after flag.Parse()
		args := flag.Args()
		var additionalTarget string
		if len(args) > 0 {
			additionalTarget = args[0]
		}

		// Use additional target if packageTarget is not provided
		finalTarget := *packageTarget
		if finalTarget == "" {
			finalTarget = additionalTarget
		}

		if err := handlePythonPackages(*packages, finalTarget, *configPath, *verbose); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Handle module management commands
	if *modules != "" {
		// Check if there are additional arguments after flag.Parse()
		args := flag.Args()
		var additionalTarget string
		if len(args) > 0 {
			additionalTarget = args[0]
		}

		// Use additional target if moduleTarget is not provided
		finalTarget := *moduleTarget
		if finalTarget == "" {
			finalTarget = additionalTarget
		}

		if err := handleLuaModules(*modules, finalTarget, *configPath, *verbose); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Handle diagnostic commands
	if *doctor {
		if err := runDiagnostics(*configPath, *verbose); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Handle environment info command
	if *envInfo {
		if err := showPythonEnvironmentInfo(*configPath, *verbose); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Если указан файл для выполнения, запускаем в пакетном режиме
	if *execFile != "" {
		if err := BatchMode(*execFile, *language, *configPath, *verbose); err != nil {
			fmt.Printf("Ошибка выполнения файла: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Load configuration
	configFilePath := *configPath
	if configFilePath == "" {
		// Try default locations
		home, _ := os.UserHomeDir()
		defaultPaths := []string{
			filepath.Join(home, ".funterm", "config.yaml"),
			"./config.yaml",
		}
		for _, path := range defaultPaths {
			if _, err := os.Stat(path); err == nil {
				configFilePath = path
				break
			}
		}
	}

	cfg, err := LoadConfig(configFilePath)
	if err != nil {
		fmt.Printf("Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	// Override config with command line flags
	if *verbose {
		cfg.Engine.Verbose = true
	}

	// Create runtime registry with disabled languages support
	registry := factory.NewRuntimeRegistry()

	// Register runtimes based on configuration
	if !cfg.IsLanguageDisabled("lua") {
		luaFactory := factory.NewLuaRuntimeFactory()
		if err := registry.RegisterFactory(luaFactory); err != nil {
			fmt.Printf("Warning: Failed to register Lua runtime: %v\n", err)
		}
	}

	if !cfg.IsLanguageDisabled("python") && !cfg.IsLanguageDisabled("py") {
		pythonPath := cfg.GetRuntimePath("python")
		executionTimeout := time.Duration(cfg.Engine.MaxExecutionTime) * time.Second
		pythonFactory := factory.NewPythonRuntimeFactoryWithConfig(pythonPath, cfg.Engine.Verbose, executionTimeout)
		if err := registry.RegisterFactory(pythonFactory); err != nil {
			fmt.Printf("Warning: Failed to register Python runtime: %v\n", err)
		}
	}

	if !cfg.IsLanguageDisabled("go") {
		goFactory := factory.NewGoRuntimeFactory()
		if err := registry.RegisterFactory(goFactory); err != nil {
			fmt.Printf("Warning: Failed to register Go runtime: %v\n", err)
		}
	}

	if !cfg.IsLanguageDisabled("node") && !cfg.IsLanguageDisabled("js") && !cfg.IsLanguageDisabled("javascript") {
		nodeFactory := factory.NewNodeRuntimeFactory()
		if err := registry.RegisterFactory(nodeFactory); err != nil {
			fmt.Printf("Warning: Failed to register Node.js runtime: %v\n", err)
		}
	}

	// Create REPL with configuration
	replInstance := repl.NewREPLWithConfig(repl.REPLConfig{
		Registry:       registry,
		Verbose:        cfg.Engine.Verbose,
		Prompt:         cfg.REPL.Prompt,
		ContinuePrompt: "... ", // Default continuation prompt
		HistoryFile:    cfg.REPL.HistoryFile,
		HistorySize:    cfg.REPL.HistorySize,
	})
	// Run the REPL
	if err := replInstance.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

// printHelp displays help information
func printHelp() {
	fmt.Println("funterm - Multi-Language REPL")
	fmt.Println()
	fmt.Println("Usage: funterm [options]")
	fmt.Println("Run script: funterm <path-to-file>")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --config <path>           Path to configuration file")
	fmt.Println("  --version                 Show version information")
	fmt.Println("  --version --verbose       Show detailed version information")
	fmt.Println("  --help                    Show this help message")
	// fmt.Println("  --exec <file>             Execute file in batch mode")
	// fmt.Println("  --lang <language>         Specify language for file execution (lua, python, go, mixed)")
	fmt.Println()
	fmt.Println("Package Management:")
	fmt.Println("  --packages <command>      Python package management")
	fmt.Println("  --package-name <name>     Target package for install/check operations")
	fmt.Println("    Commands:")
	fmt.Println("      list                   List installed packages")
	fmt.Println("      install <name>         Install a package")
	fmt.Println("      check <name>           Check if package is installed")
	fmt.Println()
	fmt.Println("Module Management:")
	fmt.Println("  --modules <command>       Lua module management")
	fmt.Println("  --module-name <name>      Target module for info/test operations")
	fmt.Println("    Commands:")
	fmt.Println("      list                   List available modules")
	fmt.Println("      info <name>            Show module information")
	fmt.Println("      test <name>            Test module loading")
	fmt.Println()
	fmt.Println("Diagnostic Commands:")
	fmt.Println("  --doctor                  Run system diagnostics")
	fmt.Println("  --env-info                Show Python environment information")
	fmt.Println("  --verbose                 Enable verbose output")
	fmt.Println()
	fmt.Println("Environment Variables:")
	fmt.Println("  SUTERM_CONFIG            Path to configuration file")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  funterm                              Run REPL with default configuration")
	fmt.Println("  funterm script.su                    Run a script file")
	// fmt.Println("  funterm --exec \"lua.print('hello')\"  Execute a command string")
	// fmt.Println("  funterm --config config.yaml         Run with custom configuration")
	fmt.Println("  funterm --no-config                  Run without loading any config file")
	fmt.Println("  funterm --help                       Show this help message")
	fmt.Println("\nConfiguration:")
	fmt.Println("  Configuration files are searched in the following order:")
	fmt.Println("  1. Path specified by --config flag")
	fmt.Println("  2. Path specified by SUTERM_CONFIG environment variable")
	fmt.Println("  3. Default locations: ~/.funterm/config.yaml, ./config.yaml")
	fmt.Println()
	fmt.Println("For more information, visit: https://github.com/funvibe/funterm")
}

// handlePythonPackages handles Python package management commands
func handlePythonPackages(commandArg, targetFromFlag, configPath string, verbose bool) error {
	// Parse the command argument to extract command and target
	// Support formats: "list", "install <package>", "check <package>"
	parts := strings.Fields(commandArg)
	if len(parts) == 0 {
		return fmt.Errorf("no command specified")
	}

	command := parts[0]
	var target string
	if len(parts) > 1 {
		target = parts[1]
	} else {
		target = targetFromFlag
	}

	if verbose {
		fmt.Printf("Python package command: %s, target: %s\n", command, target)
	}

	// Initialize Python runtime
	registry := factory.DefaultRuntimeRegistry()
	pythonRuntime, err := registry.CreateRuntimeForLanguage("python")
	if err != nil {
		return fmt.Errorf("failed to create Python runtime: %w", err)
	}

	if err := pythonRuntime.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize Python runtime: %w", err)
	}

	// Type assertion to access Python-specific methods
	pyRuntime, ok := pythonRuntime.(*python.PythonRuntime)
	if !ok {
		return fmt.Errorf("failed to cast to Python runtime")
	}

	// Set verbose mode for the Python runtime
	pyRuntime.SetVerbose(verbose)

	switch command {
	case "list":
		if verbose {
			fmt.Println("Listing installed Python packages...")
		}
		packages := pyRuntime.ListPackages()
		if len(packages) == 0 {
			fmt.Println("No packages installed")
			return nil
		}

		fmt.Printf("%-30s %-15s %s\n", "Package", "Version", "Location")
		fmt.Println(strings.Repeat("-", 70))
		for _, pkg := range packages {
			fmt.Printf("%-30s %-15s %s\n", pkg.Name, pkg.Version, pkg.Location)
		}

	case "install":
		if target == "" {
			return fmt.Errorf("package name is required for install command")
		}
		if verbose {
			fmt.Printf("Installing Python package: %s\n", target)
		}
		if err := pyRuntime.InstallPackage(target, ""); err != nil {
			return fmt.Errorf("failed to install package '%s': %w", target, err)
		}
		fmt.Printf("Successfully installed package: %s\n", target)

	case "check":
		if target == "" {
			return fmt.Errorf("package name is required for check command")
		}
		if verbose {
			fmt.Printf("Checking Python package: %s\n", target)
		}
		version, err := pyRuntime.CheckPackage(target)
		if err != nil {
			return fmt.Errorf("package '%s' is not installed: %w", target, err)
		}
		fmt.Printf("Package '%s' is installed (version: %s)\n", target, version)

	default:
		return fmt.Errorf("unknown package command: %s. Supported commands: list, install, check", command)
	}

	return nil
}

// handleLuaModules handles Lua module management commands
func handleLuaModules(commandArg, targetFromFlag, configPath string, verbose bool) error {
	// Parse the command argument to extract command and target
	// Support formats: "list", "info <module>", "test <module>"
	parts := strings.Fields(commandArg)
	if len(parts) == 0 {
		return fmt.Errorf("no command specified")
	}

	command := parts[0]
	var target string
	if len(parts) > 1 {
		target = parts[1]
	} else {
		target = targetFromFlag
	}

	if verbose {
		fmt.Printf("Lua module command: %s, target: %s\n", command, target)
	}

	// Initialize Lua runtime
	registry := factory.DefaultRuntimeRegistry()
	luaRuntime, err := registry.CreateRuntimeForLanguage("lua")
	if err != nil {
		return fmt.Errorf("failed to create Lua runtime: %w", err)
	}

	if err := luaRuntime.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize Lua runtime: %w", err)
	}

	switch command {
	case "list":
		if verbose {
			fmt.Println("Listing available Lua modules...")
		}
		modules := luaRuntime.GetModules()
		if len(modules) == 0 {
			fmt.Println("No modules available")
			return nil
		}

		fmt.Printf("%-20s %s\n", "Module", "Functions")
		fmt.Println(strings.Repeat("-", 50))
		for _, module := range modules {
			functions := luaRuntime.GetModuleFunctions(module)
			funcCount := len(functions)
			if funcCount > 5 {
				fmt.Printf("%-20s %d functions (showing first 5)\n", module, funcCount)
				for i := 0; i < 5 && i < len(functions); i++ {
					fmt.Printf("  - %s\n", functions[i])
				}
			} else {
				fmt.Printf("%-20s %d functions\n", module, funcCount)
				for _, fn := range functions {
					fmt.Printf("  - %s\n", fn)
				}
			}
			fmt.Println()
		}

	case "info":
		if target == "" {
			return fmt.Errorf("module name is required for info command")
		}
		if verbose {
			fmt.Printf("Showing info for Lua module: %s\n", target)
		}

		functions := luaRuntime.GetModuleFunctions(target)
		if len(functions) == 0 {
			return fmt.Errorf("module '%s' not found or has no functions", target)
		}

		fmt.Printf("Module: %s\n", target)
		fmt.Printf("Functions (%d):\n", len(functions))
		for _, fn := range functions {
			signature, err := luaRuntime.GetFunctionSignature(target, fn)
			if err != nil {
				fmt.Printf("  - %s\n", fn)
			} else {
				fmt.Printf("  - %s\n", signature)
			}
		}

	case "test":
		if target == "" {
			return fmt.Errorf("module name is required for test command")
		}
		if verbose {
			fmt.Printf("Testing Lua module: %s\n", target)
		}

		// Test if module can be loaded
		testCode := fmt.Sprintf("local %s = require('%s'); print('Module %s loaded successfully')", target, target, target)
		if err := luaRuntime.ExecuteBatch(testCode); err != nil {
			return fmt.Errorf("failed to test module '%s': %w", target, err)
		}

	default:
		return fmt.Errorf("unknown module command: %s. Supported commands: list, info, test", command)
	}

	return nil
}

// runDiagnostics runs system diagnostics
func runDiagnostics(configPath string, verbose bool) error {
	if verbose {
		fmt.Println("Running system diagnostics...")
	}

	fmt.Println("=== Funterm System Diagnostics ===")
	fmt.Println()

	// Check Python runtime
	fmt.Println("1. Python Runtime:")
	registry := factory.DefaultRuntimeRegistry()
	pythonRuntime, err := registry.CreateRuntimeForLanguage("python")
	if err != nil {
		fmt.Printf("   ❌ Failed to create Python runtime: %v\n", err)
	} else {
		if err := pythonRuntime.Initialize(); err != nil {
			fmt.Printf("   ❌ Failed to initialize Python runtime: %v\n", err)
		} else {
			fmt.Printf("   ✅ Python runtime initialized successfully\n")
			if pyRuntime, ok := pythonRuntime.(*python.PythonRuntime); ok {
				pyRuntime.SetVerbose(verbose)
				// Virtual environment support disabled in simplified runtime
				fmt.Printf("   ⚠️  Virtual environment support disabled (simplified runtime)\n")

				packages := pyRuntime.ListPackages()
				fmt.Printf("   📦 Installed packages: %d\n", len(packages))
			}
		}
	}
	fmt.Println()

	// Check Lua runtime
	fmt.Println("2. Lua Runtime:")
	luaRuntime, err := registry.CreateRuntimeForLanguage("lua")
	if err != nil {
		fmt.Printf("   ❌ Failed to create Lua runtime: %v\n", err)
	} else {
		if err := luaRuntime.Initialize(); err != nil {
			fmt.Printf("   ❌ Failed to initialize Lua runtime: %v\n", err)
		} else {
			fmt.Printf("   ✅ Lua runtime initialized successfully\n")
			modules := luaRuntime.GetModules()
			fmt.Printf("   📚 Available modules: %d\n", len(modules))
		}
	}
	fmt.Println()

	// Check configuration
	fmt.Println("3. Configuration:")
	if configPath == "" {
		fmt.Printf("   ℹ️  Using default configuration\n")
	} else {
		if _, err := os.Stat(configPath); err != nil {
			fmt.Printf("   ❌ Configuration file not found: %v\n", err)
		} else {
			fmt.Printf("   ✅ Configuration file found: %s\n", configPath)
		}
	}
	fmt.Println()

	// Check working directory
	fmt.Println("4. Working Directory:")
	workDir, err := os.Getwd()
	if err != nil {
		fmt.Printf("   ❌ Failed to get working directory: %v\n", err)
	} else {
		fmt.Printf("   ✅ Working directory: %s\n", workDir)
	}
	fmt.Println()

	fmt.Println("=== Diagnostics Complete ===")
	return nil
}

// showPythonEnvironmentInfo shows Python environment information
func showPythonEnvironmentInfo(configPath string, verbose bool) error {
	if verbose {
		fmt.Println("Showing Python environment information...")
	}

	fmt.Println("=== Python Environment Information ===")
	fmt.Println()

	// Initialize Python runtime
	registry := factory.DefaultRuntimeRegistry()
	pythonRuntime, err := registry.CreateRuntimeForLanguage("python")
	if err != nil {
		return fmt.Errorf("failed to create Python runtime: %w", err)
	}

	if err := pythonRuntime.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize Python runtime: %w", err)
	}

	pyRuntime, ok := pythonRuntime.(*python.PythonRuntime)
	if !ok {
		return fmt.Errorf("failed to cast to Python runtime")
	}

	// Set verbose mode for the Python runtime
	pyRuntime.SetVerbose(verbose)

	// Python version
	fmt.Println("1. Python Version:")
	if result, err := pyRuntime.ExecuteFunction("sys.version", []interface{}{}); err == nil {
		if versionStr, ok := result.(string); ok {
			fmt.Printf("   %s\n", strings.TrimSpace(versionStr))
		}
	}
	fmt.Println()

	// Virtual environment information
	fmt.Println("2. Virtual Environment:")
	fmt.Printf("   ⚠️  Virtual environment support disabled (simplified runtime)\n")
	fmt.Println()

	// Package manager information
	fmt.Println("3. Package Manager:")
	if pyRuntime.GetPackageManager() != nil {
		fmt.Printf("   ✅ Package manager available\n")
		packages := pyRuntime.ListPackages()
		fmt.Printf("   📦 Total installed packages: %d\n", len(packages))

		if verbose {
			fmt.Println("   Installed packages:")
			for _, pkg := range packages {
				fmt.Printf("      - %s (%s)\n", pkg.Name, pkg.Version)
			}
		}
	} else {
		fmt.Printf("   ❌ Package manager not available\n")
	}
	fmt.Println()

	// Python path
	fmt.Println("4. Python Paths:")
	if paths, err := pyRuntime.ExecuteFunction("sys.path", []interface{}{}); err == nil {
		if pathList, ok := paths.([]interface{}); ok {
			for i, path := range pathList {
				if pathStr, ok := path.(string); ok {
					fmt.Printf("   %d. %s\n", i+1, pathStr)
				}
			}
		}
	}

	fmt.Println("=== Environment Information Complete ===")
	return nil
}
