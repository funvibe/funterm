package lexer

import "fmt"

type TokenType int

const (
	TokenEOF        TokenType = iota
	TokenLeftParen            // (
	TokenRightParen           // )
	TokenUnknown
	TokenIdentifier // IDENTIFIER
	TokenString     // STRING
	TokenNumber     // NUMBER
	TokenDot        // DOT
	TokenLParen     // LPAREN (синоним TokenLeftParen)
	TokenRParen     // RPAREN (синоним TokenRightParen)
	TokenComma      // COMMA
	TokenSemicolon  // SEMICOLON
	// Новые токены для Этапа 2
	TokenLBracket    // [
	TokenRBracket    // ]
	TokenLBrace      // {
	TokenRBrace      // }
	TokenAssign      // =
	TokenColonEquals // :=
	TokenColon       // :
	// Новые токены для Этапа 3 - циклы
	TokenFor      // for
	TokenIn       // in
	TokenWhile    // while
	TokenBreak    // break
	TokenContinue // continue
	// Новые токены для pattern matching
	TokenMatch // match
	TokenArrow // ->
	TokenRest  // ...
	// Новые токены для битовых строк
	TokenDoubleLeftAngle  // <<
	TokenDoubleRightAngle // >>
	TokenSlash            // /
	// Новые токены для pipes и background tasks
	TokenPipe      // |>
	TokenBitwiseOr // |
	TokenAmpersand // &
	TokenCaret     // ^
	// Новые токены для операторов сравнения
	TokenLess         // <
	TokenGreater      // >
	TokenLessEqual    // <=
	TokenGreaterEqual // >=
	TokenEqual        // ==
	TokenNotEqual     // !=
	// Новые токены для if/else
	TokenIf   // if
	TokenElse // else
	// Новые токены для арифметических операторов
	TokenPlus     // +
	TokenMinus    // -
	TokenMultiply // *
	TokenModulo   // %
	TokenConcat   // ++
	TokenPower    // **
	// Новые токены для логических операторов
	TokenAnd   // &&
	TokenOr    // ||
	TokenNot   // !
	TokenTilde // ~
	// Новые токены для булевых литералов
	TokenTrue  // true
	TokenFalse // false
	TokenNil   // nil
	// Новые токены для Elvis оператора
	TokenQuestion // ?
	// Новые токены для зарезервированных ключевых слов (Task 25)
	TokenLua    // lua
	TokenPython // python
	TokenPy     // py
	TokenGo     // go
	TokenNode   // node
	TokenJS     // js
	TokenImport // import
	// Новые токены для multiline строк и block comments
	TokenTripleDoubleQuote // """
	TokenTripleSingleQuote // '''
	TokenBlockCommentStart // /*
	TokenBlockCommentEnd   // */
	// Новые токены для обработки переносов строк в кодовых блоках
	TokenNewline // \n
	// Новые токены для wildcard паттернов
	TokenUnderscore // _
	// Новые токены для размера битстринга
	TokenAt // @
)

func (t TokenType) String() string {
	switch t {
	case TokenEOF:
		return "EOF"
	case TokenLeftParen:
		return "LEFT_PAREN"
	case TokenRightParen:
		return "RIGHT_PAREN"
	case TokenIdentifier:
		return "IDENTIFIER"
	case TokenString:
		return "STRING"
	case TokenNumber:
		return "NUMBER"
	case TokenDot:
		return "DOT"
	case TokenLParen:
		return "LPAREN"
	case TokenRParen:
		return "RPAREN"
	case TokenComma:
		return "COMMA"
	case TokenSemicolon:
		return "SEMICOLON"
	case TokenLBracket:
		return "LBRACKET"
	case TokenRBracket:
		return "RBRACKET"
	case TokenLBrace:
		return "LBRACE"
	case TokenRBrace:
		return "RBRACE"
	case TokenAssign:
		return "ASSIGN"
	case TokenColonEquals:
		return "COLON_EQUALS"
	case TokenColon:
		return "COLON"
	case TokenFor:
		return "FOR"
	case TokenIn:
		return "IN"
	case TokenWhile:
		return "WHILE"
	case TokenBreak:
		return "BREAK"
	case TokenContinue:
		return "CONTINUE"
	case TokenMatch:
		return "MATCH"
	case TokenArrow:
		return "ARROW"
	case TokenRest:
		return "REST"
	case TokenDoubleLeftAngle:
		return "DOUBLE_LEFT_ANGLE"
	case TokenDoubleRightAngle:
		return "DOUBLE_RIGHT_ANGLE"
	case TokenSlash:
		return "SLASH"
	case TokenPipe:
		return "PIPE"
	case TokenBitwiseOr:
		return "BITWISE_OR"
	case TokenAmpersand:
		return "AMPERSAND"
	case TokenCaret:
		return "CARET"
	case TokenLess:
		return "LESS"
	case TokenGreater:
		return "GREATER"
	case TokenLessEqual:
		return "LESS_EQUAL"
	case TokenGreaterEqual:
		return "GREATER_EQUAL"
	case TokenEqual:
		return "EQUAL"
	case TokenNotEqual:
		return "NOT_EQUAL"
	case TokenIf:
		return "IF"
	case TokenElse:
		return "ELSE"
	case TokenPlus:
		return "PLUS"
	case TokenMinus:
		return "MINUS"
	case TokenMultiply:
		return "MULTIPLY"
	case TokenModulo:
		return "MODULO"
	case TokenConcat:
		return "CONCAT"
	case TokenPower:
		return "POWER"
	case TokenAnd:
		return "AND"
	case TokenOr:
		return "OR"
	case TokenNot:
		return "NOT"
	case TokenTilde:
		return "TILDE"
	case TokenTrue:
		return "TRUE"
	case TokenFalse:
		return "FALSE"
	case TokenNil:
		return "NIL"
	case TokenQuestion:
		return "QUESTION"
	case TokenLua:
		return "LUA"
	case TokenPython:
		return "PYTHON"
	case TokenPy:
		return "PY"
	case TokenGo:
		return "GO"
	case TokenNode:
		return "NODE"
	case TokenJS:
		return "JS"
	case TokenImport:
		return "IMPORT"
	case TokenTripleDoubleQuote:
		return "TRIPLE_DOUBLE_QUOTE"
	case TokenTripleSingleQuote:
		return "TRIPLE_SINGLE_QUOTE"
	case TokenBlockCommentStart:
		return "BLOCK_COMMENT_START"
	case TokenBlockCommentEnd:
		return "BLOCK_COMMENT_END"
	case TokenNewline:
		return "NEWLINE"
	case TokenUnderscore:
		return "UNDERSCORE"
	case TokenAt:
		return "AT"
	default:
		return "UNKNOWN"
	}
}

type Token struct {
	Type     TokenType
	Value    string
	Position int
	Line     int
	Column   int
}

func (t Token) String() string {
	return fmt.Sprintf("Token{Type: %s, Value: %q, Pos: %d, Line: %d, Col: %d}",
		t.Type, t.Value, t.Position, t.Line, t.Column)
}

// IsLanguageToken проверяет, является ли токен токеном языка
func (t Token) IsLanguageToken() bool {
	return t.Type == TokenPython ||
		t.Type == TokenPy ||
		t.Type == TokenLua ||
		t.Type == TokenGo ||
		t.Type == TokenNode ||
		t.Type == TokenJS
}

// LanguageTokenToString преобразует токен языка в строковое представление
func (t Token) LanguageTokenToString() string {
	switch t.Type {
	case TokenPython, TokenPy:
		return "python"
	case TokenLua:
		return "lua"
	case TokenGo:
		return "go"
	case TokenNode, TokenJS:
		return "node"
	default:
		return ""
	}
}

// IsLanguageIdentifierOrCallToken проверяет, является ли токен идентификатором или токеном языка
func (t Token) IsLanguageIdentifierOrCallToken() bool {
	result := t.Type == TokenIdentifier || t.IsLanguageToken()
	// Debug: print result for TokenFor
	if t.Type == TokenFor {
		fmt.Printf("DEBUG: IsLanguageIdentifierOrCallToken for TokenFor: %v (TokenIdentifier: %v, IsLanguageToken: %v)\n", result, t.Type == TokenIdentifier, t.IsLanguageToken())
	}
	return result
}
