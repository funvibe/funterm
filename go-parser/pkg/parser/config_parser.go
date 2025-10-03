package parser

import (
	"log"

	"go-parser/pkg/config"
	"go-parser/pkg/handler"
)

// ParserFromConfig creates parsers using the new configuration system
type ParserFromConfig struct {
	configLoader *config.ConfigLoader
}

// NewParserFromConfig creates a new parser factory with configuration
func NewParserFromConfig(configPaths []string) *ParserFromConfig {
	return &ParserFromConfig{
		configLoader: config.NewConfigLoader(configPaths),
	}
}

// NewParser creates a recursive parser using configuration from files
func (pfc *ParserFromConfig) NewParser() (*RecursiveParser, error) {
	// Load configuration from files
	parserConfig, err := pfc.configLoader.LoadConfig()
	if err != nil {
		return nil, err
	}

	return pfc.createRecursiveParser(parserConfig)
}

// NewIterativeParser creates an iterative parser using configuration from files
func (pfc *ParserFromConfig) NewIterativeParser() (*IterativeParser, error) {
	// Load configuration from files
	parserConfig, err := pfc.configLoader.LoadConfig()
	if err != nil {
		return nil, err
	}

	return pfc.createIterativeParser(parserConfig)
}

// NewParserWithConfig creates a recursive parser using the provided configuration
func (pfc *ParserFromConfig) NewParserWithConfig(parserConfig *config.ParserConfig) (*RecursiveParser, error) {
	// Validate configuration
	if err := config.ValidateConfig(parserConfig); err != nil {
		return nil, err
	}

	return pfc.createRecursiveParser(parserConfig)
}

// NewIterativeParserWithConfig creates an iterative parser using the provided configuration
func (pfc *ParserFromConfig) NewIterativeParserWithConfig(parserConfig *config.ParserConfig) (*IterativeParser, error) {
	// Validate configuration
	if err := config.ValidateConfig(parserConfig); err != nil {
		return nil, err
	}

	return pfc.createIterativeParser(parserConfig)
}

// createRecursiveParser creates a recursive parser with the given configuration
func (pfc *ParserFromConfig) createRecursiveParser(parserConfig *config.ParserConfig) (*RecursiveParser, error) {
	// Create registry
	registry := handler.NewConstructHandlerRegistry()

	// Create handler factory
	factory := handler.NewHandlerFactory()

	// Register handlers from configuration
	for _, handlerConfig := range parserConfig.ConstructHandlers {
		handler, err := factory.CreateHandler(handlerConfig)
		if err != nil {
			log.Printf("Failed to create handler %s: %v", handlerConfig.Name, err)
			continue
		}

		err = registry.RegisterConstructHandler(handler, handlerConfig)
		if err != nil {
			log.Printf("Failed to register handler %s: %v", handlerConfig.Name, err)
		}
	}

	// Create parser with configuration
	parser := NewRecursiveParser(registry)

	// Set max depth if recursion guard is enabled
	if parserConfig.EnableRecursionGuard {
		parser.SetMaxDepth(parserConfig.MaxDepth)
	}

	return parser, nil
}

// createIterativeParser creates an iterative parser with the given configuration
func (pfc *ParserFromConfig) createIterativeParser(parserConfig *config.ParserConfig) (*IterativeParser, error) {
	// Create registry
	registry := handler.NewConstructHandlerRegistry()

	// Create handler factory
	factory := handler.NewHandlerFactory()

	// Register handlers from configuration
	for _, handlerConfig := range parserConfig.ConstructHandlers {
		handler, err := factory.CreateHandler(handlerConfig)
		if err != nil {
			log.Printf("Failed to create handler %s: %v", handlerConfig.Name, err)
			continue
		}

		err = registry.RegisterConstructHandler(handler, handlerConfig)
		if err != nil {
			log.Printf("Failed to register handler %s: %v", handlerConfig.Name, err)
		}
	}

	// Create parser with configuration
	parser := NewIterativeParser(registry)

	// Set max depth if recursion guard is enabled
	if parserConfig.EnableRecursionGuard {
		parser.SetMaxDepth(parserConfig.MaxDepth)
	}

	return parser, nil
}

// LoadConfig loads and returns the parser configuration
func (pfc *ParserFromConfig) LoadConfig() (*config.ParserConfig, error) {
	return pfc.configLoader.LoadConfig()
}

// SaveConfig saves the parser configuration to a file
func (pfc *ParserFromConfig) SaveConfig(parserConfig *config.ParserConfig, configPath string) error {
	// Validate configuration before saving
	if err := config.ValidateConfig(parserConfig); err != nil {
		return err
	}

	return pfc.configLoader.SaveConfig(parserConfig, configPath)
}

// ValidateConfigFile validates a configuration file
func (pfc *ParserFromConfig) ValidateConfigFile(configPath string) error {
	return config.ValidateConfigFile(configPath)
}

// Convenience functions for creating parsers with default configurations

// NewRecursiveParserWithDefaultConfig creates a recursive parser with default configuration
func NewRecursiveParserWithDefaultConfig() (*RecursiveParser, error) {
	pfc := NewParserFromConfig([]string{})
	parserConfig := config.CreateDefaultParserConfig()
	return pfc.NewParserWithConfig(parserConfig)
}

// NewIterativeParserWithDefaultConfig creates an iterative parser with default configuration
func NewIterativeParserWithDefaultConfig() (*IterativeParser, error) {
	pfc := NewParserFromConfig([]string{})
	parserConfig := config.CreateDefaultParserConfig()
	return pfc.NewIterativeParserWithConfig(parserConfig)
}

// NewRecursiveParserWithMinimalConfig creates a recursive parser with minimal configuration
func NewRecursiveParserWithMinimalConfig() (*RecursiveParser, error) {
	pfc := NewParserFromConfig([]string{})
	parserConfig := config.CreateMinimalParserConfig()
	return pfc.NewParserWithConfig(parserConfig)
}

// NewIterativeParserWithMinimalConfig creates an iterative parser with minimal configuration
func NewIterativeParserWithMinimalConfig() (*IterativeParser, error) {
	pfc := NewParserFromConfig([]string{})
	parserConfig := config.CreateMinimalParserConfig()
	return pfc.NewIterativeParserWithConfig(parserConfig)
}

// NewRecursiveParserWithDevelopmentConfig creates a recursive parser with development configuration
func NewRecursiveParserWithDevelopmentConfig() (*RecursiveParser, error) {
	pfc := NewParserFromConfig([]string{})
	parserConfig := config.CreateDevelopmentParserConfig()
	return pfc.NewParserWithConfig(parserConfig)
}

// NewIterativeParserWithDevelopmentConfig creates an iterative parser with development configuration
func NewIterativeParserWithDevelopmentConfig() (*IterativeParser, error) {
	pfc := NewParserFromConfig([]string{})
	parserConfig := config.CreateDevelopmentParserConfig()
	return pfc.NewIterativeParserWithConfig(parserConfig)
}

// NewRecursiveParserFromFiles creates a recursive parser using configuration from default paths
func NewRecursiveParserFromFiles() (*RecursiveParser, error) {
	configPaths := config.GetDefaultConfigPaths()
	pfc := NewParserFromConfig(configPaths)
	return pfc.NewParser()
}

// NewIterativeParserFromFiles creates an iterative parser using configuration from default paths
func NewIterativeParserFromFiles() (*IterativeParser, error) {
	configPaths := config.GetDefaultConfigPaths()
	pfc := NewParserFromConfig(configPaths)
	return pfc.NewIterativeParser()
}
