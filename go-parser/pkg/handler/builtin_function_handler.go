package handler

import (
	"fmt"
	"strings"
	"go-parser/pkg/ast"
	"go-parser/pkg/common"
	"go-parser/pkg/config"
	"go-parser/pkg/lexer"
	"go-parser/pkg/stream"
)

// BuiltinFunctionHandler - обработчик для вызовов builtin функций (без квалификатора языка)
type BuiltinFunctionHandler struct {
	config  config.ConstructHandlerConfig
	verbose bool
}

// NewBuiltinFunctionHandler создает новый обработчик для builtin функций
func NewBuiltinFunctionHandler(config config.ConstructHandlerConfig) *BuiltinFunctionHandler {
	return NewBuiltinFunctionHandlerWithVerbose(config, false)
}

// NewBuiltinFunctionHandlerWithVerbose создает новый обработчик для builtin функций с поддержкой verbose режима
func NewBuiltinFunctionHandlerWithVerbose(config config.ConstructHandlerConfig, verbose bool) *BuiltinFunctionHandler {
	return &BuiltinFunctionHandler{
		config:  config,
		verbose: verbose,
	}
}

// CanHandle проверяет, может ли обработчик обработать токен
func (h *BuiltinFunctionHandler) CanHandle(token lexer.Token) bool {
	// Обрабатываем только обычные идентификаторы (не языковые токены)
	return token.Type == lexer.TokenIdentifier
}

// Handle обрабатывает вызов builtin функции
func (h *BuiltinFunctionHandler) Handle(ctx *common.ParseContext) (interface{}, error) {
	tokenStream := ctx.TokenStream

	if h.verbose {
		fmt.Printf("DEBUG: BuiltinFunctionHandler.Handle called with token: %s (%s)\n", tokenStream.Current().Value, tokenStream.Current().Type)
	}

	// 1. Читаем имя функции
	functionToken := tokenStream.Consume()
	functionName := functionToken.Value

	if h.verbose {
		fmt.Printf("DEBUG: BuiltinFunctionHandler - function name: %s\n", functionName)
	}

	// 2. Проверяем и потребляем открывающую скобку
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenLeftParen {
		// Если после идентификатора нет скобки, это не вызов функции - позволяем другим обработчикам попробовать
		return nil, nil
	}
	tokenStream.Consume() // Consuming '('

	if h.verbose {
		fmt.Printf("DEBUG: BuiltinFunctionHandler - consumed '('\n")
	}

	// 3. Читаем аргументы
	arguments := make([]ast.Expression, 0)

	if !tokenStream.HasMore() {
		return nil, newErrorWithPos(ctx.TokenStream, "unexpected EOF after '('")
	}

	if tokenStream.Current().Type != lexer.TokenRightParen {
		// Есть хотя бы один аргумент
		for {
			if !tokenStream.HasMore() {
				return nil, newErrorWithPos(ctx.TokenStream, "unexpected EOF in arguments")
			}

			// Читаем аргумент
			arg, err := h.parseArgument(ctx)
			if err != nil {
				return nil, newErrorWithPos(ctx.TokenStream, "failed to parse argument: %v", err)
			}

			arguments = append(arguments, arg)

			// Пропускаем newline токены
			for tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenNewline {
				tokenStream.Consume() // newline
			}

			// Проверяем разделитель или конец
			if !tokenStream.HasMore() {
				return nil, newErrorWithPos(ctx.TokenStream, "unexpected EOF after argument")
			}

			nextToken := tokenStream.Current()

			if nextToken.Type == lexer.TokenComma {
				tokenStream.Consume() // Consuming comma

				// Пропускаем newline токены после запятой
				for tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenNewline {
					tokenStream.Consume() // newline
				}

				// После запятой должен быть аргумент
				if !tokenStream.HasMore() {
					return nil, newErrorWithPos(ctx.TokenStream, "unexpected EOF after comma")
				}
				if tokenStream.Current().Type == lexer.TokenRightParen {
					return nil, newErrorWithPos(ctx.TokenStream, "unexpected ')' after comma")
				}
			} else if nextToken.Type == lexer.TokenRightParen {
				break
			} else {
				return nil, newErrorWithTokenPos(nextToken, "expected ',' or ')' after argument, got %s", nextToken.Type)
			}
		}
	}

	// 4. Проверяем закрывающую скобку
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenRightParen {
		return nil, newErrorWithPos(ctx.TokenStream, "expected ')' after arguments")
	}
	tokenStream.Consume() // Consuming ')'

	if h.verbose {
		fmt.Printf("DEBUG: BuiltinFunctionHandler - consumed ')', parsed %d arguments\n", len(arguments))
	}

	// 5. Создаем узел AST для BuiltinFunctionCall
	startPos := ast.Position{
		Line:   functionToken.Line,
		Column: functionToken.Column,
		Offset: functionToken.Position,
	}

	node := ast.NewBuiltinFunctionCall(functionName, arguments, startPos)

	if h.verbose {
		fmt.Printf("DEBUG: BuiltinFunctionHandler - created BuiltinFunctionCall: %s\n", functionName)
	}

	return node, nil
}

// extractFunctionNameFromFieldAccess извлекает полное имя функции из FieldAccess
func (h *BuiltinFunctionHandler) extractFunctionNameFromFieldAccess(fieldAccess *ast.FieldAccess) string {
	var parts []string

	// Рекурсивно собираем все части имени
	current := fieldAccess
	for current != nil {
		parts = append([]string{current.Field}, parts...) // добавляем в начало

		if fa, ok := current.Object.(*ast.FieldAccess); ok {
			current = fa
		} else if ident, ok := current.Object.(*ast.Identifier); ok {
			// Базовый идентификатор
			parts = append([]string{ident.Name}, parts...)
			break
		} else {
			// Неизвестный тип объекта
			break
		}
	}

	return strings.Join(parts, ".")
}

// isLiteralToken проверяет, является ли токен началом литерала или выражения
func (h *BuiltinFunctionHandler) isLiteralToken(tokenType lexer.TokenType) bool {
	switch tokenType {
	case lexer.TokenNumber, lexer.TokenString, lexer.TokenTrue, lexer.TokenFalse, lexer.TokenNil,
		lexer.TokenIdentifier, lexer.TokenLua, lexer.TokenPython, lexer.TokenPy, lexer.TokenGo, lexer.TokenNode, lexer.TokenJS,
		lexer.TokenLBracket, lexer.TokenLBrace, lexer.TokenDoubleLeftAngle, lexer.TokenLeftParen, lexer.TokenAt, lexer.TokenMinus:
		return true
	default:
		return false
	}
}

// parseArgument парсит один аргумент функции
func (h *BuiltinFunctionHandler) parseArgument(ctx *common.ParseContext) (ast.Expression, error) {
	tokenStream := ctx.TokenStream

	if !tokenStream.HasMore() {
		return nil, newErrorWithPos(ctx.TokenStream, "unexpected EOF in argument")
	}

	token := tokenStream.Current()

	if h.verbose {
		fmt.Printf("DEBUG: parseArgument - parsing token: %s (%s)\n", token.Value, token.Type)
	}

	// Check if this looks like a complex expression (literal followed by binary operator)
	// or starts with parentheses (potential ternary operator)
	if (h.isLiteralToken(token.Type) && tokenStream.HasMore() && isBinaryOperator(tokenStream.Peek().Type)) ||
		token.Type == lexer.TokenLeftParen {
		// This is a complex expression - use UnifiedExpressionParser
		if h.verbose {
			fmt.Printf("DEBUG: parseArgument - detected complex expression, using UnifiedExpressionParser\n")
		}
		exprParser := NewUnifiedExpressionParser(h.verbose)
		expr, err := exprParser.ParseExpression(ctx)
		if err != nil {
			return nil, newErrorWithPos(ctx.TokenStream, "failed to parse expression argument: %v", err)
		}
		if h.verbose {
			fmt.Printf("DEBUG: parseArgument - complex expression parsed successfully\n")
		}
		return expr, nil
	}

	switch token.Type {
	case lexer.TokenString:
		tokenStream.Consume()
		return &ast.StringLiteral{Value: token.Value, Raw: token.Value, Pos: tokenToPosition(token)}, nil

	case lexer.TokenNumber:
		tokenStream.Consume()
		numValue, err := parseNumber(token.Value)
		if err != nil {
			return nil, newErrorWithTokenPos(token, "invalid number format: %s", token.Value)
		}
		return createNumberLiteral(token, numValue), nil

	case lexer.TokenIdentifier, lexer.TokenLua, lexer.TokenPython, lexer.TokenPy, lexer.TokenGo, lexer.TokenNode, lexer.TokenJS:
		// First check if this is a function call (has opening paren)
		if tokenStream.HasMore() && tokenStream.Peek().Type == lexer.TokenLeftParen {
			// This is a function call - use BinaryExpressionHandler to parse it
			binaryHandler := NewBinaryExpressionHandler(config.ConstructHandlerConfig{})
			expr, err := binaryHandler.parseOperand(ctx)
			if err != nil {
				return nil, newErrorWithPos(ctx.TokenStream, "failed to parse function call argument: %v", err)
			}
			return expr, nil
		}

		// Check if this is a field access or language call
		if tokenStream.HasMore() && tokenStream.Peek().Type == lexer.TokenDot {
			// Check if this is a language call (lua.something(...))
			// Peek ahead to see the pattern
			savedPos := tokenStream.Position()
			tokenStream.Consume() // consume the dot
			if tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenIdentifier {
				tokenStream.Consume() // consume identifier
				if tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenLeftParen {
					// This is a language call: lua.something(...)
					tokenStream.SetPosition(savedPos) // restore position
					languageCallHandler := NewLanguageCallHandler(config.ConstructHandlerConfig{})
					languageCallCtx := &common.ParseContext{
						TokenStream: tokenStream,
						Depth:       ctx.Depth + 1,
						MaxDepth:    ctx.MaxDepth,
					}
					languageCallResult, err := languageCallHandler.Handle(languageCallCtx)
					if err != nil {
						return nil, newErrorWithPos(tokenStream, "failed to parse language call argument: %v", err)
					}
					expr, ok := languageCallResult.(ast.Expression)
					if !ok {
						return nil, newErrorWithPos(tokenStream, "expected expression from language call handler, got %T", languageCallResult)
					}
					return expr, nil
				} else {
					// Restore position and try field access
					if h.verbose {
						fmt.Printf("DEBUG: parseArgument - not a language call, restoring position for field access\n")
					}
					tokenStream.SetPosition(savedPos)
				}
			} else {
				if h.verbose {
					fmt.Printf("DEBUG: parseArgument - not an identifier after dot, restoring position\n")
				}
				tokenStream.SetPosition(savedPos)
			}

			// This is a field access expression like lua.v or lua.string.format(...)
			fieldAccessHandler := NewFieldAccessHandler(config.ConstructHandlerConfig{})
			fieldAccessCtx := &common.ParseContext{
				TokenStream: tokenStream,
				Depth:       ctx.Depth + 1,
				MaxDepth:    ctx.MaxDepth,
			}
			fieldAccessResult, err := fieldAccessHandler.Handle(fieldAccessCtx)
			if err != nil {
				return nil, newErrorWithPos(tokenStream, "failed to parse field access argument: %v", err)
			}

			// Check if this is followed by a function call (fieldAccess(...))
			if tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenLeftParen {
				// This is a language call: fieldAccess(...)
				if h.verbose {
					fmt.Printf("DEBUG: parseArgument - field access followed by '(', treating as language call\n")
				}
				// We need to reconstruct the language call from the field access
				fieldAccess, ok := fieldAccessResult.(*ast.FieldAccess)
				if !ok {
					return nil, newErrorWithPos(tokenStream, "expected FieldAccess for language call, got %T", fieldAccessResult)
				}

				// Extract the full function name from the field access
				functionName := h.extractFunctionNameFromFieldAccess(fieldAccess)
				if h.verbose {
					fmt.Printf("DEBUG: parseArgument - extracted function name: %s\n", functionName)
				}

				// Split into language and function parts
				parts := strings.SplitN(functionName, ".", 2)
				if len(parts) != 2 {
					return nil, newErrorWithPos(tokenStream, "invalid language call format: %s", functionName)
				}
				language := parts[0]
				funcName := parts[1]

				// Now parse the arguments
				tokenStream.Consume() // consume '('
				arguments := make([]ast.Expression, 0)

				for tokenStream.HasMore() && tokenStream.Current().Type != lexer.TokenRightParen {
					if tokenStream.Current().Type == lexer.TokenComma {
						tokenStream.Consume()
						continue
					}

					arg, err := h.parseArgument(ctx)
					if err != nil {
						return nil, newErrorWithPos(tokenStream, "failed to parse language call argument: %v", err)
					}
					arguments = append(arguments, arg)
				}

				if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenRightParen {
					return nil, newErrorWithPos(tokenStream, "expected ')' after language call arguments")
				}
				tokenStream.Consume() // consume ')'

				// Create language call
				languageCall := &ast.LanguageCall{
					Language:  language,
					Function:  funcName,
					Arguments: arguments,
					Pos:       fieldAccess.Pos,
				}
				return languageCall, nil
			}

			expr, ok := fieldAccessResult.(ast.Expression)
			if !ok {
				return nil, newErrorWithPos(tokenStream, "expected expression from field access handler, got %T", fieldAccessResult)
			}

			// Check if this expression is followed by an index expression
			if tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenLBracket {
				indexExpr, err := h.parseIndexExpression(expr, tokenStream)
				if err != nil {
					return nil, newErrorWithPos(tokenStream, "failed to parse index expression: %v", err)
				}
				expr = indexExpr
			}

			return expr, nil
		}

		// Simple identifier
		tokenStream.Consume()
		if token.Type == lexer.TokenIdentifier {
			return ast.NewIdentifier(token, token.Value), nil
		} else {
			// For language tokens, create qualified identifier
			ident := ast.NewIdentifier(token, token.Value)
			ident.Language = token.LanguageTokenToString()
			ident.Qualified = true
			return ident, nil
		}

	case lexer.TokenLBracket:
		// Array literal
		arrayHandler := NewArrayHandler(0, 0)
		arrayResult, err := arrayHandler.Handle(ctx)
		if err != nil {
			return nil, newErrorWithPos(ctx.TokenStream, "failed to parse array argument: %v", err)
		}
		if array, ok := arrayResult.(*ast.ArrayLiteral); ok {
			return array, nil
		} else {
			return nil, newErrorWithPos(ctx.TokenStream, "expected ArrayLiteral, got %T", arrayResult)
		}

	case lexer.TokenLBrace:
		// Object literal
		objectHandler := NewObjectHandler(0, 0)
		objectResult, err := objectHandler.Handle(ctx)
		if err != nil {
			return nil, newErrorWithPos(ctx.TokenStream, "failed to parse object argument: %v", err)
		}
		if object, ok := objectResult.(*ast.ObjectLiteral); ok {
			return object, nil
		} else {
			return nil, newErrorWithPos(ctx.TokenStream, "expected ObjectLiteral, got %T", objectResult)
		}

	case lexer.TokenTrue:
		// Boolean true literal
		tokenStream.Consume()
		return &ast.BooleanLiteral{Value: true, Pos: tokenToPosition(token)}, nil

	case lexer.TokenFalse:
		// Boolean false literal
		tokenStream.Consume()
		return &ast.BooleanLiteral{Value: false, Pos: tokenToPosition(token)}, nil

	case lexer.TokenNil:
		// Nil literal
		tokenStream.Consume()
		return &ast.NilLiteral{Pos: tokenToPosition(token)}, nil

	case lexer.TokenMinus:
		// Handle unary minus for negative numbers
		tokenStream.Consume() // consume the minus
		if !tokenStream.HasMore() {
			return nil, newErrorWithPos(ctx.TokenStream, "unexpected EOF after unary minus")
		}
		// Parse the operand (should be a number)
		operandToken := tokenStream.Current()
		if operandToken.Type != lexer.TokenNumber {
			return nil, newErrorWithTokenPos(operandToken, "expected number after unary minus, got %s", operandToken.Type)
		}
		tokenStream.Consume() // consume the number
		numValue, parseErr := parseNumber(operandToken.Value)
		if parseErr != nil {
			return nil, newErrorWithTokenPos(operandToken, "invalid number format: %s", operandToken.Value)
		}
		numberLiteral := createNumberLiteral(operandToken, numValue)
		// Create a unary expression for the negative number
		return ast.NewUnaryExpression("-", numberLiteral, tokenToPosition(token)), nil


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
			return nil, newErrorWithPos(ctx.TokenStream, "failed to parse bitstring argument: %v", err)
		}
		bitstringExpr, ok := bitstringResult.(*ast.BitstringExpression)
		if !ok {
			return nil, newErrorWithPos(ctx.TokenStream, "expected bitstring expression, got %T", bitstringResult)
		}
		return bitstringExpr, nil

	case lexer.TokenAt:
		// Size operator - используем UnifiedExpressionParser
		if h.verbose {
			fmt.Printf("DEBUG: parseArgument - parsing @ expression\n")
		}
		exprParser := NewUnifiedExpressionParser(h.verbose)
		expr, err := exprParser.ParseExpression(ctx)
		if err != nil {
			return nil, newErrorWithPos(ctx.TokenStream, "failed to parse @ expression argument: %v", err)
		}
		if h.verbose {
			fmt.Printf("DEBUG: parseArgument - @ expression parsed successfully\n")
		}
		return expr, nil

	default:
		return nil, newErrorWithTokenPos(token, "unsupported argument type: %s", token.Type)
	}
}

// parseIndexExpression парсит индексное выражение вида object[index]
func (h *BuiltinFunctionHandler) parseIndexExpression(object ast.Expression, tokenStream stream.TokenStream) (ast.Expression, error) {
	// Проверяем наличие открывающей квадратной скобки
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenLBracket {
		return nil, fmt.Errorf("expected '[' for index expression")
	}

	// Потребляем открывающую скобку
	tokenStream.Consume()

	// Парсим индексное выражение
	index, err := h.parseArgument(&common.ParseContext{
		TokenStream: tokenStream,
		Depth:       0,
		MaxDepth:    100,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to parse index expression: %v", err)
	}

	// Проверяем и потребляем закрывающую скобку
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenRBracket {
		return nil, fmt.Errorf("expected ']' after index expression")
	}
	tokenStream.Consume()

	// Создаем узел индексного доступа
	result := &ast.IndexExpression{
		Object: object,
		Index:  index,
		Pos:    object.Position(),
	}
	return result, nil
}

// Config возвращает конфигурацию обработчика
func (h *BuiltinFunctionHandler) Config() common.HandlerConfig {
	return common.HandlerConfig{
		IsEnabled: h.config.IsEnabled,
		Priority:  h.config.Priority,
		Name:      h.config.Name,
	}
}

// Name возвращает имя обработчика
func (h *BuiltinFunctionHandler) Name() string {
	return h.config.Name
}
