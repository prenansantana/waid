package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config holds all application configuration.
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	NATS     NATSConfig     `mapstructure:"nats"`
	Resolver ResolverConfig `mapstructure:"resolver"`
	Logging  LoggingConfig  `mapstructure:"logging"`
	Meta     MetaConfig     `mapstructure:"meta"`
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Port   int    `mapstructure:"port"`
	APIKey string `mapstructure:"api_key"`
}

// DatabaseConfig holds database connection settings.
type DatabaseConfig struct {
	Driver string `mapstructure:"driver"` // sqlite | postgres
	URL    string `mapstructure:"url"`    // file path or connection string
}

// NATSConfig holds NATS messaging settings.
type NATSConfig struct {
	Embedded bool   `mapstructure:"embedded"`
	URL      string `mapstructure:"url"`
}

// ResolverConfig holds identity resolution settings.
type ResolverConfig struct {
	AutoCreate bool `mapstructure:"auto_create"`
}

// LoggingConfig holds logging settings.
type LoggingConfig struct {
	Level  string `mapstructure:"level"`  // debug, info, warn, error
	Format string `mapstructure:"format"` // json, text
}

// MetaConfig holds Meta Cloud API webhook settings.
type MetaConfig struct {
	VerifyToken string `mapstructure:"verify_token"`
}

// Load reads configuration from the waid.yaml file and environment variables.
// Environment variables take precedence over file values.
// The prefix WAID_ is used for all env vars (e.g., WAID_SERVER_PORT).
func Load() (*Config, error) {
	v := viper.New()

	// Set defaults
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.api_key", "")
	v.SetDefault("database.driver", "sqlite")
	v.SetDefault("database.url", "waid.db")
	v.SetDefault("nats.embedded", true)
	v.SetDefault("nats.url", "nats://localhost:4222")
	v.SetDefault("resolver.auto_create", true)
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "json")

	// Config file
	v.SetConfigName("waid")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("$HOME/.waid")
	v.AddConfigPath("/etc/waid")

	// Environment variables
	v.SetEnvPrefix("WAID")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Read config file (optional — env vars alone are sufficient)
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("reading config file: %w", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshalling config: %w", err)
	}

	return &cfg, nil
}
