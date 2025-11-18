package handler

import (
	"fmt"

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


	// 2. Проверяем и разрешаем алиас через Language Registry
	resolvedLanguage, err := h.languageRegistry.ResolveAlias(language)
	if err != nil {
		return nil, newErrorWithTokenPos(languageToken, "unsupported language '%s': %v", language, err)
	}

	// 3. Проверяем и потребляем DOT
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenDot {
		return nil, newErrorWithTokenPos(languageToken, "expected DOT after language '%s'", language)
	}
	tokenStream.Consume() // Consuming dot

	// 4. Читаем имя функции (может содержать точки, например, math.sqrt)
	functionParts := []string{}

	// Читаем первый идентификатор функции
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenIdentifier {
		return nil, newErrorWithPos(ctx.TokenStream, "expected function name after DOT")
	}
	functionToken := tokenStream.Consume()
	functionParts = append(functionParts, functionToken.Value)

	// Читаем дополнительные DOT и идентификаторы
	for tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenDot {
		tokenStream.Consume() // Consuming dot

		if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenIdentifier {
			return nil, newErrorWithPos(ctx.TokenStream, "expected function name after dot")
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
	if !tokenStream.HasMore() || (tokenStream.Current().Type != lexer.TokenLeftParen && tokenStream.Current().Type != lexer.TokenLParen) {
		// Если после идентификатора идет =, то это присваивание, а не вызов функции
		if tokenStream.HasMore() {
			if h.verbose {
				fmt.Printf("DEBUG: LanguageCallHandler - current token after function name: %s (%s)\n", tokenStream.Current().Value, tokenStream.Current().Type)
			}
			if tokenStream.Current().Type == lexer.TokenAssign || tokenStream.Current().Type == lexer.TokenColonEquals {
				if h.verbose {
					fmt.Printf("DEBUG: LanguageCallHandler - detected assignment, returning specific error\n")
				}
				return nil, fmt.Errorf("not a language call - assignment detected")
			}
		}
		return nil, newErrorWithTokenPos(functionToken, "expected '(' after function name '%s'", functionName)
	}
	tokenStream.Consume() // Consuming '('

	// 6. Читаем аргументы
	arguments := make([]ast.Expression, 0)

	// Проверяем, есть ли аргументы
	if !tokenStream.HasMore() {
		// Сразу EOF после открывающей скобки
		return nil, newErrorWithPos(ctx.TokenStream, "unexpected EOF after argument")
	}

	if tokenStream.Current().Type != lexer.TokenRightParen && tokenStream.Current().Type != lexer.TokenRParen {
		// Есть хотя бы один аргумент
		for {

			if !tokenStream.HasMore() {
				if len(arguments) == 0 {
					// Если нет ни одного аргумента, но после открывающей скобки что-то было (например, lua.print(123))
					return nil, newErrorWithPos(ctx.TokenStream, "unexpected EOF after argument")
				} else {
					// Если уже есть хотя бы один аргумент, но нет закрывающей скобки
					return nil, newErrorWithPos(ctx.TokenStream, "unexpected EOF in arguments")
				}
			}

			// Читаем аргумент
			argToken := tokenStream.Current()
			if h.verbose {
				fmt.Printf("DEBUG: LanguageCallHandler - processing argument token: %s (%s) at pos %d\n", argToken.Value, argToken.Type, argToken.Position)
			}
			var arg ast.Expression

			switch argToken.Type {
			case lexer.TokenJS, lexer.TokenLua, lexer.TokenPython, lexer.TokenGo, lexer.TokenNode, lexer.TokenPy:
				// Language token - use parseArgument to handle language calls and field access
				if h.verbose {
					fmt.Printf("DEBUG: LanguageCallHandler - parsing language token argument: %s (%s)\n", argToken.Value, argToken.Type)
				}
				arg, err = h.parseArgument(ctx)
				if err != nil {
					return nil, newErrorWithPos(ctx.TokenStream, "failed to parse language token argument: %v", err)
				}
				if h.verbose {
					fmt.Printf("DEBUG: LanguageCallHandler - parsed language token argument, result type: %T, current token: %s (%s)\n", arg, tokenStream.Current().Value, tokenStream.Current().Type)
				}
				// Check if this is followed by an index access (e.g., lua.variable[0])
				if tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenLBracket {
					if h.verbose {
						fmt.Printf("DEBUG: LanguageCallHandler - detected index access after language token argument\n")
					}
					// Parse index access: consume '[', parse index expression, consume ']'
					tokenStream.Consume() // consume '['

					// Parse index expression
					index, err := h.parseArgument(ctx)
					if err != nil {
						return nil, newErrorWithPos(ctx.TokenStream, "failed to parse index expression: %v", err)
					}

					// Consume ']'
					if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenRBracket {
						return nil, newErrorWithPos(ctx.TokenStream, "expected ']' after index expression")
					}
					tokenStream.Consume() // consume ']'

					// Create index expression
					arg = &ast.IndexExpression{
						Object: arg,
						Index:  index,
						Pos:    arg.Position(),
					}
				}
			case lexer.TokenIdentifier:
				// Check if this is a builtin function call (identifier followed by '(')
				if tokenStream.HasMore() && tokenStream.Peek().Type == lexer.TokenLeftParen {
					// This might be a builtin function call
					builtinHandler := NewBuiltinFunctionHandler(config.ConstructHandlerConfig{})
					if builtinHandler.CanHandle(argToken) {
						// It's a builtin function, handle it
						result, err := builtinHandler.Handle(ctx)
						if err != nil {
							return nil, newErrorWithPos(ctx.TokenStream, "failed to parse builtin function call: %v", err)
						}
						if builtinCall, ok := result.(*ast.BuiltinFunctionCall); ok {
							arg = builtinCall
						} else {
							return nil, newErrorWithPos(ctx.TokenStream, "expected BuiltinFunctionCall, got %T", result)
						}
					} else {
						// Not a builtin function, but still might be part of complex expression
						// Check if this identifier is part of a complex expression (e.g., identifier ? ... : ...)
						if tokenStream.HasMore() && (isBinaryOperator(tokenStream.Peek().Type) || tokenStream.Peek().Type == lexer.TokenQuestion) {
							if h.verbose {
								fmt.Printf("DEBUG: LanguageCallHandler - detected complex expression starting with identifier\n")
							}
							// Use parseArgument to handle complex expressions
							var err error
							arg, err = h.parseArgument(ctx)
							if err != nil {
								return nil, newErrorWithPos(ctx.TokenStream, "failed to parse complex expression argument: %v", err)
							}
						} else {
							// Regular identifier
							tokenStream.Consume()
							arg = ast.NewVariableRead(ast.NewIdentifier(argToken, argToken.Value))
						}
					}
				} else {
					// Check if this identifier is part of a complex expression
					if tokenStream.HasMore() && (isBinaryOperator(tokenStream.Peek().Type) || tokenStream.Peek().Type == lexer.TokenQuestion) {
						if h.verbose {
							fmt.Printf("DEBUG: LanguageCallHandler - detected complex expression starting with identifier\n")
						}
						// Use parseArgument to handle complex expressions
						var err error
						arg, err = h.parseArgument(ctx)
						if err != nil {
							return nil, newErrorWithPos(ctx.TokenStream, "failed to parse complex expression argument: %v", err)
						}
					} else {
						// Regular identifier
						tokenStream.Consume()
						arg = ast.NewVariableRead(ast.NewIdentifier(argToken, argToken.Value))
					}
				}
			case lexer.TokenString:
				tokenStream.Consume()
				arg = &ast.StringLiteral{Value: argToken.Value, Raw: argToken.Value, Pos: tokenToPosition(argToken)}
			case lexer.TokenNumber:
				// Check if this number is part of a complex expression
				if tokenStream.HasMore() && isBinaryOperator(tokenStream.Peek().Type) {
					if h.verbose {
						fmt.Printf("DEBUG: LanguageCallHandler - detected complex expression starting with number\n")
					}
					// Use parseArgument to handle complex expressions
					var err error
					arg, err = h.parseArgument(ctx)
					if err != nil {
						return nil, newErrorWithPos(ctx.TokenStream, "failed to parse complex expression argument: %v", err)
					}
				} else {
					// Simple number literal
					tokenStream.Consume()
					numValue, err := parseNumber(argToken.Value)
					if err != nil {
						return nil, newErrorWithTokenPos(argToken, "invalid number format: %s", argToken.Value)
					}
					arg = createNumberLiteral(argToken, numValue)
				}
			case lexer.TokenMinus:
				// Handle unary minus for negative numbers - delegate to parseArgument
				arg, err = h.parseArgument(ctx)
				if err != nil {
					return nil, newErrorWithPos(ctx.TokenStream, "failed to parse negative number argument: %v", err)
				}
			case lexer.TokenAt:
				// Обрабатываем @ как оператор размера битстринга
				tokenStream.Consume()
				// Следующий токен должен быть выражением или открывающей скобкой
				if !tokenStream.HasMore() {
					return nil, newErrorWithTokenPos(argToken, "expected expression after @")
				}

				var sizeExpr ast.Expression
				var err error

				// Проверяем, начинается ли выражение со скобки
				if tokenStream.Current().Type == lexer.TokenLeftParen {
					// Обрабатываем @(expression) - специальный случай для nested выражений
					tokenStream.Consume() // потребляем '('

					// Если следующий токен - <<, это bitstring
					if tokenStream.Current().Type == lexer.TokenDoubleLeftAngle {
						// Используем bitstring handler для парсинга bitstring
						bitstringHandler := NewBitstringHandler(config.ConstructHandlerConfig{})
						result, bsErr := bitstringHandler.Handle(ctx)
						if bsErr != nil {
							return nil, newErrorWithPos(ctx.TokenStream, "failed to parse bitstring in @ expression: %v", bsErr)
						}
						if bs, ok := result.(*ast.BitstringExpression); ok {
							sizeExpr = bs
						} else {
							return nil, newErrorWithPos(ctx.TokenStream, "expected BitstringExpression, got %T", result)
						}
					} else {
						// Для других выражений парсим до закрывающей скобки
						sizeExpr, err = h.parseParenthesizedSizeExpression(ctx)
						if err != nil {
							return nil, newErrorWithPos(ctx.TokenStream, "failed to parse expression in @ parentheses: %v", err)
						}
					}

					// Ожидаем закрывающую скобку
					if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenRightParen {
						return nil, newErrorWithPos(ctx.TokenStream, "expected ')' after @ expression")
					}
					tokenStream.Consume() // потребляем ')'
				} else {
					// Обычное выражение без скобок
					sizeExpr, err = h.parseArgument(ctx)
					if err != nil {
						return nil, newErrorWithPos(ctx.TokenStream, "failed to parse expression after @: %v", err)
					}
				}

				// Создаем узел для получения размера
				arg = &ast.SizeExpression{
					ExprType:   "expression",
					Expression: sizeExpr,
					Pos:        tokenToPosition(argToken),
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
					return nil, newErrorWithPos(ctx.TokenStream, "failed to parse array argument: %v", err)
				}
				if array, ok := arrayResult.(*ast.ArrayLiteral); ok {
					arg = array
				} else {
					return nil, newErrorWithPos(ctx.TokenStream, "expected ArrayLiteral, got %T", arrayResult)
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
					return nil, newErrorWithPos(ctx.TokenStream, "failed to parse object argument: %v", err)
				}
				if object, ok := objectResult.(*ast.ObjectLiteral); ok {
					arg = object
				} else {
					return nil, newErrorWithPos(ctx.TokenStream, "expected ObjectLiteral, got %T", objectResult)
				}
			case lexer.TokenLeftParen:
				// Выражение в скобках - используем parseArgument для обработки
				arg, err = h.parseArgument(ctx)
				if err != nil {
					return nil, newErrorWithPos(ctx.TokenStream, "failed to parse parenthesized argument: %v", err)
				}
			case lexer.TokenDoubleLeftAngle:
				// Битстринг как аргумент - используем parseArgument для обработки
				arg, err = h.parseArgument(ctx)
				if err != nil {
					return nil, newErrorWithPos(ctx.TokenStream, "failed to parse bitstring argument: %v", err)
				}
			default:
				// Добавляем отладочный вывод для понимания, какие типы токенов не обрабатываются
				if h.verbose {
					fmt.Printf("DEBUG: LanguageCallHandler - unsupported argument type: %s, value: '%s', position: %d\n", argToken.Type, argToken.Value, argToken.Position)
				}
				return nil, newErrorWithTokenPos(argToken, "unsupported argument type: %s", argToken.Type)
			}

			arguments = append(arguments, arg)

			// Проверяем разделитель или конец
			if !tokenStream.HasMore() {
				return nil, newErrorWithPos(ctx.TokenStream, "unexpected EOF after argument")
			}

			nextToken := tokenStream.Current()

			if nextToken.Type == lexer.TokenComma {
				tokenStream.Consume() // Consuming comma
				// После запятой должен быть аргумент
				if !tokenStream.HasMore() {
					return nil, newErrorWithPos(ctx.TokenStream, "unexpected EOF after comma")
				}
				if tokenStream.Current().Type == lexer.TokenRightParen {
					return nil, newErrorWithPos(ctx.TokenStream, "unexpected ')' after comma")
				}
			} else if nextToken.Type == lexer.TokenRightParen || nextToken.Type == lexer.TokenRParen {
				break
			} else {
				// Для ошибки о недостающей закрывающей скобки возвращаем правильное сообщение об ошибке
				return nil, newErrorWithTokenPos(nextToken, "expected ',' or ')' after argument, got %s", nextToken.Type)
			}
		}
	}

	// 7. Проверяем закрывающую скобку
	if !tokenStream.HasMore() || (tokenStream.Current().Type != lexer.TokenRightParen && tokenStream.Current().Type != lexer.TokenRParen) {
		return nil, newErrorWithPos(ctx.TokenStream, "expected ')' after arguments")
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

	// Проверяем, что после language call нет лишних токенов (кроме NEWLINE и токенов начинающих новые statement)
	// Если есть &, это должен обрабатывать LanguageCallStatementHandler
	// В режиме частичного парсинга пропускаем эту проверку
	if !ctx.PartialParsingMode && !h.skipBackgroundCheck {
		if tokenStream.HasMore() {
			nextToken := tokenStream.Current()
			if nextToken.Type == lexer.TokenAmpersand {
				// Если есть &, это background task, который должен обрабатывать LanguageCallStatementHandler
				return nil, fmt.Errorf("background task detected - should be handled by LanguageCallStatementHandler")
			}
			// Разрешаем токены, которые могут начинать новые statement или продолжать выражение
			if nextToken.Type != lexer.TokenNewline &&
				!nextToken.IsLanguageIdentifierOrCallToken() &&
				nextToken.Type != lexer.TokenIdentifier &&
				nextToken.Type != lexer.TokenIf &&
				nextToken.Type != lexer.TokenFor &&
				nextToken.Type != lexer.TokenWhile &&
				nextToken.Type != lexer.TokenMatch &&
				nextToken.Type != lexer.TokenBreak &&
				nextToken.Type != lexer.TokenContinue &&
				nextToken.Type != lexer.TokenImport &&
				!isBinaryOperator(nextToken.Type) &&
				nextToken.Type != lexer.TokenQuestion &&
				nextToken.Type != lexer.TokenColon &&
				!isUnaryOperator(nextToken.Type) {
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
	if !tokenStream.HasMore() || (tokenStream.Current().Type != lexer.TokenLeftParen && tokenStream.Current().Type != lexer.TokenLParen) {
		return nil, fmt.Errorf("expected '(' after function name '%s'", functionName)
	}
	tokenStream.Consume()

	// Читаем аргументы
	arguments := make([]ast.Expression, 0)

	if !tokenStream.HasMore() {
		return nil, fmt.Errorf("unexpected EOF after '('")
	}

	if tokenStream.Current().Type != lexer.TokenRightParen && tokenStream.Current().Type != lexer.TokenRParen {
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
			fmt.Printf("DEBUG: LanguageCallHandler - after parsing arg, nextToken: %s (%s) at pos %d\n", nextToken.Value, nextToken.Type, nextToken.Position)

			if nextToken.Type == lexer.TokenComma {
				tokenStream.Consume() // Consuming comma
				// После запятой должен быть аргумент
				if !tokenStream.HasMore() {
					return nil, fmt.Errorf("unexpected EOF after comma")
				}
				if tokenStream.Current().Type == lexer.TokenRightParen {
					return nil, fmt.Errorf("unexpected ')' after comma")
				}
			} else if nextToken.Type == lexer.TokenRightParen || nextToken.Type == lexer.TokenRParen {
				break
			} else {
				return nil, fmt.Errorf("expected ',' or ')' after argument, got %s", nextToken.Type)
			}
		}
	}

	// Проверяем закрывающую скобку
	if !tokenStream.HasMore() || (tokenStream.Current().Type != lexer.TokenRightParen && tokenStream.Current().Type != lexer.TokenRParen) {
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
		// Check if this number is part of a complex expression
		if tokenStream.HasMore() && isBinaryOperator(tokenStream.Peek().Type) {
			if h.verbose {
				fmt.Printf("DEBUG: parseArgument - detected complex expression starting with number\n")
			}
			// This is a complex expression: number * identifier
			// Create the left part (number literal)
			numValue, err := parseNumber(token.Value)
			if err != nil {
				return nil, fmt.Errorf("invalid number format: %s", token.Value)
			}
			leftExpr := createNumberLiteral(token, numValue)
			tokenStream.Consume() // consume the number

			// Use BinaryExpressionHandler to parse the rest
			binaryHandler := NewBinaryExpressionHandler(config.ConstructHandlerConfig{ConstructType: common.ConstructBinaryExpression})
			result, err := binaryHandler.ParseFullExpression(ctx, leftExpr)
			if err != nil {
				return nil, fmt.Errorf("failed to parse complex expression argument: %v", err)
			}
			return result, nil
		}
		// Simple number literal
		tokenStream.Consume()
		numValue, err := parseNumber(token.Value)
		if err != nil {
			return nil, fmt.Errorf("invalid number format: %s", token.Value)
		}
		return createNumberLiteral(token, numValue), nil

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
		numValue, err := parseNumber(operandToken.Value)
		if err != nil {
			return nil, fmt.Errorf("invalid number format: %s", operandToken.Value)
		}
		numberLiteral := createNumberLiteral(operandToken, numValue)
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
		// Check for named argument with language tokens (identifier = expression or := expression)
		if tokenStream.HasMore() && (tokenStream.Peek().Type == lexer.TokenAssign || tokenStream.Peek().Type == lexer.TokenColonEquals) {
			// This is a named argument: name = value or name := value
			argName := token.LanguageTokenToString()
			argNameToken := token
			tokenStream.Consume() // consume language token
			tokenStream.Consume() // consume = or :=

			// Parse the value expression
			valueExpr, err := h.parseArgument(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to parse value for named argument '%s': %v", argName, err)
			}

			// Create named argument
			return ast.NewNamedArgument(argName, valueExpr, tokenToPosition(argNameToken)), nil
		}

		// Для language tokens, проверяем на field access или language call
		if tokenStream.HasMore() && tokenStream.Peek().Type == lexer.TokenDot {
			if h.verbose {
				fmt.Printf("DEBUG: parseArgument - language token followed by DOT, checking for language call or field access\n")
			}
			// Check if this is a language call by looking ahead for '(' after the dotted chain
			savedPos := tokenStream.Position()

			// Skip the language token and dots/identifiers to find if there's a '('
			tokenStream.Consume() // consume language token
			foundLeftParen := false

			// Skip through dots and identifiers
			for tokenStream.HasMore() {
				if tokenStream.Current().Type == lexer.TokenDot {
					tokenStream.Consume() // consume dot
					if tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenIdentifier {
						tokenStream.Consume() // consume identifier
					} else {
						break
					}
				} else if tokenStream.Current().Type == lexer.TokenLeftParen {
					foundLeftParen = true
					break
				} else {
					break
				}
			}

			tokenStream.SetPosition(savedPos) // restore position

			if foundLeftParen {
				if h.verbose {
					fmt.Printf("DEBUG: parseArgument - detected language call with dotted function name, calling LanguageCallHandler\n")
				}
				// This is a language call: language.something.something(...)
				languageCallHandler := NewLanguageCallHandler(config.ConstructHandlerConfig{})
				languageCallCtx := &common.ParseContext{
					TokenStream:        tokenStream,
					Depth:              ctx.Depth + 1,
					MaxDepth:           ctx.MaxDepth,
					PartialParsingMode: true, // We're parsing an expression inside arguments
				}
				languageCallResult, err := languageCallHandler.Handle(languageCallCtx)
				if err != nil {
					if h.verbose {
						fmt.Printf("DEBUG: parseArgument - LanguageCallHandler returned error: %v\n", err)
					}
					return nil, fmt.Errorf("failed to parse language call argument: %v", err)
				}
				if expr, ok := languageCallResult.(ast.Expression); ok {
					if h.verbose {
						fmt.Printf("DEBUG: parseArgument - language call parsed successfully, type: %T\n", expr)
					}
					return expr, nil
				} else {
					return nil, fmt.Errorf("expected expression from language call handler, got %T", languageCallResult)
				}
			} else {
				if h.verbose {
					fmt.Printf("DEBUG: parseArgument - no LEFT_PAREN found, treating as field access\n")
				}
				// Not a language call, fall through to field access handler
			}
		}

		// Language tokens represent field access
		fieldAccessHandler := NewFieldAccessHandler(config.ConstructHandlerConfig{})
		fieldAccessCtx := &common.ParseContext{
			TokenStream: tokenStream,
			Depth:       ctx.Depth + 1,
			MaxDepth:    ctx.MaxDepth,
		}
		fieldAccessResult, err := fieldAccessHandler.Handle(fieldAccessCtx)
		if err != nil {
			return nil, fmt.Errorf("failed to parse field access argument: %v", err)
		}
		if expr, ok := fieldAccessResult.(ast.Expression); ok {
			return expr, nil
		} else {
			return nil, fmt.Errorf("expected expression from field access handler, got %T", fieldAccessResult)
		}
	case lexer.TokenIdentifier:
		// Check for named argument (identifier = expression or := expression)
		if tokenStream.HasMore() && (tokenStream.Peek().Type == lexer.TokenAssign || tokenStream.Peek().Type == lexer.TokenColonEquals) {
			// This is a named argument: name = value or name := value
			argName := token.Value
			argNameToken := token
			tokenStream.Consume() // consume identifier
			tokenStream.Consume() // consume = or :=

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
				fmt.Printf("DEBUG: parseArgument - detected complex expression starting with identifier, peek token: %s (%s)\n", tokenStream.Peek().Value, tokenStream.Peek().Type)
			}

			// Создаем левую часть выражения (VariableRead из идентификатора)
			ident := ast.NewVariableRead(ast.NewIdentifier(token, token.Value))
			tokenStream.Consume() // потребляем идентификатор

			if h.verbose {
				fmt.Printf("DEBUG: parseArgument - after consuming identifier, current token: %s (%s)\n", tokenStream.Current().Value, tokenStream.Current().Type)
			}

			// Проверяем, является ли это тернарным выражением
			if tokenStream.Current().Type == lexer.TokenQuestion {
				// Используем BinaryExpressionHandler для парсинга тернарного выражения
				binaryHandler := NewBinaryExpressionHandler(config.ConstructHandlerConfig{ConstructType: common.ConstructBinaryExpression})
				result, err := binaryHandler.ParseTernaryExpression(ctx, ident)
				if err != nil {
					return nil, fmt.Errorf("failed to parse ternary expression argument: %v", err)
				}
				return result, nil
			} else {
				// Используем BinaryExpressionHandler для парсинга бинарного выражения
				binaryHandler := NewBinaryExpressionHandler(config.ConstructHandlerConfig{ConstructType: common.ConstructBinaryExpression})
				result, err := binaryHandler.ParseFullExpression(ctx, ident)
				if err != nil {
					return nil, fmt.Errorf("failed to parse binary expression argument: %v", err)
				}
				return result, nil
			}
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
				binaryHandler := NewBinaryExpressionHandler(config.ConstructHandlerConfig{ConstructType: common.ConstructBinaryExpression})
				expr, err := binaryHandler.ParseFullExpression(ctx, leftExpr)
				if err != nil {
					return nil, fmt.Errorf("failed to parse complex expression argument: %v", err)
				}
				return expr, nil
			}

			// Проверяем, является ли это вложенным вызовом функции или простой переменной
			// Вложенный вызов функции имеет как минимум 2 DOT и открывающую скобку в конце
			if dotCount >= 1 && clone.HasMore() && (clone.Current().Type == lexer.TokenLeftParen || clone.Current().Type == lexer.TokenLParen) {
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
		// Выражение в скобках - парсим полное выражение включая ternary операторы

		// Используем AssignmentHandler для парсинга всего выражения начиная с '('
		assignmentHandler := NewAssignmentHandlerWithVerbose(100, 0, h.verbose)

		// Парсим выражение используя parseComplexExpression из AssignmentHandler
		expr, err := assignmentHandler.parseComplexExpression(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to parse expression in parentheses: %v", err)
		}

		return expr, nil

	case lexer.TokenDoubleLeftAngle:
		// Битстринг как аргумент
		bitstringHandler := NewBitstringHandler(config.ConstructHandlerConfig{})
		bitstringCtx := &common.ParseContext{
			TokenStream: tokenStream,
			Parser:      nil,
			Depth:       ctx.Depth + 1,
			MaxDepth:    ctx.MaxDepth,
			Guard:       ctx.Guard,
			LoopDepth:   ctx.LoopDepth,
			InputStream: ctx.InputStream,
		}
		bitstringResult, err := bitstringHandler.Handle(bitstringCtx)
		if err != nil {
			return nil, fmt.Errorf("failed to parse bitstring argument: %v", err)
		}
		bitstringExpr, ok := bitstringResult.(*ast.BitstringExpression)
		if !ok {
			return nil, fmt.Errorf("expected bitstring expression, got %T", bitstringResult)
		}
		return bitstringExpr, nil

	default:
		return nil, fmt.Errorf("unsupported argument type: %s", token.Type)
	}
}

// parseParenthesizedSizeExpression парсит выражение в скобках для оператора @,
// останавливаясь на закрывающей скобке без её потребления
func (h *LanguageCallHandler) parseParenthesizedSizeExpression(ctx *common.ParseContext) (ast.Expression, error) {
	// Используем AssignmentHandler для парсинга комплексных выражений
	assignmentHandler := NewAssignmentHandlerWithVerbose(96, 4, h.verbose)
	expr, err := assignmentHandler.parseComplexExpression(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to parse expression in @ parentheses: %v", err)
	}

	return expr, nil
}
