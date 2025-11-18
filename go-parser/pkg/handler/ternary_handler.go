package handler

import (
	"go-parser/pkg/common"
	"go-parser/pkg/config"
	"go-parser/pkg/lexer"
)

// TernaryHandler обрабатывает тернарные операторы (condition ? true_expr : false_expr)
// и элвис-операторы (left_expr ?: right_expr)
type TernaryHandler struct {
	config config.ConstructHandlerConfig
}

// NewTernaryHandler создает новый обработчик тернарных операторов
func NewTernaryHandler(config config.ConstructHandlerConfig) *TernaryHandler {
	return &TernaryHandler{
		config: config,
	}
}

// Handle обрабатывает тернарные и элвис-выражения
func (h *TernaryHandler) Handle(ctx *common.ParseContext) (interface{}, error) {
	tokenStream := ctx.TokenStream

	if !tokenStream.HasMore() {
		return nil, newErrorWithPos(tokenStream, "unexpected EOF")
	}

	// Проверяем, есть ли у нас токен ? (начало тернарного или элвис оператора)
	if tokenStream.Current().Type != lexer.TokenQuestion {
		return nil, newErrorWithTokenPos(tokenStream.Current(), "expected ? token, got %s", tokenStream.Current().Type)
	}

	_ = tokenStream.Consume() // Consuming question token

	// Проверяем, есть ли следующий токен после ?
	if !tokenStream.HasMore() {
		return nil, newErrorWithPos(tokenStream, "unexpected EOF after ?")
	}

	// Проверяем, является ли это элвис-оператором (?:)
	if tokenStream.Current().Type == lexer.TokenColon {
		// Это элвис-оператор: left_expr ?: right_expr
		_ = tokenStream.Consume() // Consuming colon token

		// Нам нужно вернуться назад, чтобы получить левую часть выражения
		// Это будет обработано бинарным обработчиком

		// Возвращаем специальную ошибку, чтобы позволить другим обработчикам попробовать
		return nil, newErrorWithPos(tokenStream, "elvis operator detected, needs binary handler")
	}

	// Это тернарный оператор: condition ? true_expr : false_expr

	// Нам нужно вернуться назад, чтобы получить условие и левую часть
	// Это будет обработано бинарным обработчиком

	// Возвращаем специальную ошибку, чтобы позволить другим обработчикам попробовать
	return nil, newErrorWithPos(tokenStream, "ternary operator detected, needs binary handler")
}

// CanHandle проверяет, может ли обработчик обработать текущий токен
func (h *TernaryHandler) CanHandle(token lexer.Token) bool {
	// Проверяем, есть ли токен ?
	return token.Type == lexer.TokenQuestion
}

// GetPriority возвращает приоритет обработчика
func (h *TernaryHandler) GetPriority() int {
	return h.config.Priority
}

// GetOrder возвращает порядок обработчика
func (h *TernaryHandler) GetOrder() int {
	return h.config.Order
}

// GetName возвращает имя обработчика
func (h *TernaryHandler) GetName() string {
	return h.config.Name
}

// IsEnabled проверяет, включен ли обработчик
func (h *TernaryHandler) IsEnabled() bool {
	return h.config.IsEnabled
}

// IsFallback проверяет, является ли обработчик fallback-обработчиком
func (h *TernaryHandler) IsFallback() bool {
	return h.config.IsFallback
}

// GetFallbackPriority возвращает приоритет fallback-обработчика
func (h *TernaryHandler) GetFallbackPriority() int {
	return h.config.FallbackPriority
}

// Config возвращает конфигурацию обработчика
func (h *TernaryHandler) Config() common.HandlerConfig {
	return common.HandlerConfig{
		IsEnabled:        h.IsEnabled(),
		Priority:         h.GetPriority(),
		Order:            h.GetOrder(),
		FallbackPriority: h.GetFallbackPriority(),
		IsFallback:       h.IsFallback(),
		Name:             h.GetName(),
	}
}

// Name возвращает имя обработчика
func (h *TernaryHandler) Name() string {
	return h.GetName()
}
