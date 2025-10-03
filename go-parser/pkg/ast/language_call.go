package ast

// LanguageCall - узел для вызова функции другого языка
type LanguageCall struct {
	Language  string       // "lua", "python"
	Function  string       // "print", "math.sqrt"
	Arguments []Expression // Аргументы функции
	Pos       Position     // Позиция в коде
}

// statementMarker реализует интерфейс Statement
func (lc *LanguageCall) statementMarker() {}

// expressionMarker реализует интерфейс Expression
func (lc *LanguageCall) expressionMarker() {}

// ToMap преобразует узел в map для сериализации
func (lc *LanguageCall) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"type":      "LanguageCall",
		"language":  lc.Language,
		"function":  lc.Function,
		"arguments": lc.argumentsToSlice(),
		"position":  lc.Pos.ToMap(),
	}
}

// Position возвращает позицию узла в коде
func (lc *LanguageCall) Position() Position {
	return lc.Pos
}

// argumentsToSlice конвертирует аргументы в слайс интерфейсов
func (lc *LanguageCall) argumentsToSlice() []interface{} {
	result := make([]interface{}, len(lc.Arguments))
	for i, arg := range lc.Arguments {
		result[i] = arg.ToMap()
	}
	return result
}
