package serialization

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

const (
	TypeNil     = 0
	TypeBool    = 1
	TypeInt64   = 2
	TypeFloat64 = 3
	TypeString  = 4
	TypeArray   = 5
	TypeMap     = 6
)

// BinarySerializer implements StateSerializer for custom binary format
// This is a schema-less binary serializer that supports nested data structures
type BinarySerializer struct {
	version string
}

// NewBinarySerializer creates a new binary serializer
func NewBinarySerializer() *BinarySerializer {
	return &BinarySerializer{
		version: "1.0.0",
	}
}

// GetName returns the name of the serializer
func (bs *BinarySerializer) GetName() string {
	return "binary"
}

// GetVersion returns the version of the serializer
func (bs *BinarySerializer) GetVersion() string {
	return bs.version
}

// SupportsVersion checks if the serializer supports a specific version
func (bs *BinarySerializer) SupportsVersion(version string) bool {
	// Currently only supports version 1.0.0
	// Future versions should be explicitly added as they are implemented
	return version == "1.0.0"
}

// Serialize converts data to binary format
func (bs *BinarySerializer) Serialize(data interface{}) ([]byte, error) {
	var buf bytes.Buffer
	if err := bs.encodeValue(&buf, 1, data); err != nil {
		return nil, NewSerializationError("binary", "serialize", err.Error())
	}

	return buf.Bytes(), nil
}

// Deserialize converts binary bytes back to data
func (bs *BinarySerializer) Deserialize(data []byte) (interface{}, error) {
	if len(data) == 0 {
		return nil, NewSerializationError("binary", "deserialize", "data is empty")
	}

	buf := bytes.NewBuffer(data)
	value, err := bs.decodeValue(buf)
	if err != nil {
		return nil, NewSerializationError("binary", "deserialize", err.Error())
	}

	return value, nil
}

// encodeValue encodes a value with type markers
func (bs *BinarySerializer) encodeValue(buf *bytes.Buffer, fieldNumber int, value interface{}) error {
	switch v := value.(type) {
	case nil:
		bs.encodeTypedValue(buf, fieldNumber, TypeNil, nil)
	case bool:
		bs.encodeTypedValue(buf, fieldNumber, TypeBool, v)
	case int, int8, int16, int32, int64:
		int64Val, err := bs.toInt64(v)
		if err != nil {
			return err
		}
		bs.encodeTypedValue(buf, fieldNumber, TypeInt64, int64Val)
	case uint, uint8, uint16, uint32, uint64:
		uint64Val, err := bs.toUint64(v)
		if err != nil {
			return err
		}
		bs.encodeTypedValue(buf, fieldNumber, TypeInt64, int64(uint64Val))
	case float32, float64:
		float64Val, err := bs.toFloat64(v)
		if err != nil {
			return err
		}
		bs.encodeTypedValue(buf, fieldNumber, TypeFloat64, float64Val)
	case string:
		bs.encodeTypedValue(buf, fieldNumber, TypeString, v)
	case []interface{}:
		bs.encodeTypedValue(buf, fieldNumber, TypeArray, v)
	case map[string]interface{}:
		bs.encodeTypedValue(buf, fieldNumber, TypeMap, v)
	default:
		return fmt.Errorf("unsupported type: %T", value)
	}
	return nil
}

// encodeTypedValue encodes a value with type information
func (bs *BinarySerializer) encodeTypedValue(buf *bytes.Buffer, fieldNumber int, valueType int, value interface{}) {
	// Write field tag
	tag := (fieldNumber << 3) | 2 // Length-delimited wire type
	buf.WriteByte(byte(tag))

	// Create payload with type marker
	var payload bytes.Buffer
	payload.WriteByte(byte(valueType)) // Type marker

	switch valueType {
	case TypeNil:
		// No additional data
	case TypeBool:
		if value.(bool) {
			payload.WriteByte(1)
		} else {
			payload.WriteByte(0)
		}
	case TypeInt64:
		bs.encodeVarint(&payload, uint64(value.(int64)))
	case TypeFloat64:
		_ = binary.Write(&payload, binary.LittleEndian, value.(float64))
	case TypeString:
		str := value.(string)
		bs.encodeVarint(&payload, uint64(len(str)))
		payload.WriteString(str)
	case TypeArray:
		arr := value.([]interface{})
		bs.encodeVarint(&payload, uint64(len(arr)))
		for _, item := range arr {
			_ = bs.encodeValue(&payload, 1, item) // Recursive encoding
		}
	case TypeMap:
		m := value.(map[string]interface{})
		bs.encodeVarint(&payload, uint64(len(m)))
		for key, val := range m {
			// Encode key
			_ = bs.encodeValue(&payload, 1, key)
			// Encode value
			_ = bs.encodeValue(&payload, 2, val)
		}
	}

	// Write payload length and payload
	bs.encodeVarint(buf, uint64(payload.Len()))
	buf.Write(payload.Bytes())
}

// decodeValue decodes a value with type markers
func (bs *BinarySerializer) decodeValue(buf *bytes.Buffer) (interface{}, error) {
	if buf.Len() == 0 {
		return nil, fmt.Errorf("unexpected end of data")
	}

	// Read tag
	tag, err := bs.decodeVarint(buf)
	if err != nil {
		return nil, err
	}

	wireType := int(tag & 0x7)
	if wireType != 2 { // Must be length-delimited
		return nil, fmt.Errorf("expected length-delimited wire type, got %d", wireType)
	}

	// Read payload length
	length, err := bs.decodeVarint(buf)
	if err != nil {
		return nil, err
	}

	if buf.Len() < int(length) {
		return nil, fmt.Errorf("unexpected end of data")
	}

	// Read payload
	payloadData := make([]byte, length)
	if _, err := buf.Read(payloadData); err != nil {
		return nil, err
	}

	payload := bytes.NewBuffer(payloadData)

	// Read type marker
	if payload.Len() == 0 {
		return nil, fmt.Errorf("missing type marker")
	}

	typeMarker, err := payload.ReadByte()
	if err != nil {
		return nil, err
	}

	return bs.decodeTypedValue(payload, int(typeMarker))
}

// decodeTypedValue decodes a value based on type marker
func (bs *BinarySerializer) decodeTypedValue(payload *bytes.Buffer, valueType int) (interface{}, error) {
	switch valueType {
	case TypeNil:
		return nil, nil
	case TypeBool:
		if payload.Len() == 0 {
			return false, fmt.Errorf("unexpected end of data")
		}
		b, err := payload.ReadByte()
		if err != nil {
			return false, err
		}
		return b != 0, nil
	case TypeInt64:
		value, err := bs.decodeVarint(payload)
		if err != nil {
			return nil, err
		}
		return int64(value), nil
	case TypeFloat64:
		if payload.Len() < 8 {
			return nil, fmt.Errorf("unexpected end of data")
		}
		var value float64
		if err := binary.Read(payload, binary.LittleEndian, &value); err != nil {
			return nil, err
		}
		return value, nil
	case TypeString:
		length, err := bs.decodeVarint(payload)
		if err != nil {
			return nil, err
		}
		if payload.Len() < int(length) {
			return nil, fmt.Errorf("unexpected end of data")
		}
		data := make([]byte, length)
		if _, err := payload.Read(data); err != nil {
			return nil, err
		}
		return string(data), nil
	case TypeArray:
		length, err := bs.decodeVarint(payload)
		if err != nil {
			return nil, err
		}
		arr := make([]interface{}, 0, length)
		for i := 0; i < int(length); i++ {
			item, err := bs.decodeValue(payload)
			if err != nil {
				return nil, err
			}
			arr = append(arr, item)
		}
		return arr, nil
	case TypeMap:
		length, err := bs.decodeVarint(payload)
		if err != nil {
			return nil, err
		}
		m := make(map[string]interface{})
		for i := 0; i < int(length); i++ {
			// Decode key
			keyVal, err := bs.decodeValue(payload)
			if err != nil {
				return nil, err
			}
			key, ok := keyVal.(string)
			if !ok {
				return nil, fmt.Errorf("map key must be string, got %T", keyVal)
			}

			// Decode value
			value, err := bs.decodeValue(payload)
			if err != nil {
				return nil, err
			}

			m[key] = value
		}
		return m, nil
	default:
		return nil, fmt.Errorf("unknown type marker: %d", valueType)
	}
}

// Helper methods for type conversion

func (bs *BinarySerializer) toInt64(v interface{}) (int64, error) {
	switch val := v.(type) {
	case int:
		return int64(val), nil
	case int8:
		return int64(val), nil
	case int16:
		return int64(val), nil
	case int32:
		return int64(val), nil
	case int64:
		return val, nil
	case uint:
		return int64(val), nil
	case uint8:
		return int64(val), nil
	case uint16:
		return int64(val), nil
	case uint32:
		return int64(val), nil
	default:
		return 0, fmt.Errorf("cannot convert %T to int64", v)
	}
}

func (bs *BinarySerializer) toUint64(v interface{}) (uint64, error) {
	switch val := v.(type) {
	case uint:
		return uint64(val), nil
	case uint8:
		return uint64(val), nil
	case uint16:
		return uint64(val), nil
	case uint32:
		return uint64(val), nil
	case uint64:
		return val, nil
	case int:
		if val < 0 {
			return 0, fmt.Errorf("cannot convert negative int %d to uint64", val)
		}
		return uint64(val), nil
	case int8:
		if val < 0 {
			return 0, fmt.Errorf("cannot convert negative int8 %d to uint64", val)
		}
		return uint64(val), nil
	case int16:
		if val < 0 {
			return 0, fmt.Errorf("cannot convert negative int16 %d to uint64", val)
		}
		return uint64(val), nil
	case int32:
		if val < 0 {
			return 0, fmt.Errorf("cannot convert negative int32 %d to uint64", val)
		}
		return uint64(val), nil
	case int64:
		if val < 0 {
			return 0, fmt.Errorf("cannot convert negative int64 %d to uint64", val)
		}
		return uint64(val), nil
	default:
		return 0, fmt.Errorf("cannot convert %T to uint64", v)
	}
}

func (bs *BinarySerializer) toFloat64(v interface{}) (float64, error) {
	switch val := v.(type) {
	case float32:
		return float64(val), nil
	case float64:
		return val, nil
	case int:
		return float64(val), nil
	case int8:
		return float64(val), nil
	case int16:
		return float64(val), nil
	case int32:
		return float64(val), nil
	case int64:
		return float64(val), nil
	case uint:
		return float64(val), nil
	case uint8:
		return float64(val), nil
	case uint16:
		return float64(val), nil
	case uint32:
		return float64(val), nil
	case uint64:
		return float64(val), nil
	default:
		return 0, fmt.Errorf("cannot convert %T to float64", v)
	}
}

// encodeVarint encodes a varint value
func (bs *BinarySerializer) encodeVarint(buf *bytes.Buffer, value uint64) {
	for value >= 0x80 {
		buf.WriteByte(byte(value) | 0x80)
		value >>= 7
	}
	buf.WriteByte(byte(value))
}

// decodeVarint decodes a varint value
func (bs *BinarySerializer) decodeVarint(buf *bytes.Buffer) (uint64, error) {
	var result uint64
	var shift uint

	for {
		if buf.Len() == 0 {
			return 0, fmt.Errorf("unexpected end of data")
		}

		b, err := buf.ReadByte()
		if err != nil {
			return 0, err
		}

		result |= uint64(b&0x7F) << shift
		shift += 7

		if (b & 0x80) == 0 {
			break
		}

		if shift >= 64 {
			return 0, fmt.Errorf("varint too long")
		}
	}

	return result, nil
}

// GetSizeEstimate returns an estimate of the serialized size without full serialization
func (bs *BinarySerializer) GetSizeEstimate(data interface{}) (int, error) {
	return bs.estimateSize(data)
}

// estimateSize recursively estimates the size of data structures
func (bs *BinarySerializer) estimateSize(data interface{}) (int, error) {
	if data == nil {
		// Type marker (1 byte) + tag (1 byte) + length (1 byte for varint) = 3 bytes
		return 3, nil
	}

	switch v := data.(type) {
	case bool:
		// Type marker (1 byte) + tag (1 byte) + length (1 byte) + value (1 byte) = 4 bytes
		return 4, nil
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		// Type marker (1 byte) + tag (1 byte) + length (1 byte) + varint (1-3 bytes for typical values)
		// Use a more realistic estimate instead of max size
		return 6, nil
	case float32, float64:
		// Type marker (1 byte) + tag (1 byte) + length (1 byte) + float64 (8 bytes) = 11 bytes
		return 11, nil
	case string:
		// Type marker (1 byte) + tag (1 byte) + length varint (1-5 bytes) + string length
		strLen := len(v)
		lengthVarintSize := bs.varintSize(uint64(strLen))
		return 3 + lengthVarintSize + strLen, nil
	case []interface{}:
		// Type marker (1 byte) + tag (1 byte) + length varint (1-5 bytes) + array elements
		arrLen := len(v)
		lengthVarintSize := bs.varintSize(uint64(arrLen))
		totalSize := 3 + lengthVarintSize

		for _, item := range v {
			itemSize, err := bs.estimateSize(item)
			if err != nil {
				return 0, err
			}
			totalSize += itemSize
		}
		return totalSize, nil
	case map[string]interface{}:
		// Type marker (1 byte) + tag (1 byte) + length varint (1-5 bytes) + map entries
		mapLen := len(v)
		lengthVarintSize := bs.varintSize(uint64(mapLen))
		totalSize := 3 + lengthVarintSize

		for key, value := range v {
			// Estimate key size
			keySize, err := bs.estimateSize(key)
			if err != nil {
				return 0, err
			}
			// Estimate value size
			valueSize, err := bs.estimateSize(value)
			if err != nil {
				return 0, err
			}
			totalSize += keySize + valueSize
		}
		return totalSize, nil
	default:
		return 0, fmt.Errorf("unsupported type for size estimation: %T", data)
	}
}

// varintSize returns the number of bytes needed to encode a value as varint
func (bs *BinarySerializer) varintSize(value uint64) int {
	switch {
	case value < 1<<7:
		return 1
	case value < 1<<14:
		return 2
	case value < 1<<21:
		return 3
	case value < 1<<28:
		return 4
	case value < 1<<35:
		return 5
	case value < 1<<42:
		return 6
	case value < 1<<49:
		return 7
	case value < 1<<56:
		return 8
	case value < 1<<63:
		return 9
	default:
		return 10
	}
}
