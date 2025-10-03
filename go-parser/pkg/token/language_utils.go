package token

import (
	"go-parser/pkg/config"
	"go-parser/pkg/lexer"
)

// LanguageToken представляет перечисление языков
type LanguageToken int

const (
	LanguageNone LanguageToken = iota
	LanguagePython
	LanguageLua
	LanguageGo
	LanguageNode
)

// IsLanguageToken проверяет, является ли токен токеном языка
func IsLanguageToken(tokenType lexer.TokenType) bool {
	return tokenType == lexer.TokenPython ||
		tokenType == lexer.TokenPy ||
		tokenType == lexer.TokenLua ||
		tokenType == lexer.TokenGo ||
		tokenType == lexer.TokenNode ||
		tokenType == lexer.TokenJS
}

// LanguageTokenToString преобразует токен языка в строковое представление
func LanguageTokenToString(tokenType lexer.TokenType) string {
	switch tokenType {
	case lexer.TokenPython, lexer.TokenPy:
		return "python"
	case lexer.TokenLua:
		return "lua"
	case lexer.TokenGo:
		return "go"
	case lexer.TokenNode, lexer.TokenJS:
		return "node"
	default:
		return ""
	}
}

// GetAllLanguageTokens возвращает все токены языков
func GetAllLanguageTokens() []lexer.TokenType {
	return []lexer.TokenType{
		lexer.TokenPython, lexer.TokenPy, lexer.TokenLua, lexer.TokenGo, lexer.TokenNode, lexer.TokenJS,
	}
}

// GetAllLanguageTokenPatterns возвращает все паттерны токенов для конфигурации
func GetAllLanguageTokenPatterns() []config.TokenPattern {
	tokens := GetAllLanguageTokens()
	patterns := make([]config.TokenPattern, len(tokens))
	for i, token := range tokens {
		patterns[i] = config.TokenPattern{TokenType: token, Offset: 0}
	}
	return patterns
}

// IsLanguageIdentifierOrCallToken проверяет, является ли токен идентификатором или токеном языка
func IsLanguageIdentifierOrCallToken(token lexer.Token) bool {
	return token.Type == lexer.TokenIdentifier || IsLanguageToken(token.Type)
}

// GetLanguageTokenFromName возвращает токен языка по строковому имени
func GetLanguageTokenFromName(name string) lexer.TokenType {
	switch name {
	case "python":
		return lexer.TokenPython
	case "lua":
		return lexer.TokenLua
	case "go":
		return lexer.TokenGo
	case "node", "js":
		return lexer.TokenNode
	default:
		return lexer.TokenIdentifier
	}
}
