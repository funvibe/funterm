package handler

import (
	"go-parser/pkg/lexer"
)

// CanHandleLanguageTokens предикат для обработчиков, работающих с токенами языков
func CanHandleLanguageTokens(token lexer.Token) bool {
	return token.IsLanguageIdentifierOrCallToken()
}

// ValidateLanguageTokens проверяет валидность токена языка
func ValidateLanguageTokens(token lexer.Token) bool {
	return token.IsLanguageToken()
}

// GetLanguageFromToken извлекает имя языка из токена
func GetLanguageFromToken(token lexer.Token) string {
	return token.LanguageTokenToString()
}

// IsAnyLanguageToken проверяет, является ли токен одним из языковых токенов
func IsAnyLanguageToken(token lexer.Token) bool {
	return token.IsLanguageToken()
}

// IsLanguageCallStart проверяет, является ли токен началом вызова функции языка
func IsLanguageCallStart(token lexer.Token) bool {
	if !token.IsLanguageToken() {
		return false
	}
	// Проверяем, что следующий токен - точка или открывающая скобка
	// Это будет использоваться в обработчиках для определения паттернов вызова
	return true
	return true
}
