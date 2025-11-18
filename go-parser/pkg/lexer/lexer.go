package lexer

type Lexer interface {
	NextToken() Token
	Peek() Token
	HasMore() bool
	Position() int
	SetInSizeExpression(bool)
}

type SimpleLexer struct {
	input            string
	position         int
	current          rune
	line             int
	column           int
	shebangChecked   bool
	inSizeExpression bool
}

func NewLexer(input string) *SimpleLexer {
	l := &SimpleLexer{
		input:            input,
		line:             1,
		column:           0,
		shebangChecked:   false,
		inSizeExpression: false,
	}
	l.readChar()
	return l
}

func (l *SimpleLexer) readChar() {
	if l.position >= len(l.input) {
		l.current = 0 // EOF
	} else {
		l.current = rune(l.input[l.position])
	}
	l.position++
	l.column++
	if l.current == '\n' {
		l.line++
		l.column = 0
	}
}

func (l *SimpleLexer) peekChar() rune {
	if l.position >= len(l.input) {
		return 0 // EOF
	}
	return rune(l.input[l.position])
}

func (l *SimpleLexer) peekNext() rune {
	if l.position+1 >= len(l.input) {
		return 0 // EOF
	}
	return rune(l.input[l.position+1])
}

func (l *SimpleLexer) NextToken() Token {
	// Check for shebang only on the first call
	if !l.shebangChecked {
		l.skipShebang()
		l.shebangChecked = true
	}

	l.skipWhitespaceAndComments()

	token := Token{
		Position: l.position - 1,
		Line:     l.line,
		Column:   l.column,
	}

	switch l.current {
	case 0:
		token.Type = TokenEOF
		token.Value = ""
	case '\n':
		token.Type = TokenNewline
		token.Value = "\n"
		l.readChar()
		return token
	case '(':
		token.Type = TokenLeftParen
		token.Value = "("
	case ')':
		token.Type = TokenRightParen
		token.Value = ")"
	case '<':
		// Проверяем на <=
		if l.peekChar() == '=' {
			l.readChar() // потребляем '='
			token.Type = TokenLessEqual
			token.Value = "<="
			l.readChar()
			return token
		}
		// Проверяем на <<
		if l.peekChar() == '<' {
			l.readChar() // потребляем второй '<'
			token.Type = TokenDoubleLeftAngle
			token.Value = "<<"
			l.readChar()
			return token
		}
		token.Type = TokenLess
		token.Value = "<"
		l.readChar()
		return token
	case '>':
		// Проверяем на >=
		if l.peekChar() == '=' {
			l.readChar() // потребляем '='
			token.Type = TokenGreaterEqual
			token.Value = ">="
			l.readChar()
			return token
		}
		// Проверяем на >>
		if l.peekChar() == '>' {
			l.readChar() // потребляем второй '>'
			token.Type = TokenDoubleRightAngle
			token.Value = ">>"
			l.readChar()
			return token
		}
		token.Type = TokenGreater
		token.Value = ">"
		l.readChar()
		return token
	case '/':
		token.Type = TokenSlash
		token.Value = "/"
		l.readChar()
		return token
	case '[':
		token.Type = TokenLBracket
		token.Value = "["
		l.readChar()
		return token
	case ']':
		token.Type = TokenRBracket
		token.Value = "]"
		l.readChar()
		return token
	case '{':
		token.Type = TokenLBrace
		token.Value = "{"
		l.readChar()
		return token
	case '}':
		token.Type = TokenRBrace
		token.Value = "}"
		l.readChar()
		return token
	case '=':
		// Проверяем на ==
		if l.peekChar() == '=' {
			l.readChar() // потребляем второй '='
			token.Type = TokenEqual
			token.Value = "=="
			l.readChar()
			return token
		}
		token.Type = TokenAssign
		token.Value = "="
		l.readChar()
		return token
	case ':':
		// Проверяем на := (mutable assignment)
		if l.peekChar() == '=' {
			l.readChar() // потребляем '='
			token.Type = TokenColonEquals
			token.Value = ":="
			l.readChar()
			return token
		}
		token.Type = TokenColon
		token.Value = ":"
		l.readChar()
		return token
	case '.':
		// Проверяем на оператор ...
		if l.peekChar() == '.' && l.position+1 < len(l.input) && l.input[l.position+1] == '.' {
			l.readChar() // потребляем первую '.'
			l.readChar() // потребляем вторую '.'
			token.Type = TokenRest
			token.Value = "..."
			l.readChar()
			return token
		}
		token.Type = TokenDot
		token.Value = "."
		l.readChar()
		return token
	case ',':
		token.Type = TokenComma
		token.Value = ","
		l.readChar()
		return token
	case ';':
		token.Type = TokenSemicolon
		token.Value = ";"
		l.readChar()
		return token
	case '|':
		// Проверяем на |> (pipe operator)
		if l.peekChar() == '>' {
			l.readChar() // потребляем '>'
			token.Type = TokenPipe
			token.Value = "|>"
			l.readChar()
			return token
		}
		// Проверяем на ||
		if l.peekChar() == '|' {
			l.readChar() // потребляем второй '|'
			token.Type = TokenOr
			token.Value = "||"
			l.readChar()
			return token
		}
		token.Type = TokenBitwiseOr
		token.Value = "|"
		l.readChar()
		return token
	case '&':
		// Проверяем на &&
		if l.peekChar() == '&' {
			l.readChar() // потребляем второй '&'
			token.Type = TokenAnd
			token.Value = "&&"
			l.readChar()
			return token
		}
		token.Type = TokenAmpersand
		token.Value = "&"
		l.readChar()
		return token
	case '!':
		// Проверяем на !=
		if l.peekChar() == '=' {
			l.readChar() // потребляем '='
			token.Type = TokenNotEqual
			token.Value = "!="
			l.readChar()
			return token
		}
		token.Type = TokenNot
		token.Value = "!"
		l.readChar()
		return token
	case '~':
		token.Type = TokenTilde
		token.Value = "~"
		l.readChar()
		return token
	case '^':
		token.Type = TokenCaret
		token.Value = "^"
		l.readChar()
		return token
	case '+':
		// Проверяем на ++
		if l.peekChar() == '+' {
			l.readChar() // потребляем второй '+'
			token.Type = TokenConcat
			token.Value = "++"
			l.readChar()
			return token
		}
		token.Type = TokenPlus
		token.Value = "+"
		l.readChar()
		return token
	case '-':
		// Проверяем на оператор ->
		if l.peekChar() == '>' {
			l.readChar() // потребляем '>'
			token.Type = TokenArrow
			token.Value = "->"
			l.readChar()
			return token
		}
		token.Type = TokenMinus
		token.Value = "-"
		l.readChar()
		return token
	case '*':
		// Проверяем на ** (exponentiation)
		if l.peekChar() == '*' {
			l.readChar() // потребляем второй '*'
			token.Type = TokenPower
			token.Value = "**"
			l.readChar()
			return token
		}
		token.Type = TokenMultiply
		token.Value = "*"
		l.readChar()
		return token
	case '%':
		token.Type = TokenModulo
		token.Value = "%"
		l.readChar()
		return token
	case '?':
		token.Type = TokenQuestion
		token.Value = "?"
		l.readChar()
		return token
	case '@':
		token.Type = TokenAt
		token.Value = "@"
		l.readChar()
		return token
	case '"':
		// Check for triple double quotes
		if l.peekChar() == '"' && l.position+1 < len(l.input) && l.input[l.position+1] == '"' {
			return l.readMultilineString('"')
		}
		return l.readString()
	case '\'':
		// Check for triple single quotes
		if l.peekChar() == '\'' && l.position+1 < len(l.input) && l.input[l.position+1] == '\'' {
			return l.readMultilineString('\'')
		}
		return l.readString()
	default:
		if isDigit(l.current) {
			return l.readNumber()
		}
		if isLetter(l.current) {
			return l.readIdentifier()
		}
		if l.current == '_' {
			// Check if underscore is part of an identifier or standalone
			if l.position < len(l.input) && (isLetter(rune(l.input[l.position])) || isDigit(rune(l.input[l.position])) || rune(l.input[l.position]) == '_') {
				// Start of an identifier with underscore, read as identifier
				return l.readIdentifierWithUnderscore()
			}
			// Standalone underscore, treat as wildcard token
			token.Type = TokenUnderscore
			token.Value = "_"
			l.readChar()
			return token
		}
		token.Type = TokenUnknown
		token.Value = string(l.current)
	}

	l.readChar()
	return token
}

func (l *SimpleLexer) Peek() Token {
	currentPos := l.position
	currentChar := l.current
	currentLine := l.line
	currentColumn := l.column
	currentShebangChecked := l.shebangChecked

	token := l.NextToken()

	l.position = currentPos
	l.current = currentChar
	l.line = currentLine
	l.column = currentColumn
	l.shebangChecked = currentShebangChecked

	return token
}

func (l *SimpleLexer) HasMore() bool {
	return l.current != 0
}

func (l *SimpleLexer) Position() int {
	return l.position - 1
}

// SetInSizeExpression устанавливает флаг контекста size expression
func (l *SimpleLexer) SetInSizeExpression(inSizeExpr bool) {
	l.inSizeExpression = inSizeExpr
}

func (l *SimpleLexer) skipWhitespace() {
	for l.current == ' ' || l.current == '\t' || l.current == '\r' {
		l.readChar()
	}
}

func (l *SimpleLexer) skipWhitespaceAndComments() {
	for {
		// Skip whitespace (but not newlines - they are now tokens)
		for l.current == ' ' || l.current == '\t' || l.current == '\r' {
			l.readChar()
		}

		// Skip block comments
		if l.current == '/' && l.peekChar() == '*' {
			l.skipBlockComment()
			continue
		}

		// Skip single-line comments starting with #
		if l.current == '#' {
			l.skipSingleLineComment()
			continue
		}

		// Skip single-line comments starting with //
		if l.current == '/' && l.peekChar() == '/' {
			l.skipSingleLineComment()
			continue
		}

		// Skip single-line comments starting with -- (Lua style)
		if l.current == '-' && l.peekChar() == '-' {
			l.skipSingleLineComment()
			continue
		}

		// If we're not at a comment, we're done
		break
	}
}

func (l *SimpleLexer) skipBlockComment() {
	// Consume the opening /*
	l.readChar() // consume '/'
	l.readChar() // consume '*'

	for {
		if l.current == 0 {
			// Unexpected EOF
			return
		}

		if l.current == '*' && l.peekChar() == '/' {
			// Found the end of the block comment
			l.readChar() // consume '*'
			l.readChar() // consume '/'
			return
		}

		l.readChar()
	}
}

func (l *SimpleLexer) skipSingleLineComment() {
	// Skip until we find a newline or EOF
	for l.current != '\n' && l.current != '\r' && l.current != 0 {
		l.readChar()
	}
	// Also consume the newline character if it exists
	if l.current == '\n' || l.current == '\r' {
		l.readChar()
	}
}
func (l *SimpleLexer) skipShebang() {
	// Check if we're at the very beginning of the file
	if l.position != 1 {
		return
	}

	// Check if we have #! pattern
	if l.current == '#' && l.position < len(l.input) && l.input[l.position] == '!' {
		// Consume the # and !
		l.readChar() // consume #
		l.readChar() // consume !

		// Skip until the end of the line
		for l.current != '\n' && l.current != '\r' && l.current != 0 {
			l.readChar()
		}
		// Also skip the newline character if it exists
		if l.current == '\n' || l.current == '\r' {
			l.readChar()
		}
	}
}

func isLetter(ch rune) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')
}

func isLetterOrDigit(ch rune) bool {
	return isLetter(ch) || (ch >= '0' && ch <= '9') || ch == '_' || ch == '-'
}

func isDigit(ch rune) bool {
	return ch >= '0' && ch <= '9'
}

func isHexDigit(ch rune) bool {
	return (ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')
}

func (l *SimpleLexer) readIdentifier() Token {
	startPos := l.position - 1
	startLine := l.line
	startCol := l.column - 1

	// Первый символ должен быть буквой
	if !isLetter(l.current) {
		// Это не должно произойти, так как мы проверяем в NextToken
		return Token{
			Type:     TokenUnknown,
			Value:    string(l.current),
			Position: startPos,
			Line:     startLine,
			Column:   startCol,
		}
	}
	l.readChar()

	// В режиме size expression не включаем дефис в идентификаторы
	if l.inSizeExpression {
		// Последующие символы могут быть буквами, цифрами или подчеркиванием (без дефиса)
		for isLetter(l.current) || isDigit(l.current) || l.current == '_' {
			l.readChar()
		}
	} else {
		// Последующие символы могут быть буквами, цифрами, подчеркиванием или дефисом
		for isLetterOrDigit(l.current) {
			l.readChar()
		}
	}

	identifier := l.input[startPos : l.position-1]

	// Проверяем на ключевые слова
	switch identifier {
	case "for":
		return Token{
			Type:     TokenFor,
			Value:    identifier,
			Position: startPos,
			Line:     startLine,
			Column:   startCol,
		}
	case "in":
		return Token{
			Type:     TokenIn,
			Value:    identifier,
			Position: startPos,
			Line:     startLine,
			Column:   startCol,
		}
	case "while":
		return Token{
			Type:     TokenWhile,
			Value:    identifier,
			Position: startPos,
			Line:     startLine,
			Column:   startCol,
		}
	case "if":
		return Token{
			Type:     TokenIf,
			Value:    identifier,
			Position: startPos,
			Line:     startLine,
			Column:   startCol,
		}
	case "else":
		return Token{
			Type:     TokenElse,
			Value:    identifier,
			Position: startPos,
			Line:     startLine,
			Column:   startCol,
		}
	case "break":
		return Token{
			Type:     TokenBreak,
			Value:    identifier,
			Position: startPos,
			Line:     startLine,
			Column:   startCol,
		}
	case "continue":
		return Token{
			Type:     TokenContinue,
			Value:    identifier,
			Position: startPos,
			Line:     startLine,
			Column:   startCol,
		}
	case "match":
		return Token{
			Type:     TokenMatch,
			Value:    identifier,
			Position: startPos,
			Line:     startLine,
			Column:   startCol,
		}
	case "lua":
		return Token{
			Type:     TokenLua,
			Value:    identifier,
			Position: startPos,
			Line:     startLine,
			Column:   startCol,
		}
	case "go":
		return Token{
			Type:     TokenGo,
			Value:    identifier,
			Position: startPos,
			Line:     startLine,
			Column:   startCol,
		}
	case "python":
		return Token{
			Type:     TokenPython,
			Value:    identifier,
			Position: startPos,
			Line:     startLine,
			Column:   startCol,
		}
	case "py":
		token := Token{
			Type:     TokenPy,
			Value:    identifier,
			Position: startPos,
			Line:     startLine,
			Column:   startCol,
		}
		return token
	case "node":
		return Token{
			Type:     TokenNode,
			Value:    identifier,
			Position: startPos,
			Line:     startLine,
			Column:   startCol,
		}
	case "js":
		return Token{
			Type:     TokenJS,
			Value:    identifier,
			Position: startPos,
			Line:     startLine,
			Column:   startCol,
		}
	case "import":
		return Token{
			Type:     TokenImport,
			Value:    identifier,
			Position: startPos,
			Line:     startLine,
			Column:   startCol,
		}
	case "true":
		return Token{
			Type:     TokenTrue,
			Value:    identifier,
			Position: startPos,
			Line:     startLine,
			Column:   startCol,
		}
	case "false":
		return Token{
			Type:     TokenFalse,
			Value:    identifier,
			Position: startPos,
			Line:     startLine,
			Column:   startCol,
		}
	case "nil":
		return Token{
			Type:     TokenNil,
			Value:    identifier,
			Position: startPos,
			Line:     startLine,
			Column:   startCol,
		}
	default:
		return Token{
			Type:     TokenIdentifier,
			Value:    identifier,
			Position: startPos,
			Line:     startLine,
			Column:   startCol,
		}
	}
}

func (l *SimpleLexer) readIdentifierWithUnderscore() Token {
	startPos := l.position - 1
	startLine := l.line
	startCol := l.column - 1

	// Первый символ должен быть '_'
	if l.current != '_' {
		// Это не должно произойти, так как мы проверяем в NextToken
		return Token{
			Type:     TokenUnknown,
			Value:    string(l.current),
			Position: startPos,
			Line:     startLine,
			Column:   startCol,
		}
	}
	l.readChar()

	// Последующие символы могут быть буквами, цифрами или подчеркиваниями
	for isLetter(rune(l.input[l.position])) || isDigit(rune(l.input[l.position])) || rune(l.input[l.position]) == '_' {
		l.readChar()
		if l.position >= len(l.input) {
			break
		}
	}

	identifier := l.input[startPos : l.position-1]

	// Проверяем на ключевые слова (только те, которые могут начинаться с '_')
	// В большинстве случаев идентификаторы, начинающиеся с '_', не являются ключевыми словами
	return Token{
		Type:     TokenIdentifier,
		Value:    identifier,
		Position: startPos,
		Line:     startLine,
		Column:   startCol,
	}
}

func (l *SimpleLexer) readNumber() Token {
	startPos := l.position - 1
	startLine := l.line
	startCol := l.column - 1

	// Проверяем на шестнадцатеричное число (0x или 0X)
	if l.current == '0' && (l.peekChar() == 'x' || l.peekChar() == 'X') {
		l.readChar() // потребляем '0'
		l.readChar() // потребляем 'x' или 'X'

		// Читаем шестнадцатеричные цифры
		for isHexDigit(l.current) {
			l.readChar()
		}

		return Token{
			Type:     TokenNumber,
			Value:    l.input[startPos : l.position-1],
			Position: startPos,
			Line:     startLine,
			Column:   startCol,
		}
	}

	// Проверяем на двоичное число (0b или 0B)
	if l.current == '0' && (l.peekChar() == 'b' || l.peekChar() == 'B') {
		l.readChar() // потребляем '0'
		l.readChar() // потребляем 'b' или 'B'

		// Читаем двоичные цифры (только 0 и 1)
		for l.current == '0' || l.current == '1' {
			l.readChar()
		}

		return Token{
			Type:     TokenNumber,
			Value:    l.input[startPos : l.position-1],
			Position: startPos,
			Line:     startLine,
			Column:   startCol,
		}
	}

	for isDigit(l.current) {
		l.readChar()
	}

	// Проверяем на десятичную точку
	if l.current == '.' {
		l.readChar()
		for isDigit(l.current) {
			l.readChar()
		}
	}

	// Проверяем на научную нотацию (e или E)
	if l.current == 'e' || l.current == 'E' {
		l.readChar() // потребляем 'e' или 'E'

		// Проверяем на знак экспоненты (+ или -)
		if l.current == '+' || l.current == '-' {
			l.readChar() // потребляем знак
		}

		// Должна быть хотя бы одна цифра в экспоненте
		if !isDigit(l.current) {
			// Если после e/E (и возможного знака) нет цифры, это ошибка
			// Но для простоты вернем то что уже прочитали
			return Token{
				Type:     TokenNumber,
				Value:    l.input[startPos : l.position-1],
				Position: startPos,
				Line:     startLine,
				Column:   startCol,
			}
		}

		// Читаем цифры экспоненты
		for isDigit(l.current) {
			l.readChar()
		}
	}

	return Token{
		Type:     TokenNumber,
		Value:    l.input[startPos : l.position-1],
		Position: startPos,
		Line:     startLine,
		Column:   startCol,
	}
}

func (l *SimpleLexer) readString() Token {
	quote := l.current
	l.readChar() // Пропускаем открывающую кавычку

	startPos := l.position - 1
	startLine := l.line
	startCol := l.column - 1

	for l.current != quote && l.current != 0 {
		l.readChar()
	}

	if l.current == 0 {
		// Неожиданный EOF
		return Token{
			Type:     TokenUnknown,
			Value:    l.input[startPos : l.position-1],
			Position: startPos,
			Line:     startLine,
			Column:   startCol,
		}
	}

	rawValue := l.input[startPos : l.position-1]
	l.readChar() // Пропускаем закрывающую кавычку

	// Process escape sequences
	value := l.processEscapeSequences(rawValue)

	return Token{
		Type:     TokenString,
		Value:    value,
		Position: startPos,
		Line:     startLine,
		Column:   startCol,
	}
}

func (l *SimpleLexer) readMultilineString(quote rune) Token {
	// Consume the opening triple quotes
	l.readChar() // consume first quote
	l.readChar() // consume second quote
	l.readChar() // consume third quote

	startPos := l.position - 1 // Start of content (after triple quotes)
	startLine := l.line
	startCol := l.column - 1

	// Look for closing triple quotes
	for l.current != 0 {
		// First check for escape sequences - if we find a backslash,
		// skip both the backslash and the next character
		if l.current == '\\' {
			l.readChar() // consume the backslash
			if l.current != 0 {
				l.readChar() // consume the escaped character
			}
			continue
		}

		// Only check for closing triple quotes if we're not in an escape sequence
		if l.current == quote && l.peekChar() == quote && l.peekNext() == quote {
			// Found closing triple quotes
			endPos := l.position - 1
			if endPos < startPos {
				endPos = startPos
			}
			rawValue := l.input[startPos:endPos]

			// Consume the closing triple quotes
			l.readChar() // consume first quote
			l.readChar() // consume second quote
			l.readChar() // consume third quote

			// Process escape sequences according to test expectations
			value := l.processMultilineEscapeSequences(rawValue)

			return Token{
				Type:     TokenString,
				Value:    value,
				Position: startPos,
				Line:     startLine,
				Column:   startCol,
			}
		}

		l.readChar()
	}

	// Unexpected EOF
	return Token{
		Type:     TokenUnknown,
		Value:    l.input[startPos : l.position-1],
		Position: startPos,
		Line:     startLine,
		Column:   startCol,
	}
}

func (l *SimpleLexer) processEscapeSequences(input string) string {
	var result []rune

	// Convert input string to runes properly (handles UTF-8)
	inputRunes := []rune(input)

	for i := 0; i < len(inputRunes); i++ {
		if inputRunes[i] == '\\' && i+1 < len(inputRunes) {
			switch inputRunes[i+1] {
			case 'n':
				result = append(result, '\n')
				i++
			case 't':
				result = append(result, '\t')
				i++
			case 'r':
				result = append(result, '\r')
				i++
			case '\\':
				result = append(result, '\\')
				i++
			case '"':
				result = append(result, '"')
				i++
			case '\'':
				result = append(result, '\'')
				i++
			default:
				// If it's an unknown escape sequence, just keep the backslash and skip it
				// This handles cases like \" where we want to keep the quote but remove the backslash
				result = append(result, inputRunes[i+1])
				i++
			}
		} else {
			result = append(result, inputRunes[i])
		}
	}
	return string(result)
}

func (l *SimpleLexer) processMultilineEscapeSequences(input string) string {
	var result []rune

	// Convert input string to runes properly (handles UTF-8)
	inputRunes := []rune(input)

	for i := 0; i < len(inputRunes); i++ {
		if inputRunes[i] == '\\' && i+1 < len(inputRunes) {
			switch inputRunes[i+1] {
			case 'n':
				result = append(result, '\n')
				i++
			case 't':
				result = append(result, '\t')
				i++
			case 'r':
				result = append(result, '\r')
				i++
			case '"':
				result = append(result, '"')
				i++
			case '\'':
				result = append(result, '\'')
				i++
			case '\\':
				result = append(result, '\\')
				i++
			default:
				// For all other escape sequences, keep both the backslash and the next character
				result = append(result, '\\')
				result = append(result, inputRunes[i+1])
				i++
			}
		} else {
			result = append(result, inputRunes[i])
		}
	}
	return string(result)
}

func (l *SimpleLexer) processSpecificEscapeSequences(input string) string {
	var result []rune

	// Convert input string to runes properly (handles UTF-8)
	inputRunes := []rune(input)

	for i := 0; i < len(inputRunes); i++ {
		if inputRunes[i] == '\\' && i+1 < len(inputRunes) {
			switch inputRunes[i+1] {
			case 'n':
				result = append(result, '\n')
				i++
			case 't':
				result = append(result, '\t')
				i++
			case 'r':
				result = append(result, '\r')
				i++
			default:
				// For all other escape sequences, keep both the backslash and the next character
				result = append(result, '\\')
				result = append(result, inputRunes[i+1])
				i++
			}
		} else {
			result = append(result, inputRunes[i])
		}
	}
	return string(result)
}
