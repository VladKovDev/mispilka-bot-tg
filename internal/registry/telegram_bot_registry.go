package registry

import (
	"fmt"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type TelegramBotRegistry struct {
	mu   sync.RWMutex
	bots map[int64]*tgbotapi.BotAPI
}

func NewTelegramBotRegistry() *TelegramBotRegistry {
	return &TelegramBotRegistry{
		bots: make(map[int64]*tgbotapi.BotAPI),
	}
}

func (r *TelegramBotRegistry) Add(token string) (*tgbotapi.BotAPI, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}

	r.bots[bot.Self.ID] = bot
	return bot, nil
}

func (r *TelegramBotRegistry) Get(botID int64) (*tgbotapi.BotAPI, error){
	r.mu.RLock()
	defer r.mu.RUnlock()
	bot, ok := r.bots[botID]
	if !ok {
		return nil, fmt.Errorf("bot with ID %d not found", botID)
	}
	return bot, nil
}
