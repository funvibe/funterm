package handler

import (
	"go-parser/pkg/ast"
	"go-parser/pkg/common"
	"go-parser/pkg/config"
	"go-parser/pkg/lexer"
)

// VariableReadHandler - обработчик для чтения переменных
type VariableReadHandler struct {
	config common.HandlerConfig
}

// NewVariableReadHandler создает новый обработчик чтения переменных
func NewVariableReadHandler(config config.ConstructHandlerConfig) *VariableReadHandler {
	return &VariableReadHandler{
		config: common.HandlerConfig{
			IsEnabled: config.IsEnabled,
			Priority:  config.Priority,
			Name:      config.Name,
		},
	}
}

// CanHandle проверяет, может ли обработчик обработать токен
func (h *VariableReadHandler) CanHandle(token lexer.Token) bool {
	return token.Type == lexer.TokenIdentifier
}

// Handle обрабатывает чтение переменной
func (h *VariableReadHandler) Handle(ctx *common.ParseContext) (interface{}, error) {
	// Проверяем защиту от рекурсии
	if err := ctx.Guard.Enter(); err != nil {
		return nil, err
	}
	defer ctx.Guard.Exit()

	// Проверяем следующий токен - если это DOT, то это qualified переменная, позволяем другим обработчикам
	if ctx.TokenStream.HasMore() && ctx.TokenStream.Current().Type == lexer.TokenDot {
		return nil, nil
	}

	// Потребляем идентификатор
	token := ctx.TokenStream.Consume()

	// Создаем AST узлы
	identifier := ast.NewIdentifier(token, token.Value)
	variableRead := ast.NewVariableRead(identifier)

	return variableRead, nil
}

// Config возвращает конфигурацию обработчика
func (h *VariableReadHandler) Config() common.HandlerConfig {
	return h.config
}

// Name возвращает имя обработчика
func (h *VariableReadHandler) Name() string {
	return h.config.Name
}
