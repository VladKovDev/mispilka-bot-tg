package broadcast

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	domainBroadcast "mispilkabot/internal/domain/broadcast"
)

// Service manages broadcast operations
type Service struct {
	filePath string
	mu       sync.RWMutex
	registry *domainBroadcast.BroadcastRegistry
}

// NewService creates a new broadcast service
func NewService(baseDir string) *Service {
	filePath := filepath.Join(baseDir, "broadcasts.json")
	return &Service{
		filePath: filePath,
		registry: domainBroadcast.NewBroadcastRegistry(),
	}
}

// Load loads broadcasts from disk
func (s *Service) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.filePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			s.registry = domainBroadcast.NewBroadcastRegistry()
			return nil
		}
		return ErrLoadFailed
	}

	if err := json.Unmarshal(data, s.registry); err != nil {
		return ErrLoadFailed
	}

	return nil
}

// Save saves broadcasts to disk
func (s *Service) Save() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.saveLocked()
}

// saveLocked saves broadcasts to disk (caller must hold lock)
func (s *Service) saveLocked() error {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(s.filePath), 0755); err != nil {
		return ErrSaveFailed
	}

	data, err := json.MarshalIndent(s.registry, "", "  ")
	if err != nil {
		return ErrSaveFailed
	}

	if err := os.WriteFile(s.filePath, data, 0644); err != nil {
		return ErrSaveFailed
	}

	return nil
}

// loadLocked loads broadcasts from disk (caller must hold lock)
func (s *Service) loadLocked() error {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			s.registry = domainBroadcast.NewBroadcastRegistry()
			return nil
		}
		return ErrLoadFailed
	}

	if err := json.Unmarshal(data, s.registry); err != nil {
		return ErrLoadFailed
	}

	return nil
}

// CreateBroadcast creates a new broadcast
func (s *Service) CreateBroadcast(req *CreateBroadcastRequest) (*domainBroadcast.Broadcast, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.loadLocked(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	// Check for duplicate ID
	if _, exists := s.registry.Get(req.ID); exists {
		return nil, errors.New("broadcast already exists")
	}

	broadcast := &domainBroadcast.Broadcast{
		ID:             req.ID,
		Name:           req.Name,
		TemplateFile:   req.TemplateFile,
		Photos:         req.Photos,
		InlineKeyboard: req.InlineKeyboard,
		Targeting:      req.Targeting,
		CreatedAt:      time.Now(),
	}

	s.registry.Add(broadcast)

	return broadcast, s.saveLocked()
}

// GetBroadcast retrieves a broadcast by ID
func (s *Service) GetBroadcast(id string) (*domainBroadcast.Broadcast, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	broadcast, exists := s.registry.Get(id)
	if !exists {
		return nil, ErrBroadcastNotFound
	}

	return broadcast, nil
}

// ListBroadcasts returns all broadcasts
func (s *Service) ListBroadcasts() ([]*domainBroadcast.Broadcast, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.registry.List(), nil
}

// UpdateBroadcast updates an existing broadcast
func (s *Service) UpdateBroadcast(req *UpdateBroadcastRequest) (*domainBroadcast.Broadcast, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.loadLocked(); err != nil {
		return nil, err
	}

	broadcast, exists := s.registry.Get(req.ID)
	if !exists {
		return nil, ErrBroadcastNotFound
	}

	// Update fields
	if req.Name != nil {
		broadcast.Name = *req.Name
	}
	if req.TemplateFile != nil {
		broadcast.TemplateFile = *req.TemplateFile
	}
	if req.Photos != nil {
		broadcast.Photos = *req.Photos
	}
	if req.InlineKeyboard != nil {
		broadcast.InlineKeyboard = req.InlineKeyboard
	}
	if req.Targeting != nil {
		broadcast.Targeting = req.Targeting
	}

	return broadcast, s.saveLocked()
}

// DeleteBroadcast deletes a broadcast
func (s *Service) DeleteBroadcast(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.loadLocked(); err != nil {
		return err
	}

	if !s.registry.Delete(id) {
		return ErrBroadcastNotFound
	}

	return s.saveLocked()
}

// CreateBroadcastRequest contains data for creating a broadcast
type CreateBroadcastRequest struct {
	ID             string                               `json:"id"`
	Name           string                               `json:"name"`
	TemplateFile   string                               `json:"template_file,omitempty"`
	Photos         []string                             `json:"photos,omitempty"`
	InlineKeyboard *domainBroadcast.InlineKeyboardConfig `json:"inline_keyboard,omitempty"`
	Targeting      *domainBroadcast.Targeting            `json:"targeting,omitempty"`
}

// UpdateBroadcastRequest contains data for updating a broadcast
type UpdateBroadcastRequest struct {
	ID             string                                `json:"id"`
	Name           *string                               `json:"name,omitempty"`
	TemplateFile   *string                               `json:"template_file,omitempty"`
	Photos         *[]string                             `json:"photos,omitempty"`
	InlineKeyboard *domainBroadcast.InlineKeyboardConfig `json:"inline_keyboard,omitempty"`
	Targeting      *domainBroadcast.Targeting            `json:"targeting,omitempty"`
}
