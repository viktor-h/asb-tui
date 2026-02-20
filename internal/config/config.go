package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultRefreshSeconds = 10
	defaultDLQFetchMode   = "peek"
	defaultDLQFetchCount  = 10
)

type Config struct {
	Namespace           string
	RefreshInterval     time.Duration
	ActiveWarnThreshold int64
	DLQWarnThreshold    int64
	DLQFetchMode        string
	DLQFetchCount       int
}

func Load() (Config, error) {
	return load(os.Getenv)
}

func load(getenv func(string) string) (Config, error) {
	namespace := strings.TrimSpace(getenv("ASB_NAMESPACE"))
	if namespace == "" {
		return Config{}, fmt.Errorf("ASB_NAMESPACE is required")
	}

	refreshSeconds, err := parseOptionalInt(getenv("ASB_REFRESH_SECONDS"), defaultRefreshSeconds)
	if err != nil {
		return Config{}, fmt.Errorf("ASB_REFRESH_SECONDS: %w", err)
	}
	if refreshSeconds <= 0 {
		return Config{}, fmt.Errorf("ASB_REFRESH_SECONDS must be greater than 0")
	}

	activeWarnThreshold, err := parseOptionalInt64(getenv("ASB_ACTIVE_WARN_THRESHOLD"), -1)
	if err != nil {
		return Config{}, fmt.Errorf("ASB_ACTIVE_WARN_THRESHOLD: %w", err)
	}

	dlqWarnThreshold, err := parseOptionalInt64(getenv("ASB_DLQ_WARN_THRESHOLD"), -1)
	if err != nil {
		return Config{}, fmt.Errorf("ASB_DLQ_WARN_THRESHOLD: %w", err)
	}

	dlqFetchMode, err := parseDLQFetchMode(getenv("ASB_DLQ_FETCH_MODE"), defaultDLQFetchMode)
	if err != nil {
		return Config{}, fmt.Errorf("ASB_DLQ_FETCH_MODE: %w", err)
	}

	dlqFetchCount, err := parseOptionalInt(getenv("ASB_DLQ_FETCH_COUNT"), defaultDLQFetchCount)
	if err != nil {
		return Config{}, fmt.Errorf("ASB_DLQ_FETCH_COUNT: %w", err)
	}

	if activeWarnThreshold < -1 {
		return Config{}, fmt.Errorf("ASB_ACTIVE_WARN_THRESHOLD must be -1 or higher")
	}
	if dlqWarnThreshold < -1 {
		return Config{}, fmt.Errorf("ASB_DLQ_WARN_THRESHOLD must be -1 or higher")
	}
	if dlqFetchCount <= 0 {
		return Config{}, fmt.Errorf("ASB_DLQ_FETCH_COUNT must be greater than 0")
	}

	return Config{
		Namespace:           namespace,
		RefreshInterval:     time.Duration(refreshSeconds) * time.Second,
		ActiveWarnThreshold: activeWarnThreshold,
		DLQWarnThreshold:    dlqWarnThreshold,
		DLQFetchMode:        dlqFetchMode,
		DLQFetchCount:       dlqFetchCount,
	}, nil
}

func parseDLQFetchMode(raw string, fallback string) (string, error) {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "" {
		value = fallback
	}

	switch value {
	case "peek", "peeklock", "receiveanddelete":
		return value, nil
	default:
		return "", fmt.Errorf("must be one of: peek, peeklock, receiveanddelete")
	}
}

func parseOptionalInt(raw string, fallback int) (int, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return fallback, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("must be an integer")
	}

	return parsed, nil
}

func parseOptionalInt64(raw string, fallback int64) (int64, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return fallback, nil
	}

	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("must be an integer")
	}

	return parsed, nil
}
