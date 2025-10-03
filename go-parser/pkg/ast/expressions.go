package ast

import (
	"fmt"
	"go-parser/pkg/lexer"
)

// BinaryExpression - бинарное выражение (например, x < 10, a == b)
type BinaryExpression struct {
	BaseNode
	Left     Expression
	Operator string
	Right    Expression
	Pos      Position
}

// expressionMarker реализует интерфейс Expression
func (be *BinaryExpression) expressionMarker() {}

// Position возвращает позицию узла в коде
func (be *BinaryExpression) Position() Position {
	return be.Pos
}

// Type возвращает тип узла
func (be *BinaryExpression) Type() NodeType {
	return NodeInvalid // Используем NodeInvalid т.к. нет отдельного типа для бинарного выражения
}

// String возвращает строковое представление
func (be *BinaryExpression) String() string {
	return fmt.Sprintf("BinaryExpression(%s %s %s)", be.Left, be.Operator, be.Right)
}

// ToMap преобразует узел в map для сериализации
func (be *BinaryExpression) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"type":     "BinaryExpression",
		"left":     be.Left.ToMap(),
		"operator": be.Operator,
		"right":    be.Right.ToMap(),
		"position": be.Pos.ToMap(),
	}
}

// UnaryExpression - унарное выражение (например, -x, !condition)
type UnaryExpression struct {
	BaseNode
	Operator string
	Right    Expression
	Pos      Position
}

// expressionMarker реализует интерфейс Expression
func (ue *UnaryExpression) expressionMarker() {}

// Position возвращает позицию узла в коде
func (ue *UnaryExpression) Position() Position {
	return ue.Pos
}

// Type возвращает тип узла
func (ue *UnaryExpression) Type() NodeType {
	return NodeInvalid // Используем NodeInvalid т.к. нет отдельного типа для унарного выражения
}

// String возвращает строковое представление
func (ue *UnaryExpression) String() string {
	return fmt.Sprintf("UnaryExpression(%s %s)", ue.Operator, ue.Right)
}

// ToMap преобразует узел в map для сериализации
func (ue *UnaryExpression) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"type":     "UnaryExpression",
		"operator": ue.Operator,
		"right":    ue.Right.ToMap(),
		"position": ue.Pos.ToMap(),
	}
}

// ElvisExpression - Elvis оператор (тернарный оператор ?:)
type ElvisExpression struct {
	BaseNode
	Left     Expression  // Левый операнд (проверяемое значение)
	Right    Expression  // Правый операнд (значение по умолчанию)
	Question lexer.Token // Токен ?
	Colon    lexer.Token // Токен :
	Pos      Position
}

// expressionMarker реализует интерфейс Expression
func (ee *ElvisExpression) expressionMarker() {}

// Position возвращает позицию узла в коде
func (ee *ElvisExpression) Position() Position {
	return ee.Pos
}

// Type возвращает тип узла
func (ee *ElvisExpression) Type() NodeType {
	return NodeInvalid // Используем NodeInvalid т.к. нет отдельного типа для Elvis выражения
}

// String возвращает строковое представление
func (ee *ElvisExpression) String() string {
	return fmt.Sprintf("ElvisExpression(%s ?: %s)", ee.Left, ee.Right)
}

// ToMap преобразует узел в map для сериализации
func (ee *ElvisExpression) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"type":  "ElvisExpression",
		"left":  ee.Left.ToMap(),
		"right": ee.Right.ToMap(),
		"pos":   ee.Pos.ToMap(),
	}
}

// NewElvisExpression создает новый узел Elvis выражения
func NewElvisExpression(left Expression, question, colon lexer.Token, right Expression, pos Position) *ElvisExpression {
	return &ElvisExpression{
		Left:     left,
		Right:    right,
		Question: question,
		Colon:    colon,
		Pos:      pos,
	}
}

// NewBinaryExpression создает новый узел бинарного выражения
func NewBinaryExpression(left Expression, operator string, right Expression, pos Position) *BinaryExpression {
	return &BinaryExpression{
		Left:     left,
		Operator: operator,
		Right:    right,
		Pos:      pos,
	}
}

// NewUnaryExpression создает новый узел унарного выражения
func NewUnaryExpression(operator string, right Expression, pos Position) *UnaryExpression {
	return &UnaryExpression{
		Operator: operator,
		Right:    right,
		Pos:      pos,
	}
}

// FieldAccess представляет доступ к полю объекта (например, lua.x)
type FieldAccess struct {
	BaseNode
	Object Expression // Объект, к которому обращаемся (например, lua)
	Field  string     // Имя поля (например, x)
	Pos    Position
}

// expressionMarker реализует интерфейс Expression
func (fa *FieldAccess) expressionMarker() {}

// Position возвращает позицию узла в коде
func (fa *FieldAccess) Position() Position {
	return fa.Pos
}

// Type возвращает тип узла
func (fa *FieldAccess) Type() NodeType {
	return NodeInvalid // Используем NodeInvalid т.к. нет отдельного типа для доступа к полю
}

// String возвращает строковое представление
func (fa *FieldAccess) String() string {
	return fmt.Sprintf("FieldAccess(%s.%s)", fa.Object, fa.Field)
}

// ToMap преобразует узел в map для сериализации
func (fa *FieldAccess) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"type":     "FieldAccess",
		"object":   fa.Object.ToMap(),
		"field":    fa.Field,
		"position": fa.Pos.ToMap(),
	}
}

// NewFieldAccess создает новый узел доступа к полю
func NewFieldAccess(object Expression, field string, pos Position) *FieldAccess {
	return &FieldAccess{
		Object: object,
		Field:  field,
		Pos:    pos,
	}
}

// IndexExpression представляет индексированный доступ к элементу (например, dict["key"] или arr[0])
type IndexExpression struct {
	BaseNode
	Object Expression // Объект, к которому обращаемся (например, dict или arr)
	Index  Expression // Индекс (например, "key" или 0)
	Pos    Position
}

// expressionMarker реализует интерфейс Expression
func (ie *IndexExpression) expressionMarker() {}

// statementMarker реализует интерфейс Statement для IndexExpression
func (ie *IndexExpression) statementMarker() {}

// Position возвращает позицию узла в коде
func (ie *IndexExpression) Position() Position {
	return ie.Pos
}

// Type возвращает тип узла
func (ie *IndexExpression) Type() NodeType {
	return NodeInvalid // Используем NodeInvalid т.к. нет отдельного типа для индексированного доступа
}

// String возвращает строковое представление
func (ie *IndexExpression) String() string {
	return fmt.Sprintf("IndexExpression(%s[%s])", ie.Object, ie.Index)
}

// ToMap преобразует узел в map для сериализации
func (ie *IndexExpression) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"type":     "IndexExpression",
		"object":   ie.Object.ToMap(),
		"index":    ie.Index.ToMap(),
		"position": ie.Pos.ToMap(),
	}
}

// NewIndexExpression создает новый узел индексированного доступа
func NewIndexExpression(object Expression, index Expression, pos Position) *IndexExpression {
	return &IndexExpression{
		Object: object,
		Index:  index,
		Pos:    pos,
	}
}

// NamedArgument представляет именованный аргумент функции (например, days=-1)
type NamedArgument struct {
	BaseNode
	Name  string     // Имя аргумента (например, "days")
	Value Expression // Значение аргумента (например, -1)
	Pos   Position
}

// expressionMarker реализует интерфейс Expression
func (na *NamedArgument) expressionMarker() {}

// Position возвращает позицию узла в коде
func (na *NamedArgument) Position() Position {
	return na.Pos
}

// Type возвращает тип узла
func (na *NamedArgument) Type() NodeType {
	return NodeInvalid // Используем NodeInvalid т.к. нет отдельного типа для именованного аргумента
}

// String возвращает строковое представление
func (na *NamedArgument) String() string {
	return fmt.Sprintf("NamedArgument(%s=%s)", na.Name, na.Value)
}

// ToMap преобразует узел в map для сериализации
func (na *NamedArgument) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"type":     "NamedArgument",
		"name":     na.Name,
		"value":    na.Value.ToMap(),
		"position": na.Pos.ToMap(),
	}
}

// NewNamedArgument создает новый узел именованного аргумента
func NewNamedArgument(name string, value Expression, pos Position) *NamedArgument {
	return &NamedArgument{
		Name:  name,
		Value: value,
		Pos:   pos,
	}
}
