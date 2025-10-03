package lua

import (
	"encoding/json"
	"fmt"

	lua "github.com/yuin/gopher-lua"
)

// JSONModule implements the LuaModule interface for JSON functionality
type JSONModule struct{}

// Name returns the module name
func (m *JSONModule) Name() string {
	return "json"
}

// Register registers the JSON module functions in the Lua state
func (m *JSONModule) Register(L *lua.LState) error {
	// Create the JSON module table
	jsonModule := L.NewTable()

	// Register functions
	L.SetField(jsonModule, "encode", L.NewFunction(m.encode))
	L.SetField(jsonModule, "decode", L.NewFunction(m.decode))

	// Register the module globally
	L.SetGlobal("json", jsonModule)

	return nil
}

// encode converts a Lua table to a JSON string
func (m *JSONModule) encode(L *lua.LState) int {
	value := L.CheckAny(1)

	// Convert Lua value to Go value
	goValue, err := m.luaToGo(value)
	if err != nil {
		L.RaiseError("JSON encode error: %v", err)
		return 0
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(goValue)
	if err != nil {
		L.RaiseError("JSON encode error: %v", err)
		return 0
	}

	// Return JSON string
	L.Push(lua.LString(string(jsonData)))
	return 1
}

// decode converts a JSON string to a Lua table
func (m *JSONModule) decode(L *lua.LState) int {
	jsonStr := L.CheckString(1)

	// Unmarshal from JSON
	var goValue interface{}
	if err := json.Unmarshal([]byte(jsonStr), &goValue); err != nil {
		L.RaiseError("JSON decode error: %v", err)
		return 0
	}

	// Convert Go value to Lua value
	luaValue, err := m.goToLua(L, goValue)
	if err != nil {
		L.RaiseError("JSON decode error: %v", err)
		return 0
	}

	// Return Lua table
	L.Push(luaValue)
	return 1
}

// luaToGo converts a Lua value to a Go value for JSON encoding
func (m *JSONModule) luaToGo(value lua.LValue) (interface{}, error) {
	switch value.Type() {
	case lua.LTString:
		return value.String(), nil
	case lua.LTNumber:
		return float64(value.(lua.LNumber)), nil
	case lua.LTBool:
		return bool(value.(lua.LBool)), nil
	case lua.LTNil:
		return nil, nil
	case lua.LTTable:
		return m.luaTableToGo(value.(*lua.LTable))
	default:
		return nil, fmt.Errorf("unsupported Lua type for JSON: %s", value.Type().String())
	}
}

// luaTableToGo converts a Lua table to a Go value for JSON encoding
func (m *JSONModule) luaTableToGo(table *lua.LTable) (interface{}, error) {
	// Check if it's an array-like table (sequential integer keys)
	isArray := true
	maxIndex := 0

	table.ForEach(func(key, val lua.LValue) {
		if key.Type() == lua.LTNumber {
			index := int(key.(lua.LNumber))
			if index > maxIndex {
				maxIndex = index
			}
		} else {
			isArray = false
		}
	})

	if isArray && maxIndex > 0 {
		// Convert to array
		result := make([]interface{}, maxIndex)
		for i := 1; i <= maxIndex; i++ {
			val := table.RawGetInt(i)
			if val.Type() != lua.LTNil {
				goVal, err := m.luaToGo(val)
				if err != nil {
					return nil, err
				}
				result[i-1] = goVal
			} else {
				result[i-1] = nil
			}
		}
		return result, nil
	} else {
		// Convert to map
		result := make(map[string]interface{})
		table.ForEach(func(key, val lua.LValue) {
			var keyStr string
			switch key.Type() {
			case lua.LTString:
				keyStr = key.String()
			case lua.LTNumber:
				num := float64(key.(lua.LNumber))
				if num == float64(int64(num)) && num >= -9223372036854775808 && num <= 9223372036854775807 {
					keyStr = fmt.Sprintf("%.0f", num)
				} else {
					keyStr = fmt.Sprintf("%v", num)
				}
			default:
				// Skip unsupported key types
				return
			}

			goVal, err := m.luaToGo(val)
			if err != nil {
				return
			}
			result[keyStr] = goVal
		})
		return result, nil
	}
}

// goToLua converts a Go value to a Lua value for JSON decoding
func (m *JSONModule) goToLua(L *lua.LState, value interface{}) (lua.LValue, error) {
	switch v := value.(type) {
	case string:
		return lua.LString(v), nil
	case float64:
		return lua.LNumber(v), nil
	case bool:
		return lua.LBool(v), nil
	case nil:
		return lua.LNil, nil
	case []interface{}:
		table := L.NewTable()
		for i, item := range v {
			luaItem, err := m.goToLua(L, item)
			if err != nil {
				return nil, err
			}
			table.RawSetInt(i+1, luaItem) // Lua arrays are 1-based
		}
		return table, nil
	case map[string]interface{}:
		table := L.NewTable()
		for key, item := range v {
			luaItem, err := m.goToLua(L, item)
			if err != nil {
				return nil, err
			}
			table.RawSetString(key, luaItem)
		}
		return table, nil
	default:
		return nil, fmt.Errorf("unsupported Go type for JSON: %T", value)
	}
}

// Ensure JSONModule implements the LuaModule interface
var _ LuaModule = (*JSONModule)(nil)
