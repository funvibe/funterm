package handler

import (
	"fmt"
	"strconv"

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
				return nil, fmt.Errorf("expected ',' or '>>' after segment, got %s", tokenStream.Current().Type)
			}
		}
	}

	// 3. Потребляем >>
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenDoubleRightAngle {
		return nil, fmt.Errorf("expected '>>' to close bitstring")
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
		return nil, fmt.Errorf("unexpected EOF in segment")
	}

	valueToken := tokenStream.Consume()
	var err error
	segment.Value, err = h.parseExpression(tokenStream, valueToken)
	if err != nil {
		return nil, fmt.Errorf("failed to parse segment value: %v", err)
	}

	// 2. Пропускаем NEWLINE токены после значения
	for tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenNewline {
		tokenStream.Consume()
	}

	// 3. Парсим опциональные размер и спецификаторы (:Size или :Specifiers или :Size/Specifiers)
	if tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenColon {
		segment.ColonToken = tokenStream.Consume() // потребляем ':'

		if !tokenStream.HasMore() {
			return nil, fmt.Errorf("unexpected EOF after colon in segment")
		}

		// Пропускаем NEWLINE токены после двоеточия
		for tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenNewline {
			tokenStream.Consume()
		}

		// Проверяем, что идет после двоеточия
		nextToken := tokenStream.Current()

		if nextToken.Type == lexer.TokenNumber || nextToken.Type == lexer.TokenIdentifier || nextToken.Type == lexer.TokenLeftParen || nextToken.IsLanguageToken() {
			// Это размер (:Size, :Variable или :(Expression))
			sizeToken := tokenStream.Consume()

			// Парсим выражение размера
			sizeExpr, err := h.parseExpression(tokenStream, sizeToken)
			if err != nil {
				return nil, fmt.Errorf("failed to parse segment size: %v", err)
			}

			// Определяем, является ли размер динамическим
			isDynamic := h.isDynamicSizeExpression(sizeExpr)

			if isDynamic {
				// Создаем SizeExpression для динамического размера
				sizeExpression := ast.NewSizeExpression()
				sizeExpression.Pos = tokenToPosition(sizeToken)

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
					}
				} else if nextToken.Type == lexer.TokenLeftParen {
					// Выражение в скобках
					sizeExpression.ExprType = "expression"
					sizeExpression.Expression = sizeExpr
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
					return nil, fmt.Errorf("invalid specifier format - use :size/specifier or :specifier, not :identifier/slash")
				}

				// Это спецификатор типа (:Specifiers)
				segment.Specifiers = append(segment.Specifiers, specToken.Value)
			} else {
				return nil, fmt.Errorf("expected number, identifier, or '(' after colon, got %s", nextToken.Type)
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

		segment.Specifiers = append(segment.Specifiers, specValue)

		// Парсим дополнительные спецификаторы через дефис
		for tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenMinus {
			tokenStream.Consume() // потребляем '-'

			if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenIdentifier {
				return fmt.Errorf("expected specifier after hyphen")
			}

			specToken = tokenStream.Consume()
			specValue = specToken.Value

			// Проверяем, есть ли у спецификатора параметр через двоеточие
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
		numValue, err := strconv.ParseFloat(token.Value, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid number format: %s", token.Value)
		}
		return &ast.NumberLiteral{
			Value: numValue,
			Pos: ast.Position{
				Line:   token.Line,
				Column: token.Column,
				Offset: token.Position,
			},
		}, nil
	case lexer.TokenIdentifier:
		// Проверяем, не является ли это квалифицированной переменной или вызовом функции
		return h.parseComplexIdentifier(tokenStream, token)
	case lexer.TokenLeftParen:
		// Обработка выражений в скобках - используем BinaryExpressionHandler
		return h.parseParenthesizedExpression(tokenStream, token)
	case lexer.TokenNewline:
		// Проверяем, содержит ли NEWLINE текст комментария
		if len(token.Value) > 0 && token.Value[0] == '#' {
			// Это комментарий, пропускаем его и пробуем получить следующий токен
			if !tokenStream.HasMore() {
				return nil, fmt.Errorf("unexpected EOF after comment")
			}
			nextToken := tokenStream.Consume()
			return h.parseExpression(tokenStream, nextToken)
		}
		// Обычный NEWLINE, пропускаем и пробуем получить следующий токен
		if !tokenStream.HasMore() {
			return nil, fmt.Errorf("unexpected EOF after newline")
		}
		nextToken := tokenStream.Consume()
		return h.parseExpression(tokenStream, nextToken)
	default:
		// Проверяем, является ли токен языковым токеном
		if token.IsLanguageToken() {
			// Это языковой токен - обрабатываем как квалифицированную переменную
			return h.parseQualifiedIdentifierOrFunctionCall(tokenStream, token)
		}
		return nil, fmt.Errorf("unsupported expression type: %s", token.Type)
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
		return nil, fmt.Errorf("failed to parse first operand in parentheses: %v", err)
	}

	// Затем парсим полное выражение, начиная с первого операнда
	expr, err := binaryHandler.ParseFullExpression(tempCtx, leftOperand)
	if err != nil {
		return nil, fmt.Errorf("failed to parse expression in parentheses: %v", err)
	}

	// Проверяем и потребляем закрывающую скобку
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenRightParen {
		return nil, fmt.Errorf("expected ')' after expression")
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
		return nil, fmt.Errorf("unsupported language '%s': %v", language, err)
	}

	// Потребляем DOT
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenDot {
		return nil, fmt.Errorf("expected DOT after language '%s'", language)
	}
	tokenStream.Consume() // Consuming DOT

	// Читаем следующий идентификатор
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenIdentifier {
		return nil, fmt.Errorf("expected identifier after DOT")
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
		return nil, fmt.Errorf("expected '(' after function name")
	}
	tokenStream.Consume()

	// Собираем имя функции (может включать дополнительные части через DOT)
	functionParts := []string{functionToken.Value}

	// Читаем дополнительные DOT и идентификаторы для имени функции
	for tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenDot {
		tokenStream.Consume() // Consuming dot

		if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenIdentifier {
			return nil, fmt.Errorf("expected function name after dot")
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
				return nil, fmt.Errorf("unexpected EOF in function arguments")
			}

			// Читаем аргумент
			argToken := tokenStream.Consume()
			arg, err := h.parseExpression(tokenStream, argToken)
			if err != nil {
				return nil, fmt.Errorf("failed to parse function argument: %v", err)
			}
			arguments = append(arguments, arg)

			// Проверяем разделитель
			if !tokenStream.HasMore() {
				return nil, fmt.Errorf("unexpected EOF after function argument")
			}

			if tokenStream.Current().Type == lexer.TokenComma {
				tokenStream.Consume() // Consuming comma
			} else if tokenStream.Current().Type == lexer.TokenRightParen {
				break
			} else {
				return nil, fmt.Errorf("expected ',' or ')' in function arguments, got %s", tokenStream.Current().Type)
			}
		}
	}

	// Потребляем закрывающую скобку
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenRightParen {
		return nil, fmt.Errorf("expected ')' to close function call")
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
			return nil, fmt.Errorf("expected identifier after DOT")
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
		return nil, fmt.Errorf("empty path in qualified variable")
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
