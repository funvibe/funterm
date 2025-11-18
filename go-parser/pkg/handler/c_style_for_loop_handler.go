package handler

import (
	"fmt"

	"go-parser/pkg/ast"
	"go-parser/pkg/common"
	"go-parser/pkg/config"
	"go-parser/pkg/lexer"
	"go-parser/pkg/stream"
)

// protoRecursionGuard реализует защиту от рекурсии для временных контекстов
type protoRecursionGuard struct {
	maxDepth     int
	currentDepth int
}

func (rg *protoRecursionGuard) Enter() error {
	if rg.currentDepth >= rg.maxDepth {
		return fmt.Errorf("maximum recursion depth exceeded: %d", rg.maxDepth)
	}
	rg.currentDepth++
	return nil
}

func (rg *protoRecursionGuard) Exit() {
	if rg.currentDepth > 0 {
		rg.currentDepth--
	}
}

func (rg *protoRecursionGuard) CurrentDepth() int {
	return rg.currentDepth
}

func (rg *protoRecursionGuard) MaxDepth() int {
	return rg.maxDepth
}

// CStyleForLoopHandler - обработчик C-style for циклов
type CStyleForLoopHandler struct {
	config  config.ConstructHandlerConfig
	verbose bool
}

// NewCStyleForLoopHandler создает новый обработчик C-style for циклов
func NewCStyleForLoopHandler(config config.ConstructHandlerConfig) *CStyleForLoopHandler {
	return NewCStyleForLoopHandlerWithVerbose(config, false)
}

// NewCStyleForLoopHandlerWithVerbose создает новый обработчик C-style for циклов с поддержкой verbose режима
func NewCStyleForLoopHandlerWithVerbose(config config.ConstructHandlerConfig, verbose bool) *CStyleForLoopHandler {
	return &CStyleForLoopHandler{
		config:  config,
		verbose: verbose,
	}
}

// CanHandle проверяет, может ли обработчик обработать токен
func (h *CStyleForLoopHandler) CanHandle(token lexer.Token) bool {
	// Обрабатываем токен 'for'
	if token.Type != lexer.TokenFor {
		return false
	}

	// Для более точного определения нужно проверить структуру токенов
	// Но у нас нет доступа к tokenStream здесь, поэтому возвращаем true
	// и будем делать детальную проверку в Handle()
	return true
}

// Handle обрабатывает C-style for цикл
func (h *CStyleForLoopHandler) Handle(ctx *common.ParseContext) (interface{}, error) {
	// Проверяем защиту от рекурсии
	if err := ctx.Guard.Enter(); err != nil {
		return nil, err
	}
	defer ctx.Guard.Exit()

	tokenStream := ctx.TokenStream

	if h.verbose {
		fmt.Printf("DEBUG: CStyleForLoopHandler.Handle() called\n")
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
	// C-style цикл: for (init; condition; increment) { ... } или for init; condition; increment { ... }
	if !tokenStream.HasMore() {
		return nil, newErrorWithPos(tokenStream, "expected '(' or initializer after 'for' - no more tokens")
	}

	// Если все проверки прошли, потребляем токены
	tokenStream.Consume() // потребляем 'for'

	// 3. Проверяем, есть ли открывающая скобка (обязательно для C-style циклов)
	var lParenToken lexer.Token
	var rParenToken lexer.Token
	hasParens := false

	if tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenLeftParen {
		// Скобки есть - это C-style цикл, потребляем '('
		lParenToken = tokenStream.Current()
		tokenStream.Consume() // потребляем '('
		hasParens = true
	} else {
		// Скобок нет - поддерживаем вариант без скобок: for init; condition; increment { ... }
		hasParens = false
	}

	// 4. Парсим три части цикла: инициализация, условие, инкремент
	initializer, condition, increment, err := h.parseForHeader(tokenStream, hasParens)
	if err != nil {
		return nil, newErrorWithPos(tokenStream, "failed to parse for header: %v", err)
	}

	// 5. Проверяем и потребляем токен ')' (только если были открывающие скобки)
	if hasParens {
		if h.verbose {
			fmt.Printf("DEBUG: Before checking ')', current token: %s at line %d col %d\n",
				tokenStream.Current().Type, tokenStream.Current().Line, tokenStream.Current().Column)
		}
		if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenRightParen {
			return nil, newErrorWithPos(tokenStream, "expected ')' after for header, got %s", tokenStream.Current().Type)
		}
		rParenToken = tokenStream.Consume()
	} else {
		// Создаем пустой токен для закрывающей скобки
		rParenToken = lexer.Token{
			Type:     lexer.TokenRightParen,
			Value:    "",
			Line:     forToken.Line,
			Column:   forToken.Column,
			Position: forToken.Position,
		}
	}

	// 6. Проверяем токен '{'
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenLBrace {
		return nil, newErrorWithPos(tokenStream, "expected '{' after for header")
	}
	lBraceToken := tokenStream.Consume()

	// 7. Читаем тело цикла
	body, err := h.parseLoopBody(ctx)
	if err != nil {
		return nil, newErrorWithPos(tokenStream, "failed to parse loop body: %v", err)
	}

	// 8. Проверяем токен '}'
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenRBrace {
		return nil, newErrorWithPos(tokenStream, "expected '}' after loop body")
	}
	rBraceToken := tokenStream.Consume()

	// 9. Создаем узел AST
	loopNode := ast.NewCStyleForLoopStatement(
		forToken, lParenToken, rParenToken, lBraceToken, rBraceToken,
		initializer, condition, increment, body,
	)

	return loopNode, nil
}

// parseForHeader парсит заголовок C-style for цикла: (init; condition; increment) или init; condition; increment
func (h *CStyleForLoopHandler) parseForHeader(tokenStream stream.TokenStream, hasParens bool) (ast.Statement, ast.Expression, ast.Expression, error) {
	var initializer ast.Statement
	var condition ast.Expression
	var increment ast.Expression
	var err error

	// 1. Парсим инициализацию (может быть пустой)
	initializer, err = h.parseInitializer(tokenStream)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to parse initializer: %v", err)
	}

	// 2. Проверяем и потребляем ';'
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenSemicolon {
		return nil, nil, nil, fmt.Errorf("expected ';' after initializer")
	}
	tokenStream.Consume() // потребляем ';'

	// 3. Парсим условие (может быть пустым)
	condition, err = h.parseCondition(tokenStream)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to parse condition: %v", err)
	}

	// 4. Проверяем и потребляем ';'
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenSemicolon {
		return nil, nil, nil, fmt.Errorf("expected ';' after condition")
	}
	tokenStream.Consume() // потребляем ';'

	// 5. Парсим инкремент (может быть пустым)
	increment, err = h.parseIncrement(tokenStream, hasParens)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to parse increment: %v", err)
	}

	return initializer, condition, increment, nil
}

// parseInitializer парсит инициализацию (может быть пустой)
func (h *CStyleForLoopHandler) parseInitializer(tokenStream stream.TokenStream) (ast.Statement, error) {
	if h.verbose {
		fmt.Printf("DEBUG: parseInitializer called, current token: %v\n", tokenStream.Current())
	}

	// Если следующий токен - ';', то инициализация пустая
	if tokenStream.Current().Type == lexer.TokenSemicolon {
		return nil, nil
	}

	// Пробуем распарсить как присваивание переменной
	// Создаем временный контекст для парсинга
	tempCtx := &common.ParseContext{
		TokenStream: tokenStream,
		Parser:      nil,
		Depth:       0,
		MaxDepth:    100,
		Guard:       &protoRecursionGuard{maxDepth: 100, currentDepth: 0},
		LoopDepth:   0,
		InputStream: "", // Не нужен для парсинга выражений
	}

	// Используем AssignmentHandler для парсинга инициализации
	assignmentHandler := NewAssignmentHandler(0, 0)
	result, err := assignmentHandler.Handle(tempCtx)
	if err != nil {
		return nil, newErrorWithPos(tokenStream, "failed to parse initializer as assignment: %v", err)
	}

	if stmt, ok := result.(ast.Statement); ok {
		return stmt, nil
	}

	return nil, newErrorWithPos(tokenStream, "initializer is not a statement")
}

// parseCondition парсит условие (может быть пустым)
func (h *CStyleForLoopHandler) parseCondition(tokenStream stream.TokenStream) (ast.Expression, error) {
	if h.verbose {
		fmt.Printf("DEBUG: parseCondition called, current token: %v\n", tokenStream.Current())
	}

	// Если следующий токен - ';', то условие пустое
	if tokenStream.Current().Type == lexer.TokenSemicolon {
		return nil, nil
	}

	// Используем более простой подход для парсинга условия
	// Пробуем распарсить как простое выражение
	result, err := h.parseSimpleExpression(tokenStream)
	if err != nil {
		return nil, newErrorWithPos(tokenStream, "failed to parse condition as expression: %v", err)
	}

	return result, nil
}

// parseIncrement парсит инкремент (может быть пустым)
func (h *CStyleForLoopHandler) parseIncrement(tokenStream stream.TokenStream, hasParens bool) (ast.Expression, error) {
	if h.verbose {
		fmt.Printf("DEBUG: parseIncrement called, current token: %v\n", tokenStream.Current())
	}

	// Если следующий токен - ')' (или ';' если нет скобок), то инкремент пустой
	if (hasParens && tokenStream.Current().Type == lexer.TokenRightParen) ||
		(!hasParens && tokenStream.Current().Type == lexer.TokenLBrace) {
		return nil, nil
	}

	// Сохраняем начальную позицию
	startPos := tokenStream.Position()

	// Проверяем, является ли это присваиванием (переменная = выражение или переменная := выражение)
	if tokenStream.Current().Type == lexer.TokenIdentifier ||
		tokenStream.Current().Type == lexer.TokenLua ||
		tokenStream.Current().Type == lexer.TokenPython ||
		tokenStream.Current().Type == lexer.TokenPy ||
		tokenStream.Current().Type == lexer.TokenGo ||
		tokenStream.Current().Type == lexer.TokenNode ||
		tokenStream.Current().Type == lexer.TokenJS {

		// Проверяем следующий токен на наличие оператора присваивания
		if tokenStream.HasMore() {
			nextToken := tokenStream.Peek()
			if nextToken.Type == lexer.TokenAssign || nextToken.Type == lexer.TokenColonEquals {
				// Это присваивание - используем AssignmentHandler
				tempCtx := &common.ParseContext{
					TokenStream: tokenStream,
					Parser:      nil,
					Depth:       0,
					MaxDepth:    100,
					Guard:       &protoRecursionGuard{maxDepth: 100, currentDepth: 0},
					LoopDepth:   0,
					InputStream: "", // Не нужен для парсинга выражений
				}

				assignmentHandler := NewAssignmentHandlerWithVerbose(80, 4, h.verbose)
				result, err := assignmentHandler.Handle(tempCtx)
				if err != nil {
					return nil, newErrorWithPos(tokenStream, "failed to parse increment as assignment: %v", err)
				}

				if stmt, ok := result.(ast.Statement); ok {
					if h.verbose {
						fmt.Printf("DEBUG: parseIncrement parsed assignment: %T, current token after parsing: %v\n", stmt, tokenStream.Current())
					}
					// Преобразуем Statement в Expression для совместимости
					if assignStmt, ok := stmt.(*ast.VariableAssignment); ok {
						// Создаем BinaryExpression для присваивания
						operator := "="
						if assignStmt.IsMutable {
							operator = ":="
						}
						return &ast.BinaryExpression{
							Left:     &ast.Identifier{Name: assignStmt.Variable.Name, Qualified: assignStmt.Variable.Qualified, Language: assignStmt.Variable.Language, Pos: assignStmt.Position()},
							Operator: operator,
							Right:    assignStmt.Value,
							Pos: ast.Position{
								Line:   assignStmt.Position().Line,
								Column: assignStmt.Position().Column,
								Offset: assignStmt.Position().Offset,
							},
						}, nil
					}
				}

				return nil, newErrorWithPos(tokenStream, "increment assignment is not a VariableAssignment")
			}
		}
	}

	// Если это не присваивание, пробуем распарсить как простое выражение
	result, err := h.parseSimpleExpression(tokenStream)
	if err != nil {
		return nil, newErrorWithPos(tokenStream, "failed to parse increment as expression: %v", err)
	}

	if h.verbose {
		fmt.Printf("DEBUG: parseIncrement parsed expression: %T, current token after parsing: %v\n", result, tokenStream.Current())
	}

	// Проверяем, что мы продвинулись в потоке
	if tokenStream.Position() == startPos {
		return nil, newErrorWithPos(tokenStream, "parseIncrement: no progress made parsing increment expression")
	}

	return result, nil
}

// parseLoopBody парсит тело цикла
func (h *CStyleForLoopHandler) parseLoopBody(ctx *common.ParseContext) ([]ast.Statement, error) {
	tokenStream := ctx.TokenStream
	body := make([]ast.Statement, 0)

	if h.verbose {
		fmt.Printf("DEBUG parseLoopBody: Starting to parse loop body\n")
		fmt.Printf("DEBUG parseLoopBody: Current token: %s(%s)\n", tokenStream.Current().Type, tokenStream.Current().Value)
	}

	// Пропускаем пробелы после '{'
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
				fmt.Printf("DEBUG parseLoopBody: Found }, stopping body parsing\n")
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
			// Пробуем C-style первым
			nestedForHandler := NewCStyleForLoopHandler(config.ConstructHandlerConfig{})
			result, err := nestedForHandler.Handle(ctx)
			if err != nil {
				// Если не получилось, пробуем другие типы for циклов
				tokenStream.SetPosition(tokenStream.Position() - 1) // Откатываемся к токену 'for'

				// Пробуем NumericForLoop
				numericForHandler := NewNumericForLoopHandler(config.ConstructHandlerConfig{})
				result, err = numericForHandler.Handle(ctx)
				if err != nil {
					// Пробуем ForInLoop
					tokenStream.SetPosition(tokenStream.Position() - 1) // Откатываемся к токену 'for'
					forInHandler := NewForInLoopHandler(config.ConstructHandlerConfig{})
					result, err = forInHandler.Handle(ctx)
					if err != nil {
						return nil, newErrorWithPos(tokenStream, "failed to parse nested for loop: %v", err)
					}
				}
			}

			if nestedLoop, ok := result.(ast.Statement); ok {
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

		// Обрабатываем if выражения
		if current.Type == lexer.TokenIf {
			ifHandler := NewIfHandlerWithVerbose(config.ConstructHandlerConfig{}, h.verbose)
			result, err := ifHandler.Handle(ctx)
			if err != nil {
				return nil, newErrorWithPos(tokenStream, "failed to parse if statement: %v", err)
			}
			if ifStmt, ok := result.(*ast.IfStatement); ok {
				body = append(body, ifStmt)
				continue
			}
		}

		// Обрабатываем вызовы функций других языков
		if current.Type == lexer.TokenIdentifier || current.Type == lexer.TokenLua ||
			current.Type == lexer.TokenPython || current.Type == lexer.TokenPy || current.Type == lexer.TokenGo ||
			current.Type == lexer.TokenNode || current.Type == lexer.TokenJS {

			// Проверяем, не является ли это присваиванием
			if tokenStream.Peek().Type == lexer.TokenAssign || tokenStream.Peek().Type == lexer.TokenColonEquals {
				assignmentHandler := NewAssignmentHandlerWithVerbose(80, 4, h.verbose)
				result, err := assignmentHandler.Handle(ctx)
				if err == nil {
					if assignment, ok := result.(*ast.VariableAssignment); ok {
						body = append(body, assignment)
						continue
					}
				}
			}

			// Проверяем, не является ли это вызовом функции
			if tokenStream.Peek().Type == lexer.TokenDot {
				languageCallHandler := NewLanguageCallHandler(config.ConstructHandlerConfig{})
				result, err := languageCallHandler.Handle(ctx)
				if err == nil {
					if call, ok := result.(*ast.LanguageCall); ok {
						languageCallStmt := &ast.LanguageCallStatement{
							LanguageCall: call,
							IsBackground: false,
						}
						body = append(body, languageCallStmt)
						continue
					}
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
func (h *CStyleForLoopHandler) Config() common.HandlerConfig {
	return common.HandlerConfig{
		IsEnabled: h.config.IsEnabled,
		Priority:  h.config.Priority,
		Order:     h.config.Order,
		Name:      h.config.Name,
	}
}

// Name возвращает имя обработчика
func (h *CStyleForLoopHandler) Name() string {
	return h.config.Name
}

// parseSimpleExpression парсит простое выражение (идентификатор, число, или бинарное выражение)
func (h *CStyleForLoopHandler) parseSimpleExpression(tokenStream stream.TokenStream) (ast.Expression, error) {
	if !tokenStream.HasMore() {
		return nil, newErrorWithPos(tokenStream, "unexpected EOF")
	}

	currentToken := tokenStream.Current()

	// Пробуем распарсить как простое выражение
	switch currentToken.Type {
	case lexer.TokenIdentifier:
		// Это может быть переменная или начало бинарного выражения
		tokenStream.Consume()
		leftExpr := ast.NewIdentifier(currentToken, currentToken.Value)

		// Проверяем, есть ли оператор после идентификатора
		if tokenStream.HasMore() {
			nextToken := tokenStream.Current()
			if nextToken.Type == lexer.TokenLess || nextToken.Type == lexer.TokenGreater ||
				nextToken.Type == lexer.TokenLessEqual || nextToken.Type == lexer.TokenGreaterEqual ||
				nextToken.Type == lexer.TokenEqual || nextToken.Type == lexer.TokenNotEqual ||
				nextToken.Type == lexer.TokenPlus || nextToken.Type == lexer.TokenMinus ||
				nextToken.Type == lexer.TokenMultiply || nextToken.Type == lexer.TokenSlash ||
				nextToken.Type == lexer.TokenAssign || nextToken.Type == lexer.TokenColonEquals {
				return h.parseBinaryExpression(leftExpr, tokenStream)
			}
		}

		return leftExpr, nil

	case lexer.TokenNumber:
		// Числовой литерал
		tokenStream.Consume()
		numValue, err := parseNumber(currentToken.Value)
		if err != nil {
			return nil, newErrorWithTokenPos(currentToken, "invalid number format: %s", currentToken.Value)
		}
		leftExpr := createNumberLiteral(currentToken, numValue)

		// Проверяем, есть ли оператор после числа
		if tokenStream.HasMore() {
			nextToken := tokenStream.Current()
			if nextToken.Type == lexer.TokenLess || nextToken.Type == lexer.TokenGreater ||
				nextToken.Type == lexer.TokenLessEqual || nextToken.Type == lexer.TokenGreaterEqual ||
				nextToken.Type == lexer.TokenEqual || nextToken.Type == lexer.TokenNotEqual ||
				nextToken.Type == lexer.TokenPlus || nextToken.Type == lexer.TokenMinus ||
				nextToken.Type == lexer.TokenMultiply || nextToken.Type == lexer.TokenSlash ||
				nextToken.Type == lexer.TokenAssign || nextToken.Type == lexer.TokenColonEquals {
				return h.parseBinaryExpression(leftExpr, tokenStream)
			}
		}

		return leftExpr, nil

	default:
		return nil, newErrorWithTokenPos(currentToken, "unsupported expression type: %s", currentToken.Type)
	}
}

// parseBinaryExpression парсит бинарное выражение с поддержкой цепочек операций
func (h *CStyleForLoopHandler) parseBinaryExpression(leftExpr ast.Expression, tokenStream stream.TokenStream) (ast.Expression, error) {
	if h.verbose {
		fmt.Printf("DEBUG: parseBinaryExpression called with left: %T, current token: %v\n", leftExpr, tokenStream.Current())
	}

	currentExpr := leftExpr

	for tokenStream.HasMore() {
		operatorToken := tokenStream.Current()

		// Проверяем, что это бинарный оператор
		if operatorToken.Type != lexer.TokenLess && operatorToken.Type != lexer.TokenGreater &&
			operatorToken.Type != lexer.TokenLessEqual && operatorToken.Type != lexer.TokenGreaterEqual &&
			operatorToken.Type != lexer.TokenEqual && operatorToken.Type != lexer.TokenNotEqual &&
			operatorToken.Type != lexer.TokenPlus && operatorToken.Type != lexer.TokenMinus &&
			operatorToken.Type != lexer.TokenMultiply && operatorToken.Type != lexer.TokenSlash &&
			operatorToken.Type != lexer.TokenAssign && operatorToken.Type != lexer.TokenColonEquals {
			break
		}

		tokenStream.Consume() // потребляем оператор

		if h.verbose {
			fmt.Printf("DEBUG: parseBinaryExpression consumed operator: %v, current token: %v\n", operatorToken, tokenStream.Current())
		}

		// Парсим правый операнд - это может быть сложное выражение
		rightExpr, err := h.parseSimpleExpression(tokenStream)
		if err != nil {
			return nil, newErrorWithPos(tokenStream, "failed to parse right operand: %v", err)
		}

		// Проверяем, есть ли еще операторы после правого операнда (для цепочек операций)
		for tokenStream.HasMore() {
			nextOperatorToken := tokenStream.Current()

			// Проверяем, что это бинарный оператор
			if nextOperatorToken.Type != lexer.TokenLess && nextOperatorToken.Type != lexer.TokenGreater &&
				nextOperatorToken.Type != lexer.TokenLessEqual && nextOperatorToken.Type != lexer.TokenGreaterEqual &&
				nextOperatorToken.Type != lexer.TokenEqual && nextOperatorToken.Type != lexer.TokenNotEqual &&
				nextOperatorToken.Type != lexer.TokenPlus && nextOperatorToken.Type != lexer.TokenMinus &&
				nextOperatorToken.Type != lexer.TokenMultiply && nextOperatorToken.Type != lexer.TokenSlash {
				break
			}

			// Для оператора присваивания (=) мы не продолжаем цепочку, так как это имеет низкий приоритет
			if operatorToken.Type == lexer.TokenAssign || operatorToken.Type == lexer.TokenColonEquals {
				break
			}

			// Продолжаем парсинг для других операторов
			tokenStream.Consume() // потребляем следующий оператор

			if h.verbose {
				fmt.Printf("DEBUG: parseBinaryExpression continuing with operator: %v, current token: %v\n", nextOperatorToken, tokenStream.Current())
			}

			// Парсим следующий правый операнд
			nextRightExpr, err := h.parseSimpleExpression(tokenStream)
			if err != nil {
				return nil, newErrorWithPos(tokenStream, "failed to parse next right operand: %v", err)
			}

			// Создаем бинарное выражение для следующего оператора
			var nextOperatorStr string
			switch nextOperatorToken.Type {
			case lexer.TokenLess:
				nextOperatorStr = "<"
			case lexer.TokenGreater:
				nextOperatorStr = ">"
			case lexer.TokenLessEqual:
				nextOperatorStr = "<="
			case lexer.TokenGreaterEqual:
				nextOperatorStr = ">="
			case lexer.TokenEqual:
				nextOperatorStr = "=="
			case lexer.TokenNotEqual:
				nextOperatorStr = "!="
			case lexer.TokenPlus:
				nextOperatorStr = "+"
			case lexer.TokenMinus:
				nextOperatorStr = "-"
			case lexer.TokenMultiply:
				nextOperatorStr = "*"
			case lexer.TokenSlash:
				nextOperatorStr = "/"
			default:
				return nil, newErrorWithTokenPos(nextOperatorToken, "unsupported binary operator: %s", nextOperatorToken.Type)
			}

			rightExpr = &ast.BinaryExpression{
				Left:     rightExpr,
				Operator: nextOperatorStr,
				Right:    nextRightExpr,
				Pos: ast.Position{
					Line:   nextOperatorToken.Line,
					Column: nextOperatorToken.Column,
					Offset: nextOperatorToken.Position,
				},
			}

			if h.verbose {
				fmt.Printf("DEBUG: parseBinaryExpression created chained expression: %T, current token: %v\n", rightExpr, tokenStream.Current())
			}
		}

		if h.verbose {
			fmt.Printf("DEBUG: parseBinaryExpression parsed right: %T, current token: %v\n", rightExpr, tokenStream.Current())
		}

		// Создаем бинарное выражение
		var operatorStr string
		switch operatorToken.Type {
		case lexer.TokenLess:
			operatorStr = "<"
		case lexer.TokenGreater:
			operatorStr = ">"
		case lexer.TokenLessEqual:
			operatorStr = "<="
		case lexer.TokenGreaterEqual:
			operatorStr = ">="
		case lexer.TokenEqual:
			operatorStr = "=="
		case lexer.TokenNotEqual:
			operatorStr = "!="
		case lexer.TokenPlus:
			operatorStr = "+"
		case lexer.TokenMinus:
			operatorStr = "-"
		case lexer.TokenMultiply:
			operatorStr = "*"
		case lexer.TokenSlash:
			operatorStr = "/"
		case lexer.TokenAssign:
			operatorStr = "="
		case lexer.TokenColonEquals:
			operatorStr = ":="
		default:
			return nil, newErrorWithTokenPos(operatorToken, "unsupported binary operator: %s", operatorToken.Type)
		}

		// Создаем новое бинарное выражение
		currentExpr = &ast.BinaryExpression{
			Left:     currentExpr,
			Operator: operatorStr,
			Right:    rightExpr,
			Pos: ast.Position{
				Line:   operatorToken.Line,
				Column: operatorToken.Column,
				Offset: operatorToken.Position,
			},
		}

		if h.verbose {
			fmt.Printf("DEBUG: parseBinaryExpression created binary expression: %T, current token: %v\n", currentExpr, tokenStream.Current())
		}

		// Для оператора присваивания мы прекращаем парсинг, так как он имеет самый низкий приоритет
		if operatorToken.Type == lexer.TokenAssign || operatorToken.Type == lexer.TokenColonEquals {
			break
		}
	}

	return currentExpr, nil
}
