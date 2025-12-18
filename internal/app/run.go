package app

import (
	"context"
	"fmt"
	"os"

	"github.com/VladKovDev/promo-bot/internal/config"
	"github.com/VladKovDev/promo-bot/pkg/logger"
)

func Run(ctx context.Context) error {
	configPath := os.Getenv("PROMO_BOTS_CONFIG_PATH")
	cfg, err := InitConfig(configPath, ctx)
	if err != nil {
		return fmt.Errorf("failed to init config: %w", err)
	}

	logger, err := InitLogger(cfg)
	if err != nil {
		return fmt.Errorf("failed to init logger: %w", err)
	}
	logger.Debug("logger debug enabled...")
	logger.Info("hello world!")

	return nil
}

func InitConfig(configPath string, ctx context.Context) (*config.Config, error) {
	cfg, err := config.Load(configPath, ctx)
	if err != nil {
		return cfg, err
	}
	return cfg, nil
}

func InitLogger(cfg *config.Config) (logger.Logger, error) {
	return logger.New(cfg.Logger)
}
