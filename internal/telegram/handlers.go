package telegram

import (
	"fmt"
	"log"
	"mispilkabot/internal/services"
	"sort"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	commandStart   = "start"
	commandRestart = "restart"
	commandUsers   = "users"
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
	case commandUsers:
		if err := b.usersCommand(message); err != nil {
			log.Printf("Failed to handle /users command: %v", err)
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

	parsedID, err := parseID(chatID)
	if err != nil {
		return tgbotapi.MessageConfig{}, fmt.Errorf("failed to parse chatID %s: %w", chatID, err)
	}

	msg := tgbotapi.NewMessage(parsedID, text)
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

// isAdmin checks if the given user ID is in the admin list
func (b *Bot) isAdmin(userID int64) bool {
	for _, adminID := range b.cfg.AdminIDs {
		if adminID == userID {
			return true
		}
	}
	return false
}

// usersCommand sends a formatted table of all users (admin only)
func (b *Bot) usersCommand(message *tgbotapi.Message) error {
	// Check if user is admin - silently ignore if not (don't reveal command exists)
	if !b.isAdmin(message.From.ID) {
		return nil
	}

	// Get all users
	users, err := services.GetAllUsers()
	if err != nil {
		return fmt.Errorf("failed to get users: %w", err)
	}

	if len(users) == 0 {
		msg := tgbotapi.NewMessage(message.Chat.ID, "📭 Пользователей пока нет.")
		if _, err := b.bot.Send(msg); err != nil {
			return fmt.Errorf("failed to send empty users message: %w", err)
		}
		return nil
	}

	// Sort users by registration time (newest first)
	type userEntry struct {
		chatID string
		user   services.User
	}
	var sortedUsers []userEntry
	for chatID, user := range users {
		sortedUsers = append(sortedUsers, userEntry{chatID, user})
	}
	sort.Slice(sortedUsers, func(i, j int) bool {
		return sortedUsers[i].user.RegTime.After(sortedUsers[j].user.RegTime)
	})

	// Build formatted table
	var sb strings.Builder
	sb.WriteString("📊 <b>Список пользователей</b>\n\n")
	sb.WriteString(fmt.Sprintf("Всего: <b>%d</b> пользователей\n\n", len(users)))

	// Table header
	sb.WriteString("<code>")
	sb.WriteString(fmt.Sprintf("%-15s │ %-18s │ %-7s │ %-7s │ %-6s\n", "ID", "Регистрация", "Принял", "Оплатил", "Группа"))
	sb.WriteString("─────────────────┼────────────────────┼─────────┼─────────┼────────\n")

	// Table rows
	for _, entry := range sortedUsers {
		chatID := entry.chatID
		user := entry.user

		// Truncate chatID for display
		displayID := chatID
		if len(displayID) > 15 {
			displayID = displayID[:12] + "..."
		}

		// Format registration date
		regDate := user.RegTime.Format("02.01 15:04")

		// Status indicators
		accepted := "❌"
		if user.IsMessaging {
			accepted = "✅"
		}

		paid := "❌"
		if user.HasPaid() {
			paid = "✅"
		}

		group := "❌"
		if user.HasJoined() {
			group = "✅"
		}

		sb.WriteString(fmt.Sprintf("%-15s │ %-18s │ %-7s │ %-7s │ %-6s\n", displayID, regDate, accepted, paid, group))
	}

	sb.WriteString("</code>")

	msg := tgbotapi.NewMessage(message.Chat.ID, sb.String())
	msg.ParseMode = "HTML"

	if _, err := b.bot.Send(msg); err != nil {
		return fmt.Errorf("failed to send users table: %w", err)
	}

	log.Printf("Admin %d (%s) requested users table, %d users shown",
		message.From.ID, message.From.UserName, len(users))

	return nil
}
