package python

import (
	"encoding/json"
	"fmt"
	"strings"

	"funterm/errors"
)

func (pr *PythonRuntime) ensureModuleImported(functionName string) error {
	parts := strings.Split(functionName, ".")
	if len(parts) > 1 {
		moduleName := parts[0]
		// Basic check for valid identifier
		if moduleName != "" && isIdentifier(moduleName) {
			importCode := fmt.Sprintf(`
try:
    globals()['%s']
except KeyError:
    try:
        import %s
    except (ImportError, ModuleNotFoundError):
        pass
`, moduleName, moduleName)
			_, err := pr.sendAndAwait(importCode)
			return err
		}
	}
	return nil
}

// isIdentifier checks if a string is a valid Python identifier.
func isIdentifier(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		if i == 0 {
			if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_') {
				return false
			}
		} else {
			if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_') {
				return false
			}
		}
	}
	return true
}

// ExecuteFunction calls a function in the Python runtime
func (pr *PythonRuntime) ExecuteFunction(name string, args []interface{}) (interface{}, error) {
	pr.mutex.Lock()
	// Always initialize output capture for any function call
	pr.outputCapture = &strings.Builder{}
	pr.mutex.Unlock()

	if !pr.ready {
		pr.mutex.Lock()
		pr.outputCapture = nil // Clean up on error
		pr.mutex.Unlock()
		if !pr.available {
			return nil, errors.NewRuntimeError("python", "RUNTIME_UNAVAILABLE", "Python runtime is unavailable. Please install Python.")
		}
		return nil, errors.NewRuntimeError("python", "RUNTIME_NOT_INITIALIZED", "runtime is not initialized")
	}

	// Attempt to auto-import module if name suggests it (e.g., "datetime.timedelta")
	if err := pr.ensureModuleImported(name); err != nil {
		if pr.verbose {
			fmt.Printf("DEBUG: Error during auto-import check for '%s': %v\n", name, err)
		}
		// Don't return error, let the actual function call fail if module is missing
	}

	// Ensure json is imported before using it
	if err := pr.ensureJSONImported(); err != nil {
		if pr.verbose {
			fmt.Printf("DEBUG: Failed to ensure json import in ExecuteFunction: %v\n", err)
		}
	}

	argsJSON, err := json.Marshal(args)
	if err != nil {
		return nil, errors.NewRuntimeError("python", "INVALID_ARGUMENT", fmt.Sprintf("failed to marshal arguments: %v", err))
	}

	var code string
	if name == "print" {
		// For print function, just execute it directly without json wrapping
		code = fmt.Sprintf("%s(*json.loads('''%s'''))", name, string(argsJSON))
	} else {
		// For other functions, just execute and let print() output be visible
		if pr.verbose {
			fmt.Printf("DEBUG: Python function call: %s with args: %s\n", name, string(argsJSON))
		}
		// Generate unique marker for this execution
		executionID++
		uniqueMarker := fmt.Sprintf("%s-%d", EndOfOutputMarker, executionID)

		isKwargs := false
		isMixedArgs := false
		if len(args) == 1 {
			// Check if this is our mixed args structure
			if mixedArgs, ok := args[0].(map[string]interface{}); ok {
				if _, hasPositional := mixedArgs["positional"]; hasPositional {
					if _, hasKeyword := mixedArgs["keyword"]; hasKeyword {
						isMixedArgs = true
					}
				} else {
					// Regular kwargs map
					isKwargs = true
				}
			}
		}

		var callCode string
		if isMixedArgs {
			// Handle mixed positional and keyword arguments
			mixedArgs := args[0].(map[string]interface{})
			positionalJSON, _ := json.Marshal(mixedArgs["positional"])
			keywordJSON, _ := json.Marshal(mixedArgs["keyword"])
			callCode = fmt.Sprintf("%s(*json.loads('''%s'''), **json.loads('''%s'''))", name, string(positionalJSON), string(keywordJSON))
		} else if isKwargs {
			// Marshal just the map for keyword arguments
			kwargsJSON, _ := json.Marshal(args[0])
			callCode = fmt.Sprintf("%s(**json.loads('''%s'''))", name, string(kwargsJSON))
		} else {
			// Marshal all args for positional arguments
			callCode = fmt.Sprintf("%s(*json.loads('''%s'''))", name, string(argsJSON))
		}

		code = fmt.Sprintf(`
import json
_result = %s
if _result is not None:
	try:
		print(json.dumps(_result))
	except TypeError:
		print(json.dumps(str(_result)))
print('%s')
`, callCode, uniqueMarker)
		if pr.verbose {
			fmt.Printf("DEBUG: Generated Python code: %s\n", code)
		}
	}

	// Call sendAndAwait with the execution ID to ensure synchronization
	output, err := pr.sendAndAwaitWithID(code, executionID)
	if err != nil {
		return nil, pr.enhanceError(err.Error(), code)
	}

	if pr.verbose {
		fmt.Printf("DEBUG: Python execution output: '%s'\n", output)
	}

	if name == "print" {
		// For print function, capture stdout output but don't return a value
		// This matches the behavior of Lua and JavaScript runtimes
		pr.mutex.Lock()
		var result interface{} = nil
		if pr.outputCapture != nil {
			capturedOutput := pr.outputCapture.String()
			if capturedOutput != "" {
				// Don't print to console here - let the engine handle output display
				// Don't return any value for print function to match other runtimes
				result = nil
			}
		}
		// Don't reset outputCapture here - let the caller handle it via GetCapturedOutput()
		pr.mutex.Unlock()
		return result, nil
	}

	if pr.verbose {
		fmt.Printf("DEBUG: About to check if output is empty: '%s'\n", output)
	}
	if output == "" || output == "null" {
		if pr.verbose {
			fmt.Printf("DEBUG: Output is empty or null, returning nil\n")
		}
		// Don't reset outputCapture here - let the caller handle it via GetCapturedOutput()
		return nil, nil
	}

	var result interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		if pr.verbose {
			fmt.Printf("DEBUG: JSON unmarshal failed for '%s': %v\n", output, err)
		}
		// If it's not valid JSON, it might be an error message or other string output.
		// Don't reset outputCapture here - let the caller handle it via GetCapturedOutput()
		return output, nil
	}

	if pr.verbose {
		fmt.Printf("DEBUG: JSON unmarshal successful: %v\n", result)
	}

	// Check if the result represents None/null from Python
	if result == nil || (output == "null") {
		if pr.verbose {
			fmt.Printf("DEBUG: Result is nil or null, returning nil\n")
		}
		// Function returned None or null, treat as no return value
		// Don't reset outputCapture here - let the caller handle it via GetCapturedOutput()
		return nil, nil
	}

	if pr.verbose {
		fmt.Printf("DEBUG: Returning final result: %v\n", result)
	}
	// Don't reset outputCapture here - let the caller handle it via GetCapturedOutput()
	return result, nil
}

// ExecuteFunctionMultiple calls a function in the Python runtime and returns multiple values
func (pr *PythonRuntime) ExecuteFunctionMultiple(functionName string, args ...interface{}) ([]interface{}, error) {
	if !pr.ready {
		if !pr.available {
			return nil, errors.NewRuntimeError("python", "RUNTIME_UNAVAILABLE", "Python runtime is unavailable. Please install Python.")
		}
		return nil, errors.NewRuntimeError("python", "RUNTIME_NOT_INITIALIZED", "runtime is not initialized")
	}

	// Ensure json is imported before using it
	if err := pr.ensureJSONImported(); err != nil {
		if pr.verbose {
			fmt.Printf("DEBUG: Failed to ensure json import in ExecuteFunctionMultiple: %v\n", err)
		}
	}

	// Convert arguments to JSON
	argsJSON, err := json.Marshal(args)
	if err != nil {
		return nil, errors.NewRuntimeError("python", "INVALID_ARGUMENT", fmt.Sprintf("failed to marshal arguments: %v", err))
	}

	// Execute the function using the new persistent process method
	// For multiple return values, wrap the result in a list
	code := fmt.Sprintf("import json; result = %s(*json.loads('''%s''')); print(json.dumps(list(result) if isinstance(result, (list, tuple)) else [result]))", functionName, string(argsJSON))

	output, err := pr.sendAndAwait(code)
	if err != nil {
		// Enhance the error with context
		enhancedError := pr.enhanceError(err.Error(), code)
		return nil, enhancedError
	}

	// Parse the result
	output = strings.TrimSpace(output)
	if output == "" {
		return []interface{}{}, nil
	}

	// Try to parse as JSON first
	var result []interface{}
	if err := json.Unmarshal([]byte(output), &result); err == nil {
		return result, nil
	}

	// If not JSON and not an error, return as single item in slice
	return []interface{}{output}, nil
}

// SetVariable sets a variable in the Python runtime
func (pr *PythonRuntime) SetVariable(name string, value interface{}) error {
	if !pr.ready {
		if !pr.available {
			return errors.NewRuntimeError("python", "RUNTIME_UNAVAILABLE", "Python runtime is unavailable. Please install Python.")
		}
		return errors.NewRuntimeError("python", "RUNTIME_NOT_INITIALIZED", "runtime is not initialized")
	}

	// Store the variable locally
	pr.mutex.Lock()
	pr.variables[name] = value
	pr.mutex.Unlock()

	// Convert value to JSON
	valueJSON, err := json.Marshal(value)
	if err != nil {
		return errors.NewRuntimeError("python", "INVALID_ARGUMENT", fmt.Sprintf("failed to marshal value: %v", err))
	}

	// Ensure json is imported before using it
	if err := pr.ensureJSONImported(); err != nil {
		if pr.verbose {
			fmt.Printf("DEBUG: Failed to ensure json import: %v\n", err)
		}
	}

	// Set the variable in Python using the persistent process
	// Generate simple assignment code
	code := fmt.Sprintf("%s = json.loads('''%s''')", name, string(valueJSON))

	_, err = pr.sendAndAwait(code)
	if err != nil {
		return errors.NewRuntimeError("python", "EXECUTION_FAILED", fmt.Sprintf("failed to set variable: %v", err))
	}

	return nil
}

// executePythonCode executes Python code directly without buffering (like in Lua runtime)
func (pr *PythonRuntime) executePythonCode(code string) (string, error) {
	if pr.verbose {
		fmt.Printf("DEBUG: executePythonCode called with code: %s\n", code)

	}

	if !pr.ready {
		if !pr.available {
			return "", errors.NewRuntimeError("python", "RUNTIME_UNAVAILABLE", "Python runtime is unavailable. Please install Python.")
		}
		return "", errors.NewRuntimeError("python", "RUNTIME_NOT_INITIALIZED", "runtime is not initialized")
	}

	result, err := pr.sendAndAwait(code)
	if pr.verbose {
		fmt.Printf("DEBUG: executePythonCode - sendAndAwait result: '%s', error: %v\n", result, err)

	}

	return result, err
}

// GetVariable retrieves a variable from the Python runtime
func (pr *PythonRuntime) GetVariable(name string) (interface{}, error) {
	if pr.verbose {
		fmt.Printf("DEBUG: PythonRuntime.GetVariable called with name: %s\n", name)

		pr.mutex.RLock()
		fmt.Printf("DEBUG: PythonRuntime.GetVariable - Local cache contents: %v\n", pr.variables)
		pr.mutex.RUnlock()
	}

	if !pr.ready {
		if !pr.available {
			return nil, errors.NewRuntimeError("python", "RUNTIME_UNAVAILABLE", "Python runtime is unavailable. Please install Python.")
		}
		return nil, errors.NewRuntimeError("python", "RUNTIME_NOT_INITIALIZED", "runtime is not initialized")
	}

	pr.mutex.RLock()
	// First check if we have it locally
	if value, exists := pr.variables[name]; exists {
		pr.mutex.RUnlock()
		if pr.verbose {
			fmt.Printf("DEBUG: Found variable '%s' in local cache: %v\n", name, value)
		}
		return value, nil
	}
	pr.mutex.RUnlock()

	if pr.verbose {
		fmt.Printf("DEBUG: Variable '%s' not found in local cache\n", name)
	}

	// Ensure json is imported before using it
	if err := pr.ensureJSONImported(); err != nil {
		if pr.verbose {
			fmt.Printf("DEBUG: Failed to ensure json import: %v\n", err)
		}
	}

	// Before trying to get from Python, let's check what variables are actually available (only in verbose mode)
	if pr.verbose {
		listCode := "print(json.dumps([name for name in globals().keys() if not name.startswith('_')]))"
		fmt.Printf("DEBUG: Listing available Python variables with code: '%s'\n", listCode)

		availableVars, err := pr.executePythonCode(listCode)
		if err == nil {
			fmt.Printf("DEBUG: Available Python variables: %s\n", availableVars)
		} else {
			fmt.Printf("DEBUG: Failed to list available Python variables: %v\n", err)
		}
	}

	// Try to get it from Python using the persistent process
	// Generate code that prints the result
	// IMPORTANT: Don't use Eval here because it adds code to buffer!
	// We need to execute only this specific code without buffering.
	code := fmt.Sprintf("print(json.dumps(globals().get('%s')))", name)

	if pr.verbose {
		fmt.Printf("DEBUG: Executing variable retrieval code: '%s'\n", code)
	}

	// Execute the code directly without buffering
	result, err := pr.executePythonCode(code)
	if err != nil {
		if pr.verbose {
			fmt.Printf("DEBUG: Failed to get variable '%s' from Python: %v\n", name, err)
		}
		return nil, errors.NewRuntimeError("python", "EXECUTION_FAILED", fmt.Sprintf("failed to get variable: %v", err))
	}

	if pr.verbose {
		fmt.Printf("DEBUG: Raw result for variable '%s': '%s'\n", name, result)
	}

	// Convert the result
	result = strings.TrimSpace(result)
	if result == "" || result == "null" {
		if pr.verbose {
			fmt.Printf("DEBUG: Variable '%s' is null or empty\n", name)
		}
		return nil, errors.NewRuntimeError("python", "VARIABLE_NOT_FOUND", fmt.Sprintf("variable '%s' not found", name))
	}

	// Parse the JSON result
	var value interface{}
	if err := json.Unmarshal([]byte(result), &value); err != nil {
		if pr.verbose {
			fmt.Printf("DEBUG: Failed to parse JSON result for variable '%s': %v\n", name, err)
		}
		return nil, errors.NewRuntimeError("python", "EXECUTION_FAILED", fmt.Sprintf("failed to parse variable value: %v", err))
	}

	if pr.verbose {
		fmt.Printf("DEBUG: Parsed value for variable '%s': %v (type: %T)\n", name, value, value)
	}

	// Cache the result for future use (only if it's not null/None)
	if value != nil {
		pr.mutex.Lock()
		pr.variables[name] = value
		pr.mutex.Unlock()
	}

	return value, nil
}

// Eval executes arbitrary Python code.
func (pr *PythonRuntime) Eval(code string) (interface{}, error) {
	if !pr.ready {
		if !pr.available {
			return nil, errors.NewRuntimeError("python", "RUNTIME_UNAVAILABLE", "Python runtime is unavailable. Please install Python.")
		}
		return nil, errors.NewRuntimeError("python", "RUNTIME_NOT_INITIALIZED", "runtime is not initialized")
	}

	output, err := pr.sendAndAwait(code)
	if err != nil {
		return nil, pr.enhanceError(err.Error(), code)
	}

	return output, nil
}

// Execute is an alias for Eval for backward compatibility with tests.
func (pr *PythonRuntime) Execute(code string) (interface{}, error) {
	return pr.Eval(code)
}

// ExecuteBatch executes Python code in batch mode.
// For the persistent REPL, this is functionally similar to Eval,
// but it will print any output directly to stdout.
func (pr *PythonRuntime) ExecuteBatch(code string) error {
	if !pr.ready {
		if !pr.available {
			return errors.NewRuntimeError("python", "RUNTIME_UNAVAILABLE", "Python runtime is unavailable. Please install Python.")
		}
		return errors.NewRuntimeError("python", "RUNTIME_NOT_INITIALIZED", "runtime is not initialized")
	}

	output, err := pr.sendAndAwait(code)
	if err != nil {
		return pr.enhanceError(err.Error(), code)
	}

	if output != "" {
		fmt.Println(output)
	}

	return nil
}

// ExecuteCodeBlock executes a Python code block and captures variables
func (pr *PythonRuntime) ExecuteCodeBlock(code string) (interface{}, error) {
	if pr.verbose {
		fmt.Printf("DEBUG: ExecuteCodeBlock called with code: %s\n", code)
	}

	// Initialize output capture to capture print() statements
	pr.mutex.Lock()
	pr.outputCapture = &strings.Builder{}
	pr.mutex.Unlock()

	// Execute code with output capture (use sendAndAwaitWithID instead of Eval)
	executionID++
	result, err := pr.sendAndAwaitWithID(code, executionID)
	if err != nil {
		if pr.verbose {
			fmt.Printf("DEBUG: ExecuteCodeBlock error: %v\n", err)
		}
		return result, err
	}

	if pr.verbose {
		fmt.Printf("DEBUG: ExecuteCodeBlock result: %v\n", result)
	}

	// After successful execution, capture and cache variables
	if pr.verbose {
		fmt.Printf("DEBUG: Capturing variables after code block execution\n")
	}

	// Get all global variables from Python
	variablesCode := `
import json
import copy
import types
import inspect

# Create a safe serialization function
def suterm_safe_serialize(obj):
	   try:
	       # Try to serialize the object
	       return json.dumps(obj)
	   except (TypeError, ValueError):
	       # If serialization fails, return string representation
	       return str(obj)

# Function to check if an object should be included
def suterm_should_include(var_name, var_value):
	   # Skip internal variables that start with "_"
	   if var_name.startswith("_"):
	       return False
	   
	   # Skip modules
	   if isinstance(var_value, types.ModuleType):
	       return False
	   
	   # Skip functions
	   if isinstance(var_value, (types.FunctionType, types.BuiltinFunctionType, types.MethodType)):
	       return False
	   
	   # Skip classes
	   if isinstance(var_value, type):
	       return False
	   
	   # Skip our own capture variables
	   if var_name in ["suterm_safe_serialize", "suterm_should_include", "suterm_globals_dict", "suterm_global_names"]:
	       return False
	   
	   return True

# Get all global variables from Python
suterm_globals_dict = {}
# Get a list of all global variable names first to avoid "dictionary changed size during iteration"
suterm_global_names = list(globals().keys())

for var_name in suterm_global_names:
	   try:
	       var_value = globals()[var_name]
	       # Check if we should include this variable
	       if suterm_should_include(var_name, var_value):
	           # Try to serialize the value safely
	           suterm_serialized = suterm_safe_serialize(var_value)
	           suterm_globals_dict[var_name] = var_value
	   except:
	       # Skip variables that can't be processed
	       pass

print(json.dumps(suterm_globals_dict))
`

	// Temporarily disable output capture for variable retrieval
	pr.mutex.Lock()
	originalCapture := pr.outputCapture
	pr.outputCapture = nil
	pr.mutex.Unlock()

	variablesOutput, err := pr.sendAndAwait(variablesCode)

	// Restore output capture
	pr.mutex.Lock()
	pr.outputCapture = originalCapture
	pr.mutex.Unlock()

	if err != nil {
		if pr.verbose {
			fmt.Printf("DEBUG: Failed to capture variables: %v\n", err)
		}
		return result, nil // Return original result even if variable capture fails
	}

	// Parse the variables JSON
	var variables map[string]interface{}
	if err := json.Unmarshal([]byte(variablesOutput), &variables); err == nil {
		pr.mutex.Lock()
		// Update local cache with new variables
		for name, value := range variables {
			pr.variables[name] = value
			if pr.verbose {
				fmt.Printf("DEBUG: Cached variable: %s = %v\n", name, value)
			}
		}
		pr.mutex.Unlock()
	} else if pr.verbose {
		fmt.Printf("DEBUG: Failed to parse variables JSON: %v\n", err)
	}

	// Get captured output but DON'T clear it yet - let the engine handle it via GetCapturedOutput()
	pr.mutex.Lock()
	var capturedOutput string
	if pr.outputCapture != nil {
		capturedOutput = pr.outputCapture.String()
		// Don't clear outputCapture here - let the engine handle it via GetCapturedOutput()
	}
	pr.mutex.Unlock()

	if capturedOutput != "" {
		if pr.verbose {
			fmt.Printf("DEBUG: ExecuteCodeBlock captured output: '%s'\n", capturedOutput)
		}
		// Don't print to console here - let the engine handle output display
		// This matches the behavior of Lua and JavaScript runtimes
		// fmt.Print(capturedOutput)
		// if !strings.HasSuffix(capturedOutput, "\n") {
		//	fmt.Println()
		// }
	}

	// Don't clear outputCapture here - let the engine handle it via GetCapturedOutput()
	// This is important so the engine can read the captured output

	return result, nil
}

// ExecuteCodeBlockWithVariables выполняет код с сохранением указанных переменных
func (pr *PythonRuntime) ExecuteCodeBlockWithVariables(code string, variables []string) (interface{}, error) {
	if pr.verbose {
		fmt.Printf("DEBUG: ExecuteCodeBlockWithVariables called with code: %s, variables: %v\n", code, variables)
	}

	if !pr.ready {
		if !pr.available {
			return nil, errors.NewRuntimeError("python", "RUNTIME_UNAVAILABLE", "Python runtime is unavailable. Please install Python.")
		}
		return nil, errors.NewRuntimeError("python", "RUNTIME_NOT_INITIALIZED", "runtime is not initialized")
	}

	// Initialize output capture to capture print() statements
	pr.mutex.Lock()
	pr.outputCapture = &strings.Builder{}
	pr.mutex.Unlock()

	// Выполняем код с захватом вывода (используем sendAndAwaitWithID для захвата print())
	executionID++
	result, err := pr.sendAndAwaitWithID(code, executionID)
	if err != nil {
		if pr.verbose {
			fmt.Printf("DEBUG: ExecuteCodeBlockWithVariables execution error: %v\n", err)
		}
		return result, err
	}

	if pr.verbose {
		fmt.Printf("DEBUG: ExecuteCodeBlockWithVariables execution result: %v\n", result)
	}

	// После успешного выполнения, захватываем только указанные переменные
	if len(variables) > 0 {
		if pr.verbose {
			fmt.Printf("DEBUG: Capturing specified variables after code block execution: %v\n", variables)
		}

		// Создаем код для получения только указанных переменных
		var variableNames []string
		for _, v := range variables {
			variableNames = append(variableNames, fmt.Sprintf("'%s'", v))
		}

		variablesCode := fmt.Sprintf(`
import json
import copy
import types
import inspect

# Функция для безопасной сериализации объекта
def suterm_safe_serialize(obj):
    try:
        # Пытаемся сериализовать объект
        return json.dumps(obj)
    except (TypeError, ValueError):
        # Если сериализация не удалась, возвращаем строковое представление
        return str(obj)

# Функция для проверки, нужно ли включать переменную
def suterm_should_include(var_name, var_value):
    # Пропускаем внутренние переменные, которые начинаются с "_"
    if var_name.startswith("_"):
        return False
    
    # Пропускаем модули
    if isinstance(var_value, types.ModuleType):
        return False
    
    # Пропускаем функции
    if isinstance(var_value, (types.FunctionType, types.BuiltinFunctionType, types.MethodType)):
        return False
    
    # Пропускаем классы
    if isinstance(var_value, type):
        return False
    
    # Пропускаем наши собственные переменные для захвата
    if var_name in ["suterm_safe_serialize", "suterm_should_include", "suterm_globals_dict"]:
        return False
    
    # Проверяем, что переменная в списке для сохранения
    return var_name in [%s]

# Получаем указанные переменные из Python
suterm_globals_dict = {}

for var_name in [%s]:
    try:
        var_value = globals().get(var_name)
        if var_value is not None and suterm_should_include(var_name, var_value):
            # Пытаемся безопасно сериализовать значение
            suterm_serialized = suterm_safe_serialize(var_value)
            suterm_globals_dict[var_name] = var_value
    except:
        # Пропускаем переменные, которые не удалось обработать
        pass

print(json.dumps(suterm_globals_dict))
`, strings.Join(variableNames, ", "), strings.Join(variableNames, ", "))

		// Temporarily disable output capture for variable retrieval
		pr.mutex.Lock()
		originalCapture := pr.outputCapture
		pr.outputCapture = nil
		pr.mutex.Unlock()

		variablesOutput, err := pr.sendAndAwait(variablesCode)

		// Restore output capture
		pr.mutex.Lock()
		pr.outputCapture = originalCapture
		pr.mutex.Unlock()

		if err != nil {
			if pr.verbose {
				fmt.Printf("DEBUG: Failed to capture specified variables: %v\n", err)
			}
			return result, nil // Возвращаем оригинальный результат даже если захват переменных не удался
		}

		// Парсим JSON с переменными
		var variables map[string]interface{}
		if err := json.Unmarshal([]byte(variablesOutput), &variables); err == nil {
			pr.mutex.Lock()
			// Обновляем локальный кеш только с указанными переменными
			for name, value := range variables {
				pr.variables[name] = value
				if pr.verbose {
					fmt.Printf("DEBUG: Cached specified variable: %s = %v\n", name, value)
				}
			}
			pr.mutex.Unlock()
		} else if pr.verbose {
			fmt.Printf("DEBUG: Failed to parse variables JSON: %v\n", err)
		}
	}

	// Get captured output but DON'T clear it yet - let the engine handle it via GetCapturedOutput()
	pr.mutex.Lock()
	var capturedOutput string
	if pr.outputCapture != nil {
		capturedOutput = pr.outputCapture.String()
		// Don't clear outputCapture here - let the engine handle it via GetCapturedOutput()
	}
	pr.mutex.Unlock()

	if capturedOutput != "" {
		if pr.verbose {
			fmt.Printf("DEBUG: Captured output: '%s'\n", capturedOutput)
		}
		// Don't print to console here - let the engine handle output display
		// This matches the behavior of Lua and JavaScript runtimes
		// fmt.Print(capturedOutput)
		// if !strings.HasSuffix(capturedOutput, "\n") {
		//	fmt.Println()
		// }
	}

	// Don't clear outputCapture here - let the engine handle it via GetCapturedOutput()
	// This is important so the engine can read the captured output

	return result, nil
}

// GetAllVariables returns all variables from the Python runtime
func (pr *PythonRuntime) GetAllVariables() map[string]interface{} {
	pr.mutex.RLock()
	defer pr.mutex.RUnlock()

	variables := make(map[string]interface{})
	for k, v := range pr.variables {
		variables[k] = v
	}
	return variables
}
