package builder

import (
	"math"
	"testing"

	"github.com/funvibe/funbit/internal/bitstring"
)

// TestBuilderFloat16Basic тестирует базовую функциональность float16 в builder
func TestBuilderFloat16Basic(t *testing.T) {
	tests := []struct {
		name     string
		value    float32
		expected uint16 // Expected bit pattern
	}{
		{"3.14", 3.14, 0x4248}, // Проверенное каноническое значение
		{"2.5", 2.5, 0x4100},   // Точное значение в float16
		{"1.0", 1.0, 0x3C00},   // Известное точное значение
		{"0.5", 0.5, 0x3800},   // Известное точное значение
		{"2.0", 2.0, 0x4000},   // Известное точное значение
		{"-1.5", -1.5, 0xBE00}, // Отрицательное значение
		{"0.0", 0.0, 0x0000},   // Ноль
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewBuilder()
			builder.AddFloat(tt.value, bitstring.WithSize(16))

			bs, err := builder.Build()
			if err != nil {
				t.Fatalf("Build failed: %v", err)
			}

			if bs.Length() != 16 {
				t.Errorf("Expected 16 bits, got %d", bs.Length())
			}

			// Проверяем биты
			bytes := bs.ToBytes()
			if len(bytes) != 2 {
				t.Errorf("Expected 2 bytes, got %d", len(bytes))
			}

			// Big-endian по умолчанию
			actualBits := uint16(bytes[0])<<8 | uint16(bytes[1])
			if actualBits != tt.expected {
				t.Errorf("Expected bits 0x%04X, got 0x%04X", tt.expected, actualBits)
			}
		})
	}
}

// TestBuilderFloat16Endianness тестирует endianness для float16
func TestBuilderFloat16Endianness(t *testing.T) {
	value := float32(3.14)

	tests := []struct {
		name       string
		endianness string
		byte0      byte
		byte1      byte
	}{
		{"big-endian", "big", 0x42, 0x48},
		{"little-endian", "little", 0x48, 0x42},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewBuilder()
			builder.AddFloat(value, bitstring.WithSize(16), bitstring.WithEndianness(tt.endianness))

			bs, err := builder.Build()
			if err != nil {
				t.Fatalf("Build failed: %v", err)
			}

			bytes := bs.ToBytes()
			if len(bytes) != 2 {
				t.Errorf("Expected 2 bytes, got %d", len(bytes))
			}

			if bytes[0] != tt.byte0 || bytes[1] != tt.byte1 {
				t.Errorf("Expected bytes [0x%02X, 0x%02X], got [0x%02X, 0x%02X]",
					tt.byte0, tt.byte1, bytes[0], bytes[1])
			}
		})
	}
}

// TestBuilderFloat16SpecialValues тестирует специальные значения
func TestBuilderFloat16SpecialValues(t *testing.T) {
	tests := []struct {
		name      string
		value     float32
		checkFunc func(uint16) bool
	}{
		{"positive infinity", float32(math.Inf(1)), func(bits uint16) bool { return bits == 0x7C00 }},
		{"negative infinity", float32(math.Inf(-1)), func(bits uint16) bool { return bits == 0xFC00 }},
		{"NaN", float32(math.NaN()), func(bits uint16) bool {
			exp := (bits >> 10) & 0x1F
			mantissa := bits & 0x3FF
			return exp == 0x1F && mantissa != 0
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewBuilder()
			builder.AddFloat(tt.value, bitstring.WithSize(16))

			bs, err := builder.Build()
			if err != nil {
				t.Fatalf("Build failed: %v", err)
			}

			bytes := bs.ToBytes()
			actualBits := uint16(bytes[0])<<8 | uint16(bytes[1])

			if !tt.checkFunc(actualBits) {
				t.Errorf("Special value test failed for %s: got 0x%04X", tt.name, actualBits)
			}
		})
	}
}

// TestBuilderFloat16RoundTrip тестирует round-trip конверсию с точными значениями
func TestBuilderFloat16RoundTrip(t *testing.T) {
	// Тестируем с известными точными значениями в float16
	exactValues := []float32{0.0, 1.0, 2.0, 0.5, -1.5}

	for _, original := range exactValues {
		t.Run("", func(t *testing.T) {
			// Используем builder для создания bitstring
			builder := NewBuilder()
			builder.AddFloat(original, bitstring.WithSize(16))

			bs, err := builder.Build()
			if err != nil {
				t.Fatalf("Build failed: %v", err)
			}

			// Конвертируем обратно для проверки
			bytes := bs.ToBytes()
			bits := uint16(bytes[0])<<8 | uint16(bytes[1])
			reconstructed := float16BitsToFloat32(bits)

			// Для точных значений должно быть полное совпадение
			if reconstructed != original {
				t.Errorf("Round-trip failed: original=%f, got=%f", original, reconstructed)
			}
		})
	}
}

// Вспомогательная функция для тестирования - конвертирует float16 bits обратно в float32
func float16BitsToFloat32(bits uint16) float32 {
	// Используем ту же логику что и в matcher
	sign16 := (bits >> 15) & 1
	exp16 := (bits >> 10) & 0x1F
	mant16 := bits & 0x3FF

	var sign32, exp32, mant32 uint32
	sign32 = uint32(sign16)

	if exp16 == 0x1F { // Inf or NaN
		exp32 = 0xFF
		if mant16 != 0 {
			mant32 = 1 // NaN
		}
	} else if exp16 == 0 { // zero or denormal
		exp32 = 0
		mant32 = uint32(mant16)
	} else {
		// Convert exponent: float16 bias 15, float32 bias 127
		exp32 = uint32(exp16 - 15 + 127)
		// Convert mantissa: float16 has 10 bits, float32 has 23
		mant32 = uint32(mant16) << 13
	}

	float32Bits := sign32<<31 | exp32<<23 | mant32
	return math.Float32frombits(float32Bits)
}
