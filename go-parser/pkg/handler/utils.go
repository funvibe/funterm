package handler

import (
	"fmt"
	"go-parser/pkg/ast"
	"go-parser/pkg/lexer"
	"math/big"
	"strconv"
	"strings"
)

// tokenToPosition конвертирует токен в позицию AST
func tokenToPosition(token lexer.Token) ast.Position {
	return ast.Position{
		Line:   token.Line,
		Column: token.Column,
		Offset: token.Position,
	}
}

// isComparisonOperator проверяет, является ли токен оператором сравнения
func isComparisonOperator(tokenType lexer.TokenType) bool {
	switch tokenType {
	case lexer.TokenLess, lexer.TokenGreater, lexer.TokenLessEqual,
		lexer.TokenGreaterEqual, lexer.TokenEqual, lexer.TokenNotEqual:
		return true
	default:
		return false
	}
}

// isBinaryOperator проверяет, является ли токен бинарным оператором
func isBinaryOperator(tokenType lexer.TokenType) bool {
	switch tokenType {
	case lexer.TokenPlus, lexer.TokenMinus, lexer.TokenMultiply, lexer.TokenPower, lexer.TokenSlash, lexer.TokenModulo,
		lexer.TokenEqual, lexer.TokenNotEqual, lexer.TokenLess, lexer.TokenLessEqual,
		lexer.TokenGreater, lexer.TokenGreaterEqual, lexer.TokenAnd, lexer.TokenOr, lexer.TokenConcat,
		lexer.TokenBitwiseOr, lexer.TokenAmpersand, lexer.TokenCaret, lexer.TokenTilde, lexer.TokenPipe,
		lexer.TokenDoubleLeftAngle, lexer.TokenDoubleRightAngle:
		return true
	default:
		return false
	}
}

// getOperatorPrecedence возвращает приоритет оператора (чем больше число, тем выше приоритет)
func getOperatorPrecedence(tokenType lexer.TokenType) int {
	switch tokenType {
	// Pipe оператор (самый низкий приоритет)
	case lexer.TokenPipe: // |>
		return 0
	// Тернарный оператор (выше операторов сравнения)
	case lexer.TokenQuestion: // ?
		return 5
	// Логические операторы
	case lexer.TokenOr: // ||
		return 1
	case lexer.TokenAnd: // &&
		return 2
	// Операторы сравнения
	case lexer.TokenEqual, lexer.TokenNotEqual: // ==, !=
		return 3
	case lexer.TokenLess, lexer.TokenLessEqual, lexer.TokenGreater, lexer.TokenGreaterEqual: // <, <=, >, >=
		return 4
	// Битовые операторы
	case lexer.TokenBitwiseOr: // |
		return 4
	case lexer.TokenDoubleLeftAngle, lexer.TokenDoubleRightAngle: // <<, >>
		return 5
	case lexer.TokenAmpersand: // &
		return 6
	case lexer.TokenCaret: // ^
		return 7
	case lexer.TokenTilde: // ~ (bitwise NOT)
		return 7
	// Арифметические операторы
	case lexer.TokenPlus, lexer.TokenMinus: // +, -
		return 8
	case lexer.TokenMultiply, lexer.TokenSlash, lexer.TokenModulo: // *, /, %
		return 9
	// Возведение в степень (выше приоритет, чем умножение)
	case lexer.TokenPower: // **
		return 10
	// Конкатенация строк
	case lexer.TokenConcat: // ++
		return 11
	default:
		return 0 // Неизвестный оператор
	}
}

// isElvisOperator проверяет, является ли токен Elvis оператором
func isElvisOperator(tokenType lexer.TokenType) bool {
	return tokenType == lexer.TokenQuestion
}

// isTernaryOperator проверяет, является ли токен частью тернарного оператора
func isTernaryOperator(tokenType lexer.TokenType) bool {
	return tokenType == lexer.TokenQuestion
}

// parseNumber преобразует строку в число, поддерживает hex (0x...), binary (0b...), научную нотацию и большие целые
func parseNumber(s string) (interface{}, error) {
	// Проверяем, это целое число или float с точкой/экспонентой
	isInteger := true
	if strings.Contains(s, ".") || (strings.ContainsAny(s, "eE") && !strings.HasPrefix(strings.ToLower(s), "0x")) {
		isInteger = false
	}

	if isInteger {
		// Пробуем парсить как большое целое число
		bigInt := new(big.Int)
		if _, success := bigInt.SetString(s, 0); success {
			return bigInt, nil
		}

		// Если не удалось как big.Int, пробуем как int64 для обратной совместимости
		if val, err := strconv.ParseInt(s, 0, 64); err == nil {
			bigInt := big.NewInt(val)
			return bigInt, nil
		}

		return nil, fmt.Errorf("invalid integer format: %s", s)
	} else {
		// Парсим как float64
		if result, err := strconv.ParseFloat(s, 64); err == nil {
			return result, nil
		}

		return nil, fmt.Errorf("invalid float format: %s", s)
	}
}

// parseFloat преобразует строку в float64, поддерживает hex (0x...), binary (0b...) и научную нотацию
// Оставлена для обратной совместимости, но рекомендуется использовать parseNumber
func parseFloat(s string) float64 {
	// Проверяем hex формат
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		// Парсим как hex integer, затем конвертируем в float64
		if val, err := strconv.ParseInt(s, 0, 64); err == nil {
			return float64(val)
		}
	}

	// Проверяем binary формат
	if strings.HasPrefix(s, "0b") || strings.HasPrefix(s, "0B") {
		// Парсим как binary integer, затем конвертируем в float64
		if val, err := strconv.ParseInt(s, 0, 64); err == nil {
			return float64(val)
		}
	}

	// Пытаемся парсить как float (поддерживает научную нотацию)
	if result, err := strconv.ParseFloat(s, 64); err == nil {
		return result
	}

	// Fallback на старый метод для совместимости
	var result float64
	fmt.Sscanf(s, "%f", &result)
	return result
}

// createNumberLiteral creates a NumberLiteral with the appropriate type based on the value
func createNumberLiteral(token lexer.Token, value interface{}) *ast.NumberLiteral {
	switch v := value.(type) {
	case *big.Int:
		return &ast.NumberLiteral{
			IntValue: v,
			IsInt:    true,
			Pos: ast.Position{
				Line:   token.Line,
				Column: token.Column,
				Offset: token.Position,
			},
		}
	case float64:
		return &ast.NumberLiteral{
			FloatValue: v,
			IsInt:      false,
			Pos: ast.Position{
				Line:   token.Line,
				Column: token.Column,
				Offset: token.Position,
			},
		}
	default:
		// Fallback - should not happen
		return &ast.NumberLiteral{
			FloatValue: 0,
			IsInt:      false,
			Pos: ast.Position{
				Line:   token.Line,
				Column: token.Column,
				Offset: token.Position,
			},
		}
	}
}
