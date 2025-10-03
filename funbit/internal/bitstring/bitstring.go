package bitstring

// BitString represents a sequence of bits
type BitString struct {
	data     []byte
	bitLen   uint // length in bits
	capacity uint // capacity in bits (for optimization)
}

// Segment represents a single segment in bitstring construction/matching
type Segment struct {
	Value         interface{}
	Size          uint
	SizeSpecified bool // Flag to indicate if size was explicitly specified
	Type          string
	Signed        bool
	Endianness    string
	Unit          uint
	UnitSpecified bool   // Flag to indicate if unit was explicitly specified
	DynamicSize   *uint  // Pointer to variable for dynamic size
	DynamicExpr   string // Expression for dynamic size calculation
	IsDynamic     bool   // Flag to indicate if size is dynamic
}

// SegmentResult represents result of segment matching
type SegmentResult struct {
	Value     interface{}
	Matched   bool
	Remaining *BitString
}

// NewBitString creates an empty bitstring
func NewBitString() *BitString {
	return &BitString{
		data:   []byte{},
		bitLen: 0,
	}
}

// NewBitStringFromBytes creates a bitstring from byte slice
func NewBitStringFromBytes(data []byte) *BitString {
	if data == nil {
		data = []byte{}
	}

	// Create a copy of the data to avoid external modifications
	copiedData := make([]byte, len(data))
	copy(copiedData, data)

	return &BitString{
		data:   copiedData,
		bitLen: uint(len(data)) * 8,
	}
}

// NewBitStringFromBits creates a bitstring from bits with specific length
func NewBitStringFromBits(data []byte, length uint) *BitString {
	if data == nil {
		data = []byte{}
	}

	// Validate that we have enough data for the specified length
	requiredBytes := (length + 7) / 8
	if uint(len(data)) < requiredBytes {
		return nil // insufficient data
	}

	// Create a copy of the required portion
	copiedData := make([]byte, requiredBytes)
	copy(copiedData, data[:requiredBytes])

	return &BitString{
		data:   copiedData,
		bitLen: length,
	}
}

// Length returns bitstring length in bits
func (bs *BitString) Length() uint {
	return bs.bitLen
}

// IsEmpty checks if bitstring is empty
func (bs *BitString) IsEmpty() bool {
	return bs.bitLen == 0
}

// IsBinary checks if bitstring length is multiple of 8
func (bs *BitString) IsBinary() bool {
	return bs.bitLen%8 == 0
}

// ToBytes converts bitstring to byte slice (pads with zeros if needed)
func (bs *BitString) ToBytes() []byte {
	if bs.bitLen == 0 {
		return []byte{}
	}

	// If we have an exact multiple of 8 bits, return the data as-is
	if bs.IsBinary() {
		// Return a copy to avoid external modifications
		result := make([]byte, len(bs.data))
		copy(result, bs.data)
		return result
	}

	// For non-binary bitstrings, we need to pad to full bytes
	byteLen := (bs.bitLen + 7) / 8
	result := make([]byte, byteLen)
	copy(result, bs.data)

	// The last byte may have some unused bits, but that's okay
	// as the bitLen tracks the actual bit length
	return result
}

// Clone creates a copy of bitstring
func (bs *BitString) Clone() *BitString {
	if bs == nil {
		return nil
	}

	// Create copies of all data
	copiedData := make([]byte, len(bs.data))
	copy(copiedData, bs.data)

	return &BitString{
		data:     copiedData,
		bitLen:   bs.bitLen,
		capacity: bs.capacity,
	}
}
