# Telegram Slash Commands with Role-Based Visibility

Feature documentation for implementing Telegram bot slash commands menu with role-based visibility.

## Overview

This feature registers bot commands with the Telegram Bot API, enabling users to see available commands in the Telegram app's menu when typing `/`. Commands are filtered based on user roles:
- **Public users**: See `/start` and `/restart`
- **Admin users**: See `/start`, `/restart`, and `/users`

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      cmd/app/main.go                        │
│                    Application Startup                      │
└─────────────────────────┬───────────────────────────────────┘
                          │
                          │ Calls RegisterCommands()
                          ▼
┌─────────────────────────────────────────────────────────────┐
│              internal/telegram/bot.go                       │
│                   Bot.RegisterCommands()                    │
└─────────────────────────┬───────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│         internal/telegram/command_service.go                │
│              CommandService.RegisterCommands()              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  1. Register public commands for all users           │  │
│  │  2. Register admin commands for each admin user      │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────┬───────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│              Telegram Bot API (External)                     │
│     setMyCommands with BotCommandScope                      │
└─────────────────────────────────────────────────────────────┘
```

## Implementation Details

### Domain Layer (`internal/domain/command/`)

**command.go**: Defines core types
```go
type Role string

const (
    RolePublic Role = "public"
    RoleAdmin  Role = "admin"
)

type Command struct {
    Name        string
    Description string
    Role        Role
}
```

**registry.go**: Central command registry
```go
var AllCommands = CommandSlice{
    {Name: "start", Description: "Начать работу с ботом", Role: RolePublic},
    {Name: "restart", Description: "Перезапустить бота", Role: RolePublic},
    {Name: "users", Description: "Список пользователей (админ)", Role: RoleAdmin},
}
```

### Service Layer (`internal/telegram/`)

**command_mapper.go**: Converts domain commands to Telegram API types
**command_service.go**: Handles registration with Telegram Bot API

### Integration

**bot.go**: Added `RegisterCommands(ctx)` method
**main.go**: Calls `bot.RegisterCommands(ctx)` at startup

## Command Scopes

| Scope Type | Usage | Description |
|------------|-------|-------------|
| `BotCommandScopeAllPrivateChats` | Public commands | All private chat users see these commands |
| `BotCommandScopeChat` | Admin commands | Per-user scope for specific admin users |

## Adding New Commands

### 1. Add Command to Registry

Edit `/internal/domain/command/registry.go`:

```go
var AllCommands = CommandSlice{
    // ... existing commands
    {Name: "newcommand", Description: "Описание команды", Role: RolePublic},
}
```

### 2. Implement Handler

Edit `/internal/telegram/handlers.go`:

```go
const (
    commandNew = "newcommand"
)

func (b *Bot) handleCommand(message *tgbotapi.Message) {
    switch message.Command() {
    // ... existing cases
    case commandNew:
        if err := b.newCommand(message); err != nil {
            log.Printf("Failed to handle /newcommand: %v", err)
        }
    }
}
```

### 3. Restart Bot

Commands are registered at startup. Restart the bot after changes:

```bash
# Systemd
sudo systemctl restart mispilka-bot

# Or direct
./bin/app
```

## Configuration

Admin users are configured via the `ADMIN_IDS` environment variable:

```bash
# .env
ADMIN_IDS=123456789,987654321
```

## Usage Examples

### Regular User View
```
User types: /
Menu shows:
  /start  - Начать работу с ботом
  /restart - Перезапустить бота
```

### Admin User View
```
Admin types: /
Menu shows:
  /start   - Начать работу с ботом
  /restart - Перезапустить бота
  /users   - Список пользователей (админ)
```

## Troubleshooting

### Commands Not Appearing

**Problem**: Users don't see command menu

**Solutions**:
1. Check bot logs for `[COMMANDS]` entries
2. Verify bot token has sufficient permissions
3. Ensure commands were registered successfully
4. Try restarting Telegram app (clear cache)

### Admin Commands Not Visible

**Problem**: Admin doesn't see admin-specific commands

**Solutions**:
1. Verify admin ID is in `ADMIN_IDS` environment variable
2. Check logs for `[COMMANDS] Registered admin commands for user ID`
3. Ensure you're using the correct Telegram account

### Registration Fails

**Problem**: Logs show "Failed to register commands"

**Impact**: Bot still works, users just can't see command menu. Commands can still be typed manually.

**Solution**: Check network connectivity to Telegram API

## Logging

Command registration uses `[COMMANDS]` prefix:

```
[COMMANDS] Registered 2 public commands
[COMMANDS] Registered admin commands for user 123456789
```

## Error Handling

The bot gracefully handles command registration failures:
- Failures are logged but don't crash the bot
- Users can still type commands manually
- Bot functions normally without the command menu

## Future Enhancements

1. **Dynamic Commands**: Load commands from database
2. **Multi-language**: Localized command descriptions
3. **Runtime Updates**: `/reloadcommands` admin command
4. **Analytics**: Track command usage statistics

## See Also

- [ADR: Telegram Slash Commands Decision](ADR--2026-01-24--10-15--telegram-slash-commands.md)
- [Telegram Bot API: setMyCommands](https://core.telegram.org/bots/api#setmycommands)
- [go-telegram-bot-api Documentation](https://github.com/go-telegram-bot-api/telegram-bot-api)
