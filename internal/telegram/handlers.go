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
	usersPerPage   = 5
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

// userEntry represents a user with their chat ID for display
type userEntry struct {
	chatID string
	user   services.User
}

// usersCommand sends paginated list of users (admin only)
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
	sortedUsers := b.sortUsers(users)

	// Send first page
	return b.sendUsersPage(message.Chat.ID, sortedUsers, 0)
}

// sortUsers sorts users by registration time (newest first)
func (b *Bot) sortUsers(users services.UserMap) []userEntry {
	var sortedUsers []userEntry
	for chatID, user := range users {
		sortedUsers = append(sortedUsers, userEntry{chatID, user})
	}
	sort.Slice(sortedUsers, func(i, j int) bool {
		return sortedUsers[i].user.RegTime.After(sortedUsers[j].user.RegTime)
	})
	return sortedUsers
}

// formatUser formats a single user entry for display
func (b *Bot) formatUser(entry userEntry, index int) string {
	user := entry.user
	chatID := entry.chatID

	// Display name or ID
	displayName := user.UserName
	if displayName == "" {
		displayName = chatID
	}

	var sb strings.Builder

	// Header with index and name
	sb.WriteString(fmt.Sprintf("<b>%d. %s</b>\n", index+1, displayName))

	// Registration info
	sb.WriteString(fmt.Sprintf("📅 Регистрация: %s\n", user.RegTime.Format("02.01.2006 15:04")))

	// Status indicators
	sb.WriteString("📊 Статус: ")
	if user.IsMessaging {
		sb.WriteString("✅ Принял условия")
	} else {
		sb.WriteString("⏳ Не принял условия")
	}
	sb.WriteString("\n")

	// Payment info
	if user.HasPaid() {
		sb.WriteString(fmt.Sprintf("💳 Оплата: ✅ %s\n", user.GetPaymentDate().Format("02.01.2006 15:04")))
		if user.PaymentLink != "" {
			sb.WriteString(fmt.Sprintf("   Ссылка: %s\n", user.PaymentLink))
		}
	} else {
		sb.WriteString("💳 Оплата: ❌ Не оплачено\n")
	}

	// Group info
	if user.HasJoined() {
		sb.WriteString(fmt.Sprintf("👥 Группа: ✅ Вступил %s\n", user.GetJoinedAt().Format("02.01.2006 15:04")))
		if user.InviteLink != "" {
			sb.WriteString(fmt.Sprintf("   Инвайт-ссылка использована\n"))
		}
	} else {
		sb.WriteString("👥 Группа: ❌ Не вступил\n")
	}

	// Messages queue info
	if len(user.MessagesList) > 0 {
		sb.WriteString(fmt.Sprintf("📨 В очереди: %d сообщений\n", len(user.MessagesList)))
	} else {
		sb.WriteString("📨 В очереди: 0 сообщений\n")
	}

	// Technical info (collapsed)
	sb.WriteString(fmt.Sprintf("🔑 ID: <code>%s</code>\n", chatID))

	return sb.String()
}

// sendUsersPage sends a single page of users
func (b *Bot) sendUsersPage(chatID int64, sortedUsers []userEntry, page int) error {
	totalUsers := len(sortedUsers)
	totalPages := (totalUsers + usersPerPage - 1) / usersPerPage

	// Ensure page is in valid range
	if page < 0 {
		page = 0
	}
	if page >= totalPages {
		page = totalPages - 1
	}

	// Calculate slice bounds
	startIdx := page * usersPerPage
	endIdx := startIdx + usersPerPage
	if endIdx > totalUsers {
		endIdx = totalUsers
	}

	// Build message
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📊 <b>Пользователи</b> (стр. %d/%d, всего %d)\n\n",
		page+1, totalPages, totalUsers))

	// Add users for this page
	for i := startIdx; i < endIdx; i++ {
		sb.WriteString(b.formatUser(sortedUsers[i], i))
		if i < endIdx-1 {
			sb.WriteString("\n────────────────\n\n")
		}
	}

	msg := tgbotapi.NewMessage(chatID, sb.String())
	msg.ParseMode = "HTML"
	msg.DisableWebPagePreview = true

	// Add pagination keyboard
	keyboard := b.buildUsersKeyboard(page, totalPages)
	msg.ReplyMarkup = &keyboard

	if _, err := b.bot.Send(msg); err != nil {
		return fmt.Errorf("failed to send users page: %w", err)
	}

	return nil
}

// sendUsersPageEdit edits existing message with new page
func (b *Bot) sendUsersPageEdit(messageID int, chatID int64, sortedUsers []userEntry, page int) error {
	totalUsers := len(sortedUsers)
	totalPages := (totalUsers + usersPerPage - 1) / usersPerPage

	// Ensure page is in valid range
	if page < 0 {
		page = 0
	}
	if page >= totalPages {
		page = totalPages - 1
	}

	// Calculate slice bounds
	startIdx := page * usersPerPage
	endIdx := startIdx + usersPerPage
	if endIdx > totalUsers {
		endIdx = totalUsers
	}

	// Build message
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📊 <b>Пользователи</b> (стр. %d/%d, всего %d)\n\n",
		page+1, totalPages, totalUsers))

	// Add users for this page
	for i := startIdx; i < endIdx; i++ {
		sb.WriteString(b.formatUser(sortedUsers[i], i))
		if i < endIdx-1 {
			sb.WriteString("\n────────────────\n\n")
		}
	}

	edit := tgbotapi.NewEditMessageText(chatID, messageID, sb.String())
	edit.ParseMode = "HTML"
	edit.DisableWebPagePreview = true

	// Add pagination keyboard
	keyboard := b.buildUsersKeyboard(page, totalPages)
	edit.ReplyMarkup = &keyboard

	if _, err := b.bot.Send(edit); err != nil {
		return fmt.Errorf("failed to edit users page: %w", err)
	}

	return nil
}

// buildUsersKeyboard creates pagination keyboard
func (b *Bot) buildUsersKeyboard(page, totalPages int) tgbotapi.InlineKeyboardMarkup {
	if totalPages <= 1 {
		return tgbotapi.InlineKeyboardMarkup{}
	}

	var rows [][]tgbotapi.InlineKeyboardButton

	// Navigation row
	var navRow []tgbotapi.InlineKeyboardButton

	// First page button
	if page > 1 {
		btn := tgbotapi.NewInlineKeyboardButtonData("⏮", "users_page_0")
		navRow = append(navRow, btn)
	}

	// Previous button
	if page > 0 {
		btn := tgbotapi.NewInlineKeyboardButtonData("◀️", fmt.Sprintf("users_page_%d", page-1))
		navRow = append(navRow, btn)
	}

	// Current page indicator
	btn := tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%d/%d", page+1, totalPages), "ignore")
	navRow = append(navRow, btn)

	// Next button
	if page < totalPages-1 {
		btn := tgbotapi.NewInlineKeyboardButtonData("▶️", fmt.Sprintf("users_page_%d", page+1))
		navRow = append(navRow, btn)
	}

	// Last page button
	if page < totalPages-2 {
		btn := tgbotapi.NewInlineKeyboardButtonData("⏭", fmt.Sprintf("users_page_%d", totalPages-1))
		navRow = append(navRow, btn)
	}

	rows = append(rows, navRow)

	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}
