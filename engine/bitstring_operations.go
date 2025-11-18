package engine

import (
	"fmt"
	"strings"

	"funterm/errors"
	"funterm/shared"
	"go-parser/pkg/ast"
)

// BitstringSpecifiers holds parsed bitstring specifiers
type BitstringSpecifiers struct {
	Type       string // "integer", "float", "binary", "utf8", etc.
	Signed     bool   // true for signed, false for unsigned
	Endianness string // "big-endian" or "little-endian"
	Unit       int    // unit multiplier, defaults to 1
}

// parseBitstringSpecifiers parses the specifiers list (legacy function for compatibility)
func (e *ExecutionEngine) parseBitstringSpecifiers(specifiers []string) BitstringSpecifiers {
	// Use the funbit adapter for parsing
	adapter := NewFunbitAdapterWithEngine(e)
	funbitSpecs, err := adapter.parseSpecifiers(specifiers)
	if err != nil {
		// If there's an error, return default specifiers
		return BitstringSpecifiers{
			Type:       "integer",
			Signed:     false,
			Endianness: "big-endian",
			Unit:       1,
		}
	}

	// Convert FunbitBitstringSpecifiers to BitstringSpecifiers for compatibility
	return BitstringSpecifiers{
		Type:       funbitSpecs.Type,
		Signed:     funbitSpecs.Signed,
		Endianness: funbitSpecs.Endianness,
		Unit:       int(funbitSpecs.Unit),
	}
}

// executeBitstringExpression executes a bitstring expression using the funbit adapter
func (e *ExecutionEngine) executeBitstringExpression(expr *ast.BitstringExpression) (interface{}, error) {
	if e.verbose {
		fmt.Printf("DEBUG: executeBitstringExpression - processing %d segments using funbit adapter\n", len(expr.Segments))
	}

	// Create a funbit adapter with engine
	adapter := NewFunbitAdapterWithEngine(e)

	// Use the funbit adapter to execute the bitstring expression
	bitstringObject, err := adapter.ExecuteBitstringExpression(expr)
	if err != nil {
		return nil, errors.NewUserError("BITSTRING_EXECUTION_ERROR", fmt.Sprintf("failed to execute bitstring expression: %v", err))
	}

	if e.verbose {
		fmt.Printf("DEBUG: executeBitstringExpression - funbit adapter returned bitstring with length %d bits\n", bitstringObject.Len())
	}

	return bitstringObject, nil
}

// formatBitstringOutput formats a bitstring object as a bitstring in Erlang style
func (e *ExecutionEngine) formatBitstringOutput(bitstringObj interface{}) string {
	// Check if we have a BitstringObject
	if bitstringObj, ok := bitstringObj.(*shared.BitstringObject); ok {
		if bitstringObj.Len() == 0 {
			return "<<>>"
		}

		// Convert to bytes and format
		bytes := bitstringObj.BitString.ToBytes()
		var result strings.Builder
		result.WriteString("<<")

		for i, b := range bytes {
			if i > 0 {
				result.WriteString(", ")
			}
			result.WriteString(fmt.Sprintf("%d", b))
		}

		result.WriteString(">>")
		return result.String()
	}

	// If we get something else, return empty bitstring
	return "<<>>"
}
