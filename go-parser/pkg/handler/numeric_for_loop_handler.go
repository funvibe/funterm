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

// NumericForLoopHandler - обработчик Lua-style числовых циклов
type NumericForLoopHandler struct {
	config  config.ConstructHandlerConfig
	verbose bool
}

// NewNumericForLoopHandler создает новый обработчик числовых циклов
func NewNumericForLoopHandler(config config.ConstructHandlerConfig) *NumericForLoopHandler {
	return NewNumericForLoopHandlerWithVerbose(config, false)
}

// NewNumericForLoopHandlerWithVerbose создает новый обработчик числовых циклов с поддержкой verbose режима
func NewNumericForLoopHandlerWithVerbose(config config.ConstructHandlerConfig, verbose bool) *NumericForLoopHandler {
	return &NumericForLoopHandler{
		config:  config,
		verbose: verbose,
	}
}

// CanHandle проверяет, может ли обработчик обработать токен
func (h *NumericForLoopHandler) CanHandle(token lexer.Token) bool {
	// Обрабатываем токен 'for'
	return token.Type == lexer.TokenFor
}

// Handle обрабатывает Lua-style числовой цикл
func (h *NumericForLoopHandler) Handle(ctx *common.ParseContext) (interface{}, error) {
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

	// 1. Проверяем токен 'for'
	forToken := tokenStream.Current()
	if forToken.Type != lexer.TokenFor {
		return nil, fmt.Errorf("expected 'for', got %s", forToken.Type)
	}

	// 2. Проверяем структуру перед потреблением токенов
	// Lua-цикл: for var = start, end, step do ... end
	if !tokenStream.HasMore() {
		return nil, fmt.Errorf("expected loop variable after 'for' - no more tokens")
	}

	// Если все проверки прошли, это действительно Lua-цикл, потребляем токены
	tokenStream.Consume() // потребляем 'for'

	// Проверяем, что текущий токен - идентификатор (переменная)
	current := tokenStream.Current()
	if h.verbose {
		fmt.Printf("DEBUG NumericForLoopHandler: Current token after 'for': %s(%s)\n", current.Type, current.Value)
	}
	if current.Type != lexer.TokenIdentifier {
		return nil, fmt.Errorf("expected loop variable after 'for', got %s", current.Type)
	}

	// Проверяем, что после переменной идет '=' (признак Lua-цикла)
	peek1 := tokenStream.Peek()
	if h.verbose {
		fmt.Printf("DEBUG NumericForLoopHandler: Peek() token: %s(%s)\n", peek1.Type, peek1.Value)
	}
	if peek1.Type != lexer.TokenAssign {
		return nil, fmt.Errorf("expected '=' after loop variable, got %s", peek1.Type)
	}

	// 2. Читаем переменную цикла
	varToken := tokenStream.Consume() // потребляем переменную
	variable := ast.NewIdentifier(varToken, varToken.Value)

	// 3. Проверяем и потребляем токен '='
	_ = tokenStream.Current()
	tokenStream.Consume() // потребляем '='

	// 4. Читаем начальное значение
	start, err := h.parseNumericExpression(tokenStream)
	if err != nil {
		return nil, fmt.Errorf("failed to parse start value: %v", err)
	}

	// 5. Проверяем токен ','
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenComma {
		return nil, fmt.Errorf("expected ',' after start value")
	}
	tokenStream.Consume() // потребляем запятую

	// 6. Читаем конечное значение
	end, err := h.parseNumericExpression(tokenStream)
	if err != nil {
		return nil, fmt.Errorf("failed to parse end value: %v", err)
	}

	// 7. Проверяем опциональный шаг
	var step ast.ProtoNode
	if tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenComma {
		tokenStream.Consume() // потребляем запятую
		step, err = h.parseNumericExpression(tokenStream)
		if err != nil {
			return nil, fmt.Errorf("failed to parse step value: %v", err)
		}
	}

	// 8. Проверяем токен 'do'
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenDo {
		return nil, fmt.Errorf("expected 'do' after for loop parameters")
	}
	doToken := tokenStream.Consume()

	// 9. Читаем тело цикла
	body, err := h.parseLoopBody(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to parse loop body: %v", err)
	}

	// 10. Проверяем токен 'end'
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenEnd {
		return nil, fmt.Errorf("expected 'end' after loop body")
	}
	endToken := tokenStream.Consume()

	// 11. Создаем узел AST
	loopNode := ast.NewNumericForLoopStatement(forToken, doToken, endToken, variable, start, end, step, body)

	return loopNode, nil
}

// parseNumericExpression парсит числовое выражение
func (h *NumericForLoopHandler) parseNumericExpression(tokenStream stream.TokenStream) (ast.ProtoNode, error) {
	if !tokenStream.HasMore() {
		return nil, fmt.Errorf("unexpected EOF")
	}

	token := tokenStream.Consume()
	switch token.Type {
	case lexer.TokenNumber:
		// Числовой литерал
		value, err := strconv.ParseFloat(token.Value, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid number format: %s", token.Value)
		}
		return &ast.NumberLiteral{
			Value: value,
			Pos: ast.Position{
				Line:   token.Line,
				Column: token.Column,
				Offset: token.Position,
			},
		}, nil

	case lexer.TokenIdentifier:
		// Идентификатор (переменная)
		return ast.NewIdentifier(token, token.Value), nil

	default:
		return nil, fmt.Errorf("unsupported numeric expression type: %s", token.Type)
	}
}

// parseLoopBody парсит тело цикла
func (h *NumericForLoopHandler) parseLoopBody(ctx *common.ParseContext) ([]ast.Statement, error) {
	tokenStream := ctx.TokenStream
	body := make([]ast.Statement, 0)

	if h.verbose {
		fmt.Printf("DEBUG parseLoopBody: Starting to parse loop body\n")
		fmt.Printf("DEBUG parseLoopBody: Current token: %s(%s)\n", tokenStream.Current().Type, tokenStream.Current().Value)
	}

	// Пропускаем пробелы после 'do'
	for tokenStream.HasMore() {
		current := tokenStream.Current()
		if h.verbose {
			fmt.Printf("DEBUG parseLoopBody: Processing token: %s(%s)\n", current.Type, current.Value)
		}

		if current.Type == lexer.TokenEOF {
			if h.verbose {
				fmt.Printf("DEBUG parseLoopBody: Found EOF, stopping\n")
			}
			break
		}

		// Если встречаем 'end', заканчиваем тело цикла
		if current.Type == lexer.TokenEnd {
			if h.verbose {
				fmt.Printf("DEBUG parseLoopBody: Found END, stopping body parsing\n")
			}
			break
		}

		// Обрабатываем break операторы
		if current.Type == lexer.TokenBreak {
			breakHandler := NewBreakHandler(config.ConstructHandlerConfig{})
			result, err := breakHandler.Handle(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to parse break statement: %v", err)
			}
			if breakStmt, ok := result.(*ast.BreakStatement); ok {
				body = append(body, breakStmt)
				continue
			}
		}

		// Обрабатываем continue операторы
		if current.Type == lexer.TokenContinue {
			continueHandler := NewContinueHandler(config.ConstructHandlerConfig{})
			result, err := continueHandler.Handle(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to parse continue statement: %v", err)
			}
			if continueStmt, ok := result.(*ast.ContinueStatement); ok {
				body = append(body, continueStmt)
				continue
			}
		}

		// Если встречаем другой 'for', это вложенный цикл
		if current.Type == lexer.TokenFor {
			// Рекурсивно обрабатываем вложенный цикл
			nestedForHandler := NewNumericForLoopHandler(config.ConstructHandlerConfig{})
			result, err := nestedForHandler.Handle(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to parse nested for loop: %v", err)
			}
			if nestedLoop, ok := result.(*ast.NumericForLoopStatement); ok {
				body = append(body, nestedLoop)
				continue
			}
		}

		// Обрабатываем while циклы
		if current.Type == lexer.TokenWhile {
			whileHandler := NewWhileLoopHandler(config.ConstructHandlerConfig{})
			result, err := whileHandler.Handle(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to parse while loop: %v", err)
			}
			if whileLoop, ok := result.(*ast.WhileStatement); ok {
				body = append(body, whileLoop)
				continue
			}
		}

		// Обрабатываем match выражения
		if current.Type == lexer.TokenMatch {
			if h.verbose {
				fmt.Printf("DEBUG parseLoopBody: Found MATCH token, calling MatchHandler\n")
			}
			matchHandler := NewMatchHandler(config.ConstructHandlerConfig{})
			result, err := matchHandler.Handle(ctx)
			if h.verbose {
				fmt.Printf("DEBUG parseLoopBody: MatchHandler result: %v, error: %v\n", result, err)
			}
			if err != nil {
				return nil, fmt.Errorf("failed to parse match statement: %v", err)
			}
			if matchStmt, ok := result.(*ast.MatchStatement); ok {
				body = append(body, matchStmt)
				if h.verbose {
					fmt.Printf("DEBUG parseLoopBody: Successfully added match statement to body\n")
				}
				continue
			}
		}

		// Обрабатываем if выражения
		if current.Type == lexer.TokenIf {
			if h.verbose {
				fmt.Printf("DEBUG parseLoopBody: Found IF token, calling IfHandler\n")
			}
			ifHandler := NewIfHandler(config.ConstructHandlerConfig{})
			result, err := ifHandler.Handle(ctx)
			if h.verbose {
				fmt.Printf("DEBUG parseLoopBody: IfHandler result: %v, error: %v\n", result, err)
			}
			if err != nil {
				return nil, fmt.Errorf("failed to parse if statement: %v", err)
			}
			if ifStmt, ok := result.(*ast.IfStatement); ok {
				body = append(body, ifStmt)
				if h.verbose {
					fmt.Printf("DEBUG parseLoopBody: Successfully added if statement to body\n")
				}
				continue
			}
		}

		// Проверяем, не является ли это вызовом функции другого языка
		if current.Type == lexer.TokenIdentifier || current.Type == lexer.TokenLua ||
			current.Type == lexer.TokenPython || current.Type == lexer.TokenPy || current.Type == lexer.TokenGo ||
			current.Type == lexer.TokenNode || current.Type == lexer.TokenJS {

			// Сначала проверяем, не является ли это присваиванием (смотрим на следующий через DOT токен)
			if current.Type == lexer.TokenJS && tokenStream.Peek().Type == lexer.TokenDot {
				// Проверяем токен после DOT
				if tokenStream.PeekN(2).Type == lexer.TokenIdentifier {
					// Проверяем, есть ли ASSIGN после идентификатора
					identifierAfterDot := tokenStream.PeekN(2)
					if tokenStream.PeekN(3).Type == lexer.TokenAssign {
						// Это присваивание типа js.total = ...
						if h.verbose {
							fmt.Printf("DEBUG parseLoopBody: Detected assignment pattern: js.%s = ...\n", identifierAfterDot.Value)
						}
						assignmentHandler := NewAssignmentHandlerWithVerbose(80, 4, h.verbose)
						result, err := assignmentHandler.Handle(ctx)
						if err == nil {
							if assignment, ok := result.(*ast.VariableAssignment); ok {
								body = append(body, assignment)
								if h.verbose {
									fmt.Printf("DEBUG parseLoopBody: Successfully added assignment statement to body\n")
								}
								continue
							}
						} else {
							if h.verbose {
								fmt.Printf("DEBUG parseLoopBody: AssignmentHandler failed: %v\n", err)
							}
						}
					}
				}
			}

			// Проверяем, не является ли это вызовом функции другого языка
			if tokenStream.Peek().Type == lexer.TokenDot {
				if h.verbose {
					fmt.Printf("DEBUG parseLoopBody: Found potential language call: %s.%s\n", current.Value, tokenStream.Peek().Value)
				}

				// Это может быть вызов типа lua.print или python.math.sqrt
				languageCallHandler := NewLanguageCallHandler(config.ConstructHandlerConfig{})

				// Сохраняем текущую позицию для отладки
				beforePos := tokenStream.Position()
				if h.verbose {
					fmt.Printf("DEBUG parseLoopBody: Calling LanguageCallHandler at position %d\n", beforePos)
				}

				// Добавляем подробный отладочный вывод о текущем состоянии токенов
				if h.verbose {
					fmt.Printf("DEBUG parseLoopBody: Current token before LanguageCallHandler: %s(%s) at pos %d\n",
						tokenStream.Current().Type, tokenStream.Current().Value, tokenStream.Current().Position)
					if tokenStream.HasMore() {
						fmt.Printf("DEBUG parseLoopBody: Next token (peek): %s(%s) at pos %d\n",
							tokenStream.Peek().Type, tokenStream.Peek().Value, tokenStream.Peek().Position)
					}
				}

				result, err := languageCallHandler.Handle(ctx)
				if h.verbose {
					fmt.Printf("DEBUG parseLoopBody: LanguageCallHandler result: %v, error: %v\n", result, err)
				}

				if err == nil {
					if call, ok := result.(*ast.LanguageCall); ok {
						// Wrap LanguageCall in LanguageCallStatement to make it a Statement
						languageCallStmt := &ast.LanguageCallStatement{
							LanguageCall: call,
							IsBackground: false,
						}
						body = append(body, languageCallStmt)
						if h.verbose {
							fmt.Printf("DEBUG parseLoopBody: Successfully added language call statement to body\n")
						}
						continue
					}
				} else {
					if h.verbose {
						fmt.Printf("DEBUG parseLoopBody: LanguageCallHandler failed, restoring position\n")
					}
					// Восстанавливаем позицию, если обработчик не смог распарсить
					tokenStream.SetPosition(beforePos)
				}
			} else {
				if h.verbose {
					fmt.Printf("DEBUG parseLoopBody: Identifier '%s' not followed by DOT, next token: %s\n", current.Value, tokenStream.Peek().Type)
				}
			}
		}

		// Проверяем, не является ли это присваиванием
		if current.Type == lexer.TokenIdentifier || current.Type == lexer.TokenLua ||
			current.Type == lexer.TokenPython || current.Type == lexer.TokenPy || current.Type == lexer.TokenGo ||
			current.Type == lexer.TokenNode || current.Type == lexer.TokenJS {

			// Проверяем, не является ли это присваиванием (смотрим на следующий токен)
			peekForAssign := tokenStream.Peek()
			if peekForAssign.Type == lexer.TokenAssign {
				// Для всех случаев используем AssignmentHandler напрямую
				assignmentHandler := NewAssignmentHandlerWithVerbose(80, 4, h.verbose)
				result, err := assignmentHandler.Handle(ctx)
				if err == nil {
					if h.verbose {
						fmt.Printf("DEBUG parseLoopBody: AssignmentHandler succeeded\n")
					}
					if assignment, ok := result.(*ast.VariableAssignment); ok {
						body = append(body, assignment)
						continue
					}
				} else {
					if h.verbose {
						fmt.Printf("DEBUG parseLoopBody: AssignmentHandler failed: %v\n", err)
					}
				}
			}
		}

		// Проверяем наличие индексного выражения [index]
		if current.Type == lexer.TokenLBracket {
			if h.verbose {
				fmt.Printf("DEBUG parseLoopBody: Found LBRACKET, attempting to parse index expression\n")
			}

			// Потребляем открывающую скобку
			tokenStream.Consume()

			// Парсим индексное выражение с помощью BinaryExpressionHandler
			binaryHandler := NewBinaryExpressionHandler(config.ConstructHandlerConfig{})
			tempCtx := &common.ParseContext{
				TokenStream: tokenStream,
				Parser:      ctx.Parser,
				Depth:       ctx.Depth,
				MaxDepth:    ctx.MaxDepth,
				Guard:       ctx.Guard,
				LoopDepth:   ctx.LoopDepth,
				InputStream: ctx.InputStream,
			}

			// Парсим выражение внутри скобок
			indexExpr, err := binaryHandler.ParseFullExpression(tempCtx, nil)
			if err != nil {
				if h.verbose {
					fmt.Printf("DEBUG parseLoopBody: Failed to parse index expression: %v\n", err)
				}
				// Если не удалось распарсить, продолжаем со следующего токена
				continue
			}

			if h.verbose {
				fmt.Printf("DEBUG parseLoopBody: Successfully parsed index expression: %T\n", indexExpr)
			}

			// Проверяем наличие закрывающей скобки
			if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenRBracket {
				if h.verbose {
					fmt.Printf("DEBUG parseLoopBody: Expected RBRACKET after index expression, got %s\n", tokenStream.Current().Type)
				}
				continue
			}

			// Потребляем закрывающую скобку
			tokenStream.Consume()

			// Продолжаем со следующего токена
			continue
		}

		// Если не смогли распарсить, пропускаем токен
		tokenStream.Consume()
	}

	if h.verbose {
		fmt.Printf("DEBUG parseLoopBody: Finished parsing body, current token: %s(%s)\n", tokenStream.Current().Type, tokenStream.Current().Value)
		fmt.Printf("DEBUG parseLoopBody: Body length: %d\n", len(body))
	}

	return body, nil
}

// Config возвращает конфигурацию обработчика
func (h *NumericForLoopHandler) Config() common.HandlerConfig {
	return common.HandlerConfig{
		IsEnabled: h.config.IsEnabled,
		Priority:  h.config.Priority,
		Name:      h.config.Name,
	}
}

// Name возвращает имя обработчика
func (h *NumericForLoopHandler) Name() string {
	return h.config.Name
}
