package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"mispilkabot/internal/services"
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

	// Create scenario registry
	if err := createScenarioRegistry(); err != nil {
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
	for _, user := range users {
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
				"config": map[string]interface{}{
					"prodamus": map[string]interface{}{
						"product_name":    "Доступ к обучающим материалам",
						"product_price":   "500",
						"paid_content":    "default",
						"private_group_id": "",
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
