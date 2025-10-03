package acceptancetests

import (
	"testing"

	bitstringpkg "github.com/funvibe/funbit/internal/bitstring"
	"github.com/funvibe/funbit/internal/builder"
	"github.com/funvibe/funbit/internal/matcher"
)

// TestDynamicConstructionInLoop tests dynamic bitstring construction in loops
func TestDynamicConstructionInLoop(t *testing.T) {
	// Test case 1: Build bitstring from array of values in loop
	values := []int{1, 2, 3, 4, 5}

	// This should work like: py.result = <<>>; for i = 0, 4 do py.result = <<py.result/binary, js.values[i]:8>> end
	result, err := builder.BuildBitStringDynamically(func() ([]bitstringpkg.Segment, error) {
		var segments []bitstringpkg.Segment
		for _, value := range values {
			segments = append(segments, bitstringpkg.Segment{
				Value:         value,
				Size:          8,
				SizeSpecified: true,
				Type:          "integer",
			})
		}
		return segments, nil
	})

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result.Length() != 40 { // 5 values * 8 bits each = 40 bits
		t.Errorf("Expected bitstring length 40, got %d", result.Length())
	}

	// Verify the content by matching
	var a, b, c, d, e int
	matchResults, err := matcher.NewMatcher().
		Integer(&a, bitstringpkg.WithSize(8)).
		Integer(&b, bitstringpkg.WithSize(8)).
		Integer(&c, bitstringpkg.WithSize(8)).
		Integer(&d, bitstringpkg.WithSize(8)).
		Integer(&e, bitstringpkg.WithSize(8)).
		Match(result)

	if err != nil {
		t.Fatalf("Expected no error in matching, got %v", err)
	}

	if a != 1 || b != 2 || c != 3 || d != 4 || e != 5 {
		t.Errorf("Expected values 1,2,3,4,5, got %d,%d,%d,%d,%d", a, b, c, d, e)
	}

	if len(matchResults) != 5 {
		t.Errorf("Expected 5 match results, got %d", len(matchResults))
	}
}

// TestConditionalConstruction tests conditional bitstring construction
func TestConditionalConstruction(t *testing.T) {
	// Test case 1: Build with condition - like if statements in construction
	// Should work like: if withHeader { builder.AddInteger(0xMAGIC, WithSize(32)) }

	trueSegments := []bitstringpkg.Segment{
		{Value: 0xDEADBEEF, Size: 32, SizeSpecified: true, Type: "integer"},
		{Value: []byte("payload"), Size: uint(len("payload")), SizeSpecified: true, Type: "binary"},
	}

	falseSegments := []bitstringpkg.Segment{
		{Value: []byte("payload"), Size: uint(len("payload")), SizeSpecified: true, Type: "binary"},
	}

	// Test with true condition
	resultWithHeader, err := builder.BuildConditionalBitString(true, trueSegments, falseSegments)
	if err != nil {
		t.Fatalf("Expected no error with true condition, got %v", err)
	}

	// Test with false condition
	resultWithoutHeader, err := builder.BuildConditionalBitString(false, trueSegments, falseSegments)
	if err != nil {
		t.Fatalf("Expected no error with false condition, got %v", err)
	}

	// Verify lengths
	expectedLengthWithHeader := uint(32 + (8 * len("payload"))) // 32 bits magic + payload bytes
	expectedLengthWithoutHeader := uint(8 * len("payload"))     // just payload bytes

	if resultWithHeader.Length() != expectedLengthWithHeader {
		t.Errorf("Expected length with header %d, got %d", expectedLengthWithHeader, resultWithHeader.Length())
	}

	if resultWithoutHeader.Length() != expectedLengthWithoutHeader {
		t.Errorf("Expected length without header %d, got %d", expectedLengthWithoutHeader, resultWithoutHeader.Length())
	}

	// Verify content by matching
	var magic int
	var payloadWithHeader, payloadWithoutHeader []byte

	// Match result with header
	_, err = matcher.NewMatcher().
		Integer(&magic, bitstringpkg.WithSize(32)).
		Binary(&payloadWithHeader).
		Match(resultWithHeader)

	if err != nil {
		t.Fatalf("Expected no error in matching with header, got %v", err)
	}

	if magic != 0xDEADBEEF {
		t.Errorf("Expected magic 0xDEADBEEF, got %x", magic)
	}

	if string(payloadWithHeader) != "payload" {
		t.Errorf("Expected payload 'payload', got '%s'", string(payloadWithHeader))
	}

	// Match result without header
	_, err = matcher.NewMatcher().
		Binary(&payloadWithoutHeader).
		Match(resultWithoutHeader)

	if err != nil {
		t.Fatalf("Expected no error in matching without header, got %v", err)
	}

	if string(payloadWithoutHeader) != "payload" {
		t.Errorf("Expected payload 'payload', got '%s'", string(payloadWithoutHeader))
	}
}

// TestDynamicFlagConstruction tests dynamic construction with boolean flags
func TestDynamicFlagConstruction(t *testing.T) {
	// Test case: Build bitstring from boolean flags
	// Like: for flag in py.flags: py.flag_bits = <<py.flag_bits/bitstring, (flag ? 1 : 0):1>>
	flags := []bool{true, false, true} // Should result in 101 binary

	result, err := builder.BuildBitStringDynamically(func() ([]bitstringpkg.Segment, error) {
		var segments []bitstringpkg.Segment

		// Add flags dynamically
		for _, flag := range flags {
			value := 0
			if flag {
				value = 1
			}
			segments = append(segments, bitstringpkg.Segment{
				Value:         value,
				Size:          1,
				SizeSpecified: true,
				Type:          "integer",
			})
		}

		// Add 5 bits of padding to make it byte-aligned
		segments = append(segments, bitstringpkg.Segment{
			Value:         0,
			Size:          5,
			SizeSpecified: true,
			Type:          "integer",
		})

		return segments, nil
	})

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result.Length() != 8 { // 3 flag bits + 5 padding bits = 8 bits
		t.Errorf("Expected bitstring length 8, got %d", result.Length())
	}

	// Verify the content by matching
	var f1, f2, f3, padding int
	matchResults, err := matcher.NewMatcher().
		Integer(&f1, bitstringpkg.WithSize(1)).
		Integer(&f2, bitstringpkg.WithSize(1)).
		Integer(&f3, bitstringpkg.WithSize(1)).
		Integer(&padding, bitstringpkg.WithSize(5)).
		Match(result)

	if err != nil {
		t.Fatalf("Expected no error in matching, got %v", err)
	}

	if f1 != 1 || f2 != 0 || f3 != 1 {
		t.Errorf("Expected flags 1,0,1, got %d,%d,%d", f1, f2, f3)
	}

	if padding != 0 {
		t.Errorf("Expected padding 0, got %d", padding)
	}

	if len(matchResults) != 4 {
		t.Errorf("Expected 4 match results, got %d", len(matchResults))
	}
}

// TestAppendToBitString tests appending segments to existing bitstring
func TestAppendToBitString(t *testing.T) {
	// Create initial bitstring
	initial := bitstringpkg.NewBitStringFromBytes([]byte{1, 2})

	// Append additional segments
	segmentsToAppend := []bitstringpkg.Segment{
		{Value: 3, Size: 8, SizeSpecified: true, Type: "integer"},
		{Value: 4, Size: 8, SizeSpecified: true, Type: "integer"},
		{Value: 5, Size: 8, SizeSpecified: true, Type: "integer"},
	}

	updated, err := builder.AppendToBitString(initial, segmentsToAppend...)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if updated.Length() != 40 { // original 16 bits + 24 new bits = 40 bits
		t.Errorf("Expected bitstring length 40, got %d", updated.Length())
	}

	// Verify the content by matching
	var a, b, c, d, e int
	matchResults, err := matcher.NewMatcher().
		Integer(&a, bitstringpkg.WithSize(8)).
		Integer(&b, bitstringpkg.WithSize(8)).
		Integer(&c, bitstringpkg.WithSize(8)).
		Integer(&d, bitstringpkg.WithSize(8)).
		Integer(&e, bitstringpkg.WithSize(8)).
		Match(updated)

	if err != nil {
		t.Fatalf("Expected no error in matching, got %v", err)
	}

	if a != 1 || b != 2 || c != 3 || d != 4 || e != 5 {
		t.Errorf("Expected values 1,2,3,4,5, got %d,%d,%d,%d,%d", a, b, c, d, e)
	}

	if len(matchResults) != 5 {
		t.Errorf("Expected 5 match results, got %d", len(matchResults))
	}
}

// TestComplexDynamicConstruction tests complex dynamic construction scenarios
func TestComplexDynamicConstruction(t *testing.T) {
	// Test very simple scenario: just two segments
	result, err := builder.BuildBitStringDynamically(func() ([]bitstringpkg.Segment, error) {
		return []bitstringpkg.Segment{
			{Value: 42, Size: 8, SizeSpecified: true, Type: "integer"},
			{Value: []byte("OK"), Size: 2, SizeSpecified: true, Type: "binary"},
		}, nil
	})

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Calculate expected length: 8 + 16 = 24 bits
	expectedLen := uint(24)
	if result.Length() != expectedLen {
		t.Errorf("Expected length %d, got %d", expectedLen, result.Length())
	}

	// Verify by matching - very simple approach
	var value int
	var data []byte

	_, err = matcher.NewMatcher().
		Integer(&value, bitstringpkg.WithSize(8)).
		Binary(&data).
		Match(result)

	if err != nil {
		t.Fatalf("Expected no error in matching, got %v", err)
	}

	if value != 42 {
		t.Errorf("Expected value 42, got %d", value)
	}

	if string(data) != "OK" {
		t.Errorf("Expected data 'OK', got '%s'", string(data))
	}
}
