package ui

import (
	"context"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"

	"github.com/viktor/asb-tui/internal/asb"
	"github.com/viktor/asb-tui/internal/config"
	"github.com/viktor/asb-tui/internal/ui/style"
)

type tickMsg time.Time
type spinnerTickMsg struct{}

type queuesLoadedMsg struct {
	queues []QueueMetrics
	err    error
	manual bool
}

type queueLoadedMsg struct {
	queue     QueueMetrics
	err       error
	manual    bool
	queueName string
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
	m.setFocus(focusList)

	m.applyFilterAndSort()
	return m
}
