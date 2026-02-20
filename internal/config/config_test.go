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
			"ASB_USE_FAKE":              "false",
			"ASB_REFRESH_SECONDS":       "15",
			"ASB_ACTIVE_WARN_THRESHOLD": "100",
			"ASB_DLQ_WARN_THRESHOLD":    "2",
			"ASB_DLQ_FETCH_MODE":        "peeklock",
			"ASB_DLQ_FETCH_COUNT":       "25",
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
	if cfg.UseFake {
		t.Fatal("expected fake mode disabled")
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
	if cfg.DLQFetchMode != "peeklock" {
		t.Fatalf("unexpected dlq fetch mode: %s", cfg.DLQFetchMode)
	}
	if cfg.DLQFetchCount != 25 {
		t.Fatalf("unexpected dlq fetch count: %d", cfg.DLQFetchCount)
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

func TestLoadAllowsMissingNamespaceInFakeMode(t *testing.T) {
	cfg, err := load(func(key string) string {
		if key == "ASB_USE_FAKE" {
			return "true"
		}
		return ""
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !cfg.UseFake {
		t.Fatal("expected fake mode enabled")
	}
	if cfg.Namespace != "fake.servicebus.windows.net" {
		t.Fatalf("unexpected namespace: %s", cfg.Namespace)
	}
}

func TestLoadRejectsInvalidUseFakeValue(t *testing.T) {
	_, err := load(func(key string) string {
		values := map[string]string{
			"ASB_NAMESPACE": "demo.servicebus.windows.net",
			"ASB_USE_FAKE":  "not-a-bool",
		}
		return values[key]
	})
	if err == nil {
		t.Fatal("expected ASB_USE_FAKE error")
	}
	if !strings.Contains(err.Error(), "ASB_USE_FAKE") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadUsesDLQFetchDefaults(t *testing.T) {
	cfg, err := load(func(key string) string {
		if key == "ASB_NAMESPACE" {
			return "demo.servicebus.windows.net"
		}
		return ""
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.DLQFetchMode != "peek" {
		t.Fatalf("expected default mode peek, got %s", cfg.DLQFetchMode)
	}
	if cfg.DLQFetchCount != 10 {
		t.Fatalf("expected default count 10, got %d", cfg.DLQFetchCount)
	}
}

func TestLoadRejectsInvalidDLQFetchMode(t *testing.T) {
	_, err := load(func(key string) string {
		values := map[string]string{
			"ASB_NAMESPACE":      "demo.servicebus.windows.net",
			"ASB_DLQ_FETCH_MODE": "bad-mode",
		}
		return values[key]
	})
	if err == nil {
		t.Fatal("expected dlq fetch mode error")
	}
	if !strings.Contains(err.Error(), "ASB_DLQ_FETCH_MODE") {
		t.Fatalf("unexpected error: %v", err)
	}
}
