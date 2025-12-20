package app

import (
	"context"
	"fmt"
	"os"

	"github.com/VladKovDev/promo-bot/internal/config"
	"github.com/VladKovDev/promo-bot/internal/repository/postgres"
	"github.com/VladKovDev/promo-bot/pkg/logger"
)

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
	if err != nil{
		return fmt.Errorf("failed to init database: %w", err)
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
