//go:build !exclude_control_flow_tests

package engine

import (
	"fmt"
	"testing"

	"go-parser/pkg/ast"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIfStatement_Execution(t *testing.T) {
	t.Run("if (true) executes consequent", func(t *testing.T) {
		eng, mockRuntime := CreateEngineWithStatefulMock(t)

		consequentCall := &ast.LanguageCall{Language: "lua", Function: "consequent_called"}
		alternateCall := &ast.LanguageCall{Language: "lua", Function: "alternate_called"}

		ifStmt := &ast.IfStatement{
			Condition:  &ast.BooleanLiteral{Value: true},
			Consequent: &ast.BlockStatement{Statements: []ast.Statement{consequentCall}},
			Alternate:  &ast.BlockStatement{Statements: []ast.Statement{alternateCall}},
		}

		_, err := eng.executeIfStatement(ifStmt)
		require.NoError(t, err)

		calls := mockRuntime.GetCalls()
		assert.Contains(t, calls, "consequent_called([])", "Consequent block should have been called")
		assert.NotContains(t, calls, "alternate_called([])", "Alternate block should NOT have been called")
	})

	t.Run("if (false) executes alternate", func(t *testing.T) {
		eng, mockRuntime := CreateEngineWithStatefulMock(t)

		consequentCall := &ast.LanguageCall{Language: "lua", Function: "consequent_called"}
		alternateCall := &ast.LanguageCall{Language: "lua", Function: "alternate_called"}

		ifStmt := &ast.IfStatement{
			Condition:  &ast.BooleanLiteral{Value: false},
			Consequent: &ast.BlockStatement{Statements: []ast.Statement{consequentCall}},
			Alternate:  &ast.BlockStatement{Statements: []ast.Statement{alternateCall}},
		}

		_, err := eng.executeIfStatement(ifStmt)
		require.NoError(t, err)

		calls := mockRuntime.GetCalls()
		assert.NotContains(t, calls, "consequent_called([])", "Consequent block should NOT have been called")
		assert.Contains(t, calls, "alternate_called([])", "Alternate block should have been called")
	})

	t.Run("if (false) with no else does nothing", func(t *testing.T) {
		eng, mockRuntime := CreateEngineWithStatefulMock(t)

		consequentCall := &ast.LanguageCall{Language: "lua", Function: "consequent_called"}
		ifStmt := &ast.IfStatement{
			Condition:  &ast.BooleanLiteral{Value: false},
			Consequent: &ast.BlockStatement{Statements: []ast.Statement{consequentCall}},
			Alternate:  nil,
		}

		_, err := eng.executeIfStatement(ifStmt)
		require.NoError(t, err)
		assert.Empty(t, mockRuntime.GetCalls(), "No block should have been called")
	})
}

func TestWhileStatement_Execution(t *testing.T) {
	t.Run("while loop runs until condition is false", func(t *testing.T) {
		eng, mockRuntime := CreateEngineWithStatefulMock(t)

		var loopCount int
		// This mock simulates a condition that becomes false after 3 successful runs.
		mockRuntime.ExecuteFunctionFunc = func(name string, args []interface{}) (interface{}, error) {
			if name == "check_condition" {
				loopCount++
				return loopCount <= 3, nil // True for 3 iterations, then false
			}
			// This will be our loop body
			mockRuntime.mu.Lock()
			mockRuntime.calls = append(mockRuntime.calls, fmt.Sprintf("%s(%v)", name, args))
			mockRuntime.mu.Unlock()
			return nil, nil
		}

		manualCondition := &ast.LanguageCall{Language: "lua", Function: "check_condition"}
		incrementBody := &ast.LanguageCall{Language: "lua", Function: "increment_counter"}

		whileStmtManual := &ast.WhileStatement{
			Condition: manualCondition,
			Body:      &ast.BlockStatement{Statements: []ast.Statement{incrementBody}},
		}

		_, err := eng.executeWhileStatement(whileStmtManual)
		require.NoError(t, err)

		assert.Equal(t, 4, loopCount, "Condition should be checked 4 times (3 true, 1 false)")
		assert.Len(t, mockRuntime.GetCalls(), 3, "Loop body should execute 3 times")
		assert.Contains(t, mockRuntime.GetCalls(), "increment_counter([])")
	})

	t.Run("while loop respects break", func(t *testing.T) {
		eng, mockRuntime := CreateEngineWithStatefulMock(t)

		whileStmt := &ast.WhileStatement{
			Condition: &ast.BooleanLiteral{Value: true}, // Infinite loop
			Body: &ast.BlockStatement{
				Statements: []ast.Statement{
					&ast.LanguageCall{Language: "lua", Function: "body_called"},
					&ast.BreakStatement{},
					&ast.LanguageCall{Language: "lua", Function: "should_not_be_called"},
				},
			},
		}

		_, err := eng.executeWhileStatement(whileStmt)
		require.NoError(t, err)

		calls := mockRuntime.GetCalls()
		assert.Len(t, calls, 1, "Loop body should only be called once")
		assert.Equal(t, "body_called([])", calls[0])
	})

	t.Run("while loop respects continue", func(t *testing.T) {
		eng, mockRuntime := CreateEngineWithStatefulMock(t)

		var counter int
		mockRuntime.ExecuteFunctionFunc = func(name string, args []interface{}) (interface{}, error) {
			if name == "condition" {
				counter++
				return counter <= 3, nil // Loop 3 times
			}
			mockRuntime.mu.Lock()
			mockRuntime.calls = append(mockRuntime.calls, fmt.Sprintf("%s(%v)", name, args))
			mockRuntime.mu.Unlock()
			return nil, nil
		}

		whileStmt := &ast.WhileStatement{
			Condition: &ast.LanguageCall{Language: "lua", Function: "condition"},
			Body: &ast.BlockStatement{
				Statements: []ast.Statement{
					&ast.LanguageCall{Language: "lua", Function: "before_continue"},
					&ast.ContinueStatement{},
					&ast.LanguageCall{Language: "lua", Function: "after_continue"},
				},
			},
		}

		_, err := eng.executeWhileStatement(whileStmt)
		require.NoError(t, err)

		calls := mockRuntime.GetCalls()
		assert.Len(t, calls, 3, "Should be 3 calls in total")
		assert.Equal(t, "before_continue([])", calls[0])
		assert.Equal(t, "before_continue([])", calls[1])
		assert.Equal(t, "before_continue([])", calls[2])
		assert.NotContains(t, calls, "after_continue([])")
	})
}

func TestForInLoop_ControlFlow(t *testing.T) {
	t.Run("for-in respects break", func(t *testing.T) {
		eng, mockRuntime := CreateEngineWithStatefulMock(t)

		forLoop := &ast.ForInLoopStatement{
			Variable: &ast.Identifier{Name: "item", Language: "lua", Qualified: true},
			Iterable: &ast.ArrayLiteral{
				Elements: []ast.Expression{
					&ast.StringLiteral{Value: "a"},
					&ast.StringLiteral{Value: "b"},
					&ast.StringLiteral{Value: "c"},
				},
			},
			Body: []ast.Statement{
				&ast.LanguageCall{Language: "lua", Function: "body_called"},
				&ast.BreakStatement{},
			},
		}

		_, err := eng.executeForInLoop(forLoop)
		require.NoError(t, err)

		itemVal, err := mockRuntime.GetVariable("item")
		require.NoError(t, err)
		assert.Equal(t, "a", itemVal, "Loop variable should be 'a' when loop breaks")
		assert.Len(t, mockRuntime.GetCalls(), 1, "Body should be called only once")
	})

	t.Run("for-in respects continue", func(t *testing.T) {
		eng, mockRuntime := CreateEngineWithStatefulMock(t)

		forLoop := &ast.ForInLoopStatement{
			Variable: &ast.Identifier{Name: "item", Language: "lua", Qualified: true},
			Iterable: &ast.ArrayLiteral{
				Elements: []ast.Expression{
					&ast.StringLiteral{Value: "a"},
					&ast.StringLiteral{Value: "b"},
					&ast.StringLiteral{Value: "c"},
				},
			},
			Body: []ast.Statement{
				&ast.LanguageCall{Language: "lua", Function: "before_continue"},
				&ast.ContinueStatement{},
				&ast.LanguageCall{Language: "lua", Function: "after_continue"},
			},
		}

		_, err := eng.executeForInLoop(forLoop)
		require.NoError(t, err)

		itemVal, err := mockRuntime.GetVariable("item")
		require.NoError(t, err)
		assert.Equal(t, "c", itemVal, "Loop variable should be the last item at the end")

		calls := mockRuntime.GetCalls()
		assert.Len(t, calls, 3, "before_continue should be called 3 times")
		assert.NotContains(t, calls, "after_continue([])")
	})
}

func TestNumericForLoop_ControlFlow(t *testing.T) {
	t.Run("numeric-for respects break", func(t *testing.T) {
		eng, mockRuntime := CreateEngineWithStatefulMock(t)

		forLoop := &ast.NumericForLoopStatement{
			Variable: &ast.Identifier{Name: "i", Language: "lua", Qualified: true},
			Start:    &ast.NumberLiteral{Value: 1},
			End:      &ast.NumberLiteral{Value: 5},
			Step:     &ast.NumberLiteral{Value: 1},
			Body: []ast.Statement{
				&ast.LanguageCall{Language: "lua", Function: "body_called"},
				// This test requires binary expressions to work: if i == 3 then break end
				&ast.BreakStatement{},
			},
		}

		_, err := eng.executeNumericForLoop(forLoop)
		require.NoError(t, err)

		iVal, err := mockRuntime.GetVariable("i")
		require.NoError(t, err)
		assert.Equal(t, int64(1), iVal, "Loop variable should be 1 when loop breaks")
		assert.Len(t, mockRuntime.GetCalls(), 1, "Body should be called only once")
	})

	t.Run("numeric-for respects continue", func(t *testing.T) {
		eng, mockRuntime := CreateEngineWithStatefulMock(t)

		forLoop := &ast.NumericForLoopStatement{
			Variable: &ast.Identifier{Name: "i", Language: "lua", Qualified: true},
			Start:    &ast.NumberLiteral{Value: 1},
			End:      &ast.NumberLiteral{Value: 5},
			Body: []ast.Statement{
				&ast.LanguageCall{Language: "lua", Function: "before_continue"},
				&ast.ContinueStatement{},
				&ast.LanguageCall{Language: "lua", Function: "after_continue"},
			},
		}

		_, err := eng.executeNumericForLoop(forLoop)
		require.NoError(t, err)

		iVal, err := mockRuntime.GetVariable("i")
		require.NoError(t, err)
		assert.Equal(t, int64(5), iVal, "Loop variable should be 5 at the end")

		calls := mockRuntime.GetCalls()
		assert.Len(t, calls, 5, "before_continue should be called 5 times")
		assert.NotContains(t, calls, "after_continue([])")
	})
}
