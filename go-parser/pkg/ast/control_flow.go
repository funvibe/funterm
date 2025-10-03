package ast

import (
	"go-parser/pkg/lexer"
	"strings"
)

// IfStatement представляет if/else конструкцию: if (condition) { ... } else { ... }
type IfStatement struct {
	BaseNode
	Condition   Expression      // условие if
	Consequent  *BlockStatement // тело if (блок в фигурных скобках)
	Alternate   *BlockStatement // тело else (опционально)
	IfToken     lexer.Token     // токен 'if'
	LParenToken lexer.Token     // токен '('
	RParenToken lexer.Token     // токен ')'
	ElseToken   lexer.Token     // токен 'else' (опционально)
	Pos         Position        // позиция начала if
}

// NewIfStatement создает новый узел if оператора
func NewIfStatement(ifToken, lParenToken, rParenToken lexer.Token, condition Expression, consequent *BlockStatement) *IfStatement {
	return &IfStatement{
		IfToken:     ifToken,
		LParenToken: lParenToken,
		RParenToken: rParenToken,
		Condition:   condition,
		Consequent:  consequent,
		Pos:         tokenToPosition(ifToken),
	}
}

// SetElse устанавливает else блок и токен
func (n *IfStatement) SetElse(elseToken lexer.Token, alternate *BlockStatement) {
	n.ElseToken = elseToken
	n.Alternate = alternate
}

// HasElse проверяет, есть ли else блок
func (n *IfStatement) HasElse() bool {
	return n.Alternate != nil
}

// Type возвращает тип узла
func (n *IfStatement) Type() NodeType {
	return NodeIfStatement
}

// statementMarker реализует интерфейс Statement
func (n *IfStatement) statementMarker() {}

// Position возвращает позицию узла
func (n *IfStatement) Position() Position {
	return n.Pos
}

// String возвращает строковое представление
func (n *IfStatement) String() string {
	var builder strings.Builder
	builder.WriteString("IfStatement(")

	// Используем утверждение типа для вызова String()
	if condNode, ok := n.Condition.(Node); ok {
		builder.WriteString(condNode.String())
	} else {
		builder.WriteString("Condition")
	}

	builder.WriteString(") {\n")
	for i, stmt := range n.Consequent.Statements {
		if i > 0 {
			builder.WriteString("\n")
		}
		builder.WriteString("  ")
		if stmtNode, ok := stmt.(Node); ok {
			builder.WriteString(strings.ReplaceAll(stmtNode.String(), "\n", "\n  "))
		} else {
			builder.WriteString("Statement")
		}
	}
	builder.WriteString("\n}")

	if n.HasElse() {
		builder.WriteString(" else {\n")
		for i, stmt := range n.Alternate.Statements {
			if i > 0 {
				builder.WriteString("\n")
			}
			builder.WriteString("  ")
			if stmtNode, ok := stmt.(Node); ok {
				builder.WriteString(strings.ReplaceAll(stmtNode.String(), "\n", "\n  "))
			} else {
				builder.WriteString("Statement")
			}
		}
		builder.WriteString("\n}")
	}

	return builder.String()
}

// ToMap преобразует узел в map для сериализации
func (n *IfStatement) ToMap() map[string]interface{} {
	consequent := make([]interface{}, len(n.Consequent.Statements))
	for i, stmt := range n.Consequent.Statements {
		if stmtNode, ok := stmt.(ProtoNode); ok {
			consequent[i] = stmtNode.ToMap()
		} else {
			consequent[i] = map[string]interface{}{"type": "unknown_statement"}
		}
	}

	// Используем утверждение типа для вызова ToMap()
	var conditionMap interface{}
	if condNode, ok := n.Condition.(ProtoNode); ok {
		conditionMap = condNode.ToMap()
	} else {
		conditionMap = map[string]interface{}{"type": "unknown_expression"}
	}

	result := map[string]interface{}{
		"type":       "if_statement",
		"condition":  conditionMap,
		"consequent": consequent,
		"position":   n.Pos.ToMap(),
	}

	if n.HasElse() {
		alternate := make([]interface{}, len(n.Alternate.Statements))
		for i, stmt := range n.Alternate.Statements {
			if stmtNode, ok := stmt.(ProtoNode); ok {
				alternate[i] = stmtNode.ToMap()
			} else {
				alternate[i] = map[string]interface{}{"type": "unknown_statement"}
			}
		}
		result["alternate"] = alternate
	}

	return result
}
