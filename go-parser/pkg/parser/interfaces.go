package parser

import (
	"go-parser/pkg/common"
)

// Parser реализует интерфейс common.Parser
type Parser interface {
	common.Parser
}

// ParseResult - алиас для common.ParseResult
type ParseResult = common.ParseResult

// ParseContext - алиас для common.ParseContext
type ParseContext = common.ParseContext

// RecursionGuard - алиас для common.RecursionGuard
