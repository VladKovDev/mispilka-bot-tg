package app

import (
	"context"
	"fmt"

	"github.com/VladKovDev/promo-bot/internal/config"
	"github.com/VladKovDev/promo-bot/internal/delivery/http/handler"
	"github.com/VladKovDev/promo-bot/pkg/logger"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"
)

// StartBots initializes and starts all bot instances based on the provided configuration.
func (a App) StartBots(ctx context.Context, cfg config.Config, logger logger.Logger) error {
	bots, err := a.TelegramBotRepo.ListAll(ctx)
	if err != nil {
		return fmt.Errorf("failed to list telegram bots: %w", err)
	}
	if bots == nil {
		logger.Warn("no bots found in the repository")
		return nil
	}

	for _, botEntity := range bots {
		encryptor, ok := a.KeyStore.Encryptors[botEntity.EncryptionVersion]
		if !ok {
			logger.Error("no encryptor found for bot",
				zap.String("name", botEntity.Username),
				zap.Int("encryption_version", botEntity.EncryptionVersion))
			continue
		}
		tokenBytes, err := encryptor.Decrypt(botEntity.EncryptedToken)
		if err != nil {
			logger.Error("failed to decrypt token for bot",
				zap.String("name", botEntity.Username),
				zap.Error(err))
			continue
		}

		token := string(tokenBytes)
		for i := range tokenBytes {
			tokenBytes[i] = 0
		}

		if err := startBot(ctx, cfg, logger, token); err != nil {
			logger.Error("failed to start bot",
				zap.String("name", botEntity.Username),
				zap.Error(err))
			continue
		}

		logger.Info("bot started successfully", zap.String("name", botEntity.Username))
	}

	return nil
}

// start single bot instance
func startBot(ctx context.Context, cfg config.Config, logger logger.Logger, token string) error {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return fmt.Errorf("failed to create new bot API: %w", err)
	}

	bot.Debug = cfg.Logger.Level == "debug"
	logger.Info("Authorized on account",
		zap.String("account", bot.Self.UserName))

	botHandler := handler.NewBotHandler(bot, cfg, logger)

	go botHandler.Start(ctx)

	return nil
}
