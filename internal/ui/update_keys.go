package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) handleKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	if key.Matches(msg, m.keys.Quit) {
		return m, tea.Quit
	}
	if key.Matches(msg, m.keys.Help) {
		m.showHelp = !m.showHelp
		m.help.ShowAll = m.showHelp
		return m, nil
	}

	if m.focus == focusFilter {
		if msg.String() == "esc" || msg.String() == "enter" {
			m = m.setFocus(focusList)
			m.filterInput.Blur()
			m = m.applyFilterAndSort()
			return m, nil
		}

		var cmd tea.Cmd
		m.filterInput, cmd = m.filterInput.Update(msg)
		m = m.applyFilterAndSort()
		return m, cmd
	}

	if m.dlqPromptActive {
		return m.handleDLQPromptKey(msg)
	}

	if m.focus == focusDetail && m.dlqBodyViewer {
		if msg.String() == "esc" {
			m.dlqBodyViewer = false
			m.detail.GotoTop()
			return m, nil
		}
		var cmd tea.Cmd
		m.detail.SetContent(m.detailContent())
		m.detail, cmd = m.detail.Update(msg)
		return m, cmd
	}

	switch {
	case key.Matches(msg, m.keys.ToggleFocus):
		if m.width < splitMinWidth {
			return m, nil
		}
		switch m.focus {
		case focusList:
			m = m.setFocus(focusDetail)
		case focusDetail:
			m = m.setFocus(focusList)
		default:
			m = m.setFocus(focusList)
		}
		return m, nil
	case key.Matches(msg, m.keys.RefreshOne):
		if m.fetching {
			return m, nil
		}
		return m.refreshSelectedQueueCmd(true)
	case key.Matches(msg, m.keys.RefreshAll):
		if m.fetching {
			return m, nil
		}
		return m.refreshAllQueuesCmd(true)
	case key.Matches(msg, m.keys.FetchDLQ):
		if m.fetching {
			return m, nil
		}
		m = m.startDLQPrompt()
		return m, nil
	case key.Matches(msg, m.keys.CycleDLQ):
		m.dlqMode = nextDLQFetchMode(m.dlqMode)
		m.dlqBodyViewer = false
		m.dlqPromptActive = false
		m.dlqPromptError = ""
		m.dlqCountInput.Blur()
		return m, nil
	case key.Matches(msg, m.keys.OpenDLQBody):
		if m.focus == focusDetail && m.canOpenDLQBody() {
			m.dlqBodyViewer = true
			m.detail.GotoTop()
		}
		return m, nil
	case m.focus == focusDetail && m.canSelectDLQMessage() && key.Matches(msg, m.keys.Up):
		if m.dlqSelected > 0 {
			m.dlqSelected--
		}
		return m, nil
	case m.focus == focusDetail && m.canSelectDLQMessage() && key.Matches(msg, m.keys.Down):
		if m.dlqSelected < len(m.dlqMessages)-1 {
			m.dlqSelected++
		}
		return m, nil
	case key.Matches(msg, m.keys.Sort):
		m.sortMode = (m.sortMode + 1) % 3
		m = m.applyFilterAndSort()
		return m, nil
	case key.Matches(msg, m.keys.Filter):
		m = m.setFocus(focusFilter)
		m.filterInput.Focus()
		return m, nil
	case key.Matches(msg, m.keys.ClearFilter):
		m.filterInput.SetValue("")
		m = m.applyFilterAndSort()
		return m, nil
	case m.focus == focusDetail:
		var cmd tea.Cmd
		m.detail.SetContent(m.detailContent())
		m.detail, cmd = m.detail.Update(msg)
		return m, cmd
	default:
		var cmd tea.Cmd
		m.table, cmd = m.table.Update(msg)
		m.selected = m.table.Cursor()
		return m, cmd
	}
}

func (m Model) setFocus(f focusMode) Model {
	m.focus = f
	if f == focusList {
		m.table.Focus()
		return m
	}
	m.table.Blur()
	return m
}

func (m Model) startDLQPrompt() Model {
	if len(m.filtered) == 0 || m.fetchDLQ == nil {
		return m
	}
	m.dlqPromptActive = true
	m.dlqPromptError = ""
	m.dlqBodyViewer = false
	m.dlqCountInput.SetValue(strconv.Itoa(m.dlqFetchCount))
	m.dlqCountInput.Focus()
	return m
}

func (m Model) handleDLQPromptKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	if msg.String() == "esc" {
		m.dlqPromptActive = false
		m.dlqPromptError = ""
		m.dlqCountInput.Blur()
		return m, nil
	}

	if msg.String() == "enter" {
		count, err := strconv.Atoi(strings.TrimSpace(m.dlqCountInput.Value()))
		if err != nil {
			m.dlqPromptError = "Enter a whole number"
			return m, nil
		}
		if count < minDLQFetchCount || count > maxDLQFetchCount {
			m.dlqPromptError = fmt.Sprintf("Count must be between %d and %d", minDLQFetchCount, maxDLQFetchCount)
			return m, nil
		}

		m.dlqFetchCount = count
		m.dlqPromptActive = false
		m.dlqPromptError = ""
		m.dlqCountInput.Blur()
		return m.fetchSelectedDLQCmd()
	}

	var cmd tea.Cmd
	m.dlqCountInput, cmd = m.dlqCountInput.Update(msg)
	m.dlqPromptError = ""
	return m, cmd
}

func nextDLQFetchMode(mode string) string {
	switch normalizeDLQFetchMode(mode) {
	case dlqFetchModePeek:
		return dlqFetchModePeekLock
	case dlqFetchModePeekLock:
		return dlqFetchModeReceiveAndDelete
	default:
		return dlqFetchModePeek
	}
}

func normalizeDLQFetchMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case dlqFetchModePeek, dlqFetchModePeekLock, dlqFetchModeReceiveAndDelete:
		return strings.ToLower(strings.TrimSpace(mode))
	default:
		return dlqFetchModePeek
	}
}

func (m Model) canSelectDLQMessage() bool {
	if len(m.filtered) == 0 {
		return false
	}
	if m.dlqBodyViewer {
		return false
	}
	q := m.filtered[m.selected]
	return m.dlqQueueName == q.Name && len(m.dlqMessages) > 0
}

func (m Model) canOpenDLQBody() bool {
	return m.canSelectDLQMessage()
}

func emptyDash(value string) string {
	if strings.TrimSpace(value) == "" {
		return "-"
	}
	return value
}
