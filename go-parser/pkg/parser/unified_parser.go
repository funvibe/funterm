package parser

import (
	"fmt"
	"go-parser/pkg/ast"
	"go-parser/pkg/common"
	"go-parser/pkg/config"
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
	// PipeHandler будет обрабатывать токены языков и идентификаторы с высоким приоритетом для проверки |
	pipeConfig := config.ConstructHandlerConfig{
		ConstructType: common.ConstructPipe,
		Name:          "pipe-expression",
		Priority:      95, // Высокий приоритет для проверки pipe expressions
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

	// Регистрируем QualifiedVariable обработчик (низкий приоритет для простых переменных)
	qualifiedVarConfig := config.ConstructHandlerConfig{
		ConstructType: common.ConstructVariable,
		Name:          "qualified-variable",
		Priority:      10, // Низкий приоритет
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

	// Регистрируем NumericForLoop обработчик для Lua циклов (более высокий приоритет)
	numericForLoopConfig := config.ConstructHandlerConfig{
		ConstructType: common.ConstructNumericForLoop,
		Name:          "numeric-for-loop",
		Priority:      90, // Высокий приоритет для Lua циклов
		Order:         1,
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
		Order:         2,
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
			{TokenType: 39, Offset: 0}, // TokenIf = 39 (реальное значение из отладки)
		},
	}

	ifHandler := handler.NewIfHandlerWithVerbose(ifConfig, verbose)
	registry.RegisterConstructHandler(ifHandler, ifConfig)

	// Регистрируем Assignment обработчик для простых присваиваний
	assignmentConfig := config.ConstructHandlerConfig{
		ConstructType: common.ConstructAssignment,
		Name:          "assignment",
		Priority:      80, // Низкий приоритет, чтобы не мешать другим обработчикам
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
			{TokenType: lexer.TokenIdentifier, Offset: 0}, // Для выражений типа "a = b + c"
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
			parseErrors = append(parseErrors, ast.ParseError{
				Type:     ast.ErrorSyntax,
				Position: tokenToPosition(currentToken),
				Message:  fmt.Sprintf("no handler found for token: %s", currentToken.Value),
				Context:  input,
			})
			break
		}

		// 6. Пробуем каждый обработчик в порядке приоритета
		var lastErr error
		var result interface{}

		for i, h := range handlers {
			// 7. Создаем клон потока для каждого обработчика
			clonedStream := tokenStream.Clone()

			// 8. Проверяем, может ли обработчик обработать токен
			if p.verbose {
				fmt.Printf("DEBUG: UnifiedParser - trying handler: %s, CanHandle: %v\n", h.Name(), h.CanHandle(currentToken))
			}
			if !h.CanHandle(currentToken) {
				continue
			}

			// 9. Создаем контекст и вызываем обработчик с клоном потока
			ctx := &common.ParseContext{
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

			// Если QualifiedVariableHandler возвращает конкретную ошибку,
			// означающую, что это не квалифицированная переменная, то пробуем следующий обработчик
			if err.Error() == "not a qualified variable" {
				continue
			}

			// Если LanguageCallHandler возвращает конкретную ошибку для неполных выражений,
			// то это финальная ошибка, другие обработчики не пробуем
			if err.Error() == "unexpected EOF after argument" {
				lastErr = err
				break
			}

			// Если это первая ошибка (от самого высокоприоритетного обработчика),
			// и это ошибка о присваивании зарезервированному слову, сохраняем её и выходим
			if i == 0 && err.Error() == "cannot assign to reserved keyword 'lua'" ||
				i == 0 && err.Error() == "cannot assign to reserved keyword 'python'" ||
				i == 0 && err.Error() == "cannot assign to reserved keyword 'py'" {
				lastErr = err
				break
			}

			// Сохраняем последнюю ошибку
			lastErr = err
		}

		if lastErr != nil {
			// Все обработчики вернули ошибки
			parseErrors = append(parseErrors, ast.ParseError{
				Type:     ast.ErrorSyntax,
				Position: tokenToPosition(currentToken),
				Message:  lastErr.Error(),
				Context:  input,
			})
			break
		}

		// 10. Конвертируем результат в Statement
		if p.verbose {
			fmt.Printf("DEBUG: UnifiedParser converting result to Statement, result type: %T\n", result)
		}
		if statement, ok := result.(ast.Statement); ok {
			if p.verbose {
				fmt.Printf("DEBUG: UnifiedParser appending statement: %T\n", statement)
			}
			statements = append(statements, statement)
		} else if expression, ok := result.(ast.Expression); ok {
			// Если результат - Expression, оборачиваем его в LanguageCallStatement
			if langCall, ok := expression.(*ast.LanguageCall); ok {
				langCallStmt := ast.NewLanguageCallStatement(langCall, langCall.Position())
				statements = append(statements, langCallStmt)
			} else {
				// Для других типов выражений создаем обертку ExpressionStatement
				// Но в текущем AST у нас нет ExpressionStatement, поэтому используем LanguageCallStatement
				// Это временное решение, в будущем нужно добавить ExpressionStatement в AST
				pos := expression.Position()
				langCallStmt := ast.NewLanguageCallStatement(nil, pos) // Временное решение
				// Устанавливаем выражение как внутреннее поле (нужно будет модифицировать LanguageCallStatement)
				statements = append(statements, langCallStmt)
			}
		} else {
			parseErrors = append(parseErrors, ast.ParseError{
				Type:     ast.ErrorSemantic,
				Position: tokenToPosition(currentToken),
				Message:  "result is not a statement",
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

	// Этот код не должен достигаться, но на всякий случай
	return nil, []ast.ParseError{{
		Type:     ast.ErrorSyntax,
		Position: ast.Position{Line: 1, Column: 1, Offset: 0},
		Message:  "no statements parsed",
		Context:  input,
	}}
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
