package handler

import (
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

	// Проверяем guard рекурсии
	if ctx.Guard != nil {
		if err := ctx.Guard.Enter(); err != nil {
			return nil, err
		}
		defer ctx.Guard.Exit()
	}

	// Проверяем, что текущий токен - открывающая скобка
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenLeftParen {
		return nil, newErrorWithPos(tokenStream, "expected '('")
	}

	// Потребляем открывающую скобку
	leftParenToken := tokenStream.Consume()

	// Разбираем выражение внутри скобок
	expression, err := h.parseExpressionInsideParentheses(ctx)
	if err != nil {
		return nil, newErrorWithPos(tokenStream, "failed to parse expression inside parentheses: %v", err)
	}

	// Проверяем закрывающую скобку
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenRightParen {
		return nil, newErrorWithPos(tokenStream, "expected ')' after expression")
	}
	rightParenToken := tokenStream.Consume()

	// Если выражение внутри - это PipeExpression, возвращаем её напрямую
	if pipeExpr, ok := expression.(*ast.PipeExpression); ok {
		return pipeExpr, nil
	}

	// Для выражений (Expression) проверяем, есть ли бинарные операторы после скобок
	if expr, ok := expression.(ast.Expression); ok {
		// Проверяем, есть ли бинарные операторы после закрывающей скобки
		if h.hasBinaryOperatorAfterParentheses(tokenStream) {
			// Используем BinaryExpressionHandler для обработки операторов после скобок
			binaryHandler := NewBinaryExpressionHandlerWithVerbose(config.ConstructHandlerConfig{}, false)
			result, err := binaryHandler.ParseFullExpression(ctx, expr)
			if err != nil {
				return nil, newErrorWithPos(tokenStream, "failed to parse expression after parentheses: %v", err)
			}
			return result, nil
		}
		// Нет операторов после скобок, возвращаем выражение напрямую
		return expr, nil
	}

	// Для всех остальных случаев создаем ParenthesesNode (для синтаксических скобок)
	parenthesesNode := ast.NewParenthesesNode(leftParenToken, rightParenToken)

	// Если есть выражение внутри, добавляем его как дочерний узел
	if expression != nil {
		if exprNode, ok := expression.(ast.Node); ok {
			parenthesesNode.AddChild(exprNode)
		}
	}

	return parenthesesNode, nil
}

// parseExpressionInsideParentheses разбирает выражение внутри скобок
func (h *ParenthesizedExpressionHandler) parseExpressionInsideParentheses(ctx *common.ParseContext) (ast.Expression, error) {
	tokenStream := ctx.TokenStream

	if !tokenStream.HasMore() {
		return nil, newErrorWithPos(tokenStream, "unexpected end of input inside parentheses")
	}

	// Проверяем тип выражения внутри скобок
	token := tokenStream.Current()

	// Обрабатываем пустые скобки ()
	if token.Type == lexer.TokenRightParen {
		// Пустые скобки - возвращаем nil, так как выражение будет создано в Handle
		return nil, nil
	}

	// Проверяем, является ли это вложенными скобками
	if token.Type == lexer.TokenLeftParen {
		// Вложенные скобки - рекурсивно вызываем этот же обработчик
		result, err := h.Handle(ctx)
		if err != nil {
			return nil, err
		}
		// Результат может быть ParenthesesNode, который является Node, но не Expression
		if _, ok := result.(*ast.ParenthesesNode); ok {
			// Возвращаем nil для пустых вложенных скобок
			return nil, nil
		}
		expr, ok := result.(ast.Expression)
		if !ok {
			return nil, newErrorWithPos(tokenStream, "expected Expression, got %T", result)
		}
		return expr, nil
	}

	// Пробуем разобрать как сложное выражение (language call или pipe)
	return h.parseComplexExpression(ctx)
}

// parseComplexExpression разбирает сложное выражение (language call, pipe или обычное выражение)
func (h *ParenthesizedExpressionHandler) parseComplexExpression(ctx *common.ParseContext) (ast.Expression, error) {
	tokenStream := ctx.TokenStream

	// Проверяем, является ли это qualified variable (language.identifier)
	if h.isQualifiedVariable(tokenStream) {
		return h.parseQualifiedVariableInParentheses(ctx)
	}

	// Проверяем, является ли это language call (language.function(...) или language.module.function(...))
	if h.isLanguageCall(tokenStream) {
		langCall, err := h.parseLanguageCallInParentheses(ctx)
		if err != nil {
			return nil, err
		}

		// После language call проверяем следующие токены
		if tokenStream.HasMore() {
			nextToken := tokenStream.Current()
			
			// Если есть pipe operator после language call, распарсим как pipe expression
			if nextToken.Type == lexer.TokenPipe || nextToken.Type == lexer.TokenBitwiseOr {
				// Создаём парсер pipe expression, который начнётся с уже распарсенного первого языкового вызова
				pipeExpr, err := h.parsePipeExpressionFromFirstStage(ctx, langCall)
				if err != nil {
					return nil, err
				}
				return pipeExpr, nil
			}
			
			// Проверяем, есть ли бинарный оператор после language call (но не pipe)
			if h.hasBinaryOperatorAfterParentheses(tokenStream) {
				// Используем UnifiedExpressionParser, но передаем уже распарсенный левый операнд
				unifiedParser := NewUnifiedExpressionParser(false)
				result, err := unifiedParser.ContinueParsingExpression(ctx, langCall)
				if err != nil {
					return nil, err
				}
				return result, nil
			}
		}

		return langCall, nil
	}

	// Если ничего из вышеперечисленного, пробуем разобрать как обычное выражение
	// Используем UnifiedExpressionParser для разбора арифметических выражений
	unifiedParser := NewUnifiedExpressionParser(false)
	result, err := unifiedParser.ParseExpression(ctx)
	if err != nil {
		return nil, err
	}
	
	// Если результат - это BinaryExpression и содержит pipe operator |
	// попробуем распарсить как pipe expression (для случаев вроде (a | b))
	if binExpr, ok := result.(*ast.BinaryExpression); ok {
		if binExpr.Operator == "|" || binExpr.Operator == "|>" {
			// Проверяем, что левый и правый операнды - это language calls или identifiers (признаки pipe)
			isLeftPipe := h.isPipeableExpression(binExpr.Left)
			isRightPipe := h.isPipeableExpression(binExpr.Right)
			
			if isLeftPipe && isRightPipe {
				// Возможно это pipe expression, преобразуем BinaryExpression в PipeExpression
				// Собираем все stages из вложенных BinaryExpressions
				stages, operators := h.flattenPipeExpression(binExpr)
				if len(stages) > 0 {
					// Получаем позицию из первого stage
					pos := binExpr.Left.Position()
					return ast.NewPipeExpression(stages, operators, pos), nil
				}
			}
		}
	}
	
	return result, nil
}

// parsePipeExpressionFromFirstStage разбирает pipe expression, который начинается с уже распарсенного первого stage
func (h *ParenthesizedExpressionHandler) parsePipeExpressionFromFirstStage(ctx *common.ParseContext, firstStage ast.Expression) (ast.Expression, error) {
	tokenStream := ctx.TokenStream
	
	stages := []ast.Expression{firstStage}
	operators := []lexer.Token{}
	
	// Продолжаем разбирать, пока есть pipe операторы и не дошли до закрывающей скобки
	for tokenStream.HasMore() && (tokenStream.Current().Type == lexer.TokenPipe || tokenStream.Current().Type == lexer.TokenBitwiseOr) {
		// Сохраняем оператор |
		pipeToken := tokenStream.Consume()
		operators = append(operators, pipeToken)
		
		// Проверяем, что после pipe есть выражение и это не закрывающая скобка
		if !tokenStream.HasMore() || tokenStream.Current().Type == lexer.TokenRightParen {
			return nil, newErrorWithPos(tokenStream, "expected expression after pipe operator")
		}
		
		// Проверяем, что следующий токен может начинать выражение
		if !h.isValidExpressionStart(tokenStream.Current()) {
			return nil, newErrorWithTokenPos(tokenStream.Current(), "invalid expression start after pipe: %s", tokenStream.Current().Type)
		}
		
		// Разбираем следующее выражение используя UnifiedExpressionParser
		unifiedParser := NewUnifiedExpressionParser(false)
		nextExpr, err := unifiedParser.ParseExpression(ctx)
		if err != nil {
			return nil, newErrorWithPos(tokenStream, "failed to parse expression after pipe: %v", err)
		}
		stages = append(stages, nextExpr)
		
		// Если следующий токен - закрывающая скобка, прекращаем разбор
		if tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenRightParen {
			break
		}
	}
	
	if len(stages) < 2 {
		return nil, newErrorWithPos(tokenStream, "pipe expression must have at least 2 stages")
	}
	
	// Создаем PipeExpression
	pos := ast.Position{
		Line:   operators[0].Line,
		Column: operators[0].Column,
		Offset: operators[0].Position,
	}
	
	return ast.NewPipeExpression(stages, operators, pos), nil
}

// isPipeableExpression проверяет, может ли выражение быть этапом pipe (language call или identifier)
func (h *ParenthesizedExpressionHandler) isPipeableExpression(expr ast.Expression) bool {
	if expr == nil {
		return false
	}
	_, isLangCall := expr.(*ast.LanguageCall)
	idExpr, isIdentifier := expr.(*ast.Identifier)
	// Qualified identifier - это просто Identifier с флагом Qualified
	isQualifiedID := isIdentifier && idExpr != nil && idExpr.Qualified
	
	return isLangCall || isIdentifier || isQualifiedID
}

// flattenPipeExpression собирает все stages из вложенных BinaryExpressions (для pipe operators)
// Возвращает stages и строки операторов ("|" или "|>")
func (h *ParenthesizedExpressionHandler) flattenPipeExpression(binExpr *ast.BinaryExpression) ([]ast.Expression, []lexer.Token) {
	var stages []ast.Expression
	var operators []lexer.Token
	
	// Собираем stages слева направо
	current := binExpr
	for {
		if leftBin, ok := current.Left.(*ast.BinaryExpression); ok && 
			(leftBin.Operator == "|" || leftBin.Operator == "|>") {
			// Левая часть тоже pipe-like выражение, обрабатываем её рекурсивно
			leftStages, leftOps := h.flattenPipeExpression(leftBin)
			stages = append(stages, leftStages...)
			operators = append(operators, leftOps...)
		} else {
			// Левая часть - финальный stage
			stages = append(stages, current.Left)
		}
		
		// Преобразуем оператор string в lexer.Token
		// Это хак, но нам нужны токены для PipeExpression
		opToken := lexer.Token{
			Value:    current.Operator,
			Line:     current.Pos.Line,
			Column:   current.Pos.Column,
			Position: current.Pos.Offset,
		}
		if current.Operator == "|" {
			opToken.Type = lexer.TokenBitwiseOr
		} else if current.Operator == "|>" {
			opToken.Type = lexer.TokenPipe
		}
		operators = append(operators, opToken)
		
		// Если правая часть не pipe-like выражение, добавляем её и выходим
		if rightBin, ok := current.Right.(*ast.BinaryExpression); !ok || 
			(rightBin.Operator != "|" && rightBin.Operator != "|>") {
			stages = append(stages, current.Right)
			break
		}
		
		// Иначе продолжаем обработку
		current = current.Right.(*ast.BinaryExpression)
	}
	
	return stages, operators
}

// isQualifiedVariable проверяет, является ли выражение qualified variable (language.identifier)
func (h *ParenthesizedExpressionHandler) isQualifiedVariable(stream stream.TokenStream) bool {
	// Сохраняем текущую позицию
	currentPos := stream.Position()
	defer stream.SetPosition(currentPos)

	// Проверяем, что первый токен - языковой
	if !stream.HasMore() {
		return false
	}
	token := stream.Current()
	if !h.isLanguageToken(token) {
		return false
	}

	// Потребляем языковой токен
	stream.Consume()

	// Проверяем наличие точки
	if !stream.HasMore() || stream.Current().Type != lexer.TokenDot {
		return false
	}
	stream.Consume()

	// Проверяем наличие идентификатора
	if !stream.HasMore() || stream.Current().Type != lexer.TokenIdentifier {
		return false
	}
	stream.Consume()

	// Проверяем, что следующий токен - закрывающая скобка (означает конец qualified variable)
	if !stream.HasMore() || stream.Current().Type != lexer.TokenRightParen {
		return false
	}

	return true
}

// parseQualifiedVariableInParentheses разбирает qualified variable внутри скобок
func (h *ParenthesizedExpressionHandler) parseQualifiedVariableInParentheses(ctx *common.ParseContext) (ast.Expression, error) {
	tokenStream := ctx.TokenStream

	// Потребляем языковой токен
	languageToken := tokenStream.Consume()

	// Потребляем точку
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenDot {
		return nil, newErrorWithPos(tokenStream, "expected '.' after language token")
	}
	tokenStream.Consume()

	// Потребляем идентификатор
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenIdentifier {
		return nil, newErrorWithPos(tokenStream, "expected identifier after '.'")
	}
	identifierToken := tokenStream.Consume()

	// Создаем qualified identifier
	return ast.NewQualifiedIdentifier(languageToken, identifierToken, languageToken.Value, identifierToken.Value), nil
}

// hasPipeOperatorInExpression проверяет, есть ли в выражении оператор |> (pipe)
func (h *ParenthesizedExpressionHandler) hasPipeOperatorInExpression(stream stream.TokenStream) bool {
	// Сохраняем текущую позицию
	currentPos := stream.Position()
	defer stream.SetPosition(currentPos)

	// Ищем токен | или |> с учетом вложенных скобок
	parenDepth := 0
	for stream.HasMore() {
		token := stream.Current()
		if token.Type == lexer.TokenPipe || token.Type == lexer.TokenBitwiseOr { // оба | и |>
			return true
		}
		if token.Type == lexer.TokenLeftParen {
			parenDepth++
		} else if token.Type == lexer.TokenRightParen {
			if parenDepth == 0 {
				return false // Дошли до закрывающей скобки того же уровня, pipe не найден
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

	// Разбираем первое выражение используя UnifiedExpressionParser
	unifiedParser := NewUnifiedExpressionParser(false)
	firstExpr, err := unifiedParser.ParseExpression(ctx)
	if err != nil {
		return nil, newErrorWithPos(tokenStream, "failed to parse first expression in pipe: %v", err)
	}

	// Разбираем последовательность | expr, но только до закрывающей скобки
	stages := []ast.Expression{firstExpr}
	operators := []lexer.Token{}

	// Продолжаем разбирать, пока есть pipe операторы и не дошли до закрывающей скобки
	for tokenStream.HasMore() && (tokenStream.Current().Type == lexer.TokenPipe || tokenStream.Current().Type == lexer.TokenBitwiseOr) {
		// Сохраняем оператор |
		pipeToken := tokenStream.Consume()
		operators = append(operators, pipeToken)

		// Проверяем, что после pipe есть выражение и это не закрывающая скобка
		if !tokenStream.HasMore() || tokenStream.Current().Type == lexer.TokenRightParen {
			return nil, newErrorWithPos(tokenStream, "expected expression after pipe operator")
		}

		// Проверяем, что следующий токен может начинать выражение
		if !h.isValidExpressionStart(tokenStream.Current()) {
			return nil, newErrorWithTokenPos(tokenStream.Current(), "invalid expression start after pipe: %s", tokenStream.Current().Type)
		}

		// Разбираем следующее выражение используя UnifiedExpressionParser
		nextExpr, err := unifiedParser.ParseExpression(ctx)
		if err != nil {
			return nil, newErrorWithPos(tokenStream, "failed to parse expression after pipe: %v", err)
		}
		stages = append(stages, nextExpr)

		// Если следующий токен - закрывающая скобка, прекращаем разбор
		if tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenRightParen {
			break
		}
	}

	if len(stages) < 2 {
		return nil, newErrorWithPos(tokenStream, "pipe expression must have at least 2 stages")
	}

	// Создаем PipeExpression
	pos := ast.Position{
		Line:   operators[0].Line,
		Column: operators[0].Column,
		Offset: operators[0].Position,
	}

	return ast.NewPipeExpression(stages, operators, pos), nil
}

// parseLanguageCallInParentheses разбирает language call внутри скобок (включая qualified calls)
func (h *ParenthesizedExpressionHandler) parseLanguageCallInParentheses(ctx *common.ParseContext) (ast.Expression, error) {
	tokenStream := ctx.TokenStream

	if !tokenStream.HasMore() {
		return nil, newErrorWithPos(tokenStream, "unexpected end of input")
	}

	token := tokenStream.Current()

	// Проверяем, что это языковой токен
	if !h.isLanguageToken(token) {
		return nil, newErrorWithTokenPos(token, "expected language token, got %s", token.Type)
	}

	// Потребляем языковой токен
	languageToken := tokenStream.Consume()

	// Проверяем наличие точки
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenDot {
		return nil, newErrorWithPos(tokenStream, "expected '.' after language token")
	}
	tokenStream.Consume() // dotToken

	// Собираем путь функции (может быть lua.string.len или просто lua.len)
	var functionPath []string
	for tokenStream.HasMore() {
		// Проверяем наличие идентификатора
		if tokenStream.Current().Type != lexer.TokenIdentifier {
			return nil, newErrorWithPos(tokenStream, "expected identifier in function path")
		}
		pathPart := tokenStream.Consume().Value
		functionPath = append(functionPath, pathPart)

		// Проверяем, что следует дальше
		if !tokenStream.HasMore() {
			return nil, newErrorWithPos(tokenStream, "unexpected end of input in function path")
		}

		// Если это точка - может быть еще один идентификатор
		if tokenStream.Current().Type == lexer.TokenDot {
			tokenStream.Consume()
			continue
		}

		// Если это открывающая скобка - конец пути
		if tokenStream.Current().Type == lexer.TokenLeftParen {
			break
		}

		return nil, newErrorWithTokenPos(tokenStream.Current(), "unexpected token in function path: %s", tokenStream.Current().Type)
	}

	// Проверяем открывающую скобку
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenLeftParen {
		return nil, newErrorWithPos(tokenStream, "expected '(' after function name")
	}
	tokenStream.Consume() // leftParenToken

	// Парсим аргументы до закрывающей скобки
	arguments := []ast.Expression{}
	for tokenStream.HasMore() && tokenStream.Current().Type != lexer.TokenRightParen {
		// Пропускаем запятые между аргументами
		if tokenStream.Current().Type == lexer.TokenComma {
			tokenStream.Consume()
			continue
		}

		// Парсим аргумент
		arg, err := h.parseArgument(tokenStream)
		if err != nil {
			return nil, err
		}
		arguments = append(arguments, arg)

		// Проверяем, есть ли еще аргументы
		if tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenComma {
			tokenStream.Consume()
			continue
		}
	}

	// Проверяем закрывающую скобку
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenRightParen {
		return nil, newErrorWithPos(tokenStream, "expected ')' after arguments")
	}
	tokenStream.Consume() // rightParenToken

	// Собираем полное имя функции (например "string.len" из "lua.string.len")
	functionName := ""
	for i, part := range functionPath {
		if i > 0 {
			functionName += "."
		}
		functionName += part
	}

	// Создаем LanguageCall
	languageCall := &ast.LanguageCall{
		Language:  languageToken.Value,
		Function:  functionName,
		Arguments: arguments,
		Pos: ast.Position{
			Line:   languageToken.Line,
			Column: languageToken.Column,
			Offset: languageToken.Position,
		},
	}

	return languageCall, nil
}

// isLanguageToken проверяет, является ли токен языковым токеном
func (h *ParenthesizedExpressionHandler) isLanguageToken(token lexer.Token) bool {
	switch token.Type {
	case lexer.TokenLua, lexer.TokenPython, lexer.TokenPy, lexer.TokenGo, lexer.TokenNode, lexer.TokenJS:
		return true
	default:
		return false
	}
}

// isLanguageCall проверяет, является ли выражение language call (language.function(...) или language.module.function(...))
func (h *ParenthesizedExpressionHandler) isLanguageCall(stream stream.TokenStream) bool {
	// Сохраняем текущую позицию
	currentPos := stream.Position()
	defer stream.SetPosition(currentPos)

	// Проверяем, что первый токен - языковой
	if !stream.HasMore() {
		return false
	}
	token := stream.Current()
	if !h.isLanguageToken(token) {
		return false
	}

	// Потребляем языковой токен
	stream.Consume()

	// Проверяем наличие точки
	if !stream.HasMore() || stream.Current().Type != lexer.TokenDot {
		return false
	}
	stream.Consume()

	// Потребляем идентификаторы (может быть несколько для qualified calls like lua.string.len)
	// Продолжаем потреблять идентификаторы и точки пока не найдем открывающую скобку
	for stream.HasMore() {
		if stream.Current().Type != lexer.TokenIdentifier {
			return false
		}
		stream.Consume()
		
		// Проверяем, что следует дальше
		if !stream.HasMore() {
			return false
		}
		
		// Если это точка - может быть еще один идентификатор
		if stream.Current().Type == lexer.TokenDot {
			stream.Consume()
			continue
		}
		
		// Если это открывающая скобка - это language call
		if stream.Current().Type == lexer.TokenLeftParen {
			return true
		}
		
		// Иначе - не language call
		return false
	}

	return false
}

// parseArgument разбирает аргумент функции
func (h *ParenthesizedExpressionHandler) parseArgument(tokenStream stream.TokenStream) (ast.Expression, error) {
	if !tokenStream.HasMore() {
		return nil, newErrorWithPos(tokenStream, "unexpected end of input in argument")
	}

	token := tokenStream.Current()

	switch token.Type {
	case lexer.TokenString:
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
		numToken := tokenStream.Consume()
		value, err := parseNumber(numToken.Value)
		if err != nil {
			return nil, newErrorWithTokenPos(numToken, "invalid number format: %s", numToken.Value)
		}
		return createNumberLiteral(numToken, value), nil

	default:
		return nil, newErrorWithTokenPos(token, "unsupported argument type: %s", token.Type)
	}
}

// parseSingleExpression разбирает одно выражение (упрощенная версия)
func (h *ParenthesizedExpressionHandler) parseSingleExpression(ctx *common.ParseContext) (ast.Expression, error) {
	tokenStream := ctx.TokenStream

	if !tokenStream.HasMore() {
		return nil, newErrorWithPos(tokenStream, "unexpected end of input")
	}

	token := tokenStream.Current()

	// Проверяем, что токен может начинать выражение
	if !h.isValidExpressionStart(token) {
		return nil, newErrorWithTokenPos(token, "invalid expression start token: %s", token.Type)
	}

	switch token.Type {
	case lexer.TokenIdentifier, lexer.TokenLua, lexer.TokenPython, lexer.TokenPy, lexer.TokenGo, lexer.TokenNode, lexer.TokenJS:
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
		value, err := parseNumber(numToken.Value)
		if err != nil {
			return nil, newErrorWithTokenPos(numToken, "invalid number format: %s", numToken.Value)
		}
		return createNumberLiteral(numToken, value), nil

	case lexer.TokenLeftParen:
		// Вложенные скобки - рекурсивно вызываем этот же обработчик
		result, err := h.Handle(ctx)
		if err != nil {
			return nil, err
		}
		// Результат может быть ParenthesesNode, который является Node, но не Expression
		if _, ok := result.(*ast.ParenthesesNode); ok {
			// Возвращаем nil для пустых вложенных скобок
			return nil, nil
		}
		expr, ok := result.(ast.Expression)
		if !ok {
			return nil, newErrorWithPos(tokenStream, "expected Expression, got %T", result)
		}
		return expr, nil

	default:
		return nil, newErrorWithTokenPos(token, "unsupported token type: %s", token.Type)
	}
}

// collectExpressionTokens собирает токены выражения до закрывающей скобки
func (h *ParenthesizedExpressionHandler) collectExpressionTokens(tokenStream stream.TokenStream) ([]lexer.Token, error) {
	var tokens []lexer.Token
	depth := 0

	for tokenStream.HasMore() {
		token := tokenStream.Current()

		// Проверяем на закрывающую скобку (на уровне 0)
		if depth == 0 && token.Type == lexer.TokenRightParen {
			break
		}

		// Отслеживаем вложенность скобок
		if token.Type == lexer.TokenLeftParen {
			depth++
		} else if token.Type == lexer.TokenRightParen {
			depth--
			if depth < 0 {
				return nil, newErrorWithTokenPos(token, "unexpected closing bracket")
			}
		}

		tokens = append(tokens, token)
		tokenStream.Consume()
	}

	return tokens, nil
}

// hasBinaryOperatorAfterParentheses проверяет, есть ли бинарные операторы после закрывающей скобки
func (h *ParenthesizedExpressionHandler) hasBinaryOperatorAfterParentheses(tokenStream stream.TokenStream) bool {
	// Сохраняем текущую позицию
	currentPos := tokenStream.Position()
	defer tokenStream.SetPosition(currentPos)

	// Проверяем следующий токен
	if !tokenStream.HasMore() {
		return false
	}

	token := tokenStream.Current()
	// Проверяем распространенные бинарные операторы
	switch token.Type {
	case lexer.TokenPlus, lexer.TokenMinus, lexer.TokenMultiply, lexer.TokenSlash,
		lexer.TokenEqual, lexer.TokenNotEqual, lexer.TokenLess, lexer.TokenLessEqual,
		lexer.TokenGreater, lexer.TokenGreaterEqual, lexer.TokenAnd, lexer.TokenOr,
		lexer.TokenBitwiseOr, lexer.TokenDoubleLeftAngle, lexer.TokenDoubleRightAngle,
		lexer.TokenModulo, lexer.TokenConcat, lexer.TokenPower, lexer.TokenCaret,
		lexer.TokenQuestion:
		return true
	default:
		return false
	}
}

// isValidExpressionStart проверяет, может ли токен начинать выражение
func (h *ParenthesizedExpressionHandler) isValidExpressionStart(token lexer.Token) bool {
	switch token.Type {
	case lexer.TokenIdentifier, lexer.TokenLua, lexer.TokenPython, lexer.TokenPy, lexer.TokenGo, lexer.TokenNode, lexer.TokenJS, lexer.TokenString, lexer.TokenNumber, lexer.TokenLeftParen:
		return true
	default:
		return false
	}
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
