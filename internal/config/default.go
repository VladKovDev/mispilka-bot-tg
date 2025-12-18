package config

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
	}
}
