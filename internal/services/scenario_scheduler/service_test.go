package scenario_scheduler

import (
	"path/filepath"
	"testing"
	"time"

	domainScenario "mispilkabot/internal/domain/scenario"
)

func TestScheduler_ScheduleNextMessage(t *testing.T) {
	scheduler := NewScheduler(filepath.Join(t.TempDir(), "schedules.json"))

	sc := &domainScenario.Scenario{
		ID:   "test",
		Name: "Test",
		Messages: domainScenario.ScenarioMessages{
			MessagesList: []string{"msg_1", "msg_2"},
			Messages: map[string]domainScenario.MessageData{
				"msg_1": {
					Timing: domainScenario.Timing{Hours: 0, Minutes: 0},
				},
				"msg_2": {
					Timing: domainScenario.Timing{Hours: 1, Minutes: 0},
				},
			},
		},
	}

	chatID := "123456"
	state := &domainScenario.UserScenarioState{
		Status:              domainScenario.StatusActive,
		CurrentMessageIndex: 0,
	}

	// Schedule next message
	scheduledTime, err := scheduler.ScheduleNextMessage(chatID, sc, state)
	if err != nil {
		t.Fatalf("Failed to schedule: %v", err)
	}

	if scheduledTime.IsZero() {
		t.Error("Expected scheduled time to be set")
	}
}

func TestScheduler_GetNextMessage(t *testing.T) {
	scheduler := NewScheduler(filepath.Join(t.TempDir(), "schedules.json"))

	sc := &domainScenario.Scenario{
		Messages: domainScenario.ScenarioMessages{
			MessagesList: []string{"msg_1", "msg_2"},
			Messages: map[string]domainScenario.MessageData{
				"msg_1": {
					Timing: domainScenario.Timing{Hours: 0, Minutes: 0},
				},
				"msg_2": {
					Timing: domainScenario.Timing{Hours: 1, Minutes: 0},
				},
			},
		},
	}

	state := &domainScenario.UserScenarioState{
		CurrentMessageIndex: 0,
	}

	// Get next message
	msg, err := scheduler.GetNextMessage(sc, state)
	if err != nil {
		t.Fatalf("Failed to get next message: %v", err)
	}

	if msg.Timing.Hours != 0 {
		t.Errorf("Expected timing 0h, got %dh", msg.Timing.Hours)
	}
}

func TestScheduler_GetNextMessage_NoMoreMessages(t *testing.T) {
	scheduler := NewScheduler(filepath.Join(t.TempDir(), "schedules.json"))

	sc := &domainScenario.Scenario{
		Messages: domainScenario.ScenarioMessages{
			MessagesList: []string{"msg_1"},
			Messages: map[string]domainScenario.MessageData{
				"msg_1": {
					Timing: domainScenario.Timing{Hours: 0, Minutes: 0},
				},
			},
		},
	}

	state := &domainScenario.UserScenarioState{
		CurrentMessageIndex: 1, // Past the end
	}

	// Get next message should fail
	_, err := scheduler.GetNextMessage(sc, state)
	if err == nil {
		t.Error("Expected error when no more messages")
	}
}

func TestScheduler_CancelSchedule(t *testing.T) {
	scheduler := NewScheduler(filepath.Join(t.TempDir(), "schedules.json"))

	sc := &domainScenario.Scenario{
		ID:   "test",
		Name: "Test",
		Messages: domainScenario.ScenarioMessages{
			MessagesList: []string{"msg_1"},
			Messages: map[string]domainScenario.MessageData{
				"msg_1": {
					Timing: domainScenario.Timing{Hours: 1, Minutes: 0},
				},
			},
		},
	}

	state := &domainScenario.UserScenarioState{
		Status:              domainScenario.StatusActive,
		CurrentMessageIndex: 0,
	}

	chatID := "123456"

	// Schedule next message
	_, err := scheduler.ScheduleNextMessage(chatID, sc, state)
	if err != nil {
		t.Fatalf("Failed to schedule: %v", err)
	}

	// Verify schedule exists
	_, ok := scheduler.GetSchedule(chatID)
	if !ok {
		t.Error("Expected schedule to exist")
	}

	// Cancel schedule
	scheduler.CancelSchedule(chatID)

	// Verify schedule is gone
	_, ok = scheduler.GetSchedule(chatID)
	if ok {
		t.Error("Expected schedule to be cancelled")
	}
}

func TestScheduler_GetSchedule(t *testing.T) {
	scheduler := NewScheduler(filepath.Join(t.TempDir(), "schedules.json"))

	// Get non-existent schedule
	_, ok := scheduler.GetSchedule("nonexistent")
	if ok {
		t.Error("Expected no schedule for non-existent user")
	}

	sc := &domainScenario.Scenario{
		ID:   "test",
		Name: "Test",
		Messages: domainScenario.ScenarioMessages{
			MessagesList: []string{"msg_1"},
			Messages: map[string]domainScenario.MessageData{
				"msg_1": {
					Timing: domainScenario.Timing{Hours: 1, Minutes: 0},
				},
			},
		},
	}

	state := &domainScenario.UserScenarioState{
		Status:              domainScenario.StatusActive,
		CurrentMessageIndex: 0,
	}

	chatID := "123456"

	// Schedule next message
	_, err := scheduler.ScheduleNextMessage(chatID, sc, state)
	if err != nil {
		t.Fatalf("Failed to schedule: %v", err)
	}

	// Get schedule
	info, ok := scheduler.GetSchedule(chatID)
	if !ok {
		t.Fatal("Expected schedule to exist")
	}

	if info.ChatID != chatID {
		t.Errorf("Expected ChatID %s, got %s", chatID, info.ChatID)
	}
	if info.ScenarioID != "test" {
		t.Errorf("Expected ScenarioID 'test', got %s", info.ScenarioID)
	}
	if info.MessageIndex != 1 {
		t.Errorf("Expected MessageIndex 1, got %d", info.MessageIndex)
	}
}

func TestScheduler_ExportSchedules(t *testing.T) {
	scheduler := NewScheduler(filepath.Join(t.TempDir(), "schedules.json"))

	sc := &domainScenario.Scenario{
		ID:   "test",
		Name: "Test",
		Messages: domainScenario.ScenarioMessages{
			MessagesList: []string{"msg_1"},
			Messages: map[string]domainScenario.MessageData{
				"msg_1": {
					Timing: domainScenario.Timing{Hours: 1, Minutes: 0},
				},
			},
		},
	}

	state := &domainScenario.UserScenarioState{
		Status:              domainScenario.StatusActive,
		CurrentMessageIndex: 0,
	}

	chatID := "123456"

	// Schedule next message
	_, err := scheduler.ScheduleNextMessage(chatID, sc, state)
	if err != nil {
		t.Fatalf("Failed to schedule: %v", err)
	}

	// Export schedules
	schedules := scheduler.ExportSchedules()
	if len(schedules) != 1 {
		t.Errorf("Expected 1 schedule, got %d", len(schedules))
	}

	if schedules[0].ChatID != chatID {
		t.Errorf("Expected ChatID %s, got %s", chatID, schedules[0].ChatID)
	}
}

func TestScheduler_RestoreSchedules(t *testing.T) {
	scheduler := NewScheduler(filepath.Join(t.TempDir(), "schedules.json"))

	now := time.Now().Add(1 * time.Hour)

	backups := []*ScheduleInfo{
		{
			ChatID:       "123456",
			ScenarioID:   "test",
			MessageIndex: 1,
			ScheduledAt:  now,
		},
	}

	// Restore schedules
	scheduler.RestoreSchedules(backups)

	// Verify schedule was restored
	info, ok := scheduler.GetSchedule("123456")
	if !ok {
		t.Fatal("Expected schedule to be restored")
	}

	if info.ChatID != "123456" {
		t.Errorf("Expected ChatID '123456', got %s", info.ChatID)
	}
}

func TestScheduler_RestoreSchedules_Past(t *testing.T) {
	scheduler := NewScheduler(filepath.Join(t.TempDir(), "schedules.json"))

	// Schedule in the past
	now := time.Now().Add(-1 * time.Hour)

	backups := []*ScheduleInfo{
		{
			ChatID:       "123456",
			ScenarioID:   "test",
			MessageIndex: 1,
			ScheduledAt:  now,
		},
	}

	// Restore schedules - should trigger callback immediately
	scheduler.RestoreSchedules(backups)

	// Select on callback channel with timeout
	select {
	case info := <-scheduler.Callbacks():
		if info.ChatID != "123456" {
			t.Errorf("Expected ChatID '123456', got %s", info.ChatID)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Expected callback for past schedule")
	}
}

func TestScheduler_Callbacks(t *testing.T) {
	scheduler := NewScheduler(filepath.Join(t.TempDir(), "schedules.json"))

	// Verify callbacks channel exists
	callbacks := scheduler.Callbacks()
	if callbacks == nil {
		t.Error("Expected callbacks channel to exist")
	}

	// Verify it's the same channel
	callbacks2 := scheduler.Callbacks()
	if callbacks != callbacks2 {
		t.Error("Expected same callbacks channel")
	}
}
