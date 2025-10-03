package handler

import (
	"fmt"

	"go-parser/pkg/ast"
	"go-parser/pkg/common"
	"go-parser/pkg/config"
	"go-parser/pkg/lexer"
	"go-parser/pkg/stream"
)

// ParenthesizedExpressionHandler - обработчик для выражений в скобках
type ParenthesizedExpressionHandler struct {
	config config.ConstructHandlerConfig
}

// NewParenthesizedExpressionHandler создает новый обработчик для выражений в скобках
func NewParenthesizedExpressionHandler(config config.ConstructHandlerConfig) *ParenthesizedExpressionHandler {
	return &ParenthesizedExpressionHandler{
		config: config,
	}
}

// CanHandle проверяет, может ли обработчик обработать токен
func (h *ParenthesizedExpressionHandler) CanHandle(token lexer.Token) bool {
	// ParenthesizedExpressionHandler обрабатывает открывающую скобку
	return token.Type == lexer.TokenLeftParen
}

// Handle обрабатывает выражение в скобках
func (h *ParenthesizedExpressionHandler) Handle(ctx *common.ParseContext) (interface{}, error) {
	tokenStream := ctx.TokenStream

	// Проверяем, что текущий токен - открывающая скобка
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenLeftParen {
		return nil, fmt.Errorf("expected '('")
	}

	// Потребляем открывающую скобку
	leftParenToken := tokenStream.Consume()

	// Разбираем выражение внутри скобок
	expression, err := h.parseExpressionInsideParentheses(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to parse expression inside parentheses: %v", err)
	}

	// Проверяем закрывающую скобку
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenRightParen {
		return nil, fmt.Errorf("expected ')' after expression")
	}
	rightParenToken := tokenStream.Consume()

	// Создаем GroupedExpression для представления выражения в скобках
	pos := ast.Position{
		Line:   leftParenToken.Line,
		Column: leftParenToken.Column,
		Offset: leftParenToken.Position,
	}

	return h.createGroupedExpression(expression, leftParenToken, rightParenToken, pos)
}

// parseExpressionInsideParentheses разбирает выражение внутри скобок
func (h *ParenthesizedExpressionHandler) parseExpressionInsideParentheses(ctx *common.ParseContext) (ast.Expression, error) {
	tokenStream := ctx.TokenStream

	if !tokenStream.HasMore() {
		return nil, fmt.Errorf("unexpected end of input inside parentheses")
	}

	// Проверяем тип выражения внутри скобок
	token := tokenStream.Current()

	switch token.Type {
	case lexer.TokenIdentifier:
		// Это может быть language call, pipe expression или простой идентификатор
		return h.parseComplexExpression(ctx)

	case lexer.TokenString:
		// Строковый литерал
		strToken := tokenStream.Consume()
		return &ast.StringLiteral{
			Value: strToken.Value,
			Pos: ast.Position{
				Line:   strToken.Line,
				Column: strToken.Column,
				Offset: strToken.Position,
			},
		}, nil

	case lexer.TokenNumber:
		// Числовой литерал
		numToken := tokenStream.Consume()
		numValue := 0.0
		fmt.Sscanf(numToken.Value, "%f", &numValue)
		return &ast.NumberLiteral{
			Value: numValue,
			Pos: ast.Position{
				Line:   numToken.Line,
				Column: numToken.Column,
				Offset: numToken.Position,
			},
		}, nil

	case lexer.TokenLeftParen:
		// Вложенные скобки - рекурсивно вызываем этот же обработчик
		result, err := h.Handle(ctx)
		if err != nil {
			return nil, err
		}
		expr, ok := result.(ast.Expression)
		if !ok {
			return nil, fmt.Errorf("expected Expression, got %T", result)
		}
		return expr, nil

	default:
		return nil, fmt.Errorf("unsupported token type inside parentheses: %s", token.Type)
	}
}

// parseComplexExpression разбирает сложное выражение (language call или pipe)
func (h *ParenthesizedExpressionHandler) parseComplexExpression(ctx *common.ParseContext) (ast.Expression, error) {
	tokenStream := ctx.TokenStream

	// Проверяем, есть ли в потоке оператор | (для pipe expressions)
	if h.hasPipeOperatorInExpression(tokenStream) {
		return h.parsePipeExpressionInParentheses(ctx)
	}

	// Если нет pipe, пробуем разобрать как language call
	return h.parseLanguageCallInParentheses(ctx)
}

// hasPipeOperatorInExpression проверяет, есть ли в выражении оператор |
func (h *ParenthesizedExpressionHandler) hasPipeOperatorInExpression(stream stream.TokenStream) bool {
	// Сохраняем текущую позицию
	currentPos := stream.Position()
	defer stream.SetPosition(currentPos)

	// Ищем токен | с учетом вложенных скобок
	parenDepth := 0
	for stream.HasMore() {
		token := stream.Current()
		if token.Type == lexer.TokenPipe {
			return true
		}
		if token.Type == lexer.TokenLeftParen {
			parenDepth++
		} else if token.Type == lexer.TokenRightParen {
			if parenDepth == 0 {
				return false // Дошли до закрывающей скобки того же уровня, | не найден
			}
			parenDepth--
		}
		stream.Consume()
	}

	return false
}

// parsePipeExpressionInParentheses разбирает pipe expression внутри скобок
func (h *ParenthesizedExpressionHandler) parsePipeExpressionInParentheses(ctx *common.ParseContext) (ast.Expression, error) {
	tokenStream := ctx.TokenStream

	// Разбираем первое выражение
	firstExpr, err := h.parseSingleExpression(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to parse first expression in pipe: %v", err)
	}

	// Разбираем последовательность | expr, но только до закрывающей скобки
	stages := []ast.Expression{firstExpr}
	operators := []lexer.Token{}

	// Продолжаем разбирать, пока есть pipe операторы и не дошли до закрывающей скобки
	for tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenPipe {
		// Сохраняем оператор |
		pipeToken := tokenStream.Consume()
		operators = append(operators, pipeToken)

		// Проверяем, что после pipe есть выражение и это не закрывающая скобка
		if !tokenStream.HasMore() || tokenStream.Current().Type == lexer.TokenRightParen {
			return nil, fmt.Errorf("expected expression after pipe operator")
		}

		// Проверяем, что следующий токен может начинать выражение
		if !h.isValidExpressionStart(tokenStream.Current()) {
			return nil, fmt.Errorf("invalid expression start after pipe: %s", tokenStream.Current().Type)
		}

		// Разбираем следующее выражение
		nextExpr, err := h.parseSingleExpression(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to parse expression after pipe: %v", err)
		}
		stages = append(stages, nextExpr)

		// Если следующий токен - закрывающая скобка, прекращаем разбор
		if tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenRightParen {
			break
		}
	}

	if len(stages) < 2 {
		return nil, fmt.Errorf("pipe expression must have at least 2 stages")
	}

	// Создаем PipeExpression
	pos := ast.Position{
		Line:   operators[0].Line,
		Column: operators[0].Column,
		Offset: operators[0].Position,
	}

	return ast.NewPipeExpression(stages, operators, pos), nil
}

// parseLanguageCallInParentheses разбирает language call внутри скобок
func (h *ParenthesizedExpressionHandler) parseLanguageCallInParentheses(ctx *common.ParseContext) (ast.Expression, error) {
	// Используем существующий LanguageCallHandler
	languageCallHandler := NewLanguageCallHandler(config.ConstructHandlerConfig{
		IsEnabled: true,
		Priority:  100,
		Name:      "language_call_in_parentheses",
	})

	// Создаем копию контекста с включенным режимом частичного парсинга
	partialCtx := &common.ParseContext{
		TokenStream:        ctx.TokenStream,
		Parser:             ctx.Parser,
		Depth:              ctx.Depth,
		MaxDepth:           ctx.MaxDepth,
		Guard:              ctx.Guard,
		PartialParsingMode: true, // Включаем частичный парсинг
	}

	result, err := languageCallHandler.Handle(partialCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to parse language call: %v", err)
	}

	languageCall, ok := result.(*ast.LanguageCall)
	if !ok {
		return nil, fmt.Errorf("expected LanguageCall, got %T", result)
	}

	return languageCall, nil
}

// parseSingleExpression разбирает одно выражение (упрощенная версия)
func (h *ParenthesizedExpressionHandler) parseSingleExpression(ctx *common.ParseContext) (ast.Expression, error) {
	tokenStream := ctx.TokenStream

	if !tokenStream.HasMore() {
		return nil, fmt.Errorf("unexpected end of input")
	}

	token := tokenStream.Current()

	// Проверяем, что токен может начинать выражение
	if !h.isValidExpressionStart(token) {
		return nil, fmt.Errorf("invalid expression start token: %s", token.Type)
	}

	switch token.Type {
	case lexer.TokenIdentifier:
		// Пробуем разобрать как language call
		return h.parseLanguageCallInParentheses(ctx)

	case lexer.TokenString:
		// Строковый литерал
		strToken := tokenStream.Consume()
		return &ast.StringLiteral{
			Value: strToken.Value,
			Pos: ast.Position{
				Line:   strToken.Line,
				Column: strToken.Column,
				Offset: strToken.Position,
			},
		}, nil

	case lexer.TokenNumber:
		// Числовой литерал
		numToken := tokenStream.Consume()
		numValue := 0.0
		fmt.Sscanf(numToken.Value, "%f", &numValue)
		return &ast.NumberLiteral{
			Value: numValue,
			Pos: ast.Position{
				Line:   numToken.Line,
				Column: numToken.Column,
				Offset: numToken.Position,
			},
		}, nil

	case lexer.TokenLeftParen:
		// Вложенные скобки - рекурсивно вызываем этот же обработчик
		result, err := h.Handle(ctx)
		if err != nil {
			return nil, err
		}
		expr, ok := result.(ast.Expression)
		if !ok {
			return nil, fmt.Errorf("expected Expression, got %T", result)
		}
		return expr, nil

	default:
		return nil, fmt.Errorf("unsupported token type: %s", token.Type)
	}
}

// isValidExpressionStart проверяет, может ли токен начинать выражение
func (h *ParenthesizedExpressionHandler) isValidExpressionStart(token lexer.Token) bool {
	switch token.Type {
	case lexer.TokenIdentifier, lexer.TokenString, lexer.TokenNumber, lexer.TokenLeftParen:
		return true
	default:
		return false
	}
}

// createGroupedExpression создает сгруппированное выражение
func (h *ParenthesizedExpressionHandler) createGroupedExpression(expression ast.Expression, leftParen, rightParen lexer.Token, pos ast.Position) (ast.Expression, error) {
	// Создаем GroupedExpression для представления выражения в скобках
	// Это может быть просто выражение, но с пометкой о том, что оно было в скобках

	// Возвращаем само выражение, так как скобки в AST обычно не нужны,
	// но могут быть важны для приоритета операций
	return expression, nil
}

// Config возвращает конфигурацию обработчика
func (h *ParenthesizedExpressionHandler) Config() common.HandlerConfig {
	return common.HandlerConfig{
		IsEnabled: h.config.IsEnabled,
		Priority:  h.config.Priority, // Должен быть 120 (самый высокий)
		Name:      h.config.Name,
	}
}

// Name возвращает имя обработчика
func (h *ParenthesizedExpressionHandler) Name() string {
	return h.config.Name
}
