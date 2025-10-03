package parser

import (
	"log"

	"go-parser/pkg/config"
	"go-parser/pkg/handler"
)

// NewParserWithConfig - создает парсер с конфигурацией из Go-структур
func NewParserWithConfig() *RecursiveParser {
	// Создаем реестр
	registry := handler.NewConstructHandlerRegistry()

	// Создаем фабрику обработчиков
	factory := handler.NewHandlerFactory()

	// Регистрируем обработчики из конфигурации
	for _, handlerConfig := range config.HandlersConfig {
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

	return NewRecursiveParser(registry)
}

// NewIterativeParserWithConfig - создает итеративный парсер с конфигурацией из Go-структур
func NewIterativeParserWithConfig() *IterativeParser {
	// Создаем реестр
	registry := handler.NewConstructHandlerRegistry()

	// Создаем фабрику обработчиков
	factory := handler.NewHandlerFactory()

	// Регистрируем обработчики из конфигурации
	for _, handlerConfig := range config.HandlersConfig {
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

	return NewIterativeParser(registry)
}
