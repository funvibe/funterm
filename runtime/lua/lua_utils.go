package lua

import (
	"fmt"

	lua "github.com/yuin/gopher-lua"
)

// TODO: maybe useless
// luaValueToString converts a Lua value to string representation
func (lr *LuaRuntime) luaValueToString(value lua.LValue) string {
	switch value.Type() {
	case lua.LTString:
		return value.String()
	case lua.LTNumber:
		num := float64(value.(lua.LNumber))
		// Check if it's a whole number (integer)
		if num == float64(int64(num)) && num >= -9223372036854775808 && num <= 9223372036854775807 {
			// Format as integer to avoid scientific notation
			return fmt.Sprintf("%.0f", num)
		}
		return fmt.Sprintf("%v", num)
	case lua.LTBool:
		return fmt.Sprintf("%v", bool(value.(lua.LBool)))
	case lua.LTNil:
		return "nil"
	case lua.LTTable:
		return fmt.Sprintf("<table: %p>", value)
	case lua.LTFunction:
		return fmt.Sprintf("<function: %p>", value)
	default:
		return fmt.Sprintf("<%s: %v>", value.Type().String(), value)
	}
}
