package node

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"

	"funterm/errors"
	"funterm/runtime"
)

const EndOfOutputMarker = "---SUTERM-NODE-EOP---"

// NodeRuntime implements the LanguageRuntime interface for Node.js
type NodeRuntime struct {
	ready            bool
	available        bool // Flag to indicate if node executable is found
	variables        map[string]interface{}
	nodePath         string
	mutex            sync.RWMutex
	processMutex     sync.Mutex
	executionTimeout time.Duration
	// stateManager removed for simplified beta
	verbose       bool
	outputCapture *strings.Builder // For capturing stdout output
	cmd           *exec.Cmd
	stdin         io.WriteCloser
	stdout        io.ReadCloser
	stderr        io.ReadCloser
	resultChan    chan string
	errorChan     chan error
}

// NewNodeRuntime creates a new Node.js runtime instance
func NewNodeRuntime() *NodeRuntime {
	return &NodeRuntime{
		ready:            false,
		available:        false,
		variables:        make(map[string]interface{}),
		nodePath:         "node",
		mutex:            sync.RWMutex{},
		executionTimeout: 30 * time.Second,
		verbose:          false,
	}
}

// SetVerbose sets the verbose mode for the Node runtime
func (nr *NodeRuntime) SetVerbose(verbose bool) {
	nr.verbose = verbose
}

// Initialize sets up the Node runtime
func (nr *NodeRuntime) Initialize() error {
	return nr.InitializeWithConfig()
}

// InitializeWithConfig sets up the Node runtime with library configuration
func (nr *NodeRuntime) InitializeWithConfig() error {
	nr.mutex.Lock()
	defer nr.mutex.Unlock()

	if err := nr.checkNodeAvailability(); err != nil {
		fmt.Printf("Warning: Node.js runtime is not available. %v\n", err)
		nr.available = false
		// Do not return an error, just mark as unavailable
		return nil
	}
	nr.available = true

	if err := nr.startPersistentProcess(); err != nil {
		return fmt.Errorf("failed to start persistent Node.js process: %w", err)
	}

	if err := nr.initializeNodeEnvironment(); err != nil {
		return fmt.Errorf("failed to initialize Node.js environment: %w", err)
	}

	nr.ready = true
	return nil
}

func (nr *NodeRuntime) checkNodeAvailability() error {
	cmd := exec.Command(nr.nodePath, "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("'%s' executable not found in PATH", nr.nodePath)
	}
	return nil
}

func (nr *NodeRuntime) startPersistentProcess() error {
	nr.cmd = exec.Command(nr.nodePath, "-i")

	var err error
	nr.stdin, err = nr.cmd.StdinPipe()
	if err != nil {
		return err
	}
	nr.stdout, err = nr.cmd.StdoutPipe()
	if err != nil {
		return err
	}
	nr.stderr, err = nr.cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := nr.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start persistent node process: %w", err)
	}

	nr.resultChan = make(chan string)
	nr.errorChan = make(chan error)

	go nr.readOutput(nr.stdout, nr.resultChan)
	go nr.readError(nr.stderr, nr.errorChan)

	return nil
}

func (nr *NodeRuntime) readOutput(pipe io.ReadCloser, ch chan<- string) {
	scanner := bufio.NewScanner(pipe)
	var outputBuffer strings.Builder
	for scanner.Scan() {
		line := scanner.Text()
		trimmedLine := strings.TrimSpace(strings.TrimPrefix(line, "> "))
		// Ignore node's welcome message
		if strings.HasPrefix(trimmedLine, "Welcome to Node.js") {
			continue
		}
		// Ignore node's REPL help message
		if strings.HasPrefix(trimmedLine, "Type \".help\" for more information.") {
			continue
		}
		if trimmedLine == EndOfOutputMarker {
			ch <- outputBuffer.String()
			outputBuffer.Reset()
		} else if trimmedLine != "undefined" && trimmedLine != "" {
			outputBuffer.WriteString(trimmedLine + "\n")

			// Also capture to outputCapture if it's set (for console.log output)
			nr.mutex.RLock()
			if nr.outputCapture != nil {
				if nr.verbose {
					fmt.Printf("DEBUG: readOutput - writing to outputCapture: '%s'\n", trimmedLine)
				}
				nr.outputCapture.WriteString(trimmedLine + "\n")
			} else {
				if nr.verbose {
					fmt.Printf("DEBUG: readOutput - outputCapture is nil, not writing: '%s'\n", trimmedLine)
				}
			}
			nr.mutex.RUnlock()
		}
	}
}

func (nr *NodeRuntime) readError(pipe io.ReadCloser, ch chan<- error) {
	scanner := bufio.NewScanner(pipe)
	for scanner.Scan() {
		line := scanner.Text()
		// Ignore node's welcome message and blank lines
		if !strings.HasPrefix(line, "Welcome to Node.js") && line != "" {
			ch <- fmt.Errorf("node stderr: %s", line)
		}
	}
}

func (nr *NodeRuntime) sendAndAwait(code string) (string, error) {
	nr.processMutex.Lock()
	defer nr.processMutex.Unlock()

	// Clear channels
	for len(nr.resultChan) > 0 {
		<-nr.resultChan
	}
	for len(nr.errorChan) > 0 {
		<-nr.errorChan
	}

	// The marker command is 'console.log("...")'.
	// Node's REPL prints the result of the last expression, which is 'undefined' for console.log.
	// We must handle this 'undefined' output.
	fullCommand := fmt.Sprintf("%s;\nconsole.log('%s');\n", code, EndOfOutputMarker)

	if _, err := fmt.Fprint(nr.stdin, fullCommand); err != nil {
		return "", fmt.Errorf("failed to write to node stdin: %w", err)
	}

	timeout := time.After(nr.executionTimeout)
	var stderrOutput strings.Builder

	for {
		select {
		case result := <-nr.resultChan:
			// The result might contain the actual output and 'undefined' from the marker.
			// We need to trim the 'undefined' part.
			return strings.TrimSpace(result), nil
		case err := <-nr.errorChan:
			stderrOutput.WriteString(err.Error() + "\n")
			// Let's see if a result comes through anyway, sometimes node prints warnings to stderr
		case <-timeout:
			if stderrOutput.Len() > 0 {
				return "", fmt.Errorf("node execution timed out; stderr: %s", stderrOutput.String())
			}
			return "", fmt.Errorf("node execution timed out")
		}
	}
}

func (nr *NodeRuntime) initializeNodeEnvironment() error {
	// No specific initialization needed for now, but we can add things like global helpers here.
	return nil
}

// executeConsoleLog handles console.log functionality for both print and console.log calls
func (nr *NodeRuntime) executeConsoleLog(args []interface{}) (interface{}, error) {
	// Filter out nil arguments to prevent outputting "undefined"
	var filteredArgs []interface{}
	for _, arg := range args {
		if arg != nil {
			filteredArgs = append(filteredArgs, arg)
		}
	}

	// If all arguments are nil, return no output
	if len(filteredArgs) == 0 {
		return nil, nil
	}

	// Special handling for console.log
	// It prints to stdout and we'll return the output as the result.
	argsJSON, err := json.Marshal(filteredArgs)
	if err != nil {
		return nil, errors.NewRuntimeError("node", "INVALID_ARGUMENT", fmt.Sprintf("failed to marshal arguments: %v", err))
	}
	code := fmt.Sprintf("console.log(...%s)", string(argsJSON))
	output, err := nr.sendAndAwait(code)
	if err != nil {
		return nil, errors.NewRuntimeError("node", "EXECUTION_FAILED", err.Error())
	}
	// Return the output as the result instead of printing it
	if output != "" {
		return strings.TrimSpace(output), nil
	}
	return nil, nil
}

// ExecuteFunction calls a function in the Node runtime
func (nr *NodeRuntime) ExecuteFunction(name string, args []interface{}) (interface{}, error) {
	nr.mutex.Lock()
	// Always initialize output capture for any function call
	nr.outputCapture = &strings.Builder{}
	nr.mutex.Unlock()

	if !nr.ready {
		if !nr.available {
			return nil, errors.NewRuntimeError("node", "RUNTIME_UNAVAILABLE", "Node.js runtime is unavailable. Please install Node.js.")
		}
		return nil, errors.NewRuntimeError("node", "RUNTIME_NOT_INITIALIZED", "runtime is not initialized")
	}

	if name == "print" {
		// Special handling for print function - redirect to console.log
		return nr.executeConsoleLog(args)
	}

	if name == "console.log" {
		// Special handling for console.log
		return nr.executeConsoleLog(args)
	}

	argsJSON, err := json.Marshal(args)
	if err != nil {
		return nil, errors.NewRuntimeError("node", "INVALID_ARGUMENT", fmt.Sprintf("failed to marshal arguments: %v", err))
	}

	// First check if the function exists to avoid error messages
	checkCode := fmt.Sprintf("if (typeof %s !== 'undefined') { console.log('EXISTS'); } else { console.log('NOT_EXISTS'); }", name)
	checkOutput, err := nr.sendAndAwait(checkCode)
	if err != nil {
		return nil, errors.NewRuntimeError("node", "EXECUTION_FAILED", fmt.Sprintf("failed to check function: %v", err))
	}

	if strings.TrimSpace(checkOutput) != "EXISTS" {
		return nil, errors.NewRuntimeError("node", "FUNCTION_NOT_FOUND", fmt.Sprintf("function '%s' not found", name))
	}

	// Clear output capture before actual function execution
	nr.mutex.Lock()
	nr.outputCapture.Reset()
	nr.mutex.Unlock()

	// `apply` is used to call the function with an array of arguments.
	code := fmt.Sprintf("console.log(JSON.stringify(%s.apply(null, %s)))", name, string(argsJSON))
	output, err := nr.sendAndAwait(code)
	if err != nil {
		return nil, errors.NewRuntimeError("node", "EXECUTION_FAILED", err.Error())
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) > 0 {
		lastLine := lines[len(lines)-1]
		if lastLine == "undefined" || lastLine == "" {
			// It's a statement that doesn't return a value, like console.log or variable assignment.
			return nil, nil
		}
	}

	var result interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		// If it's not JSON, return it as a raw string.
		return output, nil
	}

	return result, nil
}

// SetVariable sets a variable in the Node runtime
func (nr *NodeRuntime) SetVariable(name string, value interface{}) error {
	if !nr.ready {
		if !nr.available {
			return errors.NewRuntimeError("node", "RUNTIME_UNAVAILABLE", "Node.js runtime is unavailable. Please install Node.js.")
		}
		return errors.NewRuntimeError("node", "RUNTIME_NOT_INITIALIZED", "runtime is not initialized")
	}

	valueJSON, err := json.Marshal(value)
	if err != nil {
		return errors.NewRuntimeError("node", "INVALID_ARGUMENT", fmt.Sprintf("failed to marshal value: %v", err))
	}

	code := fmt.Sprintf("var %s = %s;", name, string(valueJSON))
	_, err = nr.sendAndAwait(code)
	if err != nil {
		return errors.NewRuntimeError("node", "EXECUTION_FAILED", fmt.Sprintf("failed to set variable: %v", err))
	}
	return nil
}

// GetVariable retrieves a variable from the Node runtime
func (nr *NodeRuntime) GetVariable(name string) (interface{}, error) {
	if !nr.ready {
		if !nr.available {
			return nil, errors.NewRuntimeError("node", "RUNTIME_UNAVAILABLE", "Node.js runtime is unavailable. Please install Node.js.")
		}
		return nil, errors.NewRuntimeError("node", "RUNTIME_NOT_INITIALIZED", "runtime is not initialized")
	}

	// First check if the variable exists to avoid error messages
	checkCode := fmt.Sprintf("if (typeof %s !== 'undefined') { console.log(JSON.stringify(%s)); } else { console.log('__NOT_FOUND__'); }", name, name)
	output, err := nr.sendAndAwait(checkCode)
	if err != nil {
		return nil, errors.NewRuntimeError("node", "EXECUTION_FAILED", fmt.Sprintf("failed to get variable: %v", err))
	}

	if output == "__NOT_FOUND__" || output == "" || output == "null" || output == "undefined" {
		return nil, errors.NewRuntimeError("node", "VARIABLE_NOT_FOUND", fmt.Sprintf("variable '%s' not found", name))
	}

	var result interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		return output, nil // Return as string if not JSON
	}
	return result, nil
}

// Eval executes arbitrary code
func (nr *NodeRuntime) Eval(code string) (interface{}, error) {
	if !nr.ready {
		if !nr.available {
			return nil, errors.NewRuntimeError("node", "RUNTIME_UNAVAILABLE", "Node.js runtime is unavailable. Please install Node.js.")
		}
		return nil, errors.NewRuntimeError("node", "RUNTIME_NOT_INITIALIZED", "runtime is not initialized")
	}

	if nr.verbose {
		fmt.Printf("DEBUG: NodeRuntime Eval - original code: '%s'\n", code)
	}

	// For eval, we wrap in console.log to get the output
	wrappedCode := fmt.Sprintf("console.log(%s)", code)
	if nr.verbose {
		fmt.Printf("DEBUG: NodeRuntime Eval - wrapped code: '%s'\n", wrappedCode)
	}
	output, err := nr.sendAndAwait(wrappedCode)
	if err != nil {
		if nr.verbose {
			fmt.Printf("DEBUG: NodeRuntime Eval - wrapped version failed, trying raw: %v\n", err)
		}
		// If the wrapped version fails, try the raw version. This helps with declarations like `var x = 10;`.
		rawOutput, rawErr := nr.sendAndAwait(code)
		if rawErr != nil {
			return nil, errors.NewRuntimeError("node", "EXECUTION_FAILED", err.Error()) // Return original error
		}
		output = rawOutput
	}

	if nr.verbose {
		fmt.Printf("DEBUG: NodeRuntime Eval - output: '%s'\n", output)
	}

	if output == "" {
		return nil, nil
	}

	return strings.TrimSpace(output), nil
}

// The rest of the interface methods need to be implemented.
// For now, they can be stubs.

func (nr *NodeRuntime) GetName() string {
	return "node"
}

func (nr *NodeRuntime) IsReady() bool {
	return nr.ready
}

func (nr *NodeRuntime) Cleanup() error {
	nr.mutex.Lock()
	defer nr.mutex.Unlock()
	if nr.cmd != nil && nr.cmd.Process != nil {
		return nr.cmd.Process.Kill()
	}
	return nil
}

// --- Stubs for the rest of the LanguageRuntime interface ---

func (nr *NodeRuntime) ExecuteFunctionMultiple(functionName string, args ...interface{}) ([]interface{}, error) {
	return nil, fmt.Errorf("not implemented")
}

func (nr *NodeRuntime) Isolate() error {
	return fmt.Errorf("not implemented")
}

func (nr *NodeRuntime) SetExecutionTimeout(timeout time.Duration) {
	nr.mutex.Lock()
	defer nr.mutex.Unlock()
	nr.executionTimeout = timeout
}

func (nr *NodeRuntime) GetExecutionTimeout() time.Duration {
	nr.mutex.RLock()
	defer nr.mutex.RUnlock()
	return nr.executionTimeout
}

func (nr *NodeRuntime) GetSupportedTypes() []string {
	return []string{"string", "number", "boolean", "null", "array", "object"}
}

func (nr *NodeRuntime) ExecuteCodeBlock(code string) (interface{}, error) {
	if !nr.ready {
		if !nr.available {
			return nil, errors.NewRuntimeError("node", "RUNTIME_UNAVAILABLE", "Node.js runtime is unavailable. Please install Node.js.")
		}
		return nil, errors.NewRuntimeError("node", "RUNTIME_NOT_INITIALIZED", "runtime is not initialized")
	}

	// Initialize output capture for code blocks
	nr.mutex.Lock()
	nr.outputCapture = &strings.Builder{}
	nr.mutex.Unlock()

	if nr.verbose {
		fmt.Printf("DEBUG: ExecuteCodeBlock called with code: %s\n", code)
	}

	// Обрабатываем код для глобальной области видимости
	processedCode := nr.processCodeForGlobalScope(code)

	if nr.verbose {
		fmt.Printf("DEBUG: Processed code for global scope: %s\n", processedCode)
	}

	// Execute the processed code
	output, err := nr.sendAndAwait(processedCode)
	if err != nil {
		if nr.verbose {
			fmt.Printf("DEBUG: ExecuteCodeBlock error: %v\n", err)
		}
		return nil, errors.NewRuntimeError("node", "EXECUTION_FAILED", err.Error())
	}

	if nr.verbose {
		fmt.Printf("DEBUG: ExecuteCodeBlock output: %s\n", output)
	}

	return nr.processExecuteCodeBlockOutput(output)
}

// processCodeForGlobalScope converts let/const declarations to var for global scope
func (nr *NodeRuntime) processCodeForGlobalScope(code string) string {
	// Simple replacement of let/const with var
	// This is a basic approach; a more sophisticated parser might be needed for complex cases
	lines := strings.Split(code, "\n")
	var processedLines []string

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// Skip comments and empty lines
		if strings.HasPrefix(trimmedLine, "//") || trimmedLine == "" {
			processedLines = append(processedLines, line)
			continue
		}

		// Replace let with var
		if strings.HasPrefix(trimmedLine, "let ") {
			processedLine := strings.Replace(line, "let ", "var ", 1)
			processedLines = append(processedLines, processedLine)
			continue
		}

		// Replace const with var
		if strings.HasPrefix(trimmedLine, "const ") {
			processedLine := strings.Replace(line, "const ", "var ", 1)
			processedLines = append(processedLines, processedLine)
			continue
		}

		// Keep the line as is
		processedLines = append(processedLines, line)
	}

	return strings.Join(processedLines, "\n")
}

// processCodeForVariableCapture обрабатывает код для сохранения переменных в глобальной области видимости
func (nr *NodeRuntime) processCodeForVariableCapture(code string, variables []string) string {
	if len(variables) == 0 {
		return code
	}

	// Создаем обертку, которая автоматически сохраняет переменные в глобальную область
	// Аналогично тому, как Python делает переменные глобальными при выполнении кода
	lines := strings.Split(code, "\n")
	var processedLines []string
	variablesToCapture := make(map[string]bool)

	for _, v := range variables {
		variablesToCapture[v] = true
	}

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// Пропускаем комментарии и пустые строки
		if strings.HasPrefix(trimmedLine, "//") || trimmedLine == "" {
			processedLines = append(processedLines, line)
			continue
		}

		// Проверяем, является ли строка объявлением переменной
		// Если это одна из переменных, которые нужно сохранить, преобразуем в глобальную
		if strings.HasPrefix(trimmedLine, "var ") {
			// Извлекаем имя переменной
			parts := strings.SplitN(trimmedLine[4:], "=", 2)
			if len(parts) > 0 {
				varName := strings.TrimSpace(parts[0])
				if variablesToCapture[varName] {
					// Заменяем `var x = value` на `globalThis.x = value`
					processedLine := strings.Replace(line, "var "+varName, "globalThis."+varName, 1)
					processedLines = append(processedLines, processedLine)
					continue
				}
			}
		} else if strings.HasPrefix(trimmedLine, "let ") {
			// Извлекаем имя переменной
			parts := strings.SplitN(trimmedLine[4:], "=", 2)
			if len(parts) > 0 {
				varName := strings.TrimSpace(parts[0])
				if variablesToCapture[varName] {
					// Заменяем `let x = value` на `globalThis.x = value`
					processedLine := strings.Replace(line, "let "+varName, "globalThis."+varName, 1)
					processedLines = append(processedLines, processedLine)
					continue
				}
			}
		} else if strings.HasPrefix(trimmedLine, "const ") {
			// Извлекаем имя переменной
			parts := strings.SplitN(trimmedLine[5:], "=", 2)
			if len(parts) > 0 {
				varName := strings.TrimSpace(parts[0])
				if variablesToCapture[varName] {
					// Заменяем `const x = value` на `globalThis.x = value`
					processedLine := strings.Replace(line, "const "+varName, "globalThis."+varName, 1)
					processedLines = append(processedLines, processedLine)
					continue
				}
			}
		}

		// Если строка не содержит объявления переменной, которую нужно сохранить, оставляем как есть
		processedLines = append(processedLines, line)
	}

	// Оборачиваем весь код в функцию для изоляции области видимости
	// и явного сохранения переменных в глобальную область
	wrappedCode := fmt.Sprintf(`
(function() {
	// Выполняем оригинальный код
	%s
	
	// Явно сохраняем указанные переменные в глобальную область
	try {
		%s
	} catch (e) {
		// Игнорируем ошибки при сохранении переменных
	}
})();
`, strings.Join(processedLines, "\n"), nr.generateVariableExportCode(variables))

	return wrappedCode
}

// generateVariableExportCode генерирует код для экспорта переменных в глобальную область
func (nr *NodeRuntime) generateVariableExportCode(variables []string) string {
	var exportLines []string

	for _, v := range variables {
		exportLines = append(exportLines, fmt.Sprintf("if (typeof %s !== 'undefined') { globalThis.%s = %s; }", v, v, v))
	}

	return strings.Join(exportLines, "\n")
}

// processExecuteCodeBlockOutput processes the output from ExecuteCodeBlock
func (nr *NodeRuntime) processExecuteCodeBlockOutput(output string) (interface{}, error) {
	// Clean up Node.js REPL output
	lines := strings.Split(output, "\n")
	var cleanLines []string

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		// Skip REPL prompts and undefined output
		if trimmedLine == "" || trimmedLine == "undefined" || strings.HasPrefix(trimmedLine, ">") {
			continue
		}
		cleanLines = append(cleanLines, trimmedLine)
	}

	// Join the cleaned lines
	cleanOutput := strings.Join(cleanLines, "\n")

	// Return the cleaned output or nil if empty
	if cleanOutput == "" {
		return nil, nil
	}

	return cleanOutput, nil
}

func (nr *NodeRuntime) ExecuteBatch(code string) error {
	_, err := nr.Eval(code)
	return err
}

// Methods already exist in other files - removed duplicates

// --- Completion interface implementation ---

// getModulesDynamically attempts to get modules dynamically through Node.js introspection
func (nr *NodeRuntime) getModulesDynamically() ([]string, error) {
	if !nr.ready {
		return nil, fmt.Errorf("runtime not ready")
	}

	// JavaScript code to get built-in modules dynamically
	jsCode := `
// Get built-in modules
const builtinModules = [];
try {
	// Try to get module list from process.binding (older Node.js versions)
	if (typeof process.binding === 'function') {
		try {
			const natives = process.binding('natives');
			for (const mod in natives) {
				if (mod !== 'internal_binding' && !mod.startsWith('internal/')) {
					builtinModules.push(mod);
				}
			}
		} catch (e) {
			// Ignore error, try next method
		}
	}
	
	// Try module.builtinModules (Node.js 9.3.0+)
	if (typeof module.builtinModules === 'object') {
		builtinModules.push(...module.builtinModules);
	}
	
	// Try require('module').builtinModules (Node.js 9.3.0+ alternative)
	try {
		const mod = require('module');
		if (mod.builtinModules) {
			builtinModules.push(...mod.builtinModules);
		}
	} catch (e) {
		// Ignore error
	}
	
	// Remove duplicates and sort
	const uniqueModules = [...new Set(builtinModules)];
	uniqueModules.sort();
	console.log(JSON.stringify(uniqueModules));
} catch (e) {
	console.log(JSON.stringify([]));
}
`

	output, err := nr.sendAndAwait(jsCode)
	if err != nil {
		return nil, fmt.Errorf("failed to get modules dynamically: %w", err)
	}

	var modules []string
	if err := json.Unmarshal([]byte(output), &modules); err != nil {
		return nil, fmt.Errorf("failed to parse modules JSON: %w", err)
	}

	return modules, nil
}

// getModulesFallback returns a static list of common Node.js modules
func (nr *NodeRuntime) getModulesFallback() []string {
	return []string{
		"assert", "async_hooks", "buffer", "child_process", "cluster", "console",
		"constants", "crypto", "dgram", "dns", "domain", "events", "fs", "http",
		"http2", "https", "inspector", "module", "net", "os", "path", "perf_hooks",
		"process", "punycode", "querystring", "readline", "repl", "stream",
		"string_decoder", "timers", "tls", "trace_events", "tty", "url",
		"util", "v8", "vm", "worker_threads", "zlib",

		// Node.js 12+ modules
		"diagnostics_channel",

		// Node.js 14+ modules
		"report",

		// Node.js 15+ modules
		"worker_threads",

		// Node.js 16+ modules
		"test", "fetch",
	}
}

// GetModules returns available modules for the Node.js runtime with fallback
func (nr *NodeRuntime) GetModules() []string {
	if !nr.ready {
		return []string{}
	}

	// Try to get modules dynamically first
	if modules, err := nr.getModulesDynamically(); err == nil {
		sort.Strings(modules)
		return modules
	}

	// Fall back to static list
	modules := nr.getModulesFallback()
	sort.Strings(modules)
	return modules
}

// getModuleFunctionsDynamically attempts to get functions for a specific module dynamically
func (nr *NodeRuntime) getModuleFunctionsDynamically(module string) ([]string, error) {
	if !nr.ready {
		return nil, fmt.Errorf("runtime not ready")
	}

	// JavaScript code to get module functions dynamically
	jsCode := fmt.Sprintf(`
try {
	const mod = require('%s');
	if (!mod) {
		console.log(JSON.stringify([]));
		return;
	}
	
	const functions = [];
	
	// Get all property names
	const allProps = Object.getOwnPropertyNames(mod);
	
	// Also get prototype properties for object-like modules
	if (typeof mod === 'object' && mod !== null) {
		const protoProps = Object.getOwnPropertyNames(Object.getPrototypeOf(mod));
		allProps.push(...protoProps);
	}
	
	// Filter functions and filter out internal properties
	for (const prop of allProps) {
		if (typeof mod[prop] === 'function' &&
			!prop.startsWith('_') &&
			prop !== 'constructor' &&
			prop !== 'toString' &&
			prop !== 'valueOf' &&
			prop !== 'hasOwnProperty' &&
			prop !== 'isPrototypeOf' &&
			prop !== 'propertyIsEnumerable' &&
			prop !== 'toLocaleString') {
			functions.push(prop);
		}
	}
	
	// Remove duplicates and sort
	const uniqueFunctions = [...new Set(functions)];
	uniqueFunctions.sort();
	console.log(JSON.stringify(uniqueFunctions));
} catch (e) {
	console.log(JSON.stringify([]));
}
`, module)

	output, err := nr.sendAndAwait(jsCode)
	if err != nil {
		return nil, fmt.Errorf("failed to get module functions dynamically: %w", err)
	}

	var functions []string
	if err := json.Unmarshal([]byte(output), &functions); err != nil {
		return nil, fmt.Errorf("failed to parse functions JSON: %w", err)
	}

	return functions, nil
}

// getModuleFunctionsFallback returns a static list of functions for common modules
func (nr *NodeRuntime) getModuleFunctionsFallback(module string) []string {
	switch module {
	case "fs":
		return []string{
			"access", "accessSync", "appendFile", "appendFileSync", "chmod", "chmodSync",
			"chown", "chownSync", "close", "closeSync", "copyFile", "copyFileSync",
			"createReadStream", "createWriteStream", "exists", "existsSync", "fchmod",
			"fchmodSync", "fchown", "fchownSync", "fdatasync", "fdatasyncSync", "fstat",
			"fstatSync", "fsync", "fsyncSync", "ftruncate", "ftruncateSync", "futimes",
			"futimesSync", "lchmod", "lchmodSync", "lchown", "lchownSync", "link",
			"linkSync", "lstat", "lstatSync", "mkdir", "mkdirSync", "mkdtemp",
			"mkdtempSync", "open", "openSync", "opendir", "opendirSync", "read",
			"readSync", "readdir", "readdirSync", "readFile", "readFileSync",
			"readlink", "readlinkSync", "realpath", "realpathSync", "rename",
			"renameSync", "rmdir", "rmdirSync", "rm", "rmSync", "stat", "statSync",
			"symlink", "symlinkSync", "truncate", "truncateSync", "unlink",
			"unlinkSync", "utimes", "utimesSync", "write", "writeSync", "writeFile",
			"writeFileSync",
		}
	case "path":
		return []string{
			"basename", "delimiter", "dirname", "extname", "format", "isAbsolute",
			"join", "normalize", "parse", "posix", "relative", "resolve", "sep",
			"toNamespacedPath", "win32",
		}
	case "http":
		return []string{
			"createClient", "createServer", "get", "globalAgent", "request", "Server",
			"ServerResponse",
		}
	case "https":
		return []string{
			"createServer", "get", "globalAgent", "request", "Server",
		}
	case "os":
		return []string{
			"arch", "cpus", "endianness", "freemem", "getPriority", "homedir",
			"hostname", "loadavg", "networkInterfaces", "platform", "release",
			"setPriority", "tmpdir", "totalmem", "type", "uptime", "userInfo",
			"version",
		}
	case "util":
		return []string{
			"callbackify", "debuglog", "deprecate", "format", "inherits",
			"inspect", "isArray", "isBoolean", "isBuffer", "isDate", "isDeepStrictEqual",
			"isError", "isFunction", "isNull", "isNumber", "isObject", "isPrimitive",
			"isRegExp", "isString", "isSymbol", "isUndefined", "log", "promisify",
			"types",
		}
	case "events":
		return []string{
			"EventEmitter", "captureRejectionSymbol", "defaultMaxListeners",
			"errorMonitor", "getEventListeners", "listenerCount", "on", "once",
			"setMaxListeners",
		}
	case "stream":
		return []string{
			"Duplex", "PassThrough", "Readable", "Stream", "Transform", "Writable",
			"_isUint8Array", "_uint8ArrayToBuffer", "finished", "pipeline",
			"promises",
		}
	case "crypto":
		return []string{
			"createCipher", "createCipheriv", "createDecipher", "createDecipheriv",
			"createDiffieHellman", "createDiffieHellmanGroup", "createECDH",
			"createHash", "createHmac", "createPrivateKey", "createPublicKey",
			"createSign", "createVerify", "constants", "getCiphers", "getCurves",
			"getDiffieHellman", "getHashes", "privateEncrypt", "publicDecrypt",
			"publicEncrypt", "randomBytes", "randomFill", "randomFillSync",
			"scrypt", "scryptSync", "sign", "timingSafeEqual", "verify",
		}
	case "url":
		return []string{
			"domainToASCII", "domainToUnicode", "format", "parse", "resolve",
			"resolveObject", "URL", "URLSearchParams",
		}
	case "querystring":
		return []string{
			"decode", "encode", "escape", "parse", "stringify", "unescape",
		}
	case "string_decoder":
		return []string{
			"StringDecoder",
		}
	case "zlib":
		return []string{
			"createBrotliCompress", "createBrotliDecompress", "createDeflate",
			"createDeflateRaw", "createGunzip", "createGzip", "createInflate",
			"createInflateRaw", "createUnzip", "constants", "deflate", "deflateRaw",
			"deflateSync", "deflateRawSync", "gunzip", "gunzipSync", "gzip",
			"gzipSync", "inflate", "inflateRaw", "inflateSync", "inflateRawSync",
			"unzip", "unzipSync",
		}
	case "buffer":
		return []string{
			"Buffer", "SlowBuffer", "atob", "btoa", "constants", "kStringMaxLength",
			"kMaxLength",
		}
	case "timers":
		return []string{
			"clearImmediate", "clearInterval", "clearTimeout", "setImmediate",
			"setInterval", "setTimeout",
		}
	case "process":
		return []string{
			"abort", "chdir", "cwd", "exit", "getgid", "getgroups", "getuid", "hrtime",
			"initgroups", "kill", "memoryUsage", "nextTick", "setgid", "setgroups",
			"setuid", "umask", "uptime",
		}
	default:
		return []string{}
	}
}

// GetModuleFunctions returns available functions for a specific Node.js module with fallback
func (nr *NodeRuntime) GetModuleFunctions(module string) []string {
	if !nr.ready {
		return []string{}
	}

	// Try to get module functions dynamically first
	if functions, err := nr.getModuleFunctionsDynamically(module); err == nil {
		sort.Strings(functions)
		return functions
	}

	// Fall back to static list
	functions := nr.getModuleFunctionsFallback(module)
	sort.Strings(functions)
	return functions
}

func (nr *NodeRuntime) GetFunctionSignature(module, function string) (string, error) {
	return "", fmt.Errorf("not implemented")
}

func (nr *NodeRuntime) GetGlobalVariables() []string {
	return []string{} // TODO
}

func (nr *NodeRuntime) GetCompletionSuggestions(input string) []string {
	return []string{} // TODO
}

func (nr *NodeRuntime) GetUserDefinedFunctions() []string {
	return []string{}
}

func (nr *NodeRuntime) GetImportedModules() []string {
	return []string{}
}

func (nr *NodeRuntime) GetDynamicCompletions(input string) ([]string, error) {
	return []string{}, nil
}

func (nr *NodeRuntime) GetObjectProperties(objectName string) ([]string, error) {
	return []string{}, nil
}

func (nr *NodeRuntime) GetFunctionParameters(functionName string) ([]runtime.FunctionParameter, error) {
	return []runtime.FunctionParameter{}, nil
}

func (nr *NodeRuntime) UpdateCompletionContext(executedCode string, result interface{}) error {
	return nil
}

func (nr *NodeRuntime) RefreshRuntimeState() error {
	return nil
}

func (nr *NodeRuntime) GetRuntimeObjects() map[string]interface{} {
	return make(map[string]interface{})
}

// GetVariableFromRuntimeObject получает переменную из variables cache
func (nr *NodeRuntime) GetVariableFromRuntimeObject(name string) (interface{}, error) {
	nr.mutex.RLock()
	defer nr.mutex.RUnlock()

	if value, exists := nr.variables[name]; exists {
		return value, nil
	}
	return nil, fmt.Errorf("variable '%s' not found in runtime objects", name)
}

// ExecuteCodeBlockWithVariables выполняет код с сохранением указанных переменных
func (nr *NodeRuntime) ExecuteCodeBlockWithVariables(code string, variables []string) (interface{}, error) {
	if !nr.ready {
		if !nr.available {
			return nil, errors.NewRuntimeError("node", "RUNTIME_UNAVAILABLE", "Node.js runtime is unavailable. Please install Node.js.")
		}
		return nil, errors.NewRuntimeError("node", "RUNTIME_NOT_INITIALIZED", "runtime is not initialized")
	}

	if nr.verbose {
		fmt.Printf("DEBUG: ExecuteCodeBlockWithVariables called with code: %s, variables: %v\n", code, variables)
	}

	// Обрабатываем код для сохранения переменных в глобальной области видимости
	// Аналогично тому, как Python runtime автоматически делает переменные глобальными
	processedCode := nr.processCodeForVariableCapture(code, variables)

	if nr.verbose {
		fmt.Printf("DEBUG: Original code: %s\n", code)
		fmt.Printf("DEBUG: Processed code for variable capture: %s\n", processedCode)
	}

	// Выполняем обработанный код без буферизации (как Eval)
	output, err := nr.sendAndAwait(processedCode)
	if err != nil {
		if nr.verbose {
			fmt.Printf("DEBUG: ExecuteCodeBlockWithVariables execution error: %v\n", err)
		}
		return nil, errors.NewRuntimeError("node", "EXECUTION_FAILED", err.Error())
	}

	if nr.verbose {
		fmt.Printf("DEBUG: ExecuteCodeBlockWithVariables execution output: %s\n", output)
	}

	// После успешного выполнения, захватываем только указанные переменные
	if len(variables) > 0 {
		if nr.verbose {
			fmt.Printf("DEBUG: Capturing specified variables after code block execution: %v\n", variables)
		}

		// Создаем код для получения только указанных переменных из глобальной области
		var variableNames []string
		for _, v := range variables {
			variableNames = append(variableNames, fmt.Sprintf("'%s'", v))
		}

		variablesCode := fmt.Sprintf(`
// Функция для безопасной сериализации объекта
function funterm_safe_serialize(obj) {
    try {
        return JSON.stringify(obj);
    } catch (e) {
        return String(obj);
    }
}

// Функция для проверки, нужно ли включать переменную
function funterm_should_include(var_name, var_value) {
    // Пропускаем внутренние переменные
    if (var_name.startsWith('_') || var_name === 'funterm_safe_serialize' || var_name === 'funterm_should_include') {
        return false;
    }
    // Пропускаем функции
    if (typeof var_value === 'function') {
        return false;
    }
    return true;
}

// Получаем указанные переменные из глобальной области
var funterm_globals_dict = {};
var variablesToCapture = [%s];

for (var i = 0; i < variablesToCapture.length; i++) {
    var varName = variablesToCapture[i];
    try {
        // Проверяем глобальную область видимости (аналог globals() в Python)
        var varValue = globalThis[varName];
        if (typeof varValue !== 'undefined' && funterm_should_include(varName, varValue)) {
            funterm_globals_dict[varName] = varValue;
        }
    } catch (e) {
        // Пропускаем переменные, которые не удалось обработать
    }
}

console.log(JSON.stringify(funterm_globals_dict));
	`, strings.Join(variableNames, ", "))

		variablesOutput, err := nr.sendAndAwait(variablesCode)
		if err != nil {
			if nr.verbose {
				fmt.Printf("DEBUG: Failed to capture specified variables: %v\n", err)
			}
			// Return original result even if variable capture fails
			return nr.processExecuteCodeBlockOutput(output)
		}

		if nr.verbose {
			fmt.Printf("DEBUG: Raw variables output: '%s'\n", variablesOutput)
		}

		// Extract JSON from the output (skip REPL prompts and other noise)
		jsonStart := strings.Index(variablesOutput, "{")
		jsonEnd := strings.LastIndex(variablesOutput, "}")

		var cleanVariablesOutput string
		if jsonStart != -1 && jsonEnd != -1 && jsonEnd > jsonStart {
			cleanVariablesOutput = variablesOutput[jsonStart : jsonEnd+1]
			if nr.verbose {
				fmt.Printf("DEBUG: Cleaned variables output: '%s'\n", cleanVariablesOutput)
			}
		} else {
			if nr.verbose {
				fmt.Printf("DEBUG: No JSON found in variables output\n")
			}
			// Return original result if no JSON found
			return nr.processExecuteCodeBlockOutput(output)
		}

		// Parse the variables JSON
		var capturedVariables map[string]interface{}
		if err := json.Unmarshal([]byte(cleanVariablesOutput), &capturedVariables); err == nil {
			nr.mutex.Lock()
			// Update local cache only with specified variables
			for name, value := range capturedVariables {
				nr.variables[name] = value
				if nr.verbose {
					fmt.Printf("DEBUG: Cached specified variable: %s = %v\n", name, value)
				}
			}
			nr.mutex.Unlock()

			// Shared storage functionality removed for simplified beta version
		} else if nr.verbose {
			fmt.Printf("DEBUG: Failed to parse variables JSON: %v\n", err)
		}
	}

	return nr.processExecuteCodeBlockOutput(output)
}

// GetCapturedOutput returns the captured stdout output and clears the capture buffer
func (nr *NodeRuntime) GetCapturedOutput() string {
	nr.mutex.Lock()
	defer nr.mutex.Unlock()

	if nr.outputCapture == nil {
		if nr.verbose {
			fmt.Printf("DEBUG: GetCapturedOutput - outputCapture is nil\n")
		}
		return ""
	}

	captured := nr.outputCapture.String()
	if nr.verbose {
		fmt.Printf("DEBUG: GetCapturedOutput - captured: '%s'\n", captured)
	}

	// Trim trailing newlines to avoid extra line breaks in output
	captured = strings.TrimSpace(captured)

	nr.outputCapture.Reset()
	return captured
}

// SendAndAwait sends code to Node.js runtime and awaits response without console.log wrapper
// This is a public wrapper around the private sendAndAwait method
func (nr *NodeRuntime) SendAndAwait(code string) (string, error) {
	return nr.sendAndAwait(code)
}
