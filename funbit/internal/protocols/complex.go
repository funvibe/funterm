package protocols

import (
	"encoding/binary"
	"fmt"
	"net"
	"strings"

	"github.com/funvibe/funbit/internal/bitstring"
	"github.com/funvibe/funbit/internal/builder"
)

// NetworkPacket represents a network packet with IPv4 and TCP headers
type NetworkPacket struct {
	IPv4Header IPv4Header
	TCPHeader  TCPHeader
	Payload    []byte
}

// BuildNetworkPacket creates a network packet from a structure
func BuildNetworkPacket(packet NetworkPacket) (*bitstring.BitString, error) {
	// First create IPv4 header
	ipHeader, err := BuildIPv4Header(packet.IPv4Header)
	if err != nil {
		return nil, fmt.Errorf("failed to build IPv4 header: %v", err)
	}

	// Then create TCP header
	tcpHeader, err := BuildTCPHeader(packet.TCPHeader)
	if err != nil {
		return nil, fmt.Errorf("failed to build TCP header: %v", err)
	}

	// Assemble complete packet
	return builder.NewBuilder().
		AddBinary(ipHeader.ToBytes()).
		AddBinary(tcpHeader.ToBytes()).
		AddBinary(packet.Payload).
		Build()
}

// ParseNetworkPacket parses a network packet from a bitstring
func ParseNetworkPacket(data *bitstring.BitString) (*NetworkPacket, error) {
	var packet NetworkPacket

	// Parse IPv4 header
	ipHeader, err := ParseIPv4Header(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse IPv4 header: %v", err)
	}
	packet.IPv4Header = *ipHeader

	// Calculate offset to TCP header
	ipHeaderLength := GetIPv4HeaderLength(ipHeader)

	// Create bitstring for TCP header and payload
	remainingData := data.ToBytes()[ipHeaderLength:]
	remainingBitString := bitstring.NewBitStringFromBytes(remainingData)

	// Parse TCP header
	tcpHeader, err := ParseTCPHeader(remainingBitString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse TCP header: %v", err)
	}
	packet.TCPHeader = *tcpHeader

	// Calculate offset to payload
	tcpHeaderLength := GetTCPHeaderLength(tcpHeader)
	payloadOffset := tcpHeaderLength

	// Extract payload
	if len(remainingData) > int(payloadOffset) {
		packet.Payload = remainingData[payloadOffset:]
	} else {
		packet.Payload = []byte{}
	}

	return &packet, nil
}

// ValidateNetworkPacket performs full validation of a network packet
func ValidateNetworkPacket(packet *NetworkPacket) error {
	if packet == nil {
		return fmt.Errorf("packet is nil")
	}

	// IPv4 header validation
	if err := ValidateIPv4Header(&packet.IPv4Header); err != nil {
		return fmt.Errorf("IPv4 header validation failed: %v", err)
	}

	// TCP header validation
	if err := ValidateTCPHeader(&packet.TCPHeader); err != nil {
		return fmt.Errorf("TCP header validation failed: %v", err)
	}

	// Check protocol match
	if packet.IPv4Header.Protocol != 6 { // 6 = TCP
		return fmt.Errorf("IP protocol mismatch: expected TCP (6), got %d", packet.IPv4Header.Protocol)
	}

	// Check packet length
	expectedTotalLength := GetIPv4HeaderLength(&packet.IPv4Header) +
		GetTCPHeaderLength(&packet.TCPHeader) + uint(len(packet.Payload))

	if packet.IPv4Header.TotalLength != expectedTotalLength {
		return fmt.Errorf("packet length mismatch: IP header says %d, actual is %d",
			packet.IPv4Header.TotalLength, expectedTotalLength)
	}

	return nil
}

// FormatNetworkPacketInfo formats network packet information for output
func FormatNetworkPacketInfo(packet *NetworkPacket) string {
	var info strings.Builder

	// IPv4 information
	info.WriteString("=== IPv4 Header ===\n")
	info.WriteString(fmt.Sprintf("  Version: %d\n", packet.IPv4Header.Version))
	info.WriteString(fmt.Sprintf("  Header Length: %d bytes\n", GetIPv4HeaderLength(&packet.IPv4Header)))
	info.WriteString(fmt.Sprintf("  Total Length: %d bytes\n", packet.IPv4Header.TotalLength))
	info.WriteString(fmt.Sprintf("  Protocol: %d (TCP)\n", packet.IPv4Header.Protocol))
	info.WriteString(fmt.Sprintf("  TTL: %d\n", packet.IPv4Header.TTL))
	info.WriteString(fmt.Sprintf("  Source IP: %s\n", FormatIPAddress(uint32(packet.IPv4Header.SourceIP))))
	info.WriteString(fmt.Sprintf("  Destination IP: %s\n", FormatIPAddress(uint32(packet.IPv4Header.DestinationIP))))

	// TCP information
	info.WriteString("\n=== TCP Header ===\n")
	info.WriteString(fmt.Sprintf("  Source Port: %d\n", packet.TCPHeader.SourcePort))
	info.WriteString(fmt.Sprintf("  Destination Port: %d\n", packet.TCPHeader.DestinationPort))
	info.WriteString(fmt.Sprintf("  Sequence Number: %d\n", packet.TCPHeader.SequenceNumber))
	info.WriteString(fmt.Sprintf("  Acknowledgment: %d\n", packet.TCPHeader.Acknowledgment))
	info.WriteString(fmt.Sprintf("  Header Length: %d bytes\n", GetTCPHeaderLength(&packet.TCPHeader)))
	info.WriteString(fmt.Sprintf("  Flags: %s\n", GetTCPFlagsString(packet.TCPHeader.Flags)))
	info.WriteString(fmt.Sprintf("  Window: %d\n", packet.TCPHeader.Window))

	// Payload information
	info.WriteString("\n=== Payload ===\n")
	if len(packet.Payload) > 0 {
		// Show first 50 bytes of payload as text
		payloadText := string(packet.Payload)
		if len(payloadText) > 50 {
			payloadText = payloadText[:50] + "..."
		}
		info.WriteString(fmt.Sprintf("  Length: %d bytes\n", len(packet.Payload)))
		info.WriteString(fmt.Sprintf("  Content: %q\n", payloadText))
	} else {
		info.WriteString("  No payload\n")
	}

	// Packet type analysis
	info.WriteString("\n=== Analysis ===\n")

	// TCP flags analysis
	if IsTCPConnectionEstablishment(packet.TCPHeader.Flags) {
		info.WriteString("  Type: TCP Connection Establishment (SYN)\n")
	} else if IsTCPConnectionEstablished(packet.TCPHeader.Flags) {
		info.WriteString("  Type: TCP Connection Established (SYN+ACK)\n")
	} else if IsTCPConnectionTermination(packet.TCPHeader.Flags) {
		info.WriteString("  Type: TCP Connection Termination (FIN/RST)\n")
	} else if IsTCPDataPacket(packet.TCPHeader.Flags) {
		info.WriteString("  Type: TCP Data Packet\n")
	} else {
		info.WriteString("  Type: Unknown TCP Packet\n")
	}

	// Port analysis
	if packet.TCPHeader.DestinationPort == 80 || packet.TCPHeader.DestinationPort == 8080 {
		info.WriteString("  Service: HTTP\n")
	} else if packet.TCPHeader.DestinationPort == 443 {
		info.WriteString("  Service: HTTPS\n")
	} else if packet.TCPHeader.DestinationPort == 22 {
		info.WriteString("  Service: SSH\n")
	} else if packet.TCPHeader.DestinationPort == 21 {
		info.WriteString("  Service: FTP\n")
	} else if packet.TCPHeader.DestinationPort == 25 {
		info.WriteString("  Service: SMTP\n")
	} else {
		info.WriteString(fmt.Sprintf("  Service: Unknown (port %d)\n", packet.TCPHeader.DestinationPort))
	}

	return info.String()
}

// FormatIPAddress formats an IP address from uint32 to string
func FormatIPAddress(ip uint32) string {
	return fmt.Sprintf("%d.%d.%d.%d",
		(ip>>24)&0xFF,
		(ip>>16)&0xFF,
		(ip>>8)&0xFF,
		ip&0xFF)
}

// ParseIPAddress parses an IP address from string to uint32
func ParseIPAddress(ipStr string) (uint32, error) {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return 0, fmt.Errorf("invalid IP address: %s", ipStr)
	}

	// Convert to uint32 (IPv4 only)
	ip = ip.To4()
	if ip == nil {
		return 0, fmt.Errorf("not an IPv4 address: %s", ipStr)
	}

	return binary.BigEndian.Uint32(ip), nil
}

// CreateHTTPPacket creates an HTTP request packet
func CreateHTTPPacket(srcIP, dstIP string, srcPort, dstPort uint16, method, path, host string) (*NetworkPacket, error) {
	// Parse IP addresses
	srcIPUint, err := ParseIPAddress(srcIP)
	if err != nil {
		return nil, fmt.Errorf("invalid source IP: %v", err)
	}

	dstIPUint, err := ParseIPAddress(dstIP)
	if err != nil {
		return nil, fmt.Errorf("invalid destination IP: %v", err)
	}

	// Format HTTP request
	httpRequest := fmt.Sprintf("%s %s HTTP/1.1\r\nHost: %s\r\n\r\n", method, path, host)

	// Create IPv4 header
	ipHeader := IPv4Header{
		Version:        4,
		HeaderLength:   5,
		ServiceType:    0,
		TotalLength:    uint(20 + 20 + len(httpRequest)), // IP + TCP + payload
		Identification: 54321,
		Flags:          2, // Don't Fragment
		FragmentOffset: 0,
		TTL:            64,
		Protocol:       6, // TCP
		Checksum:       0,
		SourceIP:       uint(srcIPUint),
		DestinationIP:  uint(dstIPUint),
	}

	// Create TCP header
	tcpHeader := TCPHeader{
		SourcePort:      uint(srcPort),
		DestinationPort: uint(dstPort),
		SequenceNumber:  1000,
		Acknowledgment:  0,
		DataOffset:      5,
		Reserved:        0,
		Flags: TCPFlags{
			Reserved: 0,
			URG:      0,
			ACK:      0,
			PSH:      1,
			RST:      0,
			SYN:      0,
			FIN:      0,
		},
		Window:        8192,
		Checksum:      0,
		UrgentPointer: 0,
	}

	// Create packet
	packet := &NetworkPacket{
		IPv4Header: ipHeader,
		TCPHeader:  tcpHeader,
		Payload:    []byte(httpRequest),
	}

	return packet, nil
}

// CreateTCPSynPacket creates a TCP SYN packet for connection establishment
func CreateTCPSynPacket(srcIP, dstIP string, srcPort, dstPort uint16) (*NetworkPacket, error) {
	// Parse IP addresses
	srcIPUint, err := ParseIPAddress(srcIP)
	if err != nil {
		return nil, fmt.Errorf("invalid source IP: %v", err)
	}

	dstIPUint, err := ParseIPAddress(dstIP)
	if err != nil {
		return nil, fmt.Errorf("invalid destination IP: %v", err)
	}

	// Create IPv4 header
	ipHeader := IPv4Header{
		Version:        4,
		HeaderLength:   5,
		ServiceType:    0,
		TotalLength:    40, // 20 (IP) + 20 (TCP)
		Identification: 54321,
		Flags:          2, // Don't Fragment
		FragmentOffset: 0,
		TTL:            64,
		Protocol:       6, // TCP
		Checksum:       0,
		SourceIP:       uint(srcIPUint),
		DestinationIP:  uint(dstIPUint),
	}

	// Create TCP header with SYN flag
	tcpHeader := TCPHeader{
		SourcePort:      uint(srcPort),
		DestinationPort: uint(dstPort),
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
	}

	// Create packet
	packet := &NetworkPacket{
		IPv4Header: ipHeader,
		TCPHeader:  tcpHeader,
		Payload:    []byte{},
	}

	return packet, nil
}
