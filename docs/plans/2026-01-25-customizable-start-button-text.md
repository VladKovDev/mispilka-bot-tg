# Customizable Start Button Text Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Make the start message button text configurable via messages.json instead of hardcoded, with emojis still applied in Go code.

**Architecture:** Extract button text from messages.json inline_keyboard configuration, add emojis (✅/🔲) in Go code based on user's IsMessaging status, with fallback to hardcoded text if JSON is missing.

**Tech Stack:** Go 1.22.2, telegram-bot-api/v5, existing JSON message system

---

## Task 1: Add GetStartButtonText function to services/messages.go

**Files:**
- Modify: `internal/services/messages.go` (add function at end of file, after line 246)

**Step 1: Write the function**

Add to `internal/services/messages.go` after the `GetPhoto` function:

```go
// GetStartButtonText возвращает текст кнопки для start-сообщения.
// Сначала пытается получить текст из inline_keyboard конфигурации в messages.json.
// Если кнопка не настроена, возвращает текст по умолчанию "Принимаю".
func GetStartButtonText() (string, error) {
	keyboard, err := GetInlineKeyboard("start")
	if err != nil || keyboard == nil || len(keyboard.Rows) == 0 {
		// Fallback к тексту по умолчанию
		return "Принимаю", nil
	}

	// Берём текст из первой кнопки первой строки
	if len(keyboard.Rows[0].Buttons) > 0 {
		text := keyboard.Rows[0].Buttons[0].Text
		if text != "" {
			return text, nil
		}
	}

	// Fallback если текст пустой
	return "Принимаю", nil
}
```

**Step 2: Verify code compiles**

Run: `go build -o ./.bin/bot cmd/app/main.go`
Expected: SUCCESS (no compilation errors)

**Step 3: Commit**

```bash
git add internal/services/messages.go
git commit -m "feat: add GetStartButtonText function for customizable start button text"
```

---

## Task 2: Update buildStartMessage in telegram/handlers.go

**Files:**
- Modify: `internal/telegram/handlers.go` (lines 65-95)

**Step 1: Update buildStartMessage function**

Replace the entire `buildStartMessage` function (lines 65-95) with:

```go
// buildStartMessage creates the start command message with appropriate keyboard
func (b *Bot) buildStartMessage(chatID string) (tgbotapi.MessageConfig, error) {
	text, err := services.GetMessageText(commandStart)
	if err != nil {
		return tgbotapi.MessageConfig{}, fmt.Errorf("failed to get message text: %w", err)
	}

	parsedID, err := parseID(chatID)
	if err != nil {
		return tgbotapi.MessageConfig{}, fmt.Errorf("failed to parse chatID %s: %w", chatID, err)
	}

	msg := tgbotapi.NewMessage(parsedID, text)
	msg.ParseMode = "HTML"
	msg.DisableWebPagePreview = true

	// Get button text from messages.json (with fallback to "Принимаю")
	buttonText, _ := services.GetStartButtonText()

	// Set keyboard based on user's messaging status
	userData, err := services.GetUser(chatID)
	if err == nil {
		if userData.IsMessaging {
			msg.ReplyMarkup = dataButton("✅ "+buttonText, "decline")
		} else {
			msg.ReplyMarkup = dataButton("🔲 "+buttonText, "accept")
		}
	} else {
		// Default keyboard for new users
		msg.ReplyMarkup = dataButton("🔲 "+buttonText, "accept")
	}

	return msg, nil
}
```

**Step 2: Verify code compiles**

Run: `go build -o ./.bin/bot cmd/app/main.go`
Expected: SUCCESS

**Step 3: Commit**

```bash
git add internal/telegram/handlers.go
git commit -m "refactor: use GetStartButtonText in buildStartMessage with emoji prefix"
```

---

## Task 3: Update acceptCallback in telegram/bot.go

**Files:**
- Modify: `internal/telegram/bot.go` (lines 167-201, specifically the button update part around line 190-194)

**Step 1: Update acceptCallback function**

Modify the `acceptCallback` function to use `GetStartButtonText`. Replace lines 190-194:

Find this section:
```go
	// Update button to "✅ Принято"
	edit := tgbotapi.NewEditMessageReplyMarkup(
		callback.From.ID,
		callback.Message.MessageID,
		dataButton("✅ Принято", "decline"))
```

Replace with:
```go
	// Get button text from messages.json and update button
	buttonText, _ := services.GetStartButtonText()
	edit := tgbotapi.NewEditMessageReplyMarkup(
		callback.From.ID,
		callback.Message.MessageID,
		dataButton("✅ "+buttonText, "decline"))
```

**Step 2: Verify code compiles**

Run: `go build -o ./.bin/bot cmd/app/main.go`
Expected: SUCCESS

**Step 3: Commit**

```bash
git add internal/telegram/bot.go
git commit -m "refactor: use GetStartButtonText in acceptCallback with emoji prefix"
```

---

## Task 4: Update example messages.json with start button configuration

**Files:**
- Modify: `data/example/messages.json`

**Step 1: Add inline_keyboard configuration for start message**

Update `data/example/messages.json` to include the start button configuration:

```json
{
  "messages_list": [],
  "messages": {
    "start": {
      "inline_keyboard": {
        "rows": [
          {
            "buttons": [
              {
                "text": "Принимаю",
                "type": "callback",
                "callback_data": "accept"
              }
            ]
          }
        ]
      }
    }
  }
}
```

**Step 2: Verify JSON is valid**

Run: `cat data/example/messages.json | jq .`
Expected: Valid JSON output (no parse errors)

**Step 3: Commit**

```bash
git add data/example/messages.json
git commit -m "docs: add start button inline_keyboard configuration to example messages.json"
```

---

## Task 5: Update main data/messages.json if it exists

**Files:**
- Check: `data/messages.json`

**Step 1: Check if data/messages.json needs updating**

Run: `cat data/messages.json | jq .`

If the file exists and doesn't have `start` with `inline_keyboard`, add the configuration.

If `data/messages.json` is empty (`{}`) or doesn't have start configured, add:

```json
{
  "messages_list": [],
  "messages": {
    "start": {
      "inline_keyboard": {
        "rows": [
          {
            "buttons": [
              {
                "text": "Принимаю",
                "type": "callback",
                "callback_data": "accept"
              }
            ]
          }
        ]
      }
    }
  }
}
```

**Step 2: Verify JSON is valid**

Run: `cat data/messages.json | jq .`
Expected: Valid JSON output

**Step 3: Commit**

```bash
git add data/messages.json
git commit -m "feat: add start button configuration to messages.json"
```

---

## Task 6: Manual Testing

**Step 1: Build the bot**

Run: `make build`
Expected: Binary created at `.bin/bot`

**Step 2: Run with debug logging**

Run: `BOT_DEBUG=true make run-dev`
Expected: Bot starts without errors

**Step 3: Test /start command**

1. Send `/start` to the bot in Telegram
2. Verify the button shows "🔲 Принимаю"
3. Click the button
4. Verify it changes to "✅ Принимаю"

**Step 4: Test with custom button text**

1. Modify `data/messages.json` - change button text to something else (e.g., "Подтверждаю")
2. Restart the bot
3. Send `/start` to the bot (as a new user or after `/restart`)
4. Verify the button shows "🔲 Подтверждаю"
5. Click the button
6. Verify it changes to "✅ Подтверждаю"

**Step 5: Test fallback behavior**

1. Remove or break the `inline_keyboard` configuration for `start` in `data/messages.json`
2. Restart the bot
3. Send `/start` to the bot
4. Verify the button still works with default text "🔲 Принимаю" → "✅ Принимаю"

---

## Summary of Changes

| File | Change |
|------|--------|
| `internal/services/messages.go` | Add `GetStartButtonText()` function |
| `internal/telegram/handlers.go` | Use `GetStartButtonText()` in `buildStartMessage()` |
| `internal/telegram/bot.go` | Use `GetStartButtonText()` in `acceptCallback()` |
| `data/example/messages.json` | Add example `inline_keyboard` config for `start` |
| `data/messages.json` | Add `inline_keyboard` config for `start` |

## Behavior

**Before:** Button text hardcoded as "✅ Принято" / "🔲 Принимаю"

**After:** Button text from `data/messages.json` with emoji prefix:
- `🔲 <text>` for users who haven't accepted
- `✅ <text>` for users who have accepted
- Fallback to "Принимаю" if JSON not configured

---

## Testing Checklist

- [ ] Code compiles without errors
- [ ] Bot starts successfully
- [ ] `/start` shows button with emoji + text from JSON
- [ ] Clicking button changes emoji from 🔲 to ✅
- [ ] Custom button text works when modified in JSON
- [ ] Fallback to "Принимаю" works when JSON is missing
- [ ] Existing users with `IsMessaging=true` see ✅ button
- [ ] New users see 🔲 button
