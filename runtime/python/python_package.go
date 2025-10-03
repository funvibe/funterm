package python

import (
	"funterm/errors"
)

// enhanceError enhances an error using the error handler
func (pr *PythonRuntime) enhanceError(originalError string, command string) *errors.ExecutionError {
	context := &ExecutionContext{
		Command:        command,
		LineNumber:     pr.getCurrentLineNumber(),
		LocalVariables: pr.getLocalVariables(),
		CallStack:      []string{}, // Could be enhanced in future
	}

	return pr.errorHandler.EnhanceError(originalError, context)
}

// InstallPackage installs a Python package
func (pr *PythonRuntime) InstallPackage(packageName string, version string) error {
	if !pr.ready {
		return errors.NewRuntimeError("python", "RUNTIME_NOT_INITIALIZED", "runtime is not initialized")
	}

	if pr.packageManager == nil || !pr.packageManager.IsReady() {
		return errors.NewRuntimeError("python", "PACKAGE_MANAGER_NOT_AVAILABLE", "package manager is not available")
	}

	return pr.packageManager.InstallPackage(packageName, version)
}

// UninstallPackage uninstalls a Python package
func (pr *PythonRuntime) UninstallPackage(packageName string) error {
	if !pr.ready {
		return errors.NewRuntimeError("python", "RUNTIME_NOT_INITIALIZED", "runtime is not initialized")
	}

	if pr.packageManager == nil || !pr.packageManager.IsReady() {
		return errors.NewRuntimeError("python", "PACKAGE_MANAGER_NOT_AVAILABLE", "package manager is not available")
	}

	return pr.packageManager.UninstallPackage(packageName)
}

// CheckPackage checks if a package is installed and returns its version
func (pr *PythonRuntime) CheckPackage(packageName string) (string, error) {
	if !pr.ready {
		return "", errors.NewRuntimeError("python", "RUNTIME_NOT_INITIALIZED", "runtime is not initialized")
	}

	if pr.packageManager == nil || !pr.packageManager.IsReady() {
		return "", errors.NewRuntimeError("python", "PACKAGE_MANAGER_NOT_AVAILABLE", "package manager is not available")
	}

	return pr.packageManager.CheckPackage(packageName)
}

// ListPackages returns a list of all installed packages
func (pr *PythonRuntime) ListPackages() []PackageInfo {
	if !pr.ready || pr.packageManager == nil || !pr.packageManager.IsReady() {
		return []PackageInfo{}
	}

	return pr.packageManager.ListPackages()
}

// CreateVenv creates a new virtual environment (disabled in simplified runtime)
func (pr *PythonRuntime) CreateVenv(name string, pythonVersion string) error {
	return errors.NewRuntimeError("python", "FEATURE_DISABLED", "virtual environments are disabled in this version")
}

// ActivateVenv activates a virtual environment (disabled in simplified runtime)
func (pr *PythonRuntime) ActivateVenv(name string) error {
	return errors.NewRuntimeError("python", "FEATURE_DISABLED", "virtual environments are disabled in this version")
}

// DeactivateVenv deactivates the current virtual environment (disabled in simplified runtime)
func (pr *PythonRuntime) DeactivateVenv() error {
	return errors.NewRuntimeError("python", "FEATURE_DISABLED", "virtual environments are disabled in this version")
}

// ListVenvs returns a list of all virtual environments (disabled in simplified runtime)
func (pr *PythonRuntime) ListVenvs() []*EnvironmentInfo {
	return []*EnvironmentInfo{} // Always return empty list
}

// GetActiveVenv returns the active virtual environment (disabled in simplified runtime)
func (pr *PythonRuntime) GetActiveVenv() (*EnvironmentInfo, error) {
	return nil, errors.NewRuntimeError("python", "FEATURE_DISABLED", "virtual environments are disabled in this version")
}

// GetPackageManager returns the package manager instance
func (pr *PythonRuntime) GetPackageManager() *PythonPackageManager {
	pr.mutex.RLock()
	defer pr.mutex.RUnlock()
	return pr.packageManager
}

// GetEnvironmentManager returns the environment manager instance (disabled in simplified runtime)
func (pr *PythonRuntime) GetEnvironmentManager() *EnvironmentManager {
	return nil // Always return nil - environment manager disabled
}

// UseVirtualEnvironment controls whether to use virtual environments
// Virtual environment methods removed - functionality disabled for simplified runtime
