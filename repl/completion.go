package repl

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"funterm/runtime"
)

// Completion представляет одно предложение для автодополнения
type Completion struct {
	Text        string
	Description string
	Type        string // "user_function", "user_variable", "user_object", "module", "function", "object_method"
	Priority    int    // Приоритет: пользовательские = 10, методы объектов = 8, модули = 5, функции = 1
}

// CompletionCache управляет кешированием результатов автодополнения
type CompletionCache struct {
	items map[string]cacheEntry
	mutex sync.RWMutex
}

type cacheEntry struct {
	completions []Completion
	expiry      time.Time
}

// RuntimeCompleter реализует интерфейс readline.AutoCompleter
type RuntimeCompleter struct {
	runtimeManager *runtime.RuntimeManager
	cache          *CompletionCache
	mutex          sync.RWMutex
}

// findWordBoundaries находит границы слова для автодополнения.
// Слово - это последовательность букв, цифр, подчеркиваний и точек.
func (rc *RuntimeCompleter) findWordBoundaries(line []rune, pos int) (start, end int) {
	start = pos
	for start > 0 {
		r := line[start-1]
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '.' {
			start--
		} else {
			break
		}
	}
	return start, pos
}

// NewCompletionCache создает новый кеш автодополнения
func NewCompletionCache() *CompletionCache {
	return &CompletionCache{
		items: make(map[string]cacheEntry),
	}
}

// Set сохраняет дополнения в кеш с TTL
func (cc *CompletionCache) Set(key string, completions []Completion, ttl time.Duration) {
	cc.mutex.Lock()
	defer cc.mutex.Unlock()

	cc.items[key] = cacheEntry{
		completions: completions,
		expiry:      time.Now().Add(ttl),
	}
}

// Get получает дополнения из кеша
func (cc *CompletionCache) Get(key string) ([]Completion, bool) {
	cc.mutex.RLock()
	defer cc.mutex.RUnlock()

	entry, exists := cc.items[key]
	if !exists || time.Now().After(entry.expiry) {
		return nil, false
	}

	return entry.completions, true
}

// NewRuntimeCompleter создает новый runtime completer
func NewRuntimeCompleter(runtimeManager *runtime.RuntimeManager) *RuntimeCompleter {
	return &RuntimeCompleter{
		runtimeManager: runtimeManager,
		cache:          NewCompletionCache(),
	}
}

// Do реализует интерфейс readline.AutoCompleter, следуя документации v1.5.1
// Возвращает:
// - newLine: список суффиксов (оставшихся частей) для каждого варианта завершения.
// - length: длина префикса, который пользователь уже набрал.
func (rc *RuntimeCompleter) Do(line []rune, pos int) (newLine [][]rune, length int) {
	defer func() {
		if r := recover(); r != nil {
			newLine = [][]rune{}
			length = 0
		}
	}()

	start, _ := rc.findWordBoundaries(line, pos)
	word := string(line[start:pos])

	parts := strings.Split(word, ".")
	numParts := len(parts)

	var prefixToComplete string
	if len(word) > 0 && !strings.HasSuffix(word, ".") {
		prefixToComplete = parts[numParts-1]
	}
	length = len(prefixToComplete)

	var fullCompletions []Completion
	switch {
	case numParts == 1:
		// Контекст 1: Завершение имени языка ("p", "py")
		// Показываем и полные названия, и алиасы
		var suggestions [][]rune

		// Получаем все доступные языки
		languages := rc.runtimeManager.ListRuntimes()

		// Добавляем полные названия языков
		for _, lang := range languages {
			if strings.HasPrefix(lang, prefixToComplete) {
				// Возвращаем суффикс: то, что нужно добавить к уже введенному тексту
				suffix := strings.TrimPrefix(lang+".", prefixToComplete)
				suggestions = append(suggestions, []rune(suffix))
			}
		}

		// Добавляем алиасы
		aliases := rc.getLanguageAliases()
		for alias := range aliases {
			if strings.HasPrefix(alias, prefixToComplete) {
				// Возвращаем суффикс: то, что нужно добавить к уже введенному тексту
				suffix := strings.TrimPrefix(alias+".", prefixToComplete)
				suggestions = append(suggestions, []rune(suffix))
			}
		}

		// Возвращаем длину префикса, который пользователь уже ввел
		return suggestions, len(prefixToComplete)

	case numParts == 2:
		// Контекст 2: Завершение после языка ("lua.", "lua.s", "py.", "js.")
		language := parts[0]
		// Преобразуем алиас в полное название языка
		fullLanguage := rc.resolveLanguageAlias(language)
		if rc.isValidLanguage(fullLanguage) {
			fullCompletions = rc.getLanguageCompletionsWithContext(fullLanguage, prefixToComplete)
		}

	case numParts >= 3:
		// Контекст 3: Завершение после модуля/объекта ("lua.string.", "lua.string.u", "py.math.", "js.fs.")
		language := parts[0]
		// Преобразуем алиас в полное название языка
		fullLanguage := rc.resolveLanguageAlias(language)
		moduleOrObject := parts[1]
		if rc.isValidLanguage(fullLanguage) {
			// Это может быть как метод объекта, так и функция модуля
			if rc.isUserObject(fullLanguage, moduleOrObject) {
				fullCompletions = rc.getUserObjectMethods(fullLanguage, moduleOrObject, prefixToComplete)
			} else {
				fullCompletions = rc.getModuleCompletions(fullLanguage, moduleOrObject, prefixToComplete)
			}
		}
	}

	if len(fullCompletions) == 0 {
		return [][]rune{}, 0
	}

	// Удаляем дубликаты перед преобразованием
	uniqueCompletions := rc.removeDuplicateCompletions(fullCompletions)

	// Преобразуем полные варианты в суффиксы
	suffixes := make([][]rune, 0, len(uniqueCompletions))
	for _, completion := range uniqueCompletions {
		if strings.HasPrefix(completion.Text, prefixToComplete) {
			suffix := strings.TrimPrefix(completion.Text, prefixToComplete)
			suffixes = append(suffixes, []rune(suffix))
		}
	}

	return suffixes, length
}

// getLanguageAliases возвращает маппинг алиасов языков к их полным названиям
func (rc *RuntimeCompleter) getLanguageAliases() map[string]string {
	return map[string]string{
		"py": "python",
		"js": "node", // js -> node (JavaScript)
	}
}

// resolveLanguageAlias преобразует алиас языка в полное название
func (rc *RuntimeCompleter) resolveLanguageAlias(alias string) string {
	if aliases := rc.getLanguageAliases(); aliases[alias] != "" {
		return aliases[alias]
	}
	return alias // Если это не алиас, возвращаем как есть
}

// getAvailableLanguagesReadline возвращает доступные языки в формате readline
func (rc *RuntimeCompleter) getAvailableLanguagesReadline() [][]rune {
	languages := rc.runtimeManager.ListRuntimes()
	var suggestions [][]rune

	for _, lang := range languages {
		suggestion := []rune(lang + ".")
		suggestions = append(suggestions, suggestion)
	}

	return suggestions
}

// isValidLanguage проверяет, является ли строка именем доступного языка
func (rc *RuntimeCompleter) isValidLanguage(lang string) bool {
	languages := rc.runtimeManager.ListRuntimes()
	for _, availableLang := range languages {
		if availableLang == lang {
			return true
		}
	}
	return false
}

// getLanguageContextCompletionsReadline возвращает дополнения для языка
func (rc *RuntimeCompleter) getLanguageContextCompletionsReadline(language, prefix string) [][]rune {
	completions := rc.getLanguageCompletionsWithContext(language, prefix)

	// Если нет дополнений, возвращаем пустой результат
	// Это предотвратит показ других языков при вводе lua.
	if len(completions) == 0 {
		return [][]rune{}
	}

	return rc.convertToReadlineFormat(completions)
}

// getLanguageCompletionsWithContext получает дополнения для языка
func (rc *RuntimeCompleter) getLanguageCompletionsWithContext(language, prefix string) []Completion {
	rc.mutex.RLock()
	defer rc.mutex.RUnlock()

	cacheKey := fmt.Sprintf("language_context.%s.%s", language, prefix)
	if completions, found := rc.cache.Get(cacheKey); found {
		return completions
	}

	rt, err := rc.runtimeManager.GetRuntime(language)
	if err != nil {
		return nil
	}

	var completions []Completion

	// 1. Пользовательские функции (высший приоритет)
	userFunctions := rt.GetUserDefinedFunctions()
	for _, fn := range userFunctions {
		if strings.HasPrefix(fn, prefix) {
			completions = append(completions, Completion{
				Text:        fn,
				Description: fmt.Sprintf("Пользовательская функция в %s", language),
				Type:        "user_function",
				Priority:    10,
			})
		}
	}

	// 2. Пользовательские переменные и объекты
	userVars := rt.GetGlobalVariables()
	for _, variable := range userVars {
		if strings.HasPrefix(variable, prefix) {
			// Не проверяем тип переменной для ускорения completion
			// Все переменные считаем пользовательскими переменными
			completions = append(completions, Completion{
				Text:        variable,
				Description: fmt.Sprintf("Пользовательская переменная в %s", language),
				Type:        "user_variable",
				Priority:    10,
			})
		}
	}

	// 3. Стандартные модули (низший приоритет)
	modules := rt.GetModules()
	for _, mod := range modules {
		if strings.HasPrefix(mod, prefix) {
			completions = append(completions, Completion{
				Text:        mod,
				Description: fmt.Sprintf("Стандартный модуль %s", mod),
				Type:        "module",
				Priority:    5,
			})
		}
	}

	// Сортируем по приоритету и алфавиту
	sort.Slice(completions, func(i, j int) bool {
		if completions[i].Priority != completions[j].Priority {
			return completions[i].Priority > completions[j].Priority
		}
		return completions[i].Text < completions[j].Text
	})

	rc.cache.Set(cacheKey, completions, 2*time.Minute)
	return completions
}

// getSecondLevelCompletionsReadline обрабатывает второй уровень дополнения
func (rc *RuntimeCompleter) getSecondLevelCompletionsReadline(language, secondPart, prefix string) [][]rune {
	// Сначала проверяем, является ли secondPart пользовательским объектом
	if rc.isUserObject(language, secondPart) {
		// Это пользовательский объект - показываем его методы
		return rc.getUserObjectMethodsReadline(language, secondPart, prefix)
	}

	// Иначе считаем, что это модуль - показываем функции модуля
	return rc.getModuleCompletionsReadline(language, secondPart, prefix)
}

// isUserObject проверяет, является ли имя пользовательским объектом
func (rc *RuntimeCompleter) isUserObject(language, objectName string) bool {
	rt, err := rc.runtimeManager.GetRuntime(language)
	if err != nil {
		return false
	}

	// Проверяем, существует ли такая переменная
	_, err = rt.GetVariable(objectName)
	return err == nil
}

// getUserObjectMethodsReadline возвращает методы пользовательского объекта
func (rc *RuntimeCompleter) getUserObjectMethodsReadline(language, objectName, prefix string) [][]rune {
	completions := rc.getUserObjectMethods(language, objectName, prefix)
	return rc.convertToReadlineFormat(completions)
}

// getUserObjectMethods получает методы пользовательского объекта
func (rc *RuntimeCompleter) getUserObjectMethods(language, objectName, prefix string) []Completion {
	rc.mutex.RLock()
	defer rc.mutex.RUnlock()

	cacheKey := fmt.Sprintf("user_object.%s.%s.%s", language, objectName, prefix)
	if completions, found := rc.cache.Get(cacheKey); found {
		return completions
	}

	rt, err := rc.runtimeManager.GetRuntime(language)
	if err != nil {
		return nil
	}

	// Получаем свойства и методы объекта через runtime
	properties, err := rt.GetObjectProperties(objectName)
	if err != nil {
		// Если не удалось получить свойства динамически, используем общие методы
		return rc.getCommonMethodsForObject(language, objectName, prefix)
	}

	var completions []Completion
	for _, prop := range properties {
		if strings.HasPrefix(prop, prefix) {
			completions = append(completions, Completion{
				Text:        prop,
				Description: fmt.Sprintf("Метод объекта %s", objectName),
				Type:        "object_method",
				Priority:    8,
			})
		}
	}

	// Добавляем общие методы для известных типов объектов
	commonMethods := rc.getCommonMethodsForObject(language, objectName, prefix)
	completions = append(completions, commonMethods...)

	// Сортируем по приоритету и алфавиту
	sort.Slice(completions, func(i, j int) bool {
		if completions[i].Priority != completions[j].Priority {
			return completions[i].Priority > completions[j].Priority
		}
		return completions[i].Text < completions[j].Text
	})

	rc.cache.Set(cacheKey, completions, 1*time.Minute) // Короткое время для объектов
	return completions
}

// getCommonMethodsForObject возвращает общие методы для известных типов объектов
func (rc *RuntimeCompleter) getCommonMethodsForObject(language, objectName, prefix string) []Completion {
	var methods []Completion

	// Для Python списков
	if language == "python" || language == "py" {
		if strings.HasSuffix(objectName, "list") || strings.Contains(objectName, "list") {
			pythonListMethods := []string{"append", "extend", "insert", "remove", "pop", "clear", "index", "count", "sort", "reverse", "copy"}
			for _, method := range pythonListMethods {
				if strings.HasPrefix(method, prefix) {
					methods = append(methods, Completion{
						Text:        method,
						Description: "Метод списка Python",
						Type:        "object_method",
						Priority:    8,
					})
				}
			}
		}

		// Для Python словарей
		if strings.HasSuffix(objectName, "dict") || strings.Contains(objectName, "dict") {
			pythonDictMethods := []string{"get", "keys", "values", "items", "clear", "copy", "pop", "popitem", "update", "setdefault"}
			for _, method := range pythonDictMethods {
				if strings.HasPrefix(method, prefix) {
					methods = append(methods, Completion{
						Text:        method,
						Description: "Метод словаря Python",
						Type:        "object_method",
						Priority:    8,
					})
				}
			}
		}
	}

	// Для JavaScript массивов
	if language == "js" || language == "node" {
		if strings.HasSuffix(objectName, "array") || strings.Contains(objectName, "array") {
			jsArrayMethods := []string{"push", "pop", "shift", "unshift", "splice", "slice", "concat", "join", "reverse", "sort", "indexOf", "lastIndexOf", "forEach", "map", "filter", "reduce", "find", "some", "every"}
			for _, method := range jsArrayMethods {
				if strings.HasPrefix(method, prefix) {
					methods = append(methods, Completion{
						Text:        method,
						Description: "Метод массива JavaScript",
						Type:        "object_method",
						Priority:    8,
					})
				}
			}
		}
	}

	// Для Lua таблиц
	if language == "lua" {
		if strings.Contains(objectName, "table") {
			luaTableMethods := []string{"insert", "remove", "sort", "concat", "pack", "unpack", "move"}
			for _, method := range luaTableMethods {
				if strings.HasPrefix(method, prefix) {
					methods = append(methods, Completion{
						Text:        method,
						Description: "Метод таблицы Lua",
						Type:        "object_method",
						Priority:    8,
					})
				}
			}
		}
	}

	return methods
}

// getModuleCompletionsReadline возвращает модули для языка в формате readline
func (rc *RuntimeCompleter) getModuleCompletionsReadline(language, module, prefix string) [][]rune {
	completions := rc.getModuleCompletions(language, module, prefix)
	return rc.convertToReadlineFormat(completions)
}

// getModuleCompletions получает дополнения для модуля
func (rc *RuntimeCompleter) getModuleCompletions(language, module, prefix string) []Completion {
	rc.mutex.RLock()
	defer rc.mutex.RUnlock()

	cacheKey := fmt.Sprintf("module.%s.%s.%s", language, module, prefix)
	if completions, found := rc.cache.Get(cacheKey); found {
		return completions
	}

	rt, err := rc.runtimeManager.GetRuntime(language)
	if err != nil {
		return nil
	}

	functions := rt.GetModuleFunctions(module)
	var completions []Completion

	for _, fn := range functions {
		if strings.HasPrefix(fn, prefix) {
			completions = append(completions, Completion{
				Text:        fn,
				Description: fmt.Sprintf("Функция модуля %s", module),
				Type:        "function",
				Priority:    1,
			})
		}
	}

	rc.cache.Set(cacheKey, completions, 5*time.Minute)
	return completions
}

// getObjectCompletionsReadline возвращает свойства объекта в формате readline
func (rc *RuntimeCompleter) getObjectCompletionsReadline(language, object, prefix string) [][]rune {
	completions := rc.getObjectCompletions(language, object, prefix)
	return rc.convertToReadlineFormat(completions)
}

// getObjectCompletions получает дополнения для объекта
func (rc *RuntimeCompleter) getObjectCompletions(language, object, prefix string) []Completion {
	rc.mutex.RLock()
	defer rc.mutex.RUnlock()

	cacheKey := fmt.Sprintf("object.%s.%s.%s", language, object, prefix)
	if completions, found := rc.cache.Get(cacheKey); found {
		return completions
	}

	rt, err := rc.runtimeManager.GetRuntime(language)
	if err != nil {
		return nil
	}

	properties, err := rt.GetObjectProperties(object)
	if err != nil {
		return nil
	}

	var completions []Completion
	for _, prop := range properties {
		if strings.HasPrefix(prop, prefix) {
			completions = append(completions, Completion{
				Text:        prop,
				Description: fmt.Sprintf("Свойство объекта %s", object),
				Type:        "object_method",
				Priority:    8,
			})
		}
	}

	rc.cache.Set(cacheKey, completions, 1*time.Minute)
	return completions
}

// convertToReadlineFormat конвертирует дополнения в формат readline
func (rc *RuntimeCompleter) convertToReadlineFormat(completions []Completion) [][]rune {
	var suggestions [][]rune

	for _, completion := range completions {
		suggestion := []rune(completion.Text)
		suggestions = append(suggestions, suggestion)
	}

	return suggestions
}

// safeGetRuntime безопасно получает рантайм с обработкой ошибок
func (rc *RuntimeCompleter) safeGetRuntime(language string) (runtime.LanguageRuntime, bool) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Warning: Runtime completer error: %v\n", r)
		}
	}()

	rt, err := rc.runtimeManager.GetRuntime(language)
	return rt, err == nil
}

// fallbackCompletion предоставляет базовые дополнения при ошибках
func (rc *RuntimeCompleter) fallbackCompletion(input string) ([][]rune, int) {
	// Защита от nil или пустой строки
	if input == "" {
		return rc.getDefaultLanguages()
	}

	// Базовые языки для fallback
	languages := []string{"python", "lua", "js", "node", "go"}
	var suggestions [][]rune

	for _, lang := range languages {
		if strings.HasPrefix(lang, input) {
			suggestion := []rune(lang + ".")
			suggestions = append(suggestions, suggestion)
		}
	}

	// Если не найдено подходящих языков, возвращаем все доступные
	if len(suggestions) == 0 {
		return rc.getDefaultLanguages()
	}

	return suggestions, len(suggestions)
}

// getDefaultLanguages возвращает список доступных языков по умолчанию
func (rc *RuntimeCompleter) getDefaultLanguages() ([][]rune, int) {
	languages := []string{"python.", "lua.", "js.", "go."}
	var suggestions [][]rune

	for _, lang := range languages {
		suggestions = append(suggestions, []rune(lang))
	}

	return suggestions, len(suggestions)
}

// DoWithFallback реализует дополнение с fallback при ошибках
func (rc *RuntimeCompleter) DoWithFallback(line []rune, pos int) (newLine [][]rune, length int) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Warning: Completion error: %v\n", r)
			// Возвращаем пустой результат вместо fallback при ошибках
			newLine, length = [][]rune{}, 0
		}
	}()

	// Пытаемся выполнить основное дополнение
	result, length := rc.Do(line, pos)

	// Если результат пустой, возвращаем пустой результат вместо fallback
	// Это предотвратит показ других языков при вводе lua.
	if len(result) == 0 {
		return [][]rune{}, 0
	}

	return result, length
}

// FallbackCompleter обертка для использования с fallback
type FallbackCompleter struct {
	*RuntimeCompleter
}

// NewFallbackCompleter создает новую обертку для completer с fallback
func NewFallbackCompleter(runtimeManager *runtime.RuntimeManager) *FallbackCompleter {
	return &FallbackCompleter{
		RuntimeCompleter: NewRuntimeCompleter(runtimeManager),
	}
}

// removeDuplicateCompletions удаляет дубликаты из списка дополнений
func (rc *RuntimeCompleter) removeDuplicateCompletions(completions []Completion) []Completion {
	seen := make(map[string]bool)
	var unique []Completion

	for _, completion := range completions {
		if !seen[completion.Text] {
			seen[completion.Text] = true
			unique = append(unique, completion)
		}
	}

	return unique
}

// Do реализует интерфейс readline.AutoCompleter с fallback
func (fc *FallbackCompleter) Do(line []rune, pos int) (newLine [][]rune, length int) {
	return fc.DoWithFallback(line, pos)
}
