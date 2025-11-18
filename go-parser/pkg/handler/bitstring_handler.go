package handler

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"go-parser/pkg/ast"
	"go-parser/pkg/common"
	"go-parser/pkg/config"
	"go-parser/pkg/lexer"
	"go-parser/pkg/stream"
)

// BitstringHandler - обработчик для битовых строк <<...>>
type BitstringHandler struct {
	config config.ConstructHandlerConfig
}

// NewBitstringHandler создает новый обработчик для битовых строк
func NewBitstringHandler(config config.ConstructHandlerConfig) *BitstringHandler {
	return &BitstringHandler{config: config}
}

// CanHandle проверяет, может ли обработчик обработать токен
func (h *BitstringHandler) CanHandle(token lexer.Token) bool {
	return token.Type == lexer.TokenDoubleLeftAngle
}

// Handle обрабатывает битовую строку
func (h *BitstringHandler) Handle(ctx *common.ParseContext) (interface{}, error) {
	tokenStream := ctx.TokenStream

	// 1. Потребляем <<
	leftAngle := tokenStream.Consume()

	// 2. Парсим сегменты
	bitstring := ast.NewBitstringExpression(leftAngle, lexer.Token{})
	bitstring.Pos = tokenToPosition(leftAngle)

	// Проверяем, есть ли сегменты
	if tokenStream.Current().Type != lexer.TokenDoubleRightAngle {
		// Парсим сегменты
		for {
			segment, err := h.parseSegment(tokenStream)
			if err != nil {
				return nil, err
			}
			bitstring.AddSegment(*segment)

			// Пропускаем NEWLINE токены после сегмента
			for tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenNewline {
				tokenStream.Consume()
			}

			// Проверяем разделитель или конец
			if tokenStream.Current().Type == lexer.TokenComma {
				tokenStream.Consume() // потребляем запятую

				// Пропускаем NEWLINE токены после запятой
				for tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenNewline {
					tokenStream.Consume()
				}
			} else if tokenStream.Current().Type == lexer.TokenDoubleRightAngle {
				break
			} else {
				return nil, newErrorWithTokenPos(tokenStream.Current(), "expected ',' or '>>' after segment, got %s", tokenStream.Current().Type)
			}
		}
	}

	// 3. Потребляем >>
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenDoubleRightAngle {
		return nil, newErrorWithPos(tokenStream, "expected '>>' to close bitstring")
	}
	rightAngle := tokenStream.Consume()
	bitstring.RightAngle = rightAngle

	return bitstring, nil
}

// parseSegment парсит один сегмент битовой строки
func (h *BitstringHandler) parseSegment(tokenStream stream.TokenStream) (*ast.BitstringSegment, error) {
	segment := &ast.BitstringSegment{}

	// 1. Пропускаем NEWLINE токены перед значением
	for tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenNewline {
		tokenStream.Consume()
	}

	// 2. Парсим Value
	if !tokenStream.HasMore() {
		return nil, newErrorWithPos(tokenStream, "unexpected EOF in segment")
	}

	valueToken := tokenStream.Consume()
	var err error
	segment.Value, err = h.parseExpression(tokenStream, valueToken)
	if err != nil {
		return nil, newErrorWithPos(tokenStream, "failed to parse segment value: %v", err)
	}

	// 2. Пропускаем NEWLINE токены после значения
	for tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenNewline {
		tokenStream.Consume()
	}

	// 3. Парсим опциональные размер и спецификаторы (:Size или :Specifiers или :Size/Specifiers)
	if tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenColon {
		segment.ColonToken = tokenStream.Consume() // потребляем ':'

		if !tokenStream.HasMore() {
			return nil, newErrorWithPos(tokenStream, "unexpected EOF after colon in segment")
		}

		// Пропускаем NEWLINE токены после двоеточия
		for tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenNewline {
			tokenStream.Consume()
		}

		// Проверяем, что идет после двоеточия
		nextToken := tokenStream.Current()

		if nextToken.Type == lexer.TokenNumber || nextToken.Type == lexer.TokenIdentifier || nextToken.Type == lexer.TokenLeftParen || nextToken.IsLanguageToken() {
			// Это размер (:Size, :Variable или :(Expression))
			// Собираем токены для размера выражения
			sizeTokens, err := h.collectSizeTokens(tokenStream)
			if err != nil {
				return nil, newErrorWithPos(tokenStream, "failed to collect size tokens: %v", err)
			}

			// Парсим выражение размера с помощью shunting-yard
			sizeExpr, err := h.parseExpressionWithShuntingYard(sizeTokens)
			if err != nil {
				return nil, newErrorWithPos(tokenStream, "failed to parse segment size: %v", err)
			}

			// Определяем, является ли размер динамическим
			isDynamic := h.isDynamicSizeExpression(sizeExpr)

			if isDynamic {
				// Создаем SizeExpression для динамического размера
				sizeExpression := ast.NewSizeExpression()
				sizeExpression.Pos = tokenToPosition(sizeTokens[0])

				if nextToken.Type == lexer.TokenIdentifier || nextToken.IsLanguageToken() {
					// Простая ссылка на переменную или квалифицированная переменная
					if ident, ok := sizeExpr.(*ast.Identifier); ok {
						sizeExpression.ExprType = "variable"
						sizeExpression.Variable = ident.Name
						sizeExpression.Literal = sizeExpr
					} else {
						// Сложное выражение (включая квалифицированные идентификаторы)
						sizeExpression.ExprType = "expression"
						sizeExpression.Expression = sizeExpr
						// Для совместимости с funbit устанавливаем Variable как строковое представление без внешних скобок
						// Но сохраняем Expression для обратной совместимости
						sizeExpression.Variable = h.expressionToString(sizeExpr)
					}
				} else if nextToken.Type == lexer.TokenLeftParen {
					// Выражение в скобках
					sizeExpression.ExprType = "expression"
					sizeExpression.Expression = sizeExpr
					// Для совместимости с funbit устанавливаем Variable как строковое представление без внешних скобок
					sizeExpression.Variable = h.expressionToString(sizeExpr)
				} else {
					// Литеральное значение (число) - может быть использовано в выражениях
					sizeExpression.ExprType = "literal"
					sizeExpression.Literal = sizeExpr
				}

				segment.SizeExpression = sizeExpression
				segment.IsDynamicSize = true
				// Также сохраняем в старом поле для обратной совместимости
				segment.Size = sizeExpr
			} else {
				// Статический размер - используем старый подход
				segment.Size = sizeExpr
				segment.IsDynamicSize = false
			}

			// Проверяем, есть ли спецификаторы после размера (:Size/Specifiers)
			if tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenSlash {
				segment.SlashToken = tokenStream.Consume() // потребляем '/'

				// Парсим спецификаторы
				if err := h.parseSpecifiers(tokenStream, segment); err != nil {
					return nil, err
				}
			}
		} else {
			// Это спецификатор типа (:Specifiers) - только если это идентификатор без слэша после
			if nextToken.Type == lexer.TokenIdentifier {
				specToken := tokenStream.Consume()

				// Проверяем, что после идентификатора нет слэша (иначе это была бы ошибка)
				if tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenSlash {
					return nil, newErrorWithPos(tokenStream, "invalid specifier format - use :size/specifier or :specifier, not :identifier/slash")
				}

				// Это спецификатор типа (:Specifiers)
				segment.Specifiers = append(segment.Specifiers, specToken.Value)
			} else {
				return nil, newErrorWithTokenPos(nextToken, "expected number, identifier, or '(' after colon, got %s", nextToken.Type)
			}
		}
	}

	// 4. Пропускаем NEWLINE токены перед спецификаторами
	for tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenNewline {
		tokenStream.Consume()
	}

	// 5. Парсим опциональные спецификаторы через слэш (/Specifiers) - если не было двоеточия
	if segment.ColonToken.Type == lexer.TokenEOF && tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenSlash {
		segment.SlashToken = tokenStream.Consume() // потребляем '/'

		// Парсим спецификаторы
		if err := h.parseSpecifiers(tokenStream, segment); err != nil {
			return nil, err
		}
	}

	return segment, nil
}

// parseSpecifiers парсит спецификаторы типа после слэша
func (h *BitstringHandler) parseSpecifiers(tokenStream stream.TokenStream, segment *ast.BitstringSegment) error {
	// Парсим первый спецификатор
	if tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenIdentifier {
		specToken := tokenStream.Consume()
		specValue := specToken.Value

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

		segment.Specifiers = append(segment.Specifiers, specValue)

		// Парсим дополнительные спецификаторы через дефис
		for tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenMinus {
			tokenStream.Consume() // потребляем '-'

			if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenIdentifier {
				return newErrorWithPos(tokenStream, "expected specifier after hyphen")
			}

			specToken = tokenStream.Consume()
			specValue = specToken.Value

			// Проверяем, есть ли у спецификатора параметр через двоеточие
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

			segment.Specifiers = append(segment.Specifiers, specValue)
		}
	}

	return nil
}

// parseExpression парсит выражение (значение в сегменте) с поддержкой квалифицированных переменных
func (h *BitstringHandler) parseExpression(tokenStream stream.TokenStream, token lexer.Token) (ast.Expression, error) {
	switch token.Type {
	case lexer.TokenString:
		return &ast.StringLiteral{
			Value: token.Value,
			Pos: ast.Position{
				Line:   token.Line,
				Column: token.Column,
				Offset: token.Position,
			},
		}, nil
	case lexer.TokenNumber:
		numValue, err := parseNumber(token.Value)
		if err != nil {
			return nil, newErrorWithTokenPos(token, "invalid number format: %s", token.Value)
		}
		return createNumberLiteral(token, numValue), nil
	case lexer.TokenIdentifier:
		// Проверяем, не является ли это квалифицированной переменной или вызовом функции
		return h.parseComplexIdentifier(tokenStream, token)
	case lexer.TokenUnderscore:
		// Special case for wildcard pattern '_'
		return &ast.Identifier{
			Name: token.Value,
			Pos: ast.Position{
				Line:   token.Line,
				Column: token.Column,
				Offset: token.Position,
			},
		}, nil
	case lexer.TokenLeftParen:
		// Обработка выражений в скобках - используем BinaryExpressionHandler
		return h.parseParenthesizedExpression(tokenStream, token)
	case lexer.TokenNewline:
		// Проверяем, содержит ли NEWLINE текст комментария
		if len(token.Value) > 0 && token.Value[0] == '#' {
			// Это комментарий, пропускаем его и пробуем получить следующий токен
			if !tokenStream.HasMore() {
				return nil, newErrorWithPos(tokenStream, "unexpected EOF after comment")
			}
			nextToken := tokenStream.Consume()
			return h.parseExpression(tokenStream, nextToken)
		}
		// Обычный NEWLINE, пропускаем и пробуем получить следующий токен
		if !tokenStream.HasMore() {
			return nil, newErrorWithPos(tokenStream, "unexpected EOF after newline")
		}
		nextToken := tokenStream.Consume()
		return h.parseExpression(tokenStream, nextToken)
	default:
		// Проверяем, является ли токен языковым токеном
		if token.IsLanguageToken() {
			// Это языковой токен - обрабатываем как квалифицированную переменную
			return h.parseQualifiedIdentifierOrFunctionCall(tokenStream, token)
		}
		return nil, newErrorWithTokenPos(token, "unsupported expression type: %s", token.Type)
	}
}

// parseParenthesizedExpression парсит выражение в скобках
func (h *BitstringHandler) parseParenthesizedExpression(tokenStream stream.TokenStream, leftParenToken lexer.Token) (ast.Expression, error) {
	// Потребляем открывающую скобку (если еще не потреблена)
	if tokenStream.Current().Type == lexer.TokenLeftParen {
		tokenStream.Consume() // потребляем '('
	}

	// Создаем временный контекст для парсинга выражения в скобках
	tempCtx := &common.ParseContext{
		TokenStream: tokenStream,
		Parser:      nil,
		Depth:       0,
		MaxDepth:    100,
		Guard:       &simpleRecursionGuard{maxDepth: 100},
		LoopDepth:   0,
		InputStream: "",
	}

	// Используем BinaryExpressionHandler для парсинга выражения внутри скобок
	binaryHandler := NewBinaryExpressionHandler(config.ConstructHandlerConfig{})

	// Сначала парсим первый операнд внутри скобок
	leftOperand, err := binaryHandler.parseOperand(tempCtx)
	if err != nil {
		return nil, newErrorWithPos(tokenStream, "failed to parse first operand in parentheses: %v", err)
	}

	// Затем парсим полное выражение, начиная с первого операнда
	expr, err := binaryHandler.ParseFullExpression(tempCtx, leftOperand)
	if err != nil {
		return nil, newErrorWithPos(tokenStream, "failed to parse expression in parentheses: %v", err)
	}

	// Проверяем и потребляем закрывающую скобку
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenRightParen {
		return nil, newErrorWithPos(tokenStream, "expected ')' after expression")
	}
	tokenStream.Consume() // потребляем ')'

	return expr, nil
}

// parseComplexIdentifier парсит сложный идентификатор (квалифицированная переменная или вызов функции)
func (h *BitstringHandler) parseComplexIdentifier(tokenStream stream.TokenStream, firstToken lexer.Token) (ast.Expression, error) {
	// Проверяем, есть ли DOT после идентификатора
	if tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenDot {
		// Это может быть квалифицированная переменная или вызов функции
		return h.parseQualifiedIdentifierOrFunctionCall(tokenStream, firstToken)
	} else {
		// Простой идентификатор
		return ast.NewIdentifier(firstToken, firstToken.Value), nil
	}
}

// parseQualifiedIdentifierOrFunctionCall парсит квалифицированную переменную или вызов функции
func (h *BitstringHandler) parseQualifiedIdentifierOrFunctionCall(tokenStream stream.TokenStream, languageToken lexer.Token) (ast.Expression, error) {
	// Используем встроенный метод токена для преобразования в строковое представление языка
	language := languageToken.LanguageTokenToString()

	// Проверяем, что язык поддерживается
	languageRegistry := CreateDefaultLanguageRegistry()
	resolvedLanguage, err := languageRegistry.ResolveAlias(language)
	if err != nil {
		return nil, newErrorWithTokenPos(languageToken, "unsupported language '%s': %v", language, err)
	}

	// Потребляем DOT
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenDot {
		return nil, newErrorWithPos(tokenStream, "expected DOT after language '%s'", language)
	}
	tokenStream.Consume() // Consuming DOT

	// Читаем следующий идентификатор
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenIdentifier {
		return nil, newErrorWithPos(tokenStream, "expected identifier after DOT")
	}
	nextToken := tokenStream.Consume()

	// Проверяем, есть ли скобка (вызов функции) или еще DOT (квалифицированная переменная)
	if tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenLeftParen {
		// Это вызов функции
		return h.parseFunctionCall(tokenStream, languageToken, nextToken, resolvedLanguage)
	} else if tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenDot {
		// Это квалифицированная переменная с путем
		return h.parseQualifiedVariableWithPath(tokenStream, languageToken, nextToken, resolvedLanguage)
	} else {
		// Простая квалифицированная переменная (language.variable)
		variableName := nextToken.Value
		identifier := ast.NewQualifiedIdentifier(languageToken, nextToken, resolvedLanguage, variableName)
		return identifier, nil
	}
}

// parseFunctionCall парсит вызов функции
func (h *BitstringHandler) parseFunctionCall(tokenStream stream.TokenStream, languageToken lexer.Token, functionToken lexer.Token, resolvedLanguage string) (ast.Expression, error) {
	// Потребляем открывающую скобку
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenLeftParen {
		return nil, newErrorWithPos(tokenStream, "expected '(' after function name")
	}
	tokenStream.Consume()

	// Собираем имя функции (может включать дополнительные части через DOT)
	functionParts := []string{functionToken.Value}

	// Читаем дополнительные DOT и идентификаторы для имени функции
	for tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenDot {
		tokenStream.Consume() // Consuming dot

		if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenIdentifier {
			return nil, newErrorWithPos(tokenStream, "expected function name after dot")
		}
		functionToken = tokenStream.Consume()
		functionParts = append(functionParts, functionToken.Value)
	}

	// Собираем полное имя функции
	functionName := ""
	for i, part := range functionParts {
		if i > 0 {
			functionName += "."
		}
		functionName += part
	}

	// Парсим аргументы
	arguments := make([]ast.Expression, 0)

	if tokenStream.Current().Type != lexer.TokenRightParen {
		// Есть аргументы
		for {
			if !tokenStream.HasMore() {
				return nil, newErrorWithPos(tokenStream, "unexpected EOF in function arguments")
			}

			// Читаем аргумент
			argToken := tokenStream.Consume()
			arg, err := h.parseExpression(tokenStream, argToken)
			if err != nil {
				return nil, newErrorWithPos(tokenStream, "failed to parse function argument: %v", err)
			}
			arguments = append(arguments, arg)

			// Проверяем разделитель
			if !tokenStream.HasMore() {
				return nil, newErrorWithPos(tokenStream, "unexpected EOF after function argument")
			}

			if tokenStream.Current().Type == lexer.TokenComma {
				tokenStream.Consume() // Consuming comma
			} else if tokenStream.Current().Type == lexer.TokenRightParen {
				break
			} else {
				return nil, newErrorWithTokenPos(tokenStream.Current(), "expected ',' or ')' in function arguments, got %s", tokenStream.Current().Type)
			}
		}
	}

	// Потребляем закрывающую скобку
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenRightParen {
		return nil, newErrorWithPos(tokenStream, "expected ')' to close function call")
	}
	tokenStream.Consume()

	// Создаем узел LanguageCall
	startPos := tokenToPosition(languageToken)
	return &ast.LanguageCall{
		Language:  resolvedLanguage,
		Function:  functionName,
		Arguments: arguments,
		Pos:       startPos,
	}, nil
}

// parseQualifiedVariableWithPath парсит квалифицированную переменную с путем (language.part1.part2.variable)
func (h *BitstringHandler) parseQualifiedVariableWithPath(tokenStream stream.TokenStream, languageToken lexer.Token, firstPartToken lexer.Token, resolvedLanguage string) (ast.Expression, error) {
	pathParts := []string{firstPartToken.Value}

	// Читаем дополнительные части пути
	for {
		// Потребляем DOT
		if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenDot {
			break
		}
		tokenStream.Consume()

		// Читаем следующую часть пути
		if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenIdentifier {
			return nil, newErrorWithPos(tokenStream, "expected identifier after DOT")
		}
		partToken := tokenStream.Consume()
		pathParts = append(pathParts, partToken.Value)

		// Проверяем, следующий токен - если не DOT, то это конец пути
		if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenDot {
			break
		}
	}

	// Последняя часть пути - это имя переменной
	if len(pathParts) == 0 {
		return nil, newErrorWithPos(tokenStream, "empty path in qualified variable")
	}

	variableName := pathParts[len(pathParts)-1]
	actualPathParts := pathParts[:len(pathParts)-1]
	variableToken := lexer.Token{
		Type:     lexer.TokenIdentifier,
		Value:    variableName,
		Line:     languageToken.Line,
		Column:   languageToken.Column + len(languageToken.Value) + 1, // Приблизительно
		Position: languageToken.Position,
	}

	// Создаем QualifiedIdentifier с путем
	identifier := ast.NewQualifiedIdentifierWithPath(languageToken, variableToken, resolvedLanguage, actualPathParts, variableName)
	return identifier, nil
}

// Config возвращает конфигурацию обработчика
func (h *BitstringHandler) Config() common.HandlerConfig {
	return common.HandlerConfig{
		IsEnabled: h.config.IsEnabled,
		Priority:  h.config.Priority,
		Name:      h.config.Name,
	}
}

// Name возвращает имя обработчика
func (h *BitstringHandler) Name() string {
	return h.config.Name
}

// isDynamicSizeExpression проверяет, является ли выражение размера динамическим
// (содержит переменные или сложные выражения)
func (h *BitstringHandler) isDynamicSizeExpression(expr ast.Expression) bool {
	switch e := expr.(type) {
	case *ast.Identifier:
		// Ссылка на переменную - динамическая
		return true
	case *ast.NumberLiteral:
		// Литеральное число - статическое
		return false
	case *ast.BinaryExpression:
		// Бинарное выражение (сложение, умножение и т.д.) - динамическое
		return true
	case *ast.LanguageCall:
		// Вызов функции - динамический
		return true
	case *ast.UnaryExpression:
		// Унарное выражение - проверим операнд
		return h.isDynamicSizeExpression(e.Right)
	default:
		// Другие типы выражений считаем динамическими для безопасности
		return true
	}
}

// parseExpressionWithShuntingYard парсит выражение используя алгоритм shunting-yard
func (h *BitstringHandler) parseExpressionWithShuntingYard(tokens []lexer.Token) (ast.Expression, error) {
	if len(tokens) == 0 {
		return nil, nil
	}

	// Стеки для алгоритма shunting-yard
	var outputQueue []ast.Expression
	var operatorStack []lexer.Token

	i := 0
	for i < len(tokens) {
		token := tokens[i]

		switch token.Type {
		case lexer.TokenNumber:
			// Числа идут в output queue
			expr, err := h.createNumberLiteral(token)
			if err != nil {
				return nil, err
			}
			outputQueue = append(outputQueue, expr)
			i++

		case lexer.TokenIdentifier:
			// Идентификаторы идут в output queue
			expr := h.createIdentifier(token)
			outputQueue = append(outputQueue, expr)
			i++

		case lexer.TokenLeftParen:
			// Для битстрингов обрабатываем скобки как часть выражения
			// Просто добавляем оператор в стек без создания NestedExpression
			operatorStack = append(operatorStack, token)
			i++

		case lexer.TokenRightParen:
			// Обрабатываем закрывающую скобку
			for len(operatorStack) > 0 {
				topOp := operatorStack[len(operatorStack)-1]
				operatorStack = operatorStack[:len(operatorStack)-1]

				if topOp.Type == lexer.TokenLeftParen {
					break
				}

				binaryExpr, err := h.createBinaryExpressionFromStack(topOp, &outputQueue)
				if err != nil {
					return nil, err
				}
				outputQueue = append(outputQueue, binaryExpr)
			}
			i++

		default:
			// Проверяем, является ли токен оператором
			if info, isOp := h.getOperatorInfo(token.Type); isOp {
				// Обрабатываем операторы в стеке
				for len(operatorStack) > 0 {
					topOp := operatorStack[len(operatorStack)-1]
					topInfo, topIsOp := h.getOperatorInfo(topOp.Type)

					if !topIsOp {
						break
					}

					// Если оператор на вершине стека имеет более высокий приоритет
					// или такой же приоритет и лево-ассоциативный
					if (info.associative && info.precedence <= topInfo.precedence) ||
						(!info.associative && info.precedence < topInfo.precedence) {

						// Перемещаем оператор в output queue
						binaryExpr, err := h.createBinaryExpressionFromStack(topOp, &outputQueue)
						if err != nil {
							return nil, err
						}
						outputQueue = append(outputQueue, binaryExpr)
						operatorStack = operatorStack[:len(operatorStack)-1]
					} else {
						break
					}
				}

				// Добавляем текущий оператор в стек
				operatorStack = append(operatorStack, token)
				i++
			} else {
				// Неизвестный токен
				return nil, fmt.Errorf("неизвестный токен в выражении: %s в строке %d, колонка %d",
					token.Value, token.Line, token.Column)
			}
		}
	}

	// Перемещаем оставшиеся операторы в output queue
	for len(operatorStack) > 0 {
		op := operatorStack[len(operatorStack)-1]
		operatorStack = operatorStack[:len(operatorStack)-1]

		if op.Type == lexer.TokenLeftParen {
			return nil, fmt.Errorf("несбалансированная скобка: отсутствует ')' в строке %d, колонка %d",
				op.Line, op.Column)
		}

		binaryExpr, err := h.createBinaryExpressionFromStack(op, &outputQueue)
		if err != nil {
			return nil, err
		}
		outputQueue = append(outputQueue, binaryExpr)
	}

	// Возвращаем единственное выражение из output queue
	if len(outputQueue) != 1 {
		return nil, fmt.Errorf("ошибка парсинга: ожидалось одно выражение, получено %d", len(outputQueue))
	}

	return outputQueue[0], nil
}

// BitstringOperatorInfo содержит информацию об операторе
type BitstringOperatorInfo struct {
	precedence  int
	associative bool // true для left-associative, false для right-associative
}

// getOperatorInfo возвращает информацию об операторе
func (h *BitstringHandler) getOperatorInfo(tokenType lexer.TokenType) (BitstringOperatorInfo, bool) {
	operatorPrecedence := map[lexer.TokenType]BitstringOperatorInfo{
		lexer.TokenOr:           {precedence: 1, associative: true},
		lexer.TokenAnd:          {precedence: 2, associative: true},
		lexer.TokenEqual:        {precedence: 3, associative: true},
		lexer.TokenNotEqual:     {precedence: 3, associative: true},
		lexer.TokenLess:         {precedence: 4, associative: true},
		lexer.TokenGreater:      {precedence: 4, associative: true},
		lexer.TokenLessEqual:    {precedence: 4, associative: true},
		lexer.TokenGreaterEqual: {precedence: 4, associative: true},
		lexer.TokenPlus:         {precedence: 5, associative: true},
		lexer.TokenMinus:        {precedence: 5, associative: true},
		lexer.TokenMultiply:     {precedence: 6, associative: true},
		lexer.TokenSlash:        {precedence: 6, associative: true},
		lexer.TokenModulo:       {precedence: 6, associative: true},
		lexer.TokenPower:        {precedence: 7, associative: false}, // right-associative
		lexer.TokenAssign:       {precedence: 0, associative: true}, // assignment has lowest precedence
	}

	info, exists := operatorPrecedence[tokenType]
	return info, exists
}


// createNumberLiteral создает числовой литерал
func (h *BitstringHandler) createNumberLiteral(token lexer.Token) (ast.Expression, error) {
	pos := ast.Position{
		Line:   token.Line,
		Column: token.Column,
		Offset: token.Position,
	}

	// Пытаемся парсить как integer, потом как float
	if intVal, err := strconv.ParseInt(token.Value, 10, 64); err == nil {
		return &ast.NumberLiteral{
			IntValue: big.NewInt(intVal),
			IsInt:    true,
			Pos:      pos,
		}, nil
	}

	if floatVal, err := strconv.ParseFloat(token.Value, 64); err == nil {
		return &ast.NumberLiteral{
			FloatValue: floatVal,
			IsInt:      false,
			Pos:        pos,
		}, nil
	}

	return nil, fmt.Errorf("неверный числовой литерал: %s", token.Value)
}

// createIdentifier создает идентификатор
func (h *BitstringHandler) createIdentifier(token lexer.Token) ast.Expression {
	return &ast.Identifier{
		Name: token.Value,
		Pos: ast.Position{
			Line:   token.Line,
			Column: token.Column,
			Offset: token.Position,
		},
	}
}

// createBinaryExpressionFromStack создает бинарное выражение из оператора и стека
func (h *BitstringHandler) createBinaryExpressionFromStack(op lexer.Token, outputQueue *[]ast.Expression) (ast.Expression, error) {
	if len(*outputQueue) < 2 {
		return nil, fmt.Errorf("insufficient operands for operator %s", op.Value)
	}

	// Извлекаем правый и левый операнды
	right := (*outputQueue)[len(*outputQueue)-1]
	left := (*outputQueue)[len(*outputQueue)-2]

	// Удаляем операнды из очереди
	*outputQueue = (*outputQueue)[:len(*outputQueue)-2]

	// Создаем бинарное выражение
	pos := ast.Position{
		Line:   op.Line,
		Column: op.Column,
		Offset: op.Position,
	}

	return &ast.BinaryExpression{
		Left:     left,
		Operator: op.Value,
		Right:    right,
		BaseNode: ast.BaseNode{},
		Pos:      pos,
	}, nil
}

// expressionToString конвертирует выражение в строку для funbit
func (h *BitstringHandler) expressionToString(expr ast.Expression) string {
	if str, ok := expr.(interface{ String() string }); ok {
		exprStr := str.String()
		// Убираем внешние скобки для funbit
		if strings.HasPrefix(exprStr, "(") && strings.HasSuffix(exprStr, ")") {
			exprStr = exprStr[1 : len(exprStr)-1]
		}
		return exprStr
	}
	return ""
}

// collectSizeTokens собирает токены для выражения размера до разделителей
func (h *BitstringHandler) collectSizeTokens(tokenStream stream.TokenStream) ([]lexer.Token, error) {
	var tokens []lexer.Token
	depth := 0

	for tokenStream.HasMore() {
		token := tokenStream.Current()

		// Проверяем на разделители, которые заканчивают выражение размера
		if depth == 0 && (token.Type == lexer.TokenComma || token.Type == lexer.TokenSlash || token.Type == lexer.TokenDoubleRightAngle) {
			break
		}

		// Отслеживаем вложенность скобок
		if token.Type == lexer.TokenLeftParen {
			depth++
		} else if token.Type == lexer.TokenRightParen {
			depth--
			if depth < 0 {
				return nil, newErrorWithTokenPos(token, "unexpected ')' in size expression")
			}
		}

		tokens = append(tokens, token)
		tokenStream.Consume()

		// Если глубина стала 0 и следующий токен - разделитель, останавливаемся
		if depth == 0 && tokenStream.HasMore() {
			nextToken := tokenStream.Current()
			if nextToken.Type == lexer.TokenComma || nextToken.Type == lexer.TokenSlash || nextToken.Type == lexer.TokenDoubleRightAngle {
				break
			}
		}
	}

	if depth > 0 {
		return nil, newErrorWithPos(tokenStream, "unclosed parentheses in size expression")
	}

	if len(tokens) == 0 {
		return nil, newErrorWithPos(tokenStream, "empty size expression")
	}

	return tokens, nil
}

// isBitstringConcatenation проверяет, является ли битовая строка конкатенацией
// (переменные без спецификаторов размера и типа)
func (h *BitstringHandler) isBitstringConcatenation(tokenStream stream.TokenStream) bool {
	// Клонируем поток, чтобы не потреблять токены
	clone := tokenStream.Clone()

	// Пропускаем <<
	clone.Consume()

	// Флаги для отслеживания контекста
	afterSlash := false // true после / (спецификаторы типа)

	// Проверяем токены на наличие идентификаторов без спецификаторов
	peekAhead := 15 // Проверяем до 15 токенов вперед

	for i := 0; i < peekAhead && clone.HasMore(); i++ {
		token := clone.Current()

		// Обновляем контекст
		if token.Type == lexer.TokenSlash {
			afterSlash = true
		} else if token.Type == lexer.TokenComma {
			afterSlash = false
		} else if token.Type == lexer.TokenColon {
			afterSlash = false
		}

		if token.Type == lexer.TokenDoubleRightAngle {
			break // Дошли до конца битовой строки
		}

		if token.Type == lexer.TokenIdentifier {
			// Если идентификатор идет после / - это спецификатор типа, нормально
			if afterSlash {
				clone.Consume()
				continue
			}

			// Идентификатор не после / - проверяем, есть ли спецификатор размера
			clone.Consume()

			// Проверяем следующий токен
			if clone.HasMore() {
				nextToken := clone.Current()
				if nextToken.Type != lexer.TokenColon {
					// Идентификатор без : после него - проверяем дальше
					// Пропускаем токены до следующего разделителя или конца
					for clone.HasMore() && clone.Current().Type != lexer.TokenComma &&
						clone.Current().Type != lexer.TokenSlash &&
						clone.Current().Type != lexer.TokenDoubleRightAngle {
						clone.Consume()
					}

					// Если после идентификатора не было / перед разделителем - это конкатенация
					if clone.HasMore() && clone.Current().Type != lexer.TokenSlash {
						return true
					}
				}
				// Идентификатор со спецификатором размера - это нормально
			} else {
				// Идентификатор в конце без спецификаторов - конкатенация
				return true
			}
		} else if token.Type == lexer.TokenNumber {
			clone.Consume()

			// Проверяем следующий токен для чисел
			if clone.HasMore() {
				nextToken := clone.Current()
				// Если после числа не идет :, /, ,, или >> - это подозрительно
				if nextToken.Type != lexer.TokenColon && nextToken.Type != lexer.TokenSlash &&
					nextToken.Type != lexer.TokenComma && nextToken.Type != lexer.TokenDoubleRightAngle {
					// Это может быть конкатенацией (число без спецификатора)
					return true
				}
			}
		} else {
			clone.Consume() // Пропускаем другие токены
		}
	}

	return false
}
