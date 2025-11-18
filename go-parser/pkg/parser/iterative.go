package parser

import (
	"go-parser/pkg/ast"
	"go-parser/pkg/common"
	"go-parser/pkg/handler"
	"go-parser/pkg/lexer"
	"go-parser/pkg/stream"
)

type CleanIterativeParser struct {
	registry *handler.ConstructHandlerRegistryImpl
	maxDepth int
	guard    *recursionGuard
}

// NewCleanIterativeParser - создает новый чистый итеративный парсер
func NewCleanIterativeParser(registry *handler.ConstructHandlerRegistryImpl) *CleanIterativeParser {
	return &CleanIterativeParser{
		registry: registry,
		maxDepth: 1000, // значение по умолчанию
		guard:    newRecursionGuard(1000),
	}
}

func (p *CleanIterativeParser) Parse(input string) (*ParseResult, error) {
	lex := lexer.NewLexer(input)
	tokenStream := stream.NewTokenStream(lex)
	return p.ParseTokens(tokenStream)
}

func (p *CleanIterativeParser) ParseTokens(stream stream.TokenStream) (*ParseResult, error) {
	var results []interface{}
	tokensConsumed := 0

	// Обновляем guard с текущим maxDepth
	p.guard.maxDepth = p.maxDepth

	for stream.HasMore() {
		currentToken := stream.Current()

		// Если токен - EOF, завершаем
		if currentToken.Type == lexer.TokenEOF {
			break
		}

		// Получаем обработчик для текущего токена (создаем простую последовательность из одного токена)
		tokens := []lexer.Token{currentToken}
		handler, err := p.registry.GetHandlerForTokenSequence(tokens)
		if err != nil {
			// Пробуем fallback обработчики
			for _, constructType := range []common.ConstructType{
				common.ConstructLiteral,
				common.ConstructVariable,
				common.ConstructFunction,
				common.ConstructGroup,
			} {
				fallbackHandlers := p.registry.GetFallbackHandlers(constructType)
				for _, h := range fallbackHandlers {
					if h.CanHandle(currentToken) {
						handler = h
						err = nil
						break
					}
				}
				if err == nil {
					break
				}
			}

			if err != nil {
				// Пропускаем неизвестный токен
				stream.Consume()
				tokensConsumed++
				continue
			}
		}

		// Проверяем, может ли обработчик обработать этот токен
		if !handler.CanHandle(currentToken) {
			stream.Consume()
			tokensConsumed++
			continue
		}

		// Создаем контекст для итеративного парсинга
		ctx := &common.ParseContext{
			TokenStream: stream,
			Parser:      p,
			Depth:       p.guard.CurrentDepth(),
			MaxDepth:    p.maxDepth,
			Guard:       p.guard,
		}

		// Вызываем обработчик
		startPos := stream.Position()
		result, err := handler.Handle(ctx)
		if err != nil {
			return &ParseResult{
				Value:          nil,
				Error:          err,
				TokensConsumed: tokensConsumed,
			}, err
		}

		if result != nil {
			results = append(results, result)
		}

		tokensConsumed += (stream.Position() - startPos)
	}

	// Преобразуем результаты в map для обратной совместимости с тестами
	var value interface{}
	if len(results) == 1 {
		// Если только один результат, преобразуем его в map
		if node, ok := results[0].(ast.Node); ok {
			value = node.ToMap()
		} else {
			value = results[0]
		}
	} else if len(results) > 1 {
		// Иначе преобразуем все результаты в map и возвращаем массив
		mappedResults := make([]interface{}, len(results))
		for i, result := range results {
			if node, ok := result.(ast.Node); ok {
				mappedResults[i] = node.ToMap()
			} else {
				mappedResults[i] = result
			}
		}
		value = mappedResults
	}

	return &ParseResult{
		Value:          value,
		Error:          nil,
		TokensConsumed: tokensConsumed,
	}, nil
}

// peekTokens считывает несколько токенов без потребления
func (p *CleanIterativeParser) peekTokens(stream stream.TokenStream, count int) []lexer.Token {
	var tokens []lexer.Token

	// Сохраняем текущую позицию
	currentPos := stream.Position()

	// Читаем токены без потребления
	for i := 0; i < count && stream.HasMore(); i++ {
		tokens = append(tokens, stream.PeekN(i+1))
	}

	// Восстанавливаем позицию
	stream.SetPosition(currentPos)

	return tokens
}

func (p *CleanIterativeParser) SetMaxDepth(depth int) {
	p.maxDepth = depth
	p.guard.maxDepth = depth
}

func (p *CleanIterativeParser) GetMaxDepth() int {
	return p.maxDepth
}

// Для совместимости со старым кодом, оставляем псевдоним
type IterativeParser = CleanIterativeParser

// Для совместимости со старым кодом, оставляем псевдоним конструктора
func NewIterativeParser(registry *handler.ConstructHandlerRegistryImpl) *CleanIterativeParser {
	return NewCleanIterativeParser(registry)
}
