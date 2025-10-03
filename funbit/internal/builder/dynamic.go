package builder

import (
	"github.com/funvibe/funbit/internal/bitstring"
)

// BuildBitStringDynamically builds a bitstring using a generator function
// that returns segments dynamically. This allows for construction in loops
// and other dynamic scenarios.
func BuildBitStringDynamically(generator func() ([]bitstring.Segment, error)) (*bitstring.BitString, error) {
	if generator == nil {
		return nil, &bitstring.BitStringError{
			Code:    "INVALID_GENERATOR",
			Message: "generator function cannot be nil",
		}
	}

	// Get segments from generator
	segments, err := generator()
	if err != nil {
		return nil, err
	}

	if len(segments) == 0 {
		return bitstring.NewBitString(), nil
	}

	// Create a new builder and add all segments
	b := NewBuilder()
	for _, segment := range segments {
		b.AddSegment(segment)
	}

	// Build the final bitstring
	return b.Build()
}

// BuildConditionalBitString builds a bitstring based on a condition.
// If condition is true, uses trueSegments, otherwise uses falseSegments.
func BuildConditionalBitString(condition bool, trueSegments, falseSegments []bitstring.Segment) (*bitstring.BitString, error) {
	var segments []bitstring.Segment

	if condition {
		segments = trueSegments
	} else {
		segments = falseSegments
	}

	if len(segments) == 0 {
		return bitstring.NewBitString(), nil
	}

	// Create a new builder and add all segments
	b := NewBuilder()
	for _, segment := range segments {
		b.AddSegment(segment)
	}

	// Build the final bitstring
	return b.Build()
}

// AppendToBitString appends segments to an existing bitstring.
// Since BitString is immutable, this returns a new BitString with the appended data.
func AppendToBitString(target *bitstring.BitString, segments ...bitstring.Segment) (*bitstring.BitString, error) {
	if target == nil {
		return nil, &bitstring.BitStringError{
			Code:    "INVALID_TARGET",
			Message: "target bitstring cannot be nil",
		}
	}

	if len(segments) == 0 {
		return target.Clone(), nil // Return clone if nothing to append
	}

	// Create a temporary builder with the new segments
	tempBuilder := NewBuilder()
	for _, segment := range segments {
		tempBuilder.AddSegment(segment)
	}

	// Build the new segments
	newBitString, err := tempBuilder.Build()
	if err != nil {
		return nil, err
	}

	// Combine the existing bitstring with the new one
	combinedData := append(target.ToBytes(), newBitString.ToBytes()...)
	combinedBitString := bitstring.NewBitStringFromBytes(combinedData)

	return combinedBitString, nil
}
