package acceptancetests

import (
	"testing"

	"github.com/funvibe/funbit/pkg/funbit"
)

func TestBitstringRestPatterns(t *testing.T) {
	t.Run("Basic binary rest pattern", func(t *testing.T) {
		// Создаем тестовые данные: 8 байт
		packet := funbit.NewBitStringFromBytes([]byte{1, 2, 3, 4, 5, 6, 7, 8})

		var first, second uint
		var rest []byte

		// Пытаемся сопоставить первые 2 байта и захватить остаток как binary
		results, err := funbit.NewMatcher().
			Integer(&first, funbit.WithSize(8)).
			Integer(&second, funbit.WithSize(8)).
			RestBinary(&rest).
			Match(packet)

		if err != nil {
			t.Errorf("Expected success, got error: %v", err)
		}

		if !results[0].Matched || !results[1].Matched || !results[2].Matched {
			t.Errorf("Expected all segments to match")
		}

		if first != 1 || second != 2 {
			t.Errorf("Expected first=1, second=2, got first=%d, second=%d", first, second)
		}

		if len(rest) != 6 {
			t.Errorf("Expected rest length 6, got %d", len(rest))
		}

		expectedRest := []byte{3, 4, 5, 6, 7, 8}
		for i := range expectedRest {
			if rest[i] != expectedRest[i] {
				t.Errorf("Expected rest[%d]=%d, got %d", i, expectedRest[i], rest[i])
			}
		}
	})

	t.Run("Basic bitstring rest pattern", func(t *testing.T) {
		// Создаем тестовые данные с нечетным количеством битов
		packet := funbit.NewBitStringFromBits([]byte{0b10110110, 0b11001100}, 15) // 15 бит

		var firstPart uint
		var rest *funbit.BitString

		// Пытаемся сопоставить первые 3 бита и захватить остаток как bitstring
		results, err := funbit.NewMatcher().
			Integer(&firstPart, funbit.WithSize(3)).
			RestBitstring(&rest).
			Match(packet)

		if err != nil {
			t.Errorf("Expected success, got error: %v", err)
		}

		if !results[0].Matched || !results[1].Matched {
			t.Errorf("Expected all segments to match")
		}

		if firstPart != 0b101 { // Первые 3 бита: 101
			t.Errorf("Expected firstPart=5, got %d", firstPart)
		}

		if rest.Length() != 12 { // Остаток должен быть 12 бит
			t.Errorf("Expected rest length 12, got %d", rest.Length())
		}
	})

	t.Run("Empty binary rest", func(t *testing.T) {
		// Создаем тестовые данные: ровно 2 байта
		packet := funbit.NewBitStringFromBytes([]byte{1, 2})

		var first, second uint
		var rest []byte

		// Пытаемся сопоставить все данные и получить пустой rest
		results, err := funbit.NewMatcher().
			Integer(&first, funbit.WithSize(8)).
			Integer(&second, funbit.WithSize(8)).
			RestBinary(&rest).
			Match(packet)

		if err != nil {
			t.Errorf("Expected success, got error: %v", err)
		}

		if !results[0].Matched || !results[1].Matched || !results[2].Matched {
			t.Errorf("Expected all segments to match")
		}

		if len(rest) != 0 {
			t.Errorf("Expected empty rest, got length %d", len(rest))
		}
	})

	t.Run("Empty bitstring rest", func(t *testing.T) {
		// Создаем тестовые данные: ровно 3 бита
		packet := funbit.NewBitStringFromBits([]byte{0b10100000}, 3)

		var firstPart uint
		var rest *funbit.BitString

		// Пытаемся сопоставить все данные и получить пустой rest
		results, err := funbit.NewMatcher().
			Integer(&firstPart, funbit.WithSize(3)).
			RestBitstring(&rest).
			Match(packet)

		if err != nil {
			t.Errorf("Expected success, got error: %v", err)
		}

		if !results[0].Matched || !results[1].Matched {
			t.Errorf("Expected all segments to match")
		}

		if rest.Length() != 0 {
			t.Errorf("Expected empty rest, got length %d", rest.Length())
		}
	})

	t.Run("Binary rest with unaligned data should fail", func(t *testing.T) {
		// Создаем тестовые данные: 17 бит (не выровнено по байтам)
		packet := funbit.NewBitStringFromBits([]byte{0xFF, 0xFF, 0x80}, 17)

		var first uint
		var rest []byte

		// Пытаемся захватить остаток как binary, но данные не выровнены
		results, err := funbit.NewMatcher().
			Integer(&first, funbit.WithSize(8)).
			RestBinary(&rest).
			Match(packet)

		// Это должно провалиться, потому что остаток не выровнен по байтам
		if err == nil {
			t.Error("Expected error for unaligned binary rest, got success")
		}

		if results != nil && len(results) > 0 && results[1].Matched {
			t.Error("Expected binary rest matching to fail for unaligned data")
		}
	})

	t.Run("Complex pattern with multiple segments and rest", func(t *testing.T) {
		// Создаем сложный паттерн: заголовок + данные + остаток
		packet := funbit.NewBitStringFromBytes([]byte{
			0x01,       // version (1 byte)
			0x05,       // flags (1 byte)
			0x00, 0x0A, // length (2 bytes, big endian)
			'H', 'e', 'l', 'l', 'o', ' ', 'W', 'o', 'r', 'l', // data
			0xFF, 0xFF, 0xFF, // padding/extra data
		})

		var version, flags uint
		var length uint
		var data []byte
		var extra []byte

		results, err := funbit.NewMatcher().
			Integer(&version, funbit.WithSize(8)).
			Integer(&flags, funbit.WithSize(8)).
			Integer(&length, funbit.WithSize(16), funbit.WithEndianness("big")).
			Binary(&data, funbit.WithSize(10)). // "Hello Worl"
			RestBinary(&extra).
			Match(packet)

		if err != nil {
			t.Errorf("Expected success, got error: %v", err)
		}

		if !results[0].Matched || !results[1].Matched || !results[2].Matched || !results[3].Matched || !results[4].Matched {
			t.Errorf("Expected all segments to match")
		}

		if version != 0x01 || flags != 0x05 || length != 0x000A {
			t.Errorf("Expected header mismatch: version=%d, flags=%d, length=%d", version, flags, length)
		}

		expectedData := []byte{'H', 'e', 'l', 'l', 'o', ' ', 'W', 'o', 'r', 'l'}
		if len(data) != len(expectedData) {
			t.Errorf("Expected data length %d, got %d", len(expectedData), len(data))
		} else {
			for i := range expectedData {
				if data[i] != expectedData[i] {
					t.Errorf("Data mismatch at position %d: expected %d, got %d", i, expectedData[i], data[i])
				}
			}
		}

		if len(extra) != 3 {
			t.Errorf("Expected extra data length 3, got %d", len(extra))
		}
	})

	t.Run("Rest pattern with bitstring and binary mixed", func(t *testing.T) {
		// Создаем тестовые данные: 1 байт + 3 бита + остаток
		packet := funbit.NewBitStringFromBits([]byte{
			0xAA,                   // 8 бит: 10101010
			0xE0, 0x80, 0x00, 0x00, // Следующие 19 бит: 111 + 16 бит остатка
		}, 27) // Всего 27 бит

		var firstByte uint
		var threeBits uint
		var rest *funbit.BitString

		results, err := funbit.NewMatcher().
			Integer(&firstByte, funbit.WithSize(8)).
			Integer(&threeBits, funbit.WithSize(3)).
			RestBitstring(&rest).
			Match(packet)

		if err != nil {
			t.Errorf("Expected success, got error: %v", err)
		}

		if !results[0].Matched || !results[1].Matched || !results[2].Matched {
			t.Errorf("Expected all segments to match")
		}

		if firstByte != 0xAA {
			t.Errorf("Expected firstByte=0xAA, got 0x%02X", firstByte)
		}

		if threeBits != 0b111 { // Следующие 3 бита
			t.Errorf("Expected threeBits=7, got %d", threeBits)
		}

		if rest.Length() != 16 { // Остаток должен быть 16 бит
			t.Errorf("Expected rest length 16, got %d", rest.Length())
		}
	})
}
