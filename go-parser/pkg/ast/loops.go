package ast

import (
	"go-parser/pkg/lexer"
)

// LoopStatement интерфейс для всех типов циклов
type LoopStatement interface {
	Statement
	IsLoop() bool
}

// ForInLoopStatement представляет Python-style for-in цикл: for i in range(5): python.print(i)
type ForInLoopStatement struct {
	BaseNode
	Variable   *Identifier // переменная цикла
	Iterable   ProtoNode   // итерируемый объект (Expression или Statement)
	Body       []Statement // тело цикла
	ForToken   lexer.Token // токен 'for'
	InToken    lexer.Token // токен 'in'
	ColonToken lexer.Token // токен ':'
	Pos        Position    // позиция начала цикла
}

// NewForInLoopStatement создает новый узел for-in цикла
func NewForInLoopStatement(forToken, inToken, colonToken lexer.Token, variable *Identifier, iterable ProtoNode, body []Statement) *ForInLoopStatement {
	return &ForInLoopStatement{
		ForToken:   forToken,
		InToken:    inToken,
		ColonToken: colonToken,
		Variable:   variable,
		Iterable:   iterable,
		Body:       body,
		Pos:        tokenToPosition(forToken),
	}
}

// Type возвращает тип узла
func (n *ForInLoopStatement) Type() NodeType {
	return NodeForInLoop
}

// statementMarker реализует интерфейс Statement
func (n *ForInLoopStatement) statementMarker() {}

// Position возвращает позицию узла
func (n *ForInLoopStatement) Position() Position {
	return n.Pos
}

// ToMap преобразует узел в map для сериализации
func (n *ForInLoopStatement) ToMap() map[string]interface{} {
	body := make([]interface{}, len(n.Body))
	for i, stmt := range n.Body {
		body[i] = stmt.ToMap()
	}

	return map[string]interface{}{
		"type":     "for_in_loop",
		"variable": n.Variable.ToMap(),
		"iterable": n.Iterable.ToMap(),
		"body":     body,
		"position": n.Pos.ToMap(),
	}
}

// IsLoop реализует интерфейс LoopStatement
func (n *ForInLoopStatement) IsLoop() bool {
	return true
}

// NumericForLoopStatement представляет Lua-style числовой цикл: for i=1,5 { lua.print(i) }
type NumericForLoopStatement struct {
	BaseNode
	Variable *Identifier // переменная цикла
	Start    ProtoNode   // начальное значение
	End      ProtoNode   // конечное значение
	Step     ProtoNode   // шаг (опционально)
	Body     []Statement // тело цикла
	ForToken lexer.Token // токен 'for'
	DoToken  lexer.Token // токен 'do'
	EndToken lexer.Token // токен 'end'
	Pos      Position    // позиция начала цикла
}

// NewNumericForLoopStatement создает новый узел числового цикла
func NewNumericForLoopStatement(forToken, doToken, endToken lexer.Token, variable *Identifier, start, end, step ProtoNode, body []Statement) *NumericForLoopStatement {
	return &NumericForLoopStatement{
		ForToken: forToken,
		DoToken:  doToken,
		EndToken: endToken,
		Variable: variable,
		Start:    start,
		End:      end,
		Step:     step,
		Body:     body,
		Pos:      tokenToPosition(forToken),
	}
}

// Type возвращает тип узла
func (n *NumericForLoopStatement) Type() NodeType {
	return NodeNumericForLoop
}

// statementMarker реализует интерфейс Statement
func (n *NumericForLoopStatement) statementMarker() {}

// Position возвращает позицию узла
func (n *NumericForLoopStatement) Position() Position {
	return n.Pos
}

// ToMap преобразует узел в map для сериализации
func (n *NumericForLoopStatement) ToMap() map[string]interface{} {
	body := make([]interface{}, len(n.Body))
	for i, stmt := range n.Body {
		body[i] = stmt.ToMap()
	}

	result := map[string]interface{}{
		"type":     "numeric_for_loop",
		"variable": n.Variable.ToMap(),
		"start":    n.Start.ToMap(),
		"end":      n.End.ToMap(),
		"body":     body,
		"position": n.Pos.ToMap(),
	}

	if n.Step != nil {
		result["step"] = n.Step.ToMap()
	}

	return result
}

// IsLoop реализует интерфейс LoopStatement
func (n *NumericForLoopStatement) IsLoop() bool {
	return true
}

// CStyleForLoopStatement представляет C-style for цикл: for (init; condition; increment) { ... }
type CStyleForLoopStatement struct {
	BaseNode
	Initializer Statement   // инициализация (может быть nil)
	Condition   Expression  // условие (может быть nil)
	Increment   Expression  // инкремент (может быть nil)
	Body        []Statement // тело цикла
	ForToken    lexer.Token // токен 'for'
	LParenToken lexer.Token // токен '('
	RParenToken lexer.Token // токен ')'
	LBraceToken lexer.Token // токен '{'
	RBraceToken lexer.Token // токен '}'
	Pos         Position    // позиция начала цикла
}

// NewCStyleForLoopStatement создает новый узел C-style for цикла
func NewCStyleForLoopStatement(forToken, lParenToken, rParenToken, lBraceToken, rBraceToken lexer.Token, initializer Statement, condition Expression, increment Expression, body []Statement) *CStyleForLoopStatement {
	return &CStyleForLoopStatement{
		ForToken:    forToken,
		LParenToken: lParenToken,
		RParenToken: rParenToken,
		LBraceToken: lBraceToken,
		RBraceToken: rBraceToken,
		Initializer: initializer,
		Condition:   condition,
		Increment:   increment,
		Body:        body,
		Pos:         tokenToPosition(forToken),
	}
}

// Type возвращает тип узла
func (n *CStyleForLoopStatement) Type() NodeType {
	return NodeCStyleForLoop
}

// statementMarker реализует интерфейс Statement
func (n *CStyleForLoopStatement) statementMarker() {}

// Position возвращает позицию узла
func (n *CStyleForLoopStatement) Position() Position {
	return n.Pos
}

// ToMap преобразует узел в map для сериализации
func (n *CStyleForLoopStatement) ToMap() map[string]interface{} {
	body := make([]interface{}, len(n.Body))
	for i, stmt := range n.Body {
		body[i] = stmt.ToMap()
	}

	result := map[string]interface{}{
		"type":     "c_style_for_loop",
		"body":     body,
		"position": n.Pos.ToMap(),
	}

	if n.Initializer != nil {
		result["initializer"] = n.Initializer.ToMap()
	}

	if n.Condition != nil {
		result["condition"] = n.Condition.ToMap()
	}

	if n.Increment != nil {
		result["increment"] = n.Increment.ToMap()
	}

	return result
}

// IsLoop реализует интерфейс LoopStatement
func (n *CStyleForLoopStatement) IsLoop() bool {
	return true
}

// WhileStatement представляет цикл 'while <condition> { ... }'
type WhileStatement struct {
	BaseNode
	Condition   Expression      // Условие цикла
	Body        *BlockStatement // Тело цикла (список стейтментов)
	WhileToken  lexer.Token     // токен 'while'
	LParenToken lexer.Token     // токен '('
	RParenToken lexer.Token     // токен ')'
	LBraceToken lexer.Token     // токен '{'
	RBraceToken lexer.Token     // токен '}'
	Pos         Position        // позиция начала цикла
}

// NewWhileStatement создает новый узел while цикла
func NewWhileStatement(whileToken, lParenToken, rParenToken lexer.Token, condition Expression, body *BlockStatement) *WhileStatement {
	return &WhileStatement{
		WhileToken:  whileToken,
		LParenToken: lParenToken,
		RParenToken: rParenToken,
		Condition:   condition,
		Body:        body,
		Pos:         tokenToPosition(whileToken),
	}
}

// Type возвращает тип узла
func (n *WhileStatement) Type() NodeType {
	return NodeWhileLoop
}

// statementMarker реализует интерфейс Statement
func (n *WhileStatement) statementMarker() {}

// Position возвращает позицию узла
func (n *WhileStatement) Position() Position {
	return n.Pos
}

// ToMap преобразует узел в map для сериализации
func (n *WhileStatement) ToMap() map[string]interface{} {
	body := make([]interface{}, len(n.Body.Statements))
	for i, stmt := range n.Body.Statements {
		body[i] = stmt.ToMap()
	}

	return map[string]interface{}{
		"type":      "while_loop",
		"condition": n.Condition.ToMap(),
		"body":      body,
		"position":  n.Pos.ToMap(),
	}
}

// IsLoop реализует интерфейс LoopStatement
func (n *WhileStatement) IsLoop() bool {
	return true
}

// BreakStatement представляет оператор 'break'
type BreakStatement struct {
	BaseNode
	BreakToken lexer.Token // токен 'break'
	Pos        Position    // позиция оператора
}

// NewBreakStatement создает новый узел break оператора
func NewBreakStatement(breakToken lexer.Token) *BreakStatement {
	return &BreakStatement{
		BreakToken: breakToken,
		Pos:        tokenToPosition(breakToken),
	}
}

// Type возвращает тип узла
func (n *BreakStatement) Type() NodeType {
	return NodeBreak
}

// statementMarker реализует интерфейс Statement
func (n *BreakStatement) statementMarker() {}

// Position возвращает позицию узла
func (n *BreakStatement) Position() Position {
	return n.Pos
}

// ToMap преобразует узел в map для сериализации
func (n *BreakStatement) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"type":     "break",
		"position": n.Pos.ToMap(),
	}
}

// ContinueStatement представляет оператор 'continue'
type ContinueStatement struct {
	BaseNode
	ContinueToken lexer.Token // токен 'continue'
	Pos           Position    // позиция оператора
}

// NewContinueStatement создает новый узел continue оператора
func NewContinueStatement(continueToken lexer.Token) *ContinueStatement {
	return &ContinueStatement{
		ContinueToken: continueToken,
		Pos:           tokenToPosition(continueToken),
	}
}

// Type возвращает тип узла
func (n *ContinueStatement) Type() NodeType {
	return NodeContinue
}

// statementMarker реализует интерфейс Statement
func (n *ContinueStatement) statementMarker() {}

// Position возвращает позицию узла
func (n *ContinueStatement) Position() Position {
	return n.Pos
}

// ToMap преобразует узел в map для сериализации
func (n *ContinueStatement) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"type":     "continue",
		"position": n.Pos.ToMap(),
	}
}

// BlockStatement представляет блок стейтментов в фигурных скобках
type BlockStatement struct {
	BaseNode
	Statements  []Statement // список стейтментов в блоке
	LBraceToken lexer.Token // токен '{'
	RBraceToken lexer.Token // токен '}'
	Pos         Position    // позиция начала блока
}

// NewBlockStatement создает новый узел блока стейтментов
func NewBlockStatement(lBraceToken, rBraceToken lexer.Token, statements []Statement) *BlockStatement {
	return &BlockStatement{
		LBraceToken: lBraceToken,
		RBraceToken: rBraceToken,
		Statements:  statements,
		Pos:         tokenToPosition(lBraceToken),
	}
}

// Type возвращает тип узла
func (n *BlockStatement) Type() NodeType {
	return NodeProgram // временно используем NodeProgram, можно добавить отдельный тип
}

// statementMarker реализует интерфейс Statement
func (n *BlockStatement) statementMarker() {}

// Position возвращает позицию узла
func (n *BlockStatement) Position() Position {
	return n.Pos
}

// ToMap преобразует узел в map для сериализации
func (n *BlockStatement) ToMap() map[string]interface{} {
	statements := make([]interface{}, len(n.Statements))
	for i, stmt := range n.Statements {
		statements[i] = stmt.ToMap()
	}

	return map[string]interface{}{
		"type":       "block",
		"statements": statements,
		"position":   n.Pos.ToMap(),
	}
}

// tokenToPosition конвертирует токен в позицию AST
func tokenToPosition(token lexer.Token) Position {
	return Position{
		Line:   token.Line,
		Column: token.Column,
		Offset: token.Position,
	}
}
