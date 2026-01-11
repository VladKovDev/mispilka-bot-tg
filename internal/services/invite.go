package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
)

// GenerateInviteLink creates a new invite link for a chat/group with member limit set to 1
// Returns the invite link URL or an error if generation fails
func GenerateInviteLink(userID, groupID string, botToken string) (string, error) {
	groupIDInt, err := strconv.ParseInt(groupID, 10, 64)
	if err != nil {
		return "", fmt.Errorf("invalid groupID format: %w", err)
	}

	// Create request body for createChatInviteLink
	requestBody := map[string]interface{}{
		"chat_id":      groupIDInt,
		"member_limit": 1,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Make HTTP request to Telegram Bot API
	url := fmt.Sprintf("https://api.telegram.org/bot%s/createChatInviteLink", botToken)
	log.Printf("[DEBUG] Creating invite link: url=%q, body=%s", url, string(jsonBody))
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create invite link: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	log.Printf("[DEBUG] Telegram API response: %s", string(body))

	var result struct {
		Ok     bool `json:"ok"`
		Result struct {
			InviteLink string `json:"invite_link"`
		} `json:"result"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if !result.Ok {
		return "", fmt.Errorf("API error: ok=false")
	}

	log.Printf("[DEBUG] Invite link created successfully: %q", result.Result.InviteLink)
	return result.Result.InviteLink, nil
}

// RevokeInviteLink revokes an existing invite link for a chat
func RevokeInviteLink(inviteLink string, botToken string) error {
	// Create request body for revokeChatInviteLink
	requestBody := map[string]interface{}{
		"invite_link": inviteLink,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// Make HTTP request to Telegram Bot API
	url := fmt.Sprintf("https://api.telegram.org/bot%s/revokeChatInviteLink", botToken)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to revoke invite link: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	var result struct {
		Ok          bool   `json:"ok"`
		Description string `json:"description"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if !result.Ok {
		return fmt.Errorf("API error: %s", result.Description)
	}

	return nil
}
