package telegram_bot

import (
	"context"
	"github.com/google/uuid"
)

type Repository interface {
	Create(ctx context.Context, bot *TelegramBot) error
	GetByID(ctx context.Context, id uuid.UUID) (*TelegramBot, error)
	GetByTelegramID(ctx context.Context, telegramID int64) (*TelegramBot, error)
	Update(ctx context.Context, bot *TelegramBot) error
	Delete(ctx context.Context, id uuid.UUID) error
	ListAll(ctx context.Context) ([]*TelegramBot, error)
}
