package engine

import (
	"fmt"

	"funterm/errors"
	"funterm/shared"
	"go-parser/pkg/ast"
	"go-parser/pkg/lexer"
)

// executeBinaryExpression executes a binary expression (e.g., a + b, x < 10)
func (e *ExecutionEngine) executeBinaryExpression(binaryExpr *ast.BinaryExpression) (interface{}, error) {
	// Handle logical operators first to enable short-circuiting
	switch binaryExpr.Operator {
	case "&&":
		return e.executeLogicalAndWithShortCircuit(binaryExpr)
	case "||":
		return e.executeLogicalOrWithShortCircuit(binaryExpr)
	}

	// For all other operators, evaluate both operands first
	leftValue, err := e.convertExpressionToValue(binaryExpr.Left)
	if err != nil {
		return nil, errors.NewSystemError("BINARY_EXPR_ERROR", fmt.Sprintf("failed to evaluate left operand: %v", err))
	}

	rightValue, err := e.convertExpressionToValue(binaryExpr.Right)
	if err != nil {
		return nil, errors.NewSystemError("BINARY_EXPR_ERROR", fmt.Sprintf("failed to evaluate right operand: %v", err))
	}

	// Handle different operators
	switch binaryExpr.Operator {
	// Arithmetic operators
	case "+":
		return e.executeArithmeticAdd(leftValue, rightValue)
	case "-":
		return e.executeArithmeticSubtract(leftValue, rightValue)
	case "*":
		return e.executeArithmeticMultiply(leftValue, rightValue)
	case "/":
		return e.executeArithmeticDivide(leftValue, rightValue)
	case "%":
		return e.executeArithmeticModulo(leftValue, rightValue)

	// Comparison operators
	case "==":
		return e.executeComparisonEqual(leftValue, rightValue)
	case "!=":
		return e.executeComparisonNotEqual(leftValue, rightValue)
	case "<":
		return e.executeComparisonLess(leftValue, rightValue)
	case "<=":
		return e.executeComparisonLessEqual(leftValue, rightValue)
	case ">":
		return e.executeComparisonGreater(leftValue, rightValue)
	case ">=":
		return e.executeComparisonGreaterEqual(leftValue, rightValue)

	// Bitwise operators
	case "&":
		return e.executeBitwiseAnd(leftValue, rightValue)
	case "|":
		return e.executeBitwiseOr(leftValue, rightValue)
	case "^":
		return e.executeBitwiseXor(leftValue, rightValue)
	case "<<":
		return e.executeBitwiseLeftShift(leftValue, rightValue)
	case ">>":
		return e.executeBitwiseRightShift(leftValue, rightValue)

	// String concatenation
	case "++":
		return e.executeStringConcat(leftValue, rightValue)

	// Assignment operator
	case "=":
		return e.executeAssignment(binaryExpr.Left, rightValue)

	default:
		return nil, errors.NewSystemError("UNSUPPORTED_OPERATOR", fmt.Sprintf("unsupported binary operator: %s", binaryExpr.Operator))
	}
}

// executeArithmeticAdd handles addition operation
func (e *ExecutionEngine) executeArithmeticAdd(left, right interface{}) (interface{}, error) {
	// Debug output to see what types we're getting
	if e.verbose {
		fmt.Printf("DEBUG: executeArithmeticAdd - left type: %T, value: %v, right type: %T, value: %v\n", left, left, right, right)
	}

	// Handle numeric types
	switch l := left.(type) {
	case int64:
		switch r := right.(type) {
		case int64:
			return l + r, nil
		case float64:
			return float64(l) + r, nil
		case uint64:
			return uint64(l) + r, nil
		}
	case float64:
		switch r := right.(type) {
		case int64:
			return l + float64(r), nil
		case float64:
			return l + r, nil
		case uint64:
			return l + float64(r), nil
		}
	case uint64:
		switch r := right.(type) {
		case int64:
			return l + uint64(r), nil
		case float64:
			return float64(l) + r, nil
		case uint64:
			return l + r, nil
		}
	}

	// Handle string concatenation if both are strings
	if lStr, lOk := left.(string); lOk {
		if rStr, rOk := right.(string); rOk {
			return lStr + rStr, nil
		}
	}

	return nil, errors.NewValidationError("OPERAND_TYPE_MISMATCH", "addition requires numeric operands or two strings")
}

// executeArithmeticSubtract handles subtraction operation
func (e *ExecutionEngine) executeArithmeticSubtract(left, right interface{}) (interface{}, error) {
	switch l := left.(type) {
	case int64:
		switch r := right.(type) {
		case int64:
			return l - r, nil
		case float64:
			return float64(l) - r, nil
		}
	case float64:
		switch r := right.(type) {
		case int64:
			return l - float64(r), nil
		case float64:
			return l - r, nil
		}
	}

	return nil, errors.NewValidationError("OPERAND_TYPE_MISMATCH", "subtraction requires numeric operands")
}

// executeArithmeticMultiply handles multiplication operation
func (e *ExecutionEngine) executeArithmeticMultiply(left, right interface{}) (interface{}, error) {
	switch l := left.(type) {
	case int64:
		switch r := right.(type) {
		case int64:
			return l * r, nil
		case float64:
			return float64(l) * r, nil
		case uint64:
			return uint64(l) * r, nil
		}
	case float64:
		switch r := right.(type) {
		case int64:
			return l * float64(r), nil
		case float64:
			return l * r, nil
		case uint64:
			return l * float64(r), nil
		}
	case uint64:
		switch r := right.(type) {
		case int64:
			return l * uint64(r), nil
		case float64:
			return float64(l) * r, nil
		case uint64:
			return l * r, nil
		}
	}

	return nil, errors.NewValidationError("OPERAND_TYPE_MISMATCH", "multiplication requires numeric operands")
}

// executeArithmeticDivide handles division operation
func (e *ExecutionEngine) executeArithmeticDivide(left, right interface{}) (interface{}, error) {
	switch l := left.(type) {
	case int64:
		switch r := right.(type) {
		case int64:
			if r == 0 {
				return nil, errors.NewValidationError("DIVISION_BY_ZERO", "division by zero")
			}
			return l / r, nil
		case float64:
			if r == 0.0 {
				return nil, errors.NewValidationError("DIVISION_BY_ZERO", "division by zero")
			}
			return float64(l) / r, nil
		}
	case float64:
		switch r := right.(type) {
		case int64:
			if r == 0 {
				return nil, errors.NewValidationError("DIVISION_BY_ZERO", "division by zero")
			}
			return l / float64(r), nil
		case float64:
			if r == 0.0 {
				return nil, errors.NewValidationError("DIVISION_BY_ZERO", "division by zero")
			}
			return l / r, nil
		}
	}

	return nil, errors.NewValidationError("OPERAND_TYPE_MISMATCH", "division requires numeric operands")
}

// executeArithmeticModulo handles modulo operation
func (e *ExecutionEngine) executeArithmeticModulo(left, right interface{}) (interface{}, error) {
	switch l := left.(type) {
	case int64:
		switch r := right.(type) {
		case int64:
			if r == 0 {
				return nil, errors.NewValidationError("MODULO_BY_ZERO", "modulo by zero")
			}
			return l % r, nil
		case float64:
			if r == 0.0 {
				return nil, errors.NewValidationError("MODULO_BY_ZERO", "modulo by zero")
			}
			// Check if the float64 represents a whole number
			if r == float64(int64(r)) {
				return l % int64(r), nil
			}
			return nil, errors.NewValidationError("OPERAND_TYPE_MISMATCH", "modulo requires integer operands")
		}
	case float64:
		switch r := right.(type) {
		case int64:
			if r == 0 {
				return nil, errors.NewValidationError("MODULO_BY_ZERO", "modulo by zero")
			}
			// Check if the float64 represents a whole number
			if l == float64(int64(l)) {
				return int64(l) % r, nil
			}
			return nil, errors.NewValidationError("OPERAND_TYPE_MISMATCH", "modulo requires integer operands")
		case float64:
			if r == 0.0 {
				return nil, errors.NewValidationError("MODULO_BY_ZERO", "modulo by zero")
			}
			// Check if both float64 values represent whole numbers
			if l == float64(int64(l)) && r == float64(int64(r)) {
				return int64(l) % int64(r), nil
			}
			return nil, errors.NewValidationError("OPERAND_TYPE_MISMATCH", "modulo requires integer operands")
		}
	}

	return nil, errors.NewValidationError("OPERAND_TYPE_MISMATCH", "modulo requires integer operands")
}

// executeComparisonEqual handles equality comparison
func (e *ExecutionEngine) executeComparisonEqual(left, right interface{}) (interface{}, error) {
	// Handle different types with type conversion for numeric types
	switch l := left.(type) {
	case int64:
		switch r := right.(type) {
		case int64:
			return l == r, nil
		case float64:
			return float64(l) == r, nil
		case uint64:
			return uint64(l) == r, nil
		}
	case float64:
		switch r := right.(type) {
		case int64:
			return l == float64(r), nil
		case float64:
			return l == r, nil
		case uint64:
			return l == float64(r), nil
		}
	case uint64:
		switch r := right.(type) {
		case int64:
			return l == uint64(r), nil
		case float64:
			return float64(l) == r, nil
		case uint64:
			return l == r, nil
		}
	case string:
		if rStr, ok := right.(string); ok {
			return l == rStr, nil
		}
	case bool:
		if rBool, ok := right.(bool); ok {
			return l == rBool, nil
		}
	case *shared.BitstringObject:
		if rBs, ok := right.(*shared.BitstringObject); ok {
			lBytes := l.BitString.ToBytes()
			rBytes := rBs.BitString.ToBytes()
			if len(lBytes) != len(rBytes) {
				return false, nil
			}
			for i := range lBytes {
				if lBytes[i] != rBytes[i] {
					return false, nil
				}
			}
			return true, nil
		}
	}

	// For different types, they are not equal
	return false, nil
}

// executeComparisonNotEqual handles inequality comparison
func (e *ExecutionEngine) executeComparisonNotEqual(left, right interface{}) (interface{}, error) {
	equal, err := e.executeComparisonEqual(left, right)
	if err != nil {
		return nil, err
	}
	return !(equal.(bool)), nil
}

// executeComparisonLess handles less than comparison
func (e *ExecutionEngine) executeComparisonLess(left, right interface{}) (interface{}, error) {
	switch l := left.(type) {
	case int64:
		switch r := right.(type) {
		case int64:
			return l < r, nil
		case float64:
			return float64(l) < r, nil
		case uint64:
			return uint64(l) < r, nil
		}
	case float64:
		switch r := right.(type) {
		case int64:
			return l < float64(r), nil
		case float64:
			return l < r, nil
		case uint64:
			return l < float64(r), nil
		}
	case uint64:
		switch r := right.(type) {
		case int64:
			return l < uint64(r), nil
		case float64:
			return float64(l) < r, nil
		case uint64:
			return l < r, nil
		}
	case string:
		if rStr, ok := right.(string); ok {
			return l < rStr, nil
		}
	}

	return nil, errors.NewValidationError("OPERAND_TYPE_MISMATCH", "less than comparison requires comparable operands of the same type")
}

// executeComparisonLessEqual handles less than or equal comparison
func (e *ExecutionEngine) executeComparisonLessEqual(left, right interface{}) (interface{}, error) {
	switch l := left.(type) {
	case int64:
		switch r := right.(type) {
		case int64:
			return l <= r, nil
		case float64:
			return float64(l) <= r, nil
		case uint64:
			return uint64(l) <= r, nil
		}
	case float64:
		switch r := right.(type) {
		case int64:
			return l <= float64(r), nil
		case float64:
			return l <= r, nil
		case uint64:
			return l <= float64(r), nil
		}
	case uint64:
		switch r := right.(type) {
		case int64:
			return l <= uint64(r), nil
		case float64:
			return float64(l) <= r, nil
		case uint64:
			return l <= r, nil
		}
	case string:
		if rStr, ok := right.(string); ok {
			return l <= rStr, nil
		}
	}

	return nil, errors.NewValidationError("OPERAND_TYPE_MISMATCH", "less than or equal comparison requires comparable operands of the same type")
}

// executeComparisonGreater handles greater than comparison
func (e *ExecutionEngine) executeComparisonGreater(left, right interface{}) (interface{}, error) {
	switch l := left.(type) {
	case int64:
		switch r := right.(type) {
		case int64:
			return l > r, nil
		case float64:
			return float64(l) > r, nil
		case uint64:
			return uint64(l) > r, nil
		}
	case float64:
		switch r := right.(type) {
		case int64:
			return l > float64(r), nil
		case float64:
			return l > r, nil
		case uint64:
			return l > float64(r), nil
		}
	case uint64:
		switch r := right.(type) {
		case int64:
			return l > uint64(r), nil
		case float64:
			return float64(l) > r, nil
		case uint64:
			return l > r, nil
		}
	case string:
		if rStr, ok := right.(string); ok {
			return l > rStr, nil
		}
	}

	return nil, errors.NewValidationError("OPERAND_TYPE_MISMATCH", "greater than comparison requires comparable operands of the same type")
}

// executeComparisonGreaterEqual handles greater than or equal comparison
func (e *ExecutionEngine) executeComparisonGreaterEqual(left, right interface{}) (interface{}, error) {
	switch l := left.(type) {
	case int64:
		switch r := right.(type) {
		case int64:
			return l >= r, nil
		case float64:
			return float64(l) >= r, nil
		case uint64:
			return uint64(l) >= r, nil
		}
	case float64:
		switch r := right.(type) {
		case int64:
			return l >= float64(r), nil
		case float64:
			return l >= r, nil
		case uint64:
			return l >= float64(r), nil
		}
	case uint64:
		switch r := right.(type) {
		case int64:
			return l >= uint64(r), nil
		case float64:
			return float64(l) >= r, nil
		case uint64:
			return l >= r, nil
		}
	case string:
		if rStr, ok := right.(string); ok {
			return l >= rStr, nil
		}
	}

	return nil, errors.NewValidationError("OPERAND_TYPE_MISMATCH", "greater than or equal comparison requires comparable operands of the same type")
}

// executeLogicalAndWithShortCircuit handles logical AND operation with short-circuiting
func (e *ExecutionEngine) executeLogicalAndWithShortCircuit(binaryExpr *ast.BinaryExpression) (interface{}, error) {
	// Evaluate left operand first
	leftValue, err := e.convertExpressionToValue(binaryExpr.Left)
	if err != nil {
		return nil, errors.NewSystemError("BINARY_EXPR_ERROR", fmt.Sprintf("failed to evaluate left operand: %v", err))
	}

	// Check if left operand is falsy - if so, short-circuit and return false without evaluating right
	if !e.isTruthy(leftValue) {
		return false, nil
	}

	// Left operand is truthy, evaluate and return the truthiness of right operand
	rightValue, err := e.convertExpressionToValue(binaryExpr.Right)
	if err != nil {
		return nil, errors.NewSystemError("BINARY_EXPR_ERROR", fmt.Sprintf("failed to evaluate right operand: %v", err))
	}

	return e.isTruthy(rightValue), nil
}

// executeLogicalOrWithShortCircuit handles logical OR operation with short-circuiting
func (e *ExecutionEngine) executeLogicalOrWithShortCircuit(binaryExpr *ast.BinaryExpression) (interface{}, error) {
	// Evaluate left operand first
	leftValue, err := e.convertExpressionToValue(binaryExpr.Left)
	if err != nil {
		return nil, errors.NewSystemError("BINARY_EXPR_ERROR", fmt.Sprintf("failed to evaluate left operand: %v", err))
	}

	// Check if left operand is truthy - if so, short-circuit and return true without evaluating right
	if e.isTruthy(leftValue) {
		return true, nil
	}

	// Left operand is falsy, evaluate and return the truthiness of right operand
	rightValue, err := e.convertExpressionToValue(binaryExpr.Right)
	if err != nil {
		return nil, errors.NewSystemError("BINARY_EXPR_ERROR", fmt.Sprintf("failed to evaluate right operand: %v", err))
	}

	return e.isTruthy(rightValue), nil
}

// executeLogicalAnd handles logical AND operation (kept for backward compatibility)
func (e *ExecutionEngine) executeLogicalAnd(left, right interface{}) (interface{}, error) {
	leftBool := e.isTruthy(left)
	if !leftBool {
		// Short-circuit: if left is falsy, result is falsy
		return false, nil
	}

	return e.isTruthy(right), nil
}

// executeLogicalOr handles logical OR operation (kept for backward compatibility)
func (e *ExecutionEngine) executeLogicalOr(left, right interface{}) (interface{}, error) {
	leftBool := e.isTruthy(left)
	if leftBool {
		// Short-circuit: if left is truthy, result is truthy
		return true, nil
	}

	return e.isTruthy(right), nil
}

// executeBitwiseAnd handles bitwise AND operation
func (e *ExecutionEngine) executeBitwiseAnd(left, right interface{}) (interface{}, error) {
	switch l := left.(type) {
	case int64:
		switch r := right.(type) {
		case int64:
			return l & r, nil
		case float64:
			// Check if the float64 represents a whole number
			if r == float64(int64(r)) {
				return l & int64(r), nil
			}
			return nil, errors.NewValidationError("OPERAND_TYPE_MISMATCH", "bitwise AND requires integer operands")
		case uint64:
			return uint64(l) & r, nil
		}
	case float64:
		// Check if the float64 represents a whole number
		if l != float64(int64(l)) {
			return nil, errors.NewValidationError("OPERAND_TYPE_MISMATCH", "bitwise AND requires integer operands")
		}
		switch r := right.(type) {
		case int64:
			return int64(l) & r, nil
		case float64:
			// Check if the float64 represents a whole number
			if r == float64(int64(r)) {
				return int64(l) & int64(r), nil
			}
			return nil, errors.NewValidationError("OPERAND_TYPE_MISMATCH", "bitwise AND requires integer operands")
		case uint64:
			return uint64(int64(l)) & r, nil
		}
	case uint64:
		switch r := right.(type) {
		case int64:
			return l & uint64(r), nil
		case float64:
			// Check if the float64 represents a whole number
			if r == float64(int64(r)) {
				return l & uint64(r), nil
			}
			return nil, errors.NewValidationError("OPERAND_TYPE_MISMATCH", "bitwise AND requires integer operands")
		case uint64:
			return l & r, nil
		}
	}

	return nil, errors.NewValidationError("OPERAND_TYPE_MISMATCH", "bitwise AND requires integer operands")
}

// executeBitwiseOr handles bitwise OR operation
func (e *ExecutionEngine) executeBitwiseOr(left, right interface{}) (interface{}, error) {
	switch l := left.(type) {
	case int64:
		switch r := right.(type) {
		case int64:
			return l | r, nil
		case float64:
			// Check if the float64 represents a whole number
			if r == float64(int64(r)) {
				return l | int64(r), nil
			}
			return nil, errors.NewValidationError("OPERAND_TYPE_MISMATCH", "bitwise OR requires integer operands")
		}
	case float64:
		// Check if the float64 represents a whole number
		if l != float64(int64(l)) {
			return nil, errors.NewValidationError("OPERAND_TYPE_MISMATCH", "bitwise OR requires integer operands")
		}
		switch r := right.(type) {
		case int64:
			return int64(l) | r, nil
		case float64:
			// Check if the float64 represents a whole number
			if r == float64(int64(r)) {
				return int64(l) | int64(r), nil
			}
			return nil, errors.NewValidationError("OPERAND_TYPE_MISMATCH", "bitwise OR requires integer operands")
		}
	}

	return nil, errors.NewValidationError("OPERAND_TYPE_MISMATCH", "bitwise OR requires integer operands")
}

// executeBitwiseXor handles bitwise XOR operation
func (e *ExecutionEngine) executeBitwiseXor(left, right interface{}) (interface{}, error) {
	switch l := left.(type) {
	case int64:
		switch r := right.(type) {
		case int64:
			return l ^ r, nil
		case float64:
			// Check if the float64 represents a whole number
			if r == float64(int64(r)) {
				return l ^ int64(r), nil
			}
			return nil, errors.NewValidationError("OPERAND_TYPE_MISMATCH", "bitwise XOR requires integer operands")
		}
	case float64:
		// Check if the float64 represents a whole number
		if l != float64(int64(l)) {
			return nil, errors.NewValidationError("OPERAND_TYPE_MISMATCH", "bitwise XOR requires integer operands")
		}
		switch r := right.(type) {
		case int64:
			return int64(l) ^ r, nil
		case float64:
			// Check if the float64 represents a whole number
			if r == float64(int64(r)) {
				return int64(l) ^ int64(r), nil
			}
			return nil, errors.NewValidationError("OPERAND_TYPE_MISMATCH", "bitwise XOR requires integer operands")
		}
	}

	return nil, errors.NewValidationError("OPERAND_TYPE_MISMATCH", "bitwise XOR requires integer operands")
}

// executeBitwiseLeftShift handles bitwise left shift operation
func (e *ExecutionEngine) executeBitwiseLeftShift(left, right interface{}) (interface{}, error) {
	switch l := left.(type) {
	case int64:
		switch r := right.(type) {
		case int64:
			return l << r, nil
		case float64:
			// Check if the float64 represents a whole number
			if r == float64(int64(r)) {
				return l << int64(r), nil
			}
			return nil, errors.NewValidationError("OPERAND_TYPE_MISMATCH", "bitwise left shift requires integer operands")
		}
	case float64:
		// Check if the float64 represents a whole number
		if l != float64(int64(l)) {
			return nil, errors.NewValidationError("OPERAND_TYPE_MISMATCH", "bitwise left shift requires integer operands")
		}
		switch r := right.(type) {
		case int64:
			return int64(l) << r, nil
		case float64:
			// Check if the float64 represents a whole number
			if r == float64(int64(r)) {
				return int64(l) << int64(r), nil
			}
			return nil, errors.NewValidationError("OPERAND_TYPE_MISMATCH", "bitwise left shift requires integer operands")
		}
	}

	return nil, errors.NewValidationError("OPERAND_TYPE_MISMATCH", "bitwise left shift requires integer operands")
}

// executeBitwiseRightShift handles bitwise right shift operation
func (e *ExecutionEngine) executeBitwiseRightShift(left, right interface{}) (interface{}, error) {
	switch l := left.(type) {
	case int64:
		switch r := right.(type) {
		case int64:
			return l >> r, nil
		case float64:
			// Check if the float64 represents a whole number
			if r == float64(int64(r)) {
				return l >> int64(r), nil
			}
			return nil, errors.NewValidationError("OPERAND_TYPE_MISMATCH", "bitwise right shift requires integer operands")
		case uint64:
			return l >> uint64(r), nil
		}
	case float64:
		// Check if the float64 represents a whole number
		if l != float64(int64(l)) {
			return nil, errors.NewValidationError("OPERAND_TYPE_MISMATCH", "bitwise right shift requires integer operands")
		}
		switch r := right.(type) {
		case int64:
			return int64(l) >> r, nil
		case float64:
			// Check if the float64 represents a whole number
			if r == float64(int64(r)) {
				return int64(l) >> int64(r), nil
			}
			return nil, errors.NewValidationError("OPERAND_TYPE_MISMATCH", "bitwise right shift requires integer operands")
		case uint64:
			return int64(l) >> uint64(r), nil
		}
	case uint64:
		switch r := right.(type) {
		case int64:
			return l >> uint64(r), nil
		case float64:
			// Check if the float64 represents a whole number
			if r == float64(int64(r)) {
				return l >> uint64(r), nil
			}
			return nil, errors.NewValidationError("OPERAND_TYPE_MISMATCH", "bitwise right shift requires integer operands")
		case uint64:
			return l >> r, nil
		}
	}

	return nil, errors.NewValidationError("OPERAND_TYPE_MISMATCH", "bitwise right shift requires integer operands")
}

// executeStringConcat handles string concatenation operation
func (e *ExecutionEngine) executeStringConcat(left, right interface{}) (interface{}, error) {
	// Convert left operand to string
	leftStr := e.convertToString(left)
	
	// Convert right operand to string  
	rightStr := e.convertToString(right)
	
	return leftStr + rightStr, nil
}

// convertToString converts various types to string for concatenation
func (e *ExecutionEngine) convertToString(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case int:
		return fmt.Sprintf("%d", v)
	case int64:
		return fmt.Sprintf("%d", v)
	case float64:
		// Format floats as specified: 3.14159 -> "3.14159", 42.0 -> "42.0"
		return fmt.Sprintf("%g", v)
	case bool:
		if v {
			return "true"
		}
		return "false"
	default:
		return fmt.Sprintf("%v", value)
	}
}

// executeUnaryExpression executes a unary expression (e.g., -x, ~value)
func (e *ExecutionEngine) executeUnaryExpression(unaryExpr *ast.UnaryExpression) (interface{}, error) {
	// Evaluate the operand
	operandValue, err := e.convertExpressionToValue(unaryExpr.Right)
	if err != nil {
		return nil, errors.NewSystemError("UNARY_EXPR_ERROR", fmt.Sprintf("failed to evaluate operand: %v", err))
	}

	// Handle different operators
	switch unaryExpr.Operator {
	case "-":
		return e.executeUnaryNegation(operandValue)
	case "~":
		return e.executeUnaryBitwiseNot(operandValue)
	default:
		return nil, errors.NewSystemError("UNSUPPORTED_OPERATOR", fmt.Sprintf("unsupported unary operator: %s", unaryExpr.Operator))
	}
}

// executeUnaryNegation handles numeric negation
func (e *ExecutionEngine) executeUnaryNegation(operand interface{}) (interface{}, error) {
	switch v := operand.(type) {
	case int64:
		return -v, nil
	case float64:
		return -v, nil
	}

	return nil, errors.NewValidationError("OPERAND_TYPE_MISMATCH", "negation requires numeric operand")
}

// executeUnaryBitwiseNot handles bitwise NOT operation
func (e *ExecutionEngine) executeUnaryBitwiseNot(operand interface{}) (interface{}, error) {
	switch v := operand.(type) {
	case int64:
		return ^v, nil
	case float64:
		// Check if the float64 represents a whole number
		if v == float64(int64(v)) {
			return ^int64(v), nil
		}
		return nil, errors.NewValidationError("OPERAND_TYPE_MISMATCH", "bitwise NOT requires integer operand")
	}

	return nil, errors.NewValidationError("OPERAND_TYPE_MISMATCH", "bitwise NOT requires integer operand")
}

// executeElvisExpression executes an Elvis expression (ternary operator ?:)
func (e *ExecutionEngine) executeElvisExpression(elvisExpr *ast.ElvisExpression) (interface{}, error) {
	// Evaluate the left operand (condition)
	leftValue, err := e.convertExpressionToValue(elvisExpr.Left)
	if err != nil {
		return nil, errors.NewSystemError("ELVIS_EXPR_ERROR", fmt.Sprintf("failed to evaluate condition: %v", err))
	}

	// Check if condition is truthy
	if e.isTruthy(leftValue) {
		// Return left value if truthy
		return leftValue, nil
	} else {
		// Evaluate and return right value if falsy
		rightValue, err := e.convertExpressionToValue(elvisExpr.Right)
		if err != nil {
			return nil, errors.NewSystemError("ELVIS_EXPR_ERROR", fmt.Sprintf("failed to evaluate alternative: %v", err))
		}
		return rightValue, nil
	}
}

// executeTernaryExpression executes a ternary expression (condition ? true_expr : false_expr)
func (e *ExecutionEngine) executeTernaryExpression(ternaryExpr *ast.TernaryExpression) (interface{}, error) {
	// Evaluate the condition
	conditionValue, err := e.convertExpressionToValue(ternaryExpr.Condition)
	if err != nil {
		return nil, errors.NewSystemError("TERNARY_EXPR_ERROR", fmt.Sprintf("failed to evaluate condition: %v", err))
	}

	if e.verbose {
		fmt.Printf("DEBUG: executeTernaryExpression - condition value: %v (type: %T), truthy: %v\n", conditionValue, conditionValue, e.isTruthy(conditionValue))
	}

	// Check if condition is truthy
	if e.isTruthy(conditionValue) {
		// Evaluate and return true expression
		trueValue, err := e.convertExpressionToValue(ternaryExpr.TrueExpr)
		if err != nil {
			return nil, errors.NewSystemError("TERNARY_EXPR_ERROR", fmt.Sprintf("failed to evaluate true expression: %v", err))
		}
		if e.verbose {
			fmt.Printf("DEBUG: executeTernaryExpression - returning true value: %v (type: %T)\n", trueValue, trueValue)
		}
		return trueValue, nil
	} else {
		// Evaluate and return false expression
		falseValue, err := e.convertExpressionToValue(ternaryExpr.FalseExpr)
		if err != nil {
			return nil, errors.NewSystemError("TERNARY_EXPR_ERROR", fmt.Sprintf("failed to evaluate false expression: %v", err))
		}
		if e.verbose {
			fmt.Printf("DEBUG: executeTernaryExpression - returning false value: %v (type: %T)\n", falseValue, falseValue)
		}
		return falseValue, nil
	}
}

// convertValueToExpression converts a Go value back to an AST Expression node
// This is needed for pipe operations where the output of one stage becomes input to the next
func (e *ExecutionEngine) convertValueToExpression(value interface{}) (ast.Expression, error) {
	if value == nil {
		return &ast.StringLiteral{Value: "", Pos: ast.Position{Line: 1, Column: 1, Offset: 0}}, nil
	}

	switch v := value.(type) {
	case string:
		return &ast.StringLiteral{Value: v, Pos: ast.Position{Line: 1, Column: 1, Offset: 0}}, nil
	case int:
		return &ast.NumberLiteral{Value: float64(v), Pos: ast.Position{Line: 1, Column: 1, Offset: 0}}, nil
	case int64:
		return &ast.NumberLiteral{Value: float64(v), Pos: ast.Position{Line: 1, Column: 1, Offset: 0}}, nil
	case float64:
		return &ast.NumberLiteral{Value: v, Pos: ast.Position{Line: 1, Column: 1, Offset: 0}}, nil
	case bool:
		return &ast.BooleanLiteral{Value: v, Pos: ast.Position{Line: 1, Column: 1, Offset: 0}}, nil
	case []interface{}:
		elements := make([]ast.Expression, len(v))
		for i, elem := range v {
			expr, err := e.convertValueToExpression(elem)
			if err != nil {
				return nil, fmt.Errorf("failed to convert array element at index %d: %v", i, err)
			}
			elements[i] = expr
		}
		// Create dummy tokens for array brackets
		leftBracket := lexer.Token{Type: lexer.TokenLBracket, Value: "[", Position: 0, Line: 1, Column: 1}
		rightBracket := lexer.Token{Type: lexer.TokenRBracket, Value: "]", Position: 0, Line: 1, Column: 1}
		arrayLiteral := ast.NewArrayLiteral(leftBracket, rightBracket)
		arrayLiteral.Elements = elements
		return arrayLiteral, nil
	case map[string]interface{}:
		// Create dummy tokens for object braces
		leftBrace := lexer.Token{Type: lexer.TokenLBrace, Value: "{", Position: 0, Line: 1, Column: 1}
		rightBrace := lexer.Token{Type: lexer.TokenRBrace, Value: "}", Position: 0, Line: 1, Column: 1}
		objectLiteral := ast.NewObjectLiteral(leftBrace, rightBrace)

		for key, val := range v {
			valExpr, err := e.convertValueToExpression(val)
			if err != nil {
				return nil, fmt.Errorf("failed to convert object property '%s': %v", key, err)
			}
			// Create a simple identifier for the key
			keyIdent := &ast.Identifier{Name: key, Pos: ast.Position{Line: 1, Column: 1, Offset: 0}}
			objectLiteral.AddProperty(keyIdent, valExpr)
		}
		return objectLiteral, nil
	default:
		return nil, fmt.Errorf("unsupported type for conversion to expression: %T", value)
	}
}

// executePipeExpression executes a pipeline of commands where output of one becomes input to the next
func (e *ExecutionEngine) executePipeExpression(pipeExpr *ast.PipeExpression) (interface{}, error) {
	if len(pipeExpr.Stages) == 0 {
		return nil, errors.NewValidationError("EMPTY_PIPELINE", "pipeline must have at least one stage")
	}

	// Execute the first stage to get the initial value
	firstStage, ok := pipeExpr.Stages[0].(*ast.LanguageCall)
	if !ok {
		return nil, errors.NewValidationError("INVALID_STAGE_TYPE", "first stage must be a language call")
	}

	result, err := e.executeLanguageCallNew(firstStage)
	if err != nil {
		return nil, errors.NewSystemError("PIPELINE_ERROR", fmt.Sprintf("first stage failed: %v", err))
	}

	// Process remaining stages
	for i := 1; i < len(pipeExpr.Stages); i++ {
		stage, ok := pipeExpr.Stages[i].(*ast.LanguageCall)
		if !ok {
			return nil, errors.NewValidationError("INVALID_STAGE_TYPE", fmt.Sprintf("stage %d must be a language call", i+1))
		}

		if e.verbose {
			fmt.Printf("DEBUG: Pipe stage %d: %s.%s, input result: %v\n", i+1, stage.Language, stage.Function, result)
		}

		// Convert the previous result to an expression to use as the first argument
		pipedValueExpr, err := e.convertValueToExpression(result)
		if err != nil {
			return nil, errors.NewSystemError("PIPELINE_ERROR", fmt.Sprintf("failed to convert piped value for stage %d: %v", i+1, err))
		}

		// Create new arguments list with piped value as first argument, followed by existing arguments
		newArguments := []ast.Expression{pipedValueExpr}
		newArguments = append(newArguments, stage.Arguments...)

		// Create a new LanguageCall with the modified arguments (immutability constraint)
		modifiedStage := &ast.LanguageCall{
			Language:  stage.Language,
			Function:  stage.Function,
			Arguments: newArguments,
			Pos:       stage.Pos,
		}

		if e.verbose {
			fmt.Printf("DEBUG: Modified stage %d: %s.%s with %d arguments\n", i+1, modifiedStage.Language, modifiedStage.Function, len(modifiedStage.Arguments))
		}

		// Execute the modified stage
		result, err = e.executeLanguageCallNew(modifiedStage)
		if err != nil {
			// Fail-fast: immediately return error on any stage failure
			return nil, errors.NewSystemError("PIPELINE_ERROR", fmt.Sprintf("stage %d failed: %v", i+1, err))
		}

		if e.verbose {
			fmt.Printf("DEBUG: Pipe stage %d result: %v\n", i+1, result)
		}

		// Special debug for python.add_one calls
		if i == 3 && modifiedStage.Language == "python" && modifiedStage.Function == "add_one" && result == nil {
			if e.verbose {
				fmt.Printf("DEBUG: Detected python.add_one failure - checking if function exists\n")
			}
			// Check if the function still exists in Python global scope
			if rt, err := e.runtimeManager.GetRuntime("python"); err == nil {
				if checkResult, checkErr := rt.Eval("print('add_one' in globals())"); checkErr == nil {
					if e.verbose {
						fmt.Printf("DEBUG: add_one in globals: %v\n", checkResult)
					}
				}
				// Try to call the function directly
				if testResult, testErr := rt.Eval("print(add_one(99) if 'add_one' in globals() else 'Function not found')"); testErr == nil {
					if e.verbose {
						fmt.Printf("DEBUG: Direct add_one(99) test: %v\n", testResult)
					}
				}
			}
		}
	}

	return result, nil
}
