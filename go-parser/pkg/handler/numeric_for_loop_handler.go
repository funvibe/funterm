package handler

import (
	"fmt"
	"math/big"

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
	if token.Type != lexer.TokenFor {
		return false
	}

	if h.verbose {
		fmt.Printf("DEBUG: NumericForLoopHandler.CanHandle - returning true for token: %s(%s)\n", token.Type, token.Value)
	}

	// Для более точного определения нужно проверить структуру токенов
	// Но у нас нет доступа к tokenStream здесь, поэтому возвращаем true
	// и будем делать детальную проверку в Handle()
	return true
}

// Handle обрабатывает Lua-style числовой цикл
func (h *NumericForLoopHandler) Handle(ctx *common.ParseContext) (interface{}, error) {
	// Проверяем защиту от рекурсии
	if err := ctx.Guard.Enter(); err != nil {
		return nil, err
	}
	defer ctx.Guard.Exit()

	tokenStream := ctx.TokenStream

	if h.verbose {
		fmt.Printf("DEBUG: NumericForLoopHandler.Handle() called\n")
	}

	// Увеличиваем глубину циклов для контекстной валидации break/continue
	ctx.LoopDepth++
	defer func() {
		ctx.LoopDepth--
	}()

	// 1. Проверяем токен 'for'
	forToken := tokenStream.Current()
	if forToken.Type != lexer.TokenFor {
		return nil, newErrorWithTokenPos(forToken, "expected 'for', got %s", forToken.Type)
	}

	// 2. Проверяем структуру перед потреблением токенов
	// Lua-цикл: for var = start, end, step do ... end
	if !tokenStream.HasMore() {
		return nil, newErrorWithPos(tokenStream, "expected loop variable after 'for' - no more tokens")
	}

	// Если все проверки прошли, это действительно Lua-цикл, потребляем токены
	tokenStream.Consume() // потребляем 'for'

	// Проверяем, что текущий токен - идентификатор (переменная)
	current := tokenStream.Current()
	if h.verbose {
		fmt.Printf("DEBUG NumericForLoopHandler: Current token after 'for': %s(%s)\n", current.Type, current.Value)
	}
	if current.Type != lexer.TokenIdentifier {
		return nil, newErrorWithTokenPos(current, "expected loop variable after 'for', got %s", current.Type)
	}

	// Проверяем, что после переменной идет '=' (признак Lua-цикла)
	// НЕ поддерживаем ':=' в циклах - только '=' как в стандартном Lua
	peek1 := tokenStream.Peek()
	if h.verbose {
		fmt.Printf("DEBUG NumericForLoopHandler: Peek() token: %s(%s)\n", peek1.Type, peek1.Value)
	}
	if peek1.Type != lexer.TokenAssign {
		return nil, newErrorWithTokenPos(peek1, "expected '=' after loop variable, got %s", peek1.Type)
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
		return nil, newErrorWithPos(tokenStream, "failed to parse start value: %v", err)
	}

	// 5. Проверяем токен ','
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenComma {
		return nil, newErrorWithPos(tokenStream, "expected ',' after start value")
	}
	tokenStream.Consume() // потребляем запятую

	// 6. Читаем конечное значение
	end, err := h.parseNumericExpression(tokenStream)
	if err != nil {
		return nil, newErrorWithPos(tokenStream, "failed to parse end value: %v", err)
	}

	// 7. Проверяем опциональный шаг
	var step ast.ProtoNode
	if tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenComma {
		tokenStream.Consume() // потребляем запятую
		step, err = h.parseNumericExpression(tokenStream)
		if err != nil {
			return nil, newErrorWithPos(tokenStream, "failed to parse step value: %v", err)
		}
	}

	// 8. Проверяем токен '{'
	var doToken, endToken lexer.Token
	var body []ast.Statement

	if !tokenStream.HasMore() {
		return nil, newErrorWithPos(tokenStream, "expected '{' after for loop parameters")
	}

	currentToken := tokenStream.Current()
	if currentToken.Type != lexer.TokenLBrace {
		return nil, newErrorWithTokenPos(currentToken, "expected '{' after for loop parameters, got %s", currentToken.Type)
	}

	// Синтаксис: { ... }
	doToken = tokenStream.Consume() // потребляем '{'

	// 9. Читаем тело цикла
	body, err = h.parseLoopBody(ctx)
	if err != nil {
		return nil, newErrorWithPos(tokenStream, "failed to parse loop body: %v", err)
	}

	// 10. Проверяем токен '}'
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenRBrace {
		return nil, newErrorWithPos(tokenStream, "expected '}' after loop body")
	}
	endToken = tokenStream.Consume() // потребляем '}'

	// 11. Создаем узел AST
	loopNode := ast.NewNumericForLoopStatement(forToken, doToken, endToken, variable, start, end, step, body)

	return loopNode, nil
}

// parseNumericExpression парсит числовое выражение
func (h *NumericForLoopHandler) parseNumericExpression(tokenStream stream.TokenStream) (ast.ProtoNode, error) {
	if !tokenStream.HasMore() {
		return nil, newErrorWithPos(tokenStream, "unexpected EOF")
	}

	token := tokenStream.Current()

	// Проверяем на унарный минус
	if token.Type == lexer.TokenMinus {
		tokenStream.Consume() // потребляем минус

		// Рекурсивно парсим выражение после минуса
		expr, err := h.parseNumericExpression(tokenStream)
		if err != nil {
			return nil, newErrorWithPos(tokenStream, "failed to parse expression after unary minus: %v", err)
		}

		// Если это числовой литерал, просто инвертируем значение
		if numLit, ok := expr.(*ast.NumberLiteral); ok {
			if numLit.IsInt {
				// For big.Int, create a new negative value
				negativeInt := new(big.Int).Neg(numLit.IntValue)
				numLit.IntValue = negativeInt
			} else {
				numLit.FloatValue = -numLit.FloatValue
			}
			return numLit, nil
		}

		// Для других типов выражений создаем унарное выражение
		// (пока не реализовано, возвращаем ошибку)
		return nil, newErrorWithPos(tokenStream, "unary minus not supported for non-numeric literals")
	}

	token = tokenStream.Consume()
	switch token.Type {
	case lexer.TokenNumber:
		// Числовой литерал
		value, err := parseNumber(token.Value)
		if err != nil {
			return nil, newErrorWithTokenPos(token, "invalid number format: %s", token.Value)
		}
		return createNumberLiteral(token, value), nil

	case lexer.TokenIdentifier:
		// Идентификатор (переменная)
		return ast.NewIdentifier(token, token.Value), nil

	default:
		return nil, newErrorWithTokenPos(token, "unsupported numeric expression type: %s", token.Type)
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

		// Если встречаем '}', заканчиваем тело цикла
		if current.Type == lexer.TokenRBrace {
			if h.verbose {
				fmt.Printf("DEBUG parseLoopBody: Found %s, stopping body parsing\n", current.Type)
			}
			break
		}

		// Обрабатываем break операторы
		if current.Type == lexer.TokenBreak {
			breakHandler := NewBreakHandler(config.ConstructHandlerConfig{})
			result, err := breakHandler.Handle(ctx)
			if err != nil {
				return nil, newErrorWithPos(tokenStream, "failed to parse break statement: %v", err)
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
				return nil, newErrorWithPos(tokenStream, "failed to parse continue statement: %v", err)
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
				return nil, newErrorWithPos(tokenStream, "failed to parse nested for loop: %v", err)
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
				return nil, newErrorWithPos(tokenStream, "failed to parse while loop: %v", err)
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
				return nil, newErrorWithPos(tokenStream, "failed to parse match statement: %v", err)
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
			ifHandler := NewIfHandlerWithVerbose(config.ConstructHandlerConfig{}, h.verbose)
			result, err := ifHandler.Handle(ctx)
			if h.verbose {
				fmt.Printf("DEBUG parseLoopBody: IfHandler result: %v, error: %v\n", result, err)
			}
			if err != nil {
				return nil, newErrorWithPos(tokenStream, "failed to parse if statement: %v", err)
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
			if (current.Type == lexer.TokenJS || current.Type == lexer.TokenLua || current.Type == lexer.TokenPython ||
				current.Type == lexer.TokenPy || current.Type == lexer.TokenGo || current.Type == lexer.TokenNode) &&
				tokenStream.Peek().Type == lexer.TokenDot {
				// Проверяем токен после DOT
				if tokenStream.PeekN(2).Type == lexer.TokenIdentifier {
					// Проверяем, есть ли ASSIGN после идентификатора
					identifierAfterDot := tokenStream.PeekN(2)
					if tokenStream.PeekN(3).Type == lexer.TokenAssign || tokenStream.PeekN(3).Type == lexer.TokenColonEquals {
						// Это присваивание типа lua.value = ...
						if h.verbose {
							fmt.Printf("DEBUG parseLoopBody: Detected assignment pattern: %s.%s = ...\n", current.Value, identifierAfterDot.Value)
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
			if peekForAssign.Type == lexer.TokenAssign || peekForAssign.Type == lexer.TokenColonEquals {
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

		// Если не смогли распарсить, пропускаем токен
		tokenStream.Consume()
	}

	if h.verbose {
		fmt.Printf("DEBUG parseLoopBody: Finished parsing body, current token: %s(%s)\n", tokenStream.Current().Type, tokenStream.Current().Value)
		fmt.Printf("DEBUG parseLoopBody: Body length: %d\n", len(body))
	}

	return body, nil
}

// parseSimpleIfStatement парсит простую if конструкцию для использования внутри циклов
// Поддерживает синтаксис: if condition statement end
func (h *NumericForLoopHandler) parseSimpleIfStatement(ctx *common.ParseContext) (ast.Statement, error) {
	tokenStream := ctx.TokenStream

	// Потребляем 'if'
	ifToken := tokenStream.Consume()
	if ifToken.Type != lexer.TokenIf {
		return nil, fmt.Errorf("expected 'if', got %s", ifToken.Type)
	}

	// Парсим условие
	condition, err := h.parseCondition(tokenStream)
	if err != nil {
		return nil, fmt.Errorf("failed to parse condition: %v", err)
	}

	// Читаем тело if (одно выражение)
	bodyStatements := make([]ast.Statement, 0)

	for tokenStream.HasMore() {
		current := tokenStream.Current()

		// Если встречаем '}', заканчиваем тело
		if current.Type == lexer.TokenRBrace {
			break
		}

		// Обрабатываем break
		if current.Type == lexer.TokenBreak {
			tokenStream.Consume()
			breakStmt := ast.NewBreakStatement(current)
			bodyStatements = append(bodyStatements, breakStmt)
			continue
		}

		// Обрабатываем continue
		if current.Type == lexer.TokenContinue {
			tokenStream.Consume()
			continueStmt := ast.NewContinueStatement(current)
			bodyStatements = append(bodyStatements, continueStmt)
			continue
		}

		// Обрабатываем вызовы функций
		if current.Type == lexer.TokenIdentifier || current.Type == lexer.TokenLua ||
			current.Type == lexer.TokenPython || current.Type == lexer.TokenPy || current.Type == lexer.TokenGo ||
			current.Type == lexer.TokenNode || current.Type == lexer.TokenJS {

			if tokenStream.Peek().Type == lexer.TokenDot {
				// Это вызов функции вида js.print
				languageCallHandler := NewLanguageCallHandler(config.ConstructHandlerConfig{})
				result, err := languageCallHandler.Handle(ctx)
				if err == nil {
					if call, ok := result.(*ast.LanguageCall); ok {
						languageCallStmt := &ast.LanguageCallStatement{
							LanguageCall: call,
							IsBackground: false,
						}
						bodyStatements = append(bodyStatements, languageCallStmt)
						continue
					}
				}
			}
		}

		// Если не смогли распарсить, пропускаем токен
		tokenStream.Consume()
	}

	// Проверяем и потребляем '}'
	if !tokenStream.HasMore() {
		return nil, fmt.Errorf("expected '}' after if body")
	}

	endToken := tokenStream.Current()
	if endToken.Type != lexer.TokenRBrace {
		return nil, fmt.Errorf("expected '}' after if body, got %s", endToken.Type)
	}
	tokenStream.Consume()

	// Создаем узел AST для if
	blockStatement := ast.NewBlockStatement(ifToken, endToken, bodyStatements)
	ifNode := ast.NewIfStatement(ifToken, lexer.Token{}, lexer.Token{}, condition, blockStatement)

	return ifNode, nil
}

// parseCondition парсит условие для if
func (h *NumericForLoopHandler) parseCondition(tokenStream stream.TokenStream) (ast.Expression, error) {
	if !tokenStream.HasMore() {
		return nil, fmt.Errorf("unexpected EOF in condition")
	}

	// Парсим левый операнд
	leftToken := tokenStream.Current()
	var leftExpr ast.Expression

	// Handle parenthesized expressions
	if leftToken.Type == lexer.TokenLeftParen {
		tokenStream.Consume() // Consume '('

		// Parse the inner expression recursively
		expr, err := h.parseCondition(tokenStream)
		if err != nil {
			return nil, newErrorWithPos(tokenStream, "failed to parse parenthesized condition: %v", err)
		}

		// Expect closing ')'
		if !tokenStream.HasMore() {
			return nil, newErrorWithPos(tokenStream, "expected ')' after condition")
		}

		closingToken := tokenStream.Current()
		if closingToken.Type != lexer.TokenRightParen {
			return nil, newErrorWithTokenPos(closingToken, "expected ')' after condition, got %s", closingToken.Value)
		}

		tokenStream.Consume() // Consume ')'
		leftExpr = expr
	} else {
		// Parse regular expression
		tokenStream.Consume() // Consume the token

		switch leftToken.Type {
		case lexer.TokenIdentifier:
			leftExpr = ast.NewIdentifier(leftToken, leftToken.Value)
		case lexer.TokenNumber:
			value, err := parseNumber(leftToken.Value)
			if err != nil {
				return nil, newErrorWithTokenPos(leftToken, "invalid number format: %s", leftToken.Value)
			}
			leftExpr = createNumberLiteral(leftToken, value)
		default:
			return nil, newErrorWithTokenPos(leftToken, "unsupported condition type: %s", leftToken.Type)
		}
	}

	// Проверяем, есть ли оператор сравнения
	if !tokenStream.HasMore() {
		return leftExpr, nil
	}

	operatorToken := tokenStream.Current()
	if operatorToken.Type != lexer.TokenEqual && operatorToken.Type != lexer.TokenNotEqual &&
		operatorToken.Type != lexer.TokenLess && operatorToken.Type != lexer.TokenGreater &&
		operatorToken.Type != lexer.TokenLessEqual && operatorToken.Type != lexer.TokenGreaterEqual {
		return leftExpr, nil
	}

	// Потребляем оператор
	tokenStream.Consume()

	// Парсим правый операнд
	if !tokenStream.HasMore() {
		return nil, newErrorWithPos(tokenStream, "expected right operand after operator")
	}

	rightToken := tokenStream.Consume()
	var rightExpr ast.Expression

	switch rightToken.Type {
	case lexer.TokenIdentifier:
		rightExpr = ast.NewIdentifier(rightToken, rightToken.Value)
	case lexer.TokenNumber:
		value, err := parseNumber(rightToken.Value)
		if err != nil {
			return nil, newErrorWithTokenPos(rightToken, "invalid number format: %s", rightToken.Value)
		}
		rightExpr = createNumberLiteral(rightToken, value)
	default:
		return nil, newErrorWithTokenPos(rightToken, "unsupported condition type: %s", rightToken.Type)
	}

	// Создаем бинарное выражение
	return &ast.BinaryExpression{
		Left:     leftExpr,
		Operator: operatorToken.Value,
		Right:    rightExpr,
		Pos: ast.Position{
			Line:   leftToken.Line,
			Column: leftToken.Column,
			Offset: leftToken.Position,
		},
	}, nil
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
