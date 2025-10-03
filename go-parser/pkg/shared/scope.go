package shared

import (
	"sync"
)

// Scope represents a variable scope that stores variable bindings
// This is used for local variable scoping in constructs like loops and pattern matching
type Scope struct {
	parent    *Scope
	variables map[string]interface{}
	mutex     sync.RWMutex
}

// NewScope creates a new scope with an optional parent scope
func NewScope(parent *Scope) *Scope {
	return &Scope{
		parent:    parent,
		variables: make(map[string]interface{}),
	}
}

// Get retrieves a variable value from the scope
// It searches the current scope and all parent scopes
func (s *Scope) Get(name string) (interface{}, bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	// Check current scope first
	if value, exists := s.variables[name]; exists {
		return value, true
	}

	// If not found and we have a parent scope, check parent
	if s.parent != nil {
		return s.parent.Get(name)
	}

	// Not found in any scope
	return nil, false
}

// Set sets a variable value in the current scope
func (s *Scope) Set(name string, value interface{}) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.variables[name] = value
}

// Delete removes a variable from the current scope
func (s *Scope) Delete(name string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	delete(s.variables, name)
}

// Has checks if a variable exists in the current scope (not parent scopes)
func (s *Scope) Has(name string) bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	_, exists := s.variables[name]
	return exists
}

// Clear removes all variables from the current scope
func (s *Scope) Clear() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.variables = make(map[string]interface{})
}

// GetAll returns all variables in the current scope (not parent scopes)
func (s *Scope) GetAll() map[string]interface{} {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	// Return a copy to avoid race conditions
	result := make(map[string]interface{})
	for k, v := range s.variables {
		result[k] = v
	}
	return result
}

// GetParent returns the parent scope
func (s *Scope) GetParent() *Scope {
	return s.parent
}

// IsRoot checks if this is a root scope (no parent)
func (s *Scope) IsRoot() bool {
	return s.parent == nil
}
