# ADR: Telegram Slash Commands with Role-Based Visibility

**Status**: Accepted
**Date**: 2026-01-24
**Decision Type**: Feature

## Context and Problem Statement

The Telegram bot currently implements slash commands (`/start`, `/restart`, `/users`), but users cannot discover available commands through the Telegram app's command menu. Users must know command names in advance or read documentation elsewhere.

Additionally, admin commands (`/users`) are visible to all users through manual typing, even though non-admins receive no response (silent ignore pattern). This creates confusion.

## Decision Drivers

- **User Experience**: Users should easily discover available commands
- **Role-Based Access**: Admin commands should only be visible to admins
- **Maintainability**: Solution should be simple and follow existing patterns
- **Reliability**: Graceful degradation if command registration fails

## Considered Alternatives

### Alternative 1: Runtime Filtering
Register all commands for all users, check permissions in handlers.

**Pros**:
- Simpler registration (single API call)
- No per-admin registration needed

**Cons**:
- Poor UX - users see commands they can't use
- Confusion when typing admin commands returns nothing
- More invalid command attempts

**Decision**: ❌ Rejected

### Alternative 2: Database-Driven Commands
Store command definitions in database, load at runtime.

**Pros**:
- Commands can be changed without redeployment
- Non-developers can manage commands

**Cons**:
- Over-engineering for 3 commands
- Adds database dependency
- More failure modes

**Decision**: ❌ Rejected (future enhancement if commands >20)

### Alternative 3: Scope-Based Registration (CHOSEN)
Use Telegram's `BotCommandScope` to filter commands at API level.

**Pros**:
- Native Telegram UX
- Users only see relevant commands
- Type-safe (hardcoded commands)
- Clean separation of concerns

**Cons**:
- Per-admin API calls (negligible for <50 admins)
- Requires restart for admin list changes

**Decision**: ✅ **Selected**

## Decision

Implement scope-based command registration using Telegram Bot API's `setMyCommands` method:

1. **Public commands**: Register with `BotCommandScopeAllPrivateChats`
2. **Admin commands**: Register per-user with `BotCommandScopeChat`

### Architecture

```
Domain Layer (command package)
  ├── Command entity
  └── Registry (AllCommands)

Service Layer (telegram package)
  ├── CommandMapper (domain → Telegram API)
  └── CommandService (registration logic)

Application Layer
  ├── Bot.RegisterCommands()
  └── main.go: startup registration
```

### Implementation

**Files Created**:
- `/internal/domain/command/command.go` - Command entity
- `/internal/domain/command/registry.go` - Command definitions
- `/internal/telegram/command_mapper.go` - API type mapping
- `/internal/telegram/command_service.go` - Registration logic

**Files Modified**:
- `/internal/telegram/bot.go` - Added `RegisterCommands()` method
- `/cmd/app/main.go` - Added startup registration call

## Consequences

### Positive
- ✅ Better UX - users see only relevant commands
- ✅ Reduced confusion - admin commands hidden from non-admins
- ✅ Type-safe - commands defined in code
- ✅ Graceful degradation - bot works if registration fails

### Negative
- ⚠️ Per-admin API calls (N requests for N admins)
- ⚠️ Admin list changes require bot restart
- ⚠️ Slightly slower startup (~50ms per admin)

### Mitigations
- Startup delay is negligible for <50 admins
- Admin list changes are infrequent
- Can add `/reloadcommands` in future if needed

## Implementation Summary

### Command Registration Flow

```go
// At startup
bot.RegisterCommands(ctx)

// In CommandService
1. Register public commands (BotCommandScopeAllPrivateChats)
2. For each admin ID:
   Register all commands (BotCommandScopeChat(adminID))
```

### Error Handling

```go
// main.go
if err := bot.RegisterCommands(ctx); err != nil {
    log.Printf("Failed to register commands: %v", err)
    // Continue anyway - bot can work without command menu
}
```

## Rollback Plan

If issues arise:

1. **Immediate**: Remove `bot.RegisterCommands(ctx)` call from `main.go`
2. **Clean up**: Delete created files in `internal/domain/command/` and `internal/telegram/command_*.go`
3. **Revert**: Remove `commandService` field from `Bot` struct

Bot continues to function normally without command menu.

## Testing Checklist

- [ ] Regular user sees only `/start`, `/restart`
- [ ] Admin user sees `/start`, `/restart`, `/users`
- [ ] Non-admin typing `/users` gets silent ignore (existing behavior)
- [ ] Bot starts successfully with registration
- [ ] Bot continues if registration fails
- [ ] Commands execute correctly after selection

## Related Decisions

- [Silent Permission Pattern](#) - Admin commands use silent ignore
- [Admin Configuration](#) - Admin IDs via environment variable

## References

- [Telegram Bot API: setMyCommands](https://core.telegram.org/bots/api#setmycommands)
- [BotCommandScope Types](https://core.telegram.org/type/BotCommandScope)
- [go-telegram-bot-api](https://github.com/go-telegram-bot-api/telegram-bot-api)
