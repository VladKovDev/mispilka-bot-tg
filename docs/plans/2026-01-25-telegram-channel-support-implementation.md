# Telegram Channel Support Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add support for Telegram private channels in addition to the existing private group functionality, allowing the bot to support both resource types via configuration without code changes.

**Architecture:** The bot will use a new `ResourceType` enum (`group` or `channel`) to conditionally enable/disable group-specific features (join/leave tracking via `chat_member` events, invite link revocation). The same `User` struct works for both modes - `JoinedGroup` and `JoinedAt` fields simply remain unused in channel mode. Invite link generation uses `member_limit=1` for both types, making explicit revocation unnecessary for channels.

**Tech Stack:** Go 1.22.2, go-telegram-bot-api/v5, godotenv for config

---

## Task 1: Add ResourceType Constants and Config Fields

**Files:**
- Modify: `config/config.go:12-24`
- Test: `config/config_test.go` (to be created)

**Step 1: Write the failing test**

Create file: `config/config_test.go`

```go
package config

import (
	"os"
	"testing"
)

func TestValidate_ValidConfigGroupMode(t *testing.T) {
	cfg := &Config{
		BotToken:                   "test_token",
		PrivateResourceID:          "123456789",
		PrivateResourceType:        ResourceTypeGroup,
		ProdamusSecret:             "test_secret",
		ProdamusAPIURL:             "https://test.com",
	}

	err := Validate(cfg)
	if err != nil {
		t.Errorf("Validate() returned unexpected error for valid group config: %v", err)
	}
}

func TestValidate_ValidConfigChannelMode(t *testing.T) {
	cfg := &Config{
		BotToken:                   "test_token",
		PrivateResourceID:          "123456789",
		PrivateResourceType:        ResourceTypeChannel,
		ProdamusSecret:             "test_secret",
		ProdamusAPIURL:             "https://test.com",
	}

	err := Validate(cfg)
	if err != nil {
		t.Errorf("Validate() returned unexpected error for valid channel config: %v", err)
	}
}

func TestValidate_MissingResourceId(t *testing.T) {
	cfg := &Config{
		BotToken:                   "test_token",
		PrivateResourceID:          "",
		PrivateResourceType:        ResourceTypeGroup,
		ProdamusSecret:             "test_secret",
		ProdamusAPIURL:             "https://test.com",
	}

	err := Validate(cfg)
	if err == nil {
		t.Error("Validate() should return error when PrivateResourceID is empty")
	}
}

func TestValidate_InvalidResourceType(t *testing.T) {
	cfg := &Config{
		BotToken:                   "test_token",
		PrivateResourceID:          "123456789",
		PrivateResourceType:        ResourceType("invalid"),
		ProdamusSecret:             "test_secret",
		ProdamusAPIURL:             "https://test.com",
	}

	err := Validate(cfg)
	if err == nil {
		t.Error("Validate() should return error for invalid ResourceType")
	}
}

func TestLoad_ChannelModeFromEnv(t *testing.T) {
	// Set test env vars
	os.Setenv("BOT_TOKEN", "test_token")
	os.Setenv("PRIVATE_RESOURCE_ID", "123456789")
	os.Setenv("PRIVATE_RESOURCE_TYPE", "channel")
	os.Setenv("PRODAMUS_SECRET_KEY", "test_secret")
	os.Setenv("PRODAMUS_API_URL", "https://test.com")
	defer func() {
		os.Unsetenv("BOT_TOKEN")
		os.Unsetenv("PRIVATE_RESOURCE_ID")
		os.Unsetenv("PRIVATE_RESOURCE_TYPE")
		os.Unsetenv("PRODAMUS_SECRET_KEY")
		os.Unsetenv("PRODAMUS_API_URL")
	}()

	cfg := Load()

	if cfg.PrivateResourceID != "123456789" {
		t.Errorf("Load() PrivateResourceID = %s, want %s", cfg.PrivateResourceID, "123456789")
	}
	if cfg.PrivateResourceType != ResourceTypeChannel {
		t.Errorf("Load() PrivateResourceType = %s, want %s", cfg.PrivateResourceType, ResourceTypeChannel)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./config -v`
Expected: FAIL with "undefined: ResourceType" and "Config struct has no field PrivateResourceID/Type"

**Step 3: Write minimal implementation**

In `config/config.go`, add ResourceType constants and update Config struct:

```go
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type ResourceType string

const (
	ResourceTypeGroup    ResourceType = "group"
	ResourceTypeChannel  ResourceType = "channel"
)

type Config struct {
	BotToken                   string
	PrivateResourceID          string        // renamed from PrivateGroupID
	PrivateResourceType        ResourceType  // new field
	AdminIDs                   []int64
	WebhookHost                string
	WebhookPort                string
	WebhookPath                string
	ProdamusSecret             string
	ProdamusAPIURL             string
	ProdamusProductName        string
	ProdamusProductPrice       string
	ProdamusProductPaidContent string
}
```

Update `Load()` function:

```go
func Load() *Config {
	if err := godotenv.Load(); err != nil {
		fmt.Printf("Error loading .env file: %v\n", err)
		panic(err)
	}

	resourceType := ResourceType(getEnv("PRIVATE_RESOURCE_TYPE", string(ResourceTypeGroup)))

	return &Config{
		BotToken:                   getEnv("BOT_TOKEN", ""),
		PrivateResourceID:          getEnv("PRIVATE_RESOURCE_ID", ""), // renamed from PRIVATE_GROUP_ID
		PrivateResourceType:        resourceType,                      // new field
		AdminIDs:                   parseAdminIDs(getEnv("ADMIN_IDS", "")),
		WebhookHost:                getEnv("WEBHOOK_HOST", "0.0.0.0"),
		WebhookPort:                getEnv("WEBHOOK_PORT", "8080"),
		WebhookPath:                getEnv("WEBHOOK_PATH", "/webhook/prodamus"),
		ProdamusSecret:             getEnv("PRODAMUS_SECRET_KEY", ""),
		ProdamusAPIURL:             getEnv("PRODAMUS_API_URL", ""),
		ProdamusProductName:        getEnv("PRODAMUS_PRODUCT_NAME", "Доступ к обучающим материалам"),
		ProdamusProductPrice:       getEnv("PRODAMUS_PRODUCT_PRICE", "500"),
		ProdamusProductPaidContent: getEnv("PRODAMUS_PRODUCT_PAID_CONTENT", "Успешно! Переходите обратно в бота и вступайте в нашу закрытую группу"),
	}
}
```

Update `Validate()` function:

```go
func Validate(cfg *Config) error {
	if cfg.BotToken == "" {
		return fmt.Errorf("BOT_TOKEN is required")
	}
	if cfg.ProdamusSecret == "" {
		return fmt.Errorf("PRODAMUS_SECRET_KEY is required")
	}
	if cfg.ProdamusAPIURL == "" {
		return fmt.Errorf("PRODAMUS_API_URL is required")
	}
	if cfg.PrivateResourceID == "" {
		return fmt.Errorf("PRIVATE_RESOURCE_ID is required")
	}
	if cfg.PrivateResourceType != ResourceTypeGroup && cfg.PrivateResourceType != ResourceTypeChannel {
		return fmt.Errorf("PRIVATE_RESOURCE_TYPE must be 'group' or 'channel', got: %s", cfg.PrivateResourceType)
	}
	return nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./config -v`
Expected: PASS

**Step 5: Commit**

```bash
git add config/config.go config/config_test.go
git commit -m "feat: add ResourceType enum and rename PrivateGroupID to PrivateResourceID

- Add ResourceType constants (group, channel)
- Rename PrivateGroupID field to PrivateResourceID
- Add PrivateResourceType field to Config
- Update Load() to read PRIVATE_RESOURCE_TYPE from env (defaults to 'group')
- Add validation for PrivateResourceID and PrivateResourceType
- Add comprehensive unit tests for config validation"
```

---

## Task 2: Update Bot Helper Methods for Mode Detection

**Files:**
- Modify: `internal/telegram/bot.go:17-31`
- Test: `internal/telegram/bot_mode_test.go` (to be created)

**Step 1: Write the failing test**

Create file: `internal/telegram/bot_mode_test.go`

```go
package telegram

import (
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"mispilkabot/config"
)

func TestIsGroupMode_GroupMode(t *testing.T) {
	cfg := &config.Config{
		PrivateResourceType: config.ResourceTypeGroup,
	}
	botAPI, _ := tgbotapi.NewBotAPI("test")
	bot := NewBot(botAPI, cfg)

	if !bot.IsGroupMode() {
		t.Error("IsGroupMode() should return true for group mode")
	}
	if bot.IsChannelMode() {
		t.Error("IsChannelMode() should return false for group mode")
	}
}

func TestIsChannelMode_ChannelMode(t *testing.T) {
	cfg := &config.Config{
		PrivateResourceType: config.ResourceTypeChannel,
	}
	botAPI, _ := tgbotapi.NewBotAPI("test")
	bot := NewBot(botAPI, cfg)

	if !bot.IsChannelMode() {
		t.Error("IsChannelMode() should return true for channel mode")
	}
	if bot.IsGroupMode() {
		t.Error("IsGroupMode() should return false for channel mode")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/telegram -v -run TestIsGroupMode`
Expected: FAIL with "bot.IsGroupMode undefined"

**Step 3: Write minimal implementation**

In `internal/telegram/bot.go`, add helper methods after the `NewBot` function (around line 31):

```go
func NewBot(bot *tgbotapi.BotAPI, cfg *config.Config) *Bot {
	return &Bot{
		bot:            bot,
		cfg:            cfg,
		commandService: NewCommandService(bot),
	}
}

// IsGroupMode returns true if the bot is configured for group mode
func (b *Bot) IsGroupMode() bool {
	return b.cfg.PrivateResourceType == config.ResourceTypeGroup
}

// IsChannelMode returns true if the bot is configured for channel mode
func (b *Bot) IsChannelMode() bool {
	return b.cfg.PrivateResourceType == config.ResourceTypeChannel
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/telegram -v -run TestIsGroupMode`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/telegram/bot.go internal/telegram/bot_mode_test.go
git commit -m "feat: add IsGroupMode and IsChannelMode helper methods to Bot

Add convenience methods for checking the current resource type configuration.
These methods make conditional logic more readable throughout the codebase."
```

---

## Task 3: Update handleUpdates to Conditionally Process Group-Only Events

**Files:**
- Modify: `internal/telegram/bot.go:87-148`

**Step 1: Write the failing test**

This is integration-level, so we'll verify manually. First, let's add conditional logic.

**Step 2: Run to verify current behavior**

Run: `make build` (should compile without issues)

**Step 3: Implement conditional event handling**

In `internal/telegram/bot.go`, update `handleUpdates` method:

```go
func (b *Bot) handleUpdates(ctx context.Context, updates tgbotapi.UpdatesChannel, privateChatID int64) {
	for {
		select {
		case <-ctx.Done():
			log.Println("Shutting down bot...")
			b.bot.StopReceivingUpdates()
			return
		case update, ok := <-updates:
			if !ok {
				log.Println("Updates channel closed")
				return
			}

			// Handle chat_member updates (group join tracking) - group mode only
			if update.ChatMember != nil && b.IsGroupMode() {
				b.handleChatMember(update.ChatMember, privateChatID)
				continue
			}

			// Handle my_chat_member updates (bot's own member status changes) - both modes
			if update.MyChatMember != nil {
				b.handleMyChatMember(update.MyChatMember, privateChatID)
				continue
			}

			chatID := update.FromChat().ID
			// Handle new_chat_members and left_chat_member messages - group mode only
			if chatID == privateChatID && update.Message != nil && b.IsGroupMode() {
				// Handle new chat members (users joining the group)
				if len(update.Message.NewChatMembers) > 0 {
					for _, newMember := range update.Message.NewChatMembers {
						// Skip bots
						if newMember.IsBot {
							continue
						}
						b.handleNewChatMemberMessage(&newMember, update.Message, privateChatID)
					}
				}
				// Handle left chat member (user leaving the group)
				if update.Message.LeftChatMember != nil {
					leftMember := update.Message.LeftChatMember
					// Skip bots
					if !leftMember.IsBot {
						b.handleLeftChatMemberMessage(leftMember, update.Message, privateChatID)
					}
				}
				continue
			}

			if update.CallbackQuery != nil {
				b.handleCallbackQuery(update.CallbackQuery)
			}

			if update.Message == nil {
				continue
			}

			if update.Message.IsCommand() {
				b.handleCommand(update.Message)
			}
		}
	}
}
```

The key changes are:
1. `update.ChatMember != nil && b.IsGroupMode()` - chat_member events only in group mode
2. `chatID == privateChatID && update.Message != nil && b.IsGroupMode()` - new/left chat member messages only in group mode

**Step 4: Verify build**

Run: `make build`
Expected: Builds successfully with no errors

**Step 5: Commit**

```bash
git add internal/telegram/bot.go
git commit -m "feat: conditionally process group-only events based on ResourceType

- chat_member updates: only process in group mode
- new_chat_members/left_chat_member messages: only process in group mode
- my_chat_member updates: process in both modes (for logging)
- Callbacks and commands: unchanged (work in both modes)

This allows the bot to work with channels where join/leave tracking
is not available or needed."
```

---

## Task 4: Update Webhook Handler to Use PrivateResourceID

**Files:**
- Modify: `internal/server/prodamus/webhook.go`

**Step 1: Update field names**

In `internal/server/prodamus/webhook.go`, rename `privateGroupID` to `privateResourceID`:

```go
type Handler struct {
	secretKey            string
	privateResourceID    string  // renamed from privateGroupID
	generateInviteLinkFn func(userID, groupID string) (string, error)
	sendInviteMessage    func(userID, inviteLink string)
	mu                   sync.Mutex
	wg                   sync.WaitGroup
}
```

**Step 2: Update setter method**

```go
// SetPrivateResourceID sets the private resource ID for invite link generation
// Accepts either a group ID or channel ID
func (h *Handler) SetPrivateResourceID(resourceID string) {
	h.privateResourceID = resourceID
}
```

**Step 3: Update usage in generateInviteLink**

```go
// generateInviteLink creates an invite link for the user
func (h *Handler) generateInviteLink(userID string) (string, error) {
	if h.privateResourceID == "" {
		return "", fmt.Errorf("PRIVATE_RESOURCE_ID not set")
	}
	if h.generateInviteLinkFn == nil {
		return "", fmt.Errorf("generateInviteLink callback not set")
	}

	inviteLink, err := h.generateInviteLinkFn(userID, h.privateResourceID)
	if err != nil {
		return "", fmt.Errorf("failed to generate invite link: %w", err)
	}

	return inviteLink, nil
}
```

**Step 4: Verify build**

Run: `make build`
Expected: Builds successfully

**Step 5: Update caller (cmd/app/main.go)**

First, check how the handler is configured:

```go
handler.SetPrivateGroupID(cfg.PrivateGroupID)  // old
```

Should become:

```go
handler.SetPrivateResourceID(cfg.PrivateResourceID)  // new
```

**Step 6: Commit**

```bash
git add internal/server/prodamus/webhook.go cmd/app/main.go
git commit -m "refactor: rename privateGroupID to privateResourceID in webhook handler

- Rename Handler.privateGroupID to privateResourceID
- Rename SetPrivateGroupID() to SetPrivateResourceID()
- Update cmd/app/main.go to use new method name
- Update internal references in generateInviteLink()"
```

---

## Task 5: Update All References to PrivateGroupID in Bot

**Files:**
- Modify: `internal/telegram/bot.go`
- Modify: `internal/telegram/handlers.go`

**Step 1: Find all occurrences**

Run: `grep -n "PrivateGroupID" internal/telegram/bot.go internal/telegram/handlers.go`

**Step 2: Update bot.go references**

In `internal/telegram/bot.go`:
- Line 72: `privateChatID, err := parseID(b.cfg.PrivateGroupID)` → `privateChatID, err := parseID(b.cfg.PrivateResourceID)`
- Line 436: `newInviteLink, err := b.GenerateInviteLink(userID, b.cfg.PrivateGroupID)` → `newInviteLink, err := b.GenerateInviteLink(userID, b.cfg.PrivateResourceID)`
- Line 505: `if err := b.RevokeInviteLink(b.cfg.PrivateGroupID, inviteLink)` → `if err := b.RevokeInviteLink(b.cfg.PrivateResourceID, inviteLink)`
- Line 541: `if err := b.RevokeInviteLink(b.cfg.PrivateGroupID, inviteLink)` → `if err := b.RevokeInviteLink(b.cfg.PrivateResourceID, inviteLink)`
- Line 599: `if err := b.RevokeInviteLink(b.cfg.PrivateGroupID, user.InviteLink)` → `if err := b.RevokeInviteLink(b.cfg.PrivateResourceID, user.InviteLink)`
- Line 644: `newInviteLink, err := b.GenerateInviteLink(userID, b.cfg.PrivateGroupID)` → `newInviteLink, err := b.GenerateInviteLink(userID, b.cfg.PrivateResourceID)`

**Step 3: Update handlers.go references**

In `internal/telegram/handlers.go`, the formatUser method displays group info. Update for clarity:

```go
// 7. Group/Channel info (joined date) - group mode only
if user.HasJoined() {
	if b.IsGroupMode() {
		sb.WriteString(fmt.Sprintf("Группа: вступил %s\n", user.GetJoinedAt().Format("02.01.2006 15:04")))
	} else {
		sb.WriteString(fmt.Sprintf("Канал: вступил %s\n", user.GetJoinedAt().Format("02.01.2006 15:04")))
	}
} else {
	if b.IsGroupMode() {
		sb.WriteString("Группа: не вступил\n")
	} else {
		sb.WriteString("Канал: не вступил\n")
	}
}
// Show invite link if exists (independent of join status)
if user.InviteLink != "" {
	sb.WriteString(fmt.Sprintf("Ссылка: %s\n", user.InviteLink))
}
```

**Step 4: Verify build**

Run: `make build`
Expected: Builds successfully

**Step 5: Commit**

```bash
git add internal/telegram/bot.go internal/telegram/handlers.go
git commit -m "refactor: update all PrivateGroupID references to PrivateResourceID

- Update bot.go: all references to PrivateGroupID now use PrivateResourceID
- Update handlers.go: formatUser displays 'Группа' or 'Канал' based on mode
- Maintains backward compatibility for group mode"
```

---

## Task 6: Update .env.example with New Variables

**Files:**
- Modify: `.env.example`

**Step 1: Update .env.example**

```bash
# Telegram Bot Configuration
BOT_TOKEN=your_telegram_bot_token_here
PRIVATE_RESOURCE_ID=your_private_group_or_channel_id_here
PRIVATE_RESOURCE_TYPE=group
ADMIN_IDS=123456789,987654321

# Webhook Server Configuration
WEBHOOK_HOST=0.0.0.0
WEBHOOK_PORT=8080
WEBHOOK_PATH=/webhook/prodamus

# Prodamus Payment Configuration
PRODAMUS_SECRET_KEY=your_prodamus_secret_key_here
PRODAMUS_API_URL=https://your-prodamus-api-url.com
PRODAMUS_PRODUCT_NAME=Доступ к обучающим материалам
PRODAMUS_PRODUCT_PRICE=500
PRODAMUS_PRODUCT_PAID_CONTENT=Успешно! Переходите обратно в бота и вступайте в нашу закрытую группу
```

**Step 2: Commit**

```bash
git add .env.example
git commit -m "docs: update .env.example with new resource configuration variables

- Replace PRIVATE_GROUP_ID with PRIVATE_RESOURCE_ID
- Add PRIVATE_RESOURCE_TYPE with default value 'group'
- Update comments to reflect group/channel support"
```

---

## Task 7: Add Unit Tests for Invite Service

**Files:**
- Create: `internal/services/invite_test.go`

**Step 1: Write the test**

```go
package services

import (
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"mispilkabot/config"
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
```

Note: Full testing would require mocking the Telegram Bot API. For now, the test file is created as a placeholder.

**Step 2: Run test**

Run: `go test ./internal/services -v -run TestRevoke`
Expected: SKIP (tests are placeholders)

**Step 3: Commit**

```bash
git add internal/services/invite_test.go
git commit -m "test: add placeholder tests for invite link service

Tests are currently skipped as they require mocking the Telegram Bot API.
Added as placeholders for future test implementation."
```

---

## Task 8: Full Build and Verification

**Step 1: Run full build**

Run: `make build`
Expected: Builds successfully with no errors

**Step 2: Run all tests**

Run: `go test ./... -v`
Expected: All tests pass (some may be skipped)

**Step 3: Create migration guide**

Create file: `docs/CHANNEL_MIGRATION.md`:

```markdown
# Channel Support Migration Guide

## For Existing Users (Group Mode)

If you're currently using the bot with a private group, you need to update your `.env` file:

### Before:
```bash
PRIVATE_GROUP_ID=123456789
```

### After:
```bash
PRIVATE_RESOURCE_ID=123456789
PRIVATE_RESOURCE_TYPE=group
```

## For New Users (Channel Mode)

To use the bot with a private channel:

```bash
PRIVATE_RESOURCE_ID=123456789
PRIVATE_RESOURCE_TYPE=channel
```

## Key Differences

| Feature | Group Mode | Channel Mode |
|---------|------------|--------------|
| Join/leave tracking | ✅ Tracked via chat_member events | ❌ Not tracked |
| Invite link revocation | ✅ Explicit revoke after join | ❌ Not needed (member_limit=1) |
| JoinedGroup field | Set to true after join | Stays false |
| JoinedAt field | Set to join timestamp | nil |

## Testing

After updating your configuration:

1. Restart the bot
2. For group mode: verify join/leave still works
3. For channel mode: verify users can join via invite link
```

**Step 4: Commit**

```bash
git add docs/CHANNEL_MIGRATION.md
git commit -m "docs: add channel support migration guide

Provides instructions for existing users to migrate from PRIVATE_GROUP_ID
to PRIVATE_RESOURCE_ID, and explains differences between group and channel modes."
```

---

## Task 9: Final Integration Test - Group Mode (Regression)

**Step 1: Set up group mode environment**

Create `.env` for testing:

```bash
BOT_TOKEN=your_test_token
PRIVATE_RESOURCE_ID=your_test_group_id
PRIVATE_RESOURCE_TYPE=group
ADMIN_IDS=your_admin_id
PRODAMUS_SECRET_KEY=test_secret
PRODAMUS_API_URL=https://test-url.com
```

**Step 2: Build and run**

```bash
make build
make run-dev
```

**Step 3: Verify group mode behavior**

1. Send `/start` to bot - should receive welcome message
2. Click accept button - should receive payment link
3. (Simulate payment) - should receive group invite link
4. Join group via invite link - `JoinedGroup` should be set to true
5. Leave group - should receive new invite link (if paid)

**Step 4: Check logs for proper mode detection**

Logs should show:
- `IsGroupMode()` returning true
- `chat_member` events being processed
- Invite links being revoked after join

---

## Task 10: Final Integration Test - Channel Mode (New Feature)

**Step 1: Update environment for channel mode**

```bash
PRIVATE_RESOURCE_ID=your_test_channel_id
PRIVATE_RESOURCE_TYPE=channel
```

**Step 2: Build and run**

```bash
make build
make run-dev
```

**Step 3: Verify channel mode behavior**

1. Send `/start` to bot - should receive welcome message
2. Click accept button - should receive payment link
3. (Simulate payment) - should receive channel invite link
4. Join channel via invite link - link should work (member_limit=1)
5. No `chat_member` events should be processed (check logs)

**Step 4: Verify channel-specific behavior**

- Invite links should NOT be revoked (no revoke API calls in logs)
- `JoinedGroup` should stay false (not used in channel mode)
- Users can still receive invite links and join successfully

**Step 5: Final commit**

```bash
git add docs/plans/2026-01-25-telegram-channel-support-implementation.md
git commit -m "docs: add complete implementation plan for channel support

Comprehensive plan covering all tasks from config changes through
testing and migration guide. Ready for execution."
```

---

## Summary of Changes

| File | Change Type | Description |
|------|-------------|-------------|
| `config/config.go` | Modify | Add ResourceType, rename PrivateGroupID → PrivateResourceID, add validation |
| `config/config_test.go` | Create | Add comprehensive config validation tests |
| `internal/telegram/bot.go` | Modify | Add IsGroupMode/IsChannelMode, conditionally process events, update all PrivateGroupID refs |
| `internal/telegram/bot_mode_test.go` | Create | Add mode detection tests |
| `internal/telegram/handlers.go` | Modify | Update formatUser for group/channel display |
| `internal/server/prodamus/webhook.go` | Modify | Rename privateGroupID → privateResourceID |
| `cmd/app/main.go` | Modify | Update handler setter call |
| `.env.example` | Modify | Update env var names and add PRIVATE_RESOURCE_TYPE |
| `internal/services/invite_test.go` | Create | Add placeholder tests for invite service |
| `docs/CHANNEL_MIGRATION.md` | Create | Add migration guide for users |

## Testing Checklist

- [ ] Unit tests pass (`go test ./... -v`)
- [ ] Build succeeds (`make build`)
- [ ] Group mode regression test passes
- [ ] Channel mode integration test passes
- [ ] Invite links work for both group and channel
- [ ] Payment flow works for both modes
- [ ] `/users` command displays correct info for both modes
