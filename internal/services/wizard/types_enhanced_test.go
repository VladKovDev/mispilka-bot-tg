package wizard

import (
	"testing"
	"time"
)

func TestWizardState_EditMode(t *testing.T) {
	state := &WizardState{
		UserID:      "123",
		WizardType:  WizardTypeCreateScenario,
		CurrentStep: StepConfirmGeneral,
		StartedAt:   time.Now(),
		Timeout:     30 * time.Minute,
		Data:        make(map[string]interface{}),
	}

	// Test edit mode
	state.SetEditMode(true, StepProductName)

	if !state.IsEditMode() {
		t.Error("Expected edit mode to be true")
	}

	targetStep := state.GetEditTargetStep()
	if targetStep != StepProductName {
		t.Errorf("Expected target step %s, got %s", StepProductName, targetStep)
	}
}

func TestWizardState_MessageTracking(t *testing.T) {
	state := &WizardState{
		UserID:      "123",
		WizardType:  WizardTypeCreateScenario,
		CurrentStep: StepMessageText,
		StartedAt:   time.Now(),
		Timeout:     30 * time.Minute,
		Data:        make(map[string]interface{}),
	}

	// Set current section and message index
	state.SetCurrentSection("messages")
	state.SetCurrentMessageIndex(0)
	state.IncrementMessagesCreated()

	if state.GetCurrentSection() != "messages" {
		t.Error("Expected section 'messages'")
	}

	if state.GetCurrentMessageIndex() != 0 {
		t.Error("Expected message index 0")
	}

	if state.GetMessagesCreated() != 1 {
		t.Error("Expected 1 message created")
	}
}
