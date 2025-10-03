package handler

import (
	"fmt"

	"go-parser/pkg/ast"
	"go-parser/pkg/common"
	"go-parser/pkg/config"
	"go-parser/pkg/lexer"
)

// GroupHandler - обработчик группирующих конструкций (скобки)
type GroupHandler struct {
	config       common.HandlerConfig
	customParams map[string]interface{}
}

// NewGroupHandler - создает новый обработчик группирующих конструкций
func NewGroupHandler(config config.ConstructHandlerConfig) *GroupHandler {
	handlerConfig := common.HandlerConfig{
		IsEnabled:        config.IsEnabled,
		Priority:         config.Priority,
		Order:            config.Order,
		FallbackPriority: config.FallbackPriority,
		IsFallback:       config.IsFallback,
		Name:             config.Name,
	}

	return &GroupHandler{
		config:       handlerConfig,
		customParams: config.CustomParams,
	}
}

// CanHandle - проверяет, может ли обработчик обработать токен
func (h *GroupHandler) CanHandle(token lexer.Token) bool {
	return token.Type == lexer.TokenLeftParen
}

// Handle - обрабатывает конструкцию
func (h *GroupHandler) Handle(ctx *common.ParseContext) (interface{}, error) {
	// Проверяем защиту от рекурсии
	if err := ctx.Guard.Enter(); err != nil {
		return nil, err
	}
	defer ctx.Guard.Exit()

	// Потребляем открывающую скобку
	openParen := ctx.TokenStream.Consume()
	if openParen.Type != lexer.TokenLeftParen {
		return nil, fmt.Errorf("expected '(', got %s", openParen.Type)
	}

	var children []ast.Node

	// Обрабатываем содержимое до закрывающей скобки
	for ctx.TokenStream.HasMore() {
		current := ctx.TokenStream.Current()

		if current.Type == lexer.TokenRightParen {
			// Потребляем закрывающую скобку и завершаем
			closeParen := ctx.TokenStream.Consume()

			// Создаем узел скобок
			parenNode := ast.NewParenthesesNode(openParen, closeParen)
			for _, child := range children {
				parenNode.AddChild(child)
			}

			return parenNode, nil
		}

		if current.Type == lexer.TokenLeftParen {
			// Прямой рекурсивный вызов обработчика для вложенных скобок
			childNode, err := h.Handle(ctx)
			if err != nil {
				return nil, err
			}
			if childNode != nil {
				if node, ok := childNode.(ast.Node); ok {
					children = append(children, node)
				}
			}
		} else {
			// Пропускаем неизвестные токены
			ctx.TokenStream.Consume()
		}
	}

	// Если дошли сюда, значит не нашли закрывающую скобку
	return nil, fmt.Errorf("unclosed parentheses")
}

// Config - возвращает конфигурацию обработчика
func (h *GroupHandler) Config() common.HandlerConfig {
	return h.config
}

// Name - возвращает имя обработчика
func (h *GroupHandler) Name() string {
	return h.config.Name
}

// GetMaxDepth - возвращает максимальную глубину вложенности
func (h *GroupHandler) GetMaxDepth() int {
	if maxDepth, ok := h.customParams["maxDepth"].(int); ok {
		return maxDepth
	}
	return 100 // значение по умолчанию
}
