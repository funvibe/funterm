package ast

import (
	"fmt"
	"strings"

	"go-parser/pkg/lexer"
)

// VariableAssignment - узел для присваивания переменной
type VariableAssignment struct {
	BaseNode
	Variable *Identifier
	Assign   lexer.Token
	Value    Expression
}

// Position возвращает позицию узла в коде (реализация интерфейса Statement)
func (n *VariableAssignment) Position() Position {
	return n.Variable.Position()
}

// NewVariableAssignment создает новый узел присваивания переменной
func NewVariableAssignment(variable *Identifier, assign lexer.Token, value Expression) *VariableAssignment {
	return &VariableAssignment{
		Variable: variable,
		Assign:   assign,
		Value:    value,
	}
}

// Type возвращает тип узла
func (n *VariableAssignment) Type() NodeType {
	return NodeVariableAssignment
}

// String возвращает строковое представление
func (n *VariableAssignment) String() string {
	var builder strings.Builder
	builder.WriteString("VariableAssignment(")
	builder.WriteString(n.Variable.String())
	builder.WriteString(" = ")
	if valueNode, ok := n.Value.(Node); ok {
		builder.WriteString(valueNode.String())
	} else {
		builder.WriteString(fmt.Sprintf("%v", n.Value.ToMap()))
	}
	builder.WriteString(")")
	return builder.String()
}

// LeftToken возвращает токен переменной
func (n *VariableAssignment) LeftToken() lexer.Token {
	return n.Variable.Token
}

// RightToken возвращает токен значения
func (n *VariableAssignment) RightToken() lexer.Token {
	// Возвращаем токен значения, если он есть
	return n.Assign
}

// ToMap преобразует узел в map для сериализации
func (n *VariableAssignment) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"type":     "variable_assignment",
		"variable": n.Variable.ToMap(),
		"value":    n.Value.ToMap(),
	}
}

// VariableRead - узел для чтения переменной
type VariableRead struct {
	BaseNode
	Variable *Identifier
}

// Position возвращает позицию узла в коде (реализация интерфейса Statement)
func (n *VariableRead) Position() Position {
	return n.Variable.Position()
}

// NewVariableRead создает новый узел чтения переменной
func NewVariableRead(variable *Identifier) *VariableRead {
	return &VariableRead{
		Variable: variable,
	}
}

// Type возвращает тип узла
func (n *VariableRead) Type() NodeType {
	return NodeVariableRead
}

// String возвращает строковое представление
func (n *VariableRead) String() string {
	return fmt.Sprintf("VariableRead(%s)", n.Variable.String())
}

// LeftToken возвращает токен переменной
func (n *VariableRead) LeftToken() lexer.Token {
	return n.Variable.Token
}

// RightToken возвращает токен переменной
func (n *VariableRead) RightToken() lexer.Token {
	return n.Variable.Token
}

// ToMap преобразует узел в map для сериализации
func (n *VariableRead) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"type":     "variable_read",
		"variable": n.Variable.ToMap(),
	}
}

// Identifier - узел для идентификатора
type Identifier struct {
	BaseNode
	Token     lexer.Token
	Name      string
	Pos       Position
	Language  string   // Язык для квалифицированных имен (например, "lua" в "lua.x")
	Path      []string // Путь для сложных квалифицированных имен (например, ["table", "onemore"] в "lua.table.onemore.x")
	Qualified bool     // true если это квалифицированное имя (language.variable)
}

// NewIdentifier создает новый узел идентификатора
func NewIdentifier(token lexer.Token, name string) *Identifier {
	pos := Position{
		Line:   token.Line,
		Column: token.Column,
		Offset: token.Position,
	}
	return &Identifier{
		Token:     token,
		Name:      name,
		Pos:       pos,
		Qualified: false,
	}
}

// NewQualifiedIdentifier создает новый узел квалифицированного идентификатора
func NewQualifiedIdentifier(languageToken, nameToken lexer.Token, language, name string) *Identifier {
	pos := Position{
		Line:   languageToken.Line,
		Column: languageToken.Column,
		Offset: languageToken.Position,
	}
	return &Identifier{
		Token:     nameToken,
		Name:      name,
		Pos:       pos,
		Language:  language,
		Qualified: true,
	}
}

// NewQualifiedIdentifierWithPath создает новый узел квалифицированного идентификатора с путем
func NewQualifiedIdentifierWithPath(languageToken, nameToken lexer.Token, language string, path []string, name string) *Identifier {
	pos := Position{
		Line:   languageToken.Line,
		Column: languageToken.Column,
		Offset: languageToken.Position,
	}
	return &Identifier{
		Token:     nameToken,
		Name:      name,
		Pos:       pos,
		Language:  language,
		Path:      path,
		Qualified: true,
	}
}

// expressionMarker реализует интерфейс Expression
func (id *Identifier) expressionMarker() {}

// statementMarker реализует интерфейс Statement для VariableAssignment
func (n *VariableAssignment) statementMarker() {}

// expressionMarker реализует интерфейс Expression для VariableAssignment
func (n *VariableAssignment) expressionMarker() {}

// ExpressionAssignment - узел для присваивания выражению (например, dict["key"] = value)
type ExpressionAssignment struct {
	BaseNode
	Left   Expression  // Левая часть присваивания (выражение)
	Assign lexer.Token // Токен присваивания
	Value  Expression  // Значение для присваивания
}

// Position возвращает позицию узла в коде
func (ea *ExpressionAssignment) Position() Position {
	return ea.Left.Position()
}

// Type возвращает тип узла
func (ea *ExpressionAssignment) Type() NodeType {
	return NodeInvalid // Используем NodeInvalid т.к. нет отдельного типа
}

// String возвращает строковое представление
func (ea *ExpressionAssignment) String() string {
	return fmt.Sprintf("ExpressionAssignment(%s = %s)", ea.Left, ea.Value)
}

// ToMap преобразует узел в map для сериализации
func (ea *ExpressionAssignment) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"type":     "ExpressionAssignment",
		"left":     ea.Left.ToMap(),
		"operator": ea.Assign.Value,
		"value":    ea.Value.ToMap(),
		"position": ea.Left.Position().ToMap(),
	}
}

// NewExpressionAssignment создает новый узел присваивания выражению
func NewExpressionAssignment(left Expression, assign lexer.Token, value Expression) *ExpressionAssignment {
	return &ExpressionAssignment{
		Left:   left,
		Assign: assign,
		Value:  value,
	}
}

// statementMarker реализует интерфейс Statement
func (ea *ExpressionAssignment) statementMarker() {}

// expressionMarker реализует интерфейс Expression
func (ea *ExpressionAssignment) expressionMarker() {}

// statementMarker реализует интерфейс Statement для VariableRead
func (n *VariableRead) statementMarker() {}

// expressionMarker реализует интерфейс Expression для VariableRead
func (n *VariableRead) expressionMarker() {}

// BitstringPatternAssignment - узел для присваивания с bitstring pattern слева
type BitstringPatternAssignment struct {
	BaseNode
	Pattern *BitstringExpression // Bitstring pattern слева от =
	Assign  lexer.Token          // Токен присваивания
	Value   Expression           // Значение справа от =
}

// Position возвращает позицию узла в коде
func (bpa *BitstringPatternAssignment) Position() Position {
	return bpa.Pattern.Position()
}

// Type возвращает тип узла
func (bpa *BitstringPatternAssignment) Type() NodeType {
	return NodeInvalid // Используем NodeInvalid т.к. нет отдельного типа
}

// String возвращает строковое представление
func (bpa *BitstringPatternAssignment) String() string {
	return fmt.Sprintf("BitstringPatternAssignment(%s = %s)", bpa.Pattern.String(), bpa.Value)
}

// ToMap преобразует узел в map для сериализации
func (bpa *BitstringPatternAssignment) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"type":     "BitstringPatternAssignment",
		"pattern":  bpa.Pattern.ToMap(),
		"value":    bpa.Value.ToMap(),
		"position": bpa.Pattern.Position().ToMap(),
	}
}

// NewBitstringPatternAssignment создает новый узел bitstring pattern assignment
func NewBitstringPatternAssignment(pattern *BitstringExpression, assign lexer.Token, value Expression) *BitstringPatternAssignment {
	return &BitstringPatternAssignment{
		Pattern: pattern,
		Assign:  assign,
		Value:   value,
	}
}

// statementMarker реализует интерфейс Statement
func (bpa *BitstringPatternAssignment) statementMarker() {}

// expressionMarker реализует интерфейс Expression
func (bpa *BitstringPatternAssignment) expressionMarker() {}

// Type возвращает тип узла
func (id *Identifier) Type() NodeType {
	return NodeIdentifier
}

// String возвращает строковое представление
func (id *Identifier) String() string {
	if id.Qualified {
		if len(id.Path) > 0 {
			fullPath := id.Language
			for _, part := range id.Path {
				fullPath += "." + part
			}
			return fmt.Sprintf("Identifier(%s.%s)", fullPath, id.Name)
		}
		return fmt.Sprintf("Identifier(%s.%s)", id.Language, id.Name)
	}
	return fmt.Sprintf("Identifier(%s)", id.Name)
}

// Position возвращает позицию узла в коде
func (id *Identifier) Position() Position {
	return id.Pos
}

// ToMap преобразует узел в map для сериализации
func (id *Identifier) ToMap() map[string]interface{} {
	result := map[string]interface{}{
		"type":     "identifier",
		"name":     id.Name,
		"position": id.Pos.ToMap(),
	}
	if id.Qualified {
		result["language"] = id.Language
		result["qualified"] = true
		if len(id.Path) > 0 {
			result["path"] = id.Path
		}
	}
	return result
}
