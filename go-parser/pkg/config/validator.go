package config

import (
	"fmt"
	"os"
	"reflect"
)

// ValidationError represents a configuration validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error in field '%s': %s", e.Field, e.Message)
}

// ValidateConfig validates the parser configuration
func ValidateConfig(config *ParserConfig) error {
	if config == nil {
		return &ValidationError{
			Field:   "config",
			Message: "configuration cannot be nil",
		}
	}

	// Validate MaxDepth
	if config.MaxDepth <= 0 {
		return &ValidationError{
			Field:   "maxDepth",
			Message: "maxDepth must be positive",
		}
	}

	if config.MaxDepth > 1000 {
		return &ValidationError{
			Field:   "maxDepth",
			Message: "maxDepth is too large (max 1000)",
		}
	}

	// Validate DefaultFallbackPriority
	if config.DefaultFallbackPriority < 0 {
		return &ValidationError{
			Field:   "defaultFallbackPriority",
			Message: "defaultFallbackPriority cannot be negative",
		}
	}

	// Validate ConstructHandlers
	if len(config.ConstructHandlers) == 0 {
		return &ValidationError{
			Field:   "constructHandlers",
			Message: "at least one construct handler must be defined",
		}
	}

	// Validate each handler
	for i, handler := range config.ConstructHandlers {
		if err := validateHandler(handler, i); err != nil {
			return err
		}
	}

	// Validate handler uniqueness (by name)
	handlerNames := make(map[string]bool)
	for i, handler := range config.ConstructHandlers {
		if handlerNames[handler.Name] {
			return &ValidationError{
				Field:   fmt.Sprintf("constructHandlers[%d].name", i),
				Message: fmt.Sprintf("duplicate handler name: %s", handler.Name),
			}
		}
		handlerNames[handler.Name] = true
	}

	return nil
}

// validateHandler validates a single construct handler configuration
func validateHandler(handler ConstructHandlerConfig, index int) error {
	fieldPrefix := fmt.Sprintf("constructHandlers[%d]", index)

	// Validate Name
	if handler.Name == "" {
		return &ValidationError{
			Field:   fieldPrefix + ".name",
			Message: "handler name cannot be empty",
		}
	}

	// Validate ConstructType
	if handler.ConstructType == "" {
		return &ValidationError{
			Field:   fieldPrefix + ".constructType",
			Message: "constructType cannot be empty",
		}
	}

	// Validate ConstructType is valid
	validConstructTypes := map[string]bool{
		"literal":          true,
		"function":         true,
		"variable":         true,
		"group":            true,
		"tuple":            true,
		"match":            true,
		"for_in_loop":      true,
		"numeric_for_loop": true,
		"if":               true,
		"code_block":       true,
		"language_call":    true,
	}
	if !validConstructTypes[string(handler.ConstructType)] {
		return &ValidationError{
			Field:   fieldPrefix + ".constructType",
			Message: fmt.Sprintf("invalid constructType: %s", handler.ConstructType),
		}
	}

	// Validate Priority
	if handler.Priority < -1000 || handler.Priority > 1000 {
		return &ValidationError{
			Field:   fieldPrefix + ".priority",
			Message: "priority must be between -1000 and 1000",
		}
	}

	// Validate Order
	if handler.Order < 0 {
		return &ValidationError{
			Field:   fieldPrefix + ".order",
			Message: "order cannot be negative",
		}
	}

	// Validate FallbackPriority
	if handler.FallbackPriority < 0 {
		return &ValidationError{
			Field:   fieldPrefix + ".fallbackPriority",
			Message: "fallbackPriority cannot be negative",
		}
	}

	// Validate TokenPatterns
	if len(handler.TokenPatterns) == 0 {
		return &ValidationError{
			Field:   fieldPrefix + ".tokenPatterns",
			Message: "at least one token pattern must be defined",
		}
	}

	// Validate each token pattern
	for j, pattern := range handler.TokenPatterns {
		if err := validateTokenPattern(pattern, fieldPrefix, j); err != nil {
			return err
		}
	}

	// Validate CustomParams (if present)
	if handler.CustomParams != nil {
		if err := validateCustomParams(handler.CustomParams, fieldPrefix); err != nil {
			return err
		}
	}

	return nil
}

// validateTokenPattern validates a single token pattern
func validateTokenPattern(pattern TokenPattern, fieldPrefix string, index int) error {
	patternFieldPrefix := fmt.Sprintf("%s.tokenPatterns[%d]", fieldPrefix, index)

	// Validate TokenType
	if pattern.TokenType <= 0 {
		return &ValidationError{
			Field:   patternFieldPrefix + ".tokenType",
			Message: "tokenType must be a positive integer",
		}
	}

	// Validate Offset
	if pattern.Offset < 0 {
		return &ValidationError{
			Field:   patternFieldPrefix + ".offset",
			Message: "offset cannot be negative",
		}
	}

	if pattern.Offset > 10 {
		return &ValidationError{
			Field:   patternFieldPrefix + ".offset",
			Message: "offset is too large (max 10)",
		}
	}

	return nil
}

// validateCustomParams validates custom parameters
func validateCustomParams(params map[string]interface{}, fieldPrefix string) error {
	for key, value := range params {
		paramField := fieldPrefix + ".customParams." + key

		// Check for nil values
		if value == nil {
			return &ValidationError{
				Field:   paramField,
				Message: "custom parameter value cannot be nil",
			}
		}

		// Check for supported types
		switch v := value.(type) {
		case string, int, int64, float64, bool:
			// These are supported types
		case map[string]interface{}:
			// Recursively validate nested maps
			if err := validateCustomParams(v, paramField); err != nil {
				return err
			}
		case []interface{}:
			// Validate array elements
			for i, elem := range v {
				if elem == nil {
					return &ValidationError{
						Field:   fmt.Sprintf("%s[%d]", paramField, i),
						Message: "array element cannot be nil",
					}
				}
			}
		default:
			return &ValidationError{
				Field:   paramField,
				Message: fmt.Sprintf("unsupported type for custom parameter: %s", reflect.TypeOf(v)),
			}
		}
	}

	return nil
}

// ValidateConfigFile validates a configuration file without loading it
func ValidateConfigFile(configPath string) error {
	// First check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("config file not found: %s", configPath)
	}

	loader := NewConfigLoader([]string{configPath})
	config, err := loader.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config for validation: %w", err)
	}

	if err := ValidateConfig(config); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	return nil
}
