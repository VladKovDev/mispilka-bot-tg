package bot

import (
	"fmt"
	"log"
	"mispilkabot/internal/services"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Bot struct {
	bot *tgbotapi.BotAPI
}

func NewBot(bot *tgbotapi.BotAPI) *Bot {
	return &Bot{bot: bot}
}

func (b *Bot) Start() {
	log.Printf("Authorized on account %s", b.bot.Self.UserName)

	err := services.SetSchedules(SendMessage)
	if err != nil {
		fmt.Println("BUUUUUU")
	}

	b.handleUpdates(b.initUpdatesChanel())
}

func (b *Bot) initUpdatesChanel() tgbotapi.UpdatesChannel {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	return b.bot.GetUpdatesChan(u)
}

func (b *Bot) handleUpdates(updates tgbotapi.UpdatesChannel) {
	for update := range updates {
		if update.Message == nil {
			continue
		}
		if update.Message.IsCommand() {
			b.handleCommand(update.Message)
		}
	}
}

func (b *Bot) handleCommand(message *tgbotapi.Message) {
	msg := tgbotapi.NewMessage(message.Chat.ID, "")

	switch message.Command() {
	case "start":
		services.AddPerson(message)
		msg.Text = "start command"
	case "help":
		msg.Text = "help command"
	default:
		msg.Text = "I don't know that command"
	}

	if _, err := b.bot.Send(msg); err != nil {
		log.Panic(err)
	}
}

func SendMessage(chatID string) {
	fmt.Println(chatID)
}
