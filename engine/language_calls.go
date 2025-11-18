package engine

import (
	"fmt"
	"strings"

	"funterm/errors"
	"funterm/runtime"
	"funterm/shared"
	"go-parser/pkg/ast"

	"github.com/funvibe/funbit/pkg/funbit"
	lua "github.com/yuin/gopher-lua"
)

// executeLanguageCallNew executes a language.function() call using new parser AST
func (e *ExecutionEngine) executeLanguageCallNew(call *ast.LanguageCall) (interface{}, error) {
	// Handle alias 'py' for 'python'
	if call.Language == "py" {
		call.Language = "python"
	}
	// Handle alias 'js' for 'node'
	if call.Language == "js" {
		call.Language = "node"
	}

	// Try to get the runtime from the runtime manager first
	rt, err := e.runtimeManager.GetRuntime(call.Language)
	if err == nil {
		// Runtime found in manager, use it
		return e.executeWithRuntimeNew(rt, call)
	}

	// Try to get or create runtime from cache
	if e.runtimeRegistry != nil {
		runtime, err := e.GetOrCreateRuntime(call.Language)
		if err == nil {
			return e.executeWithRuntimeNew(runtime, call)
		}
	}

	return nil, errors.NewUserErrorWithASTPos("UNSUPPORTED_COMMAND", "unsupported command", call.Position())
}

// executeWithRuntimeNew executes a language call with a specific runtime using new parser AST
func (e *ExecutionEngine) executeWithRuntimeNew(rt runtime.LanguageRuntime, call *ast.LanguageCall) (interface{}, error) {
	if e.verbose {
		fmt.Printf("DEBUG: executeWithRuntimeNew called with Language=%s, Function=%s\n", call.Language, call.Function)
	}

	// Check if runtime is ready
	if !rt.IsReady() {
		if e.verbose {
			fmt.Printf("DEBUG: Runtime %s is not ready\n", call.Language)
		}
		return nil, errors.NewUserErrorWithASTPos("RUNTIME_NOT_READY", fmt.Sprintf("%s runtime is not ready", call.Language), call.Position())
	}
	if e.verbose {
		fmt.Printf("DEBUG: Runtime %s is ready\n", call.Language)
	}

	// Sync global variables to runtime before executing
	if err := e.syncGlobalVariablesToRuntime(rt); err != nil {
		if e.verbose {
			fmt.Printf("DEBUG: Warning - failed to sync global variables: %v\n", err)
		}
		// Continue execution even if sync fails
	}

	// Handle special builtin functions
	if call.Function == "id" {
		// Convert arguments
		args, err := e.convertExpressionsToArgs(call.Arguments)
		if err != nil {
			if e.verbose {
				fmt.Printf("DEBUG: Error converting arguments for id(): %v\n", err)
			}
			return nil, errors.NewUserErrorWithASTPos("ARGUMENT_CONVERSION_ERROR", fmt.Sprintf("argument conversion error: %v", err), call.Position())
		}
		if e.verbose {
			fmt.Printf("DEBUG: Calling id() with args: %v\n", args)
		}
		return e.executeIdFunction(args)
	}

	// Handle special eval function
	if call.Function == "eval" {
		if e.verbose {
			fmt.Printf("DEBUG: Handling eval function\n")
		}
		// Convert new parser expressions to old interface arguments
		args, err := e.convertExpressionsToArgs(call.Arguments)
		if err != nil {
			if e.verbose {
				fmt.Printf("DEBUG: Error converting arguments: %v\n", err)
			}
			return nil, errors.NewUserErrorWithASTPos("ARGUMENT_CONVERSION_ERROR", fmt.Sprintf("argument conversion error: %v", err), call.Position())
		}
		if e.verbose {
			fmt.Printf("DEBUG: Converted arguments: %v\n", args)
		}

		// Check that we have exactly one argument for eval
		if len(args) != 1 {
			if e.verbose {
				fmt.Printf("DEBUG: Wrong number of arguments: %d\n", len(args))
			}
			return nil, errors.NewUserErrorWithASTPos("EVAL_ARGUMENT_ERROR", "eval() requires exactly one argument", call.Position())
		}

		// Convert argument to string
		code, ok := args[0].(string)
		if !ok {
			if e.verbose {
				fmt.Printf("DEBUG: Argument is not a string: %T\n", args[0])
			}
			return nil, errors.NewUserErrorWithASTPos("EVAL_ARGUMENT_ERROR", "eval() requires a string argument", call.Position())
		}
		if e.verbose {
			fmt.Printf("DEBUG: Code to eval: %s\n", code)
		}

		// Use Eval method instead of ExecuteFunction
		if e.verbose {
			fmt.Printf("DEBUG: Calling rt.Eval()...\n")
		}
		result, err := rt.Eval(code)
		if err != nil {
			if e.verbose {
				fmt.Printf("DEBUG: Error from rt.Eval(): %v\n", err)
			}
			return nil, errors.NewUserErrorWithASTPos("EVAL_ERROR", fmt.Sprintf("eval error: %v", err), call.Position())
		}
		if e.verbose {
			fmt.Printf("DEBUG: Eval result: %v\n", result)
		}

		return result, nil
	}

	if e.verbose {
		fmt.Printf("DEBUG: Handling regular function call\n")
	}
	// Convert new parser expressions to old interface arguments
	args, err := e.convertExpressionsToArgs(call.Arguments)
	if err != nil {
		if e.verbose {
			fmt.Printf("DEBUG: Error converting arguments: %v\n", err)
		}
		return nil, errors.NewUserErrorWithASTPos("ARGUMENT_CONVERSION_ERROR", fmt.Sprintf("argument conversion error: %v", err), call.Position())
	}

	// Execute the function (call.Function already contains the full name including module)
	if e.verbose {
		fmt.Printf("DEBUG: Calling rt.ExecuteFunction()...\n")
	}
	result, err := rt.ExecuteFunction(call.Function, args)
	if err != nil {
		if e.verbose {
			fmt.Printf("DEBUG: Error from rt.ExecuteFunction(): %v\n", err)
		}
		return nil, errors.NewUserErrorWithASTPos("EXECUTION_ERROR", fmt.Sprintf("execution error: %v", err), call.Position())
	}
	if e.verbose {
		fmt.Printf("DEBUG: ExecuteFunction result: %v\n", result)
	}

	return result, nil
}

// executeVariableRead executes a variable read operation
func (e *ExecutionEngine) executeVariableRead(variableRead *ast.VariableRead) (interface{}, error) {
	// Extract language and variable name from the qualified identifier
	if !variableRead.Variable.Qualified {
		// Handle unqualified variables - read from global scope
		varName := variableRead.Variable.Name

		// Try to get the variable from current scope first
		if value, found := e.localScope.Get(varName); found {
			if e.verbose {
				fmt.Printf("DEBUG: executeVariableRead - found unqualified variable '%s' in local scope: %v\n", varName, value)
			}
			return value, nil
		}

		// Try to get from global variables
		if value, found := e.getGlobalVariable(varName); found {
			if e.verbose {
				fmt.Printf("DEBUG: executeVariableRead - found unqualified variable '%s' in global scope: %v\n", varName, value)
			}
			return value, nil
		}

		// Variable not found - return nil for unqualified variables (they are optional)
		if e.verbose {
			fmt.Printf("DEBUG: executeVariableRead - unqualified variable '%s' not found, returning nil\n", varName)
		}
		return nil, nil
	}

	language := variableRead.Variable.Language
	variableName := variableRead.Variable.Name
	path := variableRead.Variable.Path

	// Handle alias 'py' for 'python'
	if language == "py" {
		language = "python"
	}
	// Handle alias 'js' for 'node'
	if language == "js" {
		language = "node"
	}

	// Try to get the runtime from the runtime manager first
	rt, err := e.runtimeManager.GetRuntime(language)
	if err == nil {
		// Runtime found in manager, use it
		return e.readVariableFromRuntimeWithPath(rt, language, path, variableName)
	}

	// Try to get or create runtime from cache
	if e.runtimeRegistry != nil {
		runtime, err := e.GetOrCreateRuntime(language)
		if err == nil {
			// Read variable with the cached runtime
			return e.readVariableFromRuntimeWithPath(runtime, language, path, variableName)
		}
	}

	return nil, errors.NewUserErrorWithASTPos("UNSUPPORTED_LANGUAGE", fmt.Sprintf("unsupported language '%s' for variable read", language), variableRead.Position())
}

// executeVariableAssignment executes a variable assignment operation
func (e *ExecutionEngine) executeVariableAssignment(variableAssignment *ast.VariableAssignment) (interface{}, error) {
	// Extract language and variable name from the qualified identifier
	if !variableAssignment.Variable.Qualified {
		// Evaluate the assignment value first
		value, err := e.convertExpressionToValue(variableAssignment.Value)
		if err != nil {
			return nil, errors.NewUserErrorWithASTPos("VALUE_CONVERSION_ERROR", fmt.Sprintf("failed to convert assignment value: %v", err), variableAssignment.Value.Position())
		}

		// Check if we're at the root scope (top level)
		if e.localScope.IsRoot() {
			// Top-level unqualified variable - store as global
			// Check mutability for reassignment
			if varInfo, exists := e.getGlobalVariableInfo(variableAssignment.Variable.Name); exists {
				if !varInfo.IsMutable {
					pos := variableAssignment.Position()
					return nil, errors.NewUserErrorWithASTPos("IMMUTABLE_VARIABLE_ERROR", fmt.Sprintf("cannot reassign immutable variable '%s'", variableAssignment.Variable.Name), pos)
				}
				// Variable exists and is mutable, check operator type
				if varInfo.IsMutable && !variableAssignment.IsMutable {
					// Mutable variable being reassigned with = instead of :=
					return nil, errors.NewUserErrorWithASTPos("IMMUTABLE_VARIABLE_ERROR", fmt.Sprintf("cannot reassign mutable variable '%s' with '=', use ':=' instead", variableAssignment.Variable.Name), variableAssignment.Position())
				}
				// Variable exists, preserve existing mutability
				e.setGlobalVariableWithMutability(variableAssignment.Variable.Name, value, varInfo.IsMutable)
			} else {
				// Variable doesn't exist, use mutability from assignment
				e.setGlobalVariableWithMutability(variableAssignment.Variable.Name, value, variableAssignment.IsMutable)
			}

			if e.verbose {
				mutabilityStr := "immutable"
				if varInfo, exists := e.getGlobalVariableInfo(variableAssignment.Variable.Name); exists {
					if varInfo.IsMutable {
						mutabilityStr = "mutable"
					}
					fmt.Printf("DEBUG: executeVariableAssignment - set global variable '%s' = %v (%s) - existing variable preserved\n", variableAssignment.Variable.Name, value, mutabilityStr)
				} else {
					if variableAssignment.IsMutable {
						mutabilityStr = "mutable"
					}
					fmt.Printf("DEBUG: executeVariableAssignment - set global variable '%s' = %v (%s) - new variable\n", variableAssignment.Variable.Name, value, mutabilityStr)
				}
			}

			return value, nil
		}

		// We're in a nested scope (if, match, for, while) - check if variable exists in parent scopes or globals
		if e.verbose {
			fmt.Printf("DEBUG: executeVariableAssignment - checking scope for unqualified variable '%s'\n", variableAssignment.Variable.Name)
		}

		// Check if variable exists in parent scopes
		if varInfo, found := e.getVariableInfo(variableAssignment.Variable.Name); found {
			// Variable exists, check mutability before reassignment
			if !varInfo.IsMutable {
				return nil, errors.NewUserErrorWithASTPos("IMMUTABLE_VARIABLE_ERROR", fmt.Sprintf("cannot reassign immutable variable '%s'", variableAssignment.Variable.Name), variableAssignment.Position())
			}
			// Variable exists and is mutable, check operator type
			if varInfo.IsMutable && !variableAssignment.IsMutable {
				// Mutable variable being reassigned with = instead of :=
				return nil, errors.NewUserErrorWithASTPos("IMMUTABLE_VARIABLE_ERROR", fmt.Sprintf("cannot reassign mutable variable '%s' with '=', use ':=' instead", variableAssignment.Variable.Name), variableAssignment.Position())
			}

			// Variable exists in parent scope, update it there (not shadowing)
			// Preserve existing mutability by using setVariableInParentScope which doesn't change mutability
			if e.verbose {
				fmt.Printf("DEBUG: executeVariableAssignment - variable '%s' found in parent scope, updating there\n", variableAssignment.Variable.Name)
			}
			e.setVariableInParentScope(variableAssignment.Variable.Name, value)
		} else if varInfo, globalFound := e.getGlobalVariableInfo(variableAssignment.Variable.Name); globalFound {
			// Variable exists as global, check mutability before reassignment
			if !varInfo.IsMutable {
				return nil, errors.NewUserErrorWithASTPos("IMMUTABLE_VARIABLE_ERROR", fmt.Sprintf("cannot reassign immutable variable '%s'", variableAssignment.Variable.Name), variableAssignment.Position())
			}
			// Variable exists and is mutable, check operator type
			if varInfo.IsMutable && !variableAssignment.IsMutable {
				// Mutable variable being reassigned with = instead of :=
				return nil, errors.NewUserErrorWithASTPos("IMMUTABLE_VARIABLE_ERROR", fmt.Sprintf("cannot reassign mutable variable '%s' with '=', use ':=' instead", variableAssignment.Variable.Name), variableAssignment.Position())
			}

			// Variable exists as global, update global variable
			// Preserve existing mutability - don't use variableAssignment.IsMutable
			if e.verbose {
				fmt.Printf("DEBUG: executeVariableAssignment - variable '%s' found in globals, updating global\n", variableAssignment.Variable.Name)
			}
			e.setGlobalVariableWithMutability(variableAssignment.Variable.Name, value, varInfo.IsMutable)
		} else {
			// Variable doesn't exist, create in current scope
			if e.verbose {
				fmt.Printf("DEBUG: executeVariableAssignment - variable '%s' not found in parent scopes or globals, creating in current scope\n", variableAssignment.Variable.Name)
			}
			e.setVariableWithMutability(variableAssignment.Variable.Name, value, variableAssignment.IsMutable)
		}

		if e.verbose {
			mutabilityStr := "immutable"
			if variableAssignment.IsMutable {
				mutabilityStr = "mutable"
			}
			fmt.Printf("DEBUG: executeVariableAssignment - set variable '%s' = %v (%s)\n", variableAssignment.Variable.Name, value, mutabilityStr)
		}

		return value, nil
	}

	language := variableAssignment.Variable.Language
	variableName := variableAssignment.Variable.Name

	// Debug output to see what variable name we get
	if e.verbose {
		fmt.Printf("DEBUG: executeVariableAssignment - language=%s, variableName=%s\n", language, variableName)
	}

	// Handle alias 'py' for 'python'
	if language == "py" {
		language = "python"
	}
	// Handle alias 'js' for 'node'
	if language == "js" {
		language = "node"
	}

	// Convert the value to the appropriate format
	value, err := e.convertExpressionToValue(variableAssignment.Value)
	if err != nil {
		return nil, errors.NewUserErrorWithASTPos("VALUE_CONVERSION_ERROR", fmt.Sprintf("failed to convert value for assignment: %v", err), variableAssignment.Value.Position())
	}

	// Try to get the runtime from the runtime manager first
	rt, err := e.runtimeManager.GetRuntime(language)
	if err == nil {
		// Runtime found in manager, use it
		return e.setVariableInRuntime(rt, language, variableName, value)
	}

	// Try to get or create runtime from cache
	if e.runtimeRegistry != nil {
		runtime, err := e.GetOrCreateRuntime(language)
		if err == nil {
			// Set variable with the cached runtime
			return e.setVariableInRuntime(runtime, language, variableName, value)
		}
	}

	return nil, errors.NewUserErrorWithASTPos("UNSUPPORTED_LANGUAGE", fmt.Sprintf("unsupported language '%s' for variable assignment", language), variableAssignment.Position())
}

// readVariableFromRuntimeWithPath reads a variable from runtime, handling path-based field access
func (e *ExecutionEngine) readVariableFromRuntimeWithPath(rt runtime.LanguageRuntime, language string, path []string, variableName string) (interface{}, error) {
	// Check if runtime is ready
	if !rt.IsReady() {
		return nil, errors.NewUserError("RUNTIME_NOT_READY", fmt.Sprintf("%s runtime is not ready", language))
	}

	// If no path, use the simple variable read
	if len(path) == 0 {
		return e.readVariableFromRuntime(rt, language, variableName)
	}

	// Handle path-based access (e.g., lua.dns_query_packet.bytes)
	// First, get the base object using the path
	baseObject, err := e.getVariableWithPath(rt, language, path)
	if err != nil {
		return nil, err
	}

	// Now access the field/method on the base object
	// For Lua, we can use the metatable methods if available
	if language == "lua" {
		return e.accessLuaObjectField(baseObject, variableName)
	}

	// For other languages, we might need different approaches
	// For now, return an error for unsupported path access
	return nil, errors.NewUserError("UNSUPPORTED_PATH_ACCESS", fmt.Sprintf("path-based field access not supported for language '%s'", language))
}

// getVariableWithPath gets a variable following a path (e.g., ["dns_query_packet"] for lua.dns_query_packet.bytes)
func (e *ExecutionEngine) getVariableWithPath(rt runtime.LanguageRuntime, language string, path []string) (interface{}, error) {
	if len(path) == 0 {
		return nil, errors.NewUserError("INVALID_PATH", "path cannot be empty")
	}

	// Start with the first element in the path
	currentName := path[0]
	currentValue, err := e.readVariableFromRuntime(rt, language, currentName)
	if err != nil {
		return nil, err
	}

	// For single-element paths, return the value directly
	if len(path) == 1 {
		return currentValue, nil
	}

	// For multi-element paths, we'd need to navigate deeper
	// But for now, let's assume single-element paths (object.field)
	return currentValue, nil
}

// accessLuaObjectField accesses a field/method on a Lua object
func (e *ExecutionEngine) accessLuaObjectField(obj interface{}, fieldName string) (interface{}, error) {
	// First check if this is a BitstringObject and we're accessing 'bytes'
	if bs, ok := obj.(*shared.BitstringObject); ok && fieldName == "bytes" {
		bytes := bs.BitString.ToBytes()
		// Return as []byte - the Python runtime should handle this properly
		return bytes, nil
	}

	// Check if this is a Lua userdata with metatable methods
	if luaUserData, ok := obj.(*lua.LUserData); ok {
		// Try to call the field as a method
		if metaTable, ok := luaUserData.Metatable.(*lua.LTable); ok && metaTable != nil {
			fieldValue := metaTable.RawGetString(fieldName)
			if fieldValue != lua.LNil {
				// If it's a function, we can't call it from Go
				// But for simple field access, we need to simulate Lua's field access
				if fieldValue.Type() == lua.LTFunction {
					// This is a method - we need to call it
					// For now, let's create a temporary Lua state to call the method
					tempState := lua.NewState()
					defer tempState.Close()

					// Push the userdata and call the method
					tempState.Push(fieldValue)
					tempState.Push(luaUserData)
					err := tempState.PCall(1, 1, nil)
					if err != nil {
						return nil, errors.NewUserError("LUA_METHOD_CALL_ERROR", fmt.Sprintf("failed to call Lua method '%s': %v", fieldName, err))
					}

					// Get the result and convert it back to Go
					result := tempState.Get(-1)
					return e.convertLuaValueToGo(result)
				} else {
					// It's a regular field value
					return e.convertLuaValueToGo(fieldValue)
				}
			}
		}
	}

	return nil, errors.NewUserError("FIELD_ACCESS_ERROR", fmt.Sprintf("cannot access field '%s' on Lua object", fieldName))
}

// convertLuaValueToGo converts a Lua value back to a Go value
func (e *ExecutionEngine) convertLuaValueToGo(lValue lua.LValue) (interface{}, error) {
	switch v := lValue.(type) {
	case lua.LString:
		return string(v), nil
	case lua.LNumber:
		return float64(v), nil
	case lua.LBool:
		return bool(v), nil
	case *lua.LUserData:
		return v.Value, nil
	default:
		return nil, errors.NewUserError("UNSUPPORTED_LUA_TYPE", fmt.Sprintf("cannot convert Lua type %T to Go", v))
	}
}

// convertExpressionsToArgs converts new parser expressions to old interface arguments
// TODO: This is a temporary adapter for backward compatibility with runtimes.
// Should be removed when runtimes are updated to work with new AST structures directly.
func (e *ExecutionEngine) convertExpressionsToArgs(exprs []ast.Expression) ([]interface{}, error) {
	// Check if we have any named arguments
	hasNamedArgs := false
	for _, expr := range exprs {
		if _, ok := expr.(*ast.NamedArgument); ok {
			hasNamedArgs = true
			break
		}
	}

	if hasNamedArgs {
		// For named arguments, create a special structure
		return e.convertExpressionsWithNamedArgs(exprs)
	}

	// Standard positional arguments
	args := make([]interface{}, len(exprs))
	for i, expr := range exprs {
		value, err := e.convertExpressionToValue(expr)
		if err != nil {
			return nil, err
		}
		args[i] = value
	}
	return args, nil
}

// convertExpressionsWithNamedArgs handles expressions that contain named arguments
func (e *ExecutionEngine) convertExpressionsWithNamedArgs(exprs []ast.Expression) ([]interface{}, error) {
	// Create a map structure for mixed positional and named arguments
	result := map[string]interface{}{
		"positional": make([]interface{}, 0),
		"keyword":    make(map[string]interface{}),
	}

	for _, expr := range exprs {
		if namedArg, ok := expr.(*ast.NamedArgument); ok {
			// This is a named argument
			value, err := e.convertExpressionToValue(namedArg.Value)
			if err != nil {
				return nil, errors.NewUserErrorWithASTPos("NAMED_ARGUMENT_ERROR", fmt.Sprintf("failed to convert named argument value for '%s': %v", namedArg.Name, err), namedArg.Position())
			}
			result["keyword"].(map[string]interface{})[namedArg.Name] = value
		} else {
			// This is a positional argument
			value, err := e.convertExpressionToValue(expr)
			if err != nil {
				return nil, err
			}
			result["positional"] = append(result["positional"].([]interface{}), value)
		}
	}

	return []interface{}{result}, nil
}

// executeBitstringPatternAssignment выполняет присваивание с bitstring pattern слева
func (e *ExecutionEngine) executeBitstringPatternAssignment(assignment *ast.BitstringPatternAssignment) (interface{}, error) {
	if e.verbose {
		fmt.Printf("DEBUG: executeBitstringPatternAssignment - pattern: %s, value: %s\n", assignment.Pattern.String(), assignment.Value)
	}

	// Вычисляем значение справа от =
	value, err := e.convertExpressionToValue(assignment.Value)
	if err != nil {
		return nil, errors.NewUserErrorWithASTPos("VALUE_CONVERSION_ERROR", fmt.Sprintf("failed to convert value for bitstring pattern assignment: %v", err), assignment.Value.Position())
	}

	// Преобразуем значение в BitstringObject для pattern matching
	var bitstringData *shared.BitstringObject
	switch v := value.(type) {
	case *shared.BitstringObject:
		bitstringData = v
	case string:
		// Создаем bitstring из байтов строки
		bitString := funbit.NewBitStringFromBytes([]byte(v))
		bitstringData = &shared.BitstringObject{BitString: bitString}
	case []byte:
		// Создаем bitstring из байтов
		bitString := funbit.NewBitStringFromBytes(v)
		bitstringData = &shared.BitstringObject{BitString: bitString}
	default:
		return nil, errors.NewUserErrorWithASTPos("BITSTRING_PATTERN_ERROR", fmt.Sprintf("cannot match bitstring pattern against type %T", value), assignment.Value.Position())
	}

	// Выполняем pattern matching используя funbit adapter
	// For assignments, return false on pattern matching failure instead of error
	adapter := NewFunbitAdapterWithEngine(e)
	bindings, err := adapter.MatchBitstringWithFunbit(assignment.Pattern, bitstringData, true)
	if err != nil {
		return nil, errors.NewUserErrorWithASTPos("BITSTRING_PATTERN_ERROR", fmt.Sprintf("pattern matching failed: %v", err), assignment.Position())
	}

	// Check if bindings is empty (indicates size mismatch)
	if len(bindings) == 0 {
		if e.verbose {
			fmt.Printf("DEBUG: executeBitstringPatternAssignment - empty bindings (size mismatch), returning false\n")
		}
		return false, nil
	}

	if e.verbose {
		fmt.Printf("DEBUG: executeBitstringPatternAssignment - pattern matched, bindings: %v\n", bindings)
	}

	// Связываем переменные из pattern в соответствующий scope
	for varName, varValue := range bindings {
		// На верхнем уровне переменные должны быть квалифицированными
		// Парсим имя переменной чтобы извлечь язык и имя
		if dotIndex := strings.Index(varName, "."); dotIndex != -1 {
			// Квалифицированная переменная (например, "lua.h")
			language := varName[:dotIndex]
			variableName := varName[dotIndex+1:]

			// Handle alias 'py' for 'python'
			if language == "py" {
				language = "python"
			}
			// Handle alias 'js' for 'node'
			if language == "js" {
				language = "node"
			}

			// Записываем в shared variables
			e.SetSharedVariable(language, variableName, varValue)

			if e.verbose {
				fmt.Printf("DEBUG: executeBitstringPatternAssignment - bound qualified variable '%s.%s' = %v\n", language, variableName, varValue)
			}
		} else {
			// Неквалифицированная переменная - записываем в текущий локальный scope
			// Используем setVariable вместо прямого e.localScope.Set, чтобы
			// переменные попадали в текущий scope (который может быть nested)
			// и не утекали во внешние области видимости
			e.setVariable(varName, varValue)

			if e.verbose {
				fmt.Printf("DEBUG: executeBitstringPatternAssignment - bound local variable '%s' = %v\n", varName, varValue)
			}
		}
	}

	// Возвращаем результат матчинга: true если успешно, false если нет
	return true, nil
}

// executeBitstringPatternMatchExpression выполняет bitstring pattern matching и возвращает boolean
func (e *ExecutionEngine) executeBitstringPatternMatchExpression(matchExpr *ast.BitstringPatternMatchExpression) (interface{}, error) {
	if e.verbose {
		fmt.Printf("DEBUG: executeBitstringPatternMatchExpression - pattern: %s, value: %s\n", matchExpr.Pattern.String(), matchExpr.Value)
	}

	// Вычисляем значение справа от =
	value, err := e.convertExpressionToValue(matchExpr.Value)
	if err != nil {
		return nil, errors.NewUserErrorWithASTPos("VALUE_CONVERSION_ERROR", fmt.Sprintf("failed to convert value for bitstring pattern match: %v", err), matchExpr.Value.Position())
	}

	// Преобразуем значение в BitstringObject для pattern matching
	var bitstringData *shared.BitstringObject
	switch v := value.(type) {
	case *shared.BitstringObject:
		bitstringData = v
	case string:
		// Создаем bitstring из байтов строки
		bitString := funbit.NewBitStringFromBytes([]byte(v))
		bitstringData = &shared.BitstringObject{BitString: bitString}
	case []byte:
		// Создаем bitstring из байтов
		bitString := funbit.NewBitStringFromBytes(v)
		bitstringData = &shared.BitstringObject{BitString: bitString}
	default:
		return nil, errors.NewUserErrorWithASTPos("BITSTRING_PATTERN_ERROR", fmt.Sprintf("cannot match bitstring pattern against type %T", value), matchExpr.Value.Position())
	}

	// Выполняем pattern matching используя funbit adapter
	// For expressions, return false on pattern matching failure instead of error
	adapter := NewFunbitAdapterWithEngine(e)
	bindings, err := adapter.MatchBitstringWithFunbit(matchExpr.Pattern, bitstringData, true)
	if err != nil {
		return nil, errors.NewUserErrorWithASTPos("BITSTRING_PATTERN_ERROR", fmt.Sprintf("pattern matching failed: %v", err), matchExpr.Position())
	}

	// Check if bindings is empty (indicates size mismatch)
	if len(bindings) == 0 {
		if e.verbose {
			fmt.Printf("DEBUG: executeBitstringPatternMatchExpression - empty bindings (size mismatch), returning false\n")
		}
		return false, nil
	}

	if e.verbose {
		fmt.Printf("DEBUG: executeBitstringPatternMatchExpression - pattern matched, bindings: %v\n", bindings)
	}

	// Связываем переменные из pattern в соответствующий scope
	for varName, varValue := range bindings {
		// На верхнем уровне переменные должны быть квалифицированными
		// Парсим имя переменной чтобы извлечь язык и имя
		if dotIndex := strings.Index(varName, "."); dotIndex != -1 {
			// Квалифицированная переменная (например, "lua.h")
			language := varName[:dotIndex]
			variableName := varName[dotIndex+1:]

			// Handle alias 'py' for 'python'
			if language == "py" {
				language = "python"
			}
			// Handle alias 'js' for 'node'
			if language == "js" {
				language = "node"
			}

			// Записываем в shared variables
			e.SetSharedVariable(language, variableName, varValue)

			if e.verbose {
				fmt.Printf("DEBUG: executeBitstringPatternMatchExpression - bound qualified variable '%s.%s' = %v\n", language, variableName, varValue)
			}
		} else {
			// Неквалифицированная переменная - записываем в текущий локальный scope
			// Используем setVariable вместо прямого e.localScope.Set, чтобы
			// переменные попадали в текущий scope (который может быть nested)
			// и не утекали во внешние области видимости
			e.setVariable(varName, varValue)

			if e.verbose {
				fmt.Printf("DEBUG: executeBitstringPatternMatchExpression - bound local variable '%s' = %v\n", varName, varValue)
			}
		}
	}

	// Возвращаем результат матчинга: true если успешно, false если нет
	return true, nil
}
