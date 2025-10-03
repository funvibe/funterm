package builder

import (
	"testing"

	"github.com/funvibe/funbit/internal/bitstring"
)

// TestBuilder_AddInteger_MissingReflectionPaths tests specific reflection paths that may be missing coverage
func TestBuilder_AddInteger_MissingReflectionPaths(t *testing.T) {
	t.Run("Integer with complex interface type", func(t *testing.T) {
		b := NewBuilder()

		// Use the existing CustomInt interface and MyInt struct defined at package level
		var customInt CustomInt = MyInt{val: 42}
		result := b.AddInteger(customInt)

		if result != b {
			t.Error("Expected AddInteger() to return the same builder instance")
		}

		// Verify segment was added
		if len(b.segments) != 1 {
			t.Error("Expected 1 segment to be added")
		}
		if b.segments[0].Type != bitstring.TypeInteger {
			t.Error("Expected segment type to be integer")
		}

		// Log the actual behavior to understand reflection path
		t.Logf("Custom interface - Size: %d, SizeSpecified: %v, Signed: %v",
			b.segments[0].Size, b.segments[0].SizeSpecified, b.segments[0].Signed)
	})

	t.Run("Integer with pointer to interface", func(t *testing.T) {
		b := NewBuilder()

		// Test with pointer to interface
		type Number interface{}
		var num Number = 42
		result := b.AddInteger(&num)

		if result != b {
			t.Error("Expected AddInteger() to return the same builder instance")
		}

		// Verify segment was added
		if len(b.segments) != 1 {
			t.Error("Expected 1 segment to be added")
		}
		if b.segments[0].Type != bitstring.TypeInteger {
			t.Error("Expected segment type to be integer")
		}

		// Log the actual behavior
		t.Logf("Pointer to interface - Size: %d, SizeSpecified: %v, Signed: %v",
			b.segments[0].Size, b.segments[0].SizeSpecified, b.segments[0].Signed)
	})

	t.Run("Integer with nil interface", func(t *testing.T) {
		b := NewBuilder()

		// Test with nil interface
		var num interface{} = nil
		result := b.AddInteger(num)

		if result != b {
			t.Error("Expected AddInteger() to return the same builder instance")
		}

		// Verify segment was added
		if len(b.segments) != 1 {
			t.Error("Expected 1 segment to be added")
		}
		if b.segments[0].Type != bitstring.TypeInteger {
			t.Error("Expected segment type to be integer")
		}

		// Log the actual behavior for nil interface
		t.Logf("Nil interface - Size: %d, SizeSpecified: %v, Signed: %v",
			b.segments[0].Size, b.segments[0].SizeSpecified, b.segments[0].Signed)
	})
}
