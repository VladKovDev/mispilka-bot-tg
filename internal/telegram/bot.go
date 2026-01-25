package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mispilkabot/config"
	domainScenario "mispilkabot/internal/domain/scenario"
	"mispilkabot/internal/services"
	"mispilkabot/internal/services/scenario"
	"mispilkabot/internal/services/validation"
	"mispilkabot/internal/services/wizard"
	"net/http"
	"os"
	"path/filepath"
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

			// Check if user has an active wizard - handle text input and photos
			if b.wizardManager != nil && !update.Message.IsCommand() {
				userID := fmt.Sprint(update.Message.From.ID)
				if _, err := b.wizardManager.Get(userID); err == nil {
					// Handle text or photo input for wizard
					if update.Message.Text != "" {
						b.handleWizardMessage(update.Message)
						continue
					}
					if len(update.Message.Photo) > 0 {
						b.handleWizardPhoto(update.Message)
						continue
					}
					// Log other message types for debugging
					if update.Message.Sticker != nil {
						log.Printf("[WIZARD] User %s sent a sticker in wizard", userID)
						continue
					}
					if update.Message.Document != nil {
						log.Printf("[WIZARD] User %s sent a document (mime: %s) in wizard", userID, update.Message.Document.MimeType)
						// Check if it's an image
						if strings.HasPrefix(update.Message.Document.MimeType, "image/") {
							log.Printf("[WIZARD] Document is an image, but photo handling is not implemented for documents")
						}
						b.sendMessage(update.Message.Chat.ID, "⚠️ Please send photos directly (not as files).")
						continue
					}
					if update.Message.Voice != nil {
						log.Printf("[WIZARD] User %s sent a voice message in wizard", userID)
						b.sendMessage(update.Message.Chat.ID, "⚠️ Voice messages are not supported. Please send text or photos.")
						continue
					}
					if update.Message.Video != nil {
						log.Printf("[WIZARD] User %s sent a video in wizard", userID)
						b.sendMessage(update.Message.Chat.ID, "⚠️ Videos are not supported. Please send photos only.")
						continue
					}
					// Unknown message type - log it
					log.Printf("[WIZARD] User %s sent unknown message type in wizard", userID)
				}
			}

			if update.Message.IsCommand() {
				b.handleCommand(update.Message)
			}
		}
	}
}

func (b *Bot) handleCallbackQuery(callback *tgbotapi.CallbackQuery) {
	data := callback.Data

	// Answer callback first
	b.answerCallback(callback.ID, "")

	switch data {
	case "accept":
		b.acceptCallback(callback)
	// Wizard flow callbacks
	case "wizard_confirm_scenario":
		b.handleWizardConfirm(callback)
	case "wizard_confirm_general":
		b.handleConfirmGeneral(callback)
	case "wizard_edit_general":
		b.handleEditGeneral(callback)
	case "wizard_confirm_summary":
		b.handleConfirmSummary(callback)
	case "wizard_edit_summary":
		b.handleEditSummary(callback)
	case "wizard_confirm_message":
		b.handleConfirmMessage(callback)
	case "wizard_edit_message":
		b.handleEditMessage(callback)
	case "wizard_add_more_yes":
		b.handleAddMoreYes(callback)
	case "wizard_add_more_no":
		b.handleAddMoreNo(callback)
	case "wizard_cancel":
		b.handleWizardCancel(callback)
	// Edit field selection - general info
	case "wizard_edit_field_scenario_name":
		b.handleEditField(callback, wizard.StepScenarioName)
	case "wizard_edit_field_product_name":
		b.handleEditField(callback, wizard.StepProductName)
	case "wizard_edit_field_product_price":
		b.handleEditField(callback, wizard.StepProductPrice)
	case "wizard_edit_field_paid_content":
		b.handleEditField(callback, wizard.StepPaidContent)
	case "wizard_edit_field_private_group_id":
		b.handleEditField(callback, wizard.StepPrivateGroupID)
	// Edit field selection - summary
	case "wizard_edit_field_summary_message":
		b.handleEditField(callback, wizard.StepSummaryMessage)
	case "wizard_edit_field_summary_photos":
		b.handleEditField(callback, wizard.StepSummaryPhotos)
	case "wizard_edit_field_summary_buttons":
		b.handleEditField(callback, wizard.StepSummaryButtons)
	// Edit field selection - message
	case "wizard_edit_field_message_text":
		b.handleEditField(callback, wizard.StepMessageText)
	case "wizard_edit_field_message_photos":
		b.handleEditField(callback, wizard.StepMessagePhotos)
	case "wizard_edit_field_message_timing":
		b.handleEditField(callback, wizard.StepMessageTiming)
	case "wizard_edit_field_message_buttons":
		b.handleEditField(callback, wizard.StepMessageButtons)
	case "wizard_photo_done":
		b.handlePhotoDone(callback)
	default:
		// Check if it's a pagination callback
		if strings.HasPrefix(data, "users_page_") {
			b.usersPaginationCallback(callback)
		} else if strings.HasPrefix(data, "scenario_info_") {
			b.handleScenarioInfoCallback(callback)
		} else if strings.HasPrefix(data, "scenario_demo_") {
			b.handleScenarioDemoCallback(callback)
		}
	}
}

// handleWizardMessage handles text messages from users in active wizard sessions
func (b *Bot) handleWizardMessage(message *tgbotapi.Message) {
	userID := fmt.Sprint(message.From.ID)
	text := message.Text

	// Get current wizard state
	state, err := b.wizardManager.Get(userID)
	if err != nil {
		log.Printf("[WIZARD] Failed to get wizard state for user %s: %v", userID, err)
		return
	}

	log.Printf("[WIZARD] Processing step %s for user %s with input: %q", state.CurrentStep, userID, text)

	// Validate input based on current step
	if err := b.validateWizardInput(state.CurrentStep, text); err != nil {
		b.sendValidationError(message.Chat.ID, err)
		return
	}

	// Store the input in wizard data
	// Special handling for timing to store hours and minutes separately
	if state.CurrentStep == wizard.StepMessageTiming {
		hours, minutes, err := validation.ParseMessageTiming(text)
		if err != nil {
			b.sendValidationError(message.Chat.ID, err)
			return
		}
		// Store both the original string and parsed values
		state.Set(string(state.CurrentStep), text)
		state.Set("message_timing_hours", hours)
		state.Set("message_timing_minutes", minutes)
		log.Printf("[WIZARD] Parsed timing: %dh %dm", hours, minutes)
	} else {
		state.Set(string(state.CurrentStep), text)
	}

	// Determine next step based on current step and wizard type
	var nextStep wizard.WizardStep
	var nextPrompt string

	switch state.WizardType {
	case wizard.WizardTypeCreateScenario:
		nextStep, nextPrompt = b.getNextScenarioStep(state)
	default:
		log.Printf("[WIZARD] Unknown wizard type: %s", state.WizardType)
		b.sendWizardError(message.Chat.ID, "Unknown wizard type")
		return
	}

	// Handle confirmation step
	if isConfirmationStep(nextStep) {
		// Set last confirmed step before moving to confirmation
		state.SetLastConfirmedStep(state.CurrentStep)

		if err := b.wizardManager.Update(userID, state); err != nil {
			log.Printf("[WIZARD] Failed to update wizard state: %v", err)
		}

		if err := b.wizardManager.Advance(userID, nextStep); err != nil {
			log.Printf("[WIZARD] Failed to advance wizard for user %s: %v", userID, err)
			b.sendWizardError(message.Chat.ID, "Failed to advance wizard")
			return
		}
		b.sendConfirmationForStep(message.Chat.ID, nextStep, state)
		return
	}

	// Check if wizard is complete
	if nextStep == "" {
		if err := b.finalizeScenarioWizard(userID, state); err != nil {
			b.sendWizardError(message.Chat.ID, "Failed to create scenario: "+err.Error())
		} else {
			b.sendMessage(message.Chat.ID, "✅ Scenario created successfully!")
		}
		_ = b.wizardManager.Cancel(userID)
		return
	}

	// Advance to next step
	if err := b.wizardManager.Advance(userID, nextStep); err != nil {
		log.Printf("[WIZARD] Failed to advance wizard for user %s: %v", userID, err)
		b.sendWizardError(message.Chat.ID, "Failed to advance wizard")
		return
	}

	// Send next step prompt
	b.sendMessage(message.Chat.ID, nextPrompt)
}

// validateWizardInput validates user input for a wizard step
func (b *Bot) validateWizardInput(step wizard.WizardStep, input string) error {
	switch step {
	case wizard.StepScenarioName:
		return validation.ValidateScenarioName(input)
	case wizard.StepProductName:
		return validation.ValidateProductName(input)
	case wizard.StepProductPrice:
		return validation.ValidateProductPrice(input)
	case wizard.StepPaidContent:
		return validation.ValidatePaidContent(input)
	case wizard.StepPrivateGroupID:
		return validation.ValidatePrivateGroupID(input)
	case wizard.StepMessageTiming:
		return validation.ValidateMessageTiming(input)
	case wizard.StepSummaryButtons, wizard.StepMessageButtons:
		// Allow empty input (skip buttons) or validate the format
		if input == "" || strings.ToLower(input) == "skip" || strings.ToLower(input) == "none" {
			return nil
		}
		return validation.ValidateMessageButtons(input)
	default:
		return nil // No validation for other steps
	}
}

// sendValidationError sends a validation error message
func (b *Bot) sendValidationError(chatID int64, err error) {
	msg := tgbotapi.NewMessage(chatID, "⚠️ "+err.Error()+"\n\nPlease try again:")
	msg.ParseMode = "HTML"
	if _, err := b.bot.Send(msg); err != nil {
		log.Printf("Failed to send validation error: %v", err)
	}
}

// handleWizardPhoto handles photo messages from users in active wizard sessions
func (b *Bot) handleWizardPhoto(message *tgbotapi.Message) {
	userID := fmt.Sprint(message.From.ID)
	log.Printf("[WIZARD] Photo received from user %s, photo count: %d", userID, len(message.Photo))

	// Get current wizard state
	state, err := b.wizardManager.Get(userID)
	if err != nil {
		log.Printf("[WIZARD] Failed to get wizard state for user %s: %v", userID, err)
		return
	}

	log.Printf("[WIZARD] Current step for user %s: %s", userID, state.CurrentStep)

	// Only accept photos at photo steps
	if state.CurrentStep != wizard.StepSummaryPhotos && state.CurrentStep != wizard.StepMessagePhotos {
		log.Printf("[WIZARD] User %s sent photo at wrong step: %s", userID, state.CurrentStep)
		b.sendMessage(message.Chat.ID, "⚠️ Photos can only be added at photo configuration steps.\n\nPlease send text or use the buttons.")
		return
	}

	// Get the largest photo (highest resolution)
	photos := message.Photo
	if len(photos) == 0 {
		log.Printf("[WIZARD] No photo found in message from user %s", userID)
		b.sendMessage(message.Chat.ID, "⚠️ No photo found. Please try again.")
		return
	}

	log.Printf("[WIZARD] Processing photo from user %s, file ID: %s", userID, photos[len(photos)-1].FileID)

	largestPhoto := photos[len(photos)-1]

	// Get file info to get the file path
	// In tgbotapi v5, use the Request method to get file info
	fileReq := tgbotapi.FileConfig{FileID: largestPhoto.FileID}
	fileResp, err := b.bot.Request(fileReq)
	if err != nil {
		log.Printf("[WIZARD] Failed to get photo file info: %v", err)
		b.sendMessage(message.Chat.ID, "⚠️ Failed to process photo. Please try again.")
		return
	}

	// Extract file path from result
	var file tgbotapi.File
	if err := json.Unmarshal(fileResp.Result, &file); err != nil {
		log.Printf("[WIZARD] Failed to unmarshal file info: %v", err)
		b.sendMessage(message.Chat.ID, "⚠️ Failed to process photo. Please try again.")
		return
	}

	if file.FilePath == "" {
		log.Printf("[WIZARD] File path is empty for photo")
		b.sendMessage(message.Chat.ID, "⚠️ Failed to process photo. Please try again.")
		return
	}

	log.Printf("[WIZARD] Got file path: %s", file.FilePath)

	// Create scenarios/photos directory if it doesn't exist
	photosDir := "data/scenarios/photos"
	if err := os.MkdirAll(photosDir, 0755); err != nil {
		log.Printf("[WIZARD] Failed to create photos directory: %v", err)
		b.sendMessage(message.Chat.ID, "⚠️ Failed to save photo. Please try again.")
		return
	}

	// Generate unique filename
	ext := ".jpg"
	fileName := fmt.Sprintf("%s_%d%s", userID, time.Now().Unix(), ext)
	filePath := filepath.Join(photosDir, fileName)

	// Download the photo
	photoURL := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", b.bot.Token, file.FilePath)
	resp, err := http.Get(photoURL)
	if err != nil {
		log.Printf("[WIZARD] Failed to download photo: %v", err)
		b.sendMessage(message.Chat.ID, "⚠️ Failed to download photo. Please try again.")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("[WIZARD] Failed to download photo, status: %d", resp.StatusCode)
		b.sendMessage(message.Chat.ID, "⚠️ Failed to download photo. Please try again.")
		return
	}

	// Save to disk
	out, err := os.Create(filePath)
	if err != nil {
		log.Printf("[WIZARD] Failed to create photo file: %v", err)
		b.sendMessage(message.Chat.ID, "⚠️ Failed to save photo. Please try again.")
		return
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		log.Printf("[WIZARD] Failed to save photo: %v", err)
		b.sendMessage(message.Chat.ID, "⚠️ Failed to save photo. Please try again.")
		return
	}

	log.Printf("[WIZARD] Photo saved successfully: %s", filePath)

	// Add to wizard state's photo list
	photoKey := string(state.CurrentStep)
	existingPhotos := state.GetStringSlice(photoKey)
	existingPhotos = append(existingPhotos, filePath)
	state.Set(photoKey, existingPhotos)

	log.Printf("[WIZARD] User %s now has %d photos for step %s", userID, len(existingPhotos), state.CurrentStep)

	// Save updated state
	if err := b.wizardManager.Update(userID, state); err != nil {
		log.Printf("[WIZARD] Failed to update wizard state: %v", err)
	}

	// Confirm and ask if more photos
	msgText := fmt.Sprintf("✅ Photo saved (%d total).\n\nSend more photos or click 'Done' to continue.", len(existingPhotos))

	msg := tgbotapi.NewMessage(message.Chat.ID, msgText)
	msg.ParseMode = "HTML"

	// Add "Done" button
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Done", "wizard_photo_done"),
		),
	)
	msg.ReplyMarkup = &keyboard

	if _, err := b.bot.Send(msg); err != nil {
		log.Printf("[WIZARD] Failed to send photo confirmation: %v", err)
	} else {
		log.Printf("[WIZARD] Photo confirmation sent to user %s", userID)
	}
}

// isConfirmationStep checks if a step is a confirmation step
func isConfirmationStep(step wizard.WizardStep) bool {
	return step == wizard.StepConfirmGeneral ||
		step == wizard.StepConfirmSummary ||
		step == wizard.StepConfirmMessage
}

// sendConfirmationForStep sends the appropriate confirmation message
func (b *Bot) sendConfirmationForStep(chatID int64, step wizard.WizardStep, state *wizard.WizardState) {
	switch step {
	case wizard.StepConfirmGeneral:
		b.sendScenarioConfirmation(chatID, state)
	case wizard.StepConfirmSummary:
		b.sendSummaryConfirmation(chatID, state)
	case wizard.StepConfirmMessage:
		b.sendMessageConfirmation(chatID, state)
	default:
		b.sendWizardError(chatID, "Unknown confirmation step")
	}
}

// getNextScenarioStep determines the next step and prompt for scenario creation wizard
func (b *Bot) getNextScenarioStep(state *wizard.WizardState) (nextStep wizard.WizardStep, prompt string) {
	currentStep := state.CurrentStep

	// If returning from edit mode, go back to confirmation
	if state.IsEditMode() && state.GetEditTargetStep() == currentStep {
		state.SetEditMode(false, "")
		return wizard.StepConfirmGeneral, ""
	}

	// Normal flow
	switch currentStep {
	case wizard.StepScenarioName:
		return wizard.StepProductName, b.getPromptForStep(wizard.StepProductName)
	case wizard.StepProductName:
		return wizard.StepProductPrice, b.getPromptForStep(wizard.StepProductPrice)
	case wizard.StepProductPrice:
		return wizard.StepPaidContent, b.getPromptForStep(wizard.StepPaidContent)
	case wizard.StepPaidContent:
		return wizard.StepPrivateGroupID, b.getPromptForStep(wizard.StepPrivateGroupID)
	case wizard.StepPrivateGroupID:
		return wizard.StepConfirmGeneral, ""
	// Summary steps flow (after edit, returns to appropriate confirmation)
	case wizard.StepSummaryMessage:
		return wizard.StepSummaryPhotos, b.getPromptForStep(wizard.StepSummaryPhotos)
	case wizard.StepSummaryPhotos:
		return wizard.StepConfirmSummary, ""
	case wizard.StepSummaryButtons:
		return wizard.StepConfirmSummary, ""
	// Message steps flow
	case wizard.StepMessageText:
		return wizard.StepMessagePhotos, b.getPromptForStep(wizard.StepMessagePhotos)
	case wizard.StepMessagePhotos:
		return wizard.StepMessageTiming, b.getPromptForStep(wizard.StepMessageTiming)
	case wizard.StepMessageTiming:
		return wizard.StepMessageButtons, b.getPromptForStep(wizard.StepMessageButtons)
	case wizard.StepMessageButtons:
		return wizard.StepConfirmMessage, ""
	default:
		return "", ""
	}
}

// finalizeScenarioWizard creates the scenario from collected wizard data
func (b *Bot) finalizeScenarioWizard(userID string, state *wizard.WizardState) error {
	if b.scenarioService == nil {
		return fmt.Errorf("scenario service not initialized")
	}

	// Generate scenario ID from name (simple slug)
	scenarioName := state.GetString(string(wizard.StepScenarioName))
	scenarioID := strings.ToLower(strings.ReplaceAll(scenarioName, " ", "_"))

	// Create the scenario
	req := &scenario.CreateScenarioRequest{
		ID:   scenarioID,
		Name: scenarioName,
		Prodamus: domainScenario.ProdamusConfig{
			ProductName:    state.GetString(string(wizard.StepProductName)),
			ProductPrice:   state.GetString(string(wizard.StepProductPrice)),
			PaidContent:    state.GetString(string(wizard.StepPaidContent)),
			PrivateGroupID: state.GetString(string(wizard.StepPrivateGroupID)),
		},
	}

	if _, err := b.scenarioService.CreateScenario(req); err != nil {
		return fmt.Errorf("failed to create scenario: %w", err)
	}

	// Save all pending messages to the scenario
	pendingMessages := state.GetPendingMessages()
	for _, pendingMsg := range pendingMessages {
		// Parse buttons string into keyboard config
		keyboard, err := parseButtonsString(pendingMsg.Buttons)
		if err != nil {
			log.Printf("[WIZARD] Failed to parse buttons for message %s: %v", pendingMsg.MessageID, err)
			// Continue without buttons
		}

		addMsgReq := &scenario.AddMessageRequest{
			ScenarioID:    scenarioID,
			MessageID:     pendingMsg.MessageID,
			Timing: domainScenario.Timing{
				Hours:   pendingMsg.TimingHours,
				Minutes: pendingMsg.TimingMinutes,
			},
			TemplateFile:   fmt.Sprintf("templates/%s/%s.txt", scenarioID, pendingMsg.MessageID),
			Photos:         pendingMsg.Photos,
			InlineKeyboard: keyboard,
		}

		if err := b.scenarioService.AddMessage(addMsgReq); err != nil {
			log.Printf("[WIZARD] Failed to add message %s to scenario: %v", pendingMsg.MessageID, err)
			// Continue adding other messages
		} else {
			log.Printf("[WIZARD] Added message %s to scenario %s", pendingMsg.MessageID, scenarioID)
		}
	}

	log.Printf("[WIZARD] Scenario %s created successfully by user %s with %d messages", scenarioID, userID, len(pendingMessages))
	return nil
}

// parseButtonsString parses button configuration string into InlineKeyboardConfig
func parseButtonsString(buttonsStr string) (*domainScenario.InlineKeyboardConfig, error) {
	if buttonsStr == "" || strings.ToLower(buttonsStr) == "skip" || strings.ToLower(buttonsStr) == "none" {
		return nil, nil
	}

	// Use message builder to parse keyboard
	mb := &scenario.MessageBuilder{}
	return mb.ParseKeyboard(buttonsStr)
}

// sendWizardError sends an error message to the user
func (b *Bot) sendWizardError(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, "❌ "+text)
	msg.ParseMode = "HTML"
	if _, err := b.bot.Send(msg); err != nil {
		log.Printf("Failed to send wizard error: %v", err)
	}
}

// sendScenarioConfirmation sends a confirmation message with the collected scenario data
func (b *Bot) sendScenarioConfirmation(chatID int64, state *wizard.WizardState) {
	scenarioName := state.GetString(string(wizard.StepScenarioName))
	productName := state.GetString(string(wizard.StepProductName))
	productPrice := state.GetString(string(wizard.StepProductPrice))
	paidContent := state.GetString(string(wizard.StepPaidContent))
	privateGroupID := state.GetString(string(wizard.StepPrivateGroupID))

	var sb strings.Builder
	sb.WriteString("✅ <b>Confirm Scenario General Info</b>\n\n")
	sb.WriteString(fmt.Sprintf("<b>Name:</b> %s\n", scenarioName))
	sb.WriteString(fmt.Sprintf("<b>Product:</b> %s\n", productName))
	sb.WriteString(fmt.Sprintf("<b>Price:</b> %s ₽\n", productPrice))
	sb.WriteString(fmt.Sprintf("<b>Paid Content:</b> %s\n", paidContent))
	sb.WriteString(fmt.Sprintf("<b>Group ID:</b> <code>%s</code>\n\n", privateGroupID))
	sb.WriteString("Click <b>Confirm</b> to continue to summary configuration\n")
	sb.WriteString("or <b>Edit</b> to modify any field.")

	// Create inline keyboard with confirm/edit/cancel buttons
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Confirm", "wizard_confirm_general"),
			tgbotapi.NewInlineKeyboardButtonData("✏️ Edit", "wizard_edit_general"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ Cancel", "wizard_cancel"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, sb.String())
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = &keyboard

	if _, err := b.bot.Send(msg); err != nil {
		log.Printf("Failed to send confirmation message: %v", err)
	}
}

// sendEditFieldSelection sends a message with buttons to select which field to edit
func (b *Bot) sendEditFieldSelection(chatID int64, state *wizard.WizardState) {
	var sb strings.Builder
	sb.WriteString("✏️ <b>Edit Scenario</b>\n\n")
	sb.WriteString("Select the field you want to edit:\n")

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📝 Name", "wizard_edit_field_scenario_name"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📦 Product Name", "wizard_edit_field_product_name"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💰 Price", "wizard_edit_field_product_price"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📄 Content", "wizard_edit_field_paid_content"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("👥 Group ID", "wizard_edit_field_private_group_id"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Back to Confirmation", "wizard_confirm_general"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, sb.String())
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = &keyboard

	if _, err := b.bot.Send(msg); err != nil {
		log.Printf("Failed to send edit field selection: %v", err)
	}
}

// sendSummaryConfirmation sends summary confirmation message
func (b *Bot) sendSummaryConfirmation(chatID int64, state *wizard.WizardState) {
	summaryText := state.GetString(string(wizard.StepSummaryMessage))
	summaryPhotos := state.GetStringSlice(string(wizard.StepSummaryPhotos))

	var sb strings.Builder
	sb.WriteString("✅ <b>Confirm Summary Message</b>\n\n")

	if summaryText != "" {
		// Truncate for display
		displayText := summaryText
		if len(displayText) > 200 {
			displayText = displayText[:200] + "..."
		}
		sb.WriteString(fmt.Sprintf("<b>Message:</b>\n%s\n\n", displayText))
	} else {
		sb.WriteString("<b>Message:</b> <i>(empty)</i>\n\n")
	}

	if len(summaryPhotos) > 0 {
		sb.WriteString(fmt.Sprintf("<b>Photos:</b> %d attached\n", len(summaryPhotos)))
	} else {
		sb.WriteString("<b>Photos:</b> <i>(none)</i>\n")
	}

	sb.WriteString("\nClick <b>Confirm</b> to continue to message creation\n")
	sb.WriteString("or <b>Edit</b> to modify the summary.")

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Confirm", "wizard_confirm_summary"),
			tgbotapi.NewInlineKeyboardButtonData("✏️ Edit", "wizard_edit_summary"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Back", "wizard_edit_general"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, sb.String())
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = &keyboard

	if _, err := b.bot.Send(msg); err != nil {
		log.Printf("Failed to send summary confirmation: %v", err)
	}
}

// sendSummaryEditSelection sends summary field selection for editing
func (b *Bot) sendSummaryEditSelection(chatID int64, state *wizard.WizardState) {
	var sb strings.Builder
	sb.WriteString("✏️ <b>Edit Summary Message</b>\n\n")
	sb.WriteString("Select what you want to edit:\n")

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📝 Message Text", "wizard_edit_field_summary_message"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🖼️ Photos", "wizard_edit_field_summary_photos"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Back to Confirmation", "wizard_confirm_summary"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, sb.String())
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = &keyboard

	if _, err := b.bot.Send(msg); err != nil {
		log.Printf("Failed to send summary edit selection: %v", err)
	}
}

// sendMessageConfirmation sends message confirmation
func (b *Bot) sendMessageConfirmation(chatID int64, state *wizard.WizardState) {
	msgIndex := state.GetCurrentMessageIndex()
	msgNum := msgIndex + 1

	messageText := state.GetString(string(wizard.StepMessageText))
	photos := state.GetStringSlice(string(wizard.StepMessagePhotos))
	timingHours := state.GetInt("message_timing_hours")
	timingMinutes := state.GetInt("message_timing_minutes")

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("✅ <b>Confirm Message %d</b>\n\n", msgNum))

	if messageText != "" {
		displayText := messageText
		if len(displayText) > 200 {
			displayText = displayText[:200] + "..."
		}
		sb.WriteString(fmt.Sprintf("<b>Text:</b>\n%s\n\n", displayText))
	}

	sb.WriteString(fmt.Sprintf("<b>Timing:</b> %dh %dm after previous message\n", timingHours, timingMinutes))

	if len(photos) > 0 {
		sb.WriteString(fmt.Sprintf("<b>Photos:</b> %d attached\n", len(photos)))
	} else {
		sb.WriteString("<b>Photos:</b> <i>(none)</i>\n")
	}

	sb.WriteString("\nClick <b>Confirm</b> to save this message\n")
	sb.WriteString("or <b>Edit</b> to modify it.")

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Confirm", "wizard_confirm_message"),
			tgbotapi.NewInlineKeyboardButtonData("✏️ Edit", "wizard_edit_message"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ Cancel", "wizard_cancel"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, sb.String())
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = &keyboard

	if _, err := b.bot.Send(msg); err != nil {
		log.Printf("Failed to send message confirmation: %v", err)
	}
}

// sendAddMoreMessagesPrompt asks if user wants to add more messages
func (b *Bot) sendAddMoreMessagesPrompt(chatID int64, state *wizard.WizardState) {
	msgCount := state.GetMessagesCreated()
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("📬 <b>Messages Added: %d</b>\n\n", msgCount))
	sb.WriteString("Do you want to add another message?\n")

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("➕ Add Another Message", "wizard_add_more_yes"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Finish Scenario", "wizard_add_more_no"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, sb.String())
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = &keyboard

	if _, err := b.bot.Send(msg); err != nil {
		log.Printf("Failed to send add more prompt: %v", err)
	}
}

// sendMessageEditSelection sends message field selection for editing
func (b *Bot) sendMessageEditSelection(chatID int64, state *wizard.WizardState) {
	msgIndex := state.GetCurrentMessageIndex()
	msgNum := msgIndex + 1

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("✏️ <b>Edit Message %d</b>\n\n", msgNum))
	sb.WriteString("Select what you want to edit:\n")

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📝 Text", "wizard_edit_field_message_text"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🖼️ Photos", "wizard_edit_field_message_photos"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⏰ Timing", "wizard_edit_field_message_timing"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔘 Buttons", "wizard_edit_field_message_buttons"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Back to Confirmation", "wizard_confirm_message"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, sb.String())
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = &keyboard

	if _, err := b.bot.Send(msg); err != nil {
		log.Printf("Failed to send message edit selection: %v", err)
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

// handleWizardConfirm handles the wizard confirmation callback
func (b *Bot) handleWizardConfirm(callback *tgbotapi.CallbackQuery) {
	userID := fmt.Sprint(callback.From.ID)
	chatID := callback.From.ID

	// Get wizard state
	state, err := b.wizardManager.Get(userID)
	if err != nil {
		log.Printf("[WIZARD] Failed to get wizard state for confirmation: %v", err)
		b.answerCallback(callback.ID, "❌ Wizard expired")
		return
	}

	// Finalize scenario creation
	if err := b.finalizeScenarioWizard(userID, state); err != nil {
		log.Printf("[WIZARD] Failed to finalize scenario: %v", err)
		b.answerCallback(callback.ID, "❌ Failed to create scenario")
		return
	}

	// Cancel the wizard
	_ = b.wizardManager.Cancel(userID)

	// Edit the message to show success
	edit := tgbotapi.NewEditMessageText(chatID, callback.Message.MessageID, "✅ Scenario created successfully!")
	edit.ParseMode = "HTML"
	if _, err := b.bot.Send(edit); err != nil {
		log.Printf("Failed to edit confirmation message: %v", err)
	}

	b.answerCallback(callback.ID, "")
}

// handleWizardCancel handles the wizard cancel callback
func (b *Bot) handleWizardCancel(callback *tgbotapi.CallbackQuery) {
	userID := fmt.Sprint(callback.From.ID)
	chatID := callback.From.ID

	// Cancel the wizard
	_ = b.wizardManager.Cancel(userID)

	// Edit the message to show cancellation
	edit := tgbotapi.NewEditMessageText(chatID, callback.Message.MessageID, "❌ Scenario creation cancelled")
	edit.ParseMode = "HTML"
	if _, err := b.bot.Send(edit); err != nil {
		log.Printf("Failed to edit cancellation message: %v", err)
	}

	b.answerCallback(callback.ID, "")
}

// handleConfirmGeneral handles general info confirmation
func (b *Bot) handleConfirmGeneral(callback *tgbotapi.CallbackQuery) {
	userID := fmt.Sprint(callback.From.ID)
	chatID := callback.From.ID

	state, err := b.wizardManager.Get(userID)
	if err != nil {
		b.sendWizardError(chatID, "Wizard expired")
		return
	}

	// Mark general section as confirmed
	state.SetCurrentSection("general")
	state.SetLastConfirmedStep(wizard.StepConfirmGeneral)
	_ = b.wizardManager.Update(userID, state)

	// Move to summary message step
	_ = b.wizardManager.Advance(userID, wizard.StepSummaryMessage)
	b.sendMessage(chatID, b.getPromptForStep(wizard.StepSummaryMessage))
}

// handleEditGeneral handles edit request from general confirmation
func (b *Bot) handleEditGeneral(callback *tgbotapi.CallbackQuery) {
	userID := fmt.Sprint(callback.From.ID)
	chatID := callback.From.ID

	state, err := b.wizardManager.Get(userID)
	if err != nil {
		b.sendWizardError(chatID, "Wizard expired")
		return
	}

	// Move to edit field selection step
	_ = b.wizardManager.Advance(userID, wizard.StepEditGeneral)
	b.sendEditFieldSelection(chatID, state)
}

// handleEditField handles editing a specific field
func (b *Bot) handleEditField(callback *tgbotapi.CallbackQuery, targetStep wizard.WizardStep) {
	userID := fmt.Sprint(callback.From.ID)
	chatID := callback.From.ID

	state, err := b.wizardManager.Get(userID)
	if err != nil {
		b.sendWizardError(chatID, "Wizard expired")
		return
	}

	// Enable edit mode and set target
	state.SetEditMode(true, targetStep)
	_ = b.wizardManager.Update(userID, state)

	// Move to the target step
	_ = b.wizardManager.Advance(userID, targetStep)
	b.sendMessage(chatID, b.getPromptForStep(targetStep))
}

// handlePhotoDone handles when user clicks "Done" after adding photos
func (b *Bot) handlePhotoDone(callback *tgbotapi.CallbackQuery) {
	userID := fmt.Sprint(callback.From.ID)
	chatID := callback.From.ID

	state, err := b.wizardManager.Get(userID)
	if err != nil {
		b.sendWizardError(chatID, "Wizard expired")
		return
	}

	// Determine next step based on current photo step
	var nextStep wizard.WizardStep
	var nextPrompt string

	switch state.CurrentStep {
	case wizard.StepSummaryPhotos:
		// After summary photos, go to confirmation
		nextStep = wizard.StepConfirmSummary
		nextPrompt = ""
	case wizard.StepMessagePhotos:
		// After message photos, go to timing step
		nextStep = wizard.StepMessageTiming
		nextPrompt = b.getPromptForStep(wizard.StepMessageTiming)
	default:
		b.sendWizardError(chatID, "Unexpected step for photo_done")
		return
	}

	// Advance to next step
	if nextStep != "" {
		if err := b.wizardManager.Advance(userID, nextStep); err != nil {
			log.Printf("[WIZARD] Failed to advance wizard: %v", err)
			b.sendWizardError(chatID, "Failed to advance wizard")
			return
		}
	}

	// If confirmation step, send confirmation
	if nextStep == wizard.StepConfirmSummary {
		b.sendSummaryConfirmation(chatID, state)
		return
	}

	// Otherwise send the next prompt
	if nextPrompt != "" {
		b.sendMessage(chatID, nextPrompt)
	}
}

// handleConfirmSummary handles summary confirmation
func (b *Bot) handleConfirmSummary(callback *tgbotapi.CallbackQuery) {
	userID := fmt.Sprint(callback.From.ID)
	chatID := callback.From.ID

	state, err := b.wizardManager.Get(userID)
	if err != nil {
		b.sendWizardError(chatID, "Wizard expired")
		return
	}

	// Mark summary section as confirmed
	state.SetCurrentSection("summary")
	state.SetLastConfirmedStep(wizard.StepConfirmSummary)
	_ = b.wizardManager.Update(userID, state)

	// Move to first message creation
	state.SetCurrentMessageIndex(0)
	state.SetCurrentSection("messages")
	_ = b.wizardManager.Update(userID, state)

	_ = b.wizardManager.Advance(userID, wizard.StepMessageText)
	msgNum := state.GetCurrentMessageIndex() + 1
	b.sendMessage(chatID, fmt.Sprintf("📝 <b>Message %d</b>\n\n%s", msgNum, b.getPromptForStep(wizard.StepMessageText)))
}

// handleEditSummary handles edit request from summary confirmation
func (b *Bot) handleEditSummary(callback *tgbotapi.CallbackQuery) {
	userID := fmt.Sprint(callback.From.ID)
	chatID := callback.From.ID

	state, err := b.wizardManager.Get(userID)
	if err != nil {
		b.sendWizardError(chatID, "Wizard expired")
		return
	}

	_ = b.wizardManager.Advance(userID, wizard.StepEditSummary)
	b.sendSummaryEditSelection(chatID, state)
}

// handleConfirmMessage handles message confirmation
func (b *Bot) handleConfirmMessage(callback *tgbotapi.CallbackQuery) {
	userID := fmt.Sprint(callback.From.ID)
	chatID := callback.From.ID

	state, err := b.wizardManager.Get(userID)
	if err != nil {
		b.sendWizardError(chatID, "Wizard expired")
		return
	}

	// Save message as pending message to be added when scenario is finalized
	msgIndex := state.GetCurrentMessageIndex()
	msgNum := msgIndex + 1
	messageText := state.GetString(string(wizard.StepMessageText))
	photos := state.GetStringSlice(string(wizard.StepMessagePhotos))
	timingHours := state.GetInt("message_timing_hours")
	timingMinutes := state.GetInt("message_timing_minutes")
	buttons := state.GetString(string(wizard.StepMessageButtons))

	// Create pending message
	pendingMsg := wizard.PendingMessage{
		MessageID:     fmt.Sprintf("msg_%d", msgNum),
		TimingHours:   timingHours,
		TimingMinutes: timingMinutes,
		Photos:        photos,
		Buttons:       buttons,
	}
	state.AddPendingMessage(pendingMsg)

	// Also save message text to template (we'll use message ID as template file reference)
	// In a full implementation, this would save to a template file
	// For now, we'll store it in the state for later use
	state.Set(fmt.Sprintf("template_%s", pendingMsg.MessageID), messageText)

	state.IncrementMessagesCreated()
	state.SetLastConfirmedStep(wizard.StepConfirmMessage)
	_ = b.wizardManager.Update(userID, state)

	// Ask if user wants to add more messages
	b.sendAddMoreMessagesPrompt(chatID, state)
}

// handleEditMessage handles edit request from message confirmation
func (b *Bot) handleEditMessage(callback *tgbotapi.CallbackQuery) {
	userID := fmt.Sprint(callback.From.ID)
	chatID := callback.From.ID

	state, err := b.wizardManager.Get(userID)
	if err != nil {
		b.sendWizardError(chatID, "Wizard expired")
		return
	}

	_ = b.wizardManager.Advance(userID, wizard.StepEditMessage)
	b.sendMessageEditSelection(chatID, state)
}

// handleAddMoreYes handles adding another message
func (b *Bot) handleAddMoreYes(callback *tgbotapi.CallbackQuery) {
	userID := fmt.Sprint(callback.From.ID)
	chatID := callback.From.ID

	state, err := b.wizardManager.Get(userID)
	if err != nil {
		b.sendWizardError(chatID, "Wizard expired")
		return
	}

	// Increment message index
	newIndex := state.GetCurrentMessageIndex() + 1
	state.SetCurrentMessageIndex(newIndex)
	_ = b.wizardManager.Update(userID, state)

	// Start new message flow
	_ = b.wizardManager.Advance(userID, wizard.StepMessageText)
	msgNum := newIndex + 1
	b.sendMessage(chatID, fmt.Sprintf("📝 <b>Message %d</b>\n\n%s", msgNum, b.getPromptForStep(wizard.StepMessageText)))
}

// handleAddMoreNo handles finishing scenario creation
func (b *Bot) handleAddMoreNo(callback *tgbotapi.CallbackQuery) {
	userID := fmt.Sprint(callback.From.ID)
	chatID := callback.From.ID

	// Finalize scenario creation
	state, err := b.wizardManager.Get(userID)
	if err != nil {
		b.sendWizardError(chatID, "Wizard expired")
		return
	}

	if err := b.finalizeScenarioWizard(userID, state); err != nil {
		b.sendWizardError(chatID, "Failed to create scenario: "+err.Error())
	} else {
		edit := tgbotapi.NewEditMessageText(chatID, callback.Message.MessageID,
			"✅ Scenario created successfully!\n\n"+
				"Use /scenarios to view all scenarios.")
		edit.ParseMode = "HTML"
		if _, err := b.bot.Send(edit); err != nil {
			log.Printf("Failed to edit final message: %v", err)
		}
	}

	_ = b.wizardManager.Cancel(userID)
}

// handleScenarioInfoCallback handles the scenario info callback
func (b *Bot) handleScenarioInfoCallback(callback *tgbotapi.CallbackQuery) {
	// Extract scenario ID from callback data
	data := callback.Data
	if !strings.HasPrefix(data, "scenario_info_") {
		b.answerCallback(callback.ID, "❌ Invalid callback")
		return
	}

	scenarioID := strings.TrimPrefix(data, "scenario_info_")

	// Get scenario details
	sc, err := b.scenarioService.GetScenario(scenarioID)
	if err != nil {
		log.Printf("[ADMIN] Failed to get scenario %s: %v", scenarioID, err)
		b.answerCallback(callback.ID, "❌ Failed to load scenario")
		return
	}

	// Send scenario info with DEMO button
	if err := b.sendScenarioInfoWithDemo(callback.From.ID, sc); err != nil {
		log.Printf("[ADMIN] Failed to send scenario info: %v", err)
		b.answerCallback(callback.ID, "❌ Failed to show scenario")
		return
	}

	b.answerCallback(callback.ID, "")
}

// handleScenarioDemoCallback handles the scenario demo callback
func (b *Bot) handleScenarioDemoCallback(callback *tgbotapi.CallbackQuery) {
	// Extract scenario ID from callback data
	data := callback.Data
	if !strings.HasPrefix(data, "scenario_demo_") {
		b.answerCallback(callback.ID, "❌ Invalid callback")
		return
	}

	scenarioID := strings.TrimPrefix(data, "scenario_demo_")

	// Get scenario details
	sc, err := b.scenarioService.GetScenario(scenarioID)
	if err != nil {
		log.Printf("[ADMIN] Failed to get scenario %s: %v", scenarioID, err)
		b.answerCallback(callback.ID, "❌ Failed to load scenario")
		return
	}

	// Send demo messages
	if err := b.sendScenarioDemoMessages(callback.From.ID, sc); err != nil {
		log.Printf("[ADMIN] Failed to send demo messages: %v", err)
		b.answerCallback(callback.ID, "❌ Failed to show demo")
		return
	}

	b.answerCallback(callback.ID, "")
}

// answerCallback sends a callback answer
func (b *Bot) answerCallback(callbackID, text string) {
	callbackResponse := tgbotapi.NewCallback(callbackID, text)
	if _, err := b.bot.Send(callbackResponse); err != nil {
		log.Printf("failed to send callback response: %v", err)
	}
}

// sendScenarioInfoWithDemo sends scenario information with a DEMO button
func (b *Bot) sendScenarioInfoWithDemo(chatID int64, sc *domainScenario.Scenario) error {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("<b>Scenario:</b> %s\n", sc.Name))
	sb.WriteString(fmt.Sprintf("<b>ID:</b> <code>%s</code>\n", sc.ID))
	sb.WriteString(fmt.Sprintf("<b>Status:</b> %s\n", map[bool]string{true: "Active", false: "Inactive"}[sc.IsActive]))
	sb.WriteString(fmt.Sprintf("<b>Created:</b> %s\n\n", sc.CreatedAt.Format("02.01.2006 15:04")))

	sb.WriteString("<b>Prodamus Config:</b>\n")
	sb.WriteString(fmt.Sprintf("  Product: %s\n", sc.Config.Prodamus.ProductName))
	sb.WriteString(fmt.Sprintf("  Price: %s ₽\n", sc.Config.Prodamus.ProductPrice))
	sb.WriteString(fmt.Sprintf("  Content: %s\n", sc.Config.Prodamus.PaidContent))
	sb.WriteString(fmt.Sprintf("  Group ID: <code>%s</code>\n\n", sc.Config.Prodamus.PrivateGroupID))

	sb.WriteString(fmt.Sprintf("<b>Messages:</b> %d\n", len(sc.Messages.MessagesList)))

	msg := tgbotapi.NewMessage(chatID, sb.String())
	msg.ParseMode = "HTML"
	msg.DisableWebPagePreview = true

	// Add DEMO button if there are messages
	if len(sc.Messages.MessagesList) > 0 {
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🎬 DEMO - Preview Messages", fmt.Sprintf("scenario_demo_%s", sc.ID)),
			),
		)
		msg.ReplyMarkup = &keyboard
	}

	if _, err := b.bot.Send(msg); err != nil {
		return fmt.Errorf("failed to send scenario info: %w", err)
	}

	return nil
}

// sendScenarioDemoMessages sends actual demo messages from a scenario
func (b *Bot) sendScenarioDemoMessages(chatID int64, sc *domainScenario.Scenario) error {
	// Send each message as a separate message
	for i, msgID := range sc.Messages.MessagesList {
		msgData, ok := sc.Messages.Messages[msgID]
		if !ok {
			continue
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("📬 <b>Message %d</b>\n\n", i+1))
		sb.WriteString(fmt.Sprintf("<i>Timing: %dh %dm after previous</i>\n\n", msgData.Timing.Hours, msgData.Timing.Minutes))

		// Note: We don't parse template here, just show placeholder
		sb.WriteString(fmt.Sprintf("<b>Template:</b> %s\n", msgData.TemplateFile))
		if len(msgData.Photos) > 0 {
			sb.WriteString(fmt.Sprintf("<b>Photos:</b> %d\n", len(msgData.Photos)))
		}

		msg := tgbotapi.NewMessage(chatID, sb.String())
		msg.ParseMode = "HTML"
		msg.DisableWebPagePreview = true

		if _, err := b.bot.Send(msg); err != nil {
			log.Printf("Failed to send demo message %s: %v", msgID, err)
		}
	}

	// Send completion message
	completionMsg := tgbotapi.NewMessage(chatID, "✅ <b>End of scenario preview</b>\n\nAll messages would be sent according to their timing configuration.")
	completionMsg.ParseMode = "HTML"
	if _, err := b.bot.Send(completionMsg); err != nil {
		return fmt.Errorf("failed to send completion message: %w", err)
	}

	return nil
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
