package services

import (
	"encoding/json"
	"os"
)

type MessagesList []string

type MessageMap map[string]Message

type Message struct {
	Timing []int8 `json:"timing"`
	Name   string `json:"name"`
}

func getMessagesList() ([]string, error) {
	var data MessageMap

	raw, err := os.ReadFile("data/messages.json")
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(raw, &data)
	if err != nil {
		return nil, err
	}

	return data.getList(), nil
}

func (data MessageMap) getList() (keys MessagesList) {
	for k := range data {
		keys = append(keys, k)
	}
	return keys
}
