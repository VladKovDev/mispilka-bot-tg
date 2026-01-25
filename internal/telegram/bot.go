package telegram

import (
	"context"
	"fmt"
	"log"
	"mispilkabot/config"
	"mispilkabot/internal/services"
	"mispilkabot/internal/services/scenario"
	"mispilkabot/internal/services/wizard"
	"sort"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Bot struct {
	bot             *tgbotapi.BotAPI
	cfg             *config.Config
	commandService  *CommandService
	scenarioService *scenario.Service
	wizardManager   *wizard.Manager
}

type Media []interface{}

func NewBot(bot *tgbotapi.BotAPI, cfg *config.Config, scenarioService *scenario.Service, wizardManager *wizard.Manager) *Bot {
	b := &Bot{
		bot:             bot,
		cfg:             cfg,
		commandService:  NewCommandService(bot),
		scenarioService: scenarioService,
		wizardManager:   wizardManager,
	}
	// Register admin commands
	b.registerAdminCommands()
	return b
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

// RegisterCommands registers bot commands with Telegram API using role-based visibility
func (b *Bot) RegisterCommands(ctx context.Context) error {
	return b.commandService.RegisterCommands(ctx, b.cfg.AdminIDs)
}

func (b *Bot) Start(ctx context.Context) {
	log.Printf("Authorized on account %s", b.bot.Self.UserName)

	services.CheckStorage("data/users.json")
	services.CheckStorage("data/schedule_backup.json")
	services.CheckStorage("data/messages.json")

	err := services.SetSchedules(func(chatID string) {
		b.sendScheduledMessage(chatID)
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
			if chatID == privateChatID && update.Message != nil {
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
	services.SetSchedule(time.Now(), userID, b.sendScheduledMessage)
}

func (b *Bot) sendScheduledMessage(chatID string) {
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

	services.SetNextSchedule(chatID, last, b.sendScheduledMessage)
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

	// Get user data - if user doesn't exist, we'll still process the update
	// but won't be able to track their status properly until they start a chat
	user, err := services.GetUser(userID)

	newStatus := chatMember.NewChatMember.Status
	oldStatus := chatMember.OldChatMember.Status

	// Handle user leaving the group (left, kicked, or banned)
	if newStatus == "left" || newStatus == "kicked" || newStatus == "banned" {
		log.Printf("[LEAVE] user %s leaving group: oldStatus=%s, newStatus=%s, userExists=%v",
			userID, oldStatus, newStatus, err == nil)

		// If user doesn't exist in our database, we can't track them properly
		if err != nil {
			log.Printf("[LEAVE] user %s not found in database, cannot update leave status", userID)
			return
		}

		log.Printf("[LEAVE] user %s details: JoinedGroup=%v, HasPaid=%v, InviteLink=%s",
			userID, user.JoinedGroup, user.HasPaid(), user.InviteLink)

		if user.JoinedGroup {
			user.JoinedGroup = false
			user.JoinedAt = nil

			// Only for users who left voluntarily (not kicked/banned)
			// Generate new invite link for paid users who left on their own
			if newStatus == "left" && user.HasPaid() {
				log.Printf("[LEAVE] processing voluntary leave for paid user %s", userID)
				newInviteLink, err := b.GenerateInviteLink(userID, b.cfg.PrivateGroupID)
				if err != nil {
					log.Printf("[LEAVE] failed to generate new invite link for paid user %s: %v", userID, err)
				} else {
					user.InviteLink = newInviteLink
					log.Printf("[LEAVE] generated new invite link for paid user %s who left the group voluntarily: %s", userID, newInviteLink)

					// Send the new link to user in private message using template
					parsedID, err := parseID(userID)
					if err != nil {
						log.Printf("[LEAVE] failed to parse userID %s: %v", userID, err)
					} else {
						// Get message template
						text, err := services.GetMessageText("group_leave_new_link")
						if err != nil {
							log.Printf("[LEAVE] failed to get message template: %v", err)
							// Fallback to hardcoded message
							text = fmt.Sprintf("Вы вышли из группы. Вот ваша новая ссылка для вступления:\n%s", newInviteLink)
						} else {
							// Replace {{invite_link}} placeholder
							values := map[string]string{"invite_link": newInviteLink}
							text = services.ReplaceAllPlaceholders(text, values)
						}

						msg := tgbotapi.NewMessage(parsedID, text)
						msg.DisableWebPagePreview = true
						if _, err := b.bot.Send(msg); err != nil {
							log.Printf("[LEAVE] failed to send new invite link to user %s: %v", userID, err)
						} else {
							log.Printf("[LEAVE] successfully sent new invite link to paid user %s", userID)
						}
					}
				}
			} else if newStatus != "left" {
				log.Printf("[LEAVE] user %s was kicked/banned (status: %s), not sending new link", userID, newStatus)
			} else if !user.HasPaid() {
				log.Printf("[LEAVE] user %s left but hasn't paid, not sending new link", userID)
			}

			if err := services.ChangeUser(userID, user); err != nil {
				log.Printf("[LEAVE] failed to update user %s after leaving group: %v", userID, err)
			} else {
				log.Printf("[LEAVE] user %s left the group (status: %s), JoinedGroup reset to false", userID, newStatus)
			}
		}
		return
	}

	// Handle user joining the group (member, administrator, or creator)
	if newStatus == "member" || newStatus == "administrator" || newStatus == "creator" {
		// Check if this is a new join (was not a member/admin/creator before)
		wasNotMember := oldStatus != "member" && oldStatus != "administrator" && oldStatus != "creator"

		log.Printf("[JOIN] user %s status change: oldStatus=%s, newStatus=%s, wasNotMember=%v, userExists=%v",
			userID, oldStatus, newStatus, wasNotMember, err == nil)

		if wasNotMember {
			// User is joining the group (first time or re-joining)
			inviteLink := ""
			if chatMember.InviteLink != nil {
				inviteLink = chatMember.InviteLink.InviteLink
			}

			// If user doesn't exist in database, we can't track them properly
			// They need to start a chat with the bot first
			if err != nil {
				log.Printf("[JOIN] user %s not found in database - they need to start a chat with the bot first. inviteLink=%s", userID, inviteLink)
				// Still revoke the invite link if present for security
				if inviteLink != "" {
					if err := b.RevokeInviteLink(b.cfg.PrivateGroupID, inviteLink); err != nil {
						log.Printf("[JOIN] failed to revoke invite link for unknown user %s: %v", userID, err)
					} else {
						log.Printf("[JOIN] revoked invite link for unknown user %s", userID)
					}
				}
				return
			}

			// Allow join if user has paid (with any invite link) or if link matches stored one
			hasPaid := user.HasPaid()
			inviteLinkMatches := inviteLink != "" && user.InviteLink == inviteLink
			validJoin := hasPaid || inviteLinkMatches

			// Detailed logging for debugging
			paymentDate := "nil"
			if user.PaymentDate != nil {
				paymentDate = user.PaymentDate.Format(time.RFC3339)
			}

			log.Printf("[JOIN] user %s joining: hasPaid=%v (PaymentDate=%s), inviteLinkMatches=%v (inviteLink=%q, stored=%q), validJoin=%v",
				userID, hasPaid, paymentDate, inviteLinkMatches, inviteLink, user.InviteLink, validJoin)

			// TEMPORARY DEBUG: Always update status regardless of validJoin
			// This helps diagnose if the issue is with validation or with persistence
			user.JoinedGroup = true
			joinedAt := time.Now()
			user.JoinedAt = &joinedAt
			if err := services.ChangeUser(userID, user); err != nil {
				log.Printf("[JOIN] failed to update JoinedGroup for user %s: %v", userID, err)
			} else {
				log.Printf("[JOIN] user %s joined private group successfully, JoinedGroup set to true (paid: %v)", userID, user.HasPaid())
			}

			// Revoke the invite link for security (one-time use) - only if it's the user's stored link
			if inviteLink != "" && inviteLink == user.InviteLink {
				if err := b.RevokeInviteLink(b.cfg.PrivateGroupID, inviteLink); err != nil {
					log.Printf("[JOIN] failed to revoke invite link for user %s: %v", userID, err)
				} else {
					log.Printf("[JOIN] invite link revoked for user %s", userID)
				}
			} else if inviteLink != "" {
				log.Printf("[JOIN] invite link %q does not match stored link %q, not revoking", inviteLink, user.InviteLink)
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

// handleNewChatMemberMessage processes new_chat_members message events (users joining via message)
// This is called when the bot receives a message with new_chat_members in the private group
func (b *Bot) handleNewChatMemberMessage(newMember *tgbotapi.User, message *tgbotapi.Message, privateChatID int64) {
	userID := fmt.Sprint(newMember.ID)

	log.Printf("[JOIN_MSG] User %s (%s) joined group via message event", userID, newMember.UserName)

	// Get user data
	user, err := services.GetUser(userID)
	if err != nil {
		log.Printf("[JOIN_MSG] User %s not found in database - they need to start a chat with the bot first", userID)
		return
	}

	// Update JoinedGroup status
	user.JoinedGroup = true
	joinedAt := time.Now()
	user.JoinedAt = &joinedAt

	// Log details for debugging
	paymentDate := "nil"
	if user.PaymentDate != nil {
		paymentDate = user.PaymentDate.Format(time.RFC3339)
	}

	log.Printf("[JOIN_MSG] User %s joining: hasPaid=%v (PaymentDate=%s), storedInviteLink=%q",
		userID, user.HasPaid(), paymentDate, user.InviteLink)

	// Save updated user data
	if err := services.ChangeUser(userID, user); err != nil {
		log.Printf("[JOIN_MSG] Failed to update JoinedGroup for user %s: %v", userID, err)
	} else {
		log.Printf("[JOIN_MSG] User %s joined private group successfully, JoinedGroup set to true", userID)
	}

	// Revoke the user's stored invite link for security (one-time use)
	if user.InviteLink != "" {
		if err := b.RevokeInviteLink(b.cfg.PrivateGroupID, user.InviteLink); err != nil {
			log.Printf("[JOIN_MSG] Failed to revoke invite link for user %s: %v", userID, err)
		} else {
			log.Printf("[JOIN_MSG] Invite link revoked for user %s: %s", userID, user.InviteLink)
			// Clear the invite link after revoking
			user.InviteLink = ""
			services.ChangeUser(userID, user)
		}
	}
}

// handleLeftChatMemberMessage processes left_chat_member message events (user leaving via message)
// This is called when the bot receives a message with left_chat_member in the private group
func (b *Bot) handleLeftChatMemberMessage(leftMember *tgbotapi.User, message *tgbotapi.Message, privateChatID int64) {
	userID := fmt.Sprint(leftMember.ID)

	// Determine if user left voluntarily or was kicked
	// In message events, we check if the 'from' field matches the left member
	fromID := fmt.Sprint(message.From.ID)
	leftVoluntarily := (fromID == userID)

	log.Printf("[LEAVE_MSG] User %s left group via message event: fromID=%s, leftVoluntarily=%v", userID, fromID, leftVoluntarily)

	// Get user data
	user, err := services.GetUser(userID)
	if err != nil {
		log.Printf("[LEAVE_MSG] User %s not found in database, cannot update leave status", userID)
		return
	}

	log.Printf("[LEAVE_MSG] User %s details: JoinedGroup=%v, HasPaid=%v, InviteLink=%s",
		userID, user.JoinedGroup, user.HasPaid(), user.InviteLink)

	if !user.JoinedGroup {
		log.Printf("[LEAVE_MSG] User %s was not marked as joined, skipping", userID)
		return
	}

	// Update JoinedGroup status
	user.JoinedGroup = false
	user.JoinedAt = nil

	// If user left voluntarily and has paid, generate new invite link
	if leftVoluntarily && user.HasPaid() {
		log.Printf("[LEAVE_MSG] Processing voluntary leave for paid user %s", userID)
		newInviteLink, err := b.GenerateInviteLink(userID, b.cfg.PrivateGroupID)
		if err != nil {
			log.Printf("[LEAVE_MSG] Failed to generate new invite link for paid user %s: %v", userID, err)
		} else {
			user.InviteLink = newInviteLink
			log.Printf("[LEAVE_MSG] Generated new invite link for paid user %s who left voluntarily: %s", userID, newInviteLink)

			// Send the new link to user in private message using template
			parsedID, err := parseID(userID)
			if err != nil {
				log.Printf("[LEAVE_MSG] Failed to parse userID %s: %v", userID, err)
			} else {
				// Get message template
				text, err := services.GetMessageText("group_leave_new_link")
				if err != nil {
					log.Printf("[LEAVE_MSG] Failed to get message template: %v", err)
					// Fallback to hardcoded message
					text = fmt.Sprintf("Вы вышли из группы. ВОТ ВАША НОВАЯ ССЫЛКА:\n%s", newInviteLink)
				} else {
					// Replace {{invite_link}} placeholder
					values := map[string]string{"invite_link": newInviteLink}
					text = services.ReplaceAllPlaceholders(text, values)
				}

				msg := tgbotapi.NewMessage(parsedID, text)
				msg.DisableWebPagePreview = true
				if _, err := b.bot.Send(msg); err != nil {
					log.Printf("[LEAVE_MSG] Failed to send new invite link to user %s: %v", userID, err)
				} else {
					log.Printf("[LEAVE_MSG] Successfully sent new invite link to paid user %s", userID)
				}
			}
		}
	} else if !leftVoluntarily {
		log.Printf("[LEAVE_MSG] User %s was kicked/banned, not sending new link", userID)
	} else if !user.HasPaid() {
		log.Printf("[LEAVE_MSG] User %s left but hasn't paid, not sending new link", userID)
	}

	// Save updated user data
	if err := services.ChangeUser(userID, user); err != nil {
		log.Printf("[LEAVE_MSG] Failed to update user %s after leaving group: %v", userID, err)
	} else {
		log.Printf("[LEAVE_MSG] User %s left the group (voluntarily: %v), JoinedGroup reset to false", userID, leftVoluntarily)
	}
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
