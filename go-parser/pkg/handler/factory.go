package handler

import (
	"fmt"

	"go-parser/pkg/common"
	"go-parser/pkg/config"
)

// HandlerFactory - фабрика для создания обработчиков
type HandlerFactory interface {
	CreateHandler(config config.ConstructHandlerConfig) (common.Handler, error)
}

// HandlerFactoryImpl - реализация фабрики
type HandlerFactoryImpl struct {
	constructFactories map[common.ConstructType]func(config config.ConstructHandlerConfig) (common.Handler, error)
	languageRegistry   LanguageRegistry
}

// NewHandlerFactory - создает новую фабрику
func NewHandlerFactory() *HandlerFactoryImpl {
	factory := &HandlerFactoryImpl{
		constructFactories: make(map[common.ConstructType]func(config config.ConstructHandlerConfig) (common.Handler, error)),
		languageRegistry:   CreateDefaultLanguageRegistry(),
	}

	// Регистрируем фабрики для разных типов конструкций
	factory.RegisterFactory(common.ConstructLiteral, factory.createLiteralHandler)
	factory.RegisterFactory(common.ConstructVariable, factory.createVariableHandler)
	factory.RegisterFactory(common.ConstructTuple, factory.createTupleHandler)
	factory.RegisterFactory(common.ConstructFunction, factory.createFunctionHandler)
	factory.RegisterFactory(common.ConstructGroup, factory.createGroupHandler)
	factory.RegisterFactory(common.ConstructArray, factory.createArrayHandler)
	factory.RegisterFactory(common.ConstructObject, factory.createObjectHandler)
	factory.RegisterFactory(common.ConstructAssignment, factory.createAssignmentHandler)
	factory.RegisterFactory(common.ConstructIdentifierRead, factory.createIdentifierReadHandler)
	factory.RegisterFactory(common.ConstructPipe, factory.createPipeHandler)
	factory.RegisterFactory(common.ConstructLanguageCall, factory.createLanguageCallHandler)
	factory.RegisterFactory(common.ConstructIf, factory.createIfHandler)
	factory.RegisterFactory(common.ConstructForInLoop, factory.createForInLoopHandler)
	factory.RegisterFactory(common.ConstructCodeBlock, factory.createCodeBlockHandler)
	factory.RegisterFactory(common.ConstructBinaryExpression, factory.createTernaryHandler)
	factory.RegisterFactory(common.ConstructMatch, factory.createMatchHandler)

	return factory
}

// RegisterFactory - регистрирует фабрику для типа конструкции
func (f *HandlerFactoryImpl) RegisterFactory(
	constructType common.ConstructType,
	factoryFunc func(config config.ConstructHandlerConfig) (common.Handler, error),
) {
	f.constructFactories[constructType] = factoryFunc
}

// CreateHandler - создает обработчик по конфигурации
func (f *HandlerFactoryImpl) CreateHandler(
	config config.ConstructHandlerConfig,
) (common.Handler, error) {
	factoryFunc, exists := f.constructFactories[config.ConstructType]
	if !exists {
		return nil, fmt.Errorf("no factory registered for construct type: %s", config.ConstructType)
	}

	return factoryFunc(config)
}

// Фабричные методы для разных типов конструкций

func (f *HandlerFactoryImpl) createLiteralHandler(
	config config.ConstructHandlerConfig,
) (common.Handler, error) {
	return NewLiteralHandler(config), nil
}

func (f *HandlerFactoryImpl) createVariableHandler(
	config config.ConstructHandlerConfig,
) (common.Handler, error) {
	return NewVariableHandler(config), nil
}

func (f *HandlerFactoryImpl) createTupleHandler(
	config config.ConstructHandlerConfig,
) (common.Handler, error) {
	return NewTupleHandler(config), nil
}

func (f *HandlerFactoryImpl) createFunctionHandler(
	config config.ConstructHandlerConfig,
) (common.Handler, error) {
	return NewFunctionHandler(config), nil
}

func (f *HandlerFactoryImpl) createGroupHandler(
	config config.ConstructHandlerConfig,
) (common.Handler, error) {
	return NewGroupHandler(config), nil
}

func (f *HandlerFactoryImpl) createArrayHandler(
	config config.ConstructHandlerConfig,
) (common.Handler, error) {
	// Создаем обработчик массива с настройками из конфигурации
	priority := config.Priority
	if priority == 0 {
		priority = 10 // Приоритет по умолчанию
	}
	order := config.Order
	if order == 0 {
		order = 1 // Порядок по умолчанию
	}
	return NewArrayHandler(priority, order), nil
}

func (f *HandlerFactoryImpl) createObjectHandler(
	config config.ConstructHandlerConfig,
) (common.Handler, error) {
	// Создаем обработчик объекта с настройками из конфигурации
	priority := config.Priority
	if priority == 0 {
		priority = 10 // Приоритет по умолчанию
	}
	order := config.Order
	if order == 0 {
		order = 1 // Порядок по умолчанию
	}
	return NewObjectHandler(priority, order), nil
}

func (f *HandlerFactoryImpl) createAssignmentHandler(
	config config.ConstructHandlerConfig,
) (common.Handler, error) {
	// Создаем обработчик присваивания с настройками из конфигурации
	priority := config.Priority
	if priority == 0 {
		priority = 5 // Приоритет по умолчанию
	}
	order := config.Order
	if order == 0 {
		order = 1 // Порядок по умолчанию
	}
	return NewAssignmentHandler(priority, order), nil
}

func (f *HandlerFactoryImpl) createIdentifierReadHandler(
	config config.ConstructHandlerConfig,
) (common.Handler, error) {
	// Для чтения идентификаторов используем тот же обработчик, что и для присваивания
	priority := config.Priority
	if priority == 0 {
		priority = 5 // Приоритет по умолчанию
	}
	order := config.Order
	if order == 0 {
		order = 2 // Порядок после присваивания
	}
	return NewAssignmentHandler(priority, order), nil
}

func (f *HandlerFactoryImpl) createPipeHandler(
	config config.ConstructHandlerConfig,
) (common.Handler, error) {
	// Создаем обработчик pipe expressions с настройками из конфигурации
	priority := config.Priority
	if priority == 0 {
		priority = 1 // Самый низкий приоритет по умолчанию
	}
	order := config.Order
	if order == 0 {
		order = 1 // Порядок по умолчанию
	}

	// Обновляем приоритет и порядок в конфигурации
	config.Priority = priority
	config.Order = order

	return NewPipeHandler(config), nil
}

func (f *HandlerFactoryImpl) createLanguageCallStatementHandler(
	config config.ConstructHandlerConfig,
) (common.Handler, error) {
	// Создаем обработчик language call statements с настройками из конфигурации
	priority := config.Priority
	if priority == 0 {
		priority = 50 // Средний приоритет по умолчанию
	}
	order := config.Order
	if order == 0 {
		order = 1 // Порядок по умолчанию
	}

	// Обновляем приоритет и порядок в конфигурации
	config.Priority = priority
	config.Order = order

	return NewLanguageCallStatementHandlerWithRegistry(config, f.languageRegistry), nil
}

func (f *HandlerFactoryImpl) createLanguageCallHandler(
	config config.ConstructHandlerConfig,
) (common.Handler, error) {
	// Создаем обработчик language call expressions с настройками из конфигурации
	priority := config.Priority
	if priority == 0 {
		priority = 60 // Высокий приоритет по умолчанию для выражений
	}
	order := config.Order
	if order == 0 {
		order = 1 // Порядок по умолчанию
	}

	// Обновляем приоритет и порядок в конфигурации
	config.Priority = priority
	config.Order = order

	handler := NewLanguageCallHandlerWithRegistry(config, f.languageRegistry)
	return handler, nil
}

func (f *HandlerFactoryImpl) createIfHandler(
	config config.ConstructHandlerConfig,
) (common.Handler, error) {
	// Создаем обработчик if/else конструкций с настройками из конфигурации
	priority := config.Priority
	if priority == 0 {
		priority = 30 // Высокий приоритет по умолчанию для control flow
	}
	order := config.Order
	if order == 0 {
		order = 1 // Порядок по умолчанию
	}

	// Обновляем приоритет и порядок в конфигурации
	config.Priority = priority
	config.Order = order

	return NewIfHandler(config), nil
}

func (f *HandlerFactoryImpl) createForInLoopHandler(
	config config.ConstructHandlerConfig,
) (common.Handler, error) {
	// Создаем обработчик for-in циклов с настройками из конфигурации
	priority := config.Priority
	if priority == 0 {
		priority = 30 // Высокий приоритет по умолчанию для control flow
	}
	order := config.Order
	if order == 0 {
		order = 1 // Порядок по умолчанию
	}

	// Обновляем приоритет и порядок в конфигурации
	config.Priority = priority
	config.Order = order

	return NewForInLoopHandler(config), nil
}

func (f *HandlerFactoryImpl) createCodeBlockHandler(
	config config.ConstructHandlerConfig,
) (common.Handler, error) {
	// Создаем обработчик кодовых блоков с настройками из конфигурации
	priority := config.Priority
	if priority == 0 {
		priority = 40 // Высокий приоритет по умолчанию для кодовых блоков
	}
	order := config.Order
	if order == 0 {
		order = 1 // Порядок по умолчанию
	}

	// Обновляем приоритет и порядок в конфигурации
	config.Priority = priority
	config.Order = order

	return NewCodeBlockHandler(config), nil
}

func (f *HandlerFactoryImpl) createTernaryHandler(
	config config.ConstructHandlerConfig,
) (common.Handler, error) {
	// Создаем обработчик тернарных операторов с настройками из конфигурации
	priority := config.Priority
	if priority == 0 {
		priority = 70 // Высокий приоритет по умолчанию для тернарных операторов
	}
	order := config.Order
	if order == 0 {
		order = 1 // Порядок по умолчанию
	}

	// Обновляем приоритет и порядок в конфигурации
	config.Priority = priority
	config.Order = order

	return NewTernaryHandler(config), nil
}

func (f *HandlerFactoryImpl) createMatchHandler(
	config config.ConstructHandlerConfig,
) (common.Handler, error) {
	// Создаем обработчик pattern matching с настройками из конфигурации
	priority := config.Priority
	if priority == 0 {
		priority = 85 // Высокий приоритет по умолчанию для pattern matching
	}
	order := config.Order
	if order == 0 {
		order = 1 // Порядок по умолчанию
	}

	// Обновляем приоритет и порядок в конфигурации
	config.Priority = priority
	config.Order = order

	return NewMatchHandler(config), nil
}
