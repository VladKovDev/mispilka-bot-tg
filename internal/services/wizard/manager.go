package wizard

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	ErrWizardNotFound = errors.New("wizard not found")
	ErrWizardExpired  = errors.New("wizard expired")
	ErrSaveFailed     = errors.New("failed to save wizard state")
)

// Manager manages wizard states
type Manager struct {
	wizardsDir string
	mu         sync.RWMutex
	states     map[string]*WizardState // userID -> WizardState
}

// NewManager creates a new wizard manager
func NewManager(wizardsDir string) *Manager {
	return &Manager{
		wizardsDir: wizardsDir,
		states:     make(map[string]*WizardState),
	}
}

// Initialize initializes the manager (loads existing states)
func (m *Manager) Initialize() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Ensure directory exists
	if err := os.MkdirAll(m.wizardsDir, 0755); err != nil {
		return err
	}

	// Clean up expired wizards from disk
	entries, err := os.ReadDir(m.wizardsDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Load state
		filePath := filepath.Join(m.wizardsDir, entry.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}

		var state WizardState
		if err := json.Unmarshal(data, &state); err != nil {
			continue
		}

		// Check if expired
		if state.Expired() {
			_ = os.Remove(filePath)
			continue
		}

		// Load into memory
		m.states[state.UserID] = &state
	}

	return nil
}

// Start starts a new wizard for a user
func (m *Manager) Start(userID string, wizardType WizardType) (*WizardState, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Cancel existing wizard if any
	if existing, ok := m.states[userID]; ok {
		_ = m.deleteExisting(existing.UserID)
	}

	// Create new wizard state
	state := &WizardState{
		UserID:      userID,
		WizardType:  wizardType,
		CurrentStep: getFirstStep(wizardType),
		StartedAt:   time.Now(),
		Timeout:     30 * time.Minute,
		Data:        make(map[string]interface{}),
	}

	m.states[userID] = state

	// Save to disk
	if err := m.saveState(state); err != nil {
		delete(m.states, userID)
		return nil, err
	}

	return state, nil
}

// Get retrieves a wizard state
func (m *Manager) Get(userID string) (*WizardState, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	state, ok := m.states[userID]
	if !ok {
		return nil, ErrWizardNotFound
	}

	// Check if expired
	if state.Expired() {
		go m.Cancel(userID)
		return nil, ErrWizardExpired
	}

	return state, nil
}

// Update updates a wizard state
func (m *Manager) Update(userID string, state *WizardState) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.states[userID]; !ok {
		return ErrWizardNotFound
	}

	// Reset timeout on update
	state.ResetTimeout()

	m.states[userID] = state
	return m.saveState(state)
}

// Advance advances a wizard to the next step
func (m *Manager) Advance(userID string, nextStep WizardStep) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	state, ok := m.states[userID]
	if !ok {
		return ErrWizardNotFound
	}

	state.CurrentStep = nextStep
	state.ResetTimeout()
	return m.saveState(state)
}

// Cancel cancels a wizard
func (m *Manager) Cancel(userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	_, ok := m.states[userID]
	if !ok {
		return ErrWizardNotFound
	}

	return m.deleteExisting(userID)
}

// deleteExisting removes wizard from memory and disk
func (m *Manager) deleteExisting(userID string) error {
	delete(m.states, userID)

	filePath := filepath.Join(m.wizardsDir, userID+".json")
	if err := os.Remove(filePath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return ErrSaveFailed
	}

	return nil
}

// saveState saves wizard state to disk
func (m *Manager) saveState(state *WizardState) error {
	filePath := filepath.Join(m.wizardsDir, state.UserID+".json")
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return ErrSaveFailed
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return ErrSaveFailed
	}

	return nil
}

// CleanupExpired removes expired wizard states
func (m *Manager) CleanupExpired() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for userID, state := range m.states {
		if state.Expired() {
			_ = m.deleteExisting(userID)
		}
	}
}

// getFirstStep returns the first step for a wizard type
func getFirstStep(wizardType WizardType) WizardStep {
	switch wizardType {
	case WizardTypeCreateScenario:
		return StepScenarioName
	case WizardTypeCreateBroadcast:
		return StepBroadcastName
	default:
		return ""
	}
}
