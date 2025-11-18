package handler

import (
	"fmt"

	"go-parser/pkg/ast"
	"go-parser/pkg/common"
	"go-parser/pkg/config"
	"go-parser/pkg/lexer"
	"go-parser/pkg/stream"
)

// WhileLoopHandler - обработчик while циклов
type WhileLoopHandler struct {
	config  config.ConstructHandlerConfig
	verbose bool
}

// NewWhileLoopHandler создает новый обработчик while циклов
func NewWhileLoopHandler(config config.ConstructHandlerConfig) *WhileLoopHandler {
	return NewWhileLoopHandlerWithVerbose(config, false)
}

// NewWhileLoopHandlerWithVerbose создает новый обработчик while циклов с поддержкой verbose режима
func NewWhileLoopHandlerWithVerbose(config config.ConstructHandlerConfig, verbose bool) *WhileLoopHandler {
	return &WhileLoopHandler{
		config:  config,
		verbose: verbose,
	}
}

// CanHandle проверяет, может ли обработчик обработать токен
func (h *WhileLoopHandler) CanHandle(token lexer.Token) bool {
	// Обрабатываем токен 'while'
	return token.Type == lexer.TokenWhile
}

// Handle обрабатывает while цикл
func (h *WhileLoopHandler) Handle(ctx *common.ParseContext) (interface{}, error) {
	// Проверяем защиту от рекурсии
	if err := ctx.Guard.Enter(); err != nil {
		return nil, err
	}
	defer ctx.Guard.Exit()

	tokenStream := ctx.TokenStream

	// Увеличиваем глубину циклов для контекстной валидации break/continue
	ctx.LoopDepth++
	defer func() {
		ctx.LoopDepth--
	}()

	// 1. Проверяем токен 'while'
	whileToken := tokenStream.Current()
	if whileToken.Type != lexer.TokenWhile {
		return nil, newErrorWithTokenPos(whileToken, "expected 'while', got %s", whileToken.Type)
	}

	// 2. Проверяем структуру перед потреблением токенов
	// While-цикл: while <condition> { ... }
	if !tokenStream.HasMore() {
		return nil, newErrorWithTokenPos(whileToken, "expected condition after 'while' - no more tokens")
	}

	// Потребляем 'while'
	tokenStream.Consume()

	// 3. Проверяем наличие токена '(' (необязательный для синтаксиса без скобок)
	var lParenToken lexer.Token
	if tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenLeftParen {
		lParenToken = tokenStream.Consume()
	} else {
		// Если скобок нет, создаем пустой токен для совместимости
		lParenToken = lexer.Token{
			Type:     lexer.TokenLeftParen,
			Value:    "(",
			Position: whileToken.Position,
			Line:     whileToken.Line,
			Column:   whileToken.Column,
		}
	}

	// 4. Читаем условие цикла
	condition, err := h.parseCondition(ctx, tokenStream)
	if err != nil {
		return nil, fmt.Errorf("failed to parse condition: %v", err)
	}

	// 5. Проверяем и потребляем токен ')' (только если была открывающая скобка)
	var rParenToken lexer.Token
	if lParenToken.Position != whileToken.Position {
		// Была открывающая скобка (позиция отличается от whileToken), значит должна быть и закрывающая
		if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenRightParen {
			return nil, newErrorWithPos(tokenStream, "expected ')' after condition")
		}
		rParenToken = tokenStream.Consume()
	} else {
		// Не было открывающей скобки, создаем пустой токен для совместимости
		rParenToken = lexer.Token{
			Type:     lexer.TokenRightParen,
			Value:    ")",
			Position: whileToken.Position,
			Line:     whileToken.Line,
			Column:   whileToken.Column,
		}
	}

	// 4. Проверяем и потребляем токен '{'
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenLBrace {
		return nil, newErrorWithPos(tokenStream, "expected '{' after condition")
	}
	lBraceToken := tokenStream.Consume()

	// 5. Читаем тело цикла
	body, err := h.parseLoopBody(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to parse loop body: %v", err)
	}

	// 6. Проверяем и потребляем токен '}'
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenRBrace {
		return nil, newErrorWithPos(tokenStream, "expected '}' after loop body")
	}
	rBraceToken := tokenStream.Consume()

	// 7. Создаем узел AST
	blockStatement := ast.NewBlockStatement(lBraceToken, rBraceToken, body)
	loopNode := ast.NewWhileStatement(whileToken, lParenToken, rParenToken, condition, blockStatement)

	return loopNode, nil
}

// parseCondition парсит условие цикла
func (h *WhileLoopHandler) parseCondition(ctx *common.ParseContext, tokenStream stream.TokenStream) (ast.Expression, error) {
	if !tokenStream.HasMore() {
		return nil, newErrorWithPos(tokenStream, "unexpected EOF in condition")
	}

	// Поддерживаем простые условия и условия в скобках:
	// - Булевы литералы (true, false)
	// - Числовые литералы
	// - Идентификаторы
	// - Вызовы функций
	// - Условия в скобках: (condition)

	token := tokenStream.Current()
	switch token.Type {
	case lexer.TokenAt:
		// Unary operator @ (size operator) - first parse unary expression, then binary operators
		unaryHandler := NewUnaryExpressionHandler(config.ConstructHandlerConfig{})
		unaryResult, err := unaryHandler.Handle(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to parse unary @ expression: %v", err)
		}
		if unaryExpr, ok := unaryResult.(*ast.UnaryExpression); ok {
			// Now parse binary operators after the unary expression
			binaryHandler := NewBinaryExpressionHandlerWithVerbose(config.ConstructHandlerConfig{}, h.verbose)
			return binaryHandler.ParseFullExpression(ctx, unaryExpr)
		}
		return nil, fmt.Errorf("expected UnaryExpression from @, got %T", unaryResult)

	case lexer.TokenTrue:
		tokenStream.Consume()
		return &ast.BooleanLiteral{Value: true, Pos: matchHandlerTokenToPosition(token)}, nil

	case lexer.TokenFalse:
		tokenStream.Consume()
		return &ast.BooleanLiteral{Value: false, Pos: matchHandlerTokenToPosition(token)}, nil

	case lexer.TokenLeftParen:
		// Условие в скобках - потребляем скобку и читаем условие внутри
		tokenStream.Consume() // потребляем '('

		if !tokenStream.HasMore() {
			return nil, fmt.Errorf("expected condition after '('")
		}

		// Используем бинарный обработчик для разбора выражения внутри скобок
		binaryHandler := NewBinaryExpressionHandler(config.ConstructHandlerConfig{})

		// Сначала парсим первый операнд внутри скобок
		if !tokenStream.HasMore() {
			return nil, fmt.Errorf("expected expression after '('")
		}

		firstToken := tokenStream.Current()
		var leftExpr ast.Expression

		switch firstToken.Type {
		case lexer.TokenIdentifier, lexer.TokenLua, lexer.TokenPython, lexer.TokenPy, lexer.TokenGo, lexer.TokenNode, lexer.TokenJS:
			if tokenStream.Peek().Type == lexer.TokenLeftParen {
				// Это вызов функции
				languageCallHandler := NewLanguageCallHandler(config.ConstructHandlerConfig{})
				result, err := languageCallHandler.Handle(ctx)
				if err != nil {
					return nil, fmt.Errorf("failed to parse function call in parentheses: %v", err)
				}
				if call, ok := result.(*ast.LanguageCall); ok {
					leftExpr = call
				} else {
					return nil, fmt.Errorf("expected LanguageCall, got %T", result)
				}
			} else if tokenStream.Peek().Type == lexer.TokenDot {
				// Это доступ к полю через точку (например, lua.x)
				fieldAccessHandler := NewFieldAccessHandler(config.ConstructHandlerConfig{})
				result, err := fieldAccessHandler.Handle(ctx)
				if err != nil {
					return nil, fmt.Errorf("failed to parse field access in parentheses: %v", err)
				}
				if fieldAccess, ok := result.(*ast.FieldAccess); ok {
					leftExpr = fieldAccess
				} else {
					return nil, fmt.Errorf("expected FieldAccess, got %T", result)
				}
			} else {
				// Простой идентификатор
				tokenStream.Consume()
				leftExpr = ast.NewIdentifier(firstToken, firstToken.Value)
			}

		case lexer.TokenTrue, lexer.TokenFalse:
			tokenStream.Consume()
			leftExpr = &ast.BooleanLiteral{
				Value: firstToken.Type == lexer.TokenTrue,
				Pos: ast.Position{
					Line:   firstToken.Line,
					Column: firstToken.Column,
					Offset: firstToken.Position,
				},
			}

		default:
			return nil, fmt.Errorf("unsupported expression type inside parentheses: %s", firstToken.Type)
		}

		// Проверяем, есть ли бинарный оператор
		if tokenStream.HasMore() && isBinaryOperator(tokenStream.Current().Type) {
			// Парсим бинарное выражение
			expr, err := binaryHandler.ParseFullExpression(ctx, leftExpr)
			if err != nil {
				return nil, fmt.Errorf("failed to parse binary expression: %v", err)
			}

			// Проверяем и потребляем закрывающую скобку
			if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenRightParen {
				return nil, fmt.Errorf("expected ')' after binary expression, got %s (%s)",
					tokenStream.Current().Value, tokenStream.Current().Type)
			}
			tokenStream.Consume() // потребляем ')'

			return expr, nil
		}

		// Проверяем и потребляем закрывающую скобку (только если было условие в скобках)
		if tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenRightParen {
			tokenStream.Consume() // потребляем ')'
		}

		return leftExpr, nil

	case lexer.TokenIdentifier, lexer.TokenLua, lexer.TokenPython, lexer.TokenPy, lexer.TokenGo, lexer.TokenNode, lexer.TokenJS:
		// Это может быть простой идентификатор, вызов функции, доступ к полю или бинарное выражение
		if tokenStream.Peek().Type == lexer.TokenLeftParen {
			// Это вызов функции, делегируем LanguageCallHandler
			languageCallHandler := NewLanguageCallHandler(config.ConstructHandlerConfig{})
			result, err := languageCallHandler.Handle(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to parse function call in condition: %v", err)
			}

			if call, ok := result.(*ast.LanguageCall); ok {
				return call, nil
			}
			return nil, fmt.Errorf("expected language call in condition, got %T", result)
		} else if tokenStream.Peek().Type == lexer.TokenDot {
			// Это доступ к полю через точку (например, lua.x)
			fieldAccessHandler := NewFieldAccessHandler(config.ConstructHandlerConfig{})
			result, err := fieldAccessHandler.Handle(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to parse field access in condition: %v", err)
			}

			if fieldAccess, ok := result.(*ast.FieldAccess); ok {
				// После field access проверяем, есть ли бинарный оператор
				if tokenStream.HasMore() && isBinaryOperator(tokenStream.Current().Type) {
					// Это бинарное выражение с field access как левый операнд
					binaryHandler := NewBinaryExpressionHandler(config.ConstructHandlerConfig{})
					return binaryHandler.ParseFullExpression(ctx, fieldAccess)
				}
				return fieldAccess, nil
			}
			return nil, fmt.Errorf("expected FieldAccess in condition, got %T", result)
		} else if isBinaryOperator(tokenStream.Peek().Type) {
			// Это бинарное выражение
			binaryHandler := NewBinaryExpressionHandler(config.ConstructHandlerConfig{})
			left := ast.NewIdentifier(token, token.Value)
			tokenStream.Consume() // потребляем левый операнд
			return binaryHandler.ParseFullExpression(ctx, left)
		} else {
			// Простой идентификатор
			tokenStream.Consume()
			return ast.NewIdentifier(token, token.Value), nil
		}

	case lexer.TokenString:
		// Строковой литерал
		tokenStream.Consume()
		return &ast.StringLiteral{
			Value: token.Value,
			Pos: ast.Position{
				Line:   token.Line,
				Column: token.Column,
				Offset: token.Position,
			},
		}, nil

	case lexer.TokenNumber:
		// Числовой литерал
		tokenStream.Consume()
		value, err := parseNumber(token.Value)
		if err != nil {
			return nil, fmt.Errorf("invalid number format: %s", token.Value)
		}
		return createNumberLiteral(token, value), nil

	default:
		return nil, fmt.Errorf("unsupported condition type: %s", token.Type)
	}
}

// parseLoopBody парсит тело цикла
func (h *WhileLoopHandler) parseLoopBody(ctx *common.ParseContext) ([]ast.Statement, error) {
	tokenStream := ctx.TokenStream
	body := make([]ast.Statement, 0)

	// Пропускаем пробелы после '{'
	for tokenStream.HasMore() {
		current := tokenStream.Current()

		if current.Type == lexer.TokenEOF {
			break
		}

		// Если встречаем '}', заканчиваем тело цикла
		if current.Type == lexer.TokenRBrace {
			break
		}

		// Пропускаем пробелы и новые строки
		if current.Type == lexer.TokenNewline {
			tokenStream.Consume()
			continue
		}

		// Если встречаем 'while', это вложенный while цикл
		if current.Type == lexer.TokenWhile {
			nestedWhileHandler := NewWhileLoopHandler(config.ConstructHandlerConfig{})
			result, err := nestedWhileHandler.Handle(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to parse nested while loop: %v", err)
			}
			if nestedLoop, ok := result.(*ast.WhileStatement); ok {
				body = append(body, nestedLoop)
				continue
			}
		}

		// Если встречаем 'for', это вложенный for цикл
		if current.Type == lexer.TokenFor {
			// Сначала пробуем NumericForLoopHandler
			numericForHandler := NewNumericForLoopHandler(config.ConstructHandlerConfig{})
			result, err := numericForHandler.Handle(ctx)
			if err == nil {
				if nestedLoop, ok := result.(*ast.NumericForLoopStatement); ok {
					body = append(body, nestedLoop)
					continue
				}
			}

			// Если не получилось, пробуем ForInLoopHandler
			forInHandler := NewForInLoopHandler(config.ConstructHandlerConfig{})
			result, err = forInHandler.Handle(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to parse nested for loop: %v", err)
			}
			if nestedLoop, ok := result.(*ast.ForInLoopStatement); ok {
				body = append(body, nestedLoop)
				continue
			}
		}

		// Если встречаем 'match', это match statement
		if current.Type == lexer.TokenMatch {
			matchHandler := NewMatchHandler(config.ConstructHandlerConfig{})
			result, err := matchHandler.Handle(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to parse match statement: %v", err)
			}
			if matchStmt, ok := result.(*ast.MatchStatement); ok {
				body = append(body, matchStmt)
				continue
			}
		}

		// Если встречаем 'if', это if statement
		if current.Type == lexer.TokenIf {
			ifHandler := NewIfHandler(config.ConstructHandlerConfig{})
			result, err := ifHandler.Handle(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to parse if statement: %v", err)
			}
			if ifStmt, ok := result.(*ast.IfStatement); ok {
				body = append(body, ifStmt)
				continue
			}
		}

		// Если встречаем 'break', это break оператор
		if current.Type == lexer.TokenBreak {
			tokenStream.Consume()
			breakStatement := ast.NewBreakStatement(current)
			body = append(body, breakStatement)
			continue
		}

		// Если встречаем 'continue', это continue оператор
		if current.Type == lexer.TokenContinue {
			tokenStream.Consume()
			continueStatement := ast.NewContinueStatement(current)
			body = append(body, continueStatement)
			continue
		}

		// Для всех остальных случаев используем обработчики для разбора statements
		// Это позволит обрабатывать все типы statements включая JavaScript присваивания и вызовы функций
		if current.Type == lexer.TokenIdentifier || current.Type == lexer.TokenLua || current.Type == lexer.TokenPython ||
			current.Type == lexer.TokenPy || current.Type == lexer.TokenGo || current.Type == lexer.TokenNode ||
			current.Type == lexer.TokenJS || current.Type == lexer.TokenString || current.Type == lexer.TokenNumber {

			// Сохраняем текущую позицию
			currentPos := tokenStream.Position()

			// Сначала пробуем AssignmentHandler для присваиваний
			assignmentHandler := NewAssignmentHandler(80, 4)
			result, err := assignmentHandler.Handle(ctx)
			if err == nil {
				if assignment, ok := result.(*ast.VariableAssignment); ok {
					body = append(body, assignment)
					continue
				}
			} else {
				if h.verbose {
					fmt.Printf("DEBUG: WhileLoopHandler - AssignmentHandler failed: %v\n", err)
				}
			}

			// Если AssignmentHandler не сработал, восстанавливаем позицию
			tokenStream.SetPosition(currentPos)

			// Пробуем LanguageCallHandler для вызовов функций
			languageCallHandler := NewLanguageCallHandler(config.ConstructHandlerConfig{})
			result, err = languageCallHandler.Handle(ctx)
			if err == nil {
				if call, ok := result.(*ast.LanguageCall); ok {
					body = append(body, call)
					continue
				}
			} else {
				if h.verbose {
					fmt.Printf("DEBUG: WhileLoopHandler - LanguageCallHandler failed: %v\n", err)
				}
			}

			// Если LanguageCallHandler не сработал, восстанавливаем позицию
			tokenStream.SetPosition(currentPos)
		}

		// Проверяем, не является ли это bitstring pattern assignment (<<...>> = ...)
		if current.Type == lexer.TokenDoubleLeftAngle {
			bitstringPatternHandler := NewBitstringPatternAssignmentHandler(110, 4) // priority and order like in UnifiedParser
			result, err := bitstringPatternHandler.Handle(ctx)
			if err == nil {
				if bitstringPattern, ok := result.(*ast.BitstringPatternAssignment); ok {
					body = append(body, bitstringPattern)
					continue
				}
			} else {
				if h.verbose {
					fmt.Printf("DEBUG parseLoopBody: bitstring pattern parsing failed: %v\n", err)
				}
			}
		}

		// Проверяем, не является ли это вызовом builtin функции
		if current.Type == lexer.TokenIdentifier && tokenStream.Peek().Type == lexer.TokenLeftParen {
			if h.verbose {
				fmt.Printf("DEBUG parseLoopBody: found identifier followed by '(', trying builtin function\n")
			}
			// Это может быть builtin функция типа print(...)
			builtinHandler := NewBuiltinFunctionHandlerWithVerbose(config.ConstructHandlerConfig{}, h.verbose)
			result, err := builtinHandler.Handle(ctx)
			if err == nil {
				if h.verbose {
					fmt.Printf("DEBUG parseLoopBody: builtin function parsing succeeded\n")
				}
				if builtinCall, ok := result.(*ast.BuiltinFunctionCall); ok {
					body = append(body, builtinCall)
					continue
				}
			} else {
				if h.verbose {
					fmt.Printf("DEBUG parseLoopBody: builtin function parsing failed: %v\n", err)
				}
			}
		}

		// Если не смогли распарсить токен, пропускаем его чтобы избежать бесконечного цикла
		if h.verbose {
			fmt.Printf("DEBUG: WhileLoopHandler - skipping unparsable token: %s (%s)\n", current.Value, current.Type)
		}
		tokenStream.Consume()
	}

	return body, nil
}

// Config возвращает конфигурацию обработчика
func (h *WhileLoopHandler) Config() common.HandlerConfig {
	return common.HandlerConfig{
		IsEnabled: h.config.IsEnabled,
		Priority:  h.config.Priority,
		Name:      h.config.Name,
	}
}

// Name возвращает имя обработчика
func (h *WhileLoopHandler) Name() string {
	return h.config.Name
}
