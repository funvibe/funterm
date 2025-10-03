package engine

import (
	"fmt"
	"os"
	"strings"
	"time"

	"funterm/container"
	"funterm/errors"
	"funterm/jobmanager"
	"funterm/runtime/lua"
	"funterm/runtime/node"
	"funterm/runtime/python"
	"funterm/shared"
	"go-parser/pkg/ast"
	"go-parser/pkg/parser"

	"github.com/funvibe/funbit/pkg/funbit"
)

// Execute parses and executes a command string, returning result, isPrint flag, and error
func (e *ExecutionEngine) Execute(command string) (interface{}, bool, error) {
	if e.verbose {
		fmt.Printf("DEBUG: Executing command: '%s'\n", command)
	}

	// Parse the command
	statement, parseErrors := e.parser.Parse(command)
	if len(parseErrors) > 0 {
		if e.verbose {
			fmt.Printf("DEBUG: Parser errors: %v\n", parseErrors)
		}
		// For now, just return the first parsing error
		return nil, false, errors.NewUserError("PARSING_ERROR", parseErrors[0].Message)
	}

	if statement == nil {
		if e.verbose {
			fmt.Printf("DEBUG: Statement is nil\n")
		}
		return nil, false, errors.NewUserError("UNSUPPORTED_COMMAND", "unsupported command")
	}

	if e.verbose {
		fmt.Printf("DEBUG: Statement type: %T\n", statement)
	}

	// Determine if the statement is a print function before execution
	isPrint := false
	if langCallStmt, ok := statement.(*ast.LanguageCallStatement); ok {
		if e.isPrintFunction(langCallStmt.LanguageCall) {
			isPrint = true
		}
	}

	// Execute the statement and collect output
	result, err := e.executeStatement(statement)
	if err != nil {
		return nil, isPrint, err
	}

	// Handle bitstring output formatting
	if byteResult, ok := result.([]byte); ok {
		// Format bitstring output in Erlang style
		formattedOutput := e.formatBitstringOutput(byteResult)
		if e.verbose {
			fmt.Printf("DEBUG: Execute - returning formatted bitstring output: '%s'\n", formattedOutput)
		}
		return formattedOutput, isPrint, nil
	}

	// If this is a print statement that produced string output, return it
	if resultStr, ok := result.(string); ok {
		if e.verbose {
			fmt.Printf("DEBUG: Execute - returning string output: '%s'\n", resultStr)
		}
		return resultStr, isPrint, nil
	}

	// For other cases, return the result as is
	if e.verbose {
		fmt.Printf("DEBUG: Execute - returning result: %v\n", result)
	}
	return result, isPrint, nil
}

// ExecuteCompat parses and executes a command string, returning only result and error for backward compatibility
func (e *ExecutionEngine) ExecuteCompat(command string) (interface{}, error) {
	result, _, err := e.Execute(command)
	return result, err
}

// GetContainer returns the DI container used by this engine
func (e *ExecutionEngine) GetContainer() container.Container {
	return e.container
}

// GetParser returns the parser
func (e *ExecutionEngine) GetParser() *parser.UnifiedParser {
	return e.parser
}

// SetSharedVariable устанавливает переменную в общем хранилище
func (e *ExecutionEngine) SetSharedVariable(language, name string, value interface{}) {
	e.variablesMutex.Lock()
	defer e.variablesMutex.Unlock()

	// Инициализируем карту для языка, если она еще не существует
	if _, exists := e.sharedVariables[language]; !exists {
		e.sharedVariables[language] = make(map[string]interface{})
		if e.verbose {
			fmt.Printf("DEBUG: SetSharedVariable - Created new variable map for language=%s\n", language)
		}
	}

	// Если значение nil, удаляем переменную
	if value == nil {
		delete(e.sharedVariables[language], name)
		if e.verbose {
			fmt.Printf("DEBUG: SetSharedVariable - DELETED variable language=%s, name=%s\n", language, name)
		}
		return
	}

	// Устанавливаем значение
	e.sharedVariables[language][name] = value
	if e.verbose {
		fmt.Printf("DEBUG: SetSharedVariable - SET language=%s, name=%s, value=%v\n", language, name, value)
	}
}

// GetSharedVariable получает переменную из общего хранилища
func (e *ExecutionEngine) GetSharedVariable(language, name string) (interface{}, bool) {
	e.variablesMutex.RLock()
	defer e.variablesMutex.RUnlock()

	// Проверяем, существует ли карта для языка
	if languageVars, exists := e.sharedVariables[language]; exists {
		// Проверяем, существует ли переменная
		if value, varExists := languageVars[name]; varExists {
			if e.verbose {
				fmt.Printf("DEBUG: GetSharedVariable - FOUND language=%s, name=%s, value=%v\n", language, name, value)
			}
			return value, true
		}
	}

	if e.verbose {
		fmt.Printf("DEBUG: GetSharedVariable - NOT FOUND language=%s, name=%s\n", language, name)
	}
	return nil, false
}

// executeStatement is a helper method to execute any statement
func (e *ExecutionEngine) executeStatement(stmt ast.Statement) (interface{}, error) {
	switch s := stmt.(type) {
	case *ast.LanguageCall:
		// For LanguageCall, we need to wrap it in a LanguageCallStatement to handle print functions properly
		languageCallStmt := &ast.LanguageCallStatement{
			LanguageCall: s,
			IsBackground: false,
		}
		return e.executeLanguageCallStatement(languageCallStmt)
	case *ast.VariableRead:
		return e.executeVariableRead(s)
	case *ast.VariableAssignment:
		return e.executeVariableAssignment(s)
	case *ast.BitstringPatternAssignment:
		return e.executeBitstringPatternAssignment(s)
	case *ast.IfStatement:
		return e.executeIfStatement(s)
	case *ast.WhileStatement:
		return e.executeWhileStatement(s)
	case *ast.BreakStatement:
		return e.executeBreakStatement(s)
	case *ast.ContinueStatement:
		return e.executeContinueStatement(s)
	case *ast.ForInLoopStatement:
		return e.executeForInLoop(s)
	case *ast.NumericForLoopStatement:
		return e.executeNumericForLoop(s)
	case *ast.PipeExpression:
		return e.executePipeExpression(s)
	case *ast.LanguageCallStatement:
		return e.executeLanguageCallStatement(s)
	case *ast.MatchStatement:
		return e.executeMatchStatement(s)
	case *ast.BlockStatement:
		return e.executeBlockStatement(s)
	case *ast.CodeBlockStatement:
		return e.executeCodeBlockStatement(s)
	case *ast.ImportStatement:
		return e.executeImportStatement(s)
	case *ast.BitstringExpression:
		return e.executeBitstringExpression(s)
	case *ast.IndexExpression:
		return e.convertExpressionToValue(s)
	default:
		return nil, errors.NewUserError("UNSUPPORTED_STATEMENT", fmt.Sprintf("unsupported statement type: %T", stmt))
	}
}

// executeLanguageCallStatement executes a language call statement (with optional background execution)
func (e *ExecutionEngine) executeLanguageCallStatement(stmt *ast.LanguageCallStatement) (interface{}, error) {
	if stmt.IsBackground {
		// This is a background task, delegate to JobManager
		return e.executeBackgroundLanguageCall(stmt)
	} else {
		// For all language calls, get the runtime first (needed for output capture)
		rt, err := e.getRuntimeByName(stmt.LanguageCall.Language)
		if err != nil {
			return nil, err
		}

		// Execute the function normally
		result, err := e.executeLanguageCallNew(stmt.LanguageCall)
		if err != nil {
			return nil, err
		}

		// For Lua runtime, always check for captured output (not just for print functions)
		if stmt.LanguageCall.Language == "lua" {
			if luaRuntime, ok := rt.(*lua.LuaRuntime); ok {
				capturedOutput := luaRuntime.GetCapturedOutput()
				if e.verbose {
					fmt.Printf("DEBUG: Captured output from Lua runtime: '%s'\n", capturedOutput)
				}
				// Return the captured output if it exists, otherwise return the result
				if capturedOutput != "" {
					return capturedOutput, nil
				}
			}
		}

		// For Python runtime, only use captured output for print functions
		if stmt.LanguageCall.Language == "python" || stmt.LanguageCall.Language == "py" {
			if pythonRuntime, ok := rt.(*python.PythonRuntime); ok {
				capturedOutput := pythonRuntime.GetCapturedOutput()
				if e.verbose {
					fmt.Printf("DEBUG: Captured output from Python runtime: '%s'\n", capturedOutput)
				}
				// Only return captured output for print functions, not for regular functions
				if capturedOutput != "" && e.isPrintFunction(stmt.LanguageCall) {
					return capturedOutput, nil
				}
			}
		}

		// For JavaScript/Node runtime, always check for captured output (not just for console.log functions)
		if stmt.LanguageCall.Language == "node" || stmt.LanguageCall.Language == "js" {
			if nodeRuntime, ok := rt.(*node.NodeRuntime); ok {
				capturedOutput := nodeRuntime.GetCapturedOutput()
				if e.verbose {
					fmt.Printf("DEBUG: Captured output from Node runtime: '%s'\n", capturedOutput)
				}
				// Return the captured output if it exists, otherwise return the result
				if capturedOutput != "" {
					return capturedOutput, nil
				}
			}
		}

		// Special handling for print functions - they should return their printed value
		if e.isPrintFunction(stmt.LanguageCall) {
			if e.verbose {
				fmt.Printf("DEBUG: Print function executed, result: %v\n", result)
			}
			// Return the result from runtime execution for print functions
			return result, nil
		} else {
			// This is a regular foreground task, return the result
			return result, nil
		}
	}
}

// isPrintFunction checks if a language call is a print function
func (e *ExecutionEngine) isPrintFunction(call *ast.LanguageCall) bool {
	switch call.Language {
	case "lua":
		return call.Function == "print"
	case "go":
		return call.Function == "println" || call.Function == "printf"
	case "python", "py":
		return call.Function == "print"
	case "node", "js":
		return call.Function == "console.log"
	default:
		return false
	}
}

// executeBackgroundLanguageCall executes a language call as a background task
func (e *ExecutionEngine) executeBackgroundLanguageCall(stmt *ast.LanguageCallStatement) (interface{}, error) {
	// Create a command string for the job
	command := fmt.Sprintf("%s.%s(%s)", stmt.LanguageCall.Language, stmt.LanguageCall.Function, e.formatArgumentsForCommand(stmt.LanguageCall.Arguments))

	// Submit the job to the JobManager
	jobID, err := e.jobManager.Submit(func() (interface{}, error) {
		// This function will be executed in the background
		result, err := e.executeLanguageCallNew(stmt.LanguageCall)
		if err != nil {
			return nil, err
		}

		// Get the runtime to capture output
		rt, err := e.runtimeManager.GetRuntime(stmt.LanguageCall.Language)
		if err != nil {
			return result, nil
		}

		// For Lua runtime, check for captured output
		if stmt.LanguageCall.Language == "lua" {
			if luaRuntime, ok := rt.(*lua.LuaRuntime); ok {
				capturedOutput := luaRuntime.GetCapturedOutput()
				if e.verbose {
					fmt.Printf("DEBUG: Background job captured Lua output: '%s'\n", capturedOutput)
				}
				// Return the captured output if it exists, otherwise return the result
				if capturedOutput != "" {
					return capturedOutput, nil
				}
			}
		}

		// For Python runtime, don't capture output for background tasks to avoid interference
		// Background tasks should return their output through the normal result mechanism
		if stmt.LanguageCall.Language == "python" || stmt.LanguageCall.Language == "py" {
			if e.verbose {
				fmt.Printf("DEBUG: Background job - not capturing Python output to avoid interference\n")
			}
		}

		// For JavaScript/Node runtime, check for captured output
		if stmt.LanguageCall.Language == "node" || stmt.LanguageCall.Language == "js" {
			if nodeRuntime, ok := rt.(*node.NodeRuntime); ok {
				capturedOutput := nodeRuntime.GetCapturedOutput()
				if e.verbose {
					fmt.Printf("DEBUG: Background job captured Node output: '%s'\n", capturedOutput)
				}
				// Return the captured output if it exists, otherwise return the result
				if capturedOutput != "" {
					return capturedOutput, nil
				}
			}
		}

		return result, nil
	}, command)

	if err != nil {
		return nil, errors.NewSystemError("JOB_SUBMISSION_FAILED", fmt.Sprintf("failed to submit background job: %v", err))
	}

	// Return the job ID immediately to the caller
	return jobID, nil
}

// formatArgumentsForCommand formats arguments for display in command strings
func (e *ExecutionEngine) formatArgumentsForCommand(args []ast.Expression) string {
	if len(args) == 0 {
		return ""
	}

	var argStrings []string
	for _, arg := range args {
		switch a := arg.(type) {
		case *ast.StringLiteral:
			argStrings = append(argStrings, fmt.Sprintf("\"%s\"", a.Value))
		case *ast.NumberLiteral:
			argStrings = append(argStrings, fmt.Sprintf("%v", a.Value))
		case *ast.BooleanLiteral:
			argStrings = append(argStrings, fmt.Sprintf("%v", a.Value))
		case *ast.Identifier:
			if a.Qualified {
				argStrings = append(argStrings, fmt.Sprintf("%s.%s", a.Language, a.Name))
			} else {
				argStrings = append(argStrings, a.Name)
			}
		default:
			argStrings = append(argStrings, "...")
		}
	}

	return strings.Join(argStrings, ", ")
}

// GetJobManager returns the JobManager instance
func (e *ExecutionEngine) GetJobManager() *jobmanager.JobManager {
	return e.jobManager
}

// GetJobStatus returns the status of a specific job
func (e *ExecutionEngine) GetJobStatus(id jobmanager.JobID) (jobmanager.JobStatus, error) {
	return e.jobManager.GetJobStatus(id)
}

// ListJobs returns all jobs
func (e *ExecutionEngine) ListJobs() []*jobmanager.Job {
	return e.jobManager.ListJobs()
}

// GetJobNotificationChannel returns the channel for job notifications
func (e *ExecutionEngine) GetJobNotificationChannel() <-chan jobmanager.JobNotification {
	return e.jobManager.GetNotificationChannel()
}

// WaitForAllJobs waits for all background jobs to complete and collects their output
func (e *ExecutionEngine) WaitForAllJobs() error {
	if e.verbose {
		fmt.Printf("DEBUG: WaitForAllJobs called - waiting for all background jobs to complete\n")
	}

	// Get the number of running jobs
	runningJobs := e.jobManager.GetRunningJobsCount()
	if e.verbose {
		fmt.Printf("DEBUG: WaitForAllJobs - currently running jobs: %d\n", runningJobs)
	}

	if runningJobs == 0 {
		if e.verbose {
			fmt.Printf("DEBUG: WaitForAllJobs - no running jobs, returning immediately\n")
		}
		return nil
	}

	// Wait for all jobs to complete by checking the job list periodically
	maxWaitTime := 10 * time.Second        // Maximum wait time
	checkInterval := 10 * time.Millisecond // Check interval
	startTime := time.Now()

	for {
		runningJobs := e.jobManager.GetRunningJobsCount()
		if e.verbose {
			fmt.Printf("DEBUG: WaitForAllJobs - checking running jobs: %d\n", runningJobs)
		}

		if runningJobs == 0 {
			if e.verbose {
				fmt.Printf("DEBUG: WaitForAllJobs - all jobs completed\n")
			}
			break
		}

		// Check if we've exceeded the maximum wait time
		if time.Since(startTime) > maxWaitTime {
			if e.verbose {
				fmt.Printf("DEBUG: WaitForAllJobs - timeout reached, %d jobs still running\n", runningJobs)
			}
			return errors.NewSystemError("JOB_WAIT_TIMEOUT", fmt.Sprintf("timeout waiting for %d background jobs to complete", runningJobs))
		}

		// Wait for the check interval
		time.Sleep(checkInterval)
	}

	// All jobs completed, now collect their output
	if e.verbose {
		fmt.Printf("DEBUG: WaitForAllJobs - collecting output from completed jobs\n")
	}

	// Get all completed jobs and collect their string output
	jobs := e.jobManager.ListJobs()
	var backgroundOutput strings.Builder

	for _, job := range jobs {
		if job.GetStatus() == jobmanager.StatusCompleted {
			result := job.GetResult()
			if e.verbose {
				fmt.Printf("DEBUG: WaitForAllJobs - job %d completed with result: %v\n", job.ID, result)
			}

			// If the job result is a string, collect it
			if resultStr, ok := result.(string); ok && resultStr != "" {
				if e.verbose {
					fmt.Printf("DEBUG: WaitForAllJobs - collecting output from job %d: '%s'\n", job.ID, resultStr)
				}
				if backgroundOutput.Len() > 0 {
					backgroundOutput.WriteString("\n")
				}
				backgroundOutput.WriteString(resultStr)
			}
		}
	}

	// If we collected any background output, store it in the engine
	if backgroundOutput.Len() > 0 {
		e.backgroundOutput = backgroundOutput.String()
		if e.verbose {
			fmt.Printf("DEBUG: WaitForAllJobs - collected background output: '%s'\n", e.backgroundOutput)
		}
	} else {
		e.backgroundOutput = ""
		if e.verbose {
			fmt.Printf("DEBUG: WaitForAllJobs - no background output collected\n")
		}
	}

	return nil
}

// executeImportStatement executes an import statement by loading and evaluating a file
func (e *ExecutionEngine) executeImportStatement(importStmt *ast.ImportStatement) (interface{}, error) {
	// Extract runtime name
	runtimeName := importStmt.Runtime.Value
	if runtimeName == "py" {
		runtimeName = "python"
	}

	// Extract file path
	filePath := importStmt.Path.Value
	if e.verbose {
		fmt.Printf("DEBUG: executeImportStatement called with runtime=%s, path=%s\n", runtimeName, filePath)
	}

	// Get or create the runtime
	rt, err := e.getRuntimeByName(runtimeName)
	if err != nil {
		return nil, errors.NewSystemError("RUNTIME_NOT_FOUND", fmt.Sprintf("failed to get runtime '%s': %v", runtimeName, err))
	}

	// Read the file content
	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		return nil, errors.NewSystemError("FILE_READ_ERROR", fmt.Sprintf("failed to read file '%s': %v", filePath, err))
	}

	// Evaluate the file content in the runtime
	if e.verbose {
		fmt.Printf("DEBUG: Evaluating file content in runtime %s\n", runtimeName)
	}

	// For JavaScript runtime, we need to execute the code in global scope
	// to make imported functions available for subsequent calls
	var result interface{}
	var evalErr error

	if runtimeName == "node" || runtimeName == "js" {
		// For JavaScript, we need to wrap the code to ensure functions are global
		jsCode := string(fileContent)

		// Simple approach: explicitly assign the function to globalThis
		wrappedCode := jsCode + "\n\n// Make functions global\nglobalThis.imported_function = imported_function;\nconsole.log('Functions imported successfully');"

		if e.verbose {
			fmt.Printf("DEBUG: Wrapped JavaScript code for import: %s\n", wrappedCode)
		}

		// Use ExecuteCodeBlock instead of Eval to avoid double wrapping
		if nodeRuntime, ok := rt.(*node.NodeRuntime); ok {
			result, evalErr = nodeRuntime.ExecuteCodeBlock(wrappedCode)
			if evalErr != nil {
				return nil, errors.NewRuntimeError(runtimeName, "IMPORT_EVAL_ERROR", fmt.Sprintf("failed to evaluate imported file '%s': %v", filePath, evalErr))
			}
		} else {
			// Fallback to Eval if not NodeRuntime
			result, evalErr = rt.Eval(wrappedCode)
			if evalErr != nil {
				return nil, errors.NewRuntimeError(runtimeName, "IMPORT_EVAL_ERROR", fmt.Sprintf("failed to evaluate imported file '%s': %v", filePath, evalErr))
			}
		}
	} else {
		// For other runtimes, use standard Eval
		result, evalErr = rt.Eval(string(fileContent))
		if evalErr != nil {
			return nil, errors.NewRuntimeError(runtimeName, "IMPORT_EVAL_ERROR", fmt.Sprintf("failed to evaluate imported file '%s': %v", filePath, evalErr))
		}
	}

	if e.verbose {
		fmt.Printf("DEBUG: Import evaluation completed successfully\n")
	}
	return result, nil
}

// executeCodeBlockStatement executes a code block by evaluating the code directly
func (e *ExecutionEngine) executeCodeBlockStatement(codeBlock *ast.CodeBlockStatement) (interface{}, error) {
	// Extract runtime name
	runtimeName := codeBlock.RuntimeToken.Value
	if runtimeName == "py" {
		runtimeName = "python"
	}
	// Handle alias 'js' for 'node'
	if runtimeName == "js" {
		runtimeName = "node"
	}

	// Extract code
	code := codeBlock.Code
	if e.verbose {
		fmt.Printf("DEBUG: executeCodeBlockStatement called with runtime=%s, code length=%d\n", runtimeName, len(code))
		if codeBlock.HasVariables() {
			fmt.Printf("DEBUG: Code block specifies variables to preserve: %v\n", codeBlock.GetVariableNames())
		} else {
			fmt.Printf("DEBUG: Code block will execute in isolation (no variable preservation)\n")
		}
	}

	// Get or create the runtime
	rt, err := e.getRuntimeByName(runtimeName)
	if err != nil {
		return nil, errors.NewSystemError("RUNTIME_NOT_FOUND", fmt.Sprintf("failed to get runtime '%s': %v", runtimeName, err))
	}

	// For Python runtime, use hybrid approach based on variable specifications
	if pythonRuntime, ok := rt.(*python.PythonRuntime); ok {
		if e.verbose {
			fmt.Printf("DEBUG: Using PythonRuntime with hybrid approach for code block\n")
		}

		var result interface{}
		var err error

		if codeBlock.HasVariables() {
			// Execute with variable preservation - use ExecuteCodeBlock with specified variables
			variableNames := codeBlock.GetVariableNames()
			if e.verbose {
				fmt.Printf("DEBUG: Executing with variable preservation: %v\n", variableNames)
			}
			result, err = pythonRuntime.ExecuteCodeBlockWithVariables(code, variableNames)
		} else {
			// Execute with variable preservation using ExecuteCodeBlock to save variables
			if e.verbose {
				fmt.Printf("DEBUG: Executing with variable preservation using ExecuteCodeBlock\n")
			}
			result, err = pythonRuntime.ExecuteCodeBlock(code)
		}

		if err != nil {
			return nil, errors.NewRuntimeError(runtimeName, "CODE_BLOCK_EVAL_ERROR", fmt.Sprintf("failed to evaluate code block: %v", err))
		}

		if e.verbose {
			fmt.Printf("DEBUG: Python code block evaluation completed successfully\n")
		}

		// Get all variables from Python runtime and store them in shared storage
		if e.verbose {
			fmt.Printf("DEBUG: Storing Python variables in shared storage\n")
		}

		// Get all variables from Python runtime
		pythonVariables := pythonRuntime.GetAllVariables()
		if pythonVariables != nil {
			for varName, varValue := range pythonVariables {
				if e.verbose {
					fmt.Printf("DEBUG: Storing variable '%s' in shared storage: %v\n", varName, varValue)
				}
				e.SetSharedVariable("python", varName, varValue)
			}
		}

		// Check for captured output (similar to executeLanguageCallStatement)
		capturedOutput := pythonRuntime.GetCapturedOutput()
		if e.verbose {
			fmt.Printf("DEBUG: Captured output from Python runtime: '%s'\n", capturedOutput)
		}
		// Return the captured output if it exists, otherwise return the result
		if capturedOutput != "" {
			return capturedOutput, nil
		}

		// Wait for all background jobs to complete before returning
		if err := e.WaitForAllJobs(); err != nil {
			if e.verbose {
				fmt.Printf("DEBUG: executeCodeBlockStatement - error waiting for background jobs: %v\n", err)
			}
			return nil, err
		}

		// Check if this is a statement that shouldn't return a value (function/class definition, etc.)
		if e.isStatementWithoutReturnValue(code, runtimeName) {
			return nil, nil
		}

		// For Python code blocks, if the result is empty or None, return "✓ Executed"
		if result == nil || result == "" || result == "None" {
			return "✓ Executed", nil
		}

		return result, nil
	}

	// For Node.js runtime, use hybrid approach based on variable specifications
	if nodeRuntime, ok := rt.(*node.NodeRuntime); ok {
		if e.verbose {
			fmt.Printf("DEBUG: Using NodeRuntime with hybrid approach for code block\n")
		}

		var result interface{}
		var err error

		if codeBlock.HasVariables() {
			// Execute with variable preservation - use ExecuteCodeBlock with specified variables
			variableNames := codeBlock.GetVariableNames()
			if e.verbose {
				fmt.Printf("DEBUG: Executing with variable preservation: %v\n", variableNames)
			}
			result, err = nodeRuntime.ExecuteCodeBlockWithVariables(code, variableNames)
		} else {
			// Execute in isolation - use ExecuteCodeBlock for proper code block handling
			if e.verbose {
				fmt.Printf("DEBUG: Executing in isolation (no variable preservation)\n")
			}
			result, err = nodeRuntime.ExecuteCodeBlock(code)
		}

		if err != nil {
			return nil, errors.NewRuntimeError(runtimeName, "CODE_BLOCK_EVAL_ERROR", fmt.Sprintf("failed to evaluate code block: %v", err))
		}

		if e.verbose {
			fmt.Printf("DEBUG: Node.js code block evaluation completed successfully\n")
		}

		// For code blocks, we don't use captured output - we return the processed result from ExecuteCodeBlock
		// The captured output is already handled inside ExecuteCodeBlock via processExecuteCodeBlockOutput
		if e.verbose {
			capturedOutput := nodeRuntime.GetCapturedOutput()
			fmt.Printf("DEBUG: Captured output from Node runtime (ignored for code blocks): '%s'\n", capturedOutput)
		}

		// Check if this is a statement that shouldn't return a value
		if e.isStatementWithoutReturnValue(code, runtimeName) {
			return nil, nil
		}

		// For Node.js code blocks, if the result is just "undefined", it means we defined functions/variables
		// In this case, we should return nil to avoid printing "undefined"
		if resultStr, ok := result.(string); ok && strings.TrimSpace(resultStr) == "undefined" {
			return nil, nil
		}

		// For Node.js code blocks with variables, don't show "✓ Executed" - they should be silent
		if codeBlock.HasVariables() {
			// Handle Node.js specific output patterns for code blocks with variables
			if resultStr, ok := result.(string); ok {
				trimmedResult := strings.TrimSpace(resultStr)
				if trimmedResult == "| | undefined" || trimmedResult == "undefined" || trimmedResult == "" {
					return nil, nil
				}
			}
			if result == nil || result == "" {
				return nil, nil
			}
		} else {
			// For Node.js code blocks without variables, if the result is empty or contains REPL artifacts, return "✓ Executed"
			if result == nil || result == "" {
				return "✓ Executed", nil
			}
			// Handle Node.js specific output patterns for code blocks without variables
			if resultStr, ok := result.(string); ok {
				trimmedResult := strings.TrimSpace(resultStr)
				if trimmedResult == "| | undefined" || trimmedResult == "undefined" {
					return "✓ Executed", nil
				}
			}
		}

		// Note: Variables are now captured and stored automatically in ExecuteCodeBlockWithVariables
		if codeBlock.HasVariables() && e.verbose {
			fmt.Printf("DEBUG: Variables should already be captured by ExecuteCodeBlockWithVariables\n")
		}

		return result, nil
	}

	// For Lua runtime, use hybrid approach based on variable specifications
	if luaRuntime, ok := rt.(*lua.LuaRuntime); ok {
		if e.verbose {
			fmt.Printf("DEBUG: Using LuaRuntime with hybrid approach for code block\n")
		}

		var result interface{}
		var err error

		if codeBlock.HasVariables() {
			// Execute with variable preservation - use ExecuteCodeBlock with specified variables
			variableNames := codeBlock.GetVariableNames()
			if e.verbose {
				fmt.Printf("DEBUG: Executing with variable preservation: %v\n", variableNames)
			}
			result, err = luaRuntime.ExecuteCodeBlockWithVariables(code, variableNames)
		} else {
			// Execute in isolation - use Eval without buffering
			if e.verbose {
				fmt.Printf("DEBUG: Executing in isolation (no variable preservation)\n")
			}
			// Set engine in Lua global for functions that need it
			ud := luaRuntime.GetState().NewUserData()
			ud.Value = e
			luaRuntime.GetState().SetGlobal("execution_engine", ud)
			result, err = luaRuntime.Eval(code)
		}

		if err != nil {
			return nil, errors.NewRuntimeError(runtimeName, "CODE_BLOCK_EVAL_ERROR", fmt.Sprintf("failed to evaluate code block: %v", err))
		}

		if e.verbose {
			fmt.Printf("DEBUG: Lua code block evaluation completed successfully\n")
			fmt.Printf("DEBUG: Lua code block result: %v (type: %T)\n", result, result)
		}

		// Note: Captured output is already handled by the Eval function and returned as result
		// No need to call GetCapturedOutput() here as it would return empty buffer

		// Check if this is a statement that shouldn't return a value
		shouldReturnNil := e.isStatementWithoutReturnValue(code, runtimeName)
		if e.verbose {
			fmt.Printf("DEBUG: Lua code block - isStatementWithoutReturnValue: %v\n", shouldReturnNil)
		}
		if shouldReturnNil {
			return nil, nil
		}

		if e.verbose {
			fmt.Printf("DEBUG: Lua code block returning result: %v\n", result)
		}
		return result, nil
	}

	// For other runtimes, use the standard Eval method
	if e.verbose {
		fmt.Printf("DEBUG: Evaluating code block in runtime %s\n", runtimeName)
	}
	result, err := rt.Eval(code)
	if err != nil {
		return nil, errors.NewRuntimeError(runtimeName, "CODE_BLOCK_EVAL_ERROR", fmt.Sprintf("failed to evaluate code block: %v", err))
	}

	if e.verbose {
		fmt.Printf("DEBUG: Code block evaluation completed successfully\n")
	}

	// Check if this is a statement that shouldn't return a value
	if e.isStatementWithoutReturnValue(code, runtimeName) {
		return nil, nil
	}

	return result, nil
}

// evaluateExpression evaluates an AST expression and returns its Go value.
func (e *ExecutionEngine) evaluateExpression(expr ast.Expression) (interface{}, error) {
	switch ex := expr.(type) {
	case *ast.StringLiteral:
		return ex.Value, nil
	case *ast.NumberLiteral:
		return ex.Value, nil
	case *ast.BooleanLiteral:
		return ex.Value, nil
	case *ast.Identifier:
		// First check local scope (e.g., loop variables, pattern matching variables)
		if val, found := e.localScope.Get(ex.Name); found {
			return val, nil
		}
		// Then check shared variables
		if val, found := e.GetSharedVariable(ex.Language, ex.Name); found {
			return val, nil
		}
		// Finally, check runtime
		if rt, err := e.runtimeManager.GetRuntime(ex.Language); err == nil {
			return rt.GetVariable(ex.Name)
		}
		return nil, fmt.Errorf("undefined variable: %s", ex.Name)
	case *ast.LanguageCall:
		return e.executeLanguageCallNew(ex)
	case *ast.BinaryExpression:
		return e.executeBinaryExpression(ex)
	case *ast.UnaryExpression:
		return e.executeUnaryExpression(ex)
	case *ast.PipeExpression:
		return e.executePipeExpression(ex)
	case *ast.TernaryExpression:
		return e.executeTernaryExpression(ex)
	case *ast.ElvisExpression:
		return e.executeElvisExpression(ex)
	case *ast.IndexExpression:
		return e.executeIndexExpression(ex)
	case *ast.FieldAccess:
		return e.executeFieldAccess(ex)
	case *ast.SizeExpression:
		return e.executeSizeExpression(ex)
	default:
		return nil, fmt.Errorf("unsupported expression type for evaluation: %T", expr)
	}
}

// executeSizeExpression выполняет выражение получения размера битстринга (@variable)
func (e *ExecutionEngine) executeSizeExpression(expr *ast.SizeExpression) (interface{}, error) {
	// Получаем значение выражения
	value, err := e.evaluateExpression(expr.Expression)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate expression in size operator: %v", err)
	}

	// Проверяем, что это битстринг
	switch v := value.(type) {
	case *shared.BitstringObject:
		// Возвращаем размер в байтах
		return int(v.BitString.Length() / 8), nil
	case *funbit.BitString:
		// Возвращаем размер в байтах
		return int(v.Length() / 8), nil
	default:
		return nil, fmt.Errorf("size operator (@) can only be applied to bitstrings, got %T", value)
	}
}
