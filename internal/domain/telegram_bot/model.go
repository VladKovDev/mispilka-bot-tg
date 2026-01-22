package telegram_bot

import (
	"fmt"
	"time"
)

type TelegramBot struct {
	ID                int64
	EncryptedToken    []byte
	EncryptionVersion int
	Username          string
	FirstName         string
	BotID             int64
	OwnerID           int64
	Status            string
	LastError         string
	LastCheckedAt     time.Time
	RevokedAt         time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func (b *TelegramBot) Validate() error {
	validStatuses := map[string]bool{
		"active":   true,
		"inactive": true,
		"error":    true,
	}
	if !validStatuses[b.Status] {
		return fmt.Errorf("invalid status: %s", b.Status)
	}
	return nil
}
