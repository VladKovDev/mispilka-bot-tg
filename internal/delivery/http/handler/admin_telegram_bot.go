package handler

import (
	"context"

	"github.com/VladKovDev/promo-bot/internal/config"
	"github.com/VladKovDev/promo-bot/pkg/logger"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type AdminBotHandler struct {
	bot    *tgbotapi.BotAPI
	cfg    config.Config
	logger logger.Logger
}

func NewAdminBotHandler(bot *tgbotapi.BotAPI, cfg config.Config, logger logger.Logger) *BotHandler {
	return &BotHandler{
		bot:    bot,
		cfg:    cfg,
		logger: logger,
	}
}

// Start begins processing incoming Telegram updates and routes commands.
func (a *AdminBotHandler) Start(ctx context.Context) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := a.bot.GetUpdatesChan(u)

	for {
		select {
		case <-ctx.Done():
			return
		case upd, ok := <-updates:
			if !ok {
				return
			}

			if upd.Message == nil {
				continue
			}

			if upd.Message.IsCommand() {
				switch upd.Message.Command() {
				case "start":
					a.handleStart(upd.Message)
				case "new_bot":
					a.handleNewBot(upd.Message)
				default:
					// unhandled commands can be ignored for now
				}
			}
		}
	}
}

func (a *AdminBotHandler) handleStart(msg *tgbotapi.Message) {
	chatID := msg.Chat.ID
	reply := "start command received by admin"

	m := tgbotapi.NewMessage(chatID, reply)
	if _, err := a.bot.Send(m); err != nil {
		a.logger.Error("failed to send /start reply")
	}
}

func (a *AdminBotHandler) handleNewBot(msg *tgbotapi.Message) {
}