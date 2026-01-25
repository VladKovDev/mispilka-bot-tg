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
		fmt.Fprintf(os.Stderr, "Example: %s data/test2\n", os.Args[0])
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
		filepath.Join(sourceDir, "README.md"),
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

	// Backup messages directory
	msgsDir := filepath.Join(sourceDir, "messages")
	if info, err := os.Stat(msgsDir); err == nil && info.IsDir() {
		backupMsgsDir := filepath.Join(backupDir, "messages")
		if err := filepath.Walk(msgsDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			if filepath.Ext(path) == ".md" {
				relPath, _ := filepath.Rel(msgsDir, path)
				dest := filepath.Join(backupMsgsDir, relPath)
				if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
					return err
				}
				data, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				if err := os.WriteFile(dest, data, 0644); err != nil {
					return err
				}
				fmt.Printf("  Backed up template: %s\n", relPath)
			}
			return nil
		}); err != nil {
			return err
		}
	}

	// Backup assets directory if exists
	assetsDir := filepath.Join(sourceDir, "assets")
	if info, err := os.Stat(assetsDir); err == nil && info.IsDir() {
		backupAssetsDir := filepath.Join(backupDir, "assets")
		if err := filepath.Walk(assetsDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			relPath, _ := filepath.Rel(assetsDir, path)
			dest := filepath.Join(backupAssetsDir, relPath)
			if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
				return err
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if err := os.WriteFile(dest, data, 0644); err != nil {
				return err
			}
			fmt.Printf("  Backed up asset: %s\n", relPath)
			return nil
		}); err != nil {
			return err
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
		filepath.Join("data/assets"),
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
	// Migrate messages to scenario
	if err := migrateMessages(sourceDir, scenarioID); err != nil {
		return err
	}

	// Migrate users
	if err := migrateUsers(sourceDir, scenarioID); err != nil {
		return err
	}

	// Migrate assets
	if err := migrateAssets(sourceDir); err != nil {
		return err
	}

	// Migrate schedule backup
	if err := migrateScheduleBackup(sourceDir); err != nil {
		return err
	}

	// Create scenario config with .env values
	if err := createScenarioConfig(scenarioID, envVars); err != nil {
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

	// Create scenario registry
	if err := createScenarioRegistry(scenarioID, envVars); err != nil {
		return err
	}

	return nil
}

func migrateMessages(sourceDir, scenarioID string) error {
	sourcePath := filepath.Join(sourceDir, "messages.json")
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("  No messages.json to migrate")
			return nil
		}
		return err
	}

	// The messages.json is already in the correct format for new scenarios
	// Just copy it over
	destPath := filepath.Join("data/scenarios", scenarioID, "messages.json")
	if err := os.WriteFile(destPath, data, 0644); err != nil {
		return err
	}
	fmt.Printf("  Migrated messages to %s\n", destPath)

	// Copy message templates directory
	msgsDir := filepath.Join(sourceDir, "messages")
	destMsgsDir := filepath.Join("data/scenarios", scenarioID, "messages")
	if info, err := os.Stat(msgsDir); err == nil && info.IsDir() {
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

func migrateAssets(sourceDir string) error {
	assetsDir := filepath.Join(sourceDir, "assets")
	if info, err := os.Stat(assetsDir); err != nil || !info.IsDir() {
		fmt.Println("  No assets directory to migrate")
		return nil
	}

	destAssetsDir := "data/assets"
	count := 0
	if err := filepath.Walk(assetsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		relPath, _ := filepath.Rel(assetsDir, path)
		dest := filepath.Join(destAssetsDir, relPath)
		if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.WriteFile(dest, data, 0644); err != nil {
			return err
		}
		count++
		return nil
	}); err != nil {
		return err
	}

	fmt.Printf("  Migrated %d asset files to data/assets\n", count)
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

func createScenarioConfig(scenarioID string, envVars *EnvVars) error {
	config := map[string]interface{}{
		"id":         scenarioID,
		"name":       "Migrated Scenario",
		"created_at": time.Now().Format(time.RFC3339),
		"prodamus": map[string]string{
			"product_name":     envVars.ProdamusProductName,
			"product_price":    envVars.ProdamusProductPrice,
			"paid_content":     envVars.ProdamusPaidContent,
			"private_group_id": envVars.PrivateGroupID,
		},
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	configPath := filepath.Join("data/scenarios", scenarioID, "config.json")
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return err
	}
	fmt.Printf("  Created scenario config: %s\n", configPath)

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

func createScenarioRegistry(scenarioID string, envVars *EnvVars) error {
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
				"messages": map[string]interface{}{
					"messages_list": []string{},
					"messages":      map[string]interface{}{},
				},
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

	return nil
}

func verifyMigration(scenarioID string) error {
	requiredFiles := []string{
		"data/scenarios/registry.json",
		filepath.Join("data/scenarios", scenarioID, "config.json"),
		filepath.Join("data/scenarios", scenarioID, "messages.json"),
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

	return nil
}
