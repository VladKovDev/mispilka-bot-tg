package broadcast

import (
	"testing"
	"time"

	domainBroadcast "mispilkabot/internal/domain/broadcast"
)

func TestSender_ShouldSend(t *testing.T) {
	sender := NewSender(nil)

	bc := &domainBroadcast.Broadcast{
		ID:           "test",
		Name:         "Test",
		TemplateFile: "test.md",
		Targeting: &domainBroadcast.Targeting{
			Conditions: []string{domainBroadcast.ConditionNoActiveScenario},
		},
		CreatedAt: time.Now(),
	}

	// Test with user without active scenario
	user := &MockUser{
		ActiveScenarioID: "",
	}

	if !sender.ShouldSend(bc, user) {
		t.Error("Expected to send to user without active scenario")
	}

	// Test with user with active scenario
	userWithActive := &MockUser{
		ActiveScenarioID: "default",
	}

	if sender.ShouldSend(bc, userWithActive) {
		t.Error("Expected not to send to user with active scenario")
	}
}

// MockUser for testing
type MockUser struct {
	ActiveScenarioID string
	HasPaid          bool
}

func (m *MockUser) GetActiveScenarioID() string {
	return m.ActiveScenarioID
}

func (m *MockUser) HasPaidAnyProduct() bool {
	return m.HasPaid
}

func TestSender_ShouldSend_NoTargeting(t *testing.T) {
	sender := NewSender(nil)

	bc := &domainBroadcast.Broadcast{
		ID:           "test",
		Name:         "Test",
		TemplateFile: "test.md",
		Targeting:    nil, // No targeting
		CreatedAt:    time.Now(),
	}

	user := &MockUser{
		ActiveScenarioID: "default",
		HasPaid:          true,
	}

	// Should send to everyone when no targeting
	if !sender.ShouldSend(bc, user) {
		t.Error("Expected to send to user when no targeting specified")
	}
}

func TestSender_ShouldSend_EmptyTargeting(t *testing.T) {
	sender := NewSender(nil)

	bc := &domainBroadcast.Broadcast{
		ID:           "test",
		Name:         "Test",
		TemplateFile: "test.md",
		Targeting: &domainBroadcast.Targeting{
			Conditions: []string{}, // Empty conditions
		},
		CreatedAt: time.Now(),
	}

	user := &MockUser{
		ActiveScenarioID: "default",
		HasPaid:          true,
	}

	// Should send to everyone when targeting is empty
	if !sender.ShouldSend(bc, user) {
		t.Error("Expected to send to user when targeting conditions are empty")
	}
}

func TestSender_ShouldSend_HasNotPaid(t *testing.T) {
	sender := NewSender(nil)

	bc := &domainBroadcast.Broadcast{
		ID:           "test",
		Name:         "Test",
		TemplateFile: "test.md",
		Targeting: &domainBroadcast.Targeting{
			Conditions: []string{domainBroadcast.ConditionHasNotPaid},
		},
		CreatedAt: time.Now(),
	}

	// Test with unpaid user
	unpaidUser := &MockUser{
		HasPaid: false,
	}

	if !sender.ShouldSend(bc, unpaidUser) {
		t.Error("Expected to send to unpaid user")
	}

	// Test with paid user
	paidUser := &MockUser{
		HasPaid: true,
	}

	if sender.ShouldSend(bc, paidUser) {
		t.Error("Expected not to send to paid user")
	}
}

func TestSender_ShouldSend_CombinedConditions(t *testing.T) {
	sender := NewSender(nil)

	bc := &domainBroadcast.Broadcast{
		ID:           "test",
		Name:         "Test",
		TemplateFile: "test.md",
		Targeting: &domainBroadcast.Targeting{
			Conditions: []string{
				domainBroadcast.ConditionNoActiveScenario,
				domainBroadcast.ConditionHasNotPaid,
			},
		},
		CreatedAt: time.Now(),
	}

	// Test with user matching both conditions
	userMatching := &MockUser{
		ActiveScenarioID: "",
		HasPaid:          false,
	}

	if !sender.ShouldSend(bc, userMatching) {
		t.Error("Expected to send to user matching both conditions")
	}

	// Test with user matching only one condition
	userPartialMatch := &MockUser{
		ActiveScenarioID: "",
		HasPaid:          true,
	}

	if sender.ShouldSend(bc, userPartialMatch) {
		t.Error("Expected not to send to user matching only one condition")
	}

	// Test with user matching no conditions
	userNoMatch := &MockUser{
		ActiveScenarioID: "default",
		HasPaid:          true,
	}

	if sender.ShouldSend(bc, userNoMatch) {
		t.Error("Expected not to send to user matching no conditions")
	}
}

func TestSender_ShouldSend_UnknownCondition(t *testing.T) {
	sender := NewSender(nil)

	bc := &domainBroadcast.Broadcast{
		ID:           "test",
		Name:         "Test",
		TemplateFile: "test.md",
		Targeting: &domainBroadcast.Targeting{
			Conditions: []string{"unknown_condition"},
		},
		CreatedAt: time.Now(),
	}

	user := &MockUser{
		ActiveScenarioID: "",
		HasPaid:          false,
	}

	// Unknown condition should result in not sending
	if sender.ShouldSend(bc, user) {
		t.Error("Expected not to send to user when condition is unknown")
	}
}

func TestSender_CalculateRecipients(t *testing.T) {
	sender := NewSender(nil)

	bc := &domainBroadcast.Broadcast{
		ID:           "test",
		Name:         "Test",
		TemplateFile: "test.md",
		Targeting: &domainBroadcast.Targeting{
			Conditions: []string{domainBroadcast.ConditionNoActiveScenario},
		},
		CreatedAt: time.Now(),
	}

	users := []UserWithScenarios{
		&MockUser{ActiveScenarioID: "", HasPaid: false},     // Should receive
		&MockUser{ActiveScenarioID: "default", HasPaid: true}, // Should not receive
		&MockUser{ActiveScenarioID: "", HasPaid: true},      // Should receive
		&MockUser{ActiveScenarioID: "premium", HasPaid: true}, // Should not receive
	}

	recipients := sender.CalculateRecipients(bc, users)

	if len(recipients) != 2 {
		t.Errorf("Expected 2 recipients, got %d", len(recipients))
	}

	// Verify all recipients have no active scenario
	for _, recipient := range recipients {
		if recipient.GetActiveScenarioID() != "" {
			t.Error("Expected all recipients to have no active scenario")
		}
	}
}

func TestSender_CalculateRecipients_NoTargeting(t *testing.T) {
	sender := NewSender(nil)

	bc := &domainBroadcast.Broadcast{
		ID:           "test",
		Name:         "Test",
		TemplateFile: "test.md",
		Targeting:    nil,
		CreatedAt:    time.Now(),
	}

	users := []UserWithScenarios{
		&MockUser{ActiveScenarioID: "default", HasPaid: true},
		&MockUser{ActiveScenarioID: "premium", HasPaid: true},
		&MockUser{ActiveScenarioID: "", HasPaid: false},
	}

	recipients := sender.CalculateRecipients(bc, users)

	if len(recipients) != 3 {
		t.Errorf("Expected 3 recipients (all users) when no targeting, got %d", len(recipients))
	}
}
