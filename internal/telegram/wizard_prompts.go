package telegram

import (
	"fmt"

	"mispilkabot/internal/services/wizard"
)

// Wizard prompt constants
const (
	// General info prompts
	promptScenarioName = "📝 <b>Create New Scenario</b>\n\n" +
		"Let's create a new scenario step by step.\n\n" +
		"First, enter a <b>name</b> for this scenario:"
	promptProductName = "📦 Enter the <b>product name</b> (what users are paying for):"
	promptProductPrice = "💰 Enter the <b>product price</b> in rubles (e.g., 500):"
	promptPaidContent = "📝 Enter a <b>description</b> of the paid content:"
	promptPrivateGroupID = "👥 Enter the <b>private group ID</b>:\n\n" +
		"<i>Format: -100XXXXXXXXXX (e.g., -1001234567890)</i>\n" +
		"<i>You can find this by adding your bot to the group.</i>"

	// Summary prompts
	promptSummaryMessage = "📬 <b>Summary Message</b>\n\n" +
		"Enter the message text that users will see immediately after payment.\n\n" +
		"<i>This is the first message users receive. You can use placeholders like {{payment_link}} and {{invite_link}}</i>"
	promptSummaryPhotos = "🖼️ <b>Summary Photos</b>\n\n" +
		"Send photos to include with the summary message.\n\n" +
		"<i>Send multiple photos and click 'Done' when finished, or just click 'Done' to skip photos.</i>"
	promptSummaryButtons = "🔘 <b>Summary Buttons</b>\n\n" +
		"Add buttons to the summary message.\n\n" +
		"<b>Format (one per line):</b>\n" +
		"Button Text|url|https://example.com\n" +
		"Button Text|callback|action_name\n\n" +
		"<i>Or send 'skip' to continue without buttons.</i>"

	// Message prompts
	promptMessageText = "📝 <b>Message Text</b>\n\n" +
		"Enter the message text.\n\n" +
		"<i>Use HTML formatting: <b>, <code>, <i>, etc.</i>"
	promptMessagePhotos = "🖼️ <b>Message Photos</b>\n\n" +
		"Send photos to include with this message.\n\n" +
		"<i>Send multiple photos and click 'Done' when finished, or just click 'Done' to skip photos.</i>"
	promptMessageTiming = "⏰ <b>Message Timing</b>\n\n" +
		"Enter when this message should be sent after the previous message.\n\n" +
		"<b>Formats:</b>\n" +
		"• <code>1h 30m</code> - 1 hour 30 minutes\n" +
		"• <code>90m</code> - 90 minutes\n" +
		"• <code>2h</code> - 2 hours\n\n" +
		"<i>Minimum: 1 minute, Maximum: 1 year</i>"
	promptMessageButtons = "🔘 <b>Message Buttons</b>\n\n" +
		"Add buttons to this message.\n\n" +
		"<b>Format (one per line):</b>\n" +
		"Button Text|url|https://example.com\n" +
		"Button Text|callback|action_name\n\n" +
		"<i>Or send 'skip' to continue without buttons.</i>"
)

// getPromptForStep returns the prompt text for a wizard step
func (b *Bot) getPromptForStep(step wizard.WizardStep) string {
	switch step {
	// General info
	case wizard.StepScenarioName:
		return promptScenarioName
	case wizard.StepProductName:
		return promptProductName
	case wizard.StepProductPrice:
		return promptProductPrice
	case wizard.StepPaidContent:
		return promptPaidContent
	case wizard.StepPrivateGroupID:
		return promptPrivateGroupID
	// Summary
	case wizard.StepSummaryMessage:
		return promptSummaryMessage
	case wizard.StepSummaryPhotos:
		return promptSummaryPhotos
	case wizard.StepSummaryButtons:
		return promptSummaryButtons
	// Messages
	case wizard.StepMessageText:
		return getMessageTextPrompt()
	case wizard.StepMessagePhotos:
		return promptMessagePhotos
	case wizard.StepMessageTiming:
		return promptMessageTiming
	case wizard.StepMessageButtons:
		return promptMessageButtons
	default:
		return fmt.Sprintf("Please send data for step: %s", step)
	}
}

// getMessageTextPrompt returns a prompt with the current message number
func getMessageTextPrompt() string {
	return "📝 <b>Message Text</b>\n\n" +
		"Enter the message text.\n\n" +
		"<i>You can use template variables like {{user.user_name}}</i>"
}
