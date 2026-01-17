package services

import (
	"fmt"
	"mispilkabot/internal/models"
	"strconv"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type User struct {
	UserName     string                 `json:"user_name"`
	RegTime      time.Time              `json:"reg_time"`
	IsMessaging  bool                   `json:"is_messaging"`
	MessagesList []string               `json:"messages_list"`
	PaymentDate  *time.Time             `json:"payment_date,omitempty"`
	PaymentLink  string                 `json:"payment_link,omitempty"`
	InviteLink   string                 `json:"invite_link,omitempty"`
	JoinedGroup  bool                   `json:"joined_group,omitempty"`
	JoinedAt     *time.Time             `json:"joined_at,omitempty"`
	PaymentInfo  *models.WebhookPayload `json:"payment_info,omitempty"` // full webhook payload
}

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
