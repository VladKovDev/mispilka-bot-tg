package postgres

import (
	"context"
	"fmt"

	"github.com/VladKovDev/promo-bot/internal/domain/user"
	"github.com/VladKovDev/promo-bot/internal/infrastructure/repository/postgres/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresUserRepository struct {
	queries *sqlc.Queries
}

func NewPostgresUserRepository(db *pgxpool.Pool) user.Repository {
	return &PostgresUserRepository{
		queries: sqlc.New(db),
	}
}

func (r *PostgresUserRepository) Create(ctx context.Context, user *user.User) error {
	if err := user.Validate(); err != nil {
		return fmt.Errorf("invalid user: %w", err)
	}

	params := sqlc.CreateUserParams{
		TelegramID: &user.TelegramID,
		Username:   &user.Username,
		FirstName:  &user.FirstName,
		LastName:   &user.LastName,
	}
	sqlcUser, err := r.queries.CreateUser(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	createdUser, err := toUserEntity(sqlcUser)
	if err != nil {
		return fmt.Errorf("failed to convert sqlcUser to user entity: %w", err)
	}

	*user = *createdUser
	return nil
}

func (r *PostgresUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	sqlcID := uuidToPgtype(id)
	sqlcUser, err := r.queries.GetUserByID(ctx, sqlcID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by id: %w", err)
	}
	user, err := toUserEntity(sqlcUser)
	if err != nil {
		return nil, fmt.Errorf("failed to convert sqlcUser to user entity: %w", err)
	}
	return user, nil
}

func (r *PostgresUserRepository) GetByTelegramID(ctx context.Context, telegramID *int64) (*user.User, error) {
	sqlcUser, err := r.queries.GetUserByTelegramID(ctx, telegramID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by telegram id: %w", err)
	}
	user, err := toUserEntity(sqlcUser)
	if err != nil {
		return nil, fmt.Errorf("failed to convert sqlcUser to user entity: %w", err)
	}
	return user, nil
}

func (r *PostgresUserRepository) Update(ctx context.Context, user *user.User) error {
	arg := sqlc.UpdateUserParams{
		ID:         uuidToPgtype(user.ID),
		TelegramID: &user.TelegramID,
		Username:   &user.Username,
		FirstName:  &user.FirstName,
		LastName:   &user.LastName,}

	_, err := r.queries.UpdateUser(ctx, arg)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}
	return nil
}

func (r *PostgresUserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	sqlcID := uuidToPgtype(id)
	err := r.queries.DeleteUser(ctx, sqlcID)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	return nil
}

func (r *PostgresUserRepository) Deactivate(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
	sqlcID := uuidToPgtype(id)
	deactivatedID, err := r.queries.DeactivateUser(ctx, sqlcID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to deactivate user: %w", err)
	}
	userID, err := pgtypeToUUID(deactivatedID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to return user id: %w", err)
	}
	return userID, nil
}

func (r *PostgresUserRepository) Count(ctx context.Context) (int64, error) {
	count, err := r.queries.CountUsers(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to count users: %w", err)
	}
	return count, nil
}

func (r *PostgresUserRepository) ListAll(ctx context.Context, limit, offset int) ([]*user.User, error) {
	sqlcUsers, err := r.queries.ListUsers(ctx, sqlc.ListUsersParams{
		LimitVal:  int32(limit),
		OffsetVal: int32(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list all users: %w", err)
	}

	var users []*user.User
	for _, sqlcUser := range sqlcUsers {
		user, err := toUserEntity(sqlcUser)
		if err != nil {
			return nil, fmt.Errorf("failed to convert sqlcUser to user entity: %w", err)
		}
		users = append(users, user)
	}
	return users, nil
}

func toUserEntity(sqlcUser sqlc.User) (*user.User, error) {
	id, err := pgtypeToUUID(sqlcUser.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	user := &user.User{
		ID:         id,
		TelegramID: *sqlcUser.TelegramID,
		Username:   *sqlcUser.Username,
		FirstName:  *sqlcUser.FirstName,
		LastName:   *sqlcUser.LastName,
		IsActive:   sqlcUser.IsActive,
		CreatedAt:  pgtypeToTime(sqlcUser.CreatedAt),
		BlockedAt:  pgtypeToTime(sqlcUser.BlockedAt),
	}
	return user, nil
}
