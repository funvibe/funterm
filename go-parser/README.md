# Go Parser Framework

Гибкая и расширяемая система для создания парсеров на Go с поддержкой конфигурируемых обработчиков, приоритетов и защиты от рекурсии.

## 🆕 Новая функциональность: Парсинг вызовов функций других языков, циклов и pattern matching

Теперь система поддерживает:

### 1. Вызовы функций других языков

Формат: `language.function(arguments)`

### 2. Циклы Python-style

Формат: `for variable in iterable: body`

### 3. Циклы Lua-style

Формат: `for variable=start,end,step do body end`

### 4. Pattern Matching

Формат: `match expression { pattern -> statement, ... }`

Синтаксис вдохновлен Rust pattern matching с адаптацией для go-parser:
- Литеральные паттерны: `42`, `"hello"`
- Массивные паттерны: `[1, x, 3]`, `[head, ...tail]`
- Объектные паттерны: `{"status": "ok", "data": d}`
- Переменные паттерны: `x`
- Wildcard паттерны: `_`

### Примеры использования:

```go
package main

import (
    "fmt"
    "go-parser/pkg/parser"
)

func main() {
    p := parser.NewUnifiedParser()
    
    // Простые вызовы
    luaPrint, _ := p.Parse(`lua.print("hello")`)
    pythonSqrt, _ := p.Parse(`python.math.sqrt(4)`)
    
    // Множественные аргументы
    luaPrint2, _ := p.Parse(`lua.print("hello", "world")`)
    
    // Циклы Python-style
    forInLoop, _ := p.Parse(`for i in range(5): python.print(i)`)
    
    // Циклы Lua-style
    numericForLoop, _ := p.Parse(`for i=1,5 do lua.print(i) end`)
    
    // Pattern matching - базовый пример
    basicMatch, _ := p.Parse(`match value {
        42 -> lua.print("answer"),
        _ -> lua.print("default")
    }`)
    
    // Pattern matching - сложный пример
    complexMatch, _ := p.Parse(`match python.get_data() {
        [1, x, 3] -> lua.print("middle:", x),
        {"status": "ok", "data": d} -> lua.process_data(d),
        _ -> lua.print("unknown pattern")
    }`)
    
    fmt.Printf("%+v\n", luaPrint)
}
```

### Поддерживаемые форматы:

#### Вызовы функций:
- `lua.print("hello")` - вызов функции с строковым аргументом
- `python.math.sqrt(4)` - вызов функции с числовым аргументом
- `lua.print("hello", "world")` - множественные аргументы
- Поддержка вложенных путей функций: `python.module.submodule.function()`

#### Циклы Python-style:
- `for i in range(5): python.print(i)` - цикл с вызовом функции
- `for item in python.get_items(): lua.print(item)` - цикл с вызовом функции в качестве итератора

#### Циклы Lua-style:
- `for i=1,5 do lua.print(i) end` - простой числовой цикл
- `for i=1,10,2 do lua.print(i) end` - числовой цикл с шагом

#### Pattern Matching:
- `match value { 42 -> lua.print("answer") }` - литеральный паттерн
- `match data { [head, ...tail] -> lua.print(head, tail) }` - массивный паттерн с деструктуризацией
- `match response { {"status": "ok"} -> lua.print("success") }` - объектный паттерн
- `match value { x -> lua.print("got:", x) }` - переменный паттерн
- `match value { _ -> lua.print("default") }` - wildcard паттерн
- Поддержка сложных выражений: `match python.get_data() { [1, x, 3] -> lua.print(x) }`

## Особенности

- ✅ **Конфигурируемые обработчики** - множественные обработчики для одной конструкции с настройками приоритетов
- ✅ **Два типа парсеров** - итеративный и рекурсивный с настраиваемой защитой от рекурсии
- ✅ **Система синхронизации токенов** - обработчики могут получать все оставшиеся токены
- ✅ **Fallback механизм** - автоматическое переключение на резервные обработчики
- ✅ **Расширяемость** - легкое добавление новых типов токенов и обработчиков
- ✅ **Производительность** - оптимизации для высокой производительности

## Быстрый старт

```go
package main

import (
    "fmt"
    "log"
    
    "./pkg/lexer"
    "./pkg/stream"
    "./pkg/handler"
    "./pkg/parser"
)

func main() {
    // Создание компонентов
    registry := handler.NewHandlerRegistry()
    
    // Регистрация обработчика скобок
    parenHandler := &handler.ParenthesesHandler{
        Config: handler.HandlerConfig{
            IsEnabled: true,
            Priority:  100,
            Order:     1,
            Name:      "parentheses",
        },
    }
    
    registry.Register(lexer.TokenLeftParen, parenHandler)
    
    // Создание рекурсивного парсера
    parser := parser.NewRecursiveParser(registry)
    parser.SetMaxDepth(1000)
    
    // Парсинг
    result, err := parser.Parse("((()))")
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Результат: %+v\n", result.Value)
}
```

## Архитектура

Система состоит из следующих основных компонентов:

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│     Lexer       │───▶│   TokenStream    │───▶│     Parser      │
│                 │    │                  │    │                 │
│ - Токенизация   │    │ - Буферизация    │    │ - Рекурсивный   │
│ - Позиционирование│  │ - Синхронизация  │    │ - Итеративный   │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                                                         │
                                                         ▼
                       ┌──────────────────┐    ┌─────────────────┐
                       │ HandlerRegistry  │◀───│    Handlers     │
                       │                  │    │                 │
                       │ - Приоритеты     │    │ - Конфигурация  │
                       │ - Fallback       │    │ - Обработка     │
                       └──────────────────┘    └─────────────────┘
```

## Документация

- 📖 [**Архитектура системы**](ARCHITECTURE.md) - Детальное описание архитектуры и компонентов
- 🛠️ [**План реализации**](IMPLEMENTATION_PLAN.md) - Подробный план с примерами кода
- 📋 [**Примеры использования**](EXAMPLES.md) - Практические примеры и тестовые сценарии
- 🔄 [**Диаграммы последовательности**](SEQUENCE_DIAGRAMS.md) - Визуализация основных сценариев
- ⚡ [**Производительность и расширения**](PERFORMANCE_AND_EXTENSIONS.md) - Оптимизации и стратегии расширения
- 📚 [**API документация**](API_DOCUMENTATION.md) - Полное описание API
- 🎯 [**Pattern Matching Guide**](doc/PATTERN_MATCHING_GUIDE.md) - Полное руководство по pattern matching

## Основные компоненты

### 1. AST узлы для вызовов функций других языков, циклов и pattern matching (`pkg/ast`)

```go
// Узел вызова функции другого языка
type LanguageCall struct {
    Language  string       // "lua", "python"
    Function  string       // "print", "math.sqrt"
    Arguments []Expression // Аргументы функции
    Pos       Position     // Позиция в коде
}

// Цикл Python-style: for i in range(5): python.print(i)
type ForInLoopStatement struct {
    Variable  Expression   // Переменная цикла (i)
    Iterable  Expression   // Итерируемое выражение (range(5))
    Body      Statement    // Тело цикла
    Pos       Position     // Позиция в коде
}

// Цикл Lua-style: for i=1,5 do lua.print(i) end
type NumericForLoopStatement struct {
    Variable  Expression   // Переменная цикла (i)
    Start     Expression   // Начальное значение (1)
    End       Expression   // Конечное значение (5)
    Step      Expression   // Шаг (опционально)
    Body      Statement    // Тело цикла
    Pos       Position     // Позиция в коде
}

// Pattern matching: match expression { pattern -> statement, ... }
type MatchStatement struct {
    Expression  Expression   // Выражение для сопоставления
    Arms        []MatchArm   // Ветки сопоставления
    MatchToken  Token        // Токен 'match'
    LBraceToken Token        // Токен '{'
    RBraceToken Token        // Токен '}'
    Pos         Position     // Позиция в коде
}

// Ветка pattern matching: pattern -> statement
type MatchArm struct {
    Pattern    Pattern     // Паттерн для сопоставления
    ArrowToken Token       // Токен '->'
    Statement  Statement   // Выполняемый оператор
}

// Интерфейс для всех паттернов
type Pattern interface {
    ProtoNode
    patternMarker()
}

// Литеральный паттерн: 42, "hello"
type LiteralPattern struct {
    Value interface{} // Значение литерала
    Pos   Position    // Позиция в коде
}

// Массивный паттерн: [1, x, 3], [head, ...tail]
type ArrayPattern struct {
    Elements []Pattern // Элементы массива
    Rest     bool      // Наличие ...rest паттерна
    Pos      Position  // Позиция в коде
}

// Объектный паттерн: {"status": "ok", "data": d}
type ObjectPattern struct {
    Properties map[string]Pattern // Свойства объекта
    Pos        Position           // Позиция в коде
}

// Переменный паттерн: x
type VariablePattern struct {
    Name string    // Имя переменной
    Pos  Position  // Позиция в коде
}

// Wildcard паттерн: _
type WildcardPattern struct {
    Pos Position // Позиция в коде
}

// Строковый литерал
type StringLiteral struct {
    Value string
    Pos   Position
}

// Числовой литерал
type NumberLiteral struct {
    Value float64
    Pos   Position
}
```

### 2. Система токенов (`pkg/lexer`)

```go
type Token struct {
    Type     TokenType
    Value    string
    Position int
    Line     int
    Column   int
}

type Lexer interface {
    NextToken() Token
    Peek() Token
    HasMore() bool
}
```

### 2. Поток токенов (`pkg/stream`)

```go
type TokenStream interface {
    Current() Token
    Next() Token
    Consume() Token
    ConsumeAll() []Token
    Clone() TokenStream  // Для backtracking
}
```

### 3. Система обработчиков (`pkg/handler`)

```go
type HandlerConfig struct {
    IsEnabled        bool
    Priority         int    // Основной приоритет
    Order            int    // Порядок при одинаковом приоритете
    FallbackPriority int    // Приоритет для fallback
    IsFallback       bool
}

type Handler interface {
    CanHandle(token Token) bool
    Handle(ctx *ParseContext) (interface{}, error)
    Config() HandlerConfig
}
```

### 4. Парсеры (`pkg/parser`)

#### UnifiedParser - Новый API по ТЗ

```go
// Унифицированный парсер для вызовов функций других языков
type UnifiedParser struct {
    registry *handler.ConstructHandlerRegistryImpl
}

// Создает новый парсер
func NewUnifiedParser() *UnifiedParser

// Основной метод парсинга по ТЗ
func (p *UnifiedParser) Parse(input string) (ast.Statement, []ast.ParseError)
```

#### Legacy Parser API

```go
type Parser interface {
    Parse(input string) (*ParseResult, error)
    ParseTokens(stream TokenStream) (*ParseResult, error)
    SetMaxDepth(depth int)  // Защита от рекурсии
}
```

## Конфигурация обработчиков

Система поддерживает гибкую настройку обработчиков:

```go
// Основной обработчик с высоким приоритетом
mainHandler := &MyHandler{
    Config: HandlerConfig{
        IsEnabled: true,
        Priority:  100,
        Order:     1,
        Name:      "main-handler",
    },
}

// Fallback обработчик
fallbackHandler := &FallbackHandler{
    Config: HandlerConfig{
        IsEnabled:        true,
        Priority:         0,
        FallbackPriority: 50,
        IsFallback:       true,
        Name:             "fallback-handler",
    },
}
```

## Защита от рекурсии

Рекурсивный парсер включает настраиваемую защиту от бесконечной рекурсии:

```go
parser := parser.NewRecursiveParser(registry)
parser.SetMaxDepth(1000)  // Максимальная глубина рекурсии

// При превышении лимита будет возвращена ошибка:
// "recursion depth limit exceeded: 1000"
```

## Примеры использования

### Комментарии

Парсер поддерживает три стиля комментариев, которые полностью игнорируются при разборе кода:

#### Однострочные комментарии

**C++ стиль (`//`):**
```go
// Это однострочный комментарий
lua.print("hello") // Комментарий в конце строки
```

**Python стиль (`#`):**
```go
# Это однострочный комментарий
python.len("world") # Комментарий в конце строки
```

#### Многострочные комментарии

**C стиль (`/* ... */`):**
```go
/*
   Это многострочный комментарий
   Может занимать несколько строк
*/
lua.print(/* комментарий */ "hello")
```

#### Особенности работы комментариев

- **Комментарии игнорируются полностью** - парсер не видит содержимое комментариев
- **Встраивание в код** - комментарии могут располагаться в любом месте кода
- **Вложенность** - многострочные комментарии не поддерживают вложенность
- **Совместное использование** - все три стиля могут использоваться в одном файле

```go
// Основной алгоритм
for i = 1, 10 do
    lua.print(i) # Вывод числа
    /* Многострочный
       комментарий */
end
```

### Парсинг вызовов функций других языков

```go
// Создаем унифицированный парсер
p := parser.NewUnifiedParser()

// Парсинг вызовов функций
statement, errors := p.Parse(`lua.print("hello")`)

// Проверка ошибок
if len(errors) > 0 {
    log.Fatal(errors[0].Message)
}

// Работа с результатом
if call, ok := statement.(*ast.LanguageCall); ok {
    fmt.Printf("Language: %s\n", call.Language)
    fmt.Printf("Function: %s\n", call.Function)
    
    // Обработка аргументов
    for _, arg := range call.Arguments {
        switch a := arg.(type) {
        case *ast.StringLiteral:
            fmt.Printf("String: %s\n", a.Value)
        case *ast.NumberLiteral:
            fmt.Printf("Number: %f\n", a.Value)
        }
    }
}
```

### Парсинг циклов Python-style

```go
// Создаем унифицированный парсер
p := parser.NewUnifiedParser()

// Парсинг цикла for i in range(5): python.print(i)
statement, errors := p.Parse(`for i in range(5): python.print(i)`)

// Проверка ошибок
if len(errors) > 0 {
    log.Fatal(errors[0].Message)
}

// Работа с результатом
if forLoop, ok := statement.(*ast.ForInLoopStatement); ok {
    fmt.Printf("ForInLoop - Variable: %v\n", forLoop.Variable.ToMap())
    fmt.Printf("ForInLoop - Iterable: %v\n", forLoop.Iterable.ToMap())
    
    // Обработка тела цикла
    if body, ok := forLoop.Body.(*ast.LanguageCall); ok {
        fmt.Printf("Body - Language: %s\n", body.Language)
        fmt.Printf("Body - Function: %s\n", body.Function)
    }
}
```

### Парсинг циклов Lua-style

```go
// Создаем унифицированный парсер
p := parser.NewUnifiedParser()

// Парсинг цикла for i=1,5 do lua.print(i) end
statement, errors := p.Parse(`for i=1,5 do lua.print(i) end`)

// Проверка ошибок
if len(errors) > 0 {
    log.Fatal(errors[0].Message)
}

// Работа с результатом
if numForLoop, ok := statement.(*ast.NumericForLoopStatement); ok {
    fmt.Printf("NumericForLoop - Variable: %v\n", numForLoop.Variable.ToMap())
    fmt.Printf("NumericForLoop - Start: %v\n", numForLoop.Start.ToMap())
    fmt.Printf("NumericForLoop - End: %v\n", numForLoop.End.ToMap())
    
    // Обработка тела цикла
    if body, ok := numForLoop.Body.(*ast.LanguageCall); ok {
        fmt.Printf("Body - Language: %s\n", body.Language)
        fmt.Printf("Body - Function: %s\n", body.Function)
    }
}
```

### Парсинг Pattern Matching

```go
// Создаем унифицированный парсер
p := parser.NewUnifiedParser()

// Парсинг pattern matching с различными типами паттернов
statement, errors := p.Parse(`match python.get_data() {
    [1, x, 3] -> lua.print("middle:", x),
    {"status": "ok", "data": d} -> lua.process_data(d),
    _ -> lua.print("default")
}`)

// Проверка ошибок
if len(errors) > 0 {
    log.Fatal(errors[0].Message)
}

// Работа с результатом
if matchStmt, ok := statement.(*ast.MatchStatement); ok {
    fmt.Printf("MatchStatement - Expression: %v\n", matchStmt.Expression.ToMap())
    fmt.Printf("MatchStatement - Arms count: %d\n", len(matchStmt.Arms))
    
    // Обработка веток сопоставления
    for i, arm := range matchStmt.Arms {
        fmt.Printf("Arm %d - Pattern type: %T\n", i, arm.Pattern)
        fmt.Printf("Arm %d - Statement: %T\n", i, arm.Statement)
        
        // Анализ типа паттерна
        switch p := arm.Pattern.(type) {
        case *ast.LiteralPattern:
            fmt.Printf("  Literal value: %v\n", p.Value)
        case *ast.ArrayPattern:
            fmt.Printf("  Array elements: %d, rest: %v\n", len(p.Elements), p.Rest)
        case *ast.ObjectPattern:
            fmt.Printf("  Object properties: %d\n", len(p.Properties))
        case *ast.VariablePattern:
            fmt.Printf("  Variable name: %s\n", p.Name)
        case *ast.WildcardPattern:
            fmt.Printf("  Wildcard pattern\n")
        }
    }
}
```

### Базовый парсинг скобок

```go
input := "((()))"
result, err := parser.Parse(input)
// result.Value содержит структуру вложенных скобок
```

### Множественные обработчики

```go
registry.Register(TokenLeftParen, mainHandler)     // Priority: 100
registry.Register(TokenLeftParen, altHandler)      // Priority: 50
registry.Register(TokenLeftParen, fallbackHandler) // Fallback

// Будет использован mainHandler (наивысший приоритет)
```

### Работа с потоком токенов

```go
func (h *MyHandler) Handle(ctx *ParseContext) (interface{}, error) {
    // Получить все оставшиеся токены
    tokens := ctx.TokenStream.ConsumeAll()
    
    // Или работать по одному
    for ctx.TokenStream.HasMore() {
        token := ctx.TokenStream.Next()
        // обработка
    }
}
```

## Расширение функциональности

### Добавление новых типов токенов

```go
const (
    TokenString TokenType = iota + 100
    TokenNumber
    TokenIdentifier
)
```

### Создание кастомного обработчика

```go
type MyCustomHandler struct {
    config HandlerConfig
}

func (h *MyCustomHandler) CanHandle(token Token) bool {
    return token.Type == TokenString
}

func (h *MyCustomHandler) Handle(ctx *ParseContext) (interface{}, error) {
    // Защита от рекурсии
    if err := ctx.Guard.Enter(); err != nil {
        return nil, err
    }
    defer ctx.Guard.Exit()
    
    // Обработка токена
    token := ctx.TokenStream.Consume()
    return token.Value, nil
}
```

## Производительность

Система включает множество оптимизаций:

- **Буферизация токенов** - уменьшение количества обращений к лексеру
- **Кэширование обработчиков** - быстрый поиск лучшего обработчика
- **Пулы объектов** - переиспользование часто создаваемых структур
- **Предварительная сортировка** - избежание пересортировки при регистрации

## Мониторинг и отладка

```go
// Инструментированный парсер с метриками
instrumentedParser := parser.NewInstrumentedParser(baseParser)

// Получение метрик
metrics := instrumentedParser.GetMetrics()
fmt.Printf("Токенов обработано: %d\n", metrics.TokensProcessed)
fmt.Printf("Время парсинга: %v\n", metrics.ParseTime)
```

## Тестирование

Проект включает обширные тесты:

```bash
go test ./...                    # Запуск всех тестов
go test -bench=.                 # Бенчмарки производительности
go test -race                    # Проверка гонок данных
```

## Лицензия

MIT License - см. файл [LICENSE](LICENSE)

## Вклад в проект

Мы приветствуем вклад в развитие проекта! Пожалуйста, ознакомьтесь с [CONTRIBUTING.md](CONTRIBUTING.md) для получения информации о том, как внести свой вклад.

## Поддержка

- 📧 Email: support@go-parser.dev
- 💬 Discussions: [GitHub Discussions](https://github.com/your-org/go-parser/discussions)
- 🐛 Issues: [GitHub Issues](https://github.com/your-org/go-parser/issues)

---

**Go Parser Framework** - создавайте мощные и гибкие парсеры с легкостью! 🚀