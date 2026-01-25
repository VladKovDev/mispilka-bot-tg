package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"mispilkabot/internal/services"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <source_dir>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Example: %s data/test\n", os.Args[0])
		os.Exit(1)
	}

	sourceDir := os.Args[1]

	if err := run(sourceDir); err != nil {
		fmt.Fprintf(os.Stderr, "Migration failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Migration completed successfully!")
}

func run(sourceDir string) error {
	// Load .env file for configuration
	envVars, err := loadEnvFile()
	if err != nil {
		return fmt.Errorf("failed to load .env file: %w", err)
	}

	// Phase 1: Backup
	fmt.Println("Phase 1: Creating backup...")
	if err := createBackup(sourceDir); err != nil {
		return fmt.Errorf("backup failed: %w", err)
	}

	// Phase 2: Create directory structure
	fmt.Println("Phase 2: Creating directory structure...")
	scenarioID := "default"
	if err := createDirectories(scenarioID); err != nil {
		return fmt.Errorf("directory creation failed: %w", err)
	}

	// Phase 3: Migrate data
	fmt.Println("Phase 3: Migrating data...")
	if err := migrateData(sourceDir, scenarioID, envVars); err != nil {
		return fmt.Errorf("data migration failed: %w", err)
	}

	// Phase 4: Verify
	fmt.Println("Phase 4: Verifying migration...")
	if err := verifyMigration(scenarioID); err != nil {
		return fmt.Errorf("verification failed: %w", err)
	}

	return nil
}

// EnvVars holds environment variables
type EnvVars struct {
	BotToken              string
	PrivateGroupID        string
	ProdamusProductName   string
	ProdamusProductPrice  string
	ProdamusPaidContent   string
}

func loadEnvFile() (*EnvVars, error) {
	data, err := os.ReadFile(".env")
	if err != nil {
		return nil, err
	}

	envVars := &EnvVars{}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) == 0 || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "BOT_TOKEN":
			envVars.BotToken = value
		case "PRIVATE_GROUP_ID":
			envVars.PrivateGroupID = value
		case "PRODAMUS_PRODUCT_NAME":
			envVars.ProdamusProductName = value
		case "PRODAMUS_PRODUCT_PRICE":
			envVars.ProdamusProductPrice = value
		case "PRODAMUS_PRODUCT_PAID_CONTENT":
			envVars.ProdamusPaidContent = value
		}
	}

	return envVars, nil
}

func createBackup(sourceDir string) error {
	backupDir := filepath.Join(sourceDir, "migration_backup")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return err
	}

	files := []string{
		filepath.Join(sourceDir, "messages.json"),
		filepath.Join(sourceDir, "users.json"),
		filepath.Join(sourceDir, "schedule_backup.json"),
		filepath.Join(sourceDir, "commands.json"),
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

func createDirectories(scenarioID string) error {
	dirs := []string{
		filepath.Join("data/scenarios", scenarioID, "messages"),
		filepath.Join("data/broadcasts", "messages"),
		filepath.Join("data/buttons"),
		filepath.Join("data/templates"),
		filepath.Join("data/schedules"),
		filepath.Join("data/wizards"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
		fmt.Printf("  Created: %s\n", dir)
	}

	return nil
}

func migrateData(sourceDir, scenarioID string, envVars *EnvVars) error {
	// Migrate messages to scenario (will be stored in registry)
	migratedMessages, err := loadMigratedMessages(sourceDir)
	if err != nil {
		return err
	}

	// Migrate users
	if err := migrateUsers(sourceDir, scenarioID); err != nil {
		return err
	}

	// Migrate schedule backup
	if err := migrateScheduleBackup(sourceDir); err != nil {
		return err
	}

	// Migrate commands if exists
	if err := migrateCommands(sourceDir); err != nil {
		return err
	}

	// Create scenario registry with all data (config + messages)
	if err := createScenarioRegistry(sourceDir, scenarioID, envVars, migratedMessages); err != nil {
		return err
	}

	// Create button registry
	if err := createButtonRegistry(); err != nil {
		return err
	}

	// Create bot globals template
	if err := createBotGlobals(); err != nil {
		return err
	}

	return nil
}

// loadMigratedMessages loads and converts messages from source (kept in memory for registry)
func loadMigratedMessages(sourceDir string) (map[string]interface{}, error) {
	sourcePath := filepath.Join(sourceDir, "messages.json")
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("  No messages.json to migrate")
			return nil, nil
		}
		return nil, err
	}

	// Parse as map to handle both legacy and new formats
	var rawMsgs map[string]interface{}
	if err := json.Unmarshal(data, &rawMsgs); err != nil {
		return nil, err
	}

	// Convert legacy message format to new format (kept in memory)
	return convertLegacyMessagesRaw(rawMsgs), nil
}

func convertLegacyMessagesRaw(rawMsgs map[string]interface{}) map[string]interface{} {
	// Extract messages list
	var messagesList []string
	if ml, ok := rawMsgs["messages_list"].([]interface{}); ok {
		for _, item := range ml {
			if str, ok := item.(string); ok {
				messagesList = append(messagesList, str)
			}
		}
	}

	// Convert messages
	messagesRaw, ok := rawMsgs["messages"].(map[string]interface{})
	if !ok {
		messagesRaw = make(map[string]interface{})
	}

	newMessages := make(map[string]interface{})

	for key, value := range messagesRaw {
		msgMap, ok := value.(map[string]interface{})
		if !ok {
			continue
		}

		newMsg := make(map[string]interface{})

		// Handle timing - could be array [hours, minutes] or object {hours, minutes}
		if timing, ok := msgMap["timing"].([]interface{}); ok && len(timing) == 2 {
			// Legacy format: [hours, minutes]
			hours := int(timing[0].(float64))
			minutes := int(timing[1].(float64))
			newMsg["timing"] = map[string]int{
				"hours":   hours,
				"minutes": minutes,
			}
		} else if timing, ok := msgMap["timing"].(map[string]interface{}); ok {
			// New format: {hours, minutes}
			newMsg["timing"] = timing
		}

		// Handle template_file
		if templateFile, ok := msgMap["template_file"].(string); ok && templateFile != "" {
			newMsg["template_file"] = templateFile
		}

		// Handle url_button conversion to inline_keyboard (legacy format)
		if urlBtn, ok := msgMap["url_button"].([]interface{}); ok && len(urlBtn) == 2 {
			url := urlBtn[0].(string)
			text := urlBtn[1].(string)
			newMsg["inline_keyboard"] = map[string]interface{}{
				"rows": []map[string]interface{}{
					{
						"buttons": []map[string]interface{}{
							{
								"type": "url",
								"text": text,
								"url":  url,
							},
						},
					},
				},
			}
		}

		// Copy inline_keyboard if exists (new format)
		if keyboard, ok := msgMap["inline_keyboard"].(map[string]interface{}); ok {
			newMsg["inline_keyboard"] = keyboard
		}

		newMessages[key] = newMsg
	}

	return map[string]interface{}{
		"messages_list": messagesList,
		"messages":      newMessages,
	}
}

func migrateUsers(sourceDir, scenarioID string) error {
	sourcePath := filepath.Join(sourceDir, "users.json")
	data, err := os.ReadFile(sourcePath)
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

	// Read schedule backup to determine user progress
	schedulePath := filepath.Join(sourceDir, "schedule_backup.json")
	type ScheduleBackup map[string]string // chatID -> nextSendTime
	var schedule ScheduleBackup
	if scheduleData, err := os.ReadFile(schedulePath); err == nil {
		json.Unmarshal(scheduleData, &schedule)
	} // ignore error if file doesn't exist

	// Migrate each user
	for chatID, user := range users {
		// Calculate current message index based on user state
		messageIndex := 0
		totalMessages := len(user.MessagesList)

		// If user has joined the group, they've completed all messages
		if user.JoinedGroup {
			messageIndex = totalMessages
		} else if user.HasPaid() {
			// If paid but not joined, check if they have a scheduled message
			if _, hasSchedule := schedule[chatID]; !hasSchedule {
				// No schedule means they've completed the flow
				messageIndex = totalMessages
			}
			// Otherwise, they're still in progress - index stays at current position
		}
		// If not paid, index stays at 0

		// Create scenario state from legacy fields
		scenarioState := &services.UserScenarioState{
			Status:              services.StatusActive,
			CurrentMessageIndex: messageIndex,
			PaymentDate:         user.PaymentDate,
			PaymentLink:         user.PaymentLink,
			InviteLink:          user.InviteLink,
			JoinedGroup:         user.JoinedGroup,
			JoinedAt:            user.JoinedAt,
		}

		// If completed, mark as completed
		if messageIndex >= totalMessages && totalMessages > 0 {
			scenarioState.Status = services.StatusCompleted
			now := time.Now()
			scenarioState.CompletedAt = &now
		}

		// Update user structure
		user.Scenarios = map[string]*services.UserScenarioState{
			scenarioID: scenarioState,
		}
		user.ActiveScenarioID = scenarioID

		// Clear legacy fields
		user.MessagesList = nil
		user.PaymentDate = nil
		user.PaymentLink = ""
		user.InviteLink = ""
		user.JoinedGroup = false
		user.JoinedAt = nil
	}

	// Write migrated users to data/users.json
	data, err = json.MarshalIndent(users, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile("data/users.json", data, 0644); err != nil {
		return err
	}
	fmt.Printf("  Migrated %d users to data/users.json\n", len(users))

	return nil
}

func migrateScheduleBackup(sourceDir string) error {
	sourcePath := filepath.Join(sourceDir, "schedule_backup.json")
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("  No schedule_backup.json to migrate")
			return nil
		}
		return err
	}

	// Copy to data/schedules/backup.json
	if err := os.MkdirAll("data/schedules", 0755); err != nil {
		return err
	}

	destPath := "data/schedules/backup.json"
	if err := os.WriteFile(destPath, data, 0644); err != nil {
		return err
	}
	fmt.Printf("  Migrated schedule_backup.json to data/schedules/backup.json\n")

	return nil
}

func migrateCommands(sourceDir string) error {
	sourcePath := filepath.Join(sourceDir, "commands.json")
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("  No commands.json to migrate")
			return nil
		}
		return err
	}

	// Copy to data/commands.json
	destPath := "data/commands.json"
	if err := os.WriteFile(destPath, data, 0644); err != nil {
		return err
	}
	fmt.Printf("  Migrated commands.json to data/commands.json\n")

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
			"bot.bot_name":     "Mispilka Bot",
			"bot.support_link": "https://t.me/support",
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

func createScenarioRegistry(sourceDir, scenarioID string, envVars *EnvVars, messages map[string]interface{}) error {
	// Prepare messages structure (empty if no messages)
	messagesData := map[string]interface{}{
		"messages_list": []string{},
		"messages":      map[string]interface{}{},
	}
	if messages != nil {
		messagesData = messages
	}

	registry := map[string]interface{}{
		"scenarios": map[string]interface{}{
			scenarioID: map[string]interface{}{
				"id":         scenarioID,
				"name":       "Migrated Scenario",
				"created_at": time.Now().Format(time.RFC3339),
				"is_active":  true,
				"config": map[string]interface{}{
					"prodamus": map[string]interface{}{
						"product_name":     envVars.ProdamusProductName,
						"product_price":    envVars.ProdamusProductPrice,
						"paid_content":     envVars.ProdamusPaidContent,
						"private_group_id": envVars.PrivateGroupID,
					},
				},
				"messages":  messagesData,
				"summary": map[string]interface{}{
					"template_file": "summary.md",
				},
			},
		},
		"default_scenario_id": scenarioID,
	}

	data, err := json.MarshalIndent(registry, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile("data/scenarios/registry.json", data, 0644); err != nil {
		return err
	}
	fmt.Println("  Created scenario registry")

	// Copy message templates directory if exists
	msgsDir := filepath.Join(sourceDir, "messages")
	destMsgsDir := filepath.Join("data/scenarios", scenarioID, "messages")
	if info, err := os.Stat(msgsDir); err == nil && info.IsDir() {
		if err := os.MkdirAll(destMsgsDir, 0755); err != nil {
			return err
		}
		if err := filepath.Walk(msgsDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			if filepath.Ext(path) == ".md" {
				dest := filepath.Join(destMsgsDir, filepath.Base(path))
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
	}

	return nil
}

func verifyMigration(scenarioID string) error {
	requiredFiles := []string{
		"data/scenarios/registry.json",
		"data/buttons/registry.json",
		"data/templates/bot_globals.json",
		"data/users.json",
	}

	for _, file := range requiredFiles {
		if _, err := os.Stat(file); err != nil {
			return fmt.Errorf("required file missing: %s", file)
		}
		fmt.Printf("  ✓ %s exists\n", file)
	}

	// Verify scenario has messages loaded
	data, err := os.ReadFile("data/scenarios/registry.json")
	if err != nil {
		return err
	}
	var registry map[string]interface{}
	if err := json.Unmarshal(data, &registry); err != nil {
		return err
	}
	scenarios, ok := registry["scenarios"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid registry format: scenarios not found")
	}
	_, ok = scenarios[scenarioID].(map[string]interface{})
	if !ok {
		return fmt.Errorf("scenario %s not found in registry", scenarioID)
	}
	fmt.Printf("  ✓ Scenario '%s' loaded in registry\n", scenarioID)

	return nil
}
