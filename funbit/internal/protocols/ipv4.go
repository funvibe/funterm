package protocols

import (
	"errors"

	"github.com/funvibe/funbit/internal/bitstring"
	"github.com/funvibe/funbit/internal/builder"
	"github.com/funvibe/funbit/internal/matcher"
)

// IPv4Header represents the structure of an IPv4 header
type IPv4Header struct {
	Version        uint // Version (4 bits)
	HeaderLength   uint // Header length (4 bits)
	ServiceType    uint // Service type (8 bits)
	TotalLength    uint // Total length (16 bits)
	Identification uint // Identification (16 bits)
	Flags          uint // Flags (3 bits)
	FragmentOffset uint // Fragment offset (13 bits)
	TTL            uint // Time to live (8 bits)
	Protocol       uint // Protocol (8 bits)
	Checksum       uint // Checksum (16 bits)
	SourceIP       uint // Source IP address (32 bits)
	DestinationIP  uint // Destination IP address (32 bits)
}

// BuildIPv4Header creates an IPv4 header from a structure
func BuildIPv4Header(header IPv4Header) (*bitstring.BitString, error) {
	return builder.NewBuilder().
		AddInteger(header.Version, bitstring.WithSize(4)).
		AddInteger(header.HeaderLength, bitstring.WithSize(4)).
		AddInteger(header.ServiceType, bitstring.WithSize(8)).
		AddInteger(header.TotalLength, bitstring.WithSize(16), bitstring.WithEndianness("big")).
		AddInteger(header.Identification, bitstring.WithSize(16), bitstring.WithEndianness("big")).
		AddInteger(header.Flags, bitstring.WithSize(3)).
		AddInteger(header.FragmentOffset, bitstring.WithSize(13)).
		AddInteger(header.TTL, bitstring.WithSize(8)).
		AddInteger(header.Protocol, bitstring.WithSize(8)).
		AddInteger(header.Checksum, bitstring.WithSize(16), bitstring.WithEndianness("big")).
		AddInteger(header.SourceIP, bitstring.WithSize(32), bitstring.WithEndianness("big")).
		AddInteger(header.DestinationIP, bitstring.WithSize(32), bitstring.WithEndianness("big")).
		Build()
}

// ParseIPv4Header parses an IPv4 header from a bitstring
func ParseIPv4Header(data *bitstring.BitString) (*IPv4Header, error) {
	var header IPv4Header

	_, err := matcher.NewMatcher().
		Integer(&header.Version, bitstring.WithSize(4)).
		Integer(&header.HeaderLength, bitstring.WithSize(4)).
		Integer(&header.ServiceType, bitstring.WithSize(8)).
		Integer(&header.TotalLength, bitstring.WithSize(16), bitstring.WithEndianness("big")).
		Integer(&header.Identification, bitstring.WithSize(16), bitstring.WithEndianness("big")).
		Integer(&header.Flags, bitstring.WithSize(3)).
		Integer(&header.FragmentOffset, bitstring.WithSize(13)).
		Integer(&header.TTL, bitstring.WithSize(8)).
		Integer(&header.Protocol, bitstring.WithSize(8)).
		Integer(&header.Checksum, bitstring.WithSize(16), bitstring.WithEndianness("big")).
		Integer(&header.SourceIP, bitstring.WithSize(32), bitstring.WithEndianness("big")).
		Integer(&header.DestinationIP, bitstring.WithSize(32), bitstring.WithEndianness("big")).
		Match(data)

	if err != nil {
		return nil, err
	}

	// Basic IPv4 header validation
	if header.Version != 4 {
		return nil, errors.New("invalid IPv4 version")
	}

	if header.HeaderLength < 5 {
		return nil, errors.New("invalid IPv4 header length")
	}

	return &header, nil
}

// ValidateIPv4Header performs full validation of an IPv4 header
func ValidateIPv4Header(header *IPv4Header) error {
	if header == nil {
		return errors.New("header is nil")
	}

	if header.Version != 4 {
		return errors.New("invalid IPv4 version, must be 4")
	}

	if header.HeaderLength < 5 || header.HeaderLength > 15 {
		return errors.New("invalid IPv4 header length, must be between 5 and 15")
	}

	if header.TotalLength < 20 {
		return errors.New("invalid total length, must be at least 20 bytes")
	}

	// Check reserved bits in flags
	if header.Flags&0xE0 != 0 {
		return errors.New("reserved bits in flags must be zero")
	}

	return nil
}

// GetIPv4HeaderLength returns the header length in bytes
func GetIPv4HeaderLength(header *IPv4Header) uint {
	return header.HeaderLength * 4
}

// GetIPv4PayloadLength returns the payload length in bytes
func GetIPv4PayloadLength(header *IPv4Header) uint {
	return header.TotalLength - GetIPv4HeaderLength(header)
}
