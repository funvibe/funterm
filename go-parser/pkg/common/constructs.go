package common

// ConstructType представляет тип синтаксической конструкции
type ConstructType string

const (
	ConstructLiteral          ConstructType = "literal"           // Литералы (числа, строки)
	ConstructFunction         ConstructType = "function"          // Функции
	ConstructVariable         ConstructType = "variable"          // Переменные
	ConstructGroup            ConstructType = "group"             // Группирующие конструкции (скобки)
	ConstructArray            ConstructType = "array"             // Массивы
	ConstructObject           ConstructType = "object"            // Объекты
	ConstructAssignment       ConstructType = "assignment"        // Присваивание переменных
	ConstructVariableRead     ConstructType = "variable_read"     // Чтение переменных
	ConstructIdentifierRead   ConstructType = "identifier_read"   // Чтение идентификаторов
	ConstructForInLoop        ConstructType = "for_in_loop"       // Python for-in циклы
	ConstructNumericForLoop   ConstructType = "numeric_for_loop"  // Lua числовые циклы
	ConstructCStyleForLoop    ConstructType = "c_style_for_loop"  // C-style for циклы
	ConstructWhileLoop        ConstructType = "while_loop"        // While циклы
	ConstructBreak            ConstructType = "break"             // Break оператор
	ConstructContinue         ConstructType = "continue"          // Continue оператор
	ConstructIf               ConstructType = "if"                // If/else конструкции
	ConstructMatch            ConstructType = "match"             // Pattern matching конструкции
	ConstructBitstring        ConstructType = "bitstring"         // Bitstring конструкции
	ConstructPipe             ConstructType = "pipe"              // Pipe конструкции
	ConstructLanguageCall     ConstructType = "language_call"     // Language call конструкции
	ConstructBinaryExpression ConstructType = "binary_expression" // Бинарные выражения
	ConstructUnaryExpression  ConstructType = "unary_expression"  // Унарные выражения
	ConstructExpression       ConstructType = "expression"        // Общие выражения
	ConstructElvisExpression  ConstructType = "elvis_expression"  // Elvis выражения (тернарный оператор)
	// Native Code Integration конструкции (Task 25)
	ConstructImportStatement ConstructType = "import_statement" // Import конструкции
	ConstructCodeBlock       ConstructType = "code_block"       // Code block конструкции
)

// String возвращает строковое представление типа конструкции
func (ct ConstructType) String() string {
	return string(ct)
}
