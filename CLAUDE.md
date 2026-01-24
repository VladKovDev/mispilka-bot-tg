# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**Mispilka Bot** - A Telegram bot for managing paid access to a private group. Users pay via Prodamus payment processor, receive invite links, and get scheduled messages. Built in Go 1.22.2.

## Development Commands

```bash
# Build the bot
make build
# or: go build -o ./.bin/bot cmd/app/main.go

# Run the bot (production build)
make run

# Run in development mode
make run-dev
# or: go run cmd/app/main.go

# Run with debug logging
BOT_DEBUG=true make run-dev
```

### Required Environment Variables

Create a `.env` file (see `.env.example`):

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

### Entry Point

`cmd/app/main.go` - Wires together bot and HTTP server, handles graceful shutdown.

### Core Components

1. **Telegram Bot** (`internal/telegram/`)
   - `bot.go` - Main bot logic, update handling, group join/leave tracking
   - `handlers.go` - Command handlers (`/start`, `/restart`, `/users`)
   - `command_service.go` - Command registration with role-based visibility
   - `command_mapper.go` - Maps commands to handlers

2. **HTTP Server** (`internal/server/`)
   - `server.go` - HTTP server for webhooks
   - `prodamus/webhook.go` - Prodamus payment webhook handler

3. **Services** (`internal/services/`)
   - `users.go` - User data management (JSON file storage)
   - `invite.go` - Invite link generation/revocation
   - `messages.go` - Message templates and keyboard configuration
   - `scheduler.go` - Message scheduling for users
   - `payment/payment.go` - Payment link generation via Prodamus
   - `storage.go` - Generic JSON read/write utilities

4. **Domain** (`internal/domain/`)
   - `command/command.go` - Command definition types
   - `command/registry.go` - Central command registry with roles

5. **Config** (`config/config.go`)
   - Loads from `.env` file, validates required fields

### Data Flow

1. **User onboarding**: `/start` → creates user → sends welcome message with "Accept terms" button
2. **Accept terms**: User clicks → generates Prodamus payment link → starts message scheduling
3. **Payment**: Prodamus webhook → generates invite link → sends group invite message
4. **Group access**: User joins → `chat_member` update → marks `JoinedGroup=true` → revokes invite link
5. **Scheduled messages**: Background scheduler sends messages from user's queue
6. **Group leave**: User leaves → generates new invite link (if paid) → sends via DM

### Key Patterns

**Command Registry**: Commands defined in `internal/domain/command/registry.go` with roles (public/admin). New commands: add to `AllCommands` slice.

**Message Templates**: Stored in `data/messages.json`. Use `{{placeholder}}` syntax. Support for photos and inline keyboards.

**Storage**: JSON files in `data/` directory. Generic `ReadJSON[T]` / `WriteJSON[T]` utilities with retry logic.

**Join Tracking**: Two mechanisms handle group joins:
- `chat_member` updates (primary)
- `new_chat_members` message events (fallback)

Both update `JoinedGroup` status and revoke invite links.

**Payment Callbacks**: HTTP server uses callbacks to invoke bot methods (`GenerateInviteLink`, `SendInviteMessage`).

### Important Notes

- All user data stored in `data/users.json`
- Message scheduling persisted to `data/schedule_backup.json`
- Invite links are one-time use (revoked after join)
- Admin-only commands silently fail for non-admins
- HTML parse mode for messages (use `<b>`, `<code>`, etc.)