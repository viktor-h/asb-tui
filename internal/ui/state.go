package ui

import "time"

func (m Model) sortModeLabel() string {
	switch m.sortMode {
	case sortByActive:
		return "active"
	case sortByDLQ:
		return "dlq"
	default:
		return "name"
	}
}

func (m Model) currentSpinner() string {
	return m.spinner.View()
}

func (m Model) isStale() bool {
	if m.fetching {
		return false
	}
	if m.lastSuccess.IsZero() {
		return true
	}
	return time.Since(m.lastSuccess) > 2*m.cfg.RefreshInterval
}
