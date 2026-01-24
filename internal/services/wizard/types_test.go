package wizard

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWizardState_Expired(t *testing.T) {
	state := &WizardState{
		StartedAt: time.Now().Add(-31 * time.Minute),
		Timeout:   30 * time.Minute,
	}

	if !state.Expired() {
		t.Error("Expected wizard to be expired")
	}
}

func TestWizardState_NotExpired(t *testing.T) {
	state := &WizardState{
		StartedAt: time.Now().Add(-10 * time.Minute),
		Timeout:   30 * time.Minute,
	}

	if state.Expired() {
		t.Error("Expected wizard not to be expired")
	}
}

func TestManager_Start(t *testing.T) {
	// Create temp directory for wizards
	tmpDir := t.TempDir()
	manager := NewManager(tmpDir)

	// Initialize manager
	if err := manager.Initialize(); err != nil {
		t.Fatalf("Failed to initialize manager: %v", err)
	}

	// Start a wizard
	state, err := manager.Start("user123", WizardTypeCreateScenario)
	if err != nil {
		t.Fatalf("Failed to start wizard: %v", err)
	}

	// Verify state
	if state.UserID != "user123" {
		t.Errorf("Expected UserID user123, got %s", state.UserID)
	}
	if state.WizardType != WizardTypeCreateScenario {
		t.Errorf("Expected WizardType %s, got %s", WizardTypeCreateScenario, state.WizardType)
	}
	if state.CurrentStep != StepScenarioName {
		t.Errorf("Expected CurrentStep %s, got %s", StepScenarioName, state.CurrentStep)
	}
	if state.Timeout != 30*time.Minute {
		t.Errorf("Expected Timeout %v, got %v", 30*time.Minute, state.Timeout)
	}

	// Verify file was created
	filePath := filepath.Join(tmpDir, "user123.json")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Error("Expected wizard file to be created")
	}
}

func TestManager_Get(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManager(tmpDir)

	if err := manager.Initialize(); err != nil {
		t.Fatalf("Failed to initialize manager: %v", err)
	}

	// Start a wizard
	_, err := manager.Start("user123", WizardTypeCreateScenario)
	if err != nil {
		t.Fatalf("Failed to start wizard: %v", err)
	}

	// Get the wizard
	state, err := manager.Get("user123")
	if err != nil {
		t.Fatalf("Failed to get wizard: %v", err)
	}

	if state.UserID != "user123" {
		t.Errorf("Expected UserID user123, got %s", state.UserID)
	}

	// Try to get non-existent wizard
	_, err = manager.Get("nonexistent")
	if err != ErrWizardNotFound {
		t.Errorf("Expected ErrWizardNotFound, got %v", err)
	}
}

func TestManager_Update(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManager(tmpDir)

	if err := manager.Initialize(); err != nil {
		t.Fatalf("Failed to initialize manager: %v", err)
	}

	// Start a wizard
	_, err := manager.Start("user123", WizardTypeCreateScenario)
	if err != nil {
		t.Fatalf("Failed to start wizard: %v", err)
	}

	// Get and update the wizard
	state, _ := manager.Get("user123")
	state.Set("test_key", "test_value")

	// Update the wizard
	if err := manager.Update("user123", state); err != nil {
		t.Fatalf("Failed to update wizard: %v", err)
	}

	// Verify the update
	updatedState, _ := manager.Get("user123")
	if updatedState.GetString("test_key") != "test_value" {
		t.Error("Expected test_key to be updated")
	}

	// Try to update non-existent wizard
	nonExistentState := &WizardState{UserID: "nonexistent"}
	err = manager.Update("nonexistent", nonExistentState)
	if err != ErrWizardNotFound {
		t.Errorf("Expected ErrWizardNotFound, got %v", err)
	}
}

func TestManager_Advance(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManager(tmpDir)

	if err := manager.Initialize(); err != nil {
		t.Fatalf("Failed to initialize manager: %v", err)
	}

	// Start a wizard
	_, err := manager.Start("user123", WizardTypeCreateScenario)
	if err != nil {
		t.Fatalf("Failed to start wizard: %v", err)
	}

	// Advance to next step
	if err := manager.Advance("user123", StepProductName); err != nil {
		t.Fatalf("Failed to advance wizard: %v", err)
	}

	// Verify the advance
	state, _ := manager.Get("user123")
	if state.CurrentStep != StepProductName {
		t.Errorf("Expected CurrentStep %s, got %s", StepProductName, state.CurrentStep)
	}

	// Try to advance non-existent wizard
	err = manager.Advance("nonexistent", StepProductName)
	if err != ErrWizardNotFound {
		t.Errorf("Expected ErrWizardNotFound, got %v", err)
	}
}

func TestManager_Cancel(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManager(tmpDir)

	if err := manager.Initialize(); err != nil {
		t.Fatalf("Failed to initialize manager: %v", err)
	}

	// Start a wizard
	_, err := manager.Start("user123", WizardTypeCreateScenario)
	if err != nil {
		t.Fatalf("Failed to start wizard: %v", err)
	}

	// Cancel the wizard
	if err := manager.Cancel("user123"); err != nil {
		t.Fatalf("Failed to cancel wizard: %v", err)
	}

	// Verify the wizard is gone
	_, err = manager.Get("user123")
	if err != ErrWizardNotFound {
		t.Errorf("Expected ErrWizardNotFound after cancel, got %v", err)
	}

	// Verify file was deleted
	filePath := filepath.Join(tmpDir, "user123.json")
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Error("Expected wizard file to be deleted")
	}

	// Try to cancel non-existent wizard
	err = manager.Cancel("nonexistent")
	if err != ErrWizardNotFound {
		t.Errorf("Expected ErrWizardNotFound, got %v", err)
	}
}

func TestManager_Initialize_CleansUpExpired(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManager(tmpDir)

	// Create an expired wizard file manually
	expiredState := &WizardState{
		UserID:      "expired_user",
		WizardType:  WizardTypeCreateScenario,
		CurrentStep: StepScenarioName,
		StartedAt:   time.Now().Add(-31 * time.Minute),
		Timeout:     30 * time.Minute,
		Data:        make(map[string]interface{}),
	}

	filePath := filepath.Join(tmpDir, "expired_user.json")
	data, _ := json.Marshal(expiredState)
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		t.Fatalf("Failed to create expired wizard file: %v", err)
	}

	// Initialize manager (should clean up expired wizards)
	if err := manager.Initialize(); err != nil {
		t.Fatalf("Failed to initialize manager: %v", err)
	}

	// Verify expired wizard was not loaded
	_, err := manager.Get("expired_user")
	if err != ErrWizardNotFound {
		t.Errorf("Expected expired wizard to be cleaned up, got %v", err)
	}

	// Verify file was deleted
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Error("Expected expired wizard file to be deleted")
	}
}

func TestManager_Start_ReplacesExisting(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManager(tmpDir)

	if err := manager.Initialize(); err != nil {
		t.Fatalf("Failed to initialize manager: %v", err)
	}

	// Start first wizard
	_, err := manager.Start("user123", WizardTypeCreateScenario)
	if err != nil {
		t.Fatalf("Failed to start first wizard: %v", err)
	}

	// Start second wizard (should replace first)
	state2, err := manager.Start("user123", WizardTypeCreateBroadcast)
	if err != nil {
		t.Fatalf("Failed to start second wizard: %v", err)
	}

	if state2.WizardType != WizardTypeCreateBroadcast {
		t.Errorf("Expected WizardType %s, got %s", WizardTypeCreateBroadcast, state2.WizardType)
	}

	// Verify only one file exists
	entries, _ := os.ReadDir(tmpDir)
	if len(entries) != 1 {
		t.Errorf("Expected 1 wizard file, got %d", len(entries))
	}
}

func TestWizardState_DataAccessors(t *testing.T) {
	state := &WizardState{
		Data: make(map[string]interface{}),
	}

	// Test Set and Get
	state.Set("string_key", "string_value")
	state.Set("int_key", 42)
	state.Set("float_key", 3.14)
	state.Set("slice_key", []string{"a", "b", "c"})

	// Test GetString
	if state.GetString("string_key") != "string_value" {
		t.Error("Expected GetString to return string_value")
	}
	if state.GetString("nonexistent") != "" {
		t.Error("Expected GetString to return empty string for nonexistent key")
	}
	if state.GetString("int_key") != "" {
		t.Error("Expected GetString to return empty string for int value")
	}

	// Test GetInt
	if state.GetInt("int_key") != 42 {
		t.Errorf("Expected GetInt to return 42, got %d", state.GetInt("int_key"))
	}
	if state.GetInt("float_key") != 3 {
		t.Errorf("Expected GetInt to return 3 for float64, got %d", state.GetInt("float_key"))
	}
	if state.GetInt("nonexistent") != 0 {
		t.Error("Expected GetInt to return 0 for nonexistent key")
	}

	// Test GetStringSlice
	slice := state.GetStringSlice("slice_key")
	if slice == nil || len(slice) != 3 || slice[0] != "a" {
		t.Error("Expected GetStringSlice to return correct slice")
	}
	if state.GetStringSlice("nonexistent") != nil {
		t.Error("Expected GetStringSlice to return nil for nonexistent key")
	}

	// Test Get
	val, ok := state.Get("string_key")
	if !ok || val != "string_value" {
		t.Error("Expected Get to return string_value")
	}
	_, ok = state.Get("nonexistent")
	if ok {
		t.Error("Expected Get to return false for nonexistent key")
	}
}

func TestWizardState_Clone(t *testing.T) {
	original := &WizardState{
		UserID:      "user123",
		WizardType:  WizardTypeCreateScenario,
		CurrentStep: StepScenarioName,
		StartedAt:   time.Now(),
		Timeout:     30 * time.Minute,
		Data:        map[string]interface{}{"key": "value"},
	}

	clone := original.Clone()

	// Verify clone has same values
	if clone.UserID != original.UserID {
		t.Error("Clone UserID mismatch")
	}
	if clone.WizardType != original.WizardType {
		t.Error("Clone WizardType mismatch")
	}
	if clone.CurrentStep != original.CurrentStep {
		t.Error("Clone CurrentStep mismatch")
	}
	if clone.GetString("key") != "value" {
		t.Error("Clone data mismatch")
	}

	// Verify clone is independent
	clone.Set("new_key", "new_value")
	if original.GetString("new_key") != "" {
		t.Error("Clone should not affect original")
	}
}

func TestWizardState_ResetTimeout(t *testing.T) {
	state := &WizardState{
		StartedAt: time.Now().Add(-10 * time.Minute),
		Timeout:   30 * time.Minute,
	}

	// Should not be expired
	if state.Expired() {
		t.Error("Should not be expired yet")
	}

	// Reset timeout (now started at current time)
	state.ResetTimeout()

	// Still should not be expired
	if state.Expired() {
		t.Error("Should not be expired after reset")
	}
}
