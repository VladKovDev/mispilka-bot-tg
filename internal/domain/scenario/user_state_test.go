package scenario

import (
	"testing"
)

func TestUserScenarioState_IsCompleted(t *testing.T) {
	state := &UserScenarioState{
		Status: StatusCompleted,
	}

	if !state.IsCompleted() {
		t.Error("Expected state to be completed")
	}
}

func TestUserScenarioState_IsActive(t *testing.T) {
	state := &UserScenarioState{
		Status: StatusActive,
	}

	if !state.IsActive() {
		t.Error("Expected state to be active")
	}
}
