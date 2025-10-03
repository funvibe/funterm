package handler

import (
	"fmt"
	"go-parser/pkg/ast"
	"go-parser/pkg/common"
	"go-parser/pkg/config"
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

// parseOperand парсит левый операнд выражения (идентификаторы, литералы, скобки)
func (p *UnifiedExpressionParser) parseOperand(ctx *common.ParseContext) (ast.Expression, error) {
	// Используем parseOperand из BinaryExpressionHandler
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
