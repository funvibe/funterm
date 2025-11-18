package handler

import (
	"go-parser/pkg/ast"
	"go-parser/pkg/common"
	"go-parser/pkg/lexer"
)

// ArrayHandler - обработчик массивов
type ArrayHandler struct {
	config common.HandlerConfig
}

// NewArrayHandler создает новый обработчик массивов
func NewArrayHandler(priority, order int) *ArrayHandler {
	config := DefaultConfig("array")
	config.Priority = priority
	config.Order = order
	return &ArrayHandler{
		config: config,
	}
}

// CanHandle проверяет, может ли обработчик обработать токен
func (h *ArrayHandler) CanHandle(token lexer.Token) bool {
	return token.Type == lexer.TokenLBracket
}

// Handle обрабатывает массив
func (h *ArrayHandler) Handle(ctx *common.ParseContext) (interface{}, error) {
	// Проверяем защиту от рекурсии
	if err := ctx.Guard.Enter(); err != nil {
		return nil, err
	}
	defer ctx.Guard.Exit()

	// Потребляем открывающую скобку
	openBracket := ctx.TokenStream.Consume()
	if openBracket.Type != lexer.TokenLBracket {
		return nil, newErrorWithTokenPos(openBracket, "expected '[', got %s", openBracket.Type)
	}

	// Создаем узел массива
	arrayNode := ast.NewArrayLiteral(openBracket, lexer.Token{})

	// Обрабатываем элементы до закрывающей скобки
	for ctx.TokenStream.HasMore() {
		current := ctx.TokenStream.Current()

		if current.Type == lexer.TokenRBracket {
			// Потребляем закрывающую скобку и завершаем
			closeBracket := ctx.TokenStream.Consume()
			arrayNode.RightBracket = closeBracket
			return arrayNode, nil
		}

		// Пропускаем запятые между элементами
		if current.Type == lexer.TokenComma {
			ctx.TokenStream.Consume()
			// После запятой проверяем, не является ли следующий токен закрывающей скобкой
			// Если да, то это висячая запятая, которую мы просто игнорируем
			if ctx.TokenStream.HasMore() && ctx.TokenStream.Current().Type == lexer.TokenRBracket {
				continue
			}
			continue
		}

		// Используем AssignmentHandler для парсинга сложных выражений в элементах массива
		assignmentHandler := NewAssignmentHandler(100, 0)
		assignmentCtx := &common.ParseContext{
			TokenStream: ctx.TokenStream,
			Parser:      nil,
			Depth:       ctx.Depth + 1,
			MaxDepth:    ctx.MaxDepth,
			Guard:       ctx.Guard,
			LoopDepth:   ctx.LoopDepth,
			InputStream: ctx.InputStream,
		}

		element, err := assignmentHandler.parseComplexExpression(assignmentCtx)
		if err != nil {
			return nil, newErrorWithPos(ctx.TokenStream, "failed to parse array element: %v", err)
		}

		if element != nil {
			arrayNode.AddElement(element)
		}
	}

	// Если дошли сюда, значит не нашли закрывающую скобку
	return nil, newErrorWithPos(ctx.TokenStream, "unclosed array")
}

// Config возвращает конфигурацию обработчика
func (h *ArrayHandler) Config() common.HandlerConfig {
	return h.config
}

// Name возвращает имя обработчика
func (h *ArrayHandler) Name() string {
	return h.config.Name
}
