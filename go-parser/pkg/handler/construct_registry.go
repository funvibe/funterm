package handler

import (
	"fmt"
	"sort"

	"go-parser/pkg/common"
	"go-parser/pkg/config"
	"go-parser/pkg/lexer"
)

// ConstructHandlerRegistry - реестр обработчиков для конструкций
type ConstructHandlerRegistry interface {
	RegisterConstructHandler(handler common.Handler, config config.ConstructHandlerConfig) error
	GetHandlerForTokenSequence(tokens []lexer.Token) (common.Handler, error)
	GetAllHandlersForTokenSequence(tokens []lexer.Token) []common.Handler
	GetHandlerByConstruct(constructType common.ConstructType) (common.Handler, error)
	GetFallbackHandlers(constructType common.ConstructType) []common.Handler
}

// ConstructHandlerRegistryImpl - реализация реестра
type ConstructHandlerRegistryImpl struct {
	handlersByConstruct map[common.ConstructType][]*HandlerReference
	tokenPatternIndex   map[lexer.TokenType][]*HandlerReference
}

// HandlerReference - ссылка на обработчик с метаданными
type HandlerReference struct {
	Handler       common.Handler
	Config        config.ConstructHandlerConfig
	ConstructType common.ConstructType
}

// NewConstructHandlerRegistry - создает новый реестр
func NewConstructHandlerRegistry() *ConstructHandlerRegistryImpl {
	return &ConstructHandlerRegistryImpl{
		handlersByConstruct: make(map[common.ConstructType][]*HandlerReference),
		tokenPatternIndex:   make(map[lexer.TokenType][]*HandlerReference),
	}
}

// RegisterConstructHandler - регистрирует обработчик
func (r *ConstructHandlerRegistryImpl) RegisterConstructHandler(
	handler common.Handler,
	config config.ConstructHandlerConfig,
) error {
	if !config.IsEnabled {
		return nil
	}

	ref := &HandlerReference{
		Handler:       handler,
		Config:        config,
		ConstructType: config.ConstructType,
	}

	// Добавляем в реестр по типу конструкции
	r.handlersByConstruct[config.ConstructType] = append(
		r.handlersByConstruct[config.ConstructType],
		ref,
	)

	// Индексируем по типам токенов из паттернов
	for _, pattern := range config.TokenPatterns {
		r.tokenPatternIndex[pattern.TokenType] = append(
			r.tokenPatternIndex[pattern.TokenType],
			ref,
		)
	}

	// Сортируем обработчики для данного типа конструкции
	r.sortHandlersByConstruct(config.ConstructType)

	return nil
}

// GetHandlerForTokenSequence - получает обработчик для последовательности токенов
func (r *ConstructHandlerRegistryImpl) GetHandlerForTokenSequence(
	tokens []lexer.Token,
) (common.Handler, error) {
	if len(tokens) == 0 {
		return nil, fmt.Errorf("empty token sequence")
	}

	// Получаем кандидатов по первому токену
	candidates := r.tokenPatternIndex[tokens[0].Type]
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no handlers found for token type %s", tokens[0].Type)
	}

	// Проверяем паттерны для каждого кандидата
	var matchedHandlers []*HandlerReference

	for _, candidate := range candidates {
		if r.matchesTokenPattern(candidate.Config.TokenPatterns, tokens) {
			matchedHandlers = append(matchedHandlers, candidate)
		}
	}

	if len(matchedHandlers) == 0 {
		return nil, fmt.Errorf("no handlers matched token sequence for %v", tokens)
	}

	// Сортируем по приоритету
	sort.Slice(matchedHandlers, func(i, j int) bool {
		return matchedHandlers[i].Config.Priority > matchedHandlers[j].Config.Priority
	})

	return matchedHandlers[0].Handler, nil
}

// GetAllHandlersForTokenSequence - получает все обработчики для последовательности токенов, отсортированные по приоритету
func (r *ConstructHandlerRegistryImpl) GetAllHandlersForTokenSequence(
	tokens []lexer.Token,
) []common.Handler {
	if len(tokens) == 0 {
		return nil
	}

	// Получаем кандидатов по первому токену
	candidates := r.tokenPatternIndex[tokens[0].Type]
	if len(candidates) == 0 {
		return nil
	}

	// Проверяем паттерны для каждого кандидата
	var matchedHandlers []*HandlerReference

	for _, candidate := range candidates {
		if r.matchesTokenPattern(candidate.Config.TokenPatterns, tokens) {
			matchedHandlers = append(matchedHandlers, candidate)
		}
	}

	if len(matchedHandlers) == 0 {
		return nil
	}

	// Сортируем по приоритету
	sort.Slice(matchedHandlers, func(i, j int) bool {
		return matchedHandlers[i].Config.Priority > matchedHandlers[j].Config.Priority
	})

	// Возвращаем все обработчики в порядке приоритета
	handlers := make([]common.Handler, len(matchedHandlers))
	for i, ref := range matchedHandlers {
		handlers[i] = ref.Handler
	}

	return handlers
}

// matchesTokenPattern - проверяет соответствие токенов паттерну
// Возвращает true, если хотя бы один паттерн соответствует токенам (OR логика)
func (r *ConstructHandlerRegistryImpl) matchesTokenPattern(
	patterns []config.TokenPattern,
	tokens []lexer.Token,
) bool {
	for _, pattern := range patterns {
		if pattern.Offset >= len(tokens) {
			continue
		}

		token := tokens[pattern.Offset]
		if token.Type != pattern.TokenType {
			continue
		}

		if pattern.Value != "" && token.Value != pattern.Value {
			continue
		}

		return true
	}

	return false
}

// GetHandlerByConstruct - получает основной обработчик для типа конструкции (с наивысшим приоритетом)
func (r *ConstructHandlerRegistryImpl) GetHandlerByConstruct(
	constructType common.ConstructType,
) (common.Handler, error) {
	refs := r.handlersByConstruct[constructType]

	// Фильтруем только основные обработчики (не fallback)
	var nonFallbackRefs []*HandlerReference
	for _, ref := range refs {
		if !ref.Config.IsFallback {
			nonFallbackRefs = append(nonFallbackRefs, ref)
		}
	}

	if len(nonFallbackRefs) == 0 {
		return nil, fmt.Errorf("no primary handlers found for construct type: %s", constructType)
	}

	// Сортируем по приоритету (по убыванию)
	sort.Slice(nonFallbackRefs, func(i, j int) bool {
		return nonFallbackRefs[i].Config.Priority > nonFallbackRefs[j].Config.Priority
	})

	// Возвращаем обработчик с наивысшим приоритетом
	return nonFallbackRefs[0].Handler, nil
}

// GetFallbackHandlers - получает fallback-обработчики для типа конструкции
func (r *ConstructHandlerRegistryImpl) GetFallbackHandlers(
	constructType common.ConstructType,
) []common.Handler {
	refs := r.handlersByConstruct[constructType]

	// Создаем слайс обработчиков с их конфигурациями для сортировки
	type handlerWithPriority struct {
		handler  common.Handler
		priority int
	}

	handlerList := make([]handlerWithPriority, 0)

	for _, ref := range refs {
		if ref.Config.IsFallback {
			handlerList = append(handlerList, handlerWithPriority{
				handler:  ref.Handler,
				priority: ref.Config.FallbackPriority,
			})
		}
	}

	// Сортируем по FallbackPriority (по убыванию)
	sort.Slice(handlerList, func(i, j int) bool {
		return handlerList[i].priority > handlerList[j].priority
	})

	// Извлекаем отсортированные обработчики
	handlers := make([]common.Handler, len(handlerList))
	for i, hl := range handlerList {
		handlers[i] = hl.handler
	}

	return handlers
}

// sortHandlersByConstruct - сортирует обработчики для типа конструкции
func (r *ConstructHandlerRegistryImpl) sortHandlersByConstruct(
	constructType common.ConstructType,
) {
	handlers := r.handlersByConstruct[constructType]

	sort.Slice(handlers, func(i, j int) bool {
		configI, configJ := handlers[i].Config, handlers[j].Config

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
