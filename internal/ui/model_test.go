package ui

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/viktor/asb-tui/internal/asb"
	"github.com/viktor/asb-tui/internal/config"
)

func testConfig() config.Config {
	return config.Config{Namespace: "demo.servicebus.windows.net", RefreshInterval: 5 * time.Second}
}

func TestNewModelInitializesState(t *testing.T) {
	m := NewModel(testConfig(), asb.AuthStatus{Ready: true, Message: "ok"}, nil, nil)

	if m.cfg.Namespace == "" {
		t.Fatal("expected namespace to be set")
	}
	if m.sortMode != sortByName {
		t.Fatalf("expected sortByName, got %d", m.sortMode)
	}
	if m.focus != focusList {
		t.Fatalf("expected list focus, got %d", m.focus)
	}
	if m.fetching {
		t.Fatal("expected fetching=false on init")
	}
}

func TestQueuesLoadedSuccessUpdatesState(t *testing.T) {
	m := NewModel(testConfig(), asb.AuthStatus{Ready: true, Message: "ok"}, nil, nil)
	m.lastError = "old"

	updated, _ := m.Update(queuesLoadedMsg{queues: []QueueMetrics{{Name: "orders", Active: 2}}})
	next := updated.(*Model)

	if next.lastError != "" {
		t.Fatalf("expected cleared error, got %q", next.lastError)
	}
	if next.lastSuccess.IsZero() {
		t.Fatal("expected lastSuccess to be set")
	}
	if len(next.filtered) != 1 {
		t.Fatalf("expected one queue, got %d", len(next.filtered))
	}
}

func TestQueuesLoadedErrorPreservesExistingData(t *testing.T) {
	m := NewModel(testConfig(), asb.AuthStatus{Ready: true, Message: "ok"}, nil, nil)
	m.queues = []QueueMetrics{{Name: "orders", Active: 1}}
	m.applyFilterAndSort()

	updated, _ := m.Update(queuesLoadedMsg{err: errors.New("boom")})
	next := updated.(*Model)

	if next.lastError == "" {
		t.Fatal("expected fetch error message")
	}
	if len(next.filtered) != 1 {
		t.Fatalf("expected existing queue data to remain, got %d", len(next.filtered))
	}
}

func TestFilterModeTypingAndExit(t *testing.T) {
	m := NewModel(testConfig(), asb.AuthStatus{Ready: true, Message: "ok"}, nil, nil)
	m.queues = []QueueMetrics{{Name: "orders"}, {Name: "billing"}}
	m.applyFilterAndSort()

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	next := updated.(*Model)
	if next.focus != focusFilter {
		t.Fatal("expected filter focus after '/'")
	}

	updated, _ = next.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	next = updated.(*Model)
	if next.filterInput.Value() != "o" {
		t.Fatalf("expected filter value 'o', got %q", next.filterInput.Value())
	}
	if len(next.filtered) != 1 {
		t.Fatalf("expected filtered list length 1, got %d", len(next.filtered))
	}

	updated, _ = next.Update(tea.KeyMsg{Type: tea.KeyEsc})
	next = updated.(*Model)
	if next.focus != focusList {
		t.Fatal("expected list focus after esc")
	}
}

func TestStatusBarIsSingleLine(t *testing.T) {
	m := NewModel(testConfig(), asb.AuthStatus{Ready: true, Message: "ok"}, nil, nil)
	m.width = 60
	m.lastError = "some very long error message that should get clipped"

	status := m.renderStatusBar()
	for _, r := range status {
		if r == '\n' {
			t.Fatal("status bar should be single line")
		}
	}
}

func TestKeyRRefreshesSelectedQueue(t *testing.T) {
	m := NewModel(
		testConfig(),
		asb.AuthStatus{Ready: true, Message: "ok"},
		func(context.Context) ([]QueueMetrics, error) { return nil, nil },
		func(context.Context, string) (QueueMetrics, error) { return QueueMetrics{}, nil },
	)
	m.queues = []QueueMetrics{{Name: "orders"}, {Name: "billing"}}
	m.applyFilterAndSort()

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	if cmd == nil {
		t.Fatal("expected refresh command")
	}
	if !m.fetching {
		t.Fatal("expected fetching true")
	}
	if !strings.HasPrefix(m.loadingTag, "one:") {
		t.Fatalf("expected one queue loading tag, got %q", m.loadingTag)
	}
}

func TestKeyShiftRRefreshesAllQueues(t *testing.T) {
	m := NewModel(
		testConfig(),
		asb.AuthStatus{Ready: true, Message: "ok"},
		func(context.Context) ([]QueueMetrics, error) { return nil, nil },
		nil,
	)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'R'}})
	if cmd == nil {
		t.Fatal("expected refresh command")
	}
	if !m.fetching {
		t.Fatal("expected fetching true")
	}
	if m.loadingTag != "all" {
		t.Fatalf("expected all loading tag, got %q", m.loadingTag)
	}
}

func TestThresholdExceeded(t *testing.T) {
	if thresholdExceeded(5, -1) {
		t.Fatal("threshold -1 disables warning")
	}
	if !thresholdExceeded(6, 5) {
		t.Fatal("expected threshold exceeded")
	}
	if thresholdExceeded(5, 5) {
		t.Fatal("expected strict greater-than")
	}
}
