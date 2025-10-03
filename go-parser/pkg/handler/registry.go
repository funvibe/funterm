package handler

import (
	"fmt"
	"sort"

	"go-parser/pkg/common"
	"go-parser/pkg/lexer"
)

type HandlerRegistry interface {
	Register(tokenType lexer.TokenType, handler common.Handler) error
	GetHandlers(tokenType lexer.TokenType) []common.Handler
	GetBestHandler(tokenType lexer.TokenType) (common.Handler, error)
	GetFallbackHandlers(tokenType lexer.TokenType) []common.Handler
}

type SimpleHandlerRegistry struct {
	handlers map[lexer.TokenType][]common.Handler
}

func NewHandlerRegistry() *SimpleHandlerRegistry {
	return &SimpleHandlerRegistry{
		handlers: make(map[lexer.TokenType][]common.Handler),
	}
}

func (r *SimpleHandlerRegistry) Register(tokenType lexer.TokenType, handler common.Handler) error {
	if !handler.Config().IsEnabled {
		return nil
	}

	// Проверяем уникальность Order в пределах типа токена
	for _, h := range r.handlers[tokenType] {
		if h.Config().Order == handler.Config().Order && handler.Config().Order != 0 {
			return fmt.Errorf("handler with order %d already exists for token type %s",
				handler.Config().Order, tokenType)
		}
	}

	r.handlers[tokenType] = append(r.handlers[tokenType], handler)
	r.sortHandlers(tokenType)
	return nil
}

func (r *SimpleHandlerRegistry) sortHandlers(tokenType lexer.TokenType) {
	handlers := r.handlers[tokenType]
	sort.Slice(handlers, func(i, j int) bool {
		configI, configJ := handlers[i].Config(), handlers[j].Config()

		// Сначала по приоритету (по убыванию)
		if configI.Priority != configJ.Priority {
			return configI.Priority > configJ.Priority
		}

		// Потом по Order (по возрастанию)
		if configI.Order != configJ.Order {
			return configI.Order < configJ.Order
		}

		// По имени для стабильности
		return configI.Name < configJ.Name
	})
}

func (r *SimpleHandlerRegistry) GetHandlers(tokenType lexer.TokenType) []common.Handler {
	handlers := r.handlers[tokenType]
	result := make([]common.Handler, 0, len(handlers))

	for _, h := range handlers {
		if !h.Config().IsFallback {
			result = append(result, h)
		}
	}

	return result
}

func (r *SimpleHandlerRegistry) GetBestHandler(tokenType lexer.TokenType) (common.Handler, error) {
	handlers := r.GetHandlers(tokenType)
	if len(handlers) == 0 {
		return nil, fmt.Errorf("no handler found for token type %s", tokenType)
	}
	return handlers[0], nil
}

func (r *SimpleHandlerRegistry) GetFallbackHandlers(tokenType lexer.TokenType) []common.Handler {
	handlers := r.handlers[tokenType]
	result := make([]common.Handler, 0)

	for _, h := range handlers {
		if h.Config().IsFallback {
			result = append(result, h)
		}
	}

	// Сортируем fallback по их приоритету
	sort.Slice(result, func(i, j int) bool {
		return result[i].Config().FallbackPriority > result[j].Config().FallbackPriority
	})

	return result
}
