package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/VladKovDev/promo-bot/internal/domain/telegram_bot"
	"github.com/VladKovDev/promo-bot/internal/infrastructure/crypto"
	"github.com/VladKovDev/promo-bot/internal/infrastructure/repository/postgres/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
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
		Role:              bot.Role,
		LastError:         nil,
		LastCheckedAt:     timeToPgtype(time.Time{}),
		RevokedAt:         timePtrToPgtype(bot.RevokedAt),
		DisabledAt:        timePtrToPgtype(bot.DisabledAt),
	}

	created, err := r.queries.CreateTelegramBot(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to create telegram bot: %w", err)
	}

	createdBot, err := telegramBotFromRow(r, created)
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
	return telegramBotFromRow(r, tb)
}

func (r *PostgresTelegramBotRepository) GetByTelegramID(ctx context.Context, telegramID int64) (*telegram_bot.TelegramBot, error) {
	tb, err := r.queries.GetTelegramBotByBotID(ctx, &telegramID)
	if err != nil {
		return nil, fmt.Errorf("failed to get telegram bot by telegram id: %w", err)
	}
	return telegramBotFromRow(r, tb)
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
		Role:              bot.Role,
		LastError:         nil,
		LastCheckedAt:     timeToPgtype(time.Time{}),
		RevokedAt:         timePtrToPgtype(bot.RevokedAt),
		DisabledAt:        timePtrToPgtype(bot.DisabledAt),
	}

	updated, err := r.queries.UpdateTelegramBot(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to update telegram bot: %w", err)
	}

	updatedBot, err := telegramBotFromRow(r, updated)
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
		b, err := telegramBotFromRow(r, it)
		if err != nil {
			return nil, fmt.Errorf("failed to convert telegram bot: %w", err)
		}
		bots = append(bots, b)
	}
	return bots, nil
}

func telegramBotFromRow[T sqlc.TelegramBot | sqlc.CreateTelegramBotRow | sqlc.UpdateTelegramBotRow | sqlc.ListTelegramBotsRow | sqlc.GetTelegramBotByBotIDRow | sqlc.GetTelegramBotByIDRow](
	r *PostgresTelegramBotRepository,
	 row T,
	 ) (*telegram_bot.TelegramBot, error) {
	var (
		id pgtype.UUID
		botID *int64
		username string
		firstName *string
		lastName *string
		encryptedToken []byte
		encryptionVersion int32
		role string
		revokedAt pgtype.Timestamp
		disabledAt pgtype.Timestamp
	)
	switch v := any(row).(type) {
	case sqlc.TelegramBot:
		id = v.ID
		botID = v.BotID
		username = v.Username
		firstName = v.FirstName
		lastName = v.LastName
		encryptedToken = v.EncryptedToken
		encryptionVersion = v.EncryptionVersion
		role = v.Role
		revokedAt = v.RevokedAt
		disabledAt = v.DisabledAt
	case sqlc.CreateTelegramBotRow:
		id = v.ID
		botID = v.BotID
		username = v.Username
		firstName = v.FirstName
		lastName = v.LastName
		encryptedToken = v.EncryptedToken
		encryptionVersion = v.EncryptionVersion
		role = v.Role
		revokedAt = v.RevokedAt
		disabledAt = v.DisabledAt
	case sqlc.UpdateTelegramBotRow:
		id = v.ID
		botID = v.BotID
		username = v.Username
		firstName = v.FirstName
		lastName = v.LastName
		encryptedToken = v.EncryptedToken
		encryptionVersion = v.EncryptionVersion
		role = v.Role
		revokedAt = v.RevokedAt
		disabledAt = v.DisabledAt
	case sqlc.ListTelegramBotsRow:
		id = v.ID
		botID = v.BotID
		username = v.Username
		firstName = v.FirstName
		lastName = v.LastName
		encryptedToken = v.EncryptedToken
		encryptionVersion = v.EncryptionVersion
		role = v.Role
		revokedAt = v.RevokedAt
		disabledAt = v.DisabledAt
	case sqlc.GetTelegramBotByBotIDRow:
		id = v.ID
		botID = v.BotID
		username = v.Username
		firstName = v.FirstName
		lastName = v.LastName
		encryptedToken = v.EncryptedToken
		encryptionVersion = v.EncryptionVersion
		role = v.Role
		revokedAt = v.RevokedAt
		disabledAt = v.DisabledAt
	case sqlc.GetTelegramBotByIDRow:
		id = v.ID
		botID = v.BotID
		username = v.Username
		firstName = v.FirstName
		lastName = v.LastName
		encryptedToken = v.EncryptedToken
		encryptionVersion = v.EncryptionVersion
		role = v.Role
		revokedAt = v.RevokedAt
		disabledAt = v.DisabledAt
	default:
		return nil, fmt.Errorf("unsupported row type")
	}

	domainId, err := pgtypeToUUID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to convert telegram bot ID: %w", err)
	}

	var tokenStr string
	if len(encryptedToken) > 0 {
		enc, ok := r.keyStore.Encryptors[int(encryptionVersion)]
		if !ok {
			return nil, fmt.Errorf("unknown encryption version: %d", encryptionVersion)
		}
		token, err := enc.Decrypt(encryptedToken)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt telegram bot token: %w", err)
		}
		tokenStr = string(token)
	}

	domainRevokedAt := pgtypeToTime(revokedAt)
	domainDisabledAt := pgtypeToTime(disabledAt)

	bot := &telegram_bot.TelegramBot{
		ID:        domainId,
		BotID:      pgtypeToInt64(botID),
		Token:      tokenStr,
		Username:   username,
		FirstName:  pgtypeToString(firstName),
		LastName:   pgtypeToString(lastName),
		Role:       role,
		RevokedAt:  &domainRevokedAt,
		DisabledAt: &domainDisabledAt,
	}
	return bot, nil
}
