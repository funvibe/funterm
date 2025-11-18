package funbit

import (
	"fmt"
	"math"
	"math/big"
	"testing"
	"unicode/utf16"
)

// Helper function for byte slice comparison
func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Helper function for string contains check
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (len(substr) == 0 || findSubstring(s, substr) >= 0)
}

func findSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// TestBasicConstruction проверяет базовое создание битовых строк
func TestBasicConstruction(t *testing.T) {
	tests := []struct {
		name     string
		build    func(*Builder)
		expected []byte
		bits     uint
	}{
		{
			name: "empty bitstring",
			build: func(b *Builder) {
				// Пустая битовая строка
			},
			expected: []byte{},
			bits:     0,
		},
		{
			name: "single byte",
			build: func(b *Builder) {
				AddInteger(b, 42, WithSize(8))
			},
			expected: []byte{42},
			bits:     8,
		},
		{
			name: "multiple bytes",
			build: func(b *Builder) {
				AddInteger(b, 1, WithSize(8))
				AddInteger(b, 2, WithSize(8))
				AddInteger(b, 3, WithSize(8))
			},
			expected: []byte{1, 2, 3},
			bits:     24,
		},
		{
			name: "non-byte aligned",
			build: func(b *Builder) {
				AddInteger(b, 1, WithSize(1))
				AddInteger(b, 0, WithSize(1))
				AddInteger(b, 1, WithSize(1))
			},
			expected: nil, // Проверяем только количество бит
			bits:     3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewBuilder()
			tt.build(builder)

			bitstring, err := Build(builder)
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}

			if bitstring.Length() != tt.bits {
				t.Errorf("Expected length %d, got %d", tt.bits, bitstring.Length())
			}

			if tt.expected != nil && !bytesEqual(tt.expected, bitstring.ToBytes()) {
				t.Errorf("Expected %v, got %v", tt.expected, bitstring.ToBytes())
			}
		})
	}
}

// TestPatternMatching проверяет сопоставление с образцом
func TestPatternMatching(t *testing.T) {
	tests := []struct {
		name        string
		bitstring   []byte
		setupMatch  func(*Matcher) (interface{}, interface{}, interface{})
		validate    func(t *testing.T, v1, v2, v3 interface{})
		shouldError bool
	}{
		{
			name:      "simple integer matching",
			bitstring: []byte{42, 17, 255},
			setupMatch: func(m *Matcher) (interface{}, interface{}, interface{}) {
				var a, b, c int
				Integer(m, &a, WithSize(8))
				Integer(m, &b, WithSize(8))
				Integer(m, &c, WithSize(8))
				return &a, &b, &c
			},
			validate: func(t *testing.T, v1, v2, v3 interface{}) {
				if *v1.(*int) != 42 {
					t.Errorf("Expected 42, got %d", *v1.(*int))
				}
				if *v2.(*int) != 17 {
					t.Errorf("Expected 17, got %d", *v2.(*int))
				}
				if *v3.(*int) != 255 {
					t.Errorf("Expected 255, got %d", *v3.(*int))
				}
			},
		},
		{
			name:      "mixed types matching",
			bitstring: append([]byte{0, 100}, []byte("test")...),
			setupMatch: func(m *Matcher) (interface{}, interface{}, interface{}) {
				var num int
				var text []byte
				Integer(m, &num, WithSize(16))
				Binary(m, &text)
				return &num, &text, nil
			},
			validate: func(t *testing.T, v1, v2, v3 interface{}) {
				if *v1.(*int) != 100 {
					t.Errorf("Expected 100, got %d", *v1.(*int))
				}
				if string(*v2.(*[]byte)) != "test" {
					t.Errorf("Expected 'test', got '%s'", string(*v2.(*[]byte)))
				}
			},
		},
		{
			name:      "pattern mismatch - too short",
			bitstring: []byte{1, 2},
			setupMatch: func(m *Matcher) (interface{}, interface{}, interface{}) {
				var a, b, c int
				Integer(m, &a, WithSize(8))
				Integer(m, &b, WithSize(8))
				Integer(m, &c, WithSize(8))
				return &a, &b, &c
			},
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bs := NewBitStringFromBytes(tt.bitstring)
			matcher := NewMatcher()

			v1, v2, v3 := tt.setupMatch(matcher)

			results, err := Match(matcher, bs)

			if tt.shouldError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Fatalf("Expected no error, got %v", err)
				}
				if len(results) == 0 {
					t.Fatalf("Expected non-empty results")
				}
				if tt.validate != nil {
					tt.validate(t, v1, v2, v3)
				}
			}
		})
	}
}

// TestEndianness проверяет работу с порядком байтов
func TestEndianness(t *testing.T) {
	value := 0x1234ABCD

	tests := []struct {
		name       string
		endianness string
		expected   []byte
	}{
		{
			name:       "big-endian",
			endianness: "big",
			expected:   []byte{0x12, 0x34, 0xAB, 0xCD},
		},
		{
			name:       "little-endian",
			endianness: "little",
			expected:   []byte{0xCD, 0xAB, 0x34, 0x12},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewBuilder()
			AddInteger(builder, value,
				WithSize(32),
				WithEndianness(tt.endianness))

			bitstring, err := Build(builder)
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}

			if !bytesEqual(tt.expected, bitstring.ToBytes()) {
				t.Errorf("Expected %v, got %v", tt.expected, bitstring.ToBytes())
			}

			// Проверяем обратное преобразование
			matcher := NewMatcher()
			var extracted int
			Integer(matcher, &extracted,
				WithSize(32),
				WithEndianness(tt.endianness))

			results, err := Match(matcher, bitstring)
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}
			_ = results // Suppress unused variable warning
			if extracted != value {
				t.Errorf("Expected %v, got %v", value, extracted)
			}
		})
	}
}

// TestSignedness проверяет знаковые и беззнаковые числа
func TestSignedness(t *testing.T) {
	tests := []struct {
		name     string
		value    int
		size     uint
		signed   bool
		expected int
	}{
		{
			name:     "unsigned 8-bit max",
			value:    255,
			size:     8,
			signed:   false,
			expected: 255,
		},
		{
			name:     "signed 8-bit negative",
			value:    -1,
			size:     8,
			signed:   true,
			expected: -1,
		},
		{
			name:     "signed 8-bit overflow",
			value:    255,
			size:     8,
			signed:   true,
			expected: -1, // 255 как signed 8-bit = -1
		},
		{
			name:     "signed 16-bit negative",
			value:    -32768,
			size:     16,
			signed:   true,
			expected: -32768,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewBuilder()
			AddInteger(builder, tt.value,
				WithSize(tt.size),
				WithSigned(tt.signed))

			bitstring, err := Build(builder)
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}

			matcher := NewMatcher()
			var extracted int
			Integer(matcher, &extracted,
				WithSize(tt.size),
				WithSigned(tt.signed))

			results, err := Match(matcher, bitstring)
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}
			_ = results // Suppress unused variable warning
			if extracted != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, extracted)
			}
		})
	}
}

// TestFloatHandling проверяет работу с числами с плавающей точкой
func TestFloatHandling(t *testing.T) {
	tests := []struct {
		name      string
		value     float64
		size      uint
		unit      uint
		tolerance float64
	}{
		{
			name:      "32-bit float",
			value:     math.Pi,
			size:      32,
			unit:      1,
			tolerance: 0.0001,
		},
		{
			name:      "64-bit float",
			value:     math.E,
			size:      64,
			unit:      1,
			tolerance: 0.0000000001,
		},
		{
			name:      "64-bit via unit multiplier",
			value:     math.Phi,
			size:      32,
			unit:      2,
			tolerance: 0.0000000001,
		},
		{
			name:      "special values - infinity",
			value:     math.Inf(1),
			size:      32,
			unit:      1,
			tolerance: 0,
		},
		{
			name:      "special values - NaN",
			value:     math.NaN(),
			size:      32,
			unit:      1,
			tolerance: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewBuilder()
			AddFloat(builder, tt.value,
				WithSize(tt.size),
				WithUnit(tt.unit))

			bitstring, err := Build(builder)
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}

			matcher := NewMatcher()
			var extracted float64
			Float(matcher, &extracted,
				WithSize(tt.size),
				WithUnit(tt.unit))

			_, err = Match(matcher, bitstring)
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}

			if math.IsNaN(tt.value) {
				if !math.IsNaN(extracted) {
					t.Errorf("Expected NaN, got %v", extracted)
				}
			} else if math.IsInf(tt.value, 0) {
				if !math.IsInf(extracted, int(tt.value)) {
					t.Errorf("Expected Inf, got %v", extracted)
				}
			} else {
				if math.Abs(tt.value-extracted) > tt.tolerance {
					t.Errorf("Expected %v±%v, got %v", tt.value, tt.tolerance, extracted)
				}
			}
		})
	}
}

// TestUTFEncoding проверяет UTF кодирование
func TestUTFEncoding(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		encoding string
	}{
		{
			name:     "UTF-8 ASCII",
			text:     "Hello",
			encoding: "utf8",
		},
		{
			name:     "UTF-8 Unicode",
			text:     "Привет",
			encoding: "utf8",
		},
		{
			name:     "UTF-8 Emoji",
			text:     "🚀",
			encoding: "utf8",
		},
		{
			name:     "UTF-16 ASCII",
			text:     "Test",
			encoding: "utf16",
		},
		{
			name:     "UTF-32 Single char",
			text:     "A",
			encoding: "utf32",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewBuilder()

			switch tt.encoding {
			case "utf8":
				AddUTF8(builder, tt.text)
			case "utf16":
				AddUTF16(builder, tt.text, WithEndianness("big"))
			case "utf32":
				AddUTF32(builder, tt.text)
			}

			bitstring, err := Build(builder)
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}

			matcher := NewMatcher()
			var extractedBytes []byte

			// UTF segments in Erlang extract individual code points, not entire strings
			// To extract the entire encoded string, we use binary extraction
			RestBinary(matcher, &extractedBytes)

			_, err = Match(matcher, bitstring)
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}

			// Decode the extracted bytes back to string based on encoding
			var extracted string
			switch tt.encoding {
			case "utf8":
				extracted = string(extractedBytes)
			case "utf16":
				// Convert UTF-16 bytes back to string
				if len(extractedBytes)%2 != 0 {
					t.Fatalf("Invalid UTF-16 data length: %d", len(extractedBytes))
				}
				runes := make([]uint16, len(extractedBytes)/2)
				for i := 0; i < len(runes); i++ {
					// Big endian
					runes[i] = uint16(extractedBytes[i*2])<<8 | uint16(extractedBytes[i*2+1])
				}
				extracted = string(utf16.Decode(runes))
			case "utf32":
				// Convert UTF-32 bytes back to string
				if len(extractedBytes)%4 != 0 {
					t.Fatalf("Invalid UTF-32 data length: %d", len(extractedBytes))
				}
				runes := make([]rune, len(extractedBytes)/4)
				for i := 0; i < len(runes); i++ {
					// Big endian
					runes[i] = rune(extractedBytes[i*4])<<24 | rune(extractedBytes[i*4+1])<<16 |
						rune(extractedBytes[i*4+2])<<8 | rune(extractedBytes[i*4+3])
				}
				extracted = string(runes)
			}

			if extracted != tt.text {
				t.Errorf("Expected %v, got %v", tt.text, extracted)
			}
		})
	}
}

// TestDynamicSizing проверяет динамические размеры
func TestDynamicSizing(t *testing.T) {
	// Создаем простой тест сначала
	t.Run("Simple test", func(t *testing.T) {
		// Создаем битстроку с размером 2 и данными "Hi"
		builder := NewBuilder()
		dataSize := 2
		data := "Hi"

		AddInteger(builder, dataSize, WithSize(8))
		AddBinary(builder, []byte(data))

		bitstring, err := Build(builder)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Матчинг
		matcher := NewMatcher()
		var size int
		var payload []byte

		Integer(matcher, &size, WithSize(8))
		RegisterVariable(matcher, "size", &size)
		Binary(matcher, &payload, WithDynamicSizeExpression("size*8"), WithUnit(1))

		_, err = Match(matcher, bitstring)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if size != dataSize {
			t.Errorf("Expected %v, got %v", dataSize, size)
		}
		if string(payload) != data {
			t.Errorf("Expected %v, got %v", data, string(payload))
		}
	})

	// Оригинальный тест
	t.Run("Original test", func(t *testing.T) {
		// Создаем пакет с размером и данными
		builder := NewBuilder()
		dataSize := 5
		data := "Hello"

		AddInteger(builder, dataSize, WithSize(8))
		AddBinary(builder, []byte(data))
		AddBinary(builder, []byte(" World"))

		bitstring, err := Build(builder)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Матчинг с динамическим размером
		matcher := NewMatcher()
		var size int
		var payload []byte
		var rest []byte

		// Сначала читаем размер
		Integer(matcher, &size, WithSize(8))

		// Регистрируем переменную для использования в выражениях
		RegisterVariable(matcher, "size", &size)

		// Используем размер для чтения данных
		Binary(matcher, &payload, WithDynamicSizeExpression("size*8"), WithUnit(1))
		RestBinary(matcher, &rest)

		_, err = Match(matcher, bitstring)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if size != dataSize {
			t.Errorf("Expected %v, got %v", dataSize, size)
		}
		if string(payload) != data {
			t.Errorf("Expected %v, got %v", data, string(payload))
		}
		if string(rest) != " World" {
			t.Errorf("Expected %v, got %v", " World", string(rest))
		}
	})
}

// TestComplexProtocols проверяет парсинг сложных протоколов
func TestComplexProtocols(t *testing.T) {
	t.Run("IPv4 Header", func(t *testing.T) {
		// Создаем IPv4 заголовок
		builder := NewBuilder()

		// Version (4) and Header Length (5)
		AddInteger(builder, 4, WithSize(4))
		AddInteger(builder, 5, WithSize(4))

		// Type of Service
		AddInteger(builder, 0, WithSize(8))

		// Total Length
		AddInteger(builder, 20, WithSize(16))

		// Identification
		AddInteger(builder, 12345, WithSize(16))

		// Flags (3 bits) and Fragment Offset (13 bits)
		AddInteger(builder, 2, WithSize(3))
		AddInteger(builder, 0, WithSize(13))

		// TTL
		AddInteger(builder, 64, WithSize(8))

		// Protocol (TCP = 6)
		AddInteger(builder, 6, WithSize(8))

		// Header Checksum
		AddInteger(builder, 0, WithSize(16))

		// Source IP (192.168.0.1)
		AddInteger(builder, 0xC0A80001, WithSize(32))

		// Destination IP (8.8.8.8)
		AddInteger(builder, 0x08080808, WithSize(32))

		bitstring, err := Build(builder)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Парсинг заголовка
		matcher := NewMatcher()
		var version, headerLen, tos, totalLen, id int
		var flags, fragOffset, ttl, protocol, checksum int
		var srcIP, dstIP uint32

		Integer(matcher, &version, WithSize(4))
		Integer(matcher, &headerLen, WithSize(4))
		Integer(matcher, &tos, WithSize(8))
		Integer(matcher, &totalLen, WithSize(16))
		Integer(matcher, &id, WithSize(16))
		Integer(matcher, &flags, WithSize(3))
		Integer(matcher, &fragOffset, WithSize(13))
		Integer(matcher, &ttl, WithSize(8))
		Integer(matcher, &protocol, WithSize(8))
		Integer(matcher, &checksum, WithSize(16))
		Integer(matcher, &srcIP, WithSize(32))
		Integer(matcher, &dstIP, WithSize(32))

		_, err = Match(matcher, bitstring)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if version != 4 {
			t.Errorf("Expected version 4, got %d", version)
		}
		if headerLen != 5 {
			t.Errorf("Expected header length 5, got %d", headerLen)
		}
		if totalLen != 20 {
			t.Errorf("Expected total length 20, got %d", totalLen)
		}
		if protocol != 6 {
			t.Errorf("Expected protocol 6, got %d", protocol)
		}
		if srcIP != uint32(0xC0A80001) {
			t.Errorf("Expected source IP 0xC0A80001, got 0x%X", srcIP)
		}
		if dstIP != uint32(0x08080808) {
			t.Errorf("Expected destination IP 0x08080808, got 0x%X", dstIP)
		}
	})

	t.Run("PNG Chunk", func(t *testing.T) {
		// Создаем PNG IHDR chunk
		builder := NewBuilder()

		// Length
		AddInteger(builder, 13, WithSize(32), WithEndianness("big"))

		// Type "IHDR"
		AddBinary(builder, []byte("IHDR"))

		// Width
		AddInteger(builder, 100, WithSize(32), WithEndianness("big"))

		// Height
		AddInteger(builder, 50, WithSize(32), WithEndianness("big"))

		// Bit depth
		AddInteger(builder, 8, WithSize(8))

		// Color type (2 = RGB)
		AddInteger(builder, 2, WithSize(8))

		// Compression
		AddInteger(builder, 0, WithSize(8))

		// Filter
		AddInteger(builder, 0, WithSize(8))

		// Interlace
		AddInteger(builder, 0, WithSize(8))

		// CRC
		AddInteger(builder, 0x12345678, WithSize(32), WithEndianness("big"))

		bitstring, err := Build(builder)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Парсинг chunk
		matcher := NewMatcher()
		var length, width, height int
		var chunkType []byte
		var bitDepth, colorType, compression, filter, interlace int
		var crc uint32

		Integer(matcher, &length, WithSize(32), WithEndianness("big"))
		Binary(matcher, &chunkType, WithSize(4))
		Integer(matcher, &width, WithSize(32), WithEndianness("big"))
		Integer(matcher, &height, WithSize(32), WithEndianness("big"))
		Integer(matcher, &bitDepth, WithSize(8))
		Integer(matcher, &colorType, WithSize(8))
		Integer(matcher, &compression, WithSize(8))
		Integer(matcher, &filter, WithSize(8))
		Integer(matcher, &interlace, WithSize(8))
		Integer(matcher, &crc, WithSize(32), WithEndianness("big"))

		_, err = Match(matcher, bitstring)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if length != 13 {
			t.Errorf("Expected length 13, got %d", length)
		}
		if string(chunkType) != "IHDR" {
			t.Errorf("Expected chunk type 'IHDR', got '%s'", string(chunkType))
		}
		if width != 100 {
			t.Errorf("Expected width 100, got %d", width)
		}
		if height != 50 {
			t.Errorf("Expected height 50, got %d", height)
		}
		if bitDepth != 8 {
			t.Errorf("Expected bit depth 8, got %d", bitDepth)
		}
		if colorType != 2 {
			t.Errorf("Expected color type 2, got %d", colorType)
		}
	})
}

// TestEdgeCases проверяет граничные случаи
func TestEdgeCases(t *testing.T) {
	t.Run("Zero-sized segments", func(t *testing.T) {
		builder := NewBuilder()
		AddInteger(builder, 0, WithSize(0))
		AddBinary(builder, []byte{})

		bitstring, err := Build(builder)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if bitstring.Length() != uint(0) {
			t.Errorf("Expected length 0, got %d", bitstring.Length())
		}
	})

	t.Run("Maximum size values", func(t *testing.T) {
		builder := NewBuilder()
		AddInteger(builder, math.MaxInt64, WithSize(64))

		bitstring, err := Build(builder)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		matcher := NewMatcher()
		var value int64
		Integer(matcher, &value, WithSize(64))

		_, err = Match(matcher, bitstring)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if value != int64(math.MaxInt64) {
			t.Errorf("Expected %d, got %d", int64(math.MaxInt64), value)
		}
	})

	t.Run("Overflow handling", func(t *testing.T) {
		builder := NewBuilder()
		// 256 не помещается в 8 бит, должно быть усечено до 0
		AddInteger(builder, 256, WithSize(8))

		bitstring, err := Build(builder)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		matcher := NewMatcher()
		var value int
		Integer(matcher, &value, WithSize(8))

		_, err = Match(matcher, bitstring)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if value != 0 {
			t.Errorf("Expected 0, got %d", value)
		}
	})

	t.Run("Unaligned access", func(t *testing.T) {
		builder := NewBuilder()
		AddInteger(builder, 1, WithSize(1))
		AddInteger(builder, 3, WithSize(2))
		AddInteger(builder, 15, WithSize(4))
		AddInteger(builder, 1, WithSize(1))

		bitstring, err := Build(builder)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if bitstring.Length() != uint(8) {
			t.Errorf("Expected length 8, got %d", bitstring.Length())
		}

		matcher := NewMatcher()
		var b1, b2, b4, b1_2 int
		Integer(matcher, &b1, WithSize(1))
		Integer(matcher, &b2, WithSize(2))
		Integer(matcher, &b4, WithSize(4))
		Integer(matcher, &b1_2, WithSize(1))

		_, err = Match(matcher, bitstring)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if b1 != 1 {
			t.Errorf("Expected b1=1, got %d", b1)
		}
		if b2 != 3 {
			t.Errorf("Expected b2=3, got %d", b2)
		}
		if b4 != 15 {
			t.Errorf("Expected b4=15, got %d", b4)
		}
		if b1_2 != 1 {
			t.Errorf("Expected b1_2=1, got %d", b1_2)
		}
	})

	t.Run("Rest patterns", func(t *testing.T) {
		data := []byte{1, 2, 3, 4, 5, 6, 7, 8}
		bs := NewBitStringFromBytes(data)

		// Binary rest
		matcher := NewMatcher()
		var first, second int
		var rest []byte

		Integer(matcher, &first, WithSize(8))
		Integer(matcher, &second, WithSize(8))
		RestBinary(matcher, &rest)

		_, err := Match(matcher, bs)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if first != 1 {
			t.Errorf("Expected first=1, got %d", first)
		}
		if second != 2 {
			t.Errorf("Expected second=2, got %d", second)
		}
		if !bytesEqual([]byte{3, 4, 5, 6, 7, 8}, rest) {
			t.Errorf("Expected rest=[3,4,5,6,7,8], got %v", rest)
		}
	})

	t.Run("Bitstring rest", func(t *testing.T) {
		builder := NewBuilder()
		AddInteger(builder, 1, WithSize(3))
		AddInteger(builder, 2, WithSize(5))
		AddInteger(builder, 3, WithSize(8))
		AddInteger(builder, 4, WithSize(4))

		bitstring, err := Build(builder)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		matcher := NewMatcher()
		var a, b int
		var rest *BitString

		Integer(matcher, &a, WithSize(3))
		Integer(matcher, &b, WithSize(5))
		RestBitstring(matcher, &rest)

		_, err = Match(matcher, bitstring)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if a != 1 {
			t.Errorf("Expected a=1, got %d", a)
		}
		if b != 2 {
			t.Errorf("Expected b=2, got %d", b)
		}
		if rest.Length() != uint(12) {
			t.Errorf("Expected rest length 12, got %d", rest.Length())
		}
	})
}

// TestConcurrency проверяет потокобезопасность
func TestConcurrency(t *testing.T) {
	// Параллельное создание битовых строк
	t.Run("Concurrent building", func(t *testing.T) {
		const goroutines = 100
		results := make(chan *BitString, goroutines)

		for i := 0; i < goroutines; i++ {
			go func(id int) {
				builder := NewBuilder()
				AddInteger(builder, id, WithSize(32))
				AddBinary(builder, []byte(fmt.Sprintf("goroutine-%d", id)))

				bs, err := Build(builder)
				if err != nil {
					t.Fatalf("Expected no error, got %v", err)
				}
				results <- bs
			}(i)
		}

		// Собираем результаты
		for i := 0; i < goroutines; i++ {
			bs := <-results
			if bs == nil {
				t.Error("Expected non-nil bitstring")
			}
		}
	})

	// Параллельный матчинг
	t.Run("Concurrent matching", func(t *testing.T) {
		// Создаем общую битовую строку
		builder := NewBuilder()
		AddInteger(builder, 42, WithSize(32))
		AddBinary(builder, []byte("shared-data"))
		bitstring, err := Build(builder)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		const goroutines = 100
		results := make(chan int, goroutines)

		for i := 0; i < goroutines; i++ {
			go func() {
				matcher := NewMatcher()
				var value int
				var data []byte

				Integer(matcher, &value, WithSize(32))
				Binary(matcher, &data)

				res, err := Match(matcher, bitstring)
				if err != nil {
					t.Fatalf("Expected no error, got %v", err)
				}
				if len(res) == 0 {
					t.Fatalf("Expected non-empty results")
				}

				results <- value
			}()
		}

		// Проверяем результаты
		for i := 0; i < goroutines; i++ {
			value := <-results
			if value != 42 {
				t.Errorf("Expected 42, got %d", value)
			}
		}
	})
}

// TestMemoryEfficiency проверяет эффективность использования памяти
func TestMemoryEfficiency(t *testing.T) {
	t.Run("Large bitstrings", func(t *testing.T) {
		// Создаем большую битовую строку (1MB)
		size := 1024 * 1024
		data := make([]byte, size)
		for i := range data {
			data[i] = byte(i % 256)
		}

		bs := NewBitStringFromBytes(data)
		if bs.Length() != uint(size*8) {
			t.Errorf("Expected length %d, got %d", size*8, bs.Length())
		}

		// TODO: Implement Slice method for BitString
		// slice, err := bs.Slice(0, 1024*8)
		// if err != nil { t.Fatalf("Expected no error, got %v", err) }
		// assert.Equal(t, uint(1024*8), slice.Length())
	})

	t.Run("Memory reuse", func(t *testing.T) {
		// Проверяем переиспользование буферов
		builder := NewBuilder()

		for i := 0; i < 1000; i++ {
			AddInteger(builder, i, WithSize(32))
		}

		bs, err := Build(builder)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if bs.Length() != uint(32000) {
			t.Errorf("Expected length 32000, got %d", bs.Length())
		}

		// Создаем новый builder для переиспользования
		builder = NewBuilder()

		for i := 0; i < 500; i++ {
			AddInteger(builder, i, WithSize(16))
		}

		bs2, err := Build(builder)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if bs2.Length() != uint(8000) {
			t.Errorf("Expected length 8000, got %d", bs2.Length())
		}
	})
}

// TestErrorHandling проверяет обработку ошибок
func TestErrorHandling(t *testing.T) {
	tests := []struct {
		name        string
		operation   func() error
		expectedErr string
	}{
		{
			name: "invalid size for float",
			operation: func() error {
				builder := NewBuilder()
				AddFloat(builder, 3.14, WithSize(13)) // Invalid size
				_, err := Build(builder)
				return err
			},
			expectedErr: "float size must be",
		},
		{
			name: "pattern too long",
			operation: func() error {
				bs := NewBitStringFromBytes([]byte{1, 2})
				matcher := NewMatcher()
				var a, b, c int
				Integer(matcher, &a, WithSize(8))
				Integer(matcher, &b, WithSize(8))
				Integer(matcher, &c, WithSize(8))
				_, err := Match(matcher, bs)
				return err
			},
			expectedErr: "insufficient bits",
		},
		{
			name: "invalid UTF-8",
			operation: func() error {
				// Invalid UTF-8 sequence
				bs := NewBitStringFromBytes([]byte{0xFF, 0xFE})
				matcher := NewMatcher()
				var text string
				UTF8(matcher, &text)
				_, err := Match(matcher, bs)
				return err
			},
			expectedErr: "invalid UTF-8",
		},
		{
			name: "division by zero in expression",
			operation: func() error {
				// Test division by zero in matcher expression
				builder := NewBuilder()
				AddInteger(builder, 0, WithSize(8))  // Add zero to bitstring
				AddInteger(builder, 42, WithSize(8)) // Add some data
				bs, err := Build(builder)
				if err != nil {
					return err
				}

				matcher := NewMatcher()
				var zero int
				var result []byte
				Integer(matcher, &zero, WithSize(8)) // Extract zero from bitstring
				RegisterVariable(matcher, "zero", &zero)
				Binary(matcher, &result, WithDynamicSizeExpression("32/zero"), WithUnit(1))

				_, err = Match(matcher, bs)
				return err
			},
			expectedErr: "division by zero",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.operation()
			if err == nil {
				t.Error("Expected error, got nil")
			}
			if err != nil && !containsString(err.Error(), tt.expectedErr) {
				t.Errorf("Expected error to contain '%s', got '%s'", tt.expectedErr, err.Error())
			}
		})
	}
}

// BenchmarkConstruction измеряет производительность создания
func BenchmarkConstruction(b *testing.B) {
	b.Run("Simple integers", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			builder := NewBuilder()
			for j := 0; j < 100; j++ {
				AddInteger(builder, j, WithSize(32))
			}
			Build(builder)
		}
	})

	b.Run("Mixed types", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			builder := NewBuilder()
			AddInteger(builder, 42, WithSize(32))
			AddFloat(builder, 3.14, WithSize(32))
			AddBinary(builder, []byte("test data"))
			AddUTF8(builder, "тест")
			Build(builder)
		}
	})
}

// BenchmarkMatching измеряет производительность матчинга
func BenchmarkMatching(b *testing.B) {
	// Подготовка данных
	builder := NewBuilder()
	for i := 0; i < 100; i++ {
		AddInteger(builder, i, WithSize(32))
	}
	bitstring, _ := Build(builder)

	b.ResetTimer()

	b.Run("Simple pattern", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			matcher := NewMatcher()
			var first, second, third int
			Integer(matcher, &first, WithSize(32))
			Integer(matcher, &second, WithSize(32))
			Integer(matcher, &third, WithSize(32))
			Match(matcher, bitstring)
		}
	})

	b.Run("Complex pattern", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			matcher := NewMatcher()
			for j := 0; j < 10; j++ {
				var value int
				Integer(matcher, &value, WithSize(32))
			}
			var rest []byte
			RestBinary(matcher, &rest)
			Match(matcher, bitstring)
		}
	})
}

// TestBigIntSupport проверяет поддержку произвольной точности целых чисел
func TestBigIntSupport(t *testing.T) {
	t.Run("Basic big.Int construction and matching", func(t *testing.T) {
		// Create a huge integer (fits in 64 bits for funbit compatibility)
		hugeInt := new(big.Int)
		hugeInt.SetInt64(9223372036854775806) // Close to int64 max

		builder := NewBuilder()
		AddInteger(builder, hugeInt, WithSize(64)) // 64-bit integer

		bitstring, err := Build(builder)
		if err != nil {
			t.Fatalf("Failed to build big.Int bitstring: %v", err)
		}

		if bitstring.Length() != 64 {
			t.Errorf("Expected 64 bits, got %d", bitstring.Length())
		}

		// Pattern matching extracts as int64, then convert to big.Int
		matcher := NewMatcher()
		var extracted int64
		Integer(matcher, &extracted, WithSize(64))

		results, err := Match(matcher, bitstring)
		if err != nil {
			t.Fatalf("Failed to match big.Int pattern: %v", err)
		}

		if len(results) == 0 {
			t.Fatalf("Expected non-empty results")
		}

		extractedBigInt := big.NewInt(extracted)
		if extractedBigInt.Cmp(hugeInt) != 0 {
			t.Errorf("Expected %s, got %s", hugeInt.String(), extractedBigInt.String())
		}
	})

	t.Run("Mixed regular integers and big.Int", func(t *testing.T) {
		regularInt := 42
		hugeInt := new(big.Int)
		hugeInt.SetInt64(1234567890) // Fits in 64 bits

		builder := NewBuilder()
		AddInteger(builder, regularInt, WithSize(8)) // Regular int
		AddInteger(builder, hugeInt, WithSize(64))   // Big.Int

		bitstring, err := Build(builder)
		if err != nil {
			t.Fatalf("Failed to build mixed bitstring: %v", err)
		}

		if bitstring.Length() != 72 { // 8 + 64
			t.Errorf("Expected 72 bits, got %d", bitstring.Length())
		}

		// Extract both types (both as int64 from funbit)
		matcher := NewMatcher()
		var regularExtracted int
		var hugeExtracted int64

		Integer(matcher, &regularExtracted, WithSize(8))
		Integer(matcher, &hugeExtracted, WithSize(64))

		results, err := Match(matcher, bitstring)
		if err != nil {
			t.Fatalf("Failed to match mixed pattern: %v", err)
		}

		if len(results) == 0 {
			t.Fatalf("Expected non-empty results")
		}

		if regularExtracted != regularInt {
			t.Errorf("Expected %d, got %d", regularInt, regularExtracted)
		}

		hugeExtractedBigInt := big.NewInt(hugeExtracted)
		if hugeExtractedBigInt.Cmp(hugeInt) != 0 {
			t.Errorf("Expected %s, got %s", hugeInt.String(), hugeExtractedBigInt.String())
		}
	})

	t.Run("Big.Int with endianness", func(t *testing.T) {
		testBig := new(big.Int)
		testBig.SetInt64(0x123456789ABCDEF0) // 64-bit hex value as int64

		builder := NewBuilder()
		AddInteger(builder, testBig, WithSize(64), WithEndianness("big"))
		AddInteger(builder, testBig, WithSize(64), WithEndianness("little"))

		bitstring, err := Build(builder)
		if err != nil {
			t.Fatalf("Failed to build endianness test: %v", err)
		}

		if bitstring.Length() != 128 {
			t.Errorf("Expected 128 bits, got %d", bitstring.Length())
		}

		matcher := NewMatcher()
		var bigEndian, littleEndian int64

		Integer(matcher, &bigEndian, WithSize(64), WithEndianness("big"))
		Integer(matcher, &littleEndian, WithSize(64), WithEndianness("little"))

		results, err := Match(matcher, bitstring)
		if err != nil {
			t.Fatalf("Failed to match endianness pattern: %v", err)
		}

		if len(results) == 0 {
			t.Fatalf("Expected non-empty results")
		}

		bigEndianBigInt := big.NewInt(bigEndian)
		littleEndianBigInt := big.NewInt(littleEndian)

		if bigEndianBigInt.Cmp(testBig) != 0 {
			t.Errorf("Big-endian mismatch: expected %s, got %s", testBig.String(), bigEndianBigInt.String())
		}

		if littleEndianBigInt.Cmp(testBig) != 0 {
			t.Errorf("Little-endian mismatch: expected %s, got %s", testBig.String(), littleEndianBigInt.String())
		}
	})

	t.Run("Big.Int signedness", func(t *testing.T) {
		// Test negative big.Int
		negativeBig := new(big.Int)
		negativeBig.SetInt64(-12345)

		builder := NewBuilder()
		AddInteger(builder, negativeBig, WithSize(32), WithSigned(true))

		bitstring, err := Build(builder)
		if err != nil {
			t.Fatalf("Failed to build signed big.Int: %v", err)
		}

		matcher := NewMatcher()
		var extracted int64
		Integer(matcher, &extracted, WithSize(32), WithSigned(true))

		results, err := Match(matcher, bitstring)
		if err != nil {
			t.Fatalf("Failed to match signed big.Int: %v", err)
		}

		if len(results) == 0 {
			t.Fatalf("Expected non-empty results")
		}

		extractedBigInt := big.NewInt(extracted)
		if extractedBigInt.Cmp(negativeBig) != 0 {
			t.Errorf("Expected %s, got %s", negativeBig.String(), extractedBigInt.String())
		}
	})

	t.Run("Big.Int overflow handling", func(t *testing.T) {
		// Test big.Int that fits in 64 bits but is large
		overflowBig := new(big.Int)
		overflowBig.SetInt64(9223372036854775806) // Close to int64 max

		builder := NewBuilder()
		AddInteger(builder, overflowBig, WithSize(64))

		bitstring, err := Build(builder)
		if err != nil {
			t.Fatalf("Failed to build overflow big.Int: %v", err)
		}

		matcher := NewMatcher()
		var extracted int64
		Integer(matcher, &extracted, WithSize(64))

		results, err := Match(matcher, bitstring)
		if err != nil {
			t.Fatalf("Failed to match overflow big.Int: %v", err)
		}

		if len(results) == 0 {
			t.Fatalf("Expected non-empty results")
		}

		extractedBigInt := big.NewInt(extracted)
		if extractedBigInt.Cmp(overflowBig) != 0 {
			t.Errorf("Expected %s, got %s", overflowBig.String(), extractedBigInt.String())
		}

		// Verify it fits in int64
		if !extractedBigInt.IsInt64() {
			t.Error("Expected big.Int to fit in int64")
		}
	})
}

// TestBigIntInDynamicSizing проверяет big.Int в динамических размерах
func TestBigIntInDynamicSizing(t *testing.T) {
	t.Run("Big.Int as dynamic size", func(t *testing.T) {
		// Use big.Int to calculate dynamic size
		sizeValue := new(big.Int)
		sizeValue.SetInt64(5) // 5 bytes

		data := "Hello"

		builder := NewBuilder()
		AddInteger(builder, sizeValue, WithSize(64)) // Store size as big.Int
		AddBinary(builder, []byte(data))

		bitstring, err := Build(builder)
		if err != nil {
			t.Fatalf("Failed to build dynamic size test: %v", err)
		}

		matcher := NewMatcher()
		var extractedSize int64
		var payload []byte

		Integer(matcher, &extractedSize, WithSize(64))
		RegisterVariable(matcher, "size", &extractedSize)
		Binary(matcher, &payload, WithDynamicSizeExpression("size*8"), WithUnit(1))

		results, err := Match(matcher, bitstring)
		if err != nil {
			t.Fatalf("Failed to match dynamic size pattern: %v", err)
		}

		if len(results) == 0 {
			t.Fatalf("Expected non-empty results")
		}

		extractedSizeBigInt := big.NewInt(extractedSize)
		if extractedSizeBigInt.Cmp(sizeValue) != 0 {
			t.Errorf("Expected size %s, got %s", sizeValue.String(), extractedSizeBigInt.String())
		}

		if string(payload) != data {
			t.Errorf("Expected payload %s, got %s", data, string(payload))
		}
	})
}

// BenchmarkBigIntOperations измеряет производительность операций с big.Int
func BenchmarkBigIntOperations(b *testing.B) {
	hugeInt := new(big.Int)
	hugeInt.SetInt64(9223372036854775806) // Fits in 64 bits

	b.Run("Big.Int construction", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			builder := NewBuilder()
			AddInteger(builder, hugeInt, WithSize(64))
			Build(builder)
		}
	})

	b.Run("Big.Int matching", func(b *testing.B) {
		// Pre-build bitstring
		builder := NewBuilder()
		AddInteger(builder, hugeInt, WithSize(64))
		bitstring, _ := Build(builder)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			matcher := NewMatcher()
			var extracted int64
			Integer(matcher, &extracted, WithSize(64))
			Match(matcher, bitstring)
		}
	})

	b.Run("Mixed big.Int and regular int", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			builder := NewBuilder()
			AddInteger(builder, 42, WithSize(8))
			AddInteger(builder, hugeInt, WithSize(64))
			AddInteger(builder, i, WithSize(32))
			Build(builder)
		}
	})
}
