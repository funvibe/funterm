package serialization

import (
	"errors"
	"fmt"
)

// StateSerializer defines the interface for serializing and deserializing state
type StateSerializer interface {
	// Serialize converts state data to bytes
	Serialize(data interface{}) ([]byte, error)

	// Deserialize converts bytes back to state data
	Deserialize(data []byte) (interface{}, error)

	// GetName returns the name of the serializer
	GetName() string

	// GetVersion returns the version of the serializer
	GetVersion() string

	// SupportsVersion checks if the serializer supports a specific version
	SupportsVersion(version string) bool
}

// VersionedState represents state data with version information
type VersionedState struct {
	Data    interface{} `json:"data"`
	Version string      `json:"version"`
	Format  string      `json:"format"`
}

// SerializationError represents an error that occurred during serialization
type SerializationError struct {
	Operation string
	Message   string
	Format    string
	Context   map[string]interface{}
}

func (e *SerializationError) Error() string {
	return fmt.Sprintf("[%s serialization error] %s", e.Format, e.Message)
}

// NewSerializationError creates a new serialization error
func NewSerializationError(format, operation, message string) *SerializationError {
	return &SerializationError{
		Format:    format,
		Operation: operation,
		Message:   message,
		Context:   make(map[string]interface{}),
	}
}

// WithContext adds context information to the error
func (e *SerializationError) WithContext(key string, value interface{}) *SerializationError {
	e.Context[key] = value
	return e
}

// SerializerRegistry manages multiple serializers
type SerializerRegistry struct {
	serializers       map[string]StateSerializer
	defaultSerializer string
}

// NewSerializerRegistry creates a new serializer registry
func NewSerializerRegistry() *SerializerRegistry {
	return &SerializerRegistry{
		serializers:       make(map[string]StateSerializer),
		defaultSerializer: "json",
	}
}

// RegisterSerializer registers a serializer
func (sr *SerializerRegistry) RegisterSerializer(serializer StateSerializer) error {
	name := serializer.GetName()
	if _, exists := sr.serializers[name]; exists {
		return fmt.Errorf("serializer '%s' is already registered", name)
	}

	sr.serializers[name] = serializer
	return nil
}

// GetSerializer returns a serializer by name
func (sr *SerializerRegistry) GetSerializer(name string) (StateSerializer, error) {
	serializer, exists := sr.serializers[name]
	if !exists {
		return nil, fmt.Errorf("serializer '%s' not found", name)
	}
	return serializer, nil
}

// GetDefaultSerializer returns the default serializer
func (sr *SerializerRegistry) GetDefaultSerializer() (StateSerializer, error) {
	if sr.defaultSerializer == "" {
		return nil, errors.New("no default serializer configured")
	}
	return sr.GetSerializer(sr.defaultSerializer)
}

// SetDefaultSerializer sets the default serializer
func (sr *SerializerRegistry) SetDefaultSerializer(name string) error {
	if _, exists := sr.serializers[name]; !exists {
		return fmt.Errorf("serializer '%s' not found", name)
	}

	sr.defaultSerializer = name
	return nil
}

// ListSerializers returns the names of all registered serializers
func (sr *SerializerRegistry) ListSerializers() []string {
	var names []string
	for name := range sr.serializers {
		names = append(names, name)
	}
	return names
}

// SerializeWithVersion serializes data with version information
func (sr *SerializerRegistry) SerializeWithVersion(data interface{}, format string) (*VersionedState, error) {
	serializer, err := sr.GetSerializer(format)
	if err != nil {
		return nil, err
	}

	serializedData, err := serializer.Serialize(data)
	if err != nil {
		return nil, NewSerializationError(format, "serialize", err.Error())
	}

	return &VersionedState{
		Data:    serializedData,
		Version: serializer.GetVersion(),
		Format:  format,
	}, nil
}

// DeserializeWithVersion deserializes versioned state data
func (sr *SerializerRegistry) DeserializeWithVersion(versionedState *VersionedState) (interface{}, error) {
	serializer, err := sr.GetSerializer(versionedState.Format)
	if err != nil {
		return nil, err
	}

	if !serializer.SupportsVersion(versionedState.Version) {
		return nil, NewSerializationError(versionedState.Format, "deserialize",
			fmt.Sprintf("version '%s' not supported", versionedState.Version))
	}

	data, ok := versionedState.Data.([]byte)
	if !ok {
		return nil, NewSerializationError(versionedState.Format, "deserialize",
			"invalid data type, expected []byte")
	}

	return serializer.Deserialize(data)
}

// ConvertFormat converts serialized data from one format to another
func (sr *SerializerRegistry) ConvertFormat(data []byte, fromFormat, toFormat string) ([]byte, error) {
	// Deserialize using source format
	fromSerializer, err := sr.GetSerializer(fromFormat)
	if err != nil {
		return nil, err
	}

	deserializedData, err := fromSerializer.Deserialize(data)
	if err != nil {
		return nil, NewSerializationError(fromFormat, "deserialize", err.Error())
	}

	// Serialize using target format
	toSerializer, err := sr.GetSerializer(toFormat)
	if err != nil {
		return nil, err
	}

	serializedData, err := toSerializer.Serialize(deserializedData)
	if err != nil {
		return nil, NewSerializationError(toFormat, "serialize", err.Error())
	}

	return serializedData, nil
}

// GetSupportedFormats returns all supported serialization formats
func (sr *SerializerRegistry) GetSupportedFormats() []string {
	return sr.ListSerializers()
}

// IsFormatSupported checks if a format is supported
func (sr *SerializerRegistry) IsFormatSupported(format string) bool {
	_, exists := sr.serializers[format]
	return exists
}
