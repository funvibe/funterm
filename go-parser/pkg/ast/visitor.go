package ast

import (
	"fmt"
	"strings"
)

// Visitor - интерфейс для паттерна Visitor
type Visitor interface {
	Visit(node Node) interface{}
}

// Walk рекурсивно обходит дерево узлов
func Walk(visitor Visitor, node Node) {
	if node == nil {
		return
	}

	// Посещаем текущий узел
	result := visitor.Visit(node)

	// Если результат не nil, продолжаем обход детей
	if result == nil {
		for _, child := range node.Children() {
			Walk(visitor, child)
		}
	}
}

// BaseVisitor - базовая реализация Visitor
type BaseVisitor struct{}

func (v *BaseVisitor) Visit(node Node) interface{} {
	return nil // продолжаем обход
}

// PrintVisitor - visitor для печати дерева
type PrintVisitor struct {
	BaseVisitor
	indent int
}

func NewPrintVisitor() *PrintVisitor {
	return &PrintVisitor{indent: 0}
}

func (v *PrintVisitor) Visit(node Node) interface{} {
	indent := strings.Repeat("  ", v.indent)
	fmt.Printf("%s%s\n", indent, node.String())
	v.indent++
	return nil
}

// Print - удобная функция для печати AST
func Print(root Node) {
	visitor := NewPrintVisitor()
	Walk(visitor, root)
}

// CounterVisitor - visitor для подсчета узлов
type CounterVisitor struct {
	BaseVisitor
	count  int
	byType map[NodeType]int
}

func NewCounterVisitor() *CounterVisitor {
	return &CounterVisitor{
		byType: make(map[NodeType]int),
	}
}

func (v *CounterVisitor) Visit(node Node) interface{} {
	v.count++
	v.byType[node.Type()]++
	return nil
}

// CountNodes - подсчитывает общее количество узлов
func CountNodes(root Node) int {
	visitor := NewCounterVisitor()
	Walk(visitor, root)
	return visitor.count
}

// CountNodesByType - подсчитывает узлы по типам
func CountNodesByType(root Node) map[NodeType]int {
	visitor := NewCounterVisitor()
	Walk(visitor, root)
	return visitor.byType
}

// FindVisitor - visitor для поиска узлов по типу
type FindVisitor struct {
	BaseVisitor
	targetType NodeType
	found      []Node
}

func NewFindVisitor(targetType NodeType) *FindVisitor {
	return &FindVisitor{
		targetType: targetType,
		found:      make([]Node, 0),
	}
}

func (v *FindVisitor) Visit(node Node) interface{} {
	if node.Type() == v.targetType {
		v.found = append(v.found, node)
	}
	return nil
}

// FindNodesByType - находит все узлы указанного типа
func FindNodesByType(root Node, targetType NodeType) []Node {
	visitor := NewFindVisitor(targetType)
	Walk(visitor, root)
	return visitor.found
}

// CloneVisitor - visitor для клонирования дерева
type CloneVisitor struct {
	BaseVisitor
	clones map[Node]Node
}

func NewCloneVisitor() *CloneVisitor {
	return &CloneVisitor{
		clones: make(map[Node]Node),
	}
}

func (v *CloneVisitor) Visit(node Node) interface{} {
	// Если уже клонировали, возвращаем клон
	if clone, exists := v.clones[node]; exists {
		return clone
	}

	// Клонируем узел в зависимости от типа
	var clone Node

	switch n := node.(type) {
	case *ProgramNode:
		clone = v.cloneProgram(n)
	case *ParenthesesNode:
		clone = v.cloneParentheses(n)
	case *ArrayLiteral:
		clone = v.cloneArrayLiteral(n)
	case *ObjectLiteral:
		clone = v.cloneObjectLiteral(n)
	case *VariableAssignment:
		clone = v.cloneVariableAssignment(n)
	case *VariableRead:
		clone = v.cloneVariableRead(n)
	case *Identifier:
		clone = v.cloneIdentifier(n)
	default:
		clone = nil
	}

	if clone != nil {
		v.clones[node] = clone
	}

	return clone // возвращаем клон, чтобы не обходить детей оригинала
}

func (v *CloneVisitor) cloneProgram(node *ProgramNode) *ProgramNode {
	clone := NewProgramNode()
	for _, child := range node.Nodes() {
		childClone := v.clones[child]
		if childClone != nil {
			clone.AddNode(childClone)
		}
	}
	return clone
}

func (v *CloneVisitor) cloneParentheses(node *ParenthesesNode) *ParenthesesNode {
	clone := NewParenthesesNode(node.LeftToken(), node.RightToken())
	for _, child := range node.Children() {
		childClone := v.clones[child]
		if childClone != nil {
			clone.AddChild(childClone)
		}
	}
	return clone
}

func (v *CloneVisitor) cloneArrayLiteral(node *ArrayLiteral) *ArrayLiteral {
	clone := NewArrayLiteral(node.LeftToken(), node.RightToken())
	for _, element := range node.Elements {
		if elemNode, ok := element.(Node); ok {
			if elemClone, exists := v.clones[elemNode]; exists {
				clone.AddElement(elemClone.(Expression))
			}
		}
	}
	return clone
}

func (v *CloneVisitor) cloneObjectLiteral(node *ObjectLiteral) *ObjectLiteral {
	clone := NewObjectLiteral(node.LeftToken(), node.RightToken())
	for _, prop := range node.Properties {
		var keyClone, valueClone Expression
		if keyNode, ok := prop.Key.(Node); ok {
			if k, exists := v.clones[keyNode]; exists {
				keyClone = k.(Expression)
			}
		}
		if valueNode, ok := prop.Value.(Node); ok {
			if v, exists := v.clones[valueNode]; exists {
				valueClone = v.(Expression)
			}
		}
		if keyClone != nil && valueClone != nil {
			clone.AddProperty(keyClone, valueClone)
		}
	}
	return clone
}

func (v *CloneVisitor) cloneVariableAssignment(node *VariableAssignment) *VariableAssignment {
	var identifierClone *Identifier
	if id, exists := v.clones[node.Variable]; exists {
		identifierClone = id.(*Identifier)
	}
	var valueClone Expression
	if valueNode, ok := node.Value.(Node); ok {
		if v, exists := v.clones[valueNode]; exists {
			valueClone = v.(Expression)
		}
	}
	if identifierClone != nil && valueClone != nil {
		return NewVariableAssignment(identifierClone, node.Assign, valueClone)
	}
	return nil
}

func (v *CloneVisitor) cloneVariableRead(node *VariableRead) *VariableRead {
	var identifierClone *Identifier
	if id, exists := v.clones[node.Variable]; exists {
		identifierClone = id.(*Identifier)
	}
	if identifierClone != nil {
		return NewVariableRead(identifierClone)
	}
	return nil
}

func (v *CloneVisitor) cloneIdentifier(node *Identifier) *Identifier {
	return NewIdentifier(node.Token, node.Name)
}

// Clone - клонирует дерево
func Clone(root Node) Node {
	visitor := NewCloneVisitor()
	Walk(visitor, root)
	return visitor.clones[root]
}
