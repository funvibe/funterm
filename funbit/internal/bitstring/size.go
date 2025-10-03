package bitstring

import (
	"errors"
	"fmt"
)

// Size handling errors
var (
	ErrInvalidSize      = errors.New("invalid size")
	ErrSizeTooLarge     = errors.New("size too large")
	ErrInvalidPosition  = errors.New("invalid position")
	ErrLengthTooLarge   = errors.New("length too large")
	ErrInvalidAlignment = errors.New("invalid alignment")
)

// ValidateSize validates the size and unit for a segment
func ValidateSize(size uint, unit uint) error {
	// Size 0 is allowed in Erlang bit syntax - it contributes 0 bits
	// No validation needed for size 0

	// Note: No limit on total bits (size * unit) as per BIT_SYNTAX_SPEC.md
	// Unit validation (1-256) is handled separately in segment.go

	return nil
}

// CalculateTotalSize calculates the total size in bits for a segment
func CalculateTotalSize(segment Segment) (uint, error) {
	if !segment.SizeSpecified {
		return 0, fmt.Errorf("%w: size is required", ErrInvalidSize)
	}

	size := segment.Size
	unit := segment.Unit
	if unit == 0 {
		unit = 1 // Default unit is 1 bit
	}

	if err := ValidateSize(size, unit); err != nil {
		return 0, err
	}

	return size * unit, nil
}

// ExtractBits extracts a sequence of bits from data
func ExtractBits(data []byte, start, length uint) ([]byte, error) {
	if start >= uint(len(data))*8 {
		return nil, fmt.Errorf("%w: start position %d is beyond data length", ErrInvalidPosition, start)
	}

	if length == 0 {
		return []byte{}, nil
	}

	if start+length > uint(len(data))*8 {
		return nil, fmt.Errorf("%w: cannot extract %d bits from position %d", ErrLengthTooLarge, length, start)
	}

	// Calculate the number of bytes needed
	resultBytes := (length + 7) / 8
	result := make([]byte, resultBytes)

	for i := uint(0); i < length; i++ {
		bitPos := start + i
		bytePos := bitPos / 8
		bitInByte := uint(7 - (bitPos % 8)) // MSB first

		if data[bytePos]&(1<<bitInByte) != 0 {
			resultBytePos := i / 8
			bitInResult := uint(7 - (i % 8))
			result[resultBytePos] |= (1 << bitInResult)
		}
	}

	return result, nil
}

// SetBits sets a sequence of bits in target data
func SetBits(target, data []byte, start uint) error {
	if start >= uint(len(target))*8 {
		return fmt.Errorf("%w: start position %d is beyond target length", ErrInvalidPosition, start)
	}

	length := uint(len(data)) * 8
	if start+length > uint(len(target))*8 {
		return fmt.Errorf("%w: cannot set %d bits from position %d", ErrLengthTooLarge, length, start)
	}

	for i := uint(0); i < length; i++ {
		bitPos := start + i
		targetBytePos := bitPos / 8
		bitInTarget := uint(7 - (bitPos % 8))

		dataBytePos := i / 8
		bitInData := uint(7 - (i % 8))

		// Clear the target bit
		target[targetBytePos] &^= (1 << bitInTarget)

		// Set the target bit if data bit is set
		if data[dataBytePos]&(1<<bitInData) != 0 {
			target[targetBytePos] |= (1 << bitInTarget)
		}
	}

	return nil
}

// AlignData aligns data to a specified boundary
func AlignData(data []byte, offset, alignment uint) ([]byte, error) {
	if alignment == 0 {
		return nil, fmt.Errorf("%w: alignment must be positive", ErrInvalidAlignment)
	}

	// Calculate current bit position
	currentBits := offset + uint(len(data))*8

	// Calculate padding needed
	padding := (alignment - (currentBits % alignment)) % alignment

	if padding == 0 {
		// Already aligned
		return data, nil
	}

	// Create result with padding
	result := make([]byte, len(data))
	copy(result, data)

	// Add padding bytes if needed
	paddingBytes := (padding + 7) / 8
	if paddingBytes > 0 {
		result = append(result, make([]byte, paddingBytes)...)
	}

	return result, nil
}

// PadData pads data to reach a target bit length
func PadData(data []byte, bitLen, target uint) []byte {
	if bitLen >= target {
		return data
	}

	padding := target - bitLen
	paddingBytes := (padding + 7) / 8

	result := make([]byte, len(data))
	copy(result, data)

	if paddingBytes > 0 {
		result = append(result, make([]byte, paddingBytes)...)
	}

	return result
}

// GetBitValue gets the value of a single bit at the specified position
func GetBitValue(data []byte, pos uint) (bool, error) {
	if pos >= uint(len(data))*8 {
		return false, fmt.Errorf("%w: position %d is beyond data length", ErrInvalidPosition, pos)
	}

	bytePos := pos / 8
	bitInByte := 7 - (pos % 8)

	return (data[bytePos] & (1 << bitInByte)) != 0, nil
}

// SetBitValue sets the value of a single bit at the specified position
func SetBitValue(data []byte, pos uint, value bool) error {
	if pos >= uint(len(data))*8 {
		return fmt.Errorf("%w: position %d is beyond data length", ErrInvalidPosition, pos)
	}

	bytePos := pos / 8
	bitInByte := 7 - (pos % 8)

	if value {
		data[bytePos] |= (1 << bitInByte)
	} else {
		data[bytePos] &^= (1 << bitInByte)
	}

	return nil
}

// CountLeadingZeros counts the number of leading zero bits in the data
func CountLeadingZeros(data []byte) uint {
	if len(data) == 0 {
		return 0
	}

	count := uint(0)
	for _, b := range data {
		if b == 0 {
			count += 8
		} else {
			// Count leading zeros in this byte
			for i := 7; i >= 0; i-- {
				if b&(1<<i) != 0 {
					break
				}
				count++
			}
			break
		}
	}

	return count
}

// CountTrailingZeros counts the number of trailing zero bits in the data
func CountTrailingZeros(data []byte) uint {
	if len(data) == 0 {
		return 0
	}

	count := uint(0)
	for i := len(data) - 1; i >= 0; i-- {
		b := data[i]
		if b == 0 {
			count += 8
		} else {
			// Count trailing zeros in this byte
			for j := 0; j < 8; j++ {
				if b&(1<<j) != 0 {
					break
				}
				count++
			}
			break
		}
	}

	return count
}
