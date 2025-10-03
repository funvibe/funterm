package engine

import (
	goerrors "errors"
	"fmt"
	"strings"

	"funterm/errors"
	"funterm/runtime"
	"funterm/shared"
	"go-parser/pkg/ast"
	sharedparser "go-parser/pkg/shared"

	"github.com/funvibe/funbit/pkg/funbit"
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

	return nil, errors.NewUserError("UNSUPPORTED_COMMAND", "unsupported command")
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
		return nil, errors.NewSystemError("RUNTIME_NOT_READY", fmt.Sprintf("%s runtime is not ready", call.Language))
	}
	if e.verbose {
		fmt.Printf("DEBUG: Runtime %s is ready\n", call.Language)
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
			return nil, errors.NewSystemError("ARGUMENT_CONVERSION_ERROR", fmt.Sprintf("argument conversion error: %v", err))
		}
		if e.verbose {
			fmt.Printf("DEBUG: Converted arguments: %v\n", args)
		}

		// Check that we have exactly one argument for eval
		if len(args) != 1 {
			if e.verbose {
				fmt.Printf("DEBUG: Wrong number of arguments: %d\n", len(args))
			}
			return nil, errors.NewUserError("EVAL_ARGUMENT_ERROR", "eval() requires exactly one argument")
		}

		// Convert argument to string
		code, ok := args[0].(string)
		if !ok {
			if e.verbose {
				fmt.Printf("DEBUG: Argument is not a string: %T\n", args[0])
			}
			return nil, errors.NewUserError("EVAL_ARGUMENT_ERROR", "eval() requires a string argument")
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
			return nil, errors.NewSystemError("EVAL_ERROR", fmt.Sprintf("eval error: %v", err))
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
		return nil, errors.NewSystemError("ARGUMENT_CONVERSION_ERROR", fmt.Sprintf("argument conversion error: %v", err))
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
		return nil, errors.NewSystemError("EXECUTION_ERROR", fmt.Sprintf("execution error: %v", err))
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
		return nil, errors.NewUserError("VARIABLE_READ_ERROR", "variable read requires qualified identifier (language.variable)")
	}

	language := variableRead.Variable.Language
	variableName := variableRead.Variable.Name

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
		return e.readVariableFromRuntime(rt, language, variableName)
	}

	// Try to get or create runtime from cache
	if e.runtimeRegistry != nil {
		runtime, err := e.GetOrCreateRuntime(language)
		if err == nil {
			// Read variable with the cached runtime
			return e.readVariableFromRuntime(runtime, language, variableName)
		}
	}

	return nil, errors.NewUserError("UNSUPPORTED_LANGUAGE", fmt.Sprintf("unsupported language '%s' for variable read", language))
}

// executeVariableAssignment executes a variable assignment operation
func (e *ExecutionEngine) executeVariableAssignment(variableAssignment *ast.VariableAssignment) (interface{}, error) {
	// Extract language and variable name from the qualified identifier
	if !variableAssignment.Variable.Qualified {
		// For unqualified variables, try to infer language from context
		// This is mainly needed for loop bodies where variables are used without qualification
		if e.currentLoopLanguage != "" {
			// We're in a loop context, use the loop language
			variableAssignment.Variable.Language = e.currentLoopLanguage
			variableAssignment.Variable.Qualified = true
			if e.verbose {
				fmt.Printf("DEBUG: executeVariableAssignment - auto-qualified unqualified variable with loop language '%s'\n", e.currentLoopLanguage)
			}
		} else if e.localScope != nil {
			// We're in a block context (if, match, for, while) - use local variables
			if e.verbose {
				fmt.Printf("DEBUG: executeVariableAssignment - using local scope for unqualified variable '%s'\n", variableAssignment.Variable.Name)
			}

			// Evaluate the assignment value
			value, err := e.convertExpressionToValue(variableAssignment.Value)
			if err != nil {
				return nil, errors.NewSystemError("VALUE_CONVERSION_ERROR", fmt.Sprintf("failed to convert assignment value: %v", err))
			}

			// Store in local scope
			e.localScope.Set(variableAssignment.Variable.Name, value)

			if e.verbose {
				fmt.Printf("DEBUG: executeVariableAssignment - set local variable '%s' = %v\n", variableAssignment.Variable.Name, value)
			}

			return value, nil
		} else {
			return nil, errors.NewUserError("VARIABLE_ASSIGNMENT_ERROR", "variable assignment requires qualified identifier (language.variable)")
		}
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
		return nil, errors.NewSystemError("VALUE_CONVERSION_ERROR", fmt.Sprintf("failed to convert value for assignment: %v", err))
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

	return nil, errors.NewUserError("UNSUPPORTED_LANGUAGE", fmt.Sprintf("unsupported language '%s' for variable assignment", language))
}

// readVariableFromRuntime reads a variable from a specific runtime
func (e *ExecutionEngine) readVariableFromRuntime(rt runtime.LanguageRuntime, language, variableName string) (interface{}, error) {
	// Check if runtime is ready
	if !rt.IsReady() {
		return nil, errors.NewSystemError("RUNTIME_NOT_READY", fmt.Sprintf("%s runtime is not ready", language))
	}

	// First try to get the variable from shared storage (for persistence across calls)
	if value, found := e.GetSharedVariable(language, variableName); found {
		if e.verbose {
			fmt.Printf("DEBUG: readVariableFromRuntime - FOUND variable '%s' in shared storage for language '%s', value: %v\n", variableName, language, value)
		}
		return value, nil
	}

	if e.verbose {
		fmt.Printf("DEBUG: readVariableFromRuntime - variable '%s' NOT FOUND in shared storage for language '%s'\n", variableName, language)
	}

	// If not found in shared storage, try to get it from the runtime
	if e.verbose {
		fmt.Printf("DEBUG: readVariableFromRuntime - trying to get variable '%s' from runtime\n", variableName)
	}
	value, err := rt.GetVariable(variableName)
	if err != nil {
		if e.verbose {
			fmt.Printf("DEBUG: readVariableFromRuntime - FAILED to get variable '%s' from runtime: %v\n", variableName, err)
		}
		return nil, errors.NewRuntimeError(language, "VARIABLE_NOT_FOUND", fmt.Sprintf("variable '%s' not found in %s runtime", variableName, language))
	}

	if e.verbose {
		fmt.Printf("DEBUG: readVariableFromRuntime - SUCCESSFULLY got variable '%s' from runtime, value: %v\n", variableName, value)
	}
	return value, nil
}

// setVariableInRuntime sets a variable in a specific runtime
func (e *ExecutionEngine) setVariableInRuntime(rt runtime.LanguageRuntime, language, variableName string, value interface{}) (interface{}, error) {
	// Check if runtime is ready
	if !rt.IsReady() {
		return nil, errors.NewSystemError("RUNTIME_NOT_READY", fmt.Sprintf("%s runtime is not ready", language))
	}

	// Ensure we're working with BitstringObject for bitstring data
	if language == "lua" {
		if byteSlice, ok := value.([]byte); ok {
			// Convert legacy []byte to BitstringObject
			bitString := funbit.NewBitStringFromBytes(byteSlice)
			value = &shared.BitstringObject{BitString: bitString}
		}
		// Note: if value is already *shared.BitstringObject, it will be passed as-is
	}

	// Set the variable in the runtime
	err := rt.SetVariable(variableName, value)
	if err != nil {
		var execErr *errors.ExecutionError
		if goerrors.As(err, &execErr) {
			// If this is a variable-not-found error, we can ignore it for now
			// as it might be defined later. For other errors, we should return.
			if execErr.Code != "VARIABLE_NOT_FOUND" {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	// Also store the variable in shared storage for persistence across calls
	e.SetSharedVariable(language, variableName, value)

	// Return the value that was set (following typical assignment conventions)
	return value, nil
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
				return nil, fmt.Errorf("failed to convert named argument value for '%s': %v", namedArg.Name, err)
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

// convertExpressionToValue converts new parser expression to old interface value
// TODO: This is a temporary adapter for backward compatibility with runtimes.
// Should be removed when runtimes are updated to work with new AST structures directly.
func (e *ExecutionEngine) convertExpressionToValue(expr ast.Expression) (interface{}, error) {
	switch typedExpr := expr.(type) {
	case *ast.NamedArgument:
		// NamedArgument should not be converted directly - it should be handled by convertExpressionsWithNamedArgs
		return nil, errors.NewSystemError("UNSUPPORTED_EXPRESSION", "unsupported expression type: *ast.NamedArgument")
	case *ast.BooleanLiteral:
		return typedExpr.Value, nil
	case *ast.StringLiteral:
		return typedExpr.Value, nil
	case *ast.NumberLiteral:
		return typedExpr.Value, nil
	case *ast.VariableRead:
		// For VariableRead expressions, execute the variable read
		return e.executeVariableRead(typedExpr)
	case *ast.Identifier:
		// For identifiers in pattern matching context, we need to handle them specially
		// In pattern matching, identifiers represent variables that should be bound
		// But in argument context, they should be treated as variable reads
		if typedExpr.Qualified {
			// Qualified identifier like lua.x - try to read the variable
			varRead := &ast.VariableRead{
				Variable: typedExpr,
			}
			return e.executeVariableRead(varRead)
		} else {
			// Simple identifier - first check local scope (for pattern matching variables)
			if val, found := e.localScope.Get(typedExpr.Name); found {
				return val, nil
			}

			// If we're in a loop context, try auto-qualifying with the current loop language
			if e.currentLoopLanguage != "" {
				rt, err := e.runtimeManager.GetRuntime(e.currentLoopLanguage)
				if err == nil && rt.IsReady() {
					value, err := rt.GetVariable(typedExpr.Name)
					if err == nil {
						return value, nil
					}
				}
			}

			// If not found in local scope or loop context, try to read from all available runtimes
			// This is a simplified approach - in a real implementation we'd track the current language context
			languages := e.ListAvailableLanguages()
			for _, lang := range languages {
				rt, err := e.runtimeManager.GetRuntime(lang)
				if err == nil && rt.IsReady() {
					value, err := rt.GetVariable(typedExpr.Name)
					if err == nil {
						return value, nil
					}
				}
			}

			// If not found in any runtime, return an error
			return nil, errors.NewSystemError("VARIABLE_NOT_FOUND", fmt.Sprintf("variable '%s' not found in any runtime", typedExpr.Name))
		}
	case *ast.ArrayLiteral:
		// Convert array elements to []interface{}
		result := make([]interface{}, len(typedExpr.Elements))
		for i, element := range typedExpr.Elements {
			value, err := e.convertExpressionToValue(element)
			if err != nil {
				return nil, err
			}
			result[i] = value
		}
		return result, nil
	case *ast.ObjectLiteral:
		// Convert object properties to map[string]interface{}
		result := make(map[string]interface{})
		for _, prop := range typedExpr.Properties {
			key, err := e.extractStringKey(prop.Key)
			if err != nil {
				return nil, err
			}
			value, err := e.convertExpressionToValue(prop.Value)
			if err != nil {
				return nil, err
			}
			result[key] = value
		}
		return result, nil
	case *ast.IndexExpression:
		return e.executeIndexExpression(typedExpr)
	case *ast.FieldAccess:
		return e.executeFieldAccess(typedExpr)
	case *ast.BinaryExpression:
		return e.executeBinaryExpression(typedExpr)
	case *ast.UnaryExpression:
		return e.executeUnaryExpression(typedExpr)
	case *ast.ElvisExpression:
		return e.executeElvisExpression(typedExpr)
	case *ast.TernaryExpression:
		return e.executeTernaryExpression(typedExpr)
	case *ast.PipeExpression:
		return e.executePipeExpression(typedExpr)
	case *ast.LanguageCall:
		return e.executeLanguageCallNew(typedExpr)
	case *ast.BitstringExpression:
		return e.executeBitstringExpression(typedExpr)
	case *ast.SizeExpression:
		return e.executeSizeExpression(typedExpr)
	case *ast.BitstringPatternAssignment:
		// Выполняем inplace pattern matching и возвращаем boolean результат
		return e.executeBitstringPatternAssignment(typedExpr)
	default:
		return nil, errors.NewSystemError("UNSUPPORTED_EXPRESSION", fmt.Sprintf("unsupported expression type: %T", expr))
	}
}

// extractStringKey extracts string key from Expression (Identifier or StringLiteral)
func (e *ExecutionEngine) extractStringKey(expr ast.Expression) (string, error) {
	if ident, ok := expr.(*ast.Identifier); ok {
		return ident.Name, nil
	}
	if strLit, ok := expr.(*ast.StringLiteral); ok {
		return strLit.Value, nil
	}
	return "", fmt.Errorf("expected identifier or string literal as object key, got %T", expr)
}

// isTruthy determines if a value should be considered truthy in a boolean context
func (e *ExecutionEngine) isTruthy(value interface{}) bool {
	if value == nil {
		return false
	}

	switch v := value.(type) {
	case bool:
		return v
	case string:
		return v != ""
	case int:
		return v != 0
	case int8:
		return v != 0
	case int16:
		return v != 0
	case int32:
		return v != 0
	case int64:
		return v != 0
	case uint:
		return v != 0
	case uint8:
		return v != 0
	case uint16:
		return v != 0
	case uint32:
		return v != 0
	case uint64:
		return v != 0
	case float32, float64:
		return v != 0.0
	case []interface{}:
		return len(v) > 0
	case map[string]interface{}:
		return len(v) > 0
	default:
		// For other types, consider them truthy if they're not nil
		return true
	}
}

// executeBitstringPatternAssignment выполняет присваивание с bitstring pattern слева
func (e *ExecutionEngine) executeBitstringPatternAssignment(assignment *ast.BitstringPatternAssignment) (interface{}, error) {
	if e.verbose {
		fmt.Printf("DEBUG: executeBitstringPatternAssignment - pattern: %s, value: %s\n", assignment.Pattern.String(), assignment.Value)
	}

	// Вычисляем значение справа от =
	value, err := e.convertExpressionToValue(assignment.Value)
	if err != nil {
		return nil, errors.NewSystemError("VALUE_CONVERSION_ERROR", fmt.Sprintf("failed to convert value for bitstring pattern assignment: %v", err))
	}

	// Преобразуем значение в BitstringObject для pattern matching
	var bitstringData *shared.BitstringObject
	switch v := value.(type) {
	case *shared.BitstringObject:
		bitstringData = v
	case string:
		// Создаем bitstring из строки используя UTF-8
		builder := funbit.NewBuilder()
		funbit.AddUTF8(builder, v)
		bitString, err := funbit.Build(builder)
		if err != nil {
			return nil, errors.NewSystemError("BITSTRING_CREATION_ERROR", fmt.Sprintf("failed to create bitstring from string: %v", err))
		}
		bitstringData = &shared.BitstringObject{BitString: bitString}
	case []byte:
		// Создаем bitstring из байтов
		bitString := funbit.NewBitStringFromBytes(v)
		bitstringData = &shared.BitstringObject{BitString: bitString}
	default:
		return nil, errors.NewUserError("BITSTRING_PATTERN_ERROR", fmt.Sprintf("cannot match bitstring pattern against type %T", value))
	}

	// Выполняем pattern matching используя funbit adapter
	adapter := NewFunbitAdapterWithEngine(e)
	bindings, err := adapter.MatchBitstringWithFunbit(assignment.Pattern, bitstringData)
	if err != nil {
		return nil, errors.NewUserError("BITSTRING_PATTERN_ERROR", fmt.Sprintf("pattern matching failed: %v", err))
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
			// Неквалифицированная переменная - записываем в localScope (для блоков)
			if e.localScope == nil {
				e.localScope = sharedparser.NewScope(nil)
			}
			e.localScope.Set(varName, varValue)

			if e.verbose {
				fmt.Printf("DEBUG: executeBitstringPatternAssignment - bound local variable '%s' = %v\n", varName, varValue)
			}
		}
	}

	// Возвращаем результат матчинга: true если успешно, false если нет
	return true, nil
}
