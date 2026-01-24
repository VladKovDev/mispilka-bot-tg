package telegram

import (
	"context"
	"fmt"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"mispilkabot/internal/domain/command"
)

// CommandService handles Telegram Bot API command registration.
// It manages role-based command visibility for public and admin users.
type CommandService struct {
	bot    *tgbotapi.BotAPI
	mapper *CommandMapper
}

// NewCommandService creates a new CommandService instance.
func NewCommandService(bot *tgbotapi.BotAPI) *CommandService {
	return &CommandService{
		bot:    bot,
		mapper: NewCommandMapper(),
	}
}

// RegisterCommands registers bot commands with Telegram API using role-based visibility.
// Public commands are registered for all private chats.
// Admin commands are registered per-admin-user and include both public and admin commands.
func (s *CommandService) RegisterCommands(ctx context.Context, adminIDs []int64) error {
	// Register public commands for all private chats
	publicCmds := command.GetPublicCommands()
	publicBotCmds := s.mapper.ToBotCommands(publicCmds)
	publicScope := s.mapper.GetPublicScope()

	setPublicCmds := tgbotapi.NewSetMyCommandsWithScope(publicScope, publicBotCmds...)

	if _, err := s.bot.Request(setPublicCmds); err != nil {
		log.Printf("[COMMANDS] Failed to register public commands: %v", err)
		return fmt.Errorf("failed to register public commands: %w", err)
	}
	log.Printf("[COMMANDS] Registered %d public commands", len(publicCmds))

	// Register admin commands for each admin user
	adminCmds := command.GetAdminCommands()
	if len(adminCmds) > 0 && len(adminIDs) > 0 {
		adminBotCmds := s.mapper.ToBotCommands(adminCmds)
		// Include public commands + admin commands for admins
		allAdminCmds := append(s.mapper.ToBotCommands(publicCmds), adminBotCmds...)

		for _, adminID := range adminIDs {
			select {
			case <-ctx.Done():
				return fmt.Errorf("admin registration cancelled: %w", ctx.Err())
			default:
			}

			adminScope := s.mapper.GetAdminScope(adminID)
			setAdminCmds := tgbotapi.NewSetMyCommandsWithScope(adminScope, allAdminCmds...)
			if _, err := s.bot.Request(setAdminCmds); err != nil {
				log.Printf("[COMMANDS] Failed to register admin commands for %d: %v", adminID, err)
				return fmt.Errorf("failed to register admin commands for %d: %w", adminID, err)
			}
			log.Printf("[COMMANDS] Registered admin commands for user %d", adminID)
		}
	}

	return nil
}
