# Mispilka Bot - Telegram Bot Project

## Project Overview

Mispilka Bot is a Telegram bot built in Go that delivers scheduled messages to users. The bot supports:

- User registration via `/start` command
- Scheduled message delivery with configurable timing
- Rich message content (text, photos, inline URL buttons)
- User opt-in/opt-out for messaging via interactive buttons
- Group chat support with special handling for a private admin group
- Message queue management per user

### Technology Stack

- **Language:** Go 1.22.2
- **Telegram API:** `github.com/go-telegram-bot-api/telegram-bot-api/v5`
- **Configuration:** `github.com/joho/godotenv` for environment variables
- **Storage:** JSON files for data persistence

### Architecture

```
mispilka-bot-tg/
├── cmd/app/main.go          # Application entry point
├── config/                  # Configuration package (minimal)
├── internal/
│   ├── telegram/
│   │   ├── bot.go           # Main bot initialization and message handling
│   │   └── handlers.go      # Command handlers (/start, /restart)
│   └── services/
│       ├── messages.go      # Message content and timing management
│       ├── scheduler.go     # Scheduling logic for message delivery
│       ├── storage.go       # JSON read/write utilities with retry logic
│       └── users.go         # User data management
├── data/
│   ├── commands.json        # Command definitions (currently empty)
│   ├── messages.json        # Message queue and metadata configuration
│   ├── schedule_backup.json # Scheduled task persistence
│   └── users.json           # User database
└── Makefile                 # Build automation
```

### Environment Variables

The bot requires a `.env` file with the following variables:

- `BOT_TOKEN` - Telegram bot token from BotFather
- `PRIVATE_GROUP_ID` - Admin group chat ID for special handling

## Building and Running

### Build

```bash
make build
```

This compiles the application and places the binary at `./.bin/bot`.

### Run

```bash
make run
```

This builds and runs the bot. The bot will start receiving and processing Telegram updates.

### Manual Execution

```bash
# Build
go build -o ./.bin/bot cmd/app/main.go

# Run
./.bin/bot
```

## Development Conventions

### Project Structure

- **cmd/app/** - Entry point only; minimal logic
- **internal/** - Private application code, not intended for external use
- **config/** - Configuration handling (currently minimal, may expand)
- **data/** - Runtime data storage (JSON files, not committed to git)

### Code Organization

- **telegram package** - Handles Telegram-specific logic: update handling, message sending, callback queries
- **services package** - Business logic and data persistence: user management, message scheduling, storage operations

### Storage Pattern

- All data stored in JSON files in `data/` directory
- `storage.go` provides generic JSON read/write functions with retry logic (3 attempts)
- Storage files are automatically created on first run if missing
- `CheckStorage()` function ensures storage files exist at bot startup

### Message System

Messages are configured in `data/messages.json` with the structure:

```json
{
  "messages_list": ["message1", "message2", ...],
  "messages": {
    "message1": {
      "timing": [hours, minutes],
      "url_button": ["url", "button_text"]
    }
  }
}
```

Message content files (Markdown) are expected in `data/messages/{message_name}.md`.

Images are expected in `assets/images/{message_name}.PNG`.

### Error Handling

- Logging uses `log.Printf` for errors
- Functions return errors for caller handling
- Retry logic implemented for JSON file operations
- Missing resources (e.g., photos) are handled gracefully

### Commands

- `/start` - Initializes user interaction with message queue setup
- `/restart` - Re-registers the user (resets message queue)

### Callback Queries

- `accept` - User opts in to receive messages
- `decline` - User opts out (commented out in current implementation)

### Private Group Handling

The `PRIVATE_GROUP_ID` environment variable specifies an admin group where:
- New member joins disable messaging for those users
- Group messages are processed differently (special handling logic)

## Testing

No automated tests are currently present in the project. Manual testing is performed by:
1. Starting the bot with a valid `BOT_TOKEN`
2. Sending `/start` command in a private chat
3. Verifying message queue setup
4. Testing callback button interactions
5. Monitoring scheduled message delivery
