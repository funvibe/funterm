package engine

import (
	"fmt"
	"sync"

	"funterm/runtime"

	"github.com/stretchr/testify/require"
)

// StatefulMockRuntime is a mock runtime that can store variables and track function calls.
type StatefulMockRuntime struct {
	name  string
	ready bool
	vars  map[string]interface{}
	calls []string
	mu    sync.RWMutex

	// ExecuteFunctionFunc allows overriding the mock's behavior for specific tests.
	ExecuteFunctionFunc func(name string, args []interface{}) (interface{}, error)

	// Default implementation for unused methods
	runtime.LanguageRuntime
}

func NewStatefulMockRuntime(name string) *StatefulMockRuntime {
	return &StatefulMockRuntime{
		name:  name,
		ready: true,
		vars:  make(map[string]interface{}),
		calls: make([]string, 0),
	}
}

func (m *StatefulMockRuntime) GetName() string {
	return m.name
}

func (m *StatefulMockRuntime) IsReady() bool {
	return true
}

func (m *StatefulMockRuntime) Initialize() error {
	m.ready = true
	return nil
}

func (m *StatefulMockRuntime) SetVariable(name string, value interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.vars[name] = value
	return nil
}

func (m *StatefulMockRuntime) GetVariable(name string) (interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	val, ok := m.vars[name]
	if !ok {
		return nil, fmt.Errorf("variable '%s' not found", name)
	}
	return val, nil
}

func (m *StatefulMockRuntime) ExecuteFunction(name string, args []interface{}) (interface{}, error) {
	// If a custom function is provided for the test, use it.
	if m.ExecuteFunctionFunc != nil {
		return m.ExecuteFunctionFunc(name, args)
	}

	// Default behavior: just record the call.
	m.mu.Lock()
	defer m.mu.Unlock()
	callSignature := fmt.Sprintf("%s(%v)", name, args)
	m.calls = append(m.calls, callSignature)
	return nil, nil
}

func (m *StatefulMockRuntime) GetCalls() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	// Return a copy
	callsCopy := make([]string, len(m.calls))
	copy(callsCopy, m.calls)
	return callsCopy
}

// CreateEngineWithStatefulMock is a helper to create an engine with a stateful mock runtime.
func CreateEngineWithStatefulMock(t interface {
	Helper()
	Errorf(format string, args ...interface{})
	FailNow()
}) (*ExecutionEngine, *StatefulMockRuntime) {
	if h, ok := t.(interface{ Helper() }); ok {
		h.Helper()
	}
	eng, err := NewExecutionEngine()
	require.NoError(t, err)

	mockRuntime := NewStatefulMockRuntime("lua")
	err = eng.RegisterRuntime(mockRuntime)
	require.NoError(t, err)

	return eng, mockRuntime
}
