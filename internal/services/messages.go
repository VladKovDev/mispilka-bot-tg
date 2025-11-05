package services

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
)

type MessageData struct {
	Timing    []int    `json:"timing"`
	UrlButton []string `json:"url_button"`
}

type MessagesList []string

type MessageMap map[string]MessageData

func getMessagesData() (MessageMap, error) {
	var data MessageMap

	raw, err := os.ReadFile("data/messages.json")
	if err != nil {
		log.Printf("readfile error: %v", err)
		return nil, err
	}

	err = json.Unmarshal(raw, &data)
	if err != nil {
		log.Printf("unmarshal error: %v", err)
		return nil, err
	}
	return data, nil
}

func getMessagesList() ([]string, error) {
	var messagesList []string
	data, err := getMessagesData()
	if err != nil {
		log.Printf("message data fetching error: %s", err)
		return messagesList, err
	}
	messagesList = data.getList().reverse()
	return messagesList, nil
}

func (data MessagesList) reverse() MessagesList {
	for i := 0; i < len(data)/2; i++ {
		j := len(data) - 1 - i
		data[i], data[j] = data[j], data[i]
	}
	return data
}

func (data MessageMap) getList() (keys MessagesList) {
	for k := range data {
		keys = append(keys, k)
	}
	return keys
}

func GetMessage(messageName string) (string, error) {
	path := fmt.Sprintf("data/messages/%s.md", messageName)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func GetTiming(messageName string) ([]int, error) {
	data, err := getMessagesData()
	if err != nil {
		return nil, err
	}

	messageData := data[messageName]
	timing := messageData.Timing

	return timing, nil
}

func GetUrlButton(messageName string) (url string, text string, error error) {
	data, err := getMessagesData()
	if err != nil {
		return "", "", err
	}

	messageData := data[messageName]
	url_button := messageData.UrlButton

	return url_button[0], url_button[1], nil
}

func LastMessage(messagesList MessagesList) (string, error) {
	n := len(messagesList)
	if n == 0 {
		return "", fmt.Errorf("messagesList is empty")
	}
	last := messagesList[n-1]
	return last, nil
}
