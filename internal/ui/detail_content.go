package ui

import (
	"fmt"
	"strings"
)

func (m Model) detailContent() string {
	if len(m.filtered) == 0 {
		msg := "Select a queue once data is available"
		if m.fetching {
			msg = "Waiting for refresh"
		} else if m.lastError != "" {
			msg = "Check status bar for fetch errors"
		}
		return msg
	}

	q := m.filtered[m.selected]

	if m.dlqPromptActive {
		content := []string{
			m.styles.Header.Render("Fetch Dead-letter Messages"),
			renderMetric(m.styles, "Queue", q.Name),
			renderMetric(m.styles, "Mode", m.dlqMode),
			renderMetric(m.styles, "Count range", fmt.Sprintf("%d-%d", minDLQFetchCount, maxDLQFetchCount)),
			"",
			m.dlqCountInput.View(),
			m.styles.Muted.Render("Enter to fetch, Esc to cancel."),
		}
		if m.dlqMode == dlqFetchModeReceiveAndDelete {
			content = append(content, m.styles.Warning.Render("Warning: receiveanddelete removes messages as they are fetched."))
		}
		if m.dlqPromptError != "" {
			content = append(content, m.styles.Warning.Render(m.dlqPromptError))
		}
		return strings.Join(content, "\n")
	}

	if m.dlqBodyViewer {
		if m.dlqQueueName == q.Name && len(m.dlqMessages) > 0 {
			selected := m.dlqSelected
			if selected < 0 {
				selected = 0
			}
			if selected >= len(m.dlqMessages) {
				selected = len(m.dlqMessages) - 1
			}
			msg := m.dlqMessages[selected]
			body, isJSON := formatMessageBody(msg.Body)
			bodyLabel := "Body"
			if isJSON {
				bodyLabel = "Body (pretty JSON)"
			}
			content := []string{
				m.styles.Header.Render(fmt.Sprintf("DLQ Message %d/%d", selected+1, len(m.dlqMessages))),
				renderMetric(m.styles, "Queue", q.Name),
				renderMetric(m.styles, "Message ID", msg.MessageID),
				renderMetric(m.styles, "Sequence", fmt.Sprintf("%d", msg.SequenceNumber)),
				renderMetric(m.styles, "Delivery Count", fmt.Sprintf("%d", msg.DeliveryCount)),
				renderMetric(m.styles, "Dead-letter reason", emptyDash(msg.DeadLetterReason)),
				renderMetric(m.styles, "Dead-letter error", emptyDash(msg.DeadLetterErrorDescription)),
				"",
				m.styles.MetricKey.Render(bodyLabel + ":"),
				body,
				"",
				m.styles.Muted.Render("Esc closes body view and returns to DLQ message list."),
			}
			return strings.Join(content, "\n")
		}
	}

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
		renderMetric(m.styles, "DLQ mode", m.dlqMode),
		renderMetric(m.styles, "DLQ fetch count", fmt.Sprintf("%d", m.dlqFetchCount)),
		"",
		m.styles.Muted.Render("Tip: Tab to focus details, then use arrows/j/k to scroll."),
		m.styles.Muted.Render("Tip: Press D to fetch DLQ and choose count, m to cycle mode."),
	}

	if m.dlqMode == dlqFetchModeReceiveAndDelete {
		content = append(content, m.styles.Warning.Render("Warning: receiveanddelete removes messages as they are fetched."))
	}

	if m.fetching && strings.HasPrefix(m.loadingTag, "dlq:") {
		content = append(content, m.styles.Muted.Render("Fetching dead-letter messages..."))
	}

	if m.dlqQueueName == q.Name {
		if m.dlqLastError != "" {
			content = append(content, m.styles.Warning.Render(clip(m.dlqLastError, 140)))
		} else {
			content = append(content, "", m.styles.Header.Render("Dead-letter messages"))
			if m.dlqFetchedAt.IsZero() {
				content = append(content, m.styles.Muted.Render("Press D to fetch dead-letter messages for this queue."))
			} else {
				content = append(content, m.styles.Muted.Render(fmt.Sprintf("Fetched %d message(s) at %s", len(m.dlqMessages), m.dlqFetchedAt.Format("15:04:05"))))
			}
			for i, msg := range m.dlqMessages {
				prefix := " "
				if i == m.dlqSelected {
					prefix = ">"
				}
				reason := msg.DeadLetterReason
				if reason == "" {
					reason = "-"
				}
				line := fmt.Sprintf("%s %d. id=%s seq=%d delivery=%d",
					prefix,
					i+1,
					clip(msg.MessageID, 24),
					msg.SequenceNumber,
					msg.DeliveryCount,
				)
				content = append(content, line)
				content = append(content, "   reason="+clip(strings.ReplaceAll(reason, "\n", " "), 56))
				if msg.DeadLetterErrorDescription != "" {
					content = append(content, "   error="+clip(strings.ReplaceAll(msg.DeadLetterErrorDescription, "\n", " "), 80))
				}
			}
			content = append(content, "", m.styles.Muted.Render("Use j/k to select a message, Enter to open full body."))
		}
	} else {
		content = append(content, "", m.styles.Muted.Render("Press D to fetch dead-letter messages for this queue."))
	}

	if !m.lastSuccess.IsZero() {
		content = append(content, renderMetric(m.styles, "Last refresh", m.lastSuccess.Format("15:04:05")))
	}

	return strings.Join(content, "\n")
}
