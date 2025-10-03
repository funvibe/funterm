package lua

import (
	"funterm/errors"
)

// GetFFIEnhancer returns the FFI enhancer instance
func (lr *LuaRuntime) GetFFIEnhancer() *FFIEnhancer {
	lr.mu.Lock()
	defer lr.mu.Unlock()
	return lr.ffiEnhancer
}

// GetFFIInformation returns information about FFI capabilities
func (lr *LuaRuntime) GetFFIInformation() map[string]interface{} {
	lr.mu.Lock()
	defer lr.mu.Unlock()

	if lr.ffiEnhancer == nil {
		return map[string]interface{}{
			"available": false,
			"message":   "FFI enhancer not initialized",
		}
	}

	capabilities := lr.ffiEnhancer.GetCapabilities()
	capabilities["available"] = true
	return capabilities
}

// GetFFIFunctions returns available FFI functions
func (lr *LuaRuntime) GetFFIFunctions() map[string]*FFIFunction {
	lr.mu.Lock()
	defer lr.mu.Unlock()

	if lr.ffiEnhancer == nil {
		return make(map[string]*FFIFunction)
	}

	return lr.ffiEnhancer.ListFunctions()
}

// GetFFITypes returns available FFI types
func (lr *LuaRuntime) GetFFITypes() map[string]FFIType {
	lr.mu.Lock()
	defer lr.mu.Unlock()

	if lr.ffiEnhancer == nil {
		return make(map[string]FFIType)
	}

	return lr.ffiEnhancer.ListTypes()
}

// AddCustomFFIFunction adds a custom FFI function definition
func (lr *LuaRuntime) AddCustomFFIFunction(funcName, library, returnType string, paramTypes []string, description string) error {
	lr.mu.Lock()
	defer lr.mu.Unlock()

	if lr.ffiEnhancer == nil {
		return errors.NewRuntimeError("lua", "LUA_FFI_NOT_INITIALIZED", "FFI enhancer not initialized")
	}

	lr.ffiEnhancer.AddCustomFunction(funcName, library, returnType, paramTypes, description)
	return nil
}

// AddCustomFFIType adds a custom FFI type definition
func (lr *LuaRuntime) AddCustomFFIType(name string, size, alignment int, description string) error {
	lr.mu.Lock()
	defer lr.mu.Unlock()

	if lr.ffiEnhancer == nil {
		return errors.NewRuntimeError("lua", "LUA_FFI_NOT_INITIALIZED", "FFI enhancer not initialized")
	}

	lr.ffiEnhancer.AddCustomType(name, size, alignment, description)
	return nil
}

// GenerateFFIBindings generates Lua bindings for C functions
func (lr *LuaRuntime) GenerateFFIBindings() (string, error) {
	lr.mu.Lock()
	defer lr.mu.Unlock()

	if lr.ffiEnhancer == nil {
		return "", errors.NewRuntimeError("lua", "LUA_FFI_NOT_INITIALIZED", "FFI enhancer not initialized")
	}

	return lr.ffiEnhancer.GenerateBindings(), nil
}

// GetFFIDocumentation returns comprehensive FFI documentation
func (lr *LuaRuntime) GetFFIDocumentation() (string, error) {
	lr.mu.Lock()
	defer lr.mu.Unlock()

	if lr.ffiEnhancer == nil {
		return "", errors.NewRuntimeError("lua", "LUA_FFI_NOT_INITIALIZED", "FFI enhancer not initialized")
	}

	return lr.ffiEnhancer.GetDocumentation(), nil
}

// ValidateFFIDefinition validates an FFI definition
func (lr *LuaRuntime) ValidateFFIDefinition(definition string) error {
	lr.mu.Lock()
	defer lr.mu.Unlock()

	if lr.ffiEnhancer == nil {
		return errors.NewRuntimeError("lua", "LUA_FFI_NOT_INITIALIZED", "FFI enhancer not initialized")
	}

	return lr.ffiEnhancer.ValidateDefinition(definition)
}
