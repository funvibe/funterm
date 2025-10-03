package handler

import (
	"fmt"
	"go-parser/pkg/ast"
	"go-parser/pkg/common"
	"go-parser/pkg/config"
	"go-parser/pkg/lexer"
)

// ElvisExpressionHandler - обработчик Elvis оператора (?:)
type ElvisExpressionHandler struct {
	config config.ConstructHandlerConfig
}

// NewElvisExpressionHandler создает новый обработчик Elvis оператора
func NewElvisExpressionHandler(config config.ConstructHandlerConfig) *ElvisExpressionHandler {
	return &ElvisExpressionHandler{
		config: config,
	}
}

// CanHandle проверяет, может ли обработчик обработать токен
func (h *ElvisExpressionHandler) CanHandle(token lexer.Token) bool {
	// Обрабатываем идентификаторы, которые могут быть частью Elvis выражений
	// Например: "status = condition ? true : false" - здесь токен "status" является идентификатором
	return token.Type == lexer.TokenIdentifier
}

// Handle обрабатывает Elvis оператор
func (h *ElvisExpressionHandler) Handle(ctx *common.ParseContext) (interface{}, error) {
	tokenStream := ctx.TokenStream

	// Проверяем, есть ли токен присваивания
	if !tokenStream.HasMore() {
		return nil, fmt.Errorf("unexpected EOF in Elvis expression")
	}

	// Потребляем левый операнд (идентификатор)
	leftToken := tokenStream.Current()
	if leftToken.Type != lexer.TokenIdentifier {
		return nil, fmt.Errorf("expected identifier for Elvis expression, got %s", leftToken.Type)
	}
	tokenStream.Consume()

	// Проверяем наличие оператора присваивания
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenAssign {
		return nil, fmt.Errorf("expected '=' after identifier in Elvis expression")
	}
	tokenStream.Consume() // потребляем '='

	// Теперь парсим правую часть как Elvis выражение
	rightExpr, err := h.ParseFullExpression(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to parse right side of Elvis expression: %v", err)
	}

	// Создаем VariableAssignment с Elvis выражением
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

// ParseFullExpression парсит полное Elvis выражение начиная с левого операнда
func (h *ElvisExpressionHandler) ParseFullExpression(ctx *common.ParseContext, left ast.Expression) (ast.Expression, error) {
	tokenStream := ctx.TokenStream

	if !tokenStream.HasMore() {
		return left, nil // Нет оператора, возвращаем левый операнд как есть
	}

	operatorToken := tokenStream.Current()
	if !isElvisOperator(operatorToken.Type) {
		return left, nil // Нет Elvis оператора, возвращаем левый операнд
	}

	// Потребляем оператор ?
	tokenStream.Consume()

	// Проверяем наличие :
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenColon {
		return nil, fmt.Errorf("expected ':' after '?' in Elvis expression")
	}

	// Потребляем :
	colonToken := tokenStream.Consume()

	// Читаем правый операнд
	rightExpr, err := h.parseOperand(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to parse right operand in Elvis expression: %v", err)
	}

	// Создаем Elvis выражение
	elvisExpr := ast.NewElvisExpression(left, operatorToken, colonToken, rightExpr, left.Position())

	// Elvis оператор имеет низкий приоритет, поэтому больше не проверяем последующие операторы
	return elvisExpr, nil
}

// parseOperand парсит операнд Elvis выражения
func (h *ElvisExpressionHandler) parseOperand(ctx *common.ParseContext) (ast.Expression, error) {
	tokenStream := ctx.TokenStream

	if !tokenStream.HasMore() {
		return nil, fmt.Errorf("unexpected EOF in operand")
	}

	token := tokenStream.Current()

	switch token.Type {
	case lexer.TokenIdentifier:
		// Это может быть простой идентификатор, вызов функции или qualified variable
		if tokenStream.Peek().Type == lexer.TokenLeftParen {
			// Это вызов функции вида func()
			return h.parseLanguageCall(ctx)
		} else if tokenStream.Peek().Type == lexer.TokenDot {
			// Проверяем, не является ли это вызовом функции вида python.check_status()
			if tokenStream.HasMore() && tokenStream.PeekN(2).Type == lexer.TokenIdentifier &&
				tokenStream.HasMore() && tokenStream.PeekN(3).Type == lexer.TokenLeftParen {
				// Это вызов функции с точечной нотацией
				return h.parseQualifiedVariable(ctx)
			} else {
				// Это qualified variable без вызова функции
				return h.parseQualifiedVariable(ctx)
			}
		} else {
			// Простой идентификатор
			tokenStream.Consume()
			return ast.NewIdentifier(token, token.Value), nil
		}

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

		// Рекурсивно парсим выражение внутри скобок
		expr, err := h.ParseFullExpression(ctx, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to parse expression in parentheses: %v", err)
		}

		// Проверяем и потребляем закрывающую скобку
		if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenRightParen {
			return nil, fmt.Errorf("expected ')' after expression")
		}
		tokenStream.Consume() // потребуем ')'

		return expr, nil

	default:
		return nil, fmt.Errorf("unsupported operand type for Elvis expression: %s", token.Type)
	}
}

// parseLanguageCall парсит вызов функции другого языка
func (h *ElvisExpressionHandler) parseLanguageCall(ctx *common.ParseContext) (ast.Expression, error) {
	tokenStream := ctx.TokenStream

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
func (h *ElvisExpressionHandler) parseQualifiedVariable(ctx *common.ParseContext) (ast.Expression, error) {
	tokenStream := ctx.TokenStream

	// Потребляем первый идентификатор (язык)
	firstToken := tokenStream.Consume()
	if firstToken.Type != lexer.TokenIdentifier {
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

// Config возвращает конфигурацию обработчика
func (h *ElvisExpressionHandler) Config() common.HandlerConfig {
	return common.HandlerConfig{
		IsEnabled: h.config.IsEnabled,
		Priority:  h.config.Priority,
		Name:      h.config.Name,
	}
}

// Name возвращает имя обработчика
func (h *ElvisExpressionHandler) Name() string {
	return h.config.Name
}

// ElvisExpressionPart - вспомогательная структура для парсинга Elvis выражений
type ElvisExpressionPart struct {
	Question lexer.Token
	Colon    lexer.Token
	Right    ast.Expression
	Pos      ast.Position
}
