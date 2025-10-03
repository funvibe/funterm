package handler

import (
	"fmt"
	"go-parser/pkg/common"
	"go-parser/pkg/config"
	"go-parser/pkg/lexer"
)

// ReservedKeywordHandler - обработчик для предотвращения использования зарезервированных ключевых слов
type ReservedKeywordHandler struct {
	config config.ConstructHandlerConfig
}

// NewReservedKeywordHandler создает новый обработчик зарезервированных ключевых слов
func NewReservedKeywordHandler(config config.ConstructHandlerConfig) *ReservedKeywordHandler {
	return &ReservedKeywordHandler{
		config: config,
	}
}

// CanHandle проверяет, может ли обработчик обработать токен
func (h *ReservedKeywordHandler) CanHandle(token lexer.Token) bool {
	// Проверяем, является ли токен зарезервированным ключевым словом
	return token.Type == lexer.TokenLua || token.Type == lexer.TokenPython || token.Type == lexer.TokenPy || token.Type == lexer.TokenGo || token.Type == lexer.TokenNode || token.Type == lexer.TokenJS
}

// Handle обрабатывает попытку использования зарезервированного слова
func (h *ReservedKeywordHandler) Handle(ctx *common.ParseContext) (interface{}, error) {
	tokenStream := ctx.TokenStream

	// Получаем текущий токен (зарезервированное слово)
	reservedToken := tokenStream.Current()

	// Проверяем, следующий токен - это оператор присваивания?
	if tokenStream.Peek().Type == lexer.TokenAssign {
		// Это попытка присвоить значение зарезервированному слову - ошибка!
		return nil, fmt.Errorf("cannot assign to reserved keyword '%s'", reservedToken.Value)
	}

	// Проверяем, следующий токен - это точка (для доступа к функциям рантайма, например lua.math.sin)?
	if tokenStream.Peek().Type == lexer.TokenDot {
		// Это легальное использование ключевого слова для доступа к функциям рантайма
		// Возвращаем специальную ошибку, которая позволит другим обработчикам попробовать.
		return nil, fmt.Errorf("not a reserved keyword assignment")
	}

	// Проверяем, следующий токен - это открывающая фигурная скобка (для блоков кода)?
	if tokenStream.Peek().Type == lexer.TokenLBrace {
		// Это легальное использование ключевого слова для блоков кода
		// Возвраяем специальную ошибку, которая позволит другим обработчикам попробовать.
		return nil, fmt.Errorf("not a reserved keyword assignment")
	}

	// Проверяем, текущий токен - это import, а следующий - зарезервированное слово?
	// Это нужно для обработки import lua "file.lua"
	if reservedToken.Type == lexer.TokenImport {
		nextToken := tokenStream.Peek()
		if nextToken.Type == lexer.TokenLua || nextToken.Type == lexer.TokenPython || nextToken.Type == lexer.TokenPy || nextToken.Type == lexer.TokenGo || nextToken.Type == lexer.TokenNode || nextToken.Type == lexer.TokenJS {
			// Это легальное использование в импорте
			return nil, fmt.Errorf("not a reserved keyword assignment")
		}
	}

	// Если это ни одна из легальных конструкций, то это попытка использовать ключевое слово в недопустимом контексте
	return nil, fmt.Errorf("invalid use of reserved keyword '%s'", reservedToken.Value)
}

// Config возвращает конфигурацию обработчика
func (h *ReservedKeywordHandler) Config() common.HandlerConfig {
	return common.HandlerConfig{
		IsEnabled: h.config.IsEnabled,
		Priority:  h.config.Priority,
		Name:      h.config.Name,
	}
}

// Name возвращает имя обработчика
func (h *ReservedKeywordHandler) Name() string {
	return h.config.Name
}
