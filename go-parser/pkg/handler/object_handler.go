package handler

import (
	"fmt"

	"go-parser/pkg/ast"
	"go-parser/pkg/common"
	"go-parser/pkg/lexer"
)

// ObjectHandler - обработчик объектов
type ObjectHandler struct {
	config common.HandlerConfig
}

// NewObjectHandler создает новый обработчик объектов
func NewObjectHandler(priority, order int) *ObjectHandler {
	config := DefaultConfig("object")
	config.Priority = priority
	config.Order = order
	return &ObjectHandler{
		config: config,
	}
}

// CanHandle проверяет, может ли обработчик обработать токен
func (h *ObjectHandler) CanHandle(token lexer.Token) bool {
	return token.Type == lexer.TokenLBrace
}

// Handle обрабатывает объект
func (h *ObjectHandler) Handle(ctx *common.ParseContext) (interface{}, error) {
	// Проверяем защиту от рекурсии
	if err := ctx.Guard.Enter(); err != nil {
		return nil, err
	}
	defer ctx.Guard.Exit()

	// Потребляем открывающую фигурную скобку
	openBrace := ctx.TokenStream.Consume()
	if openBrace.Type != lexer.TokenLBrace {
		return nil, fmt.Errorf("expected '{', got %s", openBrace.Type)
	}

	// Создаем узел объекта
	objectNode := ast.NewObjectLiteral(openBrace, lexer.Token{})

	// Обрабатываем свойства до закрывающей скобки
	for ctx.TokenStream.HasMore() {
		current := ctx.TokenStream.Current()

		if current.Type == lexer.TokenRBrace {
			// Потребляем закрывающую скобку и завершаем
			closeBrace := ctx.TokenStream.Consume()
			objectNode.RightBrace = closeBrace
			return objectNode, nil
		}

		// Пропускаем запятые между свойствами
		if current.Type == lexer.TokenComma {
			ctx.TokenStream.Consume()
			// После запятой проверяем, не является ли следующий токен закрывающей скобкой
			// Если да, то это висячая запятая, которую мы просто игнорируем
			if ctx.TokenStream.HasMore() && ctx.TokenStream.Current().Type == lexer.TokenRBrace {
				continue
			}
			continue
		}

		// Обрабатываем ключ (пока только идентификаторы или строки)
		keyToken := ctx.TokenStream.Consume()
		var key ast.Expression

		switch keyToken.Type {
		case lexer.TokenString:
			key = &ast.StringLiteral{
				Value: keyToken.Value,
				Pos: ast.Position{
					Line:   keyToken.Line,
					Column: keyToken.Column,
					Offset: keyToken.Position,
				},
			}
		case lexer.TokenIdentifier:
			key = ast.NewIdentifier(keyToken, keyToken.Value)
		default:
			// Неизвестный тип токена для ключа, пропускаем
			continue
		}

		// Проверяем наличие двоеточия
		if ctx.TokenStream.Current().Type != lexer.TokenColon {
			return nil, fmt.Errorf("expected ':' after key, got %s", ctx.TokenStream.Current().Type)
		}
		ctx.TokenStream.Consume() // Потребляем двоеточие

		// Обрабатываем значение - может быть сложным выражением
		var value ast.Expression
		var err error

		// Используем AssignmentHandler для парсинга сложных выражений
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

		value, err = assignmentHandler.parseComplexExpression(assignmentCtx)
		if err != nil {
			return nil, fmt.Errorf("failed to parse object value: %v", err)
		}

		if key != nil && value != nil {
			objectNode.AddProperty(key, value)
		}
	}

	// Если дошли сюда, значит не нашли закрывающую скобку
	return nil, fmt.Errorf("unclosed object")
}

// Config возвращает конфигурацию обработчика
func (h *ObjectHandler) Config() common.HandlerConfig {
	return h.config
}

// Name возвращает имя обработчика
func (h *ObjectHandler) Name() string {
	return h.config.Name
}
