package matcher

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	bitstringpkg "github.com/funvibe/funbit/internal/bitstring"
)

// DynamicSizeContext holds the context for evaluating dynamic sizes
type DynamicSizeContext struct {
	Variables map[string]uint
}

// NewDynamicSizeContext creates a new context for dynamic size evaluation
func NewDynamicSizeContext() *DynamicSizeContext {
	return &DynamicSizeContext{
		Variables: make(map[string]uint),
	}
}

// AddVariable adds a variable to the context
func (ctx *DynamicSizeContext) AddVariable(name string, value uint) {
	ctx.Variables[name] = value
}

// GetVariable gets a variable value from the context
func (ctx *DynamicSizeContext) GetVariable(name string) (uint, bool) {
	value, exists := ctx.Variables[name]
	return value, exists
}

// EvaluateDynamicSize evaluates the dynamic size for a segment
func (m *Matcher) EvaluateDynamicSize(segment *bitstringpkg.Segment, context *DynamicSizeContext) (uint, error) {
	if !segment.IsDynamic {
		return segment.Size, nil
	}

	// Handle dynamic size from variable reference
	if segment.DynamicSize != nil {
		return *segment.DynamicSize, nil
	}

	// Handle dynamic size from expression
	if segment.DynamicExpr != "" {
		return m.EvaluateExpression(segment.DynamicExpr, context)
	}

	return 0, errors.New("dynamic size specified but no variable or expression provided")
}

// EvaluateExpression evaluates a mathematical expression for dynamic size
func (m *Matcher) EvaluateExpression(expr string, context *DynamicSizeContext) (uint, error) {
	// Simple expression evaluator
	// Supports basic arithmetic: +, -, *, /
	// Supports variable references

	// Tokenize the expression
	tokens := m.tokenizeExpression(expr)
	if len(tokens) == 0 {
		return 0, errors.New("empty expression")
	}

	// Convert to postfix notation
	postfix, err := m.infixToPostfix(tokens)
	if err != nil {
		return 0, fmt.Errorf("invalid expression: %v", err)
	}

	// Evaluate postfix expression
	result, err := m.evaluatePostfix(postfix, context)
	if err != nil {
		return 0, fmt.Errorf("evaluation error: %v", err)
	}

	return result, nil
}

// tokenizeExpression tokenizes a mathematical expression
func (m *Matcher) tokenizeExpression(expr string) []string {
	// Remove whitespace
	expr = strings.ReplaceAll(expr, " ", "")

	// Simple regex to tokenize numbers, variables, and operators
	re := regexp.MustCompile(`([0-9]+|[a-zA-Z_][a-zA-Z0-9_]*|[+\-*/()])`)
	matches := re.FindAllString(expr, -1)

	return matches
}

// infixToPostfix converts infix notation to postfix notation (Shunting-yard algorithm)
func (m *Matcher) infixToPostfix(tokens []string) ([]string, error) {
	var output []string
	var operators []string

	precedence := map[string]int{
		"+": 1,
		"-": 1,
		"*": 2,
		"/": 2,
	}

	for _, token := range tokens {
		if m.isNumber(token) || m.isVariable(token) {
			output = append(output, token)
		} else if token == "(" {
			operators = append(operators, token)
		} else if token == ")" {
			// Pop operators until we find matching parenthesis
			for len(operators) > 0 && operators[len(operators)-1] != "(" {
				output = append(output, operators[len(operators)-1])
				operators = operators[:len(operators)-1]
			}

			if len(operators) == 0 {
				return nil, errors.New("mismatched parentheses")
			}

			// Remove the opening parenthesis
			operators = operators[:len(operators)-1]
		} else if m.isOperator(token) {
			// Pop operators with higher or equal precedence
			for len(operators) > 0 && operators[len(operators)-1] != "(" &&
				precedence[operators[len(operators)-1]] >= precedence[token] {
				output = append(output, operators[len(operators)-1])
				operators = operators[:len(operators)-1]
			}
			operators = append(operators, token)
		} else {
			return nil, fmt.Errorf("invalid token: %s", token)
		}
	}

	// Pop remaining operators
	for len(operators) > 0 {
		if operators[len(operators)-1] == "(" {
			return nil, errors.New("mismatched parentheses")
		}
		output = append(output, operators[len(operators)-1])
		operators = operators[:len(operators)-1]
	}

	return output, nil
}

// evaluatePostfix evaluates a postfix expression
func (m *Matcher) evaluatePostfix(postfix []string, context *DynamicSizeContext) (uint, error) {
	var stack []uint

	for _, token := range postfix {
		if m.isNumber(token) {
			value, err := strconv.ParseUint(token, 10, 64)
			if err != nil {
				return 0, fmt.Errorf("invalid number: %s", token)
			}
			stack = append(stack, uint(value))
		} else if m.isVariable(token) {
			value, exists := context.GetVariable(token)
			if !exists {
				return 0, fmt.Errorf("undefined variable: %s", token)
			}
			stack = append(stack, value)
		} else if m.isOperator(token) {
			if len(stack) < 2 {
				return 0, fmt.Errorf("insufficient operands for operator: %s", token)
			}

			// Pop two operands
			b := stack[len(stack)-1]
			a := stack[len(stack)-2]
			stack = stack[:len(stack)-2]

			var result uint
			var err error

			switch token {
			case "+":
				result = a + b
			case "-":
				if b > a {
					return 0, fmt.Errorf("underflow in subtraction: %d - %d", a, b)
				}
				result = a - b
			case "*":
				result = a * b
			case "/":
				if b == 0 {
					return 0, errors.New("division by zero")
				}
				result = a / b
			default:
				return 0, fmt.Errorf("unknown operator: %s", token)
			}

			if err != nil {
				return 0, err
			}

			stack = append(stack, result)
		} else {
			return 0, fmt.Errorf("invalid token in postfix: %s", token)
		}
	}

	if len(stack) != 1 {
		return 0, fmt.Errorf("invalid expression: %d values left on stack", len(stack))
	}

	return stack[0], nil
}

// Helper functions
func (m *Matcher) isNumber(token string) bool {
	_, err := strconv.ParseUint(token, 10, 64)
	return err == nil
}

func (m *Matcher) isVariable(token string) bool {
	if len(token) == 0 {
		return false
	}
	// Variables start with a letter or underscore
	if token[0] != '_' && (token[0] < 'a' || token[0] > 'z') && (token[0] < 'A' || token[0] > 'Z') {
		return false
	}
	// Rest can be letters, digits, or underscores
	for _, c := range token[1:] {
		if c != '_' && (c < '0' || c > '9') && (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') {
			return false
		}
	}
	return true
}

func (m *Matcher) isOperator(token string) bool {
	return token == "+" || token == "-" || token == "*" || token == "/"
}

// BuildContextFromPattern builds a dynamic size context from a pattern by extracting bound variables
func (m *Matcher) BuildContextFromPattern(pattern []*bitstringpkg.Segment, results []bitstringpkg.SegmentResult) (*DynamicSizeContext, error) {
	context := NewDynamicSizeContext()

	// Iterate through the pattern and results to extract bound variable values
	for i, segment := range pattern {
		if i >= len(results) {
			break
		}

		result := results[i]
		if !result.Matched {
			continue
		}

		// Extract variable name from segment.Value (assuming it's a pointer)
		varName := m.getVariableName(segment.Value)
		if varName == "" {
			continue
		}

		// Convert result value to uint
		var value uint
		switch v := result.Value.(type) {
		case int:
			value = uint(v)
		case int8:
			value = uint(v)
		case int16:
			value = uint(v)
		case int32:
			value = uint(v)
		case int64:
			value = uint(v)
		case uint:
			value = v
		case uint8:
			value = uint(v)
		case uint16:
			value = uint(v)
		case uint32:
			value = uint(v)
		case uint64:
			value = uint(v)
		default:
			// Skip non-integer types for now
			continue
		}

		context.AddVariable(varName, value)
	}

	return context, nil
}

// getVariableName extracts a variable name from a pointer value
func (m *Matcher) getVariableName(value interface{}) string {
	// This is a simplified version - in a real implementation,
	// we might need more sophisticated introspection
	// For now, we'll use a convention-based approach

	// If we could get the variable name through reflection, we would do it here
	// For now, return empty string - this will be enhanced later
	return ""
}
