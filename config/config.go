package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	BotToken                   string
	PrivateGroupID             string
	AdminIDs                   []int64
	WebhookHost                string
	WebhookPort                string
	WebhookPath                string
	ProdamusSecret             string
	ProdamusAPIURL             string
	ProdamusProductName        string
	ProdamusProductPrice       string
	ProdamusProductPaidContent string
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		fmt.Printf("Error loading .env file: %v\n", err)
		panic(err)
	}

	return &Config{
		BotToken:                   getEnv("BOT_TOKEN", ""),
		PrivateGroupID:             getEnv("PRIVATE_GROUP_ID", ""),
		AdminIDs:                   parseAdminIDs(getEnv("ADMIN_IDS", "")),
		WebhookHost:                getEnv("WEBHOOK_HOST", "0.0.0.0"),
		WebhookPort:                getEnv("WEBHOOK_PORT", "8080"),
		WebhookPath:                getEnv("WEBHOOK_PATH", "/webhook/prodamus"),
		ProdamusSecret:             getEnv("PRODAMUS_SECRET_KEY", ""),
		ProdamusAPIURL:             getEnv("PRODAMUS_API_URL", ""),
		ProdamusProductName:        getEnv("PRODAMUS_PRODUCT_NAME", "Доступ к обучающим материалам"),
		ProdamusProductPrice:       getEnv("PRODAMUS_PRODUCT_PRICE", "500"),
		ProdamusProductPaidContent: getEnv("PRODAMUS_PRODUCT_PAID_CONTENT", "Успешно! Переходите обратно в бота и вступайте в нашу закрытую группу"),
	}
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func Validate(cfg *Config) error {
	if cfg.BotToken == "" {
		return fmt.Errorf("BOT_TOKEN is required")
	}
	if cfg.ProdamusSecret == "" {
		return fmt.Errorf("PRODAMUS_SECRET_KEY is required")
	}
	if cfg.ProdamusAPIURL == "" {
		return fmt.Errorf("PRODAMUS_API_URL is required")
	}
	return nil
}

// parseAdminIDs parses comma-separated admin IDs from env string
func parseAdminIDs(s string) []int64 {
	if s == "" {
		return []int64{}
	}

	parts := strings.Split(s, ",")
	var ids []int64
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, err := strconv.ParseInt(part, 10, 64)
		if err != nil {
			fmt.Printf("Warning: invalid admin ID '%s': %v\n", part, err)
			continue
		}
		ids = append(ids, id)
	}
	return ids
}
