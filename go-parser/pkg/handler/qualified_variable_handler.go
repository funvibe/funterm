package handler

import (
	"fmt"

	"go-parser/pkg/ast"
	"go-parser/pkg/common"
	"go-parser/pkg/config"
	"go-parser/pkg/lexer"
	"go-parser/pkg/stream"
)

// QualifiedVariableHandler - обработчик для квалифицированных переменных (language.variable)
type QualifiedVariableHandler struct {
	config           config.ConstructHandlerConfig
	languageRegistry LanguageRegistry
}

// NewQualifiedVariableHandler создает новый обработчик для квалифицированных переменных
func NewQualifiedVariableHandler(config config.ConstructHandlerConfig) *QualifiedVariableHandler {
	return &QualifiedVariableHandler{
		config:           config,
		languageRegistry: CreateDefaultLanguageRegistry(),
	}
}

// NewQualifiedVariableHandlerWithRegistry создает обработчик с явным указанием Language Registry
func NewQualifiedVariableHandlerWithRegistry(config config.ConstructHandlerConfig, registry LanguageRegistry) *QualifiedVariableHandler {
	return &QualifiedVariableHandler{
		config:           config,
		languageRegistry: registry,
	}
}

// CanHandle проверяет, может ли обработчик обработать токен
func (h *QualifiedVariableHandler) CanHandle(token lexer.Token) bool {
	// Для qualified variables нужен identifier, за которым следует точка
	if token.Type != lexer.TokenIdentifier && !token.IsLanguageToken() {
		return false
	}
	// Проверяем, есть ли следующий токен и является ли он точкой
	// Но CanHandle не имеет доступа к tokenStream, так что проверяем только тип токена
	// Реальную проверку делаем в Handle
	return true
}

// Handle обрабатывает квалифицированную переменную
func (h *QualifiedVariableHandler) Handle(ctx *common.ParseContext) (interface{}, error) {
	tokenStream := ctx.TokenStream

	// 1. Проверяем, есть ли следующий токен и является ли он точкой
	if !tokenStream.HasMore() || tokenStream.Peek().Type != lexer.TokenDot {
		// Это не qualified variable (нет точки), позволяем другим handlers попробовать
		return nil, nil
	}

	// 2. Читаем первый идентификатор (возможный язык)
	languageToken := tokenStream.Current()
	language := languageToken.Value

	// 3. Проверяем, что язык поддерживается
	resolvedLanguage, err := h.languageRegistry.ResolveAlias(language)
	if err != nil {
		// Если язык не поддерживается, возвращаем ожидаемую ошибку
		return nil, newErrorWithTokenPos(languageToken, "not a qualified variable")
	}

	// 4. Проверяем всю структуру перед потреблением токенов
	// Клонируем поток для предварительной проверки
	tempStream := tokenStream.Clone()

	// 4. Читаем путь: language.part1.part2. ... .variable
	// Пропускаем первый DOT
	tempStream.Consume() // Consuming first dot

	// Читаем все части пути
	pathParts := []string{}

	// Читаем первый идентификатор после языка
	if !tempStream.HasMore() || tempStream.Peek().Type != lexer.TokenIdentifier {
		return nil, fmt.Errorf("expected identifier after DOT")
	}
	firstIdentifier := tempStream.Peek()
	tempStream.Consume() // Consuming first identifier
	pathParts = append(pathParts, firstIdentifier.Value)

	// Читаем дополнительные DOT и идентификаторы
	for tempStream.HasMore() && tempStream.Peek().Type == lexer.TokenDot {
		tempStream.Consume() // Consuming dot

		if !tempStream.HasMore() || tempStream.Peek().Type != lexer.TokenIdentifier {
			return nil, fmt.Errorf("expected identifier after dot")
		}

		identifierToken := tempStream.Peek()
		tempStream.Consume() // Consuming identifier
		pathParts = append(pathParts, identifierToken.Value)
	}

	// 5. Проверяем, что есть хотя бы одна часть пути
	if len(pathParts) == 0 {
		return nil, fmt.Errorf("expected at least one identifier after language")
	}

	// 6. Определяем тип операции по следующему токену
	var operationType string // "assignment", "indexed_access", или "read"

	if !tempStream.HasMore() {
		// Если после пути ничего нет - это чтение переменной
		operationType = "read"
	} else {
		nextToken := tempStream.Peek()

		// Если следующий токен - открывающая скобка, это вызов функции, а не квалифицированная переменная
		if nextToken.Type == lexer.TokenLeftParen {
			// Это не квалифицированная переменная, а вызов функции - позволяем другим обработчикам попробовать
			return nil, nil
		}

		// Если следующий токен - левая квадратная скобка, это индексированный доступ
		if nextToken.Type == lexer.TokenLBracket {
			operationType = "indexed_access"
		} else if nextToken.Type == lexer.TokenAssign || nextToken.Type == lexer.TokenColonEquals {
			// Если следующий токен - присваивание, это присваивание
			operationType = "assignment"
		} else {
			// Иначе - чтение переменной (EOF, запятая, точка с запятой, оператор и т.д.)
			operationType = "read"
		}
	}

	// 7. Если все проверки прошли, потребляем токены из основного потока
	tokenStream.Consume() // Consuming language token

	// Проверяем и разрешаем алиас языка через Language Registry
	// (уже проверено выше, но оставляем для consistency)
	if err != nil {
		return nil, fmt.Errorf("unsupported language '%s' in qualified variable: %v", language, err)
	}

	// Читаем путь из основного потока
	tokenStream.Consume() // Consuming first DOT

	// Читаем все части пути
	actualPathParts := []string{}
	var variableToken lexer.Token

	for i := 0; i < len(pathParts); i++ {
		identifierToken := tokenStream.Consume()

		// Последняя часть пути - это переменная
		if i == len(pathParts)-1 {
			variableToken = identifierToken
		} else {
			actualPathParts = append(actualPathParts, identifierToken.Value)
		}

		// Если не последний элемент, потребляем DOT
		if i < len(pathParts)-1 {
			tokenStream.Consume() // Consuming DOT
		}
	}

	variableName := variableToken.Value

	// 8. В зависимости от типа операции создаем соответствующий узел AST
	if operationType == "assignment" {

		// Потребляем знак присваивания
		assignToken := tokenStream.Consume()

		// Обрабатываем значение
		if !tokenStream.HasMore() {
			return nil, fmt.Errorf("expected value after '='")
		}

		// Парсим сложное выражение в правой части присваивания
		value, err := h.parseAssignmentValue(tokenStream)
		if err != nil {
			return nil, fmt.Errorf("failed to parse assignment value: %v", err)
		}

		// Создаем узел присваивания с путем
		identifier := ast.NewQualifiedIdentifierWithPath(languageToken, variableToken, resolvedLanguage, actualPathParts, variableName)
		return ast.NewVariableAssignment(identifier, assignToken, value), nil

	} else if operationType == "indexed_access" {
		// Обработка индексированного доступа: lua.my_dict["key"] или lua.my_dict["key"] = value

		// Создаем qualified identifier для объекта
		identifier := ast.NewQualifiedIdentifierWithPath(languageToken, variableToken, resolvedLanguage, actualPathParts, variableName)

		// Парсим индексированный доступ
		indexExpr, err := h.parseIndexExpressionForQualified(tokenStream, identifier)
		if err != nil {
			return nil, fmt.Errorf("failed to parse index expression: %v", err)
		}

		// Проверяем, есть ли оператор присваивания после индексированного выражения
		if tokenStream.HasMore() && (tokenStream.Current().Type == lexer.TokenAssign || tokenStream.Current().Type == lexer.TokenColonEquals) {
			// Это присваивание: object["key"] = value

			// Потребляем знак присваивания
			assignToken := tokenStream.Consume()

			// Парсим значение
			if !tokenStream.HasMore() {
				return nil, fmt.Errorf("expected value after '='")
			}

			value, err := h.parseAssignmentValue(tokenStream)
			if err != nil {
				return nil, fmt.Errorf("failed to parse assignment value: %v", err)
			}

			// Создаем ExpressionAssignment с индексированным выражением в левой части
			return ast.NewExpressionAssignment(indexExpr, assignToken, value), nil
		} else {
			// Это просто индексированный доступ (чтение)
			return indexExpr, nil
		}

	} else {
		// operationType == "read"

		// Создаем узел чтения переменной с путем
		identifier := ast.NewQualifiedIdentifierWithPath(languageToken, variableToken, resolvedLanguage, actualPathParts, variableName)
		return ast.NewVariableRead(identifier), nil
	}
}

// Config возвращает конфигурацию обработчика
func (h *QualifiedVariableHandler) Config() common.HandlerConfig {
	return common.HandlerConfig{
		IsEnabled: h.config.IsEnabled,
		Priority:  h.config.Priority,
		Name:      h.config.Name,
	}
}

// Name возвращает имя обработчика
func (h *QualifiedVariableHandler) Name() string {
	return h.config.Name
}

// parseAssignmentValue парсит сложное выражение в правой части присваивания
func (h *QualifiedVariableHandler) parseAssignmentValue(tokenStream stream.TokenStream) (ast.Expression, error) {
	currentToken := tokenStream.Current()

	// Проверяем различные типы выражений
	if currentToken.IsLanguageIdentifierOrCallToken() {

		// Проверяем, является ли это LanguageCall с точкой после языка
		tempStream := tokenStream.Clone()

		// Читаем потенциальный язык
		tempStream.Consume()

		// Проверяем, есть ли точка
		if tempStream.HasMore() && tempStream.Current().Type == lexer.TokenDot {
			tempStream.Consume() // consume dot

			// Проверяем, есть ли еще идентификатор после точки
			if tempStream.HasMore() && tempStream.Current().Type == lexer.TokenIdentifier {
				tempStream.Consume()

				// Проверяем, есть ли открывающая скобка (это LanguageCall)
				if tempStream.HasMore() && tempStream.Current().Type == lexer.TokenLeftParen {
					// Это LanguageCall - используем основной поток
					languageCall, parseErr := h.parseLanguageCallValue(tokenStream)
					if parseErr == nil {
						// Проверяем, есть ли оператор pipe после LanguageCall
						if tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenPipe {
							// Это pipe expression - парсим его
							return h.parsePipeExpressionWithValue(tokenStream, languageCall)
						}
						return languageCall, nil
					}
					// Если не удалось распарсить как LanguageCall, возвращаем ошибку
					return nil, parseErr
				}
			}
		}

		// Если не LanguageCall, пробуем распарсить как квалифицированный идентификатор
		result, err := h.parseQualifiedIdentifier(tokenStream)
		if err != nil {
			return nil, err
		}
		return result, nil
	} else if currentToken.Type == lexer.TokenDoubleLeftAngle {
		// Обрабатываем битовую строку
		bitstring, parseErr := h.parseBitstringValue(tokenStream)
		if parseErr != nil {
			return nil, parseErr
		}
		return bitstring, nil
	} else {
		// Используем BinaryExpressionHandler для парсинга сложных выражений (включая Elvis оператор)
		binaryHandler := NewBinaryExpressionHandler(config.ConstructHandlerConfig{})

		// Создаем временный контекст для парсинга
		tempCtx := &common.ParseContext{
			TokenStream: tokenStream,
			Parser:      nil,
			Depth:       0,
			MaxDepth:    100,
			Guard:       &simpleRecursionGuard{maxDepth: 100},
			LoopDepth:   0,
			InputStream: "",
		}

		// Парсим полное выражение с поддержкой бинарных операторов и Elvis оператора
		result, err := binaryHandler.ParseFullExpression(tempCtx, nil)
		if err != nil {
			// Если не удалось распарсить как сложное выражение, пробуем как простое значение
			valueToken := tokenStream.Consume()
			return h.parseSimpleValue(tokenStream, valueToken)
		}
		return result, nil
	}
}

// parseLanguageCallValue парсит LanguageCall как значение в присваивании
func (h *QualifiedVariableHandler) parseLanguageCallValue(tokenStream stream.TokenStream) (*ast.LanguageCall, error) {
	// Логика аналогичная LanguageCallHandler.Handle(), но адаптированная для использования в QualifiedVariableHandler

	// 1. Читаем язык (уже знаем, что это идентификатор)
	languageToken := tokenStream.Consume()
	language := languageToken.Value

	// 2. Проверяем и разрешаем алиас через Language Registry
	resolvedLanguage, err := h.languageRegistry.ResolveAlias(language)
	if err != nil {
		return nil, fmt.Errorf("unsupported language '%s': %v", language, err)
	}

	// 3. Проверяем и потребляем DOT
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenDot {
		return nil, fmt.Errorf("expected DOT after language '%s'", language)
	}
	tokenStream.Consume() // Consuming dot

	// 4. Читаем имя функции (может содержать точки, например, math.sqrt)
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

	// 5. Проверяем и потребляем открывающую скобку
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenLeftParen {
		return nil, fmt.Errorf("expected '(' after function name '%s'", functionName)
	}
	tokenStream.Consume() // Consuming '('

	// 6. Читаем аргументы с поддержкой массивов и объектов
	arguments := make([]ast.Expression, 0)

	// Проверяем, есть ли аргументы
	if !tokenStream.HasMore() {
		return nil, fmt.Errorf("unexpected EOF after argument")
	}

	if tokenStream.Current().Type != lexer.TokenRightParen {
		// Есть хотя бы один аргумент
		for {
			if !tokenStream.HasMore() {
				return nil, fmt.Errorf("unexpected EOF in arguments")
			}

			// Читаем аргумент с поддержкой различных типов
			arg, err := h.parseArgument(tokenStream)
			if err != nil {
				return nil, err
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

	// 7. Проверяем закрывающую скобку
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenRightParen {
		return nil, fmt.Errorf("expected ')' after arguments")
	}
	tokenStream.Consume() // Consuming ')'

	// 8. Создаем узел AST
	startPos := tokenToPosition(languageToken)

	node := &ast.LanguageCall{
		Language:  resolvedLanguage, // Используем разрешенное имя языка
		Function:  functionName,
		Arguments: arguments,
		Pos:       startPos,
	}

	return node, nil
}

// parseArgument парсит один аргумент функции с поддержкой различных типов
func (h *QualifiedVariableHandler) parseArgument(tokenStream stream.TokenStream) (ast.Expression, error) {
	if !tokenStream.HasMore() {
		return nil, fmt.Errorf("unexpected EOF")
	}

	currentToken := tokenStream.Current()

	// Проверяем, является ли аргумент квалифицированным идентификатором
	if currentToken.IsLanguageIdentifierOrCallToken() {

		// Проверяем, есть ли точка после идентификатора (квалифицированный идентификатор)
		tempStream := tokenStream.Clone()
		tempStream.Consume() // consume identifier

		if tempStream.HasMore() && tempStream.Current().Type == lexer.TokenDot {
			// Это квалифицированный идентификатор
			return h.parseQualifiedIdentifier(tokenStream)
		}
	}

	// Обычный аргумент
	token := tokenStream.Consume()
	return h.parseSimpleValue(tokenStream, token)
}

// parseQualifiedIdentifier парсит квалифицированный идентификатор (language.variable)
func (h *QualifiedVariableHandler) parseQualifiedIdentifier(tokenStream stream.TokenStream) (ast.Expression, error) {
	// Читаем язык
	languageToken := tokenStream.Consume()
	language := languageToken.Value

	// Проверяем и разрешаем алиас через Language Registry
	resolvedLanguage, err := h.languageRegistry.ResolveAlias(language)
	if err != nil {
		return nil, fmt.Errorf("unsupported language '%s': %v", language, err)
	}

	// Проверяем и потребляем DOT
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenDot {
		return nil, fmt.Errorf("expected DOT after language '%s'", language)
	}
	tokenStream.Consume() // Consuming dot

	// Читаем имя переменной
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenIdentifier {
		return nil, fmt.Errorf("expected variable name after DOT")
	}
	variableToken := tokenStream.Consume()
	variableName := variableToken.Value

	// Создаем базовый квалифицированный идентификатор
	qualifiedIdent := ast.NewQualifiedIdentifier(languageToken, variableToken, resolvedLanguage, variableName)

	// Проверяем, есть ли индексированный доступ после имени переменной
	if tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenLBracket {
		// Это индексированный доступ - парсим его
		indexExpr, err := h.parseIndexExpressionForQualified(tokenStream, qualifiedIdent)
		if err != nil {
			return nil, fmt.Errorf("failed to parse index expression: %v", err)
		}
		return indexExpr, nil
	}

	// Обычный квалифицированный идентификатор
	return qualifiedIdent, nil
}

// parseSimpleValue парсит простое значение (не LanguageCall)
func (h *QualifiedVariableHandler) parseSimpleValue(tokenStream stream.TokenStream, token lexer.Token) (ast.Expression, error) {
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
		// Преобразуем строку в число
		value, err := parseNumber(token.Value)
		if err != nil {
			return nil, fmt.Errorf("invalid number format: %s", token.Value)
		}
		return createNumberLiteral(token, value), nil
	case lexer.TokenIdentifier:
		return ast.NewIdentifier(token, token.Value), nil
	case lexer.TokenLBrace:
		return h.parseObjectValue(tokenStream, token)
	case lexer.TokenLBracket:
		return h.parseArrayValue(tokenStream, token)
	default:
		return nil, fmt.Errorf("unsupported value type: %s", token.Type)
	}
}

// parseObjectValue парсит объект как значение
func (h *QualifiedVariableHandler) parseObjectValue(tokenStream stream.TokenStream, leftBrace lexer.Token) (ast.Expression, error) {
	object := ast.NewObjectLiteral(leftBrace, lexer.Token{})

	// Проверяем, есть ли свойства
	if !tokenStream.HasMore() {
		return nil, fmt.Errorf("unexpected EOF in object")
	}

	if tokenStream.Current().Type != lexer.TokenRBrace {
		// Есть хотя бы одно свойство
		for {
			if !tokenStream.HasMore() {
				return nil, fmt.Errorf("unexpected EOF in object")
			}

			// Читаем ключ
			keyToken := tokenStream.Consume()
			var key ast.Expression
			var err error

			switch keyToken.Type {
			case lexer.TokenString:
				key = &ast.StringLiteral{
					Value: keyToken.Value,
					Pos: ast.Position{
						Line:   keyToken.Line,
						Column: keyToken.Column,
						Offset: keyToken.Position,
					},
				}
			case lexer.TokenIdentifier:
				key = ast.NewIdentifier(keyToken, keyToken.Value)
			default:
				return nil, fmt.Errorf("unsupported key type: %s", keyToken.Type)
			}

			// Проверяем и потребляем двоеточие
			if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenColon {
				return nil, fmt.Errorf("expected ':' after key")
			}
			tokenStream.Consume() // Consuming colon

			// Читаем значение
			value, err := h.parseArgument(tokenStream)
			if err != nil {
				return nil, err
			}

			// Добавляем свойство в объект
			object.AddProperty(key, value)

			// Проверяем разделитель или конец
			if !tokenStream.HasMore() {
				return nil, fmt.Errorf("unexpected EOF in object")
			}

			nextToken := tokenStream.Current()
			if nextToken.Type == lexer.TokenComma {
				tokenStream.Consume() // Consuming comma
				if !tokenStream.HasMore() {
					return nil, fmt.Errorf("unexpected EOF after comma in object")
				}
				if tokenStream.Current().Type == lexer.TokenRBrace {
					break
				}
			} else if nextToken.Type == lexer.TokenRBrace {
				break
			} else {
				return nil, fmt.Errorf("expected ',' or '}' in object, got %s", nextToken.Type)
			}
		}
	}

	// Проверяем закрывающую скобку
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenRBrace {
		return nil, fmt.Errorf("expected '}' in object")
	}
	rightBrace := tokenStream.Consume()
	object.RightBrace = rightBrace

	return object, nil
}

// parseArrayValue парсит массив как значение
func (h *QualifiedVariableHandler) parseArrayValue(tokenStream stream.TokenStream, leftBracket lexer.Token) (ast.Expression, error) {
	array := ast.NewArrayLiteral(leftBracket, lexer.Token{})

	// Проверяем, есть ли элементы
	if !tokenStream.HasMore() {
		return nil, fmt.Errorf("unexpected EOF in array")
	}

	if tokenStream.Current().Type != lexer.TokenRBracket {
		// Есть хотя бы один элемент
		for {
			if !tokenStream.HasMore() {
				return nil, fmt.Errorf("unexpected EOF in array")
			}

			// Читаем элемент
			element, err := h.parseArgument(tokenStream)
			if err != nil {
				return nil, err
			}

			// Добавляем элемент в массив
			array.AddElement(element)

			// Проверяем разделитель или конец
			if !tokenStream.HasMore() {
				return nil, fmt.Errorf("unexpected EOF in array")
			}

			nextToken := tokenStream.Current()
			if nextToken.Type == lexer.TokenComma {
				tokenStream.Consume() // Consuming comma
				if !tokenStream.HasMore() {
					return nil, fmt.Errorf("unexpected EOF after comma in array")
				}
				if tokenStream.Current().Type == lexer.TokenRBracket {
					break
				}
			} else if nextToken.Type == lexer.TokenRBracket {
				break
			} else {
				return nil, fmt.Errorf("expected ',' or ']' in array, got %s", nextToken.Type)
			}
		}
	}

	// Проверяем закрывающую скобку
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenRBracket {
		return nil, fmt.Errorf("expected ']' in array")
	}
	rightBracket := tokenStream.Consume()
	array.RightBracket = rightBracket

	return array, nil
}

// parseBitstringValue парсит BitstringExpression как значение в присваивании
func (h *QualifiedVariableHandler) parseBitstringValue(tokenStream stream.TokenStream) (*ast.BitstringExpression, error) {
	// Потребляем <<
	leftAngle := tokenStream.Consume()

	// Создаем битовую строку
	bitstring := ast.NewBitstringExpression(leftAngle, lexer.Token{})
	bitstring.Pos = tokenToPosition(leftAngle)

	// Проверяем, есть ли сегменты
	if tokenStream.Current().Type != lexer.TokenDoubleRightAngle {
		// Парсим сегменты
		for {
			segment, err := h.parseBitstringSegment(tokenStream)
			if err != nil {
				return nil, err
			}
			bitstring.AddSegment(*segment)

			// Проверяем разделитель или конец
			if tokenStream.Current().Type == lexer.TokenComma {
				tokenStream.Consume() // потребляем запятую
			} else if tokenStream.Current().Type == lexer.TokenDoubleRightAngle {
				break
			} else {
				return nil, fmt.Errorf("expected ',' or '>>' after segment, got %s", tokenStream.Current().Type)
			}
		}
	}

	// Проверяем закрывающую >>
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenDoubleRightAngle {
		return nil, fmt.Errorf("expected '>>' to close bitstring")
	}
	rightAngle := tokenStream.Consume()
	bitstring.RightAngle = rightAngle

	return bitstring, nil
}

// parseBitstringSegment парсит один сегмент битовой строки
func (h *QualifiedVariableHandler) parseBitstringSegment(tokenStream stream.TokenStream) (*ast.BitstringSegment, error) {
	segment := &ast.BitstringSegment{}

	// Парсим Value
	if !tokenStream.HasMore() {
		return nil, fmt.Errorf("unexpected EOF in segment")
	}

	valueToken := tokenStream.Consume()
	var err error
	segment.Value, err = h.parseSimpleValue(tokenStream, valueToken)
	if err != nil {
		return nil, fmt.Errorf("failed to parse segment value: %v", err)
	}

	// Парсим опциональный размер (:Size)
	if tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenColon {
		segment.ColonToken = tokenStream.Consume() // потребляем ':'

		if !tokenStream.HasMore() {
			return nil, fmt.Errorf("unexpected EOF after colon in segment")
		}

		sizeToken := tokenStream.Consume()
		segment.Size, err = h.parseSimpleValue(tokenStream, sizeToken)
		if err != nil {
			return nil, fmt.Errorf("failed to parse segment size: %v", err)
		}
	}

	// Парсим опциональные спецификаторы (/Specifiers)
	if tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenSlash {
		segment.SlashToken = tokenStream.Consume() // потребляем '/'

		// Парсим первый спецификатор
		if tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenIdentifier {
			specToken := tokenStream.Consume()
			specValue := specToken.Value

			// Проверяем, есть ли у спецификатора параметр через двоеточие (например, unit:1)
			if tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenColon {
				tokenStream.Consume() // потребляем ':'

				if !tokenStream.HasMore() {
					return nil, fmt.Errorf("unexpected EOF after colon in specifier")
				}

				// Парсим значение параметра спецификатора
				paramToken := tokenStream.Consume()
				if paramToken.Type != lexer.TokenNumber && paramToken.Type != lexer.TokenIdentifier {
					return nil, fmt.Errorf("expected number or identifier as specifier parameter, got %s", paramToken.Type)
				}

				// Комбинируем спецификатор и его параметр
				specValue = specValue + ":" + paramToken.Value
			}

			segment.Specifiers = append(segment.Specifiers, specValue)

			// Парсим дополнительные спецификаторы через дефис
			for tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenMinus {
				tokenStream.Consume() // потребляем '-'

				if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenIdentifier {
					return nil, fmt.Errorf("expected specifier after hyphen")
				}

				specToken = tokenStream.Consume()
				specValue = specToken.Value

				// Проверяем, есть ли у спецификатора параметр через двоеточие
				if tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenColon {
					tokenStream.Consume() // потребляем ':'

					if !tokenStream.HasMore() {
						return nil, fmt.Errorf("unexpected EOF after colon in specifier")
					}

					// Парсим значение параметра спецификатора
					paramToken := tokenStream.Consume()
					if paramToken.Type != lexer.TokenNumber && paramToken.Type != lexer.TokenIdentifier {
						return nil, fmt.Errorf("expected number or identifier as specifier parameter, got %s", paramToken.Type)
					}

					// Комбинируем спецификатор и его параметр
					specValue = specValue + ":" + paramToken.Value
				}

				segment.Specifiers = append(segment.Specifiers, specValue)
			}
		}
	}

	return segment, nil
}

// parseIndexExpressionForQualified парсит индексированный доступ для квалифицированного идентификатора
func (h *QualifiedVariableHandler) parseIndexExpressionForQualified(tokenStream stream.TokenStream, qualifiedIdent ast.Expression) (ast.Expression, error) {
	// Потребляем левую квадратную скобку
	lBracketToken := tokenStream.Consume()
	if lBracketToken.Type != lexer.TokenLBracket {
		return nil, fmt.Errorf("expected left bracket, got %s", lBracketToken.Type)
	}

	// Парсим индекс
	indexToken := tokenStream.Current()
	var indexExpr ast.Expression

	switch indexToken.Type {
	case lexer.TokenString:
		tokenStream.Consume()
		indexExpr = &ast.StringLiteral{
			Value: indexToken.Value,
			Pos: ast.Position{
				Line:   indexToken.Line,
				Column: indexToken.Column,
				Offset: indexToken.Position,
			},
		}
	case lexer.TokenNumber:
		tokenStream.Consume()
		// Создаем числовой литерал - конвертируем строку в число
		value, err := parseNumber(indexToken.Value)
		if err != nil {
			return nil, fmt.Errorf("invalid number literal: %s", indexToken.Value)
		}
		indexExpr = createNumberLiteral(indexToken, value)
	case lexer.TokenIdentifier:
		tokenStream.Consume()
		indexExpr = ast.NewIdentifier(indexToken, indexToken.Value)
	default:
		return nil, fmt.Errorf("unsupported index type: %s", indexToken.Type)
	}

	// Потребляем правую квадратную скобку
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenRBracket {
		return nil, fmt.Errorf("expected right bracket after index expression")
	}
	tokenStream.Consume()

	// Создаем первый IndexExpression
	currentObject := qualifiedIdent
	currentIndexExpr := indexExpr
	currentPos := ast.Position{
		Line:   lBracketToken.Line,
		Column: lBracketToken.Column,
		Offset: lBracketToken.Position,
	}

	// Поддержка вложенных индексов: продолжаем парсинг, если есть еще индексы
	for tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenLBracket {
		nextLBracketToken := tokenStream.Consume()
		if nextLBracketToken.Type != lexer.TokenLBracket {
			return nil, fmt.Errorf("expected left bracket for nested index, got %s", nextLBracketToken.Type)
		}

		// Парсим индекс для вложенного доступа
		nestedIndexToken := tokenStream.Current()
		var nestedIndexExpr ast.Expression

		switch nestedIndexToken.Type {
		case lexer.TokenString:
			tokenStream.Consume()
			nestedIndexExpr = &ast.StringLiteral{
				Value: nestedIndexToken.Value,
				Pos: ast.Position{
					Line:   nestedIndexToken.Line,
					Column: nestedIndexToken.Column,
					Offset: nestedIndexToken.Position,
				},
			}
		case lexer.TokenNumber:
			tokenStream.Consume()
			value, err := parseNumber(nestedIndexToken.Value)
			if err != nil {
				return nil, fmt.Errorf("invalid number literal: %s", nestedIndexToken.Value)
			}
			nestedIndexExpr = createNumberLiteral(nestedIndexToken, value)
		case lexer.TokenIdentifier:
			tokenStream.Consume()
			nestedIndexExpr = ast.NewIdentifier(nestedIndexToken, nestedIndexToken.Value)
		default:
			return nil, fmt.Errorf("unsupported nested index type: %s", nestedIndexToken.Type)
		}

		// Потребляем правую квадратную скобку для вложенного индекса
		if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenRBracket {
			return nil, fmt.Errorf("expected right bracket after nested index expression")
		}
		tokenStream.Consume()

		// Создаем вложенный IndexExpression
		currentObject = ast.NewIndexExpression(currentObject, currentIndexExpr, currentPos)
		currentIndexExpr = nestedIndexExpr
		currentPos = ast.Position{
			Line:   nextLBracketToken.Line,
			Column: nextLBracketToken.Column,
			Offset: nextLBracketToken.Position,
		}
	}

	// Создаем финальный IndexExpression (возможно вложенный)
	finalIndexExpr := ast.NewIndexExpression(currentObject, currentIndexExpr, currentPos)

	// Проверяем, есть ли доступ к свойству после индексного выражения (например, py.data.users[0].age)
	for tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenDot {
		// Потребляем DOT
		dotToken := tokenStream.Consume()

		// Проверяем, есть ли идентификатор после DOT
		if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenIdentifier {
			return nil, fmt.Errorf("expected identifier after DOT for property access")
		}

		// Потребляем идентификатор свойства
		propertyToken := tokenStream.Consume()

		// Создаем PropertyExpression для доступа к свойству
		// В текущем AST у нас нет专门的 PropertyExpression, поэтому используем IndexExpression с строковым индексом
		propertyIndex := &ast.StringLiteral{
			Value: propertyToken.Value,
			Pos: ast.Position{
				Line:   propertyToken.Line,
				Column: propertyToken.Column,
				Offset: propertyToken.Position,
			},
		}

		// Создаем новый IndexExpression, где объект - это предыдущий индексированный доступ
		finalIndexExpr = ast.NewIndexExpression(finalIndexExpr, propertyIndex, ast.Position{
			Line:   dotToken.Line,
			Column: dotToken.Column,
			Offset: dotToken.Position,
		})
	}

	return finalIndexExpr, nil
}

// parseIndexExpression парсит индексированный доступ (object["key"] или object["key"] = value)
func (h *QualifiedVariableHandler) parseIndexExpression(tokenStream stream.TokenStream, object ast.Expression) (interface{}, error) {
	// Потребляем левую квадратную скобку
	lBracketToken := tokenStream.Consume()
	if lBracketToken.Type != lexer.TokenLBracket {
		return nil, fmt.Errorf("expected left bracket, got %s", lBracketToken.Type)
	}

	// Парсим индекс
	indexToken := tokenStream.Current()
	var indexExpr ast.Expression

	switch indexToken.Type {
	case lexer.TokenString:
		tokenStream.Consume()
		indexExpr = &ast.StringLiteral{
			Value: indexToken.Value,
			Pos: ast.Position{
				Line:   indexToken.Line,
				Column: indexToken.Column,
				Offset: indexToken.Position,
			},
		}
	case lexer.TokenNumber:
		tokenStream.Consume()
		// Создаем числовой литерал - конвертируем строку в число
		value, err := parseNumber(indexToken.Value)
		if err != nil {
			return nil, fmt.Errorf("invalid number literal: %s", indexToken.Value)
		}
		indexExpr = createNumberLiteral(indexToken, value)
	case lexer.TokenIdentifier:
		tokenStream.Consume()
		indexExpr = ast.NewIdentifier(indexToken, indexToken.Value)
	default:
		return nil, fmt.Errorf("unsupported index type: %s", indexToken.Type)
	}

	// Потребляем правую квадратную скобку
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenRBracket {
		return nil, fmt.Errorf("expected right bracket after index expression")
	}
	tokenStream.Consume()

	// Проверяем, есть ли присваивание после индекса
	if tokenStream.HasMore() && (tokenStream.Current().Type == lexer.TokenAssign || tokenStream.Current().Type == lexer.TokenColonEquals) {
		// Это присваивание: object["key"] = value

		// Потребляем знак присваивания
		assignToken := tokenStream.Consume()

		// Парсим значение
		if !tokenStream.HasMore() {
			return nil, fmt.Errorf("expected value after '='")
		}

		value, err := h.parseAssignmentValue(tokenStream)
		if err != nil {
			return nil, fmt.Errorf("failed to parse assignment value: %v", err)
		}

		// Создаем индексированный доступ
		indexExpression := ast.NewIndexExpression(object, indexExpr, ast.Position{
			Line:   lBracketToken.Line,
			Column: lBracketToken.Column,
			Offset: lBracketToken.Position,
		})

		// Создаем присваивание с индексированным выражением в левой части
		return ast.NewExpressionAssignment(indexExpression, assignToken, value), nil

	} else {
		// Это чтение: object["key"]
		indexExpression := ast.NewIndexExpression(object, indexExpr, ast.Position{
			Line:   lBracketToken.Line,
			Column: lBracketToken.Column,
			Offset: lBracketToken.Position,
		})

		return indexExpression, nil
	}
}

// parsePipeExpressionWithValue парсит pipe expression, начиная с уже разобранного значения
func (h *QualifiedVariableHandler) parsePipeExpressionWithValue(tokenStream stream.TokenStream, firstValue ast.Expression) (ast.Expression, error) {
	stages := []ast.Expression{firstValue}
	operators := []lexer.Token{}

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
		if !h.isValidExpressionStartForPipe(tokenStream.Current()) {
			return nil, fmt.Errorf("invalid expression start after pipe operator: %s", tokenStream.Current().Type)
		}

		// Разбираем следующее выражение
		nextExpr, err := h.parsePipeExpressionStage(tokenStream)
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

// isValidExpressionStartForPipe проверяет, может ли токен начинать выражение в pipe
func (h *QualifiedVariableHandler) isValidExpressionStartForPipe(token lexer.Token) bool {
	return token.IsLanguageIdentifierOrCallToken()
}

// parsePipeExpressionStage разбирает один этап pipe expression
func (h *QualifiedVariableHandler) parsePipeExpressionStage(tokenStream stream.TokenStream) (ast.Expression, error) {
	if !tokenStream.HasMore() {
		return nil, fmt.Errorf("unexpected end of input in pipe expression")
	}

	token := tokenStream.Current()

	// Проверяем, что токен может начинать выражение
	if !h.isValidExpressionStartForPipe(token) {
		return nil, fmt.Errorf("invalid expression start token: %s", token.Type)
	}

	// Разбираем как language call или квалифицированный идентификатор
	if token.IsLanguageIdentifierOrCallToken() {

		// Проверяем, является ли это LanguageCall
		tempStream := tokenStream.Clone()
		tempStream.Consume() // consume language token

		if tempStream.HasMore() && tempStream.Current().Type == lexer.TokenDot {
			tempStream.Consume() // consume dot

			if tempStream.HasMore() && tempStream.Current().Type == lexer.TokenIdentifier {
				tempStream.Consume() // consume function name

				if tempStream.HasMore() && tempStream.Current().Type == lexer.TokenLeftParen {
					// Это LanguageCall с аргументами
					return h.parseLanguageCallValue(tokenStream)
				} else {
					// Это LanguageCall без аргументов
					return h.parseLanguageCallWithoutParentheses(tokenStream)
				}
			}
		}

		// Если не LanguageCall, разбираем как квалифицированный идентификатор
		return h.parseQualifiedIdentifier(tokenStream)
	}

	return nil, fmt.Errorf("unsupported token type in pipe expression: %s", token.Type)
}

// parseLanguageCallWithoutParentheses парсит вызов языка без скобок (например, py.add_one)
func (h *QualifiedVariableHandler) parseLanguageCallWithoutParentheses(tokenStream stream.TokenStream) (ast.Expression, error) {
	// 1. Читаем язык (уже знаем, что это идентификатор)
	languageToken := tokenStream.Consume()
	language := languageToken.Value

	// 2. Проверяем и разрешаем алиас через Language Registry
	resolvedLanguage, err := h.languageRegistry.ResolveAlias(language)
	if err != nil {
		return nil, fmt.Errorf("unsupported language '%s': %v", language, err)
	}

	// 3. Проверяем и потребляем DOT
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenDot {
		return nil, fmt.Errorf("expected DOT after language '%s'", language)
	}
	tokenStream.Consume() // Consuming dot

	// 4. Читаем имя функции
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenIdentifier {
		return nil, fmt.Errorf("expected function name after DOT")
	}
	functionToken := tokenStream.Consume()
	functionName := functionToken.Value

	// 5. Создаем узел AST - LanguageCall без аргументов
	startPos := tokenToPosition(languageToken)

	node := &ast.LanguageCall{
		Language:  resolvedLanguage, // Используем разрешенное имя языка
		Function:  functionName,
		Arguments: []ast.Expression{}, // Пустые аргументы
		Pos:       startPos,
	}

	return node, nil
}
