package logging

import (
	"encoding/json"
	"fmt"
	"time"
)

// JSONFormatter formats log entries as JSON
type JSONFormatter struct{}

// NewJSONFormatter creates a new JSON formatter
func NewJSONFormatter() *JSONFormatter {
	return &JSONFormatter{}
}

// Format formats a log entry as JSON
func (f *JSONFormatter) Format(entry *LogEntry) ([]byte, error) {
	// Create a map for the JSON output
	output := make(map[string]interface{})

	// Add basic fields
	output["timestamp"] = entry.Timestamp.Format(time.RFC3339)
	output["level"] = entry.Level.String()
	output["message"] = entry.Message

	// Add optional fields if they exist
	if entry.Caller != "" {
		output["caller"] = entry.Caller
	}

	if entry.Component != "" {
		output["component"] = entry.Component
	}

	if entry.Language != "" {
		output["language"] = entry.Language
	}

	if entry.TraceID != "" {
		output["trace_id"] = entry.TraceID
	}

	if entry.SpanID != "" {
		output["span_id"] = entry.SpanID
	}

	if entry.RequestID != "" {
		output["request_id"] = entry.RequestID
	}

	if entry.UserID != "" {
		output["user_id"] = entry.UserID
	}

	if entry.Error != nil {
		output["error"] = entry.Error.Error()
	}

	if entry.Stack != "" {
		output["stack"] = entry.Stack
	}

	// Add fields if they exist
	if len(entry.Fields) > 0 {
		output["fields"] = entry.Fields
	}

	// Add context if it exists
	if len(entry.Context) > 0 {
		output["context"] = entry.Context
	}

	// Marshal to JSON
	return json.Marshal(output)
}

// GetName returns the name of the formatter
func (f *JSONFormatter) GetName() string {
	return "json"
}

// TextFormatter formats log entries as plain text
type TextFormatter struct {
	// IncludeTimestamp controls whether to include the timestamp
	IncludeTimestamp bool
	// IncludeCaller controls whether to include the caller information
	IncludeCaller bool
	// IncludeLevel controls whether to include the log level
	IncludeLevel bool
	// ColorOutput controls whether to use ANSI color codes
	ColorOutput bool
}

// NewTextFormatter creates a new text formatter with default settings
func NewTextFormatter() *TextFormatter {
	return &TextFormatter{
		IncludeTimestamp: true,
		IncludeCaller:    true,
		IncludeLevel:     true,
		ColorOutput:      false,
	}
}

// NewTextFormatterWithOptions creates a new text formatter with custom options
func NewTextFormatterWithOptions(includeTimestamp, includeCaller, includeLevel, colorOutput bool) *TextFormatter {
	return &TextFormatter{
		IncludeTimestamp: includeTimestamp,
		IncludeCaller:    includeCaller,
		IncludeLevel:     includeLevel,
		ColorOutput:      colorOutput,
	}
}

// Format formats a log entry as plain text
func (f *TextFormatter) Format(entry *LogEntry) ([]byte, error) {
	var output string

	// Add timestamp if enabled
	if f.IncludeTimestamp {
		output += fmt.Sprintf("[%s] ", entry.Timestamp.Format("2006-01-02 15:04:05.000"))
	}

	// Add level if enabled
	if f.IncludeLevel {
		levelStr := entry.Level.String()
		if f.ColorOutput {
			levelStr = f.colorizeLevel(levelStr, entry.Level)
		}
		output += fmt.Sprintf("[%s] ", levelStr)
	}

	// Add component if it exists
	if entry.Component != "" {
		output += fmt.Sprintf("[%s] ", entry.Component)
	}

	// Add language if it exists
	if entry.Language != "" {
		output += fmt.Sprintf("[%s] ", entry.Language)
	}

	// Add the message
	output += entry.Message

	// Add position information if available
	if line, ok := entry.Fields["line"].(int); ok {
		if col, ok := entry.Fields["column"].(int); ok {
			output += fmt.Sprintf(" (at line %d, col %d)", line, col)
		} else {
			output += fmt.Sprintf(" (at line %d)", line)
		}
	}

	// Add caller if enabled
	if f.IncludeCaller && entry.Caller != "" {
		output += fmt.Sprintf(" (caller: %s)", entry.Caller)
	}

	// Add trace information if it exists
	if entry.TraceID != "" {
		output += fmt.Sprintf(" (trace_id: %s)", entry.TraceID)
	}

	if entry.SpanID != "" {
		output += fmt.Sprintf(" (span_id: %s)", entry.SpanID)
	}

	// Add request information if it exists
	if entry.RequestID != "" {
		output += fmt.Sprintf(" (request_id: %s)", entry.RequestID)
	}

	if entry.UserID != "" {
		output += fmt.Sprintf(" (user_id: %s)", entry.UserID)
	}

	// Add error if it exists
	if entry.Error != nil {
		output += fmt.Sprintf(" (error: %s)", entry.Error.Error())
	}

	// Add fields if they exist
	if len(entry.Fields) > 0 {
		output += " " + f.formatFields(entry.Fields)
	}

	// Add context if it exists
	if len(entry.Context) > 0 {
		output += " " + f.formatContext(entry.Context)
	}

	// Add stack trace if it exists
	if entry.Stack != "" {
		output += "\n" + entry.Stack
	}

	// Add newline
	output += "\n"

	return []byte(output), nil
}

// GetName returns the name of the formatter
func (f *TextFormatter) GetName() string {
	return "text"
}

// formatFields formats the fields map as a string
func (f *TextFormatter) formatFields(fields map[string]interface{}) string {
	if len(fields) == 0 {
		return ""
	}

	output := "["
	first := true
	for key, value := range fields {
		if !first {
			output += ", "
		}
		output += fmt.Sprintf("%s=%v", key, value)
		first = false
	}
	output += "]"
	return output
}

// formatContext formats the context map as a string
func (f *TextFormatter) formatContext(context map[string]interface{}) string {
	if len(context) == 0 {
		return ""
	}

	output := "[context:"
	first := true
	for key, value := range context {
		if !first {
			output += ", "
		}
		output += fmt.Sprintf("%s=%v", key, value)
		first = false
	}
	output += "]"
	return output
}

// colorizeLevel adds ANSI color codes to the level string
func (f *TextFormatter) colorizeLevel(level string, logLevel LogLevel) string {
	if !f.ColorOutput {
		return level
	}

	switch logLevel {
	case LevelDebug:
		return fmt.Sprintf("\x1b[36m%s\x1b[0m", level) // Cyan
	case LevelInfo:
		return fmt.Sprintf("\x1b[32m%s\x1b[0m", level) // Green
	case LevelWarning:
		return fmt.Sprintf("\x1b[33m%s\x1b[0m", level) // Yellow
	case LevelError:
		return fmt.Sprintf("\x1b[31m%s\x1b[0m", level) // Red
	case LevelFatal:
		return fmt.Sprintf("\x1b[35m%s\x1b[0m", level) // Magenta
	default:
		return level
	}
}

// ConsoleFormatter formats log entries for console output with colors
type ConsoleFormatter struct {
	*TextFormatter
}

// NewConsoleFormatter creates a new console formatter
func NewConsoleFormatter() *ConsoleFormatter {
	return &ConsoleFormatter{
		TextFormatter: NewTextFormatterWithOptions(true, true, true, true),
	}
}

// Format formats a log entry for console output
func (f *ConsoleFormatter) Format(entry *LogEntry) ([]byte, error) {
	return f.TextFormatter.Format(entry)
}

// GetName returns the name of the formatter
func (f *ConsoleFormatter) GetName() string {
	return "console"
}

// SimpleFormatter formats log entries in a simple, minimal format
type SimpleFormatter struct{}

// NewSimpleFormatter creates a new simple formatter
func NewSimpleFormatter() *SimpleFormatter {
	return &SimpleFormatter{}
}

// Format formats a log entry in a simple format
func (f *SimpleFormatter) Format(entry *LogEntry) ([]byte, error) {
	output := fmt.Sprintf("%s %s %s",
		entry.Timestamp.Format("2006-01-02 15:04:05"),
		entry.Level.String(),
		entry.Message)

	// Add error if it exists
	if entry.Error != nil {
		output += fmt.Sprintf(" (error: %s)", entry.Error.Error())
	}

	output += "\n"
	return []byte(output), nil
}

// GetName returns the name of the formatter
func (f *SimpleFormatter) GetName() string {
	return "simple"
}
