package protocols

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"

	"github.com/funvibe/funbit/internal/bitstring"
	"github.com/funvibe/funbit/internal/builder"
	"github.com/funvibe/funbit/internal/matcher"
)

// PNGSignature represents the PNG file signature
var PNGSignature = []byte{137, 80, 78, 71, 13, 10, 26, 10}

// PNGChunk represents the structure of a PNG chunk
type PNGChunk struct {
	Length uint32 // Chunk data length (4 bytes)
	Type   []byte // Chunk type (4 bytes)
	Data   []byte // Chunk data (variable length)
	CRC    uint32 // CRC32 checksum (4 bytes)
}

// PNGHeader represents the structure of a PNG header
type PNGHeader struct {
	Signature []byte     // PNG signature (8 bytes)
	Chunks    []PNGChunk // PNG chunks
}

// PNGIHDRChunk represents the IHDR chunk of PNG (image header)
type PNGIHDRChunk struct {
	Width       uint32 // Image width (4 bytes)
	Height      uint32 // Image height (4 bytes)
	BitDepth    uint8  // Color depth (1 byte)
	ColorType   uint8  // Color type (1 byte)
	Compression uint8  // Compression method (1 byte)
	Filter      uint8  // Filter method (1 byte)
	Interlace   uint8  // Interlace method (1 byte)
}

// BuildPNGHeader creates a PNG header from a structure
func BuildPNGHeader(header PNGHeader) (*bitstring.BitString, error) {
	// Check signature
	if !bytes.Equal(header.Signature, PNGSignature) {
		return nil, errors.New("invalid PNG signature")
	}

	// Start with signature
	b := builder.NewBuilder().AddBinary(header.Signature)

	// Add chunks
	for _, chunk := range header.Chunks {
		chunkData, err := BuildPNGChunk(chunk)
		if err != nil {
			return nil, fmt.Errorf("failed to build chunk %s: %v", string(chunk.Type), err)
		}
		b.AddBinary(chunkData)
	}

	return b.Build()
}

// BuildPNGChunk creates a PNG chunk from a structure
func BuildPNGChunk(chunk PNGChunk) ([]byte, error) {
	if len(chunk.Type) != 4 {
		return nil, errors.New("chunk type must be 4 bytes")
	}

	// Create buffer for chunk
	var buf bytes.Buffer

	// Write data length (big-endian)
	if err := binary.Write(&buf, binary.BigEndian, chunk.Length); err != nil {
		return nil, fmt.Errorf("failed to write chunk length: %v", err)
	}

	// Write chunk type
	if _, err := buf.Write(chunk.Type); err != nil {
		return nil, fmt.Errorf("failed to write chunk type: %v", err)
	}

	// Write data
	if _, err := buf.Write(chunk.Data); err != nil {
		return nil, fmt.Errorf("failed to write chunk data: %v", err)
	}

	// Write CRC (big-endian)
	if err := binary.Write(&buf, binary.BigEndian, chunk.CRC); err != nil {
		return nil, fmt.Errorf("failed to write chunk CRC: %v", err)
	}

	return buf.Bytes(), nil
}

// BuildPNGIHDRChunk creates an IHDR chunk from a structure
func BuildPNGIHDRChunk(ihdr PNGIHDRChunk) (PNGChunk, error) {
	// IHDR chunk validation
	if err := ValidatePNGIHDRChunk(&ihdr); err != nil {
		return PNGChunk{}, err
	}

	// Create IHDR chunk data (13 bytes)
	data := make([]byte, 13)

	// Width and height (big-endian)
	binary.BigEndian.PutUint32(data[0:4], ihdr.Width)
	binary.BigEndian.PutUint32(data[4:8], ihdr.Height)

	// Other fields
	data[8] = ihdr.BitDepth
	data[9] = ihdr.ColorType
	data[10] = ihdr.Compression
	data[11] = ihdr.Filter
	data[12] = ihdr.Interlace

	// Calculate CRC (using 0 for simplicity)
	crc := uint32(0) // In real code, this would be CRC32 calculation

	return PNGChunk{
		Length: 13,
		Type:   []byte("IHDR"),
		Data:   data,
		CRC:    crc,
	}, nil
}

// ParsePNGHeader parses a PNG header from a bitstring
func ParsePNGHeader(data *bitstring.BitString) (*PNGHeader, error) {
	var header PNGHeader

	// Parse signature
	_, err := matcher.NewMatcher().
		Binary(&header.Signature, bitstring.WithSize(8)).
		Match(data)

	if err != nil {
		return nil, fmt.Errorf("failed to parse PNG signature: %v", err)
	}

	// Check signature
	if !bytes.Equal(header.Signature, PNGSignature) {
		return nil, errors.New("invalid PNG signature")
	}

	// Get remaining data for chunk parsing
	remainingData := data.ToBytes()[8:]

	// Parse chunks
	for len(remainingData) >= 12 { // Minimum chunk size: 4 (len) + 4 (type) + 4 (crc)
		chunk, chunkSize, err := ParsePNGChunk(remainingData)
		if err != nil {
			return nil, fmt.Errorf("failed to parse PNG chunk: %v", err)
		}

		header.Chunks = append(header.Chunks, chunk)
		remainingData = remainingData[chunkSize:]

		// For simplicity, parse only the first chunk (IHDR)
		break
	}

	return &header, nil
}

// ParsePNGChunk parses a PNG chunk from bytes
func ParsePNGChunk(data []byte) (PNGChunk, uint, error) {
	if len(data) < 12 {
		return PNGChunk{}, 0, errors.New("insufficient data for PNG chunk")
	}

	// Read length (big-endian)
	length := binary.BigEndian.Uint32(data[0:4])

	// Read type
	chunkType := data[4:8]

	// Check if there's enough data
	totalChunkSize := 12 + int(length) // 4 (len) + 4 (type) + length + 4 (crc)
	if len(data) < totalChunkSize {
		return PNGChunk{}, 0, fmt.Errorf("insufficient data for chunk %s, need %d bytes, have %d",
			string(chunkType), totalChunkSize, len(data))
	}

	// Read data
	chunkData := data[8 : 8+length]

	// Read CRC (big-endian)
	crc := binary.BigEndian.Uint32(data[8+length : 12+length])

	chunk := PNGChunk{
		Length: length,
		Type:   chunkType,
		Data:   chunkData,
		CRC:    crc,
	}

	return chunk, uint(totalChunkSize), nil
}

// ParsePNGIHDRChunk parses an IHDR chunk from data
func ParsePNGIHDRChunk(data []byte) (*PNGIHDRChunk, error) {
	if len(data) < 13 {
		return nil, errors.New("insufficient data for IHDR chunk, need 13 bytes")
	}

	ihdr := &PNGIHDRChunk{
		Width:       binary.BigEndian.Uint32(data[0:4]),
		Height:      binary.BigEndian.Uint32(data[4:8]),
		BitDepth:    data[8],
		ColorType:   data[9],
		Compression: data[10],
		Filter:      data[11],
		Interlace:   data[12],
	}

	// Validation
	if err := ValidatePNGIHDRChunk(ihdr); err != nil {
		return nil, err
	}

	return ihdr, nil
}

// ValidatePNGHeader performs PNG header validation
func ValidatePNGHeader(header *PNGHeader) error {
	if header == nil {
		return errors.New("header is nil")
	}

	// Check signature
	if !bytes.Equal(header.Signature, PNGSignature) {
		return errors.New("invalid PNG signature")
	}

	// Check for chunks
	if len(header.Chunks) == 0 {
		return errors.New("PNG must have at least one chunk")
	}

	// First chunk must be IHDR
	if string(header.Chunks[0].Type) != "IHDR" {
		return errors.New("first PNG chunk must be IHDR")
	}

	// Validate each chunk
	for i, chunk := range header.Chunks {
		if err := ValidatePNGChunk(&chunk); err != nil {
			return fmt.Errorf("chunk %d (%s) validation failed: %v", i, string(chunk.Type), err)
		}
	}

	return nil
}

// ValidatePNGChunk performs PNG chunk validation
func ValidatePNGChunk(chunk *PNGChunk) error {
	if chunk == nil {
		return errors.New("chunk is nil")
	}

	if len(chunk.Type) != 4 {
		return errors.New("chunk type must be 4 bytes")
	}

	// Check that type consists of Latin alphabet letters
	for _, c := range chunk.Type {
		if (c < 'A' || c > 'Z') && (c < 'a' || c > 'z') {
			return fmt.Errorf("chunk type must contain only letters, got: %c", c)
		}
	}

	// Check data length
	if uint32(len(chunk.Data)) != chunk.Length {
		return fmt.Errorf("chunk data length mismatch, expected %d, got %d",
			chunk.Length, len(chunk.Data))
	}

	return nil
}

// ValidatePNGIHDRChunk performs IHDR chunk validation
func ValidatePNGIHDRChunk(ihdr *PNGIHDRChunk) error {
	if ihdr == nil {
		return errors.New("IHDR chunk is nil")
	}

	// Check width and height
	if ihdr.Width == 0 {
		return errors.New("image width must be greater than 0")
	}

	if ihdr.Height == 0 {
		return errors.New("image height must be greater than 0")
	}

	// Check color depth
	validBitDepths := []uint8{1, 2, 4, 8, 16}
	isValidBitDepth := false
	for _, bd := range validBitDepths {
		if ihdr.BitDepth == bd {
			isValidBitDepth = true
			break
		}
	}
	if !isValidBitDepth {
		return fmt.Errorf("invalid bit depth %d, must be one of: %v", ihdr.BitDepth, validBitDepths)
	}

	// Check color type
	validColorTypes := []uint8{0, 2, 3, 4, 6}
	isValidColorType := false
	for _, ct := range validColorTypes {
		if ihdr.ColorType == ct {
			isValidColorType = true
			break
		}
	}
	if !isValidColorType {
		return fmt.Errorf("invalid color type %d, must be one of: %v", ihdr.ColorType, validColorTypes)
	}

	// Check compatibility of color depth and color type
	if ihdr.ColorType == 3 && ihdr.BitDepth < 8 {
		return errors.New("indexed color (type 3) requires bit depth of 8")
	}

	// Check compression method (must be 0)
	if ihdr.Compression != 0 {
		return errors.New("compression method must be 0")
	}

	// Check filter method (must be 0)
	if ihdr.Filter != 0 {
		return errors.New("filter method must be 0")
	}

	// Check interlace method (must be 0 or 1)
	if ihdr.Interlace > 1 {
		return errors.New("interlace method must be 0 or 1")
	}

	return nil
}

// GetPNGColorTypeName returns the color type name
func GetPNGColorTypeName(colorType uint8) string {
	switch colorType {
	case 0:
		return "Grayscale"
	case 2:
		return "RGB"
	case 3:
		return "Indexed"
	case 4:
		return "Grayscale with Alpha"
	case 6:
		return "RGB with Alpha"
	default:
		return "Unknown"
	}
}

// GetPNGInterlaceMethodName returns the interlace method name
func GetPNGInterlaceMethodName(interlace uint8) string {
	switch interlace {
	case 0:
		return "None"
	case 1:
		return "Adam7"
	default:
		return "Unknown"
	}
}

// FormatPNGIHDRInfo formats IHDR chunk information for output
func FormatPNGIHDRInfo(ihdr *PNGIHDRChunk) string {
	var info strings.Builder

	info.WriteString(fmt.Sprintf("PNG Image Header:\n"))
	info.WriteString(fmt.Sprintf("  Width: %d px\n", ihdr.Width))
	info.WriteString(fmt.Sprintf("  Height: %d px\n", ihdr.Height))
	info.WriteString(fmt.Sprintf("  Bit Depth: %d bits\n", ihdr.BitDepth))
	info.WriteString(fmt.Sprintf("  Color Type: %d (%s)\n", ihdr.ColorType, GetPNGColorTypeName(ihdr.ColorType)))
	info.WriteString(fmt.Sprintf("  Compression: %d\n", ihdr.Compression))
	info.WriteString(fmt.Sprintf("  Filter: %d\n", ihdr.Filter))
	info.WriteString(fmt.Sprintf("  Interlace: %d (%s)\n", ihdr.Interlace, GetPNGInterlaceMethodName(ihdr.Interlace)))

	return info.String()
}
