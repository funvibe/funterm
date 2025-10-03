package serialization

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// MessagePackSerializer implements StateSerializer for MessagePack format
// This is a simplified implementation that supports basic MessagePack features
type MessagePackSerializer struct {
	version string
}

// NewMessagePackSerializer creates a new MessagePack serializer
func NewMessagePackSerializer() *MessagePackSerializer {
	return &MessagePackSerializer{
		version: "1.0.0",
	}
}

// Serialize converts data to MessagePack bytes
func (mps *MessagePackSerializer) Serialize(data interface{}) ([]byte, error) {
	if data == nil {
		return nil, NewSerializationError("msgpack", "serialize", "data is nil")
	}

	var buf bytes.Buffer
	if err := mps.encodeValue(&buf, data); err != nil {
		return nil, NewSerializationError("msgpack", "serialize", err.Error())
	}

	return buf.Bytes(), nil
}

// Deserialize converts MessagePack bytes back to data
func (mps *MessagePackSerializer) Deserialize(data []byte) (interface{}, error) {
	if len(data) == 0 {
		return nil, NewSerializationError("msgpack", "deserialize", "data is empty")
	}

	buf := bytes.NewBuffer(data)
	value, err := mps.decodeValue(buf)
	if err != nil {
		return nil, NewSerializationError("msgpack", "deserialize", err.Error())
	}

	return value, nil
}

// GetName returns the name of the serializer
func (mps *MessagePackSerializer) GetName() string {
	return "msgpack"
}

// GetVersion returns the version of the serializer
func (mps *MessagePackSerializer) GetVersion() string {
	return mps.version
}

// SupportsVersion checks if the serializer supports a specific version
func (mps *MessagePackSerializer) SupportsVersion(version string) bool {
	// For MessagePack, we support all 1.x.x versions
	return version == "1.0.0" || (len(version) > 2 && version[:2] == "1.")
}

// encodeValue encodes a value to MessagePack format
func (mps *MessagePackSerializer) encodeValue(buf *bytes.Buffer, value interface{}) error {
	switch v := value.(type) {
	case nil:
		buf.WriteByte(0xC0) // nil
	case bool:
		if v {
			buf.WriteByte(0xC3) // true
		} else {
			buf.WriteByte(0xC2) // false
		}
	case int:
		mps.encodeInt(buf, int64(v))
	case int8:
		mps.encodeInt(buf, int64(v))
	case int16:
		mps.encodeInt(buf, int64(v))
	case int32:
		mps.encodeInt(buf, int64(v))
	case int64:
		mps.encodeInt(buf, v)
	case uint:
		mps.encodeUint(buf, uint64(v))
	case uint8:
		mps.encodeUint(buf, uint64(v))
	case uint16:
		mps.encodeUint(buf, uint64(v))
	case uint32:
		mps.encodeUint(buf, uint64(v))
	case uint64:
		mps.encodeUint(buf, v)
	case float32:
		mps.encodeFloat(buf, float64(v))
	case float64:
		mps.encodeFloat(buf, v)
	case string:
		mps.encodeString(buf, v)
	case []interface{}:
		mps.encodeArray(buf, v)
	case map[string]interface{}:
		mps.encodeMap(buf, v)
	default:
		return fmt.Errorf("unsupported type: %T", value)
	}
	return nil
}

// encodeInt encodes an integer value
func (mps *MessagePackSerializer) encodeInt(buf *bytes.Buffer, value int64) {
	switch {
	case value >= -32 && value <= 127:
		buf.WriteByte(byte(value))
	case value >= -128 && value <= 127:
		buf.WriteByte(0xD0)
		buf.WriteByte(byte(value))
	case value >= -32768 && value <= 32767:
		buf.WriteByte(0xD1)
		_ = binary.Write(buf, binary.BigEndian, int16(value))
	case value >= -2147483648 && value <= 2147483647:
		buf.WriteByte(0xD2)
		_ = binary.Write(buf, binary.BigEndian, int32(value))
	default:
		buf.WriteByte(0xD3)
		_ = binary.Write(buf, binary.BigEndian, value)
	}
}

// encodeUint encodes an unsigned integer value
func (mps *MessagePackSerializer) encodeUint(buf *bytes.Buffer, value uint64) {
	switch {
	case value <= 127:
		buf.WriteByte(byte(value))
	case value <= 255:
		buf.WriteByte(0xCC)
		buf.WriteByte(byte(value))
	case value <= 65535:
		buf.WriteByte(0xCD)
		if err := binary.Write(buf, binary.BigEndian, uint16(value)); err != nil {
			// This shouldn't happen with bytes.Buffer, but handle it gracefully
			panic(fmt.Errorf("failed to write uint16: %w", err))
		}
	case value <= 4294967295:
		buf.WriteByte(0xCE)
		if err := binary.Write(buf, binary.BigEndian, uint32(value)); err != nil {
			// This shouldn't happen with bytes.Buffer, but handle it gracefully
			panic(fmt.Errorf("failed to write uint32: %w", err))
		}
	default:
		buf.WriteByte(0xCF)
		if err := binary.Write(buf, binary.BigEndian, value); err != nil {
			// This shouldn't happen with bytes.Buffer, but handle it gracefully
			panic(fmt.Errorf("failed to write uint64: %w", err))
		}
	}
}

// encodeFloat encodes a float value
func (mps *MessagePackSerializer) encodeFloat(buf *bytes.Buffer, value float64) {
	buf.WriteByte(0xCB)
	if err := binary.Write(buf, binary.BigEndian, value); err != nil {
		// This shouldn't happen with bytes.Buffer, but handle it gracefully
		panic(fmt.Errorf("failed to write float64: %w", err))
	}
}

// encodeString encodes a string value
func (mps *MessagePackSerializer) encodeString(buf *bytes.Buffer, value string) {
	length := len(value)
	switch {
	case length < 32:
		buf.WriteByte(0xA0 | byte(length))
	case length <= 255:
		buf.WriteByte(0xD9)
		buf.WriteByte(byte(length))
	case length <= 65535:
		buf.WriteByte(0xDA)
		if err := binary.Write(buf, binary.BigEndian, uint16(length)); err != nil {
			// This shouldn't happen with bytes.Buffer, but handle it gracefully
			panic(fmt.Errorf("failed to write uint16 length: %w", err))
		}
	default:
		buf.WriteByte(0xDB)
		if err := binary.Write(buf, binary.BigEndian, uint32(length)); err != nil {
			// This shouldn't happen with bytes.Buffer, but handle it gracefully
			panic(fmt.Errorf("failed to write uint32 length: %w", err))
		}
	}
	buf.WriteString(value)
}

// encodeArray encodes an array value
func (mps *MessagePackSerializer) encodeArray(buf *bytes.Buffer, value []interface{}) {
	length := len(value)
	switch {
	case length < 16:
		buf.WriteByte(0x90 | byte(length))
	case length <= 65535:
		buf.WriteByte(0xDC)
		if err := binary.Write(buf, binary.BigEndian, uint16(length)); err != nil {
			// This shouldn't happen with bytes.Buffer, but handle it gracefully
			panic(fmt.Errorf("failed to write uint16 array length: %w", err))
		}
	default:
		buf.WriteByte(0xDD)
		if err := binary.Write(buf, binary.BigEndian, uint32(length)); err != nil {
			// This shouldn't happen with bytes.Buffer, but handle it gracefully
			panic(fmt.Errorf("failed to write uint32 array length: %w", err))
		}
	}
	for _, item := range value {
		if err := mps.encodeValue(buf, item); err != nil {
			// This shouldn't happen in normal circumstances
			panic(err)
		}
	}
}

// encodeMap encodes a map value
func (mps *MessagePackSerializer) encodeMap(buf *bytes.Buffer, value map[string]interface{}) {
	length := len(value)
	switch {
	case length < 16:
		buf.WriteByte(0x80 | byte(length))
	case length <= 65535:
		buf.WriteByte(0xDE)
		binary.Write(buf, binary.BigEndian, uint16(length))
	default:
		buf.WriteByte(0xDF)
		if err := binary.Write(buf, binary.BigEndian, uint32(length)); err != nil {
			// This shouldn't happen with bytes.Buffer, but handle it gracefully
			panic(fmt.Errorf("failed to write uint32 map length: %w", err))
		}
	}
	for key, val := range value {
		if err := mps.encodeValue(buf, key); err != nil {
			panic(err)
		}
		if err := mps.encodeValue(buf, val); err != nil {
			panic(err)
		}
	}
}

// decodeValue decodes a value from MessagePack format
func (mps *MessagePackSerializer) decodeValue(buf *bytes.Buffer) (interface{}, error) {
	if buf.Len() == 0 {
		return nil, fmt.Errorf("unexpected end of data")
	}

	b, err := buf.ReadByte()
	if err != nil {
		return nil, err
	}

	switch {
	case b == 0xC0: // nil
		return nil, nil
	case b == 0xC2: // false
		return false, nil
	case b == 0xC3: // true
		return true, nil
	case b >= 0xE0 && b <= 0xFF: // negative fixint
		return int64(int8(b)), nil
	case b >= 0x00 && b <= 0x7F: // positive fixint
		return int64(b), nil
	case b == 0xCC: // uint8
		var val uint8
		if err := binary.Read(buf, binary.BigEndian, &val); err != nil {
			return nil, err
		}
		return uint64(val), nil
	case b == 0xCD: // uint16
		var val uint16
		if err := binary.Read(buf, binary.BigEndian, &val); err != nil {
			return nil, err
		}
		return uint64(val), nil
	case b == 0xCE: // uint32
		var val uint32
		if err := binary.Read(buf, binary.BigEndian, &val); err != nil {
			return nil, err
		}
		return uint64(val), nil
	case b == 0xCF: // uint64
		var val uint64
		if err := binary.Read(buf, binary.BigEndian, &val); err != nil {
			return nil, err
		}
		return val, nil
	case b == 0xD0: // int8
		var val int8
		if err := binary.Read(buf, binary.BigEndian, &val); err != nil {
			return nil, err
		}
		return int64(val), nil
	case b == 0xD1: // int16
		var val int16
		if err := binary.Read(buf, binary.BigEndian, &val); err != nil {
			return nil, err
		}
		return int64(val), nil
	case b == 0xD2: // int32
		var val int32
		if err := binary.Read(buf, binary.BigEndian, &val); err != nil {
			return nil, err
		}
		return int64(val), nil
	case b == 0xD3: // int64
		var val int64
		if err := binary.Read(buf, binary.BigEndian, &val); err != nil {
			return nil, err
		}
		return val, nil
	case b == 0xCB: // float64
		var val float64
		if err := binary.Read(buf, binary.BigEndian, &val); err != nil {
			return nil, err
		}
		return val, nil
	case b >= 0xA0 && b <= 0xBF: // fixstr
		length := int(b & 0x1F)
		return mps.decodeString(buf, length)
	case b == 0xD9: // str8
		length, err := buf.ReadByte()
		if err != nil {
			return nil, err
		}
		return mps.decodeString(buf, int(length))
	case b == 0xDA: // str16
		var length uint16
		if err := binary.Read(buf, binary.BigEndian, &length); err != nil {
			return nil, err
		}
		return mps.decodeString(buf, int(length))
	case b == 0xDB: // str32
		var length uint32
		if err := binary.Read(buf, binary.BigEndian, &length); err != nil {
			return nil, err
		}
		return mps.decodeString(buf, int(length))
	case b >= 0x90 && b <= 0x9F: // fixarray
		length := int(b & 0x0F)
		return mps.decodeArray(buf, length)
	case b == 0xDC: // array16
		var length uint16
		if err := binary.Read(buf, binary.BigEndian, &length); err != nil {
			return nil, err
		}
		return mps.decodeArray(buf, int(length))
	case b == 0xDD: // array32
		var length uint32
		if err := binary.Read(buf, binary.BigEndian, &length); err != nil {
			return nil, err
		}
		return mps.decodeArray(buf, int(length))
	case b >= 0x80 && b <= 0x8F: // fixmap
		length := int(b & 0x0F)
		return mps.decodeMap(buf, length)
	case b == 0xDE: // map16
		var length uint16
		if err := binary.Read(buf, binary.BigEndian, &length); err != nil {
			return nil, err
		}
		return mps.decodeMap(buf, int(length))
	case b == 0xDF: // map32
		var length uint32
		if err := binary.Read(buf, binary.BigEndian, &length); err != nil {
			return nil, err
		}
		return mps.decodeMap(buf, int(length))
	default:
		return nil, fmt.Errorf("unknown MessagePack format: 0x%02X", b)
	}
}

// decodeString decodes a string value
func (mps *MessagePackSerializer) decodeString(buf *bytes.Buffer, length int) (string, error) {
	if buf.Len() < length {
		return "", fmt.Errorf("unexpected end of data")
	}
	data := make([]byte, length)
	if _, err := buf.Read(data); err != nil {
		return "", err
	}
	return string(data), nil
}

// decodeArray decodes an array value
func (mps *MessagePackSerializer) decodeArray(buf *bytes.Buffer, length int) ([]interface{}, error) {
	result := make([]interface{}, length)
	for i := 0; i < length; i++ {
		val, err := mps.decodeValue(buf)
		if err != nil {
			return nil, err
		}
		result[i] = val
	}
	return result, nil
}

// decodeMap decodes a map value
func (mps *MessagePackSerializer) decodeMap(buf *bytes.Buffer, length int) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	for i := 0; i < length; i++ {
		keyVal, err := mps.decodeValue(buf)
		if err != nil {
			return nil, err
		}
		key, ok := keyVal.(string)
		if !ok {
			return nil, fmt.Errorf("map key must be string, got %T", keyVal)
		}
		val, err := mps.decodeValue(buf)
		if err != nil {
			return nil, err
		}
		result[key] = val
	}
	return result, nil
}
