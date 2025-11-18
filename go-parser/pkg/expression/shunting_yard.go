package expression

import (
	"fmt"
	"math/big"
	"strconv"

	"go-parser/pkg/ast"
	"go-parser/pkg/lexer"
)

// OperatorInfo содержит информацию об операторе
type OperatorInfo struct {
	precedence  int
	associative bool // true для left-associative, false для right-associative
}

// Precedence значения операторов
const (
	PrecedenceLowest     = 0
	PrecedenceOr         = 1  // ||
	PrecedenceAnd        = 2  // &&
	PrecedenceBitwiseOr  = 3  // |
	PrecedenceBitwiseXor = 4  // ^
	PrecedenceBitwiseAnd = 5  // &
	PrecedenceEqual      = 6  // ==, !=
	PrecedenceCompare    = 7  // <, >, <=, >=
	PrecedenceAdd        = 8  // +, -
	PrecedenceMul        = 9  // *, /, %
	PrecedencePower      = 10 // **
	PrecedenceUnary      = 11 // ~, ! (unary)
	PrecedenceTernary    = 12 // ? :
)

// operatorPrecedence карта приоритетов операторов
var operatorPrecedence = map[lexer.TokenType]OperatorInfo{
	lexer.TokenOr:           {precedence: PrecedenceOr, associative: true},
	lexer.TokenAnd:          {precedence: PrecedenceAnd, associative: true},
	lexer.TokenBitwiseOr:    {precedence: PrecedenceBitwiseOr, associative: true},
	lexer.TokenCaret:        {precedence: PrecedenceBitwiseXor, associative: true},
	lexer.TokenAmpersand:    {precedence: PrecedenceBitwiseAnd, associative: true},
	lexer.TokenEqual:        {precedence: PrecedenceEqual, associative: true},
	lexer.TokenNotEqual:     {precedence: PrecedenceEqual, associative: true},
	lexer.TokenLess:         {precedence: PrecedenceCompare, associative: true},
	lexer.TokenGreater:      {precedence: PrecedenceCompare, associative: true},
	lexer.TokenLessEqual:    {precedence: PrecedenceCompare, associative: true},
	lexer.TokenGreaterEqual: {precedence: PrecedenceCompare, associative: true},
	lexer.TokenPlus:         {precedence: PrecedenceAdd, associative: true},
	lexer.TokenMinus:        {precedence: PrecedenceAdd, associative: true},
	lexer.TokenMultiply:     {precedence: PrecedenceMul, associative: true},
	lexer.TokenSlash:        {precedence: PrecedenceMul, associative: true},
	lexer.TokenModulo:       {precedence: PrecedenceMul, associative: true},
	lexer.TokenPower:        {precedence: PrecedencePower, associative: false}, // right-associative
	lexer.TokenNot:          {precedence: PrecedenceUnary, associative: false}, // right-associative (unary)
	lexer.TokenTilde:        {precedence: PrecedenceUnary, associative: false}, // right-associative (unary)
	lexer.TokenAssign:       {precedence: PrecedenceLowest, associative: true}, // assignment has lowest precedence
}

// ParseExpression парсит выражение используя алгоритм shunting-yard
func ParseExpression(tokens []lexer.Token) (ast.Expression, error) {
	return ParseExpressionWithDepth(tokens, 0)
}

// ParseTernaryExpression парсит тернарное выражение начиная с условия
func ParseTernaryExpression(condition ast.Expression, tokens []lexer.Token) (ast.Expression, error) {
	result, _, err := parseTernaryExpression(condition, tokens, 0)
	return result, err
}

// ParseExpressionWithDepth парсит выражение с защитой от бесконечной рекурсии
func ParseExpressionWithDepth(tokens []lexer.Token, depth int) (ast.Expression, error) {
	const maxDepth = 50
	if depth > maxDepth {
		return nil, fmt.Errorf("maximum recursion depth exceeded (%d)", maxDepth)
	}

	if len(tokens) == 0 {
		return nil, nil
	}

	// Стеки для алгоритма shunting-yard
	var outputQueue []ast.Expression
	var operatorStack []lexer.Token

	i := 0
	for i < len(tokens) {
		token := tokens[i]

		switch token.Type {
		case lexer.TokenNumber:
			// Числа идут в output queue
			expr, err := createNumberLiteral(token)
			if err != nil {
				return nil, err
			}
			outputQueue = append(outputQueue, expr)
			i++

		case lexer.TokenIdentifier:
			// Идентификаторы идут в output queue
			expr := createIdentifier(token)
			outputQueue = append(outputQueue, expr)
			i++

		case lexer.TokenTrue, lexer.TokenFalse:
			// Boolean литералы идут в output queue
			expr := &ast.BooleanLiteral{
				Value: token.Type == lexer.TokenTrue,
				Pos: ast.Position{
					Line:   token.Line,
					Column: token.Column,
					Offset: token.Position,
				},
			}
			outputQueue = append(outputQueue, expr)
			i++

		case lexer.TokenNil:
			// Nil литерал идет в output queue
			expr := ast.NewNilLiteral(token)
			outputQueue = append(outputQueue, expr)
			i++

		case lexer.TokenString:
			// Строковые литералы идут в output queue
			expr := &ast.StringLiteral{
				Value: token.Value,
				Pos: ast.Position{
					Line:   token.Line,
					Column: token.Column,
					Offset: token.Position,
				},
			}
			outputQueue = append(outputQueue, expr)
			i++

		case lexer.TokenNot, lexer.TokenTilde, lexer.TokenMinus:
			// Унарные операторы: !expr, ~expr, -expr
			// Обрабатываем специальным образом
			if i+1 >= len(tokens) {
				return nil, fmt.Errorf("missing operand for unary operator %s at line %d, column %d", token.Value, token.Line, token.Column)
			}

			// Парсим операнд
			operandTokens := tokens[i+1:]
			operand, consumed, err := parseUnaryOperand(operandTokens, depth)
			if err != nil {
				return nil, err
			}

			// Создаем унарное выражение
			pos := ast.Position{
				Line:   token.Line,
				Column: token.Column,
				Offset: token.Position,
			}
			unaryExpr := &ast.UnaryExpression{
				Operator: token.Value,
				Right:    operand,
				Pos:      pos,
			}

			outputQueue = append(outputQueue, unaryExpr)
			i += 1 + consumed

		case lexer.TokenQuestion:
			// Тернарный оператор: condition ? true_expr : false_expr
			// Condition уже должен быть в outputQueue
			if len(outputQueue) == 0 {
				return nil, fmt.Errorf("missing condition for ternary operator at line %d, column %d", token.Line, token.Column)
			}

			// Извлекаем condition из outputQueue
			condition := outputQueue[len(outputQueue)-1]
			outputQueue = outputQueue[:len(outputQueue)-1]

			// Парсим остаток тернарного выражения
			ternaryExpr, consumed, err := parseTernaryExpression(condition, tokens[i:], depth)
			if err != nil {
				return nil, err
			}

			outputQueue = append(outputQueue, ternaryExpr)
			i += consumed

		case lexer.TokenLeftParen:
			// Рекурсивно парсим вложенное выражение
			nestedExpr, consumed, err := parseNestedExpressionWithDepth(tokens[i:], depth+1)
			if err != nil {
				return nil, err
			}

			// Создаем NestedExpression
			pos := ast.Position{
				Line:   token.Line,
				Column: token.Column,
				Offset: token.Position,
			}
			nested := ast.NewNestedExpression(nestedExpr, pos)
			outputQueue = append(outputQueue, nested)

			i += consumed

		case lexer.TokenRightParen:
			// Несбалансированная скобка
			return nil, fmt.Errorf("несбалансированная скобка: неожиданная ')' в строке %d, колонка %d",
				token.Line, token.Column)

		default:
			// Проверяем, является ли токен оператором
			if info, isOp := operatorPrecedence[token.Type]; isOp {
				// Обрабатываем операторы в стеке
				for len(operatorStack) > 0 {
					topOp := operatorStack[len(operatorStack)-1]
					topInfo, topIsOp := operatorPrecedence[topOp.Type]

					if !topIsOp {
						break
					}

					// Если оператор на вершине стека имеет более высокий приоритет
					// или такой же приоритет и лево-ассоциативный
					if (info.associative && info.precedence <= topInfo.precedence) ||
						(!info.associative && info.precedence < topInfo.precedence) {

						// Перемещаем оператор в output queue
						binaryExpr, err := createBinaryExpressionFromStack(topOp, &outputQueue)
						if err != nil {
							return nil, err
						}
						outputQueue = append(outputQueue, binaryExpr)
						operatorStack = operatorStack[:len(operatorStack)-1]
					} else {
						break
					}
				}

				// Добавляем текущий оператор в стек
				operatorStack = append(operatorStack, token)
				i++
			} else {
				// Неизвестный токен
				return nil, fmt.Errorf("unknown token in expression: %s at line %d, column %d",
					token.Value, token.Line, token.Column)
			}
		}
	}

	// Перемещаем оставшиеся операторы в output queue
	for len(operatorStack) > 0 {
		op := operatorStack[len(operatorStack)-1]
		operatorStack = operatorStack[:len(operatorStack)-1]

		if op.Type == lexer.TokenLeftParen {
			return nil, fmt.Errorf("несбалансированная скобка: отсутствует ')' в строке %d, колонка %d",
				op.Line, op.Column)
		}

		binaryExpr, err := createBinaryExpressionFromStack(op, &outputQueue)
		if err != nil {
			return nil, err
		}
		outputQueue = append(outputQueue, binaryExpr)
	}

	// Возвращаем единственное выражение из output queue
	if len(outputQueue) != 1 {
		return nil, fmt.Errorf("ошибка парсинга: ожидалось одно выражение, получено %d", len(outputQueue))
	}

	return outputQueue[0], nil
}

// parseNestedExpression рекурсивно парсит выражение в скобках
func parseNestedExpression(tokens []lexer.Token) (ast.Expression, int, error) {
	return parseNestedExpressionWithDepth(tokens, 0)
}

// parseNestedExpressionWithDepth парсит вложенное выражение с защитой глубины
func parseNestedExpressionWithDepth(tokens []lexer.Token, depth int) (ast.Expression, int, error) {
	if len(tokens) == 0 || tokens[0].Type != lexer.TokenLeftParen {
		return nil, 0, fmt.Errorf("ожидалась '('")
	}

	// Ищем соответствующую закрывающую скобку
	parenDepth := 1
	i := 1
	for i < len(tokens) && parenDepth > 0 {
		switch tokens[i].Type {
		case lexer.TokenLeftParen:
			parenDepth++
		case lexer.TokenRightParen:
			parenDepth--
		}
		i++
	}

	if parenDepth > 0 {
		return nil, 0, fmt.Errorf("несбалансированная скобка: отсутствует ')' в строке %d, колонка %d",
			tokens[0].Line, tokens[0].Column)
	}

	// Извлекаем токены внутри скобок (без внешних скобок)
	innerTokens := tokens[1 : i-1]
	if len(innerTokens) == 0 {
		return nil, 0, fmt.Errorf("пустые скобки")
	}

	// Рекурсивно парсим внутреннее выражение
	expr, err := ParseExpressionWithDepth(innerTokens, depth)
	if err != nil {
		return nil, 0, err
	}

	return expr, i, nil
}

// createNumberLiteral создает числовой литерал
func createNumberLiteral(token lexer.Token) (ast.Expression, error) {
	pos := ast.Position{
		Line:   token.Line,
		Column: token.Column,
		Offset: token.Position,
	}

	// Пытаемся парсить как integer, потом как float
	if intVal, err := strconv.ParseInt(token.Value, 10, 64); err == nil {
		return &ast.NumberLiteral{
			IntValue: big.NewInt(intVal),
			IsInt:    true,
			Pos:      pos,
		}, nil
	}

	if floatVal, err := strconv.ParseFloat(token.Value, 64); err == nil {
		return &ast.NumberLiteral{
			FloatValue: floatVal,
			IsInt:      false,
			Pos:        pos,
		}, nil
	}

	return nil, fmt.Errorf("неверный числовой литерал: %s", token.Value)
}

// createIdentifier создает идентификатор
func createIdentifier(token lexer.Token) ast.Expression {
	return &ast.Identifier{
		Name: token.Value,
		Pos: ast.Position{
			Line:   token.Line,
			Column: token.Column,
			Offset: token.Position,
		},
	}
}

// parseUnaryOperand парсит операнд для унарного оператора
func parseUnaryOperand(tokens []lexer.Token, depth int) (ast.Expression, int, error) {
	if len(tokens) == 0 {
		return nil, 0, fmt.Errorf("missing operand for unary operator")
	}

	token := tokens[0]
	switch token.Type {
	case lexer.TokenNumber:
		expr, err := createNumberLiteral(token)
		if err != nil {
			return nil, 0, err
		}
		return expr, 1, nil

	case lexer.TokenIdentifier:
		expr := createIdentifier(token)
		return expr, 1, nil

	case lexer.TokenLeftParen:
		// Вложенное выражение
		nestedExpr, consumed, err := parseNestedExpressionWithDepth(tokens, depth)
		if err != nil {
			return nil, 0, err
		}
		return nestedExpr, consumed, nil

	case lexer.TokenNot, lexer.TokenTilde:
		// Другой унарный оператор
		operand, consumed, err := parseUnaryOperand(tokens[1:], depth)
		if err != nil {
			return nil, 0, err
		}

		pos := ast.Position{
			Line:   token.Line,
			Column: token.Column,
			Offset: token.Position,
		}
		unaryExpr := &ast.UnaryExpression{
			Operator: token.Value,
			Right:    operand,
			Pos:      pos,
		}
		return unaryExpr, 1 + consumed, nil

	default:
		return nil, 0, fmt.Errorf("неподдерживаемый операнд для унарного оператора: %s", token.Value)
	}
}

// parseTernaryExpression парсит тернарное выражение condition ? true_expr : false_expr
// или Elvis оператор condition ?: fallback_expr
func parseTernaryExpression(condition ast.Expression, tokens []lexer.Token, depth int) (ast.Expression, int, error) {
	if len(tokens) == 0 || tokens[0].Type != lexer.TokenQuestion {
		return nil, 0, fmt.Errorf("expected ternary operator '?'")
	}

	// Проверяем, является ли это Elvis оператором (?:)
	if len(tokens) > 1 && tokens[1].Type == lexer.TokenColon {
		// Это Elvis оператор: condition ?: fallback_expr
		colonToken := tokens[1]

		// Парсим fallback_expr (после :)
		falseTokens := tokens[2:]
		falseExpr, err := ParseExpressionWithDepth(falseTokens, depth)
		if err != nil {
			return nil, 0, fmt.Errorf("error parsing fallback-expression in Elvis operator: %v", err)
		}

		// Создаем Elvis выражение (как особый случай тернарного)
		// В тернарном выражении: left = condition, op1 = ?, op2 = :, true_expr = condition, false_expr = fallback_expr
		pos := ast.Position{
			Line:   tokens[0].Line,
			Column: tokens[0].Column,
			Offset: tokens[0].Position,
		}

		ternary := ast.NewTernaryExpression(condition, tokens[0], colonToken, condition, falseExpr, pos)

		return ternary, len(tokens), nil
	}

	// Находим позицию двоеточия (:) для тернарного оператора
	colonIndex := -1
	parenDepth := 0
	for i, token := range tokens {
		if token.Type == lexer.TokenLeftParen {
			parenDepth++
		} else if token.Type == lexer.TokenRightParen {
			parenDepth--
		} else if token.Type == lexer.TokenColon && parenDepth == 0 {
			colonIndex = i
			break
		}
	}

	if colonIndex == -1 {
		return nil, 0, fmt.Errorf("colon ':' not found in ternary expression")
	}

	// Парсим true_expr (между ? и :)
	trueTokens := tokens[1:colonIndex]
	trueExpr, err := ParseExpressionWithDepth(trueTokens, depth)
	if err != nil {
		return nil, 0, fmt.Errorf("error parsing true-expression in ternary operator: %v", err)
	}

	// Парсим false_expr (после :)
	falseTokens := tokens[colonIndex+1:]
	falseExpr, err := ParseExpressionWithDepth(falseTokens, depth)
	if err != nil {
		return nil, 0, fmt.Errorf("error parsing false-expression in ternary operator: %v", err)
	}

	// Создаем тернарное выражение
	pos := ast.Position{
		Line:   tokens[0].Line,
		Column: tokens[0].Column,
		Offset: tokens[0].Position,
	}

	ternary := ast.NewTernaryExpression(condition, tokens[0], tokens[colonIndex], trueExpr, falseExpr, pos)

	return ternary, len(tokens), nil
}

// createBinaryExpressionFromStack создает бинарное выражение из оператора и стека
func createBinaryExpressionFromStack(op lexer.Token, outputQueue *[]ast.Expression) (ast.Expression, error) {
	if len(*outputQueue) < 2 {
		return nil, fmt.Errorf("insufficient operands for operator %s", op.Value)
	}

	// Извлекаем правый и левый операнды
	right := (*outputQueue)[len(*outputQueue)-1]
	left := (*outputQueue)[len(*outputQueue)-2]

	// Удаляем операнды из очереди
	*outputQueue = (*outputQueue)[:len(*outputQueue)-2]

	// Создаем бинарное выражение
	pos := ast.Position{
		Line:   op.Line,
		Column: op.Column,
		Offset: op.Position,
	}

	return &ast.BinaryExpression{
		Left:     left,
		Operator: op.Value,
		Right:    right,
		BaseNode: ast.BaseNode{},
		Pos:      pos,
	}, nil
}
