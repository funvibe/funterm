package handler

import (
	"fmt"

	"go-parser/pkg/ast"
	"go-parser/pkg/common"
	"go-parser/pkg/config"
	"go-parser/pkg/lexer"
	"go-parser/pkg/stream"
)

// IfHandler - обработчик if/else конструкций
type IfHandler struct {
	config  config.ConstructHandlerConfig
	verbose bool
}

// NewIfHandler создает новый обработчик if/else конструкций
func NewIfHandler(config config.ConstructHandlerConfig) *IfHandler {
	return NewIfHandlerWithVerbose(config, false)
}

// NewIfHandlerWithVerbose создает новый обработчик if/else конструкций с поддержкой verbose режима
func NewIfHandlerWithVerbose(config config.ConstructHandlerConfig, verbose bool) *IfHandler {
	return &IfHandler{
		config:  config,
		verbose: verbose,
	}
}

// CanHandle проверяет, может ли обработчик обработать токен
func (h *IfHandler) CanHandle(token lexer.Token) bool {
	// Обрабатываем токен 'if'
	return token.Type == 39 // TokenIf = 39 (реальное значение из отладки)
}

// Handle обрабатывает if/else конструкцию
func (h *IfHandler) Handle(ctx *common.ParseContext) (interface{}, error) {
	// Проверяем защиту от рекурсии
	if err := ctx.Guard.Enter(); err != nil {
		return nil, err
	}
	defer ctx.Guard.Exit()

	tokenStream := ctx.TokenStream

	// 1. Проверяем токен 'if'
	ifToken := tokenStream.Current()
	if h.verbose {
		fmt.Printf("DEBUG: IfHandler.Handle called with ifToken type %d, value '%s'\n", ifToken.Type, ifToken.Value)
	}
	if ifToken.Type != lexer.TokenIf {
		return nil, fmt.Errorf("expected 'if', got %s", ifToken.Type)
	}

	// 2. Проверяем структуру перед потреблением токенов
	// If-конструкция: if (condition) { ... } [else { ... }]
	if !tokenStream.HasMore() {
		return nil, fmt.Errorf("expected '(' after 'if' - no more tokens")
	}

	// Потребляем 'if'
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
			Position: ifToken.Position,
			Line:     ifToken.Line,
			Column:   ifToken.Column,
		}
	}

	// Добавляем отладочный вывод после потребления (
	if h.verbose {
		fmt.Printf("DEBUG: After consuming (, current token type: %d, value: '%s'\n", tokenStream.Current().Type, tokenStream.Current().Value)
	}

	// 4. Читаем условие с использованием бинарного обработчика
	condition, err := h.parseConditionWithBinaryHandler(ctx)
	if err != nil {
		if h.verbose {
			fmt.Printf("DEBUG: IfHandler failed to parse condition, error: %v\n", err)
		}
		return nil, fmt.Errorf("failed to parse condition: %v", err)
	}

	// 5. Проверяем и потребляем токен ')' (только если была открывающая скобка)
	var rParenToken lexer.Token
	if lParenToken.Value == "(" {
		// Была открывающая скобка, значит должна быть и закрывающая
		if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenRightParen {
			return nil, fmt.Errorf("expected ')' after condition")
		}
		rParenToken = tokenStream.Consume()
	} else {
		// Не было открывающей скобки, создаем пустой токен для совместимости
		rParenToken = lexer.Token{
			Type:     lexer.TokenRightParen,
			Value:    ")",
			Position: ifToken.Position,
			Line:     ifToken.Line,
			Column:   ifToken.Column,
		}
	}

	// 6. Проверяем и потребляем токен '{'
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenLBrace {
		return nil, fmt.Errorf("expected '{' after ')'")
	}
	lBraceToken := tokenStream.Consume()

	// 7. Читаем тело if
	body, err := h.parseIfBody(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to parse if body: %v", err)
	}

	// 8. Проверяем и потребляем токен '}'
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenRBrace {
		return nil, fmt.Errorf("expected '}' after if body")
	}
	rBraceToken := tokenStream.Consume()

	// 9. Создаем узел AST для if
	blockStatement := ast.NewBlockStatement(lBraceToken, rBraceToken, body)
	ifNode := ast.NewIfStatement(ifToken, lParenToken, rParenToken, condition, blockStatement)

	// 10. Проверяем наличие else
	if tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenElse {
		// Потребляем 'else'
		elseToken := tokenStream.Consume()

		// Проверяем, что после 'else' идет - это 'if' (else if) или '{' (обычный else)
		if tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenIf {
			// Это конструкция 'else if'
			// Парсим вложенный if как альтернативу
			nestedIfHandler := NewIfHandler(config.ConstructHandlerConfig{})
			result, err := nestedIfHandler.Handle(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to parse else if: %v", err)
			}
			if nestedIf, ok := result.(*ast.IfStatement); ok {
				// Создаем блок для else if и добавляем к if
				elseIfBody := []ast.Statement{nestedIf}
				elseIfBlockStatement := ast.NewBlockStatement(
					nestedIf.IfToken,
					nestedIf.IfToken, // Используем ifToken как маркер конца для блока
					elseIfBody,
				)
				ifNode.SetElse(elseToken, elseIfBlockStatement)
			} else {
				return nil, fmt.Errorf("expected IfStatement in else if, got %T", result)
			}
		} else {
			// Обычный else блок
			// Проверяем и потребляем токен '{' для else
			if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenLBrace {
				return nil, fmt.Errorf("expected '{' after 'else'")
			}
			elseLBraceToken := tokenStream.Consume()

			// Читаем тело else
			elseBody, err := h.parseIfBody(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to parse else body: %v", err)
			}

			// Проверяем и потребляем токен '}' для else
			if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenRBrace {
				return nil, fmt.Errorf("expected '}' after else body")
			}
			elseRBraceToken := tokenStream.Consume()

			// Создаем блок для else и добавляем к if
			elseBlockStatement := ast.NewBlockStatement(elseLBraceToken, elseRBraceToken, elseBody)
			ifNode.SetElse(elseToken, elseBlockStatement)
		}
	}

	return ifNode, nil
}

// parseConditionWithBinaryHandler парсит условие if с использованием бинарного обработчика
func (h *IfHandler) parseConditionWithBinaryHandler(ctx *common.ParseContext) (ast.Expression, error) {
	tokenStream := ctx.TokenStream

	if h.verbose {
		fmt.Printf("DEBUG: parseConditionWithBinaryHandler called\n")
	}

	if !tokenStream.HasMore() {
		return nil, fmt.Errorf("unexpected EOF in condition")
	}

	// Добавляем отладочный вывод для проверки первого токена
	if h.verbose {
		fmt.Printf("DEBUG: parseConditionWithBinaryHandler first token type: %d, value: '%s'\n", tokenStream.Current().Type, tokenStream.Current().Value)
	}

	// Используем централизованный парсер выражений для правильной обработки приоритетов
	expressionParser := NewUnifiedExpressionParser(h.verbose)

	// Проверяем, начинается ли выражение с идентификатора или зарезервированного слова (возможно, квалифицированная переменная)
	firstToken := tokenStream.Current()
	if h.verbose {
		fmt.Printf("DEBUG: Checking complex expressions, firstToken type: %d, value: '%s'\n", firstToken.Type, firstToken.Value)
	}
	if firstToken.Type == lexer.TokenPython ||
		firstToken.Type == lexer.TokenLua ||
		firstToken.Type == lexer.TokenPy ||
		firstToken.Type == lexer.TokenGo ||
		firstToken.Type == lexer.TokenNode ||
		firstToken.Type == lexer.TokenJS {
		if h.verbose {
			fmt.Printf("DEBUG: First token is language token, checking for DOT\n")
		}
		// Проверяем, не является ли это вызовом функции с точкой (python.check_status()) или доступом к элементу массива/словаря
		if tokenStream.Peek().Type == lexer.TokenDot {
			if h.verbose {
				fmt.Printf("DEBUG: Found DOT, checking for function call or array/dict access pattern\n")
			}
			peek2 := tokenStream.PeekN(2)
			peek3 := tokenStream.PeekN(3)

			if peek2.Type == lexer.TokenIdentifier && peek3.Type == lexer.TokenLeftParen {
				if h.verbose {
					fmt.Printf("DEBUG: Detected language function call pattern in complex expressions\n")
				}
				// Это вызов функции с точечной нотацией вида py.get_none()
				// Используем LanguageCallHandler напрямую с частичным парсингом
				languageCallHandler := NewLanguageCallHandler(config.ConstructHandlerConfig{})
				// Создаем контекст с включенным режимом частичного парсинга
				partialCtx := &common.ParseContext{
					TokenStream:        ctx.TokenStream,
					Parser:             ctx.Parser,
					Depth:              ctx.Depth,
					MaxDepth:           ctx.MaxDepth,
					Guard:              ctx.Guard,
					LoopDepth:          ctx.LoopDepth,
					InputStream:        ctx.InputStream,
					PartialParsingMode: true, // Включаем режим частичного парсинга
				}
				result, err := languageCallHandler.Handle(partialCtx)
				if err != nil {
					if h.verbose {
						fmt.Printf("DEBUG: LanguageCallHandler failed with error: %v\n", err)
					}
					return nil, fmt.Errorf("failed to parse language function call: %v", err)
				}
				if call, ok := result.(*ast.LanguageCall); ok {
					if h.verbose {
						fmt.Printf("DEBUG: LanguageCallHandler succeeded, returning LanguageCall\n")
					}
					return call, nil
				} else {
					return nil, fmt.Errorf("expected LanguageCall, got %T", result)
				}
			} else {
				if h.verbose {
					fmt.Printf("DEBUG: Detected field access pattern, using BinaryExpressionHandler for full expression\n")
				}
				// Это сложное выражение с доступом к полю, возможно с последующим доступом к массиву/словарю и операторами вида py.array[0] == 1
				binaryHandler := NewBinaryExpressionHandlerWithVerbose(config.ConstructHandlerConfig{}, h.verbose)

				if h.verbose {
					fmt.Printf("DEBUG: Before parseOperand, current token: %s (%s)\n", tokenStream.Current().Value, tokenStream.Current().Type)
					fmt.Printf("DEBUG: Available tokens: ")
					for i := 0; i < 10 && tokenStream.PeekN(i).Type != lexer.TokenEOF; i++ {
						fmt.Printf("%s ", tokenStream.PeekN(i).Value)
					}
					fmt.Printf("\n")
				}

				// Сначала парсим левый операнд с помощью parseOperand
				leftExpr, err := binaryHandler.parseOperand(ctx)
				if h.verbose {
					fmt.Printf("DEBUG: BinaryExpressionHandler.parseOperand succeeded, result type: %T, result: %v\n", leftExpr, leftExpr)
				}
				if err != nil {
					if h.verbose {
						fmt.Printf("DEBUG: BinaryExpressionHandler.parseOperand failed with error: %v\n", err)
					}
					return nil, fmt.Errorf("failed to parse left operand: %v", err)
				}

				// Затем парсим полное выражение с бинарными операторами
				result, err := binaryHandler.ParseFullExpression(ctx, leftExpr)
				if err != nil {
					if h.verbose {
						fmt.Printf("DEBUG: BinaryExpressionHandler failed with error: %v\n", err)
					}
					return nil, fmt.Errorf("failed to parse binary expression: %v", err)
				}
				if h.verbose {
					fmt.Printf("DEBUG: BinaryExpressionHandler succeeded, result type: %T, result: %v\n", result, result)
				}
				if result == nil {
					return nil, fmt.Errorf("BinaryExpressionHandler returned nil result")
				}
				return result, nil
			}
		}
	}

	// НОВАЯ ЦЕНТРАЛИЗОВАННАЯ ЛОГИКА: Всегда используем UnifiedExpressionParser
	result, err := expressionParser.ParseExpression(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to parse condition: %v", err)
	}

	return result, nil

	// СТАРАЯ ЛОГИКА (закомментирована для безопасности)
	/*
			var leftExpr ast.Expression
			if h.verbose {
				fmt.Printf("DEBUG: IfHandler.parseConditionWithBinaryHandler firstToken type: %d, value: '%s'\n", firstToken.Type, firstToken.Value)
				fmt.Printf("DEBUG: Entering switch for firstToken type: %d, value: '%s'\n", firstToken.Type, firstToken.Value)
			}
			switch firstToken.Type {
			case lexer.TokenIdentifier:
				// Простой идентификатор
				tokenStream.Consume()
				leftExpr = ast.NewIdentifier(firstToken, firstToken.Value)

			case lexer.TokenPython, lexer.TokenLua, lexer.TokenPy, lexer.TokenGo, lexer.TokenJS, lexer.TokenNode:
				if h.verbose {
					fmt.Printf("DEBUG: Case for language tokens, peek type: %d\n", tokenStream.Peek().Type)
				}
				if tokenStream.Peek().Type == lexer.TokenLeftParen {
					// Это вызов функции вида func()
					languageCallHandler := NewLanguageCallHandler(config.ConstructHandlerConfig{})
					result, err := languageCallHandler.Handle(ctx)
					if err != nil {
						return nil, fmt.Errorf("failed to parse function call: %v", err)
					}
					if call, ok := result.(*ast.LanguageCall); ok {
						leftExpr = call
					} else {
						return nil, fmt.Errorf("expected LanguageCall, got %T", result)
					}
				} else if tokenStream.Peek().Type == lexer.TokenDot {
					if h.verbose {
						fmt.Printf("DEBUG: Found DOT after language token, checking for function call\n")
					}
					// Проверяем, не является ли это вызовом функции вида py.get_none()
					peek2 := tokenStream.PeekN(2)
					peek3 := tokenStream.PeekN(3)
					if h.verbose {
						fmt.Printf("DEBUG: peek2 type: %d, peek3 type: %d\n", peek2.Type, peek3.Type)
					}

					if peek2.Type == lexer.TokenIdentifier && peek3.Type == lexer.TokenLeftParen {
						if h.verbose {
							fmt.Printf("DEBUG: Detected language function call pattern\n")
						}
						// Это вызов функции с точечной нотацией вида py.get_none()
						languageCallHandler := NewLanguageCallHandler(config.ConstructHandlerConfig{})
						result, err := languageCallHandler.Handle(ctx)
						if err != nil {
							return nil, fmt.Errorf("failed to parse language function call: %v", err)
						}
						if call, ok := result.(*ast.LanguageCall); ok {
							leftExpr = call
						} else {
							return nil, fmt.Errorf("expected LanguageCall, got %T", result)
						}
					} else {
						// Это квалифицированная переменная типа python.x или lua.y
						// Парсим ее как qualified identifier
						languageToken := tokenStream.Consume() // consume language token
						tokenStream.Consume()                  // consume dot

						if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenIdentifier {
							return nil, fmt.Errorf("expected identifier after dot in qualified variable")
						}

						variableToken := tokenStream.Consume()
						variableName := variableToken.Value

						// Создаем qualified identifier
						qualifiedId := ast.NewQualifiedIdentifier(languageToken, variableToken, firstToken.Value, variableName)
						leftExpr = qualifiedId
					}
				} else {
					// Простой идентификатор
					tokenStream.Consume()
					leftExpr = ast.NewIdentifier(firstToken, firstToken.Value)
				}

			case lexer.TokenString:
				// Строковой литерал
				tokenStream.Consume()
				leftExpr = &ast.StringLiteral{
					Value: firstToken.Value,
					Pos: ast.Position{
						Line:   firstToken.Line,
						Column: firstToken.Column,
						Offset: firstToken.Position,
					},
				}

			case lexer.TokenNumber:
				// Числовой литерал
				tokenStream.Consume()
				leftExpr = &ast.NumberLiteral{
					Value: parseFloat(firstToken.Value),
					Pos: ast.Position{
						Line:   firstToken.Line,
						Column: firstToken.Column,
						Offset: firstToken.Position,
					},
				}

			case lexer.TokenTrue, lexer.TokenFalse:
				// Булев литерал
				tokenStream.Consume()
				leftExpr = &ast.BooleanLiteral{
					Value: firstToken.Type == lexer.TokenTrue,
					Pos: ast.Position{
						Line:   firstToken.Line,
						Column: firstToken.Column,
						Offset: firstToken.Position,
					},
				}

			case lexer.TokenLBracket:
				// Массивный литерал
				if h.verbose {
					fmt.Printf("DEBUG: Parsing array literal in condition\n")
				}
				arrayHandler := NewArrayHandler(100, 0)
				arrayCtx := &common.ParseContext{
					TokenStream: ctx.TokenStream,
					Parser:      ctx.Parser,
					Depth:       ctx.Depth,
					MaxDepth:    ctx.MaxDepth,
					Guard:       ctx.Guard,
					LoopDepth:   ctx.LoopDepth,
					InputStream: ctx.InputStream,
				}
				arrayResult, err := arrayHandler.Handle(arrayCtx)
				if err != nil {
					return nil, fmt.Errorf("failed to parse array in condition: %v", err)
				}
				if array, ok := arrayResult.(*ast.ArrayLiteral); ok {
					leftExpr = array
				} else {
					return nil, fmt.Errorf("expected ArrayLiteral, got %T", arrayResult)
				}

			case lexer.TokenLBrace:
				// Объектный литерал
				if h.verbose {
					fmt.Printf("DEBUG: Parsing object literal in condition\n")
				}
				objectHandler := NewObjectHandler(100, 0)
				objectCtx := &common.ParseContext{
					TokenStream: ctx.TokenStream,
					Parser:      ctx.Parser,
					Depth:       ctx.Depth,
					MaxDepth:    ctx.MaxDepth,
					Guard:       ctx.Guard,
					LoopDepth:   ctx.LoopDepth,
					InputStream: ctx.InputStream,
				}
				objectResult, err := objectHandler.Handle(objectCtx)
				if err != nil {
					return nil, fmt.Errorf("failed to parse object in condition: %v", err)
				}
				if object, ok := objectResult.(*ast.ObjectLiteral); ok {
					leftExpr = object
				} else {
					return nil, fmt.Errorf("expected ObjectLiteral, got %T", objectResult)
				}

			default:
				return nil, fmt.Errorf("unsupported condition type: %s", firstToken.Type)
			}

			// Проверяем, есть ли бинарный оператор
			if tokenStream.HasMore() && isBinaryOperator(tokenStream.Current().Type) {
				// Парсим бинарное выражение
				result, err := binaryHandler.ParseFullExpression(ctx, leftExpr)
				return result, err
			}

			return leftExpr, nil
		}

		// parseCondition парсит условие if
		func (h *IfHandler) parseCondition(ctx *common.ParseContext, tokenStream stream.TokenStream) (ast.Expression, error) {
			if !tokenStream.HasMore() {
				return nil, fmt.Errorf("unexpected EOF in condition")
			}

			// Пока поддерживаем простые условия:
			// - Булевы литералы (true, false)
			// - Числовые литералы
			// - Идентификаторы
			// - Вызовы функций
			// - Бинарные выражения с операторами сравнения

			token := tokenStream.Current()
			switch token.Type {
			case lexer.TokenIdentifier:
				// Это может быть простой идентификатор, вызов функции или бинарное выражение
				peekToken := tokenStream.Peek()
				if peekToken.Type == lexer.TokenLeftParen {
					// Это вызов функции, парсим напрямую
					call, err := h.parseLanguageCall(tokenStream)
					if err != nil {
						return nil, fmt.Errorf("failed to parse function call in condition: %v", err)
					}
					return call, nil
				} else if peekToken.Type == lexer.TokenDot {
					// Это может быть вызов функции вида lua.check()
					// Проверяем, есть ли после точки идентификатор и открывающая скобка
					// Вместо сложной логики, просто пробуем распарсить как вызов функции
					call, err := h.parseLanguageCall(tokenStream)
					if err == nil {
						return call, nil
					}
					// Если не получилось, возвращаем простой идентификатор
					tokenStream.Consume()
					return ast.NewIdentifier(token, token.Value), nil
				} else if isBinaryOperator(peekToken.Type) {
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
				return &ast.NumberLiteral{
					Value: parseFloat(token.Value),
					Pos: ast.Position{
						Line:   token.Line,
						Column: token.Column,
						Offset: token.Position,
					},
				}, nil

			default:
				return nil, fmt.Errorf("unsupported condition type: %s", token.Type)
			}
	*/
}

// parseIfBody парсит тело if/else
func (h *IfHandler) parseIfBody(ctx *common.ParseContext) ([]ast.Statement, error) {
	tokenStream := ctx.TokenStream
	body := make([]ast.Statement, 0)

	// Пропускаем пробелы после '{'
	for tokenStream.HasMore() {
		current := tokenStream.Current()

		if current.Type == lexer.TokenEOF {
			break
		}

		// Если встречаем '}', заканчиваем тело
		if current.Type == lexer.TokenRBrace {
			break
		}

		// Если встречаем 'if', это вложенный if
		if current.Type == lexer.TokenIf {
			nestedIfHandler := NewIfHandler(config.ConstructHandlerConfig{})
			result, err := nestedIfHandler.Handle(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to parse nested if: %v", err)
			}
			if nestedIf, ok := result.(*ast.IfStatement); ok {
				body = append(body, nestedIf)
				continue
			}
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

		// Пытаемся распарсить как statement
		// Поддерживаем вызовы функций и присваивания
		if current.Type == lexer.TokenPy || current.Type == lexer.TokenPython || current.Type == lexer.TokenLua || current.Type == lexer.TokenGo || current.Type == lexer.TokenNode || current.Type == lexer.TokenJS || current.Type == lexer.TokenIdentifier {
			if h.verbose {
				fmt.Printf("DEBUG: parseIfBody - found token type %d, value '%s'\n", current.Type, current.Value)
			}
			// Проверяем, не является ли это вызовом функции другого языка
			if tokenStream.Peek().Type == lexer.TokenDot {
				peek2 := tokenStream.PeekN(2)
				peek3 := tokenStream.PeekN(3)
				if h.verbose {
					fmt.Printf("DEBUG: parseIfBody - found DOT after %s, peek2 type: %d, peek3 type: %d\n", current.Value, peek2.Type, peek3.Type)
				}

				// Проверяем, не является ли это вызовом функции (py.function(...))
				if peek2.Type == lexer.TokenIdentifier && peek3.Type == lexer.TokenLeftParen {
					if h.verbose {
						fmt.Printf("DEBUG: parseIfBody - detected function call pattern\n")
					}
					// Это вызов типа lua.print или python.math.sqrt
					// Парсим напрямую без делегирования
					call, err := h.parseLanguageCall(tokenStream)
					if err == nil {
						if h.verbose {
							fmt.Printf("DEBUG: parseIfBody - language call parsing succeeded\n")
						}
						body = append(body, call)
						continue
					} else {
						if h.verbose {
							fmt.Printf("DEBUG: parseIfBody - language call parsing failed: %v\n", err)
						}
					}
				} else if peek2.Type == lexer.TokenIdentifier && tokenStream.PeekN(3).Type == lexer.TokenAssign {
					if h.verbose {
						fmt.Printf("DEBUG: parseIfBody - detected qualified assignment pattern\n")
					}
					// Это присваивание квалифицированной переменной: py.result = "value"
					// Используем AssignmentHandler напрямую, он должен обработать квалифицированные идентификаторы
					assignmentHandler := NewAssignmentHandler(80, 4)
					result, err := assignmentHandler.Handle(ctx)
					if err == nil {
						if h.verbose {
							fmt.Printf("DEBUG: parseIfBody - AssignmentHandler for qualified assignment succeeded\n")
						}
						if assignment, ok := result.(*ast.VariableAssignment); ok {
							body = append(body, assignment)
							continue
						}
					} else {
						if h.verbose {
							fmt.Printf("DEBUG: parseIfBody - AssignmentHandler for qualified assignment failed: %v\n", err)
						}
					}
				}
			}

			// Проверяем, не является ли это присваиванием (смотрим на следующий токен)
			peekForAssign := tokenStream.Peek()
			if peekForAssign.Type == lexer.TokenAssign {
				if h.verbose {
					fmt.Printf("DEBUG: parseIfBody - found ASSIGN after %s, trying to parse assignment\n", current.Value)
				}
				// Для всех случаев используем AssignmentHandler, он умеет работать с квалифицированными идентификаторами
				// Но сначала нужно проверить, не является ли это квалифицированным идентификатором
				if current.Type == lexer.TokenIdentifier && tokenStream.HasMore() && tokenStream.Peek().Type == lexer.TokenAssign {
					// Это простой идентификатор с присваиванием: result = "value"
					assignmentHandler := NewAssignmentHandler(80, 4)
					result, err := assignmentHandler.Handle(ctx)
					if err == nil {
						if h.verbose {
							fmt.Printf("DEBUG: parseIfBody - AssignmentHandler for simple identifier succeeded\n")
						}
						if assignment, ok := result.(*ast.VariableAssignment); ok {
							body = append(body, assignment)
							continue
						}
					} else {
						if h.verbose {
							fmt.Printf("DEBUG: parseIfBody - AssignmentHandler for simple identifier failed: %v\n", err)
						}
					}
				} else {
					// Для языковых токенов и сложных случаев используем AssignmentHandler напрямую
					// Он должен сам разобраться с квалифицированными идентификаторами
					assignmentHandler := NewAssignmentHandler(80, 4)
					result, err := assignmentHandler.Handle(ctx)
					if err == nil {
						if h.verbose {
							fmt.Printf("DEBUG: parseIfBody - AssignmentHandler succeeded\n")
						}
						if assignment, ok := result.(*ast.VariableAssignment); ok {
							body = append(body, assignment)
							continue
						}
					} else {
						if h.verbose {
							fmt.Printf("DEBUG: parseIfBody - AssignmentHandler failed: %v\n", err)
						}
					}
				}
			}
		}

		// Если не смогли распарсить, пропускаем токен
		tokenStream.Consume()
	}

	return body, nil
}

// Config возвращает конфигурацию обработчика
func (h *IfHandler) Config() common.HandlerConfig {
	return common.HandlerConfig{
		IsEnabled: h.config.IsEnabled,
		Priority:  h.config.Priority,
		Name:      h.config.Name,
	}
}

// Name возвращает имя обработчика
func (h *IfHandler) Name() string {
	return h.config.Name
}

// parseLanguageCall парсит вызов функции другого языка напрямую
func (h *IfHandler) parseLanguageCall(tokenStream stream.TokenStream) (*ast.LanguageCall, error) {
	// Читаем язык
	languageToken := tokenStream.Consume()
	language := languageToken.Value

	// Проверяем и потребляем DOT
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenDot {
		return nil, fmt.Errorf("expected DOT after language '%s'", language)
	}
	tokenStream.Consume() // Consuming dot

	// Читаем имя функции
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenIdentifier {
		return nil, fmt.Errorf("expected function name after DOT")
	}
	functionToken := tokenStream.Consume()
	functionName := functionToken.Value

	// Проверяем и потребляем открывающую скобку
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenLeftParen {
		return nil, fmt.Errorf("expected '(' after function name '%s'", functionName)
	}
	tokenStream.Consume() // Consuming '('

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
			argToken := tokenStream.Consume()
			var arg ast.Expression

			switch argToken.Type {
			case lexer.TokenString:
				arg = &ast.StringLiteral{
					Value: argToken.Value,
					Pos: ast.Position{
						Line:   argToken.Line,
						Column: argToken.Column,
						Offset: argToken.Position,
					},
				}
			case lexer.TokenNumber:
				arg = &ast.NumberLiteral{
					Value: parseFloat(argToken.Value),
					Pos: ast.Position{
						Line:   argToken.Line,
						Column: argToken.Column,
						Offset: argToken.Position,
					},
				}
			case lexer.TokenIdentifier:
				// Простой идентификатор (например, локальная переменная из pattern matching)
				arg = ast.NewIdentifier(argToken, argToken.Value)
			default:
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

	// Проверяем закрывающую скобку
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenRightParen {
		return nil, fmt.Errorf("expected ')' after arguments")
	}
	tokenStream.Consume() // Consuming ')'

	// Создаем узел AST
	startPos := ast.Position{
		Line:   languageToken.Line,
		Column: languageToken.Column,
		Offset: languageToken.Position,
	}

	node := &ast.LanguageCall{
		Language:  language, // Используем имя языка напрямую
		Function:  functionName,
		Arguments: arguments,
		Pos:       startPos,
	}

	return node, nil
}
