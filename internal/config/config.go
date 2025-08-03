// Пакет config отвечает за загрузку и парсинг кофигурационыых файлов
package config

import (
	"fmt"
	"github.com/spf13/viper"
	"os"
	"time"
)

type Config struct {
	Env         string `mapstructure:"env"`
	StoragePath string `mapstructure:"storage_path"`
	HTTPServer  `mapstructure:"http_server"`
}

type HTTPServer struct {
	Address     string        `mapstructure:"address"`
	Timeout     time.Duration `mapstructure:"timeout"`
	IdleTimeout time.Duration `mapstructure:"idle_timeout"`
}

// Load загружает конфигурацию из указанной директории
// 1. Сначала пытается найти config.yaml
// 2. Если не найден, пробует загрузить config.example.yaml
func Load(configPath string) (*Config, error) {
	env := os.Getenv("PAYMENT_ENV")
	if env == "" {
		env = "development"
	}

	viper.AddConfigPath(configPath)
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	viper.SetEnvPrefix("PAYMENT")
	viper.AutomaticEnv()

	viper.SetDefault("env", "development")
	viper.SetDefault("http_server.address", "0.0.0.0:8080")
	viper.SetDefault("http_server.timeout", "4s")
	viper.SetDefault("http_server.idle_timeout", "60s")
	viper.SetDefault("storage_path", "/app/data/app.db")

	if err := viper.ReadInConfig(); err != nil {
		viper.SetConfigName("config.example")
		if err := viper.MergeInConfig(); err != nil {
			return nil, fmt.Errorf("failed to read config: %w", err)
		}
	}
	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if err := parseDurations(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// parseDurations конвертирует строковые таймауты в time.Duration
func parseDurations(cfg *Config) error {
	timeout, err := time.ParseDuration(viper.GetString("http_server.timeout"))
	if err != nil {
		return fmt.Errorf("failed to parse http_server.timeout: %w", err)
	}
	idleTimeout, err := time.ParseDuration(viper.GetString("http_server.idle_timeout"))
	if err != nil {
		return fmt.Errorf("failed to parse http_server.idle_timeout: %w", err)
	}

	cfg.HTTPServer.Timeout = timeout
	cfg.HTTPServer.IdleTimeout = idleTimeout
	return nil
}
