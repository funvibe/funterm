package builder

import (
	"testing"

	"github.com/funvibe/funbit/internal/bitstring"
)

// TestBuilder_NewBuilder tests the NewBuilder function
func TestBuilder_NewBuilder(t *testing.T) {
	b := NewBuilder()

	if b == nil {
		t.Error("Expected NewBuilder() to return a non-nil builder")
	}

	if len(b.segments) != 0 {
		t.Error("Expected new builder to have empty segments")
	}
}

// TestBuilder_AddInteger tests the AddInteger method
func TestBuilder_AddInteger(t *testing.T) {
	b := NewBuilder()

	result := b.AddInteger(42)

	if result != b {
		t.Error("Expected AddInteger() to return the same builder instance")
	}

	if len(b.segments) != 1 {
		t.Error("Expected 1 segment to be added")
	}

	if b.segments[0].Type != bitstring.TypeInteger {
		t.Error("Expected segment type to be integer")
	}
}

// TestBuilder_AddInteger_AdditionalCoverage tests additional scenarios for AddInteger
func TestBuilder_AddInteger_AdditionalCoverage(t *testing.T) {
	t.Run("AddInteger with negative value", func(t *testing.T) {
		b := NewBuilder()
		result := b.AddInteger(-42)

		if result != b {
			t.Error("Expected AddInteger() to return the same builder instance")
		}

		if len(b.segments) != 1 {
			t.Error("Expected 1 segment to be added")
		}

		if b.segments[0].Type != bitstring.TypeInteger {
			t.Error("Expected segment type to be integer")
		}
	})

	t.Run("AddInteger with options", func(t *testing.T) {
		b := NewBuilder()
		result := b.AddInteger(42, bitstring.WithSize(16), bitstring.WithSigned(true))

		if result != b {
			t.Error("Expected AddInteger() to return the same builder instance")
		}

		if len(b.segments) != 1 {
			t.Error("Expected 1 segment to be added")
		}

		if b.segments[0].Size != 16 {
			t.Error("Expected segment size to be 16")
		}

		if !b.segments[0].Signed {
			t.Error("Expected segment to be signed")
		}
	})
}

// TestBuilder_AddFloat tests the AddFloat method
func TestBuilder_AddFloat(t *testing.T) {
	b := NewBuilder()

	result := b.AddFloat(3.14)

	if result != b {
		t.Error("Expected AddFloat() to return the same builder instance")
	}

	if len(b.segments) != 1 {
		t.Error("Expected 1 segment to be added")
	}

	if b.segments[0].Type != bitstring.TypeFloat {
		t.Error("Expected segment type to be float")
	}
}

// TestBuilder_AddBinary tests the AddBinary method
func TestBuilder_AddBinary(t *testing.T) {
	b := NewBuilder()
	data := []byte{0x01, 0x02, 0x03}

	result := b.AddBinary(data)

	if result != b {
		t.Error("Expected AddBinary() to return the same builder instance")
	}

	if len(b.segments) != 1 {
		t.Error("Expected 1 segment to be added")
	}

	if b.segments[0].Type != bitstring.TypeBinary {
		t.Error("Expected segment type to be binary")
	}
}

// TestBuilder_AddSegment tests the AddSegment method
func TestBuilder_AddSegment(t *testing.T) {
	b := NewBuilder()
	segment := bitstring.NewSegment(42, bitstring.WithSize(8))

	result := b.AddSegment(*segment)

	if result != b {
		t.Error("Expected AddSegment() to return the same builder instance")
	}

	if len(b.segments) != 1 {
		t.Error("Expected 1 segment to be added")
	}

	if b.segments[0].Value != segment.Value {
		t.Error("Expected segment to be the same as added")
	}
}

// TestBuilder_AddBitstring tests the AddBitstring method
func TestBuilder_AddBitstring(t *testing.T) {
	b := NewBuilder()
	bs := bitstring.NewBitStringFromBits([]byte{0xFF}, 8)

	result := b.AddBitstring(bs)

	if result != b {
		t.Error("Expected AddBitstring() to return the same builder instance")
	}

	if len(b.segments) != 1 {
		t.Error("Expected 1 segment to be added")
	}

	if b.segments[0].Type != bitstring.TypeBitstring {
		t.Error("Expected segment type to be bitstring")
	}
}

// TestBuilder_Build tests the Build method
func TestBuilder_Build(t *testing.T) {
	b := NewBuilder()
	b.AddInteger(42, bitstring.WithSize(8))

	result, err := b.Build()

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if result == nil {
		t.Error("Expected non-nil bitstring")
	}

	if result.Length() != 8 {
		t.Errorf("Expected bitstring length 8, got %d", result.Length())
	}
}

// TestBuilder_BuildContent tests the Build method and verifies content
func TestBuilder_BuildContent(t *testing.T) {
	b := NewBuilder()
	b.AddInteger(42, bitstring.WithSize(8))

	result, err := b.Build()

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if result == nil {
		t.Error("Expected non-nil bitstring")
	}

	if result.Length() != 8 {
		t.Errorf("Expected 8 bits, got %d", result.Length())
	}
}

// TestBuilder_EmptyBuild tests building with no segments
func TestBuilder_EmptyBuild(t *testing.T) {
	b := NewBuilder()

	result, err := b.Build()

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if result == nil {
		t.Error("Expected non-nil bitstring")
	}

	if result.Length() != 0 {
		t.Errorf("Expected empty bitstring, got %d bits", result.Length())
	}
}
