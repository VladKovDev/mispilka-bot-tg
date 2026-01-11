package services

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Placeholder delimiters for message templates
const (
	PlaceholderStart = "{{"
	PlaceholderEnd   = "}}"
)

// Button type constants
const (
	ButtonTypeURL      = "url"
	ButtonTypeCallback = "callback"
)

// Directory paths
const (
	MessagesDataDir  = "data/messages"
	ImagesDataDir    = "assets/images"
	MessageExtension = ".md"
	ImageExtension   = ".PNG"
)

type MessagesList []string

// Timing represents a time offset with hours and minutes
type Timing struct {
	Hours   int `json:"hours"`
	Minutes int `json:"minutes"`
}

type MessageData struct {
	Timing         Timing                `json:"timing"`
	TemplateFile   string                `json:"template_file,omitempty"`
	InlineKeyboard *InlineKeyboardConfig `json:"inline_keyboard,omitempty"`
}

// InlineKeyboardConfig определяет структуру для инлайн-клавиатур
type InlineKeyboardConfig struct {
	Rows []InlineKeyboardRowConfig `json:"rows"`
}

// InlineKeyboardRowConfig представляет одну строку инлайн-кнопок
type InlineKeyboardRowConfig struct {
	Buttons []InlineButtonConfig `json:"buttons"`
}

// InlineButtonConfig представляет одну инлайн-кнопку с поддержкой
// типов "url" и "callback"
type InlineButtonConfig struct {
	Text         string `json:"text"`
	Type         string `json:"type"`
	URL          string `json:"url,omitempty"`
	CallbackData string `json:"callback_data,omitempty"`
}

type MessageMap map[string]MessageData

type Messages struct {
	MessagesList MessagesList `json:"messages_list"`
	Messages     MessageMap   `json:"messages"`
}

func getMessages() (Messages, error) {
	messages, err := ReadJSONRetry[Messages]("data/messages.json", 3)
	if err != nil {
		return Messages{}, fmt.Errorf("failed to read messages: %w", err)
	}
	return messages, nil
}

func getMessageMap() (MessageMap, error) {
	messages, err := getMessages()
	if err != nil {
		return nil, err
	}
	return messages.Messages, nil
}

func getMessageData(messageName string) (MessageData, error) {
	messageMap, err := getMessageMap()
	if err != nil {
		return MessageData{}, err
	}

	messageData, exists := messageMap[messageName]
	if !exists {
		return MessageData{}, fmt.Errorf("message %q not found", messageName)
	}
	return messageData, nil
}

func getMessagesList() (MessagesList, error) {
	messages, err := getMessages()
	if err != nil {
		return nil, err
	}
	return messages.MessagesList.reverse(), nil
}

func (messagesList MessagesList) reverse() MessagesList {
	for i := 0; i < len(messagesList)/2; i++ {
		j := len(messagesList) - 1 - i
		messagesList[i], messagesList[j] = messagesList[j], messagesList[i]
	}
	return messagesList
}

func GetMessageText(messageName string) (string, error) {
	// Проверить, есть ли у сообщения пользовательский template_file
	messageData, err := getMessageData(messageName)
	if err != nil {
		return "", err
	}

	var path string
	if messageData.TemplateFile != "" {
		path = filepath.Join(MessagesDataDir, messageData.TemplateFile)
	} else {
		path = filepath.Join(MessagesDataDir, messageName+MessageExtension)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read message file %q: %w", path, err)
	}
	return string(data), nil
}

// ReplacePlaceholder заменяет плейсхолдеры в формате {{placeholder}} на заданное значение
func ReplacePlaceholder(text, placeholder, value string) string {
	searchPattern := PlaceholderStart + placeholder + PlaceholderEnd
	return strings.ReplaceAll(text, searchPattern, value)
}

// ReplaceAllPlaceholders заменяет все плейсхолдеры формата {{key}} на значения из карты
func ReplaceAllPlaceholders(text string, values map[string]string) string {
	for key, value := range values {
		text = ReplacePlaceholder(text, key, value)
	}
	return text
}

func GetTiming(messageName string) (Timing, error) {
	messageData, err := getMessageData(messageName)
	if err != nil {
		return Timing{}, err
	}
	return messageData.Timing, nil
}

// GetURLButton возвращает первую кнопку-ссылку, найденную в конфигурации сообщения.
// Она ищет в инлайн-клавиатуре кнопки типа "url".
func GetURLButton(messageName string) (string, string, error) {
	messageData, err := getMessageData(messageName)
	if err != nil {
		return "", "", fmt.Errorf("не удалось получить данные сообщения для %s: %w", messageName, err)
	}

	// Проверить формат инлайн-клавиатуры на наличие кнопок-ссылок
	if messageData.InlineKeyboard != nil && len(messageData.InlineKeyboard.Rows) > 0 {
		for _, row := range messageData.InlineKeyboard.Rows {
			for _, btn := range row.Buttons {
				if btn.Type == ButtonTypeURL && btn.Text != "" && btn.URL != "" {
					return btn.URL, btn.Text, nil
				}
			}
		}
	}

	return "", "", fmt.Errorf("конфигурация кнопки-ссылки не найдена для сообщения: %s", messageName)
}

// GetInlineKeyboard возвращает конфигурацию инлайн-клавиатуры для заданного сообщения.
// Поддерживает типы кнопок: url и callback.
func GetInlineKeyboard(messageName string) (*InlineKeyboardConfig, error) {
	messageData, err := getMessageData(messageName)
	if err != nil {
		return nil, fmt.Errorf("не удалось получить данные сообщения для %s: %w", messageName, err)
	}

	// Вернуть инлайн-клавиатуру, если она настроена
	if messageData.InlineKeyboard != nil {
		return messageData.InlineKeyboard, nil
	}

	return nil, nil
}

// ToTelegramKeyboard конвертирует InlineKeyboardConfig в tgbotapi.InlineKeyboardMarkup
// Эта вспомогательная функция интегрирует пользовательскую структуру клавиатуры с API бота Telegram
func ToTelegramKeyboard(config *InlineKeyboardConfig) tgbotapi.InlineKeyboardMarkup {
	if config == nil || len(config.Rows) == 0 {
		return tgbotapi.InlineKeyboardMarkup{}
	}

	var rows [][]tgbotapi.InlineKeyboardButton

	for _, rowConfig := range config.Rows {
		var row []tgbotapi.InlineKeyboardButton
		for _, btnConfig := range rowConfig.Buttons {
			var btn tgbotapi.InlineKeyboardButton
			switch btnConfig.Type {
			case ButtonTypeURL:
				btn = tgbotapi.NewInlineKeyboardButtonURL(btnConfig.Text, btnConfig.URL)
			case ButtonTypeCallback:
				btn = tgbotapi.NewInlineKeyboardButtonData(btnConfig.Text, btnConfig.CallbackData)
			}
			if btn.Text != "" {
				row = append(row, btn)
			}
		}
		if len(row) > 0 {
			rows = append(rows, row)
		}
	}

	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func LastMessage(messagesList MessagesList) (string, error) {
	n := len(messagesList)
	if n == 0 {
		return "", fmt.Errorf("messagesList пуст")
	}
	last := messagesList[n-1]
	return last, nil
}

func GetPhoto(messageName string) (string, error) {
	path := filepath.Join(ImagesDataDir, messageName+ImageExtension)

	_, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("не удалось получить фото: %w", err)
	}
	return path, nil
}
