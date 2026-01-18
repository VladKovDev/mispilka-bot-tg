package config

import "fmt"

type Validator interface {
	Validate(*Config) error
}

type validator struct{}

func NewValidator() *validator {
	return &validator{}
}
func (v validator) Validate(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("config can not be empty")
	}

	if err := v.validateDatabase(cfg.Database); err != nil {
		return fmt.Errorf("database config: %w", err)
	}

	if err := v.validateLogger(cfg.Logger); err != nil {
		return fmt.Errorf("logger config: %w", err)
	}

	if err := v.validateCrypto(cfg.Crypto); err != nil {
		return fmt.Errorf("crypto config: %w", err)
	}

	return nil
}

func (v validator) validateDatabase(database DatabaseConfig) error {
	if database.Host == "" {
		return fmt.Errorf("host is empty")
	}

	if database.Port < 1 || database.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535, got: %v", database.Port)
	}

	if database.User == "" {
		return fmt.Errorf("user name is empty")
	}

	if database.Name == "" {
		return fmt.Errorf("database name is empty")
	}

	validSSLModes := map[string]bool{
		"disable":     true,
		"require":     true,
		"verify-ca":   true,
		"verify-full": true,
	}
	if !validSSLModes[database.SSLMode] {
		return fmt.Errorf("sslmode must be (disable, require, verify-ca, verify-full), got: %v", database.SSLMode)
	}

	if database.MaxOpenConns < 1 {
		return fmt.Errorf("max open conns must be at least 1, got: %v", database.MaxOpenConns)
	}

	if database.MaxIdleConns < 0 {
		return fmt.Errorf("max idle conns must be non-negative, got: %v", database.MaxIdleConns)
	}
	if database.MaxIdleConns > database.MaxOpenConns {
		return fmt.Errorf("max_idle_conns (%d) cannot exceed max_open_conns (%d)", database.MaxIdleConns, database.MaxOpenConns)
	}

	if database.ConnMaxLifetime < 0 {
		return fmt.Errorf("conn_max_lifetime must be positive, got %v", database.ConnMaxLifetime)
	}

	if database.ConnMaxIdleTime < 0 {
		return fmt.Errorf("conn_max_idle_time must be positive, got %v", database.ConnMaxIdleTime)
	}

	return nil
}

func (v validator) validateLogger(logger LoggerConfig) error {
	validLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLevels[logger.Level] {
		return fmt.Errorf("logger level must be (debug, info, warn, error), got: %v", logger.Level)
	}

	validFormats := map[string]bool{
		"console": true,
		"json":    true,
	}
	if !validFormats[logger.Format] {
		return fmt.Errorf("logger format must be (string, json), got: %v", logger.Format)
	}

	validOutputs := map[string]bool{
		"stdout": true,
		"stderr": true,
		"file":   true,
	}
	if !validOutputs[logger.Output] {
		return fmt.Errorf("logger output must be (stdout, stderr), got: %v", logger.Output)
	}

	if logger.Output == "file" && logger.FilePath == "" {
		return fmt.Errorf("logger file_path required when output is 'file'")
	}

	if logger.MaxBackups < 0 {
		return fmt.Errorf("logger max backups must be non-negative, got: %v", logger.MaxBackups)
	}

	if logger.MaxAge < 0 {
		return fmt.Errorf("logger max age must be non-negative, got: %v", logger.MaxAge)
	}

	return nil
}

func (v validator) validateCrypto(crypto CryptoConfig) error {
	if crypto.Keys == nil {
		return fmt.Errorf("crypto keys is required")
	}

	for ver, key := range crypto.Keys {
		if len(key) != 32 {
			return fmt.Errorf("crypto key for version %d must be 32 bytes long, got: %d", ver, len(key))
		}
	}

	validAlgorithms := map[string]bool{
		"aes_gcm": true,
	}
	if !validAlgorithms[crypto.Algorithm] {
		return fmt.Errorf("crypto algorithm must be (aes_gcm), got: %v", crypto.Algorithm)
	}

	return nil
}
