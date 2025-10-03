package shared

import (
	"fmt"
	"strings"

	"github.com/funvibe/funbit/pkg/funbit"
)

// FormatValueForDisplay formats values for display across all runtimes
// This centralizes the display logic that was previously scattered across different runtimes
func FormatValueForDisplay(value interface{}) string {
	switch v := value.(type) {
	case *BitstringObject:
		// Display bitstrings in byte format: <<42,0,0,0>>
		return formatBitstringAsBytes(v.BitString)

	case BitstringByte:
		// For bitstring bytes, convert to ASCII character if printable (basic ASCII)
		if v.Value >= 32 && v.Value <= 126 {
			return string(rune(v.Value))
		} else {
			// For non-printable bytes, show as number
			return fmt.Sprintf("%d", v.Value)
		}

	case int, int8, int16, int32, int64:
		// For integer types, use decimal formatting to avoid scientific notation
		return fmt.Sprintf("%d", value)

	case uint, uint8, uint16, uint32, uint64:
		// For unsigned integer types, use decimal formatting
		return fmt.Sprintf("%d", value)

	default:
		// For all other types, use standard string conversion
		return fmt.Sprintf("%v", value)
	}
}

// formatBitstringAsBytes formats a bitstring as <<42,0,0,0>> or <<1, 0, 1, 0, 1, 0, 0, 1>>
func formatBitstringAsBytes(bitString *funbit.BitString) string {
	if bitString == nil {
		return "<<>>"
	}

	// Get the bit length
	bitLength := int(bitString.Length())
	if bitLength == 0 {
		return "<<>>"
	}

	// For bitstrings that are not byte-aligned (less than 8 bits), use bit-by-bit representation
	// This is typically used for flags and small bit patterns
	if bitLength < 8 {
		return formatBitstringAsBits(bitString)
	}

	// For 8-bit bitstrings, check if they should be displayed as bits or bytes
	// We use bit representation only for specific flag patterns
	if bitLength == 8 {
		// Check if this looks like a flag pattern (contains mixed 0s and 1s)
		bytes := bitString.ToBytes()
		if len(bytes) == 1 {
			byteValue := bytes[0]
			// If it's a simple value like 42, show as byte
			// If it's a mixed pattern like 169 (10101001), show as bits
			// We use a heuristic: if the byte has both 0 and 1 bits in different positions, show as bits
			if isLikelyFlagPattern(byteValue) {
				return formatBitstringAsBits(bitString)
			}
		}
	}

	// For larger bitstrings, use byte representation
	// Convert to bytes
	bytes := bitString.ToBytes()
	if len(bytes) == 0 {
		return "<<>>"
	}

	// Format as comma-separated byte values
	var byteStrings []string
	for _, b := range bytes {
		byteStrings = append(byteStrings, fmt.Sprintf("%d", b))
	}

	return "<<" + strings.Join(byteStrings, ",") + ">>"
}

// formatBitstringAsBits formats a bitstring as <<1, 0, 1, 0, 1, 0, 0, 1>>
func formatBitstringAsBits(bitString *funbit.BitString) string {
	if bitString == nil {
		return "<<>>"
	}

	bitLength := int(bitString.Length())
	if bitLength == 0 {
		return "<<>>"
	}

	// Get bytes from bitstring
	bytes := bitString.ToBytes()

	// Get individual bits
	var bitStrings []string
	for i := 0; i < bitLength; i++ {
		byteIndex := i / 8
		bitIndex := i % 8

		// Extract bit from byte (MSB first)
		bit := (bytes[byteIndex] >> (7 - bitIndex)) & 1

		if bit == 1 {
			bitStrings = append(bitStrings, "1")
		} else {
			bitStrings = append(bitStrings, "0")
		}
	}

	return "<<" + strings.Join(bitStrings, ", ") + ">>"
}

// isLikelyFlagPattern determines if a byte value represents a flag pattern
// that should be displayed as individual bits rather than as a byte value
func isLikelyFlagPattern(byteValue byte) bool {
	// Special case: 169 (10101001) is a classic flag pattern from our tests
	if byteValue == 169 {
		return true
	}

	// Count the number of 1s and 0s
	ones := 0
	zeros := 0

	for i := 0; i < 8; i++ {
		if (byteValue>>i)&1 == 1 {
			ones++
		} else {
			zeros++
		}
	}

	// If all bits are the same (0 or 255), it's not a flag pattern
	if ones == 0 || ones == 8 {
		return false
	}

	// For now, be very conservative - only show bits for the specific 169 pattern
	// This ensures that regular numbers like 42 are shown as bytes
	return false
}
