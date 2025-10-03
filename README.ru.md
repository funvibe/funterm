# 🚀 funterm: Мультиязыковой REPL и процессор бинарных данных

**Бесшовное объединение Python, Lua, JavaScript и Go с продвинутым парсингом битовых строк, межъязыковыми пайпами и сложными возможностями обработки данных.**

funterm интегрирует экосистему Python, скорость Lua, универсальность JavaScript и производительность Go в единую среду скриптования с операциями битовых строк в стиле Erlang и сопоставлением шаблонов.

## ✨ Ключевые возможности

- 🛠️ **Мультиязыковая поддержка**: Python, Lua, JavaScript (Node.js), Go
- 🔥 **Парсинг битовых строк**: Парсинг бинарных данных в стиле Erlang с синтаксисом `<< >>`
- ⚡ **Inplace сопоставление шаблонов**: Прямое извлечение переменных с `<<pattern>> = value`
- 🔄 **Межъязыковые пайпы**: Цепочки функций между языками с оператором `|`
- ⚡ **Фоновое выполнение**: Неблокирующие задачи с оператором `&`
- 🎯 **Продвинутое сопоставление шаблонов**: Деструктуризация данных с guards и wildcards
- 🔧 **Конкатенация строк**: Оператор `++` с автоматическим приведением типов
- 📦 **Контролируемое сохранение переменных**: Явное управление состоянием с `runtime (vars) { ... }`

## 🏃‍♂️ Быстрый старт

### Скачать готовые бинарники
Скачайте последний релиз для вашей платформы с [GitHub Releases](../../releases):
- **Linux**: `funterm-linux-amd64.tar.gz` или `funterm-linux-arm64.tar.gz`
- **macOS**: `funterm-darwin-amd64.tar.gz` или `funterm-darwin-arm64.tar.gz`
- **Windows**: `funterm-windows-amd64.zip` или `funterm-windows-arm64.zip`
- **FreeBSD/OpenBSD**: `funterm-freebsd-amd64.tar.gz` или `funterm-openbsd-amd64.tar.gz`

### Сборка из исходников
```bash
# Клонировать и собрать
git clone https://github.com/funvibe/funterm.git
cd funterm
go build -o funterm main.go config.go batch.go

# Запуск интерактивного режима
./funterm

# Выполнение скрипта
./funterm -exec script.su
```

## 🎯 Практические примеры

### 🔥 Парсинг битовых строк
```erlang
# Парсинг бинарных протоколов
lua.packet = <<0xDEADBEEF:32, 256:16/little, "payload"/binary, 0x1234:16>>

match lua.packet {
    <<header:32, size:16/little, data/binary, checksum:16>> -> {
        lua.print("Заголовок:", lua.string.format("0x%08X", header))
        lua.print("Размер:", size, "байт")
        lua.print("Данные:", data)
    }
}

# Обработка UTF-8 строк
lua.utf8_data = <<"Привет 🚀"/utf8>>
match lua.utf8_data {
    <<"Привет", emoji:32/utf8>> -> lua.print("Найден эмодзи:", emoji)
}
```

### ⚡ Inplace сопоставление шаблонов
```erlang
# Прямое присваивание шаблона (в стиле Erlang)
lua.source = "Hello"

# Извлечение переменных напрямую из шаблона
if (<<h/utf8, e/utf8, l1/utf8, l2/utf8, o/utf8>> = lua.source) {
    lua.print("Символы:", h, e, l1, l2, o)  # Вывод: 72 101 108 108 111
}

# Использование квалифицированных переменных для сохранения результатов
lua.data = "Hi"
<<lua.first/utf8, lua.second/utf8>> = lua.data
lua.print("Квалифицированный шаблон:", lua.first, lua.second)  # Вывод: 72 105

# Парсинг бинарных протоколов
lua.packet = <<0xAA:8, 12:4, 0x55:8>>
<<header:8, id:4, footer:8>> = lua.packet
lua.print("Заголовок:", header, "ID:", id, "Концовка:", footer)
```

### 🔄 Межъязыковые пайпы
```python
# Мультиязыковой конвейер обработки данных
py (process_data) {
    def process_data(text):
        return text.upper().strip()
}

lua (add_prefix) {
    function add_prefix(text)
        return "ОБРАБОТАНО: " .. text
    end
}

js (up) {
    function up(s) {
        return s.toUpperCase();
    }
}

# Цепочка операций между языками
py.greeting = "привет мир"
lua.result = js.up(py.greeting)
lua.print(lua.result)  # Вывод: ПРИВЕТ МИР
```

### 🔧 Конкатенация строк
```python
# Конкатенация строк с автоматическим приведением типов
py.name = "Алиса"
py.age = 30
py.score = 85.5

# Число + Строка = Строка
lua.message = py.age ++ " лет"
lua.print(lua.message)  # Вывод: "30 лет"

# Строка + Число = Строка  
lua.info = py.name ++ " набрала " ++ py.score
lua.print(lua.info)  # Вывод: "Алиса набрала 85.5"

# Форматирование float использует %g (42.0 становится "42")
py.value = 42.0
lua.formatted = py.value ++ " очков"
lua.print(lua.formatted)  # Вывод: "42 очков"
```

### ⚡ Фоновое выполнение
```python
# Неблокирующие задачи
py (background_task) {
    def background_task():
        import time
        time.sleep(2)
        print("Фоновая задача завершена")
}

py.background_task() &  # Выполняется в фоне
py.print("Это выполняется немедленно")
```

### 📦 Контролируемое сохранение переменных
```python
# Переменные изолированы по умолчанию
py {
    temp_var = "не сохраняется"
    print(temp_var)
}
# py.temp_var  # Ошибка: переменная недоступна

# Явное сохранение переменных
py (greeting, calculate) {
    def greeting(name):
        return f"Привет, {name}!"
    
    def calculate(x, y):
        return x * y + 10
}

# Функции сохраняются и могут использоваться
lua.print(py.greeting("Мир"))     # Вывод: Привет, Мир!
lua.print(py.calculate(5, 3))     # Вывод: 25
```

## 🛠️ Языковые рантаймы

### Python Runtime - Полная экосистема
```python
py (fetch_data) {
    import requests, json
    
    def fetch_data(url):
        response = requests.get(url)
        return response.json()
}
```

### Lua Runtime - Высокая производительность
```lua
lua (fast_filter) {
    function fast_filter(arr, predicate)
        local result = {}
        for i, v in ipairs(arr) do
            if predicate(v) then
                table.insert(result, v)
            end
        end
        return result
    end
}
```

### JavaScript Runtime - Экосистема Node.js
```javascript
js (processFiles) {
    const fs = require('fs');
    const path = require('path');
    
    function processFiles(directory) {
        const files = fs.readdirSync(directory);
        return files.filter(f => path.extname(f) === '.js');
    }
}
```

### Go Runtime - Системные операции
```go
# Высокопроизводительные утилиты (stateless)
lua.result = go.md5("hello world")      # Криптография
lua.timestamp = go.timestamp()          # Функции времени
lua.encoded = go.base64_encode("data")  # Кодирование
lua.files = go.list_dir("/tmp")         # Файловые операции
```

## 📁 Структура проекта

```
funterm/
├── main.go                    # Точка входа
├── config.go                  # Система конфигурации
├── batch.go                   # Режим пакетного выполнения
├── engine/                    # Движок выполнения
├── go-parser/                 # Парсер с поддержкой битовых строк
├── runtime/                   # Языковые рантаймы
│   ├── python/               # Python рантайм (внешний процесс)
│   ├── lua/                  # Lua рантайм (встроенный)
│   ├── node/                 # Node.js рантайм (внешний)
│   └── go/                   # Go рантайм (встроенный)
├── funbit/                   # Библиотека обработки битовых строк
├── examples/                 # Примеры использования
└── tests/                    # Тестовые сценарии (regression тесты)
    ├── 016_*.su              # Тесты битовых строк
    ├── 012_*.su              # Тесты pipe выражений
    ├── 013_*.su              # Тесты фонового выполнения
    └── 018_*.su              # Тесты inplace сопоставления шаблонов
```

## ⚙️ Конфигурация

### Поведение по умолчанию (конфиг не требуется)
```bash
# Работает "из коробки" с умными дефолтами
./funterm
```

### Опциональная конфигурация
```yaml
# ~/.funterm/config.yaml
engine:
  max_execution_time_seconds: 60  # Кастомный timeout
  verbose: true                   # Debug вывод

languages:
  disabled: ["go"]  # Отключить конкретные языки
  runtimes:
    python:
      path: "/usr/local/bin/python3.11"  # Кастомный путь к Python
```

## 📚 Примеры

Ознакомьтесь с директорией `examples/` для примеров реального использования:
- `001_welcome.su` - Базовое мультиязыковое использование
- `006_bitcoin.su` - Парсинг Bitcoin транзакций
- `008_iot.su` - Обработка IoT протоколов

Директория `tests/` содержит комплексные тестовые сценарии, демонстрирующие все возможности.

## 🤝 Участие в разработке

Проект с открытым исходным кодом, приветствующий вклад в новые функции, оптимизации и документацию.

**Лицензия**: MIT

---

**Готовы оркестрировать несколько языков с продвинутой обработкой бинарных данных?** 🚀  
`./funterm`