package broadcast

import (
	"testing"
	"time"

	domainBroadcast "mispilkabot/internal/domain/broadcast"
	"mispilkabot/internal/services"
)

func TestTargetResolver_Matches_NoTargeting(t *testing.T) {
	resolver := NewTargetResolver()

	ctx := &UserContext{
		ChatID:           "123456",
		UserName:         "testuser",
		ActiveScenarioID: "test-scenario",
		HasPaid:          true,
	}

	// No targeting should match everyone
	matches := resolver.Matches(ctx, nil)
	if !matches {
		t.Error("expected to match with nil targeting")
	}

	// Empty targeting should also match everyone
	emptyTargeting := &domainBroadcast.Targeting{
		Conditions: []string{},
	}
	matches = resolver.Matches(ctx, emptyTargeting)
	if !matches {
		t.Error("expected to match with empty targeting")
	}
}

func TestTargetResolver_Matches_NoActiveScenario(t *testing.T) {
	resolver := NewTargetResolver()

	targeting := &domainBroadcast.Targeting{
		Conditions: []string{domainBroadcast.ConditionNoActiveScenario},
	}

	// User with no active scenario should match
	ctx1 := &UserContext{
		ChatID:           "123456",
		UserName:         "testuser",
		ActiveScenarioID: "",
		HasPaid:          false,
	}
	if !resolver.Matches(ctx1, targeting) {
		t.Error("expected user with no active scenario to match")
	}

	// User with active scenario should not match
	ctx2 := &UserContext{
		ChatID:           "789012",
		UserName:         "testuser2",
		ActiveScenarioID: "test-scenario",
		HasPaid:          false,
	}
	if resolver.Matches(ctx2, targeting) {
		t.Error("expected user with active scenario to not match")
	}
}

func TestTargetResolver_Matches_HasNotPaid(t *testing.T) {
	resolver := NewTargetResolver()

	targeting := &domainBroadcast.Targeting{
		Conditions: []string{domainBroadcast.ConditionHasNotPaid},
	}

	// User who hasn't paid should match
	ctx1 := &UserContext{
		ChatID:           "123456",
		UserName:         "testuser",
		ActiveScenarioID: "",
		HasPaid:          false,
	}
	if !resolver.Matches(ctx1, targeting) {
		t.Error("expected user who hasn't paid to match")
	}

	// User who has paid should not match
	ctx2 := &UserContext{
		ChatID:           "789012",
		UserName:         "testuser2",
		ActiveScenarioID: "",
		HasPaid:          true,
	}
	if resolver.Matches(ctx2, targeting) {
		t.Error("expected user who has paid to not match")
	}
}

func TestTargetResolver_Matches_CombinedConditions(t *testing.T) {
	resolver := NewTargetResolver()

	targeting := &domainBroadcast.Targeting{
		Conditions: []string{
			domainBroadcast.ConditionNoActiveScenario,
			domainBroadcast.ConditionHasNotPaid,
		},
	}

	// User with no active scenario AND hasn't paid should match
	ctx1 := &UserContext{
		ChatID:           "123456",
		UserName:         "testuser",
		ActiveScenarioID: "",
		HasPaid:          false,
	}
	if !resolver.Matches(ctx1, targeting) {
		t.Error("expected user with no scenario and no payment to match")
	}

	// User with active scenario should not match (even if hasn't paid)
	ctx2 := &UserContext{
		ChatID:           "789012",
		UserName:         "testuser2",
		ActiveScenarioID: "test-scenario",
		HasPaid:          false,
	}
	if resolver.Matches(ctx2, targeting) {
		t.Error("expected user with active scenario to not match")
	}

	// User who has paid should not match (even if no active scenario)
	ctx3 := &UserContext{
		ChatID:           "345678",
		UserName:         "testuser3",
		ActiveScenarioID: "",
		HasPaid:          true,
	}
	if resolver.Matches(ctx3, targeting) {
		t.Error("expected user who has paid to not match")
	}
}

func TestTargetResolver_Matches_UnknownCondition(t *testing.T) {
	resolver := NewTargetResolver()

	targeting := &domainBroadcast.Targeting{
		Conditions: []string{"unknown_condition"},
	}

	ctx := &UserContext{
		ChatID:           "123456",
		UserName:         "testuser",
		ActiveScenarioID: "",
		HasPaid:          false,
	}

	// Unknown condition should not match
	if resolver.Matches(ctx, targeting) {
		t.Error("expected unknown condition to not match")
	}
}

func TestTargetResolver_MatchesFromMap(t *testing.T) {
	resolver := NewTargetResolver()

	now := time.Now()

	users := map[string]services.User{
		"user1": {
			UserName:         "user1",
			ActiveScenarioID: "",
		},
		"user2": {
			UserName:         "user2",
			ActiveScenarioID: "test-scenario",
		},
		"user3": {
			UserName:         "user3",
			ActiveScenarioID: "",
			PaymentDate:      &now,
		},
	}

	targeting := &domainBroadcast.Targeting{
		Conditions: []string{
			domainBroadcast.ConditionNoActiveScenario,
			domainBroadcast.ConditionHasNotPaid,
		},
	}

	matches := resolver.MatchesFromMap(users, targeting)

	// Should only match user1 (no scenario, no payment)
	if len(matches) != 1 {
		t.Errorf("expected 1 match, got %d", len(matches))
	}
	if len(matches) > 0 && matches[0] != "user1" {
		t.Errorf("expected match 'user1', got %s", matches[0])
	}
}

func TestTargetResolver_MatchesFromMap_NoTargeting(t *testing.T) {
	resolver := NewTargetResolver()

	users := map[string]services.User{
		"user1": {UserName: "user1"},
		"user2": {UserName: "user2"},
		"user3": {UserName: "user3"},
	}

	// No targeting should match all users
	matches := resolver.MatchesFromMap(users, nil)
	if len(matches) != 3 {
		t.Errorf("expected 3 matches with no targeting, got %d", len(matches))
	}

	// Empty targeting should also match all users
	emptyTargeting := &domainBroadcast.Targeting{
		Conditions: []string{},
	}
	matches = resolver.MatchesFromMap(users, emptyTargeting)
	if len(matches) != 3 {
		t.Errorf("expected 3 matches with empty targeting, got %d", len(matches))
	}
}

func TestTargetResolver_buildUserContextFromMap(t *testing.T) {
	resolver := NewTargetResolver()

	now := time.Now()
	dateStr := now.Format("2006-01-02")

	user := services.User{
		UserName:         "testuser",
		ActiveScenarioID: "test-scenario",
		PaymentDate:      &now,
	}

	ctx := resolver.buildUserContextFromMap("123456", user)

	if ctx.ChatID != "123456" {
		t.Errorf("expected ChatID '123456', got %s", ctx.ChatID)
	}
	if ctx.UserName != "testuser" {
		t.Errorf("expected UserName 'testuser', got %s", ctx.UserName)
	}
	if ctx.ActiveScenarioID != "test-scenario" {
		t.Errorf("expected ActiveScenarioID 'test-scenario', got %s", ctx.ActiveScenarioID)
	}
	if !ctx.HasPaid {
		t.Error("expected HasPaid to be true")
	}
	if ctx.PaymentDate == nil {
		t.Error("expected PaymentDate to be set")
	} else if *ctx.PaymentDate != dateStr {
		t.Errorf("expected PaymentDate '%s', got %s", dateStr, *ctx.PaymentDate)
	}
}

func TestTargetResolver_buildUserContextFromMap_NoPayment(t *testing.T) {
	resolver := NewTargetResolver()

	user := services.User{
		UserName:         "testuser",
		ActiveScenarioID: "",
	}

	ctx := resolver.buildUserContextFromMap("123456", user)

	if ctx.HasPaid {
		t.Error("expected HasPaid to be false")
	}
	if ctx.PaymentDate != nil {
		t.Error("expected PaymentDate to be nil")
	}
}

func TestTargetResolver_buildUserContextFromMap_ScenarioPayment(t *testing.T) {
	resolver := NewTargetResolver()

	now := time.Now()

	user := services.User{
		UserName:         "testuser",
		ActiveScenarioID: "test-scenario",
		Scenarios: map[string]*services.UserScenarioState{
			"test-scenario": {
				PaymentDate: &now,
			},
		},
	}

	ctx := resolver.buildUserContextFromMap("123456", user)

	if !ctx.HasPaid {
		t.Error("expected HasPaid to be true based on scenario payment")
	}
}
