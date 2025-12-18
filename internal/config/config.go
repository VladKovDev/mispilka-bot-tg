package config

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Env      string `yaml:"env"`
	Database DatabaseConfig
	Logger   LoggerConfig
}

type DatabaseConfig struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	User         string `mapstructure:"user"`
	Password     string `mapstructure:"password"`
	Name         string `mapstructure:"name"`
	SSLMode      string `mapstructure:"sslmode"`
	MaxOpenConns int    `mapstructure:"max_open_conns"`
	MaxIdleConns int    `mapstructure:"max_idle_conns"`
}

type LoggerConfig struct {
	Level        string `mapstructure:"level"`
	Format       string `mapstructure:"format"`
	Output       string `mapstructure:"output"`
	EnableColors bool   `mapstructure:"enable_colors"`
	FilePath     string `mapstructure:"file_path"`
	MaxSize      int    `mapstructure:"max_size"`
	MaxBackups   int    `mapstructure:"max_backups"`
	MaxAge       int    `mapstructure:"max_age"`
	Compress     bool   `mapstructure:"compress"`
}

type Loader interface {
	Load(ctx context.Context) (*Config, error)
}

type viperLoader struct {
	configPath string
	validator  Validator
}

func NewViperLoader(configPath string, validator Validator) Loader {
	if configPath == "" {
		configPath = "."
	}
	return &viperLoader{
		configPath: configPath,
		validator:  validator,
	}
}

func (l *viperLoader) Load(ctx context.Context) (*Config, error) {
	cfg := SetDefaultConfig()

	v := viper.New()

	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(l.configPath)
	v.AddConfigPath(".")

	if err := v.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFoundError) {
			return nil, fmt.Errorf("failed to read config: %w", err)
		}
	}

	// env config
	v.SetConfigName(".env")
	v.SetConfigType("env")
	v.AddConfigPath(l.configPath)
	v.AddConfigPath(".")
	if err := v.MergeInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFoundError) {
			return nil, fmt.Errorf("failed to read env: %w", err)
		}
	}

	v.AutomaticEnv()
	v.SetEnvPrefix("PROMO_BOTS")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	l.BindEnvVariables(v)

	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if err := l.validator.Validate(cfg); err != nil {
		return nil, fmt.Errorf("config failed validation: %w", err)
	}

	return cfg, nil
}

func (l *viperLoader) BindEnvVariables(v *viper.Viper) {
	// Database
	_ = v.BindEnv("database.host")
	_ = v.BindEnv("database.port")
	_ = v.BindEnv("database.user")
	_ = v.BindEnv("database.password")
	_ = v.BindEnv("database.name")
	_ = v.BindEnv("database.sslmode")
	_ = v.BindEnv("database.max_open_conns")
	_ = v.BindEnv("database.max_idle_conns")
	// Logger
	_ = v.BindEnv("logger.level")
	_ = v.BindEnv("logger.format")
	_ = v.BindEnv("logger.output")
	_ = v.BindEnv("logger.enable_colors")
	_ = v.BindEnv("logger.file_path")
	_ = v.BindEnv("logger.max_size")
	_ = v.BindEnv("logger.max_backups")
	_ = v.BindEnv("logger.max_age")
	_ = v.BindEnv("logger.compress")
}

func Load(configPath string, ctx context.Context) (*Config, error) {
	loader := NewViperLoader(configPath, NewValidator())
	return loader.Load(ctx)
}

func (c *DatabaseConfig) GetDatabaseDSN() string {
	return fmt.Sprintf(
		"%s:%s@%s:%d/%s?sslmode=%s",
		c.User,
		url.QueryEscape(c.Password),
		c.Host,
		c.Port,
		c.Name,
		c.SSLMode,
	)
}
