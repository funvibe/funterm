package ast

import (
	"fmt"
	"strings"

	"go-parser/pkg/lexer"
)

// NodeType представляет тип узла AST
type NodeType int

const (
	NodeInvalid NodeType = iota
	NodeProgram
	NodeParentheses
	NodeArrayLiteral
	NodeObjectLiteral
	NodeVariableAssignment
	NodeVariableRead
	NodeIdentifier
	NodeForInLoop      // Python for-in цикл
	NodeNumericForLoop // Lua числовой цикл
	NodeWhileLoop      // While цикл
	NodeBreak          // Break оператор
	NodeContinue       // Continue оператор
	// Pattern matching узлы
	NodeMatchStatement
	NodeMatchArm
	NodeLiteralPattern
	NodeArrayPattern
	NodeObjectPattern
	NodeVariablePattern
	NodeWildcardPattern
	// Bitstrings узлы
	NodeBitstringExpression
	NodeBitstringSegment
	NodeBitstringPattern
	NodeSizeExpression
	// Pipes и Background Tasks узлы
	NodePipeExpression
	NodeLanguageCallStatement
	// If/Else узлы
	NodeIfStatement
	// Native Code Integration узлы (Task 25)
	NodeImportStatement
	NodeCodeBlockStatement
	// Ternary expressions
	NodeTernaryExpression
)

// String возвращает строковое представление типа узла
func (t NodeType) String() string {
	switch t {
	case NodeInvalid:
		return "Invalid"
	case NodeProgram:
		return "Program"
	case NodeParentheses:
		return "Parentheses"
	case NodeArrayLiteral:
		return "ArrayLiteral"
	case NodeObjectLiteral:
		return "ObjectLiteral"
	case NodeVariableAssignment:
		return "VariableAssignment"
	case NodeVariableRead:
		return "VariableRead"
	case NodeIdentifier:
		return "Identifier"
	case NodeForInLoop:
		return "ForInLoop"
	case NodeNumericForLoop:
		return "NumericForLoop"
	case NodeWhileLoop:
		return "WhileLoop"
	case NodeBreak:
		return "Break"
	case NodeContinue:
		return "Continue"
	case NodeMatchStatement:
		return "MatchStatement"
	case NodeMatchArm:
		return "MatchArm"
	case NodeLiteralPattern:
		return "LiteralPattern"
	case NodeArrayPattern:
		return "ArrayPattern"
	case NodeObjectPattern:
		return "ObjectPattern"
	case NodeVariablePattern:
		return "VariablePattern"
	case NodeWildcardPattern:
		return "WildcardPattern"
	case NodeBitstringExpression:
		return "BitstringExpression"
	case NodeBitstringSegment:
		return "BitstringSegment"
	case NodeBitstringPattern:
		return "BitstringPattern"
	case NodeSizeExpression:
		return "SizeExpression"
	case NodePipeExpression:
		return "PipeExpression"
	case NodeLanguageCallStatement:
		return "LanguageCallStatement"
	case NodeIfStatement:
		return "IfStatement"
	case NodeImportStatement:
		return "ImportStatement"
	case NodeCodeBlockStatement:
		return "CodeBlockStatement"
	case NodeTernaryExpression:
		return "TernaryExpression"
	default:
		return "Unknown"
	}
}

// Position представляет позицию в исходном коде
type Position struct {
	Line   int
	Column int
	Offset int
}

// String возвращает строковое представление позиции
func (p Position) String() string {
	return fmt.Sprintf("%d:%d", p.Line, p.Column)
}

// ToMap преобразует позицию в map для сериализации
func (p Position) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"line":   p.Line,
		"column": p.Column,
		"offset": p.Offset,
	}
}

// Node - базовый интерфейс для всех узлов AST
type Node interface {
	Type() NodeType
	String() string
	Children() []Node
	ToMap() map[string]interface{}
}

// BaseNode - базовая структура для всех узлов
type BaseNode struct {
	children []Node
}

// Children возвращает дочерние узлы
func (n *BaseNode) Children() []Node {
	return n.children
}

// AddChild добавляет дочерний узел
func (n *BaseNode) AddChild(child Node) {
	n.children = append(n.children, child)
}

// ProgramNode - корневой узел программы
type ProgramNode struct {
	BaseNode
	nodes []Node
}

// NewProgramNode создает новый узел программы
func NewProgramNode() *ProgramNode {
	return &ProgramNode{
		nodes: make([]Node, 0),
	}
}

// Type возвращает тип узла
func (n *ProgramNode) Type() NodeType {
	return NodeProgram
}

// String возвращает строковое представление
func (n *ProgramNode) String() string {
	var builder strings.Builder
	builder.WriteString("Program")

	if len(n.nodes) > 0 {
		builder.WriteString("\n")
		for i, node := range n.nodes {
			if i > 0 {
				builder.WriteString("\n")
			}
			builder.WriteString("  ")
			builder.WriteString(strings.ReplaceAll(node.String(), "\n", "\n  "))
		}
	}

	return builder.String()
}

// Children возвращает дочерние узлы
func (n *ProgramNode) Children() []Node {
	return n.nodes
}

// AddNode добавляет узел в программу
func (n *ProgramNode) AddNode(node Node) {
	n.nodes = append(n.nodes, node)
	n.AddChild(node)
}

// Nodes возвращает все узлы программы
func (n *ProgramNode) Nodes() []Node {
	return n.nodes
}

// ToMap преобразует узел в map для обратной совместимости
func (n *ProgramNode) ToMap() map[string]interface{} {
	children := make([]interface{}, len(n.nodes))
	for i, child := range n.nodes {
		children[i] = child.ToMap()
	}

	return map[string]interface{}{
		"type":     "program",
		"children": children,
	}
}

// ParenthesesNode - узел для скобок
type ParenthesesNode struct {
	BaseNode
	LeftParen  lexer.Token
	RightParen lexer.Token
}

// NewParenthesesNode создает новый узел скобок
func NewParenthesesNode(left, right lexer.Token) *ParenthesesNode {
	return &ParenthesesNode{
		LeftParen:  left,
		RightParen: right,
	}
}

// Type возвращает тип узла
func (n *ParenthesesNode) Type() NodeType {
	return NodeParentheses
}

// String возвращает строковое представление
func (n *ParenthesesNode) String() string {
	var builder strings.Builder
	builder.WriteString("Parentheses")

	if len(n.Children()) > 0 {
		builder.WriteString("(\n")
		for i, child := range n.Children() {
			if i > 0 {
				builder.WriteString(",\n")
			}
			builder.WriteString("  ")
			builder.WriteString(strings.ReplaceAll(child.String(), "\n", "\n  "))
		}
		builder.WriteString("\n)")
	} else {
		builder.WriteString("()")
	}

	return builder.String()
}

// LeftToken возвращает левую скобку
func (n *ParenthesesNode) LeftToken() lexer.Token {
	return n.LeftParen
}

// RightToken возвращает правую скобку
func (n *ParenthesesNode) RightToken() lexer.Token {
	return n.RightParen
}

// ToMap преобразует узел в map для обратной совместимости
func (n *ParenthesesNode) ToMap() map[string]interface{} {
	children := make([]interface{}, len(n.Children()))
	for i, child := range n.Children() {
		if paren, ok := child.(*ParenthesesNode); ok {
			children[i] = paren.ToMap()
		} else {
			children[i] = child.String()
		}
	}

	// Вычисляем глубину вложенности - для текущей реализации всегда 0
	// так как глубина должна вычисляться на уровне парсера
	depth := 0

	return map[string]interface{}{
		"type":     "parentheses",
		"children": children,
		"depth":    depth,
	}
}
