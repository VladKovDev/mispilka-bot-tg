package scenario

import (
	"path/filepath"
	"testing"
)

func TestConfig_LoadSave(t *testing.T) {
	tmpDir := t.TempDir()
	scenarioDir := filepath.Join(tmpDir, "scenarios", "test")
	configPath := filepath.Join(scenarioDir, "config.json")

	// Create config
	cfg := NewConfig(configPath)
	cfg.ID = "test"
	cfg.Name = "Test Scenario"
	cfg.Prodamus.ProductName = "Test Product"
	cfg.Prodamus.ProductPrice = "1000"
	cfg.Prodamus.PaidContent = "Thank you!"
	cfg.Prodamus.PrivateGroupID = "-1001234567890"

	// Save
	err := cfg.Save()
	if err != nil {
		t.Fatalf("Failed to save: %v", err)
	}

	// Load into new config
	cfg2 := NewConfig(configPath)
	err = cfg2.Load()
	if err != nil {
		t.Fatalf("Failed to load: %v", err)
	}

	// Verify
	if cfg2.ID != "test" {
		t.Errorf("Expected ID 'test', got '%s'", cfg2.ID)
	}
	if cfg2.Name != "Test Scenario" {
		t.Errorf("Expected name 'Test Scenario', got '%s'", cfg2.Name)
	}
}
