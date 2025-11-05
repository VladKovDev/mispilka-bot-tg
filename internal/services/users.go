package services

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type User struct {
	UserName     string   `json:"user_name"`
	RegTime      string   `json:"reg_time"`
	IsMessaging  bool     `json:"is_messaging"`
	MessagesList []string `json:"messages_list"`
}

type UserMap map[string]User

func getUsers()(data UserMap){
	raw, err := os.ReadFile("data/users.json")
	if err != nil {
		log.Printf("read file error %v", err)
		return
	}

	if err := json.Unmarshal(raw, &data); err != nil {
		log.Printf("unmarshal error %v", err)
		return
	}
	return data
}

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
	t := time.Now()
	strTime := t.Format(time.RFC3339)
	messagesList, err := getMessagesList()
	if err != nil {
		log.Printf("get messagesList error %v", err)
	}
	chatID := strconv.FormatInt(message.Chat.ID, 10)
	data[chatID] = User{
		UserName:     message.From.UserName,
		RegTime:      strTime,
		IsMessaging:  false,
		MessagesList: messagesList,
	}
}

func GetPerson(chatID string) (User, error) {
	var data UserMap
	var user User

	raw, err := os.ReadFile("data/users.json")
	if err != nil {
		log.Printf("readfile error: %v", err)
		return user, err
	}

	if err := json.Unmarshal(raw, &data); err != nil {
		log.Printf("unmarshal error: %v", err)
		return user, err
	}

	user = data[chatID]
	return user, err
}

func ChangeIsMessagingStatus(chatID string, status bool){
	userData, err := GetPerson(chatID)
	if err != nil{
		log.Printf("get person error %v", err)
		return
	}

	userData.IsMessaging = status

	ChangePerson(chatID, userData)
}

func ChangePerson(chatID string, userData User) {
	users := getUsers()

	users[chatID] = userData

	updated, err := json.MarshalIndent(users, "", " ")
	if err != nil {
		log.Printf("marshal error %v", err)
		return
	}

	err = os.WriteFile("data/users.json", updated, 0644)
	if err != nil {
		log.Printf("write file error %v", err)
		return
	}
}

func IsNewPerson(chatID string)bool{
	users := getUsers()
	_, ok := users[chatID]
	if ok{
		return false
	}else{
		return true
	}
}
