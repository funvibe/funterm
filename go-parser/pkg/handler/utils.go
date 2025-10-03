package handler

import (
	"fmt"
	"go-parser/pkg/ast"
	"go-parser/pkg/lexer"
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
	case lexer.TokenPlus, lexer.TokenMinus, lexer.TokenMultiply, lexer.TokenSlash, lexer.TokenModulo,
		lexer.TokenEqual, lexer.TokenNotEqual, lexer.TokenLess, lexer.TokenLessEqual,
		lexer.TokenGreater, lexer.TokenGreaterEqual, lexer.TokenAnd, lexer.TokenOr, lexer.TokenConcat:
		return true
	default:
		return false
	}
}

// getOperatorPrecedence возвращает приоритет оператора (чем больше число, тем выше приоритет)
func getOperatorPrecedence(tokenType lexer.TokenType) int {
	switch tokenType {
	// Логические операторы (самый низкий приоритет)
	case lexer.TokenOr: // ||
		return 1
	case lexer.TokenAnd: // &&
		return 2
	// Операторы сравнения
	case lexer.TokenEqual, lexer.TokenNotEqual: // ==, !=
		return 3
	case lexer.TokenLess, lexer.TokenLessEqual, lexer.TokenGreater, lexer.TokenGreaterEqual: // <, <=, >, >=
		return 4
	// Арифметические операторы
	case lexer.TokenPlus, lexer.TokenMinus: // +, -
		return 5
	case lexer.TokenMultiply, lexer.TokenSlash, lexer.TokenModulo: // *, /, %
		return 6
	// Конкатенация строк
	case lexer.TokenConcat: // ++
		return 7
	default:
		return 0 // Неизвестный оператор
	}
}

// isElvisOperator проверяет, является ли токен Elvis оператором
func isElvisOperator(tokenType lexer.TokenType) bool {
	return tokenType == lexer.TokenQuestion
}

// parseFloat преобразует строку в float64, поддерживает hex формат (0x...)
func parseFloat(s string) float64 {
	// Проверяем hex формат
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		// Парсим как hex integer, затем конвертируем в float64
		if val, err := strconv.ParseInt(s, 0, 64); err == nil {
			return float64(val)
		}
	}

	// Обычный float парсинг
	var result float64
	fmt.Sscanf(s, "%f", &result)
	return result
}
