package telegram

import (
	"fmt"
	"log"
	"mispilkabot/internal/services"
	"sort"
	"strings"
	"time"

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
	default:
		// Check if it's an admin command
		if b.isAdmin(message.From.ID) {
			b.handleAdminCommand(message)
		}
	}
}

func (b *Bot) startCommand(message *tgbotapi.Message) error {
	chatID := fmt.Sprint(message.Chat.ID)

	// Extract scenario ID from command payload (e.g., /start scenario_id)
	payload := message.CommandArguments()

	// Get or create user
	isNew, err := services.IsNewUser(chatID)
	if err != nil {
		return fmt.Errorf("failed to check if user is new: %w", err)
	}

	if isNew {
		if err := services.AddUser(message); err != nil {
			return fmt.Errorf("failed to add new user: %w", err)
		}
	}

	// Determine which scenario to use
	scenarioID := payload
	if scenarioID == "" {
		// Use default scenario
		defaultID, err := b.scenarioService.GetDefaultScenario()
		if err != nil {
			// No default scenario set - check if any scenarios exist
			scenarios, listErr := b.scenarioService.ListScenarios()
			if listErr != nil || len(scenarios) == 0 {
				// No scenarios configured
				if b.isAdmin(message.From.ID) {
					return b.sendMessage(message.Chat.ID, "⚠️ No scenarios configured.\n\nUse /create_scenario to create one.")
				}
				return b.sendMessage(message.Chat.ID, "⚠️ Bot is not configured yet. Please contact the administrator.")
			}
			// Scenarios exist but no default is set
			if b.isAdmin(message.From.ID) {
				return b.sendMessage(message.Chat.ID, "⚠️ No default scenario set.\n\nUse /set_default_scenario {id} to set one.")
			}
			return b.sendMessage(message.Chat.ID, "⚠️ Bot is not configured yet. Please contact the administrator.")
		}
		scenarioID = defaultID
	}

	// Handle scenario logic
	return b.handleScenarioStart(chatID, scenarioID, isNew)
}

// handleScenarioStart handles the scenario start logic for a user
func (b *Bot) handleScenarioStart(chatID, scenarioID string, isNew bool) error {
	// Verify scenario exists
	if _, err := b.scenarioService.GetScenario(scenarioID); err != nil {
		return fmt.Errorf("failed to get scenario %s: %w", scenarioID, err)
	}

	// Check user's current state with this scenario
	userScenario, err := services.GetUserScenario(chatID, scenarioID)
	if err != nil {
		return fmt.Errorf("failed to get user scenario state: %w", err)
	}

	// Handle based on user's scenario status
	switch userScenario.Status {
	case services.StatusNotStarted:
		// Start the scenario
		if err := b.startScenarioForUser(chatID, scenarioID); err != nil {
			return fmt.Errorf("failed to start scenario: %w", err)
		}
		// Send welcome message
		return b.sendScenarioWelcome(chatID, scenarioID)

	case services.StatusActive:
		// User already has this scenario active - send welcome message
		return b.sendScenarioWelcome(chatID, scenarioID)

	case services.StatusCompleted:
		// User completed this scenario - send summary
		return b.sendSummary(chatID, scenarioID)

	case services.StatusStopped:
		// User stopped this scenario - restart it
		if err := b.restartScenarioForUser(chatID, scenarioID); err != nil {
			return fmt.Errorf("failed to restart scenario: %w", err)
		}
		return b.sendScenarioWelcome(chatID, scenarioID)
	}

	log.Printf("Unknown scenario status for user %s in scenario %s: %s", chatID, scenarioID, userScenario.Status)
	return fmt.Errorf("unknown scenario status: %s", userScenario.Status)
}

// startScenarioForUser initializes and starts a scenario for a user
func (b *Bot) startScenarioForUser(chatID, scenarioID string) error {
	// Create initial scenario state
	now := time.Now()
	state := &services.UserScenarioState{
		Status:              services.StatusActive,
		CurrentMessageIndex: 0,
		LastSentAt:          &now,
	}

	// Set as active scenario
	if err := services.SetUserActiveScenario(chatID, scenarioID); err != nil {
		return fmt.Errorf("failed to set active scenario: %w", err)
	}

	// Save scenario state
	if err := services.SetUserScenario(chatID, scenarioID, state); err != nil {
		return fmt.Errorf("failed to set user scenario: %w", err)
	}

	log.Printf("Started scenario %s for user %s", scenarioID, chatID)
	return nil
}

// restartScenarioForUser restarts a stopped scenario for a user
func (b *Bot) restartScenarioForUser(chatID, scenarioID string) error {
	now := time.Now()
	state := &services.UserScenarioState{
		Status:              services.StatusActive,
		CurrentMessageIndex: 0,
		LastSentAt:          &now,
	}

	if err := services.SetUserActiveScenario(chatID, scenarioID); err != nil {
		return err
	}

	if err := services.SetUserScenario(chatID, scenarioID, state); err != nil {
		return err
	}

	log.Printf("Restarted scenario %s for user %s", scenarioID, chatID)
	return nil
}

// sendScenarioWelcome sends a welcome message for a scenario
// TODO: Implement template rendering with scenario-specific variables
func (b *Bot) sendScenarioWelcome(chatID, scenarioID string) error {
	// For now, send a simple welcome message
	// TODO: Load template from scenario's welcome_template file and render with variables
	scenario, err := b.scenarioService.GetScenario(scenarioID)
	if err != nil {
		return fmt.Errorf("failed to get scenario: %w", err)
	}

	parsedID, err := parseID(chatID)
	if err != nil {
		return fmt.Errorf("failed to parse chatID: %w", err)
	}

	// Build welcome message
	text := fmt.Sprintf("Добро пожаловать в сценарий <b>%s</b>!\n\n", scenario.Name)
	text += "Нажмите кнопку ниже, чтобы начать."

	msg := tgbotapi.NewMessage(parsedID, text)
	msg.ParseMode = "HTML"
	msg.DisableWebPagePreview = true
	msg.ReplyMarkup = dataButton("🔲 Принимаю", "accept")

	if _, err := b.bot.Send(msg); err != nil {
		return fmt.Errorf("failed to send welcome message: %w", err)
	}

	log.Printf("Sent welcome message for scenario %s to user %s", scenarioID, chatID)
	return nil
}

// sendSummary sends a summary message for a completed scenario
// TODO: Implement template rendering with scenario-specific summary
func (b *Bot) sendSummary(chatID string, scenarioID string) error {
	// Get scenario details
	scenario, err := b.scenarioService.GetScenario(scenarioID)
	if err != nil {
		return fmt.Errorf("failed to get scenario: %w", err)
	}

	parsedID, err := parseID(chatID)
	if err != nil {
		return fmt.Errorf("failed to parse chatID: %w", err)
	}

	// TODO: Load and render summary template from scenario
	text := fmt.Sprintf("Вы завершили сценарий <b>%s</b>.", scenario.Name)

	msg := tgbotapi.NewMessage(parsedID, text)
	msg.ParseMode = "HTML"
	msg.DisableWebPagePreview = true

	if _, err := b.bot.Send(msg); err != nil {
		return fmt.Errorf("failed to send summary message: %w", err)
	}

	log.Printf("Sent summary message for scenario %s to user %s", scenario.ID, chatID)
	return nil
}

// loadTemplate loads a template file and renders it with provided variables
// TODO: Implement full template loading and rendering
func (b *Bot) loadTemplate(templateFile string, variables map[string]string) (string, error) {
	// TODO: Load template from file and replace placeholders
	// For now, return empty string
	return "", fmt.Errorf("template loading not yet implemented")
}


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
		msg := tgbotapi.NewMessage(message.Chat.ID, "Пользователей пока нет.")
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

	// 1. Header with index and name
	sb.WriteString(fmt.Sprintf("<b>%d. %s</b>\n", index+1, displayName))

	// 2. Phone number (if exists)
	if user.PaymentInfo != nil && user.PaymentInfo.CustomerPhone != "" {
		sb.WriteString(fmt.Sprintf("Телефон: %s\n", user.PaymentInfo.CustomerPhone))
	} else {
		sb.WriteString("Телефон: не указан\n")
	}

	// 3. Registration info
	sb.WriteString(fmt.Sprintf("Регистрация: %s\n", user.RegTime.Format("02.01.2006 15:04")))

	// 4. Status (with fixed text)
	sb.WriteString("Статус: ")
	if user.IsMessaging {
		sb.WriteString("Принял условия обработки перс. данных")
	} else {
		sb.WriteString("Не принял условия обработки перс. данных")
	}
	sb.WriteString("\n")

	// 5. Total paid amount (from payment info)
	if user.PaymentInfo != nil && user.PaymentInfo.Sum != "" {
		sb.WriteString(fmt.Sprintf("Сумма оплаты: %s ₽\n", user.PaymentInfo.Sum))
	} else {
		sb.WriteString("Сумма оплаты: не оплачено\n")
	}

	// 6. Payment info (date, link)
	if user.HasPaid() {
		sb.WriteString(fmt.Sprintf("Последняя оплата: %s\n", user.GetPaymentDate().Format("02.01.2006 15:04")))
		if user.PaymentLink != "" {
			sb.WriteString(fmt.Sprintf("Ссылка на оплату: %s\n", user.PaymentLink))
		}
	} else {
		sb.WriteString("Последняя оплата: не оплачено\n")
	}

	// 7. Group info (joined date)
	if user.HasJoined() {
		sb.WriteString(fmt.Sprintf("Группа: вступил %s\n", user.GetJoinedAt().Format("02.01.2006 15:04")))
	} else {
		sb.WriteString("Группа: не вступил\n")
	}
	// Show invite link if exists (independent of join status)
	if user.InviteLink != "" {
		sb.WriteString(fmt.Sprintf("Ссылка на группу: %s\n", user.InviteLink))
	}

	// 8. Messages queue info
	sb.WriteString(fmt.Sprintf("В очереди: %d сообщений\n", len(user.MessagesList)))

	// 9. Technical info (collapsed)
	sb.WriteString(fmt.Sprintf("ID: <code>%s</code>\n", chatID))

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
	sb.WriteString(fmt.Sprintf("<b>Пользователи</b> (стр. %d/%d, всего %d)\n\n",
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

	// Add pagination keyboard only if there are multiple pages
	if totalPages > 1 {
		keyboard := b.buildUsersKeyboard(page, totalPages)
		msg.ReplyMarkup = &keyboard
	}

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
	sb.WriteString(fmt.Sprintf("<b>Пользователи</b> (стр. %d/%d, всего %d)\n\n",
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

	// Add pagination keyboard only if there are multiple pages
	if totalPages > 1 {
		keyboard := b.buildUsersKeyboard(page, totalPages)
		edit.ReplyMarkup = &keyboard
	}

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
