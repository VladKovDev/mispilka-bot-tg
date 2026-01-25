package scenario

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"mispilkabot/internal/services"
)

// UserService provides high-level user scenario operations
type UserService struct {
	baseDir      string
	scenarioSvc  *Service
	usersPath    string
	mu           sync.RWMutex
}

// NewUserService creates a new user scenario service
func NewUserService(baseDir string) *UserService {
	return &UserService{
		baseDir:     baseDir,
		scenarioSvc: NewService(baseDir),
		usersPath:   filepath.Join(baseDir, "users.json"),
	}
}

// loadUsers loads users from the service's users.json file
func (s *UserService) loadUsers() (map[string]services.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := os.ReadFile(s.usersPath)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]services.User), nil
		}
		return nil, fmt.Errorf("failed to read users: %w", err)
	}

	var users map[string]services.User
	if err := json.Unmarshal(data, &users); err != nil {
		return nil, fmt.Errorf("failed to unmarshal users: %w", err)
	}

	if users == nil {
		return make(map[string]services.User), nil
	}

	return users, nil
}

// saveUsers saves users to the service's users.json file
func (s *UserService) saveUsers(users map[string]services.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(s.usersPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	data, err := json.MarshalIndent(users, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal users: %w", err)
	}

	if err := os.WriteFile(s.usersPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write users: %w", err)
	}

	return nil
}

// StartScenario starts a scenario for a user
func (s *UserService) StartScenario(chatID, scenarioID string) error {
	// Verify scenario exists
	if _, err := s.scenarioSvc.GetScenario(scenarioID); err != nil {
		return fmt.Errorf("scenario not found: %w", err)
	}

	// Load users
	users, err := s.loadUsers()
	if err != nil {
		return err
	}

	// Get user
	user, ok := users[chatID]
	if !ok {
		return fmt.Errorf("user not found")
	}

	// Create initial state
	now := time.Now()
	state := &services.UserScenarioState{
		Status:              services.StatusActive,
		CurrentMessageIndex: 0,
		LastSentAt:          &now,
	}

	// Set as active scenario and save state
	user.ActiveScenarioID = scenarioID
	if user.Scenarios == nil {
		user.Scenarios = make(map[string]*services.UserScenarioState)
	}
	user.Scenarios[scenarioID] = state

	users[chatID] = user

	return s.saveUsers(users)
}

// GetUserScenario retrieves user's state for a specific scenario
func (s *UserService) GetUserScenario(chatID, scenarioID string) (*services.UserScenarioState, error) {
	users, err := s.loadUsers()
	if err != nil {
		return nil, err
	}

	user, ok := users[chatID]
	if !ok {
		return &services.UserScenarioState{Status: services.StatusNotStarted}, nil
	}

	if user.Scenarios == nil {
		return &services.UserScenarioState{Status: services.StatusNotStarted}, nil
	}

	state, ok := user.Scenarios[scenarioID]
	if !ok {
		return &services.UserScenarioState{Status: services.StatusNotStarted}, nil
	}

	return state, nil
}

// GetActiveScenario retrieves user's active scenario
func (s *UserService) GetActiveScenario(chatID string) (string, *services.UserScenarioState, error) {
	users, err := s.loadUsers()
	if err != nil {
		return "", nil, err
	}

	user, ok := users[chatID]
	if !ok {
		return "", nil, nil // No user
	}

	if user.ActiveScenarioID == "" {
		return "", nil, nil // No active scenario
	}

	state, err := s.GetUserScenario(chatID, user.ActiveScenarioID)
	if err != nil {
		return "", nil, err
	}

	return user.ActiveScenarioID, state, nil
}

// AdvanceToNextMessage advances user to next message in scenario
// Returns the next message ID, or empty string if there are no more messages
func (s *UserService) AdvanceToNextMessage(chatID, scenarioID string) (string, error) {
	// Get current state
	state, err := s.GetUserScenario(chatID, scenarioID)
	if err != nil {
		return "", err
	}

	// Get next message ID from scenario
	currentID := ""
	if state.CurrentMessageIndex > 0 {
		// Need to get the current message ID
		scenario, err := s.scenarioSvc.GetScenario(scenarioID)
		if err != nil {
			return "", err
		}
		if state.CurrentMessageIndex <= len(scenario.Messages.MessagesList) {
			currentID = scenario.Messages.MessagesList[state.CurrentMessageIndex-1]
		}
	}

	nextID, err := s.scenarioSvc.GetNextMessageID(scenarioID, currentID)
	if err != nil {
		return "", err
	}

	if nextID == "" {
		return "", nil // No more messages
	}

	// Update state
	users, err := s.loadUsers()
	if err != nil {
		return "", err
	}

	user := users[chatID]
	now := time.Now()
	state.CurrentMessageIndex++
	state.LastSentAt = &now
	user.Scenarios[scenarioID] = state
	users[chatID] = user

	if err := s.saveUsers(users); err != nil {
		return "", err
	}

	return nextID, nil
}

// CompleteScenario marks a scenario as completed for a user
func (s *UserService) CompleteScenario(chatID, scenarioID string) error {
	users, err := s.loadUsers()
	if err != nil {
		return err
	}

	user := users[chatID]
	state := user.Scenarios[scenarioID]
	if state == nil {
		return fmt.Errorf("scenario %s not started for user %s", scenarioID, chatID)
	}

	now := time.Now()
	state.Status = services.StatusCompleted
	state.CompletedAt = &now

	user.Scenarios[scenarioID] = state
	users[chatID] = user

	return s.saveUsers(users)
}

// StopScenario stops a scenario for a user
func (s *UserService) StopScenario(chatID, scenarioID string) error {
	users, err := s.loadUsers()
	if err != nil {
		return err
	}

	user := users[chatID]
	state := user.Scenarios[scenarioID]
	if state == nil {
		return fmt.Errorf("scenario %s not started for user %s", scenarioID, chatID)
	}
	state.Status = services.StatusStopped

	user.Scenarios[scenarioID] = state
	users[chatID] = user

	return s.saveUsers(users)
}

// SetPaymentInfo sets payment information for a user's scenario
func (s *UserService) SetPaymentInfo(chatID, scenarioID string, paymentDate time.Time, paymentLink string) error {
	users, err := s.loadUsers()
	if err != nil {
		return err
	}

	user := users[chatID]
	state := user.Scenarios[scenarioID]
	if state == nil {
		return fmt.Errorf("scenario %s not started for user %s", scenarioID, chatID)
	}

	state.PaymentDate = &paymentDate
	state.PaymentLink = paymentLink

	user.Scenarios[scenarioID] = state
	users[chatID] = user

	return s.saveUsers(users)
}

// SetInviteLink sets invite link for a user's scenario
func (s *UserService) SetInviteLink(chatID, scenarioID, inviteLink string) error {
	users, err := s.loadUsers()
	if err != nil {
		return err
	}

	user := users[chatID]
	state := user.Scenarios[scenarioID]
	if state == nil {
		return fmt.Errorf("scenario %s not started for user %s", scenarioID, chatID)
	}

	state.InviteLink = inviteLink

	user.Scenarios[scenarioID] = state
	users[chatID] = user

	return s.saveUsers(users)
}

// SetJoinedGroup marks that user has joined the group
func (s *UserService) SetJoinedGroup(chatID, scenarioID string, joined bool, joinedAt *time.Time) error {
	users, err := s.loadUsers()
	if err != nil {
		return err
	}

	user := users[chatID]
	state := user.Scenarios[scenarioID]
	if state == nil {
		return fmt.Errorf("scenario %s not started for user %s", scenarioID, chatID)
	}

	state.JoinedGroup = joined
	state.JoinedAt = joinedAt

	user.Scenarios[scenarioID] = state
	users[chatID] = user

	return s.saveUsers(users)
}

// IsScenarioCompleted checks if user has completed a scenario
func (s *UserService) IsScenarioCompleted(chatID, scenarioID string) (bool, error) {
	state, err := s.GetUserScenario(chatID, scenarioID)
	if err != nil {
		return false, err
	}
	return state.Status == services.StatusCompleted, nil
}

// GetScenarioProgress returns user's progress through a scenario
// Returns (currentMessageIndex, totalMessages, error)
func (s *UserService) GetScenarioProgress(chatID, scenarioID string) (int, int, error) {
	state, err := s.GetUserScenario(chatID, scenarioID)
	if err != nil {
		return 0, 0, err
	}

	scenario, err := s.scenarioSvc.GetScenario(scenarioID)
	if err != nil {
		return 0, 0, err
	}

	totalMessages := len(scenario.Messages.MessagesList)
	return state.CurrentMessageIndex, totalMessages, nil
}

// HasActiveScenario checks if user has an active scenario
func (s *UserService) HasActiveScenario(chatID string) (bool, string, error) {
	scenarioID, state, err := s.GetActiveScenario(chatID)
	if err != nil {
		return false, "", err
	}

	if scenarioID == "" || state == nil {
		return false, "", nil
	}

	return state.Status == services.StatusActive, scenarioID, nil
}

// SwitchScenario switches user from current active scenario to a new one
func (s *UserService) SwitchScenario(chatID, newScenarioID string) error {
	// Verify new scenario exists
	if _, err := s.scenarioSvc.GetScenario(newScenarioID); err != nil {
		return fmt.Errorf("scenario not found: %w", err)
	}

	// Stop current active scenario if any
	currentScenarioID, _, err := s.GetActiveScenario(chatID)
	if err == nil && currentScenarioID != "" {
		if err := s.StopScenario(chatID, currentScenarioID); err != nil {
			return fmt.Errorf("failed to stop current scenario: %w", err)
		}
	}

	// Start new scenario
	return s.StartScenario(chatID, newScenarioID)
}

// CreateUser creates a new user entry
func (s *UserService) CreateUser(chatID, userName string) error {
	users, err := s.loadUsers()
	if err != nil {
		return fmt.Errorf("failed to load users: %w", err)
	}

	if _, exists := users[chatID]; exists {
		return nil // User already exists, that's fine
	}

	// Create new user
	now := time.Now()
	users[chatID] = services.User{
		UserName:    userName,
		RegTime:     now,
		IsMessaging: false,
		Scenarios:   make(map[string]*services.UserScenarioState),
	}

	return s.saveUsers(users)
}
