package lua

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	lua "github.com/yuin/gopher-lua"
)

// HTTPModule implements the LuaModule interface for HTTP functionality
type HTTPModule struct{}

// Name returns the module name
func (m *HTTPModule) Name() string {
	return "http"
}

// Register registers the HTTP module functions in the Lua state
func (m *HTTPModule) Register(L *lua.LState) error {
	// Create the HTTP module table
	httpModule := L.NewTable()

	// Register functions
	L.SetField(httpModule, "get", L.NewFunction(m.get))
	L.SetField(httpModule, "post", L.NewFunction(m.post))

	// Register the module globally
	L.SetGlobal("http", httpModule)

	return nil
}

// get performs an HTTP GET request
func (m *HTTPModule) get(L *lua.LState) int {
	url := L.CheckString(1)

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Perform GET request
	resp, err := client.Get(url)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(fmt.Sprintf("HTTP GET error: %v", err)))
		return 2
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(fmt.Sprintf("HTTP response read error: %v", err)))
		return 2
	}

	// Create response table
	responseTable := L.NewTable()
	L.SetField(responseTable, "status", lua.LNumber(resp.StatusCode))
	L.SetField(responseTable, "body", lua.LString(string(body)))
	L.SetField(responseTable, "headers", m.convertHeadersToLua(L, resp.Header))

	// Return response table and nil error
	L.Push(responseTable)
	L.Push(lua.LNil)
	return 2
}

// post performs an HTTP POST request
func (m *HTTPModule) post(L *lua.LState) int {
	url := L.CheckString(1)
	data := L.CheckAny(2)

	// Convert data to string
	var dataStr string
	switch data.Type() {
	case lua.LTString:
		dataStr = data.String()
	case lua.LTTable:
		// Convert table to JSON-like string
		dataStr = m.convertTableToQueryString(data.(*lua.LTable))
	default:
		L.Push(lua.LNil)
		L.Push(lua.LString("HTTP POST error: data must be string or table"))
		return 2
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Perform POST request
	resp, err := client.Post(url, "application/x-www-form-urlencoded", strings.NewReader(dataStr))
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(fmt.Sprintf("HTTP POST error: %v", err)))
		return 2
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(fmt.Sprintf("HTTP response read error: %v", err)))
		return 2
	}

	// Create response table
	responseTable := L.NewTable()
	L.SetField(responseTable, "status", lua.LNumber(resp.StatusCode))
	L.SetField(responseTable, "body", lua.LString(string(body)))
	L.SetField(responseTable, "headers", m.convertHeadersToLua(L, resp.Header))

	// Return response table and nil error
	L.Push(responseTable)
	L.Push(lua.LNil)
	return 2
}

// convertHeadersToLua converts HTTP headers to a Lua table
func (m *HTTPModule) convertHeadersToLua(L *lua.LState, headers http.Header) lua.LValue {
	headerTable := L.NewTable()

	for key, values := range headers {
		if len(values) == 1 {
			L.SetField(headerTable, key, lua.LString(values[0]))
		} else {
			// Handle multiple values for the same header
			valueArray := L.NewTable()
			for i, value := range values {
				L.RawSetInt(valueArray, i+1, lua.LString(value))
			}
			L.SetField(headerTable, key, valueArray)
		}
	}

	return headerTable
}

// convertTableToQueryString converts a Lua table to a query string
func (m *HTTPModule) convertTableToQueryString(table *lua.LTable) string {
	var buf bytes.Buffer
	first := true

	table.ForEach(func(key, val lua.LValue) {
		if !first {
			buf.WriteByte('&')
		}
		first = false

		// Convert key to string
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
			return // Skip unsupported key types
		}

		// Convert value to string
		var valStr string
		switch val.Type() {
		case lua.LTString:
			valStr = val.String()
		case lua.LTNumber:
			num := float64(val.(lua.LNumber))
			if num == float64(int64(num)) && num >= -9223372036854775808 && num <= 9223372036854775807 {
				valStr = fmt.Sprintf("%.0f", num)
			} else {
				valStr = fmt.Sprintf("%v", num)
			}
		case lua.LTBool:
			valStr = fmt.Sprintf("%v", bool(val.(lua.LBool)))
		default:
			return // Skip unsupported value types
		}

		buf.WriteString(keyStr)
		buf.WriteByte('=')
		buf.WriteString(valStr)
	})

	return buf.String()
}

// Ensure HTTPModule implements the LuaModule interface
var _ LuaModule = (*HTTPModule)(nil)
