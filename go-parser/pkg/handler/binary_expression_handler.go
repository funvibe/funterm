package handler

import (
	"fmt"
	"go-parser/pkg/ast"
	"go-parser/pkg/common"
	"go-parser/pkg/config"
	"go-parser/pkg/expression"
	"go-parser/pkg/lexer"
	"go-parser/pkg/stream"
)

// isUnaryOperator проверяет, является ли токен унарным оператором
func isUnaryOperator(tokenType lexer.TokenType) bool {
	switch tokenType {
	case lexer.TokenPlus, lexer.TokenMinus, lexer.TokenNot, lexer.TokenTilde, lexer.TokenAt:
		return true
	default:
		return false
	}
}

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
	// Обрабатываем идентификаторы, литералы, language calls и бинарные операторы
	// Например: "a = b + c", "nil == nil", "true == false", "1 + 2", "a + b", "cond ? true : false", "python.func()"
	return token.Type == lexer.TokenIdentifier ||
		token.Type == lexer.TokenNumber ||
		token.Type == lexer.TokenTrue ||
		token.Type == lexer.TokenFalse ||
		token.Type == lexer.TokenNil ||
		token.Type == lexer.TokenString ||
		token.IsLanguageToken() || // Для language calls: python.func(), lua.x, etc.
		token.Type == lexer.TokenPlus || // Для бинарных выражений (unary +)
		token.Type == lexer.TokenMinus || // Для бинарных выражений (unary -)
		token.Type == lexer.TokenMultiply ||
		token.Type == lexer.TokenSlash ||
		token.Type == lexer.TokenModulo ||
		token.Type == lexer.TokenPower ||
		token.Type == lexer.TokenEqual ||
		token.Type == lexer.TokenNotEqual ||
		token.Type == lexer.TokenLess ||
		token.Type == lexer.TokenLessEqual ||
		token.Type == lexer.TokenGreater ||
		token.Type == lexer.TokenGreaterEqual ||
		token.Type == lexer.TokenAnd ||
		token.Type == lexer.TokenOr ||
		token.Type == lexer.TokenAmpersand ||
		token.Type == lexer.TokenCaret ||
		token.Type == lexer.TokenDoubleLeftAngle ||
		token.Type == lexer.TokenDoubleRightAngle ||
		token.Type == lexer.TokenConcat ||
		token.Type == lexer.TokenQuestion // Для тернарных и Elvis операторов
}

// Handle обрабатывает бинарное выражение
func (h *BinaryExpressionHandler) Handle(ctx *common.ParseContext) (interface{}, error) {
	tokenStream := ctx.TokenStream

	if h.verbose {
		fmt.Printf("DEBUG: BinaryExpressionHandler.Handle called with token: %s (%s)\n", tokenStream.Current().Value, tokenStream.Current().Type)
	}

	// Проверяем, есть ли токен присваивания
	if !tokenStream.HasMore() {
		return nil, newErrorWithPos(tokenStream, "unexpected EOF in binary expression")
	}

	// Проверяем, является ли это просто бинарным выражением (не присваиванием)
	// Если следующий токен - бинарный оператор, парсим как бинарное выражение
	if tokenStream.HasMore() && isBinaryOperator(tokenStream.Peek().Type) &&
		tokenStream.Peek().Type != lexer.TokenAssign && tokenStream.Peek().Type != lexer.TokenColonEquals {
		// Это бинарное выражение, парсим его полностью
		expr, err := h.ParseFullExpression(ctx, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to parse binary expression: %v", err)
		}
		return expr, nil
	}

	// Проверяем, является ли это language call (python.func(), lua.x, etc.)
	if tokenStream.Current().IsLanguageToken() {
		// Это может быть language call или qualified variable другого языка
		// Клонируем поток для проверки
		tempStream := tokenStream.Clone()
		tempCtx := &common.ParseContext{
			TokenStream: tempStream,
			Parser:      ctx.Parser,
			Depth:       ctx.Depth,
			MaxDepth:    ctx.MaxDepth,
			Guard:       ctx.Guard,
			LoopDepth:   ctx.LoopDepth,
			InputStream: ctx.InputStream,
		}
		_, err := h.parseLanguageCallOrField(tempCtx)
		if err != nil {
			// Если это не валидный language call, позволяем другим handlers попробовать
			return nil, nil
		}
		// Если валидный, парсим по-настоящему
		leftExpr, err := h.parseLanguageCallOrField(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to parse left side language call: %v", err)
		}

		// Проверяем наличие оператора присваивания
		if !tokenStream.HasMore() || (tokenStream.Current().Type != lexer.TokenAssign && tokenStream.Current().Type != lexer.TokenColonEquals) {
			return nil, newErrorWithPos(tokenStream, "expected '=' or ':=' after language call in binary expression")
		}
		tokenStream.Consume() // потребляем '='

		// Теперь парсим правую часть как бинарное выражение
		rightExpr, err := h.ParseFullExpression(ctx, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to parse right side of binary expression: %v", err)
		}

		// Создаем VariableAssignment с language call
		return &ast.VariableAssignment{
			Variable: leftExpr.(*ast.VariableRead).Variable,
			Value:    rightExpr,
		}, nil
	}

	// Проверяем, является ли это квалифицированной переменной (lua.x)
	if tokenStream.Current().Type == lexer.TokenIdentifier &&
		tokenStream.HasMore() && tokenStream.Peek().Type == lexer.TokenDot {
		// Это квалифицированная переменная
		// Клонируем поток для проверки
		tempStream := tokenStream.Clone()
		tempCtx := &common.ParseContext{
			TokenStream: tempStream,
			Parser:      ctx.Parser,
			Depth:       ctx.Depth,
			MaxDepth:    ctx.MaxDepth,
			Guard:       ctx.Guard,
			LoopDepth:   ctx.LoopDepth,
			InputStream: ctx.InputStream,
		}
		_, err := h.parseQualifiedVariable(tempCtx)
		if err != nil {
			// Если это не валидная qualified variable, позволяем другим handlers попробовать
			return nil, nil
		}
		// Если валидная, парсим по-настоящему
		leftExpr, err := h.parseQualifiedVariable(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to parse left side qualified variable: %v", err)
		}

		// Проверяем наличие оператора присваивания
		if !tokenStream.HasMore() || (tokenStream.Current().Type != lexer.TokenAssign && tokenStream.Current().Type != lexer.TokenColonEquals) {
			return nil, newErrorWithPos(tokenStream, "expected '=' or ':=' after qualified variable in binary expression")
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

	// Потребляем левый операнд (идентификатор или литерал)
	leftToken := tokenStream.Current()
	isLiteral := leftToken.Type == lexer.TokenNumber || leftToken.Type == lexer.TokenTrue ||
		leftToken.Type == lexer.TokenFalse || leftToken.Type == lexer.TokenNil ||
		leftToken.Type == lexer.TokenString

	if leftToken.Type != lexer.TokenIdentifier && !isLiteral {
		return nil, newErrorWithTokenPos(leftToken, "expected identifier or literal for binary expression, got %s", leftToken.Type)
	}
	tokenStream.Consume()

	// Проверяем наличие оператора присваивания или бинарного оператора
	if !tokenStream.HasMore() {
		return nil, newErrorWithPos(tokenStream, "expected operator after operand")
	}

	operatorToken := tokenStream.Current()

	// Проверяем, является ли это тернарным выражением
	if operatorToken.Type == lexer.TokenQuestion {
		// Создаем leftExpr для condition
		var leftExpr ast.Expression
		if leftToken.Type == lexer.TokenIdentifier {
			leftExpr = &ast.Identifier{
				Name: leftToken.Value,
				Pos: ast.Position{
					Line:   leftToken.Line,
					Column: leftToken.Column,
					Offset: leftToken.Position,
				},
			}
		} else {
			// Создаем соответствующий литерал
			switch leftToken.Type {
			case lexer.TokenNumber:
				numValue, err := parseNumber(leftToken.Value)
				if err != nil {
					return nil, fmt.Errorf("invalid number format: %s", leftToken.Value)
				}
				leftExpr = createNumberLiteral(leftToken, numValue)
			case lexer.TokenTrue, lexer.TokenFalse:
				leftExpr = &ast.BooleanLiteral{
					Value: leftToken.Type == lexer.TokenTrue,
					Pos: ast.Position{
						Line:   leftToken.Line,
						Column: leftToken.Column,
						Offset: leftToken.Position,
					},
				}
			case lexer.TokenNil:
				leftExpr = ast.NewNilLiteral(leftToken)
			case lexer.TokenString:
				leftExpr = &ast.StringLiteral{
					Value: leftToken.Value,
					Pos: ast.Position{
						Line:   leftToken.Line,
						Column: leftToken.Column,
						Offset: leftToken.Position,
					},
				}
			}
		}

		// Парсим тернарное выражение
		expr, err := h.ParseTernaryExpression(ctx, leftExpr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse ternary expression: %v", err)
		}
		return expr, nil
	}

	if operatorToken.Type == lexer.TokenAssign || operatorToken.Type == lexer.TokenColonEquals {
		// Это присваивание
		tokenStream.Consume() // потребляем '='

		// Теперь парсим правую часть как бинарное выражение
		rightExpr, err := h.ParseFullExpression(ctx, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to parse right side of binary expression: %v", err)
		}

		// Создаем VariableAssignment
		if leftToken.Type == lexer.TokenIdentifier {
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
		} else {
			// Присваивание литералу не имеет смысла
			return nil, newErrorWithTokenPos(leftToken, "cannot assign to literal")
		}
	} else if isBinaryOperator(operatorToken.Type) {
		// Это бинарное выражение, создаем левый операнд и парсим все выражение
		var leftExpr ast.Expression
		if leftToken.Type == lexer.TokenIdentifier {
			leftExpr = &ast.Identifier{
				Name: leftToken.Value,
				Pos: ast.Position{
					Line:   leftToken.Line,
					Column: leftToken.Column,
					Offset: leftToken.Position,
				},
			}
		} else {
			// Создаем соответствующий литерал
			switch leftToken.Type {
			case lexer.TokenNumber:
				numValue, err := parseNumber(leftToken.Value)
				if err != nil {
					return nil, fmt.Errorf("invalid number format: %s", leftToken.Value)
				}
				leftExpr = createNumberLiteral(leftToken, numValue)
			case lexer.TokenTrue, lexer.TokenFalse:
				leftExpr = &ast.BooleanLiteral{
					Value: leftToken.Type == lexer.TokenTrue,
					Pos: ast.Position{
						Line:   leftToken.Line,
						Column: leftToken.Column,
						Offset: leftToken.Position,
					},
				}
			case lexer.TokenNil:
				leftExpr = ast.NewNilLiteral(leftToken)
			case lexer.TokenString:
				leftExpr = &ast.StringLiteral{
					Value: leftToken.Value,
					Pos: ast.Position{
						Line:   leftToken.Line,
						Column: leftToken.Column,
						Offset: leftToken.Position,
					},
				}
			}
		}

		// Парсим бинарное выражение начиная с левого операнда
		expr, err := h.ParseFullExpression(ctx, leftExpr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse binary expression: %v", err)
		}
		return expr, nil
	} else {
		return nil, newErrorWithTokenPos(operatorToken, "expected assignment or binary operator, got %s", operatorToken.Type)
	}
}

// ParseFullExpression парсит полное бинарное выражение начиная с левого операнда
func (h *BinaryExpressionHandler) ParseFullExpression(ctx *common.ParseContext, left ast.Expression) (ast.Expression, error) {
	if left != nil {
		// Левая часть уже распарсена, используем precedence climbing
		return h.parseExpressionWithPrecedence(ctx, left, -1)
	}

	// Собираем токены выражения
	tokens, err := h.collectExpressionTokens(ctx.TokenStream)
	if err != nil {
		return nil, err
	}

	// Используем централизованный shunting-yard парсер
	return expression.ParseExpression(tokens)
}

// parseExpressionWithPrecedence парсит выражение с учетом приоритетов операторов
func (h *BinaryExpressionHandler) parseExpressionWithPrecedence(ctx *common.ParseContext, left ast.Expression, minPrecedence int) (ast.Expression, error) {
	tokenStream := ctx.TokenStream

	if h.verbose {
		fmt.Printf("DEBUG: parseExpressionWithPrecedence called with left: %v, minPrecedence: %d, current token: %s (%s)\n", left != nil, minPrecedence, tokenStream.Current().Value, tokenStream.Current().Type)
	}

	for tokenStream.HasMore() {
		operatorToken := tokenStream.Current()

		if h.verbose {
			fmt.Printf("DEBUG: parseExpressionWithPrecedence - checking token: %s (%s), precedence: %d, minPrecedence: %d\n", operatorToken.Value, operatorToken.Type, getOperatorPrecedence(operatorToken.Type), minPrecedence)
		}

		// Проверяем бинарный оператор
		if !isBinaryOperator(operatorToken.Type) {
			if h.verbose {
				fmt.Printf("DEBUG: parseExpressionWithPrecedence - token %s is not binary operator, breaking\n", operatorToken.Value)
			}
			break // Нет бинарного оператора, выходим из цикла
		}

		precedence := getOperatorPrecedence(operatorToken.Type)

		// Для тернарного оператора используем право-ассоциативность (проверяем >= вместо <)
		if operatorToken.Type == lexer.TokenQuestion {
			if precedence < minPrecedence {
				if h.verbose {
					fmt.Printf("DEBUG: parseExpressionWithPrecedence - ternary precedence %d < minPrecedence %d, breaking\n", precedence, minPrecedence)
				}
				break
			}
			return h.ParseTernaryExpression(ctx, left)
		}

		if precedence < minPrecedence {
			if h.verbose {
				fmt.Printf("DEBUG: parseExpressionWithPrecedence - precedence %d < minPrecedence %d, breaking\n", precedence, minPrecedence)
			}
			break // Приоритет слишком низкий, выходим из цикла
		}

		// Потребляем оператор
		tokenStream.Consume()

		if h.verbose {
			fmt.Printf("DEBUG: parseExpressionWithPrecedence - consumed operator %s, current token: %s (%s)\n", operatorToken.Value, tokenStream.Current().Value, tokenStream.Current().Type)
		}

		var rightExpr ast.Expression
		var err error

		// Special handling for pipe operator (|>)
		if operatorToken.Type == lexer.TokenPipe {
			// For pipe operator, the right side should be a language call
			if !tokenStream.HasMore() {
				return nil, newErrorWithPos(tokenStream, "unexpected EOF after pipe operator")
			}

			rightToken := tokenStream.Current()
			if h.verbose {
				fmt.Printf("DEBUG: parseExpressionWithPrecedence - pipe operator, right token: %s (%s)\n", rightToken.Value, rightToken.Type)
			}

			if rightToken.IsLanguageToken() {
				// Parse qualified language call (e.g., lua.string.lower())
				rightExpr, err = h.parseQualifiedVariable(ctx)
				if err != nil {
					return nil, fmt.Errorf("failed to parse right side of pipe expression: %v", err)
				}
				if h.verbose {
					fmt.Printf("DEBUG: parseExpressionWithPrecedence - successfully parsed qualified variable: %T\n", rightExpr)
				}
			} else {
				// Parse regular operand
				rightExpr, err = h.parseOperand(ctx)
				if err != nil {
					return nil, fmt.Errorf("failed to parse right operand: %v", err)
				}
			}
		} else {
			// Читаем правый операнд для обычных бинарных операторов
			rightExpr, err = h.parseOperand(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to parse right operand: %v", err)
			}
		}

		// Проверяем следующий оператор для определения ассоциативности
		// Если следующий оператор имеет более высокий приоритет, парсим его рекурсивно
		if tokenStream.HasMore() && isBinaryOperator(tokenStream.Current().Type) {
			nextToken := tokenStream.Current()
			nextPrecedence := getOperatorPrecedence(nextToken.Type)

			if h.verbose {
				fmt.Printf("DEBUG: parseExpressionWithPrecedence - checking next operator: %s (%s), nextPrecedence: %d, current precedence: %d\n", nextToken.Value, nextToken.Type, nextPrecedence, precedence)
			}

			// Для левой ассоциативности: если следующий оператор имеет более высокий приоритет,
			// парсим правую часть рекурсивно с более высоким приоритетом
			if nextPrecedence > precedence {
				if h.verbose {
					fmt.Printf("DEBUG: parseExpressionWithPrecedence - nextPrecedence %d > precedence %d, parsing right side recursively\n", nextPrecedence, precedence)
				}
				rightExpr, err = h.parseExpressionWithPrecedence(ctx, rightExpr, nextPrecedence)
				if err != nil {
					return nil, err
				}
			} else if nextPrecedence == precedence && operatorToken.Type == lexer.TokenPower {
				// Для правой ассоциативности оператора **: если следующий оператор имеет тот же приоритет,
				// парсим правую часть рекурсивно с тем же приоритетом
				if h.verbose {
					fmt.Printf("DEBUG: parseExpressionWithPrecedence - right associative ** operator, parsing right side recursively\n")
				}
				rightExpr, err = h.parseExpressionWithPrecedence(ctx, rightExpr, precedence)
				if err != nil {
					return nil, err
				}
			}
		}

		// Создаем бинарное выражение
		var pos ast.Position
		if left != nil {
			pos = left.Position()
		} else {
			pos = ast.Position{
				Line:   operatorToken.Line,
				Column: operatorToken.Column,
				Offset: operatorToken.Position,
			}
		}
		left = ast.NewBinaryExpression(left, operatorToken.Value, rightExpr, pos)
		if h.verbose {
			fmt.Printf("DEBUG: parseExpressionWithPrecedence - created BinaryExpression: %T\n", left)
		}
	}

	// Проверяем, не является ли результат частью тернарного выражения
	if tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenQuestion {
		if h.verbose {
			fmt.Printf("DEBUG: parseExpressionWithPrecedence - found ternary operator, calling ParseTernaryExpression\n")
		}
		return h.ParseTernaryExpression(ctx, left)
	}

	if h.verbose {
		fmt.Printf("DEBUG: parseExpressionWithPrecedence - returning left: %T\n", left)
	}
	return left, nil
}

// ParseTernaryExpression парсит тернарный оператор (condition ? true_expr : false_expr)
// или Elvis оператор (condition ?: fallback_expr)
func (h *BinaryExpressionHandler) ParseTernaryExpression(ctx *common.ParseContext, condition ast.Expression) (ast.Expression, error) {
	tokenStream := ctx.TokenStream

	if h.verbose {
		fmt.Printf("DEBUG: ParseTernaryExpression called with condition type: %T\n", condition)
	}

	// Потребляем токен ?
	questionToken := tokenStream.Consume()

	// Проверяем, не является ли это Elvis оператором (condition ?: fallback_expr)
	if tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenColon {
		// Это Elvis оператор: condition ?: fallback_expr
		colonToken := tokenStream.Consume()

		// Парсим fallback выражение (это может быть другой Elvis оператор!)
		// Используем parseOperand для получения левого операнда, затем parseExpressionWithPrecedence
		// для обработки всех операторов, включая потенциальные Elvis операторы
		fallbackExpr, err := h.parseOperand(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to parse fallback operand in Elvis: %v", err)
		}

		// Парсим все следующие операторы (включая потенциальные Elvis операторы!)
		// Используем приоритет 0, чтобы обработать все операторы, включая Elvis
		fallbackExpr, err = h.parseExpressionWithPrecedence(ctx, fallbackExpr, 0)
		if err != nil {
			return nil, fmt.Errorf("failed to parse fallback expression in Elvis: %v", err)
		}

		// Создаем Elvis выражение (как особый случай тернарного)
		result := ast.NewTernaryExpression(condition, questionToken, colonToken, condition, fallbackExpr, condition.Position())
		if h.verbose {
			fmt.Printf("DEBUG: ParseTernaryExpression (Elvis) returning, current token: %s (%s)\n", tokenStream.Current().Value, tokenStream.Current().Type)
		}
		return result, nil
	}

	// Парсим true выражение с precedence выше тернарного оператора
	trueExpr, err := h.parseOperand(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to parse true expression in ternary: %v", err)
	}

	// Парсим бинарные операторы в true выражении до :
	trueExpr, err = h.parseExpressionWithPrecedence(ctx, trueExpr, 4)
	if err != nil {
		return nil, fmt.Errorf("failed to parse true expression in ternary: %v", err)
	}

	// Проверяем наличие токена :
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenColon {
		return nil, newErrorWithPos(tokenStream, "expected ':' after true expression in ternary")
	}

	// Потребляем токен :
	colonToken := tokenStream.Consume()

	// Парсим false выражение с precedence выше тернарного оператора
	falseExpr, err := h.parseOperand(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to parse false expression in ternary: %v", err)
	}

	// Парсим бинарные операторы в false выражении
	falseExpr, err = h.parseExpressionWithPrecedence(ctx, falseExpr, 4)
	if err != nil {
		return nil, fmt.Errorf("failed to parse false expression in ternary: %v", err)
	}

	// Создаем тернарное выражение
	result := ast.NewTernaryExpression(condition, questionToken, colonToken, trueExpr, falseExpr, condition.Position())
	if h.verbose {
		fmt.Printf("DEBUG: ParseTernaryExpression returning, current token: %s (%s)\n", tokenStream.Current().Value, tokenStream.Current().Type)
	}
	return result, nil
}

// parseOperand парсит операнд бинарного выражения
func (h *BinaryExpressionHandler) parseOperand(ctx *common.ParseContext) (ast.Expression, error) {
	// Сначала попробуем распарсить базовый операнд
	expr, err := h.parseBasicOperand(ctx)
	if err != nil {
		return nil, err
	}

	return expr, nil
}

// parseBasicOperand парсит базовый операнд без тернарных выражений
func (h *BinaryExpressionHandler) parseBasicOperand(ctx *common.ParseContext) (ast.Expression, error) {
	tokenStream := ctx.TokenStream

	if !tokenStream.HasMore() {
		return nil, newErrorWithPos(tokenStream, "unexpected EOF in operand")
	}

	token := tokenStream.Current()

	// Проверяем унарные операторы
	if isUnaryOperator(token.Type) {
		unaryHandler := NewUnaryExpressionHandler(config.ConstructHandlerConfig{ConstructType: common.ConstructUnaryExpression})
		result, err := unaryHandler.Handle(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to parse unary expression: %v", err)
		}
		if expr, ok := result.(*ast.UnaryExpression); ok {
			return expr, nil
		}
		return nil, fmt.Errorf("expected UnaryExpression, got %T", result)
	}

	// Проверяем language tokens (python, lua, js, etc.)
	if token.IsLanguageToken() {
		// Проверяем, не является ли это qualified variable (js.x)
		if tokenStream.HasMore() && tokenStream.Peek().Type == lexer.TokenDot {
			// Это qualified variable
			expr, err := h.parseQualifiedVariable(ctx)
			if err != nil {
				return nil, err
			}

			// Проверяем наличие индексного выражения [index] после qualified variable
			if tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenLBracket {
				return h.ParseIndexExpression(ctx, expr)
			}

			return expr, nil
		} else {
			// Это language call
			expr, err := h.parseLanguageCallOrField(ctx)
			if err != nil {
				return nil, err
			}

			// Проверяем наличие индексного выражения [index] после language call/field
			if tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenLBracket {
				return h.ParseIndexExpression(ctx, expr)
			}

			return expr, nil
		}
	}

	switch token.Type {
	case lexer.TokenIdentifier:
		// Проверяем, не является ли это вызовом builtin функции
		if tokenStream.Peek().Type == lexer.TokenLeftParen {
			// Это вызов builtin функции типа len(x)
			builtinHandler := NewBuiltinFunctionHandlerWithVerbose(config.ConstructHandlerConfig{}, h.verbose)
			result, err := builtinHandler.Handle(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to parse builtin function call: %v", err)
			}

			builtinCall, ok := result.(*ast.BuiltinFunctionCall)
			if !ok {
				return nil, fmt.Errorf("expected BuiltinFunctionCall, got %T", result)
			}

			return builtinCall, nil
		}

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
					return h.ParseIndexExpression(ctx, expr)
				}

				return expr, nil
			}
		} else {
			// Простой идентификатор
			tokenStream.Consume()
			ident := ast.NewIdentifier(token, token.Value)

			// Проверяем наличие индексного выражения [index] после простого идентификатора
			if tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenLBracket {
				return h.ParseIndexExpression(ctx, ident)
			}

			return ident, nil
		}
	case lexer.TokenDoubleLeftAngle:
		// Check if this is a bitstring pattern or shift operator
		if h.isBitstringPattern(tokenStream) {
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

			// Convert to expression for use in conditions
			bitstringPatternMatchExpression := ast.NewBitstringPatternMatchExpression(
				bitstringPatternAssignment.Pattern,
				bitstringPatternAssignment.Assign,
				bitstringPatternAssignment.Value,
			)

			return bitstringPatternMatchExpression, nil
		} else {
			// This is a shift operator, let the binary expression parser handle it
			// Return error to let the caller handle it as binary operator
			return nil, fmt.Errorf("shift operator should be handled by binary expression parser")
		}

	default:
		// Проверяем, является ли токен языковым токеном
		if token.IsLanguageToken() {
			// Это может быть квалифицированная переменная или вызов функции
			if tokenStream.Peek().Type == lexer.TokenDot {
				// Используем parseQualifiedVariable для обработки квалифицированных переменных и вызовов функций
				expr, err := h.parseQualifiedVariable(ctx)
				if err != nil {
					return nil, err
				}

				// Проверяем наличие индексного выражения [index] после qualified variable
				if tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenLBracket {
					return h.ParseIndexExpression(ctx, expr)
				}

				return expr, nil
			} else {
				return nil, newErrorWithTokenPos(token, "expected '.' after language token '%s'", token.Type)
			}
		}
		return nil, newErrorWithTokenPos(token, "unsupported operand type: %s", token.Type)

	case lexer.TokenMinus:
		// Унарный минус
		tokenStream.Consume() // потребляем '-'

		// Рекурсивно парсим операнд после минуса
		operand, err := h.parseOperand(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to parse operand after unary minus: %v", err)
		}

		// Создаем унарное выражение
		return ast.NewUnaryExpression("-", operand, ast.Position{
			Line:   token.Line,
			Column: token.Column,
			Offset: token.Position,
		}), nil

	case lexer.TokenNot:
		// Унарный логический NOT
		tokenStream.Consume() // потребляем '!'

		// Рекурсивно парсим операнд после !
		operand, err := h.parseOperand(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to parse operand after unary not: %v", err)
		}

		// Создаем унарное выражение
		return ast.NewUnaryExpression("!", operand, ast.Position{
			Line:   token.Line,
			Column: token.Column,
			Offset: token.Position,
		}), nil

	case lexer.TokenNumber:
		// Числовой литерал
		tokenStream.Consume()
		numValue, err := parseNumber(token.Value)
		if err != nil {
			return nil, fmt.Errorf("invalid number format: %s", token.Value)
		}
		return createNumberLiteral(token, numValue), nil

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

	case lexer.TokenNil:
		// Nil литерал
		tokenStream.Consume()
		return ast.NewNilLiteral(token), nil

	case lexer.TokenLBracket:
		// Array literal - используем ArrayHandler
		arrayHandler := NewArrayHandler(200, 10)
		result, err := arrayHandler.Handle(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to parse array literal: %v", err)
		}

		arrayLiteral, ok := result.(*ast.ArrayLiteral)
		if !ok {
			return nil, fmt.Errorf("expected ArrayLiteral, got %T", result)
		}

		// Парсим все индексные выражения [index][index]... в цикле
		currentExpr := ast.Expression(arrayLiteral)
		for tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenLBracket {
			indexExpr, err := h.ParseIndexExpression(ctx, currentExpr)
			if err != nil {
				return nil, err
			}
			currentExpr = indexExpr
		}

		return currentExpr, nil

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
			return nil, newErrorWithPos(tokenStream, "expected ')' after expression")
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
func (h *BinaryExpressionHandler) parseLanguageCallOrField(ctx *common.ParseContext) (ast.Expression, error) {
	// Создаем LanguageCallHandler для парсинга language calls
	languageHandler := NewLanguageCallHandlerWithVerbose(h.config, h.verbose)
	result, err := languageHandler.Handle(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to parse language call: %v", err)
	}

	// LanguageCall реализует Expression
	if expr, ok := result.(ast.Expression); ok {
		return expr, nil
	}

	return nil, fmt.Errorf("language call handler returned non-expression type: %T", result)
}

func (h *BinaryExpressionHandler) parseQualifiedVariable(ctx *common.ParseContext) (ast.Expression, error) {
	tokenStream := ctx.TokenStream

	// Потребляем первый идентификатор (язык)
	firstToken := tokenStream.Consume()
	tempFirstToken := lexer.Token{Type: firstToken.Type}
	if firstToken.Type != lexer.TokenIdentifier && !tempFirstToken.IsLanguageToken() {
		return nil, fmt.Errorf("expected identifier for qualified variable, got %s", firstToken.Type)
	}

	language := firstToken.Value

	// Проверяем, что язык поддерживается
	languageRegistry := CreateDefaultLanguageRegistry()
	_, err := languageRegistry.ResolveAlias(language)
	if err != nil {
		return nil, fmt.Errorf("not a qualified variable")
	}

	var path []string
	var lastName string

	if h.verbose {
		fmt.Printf("DEBUG: parseQualifiedVariable - starting with language: %s\n", language)
	}

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

		if h.verbose {
			fmt.Printf("DEBUG: parseQualifiedVariable - processing idToken: %s, current token: %s (%s)\n", idToken.Value, tokenStream.Current().Value, tokenStream.Current().Type)
		}

		// Проверяем, не является ли это вызовом функции
		if tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenLeftParen {
			// Это вызов функции вида python.check_status() или lua.string.lower()
			// Собираем все части имени функции через точки
			// Включаем предыдущие части из path + текущую часть
			functionParts := append(append([]string{}, path...), idToken.Value)

			if h.verbose {
				fmt.Printf("DEBUG: parseQualifiedVariable - found function call, path: %v, current part: %s, functionParts: %v\n", path, idToken.Value, functionParts)
			}

			// Проверяем, есть ли еще точки перед открывающей скобкой
			// Но сначала нужно проверить, есть ли точка после текущей скобки (для случаев вроде lua.string.lower())
			for tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenDot {
				// Потребляем точку
				tokenStream.Consume()

				// Проверяем, что после точки идет идентификатор
				if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenIdentifier {
					return nil, fmt.Errorf("expected identifier after dot in function name")
				}

				// Потребляем идентификатор
				funcToken := tokenStream.Consume()
				functionParts = append(functionParts, funcToken.Value)

				if h.verbose {
					fmt.Printf("DEBUG: parseQualifiedVariable - added function part: %s, parts now: %v\n", funcToken.Value, functionParts)
				}
			}

			// Теперь проверяем открывающую скобку
			if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenLeftParen {
				return nil, fmt.Errorf("expected '(' after function name")
			}

			// Потребляем открывающую скобку
			tokenStream.Consume()

			// Собираем полное имя функции
			functionName := ""
			for i, part := range functionParts {
				if i > 0 {
					functionName += "."
				}
				functionName += part
			}

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
		// Но мы должны проверить это перед тем, как войти в следующий цикл
		// В контексте вызова функции, все части имени функции должны быть собраны
		if tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenDot {
			// Это еще не конец, продолжаем цикл
			path = append(path, idToken.Value)
			if h.verbose {
				fmt.Printf("DEBUG: parseQualifiedVariable - added to path: %s, path now: %v\n", idToken.Value, path)
			}
		} else {
			// Это последнее имя
			lastName = idToken.Value
			if h.verbose {
				fmt.Printf("DEBUG: parseQualifiedVariable - set lastName: %s\n", lastName)
			}
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
func (h *BinaryExpressionHandler) ParseIndexExpression(ctx *common.ParseContext, object ast.Expression) (ast.Expression, error) {
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

	// Парсим индексное выражение
	// Используем parseOperand чтобы получить первый операнд
	indexExpr, err := h.parseOperand(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to parse index expression: %v", err)
	}

	// Потом парсим полное выражение, включая бинарные операторы
	// Это позволяет обработать выражения типа (1 + 2) * 0
	indexExpr, err = h.ParseFullExpression(ctx, indexExpr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse index expression: %v", err)
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

// isBitstringPattern проверяет, является ли << началом битовой строки или оператором сдвига
func (h *BinaryExpressionHandler) isBitstringPattern(tokenStream stream.TokenStream) bool {
	// Сохраняем текущую позицию
	currentPos := tokenStream.Position()
	defer tokenStream.SetPosition(currentPos)

	// Потребляем <<
	tokenStream.Consume()

	if !tokenStream.HasMore() {
		return false
	}

	// Проверяем следующий токен
	nextToken := tokenStream.Current()

	// Если следующий токен - >>, это пустая битовая строка: <<>>
	if nextToken.Type == lexer.TokenDoubleRightAngle {
		return true
	}

	// Функция для проверки валидного начала битовой строки
	isValidBitstringStart := func(token lexer.Token) bool {
		return token.Type == lexer.TokenIdentifier ||
			token.Type == lexer.TokenNumber ||
			token.Type == lexer.TokenString ||
			token.IsLanguageToken()
	}

	// Если следующий токен не является валидным началом битовой строки, это оператор сдвига
	if !isValidBitstringStart(nextToken) {
		return false
	}

	// Проверяем различные паттерны после валидного начала
	// 1. <<identifier>> или <<number>> или <<"string">> или <<language.identifier>>
	if tokenStream.HasMore() && tokenStream.Peek().Type == lexer.TokenDoubleRightAngle {
		return true
	}

	// 2. <<identifier/qualifier>> или <<identifier:size>> или <<identifier:size/qualifier>>
	peekToken := tokenStream.Peek()
	if peekToken.Type == lexer.TokenSlash {
		// Это квалификатор типа: /utf8, /binary, etc.
		return true
	} else if peekToken.Type == lexer.TokenColon {
		// Это размер: :8, :16, etc.
		return true
	}

	return false
}

// parseExpressionWithShuntingYard парсит выражение используя алгоритм shunting-yard
func (h *BinaryExpressionHandler) hasTernaryOperator(tokenStream stream.TokenStream) bool {
	// Сохраняем позицию
	pos := tokenStream.Position()
	defer tokenStream.SetPosition(pos)

	depth := 0
	for tokenStream.HasMore() {
		token := tokenStream.Current()

		// Проверяем на разделители выражений
		if depth == 0 && h.isExpressionTerminator(token.Type) {
			break
		}

		// Отслеживаем вложенность скобок
		if token.Type == lexer.TokenLeftParen || token.Type == lexer.TokenLBrace || token.Type == lexer.TokenLBracket {
			depth++
		} else if token.Type == lexer.TokenRightParen || token.Type == lexer.TokenRBrace || token.Type == lexer.TokenRBracket {
			depth--
			if depth < 0 {
				break
			}
		}

		// Проверяем на тернарный оператор
		if token.Type == lexer.TokenQuestion {
			return true
		}

		tokenStream.Consume()
	}

	return false
}

// hasTernaryOperatorInTokens проверяет, есть ли тернарный оператор в массиве токенов
func (h *BinaryExpressionHandler) hasTernaryOperatorInTokens(tokens []lexer.Token) bool {
	for _, token := range tokens {
		if token.Type == lexer.TokenQuestion {
			return true
		}
	}
	return false
}

// isExpressionTerminator проверяет, является ли токен разделителем выражения
func (h *BinaryExpressionHandler) isExpressionTerminator(tokenType lexer.TokenType) bool {
	return tokenType == lexer.TokenComma ||
		tokenType == lexer.TokenSemicolon ||
		tokenType == lexer.TokenRightParen ||
		tokenType == lexer.TokenRBrace ||
		tokenType == lexer.TokenRBracket ||
		tokenType == lexer.TokenColon ||
		tokenType == lexer.TokenDoubleRightAngle ||
		tokenType == lexer.TokenEOF ||
		tokenType == lexer.TokenNewline ||
		tokenType == lexer.TokenIf ||
		tokenType == lexer.TokenWhile ||
		tokenType == lexer.TokenFor ||
		tokenType == lexer.TokenMatch
}

// collectTernaryExpressionTokens собирает токены тернарного выражения, включая двоеточие
func (h *BinaryExpressionHandler) collectTernaryExpressionTokens(tokenStream stream.TokenStream) ([]lexer.Token, error) {
	var tokens []lexer.Token
	depth := 0
	ternaryDepth := 0 // Отслеживаем вложенность тернарных операторов

	for tokenStream.HasMore() {
		token := tokenStream.Current()

		// Отслеживаем вложенность скобок
		if token.Type == lexer.TokenLeftParen || token.Type == lexer.TokenLBrace || token.Type == lexer.TokenLBracket {
			depth++
		} else if token.Type == lexer.TokenRightParen || token.Type == lexer.TokenRBrace || token.Type == lexer.TokenRBracket {
			depth--
			if depth < 0 {
				return nil, newErrorWithTokenPos(token, "unexpected closing bracket in ternary expression")
			}
		}

		// Отслеживаем вложенность тернарных операторов
		if token.Type == lexer.TokenQuestion && depth == 0 {
			ternaryDepth++
		} else if token.Type == lexer.TokenColon && depth == 0 {
			ternaryDepth--
		}

		// Проверяем на разделители, которые заканчивают выражение (на уровне 0)
		if depth == 0 {
			// Разделители выражений: запятая, точка с запятой, скобки, ключевые слова и т.д.
			if token.Type == lexer.TokenComma ||
				token.Type == lexer.TokenSemicolon ||
				token.Type == lexer.TokenRightParen ||
				token.Type == lexer.TokenRBrace ||
				token.Type == lexer.TokenRBracket ||
				token.Type == lexer.TokenEOF ||
				(token.Type == lexer.TokenNewline && ternaryDepth <= 0) ||
				token.Type == lexer.TokenIf ||
				token.Type == lexer.TokenWhile ||
				token.Type == lexer.TokenFor ||
				token.Type == lexer.TokenMatch {
				break
			}
		}

		tokens = append(tokens, token)
		tokenStream.Consume()
	}

	if depth > 0 {
		return nil, newErrorWithPos(tokenStream, "unclosed brackets in ternary expression")
	}

	if len(tokens) == 0 {
		return nil, newErrorWithPos(tokenStream, "empty ternary expression")
	}

	return tokens, nil
}

// collectExpressionTokens собирает токены выражения до разделителей
func (h *BinaryExpressionHandler) collectExpressionTokens(tokenStream stream.TokenStream) ([]lexer.Token, error) {
	var tokens []lexer.Token
	depth := 0

	for tokenStream.HasMore() {
		token := tokenStream.Current()

		// Проверяем на разделители, которые заканчивают выражение (на уровне 0)
		if depth == 0 {
			// Разделители выражений: запятая, точка с запятой, скобки, ключевые слова и т.д.
			if token.Type == lexer.TokenComma ||
				token.Type == lexer.TokenSemicolon ||
				token.Type == lexer.TokenRightParen ||
				token.Type == lexer.TokenRBrace ||
				token.Type == lexer.TokenRBracket ||
				token.Type == lexer.TokenColon ||
				token.Type == lexer.TokenDoubleRightAngle ||
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
		if token.Type == lexer.TokenLeftParen || token.Type == lexer.TokenLBrace || token.Type == lexer.TokenLBracket {
			depth++
		} else if token.Type == lexer.TokenRightParen || token.Type == lexer.TokenRBrace || token.Type == lexer.TokenRBracket {
			depth--
			if depth < 0 {
				return nil, newErrorWithTokenPos(token, "unexpected closing bracket in expression")
			}
		}

		tokens = append(tokens, token)
		tokenStream.Consume()
	}

	if depth > 0 {
		return nil, newErrorWithPos(tokenStream, "unclosed brackets in expression")
	}

	if len(tokens) == 0 {
		return nil, newErrorWithPos(tokenStream, "empty expression")
	}

	return tokens, nil
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
