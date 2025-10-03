package ast

import (
	"fmt"
	"strings"

	"go-parser/pkg/lexer"
)

// BitstringExpression представляет конструкцию битовой строки <<segment1, segment2, ...>>
type BitstringExpression struct {
	BaseNode
	LeftAngle  lexer.Token // Левый двойной угол <<
	RightAngle lexer.Token // Правый двойной угол >>
	Segments   []BitstringSegment
	Pos        Position
}

// NewBitstringExpression создает новый узел битовой строки
func NewBitstringExpression(leftAngle, rightAngle lexer.Token) *BitstringExpression {
	return &BitstringExpression{
		LeftAngle:  leftAngle,
		RightAngle: rightAngle,
		Segments:   make([]BitstringSegment, 0),
	}
}

// AddSegment добавляет сегмент в битовую строку
func (n *BitstringExpression) AddSegment(segment BitstringSegment) {
	n.Segments = append(n.Segments, segment)
	// Добавляем дочерние узлы
	if valueNode, ok := segment.Value.(Node); ok {
		n.AddChild(valueNode)
	}
	if segment.Size != nil {
		if sizeNode, ok := segment.Size.(Node); ok {
			n.AddChild(sizeNode)
		}
	}
	if segment.SizeExpression != nil {
		n.AddChild(segment.SizeExpression)
	}
}

// Type возвращает тип узла
func (n *BitstringExpression) Type() NodeType {
	return NodeBitstringExpression
}

// String возвращает строковое представление
func (n *BitstringExpression) String() string {
	var builder strings.Builder
	builder.WriteString("BitstringExpression")

	builder.WriteString("<<")
	for i, segment := range n.Segments {
		if i > 0 {
			builder.WriteString(", ")
		}
		builder.WriteString(segment.String())
	}
	builder.WriteString(">>")

	return builder.String()
}

// Position возвращает позицию узла в коде
func (n *BitstringExpression) Position() Position {
	return n.Pos
}

// ToMap преобразует узел в map для сериализации
func (n *BitstringExpression) ToMap() map[string]interface{} {
	segments := make([]interface{}, len(n.Segments))
	for i, segment := range n.Segments {
		segments[i] = segment.ToMap()
	}

	return map[string]interface{}{
		"type":     "bitstring",
		"segments": segments,
		"position": n.Pos.ToMap(),
	}
}

// expressionMarker реализует интерфейс Expression
func (n *BitstringExpression) expressionMarker() {}

// statementMarker реализует интерфейс Statement для standalone использования
func (n *BitstringExpression) statementMarker() {}

// SizeExpression представляет динамическое выражение размера
type SizeExpression struct {
	BaseNode
	ExprType   string     // "variable", "expression", "literal"
	Variable   string     // Имя переменной для простых ссылок (ExprType == "variable")
	Expression Expression // Выражение для сложных выражений (ExprType == "expression")
	Literal    Expression // Литеральное значение (ExprType == "literal")
	Pos        Position   // Позиция в коде
}

// NewSizeExpression создает новое выражение размера
func NewSizeExpression() *SizeExpression {
	return &SizeExpression{}
}

// Type возвращает тип узла
func (n *SizeExpression) Type() NodeType {
	return NodeSizeExpression
}

// String возвращает строковое представление
func (n *SizeExpression) String() string {
	switch n.ExprType {
	case "variable":
		return n.Variable
	case "expression":
		if n.Expression != nil {
			if node, ok := n.Expression.(Node); ok {
				return node.String()
			}
		}
		return "expression"
	case "literal":
		if n.Literal != nil {
			if node, ok := n.Literal.(Node); ok {
				return node.String()
			}
		}
		return "literal"
	default:
		return "SizeExpression"
	}
}

// Position возвращает позицию узла в коде
func (n *SizeExpression) Position() Position {
	return n.Pos
}

// ToMap преобразует узел в map для сериализации
func (n *SizeExpression) ToMap() map[string]interface{} {
	result := map[string]interface{}{
		"expr_type": n.ExprType,
		"position":  n.Pos.ToMap(),
	}

	if n.Variable != "" {
		result["variable"] = n.Variable
	}
	if n.Expression != nil {
		result["expression"] = n.Expression.ToMap()
	}
	if n.Literal != nil {
		result["literal"] = n.Literal.ToMap()
	}

	return result
}

// expressionMarker реализует интерфейс Expression
func (n *SizeExpression) expressionMarker() {}

// statementMarker реализует интерфейс Statement для standalone использования
func (n *SizeExpression) statementMarker() {}

// BitstringSegment представляет один сегмент битовой строки Value:Size/Specifiers
type BitstringSegment struct {
	BaseNode
	Value          Expression      // Значение (Literal, VariableRead, и т.д.)
	Size           Expression      // Размер в битах (опционально) - для обратной совместимости
	SizeExpression *SizeExpression // Новое поле для динамических выражений размера
	IsDynamicSize  bool            // Флаг указывающий на динамический размер
	Specifiers     []string        // Список спецификаторов
	ColonToken     lexer.Token     // Токен : (опционально)
	SlashToken     lexer.Token     // Токен / (опционально)
}

// String возвращает строковое представление сегмента
func (s *BitstringSegment) String() string {
	var builder strings.Builder

	// Value
	if valueNode, ok := s.Value.(Node); ok {
		builder.WriteString(valueNode.String())
	} else {
		builder.WriteString(fmt.Sprintf("%v", s.Value.ToMap()))
	}

	// Size
	if s.Size != nil {
		builder.WriteString(":")
		if sizeNode, ok := s.Size.(Node); ok {
			builder.WriteString(sizeNode.String())
		} else {
			builder.WriteString(fmt.Sprintf("%v", s.Size.ToMap()))
		}
	}

	// Specifiers
	if len(s.Specifiers) > 0 {
		builder.WriteString("/")
		for i, spec := range s.Specifiers {
			if i > 0 {
				builder.WriteString(",")
			}
			builder.WriteString(spec)
		}
	}

	return builder.String()
}

// ToMap преобразует сегмент в map для сериализации
func (s *BitstringSegment) ToMap() map[string]interface{} {
	result := map[string]interface{}{
		"value": s.Value.ToMap(),
	}

	if s.Size != nil {
		result["size"] = s.Size.ToMap()
	}

	if s.SizeExpression != nil {
		result["size_expression"] = s.SizeExpression.ToMap()
		result["is_dynamic_size"] = s.IsDynamicSize
	}

	if len(s.Specifiers) > 0 {
		// Преобразуем []string в []interface{} для совместимости с ToMap
		specifiers := make([]interface{}, len(s.Specifiers))
		for i, spec := range s.Specifiers {
			specifiers[i] = spec
		}
		result["specifiers"] = specifiers
	}

	return result
}
