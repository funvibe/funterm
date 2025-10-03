package handler

import (
	"fmt"
	"go-parser/pkg/ast"
	"go-parser/pkg/common"
	"go-parser/pkg/config"
	"go-parser/pkg/lexer"
)

// BinaryExpressionHandler - обработчик бинарных выражений
type BinaryExpressionHandler struct {
	config  config.ConstructHandlerConfig
	verbose bool
}

// NewBinaryExpressionHandler создает новый обработчик бинарных выражений
func NewBinaryExpressionHandler(config config.ConstructHandlerConfig) *BinaryExpressionHandler {
	return NewBinaryExpressionHandlerWithVerbose(config, false)
}

// NewBinaryExpressionHandlerWithVerbose создает новый обработчик бинарных выражений с поддержкой verbose режима
func NewBinaryExpressionHandlerWithVerbose(config config.ConstructHandlerConfig, verbose bool) *BinaryExpressionHandler {
	return &BinaryExpressionHandler{
		config:  config,
		verbose: verbose,
	}
}

// CanHandle проверяет, может ли обработчик обработать токен
func (h *BinaryExpressionHandler) CanHandle(token lexer.Token) bool {
	// Обрабатываем идентификаторы, которые могут быть частью бинарных выражений
	// Например: "a = b + c" - здесь токен "a" является идентификатором
	return token.Type == lexer.TokenIdentifier
}

// Handle обрабатывает бинарное выражение
func (h *BinaryExpressionHandler) Handle(ctx *common.ParseContext) (interface{}, error) {
	tokenStream := ctx.TokenStream

	if h.verbose {
		fmt.Printf("DEBUG: BinaryExpressionHandler.Handle called with token: %s (%s)\n", tokenStream.Current().Value, tokenStream.Current().Type)
	}

	// Проверяем, есть ли токен присваивания
	if !tokenStream.HasMore() {
		return nil, fmt.Errorf("unexpected EOF in binary expression")
	}

	// Проверяем, является ли это квалифицированной переменной (lua.x)
	if tokenStream.Current().Type == lexer.TokenIdentifier &&
		tokenStream.HasMore() && tokenStream.Peek().Type == lexer.TokenDot {
		// Это квалифицированная переменная
		leftExpr, err := h.parseQualifiedVariable(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to parse left side qualified variable: %v", err)
		}

		// Проверяем наличие оператора присваивания
		if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenAssign {
			return nil, fmt.Errorf("expected '=' after qualified variable in binary expression")
		}
		tokenStream.Consume() // потребляем '='

		// Теперь парсим правую часть как бинарное выражение
		rightExpr, err := h.ParseFullExpression(ctx, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to parse right side of binary expression: %v", err)
		}

		// Создаем VariableAssignment с квалифицированной переменной
		return &ast.VariableAssignment{
			Variable: leftExpr.(*ast.VariableRead).Variable,
			Value:    rightExpr,
		}, nil
	}

	// Потребляем левый операнд (простой идентификатор)
	leftToken := tokenStream.Current()
	if leftToken.Type != lexer.TokenIdentifier {
		return nil, fmt.Errorf("expected identifier for binary expression, got %s", leftToken.Type)
	}
	tokenStream.Consume()

	// Проверяем наличие оператора присваивания
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenAssign {
		return nil, fmt.Errorf("expected '=' after identifier in binary expression")
	}
	tokenStream.Consume() // потребляем '='

	// Теперь парсим правую часть как бинарное выражение
	rightExpr, err := h.ParseFullExpression(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to parse right side of binary expression: %v", err)
	}

	// Создаем VariableAssignment с бинарным выражением
	return &ast.VariableAssignment{
		Variable: &ast.Identifier{
			Name: leftToken.Value,
			Pos: ast.Position{
				Line:   leftToken.Line,
				Column: leftToken.Column,
				Offset: leftToken.Position,
			},
		},
		Value: rightExpr,
	}, nil
}

// ParseFullExpression парсит полное бинарное выражение начиная с левого операнда
func (h *BinaryExpressionHandler) ParseFullExpression(ctx *common.ParseContext, left ast.Expression) (ast.Expression, error) {
	return h.parseExpressionWithPrecedence(ctx, left, 0)
}

// parseExpressionWithPrecedence парсит выражение с учетом приоритетов операторов
func (h *BinaryExpressionHandler) parseExpressionWithPrecedence(ctx *common.ParseContext, left ast.Expression, minPrecedence int) (ast.Expression, error) {
	tokenStream := ctx.TokenStream

	if h.verbose {
		fmt.Printf("DEBUG: parseExpressionWithPrecedence called with left: %v, minPrecedence: %d, current token: %s (%s)\n", left != nil, minPrecedence, tokenStream.Current().Value, tokenStream.Current().Type)
	}

	for tokenStream.HasMore() {
		operatorToken := tokenStream.Current()

		// Проверяем тернарный оператор
		if operatorToken.Type == lexer.TokenQuestion {
			return h.ParseTernaryExpression(ctx, left)
		}

		// Проверяем бинарный оператор
		if !isBinaryOperator(operatorToken.Type) {
			break // Нет бинарного оператора, выходим из цикла
		}

		precedence := getOperatorPrecedence(operatorToken.Type)
		if precedence < minPrecedence {
			break // Приоритет слишком низкий, выходим из цикла
		}

		// Потребляем оператор
		tokenStream.Consume()

		// Читаем правый операнд
		rightExpr, err := h.parseOperand(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to parse right operand: %v", err)
		}

		// Проверяем следующий оператор для определения ассоциативности
		for tokenStream.HasMore() && isBinaryOperator(tokenStream.Current().Type) {
			nextPrecedence := getOperatorPrecedence(tokenStream.Current().Type)
			if nextPrecedence <= precedence {
				break // Следующий оператор имеет меньший или равный приоритет
			}

			// Парсим правую часть с более высоким приоритетом
			rightExpr, err = h.parseExpressionWithPrecedence(ctx, rightExpr, nextPrecedence)
			if err != nil {
				return nil, err
			}
		}

		// Создаем бинарное выражение
		left = ast.NewBinaryExpression(left, operatorToken.Value, rightExpr, left.Position())
	}

	return left, nil
}

// ParseTernaryExpression парсит тернарный оператор (condition ? true_expr : false_expr)
func (h *BinaryExpressionHandler) ParseTernaryExpression(ctx *common.ParseContext, condition ast.Expression) (ast.Expression, error) {
	tokenStream := ctx.TokenStream

	// Потребляем токен ?
	questionToken := tokenStream.Consume()

	// Проверяем, не является ли это Elvis оператором (condition ?: fallback_expr)
	if tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenColon {
		// Это Elvis оператор: condition ?: fallback_expr
		colonToken := tokenStream.Consume()

		// Читаем fallback выражение
		fallbackExpr, err := h.parseOperand(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to parse fallback expression in Elvis: %v", err)
		}

		// Создаем Elvis выражение (как особый случай тернарного)
		return ast.NewTernaryExpression(condition, questionToken, colonToken, condition, fallbackExpr, condition.Position()), nil
	}

	// Читаем true выражение для обычного тернарного оператора
	trueExpr, err := h.parseOperand(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to parse true expression in ternary: %v", err)
	}

	// Проверяем наличие токена :
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenColon {
		return nil, fmt.Errorf("expected ':' after true expression in ternary")
	}

	// Потребляем токен :
	colonToken := tokenStream.Consume()

	// Читаем false выражение
	falseExpr, err := h.parseOperand(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to parse false expression in ternary: %v", err)
	}

	// Создаем тернарное выражение
	return ast.NewTernaryExpression(condition, questionToken, colonToken, trueExpr, falseExpr, condition.Position()), nil
}

// parseOperand парсит операнд бинарного выражения
func (h *BinaryExpressionHandler) parseOperand(ctx *common.ParseContext) (ast.Expression, error) {
	tokenStream := ctx.TokenStream

	if !tokenStream.HasMore() {
		return nil, fmt.Errorf("unexpected EOF in operand")
	}

	token := tokenStream.Current()

	switch token.Type {
	case lexer.TokenIdentifier:
		// Это может быть простой идентификатор или qualified variable
		if tokenStream.Peek().Type == lexer.TokenDot {
			// Проверяем, не является ли это вызовом функции вида python.check_status()
			if tokenStream.HasMore() && tokenStream.PeekN(2).Type == lexer.TokenIdentifier &&
				tokenStream.HasMore() && tokenStream.PeekN(3).Type == lexer.TokenLeftParen {
				// Это вызов функции с точечной нотацией - не обрабатываем в BinaryExpressionHandler
				return nil, fmt.Errorf("qualified function call detected - should be handled by LanguageCallHandler")
			} else {
				// Это qualified variable без вызова функции
				expr, err := h.parseQualifiedVariable(ctx)
				if err != nil {
					return nil, err
				}

				// Проверяем наличие индексного выражения [index] после qualified variable
				if tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenLBracket {
					return h.parseIndexExpression(ctx, expr)
				}

				return expr, nil
			}
		} else {
			// Простой идентификатор
			tokenStream.Consume()
			ident := ast.NewIdentifier(token, token.Value)

			// Проверяем наличие индексного выражения [index] после простого идентификатора
			if tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenLBracket {
				return h.parseIndexExpression(ctx, ident)
			}

			return ident, nil
		}
	case lexer.TokenDoubleLeftAngle:
		// Bitstring pattern assignment: <<pattern>> = value
		// Используем BitstringPatternAssignmentHandler
		bitstringPatternHandler := NewBitstringPatternAssignmentHandlerWithVerbose(110, 4, h.verbose)
		if !bitstringPatternHandler.CanHandle(token) {
			return nil, fmt.Errorf("expected bitstring pattern assignment, got %s", token.Type)
		}

		result, err := bitstringPatternHandler.Handle(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to parse bitstring pattern assignment: %v", err)
		}

		bitstringPatternAssignment, ok := result.(*ast.BitstringPatternAssignment)
		if !ok {
			return nil, fmt.Errorf("expected BitstringPatternAssignment, got %T", result)
		}

		return bitstringPatternAssignment, nil

	default:
		// Проверяем, является ли токен языковым токеном
		if token.IsLanguageToken() {
			// Это может быть квалифицированная переменная
			if tokenStream.Peek().Type == lexer.TokenDot {
				// Проверяем, не является ли это вызовом функции вида python.check_status()
				if tokenStream.HasMore() && tokenStream.PeekN(2).Type == lexer.TokenIdentifier &&
					tokenStream.HasMore() && tokenStream.PeekN(3).Type == lexer.TokenLeftParen {
					// Это вызов функции с точечной нотацией - не обрабатываем в BinaryExpressionHandler
					return nil, fmt.Errorf("qualified function call detected - should be handled by LanguageCallHandler")
				} else {
					// Это qualified variable без вызова функции
					expr, err := h.parseQualifiedVariable(ctx)
					if err != nil {
						return nil, err
					}

					// Проверяем наличие индексного выражения [index] после qualified variable
					if tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenLBracket {
						return h.parseIndexExpression(ctx, expr)
					}

					return expr, nil
				}
			} else {
				return nil, fmt.Errorf("expected '.' after language token '%s'", token.Type)
			}
		}
		return nil, fmt.Errorf("unsupported operand type: %s", token.Type)

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

	case lexer.TokenTrue, lexer.TokenFalse:
		// Булев литерал
		tokenStream.Consume()
		return &ast.BooleanLiteral{
			Value: token.Type == lexer.TokenTrue,
			Pos: ast.Position{
				Line:   token.Line,
				Column: token.Column,
				Offset: token.Position,
			},
		}, nil

	case lexer.TokenLeftParen:
		// Выражение в скобках
		tokenStream.Consume() // потребляем '('

		// Сначала парсим первый операнд внутри скобок
		leftOperand, err := h.parseOperand(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to parse first operand in parentheses: %v", err)
		}

		// Затем парсим полное выражение, начиная с первого операнда
		expr, err := h.ParseFullExpression(ctx, leftOperand)
		if err != nil {
			return nil, fmt.Errorf("failed to parse expression in parentheses: %v", err)
		}

		// Проверяем и потребляем закрывающую скобку
		if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenRightParen {
			return nil, fmt.Errorf("expected ')' after expression")
		}
		tokenStream.Consume() // потребляем ')'

		return expr, nil

	}
}

// parseLanguageCall парсит вызов функции другого языка
func (h *BinaryExpressionHandler) parseLanguageCall(ctx *common.ParseContext) (ast.Expression, error) {
	tokenStream := ctx.TokenStream

	// Парсим вызов функции напрямую, без использования LanguageCallHandler
	// потому что LanguageCallHandler ожидает конец выражения, а у нас могут быть бинарные операторы

	// Читаем язык
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenIdentifier {
		return nil, fmt.Errorf("expected language identifier")
	}
	languageToken := tokenStream.Consume()
	language := languageToken.Value

	// Проверяем и потребляем DOT
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenDot {
		return nil, fmt.Errorf("expected DOT after language '%s'", language)
	}
	tokenStream.Consume()

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
			arg, err := h.parseOperand(ctx)
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

	// Создаем узел AST
	startPos := ast.Position{
		Line:   languageToken.Line,
		Column: languageToken.Column,
		Offset: languageToken.Position,
	}

	return &ast.LanguageCall{
		Language:  language,
		Function:  functionName,
		Arguments: arguments,
		Pos:       startPos,
	}, nil
}

// parseQualifiedVariable парсит квалифицированную переменную или вызов функции
func (h *BinaryExpressionHandler) parseQualifiedVariable(ctx *common.ParseContext) (ast.Expression, error) {
	tokenStream := ctx.TokenStream

	// Потребляем первый идентификатор (язык)
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
		if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenIdentifier {
			return nil, fmt.Errorf("expected identifier after dot in qualified variable")
		}

		// Потребляем идентификатор
		idToken := tokenStream.Consume()

		// Проверяем, не является ли это вызовом функции
		if tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenLeftParen {
			// Это вызов функции вида python.check_status()
			functionName := idToken.Value

			// Потребляем открывающую скобку
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
					arg, err := h.parseOperand(ctx)
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

		// Если следующий токен - точка, то это часть пути
		if tokenStream.HasMore() && tokenStream.Peek().Type == lexer.TokenDot {
			path = append(path, idToken.Value)
		} else {
			// Это последнее имя
			lastName = idToken.Value
		}
	}

	// Если мы дошли сюда, это не вызов функции, а квалифицированная переменная
	// Создаем квалифицированный идентификатор
	var qualifiedIdentifier *ast.Identifier
	if len(path) > 0 {
		qualifiedIdentifier = ast.NewQualifiedIdentifierWithPath(firstToken, tokenStream.PeekN(-1), language, path, lastName)
	} else {
		qualifiedIdentifier = ast.NewQualifiedIdentifier(firstToken, tokenStream.PeekN(-1), language, lastName)
	}

	// Создаем VariableRead
	return ast.NewVariableRead(qualifiedIdentifier), nil
}

// parseIndexExpression парсит индексное выражение вида object[index]
func (h *BinaryExpressionHandler) parseIndexExpression(ctx *common.ParseContext, object ast.Expression) (ast.Expression, error) {
	tokenStream := ctx.TokenStream

	if h.verbose {
		fmt.Printf("DEBUG: parseIndexExpression called with object: %T, current token: %s (%s)\n", object, tokenStream.Current().Value, tokenStream.Current().Type)
	}

	// Проверяем наличие открывающей квадратной скобки
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenLBracket {
		return nil, fmt.Errorf("expected '[' for index expression")
	}

	// Потребляем открывающую скобку
	tokenStream.Consume()

	// Парсим индексное выражение - сначала пробуем parseOperand для простых выражений
	// но если это выражение в скобках, нужно использовать более сложную логику
	var indexExpr ast.Expression
	var err error

	if tokenStream.Current().Type == lexer.TokenLeftParen {
		// Выражение в скобках - потребляем '('
		tokenStream.Consume()

		// Сначала парсим первый операнд внутри скобок
		leftOperand, err := h.parseOperand(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to parse first operand in index parentheses: %v", err)
		}

		// Затем парсим полное выражение, начиная с первого операнда
		indexExpr, err = h.ParseFullExpression(ctx, leftOperand)
		if err != nil {
			return nil, fmt.Errorf("failed to parse index expression in parentheses: %v", err)
		}

		// Проверяем и потребляем закрывающую скобку
		if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenRightParen {
			return nil, fmt.Errorf("expected ')' after index expression")
		}
		tokenStream.Consume() // потребляем ')'
	} else {
		// Простое выражение без скобок
		indexExpr, err = h.parseOperand(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to parse index expression: %v", err)
		}
	}

	// Проверяем наличие закрывающей квадратной скобки
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenRBracket {
		return nil, fmt.Errorf("expected ']' after index expression")
	}

	// Потребляем закрывающую скобку
	tokenStream.Consume()

	// Создаем индексное выражение
	return ast.NewIndexExpression(object, indexExpr, object.Position()), nil
}

// Config возвращает конфигурацию обработчика
func (h *BinaryExpressionHandler) Config() common.HandlerConfig {
	return common.HandlerConfig{
		IsEnabled: h.config.IsEnabled,
		Priority:  h.config.Priority,
		Name:      h.config.Name,
	}
}

// Name возвращает имя обработчика
func (h *BinaryExpressionHandler) Name() string {
	return h.config.Name
}

// BinaryExpressionPart - вспомогательная структура для парсинга бинарных выражений
type BinaryExpressionPart struct {
	Operator string
	Right    ast.Expression
	Pos      ast.Position
}
