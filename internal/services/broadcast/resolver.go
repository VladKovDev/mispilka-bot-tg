package broadcast

import (
	"fmt"

	domainBroadcast "mispilkabot/internal/domain/broadcast"
	"mispilkabot/internal/services"
)

// TargetResolver resolves which users match targeting conditions
type TargetResolver struct{}

// NewTargetResolver creates a new target resolver
func NewTargetResolver() *TargetResolver {
	return &TargetResolver{}
}

// UserContext contains information about a user for targeting
type UserContext struct {
	ChatID            string
	UserName          string
	ActiveScenarioID  string
	HasPaid           bool
	PaymentDate       *string
}

// Matches checks if a user matches the targeting conditions
func (r *TargetResolver) Matches(ctx *UserContext, targeting *domainBroadcast.Targeting) bool {
	if targeting == nil || len(targeting.Conditions) == 0 {
		return true // No targeting means match everyone
	}

	for _, condition := range targeting.Conditions {
		if !r.matchesCondition(ctx, condition) {
			return false // Any condition that doesn't match means user doesn't match
		}
	}

	return true // All conditions matched
}

// matchesCondition checks if a user matches a single targeting condition
func (r *TargetResolver) matchesCondition(ctx *UserContext, condition string) bool {
	switch condition {
	case domainBroadcast.ConditionNoActiveScenario:
		return ctx.ActiveScenarioID == ""

	case domainBroadcast.ConditionHasNotPaid:
		return !ctx.HasPaid

	default:
		// Unknown condition - treat as not matching
		return false
	}
}

// FindTargetsForBroadcast finds all users that match a broadcast's targeting
// This is a helper for batch operations
func (r *TargetResolver) FindTargetsForBroadcast(broadcast *domainBroadcast.Broadcast, usersFn func() ([]*services.User, error)) ([]string, error) {
	users, err := usersFn()
	if err != nil {
		return nil, fmt.Errorf("failed to load users: %w", err)
	}

	var targets []string

	for _, user := range users {
		ctx := r.buildUserContext(user)
		if r.Matches(ctx, broadcast.Targeting) {
			targets = append(targets, userChatID(user))
		}
	}

	return targets, nil
}

// buildUserContext builds targeting context from a user
func (r *TargetResolver) buildUserContext(user *services.User) *UserContext {
	ctx := &UserContext{
		ChatID:   userChatID(user),
		UserName: user.UserName,
	}

	// Get active scenario
	ctx.ActiveScenarioID = user.ActiveScenarioID

	// Check if user has paid (using legacy field for now)
	if user.PaymentDate != nil && !user.PaymentDate.IsZero() {
		ctx.HasPaid = true
		// Format payment date as string
		dateStr := user.PaymentDate.Format("2006-01-02")
		ctx.PaymentDate = &dateStr
	}

	// Also check scenarios for payment status
	for _, state := range user.Scenarios {
		if state.PaymentDate != nil && !state.PaymentDate.IsZero() {
			ctx.HasPaid = true
			break
		}
	}

	return ctx
}

// userChatID extracts chat ID from user - helper for when we don't have a direct map
// This assumes we're iterating over users from a map where key is chatID
func userChatID(_ *services.User) string {
	// For now, we'll need the chatID passed separately
	// This is a placeholder - in practice, when iterating over a map, use the key
	return ""
}

// MatchesFromMap checks if users from a map match targeting conditions
// Returns a list of chatIDs that match
func (r *TargetResolver) MatchesFromMap(users map[string]services.User, targeting *domainBroadcast.Targeting) []string {
	var matches []string

	for chatID, user := range users {
		ctx := r.buildUserContextFromMap(chatID, user)
		if r.Matches(ctx, targeting) {
			matches = append(matches, chatID)
		}
	}

	return matches
}

// buildUserContextFromMap builds targeting context from a user with their chatID
func (r *TargetResolver) buildUserContextFromMap(chatID string, user services.User) *UserContext {
	ctx := &UserContext{
		ChatID:   chatID,
		UserName: user.UserName,
	}

	// Get active scenario
	ctx.ActiveScenarioID = user.ActiveScenarioID

	// Check if user has paid (using legacy field for now)
	if user.PaymentDate != nil && !user.PaymentDate.IsZero() {
		ctx.HasPaid = true
		// Format payment date as string
		dateStr := user.PaymentDate.Format("2006-01-02")
		ctx.PaymentDate = &dateStr
	}

	// Also check scenarios for payment status
	for _, state := range user.Scenarios {
		if state.PaymentDate != nil && !state.PaymentDate.IsZero() {
			ctx.HasPaid = true
			break
		}
	}

	return ctx
}
