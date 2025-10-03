package engine

import (
	"fmt"

	"funterm/errors"
	"funterm/shared"
	"go-parser/pkg/ast"
)

// VariableName holds language and name information
type VariableName struct {
	language string
	name     string
}

// executeExpressionAssignment executes an expression assignment like dict["key"] = value
func (e *ExecutionEngine) executeExpressionAssignment(exprAssignment *ast.ExpressionAssignment) (interface{}, error) {
	// Convert the value to the appropriate format
	value, err := e.convertExpressionToValue(exprAssignment.Value)
	if err != nil {
		return nil, errors.NewSystemError("EXPRESSION_ASSIGNMENT_ERROR", fmt.Sprintf("failed to convert value for assignment: %v", err))
	}

	// Handle different types of left-hand side expressions
	switch leftExpr := exprAssignment.Left.(type) {
	case *ast.IndexExpression:
		// Handle indexed assignment like dict["key"] = value
		return e.executeIndexedAssignment(leftExpr, value)
	default:
		return nil, errors.NewSystemError("EXPRESSION_ASSIGNMENT_ERROR", fmt.Sprintf("cannot assign to expression of type %T", exprAssignment.Left))
	}
}

// executeAssignment executes an assignment operation like x = value or dict["key"] = value
func (e *ExecutionEngine) executeAssignment(left ast.Expression, rightValue interface{}) (interface{}, error) {
	// Handle different types of left-hand side expressions
	switch leftExpr := left.(type) {
	case *ast.IndexExpression:
		// Handle indexed assignment like dict["key"] = value
		return e.executeIndexedAssignment(leftExpr, rightValue)

	case *ast.Identifier:
		// Handle simple variable assignment
		if !leftExpr.Qualified {
			return nil, errors.NewSystemError("ASSIGNMENT_ERROR", "assignment requires qualified identifier (language.variable)")
		}

		language := leftExpr.Language
		variableName := leftExpr.Name

		// Handle alias 'py' for 'python'
		if language == "py" {
			language = "python"
		}
		// Handle alias 'js' for 'node'
		if language == "js" {
			language = "node"
		}

		// Try to get the runtime from the runtime manager first
		rt, err := e.runtimeManager.GetRuntime(language)
		if err == nil {
			// Runtime found in manager, use it
			return e.setVariableInRuntime(rt, language, variableName, rightValue)
		}

		// Try to get or create runtime from cache
		if e.runtimeRegistry != nil {
			rt, err := e.GetOrCreateRuntime(language)
			if err == nil {
				return e.setVariableInRuntime(rt, language, variableName, rightValue)
			}
		}

		return nil, errors.NewSystemError("ASSIGNMENT_ERROR", fmt.Sprintf("runtime '%s' not available", language))

	default:
		return nil, errors.NewSystemError("ASSIGNMENT_ERROR", fmt.Sprintf("cannot assign to expression of type %T", left))
	}
}

// executeIndexedAssignment executes indexed assignment like dict["key"] = value
func (e *ExecutionEngine) executeIndexedAssignment(indexExpr *ast.IndexExpression, rightValue interface{}) (interface{}, error) {
	// 1. Evaluate the object (it can be a variable or another index expression)
	objectValue, err := e.convertExpressionToValue(indexExpr.Object)
	if err != nil {
		return nil, errors.NewSystemError("INDEXED_ASSIGNMENT_ERROR", fmt.Sprintf("failed to evaluate object: %v", err))
	}

	// 2. Evaluate the index
	indexValue, err := e.convertExpressionToValue(indexExpr.Index)
	if err != nil {
		return nil, errors.NewSystemError("INDEXED_ASSIGNMENT_ERROR", fmt.Sprintf("failed to evaluate index: %v", err))
	}

	// 3. For nested assignments, we need to handle the case where objectValue is also an index expression result
	// This requires getting the root variable and rebuilding the nested structure

	// Check if this is a nested assignment by looking at the AST structure
	nestedPath, rootVarName, err := e.extractNestedPath(indexExpr.Object)
	if err != nil {
		return nil, errors.NewSystemError("INDEXED_ASSIGNMENT_ERROR", fmt.Sprintf("failed to extract nested path: %v", err))
	}

	if nestedPath != nil && len(nestedPath) > 0 {
		// This is a nested assignment - need to rebuild the structure
		return e.executeNestedIndexedAssignment(rootVarName, nestedPath, indexValue, rightValue)
	}

	// 4. Handle simple (non-nested) assignments
	switch obj := objectValue.(type) {
	case map[string]interface{}:
		// Dictionary/object assignment
		key, ok := indexValue.(string)
		if !ok {
			return nil, errors.NewSystemError("INDEXED_ASSIGNMENT_ERROR", fmt.Sprintf("dictionary index must be string, got %T", indexValue))
		}
		// Create a copy of the map and modify it
		newObj := make(map[string]interface{})
		for k, v := range obj {
			newObj[k] = v
		}
		newObj[key] = rightValue

		// Get variable name for simple assignment
		varName, err := e.extractVariableName(indexExpr.Object)
		if err != nil {
			return nil, errors.NewSystemError("INDEXED_ASSIGNMENT_ERROR", fmt.Sprintf("failed to extract variable name: %v", err))
		}

		// Set the modified object back to runtime
		err = e.setVariableInRuntimeWithError(varName.language, varName.name, newObj)
		if err != nil {
			return nil, err
		}
		return rightValue, nil

	case []interface{}:
		// Array assignment
		idx, ok := indexValue.(float64)
		if !ok {
			return nil, errors.NewSystemError("INDEXED_ASSIGNMENT_ERROR", fmt.Sprintf("array index must be number, got %T", indexValue))
		}
		intIdx := int(idx)
		if intIdx < 0 || intIdx >= len(obj) {
			return nil, errors.NewSystemError("INDEXED_ASSIGNMENT_ERROR", fmt.Sprintf("array index %d out of bounds (length %d)", intIdx, len(obj)))
		}
		// Create a copy of the array and modify it
		newObj := make([]interface{}, len(obj))
		copy(newObj, obj)
		newObj[intIdx] = rightValue

		// Get variable name for simple assignment
		varName, err := e.extractVariableName(indexExpr.Object)
		if err != nil {
			return nil, errors.NewSystemError("INDEXED_ASSIGNMENT_ERROR", fmt.Sprintf("failed to extract variable name: %v", err))
		}

		// Set the modified array back to runtime
		err = e.setVariableInRuntimeWithError(varName.language, varName.name, newObj)
		if err != nil {
			return nil, err
		}
		return rightValue, nil

	default:
		return nil, errors.NewSystemError("INDEXED_ASSIGNMENT_ERROR", fmt.Sprintf("cannot assign to indexed expression of type %T", objectValue))
	}
}

// setVariableInRuntimeWithError sets a variable in runtime and returns error
func (e *ExecutionEngine) setVariableInRuntimeWithError(language, name string, value interface{}) error {
	// Try to get the runtime from the runtime manager first
	rt, err := e.runtimeManager.GetRuntime(language)
	if err == nil {
		// Runtime found in manager, use it
		return rt.SetVariable(name, value)
	}

	// Try to get or create runtime from cache
	if e.runtimeRegistry != nil {
		rt, err := e.GetOrCreateRuntime(language)
		if err == nil {
			return rt.SetVariable(name, value)
		}
	}

	return fmt.Errorf("runtime '%s' not available", language)
}

// extractNestedPath extracts the nested path from an IndexExpression AST
func (e *ExecutionEngine) extractNestedPath(expr ast.Expression) ([]interface{}, *VariableName, error) {
	var path []interface{}
	current := expr

	// Traverse the IndexExpression chain to build the path
	for {
		if indexExpr, ok := current.(*ast.IndexExpression); ok {
			// Evaluate the index and add it to the path (in reverse order)
			indexValue, err := e.convertExpressionToValue(indexExpr.Index)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to evaluate index: %v", err)
			}
			path = append([]interface{}{indexValue}, path...) // prepend to maintain correct order

			// Move to the next level
			current = indexExpr.Object
		} else {
			// We've reached the root - should be a qualified identifier
			varName, err := e.extractVariableName(current)
			if err != nil {
				return nil, nil, fmt.Errorf("root expression is not a qualified variable: %v", err)
			}

			return path, varName, nil
		}
	}
}

// executeNestedIndexedAssignment handles nested indexed assignments like dict["a"]["b"]["c"] = value
func (e *ExecutionEngine) executeNestedIndexedAssignment(rootVarName *VariableName, nestedPath []interface{}, finalIndex interface{}, value interface{}) (interface{}, error) {
	// Get the root object from runtime
	rootObject, err := e.getVariableFromRuntime(rootVarName.language, rootVarName.name)
	if err != nil {
		return nil, errors.NewSystemError("NESTED_INDEXED_ASSIGNMENT_ERROR", fmt.Sprintf("failed to get root variable %s.%s: %v", rootVarName.language, rootVarName.name, err))
	}

	// Navigate through the nested path and create/update the structure
	current := rootObject
	for i, pathIndex := range nestedPath {
		switch obj := current.(type) {
		case map[string]interface{}:
			key, ok := pathIndex.(string)
			if !ok {
				return nil, errors.NewSystemError("NESTED_INDEXED_ASSIGNMENT_ERROR", fmt.Sprintf("nested path index must be string, got %T", pathIndex))
			}

			if i == len(nestedPath)-1 {
				// This is the last level of the path - now we need to set the final index
				finalKey, ok := finalIndex.(string)
				if !ok {
					return nil, errors.NewSystemError("NESTED_INDEXED_ASSIGNMENT_ERROR", fmt.Sprintf("final index must be string, got %T", finalIndex))
				}

				// Get the final object (the one that contains the final key)
				if finalObj, exists := obj[key]; exists {
					if finalMap, ok := finalObj.(map[string]interface{}); ok {
						finalMap[finalKey] = value
					} else {
						return nil, errors.NewSystemError("NESTED_INDEXED_ASSIGNMENT_ERROR", fmt.Sprintf("final object at path %v is not a map, got %T", nestedPath, finalObj))
					}
				} else {
					return nil, errors.NewSystemError("NESTED_INDEXED_ASSIGNMENT_ERROR", fmt.Sprintf("path %v does not exist in object", nestedPath))
				}
			} else {
				// Navigate deeper
				if nextObj, exists := obj[key]; exists {
					current = nextObj
				} else {
					// Create new nested map
					newMap := make(map[string]interface{})
					obj[key] = newMap
					current = newMap
				}
			}

		default:
			return nil, errors.NewSystemError("NESTED_INDEXED_ASSIGNMENT_ERROR", fmt.Sprintf("cannot navigate nested path at level %d, type %T", i, current))
		}
	}

	// Save the modified root object back to runtime
	err = e.setVariableInRuntimeWithError(rootVarName.language, rootVarName.name, rootObject)
	if err != nil {
		return nil, err
	}

	return value, nil
}

// executeIndexExpression executes an index expression like dict["key"] or arr[0]
func (e *ExecutionEngine) executeIndexExpression(indexExpr *ast.IndexExpression) (interface{}, error) {
	// Debug output to see what we're trying to index
	if e.verbose {
		fmt.Printf("DEBUG: executeIndexExpression - object type: %T, index type: %T\n", indexExpr.Object, indexExpr.Index)
	}

	// 1. Evaluate the object (it can be a variable or another index expression)
	// Special handling for language-qualified identifiers like lua.data
	var objectValue interface{}
	var err error

	if ident, ok := indexExpr.Object.(*ast.Identifier); ok && ident.Qualified {
		// This is a qualified identifier like lua.data
		// We need to get the variable directly from the specified language runtime
		if e.verbose {
			fmt.Printf("DEBUG: executeIndexExpression - detected qualified identifier: %s.%s\n", ident.Language, ident.Name)
		}

		// Get the language name (handle aliases)
		language := ident.Language
		if language == "py" {
			language = "python"
		}
		if language == "js" {
			language = "node"
		}

		// Try to get the runtime from the runtime manager first
		rt, err := e.runtimeManager.GetRuntime(language)
		if err == nil {
			// Runtime found in manager, use it
			objectValue, err = e.readVariableFromRuntime(rt, language, ident.Name)
		} else {
			// Try to get or create runtime from cache
			if e.runtimeRegistry != nil {
				rt, err := e.GetOrCreateRuntime(language)
				if err == nil {
					objectValue, err = e.readVariableFromRuntime(rt, language, ident.Name)
				} else {
					return nil, errors.NewSystemError("INDEX_EXPR_ERROR", fmt.Sprintf("failed to evaluate qualified object '%s.%s': %v", ident.Language, ident.Name, err))
				}
			} else {
				return nil, errors.NewSystemError("INDEX_EXPR_ERROR", fmt.Sprintf("failed to evaluate qualified object '%s.%s': %v", ident.Language, ident.Name, err))
			}
		}

		if err != nil {
			return nil, errors.NewSystemError("INDEX_EXPR_ERROR", fmt.Sprintf("failed to evaluate qualified object '%s.%s': %v", ident.Language, ident.Name, err))
		}
	} else {
		// For non-qualified objects, use the standard conversion
		objectValue, err = e.convertExpressionToValue(indexExpr.Object)
		if err != nil {
			return nil, errors.NewSystemError("INDEX_EXPR_ERROR", fmt.Sprintf("failed to evaluate object: %v", err))
		}
	}

	// 2. Evaluate the index
	indexValue, err := e.convertExpressionToValue(indexExpr.Index)
	if err != nil {
		return nil, errors.NewSystemError("INDEX_EXPR_ERROR", fmt.Sprintf("failed to evaluate index: %v", err))
	}

	// Debug output to see evaluated values
	if e.verbose {
		fmt.Printf("DEBUG: executeIndexExpression - evaluated object type: %T, value: %v, evaluated index type: %T, value: %v\n", objectValue, objectValue, indexValue, indexValue)
	}

	// 3. Handle different object types
	switch obj := objectValue.(type) {
	case map[string]interface{}:
		// Dictionary/object access
		key, ok := indexValue.(string)
		if !ok {
			return nil, errors.NewSystemError("INDEX_EXPR_ERROR", fmt.Sprintf("dictionary index must be string, got %T", indexValue))
		}
		value, exists := obj[key]
		if !exists {
			return nil, errors.NewSystemError("INDEX_EXPR_ERROR", fmt.Sprintf("key '%s' not found in dictionary", key))
		}
		if e.verbose {
			fmt.Printf("DEBUG: executeIndexExpression - returning dictionary value: %v (type: %T)\n", value, value)
		}
		return value, nil

	case []interface{}:
		// Array access
		var idx float64
		switch i := indexValue.(type) {
		case float64:
			idx = i
		case int64:
			idx = float64(i)
		case int:
			idx = float64(i)
		default:
			return nil, errors.NewSystemError("INDEX_EXPR_ERROR", fmt.Sprintf("array index must be number, got %T", indexValue))
		}
		intIdx := int(idx)
		if intIdx < 0 || intIdx >= len(obj) {
			if e.verbose {
				fmt.Printf("DEBUG: executeIndexExpression - array index %d out of bounds (length %d), returning nil for wildcard matching\n", intIdx, len(obj))
			}
			// Return nil for out-of-bounds access so pattern matching can handle it with wildcard
			return nil, nil
		}
		if e.verbose {
			fmt.Printf("DEBUG: executeIndexExpression - returning array value at index %d: %v (type: %T)\n", intIdx, obj[intIdx], obj[intIdx])
		}
		return obj[intIdx], nil

	case []uint8:
		// Byte array access (from bitstrings)
		idx, ok := indexValue.(float64)
		if !ok {
			return nil, errors.NewSystemError("INDEX_EXPR_ERROR", fmt.Sprintf("byte array index must be number, got %T", indexValue))
		}
		intIdx := int(idx)
		if intIdx < 0 || intIdx >= len(obj) {
			if e.verbose {
				fmt.Printf("DEBUG: executeIndexExpression - byte array index %d out of bounds (length %d), returning nil for wildcard matching\n", intIdx, len(obj))
			}
			// Return nil for out-of-bounds access so pattern matching can handle it with wildcard
			return nil, nil
		}
		if e.verbose {
			fmt.Printf("DEBUG: executeIndexExpression - returning byte array value at index %d: %v (type: %T)\n", intIdx, obj[intIdx], obj[intIdx])
		}
		return obj[intIdx], nil

	case *shared.BitstringObject:
		// Bitstring object access (from funbit)
		idx, ok := indexValue.(float64)
		if !ok {
			return nil, errors.NewSystemError("INDEX_EXPR_ERROR", fmt.Sprintf("bitstring index must be number, got %T", indexValue))
		}
		intIdx := int(idx)
		if e.verbose {
			fmt.Printf("DEBUG: executeIndexExpression - accessing bitstring at index %d\n", intIdx)
		}
		return obj.GetByte(intIdx), nil

	default:
		return nil, errors.NewSystemError("INDEX_EXPR_ERROR", fmt.Sprintf("cannot index into type %T", objectValue))
	}
}

// executeFieldAccess executes a field access expression like lua.data.name
func (e *ExecutionEngine) executeFieldAccess(fieldAccess *ast.FieldAccess) (interface{}, error) {
	// 1. Evaluate the object (it can be an identifier or another field access)

	// Special handling for language identifiers as the object
	if ident, ok := fieldAccess.Object.(*ast.Identifier); ok {
		// Check if this is a language identifier (lua, python, py, go)
		if e.isLanguageIdentifier(ident) {
			// This is a language-based field access like lua.data.name
			// We need to get the variable from the specified language runtime
			return e.executeLanguageFieldAccess(ident, fieldAccess.Field)
		}
	}

	// For non-language objects, evaluate normally
	objectValue, err := e.convertExpressionToValue(fieldAccess.Object)
	if err != nil {
		return nil, errors.NewSystemError("FIELD_ACCESS_ERROR", fmt.Sprintf("failed to evaluate object: %v", err))
	}

	// 2. Handle different object types
	switch obj := objectValue.(type) {
	case map[string]interface{}:
		// Dictionary/object field access
		value, exists := obj[fieldAccess.Field]
		if !exists {
			return nil, errors.NewSystemError("FIELD_ACCESS_ERROR", fmt.Sprintf("field '%s' not found in object", fieldAccess.Field))
		}
		return value, nil

	default:
		return nil, errors.NewSystemError("FIELD_ACCESS_ERROR", fmt.Sprintf("cannot access field '%s' on type %T", fieldAccess.Field, objectValue))
	}
}

// extractVariableName extracts language and variable name from an expression
func (e *ExecutionEngine) extractVariableName(expr ast.Expression) (*VariableName, error) {
	switch typedExpr := expr.(type) {
	case *ast.Identifier:
		if !typedExpr.Qualified {
			return nil, fmt.Errorf("variable must be qualified (language.variable)")
		}
		language := typedExpr.Language
		if language == "py" {
			language = "python"
		}
		return &VariableName{language: language, name: typedExpr.Name}, nil
	default:
		return nil, fmt.Errorf("unsupported expression type for variable extraction: %T", expr)
	}
}

// getVariableFromRuntime gets a variable from the appropriate runtime
func (e *ExecutionEngine) getVariableFromRuntime(language, name string) (interface{}, error) {
	// Try to get the runtime from the runtime manager first
	rt, err := e.runtimeManager.GetRuntime(language)
	if err == nil {
		// Runtime found in manager, use it
		return rt.GetVariable(name)
	}

	// Try to get or create runtime from cache
	if e.runtimeRegistry != nil {
		rt, err := e.GetOrCreateRuntime(language)
		if err == nil {
			return rt.GetVariable(name)
		}
	}

	return nil, fmt.Errorf("runtime '%s' not available", language)
}

// isLanguageIdentifier checks if an identifier is a language name (lua, python, py, go, js, node)
func (e *ExecutionEngine) isLanguageIdentifier(ident *ast.Identifier) bool {
	switch ident.Name {
	case "lua", "python", "py", "go", "js", "node":
		return true
	default:
		return false
	}
}

// executeLanguageFieldAccess handles field access starting with a language identifier like lua.data.name
func (e *ExecutionEngine) executeLanguageFieldAccess(ident *ast.Identifier, field string) (interface{}, error) {
	// Get the language name
	language := ident.Name
	if language == "py" {
		language = "python"
	}
	if language == "js" {
		language = "node"
	}

	// Try to get the runtime
	rt, err := e.runtimeManager.GetRuntime(language)
	if err != nil {
		// Try to get or create runtime from cache
		if e.runtimeRegistry != nil {
			rt, err = e.GetOrCreateRuntime(language)
		}

		if err != nil {
			return nil, errors.NewSystemError("RUNTIME_NOT_FOUND", fmt.Sprintf("runtime for language '%s' not found", language))
		}
	}

	// Check if runtime is ready
	if !rt.IsReady() {
		return nil, errors.NewSystemError("RUNTIME_NOT_READY", fmt.Sprintf("%s runtime is not ready", language))
	}

	// First, try to get the variable from the runtime directly
	value, err := rt.GetVariable(field)
	if err == nil {
		if e.verbose {
			fmt.Printf("DEBUG: executeLanguageFieldAccess - found variable '%s' directly in %s runtime: %v\n", field, language, value)
		}
		return value, nil
	}

	// If not found in runtime, try to get the variable from shared storage (for cross-language access)
	if e.sharedVariables != nil {
		if sharedValue, found := e.GetSharedVariable(language, field); found {
			if e.verbose {
				fmt.Printf("DEBUG: executeLanguageFieldAccess - found variable '%s' in shared storage for language '%s': %v\n", field, language, sharedValue)
			}
			return sharedValue, nil
		}
	}

	return nil, errors.NewRuntimeError(language, "VARIABLE_NOT_FOUND", fmt.Sprintf("variable '%s' not found in %s runtime", field, language))
}
