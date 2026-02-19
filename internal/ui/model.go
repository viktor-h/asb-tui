package ui

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/viktor/asb-tui/internal/asb"
	"github.com/viktor/asb-tui/internal/config"
	"github.com/viktor/asb-tui/internal/ui/style"
)

type tickMsg time.Time
type spinnerTickMsg struct{}

type queuesLoadedMsg struct {
	queues []QueueMetrics
	err    error

	requestedAt time.Time
	manual      bool
}

type queueLoadedMsg struct {
	queue       QueueMetrics
	err         error
	requestedAt time.Time
	manual      bool
	queueName   string
}

type sortMode int
type focusMode int

const (
	sortByName sortMode = iota
	sortByActive
	sortByDLQ
)

const (
	focusList focusMode = iota
	focusFilter
	focusDetail
)

var spinnerFrames = []string{"|", "/", "-", "\\"}

type keyMap struct {
	Up          key.Binding
	Down        key.Binding
	ToggleFocus key.Binding
	Filter      key.Binding
	ClearFilter key.Binding
	Sort        key.Binding
	RefreshOne  key.Binding
	RefreshAll  key.Binding
	Help        key.Binding
	Quit        key.Binding
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.ToggleFocus, k.Filter, k.RefreshOne, k.RefreshAll, k.Help, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{
		k.Up,
		k.Down,
		k.ToggleFocus,
		k.Filter,
		k.ClearFilter,
		k.Sort,
		k.RefreshOne,
		k.RefreshAll,
		k.Help,
		k.Quit,
	}}
}

type QueueMetrics struct {
	Name      string
	Active    int64
	Scheduled int64
	Dead      int64
	Transfer  int64
}

type Model struct {
	cfg         config.Config
	authStatus  asb.AuthStatus
	styles      style.Styles
	fetchQueues func(ctx context.Context) ([]QueueMetrics, error)
	fetchQueue  func(ctx context.Context, queueName string) (QueueMetrics, error)

	queues   []QueueMetrics
	filtered []QueueMetrics
	selected int

	sortMode   sortMode
	focus      focusMode
	showHelp   bool
	fetching   bool
	loadingTag string
	spinnerPos int

	filterInput textinput.Model
	table       table.Model
	detail      viewport.Model
	help        help.Model
	keys        keyMap

	lastRefreshAttempt time.Time
	lastSuccess        time.Time
	lastError          string
	lastErrorAt        time.Time

	width  int
	height int
}

func newKeyMap() keyMap {
	return keyMap{
		Up:          key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("k/up", "move up")),
		Down:        key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("j/down", "move down")),
		ToggleFocus: key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "focus pane")),
		Filter:      key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
		ClearFilter: key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "clear filter")),
		Sort:        key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "sort")),
		RefreshOne:  key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh one")),
		RefreshAll:  key.NewBinding(key.WithKeys("R"), key.WithHelp("R", "refresh all")),
		Help:        key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		Quit:        key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	}
}

func NewModel(
	cfg config.Config,
	authStatus asb.AuthStatus,
	fetchQueues func(ctx context.Context) ([]QueueMetrics, error),
	fetchQueue func(ctx context.Context, queueName string) (QueueMetrics, error),
) *Model {
	filter := textinput.New()
	filter.Placeholder = "filter queues"
	filter.Prompt = "filter> "
	filter.CharLimit = 80
	filter.Blur()

	helpModel := help.New()
	helpModel.ShowAll = false

	columns := []table.Column{
		{Title: "QUEUE", Width: 24},
		{Title: "ACTIVE", Width: 10},
		{Title: "DEADLETTER", Width: 10},
		{Title: "SCHEDULED", Width: 10},
	}
	tbl := table.New(
		table.WithColumns(columns),
		table.WithRows([]table.Row{}),
		table.WithFocused(true),
	)
	tbl.KeyMap.LineUp = key.NewBinding(key.WithKeys("up", "k"))
	tbl.KeyMap.LineDown = key.NewBinding(key.WithKeys("down", "j"))
	tbl.KeyMap.PageUp = key.NewBinding(key.WithKeys("pgup"))
	tbl.KeyMap.PageDown = key.NewBinding(key.WithKeys("pgdown"))
	tbl.KeyMap.HalfPageUp = key.NewBinding(key.WithKeys("u"))
	tbl.KeyMap.HalfPageDown = key.NewBinding(key.WithKeys("d"))
	tblStyles := table.DefaultStyles()
	tblStyles.Header = tblStyles.Header.Bold(true)
	tblStyles.Selected = tblStyles.Selected.Bold(true)
	tbl.SetStyles(tblStyles)

	detail := viewport.New(20, 8)
	detail.SetContent("Queue detail will appear here")

	m := &Model{
		cfg:                cfg,
		authStatus:         authStatus,
		styles:             style.New(),
		fetchQueues:        fetchQueues,
		fetchQueue:         fetchQueue,
		sortMode:           sortByName,
		focus:              focusList,
		filterInput:        filter,
		table:              tbl,
		detail:             detail,
		help:               helpModel,
		keys:               newKeyMap(),
		queues:             []QueueMetrics{},
		filtered:           []QueueMetrics{},
		spinnerPos:         0,
		showHelp:           false,
		fetching:           false,
		loadingTag:         "",
		selected:           0,
		lastError:          "",
		lastErrorAt:        time.Time{},
		lastSuccess:        time.Time{},
		lastRefreshAttempt: time.Time{},
	}

	m.applyFilterAndSort()
	return m
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(m.tickCmd(), m.refreshAllQueuesCmd(false))
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.Width = msg.Width
		m.resizeTable()
		m.resizeDetail()
		return m, nil
	case tickMsg:
		if m.fetching {
			return m, m.tickCmd()
		}
		return m, tea.Batch(m.tickCmd(), m.refreshAllQueuesCmd(false))
	case spinnerTickMsg:
		if !m.fetching {
			return m, nil
		}
		m.spinnerPos = (m.spinnerPos + 1) % len(spinnerFrames)
		return m, m.spinnerCmd()
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

func (m *Model) handleKey(msg tea.KeyMsg) tea.Cmd {
	if m.focus == focusFilter {
		if msg.String() == "esc" || msg.String() == "enter" {
			m.focus = focusList
			m.filterInput.Blur()
			m.table.Focus()
			m.applyFilterAndSort()
			return nil
		}

		var cmd tea.Cmd
		m.filterInput, cmd = m.filterInput.Update(msg)
		m.applyFilterAndSort()
		return cmd
	}

	switch {
	case key.Matches(msg, m.keys.Quit):
		return tea.Quit
	case key.Matches(msg, m.keys.Help):
		m.showHelp = !m.showHelp
		m.help.ShowAll = m.showHelp
		return nil
	case key.Matches(msg, m.keys.ToggleFocus):
		if m.width < 96 {
			return nil
		}
		switch m.focus {
		case focusList:
			m.focus = focusDetail
			m.table.Blur()
		case focusDetail:
			m.focus = focusList
			m.table.Focus()
		default:
			m.focus = focusList
			m.table.Focus()
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
	case key.Matches(msg, m.keys.Up), key.Matches(msg, m.keys.Down):
		var cmd tea.Cmd
		m.table, cmd = m.table.Update(msg)
		m.selected = m.table.Cursor()
		return cmd
	case key.Matches(msg, m.keys.Sort):
		m.sortMode = (m.sortMode + 1) % 3
		m.applyFilterAndSort()
		return nil
	case key.Matches(msg, m.keys.Filter):
		m.focus = focusFilter
		m.filterInput.Focus()
		m.table.Blur()
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
		m.table, cmd = m.table.Update(msg)
		m.selected = m.table.Cursor()
		return cmd
	}
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
	m.lastRefreshAttempt = time.Now()

	requestedAt := m.lastRefreshAttempt
	return tea.Batch(
		m.spinnerCmd(),
		func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
			defer cancel()

			queues, err := m.fetchQueues(ctx)
			return queuesLoadedMsg{
				queues:      queues,
				err:         err,
				requestedAt: requestedAt,
				manual:      manual,
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
	m.lastRefreshAttempt = time.Now()
	requestedAt := m.lastRefreshAttempt

	return tea.Batch(
		m.spinnerCmd(),
		func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
			defer cancel()

			queue, err := m.fetchQueue(ctx, queueName)
			return queueLoadedMsg{
				queue:       queue,
				err:         err,
				requestedAt: requestedAt,
				manual:      manual,
				queueName:   queueName,
			}
		},
	)
}

func (m *Model) tickCmd() tea.Cmd {
	return tea.Tick(m.cfg.RefreshInterval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m *Model) spinnerCmd() tea.Cmd {
	return tea.Tick(120*time.Millisecond, func(time.Time) tea.Msg {
		return spinnerTickMsg{}
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
		return
	}
	if m.selected >= len(m.filtered) {
		m.selected = len(m.filtered) - 1
	}
	m.syncTableRows()
}

func (m *Model) syncTableRows() {
	rows := make([]table.Row, 0, len(m.filtered))
	for _, q := range m.filtered {
		rows = append(rows, table.Row{
			clip(q.Name, queueNameColumnWidth(m.table.Width())),
			fmt.Sprintf("%d", q.Active),
			fmt.Sprintf("%d", q.Dead),
			fmt.Sprintf("%d", q.Scheduled),
		})
	}
	m.table.SetRows(rows)
	if len(rows) == 0 {
		m.table.SetCursor(0)
		m.selected = 0
		return
	}
	if m.selected >= len(rows) {
		m.selected = len(rows) - 1
	}
	m.table.SetCursor(m.selected)
}

func (m *Model) resizeTable() {
	bodyHeight := max(6, m.height-4)
	listHeight := bodyHeight - 2
	if listHeight < 3 {
		listHeight = 3
	}
	listWidth := m.width
	if m.width >= 96 {
		listWidth = m.width / 2
	}
	listWidth = max(24, listWidth-2)
	nameWidth := queueNameColumnWidth(listWidth)
	columns := []table.Column{
		{Title: "QUEUE", Width: nameWidth},
		{Title: "ACTIVE", Width: 10},
		{Title: "DEADLETTER", Width: 10},
		{Title: "SCHEDULED", Width: 10},
	}
	m.table.SetColumns(columns)
	m.table.SetWidth(listWidth)
	m.table.SetHeight(listHeight)
	m.syncTableRows()
}

func (m *Model) resizeDetail() {
	bodyHeight := max(6, m.height-4)
	detailHeight := bodyHeight - 2
	if detailHeight < 3 {
		detailHeight = 3
	}
	detailWidth := m.width
	if m.width >= 96 {
		detailWidth = m.width - (m.width / 2) - 1
	}
	detailWidth = max(24, detailWidth-2)
	m.detail.Width = detailWidth
	m.detail.Height = detailHeight
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
	if m.width < 96 {
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

	m.table.Focus()
	if !focused {
		m.table.Blur()
	}

	return pane.Width(width).Height(height).Render(title + "\n" + m.table.View())
}

func (m *Model) renderQueueDetail(width, height int) string {
	pane := m.styles.Pane
	if m.focus == focusDetail {
		pane = m.styles.PaneFocused
	}

	title := m.styles.PaneTitle.Render("Queue Detail")
	m.resizeDetail()

	if len(m.filtered) == 0 {
		msg := "Select a queue once data is available"
		if m.fetching {
			msg = "Waiting for first successful refresh"
		} else if m.lastError != "" {
			msg = "Check status bar for fetch errors"
		}
		m.detail.SetContent(msg)
		return pane.Width(width).Height(height).Render(title + "\n" + m.detail.View())
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
		m.styles.Muted.Render("Tip: press Tab to focus details, then use j/k or arrows to scroll."),
	}
	if m.focus == focusDetail {
		content = append(content,
			renderMetric(m.styles, "Updated", time.Now().Format("15:04:05")),
		)
	}

	m.detail.SetContent(strings.Join(content, "\n"))
	return pane.Width(width).Height(height).Render(title + "\n" + m.detail.View())
}

func (m *Model) renderHelpLine() string {
	m.help.ShowAll = m.showHelp
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
