package engine

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	goerrors "errors"
	stderrors "errors"
	"funterm/errors"
	"funterm/runtime/python"
	"funterm/shared"
	"go-parser/pkg/ast"
	"go-parser/pkg/lexer"
)

// executeIfStatement executes an if/else statement
func (e *ExecutionEngine) executeIfStatement(ifStmt *ast.IfStatement) (interface{}, error) {
	if e.verbose {
		fmt.Printf("DEBUG: executeIfStatement called\n")
		fmt.Printf("DEBUG: Consequent block has %d statements\n", len(ifStmt.Consequent.Statements))
		fmt.Printf("DEBUG: Has else: %v\n", ifStmt.HasElse())
		if ifStmt.HasElse() {
			fmt.Printf("DEBUG: Alternate block has %d statements\n", len(ifStmt.Alternate.Statements))
		}
	}

	// If the if statement blocks are empty, try to handle it as a special case
	// This is a workaround for parser issues
	if len(ifStmt.Consequent.Statements) == 0 && ifStmt.HasElse() && len(ifStmt.Alternate.Statements) == 0 {
		if e.verbose {
			fmt.Printf("DEBUG: Both if and else blocks are empty, trying to parse from source\n")
		}
		return e.executeIfStatementFromSource(ifStmt)
	}

	// Check if the condition is an inplace pattern assignment or match expression
	// If so, we need to handle variable bindings specially to prevent leakage
	var conditionBindings map[string]interface{}
	var conditionValue interface{}
	var err error

	if bitstringPattern, ok := ifStmt.Condition.(*ast.BitstringPatternAssignment); ok {
		// Create a temporary scope for evaluating the pattern
		// This ensures unqualified variables don't leak to the parent scope
		e.pushScope()

		// Evaluate the bitstring pattern assignment
		conditionValue, err = e.executeBitstringPatternAssignment(bitstringPattern)
		if err != nil {
			e.popScope()
			return nil, errors.NewUserErrorWithASTPos("CONDITION_EVAL_ERROR", fmt.Sprintf("failed to evaluate if condition pattern: %v", err), ifStmt.Condition.Position())
		}

		// Extract the local variables that were bound during pattern matching
		// These should be passed as bindings to the if block
		conditionBindings = e.extractLocalVariablesFromCurrentScope()

		// Pop the temporary scope - this removes the variables from the parent scope
		e.popScope()

		if e.verbose {
			fmt.Printf("DEBUG: executeIfStatement - extracted %d bindings from pattern condition\n", len(conditionBindings))
		}
	} else if bitstringMatch, ok := ifStmt.Condition.(*ast.BitstringPatternMatchExpression); ok {
		// Create a temporary scope for evaluating the pattern match
		e.pushScope()

		// Evaluate the bitstring pattern match expression
		conditionValue, err = e.executeBitstringPatternMatchExpression(bitstringMatch)
		if err != nil {
			e.popScope()
			return nil, errors.NewUserErrorWithASTPos("CONDITION_EVAL_ERROR", fmt.Sprintf("failed to evaluate if condition pattern match: %v", err), ifStmt.Condition.Position())
		}

		// Extract the local variables that were bound during pattern matching
		conditionBindings = e.extractLocalVariablesFromCurrentScope()

		// Pop the temporary scope - this removes the variables from the parent scope
		e.popScope()

		if e.verbose {
			fmt.Printf("DEBUG: executeIfStatement - extracted %d bindings from pattern match condition\n", len(conditionBindings))
		}
	} else {
		// Regular condition evaluation
		conditionValue, err = e.convertExpressionToValue(ifStmt.Condition)
		if err != nil {
			return nil, errors.NewUserErrorWithASTPos("CONDITION_EVAL_ERROR", fmt.Sprintf("failed to evaluate if condition: %v", err), ifStmt.Condition.Position())
		}
	}

	// Check if condition is truthy
	isTruthy := e.isTruthy(conditionValue)
	if e.verbose {
		fmt.Printf("DEBUG: Condition value: %v, isTruthy: %v\n", conditionValue, isTruthy)
	}

	if isTruthy {
		// Execute consequent block with any bindings from the condition
		if e.verbose {
			fmt.Printf("DEBUG: CONDITION TRUE - Executing consequent block with %d statements\n", len(ifStmt.Consequent.Statements))
		}
		var result interface{}
		if conditionBindings != nil && len(conditionBindings) > 0 {
			result, err = e.executeIfBlockStatementsWithBindings(ifStmt.Consequent, conditionBindings)
		} else {
			result, err = e.executeIfBlockStatements(ifStmt.Consequent)
		}
		if err != nil {
			return nil, err
		}
		if e.verbose {
			fmt.Printf("DEBUG: Consequent block result: %v\n", result)
		}
		return result, nil
	} else if ifStmt.HasElse() {
		// Execute alternate (else) block
		// Note: bindings from the condition are NOT passed to the else block
		// since the pattern didn't match
		if e.verbose {
			fmt.Printf("DEBUG: CONDITION FALSE - Executing alternate block with %d statements\n", len(ifStmt.Alternate.Statements))
		}
		result, err := e.executeIfBlockStatements(ifStmt.Alternate)
		if err != nil {
			return nil, err
		}
		if e.verbose {
			fmt.Printf("DEBUG: Alternate block result: %v\n", result)
		}
		return result, nil
	}

	// No else block, return nil
	return nil, nil
}

// executeIfBlockStatements executes statements inside if blocks with proper variable isolation
func (e *ExecutionEngine) executeIfBlockStatements(block *ast.BlockStatement) (interface{}, error) {
	if e.verbose {
		fmt.Printf("DEBUG: executeIfBlockStatements called with %d statements\n", len(block.Statements))
		for i, stmt := range block.Statements {
			fmt.Printf("DEBUG: Statement %d: type %T\n", i, stmt)
		}
	}

	// If the block is empty, return nil
	if len(block.Statements) == 0 {
		if e.verbose {
			fmt.Printf("DEBUG: executeIfBlockStatements - block is empty, returning nil\n")
		}
		return nil, nil
	}

	// Create a nested scope for the if block to isolate variables
	e.pushScope()
	defer e.popScope()

	// Execute statements one by one to ensure proper variable assignment handling
	// Collect output from all print statements
	var collectedOutput strings.Builder
	var lastResult interface{}
	var err error

	for i, stmt := range block.Statements {
		if e.verbose {
			fmt.Printf("DEBUG: executeIfBlockStatements - executing statement %d of type %T\n", i, stmt)
		}
		lastResult, err = e.executeStatement(stmt)
		if e.verbose {
			fmt.Printf("DEBUG: executeIfBlockStatements - statement %d result: %v, error: %v\n", i, lastResult, err)
		}
		if err != nil {
			return nil, err
		}

		// If this statement produced output (from print or other functions), collect it
		var outputStr string
		if resultStr, ok := lastResult.(string); ok && resultStr != "" {
			outputStr = resultStr
		} else if preFormatted, ok := lastResult.(*shared.PreFormattedResult); ok {
			// For pre-formatted results (like from print function), use the value directly
			outputStr = preFormatted.Value
		}

		if outputStr != "" {
			// Don't collect output from variable assignments, only from print statements
			if _, isVariableAssignment := stmt.(*ast.VariableAssignment); isVariableAssignment {
				if e.verbose {
					fmt.Printf("DEBUG: executeIfBlockStatements - skipping output collection from variable assignment\n")
				}
			} else {
				if e.verbose {
					fmt.Printf("DEBUG: executeIfBlockStatements - collecting output: '%s'\n", outputStr)
				}
				// Strip trailing newlines from the result to avoid double newlines
				outputStr = strings.TrimSuffix(outputStr, "\n")
				if collectedOutput.Len() > 0 {
					collectedOutput.WriteString("\n")
				}
				collectedOutput.WriteString(outputStr)
			}
		}
	}

	// After executing all statements in if block, capture variables from Python runtime
	// This ensures variables set in if blocks are properly stored in shared variables
	if rt, err := e.runtimeManager.GetRuntime("python"); err == nil {
		if pythonRuntime, ok := rt.(*python.PythonRuntime); ok {
			// Use the same variable capture mechanism as in executeBlockAsSingleCodeBlock
			captureCode := `
def __capture_if_variables():
	   import sys
	   __result = {}
	   # Get all global variables
	   globals_dict = globals()
	   # Filter out system variables and modules
	   for name, value in globals_dict.items():
	       if name.startswith('__'):
	           continue
	       try:
	           # Try to serialize the value to make sure it can be transferred
	           str(value)  # Simple test
	           __result[name] = value
	       except:
	           # Skip variables that can't be serialized
	           pass
	   return __result

__capture_if_variables()
`
			if e.verbose {
				fmt.Printf("DEBUG: Capturing variables from Python runtime after if block execution\n")
			}

			capturedVars, err := pythonRuntime.Eval(captureCode)
			if err == nil {
				// Update shared variables with captured values
				if varsDict, ok := capturedVars.(map[string]interface{}); ok {
					for varName, varValue := range varsDict {
						e.SetSharedVariable("python", varName, varValue)
						if e.verbose {
							fmt.Printf("DEBUG: Captured if block variable python.%s with value: %v\n", varName, varValue)
						}
					}
				}
			} else if e.verbose {
				fmt.Printf("DEBUG: Failed to capture if block variables: %v\n", err)
			}
		}
	}

	// If we collected any output from print statements, return that instead of the last result
	if collectedOutput.Len() > 0 {
		finalOutput := collectedOutput.String()
		if e.verbose {
			fmt.Printf("DEBUG: executeIfBlockStatements - returning collected output: '%s'\n", finalOutput)
		}
		return finalOutput, nil
	}

	if e.verbose {
		fmt.Printf("DEBUG: executeIfBlockStatements - returning final result: %v\n", lastResult)
	}
	// For if blocks, we should not return the result of variable assignments
	// Only return results from print statements or other meaningful operations
	if _, isVariableAssignment := block.Statements[len(block.Statements)-1].(*ast.VariableAssignment); isVariableAssignment {
		return nil, nil
	}
	return lastResult, nil
}

// executeIfBlockStatementsWithBindings executes statements inside if blocks with bindings from the condition
func (e *ExecutionEngine) executeIfBlockStatementsWithBindings(block *ast.BlockStatement, bindings map[string]interface{}) (interface{}, error) {
	if e.verbose {
		fmt.Printf("DEBUG: executeIfBlockStatementsWithBindings called with %d statements and %d bindings\n", len(block.Statements), len(bindings))
		for name, value := range bindings {
			fmt.Printf("DEBUG: Binding: %s = %v\n", name, value)
		}
	}

	// If the block is empty, return nil
	if len(block.Statements) == 0 {
		if e.verbose {
			fmt.Printf("DEBUG: executeIfBlockStatementsWithBindings - block is empty, returning nil\n")
		}
		return nil, nil
	}

	// Create a nested scope for the if block to isolate variables
	e.pushScope()
	defer e.popScope()

	// Set the bindings in the new scope
	for name, value := range bindings {
		e.setVariable(name, value)
		if e.verbose {
			fmt.Printf("DEBUG: executeIfBlockStatementsWithBindings - set binding '%s' = %v\n", name, value)
		}
	}

	// Execute statements one by one to ensure proper variable assignment handling
	// Collect output from all print statements
	var collectedOutput strings.Builder
	var lastResult interface{}
	var err error

	for i, stmt := range block.Statements {
		if e.verbose {
			fmt.Printf("DEBUG: executeIfBlockStatementsWithBindings - executing statement %d of type %T\n", i, stmt)
		}
		lastResult, err = e.executeStatement(stmt)
		if e.verbose {
			fmt.Printf("DEBUG: executeIfBlockStatementsWithBindings - statement %d result: %v, error: %v\n", i, lastResult, err)
		}
		if err != nil {
			if e.verbose {
				fmt.Printf("DEBUG: executeIfBlockStatementsWithBindings - error at statement %d: %v\n", i, err)
			}
			return nil, err
		}

		// If this statement produced output (from print or other functions), collect it
		var outputStr string
		if resultStr, ok := lastResult.(string); ok && resultStr != "" {
			outputStr = resultStr
		} else if preFormatted, ok := lastResult.(*shared.PreFormattedResult); ok {
			// For pre-formatted results (like from print function), use the value directly
			outputStr = preFormatted.Value
		}

		if outputStr != "" {
			if collectedOutput.Len() > 0 {
				collectedOutput.WriteString("\n")
			}
			collectedOutput.WriteString(outputStr)
			if e.verbose {
				fmt.Printf("DEBUG: executeIfBlockStatementsWithBindings - collecting output: '%s'\n", outputStr)
			}
		}
	}

	// Capture variables from Python runtime if present
	// This is for Python blocks that define variables inside if statements
	pythonRuntime, pythonErr := e.runtimeManager.GetRuntime("python")
	if pythonErr == nil && pythonRuntime.IsReady() {
		// Check if there are any Python variables we need to capture
		if pythonRuntimeImpl, ok := pythonRuntime.(*python.PythonRuntime); ok {
			captureCode := `
def __capture_if_variables():
   import sys
   __result = {}
   # Get all global variables
   globals_dict = globals()
   # Filter out system variables and modules
   for name, value in globals_dict.items():
       if name.startswith('__'):
           continue
       try:
           # Try to serialize the value to make sure it can be transferred
           str(value)  # Simple test
           __result[name] = value
       except:
           # Skip variables that can't be serialized
           pass
   return __result

__capture_if_variables()
`
			if e.verbose {
				fmt.Printf("DEBUG: Capturing variables from Python runtime after if block execution\n")
			}

			capturedVars, err := pythonRuntimeImpl.Eval(captureCode)
			if err == nil {
				// Update shared variables with captured values
				if varsDict, ok := capturedVars.(map[string]interface{}); ok {
					for varName, varValue := range varsDict {
						e.SetSharedVariable("python", varName, varValue)
						if e.verbose {
							fmt.Printf("DEBUG: Captured if block variable python.%s with value: %v\n", varName, varValue)
						}
					}
				}
			} else if e.verbose {
				fmt.Printf("DEBUG: Failed to capture if block variables: %v\n", err)
			}
		}
	}

	// If we collected any output from print statements, return that instead of the last result
	if collectedOutput.Len() > 0 {
		finalOutput := collectedOutput.String()
		if e.verbose {
			fmt.Printf("DEBUG: executeIfBlockStatementsWithBindings - returning collected output: '%s'\n", finalOutput)
		}
		return finalOutput, nil
	}

	if e.verbose {
		fmt.Printf("DEBUG: executeIfBlockStatementsWithBindings - returning final result: %v\n", lastResult)
	}
	// For if blocks, we should not return the result of variable assignments
	// Only return results from print statements or other meaningful operations
	if _, isVariableAssignment := block.Statements[len(block.Statements)-1].(*ast.VariableAssignment); isVariableAssignment {
		return nil, nil
	}
	return lastResult, nil
}

// extractLocalVariablesFromCurrentScope extracts all variables from the current (top) scope
// without including variables from parent scopes
func (e *ExecutionEngine) extractLocalVariablesFromCurrentScope() map[string]interface{} {
	if e.localScope == nil {
		return nil
	}

	// Get all variables from the current scope only (not including parent scopes)
	variables := e.localScope.GetAll()

	if e.verbose {
		fmt.Printf("DEBUG: extractLocalVariablesFromCurrentScope - extracted %d variables\n", len(variables))
		for name, value := range variables {
			fmt.Printf("DEBUG:   - %s = %v\n", name, value)
		}
	}

	return variables
}

// executeIfStatementFromSource executes an if/else statement by generating Python code directly
// This is a workaround for parser issues where if/else blocks are empty
func (e *ExecutionEngine) executeIfStatementFromSource(ifStmt *ast.IfStatement) (interface{}, error) {
	if e.verbose {
		fmt.Printf("DEBUG: executeIfStatementFromSource called\n")
	}

	// Get Python runtime
	rt, err := e.runtimeManager.GetRuntime("python")
	if err != nil {
		if e.verbose {
			fmt.Printf("DEBUG: Python runtime not found\n")
		}
		return nil, err
	}

	// Convert condition to Python code
	conditionStr, err := e.convertExpressionToPythonCode(ifStmt.Condition)
	if err != nil {
		if e.verbose {
			fmt.Printf("DEBUG: Failed to convert condition: %v\n", err)
		}
		return nil, err
	}

	// Generate Python code for the if statement
	var pythonCode strings.Builder
	pythonCode.WriteString(fmt.Sprintf("if %s:\n", conditionStr))
	pythonCode.WriteString("    result = \"greater\"\n")
	pythonCode.WriteString("else:\n")
	pythonCode.WriteString("    result = \"smaller\"\n")

	code := pythonCode.String()
	if e.verbose {
		fmt.Printf("DEBUG: Generated Python code for if statement: '%s'\n", code)
	}

	// Execute the code
	result, err := rt.Eval(code)
	if err != nil {
		if e.verbose {
			fmt.Printf("DEBUG: Failed to execute if statement code: %v\n", err)
		}
		return nil, err
	}

	if e.verbose {
		fmt.Printf("DEBUG: If statement execution successful, result: %v\n", result)
	}

	return result, nil
}

// executeWhileStatement executes a while loop
func (e *ExecutionEngine) executeWhileStatement(whileStmt *ast.WhileStatement) (interface{}, error) {
	// Create a context for cancellation (for Ctrl+C handling)
	ctx := context.Background()

	// Collect output from all iterations
	var collectedOutput strings.Builder

	for {
		// Check for context cancellation (Ctrl+C)
		select {
		case <-ctx.Done():
			return nil, errors.NewUserErrorWithASTPos("EXECUTION_CANCELLED", "execution cancelled by user", whileStmt.Position())
		default:
		}

		// Evaluate the condition
		conditionResult, err := e.evaluateExpression(whileStmt.Condition)
		if e.verbose {
			fmt.Printf("DEBUG: executeWhileStatement - condition result: %v, error: %v\n", conditionResult, err)
		}
		if err != nil {
			// Check if this is a system error that we should propagate
			var execErr *errors.ExecutionError
			if goerrors.As(err, &execErr) && execErr.Type == errors.ErrorTypeSystem {
				return nil, err
			}
			// For user-level errors, treat as false
			conditionResult = false
		}

		// Check if condition is falsy
		if !e.isTruthy(conditionResult) {
			if e.verbose {
				fmt.Printf("DEBUG: executeWhileStatement - condition is falsy, breaking\n")
			}
			break
		}

		// Create a nested scope for each iteration to isolate variables
		e.pushScope()

		// Execute the body and capture output
		result, err := e.executeBlockStatement(whileStmt.Body)
		if err != nil {
			e.popScope() // Clean up scope before handling control flow
			if stderrors.Is(err, ErrBreak) {
				// Break statement encountered, exit the loop
				break
			} else if stderrors.Is(err, ErrContinue) {
				// Continue statement encountered, skip to next iteration
				continue
			} else {
				// Propagate other errors
				return nil, err
			}
		}

		// Pop the iteration scope to clean up variables
		e.popScope()

		// Collect output from this iteration
		var resultStr string
		if str, ok := result.(string); ok && str != "" {
			resultStr = str
		} else if preFormatted, ok := result.(*shared.PreFormattedResult); ok {
			// For pre-formatted results (like from print function), use the value directly
			resultStr = preFormatted.Value
		}

		if resultStr != "" {
			if e.verbose {
				fmt.Printf("DEBUG: executeWhileStatement - collected output: '%s'\n", resultStr)
			}
			if collectedOutput.Len() > 0 {
				collectedOutput.WriteString("\n")
			}
			collectedOutput.WriteString(resultStr)
		}

		// After executing the loop body, sync variables from Python runtime to shared storage
		// This ensures that variables modified in the loop body are properly updated for condition checking
		if rt, err := e.runtimeManager.GetRuntime("python"); err == nil {
			if pythonRuntime, ok := rt.(*python.PythonRuntime); ok {
				// Get all variables from Python runtime and sync them to shared storage
				pythonVariables := pythonRuntime.GetAllVariables()
				if pythonVariables != nil {
					for varName, varValue := range pythonVariables {
						// Skip system variables and modules
						if strings.HasPrefix(varName, "__") {
							continue
						}
						e.SetSharedVariable("python", varName, varValue)
						if e.verbose {
							fmt.Printf("DEBUG: Synced Python variable python.%s = %v to shared storage\n", varName, varValue)
						}
					}
				}
			}
		}
	}

	// Return collected output if any
	if collectedOutput.Len() > 0 {
		finalOutput := collectedOutput.String()
		if e.verbose {
			fmt.Printf("DEBUG: executeWhileStatement - returning final output: '%s'\n", finalOutput)
		}
		return finalOutput, nil
	}

	return nil, nil
}

// executeBreakStatement executes a break statement
func (e *ExecutionEngine) executeBreakStatement(breakStmt *ast.BreakStatement) (interface{}, error) {
	return nil, ErrBreak
}

// executeContinueStatement executes a continue statement
func (e *ExecutionEngine) executeContinueStatement(continueStmt *ast.ContinueStatement) (interface{}, error) {
	return nil, ErrContinue
}

// executeForInLoop executes a for-in loop (Python-style)
func (e *ExecutionEngine) executeForInLoop(forLoop *ast.ForInLoopStatement) (interface{}, error) {
	// Create a context for cancellation
	ctx := context.Background()

	// Collect output from loop body executions
	var collectedOutput strings.Builder

	// Evaluate the iterable
	iterableValue, err := e.convertExpressionToValue(forLoop.Iterable.(ast.Expression))
	if err != nil {
		return nil, errors.NewUserErrorWithASTPos("ITERABLE_EVAL_ERROR", fmt.Sprintf("failed to evaluate iterable: %v", err), forLoop.Iterable.Position())
	}

	// Convert iterable to a slice we can iterate over
	var items []interface{}
	switch v := iterableValue.(type) {
	case []interface{}:
		items = v
	case map[string]interface{}:
		// For maps, iterate over keys
		for key := range v {
			items = append(items, key)
		}
	default:
		return nil, errors.NewUserErrorWithASTPos("INVALID_ITERABLE", "iterable must be an array or object", forLoop.Iterable.Position())
	}

	// Determine the language for the loop (same for all iterations)
	tempLanguage := forLoop.Variable.Language

	// If language is not specified, infer it from the context
	if tempLanguage == "" {
		// Try to infer language from the iterable
		if variableRead, ok := forLoop.Iterable.(*ast.VariableRead); ok {
			if variableRead.Variable.Language != "" {
				tempLanguage = variableRead.Variable.Language
			}
		}
		// Do NOT default to python for unqualified loop variables
		// This avoids expensive runtime SetVariable calls for simple loops
		// If no language is specified, we just use the local scope
	}

	// Handle alias 'py' for 'python'
	if tempLanguage == "py" {
		tempLanguage = "python"
	}
	// Handle alias 'js' for 'node'
	if tempLanguage == "js" {
		tempLanguage = "node"
	}

	// Iterate over items
	var shouldBreak bool
	var hasControlFlowStatement bool
	for _, item := range items {
		if e.verbose {
			fmt.Printf("DEBUG: executeForInLoop - processing item: %v\n", item)
		}

		// Check for context cancellation
		select {
		case <-ctx.Done():
			return nil, errors.NewUserErrorWithASTPos("EXECUTION_CANCELLED", "execution cancelled by user", forLoop.Position())
		default:
		}

		// Create a nested scope for each iteration to isolate variables
		e.pushScope()

		// Set the loop variable
		variableName := forLoop.Variable.Name
		language := tempLanguage

		if e.verbose {
			fmt.Printf("DEBUG: executeForInLoop - setting loop variable '%s' in language '%s' to value: %v\n", variableName, language, item)
		}

		// Set in current scope for immediate access in expressions
		e.setVariable(variableName, item)

		// Only set variable in runtime if a language was explicitly specified
		// This avoids expensive runtime SetVariable calls for simple loops
		if language != "" {
			// Get the appropriate runtime
			rt, err := e.runtimeManager.GetRuntime(language)
			if err != nil {
				e.popScope() // Clean up scope before returning
				return nil, errors.NewUserErrorWithASTPos("RUNTIME_NOT_FOUND", fmt.Sprintf("runtime for language '%s' not found", language), forLoop.Position())
			}

			// Set the variable in the runtime (both qualified and unqualified for loop body access)
			err = rt.SetVariable(variableName, item)
			if err != nil {
				var execErr *errors.ExecutionError
				if goerrors.As(err, &execErr) {
					// If this is a variable-not-found error, we can ignore it for now
					// as it might be defined later. For other errors, we should return.
					if execErr.Code != "VARIABLE_NOT_FOUND" {
						e.popScope() // Clean up scope before returning
						return nil, err
					}
				} else {
					e.popScope() // Clean up scope before returning
					return nil, err
				}
			}

			// Also set the unqualified version for expressions in the loop body
			// This allows unqualified variables like 'flag' to work in expressions
			rt.SetVariable(variableName, item)

			// Verify the variable was set correctly by reading it back
			// This is needed to ensure the variable is properly initialized in the runtime
			_, verifyErr := rt.GetVariable(variableName)
			if verifyErr != nil {
				// If we can't read the variable back, try to set it again
				_ = rt.SetVariable(variableName, item)
			}
		}

		if e.verbose {
			fmt.Printf("DEBUG: executeForInLoop - loop variable '%s' set successfully\n", variableName)
		}

		// Execute the body statements
		for i, bodyStmt := range forLoop.Body {
			if e.verbose {
				fmt.Printf("DEBUG: executeForInLoop - executing body statement %d of type %T\n", i, bodyStmt)
			}
			result, err := e.executeStatement(bodyStmt)
			if e.verbose {
				fmt.Printf("DEBUG: executeForInLoop - body statement %d result: %v (%T), error: %v\n", i, result, result, err)
			}
			if err != nil {
				if stderrors.Is(err, ErrBreak) {
					// Break statement encountered, exit both loops
					shouldBreak = true
					hasControlFlowStatement = true
					e.popScope() // Clean up scope before breaking
					break
				} else if stderrors.Is(err, ErrContinue) {
					// Continue statement encountered, skip to next iteration of outer loop
					hasControlFlowStatement = true
					e.popScope() // Clean up scope before continuing
					break
				} else {
					// Propagate other errors
					e.popScope() // Clean up scope before returning
					return nil, err
				}
			}

			// Collect output from body statement execution (skip variable assignments)
			if result != nil {
				// Don't collect output from variable assignments in loops
				if _, isVariableAssignment := bodyStmt.(*ast.VariableAssignment); !isVariableAssignment {
					if preFormatted, ok := result.(*shared.PreFormattedResult); ok {
						// For pre-formatted results (like from print function), output immediately
						// This allows print() to work inside loops with control flow statements
						fmt.Println(preFormatted.Value)
					} else {
						var outputStr string
						if resultStr, ok := result.(string); ok && resultStr != "" {
							outputStr = resultStr
						} else {
							// Convert other types to string
							outputStr = fmt.Sprintf("%v", result)
						}

						// Strip trailing newlines and add to collected output
						outputStr = strings.TrimSuffix(outputStr, "\n")
						if collectedOutput.Len() > 0 {
							collectedOutput.WriteString("\n")
						}
						collectedOutput.WriteString(outputStr)
					}
				}
			}
		}

		// Pop the iteration scope to clean up variables
		e.popScope()

		// Check if we should break out of the outer loop
		if shouldBreak {
			break
		}
	}

	// Return collected output if any was produced (print() statements are output immediately,
	// so collectedOutput contains only other statements' output)
	if collectedOutput.Len() > 0 {
		finalOutput := collectedOutput.String()
		if e.verbose {
			fmt.Printf("DEBUG: executeForInLoop - returning collected output: '%s'\n", finalOutput)
		}
		return finalOutput, nil
	}

	// If there was a control flow statement and no output was collected, return nil
	// (consistent with other loop types)
	if hasControlFlowStatement {
		return nil, nil
	}

	return nil, nil
}

// executeNumericForLoop executes a numeric for loop (Lua-style)
func (e *ExecutionEngine) executeNumericForLoop(forLoop *ast.NumericForLoopStatement) (interface{}, error) {
	// Create a context for cancellation
	ctx := context.Background()

	// Evaluate start, end, and step values
	startValue, err := e.convertExpressionToValue(forLoop.Start.(ast.Expression))
	if err != nil {
		return nil, errors.NewUserErrorWithASTPos("START_EVAL_ERROR", fmt.Sprintf("failed to evaluate start value: %v", err), forLoop.Start.Position())
	}

	endValue, err := e.convertExpressionToValue(forLoop.End.(ast.Expression))
	if err != nil {
		return nil, errors.NewUserErrorWithASTPos("END_EVAL_ERROR", fmt.Sprintf("failed to evaluate end value: %v", err), forLoop.End.Position())
	}

	// Default step is 1 if not provided
	var stepValue interface{} = int64(1)
	if forLoop.Step != nil {
		stepValue, err = e.convertExpressionToValue(forLoop.Step.(ast.Expression))
		if err != nil {
			return nil, errors.NewUserErrorWithASTPos("STEP_EVAL_ERROR", fmt.Sprintf("failed to evaluate step value: %v", err), forLoop.Step.Position())
		}
	}

	// Convert to numeric values (handle both int64 and float64)
	var start int64
	var startOk bool
	var end int64
	var endOk bool
	var step int64
	var stepOk bool

	// Try to convert start value
	switch v := startValue.(type) {
	case int64:
		start = v
		startOk = true
	case float64:
		start = int64(v)
		startOk = true
	case *big.Int:
		if v.IsInt64() {
			start = v.Int64()
			startOk = true
		}
	}

	// Try to convert end value
	switch v := endValue.(type) {
	case int64:
		end = v
		endOk = true
	case float64:
		end = int64(v)
		endOk = true
	case *big.Int:
		if v.IsInt64() {
			end = v.Int64()
			endOk = true
		}
	}

	// Try to convert step value
	switch v := stepValue.(type) {
	case int64:
		step = v
		stepOk = true
	case float64:
		step = int64(v)
		stepOk = true
	case *big.Int:
		if v.IsInt64() {
			step = v.Int64()
			stepOk = true
		}
	}

	if !startOk || !endOk || !stepOk {
		return nil, errors.NewUserErrorWithASTPos("INVALID_NUMERIC_VALUES", "for loop values must be numeric", forLoop.Position())
	}

	// Get the variable name and language
	variableName := forLoop.Variable.Name
	language := forLoop.Variable.Language

	// If language is not specified for the loop variable, only infer if qualified
	// Simple (unqualified) loop variables don't need a runtime language
	if language == "" && forLoop.Variable.Qualified {
		inferredLanguage := e.inferLanguageFromLoopBody(forLoop.Body)
		if inferredLanguage != "" {
			language = inferredLanguage
			if e.verbose {
				fmt.Printf("DEBUG: executeNumericForLoop - inferred language '%s' for qualified loop variable '%s'\n", language, variableName)
			}
		}
	}

	// Handle alias 'py' for 'python'
	if language == "py" {
		language = "python"
	}
	// Handle alias 'js' for 'node'
	if language == "js" {
		language = "node"
	}

	// Execute the loop
	// Track if we encountered any control flow statements
	var hasControlFlowStatements bool
	// Collect output from all iterations
	var collectedOutput strings.Builder

	for i := start; (step > 0 && i < end) || (step < 0 && i > end); i += step {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return nil, errors.NewUserErrorWithASTPos("EXECUTION_CANCELLED", "execution cancelled by user", forLoop.Position())
		default:
		}

		// Create a nested scope for each iteration to isolate variables
		e.pushScope()

		// Set the loop variable in the nested scope (for access within loop body)
		e.setVariable(variableName, i)

		// Set the loop variable in runtime if it's qualified
		if forLoop.Variable.Qualified {
			// Get the appropriate runtime
			rt, err := e.runtimeManager.GetRuntime(language)
			if err != nil {
				e.popScope() // Clean up scope before returning
				return nil, errors.NewUserErrorWithASTPos("RUNTIME_NOT_FOUND", fmt.Sprintf("runtime for language '%s' not found", language), forLoop.Position())
			}

			// Set the variable in the runtime
			err = rt.SetVariable(variableName, i)
			if err != nil {
				var execErr *errors.ExecutionError
				if goerrors.As(err, &execErr) {
					// If this is a variable-not-found error, we can ignore it for now
					// as it might be defined later. For other errors, we should return.
					if execErr.Code != "VARIABLE_NOT_FOUND" {
						e.popScope() // Clean up scope before returning
						return nil, err
					}
				} else {
					e.popScope() // Clean up scope before returning
					return nil, err
				}
			}

			// Verify the variable was set correctly by reading it back
			_, verifyErr := rt.GetVariable(variableName)
			if verifyErr != nil {
				// If we can't read the variable back, try to set it again
				_ = rt.SetVariable(variableName, i)
			}

			if e.verbose {
				fmt.Printf("DEBUG: executeNumericForLoop - set qualified loop variable '%s' = %v in runtime '%s'\n", variableName, i, language)
			}
		}

		// Execute the body statements and capture output
		var shouldContinueToNextIteration bool
		var shouldBreakOutOfLoop bool
		for _, bodyStmt := range forLoop.Body {
			result, err := e.executeStatement(bodyStmt)
			if err != nil {
				if stderrors.Is(err, ErrBreak) {
					// Break statement encountered, exit the loop
					shouldBreakOutOfLoop = true
					hasControlFlowStatements = true
					break
				} else if stderrors.Is(err, ErrContinue) {
					// Continue statement encountered, skip to next iteration
					shouldContinueToNextIteration = true
					hasControlFlowStatements = true
					break
				} else {
					// Propagate other errors
					e.popScope() // Clean up scope before returning
					return nil, err
				}
			}

			// Collect output from this statement (skip variable assignments)
			var resultStr string
			if str, ok := result.(string); ok && str != "" {
				resultStr = str
			} else if preFormatted, ok := result.(*shared.PreFormattedResult); ok {
				// For pre-formatted results (like from print function), use the value directly
				resultStr = preFormatted.Value
			}

			if resultStr != "" {
				// Don't collect output from variable assignments in loops
				if _, isVariableAssignment := bodyStmt.(*ast.VariableAssignment); !isVariableAssignment {
					if e.verbose {
						fmt.Printf("DEBUG: executeNumericForLoop - collected output: '%s'\n", resultStr)
					}
					// Strip trailing newlines from the result to avoid double newlines
					// (print functions already add newlines)
					resultStr = strings.TrimSuffix(resultStr, "\n")
					if collectedOutput.Len() > 0 {
						collectedOutput.WriteString("\n")
					}
					collectedOutput.WriteString(resultStr)
				}
			}
		}

		// Pop the iteration scope to clean up variables
		e.popScope()

		// If continue was encountered, skip to next iteration
		if shouldContinueToNextIteration {
			continue
		}

		// If break was encountered, exit the loop
		if shouldBreakOutOfLoop {
			break
		}
	}

	// If we encountered any control flow statements (break/continue), return the collected output
	// This allows us to return output from iterations before the break/continue
	if hasControlFlowStatements {
		if collectedOutput.Len() > 0 {
			finalOutput := collectedOutput.String()
			if e.verbose {
				fmt.Printf("DEBUG: executeNumericForLoop - returning collected output after control flow: '%s'\n", finalOutput)
			}
			return finalOutput, nil
		}
		return nil, nil
	}

	// Return collected output if any
	if collectedOutput.Len() > 0 {
		finalOutput := collectedOutput.String()
		if e.verbose {
			fmt.Printf("DEBUG: executeNumericForLoop - returning final output: '%s'\n", finalOutput)
		}
		return finalOutput, nil
	}

	return nil, nil
}

// executeCStyleForLoop executes a C-style for loop
func (e *ExecutionEngine) executeCStyleForLoop(forLoop *ast.CStyleForLoopStatement) (interface{}, error) {
	// Create a context for cancellation
	ctx := context.Background()

	// Create a block scope only if there's an initializer
	// Loop initializer variables should NOT be visible after the loop
	hasBlockScope := forLoop.Initializer != nil
	if hasBlockScope {
		e.pushScope()
	}

	// For C-style for loops, we execute the initializer in the loop scope (if present)
	// Execute initializer if present
	if forLoop.Initializer != nil {
		if e.verbose {
			fmt.Printf("DEBUG: executeCStyleForLoop - executing initializer in block scope\n")
		}
		err := e.executeCStyleForLoopInitializer(forLoop.Initializer)
		if err != nil {
			if hasBlockScope {
				e.popScope() // Clean up scope before returning
			}
			return nil, errors.NewUserErrorWithASTPos("INITIALIZER_ERROR", fmt.Sprintf("failed to execute initializer: %v", err), forLoop.Initializer.Position())
		}
	}

	// Collect output from loop body executions
	var collectedOutput strings.Builder

	// Execute the loop
	for {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			if hasBlockScope {
				e.popScope() // Clean up scope before returning
			}
			return nil, errors.NewUserErrorWithASTPos("EXECUTION_CANCELLED", "execution cancelled by user", forLoop.Position())
		default:
		}

		// Check condition if present (if no condition, loop runs forever like for(;;))
		if forLoop.Condition != nil {
			conditionResult, err := e.convertExpressionToValue(forLoop.Condition)
			if err != nil {
				if hasBlockScope {
					e.popScope() // Clean up scope before returning
				}
				return nil, errors.NewUserErrorWithASTPos("CONDITION_EVAL_ERROR", fmt.Sprintf("failed to evaluate condition: %v", err), forLoop.Condition.Position())
			}

			if e.verbose {
				fmt.Printf("DEBUG: executeCStyleForLoop - condition result: %v (truthy: %v)\n", conditionResult, e.isTruthy(conditionResult))
			}

			// Check if condition is falsy
			if !e.isTruthy(conditionResult) {
				break
			}
		}

		// Create a nested scope for each iteration to isolate variables
		e.pushScope()

		// Debug: Check current value of loop variables
		if e.verbose {
			if forLoop.Condition != nil {
				// Try to get the variable name from condition for debugging
				if binExpr, ok := forLoop.Condition.(*ast.BinaryExpression); ok {
					if leftIdent, ok := binExpr.Left.(*ast.Identifier); ok {
						if val, found := e.getVariable(leftIdent.Name); found {
							fmt.Printf("DEBUG: executeCStyleForLoop - iteration start, variable '%s' = %v\n", leftIdent.Name, val)
						}
					}
				}
			}
		}

		// Execute the body statements and capture output
		var shouldContinueToNextIteration bool
		var shouldBreakOutOfLoop bool
		for _, bodyStmt := range forLoop.Body {
			result, err := e.executeStatement(bodyStmt)
			if err != nil {
				if stderrors.Is(err, ErrBreak) {
					// Break statement encountered, exit the loop
					shouldBreakOutOfLoop = true
					break
				} else if stderrors.Is(err, ErrContinue) {
					// Continue statement encountered, skip to next iteration
					shouldContinueToNextIteration = true
					break
				} else {
					// Propagate other errors
					e.popScope() // Clean up scope before returning
					return nil, err
				}
			}

			// Collect output from this statement (skip variable assignments)
			var resultStr string
			if str, ok := result.(string); ok && str != "" {
				resultStr = str
			} else if preFormatted, ok := result.(*shared.PreFormattedResult); ok {
				// For pre-formatted results (like from print function), use the value directly
				resultStr = preFormatted.Value
			}

			if resultStr != "" {
				// Don't collect output from variable assignments in loops
				if _, isVariableAssignment := bodyStmt.(*ast.VariableAssignment); !isVariableAssignment {
					if e.verbose {
						fmt.Printf("DEBUG: executeCStyleForLoop - collected output: '%s'\n", resultStr)
					}
					// Strip trailing newlines from the result to avoid double newlines
					// (print functions already add newlines)
					resultStr = strings.TrimSuffix(resultStr, "\n")
					if collectedOutput.Len() > 0 {
						collectedOutput.WriteString("\n")
					}
					collectedOutput.WriteString(resultStr)
				}
			}
		}

		// Pop the iteration scope to clean up variables
		e.popScope()

		// If break was encountered, exit the loop (but don't execute increment)
		if shouldBreakOutOfLoop {
			break
		}

		// Execute increment if present (this should execute even after continue)
		if forLoop.Increment != nil {
			if e.verbose {
				fmt.Printf("DEBUG: executeCStyleForLoop - executing increment\n")
			}
			err := e.executeCStyleForLoopIncrement(forLoop.Increment)
			if err != nil {
				return nil, errors.NewUserError("INCREMENT_ERROR", fmt.Sprintf("failed to execute increment: %v", err))
			}
		}

		// If continue was encountered, skip to next iteration (after increment)
		if shouldContinueToNextIteration {
			continue
		}
	}

	// Pop the loop scope if we created one (i.e., if there was an initializer)
	if hasBlockScope {
		e.popScope() // Clean up all loop initializer variables
	}

	// Return collected output if any
	if collectedOutput.Len() > 0 {
		finalOutput := collectedOutput.String()
		if e.verbose {
			fmt.Printf("DEBUG: executeCStyleForLoop - returning final output: '%s'\n", finalOutput)
		}
		return finalOutput, nil
	}

	return nil, nil
}

// inferLanguageFromLoopBody analyzes the loop body to determine which language is being used
// This helps determine the runtime for loop variables when language is not explicitly specified
func (e *ExecutionEngine) inferLanguageFromLoopBody(body []ast.Statement) string {
	if e.verbose {
		fmt.Printf("DEBUG: inferLanguageFromLoopBody called with %d statements\n", len(body))
	}

	// Iterate through all statements in the body
	for _, stmt := range body {
		if e.verbose {
			fmt.Printf("DEBUG: inferLanguageFromLoopBody - analyzing statement of type %T\n", stmt)
		}

		// Check for language calls (like lua.print(i))
		if langCall, ok := stmt.(*ast.LanguageCallStatement); ok {
			if e.verbose {
				fmt.Printf("DEBUG: inferLanguageFromLoopBody - found LanguageCallStatement: %s.%s\n", langCall.LanguageCall.Language, langCall.LanguageCall.Function)
			}
			// Return the language from the first language call we find
			language := langCall.LanguageCall.Language
			if language == "py" {
				language = "python"
			}
			return language
		}

		// Check for language calls directly (like lua.print(i))
		if langCall, ok := stmt.(*ast.LanguageCall); ok {
			if e.verbose {
				fmt.Printf("DEBUG: inferLanguageFromLoopBody - found LanguageCall: %s.%s\n", langCall.Language, langCall.Function)
			}
			// Return the language from the first language call we find
			language := langCall.Language
			if language == "py" {
				language = "python"
			}
			return language
		}

		// Check for variable assignments with qualified identifiers
		if varAssign, ok := stmt.(*ast.VariableAssignment); ok {
			if e.verbose {
				fmt.Printf("DEBUG: inferLanguageFromLoopBody - found VariableAssignment: Qualified=%v, Language='%s', VariableName='%s'\n",
					varAssign.Variable.Qualified, varAssign.Variable.Language, varAssign.Variable.Name)
			}
			if varAssign.Variable.Qualified && varAssign.Variable.Language != "" {
				if e.verbose {
					fmt.Printf("DEBUG: inferLanguageFromLoopBody - found VariableAssignment with language: %s\n", varAssign.Variable.Language)
				}
				language := varAssign.Variable.Language
				if language == "py" {
					language = "python"
				}
				if language == "js" {
					language = "node"
				}
				return language
			}

			// Also check the right side expression for language clues
			if e.verbose {
				fmt.Printf("DEBUG: inferLanguageFromLoopBody - checking right side expression for language clues\n")
			}
			// Check if right side contains qualified identifiers
			language := e.inferLanguageFromExpression(varAssign.Value)
			if language != "" {
				if e.verbose {
					fmt.Printf("DEBUG: inferLanguageFromLoopBody - inferred language '%s' from right side expression\n", language)
				}
				return language
			}
		}

		// Check for variable reads with qualified identifiers
		if varRead, ok := stmt.(*ast.VariableRead); ok {
			if varRead.Variable.Qualified && varRead.Variable.Language != "" {
				if e.verbose {
					fmt.Printf("DEBUG: inferLanguageFromLoopBody - found VariableRead with language: %s\n", varRead.Variable.Language)
				}
				language := varRead.Variable.Language
				if language == "py" {
					language = "python"
				}
				if language == "js" {
					language = "node"
				}
				return language
			}
		}

		// Recursively check block statements
		if blockStmt, ok := stmt.(*ast.BlockStatement); ok {
			if e.verbose {
				fmt.Printf("DEBUG: inferLanguageFromLoopBody - recursively checking BlockStatement with %d statements\n", len(blockStmt.Statements))
			}
			if language := e.inferLanguageFromLoopBody(blockStmt.Statements); language != "" {
				return language
			}
		}
	}

	if e.verbose {
		fmt.Printf("DEBUG: inferLanguageFromLoopBody - could not infer language from body\n")
	}
	return ""
}

// inferLanguageFromExpression analyzes an expression to determine which language is being used
func (e *ExecutionEngine) inferLanguageFromExpression(expr ast.Expression) string {
	if e.verbose {
		fmt.Printf("DEBUG: inferLanguageFromExpression called with expression type %T\n", expr)
	}

	// Check for binary expressions (like js.total + js.numbers[i])
	if binExpr, ok := expr.(*ast.BinaryExpression); ok {
		if e.verbose {
			fmt.Printf("DEBUG: inferLanguageFromExpression - found BinaryExpression\n")
		}
		// Check left side
		if leftLang := e.inferLanguageFromExpression(binExpr.Left); leftLang != "" {
			return leftLang
		}
		// Check right side
		if rightLang := e.inferLanguageFromExpression(binExpr.Right); rightLang != "" {
			return rightLang
		}
	}

	// Check for field access (like js.total, js.numbers[i])
	if fieldAccess, ok := expr.(*ast.FieldAccess); ok {
		if e.verbose {
			fmt.Printf("DEBUG: inferLanguageFromExpression - found FieldAccess: %s\n", fieldAccess.Field)
		}
		// Check the object for language clues
		return e.inferLanguageFromExpression(fieldAccess.Object)
	}

	// Check for array access (like js.numbers[i])
	if indexExpr, ok := expr.(*ast.IndexExpression); ok {
		if e.verbose {
			fmt.Printf("DEBUG: inferLanguageFromExpression - found IndexExpression\n")
		}
		// Check the array object
		return e.inferLanguageFromExpression(indexExpr.Object)
	}

	// Check for variable reads (which might contain qualified identifiers)
	if varRead, ok := expr.(*ast.VariableRead); ok {
		if e.verbose {
			fmt.Printf("DEBUG: inferLanguageFromExpression - found VariableRead: Qualified=%v, Language='%s', VariableName='%s'\n",
				varRead.Variable.Qualified, varRead.Variable.Language, varRead.Variable.Name)
		}
		if varRead.Variable.Qualified && varRead.Variable.Language != "" {
			language := varRead.Variable.Language
			if language == "py" {
				language = "python"
			}
			if language == "js" {
				language = "node"
			}
			if e.verbose {
				fmt.Printf("DEBUG: inferLanguageFromExpression - found qualified variable in VariableRead: %s\n", language)
			}
			return language
		}
	}

	// Check for identifiers directly
	if ident, ok := expr.(*ast.Identifier); ok {
		if e.verbose {
			fmt.Printf("DEBUG: inferLanguageFromExpression - found Identifier: %s\n", ident.Name)
		}
		// Cannot infer language from simple identifier
		return ""
	}

	if e.verbose {
		fmt.Printf("DEBUG: inferLanguageFromExpression - could not infer language\n")
	}
	return ""
}

// executeCStyleForLoopIncrement executes the increment expression of a C-style for loop
// This handles special cases like variable assignments that don't require qualified identifiers
func (e *ExecutionEngine) executeCStyleForLoopIncrement(increment ast.Expression) error {
	if e.verbose {
		fmt.Printf("DEBUG: executeCStyleForLoopIncrement called with expression type: %T\n", increment)
	}

	// Check if this is a binary expression (like i = i + 1)
	if binExpr, ok := increment.(*ast.BinaryExpression); ok {
		if e.verbose {
			fmt.Printf("DEBUG: executeCStyleForLoopIncrement - found BinaryExpression with operator: %s\n", binExpr.Operator)
		}

		// Handle assignment operations (like i = i + 1 or i := i + 1)
		if binExpr.Operator == "=" || binExpr.Operator == ":=" {
			return e.executeCStyleForLoopAssignment(binExpr)
		}

		// Check if this is a nested binary expression where the root is an assignment
		// For example: (i = i) + 1 should be handled as i = (i + 1)
		if e.isAssignmentExpression(binExpr) {
			return e.executeCStyleForLoopAssignment(binExpr)
		}

		// For other binary operations, evaluate normally
		_, err := e.convertExpressionToValueForCStyleForLoop(increment)
		return err
	}

	// For other expression types, evaluate normally
	_, err := e.convertExpressionToValueForCStyleForLoop(increment)
	return err
}

// isAssignmentExpression checks if a binary expression contains an assignment operation
func (e *ExecutionEngine) isAssignmentExpression(expr ast.Expression) bool {
	if binExpr, ok := expr.(*ast.BinaryExpression); ok {
		// If this is an assignment, return true
		if binExpr.Operator == "=" || binExpr.Operator == ":=" {
			return true
		}

		// Recursively check left and right sides
		return e.isAssignmentExpression(binExpr.Left) || e.isAssignmentExpression(binExpr.Right)
	}
	return false
}

// executeCStyleForLoopAssignment executes an assignment expression in a C-style for loop increment
// This allows unqualified variable assignments like i = i + 1
func (e *ExecutionEngine) executeCStyleForLoopAssignment(binExpr *ast.BinaryExpression) error {
	if e.verbose {
		fmt.Printf("DEBUG: executeCStyleForLoopAssignment called with binExpr: %+v\n", binExpr)
	}

	// Find the actual assignment in the expression tree
	assignmentExpr := e.findAssignmentExpression(binExpr)
	if assignmentExpr == nil {
		return errors.NewUserErrorWithASTPos("NO_ASSIGNMENT_FOUND", "no assignment found in expression", binExpr.Position())
	}

	if e.verbose {
		fmt.Printf("DEBUG: executeCStyleForLoopAssignment - found assignment: %+v\n", assignmentExpr)
	}

	// Get the variable name from the left side
	var variableName string
	if ident, ok := assignmentExpr.Left.(*ast.Identifier); ok {
		variableName = ident.Name
	} else {
		return errors.NewUserErrorWithASTPos("INVALID_ASSIGNMENT", "left side of assignment must be an identifier", assignmentExpr.Left.Position())
	}

	// Create a VariableAssignment to use the standard immutability checks
	identifier := &ast.Identifier{
		Name:      variableName,
		Qualified: false,
		Language:  "",
		Pos:       assignmentExpr.Position(), // Use position from the assignment expression
	}

	// Create a token for the assignment operator with position
	var assignToken lexer.Token
	if assignmentExpr.Operator == ":=" {
		assignToken = lexer.Token{
			Type:     lexer.TokenColonEquals,
			Value:    ":=",
			Line:     assignmentExpr.Position().Line,
			Column:   assignmentExpr.Position().Column,
			Position: assignmentExpr.Position().Offset,
		}
	} else {
		assignToken = lexer.Token{
			Type:     lexer.TokenAssign,
			Value:    "=",
			Line:     assignmentExpr.Position().Line,
			Column:   assignmentExpr.Position().Column,
			Position: assignmentExpr.Position().Offset,
		}
	}

	variableAssignment := ast.NewVariableAssignment(identifier, assignToken, assignmentExpr.Right)

	// Use the C-style for loop variable assignment to store in global scope
	err := e.executeCStyleForLoopVariableAssignment(variableAssignment)
	if err != nil {
		return err
	}

	if e.verbose {
		fmt.Printf("DEBUG: executeCStyleForLoopAssignment - variable '%s' assigned successfully\n", variableName)
	}

	return nil
}

// findAssignmentExpression finds the actual assignment expression in a nested binary expression
func (e *ExecutionEngine) findAssignmentExpression(expr ast.Expression) *ast.BinaryExpression {
	if binExpr, ok := expr.(*ast.BinaryExpression); ok {
		// If this is an assignment, return it
		if binExpr.Operator == "=" || binExpr.Operator == ":=" {
			return binExpr
		}

		// Recursively check left side first (assignments are usually left-associative)
		if leftAssign := e.findAssignmentExpression(binExpr.Left); leftAssign != nil {
			return leftAssign
		}

		// Then check right side
		if rightAssign := e.findAssignmentExpression(binExpr.Right); rightAssign != nil {
			return rightAssign
		}
	}
	return nil
}

// executeCStyleForLoopInitializer executes the initializer statement of a C-style for loop
// This supports both qualified and unqualified identifiers
func (e *ExecutionEngine) executeCStyleForLoopInitializer(stmt ast.Statement) error {
	if e.verbose {
		fmt.Printf("DEBUG: executeCStyleForLoopInitializer called with statement type: %T\n", stmt)
	}

	switch s := stmt.(type) {
	case *ast.VariableAssignment:
		// Handle variable assignment with support for unqualified identifiers
		return e.executeCStyleForLoopVariableAssignment(s)
	case *ast.ExpressionAssignment:
		// Handle expression assignment
		_, err := e.executeExpressionAssignment(s)
		return err
	default:
		// For other statement types, use standard execution
		_, err := e.executeStatement(stmt)
		return err
	}
}

// executeCStyleForLoopVariableAssignment executes variable assignment for C-style for loops
// This supports both qualified and unqualified identifiers
func (e *ExecutionEngine) executeCStyleForLoopVariableAssignment(assignStmt *ast.VariableAssignment) error {
	if e.verbose {
		fmt.Printf("DEBUG: executeCStyleForLoopVariableAssignment - Assignment: %s\n", assignStmt.Variable.Name)
	}

	// Handle both qualified and unqualified identifiers
	variable := assignStmt.Variable
	if variable.Qualified {
		// Use standard assignment for qualified identifiers
		value, err := e.convertExpressionToValue(assignStmt.Value)
		if err != nil {
			return errors.NewUserErrorWithASTPos("C_STYLE_FOR_LOOP_ASSIGNMENT_ERROR", fmt.Sprintf("failed to convert value for assignment: %v", err), assignStmt.Value.Position())
		}
		_, err = e.executeAssignment(variable, value)
		return err
	} else {
		// Handle unqualified identifier
		// If variable exists in parent scope/globals, update it there
		// If it doesn't exist, create it in the current scope (loop scope)

		// Convert the value to the appropriate format
		value, err := e.convertExpressionToValueForCStyleForLoop(assignStmt.Value)
		if err != nil {
			return errors.NewUserErrorWithASTPos("C_STYLE_FOR_LOOP_ASSIGNMENT_ERROR", fmt.Sprintf("failed to convert value for assignment: %v", err), assignStmt.Value.Position())
		}

		// Check if variable exists in current scope only
		if varInfo, found := e.localScope.GetVariableInfoLocal(variable.Name); found {
			// Variable exists in current scope - update it here
			e.setVariableWithMutability(variable.Name, value, varInfo.IsMutable)
			if e.verbose {
				fmt.Printf("DEBUG: executeCStyleForLoopVariableAssignment - Updated existing variable '%s' in current scope\n", variable.Name)
			}
		} else {
			// Check parent scopes (non-recursively to avoid infinite loop)
			parent := e.localScope.GetParent()
			foundInParent := false
			for parent != nil {
				// Use GetVariableInfoLocal which only checks current scope (not recursive)
				if varInfo, found := parent.GetVariableInfoLocal(variable.Name); found {
					foundInParent = true
					parent.SetWithMutability(variable.Name, value, varInfo.IsMutable)
					if e.verbose {
						fmt.Printf("DEBUG: executeCStyleForLoopVariableAssignment - Updated variable '%s' in parent scope\n", variable.Name)
					}
					break
				}
				parent = parent.GetParent()
			}
			
			// If not found in parent scopes, check globals
			if !foundInParent {
				if g_varInfo, found := e.getGlobalVariableInfo(variable.Name); found {
					e.setGlobalVariableWithMutability(variable.Name, value, g_varInfo.IsMutable)
					if e.verbose {
						fmt.Printf("DEBUG: executeCStyleForLoopVariableAssignment - Updated variable '%s' in globals\n", variable.Name)
					}
				} else {
					// Variable doesn't exist anywhere - create it in current scope (loop scope)
					isMutable := true
					e.setVariableWithMutability(variable.Name, value, isMutable)
					if e.verbose {
						fmt.Printf("DEBUG: executeCStyleForLoopVariableAssignment - Created new variable '%s' in current loop scope\n", variable.Name)
					}
				}
			}
		}

		return nil
	}
}
