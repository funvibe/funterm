package engine

import (
	"fmt"

	"funterm/errors"
	"funterm/shared"
	"go-parser/pkg/ast"
	sharedparser "go-parser/pkg/shared"

	"github.com/funvibe/funbit/pkg/funbit"
)

// executeMatchStatement executes a match statement with pattern matching
func (e *ExecutionEngine) executeMatchStatement(matchStmt *ast.MatchStatement) (interface{}, error) {
	// Evaluate the expression to be matched
	var subject interface{}
	var err error

	// Handle the match expression based on its type
	switch expr := matchStmt.Expression.(type) {
	case *ast.Identifier:
		// For identifiers in match expressions, we need to handle them specially
		// to ensure they're treated as variable reads, not function calls
		if expr.Qualified {
			// Qualified identifier like py.response - create a VariableRead
			varRead := &ast.VariableRead{
				Variable: expr,
			}
			subject, err = e.executeVariableRead(varRead)
		} else {
			// Simple identifier - use convertExpressionToValue to check local scope first
			subject, err = e.convertExpressionToValue(expr)
		}
	case *ast.VariableRead:
		// If it's already a VariableRead, execute it directly
		subject, err = e.executeVariableRead(expr)
	case *ast.FieldAccess:
		// For field access expressions like lua.data.name
		subject, err = e.executeFieldAccess(expr)
	case *ast.LanguageCall:
		// For language calls, check if they're actually variable references
		// This can happen when the parser misidentifies a variable as a function call
		if len(expr.Arguments) == 0 {
			// This might be a variable reference, try to read it as a variable
			languages := e.ListAvailableLanguages()
			for _, lang := range languages {
				// Handle aliases: js -> node, py -> python
				targetLang := expr.Language
				if expr.Language == "js" {
					targetLang = "node"
				} else if expr.Language == "py" {
					targetLang = "python"
				}

				if lang == targetLang {
					rt, runtimeErr := e.runtimeManager.GetRuntime(lang)
					if runtimeErr == nil && rt.IsReady() {
						subject, err = rt.GetVariable(expr.Function)
						if err == nil {
							break
						}
					}
				}
			}
			// If we couldn't get it as a variable, fall back to normal execution
			if err != nil {
				subject, err = e.convertExpressionToValue(expr)
			}
		} else {
			// It's a real function call with arguments
			subject, err = e.convertExpressionToValue(expr)
		}
	default:
		// For other expression types, use the standard conversion
		subject, err = e.convertExpressionToValue(expr)
	}

	if err != nil {
		return nil, errors.NewSystemError("MATCH_EXPRESSION_EVALUATION_ERROR", fmt.Sprintf("failed to evaluate match expression: %v", err))
	}

	// Iterate through match arms
	for _, arm := range matchStmt.Arms {
		// Try to match the pattern
		if matches, bindings := e.matchesPattern(arm.Pattern, subject); matches {
			// Pattern matched, execute the body with any variable bindings
			if len(bindings) > 0 {
				// Create a temporary scope for bound variables
				return e.executeStatementWithBindings(arm.Statement, bindings)
			} else {
				// No bindings, but still create local scope for local variables
				return e.executeStatementWithLocalScope(arm.Statement)
			}
		}
	}

	// No pattern matched
	return nil, errors.NewUserError("NO_PATTERN_MATCH", "no pattern in match statement matched the value")
}

// matchesPattern checks if a pattern matches a value and returns any variable bindings
func (e *ExecutionEngine) matchesPattern(pattern ast.Pattern, value interface{}) (bool, map[string]interface{}) {
	switch p := pattern.(type) {
	case *ast.LiteralPattern:
		// Compare literal values
		return e.compareValues(p.Value, value), nil

	case *ast.VariablePattern:
		// Variable pattern always matches and binds the value
		return true, map[string]interface{}{p.Name: value}

	case *ast.WildcardPattern:
		// Wildcard always matches with no bindings
		return true, nil

	case *ast.ArrayPattern:
		// Array pattern matching
		return e.matchesArrayPattern(p, value)

	case *ast.ObjectPattern:
		// Object pattern matching
		return e.matchesObjectPattern(p, value)

	case *ast.BitstringPattern:
		// Bitstring pattern matching
		return e.matchesBitstringPattern(p, value)

	default:
		// Unsupported pattern type
		return false, nil
	}
}

// compareValues compares two values for equality
func (e *ExecutionEngine) compareValues(a, b interface{}) bool {
	if e.verbose {
		fmt.Printf("DEBUG: compareValues - a=%v (%T), b=%v (%T)\n", a, a, b, b)
	}

	// Handle nil values
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	// Handle string comparison
	if aStr, ok := a.(string); ok {
		if bStr, ok := b.(string); ok {
			return aStr == bStr
		}
	}

	// Handle numeric comparison (int, int64, float64, uint64)
	if aNum, ok := a.(float64); ok {
		if bNum, ok := b.(float64); ok {
			return aNum == bNum
		}
		if bNum, ok := b.(int); ok {
			return aNum == float64(bNum)
		}
		if bNum, ok := b.(int64); ok {
			return aNum == float64(bNum)
		}
		if bNum, ok := b.(uint64); ok {
			return aNum == float64(bNum)
		}
	}
	if aNum, ok := a.(int); ok {
		if bNum, ok := b.(int); ok {
			return aNum == bNum
		}
		if bNum, ok := b.(int64); ok {
			return int64(aNum) == bNum
		}
		if bNum, ok := b.(float64); ok {
			return float64(aNum) == bNum
		}
		if bNum, ok := b.(uint64); ok {
			return float64(aNum) == float64(bNum)
		}
	}
	if aNum, ok := a.(int64); ok {
		if bNum, ok := b.(int64); ok {
			return aNum == bNum
		}
		if bNum, ok := b.(int); ok {
			return aNum == int64(bNum)
		}
		if bNum, ok := b.(float64); ok {
			return float64(aNum) == bNum
		}
		if bNum, ok := b.(uint64); ok {
			return float64(aNum) == float64(bNum)
		}
	}
	if aNum, ok := a.(uint64); ok {
		if bNum, ok := b.(uint64); ok {
			return aNum == bNum
		}
		if bNum, ok := b.(float64); ok {
			return float64(aNum) == bNum
		}
		if bNum, ok := b.(int); ok {
			return float64(aNum) == float64(bNum)
		}
		if bNum, ok := b.(int64); ok {
			return float64(aNum) == float64(bNum)
		}
	}

	// For other types, use basic equality
	return a == b
}

// matchesArrayPattern checks if an array pattern matches a value and returns any variable bindings
func (e *ExecutionEngine) matchesArrayPattern(pattern *ast.ArrayPattern, value interface{}) (bool, map[string]interface{}) {
	// Check if the value is an array
	arrayValue, ok := value.([]interface{})
	if !ok {
		return false, nil
	}

	// Check if the pattern has the same number of elements
	if len(pattern.Elements) != len(arrayValue) {
		return false, nil
	}

	// Match each element
	bindings := make(map[string]interface{})
	for i, element := range pattern.Elements {
		matches, elementBindings := e.matchesPattern(element, arrayValue[i])
		if !matches {
			return false, nil
		}
		// Merge bindings
		for k, v := range elementBindings {
			bindings[k] = v
		}
	}

	return true, bindings
}

// matchesObjectPattern checks if an object pattern matches a value and returns any variable bindings
func (e *ExecutionEngine) matchesObjectPattern(pattern *ast.ObjectPattern, value interface{}) (bool, map[string]interface{}) {
	// Check if the value is an object
	objectValue, ok := value.(map[string]interface{})
	if !ok {
		return false, nil
	}

	// Special case: empty object pattern {} should only match empty objects
	if len(pattern.Properties) == 0 {
		return len(objectValue) == 0, nil
	}

	// Match each property
	bindings := make(map[string]interface{})
	for propertyName, propertyPattern := range pattern.Properties {
		// Special handling for wildcard key (_)
		if propertyName == "_" {
			// For wildcard key, we need to find at least one property in the object that matches the pattern
			foundMatch := false
			for _, objValue := range objectValue {
				// Try to match the pattern against this object property value
				if matches, propertyBindings := e.matchesPattern(propertyPattern, objValue); matches {
					foundMatch = true
					// Merge bindings
					for k, v := range propertyBindings {
						bindings[k] = v
					}
					// We found a match, no need to check other properties
					break
				}
			}
			if !foundMatch {
				return false, nil
			}
		} else {
			// Normal property matching
			// Check if the object has the property
			propertyValue, exists := objectValue[propertyName]
			if !exists {
				return false, nil
			}

			// Match the property value
			matches, propertyBindings := e.matchesPattern(propertyPattern, propertyValue)
			if !matches {
				return false, nil
			}
			// Merge bindings
			for k, v := range propertyBindings {
				bindings[k] = v
			}
		}
	}

	return true, bindings
}

// matchesBitstringPattern checks if a bitstring pattern matches a byte slice value and returns any variable bindings
func (e *ExecutionEngine) matchesBitstringPattern(pattern *ast.BitstringPattern, value interface{}) (bool, map[string]interface{}) {
	// Convert the value to BitstringObject if needed
	var bitstringData *shared.BitstringObject
	var ok bool

	switch v := value.(type) {
	case *shared.BitstringObject:
		bitstringData = v
		ok = true
	case []byte:
		// Convert byte slice to BitstringObject
		bitString := funbit.NewBitStringFromBytes(v)
		bitstringData = &shared.BitstringObject{BitString: bitString}
		ok = true
	case string:
		// Convert string to BitstringObject using UTF-8 encoding
		builder := funbit.NewBuilder()
		funbit.AddUTF8(builder, v)
		bitString, err := funbit.Build(builder)
		if err != nil {
			if e.verbose {
				fmt.Printf("DEBUG: matchesBitstringPattern - failed to build UTF-8 bitstring: %v\n", err)
			}
			return false, nil
		}
		bitstringData = &shared.BitstringObject{BitString: bitString}
		ok = true
	case shared.BitstringByte:
		// Convert BitstringByte to BitstringObject (single byte)
		bitString := funbit.NewBitStringFromBytes([]byte{v.Value})
		bitstringData = &shared.BitstringObject{BitString: bitString}
		ok = true
	default:
		if e.verbose {
			fmt.Printf("DEBUG: matchesBitstringPattern - unsupported type %T\n", value)
		}
		return false, nil
	}

	if !ok {
		return false, nil
	}

	if e.verbose {
		fmt.Printf("DEBUG: matchesBitstringPattern - using funbit adapter for pattern matching\n")
	}

	// Convert BitstringPattern to BitstringExpression for the adapter
	patternExpr := e.convertBitstringPatternToExpression(pattern)

	// Use funbit adapter for pattern matching
	adapter := NewFunbitAdapterWithEngine(e)
	bindings, err := adapter.MatchBitstringWithFunbit(patternExpr, bitstringData)
	if err != nil {
		if e.verbose {
			fmt.Printf("DEBUG: matchesBitstringPattern - funbit matching failed: %v\n", err)
		}
		return false, nil
	}

	if e.verbose {
		fmt.Printf("DEBUG: matchesBitstringPattern - funbit matching succeeded, bindings: %v\n", bindings)
	}

	// The pattern matched if funbit matching succeeded (no error)
	// Bindings may be empty for literal patterns
	return true, bindings
}

// convertBitstringPatternToExpression converts BitstringPattern to BitstringExpression for the adapter
func (e *ExecutionEngine) convertBitstringPatternToExpression(pattern *ast.BitstringPattern) *ast.BitstringExpression {
	expr := &ast.BitstringExpression{
		Segments: make([]ast.BitstringSegment, len(pattern.Elements)),
	}

	for i, element := range pattern.Elements {
		expr.Segments[i] = ast.BitstringSegment{
			Value:          element.Value,
			Size:           element.Size,
			SizeExpression: element.SizeExpression,
			IsDynamicSize:  element.IsDynamicSize,
			Specifiers:     element.Specifiers,
			ColonToken:     element.ColonToken,
			SlashToken:     element.SlashToken,
		}
	}

	return expr
}

func (e *ExecutionEngine) executeStatementWithBindings(stmt ast.Statement, bindings map[string]interface{}) (interface{}, error) {
	// Create a new local scope for the bound variables
	// This ensures pattern matching variables are available locally
	// and don't interfere with runtime variables

	// Save the current scope
	oldScope := e.localScope

	// Create a new scope with the old scope as parent
	newScope := sharedparser.NewScope(oldScope)
	e.localScope = newScope

	// Set bound variables in the local scope
	for name, value := range bindings {
		newScope.Set(name, value)
		if e.verbose {
			fmt.Printf("DEBUG: executeStatementWithBindings - set local variable '%s' = %v\n", name, value)
		}
	}

	// Execute the statement with the new scope
	result, err := e.executeStatement(stmt)

	// Copy bound variables to the parent scope before restoring
	// This makes variables extracted from match patterns available in subsequent statements
	for name, value := range bindings {
		oldScope.Set(name, value)
		if e.verbose {
			fmt.Printf("DEBUG: executeStatementWithBindings - copied variable '%s' to parent scope\n", name)
		}
	}

	// Restore the old scope (now with copied variables)
	e.localScope = oldScope

	return result, err
}

func (e *ExecutionEngine) executeStatementWithLocalScope(stmt ast.Statement) (interface{}, error) {
	// Create a new local scope for local variables
	// This ensures local variables in match arms don't interfere with global scope

	// Save the current scope
	oldScope := e.localScope

	// Create a new scope with the old scope as parent
	newScope := sharedparser.NewScope(oldScope)
	e.localScope = newScope

	// Execute the statement with the new scope
	result, err := e.executeStatement(stmt)

	// Restore the old scope
	e.localScope = oldScope

	return result, err
}
