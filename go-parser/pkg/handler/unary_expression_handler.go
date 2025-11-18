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
		return nil, newErrorWithPos(tokenStream, "unexpected EOF in unary expression")
	}

	operatorToken := tokenStream.Current()
	if !isUnaryOperator(operatorToken.Type) {
		return nil, newErrorWithTokenPos(operatorToken, "expected unary operator, got %s", operatorToken.Type)
	}

	// Потребляем оператор
	tokenStream.Consume()

	// Читаем операнд
	if !tokenStream.HasMore() {
		return nil, newErrorWithPos(tokenStream, "unexpected EOF after unary operator")
	}

	// Специальная обработка для оператора @ (размер bitstring)
	if operatorToken.Type == lexer.TokenAt {
		operand, err := h.parseSizeOperand(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to parse size operand: %v", err)
		}
		return ast.NewUnaryExpression(operatorToken.Value, operand, ast.Position{
			Line:   operatorToken.Line,
			Column: operatorToken.Column,
			Offset: operatorToken.Position,
		}), nil
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

// parseSizeOperand парсит операнд для оператора @ (размер bitstring)
// Поддерживает как @(expression) так и @variable
func (h *UnaryExpressionHandler) parseSizeOperand(ctx *common.ParseContext) (ast.Expression, error) {
	tokenStream := ctx.TokenStream

	// Проверяем, есть ли открывающая скобка
	if tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenLeftParen {
		// Используем скобки: @(expression)
		tokenStream.Consume() // потребляем '('

		// Парсим выражение внутри скобок
		if !tokenStream.HasMore() {
			return nil, newErrorWithPos(tokenStream, "unexpected EOF in @ expression")
		}

		var expr ast.Expression
		var err error

		// Если следующий токен - <<, это bitstring, иначе обычное выражение
		if tokenStream.Current().Type == lexer.TokenDoubleLeftAngle {
			// Используем bitstring handler для парсинга bitstring
			bitstringHandler := NewBitstringHandler(config.ConstructHandlerConfig{})
			result, err := bitstringHandler.Handle(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to parse bitstring in @ expression: %v", err)
			}
			if bs, ok := result.(*ast.BitstringExpression); ok {
				expr = bs
			} else {
				return nil, fmt.Errorf("expected BitstringExpression, got %T", result)
			}
		} else {
			// Для других выражений используем binary expression handler
			binaryHandler := NewBinaryExpressionHandler(config.ConstructHandlerConfig{})
			expr, err = binaryHandler.ParseFullExpression(ctx, nil)
			if err != nil {
				return nil, fmt.Errorf("failed to parse expression in @ parentheses: %v", err)
			}
		}

		// Ожидаем закрывающую скобку
		if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenRightParen {
			return nil, newErrorWithPos(tokenStream, "expected ')' after @ expression")
		}
		tokenStream.Consume() // потребляем ')'

		return expr, nil
	} else {
		// Без скобок: @variable или @field
		// Парсим простой операнд (идентификатор, qualified variable)
		operand, err := h.parseSimpleSizeOperand(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to parse size operand: %v", err)
		}
		return operand, nil
	}
}

// parseSimpleSizeOperand парсит простой операнд для @ без скобок
func (h *UnaryExpressionHandler) parseSimpleSizeOperand(ctx *common.ParseContext) (ast.Expression, error) {
	tokenStream := ctx.TokenStream

	if !tokenStream.HasMore() {
		return nil, newErrorWithPos(tokenStream, "unexpected EOF after @ operator")
	}

	token := tokenStream.Current()

	switch token.Type {
	case lexer.TokenIdentifier:
		// Это может быть простой идентификатор или qualified variable
		if tokenStream.Peek().Type == lexer.TokenDot {
			// Это qualified variable
			return h.parseQualifiedVariable(ctx)
		} else {
			// Простой идентификатор
			tokenStream.Consume()
			return ast.NewVariableRead(ast.NewIdentifier(token, token.Value)), nil
		}

	case lexer.TokenLua, lexer.TokenPython, lexer.TokenJS, lexer.TokenGo, lexer.TokenNode, lexer.TokenPy:
		// Language token - qualified variable
		return h.parseQualifiedVariable(ctx)

	default:
		return nil, newErrorWithTokenPos(token, "unsupported operand for @ operator: %s", token.Type)
	}
}

// parseOperand парсит операнд унарного выражения
func (h *UnaryExpressionHandler) parseOperand(ctx *common.ParseContext) (ast.Expression, error) {
	tokenStream := ctx.TokenStream

	if !tokenStream.HasMore() {
		return nil, newErrorWithPos(tokenStream, "unexpected EOF in operand")
	}

	token := tokenStream.Current()

	// Проверяем language tokens (python, lua, js, etc.)
	if token.IsLanguageToken() {
		// Это language call
		return h.parseLanguageCall(ctx)
	}

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
		value, err := parseNumber(token.Value)
		if err != nil {
			return nil, newErrorWithTokenPos(token, "invalid number format: %s", token.Value)
		}
		return createNumberLiteral(token, value), nil

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
			return nil, newErrorWithPos(tokenStream, "expected ')' after expression")
		}
		tokenStream.Consume() // потребляем ')'

		return expr, nil

	case lexer.TokenMinus, lexer.TokenNot, lexer.TokenTilde, lexer.TokenAt, lexer.TokenPlus:
		// Вложенный унарный оператор - рекурсивно обрабатываем
		unaryHandler := NewUnaryExpressionHandler(config.ConstructHandlerConfig{})
		result, err := unaryHandler.Handle(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to parse nested unary operator: %v", err)
		}
		if expr, ok := result.(ast.Expression); ok {
			return expr, nil
		}
		return nil, fmt.Errorf("expected Expression, got %T", result)

	default:
		return nil, newErrorWithTokenPos(token, "unsupported operand type: %s", token.Type)
	}
}

// parseLanguageCall парсит вызов функции другого языка
func (h *UnaryExpressionHandler) parseLanguageCall(ctx *common.ParseContext) (ast.Expression, error) {
	// Используем существующий LanguageCallHandler
	languageCallHandler := NewLanguageCallHandler(config.ConstructHandlerConfig{ConstructType: common.ConstructLanguageCall})
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
		return nil, newErrorWithTokenPos(firstToken, "expected identifier for qualified variable, got %s", firstToken.Type)
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
			return nil, newErrorWithPos(tokenStream, "expected identifier after dot in qualified variable")
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

