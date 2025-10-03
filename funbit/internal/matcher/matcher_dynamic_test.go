package matcher

import (
	"testing"

	bitstringpkg "github.com/funvibe/funbit/internal/bitstring"
)

// Tests for DynamicSizeContext and dynamic size evaluation functions
func TestDynamicSizeContext_AddVariable(t *testing.T) {
	t.Run("Add and get variable", func(t *testing.T) {
		ctx := NewDynamicSizeContext()

		// Add a variable
		ctx.AddVariable("test_var", 42)

		// Get the variable
		value, exists := ctx.GetVariable("test_var")

		if !exists {
			t.Error("Expected variable to exist")
		}

		if value != 42 {
			t.Errorf("Expected value 42, got %d", value)
		}
	})

	t.Run("Get non-existent variable", func(t *testing.T) {
		ctx := NewDynamicSizeContext()

		value, exists := ctx.GetVariable("non_existent")

		if exists {
			t.Error("Expected variable to not exist")
		}

		if value != 0 {
			t.Errorf("Expected default value 0, got %d", value)
		}
	})
}

func TestMatcher_EvaluateDynamicSize(t *testing.T) {
	m := NewMatcher()

	t.Run("Static size segment", func(t *testing.T) {
		segment := &bitstringpkg.Segment{
			Size:      32,
			IsDynamic: false,
		}
		context := NewDynamicSizeContext()

		size, err := m.EvaluateDynamicSize(segment, context)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if size != 32 {
			t.Errorf("Expected size 32, got %d", size)
		}
	})

	t.Run("Dynamic size with variable reference", func(t *testing.T) {
		dynamicSize := uint(64)
		segment := &bitstringpkg.Segment{
			IsDynamic:   true,
			DynamicSize: &dynamicSize,
		}
		context := NewDynamicSizeContext()

		size, err := m.EvaluateDynamicSize(segment, context)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if size != 64 {
			t.Errorf("Expected size 64, got %d", size)
		}
	})

	t.Run("Dynamic size with expression", func(t *testing.T) {
		segment := &bitstringpkg.Segment{
			IsDynamic:   true,
			DynamicExpr: "2 * 16",
		}
		context := NewDynamicSizeContext()

		size, err := m.EvaluateDynamicSize(segment, context)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if size != 32 {
			t.Errorf("Expected size 32, got %d", size)
		}
	})

	t.Run("Dynamic size without variable or expression", func(t *testing.T) {
		segment := &bitstringpkg.Segment{
			IsDynamic: true,
		}
		context := NewDynamicSizeContext()

		_, err := m.EvaluateDynamicSize(segment, context)

		if err == nil {
			t.Error("Expected error for dynamic size without variable or expression")
		}

		expectedError := "dynamic size specified but no variable or expression provided"
		if err.Error() != expectedError {
			t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
		}
	})
}

func TestMatcher_EvaluateExpression(t *testing.T) {
	m := NewMatcher()
	context := NewDynamicSizeContext()

	t.Run("Simple arithmetic", func(t *testing.T) {
		testCases := []struct {
			expr     string
			expected uint
		}{
			{"2 + 3", 5},
			{"10 - 4", 6},
			{"3 * 4", 12},
			{"20 / 4", 5},
			{"(2 + 3) * 4", 20},
			{"2 + 3 * 4", 14},
			{"(2 + 3) * (4 + 5)", 45},
		}

		for _, tc := range testCases {
			result, err := m.EvaluateExpression(tc.expr, context)

			if err != nil {
				t.Errorf("Expression '%s': expected no error, got %v", tc.expr, err)
				continue
			}

			if result != tc.expected {
				t.Errorf("Expression '%s': expected %d, got %d", tc.expr, tc.expected, result)
			}
		}
	})

	t.Run("Expression with variables", func(t *testing.T) {
		context.AddVariable("x", 10)
		context.AddVariable("y", 5)

		testCases := []struct {
			expr     string
			expected uint
		}{
			{"x + y", 15},
			{"x - y", 5},
			{"x * y", 50},
			{"x / y", 2},
			{"x + y * 2", 20},
			{"(x + y) * 2", 30},
		}

		for _, tc := range testCases {
			result, err := m.EvaluateExpression(tc.expr, context)

			if err != nil {
				t.Errorf("Expression '%s': expected no error, got %v", tc.expr, err)
				continue
			}

			if result != tc.expected {
				t.Errorf("Expression '%s': expected %d, got %d", tc.expr, tc.expected, result)
			}
		}
	})

	t.Run("Empty expression", func(t *testing.T) {
		_, err := m.EvaluateExpression("", context)

		if err == nil {
			t.Error("Expected error for empty expression")
		}

		expectedError := "empty expression"
		if err.Error() != expectedError {
			t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
		}
	})

	t.Run("Invalid expression syntax", func(t *testing.T) {
		_, err := m.EvaluateExpression("2 + * 3", context)

		if err == nil {
			t.Error("Expected error for invalid expression syntax")
		}
	})

	t.Run("Undefined variable", func(t *testing.T) {
		_, err := m.EvaluateExpression("undefined_var + 5", context)

		if err == nil {
			t.Error("Expected error for undefined variable")
		}

		expectedError := "evaluation error: undefined variable: undefined_var"
		if err.Error() != expectedError {
			t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
		}
	})

	t.Run("Division by zero", func(t *testing.T) {
		_, err := m.EvaluateExpression("10 / 0", context)

		if err == nil {
			t.Error("Expected error for division by zero")
		}

		expectedError := "evaluation error: division by zero"
		if err.Error() != expectedError {
			t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
		}
	})

	t.Run("Underflow in subtraction", func(t *testing.T) {
		_, err := m.EvaluateExpression("5 - 10", context)

		if err == nil {
			t.Error("Expected error for underflow in subtraction")
		}

		expectedError := "evaluation error: underflow in subtraction: 5 - 10"
		if err.Error() != expectedError {
			t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
		}
	})
}

func TestMatcher_tokenizeExpression(t *testing.T) {
	m := NewMatcher()

	t.Run("Simple expression", func(t *testing.T) {
		tokens := m.tokenizeExpression("2 + 3")

		expected := []string{"2", "+", "3"}

		if len(tokens) != len(expected) {
			t.Errorf("Expected %d tokens, got %d", len(expected), len(tokens))
			return
		}

		for i, token := range tokens {
			if token != expected[i] {
				t.Errorf("Token %d: expected '%s', got '%s'", i, expected[i], token)
			}
		}
	})

	t.Run("Complex expression with spaces", func(t *testing.T) {
		tokens := m.tokenizeExpression("( 2 + 3 ) * 4")

		expected := []string{"(", "2", "+", "3", ")", "*", "4"}

		if len(tokens) != len(expected) {
			t.Errorf("Expected %d tokens, got %d", len(expected), len(tokens))
			return
		}

		for i, token := range tokens {
			if token != expected[i] {
				t.Errorf("Token %d: expected '%s', got '%s'", i, expected[i], token)
			}
		}
	})

	t.Run("Expression with variables", func(t *testing.T) {
		tokens := m.tokenizeExpression("var1 + var2 * 10")

		expected := []string{"var1", "+", "var2", "*", "10"}

		if len(tokens) != len(expected) {
			t.Errorf("Expected %d tokens, got %d", len(expected), len(tokens))
			return
		}

		for i, token := range tokens {
			if token != expected[i] {
				t.Errorf("Token %d: expected '%s', got '%s'", i, expected[i], token)
			}
		}
	})

	t.Run("Empty expression", func(t *testing.T) {
		tokens := m.tokenizeExpression("")

		if len(tokens) != 0 {
			t.Errorf("Expected empty token list, got %v", tokens)
		}
	})

	t.Run("Expression with underscores in variable names", func(t *testing.T) {
		tokens := m.tokenizeExpression("my_var + _test")

		expected := []string{"my_var", "+", "_test"}

		if len(tokens) != len(expected) {
			t.Errorf("Expected %d tokens, got %d", len(expected), len(tokens))
			return
		}

		for i, token := range tokens {
			if token != expected[i] {
				t.Errorf("Token %d: expected '%s', got '%s'", i, expected[i], token)
			}
		}
	})
}

func TestMatcher_infixToPostfix(t *testing.T) {
	m := NewMatcher()

	t.Run("Simple expression", func(t *testing.T) {
		tokens := []string{"2", "+", "3"}
		postfix, err := m.infixToPostfix(tokens)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
			return
		}

		expected := []string{"2", "3", "+"}

		if len(postfix) != len(expected) {
			t.Errorf("Expected %d tokens, got %d", len(expected), len(postfix))
			return
		}

		for i, token := range postfix {
			if token != expected[i] {
				t.Errorf("Token %d: expected '%s', got '%s'", i, expected[i], token)
			}
		}
	})

	t.Run("Expression with precedence", func(t *testing.T) {
		tokens := []string{"2", "+", "3", "*", "4"}
		postfix, err := m.infixToPostfix(tokens)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
			return
		}

		expected := []string{"2", "3", "4", "*", "+"}

		if len(postfix) != len(expected) {
			t.Errorf("Expected %d tokens, got %d", len(expected), len(postfix))
			return
		}

		for i, token := range postfix {
			if token != expected[i] {
				t.Errorf("Token %d: expected '%s', got '%s'", i, expected[i], token)
			}
		}
	})

	t.Run("Expression with parentheses", func(t *testing.T) {
		tokens := []string{"(", "2", "+", "3", ")", "*", "4"}
		postfix, err := m.infixToPostfix(tokens)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
			return
		}

		expected := []string{"2", "3", "+", "4", "*"}

		if len(postfix) != len(expected) {
			t.Errorf("Expected %d tokens, got %d", len(expected), len(postfix))
			return
		}

		for i, token := range postfix {
			if token != expected[i] {
				t.Errorf("Token %d: expected '%s', got '%s'", i, expected[i], token)
			}
		}
	})

	t.Run("Mismatched opening parenthesis", func(t *testing.T) {
		tokens := []string{"(", "2", "+", "3"}
		_, err := m.infixToPostfix(tokens)

		if err == nil {
			t.Error("Expected error for mismatched parentheses")
		}

		expectedError := "mismatched parentheses"
		if err.Error() != expectedError {
			t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
		}
	})

	t.Run("Mismatched closing parenthesis", func(t *testing.T) {
		tokens := []string{"2", "+", "3", ")"}
		_, err := m.infixToPostfix(tokens)

		if err == nil {
			t.Error("Expected error for mismatched parentheses")
		}

		expectedError := "mismatched parentheses"
		if err.Error() != expectedError {
			t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
		}
	})

	t.Run("Invalid token", func(t *testing.T) {
		tokens := []string{"2", "+", "#"}
		_, err := m.infixToPostfix(tokens)

		if err == nil {
			t.Error("Expected error for invalid token")
		}

		expectedError := "invalid token: #"
		if err.Error() != expectedError {
			t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
		}
	})
}

func TestMatcher_evaluatePostfix(t *testing.T) {
	m := NewMatcher()
	context := NewDynamicSizeContext()

	t.Run("Simple arithmetic", func(t *testing.T) {
		testCases := []struct {
			postfix  []string
			expected uint
		}{
			{[]string{"2", "3", "+"}, 5},
			{[]string{"10", "4", "-"}, 6},
			{[]string{"3", "4", "*"}, 12},
			{[]string{"20", "4", "/"}, 5},
			{[]string{"2", "3", "+", "4", "*"}, 20},
			{[]string{"2", "3", "4", "*", "+"}, 14},
		}

		for _, tc := range testCases {
			result, err := m.evaluatePostfix(tc.postfix, context)

			if err != nil {
				t.Errorf("Postfix %v: expected no error, got %v", tc.postfix, err)
				continue
			}

			if result != tc.expected {
				t.Errorf("Postfix %v: expected %d, got %d", tc.postfix, tc.expected, result)
			}
		}
	})

	t.Run("Postfix with variables", func(t *testing.T) {
		context.AddVariable("x", 10)
		context.AddVariable("y", 5)

		testCases := []struct {
			postfix  []string
			expected uint
		}{
			{[]string{"x", "y", "+"}, 15},
			{[]string{"x", "y", "-"}, 5},
			{[]string{"x", "y", "*"}, 50},
			{[]string{"x", "y", "/"}, 2},
		}

		for _, tc := range testCases {
			result, err := m.evaluatePostfix(tc.postfix, context)

			if err != nil {
				t.Errorf("Postfix %v: expected no error, got %v", tc.postfix, err)
				continue
			}

			if result != tc.expected {
				t.Errorf("Postfix %v: expected %d, got %d", tc.postfix, tc.expected, result)
			}
		}
	})

	t.Run("Invalid number that looks like variable", func(t *testing.T) {
		postfix := []string{"invalid", "3", "+"}
		_, err := m.evaluatePostfix(postfix, context)

		if err == nil {
			t.Error("Expected error for invalid token")
		}

		// Since "invalid" matches variable format, it's treated as undefined variable
		expectedError := "undefined variable: invalid"
		if err.Error() != expectedError {
			t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
		}
	})

	t.Run("Undefined variable", func(t *testing.T) {
		postfix := []string{"undefined_var", "5", "+"}
		_, err := m.evaluatePostfix(postfix, context)

		if err == nil {
			t.Error("Expected error for undefined variable")
		}

		expectedError := "undefined variable: undefined_var"
		if err.Error() != expectedError {
			t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
		}
	})

	t.Run("Insufficient operands", func(t *testing.T) {
		postfix := []string{"2", "+"}
		_, err := m.evaluatePostfix(postfix, context)

		if err == nil {
			t.Error("Expected error for insufficient operands")
		}

		expectedError := "insufficient operands for operator: +"
		if err.Error() != expectedError {
			t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
		}
	})

	t.Run("Division by zero", func(t *testing.T) {
		postfix := []string{"10", "0", "/"}
		_, err := m.evaluatePostfix(postfix, context)

		if err == nil {
			t.Error("Expected error for division by zero")
		}

		expectedError := "division by zero"
		if err.Error() != expectedError {
			t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
		}
	})

	t.Run("Underflow in subtraction", func(t *testing.T) {
		postfix := []string{"5", "10", "-"}
		_, err := m.evaluatePostfix(postfix, context)

		if err == nil {
			t.Error("Expected error for underflow in subtraction")
		}

		expectedError := "underflow in subtraction: 5 - 10"
		if err.Error() != expectedError {
			t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
		}
	})

	t.Run("Invalid expression - multiple values on stack", func(t *testing.T) {
		postfix := []string{"2", "3"}
		_, err := m.evaluatePostfix(postfix, context)

		if err == nil {
			t.Error("Expected error for invalid expression")
		}

		expectedError := "invalid expression: 2 values left on stack"
		if err.Error() != expectedError {
			t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
		}
	})

	t.Run("Invalid token in postfix that looks like variable", func(t *testing.T) {
		postfix := []string{"2", "3", "invalid"}
		_, err := m.evaluatePostfix(postfix, context)

		if err == nil {
			t.Error("Expected error for invalid token")
		}

		// Since "invalid" matches variable format, it's treated as undefined variable
		expectedError := "undefined variable: invalid"
		if err.Error() != expectedError {
			t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
		}
	})
}

func TestMatcher_isNumber(t *testing.T) {
	m := NewMatcher()

	t.Run("Valid numbers", func(t *testing.T) {
		validNumbers := []string{"0", "1", "42", "123456", "18446744073709551615"} // max uint64

		for _, num := range validNumbers {
			if !m.isNumber(num) {
				t.Errorf("Expected '%s' to be recognized as a number", num)
			}
		}
	})

	t.Run("Invalid numbers", func(t *testing.T) {
		invalidNumbers := []string{"", "abc", "12a", "1.5", "-1", " 123", "123 "}

		for _, num := range invalidNumbers {
			if m.isNumber(num) {
				t.Errorf("Expected '%s' to NOT be recognized as a number", num)
			}
		}
	})
}

func TestMatcher_isVariable(t *testing.T) {
	m := NewMatcher()

	t.Run("Valid variables", func(t *testing.T) {
		validVariables := []string{"x", "var", "my_var", "_test", "var1", "a", "A", "_", "x_y_z"}

		for _, variable := range validVariables {
			if !m.isVariable(variable) {
				t.Errorf("Expected '%s' to be recognized as a variable", variable)
			}
		}
	})

	t.Run("Invalid variables", func(t *testing.T) {
		invalidVariables := []string{"", "1var", "var-name", "var.name", " var", "var ", "123", "+", "-", "*", "/"}

		for _, variable := range invalidVariables {
			if m.isVariable(variable) {
				t.Errorf("Expected '%s' to NOT be recognized as a variable", variable)
			}
		}
	})
}

func TestMatcher_isOperator(t *testing.T) {
	m := NewMatcher()

	t.Run("Valid operators", func(t *testing.T) {
		validOperators := []string{"+", "-", "*", "/"}

		for _, op := range validOperators {
			if !m.isOperator(op) {
				t.Errorf("Expected '%s' to be recognized as an operator", op)
			}
		}
	})

	t.Run("Invalid operators", func(t *testing.T) {
		invalidOperators := []string{"", "x", "1", "(", ")", "^", "%", "++", "--", " ", "="}

		for _, op := range invalidOperators {
			if m.isOperator(op) {
				t.Errorf("Expected '%s' to NOT be recognized as an operator", op)
			}
		}
	})
}
