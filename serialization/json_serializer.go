package serialization

import (
	"encoding/json"
)

// JSONSerializer implements StateSerializer for JSON format
type JSONSerializer struct {
	version string
}

// NewJSONSerializer creates a new JSON serializer
func NewJSONSerializer() *JSONSerializer {
	return &JSONSerializer{
		version: "1.0.0",
	}
}

// Serialize converts data to JSON bytes
func (js *JSONSerializer) Serialize(data interface{}) ([]byte, error) {
	if data == nil {
		return nil, NewSerializationError("json", "serialize", "data is nil")
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, NewSerializationError("json", "serialize", err.Error())
	}

	return jsonData, nil
}

// Deserialize converts JSON bytes back to data
func (js *JSONSerializer) Deserialize(data []byte) (interface{}, error) {
	if len(data) == 0 {
		return nil, NewSerializationError("json", "deserialize", "data is empty")
	}

	var result interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, NewSerializationError("json", "deserialize", err.Error())
	}

	return result, nil
}

// GetName returns the name of the serializer
func (js *JSONSerializer) GetName() string {
	return "json"
}

// GetVersion returns the version of the serializer
func (js *JSONSerializer) GetVersion() string {
	return js.version
}

// SupportsVersion checks if the serializer supports a specific version
func (js *JSONSerializer) SupportsVersion(version string) bool {
	// For JSON, we support all 1.x.x versions
	return version == "1.0.0" || (len(version) > 2 && version[:2] == "1.")
}

// DeserializeToMap deserializes JSON bytes to a map[string]interface{}
func (js *JSONSerializer) DeserializeToMap(data []byte) (map[string]interface{}, error) {
	if len(data) == 0 {
		return nil, NewSerializationError("json", "deserialize", "data is empty")
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, NewSerializationError("json", "deserialize", err.Error())
	}

	return result, nil
}

// DeserializeToSlice deserializes JSON bytes to a []interface{}
func (js *JSONSerializer) DeserializeToSlice(data []byte) ([]interface{}, error) {
	if len(data) == 0 {
		return nil, NewSerializationError("json", "deserialize", "data is empty")
	}

	var result []interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, NewSerializationError("json", "deserialize", err.Error())
	}

	return result, nil
}

// SerializePretty serializes data to JSON with pretty formatting
func (js *JSONSerializer) SerializePretty(data interface{}) ([]byte, error) {
	if data == nil {
		return nil, NewSerializationError("json", "serialize", "data is nil")
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return nil, NewSerializationError("json", "serialize", err.Error())
	}

	return jsonData, nil
}

// ValidateJSON validates JSON data
func (js *JSONSerializer) ValidateJSON(data []byte) error {
	if len(data) == 0 {
		return NewSerializationError("json", "validate", "data is empty")
	}

	var result interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return NewSerializationError("json", "validate", err.Error())
	}

	return nil
}

// GetJSONSchema returns a JSON schema for validation (placeholder implementation)
func (js *JSONSerializer) GetJSONSchema() map[string]interface{} {
	return map[string]interface{}{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type":    "object",
		"properties": map[string]interface{}{
			"data": map[string]interface{}{
				"type": "object",
			},
			"version": map[string]interface{}{
				"type":    "string",
				"pattern": "^\\d+\\.\\d+\\.\\d+$",
			},
			"format": map[string]interface{}{
				"type": "string",
				"enum": []string{"json"},
			},
		},
		"required": []string{"data", "version", "format"},
	}
}
