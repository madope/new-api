package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	DSN                string
	Host               string
	Port               int
	DefaultPageSize    int
	MaxPageSize        int
	ReadTimeoutSeconds int
}

func Load() (Config, error) {
	cfg := Config{
		DSN:                os.Getenv("BILLING_DSN"),
		Host:               getEnv("BILLING_HOST", "127.0.0.1"),
		Port:               getEnvInt("BILLING_PORT", 3011),
		DefaultPageSize:    getEnvInt("BILLING_DEFAULT_PAGE_SIZE", 20),
		MaxPageSize:        getEnvInt("BILLING_MAX_PAGE_SIZE", 100),
		ReadTimeoutSeconds: getEnvInt("BILLING_READ_TIMEOUT_SECONDS", 15),
	}
	if cfg.DSN == "" {
		return Config{}, fmt.Errorf("BILLING_DSN is required")
	}
	if cfg.Port <= 0 {
		return Config{}, fmt.Errorf("BILLING_PORT must be greater than 0")
	}
	if cfg.DefaultPageSize <= 0 {
		return Config{}, fmt.Errorf("BILLING_DEFAULT_PAGE_SIZE must be greater than 0")
	}
	if cfg.MaxPageSize <= 0 {
		return Config{}, fmt.Errorf("BILLING_MAX_PAGE_SIZE must be greater than 0")
	}
	if cfg.DefaultPageSize > cfg.MaxPageSize {
		cfg.DefaultPageSize = cfg.MaxPageSize
	}
	if cfg.ReadTimeoutSeconds <= 0 {
		return Config{}, fmt.Errorf("BILLING_READ_TIMEOUT_SECONDS must be greater than 0")
	}
	return cfg, nil
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}
