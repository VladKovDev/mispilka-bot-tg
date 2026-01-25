package config

import (
	"os"
	"testing"
)

func TestValidate_ValidConfigGroupMode(t *testing.T) {
	cfg := &Config{
		BotToken:                   "test_token",
		PrivateResourceID:          "123456789",
		PrivateResourceType:        ResourceTypeGroup,
		ProdamusSecret:             "test_secret",
		ProdamusAPIURL:             "https://test.com",
	}

	err := Validate(cfg)
	if err != nil {
		t.Errorf("Validate() returned unexpected error for valid group config: %v", err)
	}
}

func TestValidate_ValidConfigChannelMode(t *testing.T) {
	cfg := &Config{
		BotToken:                   "test_token",
		PrivateResourceID:          "123456789",
		PrivateResourceType:        ResourceTypeChannel,
		ProdamusSecret:             "test_secret",
		ProdamusAPIURL:             "https://test.com",
	}

	err := Validate(cfg)
	if err != nil {
		t.Errorf("Validate() returned unexpected error for valid channel config: %v", err)
	}
}

func TestValidate_MissingResourceId(t *testing.T) {
	cfg := &Config{
		BotToken:                   "test_token",
		PrivateResourceID:          "",
		PrivateResourceType:        ResourceTypeGroup,
		ProdamusSecret:             "test_secret",
		ProdamusAPIURL:             "https://test.com",
	}

	err := Validate(cfg)
	if err == nil {
		t.Error("Validate() should return error when PrivateResourceID is empty")
	}
}

func TestValidate_InvalidResourceType(t *testing.T) {
	cfg := &Config{
		BotToken:                   "test_token",
		PrivateResourceID:          "123456789",
		PrivateResourceType:        ResourceType("invalid"),
		ProdamusSecret:             "test_secret",
		ProdamusAPIURL:             "https://test.com",
	}

	err := Validate(cfg)
	if err == nil {
		t.Error("Validate() should return error for invalid ResourceType")
	}
}

func TestLoad_ChannelModeFromEnv(t *testing.T) {
	// Set test env vars
	os.Setenv("BOT_TOKEN", "test_token")
	os.Setenv("PRIVATE_RESOURCE_ID", "123456789")
	os.Setenv("PRIVATE_RESOURCE_TYPE", "channel")
	os.Setenv("PRODAMUS_SECRET_KEY", "test_secret")
	os.Setenv("PRODAMUS_API_URL", "https://test.com")
	defer func() {
		os.Unsetenv("BOT_TOKEN")
		os.Unsetenv("PRIVATE_RESOURCE_ID")
		os.Unsetenv("PRIVATE_RESOURCE_TYPE")
		os.Unsetenv("PRODAMUS_SECRET_KEY")
		os.Unsetenv("PRODAMUS_API_URL")
	}()

	cfg := Load()

	if cfg.PrivateResourceID != "123456789" {
		t.Errorf("Load() PrivateResourceID = %s, want %s", cfg.PrivateResourceID, "123456789")
	}
	if cfg.PrivateResourceType != ResourceTypeChannel {
		t.Errorf("Load() PrivateResourceType = %s, want %s", cfg.PrivateResourceType, ResourceTypeChannel)
	}
}
