package handler

import (
	"fmt"

	"go-parser/pkg/ast"
	"go-parser/pkg/common"
	"go-parser/pkg/config"
	"go-parser/pkg/lexer"
)

// BitstringPatternAssignmentHandler - обработчик присваивания с bitstring pattern слева
type BitstringPatternAssignmentHandler struct {
	config  common.HandlerConfig
	verbose bool
}

// NewBitstringPatternAssignmentHandler создает новый обработчик
func NewBitstringPatternAssignmentHandler(priority, order int) *BitstringPatternAssignmentHandler {
	return NewBitstringPatternAssignmentHandlerWithVerbose(priority, order, false)
}

// NewBitstringPatternAssignmentHandlerWithVerbose создает новый обработчик с verbose режимом
func NewBitstringPatternAssignmentHandlerWithVerbose(priority, order int, verbose bool) *BitstringPatternAssignmentHandler {
	config := DefaultConfig("bitstring_pattern_assignment")
	config.Priority = priority
	config.Order = order
	return &BitstringPatternAssignmentHandler{
		config:  config,
		verbose: verbose,
	}
}

// CanHandle проверяет, может ли обработчик обработать токен
func (h *BitstringPatternAssignmentHandler) CanHandle(token lexer.Token) bool {
	// Обрабатываем только если токен начинается с <<
	return token.Type == lexer.TokenDoubleLeftAngle
}

// Handle обрабатывает bitstring pattern assignment
func (h *BitstringPatternAssignmentHandler) Handle(ctx *common.ParseContext) (interface{}, error) {
	if h.verbose {
		fmt.Printf("DEBUG: BitstringPatternAssignmentHandler.Handle - ENTRY POINT\n")
	}

	// Проверяем защиту от рекурсии
	if err := ctx.Guard.Enter(); err != nil {
		return nil, err
	}
	defer ctx.Guard.Exit()

	// Парсим bitstring pattern слева
	bitstringConfig := config.ConstructHandlerConfig{
		ConstructType: common.ConstructBitstring,
		Name:          "bitstring-temp",
		Priority:      105,
		Order:         3,
		IsEnabled:     true,
		IsFallback:    false,
		TokenPatterns: []config.TokenPattern{
			{TokenType: lexer.TokenDoubleLeftAngle, Offset: 0},
		},
	}
	bitstringHandler := NewBitstringHandler(bitstringConfig)
	if !bitstringHandler.CanHandle(ctx.TokenStream.Current()) {
		return nil, fmt.Errorf("expected bitstring pattern, got %s", ctx.TokenStream.Current().Type)
	}

	patternResult, err := bitstringHandler.Handle(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to parse bitstring pattern: %v", err)
	}

	pattern, ok := patternResult.(*ast.BitstringExpression)
	if !ok {
		return nil, fmt.Errorf("expected BitstringExpression, got %T", patternResult)
	}

	// Проверяем наличие знака присваивания
	if !ctx.TokenStream.HasMore() || (ctx.TokenStream.Current().Type != lexer.TokenAssign && ctx.TokenStream.Current().Type != lexer.TokenColonEquals) {
		return nil, fmt.Errorf("expected '=' or ':=' after bitstring pattern, got %s", ctx.TokenStream.Current().Type)
	}

	assignToken := ctx.TokenStream.Consume()

	// Парсим значение справа
	if !ctx.TokenStream.HasMore() {
		return nil, fmt.Errorf("expected value after '='")
	}

	var value ast.Expression
	currentToken := ctx.TokenStream.Current()

	// Используем существующую логику для парсинга значений
	if currentToken.Type == lexer.TokenIdentifier {
		// Может быть qualified variable или language call
		value, err = h.parseValue(ctx)
		if err != nil {
			return nil, err
		}
	} else if currentToken.Type == lexer.TokenDoubleLeftAngle {
		// Bitstring value
		valueResult, err := bitstringHandler.Handle(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to parse bitstring value: %v", err)
		}
		value, ok = valueResult.(ast.Expression)
		if !ok {
			return nil, fmt.Errorf("expected Expression from bitstring, got %T", valueResult)
		}
	} else {
		// Другие типы значений (строки, числа, etc.)
		value, err = h.parseValue(ctx)
		if err != nil {
			return nil, err
		}
	}

	return ast.NewBitstringPatternAssignment(pattern, assignToken, value), nil
}

// parseValue парсит значение справа от =
func (h *BitstringPatternAssignmentHandler) parseValue(ctx *common.ParseContext) (ast.Expression, error) {
	currentToken := ctx.TokenStream.Current()

	switch currentToken.Type {
	case lexer.TokenString:
		token := ctx.TokenStream.Consume()
		return &ast.StringLiteral{Value: token.Value, Raw: token.Value, Pos: tokenToPosition(token)}, nil
	case lexer.TokenNumber:
		token := ctx.TokenStream.Consume()
		value, err := parseNumber(token.Value)
		if err != nil {
			return nil, fmt.Errorf("invalid number format: %s", token.Value)
		}
		return createNumberLiteral(token, value), nil
	case lexer.TokenIdentifier:
		// Может быть qualified variable
		identifierToken := ctx.TokenStream.Consume()

		// Проверяем, есть ли DOT после идентификатора
		if ctx.TokenStream.HasMore() && ctx.TokenStream.Current().Type == lexer.TokenDot {
			ctx.TokenStream.Consume() // consume DOT
			if !ctx.TokenStream.HasMore() || ctx.TokenStream.Current().Type != lexer.TokenIdentifier {
				return nil, fmt.Errorf("expected identifier after DOT")
			}
			varToken := ctx.TokenStream.Consume()
			return ast.NewQualifiedIdentifier(identifierToken, varToken, identifierToken.Value, varToken.Value), nil
		}

		// Простой идентификатор
		return ast.NewIdentifier(identifierToken, identifierToken.Value), nil
	default:
		// Для языковых токенов
		if currentToken.IsLanguageIdentifierOrCallToken() {
			languageToken := ctx.TokenStream.Consume()
			if ctx.TokenStream.HasMore() && ctx.TokenStream.Current().Type == lexer.TokenDot {
				ctx.TokenStream.Consume() // consume DOT
				if !ctx.TokenStream.HasMore() || ctx.TokenStream.Current().Type != lexer.TokenIdentifier {
					return nil, fmt.Errorf("expected identifier after DOT")
				}
				varToken := ctx.TokenStream.Consume()
				return ast.NewQualifiedIdentifier(languageToken, varToken, languageToken.LanguageTokenToString(), varToken.Value), nil
			}
			return nil, fmt.Errorf("expected DOT after language token")
		}
		return nil, fmt.Errorf("unsupported value type: %s", currentToken.Type)
	}
}

// Config возвращает конфигурацию обработчика
func (h *BitstringPatternAssignmentHandler) Config() common.HandlerConfig {
	return h.config
}

// Name возвращает имя обработчика
func (h *BitstringPatternAssignmentHandler) Name() string {
	return "bitstring-pattern-assignment"
}
