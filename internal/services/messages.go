package services

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
)

type MessagesList []string

type MessageData struct {
	Timing    []int    `json:"timing"`
	URLButton []string `json:"url_button"`
}
type MessageMap map[string]MessageData

type Messages struct {
	MessagesList MessagesList `json:"messages_list"`
	Messages     MessageMap   `json:"messages"`
}

func getMessages() (messages Messages, error error) {
	raw, err := os.ReadFile("data/messages.json")
	if err != nil {
		log.Printf("readfile error: %v", err)
		return messages, err
	}

	err = json.Unmarshal(raw, &messages)
	if err != nil {
		log.Printf("unmarshal error: %v", err)
		return messages, err
	}
	return messages, nil
}

func getMessageMap() (messageMap MessageMap, error error) {
	messages, err := getMessages()
	if err != nil {
		return messageMap, err
	}
	return messages.Messages, nil
}

func getMessageData(messageName string) (messageData MessageData, error error) {
	messageMap, err := getMessageMap()
	if err != nil {
		return messageData, err
	}
	return messageMap[messageName], nil
}

func getMessagesList() (messagesList []string, error error) {
	messages, err := getMessages()
	if err != nil {
		return messagesList, nil
	}
	return messages.MessagesList, nil
}

func (messagesList MessagesList) reverse() MessagesList {
	for i := 0; i < len(messagesList)/2; i++ {
		j := len(messagesList) - 1 - i
		messagesList[i], messagesList[j] = messagesList[j], messagesList[i]
	}
	return messagesList
}

func GetMessageText(messageName string) (string, error) {
	path := fmt.Sprintf("data/messages/%s.md", messageName)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func GetTiming(messageName string) ([]int, error) {
	messageData, err := getMessageData(messageName)
	if err != nil {
		return nil, err
	}
	return  messageData.Timing, nil
}

func GetURLButton(messageName string) (URL string, text string, error error) {
	messageData, err := getMessageData(messageName)
	if err != nil {
		return "", "", err
	}
	URL_button := messageData.URLButton
	return URL_button[0], URL_button[1], nil
}

func LastMessage(messagesList MessagesList) (string, error) {
	n := len(messagesList)
	if n == 0 {
		return "", fmt.Errorf("messagesList is empty")
	}
	last := messagesList[n-1]
	return last, nil
}
