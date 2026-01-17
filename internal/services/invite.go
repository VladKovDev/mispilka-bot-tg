package services

import (
	"encoding/json"
	"fmt"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// parseChatID is a helper function to parse a chat ID from string to int64
func parseChatID(id string) (int64, error) {
	return strconv.ParseInt(id, 10, 64)
}

// GenerateInviteLink creates a new invite link for a chat/group with member limit set to 1
// Returns the invite link URL or an error if generation fails
func GenerateInviteLink(userID, groupID string, bot *tgbotapi.BotAPI) (string, error) {
	groupIDInt, err := parseChatID(groupID)
	if err != nil {
		return "", fmt.Errorf("invalid groupID format: %w", err)
	}

	// Create chat invite link using the Telegram Bot API
	linkConfig := tgbotapi.CreateChatInviteLinkConfig{
		ChatConfig:  tgbotapi.ChatConfig{ChatID: groupIDInt},
		MemberLimit: 1,
	}

	result, err := bot.Request(linkConfig)
	if err != nil {
		return "", fmt.Errorf("failed to create invite link: %w", err)
	}

	if !result.Ok {
		return "", fmt.Errorf("API error: %s", result.Description)
	}

	// Unmarshal the result to get the invite link
	if len(result.Result) == 0 {
		return "", fmt.Errorf("empty result from Telegram API")
	}
	var inviteLink tgbotapi.ChatInviteLink
	if err := json.Unmarshal(result.Result, &inviteLink); err != nil {
		return "", fmt.Errorf("failed to unmarshal invite link: %w", err)
	}

	return inviteLink.InviteLink, nil
}

// RevokeInviteLink revokes an existing invite link for a chat
func RevokeInviteLink(groupID, inviteLink string, bot *tgbotapi.BotAPI) error {
	groupIDInt, err := parseChatID(groupID)
	if err != nil {
		return fmt.Errorf("invalid groupID format: %w", err)
	}

	// Revoke chat invite link using the Telegram Bot API
	revokeConfig := tgbotapi.RevokeChatInviteLinkConfig{
		ChatConfig: tgbotapi.ChatConfig{ChatID: groupIDInt},
		InviteLink: inviteLink,
	}

	result, err := bot.Request(revokeConfig)
	if err != nil {
		return fmt.Errorf("failed to revoke invite link: %w", err)
	}

	if !result.Ok {
		return fmt.Errorf("API error: %s", result.Description)
	}

	return nil
}
