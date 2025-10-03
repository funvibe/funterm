package lua

import (
	"fmt"
	"sync"

	lua "github.com/yuin/gopher-lua"
)

// LuaModule defines the interface for built-in Lua modules
type LuaModule interface {
	// Name returns the module name (used in require() calls)
	Name() string

	// Register registers the module functions in the Lua state
	Register(L *lua.LState) error
}

// ModuleManager manages built-in Lua modules
type ModuleManager struct {
	modules map[string]LuaModule
	mu      sync.RWMutex
}

// NewModuleManager creates a new ModuleManager
func NewModuleManager() *ModuleManager {
	return &ModuleManager{
		modules: make(map[string]LuaModule),
	}
}

// RegisterModule registers a built-in module
func (mm *ModuleManager) RegisterModule(module LuaModule) error {
	if module == nil {
		return fmt.Errorf("module cannot be nil")
	}

	name := module.Name()
	if name == "" {
		return fmt.Errorf("module name cannot be empty")
	}

	mm.mu.Lock()
	defer mm.mu.Unlock()

	if _, exists := mm.modules[name]; exists {
		return fmt.Errorf("module '%s' is already registered", name)
	}

	mm.modules[name] = module
	return nil
}

// GetModule returns a registered module by name
func (mm *ModuleManager) GetModule(name string) (LuaModule, bool) {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	module, exists := mm.modules[name]
	return module, exists
}

// ListModules returns all registered module names
func (mm *ModuleManager) ListModules() []string {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	names := make([]string, 0, len(mm.modules))
	for name := range mm.modules {
		names = append(names, name)
	}

	return names
}

// RegisterAllModules registers all built-in modules in the Lua state
// and sets up package.preload for require() support
func (mm *ModuleManager) RegisterAllModules(L *lua.LState) error {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	// Get the package.preload table
	packageTable := L.GetGlobal("package")
	if packageTable.Type() != lua.LTTable {
		return fmt.Errorf("package table not found")
	}

	preloadTable := packageTable.(*lua.LTable).RawGetString("preload")
	if preloadTable.Type() != lua.LTTable {
		return fmt.Errorf("package.preload table not found")
	}

	// Register each module
	for name, module := range mm.modules {
		// Register the module functions globally first
		if err := module.Register(L); err != nil {
			return fmt.Errorf("failed to register module '%s': %v", name, err)
		}

		// Create a Lua function that will load the module when called via require()
		moduleLoader := L.NewFunction(func(L *lua.LState) int {
			// Get the already registered global module table
			moduleTable := L.GetGlobal(name)
			if moduleTable.Type() != lua.LTTable {
				L.RaiseError("module '%s' did not create a table", name)
				return 0
			}

			L.Push(moduleTable)
			return 1
		})

		// Add the loader to package.preload
		preloadTable.(*lua.LTable).RawSetString(name, moduleLoader)
	}

	return nil
}

// IsModuleRegistered checks if a module is registered
func (mm *ModuleManager) IsModuleRegistered(name string) bool {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	_, exists := mm.modules[name]
	return exists
}
