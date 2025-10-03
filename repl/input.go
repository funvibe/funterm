package repl

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/chzyer/readline"
)

// InputReader handles input with multiline support
type InputReader struct {
	rl         *readline.Instance
	multiline  bool
	prompt     string
	contPrompt string
}

// NewInputReader creates a new input reader with multiline support
func NewInputReader(prompt, contPrompt string) (*InputReader, error) {
	rl, err := readline.NewEx(&readline.Config{
		Prompt:          prompt,
		HistoryFile:     "/tmp/funterm_history",
		HistoryLimit:    1000,
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
		AutoComplete:    nil,
	})
	if err != nil {
		return nil, err
	}

	return &InputReader{
		rl:         rl,
		prompt:     prompt,
		contPrompt: contPrompt,
		multiline:  false,
	}, nil
}

// ReadLine reads a single line or multiple lines in multiline mode
func (ir *InputReader) ReadLine() (string, bool, error) {
	if !ir.multiline {
		// Single line mode
		ir.rl.SetPrompt(ir.prompt)
		line, err := ir.rl.Readline()
		if err != nil {
			return "", false, err
		}

		// Check if this line should start multiline mode
		if ir.shouldStartMultiline(line) {
			ir.multiline = true
			return line, true, nil
		}

		return line, false, nil
	} else {
		// Multiline mode - use simple readline to avoid history issues
		// We'll use the same readline instance but handle multiline separately
		ir.rl.SetPrompt(ir.contPrompt)
		line, err := ir.rl.Readline()
		if err != nil {
			// End multiline mode on error or EOF
			ir.multiline = false
			return "", false, err
		}

		// Check if this line should end multiline mode
		if ir.shouldEndMultiline(line) {
			ir.multiline = false
			return line, false, nil
		}

		return line, true, nil
	}
}

// AddHistory adds a line to the history manually
func (ir *InputReader) AddHistory(line string) {
	// Use SaveHistory method if available, otherwise this is a no-op
	// The readline library handles history automatically for single-line input
}

// GetHistory returns the current history
func (ir *InputReader) GetHistory() []string {
	// This would require accessing the internal history, which may not be available
	// Return empty slice for now - REPL manages its own history
	return []string{}
}

// shouldStartMultiline determines if a line should start multiline mode
func (ir *InputReader) shouldStartMultiline(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return false
	}

	// Check for incomplete syntax that suggests multiline input
	return ir.hasIncompleteSyntax(trimmed)
}

// shouldEndMultiline determines if a line should end multiline mode
func (ir *InputReader) shouldEndMultiline(line string) bool {
	trimmed := strings.TrimSpace(line)

	// Empty line ends multiline mode
	if trimmed == "" {
		return true
	}

	// Check if the complete multiline input is syntactically complete
	return false // Let the engine decide during execution
}

// hasIncompleteSyntax checks if the line has incomplete syntax
func (ir *InputReader) hasIncompleteSyntax(line string) bool {
	// Check for unclosed brackets, parentheses, braces
	if strings.Count(line, "(") > strings.Count(line, ")") {
		return true
	}
	if strings.Count(line, "[") > strings.Count(line, "]") {
		return true
	}
	if strings.Count(line, "{") > strings.Count(line, "}") {
		return true
	}

	// Check for trailing operators that suggest continuation
	operators := []string{":", "\\", ",", "+", "-", "*", "/", "|", "&", "or", "and", "not"}
	for _, op := range operators {
		if strings.HasSuffix(line, op) {
			return true
		}
	}

	// Check for language-specific patterns that suggest multiline input
	// Python: def, class, if, for, while, try, with, followed by :
	if strings.HasSuffix(line, ":") {
		keywords := []string{"def ", "class ", "if ", "for ", "while ", "try ", "with ", "elif ", "else"}
		for _, keyword := range keywords {
			if strings.Contains(line, keyword) {
				return true
			}
		}
	}

	// Lua: function, if, for, while, repeat, do
	luaKeywords := []string{"function ", "if ", "for ", "while ", "repeat ", "do "}
	for _, keyword := range luaKeywords {
		if strings.Contains(line, keyword) && !strings.Contains(line, "end") {
			return true
		}
	}

	// JavaScript: function, if, for, while, try, switch
	jsKeywords := []string{"function ", "if ", "for ", "while ", "try ", "switch "}
	for _, keyword := range jsKeywords {
		if strings.Contains(line, keyword) && !strings.HasSuffix(line, "{") {
			return true
		}
	}

	return false
}

// SetPrompt sets the main prompt
func (ir *InputReader) SetPrompt(prompt string) {
	ir.prompt = prompt
	if !ir.multiline {
		ir.rl.SetPrompt(prompt)
	}
}

// SetContPrompt sets the continuation prompt
func (ir *InputReader) SetContPrompt(prompt string) {
	ir.contPrompt = prompt
	if ir.multiline {
		ir.rl.SetPrompt(prompt)
	}
}

// IsMultiline returns whether we're currently in multiline mode
func (ir *InputReader) IsMultiline() bool {
	return ir.multiline
}

// ExitMultiline exits multiline mode
func (ir *InputReader) ExitMultiline() {
	ir.multiline = false
	ir.rl.SetPrompt(ir.prompt)
}

// Close closes the input reader
func (ir *InputReader) Close() error {
	return ir.rl.Close()
}

// SimpleInputReader is a fallback implementation without external dependencies
type SimpleInputReader struct {
	reader     *bufio.Reader
	multiline  bool
	prompt     string
	contPrompt string
}

// NewSimpleInputReader creates a simple input reader
func NewSimpleInputReader(prompt, contPrompt string) *SimpleInputReader {
	return &SimpleInputReader{
		reader:     bufio.NewReader(os.Stdin),
		prompt:     prompt,
		contPrompt: contPrompt,
		multiline:  false,
	}
}

// ReadLine reads a line with simple multiline detection
func (sir *SimpleInputReader) ReadLine() (string, bool, error) {
	prompt := sir.prompt
	if sir.multiline {
		prompt = sir.contPrompt
	}

	fmt.Print(prompt)

	line, err := sir.reader.ReadString('\n')
	if err != nil {
		return "", false, err
	}

	line = strings.TrimSuffix(line, "\n")

	if !sir.multiline {
		// Check if we should start multiline mode
		if sir.shouldStartMultilineSimple(line) {
			sir.multiline = true
			return line, true, nil
		}
		return line, false, nil
	} else {
		// Check if we should end multiline mode
		if sir.shouldEndMultilineSimple(line) {
			sir.multiline = false
			return line, false, nil
		}
		return line, true, nil
	}
}

// shouldStartMultilineSimple is a simplified version for the simple reader
func (sir *SimpleInputReader) shouldStartMultilineSimple(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return false
	}

	// Simple heuristic: if line ends with certain characters, continue
	trailingChars := []string{":", "\\", ",", "(", "[", "{"}
	for _, char := range trailingChars {
		if strings.HasSuffix(trimmed, char) {
			return true
		}
	}

	return false
}

// shouldEndMultilineSimple is a simplified version for the simple reader
func (sir *SimpleInputReader) shouldEndMultilineSimple(line string) bool {
	// Empty line ends multiline mode
	return strings.TrimSpace(line) == ""
}

// SetPrompt sets the main prompt
func (sir *SimpleInputReader) SetPrompt(prompt string) {
	sir.prompt = prompt
}

// SetContPrompt sets the continuation prompt
func (sir *SimpleInputReader) SetContPrompt(prompt string) {
	sir.contPrompt = prompt
}

// IsMultiline returns whether we're currently in multiline mode
func (sir *SimpleInputReader) IsMultiline() bool {
	return sir.multiline
}

// ExitMultiline exits multiline mode
func (sir *SimpleInputReader) ExitMultiline() {
	sir.multiline = false
}

// Errors
var (
	ErrInterrupted = errors.New("прервано пользователем")
	ErrReset       = errors.New("буфер сброшен")
	ErrEOF         = errors.New("конец файла")
)
