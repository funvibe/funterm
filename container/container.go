package container

import (
	"fmt"
	"reflect"
	"sync"

	"funterm/errors"
)

// DependencyLifetime defines the lifetime of a dependency
type DependencyLifetime int

const (
	// Transient means a new instance is created each time it's resolved
	Transient DependencyLifetime = iota
	// Singleton means the same instance is returned each time it's resolved
	Singleton
)

// Dependency represents a registered dependency
type Dependency struct {
	Factory   func() (interface{}, error)
	Instance  interface{}
	Lifetime  DependencyLifetime
	Type      reflect.Type
	Resolving bool       // Used for circular dependency detection
	mutex     sync.Mutex // Separate mutex for resolving flag
}

// Container defines the interface for dependency injection container
type Container interface {
	// Register registers a dependency with the container
	Register(name string, factory func() (interface{}, error), lifetime DependencyLifetime) error

	// Resolve resolves a dependency by name
	Resolve(name string) (interface{}, error)

	// MustResolve resolves a dependency by name or panics if it fails
	MustResolve(name string) interface{}

	// IsRegistered checks if a dependency is registered
	IsRegistered(name string) bool

	// GetLifetime returns the lifetime of a registered dependency
	GetLifetime(name string) (DependencyLifetime, error)
}

// DIContainer implements the Container interface
type DIContainer struct {
	dependencies map[string]*Dependency
	mutex        sync.RWMutex
}

// NewDIContainer creates a new dependency injection container
func NewDIContainer() *DIContainer {
	return &DIContainer{
		dependencies: make(map[string]*Dependency),
	}
}

// Register registers a dependency with the container
func (c *DIContainer) Register(name string, factory func() (interface{}, error), lifetime DependencyLifetime) error {
	if name == "" {
		return errors.NewValidationError("EMPTY_DEPENDENCY_NAME", "dependency name cannot be empty")
	}

	if factory == nil {
		return errors.NewValidationError("NIL_FACTORY_FUNCTION", "factory function cannot be nil")
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Check if already registered
	if _, exists := c.dependencies[name]; exists {
		return errors.NewSystemError("DEPENDENCY_ALREADY_REGISTERED", fmt.Sprintf("dependency '%s' is already registered", name))
	}

	// Create a sample instance to get the type
	sample, err := factory()
	if err != nil {
		return errors.NewSystemError("SAMPLE_INSTANCE_CREATION_FAILED", fmt.Sprintf("failed to create sample instance for '%s': %v", name, err))
	}

	dependency := &Dependency{
		Factory:  factory,
		Lifetime: lifetime,
		Type:     reflect.TypeOf(sample),
	}

	// For singleton, create the instance immediately
	if lifetime == Singleton {
		dependency.Instance = sample
	}

	c.dependencies[name] = dependency
	return nil
}

// Resolve resolves a dependency by name
func (c *DIContainer) Resolve(name string) (interface{}, error) {
	c.mutex.RLock()
	dependency, exists := c.dependencies[name]
	c.mutex.RUnlock()

	if !exists {
		return nil, errors.NewSystemError("DEPENDENCY_NOT_REGISTERED", fmt.Sprintf("dependency '%s' is not registered", name))
	}

	// Check for circular dependencies and set resolving flag with separate mutex
	dependency.mutex.Lock()
	if dependency.Resolving {
		// For singleton services, allow concurrent resolves but wait for the first one to complete
		if dependency.Lifetime == Singleton && dependency.Instance != nil {
			dependency.mutex.Unlock()
			// Instance is already created, return it
			return dependency.Instance, nil
		}
		dependency.mutex.Unlock()
		return nil, errors.NewSystemError("CIRCULAR_DEPENDENCY", fmt.Sprintf("circular dependency detected for '%s'", name))
	}
	dependency.Resolving = true
	dependency.mutex.Unlock()

	defer func() {
		dependency.mutex.Lock()
		dependency.Resolving = false
		dependency.mutex.Unlock()
	}()

	// Return existing instance for singleton
	if dependency.Lifetime == Singleton && dependency.Instance != nil {
		return dependency.Instance, nil
	}

	// Create new instance
	instance, err := dependency.Factory()
	if err != nil {
		return nil, errors.NewSystemError("INSTANCE_CREATION_FAILED", fmt.Sprintf("failed to create instance for '%s': %v", name, err))
	}

	// Store instance for singleton
	if dependency.Lifetime == Singleton {
		c.mutex.Lock()
		dependency.Instance = instance
		c.mutex.Unlock()
	}

	return instance, nil
}

// MustResolve resolves a dependency by name or panics if it fails
func (c *DIContainer) MustResolve(name string) interface{} {
	instance, err := c.Resolve(name)
	if err != nil {
		panic(errors.NewSystemError("DEPENDENCY_RESOLUTION_FAILED", fmt.Sprintf("failed to resolve dependency '%s': %v", name, err)).Error())
	}
	return instance
}

// IsRegistered checks if a dependency is registered
func (c *DIContainer) IsRegistered(name string) bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	_, exists := c.dependencies[name]
	return exists
}

// GetLifetime returns the lifetime of a registered dependency
func (c *DIContainer) GetLifetime(name string) (DependencyLifetime, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	dependency, exists := c.dependencies[name]
	if !exists {
		return Transient, errors.NewSystemError("DEPENDENCY_NOT_REGISTERED", fmt.Sprintf("dependency '%s' is not registered", name))
	}

	return dependency.Lifetime, nil
}

// GetType returns the type of a registered dependency
func (c *DIContainer) GetType(name string) (reflect.Type, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	dependency, exists := c.dependencies[name]
	if !exists {
		return nil, errors.NewSystemError("DEPENDENCY_NOT_REGISTERED", fmt.Sprintf("dependency '%s' is not registered", name))
	}

	return dependency.Type, nil
}

// ListDependencies returns the names of all registered dependencies
func (c *DIContainer) ListDependencies() []string {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	names := make([]string, 0, len(c.dependencies))
	for name := range c.dependencies {
		names = append(names, name)
	}

	return names
}

// Clear removes all registered dependencies
func (c *DIContainer) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.dependencies = make(map[string]*Dependency)
}
