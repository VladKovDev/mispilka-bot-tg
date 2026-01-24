package scenario

import "time"

// UserScenarioState tracks user's progress in a scenario
type UserScenarioState struct {
	Status              ScenarioStatus `json:"status"`
	CurrentMessageIndex int            `json:"current_message_index"`
	LastSentAt          *time.Time     `json:"last_sent_at,omitempty"`
	CompletedAt         *time.Time     `json:"completed_at,omitempty"`
	PaymentDate         *time.Time     `json:"payment_date,omitempty"`
	PaymentLink         string         `json:"payment_link,omitempty"`
	InviteLink          string         `json:"invite_link,omitempty"`
	JoinedGroup         bool           `json:"joined_group,omitempty"`
	JoinedAt            *time.Time     `json:"joined_at,omitempty"`
}

// IsCompleted returns true if scenario is completed
func (s *UserScenarioState) IsCompleted() bool {
	return s.Status == StatusCompleted
}

// IsActive returns true if scenario is active
func (s *UserScenarioState) IsActive() bool {
	return s.Status == StatusActive
}

// IsNotStarted returns true if scenario is not started
func (s *UserScenarioState) IsNotStarted() bool {
	return s.Status == StatusNotStarted
}

// IsStopped returns true if scenario is stopped
func (s *UserScenarioState) IsStopped() bool {
	return s.Status == StatusStopped
}

// MarkCompleted marks scenario as completed
func (s *UserScenarioState) MarkCompleted() {
	now := time.Now()
	s.Status = StatusCompleted
	s.CompletedAt = &now
}

// MarkActive marks scenario as active
func (s *UserScenarioState) MarkActive() {
	s.Status = StatusActive
}

// MarkStopped marks scenario as stopped
func (s *UserScenarioState) MarkStopped() {
	s.Status = StatusStopped
}

// Clone creates a deep copy of the state
func (s *UserScenarioState) Clone() *UserScenarioState {
	clone := *s
	if clone.LastSentAt != nil {
		t := *clone.LastSentAt
		clone.LastSentAt = &t
	}
	if clone.CompletedAt != nil {
		t := *clone.CompletedAt
		clone.CompletedAt = &t
	}
	if clone.PaymentDate != nil {
		t := *clone.PaymentDate
		clone.PaymentDate = &t
	}
	if clone.JoinedAt != nil {
		t := *clone.JoinedAt
		clone.JoinedAt = &t
	}
	return &clone
}
