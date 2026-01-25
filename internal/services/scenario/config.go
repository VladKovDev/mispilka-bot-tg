package scenario

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"mispilkabot/internal/domain/scenario"
)

var (
	ErrConfigLoadFailed = errors.New("failed to load config")
	ErrConfigSaveFailed = errors.New("failed to save config")
)

// Config manages scenario configuration persistence
type Config struct {
	filePath string
	mu       sync.RWMutex

	ID        string         `json:"id"`
	Name      string         `json:"name"`
	CreatedAt string         `json:"created_at"` // ISO 8601
	Prodamus  ProdamusConfig `json:"prodamus"`
}

// ProdamusConfig contains Prodamus payment settings
type ProdamusConfig struct {
	ProductName    string `json:"product_name"`
	ProductPrice   string `json:"product_price"`
	PaidContent    string `json:"paid_content"`
	PrivateGroupID string `json:"private_group_id"`
}

// NewConfig creates a new config
func NewConfig(filePath string) *Config {
	return &Config{
		filePath: filePath,
	}
}

// Load loads the config from disk
func (c *Config) Load() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	data, err := os.ReadFile(c.filePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return scenario.ErrScenarioNotFound
		}
		return ErrConfigLoadFailed
	}

	if err := json.Unmarshal(data, c); err != nil {
		return ErrConfigLoadFailed
	}

	return nil
}

// Save saves the config to disk
func (c *Config) Save() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(c.filePath), 0755); err != nil {
		return ErrConfigSaveFailed
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return ErrConfigSaveFailed
	}

	if err := os.WriteFile(c.filePath, data, 0644); err != nil {
		return ErrConfigSaveFailed
	}

	return nil
}

// ToScenario converts config to domain Scenario (without messages/summary)
func (c *Config) ToScenario() *scenario.Scenario {
	var createdAt time.Time
	if c.CreatedAt != "" {
		parsed, err := time.Parse(time.RFC3339, c.CreatedAt)
		if err == nil {
			createdAt = parsed
		}
	}

	return &scenario.Scenario{
		ID:        c.ID,
		Name:      c.Name,
		CreatedAt: createdAt,
		IsActive:  true,
		Config: scenario.ScenarioConfig{
			Prodamus: scenario.ProdamusConfig{
				ProductName:    c.Prodamus.ProductName,
				ProductPrice:   c.Prodamus.ProductPrice,
				PaidContent:    c.Prodamus.PaidContent,
				PrivateGroupID: c.Prodamus.PrivateGroupID,
			},
		},
	}
}
