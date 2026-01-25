package services

import (
	"testing"
)

func TestGenerateInviteLink_Success(t *testing.T) {
	// This test requires a mock bot or integration test
	// For now, we verify the function signature and logic
	t.Skip("Requires mock Telegram bot API")
}

func TestRevokeInviteLink_ChannelMode_NoOp(t *testing.T) {
	// Test that RevokeInviteLink handles channel mode correctly
	// In channel mode, revoke should be skipped (member_limit=1 provides security)
	t.Skip("Requires mock Telegram bot API")
}
