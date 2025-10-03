package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

// ConfigFormat represents the format of the configuration file
type ConfigFormat string

const (
	FormatJSON ConfigFormat = "json"
	FormatYAML ConfigFormat = "yaml"
)

// ParserConfig represents the overall parser configuration
type ParserConfig struct {
	MaxDepth                int                      `json:"maxDepth" yaml:"maxDepth"`
	EnableRecursionGuard    bool                     `json:"enableRecursionGuard" yaml:"enableRecursionGuard"`
	DefaultFallbackPriority int                      `json:"defaultFallbackPriority" yaml:"defaultFallbackPriority"`
	ConstructHandlers       []ConstructHandlerConfig `json:"constructHandlers" yaml:"constructHandlers"`
	CustomSettings          map[string]interface{}   `json:"customSettings" yaml:"customSettings"`
}

// ConfigLoader handles loading configuration from JSON/YAML files
type ConfigLoader struct {
	configPaths []string
}

// NewConfigLoader creates a new configuration loader
func NewConfigLoader(configPaths []string) *ConfigLoader {
	return &ConfigLoader{
		configPaths: configPaths,
	}
}

// LoadConfig loads configuration from the specified paths
func (cl *ConfigLoader) LoadConfig() (*ParserConfig, error) {
	var config *ParserConfig

	// If no config paths provided, use default configuration
	if len(cl.configPaths) == 0 {
		config = cl.CreateDefaultConfig()
	} else {
		config = &ParserConfig{
			MaxDepth:                100,
			EnableRecursionGuard:    true,
			DefaultFallbackPriority: 10,
			CustomSettings:          make(map[string]interface{}),
		}

		// Load configurations from all paths in order (later ones override earlier ones)
		configFound := false
		for _, configPath := range cl.configPaths {
			if _, err := os.Stat(configPath); err == nil {
				if err := cl.loadConfigFromFile(config, configPath); err != nil {
					return nil, fmt.Errorf("failed to load config from %s: %w", configPath, err)
				}
				configFound = true
			}
		}

		// If no config files were found, use default configuration
		if !configFound {
			config = cl.CreateDefaultConfig()
		}
	}

	// Validate the loaded configuration
	if err := ValidateConfig(config); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return config, nil
}

// loadConfigFromFile loads configuration from a single file
func (cl *ConfigLoader) loadConfigFromFile(config *ParserConfig, configPath string) error {
	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("config file not found: %s", configPath)
	}

	// Read file content
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Determine file format by extension
	format := cl.detectFormat(configPath)

	// Parse based on format
	switch format {
	case FormatJSON:
		if err := json.Unmarshal(data, config); err != nil {
			return fmt.Errorf("failed to parse JSON config: %w", err)
		}
	case FormatYAML:
		if err := yaml.Unmarshal(data, config); err != nil {
			return fmt.Errorf("failed to parse YAML config: %w", err)
		}
	default:
		return fmt.Errorf("unsupported config format: %s", format)
	}

	return nil
}

// detectFormat detects the configuration file format by extension
func (cl *ConfigLoader) detectFormat(filePath string) ConfigFormat {
	ext := filepath.Ext(filePath)
	switch ext {
	case ".json":
		return FormatJSON
	case ".yaml", ".yml":
		return FormatYAML
	default:
		return FormatJSON // Default to JSON
	}
}

// SaveConfig saves configuration to a file
func (cl *ConfigLoader) SaveConfig(config *ParserConfig, configPath string) error {
	// Determine file format by extension
	format := cl.detectFormat(configPath)

	var data []byte
	var err error

	// Serialize based on format
	switch format {
	case FormatJSON:
		data, err = json.MarshalIndent(config, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON config: %w", err)
		}
	case FormatYAML:
		data, err = yaml.Marshal(config)
		if err != nil {
			return fmt.Errorf("failed to marshal YAML config: %w", err)
		}
	default:
		return fmt.Errorf("unsupported config format: %s", format)
	}

	// Write to file
	if err := ioutil.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// CreateDefaultConfig creates a default configuration
func (cl *ConfigLoader) CreateDefaultConfig() *ParserConfig {
	return &ParserConfig{
		MaxDepth:                100,
		EnableRecursionGuard:    true,
		DefaultFallbackPriority: 10,
		ConstructHandlers:       GetDefaultHandlerConfigs(),
		CustomSettings: map[string]interface{}{
			"enableDebugMode": false,
			"logLevel":        "info",
		},
	}
}

// GetDefaultConfigPaths returns default configuration file paths
func GetDefaultConfigPaths() []string {
	return []string{
		"config/parser.json",
		"config/parser.yaml",
		"config/parser.yml",
		".parser.json",
		".parser.yaml",
		".parser.yml",
	}
}
