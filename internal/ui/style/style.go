package style

import "github.com/charmbracelet/lipgloss"

type Styles struct {
	Header        lipgloss.Style
	HeaderBadge   lipgloss.Style
	Pane          lipgloss.Style
	PaneFocused   lipgloss.Style
	PaneTitle     lipgloss.Style
	Selected      lipgloss.Style
	RowWarn       lipgloss.Style
	Muted         lipgloss.Style
	Warning       lipgloss.Style
	Error         lipgloss.Style
	OK            lipgloss.Style
	MetricKey     lipgloss.Style
	MetricValue   lipgloss.Style
	HelpKey       lipgloss.Style
	HelpText      lipgloss.Style
	StatusBar     lipgloss.Style
	StatusLoading lipgloss.Style
	StatusStale   lipgloss.Style
}

func New() Styles {
	return Styles{
		Header: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("231")).
			Background(lipgloss.Color("24")).
			Padding(0, 1),
		HeaderBadge: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("24")).
			Background(lipgloss.Color("223")).
			Padding(0, 1),
		Pane: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1),
		PaneFocused: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("33")).
			Padding(0, 1),
		PaneTitle: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("223")),
		Selected: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("31")),
		RowWarn: lipgloss.NewStyle().
			Foreground(lipgloss.Color("215")),
		Muted: lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
		Warning: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("214")),
		Error: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("203")),
		OK: lipgloss.NewStyle().
			Foreground(lipgloss.Color("78")),
		MetricKey: lipgloss.NewStyle().
			Foreground(lipgloss.Color("250")),
		MetricValue: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("231")),
		HelpKey: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("223")),
		HelpText: lipgloss.NewStyle().
			Foreground(lipgloss.Color("250")),
		StatusBar: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("231")).
			Background(lipgloss.Color("25")).
			Padding(0, 0),
		StatusLoading: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("229")),
		StatusStale: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("215")),
	}
}
