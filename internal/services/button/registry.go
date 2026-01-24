package button

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"

	domainButton "mispilkabot/internal/domain/button"
)

var (
	ErrLoadFailed = errors.New("failed to load button registry")
	ErrSaveFailed = errors.New("failed to save button registry")
)

// Registry manages button set persistence
type Registry struct {
	filePath string
	mu       sync.RWMutex

	registry *domainButton.ButtonRegistry
}

// NewRegistry creates a new button registry
func NewRegistry(filePath string) *Registry {
	return &Registry{
		filePath: filePath,
		registry: domainButton.NewButtonRegistry(),
	}
}

// Load loads the registry from disk
func (r *Registry) Load() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	data, err := os.ReadFile(r.filePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// Create new registry
			return nil
		}
		return ErrLoadFailed
	}

	if err := json.Unmarshal(data, r.registry); err != nil {
		return ErrLoadFailed
	}

	return nil
}

// Save saves the registry to disk
func (r *Registry) Save() error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(r.filePath), 0755); err != nil {
		return ErrSaveFailed
	}

	data, err := json.MarshalIndent(r.registry, "", "  ")
	if err != nil {
		return ErrSaveFailed
	}

	if err := os.WriteFile(r.filePath, data, 0644); err != nil {
		return ErrSaveFailed
	}

	return nil
}

// Get retrieves a button set by reference
func (r *Registry) Get(ref string) (*domainButton.ButtonSet, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.registry.Get(ref)
}

// Set stores a button set
func (r *Registry) Set(ref string, bs *domainButton.ButtonSet) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.registry.Set(ref, bs)
	return r.saveLocked()
}

// Delete removes a button set
func (r *Registry) Delete(ref string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.registry.Delete(ref)
	return r.saveLocked()
}

// saveLocked saves the registry to disk (caller must hold lock)
func (r *Registry) saveLocked() error {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(r.filePath), 0755); err != nil {
		return ErrSaveFailed
	}

	data, err := json.MarshalIndent(r.registry, "", "  ")
	if err != nil {
		return ErrSaveFailed
	}

	if err := os.WriteFile(r.filePath, data, 0644); err != nil {
		return ErrSaveFailed
	}

	return nil
}
