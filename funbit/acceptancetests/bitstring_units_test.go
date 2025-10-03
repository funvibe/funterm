package acceptancetests

import (
	"testing"

	"github.com/funvibe/funbit/pkg/funbit"
)

func TestBitstringUnits(t *testing.T) {
	t.Run("Unit specifier with integer - default unit:1", func(t *testing.T) {
		// <<15:4/integer-unit:1>> - 4 бита * 1 = 4 бита
		packet := funbit.NewBitStringFromBytes([]byte{0xF0}) // 11110000

		var value uint
		results, err := funbit.NewMatcher().
			Integer(&value, funbit.WithSize(4), funbit.WithUnit(1)).
			Match(packet)

		if err != nil {
			t.Errorf("Expected success, got error: %v", err)
		}

		if !results[0].Matched {
			t.Errorf("Expected segment to match")
		}

		if value != 15 { // 1111 = 15
			t.Errorf("Expected value=15, got %d", value)
		}
	})

	t.Run("Unit specifier with integer - unit:8", func(t *testing.T) {
		// <<255:1/integer-unit:8>> - 1 * 8 = 8 бит
		packet := funbit.NewBitStringFromBytes([]byte{0xFF})

		var value uint
		results, err := funbit.NewMatcher().
			Integer(&value, funbit.WithSize(1), funbit.WithUnit(8)).
			Match(packet)

		if err != nil {
			t.Errorf("Expected success, got error: %v", err)
		}

		if !results[0].Matched {
			t.Errorf("Expected segment to match")
		}

		if value != 0xFF {
			t.Errorf("Expected value=255, got %d", value)
		}
	})

	t.Run("Unit specifier with binary - unit:16 alignment", func(t *testing.T) {
		// <<data:2/binary-unit:16>> - 2 * 16 = 32 бита = 4 байта
		packet := funbit.NewBitStringFromBytes([]byte{'A', 'B', 'C', 'D'})

		var extracted []byte
		results, err := funbit.NewMatcher().
			Binary(&extracted, funbit.WithSize(2), funbit.WithUnit(16)).
			Match(packet)

		if err != nil {
			t.Errorf("Expected success, got error: %v", err)
		}

		if !results[0].Matched {
			t.Errorf("Expected segment to match")
		}

		if len(extracted) != 4 {
			t.Errorf("Expected 4 bytes, got %d", len(extracted))
		}

		expected := []byte{'A', 'B', 'C', 'D'}
		for i := range expected {
			if extracted[i] != expected[i] {
				t.Errorf("Expected[%d]=%d, got %d", i, expected[i], extracted[i])
			}
		}
	})

	t.Run("Unit specifier validation - invalid unit:0", func(t *testing.T) {
		// Unit должен быть в диапазоне 1-256
		packet := funbit.NewBitStringFromBytes([]byte{0xFF})

		var value uint
		_, err := funbit.NewMatcher().
			Integer(&value, funbit.WithSize(1), funbit.WithUnit(0)).
			Match(packet)

		if err == nil {
			t.Error("Expected error for unit=0, got success")
		}
	})

	t.Run("Unit specifier validation - invalid unit:257", func(t *testing.T) {
		// Unit должен быть в диапазоне 1-256
		packet := funbit.NewBitStringFromBytes([]byte{0xFF})

		var value uint
		_, err := funbit.NewMatcher().
			Integer(&value, funbit.WithSize(1), funbit.WithUnit(257)).
			Match(packet)

		if err == nil {
			t.Error("Expected error for unit=257, got success")
		}
	})

	t.Run("Unit specifier with default values", func(t *testing.T) {
		// Проверка значений по умолчанию:
		// integer: unit=1, binary: unit=8
		packet := funbit.NewBitStringFromBytes([]byte{0x12, 0x34, 0x56, 0x78})

		var intValue uint
		var binaryValue []byte

		results, err := funbit.NewMatcher().
			Integer(&intValue, funbit.WithSize(8)).   // По умолчанию unit=1
			Binary(&binaryValue, funbit.WithSize(2)). // По умолчанию unit=8
			Match(packet)

		if err != nil {
			t.Errorf("Expected success, got error: %v", err)
		}

		if !results[0].Matched || !results[1].Matched {
			t.Errorf("Expected all segments to match")
		}

		if intValue != 0x12 {
			t.Errorf("Expected intValue=0x12, got 0x%02X", intValue)
		}

		if len(binaryValue) != 2 {
			t.Errorf("Expected binaryValue length 2, got %d", len(binaryValue))
		}

		if binaryValue[0] != 0x34 || binaryValue[1] != 0x56 {
			t.Errorf("Expected binaryValue=[0x34, 0x56], got [0x%02X, 0x%02X]", binaryValue[0], binaryValue[1])
		}
	})

	t.Run("Complex unit specifier pattern", func(t *testing.T) {
		// Сложный паттерн с различными unit
		// <<a:4/unit:1, b:1/unit:8, c:2/unit:16>>
		// a: 4*1 = 4 бита, b: 1*8 = 8 бит, c: 2*16 = 32 бита
		// Всего: 4 + 8 + 32 = 44 бита = 5.5 байт -> нужно 6 байт

		packet := funbit.NewBitStringFromBytes([]byte{0xF1, 0x23, 0x45, 0x67, 0x89, 0xAB})

		var a, b uint
		var c []byte

		results, err := funbit.NewMatcher().
			Integer(&a, funbit.WithSize(4), funbit.WithUnit(1)).
			Integer(&b, funbit.WithSize(1), funbit.WithUnit(8)).
			Binary(&c, funbit.WithSize(2), funbit.WithUnit(16)).
			Match(packet)

		if err != nil {
			t.Errorf("Expected success, got error: %v", err)
		}

		if !results[0].Matched || !results[1].Matched || !results[2].Matched {
			t.Errorf("Expected all segments to match")
		}

		// a: первые 4 бита 0xF1 = 1111 -> 15
		if a != 15 {
			t.Errorf("Expected a=15, got %d", a)
		}

		// b: следующий байт 0x23 = 35
		if b != 0x23 {
			t.Errorf("Expected b=35, got %d", b)
		}

		// c: следующие 4 байта (2 * 16 бит)
		if len(c) != 4 {
			t.Errorf("Expected c length 4, got %d", len(c))
		}

		expected := []byte{0x45, 0x67, 0x89, 0xAB}
		for i := range expected {
			if c[i] != expected[i] {
				t.Errorf("Expected c[%d]=%d, got %d", i, expected[i], c[i])
			}
		}
	})

	t.Run("Unit specifier with insufficient data", func(t *testing.T) {
		// Недостаточно данных для указанного unit
		packet := funbit.NewBitStringFromBytes([]byte{0x12, 0x34}) // Только 2 байта

		var value []byte
		_, err := funbit.NewMatcher().
			Binary(&value, funbit.WithSize(1), funbit.WithUnit(16)). // Нужно 16 бит, есть только 16 бит (2 байта)
			Match(packet)

		if err != nil {
			t.Errorf("Expected success with exact size match, got error: %v", err)
		}
	})
}
