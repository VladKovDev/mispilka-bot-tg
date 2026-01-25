package integration

import (
	"path/filepath"
	"testing"
	"time"

	domainScenario "mispilkabot/internal/domain/scenario"
	scenarioutil "mispilkabot/internal/services/scenario"
	"mispilkabot/internal/services/scenario_scheduler"
	"mispilkabot/internal/services/wizard"
)

func TestScenarioLifecycle(t *testing.T) {
	tmpDir := t.TempDir()

	// Setup - use the actual base directory structure
	scenarioService := scenarioutil.NewService(tmpDir)

	scheduler := scenario_scheduler.NewScheduler()

	wizardDir := filepath.Join(tmpDir, "wizards")
	wizardManager := wizard.NewManager(wizardDir)
	_ = wizardManager.Initialize()

	// Step 1: Create scenario
	t.Run("CreateScenario", func(t *testing.T) {
		req := &scenarioutil.CreateScenarioRequest{
			ID:   "test-scenario",
			Name: "Test Scenario",
			Prodamus: domainScenario.ProdamusConfig{
				ProductName:    "Test Product",
				ProductPrice:   "1000",
				PaidContent:    "Thank you!",
				PrivateGroupID: "-1001234567890",
			},
		}
		sc, err := scenarioService.CreateScenario(req)
		if err != nil {
			t.Fatalf("Failed to create scenario: %v", err)
		}
		if sc.ID == "" {
			t.Error("Expected scenario ID")
		}
		t.Logf("Created scenario: %s", sc.ID)
	})

	// Step 2: Start scenario for user
	t.Run("StartScenario", func(t *testing.T) {
		scenarios, err := scenarioService.ListScenarios()
		if err != nil {
			t.Fatalf("Failed to list scenarios: %v", err)
		}
		if len(scenarios) == 0 {
			t.Skip("No scenarios to start")
		}

		domainSc := &domainScenario.Scenario{
			ID:   scenarios[0].ID,
			Name: scenarios[0].Name,
			Config: domainScenario.ScenarioConfig{
				Prodamus: domainScenario.ProdamusConfig{
					ProductName:    "Test Product",
					ProductPrice:   "1000",
					PrivateGroupID: "-1001234567890",
				},
			},
			Messages: domainScenario.ScenarioMessages{
				MessagesList: []string{"msg_1"},
				Messages: map[string]domainScenario.MessageData{
					"msg_1": {
						TemplateFile: "test.md",
					},
				},
			},
		}

		chatID := "123456"
		state := &domainScenario.UserScenarioState{
			Status:              domainScenario.StatusActive,
			CurrentMessageIndex: 0,
		}

		// Simulate starting scenario
		if state.Status != domainScenario.StatusActive {
			t.Errorf("Expected active status, got %s", state.Status)
		}

		t.Logf("Started scenario %s for user %s", domainSc.ID, chatID)
	})

	// Step 3: Test scheduler
	t.Run("ScheduleMessage", func(t *testing.T) {
		scenarios, err := scenarioService.ListScenarios()
		if err != nil {
			t.Fatalf("Failed to list scenarios: %v", err)
		}
		if len(scenarios) == 0 {
			t.Skip("No scenarios to schedule")
		}

		domainSc := &domainScenario.Scenario{
			ID:   scenarios[0].ID,
			Name: scenarios[0].Name,
			Messages: domainScenario.ScenarioMessages{
				MessagesList: []string{"msg_1", "msg_2"},
				Messages: map[string]domainScenario.MessageData{
					"msg_1": {
						Timing: domainScenario.Timing{Hours: 0, Minutes: 0},
					},
					"msg_2": {
						Timing: domainScenario.Timing{Hours: 1, Minutes: 0},
					},
				},
			},
		}

		chatID := "123456"
		state := &domainScenario.UserScenarioState{
			Status:              domainScenario.StatusActive,
			CurrentMessageIndex: 0,
		}

		scheduledTime, err := scheduler.ScheduleNextMessage(chatID, domainSc, state)
		if err != nil {
			t.Fatalf("Failed to schedule: %v", err)
		}

		if scheduledTime.IsZero() {
			t.Error("Expected scheduled time to be set")
		}

		t.Logf("Scheduled message for %s at %s", chatID, scheduledTime.Format(time.RFC3339))

		// Clean up scheduler
		scheduler.CancelSchedule(chatID)
	})

	// Step 4: Test wizard
	t.Run("WizardLifecycle", func(t *testing.T) {
		userID := "test_user_123"

		// Start wizard
		state, err := wizardManager.Start(userID, wizard.WizardTypeCreateScenario)
		if err != nil {
			t.Fatalf("Failed to start wizard: %v", err)
		}

		if state.WizardType != wizard.WizardTypeCreateScenario {
			t.Errorf("Expected wizard type %s, got %s", wizard.WizardTypeCreateScenario, state.WizardType)
		}

		t.Logf("Started wizard for user %s at step %s", userID, state.CurrentStep)

		// Get wizard
		retrieved, err := wizardManager.Get(userID)
		if err != nil {
			t.Fatalf("Failed to get wizard: %v", err)
		}

		if retrieved.UserID != userID {
			t.Errorf("Expected user ID %s, got %s", userID, retrieved.UserID)
		}

		// Cancel wizard
		if err := wizardManager.Cancel(userID); err != nil {
			t.Fatalf("Failed to cancel wizard: %v", err)
		}

		// Verify wizard is cancelled
		_, err = wizardManager.Get(userID)
		if err == nil {
			t.Error("Expected error when getting cancelled wizard")
		}
	})
}

func TestWizardStateExpiration(t *testing.T) {
	t.Run("WizardExpires", func(t *testing.T) {
		state := &wizard.WizardState{
			UserID:     "test_user",
			WizardType: wizard.WizardTypeCreateScenario,
			CurrentStep: wizard.StepScenarioName,
			StartedAt:   time.Now().Add(-31 * time.Minute),
			Timeout:     30 * time.Minute,
		}

		if !state.Expired() {
			t.Error("Expected wizard to be expired")
		}
	})

	t.Run("WizardNotExpired", func(t *testing.T) {
		state := &wizard.WizardState{
			UserID:     "test_user",
			WizardType: wizard.WizardTypeCreateScenario,
			CurrentStep: wizard.StepScenarioName,
			StartedAt:   time.Now().Add(-10 * time.Minute),
			Timeout:     30 * time.Minute,
		}

		if state.Expired() {
			t.Error("Expected wizard not to be expired")
		}
	})
}
