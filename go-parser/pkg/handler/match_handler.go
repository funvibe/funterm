package handler

import (
	"fmt"
	"strconv"
	"strings"

	"go-parser/pkg/ast"
	"go-parser/pkg/common"
	"go-parser/pkg/config"
	"go-parser/pkg/lexer"
	"go-parser/pkg/stream"
)

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
		return nil, fmt.Errorf("expected '{' after match expression")
	}
	lBraceToken := tokenStream.Consume() // {

	// 4. Парсим ветки сопоставления
	arms, err := h.parseMatchArms(tokenStream)
	if err != nil {
		return nil, err
	}

	// 5. Потребляем '}'
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenRBrace {
		return nil, fmt.Errorf("expected '}' at end of match statement")
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
	left, err := h.parsePrimaryOrComplexExpression(tokenStream)
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
		lexer.TokenAmpersand, lexer.TokenConcat:
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
		return nil, fmt.Errorf("failed to parse right operand: %v", err)
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
	trueBranch, err := h.parseExpression(tokenStream)
	if err != nil {
		return nil, fmt.Errorf("failed to parse true branch of ternary expression: %v", err)
	}
	if h.verbose {
		fmt.Printf("DEBUG: parseTernaryExpression - parsed true branch, current token: %v\n", tokenStream.Current())
	}

	// Проверяем и потребляем ':'
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenColon {
		return nil, fmt.Errorf("expected ':' after true branch of ternary expression")
	}
	colonToken := tokenStream.Consume()
	if h.verbose {
		fmt.Printf("DEBUG: parseTernaryExpression - consumed ':', current token: %v\n", tokenStream.Current())
	}

	// Парсим false-branch - здесь может быть еще один тернарный оператор
	// Для вложенных тернарных операторов нам нужно быть осторожными с парсингом
	falseBranch, err := h.parseTernaryFalseBranch(tokenStream)
	if err != nil {
		return nil, fmt.Errorf("failed to parse false branch of ternary expression: %v", err)
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
	left, err := h.parsePrimaryOrComplexExpression(tokenStream)
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
	currentToken := tokenStream.Current()

	switch currentToken.Type {
	case lexer.TokenLeftParen:
		// Выражение в скобках
		expr, err := h.parseParenthesizedExpression(tokenStream)
		if err != nil {
			return nil, err
		}

		// После скобочного выражения может быть бинарный оператор
		if tokenStream.HasMore() && h.isBinaryOperator(tokenStream.Current().Type) {
			if h.verbose {
				fmt.Printf("DEBUG: parsePrimaryOrComplexExpression - parsing binary expression after parentheses\n")
			}
			return h.parseBinaryExpressionWithLeft(tokenStream, expr)
		}

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
		var numValue float64
		var err error

		// Проверяем, является ли число шестнадцатеричным
		if len(token.Value) > 2 && (token.Value[0:2] == "0x" || token.Value[0:2] == "0X") {
			// Парсим шестнадцатеричное число
			var hexInt int64
			_, err = fmt.Sscanf(token.Value, "0x%x", &hexInt)
			if err != nil {
				return nil, fmt.Errorf("invalid hexadecimal number format: %s", token.Value)
			}
			numValue = float64(hexInt)
		} else {
			// Парсим десятичное число
			numValue, err = strconv.ParseFloat(token.Value, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid number: %s", token.Value)
			}
		}

		return &ast.NumberLiteral{
			Value: numValue,
			Pos:   matchHandlerTokenToPosition(token),
		}, nil
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
		return nil, fmt.Errorf("unsupported expression type: %s", currentToken.Type)
	}
}

// parseParenthesizedExpression парсит выражение в скобках
func (h *MatchHandler) parseParenthesizedExpression(tokenStream stream.TokenStream) (ast.Expression, error) {
	if h.verbose {
		fmt.Printf("DEBUG: parseParenthesizedExpression - starting, current token: %v\n", tokenStream.Current())
	}

	// Потребляем левую скобку
	_ = tokenStream.Consume() // (
	if h.verbose {
		fmt.Printf("DEBUG: parseParenthesizedExpression - consumed '(', current token: %v\n", tokenStream.Current())
	}

	// Парсим внутреннее выражение
	expr, err := h.parseExpression(tokenStream)
	if err != nil {
		return nil, err
	}
	if h.verbose {
		fmt.Printf("DEBUG: parseParenthesizedExpression - parsed internal expression, current token: %v\n", tokenStream.Current())
	}

	// Потребляем правую скобку
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenRightParen {
		return nil, fmt.Errorf("expected ')' after parenthesized expression")
	}
	_ = tokenStream.Consume() // )
	if h.verbose {
		fmt.Printf("DEBUG: parseParenthesizedExpression - consumed ')', current token: %v\n", tokenStream.Current())
	}

	return expr, nil // Возвращаем внутреннее выражение без скобок
}

// parseLanguageCall парсит вызов функции другого языка
func (h *MatchHandler) parseLanguageCall(tokenStream stream.TokenStream) (ast.Expression, error) {

	// 1. Читаем язык
	languageToken := tokenStream.Consume()
	language := languageToken.Value

	// 2. Проверяем и потребляем DOT
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenDot {
		return nil, fmt.Errorf("expected DOT after language '%s'", language)
	}
	tokenStream.Consume() // Consuming dot

	// 3. Читаем имя функции (может содержать точки, например, math.sqrt)
	functionParts := []string{}

	// Читаем первый идентификатор функции
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenIdentifier {
		return nil, fmt.Errorf("expected function name after DOT")
	}
	functionToken := tokenStream.Consume()
	functionParts = append(functionParts, functionToken.Value)

	// Читаем дополнительные DOT и идентификаторы
	for tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenDot {
		tokenStream.Consume() // Consuming dot

		if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenIdentifier {
			return nil, fmt.Errorf("expected function name after dot")
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
			return nil, fmt.Errorf("unexpected EOF after '('")
		}

		if tokenStream.Current().Type != lexer.TokenRightParen {
			// Есть хотя бы один аргумент
			for {
				if !tokenStream.HasMore() {
					return nil, fmt.Errorf("unexpected EOF in arguments")
				}

				// Читаем аргумент как сложное выражение с поддержкой тернарных операторов
				if h.verbose {
					fmt.Printf("DEBUG: parseLanguageCall - starting to parse argument, current token: %v\n", tokenStream.Current())
				}

				// Сначала парсим левую часть выражения
				if h.verbose {
					fmt.Printf("DEBUG: parseLanguageCall - about to call parsePrimaryOrComplexExpression\n")
				}
				left, err := h.parsePrimaryOrComplexExpression(tokenStream)
				if err != nil {
					return nil, fmt.Errorf("failed to parse function argument: %v", err)
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
							return nil, fmt.Errorf("failed to parse binary expression in argument: %v", err)
						}

						// После парсинга бинарного выражения проверяем, есть ли тернарный оператор
						if tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenQuestion {
							if h.verbose {
								fmt.Printf("DEBUG: parseLanguageCall - parsing ternary expression after binary expression\n")
							}
							arg, err = h.parseTernaryExpression(tokenStream, arg)
							if err != nil {
								return nil, fmt.Errorf("failed to parse ternary expression after binary expression: %v", err)
							}
						}
					} else if nextToken.Type == lexer.TokenQuestion {
						// Это тернарный оператор
						if h.verbose {
							fmt.Printf("DEBUG: parseLanguageCall - parsing ternary expression\n")
						}
						arg, err = h.parseTernaryExpression(tokenStream, left)
						if err != nil {
							return nil, fmt.Errorf("failed to parse ternary expression in argument: %v", err)
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
						return nil, fmt.Errorf("failed to parse binary expression after argument: %v", err)
					}
				}

				if h.verbose {
					fmt.Printf("DEBUG: parseLanguageCall - finished parsing argument, next token: %v\n", tokenStream.Current())
				}

				arguments = append(arguments, arg)

				// Проверяем разделитель или конец
				if !tokenStream.HasMore() {
					return nil, fmt.Errorf("unexpected EOF after argument")
				}

				nextToken := tokenStream.Current()

				if nextToken.Type == lexer.TokenComma {
					tokenStream.Consume() // Consuming comma
					// После запятой должен быть аргумент
					if !tokenStream.HasMore() {
						return nil, fmt.Errorf("unexpected EOF after comma")
					}
					if tokenStream.Current().Type == lexer.TokenRightParen {
						return nil, fmt.Errorf("unexpected ')' after comma")
					}
				} else if nextToken.Type == lexer.TokenRightParen {
					break
				} else {
					return nil, fmt.Errorf("expected ',' or ')' after argument, got %s", nextToken.Type)
				}
			}
		}

		// 6. Проверяем закрывающую скобку
		if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenRightParen {
			return nil, fmt.Errorf("expected ')' after arguments")
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
		return nil, fmt.Errorf("expected '[' after identifier in index expression")
	}
	_ = tokenStream.Consume() // [

	// 3. Парсим индексное выражение
	index, err := h.parseExpression(tokenStream)
	if err != nil {
		return nil, fmt.Errorf("failed to parse index expression: %v", err)
	}

	// 4. Проверяем и потребляем RBRACKET
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenRBracket {
		return nil, fmt.Errorf("expected ']' after index expression")
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
		return nil, fmt.Errorf("expected DOT after language '%s'", language)
	}
	tokenStream.Consume() // Consuming dot

	// 3. Читаем имя переменной
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenIdentifier {
		return nil, fmt.Errorf("expected variable name after DOT")
	}
	variableToken := tokenStream.Consume()
	variableName := variableToken.Value

	// 4. Проверяем и потребляем LBRACKET
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenLBracket {
		return nil, fmt.Errorf("expected '[' after variable name in language index expression")
	}
	_ = tokenStream.Consume() // [

	// 5. Парсим индексное выражение
	index, err := h.parseExpression(tokenStream)
	if err != nil {
		return nil, fmt.Errorf("failed to parse index expression: %v", err)
	}

	// 6. Проверяем и потребляем RBRACKET
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenRBracket {
		return nil, fmt.Errorf("expected ']' after index expression")
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
			return nil, fmt.Errorf("expected '->' after pattern")
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
			return nil, fmt.Errorf("unexpected EOF after newline")
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
		return nil, fmt.Errorf("unsupported pattern type: %s", currentToken.Type)
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
		var numValue float64
		var err error
		// Проверяем, является ли число шестнадцатеричным
		if len(token.Value) > 2 && (token.Value[0:2] == "0x" || token.Value[0:2] == "0X") {
			// Парсим шестнадцатеричное число
			var hexInt int64
			_, err = fmt.Sscanf(token.Value, "0x%x", &hexInt)
			if err != nil {
				return nil, fmt.Errorf("invalid hexadecimal number format: %s", token.Value)
			}
			numValue = float64(hexInt)
		} else {
			// Парсим десятичное число
			numValue, err = strconv.ParseFloat(token.Value, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid number: %s", token.Value)
			}
		}
		value = numValue
	default:
		return nil, fmt.Errorf("unsupported literal type: %s", token.Type)
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
				return nil, fmt.Errorf("expected variable name after '...'")
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
		return nil, fmt.Errorf("expected ']' to close array pattern")
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
			return nil, fmt.Errorf("object pattern key must be string, identifier or underscore, got %s", keyToken.Type)
		}

		// Потребляем ':'
		if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenColon {
			return nil, fmt.Errorf("expected ':' after object key")
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
		return nil, fmt.Errorf("expected '}' to close object pattern")
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

	// Поддерживаем language calls и assignments
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

			// Если следующий токен '=', это assignment
			if nextToken.Type == lexer.TokenAssign {
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

				return nil, fmt.Errorf("expression cannot be used as statement")
			}
		}

		// Если это простой идентификатор без '=' или '.', возвращаем ошибку
		return nil, fmt.Errorf("unexpected identifier '%s' - expected assignment or language call", currentToken.Value)
	}

	return nil, fmt.Errorf("unsupported statement type: %s", currentToken.Type)
}

// parseAssignmentStatement парсит assignment statement (например, norm = value)
func (h *MatchHandler) parseAssignmentStatement(tokenStream stream.TokenStream) (ast.Statement, error) {
	// Читаем имя переменной
	identifierToken := tokenStream.Consume()
	variableName := identifierToken.Value

	// Потребляем '='
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenAssign {
		return nil, fmt.Errorf("expected '=' after variable name")
	}
	assignToken := tokenStream.Consume()

	// Парсим выражение справа от '='
	valueExpr, err := h.parseExpression(tokenStream)
	if err != nil {
		return nil, fmt.Errorf("failed to parse assignment value: %v", err)
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
			var numValue float64
			// Проверяем, является ли число шестнадцатеричным
			if len(token.Value) > 2 && (token.Value[0:2] == "0x" || token.Value[0:2] == "0X") {
				// Парсим шестнадцатеричное число
				var hexInt int64
				_, err := fmt.Sscanf(token.Value, "0x%x", &hexInt)
				if err != nil {
					return nil, fmt.Errorf("invalid hexadecimal number format: %s", token.Value)
				}
				numValue = float64(hexInt)
			} else {
				// Парсим десятичное число
				numValue, _ = strconv.ParseFloat(token.Value, 64)
			}
			element = &ast.NumberLiteral{Value: numValue, Pos: matchHandlerTokenToPosition(token)}
		case lexer.TokenIdentifier:
			identifierToken := tokenStream.Consume()
			element = &ast.Identifier{
				Name: identifierToken.Value,
				Pos:  matchHandlerTokenToPosition(identifierToken),
			}
		default:
			return nil, fmt.Errorf("unsupported array element type: %s", currentToken.Type)
		}

		elements = append(elements, element)

		// Проверяем запятую
		if tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenComma {
			tokenStream.Consume() // ,
		}
	}

	// Потребляем ]
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenRBracket {
		return nil, fmt.Errorf("expected ']' to close array expression")
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
			return nil, fmt.Errorf("object expression key must be string or identifier, got %s", keyToken.Type)
		}

		// Потребляем ':'
		if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenColon {
			return nil, fmt.Errorf("expected ':' after object key")
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
			var numValue float64
			// Проверяем, является ли число шестнадцатеричным
			if len(token.Value) > 2 && (token.Value[0:2] == "0x" || token.Value[0:2] == "0X") {
				// Парсим шестнадцатеричное число
				var hexInt int64
				_, err := fmt.Sscanf(token.Value, "0x%x", &hexInt)
				if err != nil {
					return nil, fmt.Errorf("invalid hexadecimal number format: %s", token.Value)
				}
				numValue = float64(hexInt)
			} else {
				// Парсим десятичное число
				numValue, _ = strconv.ParseFloat(token.Value, 64)
			}
			value = &ast.NumberLiteral{Value: numValue, Pos: matchHandlerTokenToPosition(token)}
		case lexer.TokenIdentifier:
			identifierToken := tokenStream.Consume()
			value = &ast.Identifier{
				Name: identifierToken.Value,
				Pos:  matchHandlerTokenToPosition(identifierToken),
			}
		default:
			return nil, fmt.Errorf("unsupported object value type: %s", currentToken.Type)
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
		return nil, fmt.Errorf("expected '}' to close object expression")
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
				return nil, fmt.Errorf("unexpected EOF after newline")
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
					return nil, fmt.Errorf("expected size after colon")
				}

				// Парсим размер как выражение (может быть число, переменная или сложное выражение)
				sizeExpr, err := h.parseSizeExpression(tokenStream)
				if err != nil {
					return nil, fmt.Errorf("failed to parse size expression: %v", err)
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
			var numValue float64
			var err error

			// Парсим число
			// Проверяем, является ли число шестнадцатеричным
			if len(token.Value) > 2 && (token.Value[0:2] == "0x" || token.Value[0:2] == "0X") {
				// Парсим шестнадцатеричное число
				var hexInt int64
				_, err = fmt.Sscanf(token.Value, "0x%x", &hexInt)
				if err != nil {
					return nil, fmt.Errorf("invalid hexadecimal number format: %s", token.Value)
				}
				numValue = float64(hexInt)
			} else {
				// Парсим десятичное число
				numValue, err = strconv.ParseFloat(token.Value, 64)
				if err != nil {
					return nil, fmt.Errorf("invalid number: %s", token.Value)
				}
			}

			segment = ast.BitstringSegment{
				Value: &ast.NumberLiteral{Value: numValue, Pos: matchHandlerTokenToPosition(token)},
			}

			// Проверяем наличие размера через двоеточие (:Size)
			if tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenColon {
				tokenStream.Consume() // :

				if !tokenStream.HasMore() {
					return nil, fmt.Errorf("expected size after colon")
				}

				// Парсим размер как выражение (может быть число, переменная или сложное выражение)
				sizeExpr, err := h.parseSizeExpression(tokenStream)
				if err != nil {
					return nil, fmt.Errorf("failed to parse size expression: %v", err)
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
		case lexer.TokenIdentifier, lexer.TokenLua, lexer.TokenPython, lexer.TokenPy, lexer.TokenJS, lexer.TokenNode, lexer.TokenGo:
			// Переменная в битстринге (обычная или языковая)
			currentToken := tokenStream.Current()

			if currentToken.Type == lexer.TokenIdentifier {
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
					return nil, fmt.Errorf("expected '.' after language token %s", languageToken.Value)
				}
				tokenStream.Consume() // .

				// Ожидаем идентификатор переменной
				if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenIdentifier {
					return nil, fmt.Errorf("expected variable name after '%s.'", languageToken.Value)
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
					return nil, fmt.Errorf("expected size after colon")
				}

				// Парсим размер как выражение (может быть число, переменная или сложное выражение)
				sizeExpr, err := h.parseSizeExpression(tokenStream)
				if err != nil {
					return nil, fmt.Errorf("failed to parse size expression: %v", err)
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
			return nil, fmt.Errorf("unexpected token in bitstring pattern: %s", currentToken.Type)
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
		return nil, fmt.Errorf("expected '>>' to close bitstring pattern")
	}

	doubleRightAngleToken := tokenStream.Consume() // >>

	return &ast.BitstringPattern{
		Elements:   segments,
		LeftAngle:  doubleLeftAngleToken,
		RightAngle: doubleRightAngleToken,
		Pos:        matchHandlerTokenToPosition(doubleLeftAngleToken),
	}, nil
}

// parseSizeExpression парсит выражение размера (число, переменная, или сложное выражение в скобках)
func (h *MatchHandler) parseSizeExpression(tokenStream stream.TokenStream) (ast.Expression, error) {
	currentToken := tokenStream.Current()

	switch currentToken.Type {
	case lexer.TokenNumber:
		// Числовое значение
		token := tokenStream.Consume()
		var value float64
		var err error
		// Проверяем, является ли число шестнадцатеричным
		if len(token.Value) > 2 && (token.Value[0:2] == "0x" || token.Value[0:2] == "0X") {
			// Парсим шестнадцатеричное число
			var hexInt int64
			_, err = fmt.Sscanf(token.Value, "0x%x", &hexInt)
			if err != nil {
				return nil, fmt.Errorf("invalid hexadecimal number format: %s", token.Value)
			}
			value = float64(hexInt)
		} else {
			// Парсим десятичное число
			value, err = strconv.ParseFloat(token.Value, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid number: %s", token.Value)
			}
		}
		return &ast.NumberLiteral{Value: value, Pos: matchHandlerTokenToPosition(token)}, nil

	case lexer.TokenIdentifier:
		// Переменная
		token := tokenStream.Consume()
		return &ast.Identifier{Name: token.Value, Pos: matchHandlerTokenToPosition(token)}, nil

	case lexer.TokenLeftParen:
		// Выражение в скобках - извлекаем как строку для funbit
		tokenStream.Consume() // (

		// Собираем все токены до закрывающей скобки в строку
		var expressionParts []string
		parenDepth := 1 // Уже потребили одну открывающую скобку

		for tokenStream.HasMore() && parenDepth > 0 {
			token := tokenStream.Current()

			if token.Type == lexer.TokenLeftParen {
				parenDepth++
			} else if token.Type == lexer.TokenRightParen {
				parenDepth--
			}

			if parenDepth > 0 {
				// Добавляем токен в выражение (кроме закрывающей скобки)
				expressionParts = append(expressionParts, token.Value)
			}

			tokenStream.Consume()
		}

		if parenDepth > 0 {
			return nil, fmt.Errorf("unmatched opening parenthesis in expression")
		}

		// Создаем строковое выражение для funbit
		expressionStr := strings.Join(expressionParts, "")

		// Создаем специальный тип выражения для динамических размеров
		return &ast.StringLiteral{
			Value: expressionStr,
			Pos:   matchHandlerTokenToPosition(currentToken),
		}, nil

	default:
		return nil, fmt.Errorf("unexpected token in expression: %s", currentToken.Type)
	}
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

			// Проверяем, есть ли у спецификатора параметр через двоеточие (например, unit:1)
			if tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenColon {
				tokenStream.Consume() // потребляем ':'

				if !tokenStream.HasMore() {
					return fmt.Errorf("unexpected EOF after colon in specifier")
				}

				// Парсим значение параметра спецификатора
				paramToken := tokenStream.Consume()
				if paramToken.Type != lexer.TokenNumber && paramToken.Type != lexer.TokenIdentifier {
					return fmt.Errorf("expected number or identifier as specifier parameter, got %s", paramToken.Type)
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
			return nil, fmt.Errorf("unexpected EOF in block statement")
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
		return nil, fmt.Errorf("expected '}' at end of block statement")
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
		return nil, fmt.Errorf("expected '(' after 'if'")
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
		return nil, fmt.Errorf("failed to parse if condition: %v", err)
	}

	// Потребляем ')'
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenRightParen {
		return nil, fmt.Errorf("expected ')' after if condition, got %s", tokenStream.Current().Type)
	}
	rParenToken := tokenStream.Consume() // )

	// Парсим тело if (должно быть блоком)
	consequent, err := h.parseBlockStatement(tokenStream)
	if err != nil {
		return nil, fmt.Errorf("failed to parse if body: %v", err)
	}

	// Преобразуем consequent к *ast.BlockStatement
	consequentBlock, ok := consequent.(*ast.BlockStatement)
	if !ok {
		return nil, fmt.Errorf("if body must be a block statement")
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
			return nil, fmt.Errorf("failed to parse else body: %v", err)
		}

		// Преобразуем alternate к *ast.BlockStatement
		alternateBlock, ok := alternate.(*ast.BlockStatement)
		if !ok {
			return nil, fmt.Errorf("else body must be a block statement")
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
		return nil, fmt.Errorf("expected '{' after match expression")
	}
	lBraceToken := tokenStream.Consume() // {

	// 4. Парсим ветки сопоставления
	arms, err := h.parseMatchArms(tokenStream)
	if err != nil {
		return nil, err
	}

	// 5. Потребляем '}'
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenRBrace {
		return nil, fmt.Errorf("expected '}' at end of match statement")
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
		return fmt.Errorf("maximum recursion depth exceeded: %d", rg.maxDepth)
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
