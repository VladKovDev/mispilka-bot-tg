package services

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"
)

type Tasks map[string]string

func getScheduleBackup() (tasks Tasks, error error) {
	raw, err := os.ReadFile("data/schedule_backup.json")
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(raw, &tasks); err != nil {
		return nil, err
	}
	return tasks, nil
}

func backUpSchedule(chatID string, date time.Time) error {
	tasks, err := getScheduleBackup()
	if err != nil {
		return err
	}

	dateStr := date.Format(time.RFC3339)

	tasks[chatID] = dateStr

	updated, err := json.MarshalIndent(tasks, "", " ")
	if err != nil {
		return fmt.Errorf("marshal error %w", err)
	}

	err = os.WriteFile("data/schedule_backup.json", updated, 0644)
	if err != nil {
		return fmt.Errorf("write file error %w", err)
	}

	return nil
}

func SetSchedules(sendMessage func(string)) error {
	raw, err := os.ReadFile("data/schedule_backup.json")
	if err != nil {
		return err
	}

	var tasks Tasks
	if err := json.Unmarshal(raw, &tasks); err != nil {
		return err
	}
	for k, v := range tasks {
		dateStr := v
		date, err := time.Parse(time.RFC3339, dateStr)
		if err != nil {
			continue
		}
		SetSchedule(date, k, sendMessage)
	}
	return nil
}

func SetSchedule(sendTime time.Time, chatID string, sendMessage func(string)) {
	delay := time.Until(sendTime)
	if delay <= 0 {
		sendMessage(chatID)
		return
	}

	time.AfterFunc(delay, func() {
		sendMessage(chatID)
	})
}

func getDate(chatID string) (date time.Time, error error) {
	tasks, err := getScheduleBackup()
	if err != nil {
		return date, nil
	}
	dateStr, ok := tasks[chatID]
	if ok {
		date, err := time.Parse(time.RFC3339, dateStr)
		if err != nil {
			return date, err
		}
		return date, nil
	} else {
		return time.Now(), nil
	}
}

func SetNextSchedule(chatID string, messageName string, sendMessage func(string)) {
	timing, err := GetTiming(messageName)
	if err != nil {
		log.Printf("timing fetching error: %s", err)
		return
	}
	now := time.Now()

	nextDate, err := setSendTime(now, timing)
	if err != nil {
		log.Printf("sendTime error: %s", err)
		return
	}

	backUpSchedule(chatID, nextDate)

	SetSchedule(nextDate, chatID, sendMessage)
}

func setSendTime(now time.Time, timing []int) (time.Time, error) {
	nextDate := now.Add(time.Duration(timing[0])*time.Hour +
		time.Duration(timing[1])*time.Minute)
	return nextDate, nil
}
