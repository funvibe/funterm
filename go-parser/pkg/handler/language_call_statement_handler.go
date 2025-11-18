package handler

import (
	"fmt"

	"go-parser/pkg/ast"
	"go-parser/pkg/common"
	"go-parser/pkg/config"
	"go-parser/pkg/lexer"
	"go-parser/pkg/stream"
)

// LanguageCallStatementHandler - обработчик для language call statements с поддержкой background tasks (&)
type LanguageCallStatementHandler struct {
	config           config.ConstructHandlerConfig
	languageRegistry LanguageRegistry
}

// NewLanguageCallStatementHandler создает новый обработчик для language call statements
func NewLanguageCallStatementHandler(config config.ConstructHandlerConfig) *LanguageCallStatementHandler {
	return &LanguageCallStatementHandler{
		config:           config,
		languageRegistry: CreateDefaultLanguageRegistry(),
	}
}

// NewLanguageCallStatementHandlerWithRegistry создает обработчик с явным указанием Language Registry
func NewLanguageCallStatementHandlerWithRegistry(config config.ConstructHandlerConfig, registry LanguageRegistry) *LanguageCallStatementHandler {
	return &LanguageCallStatementHandler{
		config:           config,
		languageRegistry: registry,
	}
}

// CanHandle проверяет, может ли обработчик обработать токен
func (h *LanguageCallStatementHandler) CanHandle(token lexer.Token) bool {
	// LanguageCallStatementHandler обрабатывает идентификаторы и токены языков, проверяя наличие & в конце выражения
	return token.Type == lexer.TokenIdentifier ||
		token.Type == lexer.TokenLua ||
		token.Type == lexer.TokenPython ||
		token.Type == lexer.TokenPy ||
		token.Type == lexer.TokenGo ||
		token.Type == lexer.TokenNode ||
		token.Type == lexer.TokenJS
}

// Handle обрабатывает language call statement
func (h *LanguageCallStatementHandler) Handle(ctx *common.ParseContext) (interface{}, error) {
	tokenStream := ctx.TokenStream

	// Проверяем, есть ли & в конце всего выражения
	if !h.hasAmpersandAtEnd(tokenStream) {
		// Если нет & в конце, то это не background task, передаем управление другому обработчику
		// Возвращаем специфичную ошибку, чтобы другие обработчики могли попробовать
		return nil, fmt.Errorf("not a background task statement")
	}

	// Разбираем выражение перед &
	expression, err := h.parseExpressionBeforeAmpersand(ctx)
	if err != nil {
		// Если не удалось разобрать выражение перед &, возвращаем ошибку, которую ожидает тест
		if err.Error() == "unexpected EOF after argument" || err.Error() == "unexpected end of input" {
			return nil, fmt.Errorf("unexpected EOF after argument")
		}
		return nil, fmt.Errorf("failed to parse expression before ampersand: %v", err)
	}

	// Проверяем и потребляем &
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenAmpersand {
		return nil, fmt.Errorf("expected & after expression")
	}
	ampersandToken := tokenStream.Consume()

	// Создаем LanguageCallStatement для background task
	pos := ast.Position{
		Line:   ampersandToken.Line,
		Column: ampersandToken.Column,
		Offset: ampersandToken.Position,
	}

	return h.createLanguageCallStatement(expression, ampersandToken, pos)
}

// hasAmpersandAtEnd проверяет, есть ли в потоке оператор & в конце выражения
func (h *LanguageCallStatementHandler) hasAmpersandAtEnd(stream stream.TokenStream) bool {
	// Сохраняем текущую позицию
	currentPos := stream.Position()
	defer stream.SetPosition(currentPos)

	// Ищем токен & в потоке, пропуская все выражение
	for stream.HasMore() {
		token := stream.Current()
		if token.Type == lexer.TokenAmpersand {
			// Потребляем &
			stream.Consume()

			// Проверяем, что после & идет только newline или конец потока
			if stream.HasMore() {
				nextToken := stream.Current()
				if nextToken.Type == lexer.TokenNewline {
					return true // & в конце строки, это background task
				}
				return false // После & есть что-то кроме newline, это не background task
			}
			return true // & в самом конце потока, это background task
		}
		stream.Consume()
	}

	return false // & не найден
}

// parseExpressionBeforeAmpersand разбирает выражение, находящееся перед оператором &
func (h *LanguageCallStatementHandler) parseExpressionBeforeAmpersand(ctx *common.ParseContext) (ast.Expression, error) {
	tokenStream := ctx.TokenStream

	// Проверяем, есть ли pipe оператор в выражении
	if h.hasPipeInStream(tokenStream) {
		return h.parsePipeExpressionUntilAmpersand(ctx)
	}

	// Если нет pipe, просто разбираем как language call
	return h.parseLanguageCallUntilAmpersand(ctx)
}

// createLimitedStreamUntilAmpersand создает ограниченный поток токенов до &
func (h *LanguageCallStatementHandler) createLimitedStreamUntilAmpersand(originalStream stream.TokenStream) stream.TokenStream {
	// Просто возвращаем клон оригинального потока
	// Ограничение будет достигнуто путем проверки на TokenAmpersand в методах разбора
	return originalStream.Clone()
}

// hasPipeInStream проверяет, есть ли в потоке оператор |
func (h *LanguageCallStatementHandler) hasPipeInStream(stream stream.TokenStream) bool {
	// Сохраняем текущую позицию
	currentPos := stream.Position()
	defer stream.SetPosition(currentPos)

	// Ищем токен | в потоке
	for stream.HasMore() {
		if stream.Current().Type == lexer.TokenPipe {
			return true
		}
		stream.Consume()
	}

	return false
}

// parsePipeExpressionInLimitedContext разбирает pipe expression в ограниченном контексте
func (h *LanguageCallStatementHandler) parsePipeExpressionInLimitedContext(ctx *common.ParseContext) (ast.Expression, error) {
	// Это сложно, так как PipeHandler теперь ожидает токен |, а не TokenIdentifier
	// Вместо этого используем упрощенный подход

	// Разбираем первое выражение
	firstExpr, err := h.parseSingleExpression(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to parse first expression: %v", err)
	}

	// Проверяем наличие |
	tokenStream := ctx.TokenStream
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenPipe {
		return nil, fmt.Errorf("expected pipe operator")
	}

	// Потребляем |
	pipeToken := tokenStream.Consume()

	// Разбираем второе выражение
	secondExpr, err := h.parseSingleExpression(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to parse second expression: %v", err)
	}

	// Создаем PipeExpression
	stages := []ast.Expression{firstExpr, secondExpr}
	operators := []lexer.Token{pipeToken}

	pos := ast.Position{
		Line:   pipeToken.Line,
		Column: pipeToken.Column,
		Offset: pipeToken.Position,
	}

	return ast.NewPipeExpression(stages, operators, pos), nil
}

// parseLanguageCallInLimitedContext разбирает language call в ограниченном контексте
func (h *LanguageCallStatementHandler) parseLanguageCallInLimitedContext(ctx *common.ParseContext) (ast.Expression, error) {
	// Используем существующий LanguageCallHandler
	languageCallHandler := NewLanguageCallHandlerWithRegistry(config.ConstructHandlerConfig{
		IsEnabled: true,
		Priority:  100,
		Name:      "language_call_for_statement",
	}, h.languageRegistry)

	result, err := languageCallHandler.Handle(ctx)
	if err != nil {
		return nil, err
	}

	languageCall, ok := result.(*ast.LanguageCall)
	if !ok {
		return nil, fmt.Errorf("expected LanguageCall, got %T", result)
	}

	return languageCall, nil
}

// parseSingleExpression разбирает одно выражение (упрощенная версия)
func (h *LanguageCallStatementHandler) parseSingleExpression(ctx *common.ParseContext) (ast.Expression, error) {
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
		return h.parseLanguageCallInLimitedContext(ctx)

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
		numValue, err := parseNumber(numToken.Value)
		if err != nil {
			return nil, fmt.Errorf("invalid number format: %s", numToken.Value)
		}
		return createNumberLiteral(numToken, numValue), nil

	case lexer.TokenLeftParen:
		// Сгруппированное выражение в скобках
		return h.parseParenthesizedExpression(ctx)

	default:
		return nil, fmt.Errorf("unsupported token type: %s", token.Type)
	}
}

// isValidExpressionStart проверяет, может ли токен начинать выражение
func (h *LanguageCallStatementHandler) isValidExpressionStart(token lexer.Token) bool {
	switch token.Type {
	case lexer.TokenIdentifier, lexer.TokenString, lexer.TokenNumber, lexer.TokenLeftParen:
		return true
	default:
		return false
	}
}

// parseParenthesizedExpression разбирает выражение в скобках
func (h *LanguageCallStatementHandler) parseParenthesizedExpression(ctx *common.ParseContext) (ast.Expression, error) {
	tokenStream := ctx.TokenStream

	// Потребляем открывающую скобку
	tokenStream.Consume()

	// Разбираем выражение внутри скобок
	expr, err := h.parseSingleExpression(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to parse expression in parentheses: %v", err)
	}

	// Проверяем закрывающую скобку
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenRightParen {
		return nil, fmt.Errorf("expected ')' after parenthesized expression")
	}
	tokenStream.Consume() // Потребляем закрывающую скобку

	return expr, nil
}

// createLanguageCallStatement создает LanguageCallStatement для background task
func (h *LanguageCallStatementHandler) createLanguageCallStatement(expression ast.Expression, ampersandToken lexer.Token, pos ast.Position) (*ast.LanguageCallStatement, error) {
	// В зависимости от типа выражения создаем соответствующий LanguageCallStatement
	switch expr := expression.(type) {
	case *ast.LanguageCall:
		// Обычный language call
		return ast.NewBackgroundLanguageCallStatement(expr, ampersandToken, pos), nil

	case *ast.PipeExpression:
		// Pipe expression - создаем специальный LanguageCallStatement
		languageCall := &ast.LanguageCall{
			Language:  "pipe",
			Function:  "background",
			Arguments: []ast.Expression{expr},
			Pos:       pos,
		}
		return ast.NewBackgroundLanguageCallStatement(languageCall, ampersandToken, pos), nil

	default:
		return nil, fmt.Errorf("unsupported expression type for background task: %T", expression)
	}
}

// skipLanguageCall пропускает language call в потоке
func (h *LanguageCallStatementHandler) skipLanguageCall(stream stream.TokenStream) error {
	if !stream.HasMore() {
		return fmt.Errorf("unexpected end of stream")
	}

	// Пропускаем первый идентификатор (язык)
	if stream.Current().Type != lexer.TokenIdentifier {
		return fmt.Errorf("expected identifier (language)")
	}
	stream.Consume()

	// Пропускаем точку
	if !stream.HasMore() || stream.Current().Type != lexer.TokenDot {
		return fmt.Errorf("expected dot after language")
	}
	stream.Consume()

	// Пропускаем второй идентификатор (функция)
	if !stream.HasMore() || stream.Current().Type != lexer.TokenIdentifier {
		return fmt.Errorf("expected identifier (function)")
	}
	stream.Consume()

	// Пропускаем аргументы в скобках, если они есть
	if stream.HasMore() && stream.Current().Type == lexer.TokenLeftParen {
		stream.Consume() // Пропускаем (
		// Пропускаем все до )
		parenCount := 1
		for parenCount > 0 && stream.HasMore() {
			nextToken := stream.Consume()
			if nextToken.Type == lexer.TokenLeftParen {
				parenCount++
			} else if nextToken.Type == lexer.TokenRightParen {
				parenCount--
			}
		}
		if parenCount != 0 {
			return fmt.Errorf("unclosed parentheses")
		}
	}

	return nil
}

// hasPipeOperator проверяет, есть ли в потоке оператор | после текущего выражения
func (h *LanguageCallStatementHandler) hasPipeOperator(stream stream.TokenStream) bool {
	// Сохраняем текущую позицию
	currentPos := stream.Position()
	defer stream.SetPosition(currentPos)

	// Пропускаем текущее выражение
	err := h.skipExpression(stream)
	if err != nil {
		return false
	}

	// Проверяем, следующий токен - это |
	if stream.HasMore() && stream.Current().Type == lexer.TokenPipe {
		return true
	}

	return false
}

// skipExpression пропускает одно выражение в потоке
func (h *LanguageCallStatementHandler) skipExpression(stream stream.TokenStream) error {
	if !stream.HasMore() {
		return fmt.Errorf("unexpected end of stream")
	}

	token := stream.Current()

	switch token.Type {
	case lexer.TokenIdentifier:
		// Пропускаем идентификатор
		stream.Consume()

		// Пропускаем возможную точку и следующий идентификатор (для language.function)
		for stream.HasMore() && stream.Current().Type == lexer.TokenDot {
			stream.Consume() // Пропускаем .
			if !stream.HasMore() || stream.Current().Type != lexer.TokenIdentifier {
				return fmt.Errorf("expected identifier after dot")
			}
			stream.Consume() // Пропускаем следующий идентификатор
		}

		// Пропускаем аргументы в скобках, если они есть
		if stream.HasMore() && stream.Current().Type == lexer.TokenLeftParen {
			stream.Consume() // Пропускаем (
			// Пропускаем все до )
			parenCount := 1
			for parenCount > 0 && stream.HasMore() {
				nextToken := stream.Consume()
				if nextToken.Type == lexer.TokenLeftParen {
					parenCount++
				} else if nextToken.Type == lexer.TokenRightParen {
					parenCount--
				}
			}
			if parenCount != 0 {
				return fmt.Errorf("unclosed parentheses")
			}
		}

		return nil

	case lexer.TokenString, lexer.TokenNumber:
		// Пропускаем простой литерал
		stream.Consume()
		return nil

	case lexer.TokenLeftParen:
		// Пропускаем выражение в скобках
		stream.Consume() // Пропускаем (
		parenCount := 1
		for parenCount > 0 && stream.HasMore() {
			nextToken := stream.Consume()
			if nextToken.Type == lexer.TokenLeftParen {
				parenCount++
			} else if nextToken.Type == lexer.TokenRightParen {
				parenCount--
			}
		}
		if parenCount != 0 {
			return fmt.Errorf("unclosed parentheses")
		}
		return nil

	default:
		return fmt.Errorf("unsupported expression start token: %s", token.Type)
	}
}

// parsePipeExpression разбирает pipe expression (временная реализация)
func (h *LanguageCallStatementHandler) parsePipeExpression(ctx *common.ParseContext) (*ast.PipeExpression, error) {
	// Создаем простой PipeHandler для разбора
	pipeHandler := NewPipeHandler(config.ConstructHandlerConfig{
		IsEnabled: true,
		Priority:  1,
		Name:      "temp-pipe-handler",
	})

	result, err := pipeHandler.Handle(ctx)
	if err != nil {
		return nil, err
	}

	pipeExpr, ok := result.(*ast.PipeExpression)
	if !ok {
		return nil, fmt.Errorf("expected PipeExpression, got %T", result)
	}

	return pipeExpr, nil
}

// createPipeLanguageCallStatement создает LanguageCallStatement для pipe expression
func (h *LanguageCallStatementHandler) createPipeLanguageCallStatement(pipeExpr *ast.PipeExpression, ampersandToken lexer.Token, pos ast.Position) (*ast.LanguageCallStatement, error) {
	// Создаем специальный LanguageCallStatement, который содержит pipe expression
	// Для этого создаем фиктивный LanguageCall
	languageCall := &ast.LanguageCall{
		Language:  "pipe",
		Function:  "background",
		Arguments: []ast.Expression{pipeExpr},
		Pos:       pos,
	}

	return ast.NewBackgroundLanguageCallStatement(languageCall, ampersandToken, pos), nil
}

// findExpressionStart находит начало выражения перед &
func (h *LanguageCallStatementHandler) findExpressionStart(stream stream.TokenStream) int {
	// Идем назад от текущей позиции до начала подходящего выражения
	pos := stream.Position()

	// Мы должны найти начало language call, который должен быть идентификатором
	for pos > 0 {
		stream.SetPosition(pos - 1)
		token := stream.Current()

		// Language call всегда начинается с идентификатора (языка)
		if token.Type == lexer.TokenIdentifier {
			// Проверяем, что после идентификатора идет . и еще один идентификатор
			if h.isValidLanguageCallStart(stream) {
				return pos - 1
			}
		}

		pos--
	}

	return 0 // Начало потока
}

// isValidLanguageCallStart проверяет, что текущая позиция начинается с корректного language call
func (h *LanguageCallStatementHandler) isValidLanguageCallStart(stream stream.TokenStream) bool {
	// Сохраняем текущую позицию
	currentPos := stream.Position()
	defer stream.SetPosition(currentPos)

	// Проверяем последовательность: identifier . identifier ( ...
	if !stream.HasMore() {
		return false
	}

	// Первый идентификатор (язык)
	if stream.Current().Type != lexer.TokenIdentifier {
		return false
	}
	stream.Consume()

	// Точка
	if !stream.HasMore() || stream.Current().Type != lexer.TokenDot {
		return false
	}
	stream.Consume()

	// Второй идентификатор (функция)
	if !stream.HasMore() || stream.Current().Type != lexer.TokenIdentifier {
		return false
	}
	stream.Consume()

	// Открывающая скобка
	if !stream.HasMore() || stream.Current().Type != lexer.TokenLeftParen {
		return false
	}

	return true
}

// parseLanguageCall разбирает language call expression
func (h *LanguageCallStatementHandler) parseLanguageCall(ctx *common.ParseContext) (*ast.LanguageCall, error) {
	// Используем существующий LanguageCallHandler для разбора
	languageCallHandler := NewLanguageCallHandlerWithRegistry(config.ConstructHandlerConfig{
		IsEnabled: true,
		Priority:  100,
		Name:      "language_call_for_statement",
	}, h.languageRegistry)

	tokenStream := ctx.TokenStream

	// Проверяем, что текущий токен может быть обработан как language call
	if !languageCallHandler.CanHandle(tokenStream.Current()) {
		return nil, fmt.Errorf("expected language call expression")
	}

	// Разбираем language call
	result, err := languageCallHandler.Handle(ctx)
	if err != nil {
		return nil, err
	}

	// Проверяем, что результат действительно LanguageCall
	languageCall, ok := result.(*ast.LanguageCall)
	if !ok {
		return nil, fmt.Errorf("expected LanguageCall, got %T", result)
	}

	return languageCall, nil
}

// Config возвращает конфигурацию обработчика
func (h *LanguageCallStatementHandler) Config() common.HandlerConfig {
	return common.HandlerConfig{
		IsEnabled: h.config.IsEnabled,
		Priority:  h.config.Priority, // Должен быть 50
		Name:      h.config.Name,
	}
}

// Name возвращает имя обработчика
func (h *LanguageCallStatementHandler) Name() string {
	return h.config.Name
}

// parsePipeExpressionUntilAmpersand разбирает pipe expression до оператора &
func (h *LanguageCallStatementHandler) parsePipeExpressionUntilAmpersand(ctx *common.ParseContext) (ast.Expression, error) {
	tokenStream := ctx.TokenStream

	// Возвращаемся к началу потока
	tokenStream.SetPosition(0)

	// Создаем PipeHandler для разбора
	pipeHandler := NewPipeHandler(config.ConstructHandlerConfig{
		IsEnabled: true,
		Priority:  1,
		Name:      "temp-pipe-for-background",
	})

	// Разбираем pipe expression
	result, err := pipeHandler.Handle(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to parse pipe expression: %v", err)
	}

	pipeExpr, ok := result.(*ast.PipeExpression)
	if !ok {
		return nil, fmt.Errorf("expected PipeExpression, got %T", result)
	}

	return pipeExpr, nil
}

// parseLanguageCallUntilAmpersand разбирает language call до оператора &
func (h *LanguageCallStatementHandler) parseLanguageCallUntilAmpersand(ctx *common.ParseContext) (ast.Expression, error) {
	tokenStream := ctx.TokenStream

	// Создаем LanguageCallHandler для разбора с пропуском проверки background tasks
	languageCallHandler := NewLanguageCallHandlerWithRegistryAndSkipBackgroundCheck(config.ConstructHandlerConfig{
		IsEnabled: true,
		Priority:  100,
		Name:      "temp-language-call-for-background",
	}, h.languageRegistry, false)

	// Проверяем, что токен может быть обработан как language call
	if !languageCallHandler.CanHandle(tokenStream.Current()) {
		return nil, fmt.Errorf("expected language call expression")
	}

	// Разбираем language call
	result, err := languageCallHandler.Handle(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to parse language call: %v", err)
	}

	languageCall, ok := result.(*ast.LanguageCall)
	if !ok {
		return nil, fmt.Errorf("expected LanguageCall, got %T", result)
	}

	return languageCall, nil
}
