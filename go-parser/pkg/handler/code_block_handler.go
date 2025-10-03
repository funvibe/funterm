package handler

import (
	"fmt"
	"go-parser/pkg/ast"
	"go-parser/pkg/common"
	"go-parser/pkg/config"
	"go-parser/pkg/lexer"
	"go-parser/pkg/stream"
	"strings"
)

// CodeBlockHandler - обработчик для блоков кода: lua { ... } и python { ... }
type CodeBlockHandler struct {
	config  config.ConstructHandlerConfig
	verbose bool
}

// NewCodeBlockHandler создает новый обработчик кодовых блоков
func NewCodeBlockHandler(config config.ConstructHandlerConfig) *CodeBlockHandler {
	return NewCodeBlockHandlerWithVerbose(config, false)
}

// NewCodeBlockHandlerWithVerbose создает новый обработчик кодовых блоков с поддержкой verbose режима
func NewCodeBlockHandlerWithVerbose(config config.ConstructHandlerConfig, verbose bool) *CodeBlockHandler {
	return &CodeBlockHandler{
		config:  config,
		verbose: verbose,
	}
}

// CanHandle проверяет, может ли обработчик обработать токен
func (h *CodeBlockHandler) CanHandle(token lexer.Token) bool {
	return token.Type == lexer.TokenLua || token.Type == lexer.TokenPython || token.Type == lexer.TokenPy || token.Type == lexer.TokenGo || token.Type == lexer.TokenNode || token.Type == lexer.TokenJS
}

// skipWhitespaceTokens пропускает пробельные токены (переносы строк)
func (h *CodeBlockHandler) skipWhitespaceTokens(tokenStream stream.TokenStream) {
	for tokenStream.HasMore() {
		current := tokenStream.Current()
		if current.Type == lexer.TokenNewline {
			tokenStream.Consume()
		} else {
			break
		}
	}
}

// Handle обрабатывает блок кода
func (h *CodeBlockHandler) Handle(ctx *common.ParseContext) (interface{}, error) {
	tokenStream := ctx.TokenStream

	if h.verbose {
		fmt.Printf("DEBUG: CodeBlockHandler.Handle called with token: %+v\n", tokenStream.Current())
		fmt.Printf("DEBUG: CodeBlockHandler - InputStream length: %d\n", len(ctx.InputStream))
	}

	// Потребляем токен рантайма
	runtimeToken := tokenStream.Current()

	if runtimeToken.Type != lexer.TokenLua && runtimeToken.Type != lexer.TokenPython && runtimeToken.Type != lexer.TokenPy && runtimeToken.Type != lexer.TokenGo && runtimeToken.Type != lexer.TokenNode && runtimeToken.Type != lexer.TokenJS {
		return nil, fmt.Errorf("expected runtime token (lua, python, py, go, node, js), got %s", runtimeToken.Type)
	}
	if h.verbose {
		fmt.Printf("DEBUG: CodeBlockHandler - consuming runtime token\n")
	}
	tokenStream.Consume()

	// Пропускаем пробелы после рантайма
	if !tokenStream.HasMore() {
		return nil, fmt.Errorf("unexpected EOF after runtime specifier")
	}
	current := tokenStream.Current()
	if h.verbose {
		fmt.Printf("DEBUG: CodeBlockHandler - current token after runtime: %+v\n", current)
	}

	// Проверяем, не является ли следующий токен точкой (для доступа к функциям рантайма)
	if current.Type == lexer.TokenDot {
		// Это не блок кода, а доступ к функциям рантайма, например lua.math.sin
		// Возвращаем специальную ошибку, чтобы другие обработчики могли попробовать
		return nil, fmt.Errorf("not a code block statement")
	}

	// Проверяем, есть ли скобки с переменными
	var variableTokens []lexer.Token
	var lParenToken, rParenToken lexer.Token

	// Убираем проверку, которая препятствует обработке синтаксиса lua (vars) { ... }
	// if current.Type == lexer.TokenLeftParen {
	// 	return nil, fmt.Errorf("not a code block statement")
	// }

	if current.Type == lexer.TokenLeftParen {
		// Новый синтаксис: py (var1, var2) { ... }
		if h.verbose {
			fmt.Printf("DEBUG: CodeBlockHandler - found LPAREN, parsing variables\n")
		}
		lParenToken = current
		tokenStream.Consume()

		// Читаем переменные до закрывающей скобки
		for tokenStream.HasMore() && tokenStream.Current().Type != lexer.TokenRightParen {
			current = tokenStream.Current()
			if h.verbose {
				fmt.Printf("DEBUG: CodeBlockHandler - parsing variable token: %+v\n", current)
			}

			// Если сразу после открывающей скобки идет закрывающая - это пустой список переменных
			if current.Type == lexer.TokenRightParen {
				break
			}

			if current.Type == lexer.TokenIdentifier {
				variableTokens = append(variableTokens, current)
				tokenStream.Consume()

				// Пропускаем запятые и переносы строк
				for tokenStream.HasMore() {
					nextToken := tokenStream.Current()
					if nextToken.Type == lexer.TokenComma {
						tokenStream.Consume()
						// Пропускаем переносы строки после запятой
						h.skipWhitespaceTokens(tokenStream)
					} else if nextToken.Type == lexer.TokenNewline {
						tokenStream.Consume()
					} else {
						break
					}
				}
			} else if current.Type == lexer.TokenNewline {
				// Пропускаем переносы строк
				tokenStream.Consume()
			} else {
				return nil, fmt.Errorf("expected variable name or ')', got %s", current.Type)
			}
		}

		// Проверяем закрывающую скобку
		if !tokenStream.HasMore() || tokenStream.Current().Type != lexer.TokenRightParen {
			return nil, fmt.Errorf("expected ')' after variable list")
		}
		rParenToken = tokenStream.Current()
		if h.verbose {
			fmt.Printf("DEBUG: CodeBlockHandler - found RPAREN, consuming it\n")
		}
		tokenStream.Consume()

		// Пропускаем переносы строк после скобок
		h.skipWhitespaceTokens(tokenStream)
		current = tokenStream.Current()
		if h.verbose {
			fmt.Printf("DEBUG: CodeBlockHandler - current token after variables: %+v\n", current)
		}
	} else {
		if h.verbose {
			fmt.Printf("DEBUG: CodeBlockHandler - no variables, expecting LBRACE\n")
		}
		// Пропускаем переносы строк перед {
		h.skipWhitespaceTokens(tokenStream)
		current = tokenStream.Current()
		if h.verbose {
			fmt.Printf("DEBUG: CodeBlockHandler - current token after skipping newlines: %+v\n", current)
		}
	}

	// Ожидаем открывающую фигурную скобку
	if current.Type != lexer.TokenLBrace {
		return nil, fmt.Errorf("expected '{' after runtime specifier, got %s", current.Type)
	}
	if h.verbose {
		fmt.Printf("DEBUG: CodeBlockHandler - found LBRACE, consuming it\n")
	}
	lBraceToken := current
	tokenStream.Consume()

	// Ищем закрывающую скобку с подсчетом вложенности
	braceLevel := 1 // Мы уже потребили открывающую скобку
	var rBraceToken lexer.Token
	if h.verbose {
		fmt.Printf("DEBUG: CodeBlockHandler - starting brace search, braceLevel: %d, HasMore: %v\n", braceLevel, tokenStream.HasMore())
	}

	for braceLevel > 0 && tokenStream.HasMore() {
		current = tokenStream.Current()
		if h.verbose {
			fmt.Printf("DEBUG: CodeBlockHandler - in loop, current token: %+v, braceLevel: %d\n", current, braceLevel)
		}

		// Проверяем, не нашли ли мы закрывающую скобку на текущем шаге
		if current.Type == lexer.TokenRBrace && braceLevel == 1 {
			// Это наша закрывающая скобка, потребляем ее и выходим
			tokenStream.Consume()
			rBraceToken = current
			braceLevel = 0
			if h.verbose {
				fmt.Printf("DEBUG: CodeBlockHandler - found closing brace, braceLevel now: %d\n", braceLevel)
			}
			break
		}

		tokenStream.Consume()

		switch current.Type {
		case lexer.TokenLBrace:
			braceLevel++
			if h.verbose {
				fmt.Printf("DEBUG: CodeBlockHandler - found LBRACE, braceLevel now: %d\n", braceLevel)
			}
		case lexer.TokenRBrace:
			braceLevel--
			if h.verbose {
				fmt.Printf("DEBUG: CodeBlockHandler - found RBRACE, braceLevel now: %d\n", braceLevel)
			}
		case lexer.TokenEOF:
			return nil, fmt.Errorf("unexpected EOF while reading code block")
		}
	}
	if h.verbose {
		fmt.Printf("DEBUG: CodeBlockHandler - exited brace search loop, braceLevel: %d\n", braceLevel)
	}

	if braceLevel != 0 {
		return nil, fmt.Errorf("unclosed code block")
	}

	// Начало нашего блока кода - это позиция СРАЗУ ПОСЛЕ '{'
	codeStartPosition := lBraceToken.Position + len(lBraceToken.Value)

	// Конец нашего блока кода - это позиция ПЕРЕД '}'
	codeEndPosition := rBraceToken.Position

	// Проверяем, что позиции валидны
	if codeStartPosition < 0 || codeEndPosition > len(ctx.InputStream) || codeStartPosition > codeEndPosition {
		return nil, fmt.Errorf("invalid code block positions")
	}

	// Извлекаем оригинальный код из исходной строки
	rawCode := ctx.InputStream[codeStartPosition:codeEndPosition]

	if h.verbose {
		fmt.Printf("DEBUG: CodeBlockHandler - raw code before processing: %q\n", rawCode)
	}

	// Разделяем на строки и удаляем пустые строки в начале и в конце
	lines := strings.Split(rawCode, "\n")

	// Удаляем пустые строки в начале
	for len(lines) > 0 && strings.TrimSpace(lines[0]) == "" {
		lines = lines[1:]
	}

	// Удаляем пустые строки в конце
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}

	if len(lines) > 0 {
		if h.verbose {
			fmt.Printf("DEBUG: CodeBlockHandler - processing %d lines for indentation\n", len(lines))
			for i, line := range lines {
				fmt.Printf("DEBUG: CodeBlockHandler - line %d: %q\n", i, line)
			}
		}

		// Находим минимальный отступ (количество пробелов в начале) для непустых строк
		minIndent := -1
		for _, line := range lines {
			if strings.TrimSpace(line) == "" {
				continue // Пропускаем пустые строки
			}
			indent := 0
			for _, r := range line {
				if r == ' ' {
					indent++
				} else {
					break
				}
			}
			if minIndent == -1 || indent < minIndent {
				minIndent = indent
			}
		}

		if h.verbose {
			fmt.Printf("DEBUG: CodeBlockHandler - minIndent: %d\n", minIndent)
		}

		// Удаляем минимальный отступ из всех непустых строк
		if minIndent > 0 {
			for i, line := range lines {
				if strings.TrimSpace(line) != "" && len(line) >= minIndent {
					lines[i] = line[minIndent:]
					if h.verbose {
						fmt.Printf("DEBUG: CodeBlockHandler - line %d after removing %d spaces: %q\n", i, minIndent, lines[i])
					}
				}
			}
		}

		rawCode = strings.Join(lines, "\n")

		if h.verbose {
			fmt.Printf("DEBUG: CodeBlockHandler - raw code after processing: %q\n", rawCode)
		}
	} else {
		rawCode = ""
	}
	if h.verbose {
		fmt.Printf("DEBUG: CodeBlockHandler - extracted raw code: %q\n", rawCode)
		fmt.Printf("DEBUG: CodeBlockHandler - codeStartPosition: %d, codeEndPosition: %d\n", codeStartPosition, codeEndPosition)
		fmt.Printf("DEBUG: CodeBlockHandler - raw code with visible whitespace:\n")
		for _, r := range rawCode {
			if r == ' ' {
				fmt.Printf(" ")
			} else if r == '\t' {
				fmt.Printf("\\t")
			} else if r == '\n' {
				fmt.Printf("\\n\n")
			} else {
				fmt.Printf("%c", r)
			}
		}
		fmt.Printf("\n")
	}

	// Создаем узел AST с чистым, нетронутым кодом
	codeBlockStmt := ast.NewCodeBlockStatement(runtimeToken, variableTokens, lParenToken, rParenToken, lBraceToken, rBraceToken, rawCode)
	if h.verbose {
		fmt.Printf("DEBUG: CodeBlockHandler - created CodeBlockStatement successfully\n")
		if len(variableTokens) > 0 {
			fmt.Printf("DEBUG: CodeBlockHandler - variables to save: %v\n", codeBlockStmt.GetVariableNames())
		}
	}

	return codeBlockStmt, nil
}

// Config возвращает конфигурацию обработчика
func (h *CodeBlockHandler) Config() common.HandlerConfig {
	return common.HandlerConfig{
		IsEnabled: h.config.IsEnabled,
		Priority:  h.config.Priority,
		Name:      h.config.Name,
	}
}

// Name возвращает имя обработчика
func (h *CodeBlockHandler) Name() string {
	return h.config.Name
}
