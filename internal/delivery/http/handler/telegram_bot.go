package handler

import (
	"context"

	"github.com/VladKovDev/promo-bot/internal/config"
	"github.com/VladKovDev/promo-bot/pkg/logger"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type BotHandler struct {
	bot    *tgbotapi.BotAPI
	cfg    config.Config
	logger logger.Logger
}

func NewBotHandler(bot *tgbotapi.BotAPI, cfg config.Config, logger logger.Logger) *BotHandler {
	return &BotHandler{
		bot:    bot,
		cfg:    cfg,
		logger: logger,
	}
}

// Start begins processing incoming Telegram updates and routes commands.
func (h *BotHandler) Start(ctx context.Context) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := h.bot.GetUpdatesChan(u)

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
					h.handleStart(upd.Message)
				default:
					// unhandled commands can be ignored for now
				}
			}
		}
	}
}

func (h *BotHandler) handleStart(msg *tgbotapi.Message) {
	chatID := msg.Chat.ID
	reply := "start command received"

	m := tgbotapi.NewMessage(chatID, reply)
	if _, err := h.bot.Send(m); err != nil {
		h.logger.Error("failed to send /start reply")
	}
}
