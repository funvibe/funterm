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
	return fmt.Sprintf("%s %s %s", be.Left, be.Operator, be.Right)
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

// BuiltinFunctionCall представляет вызов builtin функции (без квалификатора языка)
type BuiltinFunctionCall struct {
	BaseNode
	Function  string       // Имя функции (например, "id")
	Arguments []Expression // Аргументы функции
	Pos       Position
}

// expressionMarker реализует интерфейс Expression
func (bfc *BuiltinFunctionCall) expressionMarker() {}

// statementMarker реализует интерфейс Statement
func (bfc *BuiltinFunctionCall) statementMarker() {}

// Position возвращает позицию узла в коде
func (bfc *BuiltinFunctionCall) Position() Position {
	return bfc.Pos
}

// Type возвращает тип узла
func (bfc *BuiltinFunctionCall) Type() NodeType {
	return NodeInvalid // Используем NodeInvalid т.к. нет отдельного типа для builtin функции
}

// String возвращает строковое представление
func (bfc *BuiltinFunctionCall) String() string {
	return fmt.Sprintf("BuiltinFunctionCall(%s)", bfc.Function)
}

// ToMap преобразует узел в map для сериализации
func (bfc *BuiltinFunctionCall) ToMap() map[string]interface{} {
	args := make([]interface{}, len(bfc.Arguments))
	for i, arg := range bfc.Arguments {
		args[i] = arg.ToMap()
	}
	return map[string]interface{}{
		"type":      "BuiltinFunctionCall",
		"function":  bfc.Function,
		"arguments": args,
		"position":  bfc.Pos.ToMap(),
	}
}

// NewBuiltinFunctionCall создает новый узел вызова builtin функции
func NewBuiltinFunctionCall(function string, arguments []Expression, pos Position) *BuiltinFunctionCall {
	return &BuiltinFunctionCall{
		Function:  function,
		Arguments: arguments,
		Pos:       pos,
	}
}

// NestedExpression - вложенное выражение в скобках (например, (a + b))
type NestedExpression struct {
	BaseNode
	Inner Expression // Вложенное выражение
	Pos   Position   // Позиция открывающей скобки
}

// expressionMarker реализует интерфейс Expression
func (ne *NestedExpression) expressionMarker() {}

// Position возвращает позицию узла в коде
func (ne *NestedExpression) Position() Position {
	return ne.Pos
}

// Type возвращает тип узла
func (ne *NestedExpression) Type() NodeType {
	return NodeParentheses
}

// String возвращает строковое представление
func (ne *NestedExpression) String() string {
	return fmt.Sprintf("(%s)", ne.Inner)
}

// ToMap преобразует узел в map для сериализации
func (ne *NestedExpression) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"type":     "NestedExpression",
		"inner":    ne.Inner.ToMap(),
		"position": ne.Pos.ToMap(),
	}
}

// NewNestedExpression создает новый узел вложенного выражения
func NewNestedExpression(inner Expression, pos Position) *NestedExpression {
	return &NestedExpression{
		Inner: inner,
		Pos:   pos,
	}
}

// ExpressionToString конвертирует Expression в строку для использования в funbit
func ExpressionToString(expr Expression) (string, error) {
	if expr == nil {
		return "", fmt.Errorf("expression is nil")
	}

	switch e := expr.(type) {
	case *BinaryExpression:
		leftStr, err := ExpressionToString(e.Left)
		if err != nil {
			return "", err
		}
		rightStr, err := ExpressionToString(e.Right)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s%s%s", leftStr, e.Operator, rightStr), nil

	case *UnaryExpression:
		rightStr, err := ExpressionToString(e.Right)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s%s", e.Operator, rightStr), nil

	case *NestedExpression:
		innerStr, err := ExpressionToString(e.Inner)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("(%s)", innerStr), nil

	case *Identifier:
		return e.Name, nil

	case *NumberLiteral:
		if e.IsInt {
			if e.IntValue != nil {
				return e.IntValue.String(), nil
			}
			return "0", nil
		} else {
			return fmt.Sprintf("%f", e.FloatValue), nil
		}

	case *StringLiteral:
		return e.Value, nil

	case *BooleanLiteral:
		if e.Value {
			return "true", nil
		}
		return "false", nil

	default:
		// Для других типов выражений, пытаемся конвертировать в строку через ToMap()
		return fmt.Sprintf("%T", expr), nil
	}
}
