package services

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type User struct {
	UserName     string `json:"user_name"`
	IsMessaging  bool   `json:"is_messaging"`
	MessagesList []string  `json:"messages_list"`
}

type UserMap map[string]User

func AddPerson(message *tgbotapi.Message) error {
	var data UserMap

	raw, err := os.ReadFile("data/users.json")
	if err != nil {
		return fmt.Errorf("read file error: %w", err)
	}

	if err := json.Unmarshal(raw, &data); err != nil {
		return fmt.Errorf("unmarshal error: %w", err)
	}

	data.personData(message)

	updated, err := json.MarshalIndent(data, "", " ")
	if err != nil {
		return fmt.Errorf("marshal error %w", err)
	}

	err = os.WriteFile("data/users.json", updated, 0644)
	if err != nil {
		return fmt.Errorf("write file error %w", err)
	}

	return nil
}

func (data UserMap) personData(message *tgbotapi.Message) {
	messagesList, err := getMessagesList()
	if err != nil{
		fmt.Errorf("get messagesList error %w", err)
	}
	chatID := strconv.FormatInt(message.Chat.ID, 10)
	data[chatID] = User{
		UserName:     message.From.UserName,
		IsMessaging:  true,
		MessagesList: messagesList,
	}
}
