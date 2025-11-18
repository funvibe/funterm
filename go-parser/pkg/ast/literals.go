package ast

import (
	"fmt"
	"math/big"

	"go-parser/pkg/lexer"
)

// StringLiteral - строковый литерал
type StringLiteral struct {
	BaseNode
	Value string
	Raw   string // Original string including delimiters
	Pos   Position
}

// NewStringLiteral создает новый строковый литерал
func NewStringLiteral(token lexer.Token, value, raw string) *StringLiteral {
	pos := Position{
		Line:   token.Line,
		Column: token.Column,
		Offset: token.Position,
	}
	return &StringLiteral{
		Value: value,
		Raw:   raw,
		Pos:   pos,
	}
}

// expressionMarker реализует интерфейс Expression
func (sl *StringLiteral) expressionMarker() {}

// Type возвращает тип узла
func (sl *StringLiteral) Type() NodeType {
	return NodeStringLiteral
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
	FloatValue float64  // For floating point numbers
	IntValue   *big.Int // For integer numbers (nil for floats)
	IsInt      bool     // True if this is an integer, false if float
	Pos        Position
}

// expressionMarker реализует интерфейс Expression
func (nl *NumberLiteral) expressionMarker() {}

// Type возвращает тип узла
func (nl *NumberLiteral) Type() NodeType {
	return NodeNumberLiteral
}

// String возвращает строковое представление
func (nl *NumberLiteral) String() string {
	if nl.IsInt && nl.IntValue != nil {
		return nl.IntValue.String()
	}
	return fmt.Sprintf("%v", nl.FloatValue)
}

// ToMap преобразует узел в map для сериализации
func (nl *NumberLiteral) ToMap() map[string]interface{} {
	result := map[string]interface{}{
		"type":     "NumberLiteral",
		"isInt":    nl.IsInt,
		"position": nl.Pos.ToMap(),
	}

	if nl.IsInt && nl.IntValue != nil {
		// For integers, convert to float64 for the unified value field
		// and keep the string representation in intValue field
		result["value"] = float64(nl.IntValue.Int64())
		result["intValue"] = nl.IntValue.String()
	} else {
		// For floats, include both unified value field and specific floatValue field
		result["value"] = nl.FloatValue
		result["floatValue"] = nl.FloatValue
	}

	return result
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

// NewBooleanLiteral создает новый булев литерал
func NewBooleanLiteral(token lexer.Token, value bool) *BooleanLiteral {
	pos := Position{
		Line:   token.Line,
		Column: token.Column,
		Offset: token.Position,
	}
	return &BooleanLiteral{
		Value: value,
		Pos:   pos,
	}
}

// expressionMarker реализует интерфейс Expression
func (bl *BooleanLiteral) expressionMarker() {}

// Type возвращает тип узла
func (bl *BooleanLiteral) Type() NodeType {
	return NodeBooleanLiteral
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

// NilLiteral представляет nil литерал
type NilLiteral struct {
	BaseNode
	Pos Position
}

// NewNilLiteral создает новый nil литерал
func NewNilLiteral(token lexer.Token) *NilLiteral {
	pos := Position{
		Line:   token.Line,
		Column: token.Column,
		Offset: token.Position,
	}
	return &NilLiteral{
		Pos: pos,
	}
}

// expressionMarker реализует интерфейс Expression
func (nl *NilLiteral) expressionMarker() {}

// Type возвращает тип узла
func (nl *NilLiteral) Type() NodeType {
	return NodeNilLiteral
}

// String возвращает строковое представление
func (nl *NilLiteral) String() string {
	return "NilLiteral(nil)"
}

// ToMap преобразует узел в map для сериализации
func (nl *NilLiteral) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"type":     "NilLiteral",
		"value":    nil,
		"position": nl.Pos.ToMap(),
	}
}

// Position возвращает позицию узла в коде
func (nl *NilLiteral) Position() Position {
	return nl.Pos
}
