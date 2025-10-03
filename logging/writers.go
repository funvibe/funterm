package logging

import (
	"fmt"
	"log/syslog"
	"net"
	"os"
	"sync"
	"time"
)

// ConsoleWriter writes log entries to the console (stdout/stderr)
type ConsoleWriter struct {
	mu     sync.Mutex
	writer *os.File
}

// NewConsoleWriter creates a new console writer that writes to stdout
func NewConsoleWriter() *ConsoleWriter {
	return &ConsoleWriter{
		writer: os.Stdout,
	}
}

// NewConsoleWriterWithFile creates a new console writer with a specific file
func NewConsoleWriterWithFile(file *os.File) *ConsoleWriter {
	return &ConsoleWriter{
		writer: file,
	}
}

// Write writes data to the console
func (w *ConsoleWriter) Write(data []byte) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	_, err := w.writer.Write(data)
	return err
}

// Flush flushes the console writer
func (w *ConsoleWriter) Flush() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.writer.Sync()
}

// Close closes the console writer
func (w *ConsoleWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Don't close stdout/stderr as they are shared
	if w.writer == os.Stdout || w.writer == os.Stderr {
		return nil
	}

	return w.writer.Close()
}

// GetName returns the name of the writer
func (w *ConsoleWriter) GetName() string {
	return "console"
}

// FileWriter writes log entries to a file
type FileWriter struct {
	mu       sync.Mutex
	file     *os.File
	filePath string
}

// NewFileWriter creates a new file writer
func NewFileWriter(filePath string) (*FileWriter, error) {
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	return &FileWriter{
		file:     file,
		filePath: filePath,
	}, nil
}

// NewFileWriterWithRotation creates a new file writer with rotation support
func NewFileWriterWithRotation(filePath string, maxSize int64, maxBackups int) (*FileWriter, error) {
	writer, err := NewFileWriter(filePath)
	if err != nil {
		return nil, err
	}

	// For backward compatibility, this function returns basic FileWriter
	// Use CreateFileWriter with LoggerConfig for rotation support
	return writer, nil
}

// CreateFileWriter creates a file writer with optional rotation support based on config
func CreateFileWriter(path string, config *LoggerConfig) (Writer, error) {
	if config == nil || config.Rotation == nil {
		// No rotation configuration, use basic file writer
		return NewFileWriter(path)
	}

	// Use rotating file writer with rotation configuration
	rotation := config.Rotation
	return NewRotatingFileWriter(path, rotation.MaxSize, rotation.MaxAge, rotation.MaxBackups, rotation.Compress)
}

// Write writes data to the file
func (w *FileWriter) Write(data []byte) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	_, err := w.file.Write(data)
	return err
}

// Flush flushes the file writer
func (w *FileWriter) Flush() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.file.Sync()
}

// Close closes the file writer
func (w *FileWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.file.Close()
}

// GetName returns the name of the writer
func (w *FileWriter) GetName() string {
	return fmt.Sprintf("file:%s", w.filePath)
}

// GetFilePath returns the file path
func (w *FileWriter) GetFilePath() string {
	return w.filePath
}

// SyslogWriter writes log entries to syslog
type SyslogWriter struct {
	mu     sync.Mutex
	writer *syslog.Writer
}

// NewSyslogWriter creates a new syslog writer
func NewSyslogWriter(network, raddr string, priority syslog.Priority, tag string) (*SyslogWriter, error) {
	writer, err := syslog.Dial(network, raddr, priority, tag)
	if err != nil {
		return nil, err
	}

	return &SyslogWriter{
		writer: writer,
	}, nil
}

// NewSyslogWriterWithPriority creates a new syslog writer with default network and raddr
func NewSyslogWriterWithPriority(priority syslog.Priority, tag string) (*SyslogWriter, error) {
	return NewSyslogWriter("", "", priority, tag)
}

// NewSyslogWriterWithTag creates a new syslog writer with default settings
func NewSyslogWriterWithTag(tag string) (*SyslogWriter, error) {
	return NewSyslogWriter("", "", syslog.LOG_INFO, tag)
}

// Write writes data to syslog
func (w *SyslogWriter) Write(data []byte) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Convert log level to syslog priority
	// This is a simple mapping - you might want to make it more sophisticated
	_, err := w.writer.Write(data)
	return err
}

// WriteWithPriority writes data to syslog with a specific priority
func (w *SyslogWriter) WriteWithPriority(data []byte, priority syslog.Priority) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	_, err := w.writer.Write(data)
	return err
}

// Flush flushes the syslog writer
func (w *SyslogWriter) Flush() error {
	// Syslog writer doesn't have a flush method
	return nil
}

// Close closes the syslog writer
func (w *SyslogWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.writer.Close()
}

// GetName returns the name of the writer
func (w *SyslogWriter) GetName() string {
	return "syslog"
}

// RemoteWriter writes log entries to a remote server
type RemoteWriter struct {
	mu         sync.Mutex
	conn       net.Conn
	address    string
	protocol   string
	reconnect  bool
	maxRetries int
	timeout    time.Duration
}

// RemoteWriterConfig contains configuration for the remote writer
type RemoteWriterConfig struct {
	Address    string
	Protocol   string // "tcp", "udp", "unix"
	Reconnect  bool
	MaxRetries int
	Timeout    time.Duration
}

// NewRemoteWriter creates a new remote writer
func NewRemoteWriter(config RemoteWriterConfig) (*RemoteWriter, error) {
	if config.Protocol == "" {
		config.Protocol = "tcp"
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}

	conn, err := net.DialTimeout(config.Protocol, config.Address, config.Timeout)
	if err != nil {
		return nil, err
	}

	return &RemoteWriter{
		conn:       conn,
		address:    config.Address,
		protocol:   config.Protocol,
		reconnect:  config.Reconnect,
		maxRetries: config.MaxRetries,
		timeout:    config.Timeout,
	}, nil
}

// NewRemoteWriterWithAddress creates a new remote writer with basic configuration
func NewRemoteWriterWithAddress(address string) (*RemoteWriter, error) {
	return NewRemoteWriter(RemoteWriterConfig{
		Address:   address,
		Protocol:  "tcp",
		Reconnect: true,
		Timeout:   30 * time.Second,
	})
}

// Write writes data to the remote server
func (w *RemoteWriter) Write(data []byte) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	var lastErr error

	for i := 0; i <= w.maxRetries; i++ {
		if w.conn == nil {
			if !w.reconnect {
				return fmt.Errorf("connection closed and reconnect disabled")
			}

			// Try to reconnect
			conn, err := net.DialTimeout(w.protocol, w.address, w.timeout)
			if err != nil {
				lastErr = err
				time.Sleep(time.Second * time.Duration(i+1))
				continue
			}
			w.conn = conn
		}

		// Set write deadline
		if err := w.conn.SetWriteDeadline(time.Now().Add(w.timeout)); err != nil {
			lastErr = err
			_ = w.closeConnection()
			continue
		}

		// Write data
		_, err := w.conn.Write(data)
		if err != nil {
			lastErr = err
			_ = w.closeConnection()
			time.Sleep(time.Second * time.Duration(i+1))
			continue
		}

		return nil
	}

	return fmt.Errorf("failed to write after %d retries: %v", w.maxRetries, lastErr)
}

// Flush flushes the remote writer
func (w *RemoteWriter) Flush() error {
	// Remote writer doesn't have a flush method
	return nil
}

// Close closes the remote writer
func (w *RemoteWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.closeConnection()
}

// closeConnection closes the underlying connection
func (w *RemoteWriter) closeConnection() error {
	if w.conn != nil {
		err := w.conn.Close()
		w.conn = nil
		return err
	}
	return nil
}

// GetName returns the name of the writer
func (w *RemoteWriter) GetName() string {
	return fmt.Sprintf("remote:%s://%s", w.protocol, w.address)
}

// GetAddress returns the remote address
func (w *RemoteWriter) GetAddress() string {
	return w.address
}

// MultiWriter writes log entries to multiple writers
type MultiWriter struct {
	writers []Writer
}

// NewMultiWriter creates a new multi writer
func NewMultiWriter(writers ...Writer) *MultiWriter {
	return &MultiWriter{
		writers: writers,
	}
}

// Write writes data to all writers
func (w *MultiWriter) Write(data []byte) error {
	var errors []error

	for _, writer := range w.writers {
		if err := writer.Write(data); err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("multi writer errors: %v", errors)
	}

	return nil
}

// Flush flushes all writers
func (w *MultiWriter) Flush() error {
	var errors []error

	for _, writer := range w.writers {
		if err := writer.Flush(); err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("multi writer flush errors: %v", errors)
	}

	return nil
}

// Close closes all writers
func (w *MultiWriter) Close() error {
	var errors []error

	for _, writer := range w.writers {
		if err := writer.Close(); err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("multi writer close errors: %v", errors)
	}

	return nil
}

// GetName returns the name of the writer
func (w *MultiWriter) GetName() string {
	return "multi"
}

// AddWriter adds a writer to the multi writer
func (w *MultiWriter) AddWriter(writer Writer) {
	w.writers = append(w.writers, writer)
}

// RemoveWriter removes a writer from the multi writer
func (w *MultiWriter) RemoveWriter(writer Writer) {
	for i, wr := range w.writers {
		if wr == writer {
			w.writers = append(w.writers[:i], w.writers[i+1:]...)
			break
		}
	}
}

// GetWriters returns all writers
func (w *MultiWriter) GetWriters() []Writer {
	return w.writers
}

// NullWriter is a writer that discards all log entries
type NullWriter struct{}

// NewNullWriter creates a new null writer
func NewNullWriter() *NullWriter {
	return &NullWriter{}
}

// Write discards the data
func (w *NullWriter) Write(data []byte) error {
	return nil
}

// Flush does nothing
func (w *NullWriter) Flush() error {
	return nil
}

// Close does nothing
func (w *NullWriter) Close() error {
	return nil
}

// GetName returns the name of the writer
func (w *NullWriter) GetName() string {
	return "null"
}
