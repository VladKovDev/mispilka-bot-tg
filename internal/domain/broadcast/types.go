package broadcast

import "time"

// BroadcastRegistry manages all broadcasts
type BroadcastRegistry struct {
	Broadcasts []*Broadcast `json:"broadcasts"`
}

// Broadcast represents a broadcast message
type Broadcast struct {
	ID             string                `json:"id"`
	Name           string                `json:"name"`
	TemplateFile   string                `json:"template_file"`
	Photos         []string              `json:"photos,omitempty"`
	InlineKeyboard *InlineKeyboardConfig `json:"inline_keyboard,omitempty"`
	Targeting      *Targeting            `json:"targeting,omitempty"`
	CreatedAt      time.Time             `json:"created_at"`
}

// InlineKeyboardConfig defines inline keyboard structure (reused from scenario)
type InlineKeyboardConfig struct {
	Rows []InlineKeyboardRowConfig `json:"rows"`
}

// InlineKeyboardRowConfig defines a keyboard row
type InlineKeyboardRowConfig struct {
	Buttons []InlineKeyboardButtonConfig `json:"buttons"`
}

// InlineKeyboardButtonConfig defines a button
type InlineKeyboardButtonConfig struct {
	Type     string `json:"type"`
	Text     string `json:"text"`
	URL      string `json:"url,omitempty"`
	Callback string `json:"callback,omitempty"`
}

// Targeting defines targeting conditions
type Targeting struct {
	Conditions []string `json:"conditions"`
}

// Targeting conditions
const (
	ConditionNoActiveScenario = "no_active_scenario"
	ConditionHasNotPaid       = "has_not_paid"
)

// NewBroadcastRegistry creates a new broadcast registry
func NewBroadcastRegistry() *BroadcastRegistry {
	return &BroadcastRegistry{
		Broadcasts: make([]*Broadcast, 0),
	}
}

// Get retrieves a broadcast by ID
func (r *BroadcastRegistry) Get(id string) (*Broadcast, bool) {
	for _, bc := range r.Broadcasts {
		if bc.ID == id {
			return bc, true
		}
	}
	return nil, false
}

// Add adds a broadcast to the registry
func (r *BroadcastRegistry) Add(bc *Broadcast) {
	r.Broadcasts = append(r.Broadcasts, bc)
}

// Delete removes a broadcast from the registry
func (r *BroadcastRegistry) Delete(id string) bool {
	for i, bc := range r.Broadcasts {
		if bc.ID == id {
			r.Broadcasts = append(r.Broadcasts[:i], r.Broadcasts[i+1:]...)
			return true
		}
	}
	return false
}

// List returns all broadcasts
func (r *BroadcastRegistry) List() []*Broadcast {
	return r.Broadcasts
}
