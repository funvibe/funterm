package go_runtime

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	"funterm/errors"
	"funterm/runtime"
)

// GoRuntime implements the LanguageRuntime interface for Go standard library functions
type GoRuntime struct {
	ready bool
	// stateManager removed for simplified beta
	functionRegistry map[string]func([]interface{}) (interface{}, error)
	verbose          bool
}

// NewGoRuntime creates a new Go runtime instance
func NewGoRuntime() *GoRuntime {
	gr := &GoRuntime{
		ready:            false,
		functionRegistry: make(map[string]func([]interface{}) (interface{}, error)),
		verbose:          false,
	}
	gr.initializeFunctionRegistry()
	return gr
}

// initializeFunctionRegistry sets up all available Go standard library functions
func (gr *GoRuntime) initializeFunctionRegistry() {
	// File operations
	gr.functionRegistry["read_file"] = gr.readFile
	gr.functionRegistry["write_file"] = gr.writeFile
	gr.functionRegistry["file_exists"] = gr.fileExists
	gr.functionRegistry["list_dir"] = gr.listDir
	gr.functionRegistry["delete_file"] = gr.deleteFile

	// Process operations
	gr.functionRegistry["exec"] = gr.execCommand

	// HTTP operations
	gr.functionRegistry["http_get"] = gr.httpGet
	gr.functionRegistry["http_post"] = gr.httpPost

	// Cryptographic operations
	gr.functionRegistry["md5"] = gr.md5Hash
	gr.functionRegistry["sha256"] = gr.sha256Hash

	// String operations
	gr.functionRegistry["base64_encode"] = gr.base64Encode
	gr.functionRegistry["base64_decode"] = gr.base64Decode
	gr.functionRegistry["url_encode"] = gr.urlEncode
	gr.functionRegistry["url_decode"] = gr.urlDecode
	gr.functionRegistry["to_upper"] = gr.toUpper
	gr.functionRegistry["to_lower"] = gr.toLower
	gr.functionRegistry["trim"] = gr.trim
	gr.functionRegistry["split"] = gr.split
	gr.functionRegistry["join"] = gr.join

	// JSON operations
	gr.functionRegistry["json_encode"] = gr.jsonEncode
	gr.functionRegistry["json_decode"] = gr.jsonDecode

	// YAML operations
	gr.functionRegistry["parse_yaml"] = gr.parseYAML

	// Math operations
	gr.functionRegistry["sin"] = gr.sin
	gr.functionRegistry["cos"] = gr.cos
	gr.functionRegistry["sqrt"] = gr.sqrt
	gr.functionRegistry["pow"] = gr.pow
	gr.functionRegistry["floor"] = gr.floor
	gr.functionRegistry["ceil"] = gr.ceil

	// Time operations
	gr.functionRegistry["timestamp"] = gr.timestamp
	gr.functionRegistry["format_time"] = gr.formatTime

	// Formatting operations
	gr.functionRegistry["sprintf"] = gr.sprintf

	// Print operations
	gr.functionRegistry["print"] = gr.print

	// Introspection methods
	gr.functionRegistry["GetModules"] = gr.getModulesWrapper
	gr.functionRegistry["GetAllFunctions"] = gr.getAllFunctionsWrapper
	gr.functionRegistry["GetModuleFunctions"] = gr.getModuleFunctionsWrapper
}

// Initialize sets up the Go runtime
func (gr *GoRuntime) Initialize() error {
	gr.ready = true
	return nil
}

// GetName returns the name of the language runtime
func (gr *GoRuntime) GetName() string {
	return "go"
}

// IsReady checks if the runtime is ready for execution
func (gr *GoRuntime) IsReady() bool {
	return gr.ready
}

// ExecuteFunction calls a function in the Go runtime
func (gr *GoRuntime) ExecuteFunction(name string, args []interface{}) (interface{}, error) {
	if !gr.ready {
		return nil, errors.NewRuntimeError("go", "RUNTIME_NOT_INITIALIZED", "Go runtime is not initialized")
	}

	fn, exists := gr.functionRegistry[name]
	if !exists {
		return nil, errors.NewRuntimeError("go", "FUNCTION_NOT_FOUND", fmt.Sprintf("function '%s' not found in Go runtime", name))
	}

	if gr.verbose {
		fmt.Printf("DEBUG: GoRuntime.ExecuteFunction called with %s, args: %v\n", name, args)
	}

	result, err := fn(args)
	if err != nil {
		return nil, errors.NewRuntimeError("go", "EXECUTION_ERROR", fmt.Sprintf("error executing Go function '%s': %v", name, err))
	}

	return result, nil
}

// ExecuteFunctionMultiple calls a function in the Go runtime and returns multiple values
func (gr *GoRuntime) ExecuteFunctionMultiple(functionName string, args ...interface{}) ([]interface{}, error) {
	result, err := gr.ExecuteFunction(functionName, args)
	if err != nil {
		return nil, err
	}

	// For Go runtime, we typically return single values, but we can wrap it in a slice
	return []interface{}{result}, nil
}

// Eval is not supported for Go runtime (functions are called directly)
func (gr *GoRuntime) Eval(code string) (interface{}, error) {
	return nil, errors.NewRuntimeError("go", "EVAL_NOT_SUPPORTED", "Go runtime does not support arbitrary code evaluation")
}

// ExecuteBatch executes Go code in batch mode
func (gr *GoRuntime) ExecuteBatch(code string) error {
	_, err := gr.Eval(code)
	return err
}

// SetVariable is not supported for Go runtime
func (gr *GoRuntime) SetVariable(name string, value interface{}) error {
	return errors.NewRuntimeError("go", "VARIABLES_NOT_SUPPORTED", "Go runtime does not support variable storage")
}

// GetVariable is not supported for Go runtime
func (gr *GoRuntime) GetVariable(name string) (interface{}, error) {
	return nil, errors.NewRuntimeError("go", "VARIABLES_NOT_SUPPORTED", "Go runtime does not support variable retrieval")
}

// Isolate creates an isolated state for the runtime
func (gr *GoRuntime) Isolate() error {
	// Go runtime is stateless, so isolation is not needed
	return nil
}

// Cleanup releases resources used by the runtime
func (gr *GoRuntime) Cleanup() error {
	gr.ready = false
	return nil
}

// GetSupportedTypes returns the types supported by this runtime
func (gr *GoRuntime) GetSupportedTypes() []string {
	return []string{
		"string", "int", "int64", "float64", "bool",
		"[]interface{}", "map[string]interface{}",
	}
}

// Completion interface methods

// GetModules returns available modules for the Go runtime
func (gr *GoRuntime) GetModules() []string {
	modules := []string{"io", "os", "net/http", "crypto", "encoding", "math", "time", "fmt", "json", "yaml"}
	sort.Strings(modules)
	return modules
}

// GetAllFunctions returns all available functions in the Go runtime
func (gr *GoRuntime) GetAllFunctions() []string {
	var functions []string
	for fn := range gr.functionRegistry {
		functions = append(functions, fn)
	}
	return functions
}

// GetModuleFunctions returns available functions for a specific module
func (gr *GoRuntime) GetModuleFunctions(module string) []string {
	switch module {
	case "io":
		functions := []string{"read_file", "write_file"}
		sort.Strings(functions)
		return functions
	case "os":
		functions := []string{"file_exists", "list_dir", "delete_file", "exec"}
		sort.Strings(functions)
		return functions
	case "net/http":
		functions := []string{"http_get", "http_post"}
		sort.Strings(functions)
		return functions
	case "crypto":
		functions := []string{"md5", "sha256"}
		sort.Strings(functions)
		return functions
	case "encoding":
		functions := []string{"base64_encode", "base64_decode", "url_encode", "url_decode", "to_upper", "to_lower", "trim", "split", "join"}
		sort.Strings(functions)
		return functions
	case "json":
		functions := []string{"json_encode", "json_decode"}
		sort.Strings(functions)
		return functions
	case "yaml":
		functions := []string{"parse_yaml"}
		sort.Strings(functions)
		return functions
	case "math":
		functions := []string{"sin", "cos", "sqrt", "pow", "floor", "ceil"}
		sort.Strings(functions)
		return functions
	case "time":
		functions := []string{"timestamp", "format_time"}
		sort.Strings(functions)
		return functions
	case "fmt":
		functions := []string{"sprintf"}
		sort.Strings(functions)
		return functions
	default:
		return []string{}
	}
}

// GetFunctionSignature returns the signature of a function
func (gr *GoRuntime) GetFunctionSignature(module, function string) (string, error) {
	// Simplified signatures for common functions
	signatures := map[string]map[string]string{
		"io": {
			"read_file":  "read_file(path string) string",
			"write_file": "write_file(path string, content string) bool",
		},
		"os": {
			"file_exists": "file_exists(path string) bool",
			"list_dir":    "list_dir(path string) []string",
			"delete_file": "delete_file(path string) bool",
			"exec":        "exec(command string, ...args string) string",
		},
		"net/http": {
			"http_get":  "http_get(url string) string",
			"http_post": "http_post(url string, data string) string",
		},
		"crypto": {
			"md5":    "md5(data string) string",
			"sha256": "sha256(data string) string",
		},
		"encoding": {
			"base64_encode": "base64_encode(data string) string",
			"base64_decode": "base64_decode(data string) string",
			"url_encode":    "url_encode(data string) string",
			"url_decode":    "url_decode(data string) string",
			"to_upper":      "to_upper(data string) string",
			"to_lower":      "to_lower(data string) string",
			"trim":          "trim(data string) string",
			"split":         "split(data string, separator string) []string",
			"join":          "join(elements []string, separator string) string",
		},
		"json": {
			"json_encode": "json_encode(data interface{}) string",
			"json_decode": "json_decode(data string) interface{}",
		},
		"yaml": {
			"parse_yaml": "parse_yaml(data string) interface{}",
		},
		"math": {
			"sin":   "sin(x float64) float64",
			"cos":   "cos(x float64) float64",
			"sqrt":  "sqrt(x float64) float64",
			"pow":   "pow(x float64, y float64) float64",
			"floor": "floor(x float64) float64",
			"ceil":  "ceil(x float64) float64",
		},
		"time": {
			"timestamp":   "timestamp() int64",
			"format_time": "format_time(timestamp int64, format string) string",
		},
		"fmt": {
			"sprintf": "sprintf(format string, args ...interface{}) string",
		},
	}

	if moduleSigs, ok := signatures[module]; ok {
		if sig, ok := moduleSigs[function]; ok {
			return sig, nil
		}
	}

	return "", errors.NewRuntimeError("go", "SIGNATURE_NOT_FOUND", fmt.Sprintf("signature for %s.%s not found", module, function))
}

// GetGlobalVariables returns available global variables
func (gr *GoRuntime) GetGlobalVariables() []string {
	return []string{}
}

// GetCompletionSuggestions returns completion suggestions for a given input
func (gr *GoRuntime) GetCompletionSuggestions(input string) []string {
	var suggestions []string

	// If input contains a dot, suggest functions for that module
	if strings.Contains(input, ".") {
		parts := strings.Split(input, ".")
		if len(parts) == 2 {
			module := parts[0]
			prefix := parts[1]
			functions := gr.GetModuleFunctions(module)
			for _, fn := range functions {
				if strings.HasPrefix(fn, prefix) {
					suggestions = append(suggestions, fmt.Sprintf("%s.%s", module, fn))
				}
			}
		}
	} else {
		// Suggest modules
		modules := gr.GetModules()
		for _, module := range modules {
			if strings.HasPrefix(module, input) {
				suggestions = append(suggestions, module+".")
			}
		}
	}

	return suggestions
}

// GetUserDefinedFunctions returns functions defined by the user during the session
func (gr *GoRuntime) GetUserDefinedFunctions() []string {
	// Go runtime doesn't support user-defined functions in the same way
	return []string{}
}

// GetImportedModules returns modules that have been imported during the session
func (gr *GoRuntime) GetImportedModules() []string {
	// Go runtime doesn't track imported modules like Python/Lua
	return []string{}
}

// GetDynamicCompletions returns completions based on current runtime state
func (gr *GoRuntime) GetDynamicCompletions(input string) ([]string, error) {
	return gr.GetCompletionSuggestions(input), nil
}

// GetObjectProperties returns properties and methods of a runtime object
func (gr *GoRuntime) GetObjectProperties(objectName string) ([]string, error) {
	// Go runtime doesn't support object introspection like Python/Lua
	return []string{}, nil
}

// GetFunctionParameters returns parameter names and types for a function
func (gr *GoRuntime) GetFunctionParameters(functionName string) ([]runtime.FunctionParameter, error) {
	// Simplified parameter info for known functions
	switch functionName {
	case "read_file":
		return []runtime.FunctionParameter{{Name: "path", Type: "string"}}, nil
	case "write_file":
		return []runtime.FunctionParameter{{Name: "path", Type: "string"}, {Name: "content", Type: "string"}}, nil
	case "file_exists":
		return []runtime.FunctionParameter{{Name: "path", Type: "string"}}, nil
	case "exec":
		return []runtime.FunctionParameter{{Name: "command", Type: "string"}, {Name: "args", Type: "..."}}, nil
	case "sin", "cos", "sqrt", "floor", "ceil":
		return []runtime.FunctionParameter{{Name: "x", Type: "number"}}, nil
	case "pow":
		return []runtime.FunctionParameter{{Name: "x", Type: "number"}, {Name: "y", Type: "number"}}, nil
	case "http_get":
		return []runtime.FunctionParameter{{Name: "url", Type: "string"}}, nil
	case "http_post":
		return []runtime.FunctionParameter{{Name: "url", Type: "string"}, {Name: "data", Type: "string"}}, nil
	case "md5", "sha256", "base64_encode", "base64_decode", "url_encode", "url_decode", "to_upper", "to_lower", "trim":
		return []runtime.FunctionParameter{{Name: "data", Type: "string"}}, nil
	case "split":
		return []runtime.FunctionParameter{{Name: "data", Type: "string"}, {Name: "separator", Type: "string"}}, nil
	case "join":
		return []runtime.FunctionParameter{{Name: "elements", Type: "[]string"}, {Name: "separator", Type: "string"}}, nil
	case "json_encode":
		return []runtime.FunctionParameter{{Name: "data", Type: "interface{}"}}, nil
	case "json_decode":
		return []runtime.FunctionParameter{{Name: "data", Type: "string"}}, nil
	case "parse_yaml":
		return []runtime.FunctionParameter{{Name: "data", Type: "string"}}, nil
	case "timestamp":
		return []runtime.FunctionParameter{}, nil
	case "format_time":
		return []runtime.FunctionParameter{{Name: "timestamp", Type: "number"}, {Name: "format", Type: "string"}}, nil
	case "sprintf":
		return []runtime.FunctionParameter{{Name: "format", Type: "string"}, {Name: "args", Type: "..."}}, nil
	default:
		return []runtime.FunctionParameter{}, nil
	}
}

// UpdateCompletionContext updates the completion context after code execution
func (gr *GoRuntime) UpdateCompletionContext(executedCode string, result interface{}) error {
	// Go runtime doesn't need to update completion context
	return nil
}

// RefreshRuntimeState refreshes the runtime state for completion
func (gr *GoRuntime) RefreshRuntimeState() error {
	// Go runtime is stateless, no refresh needed
	return nil
}

// GetRuntimeObjects returns all objects currently available in the runtime
func (gr *GoRuntime) GetRuntimeObjects() map[string]interface{} {
	// Go runtime doesn't maintain runtime objects like Python/Lua
	return make(map[string]interface{})
}

// State management methods

// State management methods removed for simplified beta version

// Methods already exist in other parts of the file - removed duplicates

// ===== Go Standard Library Function Implementations =====

// File operations

// readFile reads the contents of a file
func (gr *GoRuntime) readFile(args []interface{}) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("read_file expects 1 argument, got %d", len(args))
	}

	path, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("read_file expects string path, got %T", args[0])
	}

	content, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file '%s': %v", path, err)
	}

	return string(content), nil
}

// writeFile writes content to a file
func (gr *GoRuntime) writeFile(args []interface{}) (interface{}, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("write_file expects 2 arguments, got %d", len(args))
	}

	path, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("write_file expects string path, got %T", args[0])
	}

	content, ok := args[1].(string)
	if !ok {
		return nil, fmt.Errorf("write_file expects string content, got %T", args[1])
	}

	err := ioutil.WriteFile(path, []byte(content), 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to write file '%s': %v", path, err)
	}

	return true, nil
}

// fileExists checks if a file exists
func (gr *GoRuntime) fileExists(args []interface{}) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("file_exists expects 1 argument, got %d", len(args))
	}

	path, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("file_exists expects string path, got %T", args[0])
	}

	_, err := os.Stat(path)
	return err == nil, nil
}

// listDir lists files in a directory
func (gr *GoRuntime) listDir(args []interface{}) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("list_dir expects 1 argument, got %d", len(args))
	}

	path, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("list_dir expects string path, got %T", args[0])
	}

	files, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory '%s': %v", path, err)
	}

	var result []interface{}
	for _, file := range files {
		result = append(result, file.Name())
	}

	return result, nil
}

// deleteFile deletes a file
func (gr *GoRuntime) deleteFile(args []interface{}) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("delete_file expects 1 argument, got %d", len(args))
	}

	path, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("delete_file expects string path, got %T", args[0])
	}

	err := os.Remove(path)
	if err != nil {
		return nil, fmt.Errorf("failed to delete file '%s': %v", path, err)
	}

	return true, nil
}

// execCommand executes a system command
func (gr *GoRuntime) execCommand(args []interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("exec expects at least 1 argument (command), got %d", len(args))
	}

	// Extract command and arguments
	var cmdArgs []string
	var cmdName string

	// Handle different argument formats
	switch len(args) {
	case 1:
		// Single argument: command string
		cmdStr, ok := args[0].(string)
		if !ok {
			return nil, fmt.Errorf("exec expects string command, got %T", args[0])
		}
		// For complex shell commands with pipes, quotes, etc., use sh -c
		if strings.Contains(cmdStr, "|") || strings.Contains(cmdStr, "&") || strings.Contains(cmdStr, ">") ||
			strings.Contains(cmdStr, "<") || strings.Contains(cmdStr, "'") || strings.Contains(cmdStr, "\"") {
			cmdName = "sh"
			cmdArgs = []string{"-c", cmdStr}
		} else {
			// Simple command splitting (basic implementation)
			cmdArgs = strings.Fields(cmdStr)
			if len(cmdArgs) == 0 {
				return nil, fmt.Errorf("exec: empty command")
			}
			cmdName = cmdArgs[0]
			cmdArgs = cmdArgs[1:]
		}
	default:
		// Multiple arguments: command name + arguments
		var ok bool
		cmdName, ok = args[0].(string)
		if !ok {
			return nil, fmt.Errorf("exec expects string command name, got %T", args[0])
		}

		for i := 1; i < len(args); i++ {
			argStr, ok := args[i].(string)
			if !ok {
				// Convert non-string arguments to string
				argStr = fmt.Sprintf("%v", args[i])
			}
			cmdArgs = append(cmdArgs, argStr)
		}
	}

	// Create and execute the command
	cmd := exec.Command(cmdName, cmdArgs...)

	// Capture both stdout and stderr
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Return the error along with any output that was captured
		if len(output) > 0 {
			return nil, fmt.Errorf("exec failed: %v\nOutput: %s", err, string(output))
		}
		return nil, fmt.Errorf("exec failed: %v", err)
	}

	return string(output), nil
}

// HTTP operations

// httpGet performs an HTTP GET request
func (gr *GoRuntime) httpGet(args []interface{}) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("http_get expects 1 argument, got %d", len(args))
	}

	url, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("http_get expects string URL, got %T", args[0])
	}

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("HTTP GET request failed: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	return string(body), nil
}

// httpPost performs an HTTP POST request
func (gr *GoRuntime) httpPost(args []interface{}) (interface{}, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("http_post expects 2 arguments, got %d", len(args))
	}

	url, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("http_post expects string URL, got %T", args[0])
	}

	data, ok := args[1].(string)
	if !ok {
		return nil, fmt.Errorf("http_post expects string data, got %T", args[1])
	}

	resp, err := http.Post(url, "application/json", strings.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("HTTP POST request failed: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	return string(body), nil
}

// Cryptographic operations

// md5Hash computes MD5 hash of data
func (gr *GoRuntime) md5Hash(args []interface{}) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("md5 expects 1 argument, got %d", len(args))
	}

	data, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("md5 expects string data, got %T", args[0])
	}

	hash := md5.Sum([]byte(data))
	return hex.EncodeToString(hash[:]), nil
}

// sha256Hash computes SHA256 hash of data
func (gr *GoRuntime) sha256Hash(args []interface{}) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("sha256 expects 1 argument, got %d", len(args))
	}

	data, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("sha256 expects string data, got %T", args[0])
	}

	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:]), nil
}

// String operations

// base64Encode encodes data using base64
func (gr *GoRuntime) base64Encode(args []interface{}) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("base64_encode expects 1 argument, got %d", len(args))
	}

	data, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("base64_encode expects string data, got %T", args[0])
	}

	encoded := base64.StdEncoding.EncodeToString([]byte(data))
	return encoded, nil
}

// base64Decode decodes base64 data
func (gr *GoRuntime) base64Decode(args []interface{}) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("base64_decode expects 1 argument, got %d", len(args))
	}

	data, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("base64_decode expects string data, got %T", args[0])
	}

	decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, fmt.Errorf("base64 decode failed: %v", err)
	}

	return string(decoded), nil
}

// urlEncode URL-encodes a string
func (gr *GoRuntime) urlEncode(args []interface{}) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("url_encode expects 1 argument, got %d", len(args))
	}

	data, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("url_encode expects string data, got %T", args[0])
	}

	// Simple URL encoding - replace spaces and special characters
	encoded := strings.ReplaceAll(data, " ", "%20")
	encoded = strings.ReplaceAll(encoded, "&", "%26")
	encoded = strings.ReplaceAll(encoded, "=", "%3D")
	encoded = strings.ReplaceAll(encoded, "?", "%3F")

	return encoded, nil
}

// urlDecode URL-decodes a string
func (gr *GoRuntime) urlDecode(args []interface{}) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("url_decode expects 1 argument, got %d", len(args))
	}

	data, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("url_decode expects string data, got %T", args[0])
	}

	// Use Go's url.QueryUnescape for proper URL decoding
	decoded, err := url.QueryUnescape(data)
	if err != nil {
		return nil, fmt.Errorf("url decode failed: %v", err)
	}

	return decoded, nil
}

// toUpper converts string to uppercase
func (gr *GoRuntime) toUpper(args []interface{}) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("to_upper expects 1 argument, got %d", len(args))
	}

	data, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("to_upper expects string data, got %T", args[0])
	}

	return strings.ToUpper(data), nil
}

// toLower converts string to lowercase
func (gr *GoRuntime) toLower(args []interface{}) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("to_lower expects 1 argument, got %d", len(args))
	}

	data, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("to_lower expects string data, got %T", args[0])
	}

	return strings.ToLower(data), nil
}

// trim removes whitespace from both ends of string
func (gr *GoRuntime) trim(args []interface{}) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("trim expects 1 argument, got %d", len(args))
	}

	data, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("trim expects string data, got %T", args[0])
	}

	return strings.TrimSpace(data), nil
}

// split splits string by separator
func (gr *GoRuntime) split(args []interface{}) (interface{}, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("split expects 2 arguments, got %d", len(args))
	}

	data, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("split expects string data, got %T", args[0])
	}

	separator, ok := args[1].(string)
	if !ok {
		return nil, fmt.Errorf("split expects string separator, got %T", args[1])
	}

	parts := strings.Split(data, separator)
	result := make([]interface{}, len(parts))
	for i, part := range parts {
		result[i] = part
	}
	return result, nil
}

// join joins elements with separator
func (gr *GoRuntime) join(args []interface{}) (interface{}, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("join expects 2 arguments, got %d", len(args))
	}

	elements, ok := args[0].([]interface{})
	if !ok {
		return nil, fmt.Errorf("join expects []interface{} elements, got %T", args[0])
	}

	separator, ok := args[1].(string)
	if !ok {
		return nil, fmt.Errorf("join expects string separator, got %T", args[1])
	}

	// Convert []interface{} to []string
	strElements := make([]string, len(elements))
	for i, elem := range elements {
		strElements[i] = fmt.Sprintf("%v", elem)
	}

	return strings.Join(strElements, separator), nil
}

// jsonEncode encodes data to JSON string
func (gr *GoRuntime) jsonEncode(args []interface{}) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("json_encode expects 1 argument, got %d", len(args))
	}

	data := args[0]

	result, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("json encode failed: %v", err)
	}

	return string(result), nil
}

// jsonDecode decodes JSON string to interface{}
func (gr *GoRuntime) jsonDecode(args []interface{}) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("json_decode expects 1 argument, got %d", len(args))
	}

	data, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("json_decode expects string data, got %T", args[0])
	}

	var result interface{}
	err := json.Unmarshal([]byte(data), &result)
	if err != nil {
		return nil, fmt.Errorf("json decode failed: %v", err)
	}

	return result, nil
}

// parseYAML parses YAML string to interface{}
func (gr *GoRuntime) parseYAML(args []interface{}) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("parse_yaml expects 1 argument, got %d", len(args))
	}

	data, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("parse_yaml expects string data, got %T", args[0])
	}

	// Simple YAML parsing - convert YAML to JSON first, then parse as JSON
	// This is a basic implementation that handles simple YAML cases
	lines := strings.Split(data, "\n")
	result := make(map[string]interface{})
	currentMap := result

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Handle key-value pairs
		if strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])

				if value == "" {
					// Start a new nested object
					newMap := make(map[string]interface{})
					currentMap[key] = newMap
					currentMap = newMap
				} else {
					// Parse the value
					if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
						// String value
						currentMap[key] = strings.Trim(value, "\"")
					} else if value == "true" {
						currentMap[key] = true
					} else if value == "false" {
						currentMap[key] = false
					} else if value == "null" || value == "~" {
						currentMap[key] = nil
					} else {
						// Try to parse as number
						if num, err := strconv.ParseInt(value, 10, 64); err == nil {
							currentMap[key] = num
						} else if num, err := strconv.ParseFloat(value, 64); err == nil {
							currentMap[key] = num
						} else {
							// Keep as string
							currentMap[key] = value
						}
					}
				}
			}
		}
	}

	return result, nil
}

// Math operations

// sin computes sine of x
func (gr *GoRuntime) sin(args []interface{}) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("sin expects 1 argument, got %d", len(args))
	}

	x, ok := gr.toFloat64(args[0])
	if !ok {
		return nil, fmt.Errorf("sin expects numeric argument, got %T", args[0])
	}

	return math.Sin(x), nil
}

// cos computes cosine of x
func (gr *GoRuntime) cos(args []interface{}) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("cos expects 1 argument, got %d", len(args))
	}

	x, ok := gr.toFloat64(args[0])
	if !ok {
		return nil, fmt.Errorf("cos expects numeric argument, got %T", args[0])
	}

	return math.Cos(x), nil
}

// sqrt computes square root of x
func (gr *GoRuntime) sqrt(args []interface{}) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("sqrt expects 1 argument, got %d", len(args))
	}

	x, ok := gr.toFloat64(args[0])
	if !ok {
		return nil, fmt.Errorf("sqrt expects numeric argument, got %T", args[0])
	}

	return math.Sqrt(x), nil
}

// pow computes x^y
func (gr *GoRuntime) pow(args []interface{}) (interface{}, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("pow expects 2 arguments, got %d", len(args))
	}

	x, ok := gr.toFloat64(args[0])
	if !ok {
		return nil, fmt.Errorf("pow expects numeric base, got %T", args[0])
	}

	y, ok := gr.toFloat64(args[1])
	if !ok {
		return nil, fmt.Errorf("pow expects numeric exponent, got %T", args[1])
	}

	return math.Pow(x, y), nil
}

// floor computes floor of x
func (gr *GoRuntime) floor(args []interface{}) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("floor expects 1 argument, got %d", len(args))
	}

	x, ok := gr.toFloat64(args[0])
	if !ok {
		return nil, fmt.Errorf("floor expects numeric argument, got %T", args[0])
	}

	return math.Floor(x), nil
}

// ceil computes ceiling of x
func (gr *GoRuntime) ceil(args []interface{}) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("ceil expects 1 argument, got %d", len(args))
	}

	x, ok := gr.toFloat64(args[0])
	if !ok {
		return nil, fmt.Errorf("ceil expects numeric argument, got %T", args[0])
	}

	return math.Ceil(x), nil
}

// Time operations

// timestamp returns current Unix timestamp
func (gr *GoRuntime) timestamp(args []interface{}) (interface{}, error) {
	return time.Now().Unix(), nil
}

// formatTime formats a timestamp according to format string
func (gr *GoRuntime) formatTime(args []interface{}) (interface{}, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("format_time expects 2 arguments, got %d", len(args))
	}

	timestamp, ok := args[0].(int64)
	if !ok {
		// Try to convert from other numeric types
		if ts, ok := gr.toFloat64(args[0]); ok {
			timestamp = int64(ts)
		} else {
			return nil, fmt.Errorf("format_time expects int64 timestamp, got %T", args[0])
		}
	}

	format, ok := args[1].(string)
	if !ok {
		return nil, fmt.Errorf("format_time expects string format, got %T", args[1])
	}

	t := time.Unix(timestamp, 0)
	return t.Format(format), nil
}

// Formatting operations

// sprintf formats a string using Go's fmt.Sprintf
func (gr *GoRuntime) sprintf(args []interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("sprintf expects at least 1 argument, got %d", len(args))
	}

	format, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("sprintf expects string format, got %T", args[0])
	}

	// Convert remaining arguments to the format expected by fmt.Sprintf
	fmtArgs := make([]interface{}, len(args)-1)
	for i, arg := range args[1:] {
		fmtArgs[i] = arg
	}

	return fmt.Sprintf(format, fmtArgs...), nil
}

// print prints arguments to stdout
func (gr *GoRuntime) print(args []interface{}) (interface{}, error) {
	for i, arg := range args {
		if i > 0 {
			fmt.Print(" ")
		}
		fmt.Print(arg)
	}
	fmt.Println()
	return nil, nil
}

// Helper functions

// toFloat64 converts various numeric types to float64
func (gr *GoRuntime) toFloat64(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case int32:
		return float64(v), true
	case string:
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f, true
		}
	}
	return 0, false
}

// ===== Wrapper methods for introspection =====

// getModulesWrapper wraps GetModules for the function registry
func (gr *GoRuntime) getModulesWrapper(args []interface{}) (interface{}, error) {
	modules := gr.GetModules()
	result := make([]interface{}, len(modules))
	for i, module := range modules {
		result[i] = module
	}
	return result, nil
}

// getAllFunctionsWrapper wraps GetAllFunctions for the function registry
func (gr *GoRuntime) getAllFunctionsWrapper(args []interface{}) (interface{}, error) {
	functions := gr.GetAllFunctions()
	result := make([]interface{}, len(functions))
	for i, fn := range functions {
		result[i] = fn
	}
	return result, nil
}

// ExecuteCodeBlock executes a Go code block and captures variables
func (gr *GoRuntime) ExecuteCodeBlock(code string) (interface{}, error) {
	// Go runtime doesn't support arbitrary code execution
	return nil, errors.NewRuntimeError("go", "CODE_EXECUTION_NOT_SUPPORTED", "Go runtime does not support arbitrary code execution")
}

// ExecuteCodeBlockWithVariables выполняет код с сохранением указанных переменных
func (gr *GoRuntime) ExecuteCodeBlockWithVariables(code string, variables []string) (interface{}, error) {
	// Go runtime doesn't support arbitrary code execution or variables
	return nil, errors.NewRuntimeError("go", "CODE_EXECUTION_NOT_SUPPORTED", "Go runtime does not support arbitrary code execution or variables")
}

// SetExecutionTimeout sets the timeout for Go execution
func (gr *GoRuntime) SetExecutionTimeout(timeout time.Duration) {
	// Go runtime doesn't use timeout for function calls
}

// GetExecutionTimeout returns the current timeout for Go execution
func (gr *GoRuntime) GetExecutionTimeout() time.Duration {
	// Go runtime doesn't use timeout for function calls
	return 0
}

// getModuleFunctionsWrapper wraps GetModuleFunctions for the function registry
func (gr *GoRuntime) getModuleFunctionsWrapper(args []interface{}) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("GetModuleFunctions expects 1 argument, got %d", len(args))
	}

	module, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("GetModuleFunctions expects string module name, got %T", args[0])
	}

	functions := gr.GetModuleFunctions(module)
	result := make([]interface{}, len(functions))
	for i, fn := range functions {
		result[i] = fn
	}
	return result, nil
}
