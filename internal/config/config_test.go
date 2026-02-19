package config

import (
	"strings"
	"testing"
	"time"
)

func TestLoadParsesValues(t *testing.T) {
	getenv := func(key string) string {
		values := map[string]string{
			"ASB_NAMESPACE":             "demo.servicebus.windows.net",
			"ASB_REFRESH_SECONDS":       "15",
			"ASB_ACTIVE_WARN_THRESHOLD": "100",
			"ASB_DLQ_WARN_THRESHOLD":    "2",
		}
		return values[key]
	}

	cfg, err := load(getenv)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.Namespace != "demo.servicebus.windows.net" {
		t.Fatalf("unexpected namespace: %s", cfg.Namespace)
	}
	if cfg.RefreshInterval != 15*time.Second {
		t.Fatalf("unexpected refresh interval: %s", cfg.RefreshInterval)
	}
	if cfg.ActiveWarnThreshold != 100 {
		t.Fatalf("unexpected active threshold: %d", cfg.ActiveWarnThreshold)
	}
	if cfg.DLQWarnThreshold != 2 {
		t.Fatalf("unexpected dlq threshold: %d", cfg.DLQWarnThreshold)
	}
}

func TestLoadRequiresNamespace(t *testing.T) {
	_, err := load(func(string) string { return "" })
	if err == nil {
		t.Fatal("expected namespace error")
	}
	if !strings.Contains(err.Error(), "ASB_NAMESPACE") {
		t.Fatalf("unexpected error: %v", err)
	}
}
