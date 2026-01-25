# Wizard Manual Testing Checklist

## Prerequisites
- Bot is running locally
- You have admin privileges
- Clean data directory (or test scenario ID)

## Test Cases

### 1. Basic Scenario Creation

#### 1.1 Valid Flow
- [ ] Send `/create_scenario`
- [ ] Enter valid scenario name: "Test Scenario"
- [ ] Enter valid product name: "Premium Course"
- [ ] Enter valid price: "500"
- [ ] Enter valid content: "Course materials"
- [ ] Enter valid group ID: "-1001234567890"
- [ ] Verify confirmation message shows all data correctly
- [ ] Click "Confirm"
- [ ] Verify "Scenario created successfully" message
- [ ] Run `/scenarios` - verify scenario appears in list

#### 1.2 Field Validation
- [ ] Start new scenario creation
- [ ] Try empty name - verify error message
- [ ] Try invalid price "abc" - verify error message
- [ ] Try negative price "-100" - verify error message
- [ ] Try invalid group ID "12345" - verify error message (missing -100)
- [ ] Try invalid group ID "-100abc" - verify error message

#### 1.3 Edit Flow
- [ ] Complete all general info fields
- [ ] At confirmation, click "Edit"
- [ ] Verify field selection buttons appear
- [ ] Click "Price" button
- [ ] Enter new price: "1000"
- [ ] Verify return to confirmation with new price shown
- [ ] Click "Confirm" - verify scenario created with new price

#### 1.4 Cancel Flow
- [ ] Start scenario creation
- [ ] Enter some data
- [ ] At confirmation, click "Cancel"
- [ ] Verify cancellation message
- [ ] Verify wizard is cancelled (can't resume)

### 2. Summary Configuration

#### 2.1 Summary with Text Only
- [ ] After general confirmation, enter summary text
- [ ] Click "Confirm" on summary
- [ ] Verify prompt for first message appears

#### 2.2 Summary with Photos
- [ ] Enter summary text
- [ ] Send photo when prompted
- [ ] Click "Confirm" - verify photo count shown

#### 2.3 Summary Edit Flow
- [ ] Configure summary
- [ ] Click "Edit" on summary confirmation
- [ ] Select "Message Text"
- [ ] Enter new text
- [ ] Verify return to summary confirmation

### 3. Message Creation

#### 3.1 Single Message
- [ ] After summary, enter message text
- [ ] Enter timing: "0h 0m"
- [ ] Type "skip" for photos
- [ ] Type "skip" for buttons
- [ ] Verify message confirmation
- [ ] Click "Confirm"
- [ ] Click "Finish Scenario"
- [ ] Verify scenario created

#### 3.2 Multiple Messages
- [ ] Create first message (timing: 0h 0m)
- [ ] Click "Add Another Message"
- [ ] Create second message (timing: 1h 0m)
- [ ] Click "Add Another Message"
- [ ] Create third message (timing: 2h 30m)
- [ ] Click "Finish Scenario"
- [ ] Verify all 3 messages exist

#### 3.3 Message Edit Flow
- [ ] Create message
- [ ] Click "Edit" on message confirmation
- [ ] Select "Timing"
- [ ] Enter new timing: "30m"
- [ ] Verify return to message confirmation with new timing

#### 3.4 Message with Photos
- [ ] Enter message text
- [ ] Send 2 photos
- [ ] Enter timing
- [ ] Skip buttons
- [ ] Verify confirmation shows 2 photos

#### 3.5 Message with Buttons
- [ ] Enter message text
- [ ] Skip photos
- [ ] Enter timing
- [ ] Enter button: `url|Buy Now|https://example.com|buy`
- [ ] Verify confirmation shows button

### 4. Edge Cases

#### 4.1 Long Names
- [ ] Try 201 character name - verify error
- [ ] Try 200 character name - verify success

#### 4.2 Maximum Price
- [ ] Try price "1000001" - verify error
- [ ] Try price "1000000" - verify success

#### 4.3 Special Characters in Name
- [ ] Use emoji in name: "Test 🚀 Scenario"
- [ ] Use Russian characters: "Тестовый сценарий"
- [ ] Verify both work correctly

#### 4.4 Wizard Timeout
- [ ] Start scenario creation
- [ ] Wait 31 minutes
- [ ] Try to send data - verify wizard expired message

#### 4.5 Multiple Scenarios
- [ ] Create scenario "Scenario A"
- [ ] Create scenario "Scenario B"
- [ ] Create scenario "Scenario C"
- [ ] Run `/scenarios` - verify all appear

## Success Criteria
- All validation errors show user-friendly messages
- Edit flow returns to correct confirmation step
- All created scenarios have valid data
- No data loss during edit operations
- Wizard expires after timeout
