package logging

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// RotatingFileWriter implements a rotating file writer with compression and retention policies
type RotatingFileWriter struct {
	mu           sync.Mutex
	file         *os.File
	filePath     string
	currentSize  int64
	maxSize      int64
	maxAge       time.Duration
	maxBackups   int
	compress     bool
	creationTime time.Time
}

// NewRotatingFileWriter creates a new rotating file writer
func NewRotatingFileWriter(filePath string, maxSize int64, maxAge time.Duration, maxBackups int, compress bool) (*RotatingFileWriter, error) {
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	// Get current file size
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	return &RotatingFileWriter{
		file:         file,
		filePath:     filePath,
		currentSize:  info.Size(),
		maxSize:      maxSize,
		maxAge:       maxAge,
		maxBackups:   maxBackups,
		compress:     compress,
		creationTime: time.Now(),
	}, nil
}

// Write writes data to the file with rotation logic
func (w *RotatingFileWriter) Write(data []byte) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Check if rotation is needed
	if w.shouldRotate(len(data)) {
		if err := w.rotate(); err != nil {
			// If rotation fails, try to write to the current file anyway
			_, writeErr := w.file.Write(data)
			if writeErr != nil {
				return fmt.Errorf("rotation failed (%v) and write failed (%v)", err, writeErr)
			}
			return fmt.Errorf("rotation failed: %w", err)
		}
	}

	// Write data
	n, err := w.file.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write to log file: %w", err)
	}

	w.currentSize += int64(n)
	return nil
}

// shouldRotate determines if the file needs rotation
func (w *RotatingFileWriter) shouldRotate(dataSize int) bool {
	// Check size-based rotation
	if w.maxSize > 0 && (w.currentSize+int64(dataSize)) > w.maxSize {
		return true
	}

	// Check time-based rotation
	if w.maxAge > 0 && time.Since(w.creationTime) > w.maxAge {
		return true
	}

	return false
}

// rotate performs the actual file rotation
func (w *RotatingFileWriter) rotate() error {
	// Close current file
	if err := w.file.Close(); err != nil {
		return fmt.Errorf("failed to close current log file: %w", err)
	}

	// Create backup
	if err := w.createBackup(); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	// Clean up old backups
	if err := w.cleanupOldBackups(); err != nil {
		return fmt.Errorf("failed to cleanup old backups: %w", err)
	}

	// Create new file
	file, err := os.OpenFile(w.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to create new log file: %w", err)
	}

	w.file = file
	w.currentSize = 0
	w.creationTime = time.Now()

	return nil
}

// createBackup creates a backup of the current log file
func (w *RotatingFileWriter) createBackup() error {
	// Generate backup filename
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	backupPath := fmt.Sprintf("%s.%s", w.filePath, timestamp)

	// Rename current file to backup
	if err := os.Rename(w.filePath, backupPath); err != nil {
		return fmt.Errorf("failed to rename log file: %w", err)
	}

	// Compress if enabled
	if w.compress {
		if err := w.compressFile(backupPath); err != nil {
			// Don't fail the entire rotation if compression fails
			// Just log the error and continue
			fmt.Fprintf(os.Stderr, "Failed to compress backup file: %v\n", err)
		}
	}

	return nil
}

// compressFile compresses a file using gzip
func (w *RotatingFileWriter) compressFile(filePath string) error {
	// Open source file
	srcFile, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer func() {
		_ = srcFile.Close()
	}()

	// Create compressed file
	compressedPath := filePath + ".gz"
	dstFile, err := os.Create(compressedPath)
	if err != nil {
		return fmt.Errorf("failed to create compressed file: %w", err)
	}
	defer func() {
		_ = dstFile.Close()
	}()

	// Create gzip writer
	gzWriter := gzip.NewWriter(dstFile)
	defer func() {
		_ = gzWriter.Close()
	}()

	// Copy data
	if _, err := io.Copy(gzWriter, srcFile); err != nil {
		return fmt.Errorf("failed to compress data: %w", err)
	}

	// Remove original file
	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("failed to remove original file: %w", err)
	}

	return nil
}

// cleanupOldBackups removes old backup files according to retention policy
func (w *RotatingFileWriter) cleanupOldBackups() error {
	if w.maxBackups <= 0 {
		return nil // No retention limit
	}

	// Get all backup files
	dir := filepath.Dir(w.filePath)
	base := filepath.Base(w.filePath)

	files, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	var backupFiles []backupFileInfo
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		name := file.Name()
		// Check if it's a backup file (matches base.filename.timestamp or base.filename.timestamp.gz)
		if strings.HasPrefix(name, base+".") {
			info, err := file.Info()
			if err != nil {
				continue // Skip files we can't get info for
			}
			backupFiles = append(backupFiles, backupFileInfo{name: name, info: info})
		}
	}

	// Sort by modification time (newest first)
	sort.Slice(backupFiles, func(i, j int) bool {
		return backupFiles[i].info.ModTime().After(backupFiles[j].info.ModTime())
	})

	// Remove oldest backups if we have too many
	for i := w.maxBackups; i < len(backupFiles); i++ {
		fullPath := filepath.Join(dir, backupFiles[i].name)
		if err := os.Remove(fullPath); err != nil {
			return fmt.Errorf("failed to remove old backup %s: %w", fullPath, err)
		}
	}

	return nil
}

// ForceRotate forces a rotation to occur
func (w *RotatingFileWriter) ForceRotate() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.rotate()
}

// Flush flushes the file writer
func (w *RotatingFileWriter) Flush() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.file.Sync()
}

// Close closes the file writer
func (w *RotatingFileWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.file.Close()
}

// GetName returns the name of the writer
func (w *RotatingFileWriter) GetName() string {
	return fmt.Sprintf("rotating:%s", w.filePath)
}

// GetFilePath returns the file path
func (w *RotatingFileWriter) GetFilePath() string {
	return w.filePath
}

// backupFileInfo is a helper struct to hold file name and info
type backupFileInfo struct {
	name string
	info os.FileInfo
}
