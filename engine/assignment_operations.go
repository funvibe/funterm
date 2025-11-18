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
		return nil, errors.NewUserErrorWithASTPos("EXPRESSION_ASSIGNMENT_ERROR", fmt.Sprintf("failed to convert value for assignment: %v", err), exprAssignment.Value.Position())
	}

	// Handle different types of left-hand side expressions
	switch leftExpr := exprAssignment.Left.(type) {
	case *ast.IndexExpression:
		// Handle indexed assignment like dict["key"] = value
		return e.executeIndexedAssignment(leftExpr, value)
	default:
		return nil, errors.NewUserErrorWithASTPos("EXPRESSION_ASSIGNMENT_ERROR", fmt.Sprintf("cannot assign to expression of type %T", exprAssignment.Left), exprAssignment.Left.Position())
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
			return nil, errors.NewUserErrorWithASTPos("ASSIGNMENT_ERROR", "assignment requires qualified identifier (language.variable)", leftExpr.Position())
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

		return nil, errors.NewUserErrorWithASTPos("ASSIGNMENT_ERROR", fmt.Sprintf("runtime '%s' not available", language), leftExpr.Position())

	default:
		return nil, errors.NewUserErrorWithASTPos("ASSIGNMENT_ERROR", fmt.Sprintf("cannot assign to expression of type %T", left), left.Position())
	}
}

// executeIndexedAssignment executes indexed assignment like dict["key"] = value
func (e *ExecutionEngine) executeIndexedAssignment(indexExpr *ast.IndexExpression, rightValue interface{}) (interface{}, error) {
	// Check if this is a nested IndexExpression (like py.data.users[0].age = value)
	if nestedIndexExpr, ok := indexExpr.Object.(*ast.IndexExpression); ok {
		// This is a nested assignment - handle it by evaluating the nested expression first
		if e.verbose {
			fmt.Printf("DEBUG: executeIndexedAssignment - detected nested IndexExpression, handling recursively\n")
		}

		// For nested assignments, we need to handle the case where we're assigning to an index that doesn't exist
		// Instead of evaluating the nested index expression (which would fail for out-of-bounds), we need to build the path
		// and handle the assignment step by step
		return e.executeNestedIndexedAssignmentWithExpansion(nestedIndexExpr, indexExpr.Index, rightValue)
	}

	// 1. Evaluate the object (it can be a variable or another index expression)
	objectValue, err := e.convertExpressionToValue(indexExpr.Object)
	if err != nil {
		return nil, errors.NewUserErrorWithASTPos("INDEXED_ASSIGNMENT_ERROR", fmt.Sprintf("failed to evaluate object: %v", err), indexExpr.Object.Position())
	}

	// 2. Evaluate the index
	indexValue, err := e.convertExpressionToValue(indexExpr.Index)
	if err != nil {
		return nil, errors.NewUserErrorWithASTPos("INDEXED_ASSIGNMENT_ERROR", fmt.Sprintf("failed to evaluate index: %v", err), indexExpr.Index.Position())
	}

	// 3. Check mutability for simple (non-nested) assignments
	// Extract the variable name to check mutability
	varName, err := e.extractVariableName(indexExpr.Object)
	if err != nil {
		return nil, errors.NewUserErrorWithASTPos("INDEXED_ASSIGNMENT_ERROR", fmt.Sprintf("failed to extract variable name: %v", err), indexExpr.Object.Position())
	}

	// Check mutability for unqualified (global) variables
	if varName.language == "" {
		if varInfo, exists := e.getGlobalVariableInfo(varName.name); exists && !varInfo.IsMutable {
			return nil, errors.NewUserErrorWithASTPos("IMMUTABLE_VARIABLE_ERROR", fmt.Sprintf("cannot modify immutable variable '%s'", varName.name), indexExpr.Position())
		}
	}

	// 4. Handle simple (non-nested) assignments
	switch obj := objectValue.(type) {
	case map[string]interface{}:
		// Dictionary/object assignment
		key, ok := indexValue.(string)
		if !ok {
			return nil, errors.NewUserErrorWithASTPos("INDEXED_ASSIGNMENT_ERROR", fmt.Sprintf("dictionary index must be string, got %T", indexValue), indexExpr.Index.Position())
		}
		// Create a copy of the map and modify it
		newObj := make(map[string]interface{})
		for k, v := range obj {
			newObj[k] = v
		}
		newObj[key] = rightValue

		// Set the modified object back to runtime
		err = e.setVariableInRuntimeWithError(varName.language, varName.name, newObj)
		if err != nil {
			return nil, err
		}
		return rightValue, nil

	case []interface{}:
		// Array assignment
		var idx float64
		switch i := indexValue.(type) {
		case float64:
			idx = i
		case int64:
			idx = float64(i)
		case int:
			idx = float64(i)
		default:
			return nil, errors.NewUserErrorWithASTPos("INDEXED_ASSIGNMENT_ERROR", fmt.Sprintf("array index must be number, got %T", indexValue), indexExpr.Index.Position())
		}
		intIdx := int(idx)
		if intIdx < 0 {
			return nil, errors.NewUserErrorWithASTPos("INDEXED_ASSIGNMENT_ERROR", fmt.Sprintf("array index %d cannot be negative", intIdx), indexExpr.Index.Position())
		}

		// Expand array if index is beyond current length
		newObj := make([]interface{}, len(obj))
		copy(newObj, obj)

		if intIdx >= len(newObj) {
			// Expand array to accommodate the new index
			expanded := make([]interface{}, intIdx+1)
			copy(expanded, newObj)
			// Fill gaps with nil
			for i := len(newObj); i < intIdx; i++ {
				expanded[i] = nil
			}
			newObj = expanded
		}

		newObj[intIdx] = rightValue

		// Set the modified array back to runtime
		err = e.setVariableInRuntimeWithError(varName.language, varName.name, newObj)
		if err != nil {
			return nil, err
		}
		return rightValue, nil

	default:
		return nil, errors.NewUserErrorWithASTPos("INDEXED_ASSIGNMENT_ERROR", fmt.Sprintf("cannot assign to indexed expression of type %T", objectValue), indexExpr.Position())
	}
}

// executeAssignmentOnObject performs the actual assignment on an object
func (e *ExecutionEngine) executeAssignmentOnObject(objectValue interface{}, indexValue interface{}, rightValue interface{}, pos ast.Position) (interface{}, error) {
	switch obj := objectValue.(type) {
	case map[string]interface{}:
		// Dictionary/object assignment
		key, ok := indexValue.(string)
		if !ok {
			return nil, errors.NewUserErrorWithASTPos("INDEXED_ASSIGNMENT_ERROR", fmt.Sprintf("dictionary index must be string, got %T", indexValue), pos)
		}
		// Create a copy of the map and modify it
		newObj := make(map[string]interface{})
		for k, v := range obj {
			newObj[k] = v
		}
		newObj[key] = rightValue

		// For nested assignments, we need to update the object in place
		// The caller is responsible for persisting the change
		if e.verbose {
			fmt.Printf("DEBUG: executeAssignmentOnObject - modified map, new value: %v\n", newObj[key])
		}
		return rightValue, nil

	case []interface{}:
		// Array assignment
		var idx float64
		switch i := indexValue.(type) {
		case float64:
			idx = i
		case int64:
			idx = float64(i)
		case int:
			idx = float64(i)
		default:
			return nil, errors.NewUserErrorWithASTPos("INDEXED_ASSIGNMENT_ERROR", fmt.Sprintf("array index must be number, got %T", indexValue), pos)
		}
		intIdx := int(idx)
		if intIdx < 0 {
			return nil, errors.NewUserErrorWithASTPos("INDEXED_ASSIGNMENT_ERROR", fmt.Sprintf("array index %d cannot be negative", intIdx), pos)
		}

		// Expand array if index is beyond current length
		newObj := make([]interface{}, len(obj))
		copy(newObj, obj)

		if intIdx >= len(newObj) {
			// Expand array to accommodate the new index
			expanded := make([]interface{}, intIdx+1)
			copy(expanded, newObj)
			// Fill gaps with nil
			for i := len(newObj); i < intIdx; i++ {
				expanded[i] = nil
			}
			newObj = expanded
		}

		newObj[intIdx] = rightValue

		// For nested assignments, we need to update the object in place
		// The caller is responsible for persisting the change
		if e.verbose {
			fmt.Printf("DEBUG: executeAssignmentOnObject - modified array at index %d, new value: %v\n", intIdx, rightValue)
		}
		return rightValue, nil

	default:
		return nil, errors.NewUserErrorWithASTPos("INDEXED_ASSIGNMENT_ERROR", fmt.Sprintf("cannot assign to indexed expression of type %T", objectValue), pos)
	}
}

// setVariableInRuntimeWithError sets a variable in runtime and returns error
func (e *ExecutionEngine) setVariableInRuntimeWithError(language, name string, value interface{}) error {
	// Handle unqualified (global) variables
	if language == "" {
		e.setGlobalVariable(name, value)
		return nil
	}

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
func (e *ExecutionEngine) executeNestedIndexedAssignment(rootVarName *VariableName, nestedPath []interface{}, finalIndex interface{}, value interface{}, pos ast.Position) (interface{}, error) {
	// Check mutability for unqualified (global) variables
	if rootVarName.language == "" {
		if varInfo, exists := e.getGlobalVariableInfo(rootVarName.name); exists && !varInfo.IsMutable {
			return nil, errors.NewUserErrorWithASTPos("IMMUTABLE_VARIABLE_ERROR", fmt.Sprintf("cannot modify immutable variable '%s'", rootVarName.name), pos)
		}
	}

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
		// This is a qualified identifier like py.data.users
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
	} else if nestedIndexExpr, ok := indexExpr.Object.(*ast.IndexExpression); ok {
		// This is a nested index expression like py.data.users[0].age
		// We need to recursively evaluate the nested index expression first
		if e.verbose {
			fmt.Printf("DEBUG: executeIndexExpression - detected nested index expression, recursing\n")
		}
		objectValue, err = e.executeIndexExpression(nestedIndexExpr)
		if err != nil {
			return nil, errors.NewUserErrorWithASTPos("INDEX_EXPR_ERROR", fmt.Sprintf("failed to evaluate nested index expression: %v", err), nestedIndexExpr.Position())
		}
	} else {
		// For non-qualified objects, use the standard conversion
		objectValue, err = e.convertExpressionToValue(indexExpr.Object)
		if err != nil {
			return nil, errors.NewUserErrorWithASTPos("INDEX_EXPR_ERROR", fmt.Sprintf("failed to evaluate object: %v", err), indexExpr.Object.Position())
		}
	}

	// 2. Evaluate the index
	indexValue, err := e.convertExpressionToValue(indexExpr.Index)
	if err != nil {
		return nil, errors.NewUserErrorWithASTPos("INDEX_EXPR_ERROR", fmt.Sprintf("failed to evaluate index: %v", err), indexExpr.Index.Position())
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
			return nil, errors.NewUserErrorWithASTPos("INDEX_EXPR_ERROR", fmt.Sprintf("dictionary index must be string, got %T", indexValue), indexExpr.Index.Position())
		}
		value, exists := obj[key]
		if !exists {
			return nil, errors.NewUserErrorWithASTPos("INDEX_EXPR_ERROR", fmt.Sprintf("key '%s' not found in dictionary", key), indexExpr.Index.Position())
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
			return nil, errors.NewUserErrorWithASTPos("INDEX_EXPR_ERROR", fmt.Sprintf("array index must be number, got %T", indexValue), indexExpr.Index.Position())
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
		var idx float64
		switch i := indexValue.(type) {
		case float64:
			idx = i
		case int64:
			idx = float64(i)
		case int:
			idx = float64(i)
		default:
			return nil, errors.NewUserErrorWithASTPos("INDEX_EXPR_ERROR", fmt.Sprintf("byte array index must be number, got %T", indexValue), indexExpr.Index.Position())
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
		var idx float64
		switch i := indexValue.(type) {
		case float64:
			idx = i
		case int64:
			idx = float64(i)
		case int:
			idx = float64(i)
		default:
			return nil, errors.NewUserErrorWithASTPos("INDEX_EXPR_ERROR", fmt.Sprintf("bitstring index must be number, got %T", indexValue), indexExpr.Index.Position())
		}
		intIdx := int(idx)
		if e.verbose {
			fmt.Printf("DEBUG: executeIndexExpression - accessing bitstring at index %d\n", intIdx)
		}
		return obj.GetByte(intIdx), nil

	default:
		return nil, errors.NewUserErrorWithASTPos("INDEX_EXPR_ERROR", fmt.Sprintf("cannot index into type %T", objectValue), indexExpr.Position())
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
		return nil, errors.NewUserErrorWithASTPos("FIELD_ACCESS_ERROR", fmt.Sprintf("failed to evaluate object: %v", err), fieldAccess.Object.Position())
	}

	// 2. Handle different object types
	switch obj := objectValue.(type) {
	case map[string]interface{}:
		// Dictionary/object field access
		value, exists := obj[fieldAccess.Field]
		if !exists {
			return nil, errors.NewUserErrorWithASTPos("FIELD_ACCESS_ERROR", fmt.Sprintf("field '%s' not found in object", fieldAccess.Field), fieldAccess.Position())
		}
		return value, nil

	default:
		return nil, errors.NewUserErrorWithASTPos("FIELD_ACCESS_ERROR", fmt.Sprintf("cannot access field '%s' on type %T", fieldAccess.Field, objectValue), fieldAccess.Position())
	}
}

// extractVariableName extracts language and variable name from an expression
func (e *ExecutionEngine) extractVariableName(expr ast.Expression) (*VariableName, error) {
	switch typedExpr := expr.(type) {
	case *ast.Identifier:
		if !typedExpr.Qualified {
			// Неквалифицированная переменная - это глобальная переменная без runtime
			return &VariableName{language: "", name: typedExpr.Name}, nil
		}
		language := typedExpr.Language
		if language == "py" {
			language = "python"
		}
		if language == "js" {
			language = "node"
		}
		return &VariableName{language: language, name: typedExpr.Name}, nil
	default:
		return nil, fmt.Errorf("unsupported expression type for variable extraction: %T", expr)
	}
}

// getVariableFromRuntime gets a variable from the appropriate runtime
func (e *ExecutionEngine) getVariableFromRuntime(language, name string) (interface{}, error) {
	// Handle unqualified (global) variables
	if language == "" {
		if val, found := e.getGlobalVariable(name); found {
			return val, nil
		}
		return nil, fmt.Errorf("global variable '%s' not found", name)
	}

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

// executeNestedIndexedAssignmentWithExpansion handles nested indexed assignments with array expansion
func (e *ExecutionEngine) executeNestedIndexedAssignmentWithExpansion(nestedIndexExpr *ast.IndexExpression, finalIndex ast.Expression, rightValue interface{}) (interface{}, error) {
	if e.verbose {
		fmt.Printf("DEBUG: executeNestedIndexedAssignmentWithExpansion - handling nested assignment with expansion\n")
	}

	// Build the full path from the nested index expressions
	var path []interface{}
	var rootVarName *VariableName
	var err error

	// Extract the path from the nested index expression
	path, rootVarName, err = e.extractNestedPath(nestedIndexExpr)
	if err != nil {
		return nil, errors.NewUserErrorWithASTPos("INDEXED_ASSIGNMENT_ERROR", fmt.Sprintf("failed to extract nested path: %v", err), nestedIndexExpr.Position())
	}

	// Add the final index to the path
	finalIndexValue, err := e.convertExpressionToValue(finalIndex)
	if err != nil {
		return nil, errors.NewUserErrorWithASTPos("INDEXED_ASSIGNMENT_ERROR", fmt.Sprintf("failed to evaluate final index: %v", err), finalIndex.Position())
	}
	path = append(path, finalIndexValue)

	if e.verbose {
		fmt.Printf("DEBUG: executeNestedIndexedAssignmentWithExpansion - root variable: %s.%s, path: %v\n", rootVarName.language, rootVarName.name, path)
	}

	// Check mutability for unqualified (global) variables
	if rootVarName.language == "" {
		if varInfo, exists := e.getGlobalVariableInfo(rootVarName.name); exists && !varInfo.IsMutable {
			return nil, errors.NewUserErrorWithASTPos("IMMUTABLE_VARIABLE_ERROR", fmt.Sprintf("cannot modify immutable variable '%s'", rootVarName.name), finalIndex.Position())
		}
	}

	// Get the root object from runtime
	rootObject, err := e.getVariableFromRuntime(rootVarName.language, rootVarName.name)
	if err != nil {
		return nil, errors.NewUserErrorWithASTPos("INDEXED_ASSIGNMENT_ERROR", fmt.Sprintf("failed to get root variable %s.%s: %v", rootVarName.language, rootVarName.name, err), finalIndex.Position())
	}

	// Navigate through the nested path and create/update the structure with expansion
	current := rootObject
	for i, pathIndex := range path {
		if i == len(path)-1 {
			// This is the final level - perform the assignment
			return e.executeFinalAssignmentWithExpansion(current, pathIndex, rightValue, rootVarName, path[:i])
		}

		// Navigate to the next level
		switch obj := current.(type) {
		case map[string]interface{}:
			key, ok := pathIndex.(string)
			if !ok {
				return nil, errors.NewUserErrorWithASTPos("INDEXED_ASSIGNMENT_ERROR", fmt.Sprintf("nested path index must be string, got %T", pathIndex), finalIndex.Position())
			}

			if nextObj, exists := obj[key]; exists {
				current = nextObj
			} else {
				// Create new nested structure based on the next path index type
				if i+1 < len(path) {
					nextPathIndex := path[i+1]
					switch nextPathIndex.(type) {
					case string:
						// Next level is a map
						newMap := make(map[string]interface{})
						obj[key] = newMap
						current = newMap
					case float64, int64, int:
						// Next level is an array
						newArray := make([]interface{}, 0)
						obj[key] = newArray
						current = newArray
					default:
						return nil, errors.NewUserErrorWithASTPos("INDEXED_ASSIGNMENT_ERROR", fmt.Sprintf("unsupported path index type: %T", nextPathIndex), finalIndex.Position())
					}
				}
			}

		case []interface{}:
			// Array navigation with expansion
			var idx float64
			switch i := pathIndex.(type) {
			case float64:
				idx = i
			case int64:
				idx = float64(i)
			case int:
				idx = float64(i)
			default:
				return nil, errors.NewUserErrorWithASTPos("INDEXED_ASSIGNMENT_ERROR", fmt.Sprintf("array index must be number, got %T", pathIndex), finalIndex.Position())
			}
			intIdx := int(idx)

			// Expand array if index is beyond current length
			if intIdx >= len(obj) {
				expanded := make([]interface{}, intIdx+1)
				copy(expanded, obj)
				// Fill gaps with nil
				for i := len(obj); i < intIdx; i++ {
					expanded[i] = nil
				}
				// Update the array in the parent structure
				err = e.updateNestedStructure(rootVarName, path[:i], expanded)
				if err != nil {
					return nil, err
				}
				obj = expanded
			}

			if intIdx < len(obj) {
				if obj[intIdx] != nil {
					current = obj[intIdx]
				} else {
					// Create new nested structure based on the next path index type
					if i+1 < len(path) {
						nextPathIndex := path[i+1]
						switch nextPathIndex.(type) {
						case string:
							// Next level is a map
							newMap := make(map[string]interface{})
							obj[intIdx] = newMap
							current = newMap
						case float64, int64, int:
							// Next level is an array
							newArray := make([]interface{}, 0)
							obj[intIdx] = newArray
							current = newArray
						default:
							return nil, errors.NewUserErrorWithASTPos("INDEXED_ASSIGNMENT_ERROR", fmt.Sprintf("unsupported path index type: %T", nextPathIndex), finalIndex.Position())
						}
					} else {
						// This is the last level, but we're not at the final assignment yet
						// This shouldn't happen in normal flow
						return nil, errors.NewUserErrorWithASTPos("INDEXED_ASSIGNMENT_ERROR", fmt.Sprintf("unexpected end of path at index %d", intIdx), finalIndex.Position())
					}
				}
			} else {
				return nil, errors.NewUserErrorWithASTPos("INDEXED_ASSIGNMENT_ERROR", fmt.Sprintf("array index %d out of bounds after expansion", intIdx), finalIndex.Position())
			}

		default:
			return nil, errors.NewUserErrorWithASTPos("INDEXED_ASSIGNMENT_ERROR", fmt.Sprintf("cannot navigate nested path at level %d, type %T", i, current), finalIndex.Position())
		}
	}

	return rightValue, nil
}

// executeFinalAssignmentWithExpansion performs the final assignment with expansion
func (e *ExecutionEngine) executeFinalAssignmentWithExpansion(current interface{}, finalIndex interface{}, rightValue interface{}, rootVarName *VariableName, path []interface{}) (interface{}, error) {
	switch obj := current.(type) {
	case map[string]interface{}:
		key, ok := finalIndex.(string)
		if !ok {
			return nil, errors.NewUserErrorWithASTPos("INDEXED_ASSIGNMENT_ERROR", fmt.Sprintf("final index must be string, got %T", finalIndex), ast.Position{})
		}
		// Create a copy of the map and modify it
		newObj := make(map[string]interface{})
		for k, v := range obj {
			newObj[k] = v
		}
		newObj[key] = rightValue

		// Update the nested structure
		err := e.updateNestedStructure(rootVarName, path, newObj)
		if err != nil {
			return nil, err
		}
		return rightValue, nil

	case []interface{}:
		// Array assignment with expansion
		var idx float64
		switch i := finalIndex.(type) {
		case float64:
			idx = i
		case int64:
			idx = float64(i)
		case int:
			idx = float64(i)
		default:
			return nil, errors.NewUserErrorWithASTPos("INDEXED_ASSIGNMENT_ERROR", fmt.Sprintf("array index must be number, got %T", finalIndex), ast.Position{})
		}
		intIdx := int(idx)

		// Expand array if index is beyond current length
		newObj := make([]interface{}, len(obj))
		copy(newObj, obj)

		if intIdx >= len(newObj) {
			// Expand array to accommodate the new index
			expanded := make([]interface{}, intIdx+1)
			copy(expanded, newObj)
			// Fill gaps with nil
			for i := len(newObj); i < intIdx; i++ {
				expanded[i] = nil
			}
			newObj = expanded
		}

		newObj[intIdx] = rightValue

		// Update the nested structure
		err := e.updateNestedStructure(rootVarName, path, newObj)
		if err != nil {
			return nil, err
		}
		return rightValue, nil

	default:
		return nil, errors.NewUserErrorWithASTPos("INDEXED_ASSIGNMENT_ERROR", fmt.Sprintf("cannot assign to indexed expression of type %T", current), ast.Position{})
	}
}

// updateNestedStructure updates a nested structure in the runtime
func (e *ExecutionEngine) updateNestedStructure(rootVarName *VariableName, path []interface{}, newValue interface{}) error {
	// Get the root object from runtime
	rootObject, err := e.getVariableFromRuntime(rootVarName.language, rootVarName.name)
	if err != nil {
		return errors.NewUserErrorWithASTPos("INDEXED_ASSIGNMENT_ERROR", fmt.Sprintf("failed to get root variable %s.%s: %v", rootVarName.language, rootVarName.name, err), ast.Position{})
	}

	// Navigate through the path and update the structure
	current := rootObject
	for i, pathIndex := range path {
		switch obj := current.(type) {
		case map[string]interface{}:
			key, ok := pathIndex.(string)
			if !ok {
				return errors.NewUserErrorWithASTPos("INDEXED_ASSIGNMENT_ERROR", fmt.Sprintf("nested path index must be string, got %T", pathIndex), ast.Position{})
			}

			if i == len(path)-1 {
				// This is the last level - update the value
				obj[key] = newValue
			} else {
				// Navigate deeper
				if nextObj, exists := obj[key]; exists {
					current = nextObj
				} else {
					return errors.NewUserErrorWithASTPos("INDEXED_ASSIGNMENT_ERROR", fmt.Sprintf("path %v does not exist in object", path[:i+1]), ast.Position{})
				}
			}

		case []interface{}:
			var idx float64
			switch i := pathIndex.(type) {
			case float64:
				idx = i
			case int64:
				idx = float64(i)
			case int:
				idx = float64(i)
			default:
				return errors.NewUserErrorWithASTPos("INDEXED_ASSIGNMENT_ERROR", fmt.Sprintf("array index must be number, got %T", pathIndex), ast.Position{})
			}
			intIdx := int(idx)

			if i == len(path)-1 {
				// This is the last level - update the array
				if intIdx < len(obj) {
					obj[intIdx] = newValue
				} else {
					return errors.NewUserErrorWithASTPos("INDEXED_ASSIGNMENT_ERROR", fmt.Sprintf("array index %d out of bounds", intIdx), ast.Position{})
				}
			} else {
				// Navigate deeper
				if intIdx < len(obj) && obj[intIdx] != nil {
					current = obj[intIdx]
				} else {
					return errors.NewUserErrorWithASTPos("INDEXED_ASSIGNMENT_ERROR", fmt.Sprintf("path %v does not exist in array", path[:i+1]), ast.Position{})
				}
			}

		default:
			return errors.NewUserErrorWithASTPos("INDEXED_ASSIGNMENT_ERROR", fmt.Sprintf("cannot navigate nested path at level %d, type %T", i, current), ast.Position{})
		}
	}

	// Save the modified root object back to runtime
	err = e.setVariableInRuntimeWithError(rootVarName.language, rootVarName.name, rootObject)
	if err != nil {
		return err
	}

	return nil
}
