package telegram

import (
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"mispilkabot/config"
)

func TestIsGroupMode_GroupMode(t *testing.T) {
	cfg := &config.Config{
		PrivateResourceType: config.ResourceTypeGroup,
	}
	botAPI, _ := tgbotapi.NewBotAPI("test")
	bot := NewBot(botAPI, cfg)

	if !bot.IsGroupMode() {
		t.Error("IsGroupMode() should return true for group mode")
	}
	if bot.IsChannelMode() {
		t.Error("IsChannelMode() should return false for group mode")
	}
}

func TestIsChannelMode_ChannelMode(t *testing.T) {
	cfg := &config.Config{
		PrivateResourceType: config.ResourceTypeChannel,
	}
	botAPI, _ := tgbotapi.NewBotAPI("test")
	bot := NewBot(botAPI, cfg)

	if !bot.IsChannelMode() {
		t.Error("IsChannelMode() should return true for channel mode")
	}
	if bot.IsGroupMode() {
		t.Error("IsGroupMode() should return false for channel mode")
	}
}
