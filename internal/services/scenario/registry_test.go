package scenario

import (
	"path/filepath"
	"testing"
	"time"

	"mispilkabot/internal/domain/scenario"
)

func TestScenarioRegistry_LoadSave(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	// Create initial registry
	reg := NewRegistry(registryPath)
	reg.Scenarios = map[string]*scenario.Scenario{
		"test": {
			ID:        "test",
			Name:      "Test",
			CreatedAt: time.Now(),
			IsActive:  true,
			Config: scenario.ScenarioConfig{
				Prodamus: scenario.ProdamusConfig{
					ProductName:    "Test Product",
					ProductPrice:   "1000",
					PaidContent:    "Thank you!",
					PrivateGroupID: "-1001234567890",
				},
			},
		},
	}
	reg.DefaultScenarioID = "test"

	// Save
	err := reg.Save()
	if err != nil {
		t.Fatalf("Failed to save: %v", err)
	}

	// Load into new registry
	reg2 := NewRegistry(registryPath)
	err = reg2.Load()
	if err != nil {
		t.Fatalf("Failed to load: %v", err)
	}

	// Verify
	if reg2.DefaultScenarioID != "test" {
		t.Errorf("Expected default scenario 'test', got '%s'", reg2.DefaultScenarioID)
	}
	sc, ok := reg2.Scenarios["test"]
	if !ok {
		t.Fatal("Scenario 'test' not found")
	}
	if sc.Name != "Test" {
		t.Errorf("Expected name 'Test', got '%s'", sc.Name)
	}
}
