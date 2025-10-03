package handler

import (
	"fmt"

	"go-parser/pkg/ast"
	"go-parser/pkg/common"
	"go-parser/pkg/lexer"
)

type ParenthesesHandler struct {
	config common.HandlerConfig
}

func NewParenthesesHandler(priority, order int) *ParenthesesHandler {
	config := DefaultConfig("parentheses")
	config.Priority = priority
	config.Order = order
	return &ParenthesesHandler{
		config: config,
	}
}

func (h *ParenthesesHandler) CanHandle(token lexer.Token) bool {
	return token.Type == lexer.TokenLeftParen
}

func (h *ParenthesesHandler) Handle(ctx *common.ParseContext) (interface{}, error) {
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

func (h *ParenthesesHandler) Config() common.HandlerConfig {
	return h.config
}

func (h *ParenthesesHandler) Name() string {
	return h.config.Name
}
