package handler

import (
	"fmt"
	"strconv"

	"go-parser/pkg/ast"
	"go-parser/pkg/common"
	"go-parser/pkg/config"
	"go-parser/pkg/lexer"
)

// LanguageCallHandler - обработчик для вызовов функций других языков
type LanguageCallHandler struct {
	config              config.ConstructHandlerConfig
	languageRegistry    LanguageRegistry
	verbose             bool
	skipBackgroundCheck bool
}

// NewLanguageCallHandler создает новый обработчик для вызовов функций других языков
func NewLanguageCallHandler(config config.ConstructHandlerConfig) *LanguageCallHandler {
	return NewLanguageCallHandlerWithVerbose(config, false)
}

// NewLanguageCallHandlerWithVerbose создает новый обработчик для вызовов функций других языков с поддержкой verbose режима
func NewLanguageCallHandlerWithVerbose(config config.ConstructHandlerConfig, verbose bool) *LanguageCallHandler {
	return &LanguageCallHandler{
		config:              config,
		languageRegistry:    CreateDefaultLanguageRegistry(),
		verbose:             verbose,
		skipBackgroundCheck: false,
	}
}

// NewLanguageCallHandlerWithRegistry создает обработчик с явным указанием Language Registry
func NewLanguageCallHandlerWithRegistry(config config.ConstructHandlerConfig, registry LanguageRegistry) *LanguageCallHandler {
	return NewLanguageCallHandlerWithRegistryAndVerbose(config, registry, false)
}

// NewLanguageCallHandlerWithRegistryAndVerbose создает обработчик с явным указанием Language Registry и поддержкой verbose режима
func NewLanguageCallHandlerWithRegistryAndVerbose(config config.ConstructHandlerConfig, registry LanguageRegistry, verbose bool) *LanguageCallHandler {
	return &LanguageCallHandler{
		config:              config,
		languageRegistry:    registry,
		verbose:             verbose,
		skipBackgroundCheck: false,
	}
}

// NewLanguageCallHandlerWithRegistryAndSkipBackgroundCheck создает обработчик с пропуском проверки background tasks
func NewLanguageCallHandlerWithRegistryAndSkipBackgroundCheck(config config.ConstructHandlerConfig, registry LanguageRegistry, verbose bool) *LanguageCallHandler {
	return &LanguageCallHandler{
		config:              config,
		languageRegistry:    registry,
		verbose:             verbose,
		skipBackgroundCheck: true,
	}
}

// CanHandle проверяет, может ли обработчик обработать токен
func (h *LanguageCallHandler) CanHandle(token lexer.Token) bool {
	// Проверяем, что это идентификатор (возможное имя языка) или зарезервированное ключевое слово языка
	return token.IsLanguageIdentifierOrCallToken()
}

// Handle обрабатывает вызов функции другого языка
func (h *LanguageCallHandler) Handle(ctx *common.ParseContext) (interface{}, error) {
	// Логика разбора: language.function(arguments)
	tokenStream := ctx.TokenStream

	// 1. Читаем язык
	languageToken := tokenStream.Consume()
	language := languageToken.Value

	// Отладочный вывод
	if h.verbose {
		fmt.Printf("DEBUG: LanguageCallHandler.Handle called with token type %d, value '%s'\n", languageToken.Type, languageToken.Value)
	}

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
		// Если после идентификатора идет =, то это присваивание, а не вызов функции
		if tokenStream.HasMore() {
			if h.verbose {
				fmt.Printf("DEBUG: LanguageCallHandler - current token after function name: %s (%s)\n", tokenStream.Current().Value, tokenStream.Current().Type)
			}
			if tokenStream.Current().Type == lexer.TokenAssign {
				if h.verbose {
					fmt.Printf("DEBUG: LanguageCallHandler - detected assignment, returning specific error\n")
				}
				return nil, fmt.Errorf("not a language call - assignment detected")
			}
		}
		return nil, fmt.Errorf("expected '(' after function name '%s'", functionName)
	}
	tokenStream.Consume() // Consuming '('

	// 6. Читаем аргументы
	arguments := make([]ast.Expression, 0)

	// Проверяем, есть ли аргументы
	if !tokenStream.HasMore() {
		// Сразу EOF после открывающей скобки
		return nil, fmt.Errorf("unexpected EOF after argument")
	}

	if tokenStream.Current().Type != lexer.TokenRightParen {
		// Есть хотя бы один аргумент
		for {

			if !tokenStream.HasMore() {
				if len(arguments) == 0 {
					// Если нет ни одного аргумента, но после открывающей скобки что-то было (например, lua.print(123))
					return nil, fmt.Errorf("unexpected EOF after argument")
				} else {
					// Если уже есть хотя бы один аргумент, но нет закрывающей скобки
					return nil, fmt.Errorf("unexpected EOF in arguments")
				}
			}

			// Читаем аргумент
			argToken := tokenStream.Current()
			if h.verbose {
				fmt.Printf("DEBUG: LanguageCallHandler - processing argument token: %s (%s) at pos %d\n", argToken.Value, argToken.Type, argToken.Position)
			}
			var arg ast.Expression

			switch argToken.Type {
			case lexer.TokenString:
				tokenStream.Consume()
				arg = &ast.StringLiteral{Value: argToken.Value, Raw: argToken.Value, Pos: tokenToPosition(argToken)}
			case lexer.TokenNumber:
				tokenStream.Consume()
				numValue, _ := strconv.ParseFloat(argToken.Value, 64)
				arg = &ast.NumberLiteral{Value: numValue, Pos: tokenToPosition(argToken)}
			case lexer.TokenMinus:
				// Handle unary minus for negative numbers - delegate to parseArgument
				arg, err = h.parseArgument(ctx)
				if err != nil {
					return nil, fmt.Errorf("failed to parse negative number argument: %v", err)
				}
			case lexer.TokenAt:
				// Обрабатываем @ как оператор размера битстринга
				tokenStream.Consume()
				// Следующий токен должен быть выражением (переменной)
				if !tokenStream.HasMore() {
					return nil, fmt.Errorf("expected expression after @")
				}
				// Парсим выражение после @
				sizeExpr, err := h.parseArgument(ctx)
				if err != nil {
					return nil, fmt.Errorf("failed to parse expression after @: %v", err)
				}
				// Создаем узел для получения размера
				arg = &ast.SizeExpression{
					ExprType:   "expression",
					Expression: sizeExpr,
					Pos:        tokenToPosition(argToken),
				}
			case lexer.TokenJS, lexer.TokenLua, lexer.TokenPython, lexer.TokenGo, lexer.TokenNode, lexer.TokenPy:
				// Обрабатываем языковые токены как идентификаторы
				if h.verbose {
					fmt.Printf("DEBUG: LanguageCallHandler - processing language token as identifier: %s (%s)\n", argToken.Value, argToken.Type)
				}
				// Используем общий метод parseArgument для обработки сложных выражений
				arg, err = h.parseArgument(ctx)
				if err != nil {
					return nil, fmt.Errorf("failed to parse argument: %v", err)
				}
			case lexer.TokenLBracket:
				// Парсим массив
				arrayHandler := NewArrayHandler(100, 0)
				arrayCtx := &common.ParseContext{
					TokenStream: tokenStream,
					Parser:      nil,
					Depth:       ctx.Depth + 1,
					MaxDepth:    ctx.MaxDepth,
					Guard:       ctx.Guard,
					LoopDepth:   ctx.LoopDepth,
					InputStream: ctx.InputStream,
				}
				arrayResult, err := arrayHandler.Handle(arrayCtx)
				if err != nil {
					return nil, fmt.Errorf("failed to parse array argument: %v", err)
				}
				if array, ok := arrayResult.(*ast.ArrayLiteral); ok {
					arg = array
				} else {
					return nil, fmt.Errorf("expected ArrayLiteral, got %T", arrayResult)
				}
			case lexer.TokenLBrace:
				// Парсим объект
				objectHandler := NewObjectHandler(100, 0)
				objectCtx := &common.ParseContext{
					TokenStream: tokenStream,
					Parser:      nil,
					Depth:       ctx.Depth + 1,
					MaxDepth:    ctx.MaxDepth,
					Guard:       ctx.Guard,
					LoopDepth:   ctx.LoopDepth,
					InputStream: ctx.InputStream,
				}
				objectResult, err := objectHandler.Handle(objectCtx)
				if err != nil {
					return nil, fmt.Errorf("failed to parse object argument: %v", err)
				}
				if object, ok := objectResult.(*ast.ObjectLiteral); ok {
					arg = object
				} else {
					return nil, fmt.Errorf("expected ObjectLiteral, got %T", objectResult)
				}
			case lexer.TokenIdentifier: // Language tokens are handled by IsLanguageToken() check in parseArgument
				// Используем общий метод parseArgument для обработки сложных выражений
				arg, err = h.parseArgument(ctx)
				if err != nil {
					return nil, fmt.Errorf("failed to parse argument: %v", err)
				}
			case lexer.TokenLeftParen:
				// Выражение в скобках - используем parseArgument для обработки
				arg, err = h.parseArgument(ctx)
				if err != nil {
					return nil, fmt.Errorf("failed to parse parenthesized argument: %v", err)
				}
			default:
				// Добавляем отладочный вывод для понимания, какие типы токенов не обрабатываются
				if h.verbose {
					fmt.Printf("DEBUG: LanguageCallHandler - unsupported argument type: %s, value: '%s', position: %d\n", argToken.Type, argToken.Value, argToken.Position)
				}
				return nil, fmt.Errorf("unsupported argument type: %s", argToken.Type)
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

	// 7. Создаем узел AST
	startPos := tokenToPosition(languageToken)

	node := &ast.LanguageCall{
		Language:  resolvedLanguage, // Используем разрешенное имя языка
		Function:  functionName,
		Arguments: arguments,
		Pos:       startPos,
	}

	// Проверяем, что после language call нет лишних токенов (кроме NEWLINE)
	// Если есть &, это должен обрабатывать LanguageCallStatementHandler
	// В режиме частичного парсинга пропускаем эту проверку
	if !ctx.PartialParsingMode && !h.skipBackgroundCheck {
		if tokenStream.HasMore() {
			nextToken := tokenStream.Current()
			if nextToken.Type == lexer.TokenAmpersand {
				// Если есть &, это background task, который должен обрабатывать LanguageCallStatementHandler
				return nil, fmt.Errorf("background task detected - should be handled by LanguageCallStatementHandler")
			}
			if nextToken.Type != lexer.TokenNewline {
				return nil, fmt.Errorf("unexpected token '%s' after language call", nextToken.Type)
			}
		}
	}

	return node, nil
}

// Config возвращает конфигурацию обработчика
func (h *LanguageCallHandler) Config() common.HandlerConfig {
	return common.HandlerConfig{
		IsEnabled: h.config.IsEnabled,
		Priority:  h.config.Priority,
		Name:      h.config.Name,
	}
}

// Name возвращает имя обработчика
func (h *LanguageCallHandler) Name() string {
	return h.config.Name
}

// parseNestedFunctionCall парсит вложенный вызов функции с точечной нотацией
func (h *LanguageCallHandler) parseNestedFunctionCall(ctx *common.ParseContext) (*ast.LanguageCall, error) {
	if h.verbose {
		fmt.Printf("DEBUG: parseNestedFunctionCall called\n")
	}
	tokenStream := ctx.TokenStream

	// Потребляем первый идентификатор (язык)
	firstToken := tokenStream.Consume()
	if h.verbose {
		fmt.Printf("DEBUG: parseNestedFunctionCall - first token: %s (%s)\n", firstToken.Value, firstToken.Type)
	}
	if !firstToken.IsLanguageIdentifierOrCallToken() {
		return nil, fmt.Errorf("expected language identifier for nested function call, got %s", firstToken.Type)
	}

	language := firstToken.LanguageTokenToString()

	// Обрабатываем остальные части пути функции (например, string.lower)
	var functionParts []string

	// Читаем точки и идентификаторы до открывающей скобки
	for tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenDot {
		// Потребляем точку
		tokenStream.Consume()

		// Проверяем, что после точки идет идентификатор
		if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenIdentifier {
			return nil, fmt.Errorf("expected identifier after dot in nested function call")
		}

		// Потребляем идентификатор
		idToken := tokenStream.Consume()
		functionParts = append(functionParts, idToken.Value)
	}

	if len(functionParts) == 0 {
		return nil, fmt.Errorf("no function parts found in nested function call")
	}

	// Собираем полное имя функции
	functionName := ""
	for i, part := range functionParts {
		if i > 0 {
			functionName += "."
		}
		functionName += part
	}

	// Проверяем и потребляем открывающую скобку
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenLeftParen {
		return nil, fmt.Errorf("expected '(' after function name '%s'", functionName)
	}
	tokenStream.Consume()

	// Читаем аргументы
	arguments := make([]ast.Expression, 0)

	if !tokenStream.HasMore() {
		return nil, fmt.Errorf("unexpected EOF after '('")
	}

	if tokenStream.Current().Type != lexer.TokenRightParen {
		// Есть хотя бы один аргумент
		for {
			if !tokenStream.HasMore() {
				return nil, fmt.Errorf("unexpected EOF in arguments")
			}

			// Читаем аргумент
			arg, err := h.parseArgument(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to parse argument: %v", err)
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

	// Проверяем закрывающую скобку
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenRightParen {
		return nil, fmt.Errorf("expected ')' after arguments")
	}
	tokenStream.Consume()

	// Создаем узел LanguageCall
	startPos := ast.Position{
		Line:   firstToken.Line,
		Column: firstToken.Column,
		Offset: firstToken.Position,
	}

	return &ast.LanguageCall{
		Language:  language,
		Function:  functionName,
		Arguments: arguments,
		Pos:       startPos,
	}, nil
}

// parseArgument парсит один аргумент функции
func (h *LanguageCallHandler) parseArgument(ctx *common.ParseContext) (ast.Expression, error) {
	tokenStream := ctx.TokenStream

	if !tokenStream.HasMore() {
		return nil, fmt.Errorf("unexpected EOF in argument")
	}

	token := tokenStream.Current()

	switch token.Type {
	case lexer.TokenString:
		tokenStream.Consume()
		return &ast.StringLiteral{Value: token.Value, Raw: token.Value, Pos: tokenToPosition(token)}, nil

	case lexer.TokenNumber:
		tokenStream.Consume()
		numValue, _ := strconv.ParseFloat(token.Value, 64)
		return &ast.NumberLiteral{Value: numValue, Pos: tokenToPosition(token)}, nil

	case lexer.TokenMinus:
		// Handle unary minus for negative numbers
		tokenStream.Consume() // consume the minus
		if !tokenStream.HasMore() {
			return nil, fmt.Errorf("unexpected EOF after unary minus")
		}
		// Parse the operand (should be a number)
		operandToken := tokenStream.Current()
		if operandToken.Type != lexer.TokenNumber {
			return nil, fmt.Errorf("expected number after unary minus, got %s", operandToken.Type)
		}
		tokenStream.Consume() // consume the number
		numValue, _ := strconv.ParseFloat(operandToken.Value, 64)
		numberLiteral := &ast.NumberLiteral{Value: numValue, Pos: tokenToPosition(operandToken)}
		// Create a unary expression for the negative number
		return ast.NewUnaryExpression("-", numberLiteral, tokenToPosition(token)), nil

	case lexer.TokenLBracket:
		// Парсим массив
		arrayHandler := NewArrayHandler(100, 0)
		arrayCtx := &common.ParseContext{
			TokenStream: tokenStream,
			Parser:      nil,
			Depth:       ctx.Depth + 1,
			MaxDepth:    ctx.MaxDepth,
			Guard:       ctx.Guard,
			LoopDepth:   ctx.LoopDepth,
			InputStream: ctx.InputStream,
		}
		arrayResult, err := arrayHandler.Handle(arrayCtx)
		if err != nil {
			return nil, fmt.Errorf("failed to parse array argument: %v", err)
		}
		if array, ok := arrayResult.(*ast.ArrayLiteral); ok {
			return array, nil
		} else {
			return nil, fmt.Errorf("expected ArrayLiteral, got %T", arrayResult)
		}

	case lexer.TokenLBrace:
		// Парсим объект
		objectHandler := NewObjectHandler(100, 0)
		objectCtx := &common.ParseContext{
			TokenStream: tokenStream,
			Parser:      nil,
			Depth:       ctx.Depth + 1,
			MaxDepth:    ctx.MaxDepth,
			Guard:       ctx.Guard,
			LoopDepth:   ctx.LoopDepth,
			InputStream: ctx.InputStream,
		}
		objectResult, err := objectHandler.Handle(objectCtx)
		if err != nil {
			return nil, fmt.Errorf("failed to parse object argument: %v", err)
		}
		if object, ok := objectResult.(*ast.ObjectLiteral); ok {
			return object, nil
		} else {
			return nil, fmt.Errorf("expected ObjectLiteral, got %T", objectResult)
		}

	case lexer.TokenJS, lexer.TokenLua, lexer.TokenPython, lexer.TokenGo, lexer.TokenNode, lexer.TokenPy:
		// Check for named argument with language tokens (identifier = expression)
		if tokenStream.HasMore() && tokenStream.Peek().Type == lexer.TokenAssign {
			// This is a named argument: name = value
			argName := token.LanguageTokenToString()
			argNameToken := token
			tokenStream.Consume() // consume language token
			tokenStream.Consume() // consume =

			// Parse the value expression
			valueExpr, err := h.parseArgument(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to parse value for named argument '%s': %v", argName, err)
			}

			// Create named argument
			return ast.NewNamedArgument(argName, valueExpr, tokenToPosition(argNameToken)), nil
		}

		// Обрабатываем языковые токены как идентификаторы
		if h.verbose {
			fmt.Printf("DEBUG: parseArgument - processing language token as identifier: %s (%s)\n", token.Value, token.Type)
		}
		fallthrough
	case lexer.TokenIdentifier:
		// Check for named argument (identifier = expression)
		if tokenStream.HasMore() && tokenStream.Peek().Type == lexer.TokenAssign {
			// This is a named argument: name = value
			argName := token.Value
			argNameToken := token
			tokenStream.Consume() // consume identifier
			tokenStream.Consume() // consume =

			// Parse the value expression
			valueExpr, err := h.parseArgument(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to parse value for named argument '%s': %v", argName, err)
			}

			// Create named argument
			return ast.NewNamedArgument(argName, valueExpr, tokenToPosition(argNameToken)), nil
		}

		// Проверяем, не является ли это сложным выражением с операторами
		if h.verbose {
			fmt.Printf("DEBUG: parseArgument - processing complex identifier: %s (%s)\n", token.Value, token.Type)
		}

		// Проверяем, не является ли это тернарным выражением или другим сложным выражением
		if tokenStream.HasMore() && (tokenStream.Peek().Type == lexer.TokenQuestion || isBinaryOperator(tokenStream.Peek().Type)) {
			// Это сложное выражение: identifier == value && other ? true_expr : false_expr
			if h.verbose {
				fmt.Printf("DEBUG: parseArgument - detected complex expression starting with identifier\n")
			}

			// Используем BinaryExpressionHandler для парсинга сложного выражения
			// Создаем левую часть выражения (простой идентификатор)
			ident := &ast.Identifier{Name: token.Value, Pos: tokenToPosition(token)}
			tokenStream.Consume() // потребляем идентификатор

			binaryHandler := NewBinaryExpressionHandler(config.ConstructHandlerConfig{})
			result, err := binaryHandler.ParseFullExpression(ctx, ident)
			if err != nil {
				return nil, fmt.Errorf("failed to parse complex expression argument: %v", err)
			}
			return result, nil
		}

		// Проверяем, не является ли это вложенным вызовом функции
		if tokenStream.Peek().Type == lexer.TokenDot {
			// Это может быть вложенный вызов функции или квалифицированная переменная с операторами
			if h.verbose {
				fmt.Printf("DEBUG: parseArgument - found DOT after '%s', checking for complex expression\n", token.Value)
			}

			// Клонируем поток, чтобы проанализировать структуру выражения
			clone := tokenStream.Clone()
			clone.Consume() // потребляем первый идентификатор (язык)

			// Следуем по цепочке DOT и идентификаторов
			dotCount := 0
			identifierCount := 0

			for clone.HasMore() && clone.Current().Type == lexer.TokenDot {
				clone.Consume() // потребляем DOT
				dotCount++

				if clone.HasMore() && clone.Current().Type == lexer.TokenIdentifier {
					clone.Consume() // потребляем идентификатор
					identifierCount++
				} else {
					break
				}
			}

			if h.verbose {
				fmt.Printf("DEBUG: parseArgument - found %d dots and %d identifiers\n", dotCount, identifierCount)
			}

			// Проверяем наличие операторов после квалифицированной переменной
			if clone.HasMore() && (clone.Current().Type == lexer.TokenQuestion ||
				clone.Current().Type == lexer.TokenColon ||
				isBinaryOperator(clone.Current().Type)) {
				if h.verbose {
					fmt.Printf("DEBUG: parseArgument - detected complex expression with qualified variable, using BinaryExpressionHandler\n")
				}

				// Сначала парсим квалифицированную переменную как левую часть выражения
				var leftExpr ast.Expression

				// Потребляем первый идентификатор (язык)
				languageToken := tokenStream.Consume()
				if h.verbose {
					fmt.Printf("DEBUG: parseArgument - consumed language token: %s\n", languageToken.Value)
				}

				// Потребляем DOT
				if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenDot {
					return nil, fmt.Errorf("expected DOT after language token")
				}
				tokenStream.Consume() // потребляем DOT
				if h.verbose {
					fmt.Printf("DEBUG: parseArgument - consumed DOT token\n")
				}

				// Потребляем второй идентификатор (переменная)
				if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenIdentifier {
					return nil, fmt.Errorf("expected identifier after DOT")
				}
				varToken := tokenStream.Consume()
				if h.verbose {
					fmt.Printf("DEBUG: parseArgument - consumed variable token: %s\n", varToken.Value)
				}

				// Создаем квалифицированный идентификатор
				language := languageToken.LanguageTokenToString()

				qualifiedIdentifier := ast.NewQualifiedIdentifier(languageToken, varToken, language, varToken.Value)
				leftExpr = ast.NewVariableRead(qualifiedIdentifier)

				// Теперь используем BinaryExpressionHandler для парсинга оставшейся части выражения
				binaryHandler := NewBinaryExpressionHandler(config.ConstructHandlerConfig{})
				expr, err := binaryHandler.ParseFullExpression(ctx, leftExpr)
				if err != nil {
					return nil, fmt.Errorf("failed to parse complex expression argument: %v", err)
				}
				return expr, nil
			}

			// Проверяем, является ли это вложенным вызовом функции или простой переменной
			// Вложенный вызов функции имеет как минимум 2 DOT и открывающую скобку в конце
			if dotCount >= 1 && clone.HasMore() && clone.Current().Type == lexer.TokenLeftParen {
				if h.verbose {
					fmt.Printf("DEBUG: parseArgument - detected nested function call with %d dots\n", dotCount)
				}
				// Это вложенный вызов функции
				nestedCall, err := h.parseNestedFunctionCall(ctx)
				if err != nil {
					return nil, fmt.Errorf("failed to parse nested function call: %v", err)
				}
				return nestedCall, nil
			} else if dotCount >= 1 {
				// Это квалифицированная переменная или цепочка доступа к полям (1 или более DOT, нет открывающей скобки)
				if h.verbose {
					fmt.Printf("DEBUG: parseArgument - treating as field access chain (%d dots, no parenthesis)\n", dotCount)
				}

				// Показываем текущую позицию токена перед вызовом FieldAccessHandler
				if h.verbose {
					if tokenStream.HasMore() {
						fmt.Printf("DEBUG: parseArgument - current token before FieldAccessHandler: %s (%s)\n", tokenStream.Current().Value, tokenStream.Current().Type)
					} else {
						fmt.Printf("DEBUG: parseArgument - no tokens before FieldAccessHandler\n")
					}
				}

				// Используем FieldAccessHandler для парсинга цепочки полей
				fieldAccessHandler := NewFieldAccessHandler(config.ConstructHandlerConfig{})
				fieldAccessCtx := &common.ParseContext{
					TokenStream: tokenStream,
					Parser:      nil,
					Depth:       ctx.Depth + 1,
					MaxDepth:    ctx.MaxDepth,
					Guard:       ctx.Guard,
					LoopDepth:   ctx.LoopDepth,
					InputStream: ctx.InputStream,
				}

				// FieldAccessHandler обработает всю цепочку начиная с текущей позиции
				fieldAccessResult, err := fieldAccessHandler.Handle(fieldAccessCtx)
				if err != nil {
					if h.verbose {
						fmt.Printf("DEBUG: parseArgument - FieldAccessHandler failed: %v\n", err)
					}
					return nil, fmt.Errorf("failed to parse field access chain: %v", err)
				}

				if h.verbose {
					fmt.Printf("DEBUG: parseArgument - FieldAccessHandler succeeded, result type: %T\n", fieldAccessResult)
				}

				// Проверяем, есть ли после FieldAccess индексный доступ (например, lua.data.features[1])
				if tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenLBracket {
					if h.verbose {
						fmt.Printf("DEBUG: parseArgument - detected index access after field access\n")
					}

					// Парсим индексное выражение вручную
					if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenLBracket {
						return nil, fmt.Errorf("expected '[' for index access")
					}
					tokenStream.Consume() // потребляем '['

					// Создаем контекст для парсинга индекса
					indexCtx := &common.ParseContext{
						TokenStream: tokenStream,
						Parser:      nil,
						Depth:       ctx.Depth + 1,
						MaxDepth:    ctx.MaxDepth,
						Guard:       ctx.Guard,
						LoopDepth:   ctx.LoopDepth,
						InputStream: ctx.InputStream,
					}

					// Парсим индекс
					indexArg, err := h.parseArgument(indexCtx)
					if err != nil {
						return nil, fmt.Errorf("failed to parse index argument: %v", err)
					}

					// Проверяем закрывающую скобку
					if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenRBracket {
						return nil, fmt.Errorf("expected ']' after index")
					}
					tokenStream.Consume() // потребляем ']'

					// Получаем позицию из FieldAccess результата
					var pos ast.Position
					if fieldAccess, ok := fieldAccessResult.(*ast.FieldAccess); ok {
						pos = fieldAccess.Pos
					} else {
						// Запасной вариант - используем позицию из токена
						pos = ast.Position{
							Line:   1,
							Column: 1,
							Offset: 0,
						}
					}

					// Создаем IndexExpression с нашим FieldAccess как объект
					indexExpr := ast.NewIndexExpression(
						fieldAccessResult.(ast.Expression),
						indexArg,
						pos,
					)

					if h.verbose {
						fmt.Printf("DEBUG: parseArgument - created IndexExpression with FieldAccess object\n")
					}
					return indexExpr, nil
				}

				if fieldAccess, ok := fieldAccessResult.(*ast.FieldAccess); ok {
					return fieldAccess, nil
				} else {
					return nil, fmt.Errorf("expected FieldAccess, got %T", fieldAccessResult)
				}
			} else {
				return nil, fmt.Errorf("unsupported qualified expression structure: %d dots, no operators, no parenthesis", dotCount)
			}
		} else {
			// Простой идентификатор
			tokenStream.Consume()
			return ast.NewIdentifier(token, token.Value), nil
		}

	case lexer.TokenLeftParen:
		// Выражение в скобках - используем AssignmentHandler для парсинга (поддерживает ternary)
		if h.verbose {
			fmt.Printf("DEBUG: parseArgument - parsing parenthesized expression\n")
		}

		// Используем AssignmentHandler для парсинга выражения в скобках, так как он поддерживает ternary
		assignmentHandler := NewAssignmentHandler(100, 0)

		// Создаем контекст для парсинга выражения в скобках
		parenCtx := &common.ParseContext{
			TokenStream: tokenStream,
			Parser:      ctx.Parser,
			Depth:       ctx.Depth + 1,
			MaxDepth:    ctx.MaxDepth,
			Guard:       ctx.Guard,
			LoopDepth:   ctx.LoopDepth,
			InputStream: ctx.InputStream,
		}

		// Парсим выражение в скобках используя parseComplexExpression из AssignmentHandler
		expr, err := assignmentHandler.parseComplexExpression(parenCtx)
		if err != nil {
			return nil, fmt.Errorf("failed to parse expression in parentheses: %v", err)
		}

		return expr, nil

	default:
		return nil, fmt.Errorf("unsupported argument type: %s", token.Type)
	}
}
