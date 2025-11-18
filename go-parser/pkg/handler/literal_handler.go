package handler

import (
	"fmt"
	"go-parser/pkg/ast"
	"go-parser/pkg/common"
	"go-parser/pkg/config"
	"go-parser/pkg/lexer"
	"math/big"
)

// LiteralHandler - обработчик литералов (чисел, строк)
type LiteralHandler struct {
	config  common.HandlerConfig
	verbose bool
}

// NewLiteralHandler - создает новый обработчик литералов
func NewLiteralHandler(config config.ConstructHandlerConfig) *LiteralHandler {
	return NewLiteralHandlerWithVerbose(config, false)
}

// NewLiteralHandlerWithVerbose - создает новый обработчик литералов с поддержкой verbose режима
func NewLiteralHandlerWithVerbose(config config.ConstructHandlerConfig, verbose bool) *LiteralHandler {
	handlerConfig := DefaultConfig("literal")
	handlerConfig.Priority = config.Priority
	handlerConfig.Order = config.Order
	return &LiteralHandler{
		config:  handlerConfig,
		verbose: verbose,
	}
}

// CanHandle - проверяет, может ли обработчик обработать токен
func (h *LiteralHandler) CanHandle(token lexer.Token) bool {
	return token.Type == lexer.TokenNumber || token.Type == lexer.TokenTrue || token.Type == lexer.TokenFalse || token.Type == lexer.TokenNil || token.Type == lexer.TokenString
}

// Handle - обрабатывает токен и создает узел AST
func (h *LiteralHandler) Handle(ctx *common.ParseContext) (interface{}, error) {
	// Проверяем защиту от рекурсии
	if err := ctx.Guard.Enter(); err != nil {
		return nil, err
	}
	defer ctx.Guard.Exit()

	token := ctx.TokenStream.Current()

	if h.verbose {
		fmt.Printf("DEBUG: LiteralHandler.Handle called with token: %s (%s)\n", token.Value, token.Type)
	}

	// Для чисел проверяем, не является ли это частью большего выражения
	if token.Type == lexer.TokenNumber && ctx.TokenStream.HasMore() {
		nextToken := ctx.TokenStream.Peek()
		if nextToken.Type == lexer.TokenPlus || nextToken.Type == lexer.TokenMinus ||
			nextToken.Type == lexer.TokenMultiply || nextToken.Type == lexer.TokenSlash ||
			nextToken.Type == lexer.TokenPower || nextToken.Type == lexer.TokenEqual ||
			nextToken.Type == lexer.TokenNotEqual || nextToken.Type == lexer.TokenLess ||
			nextToken.Type == lexer.TokenGreater || nextToken.Type == lexer.TokenLessEqual ||
			nextToken.Type == lexer.TokenGreaterEqual || nextToken.Type == lexer.TokenAnd ||
			nextToken.Type == lexer.TokenOr || nextToken.Type == lexer.TokenBitwiseOr ||
			nextToken.Type == lexer.TokenAmpersand || nextToken.Type == lexer.TokenCaret ||
			nextToken.Type == lexer.TokenDoubleLeftAngle || nextToken.Type == lexer.TokenDoubleRightAngle ||
			nextToken.Type == lexer.TokenModulo || nextToken.Type == lexer.TokenConcat ||
			nextToken.Type == lexer.TokenQuestion { // Для тернарных и Elvis операторов
			// Это часть большего выражения, пусть ExpressionHandler или BinaryExpressionHandler обработает
			return nil, nil
		}
	}

	// Обрабатываем отрицательные числа (MINUS followed by NUMBER)
	if token.Type == lexer.TokenMinus {
		// Проверяем, следующий токен - это число
		if !ctx.TokenStream.HasMore() || ctx.TokenStream.Peek().Type != lexer.TokenNumber {
			return nil, nil // Не обрабатываем, пусть другие обработчики разбираются
		}

		// Потребляем минус
		ctx.TokenStream.Consume()

		// Потребляем число
		numberToken := ctx.TokenStream.Consume()

		// Проверим, что следующий токен не является оператором
		// Если является, то это часть большего выражения, откатимся
		if ctx.TokenStream.HasMore() {
			nextToken := ctx.TokenStream.Peek()
			if nextToken.Type == lexer.TokenPlus || nextToken.Type == lexer.TokenMinus ||
				nextToken.Type == lexer.TokenMultiply || nextToken.Type == lexer.TokenSlash ||
				nextToken.Type == lexer.TokenPower || nextToken.Type == lexer.TokenEqual ||
				nextToken.Type == lexer.TokenNotEqual || nextToken.Type == lexer.TokenLess ||
				nextToken.Type == lexer.TokenGreater || nextToken.Type == lexer.TokenLessEqual ||
				nextToken.Type == lexer.TokenGreaterEqual {
				// Это часть большего выражения, откатимся и вернем nil
				// Но мы уже потребили токены, так что это сложно
				// Вместо этого, всегда обрабатываем отрицательные числа как литералы
				// А ExpressionHandler пусть обрабатывает только положительные числа
			}
		}

		// Парсим число и создаем отрицательный NumberLiteral
		parsedValue, err := parseNumber(numberToken.Value)
		if err != nil {
			return nil, newErrorWithTokenPos(numberToken, "failed to parse number '%s': %v", numberToken.Value, err)
		}

		// Создаем отрицательное значение
		switch v := parsedValue.(type) {
		case *big.Int:
			negativeInt := new(big.Int).Neg(v)
			return createNumberLiteral(numberToken, negativeInt), nil
		case float64:
			return createNumberLiteral(numberToken, -v), nil
		default:
			return nil, newErrorWithTokenPos(numberToken, "unsupported number type: %T", parsedValue)
		}
	}

	// Обрабатываем булевые значения
	if token.Type == lexer.TokenTrue || token.Type == lexer.TokenFalse {
		// Проверим, что следующий токен не является оператором (чтобы не конфликтовать с выражениями)
		if ctx.TokenStream.HasMore() {
			nextToken := ctx.TokenStream.Peek()
			if nextToken.Type == lexer.TokenPlus || nextToken.Type == lexer.TokenMinus ||
				nextToken.Type == lexer.TokenMultiply || nextToken.Type == lexer.TokenSlash ||
				nextToken.Type == lexer.TokenPower || nextToken.Type == lexer.TokenEqual ||
				nextToken.Type == lexer.TokenNotEqual || nextToken.Type == lexer.TokenLess ||
				nextToken.Type == lexer.TokenGreater || nextToken.Type == lexer.TokenLessEqual ||
				nextToken.Type == lexer.TokenGreaterEqual || 				nextToken.Type == lexer.TokenAnd ||
				nextToken.Type == lexer.TokenOr || nextToken.Type == lexer.TokenQuestion {
				// Это часть большего выражения, пусть ExpressionHandler или BinaryExpressionHandler обработает
				return nil, nil
			}
		}
		ctx.TokenStream.Consume()
		value := token.Type == lexer.TokenTrue
		return ast.NewBooleanLiteral(token, value), nil
	}

	// Обрабатываем строковые значения
	if token.Type == lexer.TokenString {
		// Проверим, что следующий токен не является оператором (чтобы не конфликтовать с выражениями)
		if ctx.TokenStream.HasMore() {
			nextToken := ctx.TokenStream.Peek()
			if nextToken.Type == lexer.TokenPlus || nextToken.Type == lexer.TokenMinus ||
				nextToken.Type == lexer.TokenMultiply || nextToken.Type == lexer.TokenSlash ||
				nextToken.Type == lexer.TokenPower || nextToken.Type == lexer.TokenEqual ||
				nextToken.Type == lexer.TokenNotEqual || nextToken.Type == lexer.TokenLess ||
				nextToken.Type == lexer.TokenGreater || nextToken.Type == lexer.TokenLessEqual ||
				nextToken.Type == lexer.TokenGreaterEqual || 				nextToken.Type == lexer.TokenAnd ||
				nextToken.Type == lexer.TokenOr || nextToken.Type == lexer.TokenQuestion {
				// Это часть большего выражения, пусть ExpressionHandler или BinaryExpressionHandler обработает
				return nil, nil
			}
		}
		ctx.TokenStream.Consume()
		// Token.Value уже содержит processed string value после escape sequences
		// Для Raw используем оригинальный токен из input
		return ast.NewStringLiteral(token, token.Value, token.Value), nil
	}

	// Handle nil literal
	if token.Type == lexer.TokenNil {
		// Проверим, что следующий токен не является оператором (чтобы не конфликтовать с выражениями)
		if ctx.TokenStream.HasMore() {
			nextToken := ctx.TokenStream.Peek()
			if nextToken.Type == lexer.TokenPlus || nextToken.Type == lexer.TokenMinus ||
				nextToken.Type == lexer.TokenMultiply || nextToken.Type == lexer.TokenSlash ||
				nextToken.Type == lexer.TokenPower || nextToken.Type == lexer.TokenEqual ||
				nextToken.Type == lexer.TokenNotEqual || nextToken.Type == lexer.TokenLess ||
				nextToken.Type == lexer.TokenGreater || nextToken.Type == lexer.TokenLessEqual ||
				nextToken.Type == lexer.TokenGreaterEqual || 				nextToken.Type == lexer.TokenAnd ||
				nextToken.Type == lexer.TokenOr || nextToken.Type == lexer.TokenQuestion {
				// Это часть большего выражения, пусть ExpressionHandler или BinaryExpressionHandler обработает
				return nil, nil
			}
		}
		ctx.TokenStream.Consume()
		return ast.NewNilLiteral(token), nil
	}

	// Проверяем, что это токен NUMBER
	if token.Type != lexer.TokenNumber {
		return nil, nil
	}

	// Проверим, что следующий токен не является оператором (чтобы не конфликтовать с выражениями)
	if ctx.TokenStream.HasMore() {
		nextToken := ctx.TokenStream.Peek()
		if nextToken.Type == lexer.TokenPlus || nextToken.Type == lexer.TokenMinus ||
			nextToken.Type == lexer.TokenMultiply || nextToken.Type == lexer.TokenSlash ||
			nextToken.Type == lexer.TokenPower || nextToken.Type == lexer.TokenEqual ||
			nextToken.Type == lexer.TokenNotEqual || nextToken.Type == lexer.TokenLess ||
			nextToken.Type == lexer.TokenGreater || nextToken.Type == lexer.TokenLessEqual ||
			nextToken.Type == lexer.TokenGreaterEqual || nextToken.Type == lexer.TokenAnd ||
			nextToken.Type == lexer.TokenOr || nextToken.Type == lexer.TokenBitwiseOr ||
			nextToken.Type == lexer.TokenAmpersand || nextToken.Type == lexer.TokenCaret ||
			nextToken.Type == lexer.TokenDoubleLeftAngle || nextToken.Type == lexer.TokenDoubleRightAngle ||
			nextToken.Type == lexer.TokenModulo || nextToken.Type == lexer.TokenConcat {
			return nil, nil // Не обрабатываем, пусть expression обработчик разберется
		}
	}

	// Потребляем токен
	ctx.TokenStream.Consume()

	// Используем нашу функцию parseNumber для создания NumberLiteral
	parsedValue, err := parseNumber(token.Value)
	if err != nil {
		return nil, newErrorWithTokenPos(token, "failed to parse number '%s': %v", token.Value, err)
	}
	numberLiteral := createNumberLiteral(token, parsedValue)

	return numberLiteral, nil
}

// Config возвращает конфигурацию обработчика
func (h *LiteralHandler) Config() common.HandlerConfig {
	return h.config
}

// Name возвращает имя обработчика
func (h *LiteralHandler) Name() string {
	return h.config.Name
}
