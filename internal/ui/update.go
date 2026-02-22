package ui

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
)

type initialRefreshMsg struct{}

func initialRefreshCmd() tea.Cmd {
	return func() tea.Msg {
		return initialRefreshMsg{}
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.tickCmd(), initialRefreshCmd())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case initialRefreshMsg:
		next, cmd := m.refreshAllQueuesCmd(false)
		return next, cmd
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.width < splitMinWidth && m.focus == focusDetail {
			m = m.setFocus(focusList)
		}
		m.help.Width = msg.Width
		m = m.resizeTable()
		m = m.resizeDetail()
		return m, nil
	case spinner.TickMsg:
		if !m.fetching {
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case tickMsg:
		if m.fetching {
			return m, m.tickCmd()
		}
		next, refreshCmd := m.refreshAllQueuesCmd(false)
		return next, tea.Batch(next.tickCmd(), refreshCmd)
	case queuesLoadedMsg:
		return m.handleQueuesLoaded(msg), nil
	case queueLoadedMsg:
		return m.handleQueueLoaded(msg), nil
	case dlqLoadedMsg:
		return m.handleDLQLoaded(msg), nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	default:
		return m, nil
	}
}

func (m Model) handleQueuesLoaded(msg queuesLoadedMsg) Model {
	m.fetching = false
	m.loadingTag = ""
	if msg.err != nil {
		prefix := "all:auto"
		if msg.manual {
			prefix = "all:manual"
		}
		m.lastError = fmt.Sprintf("%s refresh failed: %v", prefix, msg.err)
		m.lastErrorAt = time.Now()
		return m
	}

	m.lastError = ""
	m.queues = msg.queues
	m.lastSuccess = time.Now()
	return m.applyFilterAndSort()
}

func (m Model) handleQueueLoaded(msg queueLoadedMsg) Model {
	m.fetching = false
	m.loadingTag = ""
	if msg.err != nil {
		prefix := "one:auto"
		if msg.manual {
			prefix = "one:manual"
		}
		m.lastError = fmt.Sprintf("%s refresh failed for %s: %v", prefix, msg.queueName, msg.err)
		m.lastErrorAt = time.Now()
		return m
	}

	m.lastError = ""
	m.lastSuccess = time.Now()
	replaced := false
	for i := range m.queues {
		if m.queues[i].Name == msg.queue.Name {
			m.queues[i] = msg.queue
			replaced = true
			break
		}
	}
	if !replaced {
		m.queues = append(m.queues, msg.queue)
	}
	return m.applyFilterAndSort()
}

func (m Model) refreshAllQueuesCmd(manual bool) (Model, tea.Cmd) {
	if m.fetchQueues == nil || m.fetching {
		return m, nil
	}
	fetchQueues := m.fetchQueues

	m.fetching = true
	m.loadingTag = "all"

	return m, tea.Batch(
		m.spinner.Tick,
		func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
			defer cancel()

			queues, err := fetchQueues(ctx)
			return queuesLoadedMsg{
				queues: queues,
				err:    err,
				manual: manual,
			}
		},
	)
}

func (m Model) refreshSelectedQueueCmd(manual bool) (Model, tea.Cmd) {
	if m.fetching {
		return m, nil
	}
	if len(m.filtered) == 0 {
		return m.refreshAllQueuesCmd(manual)
	}
	if m.fetchQueue == nil {
		return m.refreshAllQueuesCmd(manual)
	}

	queueName := m.filtered[m.selected].Name
	fetchQueue := m.fetchQueue
	m.fetching = true
	m.loadingTag = "one: " + queueName

	return m, tea.Batch(
		m.spinner.Tick,
		func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
			defer cancel()

			queue, err := fetchQueue(ctx, queueName)
			return queueLoadedMsg{
				queue:     queue,
				err:       err,
				manual:    manual,
				queueName: queueName,
			}
		},
	)
}

func (m Model) tickCmd() tea.Cmd {
	return tea.Tick(m.cfg.RefreshInterval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m Model) handleDLQLoaded(msg dlqLoadedMsg) Model {
	m.fetching = false
	m.loadingTag = ""
	if msg.err != nil {
		m.dlqQueueName = msg.queueName
		m.dlqLastError = fmt.Sprintf("dlq fetch failed for %s (%s): %v", msg.queueName, msg.mode, msg.err)
		m.lastError = m.dlqLastError
		m.lastErrorAt = time.Now()
		return m
	}

	m.lastError = ""
	m.dlqLastError = ""
	m.lastSuccess = time.Now()
	m.dlqFetchedAt = time.Now()
	m.dlqQueueName = msg.queueName
	m.dlqMessages = msg.messages
	m.dlqSelected = 0
	m.dlqBodyViewer = false
	return m
}

func (m Model) fetchSelectedDLQCmd() (Model, tea.Cmd) {
	if m.fetching {
		return m, nil
	}
	if m.fetchDLQ == nil || len(m.filtered) == 0 {
		return m, nil
	}

	queueName := m.filtered[m.selected].Name
	mode := m.dlqMode
	count := m.dlqFetchCount
	fetchDLQ := m.fetchDLQ
	m.fetching = true
	m.loadingTag = "dlq: " + queueName

	return m, tea.Batch(
		m.spinner.Tick,
		func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
			defer cancel()

			messages, err := fetchDLQ(ctx, queueName, mode, count)
			return dlqLoadedMsg{
				queueName: queueName,
				mode:      mode,
				messages:  messages,
				err:       err,
			}
		},
	)
}

func (m Model) applyFilterAndSort() Model {
	needle := strings.ToLower(strings.TrimSpace(m.filterInput.Value()))
	filtered := make([]QueueMetrics, 0, len(m.queues))

	for _, q := range m.queues {
		if needle == "" || strings.Contains(strings.ToLower(q.Name), needle) {
			filtered = append(filtered, q)
		}
	}

	sort.Slice(filtered, func(i, j int) bool {
		switch m.sortMode {
		case sortByActive:
			if filtered[i].Active == filtered[j].Active {
				return filtered[i].Name < filtered[j].Name
			}
			return filtered[i].Active > filtered[j].Active
		case sortByDLQ:
			if filtered[i].Dead == filtered[j].Dead {
				return filtered[i].Name < filtered[j].Name
			}
			return filtered[i].Dead > filtered[j].Dead
		default:
			return filtered[i].Name < filtered[j].Name
		}
	})

	m.filtered = filtered
	if len(m.filtered) == 0 {
		m.selected = 0
		m.table.SetRows([]table.Row{})
		m.table.SetCursor(0)
		return m
	}
	if m.selected >= len(m.filtered) {
		m.selected = len(m.filtered) - 1
	}
	return m.syncTableRows()
}
