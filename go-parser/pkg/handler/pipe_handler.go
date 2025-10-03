package handler

import (
	"fmt"

	"go-parser/pkg/ast"
	"go-parser/pkg/common"
	"go-parser/pkg/config"
	"go-parser/pkg/lexer"
	"go-parser/pkg/stream"
)

// PipeHandler - обработчик для pipe expressions (|)
type PipeHandler struct {
	config config.ConstructHandlerConfig
}

// NewPipeHandler создает новый обработчик для pipe expressions
func NewPipeHandler(config config.ConstructHandlerConfig) *PipeHandler {
	return &PipeHandler{
		config: config,
	}
}

// CanHandle проверяет, может ли обработчик обработать токен
func (h *PipeHandler) CanHandle(token lexer.Token) bool {
	// PipeHandler обрабатывает идентификаторы и токены языков, но проверяет наличие | в потоке
	// Должен иметь самый низкий приоритет (1)
	return token.IsLanguageIdentifierOrCallToken()
}

// Handle обрабатывает pipe expression
func (h *PipeHandler) Handle(ctx *common.ParseContext) (interface{}, error) {
	tokenStream := ctx.TokenStream

	// Проверяем, есть ли в потоке оператор | после текущего выражения
	if !h.hasPipeOperatorInStream(tokenStream) {
		// Если нет |, то это не pipe expression, передаем управление другому обработчику
		// Возвращаем nil, nil чтобы указать, что этот обработчик не может обработать данный токен
		return nil, nil
	}

	// Проверяем, является ли это присваиванием с pipe expression
	if h.isAssignmentWithPipeExpression(tokenStream) {
		// Обрабатываем присваивание с pipe expression
		return h.parseAssignmentWithPipeExpression(ctx)
	}

	// Разбираем pipe expression
	result, err := h.parsePipeExpression(ctx)
	if err != nil {
	}
	return result, err
}

// hasPipeOperatorInStream проверяет, есть ли в потоке оператор | после текущего выражения
func (h *PipeHandler) hasPipeOperatorInStream(stream stream.TokenStream) bool {
	// Сохраняем текущую позицию
	currentPos := stream.Position()
	defer stream.SetPosition(currentPos)

	// Ищем оператор | в оставшейся части потока
	for stream.HasMore() {
		token := stream.Current()
		if token.Type == lexer.TokenPipe {
			return true
		}
		stream.Consume()
	}

	return false
}

// isAssignmentWithPipeExpression проверяет, является ли выражение присваиванием с pipe expression
func (h *PipeHandler) isAssignmentWithPipeExpression(stream stream.TokenStream) bool {
	// Сохраняем текущую позицию
	currentPos := stream.Position()
	defer stream.SetPosition(currentPos)

	// Ищем паттерн: qualified.variable = expression | expression
	// Пропускаем квалифицированную переменную слева
	for stream.HasMore() {
		token := stream.Current()
		if token.Type == lexer.TokenAssign {
			// Нашли знак присваивания, проверяем есть ли pipe expression после него
			stream.Consume() // потребляем =

			// Ищем оператор | после присваивания
			for stream.HasMore() {
				token := stream.Current()
				if token.Type == lexer.TokenPipe {
					return true
				}
				stream.Consume()
			}
			return false
		}
		stream.Consume()
	}

	return false
}

// parseAssignmentWithPipeExpression разбирает присваивание с pipe expression
func (h *PipeHandler) parseAssignmentWithPipeExpression(ctx *common.ParseContext) (*ast.VariableAssignment, error) {
	tokenStream := ctx.TokenStream

	// Разбираем левую часть (квалифицированную переменную)
	leftExpr, err := h.parseLeftSideOfAssignment(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to parse left side of assignment: %v", err)
	}

	// Потребляем знак присваивания
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenAssign {
		return nil, fmt.Errorf("expected '=' after variable")
	}
	assignToken := tokenStream.Consume()

	// Разбираем pipe expression справа от знака присваивания
	pipeExpr, err := h.parsePipeExpression(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to parse pipe expression in assignment: %v", err)
	}

	// Создаем присваивание
	identifier, ok := leftExpr.(*ast.Identifier)
	if !ok {
		return nil, fmt.Errorf("left side of assignment must be a qualified variable")
	}

	return ast.NewVariableAssignment(identifier, assignToken, pipeExpr), nil
}

// parseLeftSideOfAssignment разбирает левую часть присваивания (квалифицированную переменную)
func (h *PipeHandler) parseLeftSideOfAssignment(ctx *common.ParseContext) (ast.Expression, error) {
	tokenStream := ctx.TokenStream

	if !tokenStream.HasMore() {
		return nil, fmt.Errorf("unexpected end of input")
	}

	// Потребляем языковой токен
	langToken := tokenStream.Consume()
	if !langToken.IsLanguageIdentifierOrCallToken() {
		return nil, fmt.Errorf("expected language token, got %s", langToken.Type)
	}

	// Проверяем наличие DOT
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenDot {
		return nil, fmt.Errorf("expected '.' after language token")
	}
	tokenStream.Consume() // потребляем DOT

	// Проверяем наличие идентификатора переменной
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenIdentifier {
		return nil, fmt.Errorf("expected variable name after '.'")
	}
	varToken := tokenStream.Consume()

	// Определяем язык
	language := langToken.LanguageTokenToString()

	// Создаем квалифицированный идентификатор
	return ast.NewQualifiedIdentifier(langToken, varToken, language, varToken.Value), nil
}

// parsePipeExpression разбирает последовательность выражений соединенных |
func (h *PipeHandler) parsePipeExpression(ctx *common.ParseContext) (*ast.PipeExpression, error) {
	tokenStream := ctx.TokenStream

	stages := []ast.Expression{}
	operators := []lexer.Token{}

	// Разбираем первое выражение
	firstExpr, err := h.parseSingleExpression(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to parse first expression in pipe: %v", err)
	}
	stages = append(stages, firstExpr)

	// Разбираем последовательность | expr
	for tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenPipe {
		// Сохраняем оператор |
		pipeToken := tokenStream.Consume()
		operators = append(operators, pipeToken)

		// Проверяем, что после pipe есть выражение
		if !tokenStream.HasMore() {
			return nil, fmt.Errorf("unexpected end of input after pipe operator")
		}

		// Проверяем, что следующий токен может начинать выражение
		if !h.isValidExpressionStart(tokenStream.Current()) {
			return nil, fmt.Errorf("invalid expression start after pipe operator: %s", tokenStream.Current().Type)
		}

		// Разбираем следующее выражение
		nextExpr, err := h.parseSingleExpression(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to parse expression after pipe: %v", err)
		}
		stages = append(stages, nextExpr)
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

// parsePipeExpressionWithLeftAssociative разбирает pipe expression с учетом того,
// что мы уже находимся на токене | и должны разобрать выражение слева и справа
func (h *PipeHandler) parsePipeExpressionWithLeftAssociative(ctx *common.ParseContext) (*ast.PipeExpression, error) {
	tokenStream := ctx.TokenStream

	// Сохраняем токен |
	pipeToken := tokenStream.Consume()

	// Нам нужно найти выражение слева от |
	// Для этого мы должны вернуться назад и разобрать выражение до |
	leftExpr, err := h.parseExpressionBeforePipe(ctx, pipeToken)
	if err != nil {
		return nil, fmt.Errorf("failed to parse expression before pipe: %v", err)
	}

	// Теперь разбираем выражение справа от |
	rightExpr, err := h.parseExpressionAfterPipe(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to parse expression after pipe: %v", err)
	}

	// Создаем PipeExpression с двумя стадиями
	stages := []ast.Expression{leftExpr, rightExpr}
	operators := []lexer.Token{pipeToken}

	pos := ast.Position{
		Line:   pipeToken.Line,
		Column: pipeToken.Column,
		Offset: pipeToken.Position,
	}

	return ast.NewPipeExpression(stages, operators, pos), nil
}

// parseExpressionBeforePipe разбирает выражение, находящееся перед оператором |
func (h *PipeHandler) parseExpressionBeforePipe(ctx *common.ParseContext, pipeToken lexer.Token) (ast.Expression, error) {
	tokenStream := ctx.TokenStream

	// Возвращаемся к началу потока для разбора выражения слева
	tokenStream.SetPosition(0)

	// Создаем контекст для разбора выражения до |
	exprCtx := &common.ParseContext{
		TokenStream: tokenStream,
		Parser:      ctx.Parser,
		Depth:       ctx.Depth,
		MaxDepth:    ctx.MaxDepth,
		Guard:       ctx.Guard,
	}

	// Разбираем выражение до |
	return h.parseSingleExpressionUntilPipe(exprCtx, pipeToken)
}

// parseExpressionAfterPipe разбирает выражение, находящееся после оператора |
func (h *PipeHandler) parseExpressionAfterPipe(ctx *common.ParseContext) (ast.Expression, error) {
	tokenStream := ctx.TokenStream

	if !tokenStream.HasMore() {
		return nil, fmt.Errorf("unexpected end of input after pipe operator")
	}

	// Проверяем, что следующий токен может начинать выражение
	nextToken := tokenStream.Current()
	if !h.isValidExpressionStart(nextToken) {
		return nil, fmt.Errorf("invalid token after pipe operator: %s", nextToken.Type)
	}

	// Разбираем следующее выражение
	return h.parseSingleExpression(ctx)
}

// parseSingleExpressionUntilPipe разбирает одно выражение до достижения оператора |
func (h *PipeHandler) parseSingleExpressionUntilPipe(ctx *common.ParseContext, stopAtPipe lexer.Token) (ast.Expression, error) {
	tokenStream := ctx.TokenStream

	if !tokenStream.HasMore() {
		return nil, fmt.Errorf("unexpected end of input")
	}

	token := tokenStream.Current()

	// Проверяем, не достигли ли мы оператора |
	if token.Type == lexer.TokenPipe {
		return nil, fmt.Errorf("expected expression before pipe operator")
	}

	switch token.Type {
	case lexer.TokenIdentifier:
		// Это может быть language call или простой идентификатор
		return h.parseLanguageCallOrIdentifierUntilPipe(ctx, stopAtPipe)

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
		// Сгруппированное выражение в скобках
		return h.parseParenthesizedExpressionUntilPipe(ctx, stopAtPipe)

	default:
		// Проверяем, является ли токен языковым токеном
		if token.IsLanguageIdentifierOrCallToken() {
			// Разбираем как language call или идентификатор
			return h.parseLanguageCallOrIdentifierUntilPipe(ctx, stopAtPipe)
		}
		return nil, fmt.Errorf("unsupported token type in pipe expression: %s", token.Type)
	}
}

// parseLanguageCallOrIdentifierUntilPipe разбирает language call или простой идентификатор до достижения |
func (h *PipeHandler) parseLanguageCallOrIdentifierUntilPipe(ctx *common.ParseContext, stopAtPipe lexer.Token) (ast.Expression, error) {
	// Пробуем разобрать как language call вручную (без использования другого обработчика)
	return h.parseLanguageCallManuallyUntilPipe(ctx, stopAtPipe)
}

// parseLanguageCallManuallyUntilPipe разбирает language call вручную до достижения |
func (h *PipeHandler) parseLanguageCallManuallyUntilPipe(ctx *common.ParseContext, stopAtPipe lexer.Token) (ast.Expression, error) {
	tokenStream := ctx.TokenStream

	if !tokenStream.HasMore() {
		return nil, fmt.Errorf("unexpected end of input")
	}

	// Потребляем токен языка (имя языка)
	langToken := tokenStream.Consume()
	if !langToken.IsLanguageIdentifierOrCallToken() {
		return nil, fmt.Errorf("expected language identifier")
	}

	// Проверяем наличие точки
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenDot {
		// Если нет точки, это просто идентификатор
		return ast.NewIdentifier(langToken, langToken.Value), nil
	}

	// Потребляем точку
	tokenStream.Consume()

	// Проверяем наличие идентификатора функции
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenIdentifier {
		return nil, fmt.Errorf("expected function identifier after dot")
	}

	// Потребляем идентификатор функции
	funcToken := tokenStream.Consume()

	// Проверяем наличие открывающей скобки
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenLeftParen {
		// Если нет скобок, это квалифицированное имя
		qualifiedName := langToken.Value + "." + funcToken.Value
		return ast.NewIdentifier(funcToken, qualifiedName), nil
	}

	// Потребляем открывающую скобку
	tokenStream.Consume()

	// Разбираем аргументы до закрывающей скобки или |
	args := []ast.Expression{}

	for tokenStream.HasMore() && tokenStream.Current().Type != lexer.TokenRightParen && tokenStream.Current().Type != lexer.TokenPipe {
		// Пропускаем запятые между аргументами
		if tokenStream.Current().Type == lexer.TokenComma {
			tokenStream.Consume()
			continue
		}

		// Разбираем аргумент
		arg, err := h.parseSingleExpressionUntilPipe(ctx, stopAtPipe)
		if err != nil {
			return nil, fmt.Errorf("failed to parse argument: %v", err)
		}
		args = append(args, arg)
	}

	// Проверяем закрывающую скобку
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenRightParen {
		return nil, fmt.Errorf("expected ')' after arguments")
	}
	tokenStream.Consume() // Потребляем закрывающую скобку

	// Создаем LanguageCall
	pos := ast.Position{
		Line:   langToken.Line,
		Column: langToken.Column,
		Offset: langToken.Position,
	}

	return &ast.LanguageCall{
		Language:  langToken.Value,
		Function:  funcToken.Value,
		Arguments: args,
		Pos:       pos,
	}, nil
}

// parseParenthesizedExpressionUntilPipe разбирает выражение в скобках до достижения |
func (h *PipeHandler) parseParenthesizedExpressionUntilPipe(ctx *common.ParseContext, stopAtPipe lexer.Token) (ast.Expression, error) {
	tokenStream := ctx.TokenStream

	// Потребляем открывающую скобку
	tokenStream.Consume()

	// Разбираем выражение внутри скобок до |
	expr, err := h.parseSingleExpressionUntilPipe(ctx, stopAtPipe)
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

// hasPipeOperator проверяет, есть ли в потоке оператор |
func (h *PipeHandler) hasPipeOperator(stream stream.TokenStream) bool {
	// Сохраняем текущую позицию
	currentPos := stream.Position()
	defer stream.SetPosition(currentPos)

	// Пропускаем текущее выражение (идентификатор и возможные аргументы)
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
func (h *PipeHandler) skipExpression(stream stream.TokenStream) error {
	if !stream.HasMore() {
		return fmt.Errorf("unexpected end of stream")
	}

	token := stream.Current()

	switch token.Type {
	case lexer.TokenIdentifier:
		// Пропускаем идентификатор
		stream.Consume()

		// Пропускаем возможную точку и следующий идентификатор (для qualified.variable)
		if stream.HasMore() && stream.Current().Type == lexer.TokenDot {
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
		// Проверяем, является ли токен языковым токеном
		if token.IsLanguageIdentifierOrCallToken() {
			// Пропускаем языковой токен
			stream.Consume()

			// Пропускаем возможную точку и следующий идентификатор (для language.function)
			if stream.HasMore() && stream.Current().Type == lexer.TokenDot {
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
		}
		return fmt.Errorf("unsupported expression start token: %s", token.Type)

	}
}

// parsePipeExpressionLeftAssociative разбирает пайплайн с left-associative семантикой
func (h *PipeHandler) parsePipeExpressionLeftAssociative(ctx *common.ParseContext, firstPipeToken lexer.Token) (interface{}, error) {
	tokenStream := ctx.TokenStream

	// Нам нужно вернуться назад и разобрать выражение до первого |
	// Это сложная задача, так как нам нужно найти начало первого выражения

	// Создаем копию потока токенов для поиска начала
	clone := tokenStream.Clone()

	// Ищем начало первого выражения (перед первым |)
	startPos := h.findExpressionStart(clone)
	if startPos < 0 {
		return nil, fmt.Errorf("could not find start of pipe expression")
	}

	// Возвращаемся к найденной позиции
	tokenStream.SetPosition(startPos)

	// Теперь разбираем пайплайн слева направо
	return h.parsePipeSequence(ctx)
}

// findExpressionStart находит начало первого выражения в пайплайне
func (h *PipeHandler) findExpressionStart(stream stream.TokenStream) int {
	// Идем назад до начала потока или до нахождения подходящего начала выражения
	pos := stream.Position()

	for pos > 0 {
		stream.SetPosition(pos - 1)
		token := stream.Current()

		// Если мы находим токен, который может начинать выражение, это наш кандидат
		if h.isExpressionStartToken(token) {
			return pos - 1
		}

		pos--
	}

	return 0 // Начало потока
}

// isExpressionStartToken проверяет, может ли токен начинать выражение
func (h *PipeHandler) isExpressionStartToken(token lexer.Token) bool {
	switch token.Type {
	case lexer.TokenIdentifier:
		return true
	case lexer.TokenString:
		return true
	case lexer.TokenNumber:
		return true
	case lexer.TokenLeftParen:
		return true
	// Добавьте другие токены, которые могут начинать выражение
	default:
		return false
	}
}

// parsePipeSequence разбирает последовательность выражений соединенных |
func (h *PipeHandler) parsePipeSequence(ctx *common.ParseContext) (*ast.PipeExpression, error) {
	tokenStream := ctx.TokenStream

	stages := []ast.Expression{}
	operators := []lexer.Token{}

	// Разбираем первое выражение
	firstExpr, err := h.parseSingleExpression(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to parse first expression in pipe: %v", err)
	}
	stages = append(stages, firstExpr)

	// Разбираем последовательность | expr
	for tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenPipe {
		// Сохраняем оператор |
		pipeToken := tokenStream.Consume()
		operators = append(operators, pipeToken)

		// Разбираем следующее выражение
		nextExpr, err := h.parseSingleExpression(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to parse expression after pipe: %v", err)
		}
		stages = append(stages, nextExpr)
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

// parseSingleExpression разбирает одно выражение в пайплайне
func (h *PipeHandler) parseSingleExpression(ctx *common.ParseContext) (ast.Expression, error) {
	tokenStream := ctx.TokenStream

	if !tokenStream.HasMore() {
		return nil, fmt.Errorf("unexpected end of input in pipe expression")
	}

	token := tokenStream.Current()

	// Проверяем, что токен может начинать выражение
	if !h.isValidExpressionStart(token) {
		return nil, fmt.Errorf("invalid expression start token: %s", token.Type)
	}

	switch token.Type {
	case lexer.TokenIdentifier:
		// Это может быть language call или простой идентификатор
		// Используем AssignmentHandler для парсинга сложных выражений
		assignmentHandler := NewAssignmentHandler(100, 1)

		// Создаем временный контекст
		tempCtx := &common.ParseContext{
			TokenStream: tokenStream,
			Parser:      ctx.Parser,
			Depth:       ctx.Depth,
			MaxDepth:    ctx.MaxDepth,
			Guard:       ctx.Guard,
		}

		// Используем parseComplexExpression для разбора выражения
		result, err := assignmentHandler.parseComplexExpression(tempCtx)
		if err != nil {
			return nil, fmt.Errorf("failed to parse complex expression in pipe: %v", err)
		}

		// В pipe expression всегда преобразуем QualifiedIdentifier в LanguageCall
		// Это необходимо для правильного выполнения pipe выражений
		if identifier, ok := result.(*ast.Identifier); ok {
			if identifier.Qualified {
				// Создаем LanguageCall без аргументов
				languageCall := &ast.LanguageCall{
					Language:  identifier.Language,
					Function:  identifier.Name,
					Arguments: []ast.Expression{}, // Пустые аргументы
					Pos:       identifier.Pos,
				}
				return languageCall, nil
			}
		}

		// Если это не QualifiedIdentifier, возвращаем результат как есть
		return result, nil

	case lexer.TokenString:
		// Строковый литерал - разрешаем в pipe expression
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
		// Числовой литерал - НЕ разрешаем в pipe expression (только language calls и идентификаторы)
		return nil, fmt.Errorf("numbers cannot be used as pipe stages, only language calls or identifiers")

	case lexer.TokenLeftParen:
		// Сгруппированное выражение в скобках
		return h.parseParenthesizedExpression(ctx)

	default:
		// Проверяем, является ли токен языковым токеном
		if token.IsLanguageIdentifierOrCallToken() {
			// Разбираем как language call
			return h.parseLanguageCallOrIdentifier(ctx)
		}
		return nil, fmt.Errorf("unsupported token type in pipe expression: %s", token.Type)
	}
}

// isFollowedByPipeOperator проверяет, следует ли за текущим выражением оператор |
func (h *PipeHandler) isFollowedByPipeOperator(stream stream.TokenStream) bool {
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

// isValidExpressionStart проверяет, может ли токен начинать выражение
func (h *PipeHandler) isValidExpressionStart(token lexer.Token) bool {
	switch token.Type {
	case lexer.TokenIdentifier, lexer.TokenString, lexer.TokenNumber, lexer.TokenLeftParen:
		return true
	default:
		// Проверяем, является ли токен языковым токеном
		return token.IsLanguageIdentifierOrCallToken()
	}
}

// parseLanguageCallOrIdentifier разбирает language call или простой идентификатор
func (h *PipeHandler) parseLanguageCallOrIdentifier(ctx *common.ParseContext) (ast.Expression, error) {
	tokenStream := ctx.TokenStream

	// Сохраняем текущую позицию на случай отката
	currentPos := tokenStream.Position()

	// Пробуем разобрать как language call через существующий обработчик
	languageCallHandler := NewLanguageCallHandler(config.ConstructHandlerConfig{
		IsEnabled: true,
		Priority:  100, // Высокий приоритет для language calls
		Name:      "language_call_in_pipe",
	})

	if languageCallHandler.CanHandle(tokenStream.Current()) {
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
		if err == nil {
			if langCall, ok := result.(*ast.LanguageCall); ok {
				return langCall, nil
			}
		} else {
		}
	}

	// Если не получилось разобрать как language call, откатываемся и разбираем как идентификатор
	tokenStream.SetPosition(currentPos)

	// Разбираем как простой идентификатор
	identToken := tokenStream.Consume()
	identifier := ast.NewIdentifier(identToken, identToken.Value)

	return identifier, nil
}

// parseParenthesizedExpression разбирает выражение в скобках
func (h *PipeHandler) parseParenthesizedExpression(ctx *common.ParseContext) (ast.Expression, error) {
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

// Config возвращает конфигурацию обработчика
func (h *PipeHandler) Config() common.HandlerConfig {
	return common.HandlerConfig{
		IsEnabled: h.config.IsEnabled,
		Priority:  h.config.Priority, // Должен быть 1 (самый низкий)
		Name:      h.config.Name,
	}
}

// Name возвращает имя обработчика
func (h *PipeHandler) Name() string {
	return h.config.Name
}
