package protocols

import (
	"bytes"
	"testing"

	"github.com/funvibe/funbit/internal/bitstring"
)

func TestBuildPNGHeader(t *testing.T) {
	tests := []struct {
		name     string
		header   PNGHeader
		wantErr  bool
		validate func(*testing.T, *bitstring.BitString)
	}{
		{
			name: "Valid PNG header with IHDR chunk",
			header: PNGHeader{
				Signature: PNGSignature,
				Chunks: []PNGChunk{
					{
						Length: 13,
						Type:   []byte("IHDR"),
						Data:   []byte{0, 0, 0, 1, 0, 0, 0, 1, 8, 2, 0, 0, 0}, // 1x1 RGB image
						CRC:    0,
					},
				},
			},
			wantErr: false,
			validate: func(t *testing.T, bs *bitstring.BitString) {
				if bs == nil {
					t.Error("Expected BitString, got nil")
					return
				}
				byteData := bs.ToBytes()
				// PNG signature (8) + chunk length (4) + chunk type (4) + chunk data (13) + chunk crc (4) = 33 bytes
				expectedLength := 33
				if len(byteData) != expectedLength {
					t.Errorf("Expected %d bytes, got %d", expectedLength, len(byteData))
				}
				// Проверяем сигнатуру
				if !bytes.Equal(byteData[:8], PNGSignature) {
					t.Errorf("PNG signature mismatch")
				}
				// Проверяем тип чанка
				if string(byteData[12:16]) != "IHDR" {
					t.Errorf("Expected IHDR chunk, got %s", string(byteData[12:16]))
				}
			},
		},
		{
			name: "Invalid PNG signature",
			header: PNGHeader{
				Signature: []byte{1, 2, 3, 4, 5, 6, 7, 8}, // Invalid
				Chunks: []PNGChunk{
					{
						Length: 13,
						Type:   []byte("IHDR"),
						Data:   make([]byte, 13),
						CRC:    0,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Empty chunks",
			header: PNGHeader{
				Signature: PNGSignature,
				Chunks:    []PNGChunk{},
			},
			wantErr: false,
			validate: func(t *testing.T, bs *bitstring.BitString) {
				if bs == nil {
					t.Error("Expected BitString, got nil")
					return
				}
				sigBytes := bs.ToBytes()
				// Только сигнатура PNG
				if len(sigBytes) != 8 {
					t.Errorf("Expected 8 bytes (signature only), got %d", len(sigBytes))
				}
				if !bytes.Equal(sigBytes, PNGSignature) {
					t.Errorf("PNG signature mismatch")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BuildPNGHeader(tt.header)
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildPNGHeader() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.validate != nil {
				tt.validate(t, got)
			}
		})
	}
}

func TestBuildPNGChunk(t *testing.T) {
	tests := []struct {
		name     string
		chunk    PNGChunk
		wantErr  bool
		validate func(*testing.T, []byte)
	}{
		{
			name: "Valid IHDR chunk",
			chunk: PNGChunk{
				Length: 13,
				Type:   []byte("IHDR"),
				Data:   []byte{0, 0, 0, 1, 0, 0, 0, 1, 8, 2, 0, 0, 0},
				CRC:    0x8D04A5FA,
			},
			wantErr: false,
			validate: func(t *testing.T, data []byte) {
				if len(data) != 25 { // 4 (len) + 4 (type) + 13 (data) + 4 (crc)
					t.Errorf("Expected 25 bytes, got %d", len(data))
				}
				// Проверяем длину (big-endian)
				length := int(data[0])<<24 | int(data[1])<<16 | int(data[2])<<8 | int(data[3])
				if length != 13 {
					t.Errorf("Expected length 13, got %d", length)
				}
				// Проверяем тип
				if string(data[4:8]) != "IHDR" {
					t.Errorf("Expected IHDR, got %s", string(data[4:8]))
				}
				// Проверяем CRC
				crc := uint32(data[21])<<24 | uint32(data[22])<<16 | uint32(data[23])<<8 | uint32(data[24])
				if crc != 0x8D04A5FA {
					t.Errorf("Expected CRC 0x8D04A5FA, got 0x%08X", crc)
				}
			},
		},
		{
			name: "Invalid chunk type length",
			chunk: PNGChunk{
				Length: 13,
				Type:   []byte("IH"), // Invalid length
				Data:   make([]byte, 13),
				CRC:    0,
			},
			wantErr: true,
		},
		{
			name: "Empty chunk",
			chunk: PNGChunk{
				Length: 0,
				Type:   []byte("IEND"),
				Data:   []byte{},
				CRC:    0,
			},
			wantErr: false,
			validate: func(t *testing.T, data []byte) {
				if len(data) != 12 { // 4 (len) + 4 (type) + 0 (data) + 4 (crc)
					t.Errorf("Expected 12 bytes, got %d", len(data))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BuildPNGChunk(tt.chunk)
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildPNGChunk() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.validate != nil {
				tt.validate(t, got)
			}
		})
	}
}

func TestBuildPNGIHDRChunk(t *testing.T) {
	tests := []struct {
		name     string
		ihdr     PNGIHDRChunk
		wantErr  bool
		validate func(*testing.T, PNGChunk)
	}{
		{
			name: "Valid IHDR chunk",
			ihdr: PNGIHDRChunk{
				Width:       100,
				Height:      100,
				BitDepth:    8,
				ColorType:   2, // RGB
				Compression: 0,
				Filter:      0,
				Interlace:   0,
			},
			wantErr: false,
			validate: func(t *testing.T, chunk PNGChunk) {
				if string(chunk.Type) != "IHDR" {
					t.Errorf("Expected IHDR chunk, got %s", string(chunk.Type))
				}
				if chunk.Length != 13 {
					t.Errorf("Expected length 13, got %d", chunk.Length)
				}
				if len(chunk.Data) != 13 {
					t.Errorf("Expected data length 13, got %d", len(chunk.Data))
				}
				// Проверяем ширину и высоту (big-endian)
				width := uint32(chunk.Data[0])<<24 | uint32(chunk.Data[1])<<16 | uint32(chunk.Data[2])<<8 | uint32(chunk.Data[3])
				height := uint32(chunk.Data[4])<<24 | uint32(chunk.Data[5])<<16 | uint32(chunk.Data[6])<<8 | uint32(chunk.Data[7])
				if width != 100 {
					t.Errorf("Expected width 100, got %d", width)
				}
				if height != 100 {
					t.Errorf("Expected height 100, got %d", height)
				}
				if chunk.Data[8] != 8 { // BitDepth
					t.Errorf("Expected bit depth 8, got %d", chunk.Data[8])
				}
				if chunk.Data[9] != 2 { // ColorType
					t.Errorf("Expected color type 2, got %d", chunk.Data[9])
				}
			},
		},
		{
			name: "Invalid width",
			ihdr: PNGIHDRChunk{
				Width:     0, // Invalid
				Height:    100,
				BitDepth:  8,
				ColorType: 2,
			},
			wantErr: true,
		},
		{
			name: "Invalid height",
			ihdr: PNGIHDRChunk{
				Width:     100,
				Height:    0, // Invalid
				BitDepth:  8,
				ColorType: 2,
			},
			wantErr: true,
		},
		{
			name: "Invalid bit depth",
			ihdr: PNGIHDRChunk{
				Width:     100,
				Height:    100,
				BitDepth:  7, // Invalid
				ColorType: 2,
			},
			wantErr: true,
		},
		{
			name: "Invalid color type",
			ihdr: PNGIHDRChunk{
				Width:     100,
				Height:    100,
				BitDepth:  8,
				ColorType: 5, // Invalid
			},
			wantErr: true,
		},
		{
			name: "Invalid compression method",
			ihdr: PNGIHDRChunk{
				Width:       100,
				Height:      100,
				BitDepth:    8,
				ColorType:   2,
				Compression: 1, // Invalid
			},
			wantErr: true,
		},
		{
			name: "Invalid filter method",
			ihdr: PNGIHDRChunk{
				Width:       100,
				Height:      100,
				BitDepth:    8,
				ColorType:   2,
				Compression: 0,
				Filter:      1, // Invalid
			},
			wantErr: true,
		},
		{
			name: "Invalid interlace method",
			ihdr: PNGIHDRChunk{
				Width:       100,
				Height:      100,
				BitDepth:    8,
				ColorType:   2,
				Compression: 0,
				Filter:      0,
				Interlace:   2, // Invalid
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BuildPNGIHDRChunk(tt.ihdr)
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildPNGIHDRChunk() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.validate != nil {
				tt.validate(t, got)
			}
		})
	}
}

func TestParsePNGHeader(t *testing.T) {
	// Создаем тестовые данные - простой PNG заголовок
	testHeader := PNGHeader{
		Signature: PNGSignature,
		Chunks: []PNGChunk{
			{
				Length: 13,
				Type:   []byte("IHDR"),
				Data:   []byte{0, 0, 0, 1, 0, 0, 0, 1, 8, 2, 0, 0, 0},
				CRC:    0,
			},
		},
	}

	headerBS, err := BuildPNGHeader(testHeader)
	if err != nil {
		t.Fatalf("Failed to build test header: %v", err)
	}

	tests := []struct {
		name     string
		data     *bitstring.BitString
		wantErr  bool
		validate func(*testing.T, *PNGHeader)
	}{
		{
			name:    "Parse valid PNG header",
			data:    headerBS,
			wantErr: false,
			validate: func(t *testing.T, header *PNGHeader) {
				if header == nil {
					t.Error("Expected PNGHeader, got nil")
					return
				}
				if !bytes.Equal(header.Signature, PNGSignature) {
					t.Error("PNG signature mismatch")
				}
				if len(header.Chunks) == 0 {
					t.Error("Expected at least one chunk")
				}
				if string(header.Chunks[0].Type) != "IHDR" {
					t.Errorf("Expected IHDR chunk, got %s", string(header.Chunks[0].Type))
				}
			},
		},
		{
			name:    "Invalid PNG signature",
			data:    bitstring.NewBitStringFromBytes([]byte{1, 2, 3, 4, 5, 6, 7, 8}),
			wantErr: true,
		},
		{
			name:    "Insufficient data",
			data:    bitstring.NewBitStringFromBytes([]byte{137, 80, 78, 71}),
			wantErr: true,
		},
		{
			name:    "Nil data",
			data:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParsePNGHeader(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParsePNGHeader() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.validate != nil {
				tt.validate(t, got)
			}
		})
	}
}

func TestParsePNGChunk(t *testing.T) {
	// Создаем тестовые данные - простой PNG чанк
	testChunk := PNGChunk{
		Length: 13,
		Type:   []byte("IHDR"),
		Data:   []byte{0, 0, 0, 1, 0, 0, 0, 1, 8, 2, 0, 0, 0},
		CRC:    0x8D04A5FA,
	}

	chunkData, err := BuildPNGChunk(testChunk)
	if err != nil {
		t.Fatalf("Failed to build test chunk: %v", err)
	}

	tests := []struct {
		name     string
		data     []byte
		wantErr  bool
		validate func(*testing.T, PNGChunk, uint)
	}{
		{
			name:    "Parse valid PNG chunk",
			data:    chunkData,
			wantErr: false,
			validate: func(t *testing.T, chunk PNGChunk, size uint) {
				if string(chunk.Type) != "IHDR" {
					t.Errorf("Expected IHDR, got %s", string(chunk.Type))
				}
				if chunk.Length != 13 {
					t.Errorf("Expected length 13, got %d", chunk.Length)
				}
				if len(chunk.Data) != 13 {
					t.Errorf("Expected data length 13, got %d", len(chunk.Data))
				}
				if size != 25 { // 4 + 4 + 13 + 4
					t.Errorf("Expected chunk size 25, got %d", size)
				}
			},
		},
		{
			name:    "Insufficient data for chunk header",
			data:    []byte{0, 0, 0},
			wantErr: true,
		},
		{
			name:    "Insufficient data for chunk content",
			data:    append([]byte{0, 0, 0, 13, 'I', 'H', 'D', 'R'}, make([]byte, 10)...), // Only 10 bytes data instead of 13
			wantErr: true,
		},
		{
			name:    "Empty data",
			data:    []byte{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, size, err := ParsePNGChunk(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParsePNGChunk() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.validate != nil {
				tt.validate(t, got, size)
			}
		})
	}
}

func TestParsePNGIHDRChunk(t *testing.T) {
	// Создаем тестовые данные - IHDR чанк
	testIHDR := PNGIHDRChunk{
		Width:       100,
		Height:      200,
		BitDepth:    8,
		ColorType:   2,
		Compression: 0,
		Filter:      0,
		Interlace:   0,
	}

	chunk, err := BuildPNGIHDRChunk(testIHDR)
	if err != nil {
		t.Fatalf("Failed to build test IHDR chunk: %v", err)
	}

	tests := []struct {
		name     string
		data     []byte
		wantErr  bool
		validate func(*testing.T, *PNGIHDRChunk)
	}{
		{
			name:    "Parse valid IHDR chunk",
			data:    chunk.Data,
			wantErr: false,
			validate: func(t *testing.T, ihdr *PNGIHDRChunk) {
				if ihdr == nil {
					t.Error("Expected PNGIHDRChunk, got nil")
					return
				}
				if ihdr.Width != 100 {
					t.Errorf("Expected width 100, got %d", ihdr.Width)
				}
				if ihdr.Height != 200 {
					t.Errorf("Expected height 200, got %d", ihdr.Height)
				}
				if ihdr.BitDepth != 8 {
					t.Errorf("Expected bit depth 8, got %d", ihdr.BitDepth)
				}
				if ihdr.ColorType != 2 {
					t.Errorf("Expected color type 2, got %d", ihdr.ColorType)
				}
				if ihdr.Compression != 0 {
					t.Errorf("Expected compression 0, got %d", ihdr.Compression)
				}
				if ihdr.Filter != 0 {
					t.Errorf("Expected filter 0, got %d", ihdr.Filter)
				}
				if ihdr.Interlace != 0 {
					t.Errorf("Expected interlace 0, got %d", ihdr.Interlace)
				}
			},
		},
		{
			name:    "Insufficient data",
			data:    []byte{0, 0, 0, 1, 0, 0, 0, 1, 8, 2},
			wantErr: true,
		},
		{
			name:    "Invalid IHDR data",
			data:    []byte{0, 0, 0, 0, 0, 0, 0, 1, 8, 2, 0, 0, 0}, // Width = 0
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParsePNGIHDRChunk(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParsePNGIHDRChunk() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.validate != nil {
				tt.validate(t, got)
			}
		})
	}
}

func TestValidatePNGHeader(t *testing.T) {
	tests := []struct {
		name    string
		header  *PNGHeader
		wantErr bool
	}{
		{
			name: "Valid PNG header",
			header: &PNGHeader{
				Signature: PNGSignature,
				Chunks: []PNGChunk{
					{
						Length: 13,
						Type:   []byte("IHDR"),
						Data:   make([]byte, 13),
						CRC:    0,
					},
				},
			},
			wantErr: false,
		},
		{
			name:    "Nil header",
			header:  nil,
			wantErr: true,
		},
		{
			name: "Invalid signature",
			header: &PNGHeader{
				Signature: []byte{1, 2, 3, 4, 5, 6, 7, 8},
				Chunks: []PNGChunk{
					{
						Length: 13,
						Type:   []byte("IHDR"),
						Data:   make([]byte, 13),
						CRC:    0,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "No chunks",
			header: &PNGHeader{
				Signature: PNGSignature,
				Chunks:    []PNGChunk{},
			},
			wantErr: true,
		},
		{
			name: "First chunk not IHDR",
			header: &PNGHeader{
				Signature: PNGSignature,
				Chunks: []PNGChunk{
					{
						Length: 0,
						Type:   []byte("IEND"),
						Data:   []byte{},
						CRC:    0,
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePNGHeader(tt.header)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePNGHeader() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidatePNGChunk(t *testing.T) {
	tests := []struct {
		name    string
		chunk   *PNGChunk
		wantErr bool
	}{
		{
			name: "Valid chunk",
			chunk: &PNGChunk{
				Length: 13,
				Type:   []byte("IHDR"),
				Data:   make([]byte, 13),
				CRC:    0,
			},
			wantErr: false,
		},
		{
			name:    "Nil chunk",
			chunk:   nil,
			wantErr: true,
		},
		{
			name: "Invalid type length",
			chunk: &PNGChunk{
				Length: 13,
				Type:   []byte("IH"),
				Data:   make([]byte, 13),
				CRC:    0,
			},
			wantErr: true,
		},
		{
			name: "Invalid type characters",
			chunk: &PNGChunk{
				Length: 13,
				Type:   []byte("IH1R"), // '1' is not a letter
				Data:   make([]byte, 13),
				CRC:    0,
			},
			wantErr: true,
		},
		{
			name: "Data length mismatch",
			chunk: &PNGChunk{
				Length: 13,
				Type:   []byte("IHDR"),
				Data:   make([]byte, 10), // Length mismatch
				CRC:    0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePNGChunk(tt.chunk)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePNGChunk() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidatePNGIHDRChunk(t *testing.T) {
	tests := []struct {
		name    string
		ihdr    *PNGIHDRChunk
		wantErr bool
	}{
		{
			name: "Valid IHDR chunk",
			ihdr: &PNGIHDRChunk{
				Width:       100,
				Height:      100,
				BitDepth:    8,
				ColorType:   2,
				Compression: 0,
				Filter:      0,
				Interlace:   0,
			},
			wantErr: false,
		},
		{
			name:    "Nil IHDR",
			ihdr:    nil,
			wantErr: true,
		},
		{
			name: "Zero width",
			ihdr: &PNGIHDRChunk{
				Width:     0,
				Height:    100,
				BitDepth:  8,
				ColorType: 2,
			},
			wantErr: true,
		},
		{
			name: "Zero height",
			ihdr: &PNGIHDRChunk{
				Width:     100,
				Height:    0,
				BitDepth:  8,
				ColorType: 2,
			},
			wantErr: true,
		},
		{
			name: "Invalid bit depth",
			ihdr: &PNGIHDRChunk{
				Width:     100,
				Height:    100,
				BitDepth:  7, // Invalid
				ColorType: 2,
			},
			wantErr: true,
		},
		{
			name: "Invalid color type",
			ihdr: &PNGIHDRChunk{
				Width:     100,
				Height:    100,
				BitDepth:  8,
				ColorType: 5, // Invalid
			},
			wantErr: true,
		},
		{
			name: "Incompatible bit depth and color type",
			ihdr: &PNGIHDRChunk{
				Width:     100,
				Height:    100,
				BitDepth:  4, // Invalid with color type 3
				ColorType: 3,
			},
			wantErr: true,
		},
		{
			name: "Invalid compression method",
			ihdr: &PNGIHDRChunk{
				Width:       100,
				Height:      100,
				BitDepth:    8,
				ColorType:   2,
				Compression: 1, // Invalid
			},
			wantErr: true,
		},
		{
			name: "Invalid filter method",
			ihdr: &PNGIHDRChunk{
				Width:       100,
				Height:      100,
				BitDepth:    8,
				ColorType:   2,
				Compression: 0,
				Filter:      1, // Invalid
			},
			wantErr: true,
		},
		{
			name: "Invalid interlace method",
			ihdr: &PNGIHDRChunk{
				Width:       100,
				Height:      100,
				BitDepth:    8,
				ColorType:   2,
				Compression: 0,
				Filter:      0,
				Interlace:   2, // Invalid
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePNGIHDRChunk(tt.ihdr)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePNGIHDRChunk() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetPNGColorTypeName(t *testing.T) {
	tests := []struct {
		name      string
		colorType uint8
		want      string
	}{
		{"Grayscale", 0, "Grayscale"},
		{"RGB", 2, "RGB"},
		{"Indexed", 3, "Indexed"},
		{"Grayscale with Alpha", 4, "Grayscale with Alpha"},
		{"RGB with Alpha", 6, "RGB with Alpha"},
		{"Unknown", 1, "Unknown"},
		{"Unknown", 5, "Unknown"},
		{"Unknown", 7, "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetPNGColorTypeName(tt.colorType); got != tt.want {
				t.Errorf("GetPNGColorTypeName(%d) = %v, want %v", tt.colorType, got, tt.want)
			}
		})
	}
}

func TestGetPNGInterlaceMethodName(t *testing.T) {
	tests := []struct {
		name      string
		interlace uint8
		want      string
	}{
		{"None", 0, "None"},
		{"Adam7", 1, "Adam7"},
		{"Unknown", 2, "Unknown"},
		{"Unknown", 255, "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetPNGInterlaceMethodName(tt.interlace); got != tt.want {
				t.Errorf("GetPNGInterlaceMethodName(%d) = %v, want %v", tt.interlace, got, tt.want)
			}
		})
	}
}

func TestFormatPNGIHDRInfo(t *testing.T) {
	tests := []struct {
		name     string
		ihdr     *PNGIHDRChunk
		contains []string
	}{
		{
			name: "Basic info",
			ihdr: &PNGIHDRChunk{
				Width:       100,
				Height:      200,
				BitDepth:    8,
				ColorType:   2,
				Compression: 0,
				Filter:      0,
				Interlace:   0,
			},
			contains: []string{
				"PNG Image Header:",
				"Width: 100 px",
				"Height: 200 px",
				"Bit Depth: 8 bits",
				"Color Type: 2 (RGB)",
				"Compression: 0",
				"Filter: 0",
				"Interlace: 0 (None)",
			},
		},
		{
			name: "With alpha",
			ihdr: &PNGIHDRChunk{
				Width:       50,
				Height:      75,
				BitDepth:    16,
				ColorType:   6,
				Compression: 0,
				Filter:      0,
				Interlace:   1,
			},
			contains: []string{
				"Width: 50 px",
				"Height: 75 px",
				"Bit Depth: 16 bits",
				"Color Type: 6 (RGB with Alpha)",
				"Interlace: 1 (Adam7)",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatPNGIHDRInfo(tt.ihdr)
			for _, substr := range tt.contains {
				if !containsSubstring(result, substr) {
					t.Errorf("Expected result to contain %q, but it doesn't. Result:\n%s", substr, result)
				}
			}
		})
	}
}

// Helper function
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) &&
			(s[:len(substr)] == substr ||
				s[len(s)-len(substr):] == substr ||
				containsSubstring(s[1:], substr))))
}
