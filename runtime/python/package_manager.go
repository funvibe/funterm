package python

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"funterm/errors"
)

// PackageInfo represents information about an installed Python package
type PackageInfo struct {
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	Location     string            `json:"location"`
	Dependencies []string          `json:"dependencies"`
	Metadata     map[string]string `json:"metadata"`
}

// PythonPackageManager manages Python package installation and operations
type PythonPackageManager struct {
	pythonPath        string
	pipPath           string
	venvPath          string
	installedPackages map[string]PackageInfo
	mutex             sync.RWMutex
	ready             bool
	executionTimeout  time.Duration
}

// NewPythonPackageManager creates a new package manager instance
func NewPythonPackageManager(pythonPath, venvPath string) *PythonPackageManager {
	// Create the struct instance first
	pm := &PythonPackageManager{}

	// Initialize fields one by one to avoid any potential memory alignment issues
	if pythonPath == "" {
		pm.pythonPath = "python3"
	} else {
		pm.pythonPath = pythonPath
	}

	pm.pipPath = "pip3"
	pm.venvPath = venvPath
	pm.installedPackages = make(map[string]PackageInfo)
	pm.executionTimeout = 5 * time.Minute
	pm.ready = false

	return pm
}

// Initialize sets up the package manager and checks for required tools
func (pm *PythonPackageManager) Initialize() error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	// Check if pip is available
	if err := pm.checkPipAvailability(); err != nil {
		return fmt.Errorf("pip not available: %w", err)
	}

	// Load installed packages
	if err := pm.loadInstalledPackages(); err != nil {
		// Don't fail if we can't load packages, just start with empty list
		fmt.Printf("Warning: Could not load installed packages: %v\n", err)
	}

	pm.ready = true
	return nil
}

// checkPipAvailability checks if pip is available on the system
func (pm *PythonPackageManager) checkPipAvailability() error {
	cmd := exec.Command(pm.pipPath, "--version")
	if err := cmd.Run(); err != nil {
		// Try pip as fallback
		pm.pipPath = "pip"
		cmd = exec.Command(pm.pipPath, "--version")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("neither pip3 nor pip found in PATH")
		}
	}
	return nil
}

// loadInstalledPackages loads the list of currently installed packages
func (pm *PythonPackageManager) loadInstalledPackages() error {
	ctx, cancel := context.WithTimeout(context.Background(), pm.executionTimeout)
	defer cancel()

	pipPath := pm.pipPath
	if pm.venvPath != "" {
		// Use pip from virtual environment
		pipPath = filepath.Join(pm.venvPath, "bin", "pip")
		if _, err := os.Stat(pipPath); err != nil {
			// Fallback to system pip
			pipPath = pm.pipPath
		}
	}

	cmd := exec.CommandContext(ctx, pipPath, "list", "--format=json")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to list packages: %w", err)
	}

	var packages []struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}

	if err := json.Unmarshal(output, &packages); err != nil {
		return fmt.Errorf("failed to parse package list: %w", err)
	}

	pm.installedPackages = make(map[string]PackageInfo)
	for _, pkg := range packages {
		pm.installedPackages[strings.ToLower(pkg.Name)] = PackageInfo{
			Name:    pkg.Name,
			Version: pkg.Version,
		}
	}

	return nil
}

// InstallPackage installs a Python package
func (pm *PythonPackageManager) InstallPackage(packageName string, version string) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	if !pm.ready {
		return errors.NewRuntimeError("python", "PACKAGE_MANAGER_NOT_INITIALIZED", "package manager is not initialized")
	}

	// Check if package is already installed (without calling IsPackageInstalled to avoid deadlock)
	lowerName := strings.ToLower(packageName)
	if _, exists := pm.installedPackages[lowerName]; exists {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), pm.executionTimeout)
	defer cancel()

	pipPath := pm.pipPath
	if pm.venvPath != "" {
		// Use pip from virtual environment
		pipPath = filepath.Join(pm.venvPath, "bin", "pip")
		if _, err := os.Stat(pipPath); err != nil {
			// Fallback to system pip
			pipPath = pm.pipPath
		}
	}

	// Prepare install command
	installSpec := packageName
	if version != "" {
		installSpec = fmt.Sprintf("%s==%s", packageName, version)
	}

	args := []string{"install", installSpec}
	if pm.venvPath != "" {
		// Install in virtual environment
		args = append(args, "--upgrade")
	}

	cmd := exec.CommandContext(ctx, pipPath, args...)

	// Capture output for better error handling
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		errorOutput := stderr.String()
		return fmt.Errorf("failed to install package %s: %w\nOutput: %s", packageName, err, errorOutput)
	}

	// Reload installed packages
	if loadErr := pm.loadInstalledPackages(); loadErr != nil {
		fmt.Printf("Warning: Could not reload package list after installation: %v\n", loadErr)
	}

	return nil
}

// UninstallPackage uninstalls a Python package
func (pm *PythonPackageManager) UninstallPackage(packageName string) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	if !pm.ready {
		return errors.NewRuntimeError("python", "PACKAGE_MANAGER_NOT_INITIALIZED", "package manager is not initialized")
	}

	// Check if package is installed (without calling IsPackageInstalled to avoid deadlock)
	lowerName := strings.ToLower(packageName)
	if _, exists := pm.installedPackages[lowerName]; !exists {
		return fmt.Errorf("package %s is not installed", packageName)
	}

	ctx, cancel := context.WithTimeout(context.Background(), pm.executionTimeout)
	defer cancel()

	pipPath := pm.pipPath
	if pm.venvPath != "" {
		// Use pip from virtual environment
		pipPath = filepath.Join(pm.venvPath, "bin", "pip")
		if _, err := os.Stat(pipPath); err != nil {
			// Fallback to system pip
			pipPath = pm.pipPath
		}
	}

	cmd := exec.CommandContext(ctx, pipPath, "uninstall", "-y", packageName)

	// Capture output for better error handling
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		errorOutput := stderr.String()
		return fmt.Errorf("failed to uninstall package %s: %w\nOutput: %s", packageName, err, errorOutput)
	}

	// Reload installed packages
	if loadErr := pm.loadInstalledPackages(); loadErr != nil {
		fmt.Printf("Warning: Could not reload package list after uninstallation: %v\n", loadErr)
	}

	return nil
}

// CheckPackage checks if a package is installed and returns its version
func (pm *PythonPackageManager) CheckPackage(packageName string) (string, error) {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	if !pm.ready {
		return "", errors.NewRuntimeError("python", "PACKAGE_MANAGER_NOT_INITIALIZED", "package manager is not initialized")
	}

	lowerName := strings.ToLower(packageName)
	if pkg, exists := pm.installedPackages[lowerName]; exists {
		return pkg.Version, nil
	}

	return "", fmt.Errorf("package %s is not installed", packageName)
}

// IsPackageInstalled checks if a package is installed
func (pm *PythonPackageManager) IsPackageInstalled(packageName string) bool {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	lowerName := strings.ToLower(packageName)
	_, exists := pm.installedPackages[lowerName]
	return exists
}

// ListPackages returns a list of all installed packages
func (pm *PythonPackageManager) ListPackages() []PackageInfo {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	packages := make([]PackageInfo, 0, len(pm.installedPackages))
	for _, pkg := range pm.installedPackages {
		packages = append(packages, pkg)
	}

	return packages
}

// GetPackageInfo returns detailed information about a package
func (pm *PythonPackageManager) GetPackageInfo(packageName string) (*PackageInfo, error) {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	if !pm.ready {
		return nil, errors.NewRuntimeError("python", "PACKAGE_MANAGER_NOT_INITIALIZED", "package manager is not initialized")
	}

	lowerName := strings.ToLower(packageName)
	if pkg, exists := pm.installedPackages[lowerName]; exists {
		return &pkg, nil
	}

	return nil, fmt.Errorf("package %s is not installed", packageName)
}

// UpgradePackage upgrades a package to the latest version
func (pm *PythonPackageManager) UpgradePackage(packageName string) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	if !pm.ready {
		return errors.NewRuntimeError("python", "PACKAGE_MANAGER_NOT_INITIALIZED", "package manager is not initialized")
	}

	// Check if package is installed (without calling IsPackageInstalled to avoid deadlock)
	lowerName := strings.ToLower(packageName)
	if _, exists := pm.installedPackages[lowerName]; !exists {
		return fmt.Errorf("package %s is not installed", packageName)
	}

	ctx, cancel := context.WithTimeout(context.Background(), pm.executionTimeout)
	defer cancel()

	pipPath := pm.pipPath
	if pm.venvPath != "" {
		// Use pip from virtual environment
		pipPath = filepath.Join(pm.venvPath, "bin", "pip")
		if _, err := os.Stat(pipPath); err != nil {
			// Fallback to system pip
			pipPath = pm.pipPath
		}
	}

	cmd := exec.CommandContext(ctx, pipPath, "install", "--upgrade", packageName)

	// Capture output for better error handling
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		errorOutput := stderr.String()
		return fmt.Errorf("failed to upgrade package %s: %w\nOutput: %s", packageName, err, errorOutput)
	}

	// Reload installed packages
	if loadErr := pm.loadInstalledPackages(); loadErr != nil {
		fmt.Printf("Warning: Could not reload package list after upgrade: %v\n", loadErr)
	}

	return nil
}

// SearchPackages searches for packages on PyPI
func (pm *PythonPackageManager) SearchPackages(query string) ([]string, error) {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	if !pm.ready {
		return nil, errors.NewRuntimeError("python", "PACKAGE_MANAGER_NOT_INITIALIZED", "package manager is not initialized")
	}

	ctx, cancel := context.WithTimeout(context.Background(), pm.executionTimeout)
	defer cancel()

	pipPath := pm.pipPath
	if pm.venvPath != "" {
		// Use pip from virtual environment
		pipPath = filepath.Join(pm.venvPath, "bin", "pip")
		if _, err := os.Stat(pipPath); err != nil {
			// Fallback to system pip
			pipPath = pm.pipPath
		}
	}

	cmd := exec.CommandContext(ctx, pipPath, "search", query)

	// Capture output
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		errorOutput := stderr.String()
		return nil, fmt.Errorf("failed to search packages: %w\nOutput: %s", err, errorOutput)
	}

	// Parse search results
	return pm.parseSearchResults(stdout.String()), nil
}

// parseSearchResults parses pip search output to extract package names
func (pm *PythonPackageManager) parseSearchResults(output string) []string {
	var packages []string
	scanner := bufio.NewScanner(strings.NewReader(output))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			// Extract package name (usually before the first space or hyphen)
			if strings.Contains(line, " ") {
				parts := strings.SplitN(line, " ", 2)
				if len(parts) > 0 {
					pkgName := strings.TrimSpace(parts[0])
					if pkgName != "" {
						packages = append(packages, pkgName)
					}
				}
			}
		}
	}

	return packages
}

// SetExecutionTimeout sets the timeout for package operations
func (pm *PythonPackageManager) SetExecutionTimeout(timeout time.Duration) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()
	pm.executionTimeout = timeout
}

// GetExecutionTimeout returns the current timeout for package operations
func (pm *PythonPackageManager) GetExecutionTimeout() time.Duration {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()
	return pm.executionTimeout
}

// IsReady checks if the package manager is ready for operations
func (pm *PythonPackageManager) IsReady() bool {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()
	return pm.ready
}

// Cleanup releases resources used by the package manager
func (pm *PythonPackageManager) Cleanup() error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	pm.ready = false
	pm.installedPackages = make(map[string]PackageInfo)
	return nil
}

// InstallBasicPackages installs commonly used packages
func (pm *PythonPackageManager) InstallBasicPackages() error {
	basicPackages := []struct {
		name    string
		version string
	}{
		{"numpy", ""},
		{"pandas", ""},
		{"requests", ""},
	}

	for _, pkg := range basicPackages {
		if err := pm.InstallPackage(pkg.name, pkg.version); err != nil {
			fmt.Printf("Warning: Failed to install %s: %v\n", pkg.name, err)
			// Continue with other packages even if one fails
		}
	}

	return nil
}
