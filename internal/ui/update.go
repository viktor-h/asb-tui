package ui

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) Init() tea.Cmd {
	return tea.Batch(m.tickCmd(), m.refreshAllQueuesCmd(false))
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.width < splitMinWidth && m.focus == focusDetail {
			m.setFocus(focusList)
		}
		m.help.Width = msg.Width
		m.resizeTable()
		m.resizeDetail()
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
		return m, tea.Batch(m.tickCmd(), m.refreshAllQueuesCmd(false))
	case queuesLoadedMsg:
		return m, m.handleQueuesLoaded(msg)
	case queueLoadedMsg:
		return m, m.handleQueueLoaded(msg)
	case tea.KeyMsg:
		return m, m.handleKey(msg)
	default:
		return m, nil
	}
}

func (m *Model) handleKey(msg tea.KeyMsg) tea.Cmd {
	if key.Matches(msg, m.keys.Quit) {
		return tea.Quit
	}
	if key.Matches(msg, m.keys.Help) {
		m.showHelp = !m.showHelp
		m.help.ShowAll = m.showHelp
		return nil
	}

	if m.focus == focusFilter {
		if msg.String() == "esc" || msg.String() == "enter" {
			m.setFocus(focusList)
			m.filterInput.Blur()
			m.applyFilterAndSort()
			return nil
		}

		var cmd tea.Cmd
		m.filterInput, cmd = m.filterInput.Update(msg)
		m.applyFilterAndSort()
		return cmd
	}

	switch {
	case key.Matches(msg, m.keys.ToggleFocus):
		if m.width < splitMinWidth {
			return nil
		}
		switch m.focus {
		case focusList:
			m.setFocus(focusDetail)
		case focusDetail:
			m.setFocus(focusList)
		default:
			m.setFocus(focusList)
		}
		return nil
	case key.Matches(msg, m.keys.RefreshOne):
		if m.fetching {
			return nil
		}
		return m.refreshSelectedQueueCmd(true)
	case key.Matches(msg, m.keys.RefreshAll):
		if m.fetching {
			return nil
		}
		return m.refreshAllQueuesCmd(true)
	case key.Matches(msg, m.keys.Sort):
		m.sortMode = (m.sortMode + 1) % 3
		m.applyFilterAndSort()
		return nil
	case key.Matches(msg, m.keys.Filter):
		m.setFocus(focusFilter)
		m.filterInput.Focus()
		return nil
	case key.Matches(msg, m.keys.ClearFilter):
		m.filterInput.SetValue("")
		m.applyFilterAndSort()
		return nil
	case m.focus == focusDetail:
		var cmd tea.Cmd
		m.detail, cmd = m.detail.Update(msg)
		return cmd
	default:
		var cmd tea.Cmd
		prev := m.table.Cursor()
		m.table, cmd = m.table.Update(msg)
		m.selected = m.table.Cursor()
		if m.selected != prev {
			m.syncDetailContent()
		}
		return cmd
	}
}

func (m *Model) setFocus(f focusMode) {
	m.focus = f
	if f == focusList {
		m.table.Focus()
		return
	}
	m.table.Blur()
}

func (m *Model) handleQueuesLoaded(msg queuesLoadedMsg) tea.Cmd {
	m.fetching = false
	m.loadingTag = ""
	if msg.err != nil {
		prefix := "all:auto"
		if msg.manual {
			prefix = "all:manual"
		}
		m.lastError = fmt.Sprintf("%s refresh failed: %v", prefix, msg.err)
		m.lastErrorAt = time.Now()
		m.syncDetailContent()
		return nil
	}

	m.lastError = ""
	m.queues = msg.queues
	m.lastSuccess = time.Now()
	m.applyFilterAndSort()
	return nil
}

func (m *Model) handleQueueLoaded(msg queueLoadedMsg) tea.Cmd {
	m.fetching = false
	m.loadingTag = ""
	if msg.err != nil {
		prefix := "one:auto"
		if msg.manual {
			prefix = "one:manual"
		}
		m.lastError = fmt.Sprintf("%s refresh failed for %s: %v", prefix, msg.queueName, msg.err)
		m.lastErrorAt = time.Now()
		m.syncDetailContent()
		return nil
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
	m.applyFilterAndSort()
	return nil
}

func (m *Model) refreshAllQueuesCmd(manual bool) tea.Cmd {
	if m.fetchQueues == nil || m.fetching {
		return nil
	}

	m.fetching = true
	m.loadingTag = "all"
	m.syncDetailContent()

	return tea.Batch(
		m.spinner.Tick,
		func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
			defer cancel()

			queues, err := m.fetchQueues(ctx)
			return queuesLoadedMsg{
				queues: queues,
				err:    err,
				manual: manual,
			}
		},
	)
}

func (m *Model) refreshSelectedQueueCmd(manual bool) tea.Cmd {
	if m.fetching {
		return nil
	}
	if len(m.filtered) == 0 {
		return m.refreshAllQueuesCmd(manual)
	}
	if m.fetchQueue == nil {
		return m.refreshAllQueuesCmd(manual)
	}

	queueName := m.filtered[m.selected].Name
	m.fetching = true
	m.loadingTag = "one: " + queueName
	m.syncDetailContent()

	return tea.Batch(
		m.spinner.Tick,
		func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
			defer cancel()

			queue, err := m.fetchQueue(ctx, queueName)
			return queueLoadedMsg{
				queue:     queue,
				err:       err,
				manual:    manual,
				queueName: queueName,
			}
		},
	)
}

func (m *Model) tickCmd() tea.Cmd {
	return tea.Tick(m.cfg.RefreshInterval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m *Model) applyFilterAndSort() {
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
		m.syncDetailContent()
		return
	}
	if m.selected >= len(m.filtered) {
		m.selected = len(m.filtered) - 1
	}
	m.syncTableRows()
	m.syncDetailContent()
}

func (m *Model) syncDetailContent() {
	if len(m.filtered) == 0 {
		msg := "Select a queue once data is available"
		if m.fetching {
			msg = "Waiting for refresh"
		} else if m.lastError != "" {
			msg = "Check status bar for fetch errors"
		}
		m.detail.SetContent(msg)
		return
	}

	q := m.filtered[m.selected]
	activeStyle := m.styles.OK
	if thresholdExceeded(q.Active, m.cfg.ActiveWarnThreshold) {
		activeStyle = m.styles.Warning
	}
	dlqStyle := m.styles.OK
	if thresholdExceeded(q.Dead, m.cfg.DLQWarnThreshold) {
		dlqStyle = m.styles.Warning
	}

	content := []string{
		renderMetric(m.styles, "Name", q.Name),
		renderMetricStyled(m.styles, "Active", fmt.Sprintf("%d", q.Active), activeStyle),
		renderMetric(m.styles, "Scheduled", fmt.Sprintf("%d", q.Scheduled)),
		renderMetricStyled(m.styles, "Dead-letter", fmt.Sprintf("%d", q.Dead), dlqStyle),
		renderMetric(m.styles, "Transfer DLQ", fmt.Sprintf("%d", q.Transfer)),
		"",
		m.styles.Muted.Render("Tip: Tab to focus details, then use arrows/j/k to scroll."),
	}
	if !m.lastSuccess.IsZero() {
		content = append(content, renderMetric(m.styles, "Last refresh", m.lastSuccess.Format("15:04:05")))
	}
	m.detail.SetContent(strings.Join(content, "\n"))
}
