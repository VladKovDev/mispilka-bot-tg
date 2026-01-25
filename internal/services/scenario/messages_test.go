package scenario

import (
	"path/filepath"
	"testing"

	domainScenario "mispilkabot/internal/domain/scenario"
)

func TestScenarioMessages_LoadSave(t *testing.T) {
	tmpDir := t.TempDir()
	scenarioDir := filepath.Join(tmpDir, "scenarios", "test")
	messagesPath := filepath.Join(scenarioDir, "messages.json")

	// Create messages
	msgs := NewScenarioMessages(messagesPath)
	msgs.MessagesList = []string{"msg_1", "msg_2"}
	msgs.Messages = map[string]domainScenario.MessageData{
		"msg_1": {
			Timing: domainScenario.Timing{Hours: 0, Minutes: 0},
			TemplateFile: "msg_1.md",
		},
		"msg_2": {
			Timing: domainScenario.Timing{Hours: 1, Minutes: 0},
			TemplateFile: "msg_2.md",
		},
	}

	// Save
	err := msgs.Save()
	if err != nil {
		t.Fatalf("Failed to save: %v", err)
	}

	// Load into new messages
	msgs2 := NewScenarioMessages(messagesPath)
	err = msgs2.Load()
	if err != nil {
		t.Fatalf("Failed to load: %v", err)
	}

	// Verify
	if len(msgs2.MessagesList) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(msgs2.MessagesList))
	}
	if msgs2.MessagesList[0] != "msg_1" {
		t.Errorf("Expected msg_1, got %s", msgs2.MessagesList[0])
	}
}
