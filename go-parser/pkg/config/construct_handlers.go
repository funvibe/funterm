package config

import (
	"go-parser/pkg/common"
	"go-parser/pkg/lexer"
)

// ConstructHandlerConfig - конфигурация обработчика конструкции
type ConstructHandlerConfig struct {
	// Основные параметры
	ConstructType    common.ConstructType `json:"constructType" yaml:"constructType"`
	Name             string               `json:"name" yaml:"name"`
	Priority         int                  `json:"priority" yaml:"priority"`
	Order            int                  `json:"order" yaml:"order"`
	IsEnabled        bool                 `json:"isEnabled" yaml:"isEnabled"`
	IsFallback       bool                 `json:"isFallback" yaml:"isFallback"`
	FallbackPriority int                  `json:"fallbackPriority" yaml:"fallbackPriority"`

	// Паттерны токенов для идентификации конструкции
	TokenPatterns []TokenPattern `json:"tokenPatterns" yaml:"tokenPatterns"`

	// Специфичные параметры для типа конструкции
	CustomParams map[string]interface{} `json:"customParams" yaml:"customParams"`
}

// TokenPattern - паттерн токена для идентификации конструкции
type TokenPattern struct {
	TokenType lexer.TokenType `json:"tokenType" yaml:"tokenType"`
	Value     string          `json:"value,omitempty" yaml:"value,omitempty"` // Опциональное значение
	Offset    int             `json:"offset" yaml:"offset"`                   // Смещение от начала конструкции
}

// HandlersConfig - централизованная конфигурация всех обработчиков
var HandlersConfig = []ConstructHandlerConfig{
	// Группирующие конструкции (скобки) - основной обработчик для текущей реализации
	{
		ConstructType: common.ConstructGroup,
		Name:          "parentheses-group",
		Priority:      100,
		Order:         1,
		IsEnabled:     true,
		TokenPatterns: []TokenPattern{
			{
				TokenType: lexer.TokenLeftParen,
				Offset:    0,
			},
		},
		CustomParams: map[string]interface{}{
			"maxDepth": 100,
		},
	},

	// Number literals (включено)
	{
		ConstructType: common.ConstructLiteral,
		Name:          "number-literal",
		Priority:      90,
		Order:         1,
		IsEnabled:     true, // Включено для поддержки чисел
		TokenPatterns: []TokenPattern{
			{
				TokenType: lexer.TokenNumber, // Используем TokenNumber
				Offset:    0,
			},
		},
		CustomParams: map[string]interface{}{
			"supportsDecimal": true,
			"supportsHex":     true,
			"supportsBinary":  true,
		},
	},
	{
		ConstructType: common.ConstructLiteral,
		Name:          "string-literal",
		Priority:      90,
		Order:         2,
		IsEnabled:     false, // Отключено, пока не реализован лексер для строк
		TokenPatterns: []TokenPattern{
			{
				TokenType: lexer.TokenUnknown, // Будет заменено на TokenString
				Offset:    0,
			},
		},
		CustomParams: map[string]interface{}{
			"supportsEscape": true,
			"multiline":      false,
		},
	},

	// Переменные (отключены, пока не реализован лексер для идентификаторов)
	{
		ConstructType: common.ConstructVariable,
		Name:          "identifier",
		Priority:      80,
		Order:         1,
		IsEnabled:     false, // Отключено, пока не реализован лексер для идентификаторов
		TokenPatterns: []TokenPattern{
			{
				TokenType: lexer.TokenUnknown, // Будет заменено на TokenIdentifier
				Offset:    0,
			},
		},
		CustomParams: map[string]interface{}{
			"pattern": "^[a-zA-Z_][a-zA-Z0-9_]*$",
		},
	},

	// Функции (отключены, пока не реализован лексер для ключевых слов)
	{
		ConstructType: common.ConstructFunction,
		Name:          "function-definition",
		Priority:      60,
		Order:         1,
		IsEnabled:     false, // Отключено, пока не реализован лексер для ключевых слов
		TokenPatterns: []TokenPattern{
			{
				TokenType: lexer.TokenUnknown, // Будет заменено на TokenKeyword
				Value:     "func",
				Offset:    0,
			},
		},
		CustomParams: map[string]interface{}{
			"supportsAnonymous": true,
			"supportsParams":    true,
			"supportsReturn":    true,
		},
	},
	{
		ConstructType: common.ConstructFunction,
		Name:          "function-call",
		Priority:      60,
		Order:         2,
		IsEnabled:     false, // Отключено, пока не реализован лексер для идентификаторов
		TokenPatterns: []TokenPattern{
			{
				TokenType: lexer.TokenUnknown, // Будет заменено на TokenIdentifier
				Offset:    0,
			},
			{
				TokenType: lexer.TokenLeftParen,
				Offset:    1,
			},
		},
		CustomParams: map[string]interface{}{
			"maxParams": 10,
		},
	},

	// Pattern matching (включен для поддержки match конструкций)
	{
		ConstructType: common.ConstructMatch,
		Name:          "pattern-matching",
		Priority:      85,
		Order:         1,
		IsEnabled:     true,
		TokenPatterns: []TokenPattern{
			{
				TokenType: lexer.TokenMatch,
				Offset:    0,
			},
		},
		CustomParams: map[string]interface{}{
			"supportsArrayPatterns":    true,
			"supportsObjectPatterns":   true,
			"supportsVariablePatterns": true,
			"supportsWildcardPatterns": true,
		},
	},
}
