package lua

import (
	"fmt"
	"strings"
	"time"

	"funterm/errors"

	lua "github.com/yuin/gopher-lua"
)

// ExecuteFunction calls a function in the Lua runtime
func (lr *LuaRuntime) ExecuteFunction(name string, args []interface{}) (interface{}, error) {
	lr.mu.Lock()
	defer lr.mu.Unlock()

	// Initialize output capture if not already set
	if lr.outputCapture == nil {
		lr.outputCapture = &strings.Builder{}
	}
	if !lr.ready {
		return nil, errors.NewRuntimeError("lua", "LUA_RUNTIME_NOT_INITIALIZED", "runtime is not initialized")
	}

	// Split function name to handle nested calls (e.g., "math.sin")
	parts := strings.Split(name, ".")
	if len(parts) == 0 {
		return nil, errors.NewRuntimeError("lua", "LUA_EMPTY_FUNCTION_NAME", "empty function name")
	}

	// Get the function from Lua state
	var fn lua.LValue
	if len(parts) == 1 {
		// Global function
		fn = lr.state.GetGlobal(parts[0])
	} else {
		// Nested function (e.g., math.sin)
		current := lr.state.GetGlobal(parts[0])
		for i := 1; i < len(parts)-1; i++ {
			if current.Type() != lua.LTTable {
				return nil, errors.NewRuntimeError("lua", "LUA_NOT_A_TABLE", fmt.Sprintf("'%s' is not a table", strings.Join(parts[:i], ".")))
			}
			current = current.(*lua.LTable).RawGetString(parts[i])
		}

		if current.Type() != lua.LTTable {
			return nil, errors.NewRuntimeError("lua", "LUA_NOT_A_TABLE", fmt.Sprintf("'%s' is not a table", strings.Join(parts[:len(parts)-1], ".")))
		}

		fn = current.(*lua.LTable).RawGetString(parts[len(parts)-1])
	}

	if fn.Type() != lua.LTFunction {
		return nil, errors.NewRuntimeError("lua", "LUA_NOT_A_FUNCTION", fmt.Sprintf("'%s' is not a function", name))
	}

	// Push function and arguments onto stack
	lr.state.Push(fn)

	for _, arg := range args {
		luaValue, err := lr.GoToLua(arg)
		if err != nil {
			return nil, errors.NewRuntimeError("lua", "LUA_ARGUMENT_CONVERSION_ERROR", fmt.Sprintf("argument conversion error: %v", err))
		}
		lr.state.Push(luaValue)
	}

	// Call the function
	if err := lr.state.PCall(len(args), lua.MultRet, nil); err != nil {
		return nil, errors.NewRuntimeError("lua", "LUA_FUNCTION_CALL_ERROR", fmt.Sprintf("function call error: %v", err))
	}

	// Get return values
	retCount := lr.state.GetTop()

	if retCount == 0 {
		return nil, nil
	}

	// Return only the first return value (for backward compatibility)
	result := lr.state.Get(-1)
	lr.state.Pop(retCount)

	// Check if we have captured any output
	output := lr.outputCapture.String()
	var capturedOutput string
	if output != "" {
		// Capture all lines except the last empty one
		lines := strings.Split(strings.TrimSpace(output), "\n")
		capturedOutput = strings.Join(lines, "\n")
		// Don't print to console here - let the engine handle output display
		// for _, line := range lines {
		//	fmt.Println(line)
		// }
	}

	// Debug output
	// fmt.Printf("DEBUG: ExecuteFunction - output buffer: '%s', capturedOutput: '%s'\n", output, capturedOutput)

	// Reset output capture
	lr.outputCapture = nil

	goResult := lr.convertLuaValueToGo(result)

	// Debug output
	// fmt.Printf("DEBUG: ExecuteFunction - goResult: %v, capturedOutput not empty: %v, goResult is nil: %v\n", goResult, capturedOutput != "", goResult == nil)

	// If we have captured output and the main result is nil, return the captured output instead
	if capturedOutput != "" && goResult == nil {
		// fmt.Printf("DEBUG: ExecuteFunction - returning captured output: '%s'\n", capturedOutput)
		return capturedOutput, nil
	}

	// fmt.Printf("DEBUG: ExecuteFunction - returning goResult: %v\n", goResult)
	return goResult, nil
}

// ExecuteFunctionMultiple calls a function in the Lua runtime and returns multiple values
func (lr *LuaRuntime) ExecuteFunctionMultiple(functionName string, args ...interface{}) ([]interface{}, error) {
	lr.mu.Lock()
	defer lr.mu.Unlock()

	// Initialize output capture
	lr.outputCapture = &strings.Builder{}
	if !lr.ready {
		return nil, errors.NewRuntimeError("lua", "LUA_RUNTIME_NOT_INITIALIZED", "runtime is not initialized")
	}

	// Split function name to handle nested calls (e.g., "math.sin")
	parts := strings.Split(functionName, ".")
	if len(parts) == 0 {
		return nil, errors.NewRuntimeError("lua", "LUA_EMPTY_FUNCTION_NAME", "empty function name")
	}

	// Get the function from Lua state
	var fn lua.LValue
	if len(parts) == 1 {
		// Global function
		fn = lr.state.GetGlobal(parts[0])
	} else {
		// Nested function (e.g., math.sin)
		current := lr.state.GetGlobal(parts[0])
		for i := 1; i < len(parts)-1; i++ {
			if current.Type() != lua.LTTable {
				return nil, errors.NewRuntimeError("lua", "LUA_NOT_A_TABLE", fmt.Sprintf("'%s' is not a table", strings.Join(parts[:i], ".")))
			}
			current = current.(*lua.LTable).RawGetString(parts[i])
		}

		if current.Type() != lua.LTTable {
			return nil, errors.NewRuntimeError("lua", "LUA_NOT_A_TABLE", fmt.Sprintf("'%s' is not a table", strings.Join(parts[:len(parts)-1], ".")))
		}

		fn = current.(*lua.LTable).RawGetString(parts[len(parts)-1])
	}

	if fn.Type() != lua.LTFunction {
		return nil, errors.NewRuntimeError("lua", "LUA_NOT_A_FUNCTION", fmt.Sprintf("'%s' is not a function", functionName))
	}

	// Push function and arguments onto stack
	lr.state.Push(fn)

	for _, arg := range args {
		luaValue, err := lr.GoToLua(arg)
		if err != nil {
			return nil, errors.NewRuntimeError("lua", "LUA_ARGUMENT_CONVERSION_ERROR", fmt.Sprintf("argument conversion error: %v", err))
		}
		lr.state.Push(luaValue)
	}

	// Call the function
	if err := lr.state.PCall(len(args), lua.MultRet, nil); err != nil {
		return nil, errors.NewRuntimeError("lua", "LUA_FUNCTION_CALL_ERROR", fmt.Sprintf("function call error: %v", err))
	}

	// Get return values
	retCount := lr.state.GetTop()

	if retCount == 0 {
		return []interface{}{}, nil
	}

	// Handle multiple return values
	results := make([]interface{}, retCount)
	for i := 0; i < retCount; i++ {
		result := lr.state.Get(-1)
		results[retCount-1-i] = lr.convertLuaValueToGo(result)
		lr.state.Pop(1)
	}

	// Check if we have captured any output
	output := lr.outputCapture.String()
	var capturedOutput string
	if output != "" {
		// Capture all lines except the last empty one
		lines := strings.Split(strings.TrimSpace(output), "\n")
		capturedOutput = strings.Join(lines, "\n")
		// Don't print to console here - let the engine handle output display
		// for _, line := range lines {
		//	fmt.Println(line)
		// }
	}

	// Reset output capture
	lr.outputCapture = nil

	// If we have captured output and no results, return the captured output as a single result
	if capturedOutput != "" && len(results) == 0 {
		return []interface{}{capturedOutput}, nil
	}

	return results, nil
}

// SetVariable sets a variable in the Lua runtime
func (lr *LuaRuntime) SetVariable(name string, value interface{}) error {
	lr.mu.Lock()
	defer lr.mu.Unlock()

	if !lr.ready {
		return errors.NewRuntimeError("lua", "LUA_RUNTIME_NOT_INITIALIZED", "runtime is not initialized")
	}

	// Create channels for result and error
	resultChan := make(chan interface{})
	errorChan := make(chan error)

	// Execute variable setting in a separate goroutine for timeout handling
	go func() {
		defer func() {
			if r := recover(); r != nil {
				errorChan <- fmt.Errorf("panic during Lua variable setting: %v", r)
			}
		}()

		luaValue, err := lr.GoToLua(value)
		if err != nil {
			errorChan <- errors.NewRuntimeError("lua", "LUA_VALUE_CONVERSION_ERROR", fmt.Sprintf("value conversion error: %v", err))
			return
		}

		lr.state.SetGlobal(name, luaValue)
		resultChan <- nil // Success
	}()

	// Set timeout
	timeout := 5 * time.Second
	select {
	case <-resultChan:
		return nil
	case err := <-errorChan:
		return err
	case <-time.After(timeout):
		return errors.NewRuntimeError("lua", "LUA_VARIABLE_TIMEOUT", fmt.Sprintf("Lua variable setting timed out after %v", timeout))
	}
}

// GetVariable retrieves a variable from the Lua runtime
func (lr *LuaRuntime) GetVariable(name string) (interface{}, error) {
	lr.mu.Lock()
	defer lr.mu.Unlock()

	if !lr.ready {
		return nil, errors.NewRuntimeError("lua", "LUA_RUNTIME_NOT_INITIALIZED", "runtime is not initialized")
	}

	// Create channels for result and error
	resultChan := make(chan interface{})
	errorChan := make(chan error)

	// Execute variable retrieval in a separate goroutine for timeout handling
	go func() {
		defer func() {
			if r := recover(); r != nil {
				errorChan <- fmt.Errorf("panic during Lua variable retrieval: %v", r)
			}
		}()

		value := lr.state.GetGlobal(name)
		if value.Type() == lua.LTNil {
			errorChan <- errors.NewRuntimeError("lua", "LUA_VARIABLE_NOT_FOUND", fmt.Sprintf("variable '%s' not found", name))
			return
		}

		resultChan <- lr.luaToGo(value)
	}()

	// Set timeout
	timeout := 5 * time.Second
	select {
	case result := <-resultChan:
		return result, nil
	case err := <-errorChan:
		return nil, err
	case <-time.After(timeout):
		return nil, errors.NewRuntimeError("lua", "LUA_VARIABLE_TIMEOUT", fmt.Sprintf("Lua variable retrieval timed out after %v", timeout))
	}
}

// Eval выполняет произвольный код на Lua
func (lr *LuaRuntime) Eval(code string) (interface{}, error) {
	lr.mu.Lock()
	defer lr.mu.Unlock()

	// Initialize output capture
	lr.outputCapture = &strings.Builder{}

	if !lr.ready {
		return nil, errors.NewRuntimeError("lua", "LUA_RUNTIME_NOT_INITIALIZED", "runtime is not initialized")
	}

	// Создаем канал для результата и ошибки
	resultChan := make(chan interface{})
	errorChan := make(chan error)

	// Выполняем код в отдельной горутине для возможности таймаута
	go func() {
		defer func() {
			if r := recover(); r != nil {
				errorChan <- fmt.Errorf("panic during Lua execution: %v", r)
			}
		}()

		// Выполняем код и получаем результат (без буферизации)
		err := lr.state.DoString(code)
		if err != nil {
			errorChan <- errors.NewRuntimeError("lua", "LUA_EVAL_ERROR", fmt.Sprintf("error evaluating code: %v", err))
			return
		}

		// Получаем результат из стека
		var result interface{}
		if lr.state.GetTop() > 0 {
			// Если есть возвращаемое значение, берем первое (самое верхнее)
			luaResult := lr.state.Get(-1)
			result = lr.convertLuaValueToGo(luaResult)
			// Очищаем весь стек
			lr.state.Pop(lr.state.GetTop())
		} else {
			result = nil
		}

		resultChan <- result
	}()

	// Устанавливаем таймаут
	timeout := 10 * time.Second
	select {
	case result := <-resultChan:
		// Проверяем, есть ли захваченный вывод
		output := lr.outputCapture.String()
		var capturedOutput string
		if output != "" {
			// Capture all lines except the last empty one
			lines := strings.Split(strings.TrimSpace(output), "\n")
			capturedOutput = strings.Join(lines, "\n")
			// Don't print to console here - let the engine handle output display
			// for _, line := range lines {
			//	fmt.Println(line)
			// }
		}
		// Сбрасываем захват вывода
		lr.outputCapture = nil

		// Debug output
		// fmt.Printf("DEBUG: Lua Eval - result: %v, capturedOutput: '%s', output: '%s'\n", result, capturedOutput, output)

		// If we have captured output and the main result is nil, return the captured output instead
		if capturedOutput != "" && result == nil {
			// fmt.Printf("DEBUG: Lua Eval - returning captured output: '%s'\n", capturedOutput)
			return capturedOutput, nil
		}

		// fmt.Printf("DEBUG: Lua Eval - returning result: %v\n", result)
		return result, nil
	case err := <-errorChan:
		// Сбрасываем захват вывода
		lr.outputCapture = nil
		return nil, err
	case <-time.After(timeout):
		// Сбрасываем захват вывода
		lr.outputCapture = nil
		return nil, errors.NewRuntimeError("lua", "LUA_EVAL_TIMEOUT", fmt.Sprintf("Lua execution timed out after %v", timeout))
	}
}

// ExecuteBatch executes Lua code in batch mode and displays all output
func (lr *LuaRuntime) ExecuteBatch(code string) error {
	lr.mu.Lock()
	defer lr.mu.Unlock()

	if !lr.ready {
		return errors.NewRuntimeError("lua", "LUA_RUNTIME_NOT_INITIALIZED", "runtime is not initialized")
	}

	// Execute code without output capture - let print statements go directly to console
	err := lr.state.DoString(code)
	if err != nil {
		return errors.NewRuntimeError("lua", "LUA_BATCH_ERROR", fmt.Sprintf("batch execution error: %v", err))
	}

	return nil
}

// ExecuteCodeBlockWithVariables выполняет код с сохранением указанных переменных
func (lr *LuaRuntime) ExecuteCodeBlockWithVariables(code string, variables []string) (interface{}, error) {
	lr.mu.Lock()
	defer lr.mu.Unlock()

	// Initialize output capture
	lr.outputCapture = &strings.Builder{}

	if !lr.ready {
		return nil, errors.NewRuntimeError("lua", "LUA_RUNTIME_NOT_INITIALIZED", "runtime is not initialized")
	}

	// Выполняем код без буферизации (как Eval)
	err := lr.state.DoString(code)
	if err != nil {
		return nil, errors.NewRuntimeError("lua", "LUA_EVAL_ERROR", fmt.Sprintf("error evaluating code: %v", err))
	}

	// Получаем результат из стека
	var result interface{}
	if lr.state.GetTop() > 0 {
		// Если есть возвращаемое значение, берем первое (самое верхнее)
		luaResult := lr.state.Get(-1)
		result = lr.convertLuaValueToGo(luaResult)
		// Очищаем весь стек
		lr.state.Pop(lr.state.GetTop())
	} else {
		result = nil
	}

	// Проверяем, есть ли захваченный вывод
	output := lr.outputCapture.String()
	var capturedOutput string
	if output != "" {
		// Capture all lines except the last empty one
		lines := strings.Split(strings.TrimSpace(output), "\n")
		capturedOutput = strings.Join(lines, "\n")
		// Don't print to console here - let the engine handle output display
		// for _, line := range lines {
		//	fmt.Println(line)
		// }
	}
	// Сбрасываем захват вывода
	lr.outputCapture = nil

	// If we have captured output and the main result is nil, return the captured output instead
	if capturedOutput != "" && result == nil {
		result = capturedOutput
	}

	// После успешного выполнения, захватываем только указанные переменные
	if len(variables) > 0 {
		// Получаем указанные переменные из Lua
		for _, varName := range variables {
			value := lr.state.GetGlobal(varName)
			if value.Type() != lua.LTNil {
				// Конвертируем значение Lua в Go и сохраняем
				goValue := lr.luaToGo(value)
				// Для простоты сохраняем в runtimeObjects, так как у Lua нет отдельного кеша переменных
				lr.runtimeObjects[varName] = goValue
			}
		}
	}

	return result, nil
}

// SetOutputCapture sets the output capture buffer
func (lr *LuaRuntime) SetOutputCapture(buffer *strings.Builder) {
	lr.outputCapture = buffer
}

// GetCapturedOutput returns the captured output and clears the buffer
func (lr *LuaRuntime) GetCapturedOutput() string {
	if lr.outputCapture == nil {
		return ""
	}
	output := lr.outputCapture.String()
	lr.outputCapture.Reset()
	// Trim trailing newlines to avoid extra line breaks in output, but preserve leading spaces
	return strings.TrimRight(output, "\n\r")
}
