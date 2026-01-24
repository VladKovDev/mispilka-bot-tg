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
