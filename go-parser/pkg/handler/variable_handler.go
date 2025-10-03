package handler

import (
	"fmt"

	"go-parser/pkg/common"
	"go-parser/pkg/config"
	"go-parser/pkg/lexer"
)

// VariableHandler - обработчик переменных (идентификаторов)
type VariableHandler struct {
	config       common.HandlerConfig
	customParams map[string]interface{}
}

// NewVariableHandler - создает новый обработчик переменных
func NewVariableHandler(config config.ConstructHandlerConfig) *VariableHandler {
	handlerConfig := common.HandlerConfig{
		IsEnabled:        config.IsEnabled,
		Priority:         config.Priority,
		Order:            config.Order,
		FallbackPriority: config.FallbackPriority,
		IsFallback:       config.IsFallback,
		Name:             config.Name,
	}

	return &VariableHandler{
		config:       handlerConfig,
		customParams: config.CustomParams,
	}
}

// CanHandle - проверяет, может ли обработчик обработать токен
func (h *VariableHandler) CanHandle(token lexer.Token) bool {
	// В будущем здесь будет проверка на TokenIdentifier
	// Пока возвращаем false, так как соответствующий токен не реализован
	return false
}

// Handle - обрабатывает переменную
func (h *VariableHandler) Handle(ctx *common.ParseContext) (interface{}, error) {
	// В будущем здесь будет обработка идентификаторов
	return nil, fmt.Errorf("variable handler not implemented yet")
}

// Config - возвращает конфигурацию обработчика
func (h *VariableHandler) Config() common.HandlerConfig {
	return h.config
}

// Name - возвращает имя обработчика
func (h *VariableHandler) Name() string {
	return h.config.Name
}
