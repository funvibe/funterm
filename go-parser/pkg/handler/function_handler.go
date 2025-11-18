package handler

import (
	"go-parser/pkg/common"
	"go-parser/pkg/config"
	"go-parser/pkg/lexer"
)

// FunctionHandler - обработчик функций (определение и вызов)
type FunctionHandler struct {
	config       common.HandlerConfig
	customParams map[string]interface{}
}

// NewFunctionHandler - создает новый обработчик функций
func NewFunctionHandler(config config.ConstructHandlerConfig) *FunctionHandler {
	handlerConfig := common.HandlerConfig{
		IsEnabled:        config.IsEnabled,
		Priority:         config.Priority,
		Order:            config.Order,
		FallbackPriority: config.FallbackPriority,
		IsFallback:       config.IsFallback,
		Name:             config.Name,
	}

	return &FunctionHandler{
		config:       handlerConfig,
		customParams: config.CustomParams,
	}
}

// CanHandle - проверяет, может ли обработчик обработать токен
func (h *FunctionHandler) CanHandle(token lexer.Token) bool {
	// В будущем здесь будет проверка на TokenKeyword (для определения функций)
	// или TokenIdentifier (для вызова функций)
	// Пока возвращаем false, так как соответствующие токены не реализованы
	return false
}

// Handle - обрабатывает функцию
func (h *FunctionHandler) Handle(ctx *common.ParseContext) (interface{}, error) {
	// В будущем здесь будет обработка определения и вызова функций
	return nil, newErrorWithPos(nil, "function handler not implemented yet")
}

// Config - возвращает конфигурацию обработчика
func (h *FunctionHandler) Config() common.HandlerConfig {
	return h.config
}

// Name - возвращает имя обработчика
func (h *FunctionHandler) Name() string {
	return h.config.Name
}

// SupportsAnonymous - проверяет, поддерживаются ли анонимные функции
func (h *FunctionHandler) SupportsAnonymous() bool {
	if supportsAnonymous, ok := h.customParams["supportsAnonymous"].(bool); ok {
		return supportsAnonymous
	}
	return true // значение по умолчанию
}

// SupportsParams - проверяет, поддерживаются ли параметры
func (h *FunctionHandler) SupportsParams() bool {
	if supportsParams, ok := h.customParams["supportsParams"].(bool); ok {
		return supportsParams
	}
	return true // значение по умолчанию
}

// SupportsReturn - проверяет, поддерживаются ли возвращаемые значения
func (h *FunctionHandler) SupportsReturn() bool {
	if supportsReturn, ok := h.customParams["supportsReturn"].(bool); ok {
		return supportsReturn
	}
	return true // значение по умолчанию
}

// GetMaxParams - возвращает максимальное количество параметров
func (h *FunctionHandler) GetMaxParams() int {
	if maxParams, ok := h.customParams["maxParams"].(int); ok {
		return maxParams
	}
	return 10 // значение по умолчанию
}
