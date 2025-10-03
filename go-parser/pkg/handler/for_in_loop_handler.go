package handler

import (
	"fmt"
	"strconv"

	"go-parser/pkg/ast"
	"go-parser/pkg/common"
	"go-parser/pkg/config"
	"go-parser/pkg/lexer"
)

// ForInLoopHandler - обработчик Python-style for-in циклов
type ForInLoopHandler struct {
	config  config.ConstructHandlerConfig
	verbose bool
}

// NewForInLoopHandler создает новый обработчик for-in циклов
func NewForInLoopHandler(config config.ConstructHandlerConfig) *ForInLoopHandler {
	return NewForInLoopHandlerWithVerbose(config, false)
}

// NewForInLoopHandlerWithVerbose создает новый обработчик for-in циклов с поддержкой verbose режима
func NewForInLoopHandlerWithVerbose(config config.ConstructHandlerConfig, verbose bool) *ForInLoopHandler {
	return &ForInLoopHandler{
		config:  config,
		verbose: verbose,
	}
}

// CanHandle проверяет, может ли обработчик обработать токен
func (h *ForInLoopHandler) CanHandle(token lexer.Token) bool {
	// Обрабатываем токен 'for'
	return token.Type == lexer.TokenFor
}

// Handle обрабатывает Python-style for-in цикл
func (h *ForInLoopHandler) Handle(ctx *common.ParseContext) (interface{}, error) {
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

	// 1. Проверяем и потребляем токен 'for'
	forToken := tokenStream.Current()
	if forToken.Type != lexer.TokenFor {
		return nil, fmt.Errorf("expected 'for', got %s", forToken.Type)
	}

	// 2. Проверяем структуру перед потреблением токенов
	// Python-цикл: for var in iterable: ...
	// Lua-цикл: for i,v in ipairs({...}) do ... end
	if !tokenStream.HasMore() {
		return nil, fmt.Errorf("expected loop variable after 'for' - no more tokens")
	}

	// Проверяем, что следующий токен - идентификатор (переменная)
	peekToken := tokenStream.Peek()
	if peekToken.Type != lexer.TokenIdentifier {
		return nil, fmt.Errorf("expected loop variable after 'for'")
	}

	// Если все проверки прошли, потребляем токены
	tokenStream.Consume() // потребляем 'for'

	// 2. Читаем переменную(ые) цикла
	var variables []*ast.Identifier
	varToken := tokenStream.Current()
	varToken = tokenStream.Consume()
	variables = append(variables, ast.NewIdentifier(varToken, varToken.Value))

	// Проверяем наличие дополнительных переменных через запятую (Lua-стиль: i,v)
	for tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenComma {
		// Потребляем запятую
		tokenStream.Consume()

		// Проверяем наличие следующей переменной
		if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenIdentifier {
			return nil, fmt.Errorf("expected variable after comma")
		}

		// Читаем следующую переменную
		nextVarToken := tokenStream.Consume()
		variables = append(variables, ast.NewIdentifier(nextVarToken, nextVarToken.Value))
	}

	// 3. Проверяем и потребляем токен 'in'
	var inToken lexer.Token
	currentToken := tokenStream.Current()
	if currentToken.Type != lexer.TokenIn {
		return nil, fmt.Errorf("expected 'in' after loop variable(s), got %s", currentToken.Type)
	}

	inToken = tokenStream.Consume()

	// 4. Читаем итерируемое выражение
	// Пока поддерживаем только простые случаи: идентификаторы и вызовы функций
	var iterable ast.ProtoNode
	if !tokenStream.HasMore() {
		return nil, fmt.Errorf("expected iterable after 'in'")
	}

	currentToken = tokenStream.Current()
	switch currentToken.Type {
	case lexer.TokenIdentifier:
		// Это может быть вызов функции, простой идентификатор или qualified variable
		if tokenStream.Peek().Type == lexer.TokenLeftParen {
			// Это вызов функции, делегируем LanguageCallHandler
			return h.handleFunctionCallAsIterable(ctx, variables[0], inToken)
		} else if tokenStream.Peek().Type == lexer.TokenDot {
			// Это qualified variable (python.my_list)
			return h.handleQualifiedVariableAsIterable(ctx, variables[0], inToken)
		} else {
			// Простой идентификатор
			tokenStream.Consume()
			iterable = ast.NewIdentifier(currentToken, currentToken.Value)
		}

	case lexer.TokenLBracket:
		// Массив как итерируемый объект - пока не поддерживаем как Expression
		return nil, fmt.Errorf("arrays as iterables are not yet supported")

	case lexer.TokenLBrace:
		// Объект как итерируемый объект - пока не поддерживаем как Expression
		return nil, fmt.Errorf("objects as iterables are not yet supported")

	default:
		// Проверяем, является ли токен языковым токеном
		if currentToken.IsLanguageToken() {
			// Check if this is the start of a qualified variable like py.numbers
			if tokenStream.Peek().Type == lexer.TokenDot {
				return h.handleQualifiedVariableAsIterable(ctx, variables[0], inToken)
			} else {
				return nil, fmt.Errorf("expected '.' after language token '%s'", currentToken.Type)
			}
		}
		return nil, fmt.Errorf("unsupported iterable type: %s", currentToken.Type)

	}

	// 5. Проверяем токен ':'
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenColon {
		return nil, fmt.Errorf("expected ':' after iterable")
	}
	colonToken := tokenStream.Consume()

	// 6. Читаем тело цикла
	body, err := h.parseLoopBody(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to parse loop body: %v", err)
	}

	// 7. Создаем узел AST
	// Для множественных переменных используем первую как основную
	loopNode := ast.NewForInLoopStatement(forToken, inToken, colonToken, variables[0], iterable, body)

	return loopNode, nil
}

// handleFunctionCallAsIterable обрабатывает вызов функции как итерируемый объект
func (h *ForInLoopHandler) handleFunctionCallAsIterable(ctx *common.ParseContext, variable *ast.Identifier, inToken lexer.Token) (interface{}, error) {
	tokenStream := ctx.TokenStream

	// Сохраняем текущую позицию для восстановления
	currentPos := tokenStream.Position()

	// Сначала пробуем разобрать как обычный вызов функции (без префикса языка)
	// Это для случаев вроде ipairs(), pairs() и т.д.
	functionCall, err := h.parseBareFunctionCall(ctx)
	if err == nil {
		// Проверяем токен ':' для Python-стиля
		if tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenColon {
			colonToken := tokenStream.Consume()

			// Читаем тело цикла
			body, err := h.parseLoopBody(ctx)
			if err != nil {
				tokenStream.SetPosition(currentPos)
				return nil, fmt.Errorf("failed to parse loop body: %v", err)
			}

			// Восстанавливаем токен 'for'
			forToken := lexer.Token{
				Type:     lexer.TokenFor,
				Value:    "for",
				Line:     variable.Token.Line,
				Column:   variable.Token.Column - 5, // Приблизительная позиция
				Position: variable.Token.Position - 5,
			}

			// Создаем узел AST с вызовом функции
			loopNode := ast.NewForInLoopStatement(forToken, inToken, colonToken, variable, functionCall, body)
			return loopNode, nil
		}

		// Проверяем токен 'do' для Lua-стиля
		if tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenDo {
			doToken := tokenStream.Consume()

			// Читаем тело цикла
			body, err := h.parseLoopBody(ctx)
			if err != nil {
				tokenStream.SetPosition(currentPos)
				return nil, fmt.Errorf("failed to parse loop body: %v", err)
			}

			// Восстанавливаем токен 'for'
			forToken := lexer.Token{
				Type:     lexer.TokenFor,
				Value:    "for",
				Line:     variable.Token.Line,
				Column:   variable.Token.Column - 5, // Приблизительная позиция
				Position: variable.Token.Position - 5,
			}

			// Создаем узел AST с вызовом функции
			loopNode := ast.NewForInLoopStatement(forToken, inToken, doToken, variable, functionCall, body)
			return loopNode, nil
		}
	}

	// Если не удалось разобрать как голый вызов, пробуем как LanguageCall
	tokenStream.SetPosition(currentPos)
	languageCallHandler := NewLanguageCallHandler(config.ConstructHandlerConfig{})
	callResult, err := languageCallHandler.Handle(ctx)
	if err != nil {
		// Восстанавливаем позицию при ошибке
		tokenStream.SetPosition(currentPos)
		return nil, fmt.Errorf("failed to parse function call as iterable: %v", err)
	}

	// Проверяем, что результат - LanguageCall
	languageCall, ok := callResult.(*ast.LanguageCall)
	if !ok {
		tokenStream.SetPosition(currentPos)
		return nil, fmt.Errorf("expected language call as iterable")
	}

	// Проверяем токен ':'
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenColon {
		tokenStream.SetPosition(currentPos)
		return nil, fmt.Errorf("expected ':' after function call")
	}
	colonToken := tokenStream.Consume()

	// Читаем тело цикла
	body, err := h.parseLoopBody(ctx)
	if err != nil {
		tokenStream.SetPosition(currentPos)
		return nil, fmt.Errorf("failed to parse loop body: %v", err)
	}

	// Восстанавливаем токен 'for'
	forToken := lexer.Token{
		Type:     lexer.TokenFor,
		Value:    "for",
		Line:     variable.Token.Line,
		Column:   variable.Token.Column - 5, // Приблизительная позиция
		Position: variable.Token.Position - 5,
	}

	// Создаем узел AST
	loopNode := ast.NewForInLoopStatement(forToken, inToken, colonToken, variable, languageCall, body)

	return loopNode, nil
}

// parseBareFunctionCall парсит вызов функции без префикса языка (например, ipairs({...}))
func (h *ForInLoopHandler) parseBareFunctionCall(ctx *common.ParseContext) (ast.Expression, error) {
	tokenStream := ctx.TokenStream

	// 1. Читаем имя функции
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenIdentifier {
		return nil, fmt.Errorf("expected function name")
	}
	functionToken := tokenStream.Consume()
	functionName := functionToken.Value

	// 2. Проверяем и потребляем открывающую скобку
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenLeftParen {
		return nil, fmt.Errorf("expected '(' after function name '%s'", functionName)
	}
	tokenStream.Consume() // Consuming '('

	// 3. Читаем аргументы
	arguments := make([]ast.Expression, 0)

	// Проверяем, есть ли аргументы
	if !tokenStream.HasMore() {
		return nil, fmt.Errorf("unexpected EOF after argument")
	}

	if tokenStream.Current().Type != lexer.TokenRightParen {
		// Есть хотя бы один аргумент
		for {
			if !tokenStream.HasMore() {
				if len(arguments) == 0 {
					return nil, fmt.Errorf("unexpected EOF after argument")
				} else {
					return nil, fmt.Errorf("unexpected EOF in arguments")
				}
			}

			// Читаем аргумент
			argToken := tokenStream.Current()
			var arg ast.Expression

			switch argToken.Type {
			case lexer.TokenString:
				tokenStream.Consume()
				arg = &ast.StringLiteral{Value: argToken.Value, Pos: tokenToPosition(argToken)}
			case lexer.TokenNumber:
				tokenStream.Consume()
				numValue, _ := strconv.ParseFloat(argToken.Value, 64)
				arg = &ast.NumberLiteral{Value: numValue, Pos: tokenToPosition(argToken)}
			case lexer.TokenLBracket:
				// Массив как аргумент
				arrayHandler := NewArrayHandler(10, 1)
				result, err := arrayHandler.Handle(ctx)
				if err != nil {
					return nil, fmt.Errorf("failed to parse array argument: %v", err)
				}
				if arrayExpr, ok := result.(ast.Expression); ok {
					arg = arrayExpr
				} else {
					return nil, fmt.Errorf("expected array expression, got %T", result)
				}
			case lexer.TokenLBrace:
				// В Lua фигурные скобки используются для массивов, а не объектов
				luaArray, err := h.parseLuaArray(ctx)
				if err != nil {
					return nil, fmt.Errorf("failed to parse Lua array: %v", err)
				}
				arg = luaArray
			case lexer.TokenIdentifier:
				// Простой идентификатор
				tokenStream.Consume()
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

	// 4. Проверяем закрывающую скобку
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenRightParen {
		return nil, fmt.Errorf("expected ')' after arguments")
	}
	tokenStream.Consume() // Consuming ')'

	// 5. Создаем узел AST для вызова функции
	// Для простоты используем LanguageCall с пустым языком (позже можно будет создать специальный тип)
	startPos := tokenToPosition(functionToken)

	node := &ast.LanguageCall{
		Language:  "", // Пустой язык для голых вызовов
		Function:  functionName,
		Arguments: arguments,
		Pos:       startPos,
	}

	return node, nil
}

// parseLuaArray парсит Lua-массив в фигурных скобках
func (h *ForInLoopHandler) parseLuaArray(ctx *common.ParseContext) (ast.Expression, error) {
	tokenStream := ctx.TokenStream

	// Потребляем открывающую фигурную скобку
	openBrace := tokenStream.Consume()
	if openBrace.Type != lexer.TokenLBrace {
		return nil, fmt.Errorf("expected '{', got %s", openBrace.Type)
	}

	// Создаем узел массива (используем ArrayLiteral для представления Lua-массива)
	arrayNode := ast.NewArrayLiteral(openBrace, lexer.Token{})

	// Обрабатываем элементы до закрывающей скобки
	for tokenStream.HasMore() {
		current := tokenStream.Current()

		if current.Type == lexer.TokenRBrace {
			// Потребляем закрывающую скобку и завершаем
			closeBrace := tokenStream.Consume()
			arrayNode.RightBracket = closeBrace
			return arrayNode, nil
		}

		// Пропускаем запятые между элементами
		if current.Type == lexer.TokenComma {
			tokenStream.Consume()
			continue
		}

		// Пытаемся обработать элемент как выражение
		elementToken := tokenStream.Consume()

		var element ast.Expression
		switch elementToken.Type {
		case lexer.TokenString:
			element = &ast.StringLiteral{
				Value: elementToken.Value,
				Pos: ast.Position{
					Line:   elementToken.Line,
					Column: elementToken.Column,
					Offset: elementToken.Position,
				},
			}
		case lexer.TokenNumber:
			numValue, _ := strconv.ParseFloat(elementToken.Value, 64)
			element = &ast.NumberLiteral{
				Value: numValue,
				Pos: ast.Position{
					Line:   elementToken.Line,
					Column: elementToken.Column,
					Offset: elementToken.Position,
				},
			}
		case lexer.TokenIdentifier:
			element = ast.NewIdentifier(elementToken, elementToken.Value)
		default:
			// Неизвестный тип токена, пропускаем
			continue
		}

		if element != nil {
			arrayNode.Elements = append(arrayNode.Elements, element)
		}
	}

	// Если дошли сюда, значит не нашли закрывающую скобку
	return nil, fmt.Errorf("unclosed Lua array")
}

// parseLoopBody парсит тело цикла
func (h *ForInLoopHandler) parseLoopBody(ctx *common.ParseContext) ([]ast.Statement, error) {
	tokenStream := ctx.TokenStream
	body := make([]ast.Statement, 0)

	// Debug output
	if h.verbose {
		fmt.Printf("DEBUG: parseLoopBody called, current token: %v\n", tokenStream.Current())
	}

	// Проверяем, что тело цикла не пустое
	if !tokenStream.HasMore() {
		return body, nil
	}

	current := tokenStream.Current()
	if h.verbose {
		fmt.Printf("DEBUG parseLoopBody: Processing token: %s(%s)\n", current.Type, current.Value)
	}

	// Для однострочного цикла (Python style) мы парсим только одно выражение после ':'
	// и останавливаемся после newline или EOF

	if current.Type == lexer.TokenEOF {
		if h.verbose {
			fmt.Printf("DEBUG parseLoopBody: Found EOF, stopping\n")
		}
		return body, nil
	}

	// Обрабатываем match выражения
	if current.Type == lexer.TokenMatch {
		matchHandler := NewMatchHandler(config.ConstructHandlerConfig{})
		result, err := matchHandler.Handle(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to parse match statement: %v", err)
		}
		if matchStmt, ok := result.(*ast.MatchStatement); ok {
			body = append(body, matchStmt)
			return body, nil
		}
	}

	// Проверяем, начинается ли тело с квалифицированного идентификатора (py.total = ...)
	if isLanguageToken(current.Type) && tokenStream.HasMore() && tokenStream.Peek().Type == lexer.TokenDot {
		// Сохраняем позицию, чтобы можно было откатиться
		savedPos := tokenStream.Position()

		// Проверяем, является ли это присваиванием или language call
		assignment, err := h.parseQualifiedAssignment(ctx)
		if err != nil {
			// Если не удалось разобрать как присваивание, пробуем разобрать как language call
			tokenStream.SetPosition(savedPos) // откатываемся

			languageCallHandler := NewLanguageCallHandler(config.ConstructHandlerConfig{})
			result, callErr := languageCallHandler.Handle(ctx)
			if callErr != nil {
				return nil, fmt.Errorf("failed to parse assignment or language call in loop body: assignment error: %v, language call error: %v", err, callErr)
			}
			if callStmt, ok := result.(*ast.LanguageCall); ok {
				body = append(body, callStmt)
				if h.verbose {
					fmt.Printf("DEBUG: parseLoopBody - added language call statement to body\n")
				}

				// Пропускаем токен новой строки после успешного парсинга
				if tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenNewline {
					tokenStream.Consume()
					if h.verbose {
						fmt.Printf("DEBUG: parseLoopBody - consumed newline after language call\n")
					}
				}
				return body, nil
			}
		} else {
			// Успешно разобрали как присваивание
			body = append(body, assignment)
			if h.verbose {
				fmt.Printf("DEBUG: parseLoopBody - added qualified assignment to body, total body statements: %d\n", len(body))
			}

			// Пропускаем токен новой строки после успешного парсинга присваивания
			if tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenNewline {
				tokenStream.Consume()
				if h.verbose {
					fmt.Printf("DEBUG: parseLoopBody - consumed newline after assignment\n")
				}
			}
			return body, nil
		}
	} else {
		// Здесь можно добавить обработку других выражений, если потребуется
		if h.verbose {
			fmt.Printf("DEBUG: parseLoopBody - unsupported expression in single-line loop body, starts with %s\n", current.Type)
		}
		return nil, fmt.Errorf("unsupported expression in single-line loop body, starts with %s", current.Type)
	}

	if h.verbose {
		fmt.Printf("DEBUG: parseLoopBody returning %d statements\n", len(body))
	}
	return body, nil
}

// Config возвращает конфигурацию обработчика
func (h *ForInLoopHandler) Config() common.HandlerConfig {
	return common.HandlerConfig{
		IsEnabled: h.config.IsEnabled,
		Priority:  h.config.Priority,
		Order:     h.config.Order,
		Name:      h.config.Name,
	}
}

// Name возвращает имя обработчика
func (h *ForInLoopHandler) Name() string {
	return h.config.Name
}

// handleQualifiedVariableAsIterable обрабатывает qualified variable как итерируемый объект
func (h *ForInLoopHandler) handleQualifiedVariableAsIterable(ctx *common.ParseContext, variable *ast.Identifier, inToken lexer.Token) (interface{}, error) {
	tokenStream := ctx.TokenStream

	// Сохраняем текущую позицию
	currentPos := tokenStream.Position()

	// Потребляем первый токен (язык)
	firstToken := tokenStream.Consume()
	tempFirstToken := lexer.Token{Type: firstToken.Type}
	if firstToken.Type != lexer.TokenIdentifier && !tempFirstToken.IsLanguageToken() {
		return nil, fmt.Errorf("expected identifier for qualified variable, got %s", firstToken.Type)
	}

	language := firstToken.Value
	var path []string
	var lastName string

	// Обрабатываем остальные части qualified variable
	for tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenDot {
		// Потребляем точку
		tokenStream.Consume()

		// Проверяем, что после точки идет идентификатор
		tempCurrentToken := lexer.Token{Type: tokenStream.Current().Type}
		if !tokenStream.HasMore() || (tokenStream.Current().Type != lexer.TokenIdentifier && !tempCurrentToken.IsLanguageToken()) {
			return nil, fmt.Errorf("expected identifier after dot in qualified variable")
		}

		// Потребляем идентификатор
		idToken := tokenStream.Consume()

		// Если следующий токен - точка, то это часть пути
		if tokenStream.HasMore() && tokenStream.Peek().Type == lexer.TokenDot {
			path = append(path, idToken.Value)
		} else {
			// Это последнее имя
			lastName = idToken.Value
		}
	}

	// Создаем квалифицированный идентификатор
	var qualifiedIdentifier *ast.Identifier
	if len(path) > 0 {
		qualifiedIdentifier = ast.NewQualifiedIdentifierWithPath(firstToken, tokenStream.PeekN(-1), language, path, lastName)
	} else {
		qualifiedIdentifier = ast.NewQualifiedIdentifier(firstToken, tokenStream.PeekN(-1), language, lastName)
	}

	// Создаем VariableRead
	qualifiedVarNode := ast.NewVariableRead(qualifiedIdentifier)

	// 5. Проверяем токен ':'
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenColon {
		tokenStream.SetPosition(currentPos)
		return nil, fmt.Errorf("expected ':' after iterable")
	}
	colonToken := tokenStream.Consume()

	// 6. Читаем тело цикла
	body, err := h.parseLoopBody(ctx)
	if err != nil {
		tokenStream.SetPosition(currentPos)
		return nil, fmt.Errorf("failed to parse loop body: %v", err)
	}

	// 7. Создаем узел AST
	forToken := lexer.Token{
		Type:     lexer.TokenFor,
		Value:    "for",
		Line:     variable.Token.Line,
		Column:   variable.Token.Column - 5, // Приблизительная позиция
		Position: variable.Token.Position - 5,
	}

	loopNode := ast.NewForInLoopStatement(forToken, inToken, colonToken, variable, qualifiedVarNode, body)

	return loopNode, nil
}

// isLanguageToken - вспомогательная функция для проверки токенов языка
func isLanguageToken(tokenType lexer.TokenType) bool {
	// Создаем временный токен для проверки
	tempToken := lexer.Token{Type: tokenType}
	return tempToken.IsLanguageToken()
}

// parseQualifiedAssignment парсит присваивание квалифицированной переменной
// (например, py.total = py.total + item) внутри тела цикла.
func (h *ForInLoopHandler) parseQualifiedAssignment(ctx *common.ParseContext) (*ast.VariableAssignment, error) {
	tokenStream := ctx.TokenStream

	// 1. Парсим левую часть (py.total)
	langToken := tokenStream.Consume() // PY token

	// Проверяем и потребляем точку
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenDot {
		return nil, fmt.Errorf("expected '.' after language token '%s'", langToken.Type)
	}
	tokenStream.Consume() // DOT token

	// Проверяем и потребляем идентификатор
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenIdentifier {
		return nil, fmt.Errorf("expected identifier after '.'")
	}
	varToken := tokenStream.Consume() // IDENTIFIER token (total)

	// Создаем квалифицированный идентификатор
	leftIdentifier := ast.NewQualifiedIdentifier(langToken, varToken, langToken.Value, varToken.Value)

	// 2. Потребляем знак присваивания
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenAssign {
		return nil, fmt.Errorf("expected '=' after identifier in assignment")
	}
	assignToken := tokenStream.Consume()

	// 3. Парсим правую часть (py.total + item)
	// Это бинарное выражение, которое может содержать квалифицированные идентификаторы
	rightExpr, err := h.parseBinaryExpressionWithQualifiedVars(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to parse right-hand side of assignment: %v", err)
	}

	// Создаем и возвращаем узел присваивания
	return ast.NewVariableAssignment(leftIdentifier, assignToken, rightExpr), nil
}

// parseBinaryExpressionWithQualifiedVars парсит бинарное выражение, которое может содержать квалифицированные идентификаторы
// (например, py.total + item)
func (h *ForInLoopHandler) parseBinaryExpressionWithQualifiedVars(ctx *common.ParseContext) (ast.Expression, error) {
	tokenStream := ctx.TokenStream

	// 1. Парсим левый операнд
	var leftExpr ast.Expression
	var err error
	currentToken := tokenStream.Current()

	if isLanguageToken(currentToken.Type) {
		// Это квалифицированный идентификатор (py.total)
		langToken := tokenStream.Consume() // PY token

		// Проверяем и потребляем точку
		if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenDot {
			return nil, fmt.Errorf("expected '.' after language token '%s'", langToken.Type)
		}
		tokenStream.Consume() // DOT token

		// Проверяем и потребляем идентификатор
		if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenIdentifier {
			return nil, fmt.Errorf("expected identifier after '.'")
		}
		varToken := tokenStream.Consume() // IDENTIFIER token (total)

		// Создаем квалифицированный идентификатор
		leftExpr = ast.NewVariableRead(ast.NewQualifiedIdentifier(langToken, varToken, langToken.Value, varToken.Value))
	} else if currentToken.Type == lexer.TokenIdentifier {
		// Это простой идентификатор (item)
		identToken := tokenStream.Consume()
		leftExpr = ast.NewIdentifier(identToken, identToken.Value)
	} else if currentToken.Type == lexer.TokenDoubleLeftAngle {
		// Это битовая строка (<<...>>)
		assignmentHandler := NewAssignmentHandler(0, 0)
		leftExpr, err = assignmentHandler.parseBitstringValue(tokenStream)
		if err != nil {
			return nil, fmt.Errorf("failed to parse bitstring expression: %v", err)
		}
	} else {
		return nil, fmt.Errorf("expected identifier, language token, or bitstring for left operand, got %s", currentToken.Type)
	}

	// 2. Проверяем и потребляем оператор
	if !tokenStream.HasMore() {
		return leftExpr, nil // Если больше нет токенов, возвращаем левый операнд как есть
	}

	operatorToken := tokenStream.Current()
	if operatorToken.Type != lexer.TokenPlus && operatorToken.Type != lexer.TokenMinus &&
		operatorToken.Type != lexer.TokenMultiply && operatorToken.Type != lexer.TokenSlash {
		return leftExpr, nil // Если это не бинарный оператор, возвращаем левый операнд как есть
	}

	tokenStream.Consume() // Потребляем оператор

	// 3. Парсим правый операнд
	if !tokenStream.HasMore() {
		return nil, fmt.Errorf("expected right operand after operator")
	}

	rightToken := tokenStream.Current()
	var rightExpr ast.Expression

	if isLanguageToken(rightToken.Type) {
		// Это квалифицированный идентификатор (py.total)
		langToken := tokenStream.Consume() // PY token

		// Проверяем и потребляем точку
		if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenDot {
			return nil, fmt.Errorf("expected '.' after language token '%s'", langToken.Type)
		}
		tokenStream.Consume() // DOT token

		// Проверяем и потребляем идентификатор
		if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenIdentifier {
			return nil, fmt.Errorf("expected identifier after '.'")
		}
		varToken := tokenStream.Consume() // IDENTIFIER token (total)

		// Создаем квалифицированный идентификатор
		rightExpr = ast.NewVariableRead(ast.NewQualifiedIdentifier(langToken, varToken, langToken.Value, varToken.Value))
	} else if rightToken.Type == lexer.TokenIdentifier {
		// Это простой идентификатор (item)
		identToken := tokenStream.Consume()
		rightExpr = ast.NewIdentifier(identToken, identToken.Value)
	} else if rightToken.Type == lexer.TokenNumber {
		// Это числовой литерал
		numToken := tokenStream.Consume()
		numValue, _ := strconv.ParseFloat(numToken.Value, 64)
		rightExpr = &ast.NumberLiteral{Value: numValue, Pos: tokenToPosition(numToken)}
	} else if rightToken.Type == lexer.TokenDoubleLeftAngle {
		// Это битовая строка (<<...>>)
		assignmentHandler := NewAssignmentHandler(0, 0)
		rightExpr, err = assignmentHandler.parseBitstringValue(tokenStream)
		if err != nil {
			return nil, fmt.Errorf("failed to parse bitstring expression: %v", err)
		}
	} else {
		return nil, fmt.Errorf("expected identifier, language token, number, or bitstring for right operand, got %s", rightToken.Type)
	}

	// 4. Создаем бинарное выражение - используем строковое представление оператора
	var operatorStr string
	switch operatorToken.Type {
	case lexer.TokenPlus:
		operatorStr = "+"
	case lexer.TokenMinus:
		operatorStr = "-"
	case lexer.TokenMultiply:
		operatorStr = "*"
	case lexer.TokenSlash:
		operatorStr = "/"
	default:
		return nil, fmt.Errorf("unsupported binary operator: %s", operatorToken.Type)
	}

	// Создаем позицию вручную
	pos := ast.Position{
		Line:   operatorToken.Line,
		Column: operatorToken.Column,
		Offset: operatorToken.Position,
	}

	return &ast.BinaryExpression{
		Left:     leftExpr,
		Operator: operatorStr,
		Right:    rightExpr,
		Pos:      pos,
	}, nil
}
