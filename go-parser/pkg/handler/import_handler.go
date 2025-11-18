package handler

import (
	"go-parser/pkg/ast"
	"go-parser/pkg/common"
	"go-parser/pkg/config"
	"go-parser/pkg/lexer"
)

// ImportHandler - обработчик для импорта внешних файлов
type ImportHandler struct {
	config config.ConstructHandlerConfig
}

// NewImportHandler создает новый обработчик импорта
func NewImportHandler(config config.ConstructHandlerConfig) *ImportHandler {
	return &ImportHandler{
		config: config,
	}
}

// CanHandle проверяет, может ли обработчик обработать токен
func (h *ImportHandler) CanHandle(token lexer.Token) bool {
	return token.Type == lexer.TokenImport
}

// Handle обрабатывает импорт
func (h *ImportHandler) Handle(ctx *common.ParseContext) (interface{}, error) {
	tokenStream := ctx.TokenStream

	// Потребляем токен import
	importToken := tokenStream.Current()
	if importToken.Type != lexer.TokenImport {
		return nil, newErrorWithTokenPos(importToken, "expected import token, got %s", importToken.Type)
	}
	tokenStream.Consume()

	// Пропускаем пробелы после import
	if !tokenStream.HasMore() {
		return nil, newErrorWithPos(tokenStream, "unexpected EOF after import")
	}
	current := tokenStream.Current()

	// Ожидаем один из рантаймов: lua, python, py
	var runtimeToken lexer.Token
	switch current.Type {
	case lexer.TokenLua, lexer.TokenPython, lexer.TokenPy, lexer.TokenNode, lexer.TokenJS:
		runtimeToken = current
		tokenStream.Consume()
	default:
		return nil, newErrorWithTokenPos(current, "expected runtime specifier (lua, python, py, node) after import, got %s", current.Type)
	}

	// Пропускаем пробелы после рантайма
	if !tokenStream.HasMore() {
		return nil, newErrorWithPos(tokenStream, "unexpected EOF after runtime specifier")
	}
	current = tokenStream.Current()

	// Ожидаем строковый литерал с путем к файлу
	if current.Type != lexer.TokenString {
		return nil, newErrorWithTokenPos(current, "expected string literal with file path after runtime, got %s", current.Type)
	}

	// Создаем строковый литерал для пути
	pathLiteral := &ast.StringLiteral{
		Value: current.Value,
		Pos: ast.Position{
			Line:   current.Line,
			Column: current.Column,
			Offset: current.Position,
		},
	}
	tokenStream.Consume()

	// Создаем узел импорта
	importStmt := ast.NewImportStatement(importToken, runtimeToken, pathLiteral)

	return importStmt, nil
}

// Config возвращает конфигурацию обработчика
func (h *ImportHandler) Config() common.HandlerConfig {
	return common.HandlerConfig{
		IsEnabled: h.config.IsEnabled,
		Priority:  h.config.Priority,
		Name:      h.config.Name,
	}
}

// Name возвращает имя обработчика
func (h *ImportHandler) Name() string {
	return h.config.Name
}
