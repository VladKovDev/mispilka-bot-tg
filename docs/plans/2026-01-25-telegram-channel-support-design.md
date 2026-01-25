# Telegram Channel Support Design

**Date:** 2026-01-25
**Status:** Design
**Author:** Claude (brainstorming session)

## Overview

Add support for Telegram private channels in addition to the existing private group functionality. The bot should support both resource types (group/channel) via configuration without code changes.

## Key Differences: Group vs Channel

| Aspect | Group Mode | Channel Mode |
|--------|------------|--------------|
| Invite link creation | `member_limit=1` | `member_limit=1` |
| Invite link revocation | ✅ Explicit revoke after join | ❌ Not needed (member_limit sufficient) |
| Join/leave tracking | ✅ Tracked via `chat_member` events | ❌ Not tracked |
| `JoinedGroup` field | `true` after join | `false` (not used) |
| `JoinedAt` field | Set to join timestamp | `nil` (not used) |
| Payment flow | Same as before | Same as before |

## Configuration

### Environment Variables

Replace `PRIVATE_GROUP_ID` with:

```bash
PRIVATE_RESOURCE_ID=123456789     # ID группы или канала
PRIVATE_RESOURCE_TYPE=group        # 'group' или 'channel'
```

### Config Changes (`config/config.go`)

```go
type ResourceType string

const (
    ResourceTypeGroup    ResourceType = "group"
    ResourceTypeChannel  ResourceType = "channel"
)

type Config struct {
    BotToken                   string
    PrivateResourceID          string        // renamed from PrivateGroupID
    PrivateResourceType        ResourceType  // new field
    // ... other fields unchanged
}
```

### Validation

```go
func Validate(cfg *Config) error {
    // ... existing validations ...

    if cfg.PrivateResourceID == "" {
        return fmt.Errorf("PRIVATE_RESOURCE_ID is required")
    }
    if cfg.PrivateResourceType != ResourceTypeGroup && cfg.PrivateResourceType != ResourceTypeChannel {
        return fmt.Errorf("PRIVATE_RESOURCE_TYPE must be 'group' or 'channel', got: %s", cfg.PrivateResourceType)
    }

    return nil
}
```

## Data Model

### User Structure (`internal/services/users.go`)

No changes to the `User` struct — existing fields work for both modes:

```go
type User struct {
    UserName     string                 `json:"user_name"`
    RegTime      time.Time              `json:"reg_time"`
    IsMessaging  bool                   `json:"is_messaging"`
    MessagesList []string               `json:"messages_list"`
    PaymentDate  *time.Time             `json:"payment_date,omitempty"`
    PaymentLink  string                 `json:"payment_link,omitempty"`
    InviteLink   string                 `json:"invite_link,omitempty"`    // works for both
    JoinedGroup  bool                   `json:"joined_group,omitempty"`   // group mode only
    JoinedAt     *time.Time             `json:"joined_at,omitempty"`      // group mode only
    PaymentInfo  *models.WebhookPayload `json:"payment_info,omitempty"`
}
```

### Field Population by Mode

| Field | Group Mode | Channel Mode |
|-------|------------|--------------|
| `InviteLink` | Created and stored | Created and stored |
| `JoinedGroup` | `true` after join | `false` (never set) |
| `JoinedAt` | Join timestamp | `nil` (never set) |

## Bot Logic Changes

### Helper Methods (`internal/telegram/bot.go`)

```go
func (b *Bot) IsGroupMode() bool {
    return b.cfg.PrivateResourceType == ResourceTypeGroup
}

func (b *Bot) IsChannelMode() bool {
    return b.cfg.PrivateResourceType == ResourceTypeChannel
}
```

### Update Handling (`handleUpdates`)

```go
func (b *Bot) handleUpdates(ctx context.Context, updates tgbotapi.UpdatesChannel, privateChatID int64) {
    for {
        select {
        case <-ctx.Done():
            // ...
        case update := <-updates:
            // chat_member events - group mode only
            if update.ChatMember != nil && b.IsGroupMode() {
                b.handleChatMember(update.ChatMember, privateChatID)
                continue
            }

            // my_chat_member - log for both modes
            if update.MyChatMember != nil {
                b.handleMyChatMember(update.MyChatMember, privateChatID)
                continue
            }

            // new_chat_members / left_chat_member - group mode only
            chatID := update.FromChat().ID
            if chatID == privateChatID && update.Message != nil && b.IsGroupMode() {
                // existing group join/leave logic...
            }

            // callbacks and commands - unchanged
            // ...
        }
    }
}
```

### Behavior Matrix

| Method/Handler | Group Mode | Channel Mode |
|----------------|------------|--------------|
| `handleChatMember` | ✅ executes | ❌ skipped |
| `handleNewChatMemberMessage` | ✅ executes | ❌ skipped |
| `handleLeftChatMemberMessage` | ✅ executes | ❌ skipped |
| `acceptCallback` (payment) | `IsMessaging = false` | `IsMessaging = false` |
| `SendInviteMessage` | Sends invite link | Sends invite link |
| Scheduled messages | Stop after payment | Stop after payment |

## Invite Link Service Changes

### GenerateInviteLink (`internal/services/invite.go`)

No changes required — `createChatInviteLink` with `member_limit=1` works for both.

### RevokeInviteLink (`internal/services/invite.go`)

Add conditional logic:

```go
func RevokeInviteLink(chatID, inviteLink string, bot *tgbotapi.BotAPI, resourceType ResourceType) error {
    // For channels, skip revoke - member_limit=1 provides security
    if resourceType == ResourceTypeChannel {
        log.Printf("[INVITE] Skipping revoke for channel mode (member_limit=1 provides security)")
        return nil
    }

    // For groups, revoke as before
    // ... existing revoke logic ...
}
```

Note: `RevokeInviteLink` calls are already inside `b.IsGroupMode()` blocks, so channel mode never reaches them.

## Error Handling

### Admin Notifications on Invite Link Failure

```go
link, err := b.GenerateInviteLink(userID, b.cfg.PrivateResourceID)
if err != nil {
    log.Printf("[INVITE_ERROR] Failed to create invite link for user %s: %v", userID, err)

    // Notify admins
    for _, adminID := range b.cfg.AdminIDs {
        msg := tgbotapi.NewMessage(adminID,
            fmt.Sprintf("⚠️ Ошибка создания ссылки для пользователя %s: %v", userID, err))
        b.bot.Send(msg)
    }
    return
}
```

## Unit Tests

### Test Files to Create

**`config/config_test.go`**
- `TestValidate_ValidConfig` — all fields correct
- `TestValidate_MissingResourceId` — error when ID missing
- `TestValidate_InvalidResourceType` — error on invalid type

**`internal/telegram/bot_mode_test.go`**
- `TestIsGroupMode` — returns true for group
- `TestIsChannelMode` — returns true for channel

**`internal/services/invite_test.go`**
- `TestGenerateInviteLink_Success` — creates link
- `TestRevokeInviteLink_ChannelMode_NoOp` — doesn't call API for channel

## Implementation Checklist

- [ ] Update `config/config.go` with new fields and validation
- [ ] Add `ResourceType` constants
- [ ] Update `.env.example` with new variables
- [ ] Add `IsGroupMode()` / `IsChannelMode()` to Bot
- [ ] Update `handleUpdates` to conditionally process chat_member events
- [ ] Update `RevokeInviteLink` signature to accept `ResourceType`
- [ ] Add unit tests
- [ ] Test group mode (regression)
- [ ] Test channel mode (new functionality)

## Files to Modify

1. `config/config.go` — new fields, validation
2. `internal/telegram/bot.go` — mode helpers, conditional event handling
3. `internal/services/invite.go` — conditional revoke logic
4. `.env.example` — new environment variables

## Notes

- **Telegram Bot API**: `createChatInviteLink` and `revokeChatInviteLink` work for both chats and channels
- **Channel invite links**: `member_limit=1` automatically invalidates after first use, making explicit revoke unnecessary
- **No breaking changes**: Existing users can migrate by updating `.env` file
