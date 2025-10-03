package ast

import (
	"fmt"
	"strings"

	"go-parser/pkg/lexer"
)

// ArrayLiteral - узел для массива
type ArrayLiteral struct {
	BaseNode
	LeftBracket  lexer.Token
	RightBracket lexer.Token
	Elements     []Expression // Элементы массива
}

// expressionMarker реализует интерфейс Expression
func (n *ArrayLiteral) expressionMarker() {}

// NewArrayLiteral создает новый узел массива
func NewArrayLiteral(left, right lexer.Token) *ArrayLiteral {
	return &ArrayLiteral{
		LeftBracket:  left,
		RightBracket: right,
		Elements:     make([]Expression, 0),
	}
}

// Type возвращает тип узла
func (n *ArrayLiteral) Type() NodeType {
	return NodeArrayLiteral
}

// String возвращает строковое представление
func (n *ArrayLiteral) String() string {
	var builder strings.Builder
	builder.WriteString("Array")

	if len(n.Elements) > 0 {
		builder.WriteString("[\n")
		for i, element := range n.Elements {
			if i > 0 {
				builder.WriteString(",\n")
			}
			builder.WriteString("  ")
			if elemNode, ok := element.(Node); ok {
				builder.WriteString(strings.ReplaceAll(elemNode.String(), "\n", "\n  "))
			} else {
				builder.WriteString(fmt.Sprintf("%v", element.ToMap()))
			}
		}
		builder.WriteString("\n]")
	} else {
		builder.WriteString("[]")
	}

	return builder.String()
}

// AddElement добавляет элемент в массив
func (n *ArrayLiteral) AddElement(element Expression) {
	n.Elements = append(n.Elements, element)
	if elemNode, ok := element.(Node); ok {
		n.AddChild(elemNode)
	}
}

// LeftToken возвращает левую скобку
func (n *ArrayLiteral) LeftToken() lexer.Token {
	return n.LeftBracket
}

// RightToken возвращает правую скобку
func (n *ArrayLiteral) RightToken() lexer.Token {
	return n.RightBracket
}

// Position возвращает позицию узла в коде
func (n *ArrayLiteral) Position() Position {
	return Position{
		Line:   n.LeftBracket.Line,
		Column: n.LeftBracket.Column,
		Offset: n.LeftBracket.Position,
	}
}

// ToMap преобразует узел в map для сериализации
func (n *ArrayLiteral) ToMap() map[string]interface{} {
	elements := make([]interface{}, len(n.Elements))
	for i, element := range n.Elements {
		elements[i] = element.ToMap()
	}

	return map[string]interface{}{
		"type":     "array",
		"elements": elements,
	}
}

// ObjectProperty - свойство объекта (ключ: значение)
type ObjectProperty struct {
	Key   Expression
	Value Expression
}

// ObjectLiteral - узел для объекта
type ObjectLiteral struct {
	BaseNode
	LeftBrace  lexer.Token
	RightBrace lexer.Token
	Properties []ObjectProperty // Свойства объекта
}

// expressionMarker реализует интерфейс Expression
func (n *ObjectLiteral) expressionMarker() {}

// NewObjectLiteral создает новый узел объекта
func NewObjectLiteral(left, right lexer.Token) *ObjectLiteral {
	return &ObjectLiteral{
		LeftBrace:  left,
		RightBrace: right,
		Properties: make([]ObjectProperty, 0),
	}
}

// Type возвращает тип узла
func (n *ObjectLiteral) Type() NodeType {
	return NodeObjectLiteral
}

// String возвращает строковое представление
func (n *ObjectLiteral) String() string {
	var builder strings.Builder
	builder.WriteString("Object")

	if len(n.Properties) > 0 {
		builder.WriteString("{\n")
		for i, prop := range n.Properties {
			if i > 0 {
				builder.WriteString(",\n")
			}
			builder.WriteString("  ")
			if keyNode, ok := prop.Key.(Node); ok {
				builder.WriteString(keyNode.String())
			} else {
				builder.WriteString(fmt.Sprintf("%v", prop.Key.ToMap()))
			}
			builder.WriteString(": ")
			if valueNode, ok := prop.Value.(Node); ok {
				builder.WriteString(strings.ReplaceAll(valueNode.String(), "\n", "\n  "))
			} else {
				builder.WriteString(fmt.Sprintf("%v", prop.Value.ToMap()))
			}
		}
		builder.WriteString("\n}")
	} else {
		builder.WriteString("{}")
	}

	return builder.String()
}

// AddProperty добавляет свойство в объект
func (n *ObjectLiteral) AddProperty(key, value Expression) {
	n.Properties = append(n.Properties, ObjectProperty{
		Key:   key,
		Value: value,
	})
	if keyNode, ok := key.(Node); ok {
		n.AddChild(keyNode)
	}
	if valueNode, ok := value.(Node); ok {
		n.AddChild(valueNode)
	}
}

// Position возвращает позицию узла в коде
func (n *ObjectLiteral) Position() Position {
	return Position{
		Line:   n.LeftBrace.Line,
		Column: n.LeftBrace.Column,
		Offset: n.LeftBrace.Position,
	}
}

// LeftToken возвращает левую фигурную скобку
func (n *ObjectLiteral) LeftToken() lexer.Token {
	return n.LeftBrace
}

// RightToken возвращает правую фигурную скобку
func (n *ObjectLiteral) RightToken() lexer.Token {
	return n.RightBrace
}

// ToMap преобразует узел в map для сериализации
func (n *ObjectLiteral) ToMap() map[string]interface{} {
	properties := make([]map[string]interface{}, len(n.Properties))
	for i, prop := range n.Properties {
		properties[i] = map[string]interface{}{
			"key":   prop.Key.ToMap(),
			"value": prop.Value.ToMap(),
		}
	}

	return map[string]interface{}{
		"type":       "object",
		"properties": properties,
	}
}
