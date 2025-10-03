package repl

import (
	"fmt"
)

// DisplayManager manages visual indicators and formatting for the REPL
type DisplayManager struct {
	useColors bool
	width     int
	verbose   bool
}

// NewDisplayManager creates a new display manager
func NewDisplayManager(useColors, verbose bool) *DisplayManager {
	return &DisplayManager{
		useColors: useColors,
		width:     80, // Standard terminal width
		verbose:   verbose,
	}
}

// GetPrompt returns the appropriate prompt based on buffer state
func (dm *DisplayManager) GetPrompt(buffer *MultiLineBuffer, inputReader interface{}) string {
	var isMultiline bool

	// Check if input reader is in multiline mode
	if ir, ok := inputReader.(interface{ IsMultiline() bool }); ok {
		isMultiline = ir.IsMultiline()
	} else if sir, ok := inputReader.(interface{ IsMultiline() bool }); ok {
		isMultiline = sir.IsMultiline()
	}

	if isMultiline || buffer.IsActive() {
		return dm.formatPrompt("... ", "continuation")
	}
	return dm.formatPrompt(">>> ", "primary")
}

// formatPrompt formats a prompt with optional colors
func (dm *DisplayManager) formatPrompt(text, promptType string) string {
	if !dm.useColors {
		switch promptType {
		case "continuation":
			return "... "
		default:
			return text
		}
	}

	// ANSI color codes
	colors := map[string]string{
		"primary":      "\033[36m", // Cyan
		"continuation": "\033[90m", // Dark gray
		"reset":        "\033[0m",
		"success":      "\033[32m", // Green
		"error":        "\033[31m", // Red
		"warning":      "\033[33m", // Yellow
		"info":         "\033[34m", // Blue
	}

	color, ok := colors[promptType]
	if !ok {
		color = colors["primary"]
	}

	return fmt.Sprintf("%s%s%s", color, text, colors["reset"])
}

// ShowBufferStatus displays the current buffer status
func (dm *DisplayManager) ShowBufferStatus(buffer *MultiLineBuffer) {
	if !buffer.IsActive() || buffer.IsEmpty() {
		return
	}

	fmt.Printf("\n%s--- Многострочный режим (%d строк) ---%s\n",
		dm.formatPrompt("", "continuation"),
		buffer.GetLineCount(),
		dm.formatPrompt("", "reset"))
}

// ShowExecutionResult displays the result of command execution
func (dm *DisplayManager) ShowExecutionResult(result interface{}, isPrint bool, err error) {
	if err != nil {
		fmt.Printf("%sОшибка: %v%s\n", dm.formatPrompt("", "error"), err, dm.formatPrompt("", "reset"))
		return
	}

	if result != nil {
		if str, ok := result.(string); ok {
			fmt.Println(str)
		} else {
			fmt.Printf("%v\n", result)
		}
	} else if !isPrint {
		fmt.Printf("%s✓ Executed%s\n", dm.formatPrompt("", "success"), dm.formatPrompt("", "reset"))
	}
}

// ShowHelp displays help information for multiline mode
func (dm *DisplayManager) ShowHelp() {
	helpText := `
%sМногострочный режим Funterm REPL:%s

%sКлавиши:%s
  Enter        - добавить строку в буфер (пустая строка выполняет буфер)
  Ctrl+C       - сбросить буфер или прервать выполнение
  Ctrl+D       - выйти из REPL

%sКоманды:%s
  .reset       - сбросить буфер
  .buffer      - показать содержимое буфера
  .multiline   - начать многострочный режим
  .help        - эта справка
  .clear       - очистить экран
  .exit        - выйти из REPL

%sПримеры:%s
  >>> python.def add(a, b):%s
  ...     return a + b%s
  ... %s
  ✓ Executed
  
  >>> python.add(2, 3)
  5

  >>> for item in py.numbers:%s
  ...     py.total = py.total + item%s
  ... %s
  ✓ Executed

%sСоветы:%s
  • Используйте .multiline для начала многострочного режима
  • В многострочном режиме все строки добавляются в буфер
  • Пустая строка завершает многострочный режим и выполняет код
  • Используйте .reset чтобы очистить буфер без выполнения
  • Поддерживаются все языки: Python, Lua, JavaScript, Go

`

	fmt.Printf(helpText,
		dm.formatPrompt("", "info"), dm.formatPrompt("", "reset"),
		dm.formatPrompt("", "primary"), dm.formatPrompt("", "reset"),
		dm.formatPrompt("", "primary"), dm.formatPrompt("", "reset"),
		dm.formatPrompt("", "primary"), dm.formatPrompt("", "reset"),
		dm.formatPrompt("", "continuation"), dm.formatPrompt("", "continuation"),
		dm.formatPrompt("", "continuation"),
		dm.formatPrompt("", "continuation"), dm.formatPrompt("", "continuation"),
		dm.formatPrompt("", "primary"), dm.formatPrompt("", "reset"),
	)
}

// ShowBufferContent displays the content of the buffer
func (dm *DisplayManager) ShowBufferContent(buffer *MultiLineBuffer) {
	if buffer.IsActive() && !buffer.IsEmpty() {
		fmt.Printf("%sБуфер содержит %d строк:%s\n",
			dm.formatPrompt("", "info"), buffer.GetLineCount(), dm.formatPrompt("", "reset"))
		for i, line := range buffer.lines {
			fmt.Printf("%s%3d:%s %s\n",
				dm.formatPrompt("", "continuation"),
				i+1,
				dm.formatPrompt("", "reset"),
				line)
		}
	} else {
		fmt.Printf("%sБуфер пуст%s\n", dm.formatPrompt("", "info"), dm.formatPrompt("", "reset"))
	}
}

// ShowWelcome displays the welcome message
func (dm *DisplayManager) ShowWelcome() {
	welcomeText := `
%sДобро пожаловать в Funterm - Мультиязыковой REPL%s

%sДоступные языки:%s go, js, lua, python, py, node

%sОсновные команды:%s
  :help        - показать эту справку
  :quit        - выйти из REPL
  :languages   - показать доступные языки
  :run <file>  - выполнить файл со смешанным кодом

%sМногострочный режим:%s
  • Используйте .multiline для начала многострочного режима
  • Поддержка функций, циклов, условий
  • Введите .help для подробной справки

%sНачните работу!%s
`

	fmt.Printf(welcomeText,
		dm.formatPrompt("", "success"), dm.formatPrompt("", "reset"),
		dm.formatPrompt("", "primary"), dm.formatPrompt("", "reset"),
		dm.formatPrompt("", "primary"), dm.formatPrompt("", "reset"),
		dm.formatPrompt("", "info"), dm.formatPrompt("", "reset"),
		dm.formatPrompt("", "warning"), dm.formatPrompt("", "reset"),
	)
}

// ShowError displays an error message
func (dm *DisplayManager) ShowError(message string) {
	fmt.Printf("%sОшибка: %s%s\n", dm.formatPrompt("", "error"), message, dm.formatPrompt("", "reset"))
}

// ShowWarning displays a warning message
func (dm *DisplayManager) ShowWarning(message string) {
	fmt.Printf("%sПредупреждение: %s%s\n", dm.formatPrompt("", "warning"), message, dm.formatPrompt("", "reset"))
}

// ShowInfo displays an info message
func (dm *DisplayManager) ShowInfo(message string) {
	fmt.Printf("%sИнформация: %s%s\n", dm.formatPrompt("", "info"), message, dm.formatPrompt("", "reset"))
}

// ShowSuccess displays a success message
func (dm *DisplayManager) ShowSuccess(message string) {
	fmt.Printf("%s✓ %s%s\n", dm.formatPrompt("", "success"), message, dm.formatPrompt("", "reset"))
}

// FormatResult formats the result for display
func (dm *DisplayManager) FormatResult(result interface{}) string {
	switch v := result.(type) {
	case string:
		return fmt.Sprintf("\"%s\"", v)
	case fmt.Stringer:
		return v.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}

// ClearScreen clears the terminal screen
func (dm *DisplayManager) ClearScreen() {
	fmt.Print("\033[H\033[2J") // ANSI escape codes to clear screen
}

// SetWidth sets the terminal width for formatting
func (dm *DisplayManager) SetWidth(width int) {
	dm.width = width
}

// SetColors enables or disables color output
func (dm *DisplayManager) SetColors(enabled bool) {
	dm.useColors = enabled
}

// IsColorEnabled returns whether color output is enabled
func (dm *DisplayManager) IsColorEnabled() bool {
	return dm.useColors
}

// ShowVersion displays version information
func (dm *DisplayManager) ShowVersion() {
	fmt.Printf("%sFunterm v0.1.0 - Мультиязыковой REPL с поддержкой многострочного ввода%s\n",
		dm.formatPrompt("", "success"), dm.formatPrompt("", "reset"))
}

// ShowAvailableLanguages displays available languages
func (dm *DisplayManager) ShowAvailableLanguages(languages []string) {
	if len(languages) == 0 {
		fmt.Printf("%sНет доступных языков%s\n", dm.formatPrompt("", "warning"), dm.formatPrompt("", "reset"))
		return
	}

	fmt.Printf("%sДоступные языки:%s\n", dm.formatPrompt("", "primary"), dm.formatPrompt("", "reset"))
	for _, lang := range languages {
		fmt.Printf("  %s• %s%s\n", dm.formatPrompt("", "success"), lang, dm.formatPrompt("", "reset"))
	}
}

// ShowHistory displays command history
func (dm *DisplayManager) ShowHistory(history []string) {
	if len(history) == 0 {
		fmt.Printf("%sНет истории команд%s\n", dm.formatPrompt("", "info"), dm.formatPrompt("", "reset"))
		return
	}

	fmt.Printf("%sИстория команд:%s\n", dm.formatPrompt("", "primary"), dm.formatPrompt("", "reset"))
	for i, cmd := range history {
		fmt.Printf("%s%3d:%s %s\n",
			dm.formatPrompt("", "continuation"),
			i+1,
			dm.formatPrompt("", "reset"),
			cmd)
	}
}

// ShowMultilineIndicator shows a visual indicator when entering multiline mode
func (dm *DisplayManager) ShowMultilineIndicator() {
	if dm.verbose {
		fmt.Printf("%s--- Вход в многострочный режим ---%s\n",
			dm.formatPrompt("", "info"), dm.formatPrompt("", "reset"))
	}
}

// ShowExitMultilineIndicator shows a visual indicator when exiting multiline mode
func (dm *DisplayManager) ShowExitMultilineIndicator() {
	if dm.verbose {
		fmt.Printf("%s--- Выход из многострочного режима ---%s\n",
			dm.formatPrompt("", "info"), dm.formatPrompt("", "reset"))
	}
}
