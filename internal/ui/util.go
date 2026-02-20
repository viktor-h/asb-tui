package ui

import (
	"encoding/json"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/viktor/asb-tui/internal/ui/style"
)

func thresholdExceeded(value, threshold int64) bool {
	return threshold >= 0 && value > threshold
}

func renderMetric(styles style.Styles, key, value string) string {
	return styles.MetricKey.Render(key+": ") + styles.MetricValue.Render(value)
}

func renderMetricStyled(styles style.Styles, key, value string, valueStyle lipgloss.Style) string {
	return styles.MetricKey.Render(key+": ") + valueStyle.Render(value)
}

func clip(input string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	runes := []rune(input)
	if len(runes) <= maxLen {
		return input
	}
	if maxLen <= 3 {
		return string(runes[:maxLen])
	}
	return string(runes[:maxLen-3]) + "..."
}

func truncateSingleLine(input string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	input = strings.ReplaceAll(input, "\n", " ")
	return clip(input, maxLen)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func queueNameColumnWidth(totalWidth int) int {
	reserved := 2 + 10 + 2 + 10 + 2 + 10
	width := totalWidth - reserved
	if width < 12 {
		return 12
	}
	if width > 42 {
		return 42
	}
	return width
}

func formatMessageBody(body string) (string, bool) {
	trimmed := strings.TrimSpace(body)
	if trimmed == "" {
		return "", false
	}

	var decoded any
	if err := json.Unmarshal([]byte(trimmed), &decoded); err != nil {
		return body, false
	}

	formatted, err := json.MarshalIndent(decoded, "", "  ")
	if err != nil {
		return body, false
	}

	return string(formatted), true
}

func clampDLQFetchCount(value int) int {
	if value <= 0 {
		return defaultDLQFetchCount
	}
	if value < minDLQFetchCount {
		return minDLQFetchCount
	}
	if value > maxDLQFetchCount {
		return maxDLQFetchCount
	}
	return value
}
