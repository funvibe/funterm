package handler

import (
	"fmt"
	"math/big"

	"go-parser/pkg/ast"
	"go-parser/pkg/common"
	"go-parser/pkg/config"
	"go-parser/pkg/lexer"
	"go-parser/pkg/stream"
)

// isUnaryOperator проверяет, является ли токен унарным оператором
func (h *MatchHandler) isUnaryOperator(tokenType lexer.TokenType) bool {
	switch tokenType {
	case lexer.TokenPlus, lexer.TokenMinus, lexer.TokenNot, lexer.TokenTilde, lexer.TokenAt:
		return true
	default:
		return false
	}
}

// MatchHandler - обработчик для match конструкций
type MatchHandler struct {
	config  config.ConstructHandlerConfig
	verbose bool
}

// NewMatchHandler создает новый обработчик для match конструкций
func NewMatchHandler(config config.ConstructHandlerConfig) *MatchHandler {
	return NewMatchHandlerWithVerbose(config, false)
}

// NewMatchHandlerWithVerbose создает новый обработчик для match конструкций с поддержкой verbose режима
func NewMatchHandlerWithVerbose(config config.ConstructHandlerConfig, verbose bool) *MatchHandler {
	return &MatchHandler{
		config:  config,
		verbose: verbose,
	}
}

// CanHandle проверяет, может ли обработчик обработать токен
func (h *MatchHandler) CanHandle(token lexer.Token) bool {
	return token.Type == lexer.TokenMatch
}

// Handle обрабатывает match конструкцию
func (h *MatchHandler) Handle(ctx *common.ParseContext) (interface{}, error) {
	tokenStream := ctx.TokenStream

	// 1. Потребляем 'match'
	matchToken := tokenStream.Consume()

	// 2. Парсим выражение для сопоставления
	expression, err := h.parseExpression(tokenStream)
	if err != nil {
		return nil, err
	}

	// 3. Потребляем '{'
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenLBrace {
		return nil, newErrorWithPos(tokenStream, "expected '{' after match expression")
	}
	lBraceToken := tokenStream.Consume() // {

	// 4. Парсим ветки сопоставления
	arms, err := h.parseMatchArms(tokenStream)
	if err != nil {
		return nil, err
	}

	// 5. Потребляем '}'
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenRBrace {
		return nil, newErrorWithPos(tokenStream, "expected '}' at end of match statement")
	}
	rBraceToken := tokenStream.Consume() // }

	// 6. Создаем MatchStatement
	startPos := matchHandlerTokenToPosition(matchToken)
	matchStmt := &ast.MatchStatement{
		Expression:  expression,
		Arms:        arms,
		MatchToken:  matchToken,
		LBraceToken: lBraceToken,
		RBraceToken: rBraceToken,
		Pos:         startPos,
	}

	return matchStmt, nil
}

// parseExpression парсит выражение после match
func (h *MatchHandler) parseExpression(tokenStream stream.TokenStream) (ast.Expression, error) {
	// ВРЕМЕННО: Возвращаемся к старой логике из-за проблем с LanguageCallHandler
	// TODO: Интегрировать UnifiedExpressionParser после исправления LanguageCallHandler

	// Сначала парсим левую часть выражения
	left, err := h.parsePrimaryOrComplexExpressionWithDepth(tokenStream, 0)
	if err != nil {
		return nil, err
	}

	// Проверяем, есть ли бинарный оператор после левой части
	if tokenStream.HasMore() {
		nextToken := tokenStream.Current()
		if h.isBinaryOperator(nextToken.Type) {
			// Это бинарное выражение
			return h.parseBinaryExpressionWithLeft(tokenStream, left)
		} else if nextToken.Type == lexer.TokenQuestion {
			// Это тернарный оператор
			return h.parseTernaryExpression(tokenStream, left)
		}
	}

	return left, nil
}

// isBinaryOperator проверяет, является ли токен бинарным оператором
func (h *MatchHandler) isBinaryOperator(tokenType lexer.TokenType) bool {
	switch tokenType {
	case lexer.TokenGreater, lexer.TokenLess, lexer.TokenGreaterEqual, lexer.TokenLessEqual,
		lexer.TokenEqual, lexer.TokenNotEqual, lexer.TokenPlus, lexer.TokenMinus, lexer.TokenMultiply, lexer.TokenSlash,
		lexer.TokenAnd, lexer.TokenOr, lexer.TokenModulo, lexer.TokenDoubleRightAngle, lexer.TokenDoubleLeftAngle,
		lexer.TokenAmpersand, lexer.TokenCaret, lexer.TokenConcat:
		return true
	default:
		return false
	}
}

// parseBinaryExpressionWithLeft парсит бинарное выражение с уже распарсенной левой частью
func (h *MatchHandler) parseBinaryExpressionWithLeft(tokenStream stream.TokenStream, left ast.Expression) (ast.Expression, error) {
	// Потребляем оператор
	operatorToken := tokenStream.Consume()
	operator := operatorToken.Value

	// Парсим правую часть
	right, err := h.parsePrimaryOrComplexExpression(tokenStream)
	if err != nil {
		return nil, newErrorWithPos(tokenStream, "failed to parse right operand: %v", err)
	}

	// Создаем бинарное выражение
	binaryExpr := ast.NewBinaryExpression(left, operator, right, matchHandlerTokenToPosition(operatorToken))

	// Проверяем наличие дополнительных операторов (для цепочек типа a == b && c == d)
	if tokenStream.HasMore() && h.isBinaryOperator(tokenStream.Current().Type) {
		return h.parseBinaryExpressionWithLeft(tokenStream, binaryExpr)
	}

	return binaryExpr, nil
}

// parseTernaryExpression парсит тернарное выражение с уже распарсенной левой частью
func (h *MatchHandler) parseTernaryExpression(tokenStream stream.TokenStream, condition ast.Expression) (ast.Expression, error) {
	if h.verbose {
		fmt.Printf("DEBUG: parseTernaryExpression - starting, current token: %v\n", tokenStream.Current())
	}

	// Потребляем '?'
	questionToken := tokenStream.Consume()
	if h.verbose {
		fmt.Printf("DEBUG: parseTernaryExpression - consumed '?', current token: %v\n", tokenStream.Current())
	}

	// Парсим true-branch
	trueBranch, err := h.parsePrimaryOrComplexExpression(tokenStream)
	if err != nil {
		return nil, newErrorWithPos(tokenStream, "failed to parse true branch of ternary expression: %v", err)
	}
	if h.verbose {
		fmt.Printf("DEBUG: parseTernaryExpression - parsed true branch, current token: %v\n", tokenStream.Current())
	}

	// Проверяем и потребляем ':'
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenColon {
		return nil, newErrorWithPos(tokenStream, "expected ':' after true branch of ternary expression")
	}
	colonToken := tokenStream.Consume()
	if h.verbose {
		fmt.Printf("DEBUG: parseTernaryExpression - consumed ':', current token: %v\n", tokenStream.Current())
	}

	// Парсим false-branch - здесь может быть еще один тернарный оператор
	// Для вложенных тернарных операторов нам нужно быть осторожными с парсингом
	falseBranch, err := h.parseTernaryFalseBranch(tokenStream)
	if err != nil {
		return nil, newErrorWithPos(tokenStream, "failed to parse false branch of ternary expression: %v", err)
	}
	if h.verbose {
		fmt.Printf("DEBUG: parseTernaryExpression - parsed false branch, current token: %v\n", tokenStream.Current())
	}

	return ast.NewTernaryExpression(condition, questionToken, colonToken, trueBranch, falseBranch, condition.Position()), nil
}

// parseTernaryFalseBranch парсит false-ветвь тернарного оператора с поддержкой вложенных тернарных операторов
func (h *MatchHandler) parseTernaryFalseBranch(tokenStream stream.TokenStream) (ast.Expression, error) {
	if h.verbose {
		fmt.Printf("DEBUG: parseTernaryFalseBranch - starting, current token: %v\n", tokenStream.Current())
	}

	// Парсим левую часть выражения
	left, err := h.parsePrimaryOrComplexExpressionWithDepth(tokenStream, 0)
	if err != nil {
		return nil, err
	}
	if h.verbose {
		fmt.Printf("DEBUG: parseTernaryFalseBranch - parsed left part, current token: %v\n", tokenStream.Current())
	}

	// Проверяем, есть ли бинарный оператор
	if tokenStream.HasMore() && h.isBinaryOperator(tokenStream.Current().Type) {
		// Парсим бинарное выражение
		left, err = h.parseBinaryExpressionWithLeft(tokenStream, left)
		if err != nil {
			return nil, err
		}
		if h.verbose {
			fmt.Printf("DEBUG: parseTernaryFalseBranch - parsed binary expression, current token: %v\n", tokenStream.Current())
		}
	}

	// Проверяем, есть ли тернарный оператор
	if tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenQuestion {
		// Парсим вложенный тернарный оператор
		result, err := h.parseTernaryExpression(tokenStream, left)
		if h.verbose {
			fmt.Printf("DEBUG: parseTernaryFalseBranch - parsed nested ternary, current token: %v, result: %v, error: %v\n", tokenStream.Current(), result, err)
		}
		return result, err
	}

	if h.verbose {
		fmt.Printf("DEBUG: parseTernaryFalseBranch - finished, current token: %v, result: %v\n", tokenStream.Current(), left)
	}
	return left, nil
}

// parsePrimaryOrComplexExpression парсит первичные или сложные выражения
func (h *MatchHandler) parsePrimaryOrComplexExpression(tokenStream stream.TokenStream) (ast.Expression, error) {
	return h.parsePrimaryOrComplexExpressionWithDepth(tokenStream, 0)
}

// parsePrimaryOrComplexExpressionWithDepth парсит первичные или сложные выражения с отслеживанием глубины скобок
func (h *MatchHandler) parsePrimaryOrComplexExpressionWithDepth(tokenStream stream.TokenStream, parenDepth int) (ast.Expression, error) {
	currentToken := tokenStream.Current()

	// Проверяем унарные операторы
	if h.isUnaryOperator(currentToken.Type) {
		unaryHandler := NewUnaryExpressionHandler(config.ConstructHandlerConfig{})
		ctx := &common.ParseContext{
			TokenStream: tokenStream,
			Depth:       0,
			MaxDepth:    100,
		}
		result, err := unaryHandler.Handle(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to parse unary expression: %v", err)
		}
		if expr, ok := result.(*ast.UnaryExpression); ok {
			return expr, nil
		}
		return nil, fmt.Errorf("expected UnaryExpression, got %T", result)
	}

	switch currentToken.Type {
	case lexer.TokenLeftParen:
		// Выражение в скобках - используем общий парсер для обычных выражений
		expr, err := h.parseGeneralParenthesizedExpression(tokenStream)
		if err != nil {
			return nil, err
		}

		// После скобочного выражения может быть тернарный оператор
		// В отличие от size выражений, здесь мы продолжаем парсинг
		return expr, nil
	case lexer.TokenString:
		// Строковый литерал
		token := tokenStream.Consume()
		return &ast.StringLiteral{
			Value: token.Value,
			Pos:   matchHandlerTokenToPosition(token),
		}, nil
	case lexer.TokenNumber:
		// Числовой литерал
		token := tokenStream.Consume()
		numValue, err := parseNumber(token.Value)
		if err != nil {
			return nil, newErrorWithTokenPos(token, "invalid number format: %s", token.Value)
		}
		return createNumberLiteral(token, numValue), nil
	case lexer.TokenIdentifier, lexer.TokenPy, lexer.TokenLua, lexer.TokenPython, lexer.TokenGo, lexer.TokenNode, lexer.TokenJS:
		if tokenStream.HasMore() {
			nextToken := tokenStream.Peek()
			if nextToken.Type == lexer.TokenDot {
				// Проверяем, есть ли еще токены после DOT
				if tokenStream.HasMore() {
					// Используем PeekN(3) чтобы проверить токен после DOT и следующего
					thirdToken := tokenStream.PeekN(3)
					if thirdToken.Type == lexer.TokenLBracket {
						// Если после DOT и IDENTIFIER идет LBRACKET, это индексный доступ
						return h.parseLanguageIndexExpression(tokenStream)
					} else {
						// Иначе это language call
						return h.parseLanguageCall(tokenStream)
					}
				} else {
					// Если нет больше токенов, это language call
					return h.parseLanguageCall(tokenStream)
				}
			} else if nextToken.Type == lexer.TokenLBracket {
				// Если следующий токен LBRACKET, это индексный доступ
				return h.parseIndexExpression(tokenStream)
			} else {
				// Иначе это простой идентификатор
				identifierToken := tokenStream.Consume()
				return &ast.Identifier{
					Name: identifierToken.Value,
					Pos:  matchHandlerTokenToPosition(identifierToken),
				}, nil
			}
		} else {
			// Иначе это простой идентификатор
			identifierToken := tokenStream.Consume()
			return &ast.Identifier{
				Name: identifierToken.Value,
				Pos:  matchHandlerTokenToPosition(identifierToken),
			}, nil
		}
	case lexer.TokenLBracket:
		// Массив как выражение
		return h.parseArrayExpression(tokenStream)
	case lexer.TokenLBrace:
		// Объект как выражение
		return h.parseObjectExpression(tokenStream)
	default:
		return nil, newErrorWithTokenPos(currentToken, "unsupported expression type: %s", currentToken.Type)
	}
}

// parseParenthesizedExpression парсит выражение в скобках (общий случай)
func (h *MatchHandler) parseParenthesizedExpression(tokenStream stream.TokenStream) (ast.Expression, error) {
	return h.parseGeneralParenthesizedExpression(tokenStream)
}

// parseGeneralParenthesizedExpression парсит выражение в скобках для обычных выражений
func (h *MatchHandler) parseGeneralParenthesizedExpression(tokenStream stream.TokenStream) (ast.Expression, error) {
	if h.verbose {
		fmt.Printf("DEBUG: parseGeneralParenthesizedExpression - starting, current token: %v\n", tokenStream.Current())
	}

	// Потребляем левую скобку
	_ = tokenStream.Consume() // (
	if h.verbose {
		fmt.Printf("DEBUG: parseGeneralParenthesizedExpression - consumed '(', current token: %v\n", tokenStream.Current())
	}

	// Парсим внутреннее выражение как полное выражение (без ограничений)
	expr, err := h.parseExpression(tokenStream)
	if err != nil {
		return nil, err
	}
	if h.verbose {
		fmt.Printf("DEBUG: parseGeneralParenthesizedExpression - parsed internal expression, current token: %v\n", tokenStream.Current())
	}

	// Потребляем правую скобку
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenRightParen {
		return nil, newErrorWithPos(tokenStream, "expected ')' after parenthesized expression")
	}
	_ = tokenStream.Consume() // )
	if h.verbose {
		fmt.Printf("DEBUG: parseGeneralParenthesizedExpression - consumed ')', current token: %v\n", tokenStream.Current())
	}

	// Создаем NestedExpression для сохранения скобок в AST
	parenthesizedExpr := ast.NewNestedExpression(expr, expr.Position())

	return parenthesizedExpr, nil
}

// parseSizeParenthesizedExpression парсит выражение в скобках для size expressions
func (h *MatchHandler) parseSizeParenthesizedExpression(tokenStream stream.TokenStream, parenDepth int) (ast.Expression, error) {
	if h.verbose {
		fmt.Printf("DEBUG: parseSizeParenthesizedExpression - starting, current token: %v\n", tokenStream.Current())
	}

	// Потребляем левую скобку
	_ = tokenStream.Consume() // (
	if h.verbose {
		fmt.Printf("DEBUG: parseSizeParenthesizedExpression - consumed '(', current token: %v\n", tokenStream.Current())
	}

	// Увеличиваем глубину скобок
	newDepth := parenDepth + 1

	// Парсим внутреннее выражение с ограничениями size expression
	expr, err := h.parseSizeExpressionLimitedWithDepth(tokenStream, newDepth)
	if err != nil {
		return nil, err
	}
	if h.verbose {
		fmt.Printf("DEBUG: parseSizeParenthesizedExpression - parsed internal expression, current token: %v\n", tokenStream.Current())
	}

	// Потребляем правую скобку
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenRightParen {
		return nil, newErrorWithPos(tokenStream, "expected ')' in size expression")
	}
	_ = tokenStream.Consume() // )
	if h.verbose {
		fmt.Printf("DEBUG: parseSizeParenthesizedExpression - consumed ')', current token: %v\n", tokenStream.Current())
	}

	// Создаем NestedExpression для сохранения скобок в AST
	parenthesizedExpr := ast.NewNestedExpression(expr, expr.Position())

	return parenthesizedExpr, nil
}

// parseLanguageCall парсит вызов функции другого языка
func (h *MatchHandler) parseLanguageCall(tokenStream stream.TokenStream) (ast.Expression, error) {

	// 1. Читаем язык
	languageToken := tokenStream.Consume()
	language := languageToken.Value

	// 2. Проверяем и потребляем DOT
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenDot {
		return nil, newErrorWithPos(tokenStream, "expected DOT after language '%s'", language)
	}
	tokenStream.Consume() // Consuming dot

	// 3. Читаем имя функции (может содержать точки, например, math.sqrt)
	functionParts := []string{}

	// Читаем первый идентификатор функции
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenIdentifier {
		return nil, newErrorWithPos(tokenStream, "expected function name after DOT")
	}
	functionToken := tokenStream.Consume()
	functionParts = append(functionParts, functionToken.Value)

	// Читаем дополнительные DOT и идентификаторы
	for tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenDot {
		tokenStream.Consume() // Consuming dot

		if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenIdentifier {
			return nil, newErrorWithPos(tokenStream, "expected function name after dot")
		}
		functionToken = tokenStream.Consume()
		functionParts = append(functionParts, functionToken.Value)
	}

	// Собираем полное имя функции (например, "math.sqrt")
	functionName := ""
	for i, part := range functionParts {
		if i > 0 {
			functionName += "."
		}
		functionName += part
	}

	// 4. Проверяем, есть ли открывающая скобка (аргументы)
	arguments := make([]ast.Expression, 0)

	if tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenLeftParen {
		// Есть аргументы - потребляем открывающую скобку
		tokenStream.Consume() // Consuming '('

		// 5. Читаем аргументы
		// Проверяем, есть ли аргументы
		if !tokenStream.HasMore() {
			return nil, newErrorWithPos(tokenStream, "unexpected EOF after '('")
		}

		if tokenStream.Current().Type != lexer.TokenRightParen {
			// Есть хотя бы один аргумент
			for {
				if !tokenStream.HasMore() {
					return nil, newErrorWithPos(tokenStream, "unexpected EOF in arguments")
				}

				// Читаем аргумент как сложное выражение с поддержкой тернарных операторов
				if h.verbose {
					fmt.Printf("DEBUG: parseLanguageCall - starting to parse argument, current token: %v\n", tokenStream.Current())
				}

				// Сначала парсим левую часть выражения
				if h.verbose {
					fmt.Printf("DEBUG: parseLanguageCall - about to call parsePrimaryOrComplexExpression\n")
				}
				left, err := h.parsePrimaryOrComplexExpressionWithDepth(tokenStream, 0)
				if err != nil {
					return nil, newErrorWithPos(tokenStream, "failed to parse function argument: %v", err)
				}
				if h.verbose {
					fmt.Printf("DEBUG: parseLanguageCall - parsePrimaryOrComplexExpression returned, current token: %v\n", tokenStream.Current())
				}

				// Проверяем, есть ли бинарный оператор или тернарный оператор после левой части
				var arg ast.Expression
				if tokenStream.HasMore() {
					nextToken := tokenStream.Current()
					if h.verbose {
						fmt.Printf("DEBUG: parseLanguageCall - nextToken: %v\n", nextToken)
					}
					if h.isBinaryOperator(nextToken.Type) {
						// Это бинарное выражение
						if h.verbose {
							fmt.Printf("DEBUG: parseLanguageCall - parsing binary expression\n")
						}
						arg, err = h.parseBinaryExpressionWithLeft(tokenStream, left)
						if err != nil {
							return nil, newErrorWithPos(tokenStream, "failed to parse binary expression in argument: %v", err)
						}

						// После парсинга бинарного выражения проверяем, есть ли тернарный оператор
						if tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenQuestion {
							if h.verbose {
								fmt.Printf("DEBUG: parseLanguageCall - parsing ternary expression after binary expression\n")
							}
							arg, err = h.parseTernaryExpression(tokenStream, arg)
							if err != nil {
								return nil, newErrorWithPos(tokenStream, "failed to parse ternary expression after binary expression: %v", err)
							}
						}
					} else if nextToken.Type == lexer.TokenQuestion {
						// Это тернарный оператор
						if h.verbose {
							fmt.Printf("DEBUG: parseLanguageCall - parsing ternary expression\n")
						}
						arg, err = h.parseTernaryExpression(tokenStream, left)
						if err != nil {
							return nil, newErrorWithPos(tokenStream, "failed to parse ternary expression in argument: %v", err)
						}
					} else {
						// Это простое выражение
						if h.verbose {
							fmt.Printf("DEBUG: parseLanguageCall - using simple expression\n")
						}
						arg = left
					}
				} else {
					// Это простое выражение
					if h.verbose {
						fmt.Printf("DEBUG: parseLanguageCall - no more tokens, using simple expression\n")
					}
					arg = left
				}

				// Проверяем, есть ли бинарный оператор после аргумента
				if tokenStream.HasMore() && h.isBinaryOperator(tokenStream.Current().Type) {
					if h.verbose {
						fmt.Printf("DEBUG: parseLanguageCall - parsing binary expression after argument\n")
					}
					// Парсим бинарное выражение
					arg, err = h.parseBinaryExpressionWithLeft(tokenStream, arg)
					if err != nil {
						return nil, newErrorWithPos(tokenStream, "failed to parse binary expression after argument: %v", err)
					}
				}

				if h.verbose {
					fmt.Printf("DEBUG: parseLanguageCall - finished parsing argument, next token: %v\n", tokenStream.Current())
				}

				arguments = append(arguments, arg)

				// Проверяем разделитель или конец
				if !tokenStream.HasMore() {
					return nil, newErrorWithPos(tokenStream, "unexpected EOF after argument")
				}

				nextToken := tokenStream.Current()

				if nextToken.Type == lexer.TokenComma {
					tokenStream.Consume() // Consuming comma
					// После запятой должен быть аргумент
					if !tokenStream.HasMore() {
						return nil, newErrorWithPos(tokenStream, "unexpected EOF after comma")
					}
					if tokenStream.Current().Type == lexer.TokenRightParen {
						return nil, newErrorWithPos(tokenStream, "unexpected ')' after comma")
					}
				} else if nextToken.Type == lexer.TokenRightParen {
					break
				} else {
					return nil, newErrorWithTokenPos(nextToken, "expected ',' or ')' after argument, got %s", nextToken.Type)
				}
			}
		}

		// 6. Проверяем закрывающую скобку
		if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenRightParen {
			return nil, newErrorWithPos(tokenStream, "expected ')' after arguments")
		}
		tokenStream.Consume() // Consuming ')'
	}

	// 7. Создаем узел AST
	startPos := matchHandlerTokenToPosition(languageToken)

	node := &ast.LanguageCall{
		Language:  language,
		Function:  functionName,
		Arguments: arguments,
		Pos:       startPos,
	}

	return node, nil
}

// parseIndexExpression парсит индексное выражение вроде lua.data[i]
func (h *MatchHandler) parseIndexExpression(tokenStream stream.TokenStream) (ast.Expression, error) {

	// 1. Читаем базовый идентификатор
	baseToken := tokenStream.Consume()

	// 2. Проверяем и потребляем LBRACKET
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenLBracket {
		return nil, newErrorWithPos(tokenStream, "expected '[' after identifier in index expression")
	}
	_ = tokenStream.Consume() // [

	// 3. Парсим индексное выражение
	index, err := h.parseExpression(tokenStream)
	if err != nil {
		return nil, newErrorWithPos(tokenStream, "failed to parse index expression: %v", err)
	}

	// 4. Проверяем и потребляем RBRACKET
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenRBracket {
		return nil, newErrorWithPos(tokenStream, "expected ']' after index expression")
	}
	_ = tokenStream.Consume() // ]

	// 5. Создаем узел индексного доступа
	result := &ast.IndexExpression{
		Object: &ast.Identifier{
			Name: baseToken.Value,
			Pos:  matchHandlerTokenToPosition(baseToken),
		},
		Index: index,
		Pos:   matchHandlerTokenToPosition(baseToken),
	}
	return result, nil
}

// parseLanguageIndexExpression парсит индексное выражение с языком вроде lua.data[i]
func (h *MatchHandler) parseLanguageIndexExpression(tokenStream stream.TokenStream) (ast.Expression, error) {

	// 1. Читаем язык
	languageToken := tokenStream.Consume()
	language := languageToken.Value

	// 2. Проверяем и потребляем DOT
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenDot {
		return nil, newErrorWithPos(tokenStream, "expected DOT after language '%s'", language)
	}
	tokenStream.Consume() // Consuming dot

	// 3. Читаем имя переменной
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenIdentifier {
		return nil, newErrorWithPos(tokenStream, "expected variable name after DOT")
	}
	variableToken := tokenStream.Consume()
	variableName := variableToken.Value

	// 4. Проверяем и потребляем LBRACKET
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenLBracket {
		return nil, newErrorWithPos(tokenStream, "expected '[' after variable name in language index expression")
	}
	_ = tokenStream.Consume() // [

	// 5. Парсим индексное выражение
	index, err := h.parseExpression(tokenStream)
	if err != nil {
		return nil, newErrorWithPos(tokenStream, "failed to parse index expression: %v", err)
	}

	// 6. Проверяем и потребляем RBRACKET
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenRBracket {
		return nil, newErrorWithPos(tokenStream, "expected ']' after index expression")
	}
	_ = tokenStream.Consume() // ]

	// 7. Создаем узел индексного доступа
	result := &ast.IndexExpression{
		Object: &ast.Identifier{
			Name:      variableName,
			Language:  language,
			Qualified: true,
			Pos:       matchHandlerTokenToPosition(languageToken),
		},
		Index: index,
		Pos:   matchHandlerTokenToPosition(languageToken),
	}
	return result, nil
}

// parseMatchArms парсит ветки сопоставления
func (h *MatchHandler) parseMatchArms(tokenStream stream.TokenStream) ([]ast.MatchArm, error) {
	arms := make([]ast.MatchArm, 0)

	for {
		// Пропускаем newline токены между ветками
		for tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenNewline {
			tokenStream.Consume() // newline
		}

		if !tokenStream.HasMore() {
			break
		}

		// Если видим '}', значит ветки закончились
		if tokenStream.Current().Type == lexer.TokenRBrace {
			break
		}

		// Парсим паттерн
		pattern, err := h.parsePattern(tokenStream)
		if err != nil {
			return nil, err
		}

		// Потребляем '->'
		if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenArrow {
			return nil, newErrorWithPos(tokenStream, "expected '->' after pattern")
		}
		arrowToken := tokenStream.Consume()

		// Парсим statement
		statement, err := h.parseStatement(tokenStream)
		if err != nil {
			return nil, err
		}

		// Создаем MatchArm
		arm := ast.MatchArm{
			Pattern:    pattern,
			ArrowToken: arrowToken,
			Statement:  statement,
		}
		arms = append(arms, arm)

		// Пропускаем newline токены после ветки
		for tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenNewline {
			tokenStream.Consume() // newline
		}

		if !tokenStream.HasMore() {
			break
		}

		// Проверяем запятую или конец
		if tokenStream.Current().Type == lexer.TokenComma {
			tokenStream.Consume() // ,
			// После запятой должна быть следующая ветка
			if !tokenStream.HasMore() {
				return nil, fmt.Errorf("expected pattern after comma")
			}
		} else if tokenStream.Current().Type == lexer.TokenRBrace {
			break // Конец match statement
		}
	}

	return arms, nil
}

// parsePattern парсит паттерн
func (h *MatchHandler) parsePattern(tokenStream stream.TokenStream) (ast.Pattern, error) {
	currentToken := tokenStream.Current()

	// Пропускаем newline токены
	for currentToken.Type == lexer.TokenNewline {
		tokenStream.Consume() // newline
		if !tokenStream.HasMore() {
			return nil, newErrorWithPos(tokenStream, "unexpected EOF after newline")
		}
		currentToken = tokenStream.Current()
	}

	switch currentToken.Type {
	case lexer.TokenString:
		return h.parseLiteralPattern(tokenStream)
	case lexer.TokenNumber:
		return h.parseLiteralPattern(tokenStream)
	case lexer.TokenLBracket:
		return h.parseArrayPattern(tokenStream)
	case lexer.TokenLBrace:
		return h.parseObjectPattern(tokenStream)
	case lexer.TokenDoubleLeftAngle:
		// Битстринг как паттерн
		return h.parseBitstringPattern(tokenStream)
	case lexer.TokenIdentifier:
		// Проверяем на wildcard или переменную
		if currentToken.Value == "_" {
			return h.parseWildcardPattern(tokenStream)
		}
		return h.parseVariablePattern(tokenStream)
	case lexer.TokenUnderscore:
		// Обработка токена underscore как wildcard
		return h.parseWildcardPattern(tokenStream)
	default:
		return nil, newErrorWithTokenPos(currentToken, "unsupported pattern type: %s", currentToken.Type)
	}
}

// parseLiteralPattern парсит литеральный паттерн
func (h *MatchHandler) parseLiteralPattern(tokenStream stream.TokenStream) (ast.Pattern, error) {
	token := tokenStream.Consume()
	var value interface{}

	switch token.Type {
	case lexer.TokenString:
		value = token.Value
	case lexer.TokenNumber:
		// Конвертация в число
		numValue, err := parseNumber(token.Value)
		if err != nil {
			return nil, newErrorWithTokenPos(token, "invalid number format: %s", token.Value)
		}
		// For literal patterns, we need the actual value, not the NumberLiteral
		if numInt, ok := numValue.(*big.Int); ok {
			value = numInt
		} else {
			value = numValue
		}
	default:
		return nil, newErrorWithTokenPos(token, "unsupported literal type: %s", token.Type)
	}

	return &ast.LiteralPattern{
		Value: value,
		Pos:   matchHandlerTokenToPosition(token),
	}, nil
}

// parseArrayPattern парсит массивный паттерн
func (h *MatchHandler) parseArrayPattern(tokenStream stream.TokenStream) (ast.Pattern, error) {
	lBracketToken := tokenStream.Consume() // [

	elements := make([]ast.Pattern, 0)
	hasRest := false

	for tokenStream.HasMore() && tokenStream.Current().Type != lexer.TokenRBracket {
		if tokenStream.Current().Type == lexer.TokenRest {
			// ...rest паттерн
			tokenStream.Consume() // ...

			// После ... должна идти переменная
			if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenIdentifier {
				return nil, newErrorWithPos(tokenStream, "expected variable name after '...'")
			}

			// Парсим переменную после ...
			restVar, err := h.parseVariablePattern(tokenStream)
			if err != nil {
				return nil, err
			}

			elements = append(elements, restVar)
			hasRest = true
			break
		}

		pattern, err := h.parsePattern(tokenStream)
		if err != nil {
			return nil, err
		}
		elements = append(elements, pattern)

		// Проверяем запятую
		if tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenComma {
			tokenStream.Consume() // ,
		}
	}

	// Потребляем ]
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenRBracket {
		return nil, newErrorWithPos(tokenStream, "expected ']' to close array pattern")
	}
	tokenStream.Consume() // ]

	return &ast.ArrayPattern{
		Elements: elements,
		Rest:     hasRest,
		Pos:      matchHandlerTokenToPosition(lBracketToken),
	}, nil
}

// parseObjectPattern парсит объектный паттерн
func (h *MatchHandler) parseObjectPattern(tokenStream stream.TokenStream) (ast.Pattern, error) {
	lBraceToken := tokenStream.Consume() // {

	properties := make(map[string]ast.Pattern)

	for tokenStream.HasMore() && tokenStream.Current().Type != lexer.TokenRBrace {
		// Парсим ключ (должен быть строкой или идентификатор)
		keyToken := tokenStream.Current()
		var key string

		switch keyToken.Type {
		case lexer.TokenString:
			key = keyToken.Value
			tokenStream.Consume()
		case lexer.TokenIdentifier:
			key = keyToken.Value
			tokenStream.Consume()
		case lexer.TokenUnderscore:
			// Поддерживаем underscore как ключ объекта для wildcard паттернов
			key = "_"
			tokenStream.Consume()
		default:
			return nil, newErrorWithTokenPos(keyToken, "object pattern key must be string, identifier or underscore, got %s", keyToken.Type)
		}

		// Потребляем ':'
		if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenColon {
			return nil, newErrorWithPos(tokenStream, "expected ':' after object key")
		}
		tokenStream.Consume() // :

		// Парсим значение-паттерн
		valuePattern, err := h.parsePattern(tokenStream)
		if err != nil {
			return nil, err
		}

		properties[key] = valuePattern

		// Проверяем запятую
		if tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenComma {
			tokenStream.Consume() // ,
		}
	}

	// Потребляем }
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenRBrace {
		return nil, newErrorWithPos(tokenStream, "expected '}' to close object pattern")
	}
	tokenStream.Consume() // }

	return &ast.ObjectPattern{
		Properties: properties,
		Pos:        matchHandlerTokenToPosition(lBraceToken),
	}, nil
}

// parseVariablePattern парсит переменный паттерн
func (h *MatchHandler) parseVariablePattern(tokenStream stream.TokenStream) (ast.Pattern, error) {
	token := tokenStream.Consume()

	return &ast.VariablePattern{
		Name: token.Value,
		Pos:  matchHandlerTokenToPosition(token),
	}, nil
}

// parseWildcardPattern парсит wildcard паттерн
func (h *MatchHandler) parseWildcardPattern(tokenStream stream.TokenStream) (ast.Pattern, error) {
	token := tokenStream.Consume() // _

	return &ast.WildcardPattern{
		Pos: matchHandlerTokenToPosition(token),
	}, nil
}

// parseStatement парсит statement
func (h *MatchHandler) parseStatement(tokenStream stream.TokenStream) (ast.Statement, error) {
	currentToken := tokenStream.Current()

	// Поддерживаем блоковые операторы { ... }
	if currentToken.Type == lexer.TokenLBrace {
		return h.parseBlockStatement(tokenStream)
	}

	// Поддерживаем if операторы
	if currentToken.Type == lexer.TokenIf {
		return h.parseIfStatement(tokenStream)
	}

	// Поддерживаем вложенные match операторы
	if currentToken.Type == lexer.TokenMatch {
		return h.parseMatchStatement(tokenStream)
	}

	// Поддерживаем language calls, assignments и builtin функции
	if currentToken.Type == lexer.TokenIdentifier ||
		currentToken.Type == lexer.TokenLua ||
		currentToken.Type == lexer.TokenPython ||
		currentToken.Type == lexer.TokenPy ||
		currentToken.Type == lexer.TokenGo ||
		currentToken.Type == lexer.TokenNode ||
		currentToken.Type == lexer.TokenJS {

		// Проверяем, что идет после идентификатора
		if tokenStream.HasMore() {
			nextToken := tokenStream.Peek()

			// Если следующий токен '=' или ':=', это assignment
			if nextToken.Type == lexer.TokenAssign || nextToken.Type == lexer.TokenColonEquals {
				return h.parseAssignmentStatement(tokenStream)
			}

			// Если следующий токен '.', это language call
			if nextToken.Type == lexer.TokenDot {
				// Пробуем распарсить как language call
				expr, err := h.parseLanguageCall(tokenStream)
				if err != nil {
					return nil, err
				}

				// Преобразуем expression в statement (для простоты)
				if stmt, ok := expr.(ast.Statement); ok {
					return stmt, nil
				}

				return nil, newErrorWithPos(tokenStream, "expression cannot be used as statement")
			}

			// Если следующий токен '(', это может быть builtin функция
			if nextToken.Type == lexer.TokenLeftParen {
				// Пробуем распарсить как builtin функцию
				// Создаем временный контекст для парсера builtin функций
				ctx := &common.ParseContext{
					TokenStream: tokenStream,
					Depth:       0,
					MaxDepth:    100,
				}

				// Создаем конфигурацию для builtin function handler
				builtinConfig := config.ConstructHandlerConfig{
					ConstructType: common.ConstructFunction,
					Name:          "builtin-function-call",
					Priority:      120,
					Order:         1,
					IsEnabled:     true,
					IsFallback:    false,
					TokenPatterns: []config.TokenPattern{
						{TokenType: lexer.TokenIdentifier, Offset: 0},
					},
				}

				// Используем BuiltinFunctionHandler для парсинга
				builtinHandler := NewBuiltinFunctionHandlerWithVerbose(builtinConfig, h.verbose)
				if h.verbose {
					fmt.Printf("DEBUG: MatchHandler - trying to parse builtin function: %s\n", currentToken.Value)
				}
				result, err := builtinHandler.Handle(ctx)
				if h.verbose {
					fmt.Printf("DEBUG: MatchHandler - builtin handler result: %v, error: %v\n", result, err)
				}
				if err != nil {
					// Если это не builtin функция, возвращаем ошибку
					return nil, newErrorWithTokenPos(currentToken, "unexpected identifier '%s' - expected assignment, language call, or builtin function", currentToken.Value)
				}

				// BuiltinFunctionCall implements Statement interface
				if builtinCall, ok := result.(*ast.BuiltinFunctionCall); ok {
					return builtinCall, nil
				}

				fmt.Printf("DEBUG: expected BuiltinFunctionCall, got %T, value: %v\n", result, result)
				return nil, newErrorWithPos(tokenStream, "expected BuiltinFunctionCall, got %T", result)
			}
		}

		// Если это простой идентификатор без '=', '.', или '(', возвращаем ошибку
		return nil, newErrorWithTokenPos(currentToken, "unexpected identifier '%s' - expected assignment, language call, or builtin function", currentToken.Value)
	}

	return nil, newErrorWithTokenPos(currentToken, "unsupported statement type: %s", currentToken.Type)
}

// parseAssignmentStatement парсит assignment statement (например, norm = value)
func (h *MatchHandler) parseAssignmentStatement(tokenStream stream.TokenStream) (ast.Statement, error) {
	// Читаем имя переменной
	identifierToken := tokenStream.Consume()
	variableName := identifierToken.Value

	// Потребляем '=' или ':='
	if !tokenStream.HasMore() || (tokenStream.Current().Type != lexer.TokenAssign && tokenStream.Current().Type != lexer.TokenColonEquals) {
		return nil, newErrorWithPos(tokenStream, "expected '=' or ':=' after variable name")
	}
	assignToken := tokenStream.Consume()

	// Парсим выражение справа от '='
	valueExpr, err := h.parseExpression(tokenStream)
	if err != nil {
		return nil, newErrorWithPos(tokenStream, "failed to parse assignment value: %v", err)
	}

	// Создаем простой идентификатор (локальная переменная)
	variable := ast.NewIdentifier(identifierToken, variableName)

	// Создаем VariableAssignment
	assignment := &ast.VariableAssignment{
		Variable: variable,
		Assign:   assignToken,
		Value:    valueExpr,
	}

	return assignment, nil
}

// Config возвращает конфигурацию обработчика
func (h *MatchHandler) Config() common.HandlerConfig {
	return common.HandlerConfig{
		IsEnabled: h.config.IsEnabled,
		Priority:  h.config.Priority,
		Name:      h.config.Name,
	}
}

// Name возвращает имя обработчика
func (h *MatchHandler) Name() string {
	return h.config.Name
}

// parseArrayExpression парсит массив как выражение
func (h *MatchHandler) parseArrayExpression(tokenStream stream.TokenStream) (ast.Expression, error) {
	lBracketToken := tokenStream.Consume() // [

	elements := make([]ast.Expression, 0)

	for tokenStream.HasMore() && tokenStream.Current().Type != lexer.TokenRBracket {
		// Парсим элемент как простое выражение
		currentToken := tokenStream.Current()
		var element ast.Expression

		switch currentToken.Type {
		case lexer.TokenString:
			token := tokenStream.Consume()
			element = &ast.StringLiteral{Value: token.Value, Pos: matchHandlerTokenToPosition(token)}
		case lexer.TokenNumber:
			token := tokenStream.Consume()
			numValue, err := parseNumber(token.Value)
			if err != nil {
				return nil, newErrorWithTokenPos(token, "invalid number format: %s", token.Value)
			}
			element = createNumberLiteral(token, numValue)
		case lexer.TokenIdentifier:
			identifierToken := tokenStream.Consume()
			element = &ast.Identifier{
				Name: identifierToken.Value,
				Pos:  matchHandlerTokenToPosition(identifierToken),
			}
		default:
			return nil, newErrorWithTokenPos(currentToken, "unsupported array element type: %s", currentToken.Type)
		}

		elements = append(elements, element)

		// Проверяем запятую
		if tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenComma {
			tokenStream.Consume() // ,
		}
	}

	// Потребляем ]
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenRBracket {
		return nil, newErrorWithPos(tokenStream, "expected ']' to close array expression")
	}
	tokenStream.Consume() // ]

	return &ast.ArrayLiteral{
		Elements:     elements,
		LeftBracket:  lBracketToken,
		RightBracket: tokenStream.Current(), // This was already consumed
	}, nil
}

// parseObjectExpression парсит объект как выражение
func (h *MatchHandler) parseObjectExpression(tokenStream stream.TokenStream) (ast.Expression, error) {
	lBraceToken := tokenStream.Consume() // {

	properties := make([]ast.ObjectProperty, 0)

	for tokenStream.HasMore() && tokenStream.Current().Type != lexer.TokenRBrace {
		// Парсим ключ (должен быть строкой или идентификатор)
		keyToken := tokenStream.Current()
		var key ast.Expression

		switch keyToken.Type {
		case lexer.TokenString:
			token := tokenStream.Consume()
			key = &ast.StringLiteral{Value: token.Value, Pos: matchHandlerTokenToPosition(token)}
		case lexer.TokenIdentifier:
			identifierToken := tokenStream.Consume()
			key = &ast.Identifier{
				Name: identifierToken.Value,
				Pos:  matchHandlerTokenToPosition(identifierToken),
			}
		default:
			return nil, newErrorWithTokenPos(keyToken, "object expression key must be string or identifier, got %s", keyToken.Type)
		}

		// Потребляем ':'
		if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenColon {
			return nil, newErrorWithPos(tokenStream, "expected ':' after object key")
		}
		tokenStream.Consume() // :

		// Парсим значение-выражение
		currentToken := tokenStream.Current()
		var value ast.Expression

		switch currentToken.Type {
		case lexer.TokenString:
			token := tokenStream.Consume()
			value = &ast.StringLiteral{Value: token.Value, Pos: matchHandlerTokenToPosition(token)}
		case lexer.TokenNumber:
			token := tokenStream.Consume()
			numValue, err := parseNumber(token.Value)
			if err != nil {
				return nil, newErrorWithTokenPos(token, "invalid number format: %s", token.Value)
			}
			value = createNumberLiteral(token, numValue)
		case lexer.TokenIdentifier:
			identifierToken := tokenStream.Consume()
			value = &ast.Identifier{
				Name: identifierToken.Value,
				Pos:  matchHandlerTokenToPosition(identifierToken),
			}
		default:
			return nil, newErrorWithTokenPos(currentToken, "unsupported object value type: %s", currentToken.Type)
		}

		properties = append(properties, ast.ObjectProperty{
			Key:   key,
			Value: value,
		})

		// Проверяем запятую
		if tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenComma {
			tokenStream.Consume() // ,
		}
	}

	// Потребляем }
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenRBrace {
		return nil, newErrorWithPos(tokenStream, "expected '}' to close object expression")
	}
	rBraceToken := tokenStream.Consume() // }

	return &ast.ObjectLiteral{
		Properties: properties,
		LeftBrace:  lBraceToken,
		RightBrace: rBraceToken,
	}, nil
}

// parseBitstringPattern парсит битстринг как паттерн
func (h *MatchHandler) parseBitstringPattern(tokenStream stream.TokenStream) (ast.Pattern, error) {
	doubleLeftAngleToken := tokenStream.Consume() // <<

	segments := make([]ast.BitstringSegment, 0)

	for tokenStream.HasMore() && tokenStream.Current().Type != lexer.TokenDoubleRightAngle {
		currentToken := tokenStream.Current()

		// Пропускаем NEWLINE токены перед элементом
		for currentToken.Type == lexer.TokenNewline {
			tokenStream.Consume() // newline
			if !tokenStream.HasMore() {
				return nil, newErrorWithPos(tokenStream, "unexpected EOF after newline")
			}
			currentToken = tokenStream.Current()
		}

		var segment ast.BitstringSegment

		switch currentToken.Type {
		case lexer.TokenString:
			// Строковый элемент в битстринге
			token := tokenStream.Consume()
			segment = ast.BitstringSegment{
				Value: &ast.StringLiteral{Value: token.Value, Pos: matchHandlerTokenToPosition(token)},
			}

			// Проверяем наличие размера через двоеточие (:Size)
			if tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenColon {
				tokenStream.Consume() // :

				if !tokenStream.HasMore() {
					return nil, newErrorWithPos(tokenStream, "expected size after colon")
				}

				// Парсим размер как выражение (может быть число, переменная или сложное выражение)
				sizeExpr, err := h.parseSizeExpression(tokenStream)
				if err != nil {
					return nil, newErrorWithPos(tokenStream, "failed to parse size expression: %v", err)
				}

				segment.Size = sizeExpr

				// Определяем, является ли размер динамическим
				isDynamic := h.isDynamicSizeExpression(sizeExpr)
				segment.IsDynamicSize = isDynamic

				if isDynamic {
					// Создаем SizeExpression для динамического размера
					sizeExpression := ast.NewSizeExpression()
					sizeExpression.Pos = sizeExpr.Position()

					if strLit, ok := sizeExpr.(*ast.StringLiteral); ok {
						// Если это строковый литерал, используем его значение как выражение
						sizeExpression.ExprType = "expression"
						sizeExpression.Variable = strLit.Value // Используем значение строки как выражение
						sizeExpression.Literal = sizeExpr
					} else {
						// Сложное выражение
						sizeExpression.ExprType = "expression"
						sizeExpression.Expression = sizeExpr
					}

					segment.SizeExpression = sizeExpression
				}
				segment.ColonToken = lexer.Token{Type: lexer.TokenColon, Value: ":"}
			}

			// Проверяем наличие спецификаторов через слэш (/Specifiers)
			if tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenSlash {
				tokenStream.Consume() // /
				segment.SlashToken = tokenStream.Current()

				// Парсим спецификаторы
				if err := h.parseBitstringSpecifiers(tokenStream, &segment); err != nil {
					return nil, err
				}
			}
		case lexer.TokenNumber:
			// Числовой элемент в битстринге
			token := tokenStream.Consume()
			numValue, err := parseNumber(token.Value)
			if err != nil {
				return nil, newErrorWithTokenPos(token, "invalid number format: %s", token.Value)
			}

			segment = ast.BitstringSegment{
				Value: createNumberLiteral(token, numValue),
			}

			// Проверяем наличие размера через двоеточие (:Size)
			if tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenColon {
				tokenStream.Consume() // :

				if !tokenStream.HasMore() {
					return nil, newErrorWithPos(tokenStream, "expected size after colon")
				}

				// Парсим размер как выражение (может быть число, переменная или сложное выражение)
				sizeExpr, err := h.parseSizeExpression(tokenStream)
				if err != nil {
					return nil, newErrorWithPos(tokenStream, "failed to parse size expression: %v", err)
				}

				segment.Size = sizeExpr

				// Определяем, является ли размер динамическим
				isDynamic := h.isDynamicSizeExpression(sizeExpr)
				segment.IsDynamicSize = isDynamic

				if isDynamic {
					// Создаем SizeExpression для динамического размера
					sizeExpression := ast.NewSizeExpression()
					sizeExpression.Pos = sizeExpr.Position()

					if ident, ok := sizeExpr.(*ast.Identifier); ok {
						// Простая переменная
						sizeExpression.ExprType = "variable"
						sizeExpression.Variable = ident.Name
						sizeExpression.Literal = sizeExpr
					} else if strLit, ok := sizeExpr.(*ast.StringLiteral); ok {
						// Строковое выражение из скобок (например, "total-6")
						sizeExpression.ExprType = "expression"
						sizeExpression.Variable = strLit.Value // Используем значение строки как выражение
						sizeExpression.Literal = sizeExpr
					} else {
						// Сложное выражение
						sizeExpression.ExprType = "expression"
						sizeExpression.Expression = sizeExpr
					}

					segment.SizeExpression = sizeExpression
				}
				segment.ColonToken = lexer.Token{Type: lexer.TokenColon, Value: ":"}
			}

			// Проверяем наличие спецификаторов через слэш (/Specifiers)
			if tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenSlash {
				tokenStream.Consume() // /
				segment.SlashToken = tokenStream.Current()

				// Парсим спецификаторы
				if err := h.parseBitstringSpecifiers(tokenStream, &segment); err != nil {
					return nil, err
				}
			}
		case lexer.TokenIdentifier, lexer.TokenUnderscore, lexer.TokenLua, lexer.TokenPython, lexer.TokenPy, lexer.TokenJS, lexer.TokenNode, lexer.TokenGo:
			// Переменная в битстринге (обычная или языковая)
			currentToken := tokenStream.Current()

			if currentToken.Type == lexer.TokenUnderscore {
				// Wildcard pattern '_'
				token := tokenStream.Consume()
				segment = ast.BitstringSegment{
					Value: &ast.Identifier{Name: token.Value, Pos: matchHandlerTokenToPosition(token)},
				}
			} else if currentToken.Type == lexer.TokenIdentifier {
				// Обычная переменная
				token := tokenStream.Consume()
				segment = ast.BitstringSegment{
					Value: &ast.Identifier{Name: token.Value, Pos: matchHandlerTokenToPosition(token)},
				}
			} else {
				// Языковая переменная (например, lua.variable)
				languageToken := tokenStream.Consume()

				// Ожидаем точку
				if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenDot {
					return nil, newErrorWithPos(tokenStream, "expected '.' after language token %s", languageToken.Value)
				}
				tokenStream.Consume() // .

				// Ожидаем идентификатор переменной
				if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenIdentifier {
					return nil, newErrorWithPos(tokenStream, "expected variable name after '%s.'", languageToken.Value)
				}
				variableToken := tokenStream.Consume()

				// Создаем квалифицированный идентификатор
				qualifiedId := &ast.Identifier{
					Name:      fmt.Sprintf("%s.%s", languageToken.Value, variableToken.Value),
					Qualified: true,
					Pos:       matchHandlerTokenToPosition(languageToken),
				}

				segment = ast.BitstringSegment{
					Value: qualifiedId,
				}
			}

			// Проверяем наличие размера через двоеточие (:Size)
			if tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenColon {
				tokenStream.Consume() // :

				if !tokenStream.HasMore() {
					return nil, newErrorWithPos(tokenStream, "expected size after colon")
				}

				// Парсим размер как выражение (может быть число, переменная или сложное выражение)
				sizeExpr, err := h.parseSizeExpression(tokenStream)
				if err != nil {
					return nil, newErrorWithPos(tokenStream, "failed to parse size expression: %v", err)
				}

				segment.Size = sizeExpr

				// Определяем, является ли размер динамическим
				isDynamic := h.isDynamicSizeExpression(sizeExpr)
				segment.IsDynamicSize = isDynamic

				if isDynamic {
					// Создаем SizeExpression для динамического размера
					sizeExpression := ast.NewSizeExpression()
					sizeExpression.Pos = sizeExpr.Position()

					if ident, ok := sizeExpr.(*ast.Identifier); ok {
						// Простая переменная
						sizeExpression.ExprType = "variable"
						sizeExpression.Variable = ident.Name
						sizeExpression.Literal = sizeExpr
					} else if strLit, ok := sizeExpr.(*ast.StringLiteral); ok {
						// Строковое выражение из скобок (например, "total-6")
						sizeExpression.ExprType = "expression"
						sizeExpression.Variable = strLit.Value // Используем значение строки как выражение
						sizeExpression.Literal = sizeExpr
					} else {
						// Сложное выражение
						sizeExpression.ExprType = "expression"
						sizeExpression.Expression = sizeExpr
					}

					segment.SizeExpression = sizeExpression
				}
				segment.ColonToken = lexer.Token{Type: lexer.TokenColon, Value: ":"}
			}

			// Проверяем наличие спецификаторов через слэш (/Specifiers)
			if tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenSlash {
				tokenStream.Consume() // /
				segment.SlashToken = tokenStream.Current()

				// Парсим спецификаторы
				if err := h.parseBitstringSpecifiers(tokenStream, &segment); err != nil {
					return nil, err
				}
			}
		default:
			return nil, newErrorWithTokenPos(currentToken, "unexpected token in bitstring pattern: %s", currentToken.Type)
		}

		segments = append(segments, segment)

		// Проверяем запятую
		if tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenComma {
			tokenStream.Consume() // ,
		}

		// Пропускаем NEWLINE токены после элемента или запятой
		for tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenNewline {
			tokenStream.Consume() // newline
		}
	}

	// Пропускаем финальные NEWLINE токены перед >>
	for tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenNewline {
		tokenStream.Consume() // newline
	}

	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenDoubleRightAngle {
		return nil, newErrorWithPos(tokenStream, "expected '>>' to close bitstring pattern")
	}

	doubleRightAngleToken := tokenStream.Consume() // >>

	return &ast.BitstringPattern{
		Elements:   segments,
		LeftAngle:  doubleLeftAngleToken,
		RightAngle: doubleRightAngleToken,
		Pos:        matchHandlerTokenToPosition(doubleLeftAngleToken),
	}, nil
}

// parseSizeExpression парсит выражение размера (число, переменная, или сложное выражение с арифметикой)
func (h *MatchHandler) parseSizeExpression(tokenStream stream.TokenStream) (ast.Expression, error) {
	// Устанавливаем флаг контекста size expression в лексере
	if lexer := tokenStream.GetLexer(); lexer != nil {
		lexer.SetInSizeExpression(true)
		defer lexer.SetInSizeExpression(false)
	}

	// Используем ограниченный парсер выражений для size выражений
	return h.parseSizeExpressionLimited(tokenStream)
}

// parseSizeExpressionLimited парсит арифметическое выражение размера с ограничениями
func (h *MatchHandler) parseSizeExpressionLimited(tokenStream stream.TokenStream) (ast.Expression, error) {
	return h.parseSizeExpressionLimitedWithDepth(tokenStream, 0)
}

// parseSizeExpressionLimitedWithDepth парсит арифметическое выражение размера с отслеживанием глубины скобок
func (h *MatchHandler) parseSizeExpressionLimitedWithDepth(tokenStream stream.TokenStream, parenDepth int) (ast.Expression, error) {
	// Парсим левую часть выражения
	left, err := h.parseSizePrimaryOrComplexExpressionWithDepth(tokenStream, parenDepth)
	if err != nil {
		return nil, err
	}

	// Проверяем, есть ли бинарный оператор после левой части
	for tokenStream.HasMore() {
		nextToken := tokenStream.Current()

		// Проверяем, является ли токен терминатором size выражения
		// / является терминатором только когда мы не в скобках
		isTerminator := h.isSizeExpressionTerminator(nextToken.Type)
		isSlashInParens := nextToken.Type == lexer.TokenSlash && parenDepth > 0
		shouldBreak := isTerminator && !isSlashInParens

		if shouldBreak {
			break
		}

		if h.isBinaryOperator(nextToken.Type) {
			// Это бинарное выражение
			left, err = h.parseBinaryExpressionWithLeftLimited(tokenStream, left, parenDepth)
			if err != nil {
				return nil, err
			}
		} else {
			// Неожиданный токен
			break
		}
	}

	return left, nil
}

// parseSizePrimaryOrComplexExpressionWithDepth парсит первичные выражения для size context с поддержкой скобок
func (h *MatchHandler) parseSizePrimaryOrComplexExpressionWithDepth(tokenStream stream.TokenStream, parenDepth int) (ast.Expression, error) {
	currentToken := tokenStream.Current()

	switch currentToken.Type {
	case lexer.TokenLeftParen:
		// Выражение в скобках - используем size-specific парсер
		return h.parseSizeParenthesizedExpression(tokenStream, parenDepth)
	case lexer.TokenString:
		// Строковый литерал
		token := tokenStream.Consume()
		return &ast.StringLiteral{
			Value: token.Value,
			Pos:   matchHandlerTokenToPosition(token),
		}, nil
	case lexer.TokenNumber:
		// Числовой литерал
		token := tokenStream.Consume()
		numValue, err := parseNumber(token.Value)
		if err != nil {
			return nil, newErrorWithTokenPos(token, "invalid number format: %s", token.Value)
		}
		return createNumberLiteral(token, numValue), nil
	case lexer.TokenIdentifier:
		// Переменная
		token := tokenStream.Consume()
		return &ast.Identifier{
			Name: token.Value,
			Pos:  matchHandlerTokenToPosition(token),
		}, nil
	default:
		return nil, newErrorWithTokenPos(currentToken, "unsupported expression type in size context: %s", currentToken.Type)
	}
}

// isSizeExpressionTerminator проверяет, является ли токен терминатором size выражения
func (h *MatchHandler) isSizeExpressionTerminator(tokenType lexer.TokenType) bool {
	switch tokenType {
	case lexer.TokenSlash, // Начало спецификаторов /binary (когда не в скобках)
		lexer.TokenComma,            // Разделитель сегментов
		lexer.TokenDoubleRightAngle, // Конец битстринга >>
		lexer.TokenNewline,          // Новая строка
		lexer.TokenRBrace,           // Конец match
		lexer.TokenArrow:            // -> в match
		return true
	default:
		return false
	}
}

// getOperatorPriority возвращает приоритет оператора для size expressions
func (h *MatchHandler) getOperatorPriority(op string) int {
	switch op {
	case "*", "/", "%":
		return 3
	case "+", "-":
		return 2
	default:
		return 1
	}
}

// parseBinaryExpressionWithLeftLimited парсит бинарное выражение с ограничениями и учетом приоритета
func (h *MatchHandler) parseBinaryExpressionWithLeftLimited(tokenStream stream.TokenStream, left ast.Expression, parenDepth int) (ast.Expression, error) {
	// Потребляем оператор
	operatorToken := tokenStream.Consume()
	operator := operatorToken.Value

	// Парсим правую часть с ограничениями
	right, err := h.parseSizePrimaryOrComplexExpressionWithDepth(tokenStream, parenDepth)
	if err != nil {
		return nil, newErrorWithPos(tokenStream, "failed to parse right operand: %v", err)
	}

	// Создаем бинарное выражение
	binaryExpr := ast.NewBinaryExpression(left, operator, right, matchHandlerTokenToPosition(operatorToken))

	// Проверяем наличие дополнительных операторов с учетом приоритета
	for tokenStream.HasMore() {
		nextToken := tokenStream.Current()

		// Проверяем, является ли токен терминатором size выражения
		// / является терминатором только когда мы не в скобках
		if h.isSizeExpressionTerminator(nextToken.Type) && !(nextToken.Type == lexer.TokenSlash && parenDepth > 0) {
			break
		}

		if h.isBinaryOperator(nextToken.Type) {
			nextOperator := nextToken.Value
			nextPriority := h.getOperatorPriority(nextOperator)
			currentPriority := h.getOperatorPriority(operator)

			// Если следующий оператор имеет более высокий приоритет, парсим его рекурсивно
			if nextPriority > currentPriority {
				// Парсим следующее выражение с более высоким приоритетом
				newRight, err := h.parseBinaryExpressionWithLeftLimited(tokenStream, right, parenDepth)
				if err != nil {
					return nil, err
				}
				// Обновляем правую часть текущего выражения
				binaryExpr.Right = newRight
			} else {
				// Следующий оператор имеет такой же или более низкий приоритет
				// Создаем новое выражение с текущим как левый операнд
				newResult, err := h.parseBinaryExpressionWithLeftLimited(tokenStream, binaryExpr, parenDepth)
				if err != nil {
					return nil, err
				}
				binaryExpr = newResult.(*ast.BinaryExpression)
			}
		} else {
			break
		}
	}

	return binaryExpr, nil
}

// isDynamicSizeExpression проверяет, является ли выражение размера динамическим
func (h *MatchHandler) isDynamicSizeExpression(expr ast.Expression) bool {
	switch expr.(type) {
	case *ast.Identifier:
		// Ссылка на переменную - динамическая
		return true
	case *ast.NumberLiteral:
		// Литеральное число - статическое
		return false
	case *ast.StringLiteral:
		// Строковое выражение (из скобок) - динамическое
		return true
	case *ast.BinaryExpression:
		// Бинарное выражение - динамическое
		return true
	default:
		// Другие типы выражений считаем динамическими для безопасности
		return true
	}
}
func (h *MatchHandler) parseBitstringSpecifiers(tokenStream stream.TokenStream, segment *ast.BitstringSegment) error {
	// Парсим спецификаторы
	specifiers := make([]string, 0)

	for tokenStream.HasMore() && tokenStream.Current().Type != lexer.TokenComma && tokenStream.Current().Type != lexer.TokenDoubleRightAngle {
		specToken := tokenStream.Current()
		if specToken.Type == lexer.TokenIdentifier {
			specValue := specToken.Value
			tokenStream.Consume()

			// Проверяем на дефис для составных спецификаторов (big-endian, little-endian, etc.)
			for tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenMinus {
				tokenStream.Consume() // потребляем '-'

				if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenIdentifier {
					return newErrorWithPos(tokenStream, "expected identifier after '-' in specifier")
				}

				nextIdent := tokenStream.Consume()
				specValue += "-" + nextIdent.Value
			}

			// Проверяем, есть ли у спецификатора параметр через двоеточие (например, unit:1)
			if tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenColon {
				tokenStream.Consume() // потребляем ':'

				if !tokenStream.HasMore() {
					return newErrorWithPos(tokenStream, "unexpected EOF after colon in specifier")
				}

				// Парсим значение параметра спецификатора
				paramToken := tokenStream.Consume()
				if paramToken.Type != lexer.TokenNumber && paramToken.Type != lexer.TokenIdentifier {
					return newErrorWithTokenPos(paramToken, "expected number or identifier as specifier parameter, got %s", paramToken.Type)
				}

				// Комбинируем спецификатор и его параметр
				specValue = specValue + ":" + paramToken.Value
			}

			specifiers = append(specifiers, specValue)

			// Проверяем разделитель спецификаторов
			if tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenSemicolon {
				tokenStream.Consume() // ;
			}
		} else {
			break
		}
	}

	segment.Specifiers = specifiers
	return nil
}

// matchHandlerTokenToPosition конвертирует токен в позицию AST
// parseBlockStatement парсит блоковый оператор { ... }
func (h *MatchHandler) parseBlockStatement(tokenStream stream.TokenStream) (ast.Statement, error) {
	// Потребляем открывающую фигурную скобку
	lBraceToken := tokenStream.Consume() // {

	// Создаем пустой блок statement
	blockStmt := &ast.BlockStatement{
		LBraceToken: lBraceToken,
		Pos:         matchHandlerTokenToPosition(lBraceToken),
	}

	// Парсим statements внутри блока
	for tokenStream.HasMore() && tokenStream.Current().Type != lexer.TokenRBrace {
		// Пропускаем newline токены
		for tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenNewline {
			tokenStream.Consume() // newline
		}

		if !tokenStream.HasMore() {
			return nil, newErrorWithPos(tokenStream, "unexpected EOF in block statement")
		}

		// Если видим '}', значит блок закончился
		if tokenStream.Current().Type == lexer.TokenRBrace {
			break
		}

		// Парсим statement
		stmt, err := h.parseStatement(tokenStream)
		if err != nil {
			return nil, err
		}

		blockStmt.Statements = append(blockStmt.Statements, stmt)

		// Пропускаем newline токены после statement
		for tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenNewline {
			tokenStream.Consume() // newline
		}
	}

	// Потребляем закрывающую фигурную скобку
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenRBrace {
		return nil, newErrorWithPos(tokenStream, "expected '}' at end of block statement")
	}
	rBraceToken := tokenStream.Consume() // }
	blockStmt.RBraceToken = rBraceToken

	return blockStmt, nil
}

// parseIfStatement парсит if оператор
func (h *MatchHandler) parseIfStatement(tokenStream stream.TokenStream) (ast.Statement, error) {
	// Потребляем токен 'if'
	ifToken := tokenStream.Consume() // if

	// Потребляем '('
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenLeftParen {
		return nil, newErrorWithPos(tokenStream, "expected '(' after 'if'")
	}
	lParenToken := tokenStream.Consume() // (

	// Парсим условие с помощью UnifiedExpressionParser для правильных приоритетов
	ctx := &common.ParseContext{
		TokenStream: tokenStream,
		Depth:       0,
		MaxDepth:    100,
	}
	expressionParser := NewUnifiedExpressionParser(h.verbose)
	condition, err := expressionParser.ParseExpression(ctx)
	if err != nil {
		return nil, newErrorWithPos(tokenStream, "failed to parse if condition: %v", err)
	}

	// Потребляем ')'
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenRightParen {
		return nil, newErrorWithPos(tokenStream, "expected ')' after if condition, got %s", tokenStream.Current().Type)
	}
	rParenToken := tokenStream.Consume() // )

	// Парсим тело if (должно быть блоком)
	consequent, err := h.parseBlockStatement(tokenStream)
	if err != nil {
		return nil, newErrorWithPos(tokenStream, "failed to parse if body: %v", err)
	}

	// Преобразуем consequent к *ast.BlockStatement
	consequentBlock, ok := consequent.(*ast.BlockStatement)
	if !ok {
		return nil, newErrorWithPos(tokenStream, "if body must be a block statement")
	}

	// Создаем IfStatement
	ifStmt := &ast.IfStatement{
		Condition:   condition,
		Consequent:  consequentBlock,
		IfToken:     ifToken,
		LParenToken: lParenToken,
		RParenToken: rParenToken,
		Pos:         matchHandlerTokenToPosition(ifToken),
	}

	// Проверяем наличие else
	if tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenElse {
		elseToken := tokenStream.Consume() // else

		// Парсим тело else (должно быть блоком)
		alternate, err := h.parseBlockStatement(tokenStream)
		if err != nil {
			return nil, newErrorWithPos(tokenStream, "failed to parse else body: %v", err)
		}

		// Преобразуем alternate к *ast.BlockStatement
		alternateBlock, ok := alternate.(*ast.BlockStatement)
		if !ok {
			return nil, newErrorWithPos(tokenStream, "else body must be a block statement")
		}

		ifStmt.ElseToken = elseToken
		ifStmt.Alternate = alternateBlock
	}

	return ifStmt, nil
}

// parseMatchStatement парсит вложенный match оператор
func (h *MatchHandler) parseMatchStatement(tokenStream stream.TokenStream) (ast.Statement, error) {
	// 1. Потребляем 'match'
	matchToken := tokenStream.Consume()

	// 2. Парсим выражение для сопоставления
	expression, err := h.parseExpression(tokenStream)
	if err != nil {
		return nil, err
	}

	// 3. Потребляем '{'
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenLBrace {
		return nil, newErrorWithPos(tokenStream, "expected '{' after match expression")
	}
	lBraceToken := tokenStream.Consume() // {

	// 4. Парсим ветки сопоставления
	arms, err := h.parseMatchArms(tokenStream)
	if err != nil {
		return nil, err
	}

	// 5. Потребляем '}'
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenRBrace {
		return nil, newErrorWithPos(tokenStream, "expected '}' at end of match statement")
	}
	rBraceToken := tokenStream.Consume() // }

	// 6. Создаем MatchStatement
	startPos := matchHandlerTokenToPosition(matchToken)
	matchStmt := &ast.MatchStatement{
		Expression:  expression,
		Arms:        arms,
		MatchToken:  matchToken,
		LBraceToken: lBraceToken,
		RBraceToken: rBraceToken,
		Pos:         startPos,
	}

	return matchStmt, nil
}

// SimpleRecursionGuard - простая реализация защиты от рекурсии
type SimpleRecursionGuard struct {
	maxDepth     int
	currentDepth int
}

func (rg *SimpleRecursionGuard) Enter() error {
	if rg.currentDepth >= rg.maxDepth {
		return newErrorWithPos(nil, "maximum recursion depth exceeded: %d", rg.maxDepth)
	}
	rg.currentDepth++
	return nil
}

func (rg *SimpleRecursionGuard) Exit() {
	if rg.currentDepth > 0 {
		rg.currentDepth--
	}
}

func (rg *SimpleRecursionGuard) CurrentDepth() int {
	return rg.currentDepth
}

func (rg *SimpleRecursionGuard) MaxDepth() int {
	return rg.maxDepth
}

func matchHandlerTokenToPosition(token lexer.Token) ast.Position {
	return ast.Position{
		Line:   token.Line,
		Column: token.Column,
		Offset: token.Position,
	}
}
