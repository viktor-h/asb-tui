package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/table"
)

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
