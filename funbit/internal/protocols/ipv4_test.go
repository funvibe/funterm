package protocols

import (
	"testing"

	"github.com/funvibe/funbit/internal/bitstring"
)

func TestBuildIPv4Header(t *testing.T) {
	tests := []struct {
		name     string
		header   IPv4Header
		wantErr  bool
		validate func(*testing.T, *bitstring.BitString)
	}{
		{
			name: "Basic IPv4 header",
			header: IPv4Header{
				Version:        4,
				HeaderLength:   5,
				ServiceType:    0,
				TotalLength:    40,
				Identification: 12345,
				Flags:          2, // Don't Fragment
				FragmentOffset: 0,
				TTL:            64,
				Protocol:       6, // TCP
				Checksum:       0,
				SourceIP:       0xC0A80101, // 192.168.1.1
				DestinationIP:  0xC0A80102, // 192.168.1.2
			},
			wantErr: false,
			validate: func(t *testing.T, bs *bitstring.BitString) {
				if bs == nil {
					t.Error("Expected BitString, got nil")
					return
				}
				// IPv4 заголовок должен быть 20 байт = 160 бит
				if bs.Length() != 160 {
					t.Errorf("Expected BitString size 160, got %d", bs.Length())
				}
				bytes := bs.ToBytes()
				if len(bytes) != 20 {
					t.Errorf("Expected 20 bytes, got %d", len(bytes))
				}
				// Проверяем версию и длину заголовка (первый байт)
				if bytes[0] != 0x45 { // Version 4 (4), Header Length 5 (5)
					t.Errorf("Expected version and header length 0x45, got 0x%02X", bytes[0])
				}
				// Проверяем протокол (10-й байт, индекс 9)
				if bytes[9] != 6 { // TCP
					t.Errorf("Expected protocol 6 (TCP), got %d", bytes[9])
				}
			},
		},
		{
			name: "Minimal IPv4 header",
			header: IPv4Header{
				Version:        4,
				HeaderLength:   5,
				ServiceType:    0,
				TotalLength:    20,
				Identification: 1,
				Flags:          0,
				FragmentOffset: 0,
				TTL:            1,
				Protocol:       1, // ICMP
				Checksum:       0,
				SourceIP:       1,
				DestinationIP:  2,
			},
			wantErr: false,
			validate: func(t *testing.T, bs *bitstring.BitString) {
				if bs == nil {
					t.Error("Expected BitString, got nil")
					return
				}
				if bs.Length() != 160 {
					t.Errorf("Expected BitString size 160, got %d", bs.Length())
				}
				bytes := bs.ToBytes()
				if len(bytes) != 20 {
					t.Errorf("Expected 20 bytes, got %d", len(bytes))
				}
			},
		},
		{
			name: "Maximum values",
			header: IPv4Header{
				Version:        4,
				HeaderLength:   15,
				ServiceType:    255,
				TotalLength:    65535,
				Identification: 65535,
				Flags:          7,
				FragmentOffset: 8191,
				TTL:            255,
				Protocol:       255,
				Checksum:       65535,
				SourceIP:       0xFFFFFFFF,
				DestinationIP:  0xFFFFFFFF,
			},
			wantErr: false,
			validate: func(t *testing.T, bs *bitstring.BitString) {
				if bs == nil {
					t.Error("Expected BitString, got nil")
					return
				}
				if bs.Length() != 160 {
					t.Errorf("Expected BitString size 160, got %d", bs.Length())
				}
				bytes := bs.ToBytes()
				if len(bytes) != 20 {
					t.Errorf("Expected 20 bytes, got %d", len(bytes))
				}
				// Проверяем версию и длину заголовка (первый байт)
				if bytes[0] != 0x4F { // Version 4 (4), Header Length 15 (F)
					t.Errorf("Expected version and header length 0x4F, got 0x%02X", bytes[0])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BuildIPv4Header(tt.header)
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildIPv4Header() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.validate != nil {
				tt.validate(t, got)
			}
		})
	}
}

func TestParseIPv4Header(t *testing.T) {
	// Создаем тестовые данные - простой IPv4 заголовок
	testHeader := IPv4Header{
		Version:        4,
		HeaderLength:   5,
		ServiceType:    0,
		TotalLength:    40,
		Identification: 12345,
		Flags:          2, // Don't Fragment
		FragmentOffset: 0,
		TTL:            64,
		Protocol:       6, // TCP
		Checksum:       0,
		SourceIP:       0xC0A80101, // 192.168.1.1
		DestinationIP:  0xC0A80102, // 192.168.1.2
	}

	headerBS, err := BuildIPv4Header(testHeader)
	if err != nil {
		t.Fatalf("Failed to build test header: %v", err)
	}

	tests := []struct {
		name     string
		data     *bitstring.BitString
		wantErr  bool
		validate func(*testing.T, *IPv4Header)
	}{
		{
			name:    "Parse valid IPv4 header",
			data:    headerBS,
			wantErr: false,
			validate: func(t *testing.T, header *IPv4Header) {
				if header == nil {
					t.Error("Expected IPv4Header, got nil")
					return
				}
				if header.Version != 4 {
					t.Errorf("Expected Version 4, got %d", header.Version)
				}
				if header.HeaderLength != 5 {
					t.Errorf("Expected HeaderLength 5, got %d", header.HeaderLength)
				}
				if header.ServiceType != 0 {
					t.Errorf("Expected ServiceType 0, got %d", header.ServiceType)
				}
				if header.TotalLength != 40 {
					t.Errorf("Expected TotalLength 40, got %d", header.TotalLength)
				}
				if header.Identification != 12345 {
					t.Errorf("Expected Identification 12345, got %d", header.Identification)
				}
				if header.Flags != 2 {
					t.Errorf("Expected Flags 2, got %d", header.Flags)
				}
				if header.FragmentOffset != 0 {
					t.Errorf("Expected FragmentOffset 0, got %d", header.FragmentOffset)
				}
				if header.TTL != 64 {
					t.Errorf("Expected TTL 64, got %d", header.TTL)
				}
				if header.Protocol != 6 {
					t.Errorf("Expected Protocol 6, got %d", header.Protocol)
				}
				if header.Checksum != 0 {
					t.Errorf("Expected Checksum 0, got %d", header.Checksum)
				}
				if header.SourceIP != 0xC0A80101 {
					t.Errorf("Expected SourceIP 0xC0A80101, got 0x%08X", header.SourceIP)
				}
				if header.DestinationIP != 0xC0A80102 {
					t.Errorf("Expected DestinationIP 0xC0A80102, got 0x%08X", header.DestinationIP)
				}
			},
		},
		{
			name:    "Invalid version",
			data:    createIPv4HeaderWithVersion(5), // Invalid version
			wantErr: true,
		},
		{
			name:    "Invalid header length",
			data:    createIPv4HeaderWithHeaderLength(4), // Invalid header length
			wantErr: true,
		},
		{
			name:    "Insufficient data",
			data:    bitstring.NewBitStringFromBytes([]byte{0, 0, 0}),
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
			got, err := ParseIPv4Header(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseIPv4Header() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.validate != nil {
				tt.validate(t, got)
			}
		})
	}
}

func TestValidateIPv4Header(t *testing.T) {
	tests := []struct {
		name    string
		header  *IPv4Header
		wantErr bool
	}{
		{
			name: "Valid header",
			header: &IPv4Header{
				Version:        4,
				HeaderLength:   5,
				ServiceType:    0,
				TotalLength:    40,
				Identification: 12345,
				Flags:          2, // Don't Fragment
				FragmentOffset: 0,
				TTL:            64,
				Protocol:       6, // TCP
				Checksum:       0,
				SourceIP:       0xC0A80101,
				DestinationIP:  0xC0A80102,
			},
			wantErr: false,
		},
		{
			name:    "Nil header",
			header:  nil,
			wantErr: true,
		},
		{
			name: "Invalid version",
			header: &IPv4Header{
				Version:      5, // Invalid
				HeaderLength: 5,
				TotalLength:  40,
			},
			wantErr: true,
		},
		{
			name: "Invalid header length - too small",
			header: &IPv4Header{
				Version:      4,
				HeaderLength: 4, // Invalid
				TotalLength:  40,
			},
			wantErr: true,
		},
		{
			name: "Invalid header length - too large",
			header: &IPv4Header{
				Version:      4,
				HeaderLength: 16, // Invalid
				TotalLength:  40,
			},
			wantErr: true,
		},
		{
			name: "Invalid total length - too small",
			header: &IPv4Header{
				Version:      4,
				HeaderLength: 5,
				TotalLength:  19, // Invalid
			},
			wantErr: true,
		},
		{
			name: "Invalid flags - reserved bits set",
			header: &IPv4Header{
				Version:      4,
				HeaderLength: 5,
				TotalLength:  40,
				Flags:        0xE0, // Reserved bits set
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateIPv4Header(tt.header)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateIPv4Header() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetIPv4HeaderLength(t *testing.T) {
	tests := []struct {
		name   string
		header *IPv4Header
		want   uint
	}{
		{
			name: "Minimal header length",
			header: &IPv4Header{
				HeaderLength: 5,
			},
			want: 20, // 5 * 4
		},
		{
			name: "Maximum header length",
			header: &IPv4Header{
				HeaderLength: 15,
			},
			want: 60, // 15 * 4
		},
		{
			name: "Middle value",
			header: &IPv4Header{
				HeaderLength: 10,
			},
			want: 40, // 10 * 4
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetIPv4HeaderLength(tt.header); got != tt.want {
				t.Errorf("GetIPv4HeaderLength() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetIPv4PayloadLength(t *testing.T) {
	tests := []struct {
		name   string
		header *IPv4Header
		want   uint
	}{
		{
			name: "No payload",
			header: &IPv4Header{
				HeaderLength: 5,
				TotalLength:  20,
			},
			want: 0, // 20 - 20
		},
		{
			name: "Small payload",
			header: &IPv4Header{
				HeaderLength: 5,
				TotalLength:  40,
			},
			want: 20, // 40 - 20
		},
		{
			name: "Large payload with options",
			header: &IPv4Header{
				HeaderLength: 10, // 40 bytes header
				TotalLength:  100,
			},
			want: 60, // 100 - 40
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetIPv4PayloadLength(tt.header); got != tt.want {
				t.Errorf("GetIPv4PayloadLength() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Helper functions

func createIPv4HeaderWithVersion(version uint) *bitstring.BitString {
	header := IPv4Header{
		Version:        version,
		HeaderLength:   5,
		ServiceType:    0,
		TotalLength:    40,
		Identification: 12345,
		Flags:          2,
		FragmentOffset: 0,
		TTL:            64,
		Protocol:       6,
		Checksum:       0,
		SourceIP:       0xC0A80101,
		DestinationIP:  0xC0A80102,
	}
	bs, _ := BuildIPv4Header(header)
	return bs
}

func createIPv4HeaderWithHeaderLength(headerLength uint) *bitstring.BitString {
	header := IPv4Header{
		Version:        4,
		HeaderLength:   headerLength,
		ServiceType:    0,
		TotalLength:    40,
		Identification: 12345,
		Flags:          2,
		FragmentOffset: 0,
		TTL:            64,
		Protocol:       6,
		Checksum:       0,
		SourceIP:       0xC0A80101,
		DestinationIP:  0xC0A80102,
	}
	bs, _ := BuildIPv4Header(header)
	return bs
}
