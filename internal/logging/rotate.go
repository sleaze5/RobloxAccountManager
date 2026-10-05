package logging

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// maxHeldBytes bounds the entries kept in memory while a writer is held.
const maxHeldBytes = 1 << 20

type rotatingWriter struct {
	mu       sync.Mutex
	path     string
	maxBytes int64
	retained int
	file     *os.File
	size     int64
	header   []byte
	resume   bool
	closed   bool
	held     bool
	pending  []byte
}

func newRotatingWriter(path string, maxBytes int64, retained int, header []byte, resume bool) (*rotatingWriter, error) {
	if maxBytes <= 0 {
		return nil, fmt.Errorf("log size limit must be positive")
	}
	if retained < 1 {
		return nil, fmt.Errorf("retained log count must be positive")
	}
	return &rotatingWriter{path: path, maxBytes: maxBytes, retained: retained, header: header, resume: resume}, nil
}

func (writer *rotatingWriter) ensureOpen() error {
	writer.mu.Lock()
	defer writer.mu.Unlock()
	if writer.closed {
		return os.ErrClosed
	}
	if writer.file != nil || writer.held {
		return nil
	}
	return writer.open()
}

// release opens the log file and writes the entries kept while the writer was held.
func (writer *rotatingWriter) release(open bool) error {
	writer.mu.Lock()
	defer writer.mu.Unlock()
	if writer.closed {
		return os.ErrClosed
	}
	if !writer.held {
		return nil
	}
	if !open && len(writer.pending) == 0 {
		writer.held = false
		return nil
	}
	if err := writer.open(); err != nil {
		return err
	}
	pending := writer.pending
	writer.held, writer.pending = false, nil
	written, err := writer.file.Write(pending)
	writer.size += int64(written)
	if err != nil {
		return fmt.Errorf("write held log entries: %w", err)
	}
	return nil
}

func (writer *rotatingWriter) Write(data []byte) (int, error) {
	writer.mu.Lock()
	defer writer.mu.Unlock()
	if writer.held && !writer.closed {
		if len(writer.pending)+len(data) <= maxHeldBytes {
			writer.pending = append(writer.pending, data...)
		}
		return len(data), nil
	}
	if writer.file == nil {
		return 0, os.ErrClosed
	}
	if writer.size > int64(len(writer.header)) && writer.size+int64(len(data)) > writer.maxBytes {
		if err := writer.rotate(); err != nil {
			return 0, err
		}
	}
	written, err := writer.file.Write(data)
	writer.size += int64(written)
	return written, err
}

func (writer *rotatingWriter) Close() error {
	writer.mu.Lock()
	defer writer.mu.Unlock()
	writer.closed = true
	if writer.file == nil {
		return nil
	}
	err := writer.file.Close()
	writer.file = nil
	return err
}

func (writer *rotatingWriter) Sync() error {
	writer.mu.Lock()
	defer writer.mu.Unlock()
	if writer.file == nil {
		return nil
	}
	return writer.file.Sync()
}

func (writer *rotatingWriter) open() error {
	flags := os.O_CREATE | os.O_APPEND | os.O_WRONLY
	if !writer.resume {
		flags |= os.O_EXCL
	}
	file, err := os.OpenFile(writer.path, flags, 0o600)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return fmt.Errorf("inspect log file: %w", err)
	}
	writer.file = file
	writer.size = info.Size()
	writer.resume = false
	if writer.size == 0 {
		written, err := file.Write(writer.header)
		writer.size = int64(written)
		if err != nil {
			_ = file.Close()
			writer.file = nil
			return fmt.Errorf("write launch header: %w", err)
		}
	}
	return nil
}

func (writer *rotatingWriter) rotate() error {
	if err := writer.file.Close(); err != nil {
		return fmt.Errorf("close log file for rotation: %w", err)
	}
	writer.file = nil
	oldest := rotatedPath(writer.path, writer.retained)
	if err := os.Remove(oldest); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove oldest log file: %w", err)
	}
	for index := writer.retained - 1; index >= 1; index-- {
		from := rotatedPath(writer.path, index)
		to := rotatedPath(writer.path, index+1)
		if err := os.Rename(from, to); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("rotate log file: %w", err)
		}
	}
	if err := os.Rename(writer.path, rotatedPath(writer.path, 1)); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("archive log file: %w", err)
	}
	return writer.open()
}

func rotatedPath(path string, index int) string {
	extension := filepath.Ext(path)
	base := path[:len(path)-len(extension)]
	return fmt.Sprintf("%s.%d%s", base, index, extension)
}
