package handler

import (
	"fmt"
	"go-parser/pkg/ast"
	"go-parser/pkg/common"
	"go-parser/pkg/config"
	"go-parser/pkg/lexer"
)

// UnaryExpressionHandler - обработчик унарных выражений
type UnaryExpressionHandler struct {
	config config.ConstructHandlerConfig
}

// NewUnaryExpressionHandler создает новый обработчик унарных выражений
func NewUnaryExpressionHandler(config config.ConstructHandlerConfig) *UnaryExpressionHandler {
	return &UnaryExpressionHandler{
		config: config,
	}
}

// CanHandle проверяет, может ли обработчик обработать токен
func (h *UnaryExpressionHandler) CanHandle(token lexer.Token) bool {
	// Обрабатываем унарные операторы
	return isUnaryOperator(token.Type)
}

// Handle обрабатывает унарное выражение
func (h *UnaryExpressionHandler) Handle(ctx *common.ParseContext) (interface{}, error) {
	tokenStream := ctx.TokenStream

	if !tokenStream.HasMore() {
		return nil, fmt.Errorf("unexpected EOF in unary expression")
	}

	operatorToken := tokenStream.Current()
	if !isUnaryOperator(operatorToken.Type) {
		return nil, fmt.Errorf("expected unary operator, got %s", operatorToken.Type)
	}

	// Потребляем оператор
	tokenStream.Consume()

	// Читаем операнд
	if !tokenStream.HasMore() {
		return nil, fmt.Errorf("unexpected EOF after unary operator")
	}

	operand, err := h.parseOperand(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to parse operand: %v", err)
	}

	// Создаем унарное выражение
	return ast.NewUnaryExpression(operatorToken.Value, operand, ast.Position{
		Line:   operatorToken.Line,
		Column: operatorToken.Column,
		Offset: operatorToken.Position,
	}), nil
}

// parseOperand парсит операнд унарного выражения
func (h *UnaryExpressionHandler) parseOperand(ctx *common.ParseContext) (ast.Expression, error) {
	tokenStream := ctx.TokenStream

	if !tokenStream.HasMore() {
		return nil, fmt.Errorf("unexpected EOF in operand")
	}

	token := tokenStream.Current()

	switch token.Type {
	case lexer.TokenIdentifier:
		// Это может быть простой идентификатор или вызов функции
		if tokenStream.Peek().Type == lexer.TokenLeftParen {
			// Это вызов функции
			return h.parseLanguageCall(ctx)
		} else if tokenStream.Peek().Type == lexer.TokenDot {
			// Это может быть qualified variable
			return h.parseQualifiedVariable(ctx)
		} else {
			// Простой идентификатор
			tokenStream.Consume()
			return ast.NewIdentifier(token, token.Value), nil
		}

	case lexer.TokenNumber:
		// Числовой литерал
		tokenStream.Consume()
		return &ast.NumberLiteral{
			Value: parseFloat(token.Value),
			Pos: ast.Position{
				Line:   token.Line,
				Column: token.Column,
				Offset: token.Position,
			},
		}, nil

	case lexer.TokenString:
		// Строковой литерал
		tokenStream.Consume()
		return &ast.StringLiteral{
			Value: token.Value,
			Pos: ast.Position{
				Line:   token.Line,
				Column: token.Column,
				Offset: token.Position,
			},
		}, nil

	case lexer.TokenTrue, lexer.TokenFalse:
		// Булев литерал
		tokenStream.Consume()
		return &ast.BooleanLiteral{
			Value: token.Type == lexer.TokenTrue,
			Pos: ast.Position{
				Line:   token.Line,
				Column: token.Column,
				Offset: token.Position,
			},
		}, nil

	case lexer.TokenLeftParen:
		// Выражение в скобках
		tokenStream.Consume() // потребляем '('

		// Парсим выражение внутри скобок
		// Для унарных выражений мы можем использовать бинарный обработчик
		binaryHandler := NewBinaryExpressionHandler(config.ConstructHandlerConfig{})
		expr, err := binaryHandler.ParseFullExpression(ctx, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to parse expression in parentheses: %v", err)
		}

		// Проверяем и потребляем закрывающую скобку
		if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenRightParen {
			return nil, fmt.Errorf("expected ')' after expression")
		}
		tokenStream.Consume() // потребляем ')'

		return expr, nil

	default:
		return nil, fmt.Errorf("unsupported operand type: %s", token.Type)
	}
}

// parseLanguageCall парсит вызов функции другого языка
func (h *UnaryExpressionHandler) parseLanguageCall(ctx *common.ParseContext) (ast.Expression, error) {
	// Используем существующий LanguageCallHandler
	languageCallHandler := NewLanguageCallHandler(config.ConstructHandlerConfig{})
	result, err := languageCallHandler.Handle(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to parse language call: %v", err)
	}

	if call, ok := result.(*ast.LanguageCall); ok {
		return call, nil
	}

	return nil, fmt.Errorf("expected LanguageCall, got %T", result)
}

// parseQualifiedVariable парсит квалифицированную переменную
func (h *UnaryExpressionHandler) parseQualifiedVariable(ctx *common.ParseContext) (ast.Expression, error) {
	tokenStream := ctx.TokenStream

	// Потребляем первый идентификатор (язык)
	firstToken := tokenStream.Consume()
	if firstToken.Type != lexer.TokenIdentifier {
		return nil, fmt.Errorf("expected identifier for qualified variable, got %s", firstToken.Type)
	}

	language := firstToken.Value
	var path []string
	var lastName string

	// Обрабатываем остальные части qualified variable
	for tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenDot {
		// Потребляем точку
		tokenStream.Consume()

		// Проверяем, что после точки идет идентификатор
		if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenIdentifier {
			return nil, fmt.Errorf("expected identifier after dot in qualified variable")
		}

		// Потребляем идентификатор
		idToken := tokenStream.Consume()

		// Если следующий токен - точка, то это часть пути
		if tokenStream.HasMore() && tokenStream.Peek().Type == lexer.TokenDot {
			path = append(path, idToken.Value)
		} else {
			// Это последнее имя
			lastName = idToken.Value
		}
	}

	// Создаем квалифицированный идентификатор
	var qualifiedIdentifier *ast.Identifier
	if len(path) > 0 {
		qualifiedIdentifier = ast.NewQualifiedIdentifierWithPath(firstToken, tokenStream.PeekN(-1), language, path, lastName)
	} else {
		qualifiedIdentifier = ast.NewQualifiedIdentifier(firstToken, tokenStream.PeekN(-1), language, lastName)
	}

	// Создаем VariableRead
	return ast.NewVariableRead(qualifiedIdentifier), nil
}

// Config возвращает конфигурацию обработчика
func (h *UnaryExpressionHandler) Config() common.HandlerConfig {
	return common.HandlerConfig{
		IsEnabled: h.config.IsEnabled,
		Priority:  h.config.Priority,
		Name:      h.config.Name,
	}
}

// Name возвращает имя обработчика
func (h *UnaryExpressionHandler) Name() string {
	return h.config.Name
}

// isUnaryOperator проверяет, является ли токен унарным оператором
func isUnaryOperator(tokenType lexer.TokenType) bool {
	switch tokenType {
	case lexer.TokenPlus, lexer.TokenMinus, lexer.TokenNot, lexer.TokenTilde:
		return true
	default:
		return false
	}
}
