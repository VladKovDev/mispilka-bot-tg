package scenario

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"

	domainScenario "mispilkabot/internal/domain/scenario"
)

var (
	ErrMessagesLoadFailed = errors.New("failed to load messages")
	ErrMessagesSaveFailed = errors.New("failed to save messages")
)

// ScenarioMessages manages scenario messages persistence
// Uses domain types directly to avoid duplication
type ScenarioMessages struct {
	filePath string
	mu       sync.RWMutex

	MessagesList []string                                `json:"messages_list"`
	Messages     map[string]domainScenario.MessageData `json:"messages"`
}

// NewScenarioMessages creates a new scenario messages
func NewScenarioMessages(filePath string) *ScenarioMessages {
	return &ScenarioMessages{
		filePath:     filePath,
		Messages:     make(map[string]domainScenario.MessageData),
		MessagesList: make([]string, 0),
	}
}

// Load loads the messages from disk
func (m *ScenarioMessages) Load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := os.ReadFile(m.filePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return domainScenario.ErrMessageNotFound
		}
		return ErrMessagesLoadFailed
	}

	if err := json.Unmarshal(data, m); err != nil {
		return ErrMessagesLoadFailed
	}

	return nil
}

// Save saves the messages to disk
func (m *ScenarioMessages) Save() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(m.filePath), 0755); err != nil {
		return ErrMessagesSaveFailed
	}

	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return ErrMessagesSaveFailed
	}

	if err := os.WriteFile(m.filePath, data, 0644); err != nil {
		return ErrMessagesSaveFailed
	}

	return nil
}

// ToDomain converts to domain scenario messages
// Now returns a copy since we're using domain types directly
func (m *ScenarioMessages) ToDomain() domainScenario.ScenarioMessages {
	msgs := domainScenario.ScenarioMessages{
		MessagesList: make([]string, len(m.MessagesList)),
		Messages:     make(map[string]domainScenario.MessageData, len(m.Messages)),
	}
	copy(msgs.MessagesList, m.MessagesList)
	for id, md := range m.Messages {
		msgs.Messages[id] = md
	}
	return msgs
}
