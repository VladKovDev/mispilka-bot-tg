package wizard

import (
	"time"
)

// WizardStep represents a step in the wizard
type WizardStep string

const (
	// General scenario info steps
	StepScenarioName    WizardStep = "scenario_name"
	StepProductName     WizardStep = "product_name"
	StepProductPrice    WizardStep = "product_price"
	StepPaidContent     WizardStep = "paid_content"
	StepPrivateGroupID  WizardStep = "private_group_id"
	StepConfirmGeneral  WizardStep = "confirm_general"
	StepEditGeneral     WizardStep = "edit_general"

	// Summary steps
	StepSummaryMessage  WizardStep = "summary_message"
	StepSummaryPhotos   WizardStep = "summary_photos"
	StepSummaryButtons  WizardStep = "summary_buttons"
	StepConfirmSummary  WizardStep = "confirm_summary"
	StepEditSummary     WizardStep = "edit_summary"

	// Message steps
	StepMessageText     WizardStep = "message_text"
	StepMessagePhotos   WizardStep = "message_photos"
	StepMessageTiming   WizardStep = "message_timing"
	StepMessageButtons  WizardStep = "message_buttons"
	StepConfirmMessage  WizardStep = "confirm_message"
	StepEditMessage     WizardStep = "edit_message"

	// Flow control
	StepAddMoreMessages WizardStep = "add_more_messages"

	// Broadcast steps
	StepBroadcastName   WizardStep = "broadcast_name"
	StepBroadcastText   WizardStep = "broadcast_text"
	StepBroadcastPhotos WizardStep = "broadcast_photos"
	StepConfirmBroadcast WizardStep = "confirm_broadcast"
)

// WizardState represents the state of an active wizard
type WizardState struct {
	UserID      string                 `json:"user_id"`
	WizardType  WizardType             `json:"wizard_type"`
	CurrentStep WizardStep             `json:"current_step"`
	StartedAt   time.Time              `json:"started_at"`
	Timeout     time.Duration          `json:"timeout"`
	Data        map[string]interface{} `json:"data"`
}

// WizardType represents the type of wizard
type WizardType string

const (
	WizardTypeCreateScenario  WizardType = "create_scenario"
	WizardTypeCreateBroadcast WizardType = "create_broadcast"
)

// Expired checks if the wizard has expired
func (w *WizardState) Expired() bool {
	return time.Since(w.StartedAt) > w.Timeout
}

// ResetTimeout resets the wizard timeout
func (w *WizardState) ResetTimeout() {
	w.StartedAt = time.Now()
}

// Set sets a data value
func (w *WizardState) Set(key string, value interface{}) {
	if w.Data == nil {
		w.Data = make(map[string]interface{})
	}
	w.Data[key] = value
}

// Get gets a data value
func (w *WizardState) Get(key string) (interface{}, bool) {
	if w.Data == nil {
		return nil, false
	}
	val, ok := w.Data[key]
	return val, ok
}

// GetString gets a string value
func (w *WizardState) GetString(key string) string {
	val, ok := w.Get(key)
	if !ok {
		return ""
	}
	if str, ok := val.(string); ok {
		return str
	}
	return ""
}

// GetInt gets an int value
func (w *WizardState) GetInt(key string) int {
	val, ok := w.Get(key)
	if !ok {
		return 0
	}
	if i, ok := val.(int); ok {
		return i
	}
	if f, ok := val.(float64); ok {
		return int(f)
	}
	return 0
}

// GetStringSlice gets a string slice value
func (w *WizardState) GetStringSlice(key string) []string {
	val, ok := w.Get(key)
	if !ok {
		return nil
	}
	if slice, ok := val.([]string); ok {
		return slice
	}
	return nil
}

// Clone creates a clone of the wizard state
func (w *WizardState) Clone() *WizardState {
	clone := &WizardState{
		UserID:      w.UserID,
		WizardType:  w.WizardType,
		CurrentStep: w.CurrentStep,
		StartedAt:   w.StartedAt,
		Timeout:     w.Timeout,
		Data:        make(map[string]interface{}),
	}
	for k, v := range w.Data {
		clone.Data[k] = v
	}
	return clone
}

// EditMode and tracking methods

// SetEditMode sets the edit mode and target step
func (w *WizardState) SetEditMode(enabled bool, targetStep WizardStep) {
	w.Set("edit_mode", enabled)
	w.Set("edit_target_step", string(targetStep))
}

// IsEditMode returns true if in edit mode
func (w *WizardState) IsEditMode() bool {
	val, ok := w.Get("edit_mode")
	if !ok {
		return false
	}
	if b, ok := val.(bool); ok {
		return b
	}
	return false
}

// GetEditTargetStep returns the step being edited
func (w *WizardState) GetEditTargetStep() WizardStep {
	val := w.GetString("edit_target_step")
	return WizardStep(val)
}

// SetCurrentSection sets the current wizard section
func (w *WizardState) SetCurrentSection(section string) {
	w.Set("current_section", section)
}

// GetCurrentSection returns the current wizard section
func (w *WizardState) GetCurrentSection() string {
	return w.GetString("current_section")
}

// SetCurrentMessageIndex sets the current message index
func (w *WizardState) SetCurrentMessageIndex(index int) {
	w.Set("current_message_index", index)
}

// GetCurrentMessageIndex returns the current message index
func (w *WizardState) GetCurrentMessageIndex() int {
	return w.GetInt("current_message_index")
}

// IncrementMessagesCreated increments the message counter
func (w *WizardState) IncrementMessagesCreated() {
	count := w.GetMessagesCreated()
	w.Set("messages_created", count+1)
}

// GetMessagesCreated returns the number of messages created
func (w *WizardState) GetMessagesCreated() int {
	return w.GetInt("messages_created")
}

// GetLastConfirmedStep returns the last confirmed step
func (w *WizardState) GetLastConfirmedStep() WizardStep {
	val := w.GetString("last_confirmed_step")
	return WizardStep(val)
}

// SetLastConfirmedStep sets the last confirmed step
func (w *WizardState) SetLastConfirmedStep(step WizardStep) {
	w.Set("last_confirmed_step", string(step))
}

// GetScenarioID returns the scenario ID being created/edited
func (w *WizardState) GetScenarioID() string {
	return w.GetString("scenario_id")
}

// SetScenarioID sets the scenario ID
func (w *WizardState) SetScenarioID(id string) {
	w.Set("scenario_id", id)
}
