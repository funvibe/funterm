package python

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"funterm/errors"
)

// EnvironmentInfo represents information about a Python virtual environment
type EnvironmentInfo struct {
	Path         string            `json:"path"`
	Name         string            `json:"name"`
	PythonPath   string            `json:"python_path"`
	PipPath      string            `json:"pip_path"`
	IsActive     bool              `json:"is_active"`
	CreatedAt    string            `json:"created_at"`
	LastModified string            `json:"last_modified"`
	Size         int64             `json:"size"`
	Metadata     map[string]string `json:"metadata"`
}

// EnvironmentManager manages Python virtual environments
type EnvironmentManager struct {
	pythonPath        string
	basePath          string
	environments      map[string]*EnvironmentInfo
	activeEnvironment string
	mutex             sync.RWMutex
	ready             bool
	executionTimeout  time.Duration
}

// NewEnvironmentManager creates a new environment manager instance
func NewEnvironmentManager(pythonPath, basePath string) *EnvironmentManager {
	if pythonPath == "" {
		pythonPath = "python3"
	}
	if basePath == "" {
		basePath = filepath.Join(".", "funterm_envs")
	}

	return &EnvironmentManager{
		pythonPath:       pythonPath,
		basePath:         basePath,
		environments:     make(map[string]*EnvironmentInfo),
		executionTimeout: 10 * time.Minute, // Longer timeout for environment operations
	}
}

// Initialize sets up the environment manager and loads existing environments
func (em *EnvironmentManager) Initialize() error {
	em.mutex.Lock()
	defer em.mutex.Unlock()

	// Create base directory if it doesn't exist
	if err := os.MkdirAll(em.basePath, 0755); err != nil {
		return fmt.Errorf("failed to create base directory: %w", err)
	}

	// Load existing environments
	if err := em.loadEnvironments(); err != nil {
		// Don't fail if we can't load environments, just start with empty list
		fmt.Printf("Warning: Could not load existing environments: %v\n", err)
	}

	em.ready = true
	return nil
}

// loadEnvironments scans the base directory for existing virtual environments
func (em *EnvironmentManager) loadEnvironments() error {
	entries, err := os.ReadDir(em.basePath)
	if err != nil {
		return fmt.Errorf("failed to read base directory: %w", err)
	}

	em.environments = make(map[string]*EnvironmentInfo)

	for _, entry := range entries {
		if entry.IsDir() {
			envPath := filepath.Join(em.basePath, entry.Name())
			if em.isValidVirtualEnvironment(envPath) {
				envInfo, err := em.getEnvironmentInfo(envPath)
				if err == nil {
					em.environments[entry.Name()] = envInfo
				}
			}
		}
	}

	return nil
}

// isValidVirtualEnvironment checks if a directory is a valid Python virtual environment
func (em *EnvironmentManager) isValidVirtualEnvironment(path string) bool {
	// Check for common virtual environment indicators
	var pythonExe, pipExe string

	if runtime.GOOS == "windows" {
		pythonExe = filepath.Join(path, "Scripts", "python.exe")
		pipExe = filepath.Join(path, "Scripts", "pip.exe")
	} else {
		pythonExe = filepath.Join(path, "bin", "python")
		pipExe = filepath.Join(path, "bin", "pip")
	}

	// Check if both python and pip executables exist
	if _, err := os.Stat(pythonExe); err != nil {
		return false
	}

	if _, err := os.Stat(pipExe); err != nil {
		return false
	}

	// Additionally, check for pyvenv.cfg file (standard in Python 3.3+ venvs)
	pyvenvCfg := filepath.Join(path, "pyvenv.cfg")
	if _, err := os.Stat(pyvenvCfg); err == nil {
		return true
	}

	// Check for lib/pythonX.Y/site-packages directory
	libDirs, err := filepath.Glob(filepath.Join(path, "lib", "python*.*", "site-packages"))
	if err == nil && len(libDirs) > 0 {
		return true
	}

	return false
}

// getEnvironmentInfo gathers information about a virtual environment
func (em *EnvironmentManager) getEnvironmentInfo(path string) (*EnvironmentInfo, error) {
	info := &EnvironmentInfo{
		Path:     path,
		Name:     filepath.Base(path),
		Metadata: make(map[string]string),
	}

	// Set Python and pip paths
	if runtime.GOOS == "windows" {
		info.PythonPath = filepath.Join(path, "Scripts", "python.exe")
		info.PipPath = filepath.Join(path, "Scripts", "pip.exe")
	} else {
		info.PythonPath = filepath.Join(path, "bin", "python")
		info.PipPath = filepath.Join(path, "bin", "pip")
	}

	// Get file info for timestamps and size
	fileInfo, err := os.Stat(path)
	if err == nil {
		info.CreatedAt = fileInfo.ModTime().Format(time.RFC3339)
		info.LastModified = fileInfo.ModTime().Format(time.RFC3339)
	}

	// Calculate size
	size, err := em.getDirectorySize(path)
	if err == nil {
		info.Size = size
	}

	// Check if this is the active environment
	info.IsActive = (em.activeEnvironment == info.Name)

	// Get Python version
	if version, err := em.getPythonVersion(info.PythonPath); err == nil {
		info.Metadata["python_version"] = version
	}

	return info, nil
}

// getDirectorySize calculates the total size of a directory
func (em *EnvironmentManager) getDirectorySize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size, err
}

// getPythonVersion gets the Python version from a specific Python executable
func (em *EnvironmentManager) getPythonVersion(pythonPath string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), em.executionTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, pythonPath, "--version")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get Python version: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// CreateVenv creates a new virtual environment
func (em *EnvironmentManager) CreateVenv(name string, pythonVersion string) error {
	em.mutex.Lock()
	defer em.mutex.Unlock()

	if !em.ready {
		return errors.NewRuntimeError("python", "ENVIRONMENT_MANAGER_NOT_INITIALIZED", "environment manager is not initialized")
	}

	// Check if environment already exists
	if _, exists := em.environments[name]; exists {
		return fmt.Errorf("environment '%s' already exists", name)
	}

	envPath := filepath.Join(em.basePath, name)

	// Create the virtual environment
	ctx, cancel := context.WithTimeout(context.Background(), em.executionTimeout)
	defer cancel()

	var cmd *exec.Cmd
	if pythonVersion != "" {
		// Use specific Python version
		cmd = exec.CommandContext(ctx, pythonVersion, "-m", "venv", envPath)
	} else {
		// Use default Python path
		cmd = exec.CommandContext(ctx, em.pythonPath, "-m", "venv", envPath)
	}

	// Capture output for better error handling
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		errorOutput := stderr.String()
		return fmt.Errorf("failed to create virtual environment '%s': %w\nOutput: %s", name, err, errorOutput)
	}

	// Load the new environment info
	envInfo, err := em.getEnvironmentInfo(envPath)
	if err == nil {
		em.environments[name] = envInfo
	}

	return nil
}

// RemoveVenv removes a virtual environment
func (em *EnvironmentManager) RemoveVenv(name string) error {
	em.mutex.Lock()
	defer em.mutex.Unlock()

	if !em.ready {
		return errors.NewRuntimeError("python", "ENVIRONMENT_MANAGER_NOT_INITIALIZED", "environment manager is not initialized")
	}

	// Check if environment exists
	envInfo, exists := em.environments[name]
	if !exists {
		return fmt.Errorf("environment '%s' does not exist", name)
	}

	// Cannot remove active environment
	if em.activeEnvironment == name {
		return fmt.Errorf("cannot remove active environment '%s'", name)
	}

	// Remove the environment directory
	if err := os.RemoveAll(envInfo.Path); err != nil {
		return fmt.Errorf("failed to remove environment directory: %w", err)
	}

	// Remove from environments map
	delete(em.environments, name)

	return nil
}

// ActivateVenv activates a virtual environment
func (em *EnvironmentManager) ActivateVenv(name string) error {
	em.mutex.Lock()
	defer em.mutex.Unlock()

	if !em.ready {
		return errors.NewRuntimeError("python", "ENVIRONMENT_MANAGER_NOT_INITIALIZED", "environment manager is not initialized")
	}

	// Check if environment exists
	if _, exists := em.environments[name]; !exists {
		return fmt.Errorf("environment '%s' does not exist", name)
	}

	// Deactivate current environment if any
	if em.activeEnvironment != "" {
		if oldEnv, exists := em.environments[em.activeEnvironment]; exists {
			oldEnv.IsActive = false
		}
	}

	// Activate the new environment
	em.activeEnvironment = name
	if env, exists := em.environments[name]; exists {
		env.IsActive = true
	}

	return nil
}

// DeactivateVenv deactivates the current virtual environment
func (em *EnvironmentManager) DeactivateVenv() error {
	em.mutex.Lock()
	defer em.mutex.Unlock()

	if !em.ready {
		return errors.NewRuntimeError("python", "ENVIRONMENT_MANAGER_NOT_INITIALIZED", "environment manager is not initialized")
	}

	if em.activeEnvironment == "" {
		return nil // No active environment
	}

	// Deactivate current environment
	if env, exists := em.environments[em.activeEnvironment]; exists {
		env.IsActive = false
	}

	em.activeEnvironment = ""
	return nil
}

// GetActiveVenv returns the active virtual environment
func (em *EnvironmentManager) GetActiveVenv() (*EnvironmentInfo, error) {
	if em == nil {
		return nil, fmt.Errorf("environment manager is nil")
	}

	em.mutex.RLock()
	defer em.mutex.RUnlock()

	if !em.ready {
		return nil, errors.NewRuntimeError("python", "ENVIRONMENT_MANAGER_NOT_INITIALIZED", "environment manager is not initialized")
	}

	if em.activeEnvironment == "" {
		return nil, nil // No active environment
	}

	if env, exists := em.environments[em.activeEnvironment]; exists {
		return env, nil
	}

	return nil, fmt.Errorf("active environment '%s' not found", em.activeEnvironment)
}

// ListVenvs returns a list of all virtual environments
func (em *EnvironmentManager) ListVenvs() []*EnvironmentInfo {
	em.mutex.RLock()
	defer em.mutex.RUnlock()

	envs := make([]*EnvironmentInfo, 0, len(em.environments))
	for _, env := range em.environments {
		envs = append(envs, env)
	}

	return envs
}

// GetVenvInfo returns information about a specific virtual environment
func (em *EnvironmentManager) GetVenvInfo(name string) (*EnvironmentInfo, error) {
	em.mutex.RLock()
	defer em.mutex.RUnlock()

	if !em.ready {
		return nil, errors.NewRuntimeError("python", "ENVIRONMENT_MANAGER_NOT_INITIALIZED", "environment manager is not initialized")
	}

	if env, exists := em.environments[name]; exists {
		return env, nil
	}

	return nil, fmt.Errorf("environment '%s' not found", name)
}

// VenvExists checks if a virtual environment exists
func (em *EnvironmentManager) VenvExists(name string) bool {
	em.mutex.RLock()
	defer em.mutex.RUnlock()

	_, exists := em.environments[name]
	return exists
}

// UpgradeVenv upgrades a virtual environment's packages
func (em *EnvironmentManager) UpgradeVenv(name string) error {
	em.mutex.Lock()
	defer em.mutex.Unlock()

	if !em.ready {
		return errors.NewRuntimeError("python", "ENVIRONMENT_MANAGER_NOT_INITIALIZED", "environment manager is not initialized")
	}

	// Check if environment exists
	envInfo, exists := em.environments[name]
	if !exists {
		return fmt.Errorf("environment '%s' does not exist", name)
	}

	// Upgrade pip in the environment
	ctx, cancel := context.WithTimeout(context.Background(), em.executionTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, envInfo.PipPath, "install", "--upgrade", "pip")

	// Capture output for better error handling
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		errorOutput := stderr.String()
		return fmt.Errorf("failed to upgrade pip in environment '%s': %w\nOutput: %s", name, err, errorOutput)
	}

	return nil
}

// CreateDefaultVenv creates a default virtual environment with basic packages
func (em *EnvironmentManager) CreateDefaultVenv() error {
	defaultEnvName := "funterm_env"

	// Check if default environment already exists
	if em.VenvExists(defaultEnvName) {
		// Just activate it
		return em.ActivateVenv(defaultEnvName)
	}

	// Create the default environment
	if err := em.CreateVenv(defaultEnvName, ""); err != nil {
		return fmt.Errorf("failed to create default environment: %w", err)
	}

	// Activate it
	if err := em.ActivateVenv(defaultEnvName); err != nil {
		return fmt.Errorf("failed to activate default environment: %w", err)
	}

	return nil
}

// GetPythonPath returns the Python executable path for the active environment
func (em *EnvironmentManager) GetPythonPath() (string, error) {
	em.mutex.RLock()
	defer em.mutex.RUnlock()

	if !em.ready {
		return "", errors.NewRuntimeError("python", "ENVIRONMENT_MANAGER_NOT_INITIALIZED", "environment manager is not initialized")
	}

	if em.activeEnvironment == "" {
		// Return system Python path
		return em.pythonPath, nil
	}

	if env, exists := em.environments[em.activeEnvironment]; exists {
		return env.PythonPath, nil
	}

	return "", fmt.Errorf("active environment not found")
}

// GetPipPath returns the pip executable path for the active environment
func (em *EnvironmentManager) GetPipPath() (string, error) {
	em.mutex.RLock()
	defer em.mutex.RUnlock()

	if !em.ready {
		return "", errors.NewRuntimeError("python", "ENVIRONMENT_MANAGER_NOT_INITIALIZED", "environment manager is not initialized")
	}

	if em.activeEnvironment == "" {
		// Return system pip path
		return "pip3", nil
	}

	if env, exists := em.environments[em.activeEnvironment]; exists {
		return env.PipPath, nil
	}

	return "", fmt.Errorf("active environment not found")
}

// SetExecutionTimeout sets the timeout for environment operations
func (em *EnvironmentManager) SetExecutionTimeout(timeout time.Duration) {
	em.mutex.Lock()
	defer em.mutex.Unlock()
	em.executionTimeout = timeout
}

// GetExecutionTimeout returns the current timeout for environment operations
func (em *EnvironmentManager) GetExecutionTimeout() time.Duration {
	em.mutex.RLock()
	defer em.mutex.RUnlock()
	return em.executionTimeout
}

// IsReady checks if the environment manager is ready for operations
func (em *EnvironmentManager) IsReady() bool {
	em.mutex.RLock()
	defer em.mutex.RUnlock()
	return em.ready
}

// Cleanup releases resources used by the environment manager
func (em *EnvironmentManager) Cleanup() error {
	em.mutex.Lock()
	defer em.mutex.Unlock()

	em.ready = false
	em.environments = make(map[string]*EnvironmentInfo)
	em.activeEnvironment = ""
	return nil
}
