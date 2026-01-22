package postgres

import (
	"context"
	"fmt"

	"github.com/VladKovDev/promo-bot/internal/domain/telegram_bot"
	"github.com/VladKovDev/promo-bot/internal/infrastructure/repository/postgres/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresTelegramBotRepository struct {
	queries *sqlc.Queries
}

func NewPostgresTelegramBotRepository(db *pgxpool.Pool) telegram_bot.Repository {
	return &PostgresTelegramBotRepository{
		queries: sqlc.New(db),
	}
}

func (r *PostgresTelegramBotRepository) Create(ctx context.Context, bot *telegram_bot.TelegramBot) error {
	if err := bot.Validate(); err != nil {
		return fmt.Errorf("invalid telegram bot: %w", err)
	}

	var firstName *string
	if bot.FirstName != "" {
		firstName = &bot.FirstName
	}

	var lastError *string
	if bot.LastError != "" {
		lastError = &bot.LastError
	}

	params := sqlc.CreateTelegramBotParams{
		BotID:             &bot.BotID,
		Username:          bot.Username,
		FirstName:         firstName,
		EncryptedToken:    bot.EncryptedToken,
		EncryptionVersion: int32(bot.EncryptionVersion),
		Status:            bot.Status,
		LastError:         lastError,
		LastCheckedAt:     timeToPgtype(bot.LastCheckedAt),
		RevokedAt:         timeToPgtype(bot.RevokedAt),
	}

	created, err := r.queries.CreateTelegramBot(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to create telegram bot: %w", err)
	}

	createdBot, _ := toTelegramBotEntity(created)
	*bot = *createdBot
	return nil
}

func (r *PostgresTelegramBotRepository) GetByID(ctx context.Context, id uuid.UUID) (*telegram_bot.TelegramBot, error) {
	tb, err := r.queries.GetTelegramBotByID(ctx, uuidToPgtype(id))
	if err != nil {
		return nil, fmt.Errorf("failed to get telegram bot by id: %w", err)
	}
	return toTelegramBotEntity(tb)
}

func (r *PostgresTelegramBotRepository) GetByTelegramID(ctx context.Context, telegramID int64) (*telegram_bot.TelegramBot, error) {
	tb, err := r.queries.GetTelegramBotByBotID(ctx, &telegramID)
	if err != nil {
		return nil, fmt.Errorf("failed to get telegram bot by telegram id: %w", err)
	}
	return toTelegramBotEntity(tb)
}

func (r *PostgresTelegramBotRepository) Update(ctx context.Context, bot *telegram_bot.TelegramBot) error {
	if err := bot.Validate(); err != nil {
		return fmt.Errorf("invalid telegram bot: %w", err)
	}

	var firstName *string
	if bot.FirstName != "" {
		firstName = &bot.FirstName
	}

	var lastError *string
	if bot.LastError != "" {
		lastError = &bot.LastError
	}

	params := sqlc.UpdateTelegramBotParams{
		BotID:             &bot.BotID,
		Username:          bot.Username,
		FirstName:         firstName,
		EncryptedToken:    bot.EncryptedToken,
		EncryptionVersion: int32(bot.EncryptionVersion),
		Status:            bot.Status,
		LastError:         lastError,
		LastCheckedAt:     timeToPgtype(bot.LastCheckedAt),
		RevokedAt:         timeToPgtype(bot.RevokedAt),
	}

	updated, err := r.queries.UpdateTelegramBot(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to update telegram bot: %w", err)
	}

	updatedBot, _ := toTelegramBotEntity(updated)
	*bot = *updatedBot
	return nil
}

func (r *PostgresTelegramBotRepository) Delete(ctx context.Context, id uuid.UUID) error {
	ID := uuidToPgtype(id)
	if err := r.queries.DeleteTelegramBot(ctx, ID); err != nil {
		return fmt.Errorf("failed to delete telegram bot: %w", err)
	}
	return nil
}

func (r *PostgresTelegramBotRepository) ListAll(ctx context.Context) ([]*telegram_bot.TelegramBot, error) {
	items, err := r.queries.ListTelegramBots(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list telegram bots: %w", err)
	}
	var bots []*telegram_bot.TelegramBot
	for _, it := range items {
		b, err := toTelegramBotEntity(it)
		if err != nil {
			return nil, fmt.Errorf("failed to convert telegram bot: %w", err)
		}
		bots = append(bots, b)
	}
	return bots, nil
}

func toTelegramBotEntity(tb sqlc.TelegramBot) (*telegram_bot.TelegramBot, error) {
	var botID int64
	if tb.BotID != nil {
		botID = *tb.BotID
	}

	var firstName string
	if tb.FirstName != nil {
		firstName = *tb.FirstName
	}

	var lastError string
	if tb.LastError != nil {
		lastError = *tb.LastError
	}

	return &telegram_bot.TelegramBot{
		ID:                0,
		EncryptedToken:    tb.EncryptedToken,
		EncryptionVersion: int(tb.EncryptionVersion),
		Username:          tb.Username,
		FirstName:         firstName,
		BotID:             botID,
		OwnerID:           0,
		Status:            tb.Status,
		LastError:         lastError,
		LastCheckedAt:     pgtypeToTime(tb.LastCheckedAt),
		RevokedAt:         pgtypeToTime(tb.RevokedAt),
		CreatedAt:         pgtypeToTime(tb.CreatedAt),
		UpdatedAt:         pgtypeToTime(tb.UpdatedAt),
	}, nil
}
