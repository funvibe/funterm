package ast

import (
	"fmt"
)

// Базовые интерфейсы для прототипа по ТЗ

// ProtoNode - базовый интерфейс для всех узлов AST (по ТЗ)
type ProtoNode interface {
	ToMap() map[string]interface{}
	Position() Position
}

// Statement - интерфейс для выражений-инструкций
type Statement interface {
	ProtoNode
	statementMarker()
}

// Expression - интерфейс для выражений
type Expression interface {
	ProtoNode
	expressionMarker()
}

// ParseErrorType - типы ошибок парсинга
type ParseErrorType int

const (
	ErrorSyntax ParseErrorType = iota
	ErrorSemantic
	ErrorType
)

// ParseError - ошибка парсинга (по ТЗ)
type ParseError struct {
	Type     ParseErrorType
	Position Position
	Message  string
	Context  string
}

// Error реализует интерфейс error
func (pe *ParseError) Error() string {
	return pe.String()
}

// String возвращает строковое представление ошибки
func (pe *ParseError) String() string {
	var typeStr string
	switch pe.Type {
	case ErrorSyntax:
		typeStr = "syntax"
	case ErrorSemantic:
		typeStr = "semantic"
	case ErrorType:
		typeStr = "type"
	default:
		typeStr = "unknown"
	}
	return fmt.Sprintf("%s error at line %d, column %d: %s",
		typeStr, pe.Position.Line, pe.Position.Column, pe.Message)
}
