package handler

import (
	"fmt"
	"go-parser/pkg/ast"
	"go-parser/pkg/common"
	"go-parser/pkg/config"
	"go-parser/pkg/lexer"
)

// UnifiedExpressionParser - централизованный парсер выражений с правильными приоритетами
type UnifiedExpressionParser struct {
	binaryHandler *BinaryExpressionHandler
	verbose       bool
}


// NewUnifiedExpressionParser создает новый централизованный парсер выражений
func NewUnifiedExpressionParser(verbose bool) *UnifiedExpressionParser {
	return &UnifiedExpressionParser{
		binaryHandler: NewBinaryExpressionHandlerWithVerbose(config.ConstructHandlerConfig{}, verbose),
		verbose:       verbose,
	}
}

// ParseExpression парсит любое выражение с правильными приоритетами операторов
func (p *UnifiedExpressionParser) ParseExpression(ctx *common.ParseContext) (ast.Expression, error) {
	if p.verbose {
		fmt.Printf("DEBUG: UnifiedExpressionParser.ParseExpression - current token: %s (%s)\n",
			ctx.TokenStream.Current().Value, ctx.TokenStream.Current().Type)
	}

	// Сначала парсим левый операнд
	leftOperand, err := p.parseOperand(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to parse operand: %v", err)
	}

	// Затем используем ParseFullExpression для обработки бинарных операторов
	result, err := p.binaryHandler.ParseFullExpression(ctx, leftOperand)
	if err != nil {
		return nil, fmt.Errorf("failed to parse full expression: %v", err)
	}

	if p.verbose {
		fmt.Printf("DEBUG: UnifiedExpressionParser.ParseExpression - successfully parsed: %T\n", result)
	}

	return result, nil
}

// ContinueParsingExpression продолжает парсинг выражения, начиная с уже готового левого операнда
// Используется для случаев, когда левый операнд уже был распарсен (например, language call)
// и нам нужно обработать бинарные операторы после него
func (p *UnifiedExpressionParser) ContinueParsingExpression(ctx *common.ParseContext, leftOperand ast.Expression) (ast.Expression, error) {
	if p.verbose {
		fmt.Printf("DEBUG: UnifiedExpressionParser.ContinueParsingExpression - left operand: %T, current token: %s (%s)\n",
			leftOperand, ctx.TokenStream.Current().Value, ctx.TokenStream.Current().Type)
	}

	// Используем ParseFullExpression для обработки бинарных операторов
	result, err := p.binaryHandler.ParseFullExpression(ctx, leftOperand)
	if err != nil {
		return nil, fmt.Errorf("failed to continue parsing expression: %v", err)
	}

	if p.verbose {
		fmt.Printf("DEBUG: UnifiedExpressionParser.ContinueParsingExpression - successfully parsed: %T\n", result)
	}

	return result, nil
}

// ExpressionHandler - обработчик для выражений
type ExpressionHandler struct {
	config  common.HandlerConfig
	verbose bool
	parser  *UnifiedExpressionParser
}

// NewExpressionHandler создает новый обработчик выражений
func NewExpressionHandler(config config.ConstructHandlerConfig) *ExpressionHandler {
	return NewExpressionHandlerWithVerbose(config, false)
}

// NewExpressionHandlerWithVerbose создает новый обработчик выражений с verbose режимом
func NewExpressionHandlerWithVerbose(config config.ConstructHandlerConfig, verbose bool) *ExpressionHandler {
	handlerConfig := DefaultConfig("expression")
	handlerConfig.Name = config.Name // Используем переданное имя
	handlerConfig.Priority = config.Priority
	handlerConfig.Order = config.Order
	return &ExpressionHandler{
		config:  handlerConfig,
		verbose: verbose,
		parser:  NewUnifiedExpressionParser(verbose),
	}
}

// CanHandle проверяет, может ли обработчик обработать токен
func (h *ExpressionHandler) CanHandle(token lexer.Token) bool {
	var canHandle bool

	// Определяем, какие токены обрабатывать на основе имени обработчика
	switch h.config.Name {
	case "expression-minus":
		canHandle = token.Type == lexer.TokenMinus
	case "expression-number":
		canHandle = token.Type == lexer.TokenNumber
	default:
		canHandle = false
	}

	if h.verbose && canHandle {
		fmt.Printf("DEBUG: ExpressionHandler(%s).CanHandle: token %s (%s) - YES\n", h.config.Name, token.Value, token.Type)
	}
	return canHandle
}

// Handle обрабатывает выражение
func (h *ExpressionHandler) Handle(ctx *common.ParseContext) (interface{}, error) {
	if h.verbose {
		fmt.Printf("DEBUG: ExpressionHandler.Handle called with token: %s (%s)\n",
			ctx.TokenStream.Current().Value, ctx.TokenStream.Current().Type)
	}

	// Для MINUS всегда пытаемся обработать как выражение
	// Используем UnifiedExpressionParser для парсинга
	expr, err := h.parser.ParseExpression(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to parse expression: %v", err)
	}

	return expr, nil
}

// Name возвращает имя обработчика
func (h *ExpressionHandler) Name() string {
	return h.config.Name
}

// Config возвращает конфигурацию обработчика
func (h *ExpressionHandler) Config() common.HandlerConfig {
	return h.config
}

// parseOperand парсит левый операнд выражения (идентификаторы, литералы, скобки)
func (p *UnifiedExpressionParser) parseOperand(ctx *common.ParseContext) (ast.Expression, error) {
	tokenStream := ctx.TokenStream

	if !tokenStream.HasMore() {
		return nil, fmt.Errorf("unexpected EOF in operand")
	}

	token := tokenStream.Current()

	// Проверяем унарные операторы
	if isUnaryOperator(token.Type) {
		unaryHandler := NewUnaryExpressionHandler(config.ConstructHandlerConfig{})
		result, err := unaryHandler.Handle(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to parse unary expression: %v", err)
		}
		if expr, ok := result.(*ast.UnaryExpression); ok {
			return expr, nil
		}
		return nil, fmt.Errorf("expected UnaryExpression, got %T", result)
	}

	// Используем parseOperand из BinaryExpressionHandler для обычных операндов
	return p.binaryHandler.parseOperand(ctx)
}

// ParseExpressionFromOperand парсит выражение начиная с уже распарсенного левого операнда
func (p *UnifiedExpressionParser) ParseExpressionFromOperand(ctx *common.ParseContext, leftOperand ast.Expression) (ast.Expression, error) {
	if p.verbose {
		fmt.Printf("DEBUG: UnifiedExpressionParser.ParseExpressionFromOperand - left: %T, current token: %s (%s)\n",
			leftOperand, ctx.TokenStream.Current().Value, ctx.TokenStream.Current().Type)
	}

	// Используем ParseFullExpression для обработки с правильными приоритетами
	result, err := p.binaryHandler.ParseFullExpression(ctx, leftOperand)
	if err != nil {
		return nil, fmt.Errorf("failed to parse expression from operand: %v", err)
	}

	if p.verbose {
		fmt.Printf("DEBUG: UnifiedExpressionParser.ParseExpressionFromOperand - successfully parsed: %T\n", result)
	}

	return result, nil
}
