package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config represents the simple application configuration
type Config struct {
	REPL      REPLConfig      `json:"repl" yaml:"repl"`
	Engine    EngineConfig    `json:"engine" yaml:"engine"`
	Logging   LoggingConfig   `json:"logging" yaml:"logging"`
	Languages LanguagesConfig `json:"languages" yaml:"languages"`
}

// REPLConfig contains REPL configuration
type REPLConfig struct {
	Prompt      string `json:"prompt" yaml:"prompt"`
	HistorySize int    `json:"history_size" yaml:"history_size"`
	HistoryFile string `json:"history_file" yaml:"history_file"`
	ShowWelcome bool   `json:"show_welcome" yaml:"show_welcome"`
}

// EngineConfig contains execution engine configuration
type EngineConfig struct {
	MaxExecutionTime int  `json:"max_execution_time_seconds" yaml:"max_execution_time_seconds"`
	Verbose          bool `json:"verbose" yaml:"verbose"`
}

// LoggingConfig contains logging configuration
type LoggingConfig struct {
	Level string `json:"level" yaml:"level"`
}

// LanguagesConfig contains language-specific configuration
type LanguagesConfig struct {
	Disabled []string                 `json:"disabled" yaml:"disabled"`
	Runtimes map[string]RuntimeConfig `json:"runtimes" yaml:"runtimes"`
}

// RuntimeConfig contains runtime-specific configuration
type RuntimeConfig struct {
	Path string `json:"path,omitempty" yaml:"path,omitempty"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		REPL: REPLConfig{
			Prompt:      "> ",
			HistorySize: 1000,
			HistoryFile: "/tmp/funterm_history",
			ShowWelcome: true,
		},
		Engine: EngineConfig{
			MaxExecutionTime: 30,
			Verbose:          false,
		},
		Logging: LoggingConfig{
			Level: "info",
		},
		Languages: LanguagesConfig{
			Disabled: []string{}, // По умолчанию все языки включены
			Runtimes: map[string]RuntimeConfig{
				// Пути указываются только если нужны нестандартные
				// По умолчанию используются: python3, lua, node, go
			},
		},
	}
}

// LoadConfig loads configuration from a file
func LoadConfig(path string) (*Config, error) {
	// Start with default config
	config := DefaultConfig()

	// If no path provided, return default config
	if path == "" {
		return config, nil
	}

	// Expand ~ in path
	path = expandHome(path)

	// Read the config file
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, return default config
			return config, nil
		}
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	// Determine file format by extension
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".json":
		if err := json.Unmarshal(data, config); err != nil {
			return nil, fmt.Errorf("failed to parse JSON config: %v", err)
		}
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, config); err != nil {
			return nil, fmt.Errorf("failed to parse YAML config: %v", err)
		}
	default:
		// Try YAML as default
		if err := yaml.Unmarshal(data, config); err != nil {
			return nil, fmt.Errorf("failed to parse config file: %v", err)
		}
	}

	return config, nil
}

// SaveConfig saves configuration to a file
func SaveConfig(config *Config, path string) error {
	// Expand ~ in path
	path = expandHome(path)

	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %v", err)
	}

	// Determine file format by extension
	ext := strings.ToLower(filepath.Ext(path))
	var data []byte
	var err error

	switch ext {
	case ".json":
		data, err = json.MarshalIndent(config, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON config: %v", err)
		}
	case ".yaml", ".yml":
		data, err = yaml.Marshal(config)
		if err != nil {
			return fmt.Errorf("failed to marshal YAML config: %v", err)
		}
	default:
		// Default to YAML
		data, err = yaml.Marshal(config)
		if err != nil {
			return fmt.Errorf("failed to marshal YAML config: %v", err)
		}
	}

	// Write the config file
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %v", err)
	}

	return nil
}

// expandHome expands ~ to the user's home directory
func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}

// IsLanguageDisabled checks if a language is disabled in config
func (c *Config) IsLanguageDisabled(language string) bool {
	for _, disabled := range c.Languages.Disabled {
		if disabled == language {
			return true
		}
	}
	return false
}

// GetRuntimePath returns the path for a specific runtime
func (c *Config) GetRuntimePath(language string) string {
	if runtime, exists := c.Languages.Runtimes[language]; exists && runtime.Path != "" {
		return runtime.Path
	}

	// Default paths
	switch language {
	case "python", "py":
		return "python3"
	case "lua":
		return "lua"
	case "node", "js":
		return "node"
	case "go":
		return "go"
	default:
		return language
	}
}
