package config

import "time"

func SetDefaultConfig() *Config {
	return &Config{
		Database: DatabaseConfig{
			Host:         "localhost",
			Port:         5432,
			User:         "postgres",
			Password:     "",
			Name:         "promo-bots",
			SSLMode:      "require",
			MaxOpenConns: 10,
			MaxIdleConns: 5,
			ConnMaxLifetime: 1 * time.Hour,
			ConnMaxIdleTime: 15 * time.Minute,
			HealthCheckPeriod: 1 * time.Minute,
		},
		Logger: LoggerConfig{
			Level:        "info",
			Format:       "json",
			Output:       "stdout",
			EnableColors: false,
			FilePath:     "",
			MaxSize:      0,
			MaxBackups:   0,
			MaxAge:       0,
			Compress:     false,
		},
		Crypto: CryptoConfig{
			Algorithm: "aes_gcm",
		},
	}
}
