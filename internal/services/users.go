package services

import (
	"fmt"
	"log"
	"strconv"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type User struct {
	UserName     string    `json:"user_name"`
	RegTime      time.Time `json:"reg_time"`
	IsMessaging  bool      `json:"is_messaging"`
	MessagesList []string  `json:"messages_list"`
}

type UserMap map[string]User

func AddUser(message *tgbotapi.Message) error {
	data, err := ReadJSONRetry[UserMap]("data/users.json", 3)
	if err != nil {
		return err
	}

	data.userData(message)

	if err = WriteJSONRetry[UserMap]("data/users.json", data, 3); err != nil {
		return err
	}
	return nil
}

func (data UserMap) userData(message *tgbotapi.Message) error {
	t := time.Now()
	messagesList, err := getMessagesList()
	if err != nil {
		return err
	}
	chatID := strconv.FormatInt(message.Chat.ID, 10)
	data[chatID] = User{
		UserName:     message.From.UserName,
		RegTime:      t,
		IsMessaging:  false,
		MessagesList: messagesList,
	}
	return nil
}

func GetUser(chatID string) (User, error) {
	var users UserMap
	var user User

	users, err := ReadJSONRetry[UserMap]("data/users.json", 3)
	if err != nil {
		return user, err
	}

	user, ok := users[chatID]
	if !ok {
		return user, fmt.Errorf("user not found")
	}
	return user, nil
}

func ChangeIsMessaging(chatID string, status bool) error {
	userData, err := GetUser(chatID)
	if err != nil {
		return err
	}
	userData.IsMessaging = status
	err = ChangeUser(chatID, userData)
	if err != nil {
		return err
	}
	return nil
}

func ChangeUser(chatID string, userData User) error {
	users, err := ReadJSONRetry[UserMap]("data/users.json", 3)
	if err != nil {
		return err
	}

	users[chatID] = userData

	if err := WriteJSONRetry[UserMap]("data/users.json", users, 3); err != nil {
		return err
	}
	return nil
}

func IsNewUser(chatID string) bool {
	users, err := ReadJSON[UserMap]("data/users.json")
	if err != nil {
		log.Printf("Failed to load users: %v", err)
		return false
	}
	_, ok := users[chatID]
	if ok {
		return false
	} else {
		return true
	}
}
