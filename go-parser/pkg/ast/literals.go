package ast

import "fmt"

// StringLiteral - строковый литерал
type StringLiteral struct {
	BaseNode
	Value string
	Raw   string // Original string including delimiters
	Pos   Position
}

// expressionMarker реализует интерфейс Expression
func (sl *StringLiteral) expressionMarker() {}

// Type возвращает тип узла
func (sl *StringLiteral) Type() NodeType {
	return NodeInvalid // Используем NodeInvalid т.к. нет отдельного типа для строкового литерала
}

// String возвращает строковое представление
func (sl *StringLiteral) String() string {
	return fmt.Sprintf("StringLiteral(%q)", sl.Value)
}

// ToMap преобразует узел в map для сериализации
func (sl *StringLiteral) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"type":     "StringLiteral",
		"value":    sl.Value,
		"raw":      sl.Raw,
		"position": sl.Pos.ToMap(),
	}
}

// Position возвращает позицию узла в коде
func (sl *StringLiteral) Position() Position {
	return sl.Pos
}

// NumberLiteral - числовой литерал
type NumberLiteral struct {
	BaseNode
	Value float64
	Pos   Position
}

// expressionMarker реализует интерфейс Expression
func (nl *NumberLiteral) expressionMarker() {}

// Type возвращает тип узла
func (nl *NumberLiteral) Type() NodeType {
	return NodeInvalid // Используем NodeInvalid т.к. нет отдельного типа для числового литерала
}

// String возвращает строковое представление
func (nl *NumberLiteral) String() string {
	return fmt.Sprintf("NumberLiteral(%v)", nl.Value)
}

// ToMap преобразует узел в map для сериализации
func (nl *NumberLiteral) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"type":     "NumberLiteral",
		"value":    nl.Value,
		"position": nl.Pos.ToMap(),
	}
}

// Position возвращает позицию узла в коде
func (nl *NumberLiteral) Position() Position {
	return nl.Pos
}

// BooleanLiteral - булев литерал
type BooleanLiteral struct {
	BaseNode
	Value bool
	Pos   Position
}

// expressionMarker реализует интерфейс Expression
func (bl *BooleanLiteral) expressionMarker() {}

// Type возвращает тип узла
func (bl *BooleanLiteral) Type() NodeType {
	return NodeInvalid // Используем NodeInvalid т.к. нет отдельного типа для булева литерала
}

// String возвращает строковое представление
func (bl *BooleanLiteral) String() string {
	return fmt.Sprintf("BooleanLiteral(%t)", bl.Value)
}

// ToMap преобразует узел в map для сериализации
func (bl *BooleanLiteral) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"type":     "BooleanLiteral",
		"value":    bl.Value,
		"position": bl.Pos.ToMap(),
	}
}

// Position возвращает позицию узла в коде
func (bl *BooleanLiteral) Position() Position {
	return bl.Pos
}
