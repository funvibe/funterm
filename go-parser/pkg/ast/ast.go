package ast

type ExpressionStatement struct {
    Expression Expression
}

// statementMarker implements Statement interface
func (es *ExpressionStatement) statementMarker() {}

// Position implements ProtoNode interface
func (es *ExpressionStatement) Position() Position {
    if es.Expression != nil {
        return es.Expression.Position()
    }
    return Position{Line: 1, Column: 1, Offset: 0}
}

// ToMap implements ProtoNode interface
func (es *ExpressionStatement) ToMap() map[string]interface{} {
    return map[string]interface{}{
        "type":       "ExpressionStatement",
        "expression": es.Expression.ToMap(),
    }
}