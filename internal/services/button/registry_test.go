package button

import (
	"path/filepath"
	"testing"

	domainButton "mispilkabot/internal/domain/button"
)

func TestButtonRegistry_LoadSave(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "buttons", "registry.json")

	reg := NewRegistry(registryPath)

	bs := &domainButton.ButtonSet{
		Rows: []domainButton.ButtonRow{
			{
				Buttons: []domainButton.Button{
					{
						Type: "url",
						Text: "Pay",
						URL:  "https://pay.example.com",
					},
				},
			},
		},
	}

	err := reg.Set("payment_button", bs)
	if err != nil {
		t.Fatalf("Failed to set button set: %v", err)
	}

	// Load into new registry
	reg2 := NewRegistry(registryPath)
	err = reg2.Load()
	if err != nil {
		t.Fatalf("Failed to load: %v", err)
	}

	// Verify
	loaded, ok := reg2.Get("payment_button")
	if !ok {
		t.Fatal("Button set not found")
	}
	if len(loaded.Rows) != 1 {
		t.Errorf("Expected 1 row, got %d", len(loaded.Rows))
	}
	if loaded.Rows[0].Buttons[0].Text != "Pay" {
		t.Errorf("Expected button text 'Pay', got '%s'", loaded.Rows[0].Buttons[0].Text)
	}
}

func TestButtonRegistry_Set(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")
	reg := NewRegistry(registryPath)

	bs := &domainButton.ButtonSet{
		Rows: []domainButton.ButtonRow{
			{
				Buttons: []domainButton.Button{
					{
						Type: "callback",
						Text: "Click",
						Callback: "test_callback",
					},
				},
			},
		},
	}

	err := reg.Set("test_button", bs)
	if err != nil {
		t.Fatalf("Failed to set button set: %v", err)
	}

	// Verify it was added
	_, ok := reg.Get("test_button")
	if !ok {
		t.Error("Button set not found after Set")
	}
}

func TestButtonRegistry_Get(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")
	reg := NewRegistry(registryPath)

	// Get non-existent button set
	_, ok := reg.Get("nonexistent")
	if ok {
		t.Error("Expected false for non-existent button set")
	}

	bs := &domainButton.ButtonSet{
		Rows: []domainButton.ButtonRow{
			{
				Buttons: []domainButton.Button{
					{
						Type: "url",
						Text: "Test",
						URL:  "https://test.example.com",
					},
				},
			},
		},
	}

	err := reg.Set("test_button", bs)
	if err != nil {
		t.Fatalf("Failed to set button set: %v", err)
	}

	// Get existing button set
	loaded, ok := reg.Get("test_button")
	if !ok {
		t.Fatal("Expected true for existing button set")
	}
	if len(loaded.Rows) != 1 {
		t.Errorf("Expected 1 row, got %d", len(loaded.Rows))
	}
	if loaded.Rows[0].Buttons[0].Text != "Test" {
		t.Errorf("Expected button text 'Test', got '%s'", loaded.Rows[0].Buttons[0].Text)
	}
}

func TestButtonRegistry_Delete(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")
	reg := NewRegistry(registryPath)

	bs := &domainButton.ButtonSet{
		Rows: []domainButton.ButtonRow{
			{
				Buttons: []domainButton.Button{
					{
						Type: "callback",
						Text: "Delete Me",
						Callback: "delete_callback",
					},
				},
			},
		},
	}

	err := reg.Set("test_button", bs)
	if err != nil {
		t.Fatalf("Failed to set button set: %v", err)
	}

	// Verify it exists
	_, ok := reg.Get("test_button")
	if !ok {
		t.Fatal("Button set should exist before delete")
	}

	// Delete button set
	err = reg.Delete("test_button")
	if err != nil {
		t.Fatalf("Failed to delete button set: %v", err)
	}

	// Verify it's gone
	_, ok = reg.Get("test_button")
	if ok {
		t.Error("Button set still exists after Delete")
	}
}

func TestButtonRegistry_Load_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	reg := NewRegistry(registryPath)

	// Load should succeed even if file doesn't exist
	err := reg.Load()
	if err != nil {
		t.Fatalf("Failed to load empty registry: %v", err)
	}

	// Get should return false for empty registry
	_, ok := reg.Get("nonexistent")
	if ok {
		t.Error("Expected false for non-existent button set in empty registry")
	}
}
