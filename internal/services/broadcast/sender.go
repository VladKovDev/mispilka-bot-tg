package broadcast

import (
	domainBroadcast "mispilkabot/internal/domain/broadcast"
)

// UserWithScenarios interface for checking user state
type UserWithScenarios interface {
	GetActiveScenarioID() string
	HasPaidAnyProduct() bool
}

// Sender handles broadcast sending with targeting
type Sender struct {
	registry *Registry
}

// NewSender creates a new broadcast sender
func NewSender(registry *Registry) *Sender {
	return &Sender{
		registry: registry,
	}
}

// ShouldSend determines if a broadcast should be sent to a user
func (s *Sender) ShouldSend(bc *domainBroadcast.Broadcast, user UserWithScenarios) bool {
	if bc.Targeting == nil || len(bc.Targeting.Conditions) == 0 {
		// No targeting - send to everyone
		return true
	}

	for _, condition := range bc.Targeting.Conditions {
		if !s.checkCondition(condition, user) {
			return false
		}
	}

	return true
}

// checkCondition checks a single targeting condition
func (s *Sender) checkCondition(condition string, user UserWithScenarios) bool {
	switch condition {
	case domainBroadcast.ConditionNoActiveScenario:
		return user.GetActiveScenarioID() == ""
	case domainBroadcast.ConditionHasNotPaid:
		return !user.HasPaidAnyProduct()
	default:
		// Unknown condition - assume false
		return false
	}
}

// CalculateRecipients calculates which users should receive a broadcast
func (s *Sender) CalculateRecipients(bc *domainBroadcast.Broadcast, users []UserWithScenarios) []UserWithScenarios {
	recipients := make([]UserWithScenarios, 0)
	for _, user := range users {
		if s.ShouldSend(bc, user) {
			recipients = append(recipients, user)
		}
	}
	return recipients
}
