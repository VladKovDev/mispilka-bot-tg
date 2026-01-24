package scenario

import (
	"testing"
	"time"

	domainScenario "mispilkabot/internal/domain/scenario"
	"mispilkabot/internal/services"
)

func TestUserService_StartScenario(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewUserService(tmpDir)

	// Create a user first
	chatID := "123456"
	if err := svc.CreateUser(chatID, "testuser"); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// Create a scenario first
	scenarioSvc := NewService(tmpDir)
	scenarioReq := &CreateScenarioRequest{
		ID:   "test-scenario",
		Name: "Test Scenario",
		Prodamus: domainScenario.ProdamusConfig{
			ProductName:    "Product",
			ProductPrice:   "100",
			PrivateGroupID: "-1001234567890",
		},
	}
	if _, err := scenarioSvc.CreateScenario(scenarioReq); err != nil {
		t.Fatalf("failed to create scenario: %v", err)
	}

	// Start scenario for user
	err := svc.StartScenario(chatID, "test-scenario")
	if err != nil {
		t.Fatalf("failed to start scenario: %v", err)
	}

	// Verify user state
	state, err := svc.GetUserScenario(chatID, "test-scenario")
	if err != nil {
		t.Fatalf("failed to get user scenario: %v", err)
	}

	if state.Status != services.StatusActive {
		t.Errorf("expected status %s, got %s", services.StatusActive, state.Status)
	}
	if state.CurrentMessageIndex != 0 {
		t.Errorf("expected message index 0, got %d", state.CurrentMessageIndex)
	}
}

func TestUserService_GetActiveScenario(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewUserService(tmpDir)

	// Create a user first
	chatID := "123456"
	if err := svc.CreateUser(chatID, "testuser"); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// Create a scenario first
	scenarioSvc := NewService(tmpDir)
	scenarioReq := &CreateScenarioRequest{
		ID:   "test-scenario",
		Name: "Test Scenario",
		Prodamus: domainScenario.ProdamusConfig{
			ProductName:    "Product",
			ProductPrice:   "100",
			PrivateGroupID: "-1001234567890",
		},
	}
	if _, err := scenarioSvc.CreateScenario(scenarioReq); err != nil {
		t.Fatalf("failed to create scenario: %v", err)
	}

	// No active scenario initially
	scenarioID, state, err := svc.GetActiveScenario(chatID)
	if err != nil {
		t.Fatalf("failed to get active scenario: %v", err)
	}
	if scenarioID != "" {
		t.Errorf("expected no active scenario, got %s", scenarioID)
	}
	if state != nil {
		t.Error("expected nil state")
	}

	// Start scenario
	if err := svc.StartScenario(chatID, "test-scenario"); err != nil {
		t.Fatalf("failed to start scenario: %v", err)
	}

	// Now there should be an active scenario
	scenarioID, state, err = svc.GetActiveScenario(chatID)
	if err != nil {
		t.Fatalf("failed to get active scenario: %v", err)
	}
	if scenarioID != "test-scenario" {
		t.Errorf("expected active scenario 'test-scenario', got %s", scenarioID)
	}
	if state.Status != services.StatusActive {
		t.Errorf("expected status %s", services.StatusActive)
	}
}

func TestUserService_AdvanceToNextMessage(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewUserService(tmpDir)

	// Create a user first
	chatID := "123456"
	if err := svc.CreateUser(chatID, "testuser"); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// Create a scenario with messages
	scenarioSvc := NewService(tmpDir)
	scenarioReq := &CreateScenarioRequest{
		ID:   "test-scenario",
		Name: "Test Scenario",
		Prodamus: domainScenario.ProdamusConfig{
			ProductName:    "Product",
			ProductPrice:   "100",
			PrivateGroupID: "-1001234567890",
		},
	}
	if _, err := scenarioSvc.CreateScenario(scenarioReq); err != nil {
		t.Fatalf("failed to create scenario: %v", err)
	}

	// Add messages
	for i := 1; i <= 3; i++ {
		msgReq := &AddMessageRequest{
			ScenarioID: "test-scenario",
			MessageID:  "msg_" + string(rune('0'+i)),
			Timing: domainScenario.Timing{
				Hours: i,
			},
			TemplateFile: "msg.md",
		}
		if err := scenarioSvc.AddMessage(msgReq); err != nil {
			t.Fatalf("failed to add message: %v", err)
		}
	}

	if err := svc.StartScenario(chatID, "test-scenario"); err != nil {
		t.Fatalf("failed to start scenario: %v", err)
	}

	// Get first message (needed to establish current position)
	firstID, err := svc.scenarioSvc.GetFirstMessageID("test-scenario")
	if err != nil {
		t.Fatalf("failed to get first message: %v", err)
	}
	if firstID != "msg_1" {
		t.Fatalf("expected first message 'msg_1', got %s", firstID)
	}

	// Simulate sending first message by updating state to point to msg_1
	users, err := svc.loadUsers()
	if err != nil {
		t.Fatalf("failed to load users: %v", err)
	}
	user := users[chatID]
	now := time.Now()
	user.Scenarios["test-scenario"].CurrentMessageIndex = 1
	user.Scenarios["test-scenario"].LastSentAt = &now
	users[chatID] = user
	if err := svc.saveUsers(users); err != nil {
		t.Fatalf("failed to save users: %v", err)
	}

	// Now advance to next message
	nextID, err := svc.AdvanceToNextMessage(chatID, "test-scenario")
	if err != nil {
		t.Fatalf("failed to advance to next message: %v", err)
	}
	if nextID != "msg_2" {
		t.Errorf("expected next message 'msg_2', got %s", nextID)
	}

	// Verify state was updated
	state, _ := svc.GetUserScenario(chatID, "test-scenario")
	if state.CurrentMessageIndex != 2 {
		t.Errorf("expected message index 2, got %d", state.CurrentMessageIndex)
	}
}

func TestUserService_CompleteScenario(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewUserService(tmpDir)

	// Create a user first
	chatID := "123456"
	if err := svc.CreateUser(chatID, "testuser"); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// Create a scenario first
	scenarioSvc := NewService(tmpDir)
	scenarioReq := &CreateScenarioRequest{
		ID:   "test-scenario",
		Name: "Test Scenario",
		Prodamus: domainScenario.ProdamusConfig{
			ProductName:    "Product",
			ProductPrice:   "100",
			PrivateGroupID: "-1001234567890",
		},
	}
	if _, err := scenarioSvc.CreateScenario(scenarioReq); err != nil {
		t.Fatalf("failed to create scenario: %v", err)
	}

	if err := svc.StartScenario(chatID, "test-scenario"); err != nil {
		t.Fatalf("failed to start scenario: %v", err)
	}

	// Complete scenario
	if err := svc.CompleteScenario(chatID, "test-scenario"); err != nil {
		t.Fatalf("failed to complete scenario: %v", err)
	}

	// Verify state
	state, _ := svc.GetUserScenario(chatID, "test-scenario")
	if state.Status != services.StatusCompleted {
		t.Errorf("expected status %s, got %s", services.StatusCompleted, state.Status)
	}
	if state.CompletedAt == nil {
		t.Error("expected CompletedAt to be set")
	}
}

func TestUserService_SetPaymentInfo(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewUserService(tmpDir)

	// Create a user first
	chatID := "123456"
	if err := svc.CreateUser(chatID, "testuser"); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// Create a scenario first
	scenarioSvc := NewService(tmpDir)
	scenarioReq := &CreateScenarioRequest{
		ID:   "test-scenario",
		Name: "Test Scenario",
		Prodamus: domainScenario.ProdamusConfig{
			ProductName:    "Product",
			ProductPrice:   "100",
			PrivateGroupID: "-1001234567890",
		},
	}
	if _, err := scenarioSvc.CreateScenario(scenarioReq); err != nil {
		t.Fatalf("failed to create scenario: %v", err)
	}

	if err := svc.StartScenario(chatID, "test-scenario"); err != nil {
		t.Fatalf("failed to start scenario: %v", err)
	}

	paymentDate := time.Now()
	paymentLink := "https://pay.example.com/abc"

	// Set payment info
	if err := svc.SetPaymentInfo(chatID, "test-scenario", paymentDate, paymentLink); err != nil {
		t.Fatalf("failed to set payment info: %v", err)
	}

	// Verify state
	state, _ := svc.GetUserScenario(chatID, "test-scenario")
	if state.PaymentDate == nil {
		t.Error("expected PaymentDate to be set")
	}
	if state.PaymentLink != paymentLink {
		t.Errorf("expected payment link %s, got %s", paymentLink, state.PaymentLink)
	}
}

func TestUserService_SetInviteLink(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewUserService(tmpDir)

	// Create a user first
	chatID := "123456"
	if err := svc.CreateUser(chatID, "testuser"); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// Create a scenario first
	scenarioSvc := NewService(tmpDir)
	scenarioReq := &CreateScenarioRequest{
		ID:   "test-scenario",
		Name: "Test Scenario",
		Prodamus: domainScenario.ProdamusConfig{
			ProductName:    "Product",
			ProductPrice:   "100",
			PrivateGroupID: "-1001234567890",
		},
	}
	if _, err := scenarioSvc.CreateScenario(scenarioReq); err != nil {
		t.Fatalf("failed to create scenario: %v", err)
	}

	if err := svc.StartScenario(chatID, "test-scenario"); err != nil {
		t.Fatalf("failed to start scenario: %v", err)
	}

	inviteLink := "https://t.me/+AbCdEfGhIjKlMnOp"

	// Set invite link
	if err := svc.SetInviteLink(chatID, "test-scenario", inviteLink); err != nil {
		t.Fatalf("failed to set invite link: %v", err)
	}

	// Verify state
	state, _ := svc.GetUserScenario(chatID, "test-scenario")
	if state.InviteLink != inviteLink {
		t.Errorf("expected invite link %s, got %s", inviteLink, state.InviteLink)
	}
}

func TestUserService_SetJoinedGroup(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewUserService(tmpDir)

	// Create a user first
	chatID := "123456"
	if err := svc.CreateUser(chatID, "testuser"); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// Create a scenario first
	scenarioSvc := NewService(tmpDir)
	scenarioReq := &CreateScenarioRequest{
		ID:   "test-scenario",
		Name: "Test Scenario",
		Prodamus: domainScenario.ProdamusConfig{
			ProductName:    "Product",
			ProductPrice:   "100",
			PrivateGroupID: "-1001234567890",
		},
	}
	if _, err := scenarioSvc.CreateScenario(scenarioReq); err != nil {
		t.Fatalf("failed to create scenario: %v", err)
	}

	if err := svc.StartScenario(chatID, "test-scenario"); err != nil {
		t.Fatalf("failed to start scenario: %v", err)
	}

	joinedAt := time.Now()

	// Set joined group
	if err := svc.SetJoinedGroup(chatID, "test-scenario", true, &joinedAt); err != nil {
		t.Fatalf("failed to set joined group: %v", err)
	}

	// Verify state
	state, _ := svc.GetUserScenario(chatID, "test-scenario")
	if !state.JoinedGroup {
		t.Error("expected JoinedGroup to be true")
	}
	if state.JoinedAt == nil {
		t.Error("expected JoinedAt to be set")
	}
}
