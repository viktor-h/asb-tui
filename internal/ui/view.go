package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

func (m *Model) View() string {
	if m.width <= 0 || m.height <= 0 {
		return "Loading UI..."
	}

	header := m.renderHeader()
	body := m.renderBody()
	help := m.renderHelpLine()
	status := m.renderStatusBar()

	return lipgloss.JoinVertical(lipgloss.Left, header, body, help, status)
}

func (m *Model) renderHeader() string {
	left := m.styles.Header.Render("ASB TUI") + " " + m.styles.HeaderBadge.Render(m.cfg.Namespace)

	filterLabel := m.filterInput.View()
	if m.focus != focusFilter && strings.TrimSpace(m.filterInput.Value()) == "" {
		filterLabel = m.styles.Muted.Render("filter: / to edit")
	}

	if m.width <= 0 {
		return left
	}
	available := m.width - lipgloss.Width(left) - 1
	if available <= 0 {
		return truncateSingleLine(left, m.width)
	}
	right := lipgloss.NewStyle().Width(available).Align(lipgloss.Right).Render(filterLabel)
	line := lipgloss.JoinHorizontal(lipgloss.Top, left, " ", right)
	return truncateSingleLine(line, m.width)
}

func (m *Model) renderBody() string {
	bodyHeight := max(6, m.height-4)
	if m.width < splitMinWidth {
		return m.renderQueueList(m.width, bodyHeight, m.focus == focusList)
	}

	leftWidth := m.width / 2
	rightWidth := m.width - leftWidth - 1
	left := m.renderQueueList(leftWidth, bodyHeight, m.focus == focusList)
	right := m.renderQueueDetail(rightWidth, bodyHeight)
	return lipgloss.JoinHorizontal(lipgloss.Top, left, " ", right)
}

func (m *Model) renderQueueList(width, height int, focused bool) string {
	pane := m.styles.Pane
	if focused {
		pane = m.styles.PaneFocused
	}

	title := m.styles.PaneTitle.Render("Queues")
	if m.fetching && len(m.filtered) == 0 {
		return pane.Width(width).Height(height).Render(title + "\n\n" + m.currentSpinner() + " Loading queue runtime properties...")
	}

	if len(m.filtered) == 0 {
		empty := "No queues found in namespace"
		if strings.TrimSpace(m.filterInput.Value()) != "" {
			empty = "No queues match current filter"
		}
		if m.lastError != "" {
			empty = "Unable to load queue data"
		}
		return pane.Width(width).Height(height).Render(title + "\n\n" + empty)
	}

	return pane.Width(width).Height(height).Render(title + "\n" + m.table.View())
}

func (m *Model) renderQueueDetail(width, height int) string {
	pane := m.styles.Pane
	if m.focus == focusDetail {
		pane = m.styles.PaneFocused
	}

	title := m.styles.PaneTitle.Render("Queue Detail")
	return pane.Width(width).Height(height).Render(title + "\n" + m.detail.View())
}

func (m *Model) renderHelpLine() string {
	line := m.help.View(m.keys)
	if !m.showHelp {
		return truncateSingleLine(line, m.width)
	}
	return line
}

func (m *Model) renderStatusBar() string {
	auth := "auth: ready"
	if !m.authStatus.Ready {
		auth = "auth: unavailable"
	}

	state := "state: fresh"
	if m.fetching {
		state = "state: loading " + m.currentSpinner()
		if m.loadingTag != "" {
			state += " [" + m.loadingTag + "]"
		}
	} else if m.isStale() {
		state = "state: stale"
	}

	lastSuccess := "last success: never"
	if !m.lastSuccess.IsZero() {
		lastSuccess = "last success: " + m.lastSuccess.Format("15:04:05")
	}

	errorText := ""
	if m.lastError != "" {
		age := ""
		if !m.lastErrorAt.IsZero() {
			age = fmt.Sprintf(" (%s ago)", time.Since(m.lastErrorAt).Round(time.Second))
		}
		errorText = "  err: " + clip(m.lastError, 40) + age
	}

	line := "STATUS  " + auth + "  " + state + "  refresh: " + m.cfg.RefreshInterval.String() +
		"  " + lastSuccess + "  sort: " + m.sortModeLabel() + errorText
	line = truncateSingleLine(line, m.width)
	return m.styles.StatusBar.Width(max(1, m.width)).Render(line)
}
