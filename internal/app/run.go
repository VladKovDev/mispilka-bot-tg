package app

import (
	"context"
	"fmt"
	"os"

	"github.com/VladKovDev/promo-bot/internal/config"
	"github.com/VladKovDev/promo-bot/internal/crypto"
	"github.com/VladKovDev/promo-bot/internal/domain/repository"
	"github.com/VladKovDev/promo-bot/internal/repository/postgres"
	"github.com/VladKovDev/promo-bot/pkg/logger"
)

// App holds high-level application dependencies.
type App struct {
	Config          *config.Config
	Logger          logger.Logger
	DB              *postgres.Pool
	KeyStore        *crypto.KeyStore
	UserRepo        repository.UserRepository
	TelegramBotRepo repository.TelegramBotRepository
}

// NewApp constructs the application object and initializes repositories.
func NewApp(cfg *config.Config, pool *postgres.Pool, logger logger.Logger, keyStore *crypto.KeyStore) *App {
	var userRepo repository.UserRepository
	if pool != nil && pool.Pool != nil {
		userRepo = postgres.NewPostgresUserRepository(pool.Pool)
	}
	var telegramBotRepo repository.TelegramBotRepository
	if pool != nil && pool.Pool != nil {
		telegramBotRepo = postgres.NewPostgresTelegramBotRepository(pool.Pool)
	}

	return &App{
		Config:          cfg,
		Logger:          logger,
		DB:              pool,
		KeyStore:        keyStore,
		UserRepo:        userRepo,
		TelegramBotRepo: telegramBotRepo,
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

	app := NewApp(cfg, pool, logger, keyStore)

	if err := app.StartBots(ctx, *cfg, logger); err != nil {
		return fmt.Errorf("failed to start bots: %w", err)
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
