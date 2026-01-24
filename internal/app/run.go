package app

import (
	"context"
	"fmt"
	"os"

	"github.com/VladKovDev/promo-bot/internal/config"
	"github.com/VladKovDev/promo-bot/internal/domain/telegram_bot"
	"github.com/VladKovDev/promo-bot/internal/domain/user"
	"github.com/VladKovDev/promo-bot/internal/infrastructure/crypto"
	"github.com/VladKovDev/promo-bot/internal/infrastructure/repository/postgres"
	"github.com/VladKovDev/promo-bot/internal/infrastructure/telegram"
	"github.com/VladKovDev/promo-bot/internal/registry"
	"github.com/VladKovDev/promo-bot/pkg/logger"
	"go.uber.org/zap"
)

// App holds high-level application dependencies.
type App struct {
	Config              *config.Config
	Logger              logger.Logger
	DB                  *postgres.Pool
	KeyStore            *crypto.KeyStore
	UserRepo            user.Repository
	TelegramBotRepo     telegram_bot.Repository
	TelegramBotService  *telegram_bot.Service
	TelegramBotRegistry *registry.TelegramBotRegistry
}

// NewApp constructs the application object and initializes repositories.
func NewApp(cfg *config.Config,
	pool *postgres.Pool,
	logger logger.Logger,
	keyStore *crypto.KeyStore,
	telegramBotRegistry *registry.TelegramBotRegistry) *App {

	var userRepo user.Repository
	if pool != nil && pool.Pool != nil {
		userRepo = postgres.NewPostgresUserRepository(pool.Pool)
	}
	var telegramBotRepo telegram_bot.Repository
	if pool != nil && pool.Pool != nil {
		telegramBotRepo = postgres.NewPostgresTelegramBotRepository(pool.Pool, keyStore)
	}
	var telegramBotService *telegram_bot.Service
	if telegramBotRepo != nil && telegramBotRegistry != nil {
		botSender := telegram.NewSender(telegramBotRegistry)
		telegramBotService = telegram_bot.NewService(telegramBotRepo, *botSender)
	}

	return &App{
		Config:              cfg,
		Logger:              logger,
		DB:                  pool,
		KeyStore:            keyStore,
		UserRepo:            userRepo,
		TelegramBotRepo:     telegramBotRepo,
		TelegramBotService:  telegramBotService,
		TelegramBotRegistry: telegramBotRegistry,
	}
}

func Run(ctx context.Context) error {
	configPath := os.Getenv("PROMO_BOTS_CONFIG_PATH")
	cfg, err := initConfig(configPath, ctx)
	if err != nil {
		return fmt.Errorf("failed to init config: %w", err)
	}

	logger, err := initLogger(cfg)
	if err != nil {
		return fmt.Errorf("failed to init logger: %w", err)
	}
	logger.Debug("logger debug enabled...")

	pool, err := initPostgresDatabase(ctx, cfg, logger)
	if err != nil {
		return fmt.Errorf("failed to init database: %w", err)
	}

	keyStore, err := initEncryptor(cfg)
	if err != nil {
		return fmt.Errorf("failed to init encryptor: %w", err)
	}

	telegram_bot_registry := registry.NewTelegramBotRegistry()

	app := NewApp(cfg, pool, logger, keyStore, telegram_bot_registry)

	err = app.InitBots(ctx)
	if err != nil {
		return fmt.Errorf("failed to init bots: %w", err)
	}

	gracefulShutdown(ctx, logger, pool)

	return nil
}

func initConfig(configPath string, ctx context.Context) (*config.Config, error) {
	cfg, err := config.Load(configPath, ctx)
	if err != nil {
		return cfg, err
	}
	return cfg, nil
}

func initLogger(cfg *config.Config) (logger.Logger, error) {
	return logger.New(cfg.Logger)
}

func initPostgresDatabase(ctx context.Context, cfg *config.Config, logger logger.Logger) (*postgres.Pool, error) {
	return postgres.NewPool(ctx, &cfg.Database, logger)
}

func initEncryptor(cfg *config.Config) (*crypto.KeyStore, error) {
	return crypto.NewAESKeyStore(cfg.Crypto.CurrentVersion, cfg.Crypto.Keys)
}

func (a *App) InitBots(ctx context.Context) error {
	telegram_bots, err := a.TelegramBotRepo.ListAll(ctx)
	if err != nil {
		return fmt.Errorf("failed to load telegram bots: %w", err)
	}
	for _, bot := range telegram_bots {
		if !bot.IsActive() {
			a.Logger.Info("Skipping inactive Telegram bot",
				zap.String("bot_id", fmt.Sprint(bot.ID)))
			continue
		}

		_, err := a.TelegramBotRegistry.Add(bot.Token)
		if err != nil {
			a.Logger.Error("Failed to initialize Telegram bot",
				zap.String("bot_id", fmt.Sprint(bot.ID)),
				zap.Error(err))
			continue
		}
		a.Logger.Info("Initialized Telegram bot successfully",
			zap.String("bot_id", fmt.Sprint(bot.ID)))
	}
	return nil
}
