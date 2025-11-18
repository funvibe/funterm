package shared

import (
	"sync"
)

// VariableInfo хранит информацию о переменной включая ее значение и флаг изменяемости
type VariableInfo struct {
	Value     interface{}
	IsMutable bool
}

// Scope represents a variable scope that stores variable bindings
// This is used for local variable scoping in constructs like loops and pattern matching
type Scope struct {
	parent    *Scope
	variables map[string]*VariableInfo
	mutex     sync.RWMutex
}

// NewScope creates a new scope with an optional parent scope
func NewScope(parent *Scope) *Scope {
	return &Scope{
		parent:    parent,
		variables: make(map[string]*VariableInfo),
	}
}

// Get retrieves a variable value from the scope
// It searches the current scope and all parent scopes
func (s *Scope) Get(name string) (interface{}, bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	// Check current scope first
	if varInfo, exists := s.variables[name]; exists {
		return varInfo.Value, true
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

	s.variables[name] = &VariableInfo{
		Value:     value,
		IsMutable: true, // По умолчанию переменные изменяемые для обратной совместимости
	}
}

// Delete removes a variable from the current scope
func (s *Scope) Delete(name string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	delete(s.variables, name)
}

// SetWithMutability sets a variable value with explicit mutability flag in the current scope
func (s *Scope) SetWithMutability(name string, value interface{}, isMutable bool) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.variables[name] = &VariableInfo{
		Value:     value,
		IsMutable: isMutable,
	}
}

// GetVariableInfo retrieves complete variable information from the scope
// It searches the current scope and all parent scopes
func (s *Scope) GetVariableInfo(name string) (*VariableInfo, bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	// Check current scope first
	if varInfo, exists := s.variables[name]; exists {
		return varInfo, true
	}

	// If not found and we have a parent scope, check parent
	if s.parent != nil {
		return s.parent.GetVariableInfo(name)
	}

	// Not found in any scope
	return nil, false
}

// GetVariableInfoLocal retrieves complete variable information from the current scope only
// Does NOT search parent scopes
func (s *Scope) GetVariableInfoLocal(name string) (*VariableInfo, bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	// Check only current scope
	if varInfo, exists := s.variables[name]; exists {
		return varInfo, true
	}

	// Not found in current scope
	return nil, false
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

	s.variables = make(map[string]*VariableInfo)
}

// GetAll returns all variables in the current scope (not parent scopes)
func (s *Scope) GetAll() map[string]interface{} {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	// Return a copy to avoid race conditions
	result := make(map[string]interface{})
	for k, varInfo := range s.variables {
		result[k] = varInfo.Value
	}
	return result
}

// GetAllWithInfo returns all variables with their mutability info in the current scope (not parent scopes)
func (s *Scope) GetAllWithInfo() map[string]*VariableInfo {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	// Return a copy to avoid race conditions
	result := make(map[string]*VariableInfo)
	for k, varInfo := range s.variables {
		result[k] = &VariableInfo{
			Value:     varInfo.Value,
			IsMutable: varInfo.IsMutable,
		}
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
