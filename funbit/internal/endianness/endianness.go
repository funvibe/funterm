package endianness

import (
	"encoding/binary"
	"unsafe"
)

// GetNativeEndianness returns the native endianness of the system
func GetNativeEndianness() string {
	if isLittleEndian() {
		return "little"
	}
	return "big"
}

// ConvertEndianness converts data between different endianness
func ConvertEndianness(data []byte, from, to string, size uint) ([]byte, error) {
	if from == to {
		return data, nil
	}

	// Resolve native endianness
	if from == "native" {
		from = GetNativeEndianness()
	}
	if to == "native" {
		to = GetNativeEndianness()
	}

	// If still the same after resolving native, return as-is
	if from == to {
		return data, nil
	}

	// Create a copy and reverse the byte order
	result := make([]byte, len(data))
	copy(result, data)
	reverseBytes(result)
	return result, nil
}

// EncodeWithEndianness encodes a value with the specified endianness
func EncodeWithEndianness(value interface{}, size uint, endianness string) ([]byte, error) {
	// Resolve native endianness
	if endianness == "native" {
		endianness = GetNativeEndianness()
	}

	data := make([]byte, size/8)

	switch v := value.(type) {
	case uint64:
		switch size {
		case 16:
			if endianness == "little" {
				binary.LittleEndian.PutUint16(data, uint16(v))
			} else {
				binary.BigEndian.PutUint16(data, uint16(v))
			}
		case 32:
			if endianness == "little" {
				binary.LittleEndian.PutUint32(data, uint32(v))
			} else {
				binary.BigEndian.PutUint32(data, uint32(v))
			}
		case 64:
			if endianness == "little" {
				binary.LittleEndian.PutUint64(data, v)
			} else {
				binary.BigEndian.PutUint64(data, v)
			}
		default:
			return nil, unsupportedSizeError(size)
		}
	case int64:
		switch size {
		case 16:
			if endianness == "little" {
				binary.LittleEndian.PutUint16(data, uint16(v))
			} else {
				binary.BigEndian.PutUint16(data, uint16(v))
			}
		case 32:
			if endianness == "little" {
				binary.LittleEndian.PutUint32(data, uint32(v))
			} else {
				binary.BigEndian.PutUint32(data, uint32(v))
			}
		case 64:
			if endianness == "little" {
				binary.LittleEndian.PutUint64(data, uint64(v))
			} else {
				binary.BigEndian.PutUint64(data, uint64(v))
			}
		default:
			return nil, unsupportedSizeError(size)
		}
	default:
		return nil, unsupportedTypeError(value)
	}

	return data, nil
}

// DecodeWithEndianness decodes data with the specified endianness
func DecodeWithEndianness(data []byte, size uint, endianness string) (interface{}, error) {
	// Resolve native endianness
	if endianness == "native" {
		endianness = GetNativeEndianness()
	}

	switch size {
	case 16:
		if endianness == "little" {
			return uint64(binary.LittleEndian.Uint16(data)), nil
		} else {
			return uint64(binary.BigEndian.Uint16(data)), nil
		}
	case 32:
		if endianness == "little" {
			return uint64(binary.LittleEndian.Uint32(data)), nil
		} else {
			return uint64(binary.BigEndian.Uint32(data)), nil
		}
	case 64:
		if endianness == "little" {
			return binary.LittleEndian.Uint64(data), nil
		} else {
			return binary.BigEndian.Uint64(data), nil
		}
	default:
		return nil, unsupportedSizeError(size)
	}
}

// reverseBytes reverses the byte order in place
func reverseBytes(data []byte) {
	for i, j := 0, len(data)-1; i < j; i, j = i+1, j-1 {
		data[i], data[j] = data[j], data[i]
	}
}

// isLittleEndian checks if the system is little-endian
func isLittleEndian() bool {
	var i uint32 = 0x01020304
	return *(*byte)(unsafe.Pointer(&i)) == 0x04
}

func unsupportedSizeError(size uint) error {
	return &EndiannessError{
		Code:    "UNSUPPORTED_SIZE",
		Message: "unsupported size for endianness conversion",
		Size:    size,
	}
}

func unsupportedTypeError(value interface{}) error {
	return &EndiannessError{
		Code:    "UNSUPPORTED_TYPE",
		Message: "unsupported type for endianness conversion",
		Type:    value,
	}
}

// EndiannessError represents an error in endianness operations
type EndiannessError struct {
	Code    string
	Message string
	Size    uint
	Type    interface{}
}

func (e *EndiannessError) Error() string {
	if e.Size != 0 {
		return e.Message + " (size: " + string(rune('0'+e.Size)) + ")"
	}
	if e.Type != nil {
		return e.Message + " (type: " + string(e.Type.(string)) + ")"
	}
	return e.Message
}
