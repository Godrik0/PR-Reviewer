package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig
	Storage  StorageConfig
	Auth     AuthConfig
	LogLevel string
}

type ServerConfig struct {
	Port            int
	ReadTimeout     int
	WriteTimeout    int
	ShutdownTimeout int
}

type StorageConfig struct {
	Type        string
	PostgresURL string
}

type AuthConfig struct {
	Type       string
	AdminToken string
	UserToken  string
}

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath("/etc/pr-reviewer")

	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.read_timeout", 10)
	viper.SetDefault("server.write_timeout", 10)
	viper.SetDefault("server.shutdown_timeout", 10)
	viper.SetDefault("storage.type", "memory")
	viper.SetDefault("storage.postgres_url", "")
	viper.SetDefault("auth.type", "static")
	viper.SetDefault("auth.admin_token", "admin-secret-token")
	viper.SetDefault("auth.user_token", "user-secret-token")
	viper.SetDefault("log_level", "info")

	viper.AutomaticEnv()
	viper.SetEnvPrefix("PR_REVIEWER")

	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	cfg := &Config{
		Server: ServerConfig{
			Port:            viper.GetInt("server.port"),
			ReadTimeout:     viper.GetInt("server.read_timeout"),
			WriteTimeout:    viper.GetInt("server.write_timeout"),
			ShutdownTimeout: viper.GetInt("server.shutdown_timeout"),
		},
		Storage: StorageConfig{
			Type:        viper.GetString("storage.type"),
			PostgresURL: viper.GetString("storage.postgres_url"),
		},
		Auth: AuthConfig{
			Type:       viper.GetString("auth.type"),
			AdminToken: viper.GetString("auth.admin_token"),
			UserToken:  viper.GetString("auth.user_token"),
		},
		LogLevel: viper.GetString("log_level"),
	}

	return cfg, nil
}
