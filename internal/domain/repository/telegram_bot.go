package repository

import (
	"context"

	"github.com/VladKovDev/promo-bot/internal/domain/entity"
	"github.com/google/uuid"
)

type TelegramBotRepository interface {
	Create(ctx context.Context, bot *entity.TelegramBot) error
	GetByID(ctx context.Context, id uuid.UUID) (*entity.TelegramBot, error)
	GetByTelegramID(ctx context.Context, telegramID int64) (*entity.TelegramBot, error)
	Update(ctx context.Context, bot *entity.TelegramBot) error
	Delete(ctx context.Context, id uuid.UUID) error
	ListAll(ctx context.Context) ([]*entity.TelegramBot, error)
}
