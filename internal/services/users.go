package services

import (
	"fmt"
	"mispilkabot/internal/models"
	"strconv"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// User represents a user in the system
type User struct {
	UserName     string                         `json:"user_name"`
	RegTime      time.Time                      `json:"reg_time"`
	IsMessaging  bool                           `json:"is_messaging"`
	Scenarios    map[string]*UserScenarioState  `json:"scenarios,omitempty"`
	ActiveScenarioID    string                         `json:"active_scenario_id,omitempty"`
	// Legacy fields for migration compatibility
	MessagesList        []string                       `json:"messages_list,omitempty"`
	PaymentDate         *time.Time                     `json:"payment_date,omitempty"`
	PaymentLink         string                         `json:"payment_link,omitempty"`
	InviteLink          string                         `json:"invite_link,omitempty"`
	JoinedGroup         bool                           `json:"joined_group,omitempty"`
	JoinedAt            *time.Time                     `json:"joined_at,omitempty"`
	PaymentInfo         *models.WebhookPayload         `json:"payment_info,omitempty"`
}

// UserScenarioState tracks user's progress in a scenario
type UserScenarioState struct {
	Status              ScenarioStatus   `json:"status"`
	CurrentMessageIndex int              `json:"current_message_index"`
	LastSentAt          *time.Time       `json:"last_sent_at,omitempty"`
	CompletedAt         *time.Time       `json:"completed_at,omitempty"`
	PaymentDate         *time.Time       `json:"payment_date,omitempty"`
	PaymentLink         string           `json:"payment_link,omitempty"`
	InviteLink          string           `json:"invite_link,omitempty"`
	JoinedGroup         bool             `json:"joined_group,omitempty"`
	JoinedAt            *time.Time       `json:"joined_at,omitempty"`
}

// ScenarioStatus represents user's status in a scenario
type ScenarioStatus string

const (
	StatusNotStarted ScenarioStatus = "not_started"
	StatusActive     ScenarioStatus = "active"
	StatusCompleted  ScenarioStatus = "completed"
	StatusStopped    ScenarioStatus = "stopped"
)

type UserMap map[string]User

// AddUser creates a new user entry in the users.json file.
// Returns an error if the user data cannot be read, prepared, or written.
func AddUser(message *tgbotapi.Message) error {
	data, err := ReadJSONRetry[UserMap]("data/users.json", 3)
	if err != nil {
		return fmt.Errorf("failed to read users data: %w", err)
	}

	if err = data.userData(message); err != nil {
		return fmt.Errorf("failed to prepare user data: %w", err)
	}

	if err = WriteJSONRetry("data/users.json", data, 3); err != nil {
		return fmt.Errorf("failed to write users data: %w", err)
	}
	return nil
}

// userData creates a new User entry from a telegram message.
// This is called by AddUser to prepare the user data before writing to disk.
func (data UserMap) userData(message *tgbotapi.Message) error {
	t := time.Now()
	// Get messages from messages_list - only these are sequential flow messages
	messagesList, err := getMessagesList()
	if err != nil {
		return err
	}
	chatID := strconv.FormatInt(message.Chat.ID, 10)
	data[chatID] = User{
		UserName:     message.From.UserName,
		RegTime:      t,
		IsMessaging:  false,
		MessagesList: messagesList,
	}
	return nil
}

func GetUser(chatID string) (User, error) {
	var users UserMap
	var user User

	users, err := ReadJSONRetry[UserMap]("data/users.json", 3)
	if err != nil {
		return user, err
	}

	user, ok := users[chatID]
	if !ok {
		return user, fmt.Errorf("user not found")
	}
	return user, nil
}

// SetIsMessaging updates the messaging status for a user
func SetIsMessaging(chatID string, status bool) error {
	userData, err := GetUser(chatID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}
	userData.IsMessaging = status
	if err := ChangeUser(chatID, userData); err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}
	return nil
}

// ChangeIsMessaging is deprecated - use SetIsMessaging instead
func ChangeIsMessaging(chatID string, status bool) error {
	return SetIsMessaging(chatID, status)
}

// SetPaymentDate updates the payment date and resets messaging status for a user
func SetPaymentDate(chatID string, paymentDate time.Time) error {
	userData, err := GetUser(chatID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}
	userData.PaymentDate = &paymentDate
	userData.IsMessaging = false
	if err := ChangeUser(chatID, userData); err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}
	return nil
}

// SetPaymentLink updates the payment link for a user
func SetPaymentLink(chatID, paymentLink string) error {
	userData, err := GetUser(chatID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}
	userData.PaymentLink = paymentLink
	if err := ChangeUser(chatID, userData); err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}
	return nil
}

// SetInviteLink updates the invite link for a user
func SetInviteLink(chatID, inviteLink string) error {
	userData, err := GetUser(chatID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}
	userData.InviteLink = inviteLink
	if err := ChangeUser(chatID, userData); err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}
	return nil
}

// SetJoinedGroup updates the joined group status for a user
func SetJoinedGroup(chatID string, joined bool) error {
	userData, err := GetUser(chatID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}
	userData.JoinedGroup = joined
	if err := ChangeUser(chatID, userData); err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}
	return nil
}

// ChangeUser updates an existing user entry in the users.json file.
// Returns an error if the user data cannot be read, the user doesn't exist, or the write fails.
func ChangeUser(chatID string, userData User) error {
	users, err := ReadJSONRetry[UserMap]("data/users.json", 3)
	if err != nil {
		return fmt.Errorf("failed to read users data: %w", err)
	}

	users[chatID] = userData

	if err := WriteJSONRetry("data/users.json", users, 3); err != nil {
		return fmt.Errorf("failed to write users data: %w", err)
	}
	return nil
}

// HasPaid returns true if the user has paid (PaymentDate is not nil and not zero time).
func (u *User) HasPaid() bool {
	if u.PaymentDate == nil {
		return false
	}
	return !u.PaymentDate.IsZero()
}

// GetPaymentDate returns the payment date if set, otherwise returns zero time.
// Use HasPaid() to check if the user has paid.
func (u *User) GetPaymentDate() time.Time {
	if u.PaymentDate == nil {
		return time.Time{}
	}
	return *u.PaymentDate
}

// HasJoined returns true if the user has joined the group (JoinedGroup is true).
func (u *User) HasJoined() bool {
	return u.JoinedGroup
}

// GetJoinedAt returns the time when the user joined the group if set, otherwise returns zero time.
func (u *User) GetJoinedAt() time.Time {
	if u.JoinedAt == nil {
		return time.Time{}
	}
	return *u.JoinedAt
}

// IsNewUser checks if a user with the given chat ID exists in the system.
// Returns true if the user doesn't exist (is new), false if they exist.
// Returns an error if the user data cannot be read.
func IsNewUser(chatID string) (bool, error) {
	users, err := ReadJSONRetry[UserMap]("data/users.json", 3)
	if err != nil {
		return false, fmt.Errorf("failed to load users: %w", err)
	}
	_, ok := users[chatID]
	return !ok, nil
}

// GetAllUsers returns all users from the users.json file.
// Returns an error if the user data cannot be read.
func GetAllUsers() (UserMap, error) {
	users, err := ReadJSONRetry[UserMap]("data/users.json", 3)
	if err != nil {
		return nil, fmt.Errorf("failed to load users: %w", err)
	}
	return users, nil
}

// GetUserScenario retrieves user's state for a specific scenario
func GetUserScenario(chatID, scenarioID string) (*UserScenarioState, error) {
	user, err := GetUser(chatID)
	if err != nil {
		return nil, err
	}

	if user.Scenarios == nil {
		// Initialize with not_started state
		return &UserScenarioState{Status: StatusNotStarted}, nil
	}

	state, ok := user.Scenarios[scenarioID]
	if !ok {
		return &UserScenarioState{Status: StatusNotStarted}, nil
	}

	return state, nil
}

// SetUserScenario sets user's state for a specific scenario
func SetUserScenario(chatID, scenarioID string, state *UserScenarioState) error {
	users, err := ReadJSONRetry[UserMap]("data/users.json", 3)
	if err != nil {
		return fmt.Errorf("failed to read users data: %w", err)
	}

	user, ok := users[chatID]
	if !ok {
		return fmt.Errorf("user not found")
	}

	if user.Scenarios == nil {
		user.Scenarios = make(map[string]*UserScenarioState)
	}

	user.Scenarios[scenarioID] = state

	return WriteJSONRetry("data/users.json", users, 3)
}

// GetUserActiveScenario retrieves user's active scenario
func GetUserActiveScenario(chatID string) (string, *UserScenarioState, error) {
	user, err := GetUser(chatID)
	if err != nil {
		return "", nil, err
	}

	if user.ActiveScenarioID == "" {
		return "", nil, nil // No active scenario
	}

	state, err := GetUserScenario(chatID, user.ActiveScenarioID)
	if err != nil {
		return "", nil, err
	}

	return user.ActiveScenarioID, state, nil
}

// SetUserActiveScenario sets user's active scenario
func SetUserActiveScenario(chatID, scenarioID string) error {
	users, err := ReadJSONRetry[UserMap]("data/users.json", 3)
	if err != nil {
		return fmt.Errorf("failed to read users data: %w", err)
	}

	user, ok := users[chatID]
	if !ok {
		return fmt.Errorf("user not found")
	}

	user.ActiveScenarioID = scenarioID

	return WriteJSONRetry("data/users.json", users, 3)
}

// IsScenarioCompleted checks if user has completed a scenario
func IsScenarioCompleted(chatID, scenarioID string) (bool, error) {
	state, err := GetUserScenario(chatID, scenarioID)
	if err != nil {
		return false, err
	}
	return state.Status == StatusCompleted, nil
}
