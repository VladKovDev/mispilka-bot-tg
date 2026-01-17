package telegram

import (
	"context"
	"fmt"
	"log"
	"mispilkabot/config"
	"mispilkabot/internal/services"
	"sort"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Bot struct {
	bot *tgbotapi.BotAPI
	cfg *config.Config
}

type Media []interface{}

func NewBot(bot *tgbotapi.BotAPI, cfg *config.Config) *Bot {
	return &Bot{bot: bot, cfg: cfg}
}

// GenerateInviteLink creates a new invite link for the specified group
func (b *Bot) GenerateInviteLink(userID, groupID string) (string, error) {
	return services.GenerateInviteLink(userID, groupID, b.bot)
}

// RevokeInviteLink revokes an existing invite link
func (b *Bot) RevokeInviteLink(groupID, inviteLink string) error {
	return services.RevokeInviteLink(groupID, inviteLink, b.bot)
}

// Request makes an API request to Telegram and returns the response
func (b *Bot) Request(c tgbotapi.Chattable) (tgbotapi.APIResponse, error) {
	resp, err := b.bot.Request(c)
	if err != nil {
		return tgbotapi.APIResponse{}, err
	}
	return *resp, nil
}

func (b *Bot) Start(ctx context.Context) {
	log.Printf("Authorized on account %s", b.bot.Self.UserName)

	services.CheckStorage("data/users.json")
	services.CheckStorage("data/schedule_backup.json")
	services.CheckStorage("data/messages.json")

	err := services.SetSchedules(func(chatID string) {
		b.sendMessage(chatID)
	})

	if err != nil {
		log.Fatalf("SetSchedules failed to restore scheduled messages: %v", err)
	}

	privateChatID, err := parseID(b.cfg.PrivateGroupID)
	if err != nil {
		log.Fatalf("Failed to parse PrivateGroupID from config: %v", err)
	}

	b.handleUpdates(ctx, b.initUpdatesChannel(), privateChatID)
}

func (b *Bot) initUpdatesChannel() tgbotapi.UpdatesChannel {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	return b.bot.GetUpdatesChan(u)
}

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

			// Handle chat_member updates (group join tracking)
			if update.ChatMember != nil {
				b.handleChatMember(update.ChatMember, privateChatID)
				continue
			}

			// Handle my_chat_member updates (bot's own member status changes)
			if update.MyChatMember != nil {
				b.handleMyChatMember(update.MyChatMember, privateChatID)
				continue
			}

			chatID := update.FromChat().ID
			if chatID == privateChatID {
				if update.Message != nil && len(update.Message.NewChatMembers) > 0 {
					for _, newUser := range update.Message.NewChatMembers {
						if err := services.ChangeIsMessaging(fmt.Sprint(newUser.ID), false); err != nil {
							log.Printf("Failed to update messaging status for user %d: %v", newUser.ID, err)
						}
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

func (b *Bot) handleCallbackQuery(callback *tgbotapi.CallbackQuery) {
	switch callback.Data {
	case "accept":
		b.acceptCallback(callback)
	default:
		// Check if it's a pagination callback (format: users_page_1)
		if strings.HasPrefix(callback.Data, "users_page_") {
			b.usersPaginationCallback(callback)
		} else {
			callbackResponse := tgbotapi.NewCallback(callback.ID, "")
			if _, err := b.bot.Send(callbackResponse); err != nil {
				log.Printf("failed to send callback response: %v", err)
			}
		}
	}
}

func (b *Bot) acceptCallback(callback *tgbotapi.CallbackQuery) {
	userID := fmt.Sprint(callback.From.ID)

	// Set messaging status to true
	services.ChangeIsMessaging(userID, true)

	// Generate payment link via Prodamus
	prodamusClient := services.NewProdamusClient(b.cfg)
	paymentLink, err := prodamusClient.GeneratePaymentLink(userID, b.cfg.ProdamusProductName, b.cfg.ProdamusProductPrice, b.cfg.ProdamusProductPaidContent)
	if err != nil {
		log.Printf("[PAYMENT_ERROR] Failed to generate payment link for user %s: %v", userID, err)
		log.Printf("[PAYMENT_ERROR] Prodamus API URL: %s", b.cfg.ProdamusAPIURL)
		log.Printf("[PAYMENT_ERROR] User %s will continue without payment link. Keyboard buttons with {{payment_link}} placeholder will be filtered out.", userID)
		// Still continue even if payment link generation fails
		// Messages with payment buttons will be filtered to avoid invalid URL errors
	} else {
		log.Printf("[PAYMENT_SUCCESS] Generated payment link for user %s: %s", userID, paymentLink)
		// Save payment link to user data
		if err := services.SetPaymentLink(userID, paymentLink); err != nil {
			log.Printf("[PAYMENT_ERROR] Failed to save payment link for user %s: %v", userID, err)
		}
	}

	// Update button to "✅ Принято"
	edit := tgbotapi.NewEditMessageReplyMarkup(
		callback.From.ID,
		callback.Message.MessageID,
		dataButton("✅ Принято", "decline"))
	if _, err := b.bot.Send(edit); err != nil {
		log.Printf("failed to update button markup for user %s: %v", userID, err)
	}

	// Start message scheduling
	services.SetSchedule(time.Now(), userID, b.sendMessage)
}

func (b *Bot) sendMessage(chatID string) {
	data, err := services.GetUser(chatID)
	if err != nil {
		log.Printf("person data fetching error: %v", err)
		return
	}

	if !data.IsMessaging {
		return
	}

	last, err := services.LastMessage(data.MessagesList)
	if err != nil {
		log.Printf("failed to get last message for chat %s: %v", chatID, err)
		return
	}

	text, err := services.GetMessageText(last)
	if err != nil {
		log.Printf("message fetching error: %v", err)
		return
	}

	keyboardConfig, err := services.GetInlineKeyboard(last)
	if err != nil {
		log.Printf("failed to get keyboard config for message %s: %v", last, err)
		return
	}

	values := map[string]string{
		"payment_price": b.cfg.ProdamusProductPrice,
		"payment_link":  data.PaymentLink,
	}

	text = services.ReplaceAllPlaceholders(text, values)
	keyboard := processKeyboard(keyboardConfig, values)

	var msg tgbotapi.Chattable
	photoPath, err := services.GetPhoto(last)
	if err != nil {
		parsedID, err := parseID(chatID)
		if err != nil {
			log.Printf("failed to parse chatID %s: %v", chatID, err)
			return
		}
		m := tgbotapi.NewMessage(parsedID, text)
		m.ParseMode = "HTML"
		// m.DisableWebPagePreview = true
		if len(keyboard.InlineKeyboard) > 0 {
			m.ReplyMarkup = keyboard
		}
		msg = m
	} else {
		parsedID, err := parseID(chatID)
		if err != nil {
			log.Printf("failed to parse chatID %s: %v", chatID, err)
			return
		}
		p := tgbotapi.NewPhoto(parsedID, tgbotapi.FilePath(photoPath))
		p.Caption = text
		p.ParseMode = "HTML"
		if len(keyboard.InlineKeyboard) > 0 {
			p.ReplyMarkup = keyboard
		}
		msg = p
	}

	if _, err := b.bot.Send(msg); err != nil {
		log.Printf("send error to %s: %v", chatID, err)
		return
	}

	data.MessagesList = data.MessagesList[:len(data.MessagesList)-1]
	services.ChangeUser(chatID, data)

	last, err = services.LastMessage(data.MessagesList)
	if err != nil {
		log.Printf("failed to get next message for chat %s: %v", chatID, err)
		return
	}

	services.SetNextSchedule(chatID, last, b.sendMessage)
}

func (b *Bot) SendInviteMessage(userID string, inviteLink string) {
	text, err := services.GetMessageText("group_invite")
	if err != nil {
		log.Printf("failed to load group_invite template: %v", err)
		return
	}

	keyboardConfig, err := services.GetInlineKeyboard("group_invite")
	if err != nil {
		log.Printf("failed to get button config for group_invite: %v", err)
		return
	}

	values := map[string]string{"invite_link": inviteLink}
	text = services.ReplaceAllPlaceholders(text, values)

	keyboard := processKeyboard(keyboardConfig, values)

	parsedID, err := parseID(userID)
	if err != nil {
		log.Printf("failed to parse userID %s: %v", userID, err)
		return
	}
	m := tgbotapi.NewMessage(parsedID, text)
	m.ParseMode = "HTML"
	m.DisableWebPagePreview = true

	if len(keyboard.InlineKeyboard) > 0 {
		m.ReplyMarkup = keyboard
	}

	if _, err := b.bot.Send(m); err != nil {
		log.Printf("failed to send invite message to %s: %v", userID, err)
		return
	}

	log.Printf("invite message sent successfully to %s", userID)
}

// processKeyboard processes an inline keyboard configuration by applying placeholder values
// and filtering out buttons with incomplete data (e.g., missing URLs or unreplaced placeholders).
// This is particularly useful for filtering out payment buttons when payment links are unavailable.
func processKeyboard(config *services.InlineKeyboardConfig, values map[string]string) tgbotapi.InlineKeyboardMarkup {
	if config == nil {
		return tgbotapi.InlineKeyboardMarkup{}
	}

	var validRows [][]tgbotapi.InlineKeyboardButton

	for _, row := range config.Rows {
		var validButtons []tgbotapi.InlineKeyboardButton

		for _, btn := range row.Buttons {
			// Handle non-URL buttons (callback type)
			if btn.Type != services.ButtonTypeURL {
				if btn.Text != "" {
					var newBtn tgbotapi.InlineKeyboardButton
					switch btn.Type {
					case services.ButtonTypeCallback:
						newBtn = tgbotapi.NewInlineKeyboardButtonData(btn.Text, btn.CallbackData)
					}
					if newBtn.Text != "" {
						validButtons = append(validButtons, newBtn)
					}
				}
				continue
			}

			// Handle URL buttons - replace placeholders and validate
			text := services.ReplaceAllPlaceholders(btn.Text, values)
			url := services.ReplaceAllPlaceholders(btn.URL, values)

			// Filter out buttons with unreplaced placeholders (still contain {{...}}) or empty URLs
			if strings.Contains(url, services.PlaceholderStart) || url == "" {
				continue
			}

			if text != "" {
				validButtons = append(validButtons, tgbotapi.NewInlineKeyboardButtonURL(text, url))
			}
		}

		if len(validButtons) > 0 {
			validRows = append(validRows, validButtons)
		}
	}

	if len(validRows) == 0 {
		return tgbotapi.InlineKeyboardMarkup{}
	}

	return tgbotapi.NewInlineKeyboardMarkup(validRows...)
}

// dataButton creates a callback button for inline keyboard interactions
// Used in callback query handlers for accept/decline actions
func dataButton(text string, calldata string) tgbotapi.InlineKeyboardMarkup {
	btn := tgbotapi.NewInlineKeyboardButtonData(text, calldata)
	row := tgbotapi.NewInlineKeyboardRow(btn)
	return tgbotapi.NewInlineKeyboardMarkup(row)
}

func parseID(s string) (int64, error) {
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse ID %q: %w", s, err)
	}
	return id, nil
}

// handleChatMember processes chat_member updates to track when users join/leave the private group
func (b *Bot) handleChatMember(chatMember *tgbotapi.ChatMemberUpdated, privateChatID int64) {
	// Only process updates for the private group
	if chatMember.Chat.ID != privateChatID {
		return
	}

	userID := fmt.Sprint(chatMember.NewChatMember.User.ID)

	// Get user data
	user, err := services.GetUser(userID)
	if err != nil {
		log.Printf("user %s not found when processing group member update: %v", userID, err)
		return
	}

	newStatus := chatMember.NewChatMember.Status
	oldStatus := chatMember.OldChatMember.Status

	// Handle user leaving the group (left, kicked, or banned)
	if newStatus == "left" || newStatus == "kicked" || newStatus == "banned" {
		if user.JoinedGroup {
			user.JoinedGroup = false
			user.JoinedAt = nil

			// For paid users, generate new invite link and send it to them
			if user.HasPaid() {
				newInviteLink, err := b.GenerateInviteLink(userID, b.cfg.PrivateGroupID)
				if err != nil {
					log.Printf("failed to generate new invite link for paid user %s: %v", userID, err)
				} else {
					user.InviteLink = newInviteLink
					log.Printf("generated new invite link for paid user %s who left the group", userID)

					// Send the new link to user in private message
					parsedID, err := parseID(userID)
					if err != nil {
						log.Printf("failed to parse userID %s: %v", userID, err)
					} else {
						msg := tgbotapi.NewMessage(parsedID, fmt.Sprintf("Вы вышли из группы. Вот ваша новая ссылка для вступления:\n%s", newInviteLink))
						msg.DisableWebPagePreview = true
						if _, err := b.bot.Send(msg); err != nil {
							log.Printf("failed to send new invite link to user %s: %v", userID, err)
						} else {
							log.Printf("sent new invite link to paid user %s", userID)
						}
					}
				}
			}

			if err := services.ChangeUser(userID, user); err != nil {
				log.Printf("failed to update user %s after leaving group: %v", userID, err)
			} else {
				log.Printf("user %s left the group, JoinedGroup reset to false", userID)
			}
		}
		return
	}

	// Handle user joining the group (member, administrator, or creator)
	if newStatus == "member" || newStatus == "administrator" || newStatus == "creator" {
		// Check if this is a new join (was not member before)
		if oldStatus == "left" || oldStatus == "kicked" {
			// User was previously not a member, now joining
			inviteLink := ""
			if chatMember.InviteLink != nil {
				inviteLink = chatMember.InviteLink.InviteLink
			}

			// Allow re-join if user has paid (with any invite link) or if link matches stored one
			validJoin := user.HasPaid() || (inviteLink != "" && user.InviteLink == inviteLink)

			if validJoin && !user.JoinedGroup {
				user.JoinedGroup = true
				joinedAt := time.Now()
				user.JoinedAt = &joinedAt
				if err := services.ChangeUser(userID, user); err != nil {
					log.Printf("failed to update JoinedGroup for user %s: %v", userID, err)
				} else {
					log.Printf("user %s joined private group, JoinedGroup set to true (paid: %v)", userID, user.HasPaid())
				}
			}

			// Revoke the invite link for security (one-time use)
			if inviteLink != "" {
				if err := b.RevokeInviteLink(b.cfg.PrivateGroupID, inviteLink); err != nil {
					log.Printf("failed to revoke invite link for user %s: %v", userID, err)
				} else {
					log.Printf("invite link revoked for user %s", userID)
				}
			}
		}
	}
}

// handleMyChatMember processes my_chat_member updates (bot's own member status changes)
func (b *Bot) handleMyChatMember(chatMember *tgbotapi.ChatMemberUpdated, privateChatID int64) {
	// Log the event for monitoring purposes
	log.Printf("Bot's member status changed in chat %d: %s -> %s",
		chatMember.Chat.ID,
		chatMember.OldChatMember.Status,
		chatMember.NewChatMember.Status)
}

// usersPaginationCallback handles pagination button clicks for users list
// This is defined in bot.go to be called from handleCallbackQuery
func (b *Bot) usersPaginationCallback(callback *tgbotapi.CallbackQuery) {
	// Import services to get users data
	users, err := services.GetAllUsers()
	if err != nil {
		log.Printf("Failed to get users for pagination: %v", err)
		resp := tgbotapi.NewCallback(callback.ID, "Ошибка")
		b.bot.Send(resp)
		return
	}

	// Parse page number from callback data (format: users_page_1)
	pageStr := strings.TrimPrefix(callback.Data, "users_page_")
	page, err := strconv.Atoi(pageStr)
	if err != nil {
		log.Printf("Failed to parse page number from callback: %v", err)
		resp := tgbotapi.NewCallback(callback.ID, "")
		b.bot.Send(resp)
		return
	}

	// Sort users by registration time (newest first)
	var sortedUsers []userEntry
	for chatID, user := range users {
		sortedUsers = append(sortedUsers, userEntry{chatID, user})
	}
	sort.Slice(sortedUsers, func(i, j int) bool {
		return sortedUsers[i].user.RegTime.After(sortedUsers[j].user.RegTime)
	})

	// Call the edit function from handlers
	if err := b.sendUsersPageEdit(callback.Message.MessageID, callback.Message.Chat.ID, sortedUsers, page); err != nil {
		log.Printf("Failed to send users page: %v", err)
		resp := tgbotapi.NewCallback(callback.ID, "Ошибка")
		b.bot.Send(resp)
		return
	}

	// Answer callback
	resp := tgbotapi.NewCallback(callback.ID, "")
	b.bot.Send(resp)
}
