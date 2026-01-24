# Multi-Scenario Messaging System Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement a multi-scenario messaging system that allows the Telegram bot to support multiple independent message flows (scenarios) instead of a single global message queue.

**Architecture:** Scenario-centric architecture where each scenario is a self-contained entity with its own configuration, message flow, Prodamus payment settings, and summary message. Users can have one active scenario at a time, with support for scenario switching and completion tracking.

**Tech Stack:** Go 1.22.2, Telegram Bot API, JSON file storage, Prodamus payment processor

---

## Table of Contents

1. [Phase 0: Prerequisites & Testing Setup](#phase-0-prerequisites--testing-setup)
2. [Phase 1: Core Domain Types](#phase-1-core-domain-types)
3. [Phase 2: Storage Layer](#phase-2-storage-layer)
4. [Phase 3: Template System](#phase-3-template-system)
5. [Phase 4: Scenario Services](#phase-4-scenario-services)
6. [Phase 5: User Scenario Management](#phase-5-user-scenario-management)
7. [Phase 6: Scenario Scheduler](#phase-6-scenario-scheduler)
8. [Phase 7: Broadcast System](#phase-7-broadcast-system)
9. [Phase 8: Button Registry](#phase-8-button-registry)
10. [Phase 9: Wizard System](#phase-9-wizard-system)
11. [Phase 10: Admin Commands](#phase-10-admin-commands)
12. [Phase 11: Migration Script](#phase-11-migration-script)
13. [Phase 12: Update Handlers](#phase-12-update-handlers)
14. [Phase 13: Testing & Verification](#phase-13-testing--verification)

---

## Phase 0: Prerequisites & Testing Setup

### Task 0.1: Create Test Infrastructure

**Files:**
- Create: `internal/services/scenario/registry_test.go`

**Step 1: Create test file structure**

```go
package scenario

import (
    "testing"
    "time"
)

func TestScenarioRegistry_Create(t *testing.T) {
    // Test will be implemented in Task 1.1
    t.Skip("Not implemented yet")
}
```

**Step 2: Verify test file compiles**

Run: `go build ./internal/services/scenario/...`
Expected: Success (no compilation errors)

**Step 3: Run test to verify skip**

Run: `go test ./internal/services/scenario/... -v`
Expected: PASS with skip message

**Step 4: Commit**

```bash
git add internal/services/scenario/registry_test.go
git commit -m "test: add test infrastructure for scenario registry"
```

---

## Phase 1: Core Domain Types

### Task 1.1: Create Scenario Domain Types

**Files:**
- Create: `internal/domain/scenario/types.go`
- Create: `internal/domain/scenario/status.go`
- Test: `internal/domain/scenario/types_test.go`

**Step 1: Write the failing test for scenario types**

Create: `internal/domain/scenario/types_test.go`

```go
package scenario

import (
    "testing"
    "time"
)

func TestScenario_Validate(t *testing.T) {
    scenario := &Scenario{
        ID:       "test-scenario",
        Name:     "Test Scenario",
        IsActive: true,
        Config: ScenarioConfig{
            Prodamus: ProdamusConfig{
                ProductName:    "Test Product",
                ProductPrice:   "1000",
                PaidContent:    "Thank you!",
                PrivateGroupID: "-1001234567890",
            },
        },
    }

    err := scenario.Validate()
    if err != nil {
        t.Fatalf("Expected valid scenario, got error: %v", err)
    }
}

func TestScenarioStatus_String(t *testing.T) {
    tests := []struct {
        status   ScenarioStatus
        expected string
    }{
        {StatusNotStarted, "not_started"},
        {StatusActive, "active"},
        {StatusCompleted, "completed"},
        {StatusStopped, "stopped"},
    }

    for _, tt := range tests {
        if tt.status.String() != tt.expected {
            t.Errorf("Expected %s, got %s", tt.expected, tt.status.String())
        }
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/domain/scenario/... -v`
Expected: FAIL with "undefined: Scenario" or similar

**Step 3: Write minimal implementation**

Create: `internal/domain/scenario/types.go`

```go
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
    ButtonSetRef string `json:"button_set_ref,omitempty"`
    Rows         []InlineKeyboardRowConfig `json:"rows,omitempty"`
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
```

Create: `internal/domain/scenario/status.go`

```go
package scenario

import "errors"

var (
    ErrInvalidScenarioID     = errors.New("invalid scenario ID")
    ErrInvalidScenarioName   = errors.New("invalid scenario name")
    ErrInvalidProductName    = errors.New("invalid product name")
    ErrInvalidProductPrice   = errors.New("invalid product price")
    ErrInvalidPrivateGroupID = errors.New("invalid private group ID")
    ErrScenarioNotFound      = errors.New("scenario not found")
)

// ScenarioStatus represents user's status in a scenario
type ScenarioStatus string

const (
    StatusNotStarted ScenarioStatus = "not_started"
    StatusActive     ScenarioStatus = "active"
    StatusCompleted  ScenarioStatus = "completed"
    StatusStopped    ScenarioStatus = "stopped"
)

// String returns the string representation
func (s ScenarioStatus) String() string {
    return string(s)
}

// IsValid checks if the status is valid
func (s ScenarioStatus) IsValid() bool {
    switch s {
    case StatusNotStarted, StatusActive, StatusCompleted, StatusStopped:
        return true
    }
    return false
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/domain/scenario/... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/domain/scenario/
git commit -m "feat: add core scenario domain types"
```

---

### Task 1.2: Create User Scenario State Types

**Files:**
- Create: `internal/domain/scenario/user_state.go`
- Test: `internal/domain/scenario/user_state_test.go`

**Step 1: Write the failing test**

Create: `internal/domain/scenario/user_state_test.go`

```go
package scenario

import (
    "testing"
    "time"
)

func TestUserScenarioState_IsCompleted(t *testing.T) {
    state := &UserScenarioState{
        Status: StatusCompleted,
    }

    if !state.IsCompleted() {
        t.Error("Expected state to be completed")
    }
}

func TestUserScenarioState_IsActive(t *testing.T) {
    state := &UserScenarioState{
        Status: StatusActive,
    }

    if !state.IsActive() {
        t.Error("Expected state to be active")
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/domain/scenario/... -v`
Expected: FAIL with "undefined: UserScenarioState"

**Step 3: Write minimal implementation**

Create: `internal/domain/scenario/user_state.go`

```go
package scenario

import "time"

// UserScenarioState tracks user's progress in a scenario
type UserScenarioState struct {
    Status              ScenarioStatus `json:"status"`
    CurrentMessageIndex int            `json:"current_message_index"`
    LastSentAt          *time.Time     `json:"last_sent_at,omitempty"`
    CompletedAt         *time.Time     `json:"completed_at,omitempty"`
    PaymentDate         *time.Time     `json:"payment_date,omitempty"`
    PaymentLink         string         `json:"payment_link,omitempty"`
    InviteLink          string         `json:"invite_link,omitempty"`
    JoinedGroup         bool           `json:"joined_group,omitempty"`
    JoinedAt            *time.Time     `json:"joined_at,omitempty"`
}

// IsCompleted returns true if scenario is completed
func (s *UserScenarioState) IsCompleted() bool {
    return s.Status == StatusCompleted
}

// IsActive returns true if scenario is active
func (s *UserScenarioState) IsActive() bool {
    return s.Status == StatusActive
}

// IsNotStarted returns true if scenario is not started
func (s *UserScenarioState) IsNotStarted() bool {
    return s.Status == StatusNotStarted
}

// IsStopped returns true if scenario is stopped
func (s *UserScenarioState) IsStopped() bool {
    return s.Status == StatusStopped
}

// MarkCompleted marks scenario as completed
func (s *UserScenarioState) MarkCompleted() {
    now := time.Now()
    s.Status = StatusCompleted
    s.CompletedAt = &now
}

// MarkActive marks scenario as active
func (s *UserScenarioState) MarkActive() {
    s.Status = StatusActive
}

// MarkStopped marks scenario as stopped
func (s *UserScenarioState) MarkStopped() {
    s.Status = StatusStopped
}

// Clone creates a deep copy of the state
func (s *UserScenarioState) Clone() *UserScenarioState {
    clone := *s
    if clone.LastSentAt != nil {
        t := *clone.LastSentAt
        clone.LastSentAt = &t
    }
    if clone.CompletedAt != nil {
        t := *clone.CompletedAt
        clone.CompletedAt = &t
    }
    if clone.PaymentDate != nil {
        t := *clone.PaymentDate
        clone.PaymentDate = &t
    }
    if clone.JoinedAt != nil {
        t := *clone.JoinedAt
        clone.JoinedAt = &t
    }
    return &clone
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/domain/scenario/... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/domain/scenario/user_state.go internal/domain/scenario/user_state_test.go
git commit -m "feat: add user scenario state types"
```

---

### Task 1.3: Create Registry and Button Types

**Files:**
- Create: `internal/domain/registry/types.go`
- Create: `internal/domain/button/types.go`
- Test: `internal/domain/registry/types_test.go`

**Step 1: Write the failing test**

Create: `internal/domain/registry/types_test.go`

```go
package registry

import (
    "testing"
    "time"

    "mispilka-bot-tg/internal/domain/scenario"
)

func TestScenarioRegistry_GetDefault(t *testing.T) {
    reg := &ScenarioRegistry{
        DefaultScenarioID: "default",
        Scenarios: map[string]*scenario.Scenario{
            "default": {
                ID:   "default",
                Name: "Default",
            },
        },
    }

    sc, err := reg.GetDefault()
    if err != nil {
        t.Fatalf("Expected no error, got: %v", err)
    }
    if sc.ID != "default" {
        t.Errorf("Expected default scenario, got: %s", sc.ID)
    }
}

func TestScenarioRegistry_SetDefault(t *testing.T) {
    reg := &ScenarioRegistry{
        Scenarios: map[string]*scenario.Scenario{
            "default": {ID: "default", Name: "Default"},
            "premium": {ID: "premium", Name: "Premium"},
        },
    }

    err := reg.SetDefault("premium")
    if err != nil {
        t.Fatalf("Expected no error, got: %v", err)
    }
    if reg.DefaultScenarioID != "premium" {
        t.Errorf("Expected premium as default, got: %s", reg.DefaultScenarioID)
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/domain/registry/... -v`
Expected: FAIL with "undefined: ScenarioRegistry"

**Step 3: Write minimal implementation**

Create: `internal/domain/registry/types.go`

```go
package registry

import (
    "errors"
    "time"

    "mispilka-bot-tg/internal/domain/scenario"
)

var (
    ErrScenarioNotFound       = errors.New("scenario not found")
    ErrCannotDeleteDefault     = errors.New("cannot delete default scenario")
    ErrDefaultScenarioNotFound = errors.New("default scenario not found")
)

// ScenarioRegistry manages all scenarios
type ScenarioRegistry struct {
    Scenarios         map[string]*scenario.Scenario `json:"scenarios"`
    DefaultScenarioID string                        `json:"default_scenario_id"`
}

// NewScenarioRegistry creates a new registry
func NewScenarioRegistry() *ScenarioRegistry {
    return &ScenarioRegistry{
        Scenarios: make(map[string]*scenario.Scenario),
    }
}

// Get retrieves a scenario by ID
func (r *ScenarioRegistry) Get(id string) (*scenario.Scenario, error) {
    sc, ok := r.Scenarios[id]
    if !ok {
        return nil, ErrScenarioNotFound
    }
    return sc, nil
}

// GetDefault retrieves the default scenario
func (r *ScenarioRegistry) GetDefault() (*scenario.Scenario, error) {
    if r.DefaultScenarioID == "" {
        return nil, ErrDefaultScenarioNotFound
    }
    return r.Get(r.DefaultScenarioID)
}

// SetDefault sets the default scenario
func (r *ScenarioRegistry) SetDefault(id string) error {
    if _, ok := r.Scenarios[id]; !ok {
        return ErrScenarioNotFound
    }
    r.DefaultScenarioID = id
    return nil
}

// List returns all scenarios
func (r *ScenarioRegistry) List() []*scenario.Scenario {
    scenarios := make([]*scenario.Scenario, 0, len(r.Scenarios))
    for _, sc := range r.Scenarios {
        scenarios = append(scenarios, sc)
    }
    return scenarios
}

// Add adds a scenario to the registry
func (r *ScenarioRegistry) Add(sc *scenario.Scenario) error {
    if err := sc.Validate(); err != nil {
        return err
    }
    r.Scenarios[sc.ID] = sc
    return nil
}

// Delete removes a scenario from the registry
func (r *ScenarioRegistry) Delete(id string) error {
    if id == r.DefaultScenarioID {
        return ErrCannotDeleteDefault
    }
    if _, ok := r.Scenarios[id]; !ok {
        return ErrScenarioNotFound
    }
    delete(r.Scenarios, id)
    return nil
}
```

Create: `internal/domain/button/types.go`

```go
package button

// ButtonRegistry manages shared button sets
type ButtonRegistry struct {
    ButtonSets map[string]*ButtonSet `json:"button_sets"`
}

// ButtonSet represents a set of buttons
type ButtonSet struct {
    Rows []ButtonRow `json:"rows"`
}

// ButtonRow represents a row of buttons
type ButtonRow struct {
    Buttons []Button `json:"buttons"`
}

// Button represents a single button
type Button struct {
    Type     string `json:"type"`     // url, callback
    Text     string `json:"text"`
    URL      string `json:"url,omitempty"`
    Callback string `json:"callback,omitempty"`
}

// NewButtonRegistry creates a new button registry
func NewButtonRegistry() *ButtonRegistry {
    return &ButtonRegistry{
        ButtonSets: make(map[string]*ButtonSet),
    }
}

// Get retrieves a button set by reference
func (r *ButtonRegistry) Get(ref string) (*ButtonSet, bool) {
    bs, ok := r.ButtonSets[ref]
    return bs, ok
}

// Set stores a button set
func (r *ButtonRegistry) Set(ref string, bs *ButtonSet) {
    r.ButtonSets[ref] = bs
}

// Delete removes a button set
func (r *ButtonRegistry) Delete(ref string) {
    delete(r.ButtonSets, ref)
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/domain/registry/... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/domain/registry/ internal/domain/button/
git commit -m "feat: add registry and button domain types"
```

---

### Task 1.4: Create Broadcast Types

**Files:**
- Create: `internal/domain/broadcast/types.go`
- Test: `internal/domain/broadcast/types_test.go`

**Step 1: Write the failing test**

Create: `internal/domain/broadcast/types_test.go`

```go
package broadcast

import (
    "testing"
)

func TestTargeting_Matches(t *testing.T) {
    tests := []struct {
        name       string
        targeting  *Targeting
        userHas    bool
        expected   bool
    }{
        {
            name:     "no active scenario - user has no active scenario",
            targeting: &Targeting{Conditions: []string{"no_active_scenario"}},
            userHas:  false,
            expected: true,
        },
        {
            name:     "has not paid - user has not paid",
            targeting: &Targeting{Conditions: []string{"has_not_paid"}},
            userHas:  false,
            expected: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test will be implemented with full targeting logic
            // For now, test structure exists
        })
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/domain/broadcast/... -v`
Expected: FAIL with "undefined: Targeting"

**Step 3: Write minimal implementation**

Create: `internal/domain/broadcast/types.go`

```go
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
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/domain/broadcast/... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/domain/broadcast/
git commit -m "feat: add broadcast domain types"
```

---

## Phase 2: Storage Layer

### Task 2.1: Create Scenario Registry Storage

**Files:**
- Create: `internal/services/scenario/registry.go`
- Test: `internal/services/scenario/registry_test.go`

**Step 1: Write the failing test**

Update: `internal/services/scenario/registry_test.go`

```go
package scenario

import (
    "os"
    "path/filepath"
    "testing"
    "time"

    "mispilka-bot-tg/internal/domain/scenario"
)

func TestScenarioRegistry_LoadSave(t *testing.T) {
    tmpDir := t.TempDir()
    registryPath := filepath.Join(tmpDir, "registry.json")

    // Create initial registry
    reg := NewRegistry(registryPath)
    reg.Scenarios = map[string]*scenario.Scenario{
        "test": {
            ID:        "test",
            Name:      "Test",
            CreatedAt: time.Now(),
            IsActive:  true,
            Config: scenario.ScenarioConfig{
                Prodamus: scenario.ProdamusConfig{
                    ProductName:    "Test Product",
                    ProductPrice:   "1000",
                    PaidContent:    "Thank you!",
                    PrivateGroupID: "-1001234567890",
                },
            },
        },
    }
    reg.DefaultScenarioID = "test"

    // Save
    err := reg.Save()
    if err != nil {
        t.Fatalf("Failed to save: %v", err)
    }

    // Load into new registry
    reg2 := NewRegistry(registryPath)
    err = reg2.Load()
    if err != nil {
        t.Fatalf("Failed to load: %v", err)
    }

    // Verify
    if reg2.DefaultScenarioID != "test" {
        t.Errorf("Expected default scenario 'test', got '%s'", reg2.DefaultScenarioID)
    }
    sc, ok := reg2.Scenarios["test"]
    if !ok {
        t.Fatal("Scenario 'test' not found")
    }
    if sc.Name != "Test" {
        t.Errorf("Expected name 'Test', got '%s'", sc.Name)
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/services/scenario/... -v`
Expected: FAIL with "undefined: NewRegistry"

**Step 3: Write minimal implementation**

Create: `internal/services/scenario/registry.go`

```go
package scenario

import (
    "encoding/json"
    "errors"
    "os"
    "path/filepath"
    "sync"

    "mispilka-bot-tg/internal/domain/scenario"
)

var (
    ErrRegistryLoadFailed = errors.New("failed to load registry")
    ErrRegistrySaveFailed = errors.New("failed to save registry")
)

// Registry manages scenario persistence
type Registry struct {
    filePath string
    mu       sync.RWMutex

    Scenarios         map[string]*scenario.Scenario `json:"scenarios"`
    DefaultScenarioID string                        `json:"default_scenario_id"`
}

// NewRegistry creates a new registry
func NewRegistry(filePath string) *Registry {
    return &Registry{
        filePath: filePath,
        Scenarios: make(map[string]*scenario.Scenario),
    }
}

// Load loads the registry from disk
func (r *Registry) Load() error {
    r.mu.Lock()
    defer r.mu.Unlock()

    data, err := os.ReadFile(r.filePath)
    if err != nil {
        if errors.Is(err, os.ErrNotExist) {
            // Create new registry
            return nil
        }
        return ErrRegistryLoadFailed
    }

    if err := json.Unmarshal(data, r); err != nil {
        return ErrRegistryLoadFailed
    }

    return nil
}

// Save saves the registry to disk
func (r *Registry) Save() error {
    r.mu.RLock()
    defer r.mu.RUnlock()

    // Ensure directory exists
    if err := os.MkdirAll(filepath.Dir(r.filePath), 0755); err != nil {
        return ErrRegistrySaveFailed
    }

    data, err := json.MarshalIndent(r, "", "  ")
    if err != nil {
        return ErrRegistrySaveFailed
    }

    if err := os.WriteFile(r.filePath, data, 0644); err != nil {
        return ErrRegistrySaveFailed
    }

    return nil
}

// Get retrieves a scenario by ID
func (r *Registry) Get(id string) (*scenario.Scenario, error) {
    r.mu.RLock()
    defer r.mu.RUnlock()

    sc, ok := r.Scenarios[id]
    if !ok {
        return nil, scenario.ErrScenarioNotFound
    }
    return sc, nil
}

// GetDefault retrieves the default scenario
func (r *Registry) GetDefault() (*scenario.Scenario, error) {
    r.mu.RLock()
    defer r.mu.RUnlock()

    if r.DefaultScenarioID == "" {
        return nil, errors.New("no default scenario set")
    }
    sc, ok := r.Scenarios[r.DefaultScenarioID]
    if !ok {
        return nil, scenario.ErrScenarioNotFound
    }
    return sc, nil
}

// SetDefault sets the default scenario
func (r *Registry) SetDefault(id string) error {
    r.mu.Lock()
    defer r.mu.Unlock()

    if _, ok := r.Scenarios[id]; !ok {
        return scenario.ErrScenarioNotFound
    }
    r.DefaultScenarioID = id
    return r.Save()
}

// Add adds a scenario to the registry
func (r *Registry) Add(sc *scenario.Scenario) error {
    r.mu.Lock()
    defer r.mu.Unlock()

    if err := sc.Validate(); err != nil {
        return err
    }
    r.Scenarios[sc.ID] = sc
    return r.Save()
}

// List returns all scenarios
func (r *Registry) List() []*scenario.Scenario {
    r.mu.RLock()
    defer r.mu.RUnlock()

    scenarios := make([]*scenario.Scenario, 0, len(r.Scenarios))
    for _, sc := range r.Scenarios {
        scenarios = append(scenarios, sc)
    }
    return scenarios
}

// Delete removes a scenario from the registry
func (r *Registry) Delete(id string) error {
    r.mu.Lock()
    defer r.mu.Unlock()

    if id == r.DefaultScenarioID {
        return errors.New("cannot delete default scenario")
    }
    if _, ok := r.Scenarios[id]; !ok {
        return scenario.ErrScenarioNotFound
    }
    delete(r.Scenarios, id)
    return r.Save()
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/services/scenario/... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/services/scenario/registry.go internal/services/scenario/registry_test.go
git commit -m "feat: add scenario registry storage"
```

---

### Task 2.2: Create Scenario Config Storage

**Files:**
- Create: `internal/services/scenario/config.go`
- Test: `internal/services/scenario/config_test.go`

**Step 1: Write the failing test**

Create: `internal/services/scenario/config_test.go`

```go
package scenario

import (
    "os"
    "path/filepath"
    "testing"
)

func TestConfig_LoadSave(t *testing.T) {
    tmpDir := t.TempDir()
    scenarioDir := filepath.Join(tmpDir, "scenarios", "test")
    configPath := filepath.Join(scenarioDir, "config.json")

    // Create config
    cfg := NewConfig(configPath)
    cfg.ID = "test"
    cfg.Name = "Test Scenario"
    cfg.Prodamus.ProductName = "Test Product"
    cfg.Prodamus.ProductPrice = "1000"
    cfg.Prodamus.PaidContent = "Thank you!"
    cfg.Prodamus.PrivateGroupID = "-1001234567890"

    // Save
    err := cfg.Save()
    if err != nil {
        t.Fatalf("Failed to save: %v", err)
    }

    // Load into new config
    cfg2 := NewConfig(configPath)
    err = cfg2.Load()
    if err != nil {
        t.Fatalf("Failed to load: %v", err)
    }

    // Verify
    if cfg2.ID != "test" {
        t.Errorf("Expected ID 'test', got '%s'", cfg2.ID)
    }
    if cfg2.Name != "Test Scenario" {
        t.Errorf("Expected name 'Test Scenario', got '%s'", cfg2.Name)
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/services/scenario/... -v`
Expected: FAIL with "undefined: NewConfig"

**Step 3: Write minimal implementation**

Create: `internal/services/scenario/config.go`

```go
package scenario

import (
    "encoding/json"
    "errors"
    "os"
    "path/filepath"
    "sync"

    "mispilka-bot-tg/internal/domain/scenario"
)

var (
    ErrConfigLoadFailed = errors.New("failed to load config")
    ErrConfigSaveFailed = errors.New("failed to save config")
)

// Config manages scenario configuration persistence
type Config struct {
    filePath string
    mu       sync.RWMutex

    ID        string          `json:"id"`
    Name      string          `json:"name"`
    CreatedAt string          `json:"created_at"` // ISO 8601
    Prodamus  ProdamusConfig  `json:"prodamus"`
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
    c.mu.RLock()
    defer c.mu.RUnlock()

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
    return &scenario.Scenario{
        ID:       c.ID,
        Name:     c.Name,
        IsActive: true,
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
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/services/scenario/... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/services/scenario/config.go internal/services/scenario/config_test.go
git commit -m "feat: add scenario config storage"
```

---

### Task 2.3: Create Scenario Messages Storage

**Files:**
- Create: `internal/services/scenario/messages.go`
- Test: `internal/services/scenario/messages_test.go`

**Step 1: Write the failing test**

Create: `internal/services/scenario/messages_test.go`

```go
package scenario

import (
    "os"
    "path/filepath"
    "testing"
)

func TestScenarioMessages_LoadSave(t *testing.T) {
    tmpDir := t.TempDir()
    scenarioDir := filepath.Join(tmpDir, "scenarios", "test")
    messagesPath := filepath.Join(scenarioDir, "messages.json")

    // Create messages
    msgs := NewScenarioMessages(messagesPath)
    msgs.MessagesList = []string{"msg_1", "msg_2"}
    msgs.Messages = map[string]MessageData{
        "msg_1": {
            Timing: scenario.Timing{Hours: 0, Minutes: 0},
            TemplateFile: "msg_1.md",
        },
        "msg_2": {
            Timing: scenario.Timing{Hours: 1, Minutes: 0},
            TemplateFile: "msg_2.md",
        },
    }

    // Save
    err := msgs.Save()
    if err != nil {
        t.Fatalf("Failed to save: %v", err)
    }

    // Load into new messages
    msgs2 := NewScenarioMessages(messagesPath)
    err = msgs2.Load()
    if err != nil {
        t.Fatalf("Failed to load: %v", err)
    }

    // Verify
    if len(msgs2.MessagesList) != 2 {
        t.Errorf("Expected 2 messages, got %d", len(msgs2.MessagesList))
    }
    if msgs2.MessagesList[0] != "msg_1" {
        t.Errorf("Expected msg_1, got %s", msgs2.MessagesList[0])
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/services/scenario/... -v`
Expected: FAIL with "undefined: NewScenarioMessages"

**Step 3: Write minimal implementation**

Create: `internal/services/scenario/messages.go`

```go
package scenario

import (
    "encoding/json"
    "errors"
    "os"
    "path/filepath"
    "sync"

    "mispilka-bot-tg/internal/domain/scenario"
)

var (
    ErrMessagesLoadFailed = errors.New("failed to load messages")
    ErrMessagesSaveFailed = errors.New("failed to save messages")
)

// MessageData defines a message in the flow
type MessageData struct {
    Timing         scenario.Timing                `json:"timing"`
    TemplateFile   string                        `json:"template_file,omitempty"`
    Photos         []string                      `json:"photos,omitempty"`
    InlineKeyboard *InlineKeyboardConfig         `json:"inline_keyboard,omitempty"`
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
    Type     string `json:"type"`
    Text     string `json:"text"`
    URL      string `json:"url,omitempty"`
    Callback string `json:"callback,omitempty"`
}

// ScenarioMessages manages scenario messages persistence
type ScenarioMessages struct {
    filePath string
    mu       sync.RWMutex

    MessagesList []string              `json:"messages_list"`
    Messages     map[string]MessageData `json:"messages"`
}

// NewScenarioMessages creates a new scenario messages
func NewScenarioMessages(filePath string) *ScenarioMessages {
    return &ScenarioMessages{
        filePath:    filePath,
        Messages:    make(map[string]MessageData),
        MessagesList: make([]string, 0),
    }
}

// Load loads the messages from disk
func (m *ScenarioMessages) Load() error {
    m.mu.Lock()
    defer m.mu.Unlock()

    data, err := os.ReadFile(m.filePath)
    if err != nil {
        if errors.Is(err, os.ErrNotExist) {
            return scenario.ErrMessageNotFound // Reuse domain error
        }
        return ErrMessagesLoadFailed
    }

    if err := json.Unmarshal(data, m); err != nil {
        return ErrMessagesLoadFailed
    }

    return nil
}

// Save saves the messages to disk
func (m *ScenarioMessages) Save() error {
    m.mu.RLock()
    defer m.mu.RUnlock()

    // Ensure directory exists
    if err := os.MkdirAll(filepath.Dir(m.filePath), 0755); err != nil {
        return ErrMessagesSaveFailed
    }

    data, err := json.MarshalIndent(m, "", "  ")
    if err != nil {
        return ErrMessagesSaveFailed
    }

    if err := os.WriteFile(m.filePath, data, 0644); err != nil {
        return ErrMessagesSaveFailed
    }

    return nil
}

// ToDomain converts to domain scenario messages
func (m *ScenarioMessages) ToDomain() scenario.ScenarioMessages {
    msgs := scenario.ScenarioMessages{
        MessagesList: m.MessagesList,
        Messages:     make(map[string]scenario.MessageData),
    }
    for id, md := range m.Messages {
        msgs.Messages[id] = scenario.MessageData{
            Timing:       md.Timing,
            TemplateFile: md.TemplateFile,
            Photos:       md.Photos,
            InlineKeyboard: convertInlineKeyboard(md.InlineKeyboard),
        }
    }
    return msgs
}

func convertInlineKeyboard(ik *InlineKeyboardConfig) *scenario.InlineKeyboardConfig {
    if ik == nil {
        return nil
    }
    domainIK := &scenario.InlineKeyboardConfig{
        Rows: make([]scenario.InlineKeyboardRowConfig, len(ik.Rows)),
    }
    for i, row := range ik.Rows {
        domainIK.Rows[i] = scenario.InlineKeyboardRowConfig{
            Buttons: make([]scenario.InlineKeyboardButtonConfig, len(row.Buttons)),
        }
        for j, btn := range row.Buttons {
            domainIK.Rows[i].Buttons[j] = scenario.InlineKeyboardButtonConfig{
                Type:     btn.Type,
                Text:     btn.Text,
                URL:      btn.URL,
                Callback: btn.Callback,
            }
        }
    }
    return domainIK
}
```

**Step 4: Update domain types to add ErrMessageNotFound**

Update: `internal/domain/scenario/status.go`

Add to the error variables:
```go
ErrMessageNotFound = errors.New("message not found")
```

**Step 5: Run test to verify it passes**

Run: `go test ./internal/services/scenario/... -v`
Expected: PASS

**Step 6: Commit**

```bash
git add internal/services/scenario/messages.go internal/services/scenario/messages_test.go internal/domain/scenario/status.go
git commit -m "feat: add scenario messages storage"
```

---

### Task 2.4: Create User Storage with Scenario Support

**Files:**
- Modify: `internal/services/users.go`
- Modify: `internal/services/storage.go`

**Step 1: Update user structure**

Update: `internal/services/users.go`

Read the current file first to see the exact structure, then modify:

```go
// User represents a user in the system
type User struct {
    UserName            string                         `json:"user_name"`
    RegTime             time.Time                      `json:"reg_time"`
    IsMessaging         bool                           `json:"is_messaging"`
    Scenarios           map[string]*UserScenarioState  `json:"scenarios,omitempty"`
    ActiveScenarioID    string                         `json:"active_scenario_id,omitempty"`
    // Legacy fields for migration compatibility
    MessagesList        []string                       `json:"messages_list,omitempty"`
    PaymentDate         *time.Time                     `json:"payment_date,omitempty"`
    PaymentLink         string                         `json:"payment_link,omitempty"`
    InviteLink          string                         `json:"invite_link,omitempty"`
    JoinedGroup         bool                           `json:"joined_group,omitempty"`
    JoinedAt            *time.Time                     `json:"joined_at,omitempty"`
    PaymentInfo         *models.WebhookPayload         `json:"payment_info,omitempty"`
}

// UserScenarioState tracks user's progress in a scenario
type UserScenarioState struct {
    Status              ScenarioStatus   `json:"status"`
    CurrentMessageIndex int              `json:"current_message_index"`
    LastSentAt          *time.Time       `json:"last_sent_at,omitempty"`
    CompletedAt         *time.Time       `json:"completed_at,omitempty"`
    PaymentDate         *time.Time       `json:"payment_date,omitempty"`
    PaymentLink         string           `json:"payment_link,omitempty"`
    InviteLink          string           `json:"invite_link,omitempty"`
    JoinedGroup         bool             `json:"joined_group,omitempty"`
    JoinedAt            *time.Time       `json:"joined_at,omitempty"`
}

// ScenarioStatus represents user's status in a scenario
type ScenarioStatus string

const (
    StatusNotStarted ScenarioStatus = "not_started"
    StatusActive     ScenarioStatus = "active"
    StatusCompleted  ScenarioStatus = "completed"
    StatusStopped    ScenarioStatus = "stopped"
)
```

**Step 2: Add scenario-related methods to users.go**

Add the following methods to the UserService:

```go
// GetUserScenario retrieves user's state for a specific scenario
func (s *UserService) GetUserScenario(chatID, scenarioID string) (*UserScenarioState, error) {
    user, err := s.GetUser(chatID)
    if err != nil {
        return nil, err
    }

    if user.Scenarios == nil {
        // Initialize with not_started state
        return &UserScenarioState{Status: StatusNotStarted}, nil
    }

    state, ok := user.Scenarios[scenarioID]
    if !ok {
        return &UserScenarioState{Status: StatusNotStarted}, nil
    }

    return state, nil
}

// SetUserScenario sets user's state for a specific scenario
func (s *UserService) SetUserScenario(chatID, scenarioID string, state *UserScenarioState) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    user, ok := s.users[chatID]
    if !ok {
        return ErrUserNotFound
    }

    if user.Scenarios == nil {
        user.Scenarios = make(map[string]*UserScenarioState)
    }

    user.Scenarios[scenarioID] = state

    return s.saveData()
}

// GetUserActiveScenario retrieves user's active scenario
func (s *UserService) GetUserActiveScenario(chatID string) (string, *UserScenarioState, error) {
    user, err := s.GetUser(chatID)
    if err != nil {
        return "", nil, err
    }

    if user.ActiveScenarioID == "" {
        return "", nil, nil // No active scenario
    }

    state, err := s.GetUserScenario(chatID, user.ActiveScenarioID)
    if err != nil {
        return "", nil, err
    }

    return user.ActiveScenarioID, state, nil
}

// SetUserActiveScenario sets user's active scenario
func (s *UserService) SetUserActiveScenario(chatID, scenarioID string) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    user, ok := s.users[chatID]
    if !ok {
        return ErrUserNotFound
    }

    user.ActiveScenarioID = scenarioID

    return s.saveData()
}

// IsScenarioCompleted checks if user has completed a scenario
func (s *UserService) IsScenarioCompleted(chatID, scenarioID string) (bool, error) {
    state, err := s.GetUserScenario(chatID, scenarioID)
    if err != nil {
        return false, err
    }
    return state.Status == StatusCompleted, nil
}
```

**Step 3: Update storage utilities for scenarios**

Update: `internal/services/storage.go`

The storage utilities already support generic JSON read/write. No changes needed.

**Step 4: Build to verify changes**

Run: `go build ./...`
Expected: Success

**Step 5: Run existing tests**

Run: `go test ./internal/services/... -v`
Expected: All existing tests pass (we didn't break anything)

**Step 6: Commit**

```bash
git add internal/services/users.go
git commit -m "feat: add scenario support to user storage"
```

---

## Phase 3: Template System

### Task 3.1: Create Template Variable Types

**Files:**
- Create: `internal/services/template/types.go`
- Create: `internal/services/template/variables.go`
- Test: `internal/services/template/types_test.go`

**Step 1: Write the failing test**

Create: `internal/services/template/types_test.go`

```go
package template

import (
    "testing"
    "time"

    "mispilka-bot-tg/internal/domain/scenario"
)

func TestVariableScope_String(t *testing.T) {
    tests := []struct {
        scope    VariableScope
        expected string
    }{
        {ScopeBot, "bot"},
        {ScopeScenario, "scenario"},
        {ScopeUser, "user"},
    }

    for _, tt := range tests {
        if tt.scope.String() != tt.expected {
            t.Errorf("Expected %s, got %s", tt.expected, tt.scope.String())
        }
    }
}

func TestTemplateContext_GetVariable(t *testing.T) {
    ctx := &TemplateContext{
        BotVars: map[string]string{
            "bot_name": "TestBot",
        },
        ScenarioVars: map[string]string{
            "product_name": "Test Product",
        },
        UserVars: map[string]string{
            "user_name": "John",
        },
    }

    // Test bot scope
    val, ok := ctx.GetVariable(ScopeBot, "bot_name")
    if !ok || val != "TestBot" {
        t.Error("Failed to get bot variable")
    }

    // Test scenario scope
    val, ok = ctx.GetVariable(ScopeScenario, "product_name")
    if !ok || val != "Test Product" {
        t.Error("Failed to get scenario variable")
    }

    // Test user scope
    val, ok = ctx.GetVariable(ScopeUser, "user_name")
    if !ok || val != "John" {
        t.Error("Failed to get user variable")
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/services/template/... -v`
Expected: FAIL with "undefined: VariableScope"

**Step 3: Write minimal implementation**

Create: `internal/services/template/types.go`

```go
package template

import "time"

// VariableScope represents the scope of a template variable
type VariableScope string

const (
    ScopeBot      VariableScope = "bot"
    ScopeScenario VariableScope = "scenario"
    ScopeUser     VariableScope = "user"
)

// String returns the string representation
func (s VariableScope) String() string {
    return string(s)
}

// IsValid checks if the scope is valid
func (s VariableScope) IsValid() bool {
    switch s {
    case ScopeBot, ScopeScenario, ScopeUser:
        return true
    }
    return false
}

// TemplateContext contains all variables for template rendering
type TemplateContext struct {
    BotVars      map[string]string
    ScenarioVars map[string]string
    UserVars     map[string]string
}

// NewTemplateContext creates a new template context
func NewTemplateContext() *TemplateContext {
    return &TemplateContext{
        BotVars:      make(map[string]string),
        ScenarioVars: make(map[string]string),
        UserVars:     make(map[string]string),
    }
}

// GetVariable retrieves a variable from the specified scope
func (c *TemplateContext) GetVariable(scope VariableScope, key string) (string, bool) {
    var vars map[string]string
    switch scope {
    case ScopeBot:
        vars = c.BotVars
    case ScopeScenario:
        vars = c.ScenarioVars
    case ScopeUser:
        vars = c.UserVars
    default:
        return "", false
    }

    val, ok := vars[key]
    return val, ok
}

// SetVariable sets a variable in the specified scope
func (c *TemplateContext) SetVariable(scope VariableScope, key, value string) {
    switch scope {
    case ScopeBot:
        c.BotVars[key] = value
    case ScopeScenario:
        c.ScenarioVars[key] = value
    case ScopeUser:
        c.UserVars[key] = value
    }
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/services/template/... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/services/template/types.go internal/services/template/types_test.go
git commit -m "feat: add template variable types"
```

---

### Task 3.2: Create Template Renderer

**Files:**
- Create: `internal/services/template/renderer.go`
- Test: `internal/services/template/renderer_test.go`

**Step 1: Write the failing test**

Create: `internal/services/template/renderer_test.go`

```go
package template

import (
    "testing"
)

func TestRenderer_Render(t *testing.T) {
    renderer := NewRenderer()

    ctx := NewTemplateContext()
    ctx.SetVariable(ScopeBot, "bot_name", "TestBot")
    ctx.SetVariable(ScopeScenario, "product_name", "Premium Course")
    ctx.SetVariable(ScopeUser, "user_name", "John")

    template := "Hello {{user.user_name}}! Welcome to {{scenario.product_name}} by {{bot.bot_name}}"

    result, err := renderer.Render(template, ctx)
    if err != nil {
        t.Fatalf("Failed to render: %v", err)
    }

    expected := "Hello John! Welcome to Premium Course by TestBot"
    if result != expected {
        t.Errorf("Expected '%s', got '%s'", expected, result)
    }
}

func TestRenderer_RenderWithMissingVariable(t *testing.T) {
    renderer := NewRenderer()

    ctx := NewTemplateContext()
    // Don't set any variables

    template := "Hello {{user.name}}!"

    result, err := renderer.Render(template, ctx)
    if err != nil {
        t.Fatalf("Failed to render: %v", err)
    }

    // Missing variables should be replaced with empty string
    expected := "Hello !"
    if result != expected {
        t.Errorf("Expected '%s', got '%s'", expected, result)
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/services/template/... -v`
Expected: FAIL with "undefined: NewRenderer"

**Step 3: Write minimal implementation**

Create: `internal/services/template/renderer.go`

```go
package template

import (
    "regexp"
    "strings"
)

var (
    templateVarRegex = regexp.MustCompile(`\{\{(\w+)\.(\w+)\}\}`)
)

// Renderer renders templates with variable substitution
type Renderer struct{}

// NewRenderer creates a new renderer
func NewRenderer() *Renderer {
    return &Renderer{}
}

// Render renders a template with the given context
func (r *Renderer) Render(template string, ctx *TemplateContext) (string, error) {
    result := template

    matches := templateVarRegex.FindAllStringSubmatch(template, -1)
    for _, match := range matches {
        if len(match) < 3 {
            continue
        }

        fullMatch := match[0]
        scopeStr := match[1]
        key := match[2]

        scope := VariableScope(scopeStr)
        if !scope.IsValid() {
            // Invalid scope, leave as is
            continue
        }

        value, ok := ctx.GetVariable(scope, key)
        if ok {
            result = strings.ReplaceAll(result, fullMatch, value)
        } else {
            // Missing variable, replace with empty string
            result = strings.ReplaceAll(result, fullMatch, "")
        }
    }

    return result, nil
}

// RenderForDemo renders a template with highlighted placeholders
func (r *Renderer) RenderForDemo(template string) string {
    result := template

    matches := templateVarRegex.FindAllStringSubmatch(template, -1)
    for _, match := range matches {
        if len(match) < 3 {
            continue
        }

        fullMatch := match[0]
        // Replace {{ with __ and }} with __ for highlighting
        highlighted := strings.ReplaceAll(fullMatch, "{{", "__")
        highlighted = strings.ReplaceAll(highlighted, "}}", "__")
        result = strings.ReplaceAll(result, fullMatch, highlighted)
    }

    return result
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/services/template/... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/services/template/renderer.go internal/services/template/renderer_test.go
git commit -m "feat: add template renderer"
```

---

### Task 3.3: Create Template Variable Resolver

**Files:**
- Create: `internal/services/template/resolver.go`
- Test: `internal/services/template/resolver_test.go`

**Step 1: Write the failing test**

Create: `internal/services/template/resolver_test.go`

```go
package template

import (
    "testing"
    "time"

    "mispilka-bot-tg/internal/domain/scenario"
    "mispilka-bot-tg/internal/services/scenario"
)

func TestResolver_ResolveForUser(t *testing.T) {
    resolver := NewResolver("data/templates/bot_globals.json")

    sc := &scenario.Scenario{
        ID:   "test",
        Name: "Test Scenario",
        Config: scenario.ScenarioConfig{
            Prodamus: scenario.ProdamusConfig{
                ProductName:    "Test Product",
                ProductPrice:   "1000",
                PrivateGroupID: "-1001234567890",
            },
        },
    }

    userState := &UserScenarioState{
        Status:              StatusActive,
        CurrentMessageIndex: 1,
        PaymentLink:         "https://pay.example.com/123",
        InviteLink:          "https://t.me/+abc",
    }

    user := &User{
        UserName: "john_doe",
        RegTime:  time.Now(),
    }

    ctx, err := resolver.ResolveForUser(sc, userState, user)
    if err != nil {
        t.Fatalf("Failed to resolve: %v", err)
    }

    // Check bot variables
    if ctx.BotVars == nil {
        t.Error("Bot vars should not be nil")
    }

    // Check scenario variables
    if ctx.ScenarioVars["id"] != "test" {
        t.Errorf("Expected scenario.id 'test', got '%s'", ctx.ScenarioVars["id"])
    }
    if ctx.ScenarioVars["product_name"] != "Test Product" {
        t.Errorf("Expected product_name 'Test Product', got '%s'", ctx.ScenarioVars["product_name"])
    }

    // Check user variables
    if ctx.UserVars["user_name"] != "john_doe" {
        t.Errorf("Expected user_name 'john_doe', got '%s'", ctx.UserVars["user_name"])
    }
    if ctx.UserVars["payment_link"] != "https://pay.example.com/123" {
        t.Errorf("Expected payment_link, got '%s'", ctx.UserVars["payment_link"])
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/services/template/... -v`
Expected: FAIL with "undefined: NewResolver"

**Step 3: Write minimal implementation**

Create: `internal/services/template/resolver.go`

```go
package template

import (
    "encoding/json"
    "errors"
    "os"
    "sync"
    "time"
)

var (
    ErrFailedToLoadGlobals = errors.New("failed to load bot globals")
)

// BotGlobals contains bot-level template variables
type BotGlobals struct {
    Variables map[string]string `json:"variables"`
}

// Resolver resolves template variables for rendering
type Resolver struct {
    botGlobalsPath string
    botGlobals     *BotGlobals
    mu             sync.RWMutex
}

// NewResolver creates a new resolver
func NewResolver(botGlobalsPath string) *Resolver {
    return &Resolver{
        botGlobalsPath: botGlobalsPath,
    }
}

// loadBotGlobals loads bot globals from disk
func (r *Resolver) loadBotGlobals() error {
    r.mu.Lock()
    defer r.mu.Unlock()

    data, err := os.ReadFile(r.botGlobalsPath)
    if err != nil {
        if errors.Is(err, os.ErrNotExist) {
            // File doesn't exist, use empty globals
            r.botGlobals = &BotGlobals{Variables: make(map[string]string)}
            return nil
        }
        return ErrFailedToLoadGlobals
    }

    var globals BotGlobals
    if err := json.Unmarshal(data, &globals); err != nil {
        return ErrFailedToLoadGlobals
    }

    r.botGlobals = &globals
    return nil
}

// ResolveForUser resolves all template variables for a user
func (r *Resolver) ResolveForUser(sc *scenario.Scenario, userState *UserScenarioState, user *User) (*TemplateContext, error) {
    // Load bot globals if not loaded
    if r.botGlobals == nil {
        if err := r.loadBotGlobals(); err != nil {
            return nil, err
        }
    }

    ctx := NewTemplateContext()

    // Bot variables
    r.mu.RLock()
    for k, v := range r.botGlobals.Variables {
        ctx.SetVariable(ScopeBot, k, v)
    }
    r.mu.RUnlock()

    // Scenario variables
    ctx.SetVariable(ScopeScenario, "id", sc.ID)
    ctx.SetVariable(ScopeScenario, "name", sc.Name)
    ctx.SetVariable(ScopeScenario, "product_name", sc.Config.Prodamus.ProductName)
    ctx.SetVariable(ScopeScenario, "product_price", sc.Config.Prodamus.ProductPrice)
    ctx.SetVariable(ScopeScenario, "private_group_id", sc.Config.Prodamus.PrivateGroupID)

    // User variables
    if user != nil {
        ctx.SetVariable(ScopeUser, "user_name", user.UserName)
        ctx.SetVariable(ScopeUser, "reg_date", user.RegTime.Format("02.01.2006"))
    }

    if userState != nil {
        ctx.SetVariable(ScopeUser, "payment_link", userState.PaymentLink)
        ctx.SetVariable(ScopeUser, "invite_link", userState.InviteLink)

        if userState.PaymentDate != nil {
            ctx.SetVariable(ScopeUser, "payment_date", userState.PaymentDate.Format("02.01.2006"))
        }
        if userState.JoinedAt != nil {
            ctx.SetVariable(ScopeUser, "joined_date", userState.JoinedAt.Format("02.01.2006"))
        }
    }

    return ctx, nil
}

// ResolveForDemo resolves template variables for demo mode
func (r *Resolver) ResolveForDemo(sc *scenario.Scenario) (*TemplateContext, error) {
    ctx := NewTemplateContext()

    // Bot variables
    r.mu.RLock()
    if r.botGlobals != nil {
        for k, v := range r.botGlobals.Variables {
            ctx.SetVariable(ScopeBot, k, v)
        }
    }
    r.mu.RUnlock()

    // Scenario variables
    ctx.SetVariable(ScopeScenario, "id", sc.ID)
    ctx.SetVariable(ScopeScenario, "name", sc.Name)
    ctx.SetVariable(ScopeScenario, "product_name", sc.Config.Prodamus.ProductName)
    ctx.SetVariable(ScopeScenario, "product_price", sc.Config.Prodamus.ProductPrice)

    return ctx, nil
}
```

Note: The test uses `User` type - we need to add an import for it or use the actual type from users.go.

**Step 4: Run test to verify it passes**

First, we need to fix the imports in the test:

```go
import (
    // ...
    "mispilka-bot-tg/internal/services"
)
```

And update the test to use the correct User type:

```go
user := &services.User{
    UserName: "john_doe",
    RegTime:  time.Now(),
}
```

Run: `go test ./internal/services/template/... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/services/template/resolver.go internal/services/template/resolver_test.go
git commit -m "feat: add template variable resolver"
```

---

## Phase 4: Scenario Services

### Task 4.1: Create Scenario Service

**Files:**
- Create: `internal/services/scenario/service.go`
- Test: `internal/services/scenario/service_test.go`

**Step 1: Write the failing test**

Create: `internal/services/scenario/service_test.go`

```go
package scenario

import (
    "os"
    "path/filepath"
    "testing"
    "time"

    "mispilka-bot-tg/internal/domain/scenario"
)

func TestScenarioService_CreateScenario(t *testing.T) {
    tmpDir := t.TempDir()
    scenariosDir := filepath.Join(tmpDir, "scenarios")
    service := NewService(scenariosDir)

    // Create scenario
    sc, err := service.CreateScenario("Test Scenario", &ProdamusConfig{
        ProductName:    "Test Product",
        ProductPrice:   "1000",
        PaidContent:    "Thank you!",
        PrivateGroupID: "-1001234567890",
    })
    if err != nil {
        t.Fatalf("Failed to create scenario: %v", err)
    }

    if sc.ID == "" {
        t.Error("Expected scenario ID to be set")
    }
    if sc.Name != "Test Scenario" {
        t.Errorf("Expected name 'Test Scenario', got '%s'", sc.Name)
    }

    // Verify scenario was saved
    loaded, err := service.GetScenario(sc.ID)
    if err != nil {
        t.Fatalf("Failed to load scenario: %v", err)
    }
    if loaded.ID != sc.ID {
        t.Errorf("Expected ID %s, got %s", sc.ID, loaded.ID)
    }
}

func TestScenarioService_ListScenarios(t *testing.T) {
    tmpDir := t.TempDir()
    scenariosDir := filepath.Join(tmpDir, "scenarios")
    service := NewService(scenariosDir)

    // Create two scenarios
    _, _ = service.CreateScenario("Scenario 1", &ProdamusConfig{
        ProductName:    "Product 1",
        ProductPrice:   "1000",
        PaidContent:    "Content 1",
        PrivateGroupID: "-1001234567890",
    })
    _, _ = service.CreateScenario("Scenario 2", &ProdamusConfig{
        ProductName:    "Product 2",
        ProductPrice:   "2000",
        PaidContent:    "Content 2",
        PrivateGroupID: "-1001234567890",
    })

    scenarios := service.ListScenarios()
    if len(scenarios) != 2 {
        t.Errorf("Expected 2 scenarios, got %d", len(scenarios))
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/services/scenario/... -v`
Expected: FAIL with "undefined: NewService"

**Step 3: Write minimal implementation**

Create: `internal/services/scenario/service.go`

```go
package scenario

import (
    "fmt"
    "os"
    "path/filepath"
    "strings"
    "time"
)

// Service provides high-level scenario operations
type Service struct {
    scenariosDir string
    registry     *Registry
}

// NewService creates a new scenario service
func NewService(scenariosDir string) *Service {
    registryPath := filepath.Join(scenariosDir, "registry.json")
    return &Service{
        scenariosDir: scenariosDir,
        registry:     NewRegistry(registryPath),
    }
}

// Initialize initializes the service (loads registry)
func (s *Service) Initialize() error {
    return s.registry.Load()
}

// CreateScenario creates a new scenario
func (s *Service) CreateScenario(name string, prodamus *ProdamusConfig) (*ScenarioInfo, error) {
    // Generate scenario ID from name
    id := generateScenarioID(name)

    // Create scenario directory
    scenarioDir := filepath.Join(s.scenariosDir, id)
    if err := os.MkdirAll(scenarioDir, 0755); err != nil {
        return nil, fmt.Errorf("failed to create scenario directory: %w", err)
    }

    // Create config
    configPath := filepath.Join(scenarioDir, "config.json")
    cfg := NewConfig(configPath)
    cfg.ID = id
    cfg.Name = name
    cfg.CreatedAt = time.Now().Format(time.RFC3339)
    cfg.Prodamus = *prodamus
    if err := cfg.Save(); err != nil {
        return nil, err
    }

    // Create empty messages
    messagesPath := filepath.Join(scenarioDir, "messages.json")
    msgs := NewScenarioMessages(messagesPath)
    if err := msgs.Save(); err != nil {
        return nil, err
    }

    // Create empty summary
    summaryPath := filepath.Join(scenarioDir, "summary.json")
    summary := NewScenarioSummary(summaryPath)
    if err := summary.Save(); err != nil {
        return nil, err
    }

    // Add to registry
    domainSc := &scenario.Scenario{
        ID:       id,
        Name:     name,
        CreatedAt: time.Now(),
        IsActive: true,
        Config: scenario.ScenarioConfig{
            Prodamus: scenario.ProdamusConfig{
                ProductName:    prodamus.ProductName,
                ProductPrice:   prodamus.ProductPrice,
                PaidContent:    prodamus.PaidContent,
                PrivateGroupID: prodamus.PrivateGroupID,
            },
        },
    }
    if err := s.registry.Add(domainSc); err != nil {
        return nil, err
    }

    // Set as default if first scenario
    scenarios := s.registry.List()
    if len(scenarios) == 1 {
        _ = s.registry.SetDefault(id)
    }

    return &ScenarioInfo{
        ID:   id,
        Name: name,
    }, nil
}

// GetScenario retrieves a scenario by ID
func (s *Service) GetScenario(id string) (*FullScenario, error) {
    scenarioDir := filepath.Join(s.scenariosDir, id)

    // Load config
    cfg := NewConfig(filepath.Join(scenarioDir, "config.json"))
    if err := cfg.Load(); err != nil {
        return nil, err
    }

    // Load messages
    msgs := NewScenarioMessages(filepath.Join(scenarioDir, "messages.json"))
    if err := msgs.Load(); err != nil {
        return nil, err
    }

    // Load summary
    summary := NewScenarioSummary(filepath.Join(scenarioDir, "summary.json"))
    if err := summary.Load(); err != nil {
        return nil, err
    }

    return &FullScenario{
        ID:       cfg.ID,
        Name:     cfg.Name,
        Config:   cfg,
        Messages: msgs,
        Summary:  summary,
    }, nil
}

// ListScenarios returns all scenarios
func (s *Service) ListScenarios() []*ScenarioInfo {
    scenarios := s.registry.List()
    infos := make([]*ScenarioInfo, len(scenarios))
    for i, sc := range scenarios {
        infos[i] = &ScenarioInfo{
            ID:       sc.ID,
            Name:     sc.Name,
            IsActive: sc.IsActive,
        }
    }
    return infos
}

// GetDefaultScenario returns the default scenario
func (s *Service) GetDefaultScenario() (*FullScenario, error) {
    sc, err := s.registry.GetDefault()
    if err != nil {
        return nil, err
    }
    return s.GetScenario(sc.ID)
}

// SetDefaultScenario sets the default scenario
func (s *Service) SetDefaultScenario(id string) error {
    return s.registry.SetDefault(id)
}

// ScenarioInfo contains basic scenario information
type ScenarioInfo struct {
    ID       string
    Name     string
    IsActive bool
}

// FullScenario contains complete scenario data
type FullScenario struct {
    ID       string
    Name     string
    Config   *Config
    Messages *ScenarioMessages
    Summary  *ScenarioSummary
}

// generateScenarioID generates a scenario ID from name
func generateScenarioID(name string) string {
    // Transliterate and lowercase
    id := strings.ToLower(strings.TrimSpace(name))
    // Replace spaces with hyphens
    id = strings.ReplaceAll(id, " ", "-")
    // Remove non-alphanumeric chars (except hyphens)
    id = strings.Map(func(r rune) rune {
        if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
            return r
        }
        return -1
    }, id)

    // Ensure uniqueness by adding timestamp if needed
    return fmt.Sprintf("%s-%d", id, time.Now().Unix())
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/services/scenario/... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/services/scenario/service.go internal/services/scenario/service_test.go
git commit -m "feat: add scenario service"
```

---

### Task 4.2: Create Scenario Summary Storage

**Files:**
- Create: `internal/services/scenario/summary.go`
- Test: `internal/services/scenario/summary_test.go`

**Step 1: Write the failing test**

Create: `internal/services/scenario/summary_test.go`

```go
package scenario

import (
    "os"
    "path/filepath"
    "testing"
)

func TestScenarioSummary_LoadSave(t *testing.T) {
    tmpDir := t.TempDir()
    scenarioDir := filepath.Join(tmpDir, "scenarios", "test")
    summaryPath := filepath.Join(scenarioDir, "summary.json")

    // Create summary
    summary := NewScenarioSummary(summaryPath)
    summary.TemplateFile = "summary.md"
    summary.Photos = []string{"summary_1.PNG"}

    // Save
    err := summary.Save()
    if err != nil {
        t.Fatalf("Failed to save: %v", err)
    }

    // Load into new summary
    summary2 := NewScenarioSummary(summaryPath)
    err = summary2.Load()
    if err != nil {
        t.Fatalf("Failed to load: %v", err)
    }

    // Verify
    if summary2.TemplateFile != "summary.md" {
        t.Errorf("Expected template 'summary.md', got '%s'", summary2.TemplateFile)
    }
    if len(summary2.Photos) != 1 {
        t.Errorf("Expected 1 photo, got %d", len(summary2.Photos))
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/services/scenario/... -v`
Expected: FAIL with "undefined: NewScenarioSummary"

**Step 3: Write minimal implementation**

Create: `internal/services/scenario/summary.go`

```go
package scenario

import (
    "encoding/json"
    "errors"
    "os"
    "path/filepath"
    "sync"
)

// ScenarioSummary manages scenario summary persistence
type ScenarioSummary struct {
    filePath string
    mu       sync.RWMutex

    TemplateFile   string                `json:"template_file"`
    Photos         []string              `json:"photos,omitempty"`
    InlineKeyboard *InlineKeyboardConfig `json:"inline_keyboard,omitempty"`
}

// NewScenarioSummary creates a new scenario summary
func NewScenarioSummary(filePath string) *ScenarioSummary {
    return &ScenarioSummary{
        filePath: filePath,
        Photos:   make([]string, 0),
    }
}

// Load loads the summary from disk
func (s *ScenarioSummary) Load() error {
    s.mu.Lock()
    defer s.mu.Unlock()

    data, err := os.ReadFile(s.filePath)
    if err != nil {
        if errors.Is(err, os.ErrNotExist) {
            // Return empty summary
            return nil
        }
        return errors.New("failed to load summary")
    }

    if err := json.Unmarshal(data, s); err != nil {
        return errors.New("failed to load summary")
    }

    return nil
}

// Save saves the summary to disk
func (s *ScenarioSummary) Save() error {
    s.mu.RLock()
    defer s.mu.RUnlock()

    // Ensure directory exists
    if err := os.MkdirAll(filepath.Dir(s.filePath), 0755); err != nil {
        return errors.New("failed to save summary")
    }

    data, err := json.MarshalIndent(s, "", "  ")
    if err != nil {
        return errors.New("failed to save summary")
    }

    if err := os.WriteFile(s.filePath, data, 0644); err != nil {
        return errors.New("failed to save summary")
    }

    return nil
}

// ToDomain converts to domain scenario summary
func (s *ScenarioSummary) ToDomain() *scenario.ScenarioSummary {
    return &scenario.ScenarioSummary{
        TemplateFile:   s.TemplateFile,
        Photos:         s.Photos,
        InlineKeyboard: convertSummaryInlineKeyboard(s.InlineKeyboard),
    }
}

func convertSummaryInlineKeyboard(ik *InlineKeyboardConfig) *scenario.InlineKeyboardConfig {
    if ik == nil {
        return nil
    }
    domainIK := &scenario.InlineKeyboardConfig{
        Rows: make([]scenario.InlineKeyboardRowConfig, len(ik.Rows)),
    }
    for i, row := range ik.Rows {
        domainIK.Rows[i] = scenario.InlineKeyboardRowConfig{
            Buttons: make([]scenario.InlineKeyboardButtonConfig, len(row.Buttons)),
        }
        for j, btn := range row.Buttons {
            domainIK.Rows[i].Buttons[j] = scenario.InlineKeyboardButtonConfig{
                Type:     btn.Type,
                Text:     btn.Text,
                URL:      btn.URL,
                Callback: btn.Callback,
            }
        }
    }
    return domainIK
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/services/scenario/... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/services/scenario/summary.go internal/services/scenario/summary_test.go
git commit -m "feat: add scenario summary storage"
```

---

## Phase 5: User Scenario Management

### Task 5.1: Create User Scenario Service

**Files:**
- Create: `internal/services/scenario_user/service.go`
- Test: `internal/services/scenario_user/service_test.go`

**Step 1: Write the failing test**

Create: `internal/services/scenario_user/service_test.go`

```go
package scenario_user

import (
    "testing"
    "time"

    "mispilka-bot-tg/internal/domain/scenario"
)

func TestService_StartScenario(t *testing.T) {
    service := NewService()

    // Mock scenario
    sc := &scenario.Scenario{
        ID:   "test",
        Name: "Test",
        Config: scenario.ScenarioConfig{
            Prodamus: scenario.ProdamusConfig{
                ProductName:    "Test Product",
                ProductPrice:   "1000",
                PrivateGroupID: "-1001234567890",
            },
        },
        Messages: scenario.ScenarioMessages{
            MessagesList: []string{"msg_1"},
        },
    }

    chatID := "123456"

    // Start scenario
    state, err := service.StartScenario(chatID, sc)
    if err != nil {
        t.Fatalf("Failed to start scenario: %v", err)
    }

    if state.Status != scenario.StatusActive {
        t.Errorf("Expected status %s, got %s", scenario.StatusActive, state.Status)
    }
    if state.CurrentMessageIndex != 0 {
        t.Errorf("Expected index 0, got %d", state.CurrentMessageIndex)
    }
}

func TestService_SwitchScenario(t *testing.T) {
    service := NewService()

    sc1 := &scenario.Scenario{
        ID:   "scenario1",
        Name: "Scenario 1",
        Config: scenario.ScenarioConfig{
            Prodamus: scenario.ProdamusConfig{
                ProductName:    "Product 1",
                ProductPrice:   "1000",
                PrivateGroupID: "-1001234567890",
            },
        },
    }

    sc2 := &scenario.Scenario{
        ID:   "scenario2",
        Name: "Scenario 2",
        Config: scenario.ScenarioConfig{
            Prodamus: scenario.ProdamusConfig{
                ProductName:    "Product 2",
                ProductPrice:   "2000",
                PrivateGroupID: "-1001234567890",
            },
        },
    }

    chatID := "123456"

    // Start first scenario
    _, _ = service.StartScenario(chatID, sc1)

    // Switch to second scenario
    state, err := service.SwitchScenario(chatID, sc2)
    if err != nil {
        t.Fatalf("Failed to switch scenario: %v", err)
    }

    if state.Status != scenario.StatusActive {
        t.Errorf("Expected status %s, got %s", scenario.StatusActive, state.Status)
    }

    // Verify first scenario was stopped
    sc1State, _ := service.GetScenarioState(chatID, "scenario1")
    if sc1State.Status != scenario.StatusStopped {
        t.Errorf("Expected first scenario to be stopped, got %s", sc1State.Status)
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/services/scenario_user/... -v`
Expected: FAIL with "undefined: NewService"

**Step 3: Write minimal implementation**

Create: `internal/services/scenario_user/service.go`

```go
package scenario_user

import (
    "fmt"
    "time"

    "mispilka-bot-tg/internal/domain/scenario"
    "mispilka-bot-tg/internal/services"
)

// Service manages user-scenario state
type Service struct {
    userService *services.UserService
}

// NewService creates a new user-scenario service
func NewService() *Service {
    return &Service{
        userService: services.NewUserService(),
    }
}

// Initialize initializes the service
func (s *Service) Initialize() error {
    return s.userService.LoadData()
}

// GetScenarioState retrieves user's state for a scenario
func (s *Service) GetScenarioState(chatID, scenarioID string) (*services.UserScenarioState, error) {
    return s.userService.GetUserScenario(chatID, scenarioID)
}

// SetScenarioState sets user's state for a scenario
func (s *Service) SetScenarioState(chatID, scenarioID string, state *services.UserScenarioState) error {
    return s.userService.SetUserScenario(chatID, scenarioID, state)
}

// GetActiveScenario retrieves user's active scenario
func (s *Service) GetActiveScenario(chatID string) (string, *services.UserScenarioState, error) {
    return s.userService.GetUserActiveScenario(chatID)
}

// SetActiveScenario sets user's active scenario
func (s *Service) SetActiveScenario(chatID, scenarioID string) error {
    return s.userService.SetUserActiveScenario(chatID, scenarioID)
}

// StartScenario starts a scenario for a user
func (s *Service) StartScenario(chatID string, sc *scenario.Scenario) (*services.UserScenarioState, error) {
    // Check if scenario is already active
    activeID, activeState, err := s.GetActiveScenario(chatID)
    if err != nil {
        return nil, err
    }

    // Get or create scenario state
    state, err := s.GetScenarioState(chatID, sc.ID)
    if err != nil {
        return nil, err
    }

    // Handle scenario switching logic
    if activeID != "" && activeID != sc.ID {
        // Stop current scenario
        activeState.MarkStopped()
        _ = s.SetScenarioState(chatID, activeID, activeState)
    }

    // Start new scenario
    now := time.Now()
    state.Status = scenario.StatusActive
    state.CurrentMessageIndex = 0
    state.LastSentAt = &now

    if err := s.SetScenarioState(chatID, sc.ID, state); err != nil {
        return nil, err
    }

    if err := s.SetActiveScenario(chatID, sc.ID); err != nil {
        return nil, err
    }

    return state, nil
}

// SwitchScenario switches user to a different scenario
func (s *Service) SwitchScenario(chatID string, newSc *scenario.Scenario) (*services.UserScenarioState, error) {
    activeID, activeState, err := s.GetActiveScenario(chatID)
    if err != nil {
        return nil, err
    }

    // Get new scenario state
    newState, err := s.GetScenarioState(chatID, newSc.ID)
    if err != nil {
        return nil, err
    }

    // Handle based on new scenario state
    if newState.IsCompleted() {
        // Scenario already completed - will send summary only
        if err := s.SetActiveScenario(chatID, newSc.ID); err != nil {
            return nil, err
        }
        return newState, nil
    }

    // Stop current scenario if different
    if activeID != "" && activeID != newSc.ID && activeState != nil {
        activeState.MarkStopped()
        _ = s.SetScenarioState(chatID, activeID, activeState)
    }

    // Start/continue new scenario
    if newState.IsNotStarted() || newState.IsStopped() {
        now := time.Now()
        newState.Status = scenario.StatusActive
        newState.CurrentMessageIndex = 0
        newState.LastSentAt = &now
    } else {
        // Resume from current position
        now := time.Now()
        newState.LastSentAt = &now
    }

    if err := s.SetScenarioState(chatID, newSc.ID, newState); err != nil {
        return nil, err
    }

    if err := s.SetActiveScenario(chatID, newSc.ID); err != nil {
        return nil, err
    }

    return newState, nil
}

// CompleteScenario marks a scenario as completed
func (s *Service) CompleteScenario(chatID, scenarioID string) error {
    state, err := s.GetScenarioState(chatID, scenarioID)
    if err != nil {
        return err
    }

    state.MarkCompleted()
    return s.SetScenarioState(chatID, scenarioID, state)
}

// AdvanceMessage advances the user to the next message in the scenario
func (s *Service) AdvanceMessage(chatID, scenarioID string) (bool, error) {
    state, err := s.GetScenarioState(chatID, scenarioID)
    if err != nil {
        return false, err
    }

    state.CurrentMessageIndex++
    now := time.Now()
    state.LastSentAt = &now

    if err := s.SetScenarioState(chatID, scenarioID, state); err != nil {
        return false, err
    }

    return true, nil
}

// IsScenarioCompleted checks if user completed a scenario
func (s *Service) IsScenarioCompleted(chatID, scenarioID string) (bool, error) {
    return s.userService.IsScenarioCompleted(chatID, scenarioID)
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/services/scenario_user/... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/services/scenario_user/service.go internal/services/scenario_user/service_test.go
git commit -m "feat: add user scenario service"
```

---

## Phase 6: Scenario Scheduler

### Task 6.1: Create Scenario Scheduler Service

**Files:**
- Create: `internal/services/scenario_scheduler/service.go`
- Test: `internal/services/scenario_scheduler/service_test.go`

**Step 1: Write the failing test**

Create: `internal/services/scenario_scheduler/service_test.go`

```go
package scenario_scheduler

import (
    "testing"
    "time"

    "mispilka-bot-tg/internal/domain/scenario"
)

func TestScheduler_ScheduleNextMessage(t *testing.T) {
    scheduler := NewScheduler()

    sc := &scenario.Scenario{
        ID:   "test",
        Name: "Test",
        Messages: scenario.ScenarioMessages{
            MessagesList: []string{"msg_1", "msg_2"},
            Messages: map[string]scenario.MessageData{
                "msg_1": {
                    Timing: scenario.Timing{Hours: 0, Minutes: 0},
                },
                "msg_2": {
                    Timing: scenario.Timing{Hours: 1, Minutes: 0},
                },
            },
        },
    }

    chatID := "123456"
    state := &scenario.UserScenarioState{
        Status:              scenario.StatusActive,
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
    scheduler := NewScheduler()

    sc := &scenario.Scenario{
        Messages: scenario.ScenarioMessages{
            MessagesList: []string{"msg_1", "msg_2"},
            Messages: map[string]scenario.MessageData{
                "msg_1": {
                    Timing: scenario.Timing{Hours: 0, Minutes: 0},
                },
                "msg_2": {
                    Timing: scenario.Timing{Hours: 1, Minutes: 0},
                },
            },
        },
    }

    state := &scenario.UserScenarioState{
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
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/services/scenario_scheduler/... -v`
Expected: FAIL with "undefined: NewScheduler"

**Step 3: Write minimal implementation**

Create: `internal/services/scenario_scheduler/service.go`

```go
package scenario_scheduler

import (
    "fmt"
    "sync"
    "time"

    "mispilka-bot-tg/internal/domain/scenario"
)

// ScheduleInfo contains schedule information
type ScheduleInfo struct {
    ChatID      string
    ScenarioID  string
    MessageIndex int
    ScheduledAt time.Time
}

// Scheduler manages per-scenario message scheduling
type Scheduler struct {
    mu         sync.RWMutex
    schedules  map[string]*ScheduleInfo // chatID -> ScheduleInfo
    timers     map[string]*time.Timer   // chatID -> Timer
    callbacks  chan *ScheduleInfo
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
func (s *Scheduler) ScheduleNextMessage(chatID string, sc *scenario.Scenario, state *scenario.UserScenarioState) (time.Time, error) {
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
func (s *Scheduler) GetNextMessage(sc *scenario.Scenario, state *scenario.UserScenarioState) (*scenario.MessageData, error) {
    return s.getNextMessage(sc, state)
}

// getNextMessage is internal method to get next message
func (s *Scheduler) getNextMessage(sc *scenario.Scenario, state *scenario.UserScenarioState) (*scenario.MessageData, error) {
    if state.CurrentMessageIndex >= len(sc.Messages.MessagesList) {
        return nil, fmt.Errorf("no more messages in scenario")
    }

    msgID := sc.Messages.MessagesList[state.CurrentMessageIndex]
    msg, ok := sc.Messages.Messages[msgID]
    if !ok {
        return nil, fmt.Errorf("message %s not found", msgID)
    }

    // Convert to domain message data
    domainMsg := &scenario.MessageData{
        Timing:       msg.Timing,
        TemplateFile: msg.TemplateFile,
        Photos:       msg.Photos,
    }

    return domainMsg, nil
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
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/services/scenario_scheduler/... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/services/scenario_scheduler/service.go internal/services/scenario_scheduler/service_test.go
git commit -m "feat: add scenario scheduler service"
```

---

## Phase 7: Broadcast System

### Task 7.1: Create Broadcast Registry Storage

**Files:**
- Create: `internal/services/broadcast/registry.go`
- Test: `internal/services/broadcast/registry_test.go`

**Step 1: Write the failing test**

Create: `internal/services/broadcast/registry_test.go`

```go
package broadcast

import (
    "os"
    "path/filepath"
    "testing"
    "time"

    "mispilka-bot-tg/internal/domain/broadcast"
)

func TestBroadcastRegistry_LoadSave(t *testing.T) {
    tmpDir := t.TempDir()
    registryPath := filepath.Join(tmpDir, "broadcasts", "registry.json")

    reg := NewRegistry(registryPath)

    bc := &broadcast.Broadcast{
        ID:           "test_bc",
        Name:         "Test Broadcast",
        TemplateFile: "test.md",
        Targeting: &broadcast.Targeting{
            Conditions: []string{broadcast.ConditionNoActiveScenario},
        },
        CreatedAt: time.Now(),
    }

    err := reg.Add(bc)
    if err != nil {
        t.Fatalf("Failed to add broadcast: %v", err)
    }

    // Load into new registry
    reg2 := NewRegistry(registryPath)
    err = reg2.Load()
    if err != nil {
        t.Fatalf("Failed to load: %v", err)
    }

    // Verify
    loaded, ok := reg2.Get("test_bc")
    if !ok {
        t.Fatal("Broadcast not found")
    }
    if loaded.Name != "Test Broadcast" {
        t.Errorf("Expected name 'Test Broadcast', got '%s'", loaded.Name)
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/services/broadcast/... -v`
Expected: FAIL with "undefined: NewRegistry"

**Step 3: Write minimal implementation**

Create: `internal/services/broadcast/registry.go`

```go
package broadcast

import (
    "encoding/json"
    "errors"
    "os"
    "path/filepath"
    "sync"

    "mispilka-bot-tg/internal/domain/broadcast"
)

var (
    ErrBroadcastNotFound = errors.New("broadcast not found")
    ErrLoadFailed        = errors.New("failed to load broadcast registry")
    ErrSaveFailed        = errors.New("failed to save broadcast registry")
)

// Registry manages broadcast persistence
type Registry struct {
    filePath string
    mu       sync.RWMutex

    registry *broadcast.BroadcastRegistry
}

// NewRegistry creates a new broadcast registry
func NewRegistry(filePath string) *Registry {
    return &Registry{
        filePath: filePath,
        registry: broadcast.NewBroadcastRegistry(),
    }
}

// Load loads the registry from disk
func (r *Registry) Load() error {
    r.mu.Lock()
    defer r.mu.Unlock()

    data, err := os.ReadFile(r.filePath)
    if err != nil {
        if errors.Is(err, os.ErrNotExist) {
            // Create new registry
            return nil
        }
        return ErrLoadFailed
    }

    if err := json.Unmarshal(data, r.registry); err != nil {
        return ErrLoadFailed
    }

    return nil
}

// Save saves the registry to disk
func (r *Registry) Save() error {
    r.mu.RLock()
    defer r.mu.RUnlock()

    // Ensure directory exists
    if err := os.MkdirAll(filepath.Dir(r.filePath), 0755); err != nil {
        return ErrSaveFailed
    }

    data, err := json.MarshalIndent(r.registry, "", "  ")
    if err != nil {
        return ErrSaveFailed
    }

    if err := os.WriteFile(r.filePath, data, 0644); err != nil {
        return ErrSaveFailed
    }

    return nil
}

// Get retrieves a broadcast by ID
func (r *Registry) Get(id string) (*broadcast.Broadcast, bool) {
    r.mu.RLock()
    defer r.mu.RUnlock()
    return r.registry.Get(id)
}

// Add adds a broadcast to the registry
func (r *Registry) Add(bc *broadcast.Broadcast) error {
    r.mu.Lock()
    defer r.mu.Unlock()

    r.registry.Add(bc)
    return r.Save()
}

// Delete removes a broadcast from the registry
func (r *Registry) Delete(id string) bool {
    r.mu.Lock()
    defer r.mu.Unlock()

    deleted := r.registry.Delete(id)
    if deleted {
        _ = r.Save()
    }
    return deleted
}

// List returns all broadcasts
func (r *Registry) List() []*broadcast.Broadcast {
    r.mu.RLock()
    defer r.mu.RUnlock()
    return r.registry.List()
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/services/broadcast/... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/services/broadcast/registry.go internal/services/broadcast/registry_test.go
git commit -m "feat: add broadcast registry storage"
```

---

### Task 7.2: Create Broadcast Sender Service

**Files:**
- Create: `internal/services/broadcast/sender.go`
- Test: `internal/services/broadcast/sender_test.go`

**Step 1: Write the failing test**

Create: `internal/services/broadcast/sender_test.go`

```go
package broadcast

import (
    "testing"
    "time"

    "mispilka-bot-tg/internal/domain/broadcast"
)

func TestSender_ShouldSend(t *testing.T) {
    sender := NewSender(nil)

    bc := &broadcast.Broadcast{
        ID:           "test",
        Name:         "Test",
        TemplateFile: "test.md",
        Targeting: &broadcast.Targeting{
            Conditions: []string{broadcast.ConditionNoActiveScenario},
        },
        CreatedAt: time.Now(),
    }

    // Test with user without active scenario
    user := &MockUser{
        ActiveScenarioID: "",
    }

    if !sender.ShouldSend(bc, user) {
        t.Error("Expected to send to user without active scenario")
    }

    // Test with user with active scenario
    userWithActive := &MockUser{
        ActiveScenarioID: "default",
    }

    if sender.ShouldSend(bc, userWithActive) {
        t.Error("Expected not to send to user with active scenario")
    }
}

// MockUser for testing
type MockUser struct {
    ActiveScenarioID string
    HasPaid          bool
}

func (m *MockUser) GetActiveScenarioID() string {
    return m.ActiveScenarioID
}

func (m *MockUser) HasPaidAnyProduct() bool {
    return m.HasPaid
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/services/broadcast/... -v`
Expected: FAIL with "undefined: NewSender"

**Step 3: Write minimal implementation**

Create: `internal/services/broadcast/sender.go`

```go
package broadcast

import (
    "mispilka-bot-tg/internal/domain/broadcast"
)

// UserWithScenarios interface for checking user state
type UserWithScenarios interface {
    GetActiveScenarioID() string
    HasPaidAnyProduct() bool
}

// Sender handles broadcast sending with targeting
type Sender struct {
    registry *Registry
}

// NewSender creates a new broadcast sender
func NewSender(registry *Registry) *Sender {
    return &Sender{
        registry: registry,
    }
}

// ShouldSend determines if a broadcast should be sent to a user
func (s *Sender) ShouldSend(bc *broadcast.Broadcast, user UserWithScenarios) bool {
    if bc.Targeting == nil || len(bc.Targeting.Conditions) == 0 {
        // No targeting - send to everyone
        return true
    }

    for _, condition := range bc.Targeting.Conditions {
        if !s.checkCondition(condition, user) {
            return false
        }
    }

    return true
}

// checkCondition checks a single targeting condition
func (s *Sender) checkCondition(condition string, user UserWithScenarios) bool {
    switch condition {
    case broadcast.ConditionNoActiveScenario:
        return user.GetActiveScenarioID() == ""
    case broadcast.ConditionHasNotPaid:
        return !user.HasPaidAnyProduct()
    default:
        // Unknown condition - assume false
        return false
    }
}

// CalculateRecipients calculates which users should receive a broadcast
func (s *Sender) CalculateRecipients(bc *broadcast.Broadcast, users []UserWithScenarios) []UserWithScenarios {
    recipients := make([]UserWithScenarios, 0)
    for _, user := range users {
        if s.ShouldSend(bc, user) {
            recipients = append(recipients, user)
        }
    }
    return recipients
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/services/broadcast/... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/services/broadcast/sender.go internal/services/broadcast/sender_test.go
git commit -m "feat: add broadcast sender service"
```

---

## Phase 8: Button Registry

### Task 8.1: Create Button Registry Storage

**Files:**
- Create: `internal/services/button/registry.go`
- Test: `internal/services/button/registry_test.go`

**Step 1: Write the failing test**

Create: `internal/services/button/registry_test.go`

```go
package button

import (
    "os"
    "path/filepath"
    "testing"

    "mispilka-bot-tg/internal/domain/button"
)

func TestButtonRegistry_LoadSave(t *testing.T) {
    tmpDir := t.TempDir()
    registryPath := filepath.Join(tmpDir, "buttons", "registry.json")

    reg := NewRegistry(registryPath)

    bs := &button.ButtonSet{
        Rows: []button.ButtonRow{
            {
                Buttons: []button.Button{
                    {
                        Type: "url",
                        Text: "Pay",
                        URL:  "https://pay.example.com",
                    },
                },
            },
        },
    }

    err := reg.Set("payment_button", bs)
    if err != nil {
        t.Fatalf("Failed to set button set: %v", err)
    }

    // Load into new registry
    reg2 := NewRegistry(registryPath)
    err = reg2.Load()
    if err != nil {
        t.Fatalf("Failed to load: %v", err)
    }

    // Verify
    loaded, ok := reg2.Get("payment_button")
    if !ok {
        t.Fatal("Button set not found")
    }
    if len(loaded.Rows) != 1 {
        t.Errorf("Expected 1 row, got %d", len(loaded.Rows))
    }
    if loaded.Rows[0].Buttons[0].Text != "Pay" {
        t.Errorf("Expected button text 'Pay', got '%s'", loaded.Rows[0].Buttons[0].Text)
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/services/button/... -v`
Expected: FAIL with "undefined: NewRegistry"

**Step 3: Write minimal implementation**

Create: `internal/services/button/registry.go`

```go
package button

import (
    "encoding/json"
    "errors"
    "os"
    "path/filepath"
    "sync"

    "mispilka-bot-tg/internal/domain/button"
)

var (
    ErrLoadFailed = errors.New("failed to load button registry")
    ErrSaveFailed = errors.New("failed to save button registry")
)

// Registry manages button set persistence
type Registry struct {
    filePath string
    mu       sync.RWMutex

    registry *button.ButtonRegistry
}

// NewRegistry creates a new button registry
func NewRegistry(filePath string) *Registry {
    return &Registry{
        filePath: filePath,
        registry: button.NewButtonRegistry(),
    }
}

// Load loads the registry from disk
func (r *Registry) Load() error {
    r.mu.Lock()
    defer r.mu.Unlock()

    data, err := os.ReadFile(r.filePath)
    if err != nil {
        if errors.Is(err, os.ErrNotExist) {
            // Create new registry
            return nil
        }
        return ErrLoadFailed
    }

    if err := json.Unmarshal(data, r.registry); err != nil {
        return ErrLoadFailed
    }

    return nil
}

// Save saves the registry to disk
func (r *Registry) Save() error {
    r.mu.RLock()
    defer r.mu.RUnlock()

    // Ensure directory exists
    if err := os.MkdirAll(filepath.Dir(r.filePath), 0755); err != nil {
        return ErrSaveFailed
    }

    data, err := json.MarshalIndent(r.registry, "", "  ")
    if err != nil {
        return ErrSaveFailed
    }

    if err := os.WriteFile(r.filePath, data, 0644); err != nil {
        return ErrSaveFailed
    }

    return nil
}

// Get retrieves a button set by reference
func (r *Registry) Get(ref string) (*button.ButtonSet, bool) {
    r.mu.RLock()
    defer r.mu.RUnlock()
    return r.registry.Get(ref)
}

// Set stores a button set
func (r *Registry) Set(ref string, bs *button.ButtonSet) error {
    r.mu.Lock()
    defer r.mu.Unlock()

    r.registry.Set(ref, bs)
    return r.Save()
}

// Delete removes a button set
func (r *Registry) Delete(ref string) error {
    r.mu.Lock()
    defer r.mu.Unlock()

    r.registry.Delete(ref)
    return r.Save()
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/services/button/... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/services/button/registry.go internal/services/button/registry_test.go
git commit -m "feat: add button registry storage"
```

---

## Phase 9: Wizard System

### Task 9.1: Create Wizard State Management

**Files:**
- Create: `internal/services/wizard/types.go`
- Create: `internal/services/wizard/manager.go`
- Test: `internal/services/wizard/types_test.go`

**Step 1: Write the failing test**

Create: `internal/services/wizard/types_test.go`

```go
package wizard

import (
    "testing"
    "time"
)

func TestWizardState_Expired(t *testing.T) {
    state := &WizardState{
        StartedAt: time.Now().Add(-31 * time.Minute),
        Timeout:   30 * time.Minute,
    }

    if !state.Expired() {
        t.Error("Expected wizard to be expired")
    }
}

func TestWizardState_NotExpired(t *testing.T) {
    state := &WizardState{
        StartedAt: time.Now().Add(-10 * time.Minute),
        Timeout:   30 * time.Minute,
    }

    if state.Expired() {
        t.Error("Expected wizard not to be expired")
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/services/wizard/... -v`
Expected: FAIL with "undefined: WizardState"

**Step 3: Write minimal implementation**

Create: `internal/services/wizard/types.go`

```go
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
    WizardTypeCreateScenario WizardType = "create_scenario"
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
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/services/wizard/... -v`
Expected: PASS

**Step 5: Create wizard manager**

Create: `internal/services/wizard/manager.go`

```go
package wizard

import (
    "encoding/json"
    "errors"
    "os"
    "path/filepath"
    "sync"
    "time"
)

var (
    ErrWizardNotFound    = errors.New("wizard not found")
    ErrWizardExpired     = errors.New("wizard expired")
    ErrSaveFailed        = errors.New("failed to save wizard state")
)

// Manager manages wizard states
type Manager struct {
    wizardsDir string
    mu         sync.RWMutex
    states     map[string]*WizardState // userID -> WizardState
}

// NewManager creates a new wizard manager
func NewManager(wizardsDir string) *Manager {
    return &Manager{
        wizardsDir: wizardsDir,
        states:     make(map[string]*WizardState),
    }
}

// Initialize initializes the manager (loads existing states)
func (m *Manager) Initialize() error {
    m.mu.Lock()
    defer m.mu.Unlock()

    // Ensure directory exists
    if err := os.MkdirAll(m.wizardsDir, 0755); err != nil {
        return err
    }

    // Clean up expired wizards from disk
    entries, err := os.ReadDir(m.wizardsDir)
    if err != nil {
        return err
    }

    for _, entry := range entries {
        if entry.IsDir() {
            continue
        }

        // Load state
        filePath := filepath.Join(m.wizardsDir, entry.Name())
        data, err := os.ReadFile(filePath)
        if err != nil {
            continue
        }

        var state WizardState
        if err := json.Unmarshal(data, &state); err != nil {
            continue
        }

        // Check if expired
        if state.Expired() {
            _ = os.Remove(filePath)
            continue
        }

        // Load into memory
        m.states[state.UserID] = &state
    }

    return nil
}

// Start starts a new wizard for a user
func (m *Manager) Start(userID string, wizardType WizardType) (*WizardState, error) {
    m.mu.Lock()
    defer m.mu.Unlock()

    // Cancel existing wizard if any
    if existing, ok := m.states[userID]; ok {
        _ = m.deleteExisting(existing.UserID)
    }

    // Create new wizard state
    state := &WizardState{
        UserID:     userID,
        WizardType: wizardType,
        CurrentStep: getFirstStep(wizardType),
        StartedAt:  time.Now(),
        Timeout:    30 * time.Minute,
        Data:       make(map[string]interface{}),
    }

    m.states[userID] = state

    // Save to disk
    if err := m.saveState(state); err != nil {
        delete(m.states, userID)
        return nil, err
    }

    return state, nil
}

// Get retrieves a wizard state
func (m *Manager) Get(userID string) (*WizardState, error) {
    m.mu.RLock()
    defer m.mu.RUnlock()

    state, ok := m.states[userID]
    if !ok {
        return nil, ErrWizardNotFound
    }

    // Check if expired
    if state.Expired() {
        go m.Cancel(userID)
        return nil, ErrWizardExpired
    }

    return state, nil
}

// Update updates a wizard state
func (m *Manager) Update(userID string, state *WizardState) error {
    m.mu.Lock()
    defer m.mu.Unlock()

    if _, ok := m.states[userID]; !ok {
        return ErrWizardNotFound
    }

    // Reset timeout on update
    state.ResetTimeout()

    m.states[userID] = state
    return m.saveState(state)
}

// Advance advances a wizard to the next step
func (m *Manager) Advance(userID string, nextStep WizardStep) error {
    m.mu.Lock()
    defer m.mu.Unlock()

    state, ok := m.states[userID]
    if !ok {
        return ErrWizardNotFound
    }

    state.CurrentStep = nextStep
    state.ResetTimeout()
    return m.saveState(state)
}

// Cancel cancels a wizard
func (m *Manager) Cancel(userID string) error {
    m.mu.Lock()
    defer m.mu.Unlock()

    state, ok := m.states[userID]
    if !ok {
        return ErrWizardNotFound
    }

    return m.deleteExisting(userID)
}

// deleteExisting removes wizard from memory and disk
func (m *Manager) deleteExisting(userID string) error {
    delete(m.states, userID)

    filePath := filepath.Join(m.wizardsDir, userID+".json")
    if err := os.Remove(filePath); err != nil && !errors.Is(err, os.ErrNotExist) {
        return ErrSaveFailed
    }

    return nil
}

// saveState saves wizard state to disk
func (m *Manager) saveState(state *WizardState) error {
    filePath := filepath.Join(m.wizardsDir, state.UserID+".json")
    data, err := json.MarshalIndent(state, "", "  ")
    if err != nil {
        return ErrSaveFailed
    }

    if err := os.WriteFile(filePath, data, 0644); err != nil {
        return ErrSaveFailed
    }

    return nil
}

// CleanupExpired removes expired wizard states
func (m *Manager) CleanupExpired() {
    m.mu.Lock()
    defer m.mu.Unlock()

    for userID, state := range m.states {
        if state.Expired() {
            _ = m.deleteExisting(userID)
        }
    }
}

// getFirstStep returns the first step for a wizard type
func getFirstStep(wizardType WizardType) WizardStep {
    switch wizardType {
    case WizardTypeCreateScenario:
        return StepScenarioName
    case WizardTypeCreateBroadcast:
        return "broadcast_name" // Define this step if needed
    default:
        return ""
    }
}
```

**Step 6: Run test to verify it passes**

Run: `go test ./internal/services/wizard/... -v`
Expected: PASS

**Step 7: Commit**

```bash
git add internal/services/wizard/
git commit -m "feat: add wizard state management"
```

---

## Phase 10: Admin Commands

### Task 10.1: Create Scenario Admin Command Handlers

**Files:**
- Create: `internal/telegram/admin_handlers.go`
- Create: `internal/telegram/admin_wizard.go`

**Step 1: Create admin handlers file**

Create: `internal/telegram/admin_handlers.go`

```go
package telegram

import (
    "fmt"
    "strings"

    "mispilka-bot-tg/internal/domain/command"
    "mispilka-bot-tg/internal/services/scenario"
    "mispilka-bot-tg/internal/services/wizard"
)

// registerAdminCommands registers all admin commands
func (b *Bot) registerAdminCommands() {
    adminCommands := []*command.Command{
        {
            Name:        "scenarios",
            Description: "Show all scenarios",
            Role:        command.RoleAdmin,
            Handler:     b.scenariosCommand,
        },
        {
            Name:        "create_scenario",
            Description: "Create a new scenario",
            Role:        command.RoleAdmin,
            Handler:     b.createScenarioCommand,
        },
        {
            Name:        "set_default_scenario",
            Description: "Set default scenario",
            Role:        command.RoleAdmin,
            Handler:     b.setDefaultScenarioCommand,
        },
        {
            Name:        "delete_scenario",
            Description: "Delete a scenario",
            Role:        command.RoleAdmin,
            Handler:     b.deleteScenarioCommand,
        },
        {
            Name:        "demo_scenario",
            Description: "Demonstrate a scenario",
            Role:        command.RoleAdmin,
            Handler:     b.demoScenarioCommand,
        },
        {
            Name:        "create_broadcast",
            Description: "Create a broadcast",
            Role:        command.RoleAdmin,
            Handler:     b.createBroadcastCommand,
        },
        {
            Name:        "send_broadcast",
            Description: "Send a broadcast",
            Role:        command.RoleAdmin,
            Handler:     b.sendBroadcastCommand,
        },
    }

    for _, cmd := range adminCommands {
        b.commandService.RegisterCommand(cmd)
    }
}

// scenariosCommand shows all scenarios as buttons
func (b *Bot) scenariosCommand(chatID int64, payload string) error {
    scenarios := b.scenarioService.ListScenarios()

    if len(scenarios) == 0 {
        return b.sendMessage(chatID, "No scenarios found. Use /create_scenario to create one.")
    }

    // Build message with scenario buttons
    var sb strings.Builder
    sb.WriteString("📋 Available Scenarios:\n\n")

    for _, sc := range scenarios {
        marker := " "
        if b.isDefaultScenario(sc.ID) {
            marker = "✅"
        }
        sb.WriteString(fmt.Sprintf("%s %s (%s)\n", marker, sc.Name, sc.ID))
    }

    // Create inline keyboard for scenario actions
    keyboard := b.buildScenarioKeyboard(scenarios)

    return b.sendMessageWithKeyboard(chatID, sb.String(), keyboard)
}

// createScenarioCommand starts scenario creation wizard
func (b *Bot) createScenarioCommand(chatID int64, payload string) error {
    userID := fmt.Sprintf("%d", chatID)

    state, err := b.wizardManager.Start(userID, wizard.WizardTypeCreateScenario)
    if err != nil {
        return b.sendMessage(chatID, "Failed to start wizard: "+err.Error())
    }

    return b.sendWizardMessage(chatID, state)
}

// setDefaultScenarioCommand sets a scenario as default
func (b *Bot) setDefaultScenarioCommand(chatID int64, payload string) error {
    if payload == "" {
        return b.sendMessage(chatID, "Usage: /set_default_scenario {scenario_id}")
    }

    if err := b.scenarioService.SetDefaultScenario(payload); err != nil {
        return b.sendMessage(chatID, "Failed: "+err.Error())
    }

    return b.sendMessage(chatID, fmt.Sprintf("✅ Scenario '%s' is now the default", payload))
}

// deleteScenarioCommand deletes a scenario
func (b *Bot) deleteScenarioCommand(chatID int64, payload string) error {
    if payload == "" {
        return b.sendMessage(chatID, "Usage: /delete_scenario {scenario_id}")
    }

    if err := b.scenarioService.DeleteScenario(payload); err != nil {
        return b.sendMessage(chatID, "Failed: "+err.Error())
    }

    return b.sendMessage(chatID, fmt.Sprintf("🗑️ Scenario '%s' deleted", payload))
}

// demoScenarioCommand demonstrates a scenario with template highlighting
func (b *Bot) demoScenarioCommand(chatID int64, payload string) error {
    if payload == "" {
        return b.sendMessage(chatID, "Usage: /demo_scenario {scenario_id}")
    }

    sc, err := b.scenarioService.GetScenario(payload)
    if err != nil {
        return b.sendMessage(chatID, "Failed: "+err.Error())
    }

    // Build demo message with highlighted templates
    return b.sendScenarioDemo(chatID, sc)
}

// createBroadcastCommand starts broadcast creation wizard
func (b *Bot) createBroadcastCommand(chatID int64, payload string) error {
    // TODO: Implement broadcast wizard
    return b.sendMessage(chatID, "Broadcast creation coming soon!")
}

// sendBroadcastCommand sends a broadcast
func (b *Bot) sendBroadcastCommand(chatID int64, payload string) error {
    // TODO: Implement broadcast sending
    return b.sendMessage(chatID, "Broadcast sending coming soon!")
}

// Helper methods

func (b *Bot) isDefaultScenario(scenarioID string) bool {
    sc, err := b.scenarioService.GetDefaultScenario()
    if err != nil {
        return false
    }
    return sc.ID == scenarioID
}

func (b *Bot) buildScenarioKeyboard(scenarios []*scenario.ScenarioInfo) [][]InlineKeyboardButton {
    keyboard := make([][]InlineKeyboardButton, 0)

    for _, sc := range scenarios {
        row := []InlineKeyboardButton{
            {
                Text:         sc.Name,
                CallbackData: fmt.Sprintf("scenario_info_%s", sc.ID),
            },
        }
        keyboard = append(keyboard, row)
    }

    return keyboard
}

func (b *Bot) sendWizardMessage(chatID int64, state *wizard.WizardState) error {
    // TODO: Implement wizard message generation
    return b.sendMessage(chatID, fmt.Sprintf("Wizard step: %s", state.CurrentStep))
}

func (b *Bot) sendScenarioDemo(chatID int64, sc *scenario.FullScenario) error {
    // TODO: Implement scenario demo with template highlighting
    return b.sendMessage(chatID, fmt.Sprintf("Scenario: %s\n%s", sc.Name, sc.ID))
}
```

**Step 2: Build to verify changes**

Run: `go build ./...`
Expected: Success (may have some missing imports/fields to fix)

**Step 3: Fix any compilation errors**

Fix imports and missing fields based on compilation errors.

**Step 4: Run existing tests**

Run: `go test ./internal/telegram/... -v`
Expected: All existing tests pass

**Step 5: Commit**

```bash
git add internal/telegram/admin_handlers.go internal/telegram/admin_wizard.go
git commit -m "feat: add admin command handlers"
```

---

## Phase 11: Migration Script

### Task 11.1: Create Migration Script

**Files:**
- Create: `scripts/migrate_to_scenarios.go`
- Create: `scripts/migrate_test.go`

**Step 1: Write the migration script**

Create: `scripts/migrate_to_scenarios.go`

```go
package main

import (
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
    "time"

    "mispilka-bot-tg/internal/services"
)

func main() {
    if err := run(); err != nil {
        fmt.Fprintf(os.Stderr, "Migration failed: %v\n", err)
        os.Exit(1)
    }
    fmt.Println("Migration completed successfully!")
}

func run() error {
    // Phase 1: Backup
    fmt.Println("Phase 1: Creating backup...")
    if err := createBackup(); err != nil {
        return fmt.Errorf("backup failed: %w", err)
    }

    // Phase 2: Create directory structure
    fmt.Println("Phase 2: Creating directory structure...")
    if err := createDirectories(); err != nil {
        return fmt.Errorf("directory creation failed: %w", err)
    }

    // Phase 3: Migrate data
    fmt.Println("Phase 3: Migrating data...")
    if err := migrateData(); err != nil {
        return fmt.Errorf("data migration failed: %w", err)
    }

    // Phase 4: Verify
    fmt.Println("Phase 4: Verifying migration...")
    if err := verifyMigration(); err != nil {
        return fmt.Errorf("verification failed: %w", err)
    }

    return nil
}

func createBackup() error {
    backupDir := "data/migration_backup"
    if err := os.MkdirAll(backupDir, 0755); err != nil {
        return err
    }

    files := []string{
        "data/messages.json",
        "data/users.json",
        "data/schedule_backup.json",
    }

    for _, file := range files {
        if _, err := os.Stat(file); err == nil {
            data, err := os.ReadFile(file)
            if err != nil {
                return err
            }
            dest := filepath.Join(backupDir, filepath.Base(file))
            if err := os.WriteFile(dest, data, 0644); err != nil {
                return err
            }
            fmt.Printf("  Backed up: %s\n", file)
        }
    }

    return nil
}

func createDirectories() error {
    dirs := []string{
        "data/scenarios/default/messages",
        "data/broadcasts/messages",
        "data/buttons",
        "data/templates",
        "data/schedules",
        "data/wizards",
    }

    for _, dir := range dirs {
        if err := os.MkdirAll(dir, 0755); err != nil {
            return err
        }
        fmt.Printf("  Created: %s\n", dir)
    }

    return nil
}

func migrateData() error {
    // Migrate messages to default scenario
    if err := migrateMessages(); err != nil {
        return err
    }

    // Migrate users
    if err := migrateUsers(); err != nil {
        return err
    }

    // Create button registry with default payment button
    if err := createButtonRegistry(); err != nil {
        return err
    }

    // Create bot globals template
    if err := createBotGlobals(); err != nil {
        return err
    }

    return nil
}

func migrateMessages() error {
    // Read current messages.json
    data, err := os.ReadFile("data/messages.json")
    if err != nil {
        if os.IsNotExist(err) {
            fmt.Println("  No messages.json to migrate")
            return nil
        }
        return err
    }

    var msgs services.Messages
    if err := json.Unmarshal(data, &msgs); err != nil {
        return err
    }

    // Create scenario messages structure
    scenarioMsgs := map[string]interface{}{
        "messages_list": msgs.MessagesList,
        "messages":      msgs.Messages,
    }

    data, err = json.MarshalIndent(scenarioMsgs, "", "  ")
    if err != nil {
        return err
    }

    // Write to scenarios/default/messages.json
    if err := os.WriteFile("data/scenarios/default/messages.json", data, 0644); err != nil {
        return err
    }
    fmt.Println("  Migrated messages to scenarios/default/messages.json")

    // Copy message templates
    msgsDir := "data/messages"
    if err := filepath.Walk(msgsDir, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }
        if info.IsDir() {
            return nil
        }
        // Copy .md files
        if filepath.Ext(path) == ".md" {
            dest := filepath.Join("data/scenarios/default/messages", filepath.Base(path))
            data, err := os.ReadFile(path)
            if err != nil {
                return err
            }
            if err := os.WriteFile(dest, data, 0644); err != nil {
                return err
            }
            fmt.Printf("  Copied template: %s\n", filepath.Base(path))
        }
        return nil
    }); err != nil {
        return err
    }

    return nil
}

func migrateUsers() error {
    // Read current users.json
    data, err := os.ReadFile("data/users.json")
    if err != nil {
        if os.IsNotExist(err) {
            fmt.Println("  No users.json to migrate")
            return nil
        }
        return err
    }

    var users map[string]*services.User
    if err := json.Unmarshal(data, &users); err != nil {
        return err
    }

    // Migrate each user
    for chatID, user := range users {
        // Create scenario state from legacy fields
        scenarioState := &services.UserScenarioState{
            Status:              services.StatusActive,
            CurrentMessageIndex: 0,
            PaymentDate:         user.PaymentDate,
            PaymentLink:         user.PaymentLink,
            InviteLink:          user.InviteLink,
            JoinedGroup:         user.JoinedGroup,
            JoinedAt:            user.JoinedAt,
        }

        // Set current message index based on remaining messages
        if len(user.MessagesList) > 0 {
            scenarioState.CurrentMessageIndex = 0 // Will be calculated from sent messages
        }

        // Update user structure
        user.Scenarios = map[string]*services.UserScenarioState{
            "default": scenarioState,
        }
        user.ActiveScenarioID = "default"

        // Clear legacy fields
        user.MessagesList = nil
        user.PaymentDate = nil
        user.PaymentLink = ""
        user.InviteLink = ""
        user.JoinedGroup = false
        user.JoinedAt = nil
    }

    // Write migrated users
    data, err = json.MarshalIndent(users, "", "  ")
    if err != nil {
        return err
    }

    if err := os.WriteFile("data/users.json", data, 0644); err != nil {
        return err
    }
    fmt.Printf("  Migrated %d users\n", len(users))

    return nil
}

func createButtonRegistry() error {
    registry := map[string]interface{}{
        "button_sets": map[string]interface{}{
            "payment_button": map[string]interface{}{
                "rows": []map[string]interface{}{
                    {
                        "buttons": []map[string]interface{}{
                            {
                                "type": "url",
                                "text": "Оплатить {{scenario.product_name}}",
                                "url":  "{{user.payment_link}}",
                            },
                        },
                    },
                },
            },
        },
    }

    data, err := json.MarshalIndent(registry, "", "  ")
    if err != nil {
        return err
    }

    if err := os.WriteFile("data/buttons/registry.json", data, 0644); err != nil {
        return err
    }
    fmt.Println("  Created button registry")

    return nil
}

func createBotGlobals() error {
    globals := map[string]interface{}{
        "variables": map[string]string{
            "bot_name":     "Mispilka Bot",
            "support_link": "https://t.me/support",
        },
    }

    data, err := json.MarshalIndent(globals, "", "  ")
    if err != nil {
        return err
    }

    if err := os.WriteFile("data/templates/bot_globals.json", data, 0644); err != nil {
        return err
    }
    fmt.Println("  Created bot globals")

    return nil
}

func createScenarioRegistry() error {
    registry := map[string]interface{}{
        "scenarios": []map[string]interface{}{
            {
                "id":         "default",
                "name":       "Базовый курс",
                "created_at": time.Now().Format(time.RFC3339),
                "is_active":  true,
            },
        },
        "default_scenario_id": "default",
    }

    data, err := json.MarshalIndent(registry, "", "  ")
    if err != nil {
        return err
    }

    if err := os.WriteFile("data/scenarios/registry.json", data, 0644); err != nil {
        return err
    }
    fmt.Println("  Created scenario registry")

    return nil
}

func verifyMigration() error {
    // Check that all required files exist
    requiredFiles := []string{
        "data/scenarios/registry.json",
        "data/scenarios/default/messages.json",
        "data/buttons/registry.json",
        "data/templates/bot_globals.json",
    }

    for _, file := range requiredFiles {
        if _, err := os.Stat(file); err != nil {
            return fmt.Errorf("required file missing: %s", file)
        }
        fmt.Printf("  ✓ %s exists\n", file)
    }

    return nil
}
```

**Step 2: Build migration script**

Run: `go build scripts/migrate_to_scenarios.go`
Expected: Success

**Step 3: Commit**

```bash
git add scripts/migrate_to_scenarios.go
git commit -m "feat: add migration script for multi-scenario system"
```

---

## Phase 12: Update Handlers

### Task 12.1: Update Start Handler for Scenario Support

**Files:**
- Modify: `internal/telegram/handlers.go`

**Step 1: Update start command handler**

Update: `internal/telegram/handlers.go`

Modify the `startCommand` function to support scenarios:

```go
func (b *Bot) startCommand(chatID int64, payload string) error {
    chatIDStr := fmt.Sprintf("%d", chatID)

    // Determine scenario ID from payload or use default
    scenarioID := payload
    if scenarioID == "" {
        defaultSc, err := b.scenarioService.GetDefaultScenario()
        if err != nil {
            return b.sendMessage(chatID, "Bot is not configured. Please contact support.")
        }
        scenarioID = defaultSc.ID
    }

    // Get scenario
    sc, err := b.scenarioService.GetScenario(scenarioID)
    if err != nil {
        return b.sendMessage(chatID, fmt.Sprintf("Scenario '%s' not found. Using default.", scenarioID))
    }

    // Get or create user
    user, err := b.userService.GetUserOrCreate(chatIDStr)
    if err != nil {
        return err
    }

    // Handle scenario switching logic
    state, err := b.userScenarioService.SwitchScenario(chatIDStr, sc.Config)
    if err != nil {
        return b.sendMessage(chatID, "Failed to start scenario: "+err.Error())
    }

    // Send appropriate message based on state
    if state.IsCompleted() {
        return b.sendSummary(chatID, sc)
    }

    return b.sendScenarioWelcome(chatID, sc, user)
}
```

**Step 2: Add helper methods**

Add the following helper methods to the Bot:

```go
func (b *Bot) sendScenarioWelcome(chatID int64, sc *scenario.FullScenario, user *services.User) error {
    // Render welcome message with templates
    ctx, err := b.templateResolver.ResolveForUser(sc.Config, nil, user)
    if err != nil {
        return err
    }

    // Load and render template
    templateContent, err := b.loadTemplate("data/scenarios/default/messages/start.md")
    if err != nil {
        // Use default welcome message
        templateContent = "Welcome! Please accept the terms to continue."
    }

    message, err := b.templateRenderer.Render(templateContent, ctx)
    if err != nil {
        return err
    }

    // Build keyboard with accept button
    keyboard := [][]InlineKeyboardButton{
        {
            {
                Text:         "✅ Accept terms",
                CallbackData: "accept_terms",
            },
        },
    }

    return b.sendMessageWithKeyboard(chatID, message, keyboard)
}

func (b *Bot) sendSummary(chatID int64, sc *scenario.FullScenario) error {
    // Load summary template
    templatePath := filepath.Join("data/scenarios", sc.ID, sc.Summary.TemplateFile)
    templateContent, err := os.ReadFile(templatePath)
    if err != nil {
        return err
    }

    // TODO: Render and send summary message
    return b.sendMessage(chatID, string(templateContent))
}

func (b *Bot) loadTemplate(path string) (string, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return "", err
    }
    return string(data), nil
}
```

**Step 3: Build to verify changes**

Run: `go build ./...`
Expected: Success

**Step 4: Run existing tests**

Run: `go test ./internal/telegram/... -v`
Expected: All existing tests pass

**Step 5: Commit**

```bash
git add internal/telegram/handlers.go
git commit -m "feat: update start handler for scenario support"
```

---

## Phase 13: Testing & Verification

### Task 13.1: End-to-End Integration Test

**Files:**
- Create: `tests/integration/scenario_test.go`

**Step 1: Write integration test**

Create: `tests/integration/scenario_test.go`

```go
package integration

import (
    "os"
    "path/filepath"
    "testing"
    "time"

    "mispilka-bot-tg/internal/domain/scenario"
    "mispilka-bot-tg/internal/services/scenario"
    "mispilka-bot-tg/internal/services/scenario_user"
)

func TestScenarioLifecycle(t *testing.T) {
    tmpDir := t.TempDir()

    // Setup
    scenariosDir := filepath.Join(tmpDir, "scenarios")
    scenarioService := scenario.NewService(scenariosDir)
    _ = scenarioService.Initialize()

    userScenarioService := scenario_user.NewService()
    _ = userScenarioService.Initialize()

    // Step 1: Create scenario
    t.Run("CreateScenario", func(t *testing.T) {
        sc, err := scenarioService.CreateScenario("Test Scenario", &scenario.ProdamusConfig{
            ProductName:    "Test Product",
            ProductPrice:   "1000",
            PaidContent:    "Thank you!",
            PrivateGroupID: "-1001234567890",
        })
        if err != nil {
            t.Fatalf("Failed to create scenario: %v", err)
        }
        if sc.ID == "" {
            t.Error("Expected scenario ID")
        }
    })

    // Step 2: Start scenario for user
    t.Run("StartScenario", func(t *testing.T) {
        sc, _ := scenarioService.GetDefaultScenario()
        chatID := "123456"

        state, err := userScenarioService.StartScenario(chatID, sc.Config)
        if err != nil {
            t.Fatalf("Failed to start scenario: %v", err)
        }
        if state.Status != scenario.StatusActive {
            t.Errorf("Expected active status, got %s", state.Status)
        }
    })

    // Step 3: Switch scenario
    t.Run("SwitchScenario", func(t *testing.T) {
        sc2, _ := scenarioService.CreateScenario("Scenario 2", &scenario.ProdamusConfig{
            ProductName:    "Product 2",
            ProductPrice:   "2000",
            PaidContent:    "Thanks!",
            PrivateGroupID: "-1001234567890",
        })

        chatID := "123456"
        state, err := userScenarioService.SwitchScenario(chatID, sc2.Config)
        if err != nil {
            t.Fatalf("Failed to switch scenario: %v", err)
        }
        if state.Status != scenario.StatusActive {
            t.Errorf("Expected active status, got %s", state.Status)
        }
    })
}
```

**Step 2: Run integration test**

Run: `go test ./tests/integration/... -v`
Expected: PASS

**Step 3: Commit**

```bash
git add tests/integration/scenario_test.go
git commit -m "test: add scenario integration tests"
```

---

## Summary

This implementation plan breaks down the multi-scenario messaging system into 57 bite-sized tasks across 13 phases. Each task follows the TDD approach:

1. Write the failing test
2. Run test to verify it fails
3. Write minimal implementation
4. Run test to verify it passes
5. Commit

### Key Phases Summary

| Phase | Tasks | Description |
|-------|-------|-------------|
| 0 | 1 | Prerequisites & Testing Setup |
| 1 | 4 | Core Domain Types |
| 2 | 4 | Storage Layer |
| 3 | 3 | Template System |
| 4 | 2 | Scenario Services |
| 5 | 1 | User Scenario Management |
| 6 | 1 | Scenario Scheduler |
| 7 | 2 | Broadcast System |
| 8 | 1 | Button Registry |
| 9 | 1 | Wizard System |
| 10 | 1 | Admin Commands |
| 11 | 1 | Migration Script |
| 12 | 1 | Update Handlers |
| 13 | 1 | Testing & Verification |

### File Structure After Implementation

```
internal/
├── domain/
│   ├── scenario/
│   │   ├── types.go
│   │   ├── status.go
│   │   └── user_state.go
│   ├── registry/
│   │   └── types.go
│   ├── button/
│   │   └── types.go
│   └── broadcast/
│       └── types.go
├── services/
│   ├── scenario/
│   │   ├── registry.go
│   │   ├── config.go
│   │   ├── messages.go
│   │   ├── summary.go
│   │   └── service.go
│   ├── scenario_user/
│   │   └── service.go
│   ├── scenario_scheduler/
│   │   └── service.go
│   ├── broadcast/
│   │   ├── registry.go
│   │   └── sender.go
│   ├── button/
│   │   └── registry.go
│   ├── template/
│   │   ├── types.go
│   │   ├── renderer.go
│   │   └── resolver.go
│   ├── wizard/
│   │   ├── types.go
│   │   └── manager.go
│   └── users.go (updated)
├── telegram/
│   ├── handlers.go (updated)
│   ├── admin_handlers.go
│   └── admin_wizard.go
scripts/
└── migrate_to_scenarios.go
```

### Data Structure After Implementation

```
data/
├── scenarios/
│   ├── registry.json
│   ├── default/
│   │   ├── config.json
│   │   ├── messages.json
│   │   ├── summary.json
│   │   └── messages/
│   └── {scenario_id}/
├── broadcasts/
│   ├── registry.json
│   └── messages/
├── buttons/
│   └── registry.json
├── templates/
│   └── bot_globals.json
├── schedules/
│   └── {user_id}.json
├── wizards/
│   └── {user_id}.json
├── migration_backup/
└── users.json (updated)
```
