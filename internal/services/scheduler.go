package services

import (
	"fmt"
	"time"
)

type Tasks map[string]string

func getScheduleBackup() (Tasks, error) {
	tasks, err := ReadJSONRetry[Tasks]("data/schedule_backup.json", 3)
	if err != nil {
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

	err = WriteJSONRetry("data/schedule_backup.json", tasks, 3)
	if err != nil {
		return err
	}
	return nil
}

func SetSchedules(sendMessage func(string)) error {
	tasks, err := ReadJSONRetry[Tasks]("data/schedule_backup.json", 3)
	if err != nil {
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

func SetSchedule(sendTime time.Time, userID string, sendMessage func(string)) {
	delay := time.Until(sendTime)
	if delay <= 0 {
		sendMessage(userID)
		return
	}

	time.AfterFunc(delay, func() {
		sendMessage(userID)
	})
}

func SetNextSchedule(chatID string, messageName string, sendMessage func(string)) error {
	timing, err := GetTiming(messageName)
	if err != nil {
		return fmt.Errorf("failed to set next schedule: %w", err)
	}
	now := time.Now()
	nextDate := setSendTime(now, timing)
	if err := backUpSchedule(chatID, nextDate); err != nil {
		return err
	}
	SetSchedule(nextDate, chatID, sendMessage)
	return nil
}

func setSendTime(now time.Time, timing Timing) time.Time {
	return now.Add(time.Duration(timing.Hours)*time.Hour +
		time.Duration(timing.Minutes)*time.Minute)
}
