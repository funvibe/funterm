package python

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// sendAndAwait is the new core method for all communication with the Python REPL.
func (pr *PythonRuntime) sendAndAwait(code string) (string, error) {
	// Thread-safe access to Python process - use separate mutex to prevent race conditions
	pr.processMutex.Lock()
	defer pr.processMutex.Unlock()

	// Generate a unique execution ID for this call
	executionID++
	currentExecutionID := executionID
	uniqueMarker := fmt.Sprintf("%s-%d", EndOfOutputMarker, currentExecutionID)

	if pr.verbose {
		fmt.Printf("DEBUG: sendAndAwait execution ID: %d, unique marker: %s\n", currentExecutionID, uniqueMarker)
	}

	// Clear any pending results from previous executions to avoid race conditions
	for len(pr.resultChan) > 0 {
		<-pr.resultChan
	}
	for len(pr.errorChan) > 0 {
		<-pr.errorChan
	}

	// Ensure the code block is treated as a single unit by the REPL.
	// We use compile() for multi-line code to ensure it's executed atomically.
	// For simple, single-line code, we can send it directly.
	var finalCode string
	var markerCmd string

	// Check if this is multiline Python code that needs special handling
	isMultilinePython := strings.Contains(code, "\n") ||
		strings.HasPrefix(strings.TrimSpace(code), "def ") ||
		strings.HasPrefix(strings.TrimSpace(code), "class ") ||
		strings.HasPrefix(strings.TrimSpace(code), "if ") && strings.Contains(code, ":") ||
		strings.HasPrefix(strings.TrimSpace(code), "for ") && strings.Contains(code, ":") ||
		strings.HasPrefix(strings.TrimSpace(code), "while ") && strings.Contains(code, ":") ||
		strings.HasPrefix(strings.TrimSpace(code), "try:") ||
		strings.HasPrefix(strings.TrimSpace(code), "with ")

	if isMultilinePython {
		// For multiline code, wrap it in exec() and include the marker in the same exec() call
		// This ensures both the code and marker are executed atomically
		if pr.verbose {
			fmt.Printf("DEBUG: Original code before dedent:\n%q\n", code)
		}

		// First apply dedent to fix basic indentation issues
		dedentedCode := dedent(code)
		if pr.verbose {
			fmt.Printf("DEBUG: Code after dedent:\n%q\n", dedentedCode)
		}

		// Try to format the code using ast module to fix any remaining indentation issues
		formattedCode, err := formatPythonCode(dedentedCode)
		if err != nil {
			if pr.verbose {
				fmt.Printf("DEBUG: Failed to format Python code, using dedented code: %v\n", err)
			}
			// If formatting fails, use the dedented code
			formattedCode = dedentedCode
		} else if pr.verbose {
			fmt.Printf("DEBUG: Code after formatting:\n%q\n", formattedCode)
		}

		wrappedCode := fmt.Sprintf("%s\nprint('%s')", formattedCode, uniqueMarker)
		finalCode = fmt.Sprintf("exec(%s)", strconv.Quote(wrappedCode))
		markerCmd = "" // Marker is already included in finalCode
		if pr.verbose {
			fmt.Printf("DEBUG: Wrapped code:\n%q\n", wrappedCode)
		}
	} else {
		// For single-line code, send it normally
		if pr.verbose {
			fmt.Printf("DEBUG: Final code to execute:\n%s\n", finalCode)
		}
		finalCode = code
		markerCmd = fmt.Sprintf("print('%s')\n", uniqueMarker)
	}

	// Ensure the code ends with a newline to be executed by the REPL.
	if !strings.HasSuffix(finalCode, "\n") {
		finalCode += "\n"
	}

	// Send the command (marker is already included for multiline code)
	if _, err := fmt.Fprint(pr.stdin, finalCode); err != nil {
		return "", fmt.Errorf("failed to write code to python stdin: %w", err)
	}

	// For single-line code without marker, send it separately
	if markerCmd != "" {
		if _, err := fmt.Fprint(pr.stdin, markerCmd); err != nil {
			return "", fmt.Errorf("failed to write marker to python stdin: %w", err)
		}
	}

	// Absolutely final robust error handling logic.
	// Collect all outputs until the result marker is received, then decide.
	var stdoutResult string
	var stderrResult strings.Builder
	var resultReceived bool

	timeout := time.After(pr.executionTimeout)

	for !resultReceived {
		select {
		case result := <-pr.resultChan:
			if pr.verbose {
				fmt.Printf("DEBUG: sendAndAwait received result from resultChan: '%s'\n", result)
			}
			// Only accept results that are meant for this execution
			// We'll check for the unique marker in the readOutput method
			stdoutResult = result
			resultReceived = true // Marker received, we can stop the main loop.
		case err := <-pr.errorChan:
			if pr.verbose {
				fmt.Printf("DEBUG: sendAndAwait received error from errorChan: '%s'\n", err.Error())
			}
			stderrResult.WriteString(err.Error())

			// Check if this is a Python traceback (exception) that should terminate execution
			errorStr := err.Error()
			if strings.Contains(errorStr, "Traceback (most recent call last):") ||
				strings.Contains(errorStr, "SyntaxError:") ||
				strings.Contains(errorStr, "ValueError:") ||
				strings.Contains(errorStr, "TypeError:") ||
				strings.Contains(errorStr, "NameError:") ||
				strings.Contains(errorStr, "IndexError:") ||
				strings.Contains(errorStr, "KeyError:") ||
				strings.Contains(errorStr, "AttributeError:") {
				// This is a Python exception, we should stop waiting for result and return error
				if pr.verbose {
					fmt.Printf("DEBUG: sendAndAwait detected Python exception, terminating execution\n")
				}
				resultReceived = true
				continue
			}
		case <-timeout:
			if pr.cmd != nil && pr.cmd.Process != nil {
				_ = pr.cmd.Process.Kill()
			}
			// If we timed out but got some error message, return that.
			if stderrResult.Len() > 0 {
				return "", fmt.Errorf("%s", stderrResult.String())
			}
			return "", fmt.Errorf("python execution timed out after %v", pr.executionTimeout)
		}
	}

	if pr.verbose {
		fmt.Printf("DEBUG: sendAndAwait final stdoutResult before trimming: '%s'\n", stdoutResult)
		fmt.Printf("DEBUG: sendAndAwait stderrResult: '%s'\n", stderrResult.String())
	}

	// After receiving the result, do a final short drain on stderr to catch trailing error messages.
	drainTimeout := time.After(50 * time.Millisecond)
	draining := true
	for draining {
		select {
		case err := <-pr.errorChan:
			stderrResult.WriteString(err.Error())
		case <-drainTimeout:
			draining = false
		}
	}

	// Final decision: if a real error occurred, stderr will contain a traceback.
	errorString := stderrResult.String()
	if pr.verbose {
		fmt.Printf("DEBUG: sendAndAwait final errorString: '%s'\n", errorString)
	}
	if strings.Contains(errorString, "Traceback (most recent call last):") || strings.Contains(errorString, "SyntaxError:") {
		return "", fmt.Errorf("%s", errorString)
	}

	trimmedResult := strings.TrimSpace(stdoutResult)
	if pr.verbose {
		fmt.Printf("DEBUG: sendAndAwait returning trimmed result: '%s'\n", trimmedResult)
	}
	return trimmedResult, nil
}

// sendAndAwaitWithID is the new core method for all communication with the Python REPL.
func (pr *PythonRuntime) sendAndAwaitWithID(code string, execID int64) (string, error) {
	// Thread-safe access to Python process - use separate mutex to prevent race conditions
	pr.processMutex.Lock()
	defer pr.processMutex.Unlock()

	currentExecutionID := execID
	uniqueMarker := fmt.Sprintf("%s-%d", EndOfOutputMarker, currentExecutionID)

	if pr.verbose {
		fmt.Printf("DEBUG: sendAndAwaitWithID execution ID: %d, unique marker: %s\n", currentExecutionID, uniqueMarker)
	}

	// Clear any pending results from previous executions to avoid race conditions
	for len(pr.resultChan) > 0 {
		<-pr.resultChan
	}
	for len(pr.errorChan) > 0 {
		<-pr.errorChan
	}

	// Ensure the code block is treated as a single unit by the REPL.
	// We use compile() for multi-line code to ensure it's executed atomically.
	// For simple, single-line code, we can send it directly.
	var finalCode string
	var markerCmd string

	// Check if this is multiline Python code that needs special handling
	isMultilinePython := strings.Contains(code, "\n") ||
		strings.HasPrefix(strings.TrimSpace(code), "def ") ||
		strings.HasPrefix(strings.TrimSpace(code), "class ") ||
		strings.HasPrefix(strings.TrimSpace(code), "if ") && strings.Contains(code, ":") ||
		strings.HasPrefix(strings.TrimSpace(code), "for ") && strings.Contains(code, ":") ||
		strings.HasPrefix(strings.TrimSpace(code), "while ") && strings.Contains(code, ":") ||
		strings.HasPrefix(strings.TrimSpace(code), "try:") ||
		strings.HasPrefix(strings.TrimSpace(code), "with ")

	if isMultilinePython {
		// For multiline code, wrap it in exec() and include the marker in the same exec() call
		// This ensures both the code and marker are executed atomically

		// First apply dedent to fix basic indentation issues
		dedentedCode := dedent(code)
		if pr.verbose {
			fmt.Printf("DEBUG: Code after dedent:\n%q\n", dedentedCode)
		}

		// Then format the code using ast module to fix any remaining indentation issues
		formattedCode, err := formatPythonCode(dedentedCode)
		if err != nil {
			if pr.verbose {
				fmt.Printf("DEBUG: Failed to format Python code, using dedented code: %v\n", err)
			}
			// If formatting fails, use dedented code
			formattedCode = dedentedCode
		} else if pr.verbose {
			fmt.Printf("DEBUG: Code after formatting:\n%q\n", formattedCode)
		}

		wrappedCode := fmt.Sprintf("%s\nprint('%s')", formattedCode, uniqueMarker)
		finalCode = fmt.Sprintf("exec(%s)", strconv.Quote(wrappedCode))
		markerCmd = "" // Marker is already included in finalCode
	} else {
		// For single-line code, send it normally
		finalCode = code
		markerCmd = fmt.Sprintf("print('%s')\n", uniqueMarker)
	}

	// Ensure the code ends with a newline to be executed by the REPL.
	if !strings.HasSuffix(finalCode, "\n") {
		finalCode += "\n"
	}

	// Send the command (marker is already included for multiline code)
	if _, err := fmt.Fprint(pr.stdin, finalCode); err != nil {
		return "", fmt.Errorf("failed to write code to python stdin: %w", err)
	}

	// For single-line code without marker, send it separately
	if markerCmd != "" {
		if _, err := fmt.Fprint(pr.stdin, markerCmd); err != nil {
			return "", fmt.Errorf("failed to write marker to python stdin: %w", err)
		}
	}

	// Absolutely final robust error handling logic.
	// Collect all outputs until the result marker is received, then decide.
	var stdoutResult string
	var stderrResult strings.Builder
	var resultReceived bool

	timeout := time.After(pr.executionTimeout)

	for !resultReceived {
		select {
		case result := <-pr.resultChan:
			if pr.verbose {
				fmt.Printf("DEBUG: sendAndAwaitWithID received result from resultChan: '%s'\n", result)
			}

			// Check if we got an empty result (which might be a race condition)
			if result == "" {
				if pr.verbose {
					fmt.Printf("DEBUG: sendAndAwaitWithID received empty result, waiting briefly for actual result...\n")
				}

				// Wait a short time to see if the actual result arrives
				time.Sleep(20 * time.Millisecond)

				// Check if there's another result waiting (this should be the actual one)
				select {
				case actualResult := <-pr.resultChan:
					if pr.verbose {
						fmt.Printf("DEBUG: sendAndAwaitWithID received actual result after delay: '%s'\n", actualResult)
					}
					stdoutResult = actualResult
					resultReceived = true
				default:
					// No actual result arrived, use the empty result
					if pr.verbose {
						fmt.Printf("DEBUG: sendAndAwaitWithID no actual result arrived after delay, using empty result\n")
					}
					stdoutResult = result
					resultReceived = true
				}
			} else {
				// We got a non-empty result, use it
				stdoutResult = result
				resultReceived = true
			}
		case err := <-pr.errorChan:
			if pr.verbose {
				fmt.Printf("DEBUG: sendAndAwaitWithID received error from errorChan: '%s'\n", err.Error())
			}
			stderrResult.WriteString(err.Error())

			// Check if this is a Python traceback (exception) that should terminate execution
			errorStr := err.Error()
			if strings.Contains(errorStr, "Traceback (most recent call last):") ||
				strings.Contains(errorStr, "SyntaxError:") ||
				strings.Contains(errorStr, "ValueError:") ||
				strings.Contains(errorStr, "TypeError:") ||
				strings.Contains(errorStr, "NameError:") ||
				strings.Contains(errorStr, "IndexError:") ||
				strings.Contains(errorStr, "KeyError:") ||
				strings.Contains(errorStr, "AttributeError:") {
				// This is a Python exception, we should stop waiting for result and return error
				if pr.verbose {
					fmt.Printf("DEBUG: sendAndAwaitWithID detected Python exception, terminating execution\n")
				}
				resultReceived = true
				continue
			}
		case <-timeout:
			if pr.cmd != nil && pr.cmd.Process != nil {
				_ = pr.cmd.Process.Kill()
			}
			// If we timed out but got some error message, return that.
			if stderrResult.Len() > 0 {
				return "", fmt.Errorf("%s", stderrResult.String())
			}
			return "", fmt.Errorf("python execution timed out after %v", pr.executionTimeout)
		}
	}

	if pr.verbose {
		fmt.Printf("DEBUG: sendAndAwaitWithID final stdoutResult before trimming: '%s'\n", stdoutResult)
		fmt.Printf("DEBUG: sendAndAwaitWithID stderrResult: '%s'\n", stderrResult.String())
	}

	// After receiving the result, do a final short drain on stderr to catch trailing error messages.
	drainTimeout := time.After(50 * time.Millisecond)
	draining := true
	for draining {
		select {
		case err := <-pr.errorChan:
			stderrResult.WriteString(err.Error())
		case <-drainTimeout:
			draining = false
		}
	}

	// Final decision: if a real error occurred, stderr will contain a traceback.
	errorString := stderrResult.String()
	if pr.verbose {
		fmt.Printf("DEBUG: sendAndAwaitWithID final errorString: '%s'\n", errorString)
	}
	if strings.Contains(errorString, "Traceback (most recent call last):") || strings.Contains(errorString, "SyntaxError:") {
		return "", fmt.Errorf("%s", errorString)
	}

	trimmedResult := strings.TrimSpace(stdoutResult)
	if pr.verbose {
		fmt.Printf("DEBUG: sendAndAwaitWithID returning trimmed result: '%s'\n", trimmedResult)
	}
	return trimmedResult, nil
}

// dedent removes common leading whitespace from a block of text.
// Special handling for code extracted from py { ... } blocks where the first line
// might have different indentation than subsequent lines.
func dedent(code string) string {
	lines := strings.Split(code, "\n")

	// If we have only one line or empty code, return as is
	if len(lines) <= 1 {
		return strings.TrimSpace(code)
	}

	// For Python code blocks, we need to be more careful about indentation
	// First, find the minimum indentation of non-empty lines
	minIndent := -1
	nonEmptyLines := 0

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		nonEmptyLines++
		indent := len(line) - len(strings.TrimLeft(line, " \t"))
		if minIndent == -1 || indent < minIndent {
			minIndent = indent
		}
	}

	// If all lines are empty, return as is
	if nonEmptyLines == 0 {
		return strings.TrimSpace(code)
	}

	// If we found a valid minimum indentation, remove it from all lines
	if minIndent > 0 {
		var result strings.Builder
		for i, line := range lines {
			if strings.TrimSpace(line) == "" {
				// Preserve empty lines but without extra indentation
				if i > 0 && i < len(lines)-1 {
					result.WriteString("\n")
				}
			} else {
				// Remove the common indentation
				lineIndent := len(line) - len(strings.TrimLeft(line, " \t"))
				if lineIndent >= minIndent {
					result.WriteString(line[minIndent:])
				} else {
					// This line has less indentation, keep it as is
					result.WriteString(line)
				}
				if i < len(lines)-1 {
					result.WriteString("\n")
				}
			}
		}
		return strings.Trim(result.String(), "\n")
	}

	// If no indentation to remove, return the code as is (trimmed)
	return strings.TrimSpace(code)
}

// formatPythonCode форматирует Python код с использованием модуля ast
func formatPythonCode(code string) (string, error) {
	// Если код пустой, возвращаем его как есть
	if strings.TrimSpace(code) == "" {
		return code, nil
	}

	// Создаем Python скрипт для форматирования кода с использованием ast
	formatScript := `
import ast
import sys

def format_code(code_str):
    try:
        # Парсим код в AST
        tree = ast.parse(code_str)
        
        # Восстанавливаем код из AST - это автоматически форматирует его
        formatted_code = ast.unparse(tree)
        
        return formatted_code
    except SyntaxError as e:
        # Если синтаксическая ошибка, возвращаем оригинальный код
        return code_str
    except Exception as e:
        # При любой другой ошибке, возвращаем оригинальный код
        return code_str

# Читаем код из stdin
code = sys.stdin.read()
# Форматируем его
formatted = format_code(code)
# Выводим результат
print(formatted)
`

	// Создаем временный файл с Python скриптом
	tmpFile := "/tmp/funterm_format_code.py"
	file, err := os.Create(tmpFile)
	if err != nil {
		return code, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer func() {
		_ = os.Remove(tmpFile)
	}()

	// Записываем скрипт в файл
	_, err = file.WriteString(formatScript)
	if err != nil {
		_ = file.Close()
		return code, fmt.Errorf("failed to write format script: %w", err)
	}
	_ = file.Close()

	// Выполняем скрипт, передавая код на stdin
	cmd := exec.Command("python3", tmpFile)

	// Создаем pipe для stdin
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return code, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	// Записываем код в stdin
	go func() {
		defer func() {
			_ = stdin.Close()
		}()
		if _, err := io.WriteString(stdin, code); err != nil {
			// Log the error but continue
			fmt.Printf("Warning: Failed to write code to stdin: %v\n", err)
		}
	}()

	// Получаем вывод
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Если выполнение не удалось, возвращаем оригинальный код
		return code, nil
	}

	// Возвращаем отформатированный код
	formattedCode := strings.TrimSpace(string(output))
	if formattedCode == "" {
		return code, nil
	}

	return formattedCode, nil
}
