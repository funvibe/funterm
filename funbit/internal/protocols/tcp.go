package protocols

import (
	"fmt"
	"strings"

	"github.com/funvibe/funbit/internal/bitstring"
	"github.com/funvibe/funbit/internal/builder"
	"github.com/funvibe/funbit/internal/matcher"
)

// TCPFlags represents the structure of TCP flags
type TCPFlags struct {
	Reserved uint // Reserved bits (2 bits)
	URG      uint // Urgent flag (1 bit)
	ACK      uint // Acknowledgment flag (1 bit)
	PSH      uint // Push flag (1 bit)
	RST      uint // Reset flag (1 bit)
	SYN      uint // Synchronize flag (1 bit)
	FIN      uint // Finish flag (1 bit)
}

// TCPHeader represents a complete TCP header
type TCPHeader struct {
	SourcePort      uint     // Source port (16 bits)
	DestinationPort uint     // Destination port (16 bits)
	SequenceNumber  uint     // Sequence number (32 bits)
	Acknowledgment  uint     // Acknowledgment number (32 bits)
	DataOffset      uint     // Data offset (4 bits)
	Reserved        uint     // Reserved (3 bits)
	Flags           TCPFlags // Flags (9 bits: 2 reserved + 7 flags)
	Window          uint     // Window size (16 bits)
	Checksum        uint     // Checksum (16 bits)
	UrgentPointer   uint     // Urgent pointer (16 bits)
}

// BuildTCPFlags creates TCP flags from a structure
func BuildTCPFlags(flags TCPFlags) (*bitstring.BitString, error) {
	return builder.NewBuilder().
		AddInteger(flags.Reserved, bitstring.WithSize(2)).
		AddInteger(flags.URG, bitstring.WithSize(1)).
		AddInteger(flags.ACK, bitstring.WithSize(1)).
		AddInteger(flags.PSH, bitstring.WithSize(1)).
		AddInteger(flags.RST, bitstring.WithSize(1)).
		AddInteger(flags.SYN, bitstring.WithSize(1)).
		AddInteger(flags.FIN, bitstring.WithSize(1)).
		Build()
}

// ParseTCPFlags parses TCP flags from a bitstring
func ParseTCPFlags(data *bitstring.BitString) (*TCPFlags, error) {
	var flags TCPFlags

	_, err := matcher.NewMatcher().
		Integer(&flags.Reserved, bitstring.WithSize(2)).
		Integer(&flags.URG, bitstring.WithSize(1)).
		Integer(&flags.ACK, bitstring.WithSize(1)).
		Integer(&flags.PSH, bitstring.WithSize(1)).
		Integer(&flags.RST, bitstring.WithSize(1)).
		Integer(&flags.SYN, bitstring.WithSize(1)).
		Integer(&flags.FIN, bitstring.WithSize(1)).
		Match(data)

	if err != nil {
		return nil, err
	}

	return &flags, nil
}

// BuildTCPHeader creates a complete TCP header from a structure
func BuildTCPHeader(header TCPHeader) (*bitstring.BitString, error) {
	return builder.NewBuilder().
		AddInteger(header.SourcePort, bitstring.WithSize(16), bitstring.WithEndianness("big")).
		AddInteger(header.DestinationPort, bitstring.WithSize(16), bitstring.WithEndianness("big")).
		AddInteger(header.SequenceNumber, bitstring.WithSize(32), bitstring.WithEndianness("big")).
		AddInteger(header.Acknowledgment, bitstring.WithSize(32), bitstring.WithEndianness("big")).
		AddInteger(header.DataOffset, bitstring.WithSize(4)).
		AddInteger(header.Reserved, bitstring.WithSize(3)).
		AddInteger(header.Flags.Reserved, bitstring.WithSize(2)).
		AddInteger(header.Flags.URG, bitstring.WithSize(1)).
		AddInteger(header.Flags.ACK, bitstring.WithSize(1)).
		AddInteger(header.Flags.PSH, bitstring.WithSize(1)).
		AddInteger(header.Flags.RST, bitstring.WithSize(1)).
		AddInteger(header.Flags.SYN, bitstring.WithSize(1)).
		AddInteger(header.Flags.FIN, bitstring.WithSize(1)).
		AddInteger(header.Window, bitstring.WithSize(16), bitstring.WithEndianness("big")).
		AddInteger(header.Checksum, bitstring.WithSize(16), bitstring.WithEndianness("big")).
		AddInteger(header.UrgentPointer, bitstring.WithSize(16), bitstring.WithEndianness("big")).
		Build()
}

// ParseTCPHeader parses a complete TCP header from a bitstring
func ParseTCPHeader(data *bitstring.BitString) (*TCPHeader, error) {
	var header TCPHeader

	_, err := matcher.NewMatcher().
		Integer(&header.SourcePort, bitstring.WithSize(16), bitstring.WithEndianness("big")).
		Integer(&header.DestinationPort, bitstring.WithSize(16), bitstring.WithEndianness("big")).
		Integer(&header.SequenceNumber, bitstring.WithSize(32), bitstring.WithEndianness("big")).
		Integer(&header.Acknowledgment, bitstring.WithSize(32), bitstring.WithEndianness("big")).
		Integer(&header.DataOffset, bitstring.WithSize(4)).
		Integer(&header.Reserved, bitstring.WithSize(3)).
		Integer(&header.Flags.Reserved, bitstring.WithSize(2)).
		Integer(&header.Flags.URG, bitstring.WithSize(1)).
		Integer(&header.Flags.ACK, bitstring.WithSize(1)).
		Integer(&header.Flags.PSH, bitstring.WithSize(1)).
		Integer(&header.Flags.RST, bitstring.WithSize(1)).
		Integer(&header.Flags.SYN, bitstring.WithSize(1)).
		Integer(&header.Flags.FIN, bitstring.WithSize(1)).
		Integer(&header.Window, bitstring.WithSize(16), bitstring.WithEndianness("big")).
		Integer(&header.Checksum, bitstring.WithSize(16), bitstring.WithEndianness("big")).
		Integer(&header.UrgentPointer, bitstring.WithSize(16), bitstring.WithEndianness("big")).
		Match(data)

	if err != nil {
		return nil, err
	}

	return &header, nil
}

// ValidateTCPFlags performs validation of TCP flags
func ValidateTCPFlags(flags *TCPFlags) error {
	if flags == nil {
		return fmt.Errorf("flags is nil")
	}

	// Reserved bits must be 0
	if flags.Reserved != 0 {
		return fmt.Errorf("reserved bits must be zero")
	}

	// Check flag values (must be 0 or 1)
	flagValues := []uint{flags.URG, flags.ACK, flags.PSH, flags.RST, flags.SYN, flags.FIN}
	for i, value := range flagValues {
		if value > 1 {
			flagNames := []string{"URG", "ACK", "PSH", "RST", "SYN", "FIN"}
			return fmt.Errorf("%s flag must be 0 or 1", flagNames[i])
		}
	}

	return nil
}

// ValidateTCPHeader performs full validation of a TCP header
func ValidateTCPHeader(header *TCPHeader) error {
	if header == nil {
		return fmt.Errorf("header is nil")
	}

	// Port validation
	if header.SourcePort > 65535 {
		return fmt.Errorf("invalid source port")
	}

	if header.DestinationPort > 65535 {
		return fmt.Errorf("invalid destination port")
	}

	// Data offset validation
	if header.DataOffset < 5 || header.DataOffset > 15 {
		return fmt.Errorf("invalid data offset, must be between 5 and 15")
	}

	// Reserved bits validation
	if header.Reserved != 0 {
		return fmt.Errorf("reserved bits must be zero")
	}

	// Flags validation
	if err := ValidateTCPFlags(&header.Flags); err != nil {
		return err
	}

	// Window size validation
	if header.Window > 65535 {
		return fmt.Errorf("invalid window size")
	}

	return nil
}

// GetTCPHeaderLength returns the TCP header length in bytes
func GetTCPHeaderLength(header *TCPHeader) uint {
	return header.DataOffset * 4
}

// GetTCPFlagsString returns a string representation of set flags
func GetTCPFlagsString(flags TCPFlags) string {
	var flagStrings []string

	if flags.URG == 1 {
		flagStrings = append(flagStrings, "URG")
	}
	if flags.ACK == 1 {
		flagStrings = append(flagStrings, "ACK")
	}
	if flags.PSH == 1 {
		flagStrings = append(flagStrings, "PSH")
	}
	if flags.RST == 1 {
		flagStrings = append(flagStrings, "RST")
	}
	if flags.SYN == 1 {
		flagStrings = append(flagStrings, "SYN")
	}
	if flags.FIN == 1 {
		flagStrings = append(flagStrings, "FIN")
	}

	if len(flagStrings) == 0 {
		return "NONE"
	}

	return strings.Join(flagStrings, "|")
}

// IsTCPConnectionEstablishment checks if the packet is a connection establishment (SYN)
func IsTCPConnectionEstablishment(flags TCPFlags) bool {
	return flags.SYN == 1 && flags.ACK == 0
}

// IsTCPConnectionEstablished checks if the packet is a connection establishment confirmation (SYN+ACK)
func IsTCPConnectionEstablished(flags TCPFlags) bool {
	return flags.SYN == 1 && flags.ACK == 1
}

// IsTCPConnectionTermination checks if the packet is a connection termination (FIN or RST)
func IsTCPConnectionTermination(flags TCPFlags) bool {
	return flags.FIN == 1 || flags.RST == 1
}

// IsTCPDataPacket checks if the packet contains data (PSH and ACK)
func IsTCPDataPacket(flags TCPFlags) bool {
	return flags.PSH == 1 && flags.ACK == 1
}
