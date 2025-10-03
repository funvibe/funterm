package utf

import (
	"testing"
)

func TestValidateUnicodeCodePoint(t *testing.T) {
	tests := []struct {
		name      string
		codePoint int
		wantErr   bool
	}{
		{
			name:      "Valid ASCII character",
			codePoint: 0x41, // 'A'
			wantErr:   false,
		},
		{
			name:      "Valid BMP character",
			codePoint: 0x20AC, // Euro sign
			wantErr:   false,
		},
		{
			name:      "Valid supplementary plane character",
			codePoint: 0x1F600, // Grinning face emoji
			wantErr:   false,
		},
		{
			name:      "Valid maximum Unicode code point",
			codePoint: 0x10FFFF,
			wantErr:   false,
		},
		{
			name:      "Invalid - negative code point",
			codePoint: -1,
			wantErr:   true,
		},
		{
			name:      "Invalid - too large code point",
			codePoint: 0x110000,
			wantErr:   true,
		},
		{
			name:      "Invalid - high surrogate",
			codePoint: 0xD800,
			wantErr:   true,
		},
		{
			name:      "Invalid - low surrogate",
			codePoint: 0xDC00,
			wantErr:   true,
		},
		{
			name:      "Invalid - surrogate range middle",
			codePoint: 0xDBFF,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUnicodeCodePoint(tt.codePoint)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateUnicodeCodePoint() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIsValidUnicodeCodePoint(t *testing.T) {
	tests := []struct {
		name      string
		codePoint int
		want      bool
	}{
		{
			name:      "Valid ASCII character",
			codePoint: 0x41, // 'A'
			want:      true,
		},
		{
			name:      "Valid BMP character",
			codePoint: 0x20AC, // Euro sign
			want:      true,
		},
		{
			name:      "Valid supplementary plane character",
			codePoint: 0x1F600, // Grinning face emoji
			want:      true,
		},
		{
			name:      "Invalid - negative code point",
			codePoint: -1,
			want:      false,
		},
		{
			name:      "Invalid - too large code point",
			codePoint: 0x110000,
			want:      false,
		},
		{
			name:      "Invalid - high surrogate",
			codePoint: 0xD800,
			want:      false,
		},
		{
			name:      "Invalid - low surrogate",
			codePoint: 0xDC00,
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidUnicodeCodePoint(tt.codePoint); got != tt.want {
				t.Errorf("IsValidUnicodeCodePoint() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUTF8Encoder(t *testing.T) {
	encoder := NewUTF8Encoder()

	t.Run("NewUTF8Encoder", func(t *testing.T) {
		if encoder == nil {
			t.Error("NewUTF8Encoder() returned nil")
		}
	})

	t.Run("Encode", func(t *testing.T) {
		tests := []struct {
			name      string
			codePoint int
			want      []byte
			wantErr   bool
		}{
			{
				name:      "ASCII character",
				codePoint: 0x41, // 'A'
				want:      []byte{0x41},
				wantErr:   false,
			},
			{
				name:      "2-byte UTF-8",
				codePoint: 0x00A9, // Copyright sign
				want:      []byte{0xC2, 0xA9},
				wantErr:   false,
			},
			{
				name:      "3-byte UTF-8",
				codePoint: 0x20AC, // Euro sign
				want:      []byte{0xE2, 0x82, 0xAC},
				wantErr:   false,
			},
			{
				name:      "4-byte UTF-8",
				codePoint: 0x1F600, // Grinning face emoji
				want:      []byte{0xF0, 0x9F, 0x98, 0x80},
				wantErr:   false,
			},
			{
				name:      "Invalid code point",
				codePoint: 0xD800, // Surrogate
				want:      nil,
				wantErr:   true,
			},
			{
				name:      "Too large code point",
				codePoint: 0x110000,
				want:      nil,
				wantErr:   true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, err := encoder.Encode(tt.codePoint)
				if (err != nil) != tt.wantErr {
					t.Errorf("UTF8Encoder.Encode() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				if !tt.wantErr && len(got) != len(tt.want) {
					t.Errorf("UTF8Encoder.Encode() length = %d, want length %d", len(got), len(tt.want))
					return
				}
				for i := range tt.want {
					if got[i] != tt.want[i] {
						t.Errorf("UTF8Encoder.Encode() byte %d = 0x%02X, want 0x%02X", i, got[i], tt.want[i])
					}
				}
			})
		}
	})

	t.Run("Decode", func(t *testing.T) {
		tests := []struct {
			name    string
			data    []byte
			want    int
			wantErr bool
		}{
			{
				name:    "ASCII character",
				data:    []byte{0x41}, // 'A'
				want:    0x41,
				wantErr: false,
			},
			{
				name:    "2-byte UTF-8",
				data:    []byte{0xC2, 0xA9}, // Copyright sign
				want:    0x00A9,
				wantErr: false,
			},
			{
				name:    "3-byte UTF-8",
				data:    []byte{0xE2, 0x82, 0xAC}, // Euro sign
				want:    0x20AC,
				wantErr: false,
			},
			{
				name:    "4-byte UTF-8",
				data:    []byte{0xF0, 0x9F, 0x98, 0x80}, // Grinning face emoji
				want:    0x1F600,
				wantErr: false,
			},
			{
				name:    "Empty data",
				data:    []byte{},
				want:    0,
				wantErr: true,
			},
			{
				name:    "Invalid UTF-8 sequence",
				data:    []byte{0xFF, 0xFF},
				want:    0,
				wantErr: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, err := encoder.Decode(tt.data)
				if (err != nil) != tt.wantErr {
					t.Errorf("UTF8Encoder.Decode() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				if !tt.wantErr && got != tt.want {
					t.Errorf("UTF8Encoder.Decode() = %d, want %d", got, tt.want)
				}
			})
		}
	})

	t.Run("RequiredBytes", func(t *testing.T) {
		tests := []struct {
			name      string
			codePoint int
			want      int
		}{
			{
				name:      "ASCII character",
				codePoint: 0x41, // 'A'
				want:      1,
			},
			{
				name:      "2-byte UTF-8",
				codePoint: 0x00A9, // Copyright sign
				want:      2,
			},
			{
				name:      "3-byte UTF-8",
				codePoint: 0x20AC, // Euro sign
				want:      3,
			},
			{
				name:      "4-byte UTF-8",
				codePoint: 0x1F600, // Grinning face emoji
				want:      4,
			},
			{
				name:      "Invalid code point",
				codePoint: 0xD800, // Surrogate
				want:      0,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if got := encoder.RequiredBytes(tt.codePoint); got != tt.want {
					t.Errorf("UTF8Encoder.RequiredBytes() = %d, want %d", got, tt.want)
				}
			})
		}
	})
}

func TestUTF16Encoder(t *testing.T) {
	encoder := NewUTF16Encoder()

	t.Run("NewUTF16Encoder", func(t *testing.T) {
		if encoder == nil {
			t.Error("NewUTF16Encoder() returned nil")
		}
	})

	t.Run("Encode", func(t *testing.T) {
		tests := []struct {
			name       string
			codePoint  int
			endianness string
			want       []byte
			wantErr    bool
		}{
			{
				name:       "ASCII character big-endian",
				codePoint:  0x41, // 'A'
				endianness: "big",
				want:       []byte{0x00, 0x41},
				wantErr:    false,
			},
			{
				name:       "ASCII character little-endian",
				codePoint:  0x41, // 'A'
				endianness: "little",
				want:       []byte{0x41, 0x00},
				wantErr:    false,
			},
			{
				name:       "BMP character big-endian",
				codePoint:  0x20AC, // Euro sign
				endianness: "big",
				want:       []byte{0x20, 0xAC},
				wantErr:    false,
			},
			{
				name:       "BMP character little-endian",
				codePoint:  0x20AC, // Euro sign
				endianness: "little",
				want:       []byte{0xAC, 0x20},
				wantErr:    false,
			},
			{
				name:       "Supplementary plane character big-endian",
				codePoint:  0x1F600, // Grinning face emoji
				endianness: "big",
				want:       []byte{0xD8, 0x3D, 0xDE, 0x00},
				wantErr:    false,
			},
			{
				name:       "Supplementary plane character little-endian",
				codePoint:  0x1F600, // Grinning face emoji
				endianness: "little",
				want:       []byte{0x3D, 0xD8, 0x00, 0xDE},
				wantErr:    false,
			},
			{
				name:       "Invalid code point",
				codePoint:  0xD800, // Surrogate
				endianness: "big",
				want:       nil,
				wantErr:    true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, err := encoder.Encode(tt.codePoint, tt.endianness)
				if (err != nil) != tt.wantErr {
					t.Errorf("UTF16Encoder.Encode() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				if !tt.wantErr && len(got) != len(tt.want) {
					t.Errorf("UTF16Encoder.Encode() length = %d, want length %d", len(got), len(tt.want))
					return
				}
				for i := range tt.want {
					if got[i] != tt.want[i] {
						t.Errorf("UTF16Encoder.Encode() byte %d = 0x%02X, want 0x%02X", i, got[i], tt.want[i])
					}
				}
			})
		}
	})

	t.Run("Decode", func(t *testing.T) {
		tests := []struct {
			name       string
			data       []byte
			endianness string
			want       int
			wantErr    bool
		}{
			{
				name:       "ASCII character big-endian",
				data:       []byte{0x00, 0x41}, // 'A'
				endianness: "big",
				want:       0x41,
				wantErr:    false,
			},
			{
				name:       "ASCII character little-endian",
				data:       []byte{0x41, 0x00}, // 'A'
				endianness: "little",
				want:       0x41,
				wantErr:    false,
			},
			{
				name:       "BMP character big-endian",
				data:       []byte{0x20, 0xAC}, // Euro sign
				endianness: "big",
				want:       0x20AC,
				wantErr:    false,
			},
			{
				name:       "BMP character little-endian",
				data:       []byte{0xAC, 0x20}, // Euro sign
				endianness: "little",
				want:       0x20AC,
				wantErr:    false,
			},
			{
				name:       "Supplementary plane character big-endian",
				data:       []byte{0xD8, 0x3D, 0xDE, 0x00}, // Grinning face emoji
				endianness: "big",
				want:       0x1F600,
				wantErr:    false,
			},
			{
				name:       "Supplementary plane character little-endian",
				data:       []byte{0x3D, 0xD8, 0x00, 0xDE}, // Grinning face emoji
				endianness: "little",
				want:       0x1F600,
				wantErr:    false,
			},
			{
				name:       "Insufficient data",
				data:       []byte{0x00},
				endianness: "big",
				want:       0,
				wantErr:    true,
			},
			{
				name:       "Invalid surrogate pair",
				data:       []byte{0xD8, 0x00, 0x00, 0x00},
				endianness: "big",
				want:       0,
				wantErr:    true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, err := encoder.Decode(tt.data, tt.endianness)
				if (err != nil) != tt.wantErr {
					t.Errorf("UTF16Encoder.Decode() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				if !tt.wantErr && got != tt.want {
					t.Errorf("UTF16Encoder.Decode() = %d, want %d", got, tt.want)
				}
			})
		}
	})

	t.Run("RequiredBytes", func(t *testing.T) {
		tests := []struct {
			name      string
			codePoint int
			want      int
		}{
			{
				name:      "ASCII character",
				codePoint: 0x41, // 'A'
				want:      2,
			},
			{
				name:      "BMP character",
				codePoint: 0x20AC, // Euro sign
				want:      2,
			},
			{
				name:      "Supplementary plane character",
				codePoint: 0x1F600, // Grinning face emoji
				want:      4,
			},
			{
				name:      "Invalid code point",
				codePoint: 0xD800, // Surrogate
				want:      0,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if got := encoder.RequiredBytes(tt.codePoint); got != tt.want {
					t.Errorf("UTF16Encoder.RequiredBytes() = %d, want %d", got, tt.want)
				}
			})
		}
	})

	t.Run("encodeUint16", func(t *testing.T) {
		tests := []struct {
			name       string
			value      uint16
			endianness string
			want       []byte
		}{
			{
				name:       "Big-endian",
				value:      0x1234,
				endianness: "big",
				want:       []byte{0x12, 0x34},
			},
			{
				name:       "Little-endian",
				value:      0x1234,
				endianness: "little",
				want:       []byte{0x34, 0x12},
			},
			{
				name:       "Default (big-endian)",
				value:      0x1234,
				endianness: "unknown",
				want:       []byte{0x12, 0x34},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got := encoder.encodeUint16(tt.value, tt.endianness)
				if len(got) != len(tt.want) {
					t.Errorf("encodeUint16() length = %d, want length %d", len(got), len(tt.want))
					return
				}
				for i := range tt.want {
					if got[i] != tt.want[i] {
						t.Errorf("encodeUint16() byte %d = 0x%02X, want 0x%02X", i, got[i], tt.want[i])
					}
				}
			})
		}
	})

	t.Run("decodeUint16", func(t *testing.T) {
		tests := []struct {
			name       string
			data       []byte
			endianness string
			want       uint16
		}{
			{
				name:       "Big-endian",
				data:       []byte{0x12, 0x34},
				endianness: "big",
				want:       0x1234,
			},
			{
				name:       "Little-endian",
				data:       []byte{0x34, 0x12},
				endianness: "little",
				want:       0x1234,
			},
			{
				name:       "Default (big-endian)",
				data:       []byte{0x12, 0x34},
				endianness: "unknown",
				want:       0x1234,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got := encoder.decodeUint16(tt.data, tt.endianness)
				if got != tt.want {
					t.Errorf("decodeUint16() = 0x%04X, want 0x%04X", got, tt.want)
				}
			})
		}
	})
}

func TestUTF32Encoder(t *testing.T) {
	encoder := NewUTF32Encoder()

	t.Run("NewUTF32Encoder", func(t *testing.T) {
		if encoder == nil {
			t.Error("NewUTF32Encoder() returned nil")
		}
	})

	t.Run("Encode", func(t *testing.T) {
		tests := []struct {
			name       string
			codePoint  int
			endianness string
			want       []byte
			wantErr    bool
		}{
			{
				name:       "ASCII character big-endian",
				codePoint:  0x41, // 'A'
				endianness: "big",
				want:       []byte{0x00, 0x00, 0x00, 0x41},
				wantErr:    false,
			},
			{
				name:       "ASCII character little-endian",
				codePoint:  0x41, // 'A'
				endianness: "little",
				want:       []byte{0x41, 0x00, 0x00, 0x00},
				wantErr:    false,
			},
			{
				name:       "BMP character big-endian",
				codePoint:  0x20AC, // Euro sign
				endianness: "big",
				want:       []byte{0x00, 0x00, 0x20, 0xAC},
				wantErr:    false,
			},
			{
				name:       "BMP character little-endian",
				codePoint:  0x20AC, // Euro sign
				endianness: "little",
				want:       []byte{0xAC, 0x20, 0x00, 0x00},
				wantErr:    false,
			},
			{
				name:       "Supplementary plane character big-endian",
				codePoint:  0x1F600, // Grinning face emoji
				endianness: "big",
				want:       []byte{0x00, 0x01, 0xF6, 0x00},
				wantErr:    false,
			},
			{
				name:       "Supplementary plane character little-endian",
				codePoint:  0x1F600, // Grinning face emoji
				endianness: "little",
				want:       []byte{0x00, 0xF6, 0x01, 0x00},
				wantErr:    false,
			},
			{
				name:       "Invalid code point",
				codePoint:  0xD800, // Surrogate
				endianness: "big",
				want:       nil,
				wantErr:    true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, err := encoder.Encode(tt.codePoint, tt.endianness)
				if (err != nil) != tt.wantErr {
					t.Errorf("UTF32Encoder.Encode() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				if !tt.wantErr && len(got) != len(tt.want) {
					t.Errorf("UTF32Encoder.Encode() length = %d, want length %d", len(got), len(tt.want))
					return
				}
				for i := range tt.want {
					if got[i] != tt.want[i] {
						t.Errorf("UTF32Encoder.Encode() byte %d = 0x%02X, want 0x%02X", i, got[i], tt.want[i])
					}
				}
			})
		}
	})

	t.Run("Decode", func(t *testing.T) {
		tests := []struct {
			name       string
			data       []byte
			endianness string
			want       int
			wantErr    bool
		}{
			{
				name:       "ASCII character big-endian",
				data:       []byte{0x00, 0x00, 0x00, 0x41}, // 'A'
				endianness: "big",
				want:       0x41,
				wantErr:    false,
			},
			{
				name:       "ASCII character little-endian",
				data:       []byte{0x41, 0x00, 0x00, 0x00}, // 'A'
				endianness: "little",
				want:       0x41,
				wantErr:    false,
			},
			{
				name:       "BMP character big-endian",
				data:       []byte{0x00, 0x00, 0x20, 0xAC}, // Euro sign
				endianness: "big",
				want:       0x20AC,
				wantErr:    false,
			},
			{
				name:       "BMP character little-endian",
				data:       []byte{0xAC, 0x20, 0x00, 0x00}, // Euro sign
				endianness: "little",
				want:       0x20AC,
				wantErr:    false,
			},
			{
				name:       "Supplementary plane character big-endian",
				data:       []byte{0x00, 0x01, 0xF6, 0x00}, // Grinning face emoji
				endianness: "big",
				want:       0x1F600,
				wantErr:    false,
			},
			{
				name:       "Supplementary plane character little-endian",
				data:       []byte{0x00, 0xF6, 0x01, 0x00}, // Grinning face emoji
				endianness: "little",
				want:       0x1F600,
				wantErr:    false,
			},
			{
				name:       "Insufficient data",
				data:       []byte{0x00, 0x00, 0x00},
				endianness: "big",
				want:       0,
				wantErr:    true,
			},
			{
				name:       "Invalid code point",
				data:       []byte{0x00, 0x11, 0x00, 0x00}, // Too large
				endianness: "big",
				want:       0,
				wantErr:    true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, err := encoder.Decode(tt.data, tt.endianness)
				if (err != nil) != tt.wantErr {
					t.Errorf("UTF32Encoder.Decode() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				if !tt.wantErr && got != tt.want {
					t.Errorf("UTF32Encoder.Decode() = %d, want %d", got, tt.want)
				}
			})
		}
	})

	t.Run("RequiredBytes", func(t *testing.T) {
		tests := []struct {
			name      string
			codePoint int
			want      int
		}{
			{
				name:      "ASCII character",
				codePoint: 0x41, // 'A'
				want:      4,
			},
			{
				name:      "BMP character",
				codePoint: 0x20AC, // Euro sign
				want:      4,
			},
			{
				name:      "Supplementary plane character",
				codePoint: 0x1F600, // Grinning face emoji
				want:      4,
			},
			{
				name:      "Invalid code point",
				codePoint: 0xD800, // Surrogate
				want:      0,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if got := encoder.RequiredBytes(tt.codePoint); got != tt.want {
					t.Errorf("UTF32Encoder.RequiredBytes() = %d, want %d", got, tt.want)
				}
			})
		}
	})

	t.Run("encodeUint32", func(t *testing.T) {
		tests := []struct {
			name       string
			value      uint32
			endianness string
			want       []byte
		}{
			{
				name:       "Big-endian",
				value:      0x12345678,
				endianness: "big",
				want:       []byte{0x12, 0x34, 0x56, 0x78},
			},
			{
				name:       "Little-endian",
				value:      0x12345678,
				endianness: "little",
				want:       []byte{0x78, 0x56, 0x34, 0x12},
			},
			{
				name:       "Default (big-endian)",
				value:      0x12345678,
				endianness: "unknown",
				want:       []byte{0x12, 0x34, 0x56, 0x78},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got := encoder.encodeUint32(tt.value, tt.endianness)
				if len(got) != len(tt.want) {
					t.Errorf("encodeUint32() length = %d, want length %d", len(got), len(tt.want))
					return
				}
				for i := range tt.want {
					if got[i] != tt.want[i] {
						t.Errorf("encodeUint32() byte %d = 0x%02X, want 0x%02X", i, got[i], tt.want[i])
					}
				}
			})
		}
	})

	t.Run("decodeUint32", func(t *testing.T) {
		tests := []struct {
			name       string
			data       []byte
			endianness string
			want       uint32
		}{
			{
				name:       "Big-endian",
				data:       []byte{0x12, 0x34, 0x56, 0x78},
				endianness: "big",
				want:       0x12345678,
			},
			{
				name:       "Little-endian",
				data:       []byte{0x78, 0x56, 0x34, 0x12},
				endianness: "little",
				want:       0x12345678,
			},
			{
				name:       "Default (big-endian)",
				data:       []byte{0x12, 0x34, 0x56, 0x78},
				endianness: "unknown",
				want:       0x12345678,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got := encoder.decodeUint32(tt.data, tt.endianness)
				if got != tt.want {
					t.Errorf("decodeUint32() = 0x%08X, want 0x%08X", got, tt.want)
				}
			})
		}
	})
}

func TestGetEncoder(t *testing.T) {
	tests := []struct {
		name     string
		utfType  string
		wantType interface{}
	}{
		{
			name:     "UTF-8 encoder",
			utfType:  "utf8",
			wantType: &UTF8Encoder{},
		},
		{
			name:     "UTF-16 encoder",
			utfType:  "utf16",
			wantType: &UTF16Encoder{},
		},
		{
			name:     "UTF-32 encoder",
			utfType:  "utf32",
			wantType: &UTF32Encoder{},
		},
		{
			name:     "Unknown encoder type",
			utfType:  "unknown",
			wantType: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetEncoder(tt.utfType)

			if tt.wantType == nil {
				if got != nil {
					t.Errorf("GetEncoder() = %v, want nil", got)
				}
			} else {
				if got == nil {
					t.Errorf("GetEncoder() = nil, want non-nil")
					return
				}

				// Check type
				switch tt.wantType.(type) {
				case *UTF8Encoder:
					if _, ok := got.(*UTF8Encoder); !ok {
						t.Errorf("GetEncoder() = %T, want *UTF8Encoder", got)
					}
				case *UTF16Encoder:
					if _, ok := got.(*UTF16Encoder); !ok {
						t.Errorf("GetEncoder() = %T, want *UTF16Encoder", got)
					}
				case *UTF32Encoder:
					if _, ok := got.(*UTF32Encoder); !ok {
						t.Errorf("GetEncoder() = %T, want *UTF32Encoder", got)
					}
				}
			}
		})
	}
}
