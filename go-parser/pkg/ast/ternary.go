package ast

import (
	"fmt"
	"go-parser/pkg/lexer"
)

// TernaryExpression представляет тернарный оператор condition ? true_expr : false_expr
type TernaryExpression struct {
	BaseNode
	Condition Expression
	Question  lexer.Token
	TrueExpr  Expression
	Colon     lexer.Token
	FalseExpr Expression
	Pos       Position
}

// NewTernaryExpression создает новый узел тернарного выражения
func NewTernaryExpression(condition Expression, question, colon lexer.Token, trueExpr, falseExpr Expression, pos Position) *TernaryExpression {
	return &TernaryExpression{
		Condition: condition,
		Question:  question,
		TrueExpr:  trueExpr,
		Colon:     colon,
		FalseExpr: falseExpr,
		Pos:       pos,
	}
}

// Type возвращает тип узла
func (n *TernaryExpression) Type() NodeType {
	return NodeInvalid // Используем NodeInvalid как в других выражениях
}

// Position возвращает позицию узла
func (n *TernaryExpression) Position() Position {
	return n.Pos
}

// ToMap преобразует узел в map для сериализации
func (n *TernaryExpression) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"type":       "ternary_expression",
		"condition":  n.Condition.(ProtoNode).ToMap(),
		"true_expr":  n.TrueExpr.(ProtoNode).ToMap(),
		"false_expr": n.FalseExpr.(ProtoNode).ToMap(),
		"position":   n.Position().ToMap(),
	}
}

// String возвращает строковое представление
func (n *TernaryExpression) String() string {
	return fmt.Sprintf("TernaryExpression(%s ? %s : %s)",
		fmt.Sprintf("%v", n.Condition),
		fmt.Sprintf("%v", n.TrueExpr),
		fmt.Sprintf("%v", n.FalseExpr))
}

// Children возвращает дочерние узлы
func (n *TernaryExpression) Children() []Node {
	return []Node{n.Condition.(Node), n.TrueExpr.(Node), n.FalseExpr.(Node)}
}

// expressionMarker реализует интерфейс Expression
func (n *TernaryExpression) expressionMarker() {}
