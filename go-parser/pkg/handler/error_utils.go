package handler

import (
	"fmt"
	"go-parser/pkg/ast"
	"go-parser/pkg/lexer"
	"go-parser/pkg/stream"
)

// newErrorWithPos creates a position-aware error using the current token's position
func newErrorWithPos(tokenStream stream.TokenStream, format string, args ...interface{}) error {
	if tokenStream.HasMore() {
		token := tokenStream.Current()
		pos := ast.Position{
			Line:   token.Line,
			Column: token.Column,
			Offset: token.Position,
		}
		return fmt.Errorf("%s at line %d, column %d", fmt.Sprintf(format, args...), pos.Line, pos.Column)
	}
	return fmt.Errorf(format, args...)
}

// newErrorWithTokenPos creates a position-aware error using a specific token's position
func newErrorWithTokenPos(token lexer.Token, format string, args ...interface{}) error {
	pos := ast.Position{
		Line:   token.Line,
		Column: token.Column,
		Offset: token.Position,
	}
	return fmt.Errorf("%s at line %d, column %d", fmt.Sprintf(format, args...), pos.Line, pos.Column)
}
