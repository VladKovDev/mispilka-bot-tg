package scenario

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	domainScenario "mispilkabot/internal/domain/scenario"
)

// MessageBuilder helps build messages from wizard data
type MessageBuilder struct {
	scenarioID string
	msgIndex   int
}

// NewMessageBuilder creates a new message builder
func NewMessageBuilder(scenarioID string, msgIndex int) *MessageBuilder {
	return &MessageBuilder{
		scenarioID: scenarioID,
		msgIndex:   msgIndex,
	}
}

// GenerateMessageID generates a unique message ID
func (mb *MessageBuilder) GenerateMessageID() string {
	return fmt.Sprintf("msg_%d", mb.msgIndex+1)
}

// ParseTiming parses timing string like "1h 30m"
func (mb *MessageBuilder) ParseTiming(timingStr string) (domainScenario.Timing, error) {
	timing := domainScenario.Timing{}

	// Default to 0 if empty
	if timingStr == "" || timingStr == "skip" {
		return timing, nil
	}

	// Parse "Xh Ym" format
	hoursRegex := regexp.MustCompile(`(\d+)h`)
	minutesRegex := regexp.MustCompile(`(\d+)m`)

	if hoursMatch := hoursRegex.FindStringSubmatch(timingStr); len(hoursMatch) > 1 {
		hours, err := strconv.Atoi(hoursMatch[1])
		if err != nil {
			return timing, fmt.Errorf("invalid hours: %w", err)
		}
		timing.Hours = hours
	}

	if minutesMatch := minutesRegex.FindStringSubmatch(timingStr); len(minutesMatch) > 1 {
		minutes, err := strconv.Atoi(minutesMatch[1])
		if err != nil {
			return timing, fmt.Errorf("invalid minutes: %w", err)
		}
		timing.Minutes = minutes
	}

	return timing, nil
}

// ParseButtonRow parses a button row from string format
func (mb *MessageBuilder) ParseButtonRow(rowStr string) ([]domainScenario.InlineKeyboardButtonConfig, error) {
	if rowStr == "" || rowStr == "skip" {
		return nil, nil
	}

	// Format: "type|text|url|callback;type|text|url|callback"
	buttons := make([]domainScenario.InlineKeyboardButtonConfig, 0)
	buttonStrings := strings.Split(rowStr, ";")

	for _, btnStr := range buttonStrings {
		parts := strings.Split(btnStr, "|")
		if len(parts) < 3 {
			return nil, fmt.Errorf("invalid button format: %s", btnStr)
		}

		button := domainScenario.InlineKeyboardButtonConfig{
			Type: parts[0],
			Text: parts[1],
		}

		if button.Type == "url" {
			button.URL = parts[2]
		} else if button.Type == "callback" {
			button.Callback = parts[2]
		}

		buttons = append(buttons, button)
	}

	return buttons, nil
}

// ParseKeyboard parses full keyboard configuration
func (mb *MessageBuilder) ParseKeyboard(keyboardStr string) (*domainScenario.InlineKeyboardConfig, error) {
	if keyboardStr == "" || keyboardStr == "skip" {
		return nil, nil
	}

	rows := strings.Split(keyboardStr, "\n")
	config := &domainScenario.InlineKeyboardConfig{
		Rows: make([]domainScenario.InlineKeyboardRowConfig, 0, len(rows)),
	}

	for _, rowStr := range rows {
		rowStr = strings.TrimSpace(rowStr)
		if rowStr == "" {
			continue
		}

		buttons, err := mb.ParseButtonRow(rowStr)
		if err != nil {
			return nil, err
		}

		if len(buttons) > 0 {
			config.Rows = append(config.Rows, domainScenario.InlineKeyboardRowConfig{
				Buttons: buttons,
			})
		}
	}

	if len(config.Rows) == 0 {
		return nil, nil
	}

	return config, nil
}

// BuildAddMessageRequest builds an AddMessageRequest from wizard data
// The data map should contain:
// - message_text (string): the message text content
// - message_photos ([]string): list of photo file IDs
// - message_timing_hours (int): hours delay
// - message_timing_minutes (int): minutes delay
// - message_buttons (string): keyboard configuration string
func (mb *MessageBuilder) BuildAddMessageRequest(data map[string]interface{}) (*AddMessageRequest, error) {
	// Get message data from wizard state
	_, _ = data["message_text"].(string) // We don't use this directly, it goes to a template
	photosSlice, _ := data["message_photos"].([]string)

	// Get timing from separate hours/minutes values (wizard state format)
	timingHours, _ := data["message_timing_hours"].(int)
	timingMinutes, _ := data["message_timing_minutes"].(int)

	timing := domainScenario.Timing{
		Hours:   timingHours,
		Minutes: timingMinutes,
	}

	// Get keyboard string
	keyboardStr, _ := data["message_buttons"].(string)

	// Parse keyboard
	keyboard, err := mb.ParseKeyboard(keyboardStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse keyboard: %w", err)
	}

	return &AddMessageRequest{
		ScenarioID:     mb.scenarioID,
		MessageID:      mb.GenerateMessageID(),
		Timing:         timing,
		TemplateFile:   "", // Will be set when template is saved
		Photos:         photosSlice,
		InlineKeyboard: keyboard,
	}, nil
}
