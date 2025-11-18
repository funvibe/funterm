package config

import (
	"go-parser/pkg/common"
	"go-parser/pkg/lexer"
)

// GetDefaultHandlerConfigs returns the default configuration for all handlers
func GetDefaultHandlerConfigs() []ConstructHandlerConfig {
	return []ConstructHandlerConfig{
		// Группирующие конструкции (скобки) - основной обработчик для текущей реализации
		{
			ConstructType:    common.ConstructGroup,
			Name:             "parentheses-group",
			Priority:         100,
			Order:            1,
			IsEnabled:        true,
			IsFallback:       false,
			FallbackPriority: 0,
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

		// Заглушки для будущих типов конструкций (отключены)
		{
			ConstructType:    common.ConstructLiteral,
			Name:             "number-literal",
			Priority:         90,
			Order:            1,
			IsEnabled:        false, // Отключено, пока не реализован лексер для чисел
			IsFallback:       false,
			FallbackPriority: 0,
			TokenPatterns: []TokenPattern{
				{
					TokenType: lexer.TokenUnknown, // Будет заменено на TokenNumber
					Offset:    0,
				},
			},
			CustomParams: map[string]interface{}{
				"supportsDecimal": true,
				"supportsHex":     true,
				"supportsBinary":  false,
			},
		},
		{
			ConstructType:    common.ConstructLiteral,
			Name:             "string-literal",
			Priority:         90,
			Order:            2,
			IsEnabled:        false, // Отключено, пока не реализован лексер для строк
			IsFallback:       false,
			FallbackPriority: 0,
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
			ConstructType:    common.ConstructVariable,
			Name:             "identifier",
			Priority:         80,
			Order:            1,
			IsEnabled:        false, // Отключено, пока не реализован лексер для идентификаторов
			IsFallback:       false,
			FallbackPriority: 0,
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
			ConstructType:    common.ConstructFunction,
			Name:             "function-definition",
			Priority:         60,
			Order:            1,
			IsEnabled:        false, // Отключено, пока не реализован лексер для ключевых слов
			IsFallback:       false,
			FallbackPriority: 0,
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
			ConstructType:    common.ConstructFunction,
			Name:             "function-call",
			Priority:         60,
			Order:            2,
			IsEnabled:        false, // Отключено, пока не реализован лексер для идентификаторов
			IsFallback:       false,
			FallbackPriority: 0,
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
			ConstructType:    common.ConstructMatch,
			Name:             "pattern-matching",
			Priority:         85,
			Order:            1,
			IsEnabled:        true,
			IsFallback:       false,
			FallbackPriority: 0,
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
}

// CreateDefaultParserConfig creates a default parser configuration
func CreateDefaultParserConfig() *ParserConfig {
	return &ParserConfig{
		MaxDepth:                100,
		EnableRecursionGuard:    true,
		DefaultFallbackPriority: 10,
		ConstructHandlers:       GetDefaultHandlerConfigs(),
		CustomSettings: map[string]interface{}{
			"enableDebugMode": false,
			"logLevel":        "info",
		},
	}
}

// CreateMinimalParserConfig creates a minimal parser configuration with only the essential handlers
func CreateMinimalParserConfig() *ParserConfig {
	return &ParserConfig{
		MaxDepth:                10,
		EnableRecursionGuard:    true,
		DefaultFallbackPriority: 5,
		ConstructHandlers: []ConstructHandlerConfig{
			{
				ConstructType:    common.ConstructGroup,
				Name:             "parentheses-group",
				Priority:         100,
				Order:            1,
				IsEnabled:        true,
				IsFallback:       false,
				FallbackPriority: 0,
				TokenPatterns: []TokenPattern{
					{
						TokenType: lexer.TokenLeftParen,
						Offset:    0,
					},
				},
				CustomParams: map[string]interface{}{
					"maxDepth": 10,
				},
			},
		},
		CustomSettings: map[string]interface{}{
			"enableDebugMode": false,
			"logLevel":        "error",
		},
	}
}

// CreateDevelopmentParserConfig creates a development parser configuration with debug features
func CreateDevelopmentParserConfig() *ParserConfig {
	return &ParserConfig{
		MaxDepth:                1000,
		EnableRecursionGuard:    true,
		DefaultFallbackPriority: 100,
		ConstructHandlers:       GetDefaultHandlerConfigs(),
		CustomSettings: map[string]interface{}{
			"enableDebugMode": true,
			"logLevel":        "debug",
			"detailedErrors":  true,
			"performanceLog":  true,
		},
	}
}
