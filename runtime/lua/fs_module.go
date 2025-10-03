package lua

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	lua "github.com/yuin/gopher-lua"
)

// FSModule implements the LuaModule interface for filesystem functionality
type FSModule struct {
	baseDir string // Base directory for security restrictions
}

// Name returns the module name
func (m *FSModule) Name() string {
	return "fs"
}

// NewFSModule creates a new FSModule with the specified base directory
func NewFSModule(baseDir string) *FSModule {
	return &FSModule{
		baseDir: baseDir,
	}
}

// Register registers the filesystem module functions in the Lua state
func (m *FSModule) Register(L *lua.LState) error {
	// Create the filesystem module table
	fsModule := L.NewTable()

	// Register functions
	L.SetField(fsModule, "exists", L.NewFunction(m.exists))
	L.SetField(fsModule, "read", L.NewFunction(m.read))
	L.SetField(fsModule, "write", L.NewFunction(m.write))
	L.SetField(fsModule, "list", L.NewFunction(m.list))

	// Register the module globally
	L.SetGlobal("fs", fsModule)

	return nil
}

// exists checks if a file or directory exists
func (m *FSModule) exists(L *lua.LState) int {
	path := L.CheckString(1)

	// Security check - ensure path is within base directory
	fullPath, err := m.securePath(path)
	if err != nil {
		L.Push(lua.LBool(false))
		L.Push(lua.LString(fmt.Sprintf("security error: %v", err)))
		return 2
	}

	// Check if path exists
	_, err = os.Stat(fullPath)
	exists := err == nil

	L.Push(lua.LBool(exists))
	L.Push(lua.LNil)
	return 2
}

// read reads the contents of a file
func (m *FSModule) read(L *lua.LState) int {
	path := L.CheckString(1)

	// Security check - ensure path is within base directory
	fullPath, err := m.securePath(path)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(fmt.Sprintf("security error: %v", err)))
		return 2
	}

	// Read file contents
	content, err := ioutil.ReadFile(fullPath)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(fmt.Sprintf("read error: %v", err)))
		return 2
	}

	L.Push(lua.LString(string(content)))
	L.Push(lua.LNil)
	return 2
}

// write writes content to a file
func (m *FSModule) write(L *lua.LState) int {
	path := L.CheckString(1)
	content := L.CheckString(2)

	// Security check - ensure path is within base directory
	fullPath, err := m.securePath(path)
	if err != nil {
		L.Push(lua.LBool(false))
		L.Push(lua.LString(fmt.Sprintf("security error: %v", err)))
		return 2
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		L.Push(lua.LBool(false))
		L.Push(lua.LString(fmt.Sprintf("directory creation error: %v", err)))
		return 2
	}

	// Write content to file
	err = ioutil.WriteFile(fullPath, []byte(content), 0644)
	if err != nil {
		L.Push(lua.LBool(false))
		L.Push(lua.LString(fmt.Sprintf("write error: %v", err)))
		return 2
	}

	L.Push(lua.LBool(true))
	L.Push(lua.LNil)
	return 2
}

// list lists files and directories in the specified path
func (m *FSModule) list(L *lua.LState) int {
	path := L.CheckString(1)

	// Security check - ensure path is within base directory
	fullPath, err := m.securePath(path)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(fmt.Sprintf("security error: %v", err)))
		return 2
	}

	// Read directory contents
	files, err := ioutil.ReadDir(fullPath)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(fmt.Sprintf("list error: %v", err)))
		return 2
	}

	// Create result table
	resultTable := L.NewTable()
	for i, file := range files {
		fileInfo := L.NewTable()
		L.SetField(fileInfo, "name", lua.LString(file.Name()))
		L.SetField(fileInfo, "is_dir", lua.LBool(file.IsDir()))
		L.SetField(fileInfo, "size", lua.LNumber(file.Size()))

		L.RawSetInt(resultTable, i+1, fileInfo) // Lua arrays are 1-based
	}

	L.Push(resultTable)
	L.Push(lua.LNil)
	return 2
}

// securePath ensures the path is within the base directory and returns the full path
func (m *FSModule) securePath(path string) (string, error) {
	// Clean the path to remove any ".." or "."
	cleanPath := filepath.Clean(path)

	// If path is absolute, check if it's within base directory
	if filepath.IsAbs(cleanPath) {
		relPath, err := filepath.Rel(m.baseDir, cleanPath)
		if err != nil {
			return "", fmt.Errorf("path is outside allowed directory")
		}
		if strings.HasPrefix(relPath, "..") {
			return "", fmt.Errorf("path is outside allowed directory")
		}
		return cleanPath, nil
	}

	// For relative paths, join with base directory
	fullPath := filepath.Join(m.baseDir, cleanPath)

	// Verify the final path is still within base directory
	relPath, err := filepath.Rel(m.baseDir, fullPath)
	if err != nil {
		return "", fmt.Errorf("path is outside allowed directory")
	}
	if strings.HasPrefix(relPath, "..") {
		return "", fmt.Errorf("path is outside allowed directory")
	}

	return fullPath, nil
}

// Ensure FSModule implements the LuaModule interface
var _ LuaModule = (*FSModule)(nil)
