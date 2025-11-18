package handler

import (
	"fmt"
	"strings"

	"go-parser/pkg/common"
	"go-parser/pkg/lexer"
)

// LanguageHandler - обработчик для конкретного языка
type LanguageHandler struct {
	Language     string
	Constructs   map[common.ConstructType]common.Handler
	TokenMapping map[lexer.TokenType]common.ConstructType
	Priority     int
}

// LanguageRegistry - реестр обработчиков для разных языков
type LanguageRegistry interface {
	RegisterLanguage(language string, handler *LanguageHandler) error
	GetHandler(language string, token lexer.Token) (common.Handler, error)
	GetLanguageHandler(language string) (*LanguageHandler, error)
	GetSupportedLanguages() []string

	// Методы для работы с алиасами
	RegisterAlias(alias, language string) error
	ResolveAlias(alias string) (string, error)
	GetAliases(language string) ([]string, error)
	IsLanguageSupported(language string) bool
	GetSupportedLanguagesWithAliases() map[string][]string
}

// LanguageRegistryImpl - реализация реестра языков
type LanguageRegistryImpl struct {
	languages map[string]*LanguageHandler
	aliases   map[string]string // Алиас -> полный язык
	priority  []string          // Языки в порядке приоритета
}

// NewLanguageRegistry создает новый реестр языков
func NewLanguageRegistry() *LanguageRegistryImpl {
	return &LanguageRegistryImpl{
		languages: make(map[string]*LanguageHandler),
		aliases:   make(map[string]string),
		priority:  make([]string, 0),
	}
}

// RegisterLanguage регистрирует обработчик для языка
func (r *LanguageRegistryImpl) RegisterLanguage(language string, handler *LanguageHandler) error {
	language = strings.ToLower(language)

	if _, exists := r.languages[language]; exists {
		return fmt.Errorf("language '%s' already registered", language)
	}

	r.languages[language] = handler

	// Добавляем в список приоритетов, если еще нет
	found := false
	for _, lang := range r.priority {
		if lang == language {
			found = true
			break
		}
	}
	if !found {
		r.priority = append(r.priority, language)
	}

	return nil
}

// GetHandler получает обработчик для языка и токена
func (r *LanguageRegistryImpl) GetHandler(language string, token lexer.Token) (common.Handler, error) {
	language = strings.ToLower(language)

	// Разрешаем алиас в полное имя языка
	resolvedLanguage, err := r.ResolveAlias(language)
	if err != nil {
		return nil, err
	}

	langHandler, err := r.GetLanguageHandler(resolvedLanguage)
	if err != nil {
		return nil, err
	}

	// Ищем конструкцию по типу токена
	constructType, exists := langHandler.TokenMapping[token.Type]
	if !exists {
		return nil, fmt.Errorf("no construct mapping for token type %s in language %s", token.Type, language)
	}

	// Получаем обработчик для конструкции
	handler, exists := langHandler.Constructs[constructType]
	if !exists {
		return nil, fmt.Errorf("no handler for construct type %s in language %s", constructType, language)
	}

	return handler, nil
}

// GetLanguageHandler получает обработчик языка
func (r *LanguageRegistryImpl) GetLanguageHandler(language string) (*LanguageHandler, error) {
	language = strings.ToLower(language)

	// Разрешаем алиас в полное имя языка
	resolvedLanguage, err := r.ResolveAlias(language)
	if err != nil {
		return nil, err
	}

	handler, exists := r.languages[resolvedLanguage]
	if !exists {
		return nil, fmt.Errorf("language '%s' not registered", resolvedLanguage)
	}

	return handler, nil
}

// GetSupportedLanguages возвращает список поддерживаемых языков
func (r *LanguageRegistryImpl) GetSupportedLanguages() []string {
	result := make([]string, len(r.priority))
	copy(result, r.priority)
	return result
}

// RegisterAlias регистрирует алиас для языка
func (r *LanguageRegistryImpl) RegisterAlias(alias, language string) error {
	alias = strings.ToLower(alias)
	language = strings.ToLower(language)

	// Проверяем, что язык существует
	if _, exists := r.languages[language]; !exists {
		return fmt.Errorf("language '%s' not registered", language)
	}

	// Проверяем, что алиас не занят
	if _, exists := r.aliases[alias]; exists {
		return fmt.Errorf("alias '%s' already registered", alias)
	}

	// Проверяем, что алиас не совпадает с именем языка
	if _, exists := r.languages[alias]; exists {
		return fmt.Errorf("alias '%s' conflicts with existing language name", alias)
	}

	r.aliases[alias] = language
	return nil
}

// ResolveAlias разрешает алиас в полное имя языка
func (r *LanguageRegistryImpl) ResolveAlias(alias string) (string, error) {
	alias = strings.ToLower(alias)

	// Если это не алиас, а полное имя языка - возвращаем как есть
	if _, exists := r.languages[alias]; exists {
		return alias, nil
	}

	// Ищем алиас
	language, exists := r.aliases[alias]
	if !exists {
		return "", fmt.Errorf("alias '%s' not found", alias)
	}

	return language, nil
}

// GetAliases возвращает все алиасы для языка
func (r *LanguageRegistryImpl) GetAliases(language string) ([]string, error) {
	language = strings.ToLower(language)

	// Проверяем, что язык существует
	if _, exists := r.languages[language]; !exists {
		return nil, fmt.Errorf("language '%s' not registered", language)
	}

	var aliases []string
	for alias, lang := range r.aliases {
		if lang == language {
			aliases = append(aliases, alias)
		}
	}

	return aliases, nil
}

// IsLanguageSupported проверяет, поддерживается ли язык (включая алиасы)
func (r *LanguageRegistryImpl) IsLanguageSupported(language string) bool {
	language = strings.ToLower(language)

	// Проверяем как полный язык
	if _, exists := r.languages[language]; exists {
		return true
	}

	// Проверяем как алиас
	if _, exists := r.aliases[language]; exists {
		return true
	}

	return false
}

// GetSupportedLanguagesWithAliases возвращает список поддерживаемых языков и их алиасов
func (r *LanguageRegistryImpl) GetSupportedLanguagesWithAliases() map[string][]string {
	result := make(map[string][]string)

	// Добавляем все основные языки
	for lang := range r.languages {
		aliases, _ := r.GetAliases(lang)
		result[lang] = aliases
	}

	return result
}

// CreateDefaultLanguageRegistry создает реестр с обработчиками по умолчанию
func CreateDefaultLanguageRegistry() LanguageRegistry {
	registry := NewLanguageRegistry()

	// Регистрируем обработчик для Lua
	luaHandler := &LanguageHandler{
		Language: "lua",
		Constructs: map[common.ConstructType]common.Handler{
			common.ConstructArray:      NewArrayHandler(10, 1),
			common.ConstructObject:     NewObjectHandler(10, 1),
			common.ConstructAssignment: NewAssignmentHandler(5, 1),
		},
		TokenMapping: map[lexer.TokenType]common.ConstructType{
			lexer.TokenLBracket:   common.ConstructArray,
			lexer.TokenLBrace:     common.ConstructObject,
			lexer.TokenIdentifier: common.ConstructAssignment,
		},
		Priority: 100,
	}
	registry.RegisterLanguage("lua", luaHandler)

	// Регистрируем обработчик для Python
	pythonHandler := &LanguageHandler{
		Language: "python",
		Constructs: map[common.ConstructType]common.Handler{
			common.ConstructArray:      NewArrayHandler(10, 1),
			common.ConstructObject:     NewObjectHandler(10, 1),
			common.ConstructAssignment: NewAssignmentHandler(5, 1),
		},
		TokenMapping: map[lexer.TokenType]common.ConstructType{
			lexer.TokenLBracket:   common.ConstructArray,
			lexer.TokenLBrace:     common.ConstructObject,
			lexer.TokenIdentifier: common.ConstructAssignment,
		},
		Priority: 90,
	}
	registry.RegisterLanguage("python", pythonHandler)

	// Регистрируем обработчик для Go
	goHandler := &LanguageHandler{
		Language: "go",
		Constructs: map[common.ConstructType]common.Handler{
			common.ConstructArray:      NewArrayHandler(10, 1),
			common.ConstructObject:     NewObjectHandler(10, 1),
			common.ConstructAssignment: NewAssignmentHandler(5, 1),
		},
		TokenMapping: map[lexer.TokenType]common.ConstructType{
			lexer.TokenLBracket:   common.ConstructArray,
			lexer.TokenLBrace:     common.ConstructObject,
			lexer.TokenIdentifier: common.ConstructAssignment,
		},
		Priority: 80,
	}
	registry.RegisterLanguage("go", goHandler)

	// Регистрируем обработчик для Node.js
	nodeHandler := &LanguageHandler{
		Language: "node",
		Constructs: map[common.ConstructType]common.Handler{
			common.ConstructArray:      NewArrayHandler(10, 1),
			common.ConstructObject:     NewObjectHandler(10, 1),
			common.ConstructAssignment: NewAssignmentHandler(5, 1),
		},
		TokenMapping: map[lexer.TokenType]common.ConstructType{
			lexer.TokenLBracket:   common.ConstructArray,
			lexer.TokenLBrace:     common.ConstructObject,
			lexer.TokenIdentifier: common.ConstructAssignment,
		},
		Priority: 70,
	}
	registry.RegisterLanguage("node", nodeHandler)

	// Регистрируем стандартные алиасы
	registry.RegisterAlias("py", "python")
	registry.RegisterAlias("js", "node")
	registry.RegisterAlias("l", "lua")

	return registry
}

// LanguageAwareHandlerRegistry - комбинированный реестр, учитывающий язык
type LanguageAwareHandlerRegistry struct {
	languageRegistry  LanguageRegistry
	constructRegistry *ConstructHandlerRegistryImpl
	currentLanguage   string
}

// NewLanguageAwareHandlerRegistry создает комбинированный реестр
func NewLanguageAwareHandlerRegistry() *LanguageAwareHandlerRegistry {
	return &LanguageAwareHandlerRegistry{
		languageRegistry:  CreateDefaultLanguageRegistry(),
		constructRegistry: NewConstructHandlerRegistry(),
		currentLanguage:   "lua", // Язык по умолчанию
	}
}

// SetLanguage устанавливает текущий язык
func (r *LanguageAwareHandlerRegistry) SetLanguage(language string) error {
	language = strings.ToLower(language)

	supported := r.languageRegistry.GetSupportedLanguages()
	found := false
	for _, lang := range supported {
		if lang == language {
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("language '%s' is not supported. Supported languages: %v", language, supported)
	}

	r.currentLanguage = language
	return nil
}

// GetCurrentLanguage возвращает текущий язык
func (r *LanguageAwareHandlerRegistry) GetCurrentLanguage() string {
	return r.currentLanguage
}

// GetHandlerForToken получает обработчик для токена с учетом текущего языка
func (r *LanguageAwareHandlerRegistry) GetHandlerForToken(token lexer.Token) (common.Handler, error) {
	// Сначала пробуем получить обработчик из языкового реестра
	handler, err := r.languageRegistry.GetHandler(r.currentLanguage, token)
	if err == nil {
		return handler, nil
	}

	// Если не нашли, пробуем получить из реестра конструкций
	tokens := []lexer.Token{token}
	return r.constructRegistry.GetHandlerForTokenSequence(tokens)
}

// RegisterHandlerForLanguage регистрирует обработчик для конкретного языка
func (r *LanguageAwareHandlerRegistry) RegisterHandlerForLanguage(
	language string,
	constructType common.ConstructType,
	handler common.Handler,
	tokenTypes []lexer.TokenType,
) error {
	language = strings.ToLower(language)

	langHandler, err := r.languageRegistry.GetLanguageHandler(language)
	if err != nil {
		return fmt.Errorf("language '%s' not registered", language)
	}

	// Добавляем обработчик
	langHandler.Constructs[constructType] = handler

	// Добавляем маппинг токенов
	for _, tokenType := range tokenTypes {
		langHandler.TokenMapping[tokenType] = constructType
	}

	return nil
}
