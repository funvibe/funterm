package handler

import (
	"fmt"

	"go-parser/pkg/ast"
	"go-parser/pkg/common"
	"go-parser/pkg/config"
	"go-parser/pkg/lexer"
)

// ContinueHandler - обработчик оператора continue с контекстной проверкой
type ContinueHandler struct {
	config config.ConstructHandlerConfig
}

// NewContinueHandler создает новый обработчик оператора continue
func NewContinueHandler(config config.ConstructHandlerConfig) *ContinueHandler {
	return &ContinueHandler{
		config: config,
	}
}

// CanHandle проверяет, может ли обработчик обработать токен
func (h *ContinueHandler) CanHandle(token lexer.Token) bool {
	// Обрабатываем токен 'continue'
	return token.Type == lexer.TokenContinue
}

// Handle обрабатывает оператор continue с контекстной проверкой
func (h *ContinueHandler) Handle(ctx *common.ParseContext) (interface{}, error) {
	tokenStream := ctx.TokenStream

	// 1. Проверяем токен 'continue'
	continueToken := tokenStream.Current()
	if continueToken.Type != lexer.TokenContinue {
		return nil, fmt.Errorf("expected 'continue', got %s", continueToken.Type)
	}

	// 2. КОНТЕКСТНАЯ ВАЛИДАЦИЯ (ОБЯЗАТЕЛЬНО по ТЗ)
	// Проверяем, что continue используется внутри цикла
	if ctx.LoopDepth <= 0 {
		return nil, fmt.Errorf("continue statement can only be used inside a loop at line %d, column %d",
			continueToken.Line, continueToken.Column)
	}

	// 3. Потребляем токен 'continue'
	tokenStream.Consume()

	// 4. Создаем узел AST
	continueStatement := ast.NewContinueStatement(continueToken)

	return continueStatement, nil
}

// Config возвращает конфигурацию обработчика
func (h *ContinueHandler) Config() common.HandlerConfig {
	return common.HandlerConfig{
		IsEnabled: h.config.IsEnabled,
		Priority:  h.config.Priority,
		Name:      h.config.Name,
	}
}

// Name возвращает имя обработчика
func (h *ContinueHandler) Name() string {
	return h.config.Name
}
