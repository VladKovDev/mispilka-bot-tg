package user

import (
	"github.com/google/uuid"
)

type User struct {
	ID         uuid.UUID
	TelegramID int64
	Username   string
	FirstName  string
	LastName   string
	IsActive   bool
}

func (u *User) Validate() error {
	return nil
}
