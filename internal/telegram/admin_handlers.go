package telegram

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"mispilkabot/internal/domain/command"
	domainScenario "mispilkabot/internal/domain/scenario"
	"mispilkabot/internal/services/scenario"
	"mispilkabot/internal/services/wizard"
)

// registerAdminCommands registers all admin commands
func (b *Bot) registerAdminCommands() {
	// Type assertion to ensure scenario service is compatible
	var _ *scenario.Service = b.scenarioService

	adminCommands := []*command.Command{
		{
			Name:        "scenarios",
			Description: "Show all scenarios",
			Role:        command.RoleAdmin,
		},
		{
			Name:        "create_scenario",
			Description: "Create a new scenario",
			Role:        command.RoleAdmin,
		},
		{
			Name:        "set_default_scenario",
			Description: "Set default scenario",
			Role:        command.RoleAdmin,
		},
		{
			Name:        "delete_scenario",
			Description: "Delete a scenario",
			Role:        command.RoleAdmin,
		},
		{
			Name:        "demo_scenario",
			Description: "Demonstrate a scenario",
			Role:        command.RoleAdmin,
		},
		{
			Name:        "create_broadcast",
			Description: "Create a broadcast",
			Role:        command.RoleAdmin,
		},
		{
			Name:        "send_broadcast",
			Description: "Send a broadcast",
			Role:        command.RoleAdmin,
		},
	}

	// Register commands with the command service
	for _, cmd := range adminCommands {
		b.commandService.RegisterCommand(cmd)
	}
}

// handleAdminCommand handles admin-specific commands
func (b *Bot) handleAdminCommand(message *tgbotapi.Message) {
	if !b.isAdmin(message.From.ID) {
		log.Printf("[ADMIN] User %d attempted to use admin command without permission", message.From.ID)
		return
	}

	cmd := message.Command()
	args := message.CommandArguments()

	switch cmd {
	case "scenarios":
		if err := b.scenariosCommand(message); err != nil {
			log.Printf("[ADMIN] Failed to handle /scenarios command: %v", err)
		}
	case "create_scenario":
		if err := b.createScenarioCommand(message); err != nil {
			log.Printf("[ADMIN] Failed to handle /create_scenario command: %v", err)
		}
	case "set_default_scenario":
		if err := b.setDefaultScenarioCommand(message, args); err != nil {
			log.Printf("[ADMIN] Failed to handle /set_default_scenario command: %v", err)
		}
	case "delete_scenario":
		if err := b.deleteScenarioCommand(message, args); err != nil {
			log.Printf("[ADMIN] Failed to handle /delete_scenario command: %v", err)
		}
	case "demo_scenario":
		if err := b.demoScenarioCommand(message, args); err != nil {
			log.Printf("[ADMIN] Failed to handle /demo_scenario command: %v", err)
		}
	case "create_broadcast":
		if err := b.createBroadcastCommand(message); err != nil {
			log.Printf("[ADMIN] Failed to handle /create_broadcast command: %v", err)
		}
	case "send_broadcast":
		if err := b.sendBroadcastCommand(message); err != nil {
			log.Printf("[ADMIN] Failed to handle /send_broadcast command: %v", err)
		}
	}
}

// scenariosCommand shows all scenarios as buttons
func (b *Bot) scenariosCommand(message *tgbotapi.Message) error {
	if b.scenarioService == nil {
		return b.sendMessage(fmt.Sprint(message.Chat.ID), "Scenario service not initialized")
	}

	scenarios, err := b.scenarioService.ListScenarios()
	if err != nil {
		return b.sendMessage(fmt.Sprint(message.Chat.ID), "Failed to load scenarios: "+err.Error())
	}

	if len(scenarios) == 0 {
		return b.sendMessage(fmt.Sprint(message.Chat.ID), "No scenarios found. Use /create_scenario to create one.")
	}

	// Build message with scenario list
	var sb strings.Builder
	sb.WriteString("📋 Available Scenarios:\n\n")

	defaultID, err := b.scenarioService.GetDefaultScenario()
	if err != nil {
		log.Printf("[ADMIN] Failed to get default scenario: %v", err)
	}

	for _, sc := range scenarios {
		marker := " "
		if defaultID != "" && sc.ID == defaultID {
			marker = "✅"
		}
		sb.WriteString(fmt.Sprintf("%s %s (%s)\n", marker, sc.Name, sc.ID))
	}

	// Create inline keyboard for scenario actions
	keyboard := b.buildScenarioKeyboard(scenarios)

	return b.sendMessageWithKeyboard(message.Chat.ID, sb.String(), keyboard)
}

// createScenarioCommand starts scenario creation wizard
func (b *Bot) createScenarioCommand(message *tgbotapi.Message) error {
	if b.wizardManager == nil {
		return b.sendMessage(fmt.Sprint(message.Chat.ID), "Wizard manager not initialized")
	}

	userID := fmt.Sprint(message.From.ID)

	state, err := b.wizardManager.Start(userID, wizard.WizardTypeCreateScenario)
	if err != nil {
		return b.sendMessage(fmt.Sprint(message.Chat.ID), "Failed to start wizard: "+err.Error())
	}

	return b.sendWizardMessage(message.Chat.ID, state)
}

// setDefaultScenarioCommand sets a scenario as default
func (b *Bot) setDefaultScenarioCommand(message *tgbotapi.Message, payload string) error {
	if payload == "" {
		return b.sendMessage(fmt.Sprint(message.Chat.ID), "Usage: /set_default_scenario {scenario_id}")
	}

	if b.scenarioService == nil {
		return b.sendMessage(fmt.Sprint(message.Chat.ID), "Scenario service not initialized")
	}

	if err := b.scenarioService.SetDefaultScenario(payload); err != nil {
		return b.sendMessage(fmt.Sprint(message.Chat.ID), "Failed: "+err.Error())
	}

	return b.sendMessage(fmt.Sprint(message.Chat.ID), fmt.Sprintf("✅ Scenario '%s' is now the default", payload))
}

// deleteScenarioCommand deletes a scenario
func (b *Bot) deleteScenarioCommand(message *tgbotapi.Message, payload string) error {
	if payload == "" {
		return b.sendMessage(fmt.Sprint(message.Chat.ID), "Usage: /delete_scenario {scenario_id}")
	}

	if b.scenarioService == nil {
		return b.sendMessage(fmt.Sprint(message.Chat.ID), "Scenario service not initialized")
	}

	if err := b.scenarioService.DeleteScenario(payload); err != nil {
		return b.sendMessage(fmt.Sprint(message.Chat.ID), "Failed: "+err.Error())
	}

	return b.sendMessage(fmt.Sprint(message.Chat.ID), fmt.Sprintf("🗑️ Scenario '%s' deleted", payload))
}

// demoScenarioCommand demonstrates a scenario with template highlighting
func (b *Bot) demoScenarioCommand(message *tgbotapi.Message, payload string) error {
	if payload == "" {
		return b.sendMessage(fmt.Sprint(message.Chat.ID), "Usage: /demo_scenario {scenario_id}")
	}

	if b.scenarioService == nil {
		return b.sendMessage(fmt.Sprint(message.Chat.ID), "Scenario service not initialized")
	}

	sc, err := b.scenarioService.GetScenario(payload)
	if err != nil {
		return b.sendMessage(fmt.Sprint(message.Chat.ID), "Failed: "+err.Error())
	}

	// Build demo message with scenario details
	return b.sendScenarioDemo(message.Chat.ID, sc)
}

// createBroadcastCommand starts broadcast creation wizard
func (b *Bot) createBroadcastCommand(message *tgbotapi.Message) error {
	// TODO: Implement broadcast wizard
	return b.sendMessage(fmt.Sprint(message.Chat.ID), "Broadcast creation coming soon!")
}

// sendBroadcastCommand sends a broadcast
func (b *Bot) sendBroadcastCommand(message *tgbotapi.Message) error {
	// TODO: Implement broadcast sending
	return b.sendMessage(fmt.Sprint(message.Chat.ID), "Broadcast sending coming soon!")
}

// Helper methods

// isDefaultScenario checks if the given scenario ID is the default one
func (b *Bot) isDefaultScenario(scenarioID string) bool {
	if b.scenarioService == nil {
		return false
	}
	defaultID, err := b.scenarioService.GetDefaultScenario()
	if err != nil {
		return false
	}
	return defaultID == scenarioID
}

// buildScenarioKeyboard builds an inline keyboard for scenario actions
func (b *Bot) buildScenarioKeyboard(scenarios []*domainScenario.Scenario) tgbotapi.InlineKeyboardMarkup {
	keyboard := make([][]tgbotapi.InlineKeyboardButton, 0)

	for _, sc := range scenarios {
		btn := tgbotapi.NewInlineKeyboardButtonData(sc.Name, fmt.Sprintf("scenario_info_%s", sc.ID))
		row := []tgbotapi.InlineKeyboardButton{btn}
		keyboard = append(keyboard, row)
	}

	return tgbotapi.NewInlineKeyboardMarkup(keyboard...)
}

// sendWizardMessage sends a message for the current wizard step
func (b *Bot) sendWizardMessage(chatID int64, state *wizard.WizardState) error {
	// TODO: Implement proper wizard message generation based on step
	// For now, just show the current step
	msgText := fmt.Sprintf("Wizard started. Current step: %s\n\nPlease send the required data.", state.CurrentStep)

	msg := tgbotapi.NewMessage(chatID, msgText)
	msg.ParseMode = "HTML"

	if _, err := b.bot.Send(msg); err != nil {
		return fmt.Errorf("failed to send wizard message: %w", err)
	}

	return nil
}

// sendScenarioDemo sends a demonstration of a scenario
func (b *Bot) sendScenarioDemo(chatID int64, sc *domainScenario.Scenario) error {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("<b>Scenario:</b> %s\n", sc.Name))
	sb.WriteString(fmt.Sprintf("<b>ID:</b> <code>%s</code>\n", sc.ID))
	sb.WriteString(fmt.Sprintf("<b>Status:</b> %s\n", map[bool]string{true: "Active", false: "Inactive"}[sc.IsActive]))
	sb.WriteString(fmt.Sprintf("<b>Created:</b> %s\n\n", sc.CreatedAt.Format("02.01.2006 15:04")))

	sb.WriteString("<b>Prodamus Config:</b>\n")
	sb.WriteString(fmt.Sprintf("  Product: %s\n", sc.Config.Prodamus.ProductName))
	sb.WriteString(fmt.Sprintf("  Price: %s ₽\n", sc.Config.Prodamus.ProductPrice))
	sb.WriteString(fmt.Sprintf("  Content: %s\n", sc.Config.Prodamus.PaidContent))
	sb.WriteString(fmt.Sprintf("  Group ID: <code>%s</code>\n\n", sc.Config.Prodamus.PrivateGroupID))

	sb.WriteString(fmt.Sprintf("<b>Messages:</b> %d\n", len(sc.Messages.MessagesList)))
	for i, msgID := range sc.Messages.MessagesList {
		if msgData, ok := sc.Messages.Messages[msgID]; ok {
			sb.WriteString(fmt.Sprintf("  %d. %s (after %dh %dm)\n",
				i+1, msgID, msgData.Timing.Hours, msgData.Timing.Minutes))
		}
	}

	msg := tgbotapi.NewMessage(chatID, sb.String())
	msg.ParseMode = "HTML"
	msg.DisableWebPagePreview = true

	if _, err := b.bot.Send(msg); err != nil {
		return fmt.Errorf("failed to send scenario demo: %w", err)
	}

	return nil
}

// sendMessageWithKeyboard sends a message with an inline keyboard
func (b *Bot) sendMessageWithKeyboard(chatID int64, text string, keyboard tgbotapi.InlineKeyboardMarkup) error {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	msg.DisableWebPagePreview = true

	if len(keyboard.InlineKeyboard) > 0 {
		msg.ReplyMarkup = &keyboard
	}

	if _, err := b.bot.Send(msg); err != nil {
		return fmt.Errorf("failed to send message with keyboard: %w", err)
	}

	return nil
}

// sendMessage sends a simple text message
func (b *Bot) sendMessage(chatID string, text string) error {
	parsedID, err := strconv.ParseInt(chatID, 10, 64)
	if err != nil {
		return fmt.Errorf("failed to parse chatID %s: %w", chatID, err)
	}

	msg := tgbotapi.NewMessage(parsedID, text)
	msg.ParseMode = "HTML"

	if _, err := b.bot.Send(msg); err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}
