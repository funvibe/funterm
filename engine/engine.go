package engine

import (
	goerrors "errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"sync"
	"time"

	"funterm/container"
	"funterm/errors"
	"funterm/jobmanager"
	"funterm/runtime"
	"funterm/runtime/lua"
	"funterm/runtime/node"
	"funterm/runtime/python"
	"funterm/shared"
	"go-parser/pkg/ast"
	"go-parser/pkg/parser"
	sharedparser "go-parser/pkg/shared"

	"github.com/funvibe/funbit/pkg/funbit"
)

// executeIdFunction is a builtin function that returns the value of a variable
func (e *ExecutionEngine) executeIdFunction(args []interface{}) (interface{}, error) {
	if len(args) != 1 {
		return nil, errors.NewUserError("ID_ARGUMENT_ERROR", "id() function requires exactly one argument")
	}

	// Return the argument as-is - this allows us to test variable access
	return args[0], nil
}

// executePrintFunction is a builtin print function that returns formatted values for REPL display
func (e *ExecutionEngine) executePrintFunction(args []interface{}) (interface{}, error) {
	// Format arguments using the shared formatter (same as REPL but without quotes for strings)
	var strArgs []string
	for _, arg := range args {
		strArgs = append(strArgs, shared.FormatValueForDisplay(arg))
	}

	// Join with spaces and return as pre-formatted result (for REPL to display with => without extra quotes)
	output := strings.Join(strArgs, " ")
	return &shared.PreFormattedResult{Value: output}, nil
}



// executeLenFunction is a builtin function that returns the length of arrays, strings, or maps
func (e *ExecutionEngine) executeLenFunction(args []interface{}) (interface{}, error) {
	if len(args) != 1 {
		return nil, errors.NewUserError("LEN_ARGUMENT_ERROR", "len() function requires exactly one argument")
	}

	arg := args[0]

	// Handle different types
	switch v := arg.(type) {
	case string:
		return float64(len(v)), nil
	case []interface{}:
		return float64(len(v)), nil
	case []byte:
		return float64(len(v)), nil
	case map[string]interface{}:
		return float64(len(v)), nil
	case nil:
		return float64(0), nil
	default:
		// Try to handle as a generic slice/array using reflection
		// This is for compatibility with different types from runtimes
		return float64(0), errors.NewUserError("LEN_TYPE_ERROR", fmt.Sprintf("len() cannot determine length of type %T", arg))
	}
}

// executeConcatFunction is a builtin function that concatenates two arrays
func (e *ExecutionEngine) executeConcatFunction(args []interface{}) (interface{}, error) {
	if len(args) < 2 {
		return nil, errors.NewUserError("CONCAT_ARGUMENT_ERROR", "concat() function requires two or more arguments")
	}

	result := make([]interface{}, 0)
	for i := 0; i < len(args); i++ {
		ary, ok := args[i].([]interface{})
		if !ok {
			return nil, errors.NewUserError("CONCAT_TYPE_ERROR", fmt.Sprintf("concat() function requires arguments to be arrays, bad argument number: %d", i+1))
		}
		result = append(result, ary...)
	}

	return result, nil
}

// Execute parses and executes a command string, returning result, isPrint flag, and error
func (e *ExecutionEngine) Execute(command string) (interface{}, bool, bool, error) {
	if e.verbose {
		fmt.Printf("DEBUG: Executing command: '%s'\n", command)
	}

	// Parse the command
	statement, parseErrors := e.parser.Parse(command)
	if len(parseErrors) > 0 {
		if e.verbose {
			fmt.Printf("DEBUG: Parser errors: %v\n", parseErrors)
		}
		// Return the first parsing error with position information
		firstError := parseErrors[0]
		return nil, false, false, errors.NewUserErrorWithASTPos("PARSING_ERROR", firstError.Message, firstError.Position)
	}

	if statement == nil {
		if e.verbose {
			fmt.Printf("DEBUG: Statement is nil\n")
		}
		return nil, false, false, errors.NewUserError("UNSUPPORTED_COMMAND", "unsupported command")
	}

	if e.verbose {
		fmt.Printf("DEBUG: Statement type: %T\n", statement)
	}

	// Determine if the statement is a print function before execution
	isPrint := false
	hasResult := true // By default, assume we have a result to display
	if langCallStmt, ok := statement.(*ast.LanguageCallStatement); ok && langCallStmt.LanguageCall != nil {
		if e.isPrintFunction(langCallStmt.LanguageCall) {
			isPrint = true
			hasResult = false // Print functions don't have a result to display
		}
	}

	// ExpressionStatement should always display its result, even if nil
	if _, ok := statement.(*ast.ExpressionStatement); ok {
		isPrint = false
		hasResult = true // Always show the result of expressions
	}

	// Execute the statement and collect output
	result, err := e.executeStatement(statement)
	if err != nil {
		return nil, isPrint, hasResult, err
	}

	// Check if the result indicates a print function was executed
	if e.isPrintResult(result) {
		isPrint = true
		hasResult = false
	}

	// Handle bitstring output formatting
	if byteResult, ok := result.([]byte); ok {
		// Format bitstring output in Erlang style
		formattedOutput := e.formatBitstringOutput(byteResult)
		if e.verbose {
			fmt.Printf("DEBUG: Execute - returning formatted bitstring output: '%s'\n", formattedOutput)
		}
		return formattedOutput, isPrint, hasResult, nil
	}

	// If this is a pre-formatted result (like from print function), return its value
	if preFormatted, ok := result.(*shared.PreFormattedResult); ok {
		if e.verbose {
			fmt.Printf("DEBUG: Execute - returning pre-formatted result: '%s'\n", preFormatted.Value)
		}
		return preFormatted.Value, isPrint, hasResult, nil
	}

	// If this is a print statement that produced string output, return it
	if resultStr, ok := result.(string); ok {
		if e.verbose {
			fmt.Printf("DEBUG: Execute - returning string output: '%s'\n", resultStr)
		}
		return resultStr, isPrint, hasResult, nil
	}

	// For other cases, return the result as is
	if e.verbose {
		fmt.Printf("DEBUG: Execute - returning result: %v\n", result)
	}
	return result, isPrint, hasResult, nil
}

// ExecuteCompat parses and executes a command string, returning only result and error for backward compatibility
func (e *ExecutionEngine) ExecuteCompat(command string) (interface{}, error) {
	result, _, _, err := e.Execute(command)
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
	if e.verbose {
		fmt.Printf("DEBUG: executeStatement called with type %T\n", stmt)
	}
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
		if e.verbose {
			fmt.Printf("DEBUG: executeStatement - found VariableAssignment, calling executeVariableAssignment\n")
		}
		return e.executeVariableAssignment(s)
	case *ast.ExpressionAssignment:
		return e.executeExpressionAssignment(s)
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
	case *ast.CStyleForLoopStatement:
		return e.executeCStyleForLoop(s)
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
	case *ast.BuiltinFunctionCall:
		return e.executeBuiltinFunctionCall(s)
	case *ast.ExpressionStatement:
		return e.convertExpressionToValue(s.Expression)
	default:
		return nil, errors.NewUserError("UNSUPPORTED_STATEMENT", fmt.Sprintf("unsupported statement type: %T", stmt))
	}
}

// executeLanguageCallStatement executes a language call statement (with optional background execution)
func (e *ExecutionEngine) executeLanguageCallStatement(stmt *ast.LanguageCallStatement) (interface{}, error) {
	if stmt.LanguageCall == nil {
		return nil, errors.NewUserError("INVALID_LANGUAGE_CALL", "language call is nil")
	}

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

// isPrintResult checks if the result indicates a print function was executed
func (e *ExecutionEngine) isPrintResult(result interface{}) bool {
	return false // builtin print now returns a result to display
}

// executeBackgroundLanguageCall executes a language call as a background task
func (e *ExecutionEngine) executeBackgroundLanguageCall(stmt *ast.LanguageCallStatement) (interface{}, error) {
	// Create a command string for the job
	command := fmt.Sprintf("%s.%s(%s)", stmt.LanguageCall.Language, stmt.LanguageCall.Function, e.formatArgumentsForCommand(stmt.LanguageCall.Arguments))

	// Clone the current scope for background task isolation
	clonedScope := e.cloneCurrentScope()

	// Clone shared variables for background task
	clonedSharedVariables := e.cloneSharedVariables()

	// Submit the job to the JobManager
	jobID, err := e.jobManager.Submit(func() (interface{}, error) {
		// Create a temporary engine instance for this background task
		backgroundEngine := &ExecutionEngine{
			parser:            e.parser,
			runtimeManager:    e.runtimeManager,
			runtimeRegistry:   e.runtimeRegistry,
			container:         e.container,
			jobManager:        e.jobManager, // Share the same job manager
			sharedVariables:   clonedSharedVariables,
			variablesMutex:    sync.RWMutex{},
			verbose:           e.verbose,
			jobFinished:       e.jobFinished,
			localScope:        clonedScope,
			scopeStack:        []*sharedparser.Scope{clonedScope},
			backgroundOutput:  "",
			runtimeCache:      e.runtimeCache,
			runtimeCacheMutex: sync.RWMutex{},
		}

		// This function will be executed in the background with isolated scope
		result, err := backgroundEngine.executeLanguageCallNew(stmt.LanguageCall)
		if err != nil {
			return nil, err
		}

		// Get the runtime to capture output
		rt, err := backgroundEngine.runtimeManager.GetRuntime(stmt.LanguageCall.Language)
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
			if a.IsInt {
				argStrings = append(argStrings, a.IntValue.String())
			} else {
				argStrings = append(argStrings, fmt.Sprintf("%v", a.FloatValue))
			}
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

			// Get variables from Node runtime and store them in shared storage
			if e.verbose {
				fmt.Printf("DEBUG: Storing Node.js variables in shared storage\n")
			}

			// Get the specified variables from Node runtime's variables cache
			for _, varName := range variableNames {
				if nodeVarValue, err := nodeRuntime.GetVariableFromRuntimeObject(varName); err == nil {
					if e.verbose {
						fmt.Printf("DEBUG: Storing variable '%s' in shared storage: %v\n", varName, nodeVarValue)
					}
					e.SetSharedVariable("node", varName, nodeVarValue)
				}
			}
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

			// Get variables from Lua runtime and store them in shared storage
			if e.verbose {
				fmt.Printf("DEBUG: Storing Lua variables in shared storage\n")
			}

			// Get the specified variables from Lua runtime's runtimeObjects
			for _, varName := range variableNames {
				if luaVarValue, err := luaRuntime.GetVariableFromRuntimeObject(varName); err == nil {
					if e.verbose {
						fmt.Printf("DEBUG: Storing variable '%s' in shared storage: %v\n", varName, luaVarValue)
					}
					e.SetSharedVariable("lua", varName, luaVarValue)
				}
			}
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
		if ex.IsInt {
			if ex.IntValue.IsInt64() {
				return ex.IntValue.Int64(), nil
			} else {
				return ex.IntValue, nil
			}
		} else {
			return ex.FloatValue, nil
		}
	case *ast.BooleanLiteral:
		return ex.Value, nil
	case *ast.Identifier:
		// First check local scope chain (e.g., loop variables, pattern matching variables)
		if val, found := e.getVariable(ex.Name); found {
			return val, nil
		}

		// For unqualified identifiers (ex.Language is empty), DO NOT check runtime
		// Variables from code blocks should only be accessible via qualified access
		if ex.Language == "" {
			// Return nil instead of error for undefined unqualified variables
			// This prevents accessing runtime variables without qualification
			return nil, nil
		}

		// For qualified identifiers, first check shared variables
		// This ensures that runtime variables are isolated and only accessible via qualification
		if val, found := e.GetSharedVariable(ex.Language, ex.Name); found {
			return val, nil
		}

		// If not found in shared storage, try to sync from runtime and then check again
		// This ensures we have the latest value from runtime execution
		if rt, err := e.runtimeManager.GetRuntime(ex.Language); err == nil {
			if runtimeVal, runtimeErr := rt.GetVariable(ex.Name); runtimeErr == nil {
				// Sync the value to shared storage for future access
				e.SetSharedVariable(ex.Language, ex.Name, runtimeVal)
				if e.verbose {
					fmt.Printf("DEBUG: evaluateExpression - synced runtime variable %s.%s = %v to shared storage\n", ex.Language, ex.Name, runtimeVal)
				}
				return runtimeVal, nil
			}
		}

		// If not found anywhere, return nil instead of error
		// This maintains consistency with unqualified variable handling
		return nil, nil
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
	case *ast.BuiltinFunctionCall:
		return e.executeBuiltinFunctionCall(ex)
	case *ast.NestedExpression:
		// Для вложенных выражений просто вычисляем внутреннее выражение
		return e.evaluateExpression(ex.Inner)
	default:
		return nil, fmt.Errorf("unsupported expression type for evaluation: %T", expr)
	}
}

// executeSizeExpression выполняет выражение получения размера битстринга (@variable)
func (e *ExecutionEngine) executeSizeExpression(expr *ast.SizeExpression) (interface{}, error) {
	// Получаем значение выражения
	var value interface{}
	var err error

	if expr.Expression != nil {
		value, err = e.convertExpressionToValue(expr.Expression)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate expression in size operator: %v", err)
		}
	} else if expr.Variable != "" {
		// Legacy support for Variable field
		if val, found := e.getVariable(expr.Variable); found {
			value = val
		} else if val, found := e.getGlobalVariable(expr.Variable); found {
			value = val
		} else {
			return nil, fmt.Errorf("undefined variable '%s' in size operator", expr.Variable)
		}
	} else {
		return nil, fmt.Errorf("invalid size expression: no expression or variable specified")
	}

	// Конвертируем различные типы в BitstringObject для получения размера
	var bitstringData *shared.BitstringObject

	switch v := value.(type) {
	case *shared.BitstringObject:
		bitstringData = v
	case *funbit.BitString:
		// Оборачиваем в BitstringObject
		bitstringData = &shared.BitstringObject{BitString: v}
	case shared.BitstringByte:
		// Convert BitstringByte to BitstringObject (single byte)
		bitString := funbit.NewBitStringFromBytes([]byte{v.Value})
		bitstringData = &shared.BitstringObject{BitString: bitString}
	case string:
		// Convert string (from binary patterns) to BitstringObject
		bitString := funbit.NewBitStringFromBytes([]byte(v))
		bitstringData = &shared.BitstringObject{BitString: bitString}
	case []byte:
		// Convert byte slice to BitstringObject
		bitString := funbit.NewBitStringFromBytes(v)
		bitstringData = &shared.BitstringObject{BitString: bitString}
	default:
		return nil, fmt.Errorf("size operator (@) can only be applied to bitstrings, got %T", value)
	}

	// Return size in bytes
	return int(bitstringData.BitString.Length() / 8), nil
}

// executeBuiltinFunctionCall executes a builtin function call (like id())
func (e *ExecutionEngine) executeBuiltinFunctionCall(call *ast.BuiltinFunctionCall) (interface{}, error) {
	if e.verbose {
		fmt.Printf("DEBUG: executeBuiltinFunctionCall called with function: %s, args: %v\n", call.Function, call.Arguments)
	}

	// Convert arguments from AST expressions to Go values
	args := make([]interface{}, len(call.Arguments))
	for i, arg := range call.Arguments {
		value, err := e.convertExpressionToValue(arg)
		if err != nil {
			return nil, fmt.Errorf("failed to convert argument %d: %v", i, err)
		}
		args[i] = value
	}

	switch call.Function {
	case "id":
		return e.executeIdFunction(args)
	case "len":
		return e.executeLenFunction(args)
	case "concat":
		return e.executeConcatFunction(args)
	case "print":
		return e.executePrintFunction(args)
	default:
		return nil, fmt.Errorf("unsupported builtin function: %s", call.Function)
	}
}

// pushScope creates a new nested scope and pushes it onto the stack
func (e *ExecutionEngine) pushScope() {
	if e.verbose {
		fmt.Printf("DEBUG: Pushing new scope onto stack (current depth: %d, current scope isRoot: %v)\n", len(e.scopeStack), e.localScope.IsRoot())
	}
	newScope := sharedparser.NewScope(e.localScope)
	e.scopeStack = append(e.scopeStack, newScope)
	e.localScope = newScope
	if e.verbose {
		fmt.Printf("DEBUG: New scope created with parent, new depth: %d, new scope isRoot: %v\n", len(e.scopeStack), e.localScope.IsRoot())
	}
}

// popScope removes the current scope from the stack and restores the previous one
func (e *ExecutionEngine) popScope() {
	if len(e.scopeStack) <= 1 {
		// Never pop the root scope
		if e.verbose {
			fmt.Printf("DEBUG: Cannot pop root scope, keeping current scope\n")
		}
		return
	}

	if e.verbose {
		fmt.Printf("DEBUG: Popping scope from stack (current depth: %d, current scope isRoot: %v)\n", len(e.scopeStack), e.localScope.IsRoot())
	}

	// Remove current scope from stack
	e.scopeStack = e.scopeStack[:len(e.scopeStack)-1]
	// Restore previous scope as current
	e.localScope = e.scopeStack[len(e.scopeStack)-1]

	if e.verbose {
		fmt.Printf("DEBUG: Restored scope, new depth: %d, restored scope isRoot: %v\n", len(e.scopeStack), e.localScope.IsRoot())
	}
}

// getCurrentScope returns the current scope
func (e *ExecutionEngine) getCurrentScope() *sharedparser.Scope {
	return e.localScope
}

// setVariable sets a variable in the current scope only
func (e *ExecutionEngine) setVariable(name string, value interface{}) {
	if e.verbose {
		fmt.Printf("DEBUG: Setting variable '%s' = %v in current scope (depth: %d, isRoot: %v)\n", name, value, len(e.scopeStack), e.localScope.IsRoot())
	}
	e.localScope.Set(name, value)
}

// setVariableInParentScope finds the parent scope that contains the variable and updates it there
func (e *ExecutionEngine) setVariableInParentScope(name string, value interface{}) {
	if e.verbose {
		fmt.Printf("DEBUG: setVariableInParentScope - looking for variable '%s' in parent scopes\n", name)
	}

	// Start from the parent scope (skip current scope)
	parent := e.localScope.GetParent()
	depth := len(e.scopeStack) - 1

	for parent != nil {
		if varInfo, found := parent.GetVariableInfo(name); found {
			if e.verbose {
				mutabilityStr := "immutable"
				if varInfo.IsMutable {
					mutabilityStr = "mutable"
				}
				fmt.Printf("DEBUG: setVariableInParentScope - found variable '%s' in parent scope at depth %d, updating there (preserving %s)\n", name, depth, mutabilityStr)
			}
			// Preserve existing mutability
			parent.SetWithMutability(name, value, varInfo.IsMutable)
			return
		}
		parent = parent.GetParent()
		depth--
	}

	// If we get here, the variable wasn't found in any parent scope
	// Check if it exists in globals
	if varInfo, found := e.getGlobalVariableInfo(name); found {
		if e.verbose {
			mutabilityStr := "immutable"
			if varInfo.IsMutable {
				mutabilityStr = "mutable"
			}
			fmt.Printf("DEBUG: setVariableInParentScope - found variable '%s' in global variables, updating there (preserving %s)\n", name, mutabilityStr)
		}
		// Preserve existing mutability
		e.setGlobalVariableWithMutability(name, value, varInfo.IsMutable)
		return
	}
	
	// Variable not found anywhere - create in current scope as fallback
	if e.verbose {
		fmt.Printf("DEBUG: setVariableInParentScope - variable '%s' not found in any parent scope or globals, creating in current scope\n", name)
	}
	e.setVariable(name, value)
}

// getVariable retrieves a variable from the scope chain (current + parents)
func (e *ExecutionEngine) getVariable(name string) (interface{}, bool) {
	value, found := e.localScope.Get(name)
	
	// If not found in local scopes, check global variables
	if !found {
		globalValue, globalFound := e.getGlobalVariable(name)
		if globalFound {
			value = globalValue
			found = true
		}
	}
	
	if e.verbose {
		fmt.Printf("DEBUG: Getting variable '%s' = %v, found: %v (scope depth: %d, current scope isRoot: %v)\n", name, value, found, len(e.scopeStack), e.localScope.IsRoot())
		if !found {
			// Debug: check if variable exists in parent scopes manually
			parent := e.localScope.GetParent()
			depth := len(e.scopeStack) - 1
			for parent != nil {
				if parentVal, parentFound := parent.Get(name); parentFound {
					fmt.Printf("DEBUG: Variable '%s' FOUND in parent scope at depth %d: %v\n", name, depth, parentVal)
					break
				}
				parent = parent.GetParent()
				depth--
			}
			if parent == nil {
				fmt.Printf("DEBUG: Variable '%s' NOT FOUND in any parent scope\n", name)
			}
		}
	}
	return value, found
}

// setGlobalVariable sets a global variable accessible from all runtimes
func (e *ExecutionEngine) setGlobalVariable(name string, value interface{}) {
	e.globalMutex.Lock()
	defer e.globalMutex.Unlock()

	// Convert to VariableInfo structure (mutable by default for backward compatibility)
	varInfo := &sharedparser.VariableInfo{
		Value:     value,
		IsMutable: true,
	}

	e.globalVariables[name] = varInfo

	// Инвалидируем кэш синхронизации для этой переменной
	e.syncedGlobalMutex.Lock()
	delete(e.lastSyncedGlobals, name)
	e.syncedGlobalMutex.Unlock()

	if e.verbose {
		fmt.Printf("DEBUG: Set global variable '%s' = %v (mutable)\n", name, value)
	}
}

// getGlobalVariable retrieves a global variable value
func (e *ExecutionEngine) getGlobalVariable(name string) (interface{}, bool) {
	e.globalMutex.RLock()
	defer e.globalMutex.RUnlock()

	varInfo, found := e.globalVariables[name]
	var value interface{}
	if found {
		value = varInfo.Value
	}

	if e.verbose {
		fmt.Printf("DEBUG: Get global variable '%s' = %v, found: %v\n", name, value, found)
	}

	return value, found
}

// getAllGlobalVariables returns a copy of all global variables
func (e *ExecutionEngine) getAllGlobalVariables() map[string]interface{} {
	e.globalMutex.RLock()
	defer e.globalMutex.RUnlock()

	// Return a copy to avoid race conditions
	result := make(map[string]interface{})
	for k, varInfo := range e.globalVariables {
		result[k] = varInfo.Value
	}

	return result
}

// syncGlobalVariablesToRuntime synchronizes all global variables to a specific runtime
func (e *ExecutionEngine) syncGlobalVariablesToRuntime(rt runtime.LanguageRuntime) error {
	e.globalMutex.RLock()
	globals := make(map[string]interface{})
	for name, varInfo := range e.globalVariables {
		globals[name] = varInfo.Value
	}
	e.globalMutex.RUnlock()

	// Используем кэш для отслеживания изменений и избегаем повторной синхронизации
	e.syncedGlobalMutex.Lock()
	defer e.syncedGlobalMutex.Unlock()

	// Счетчик синхронизированных переменных для debug
	var syncCount int
	
	for name, value := range globals {
		// Проверяем, изменилась ли переменная или это новая переменная
		lastValue, exists := e.lastSyncedGlobals[name]
		if exists && reflect.DeepEqual(lastValue, value) {
			// Переменная не изменилась, пропускаем
			if e.verbose {
				fmt.Printf("DEBUG: Skipping sync for unchanged global variable '%s'\n", name)
			}
			continue
		}

		// Переменная изменилась или это новая переменная - синхронизируем
		if err := rt.SetVariable(name, value); err != nil {
			if e.verbose {
				fmt.Printf("DEBUG: Warning - failed to sync global variable '%s' to runtime: %v\n", name, err)
			}
			// Continue with other variables even if one fails
		} else {
			syncCount++
			if e.verbose {
				fmt.Printf("DEBUG: Synced global variable '%s' = %v to runtime\n", name, value)
			}
		}
		
		// Обновляем кэш
		e.lastSyncedGlobals[name] = value
	}

	// Удаляем переменные из кэша если их больше нет в globals
	for name := range e.lastSyncedGlobals {
		if _, exists := globals[name]; !exists {
			delete(e.lastSyncedGlobals, name)
			if e.verbose {
				fmt.Printf("DEBUG: Removed deleted global variable '%s' from sync cache\n", name)
			}
		}
	}

	if e.verbose && syncCount == 0 && len(globals) > 0 {
		fmt.Printf("DEBUG: All %d global variables already synced (no changes detected)\n", len(globals))
	}

	return nil
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
	case *ast.NilLiteral:
		return nil, nil
	case *ast.StringLiteral:
		return typedExpr.Value, nil
	case *ast.NumberLiteral:
		if typedExpr.IsInt {
			if typedExpr.IntValue.IsInt64() {
				return typedExpr.IntValue.Int64(), nil
			} else {
				return typedExpr.IntValue, nil
			}
		} else {
			return typedExpr.FloatValue, nil
		}
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
			// Simple identifier - check local scope first, then global variables
			if val, found := e.getVariable(typedExpr.Name); found {
				return val, nil
			}

			// Check global variables
			if val, found := e.getGlobalVariable(typedExpr.Name); found {
				return val, nil
			}

			// If not found in local or global scope, return nil instead of error
			// This prevents accessing runtime variables without qualification
			return nil, nil
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
	case *ast.BuiltinFunctionCall:
		return e.executeBuiltinFunctionCall(typedExpr)
	case *ast.NestedExpression:
		// Для вложенных выражений просто преобразуем внутреннее выражение
		return e.convertExpressionToValue(typedExpr.Inner)
	case *ast.BitstringPatternAssignment:
		// Выполняем inplace pattern matching и возвращаем boolean результат
		return e.executeBitstringPatternAssignment(typedExpr)
	case *ast.BitstringPatternMatchExpression:
		// Выполняем pattern matching и возвращаем boolean результат
		return e.executeBitstringPatternMatchExpression(typedExpr)
	case *ast.VariableAssignment:
		// Для цепочного присваивания (a = b = 3)
		// Выполняем вложенное присваивание, которое вернёт значение
		result, err := e.executeVariableAssignment(typedExpr)
		if err != nil {
			return nil, err
		}
		// Возвращаем результат выполнения присваивания (например для a = (b = 3), это 3)
		return result, nil
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

// cloneCurrentScope creates a deep copy of the current scope for background task isolation
func (e *ExecutionEngine) cloneCurrentScope() *sharedparser.Scope {
	e.variablesMutex.RLock()
	defer e.variablesMutex.RUnlock()

	// Create a new scope with the same parent as the current scope
	clonedScope := sharedparser.NewScope(e.localScope.GetParent())

	// Copy all variables from the current scope
	if e.localScope != nil {
		// Get all variables from the current scope and its parents
		currentVars := make(map[string]interface{})
		e.copyScopeVariables(e.localScope, currentVars)

		// Set the copied variables in the cloned scope
		for name, value := range currentVars {
			clonedScope.Set(name, value)
		}
	}

	return clonedScope
}

// copyScopeVariables recursively copies variables from a scope and all its parents
func (e *ExecutionEngine) copyScopeVariables(scope *sharedparser.Scope, target map[string]interface{}) {
	if scope == nil {
		return
	}

	// First copy from parent scopes (to maintain proper shadowing)
	e.copyScopeVariables(scope.GetParent(), target)

	// Then copy variables from this scope
	if vars := scope.GetAll(); vars != nil {
		for name, value := range vars {
			target[name] = value
		}
	}
}

// cloneSharedVariables creates a deep copy of shared variables for background task isolation
func (e *ExecutionEngine) cloneSharedVariables() map[string]map[string]interface{} {
	e.variablesMutex.RLock()
	defer e.variablesMutex.RUnlock()

	cloned := make(map[string]map[string]interface{})

	for language, vars := range e.sharedVariables {
		cloned[language] = make(map[string]interface{})
		for name, value := range vars {
			cloned[language][name] = value
		}
	}

	return cloned
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

// setVariableWithMutability sets a variable with explicit mutability flag in the current scope
func (e *ExecutionEngine) setVariableWithMutability(name string, value interface{}, isMutable bool) {
	if e.verbose {
		mutabilityStr := "immutable"
		if isMutable {
			mutabilityStr = "mutable"
		}
		fmt.Printf("DEBUG: Setting variable '%s' = %v (%s) in current scope (depth: %d, isRoot: %v)\n", name, value, mutabilityStr, len(e.scopeStack), e.localScope.IsRoot())
	}
	e.localScope.SetWithMutability(name, value, isMutable)
}

// getVariableInfo retrieves variable information from the scope chain (current + parents)
func (e *ExecutionEngine) getVariableInfo(name string) (*sharedparser.VariableInfo, bool) {
	varInfo, found := e.localScope.GetVariableInfo(name)
	if e.verbose {
		if found {
			mutabilityStr := "immutable"
			if varInfo.IsMutable {
				mutabilityStr = "mutable"
			}
			fmt.Printf("DEBUG: Getting variable info '%s' = %v (%s), found: %v (scope depth: %d, current scope isRoot: %v)\n", name, varInfo.Value, mutabilityStr, found, len(e.scopeStack), e.localScope.IsRoot())
		} else {
			fmt.Printf("DEBUG: Getting variable info '%s', found: %v (scope depth: %d, current scope isRoot: %v)\n", name, found, len(e.scopeStack), e.localScope.IsRoot())
		}
	}
	return varInfo, found
}

// setGlobalVariableWithMutability sets a global variable with explicit mutability flag
func (e *ExecutionEngine) setGlobalVariableWithMutability(name string, value interface{}, isMutable bool) {
	e.globalMutex.Lock()
	defer e.globalMutex.Unlock()

	// Convert to VariableInfo structure
	varInfo := &sharedparser.VariableInfo{
		Value:     value,
		IsMutable: isMutable,
	}

	e.globalVariables[name] = varInfo

	// Инвалидируем кэш синхронизации для этой переменной
	e.syncedGlobalMutex.Lock()
	delete(e.lastSyncedGlobals, name)
	e.syncedGlobalMutex.Unlock()

	if e.verbose {
		mutabilityStr := "immutable"
		if isMutable {
			mutabilityStr = "mutable"
		}
		fmt.Printf("DEBUG: Set global variable '%s' = %v (%s)\n", name, value, mutabilityStr)
	}
}

// getGlobalVariableInfo retrieves global variable information
func (e *ExecutionEngine) getGlobalVariableInfo(name string) (*sharedparser.VariableInfo, bool) {
	e.globalMutex.RLock()
	defer e.globalMutex.RUnlock()

	varInfo, found := e.globalVariables[name]

	if e.verbose {
		if found {
			mutabilityStr := "immutable"
			if varInfo.IsMutable {
				mutabilityStr = "mutable"
			}
			fmt.Printf("DEBUG: Get global variable info '%s' = %v (%s), found: %v\n", name, varInfo.Value, mutabilityStr, found)
		} else {
			fmt.Printf("DEBUG: Get global variable info '%s', found: %v\n", name, found)
		}
	}

	return varInfo, found
}

// convertExpressionToValueForCStyleForLoop converts expression to value for C-style for loop
// This method prioritizes global variables for C-style for loop variable access
func (e *ExecutionEngine) convertExpressionToValueForCStyleForLoop(expr ast.Expression) (interface{}, error) {
	switch typedExpr := expr.(type) {
	case *ast.BooleanLiteral:
		return typedExpr.Value, nil
	case *ast.StringLiteral:
		return typedExpr.Value, nil
	case *ast.NumberLiteral:
		if typedExpr.IsInt {
			if typedExpr.IntValue.IsInt64() {
				return typedExpr.IntValue.Int64(), nil
			} else {
				return typedExpr.IntValue, nil
			}
		} else {
			return typedExpr.FloatValue, nil
		}
	case *ast.VariableRead:
		// For VariableRead expressions, execute the variable read
		return e.executeVariableRead(typedExpr)
	case *ast.Identifier:
		// For C-style for loop, prioritize global variables
		if val, found := e.getGlobalVariable(typedExpr.Name); found {
			return val, nil
		}

		// If not found in global, check local scope
		if val, found := e.getVariable(typedExpr.Name); found {
			return val, nil
		}

		// If not found anywhere, return nil instead of error
		return nil, nil
	case *ast.ArrayLiteral:
		// Convert array elements to []interface{}
		result := make([]interface{}, len(typedExpr.Elements))
		for i, element := range typedExpr.Elements {
			value, err := e.convertExpressionToValueForCStyleForLoop(element)
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
			value, err := e.convertExpressionToValueForCStyleForLoop(prop.Value)
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
	case *ast.BuiltinFunctionCall:
		return e.executeBuiltinFunctionCall(typedExpr)
	case *ast.NestedExpression:
		// Для вложенных выражений просто преобразуем внутреннее выражение
		return e.convertExpressionToValueForCStyleForLoop(typedExpr.Inner)
	case *ast.BitstringPatternAssignment:
		// Выполняем inplace pattern matching и возвращаем boolean результат
		return e.executeBitstringPatternAssignment(typedExpr)
	case *ast.BitstringPatternMatchExpression:
		// Выполняем pattern matching и возвращаем boolean результат
		return e.executeBitstringPatternMatchExpression(typedExpr)
	default:
		return nil, errors.NewSystemError("UNSUPPORTED_EXPRESSION", fmt.Sprintf("unsupported expression type: %T", expr))
	}
}
