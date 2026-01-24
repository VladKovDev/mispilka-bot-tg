package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/VladKovDev/promo-bot/internal/domain/telegram_bot"
	"github.com/VladKovDev/promo-bot/internal/infrastructure/crypto"
	"github.com/VladKovDev/promo-bot/internal/infrastructure/repository/postgres/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresTelegramBotRepository struct {
	queries  *sqlc.Queries
	keyStore *crypto.KeyStore
}

func NewPostgresTelegramBotRepository(db *pgxpool.Pool, keyStore *crypto.KeyStore) telegram_bot.Repository {
	return &PostgresTelegramBotRepository{
		queries:  sqlc.New(db),
		keyStore: keyStore,
	}
}

func (r *PostgresTelegramBotRepository) Create(ctx context.Context, bot *telegram_bot.TelegramBot) error {
	var botID *int64
	if bot.BotID != 0 {
		botID = &bot.BotID
	}

	var firstName *string
	if bot.FirstName != "" {
		firstName = &bot.FirstName
	}

	var lastName *string
	if bot.LastName != "" {
		lastName = &bot.LastName
	}

	enc := r.keyStore.Encryptors[r.keyStore.Current]
	encryptedToken, err := enc.Encrypt([]byte(bot.Token))
	if err != nil {
		return fmt.Errorf("failed to encrypt token: %w", err)
	}

	params := sqlc.CreateTelegramBotParams{
		BotID:             botID,
		Username:          bot.Username,
		FirstName:         firstName,
		LastName:          lastName,
		EncryptedToken:    encryptedToken,
		EncryptionVersion: int32(r.keyStore.Current),
		LastError:         nil,
		LastCheckedAt:     timeToPgtype(time.Time{}),
		RevokedAt:         timeToPgtype(bot.RevokedAt),
		DisabledAt:        timeToPgtype(bot.DisabledAt),
	}

	created, err := r.queries.CreateTelegramBot(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to create telegram bot: %w", err)
	}

	createdBot, err := r.toDomain(created)
	if err != nil {
		return fmt.Errorf("failed to map created telegram bot: %w", err)
	}
	*bot = *createdBot
	return nil
}

func (r *PostgresTelegramBotRepository) GetByID(ctx context.Context, id uuid.UUID) (*telegram_bot.TelegramBot, error) {
	tb, err := r.queries.GetTelegramBotByID(ctx, uuidToPgtype(id))
	if err != nil {
		return nil, fmt.Errorf("failed to get telegram bot by id: %w", err)
	}
	return r.toDomain(tb)
}

func (r *PostgresTelegramBotRepository) GetByTelegramID(ctx context.Context, telegramID int64) (*telegram_bot.TelegramBot, error) {
	tb, err := r.queries.GetTelegramBotByBotID(ctx, &telegramID)
	if err != nil {
		return nil, fmt.Errorf("failed to get telegram bot by telegram id: %w", err)
	}
	return r.toDomain(tb)
}

func (r *PostgresTelegramBotRepository) Update(ctx context.Context, bot *telegram_bot.TelegramBot) error {
	var botID *int64
	if bot.BotID != 0 {
		botID = &bot.BotID
	}

	var firstName *string
	if bot.FirstName != "" {
		firstName = &bot.FirstName
	}

	var lastName *string
	if bot.LastName != "" {
		lastName = &bot.LastName
	}

	enc := r.keyStore.Encryptors[r.keyStore.Current]
	encryptedToken, err := enc.Encrypt([]byte(bot.Token))
	if err != nil {
		return fmt.Errorf("failed to encrypt token: %w", err)
	}

	params := sqlc.UpdateTelegramBotParams{
		BotID:             botID,
		Username:          bot.Username,
		FirstName:         firstName,
		LastName:          lastName,
		EncryptedToken:    encryptedToken,
		EncryptionVersion: int32(r.keyStore.Current),
		LastError:         nil,
		LastCheckedAt:     timeToPgtype(time.Time{}),
		RevokedAt:         timeToPgtype(bot.RevokedAt),
		DisabledAt:        timeToPgtype(bot.DisabledAt),
	}

	updated, err := r.queries.UpdateTelegramBot(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to update telegram bot: %w", err)
	}

	updatedBot, err := r.toDomain(updated)
	if err != nil {
		return fmt.Errorf("failed to map updated telegram bot: %w", err)
	}
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
		b, err := r.toDomain(it)
		if err != nil {
			return nil, fmt.Errorf("failed to convert telegram bot: %w", err)
		}
		bots = append(bots, b)
	}
	return bots, nil
}

func (r *PostgresTelegramBotRepository) toDomain(tb sqlc.TelegramBot) (*telegram_bot.TelegramBot, error) {
	id, err := pgtypeToUUID(tb.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to convert telegram bot ID: %w", err)
	}

	var tokenStr string
	if len(tb.EncryptedToken) > 0 {
		enc, ok := r.keyStore.Encryptors[int(tb.EncryptionVersion)]
		if !ok {
			return nil, fmt.Errorf("unknown encryption version: %d", tb.EncryptionVersion)
		}
		token, err := enc.Decrypt(tb.EncryptedToken)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt telegram bot token: %w", err)
		}
		tokenStr = string(token)
	}

	bot := &telegram_bot.TelegramBot{
		ID:         id,
		BotID:      pgtypeToInt64(tb.BotID),
		Token:      tokenStr,
		Username:   tb.Username,
		FirstName:  pgtypeToString(tb.FirstName),
		LastName:   pgtypeToString(tb.LastName),
		RevokedAt:  pgtypeToTime(tb.RevokedAt),
		DisabledAt: pgtypeToTime(tb.DisabledAt),
	}
	return bot, nil
}
