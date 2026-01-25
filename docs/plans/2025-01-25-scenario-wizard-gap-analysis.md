# Scenario Wizard Gap Analysis

**Date:** 2025-01-25
**Purpose:** Analyze gaps between design documents and current implementation

## Executive Summary

The current scenario wizard implementation provides a basic flow for creating scenario configurations but is missing critical features defined in the design documents:

1. **No edit functionality** - Cannot edit scenario fields after confirmation
2. **No message creation flow** - Cannot add messages to scenarios after creation
3. **No summary message configuration** - Summary steps defined but unused
4. **No field validation** - Price format, group ID format, length checks missing
5. **Confirmation is single-stage** - Design calls for multi-stage confirmations with edit option

---

## Detailed Gap Analysis

### 1. Confirmation Flow

| Feature | Design Spec | Current Implementation | Status |
|---------|-------------|----------------------|--------|
| Confirmation with edit button | ✅ Required | ❌ Missing | **NOT IMPLEMENTED** |
| Edit specific field step | ✅ Required | ❌ Missing | **NOT IMPLEMENTED** |
| Return to confirmation after edit | ✅ Required | ❌ Missing | **NOT IMPLEMENTED** |

**Current Flow:**
```
StepScenarioName → StepProductName → StepProductPrice → StepPaidContent → StepPrivateGroupID → StepConfirmGeneral → (Confirm/Cancel) → DONE
```

**Required Flow (from design):**
```
StepScenarioName → StepProductName → StepProductPrice → StepPaidContent → StepPrivateGroupID
  → StepConfirmGeneral (Confirm/Edit buttons)
  → If Edit: StepEditGeneral (select field) → edit field → back to StepConfirmGeneral
  → If Confirm: continue to next section
```

### 2. Message Creation Flow

| Feature | Design Spec | Current Implementation | Status |
|---------|-------------|----------------------|--------|
| Add messages after scenario creation | ✅ Required | ❌ Missing | **NOT IMPLEMENTED** |
| Message text template input | ✅ Required | ❌ Missing | **NOT IMPLEMENTED** |
| Photo upload for messages | ✅ Required | ❌ Missing | **NOT IMPLEMENTED** |
| Message timing (hours/minutes) | ✅ Required | ❌ Missing | **NOT IMPLEMENTED** |
| Inline keyboard configuration | ✅ Required | ❌ Missing | **NOT IMPLEMENTED** |
| Message confirmation with edit | ✅ Required | ❌ Missing | **NOT IMPLEMENTED** |
| Add multiple messages | ✅ Required | ❌ Missing | **NOT IMPLEMENTED** |

**Required Flow (from design):**
```
After scenario config confirmed:
→ StepSummaryMessage → StepSummaryPhotos → StepSummaryButtons → StepConfirmSummary
→ If Edit: StepEditSummary → back to confirmation
→ If Confirm: → StepMessageText (first message)
→ StepMessagePhotos → StepMessageTiming → StepMessageButtons → StepConfirmMessage
→ If Edit: StepEditMessage → back to message confirmation
→ If Confirm: → StepAddMoreMessages
→ If yes: back to StepMessageText for next message
→ If no: wizard complete
```

**Current Reality:**
- Scenario created with EMPTY message list
- No way to add messages through wizard
- No command to add/edit messages separately

### 3. Field Validation

| Field | Design Requirement | Current Validation | Status |
|-------|-------------------|-------------------|--------|
| Product price | Numeric, positive integer | ❌ Only checks non-empty | **INSUFFICIENT** |
| Private group ID | Valid Telegram group ID format | ❌ Only checks non-empty | **MISSING** |
| Scenario name | Max length, unique | ❌ Only checks non-empty | **INSUFFICIENT** |
| Product name | Max length | ❌ Only checks non-empty | **MISSING** |
| Paid content | Max length | ❌ Only checks non-empty | **MISSING** |

**Current validation in `internal/services/scenario/service.go:78-92`:**
```go
if req.ID == "" { return nil, domainScenario.ErrInvalidScenarioID }
if req.Name == "" { return nil, domainScenario.ErrInvalidScenarioName }
if req.Prodamus.ProductName == "" { return nil, domainScenario.ErrInvalidProductName }
if req.Prodamus.ProductPrice == "" { return nil, domainScenario.ErrInvalidProductPrice }
if req.Prodamus.PrivateGroupID == "" { return nil, domainScenario.ErrInvalidPrivateGroupID }
```

**Missing:**
- Price format: `^\d+$` (must be digits only)
- Group ID format: `^-\d{10}$` (Telegram private group ID format)
- Field length limits (prevent abuse)
- Duplicate ID check (partially done via service layer)

### 4. Wizard State Management

| Feature | Design Spec | Current Implementation | Status |
|---------|-------------|----------------------|--------|
| Edit mode tracking | ✅ Required | ❌ Missing | **NOT IMPLEMENTED** |
| Section tracking (general/summary/messages) | ✅ Required | ❌ Missing | **NOT IMPLEMENTED** |
| Message index tracking | ✅ Required | ❌ Missing | **NOT IMPLEMENTED** |

**Current `WizardState` in `internal/services/wizard/types.go`:**
```go
type WizardState struct {
    UserID      string                 `json:"user_id"`
    WizardType  WizardType             `json:"wizard_type"`
    CurrentStep WizardStep             `json:"current_step"`
    StartedAt   time.Time              `json:"started_at"`
    Timeout     time.Duration          `json:"timeout"`
    Data        map[string]interface{} `json:"data"`
}
```

**Missing fields needed for edit flow:**
- `EditMode bool` - Whether we're in edit mode
- `EditTargetStep WizardStep` - Which step we're editing
- `CurrentSection string` - "general", "summary", or "messages"
- `CurrentMessageIndex int` - Which message we're editing/creating
- `MessagesCreated int` - Count of messages added

### 5. Callback Handlers

| Callback | Design Spec | Current Implementation | Status |
|----------|-------------|----------------------|--------|
| `wizard_confirm_scenario` | ✅ Implemented | ✅ Implemented | WORKS |
| `wizard_cancel` | ✅ Implemented | ✅ Implemented | WORKS |
| `wizard_edit_general` | ✅ Required | ❌ Missing | **NOT IMPLEMENTED** |
| `wizard_edit_{field}` | ✅ Required | ❌ Missing | **NOT IMPLEMENTED** |
| `wizard_confirm_summary` | ✅ Required | ❌ Missing | **NOT IMPLEMENTED** |
| `wizard_edit_summary` | ✅ Required | ❌ Missing | **NOT IMPLEMENTED** |
| `wizard_confirm_message` | ✅ Required | ❌ Missing | **NOT IMPLEMENTED** |
| `wizard_edit_message` | ✅ Required | ❌ Missing | **NOT IMPLEMENTED** |
| `wizard_add_more_messages` | ✅ Required | ❌ Missing | **NOT IMPLEMENTED** |

**Current callbacks in `internal/telegram/bot.go:169-189`:**
```go
switch callback.Data {
case "accept":
    b.acceptCallback(callback)
case "wizard_confirm_scenario":
    b.handleWizardConfirm(callback)
case "wizard_cancel":
    b.handleWizardCancel(callback)
// ... many missing handlers
}
```

### 6. Data Storage

| Component | Design Spec | Current Implementation | Status |
|-----------|-------------|----------------------|--------|
| Scenario registry | ✅ JSON in `data/scenarios/registry.json` | ✅ Implemented | WORKS |
| Scenario config | ✅ JSON in `data/scenarios/{id}/config.json` | ✅ Implemented | WORKS |
| Scenario messages | ✅ JSON in `data/scenarios/{id}/messages.json` | ✅ Implemented | **EMPTY** |
| Scenario summary | ✅ JSON in `data/scenarios/{id}/summary.json` | ✅ Implemented | **MISSING** |
| Button registry | ✅ JSON in `data/buttons/registry.json` | ✅ Implemented | UNUSED |

**Issue:** Message storage structure exists but is never populated through the wizard.

---

## Critical Missing Features Summary

### Priority 1: Edit Functionality (BLOCKS ALL USAGE)
- No way to correct mistakes during scenario creation
- Must have: Confirm → Edit → Confirm cycle for each section

### Priority 2: Message Creation (CORE FEATURE)
- Scenarios without messages are useless
- Must have: Full message creation flow with templates, photos, timing, keyboards

### Priority 3: Field Validation (DATA QUALITY)
- Invalid data can be saved (e.g., "abc" as price)
- Must have: Format validation before saving

### Priority 4: Summary Configuration (IMPORTANT)
- Summary message is a key part of the scenario
- Must have: Summary text, photos, keyboard configuration

---

## Implementation Strategy

Given the scope of missing features, the recommended approach is:

1. **Phase 1: Enhanced Validation** - Add validation to prevent bad data
2. **Phase 2: Edit Flow** - Add edit functionality to general config
3. **Phase 3: Summary Configuration** - Add summary message setup
4. **Phase 4: Message Creation** - Full message creation wizard flow
5. **Phase 5: Message Edit Flow** - Edit existing messages

This allows incremental testing and prevents a single massive change.

---

## Files Requiring Changes

### Existing Files to Modify:
1. `internal/services/wizard/types.go` - Add edit mode state tracking
2. `internal/services/scenario/service.go` - Add validation methods
3. `internal/services/scenario/validator.go` - **CREATE NEW** - Centralized validation
4. `internal/telegram/bot.go` - Add new callback handlers
5. `internal/telegram/admin_handlers.go` - Add new step prompts
6. `internal/telegram/wizard_handlers.go` - **CREATE NEW** - Separate wizard handlers

### New Files to Create:
1. `internal/services/scenario/validator.go` - Field validation logic
2. `internal/services/scenario/message.go` - Message creation service methods
3. `internal/telegram/wizard_handlers.go` - All wizard-related handlers
4. `internal/telegram/wizard_prompts.go` - Wizard prompt templates
5. `internal/services/storage/message.go` - Message file operations
6. `internal/services/storage/summary.go` - Summary file operations
7. `internal/services/storage/button.go` - Button registry operations

---

## Test Coverage Needs

Currently there are NO tests for wizard flows. Needed tests:
1. Validation tests (`internal/services/scenario/validator_test.go`)
2. Wizard state transition tests
3. Callback handler tests
4. End-to-end wizard flow tests

---

## Next Steps

This analysis feeds into the new implementation plan: `2025-01-25-scenario-wizard-redesign.md`

The implementation plan will:
1. Address all identified gaps
2. Follow TDD methodology
3. Provide exact code for each step
4. Enable incremental progress with frequent commits
