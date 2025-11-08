package telegram

import (
	"fmt"
	"log"
	"mispilkabot/internal/services"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	commandStart   = "start"
	commandRestart = "restart"
)

func (b *Bot) handleCommand(message *tgbotapi.Message) {
	lowerCommand := strings.ToLower(message.Command())
	switch lowerCommand {
	case commandStart:
		b.startCommand(message)
	case commandRestart:
		if err := services.AddUser(message); err != nil {
			log.Printf("Failed to add user: %v", err)
		}
	}
}

func (b *Bot) startCommand(message *tgbotapi.Message)error{
	if services.IsNewUser(fmt.Sprint(message.Chat.ID)) {
		err := services.AddUser(message)
		if err != nil {
			return err
		}
	}

	text, err := services.GetMessageText(commandStart)
	if err != nil {
		return fmt.Errorf("failed to get message text: %w", err)
	}
	msg := tgbotapi.NewMessage(message.Chat.ID, text)

	userData, err := services.GetUser(fmt.Sprint(message.Chat.ID))
	if err == nil {
		if userData.IsMessaging {
			msg.ReplyMarkup = dataButton("✅ Принято", "decline")
		} else {
			msg.ReplyMarkup = dataButton("🔲 Принимаю", "accept")
		}
	} else {
		msg.ReplyMarkup = dataButton("🔲 Принимаю", "accept")
	}
	msg.ParseMode = "HTML"
	if _, err := b.bot.Send(msg); err != nil {
		return err
	}
	return nil
}
