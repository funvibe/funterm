package protocols

import (
	"testing"

	"github.com/funvibe/funbit/internal/bitstring"
)

func TestBuildNetworkPacket(t *testing.T) {
	tests := []struct {
		name     string
		packet   NetworkPacket
		wantErr  bool
		validate func(*testing.T, *bitstring.BitString)
	}{
		{
			name: "Valid network packet",
			packet: NetworkPacket{
				IPv4Header: IPv4Header{
					Version:        4,
					HeaderLength:   5,
					ServiceType:    0,
					TotalLength:    45, // 20 (IP) + 20 (TCP) + 5 (payload)
					Identification: 12345,
					Flags:          2, // Don't Fragment
					FragmentOffset: 0,
					TTL:            64,
					Protocol:       6, // TCP
					Checksum:       0,
					SourceIP:       0xC0A80101, // 192.168.1.1
					DestinationIP:  0xC0A80102, // 192.168.1.2
				},
				TCPHeader: TCPHeader{
					SourcePort:      8080,
					DestinationPort: 80,
					SequenceNumber:  1000,
					Acknowledgment:  2000,
					DataOffset:      5,
					Reserved:        0,
					Flags: TCPFlags{
						Reserved: 0,
						URG:      0,
						ACK:      1,
						PSH:      1,
						RST:      0,
						SYN:      0,
						FIN:      0,
					},
					Window:        8192,
					Checksum:      0,
					UrgentPointer: 0,
				},
				Payload: []byte("hello"),
			},
			wantErr: false,
			validate: func(t *testing.T, bs *bitstring.BitString) {
				if bs == nil {
					t.Error("Expected BitString, got nil")
					return
				}
				// IP header (20 bytes) + TCP header (20 bytes) + payload (5 bytes) = 45 bytes = 360 bits
				if bs.Length() != 360 {
					t.Errorf("Expected BitString size 360, got %d", bs.Length())
				}
				bytes := bs.ToBytes()
				if len(bytes) != 45 {
					t.Errorf("Expected 45 bytes, got %d", len(bytes))
				}
				// Проверяем версию и длину заголовка IP (первый байт)
				if bytes[0] != 0x45 { // Version 4 (4), Header Length 5 (5)
					t.Errorf("Expected version and header length 0x45, got 0x%02X", bytes[0])
				}
				// Проверяем протокол IP (10-й байт, индекс 9)
				if bytes[9] != 6 { // TCP
					t.Errorf("Expected protocol 6 (TCP), got %d", bytes[9])
				}
				// Проверяем порты TCP (байты 20-23)
				if bytes[20] != 0x1F || bytes[21] != 0x90 { // 8080
					t.Errorf("Expected source port 8080, got %d", int(bytes[20])<<8|int(bytes[21]))
				}
				if bytes[22] != 0x00 || bytes[23] != 0x50 { // 80
					t.Errorf("Expected destination port 80, got %d", int(bytes[22])<<8|int(bytes[23]))
				}
				// Проверяем payload
				if string(bytes[40:]) != "hello" {
					t.Errorf("Expected payload 'hello', got '%s'", string(bytes[40:]))
				}
			},
		},
		{
			name: "Network packet without payload",
			packet: NetworkPacket{
				IPv4Header: IPv4Header{
					Version:        4,
					HeaderLength:   5,
					ServiceType:    0,
					TotalLength:    40, // 20 (IP) + 20 (TCP)
					Identification: 12345,
					Flags:          2,
					FragmentOffset: 0,
					TTL:            64,
					Protocol:       6, // TCP
					Checksum:       0,
					SourceIP:       0xC0A80101,
					DestinationIP:  0xC0A80102,
				},
				TCPHeader: TCPHeader{
					SourcePort:      8080,
					DestinationPort: 80,
					SequenceNumber:  1000,
					Acknowledgment:  0,
					DataOffset:      5,
					Reserved:        0,
					Flags: TCPFlags{
						Reserved: 0,
						URG:      0,
						ACK:      0,
						PSH:      0,
						RST:      0,
						SYN:      1,
						FIN:      0,
					},
					Window:        8192,
					Checksum:      0,
					UrgentPointer: 0,
				},
				Payload: []byte{},
			},
			wantErr: false,
			validate: func(t *testing.T, bs *bitstring.BitString) {
				if bs == nil {
					t.Error("Expected BitString, got nil")
					return
				}
				if bs.Length() != 320 { // 40 bytes = 320 bits
					t.Errorf("Expected BitString size 320, got %d", bs.Length())
				}
				bytes := bs.ToBytes()
				if len(bytes) != 40 {
					t.Errorf("Expected 40 bytes, got %d", len(bytes))
				}
			},
		},
		{
			name: "Valid packet with minimal data",
			packet: NetworkPacket{
				IPv4Header: IPv4Header{
					Version:        4,
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
				},
				TCPHeader: TCPHeader{
					SourcePort:      8080,
					DestinationPort: 80,
					SequenceNumber:  1000,
					Acknowledgment:  0,
					DataOffset:      5,
					Reserved:        0,
					Flags: TCPFlags{
						Reserved: 0,
						URG:      0,
						ACK:      0,
						PSH:      0,
						RST:      0,
						SYN:      1,
						FIN:      0,
					},
					Window:        8192,
					Checksum:      0,
					UrgentPointer: 0,
				},
				Payload: []byte{},
			},
			wantErr: false,
			validate: func(t *testing.T, bs *bitstring.BitString) {
				if bs == nil {
					t.Error("Expected BitString, got nil")
					return
				}
				if bs.Length() != 320 { // 40 bytes = 320 bits
					t.Errorf("Expected BitString size 320, got %d", bs.Length())
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BuildNetworkPacket(tt.packet)
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildNetworkPacket() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.validate != nil {
				tt.validate(t, got)
			}
		})
	}
}

func TestParseNetworkPacket(t *testing.T) {
	// Создаем тестовые данные - простой сетевой пакет
	testPacket := NetworkPacket{
		IPv4Header: IPv4Header{
			Version:        4,
			HeaderLength:   5,
			ServiceType:    0,
			TotalLength:    45, // 20 (IP) + 20 (TCP) + 5 (payload)
			Identification: 12345,
			Flags:          2, // Don't Fragment
			FragmentOffset: 0,
			TTL:            64,
			Protocol:       6, // TCP
			Checksum:       0,
			SourceIP:       0xC0A80101, // 192.168.1.1
			DestinationIP:  0xC0A80102, // 192.168.1.2
		},
		TCPHeader: TCPHeader{
			SourcePort:      8080,
			DestinationPort: 80,
			SequenceNumber:  1000,
			Acknowledgment:  2000,
			DataOffset:      5,
			Reserved:        0,
			Flags: TCPFlags{
				Reserved: 0,
				URG:      0,
				ACK:      1,
				PSH:      1,
				RST:      0,
				SYN:      0,
				FIN:      0,
			},
			Window:        8192,
			Checksum:      0,
			UrgentPointer: 0,
		},
		Payload: []byte("hello"),
	}

	packetBS, err := BuildNetworkPacket(testPacket)
	if err != nil {
		t.Fatalf("Failed to build test packet: %v", err)
	}

	tests := []struct {
		name     string
		data     *bitstring.BitString
		wantErr  bool
		validate func(*testing.T, *NetworkPacket)
	}{
		{
			name:    "Parse valid network packet",
			data:    packetBS,
			wantErr: false,
			validate: func(t *testing.T, packet *NetworkPacket) {
				if packet == nil {
					t.Error("Expected NetworkPacket, got nil")
					return
				}
				// Проверяем IPv4 заголовок
				if packet.IPv4Header.Version != 4 {
					t.Errorf("Expected IPv4 version 4, got %d", packet.IPv4Header.Version)
				}
				if packet.IPv4Header.HeaderLength != 5 {
					t.Errorf("Expected IPv4 header length 5, got %d", packet.IPv4Header.HeaderLength)
				}
				if packet.IPv4Header.Protocol != 6 {
					t.Errorf("Expected protocol 6 (TCP), got %d", packet.IPv4Header.Protocol)
				}
				if packet.IPv4Header.SourceIP != 0xC0A80101 {
					t.Errorf("Expected source IP 0xC0A80101, got 0x%08X", packet.IPv4Header.SourceIP)
				}
				if packet.IPv4Header.DestinationIP != 0xC0A80102 {
					t.Errorf("Expected destination IP 0xC0A80102, got 0x%08X", packet.IPv4Header.DestinationIP)
				}

				// Проверяем TCP заголовок
				if packet.TCPHeader.SourcePort != 8080 {
					t.Errorf("Expected source port 8080, got %d", packet.TCPHeader.SourcePort)
				}
				if packet.TCPHeader.DestinationPort != 80 {
					t.Errorf("Expected destination port 80, got %d", packet.TCPHeader.DestinationPort)
				}
				if packet.TCPHeader.SequenceNumber != 1000 {
					t.Errorf("Expected sequence number 1000, got %d", packet.TCPHeader.SequenceNumber)
				}
				if packet.TCPHeader.Acknowledgment != 2000 {
					t.Errorf("Expected acknowledgment 2000, got %d", packet.TCPHeader.Acknowledgment)
				}
				if packet.TCPHeader.Flags.ACK != 1 {
					t.Errorf("Expected ACK flag 1, got %d", packet.TCPHeader.Flags.ACK)
				}
				if packet.TCPHeader.Flags.PSH != 1 {
					t.Errorf("Expected PSH flag 1, got %d", packet.TCPHeader.Flags.PSH)
				}

				// Проверяем payload
				if string(packet.Payload) != "hello" {
					t.Errorf("Expected payload 'hello', got '%s'", string(packet.Payload))
				}
			},
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
			got, err := ParseNetworkPacket(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseNetworkPacket() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.validate != nil {
				tt.validate(t, got)
			}
		})
	}
}

func TestValidateNetworkPacket(t *testing.T) {
	tests := []struct {
		name    string
		packet  *NetworkPacket
		wantErr bool
	}{
		{
			name: "Valid network packet",
			packet: &NetworkPacket{
				IPv4Header: IPv4Header{
					Version:        4,
					HeaderLength:   5,
					ServiceType:    0,
					TotalLength:    45, // 20 (IP) + 20 (TCP) + 5 (payload)
					Identification: 12345,
					Flags:          2, // Don't Fragment
					FragmentOffset: 0,
					TTL:            64,
					Protocol:       6, // TCP
					Checksum:       0,
					SourceIP:       0xC0A80101,
					DestinationIP:  0xC0A80102,
				},
				TCPHeader: TCPHeader{
					SourcePort:      8080,
					DestinationPort: 80,
					SequenceNumber:  1000,
					Acknowledgment:  2000,
					DataOffset:      5,
					Reserved:        0,
					Flags: TCPFlags{
						Reserved: 0,
						URG:      0,
						ACK:      1,
						PSH:      1,
						RST:      0,
						SYN:      0,
						FIN:      0,
					},
					Window:        8192,
					Checksum:      0,
					UrgentPointer: 0,
				},
				Payload: []byte("hello"),
			},
			wantErr: false,
		},
		{
			name:    "Nil packet",
			packet:  nil,
			wantErr: true,
		},
		{
			name: "Invalid IPv4 header",
			packet: &NetworkPacket{
				IPv4Header: IPv4Header{
					Version:      5, // Invalid
					HeaderLength: 5,
					TotalLength:  45,
					Protocol:     6,
				},
				TCPHeader: TCPHeader{
					SourcePort:      8080,
					DestinationPort: 80,
					DataOffset:      5,
				},
			},
			wantErr: true,
		},
		{
			name: "Invalid TCP header",
			packet: &NetworkPacket{
				IPv4Header: IPv4Header{
					Version:      4,
					HeaderLength: 5,
					TotalLength:  45,
					Protocol:     6,
				},
				TCPHeader: TCPHeader{
					SourcePort:      70000, // Invalid
					DestinationPort: 80,
					DataOffset:      5,
				},
			},
			wantErr: true,
		},
		{
			name: "Protocol mismatch",
			packet: &NetworkPacket{
				IPv4Header: IPv4Header{
					Version:      4,
					HeaderLength: 5,
					TotalLength:  45,
					Protocol:     17, // UDP
				},
				TCPHeader: TCPHeader{
					SourcePort:      8080,
					DestinationPort: 80,
					DataOffset:      5,
				},
			},
			wantErr: true,
		},
		{
			name: "Packet length mismatch",
			packet: &NetworkPacket{
				IPv4Header: IPv4Header{
					Version:      4,
					HeaderLength: 5,
					TotalLength:  100, // Wrong length
					Protocol:     6,
				},
				TCPHeader: TCPHeader{
					SourcePort:      8080,
					DestinationPort: 80,
					DataOffset:      5,
				},
				Payload: []byte("hello"),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateNetworkPacket(tt.packet)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateNetworkPacket() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFormatNetworkPacketInfo(t *testing.T) {
	tests := []struct {
		name     string
		packet   *NetworkPacket
		contains []string
	}{
		{
			name: "Basic HTTP packet",
			packet: &NetworkPacket{
				IPv4Header: IPv4Header{
					Version:       4,
					HeaderLength:  5,
					TotalLength:   70,
					Protocol:      6, // TCP
					TTL:           64,
					SourceIP:      0xC0A80101, // 192.168.1.1
					DestinationIP: 0xC0A80102, // 192.168.1.2
				},
				TCPHeader: TCPHeader{
					SourcePort:      12345,
					DestinationPort: 80,
					SequenceNumber:  1000,
					Acknowledgment:  2000,
					DataOffset:      5,
					Flags: TCPFlags{
						URG: 0,
						ACK: 1,
						PSH: 1,
						RST: 0,
						SYN: 0,
						FIN: 0,
					},
					Window: 8192,
				},
				Payload: []byte("GET / HTTP/1.1\r\nHost: example.com\r\n\r\n"),
			},
			contains: []string{
				"=== IPv4 Header ===",
				"Version: 4",
				"Header Length: 20 bytes",
				"Total Length: 70 bytes",
				"Protocol: 6 (TCP)",
				"TTL: 64",
				"Source IP: 192.168.1.1",
				"Destination IP: 192.168.1.2",
				"=== TCP Header ===",
				"Source Port: 12345",
				"Destination Port: 80",
				"Sequence Number: 1000",
				"Acknowledgment: 2000",
				"Header Length: 20 bytes",
				"Flags: ACK|PSH",
				"Window: 8192",
				"=== Payload ===",
				"Length: 37 bytes",
				"Content: \"GET / HTTP/1.1\\r\\nHost: example.com\\r\\n\\r\\n\"",
				"=== Analysis ===",
				"Type: TCP Data Packet",
				"Service: HTTP",
			},
		},
		{
			name: "SYN packet",
			packet: &NetworkPacket{
				IPv4Header: IPv4Header{
					Version:       4,
					HeaderLength:  5,
					TotalLength:   40,
					Protocol:      6,
					SourceIP:      0xC0A80101,
					DestinationIP: 0xC0A80102,
				},
				TCPHeader: TCPHeader{
					SourcePort:      12345,
					DestinationPort: 22,
					DataOffset:      5,
					Flags: TCPFlags{
						URG: 0,
						ACK: 0,
						PSH: 0,
						RST: 0,
						SYN: 1,
						FIN: 0,
					},
					Window: 8192,
				},
				Payload: []byte{},
			},
			contains: []string{
				"Type: TCP Connection Establishment (SYN)",
				"Service: SSH",
			},
		},
		{
			name: "SYN+ACK packet",
			packet: &NetworkPacket{
				IPv4Header: IPv4Header{
					Version:       4,
					HeaderLength:  5,
					TotalLength:   40,
					Protocol:      6,
					SourceIP:      0xC0A80101,
					DestinationIP: 0xC0A80102,
				},
				TCPHeader: TCPHeader{
					SourcePort:      22,
					DestinationPort: 12345,
					DataOffset:      5,
					Flags: TCPFlags{
						URG: 0,
						ACK: 1,
						PSH: 0,
						RST: 0,
						SYN: 1,
						FIN: 0,
					},
					Window: 8192,
				},
				Payload: []byte{},
			},
			contains: []string{
				"Type: TCP Connection Established (SYN+ACK)",
				"Service: Unknown (port 12345)",
			},
		},
		{
			name: "FIN packet",
			packet: &NetworkPacket{
				IPv4Header: IPv4Header{
					Version:       4,
					HeaderLength:  5,
					TotalLength:   40,
					Protocol:      6,
					SourceIP:      0xC0A80101,
					DestinationIP: 0xC0A80102,
				},
				TCPHeader: TCPHeader{
					SourcePort:      12345,
					DestinationPort: 80,
					DataOffset:      5,
					Flags: TCPFlags{
						URG: 0,
						ACK: 0,
						PSH: 0,
						RST: 0,
						SYN: 0,
						FIN: 1,
					},
					Window: 8192,
				},
				Payload: []byte{},
			},
			contains: []string{
				"Type: TCP Connection Termination (FIN/RST)",
				"Service: HTTP",
			},
		},
		{
			name: "Unknown service",
			packet: &NetworkPacket{
				IPv4Header: IPv4Header{
					Version:       4,
					HeaderLength:  5,
					TotalLength:   40,
					Protocol:      6,
					SourceIP:      0xC0A80101,
					DestinationIP: 0xC0A80102,
				},
				TCPHeader: TCPHeader{
					SourcePort:      12345,
					DestinationPort: 9999, // Unknown port
					DataOffset:      5,
					Flags: TCPFlags{
						URG: 0,
						ACK: 1,
						PSH: 1,
						RST: 0,
						SYN: 0,
						FIN: 0,
					},
					Window: 8192,
				},
				Payload: []byte("data"),
			},
			contains: []string{
				"Service: Unknown (port 9999)",
			},
		},
		{
			name: "No payload",
			packet: &NetworkPacket{
				IPv4Header: IPv4Header{
					Version:       4,
					HeaderLength:  5,
					TotalLength:   40,
					Protocol:      6,
					SourceIP:      0xC0A80101,
					DestinationIP: 0xC0A80102,
				},
				TCPHeader: TCPHeader{
					SourcePort:      12345,
					DestinationPort: 80,
					DataOffset:      5,
					Flags: TCPFlags{
						URG: 0,
						ACK: 1,
						PSH: 0,
						RST: 0,
						SYN: 0,
						FIN: 0,
					},
					Window: 8192,
				},
				Payload: []byte{},
			},
			contains: []string{
				"No payload",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatNetworkPacketInfo(tt.packet)
			for _, substr := range tt.contains {
				if !containsSubstring(result, substr) {
					t.Errorf("Expected result to contain %q, but it doesn't. Result:\n%s", substr, result)
				}
			}
		})
	}
}

func TestFormatIPAddress(t *testing.T) {
	tests := []struct {
		name string
		ip   uint32
		want string
	}{
		{"Zero IP", 0, "0.0.0.0"},
		{"Localhost", 0x7F000001, "127.0.0.1"},
		{"Private IP", 0xC0A80101, "192.168.1.1"},
		{"Broadcast", 0xFFFFFFFF, "255.255.255.255"},
		{"Random IP", 0x12345678, "18.52.86.120"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatIPAddress(tt.ip); got != tt.want {
				t.Errorf("FormatIPAddress(%d) = %v, want %v", tt.ip, got, tt.want)
			}
		})
	}
}

func TestParseIPAddress(t *testing.T) {
	tests := []struct {
		name    string
		ipStr   string
		want    uint32
		wantErr bool
	}{
		{"Zero IP", "0.0.0.0", 0, false},
		{"Localhost", "127.0.0.1", 0x7F000001, false},
		{"Private IP", "192.168.1.1", 0xC0A80101, false},
		{"Broadcast", "255.255.255.255", 0xFFFFFFFF, false},
		{"Random IP", "18.52.86.120", 0x12345678, false},
		{"Invalid IP", "256.1.1.1", 0, true},
		{"Invalid format", "192.168.1", 0, true},
		{"Invalid characters", "192.168.1.x", 0, true},
		{"Empty string", "", 0, true},
		{"IPv6 address", "::1", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseIPAddress(tt.ipStr)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseIPAddress(%q) error = %v, wantErr %v", tt.ipStr, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ParseIPAddress(%q) = %v, want %v", tt.ipStr, got, tt.want)
			}
		})
	}
}

func TestCreateHTTPPacket(t *testing.T) {
	tests := []struct {
		name     string
		srcIP    string
		dstIP    string
		srcPort  uint16
		dstPort  uint16
		method   string
		path     string
		host     string
		wantErr  bool
		validate func(*testing.T, *NetworkPacket)
	}{
		{
			name:    "Valid HTTP GET request",
			srcIP:   "192.168.1.1",
			dstIP:   "192.168.1.2",
			srcPort: 12345,
			dstPort: 80,
			method:  "GET",
			path:    "/index.html",
			host:    "example.com",
			wantErr: false,
			validate: func(t *testing.T, packet *NetworkPacket) {
				if packet == nil {
					t.Error("Expected NetworkPacket, got nil")
					return
				}
				// Проверяем IP заголовки
				if packet.IPv4Header.SourceIP != 0xC0A80101 {
					t.Errorf("Expected source IP 0xC0A80101, got 0x%08X", packet.IPv4Header.SourceIP)
				}
				if packet.IPv4Header.DestinationIP != 0xC0A80102 {
					t.Errorf("Expected destination IP 0xC0A80102, got 0x%08X", packet.IPv4Header.DestinationIP)
				}
				if packet.IPv4Header.Protocol != 6 {
					t.Errorf("Expected protocol 6 (TCP), got %d", packet.IPv4Header.Protocol)
				}

				// Проверяем TCP заголовки
				if packet.TCPHeader.SourcePort != 12345 {
					t.Errorf("Expected source port 12345, got %d", packet.TCPHeader.SourcePort)
				}
				if packet.TCPHeader.DestinationPort != 80 {
					t.Errorf("Expected destination port 80, got %d", packet.TCPHeader.DestinationPort)
				}
				if packet.TCPHeader.Flags.PSH != 1 {
					t.Errorf("Expected PSH flag 1, got %d", packet.TCPHeader.Flags.PSH)
				}

				// Проверяем payload
				expectedPayload := "GET /index.html HTTP/1.1\r\nHost: example.com\r\n\r\n"
				if string(packet.Payload) != expectedPayload {
					t.Errorf("Expected payload %q, got %q", expectedPayload, string(packet.Payload))
				}
			},
		},
		{
			name:    "HTTPS request",
			srcIP:   "10.0.0.1",
			dstIP:   "10.0.0.2",
			srcPort: 54321,
			dstPort: 443,
			method:  "POST",
			path:    "/api/data",
			host:    "api.example.com",
			wantErr: false,
			validate: func(t *testing.T, packet *NetworkPacket) {
				if packet == nil {
					t.Error("Expected NetworkPacket, got nil")
					return
				}
				if packet.TCPHeader.DestinationPort != 443 {
					t.Errorf("Expected destination port 443, got %d", packet.TCPHeader.DestinationPort)
				}
			},
		},
		{
			name:    "Invalid source IP",
			srcIP:   "256.1.1.1", // Invalid
			dstIP:   "192.168.1.2",
			srcPort: 12345,
			dstPort: 80,
			method:  "GET",
			path:    "/",
			host:    "example.com",
			wantErr: true,
		},
		{
			name:    "Invalid destination IP",
			srcIP:   "192.168.1.1",
			dstIP:   "invalid", // Invalid
			srcPort: 12345,
			dstPort: 80,
			method:  "GET",
			path:    "/",
			host:    "example.com",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CreateHTTPPacket(tt.srcIP, tt.dstIP, tt.srcPort, tt.dstPort, tt.method, tt.path, tt.host)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateHTTPPacket() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.validate != nil {
				tt.validate(t, got)
			}
		})
	}
}

func TestCreateTCPSynPacket(t *testing.T) {
	tests := []struct {
		name     string
		srcIP    string
		dstIP    string
		srcPort  uint16
		dstPort  uint16
		wantErr  bool
		validate func(*testing.T, *NetworkPacket)
	}{
		{
			name:    "Valid TCP SYN packet",
			srcIP:   "192.168.1.1",
			dstIP:   "192.168.1.2",
			srcPort: 12345,
			dstPort: 80,
			wantErr: false,
			validate: func(t *testing.T, packet *NetworkPacket) {
				if packet == nil {
					t.Error("Expected NetworkPacket, got nil")
					return
				}
				// Проверяем IP заголовки
				if packet.IPv4Header.SourceIP != 0xC0A80101 {
					t.Errorf("Expected source IP 0xC0A80101, got 0x%08X", packet.IPv4Header.SourceIP)
				}
				if packet.IPv4Header.DestinationIP != 0xC0A80102 {
					t.Errorf("Expected destination IP 0xC0A80102, got 0x%08X", packet.IPv4Header.DestinationIP)
				}
				if packet.IPv4Header.Protocol != 6 {
					t.Errorf("Expected protocol 6 (TCP), got %d", packet.IPv4Header.Protocol)
				}
				if packet.IPv4Header.TotalLength != 40 {
					t.Errorf("Expected total length 40, got %d", packet.IPv4Header.TotalLength)
				}

				// Проверяем TCP заголовки
				if packet.TCPHeader.SourcePort != 12345 {
					t.Errorf("Expected source port 12345, got %d", packet.TCPHeader.SourcePort)
				}
				if packet.TCPHeader.DestinationPort != 80 {
					t.Errorf("Expected destination port 80, got %d", packet.TCPHeader.DestinationPort)
				}
				if packet.TCPHeader.Flags.SYN != 1 {
					t.Errorf("Expected SYN flag 1, got %d", packet.TCPHeader.Flags.SYN)
				}
				if packet.TCPHeader.Flags.ACK != 0 {
					t.Errorf("Expected ACK flag 0, got %d", packet.TCPHeader.Flags.ACK)
				}

				// Проверяем, что payload пуст
				if len(packet.Payload) != 0 {
					t.Errorf("Expected empty payload, got %d bytes", len(packet.Payload))
				}
			},
		},
		{
			name:    "SSH SYN packet",
			srcIP:   "10.0.0.1",
			dstIP:   "10.0.0.2",
			srcPort: 54321,
			dstPort: 22,
			wantErr: false,
			validate: func(t *testing.T, packet *NetworkPacket) {
				if packet == nil {
					t.Error("Expected NetworkPacket, got nil")
					return
				}
				if packet.TCPHeader.DestinationPort != 22 {
					t.Errorf("Expected destination port 22, got %d", packet.TCPHeader.DestinationPort)
				}
			},
		},
		{
			name:    "Invalid source IP",
			srcIP:   "256.1.1.1", // Invalid
			dstIP:   "192.168.1.2",
			srcPort: 12345,
			dstPort: 80,
			wantErr: true,
		},
		{
			name:    "Invalid destination IP",
			srcIP:   "192.168.1.1",
			dstIP:   "invalid", // Invalid
			srcPort: 12345,
			dstPort: 80,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CreateTCPSynPacket(tt.srcIP, tt.dstIP, tt.srcPort, tt.dstPort)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateTCPSynPacket() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.validate != nil {
				tt.validate(t, got)
			}
		})
	}
}
