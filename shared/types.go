package shared

import "github.com/funvibe/funbit/pkg/funbit"

// BitstringObject wraps a funbit.BitString with methods for Lua
type BitstringObject struct {
	BitString *funbit.BitString
}

// BitstringByte represents a byte extracted from a bitstring
// This allows print functions to distinguish between regular numbers and bitstring bytes
type BitstringByte struct {
	Value byte
}

// Len returns the length in bits
func (bo *BitstringObject) Len() int {
	return int(bo.BitString.Length())
}

// GetByte returns byte at index (for []byte access)
func (bo *BitstringObject) GetByte(index int) BitstringByte {
	bytes := bo.BitString.ToBytes()
	if index < 0 || index >= len(bytes) {
		return BitstringByte{Value: 0}
	}
	return BitstringByte{Value: bytes[index]}
}
