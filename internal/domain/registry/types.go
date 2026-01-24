package registry

import (
	"errors"

	"mispilkabot/internal/domain/scenario"
)

var (
	ErrScenarioNotFound       = errors.New("scenario not found")
	ErrCannotDeleteDefault     = errors.New("cannot delete default scenario")
	ErrDefaultScenarioNotFound = errors.New("default scenario not found")
)

// ScenarioRegistry manages all scenarios
type ScenarioRegistry struct {
	Scenarios         map[string]*scenario.Scenario `json:"scenarios"`
	DefaultScenarioID string                        `json:"default_scenario_id"`
}

// NewScenarioRegistry creates a new registry
func NewScenarioRegistry() *ScenarioRegistry {
	return &ScenarioRegistry{
		Scenarios: make(map[string]*scenario.Scenario),
	}
}

// Get retrieves a scenario by ID
func (r *ScenarioRegistry) Get(id string) (*scenario.Scenario, error) {
	sc, ok := r.Scenarios[id]
	if !ok {
		return nil, ErrScenarioNotFound
	}
	return sc, nil
}

// GetDefault retrieves the default scenario
func (r *ScenarioRegistry) GetDefault() (*scenario.Scenario, error) {
	if r.DefaultScenarioID == "" {
		return nil, ErrDefaultScenarioNotFound
	}
	return r.Get(r.DefaultScenarioID)
}

// SetDefault sets the default scenario
func (r *ScenarioRegistry) SetDefault(id string) error {
	if _, ok := r.Scenarios[id]; !ok {
		return ErrScenarioNotFound
	}
	r.DefaultScenarioID = id
	return nil
}

// List returns all scenarios
func (r *ScenarioRegistry) List() []*scenario.Scenario {
	scenarios := make([]*scenario.Scenario, 0, len(r.Scenarios))
	for _, sc := range r.Scenarios {
		scenarios = append(scenarios, sc)
	}
	return scenarios
}

// Add adds a scenario to the registry
func (r *ScenarioRegistry) Add(sc *scenario.Scenario) error {
	if err := sc.Validate(); err != nil {
		return err
	}
	r.Scenarios[sc.ID] = sc
	return nil
}

// Delete removes a scenario from the registry
func (r *ScenarioRegistry) Delete(id string) error {
	if id == r.DefaultScenarioID {
		return ErrCannotDeleteDefault
	}
	if _, ok := r.Scenarios[id]; !ok {
		return ErrScenarioNotFound
	}
	delete(r.Scenarios, id)
	return nil
}
