package broadcast

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"

	domainBroadcast "mispilkabot/internal/domain/broadcast"
)

var (
	ErrBroadcastNotFound = errors.New("broadcast not found")
	ErrLoadFailed        = errors.New("failed to load broadcast registry")
	ErrSaveFailed        = errors.New("failed to save broadcast registry")
)

// Registry manages broadcast persistence
type Registry struct {
	filePath string
	mu       sync.RWMutex

	registry *domainBroadcast.BroadcastRegistry
}

// NewRegistry creates a new broadcast registry
func NewRegistry(filePath string) *Registry {
	return &Registry{
		filePath: filePath,
		registry: domainBroadcast.NewBroadcastRegistry(),
	}
}

// Load loads broadcasts from disk
func (r *Registry) Load() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.loadLocked()
}

// loadLocked loads broadcasts from disk (caller must hold lock)
func (r *Registry) loadLocked() error {
	data, err := os.ReadFile(r.filePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			r.registry = domainBroadcast.NewBroadcastRegistry()
			return nil
		}
		return ErrLoadFailed
	}

	if err := json.Unmarshal(data, r.registry); err != nil {
		return ErrLoadFailed
	}

	return nil
}

// Save saves broadcasts to disk
func (r *Registry) Save() error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.saveLocked()
}

// saveLocked saves broadcasts to disk (caller must hold lock)
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

// Add adds a broadcast to the registry
func (r *Registry) Add(bc *domainBroadcast.Broadcast) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := r.loadLocked(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	// Check for duplicate
	if _, exists := r.registry.Get(bc.ID); exists {
		return errors.New("broadcast already exists")
	}

	r.registry.Add(bc)

	return r.saveLocked()
}

// Get retrieves a broadcast by ID
func (r *Registry) Get(id string) (*domainBroadcast.Broadcast, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.registry.Get(id)
}

// List returns all broadcasts
func (r *Registry) List() []*domainBroadcast.Broadcast {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.registry.List()
}

// Delete removes a broadcast from the registry
func (r *Registry) Delete(id string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := r.loadLocked(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return false
	}

	deleted := r.registry.Delete(id)
	if !deleted {
		return false
	}

	if err := r.saveLocked(); err != nil {
		return false
	}

	return true
}
