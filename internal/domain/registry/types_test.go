package registry

import (
	"testing"

	"mispilkabot/internal/domain/scenario"
)

func TestScenarioRegistry_GetDefault(t *testing.T) {
	reg := &ScenarioRegistry{
		DefaultScenarioID: "default",
		Scenarios: map[string]*scenario.Scenario{
			"default": {
				ID:   "default",
				Name: "Default",
			},
		},
	}

	sc, err := reg.GetDefault()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if sc.ID != "default" {
		t.Errorf("Expected default scenario, got: %s", sc.ID)
	}
}

func TestScenarioRegistry_SetDefault(t *testing.T) {
	reg := &ScenarioRegistry{
		Scenarios: map[string]*scenario.Scenario{
			"default": {ID: "default", Name: "Default"},
			"premium": {ID: "premium", Name: "Premium"},
		},
	}

	err := reg.SetDefault("premium")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if reg.DefaultScenarioID != "premium" {
		t.Errorf("Expected premium as default, got: %s", reg.DefaultScenarioID)
	}
}
