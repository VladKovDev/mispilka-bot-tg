package scenario

import "time"

// Scenario represents a complete messaging scenario
type Scenario struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	CreatedAt time.Time       `json:"created_at"`
	IsActive  bool            `json:"is_active"`
	Config    ScenarioConfig  `json:"config"`
	Messages  ScenarioMessages `json:"messages"`
	Summary   ScenarioSummary `json:"summary"`
}

// ScenarioConfig contains scenario configuration
type ScenarioConfig struct {
	Prodamus ProdamusConfig `json:"prodamus"`
}

// ProdamusConfig contains Prodamus payment settings
type ProdamusConfig struct {
	ProductName    string `json:"product_name"`
	ProductPrice   string `json:"product_price"`
	PaidContent    string `json:"paid_content"`
	PrivateGroupID string `json:"private_group_id"`
}

// ScenarioMessages contains the message flow
type ScenarioMessages struct {
	MessagesList []string              `json:"messages_list"`
	Messages     map[string]MessageData `json:"messages"`
}

// MessageData defines a single message in the flow
type MessageData struct {
	Timing         Timing                `json:"timing"`
	TemplateFile   string                `json:"template_file,omitempty"`
	Photos         []string              `json:"photos,omitempty"`
	InlineKeyboard *InlineKeyboardConfig `json:"inline_keyboard,omitempty"`
}

// Timing defines when to send the message relative to previous
type Timing struct {
	Hours   int `json:"hours"`
	Minutes int `json:"minutes"`
}

// InlineKeyboardConfig defines inline keyboard structure
type InlineKeyboardConfig struct {
	ButtonSetRef string                      `json:"button_set_ref,omitempty"`
	Rows         []InlineKeyboardRowConfig   `json:"rows,omitempty"`
}

// InlineKeyboardRowConfig defines a keyboard row
type InlineKeyboardRowConfig struct {
	Buttons []InlineKeyboardButtonConfig `json:"buttons"`
}

// InlineKeyboardButtonConfig defines a button
type InlineKeyboardButtonConfig struct {
	Type     string `json:"type"` // url, callback
	Text     string `json:"text"`
	URL      string `json:"url,omitempty"`
	Callback string `json:"callback,omitempty"`
}

// ScenarioSummary defines the summary message
type ScenarioSummary struct {
	TemplateFile   string                `json:"template_file"`
	Photos         []string              `json:"photos,omitempty"`
	InlineKeyboard *InlineKeyboardConfig `json:"inline_keyboard,omitempty"`
}

// Validate validates the scenario
func (s *Scenario) Validate() error {
	if s.ID == "" {
		return ErrInvalidScenarioID
	}
	if s.Name == "" {
		return ErrInvalidScenarioName
	}
	if s.Config.Prodamus.ProductName == "" {
		return ErrInvalidProductName
	}
	if s.Config.Prodamus.ProductPrice == "" {
		return ErrInvalidProductPrice
	}
	if s.Config.Prodamus.PrivateGroupID == "" {
		return ErrInvalidPrivateGroupID
	}
	return nil
}
