package scenario

import (
	"os"
	"path/filepath"
	"testing"

	domainScenario "mispilkabot/internal/domain/scenario"
)

func TestService_CreateScenario(t *testing.T) {
	tmpDir := t.TempDir()
	scenarioDir := filepath.Join(tmpDir, "scenarios", "test-scenario")

	svc := NewService(tmpDir)

	req := &CreateScenarioRequest{
		ID:   "test-scenario",
		Name: "Test Scenario",
		Prodamus: domainScenario.ProdamusConfig{
			ProductName:    "Test Product",
			ProductPrice:   "100",
			PaidContent:    "premium",
			PrivateGroupID: "-1001234567890",
		},
	}

	scenario, err := svc.CreateScenario(req)
	if err != nil {
		t.Fatalf("failed to create scenario: %v", err)
	}

	if scenario.ID != req.ID {
		t.Errorf("expected ID %s, got %s", req.ID, scenario.ID)
	}
	if scenario.Name != req.Name {
		t.Errorf("expected Name %s, got %s", req.Name, scenario.Name)
	}
	if !scenario.IsActive {
		t.Error("expected IsActive to be true")
	}

	// Verify files were created
	if _, err := os.Stat(filepath.Join(scenarioDir, "config.json")); os.IsNotExist(err) {
		t.Error("config.json was not created")
	}
	if _, err := os.Stat(filepath.Join(scenarioDir, "messages.json")); os.IsNotExist(err) {
		t.Error("messages.json was not created")
	}
}

func TestService_GetScenario(t *testing.T) {
	tmpDir := t.TempDir()

	svc := NewService(tmpDir)

	// First create a scenario
	req := &CreateScenarioRequest{
		ID:   "test-scenario",
		Name: "Test Scenario",
		Prodamus: domainScenario.ProdamusConfig{
			ProductName:    "Test Product",
			ProductPrice:   "100",
			PrivateGroupID: "-1001234567890",
		},
	}

	_, err := svc.CreateScenario(req)
	if err != nil {
		t.Fatalf("failed to create scenario: %v", err)
	}

	// Now get it
	scenario, err := svc.GetScenario("test-scenario")
	if err != nil {
		t.Fatalf("failed to get scenario: %v", err)
	}

	if scenario.ID != "test-scenario" {
		t.Errorf("expected ID test-scenario, got %s", scenario.ID)
	}

	// Try to get non-existent scenario
	_, err = svc.GetScenario("non-existent")
	if err == nil {
		t.Error("expected error for non-existent scenario")
	}
}

func TestService_ListScenarios(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewService(tmpDir)

	// Create multiple scenarios
	ids := []string{"scenario-1", "scenario-2", "scenario-3"}
	for _, id := range ids {
		req := &CreateScenarioRequest{
			ID:   id,
			Name: "Scenario " + id,
			Prodamus: domainScenario.ProdamusConfig{
				ProductName:    "Product",
				ProductPrice:   "100",
				PrivateGroupID: "-1001234567890",
			},
		}
		if _, err := svc.CreateScenario(req); err != nil {
			t.Fatalf("failed to create scenario %s: %v", id, err)
		}
	}

	scenarios, err := svc.ListScenarios()
	if err != nil {
		t.Fatalf("failed to list scenarios: %v", err)
	}

	if len(scenarios) != 3 {
		t.Errorf("expected 3 scenarios, got %d", len(scenarios))
	}

	// Verify all IDs are present
	idMap := make(map[string]bool)
	for _, s := range scenarios {
		idMap[s.ID] = true
	}
	for _, id := range ids {
		if !idMap[id] {
			t.Errorf("scenario %s not found in list", id)
		}
	}
}

func TestService_UpdateScenario(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewService(tmpDir)

	// Create a scenario
	createReq := &CreateScenarioRequest{
		ID:   "test-scenario",
		Name: "Original Name",
		Prodamus: domainScenario.ProdamusConfig{
			ProductName:    "Original Product",
			ProductPrice:   "100",
			PrivateGroupID: "-1001234567890",
		},
	}

	_, err := svc.CreateScenario(createReq)
	if err != nil {
		t.Fatalf("failed to create scenario: %v", err)
	}

	// Update it
	updateReq := &UpdateScenarioRequest{
		ID:   "test-scenario",
		Name: "Updated Name",
		Prodamus: &domainScenario.ProdamusConfig{
			ProductName:  "Updated Product",
			ProductPrice: "200",
		},
	}

	scenario, err := svc.UpdateScenario(updateReq)
	if err != nil {
		t.Fatalf("failed to update scenario: %v", err)
	}

	if scenario.Name != "Updated Name" {
		t.Errorf("expected name 'Updated Name', got %s", scenario.Name)
	}
	if scenario.Config.Prodamus.ProductName != "Updated Product" {
		t.Errorf("expected product name 'Updated Product', got %s", scenario.Config.Prodamus.ProductName)
	}
}

func TestService_DeleteScenario(t *testing.T) {
	tmpDir := t.TempDir()
	scenarioDir := filepath.Join(tmpDir, "scenarios", "test-scenario")

	svc := NewService(tmpDir)

	// Create a scenario
	req := &CreateScenarioRequest{
		ID:   "test-scenario",
		Name: "Test Scenario",
		Prodamus: domainScenario.ProdamusConfig{
			ProductName:    "Product",
			ProductPrice:   "100",
			PrivateGroupID: "-1001234567890",
		},
	}

	_, err := svc.CreateScenario(req)
	if err != nil {
		t.Fatalf("failed to create scenario: %v", err)
	}

	// Verify it exists
	if _, err := svc.GetScenario("test-scenario"); err != nil {
		t.Fatal("scenario should exist before deletion")
	}

	// Delete it
	err = svc.DeleteScenario("test-scenario")
	if err != nil {
		t.Fatalf("failed to delete scenario: %v", err)
	}

	// Verify it's gone
	if _, err := svc.GetScenario("test-scenario"); err == nil {
		t.Error("scenario should not exist after deletion")
	}

	// Verify directory is removed
	if _, err := os.Stat(scenarioDir); !os.IsNotExist(err) {
		t.Error("scenario directory should be removed")
	}
}

func TestService_SetDefaultScenario(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewService(tmpDir)

	// Create scenarios
	for _, id := range []string{"scenario-1", "scenario-2"} {
		req := &CreateScenarioRequest{
			ID:   id,
			Name: "Scenario " + id,
			Prodamus: domainScenario.ProdamusConfig{
				ProductName:    "Product",
				ProductPrice:   "100",
				PrivateGroupID: "-1001234567890",
			},
		}
		if _, err := svc.CreateScenario(req); err != nil {
			t.Fatalf("failed to create scenario: %v", err)
		}
	}

	// Set default
	err := svc.SetDefaultScenario("scenario-2")
	if err != nil {
		t.Fatalf("failed to set default scenario: %v", err)
	}

	// Get default
	defaultID, err := svc.GetDefaultScenario()
	if err != nil {
		t.Fatalf("failed to get default scenario: %v", err)
	}

	if defaultID != "scenario-2" {
		t.Errorf("expected default scenario 'scenario-2', got %s", defaultID)
	}
}

func TestService_AddMessage(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewService(tmpDir)

	// Create a scenario
	req := &CreateScenarioRequest{
		ID:   "test-scenario",
		Name: "Test Scenario",
		Prodamus: domainScenario.ProdamusConfig{
			ProductName:    "Product",
			ProductPrice:   "100",
			PrivateGroupID: "-1001234567890",
		},
	}

	_, err := svc.CreateScenario(req)
	if err != nil {
		t.Fatalf("failed to create scenario: %v", err)
	}

	// Add a message
	msgReq := &AddMessageRequest{
		ScenarioID: "test-scenario",
		MessageID:  "msg_1",
		Timing: domainScenario.Timing{
			Hours:   1,
			Minutes: 30,
		},
		TemplateFile: "welcome.md",
	}

	err = svc.AddMessage(msgReq)
	if err != nil {
		t.Fatalf("failed to add message: %v", err)
	}

	// Verify it was added
	scenario, err := svc.GetScenario("test-scenario")
	if err != nil {
		t.Fatalf("failed to get scenario: %v", err)
	}

	if len(scenario.Messages.MessagesList) != 1 {
		t.Errorf("expected 1 message, got %d", len(scenario.Messages.MessagesList))
	}

	if scenario.Messages.MessagesList[0] != "msg_1" {
		t.Errorf("expected message ID 'msg_1', got %s", scenario.Messages.MessagesList[0])
	}
}

func TestService_UpdateMessage(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewService(tmpDir)

	// Create a scenario with a message
	createReq := &CreateScenarioRequest{
		ID:   "test-scenario",
		Name: "Test Scenario",
		Prodamus: domainScenario.ProdamusConfig{
			ProductName:    "Product",
			ProductPrice:   "100",
			PrivateGroupID: "-1001234567890",
		},
	}

	_, err := svc.CreateScenario(createReq)
	if err != nil {
		t.Fatalf("failed to create scenario: %v", err)
	}

	msgReq := &AddMessageRequest{
		ScenarioID: "test-scenario",
		MessageID:  "msg_1",
		Timing: domainScenario.Timing{
			Hours:   1,
			Minutes: 0,
		},
		TemplateFile: "original.md",
	}

	if err := svc.AddMessage(msgReq); err != nil {
		t.Fatalf("failed to add message: %v", err)
	}

	// Update the message
	updatedFile := "updated.md"
	updateReq := &UpdateMessageRequest{
		ScenarioID: "test-scenario",
		MessageID:  "msg_1",
		Timing: &domainScenario.Timing{
			Hours:   2,
			Minutes: 30,
		},
		TemplateFile: &updatedFile,
	}

	if err := svc.UpdateMessage(updateReq); err != nil {
		t.Fatalf("failed to update message: %v", err)
	}

	// Verify it was updated
	scenario, err := svc.GetScenario("test-scenario")
	if err != nil {
		t.Fatalf("failed to get scenario: %v", err)
	}

	msg := scenario.Messages.Messages["msg_1"]
	if msg.TemplateFile != "updated.md" {
		t.Errorf("expected template file 'updated.md', got %s", msg.TemplateFile)
	}
	if msg.Timing.Hours != 2 {
		t.Errorf("expected timing 2 hours, got %d", msg.Timing.Hours)
	}
}
