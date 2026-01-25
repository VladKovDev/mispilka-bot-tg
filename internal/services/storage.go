package services

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

func ReadJSON[T any](path string) (T, error) {
	var data T
	raw, err := os.ReadFile(path)
	if err != nil {
		return data, fmt.Errorf("read JSON file %q: %w", path, err)
	}

	if err := json.Unmarshal(raw, &data); err != nil {
		return data, fmt.Errorf("unmarshal JSON file %q: %w", path, err)
	}
	return data, nil
}

func WriteJSON[T any](path string, data T) error {
	raw, err := json.MarshalIndent(data, "", " ")
	if err != nil {
		return fmt.Errorf("marshal JSON file %q: %w", path, err)
	}

	err = os.WriteFile(path, raw, 0644)
	if err != nil {
		return fmt.Errorf("write JSON file %q: %w", path, err)
	}

	return nil
}

func ReadJSONRetry[T any](path string, attempts int) (T, error) {
	var data T
	var err error
	for i := 0; i < attempts; i++ {
		data, err = ReadJSON[T](path)
		if err == nil {
			return data, nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return data, fmt.Errorf("ReadJSON failed after %d attempts: %w", attempts, err)
}

func WriteJSONRetry[T any](path string, data T, attempts int) error {
	var err error
	for i := 0; i < attempts; i++ {
		err = WriteJSON(path, data)
		if err == nil {
			return nil
		}
		// Exponential backoff: 100ms, 200ms, 300ms...
		if i < attempts-1 {
			time.Sleep(time.Duration(100*(i+1)) * time.Millisecond)
		}
	}
	return fmt.Errorf("WriteJSON failed after %d attempts: %w", attempts, err)
}

func CheckStorage(path string) error {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, create it
			_, err = os.Create(path)
			if err != nil {
				return fmt.Errorf("failed to create file %q: %w", path, err)
			}
			log.Printf("created file: %v", path)

			// Initialize with empty struct
			v := struct{}{}
			if err := WriteJSON(path, v); err != nil {
				return fmt.Errorf("failed to initialize file %q: %w", path, err)
			}
		} else {
			// Some other error occurred
			return fmt.Errorf("failed to stat file %q: %w", path, err)
		}
	}
	return nil
}

// AtomicWriteFile writes data to a file atomically using a temporary file.
// This ensures that the original file is never corrupted if the process crashes
// during the write operation.
func AtomicWriteFile(path string, data []byte) error {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write to temporary file
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	// Atomic rename (POSIX guarantees this is atomic)
	if err := os.Rename(tmpPath, path); err != nil {
		// Clean up temp file if rename fails
		os.Remove(tmpPath)
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// FileLock provides file-level locking using flock (POSIX advisory locking)
type FileLock struct {
	file *os.File
}

// NewFileLock creates a new file lock for the given path
func NewFileLock(path string) (*FileLock, error) {
	// Open file (create if doesn't exist)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open file for locking: %w", err)
	}

	return &FileLock{file: file}, nil
}

// Lock acquires an exclusive lock on the file (blocks until acquired)
func (fl *FileLock) Lock() error {
	if err := syscall.Flock(int(fl.file.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("failed to lock file: %w", err)
	}
	return nil
}

// TryLock attempts to acquire an exclusive lock without blocking
func (fl *FileLock) TryLock() error {
	if err := syscall.Flock(int(fl.file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		if err == syscall.EWOULDBLOCK {
			return fmt.Errorf("file is already locked")
		}
		return fmt.Errorf("failed to try lock file: %w", err)
	}
	return nil
}

// Unlock releases the lock on the file
func (fl *FileLock) Unlock() error {
	if err := syscall.Flock(int(fl.file.Fd()), syscall.LOCK_UN); err != nil {
		return fmt.Errorf("failed to unlock file: %w", err)
	}
	return nil
}

// Close closes the file handle (also releases the lock)
func (fl *FileLock) Close() error {
	return fl.file.Close()
}
