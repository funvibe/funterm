package ast

import (
	"fmt"
	"strings"

	"go-parser/pkg/lexer"
)

// ImportStatement представляет импорт внешнего файла: import lua "path/to/file.lua"
type ImportStatement struct {
	BaseNode
	ImportToken lexer.Token    // токен 'import'
	Runtime     lexer.Token    // токен рантайма ('lua', 'python', 'py')
	Path        *StringLiteral // путь к файлу
	Pos         Position       // позиция начала импорта
}

// NewImportStatement создает новый узел импорта
func NewImportStatement(importToken, runtimeToken lexer.Token, path *StringLiteral) *ImportStatement {
	return &ImportStatement{
		ImportToken: importToken,
		Runtime:     runtimeToken,
		Path:        path,
		Pos:         tokenToPosition(importToken),
	}
}

// Type возвращает тип узла
func (n *ImportStatement) Type() NodeType {
	return NodeImportStatement
}

// statementMarker реализует интерфейс Statement
func (n *ImportStatement) statementMarker() {}

// Position возвращает позицию узла
func (n *ImportStatement) Position() Position {
	return n.Pos
}

// String возвращает строковое представление
func (n *ImportStatement) String() string {
	return fmt.Sprintf("ImportStatement(%s %s)", n.Runtime.Value, n.Path.String())
}

// ToMap преобразует узел в map для сериализации
func (n *ImportStatement) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"type":     "import_statement",
		"runtime":  n.Runtime.Value,
		"path":     n.Path.ToMap(),
		"position": n.Pos.ToMap(),
	}
}

// CodeBlockStatement представляет блок кода: lua { ... } или python { ... }
// Новый синтаксис: py (var1, var2) { ... }
type CodeBlockStatement struct {
	BaseNode
	RuntimeToken   lexer.Token   // токен рантайма ('lua', 'python', 'py')
	VariableTokens []lexer.Token // токены переменных в скобках (если есть)
	LParenToken    lexer.Token   // токен '(' (если есть переменные)
	RParenToken    lexer.Token   // токен ')' (если есть переменные)
	LBraceToken    lexer.Token   // токен '{'
	RBraceToken    lexer.Token   // токен '}'
	Code           string        // сырой код внутри фигурных скобок
	Pos            Position      // позиция начала блока
}

// NewCodeBlockStatement создает новый узел блока кода
func NewCodeBlockStatement(runtimeToken lexer.Token, variableTokens []lexer.Token, lParenToken, rParenToken, lBraceToken, rBraceToken lexer.Token, code string) *CodeBlockStatement {
	return &CodeBlockStatement{
		RuntimeToken:   runtimeToken,
		VariableTokens: variableTokens,
		LParenToken:    lParenToken,
		RParenToken:    rParenToken,
		LBraceToken:    lBraceToken,
		RBraceToken:    rBraceToken,
		Code:           code,
		Pos:            tokenToPosition(runtimeToken),
	}
}

// GetVariableNames возвращает имена переменных для сохранения
func (n *CodeBlockStatement) GetVariableNames() []string {
	var names []string
	for _, token := range n.VariableTokens {
		names = append(names, token.Value)
	}
	return names
}

// HasVariables проверяет, есть ли переменные для сохранения
func (n *CodeBlockStatement) HasVariables() bool {
	return len(n.VariableTokens) > 0
}

// Type возвращает тип узла
func (n *CodeBlockStatement) Type() NodeType {
	return NodeCodeBlockStatement
}

// statementMarker реализует интерфейс Statement
func (n *CodeBlockStatement) statementMarker() {}

// Position возвращает позицию узла
func (n *CodeBlockStatement) Position() Position {
	return n.Pos
}

// String возвращает строковое представление
func (n *CodeBlockStatement) String() string {
	var builder strings.Builder
	
	if n.HasVariables() {
		// Новый синтаксис: py (var1, var2) { ... }
		builder.WriteString(fmt.Sprintf("CodeBlockStatement(%s (", n.RuntimeToken.Value))
		for i, token := range n.VariableTokens {
			if i > 0 {
				builder.WriteString(", ")
			}
			builder.WriteString(token.Value)
		}
		builder.WriteString(") {")
	} else {
		// Старый синтаксис: py { ... }
		builder.WriteString(fmt.Sprintf("CodeBlockStatement(%s {", n.RuntimeToken.Value))
	}

	// Показываем первые несколько строк кода для краткости
	lines := strings.Split(n.Code, "\n")
	if len(lines) <= 3 {
		builder.WriteString(n.Code)
	} else {
		builder.WriteString(lines[0])
		builder.WriteString("\n  ")
		builder.WriteString(lines[1])
		builder.WriteString("\n  ")
		builder.WriteString(lines[2])
		builder.WriteString("\n  ...")
	}

	builder.WriteString("})")
	return builder.String()
}

// ToMap преобразует узел в map для сериализации
func (n *CodeBlockStatement) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"type":     "code_block_statement",
		"runtime":  n.RuntimeToken.Value,
		"code":     n.Code,
		"position": n.Pos.ToMap(),
	}
}
