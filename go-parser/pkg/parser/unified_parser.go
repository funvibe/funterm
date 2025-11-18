package parser

import (
	"fmt"
	"strings"
	"go-parser/pkg/ast"
	"go-parser/pkg/common"
	"go-parser/pkg/config"
	exprparser "go-parser/pkg/expression"
	"go-parser/pkg/handler"
	"go-parser/pkg/lexer"
	"go-parser/pkg/stream"
)

// UnifiedParser - реализация парсера по ТЗ
type UnifiedParser struct {
	lexer    *lexer.Lexer
	registry *handler.ConstructHandlerRegistryImpl
	verbose  bool
}

// ProtoParser - интерфейс парсера по ТЗ (не конфликтует с existing Parser)
type ProtoParser interface {
	Parse(input string) (ast.Statement, []ast.ParseError)
}

// NewUnifiedParser создает новый парсер
func NewUnifiedParser() *UnifiedParser {
	return NewUnifiedParserWithVerbose(false)
}

// NewUnifiedParserWithRegistry создает новый парсер с указанным реестром языков
func NewUnifiedParserWithRegistry(registry handler.LanguageRegistry, verbose bool) *UnifiedParser {
	p := NewUnifiedParserWithVerbose(verbose)
	// Тут нужно будет заменить реестр в обработчиках, но это сложно.
	// Проще будет передавать реестр в конструкторы обработчиков.
	// Пока что оставим так, это потребует большого рефакторинга.
	return p
}

// NewUnifiedParserWithVerbose создает новый парсер с указанием verbose режима
func NewUnifiedParserWithVerbose(verbose bool) *UnifiedParser {
	// Создаем реестр и регистрируем обработчики
	registry := handler.NewConstructHandlerRegistry()

	// Регистрируем ParenthesizedExpression обработчик для выражений в скобках
	// Должен иметь самый высокий приоритет для скобок
	parenthesizedConfig := config.ConstructHandlerConfig{
		ConstructType: common.ConstructGroup,
		Name:          "parenthesized-expression",
		Priority:      120, // Самый высокий приоритет для скобок
		Order:         0,
		IsEnabled:     true,
		IsFallback:    false,
		TokenPatterns: []config.TokenPattern{
			{TokenType: lexer.TokenLeftParen, Offset: 0},
		},
	}

	parenthesizedHandler := handler.NewParenthesizedExpressionHandler(parenthesizedConfig)
	registry.RegisterConstructHandler(parenthesizedHandler, parenthesizedConfig)

	// Регистрируем ReservedKeyword обработчик (Task 25)
	// Должен иметь абсолютный самый высокий приоритет для проверки зарезервированных слов
	reservedKeywordConfig := config.ConstructHandlerConfig{
		ConstructType: common.ConstructVariable, // Используем существующий тип
		Name:          "reserved-keyword-guard",
		Priority:      160, // Абсолютный самый высокий приоритет
		Order:         1,
		IsEnabled:     true,
		IsFallback:    false,
		TokenPatterns: []config.TokenPattern{
			{TokenType: lexer.TokenLua, Offset: 0},
			{TokenType: lexer.TokenPython, Offset: 0},
			{TokenType: lexer.TokenPy, Offset: 0},
			{TokenType: lexer.TokenGo, Offset: 0},
			{TokenType: lexer.TokenNode, Offset: 0},
			{TokenType: lexer.TokenJS, Offset: 0},
		},
	}

	reservedKeywordHandler := handler.NewReservedKeywordHandler(reservedKeywordConfig)
	registry.RegisterConstructHandler(reservedKeywordHandler, reservedKeywordConfig)

	// Регистрируем Import обработчик (Task 25)
	// Должен иметь очень высокий приоритет для импортов
	importConfig := config.ConstructHandlerConfig{
		ConstructType: common.ConstructImportStatement,
		Name:          "import-statement",
		Priority:      140, // Очень высокий приоритет для импортов
		Order:         1,
		IsEnabled:     true,
		IsFallback:    false,
		TokenPatterns: []config.TokenPattern{
			{TokenType: lexer.TokenImport, Offset: 0},
		},
	}

	importHandler := handler.NewImportHandler(importConfig)
	registry.RegisterConstructHandler(importHandler, importConfig)

	// Регистрируем CodeBlock обработчик (Task 25)
	// Должен иметь высокий приоритет для блоков кода
	codeBlockConfig := config.ConstructHandlerConfig{
		ConstructType: common.ConstructCodeBlock,
		Name:          "code-block-statement",
		Priority:      130, // Высокий приоритет для блоков кода
		Order:         1,
		IsEnabled:     true,
		IsFallback:    false,
		TokenPatterns: []config.TokenPattern{
			{TokenType: lexer.TokenLua, Offset: 0},
			{TokenType: lexer.TokenPython, Offset: 0},
			{TokenType: lexer.TokenPy, Offset: 0},
			{TokenType: lexer.TokenGo, Offset: 0},
			{TokenType: lexer.TokenNode, Offset: 0},
			{TokenType: lexer.TokenJS, Offset: 0},
		},
	}

	codeBlockHandler := handler.NewCodeBlockHandlerWithVerbose(codeBlockConfig, verbose)
	registry.RegisterConstructHandler(codeBlockHandler, codeBlockConfig)

	// Регистрируем Pipe обработчик для pipe expressions
	// PipeHandler должен иметь высокий приоритет для обработки pipe expressions
	pipeConfig := config.ConstructHandlerConfig{
		ConstructType: common.ConstructPipe,
		Name:          "pipe-expression",
		Priority:      121, // Приоритет выше чем ParenthesizedExpressionHandler (120) для обработки скобок с pipe внутри
		Order:         1,
		IsEnabled:     true,
		IsFallback:    false,
		TokenPatterns: []config.TokenPattern{
			{TokenType: lexer.TokenIdentifier, Offset: 0}, // Обрабатываем TokenIdentifier
			{TokenType: lexer.TokenLua, Offset: 0},        // Обрабатываем токены языков
			{TokenType: lexer.TokenPython, Offset: 0},
			{TokenType: lexer.TokenPy, Offset: 0},
			{TokenType: lexer.TokenGo, Offset: 0},
			{TokenType: lexer.TokenNode, Offset: 0},
			{TokenType: lexer.TokenJS, Offset: 0},
			{TokenType: lexer.TokenLeftParen, Offset: 0}, // Обрабатываем скобки (если есть | внутри)
			{TokenType: lexer.TokenPipe, Offset: 0},      // Обрабатываем операторы |
			{TokenType: lexer.TokenBitwiseOr, Offset: 0}, // Обрабатываем операторы |
		},
	}

	pipeHandler := handler.NewPipeHandler(pipeConfig)
	registry.RegisterConstructHandler(pipeHandler, pipeConfig)

	// Регистрируем LanguageCall обработчик
	languageCallConfig := config.ConstructHandlerConfig{
		ConstructType: common.ConstructLanguageCall,
		Name:          "language-call",
		Priority:      110, // Приоритет ниже LanguageCallStatement но выше остальных
		Order:         2,
		IsEnabled:     true,
		IsFallback:    false,
		TokenPatterns: []config.TokenPattern{
			{TokenType: lexer.TokenIdentifier, Offset: 0},
			{TokenType: lexer.TokenLua, Offset: 0},
			{TokenType: lexer.TokenPython, Offset: 0},
			{TokenType: lexer.TokenPy, Offset: 0},
			{TokenType: lexer.TokenGo, Offset: 0},
			{TokenType: lexer.TokenNode, Offset: 0},
			{TokenType: lexer.TokenJS, Offset: 0},
		},
	}

	languageCallHandler := handler.NewLanguageCallHandlerWithVerbose(languageCallConfig, verbose)
	registry.RegisterConstructHandler(languageCallHandler, languageCallConfig)

	// Регистрируем BuiltinFunction обработчик для вызовов builtin функций
	builtinFunctionConfig := config.ConstructHandlerConfig{
		ConstructType: common.ConstructFunction,
		Name:          "builtin-function-call",
		Priority:      120, // Самый высокий приоритет для builtin функций
		Order:         1,
		IsEnabled:     true,
		IsFallback:    false,
		TokenPatterns: []config.TokenPattern{
			{TokenType: lexer.TokenIdentifier, Offset: 0},
		},
	}

	builtinFunctionHandler := handler.NewBuiltinFunctionHandlerWithVerbose(builtinFunctionConfig, verbose)
	registry.RegisterConstructHandler(builtinFunctionHandler, builtinFunctionConfig)

	// Регистрируем QualifiedVariable обработчик (низкий приоритет для простых переменных)
	qualifiedVarConfig := config.ConstructHandlerConfig{
		ConstructType: common.ConstructVariable,
		Name:          "qualified-variable",
		Priority:      15, // Выше чем VariableReadHandler
		Order:         4,
		IsEnabled:     true,
		IsFallback:    false,
		TokenPatterns: []config.TokenPattern{
			{TokenType: lexer.TokenIdentifier, Offset: 0},
			{TokenType: lexer.TokenLua, Offset: 0},
			{TokenType: lexer.TokenPython, Offset: 0},
			{TokenType: lexer.TokenPy, Offset: 0},
			{TokenType: lexer.TokenGo, Offset: 0},
			{TokenType: lexer.TokenNode, Offset: 0},
			{TokenType: lexer.TokenJS, Offset: 0},
		},
	}

	qualifiedVarHandler := handler.NewQualifiedVariableHandler(qualifiedVarConfig)
	registry.RegisterConstructHandler(qualifiedVarHandler, qualifiedVarConfig)

	// Регистрируем CStyleForLoop обработчик для C-style циклов (самый высокий приоритет)
	cStyleForLoopConfig := config.ConstructHandlerConfig{
		ConstructType: common.ConstructCStyleForLoop,
		Name:          "c-style-for-loop",
		Priority:      95, // Выше приоритет, чем NumericForLoop
		Order:         1,
		IsEnabled:     true,
		IsFallback:    false,
		TokenPatterns: []config.TokenPattern{
			{TokenType: lexer.TokenFor, Offset: 0},
		},
	}

	cStyleForLoopHandler := handler.NewCStyleForLoopHandlerWithVerbose(cStyleForLoopConfig, verbose)
	registry.RegisterConstructHandler(cStyleForLoopHandler, cStyleForLoopConfig)

	// Регистрируем NumericForLoop обработчик для Lua циклов
	numericForLoopConfig := config.ConstructHandlerConfig{
		ConstructType: common.ConstructNumericForLoop,
		Name:          "numeric-for-loop",
		Priority:      85, // Ниже приоритет, чем CStyleForLoop
		Order:         2,
		IsEnabled:     true,
		IsFallback:    false,
		TokenPatterns: []config.TokenPattern{
			{TokenType: lexer.TokenFor, Offset: 0},
		},
	}

	numericForLoopHandler := handler.NewNumericForLoopHandlerWithVerbose(numericForLoopConfig, verbose)
	registry.RegisterConstructHandler(numericForLoopHandler, numericForLoopConfig)

	// Регистрируем ForInLoop обработчик для Python циклов
	forInLoopConfig := config.ConstructHandlerConfig{
		ConstructType: common.ConstructForInLoop,
		Name:          "for-in-loop",
		Priority:      85, // Ниже приоритет, чем NumericForLoop
		Order:         3,
		IsEnabled:     true,
		IsFallback:    false,
		TokenPatterns: []config.TokenPattern{
			{TokenType: lexer.TokenFor, Offset: 0},
		},
	}

	forInLoopHandler := handler.NewForInLoopHandlerWithVerbose(forInLoopConfig, verbose)
	registry.RegisterConstructHandler(forInLoopHandler, forInLoopConfig)

	// Регистрируем Match обработчик
	matchConfig := config.ConstructHandlerConfig{
		ConstructType: common.ConstructMatch,
		Name:          "match-statement",
		Priority:      110, // Высокий приоритет
		Order:         1,
		IsEnabled:     true,
		IsFallback:    false,
		TokenPatterns: []config.TokenPattern{
			{TokenType: lexer.TokenMatch, Offset: 0},
		},
	}

	matchHandler := handler.NewMatchHandlerWithVerbose(matchConfig, verbose)
	registry.RegisterConstructHandler(matchHandler, matchConfig)

	// Регистрируем Bitstring обработчик
	bitstringConfig := config.ConstructHandlerConfig{
		ConstructType: common.ConstructBitstring,
		Name:          "bitstring",
		Priority:      105, // Приоритет между LanguageCall (100) и QualifiedVariable (95)
		Order:         3,
		IsEnabled:     true,
		IsFallback:    false,
		TokenPatterns: []config.TokenPattern{
			{TokenType: lexer.TokenDoubleLeftAngle, Offset: 0},
		},
	}

	bitstringHandler := handler.NewBitstringHandler(bitstringConfig)
	registry.RegisterConstructHandler(bitstringHandler, bitstringConfig)

	// Регистрируем LanguageCallStatement обработчик для background tasks
	// Должен иметь самый высокий приоритет для обнаружения & в конце
	languageCallStatementConfig := config.ConstructHandlerConfig{
		ConstructType: common.ConstructLanguageCall,
		Name:          "language-call-statement",
		Priority:      115, // Самый высокий приоритет для & (выше LanguageCall и PipeHandler)
		Order:         1,
		IsEnabled:     true,
		IsFallback:    false,
		TokenPatterns: []config.TokenPattern{
			{TokenType: lexer.TokenIdentifier, Offset: 0},
			{TokenType: lexer.TokenLua, Offset: 0},
			{TokenType: lexer.TokenPython, Offset: 0},
			{TokenType: lexer.TokenPy, Offset: 0},
			{TokenType: lexer.TokenGo, Offset: 0},
			{TokenType: lexer.TokenNode, Offset: 0},
			{TokenType: lexer.TokenJS, Offset: 0},
		},
	}

	languageCallStatementHandler := handler.NewLanguageCallStatementHandler(languageCallStatementConfig)
	registry.RegisterConstructHandler(languageCallStatementHandler, languageCallStatementConfig)

	// Регистрируем BackgroundTask обработчик для & токена и language tokens
	// Должен иметь самый высокий приоритет для обработки background tasks
	backgroundTaskConfig := config.ConstructHandlerConfig{
		ConstructType: common.ConstructLanguageCall,
		Name:          "background-task",
		Priority:      200, // Самый высокий приоритет для background tasks
		Order:         1,
		IsEnabled:     true,
		IsFallback:    false,
		TokenPatterns: []config.TokenPattern{
			{TokenType: lexer.TokenAmpersand, Offset: 0}, // Для случаев когда & в начале
			{TokenType: lexer.TokenLua, Offset: 0},       // Для background tasks с языковыми токенами
			{TokenType: lexer.TokenPython, Offset: 0},
			{TokenType: lexer.TokenPy, Offset: 0},
			{TokenType: lexer.TokenGo, Offset: 0},
			{TokenType: lexer.TokenNode, Offset: 0},
			{TokenType: lexer.TokenJS, Offset: 0},
		},
	}

	backgroundTaskHandler := handler.NewBackgroundTaskHandler(backgroundTaskConfig)
	registry.RegisterConstructHandler(backgroundTaskHandler, backgroundTaskConfig)

	// Регистрируем While обработчик для while циклов
	whileLoopConfig := config.ConstructHandlerConfig{
		ConstructType: common.ConstructWhileLoop,
		Name:          "while-loop",
		Priority:      100, // Высокий приоритет для while циклов
		Order:         1,
		IsEnabled:     true,
		IsFallback:    false,
		TokenPatterns: []config.TokenPattern{
			{TokenType: lexer.TokenWhile, Offset: 0},
		},
	}

	whileLoopHandler := handler.NewWhileLoopHandler(whileLoopConfig)
	registry.RegisterConstructHandler(whileLoopHandler, whileLoopConfig)

	// Регистрируем Break обработчик с высоким приоритетом
	breakConfig := config.ConstructHandlerConfig{
		ConstructType: common.ConstructBreak,
		Name:          "break-statement",
		Priority:      150, // Очень высокий приоритет для break
		Order:         1,
		IsEnabled:     true,
		IsFallback:    false,
		TokenPatterns: []config.TokenPattern{
			{TokenType: lexer.TokenBreak, Offset: 0},
		},
	}

	breakHandler := handler.NewBreakHandler(breakConfig)
	registry.RegisterConstructHandler(breakHandler, breakConfig)

	// Регистрируем Continue обработчик с высоким приоритетом
	continueConfig := config.ConstructHandlerConfig{
		ConstructType: common.ConstructContinue,
		Name:          "continue-statement",
		Priority:      150, // Очень высокий приоритет для continue
		Order:         1,
		IsEnabled:     true,
		IsFallback:    false,
		TokenPatterns: []config.TokenPattern{
			{TokenType: lexer.TokenContinue, Offset: 0},
		},
	}

	continueHandler := handler.NewContinueHandler(continueConfig)
	registry.RegisterConstructHandler(continueHandler, continueConfig)

	// Регистрируем If обработчик для if/else конструкций
	ifConfig := config.ConstructHandlerConfig{
		ConstructType: common.ConstructIf, // Нужно добавить этот тип в common
		Name:          "if-statement",
		Priority:      100, // Приоритет как у while циклов
		Order:         1,
		IsEnabled:     true,
		IsFallback:    false,
		TokenPatterns: []config.TokenPattern{
			{TokenType: lexer.TokenIf, Offset: 0}, // Use proper token constant
		},
	}

	ifHandler := handler.NewIfHandlerWithVerbose(ifConfig, verbose)
	registry.RegisterConstructHandler(ifHandler, ifConfig)

	// Регистрируем Assignment обработчик для простых присваиваний
	assignmentConfig := config.ConstructHandlerConfig{
		ConstructType: common.ConstructAssignment,
		Name:          "assignment",
		Priority:      96, // Выше IndexExpressionHandler (95) для поддержки индексного присваивания
		Order:         4,
		IsEnabled:     true,
		IsFallback:    false,
		TokenPatterns: []config.TokenPattern{
			{TokenType: lexer.TokenIdentifier, Offset: 0},
			{TokenType: lexer.TokenLua, Offset: 0},
			{TokenType: lexer.TokenPython, Offset: 0},
			{TokenType: lexer.TokenPy, Offset: 0},
			{TokenType: lexer.TokenGo, Offset: 0},
			{TokenType: lexer.TokenNode, Offset: 0},
			{TokenType: lexer.TokenJS, Offset: 0},
		},
	}

	assignmentHandler := handler.NewAssignmentHandlerWithVerbose(assignmentConfig.Priority, assignmentConfig.Order, verbose)
	registry.RegisterConstructHandler(assignmentHandler, assignmentConfig)

	// Регистрируем BitstringPatternAssignment обработчик для inplace pattern matching
	bitstringPatternAssignmentConfig := config.ConstructHandlerConfig{
		ConstructType: common.ConstructAssignment, // Используем тот же тип, что и для обычного присваивания
		Name:          "bitstring-pattern-assignment",
		Priority:      110, // Выше приоритет чем у BitstringHandler (105)
		Order:         4,
		IsEnabled:     true,
		IsFallback:    false,
		TokenPatterns: []config.TokenPattern{
			{TokenType: lexer.TokenDoubleLeftAngle, Offset: 0}, // Для bitstring patterns <<...>>
		},
	}

	bitstringPatternAssignmentHandler := handler.NewBitstringPatternAssignmentHandlerWithVerbose(bitstringPatternAssignmentConfig.Priority, bitstringPatternAssignmentConfig.Order, verbose)
	registry.RegisterConstructHandler(bitstringPatternAssignmentHandler, bitstringPatternAssignmentConfig)

	// Регистрируем BinaryExpression обработчик для бинарных операций
	binaryExpressionConfig := config.ConstructHandlerConfig{
		ConstructType: common.ConstructBinaryExpression,
		Name:          "binary-expression",
		Priority:      70, // Средний приоритет для бинарных выражений
		Order:         5,
		IsEnabled:     true,
		IsFallback:    false,
		TokenPatterns: []config.TokenPattern{
			{TokenType: lexer.TokenIdentifier, Offset: 0},    // Для выражений типа "a = b + c"
			{TokenType: lexer.TokenNumber, Offset: 0},       // Для выражений типа "1 + 2"
			{TokenType: lexer.TokenTrue, Offset: 0},         // Для выражений типа "true == false"
			{TokenType: lexer.TokenFalse, Offset: 0},        // Для выражений типа "false != true"
			{TokenType: lexer.TokenNil, Offset: 0},          // Для выражений типа "nil == nil"
			{TokenType: lexer.TokenString, Offset: 0},       // Для выражений типа "'a' ++ 'b'"
			{TokenType: lexer.TokenQuestion, Offset: 0},     // Для тернарных выражений типа "cond ? true : false"
			{TokenType: lexer.TokenPlus, Offset: 0},         // Для бинарных выражений типа "a + b"
			{TokenType: lexer.TokenMinus, Offset: 0},        // Для бинарных выражений типа "a - b"
			{TokenType: lexer.TokenMultiply, Offset: 0},     // Для бинарных выражений типа "a * b"
			{TokenType: lexer.TokenSlash, Offset: 0},        // Для бинарных выражений типа "a / b"
			{TokenType: lexer.TokenModulo, Offset: 0},       // Для бинарных выражений типа "a % b"
			{TokenType: lexer.TokenPower, Offset: 0},        // Для бинарных выражений типа "a ** b"
			{TokenType: lexer.TokenEqual, Offset: 0},        // Для бинарных выражений типа "a == b"
			{TokenType: lexer.TokenNotEqual, Offset: 0},     // Для бинарных выражений типа "a != b"
			{TokenType: lexer.TokenLess, Offset: 0},         // Для бинарных выражений типа "a < b"
			{TokenType: lexer.TokenLessEqual, Offset: 0},    // Для бинарных выражений типа "a <= b"
			{TokenType: lexer.TokenGreater, Offset: 0},      // Для бинарных выражений типа "a > b"
			{TokenType: lexer.TokenGreaterEqual, Offset: 0}, // Для бинарных выражений типа "a >= b"
			{TokenType: lexer.TokenAnd, Offset: 0},          // Для бинарных выражений типа "a && b"
			{TokenType: lexer.TokenOr, Offset: 0},           // Для бинарных выражений типа "a || b"
			{TokenType: lexer.TokenAmpersand, Offset: 0},   // Для бинарных выражений типа "a & b"
			{TokenType: lexer.TokenCaret, Offset: 0},        // Для бинарных выражений типа "a ^ b"
			{TokenType: lexer.TokenDoubleLeftAngle, Offset: 0},  // Для бинарных выражений типа "a << b"
			{TokenType: lexer.TokenDoubleRightAngle, Offset: 0}, // Для бинарных выражений типа "a >> b"
			{TokenType: lexer.TokenConcat, Offset: 0},       // Для бинарных выражений типа "a ++ b"
		},
	}

	binaryExpressionHandler := handler.NewBinaryExpressionHandlerWithVerbose(binaryExpressionConfig, verbose)
	registry.RegisterConstructHandler(binaryExpressionHandler, binaryExpressionConfig)

	// Регистрируем IndexExpression обработчик для индексированного доступа (dict["key"])
	indexExpressionConfig := config.ConstructHandlerConfig{
		ConstructType: common.ConstructVariable, // Используем тип для переменных/выражений
		Name:          "index-expression",
		Priority:      95, // Высокий приоритет для индексированного доступа
		Order:         1,
		IsEnabled:     true,
		IsFallback:    false,
		TokenPatterns: []config.TokenPattern{
			{TokenType: lexer.TokenIdentifier, Offset: 0}, // Идентификатор, за которым следует [
		},
	}

	indexExpressionHandler := handler.NewIndexExpressionHandler(indexExpressionConfig)
	registry.RegisterConstructHandler(indexExpressionHandler, indexExpressionConfig)

	// Регистрируем UnaryExpression обработчик для унарных операций
	unaryExpressionConfig := config.ConstructHandlerConfig{
		ConstructType: common.ConstructUnaryExpression,
		Name:          "unary-expression",
		Priority:      150, // Очень высокий приоритет для унарных операторов (выше LanguageCall)
		Order:         1,
		IsEnabled:     true,
		IsFallback:    false,
		TokenPatterns: []config.TokenPattern{
			{TokenType: lexer.TokenPlus, Offset: 0},   // +
			{TokenType: lexer.TokenMinus, Offset: 0},  // -
			{TokenType: lexer.TokenNot, Offset: 0},    // !
			{TokenType: lexer.TokenTilde, Offset: 0},  // ~
			{TokenType: lexer.TokenAt, Offset: 0},     // @
		},
	}

	unaryExpressionHandler := handler.NewUnaryExpressionHandler(unaryExpressionConfig)
	registry.RegisterConstructHandler(unaryExpressionHandler, unaryExpressionConfig)

	// Регистрируем Literal обработчик для чисел (высокий приоритет)
	literalConfig := config.ConstructHandlerConfig{
		ConstructType: common.ConstructLiteral,
		Name:          "literal",
		Priority:      150, // Высокий приоритет для литералов
		Order:         0,
		IsEnabled:     true,
		IsFallback:    false,
		TokenPatterns: []config.TokenPattern{
			{TokenType: lexer.TokenNumber, Offset: 0},
			{TokenType: lexer.TokenMinus, Offset: 0}, // Для отрицательных чисел
			{TokenType: lexer.TokenTrue, Offset: 0},  // Для булевых значений
			{TokenType: lexer.TokenFalse, Offset: 0}, // Для булевых значений
			{TokenType: lexer.TokenNil, Offset: 0},    // Для nil значения
			{TokenType: lexer.TokenString, Offset: 0}, // Для строковых литералов
		},
	}

	literalHandler := handler.NewLiteralHandler(literalConfig)
	registry.RegisterConstructHandler(literalHandler, literalConfig)

	// Регистрируем Expression обработчик для выражений (высокий приоритет для MINUS)
	expressionMinusConfig := config.ConstructHandlerConfig{
		ConstructType: common.ConstructExpression,
		Name:          "expression-minus",
		Priority:      160, // Высокий приоритет для обработки MINUS перед literal
		Order:         10,
		IsEnabled:     true,
		IsFallback:    false,
		TokenPatterns: []config.TokenPattern{
			{TokenType: lexer.TokenMinus, Offset: 0}, // Обработаем MINUS перед literal
		},
	}

	expressionMinusHandler := handler.NewExpressionHandler(expressionMinusConfig)
	registry.RegisterConstructHandler(expressionMinusHandler, expressionMinusConfig)

	// Регистрируем Expression обработчик для выражений (низкий приоритет для NUMBER)
	expressionNumberConfig := config.ConstructHandlerConfig{
		ConstructType: common.ConstructExpression,
		Name:          "expression-number",
		Priority:      5, // Низкий приоритет, чтобы вызываться после literal
		Order:         11,
		IsEnabled:     true,
		IsFallback:    false,
		TokenPatterns: []config.TokenPattern{
			{TokenType: lexer.TokenNumber, Offset: 0}, // Обработаем NUMBER после literal
		},
	}

	expressionNumberHandler := handler.NewExpressionHandler(expressionNumberConfig)
	registry.RegisterConstructHandler(expressionNumberHandler, expressionNumberConfig)

	// Регистрируем Array обработчик для массивов [1, 2, 3]
	arrayConfig := config.ConstructHandlerConfig{
		ConstructType: common.ConstructArray,
		Name:          "array",
		Priority:      200, // Высокий приоритет для массивов
		Order:         10,
		IsEnabled:     true,
		IsFallback:    false,
		TokenPatterns: []config.TokenPattern{
			{TokenType: lexer.TokenLBracket, Offset: 0}, // Начинается с [
		},
	}

	arrayHandler := handler.NewArrayHandler(200, 10)
	registry.RegisterConstructHandler(arrayHandler, arrayConfig)

	// Регистрируем VariableRead обработчик для чтения переменных
	variableReadConfig := config.ConstructHandlerConfig{
		ConstructType: common.ConstructVariableRead,
		Name:          "variable-read",
		Priority:      10, // Низкий приоритет, fallback для идентификаторов
		Order:         15,
		IsEnabled:     true,
		IsFallback:    true,
		TokenPatterns: []config.TokenPattern{
			{TokenType: lexer.TokenIdentifier, Offset: 0}, // Обычные идентификаторы
		},
	}

	variableReadHandler := handler.NewVariableReadHandler(variableReadConfig)
	registry.RegisterConstructHandler(variableReadHandler, variableReadConfig)

	// Регистрируем Object обработчик для объектов {"key": "value"}
	objectConfig := config.ConstructHandlerConfig{
		ConstructType: common.ConstructObject,
		Name:          "object",
		Priority:      200, // Высокий приоритет для объектов
		Order:         11,
		IsEnabled:     true,
		IsFallback:    false,
		TokenPatterns: []config.TokenPattern{
			{TokenType: lexer.TokenLBrace, Offset: 0}, // Начинается с {
		},
	}

	objectHandler := handler.NewObjectHandler(200, 11)
	registry.RegisterConstructHandler(objectHandler, objectConfig)

	return &UnifiedParser{
		registry: registry,
		verbose:  verbose,
	}
}

// Parse разбирает входную строку и возвращает AST
func (p *UnifiedParser) Parse(input string) (ast.Statement, []ast.ParseError) {
	// 1. Создаем лексер
	lex := lexer.NewLexer(input)
	tokenStream := stream.NewTokenStream(lex)

	// 2. Проверяем, есть ли токены
	if !tokenStream.HasMore() {
		return nil, []ast.ParseError{{
			Type:     ast.ErrorSyntax,
			Position: ast.Position{Line: 1, Column: 1, Offset: 0},
			Message:  "empty input",
			Context:  input,
		}}
	}

	// Собираем все statements из ввода
	statements := []ast.Statement{}
	var parseErrors []ast.ParseError

	// 3. Обрабатываем все statements в вводе
	for tokenStream.HasMore() {
		// Пропускаем whitespace токены
		// Пропускаем whitespace и newlines между statements
		for tokenStream.HasMore() {
			currentToken := tokenStream.Current()
			if currentToken.Type == lexer.TokenNewline {
				tokenStream.Consume()
			} else {
				// Проверяем, является ли текущий токен whitespace (пробел, таб и т.д.)
				// В лексере нет отдельного токена для whitespace, он просто пропускается
				// Но нам нужно проверить наличие пробелов вручную
				if currentToken.Value == " " || currentToken.Value == "\t" || currentToken.Value == "\r" {
					tokenStream.Consume()
				} else {
					break
				}
			}
		}

		if !tokenStream.HasMore() {
			break
		}

		// 4. Получаем текущий токен
		currentToken := tokenStream.Current()

		// 5. Получаем все обработчики для токена
		tokens := []lexer.Token{currentToken}
		handlers := p.registry.GetAllHandlersForTokenSequence(tokens)
		if len(handlers) == 0 {
			if p.verbose {
				fmt.Printf("DEBUG: UnifiedParser - no handler found for token: %s (%s) at line %d col %d\n",
					currentToken.Value, currentToken.Type, currentToken.Line, currentToken.Column)
			}
			parseErrors = append(parseErrors, ast.ParseError{
				Type:     ast.ErrorSyntax,
				Position: tokenToPosition(currentToken),
				Message:  fmt.Sprintf("no handler found for token: %s line %d col %d", currentToken.Value, currentToken.Line, currentToken.Column),
				Context:  input,
			})
			break
		}

	// 6. Пробуем каждый обработчик в порядке приоритета
	var lastErr error
	var result interface{}
	var ctx *common.ParseContext

	if p.verbose {
		fmt.Printf("DEBUG: UnifiedParser - available handlers: ")
		for _, h := range handlers {
			fmt.Printf("%s ", h.Name())
		}
		fmt.Printf("\n")
	}

	for i, h := range handlers {
		// 7. Создаем клон потока для каждого обработчика
		clonedStream := tokenStream.Clone()

		// 8. Проверяем, может ли обработчик обработать токен
		if p.verbose {
			fmt.Printf("DEBUG: UnifiedParser - current token: %s(%s), trying handler: %s, CanHandle: %v\n", currentToken.Type, currentToken.Value, h.Name(), h.CanHandle(currentToken))
		}
		if !h.CanHandle(currentToken) {
			continue
		}

		// 9. Создаем контекст и вызываем обработчик с клоном потока
		ctx = &common.ParseContext{
				TokenStream: clonedStream,
				Parser:      nil, // Не используем старый интерфейс
				Depth:       0,
				MaxDepth:    100,
				Guard:       newProtoRecursionGuard(100),
				LoopDepth:   0,     // Инициализируем глубину циклов для контекстной валидации
				InputStream: input, // Передаем оригинальный исходный код
			}

			var err error
			result, err = h.Handle(ctx)
			if err == nil && result != nil {
				// Обработчик успешно обработал входные данные и вернул непустой результат
				// Для CodeBlockStatement нам нужно остановиться после закрывающей скобки

				// Синхронизируем позицию основного потока с позицией клона
				// Это правильно продвигает позицию в потоке
				tokenStream.SetPosition(clonedStream.Position())

				lastErr = nil // Сбрасываем последнюю ошибку
				break
			}

			// Если обработчик вернул nil, nil - это означает, что он не может обработать данный токен
			// Нужно попробовать следующий обработчик
			if err == nil && result == nil {
				continue
			}

			// Если обработчик с более высоким приоритетом потребил токены, но вернул ошибку,
			// то это финальная ошибка, другие обработчики не пробуем
			if i == 0 && isPartialMatchError(err.Error()) {
				lastErr = err
				break
			}

			// Если LanguageCallStatementHandler возвращает конкретную ошибку,
			// означающую, что это не background task, то пробуем следующий обработчик
			if err.Error() == "not a background task statement" {
				continue
			}

			// Если LanguageCallHandler возвращает конкретную ошибку,
			// означающую, что это background task, который должен обрабатывать LanguageCallStatementHandler,
			// то пробуем следующий обработчик
			if err.Error() == "background task detected - should be handled by LanguageCallStatementHandler" {
				continue
			}

			// Если ReservedKeywordHandler возвращает конкретную ошибку,
			// означающую, что это не присваивание зарезервированному слову, то пробуем следующий обработчик
			if err.Error() == "not a reserved keyword assignment" {
				if p.verbose {
					fmt.Printf("DEBUG: UnifiedParser - skipping to next handler due to reserved keyword assignment error\n")
				}
				continue
			}

			// Если LanguageCallHandler возвращает конкретную ошибку,
			// означающую, что это не вызов функции, а присваивание, то пробуем следующий обработчик
			if err.Error() == "not a language call - assignment detected" {
				if p.verbose {
					fmt.Printf("DEBUG: UnifiedParser - skipping to next handler due to language call assignment error\n")
				}
				continue
			}

			// Если BinaryExpressionHandler возвращает конкретную ошибку,
			// означающую, что это вызов функции с точечной нотацией, который должен обрабатывать LanguageCallHandler, то пробуем следующий обработчик
			if err.Error() == "qualified function call detected - should be handled by LanguageCallHandler" {
				if p.verbose {
					fmt.Printf("DEBUG: UnifiedParser - skipping to next handler due to qualified function call detection\n")
				}
				continue
			}

			// Если CodeBlockHandler возвращает конкретную ошибку,
			// означающую, что это не блок кода, то пробуем следующий обработчик
			if err.Error() == "not a code block statement" {
				continue
			}

			// Если любой handler возвращает ошибку о неквалифицированной переменной, то это финальная ошибка
			if strings.Contains(err.Error(), "not a qualified variable") {
				lastErr = err
				break
			}

			// Если LanguageCallHandler возвращает конкретную ошибку для неполных выражений,
			// то это финальная ошибка, другие обработчики не пробуем
			if err.Error() == "unexpected EOF after argument" {
				lastErr = err
				break
			}

			// Если это ошибка от ReservedKeywordHandler (второй по приоритету после background-task),
			// и это ошибка о присваивании зарезервированному слову, сохраняем её и выходим
			if i == 1 && contains(err.Error(), "cannot assign to reserved keyword") {
				lastErr = err
				break
			}

			// Если это ошибка от ReservedKeywordHandler и это "not a qualified variable", сохраняем её и выходим
			if i == 1 && err.Error() == "not a qualified variable" {
				lastErr = err
				break
			}

			// Сохраняем последнюю ошибку
			lastErr = err
		}

		if lastErr != nil {
			// Все обработчики вернули ошибки
			position := tokenToPosition(currentToken)
			if posErr, ok := lastErr.(interface{ GetPosition() ast.Position }); ok {
				position = posErr.GetPosition()
			}

			parseErrors = append(parseErrors, ast.ParseError{
				Type:     ast.ErrorSyntax,
				Position: position,
				Message:  lastErr.Error(),
				Context:  input,
			})
			break
		}

	// 10. Проверяем наличие индексных выражений [index] после array literal
	// Это нужно сделать ДО конвертации в Statement
	if arrayLit, ok := result.(*ast.ArrayLiteral); ok {
		currentExpr := ast.Expression(arrayLit)
		binaryHandler := handler.NewBinaryExpressionHandler(config.ConstructHandlerConfig{})
		for ctx.TokenStream.HasMore() && ctx.TokenStream.Current().Type == lexer.TokenLBracket {
			indexExpr, err := binaryHandler.ParseIndexExpression(ctx, currentExpr)
			if err != nil {
				parseErrors = append(parseErrors, ast.ParseError{
					Type:     ast.ErrorSyntax,
					Position: tokenToPosition(ctx.TokenStream.Current()),
					Message:  err.Error(),
					Context:  input,
				})
				break
			}
			currentExpr = indexExpr
		}
		result = currentExpr
		// Синхронизируем основной tokenStream с ctx.TokenStream
		tokenStream.SetPosition(ctx.TokenStream.Position())
	}

	// 11. Конвертируем результат в Statement
	if p.verbose {
		fmt.Printf("DEBUG: UnifiedParser converting result to Statement, result type: %T\n", result)
	}
	// Сначала проверяем LanguageCall и BuiltinFunctionCall специально (они реализуют Statement но нуждаются в special handling)
	if langCall, ok := result.(*ast.LanguageCall); ok {
		// Проверяем, идет ли после language call elvis/ternary оператор
		if tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenQuestion {
			// Это elvis или ternary выражение - продолжаем парсить
			// Создаем новый контекст с основным tokenStream
			elvisCtx := &common.ParseContext{
				TokenStream: tokenStream,
				Parser:      nil,
				Depth:       0,
				MaxDepth:    100,
				Guard:       newProtoRecursionGuard(100),
				LoopDepth:   0,
				InputStream: input,
			}
			binaryExprHandler := handler.NewBinaryExpressionHandlerWithVerbose(config.ConstructHandlerConfig{}, p.verbose)
			fullExpr, err := binaryExprHandler.ParseFullExpression(elvisCtx, langCall)
			if err == nil && fullExpr != nil {
				// Успешно распарсили полное выражение
				// Синхронизируем позицию основного потока
				tokenStream.SetPosition(elvisCtx.TokenStream.Position())
				exprStmt := &ast.ExpressionStatement{Expression: fullExpr}
				statements = append(statements, exprStmt)
		} else {
			// Если не получилось, LanguageCall реализует Statement интерфейс, добавляем напрямую
			statements = append(statements, langCall)
		}
		} else {
			// Нет elvis operator, LanguageCall реализует Statement интерфейс, добавляем напрямую
			statements = append(statements, langCall)
		}
	} else if builtinCall, ok := result.(*ast.BuiltinFunctionCall); ok {
		// Проверяем, идет ли после builtin call elvis/ternary оператор
		if tokenStream.HasMore() && tokenStream.Current().Type == lexer.TokenQuestion {
			// Это elvis или ternary выражение - продолжаем парсить
			// Создаем новый контекст с основным tokenStream
			elvisCtx := &common.ParseContext{
				TokenStream: tokenStream,
				Parser:      nil,
				Depth:       0,
				MaxDepth:    100,
				Guard:       newProtoRecursionGuard(100),
				LoopDepth:   0,
				InputStream: input,
			}
			binaryExprHandler := handler.NewBinaryExpressionHandlerWithVerbose(config.ConstructHandlerConfig{}, p.verbose)
			fullExpr, err := binaryExprHandler.ParseFullExpression(elvisCtx, builtinCall)
			if err == nil && fullExpr != nil {
				// Успешно распарсили полное выражение
				// Синхронизируем позицию основного потока
				tokenStream.SetPosition(elvisCtx.TokenStream.Position())
				exprStmt := &ast.ExpressionStatement{Expression: fullExpr}
				statements = append(statements, exprStmt)
			} else {
				// Если не получилось, просто оборачиваем BuiltinFunctionCall в ExpressionStatement
				exprStmt := &ast.ExpressionStatement{Expression: builtinCall}
				statements = append(statements, exprStmt)
			}
		} else {
			// Просто оборачиваем BuiltinFunctionCall в ExpressionStatement
			exprStmt := &ast.ExpressionStatement{Expression: builtinCall}
			statements = append(statements, exprStmt)
		}
	} else if statement, ok := result.(ast.Statement); ok {
		if p.verbose {
			fmt.Printf("DEBUG: UnifiedParser appending statement: %T\n", statement)
		}
		statements = append(statements, statement)
	} else if expression, ok := result.(ast.Expression); ok {
		// Если результат - Expression, проверяем, реализует ли он также Statement
		if statement, ok := expression.(ast.Statement); ok {
			// Если Expression также является Statement (например, PipeExpression), добавляем его напрямую
			statements = append(statements, statement)
		} else if pipeExpr, ok := expression.(*ast.PipeExpression); ok {
			// PipeExpression реализует оба интерфейса Expression и Statement
			statements = append(statements, pipeExpr)
		} else {
				// Для выражений от ExpressionHandler, оборачиваем в ExpressionStatement
				if _, isTernaryExpr := expression.(*ast.TernaryExpression); isTernaryExpr {
					exprStmt := &ast.ExpressionStatement{Expression: expression}
					statements = append(statements, exprStmt)
				} else if _, isElvisExpr := expression.(*ast.ElvisExpression); isElvisExpr {
					exprStmt := &ast.ExpressionStatement{Expression: expression}
					statements = append(statements, exprStmt)
				} else if _, isBinaryExpr := expression.(*ast.BinaryExpression); isBinaryExpr {
					exprStmt := &ast.ExpressionStatement{Expression: expression}
					statements = append(statements, exprStmt)
				} else if _, isUnaryExpr := expression.(*ast.UnaryExpression); isUnaryExpr {
					exprStmt := &ast.ExpressionStatement{Expression: expression}
					statements = append(statements, exprStmt)
				} else if _, isIdentifier := expression.(*ast.Identifier); isIdentifier {
					exprStmt := &ast.ExpressionStatement{Expression: expression}
					statements = append(statements, exprStmt)
				} else if _, isNumberLiteral := expression.(*ast.NumberLiteral); isNumberLiteral {
					exprStmt := &ast.ExpressionStatement{Expression: expression}
					statements = append(statements, exprStmt)
				} else if _, isArrayLiteral := expression.(*ast.ArrayLiteral); isArrayLiteral {
					exprStmt := &ast.ExpressionStatement{Expression: expression}
					statements = append(statements, exprStmt)
				} else if _, isObjectLiteral := expression.(*ast.ObjectLiteral); isObjectLiteral {
					exprStmt := &ast.ExpressionStatement{Expression: expression}
					statements = append(statements, exprStmt)
			} else if _, isBooleanLiteral := expression.(*ast.BooleanLiteral); isBooleanLiteral {
				exprStmt := &ast.ExpressionStatement{Expression: expression}
				statements = append(statements, exprStmt)
			} else if _, isNilLiteral := expression.(*ast.NilLiteral); isNilLiteral {
				exprStmt := &ast.ExpressionStatement{Expression: expression}
				statements = append(statements, exprStmt)
			} else if _, isStringLiteral := expression.(*ast.StringLiteral); isStringLiteral {
				exprStmt := &ast.ExpressionStatement{Expression: expression}
				statements = append(statements, exprStmt)
			} else if _, isVariableRead := expression.(*ast.VariableRead); isVariableRead {
					exprStmt := &ast.ExpressionStatement{Expression: expression}
					statements = append(statements, exprStmt)
				} else {
					// Для других типов выражений, игнорируем (они должны быть частью других конструкций)
					if p.verbose {
						fmt.Printf("DEBUG: UnifiedParser - ignoring expression result that should be part of a statement: %T\n", expression)
					}
					continue // Пропускаем это выражение, оно должно быть частью другого statement
				}
			}
	} else {
		if p.verbose {
			fmt.Printf("DEBUG: UnifiedParser - result is neither Statement nor Expression: %T (value: %v)\n", result, result)
		}
		parseErrors = append(parseErrors, ast.ParseError{
			Type:     ast.ErrorSemantic,
			Position: tokenToPosition(currentToken),
			Message:  fmt.Sprintf("result is not a statement (got %T)", result),
			Context:  input,
		})
		break
	}
	}

	// 11. Если были ошибки парсинга, возвращаем их
	if len(parseErrors) > 0 {
		return nil, parseErrors
	}

	// 12. Если был только один statement, возвращаем его напрямую
	if len(statements) == 1 {
		return statements[0], nil
	}

	// 13. Если было несколько statements, оборачиваем их в BlockStatement
	if len(statements) > 1 {
		// Создаем фиктивные токены для BlockStatement (так как у нас нет реальных фигурных скобок)
		dummyLBrace := lexer.Token{Type: lexer.TokenLBrace, Value: "{", Position: 0, Line: 1, Column: 1}
		dummyRBrace := lexer.Token{Type: lexer.TokenRBrace, Value: "}", Position: 0, Line: 1, Column: 1}

		blockStmt := ast.NewBlockStatement(dummyLBrace, dummyRBrace, statements)
		return blockStmt, nil
	}


	// 14. Если не было распарсено ни одного statement, пытаемся распарсить как standalone expression
	if len(statements) == 0 {
		if p.verbose {
			fmt.Printf("DEBUG: UnifiedParser - no statements parsed, trying expression fallback\n")
		}

		// Создаем новый лексер для всего ввода
		fallbackLexer := lexer.NewLexer(input)
		fallbackTokenStream := stream.NewTokenStream(fallbackLexer)

		// Собираем все токены выражения, пропуская newlines
		exprTokens := []lexer.Token{}
		for fallbackTokenStream.HasMore() {
			token := fallbackTokenStream.Current()
			if token.Type != lexer.TokenNewline {
				exprTokens = append(exprTokens, token)
			}
			fallbackTokenStream.Consume()
		}

		if p.verbose {
			fmt.Printf("DEBUG: UnifiedParser - expression fallback tokens: %v\n", exprTokens)
		}

		// Пробуем распарсить как выражение
		if len(exprTokens) > 0 {
			expr, exprErr := exprparser.ParseExpression(exprTokens)
			if p.verbose {
				fmt.Printf("DEBUG: UnifiedParser - expression parse result: expr=%v, err=%v\n", expr, exprErr)
			}
			if exprErr == nil && expr != nil {
				// Успешно распарсили выражение, оборачиваем в ExpressionStatement
				exprStmt := &ast.ExpressionStatement{Expression: expr}
				return exprStmt, nil
			}
		}
	}

	// Этот код не должен достигаться, но на всякий случай
	return nil, []ast.ParseError{{
		Type:     ast.ErrorSyntax,
		Position: ast.Position{Line: 1, Column: 1, Offset: 0},
		Message:  "no statements parsed",
		Context:  input,
	}}
}

// tryParseLineAsExpression пытается распарсить текущую строку как выражение
func (p *UnifiedParser) tryParseLineAsExpression(tokenStream stream.TokenStream, input string, statements *[]ast.Statement) bool {
	// Найдем начало и конец текущей строки
	currentPos := tokenStream.Position()
	lineStart := strings.LastIndex(input[:currentPos], "\n")
	if lineStart == -1 {
		lineStart = 0
	} else {
		lineStart++ // После \n
	}

	lineEnd := strings.Index(input[currentPos:], "\n")
	if lineEnd == -1 {
		lineEnd = len(input)
	} else {
		lineEnd += currentPos
	}

	lineInput := input[lineStart:lineEnd]
	if p.verbose {
		fmt.Printf("DEBUG: UnifiedParser - trying line expression for: '%s'\n", lineInput)
	}

	// Создаем новый лексер для строки
	fallbackLexer := lexer.NewLexer(lineInput)
	fallbackTokenStream := stream.NewTokenStream(fallbackLexer)

	// Собираем токены строки
	lineTokens := []lexer.Token{}
	for fallbackTokenStream.HasMore() {
		token := fallbackTokenStream.Current()
		if token.Type != lexer.TokenNewline {
			lineTokens = append(lineTokens, token)
		}
		fallbackTokenStream.Consume()
	}

	// Пробуем распарсить как выражение
	if len(lineTokens) > 0 {
		expr, exprErr := exprparser.ParseExpression(lineTokens)
		if p.verbose {
			fmt.Printf("DEBUG: UnifiedParser - line expression parse result: expr=%v, err=%v\n", expr, exprErr)
		}
		if exprErr == nil && expr != nil {
			// Успешно распарсили, создаем ExpressionStatement
			exprStmt := &ast.ExpressionStatement{Expression: expr}
			*statements = append(*statements, exprStmt)

			// Продвигаем основной поток до конца строки
			for tokenStream.HasMore() {
				currToken := tokenStream.Current()
				tokenStream.Consume()
				if currToken.Type == lexer.TokenNewline {
					break
				}
			}
			return true
		}
	}
	return false
}

// tokenToPosition конвертирует токен в позицию AST
func tokenToPosition(token lexer.Token) ast.Position {
	return ast.Position{
		Line:   token.Line,
		Column: token.Column,
		Offset: token.Position,
	}
}

// protoRecursionGuard реализует защиту от рекурсии для прототипа
type protoRecursionGuard struct {
	maxDepth     int
	currentDepth int
}

func newProtoRecursionGuard(maxDepth int) *protoRecursionGuard {
	return &protoRecursionGuard{
		maxDepth:     maxDepth,
		currentDepth: 0,
	}
}

func (rg *protoRecursionGuard) Enter() error {
	if rg.currentDepth >= rg.maxDepth {
		return fmt.Errorf("maximum recursion depth exceeded: %d", rg.maxDepth)
	}
	rg.currentDepth++
	return nil
}

func (rg *protoRecursionGuard) Exit() {
	if rg.currentDepth > 0 {
		rg.currentDepth--
	}
}

func (rg *protoRecursionGuard) CurrentDepth() int {
	return rg.currentDepth
}

func (rg *protoRecursionGuard) MaxDepth() int {
	return rg.maxDepth
}

// isPartialMatchError проверяет, является ли ошибка результатом частичного совпадения
// когда обработчик смог распознать часть конструкции, но не смог завершить
func isPartialMatchError(errMsg string) bool {
	// Ошибки, которые indicate частичное совпадение с конструкцией вызова функции
	partialMatchErrors := []string{
		"unexpected EOF after argument",
		"expected ',' or ')' after argument",
		"expected ')' after arguments",
		"unsupported argument type:",
	}

	for _, pattern := range partialMatchErrors {
		if contains(errMsg, pattern) {
			return true
		}
	}
	return false
}

// contains проверяет, содержит ли строка подстроку
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr ||
		len(s) > len(substr) && findSubstring(s, substr)
}

// findSubstring ищет подстроку в строке
func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
