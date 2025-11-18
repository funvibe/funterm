package parser

import (
	"fmt"

	"go-parser/pkg/ast"
	"go-parser/pkg/common"
	"go-parser/pkg/handler"
	"go-parser/pkg/lexer"
	"go-parser/pkg/stream"
)

type recursionGuard struct {
	maxDepth     int
	currentDepth int
}

func newRecursionGuard(maxDepth int) *recursionGuard {
	return &recursionGuard{
		maxDepth:     maxDepth,
		currentDepth: 0,
	}
}

func (g *recursionGuard) Enter() error {
	g.currentDepth++
	if g.currentDepth > g.maxDepth {
		return fmt.Errorf("recursion depth limit exceeded: %d", g.maxDepth)
	}
	return nil
}

func (g *recursionGuard) Exit() {
	g.currentDepth--
}

func (g *recursionGuard) CurrentDepth() int {
	return g.currentDepth
}

func (g *recursionGuard) MaxDepth() int {
	return g.maxDepth
}

type RecursiveParser struct {
	registry *handler.ConstructHandlerRegistryImpl
	maxDepth int
	guard    *recursionGuard
}

func NewRecursiveParser(registry *handler.ConstructHandlerRegistryImpl) *RecursiveParser {
	return &RecursiveParser{
		registry: registry,
		maxDepth: 1000, // значение по умолчанию
		guard:    newRecursionGuard(1000),
	}
}

// NewRecursiveParserWithLegacyRegistry - конструктор для совместимости со старым реестром
func NewRecursiveParserWithLegacyRegistry(registry *handler.SimpleHandlerRegistry) *RecursiveParser {
	// Создаем заглушку для нового интерфейса
	// TODO: Реализовать адаптер между старым и новым реестром
	return &RecursiveParser{
		registry: nil, // Заглушка
		maxDepth: 1000,
		guard:    newRecursionGuard(1000),
	}
}

func (p *RecursiveParser) Parse(input string) (*ParseResult, error) {
	lex := lexer.NewLexer(input)
	tokenStream := stream.NewTokenStream(lex)
	return p.ParseTokens(tokenStream)
}

func (p *RecursiveParser) ParseTokens(stream stream.TokenStream) (*ParseResult, error) {
	if !stream.HasMore() {
		return &ParseResult{
			Value:          nil,
			Error:          nil,
			TokensConsumed: 0,
		}, nil
	}

	// Обновляем guard с текущим maxDepth
	p.guard.maxDepth = p.maxDepth

	// Создаем контекст парсинга
	ctx := &common.ParseContext{
		TokenStream: stream,
		Parser:      p,
		Depth:       p.guard.CurrentDepth(),
		MaxDepth:    p.maxDepth,
		Guard:       p.guard,
	}

	// Получаем текущий токен для определения конструкции
	currentToken := stream.Current()

	// Если токен - EOF или RIGHT_PAREN (после обработки скобок), завершаем
	if currentToken.Type == lexer.TokenEOF || currentToken.Type == lexer.TokenRightParen {
		return &ParseResult{
			Value:          nil,
			Error:          nil,
			TokensConsumed: 0,
		}, nil
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
			return &ParseResult{
				Value:          nil,
				Error:          nil,
				TokensConsumed: 1,
			}, nil
		}
	}

	// Проверяем, может ли обработчик обработать этот токен
	if !handler.CanHandle(currentToken) {
		stream.Consume()
		return &ParseResult{
			Value:          nil,
			Error:          nil,
			TokensConsumed: 1,
		}, nil
	}

	// Вызываем обработчик
	startPos := stream.Position()
	result, err := handler.Handle(ctx)
	if err != nil {
		return &ParseResult{
			Value:          nil,
			Error:          err,
			TokensConsumed: 0,
		}, err
	}

	tokensConsumed := stream.Position() - startPos

	// Создаем ProgramNode если еще не создан
	var rootNode ast.Node
	if result != nil {
		if astNode, ok := result.(ast.Node); ok {
			rootNode = astNode
		} else {
			// Для обратной совместимости
			rootNode = &ast.ParenthesesNode{} // заглушка
		}
	}

	// Если есть еще токены, создаем ProgramNode
	if stream.HasMore() {
		program := ast.NewProgramNode()
		if rootNode != nil {
			program.AddNode(rootNode)
		}

		// Обрабатываем оставшиеся токены
		for stream.HasMore() {
			currentToken := stream.Current()

			// Если токен - EOF, завершаем
			if currentToken.Type == lexer.TokenEOF {
				break
			}

			// Получаем обработчик для текущего токена
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
					stream.Consume()
					continue
				}
			}

			if !handler.CanHandle(currentToken) {
				stream.Consume()
				continue
			}

			startPos := stream.Position()
			childResult, err := handler.Handle(ctx)
			if err != nil {
				return &ParseResult{
					Value:          nil,
					Error:          err,
					TokensConsumed: tokensConsumed,
				}, err
			}

			if childResult != nil {
				if childNode, ok := childResult.(ast.Node); ok {
					program.AddNode(childNode)
				}
			}

			tokensConsumed += (stream.Position() - startPos)
		}

		rootNode = program
	}

	// Преобразуем узел в map для обратной совместимости с тестами
	var value interface{}
	if rootNode != nil {
		value = rootNode.ToMap()
	}

	return &ParseResult{
		Value:          value,
		Error:          nil,
		TokensConsumed: tokensConsumed,
	}, nil
}

func (p *RecursiveParser) SetMaxDepth(depth int) {
	p.maxDepth = depth
	p.guard.maxDepth = depth
}

func (p *RecursiveParser) GetMaxDepth() int {
	return p.maxDepth
}

// peekTokens считывает несколько токенов без потребления
func (p *RecursiveParser) peekTokens(stream stream.TokenStream, count int) []lexer.Token {
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
