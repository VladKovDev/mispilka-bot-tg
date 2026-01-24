package telegram_bot

import (
	"context"

	"github.com/VladKovDev/promo-bot/internal/infrastructure/telegram"
)

type Service struct {
	repo   Repository
	sender telegram.Sender
}

func NewService(repo Repository, sender telegram.Sender) *Service {
	return &Service{repo: repo, sender: sender}
}

func (s *Service) HandleStart(ctx context.Context, botID, ChatID int64) error {
	return s.sender.SendMessage(ctx, botID, ChatID, "Welcome to the bot!")
}