package common

import (
	"go-parser/pkg/lexer"
	"go-parser/pkg/stream"
)

type Handler interface {
	CanHandle(token lexer.Token) bool
	Handle(ctx *ParseContext) (interface{}, error)
	Config() HandlerConfig
	Name() string
}

type HandlerConfig struct {
	IsEnabled        bool
	Priority         int
	Order            int
	FallbackPriority int
	IsFallback       bool
	Name             string
}

type Parser interface {
	Parse(input string) (*ParseResult, error)
	ParseTokens(stream stream.TokenStream) (*ParseResult, error)
	SetMaxDepth(depth int)
	GetMaxDepth() int
}

type RecursionGuard interface {
	Enter() error
	Exit()
	CurrentDepth() int
	MaxDepth() int
}

type ParseContext struct {
	TokenStream        stream.TokenStream
	Lexer              lexer.Lexer // Сохраняем ссылку на лексер для управления состоянием
	Parser             Parser
	Depth              int
	MaxDepth           int
	Guard              RecursionGuard
	PartialParsingMode bool
	LoopDepth          int    // Глубина вложенности циклов для контекстной валидации break/continue
	InputStream        string // Оригинальный исходный код для извлечения сырых блоков
}

type ParseResult struct {
	Value          interface{}
	Error          error
	TokensConsumed int
}
