package serialization

// NewDefaultSerializerRegistry creates a serializer registry with all default serializers
func NewDefaultSerializerRegistry() *SerializerRegistry {
	registry := NewSerializerRegistry()

	// Register JSON serializer
	jsonSerializer := NewJSONSerializer()
	if err := registry.RegisterSerializer(jsonSerializer); err != nil {
		// Log error but continue
	}

	// Register MessagePack serializer
	msgpackSerializer := NewMessagePackSerializer()
	if err := registry.RegisterSerializer(msgpackSerializer); err != nil {
		// Log error but continue
	}

	// Register Binary serializer (replaces protobuf)
	binarySerializer := NewBinarySerializer()
	if err := registry.RegisterSerializer(binarySerializer); err != nil {
		// Log error but continue
	}

	// Set JSON as default
	if err := registry.SetDefaultSerializer("json"); err != nil {
		// Log error but continue
	}

	return registry
}

// GetSerializer returns a serializer by name from the default registry
func GetSerializer(name string) (StateSerializer, error) {
	registry := NewDefaultSerializerRegistry()
	return registry.GetSerializer(name)
}

// Serialize serializes data using the specified format
func Serialize(data interface{}, format string) ([]byte, error) {
	registry := NewDefaultSerializerRegistry()
	serializer, err := registry.GetSerializer(format)
	if err != nil {
		return nil, err
	}
	return serializer.Serialize(data)
}

// Deserialize deserializes data using the specified format
func Deserialize(data []byte, format string) (interface{}, error) {
	registry := NewDefaultSerializerRegistry()
	serializer, err := registry.GetSerializer(format)
	if err != nil {
		return nil, err
	}
	return serializer.Deserialize(data)
}

// ConvertFormat converts data from one format to another
func ConvertFormat(data []byte, fromFormat, toFormat string) ([]byte, error) {
	registry := NewDefaultSerializerRegistry()
	return registry.ConvertFormat(data, fromFormat, toFormat)
}

// GetSupportedFormats returns all supported serialization formats
func GetSupportedFormats() []string {
	registry := NewDefaultSerializerRegistry()
	return registry.GetSupportedFormats()
}

// IsFormatSupported checks if a format is supported
func IsFormatSupported(format string) bool {
	registry := NewDefaultSerializerRegistry()
	return registry.IsFormatSupported(format)
}
