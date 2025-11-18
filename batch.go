package main

import (
	"fmt"
	"funterm/factory"
	"funterm/repl"
	"funterm/shared"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// BatchMode выполняет файл в пакетном режиме (без интерактивного REPL)
func BatchMode(filePath string, language string, configPath string, verbose bool) error {
	// Load configuration
	cfg, err := LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("ошибка загрузки конфигурации: %v", err)
	}

	// Override config with command line flags
	if verbose {
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

	// Отключаем приветственное сообщение в пакетном режиме
	replInstance.SetWelcomeMessage(false)

	// Инициализируем рантаймы
	if err := replInstance.GetEngine().InitializeRuntimes(); err != nil {
		return fmt.Errorf("ошибка инициализации рантаймов: %v", err)
	}

	// Дополнительно вызываем метод инициализации из REPL
	if err := replInstance.InitializeRuntimes(); err != nil {
		return fmt.Errorf("ошибка инициализации рантаймов REPL: %v", err)
	}

	// Определяем тип файла по расширению, если язык не указан
	if language == "" {
		ext := strings.ToLower(filepath.Ext(filePath))
		switch ext {
		case ".lua":
			language = "lua"
		case ".py":
			language = "python"
		case ".su":
			// Смешанный файл
			return executeMixedFile(replInstance, filePath, verbose)
		default:
			return fmt.Errorf("не удалось определить язык по расширению файла: %s", ext)
		}
	}

	// Выполняем файл
	if language == "mixed" {
		return executeMixedFile(replInstance, filePath, verbose)
	} else {
		return executeFile(replInstance, language, filePath)
	}
}

// executeFile выполняет файл на указанном языке
func executeFile(r *repl.REPL, language, filePath string) error {
	// Проверяем, доступен ли язык
	if !r.GetEngine().IsLanguageAvailable(language) {
		return fmt.Errorf("язык '%s' недоступен", language)
	}

	// Читаем содержимое файла
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("ошибка чтения файла: %v", err)
	}

	// Получаем рантайм для языка
	runtime, err := r.GetEngine().GetRuntimeManager().GetRuntime(language)
	if err != nil {
		return fmt.Errorf("рантайм для языка '%s' не найден: %v", language, err)
	}

	// Выполняем весь файл сразу через метод ExecuteBatch для корректного вывода
	err = runtime.ExecuteBatch(string(content))
	if err != nil {
		return fmt.Errorf("script execution error: %v", err)
	}

	return nil
}

// executeMixedFile выполняет смешанный файл
func executeMixedFile(r *repl.REPL, filePath string, verbose bool) error {
	// Читаем содержимое файла
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("ошибка чтения файла: %v", err)
	}

	fileContent := string(content)

	if verbose {
		fmt.Printf("Executing mixed language file: %s (%d characters)\n", filePath, len(fileContent))
	}

	// Выполняем весь файл как единое целое через ExecutionEngine
	// Это позволяет правильно обрабатывать многострочные конструкции как блоки кода
	result, _, _, err := r.GetEngine().Execute(fileContent)
	if err != nil {
		return fmt.Errorf("script execution error: %v", err)
	}

	// Выводим результат выполнения, если он не пустой
	if result != nil && result != "" {
		if preFormatted, ok := result.(*shared.PreFormattedResult); ok {
			fmt.Printf("=> %s\n", preFormatted.Value)
		} else {
			fmt.Printf("=> %v\n", result)
		}
	}

	if verbose {
		fmt.Println("Mixed file executed successfully")
	}
	return nil
}
