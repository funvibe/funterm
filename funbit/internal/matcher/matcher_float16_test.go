package matcher

import (
	"encoding/binary"
	"math"
	"testing"

	bitstringpkg "github.com/funvibe/funbit/internal/bitstring"
)

// TestMatcherFloat16Basic тестирует базовую функциональность float16 в matcher
func TestMatcherFloat16Basic(t *testing.T) {
	tests := []struct {
		name     string
		bits     uint16  // Входные биты
		expected float64 // Ожидаемое значение
	}{
		{"3.14", 0x4248, 3.140625}, // Каноническое значение для 3.14 в float16
		{"2.5", 0x4100, 2.5},       // Точное значение
		{"1.0", 0x3C00, 1.0},       // Известное точное значение
		{"0.5", 0x3800, 0.5},       // Известное точное значение
		{"2.0", 0x4000, 2.0},       // Известное точное значение
		{"-1.5", 0xBE00, -1.5},     // Отрицательное значение
		{"0.0", 0x0000, 0.0},       // Ноль
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создаем bitstring из bits
			bytes := make([]byte, 2)
			binary.BigEndian.PutUint16(bytes, tt.bits)
			bs := bitstringpkg.NewBitStringFromBytes(bytes)

			// Извлекаем float используя метод matcher
			m := NewMatcher()
			result, err := m.extractFloat(bs, 0, 16, "big")
			if err != nil {
				t.Fatalf("extractFloat failed: %v", err)
			}

			tolerance := 0.000001
			if math.Abs(result-tt.expected) > tolerance {
				t.Errorf("Expected %f, got %f", tt.expected, result)
			}
		})
	}
}

// TestMatcherFloat16Endianness тестирует различные endianness для float16
func TestMatcherFloat16Endianness(t *testing.T) {
	// Используем значение 3.14 -> 0x4248
	expectedFloat := 3.140625

	tests := []struct {
		name       string
		endianness string
		bytes      []byte
	}{
		{"big-endian", "big", []byte{0x42, 0x48}},
		{"little-endian", "little", []byte{0x48, 0x42}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bs := bitstringpkg.NewBitStringFromBytes(tt.bytes)

			m := NewMatcher()
			result, err := m.extractFloat(bs, 0, 16, tt.endianness)
			if err != nil {
				t.Fatalf("extractFloat failed: %v", err)
			}

			tolerance := 0.000001
			if math.Abs(result-expectedFloat) > tolerance {
				t.Errorf("Expected %f, got %f", expectedFloat, result)
			}
		})
	}
}

// TestMatcherFloat16SpecialValues тестирует специальные значения
func TestMatcherFloat16SpecialValues(t *testing.T) {
	tests := []struct {
		name        string
		bits        uint16
		checkFunc   func(float64) bool
		description string
	}{
		{"positive infinity", 0x7C00, func(f float64) bool { return math.IsInf(f, 1) }, "positive infinity"},
		{"negative infinity", 0xFC00, func(f float64) bool { return math.IsInf(f, -1) }, "negative infinity"},
		{"NaN", 0x7E01, math.IsNaN, "NaN"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bytes := make([]byte, 2)
			binary.BigEndian.PutUint16(bytes, tt.bits)
			bs := bitstringpkg.NewBitStringFromBytes(bytes)

			m := NewMatcher()
			result, err := m.extractFloat(bs, 0, 16, "big")
			if err != nil {
				t.Fatalf("extractFloat failed: %v", err)
			}

			if !tt.checkFunc(result) {
				t.Errorf("Expected %s, got %f", tt.description, result)
			}
		})
	}
}

// TestMatcherFloat16RoundTrip тестирует round-trip конверсию
func TestMatcherFloat16RoundTrip(t *testing.T) {
	// Тестируем с известными точными значениями
	exactValues := []float64{0.0, 1.0, 2.0, 0.5, -1.5}

	for _, original := range exactValues {
		t.Run("", func(t *testing.T) {
			// Конвертируем в float16 bits используя логику из builder
			bits := float64ToFloat16Bits(original)

			// Создаем bitstring
			bytes := make([]byte, 2)
			binary.BigEndian.PutUint16(bytes, bits)
			bs := bitstringpkg.NewBitStringFromBytes(bytes)

			// Извлекаем обратно
			m := NewMatcher()
			extracted, err := m.extractFloat(bs, 0, 16, "big")
			if err != nil {
				t.Fatalf("extractFloat failed: %v", err)
			}

			// Для точных значений должно быть полное совпадение
			if extracted != original {
				t.Errorf("Round-trip failed: original=%f, got=%f", original, extracted)
			}
		})
	}
}

// Вспомогательная функция - копия из builder для тестирования round-trip
func float64ToFloat16Bits(f float64) uint16 {
	// Convert to float32 first for precision
	f32 := float32(f)
	bits32 := math.Float32bits(f32)

	// Extract components from float32
	sign := (bits32 >> 31) & 1
	exp32 := (bits32 >> 23) & 0xFF
	mant32 := bits32 & 0x7FFFFF

	var sign16, exp16, mant16 uint16
	sign16 = uint16(sign)

	if exp32 == 0xFF { // Inf or NaN
		exp16 = 0x1F
		if mant32 != 0 {
			mant16 = 1 // quiet NaN
		}
	} else if exp32 == 0 { // zero
		exp16 = 0
		mant16 = 0
	} else {
		// Convert exponent: float32 bias 127, float16 bias 15
		exp32Int := int(exp32) - 127 + 15

		// Check for exponent overflow/underflow
		if exp32Int >= 0x1F { // overflow to inf
			exp16 = 0x1F
			mant16 = 0
		} else if exp32Int <= 0 { // underflow to zero or denormal
			exp16 = 0
			mant16 = 0
		} else {
			exp16 = uint16(exp32Int)
			// Convert mantissa: float32 has 23 bits, float16 has 10
			// Direct conversion without adding implicit bit
			mant16 = uint16(mant32 >> 13)

			// Round to nearest (round half to even)
			roundBit := mant32 & 0x1000
			stickyBits := mant32 & 0x0FFF

			if roundBit != 0 {
				if stickyBits != 0 || (mant16&1) != 0 {
					mant16++
					// Check for mantissa overflow
					if mant16 >= 0x400 {
						mant16 = 0
						exp16++
						if exp16 >= 0x1F {
							exp16 = 0x1F
							mant16 = 0
						}
					}
				}
			}
		}
	}

	return sign16<<15 | exp16<<10 | mant16
}
