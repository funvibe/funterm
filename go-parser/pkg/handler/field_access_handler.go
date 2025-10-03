package handler

import (
	"fmt"

	"go-parser/pkg/ast"
	"go-parser/pkg/common"
	"go-parser/pkg/config"
	"go-parser/pkg/lexer"
)

// FieldAccessHandler - обработчик для доступа к полям объектов (например, lua.x)
type FieldAccessHandler struct {
	config  config.ConstructHandlerConfig
	verbose bool
}

// NewFieldAccessHandler создает новый обработчик для доступа к полям
func NewFieldAccessHandler(config config.ConstructHandlerConfig) *FieldAccessHandler {
	return NewFieldAccessHandlerWithVerbose(config, false)
}

// NewFieldAccessHandlerWithVerbose создает новый обработчик для доступа к полям с поддержкой verbose режима
func NewFieldAccessHandlerWithVerbose(config config.ConstructHandlerConfig, verbose bool) *FieldAccessHandler {
	return &FieldAccessHandler{
		config:  config,
		verbose: verbose,
	}
}

// CanHandle проверяет, может ли обработчик обработать токен
func (h *FieldAccessHandler) CanHandle(token lexer.Token) bool {
	// Обрабатываем идентификаторы и языковые токены, за которыми следует DOT
	return token.Type == lexer.TokenIdentifier ||
		token.Type == lexer.TokenLua ||
		token.Type == lexer.TokenPython ||
		token.Type == lexer.TokenPy ||
		token.Type == lexer.TokenGo ||
		token.Type == lexer.TokenNode ||
		token.Type == lexer.TokenJS
}

// Handle обрабатывает доступ к полю с поддержкой цепочек (например, lua.data.name)
func (h *FieldAccessHandler) Handle(ctx *common.ParseContext) (interface{}, error) {
	tokenStream := ctx.TokenStream

	// 1. Читаем первый идентификатор (например, lua)
	firstToken := tokenStream.Consume()
	if firstToken.Type != lexer.TokenIdentifier &&
		firstToken.Type != lexer.TokenLua &&
		firstToken.Type != lexer.TokenPython &&
		firstToken.Type != lexer.TokenPy &&
		firstToken.Type != lexer.TokenGo &&
		firstToken.Type != lexer.TokenNode &&
		firstToken.Type != lexer.TokenJS {
		return nil, fmt.Errorf("expected identifier as first part of field access, got %s", firstToken.Type)
	}

	if h.verbose {
		fmt.Printf("DEBUG: FieldAccessHandler - first token: %s (%s)\n", firstToken.Value, firstToken.Type)
	}

	// Начинаем строить цепочку с первого идентификатора
	var currentObject ast.Expression
	// Проверяем, является ли первый токен языковым токеном
	if firstToken.Type == lexer.TokenLua || firstToken.Type == lexer.TokenPython ||
		firstToken.Type == lexer.TokenPy || firstToken.Type == lexer.TokenGo ||
		firstToken.Type == lexer.TokenJS || firstToken.Type == lexer.TokenNode {
		// Создаем квалифицированный идентификатор для языкового токена
		language := firstToken.Value
		if language == "js" {
			language = "node" // Нормализуем js в node
		}
		currentObject = ast.NewIdentifier(firstToken, firstToken.Value)
		// Устанавливаем информацию о языке вручную, так как NewQualifiedIdentifier требует два токена
		if ident, ok := currentObject.(*ast.Identifier); ok {
			ident.Language = language
			ident.Qualified = true
		}
		if h.verbose {
			fmt.Printf("DEBUG: FieldAccessHandler - created qualified identifier with language: %s\n", language)
		}
	} else {
		// Создаем обычный идентификатор
		currentObject = ast.NewIdentifier(firstToken, firstToken.Value)
	}

	// 2. Обрабатываем цепочку полей (например, .data.name)
	fieldCount := 0
	for tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenDot {
		// Потребляем DOT
		tokenStream.Consume()

		// Читаем имя поля - поддерживаем идентификаторы и булевы литералы
		if !tokenStream.HasMore() ||
			(tokenStream.Current().Type != lexer.TokenIdentifier &&
				tokenStream.Current().Type != lexer.TokenTrue &&
				tokenStream.Current().Type != lexer.TokenFalse) {
			return nil, fmt.Errorf("expected field name after DOT")
		}
		fieldToken := tokenStream.Consume()
		fieldCount++

		if h.verbose {
			fmt.Printf("DEBUG: FieldAccessHandler - processing field: %s\n", fieldToken.Value)
		}

		// Создаем узел FieldAccess для текущего уровня
		fieldAccess := ast.NewFieldAccess(currentObject, fieldToken.Value, ast.Position{
			Line:   firstToken.Line,
			Column: firstToken.Column,
			Offset: firstToken.Position,
		})

		// Текущий FieldAccess становится объектом для следующего уровня (если есть)
		currentObject = fieldAccess
	}

	if h.verbose {
		fmt.Printf("DEBUG: FieldAccessHandler - processed %d fields, returning: %T\n", fieldCount, currentObject)
		if tokenStream.HasMore() {
			fmt.Printf("DEBUG: FieldAccessHandler - next token: %s (%s)\n", tokenStream.Current().Value, tokenStream.Current().Type)
		} else {
			fmt.Printf("DEBUG: FieldAccessHandler - no more tokens\n")
		}
	}

	return currentObject, nil
}

// Config возвращает конфигурацию обработчика
func (h *FieldAccessHandler) Config() common.HandlerConfig {
	return common.HandlerConfig{
		IsEnabled: h.config.IsEnabled,
		Priority:  h.config.Priority,
		Name:      h.config.Name,
	}
}

// Name возвращает имя обработчика
func (h *FieldAccessHandler) Name() string {
	return h.config.Name
}
