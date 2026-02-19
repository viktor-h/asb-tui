package ui

import "time"

func (m *Model) sortModeLabel() string {
	switch m.sortMode {
	case sortByActive:
		return "active"
	case sortByDLQ:
		return "dlq"
	default:
		return "name"
	}
}

func (m *Model) currentSpinner() string {
	if len(spinnerFrames) == 0 {
		return ""
	}
	return spinnerFrames[m.spinnerPos%len(spinnerFrames)]
}

func (m *Model) isStale() bool {
	if m.fetching {
		return false
	}
	if m.lastSuccess.IsZero() {
		return true
	}
	return time.Since(m.lastSuccess) > 2*m.cfg.RefreshInterval
}
