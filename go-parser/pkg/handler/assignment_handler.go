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

// AssignmentHandler - обработчик присваивания переменных
type AssignmentHandler struct {
	config  common.HandlerConfig
	verbose bool
}

// NewAssignmentHandler создает новый обработчик присваивания
func NewAssignmentHandler(priority, order int) *AssignmentHandler {
	return NewAssignmentHandlerWithVerbose(priority, order, false)
}

// NewAssignmentHandlerWithVerbose создает новый обработчик присваивания с поддержкой verbose режима
func NewAssignmentHandlerWithVerbose(priority, order int, verbose bool) *AssignmentHandler {
	config := DefaultConfig("assignment")
	config.Priority = priority
	config.Order = order
	return &AssignmentHandler{
		config:  config,
		verbose: verbose,
	}
}

// CanHandle проверяет, может ли обработчик обработать токен
func (h *AssignmentHandler) CanHandle(token lexer.Token) bool {
	// Поддерживаем обычные идентификаторы и языковые токены (PY, LUA, GO, JS, etc.)
	// Но НЕ keywords как 'for', 'if', 'while' и т.д.
	if token.Type == lexer.TokenFor || token.Type == lexer.TokenIf || token.Type == lexer.TokenWhile ||
		token.Type == lexer.TokenMatch || token.Type == lexer.TokenIn {
		return false
	}
	result := token.IsLanguageIdentifierOrCallToken()
	if h.verbose && result {
		fmt.Printf("DEBUG: AssignmentHandler.CanHandle - token %s(%s) -> true\n", token.Type, token.Value)
	}
	return result
}

// Handle обрабатывает присваивание переменной
func (h *AssignmentHandler) Handle(ctx *common.ParseContext) (interface{}, error) {
	if h.verbose {
		fmt.Printf("DEBUG: AssignmentHandler.Handle - ENTRY POINT, current token: %s(%s)\n", ctx.TokenStream.Current().Type, ctx.TokenStream.Current().Value)
	}
	// Проверяем защиту от рекурсии
	if err := ctx.Guard.Enter(); err != nil {
		return nil, err
	}
	defer ctx.Guard.Exit()

	// Отладочная информация
	if h.verbose {
		fmt.Printf("DEBUG: *** AssignmentHandler.Handle called, current token: %v\n", ctx.TokenStream.Current())
		if ctx.TokenStream.HasMore() {
			fmt.Printf("DEBUG: *** AssignmentHandler.Handle next token: %v\n", ctx.TokenStream.Peek())
		}
	}

	// Потребляем идентификатор или языковой токен
	identifierToken := ctx.TokenStream.Consume()
	if h.verbose {
		fmt.Printf("DEBUG: AssignmentHandler.Handle consumed token: %v, IsLanguageIdentifierOrCallToken: %v\n", identifierToken, identifierToken.IsLanguageIdentifierOrCallToken())
	}
	if !identifierToken.IsLanguageIdentifierOrCallToken() {
		if h.verbose {
			fmt.Printf("DEBUG: AssignmentHandler - rejecting token %s(%s) as not language identifier or call token\n", identifierToken.Type, identifierToken.Value)
		}
		return nil, newErrorWithTokenPos(identifierToken, "expected identifier or language token, got %s", identifierToken.Type)
	}

	// Проверяем наличие знака присваивания
	// Сначала пропускаем квалифицированную часть переменной (если есть)
	var varToken lexer.Token
	var hasIndexExpression bool
	var indexStart int          // Позиция начала индексного выражения
	var qualifiedParts []string // Собираем все части квалифицированного идентификатора

	// Обрабатываем квалифицированные идентификаторы с множественными точками (py.data.users[0])
	for ctx.TokenStream.HasMore() && ctx.TokenStream.Current().Type == lexer.TokenDot {
		// Пропускаем DOT
		ctx.TokenStream.Consume() // потребляем DOT

		if !ctx.TokenStream.HasMore() || (ctx.TokenStream.Current().Type != lexer.TokenIdentifier &&
			ctx.TokenStream.Current().Type != lexer.TokenTrue &&
			ctx.TokenStream.Current().Type != lexer.TokenFalse) {
			return nil, newErrorWithPos(ctx.TokenStream, "expected identifier or literal after DOT")
		}

		partToken := ctx.TokenStream.Consume() // потребляем и сохраняем идентификатор или литерал
		qualifiedParts = append(qualifiedParts, partToken.Value)

		if h.verbose {
			fmt.Printf("DEBUG: AssignmentHandler - collected qualified part: %v\n", partToken)
		}

		// Проверяем, не начинается ли индексное выражение после этой части
		if ctx.TokenStream.HasMore() && ctx.TokenStream.Current().Type == lexer.TokenLBracket {
			// Это последняя часть перед индексным выражением
			varToken = partToken
			break
		}

		// Проверяем, не является ли это концом левой части присваивания
		// Если следующий токен - это оператор присваивания, то это последняя часть
		if ctx.TokenStream.HasMore() && (ctx.TokenStream.Current().Type == lexer.TokenAssign || ctx.TokenStream.Current().Type == lexer.TokenColonEquals) {
			varToken = partToken
			break
		}
	}

	// Если у нас есть квалифицированные части, но varToken не установлен (нет индексного выражения),
	// то varToken - это последняя собранная часть
	if len(qualifiedParts) > 0 && varToken.Value == "" {
		varToken = lexer.Token{
			Type:   lexer.TokenIdentifier,
			Value:  qualifiedParts[len(qualifiedParts)-1],
			Line:   identifierToken.Line,
			Column: identifierToken.Column + len(identifierToken.Value) + 1,
		}
	}

	// Проверяем наличие индексного выражения (для индексного присваивания arr[0] = value)
	if ctx.TokenStream.HasMore() && ctx.TokenStream.Current().Type == lexer.TokenLBracket {
		hasIndexExpression = true
		// Сохраняем текущую позицию для последующего парсинга
		indexStart = ctx.TokenStream.Position()

		// Пропускаем индексное выражение [...] до закрывающей скобки
		ctx.TokenStream.Consume() // потребляем '['
		depth := 1
		for ctx.TokenStream.HasMore() && depth > 0 {
			token := ctx.TokenStream.Current()
			if token.Type == lexer.TokenLBracket {
				depth++
			} else if token.Type == lexer.TokenRBracket {
				depth--
			}
			ctx.TokenStream.Consume()
		}
		if h.verbose {
			fmt.Printf("DEBUG: AssignmentHandler - detected index expression\n")
		}
	}

	if h.verbose {
		fmt.Printf("DEBUG: AssignmentHandler - checking for assignment operator, current token: %v\n", ctx.TokenStream.Current())
	}
	if !ctx.TokenStream.HasMore() || (ctx.TokenStream.Current().Type != lexer.TokenAssign && ctx.TokenStream.Current().Type != lexer.TokenColonEquals) {
		if h.verbose {
			fmt.Printf("DEBUG: AssignmentHandler - no assignment operator found, current token: %v\n", ctx.TokenStream.Current())
		}
		// Если нет знака присваивания, проверяем, не должен ли этот случай обрабатываться другим обработчиком

		// Если следующий токен - DOT после индексного выражения, это может быть доступ к свойству индексированного объекта
		if ctx.TokenStream.HasMore() && ctx.TokenStream.Current().Type == lexer.TokenDot {
			if hasIndexExpression {
				if h.verbose {
					fmt.Printf("DEBUG: AssignmentHandler - found DOT after index expression, handling property access\n")
				}
				// Это случай вроде py.data.users[0].age = 26
				// Обрабатываем это как вложенный IndexExpression
				return h.handlePropertyAccessAfterIndex(ctx, identifierToken, varToken, qualifiedParts, indexStart)
			} else {
				if h.verbose {
					fmt.Printf("DEBUG: AssignmentHandler - found DOT token, returning error\n")
				}
				return nil, newErrorWithPos(ctx.TokenStream, "expected identifier after DOT")
			}
		}

		// Если есть еще токены и это не DOT и не LEFT_PAREN и не EOF и не NEWLINE, это недопустимый синтаксис
		if ctx.TokenStream.HasMore() && ctx.TokenStream.Current().Type != lexer.TokenDot && ctx.TokenStream.Current().Type != lexer.TokenLeftParen && ctx.TokenStream.Current().Type != lexer.TokenNewline {
			return nil, newErrorWithPos(ctx.TokenStream, "unexpected token after identifier: %s", ctx.TokenStream.Current().Type)
		}

		// Если идентификатор является именем языка, это должен обрабатывать QualifiedVariableHandler
		// Проверяем, является ли идентификатор поддерживаемым языком
		languageRegistry := CreateDefaultLanguageRegistry()
		if _, err := languageRegistry.ResolveAlias(identifierToken.Value); err == nil {
			// Это имя языка, но без DOT - позволяем другим обработчикам попробовать
			return nil, nil
		}

		// Если нет знака присваивания и это не имя языка, позволяем VariableReadHandler обработать
		// (AssignmentHandler теперь не обрабатывает чтение переменных - это делает VariableReadHandler)
		return nil, nil
	}

	// Потребляем знак присваивания
	assignToken := ctx.TokenStream.Consume()
	if assignToken.Type != lexer.TokenAssign {
		return nil, newErrorWithTokenPos(assignToken, "expected '=', got %s", assignToken.Type)
	}

	// Обрабатываем значение
	if !ctx.TokenStream.HasMore() {
		return nil, newErrorWithPos(ctx.TokenStream, "expected value after '='")
	}

	var value ast.Expression
	var err error

	// Проверяем, является ли значение вызовом функции LanguageCall или сложным выражением
	currentToken := ctx.TokenStream.Current()

	if h.verbose {
		fmt.Printf("DEBUG: AssignmentHandler - parsing value, currentToken: %v (%s)\n", currentToken, currentToken.Type)
	}

	// ПЕРВОЕ - проверяем, может ли быть присваивание (идентификатор/язык + =)
	// Это нужно для цепочного присваивания (a = b = 3) и (lua.x = lua.y = 5)
	if currentToken.Type == lexer.TokenIdentifier || currentToken.IsLanguageToken() {
		if h.verbose {
			fmt.Printf("DEBUG: AssignmentHandler - currentToken is identifier or language token, checking for chained assignment\n")
		}
		// Сохраняем позицию для проверки
		savedPos := ctx.TokenStream.Position()
		ctx.TokenStream.Consume() // потребляем идентификатор или язык
		
		// Для языковых токенов нужно пропустить .identifier часть
		shouldCheck := true
		if currentToken.IsLanguageToken() {
			// Проверяем, есть ли точка и идентификатор после языка
			if ctx.TokenStream.HasMore() && ctx.TokenStream.Current().Type == lexer.TokenDot {
				ctx.TokenStream.Consume() // потребляем точку
				if ctx.TokenStream.HasMore() && ctx.TokenStream.Current().Type == lexer.TokenIdentifier {
					ctx.TokenStream.Consume() // потребляем идентификатор
				} else {
					shouldCheck = false
				}
			} else {
				shouldCheck = false
			}
		}
		
		if shouldCheck && ctx.TokenStream.HasMore() && ctx.TokenStream.Current().Type == lexer.TokenAssign {
			// Это цепочное присваивание! Восстанавливаем позицию и обрабатываем рекурсивно
			ctx.TokenStream.SetPosition(savedPos)
			if h.verbose {
				fmt.Printf("DEBUG: AssignmentHandler - detected chained assignment, parsing recursively\n")
			}
			result, err := h.Handle(ctx)
			if err != nil {
				return nil, err
			}
			
			// result это VariableAssignment (которая реализует Expression интерфейс)
			if va, ok := result.(ast.Expression); ok {
				value = va
				if h.verbose {
					fmt.Printf("DEBUG: AssignmentHandler - successfully parsed chained assignment\n")
				}
			} else {
				return nil, fmt.Errorf("unexpected result type from chained assignment: %T", result)
			}
		} else {
			// Это не присваивание, восстанавливаем позицию
			ctx.TokenStream.SetPosition(savedPos)
			if h.verbose {
				fmt.Printf("DEBUG: AssignmentHandler - not a chained assignment, checking other cases\n")
			}
			
			if h.verbose {
				fmt.Printf("DEBUG: AssignmentHandler - current token is identifier or language token: %v (%s)\n", currentToken, currentToken.Type)
			}
			// Клонируем поток для проверки LanguageCall без потребления токенов
			clonedStream := ctx.TokenStream.Clone()
			// Пробуем распарсить как LanguageCall
			_, parseErr := h.parseLanguageCallValue(clonedStream)
			if h.verbose {
				fmt.Printf("DEBUG: AssignmentHandler - parseLanguageCallValue result: %v\n", parseErr)
			}
			if parseErr == nil {
				// Если это LanguageCall, проверяем, есть ли после него операторы
				if clonedStream.HasMore() && (clonedStream.Current().Type == lexer.TokenQuestion || isBinaryOperator(clonedStream.Current().Type)) {
					// Есть операторы после LanguageCall, используем parseComplexExpression
					value, err = h.parseComplexExpression(ctx)
					if err != nil {
						return nil, err
					}
				} else {
					// Нет операторов после LanguageCall, потребляем токены из оригинального потока
					value, err = h.parseLanguageCallValue(ctx.TokenStream)
					if err != nil {
						return nil, err
					}
				}
			} else {
				// Если не LanguageCall, это может быть сложное выражение (включая Elvis оператор)
				value, err = h.parseComplexExpression(ctx)
				if err != nil {
					return nil, err
				}
			}
		}
	} else if currentToken.Type == lexer.TokenDoubleLeftAngle {
		// Обрабатываем битовую строку (с поддержкой смешанной конкатенации)
		value, err = h.parseBitstringValue(ctx.TokenStream)
		if err != nil {
			return nil, err
		}
	} else {
		// Обрабатываем сложное выражение (может включать Elvis оператор)
		if h.verbose {
			fmt.Printf("DEBUG: AssignmentHandler - about to call parseComplexExpression, current token: %s (%s)\n", ctx.TokenStream.Current().Value, ctx.TokenStream.Current().Type)
		}
		value, err = h.parseComplexExpression(ctx)
		if err != nil {
			return nil, err
		}
		if h.verbose {
			fmt.Printf("DEBUG: AssignmentHandler - parseComplexExpression completed, current token: %s (%s)\n", ctx.TokenStream.Current().Value, ctx.TokenStream.Current().Type)
		}
	}

	// Создаем узел присваивания
	var identifier *ast.Identifier

	// Если это языковой токен и мы уже обработали квалифицированную часть (varToken не пустой),
	// создаем квалифицированный идентификатор
	if identifierToken.IsLanguageToken() && (varToken.Value != "" || len(qualifiedParts) > 0) {
		// Если значение уже было распарсено (например, сложное выражение),
		// то квалифицированная часть уже была обработана в начале метода
		// и нам нужно просто создать идентификатор на основе уже потребленных токенов
		if h.verbose {
			fmt.Printf("DEBUG: AssignmentHandler - creating qualified identifier from already consumed tokens\n")
		}

		// Преобразуем языковой токен в строку
		language := identifierToken.LanguageTokenToString()

		// Создаем квалифицированный идентификатор
		// Используем сохраненный varToken для получения правильного имени переменной
		var varName string
		if varToken.Value != "" {
			// Если varToken - это литерал (true, false), создаем специальный идентификатор
			if varToken.Type == lexer.TokenTrue || varToken.Type == lexer.TokenFalse {
				varName = varToken.Value // Используем значение литерала как имя переменной
			} else {
				varName = varToken.Value
			}
		} else if len(qualifiedParts) > 0 {
			// Используем последнюю часть из qualifiedParts
			varName = qualifiedParts[len(qualifiedParts)-1]
			varToken = lexer.Token{
				Type:   lexer.TokenIdentifier,
				Value:  varName,
				Line:   identifierToken.Line,
				Column: identifierToken.Column + len(identifierToken.Value) + 1,
			}
		} else {
			return nil, newErrorWithPos(ctx.TokenStream, "internal error: expected qualified part to be consumed")
		}

		// Для индексных присваиваний нам нужно использовать полный путь
		// Создаем QualifiedIdentifierWithPath со всеми частями, включая последнюю как имя переменной
		if len(qualifiedParts) > 0 {
			// Создаем путь без последнего элемента (который является именем переменной)
			pathParts := qualifiedParts[:len(qualifiedParts)-1]
			identifier = ast.NewQualifiedIdentifierWithPath(identifierToken, varToken, language, pathParts, varName)
		} else {
			// Простая квалифицированная переменная
			identifier = ast.NewQualifiedIdentifier(identifierToken, varToken, language, varName)
		}

		if h.verbose {
			fmt.Printf("DEBUG: AssignmentHandler - created qualified identifier: language=%s, variable=%s, Qualified=%v\n", language, varName, identifier.Qualified)
		}
	} else if identifierToken.IsLanguageToken() {

		// Отладочная информация
		if h.verbose {
			fmt.Printf("DEBUG: AssignmentHandler - +++ processing language token '%s', current token: %v\n", identifierToken.Type, identifierToken)
			if ctx.TokenStream.HasMore() {
				fmt.Printf("DEBUG: AssignmentHandler - +++ next token in stream: %v\n", ctx.TokenStream.Current())
			}
		}

		// Проверяем наличие DOT после языкового токена
		if h.verbose {
			fmt.Printf("DEBUG: AssignmentHandler - checking for DOT after language token, current token: %v\n", ctx.TokenStream.Current())
		}
		if !ctx.TokenStream.HasMore() || ctx.TokenStream.Current().Type != lexer.TokenDot {
			return nil, newErrorWithPos(ctx.TokenStream, "expected '.' after language token '%s'", identifierToken.Type)
		}

		// Потребляем DOT
		ctx.TokenStream.Consume()

		// Проверяем наличие идентификатора после DOT
		if !ctx.TokenStream.HasMore() || ctx.TokenStream.Current().Type != lexer.TokenIdentifier {
			return nil, newErrorWithPos(ctx.TokenStream, "expected identifier after '.'")
		}

		// Потребляем идентификатор
		varToken := ctx.TokenStream.Consume()

		// Преобразуем языковой токен в строку
		language := identifierToken.LanguageTokenToString()

		// Создаем квалифицированный идентификатор
		identifier = ast.NewQualifiedIdentifier(identifierToken, varToken, language, varToken.Value)
	} else {
		// Обычный идентификатор
		identifier = ast.NewIdentifier(identifierToken, identifierToken.Value)
	}

	// Если это индексное присваивание, создаем ExpressionAssignment с IndexExpression
	if hasIndexExpression {
		if h.verbose {
			fmt.Printf("DEBUG: AssignmentHandler - creating ExpressionAssignment for index assignment\n")
		}

		// Сохраняем текущую позицию ПОСЛЕ парсинга значения
		finalPos := ctx.TokenStream.Position()

		// Создаем IndexExpression - нужно распарсить индексное выражение
		// Для этого восстанавливаем позицию потока и парсим индекс
		ctx.TokenStream.SetPosition(indexStart)

		// Потребляем '['
		ctx.TokenStream.Consume()

		// Используем UnifiedExpressionParser для парсинга индексного выражения
		exprParser := NewUnifiedExpressionParser(h.verbose)
		indexExpr, err := exprParser.ParseExpression(ctx)
		if err != nil {
			return nil, newErrorWithPos(ctx.TokenStream, "failed to parse index expression: %v", err)
		}

		if h.verbose {
			fmt.Printf("DEBUG: AssignmentHandler - successfully parsed index expression: %T\n", indexExpr)
		}

		// Проверяем и потребляем ']'
		if !ctx.TokenStream.HasMore() || ctx.TokenStream.Current().Type != lexer.TokenRBracket {
			return nil, newErrorWithPos(ctx.TokenStream, "expected ']' after index expression")
		}
		ctx.TokenStream.Consume()

		// Восстанавливаем позицию ПОСЛЕ значения, чтобы корректно продвинуть поток
		ctx.TokenStream.SetPosition(finalPos)

		// Создаем IndexExpression
		indexExpression := &ast.IndexExpression{
			Object: identifier,
			Index:  indexExpr,
			Pos:    identifier.Position(),
		}

		if h.verbose {
			fmt.Printf("DEBUG: AssignmentHandler - returning ExpressionAssignment with IndexExpression\n")
		}

		// Создаем и возвращаем ExpressionAssignment
		return &ast.ExpressionAssignment{
			Left:   indexExpression,
			Assign: assignToken,
			Value:  value,
		}, nil
	}

	return ast.NewVariableAssignment(identifier, assignToken, value), nil
}

// Config возвращает конфигурацию обработчика
func (h *AssignmentHandler) Config() common.HandlerConfig {
	return h.config
}

// Name возвращает имя обработчика
func (h *AssignmentHandler) Name() string {
	return h.config.Name
}

// parseBitstringValue парсит BitstringExpression как значение в присваивании
func (h *AssignmentHandler) parseBitstringValue(tokenStream stream.TokenStream) (*ast.BitstringExpression, error) {
	// Потребляем <<
	leftAngle := tokenStream.Consume()

	// Создаем битовую строку
	bitstring := ast.NewBitstringExpression(leftAngle, lexer.Token{})
	bitstring.Pos = tokenToPosition(leftAngle)

	// Проверяем, есть ли сегменты
	if tokenStream.Current().Type != lexer.TokenDoubleRightAngle {
		// Парсим сегменты
		for {
			if h.verbose {
				fmt.Printf("DEBUG: parseBitstringValue - starting segment parsing, current token: %v\n", tokenStream.Current())
			}
			segment, err := h.parseBitstringSegment(tokenStream)
			if err != nil {
				return nil, err
			}
			bitstring.AddSegment(*segment)

			// Пропускаем NEWLINE токены после сегмента
			for tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenNewline {
				if h.verbose {
					fmt.Printf("DEBUG: parseBitstringValue - skipping NEWLINE after segment: %v\n", tokenStream.Current())
				}
				tokenStream.Consume()
			}

			// Проверяем разделитель или конец
			if h.verbose {
				fmt.Printf("DEBUG: parseBitstringValue - checking for comma or >>, current token: %v\n", tokenStream.Current())
			}
			if tokenStream.Current().Type == lexer.TokenComma {
				tokenStream.Consume() // потребляем запятую

				// Пропускаем NEWLINE токены после запятой
				for tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenNewline {
					if h.verbose {
						fmt.Printf("DEBUG: parseBitstringValue - skipping NEWLINE after comma: %v\n", tokenStream.Current())
					}
					tokenStream.Consume()
				}
			} else if tokenStream.Current().Type == lexer.TokenDoubleRightAngle {
				break
			} else {
				return nil, fmt.Errorf("expected ',' or '>>' after segment, got %s", tokenStream.Current().Type)
			}
		}
	}

	// Проверяем закрывающую >>
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenDoubleRightAngle {
		return nil, fmt.Errorf("expected '>>' to close bitstring")
	}
	rightAngle := tokenStream.Consume()
	bitstring.RightAngle = rightAngle

	return bitstring, nil
}

// parseBitstringSegment парсит один сегмент битовой строки
func (h *AssignmentHandler) parseBitstringSegment(tokenStream stream.TokenStream) (*ast.BitstringSegment, error) {
	segment := &ast.BitstringSegment{}

	if h.verbose {
		fmt.Printf("DEBUG: parseBitstringSegment - starting, current token: %v\n", tokenStream.Current())
	}

	// 1. Пропускаем NEWLINE токены перед значением
	for tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenNewline {
		if h.verbose {
			fmt.Printf("DEBUG: parseBitstringSegment - skipping NEWLINE token: %v\n", tokenStream.Current())
		}
		tokenStream.Consume()
	}

	// 2. Парсим Value
	if !tokenStream.HasMore() {
		return nil, fmt.Errorf("unexpected EOF in segment")
	}

	// Проверяем, не является ли значение вложенным битстрингом
	if tokenStream.Current().Type == lexer.TokenDoubleLeftAngle {
		if h.verbose {
			fmt.Printf("DEBUG: parseBitstringSegment - found nested bitstring as value\n")
		}
		// Обрабатываем вложенный битстринг
		var err error
		segment.Value, err = h.parseBitstringValue(tokenStream)
		if err != nil {
			return nil, fmt.Errorf("failed to parse nested bitstring value: %v", err)
		}
	} else {
		valueToken := tokenStream.Consume()
		if h.verbose {
			fmt.Printf("DEBUG: parseBitstringSegment - valueToken: %v (type: %s, value: '%s')\n", valueToken, valueToken.Type, valueToken.Value)
		}
		var err error
		segment.Value, err = h.parseExpression(tokenStream, valueToken)
		if err != nil {
			return nil, fmt.Errorf("failed to parse segment value: %v", err)
		}
	}

	// TODO: Временно отключаем проверку конкатенации - есть проблема с парсингом спецификаторов
	// Проверяем на конкатенацию: если значение - это переменная (идентификатор)
	// и нет спецификаторов ни размера, ни типа, то это конкатенация
	// if identifier, ok := segment.Value.(*ast.Identifier); ok {
	// 	// Временная отладка
	// 	// Блокируем только если нет ни размера, ни спецификаторов
	// 	if segment.Size == nil && len(segment.Specifiers) == 0 {
	// 		// Переменная без каких-либо спецификаторов - это конкатенация
	// 		return nil, fmt.Errorf("result is not a statement")
	// 	}
	// }

	// 3. Пропускаем NEWLINE токены после значения
	for tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenNewline {
		if h.verbose {
			fmt.Printf("DEBUG: parseBitstringSegment - skipping NEWLINE after value: %v\n", tokenStream.Current())
		}
		tokenStream.Consume()
	}

	// 5. Парсим опциональный размер (:Size)
	if tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenColon {
		if h.verbose {
			fmt.Printf("DEBUG: parseBitstringSegment - found colon, current token: %v\n", tokenStream.Current())
		}
		segment.ColonToken = tokenStream.Consume() // потребляем ':'

		if !tokenStream.HasMore() {
			return nil, fmt.Errorf("unexpected EOF after colon in segment")
		}

		// Пропускаем NEWLINE токены после двоеточия
		for tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenNewline {
			if h.verbose {
				fmt.Printf("DEBUG: parseBitstringSegment - skipping NEWLINE after colon: %v\n", tokenStream.Current())
			}
			tokenStream.Consume()
		}

		sizeToken := tokenStream.Consume()
		if h.verbose {
			fmt.Printf("DEBUG: parseBitstringSegment - sizeToken: %v\n", sizeToken)
		}
		var err error
		segment.Size, err = h.parseExpression(tokenStream, sizeToken)
		if err != nil {
			return nil, fmt.Errorf("failed to parse segment size: %v", err)
		}
	}

	// 5. Пропускаем NEWLINE токены перед спецификаторами
	for tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenNewline {
		tokenStream.Consume()
	}

	// 6. Парсим опциональные спецификаторы (/Specifiers)
	if tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenSlash {
		segment.SlashToken = tokenStream.Consume() // потребляем '/'

		// Пропускаем NEWLINE токены после слэша
		for tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenNewline {
			tokenStream.Consume()
		}

		// Парсим первый спецификатор
		if tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenIdentifier {
			specToken := tokenStream.Consume()
			specValue := specToken.Value

			// Проверяем, есть ли у спецификатора параметр через двоеточие (например, unit:1)
			if tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenColon {
				tokenStream.Consume() // потребляем ':'

				if !tokenStream.HasMore() {
					return nil, fmt.Errorf("unexpected EOF after colon in specifier")
				}

				// Парсим значение параметра спецификатора
				paramToken := tokenStream.Consume()
				if paramToken.Type != lexer.TokenNumber && paramToken.Type != lexer.TokenIdentifier {
					return nil, fmt.Errorf("expected number or identifier as specifier parameter, got %s", paramToken.Type)
				}

				// Комбинируем спецификатор и его параметр
				specValue = specValue + ":" + paramToken.Value
			}

			segment.Specifiers = append(segment.Specifiers, specValue)

			// Парсим дополнительные спецификаторы через дефис
			for tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenMinus {
				tokenStream.Consume() // потребляем '-'

				if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenIdentifier {
					return nil, fmt.Errorf("expected specifier after hyphen")
				}

				specToken = tokenStream.Consume()
				specValue = specToken.Value

				// Проверяем, есть ли у спецификатора параметр через двоеточие
				if tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenColon {
					tokenStream.Consume() // потребляем ':'

					if !tokenStream.HasMore() {
						return nil, fmt.Errorf("unexpected EOF after colon in specifier")
					}

					// Парсим значение параметра спецификатора
					paramToken := tokenStream.Consume()
					if paramToken.Type != lexer.TokenNumber && paramToken.Type != lexer.TokenIdentifier {
						return nil, fmt.Errorf("expected number or identifier as specifier parameter, got %s", paramToken.Type)
					}

					// Комбинируем спецификатор и его параметр
					specValue = specValue + ":" + paramToken.Value
				}

				segment.Specifiers = append(segment.Specifiers, specValue)
			}
		}
	}

	return segment, nil
}

// parseLanguageCallValue парсит LanguageCall как значение в присваивании
func (h *AssignmentHandler) parseLanguageCallValue(tokenStream stream.TokenStream) (*ast.LanguageCall, error) {
	if h.verbose {
		fmt.Printf("DEBUG: parseLanguageCallValue - ENTRY, current token: %v\n", tokenStream.Current())
	}
	// Create a language registry for resolving language aliases
	languageRegistry := CreateDefaultLanguageRegistry()

	// 1. Читаем язык (уже знаем, что это идентификатор)
	languageToken := tokenStream.Consume()
	if h.verbose {
		fmt.Printf("DEBUG: parseLanguageCallValue - consumed language token: %v\n", languageToken)
	}
	language := languageToken.Value

	// 2. Проверяем и разрешаем алиас через Language Registry
	resolvedLanguage, err := languageRegistry.ResolveAlias(language)
	if err != nil {
		return nil, fmt.Errorf("unsupported language '%s': %v", language, err)
	}

	// 3. Проверяем и потребляем DOT
	if h.verbose {
		fmt.Printf("DEBUG: parseLanguageCallValue - checking for DOT, current token: %v\n", tokenStream.Current())
	}
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenDot {
		return nil, fmt.Errorf("expected DOT after language '%s'", language)
	}
	tokenStream.Consume() // Consuming dot
	if h.verbose {
		fmt.Printf("DEBUG: parseLanguageCallValue - consumed DOT, current token: %v\n", tokenStream.Current())
	}

	// 4. Читаем имя функции (может содержать точки, например, math.sqrt)
	functionParts := []string{}

	// Читаем первый идентификатор функции
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenIdentifier {
		return nil, fmt.Errorf("expected function name after DOT")
	}
	functionToken := tokenStream.Consume()
	functionParts = append(functionParts, functionToken.Value)

	// Читаем дополнительные DOT и идентификаторы
	for tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenDot {
		tokenStream.Consume() // Consuming dot

		if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenIdentifier {
			return nil, fmt.Errorf("expected function name after dot")
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
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenLeftParen {
		return nil, fmt.Errorf("expected '(' after function name '%s'", functionName)
	}
	tokenStream.Consume() // Consuming '('

	// 6. Читаем аргументы (базовая поддержка)
	arguments := make([]ast.Expression, 0)

	if !tokenStream.HasMore() {
		return nil, fmt.Errorf("unexpected EOF after argument")
	}

	if tokenStream.Current().Type != lexer.TokenRightParen {
		// Есть хотя бы один аргумент
		for {
			if !tokenStream.HasMore() {
				return nil, fmt.Errorf("unexpected EOF in arguments")
			}

			// Читаем аргумент
			argToken := tokenStream.Current()
			var arg ast.Expression

			switch argToken.Type {
			case lexer.TokenString:
				tokenStream.Consume() // потребляем токен
				arg = &ast.StringLiteral{
					Value: argToken.Value,
					Pos:   tokenToPosition(argToken),
				}
			case lexer.TokenNumber:
				tokenStream.Consume() // потребляем токен
				numValue, parseErr := parseNumber(argToken.Value)
				if parseErr != nil {
					return nil, fmt.Errorf("invalid number format: %s", argToken.Value)
				}
				arg = createNumberLiteral(argToken, numValue)
			case lexer.TokenIdentifier:
				// Проверяем, не является ли это вложенным вызовом функции
				if tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenDot {
					// Это может быть вложенный вызов функции
					tokenStream.Consume() // потребляем идентификатор перед вызовом parseNestedFunctionCall
					nestedCall, err := h.parseNestedFunctionCall(tokenStream, argToken)
					if err != nil {
						return nil, fmt.Errorf("failed to parse nested function call argument: %v", err)
					}
					arg = nestedCall
				} else {
					tokenStream.Consume() // потребляем токен
					arg = ast.NewIdentifier(argToken, argToken.Value)
				}
			case lexer.TokenLBrace:
				// Обработка объектных литералов как аргументов функции
				// Создаем временный контекст для парсинга объекта
				tempCtx := &common.ParseContext{
					TokenStream: tokenStream,
					Parser:      nil,
					Depth:       0,
					MaxDepth:    100,
					Guard:       &simpleRecursionGuard{maxDepth: 100},
					LoopDepth:   0,
					InputStream: "",
				}
				// Используем ObjectHandler для парсинга объекта
				// Важно: не потребляем LBRACE здесь, ObjectHandler сделает это сам
				objectHandler := NewObjectHandler(100, 0)
				objectResult, err := objectHandler.Handle(tempCtx)
				if err != nil {
					return nil, fmt.Errorf("failed to parse object argument: %v", err)
				}
				if object, ok := objectResult.(*ast.ObjectLiteral); ok {
					arg = object
				} else {
					return nil, fmt.Errorf("expected ObjectLiteral, got %T", objectResult)
				}
			default:
				// Для всех остальных типов токенов, потребляем их и обрабатываем как обычно
				tokenStream.Consume()
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

	// 7. Проверяем закрывающую скобку
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenRightParen {
		return nil, fmt.Errorf("expected ')' after arguments")
	}
	tokenStream.Consume() // Consuming ')'

	// 8. Создаем узел AST
	startPos := tokenToPosition(languageToken)

	node := &ast.LanguageCall{
		Language:  resolvedLanguage,
		Function:  functionName,
		Arguments: arguments,
		Pos:       startPos,
	}

	return node, nil
}

// parseSimpleValue парсит простое значение (не LanguageCall)
func (h *AssignmentHandler) parseSimpleValue(tokenStream stream.TokenStream, token lexer.Token) (ast.Expression, error) {
	switch token.Type {
	case lexer.TokenString:
		return &ast.StringLiteral{
			Value: token.Value,
			Pos: ast.Position{
				Line:   token.Line,
				Column: token.Column,
				Offset: token.Position,
			},
		}, nil
	case lexer.TokenNumber:
		// Преобразуем строку в число с поддержкой big.Int
		numValue, err := parseNumber(token.Value)
		if err != nil {
			return nil, fmt.Errorf("invalid number format: %s", token.Value)
		}
		return createNumberLiteral(token, numValue), nil
	case lexer.TokenIdentifier:
		return ast.NewIdentifier(token, token.Value), nil
	default:
		return nil, fmt.Errorf("unsupported value type: %s", token.Type)
	}
}

// parseExpression парсит выражение (значение в сегменте) с поддержкой квалифицированных переменных
func (h *AssignmentHandler) parseExpression(tokenStream stream.TokenStream, token lexer.Token) (ast.Expression, error) {
	// Проверяем, не является ли это унарным оператором
	if h.isUnaryOperator(token.Type) {
		// Для отрицательных чисел, проверяем, следующий токен - это число
		if token.Type == lexer.TokenMinus && tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenNumber {
			// Потребляем число
			numberToken := tokenStream.Consume()

			// Парсим число и создаем отрицательный NumberLiteral
			numValue, err := parseNumber(numberToken.Value)
			if err != nil {
				return nil, fmt.Errorf("failed to parse number '%s': %v", numberToken.Value, err)
			}

			// Создаем отрицательное значение
			switch v := numValue.(type) {
			case *big.Int:
				negativeInt := new(big.Int).Neg(v)
				return createNumberLiteral(numberToken, negativeInt), nil
			case float64:
				return createNumberLiteral(numberToken, -v), nil
			default:
				return nil, fmt.Errorf("unsupported number type: %T", numValue)
			}
		}

		// Токен уже потреблен, поэтому парсим операнд напрямую
		if !tokenStream.HasMore() {
			return nil, fmt.Errorf("unexpected EOF after unary operator %s", token.Type)
		}

		// Парсим операнд
		operandToken := tokenStream.Consume()
		operand, err := h.parseExpression(tokenStream, operandToken)
		if err != nil {
			return nil, fmt.Errorf("failed to parse operand for unary operator %s: %v", token.Type, err)
		}

		// Создаем унарное выражение
		return ast.NewUnaryExpression(token.Value, operand, ast.Position{
			Line:   token.Line,
			Column: token.Column,
			Offset: token.Position,
		}), nil
	}

	switch token.Type {
	case lexer.TokenString:
		return &ast.StringLiteral{
			Value: token.Value,
			Pos: ast.Position{
				Line:   token.Line,
				Column: token.Column,
				Offset: token.Position,
			},
		}, nil
	case lexer.TokenNumber:
		if h.verbose {
			fmt.Printf("DEBUG: parseExpression - processing NUMBER token: %v (value: '%s')\n", token, token.Value)
		}
		// Используем parseNumber для поддержки big.Int, hex, binary и decimal форматов
		numValue, err := parseNumber(token.Value)
		if err != nil {
			return nil, fmt.Errorf("invalid number format: %s", token.Value)
		}

		if h.verbose {
			switch v := numValue.(type) {
			case *big.Int:
				fmt.Printf("DEBUG: parseExpression - created NumberLiteral with big.Int value: %s\n", v.String())
			case float64:
				fmt.Printf("DEBUG: parseExpression - created NumberLiteral with float64 value: %f\n", v)
			}
		}
		return createNumberLiteral(token, numValue), nil
	case lexer.TokenIdentifier:
		// Проверяем, не является ли это квалифицированной переменной или вызовом функции
		return h.parseComplexIdentifier(tokenStream, token)
	case lexer.TokenNewline:
		if h.verbose {
			fmt.Printf("DEBUG: parseExpression - got NEWLINE token: %v (value: '%s')\n", token, token.Value)
		}
		// Проверяем, содержит ли NEWLINE текст комментария
		if len(token.Value) > 0 && token.Value[0] == '#' {
			if h.verbose {
				fmt.Printf("DEBUG: parseExpression - NEWLINE contains comment, skipping\n")
			}
			// Это комментарий, пропускаем его и пробуем получить следующий токен
			if !tokenStream.HasMore() {
				return nil, fmt.Errorf("unexpected EOF after comment")
			}
			nextToken := tokenStream.Consume()
			return h.parseExpression(tokenStream, nextToken)
		}
		// Обычный NEWLINE, пропускаем и пробуем получить следующий токен
		if h.verbose {
			fmt.Printf("DEBUG: parseExpression - regular NEWLINE, skipping\n")
		}
		if !tokenStream.HasMore() {
			return nil, fmt.Errorf("unexpected EOF after newline")
		}
		nextToken := tokenStream.Consume()
		return h.parseExpression(tokenStream, nextToken)
	default:
		// Проверяем, является ли токен языковым токеном
		if token.IsLanguageToken() {
			// Это языковой токен - обрабатываем как квалифицированную переменную
			return h.parseQualifiedIdentifierOrFunctionCall(tokenStream, token)
		}
		return nil, fmt.Errorf("unsupported expression type: %s", token.Type)
	case lexer.TokenLeftParen:
		// Обработка выражений в скобках - используем BinaryExpressionHandler
		return h.parseParenthesizedExpression(tokenStream, token)
	}
}

// parseParenthesizedExpression парсит выражение в скобках
func (h *AssignmentHandler) parseParenthesizedExpression(tokenStream stream.TokenStream, leftParenToken lexer.Token) (ast.Expression, error) {
	// Создаем временный контекст для парсинга выражения в скобках
	tempCtx := &common.ParseContext{
		TokenStream: tokenStream,
		Parser:      nil,
		Depth:       0,
		MaxDepth:    100,
		Guard:       &simpleRecursionGuard{maxDepth: 100},
		LoopDepth:   0,
		InputStream: "",
	}

	// Используем BinaryExpressionHandler для парсинга выражения внутри скобок
	binaryHandler := NewBinaryExpressionHandler(config.ConstructHandlerConfig{})

	// Сначала парсим первый операнд внутри скобок
	leftOperand, err := binaryHandler.parseOperand(tempCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to parse first operand in parentheses: %v", err)
	}

	// Затем парсим полное выражение, начиная с первого операнда
	expr, err := binaryHandler.ParseFullExpression(tempCtx, leftOperand)
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

// parseComplexIdentifier парсит сложный идентификатор (квалифицированная переменная или вызов функции)
func (h *AssignmentHandler) parseComplexIdentifier(tokenStream stream.TokenStream, firstToken lexer.Token) (ast.Expression, error) {
	// Проверяем, есть ли DOT после идентификатора
	if tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenDot {
		// Это может быть квалифицированная переменная или вызов функции
		return h.parseQualifiedIdentifierOrFunctionCall(tokenStream, firstToken)
	} else {
		// Простой идентификатор
		return ast.NewIdentifier(firstToken, firstToken.Value), nil
	}
}

// parseQualifiedIdentifierOrFunctionCall парсит квалифицированную переменную или вызов функции
func (h *AssignmentHandler) parseQualifiedIdentifierOrFunctionCall(tokenStream stream.TokenStream, languageToken lexer.Token) (ast.Expression, error) {
	// Проверяем, что язык поддерживается
	languageRegistry := CreateDefaultLanguageRegistry()
	language := languageToken.Value

	// Используем встроенный метод токена для преобразования в строковое представление языка
	language = languageToken.LanguageTokenToString()

	if h.verbose {
		fmt.Printf("DEBUG: parseQualifiedIdentifierOrFunctionCall - language token type: %v, resolved language: %v\n", languageToken.Type, language)
	}

	resolvedLanguage, err := languageRegistry.ResolveAlias(language)
	if err != nil {
		return nil, fmt.Errorf("unsupported language '%s': %v", language, err)
	}

	// Потребляем DOT
	if h.verbose {
		fmt.Printf("DEBUG: parseQualifiedIdentifierOrFunctionCall - checking for DOT, current token: %v\n", tokenStream.Current())
	}
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenDot {
		return nil, fmt.Errorf("expected DOT after language '%s'", language)
	}
	tokenStream.Consume() // Consuming DOT
	if h.verbose {
		fmt.Printf("DEBUG: parseQualifiedIdentifierOrFunctionCall - consumed DOT\n")
	}

	// Читаем следующий идентификатор
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenIdentifier {
		return nil, fmt.Errorf("expected identifier after DOT")
	}
	nextToken := tokenStream.Consume()

	// Собираем все части имени функции через DOT
	functionParts := []string{nextToken.Value}

	// Читаем дополнительные DOT и идентификаторы
	for tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenDot {
		tokenStream.Consume() // Consuming dot

		if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenIdentifier {
			return nil, fmt.Errorf("expected identifier after dot")
		}
		nextToken = tokenStream.Consume()
		functionParts = append(functionParts, nextToken.Value)
	}

	// Проверяем, есть ли скобка (вызов функции)
	if tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenLeftParen {
		// Это вызов функции
		return h.parseFunctionCallWithParts(tokenStream, languageToken, functionParts, resolvedLanguage)
	} else {
		// Это квалифицированная переменная (language.part1.part2.variable)
		if len(functionParts) == 1 {
			// Простая квалифицированная переменная (language.variable)
			variableName := functionParts[0]
			identifier := ast.NewQualifiedIdentifier(languageToken, nextToken, resolvedLanguage, variableName)
			return identifier, nil
		} else {
			// Квалифицированная переменная с путем
			variableName := functionParts[len(functionParts)-1]
			actualPathParts := functionParts[:len(functionParts)-1]
			variableToken := lexer.Token{
				Type:     lexer.TokenIdentifier,
				Value:    variableName,
				Line:     languageToken.Line,
				Column:   languageToken.Column + len(languageToken.Value) + 1, // Приблизительно
				Position: languageToken.Position,
			}
			identifier := ast.NewQualifiedIdentifierWithPath(languageToken, variableToken, resolvedLanguage, actualPathParts, variableName)
			return identifier, nil
		}
	}
}

// parseFunctionCall парсит вызов функции
func (h *AssignmentHandler) parseFunctionCall(tokenStream stream.TokenStream, languageToken lexer.Token, functionToken lexer.Token, resolvedLanguage string) (ast.Expression, error) {
	// Потребляем открывающую скобку
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenLeftParen {
		return nil, fmt.Errorf("expected '(' after function name")
	}
	tokenStream.Consume()

	// Собираем имя функции (может включать дополнительные части через DOT)
	functionParts := []string{functionToken.Value}

	// Читаем дополнительные DOT и идентификаторы для имени функции
	for tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenDot {
		tokenStream.Consume() // Consuming dot

		if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenIdentifier {
			return nil, fmt.Errorf("expected function name after dot")
		}
		functionToken = tokenStream.Consume()
		functionParts = append(functionParts, functionToken.Value)
	}

	// Собираем полное имя функции
	functionName := ""
	for i, part := range functionParts {
		if i > 0 {
			functionName += "."
		}
		functionName += part
	}

	// Парсим аргументы
	arguments := make([]ast.Expression, 0)

	if tokenStream.Current().Type != lexer.TokenRightParen {
		// Есть аргументы
		for {
			if !tokenStream.HasMore() {
				return nil, fmt.Errorf("unexpected EOF in function arguments")
			}

			// Читаем аргумент
			argToken := tokenStream.Consume()
			arg, err := h.parseExpression(tokenStream, argToken)
			if err != nil {
				return nil, fmt.Errorf("failed to parse function argument: %v", err)
			}
			arguments = append(arguments, arg)

			// Проверяем разделитель
			if !tokenStream.HasMore() {
				return nil, fmt.Errorf("unexpected EOF after function argument")
			}

			if tokenStream.Current().Type == lexer.TokenComma {
				tokenStream.Consume() // Consuming comma
			} else if tokenStream.Current().Type == lexer.TokenRightParen {
				break
			} else {
				return nil, fmt.Errorf("expected ',' or ')' in function arguments, got %s", tokenStream.Current().Type)
			}
		}
	}

	// Потребляем закрывающую скобку
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenRightParen {
		return nil, fmt.Errorf("expected ')' to close function call")
	}
	tokenStream.Consume()

	// Создаем узел LanguageCall
	startPos := tokenToPosition(languageToken)
	return &ast.LanguageCall{
		Language:  resolvedLanguage,
		Function:  functionName,
		Arguments: arguments,
		Pos:       startPos,
	}, nil
}

// parseFunctionCallWithParts парсит вызов функции с уже собранными частями имени
func (h *AssignmentHandler) parseFunctionCallWithParts(tokenStream stream.TokenStream, languageToken lexer.Token, functionParts []string, resolvedLanguage string) (ast.Expression, error) {
	// Потребляем открывающую скобку
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenLeftParen {
		return nil, fmt.Errorf("expected '(' after function name")
	}
	tokenStream.Consume()

	// Собираем полное имя функции
	functionName := ""
	for i, part := range functionParts {
		if i > 0 {
			functionName += "."
		}
		functionName += part
	}

	// Парсим аргументы
	arguments := make([]ast.Expression, 0)

	if tokenStream.Current().Type != lexer.TokenRightParen {
		// Есть аргументы
		for {
			if !tokenStream.HasMore() {
				return nil, fmt.Errorf("unexpected EOF in function arguments")
			}

			// Читаем аргумент
			argToken := tokenStream.Current()
			var arg ast.Expression
			var err error

			// Проверяем, не является ли это унарным оператором
			if h.isUnaryOperator(argToken.Type) {
				// Создаем временный контекст для парсинга унарного выражения
				tempCtx := &common.ParseContext{
					TokenStream: tokenStream,
					Parser:      nil,
					Depth:       0,
					MaxDepth:    100,
					Guard:       &simpleRecursionGuard{maxDepth: 100},
					LoopDepth:   0,
					InputStream: "",
				}
				// Используем UnaryExpressionHandler для парсинга унарных выражений
				unaryHandler := NewUnaryExpressionHandler(config.ConstructHandlerConfig{ConstructType: common.ConstructUnaryExpression})
				result, err := unaryHandler.Handle(tempCtx)
				if err != nil {
					return nil, fmt.Errorf("failed to parse function argument: %v", err)
				}

				// Преобразуем interface{} в ast.Expression
				if expr, ok := result.(ast.Expression); ok {
					arg = expr
				} else {
					return nil, fmt.Errorf("unary expression handler returned non-expression type: %T", result)
				}
			} else {
				// Потребляем токен только если это не унарный оператор (UnaryExpressionHandler потребляет его сам)
				tokenStream.Consume()
				arg, err = h.parseExpression(tokenStream, argToken)
				if err != nil {
					return nil, fmt.Errorf("failed to parse function argument: %v", err)
				}
			}
			arguments = append(arguments, arg)

			// Проверяем разделитель
			if !tokenStream.HasMore() {
				return nil, fmt.Errorf("unexpected EOF after function argument")
			}

			if tokenStream.Current().Type == lexer.TokenComma {
				tokenStream.Consume() // Consuming comma
			} else if tokenStream.Current().Type == lexer.TokenRightParen {
				break
			} else {
				return nil, fmt.Errorf("expected ',' or ')' in function arguments, got %s", tokenStream.Current().Type)
			}
		}
	}

	// Потребляем закрывающую скобку
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenRightParen {
		return nil, fmt.Errorf("expected ')' to close function call")
	}
	tokenStream.Consume()

	// Создаем узел LanguageCall
	startPos := tokenToPosition(languageToken)
	return &ast.LanguageCall{
		Language:  resolvedLanguage,
		Function:  functionName,
		Arguments: arguments,
		Pos:       startPos,
	}, nil
}

// parseQualifiedVariableWithPath парсит квалифицированную переменную с путем (language.part1.part2.variable)
func (h *AssignmentHandler) parseQualifiedVariableWithPath(tokenStream stream.TokenStream, languageToken lexer.Token, firstPartToken lexer.Token, resolvedLanguage string) (ast.Expression, error) {
	pathParts := []string{firstPartToken.Value}

	// Читаем дополнительные части пути
	for {
		// Потребляем DOT
		if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenDot {
			break
		}
		tokenStream.Consume()

		// Читаем следующую часть пути
		if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenIdentifier {
			return nil, fmt.Errorf("expected identifier after DOT")
		}
		partToken := tokenStream.Consume()
		pathParts = append(pathParts, partToken.Value)

		// Проверяем, следующий токен - если не DOT, то это конец пути
		if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenDot {
			break
		}
	}

	// Последняя часть пути - это имя переменной
	if len(pathParts) == 0 {
		return nil, fmt.Errorf("empty path in qualified variable")
	}

	variableName := pathParts[len(pathParts)-1]
	actualPathParts := pathParts[:len(pathParts)-1]
	variableToken := lexer.Token{
		Type:     lexer.TokenIdentifier,
		Value:    variableName,
		Line:     languageToken.Line,
		Column:   languageToken.Column + len(languageToken.Value) + 1, // Приблизительно
		Position: languageToken.Position,
	}

	// Создаем QualifiedIdentifier с путем
	identifier := ast.NewQualifiedIdentifierWithPath(languageToken, variableToken, resolvedLanguage, actualPathParts, variableName)
	return identifier, nil
}

// simpleRecursionGuard - простая реализация защиты от рекурсии
type simpleRecursionGuard struct {
	maxDepth     int
	currentDepth int
}

func (rg *simpleRecursionGuard) Enter() error {
	if rg.currentDepth >= rg.maxDepth {
		return fmt.Errorf("maximum recursion depth exceeded: %d", rg.maxDepth)
	}
	rg.currentDepth++
	return nil
}

func (rg *simpleRecursionGuard) Exit() {
	if rg.currentDepth > 0 {
		rg.currentDepth--
	}
}

func (rg *simpleRecursionGuard) CurrentDepth() int {
	return rg.currentDepth
}

func (rg *simpleRecursionGuard) MaxDepth() int {
	return rg.maxDepth
}

// isBitstringConcatenation проверяет, является ли битовая строка конкатенацией
// (переменные без спецификаторов размера и типа)
func (h *AssignmentHandler) isBitstringConcatenation(tokenStream stream.TokenStream) bool {
	// Клонируем поток, чтобы не потреблять токены
	clone := tokenStream.Clone()

	// Пропускаем <<
	clone.Consume()

	// Флаги для отслеживания контекста
	afterSlash := false // true после / (спецификаторы типа)

	// Проверяем токены на наличие идентификаторов без спецификаторов
	peekAhead := 15 // Проверяем до 15 токенов вперед

	for i := 0; i < peekAhead && clone.HasMore(); i++ {
		token := clone.Current()

		// Обновляем контекст
		if token.Type == lexer.TokenSlash {
			afterSlash = true
		} else if token.Type == lexer.TokenComma {
			afterSlash = false
		} else if token.Type == lexer.TokenColon {
			afterSlash = false
		}

		if token.Type == lexer.TokenDoubleRightAngle {
			break // Дошли до конца битовой строки
		}

		if token.Type == lexer.TokenIdentifier {
			// Если идентификатор идет после / - это спецификатор типа, нормально
			if afterSlash {
				clone.Consume()
				continue
			}

			// Идентификатор не после / - проверяем, есть ли спецификатор размера
			clone.Consume()

			// Проверяем следующий токен
			if clone.HasMore() {
				nextToken := clone.Current()
				if nextToken.Type != lexer.TokenColon {
					// Идентификатор без : после него - проверяем дальше
					// Пропускаем токены до следующего разделителя или конца
					for clone.HasMore() && clone.Current().Type != lexer.TokenComma &&
						clone.Current().Type != lexer.TokenSlash &&
						clone.Current().Type != lexer.TokenDoubleRightAngle {
						clone.Consume()
					}

					// Если после идентификатора не было / перед разделителем - это конкатенация
					if clone.HasMore() && clone.Current().Type != lexer.TokenSlash {
						return true
					}
				}
				// Идентификатор со спецификатором размера - это нормально
			} else {
				// Идентификатор в конце без спецификаторов - конкатенация
				return true
			}
		} else if token.Type == lexer.TokenNumber {
			clone.Consume()

			// Проверяем следующий токен для чисел
			if clone.HasMore() {
				nextToken := clone.Current()
				// Если после числа не идет :, /, ,, или >> - это подозрительно
				if nextToken.Type != lexer.TokenColon && nextToken.Type != lexer.TokenSlash &&
					nextToken.Type != lexer.TokenComma && nextToken.Type != lexer.TokenDoubleRightAngle {
					// Это может быть конкатенацией (число без спецификатора)
					return true
				}
			}
		} else {
			clone.Consume() // Пропускаем другие токены
		}
	}

	return false
}

// isUnaryOperator проверяет, является ли токен унарным оператором
func (h *AssignmentHandler) isUnaryOperator(tokenType lexer.TokenType) bool {
	switch tokenType {
	case lexer.TokenPlus, lexer.TokenMinus, lexer.TokenNot, lexer.TokenTilde, lexer.TokenAt:
		return true
	default:
		return false
	}
}

// parseComplexExpression парсит сложное выражение, которое может включать Elvis оператор
func (h *AssignmentHandler) parseComplexExpression(ctx *common.ParseContext) (ast.Expression, error) {
	tokenStream := ctx.TokenStream

	if !tokenStream.HasMore() {
		return nil, fmt.Errorf("unexpected EOF in complex expression")
	}

	// Парсим первый операнд как простой идентификатор или литерал
	var leftExpr ast.Expression

	currentToken := tokenStream.Current()

	// Проверяем, не является ли это унарным оператором
	if h.isUnaryOperator(currentToken.Type) {
		// Для отрицательных чисел, проверяем, следующий токен - это число
		if currentToken.Type == lexer.TokenMinus && tokenStream.HasMore() && tokenStream.Peek().Type == lexer.TokenNumber {
			// Потребляем минус
			tokenStream.Consume()

			// Потребляем число
			numberToken := tokenStream.Consume()

			// Парсим число и создаем отрицательный NumberLiteral
			numValue, err := parseNumber(numberToken.Value)
			if err != nil {
				return nil, fmt.Errorf("failed to parse number '%s': %v", numberToken.Value, err)
			}

			// Создаем отрицательное значение
			switch v := numValue.(type) {
			case *big.Int:
				negativeInt := new(big.Int).Neg(v)
				leftExpr = createNumberLiteral(numberToken, negativeInt)
			case float64:
				leftExpr = createNumberLiteral(numberToken, -v)
			default:
				return nil, fmt.Errorf("unsupported number type: %T", numValue)
			}
			// Продолжаем парсинг после отрицательного числа
			goto continue_parsing
		}

		// Используем UnaryExpressionHandler для парсинга унарных выражений
		unaryHandler := NewUnaryExpressionHandler(config.ConstructHandlerConfig{ConstructType: common.ConstructUnaryExpression})
		result, err := unaryHandler.Handle(ctx)
		if err != nil {
			return nil, err
		}

		// Преобразуем interface{} в ast.Expression
		if expr, ok := result.(ast.Expression); ok {
			leftExpr = expr
			// Продолжаем парсинг после унарного выражения
			goto continue_parsing
		}

		return nil, fmt.Errorf("unary expression handler returned non-expression type: %T", result)
	}

	switch currentToken.Type {
	case lexer.TokenIdentifier:
		// Проверяем, не является ли это вызовом builtin функции
		if tokenStream.HasMore() && tokenStream.Peek().Type == lexer.TokenLeftParen {
			// Это вызов builtin функции
			builtinHandler := NewBuiltinFunctionHandlerWithVerbose(config.ConstructHandlerConfig{}, h.verbose)
			result, err := builtinHandler.Handle(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to parse builtin function call: %v", err)
			}

			builtinCall, ok := result.(*ast.BuiltinFunctionCall)
			if !ok {
				return nil, fmt.Errorf("expected BuiltinFunctionCall, got %T", result)
			}

			leftExpr = builtinCall
		} else {
			// Простой идентификатор
			leftExpr = ast.NewIdentifier(currentToken, currentToken.Value)
			tokenStream.Consume()
		}
	case lexer.TokenNil:
		leftExpr = ast.NewNilLiteral(currentToken)
		tokenStream.Consume()
	default:
		// Проверяем, является ли токен языковым токеном
		if currentToken.IsLanguageToken() {
			// Обрабатываем квалифицированные переменные в начале выражения
			if h.verbose {
				fmt.Printf("DEBUG: parseComplexExpression - processing language token: %v\n", currentToken)
			}

			// Проверяем, есть ли следующий токен и является ли он DOT
			if tokenStream.HasMore() && tokenStream.Peek().Type == lexer.TokenDot {
				// Это может быть квалифицированная переменная или вызов функции
				// Потребляем языковой токен перед вызовом parseQualifiedIdentifierOrFunctionCall
				tokenStream.Consume()
				var err error
				leftExpr, err = h.parseQualifiedIdentifierOrFunctionCall(tokenStream, currentToken)
				if err != nil {
					if h.verbose {
						fmt.Printf("DEBUG: parseComplexExpression - parseQualifiedIdentifierOrFunctionCall error: %v\n", err)
					}
					return nil, err
				}
				if h.verbose {
					fmt.Printf("DEBUG: parseComplexExpression - successfully parsed qualified identifier: %v\n", leftExpr)
				}
			} else {
				// Это просто языковой токен без DOT - не поддерживается в этом контексте
				return nil, fmt.Errorf("language token '%s' not followed by DOT in expression context", currentToken.Value)
			}

			// Проверяем наличие индексного выражения [index] после квалифицированного идентификатора
			if tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenLBracket {
				if h.verbose {
					fmt.Printf("DEBUG: parseComplexExpression - found LBRACKET after qualified identifier, parsing index expression\n")
				}

				// Потребляем открывающую скобку
				tokenStream.Consume()

				// Парсим индексное выражение
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

				// Сначала парсим первый операнд индексного выражения
				leftOperand, err := binaryHandler.parseOperand(tempCtx)
				if err != nil {
					return nil, fmt.Errorf("failed to parse first operand in index expression: %v", err)
				}

				// Затем парсим полное индексное выражение, начиная с первого операнда
				indexExpr, err := binaryHandler.ParseFullExpression(tempCtx, leftOperand)
				if err != nil {
					return nil, fmt.Errorf("failed to parse index expression: %v", err)
				}

				if h.verbose {
					fmt.Printf("DEBUG: parseComplexExpression - successfully parsed index expression: %T\n", indexExpr)
				}

				// Проверяем наличие закрывающей скобки
				if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenRBracket {
					return nil, fmt.Errorf("expected ']' after index expression")
				}

				// Потребляем закрывающую скобку
				tokenStream.Consume()

				// Создаем индексное выражение
				leftExpr = ast.NewIndexExpression(leftExpr, indexExpr, leftExpr.Position())

				if h.verbose {
					fmt.Printf("DEBUG: parseComplexExpression - created index expression: %T\n", leftExpr)
				}
			}
			// После парсинга языкового токена, продолжаем проверять операторы
			goto check_operators
		} else {
			return nil, fmt.Errorf("unsupported expression start: %s", currentToken.Type)
		}
	case lexer.TokenString:
		leftExpr = &ast.StringLiteral{
			Value: currentToken.Value,
			Pos: ast.Position{
				Line:   currentToken.Line,
				Column: currentToken.Column,
				Offset: currentToken.Position,
			},
		}
		tokenStream.Consume()
	case lexer.TokenNumber:
		// Используем parseNumber для поддержки big.Int, hex, binary и decimal форматов
		numValue, err := parseNumber(currentToken.Value)
		if err != nil {
			return nil, fmt.Errorf("invalid number format: %s", currentToken.Value)
		}

		leftExpr = createNumberLiteral(currentToken, numValue)
		tokenStream.Consume()
	case lexer.TokenTrue, lexer.TokenFalse:
		boolValue := currentToken.Type == lexer.TokenTrue
		leftExpr = &ast.BooleanLiteral{
			Value: boolValue,
			Pos: ast.Position{
				Line:   currentToken.Line,
				Column: currentToken.Column,
				Offset: currentToken.Position,
			},
		}
		tokenStream.Consume()
	case lexer.TokenLeftParen:
		// Обработка выражений в скобках
		tokenStream.Consume() // потребляем '('

		// Используем BinaryExpressionHandler для парсинга выражения внутри скобок
		binaryHandler := NewBinaryExpressionHandler(config.ConstructHandlerConfig{})

		// Сначала парсим первый операнд внутри скобок
		leftOperand, err := binaryHandler.parseOperand(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to parse first operand in parentheses: %v", err)
		}

		// Затем парсим полное выражение, начиная с первого операнда
		expr, err := binaryHandler.ParseFullExpression(ctx, leftOperand)
		if err != nil {
			return nil, fmt.Errorf("failed to parse expression in parentheses: %v", err)
		}

		// Проверяем и потребляем закрывающую скобку
		if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenRightParen {
			return nil, fmt.Errorf("expected ')' after expression")
		}
		tokenStream.Consume() // потребляем ')'

		leftExpr = expr
	case lexer.TokenLBrace:
		// Обработка объектов как сложных выражений
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
			return nil, fmt.Errorf("failed to parse object expression: %v", err)
		}
		if object, ok := objectResult.(*ast.ObjectLiteral); ok {
			leftExpr = object
		} else {
			return nil, fmt.Errorf("expected ObjectLiteral, got %T", objectResult)
		}
	case lexer.TokenLBracket:
		// Обработка массивов как сложных выражений
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
			return nil, fmt.Errorf("failed to parse array expression: %v", err)
		}
		if array, ok := arrayResult.(*ast.ArrayLiteral); ok {
			leftExpr = array
		} else {
			return nil, fmt.Errorf("expected ArrayLiteral, got %T", arrayResult)
		}
		
		// Парсим все индексные выражения [index][index]... в цикле
		binaryHandler := NewBinaryExpressionHandler(config.ConstructHandlerConfig{})
		for tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenLBracket {
			indexExpr, err := binaryHandler.ParseIndexExpression(ctx, leftExpr)
			if err != nil {
				return nil, err
			}
			leftExpr = indexExpr
		}
	}

check_operators:
	// Проверяем наличие бинарных операторов (включая >, <, == и т.д.)
	if tokenStream.HasMore() && isBinaryOperator(tokenStream.Current().Type) {
		if h.verbose {
			fmt.Printf("DEBUG: parseComplexExpression - found binary operator: %s (%s)\n", tokenStream.Current().Value, tokenStream.Current().Type)
		}
		// Используем старую логику precedence climbing для совместимости
		binaryHandler := NewBinaryExpressionHandlerWithVerbose(config.ConstructHandlerConfig{}, h.verbose)
		result, err := binaryHandler.ParseFullExpression(ctx, leftExpr)
		return result, err
	}

	// Проверяем наличие тернарного оператора (?) - это должно быть после всех случаев парсинга leftExpr
	if tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenQuestion {
		if h.verbose {
			fmt.Printf("DEBUG: parseComplexExpression - found ternary operator: ?\n")
		}
		// Используем BinaryExpressionHandler для парсинга тернарных выражений
		binaryHandler := NewBinaryExpressionHandler(config.ConstructHandlerConfig{})
		result, err := binaryHandler.ParseTernaryExpression(ctx, leftExpr)
		if h.verbose {
			fmt.Printf("DEBUG: parseComplexExpression - ParseTernaryExpression result: %T, error: %v\n", result, err)
		}
		return result, err
	}

continue_parsing:
	// Проверяем наличие бинарных операторов (включая >, <, == и т.д.)
	if h.verbose {
		fmt.Printf("DEBUG: continue_parsing - current token: %s (%s), hasMore: %v\n", tokenStream.Current().Value, tokenStream.Current().Type, tokenStream.HasMore())
	}
	if tokenStream.HasMore() && isBinaryOperator(tokenStream.Current().Type) {
		if h.verbose {
			fmt.Printf("DEBUG: parseComplexExpression - found binary operator after unary: %s (%s)\n", tokenStream.Current().Value, tokenStream.Current().Type)
		}
		// Используем старую логику precedence climbing для совместимости
		binaryHandler := NewBinaryExpressionHandlerWithVerbose(config.ConstructHandlerConfig{}, h.verbose)
		result, err := binaryHandler.ParseFullExpression(ctx, leftExpr)
		return result, err
	}

	// Проверяем наличие тернарного оператора (?) - это должно быть после всех случаев парсинга leftExpr
	if tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenQuestion {
		if h.verbose {
			fmt.Printf("DEBUG: parseComplexExpression - found ternary operator after unary: ?\n")
		}
		// Используем BinaryExpressionHandler для парсинга тернарных выражений
		binaryHandler := NewBinaryExpressionHandler(config.ConstructHandlerConfig{})
		result, err := binaryHandler.ParseTernaryExpression(ctx, leftExpr)
		if h.verbose {
			fmt.Printf("DEBUG: parseComplexExpression - ParseTernaryExpression result: %T, error: %v\n", result, err)
		}
		return result, err
	}

	// Если нет операторов, возвращаем левый операнд
	if h.verbose {
		fmt.Printf("DEBUG: parseComplexExpression - returning leftExpr: %T, current token: %s (%s)\n", leftExpr, tokenStream.Current().Value, tokenStream.Current().Type)
		if tokenStream.HasMore() {
			fmt.Printf("DEBUG: parseComplexExpression - next token available: %s (%s)\n", tokenStream.Peek().Value, tokenStream.Peek().Type)
		} else {
			fmt.Printf("DEBUG: parseComplexExpression - no more tokens available\n")
		}
	}

	// Special check: if we're in a parenthesized context and the current token is RIGHT_PAREN,
	// we should NOT consume it - it belongs to the parent parser
	if tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenRightParen {
		if h.verbose {
			fmt.Printf("DEBUG: parseComplexExpression - detected RIGHT_PAREN, not consuming it (belongs to parent)\n")
		}
	}

	return leftExpr, nil
}

// parseNestedFunctionCall парсит вложенный вызов функции как аргумент
func (h *AssignmentHandler) parseNestedFunctionCall(tokenStream stream.TokenStream, firstToken lexer.Token) (*ast.LanguageCall, error) {
	// Проверяем, что первый токен - идентификатор языка
	languageRegistry := CreateDefaultLanguageRegistry()
	language := firstToken.Value
	resolvedLanguage, err := languageRegistry.ResolveAlias(language)
	if err != nil {
		return nil, fmt.Errorf("unsupported language '%s': %v", language, err)
	}

	// Потребляем DOT
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenDot {
		return nil, fmt.Errorf("expected DOT after language '%s'", language)
	}
	tokenStream.Consume() // Consuming DOT

	// Читаем имя функции (может содержать точки, например, string.lower)
	functionParts := []string{}

	// Читаем первый идентификатор функции
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenIdentifier {
		return nil, fmt.Errorf("expected function name after DOT")
	}
	functionToken := tokenStream.Consume()
	functionParts = append(functionParts, functionToken.Value)

	// Читаем дополнительные DOT и идентификаторы
	for tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenDot {
		tokenStream.Consume() // Consuming dot

		if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenIdentifier {
			return nil, fmt.Errorf("expected function name after dot")
		}
		functionToken = tokenStream.Consume()
		functionParts = append(functionParts, functionToken.Value)
	}

	// Собираем полное имя функции (например, "string.lower")
	functionName := ""
	for i, part := range functionParts {
		if i > 0 {
			functionName += "."
		}
		functionName += part
	}

	// Проверяем и потребляем открывающую скобку
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenLeftParen {
		return nil, fmt.Errorf("expected '(' after function name '%s'", functionName)
	}
	tokenStream.Consume() // Consuming '('

	// Читаем аргументы
	arguments := make([]ast.Expression, 0)

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
					Pos:   tokenToPosition(argToken),
				}
			case lexer.TokenNumber:
				numValue, parseErr := parseNumber(argToken.Value)
				if parseErr != nil {
					return nil, fmt.Errorf("invalid number format: %s", argToken.Value)
				}
				arg = createNumberLiteral(argToken, numValue)
			case lexer.TokenIdentifier:
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
	startPos := tokenToPosition(firstToken)

	node := &ast.LanguageCall{
		Language:  resolvedLanguage,
		Function:  functionName,
		Arguments: arguments,
		Pos:       startPos,
	}

	return node, nil
}

// handlePropertyAccessAfterIndex обрабатывает доступ к свойству после индексного выражения (py.data.users[0].age = value)
func (h *AssignmentHandler) handlePropertyAccessAfterIndex(ctx *common.ParseContext, identifierToken, varToken lexer.Token, qualifiedParts []string, indexStart int) (interface{}, error) {
	tokenStream := ctx.TokenStream

	// Потребляем DOT после индексного выражения
	dotToken := tokenStream.Consume()

	// Проверяем, есть ли идентификатор после DOT
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenIdentifier {
		return nil, newErrorWithPos(ctx.TokenStream, "expected identifier after DOT for property access")
	}

	// Потребляем идентификатор свойства
	propertyToken := tokenStream.Consume()

	// Проверяем наличие оператора присваивания
	if !tokenStream.HasMore() || (tokenStream.Current().Type != lexer.TokenAssign && tokenStream.Current().Type != lexer.TokenColonEquals) {
		return nil, newErrorWithPos(ctx.TokenStream, "expected assignment operator after property access")
	}

	// Потребляем знак присваивания
	assignToken := tokenStream.Consume()

	// Обрабатываем значение
	if !tokenStream.HasMore() {
		return nil, newErrorWithPos(ctx.TokenStream, "expected value after '='")
	}

	var value ast.Expression
	var err error

	// Парсим значение
	if h.verbose {
		fmt.Printf("DEBUG: handlePropertyAccessAfterIndex - about to call parseComplexExpression, current token: %s (%s)\n", tokenStream.Current().Value, tokenStream.Current().Type)
	}
	value, err = h.parseComplexExpression(ctx)
	if err != nil {
		return nil, err
	}

	// Создаем квалифицированный идентификатор
	language := identifierToken.LanguageTokenToString()
	var varName string
	if varToken.Value != "" {
		varName = varToken.Value
	} else if len(qualifiedParts) > 0 {
		varName = qualifiedParts[len(qualifiedParts)-1]
	} else {
		return nil, newErrorWithPos(ctx.TokenStream, "internal error: expected qualified part to be consumed")
	}

	var identifier *ast.Identifier
	if len(qualifiedParts) > 1 {
		// Создаем путь без последнего элемента (который является именем переменной)
		pathParts := qualifiedParts[:len(qualifiedParts)-1]
		identifier = ast.NewQualifiedIdentifierWithPath(identifierToken, varToken, language, pathParts, varName)
	} else {
		// Простая квалифицированная переменная
		identifier = ast.NewQualifiedIdentifier(identifierToken, varToken, language, varName)
	}

	// Восстанавливаем позицию потока для парсинга индексного выражения
	ctx.TokenStream.SetPosition(indexStart)

	// Потребляем '['
	ctx.TokenStream.Consume()

	// Используем UnifiedExpressionParser для парсинга индексного выражения
	exprParser := NewUnifiedExpressionParser(h.verbose)
	indexExpr, err := exprParser.ParseExpression(ctx)
	if err != nil {
		return nil, newErrorWithPos(ctx.TokenStream, "failed to parse index expression: %v", err)
	}

	// Проверяем и потребляем ']'
	if !ctx.TokenStream.HasMore() || ctx.TokenStream.Current().Type != lexer.TokenRBracket {
		return nil, newErrorWithPos(ctx.TokenStream, "expected ']' after index expression")
	}
	ctx.TokenStream.Consume()

	// Создаем первый IndexExpression для py.data.users[0]
	firstIndexExpr := &ast.IndexExpression{
		Object: identifier,
		Index:  indexExpr,
		Pos:    identifier.Position(),
	}

	// Создаем второй IndexExpression для доступа к свойству .age
	propertyIndex := &ast.StringLiteral{
		Value: propertyToken.Value,
		Pos: ast.Position{
			Line:   propertyToken.Line,
			Column: propertyToken.Column,
			Offset: propertyToken.Position,
		},
	}

	// Создаем вложенный IndexExpression
	finalIndexExpr := &ast.IndexExpression{
		Object: firstIndexExpr,
		Index:  propertyIndex,
		Pos: ast.Position{
			Line:   dotToken.Line,
			Column: dotToken.Column,
			Offset: dotToken.Position,
		},
	}

	if h.verbose {
		fmt.Printf("DEBUG: handlePropertyAccessAfterIndex - created nested IndexExpression for property access\n")
		fmt.Printf("DEBUG: handlePropertyAccessAfterIndex - current token after parsing value: %s (%s)\n", tokenStream.Current().Value, tokenStream.Current().Type)
	}

	// Создаем ExpressionAssignment
	assignment := &ast.ExpressionAssignment{
		Left:   finalIndexExpr,
		Assign: assignToken,
		Value:  value,
	}

	if h.verbose {
		fmt.Printf("DEBUG: handlePropertyAccessAfterIndex - created assignment: %+v\n", assignment)
	}

	// Убедимся, что токен поток находится в правильной позиции после парсинга всего выражения
	// Пропускаем любые оставшиеся токены до конца строки или EOF
	for tokenStream.HasMore() && tokenStream.Current().Type != lexer.TokenNewline && tokenStream.Current().Type != lexer.TokenEOF {
		if h.verbose {
			fmt.Printf("DEBUG: handlePropertyAccessAfterIndex - skipping remaining token: %s (%s)\n", tokenStream.Current().Value, tokenStream.Current().Type)
		}
		tokenStream.Consume()
	}

	return assignment, nil
}

// collectExpressionTokens собирает токены выражения до разделителей
func (h *AssignmentHandler) collectExpressionTokens(tokenStream stream.TokenStream) ([]lexer.Token, error) {
	var tokens []lexer.Token
	depth := 0

	for tokenStream.HasMore() {
		token := tokenStream.Current()

		// Проверяем на разделители выражений (на уровне 0)
		if depth == 0 {
			if token.Type == lexer.TokenComma ||
				token.Type == lexer.TokenSemicolon ||
				token.Type == lexer.TokenNewline ||
				token.Type == lexer.TokenEOF ||
				token.Type == lexer.TokenIf ||
				token.Type == lexer.TokenWhile ||
				token.Type == lexer.TokenFor ||
				token.Type == lexer.TokenMatch {
				break
			}
		}

		// Отслеживаем вложенность скобок
		if token.Type == lexer.TokenLeftParen {
			depth++
		} else if token.Type == lexer.TokenRightParen {
			depth--
			if depth < 0 {
				return nil, newErrorWithTokenPos(token, "unexpected closing bracket")
			}
		}

		tokens = append(tokens, token)
		tokenStream.Consume()
	}

	return tokens, nil
}

// expressionToTokens конвертирует выражение обратно в токены (приблизительно)
func (h *AssignmentHandler) expressionToTokens(expr ast.Expression) ([]lexer.Token, error) {
	// Это упрощенная реализация - конвертируем строковое представление обратно в токены
	// В реальности нужна более сложная логика
	exprStr := ""
	if strExpr, ok := expr.(interface{ String() string }); ok {
		exprStr = strExpr.String()
	} else {
		exprStr = fmt.Sprintf("%v", expr)
	}

	// Для простых случаев создаем токены на основе строки
	var tokens []lexer.Token
	pos := ast.Position{Line: 1, Column: 1, Offset: 0}

	// Простой парсер - разбиваем по пробелам и операторам
	// Это не идеально, но работает для базовых случаев
	i := 0
	for i < len(exprStr) {
		// Пропускаем пробелы
		for i < len(exprStr) && (exprStr[i] == ' ' || exprStr[i] == '\t') {
			i++
			pos.Column++
		}
		if i >= len(exprStr) {
			break
		}

		start := i
		if exprStr[i] == '"' || exprStr[i] == '\'' {
			// Строка
			quote := exprStr[i]
			i++
			for i < len(exprStr) && exprStr[i] != quote {
				i++
			}
			if i < len(exprStr) {
				i++ // включаем закрывающую кавычку
			}
			tokens = append(tokens, lexer.Token{
				Type:     lexer.TokenString,
				Value:    exprStr[start:i],
				Line:     pos.Line,
				Column:   pos.Column,
				Position: pos.Offset,
			})
		} else if exprStr[i] >= '0' && exprStr[i] <= '9' {
			// Число
			for i < len(exprStr) && ((exprStr[i] >= '0' && exprStr[i] <= '9') || exprStr[i] == '.') {
				i++
			}
			tokens = append(tokens, lexer.Token{
				Type:     lexer.TokenNumber,
				Value:    exprStr[start:i],
				Line:     pos.Line,
				Column:   pos.Column,
				Position: pos.Offset,
			})
		} else if (exprStr[i] >= 'a' && exprStr[i] <= 'z') || (exprStr[i] >= 'A' && exprStr[i] <= 'Z') || exprStr[i] == '_' {
			// Идентификатор
			for i < len(exprStr) && ((exprStr[i] >= 'a' && exprStr[i] <= 'z') || (exprStr[i] >= 'A' && exprStr[i] <= 'Z') || (exprStr[i] >= '0' && exprStr[i] <= '9') || exprStr[i] == '_' || exprStr[i] == '.') {
				i++
			}
			tokens = append(tokens, lexer.Token{
				Type:     lexer.TokenIdentifier,
				Value:    exprStr[start:i],
				Line:     pos.Line,
				Column:   pos.Column,
				Position: pos.Offset,
			})
		} else {
			// Оператор или символ
			tokenLen := 1
			if i+1 < len(exprStr) {
				twoChar := exprStr[i : i+2]
				if twoChar == "==" || twoChar == "!=" || twoChar == "<=" || twoChar == ">=" || twoChar == "&&" || twoChar == "||" {
					tokenLen = 2
				}
			}
			op := exprStr[start : start+tokenLen]
			var tokenType lexer.TokenType
			switch op {
			case "==":
				tokenType = lexer.TokenEqual
			case "!=":
				tokenType = lexer.TokenNotEqual
			case "<":
				tokenType = lexer.TokenLess
			case ">":
				tokenType = lexer.TokenGreater
			case "<=":
				tokenType = lexer.TokenLessEqual
			case ">=":
				tokenType = lexer.TokenGreaterEqual
			case "&&":
				tokenType = lexer.TokenAnd
			case "||":
				tokenType = lexer.TokenOr
			case "+":
				tokenType = lexer.TokenPlus
			case "-":
				tokenType = lexer.TokenMinus
			case "*":
				tokenType = lexer.TokenMultiply
			case "/":
				tokenType = lexer.TokenSlash
			case "%":
				tokenType = lexer.TokenModulo
			default:
				tokenType = lexer.TokenIdentifier // fallback
			}
			tokens = append(tokens, lexer.Token{
				Type:     tokenType,
				Value:    op,
				Line:     pos.Line,
				Column:   pos.Column,
				Position: pos.Offset,
			})
			i += tokenLen
		}
		pos.Column += i - start
		pos.Offset += i - start
	}

	return tokens, nil
}
