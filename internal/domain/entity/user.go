package entity

import (
	"time"

	"errors"

	"github.com/google/uuid"
)

type User struct {
	ID         uuid.UUID
	TelegramID int64
	Username   string
	FirstName  string
	LastName   string
	Role       string
	CreatedAt  time.Time
	IsActive   bool
	BlockedAt  time.Time
}

func (u *User) Validate() error {
	if u.Username == "" {
		return errors.New("invalid input: username is required")
	}
	if u.Role == "" {
		return errors.New("invalid input: role is required")
	}
	return nil
}
