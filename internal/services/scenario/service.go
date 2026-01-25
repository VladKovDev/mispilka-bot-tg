package scenario

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	domainScenario "mispilkabot/internal/domain/scenario"
)

var (
	// ErrScenarioAlreadyExists is returned when trying to create a scenario with duplicate ID
	ErrScenarioAlreadyExists = errors.New("scenario already exists")
	// ErrScenarioInUse is returned when trying to delete a scenario that's in use
	ErrScenarioInUse = errors.New("scenario is in use")
)

// Service provides high-level scenario management operations
type Service struct {
	baseDir      string
	scenariosDir string
	registry     *Registry
}

// NewService creates a new scenario service
func NewService(baseDir string) *Service {
	scenariosDir := filepath.Join(baseDir, "scenarios")
	registryPath := filepath.Join(baseDir, "registry.json")

	return &Service{
		baseDir:      baseDir,
		scenariosDir: scenariosDir,
		registry:     NewRegistry(registryPath),
	}
}

// CreateScenarioRequest contains data for creating a new scenario
type CreateScenarioRequest struct {
	ID        string                         `json:"id"`
	Name      string                         `json:"name"`
	Prodamus  domainScenario.ProdamusConfig  `json:"prodamus"`
	CreatedAt *time.Time                     `json:"created_at,omitempty"`
}

// UpdateScenarioRequest contains data for updating a scenario
type UpdateScenarioRequest struct {
	ID        string                            `json:"id"`
	Name      string                            `json:"name,omitempty"`
	Prodamus  *domainScenario.ProdamusConfig    `json:"prodamus,omitempty"`
	IsActive  *bool                             `json:"is_active,omitempty"`
}

// AddMessageRequest contains data for adding a message to a scenario
type AddMessageRequest struct {
	ScenarioID    string                        `json:"scenario_id"`
	MessageID     string                        `json:"message_id"`
	Timing        domainScenario.Timing         `json:"timing"`
	TemplateFile  string                        `json:"template_file,omitempty"`
	Photos        []string                      `json:"photos,omitempty"`
	InlineKeyboard *domainScenario.InlineKeyboardConfig `json:"inline_keyboard,omitempty"`
}

// UpdateMessageRequest contains data for updating a message
type UpdateMessageRequest struct {
	ScenarioID     string                            `json:"scenario_id"`
	MessageID      string                            `json:"message_id"`
	Timing         *domainScenario.Timing            `json:"timing,omitempty"`
	TemplateFile   *string                           `json:"template_file,omitempty"`
	Photos         *[]string                         `json:"photos,omitempty"`
	InlineKeyboard *domainScenario.InlineKeyboardConfig `json:"inline_keyboard,omitempty"`
}

// CreateScenario creates a new scenario with config and empty messages
func (s *Service) CreateScenario(req *CreateScenarioRequest) (*domainScenario.Scenario, error) {
	// Validate request
	if req.ID == "" {
		return nil, domainScenario.ErrInvalidScenarioID
	}
	if req.Name == "" {
		return nil, domainScenario.ErrInvalidScenarioName
	}
	if req.Prodamus.ProductName == "" {
		return nil, domainScenario.ErrInvalidProductName
	}
	if req.Prodamus.ProductPrice == "" {
		return nil, domainScenario.ErrInvalidProductPrice
	}
	if req.Prodamus.PrivateGroupID == "" {
		return nil, domainScenario.ErrInvalidPrivateGroupID
	}

	// Load registry to check for duplicates
	if err := s.registry.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("failed to load registry: %w", err)
	}

	if _, exists := s.registry.Scenarios[req.ID]; exists {
		return nil, ErrScenarioAlreadyExists
	}

	// Set created at if not provided
	createdAt := time.Now()
	if req.CreatedAt != nil {
		createdAt = *req.CreatedAt
	}

	// Create scenario directory
	scenarioDir := filepath.Join(s.scenariosDir, req.ID)
	if err := os.MkdirAll(scenarioDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create scenario directory: %w", err)
	}

	// Create config
	configPath := filepath.Join(scenarioDir, "config.json")
	config := NewConfig(configPath)
	config.ID = req.ID
	config.Name = req.Name
	config.CreatedAt = createdAt.Format(time.RFC3339)
	config.Prodamus = ProdamusConfig{
		ProductName:    req.Prodamus.ProductName,
		ProductPrice:   req.Prodamus.ProductPrice,
		PaidContent:    req.Prodamus.PaidContent,
		PrivateGroupID: req.Prodamus.PrivateGroupID,
	}

	if err := config.Save(); err != nil {
		return nil, fmt.Errorf("failed to save config: %w", err)
	}

	// Create empty messages
	messagesPath := filepath.Join(scenarioDir, "messages.json")
	messages := NewScenarioMessages(messagesPath)
	if err := messages.Save(); err != nil {
		return nil, fmt.Errorf("failed to save messages: %w", err)
	}

	// Update registry
	domainScenario := config.ToScenario()
	s.registry.Scenarios[req.ID] = domainScenario

	if s.registry.DefaultScenarioID == "" {
		s.registry.DefaultScenarioID = req.ID
	}

	if err := s.registry.Save(); err != nil {
		return nil, fmt.Errorf("failed to save registry: %w", err)
	}

	return domainScenario, nil
}

// GetScenario retrieves a scenario by ID
func (s *Service) GetScenario(scenarioID string) (*domainScenario.Scenario, error) {
	// Load registry for metadata
	if err := s.registry.Load(); err != nil {
		return nil, fmt.Errorf("failed to load registry: %w", err)
	}

	// Check if scenario exists in registry or filesystem
	_, inRegistry := s.registry.Scenarios[scenarioID]
	scenarioDir := filepath.Join(s.scenariosDir, scenarioID)
	if _, err := os.Stat(scenarioDir); os.IsNotExist(err) {
		if !inRegistry {
			return nil, domainScenario.ErrScenarioNotFound
		}
	}

	// Load full scenario data from filesystem
	// Load config
	configPath := filepath.Join(scenarioDir, "config.json")
	config := NewConfig(configPath)
	if err := config.Load(); err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Load messages
	messagesPath := filepath.Join(scenarioDir, "messages.json")
	messages := NewScenarioMessages(messagesPath)
	if err := messages.Load(); err != nil && !errors.Is(err, domainScenario.ErrMessageNotFound) {
		return nil, fmt.Errorf("failed to load messages: %w", err)
	}

	// Combine into full scenario
	scenario := config.ToScenario()
	scenario.Messages = messages.ToDomain()

	return scenario, nil
}

// ListScenarios returns all scenarios
func (s *Service) ListScenarios() ([]*domainScenario.Scenario, error) {
	// Load registry for metadata
	if err := s.registry.Load(); err != nil {
		return nil, fmt.Errorf("failed to load registry: %w", err)
	}

	// Scan scenarios directory for all scenario directories
	entries, err := os.ReadDir(s.scenariosDir)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to read scenarios directory: %w", err)
	}

	scenarios := make([]*domainScenario.Scenario, 0)

	// Collect scenario IDs from both registry and filesystem
	scenarioIDs := make(map[string]bool)

	// Add IDs from registry
	for id := range s.registry.Scenarios {
		scenarioIDs[id] = true
	}

	// Add IDs from filesystem
	for _, entry := range entries {
		if entry.IsDir() {
			scenarioIDs[entry.Name()] = true
		}
	}

	// Load each scenario
	for scenarioID := range scenarioIDs {
		scenario, err := s.GetScenario(scenarioID)
		if err != nil {
			log.Printf("[SCENARIO] Warning: failed to load scenario %s: %v", scenarioID, err)
			continue
		}
		scenarios = append(scenarios, scenario)
	}

	return scenarios, nil
}

// UpdateScenario updates scenario configuration
func (s *Service) UpdateScenario(req *UpdateScenarioRequest) (*domainScenario.Scenario, error) {
	if req.ID == "" {
		return nil, domainScenario.ErrInvalidScenarioID
	}

	scenarioDir := filepath.Join(s.scenariosDir, req.ID)
	configPath := filepath.Join(scenarioDir, "config.json")
	config := NewConfig(configPath)

	if err := config.Load(); err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Update fields
	if req.Name != "" {
		config.Name = req.Name
	}
	if req.Prodamus != nil {
		if req.Prodamus.ProductName != "" {
			config.Prodamus.ProductName = req.Prodamus.ProductName
		}
		if req.Prodamus.ProductPrice != "" {
			config.Prodamus.ProductPrice = req.Prodamus.ProductPrice
		}
		if req.Prodamus.PaidContent != "" {
			config.Prodamus.PaidContent = req.Prodamus.PaidContent
		}
		if req.Prodamus.PrivateGroupID != "" {
			config.Prodamus.PrivateGroupID = req.Prodamus.PrivateGroupID
		}
	}

	if err := config.Save(); err != nil {
		return nil, fmt.Errorf("failed to save config: %w", err)
	}

	// Update registry
	if err := s.registry.Load(); err != nil {
		return nil, fmt.Errorf("failed to load registry: %w", err)
	}

	domainScenario := config.ToScenario()
	s.registry.Scenarios[req.ID] = domainScenario

	if err := s.registry.Save(); err != nil {
		return nil, fmt.Errorf("failed to save registry: %w", err)
	}

	return s.GetScenario(req.ID)
}

// DeleteScenario deletes a scenario
func (s *Service) DeleteScenario(scenarioID string) error {
	if err := s.registry.Load(); err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}

	if _, exists := s.registry.Scenarios[scenarioID]; !exists {
		return domainScenario.ErrScenarioNotFound
	}

	// Check if it's the default scenario
	if s.registry.DefaultScenarioID == scenarioID {
		// Check if there are other scenarios
		if len(s.registry.Scenarios) > 1 {
			return ErrScenarioInUse
		}
		s.registry.DefaultScenarioID = ""
	}

	// Remove from registry
	delete(s.registry.Scenarios, scenarioID)

	if err := s.registry.Save(); err != nil {
		return fmt.Errorf("failed to save registry: %w", err)
	}

	// Remove scenario directory
	scenarioDir := filepath.Join(s.scenariosDir, scenarioID)
	if err := os.RemoveAll(scenarioDir); err != nil {
		return fmt.Errorf("failed to remove scenario directory: %w", err)
	}

	return nil
}

// SetDefaultScenario sets the default scenario ID
func (s *Service) SetDefaultScenario(scenarioID string) error {
	if err := s.registry.Load(); err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}

	// Check if scenario exists by checking if its directory exists
	scenarioDir := filepath.Join(s.scenariosDir, scenarioID)
	if _, err := os.Stat(scenarioDir); os.IsNotExist(err) {
		return domainScenario.ErrScenarioNotFound
	}

	s.registry.DefaultScenarioID = scenarioID

	if err := s.registry.Save(); err != nil {
		return fmt.Errorf("failed to save registry: %w", err)
	}

	log.Printf("[SCENARIO] Set default scenario to: %s", scenarioID)
	return nil
}

// GetDefaultScenario returns the default scenario ID
func (s *Service) GetDefaultScenario() (string, error) {
	if err := s.registry.Load(); err != nil {
		return "", fmt.Errorf("failed to load registry: %w", err)
	}

	if s.registry.DefaultScenarioID == "" {
		return "", errors.New("no default scenario set")
	}

	return s.registry.DefaultScenarioID, nil
}

// AddMessage adds a message to a scenario
func (s *Service) AddMessage(req *AddMessageRequest) error {
	scenarioDir := filepath.Join(s.scenariosDir, req.ScenarioID)
	messagesPath := filepath.Join(scenarioDir, "messages.json")
	messages := NewScenarioMessages(messagesPath)

	if err := messages.Load(); err != nil && !errors.Is(err, domainScenario.ErrMessageNotFound) {
		return fmt.Errorf("failed to load messages: %w", err)
	}

	// Add message ID to list if not already present
	found := false
	for _, id := range messages.MessagesList {
		if id == req.MessageID {
			found = true
			break
		}
	}
	if !found {
		messages.MessagesList = append(messages.MessagesList, req.MessageID)
	}

	// Add message data
	messages.Messages[req.MessageID] = domainScenario.MessageData{
		Timing:         req.Timing,
		TemplateFile:   req.TemplateFile,
		Photos:         req.Photos,
		InlineKeyboard: req.InlineKeyboard,
	}

	if err := messages.Save(); err != nil {
		return fmt.Errorf("failed to save messages: %w", err)
	}

	return nil
}

// UpdateMessage updates an existing message in a scenario
func (s *Service) UpdateMessage(req *UpdateMessageRequest) error {
	scenarioDir := filepath.Join(s.scenariosDir, req.ScenarioID)
	messagesPath := filepath.Join(scenarioDir, "messages.json")
	messages := NewScenarioMessages(messagesPath)

	if err := messages.Load(); err != nil {
		return fmt.Errorf("failed to load messages: %w", err)
	}

	msgData, ok := messages.Messages[req.MessageID]
	if !ok {
		return domainScenario.ErrMessageNotFound
	}

	// Update fields
	if req.Timing != nil {
		msgData.Timing = *req.Timing
	}
	if req.TemplateFile != nil {
		msgData.TemplateFile = *req.TemplateFile
	}
	if req.Photos != nil {
		msgData.Photos = *req.Photos
	}
	if req.InlineKeyboard != nil {
		msgData.InlineKeyboard = req.InlineKeyboard
	}

	messages.Messages[req.MessageID] = msgData

	if err := messages.Save(); err != nil {
		return fmt.Errorf("failed to save messages: %w", err)
	}

	return nil
}

// DeleteMessage removes a message from a scenario
func (s *Service) DeleteMessage(scenarioID, messageID string) error {
	scenarioDir := filepath.Join(s.scenariosDir, scenarioID)
	messagesPath := filepath.Join(scenarioDir, "messages.json")
	messages := NewScenarioMessages(messagesPath)

	if err := messages.Load(); err != nil {
		return fmt.Errorf("failed to load messages: %w", err)
	}

	if _, ok := messages.Messages[messageID]; !ok {
		return domainScenario.ErrMessageNotFound
	}

	// Remove from map
	delete(messages.Messages, messageID)

	// Remove from list
	newList := make([]string, 0, len(messages.MessagesList))
	for _, id := range messages.MessagesList {
		if id != messageID {
			newList = append(newList, id)
		}
	}
	messages.MessagesList = newList

	if err := messages.Save(); err != nil {
		return fmt.Errorf("failed to save messages: %w", err)
	}

	return nil
}

// GetNextMessageID returns the next message ID in the sequence, or empty string if last
func (s *Service) GetNextMessageID(scenarioID, currentMessageID string) (string, error) {
	scenario, err := s.GetScenario(scenarioID)
	if err != nil {
		return "", err
	}

	msgList := scenario.Messages.MessagesList
	for i, id := range msgList {
		if id == currentMessageID && i+1 < len(msgList) {
			return msgList[i+1], nil
		}
	}

	return "", nil // No next message
}

// GetFirstMessageID returns the first message ID in the scenario
func (s *Service) GetFirstMessageID(scenarioID string) (string, error) {
	scenario, err := s.GetScenario(scenarioID)
	if err != nil {
		return "", err
	}

	if len(scenario.Messages.MessagesList) == 0 {
		return "", nil
	}

	return scenario.Messages.MessagesList[0], nil
}
