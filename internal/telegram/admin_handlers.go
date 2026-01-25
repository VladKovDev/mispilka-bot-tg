package telegram

import (
	"fmt"
	"log"
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
		return b.sendMessage(message.Chat.ID, "Scenario service not initialized")
	}

	scenarios, err := b.scenarioService.ListScenarios()
	if err != nil {
		return b.sendMessage(message.Chat.ID, "Failed to load scenarios: "+err.Error())
	}

	if len(scenarios) == 0 {
		return b.sendMessage(message.Chat.ID, "No scenarios found. Use /create_scenario to create one.")
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
		return b.sendMessage(message.Chat.ID, "Wizard manager not initialized")
	}

	userID := fmt.Sprint(message.From.ID)

	state, err := b.wizardManager.Start(userID, wizard.WizardTypeCreateScenario)
	if err != nil {
		return b.sendMessage(message.Chat.ID, "Failed to start wizard: "+err.Error())
	}

	return b.sendWizardMessage(message.Chat.ID, state)
}

// setDefaultScenarioCommand sets a scenario as default
func (b *Bot) setDefaultScenarioCommand(message *tgbotapi.Message, payload string) error {
	if payload == "" {
		return b.sendMessage(message.Chat.ID, "Usage: /set_default_scenario {scenario_id}")
	}

	if b.scenarioService == nil {
		return b.sendMessage(message.Chat.ID, "Scenario service not initialized")
	}

	if err := b.scenarioService.SetDefaultScenario(payload); err != nil {
		return b.sendMessage(message.Chat.ID, "Failed: "+err.Error())
	}

	return b.sendMessage(message.Chat.ID, fmt.Sprintf("✅ Scenario '%s' is now the default", payload))
}

// deleteScenarioCommand deletes a scenario
func (b *Bot) deleteScenarioCommand(message *tgbotapi.Message, payload string) error {
	if payload == "" {
		return b.sendMessage(message.Chat.ID, "Usage: /delete_scenario {scenario_id}")
	}

	if b.scenarioService == nil {
		return b.sendMessage(message.Chat.ID, "Scenario service not initialized")
	}

	if err := b.scenarioService.DeleteScenario(payload); err != nil {
		return b.sendMessage(message.Chat.ID, "Failed: "+err.Error())
	}

	return b.sendMessage(message.Chat.ID, fmt.Sprintf("🗑️ Scenario '%s' deleted", payload))
}

// createBroadcastCommand starts broadcast creation wizard
func (b *Bot) createBroadcastCommand(message *tgbotapi.Message) error {
	// TODO: Implement broadcast wizard
	return b.sendMessage(message.Chat.ID, "Broadcast creation coming soon!")
}

// sendBroadcastCommand sends a broadcast
func (b *Bot) sendBroadcastCommand(message *tgbotapi.Message) error {
	// TODO: Implement broadcast sending
	return b.sendMessage(message.Chat.ID, "Broadcast sending coming soon!")
}

// Helper methods

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
	// Generate appropriate prompt based on wizard type and current step
	var prompt string

	switch state.WizardType {
	case wizard.WizardTypeCreateScenario:
		prompt = b.getScenarioStepPrompt(state.CurrentStep)
	case wizard.WizardTypeCreateBroadcast:
		prompt = "Broadcast wizard coming soon!"
	default:
		prompt = fmt.Sprintf("Unknown wizard type: %s", state.WizardType)
	}

	msg := tgbotapi.NewMessage(chatID, prompt)
	msg.ParseMode = "HTML"

	if _, err := b.bot.Send(msg); err != nil {
		return fmt.Errorf("failed to send wizard message: %w", err)
	}

	return nil
}

// getScenarioStepPrompt returns the prompt text for a scenario creation step
func (b *Bot) getScenarioStepPrompt(step wizard.WizardStep) string {
	switch step {
	case wizard.StepScenarioName:
		return "📝 <b>Create New Scenario</b>\n\n" +
			"Let's create a new scenario step by step.\n\n" +
			"First, enter a <b>name</b> for this scenario:"
	case wizard.StepProductName:
		return "📦 Enter the <b>product name</b> (what users are paying for):"
	case wizard.StepProductPrice:
		return "💰 Enter the <b>product price</b> in rubles (e.g., 500):"
	case wizard.StepPaidContent:
		return "📝 Enter a <b>description</b> of the paid content:"
	case wizard.StepPrivateGroupID:
		return "👥 Enter the <b>private group ID</b>:\n\n" +
			"<i>You can find this by adding your bot to the group and using /users command to see the group ID format.</i>"
	case wizard.StepSummaryMessage:
		return "📬 <b>Summary Message</b>\n\n" +
			"Enter the message text that users will see immediately after payment:\n\n" +
			"<i>This is the first message users receive. You can use placeholders like {{payment_link}} and {{invite_link}}</i>"
	case wizard.StepSummaryPhotos:
		return "🖼️ <b>Summary Photos</b>\n\n" +
			"Send photos to include with the summary message.\n\n" +
			"<i>Send multiple photos and click 'Done' when finished, or just click 'Done' to skip photos.</i>"
	case wizard.StepSummaryButtons:
		return "🔘 <b>Summary Buttons</b>\n\n" +
			"Add buttons to the summary message.\n\n" +
			"<b>Format (one per line):</b>\n" +
			"Button Text|url|https://example.com\n" +
			"Button Text|callback|action_name\n\n" +
			"<i>Or send 'skip' to continue without buttons.</i>"
	case wizard.StepMessageText:
		return "📝 <b>Message Text</b>\n\n" +
			"Enter the message text:\n\n" +
			"<i>Use HTML formatting: <b>, <code>, <i>, etc.</i>"
	case wizard.StepMessagePhotos:
		return "🖼️ <b>Message Photos</b>\n\n" +
			"Send photos to include with this message.\n\n" +
			"<i>Send multiple photos and click 'Done' when finished, or just click 'Done' to skip photos.</i>"
	case wizard.StepMessageTiming:
		return "⏰ <b>Message Timing</b>\n\n" +
			"Enter when this message should be sent after the previous message:\n\n" +
			"<b>Formats:</b>\n" +
			"• <code>1h 30m</code> - 1 hour 30 minutes\n" +
			"• <code>90m</code> - 90 minutes\n" +
			"• <code>2h</code> - 2 hours\n\n" +
			"<i>Minimum: 1 minute, Maximum: 1 year</i>"
	case wizard.StepMessageButtons:
		return "🔘 <b>Message Buttons</b>\n\n" +
			"Add buttons to this message.\n\n" +
			"<b>Format (one per line):</b>\n" +
			"Button Text|url|https://example.com\n" +
			"Button Text|callback|action_name\n\n" +
			"<i>Or send 'skip' to continue without buttons.</i>"
	default:
		return fmt.Sprintf("Please send data for step: %s", step)
	}
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
func (b *Bot) sendMessage(chatID int64, text string) error {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"

	if _, err := b.bot.Send(msg); err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}
