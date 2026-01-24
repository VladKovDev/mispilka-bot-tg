package scenario

import "errors"

var (
	ErrInvalidScenarioID     = errors.New("invalid scenario ID")
	ErrInvalidScenarioName   = errors.New("invalid scenario name")
	ErrInvalidProductName    = errors.New("invalid product name")
	ErrInvalidProductPrice   = errors.New("invalid product price")
	ErrInvalidPrivateGroupID = errors.New("invalid private group ID")
	ErrScenarioNotFound      = errors.New("scenario not found")
)

// ScenarioStatus represents user's status in a scenario
type ScenarioStatus string

const (
	StatusNotStarted ScenarioStatus = "not_started"
	StatusActive     ScenarioStatus = "active"
	StatusCompleted  ScenarioStatus = "completed"
	StatusStopped    ScenarioStatus = "stopped"
)

// String returns the string representation
func (s ScenarioStatus) String() string {
	return string(s)
}

// IsValid checks if the status is valid
func (s ScenarioStatus) IsValid() bool {
	switch s {
	case StatusNotStarted, StatusActive, StatusCompleted, StatusStopped:
		return true
	}
	return false
}
