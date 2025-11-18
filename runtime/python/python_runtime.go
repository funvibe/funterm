package python

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"
)

const EndOfOutputMarker = "---SUTERM-PYTHON-EOP---"

// Global counter for unique execution IDs
var executionID int64

// Global shared Python runtime for tests
var (
	sharedTestRuntime *PythonRuntime
	testRuntimeOnce   sync.Once
	testRuntimeMutex  sync.Mutex
)

// PythonRuntime implements the LanguageRuntime interface for Python
type PythonRuntime struct {
	ready            bool
	available        bool // Flag to indicate if python executable is found
	variables        map[string]interface{}
	pythonPath       string
	mutex            sync.RWMutex
	processMutex     sync.Mutex    // Separate mutex for Python process access
	executionTimeout time.Duration // Timeout for Python execution
	// stateManager removed for simplified config
	errorHandler   *PythonErrorHandler
	packageManager *PythonPackageManager
	verbose        bool             // Enable verbose output
	outputCapture  *strings.Builder // For capturing stdout output
	// Test mode fields
	testMode bool // Enable test mode with shared instance
	// Persistent process fields
	cmd        *exec.Cmd
	stdin      io.WriteCloser
	stdout     io.ReadCloser
	stderr     io.ReadCloser
	resultChan chan string
	errorChan  chan error
}

// NewPythonRuntime creates a new Python runtime instance
func NewPythonRuntime() *PythonRuntime {
	return &PythonRuntime{
		ready:            false,
		available:        false, // Flag to indicate if python executable is found
		testMode:         false,
		variables:        make(map[string]interface{}),
		pythonPath:       "python3", // Default to python3, can be overridden
		mutex:            sync.RWMutex{},
		executionTimeout: 30 * time.Second, // Increased default timeout
		errorHandler:     NewPythonErrorHandler(),
		verbose:          false,
		outputCapture:    nil,
		cmd:              nil,
		stdin:            nil,
		stdout:           nil,
		stderr:           nil,
		resultChan:       nil,
		errorChan:        nil,
	}
}

// SetVerbose sets the verbose mode for the Python runtime
func (pr *PythonRuntime) SetVerbose(verbose bool) {
	pr.mutex.Lock()
	defer pr.mutex.Unlock()
	pr.verbose = verbose
}

// SetExecutionTimeout sets the execution timeout for the Python runtime
func (pr *PythonRuntime) SetExecutionTimeout(timeout time.Duration) {
	pr.mutex.Lock()
	defer pr.mutex.Unlock()
	pr.executionTimeout = timeout
}

// Initialize sets up the Python runtime with default configuration
func (pr *PythonRuntime) Initialize() error {
	// Use default python path and non-verbose mode
	// This will be overridden by InitializeWithConfig if called explicitly
	return pr.InitializeWithConfig("python3", false)
}

// InitializeWithConfig sets up the Python runtime with library configuration
func (pr *PythonRuntime) InitializeWithConfig(pythonPath string, verbose bool) error {
	pr.mutex.Lock()
	defer pr.mutex.Unlock()

	// Set configuration
	if pythonPath != "" {
		pr.pythonPath = pythonPath
	}
	pr.verbose = verbose

	// Check if Python is available
	if err := pr.checkPythonAvailability(); err != nil {
		// fmt.Printf("Warning: Python runtime is not available. %v\n", err)
		pr.ready = false
		pr.available = false
		// Do not return an error, just mark as unavailable
		return nil
	}
	pr.available = true

	// Environment manager initialization removed - virtual environments disabled by default

	// Initialize package manager
	if err := pr.initializePackageManager(); err != nil {
		fmt.Printf("Warning: Failed to initialize package manager: %v\n", err)
		// Continue without package management
	}

	// Simplified configuration - no complex library config needed for beta

	// Start the persistent Python process
	if err := pr.startPersistentProcess(); err != nil {
		fmt.Printf("Warning: Failed to start persistent Python process: %v\n", err)
		pr.available = false
		pr.ready = false
		// Do not return an error, just mark as unavailable
		return nil
	}

	// Initialize basic Python environment (skip for test mode to avoid hanging)
	if !pr.testMode {
		if err := pr.initializePythonEnvironment(); err != nil {
			fmt.Printf("Warning: Failed to initialize Python environment: %v\n", err)
			pr.available = false
			pr.ready = false
			// Do not return an error, just mark as unavailable
			return nil
		}
	}

	pr.ready = true
	return nil
}

// startPersistentProcess starts a persistent Python process for stateful execution
func (pr *PythonRuntime) startPersistentProcess() error {
	pr.cmd = exec.Command(pr.pythonPath, "-i", "-u")

	var err error
	pr.stdin, err = pr.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdin pipe: %w", err)
	}
	pr.stdout, err = pr.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe: %w", err)
	}
	pr.stderr, err = pr.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	if err := pr.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start persistent python process: %w", err)
	}

	pr.resultChan = make(chan string)
	pr.errorChan = make(chan error)

	// Start the dedicated stdout and stderr reader goroutines
	go pr.readOutput(pr.stdout, pr.resultChan)
	go pr.readError(pr.stderr, pr.errorChan)

	return nil
}

// checkPythonAvailability checks if Python is available on the system
func (pr *PythonRuntime) checkPythonAvailability() error {
	originalPath := pr.pythonPath
	cmd := exec.Command(pr.pythonPath, "--version")
	if err := cmd.Run(); err != nil {
		// Only try fallback if the original path was the default "python3"
		if originalPath == "python3" {
			pr.pythonPath = "python"
			cmd = exec.Command(pr.pythonPath, "--version")
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("neither python3 nor python found in PATH")
			}
		} else {
			// Custom path specified but not found
			return fmt.Errorf("python executable not found at specified path: %s", originalPath)
		}
	}
	return nil
}

// initializePythonEnvironment sets up the basic Python environment
func (pr *PythonRuntime) initializePythonEnvironment() error {
	// We need json for data exchange and other common modules for utility.
	envCode := `
import json
import math
import os
import sys
`
	_, err := pr.sendAndAwait(envCode)
	return err
}

// ensureJSONImported ensures that json module is imported
func (pr *PythonRuntime) ensureJSONImported() error {
	// Check if json is already imported (silently)
	checkCode := `
try:
    json
except NameError:
    import json
`
	_, err := pr.sendAndAwait(checkCode)
	return err
}

// clearPythonBuffers clears any remaining output in Python process buffers
func (pr *PythonRuntime) clearPythonBuffers() error {
	if pr.cmd == nil || pr.cmd.Process == nil {
		return nil // No process to clear
	}

	// Send multiple commands to thoroughly clear buffers
	clearCommands := []string{
		// Clear stdout/stderr buffers
		`
import sys
sys.stdout.flush()
sys.stderr.flush()
`,
		// Send empty print to consume any remaining output
		`print("")`,
		// Another flush to ensure everything is cleared
		`
import sys
sys.stdout.flush()
sys.stderr.flush()
`,
	}

	for _, cmd := range clearCommands {
		_, err := pr.sendAndAwait(cmd)
		if err != nil {
			if pr.verbose {
				fmt.Printf("DEBUG: Failed to execute clear command: %v\n", err)
			}
		}
	}

	return nil
}

// initializeEnvironmentManager removed - virtual environments disabled for simplified runtime

// initializePackageManager sets up the package manager
func (pr *PythonRuntime) initializePackageManager() error {
	// Virtual environment support removed - use system Python only
	var venvPath string

	// Create package manager
	pr.packageManager = &PythonPackageManager{
		pythonPath:        pr.pythonPath,
		pipPath:           "pip3",
		venvPath:          venvPath,
		installedPackages: make(map[string]PackageInfo),
		executionTimeout:  5 * time.Minute,
		ready:             false,
	}

	// Reduce timeout for testing (2 minutes instead of 5)
	if os.Getenv("GO_TEST") != "" {
		pr.packageManager.executionTimeout = 2 * time.Minute
	}

	if err := pr.packageManager.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize package manager: %w", err)
	}

	return nil
}

// setupDefaultEnvironment removed - virtual environments disabled for simplified runtime

// Cleanup releases resources used by the runtime
func (pr *PythonRuntime) Cleanup() error {
	pr.mutex.Lock()
	defer pr.mutex.Unlock()

	if pr.cmd != nil && pr.cmd.Process != nil {
		pr.cmd.Process.Kill()
		pr.cmd = nil
	}

	if pr.stdin != nil {
		pr.stdin.Close()
		pr.stdin = nil
	}

	pr.ready = false
	pr.available = false
	return nil
}

// GetName returns the name of the language runtime
func (pr *PythonRuntime) GetName() string {
	return "python"
}

// GetSupportedTypes returns the types supported by this runtime
func (pr *PythonRuntime) GetSupportedTypes() []string {
	return []string{"int", "float", "str", "bool", "list", "dict", "tuple", "set"}
}

// IsReady checks if the runtime is ready for execution
func (pr *PythonRuntime) IsReady() bool {
	pr.mutex.RLock()
	defer pr.mutex.RUnlock()
	return pr.ready
}

// Isolate creates an isolated state for the runtime
func (pr *PythonRuntime) Isolate() error {
	pr.mutex.Lock()
	defer pr.mutex.Unlock()

	if !pr.ready {
		return fmt.Errorf("Python runtime is not initialized")
	}

	// Clear all variables by restarting the Python process
	if pr.cmd != nil && pr.cmd.Process != nil {
		pr.cmd.Process.Kill()
		pr.cmd.Wait()
	}

	// Clear the local variable cache
	pr.variables = make(map[string]interface{})

	// Restart the process
	return pr.startPersistentProcess()
}

// RefreshRuntimeState method already exists in python_introspection.go

// indent добавляет отступы к каждой строке кода
func indent(code string, prefix string) string {
	lines := strings.Split(code, "\n")
	for i, line := range lines {
		lines[i] = prefix + line
	}
	return strings.Join(lines, "\n")
}

// SetTestMode enables or disables test mode
func (pr *PythonRuntime) SetTestMode(enabled bool) {
	pr.testMode = enabled
}

// GetSharedTestRuntime returns the global shared Python runtime for tests
func GetSharedTestRuntime() *PythonRuntime {
	testRuntimeOnce.Do(func() {
		sharedTestRuntime = NewPythonRuntime()
		sharedTestRuntime.testMode = true
		sharedTestRuntime.verbose = true // Enable verbose for debugging
		if err := sharedTestRuntime.Initialize(); err != nil {
			fmt.Printf("Warning: Failed to initialize shared test Python runtime: %v\n", err)
			// Don't fail completely, mark as not ready
			sharedTestRuntime.ready = false
		}
	})
	return sharedTestRuntime
}

// ResetForTest clears Python state between tests
func (pr *PythonRuntime) ResetForTest() error {
	if !pr.testMode {
		return fmt.Errorf("ResetForTest can only be called in test mode")
	}

	testRuntimeMutex.Lock()
	defer testRuntimeMutex.Unlock()

	// Check if Python process is still alive
	if pr.cmd != nil && pr.cmd.Process != nil {
		// Try to check if process is still running
		if err := pr.cmd.Process.Signal(syscall.Signal(0)); err != nil {
			// Process is dead, restart it
			if pr.verbose {
				fmt.Printf("DEBUG: Python process is dead, restarting for test reset\n")
			}
			if err := pr.startPersistentProcess(); err != nil {
				return fmt.Errorf("failed to restart Python process for test reset: %w", err)
			}
		}
	} else {
		// No process exists, start one
		if err := pr.startPersistentProcess(); err != nil {
			return fmt.Errorf("failed to start Python process for test reset: %w", err)
		}
	}

	// Reset global execution ID counter for clean test state
	executionID = 0

	// Clear local variable cache and output capture
	pr.mutex.Lock()
	pr.variables = make(map[string]interface{})
	pr.outputCapture = nil // Clear output capture between tests
	pr.mutex.Unlock()

	// Clear any remaining output in Python process buffers
	if err := pr.clearPythonBuffers(); err != nil {
		if pr.verbose {
			fmt.Printf("DEBUG: Failed to clear Python buffers: %v\n", err)
		}
	}

	// Clear all variables except built-ins (simplified for test mode)
	resetCode := `
# Clear all user-defined variables
user_vars = [name for name in globals() if not name.startswith('_') and name not in ['__builtins__', '__name__', '__doc__']]
for var in user_vars:
    del globals()[var]
`

	_, err := pr.sendAndAwait(resetCode)
	if err != nil {
		// If reset fails, try to restart the process
		if pr.verbose {
			fmt.Printf("DEBUG: Reset failed, restarting Python process: %v\n", err)
		}
		if restartErr := pr.startPersistentProcess(); restartErr != nil {
			return fmt.Errorf("failed to restart Python process after reset failure: %w", restartErr)
		}
		// Reset execution ID again after restart
		executionID = 0
		// Try reset again after restart
		_, err = pr.sendAndAwait(resetCode)
	}

	return err
}

// CleanupSharedTestRuntime cleans up the shared test runtime
func CleanupSharedTestRuntime() {
	testRuntimeMutex.Lock()
	defer testRuntimeMutex.Unlock()

	if sharedTestRuntime != nil {
		// Cleanup method removed for simplified beta version
		sharedTestRuntime = nil
		testRuntimeOnce = sync.Once{} // Reset the once
	}
}
