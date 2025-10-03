package ast

import (
	"fmt"
	"strings"

	"go-parser/pkg/lexer"
)

// formatArguments форматирует аргументы для строкового представления
func formatArguments(arguments []Expression) string {
	if len(arguments) == 0 {
		return ""
	}

	var args []string
	for _, arg := range arguments {
		args = append(args, fmt.Sprintf("%v", arg.ToMap()))
	}

	return strings.Join(args, ", ")
}

// PipeExpression - узел для представления пайплайнов
type PipeExpression struct {
	BaseNode
	Stages    []Expression  // Этапы пайплайна
	Operators []lexer.Token // Операторы | между этапами
	Pos       Position      // Позиция в коде
}

// NewPipeExpression создает новый узел PipeExpression
func NewPipeExpression(stages []Expression, operators []lexer.Token, pos Position) *PipeExpression {
	return &PipeExpression{
		Stages:    stages,
		Operators: operators,
		Pos:       pos,
	}
}

// Type возвращает тип узла
func (pe *PipeExpression) Type() NodeType {
	return NodePipeExpression
}

// String возвращает строковое представление
func (pe *PipeExpression) String() string {
	var builder strings.Builder
	builder.WriteString("PipeExpression(")

	for i, stage := range pe.Stages {
		if i > 0 {
			builder.WriteString(" | ")
		}
		if node, ok := stage.(Node); ok {
			builder.WriteString(node.String())
		} else {
			builder.WriteString(fmt.Sprintf("%v", stage.ToMap()))
		}
	}

	builder.WriteString(")")
	return builder.String()
}

// Position возвращает позицию узла в коде
func (pe *PipeExpression) Position() Position {
	return pe.Pos
}

// ToMap преобразует узел в map для сериализации
func (pe *PipeExpression) ToMap() map[string]interface{} {
	stages := make([]interface{}, len(pe.Stages))
	for i, stage := range pe.Stages {
		stages[i] = stage.ToMap()
	}

	operators := make([]interface{}, len(pe.Operators))
	for i, op := range pe.Operators {
		operators[i] = map[string]interface{}{
			"type":  op.Type.String(),
			"value": op.Value,
			"line":  op.Line,
			"col":   op.Column,
		}
	}

	return map[string]interface{}{
		"type":      "pipe_expression",
		"stages":    stages,
		"operators": operators,
		"position":  pe.Pos.ToMap(),
	}
}

// expressionMarker реализует интерфейс Expression
func (pe *PipeExpression) expressionMarker() {}

// statementMarker реализует интерфейс Statement
func (pe *PipeExpression) statementMarker() {}

// LanguageCallStatement - узел для вызова функции другого языка с поддержкой background tasks
type LanguageCallStatement struct {
	LanguageCall    *LanguageCall
	IsBackground    bool        // true для background tasks (&)
	BackgroundToken lexer.Token // Токен & для background tasks
	Pos             Position    // Позиция в коде
}

// NewLanguageCallStatement создает новый узел LanguageCallStatement
func NewLanguageCallStatement(languageCall *LanguageCall, pos Position) *LanguageCallStatement {
	return &LanguageCallStatement{
		LanguageCall: languageCall,
		IsBackground: false,
		Pos:          pos,
	}
}

// NewBackgroundLanguageCallStatement создает новый узел LanguageCallStatement для background task
func NewBackgroundLanguageCallStatement(languageCall *LanguageCall, backgroundToken lexer.Token, pos Position) *LanguageCallStatement {
	return &LanguageCallStatement{
		LanguageCall:    languageCall,
		IsBackground:    true,
		BackgroundToken: backgroundToken,
		Pos:             pos,
	}
}

// Type возвращает тип узла
func (lcs *LanguageCallStatement) Type() NodeType {
	return NodeLanguageCallStatement
}

// String возвращает строковое представление
func (lcs *LanguageCallStatement) String() string {
	var builder strings.Builder
	builder.WriteString("LanguageCallStatement(")

	if lcs.LanguageCall != nil {
		builder.WriteString(fmt.Sprintf("%s.%s(%s)", lcs.LanguageCall.Language, lcs.LanguageCall.Function, formatArguments(lcs.LanguageCall.Arguments)))
	}

	if lcs.IsBackground {
		builder.WriteString(" &")
	}

	builder.WriteString(")")
	return builder.String()
}

// Position возвращает позицию узла в коде
func (lcs *LanguageCallStatement) Position() Position {
	return lcs.Pos
}

// ToMap преобразует узел в map для сериализации
func (lcs *LanguageCallStatement) ToMap() map[string]interface{} {
	result := map[string]interface{}{
		"type":          "language_call_statement",
		"position":      lcs.Pos.ToMap(),
		"is_background": lcs.IsBackground,
	}

	if lcs.LanguageCall != nil {
		result["language_call"] = lcs.LanguageCall.ToMap()
	}

	if lcs.IsBackground {
		result["background_token"] = map[string]interface{}{
			"type":  lcs.BackgroundToken.Type.String(),
			"value": lcs.BackgroundToken.Value,
			"line":  lcs.BackgroundToken.Line,
			"col":   lcs.BackgroundToken.Column,
		}
	}

	return result
}

// statementMarker реализует интерфейс Statement
func (lcs *LanguageCallStatement) statementMarker() {}
