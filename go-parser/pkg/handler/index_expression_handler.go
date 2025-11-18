package handler

import (
	"go-parser/pkg/ast"
	"go-parser/pkg/common"
	"go-parser/pkg/config"
	"go-parser/pkg/lexer"
)

// IndexExpressionHandler - обработчик для индексированного доступа (например, dict["key"] или arr[0])
type IndexExpressionHandler struct {
	config config.ConstructHandlerConfig
}

// NewIndexExpressionHandler создает новый обработчик для индексированного доступа
func NewIndexExpressionHandler(config config.ConstructHandlerConfig) *IndexExpressionHandler {
	return &IndexExpressionHandler{
		config: config,
	}
}

// CanHandle проверяет, может ли обработчик обработать токен
func (h *IndexExpressionHandler) CanHandle(token lexer.Token) bool {
	// Обрабатываем идентификаторы, за которыми следует LEFT_BRACKET
	return token.Type == lexer.TokenIdentifier
}

// Handle обрабатывает индексированный доступ
func (h *IndexExpressionHandler) Handle(ctx *common.ParseContext) (interface{}, error) {
	tokenStream := ctx.TokenStream

	// 1. Читаем объект (например, dict)
	objectToken := tokenStream.Current()
	if objectToken.Type != lexer.TokenIdentifier {
		return nil, newErrorWithTokenPos(objectToken, "expected identifier for object, got %s", objectToken.Type)
	}
	tokenStream.Consume()
	object := ast.NewIdentifier(objectToken, objectToken.Value)

	// 2. Проверяем и потребляем LEFT_BRACKET
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenLBracket {
		return nil, newErrorWithPos(tokenStream, "expected LEFT_BRACKET after object '%s'", objectToken.Value)
	}
	tokenStream.Consume() // потребляем LEFT_BRACKET

	// 3. Парсим индекс (может быть любое выражение)
	if !tokenStream.HasMore() {
		return nil, newErrorWithPos(tokenStream, "expected index expression after LEFT_BRACKET")
	}

	// Простая обработка индекса - пока только строковые литералы и числа
	indexToken := tokenStream.Current()
	var indexExpr ast.Expression

	switch indexToken.Type {
	case lexer.TokenString:
		tokenStream.Consume()
		indexExpr = &ast.StringLiteral{
			Value: indexToken.Value,
			Pos: ast.Position{
				Line:   indexToken.Line,
				Column: indexToken.Column,
				Offset: indexToken.Position,
			},
		}
	case lexer.TokenNumber:
		tokenStream.Consume()
		// Создаем числовой литерал - конвертируем строку в число
		numValue, err := parseNumber(indexToken.Value)
		if err != nil {
			return nil, newErrorWithTokenPos(indexToken, "invalid number literal: %s", indexToken.Value)
		}
		indexExpr = createNumberLiteral(indexToken, numValue)
	case lexer.TokenIdentifier:
		tokenStream.Consume()
		indexExpr = ast.NewIdentifier(indexToken, indexToken.Value)
	default:
		return nil, newErrorWithTokenPos(indexToken, "unsupported index type: %s", indexToken.Type)
	}

	// 4. Проверяем и потребляем RIGHT_BRACKET
	if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenRBracket {
		return nil, newErrorWithPos(tokenStream, "expected RIGHT_BRACKET after index expression")
	}
	tokenStream.Consume() // потребляем RIGHT_BRACKET

	// 5. Создаем узел AST
	indexExpression := ast.NewIndexExpression(object, indexExpr, ast.Position{
		Line:   objectToken.Line,
		Column: objectToken.Column,
		Offset: objectToken.Position,
	})

	return indexExpression, nil
}

// Config возвращает конфигурацию обработчика
func (h *IndexExpressionHandler) Config() common.HandlerConfig {
	return common.HandlerConfig{
		IsEnabled: h.config.IsEnabled,
		Priority:  h.config.Priority,
		Name:      h.config.Name,
	}
}

// Name возвращает имя обработчика
func (h *IndexExpressionHandler) Name() string {
	return h.config.Name
}
