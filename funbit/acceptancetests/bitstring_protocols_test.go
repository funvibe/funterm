package acceptancetests

import (
	"testing"

	"github.com/funvibe/funbit/internal/bitstring"
	"github.com/funvibe/funbit/internal/builder"
	"github.com/funvibe/funbit/internal/matcher"
)

// TestBitstringIPv4Header тестирование парсинга IPv4 заголовков
func TestBitstringIPv4Header(t *testing.T) {
	// Создаем IPv4 заголовок (20 байт = 160 бит)
	// Структура:
	// Version (4 bits): 4
	// Header Length (4 bits): 5 (5 * 4 = 20 bytes)
	// Service Type (8 bits): 0
	// Total Length (16 bits): 20
	// Identification (16 bits): 12345
	// Flags (3 bits): 2 (Don't Fragment)
	// Fragment Offset (13 bits): 0
	// TTL (8 bits): 64
	// Protocol (8 bits): 6 (TCP)
	// Checksum (16 bits): 0
	// Source IP (32 bits): 0xC0A80001 (192.168.0.1)
	// Destination IP (32 bits): 0x08080808 (8.8.8.8)

	version := uint(4)
	headerLength := uint(5)
	serviceType := uint(0)
	totalLength := uint(20)
	identification := uint(12345)
	flags := uint(2) // Don't Fragment
	fragmentOffset := uint(0)
	ttl := uint(64)
	protocol := uint(6) // TCP
	checksum := uint(0)
	srcIP := uint(0xC0A80001) // 192.168.0.1
	dstIP := uint(0x08080808) // 8.8.8.8

	ipHeader, err := builder.NewBuilder().
		AddInteger(version, bitstring.WithSize(4)).
		AddInteger(headerLength, bitstring.WithSize(4)).
		AddInteger(serviceType, bitstring.WithSize(8)).
		AddInteger(totalLength, bitstring.WithSize(16), bitstring.WithEndianness("big")).
		AddInteger(identification, bitstring.WithSize(16), bitstring.WithEndianness("big")).
		AddInteger(flags, bitstring.WithSize(3)).
		AddInteger(fragmentOffset, bitstring.WithSize(13)).
		AddInteger(ttl, bitstring.WithSize(8)).
		AddInteger(protocol, bitstring.WithSize(8)).
		AddInteger(checksum, bitstring.WithSize(16), bitstring.WithEndianness("big")).
		AddInteger(srcIP, bitstring.WithSize(32), bitstring.WithEndianness("big")).
		AddInteger(dstIP, bitstring.WithSize(32), bitstring.WithEndianness("big")).
		Build()

	if err != nil {
		t.Fatalf("Expected to build IPv4 header, got error: %v", err)
	}

	// Теперь парсим заголовок
	var parsedVersion, parsedHeaderLength, parsedServiceType uint
	var parsedTotalLength, parsedIdentification uint
	var parsedFlags, parsedFragmentOffset uint
	var parsedTTL, parsedProtocol, parsedChecksum uint
	var parsedSrcIP, parsedDstIP uint

	_, err = matcher.NewMatcher().
		Integer(&parsedVersion, bitstring.WithSize(4)).
		Integer(&parsedHeaderLength, bitstring.WithSize(4)).
		Integer(&parsedServiceType, bitstring.WithSize(8)).
		Integer(&parsedTotalLength, bitstring.WithSize(16), bitstring.WithEndianness("big")).
		Integer(&parsedIdentification, bitstring.WithSize(16), bitstring.WithEndianness("big")).
		Integer(&parsedFlags, bitstring.WithSize(3)).
		Integer(&parsedFragmentOffset, bitstring.WithSize(13)).
		Integer(&parsedTTL, bitstring.WithSize(8)).
		Integer(&parsedProtocol, bitstring.WithSize(8)).
		Integer(&parsedChecksum, bitstring.WithSize(16), bitstring.WithEndianness("big")).
		Integer(&parsedSrcIP, bitstring.WithSize(32), bitstring.WithEndianness("big")).
		Integer(&parsedDstIP, bitstring.WithSize(32), bitstring.WithEndianness("big")).
		Match(ipHeader)

	if err != nil {
		t.Fatalf("Expected to match IPv4 header, got error: %v", err)
	}

	// Проверяем значения
	if parsedVersion != 4 {
		t.Errorf("Expected IP version 4, got %d", parsedVersion)
	}
	if parsedHeaderLength != 5 {
		t.Errorf("Expected header length 5, got %d", parsedHeaderLength)
	}
	if parsedServiceType != 0 {
		t.Errorf("Expected service type 0, got %d", parsedServiceType)
	}
	if parsedTotalLength != 20 {
		t.Errorf("Expected total length 20, got %d", parsedTotalLength)
	}
	if parsedIdentification != 12345 {
		t.Errorf("Expected identification 12345, got %d", parsedIdentification)
	}
	if parsedFlags != 2 {
		t.Errorf("Expected flags 2, got %d", parsedFlags)
	}
	if parsedFragmentOffset != 0 {
		t.Errorf("Expected fragment offset 0, got %d", parsedFragmentOffset)
	}
	if parsedTTL != 64 {
		t.Errorf("Expected TTL 64, got %d", parsedTTL)
	}
	if parsedProtocol != 6 {
		t.Errorf("Expected protocol 6 (TCP), got %d", parsedProtocol)
	}
	if parsedChecksum != 0 {
		t.Errorf("Expected checksum 0, got %d", parsedChecksum)
	}
	if parsedSrcIP != 0xC0A80001 {
		t.Errorf("Expected source IP 0xC0A80001, got 0x%08X", parsedSrcIP)
	}
	if parsedDstIP != 0x08080808 {
		t.Errorf("Expected destination IP 0x08080808, got 0x%08X", parsedDstIP)
	}
}

// TestBitstringTCPFlags тестирование TCP флагов
func TestBitstringTCPFlags(t *testing.T) {
	// Создаем TCP флаги (1 байт = 8 бит)
	// Структура:
	// Reserved (2 bits): 0
	// URG (1 bit): 0
	// ACK (1 bit): 1
	// PSH (1 bit): 1
	// RST (1 bit): 1
	// SYN (1 bit): 0
	// FIN (1 bit): 0
	// Результат: 00 0 1 1 1 0 0 = 00111000 = 0x38

	reserved := uint(0)
	urg := uint(0)
	ack := uint(1)
	psh := uint(1)
	rst := uint(1)
	syn := uint(0)
	fin := uint(0)

	tcpFlags, err := builder.NewBuilder().
		AddInteger(reserved, bitstring.WithSize(2)).
		AddInteger(urg, bitstring.WithSize(1)).
		AddInteger(ack, bitstring.WithSize(1)).
		AddInteger(psh, bitstring.WithSize(1)).
		AddInteger(rst, bitstring.WithSize(1)).
		AddInteger(syn, bitstring.WithSize(1)).
		AddInteger(fin, bitstring.WithSize(1)).
		Build()

	if err != nil {
		t.Fatalf("Expected to build TCP flags, got error: %v", err)
	}

	// Теперь парсим флаги
	var parsedReserved, parsedUrg, parsedAck, parsedPsh uint
	var parsedRst, parsedSyn, parsedFin uint

	_, err = matcher.NewMatcher().
		Integer(&parsedReserved, bitstring.WithSize(2)).
		Integer(&parsedUrg, bitstring.WithSize(1)).
		Integer(&parsedAck, bitstring.WithSize(1)).
		Integer(&parsedPsh, bitstring.WithSize(1)).
		Integer(&parsedRst, bitstring.WithSize(1)).
		Integer(&parsedSyn, bitstring.WithSize(1)).
		Integer(&parsedFin, bitstring.WithSize(1)).
		Match(tcpFlags)

	if err != nil {
		t.Fatalf("Expected to match TCP flags, got error: %v", err)
	}

	// Проверяем значения
	if parsedReserved != 0 {
		t.Errorf("Expected reserved 0, got %d", parsedReserved)
	}
	if parsedUrg != 0 {
		t.Errorf("Expected URG 0, got %d", parsedUrg)
	}
	if parsedAck != 1 {
		t.Errorf("Expected ACK 1, got %d", parsedAck)
	}
	if parsedPsh != 1 {
		t.Errorf("Expected PSH 1, got %d", parsedPsh)
	}
	if parsedRst != 1 {
		t.Errorf("Expected RST 1, got %d", parsedRst)
	}
	if parsedSyn != 0 {
		t.Errorf("Expected SYN 0, got %d", parsedSyn)
	}
	if parsedFin != 0 {
		t.Errorf("Expected FIN 0, got %d", parsedFin)
	}
}

// TestBitstringPNGHeader тестирование парсинга PNG заголовка
func TestBitstringPNGHeader(t *testing.T) {
	// PNG signature: 137 80 78 71 13 10 26 10
	pngSignature := []byte{137, 80, 78, 71, 13, 10, 26, 10}

	// IHDR chunk data
	chunkLength := uint(13) // IHDR всегда 13 байт
	chunkType := []byte("IHDR")
	width := uint(100)
	height := uint(50)
	bitDepth := uint(8)
	colorType := uint(2) // RGB
	compression := uint(0)
	filter := uint(0)
	interlace := uint(0)

	// Создаем полный PNG заголовок
	pngHeader, err := builder.NewBuilder().
		AddBinary(pngSignature).
		AddInteger(chunkLength, bitstring.WithSize(32), bitstring.WithEndianness("big")).
		AddBinary(chunkType).
		AddInteger(width, bitstring.WithSize(32), bitstring.WithEndianness("big")).
		AddInteger(height, bitstring.WithSize(32), bitstring.WithEndianness("big")).
		AddInteger(bitDepth, bitstring.WithSize(8)).
		AddInteger(colorType, bitstring.WithSize(8)).
		AddInteger(compression, bitstring.WithSize(8)).
		AddInteger(filter, bitstring.WithSize(8)).
		AddInteger(interlace, bitstring.WithSize(8)).
		Build()

	if err != nil {
		t.Fatalf("Expected to build PNG header, got error: %v", err)
	}

	// Теперь парсим PNG заголовок
	var parsedSignature []byte
	var parsedChunkLength, parsedWidth, parsedHeight uint
	var parsedBitDepth, parsedColorType uint
	var parsedCompression, parsedFilter, parsedInterlace uint
	var parsedChunkType []byte

	_, err = matcher.NewMatcher().
		Binary(&parsedSignature, bitstring.WithSize(8)). // 8 bytes
		Integer(&parsedChunkLength, bitstring.WithSize(32), bitstring.WithEndianness("big")).
		Binary(&parsedChunkType, bitstring.WithSize(4)). // 4 bytes
		Integer(&parsedWidth, bitstring.WithSize(32), bitstring.WithEndianness("big")).
		Integer(&parsedHeight, bitstring.WithSize(32), bitstring.WithEndianness("big")).
		Integer(&parsedBitDepth, bitstring.WithSize(8)).
		Integer(&parsedColorType, bitstring.WithSize(8)).
		Integer(&parsedCompression, bitstring.WithSize(8)).
		Integer(&parsedFilter, bitstring.WithSize(8)).
		Integer(&parsedInterlace, bitstring.WithSize(8)).
		Match(pngHeader)

	if err != nil {
		t.Fatalf("Expected to match PNG header, got error: %v", err)
	}

	// Проверяем PNG signature
	if len(parsedSignature) != 8 || !byteSlicesEqual(parsedSignature, pngSignature) {
		t.Errorf("Expected PNG signature %v, got %v", pngSignature, parsedSignature)
	}

	// Проверяем chunk type
	if string(parsedChunkType) != "IHDR" {
		t.Errorf("Expected chunk type 'IHDR', got '%s'", string(parsedChunkType))
	}

	// Проверяем IHDR данные
	if parsedChunkLength != 13 {
		t.Errorf("Expected chunk length 13, got %d", parsedChunkLength)
	}
	if parsedWidth != 100 {
		t.Errorf("Expected width 100, got %d", parsedWidth)
	}
	if parsedHeight != 50 {
		t.Errorf("Expected height 50, got %d", parsedHeight)
	}
	if parsedBitDepth != 8 {
		t.Errorf("Expected bit depth 8, got %d", parsedBitDepth)
	}
	if parsedColorType != 2 {
		t.Errorf("Expected color type 2 (RGB), got %d", parsedColorType)
	}
	if parsedCompression != 0 {
		t.Errorf("Expected compression 0, got %d", parsedCompression)
	}
	if parsedFilter != 0 {
		t.Errorf("Expected filter 0, got %d", parsedFilter)
	}
	if parsedInterlace != 0 {
		t.Errorf("Expected interlace 0, got %d", parsedInterlace)
	}
}

// Helper function to compare byte slices
func byteSlicesEqual(a, b []byte) bool {
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

// TestBitstringComplexProtocol тестирование комплексного протокола
func TestBitstringComplexProtocol(t *testing.T) {
	// Комплексный тест: IPv4 пакет с TCP сегментом
	// IPv4 header (20 bytes) + TCP header (20 bytes) + payload

	// IPv4 header
	version := uint(4)
	headerLength := uint(5)
	serviceType := uint(0)
	totalLength := uint(40) // 20 (IP) + 20 (TCP)
	identification := uint(54321)
	flags := uint(2) // Don't Fragment
	fragmentOffset := uint(0)
	ttl := uint(64)
	protocol := uint(6) // TCP
	checksum := uint(0)
	srcIP := uint(0xC0A80001) // 192.168.0.1
	dstIP := uint(0x08080808) // 8.8.8.8

	// TCP header
	srcPort := uint(12345)
	dstPort := uint(80)
	seqNum := uint(1000)
	ackNum := uint(2000)
	dataOffset := uint(5) // 5 * 4 = 20 bytes
	tcpReserved := uint(0)
	tcpFlags := uint(0x18) // PSH + ACK
	tcpWindow := uint(8192)
	tcpChecksum := uint(0)
	tcpUrgent := uint(0)

	// Payload
	payload := []byte("GET / HTTP/1.1\r\n")

	// Собираем пакет
	packet, err := builder.NewBuilder().
		// IPv4 header
		AddInteger(version, bitstring.WithSize(4)).
		AddInteger(headerLength, bitstring.WithSize(4)).
		AddInteger(serviceType, bitstring.WithSize(8)).
		AddInteger(totalLength, bitstring.WithSize(16), bitstring.WithEndianness("big")).
		AddInteger(identification, bitstring.WithSize(16), bitstring.WithEndianness("big")).
		AddInteger(flags, bitstring.WithSize(3)).
		AddInteger(fragmentOffset, bitstring.WithSize(13)).
		AddInteger(ttl, bitstring.WithSize(8)).
		AddInteger(protocol, bitstring.WithSize(8)).
		AddInteger(checksum, bitstring.WithSize(16), bitstring.WithEndianness("big")).
		AddInteger(srcIP, bitstring.WithSize(32), bitstring.WithEndianness("big")).
		AddInteger(dstIP, bitstring.WithSize(32), bitstring.WithEndianness("big")).
		// TCP header
		AddInteger(srcPort, bitstring.WithSize(16), bitstring.WithEndianness("big")).
		AddInteger(dstPort, bitstring.WithSize(16), bitstring.WithEndianness("big")).
		AddInteger(seqNum, bitstring.WithSize(32), bitstring.WithEndianness("big")).
		AddInteger(ackNum, bitstring.WithSize(32), bitstring.WithEndianness("big")).
		AddInteger(dataOffset, bitstring.WithSize(4)).
		AddInteger(tcpReserved, bitstring.WithSize(3)).
		AddInteger(tcpFlags, bitstring.WithSize(6)).
		AddInteger(tcpWindow, bitstring.WithSize(16), bitstring.WithEndianness("big")).
		AddInteger(tcpChecksum, bitstring.WithSize(16), bitstring.WithEndianness("big")).
		AddInteger(tcpUrgent, bitstring.WithSize(16), bitstring.WithEndianness("big")).
		// Payload
		AddBinary(payload).
		Build()

	if err != nil {
		t.Fatalf("Expected to build complex packet, got error: %v", err)
	}

	// Парсим пакет
	var parsedVersion, parsedHeaderLength uint
	var parsedServiceType uint
	var parsedTotalLength, parsedIdentification uint
	var parsedFlags, parsedFragmentOffset uint
	var parsedTTL, parsedProtocol, parsedChecksum uint
	var parsedSrcIP, parsedDstIP uint
	var parsedSrcPort, parsedDstPort uint
	var parsedSeqNum, parsedAckNum uint
	var parsedDataOffset, parsedTCPReserved uint
	var parsedTCPFlags, parsedTCPWindow, parsedTCPChecksum, parsedTCPUrgent uint
	var parsedPayload []byte

	_, err = matcher.NewMatcher().
		// IPv4 header
		Integer(&parsedVersion, bitstring.WithSize(4)).
		Integer(&parsedHeaderLength, bitstring.WithSize(4)).
		Integer(&parsedServiceType, bitstring.WithSize(8)).
		Integer(&parsedTotalLength, bitstring.WithSize(16), bitstring.WithEndianness("big")).
		Integer(&parsedIdentification, bitstring.WithSize(16), bitstring.WithEndianness("big")).
		Integer(&parsedFlags, bitstring.WithSize(3)).
		Integer(&parsedFragmentOffset, bitstring.WithSize(13)).
		Integer(&parsedTTL, bitstring.WithSize(8)).
		Integer(&parsedProtocol, bitstring.WithSize(8)).
		Integer(&parsedChecksum, bitstring.WithSize(16), bitstring.WithEndianness("big")).
		Integer(&parsedSrcIP, bitstring.WithSize(32), bitstring.WithEndianness("big")).
		Integer(&parsedDstIP, bitstring.WithSize(32), bitstring.WithEndianness("big")).
		// TCP header
		Integer(&parsedSrcPort, bitstring.WithSize(16), bitstring.WithEndianness("big")).
		Integer(&parsedDstPort, bitstring.WithSize(16), bitstring.WithEndianness("big")).
		Integer(&parsedSeqNum, bitstring.WithSize(32), bitstring.WithEndianness("big")).
		Integer(&parsedAckNum, bitstring.WithSize(32), bitstring.WithEndianness("big")).
		Integer(&parsedDataOffset, bitstring.WithSize(4)).
		Integer(&parsedTCPReserved, bitstring.WithSize(3)).
		Integer(&parsedTCPFlags, bitstring.WithSize(6)).
		Integer(&parsedTCPWindow, bitstring.WithSize(16), bitstring.WithEndianness("big")).
		Integer(&parsedTCPChecksum, bitstring.WithSize(16), bitstring.WithEndianness("big")).
		Integer(&parsedTCPUrgent, bitstring.WithSize(16), bitstring.WithEndianness("big")).
		// Payload
		Binary(&parsedPayload, bitstring.WithSize(uint(len("GET / HTTP/1.1\r\n")))).
		Match(packet)

	if err != nil {
		t.Fatalf("Expected to match complex packet, got error: %v", err)
	}

	// Проверяем ключевые значения
	if parsedVersion != 4 {
		t.Errorf("Expected IP version 4, got %d", parsedVersion)
	}
	if parsedProtocol != 6 {
		t.Errorf("Expected protocol 6 (TCP), got %d", parsedProtocol)
	}
	if parsedSrcPort != 12345 {
		t.Errorf("Expected source port 12345, got %d", parsedSrcPort)
	}
	if parsedDstPort != 80 {
		t.Errorf("Expected destination port 80, got %d", parsedDstPort)
	}
	if string(parsedPayload) != "GET / HTTP/1.1\r\n" {
		t.Errorf("Expected payload 'GET / HTTP/1.1\\r\\n', got '%s'", string(parsedPayload))
	}
}
