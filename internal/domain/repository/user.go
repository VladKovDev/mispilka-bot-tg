package repository

import (
	"context"

	"github.com/VladKovDev/promo-bot/internal/domain/entity"
	"github.com/google/uuid"
)

type UserRepository interface {
	Create(ctx context.Context, user *entity.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*entity.User, error)
	GetByTelegramID(ctx context.Context, telegramID *int64) (*entity.User, error)
	Update(ctx context.Context, user *entity.User) error
	Delete(ctx context.Context, id uuid.UUID) error
	Deactivate(ctx context.Context, id uuid.UUID) (uuid.UUID, error)

	Count(ctx context.Context) (int64, error)
	ListAll(ctx context.Context, limit, offset int) ([]*entity.User, error)
}
