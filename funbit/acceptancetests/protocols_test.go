package acceptancetests

import (
	"testing"

	"github.com/funvibe/funbit/internal/protocols"
)

// TestProtocolsIPv4Header тестирование функциональности IPv4 протокола
func TestProtocolsIPv4Header(t *testing.T) {
	// Создаем IPv4 заголовок
	header := protocols.IPv4Header{
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
		SourceIP:       0xC0A80001, // 192.168.0.1
		DestinationIP:  0x08080808, // 8.8.8.8
	}

	// Тестируем построение
	bitstring, err := protocols.BuildIPv4Header(header)
	if err != nil {
		t.Fatalf("Expected to build IPv4 header, got error: %v", err)
	}

	// Тестируем парсинг
	parsedHeader, err := protocols.ParseIPv4Header(bitstring)
	if err != nil {
		t.Fatalf("Expected to parse IPv4 header, got error: %v", err)
	}

	// Проверяем значения
	if parsedHeader.Version != 4 {
		t.Errorf("Expected version 4, got %d", parsedHeader.Version)
	}
	if parsedHeader.HeaderLength != 5 {
		t.Errorf("Expected header length 5, got %d", parsedHeader.HeaderLength)
	}
	if parsedHeader.Protocol != 6 {
		t.Errorf("Expected protocol 6, got %d", parsedHeader.Protocol)
	}
	if parsedHeader.SourceIP != 0xC0A80001 {
		t.Errorf("Expected source IP 0xC0A80001, got 0x%08X", parsedHeader.SourceIP)
	}
	if parsedHeader.DestinationIP != 0x08080808 {
		t.Errorf("Expected destination IP 0x08080808, got 0x%08X", parsedHeader.DestinationIP)
	}

	// Тестируем валидацию
	if err := protocols.ValidateIPv4Header(parsedHeader); err != nil {
		t.Errorf("Expected valid header, got validation error: %v", err)
	}

	// Тестируем вспомогательные функции
	headerLength := protocols.GetIPv4HeaderLength(parsedHeader)
	if headerLength != 20 {
		t.Errorf("Expected header length 20 bytes, got %d", headerLength)
	}

	payloadLength := protocols.GetIPv4PayloadLength(parsedHeader)
	if payloadLength != 20 {
		t.Errorf("Expected payload length 20 bytes, got %d", payloadLength)
	}
}

// TestProtocolsTCPFlags тестирование функциональности TCP флагов
func TestProtocolsTCPFlags(t *testing.T) {
	// Создаем TCP флаги
	flags := protocols.TCPFlags{
		Reserved: 0,
		URG:      0,
		ACK:      1,
		PSH:      1,
		RST:      1,
		SYN:      0,
		FIN:      0,
	}

	// Тестируем построение
	bitstring, err := protocols.BuildTCPFlags(flags)
	if err != nil {
		t.Fatalf("Expected to build TCP flags, got error: %v", err)
	}

	// Тестируем парсинг
	parsedFlags, err := protocols.ParseTCPFlags(bitstring)
	if err != nil {
		t.Fatalf("Expected to parse TCP flags, got error: %v", err)
	}

	// Проверяем значения
	if parsedFlags.Reserved != 0 {
		t.Errorf("Expected reserved 0, got %d", parsedFlags.Reserved)
	}
	if parsedFlags.ACK != 1 {
		t.Errorf("Expected ACK 1, got %d", parsedFlags.ACK)
	}
	if parsedFlags.PSH != 1 {
		t.Errorf("Expected PSH 1, got %d", parsedFlags.PSH)
	}
	if parsedFlags.RST != 1 {
		t.Errorf("Expected RST 1, got %d", parsedFlags.RST)
	}
	if parsedFlags.SYN != 0 {
		t.Errorf("Expected SYN 0, got %d", parsedFlags.SYN)
	}
	if parsedFlags.FIN != 0 {
		t.Errorf("Expected FIN 0, got %d", parsedFlags.FIN)
	}

	// Тестируем валидацию
	if err := protocols.ValidateTCPFlags(parsedFlags); err != nil {
		t.Errorf("Expected valid flags, got validation error: %v", err)
	}

	// Тестируем вспомогательные функции
	flagsString := protocols.GetTCPFlagsString(*parsedFlags)
	expectedString := "ACK|PSH|RST"
	if flagsString != expectedString {
		t.Errorf("Expected flags string '%s', got '%s'", expectedString, flagsString)
	}

	// Тестируем функции анализа
	if protocols.IsTCPConnectionEstablishment(*parsedFlags) {
		t.Errorf("Expected not to be connection establishment")
	}
	if protocols.IsTCPConnectionEstablished(*parsedFlags) {
		t.Errorf("Expected not to be connection established")
	}
	if !protocols.IsTCPConnectionTermination(*parsedFlags) {
		t.Errorf("Expected to be connection termination")
	}
	if !protocols.IsTCPDataPacket(*parsedFlags) {
		t.Errorf("Expected to be data packet with ACK+PSH flags")
	}
}

// TestProtocolsTCPHeader тестирование функциональности TCP заголовка
func TestProtocolsTCPHeader(t *testing.T) {
	// Создаем TCP заголовок
	header := protocols.TCPHeader{
		SourcePort:      12345,
		DestinationPort: 80,
		SequenceNumber:  1000,
		Acknowledgment:  2000,
		DataOffset:      5,
		Reserved:        0,
		Flags: protocols.TCPFlags{
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
	}

	// Тестируем построение
	bitstring, err := protocols.BuildTCPHeader(header)
	if err != nil {
		t.Fatalf("Expected to build TCP header, got error: %v", err)
	}

	// Тестируем парсинг
	parsedHeader, err := protocols.ParseTCPHeader(bitstring)
	if err != nil {
		t.Fatalf("Expected to parse TCP header, got error: %v", err)
	}

	// Проверяем значения
	if parsedHeader.SourcePort != 12345 {
		t.Errorf("Expected source port 12345, got %d", parsedHeader.SourcePort)
	}
	if parsedHeader.DestinationPort != 80 {
		t.Errorf("Expected destination port 80, got %d", parsedHeader.DestinationPort)
	}
	if parsedHeader.SequenceNumber != 1000 {
		t.Errorf("Expected sequence number 1000, got %d", parsedHeader.SequenceNumber)
	}
	if parsedHeader.Acknowledgment != 2000 {
		t.Errorf("Expected acknowledgment 2000, got %d", parsedHeader.Acknowledgment)
	}
	if parsedHeader.DataOffset != 5 {
		t.Errorf("Expected data offset 5, got %d", parsedHeader.DataOffset)
	}

	// Тестируем валидацию
	if err := protocols.ValidateTCPHeader(parsedHeader); err != nil {
		t.Errorf("Expected valid header, got validation error: %v", err)
	}

	// Тестируем вспомогательные функции
	headerLength := protocols.GetTCPHeaderLength(parsedHeader)
	if headerLength != 20 {
		t.Errorf("Expected header length 20 bytes, got %d", headerLength)
	}
}

// TestProtocolsPNGHeader тестирование функциональности PNG протокола
func TestProtocolsPNGHeader(t *testing.T) {
	// Создаем IHDR чанк
	ihdr := protocols.PNGIHDRChunk{
		Width:       100,
		Height:      50,
		BitDepth:    8,
		ColorType:   2, // RGB
		Compression: 0,
		Filter:      0,
		Interlace:   0,
	}

	// Тестируем построение IHDR чанка
	chunk, err := protocols.BuildPNGIHDRChunk(ihdr)
	if err != nil {
		t.Fatalf("Expected to build PNG IHDR chunk, got error: %v", err)
	}

	// Создаем PNG заголовок
	pngHeader := protocols.PNGHeader{
		Signature: protocols.PNGSignature,
		Chunks:    []protocols.PNGChunk{chunk},
	}

	// Тестируем построение PNG заголовка
	bitstring, err := protocols.BuildPNGHeader(pngHeader)
	if err != nil {
		t.Fatalf("Expected to build PNG header, got error: %v", err)
	}

	// Тестируем парсинг PNG заголовка
	parsedHeader, err := protocols.ParsePNGHeader(bitstring)
	if err != nil {
		t.Fatalf("Expected to parse PNG header, got error: %v", err)
	}

	// Проверяем сигнатуру
	if len(parsedHeader.Signature) != len(protocols.PNGSignature) {
		t.Errorf("Expected signature length %d, got %d", len(protocols.PNGSignature), len(parsedHeader.Signature))
	}

	// Проверяем наличие чанков
	if len(parsedHeader.Chunks) == 0 {
		t.Errorf("Expected at least one chunk, got none")
	}

	// Проверяем тип первого чанка
	if string(parsedHeader.Chunks[0].Type) != "IHDR" {
		t.Errorf("Expected first chunk type 'IHDR', got '%s'", string(parsedHeader.Chunks[0].Type))
	}

	// Тестируем валидацию PNG заголовка
	if err := protocols.ValidatePNGHeader(parsedHeader); err != nil {
		t.Errorf("Expected valid PNG header, got validation error: %v", err)
	}

	// Тестируем парсинг IHDR чанка
	parsedIHDR, err := protocols.ParsePNGIHDRChunk(parsedHeader.Chunks[0].Data)
	if err != nil {
		t.Fatalf("Expected to parse IHDR chunk, got error: %v", err)
	}

	// Проверяем значения IHDR
	if parsedIHDR.Width != 100 {
		t.Errorf("Expected width 100, got %d", parsedIHDR.Width)
	}
	if parsedIHDR.Height != 50 {
		t.Errorf("Expected height 50, got %d", parsedIHDR.Height)
	}
	if parsedIHDR.BitDepth != 8 {
		t.Errorf("Expected bit depth 8, got %d", parsedIHDR.BitDepth)
	}
	if parsedIHDR.ColorType != 2 {
		t.Errorf("Expected color type 2, got %d", parsedIHDR.ColorType)
	}

	// Тестируем валидацию IHDR чанка
	if err := protocols.ValidatePNGIHDRChunk(parsedIHDR); err != nil {
		t.Errorf("Expected valid IHDR chunk, got validation error: %v", err)
	}

	// Тестируем вспомогательные функции
	colorTypeName := protocols.GetPNGColorTypeName(parsedIHDR.ColorType)
	if colorTypeName != "RGB" {
		t.Errorf("Expected color type name 'RGB', got '%s'", colorTypeName)
	}

	interlaceName := protocols.GetPNGInterlaceMethodName(parsedIHDR.Interlace)
	if interlaceName != "None" {
		t.Errorf("Expected interlace method name 'None', got '%s'", interlaceName)
	}

	// Тестируем форматирование информации
	info := protocols.FormatPNGIHDRInfo(parsedIHDR)
	if info == "" {
		t.Errorf("Expected non-empty IHDR info string")
	}
}

// TestProtocolsComplexNetworkPacket тестирование комплексного сетевого пакета
func TestProtocolsComplexNetworkPacket(t *testing.T) {
	// Создаем сетевой пакет
	packet := protocols.NetworkPacket{
		IPv4Header: protocols.IPv4Header{
			Version:        4,
			HeaderLength:   5,
			ServiceType:    0,
			TotalLength:    49, // 20 (IP) + 20 (TCP) + 9 (payload)
			Identification: 54321,
			Flags:          2,
			FragmentOffset: 0,
			TTL:            64,
			Protocol:       6, // TCP
			Checksum:       0,
			SourceIP:       0xC0A80001, // 192.168.0.1
			DestinationIP:  0x08080808, // 8.8.8.8
		},
		TCPHeader: protocols.TCPHeader{
			SourcePort:      12345,
			DestinationPort: 80,
			SequenceNumber:  1000,
			Acknowledgment:  2000,
			DataOffset:      5,
			Reserved:        0,
			Flags: protocols.TCPFlags{
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
		Payload: []byte("TEST DATA"),
	}

	// Тестируем построение пакета
	bitstring, err := protocols.BuildNetworkPacket(packet)
	if err != nil {
		t.Fatalf("Expected to build network packet, got error: %v", err)
	}

	// Тестируем парсинг пакета
	parsedPacket, err := protocols.ParseNetworkPacket(bitstring)
	if err != nil {
		t.Fatalf("Expected to parse network packet, got error: %v", err)
	}

	// Проверяем IPv4 заголовок
	if parsedPacket.IPv4Header.Version != 4 {
		t.Errorf("Expected IP version 4, got %d", parsedPacket.IPv4Header.Version)
	}
	if parsedPacket.IPv4Header.Protocol != 6 {
		t.Errorf("Expected IP protocol 6, got %d", parsedPacket.IPv4Header.Protocol)
	}
	if parsedPacket.IPv4Header.SourceIP != 0xC0A80001 {
		t.Errorf("Expected source IP 0xC0A80001, got 0x%08X", parsedPacket.IPv4Header.SourceIP)
	}

	// Проверяем TCP заголовок
	if parsedPacket.TCPHeader.SourcePort != 12345 {
		t.Errorf("Expected source port 12345, got %d", parsedPacket.TCPHeader.SourcePort)
	}
	if parsedPacket.TCPHeader.DestinationPort != 80 {
		t.Errorf("Expected destination port 80, got %d", parsedPacket.TCPHeader.DestinationPort)
	}
	if parsedPacket.TCPHeader.Flags.ACK != 1 {
		t.Errorf("Expected ACK flag 1, got %d", parsedPacket.TCPHeader.Flags.ACK)
	}

	// Проверяем payload
	if string(parsedPacket.Payload) != "TEST DATA" {
		t.Errorf("Expected payload 'TEST DATA', got '%s'", string(parsedPacket.Payload))
	}

	// Тестируем валидацию пакета
	if err := protocols.ValidateNetworkPacket(parsedPacket); err != nil {
		t.Errorf("Expected valid packet, got validation error: %v", err)
	}

	// Тестируем форматирование информации
	info := protocols.FormatNetworkPacketInfo(parsedPacket)
	if info == "" {
		t.Errorf("Expected non-empty packet info string")
	}
}

// TestProtocolsUtilityFunctions тестирование вспомогательных функций
func TestProtocolsUtilityFunctions(t *testing.T) {
	// Тестируем форматирование и парсинг IP-адресов
	testIP := "192.168.1.1"
	ipUint, err := protocols.ParseIPAddress(testIP)
	if err != nil {
		t.Fatalf("Expected to parse IP address, got error: %v", err)
	}

	formattedIP := protocols.FormatIPAddress(ipUint)
	if formattedIP != testIP {
		t.Errorf("Expected IP '%s', got '%s'", testIP, formattedIP)
	}

	// Тестируем создание HTTP пакета
	httpPacket, err := protocols.CreateHTTPPacket("192.168.1.1", "8.8.8.8", 12345, 80, "GET", "/", "example.com")
	if err != nil {
		t.Fatalf("Expected to create HTTP packet, got error: %v", err)
	}

	if err := protocols.ValidateNetworkPacket(httpPacket); err != nil {
		t.Errorf("Expected valid HTTP packet, got validation error: %v", err)
	}

	// Тестируем создание TCP SYN пакета
	synPacket, err := protocols.CreateTCPSynPacket("192.168.1.1", "8.8.8.8", 12345, 80)
	if err != nil {
		t.Fatalf("Expected to create TCP SYN packet, got error: %v", err)
	}

	if err := protocols.ValidateNetworkPacket(synPacket); err != nil {
		t.Errorf("Expected valid TCP SYN packet, got validation error: %v", err)
	}

	// Проверяем, что это действительно SYN пакет
	if !protocols.IsTCPConnectionEstablishment(synPacket.TCPHeader.Flags) {
		t.Errorf("Expected TCP SYN packet to be connection establishment")
	}
}
