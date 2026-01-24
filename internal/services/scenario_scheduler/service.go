package scenario_scheduler

import (
	"fmt"
	"sync"
	"time"

	domainScenario "mispilkabot/internal/domain/scenario"
)

// ScheduleInfo contains schedule information
type ScheduleInfo struct {
	ChatID       string
	ScenarioID   string
	MessageIndex int
	ScheduledAt  time.Time
}

// Scheduler manages per-scenario message scheduling
type Scheduler struct {
	mu        sync.RWMutex
	schedules map[string]*ScheduleInfo // chatID -> ScheduleInfo
	timers    map[string]*time.Timer   // chatID -> Timer
	callbacks chan *ScheduleInfo
}

// NewScheduler creates a new scenario scheduler
func NewScheduler() *Scheduler {
	return &Scheduler{
		schedules: make(map[string]*ScheduleInfo),
		timers:    make(map[string]*time.Timer),
		callbacks: make(chan *ScheduleInfo, 100),
	}
}

// ScheduleNextMessage schedules the next message for a user in a scenario
func (s *Scheduler) ScheduleNextMessage(chatID string, sc *domainScenario.Scenario, state *domainScenario.UserScenarioState) (time.Time, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Get next message
	msg, err := s.getNextMessage(sc, state)
	if err != nil {
		return time.Time{}, err
	}

	// Calculate scheduled time
	now := time.Now()
	scheduledAt := now.Add(time.Duration(msg.Timing.Hours)*time.Hour + time.Duration(msg.Timing.Minutes)*time.Minute)

	// Cancel any existing timer for this user
	if timer, ok := s.timers[chatID]; ok {
		timer.Stop()
	}

	// Store schedule info
	info := &ScheduleInfo{
		ChatID:       chatID,
		ScenarioID:   sc.ID,
		MessageIndex: state.CurrentMessageIndex + 1,
		ScheduledAt:  scheduledAt,
	}
	s.schedules[chatID] = info

	// Set timer
	duration := scheduledAt.Sub(now)
	timer := time.AfterFunc(duration, func() {
		s.callbacks <- info
	})
	s.timers[chatID] = timer

	return scheduledAt, nil
}

// GetNextMessage returns the next message to send
func (s *Scheduler) GetNextMessage(sc *domainScenario.Scenario, state *domainScenario.UserScenarioState) (*domainScenario.MessageData, error) {
	return s.getNextMessage(sc, state)
}

// getNextMessage is internal method to get next message
func (s *Scheduler) getNextMessage(sc *domainScenario.Scenario, state *domainScenario.UserScenarioState) (*domainScenario.MessageData, error) {
	if state.CurrentMessageIndex >= len(sc.Messages.MessagesList) {
		return nil, fmt.Errorf("no more messages in scenario")
	}

	msgID := sc.Messages.MessagesList[state.CurrentMessageIndex]
	msg, ok := sc.Messages.Messages[msgID]
	if !ok {
		return nil, fmt.Errorf("message %s not found", msgID)
	}

	// Return pointer to copy to avoid modifying original
	msgCopy := msg
	return &msgCopy, nil
}

// CancelSchedule cancels a pending schedule
func (s *Scheduler) CancelSchedule(chatID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if timer, ok := s.timers[chatID]; ok {
		timer.Stop()
		delete(s.timers, chatID)
	}
	delete(s.schedules, chatID)
}

// GetSchedule retrieves schedule info for a user
func (s *Scheduler) GetSchedule(chatID string) (*ScheduleInfo, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	info, ok := s.schedules[chatID]
	return info, ok
}

// Callbacks returns the callback channel
func (s *Scheduler) Callbacks() <-chan *ScheduleInfo {
	return s.callbacks
}

// RestoreSchedules restores schedules from backup
func (s *Scheduler) RestoreSchedules(backups []*ScheduleInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for _, info := range backups {
		if info.ScheduledAt.Before(now) {
			// Already passed, send immediately
			go func() {
				s.callbacks <- info
			}()
			continue
		}

		// Cancel any existing timer for this user
		if timer, ok := s.timers[info.ChatID]; ok {
			timer.Stop()
		}

		// Schedule for future
		duration := info.ScheduledAt.Sub(now)
		timer := time.AfterFunc(duration, func() {
			s.callbacks <- info
		})
		s.schedules[info.ChatID] = info
		s.timers[info.ChatID] = timer
	}
}

// ExportSchedules exports current schedules for backup
func (s *Scheduler) ExportSchedules() []*ScheduleInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	schedules := make([]*ScheduleInfo, 0, len(s.schedules))
	for _, info := range s.schedules {
		schedules = append(schedules, info)
	}
	return schedules
}
