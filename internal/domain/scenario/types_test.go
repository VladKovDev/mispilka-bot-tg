package scenario

import (
	"testing"
)

func TestScenario_Validate(t *testing.T) {
	scenario := &Scenario{
		ID:       "test-scenario",
		Name:     "Test Scenario",
		IsActive: true,
		Config: ScenarioConfig{
			Prodamus: ProdamusConfig{
				ProductName:    "Test Product",
				ProductPrice:   "1000",
				PaidContent:    "Thank you!",
				PrivateGroupID: "-1001234567890",
			},
		},
	}

	err := scenario.Validate()
	if err != nil {
		t.Fatalf("Expected valid scenario, got error: %v", err)
	}
}

func TestScenarioStatus_String(t *testing.T) {
	tests := []struct {
		status   ScenarioStatus
		expected string
	}{
		{StatusNotStarted, "not_started"},
		{StatusActive, "active"},
		{StatusCompleted, "completed"},
		{StatusStopped, "stopped"},
	}

	for _, tt := range tests {
		if tt.status.String() != tt.expected {
			t.Errorf("Expected %s, got %s", tt.expected, tt.status.String())
		}
	}
}
