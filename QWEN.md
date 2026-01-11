# Mispilka Bot - Telegram Bot Project

## Project Overview

Mispilka Bot is a Telegram bot built in Go that delivers scheduled messages to users. The bot supports:

- User registration via `/start` command
- Scheduled message delivery with configurable timing
- Rich message content (text, photos, inline URL/callback buttons)
- User opt-in/opt-out for messaging via interactive buttons
- Group chat support with special handling for a private admin group
- Message queue management per user
- Prodamus payment integration for access to a private group
- Invite link generation and automatic revocation after use

### Technology Stack

- **Language:** Go 1.22.2
- **Telegram API:** `github.com/go-telegram-bot-api/telegram-bot-api/v5`
- **Configuration:** `github.com/joho/godotenv` for environment variables
- **Storage:** JSON files for data persistence
- **HTTP Server:** Standard library `net/http` for webhook handling

### Architecture

```
mispilka-bot-tg/
├── cmd/app/main.go          # Application entry point
├── config/                  # Configuration package
│   └── config.go           # Environment variable loading and validation
├── internal/
│   ├── telegram/
│   │   ├── bot.go           # Main bot initialization and message handling
│   │   └── handlers.go      # Command handlers (/start, /restart)
│   ├── server/
│   │   ├── server.go        # HTTP server for webhooks
│   │   └── prodamus/
│   │       └── webhook.go   # Prodamus webhook handler
│   ├── repository/          # (Empty - reserved for future use)
│   └── services/
│       ├── payment/         # (Payment services)
│       ├── invite.go        # Invite link generation and revocation
│       ├── messages.go      # Message content and timing management
│       ├── prodamus.go      # Prodamus payment client
│       ├── scheduler.go     # Scheduling logic for message delivery
│       ├── storage.go       # JSON read/write utilities with retry logic
│       └── users.go         # User data management
├── data/
│   ├── messages/            # Message text templates (Markdown)
│   ├── commands.json        # Command definitions (currently empty)
│   ├── messages.json        # Message queue and metadata configuration
│   ├── schedule_backup.json # Scheduled task persistence
│   └── users.json           # User database
├── assets/
│   └── images/              # Message images (PNG format)
└── Makefile                 # Build automation
```

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

### Run in Development Mode

```bash
make run-dev
```

Runs the bot directly with `go run` without creating a binary.

### Manual Execution

```bash
# Build
go build -o ./.bin/bot cmd/app/main.go

# Run
./.bin/bot
```

## Environment Variables

The bot requires a `.env` file with the following variables:

### Required

- `BOT_TOKEN` - Telegram bot token from BotFather
- `PRODAMUS_SECRET_KEY` - Prodamus webhook signature verification key
- `PRODAMUS_API_URL` - Prodamus payment API URL (e.g., https://demo.payform.ru/)

### Optional (with defaults)

- `PRIVATE_GROUP_ID` - Private group chat ID for invite link generation
- `WEBHOOK_HOST` - HTTP server host (default: 0.0.0.0)
- `WEBHOOK_PORT` - HTTP server port (default: 8080)
- `WEBHOOK_PATH` - Webhook endpoint path (default: /webhook/prodamus)
- `PRODAMUS_PRODUCT_NAME` - Product name for payment (default: "Обучающие материалы")
- `PRODAMUS_PRODUCT_PRICE` - Product price in RUB (default: "500")
- `PRODAMUS_PRODUCT_PAID_CONTENT` - Success message after payment

## Development Conventions

### Project Structure

- **cmd/app/** - Entry point only; minimal logic
- **internal/** - Private application code, not intended for external use
- **config/** - Configuration handling
- **data/** - Runtime data storage (JSON files, not committed to git)
- **assets/** - Static assets (images, documents)

### Code Organization

- **telegram package** - Handles Telegram-specific logic: update handling, message sending, callback queries
- **server package** - HTTP server for receiving webhooks (Prodamus payment notifications)
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
      "timing": {"hours": 1, "minutes": 30},
      "template_file": "custom.md",
      "inline_keyboard": {
        "rows": [
          {
            "buttons": [
              {"type": "url", "text": "Button", "url": "https://example.com"},
              {"type": "callback", "text": "Click", "callback_data": "action"}
            ]
          }
        ]
      }
    }
  }
}
```

Message content files (Markdown) are expected in `data/messages/{message_name}.md` or custom template file.

Images are expected in `assets/images/{message_name}.PNG` (uppercase PNG extension).

**Placeholders supported in message text and buttons:**
- `{{payment_price}}` - Product price from config
- `{{payment_link}}` - User's unique payment link
- `{{invite_link}}` - User's unique group invite link (for group_invite message)

**Special Messages:**
- `start` - Initial message sent after /start command
- `group_invite` - Sent after successful payment with invite link placeholder

### Error Handling

- Logging uses `log.Printf` for errors
- Functions return errors for caller handling
- Retry logic implemented for JSON file operations
- Missing resources (e.g., photos) are handled gracefully

### Commands

- `/start` - Initializes user interaction with message queue setup; sends appropriate keyboard based on messaging status
- `/restart` - Re-registers the user (resets message queue)

### Callback Queries

- `accept` - User opts in to receive messages; generates payment link via Prodamus
- `decline` - User opts out (optional, can be enabled by uncommenting handlers)

### Private Group Handling

The `PRIVATE_GROUP_ID` environment variable specifies the private group for paid users:
- Payment webhook generates unique invite links for this group
- `chat_member` updates detect when user joins and revoke invite link
- Used for granting group access after successful payment

### Prodamus Payment Integration

**Payment Link Generation:**
- Uses Prodamus REST API via GET request
- Generates unique payment links with order_id mapping
- Stores payment link in user data for use in messages

**Webhook Endpoint:** `http://{WEBHOOK_HOST}:{WEBHOOK_PORT}{WEBHOOK_PATH}` (default: http://0.0.0.0:8080/webhook/prodamus)

**Webhook Processing:**
- Receives POST requests with payment data (JSON format)
- Verifies HMAC-SHA256 signature using `PRODAMUS_SECRET_KEY`
- Processes successful payments (`status: success` or `paid`)
- Generates unique Telegram group invite links via `createChatInviteLink`
- Stores invite link in `users.json`
- Sends `group_invite` message with `{{invite_link}}` placeholder
- Revokes invite link after user joins (tracked via `chat_member` update)
- Logs all request data: headers, query params, body
- Always returns 200 OK to prevent retry loops

**See:** `internal/server/README.md` for detailed webhook configuration and usage

### User Data Model

```go
type User struct {
    UserName     string     `json:"user_name"`
    RegTime      time.Time  `json:"reg_time"`
    IsMessaging  bool       `json:"is_messaging"`
    MessagesList []string   `json:"messages_list"`
    PaymentDate  *time.Time `json:"payment_date,omitempty"`
    PaymentLink  string     `json:"payment_link,omitempty"`
    InviteLink   string     `json:"invite_link,omitempty"`
    JoinedGroup  bool       `json:"joined_group,omitempty"`
    JoinedAt     *time.Time `json:"joined_at,omitempty"`
}
```

Pointer fields (`PaymentDate`, `JoinedAt`) use `nil` to represent "not set" and avoid zero-time serialization issues.

## Testing

No automated tests are currently present in the project. Manual testing is performed by:
1. Starting the bot with a valid `BOT_TOKEN`
2. Sending `/start` command in a private chat
3. Verifying message queue setup
4. Testing callback button interactions (accept/decline)
5. Monitoring scheduled message delivery
6. Testing Prodamus webhook integration (requires payment system setup)
7. Verifying invite link generation and group access

## Important Notes

- The `internal/repository` directory is currently empty and reserved for future database abstraction
- Message queue is processed from the end (LIFO behavior for immediate consumption)
- Keyboard buttons with unresolved placeholders (`{{...}}`) are filtered out to prevent invalid URLs
- Graceful shutdown is supported via SIGINT/SIGTERM signals
- Both HTTP server and Telegram bot run concurrently in separate goroutines
