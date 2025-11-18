package logging

import (
	"context"
	"fmt"
	"funterm/errors"
	"os"
	"runtime"
	"time"
)

// LogLevel represents the severity level of a log entry
type LogLevel int

const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarning
	LevelError
	LevelFatal
)

// String returns the string representation of a log level
func (l LogLevel) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarning:
		return "WARNING"
	case LevelError:
		return "ERROR"
	case LevelFatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// LogField represents a key-value pair for structured logging
type LogField struct {
	Key   string      `json:"key"`
	Value interface{} `json:"value"`
}

// LogEntry represents a single log entry
type LogEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	Level     LogLevel               `json:"level"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
	Caller    string                 `json:"caller,omitempty"`
	Stack     string                 `json:"stack,omitempty"`
	Context   map[string]interface{} `json:"context,omitempty"`
	Error     error                  `json:"error,omitempty"`
	TraceID   string                 `json:"trace_id,omitempty"`
	SpanID    string                 `json:"span_id,omitempty"`
	RequestID string                 `json:"request_id,omitempty"`
	UserID    string                 `json:"user_id,omitempty"`
	Component string                 `json:"component,omitempty"`
	Language  string                 `json:"language,omitempty"`
}

// Logger defines the interface for structured logging
type Logger interface {
	// Debug logs a debug message
	Debug(msg string, fields ...LogField)

	// Info logs an info message
	Info(msg string, fields ...LogField)

	// Warn logs a warning message
	Warn(msg string, fields ...LogField)

	// Error logs an error message
	Error(msg string, fields ...LogField)

	// ErrorWithPosition logs an error message with position information
	ErrorWithPosition(msg string, line, column int, fields ...LogField)

	// ErrorWithASTPosition logs an error message with AST position information
	ErrorWithASTPosition(msg string, file string, line, column int, fields ...LogField)

	// ErrorExecution logs an execution error with position information
	ErrorExecution(err error, fields ...LogField)

	// Fatal logs a fatal message and exits the program
	Fatal(msg string, fields ...LogField)

	// WithFields returns a new logger with the specified fields
	WithFields(fields ...LogField) Logger

	// WithError returns a new logger with the specified error
	WithError(err error) Logger

	// WithContext returns a new logger with the specified context
	WithContext(ctx context.Context) Logger

	// WithComponent returns a new logger with the specified component
	WithComponent(component string) Logger

	// WithLanguage returns a new logger with the specified language
	WithLanguage(language string) Logger

	// WithTrace returns a new logger with the specified trace and span IDs
	WithTrace(traceID, spanID string) Logger

	// WithRequest returns a new logger with the specified request ID
	WithRequest(requestID string) Logger

	// WithUser returns a new logger with the specified user ID
	WithUser(userID string) Logger

	// SetLevel sets the minimum log level
	SetLevel(level LogLevel)

	// GetLevel returns the current minimum log level
	GetLevel() LogLevel
}

// Formatter defines the interface for log formatting
type Formatter interface {
	// Format formats a log entry into a byte slice
	Format(entry *LogEntry) ([]byte, error)

	// GetName returns the name of the formatter
	GetName() string
}

// Writer defines the interface for log output
type Writer interface {
	// Write writes the formatted log entry
	Write(data []byte) error

	// Flush flushes any buffered data
	Flush() error

	// Close closes the writer
	Close() error

	// GetName returns the name of the writer
	GetName() string
}

// DefaultLogger is the default implementation of Logger
type DefaultLogger struct {
	level      LogLevel
	fields     map[string]interface{}
	error      error
	context    context.Context
	component  string
	language   string
	traceID    string
	spanID     string
	requestID  string
	userID     string
	formatters []Formatter
	writers    []Writer
	callerSkip int
}

// NewDefaultLogger creates a new default logger
func NewDefaultLogger() *DefaultLogger {
	return &DefaultLogger{
		level:      LevelInfo,
		fields:     make(map[string]interface{}),
		formatters: []Formatter{NewJSONFormatter()},
		writers:    []Writer{NewConsoleWriter()},
		callerSkip: 2,
	}
}

// NewDefaultLoggerWithConfig creates a new default logger with configuration
func NewDefaultLoggerWithConfig(config LoggerConfig) *DefaultLogger {
	logger := &DefaultLogger{
		level:      config.Level,
		fields:     make(map[string]interface{}),
		formatters: config.Formatters,
		writers:    config.Writers,
		callerSkip: config.CallerSkip,
	}

	if logger.formatters == nil {
		logger.formatters = []Formatter{NewJSONFormatter()}
	}

	if logger.writers == nil {
		logger.writers = []Writer{NewConsoleWriter()}
	}

	return logger
}

// RotationConfig contains configuration for log rotation
type RotationConfig struct {
	MaxSize    int64         // Maximum size in bytes before rotation
	MaxAge     time.Duration // Maximum age before rotation
	MaxBackups int           // Maximum number of backup files to keep
	Compress   bool          // Whether to compress backup files
}

// LoggerConfig contains configuration for the logger
type LoggerConfig struct {
	Level      LogLevel
	Formatters []Formatter
	Writers    []Writer
	CallerSkip int
	Rotation   *RotationConfig // Optional rotation configuration
}

// ApplyLogLevel applies log level from string configuration
func (lc *LoggerConfig) ApplyLogLevel(levelStr string) {
	switch levelStr {
	case "debug":
		lc.Level = LevelDebug
	case "info":
		lc.Level = LevelInfo
	case "warning", "warn":
		lc.Level = LevelWarning
	case "error":
		lc.Level = LevelError
	case "fatal":
		lc.Level = LevelFatal
	default:
		lc.Level = LevelInfo
	}
}

// Debug logs a debug message
func (l *DefaultLogger) Debug(msg string, fields ...LogField) {
	l.log(LevelDebug, msg, fields...)
}

// Info logs an info message
func (l *DefaultLogger) Info(msg string, fields ...LogField) {
	l.log(LevelInfo, msg, fields...)
}

// Warn logs a warning message
func (l *DefaultLogger) Warn(msg string, fields ...LogField) {
	l.log(LevelWarning, msg, fields...)
}

// Error logs an error message
func (l *DefaultLogger) Error(msg string, fields ...LogField) {
	l.log(LevelError, msg, fields...)
}

// ErrorWithPosition logs an error message with position information
func (l *DefaultLogger) ErrorWithPosition(msg string, line, column int, fields ...LogField) {
	posFields := append(fields, LogField{Key: "line", Value: line}, LogField{Key: "column", Value: column})
	l.log(LevelError, msg, posFields...)
}

// ErrorWithASTPosition logs an error message with AST position information
func (l *DefaultLogger) ErrorWithASTPosition(msg string, file string, line, column int, fields ...LogField) {
	posFields := append(fields,
		LogField{Key: "file", Value: file},
		LogField{Key: "line", Value: line},
		LogField{Key: "column", Value: column})
	l.log(LevelError, msg, posFields...)
}

// ErrorExecution logs an execution error with position information
func (l *DefaultLogger) ErrorExecution(err error, fields ...LogField) {
	// Try to extract position information from the error if it's an ExecutionError
	if execErr, ok := err.(*errors.ExecutionError); ok {
		posFields := append(fields,
			LogField{Key: "error_code", Value: execErr.Code},
			LogField{Key: "error_type", Value: string(execErr.Type)},
			LogField{Key: "line", Value: execErr.Line},
			LogField{Key: "column", Value: execErr.Col})
		if execErr.Language != "" {
			posFields = append(posFields, LogField{Key: "language", Value: execErr.Language})
		}
		l.log(LevelError, execErr.Message, posFields...)
	} else {
		// For other error types, just log the error
		errorFields := append(fields, LogField{Key: "error", Value: err.Error()})
		l.log(LevelError, err.Error(), errorFields...)
	}
}

// Fatal logs a fatal message and exits the program
func (l *DefaultLogger) Fatal(msg string, fields ...LogField) {
	l.log(LevelFatal, msg, fields...)
	l.flush()
	os.Exit(1)
}

// WithFields returns a new logger with the specified fields
func (l *DefaultLogger) WithFields(fields ...LogField) Logger {
	newLogger := l.copy()
	for _, field := range fields {
		newLogger.fields[field.Key] = field.Value
	}
	return newLogger
}

// WithError returns a new logger with the specified error
func (l *DefaultLogger) WithError(err error) Logger {
	newLogger := l.copy()
	newLogger.error = err
	return newLogger
}

// WithContext returns a new logger with the specified context
func (l *DefaultLogger) WithContext(ctx context.Context) Logger {
	newLogger := l.copy()
	newLogger.context = ctx

	// Extract common context values
	if ctx != nil {
		if traceID := ctx.Value("trace_id"); traceID != nil {
			newLogger.traceID = fmt.Sprintf("%v", traceID)
		}
		if spanID := ctx.Value("span_id"); spanID != nil {
			newLogger.spanID = fmt.Sprintf("%v", spanID)
		}
		if requestID := ctx.Value("request_id"); requestID != nil {
			newLogger.requestID = fmt.Sprintf("%v", requestID)
		}
		if userID := ctx.Value("user_id"); userID != nil {
			newLogger.userID = fmt.Sprintf("%v", userID)
		}
	}

	return newLogger
}

// WithComponent returns a new logger with the specified component
func (l *DefaultLogger) WithComponent(component string) Logger {
	newLogger := l.copy()
	newLogger.component = component
	return newLogger
}

// WithLanguage returns a new logger with the specified language
func (l *DefaultLogger) WithLanguage(language string) Logger {
	newLogger := l.copy()
	newLogger.language = language
	return newLogger
}

// WithTrace returns a new logger with the specified trace and span IDs
func (l *DefaultLogger) WithTrace(traceID, spanID string) Logger {
	newLogger := l.copy()
	newLogger.traceID = traceID
	newLogger.spanID = spanID
	return newLogger
}

// WithRequest returns a new logger with the specified request ID
func (l *DefaultLogger) WithRequest(requestID string) Logger {
	newLogger := l.copy()
	newLogger.requestID = requestID
	return newLogger
}

// WithUser returns a new logger with the specified user ID
func (l *DefaultLogger) WithUser(userID string) Logger {
	newLogger := l.copy()
	newLogger.userID = userID
	return newLogger
}

// SetLevel sets the minimum log level
func (l *DefaultLogger) SetLevel(level LogLevel) {
	l.level = level
}

// GetLevel returns the current minimum log level
func (l *DefaultLogger) GetLevel() LogLevel {
	return l.level
}

// log is the internal logging method
func (l *DefaultLogger) log(level LogLevel, msg string, fields ...LogField) {
	if level < l.level {
		return
	}

	// Create log entry
	entry := &LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   msg,
		Fields:    make(map[string]interface{}),
		Caller:    l.getCaller(),
		Component: l.component,
		Language:  l.language,
		TraceID:   l.traceID,
		SpanID:    l.spanID,
		RequestID: l.requestID,
		UserID:    l.userID,
		Error:     l.error,
	}

	// Add default fields
	for k, v := range l.fields {
		entry.Fields[k] = v
	}

	// Add additional fields
	for _, field := range fields {
		entry.Fields[field.Key] = field.Value
	}

	// Add context fields
	if l.context != nil {
		entry.Context = make(map[string]interface{})
		if userID := l.context.Value("user_id"); userID != nil {
			entry.Context["user_id"] = userID
		}
		if sessionID := l.context.Value("session_id"); sessionID != nil {
			entry.Context["session_id"] = sessionID
		}
		if requestID := l.context.Value("request_id"); requestID != nil {
			entry.Context["request_id"] = requestID
		}
	}

	// Add stack trace for error levels
	if level >= LevelError {
		entry.Stack = l.getStackTrace()
	}

	// Format and write the log entry
	for _, formatter := range l.formatters {
		data, err := formatter.Format(entry)
		if err != nil {
			// If formatting fails, try to log the error
			errorEntry := &LogEntry{
				Timestamp: time.Now(),
				Level:     LevelError,
				Message:   fmt.Sprintf("Failed to format log entry: %v", err),
			}
			// For now, just write a simple error message
			errorMsg := fmt.Sprintf("Failed to format log entry: %v - Original message: %s\n", err, errorEntry.Message)
			l.writeToAllWriters([]byte(errorMsg))
			continue
		}

		l.writeToAllWriters(data)
	}
}

// writeToAllWriters writes data to all writers
func (l *DefaultLogger) writeToAllWriters(data []byte) {
	for _, writer := range l.writers {
		if err := writer.Write(data); err != nil {
			// If writing fails, try to log the error to stderr
			fmt.Fprintf(os.Stderr, "Failed to write log: %v\n", err)
		}
	}
}

// flush flushes all writers
func (l *DefaultLogger) flush() {
	for _, writer := range l.writers {
		if err := writer.Flush(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to flush log writer: %v\n", err)
		}
	}
}

// copy creates a copy of the logger
func (l *DefaultLogger) copy() *DefaultLogger {
	newLogger := &DefaultLogger{
		level:      l.level,
		fields:     make(map[string]interface{}),
		error:      l.error,
		context:    l.context,
		component:  l.component,
		language:   l.language,
		traceID:    l.traceID,
		spanID:     l.spanID,
		requestID:  l.requestID,
		userID:     l.userID,
		formatters: l.formatters,
		writers:    l.writers,
		callerSkip: l.callerSkip,
	}

	// Copy fields
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}

	return newLogger
}

// getCaller returns the caller information
func (l *DefaultLogger) getCaller() string {
	_, file, line, ok := runtime.Caller(l.callerSkip)
	if !ok {
		return ""
	}
	return fmt.Sprintf("%s:%d", file, line)
}

// getStackTrace returns the stack trace
func (l *DefaultLogger) getStackTrace() string {
	buf := make([]byte, 4096)
	n := runtime.Stack(buf, false)
	return string(buf[:n])
}

// Field creates a new field
func Field(key string, value interface{}) LogField {
	return LogField{Key: key, Value: value}
}

// StringField creates a new string field
func StringField(key, value string) LogField {
	return LogField{Key: key, Value: value}
}

// IntField creates a new int field
func IntField(key string, value int) LogField {
	return LogField{Key: key, Value: value}
}

// Int64Field creates a new int64 field
func Int64Field(key string, value int64) LogField {
	return LogField{Key: key, Value: value}
}

// Float64Field creates a new float64 field
func Float64Field(key string, value float64) LogField {
	return LogField{Key: key, Value: value}
}

// BoolField creates a new bool field
func BoolField(key string, value bool) LogField {
	return LogField{Key: key, Value: value}
}

// ErrorField creates a new error field
func ErrorField(key string, value error) LogField {
	return LogField{Key: key, Value: value.Error()}
}

// TimeField creates a new time field
func TimeField(key string, value time.Time) LogField {
	return LogField{Key: key, Value: value.Format(time.RFC3339)}
}

// DurationField creates a new duration field
func DurationField(key string, value time.Duration) LogField {
	return LogField{Key: key, Value: value.String()}
}

// AnyField creates a new field with any value
func AnyField(key string, value interface{}) LogField {
	return LogField{Key: key, Value: value}
}
