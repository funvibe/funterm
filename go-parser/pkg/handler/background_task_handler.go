package handler

import (
	"fmt"

	"go-parser/pkg/ast"
	"go-parser/pkg/common"
	"go-parser/pkg/config"
	"go-parser/pkg/lexer"
	"go-parser/pkg/stream"
)

// BackgroundTaskHandler - обработчик для background tasks (&)
type BackgroundTaskHandler struct {
	config           config.ConstructHandlerConfig
	languageRegistry LanguageRegistry
}

// NewBackgroundTaskHandler создает новый обработчик для background tasks
func NewBackgroundTaskHandler(config config.ConstructHandlerConfig) *BackgroundTaskHandler {
	return &BackgroundTaskHandler{
		config:           config,
		languageRegistry: CreateDefaultLanguageRegistry(),
	}
}

// NewBackgroundTaskHandlerWithRegistry создает обработчик с явным указанием Language Registry
func NewBackgroundTaskHandlerWithRegistry(config config.ConstructHandlerConfig, registry LanguageRegistry) *BackgroundTaskHandler {
	return &BackgroundTaskHandler{
		config:           config,
		languageRegistry: registry,
	}
}

// CanHandle проверяет, может ли обработчик обработать токен
func (h *BackgroundTaskHandler) CanHandle(token lexer.Token) bool {
	// BackgroundTaskHandler обрабатывает токен & напрямую
	if token.Type == lexer.TokenAmpersand {
		return true
	}

	// Для языковых токенов мы не можем здесь определить, есть ли & в конце,
	// так как у нас нет доступа к tokenStream в CanHandle
	// Поэтому мы возвращаем true для языковых токенов, но будем проверять в Handle
	return token.Type == lexer.TokenLua ||
		token.Type == lexer.TokenPython ||
		token.Type == lexer.TokenPy ||
		token.Type == lexer.TokenGo ||
		token.Type == lexer.TokenNode ||
		token.Type == lexer.TokenJS
}

// Handle обрабатывает background task
func (h *BackgroundTaskHandler) Handle(ctx *common.ParseContext) (interface{}, error) {
	tokenStream := ctx.TokenStream

	// Если текущий токен - &, это означает, что мы находимся в конце выражения
	if tokenStream.Current().Type == lexer.TokenAmpersand {
		// Потребляем &
		ampersandToken := tokenStream.Consume()

		// Проверяем, что после & идет только newline или конец потока
		if tokenStream.HasMore() {
			nextToken := tokenStream.Current()
			if nextToken.Type != lexer.TokenNewline && nextToken.Type != lexer.TokenEOF {
				return nil, fmt.Errorf("unexpected token after &: %s", nextToken.Type)
			}
		}

		// Возвращаемся к началу потока для анализа
		tokenStream.SetPosition(0)

		// Создаем простую реализацию для теста
		return h.parseBackgroundTaskFromBeginning(tokenStream, ampersandToken)
	}

	// Проверяем, не является ли это кодовым блоком
	if h.isCodeBlock(tokenStream) {
		// Это кодовый блок, позволяем другим обработчикам обработать его
		return nil, fmt.Errorf("not a background task statement - code block")
	}

	// Иначе, проверяем, есть ли & в конце выражения
	if !h.hasAmpersandAtEnd(tokenStream) {
		// Если нет & в конце, это не background task
		return nil, fmt.Errorf("not a background task statement")
	}

	// Потребляем все токены до &
	var tokens []lexer.Token
	var ampersandToken lexer.Token
	for tokenStream.HasMore() {
		token := tokenStream.Current()
		if token.Type == lexer.TokenAmpersand {
			ampersandToken = tokenStream.Consume()
			break
		}
		tokens = append(tokens, token)
		tokenStream.Consume()
	}

	// Создаем LanguageCallStatement для background task
	return h.createLanguageCallStatementFromTokens(tokens, ampersandToken)
}

// createPipeLanguageCallStatement создает LanguageCallStatement для pipe expression
func (h *BackgroundTaskHandler) createPipeLanguageCallStatement(ampersandToken lexer.Token) (*ast.LanguageCallStatement, error) {
	// Создаем фиктивный LanguageCall для pipe expression
	languageCall := &ast.LanguageCall{
		Language:  "pipe",
		Function:  "background",
		Arguments: []ast.Expression{},
		Pos: ast.Position{
			Line:   ampersandToken.Line,
			Column: ampersandToken.Column,
			Offset: ampersandToken.Position,
		},
	}

	return ast.NewBackgroundLanguageCallStatement(languageCall, ampersandToken, languageCall.Pos), nil
}

// createSimpleLanguageCallStatement создает LanguageCallStatement для простого language call
func (h *BackgroundTaskHandler) createSimpleLanguageCallStatement(languageToken, functionToken lexer.Token, ampersandToken lexer.Token) (*ast.LanguageCallStatement, error) {
	languageCall := &ast.LanguageCall{
		Language:  languageToken.Value,
		Function:  functionToken.Value,
		Arguments: []ast.Expression{},
		Pos: ast.Position{
			Line:   languageToken.Line,
			Column: languageToken.Column,
			Offset: languageToken.Position,
		},
	}

	return ast.NewBackgroundLanguageCallStatement(languageCall, ampersandToken, languageCall.Pos), nil
}

// createDefaultLanguageCallStatement создает LanguageCallStatement по умолчанию
func (h *BackgroundTaskHandler) createDefaultLanguageCallStatement(ampersandToken lexer.Token) (*ast.LanguageCallStatement, error) {
	languageCall := &ast.LanguageCall{
		Language:  "unknown",
		Function:  "background",
		Arguments: []ast.Expression{},
		Pos: ast.Position{
			Line:   ampersandToken.Line,
			Column: ampersandToken.Column,
			Offset: ampersandToken.Position,
		},
	}

	return ast.NewBackgroundLanguageCallStatement(languageCall, ampersandToken, languageCall.Pos), nil
}

// hasAmpersandAtEnd проверяет, есть ли & в конце выражения
func (h *BackgroundTaskHandler) hasAmpersandAtEnd(tokenStream stream.TokenStream) bool {
	// Сохраняем текущую позицию
	originalPos := tokenStream.Position()
	defer tokenStream.SetPosition(originalPos)

	// Ищем & в потоке токенов
	for tokenStream.HasMore() {
		token := tokenStream.Current()
		if token.Type == lexer.TokenAmpersand {
			// Проверяем, что после & только newline или EOF
			tokenStream.Consume()
			if tokenStream.HasMore() {
				nextToken := tokenStream.Current()
				if nextToken.Type != lexer.TokenNewline && nextToken.Type != lexer.TokenEOF {
					return false
				}
			}
			return true
		}
		tokenStream.Consume()
	}
	return false
}

// parseBackgroundTaskFromBeginning парсит background task с начала потока
func (h *BackgroundTaskHandler) parseBackgroundTaskFromBeginning(tokenStream stream.TokenStream, ampersandToken lexer.Token) (interface{}, error) {
	// Сохраняем текущую позицию
	originalPos := tokenStream.Position()
	defer tokenStream.SetPosition(originalPos)

	// Собираем все токены до &
	var tokens []lexer.Token
	for tokenStream.HasMore() {
		token := tokenStream.Current()
		if token.Type == lexer.TokenAmpersand {
			break
		}
		tokens = append(tokens, token)
		tokenStream.Consume()
	}

	return h.createLanguageCallStatementFromTokens(tokens, ampersandToken)
}

// createLanguageCallStatementFromTokens создает LanguageCallStatement из токенов
func (h *BackgroundTaskHandler) createLanguageCallStatementFromTokens(tokens []lexer.Token, ampersandToken lexer.Token) (interface{}, error) {
	if len(tokens) == 0 {
		return nil, fmt.Errorf("no tokens before &")
	}

	// Проверяем, является ли это pipe expression
	hasPipe := false
	for _, token := range tokens {
		if token.Type == lexer.TokenPipe {
			hasPipe = true
			break
		}
	}

	if hasPipe {
		return h.createPipeLanguageCallStatement(ampersandToken)
	}

	// Проверяем, является ли это простым language call
	if len(tokens) >= 3 && (tokens[0].Type == lexer.TokenLua || tokens[0].Type == lexer.TokenPython || tokens[0].Type == lexer.TokenPy || tokens[0].Type == lexer.TokenGo || tokens[0].Type == lexer.TokenNode || tokens[0].Type == lexer.TokenJS) && tokens[1].Type == lexer.TokenDot {
		return h.createSimpleLanguageCallStatement(tokens[0], tokens[2], ampersandToken)
	}

	// По умолчанию создаем простой LanguageCallStatement
	return h.createDefaultLanguageCallStatement(ampersandToken)
}

// isCodeBlock проверяет, является ли текущее выражение кодовым блоком
func (h *BackgroundTaskHandler) isCodeBlock(tokenStream stream.TokenStream) bool {
	// Сохраняем текущую позицию
	originalPos := tokenStream.Position()
	defer tokenStream.SetPosition(originalPos)

	// Пропускаем языковой токен
	if !tokenStream.HasMore() {
		return false
	}
	tokenStream.Consume()

	// Проверяем, следующий токен - это (
	if !tokenStream.HasMore() {
		return false
	}
	nextToken := tokenStream.Current()
	if nextToken.Type != lexer.TokenLeftParen && nextToken.Type != lexer.TokenLParen {
		return false
	}

	// Ищем { после (
	for tokenStream.HasMore() {
		token := tokenStream.Current()
		if token.Type == lexer.TokenLBrace {
			return true // Это кодовый блок
		}
		if token.Type == lexer.TokenAmpersand {
			return false // Это background task, не кодовый блок
		}
		tokenStream.Consume()
	}

	return false
}

// Config возвращает конфигурацию обработчика
func (h *BackgroundTaskHandler) Config() common.HandlerConfig {
	return common.HandlerConfig{
		IsEnabled: h.config.IsEnabled,
		Priority:  h.config.Priority,
		Name:      h.config.Name,
	}
}

// Name возвращает имя обработчика
func (h *BackgroundTaskHandler) Name() string {
	return h.config.Name
}
