package template

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

var (
	// ErrTemplateNotFound is returned when a template file doesn't exist
	ErrTemplateNotFound = errors.New("template not found")
	// ErrTemplateReadFailed is returned when reading a template file fails
	ErrTemplateReadFailed = errors.New("failed to read template")
)

// Loader handles loading template files from disk
type Loader struct {
	baseDir string
	mu      sync.RWMutex
	cache   map[string]string // Optional caching
}

// NewLoader creates a new template loader for the given directory
func NewLoader(baseDir string) *Loader {
	return &Loader{
		baseDir: baseDir,
		cache:   make(map[string]string),
	}
}

// Load loads a template file by name from the base directory
// The fileName should include the file extension (e.g., "welcome.md")
func (l *Loader) Load(fileName string) (string, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	// Construct full path
	fullPath := filepath.Join(l.baseDir, fileName)

	// Check if file exists
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return "", fmt.Errorf("%w: %s", ErrTemplateNotFound, fileName)
	}

	// Read file content
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("%w: %s: %v", ErrTemplateReadFailed, fileName, err)
	}

	return string(content), nil
}

// MustLoad is like Load but panics on error
// Useful for initialization when template files are expected to exist
func (l *Loader) MustLoad(fileName string) string {
	content, err := l.Load(fileName)
	if err != nil {
		panic(err)
	}
	return content
}

// LoadWithCache loads a template file with caching
// Subsequent calls for the same file return cached content
func (l *Loader) LoadWithCache(fileName string) (string, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Check cache first
	if content, ok := l.cache[fileName]; ok {
		return content, nil
	}

	// Load from disk
	content, err := l.Load(fileName)
	if err != nil {
		return "", err
	}

	// Cache the result
	l.cache[fileName] = content
	return content, nil
}

// ClearCache clears the template cache
func (l *Loader) ClearCache() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.cache = make(map[string]string)
}

// GetFullPath returns the full path for a template file
func (l *Loader) GetFullPath(fileName string) string {
	return filepath.Join(l.baseDir, fileName)
}

// Exists checks if a template file exists
func (l *Loader) Exists(fileName string) bool {
	fullPath := filepath.Join(l.baseDir, fileName)
	_, err := os.Stat(fullPath)
	return !os.IsNotExist(err)
}
