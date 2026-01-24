# Multi-Scenario Messaging System Design

**Date:** 2024-01-24
**Author:** Design generated via brainstorming session
**Status:** Approved

## Overview

This document describes the design for a multi-scenario messaging system that allows the Telegram bot to support multiple independent message flows (scenarios) instead of a single global message queue. Each scenario has its own configuration, message flow, Prodamus payment settings, and summary message.

## Table of Contents

1. [Requirements](#requirements)
2. [Architectural Approach](#architectural-approach)
3. [Data Structures](#data-structures)
4. [Component Design](#component-design)
5. [Admin Commands & Handlers](#admin-commands--handlers)
6. [Data Flow & State Management](#data-flow--state-management)
7. [Template System](#template-system)
8. [Error Handling](#error-handling)
9. [Migration Plan](#migration-plan)

---

## Requirements

### Functional Requirements

1. **Multiple Scenarios**: Admin can create multiple independent scenarios, each with:
   - Unique ID and name
   - Custom message flow
   - Prodamus payment configuration (product_name, product_price, paid_content, private_group_id)
   - Reserved summary message (not in message queue)

2. **Default Scenario**: First scenario becomes default; admin can change which scenario is default

3. **Scenario Activation**:
   - `/start` without parameters → activates default scenario (if not already started)
   - `/start {scenario_id}` → activates specific scenario

4. **User Scenario Constraints**: User can only have 1 active scenario at a time

5. **Scenario Switching Logic**:
   - If scenario completed → send summary only
   - If scenario not current and not completed → stop current, start new from last position
   - If scenario is current and not completed → resend last message
   - If scenario not started → start from beginning

6. **Broadcast Messages**: Non-scenario messages admin can send with targeting conditions:
   - "No active scenario"
   - "Has not paid any product"

7. **Template System**: Three-level scope:
   - `{{bot.variable}}` - Global bot settings
   - `{{scenario.variable}}` - Scenario-specific
   - `{{user.variable}}` - User-specific

8. **Admin Features**:
   - Conversational wizard for scenario creation
   - List scenarios as buttons
   - Demonstrate scenario with template highlighting
   - Set default scenario

### Non-Functional Requirements

- JSON file storage (no database)
- Atomic file operations with retry logic
- Graceful handling of concurrent access
- Migration path from existing single-scenario system

---

## Architectural Approach

### Chosen Approach: Scenario-Centric

Each scenario is a self-contained entity with its own configuration, message flow, and payment settings. Data is organized around scenarios first.

**Rationale:**
- Clear separation - each scenario is isolated
- Easy to add/remove scenarios without affecting others
- Natural migration path from current structure
- Simple to understand and maintain

---

## Data Structures

### File Structure

```
data/
├── scenarios/
│   ├── registry.json              # All scenarios with metadata
│   ├── default/                   # Default scenario (migrated)
│   │   ├── config.json            # Scenario config (Prodamus settings)
│   │   ├── messages.json          # Message flow for this scenario
│   │   ├── summary.json           # Reserved summary message
│   │   └── messages/              # Template files
│   │       ├── start.md
│   │       └── *.md
│   └── {scenario_id}/             # Additional scenarios
│       └── ...
├── broadcasts/
│   ├── registry.json              # Broadcast registry
│   └── messages/                  # Broadcast templates
│       └── *.md
├── buttons/
│   └── registry.json              # Shared button sets
├── templates/
│   └── bot_globals.json           # Bot-level template variables
├── schedules/
│   └── {user_id}.json             # Per-user schedule state
├── wizards/
│   └── {user_id}.json             # Active wizard states (temp)
└── users.json                     # Updated user structure
```

### Core Data Structures

#### Scenario Registry

```json
{
  "scenarios": [
    {
      "id": "default",
      "name": "Базовый курс",
      "created_at": "2024-01-15T10:00:00Z",
      "is_active": true
    }
  ],
  "default_scenario_id": "default"
}
```

#### Scenario Config

```json
{
  "id": "premium",
  "name": "Премиум курс",
  "created_at": "2024-02-01T10:00:00Z",
  "prodamus": {
    "product_name": "Премиум доступ",
    "product_price": "1500",
    "paid_content": "Спасибо за оплату премиум!",
    "private_group_id": "-1001234567890"
  }
}
```

#### Scenario Messages

```json
{
  "messages_list": ["msg_1", "msg_2", "msg_3"],
  "messages": {
    "msg_1": {
      "timing": {"hours": 0, "minutes": 0},
      "template_file": "msg_1.md",
      "photos": ["msg_1_1.PNG", "msg_1_2.PNG"],
      "inline_keyboard": {
        "button_set_ref": "payment_button"
      }
    }
  }
}
```

#### Scenario Summary

```json
{
  "template_file": "summary.md",
  "photos": ["summary_1.PNG", "summary_2.PNG"],
  "inline_keyboard": {
    "button_set_ref": "payment_button"
  }
}
```

#### Button Registry

```json
{
  "button_sets": {
    "payment_button": {
      "rows": [{
        "buttons": [{
          "type": "url",
          "text": "Оплатить {{scenario.product_name}}",
          "url": "{{user.payment_link}}"
        }]
      }]
    }
  }
}
```

#### Bot Template Globals

```json
{
  "variables": {
    "bot_name": "Mispilka Bot",
    "support_link": "https://t.me/support"
  }
}
```

#### Broadcast Registry

```json
{
  "broadcasts": [
    {
      "id": "bc_1",
      "name": "Объявление",
      "template_file": "bc_1.md",
      "photos": ["bc_1_1.PNG"],
      "inline_keyboard": {
        "rows": [{
          "buttons": [{
            "type": "url",
            "text": "Подробнее",
            "url": "https://example.com"
          }]
        }]
      },
      "targeting": {
        "conditions": ["no_active_scenario", "has_not_paid"]
      },
      "created_at": "2024-02-01T10:00:00Z"
    }
  ]
}
```

#### Updated User Structure

```json
{
  "123456": {
    "user_name": "john_doe",
    "reg_time": "2024-01-15T10:00:00Z",
    "scenarios": {
      "default": {
        "status": "active",
        "current_message_index": 2,
        "last_sent_at": "2024-01-15T12:00:00Z",
        "completed_at": null,
        "payment_date": "2024-01-15T11:00:00Z",
        "payment_link": "https://pay...",
        "invite_link": "https://t.me/+abc",
        "joined_group": true,
        "joined_at": "2024-01-15T11:30:00Z"
      }
    },
    "active_scenario_id": "default"
  }
}
```

**User scenario statuses:** `not_started`, `active`, `completed`, `stopped`

---

## Component Design

### Services Structure

```
internal/services/
├── scenario/
│   ├── registry.go          # Scenario CRUD operations
│   ├── config.go            # Scenario configuration management
│   └── messages.go          # Scenario message flow management
├── broadcast/
│   ├── registry.go          # Broadcast CRUD operations
│   └── sender.go            # Broadcast sending with targeting
├── template/
│   ├── engine.go            # Template rendering with scopes
│   └── resolver.go          # Variable resolution (bot/scenario/user)
├── button/
│   └── registry.go          # Shared button set management
├── scheduler/
│   └── scenario.go          # Per-scenario message scheduling
└── user/
    └── scenario.go          # User-scenario state management
```

### Key Service Interfaces

```go
// Scenario Registry
type ScenarioRegistry struct {
    scenarios map[string]*Scenario
    defaultID string
    mu        sync.RWMutex
}

type Scenario struct {
    ID        string
    Name      string
    CreatedAt time.Time
    IsActive  bool
    Config    ScenarioConfig
    Messages  ScenarioMessages
    Summary   ScenarioSummary
}

func (r *ScenarioRegistry) Get(id string) (*Scenario, error)
func (r *ScenarioRegistry) Create(name string) (*Scenario, error)
func (r *ScenarioRegistry) List() []*Scenario
func (r *ScenarioRegistry) SetDefault(id string) error
func (r *ScenarioRegistry) GetDefault() (*Scenario, error)
```

```go
// User Scenario State
type UserScenarioState struct {
    Status              ScenarioStatus
    CurrentMessageIndex int
    LastSentAt          *time.Time
    CompletedAt         *time.Time
    PaymentDate         *time.Time
    PaymentLink         string
    InviteLink          string
    JoinedGroup         bool
    JoinedAt            *time.Time
}

type ScenarioStatus string

const (
    StatusNotStarted ScenarioStatus = "not_started"
    StatusActive     ScenarioStatus = "active"
    StatusCompleted  ScenarioStatus = "completed"
    StatusStopped    ScenarioStatus = "stopped"
)

func GetUserScenario(chatID, scenarioID string) (*UserScenarioState, error)
func SetUserScenario(chatID, scenarioID string, state *UserScenarioState) error
func GetUserActiveScenario(chatID string) (scenarioID string, state *UserScenarioState, err error)
func SwitchUserScenario(chatID, newScenarioID string) error
func IsScenarioCompleted(chatID, scenarioID string) (bool, error)
```

---

## Admin Commands & Handlers

### Commands

- `/scenarios` - Show all scenarios as buttons
- `/create_scenario` - Start scenario creation wizard
- `/edit_scenario {id}` - Edit existing scenario
- `/set_default_scenario {id}` - Set scenario as default
- `/delete_scenario {id}` - Delete scenario (if not default)
- `/demo_scenario {id}` - Demonstrate scenario with template highlighting
- `/create_broadcast` - Start broadcast creation wizard
- `/send_broadcast {id}` - Send broadcast with targeting options

### Wizard Steps

```go
type WizardStep string

const (
    // General scenario info steps
    StepScenarioName     WizardStep = "scenario_name"
    StepProductName      WizardStep = "product_name"
    StepProductPrice     WizardStep = "product_price"
    StepPaidContent      WizardStep = "paid_content"
    StepPrivateGroupID   WizardStep = "private_group_id"
    StepConfirmGeneral   WizardStep = "confirm_general"
    StepEditGeneral      WizardStep = "edit_general"

    // Summary steps
    StepSummaryMessage   WizardStep = "summary_message"
    StepSummaryPhotos    WizardStep = "summary_photos"
    StepSummaryButtons   WizardStep = "summary_buttons"
    StepConfirmSummary   WizardStep = "confirm_summary"
    StepEditSummary      WizardStep = "edit_summary"

    // Message steps
    StepMessageText      WizardStep = "message_text"
    StepMessagePhotos    WizardStep = "message_photos"
    StepMessageTiming    WizardStep = "message_timing"
    StepMessageButtons   WizardStep = "message_buttons"
    StepConfirmMessage   WizardStep = "confirm_message"
    StepEditMessage      WizardStep = "edit_message"

    // Flow control
    StepAddMoreMessages  WizardStep = "add_more_messages"
)
```

### Confirmation Flow

Each section (general info, summary, message) has a confirmation step:

1. Collect all data for the section
2. Show summary with all entered values
3. Present buttons: "✅ Подтвердить" and "✏️ Редактировать"
4. If "Редактировать" clicked → show field selection → edit → return to confirmation
5. If "Подтвердить" clicked → proceed to next section

---

## Data Flow & State Management

### User Scenario Lifecycle

```
User sends /start
        ↓
Determine scenario ID (query param or default)
        ↓
Get/create user state for scenario
        ↓
┌─────────────────────────────────────────────┐
│ Scenario State Decision                     │
│                                             │
│ COMPLETED? → Send summary only             │
│ NOT CURRENT & NOT COMPLETED? → Switch       │
│ CURRENT & NOT COMPLETED? → Resend last     │
│ NOT STARTED? → Start from beginning         │
└─────────────────────────────────────────────┘
        ↓
Schedule first/next message
```

### Scenario Switching Logic

```go
func SwitchUserScenario(chatID, newScenarioID string) error {
    user, _ := GetUser(chatID)
    userState := getUserState(user, newScenarioID)

    switch {
    case userState.Status == StatusCompleted:
        return handleCompletedScenario(chatID, scenario, userState)

    case user.ActiveScenarioID != "" && user.ActiveScenarioID != newScenarioID:
        return handleScenarioSwitch(chatID, user, newScenarioID, userState)

    case user.ActiveScenarioID == newScenarioID && userState.Status == StatusActive:
        return handleResendLastMessage(chatID, scenario, userState)

    default:
        return handleStartScenario(chatID, scenario, userState)
    }
}
```

---

## Template System

### Syntax

`{{scope.variable_name}}`

**Scopes:**
- `bot` - Global bot-level variables
- `scenario` - Scenario-specific variables
- `user` - User-specific variables

### Template Variables

**Bot globals** (`data/templates/bot_globals.json`):
- `bot_name`
- `support_link`
- `company_name`

**Scenario auto-generated**:
- `scenario.id`
- `scenario.name`
- `scenario.product_name`
- `scenario.product_price`
- `scenario.private_group_id`

**User auto-generated**:
- `user.user_name`
- `user.payment_link`
- `user.invite_link`
- `user.reg_date`
- `user.payment_date`
- `user.joined_date`

### Example Template

```markdown
Привет! Курс "{{scenario.product_name}}" стоит {{scenario.product_price}}₽

Твоя ссылка: {{user.payment_link}}
Поддержка: {{bot.support_link}}
```

### Template Highlighting for Demo

For demonstration mode, templates are highlighted:
```
__scenario.product_name__
```

---

## Error Handling

### Error Codes

```go
const (
    ErrScenarioNotFound      = "SCENARIO_NOT_FOUND"
    ErrScenarioAlreadyActive = "SCENARIO_ALREADY_ACTIVE"
    ErrInvalidScenarioState  = "INVALID_SCENARIO_STATE"
    ErrMessageNotFound       = "MESSAGE_NOT_FOUND"
    ErrTemplateRenderFailed  = "TEMPLATE_RENDER_FAILED"
    ErrScheduleFailed        = "SCHEDULE_FAILED"
    ErrUserNotFound          = "USER_NOT_FOUND"
    ErrPaymentRequired       = "PAYMENT_REQUIRED"
)
```

### Key Error Scenarios

1. **Scenario not found** → Return error, use default
2. **Non-existent scenario in /start** → Notify user, use default
3. **Payment without active scenario** → Error notification
4. **Concurrent access** → RWMutex protection
5. **Wizard timeout** → 30-minute expiration
6. **Bot restart** → Restore schedules from `data/schedules/`

---

## Migration Plan

### Phase 1: Backup

```bash
mkdir -p data/migration_backup
cp data/messages.json data/migration_backup/
cp data/users.json data/migration_backup/
cp data/schedule_backup.json data/migration_backup/
```

### Phase 2: Create Directory Structure

```bash
mkdir -p data/scenarios/default/messages
mkdir -p data/broadcasts/messages
mkdir -p data/buttons
mkdir -p data/templates
mkdir -p data/schedules
mkdir -p data/wizards
```

### Phase 3: Migrate Data

1. Read current `messages.json`
2. Create "default" scenario with migrated data
3. Copy message templates to `data/scenarios/default/messages/`
4. Create scenario registry with default scenario
5. Migrate users to new structure
6. Create button registry with default payment button
7. Create bot globals template

### Phase 4: Update Config

**Remove from `.env`:**
- `PRODAMUS_PRODUCT_NAME`
- `PRODAMUS_PRODUCT_PRICE`
- `PRODAMUS_PRODUCT_PAID_CONTENT`
- `PRIVATE_GROUP_ID`

These are now per-scenario in `data/scenarios/{id}/config.json`

### Phase 5: Verification

- Test bot startup
- Verify existing users still work
- Test scenario switching
- Verify payment flow

---

## Summary

### Key Changes

| Area | Before | After |
|------|--------|-------|
| Scenarios | Single global flow | Multiple independent scenarios |
| Prodamus config | Environment variables | Per-scenario config |
| User state | Single `messages_list` | Per-scenario state |
| Templates | `{{variable}}` | `{{scope.variable}}` |
| Messages | Single messages.json | Per-scenario messages.json |
| Buttons | Inline in messages | Shared button registry |
| Broadcasts | Not supported | Full broadcast system |
| Admin management | Commands only | Wizard-based creation |

### File Structure Summary

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
└── users.json
```
