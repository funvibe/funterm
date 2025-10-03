package ast

import (
	"fmt"
	"strings"

	"go-parser/pkg/lexer"
)

// Pattern - базовый интерфейс для всех паттернов
type Pattern interface {
	ProtoNode
	patternMarker()
}

// MatchStatement - конструкция match
type MatchStatement struct {
	BaseNode
	Expression  Expression  // Выражение для сопоставления
	Arms        []MatchArm  // Ветки сопоставления
	MatchToken  lexer.Token // Токен 'match'
	LBraceToken lexer.Token // Токен '{'
	RBraceToken lexer.Token // Токен '}'
	Pos         Position
}

// statementMarker реализует интерфейс Statement
func (n *MatchStatement) statementMarker() {}

// Type возвращает тип узла
func (n *MatchStatement) Type() NodeType { return NodeMatchStatement }

// Position возвращает позицию узла в коде
func (n *MatchStatement) Position() Position { return n.Pos }

// String возвращает строковое представление
func (n *MatchStatement) String() string {
	var builder strings.Builder
	builder.WriteString("Match(")
	if exprNode, ok := n.Expression.(Node); ok {
		builder.WriteString(exprNode.String())
	} else {
		builder.WriteString(fmt.Sprintf("%v", n.Expression.ToMap()))
	}

	builder.WriteString(") {\n")
	for i, arm := range n.Arms {
		if i > 0 {
			builder.WriteString(",\n")
		}
		builder.WriteString("  ")
		builder.WriteString(arm.String())
	}
	builder.WriteString("\n}")
	return builder.String()
}

// ToMap преобразует узел в map для сериализации
func (n *MatchStatement) ToMap() map[string]interface{} {
	arms := make([]interface{}, len(n.Arms))
	for i, arm := range n.Arms {
		arms[i] = arm.ToMap()
	}

	return map[string]interface{}{
		"type":       "match_statement",
		"expression": n.Expression.ToMap(),
		"arms":       arms,
		"position":   n.Pos.ToMap(),
	}
}

// MatchArm - одна ветка pattern -> statement
type MatchArm struct {
	BaseNode
	Pattern    Pattern     // Паттерн
	ArrowToken lexer.Token // Токен '->'
	Statement  Statement   // Выполняемый код
}

// Type возвращает тип узла
func (n *MatchArm) Type() NodeType { return NodeMatchArm }

// Position возвращает позицию узла в коде
func (n *MatchArm) Position() Position { return n.Pattern.Position() }

// String возвращает строковое представление
func (n *MatchArm) String() string {
	var builder strings.Builder
	if patternNode, ok := n.Pattern.(Node); ok {
		builder.WriteString(patternNode.String())
	} else {
		builder.WriteString(fmt.Sprintf("%v", n.Pattern.ToMap()))
	}
	builder.WriteString(" -> ")
	if stmtNode, ok := n.Statement.(Node); ok {
		builder.WriteString(stmtNode.String())
	} else {
		builder.WriteString(fmt.Sprintf("%v", n.Statement.ToMap()))
	}
	return builder.String()
}

// ToMap преобразует узел в map для сериализации
func (n *MatchArm) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"type":      "match_arm",
		"pattern":   n.Pattern.ToMap(),
		"statement": n.Statement.ToMap(),
		"position":  n.Position().ToMap(),
	}
}

// LiteralPattern - литеральный паттерн
type LiteralPattern struct {
	BaseNode
	Value interface{} // "hello", 42, true, false
	Pos   Position
}

// patternMarker реализует интерфейс Pattern
func (n *LiteralPattern) patternMarker() {}

// Type возвращает тип узла
func (n *LiteralPattern) Type() NodeType { return NodeLiteralPattern }

// Position возвращает позицию узла в коде
func (n *LiteralPattern) Position() Position { return n.Pos }

// String возвращает строковое представление
func (n *LiteralPattern) String() string {
	return fmt.Sprintf("Literal(%v)", n.Value)
}

// ToMap преобразует узел в map для сериализации
func (n *LiteralPattern) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"type":     "literal_pattern",
		"value":    n.Value,
		"position": n.Pos.ToMap(),
	}
}

// ArrayPattern - массивный паттерн
type ArrayPattern struct {
	BaseNode
	Elements []Pattern // Элементы массива
	Rest     bool      // Есть ли ...rest
	Pos      Position
}

// patternMarker реализует интерфейс Pattern
func (n *ArrayPattern) patternMarker() {}

// Type возвращает тип узла
func (n *ArrayPattern) Type() NodeType { return NodeArrayPattern }

// Position возвращает позицию узла в коде
func (n *ArrayPattern) Position() Position { return n.Pos }

// String возвращает строковое представление
func (n *ArrayPattern) String() string {
	var builder strings.Builder
	builder.WriteString("[")

	for i, elem := range n.Elements {
		if i > 0 {
			builder.WriteString(", ")
		}
		if elemNode, ok := elem.(Node); ok {
			builder.WriteString(elemNode.String())
		} else {
			builder.WriteString(fmt.Sprintf("%v", elem.ToMap()))
		}
	}

	if n.Rest {
		if len(n.Elements) > 0 {
			builder.WriteString(", ")
		}
		builder.WriteString("...")
	}

	builder.WriteString("]")
	return builder.String()
}

// ToMap преобразует узел в map для сериализации
func (n *ArrayPattern) ToMap() map[string]interface{} {
	elements := make([]interface{}, len(n.Elements))
	for i, elem := range n.Elements {
		elements[i] = elem.ToMap()
	}

	return map[string]interface{}{
		"type":     "array_pattern",
		"elements": elements,
		"rest":     n.Rest,
		"position": n.Pos.ToMap(),
	}
}

// ObjectPattern - объектный паттерн
type ObjectPattern struct {
	BaseNode
	Properties map[string]Pattern // Свойства объекта
	Pos        Position
}

// patternMarker реализует интерфейс Pattern
func (n *ObjectPattern) patternMarker() {}

// Type возвращает тип узла
func (n *ObjectPattern) Type() NodeType { return NodeObjectPattern }

// Position возвращает позицию узла в коде
func (n *ObjectPattern) Position() Position { return n.Pos }

// String возвращает строковое представление
func (n *ObjectPattern) String() string {
	var builder strings.Builder
	builder.WriteString("{")

	i := 0
	for key, value := range n.Properties {
		if i > 0 {
			builder.WriteString(", ")
		}
		builder.WriteString(fmt.Sprintf("%q: ", key))
		if valueNode, ok := value.(Node); ok {
			builder.WriteString(valueNode.String())
		} else {
			builder.WriteString(fmt.Sprintf("%v", value.ToMap()))
		}
		i++
	}

	builder.WriteString("}")
	return builder.String()
}

// ToMap преобразует узел в map для сериализации
func (n *ObjectPattern) ToMap() map[string]interface{} {
	properties := make(map[string]interface{})
	for key, value := range n.Properties {
		properties[key] = value.ToMap()
	}

	return map[string]interface{}{
		"type":       "object_pattern",
		"properties": properties,
		"position":   n.Pos.ToMap(),
	}
}

// VariablePattern - переменный паттерн
type VariablePattern struct {
	BaseNode
	Name string // Имя переменной
	Pos  Position
}

// patternMarker реализует интерфейс Pattern
func (n *VariablePattern) patternMarker() {}

// Type возвращает тип узла
func (n *VariablePattern) Type() NodeType { return NodeVariablePattern }

// Position возвращает позицию узла в коде
func (n *VariablePattern) Position() Position { return n.Pos }

// String возвращает строковое представление
func (n *VariablePattern) String() string {
	return fmt.Sprintf("Variable(%s)", n.Name)
}

// ToMap преобразует узел в map для сериализации
func (n *VariablePattern) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"type":     "variable_pattern",
		"name":     n.Name,
		"position": n.Pos.ToMap(),
	}
}

// WildcardPattern - wildcard паттерн
type WildcardPattern struct {
	BaseNode
	Pos Position
}

// patternMarker реализует интерфейс Pattern
func (n *WildcardPattern) patternMarker() {}

// Type возвращает тип узла
func (n *WildcardPattern) Type() NodeType { return NodeWildcardPattern }

// Position возвращает позицию узла в коде
func (n *WildcardPattern) Position() Position { return n.Pos }

// String возвращает строковое представление
func (n *WildcardPattern) String() string {
	return "Wildcard(_)"
}

// ToMap преобразует узел в map для сериализации
func (n *WildcardPattern) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"type":     "wildcard_pattern",
		"position": n.Pos.ToMap(),
	}
}

// BitstringPattern - битстринг паттерн
type BitstringPattern struct {
	BaseNode
	Elements   []BitstringSegment // Элементы битстринга
	LeftAngle  lexer.Token        // Левый двойной угол <<
	RightAngle lexer.Token        // Правый двойной угол >>
	Pos        Position
}

// patternMarker реализует интерфейс Pattern
func (n *BitstringPattern) patternMarker() {}

// Type возвращает тип узла
func (n *BitstringPattern) Type() NodeType { return NodeBitstringPattern }

// Position возвращает позицию узла в коде
func (n *BitstringPattern) Position() Position { return n.Pos }

// String возвращает строковое представление
func (n *BitstringPattern) String() string {
	var builder strings.Builder
	builder.WriteString("BitstringPattern<<")

	for i, segment := range n.Elements {
		if i > 0 {
			builder.WriteString(", ")
		}
		builder.WriteString(segment.String())
	}

	builder.WriteString(">>")
	return builder.String()
}

// ToMap преобразует узел в map для сериализации
func (n *BitstringPattern) ToMap() map[string]interface{} {
	elements := make([]interface{}, len(n.Elements))
	for i, element := range n.Elements {
		elements[i] = element.ToMap()
	}

	return map[string]interface{}{
		"type":     "bitstring_pattern",
		"elements": elements,
		"position": n.Pos.ToMap(),
	}
}
