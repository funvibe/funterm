package repl

import (
	"strings"
)

// MultiLineBuffer represents a buffer for multiline input
type MultiLineBuffer struct {
	lines    []string
	isActive bool
}

// NewMultiLineBuffer creates a new buffer
func NewMultiLineBuffer() *MultiLineBuffer {
	return &MultiLineBuffer{
		lines:    []string{},
		isActive: false,
	}
}

// AddLine adds a line to the buffer and activates it
func (b *MultiLineBuffer) AddLine(line string) {
	b.lines = append(b.lines, line)
	b.isActive = true
}

// GetContent returns the buffer content as a single string
func (b *MultiLineBuffer) GetContent() string {
	return strings.Join(b.lines, "\n")
}

// Clear clears the buffer and deactivates it
func (b *MultiLineBuffer) Clear() {
	b.lines = []string{}
	b.isActive = false
}

// IsActive returns true if the buffer is active
func (b *MultiLineBuffer) IsActive() bool {
	return b.isActive
}

// IsEmpty returns true if the buffer is empty
func (b *MultiLineBuffer) IsEmpty() bool {
	return len(b.lines) == 0
}

// GetLineCount returns the number of lines in the buffer
func (b *MultiLineBuffer) GetLineCount() int {
	return len(b.lines)
}

// GetLines returns all lines in the buffer
func (b *MultiLineBuffer) GetLines() []string {
	return b.lines
}

// RemoveLastLine removes and returns the last line from the buffer
func (b *MultiLineBuffer) RemoveLastLine() string {
	if len(b.lines) == 0 {
		return ""
	}

	lastIndex := len(b.lines) - 1
	lastLine := b.lines[lastIndex]
	b.lines = b.lines[:lastIndex]

	// Deactivate buffer if it becomes empty
	if len(b.lines) == 0 {
		b.isActive = false
	}

	return lastLine
}

// SetLines sets the buffer content from a slice of lines
func (b *MultiLineBuffer) SetLines(lines []string) {
	b.lines = lines
	b.isActive = len(lines) > 0
}

// SetActive sets the active state of the buffer
func (b *MultiLineBuffer) SetActive(active bool) {
	b.isActive = active
}
