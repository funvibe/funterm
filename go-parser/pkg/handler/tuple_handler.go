package handler

import (
	"fmt"

	"go-parser/pkg/common"
	"go-parser/pkg/config"
	"go-parser/pkg/lexer"
)

// TupleHandler - обработчик кортежей/tuple
type TupleHandler struct {
	config       common.HandlerConfig
	customParams map[string]interface{}
}

// NewTupleHandler - создает новый обработчик кортежей
func NewTupleHandler(config config.ConstructHandlerConfig) *TupleHandler {
	handlerConfig := common.HandlerConfig{
		IsEnabled:        config.IsEnabled,
		Priority:         config.Priority,
		Order:            config.Order,
		FallbackPriority: config.FallbackPriority,
		IsFallback:       config.IsFallback,
		Name:             config.Name,
	}

	return &TupleHandler{
		config:       handlerConfig,
		customParams: config.CustomParams,
	}
}

// CanHandle - проверяет, может ли обработчик обработать токен
func (h *TupleHandler) CanHandle(token lexer.Token) bool {
	// Кортежи начинаются с открывающей скобки, но имеют другую логику обработки,
	// чем группирующие конструкции
	return token.Type == lexer.TokenLeftParen
}

// Handle - обрабатывает кортеж
func (h *TupleHandler) Handle(ctx *common.ParseContext) (interface{}, error) {
	// В будущем здесь будет специальная обработка кортежей
	// Пока возвращаем ошибку, так как отдельный обработчик кортежей не реализован
	return nil, fmt.Errorf("tuple handler not implemented yet")
}

// Config - возвращает конфигурацию обработчика
func (h *TupleHandler) Config() common.HandlerConfig {
	return h.config
}

// Name - возвращает имя обработчика
func (h *TupleHandler) Name() string {
	return h.config.Name
}

// GetMaxElements - возвращает максимальное количество элементов в кортеже
func (h *TupleHandler) GetMaxElements() int {
	if maxElements, ok := h.customParams["maxElements"].(int); ok {
		return maxElements
	}
	return 100 // значение по умолчанию
}

// AllowEmpty - проверяет, разрешены ли пустые кортежи
func (h *TupleHandler) AllowEmpty() bool {
	if allowEmpty, ok := h.customParams["allowEmpty"].(bool); ok {
		return allowEmpty
	}
	return true // значение по умолчанию
}
