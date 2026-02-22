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
	m := NewModel(testConfig(), asb.AuthStatus{Ready: true, Message: "ok"}, nil, nil, nil)

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
	m := NewModel(testConfig(), asb.AuthStatus{Ready: true, Message: "ok"}, nil, nil, nil)
	m.lastError = "old"

	updated, _ := m.Update(queuesLoadedMsg{queues: []QueueMetrics{{Name: "orders", Active: 2}}})
	next := updated.(Model)

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
	m := NewModel(testConfig(), asb.AuthStatus{Ready: true, Message: "ok"}, nil, nil, nil)
	m.queues = []QueueMetrics{{Name: "orders", Active: 1}}
	m = m.applyFilterAndSort()

	updated, _ := m.Update(queuesLoadedMsg{err: errors.New("boom")})
	next := updated.(Model)

	if next.lastError == "" {
		t.Fatal("expected fetch error message")
	}
	if len(next.filtered) != 1 {
		t.Fatalf("expected existing queue data to remain, got %d", len(next.filtered))
	}
}

func TestFilterModeTypingAndExit(t *testing.T) {
	m := NewModel(testConfig(), asb.AuthStatus{Ready: true, Message: "ok"}, nil, nil, nil)
	m.queues = []QueueMetrics{{Name: "orders"}, {Name: "billing"}}
	m = m.applyFilterAndSort()

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	next := updated.(Model)
	if next.focus != focusFilter {
		t.Fatal("expected filter focus after '/'")
	}

	updated, _ = next.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	next = updated.(Model)
	if next.filterInput.Value() != "o" {
		t.Fatalf("expected filter value 'o', got %q", next.filterInput.Value())
	}
	if len(next.filtered) != 1 {
		t.Fatalf("expected filtered list length 1, got %d", len(next.filtered))
	}

	updated, _ = next.Update(tea.KeyMsg{Type: tea.KeyEsc})
	next = updated.(Model)
	if next.focus != focusList {
		t.Fatal("expected list focus after esc")
	}
}

func TestStatusBarIsSingleLine(t *testing.T) {
	m := NewModel(testConfig(), asb.AuthStatus{Ready: true, Message: "ok"}, nil, nil, nil)
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
		nil,
	)
	m.queues = []QueueMetrics{{Name: "orders"}, {Name: "billing"}}
	m = m.applyFilterAndSort()

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	next := updated.(Model)
	if cmd == nil {
		t.Fatal("expected refresh command")
	}
	if !next.fetching {
		t.Fatal("expected fetching true")
	}
	if !strings.HasPrefix(next.loadingTag, "one:") {
		t.Fatalf("expected one queue loading tag, got %q", next.loadingTag)
	}
}

func TestKeyShiftRRefreshesAllQueues(t *testing.T) {
	m := NewModel(
		testConfig(),
		asb.AuthStatus{Ready: true, Message: "ok"},
		func(context.Context) ([]QueueMetrics, error) { return nil, nil },
		nil,
		nil,
	)

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'R'}})
	next := updated.(Model)
	if cmd == nil {
		t.Fatal("expected refresh command")
	}
	if !next.fetching {
		t.Fatal("expected fetching true")
	}
	if next.loadingTag != "all" {
		t.Fatalf("expected all loading tag, got %q", next.loadingTag)
	}
}

func TestDetailFocusRoutesNavigationToViewport(t *testing.T) {
	m := NewModel(testConfig(), asb.AuthStatus{Ready: true, Message: "ok"}, nil, nil, nil)
	m.width = 120
	m.height = 40
	m = m.resizeTable()
	m = m.resizeDetail()
	m.queues = []QueueMetrics{{Name: "orders", Active: 1}, {Name: "billing", Active: 2}}
	m = m.applyFilterAndSort()
	m.selected = 1
	m.table.SetCursor(1)
	m = m.setFocus(focusDetail)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	next := updated.(Model)
	if next.selected != 1 {
		t.Fatalf("expected table selection unchanged in detail focus, got %d", next.selected)
	}
}

func TestResizeNarrowForcesListFocus(t *testing.T) {
	m := NewModel(testConfig(), asb.AuthStatus{Ready: true, Message: "ok"}, nil, nil, nil)
	m = m.setFocus(focusDetail)

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 30})
	next := updated.(Model)
	if next.focus != focusList {
		t.Fatalf("expected focusList after narrow resize, got %d", next.focus)
	}
}

func TestGlobalHelpAndQuitWorkInFilterMode(t *testing.T) {
	m := NewModel(testConfig(), asb.AuthStatus{Ready: true, Message: "ok"}, nil, nil, nil)
	m = m.setFocus(focusFilter)
	m.filterInput.Focus()

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	next := updated.(Model)
	if !next.showHelp {
		t.Fatal("expected help to toggle while in filter mode")
	}

	_, cmd := next.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatal("expected quit command while in filter mode")
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

func TestDefaultDLQModeIsPeek(t *testing.T) {
	m := NewModel(testConfig(), asb.AuthStatus{Ready: true, Message: "ok"}, nil, nil, nil)
	if m.dlqMode != dlqFetchModePeek {
		t.Fatalf("expected default dlq mode peek, got %q", m.dlqMode)
	}
}

func TestCycleDLQMode(t *testing.T) {
	m := NewModel(testConfig(), asb.AuthStatus{Ready: true, Message: "ok"}, nil, nil, nil)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	m = updated.(Model)
	if m.dlqMode != dlqFetchModePeekLock {
		t.Fatalf("expected peeklock, got %q", m.dlqMode)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	m = updated.(Model)
	if m.dlqMode != dlqFetchModeReceiveAndDelete {
		t.Fatalf("expected receiveanddelete, got %q", m.dlqMode)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	m = updated.(Model)
	if m.dlqMode != dlqFetchModePeek {
		t.Fatalf("expected peek, got %q", m.dlqMode)
	}
}

func TestReceiveAndDeleteWarningInDetail(t *testing.T) {
	m := NewModel(testConfig(), asb.AuthStatus{Ready: true, Message: "ok"}, nil, nil, nil)
	m.queues = []QueueMetrics{{Name: "orders", Dead: 1}}
	m = m.applyFilterAndSort()
	m.detail.Height = 100
	m.detail.Width = 200
	m.dlqMode = dlqFetchModeReceiveAndDelete
	m.detail.SetContent(m.detailContent())

	if !strings.Contains(m.detail.View(), "receiveanddelete removes messages") {
		t.Fatal("expected receiveanddelete warning in detail pane")
	}
}

func TestDLQFetchOpensPromptWithPrefilledCount(t *testing.T) {
	m := NewModel(
		testConfig(),
		asb.AuthStatus{Ready: true, Message: "ok"},
		func(context.Context) ([]QueueMetrics, error) { return nil, nil },
		nil,
		func(context.Context, string, string, int) ([]DLQMessage, error) { return nil, nil },
	)
	m.queues = []QueueMetrics{{Name: "orders"}}
	m = m.applyFilterAndSort()

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'D'}})
	m = updated.(Model)
	if cmd != nil {
		t.Fatal("expected D to open prompt only")
	}
	if !m.dlqPromptActive {
		t.Fatal("expected prompt to be active")
	}
	if m.fetching {
		t.Fatal("expected not fetching before entering prompt value")
	}
	if m.dlqCountInput.Value() != "10" {
		t.Fatalf("expected prefilled count 10, got %q", m.dlqCountInput.Value())
	}
}

func TestDLQPromptEnterStartsFetch(t *testing.T) {
	m := NewModel(
		testConfig(),
		asb.AuthStatus{Ready: true, Message: "ok"},
		func(context.Context) ([]QueueMetrics, error) { return nil, nil },
		nil,
		func(context.Context, string, string, int) ([]DLQMessage, error) { return nil, nil },
	)
	m.queues = []QueueMetrics{{Name: "orders"}}
	m = m.applyFilterAndSort()

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'D'}})
	m = updated.(Model)
	m.dlqCountInput.SetValue("23")

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if cmd == nil {
		t.Fatal("expected enter to start fetch")
	}
	if m.dlqPromptActive {
		t.Fatal("expected prompt to close after valid enter")
	}
	if m.dlqFetchCount != 23 {
		t.Fatalf("expected count 23, got %d", m.dlqFetchCount)
	}
	if !m.fetching {
		t.Fatal("expected fetching true after enter")
	}
}

func TestDLQPromptEscCancels(t *testing.T) {
	m := NewModel(testConfig(), asb.AuthStatus{Ready: true, Message: "ok"}, nil, nil, nil)
	m.queues = []QueueMetrics{{Name: "orders"}}
	m = m.applyFilterAndSort()
	m = m.startDLQPrompt()

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	if cmd != nil {
		t.Fatal("expected no command on prompt cancel")
	}
	if m.dlqPromptActive {
		t.Fatal("expected prompt to close on esc")
	}
}

func TestEnterOpensDLQBodyAndEscCloses(t *testing.T) {
	m := NewModel(testConfig(), asb.AuthStatus{Ready: true, Message: "ok"}, nil, nil, nil)
	m.width = 120
	m.height = 40
	m = m.resizeTable()
	m = m.resizeDetail()
	m.queues = []QueueMetrics{{Name: "orders", Dead: 2}}
	m = m.applyFilterAndSort()
	m.dlqQueueName = "orders"
	m.dlqMessages = []DLQMessage{{MessageID: "m1", Body: "{\"kind\":\"invoice\",\"id\":7}"}}
	m.dlqSelected = 0
	m = m.setFocus(focusDetail)
	m.detail.SetContent(m.detailContent())

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if !m.dlqBodyViewer {
		t.Fatal("expected enter to open body viewer")
	}
	if !strings.Contains(m.detailContent(), "Body (pretty JSON)") {
		t.Fatal("expected pretty JSON body label")
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	if m.dlqBodyViewer {
		t.Fatal("expected esc to close body viewer")
	}
	if !strings.Contains(m.detailContent(), "Dead-letter messages") {
		t.Fatal("expected detail to return to message list")
	}
}

func TestDetailFocusJKSelectsDLQMessage(t *testing.T) {
	m := NewModel(testConfig(), asb.AuthStatus{Ready: true, Message: "ok"}, nil, nil, nil)
	m.width = 120
	m.height = 40
	m = m.resizeTable()
	m = m.resizeDetail()
	m.queues = []QueueMetrics{{Name: "orders", Dead: 2}}
	m = m.applyFilterAndSort()
	m.dlqQueueName = "orders"
	m.dlqMessages = []DLQMessage{{MessageID: "m1", Body: "a"}, {MessageID: "m2", Body: "b"}}
	m.dlqSelected = 0
	m = m.setFocus(focusDetail)
	m.detail.SetContent(m.detailContent())

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = updated.(Model)
	if m.dlqSelected != 1 {
		t.Fatalf("expected selected index 1, got %d", m.dlqSelected)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = updated.(Model)
	if m.dlqSelected != 0 {
		t.Fatalf("expected selected index 0, got %d", m.dlqSelected)
	}
}

func TestDLQListShowsReasonPreview(t *testing.T) {
	m := NewModel(testConfig(), asb.AuthStatus{Ready: true, Message: "ok"}, nil, nil, nil)
	m.width = 120
	m.height = 40
	m = m.resizeTable()
	m = m.resizeDetail()
	m.queues = []QueueMetrics{{Name: "orders", Dead: 1}}
	m = m.applyFilterAndSort()
	m.dlqQueueName = "orders"
	m.dlqMessages = []DLQMessage{{
		MessageID:        "m1",
		DeadLetterReason: "ValidationFailed: missing customer id",
		Body:             `{"event":"order.submitted"}`,
	}}
	m.detail.SetContent(m.detailContent())

	detail := m.detail.View()
	if !strings.Contains(detail, "reason=ValidationFailed") {
		t.Fatalf("expected reason preview in detail, got: %q", detail)
	}
}
