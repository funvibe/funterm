package handler

import (
	"fmt"

	"go-parser/pkg/ast"
	"go-parser/pkg/common"
	"go-parser/pkg/config"
	"go-parser/pkg/lexer"
)

// BreakHandler - обработчик оператора break с контекстной проверкой
type BreakHandler struct {
	config config.ConstructHandlerConfig
}

// NewBreakHandler создает новый обработчик оператора break
func NewBreakHandler(config config.ConstructHandlerConfig) *BreakHandler {
	return &BreakHandler{
		config: config,
	}
}

// CanHandle проверяет, может ли обработчик обработать токен
func (h *BreakHandler) CanHandle(token lexer.Token) bool {
	// Обрабатываем токен 'break'
	return token.Type == lexer.TokenBreak
}

// Handle обрабатывает оператор break с контекстной проверкой
func (h *BreakHandler) Handle(ctx *common.ParseContext) (interface{}, error) {
	tokenStream := ctx.TokenStream

	// 1. Проверяем токен 'break'
	breakToken := tokenStream.Current()
	if breakToken.Type != lexer.TokenBreak {
		return nil, fmt.Errorf("expected 'break', got %s", breakToken.Type)
	}

	// 2. КОНТЕКСТНАЯ ВАЛИДАЦИЯ (ОБЯЗАТЕЛЬНО по ТЗ)
	// Проверяем, что break используется внутри цикла
	if ctx.LoopDepth <= 0 {
		return nil, fmt.Errorf("break statement can only be used inside a loop at line %d, column %d",
			breakToken.Line, breakToken.Column)
	}

	// 3. Потребляем токен 'break'
	tokenStream.Consume()

	// 4. Создаем узел AST
	breakStatement := ast.NewBreakStatement(breakToken)

	return breakStatement, nil
}

// Config возвращает конфигурацию обработчика
func (h *BreakHandler) Config() common.HandlerConfig {
	return common.HandlerConfig{
		IsEnabled: h.config.IsEnabled,
		Priority:  h.config.Priority,
		Name:      h.config.Name,
	}
}

// Name возвращает имя обработчика
func (h *BreakHandler) Name() string {
	return h.config.Name
}
