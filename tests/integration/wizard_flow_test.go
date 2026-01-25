//go:build integration

package integration

import (
	"path/filepath"
	"testing"

	domainScenario "mispilkabot/internal/domain/scenario"
	"mispilkabot/internal/services/scenario"
	"mispilkabot/internal/services/wizard"
)

// TestWizardScenarioCreationFlow tests the complete wizard flow
func TestWizardScenarioCreationFlow(t *testing.T) {
	// Setup
	tmpDir := t.TempDir()
	wizardsDir := filepath.Join(tmpDir, "wizards")

	wm := wizard.NewManager(wizardsDir)
	if err := wm.Initialize(); err != nil {
		t.Fatalf("Failed to initialize wizard manager: %v", err)
	}

	ss := scenario.NewService(tmpDir)

	userID := "test_user_123"

	// Start wizard
	state, err := wm.Start(userID, wizard.WizardTypeCreateScenario)
	if err != nil {
		t.Fatalf("Failed to start wizard: %v", err)
	}

	if state.CurrentStep != wizard.StepScenarioName {
		t.Errorf("Expected step %s, got %s", wizard.StepScenarioName, state.CurrentStep)
	}

	// Simulate entering data
	testData := map[wizard.WizardStep]string{
		wizard.StepScenarioName:   "Test Scenario",
		wizard.StepProductName:    "Test Product",
		wizard.StepProductPrice:   "500",
		wizard.StepPaidContent:    "Test content description",
		wizard.StepPrivateGroupID: "-1001234567890",
	}

	// Go through each step
	for step, value := range testData {
		state.Set(string(step), value)

		// Advance to next step
		var nextStep wizard.WizardStep
		switch step {
		case wizard.StepScenarioName:
			nextStep = wizard.StepProductName
		case wizard.StepProductName:
			nextStep = wizard.StepProductPrice
		case wizard.StepProductPrice:
			nextStep = wizard.StepPaidContent
		case wizard.StepPaidContent:
			nextStep = wizard.StepPrivateGroupID
		case wizard.StepPrivateGroupID:
			nextStep = wizard.StepConfirmGeneral
		}

		if nextStep != "" {
			if err := wm.Advance(userID, nextStep); err != nil {
				t.Fatalf("Failed to advance to %s: %v", nextStep, err)
			}

			// Reload state
			state, err = wm.Get(userID)
			if err != nil {
				t.Fatalf("Failed to get state: %v", err)
			}

			if state.CurrentStep != nextStep {
				t.Errorf("Expected step %s, got %s", nextStep, state.CurrentStep)
			}
		}
	}

	// Verify confirmation state
	if state.CurrentStep != wizard.StepConfirmGeneral {
		t.Errorf("Expected step %s, got %s", wizard.StepConfirmGeneral, state.CurrentStep)
	}

	// Create scenario
	req := &scenario.CreateScenarioRequest{
		ID:   "test_scenario",
		Name: state.GetString(string(wizard.StepScenarioName)),
		Prodamus: domainScenario.ProdamusConfig{
			ProductName:    state.GetString(string(wizard.StepProductName)),
			ProductPrice:   state.GetString(string(wizard.StepProductPrice)),
			PaidContent:    state.GetString(string(wizard.StepPaidContent)),
			PrivateGroupID: state.GetString(string(wizard.StepPrivateGroupID)),
		},
	}

	sc, err := ss.CreateScenario(req)
	if err != nil {
		t.Fatalf("Failed to create scenario: %v", err)
	}

	if sc.ID != req.ID {
		t.Errorf("Expected ID %s, got %s", req.ID, sc.ID)
	}

	if sc.Name != req.Name {
		t.Errorf("Expected name %s, got %s", req.Name, sc.Name)
	}

	// Cleanup
	_ = wm.Cancel(userID)
}

// TestWizardTimeout tests wizard expiration
func TestWizardTimeout(t *testing.T) {
	tmpDir := t.TempDir()
	wizardsDir := filepath.Join(tmpDir, "wizards")

	wm := wizard.NewManager(wizardsDir)
	if err := wm.Initialize(); err != nil {
		t.Fatalf("Failed to initialize wizard manager: %v", err)
	}

	userID := "test_user_timeout"

	// Start wizard
	state, err := wm.Start(userID, wizard.WizardTypeCreateScenario)
	if err != nil {
		t.Fatalf("Failed to start wizard: %v", err)
	}

	// Verify wizard is active (not expired immediately)
	if state.Expired() {
		t.Error("Newly created wizard should not be expired")
	}

	// Cancel the wizard to clean up
	_ = wm.Cancel(userID)

	// After cancellation, trying to get should return not found
	_, err = wm.Get(userID)
	if err == nil {
		t.Error("Expected error after cancellation, got nil")
	}
	if err != wizard.ErrWizardNotFound {
		t.Errorf("Expected ErrWizardNotFound, got %v", err)
	}
}

// TestWizardEditMode tests edit mode functionality
func TestWizardEditMode(t *testing.T) {
	tmpDir := t.TempDir()
	wizardsDir := filepath.Join(tmpDir, "wizards")

	wm := wizard.NewManager(wizardsDir)
	if err := wm.Initialize(); err != nil {
		t.Fatalf("Failed to initialize wizard manager: %v", err)
	}

	userID := "test_user_edit"

	// Start wizard and advance to confirmation
	state, err := wm.Start(userID, wizard.WizardTypeCreateScenario)
	if err != nil {
		t.Fatalf("Failed to start wizard: %v", err)
	}

	// Enter some data
	state.Set(string(wizard.StepScenarioName), "Original Name")
	state.Set(string(wizard.StepProductName), "Original Product")
	state.Set(string(wizard.StepProductPrice), "500")
	state.Set(string(wizard.StepPaidContent), "Original Content")
	state.Set(string(wizard.StepPrivateGroupID), "-1001234567890")

	// Advance to confirmation
	_ = wm.Advance(userID, wizard.StepConfirmGeneral)
	state, _ = wm.Get(userID)

	// Enable edit mode
	state.SetEditMode(true, wizard.StepProductName)
	_ = wm.Update(userID, state)

	// Verify edit mode is enabled
	if !state.IsEditMode() {
		t.Error("Expected edit mode to be enabled")
	}

	targetStep := state.GetEditTargetStep()
	if targetStep != wizard.StepProductName {
		t.Errorf("Expected target step %s, got %s", wizard.StepProductName, targetStep)
	}

	// Cleanup
	_ = wm.Cancel(userID)
}

// TestWizardMessageTracking tests message index tracking
func TestWizardMessageTracking(t *testing.T) {
	tmpDir := t.TempDir()
	wizardsDir := filepath.Join(tmpDir, "wizards")

	wm := wizard.NewManager(wizardsDir)
	if err := wm.Initialize(); err != nil {
		t.Fatalf("Failed to initialize wizard manager: %v", err)
	}

	userID := "test_user_messages"

	state, err := wm.Start(userID, wizard.WizardTypeCreateScenario)
	if err != nil {
		t.Fatalf("Failed to start wizard: %v", err)
	}

	// Set current section and message index
	state.SetCurrentSection("messages")
	state.SetCurrentMessageIndex(0)
	state.IncrementMessagesCreated()

	if state.GetCurrentSection() != "messages" {
		t.Errorf("Expected section 'messages', got '%s'", state.GetCurrentSection())
	}

	if state.GetCurrentMessageIndex() != 0 {
		t.Errorf("Expected message index 0, got %d", state.GetCurrentMessageIndex())
	}

	if state.GetMessagesCreated() != 1 {
		t.Errorf("Expected 1 message created, got %d", state.GetMessagesCreated())
	}

	// Cleanup
	_ = wm.Cancel(userID)
}

// TestWizardLastConfirmedStep tests tracking of last confirmed step
func TestWizardLastConfirmedStep(t *testing.T) {
	tmpDir := t.TempDir()
	wizardsDir := filepath.Join(tmpDir, "wizards")

	wm := wizard.NewManager(wizardsDir)
	if err := wm.Initialize(); err != nil {
		t.Fatalf("Failed to initialize wizard manager: %v", err)
	}

	userID := "test_user_confirmed"

	state, err := wm.Start(userID, wizard.WizardTypeCreateScenario)
	if err != nil {
		t.Fatalf("Failed to start wizard: %v", err)
	}

	// Set last confirmed step
	state.SetLastConfirmedStep(wizard.StepProductName)

	if state.GetLastConfirmedStep() != wizard.StepProductName {
		t.Errorf("Expected last confirmed step %s, got %s", wizard.StepProductName, state.GetLastConfirmedStep())
	}

	// Cleanup
	_ = wm.Cancel(userID)
}
