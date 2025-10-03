package engine

import (
	stderrors "errors"
	"fmt"
	"strings"

	"funterm/runtime/python"
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
			// For other errors: if an error occurs, stop execution silently
			// without propagating the error. This matches the expected behavior
			// where exceptions should stop execution but not be reported as errors.
			return nil, nil
		}

		// If this statement produced output (from print statements, function calls, or nested blocks), collect it
		if lastResult != nil {
			// Don't collect output from variable assignments, bitstring pattern assignments, or background calls
			if _, isVariableAssignment := stmt.(*ast.VariableAssignment); isVariableAssignment {
				if e.verbose {
					fmt.Printf("DEBUG: executeBlockStatement - skipping output collection from variable assignment\n")
				}
			} else if _, isBitstringPatternAssignment := stmt.(*ast.BitstringPatternAssignment); isBitstringPatternAssignment {
				if e.verbose {
					fmt.Printf("DEBUG: executeBlockStatement - skipping output collection from bitstring pattern assignment\n")
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
	case *ast.IfStatement, *ast.WhileStatement, *ast.ForInLoopStatement, *ast.NumericForLoopStatement:
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
		// Use a special Python function to capture all variables and return them as a dictionary
		captureCode := `
def __capture_variables():
	   import sys
	   result = {}
	   # Get all global variables
	   globals_dict = globals()
	   # Filter out system variables and modules
	   for name, value in globals_dict.items():
	       if not name.startswith('__') and not name.startswith('_') and name != 'result':
	           try:
	               # Try to serialize the value to make sure it can be transferred
	               str(value)  # Simple test
	               result[name] = value
	           except:
	               # Skip variables that can't be serialized
	               pass
	   return result

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
	value, err := e.convertExpressionToValue(assign.Value)
	if err != nil {
		return "", err
	}

	// Convert value to Python literal
	valueStr := e.convertValueToPythonLiteral(value)

	// For Python code generation, use only the variable name without language prefix
	// since the code will be executed directly in Python runtime
	variableName := assign.Variable.Name

	return fmt.Sprintf("%s = %s", variableName, valueStr), nil
}

// convertLanguageCallToPython converts a language call to Python code
func (e *ExecutionEngine) convertLanguageCallToPython(call *ast.LanguageCall) (string, error) {
	// Convert arguments to Python literals
	var args []string
	for _, arg := range call.Arguments {
		value, err := e.convertExpressionToValue(arg)
		if err != nil {
			return "", err
		}
		args = append(args, e.convertValueToPythonLiteral(value))
	}

	argsStr := strings.Join(args, ", ")
	return fmt.Sprintf("%s(%s)", call.Function, argsStr), nil
}

// convertExpressionAssignmentToPython converts an expression assignment to Python code
func (e *ExecutionEngine) convertExpressionAssignmentToPython(assign *ast.ExpressionAssignment) (string, error) {
	// For now, only handle simple indexed assignments like dict["key"] = value
	if indexExpr, ok := assign.Left.(*ast.IndexExpression); ok {
		// Convert the object and index
		objStr, err := e.convertExpressionToPythonCode(indexExpr.Object)
		if err != nil {
			return "", err
		}

		indexStr, err := e.convertExpressionToPythonCode(indexExpr.Index)
		if err != nil {
			return "", err
		}

		value, err := e.convertExpressionToValue(assign.Value)
		if err != nil {
			return "", err
		}

		valueStr := e.convertValueToPythonLiteral(value)
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
		return ex.Name, nil
	case *ast.StringLiteral:
		return fmt.Sprintf("\"%s\"", strings.ReplaceAll(ex.Value, "\"", "\\\"")), nil
	case *ast.NumberLiteral:
		return fmt.Sprintf("%v", ex.Value), nil
	case *ast.BooleanLiteral:
		return fmt.Sprintf("%t", ex.Value), nil
	case *ast.BinaryExpression:
		return e.convertBinaryExpressionToPython(ex)
	case *ast.IndexExpression:
		return e.convertIndexExpressionToPython(ex)
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
