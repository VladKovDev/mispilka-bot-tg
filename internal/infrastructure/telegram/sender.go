package telegram

import (
	"context"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type BotProvider interface {
	Get(botID int64) (*tgbotapi.BotAPI, error)
}

type Sender struct {
	botProvider BotProvider
}

func NewSender(botProvider BotProvider) *Sender {
	return &Sender{botProvider: botProvider}
}

func (s *Sender) SendMessage(ctx context.Context, botID, chatID int64, text string) error {
	bot, err := s.botProvider.Get(botID)
	if err != nil {
		return err
	}

	msg := tgbotapi.NewMessage(chatID, text)
	_, err = bot.Send(msg)
	return err
}
