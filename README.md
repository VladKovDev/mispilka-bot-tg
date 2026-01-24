# Mispilka Bot

A Telegram bot for managing paid access to private groups using Prodamus payment processing.

## Technical Stack

- **Language**: Go 1.22.2
- **Telegram API**: [telegram-bot-api/v5](https://github.com/go-telegram-bot-api/telegram-bot-api/v5)
- **Payment**: Prodamus payment processor (webhook-based)
- **Storage**: JSON files (`data/users.json`, `data/messages.json`, `data/schedule_backup.json`)
- **Configuration**: Environment variables via `.env` file

## Quick Start

### Prerequisites

- Go 1.22.2 or later
- Telegram bot token from [@BotFather](https://t.me/botfather)
- Prodamus account (for payment processing)

### Setup

1. **Clone the repository:**

   ```bash
   git clone <repo-url>
   cd mispilka-bot-tg
   ```

2. **Install Go dependencies:**

   ```bash
   go mod download
   ```

3. **Configure environment:**

   ```bash
   cp .env.example .env
   # Edit .env with your values (see Configuration section below)
   ```

4. **Run the bot:**

   ```bash
   make run-dev
   ```

   Or for production build:

   ```bash
   make build && make run
   ```

The bot will start listening for Telegram updates and the HTTP webhook server will begin accepting Prodamus payment callbacks.

## Configuration

Required environment variables in `.env`:

```bash
BOT_TOKEN=              # Telegram bot token (required)
PRIVATE_GROUP_ID=       # Private Telegram group ID (required)
ADMIN_IDS=              # Comma-separated admin Telegram IDs
PRODAMUS_SECRET_KEY=    # Prodamus webhook secret (required)
PRODAMUS_API_URL=       # Prodamus API URL (required)
PRODAMUS_PRODUCT_NAME=  # Product name (default: "Доступ к обучающим материалам")
PRODAMUS_PRODUCT_PRICE= # Price (default: "500")
WEBHOOK_HOST=           # HTTP server host (default: "0.0.0.0")
WEBHOOK_PORT=           # HTTP server port (default: "8080")
WEBHOOK_PATH=           # Webhook path (default: "/webhook/prodamus")
```

## Architecture

```
┌─────────────────┐
│   cmd/app/main  │  Entry point: wires components, graceful shutdown
└────────┬────────┘
         │
    ┌────┴────┐
    │         │
┌───▼────┐  ┌─▼──────────┐
│  Bot   │  │ HTTP Server│
│(tg/)   │  │ (server/)  │
└───┬────┘  └─┬──────────┘
    │          │
    │      ┌───▼──────────────┐
    │      │ Prodamus Webhook │
    │      │ (Payment Handler)│
    │      └───────────────────┘
    │
┌───▼───────────────────────────┐
│         Services              │
│  ┌────────┐  ┌──────────────┐ │
│  │ Users  │  │   Messages   │ │
│  └────────┘  └──────────────┘ │
│  ┌────────┐  ┌──────────────┐ │
│  │Invite  │  │  Scheduler   │ │
│  └────────┘  └──────────────┘ │
│  ┌────────┐                   │
│  │Payment │                   │
│  └────────┘                   │
└────────────┬──────────────────┘
             │
        ┌────▼────┐
        │ JSON    │
        │ Storage │
        │(data/)  │
        └─────────┘
```

### How It Works

1. **User onboarding**: `/start` → creates user → sends welcome with "Accept terms" button
2. **Accept terms**: User clicks → generates Prodamus payment link → starts message scheduling
3. **Payment**: Prodamus webhook → generates invite link → sends group invite message
4. **Group access**: User joins → `chat_member` update → marks `JoinedGroup=true` → revokes invite link
5. **Scheduled messages**: Background scheduler sends messages from user's queue
6. **Group leave**: User leaves → generates new invite link (if paid) → sends via DM

### Key Components

- **`internal/telegram/bot.go`** - Main bot logic, update handling, group join/leave tracking
- **`internal/telegram/handlers.go`** - Command handlers (`/start`, `/restart`, `/users`)
- **`internal/domain/command/registry.go`** - Central command registry with role-based visibility
- **`internal/server/prodamus/webhook.go`** - Payment webhook handler
- **`internal/services/*.go`** - User management, invites, messages, scheduling, storage

## Code Exploration Path

Recommended reading order to understand the codebase:

1. `cmd/app/main.go` - See how everything connects
2. `internal/telegram/bot.go` - Core bot logic and update handling
3. `internal/telegram/handlers.go` - Command implementation
4. `internal/services/users.go` - User data management
5. `internal/domain/command/registry.go` - How commands are registered

## Further Documentation

- **`CLAUDE.md`** - Comprehensive guide for Claude Code and developers working on this codebase
- **`docs/features/ADR--2026-01-24--10-15--telegram-slash-commands.md`** - Architecture decision on command system
- **`.env.example`** - Complete configuration reference

## Troubleshooting

**Bot doesn't respond to commands:**

- Verify `BOT_TOKEN` is correct and bot is running
- Check that the bot has been started in Telegram: send `/start` to your bot
- Enable debug logging: `BOT_DEBUG=true make run-dev`

**Webhook not receiving payments:**

- Verify `PRODAMUS_SECRET_KEY` matches your Prodamus account settings
- Check `WEBHOOK_HOST` and `WEBHOOK_PORT` - ensure the server is publicly accessible
- Check logs for webhook errors: `[PAYMENT_ERROR]` entries indicate issues

**Invite links not working:**

- Verify `PRIVATE_GROUP_ID` is correct (use numeric ID, not username)
- Ensure the bot is an administrator in the private group
- Check bot has permission to create invite links

**Users not joining group after payment:**

- Check `data/users.json` to see if user has `PaymentDate` set
- Verify invite link was generated (check `InviteLink` field)
- Ensure user is clicking the correct invite link sent by the bot

## Development Commands

```bash
# Build the bot
make build

# Run development mode
make run-dev

# Run with debug logging
BOT_DEBUG=true make run-dev

# Run production build
make run
```