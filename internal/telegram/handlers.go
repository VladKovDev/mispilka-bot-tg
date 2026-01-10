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
		if err := b.startCommand(message); err != nil {
			log.Printf("Failed to handle /start command: %v", err)
		}
	case commandRestart:
		if err := services.AddUser(message); err != nil {
			log.Printf("Failed to add user: %v", err)
		}
	}
}

func (b *Bot) startCommand(message *tgbotapi.Message) error {
	chatID := fmt.Sprint(message.Chat.ID)

	// Add user if new
	isNew, err := services.IsNewUser(chatID)
	if err != nil {
		return fmt.Errorf("failed to check if user is new: %w", err)
	}
	if isNew {
		if err := services.AddUser(message); err != nil {
			return fmt.Errorf("failed to add new user: %w", err)
		}
	}

	// Build and send start message
	msg, err := b.buildStartMessage(chatID)
	if err != nil {
		return fmt.Errorf("failed to build start message: %w", err)
	}

	if _, err := b.bot.Send(msg); err != nil {
		return fmt.Errorf("failed to send start message: %w", err)
	}

	return nil
}

// buildStartMessage creates the start command message with appropriate keyboard
func (b *Bot) buildStartMessage(chatID string) (tgbotapi.MessageConfig, error) {
	text, err := services.GetMessageText(commandStart)
	if err != nil {
		return tgbotapi.MessageConfig{}, fmt.Errorf("failed to get message text: %w", err)
	}

	msg := tgbotapi.NewMessage(parseID(chatID), text)
	msg.ParseMode = "HTML"
	msg.DisableWebPagePreview = true

	// Set keyboard based on user's messaging status
	userData, err := services.GetUser(chatID)
	if err == nil {
		if userData.IsMessaging {
			msg.ReplyMarkup = dataButton("✅ Принято", "decline")
		} else {
			msg.ReplyMarkup = dataButton("🔲 Принимаю", "accept")
		}
	} else {
		// Default keyboard for new users
		msg.ReplyMarkup = dataButton("🔲 Принимаю", "accept")
	}

	return msg, nil
}
