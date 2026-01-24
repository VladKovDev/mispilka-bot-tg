package telegram_bot

import (
	"time"

	"github.com/google/uuid"
)

type TelegramBot struct {
	ID         uuid.UUID
	BotID      int64
	Token      string
	Username   string
	FirstName  string
	LastName   string
	Role       string
	RevokedAt  *time.Time
	DisabledAt *time.Time
}

func (tb *TelegramBot) IsActive() bool {
	return tb.RevokedAt == nil && tb.DisabledAt == nil
}
