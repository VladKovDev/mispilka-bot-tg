package telegram

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"mispilkabot/internal/domain/command"
)

// CommandMapper converts domain commands to Telegram Bot API commands and scopes.
type CommandMapper struct{}

// NewCommandMapper creates a new CommandMapper instance.
func NewCommandMapper() *CommandMapper {
	return &CommandMapper{}
}

// ToBotCommands converts domain commands to Telegram BotCommand slice.
func (m *CommandMapper) ToBotCommands(cmds []command.Command) []tgbotapi.BotCommand {
	result := make([]tgbotapi.BotCommand, len(cmds))
	for i, cmd := range cmds {
		result[i] = tgbotapi.BotCommand{
			Command:     "/" + cmd.Name,
			Description: cmd.Description,
		}
	}
	return result
}

// GetPublicScope returns the scope for all private chats.
func (m *CommandMapper) GetPublicScope() tgbotapi.BotCommandScope {
	return tgbotapi.NewBotCommandScopeAllPrivateChats()
}

// GetAdminScope returns the scope for a specific admin user.
func (m *CommandMapper) GetAdminScope(userID int64) tgbotapi.BotCommandScope {
	return tgbotapi.NewBotCommandScopeChat(userID)
}
