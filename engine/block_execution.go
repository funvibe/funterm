package engine

import (
	stderrors "errors"
	"fmt"
	"strings"

	"funterm/runtime/python"
	"funterm/shared"
	"go-parser/pkg/ast"
)

// executeBlockStatement executes a block statement (a list of statements)
func (e *ExecutionEngine) executeBlockStatement(block *ast.BlockStatement) (interface{}, error) {
	if e.verbose {
		fmt.Printf("DEBUG: executeBlockStatement called with %d statements\n", len(block.Statements))
		for i, stmt := range block.Statements {
			fmt.Printf("DEBUG: Statement %d: type %T\n", i, stmt)
		}
	}

	// If the block is empty, return nil
	if len(block.Statements) == 0 {
		if e.verbose {
			fmt.Printf("DEBUG: executeBlockStatement - block is empty, returning nil\n")
		}
		return nil, nil
	}

	// Check if all statements in the block are Python statements that can be executed together
	canExecuteAsSingle := e.canExecuteAsSingleCodeBlock(block)
	if e.verbose {
		fmt.Printf("DEBUG: executeBlockStatement - canExecuteAsSingleCodeBlock: %v\n", canExecuteAsSingle)
	}

	if canExecuteAsSingle {
		if e.verbose {
			fmt.Printf("DEBUG: executeBlockStatement - executing as single code block\n")
		}
		return e.executeBlockAsSingleCodeBlock(block)
	}

	// Otherwise, execute statements one by one (fallback behavior)
	if e.verbose {
		fmt.Printf("DEBUG: executeBlockStatement - executing statements one by one (fallback)\n")
	}
	var collectedOutput strings.Builder
	var lastResult interface{}
	var err error


	for i, stmt := range block.Statements {
		if e.verbose {
			fmt.Printf("DEBUG: executeBlockStatement - executing statement %d of type %T\n", i, stmt)
		}
		lastResult, err = e.executeStatement(stmt)
		if e.verbose {
			fmt.Printf("DEBUG: executeBlockStatement - statement %d result: %v, error: %v\n", i, lastResult, err)
		}
		if err != nil {
			// Check if this is a control flow error (break/continue) that should be propagated
			if stderrors.Is(err, ErrBreak) || stderrors.Is(err, ErrContinue) {
				// Propagate break/continue errors to the calling loop
				return nil, err
			}
			// For other execution errors (including immutable variable errors): propagate them to stop execution
			if e.verbose {
				fmt.Printf("DEBUG: executeBlockStatement - propagating execution error: %v\n", err)
			}
			return nil, err
		}

		// If this statement produced output (from print statements, function calls, or nested blocks), collect it
		if e.verbose {
			fmt.Printf("DEBUG: executeBlockStatement - processing statement result: %v (type: %T)\n", lastResult, lastResult)
		}
		if lastResult != nil {
			// Special handling for bitstring pattern assignments - always collect their results in batch mode
			if _, isBitstringPatternAssignment := stmt.(*ast.BitstringPatternAssignment); isBitstringPatternAssignment {
				outputStr := fmt.Sprintf("%v", lastResult)
				if e.verbose {
					fmt.Printf("DEBUG: executeBlockStatement - collecting bitstring pattern assignment result: '%s'\n", outputStr)
				}
				if collectedOutput.Len() > 0 {
					collectedOutput.WriteString("\n")
				}
				collectedOutput.WriteString(outputStr)
			} else if _, isVariableAssignment := stmt.(*ast.VariableAssignment); isVariableAssignment {
				if e.verbose {
					fmt.Printf("DEBUG: executeBlockStatement - skipping output collection from variable assignment\n")
				}
			} else if _, isExpressionAssignment := stmt.(*ast.ExpressionAssignment); isExpressionAssignment {
				if e.verbose {
					fmt.Printf("DEBUG: executeBlockStatement - skipping output collection from expression assignment\n")
				}
			} else if langCallStmt, isLanguageCallStatement := stmt.(*ast.LanguageCallStatement); isLanguageCallStatement && langCallStmt.IsBackground {
				if e.verbose {
					fmt.Printf("DEBUG: executeBlockStatement - skipping output collection from background call\n")
				}
			} else {
				shouldCollectOutput := true

				// Special handling for JavaScript code blocks without variables
				if codeBlockStmt, isCodeBlockStatement := stmt.(*ast.CodeBlockStatement); isCodeBlockStatement {
					if (codeBlockStmt.RuntimeToken.Value == "js" || codeBlockStmt.RuntimeToken.Value == "node") && !codeBlockStmt.HasVariables() && len(block.Statements) > 1 {
						if resultStr, ok := lastResult.(string); ok && strings.TrimSpace(resultStr) == "✓ Executed" {
							// Check if there are non-code-block statements
							hasOtherStatements := false
							for _, s := range block.Statements {
								if _, isOtherCodeBlock := s.(*ast.CodeBlockStatement); !isOtherCodeBlock {
									hasOtherStatements = true
									break
								}
							}
							if hasOtherStatements {
								if e.verbose {
									fmt.Printf("DEBUG: executeBlockStatement - skipping '✓ Executed' from JavaScript code block (part of larger script)\n")
								}
								shouldCollectOutput = false
							}
						}
					}
				}

				if shouldCollectOutput {
					var outputStr string
					if resultStr, ok := lastResult.(string); ok && resultStr != "" {
						outputStr = resultStr
					} else if preFormatted, ok := lastResult.(*shared.PreFormattedResult); ok {
						// For pre-formatted results (like from print function), use the value directly
						outputStr = preFormatted.Value
					} else {
						// Convert other types to string (for function calls that return values)
						outputStr = fmt.Sprintf("%v", lastResult)
					}

					if e.verbose {
						fmt.Printf("DEBUG: executeBlockStatement - collecting output: '%s'\n", outputStr)
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
	}

	// Wait for all background jobs to complete before returning
	if err := e.WaitForAllJobs(); err != nil {
		if e.verbose {
			fmt.Printf("DEBUG: executeBlockStatement - error waiting for background jobs: %v\n", err)
		}
		return nil, err
	}

	// Check if we have any background output to include
	if e.backgroundOutput != "" {
		if e.verbose {
			fmt.Printf("DEBUG: executeBlockStatement - including background output: '%s'\n", e.backgroundOutput)
		}
		if collectedOutput.Len() > 0 {
			collectedOutput.WriteString("\n")
		}
		collectedOutput.WriteString(e.backgroundOutput)
		// Clear background output after consuming it
		e.backgroundOutput = ""
	}

	// If we collected any output from print statements, return that instead of the last result
	if collectedOutput.Len() > 0 {
		finalOutput := collectedOutput.String()
		if e.verbose {
			fmt.Printf("DEBUG: executeBlockStatement - returning collected output before cleaning: '%s'\n", finalOutput)
		}
		// Clean the output to remove unwanted REPL prompts and debug messages
		cleanedOutput := e.cleanREPLOutput(finalOutput)
		if e.verbose {
			fmt.Printf("DEBUG: executeBlockStatement - returning collected output after cleaning: '%s'\n", cleanedOutput)
		}
		return cleanedOutput, nil
	}

	// Check if the last result is a string (might be from a single print statement)
	if lastResultStr, ok := lastResult.(string); ok && lastResultStr != "" {
		if e.verbose {
			fmt.Printf("DEBUG: executeBlockStatement - returning last result as string before cleaning: '%s'\n", lastResultStr)
		}
		// Clean the output to remove unwanted REPL prompts and debug messages
		cleanedOutput := e.cleanREPLOutput(lastResultStr)
		if e.verbose {
			fmt.Printf("DEBUG: executeBlockStatement - returning last result as string after cleaning: '%s'\n", cleanedOutput)
		}
		return cleanedOutput, nil
	}

	if e.verbose {
		fmt.Printf("DEBUG: executeBlockStatement - returning final result: %v\n", lastResult)
	}
	return lastResult, nil
}

// canExecuteAsSingleCodeBlock checks if all statements in the block can be executed as a single code block
func (e *ExecutionEngine) canExecuteAsSingleCodeBlock(block *ast.BlockStatement) bool {
	if e.verbose {
		fmt.Printf("DEBUG: canExecuteAsSingleCodeBlock called with %d statements\n", len(block.Statements))
	}

	if len(block.Statements) == 0 {
		if e.verbose {
			fmt.Printf("DEBUG: canExecuteAsSingleCodeBlock - block is empty, returning false\n")
		}
		return false
	}

	// Check if all statements are Python statements
	for i, stmt := range block.Statements {
		isPython := e.isPythonStatement(stmt)
		if e.verbose {
			fmt.Printf("DEBUG: canExecuteAsSingleCodeBlock - statement %d (type %T) isPython: %v\n", i, stmt, isPython)
		}
		if !isPython {
			if e.verbose {
				fmt.Printf("DEBUG: canExecuteAsSingleCodeBlock - statement %d is not Python, returning false\n", i)
			}
			return false
		}
	}

	if e.verbose {
		fmt.Printf("DEBUG: canExecuteAsSingleCodeBlock - all statements are Python, returning true\n")
	}
	return true
}

// isPythonStatement checks if a statement is a Python statement
func (e *ExecutionEngine) isPythonStatement(stmt ast.Statement) bool {
	switch s := stmt.(type) {
	case *ast.VariableAssignment:
		return s.Variable.Language == "py" || s.Variable.Language == "python"
	case *ast.VariableRead:
		return s.Variable.Language == "py" || s.Variable.Language == "python"
	case *ast.LanguageCall:
		return s.Language == "py" || s.Language == "python"
	case *ast.LanguageCallStatement:
		return s.LanguageCall.Language == "py" || s.LanguageCall.Language == "python"
	case *ast.ExpressionAssignment:
		// For indexed assignments, check if the base object is Python
		if indexExpr, ok := s.Left.(*ast.IndexExpression); ok {
			return e.isPythonExpression(indexExpr.Object)
		}
		return false
	case *ast.IfStatement, *ast.WhileStatement, *ast.ForInLoopStatement, *ast.NumericForLoopStatement, *ast.CStyleForLoopStatement:
		// Control flow statements should never be executed as single code blocks
		// They should always be handled by their own execution methods
		return false
	case *ast.BlockStatement:
		// Nested blocks - check recursively
		return e.canExecuteAsSingleCodeBlock(s)
	default:
		return false
	}
}

// isPythonExpression checks if an expression is a Python expression
func (e *ExecutionEngine) isPythonExpression(expr ast.Expression) bool {
	switch typedExpr := expr.(type) {
	case *ast.Identifier:
		return typedExpr.Language == "py" || typedExpr.Language == "python"
	case *ast.IndexExpression:
		return e.isPythonExpression(typedExpr.Object)
	default:
		return false
	}
}

// executeBlockAsSingleCodeBlock executes all statements in a block as a single Python code block
func (e *ExecutionEngine) executeBlockAsSingleCodeBlock(block *ast.BlockStatement) (interface{}, error) {
	if e.verbose {
		fmt.Printf("DEBUG: executeBlockAsSingleCodeBlock called with %d statements\n", len(block.Statements))
	}

	// Check if this block contains if statements - if so, always execute individually
	// This is because ExecuteCodeBlock doesn't return the values of assignments
	for _, stmt := range block.Statements {
		if _, ok := stmt.(*ast.IfStatement); ok {
			if e.verbose {
				fmt.Printf("DEBUG: Block contains if statement, executing statements individually\n")
			}
			return e.executeBlockStatementFallback(block)
		}
	}

	// Get Python runtime
	rt, err := e.runtimeManager.GetRuntime("python")
	if err != nil {
		if e.verbose {
			fmt.Printf("DEBUG: Python runtime not found, falling back to individual statement execution\n")
		}
		return e.executeBlockStatementFallback(block)
	}

	// Convert all statements to Python code
	pythonCode, err := e.convertBlockToPythonCode(block)
	if err != nil {
		if e.verbose {
			fmt.Printf("DEBUG: Failed to convert block to Python code: %v\n", err)
		}
		return e.executeBlockStatementFallback(block)
	}

	if e.verbose {
		fmt.Printf("DEBUG: Generated Python code: %s\n", pythonCode)
	}

	// Execute the code as a single block
	var result interface{}

	// For Python runtime, use ExecuteCodeBlock if available for better state management
	if pythonRuntime, ok := rt.(*python.PythonRuntime); ok {
		if e.verbose {
			fmt.Printf("DEBUG: Using PythonRuntime ExecuteCodeBlock for block execution\n")
		}
		result, err = pythonRuntime.ExecuteCodeBlock(pythonCode)
	} else {
		// Fallback to Eval for other runtime types
		if e.verbose {
			fmt.Printf("DEBUG: Using rt.Eval for block execution\n")
		}
		result, err = rt.Eval(pythonCode)
	}

	if err != nil {
		if e.verbose {
			fmt.Printf("DEBUG: Failed to execute Python code block: %v\n", err)
		}
		return e.executeBlockStatementFallback(block)
	}

	if e.verbose {
		fmt.Printf("DEBUG: Python code block execution successful, result: %v\n", result)
	}

	// After executing a Python code block, we need to extract any variables that were set
	// and store them in shared variables to maintain consistency with individual statement execution
	if pythonRuntime, ok := rt.(*python.PythonRuntime); ok {
		// For assignment blocks, try to return the value of the last assigned variable
		// Check if the last statement is an assignment and return its value
		if len(block.Statements) > 0 {
			lastStmt := block.Statements[len(block.Statements)-1]
			if e.verbose {
				fmt.Printf("DEBUG: Last statement type: %T\n", lastStmt)
			}

			// Try to get the value of the assigned variable
			switch stmt := lastStmt.(type) {
			case *ast.VariableAssignment:
				if stmt.Variable.Qualified {
					varValue, err := e.readVariableFromRuntime(rt, stmt.Variable.Language, stmt.Variable.Name)
					if err == nil && varValue != nil {
						if e.verbose {
							fmt.Printf("DEBUG: Returning value from VariableAssignment: %v\n", varValue)
						}
						result = varValue
					}
				}
			case *ast.ExpressionAssignment:
				// For expression assignments, extract the base variable name from the left side
				// and return its value
				baseVarName := e.extractBaseVariableFromExpression(stmt.Left)
				if baseVarName != "" {
					varValue, err := e.readVariableFromRuntime(rt, "python", baseVarName)
					if err == nil && varValue != nil {
						if e.verbose {
							fmt.Printf("DEBUG: Returning value from ExpressionAssignment (%s variable): %v\n", baseVarName, varValue)
						}
						result = varValue
					}
				}
			}
		}
		// Use a special Python function to capture all variables and return them as a dictionary
		captureCode := `
def __capture_variables():
	   import sys
	   __result = {}
	   # Get all global variables
	   globals_dict = globals()
	   # Filter out system variables and modules
	   for name, value in globals_dict.items():
	       if not name.startswith('__'):
	           try:
	               # Try to serialize the value to make sure it can be transferred
	               str(value)  # Simple test
	               __result[name] = value
	           except:
	               # Skip variables that can't be serialized
	               pass
	   return __result

__capture_variables()
`
		if e.verbose {
			fmt.Printf("DEBUG: Capturing variables from Python runtime\n")
		}

		capturedVars, err := pythonRuntime.Eval(captureCode)
		if err == nil {
			// Update shared variables with captured values
			if varsDict, ok := capturedVars.(map[string]interface{}); ok {
				for varName, varValue := range varsDict {
					e.SetSharedVariable("python", varName, varValue)
					if e.verbose {
						fmt.Printf("DEBUG: Captured variable python.%s with value: %v\n", varName, varValue)
					}
				}
			}
		} else if e.verbose {
			fmt.Printf("DEBUG: Failed to capture variables: %v\n", err)
		}
	}

	return result, nil
}

// convertBlockToPythonCode converts a block of statements to Python code
func (e *ExecutionEngine) convertBlockToPythonCode(block *ast.BlockStatement) (string, error) {
	if e.verbose {
		fmt.Printf("DEBUG: convertBlockToPythonCode called with %d statements\n", len(block.Statements))
		for i, stmt := range block.Statements {
			fmt.Printf("DEBUG: Statement %d: type %T\n", i, stmt)
		}
	}

	var codeBuilder strings.Builder

	for _, stmt := range block.Statements {
		pythonCode, err := e.convertStatementToPythonCode(stmt)
		if err != nil {
			if e.verbose {
				fmt.Printf("DEBUG: Failed to convert statement of type %T to Python code: %v\n", stmt, err)
			}
			return "", err
		}
		if pythonCode != "" {
			if e.verbose {
				fmt.Printf("DEBUG: Converted statement of type %T to Python code: '%s'\n", stmt, pythonCode)
			}
			codeBuilder.WriteString(pythonCode)
			codeBuilder.WriteString("\n")
		} else {
			if e.verbose {
				fmt.Printf("DEBUG: Statement of type %T resulted in empty Python code\n", stmt)
			}
		}
	}

	result := strings.TrimSpace(codeBuilder.String())
	if e.verbose {
		fmt.Printf("DEBUG: Final Python code for block: '%s'\n", result)
	}
	return result, nil
}

// convertStatementToPythonCode converts a single statement to Python code
func (e *ExecutionEngine) convertStatementToPythonCode(stmt ast.Statement) (string, error) {
	switch s := stmt.(type) {
	case *ast.VariableAssignment:
		return e.convertVariableAssignmentToPython(s)
	case *ast.LanguageCall:
		return e.convertLanguageCallToPython(s)
	case *ast.ExpressionAssignment:
		return e.convertExpressionAssignmentToPython(s)
	case *ast.IfStatement:
		return e.convertIfStatementToPython(s)
	default:
		return "", fmt.Errorf("unsupported statement type for Python conversion: %T", stmt)
	}
}

// convertVariableAssignmentToPython converts a variable assignment to Python code
func (e *ExecutionEngine) convertVariableAssignmentToPython(assign *ast.VariableAssignment) (string, error) {
	// Convert the right-hand expression to Python code instead of evaluating it
	// This avoids issues with variables that don't exist yet during conversion
	valueStr, err := e.convertExpressionToPythonCode(assign.Value)
	if err != nil {
		return "", err
	}

	// For Python code generation, use only the variable name without language prefix
	// since the code will be executed directly in Python runtime
	variableName := assign.Variable.Name

	return fmt.Sprintf("%s = %s", variableName, valueStr), nil
}

// convertLanguageCallToPython converts a language call to Python code
func (e *ExecutionEngine) convertLanguageCallToPython(call *ast.LanguageCall) (string, error) {
	// Convert arguments to Python code instead of evaluating them
	// This avoids issues with variables that don't exist yet during conversion
	var args []string
	for _, arg := range call.Arguments {
		argStr, err := e.convertExpressionToPythonCode(arg)
		if err != nil {
			return "", err
		}
		args = append(args, argStr)
	}

	argsStr := strings.Join(args, ", ")
	return fmt.Sprintf("%s(%s)", call.Function, argsStr), nil
}

// convertExpressionAssignmentToPython converts an expression assignment to Python code
func (e *ExecutionEngine) convertExpressionAssignmentToPython(assign *ast.ExpressionAssignment) (string, error) {
	if e.verbose {
		fmt.Printf("DEBUG: convertExpressionAssignmentToPython called with assign.Left type: %T\n", assign.Left)
	}

	// For now, only handle simple indexed assignments like dict["key"] = value
	if indexExpr, ok := assign.Left.(*ast.IndexExpression); ok {
		if e.verbose {
			fmt.Printf("DEBUG: convertExpressionAssignmentToPython - found IndexExpression\n")
			fmt.Printf("DEBUG: convertExpressionAssignmentToPython - indexExpr.Object type: %T\n", indexExpr.Object)
			fmt.Printf("DEBUG: convertExpressionAssignmentToPython - indexExpr.Index type: %T\n", indexExpr.Index)
		}

		// Convert the object and index
		objStr, err := e.convertExpressionToPythonCode(indexExpr.Object)
		if err != nil {
			return "", err
		}

		indexStr, err := e.convertExpressionToPythonCode(indexExpr.Index)
		if err != nil {
			return "", err
		}

		if e.verbose {
			fmt.Printf("DEBUG: convertExpressionAssignmentToPython - objStr: '%s', indexStr: '%s'\n", objStr, indexStr)
		}

		// Convert the right-hand expression to Python code instead of evaluating it
		// This avoids issues with variables that don't exist yet during conversion
		valueStr, err := e.convertExpressionToPythonCode(assign.Value)
		if err != nil {
			return "", err
		}

		// Check if this is an array assignment that might need expansion
		// Case 1: Nested index expression like data["users"][0] = ...
		if _, ok := indexExpr.Object.(*ast.IndexExpression); ok {
			if e.verbose {
				fmt.Printf("DEBUG: convertExpressionAssignmentToPython - detected nested index expression\n")
			}
			if numLit, ok := indexExpr.Index.(*ast.NumberLiteral); ok && numLit.IsInt {
				if e.verbose {
					fmt.Printf("DEBUG: convertExpressionAssignmentToPython - detected numeric index in nested expression, generating array expansion code\n")
				}
				// This looks like an array assignment to a specific index
				// Generate code that handles array expansion
				return fmt.Sprintf("if len(%s) <= %s: %s.append(%s)\nelse: %s[%s] = %s",
					objStr, indexStr, objStr, valueStr, objStr, indexStr, valueStr), nil
			}
		}

		// Case 2: Simple index expression with numeric index like data["users"][0] = ...
		// where the object itself is an index expression accessing a list
		if numLit, ok := indexExpr.Index.(*ast.NumberLiteral); ok && numLit.IsInt {
			if e.verbose {
				fmt.Printf("DEBUG: convertExpressionAssignmentToPython - detected numeric index in simple expression, checking if object is index expression\n")
			}
			// Check if the object string contains brackets (like data["users"])
			// This indicates that the object is itself an index expression
			if strings.Contains(objStr, "[") && strings.Contains(objStr, "]") {
				if e.verbose {
					fmt.Printf("DEBUG: convertExpressionAssignmentToPython - object string contains brackets, generating array expansion code\n")
				}
				// This looks like data["users"][0] = ... where data["users"] might be a list
				// Generate code that handles array expansion
				return fmt.Sprintf("if len(%s) <= %s: %s.append(%s)\nelse: %s[%s] = %s",
					objStr, indexStr, objStr, valueStr, objStr, indexStr, valueStr), nil
			}
		}

		return fmt.Sprintf("%s[%s] = %s", objStr, indexStr, valueStr), nil
	}

	return "", fmt.Errorf("unsupported expression assignment type: %T", assign.Left)
}

// convertIfStatementToPython converts an if statement to Python code
func (e *ExecutionEngine) convertIfStatementToPython(ifStmt *ast.IfStatement) (string, error) {
	// Convert condition
	conditionStr, err := e.convertExpressionToPythonCode(ifStmt.Condition)
	if err != nil {
		return "", err
	}

	// Convert consequent block
	consequentStr, err := e.convertBlockToPythonCode(ifStmt.Consequent)
	if err != nil {
		if e.verbose {
			fmt.Printf("DEBUG: Failed to convert consequent block: %v\n", err)
		}
		return "", err
	}

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("if %s:\n", conditionStr))
	if consequentStr != "" {
		builder.WriteString(e.indentCode(consequentStr))
	} else {
		builder.WriteString("    pass\n")
	}

	// Convert else block if present
	if ifStmt.HasElse() {
		elseStr, err := e.convertBlockToPythonCode(ifStmt.Alternate)
		if err != nil {
			if e.verbose {
				fmt.Printf("DEBUG: Failed to convert else block: %v\n", err)
			}
			return "", err
		}
		builder.WriteString("else:\n")
		if elseStr != "" {
			builder.WriteString(e.indentCode(elseStr))
		} else {
			builder.WriteString("    pass\n")
		}
	}

	result := builder.String()
	if e.verbose {
		fmt.Printf("DEBUG: Generated IfStatement Python code length: %d\n", len(result))
		fmt.Printf("DEBUG: Generated IfStatement Python code (with quotes): '%s'\n", result)
		fmt.Printf("DEBUG: Generated IfStatement Python code (with brackets): [%s]\n", result)
		fmt.Printf("DEBUG: Generated IfStatement Python code (hex): %x\n", result)
	}
	return result, nil
}

// convertExpressionToPythonCode converts an expression to Python code
func (e *ExecutionEngine) convertExpressionToPythonCode(expr ast.Expression) (string, error) {
	switch ex := expr.(type) {
	case *ast.Identifier:
		// For Python code generation, use only the variable name without language prefix
		// since the code will be executed directly in Python runtime
		if ex.Qualified && len(ex.Path) > 0 {
			// For qualified identifiers with path, construct the full path using bracket notation
			// e.g., py.data.users becomes data["users"] for dictionary access
			var fullPath strings.Builder
			// The first part is the base variable name
			if len(ex.Path) > 0 {
				fullPath.WriteString(ex.Path[0])
				// Subsequent parts use bracket notation for dictionary access
				for i := 1; i < len(ex.Path); i++ {
					fullPath.WriteString(fmt.Sprintf("[\"%s\"]", ex.Path[i]))
				}
				// The final name also uses bracket notation
				fullPath.WriteString(fmt.Sprintf("[\"%s\"]", ex.Name))
			}
			return fullPath.String(), nil
		}
		return ex.Name, nil
	case *ast.StringLiteral:
		return fmt.Sprintf("\"%s\"", strings.ReplaceAll(ex.Value, "\"", "\\\"")), nil
	case *ast.NumberLiteral:
		if ex.IsInt {
			return ex.IntValue.String(), nil
		} else {
			return fmt.Sprintf("%v", ex.FloatValue), nil
		}
	case *ast.BooleanLiteral:
		return fmt.Sprintf("%t", ex.Value), nil
	case *ast.ObjectLiteral:
		return e.convertObjectLiteralToPython(ex)
	case *ast.ArrayLiteral:
		return e.convertArrayLiteralToPython(ex)
	case *ast.BinaryExpression:
		return e.convertBinaryExpressionToPython(ex)
	case *ast.IndexExpression:
		return e.convertIndexExpressionToPython(ex)
	case *ast.FieldAccess:
		return e.convertFieldAccessToPython(ex)
	default:
		return "", fmt.Errorf("unsupported expression type for Python conversion: %T", expr)
	}
}

// convertBinaryExpressionToPython converts a binary expression to Python code
func (e *ExecutionEngine) convertBinaryExpressionToPython(expr *ast.BinaryExpression) (string, error) {
	leftStr, err := e.convertExpressionToPythonCode(expr.Left)
	if err != nil {
		return "", err
	}

	rightStr, err := e.convertExpressionToPythonCode(expr.Right)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("(%s %s %s)", leftStr, expr.Operator, rightStr), nil
}

// convertIndexExpressionToPython converts an index expression to Python code
func (e *ExecutionEngine) convertIndexExpressionToPython(expr *ast.IndexExpression) (string, error) {
	objStr, err := e.convertExpressionToPythonCode(expr.Object)
	if err != nil {
		return "", err
	}

	indexStr, err := e.convertExpressionToPythonCode(expr.Index)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s[%s]", objStr, indexStr), nil
}

// convertFieldAccessToPython converts a field access expression to Python code
func (e *ExecutionEngine) convertFieldAccessToPython(expr *ast.FieldAccess) (string, error) {
	// Check if this is a qualified identifier like py.x that should be converted to just x
	if ident, ok := expr.Object.(*ast.Identifier); ok {
		if ident.Language == "py" || ident.Language == "python" {
			// For Python code generation, py.x becomes just x since we're in Python runtime
			return expr.Field, nil
		}
	}

	// For other field accesses, convert normally
	objStr, err := e.convertExpressionToPythonCode(expr.Object)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s.%s", objStr, expr.Field), nil
}

// convertObjectLiteralToPython converts an object literal to Python code
func (e *ExecutionEngine) convertObjectLiteralToPython(obj *ast.ObjectLiteral) (string, error) {
	var builder strings.Builder
	builder.WriteString("{")

	for i, prop := range obj.Properties {
		// Convert key
		keyStr, err := e.convertExpressionToPythonCode(prop.Key)
		if err != nil {
			return "", err
		}

		// Convert value
		valueStr, err := e.convertExpressionToPythonCode(prop.Value)
		if err != nil {
			return "", err
		}

		// For string keys, Python allows both quoted and unquoted format
		// We'll use the format that matches the original
		if strLit, ok := prop.Key.(*ast.StringLiteral); ok {
			builder.WriteString(fmt.Sprintf("\"%s\": %s", strLit.Value, valueStr))
		} else {
			builder.WriteString(fmt.Sprintf("%s: %s", keyStr, valueStr))
		}

		if i < len(obj.Properties)-1 {
			builder.WriteString(", ")
		}
	}

	builder.WriteString("}")
	return builder.String(), nil
}

// convertArrayLiteralToPython converts an array literal to Python code
func (e *ExecutionEngine) convertArrayLiteralToPython(arr *ast.ArrayLiteral) (string, error) {
	var builder strings.Builder
	builder.WriteString("[")

	for i, elem := range arr.Elements {
		elemStr, err := e.convertExpressionToPythonCode(elem)
		if err != nil {
			return "", err
		}

		builder.WriteString(elemStr)

		if i < len(arr.Elements)-1 {
			builder.WriteString(", ")
		}
	}

	builder.WriteString("]")
	return builder.String(), nil
}

// convertValueToPythonLiteral converts a Go value to Python literal representation
func (e *ExecutionEngine) convertValueToPythonLiteral(value interface{}) string {
	switch v := value.(type) {
	case string:
		return fmt.Sprintf("\"%s\"", strings.ReplaceAll(v, "\"", "\\\""))
	case int, int64:
		return fmt.Sprintf("%d", v)
	case float64:
		return fmt.Sprintf("%v", v)
	case bool:
		return fmt.Sprintf("%t", v)
	case nil:
		return "None"
	default:
		// For other types, try to convert to string
		return fmt.Sprintf("\"%v\"", v)
	}
}

// indentCode adds 4 spaces of indentation to each line of code
func (e *ExecutionEngine) indentCode(code string) string {
	lines := strings.Split(code, "\n")
	for i, line := range lines {
		if line != "" {
			lines[i] = "    " + line
		}
	}
	return strings.Join(lines, "\n")
}

// executeBlockStatementFallback executes statements one by one (original behavior)
func (e *ExecutionEngine) executeBlockStatementFallback(block *ast.BlockStatement) (interface{}, error) {
	if e.verbose {
		fmt.Printf("DEBUG: Using fallback execution for block with %d statements\n", len(block.Statements))
	}

	var lastResult interface{}
	var err error

	for _, stmt := range block.Statements {
		lastResult, err = e.executeStatement(stmt)
		if err != nil {
			return nil, err
		}
	}

	return lastResult, nil
}

// cleanREPLOutput removes unwanted REPL prompts and debug messages from output
func (e *ExecutionEngine) cleanREPLOutput(output string) string {
	if e.verbose {
		fmt.Printf("DEBUG: cleanREPLOutput input: '%s'\n", output)
	}

	// Split output into lines
	lines := strings.Split(output, "\n")
	var cleanedLines []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Skip empty lines
		if trimmed == "" {
			continue
		}
		// Skip undefined outputs (including the | | undefined format)
		if trimmed == "undefined" || strings.Contains(trimmed, "undefined") {
			continue
		}
		// Skip function declarations
		if strings.HasPrefix(trimmed, "[Function:") {
			continue
		}
		// Skip success messages
		if trimmed == "Functions imported successfully" {
			continue
		}
		// Replace tabs with spaces for consistent output formatting
		line = strings.ReplaceAll(line, "\t", " ")
		// Keep the line
		cleanedLines = append(cleanedLines, line)
	}

	result := strings.Join(cleanedLines, "\n")
	if e.verbose {
		fmt.Printf("DEBUG: cleanREPLOutput output: '%s'\n", result)
	}
	return result
}

// findBaseIdentifier traverses an index expression to find the base identifier
func (e *ExecutionEngine) findBaseIdentifier(expr *ast.IndexExpression) *ast.Identifier {
	// Recursively traverse nested index expressions to find the base identifier
	current := expr
	for {
		if ident, ok := current.Object.(*ast.Identifier); ok {
			return ident
		}
		if nestedIndex, ok := current.Object.(*ast.IndexExpression); ok {
			current = nestedIndex
		} else {
			break
		}
	}
	return nil
}

// extractBaseVariableFromExpression extracts the base variable name from an expression
// For example, from py.data.users[0] it extracts "data"
// from py.data.users[0].age it extracts "data"
func (e *ExecutionEngine) extractBaseVariableFromExpression(expr ast.Expression) string {
	if e.verbose {
		fmt.Printf("DEBUG: extractBaseVariableFromExpression called with type %T\n", expr)
	}

	switch ex := expr.(type) {
	case *ast.IndexExpression:
		// For index expressions, find the base identifier
		baseIdent := e.findBaseIdentifier(ex)
		if baseIdent != nil {
			if e.verbose {
				fmt.Printf("DEBUG: extractBaseVariableFromExpression - found base identifier: %s, path: %v\n", baseIdent.Name, baseIdent.Path)
			}
			// For qualified identifiers, return the first part of the path
			// For py.data.users, Path is [data], so we return "data"
			if baseIdent.Qualified && len(baseIdent.Path) > 0 {
				return baseIdent.Path[0]
			}
			// For simple identifiers like x, return the name
			return baseIdent.Name
		}
	case *ast.Identifier:
		// For simple identifiers, return the name
		if e.verbose {
			fmt.Printf("DEBUG: extractBaseVariableFromExpression - found simple identifier: %s, path: %v\n", ex.Name, ex.Path)
		}
		// For qualified identifiers, return the first part of the path
		// For py.data.users, Path is [data], so we return "data"
		if ex.Qualified && len(ex.Path) > 0 {
			return ex.Path[0]
		}
		// For simple identifiers like x, return the name
		return ex.Name
	}

	if e.verbose {
		fmt.Printf("DEBUG: extractBaseVariableFromExpression - could not extract base variable\n")
	}
	return ""
}
