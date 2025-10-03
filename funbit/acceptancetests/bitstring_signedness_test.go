package acceptancetests

import (
	"testing"

	"github.com/funvibe/funbit/internal/bitstring"
	"github.com/funvibe/funbit/internal/builder"
	"github.com/funvibe/funbit/internal/matcher"
)

func TestBitstringSignedness(t *testing.T) {
	// Test 1: Basic unsigned integers (default behavior)
	t.Run("Unsigned integers default", func(t *testing.T) {
		bs, err := builder.NewBuilder().
			AddInteger(255).
			AddInteger(127).
			AddInteger(128).
			Build()
		if err != nil {
			t.Fatalf("Failed to build bitstring: %v", err)
		}

		var a, b, c int
		results, err := matcher.NewMatcher().
			Integer(&a).
			Integer(&b).
			Integer(&c).
			Match(bs)
		if err != nil {
			t.Fatalf("Failed to match bitstring: %v", err)
		}

		if a != 255 || b != 127 || c != 128 {
			t.Errorf("Expected 255, 127, 128, got %d, %d, %d", a, b, c)
		}
		if len(results) != 3 {
			t.Errorf("Expected 3 results, got %d", len(results))
		}
	})

	// Test 2: Explicit signed integers
	t.Run("Explicit signed integers", func(t *testing.T) {
		bs, err := builder.NewBuilder().
			AddInteger(-1, bitstring.WithSigned(true)).   // -1 as 8-bit signed
			AddInteger(127, bitstring.WithSigned(true)).  // 127 as 8-bit signed (max positive)
			AddInteger(-128, bitstring.WithSigned(true)). // -128 as 8-bit signed (min negative)
			Build()
		if err != nil {
			t.Fatalf("Failed to build bitstring: %v", err)
		}

		var a, b, c int
		results, err := matcher.NewMatcher().
			Integer(&a, bitstring.WithSigned(true)).
			Integer(&b, bitstring.WithSigned(true)).
			Integer(&c, bitstring.WithSigned(true)).
			Match(bs)
		if err != nil {
			t.Fatalf("Failed to match bitstring: %v", err)
		}

		if a != -1 || b != 127 || c != -128 {
			t.Errorf("Expected -1, 127, -128, got %d, %d, %d", a, b, c)
		}
		if len(results) != 3 {
			t.Errorf("Expected 3 results, got %d", len(results))
		}
	})

	// Test 3: Negative number encoding
	t.Run("Negative signed numbers", func(t *testing.T) {
		bs, err := builder.NewBuilder().
			AddInteger(-1, bitstring.WithSigned(true)).
			AddInteger(-128, bitstring.WithSigned(true)).
			AddInteger(127, bitstring.WithSigned(true)).
			Build()
		if err != nil {
			t.Fatalf("Failed to build bitstring: %v", err)
		}

		var a, b, c int
		results, err := matcher.NewMatcher().
			Integer(&a, bitstring.WithSigned(true)).
			Integer(&b, bitstring.WithSigned(true)).
			Integer(&c, bitstring.WithSigned(true)).
			Match(bs)
		if err != nil {
			t.Fatalf("Failed to match bitstring: %v", err)
		}

		if a != -1 || b != -128 || c != 127 {
			t.Errorf("Expected -1, -128, 127, got %d, %d, %d", a, b, c)
		}
		if len(results) != 3 {
			t.Errorf("Expected 3 results, got %d", len(results))
		}
	})

	// Test 4: Mixed signed/unsigned
	t.Run("Mixed signed and unsigned", func(t *testing.T) {
		bs, err := builder.NewBuilder().
			AddInteger(255).                            // Unsigned
			AddInteger(-1, bitstring.WithSigned(true)). // Signed
			AddInteger(127).                            // Unsigned
			Build()
		if err != nil {
			t.Fatalf("Failed to build bitstring: %v", err)
		}

		var a, b, c int
		results, err := matcher.NewMatcher().
			Integer(&a).                             // Unsigned (default)
			Integer(&b, bitstring.WithSigned(true)). // Signed
			Integer(&c).                             // Unsigned (default)
			Match(bs)
		if err != nil {
			t.Fatalf("Failed to match bitstring: %v", err)
		}

		if a != 255 || b != -1 || c != 127 {
			t.Errorf("Expected 255, -1, 127, got %d, %d, %d", a, b, c)
		}
		if len(results) != 3 {
			t.Errorf("Expected 3 results, got %d", len(results))
		}
	})

	// Test 5: 16-bit signed/unsigned
	t.Run("16-bit signed and unsigned", func(t *testing.T) {
		bs, err := builder.NewBuilder().
			AddInteger(1000, bitstring.WithSize(16)).
			AddInteger(-1000, bitstring.WithSigned(true), bitstring.WithSize(16)).
			AddInteger(40000, bitstring.WithSize(16)). // Should wrap around for 16-bit unsigned
			Build()
		if err != nil {
			t.Fatalf("Failed to build bitstring: %v", err)
		}

		var a, b, c int
		results, err := matcher.NewMatcher().
			Integer(&a, bitstring.WithSize(16)).
			Integer(&b, bitstring.WithSigned(true), bitstring.WithSize(16)).
			Integer(&c, bitstring.WithSize(16)).
			Match(bs)
		if err != nil {
			t.Fatalf("Failed to match bitstring: %v", err)
		}

		// 40000 % 65536 = 40000 - 65536 = -25536, but as unsigned it should be interpreted as positive
		expectedC := 40000 % 65536
		if a != 1000 || b != -1000 || c != expectedC {
			t.Errorf("Expected 1000, -1000, %d, got %d, %d, %d", expectedC, a, b, c)
		}
		if len(results) != 3 {
			t.Errorf("Expected 3 results, got %d", len(results))
		}
	})

	// Test 6: Signed range validation
	t.Run("Signed range validation", func(t *testing.T) {
		// Test maximum positive 8-bit signed value
		bs, err := builder.NewBuilder().
			AddInteger(127, bitstring.WithSigned(true), bitstring.WithSize(8)).
			Build()
		if err != nil {
			t.Fatalf("Failed to build bitstring: %v", err)
		}

		var val int
		_, err = matcher.NewMatcher().
			Integer(&val, bitstring.WithSigned(true), bitstring.WithSize(8)).
			Match(bs)
		if err != nil {
			t.Fatalf("Failed to match bitstring: %v", err)
		}
		if val != 127 {
			t.Errorf("Expected 127, got %d", val)
		}

		// Test minimum negative 8-bit signed value
		bs2, err := builder.NewBuilder().
			AddInteger(-128, bitstring.WithSigned(true), bitstring.WithSize(8)).
			Build()
		if err != nil {
			t.Fatalf("Failed to build bitstring: %v", err)
		}

		_, err = matcher.NewMatcher().
			Integer(&val, bitstring.WithSigned(true), bitstring.WithSize(8)).
			Match(bs2)
		if err != nil {
			t.Fatalf("Failed to match bitstring: %v", err)
		}
		if val != -128 {
			t.Errorf("Expected -128, got %d", val)
		}
	})

	// Test 7: 32-bit signed integers
	t.Run("32-bit signed integers", func(t *testing.T) {
		bs, err := builder.NewBuilder().
			AddInteger(-123456, bitstring.WithSigned(true), bitstring.WithSize(32)).
			AddInteger(123456, bitstring.WithSigned(true), bitstring.WithSize(32)).
			AddInteger(-2147483648, bitstring.WithSigned(true), bitstring.WithSize(32)). // Min 32-bit signed
			Build()
		if err != nil {
			t.Fatalf("Failed to build bitstring: %v", err)
		}

		var a, b, c int
		results, err := matcher.NewMatcher().
			Integer(&a, bitstring.WithSigned(true), bitstring.WithSize(32)).
			Integer(&b, bitstring.WithSigned(true), bitstring.WithSize(32)).
			Integer(&c, bitstring.WithSigned(true), bitstring.WithSize(32)).
			Match(bs)
		if err != nil {
			t.Fatalf("Failed to match bitstring: %v", err)
		}

		if a != -123456 || b != 123456 || c != -2147483648 {
			t.Errorf("Expected -123456, 123456, -2147483648, got %d, %d, %d", a, b, c)
		}
		if len(results) != 3 {
			t.Errorf("Expected 3 results, got %d", len(results))
		}
	})

	// Test 8: 64-bit signed integers
	t.Run("64-bit signed integers", func(t *testing.T) {
		bs, err := builder.NewBuilder().
			AddInteger(-9223372036854775808, bitstring.WithSigned(true), bitstring.WithSize(64)). // Min 64-bit signed
			AddInteger(9223372036854775807, bitstring.WithSigned(true), bitstring.WithSize(64)).  // Max 64-bit signed
			Build()
		if err != nil {
			t.Fatalf("Failed to build bitstring: %v", err)
		}

		var a, b int64
		results, err := matcher.NewMatcher().
			Integer(&a, bitstring.WithSigned(true), bitstring.WithSize(64)).
			Integer(&b, bitstring.WithSigned(true), bitstring.WithSize(64)).
			Match(bs)
		if err != nil {
			t.Fatalf("Failed to match bitstring: %v", err)
		}

		if a != -9223372036854775808 || b != 9223372036854775807 {
			t.Errorf("Expected -9223372036854775808, 9223372036854775807, got %d, %d", a, b)
		}
		if len(results) != 2 {
			t.Errorf("Expected 2 results, got %d", len(results))
		}
	})
}
