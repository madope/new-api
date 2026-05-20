package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Mode                string
	SourceDSN           string
	TargetDSN           string
	IncrementalInterval time.Duration
	RewindInterval      time.Duration
	RewindWindow        time.Duration
	BatchSize           int
	BackfillStart       *time.Time
	BackfillEnd         *time.Time
}

func Load() (Config, error) {
	cfg := Config{
		Mode:                getEnv("LOGSAGG_MODE", "run"),
		SourceDSN:           resolveSourceDSN(),
		TargetDSN:           resolveTargetDSN(),
		IncrementalInterval: time.Duration(getEnvInt("LOGSAGG_INCREMENTAL_INTERVAL_SECONDS", 15)) * time.Second,
		RewindInterval:      time.Duration(getEnvInt("LOGSAGG_REWIND_INTERVAL_MINUTES", 10)) * time.Minute,
		RewindWindow:        time.Duration(getEnvInt("LOGSAGG_REWIND_WINDOW_MINUTES", 30)) * time.Minute,
		BatchSize:           getEnvInt("LOGSAGG_BATCH_SIZE", 500),
	}

	if cfg.SourceDSN == "" {
		return Config{}, fmt.Errorf("LOGSAGG_SOURCE_DSN or LOGSAGG_DSN is required")
	}
	if cfg.BatchSize <= 0 {
		return Config{}, fmt.Errorf("LOGSAGG_BATCH_SIZE must be greater than 0")
	}

	switch cfg.Mode {
	case "run":
		return cfg, nil
	case "backfill":
		start, err := parseRequiredTime("LOGSAGG_BACKFILL_START")
		if err != nil {
			return Config{}, err
		}
		end, err := parseRequiredTime("LOGSAGG_BACKFILL_END")
		if err != nil {
			return Config{}, err
		}
		if !end.After(start) {
			return Config{}, fmt.Errorf("LOGSAGG_BACKFILL_END must be after LOGSAGG_BACKFILL_START")
		}
		cfg.BackfillStart = &start
		cfg.BackfillEnd = &end
		return cfg, nil
	default:
		return Config{}, fmt.Errorf("invalid LOGSAGG_MODE: %s", cfg.Mode)
	}
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

func parseRequiredTime(key string) (time.Time, error) {
	value := os.Getenv(key)
	if value == "" {
		return time.Time{}, fmt.Errorf("%s is required", key)
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid %s: %w", key, err)
	}
	return parsed.UTC(), nil
}

func resolveSourceDSN() string {
	if value := os.Getenv("LOGSAGG_SOURCE_DSN"); value != "" {
		return value
	}
	return os.Getenv("LOGSAGG_DSN")
}

func resolveTargetDSN() string {
	if value := os.Getenv("LOGSAGG_TARGET_DSN"); value != "" {
		return value
	}
	return resolveSourceDSN()
}
