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

// MessageData defines a message in the flow
type MessageData struct {
	Timing         domainScenario.Timing       `json:"timing"`
	TemplateFile   string                       `json:"template_file,omitempty"`
	Photos         []string                     `json:"photos,omitempty"`
	InlineKeyboard *InlineKeyboardConfig        `json:"inline_keyboard,omitempty"`
}

// InlineKeyboardConfig defines inline keyboard structure
type InlineKeyboardConfig struct {
	ButtonSetRef string                    `json:"button_set_ref,omitempty"`
	Rows         []InlineKeyboardRowConfig `json:"rows,omitempty"`
}

// InlineKeyboardRowConfig defines a keyboard row
type InlineKeyboardRowConfig struct {
	Buttons []InlineKeyboardButtonConfig `json:"buttons"`
}

// InlineKeyboardButtonConfig defines a button
type InlineKeyboardButtonConfig struct {
	Type     string `json:"type"`
	Text     string `json:"text"`
	URL      string `json:"url,omitempty"`
	Callback string `json:"callback,omitempty"`
}

// ScenarioMessages manages scenario messages persistence
type ScenarioMessages struct {
	filePath string
	mu       sync.RWMutex

	MessagesList []string              `json:"messages_list"`
	Messages     map[string]MessageData `json:"messages"`
}

// NewScenarioMessages creates a new scenario messages
func NewScenarioMessages(filePath string) *ScenarioMessages {
	return &ScenarioMessages{
		filePath:     filePath,
		Messages:     make(map[string]MessageData),
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
	m.mu.RLock()
	defer m.mu.RUnlock()

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
func (m *ScenarioMessages) ToDomain() domainScenario.ScenarioMessages {
	msgs := domainScenario.ScenarioMessages{
		MessagesList: m.MessagesList,
		Messages:     make(map[string]domainScenario.MessageData),
	}
	for id, md := range m.Messages {
		msgs.Messages[id] = domainScenario.MessageData{
			Timing:         md.Timing,
			TemplateFile:   md.TemplateFile,
			Photos:         md.Photos,
			InlineKeyboard: convertInlineKeyboard(md.InlineKeyboard),
		}
	}
	return msgs
}

func convertInlineKeyboard(ik *InlineKeyboardConfig) *domainScenario.InlineKeyboardConfig {
	if ik == nil {
		return nil
	}
	domainIK := &domainScenario.InlineKeyboardConfig{
		Rows: make([]domainScenario.InlineKeyboardRowConfig, len(ik.Rows)),
	}
	for i, row := range ik.Rows {
		domainIK.Rows[i] = domainScenario.InlineKeyboardRowConfig{
			Buttons: make([]domainScenario.InlineKeyboardButtonConfig, len(row.Buttons)),
		}
		for j, btn := range row.Buttons {
			domainIK.Rows[i].Buttons[j] = domainScenario.InlineKeyboardButtonConfig{
				Type:     btn.Type,
				Text:     btn.Text,
				URL:      btn.URL,
				Callback: btn.Callback,
			}
		}
	}
	return domainIK
}
