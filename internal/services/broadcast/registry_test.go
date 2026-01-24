package broadcast

import (
	"path/filepath"
	"testing"
	"time"

	"mispilkabot/internal/domain/broadcast"
)

func TestBroadcastRegistry_LoadSave(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "broadcasts", "registry.json")

	reg := NewRegistry(registryPath)

	bc := &broadcast.Broadcast{
		ID:           "test_bc",
		Name:         "Test Broadcast",
		TemplateFile: "test.md",
		Targeting: &broadcast.Targeting{
			Conditions: []string{broadcast.ConditionNoActiveScenario},
		},
		CreatedAt: time.Now(),
	}

	err := reg.Add(bc)
	if err != nil {
		t.Fatalf("Failed to add broadcast: %v", err)
	}

	// Load into new registry
	reg2 := NewRegistry(registryPath)
	err = reg2.Load()
	if err != nil {
		t.Fatalf("Failed to load: %v", err)
	}

	// Verify
	loaded, ok := reg2.Get("test_bc")
	if !ok {
		t.Fatal("Broadcast not found")
	}
	if loaded.Name != "Test Broadcast" {
		t.Errorf("Expected name 'Test Broadcast', got '%s'", loaded.Name)
	}
}
