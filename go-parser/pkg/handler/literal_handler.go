package handler

import (
	"fmt"

	"go-parser/pkg/common"
	"go-parser/pkg/config"
	"go-parser/pkg/lexer"
)

// LiteralHandler - обработчик литералов (числа, строки и т.д.)
type LiteralHandler struct {
	config       common.HandlerConfig
	customParams map[string]interface{}
}

// NewLiteralHandler - создает новый обработчик литералов
func NewLiteralHandler(config config.ConstructHandlerConfig) *LiteralHandler {
	handlerConfig := common.HandlerConfig{
		IsEnabled:        config.IsEnabled,
		Priority:         config.Priority,
		Order:            config.Order,
		FallbackPriority: config.FallbackPriority,
		IsFallback:       config.IsFallback,
		Name:             config.Name,
	}

	return &LiteralHandler{
		config:       handlerConfig,
		customParams: config.CustomParams,
	}
}

// CanHandle - проверяет, может ли обработчик обработать токен
func (h *LiteralHandler) CanHandle(token lexer.Token) bool {
	// В будущем здесь будет проверка на TokenNumber, TokenString и т.д.
	// Пока возвращаем false, так как соответствующие токены не реализованы
	return false
}

// Handle - обрабатывает литерал
func (h *LiteralHandler) Handle(ctx *common.ParseContext) (interface{}, error) {
	// В будущем здесь будет обработка чисел, строк и других литералов
	return nil, fmt.Errorf("literal handler not implemented yet")
}

// Config - возвращает конфигурацию обработчика
func (h *LiteralHandler) Config() common.HandlerConfig {
	return h.config
}

// Name - возвращает имя обработчика
func (h *LiteralHandler) Name() string {
	return h.config.Name
}
