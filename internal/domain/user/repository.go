package user

import (
	"context"
	"github.com/google/uuid"
)

type Repository interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetByTelegramID(ctx context.Context, telegramID *int64) (*User, error)
	Update(ctx context.Context, user *User) error
	Delete(ctx context.Context, id uuid.UUID) error
	Deactivate(ctx context.Context, id uuid.UUID) (uuid.UUID, error)

	Count(ctx context.Context) (int64, error)
	ListAll(ctx context.Context, limit, offset int) ([]*User, error)
}
