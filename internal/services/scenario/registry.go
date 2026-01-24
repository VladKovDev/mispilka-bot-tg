package scenario

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"

	"mispilkabot/internal/domain/scenario"
)

var (
	ErrRegistryLoadFailed = errors.New("failed to load registry")
	ErrRegistrySaveFailed = errors.New("failed to save registry")
)

// Registry manages scenario persistence
type Registry struct {
	filePath string
	mu       sync.RWMutex

	Scenarios         map[string]*scenario.Scenario `json:"scenarios"`
	DefaultScenarioID string                        `json:"default_scenario_id"`
}

// NewRegistry creates a new registry
func NewRegistry(filePath string) *Registry {
	return &Registry{
		filePath:  filePath,
		Scenarios: make(map[string]*scenario.Scenario),
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
		return ErrRegistryLoadFailed
	}

	if err := json.Unmarshal(data, r); err != nil {
		return ErrRegistryLoadFailed
	}

	return nil
}

// Save saves the registry to disk
func (r *Registry) Save() error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(r.filePath), 0755); err != nil {
		return ErrRegistrySaveFailed
	}

	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return ErrRegistrySaveFailed
	}

	if err := os.WriteFile(r.filePath, data, 0644); err != nil {
		return ErrRegistrySaveFailed
	}

	return nil
}

// Get retrieves a scenario by ID
func (r *Registry) Get(id string) (*scenario.Scenario, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	sc, ok := r.Scenarios[id]
	if !ok {
		return nil, scenario.ErrScenarioNotFound
	}
	return sc, nil
}

// GetDefault retrieves the default scenario
func (r *Registry) GetDefault() (*scenario.Scenario, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.DefaultScenarioID == "" {
		return nil, errors.New("no default scenario set")
	}
	sc, ok := r.Scenarios[r.DefaultScenarioID]
	if !ok {
		return nil, scenario.ErrScenarioNotFound
	}
	return sc, nil
}

// SetDefault sets the default scenario
func (r *Registry) SetDefault(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.Scenarios[id]; !ok {
		return scenario.ErrScenarioNotFound
	}
	r.DefaultScenarioID = id
	return r.Save()
}

// Add adds a scenario to the registry
func (r *Registry) Add(sc *scenario.Scenario) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := sc.Validate(); err != nil {
		return err
	}
	r.Scenarios[sc.ID] = sc
	return r.Save()
}

// List returns all scenarios
func (r *Registry) List() []*scenario.Scenario {
	r.mu.RLock()
	defer r.mu.RUnlock()

	scenarios := make([]*scenario.Scenario, 0, len(r.Scenarios))
	for _, sc := range r.Scenarios {
		scenarios = append(scenarios, sc)
	}
	return scenarios
}

// Delete removes a scenario from the registry
func (r *Registry) Delete(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if id == r.DefaultScenarioID {
		return errors.New("cannot delete default scenario")
	}
	if _, ok := r.Scenarios[id]; !ok {
		return scenario.ErrScenarioNotFound
	}
	delete(r.Scenarios, id)
	return r.Save()
}
