package views

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SortDirection represents the sort order for a column.
type SortDirection int

const (
	// SortNone means no sorting is applied on this column.
	SortNone SortDirection = iota
	// SortAsc sorts the column in ascending order.
	SortAsc
	// SortDesc sorts the column in descending order.
	SortDesc
)

// TableStyle holds the color and style values used to render the resource table.
// Pass this in from the parent package to avoid circular imports.
type TableStyle struct {
	// BorderColor is used for the title bar box-drawing characters.
	BorderColor lipgloss.Color
	// TitleColor is used for the resource kind text in the title.
	TitleColor lipgloss.Color
	// ScopeColor is used for the scope text in parentheses.
	ScopeColor lipgloss.Color
	// CountColor is used for the row count in the title.
	CountColor lipgloss.Color
	// DimColor is used for inactive/secondary elements.
	DimColor lipgloss.Color
	// HeaderColor is used for column header text.
	HeaderColor lipgloss.Color
	// SelectedBg is the background color for the selected row.
	SelectedBg lipgloss.Color
	// SelectedFg is the foreground color for the selected row.
	SelectedFg lipgloss.Color
}

// DefaultTableStyle returns a TableStyle using the project's orange-accent k9s palette.
func DefaultTableStyle() TableStyle {
	return TableStyle{
		BorderColor: lipgloss.Color("240"), // dim for border lines
		TitleColor:  lipgloss.Color("36"),  // cyan for resource kind
		ScopeColor:  lipgloss.Color("206"), // magenta/pink for scope
		CountColor:  lipgloss.Color("69"),  // blue for count
		DimColor:    lipgloss.Color("240"), // dim
		HeaderColor: lipgloss.Color("255"), // white
		SelectedBg:  lipgloss.Color("214"), // orange
		SelectedFg:  lipgloss.Color("0"),   // black on orange
	}
}

// sortState tracks which column is sorted and in what direction.
type sortState struct {
	colIdx    int
	direction SortDirection
}

// ResourceTable wraps bubbles/table.Model with k9s-style title bar,
// column sorting, and client-side filtering.
type ResourceTable struct {
	// inner is the wrapped bubbles table model.
	inner table.Model

	// kind is the resource kind displayed in the title (e.g. "agents", "sessions").
	kind string
	// scope is shown in parentheses in the title (e.g. "ambient-platform", "all").
	scope string

	// style controls rendering colors.
	style TableStyle

	// allRows holds the unfiltered data rows.
	allRows []table.Row
	// filterPredicate is the active client-side filter. Nil means no filter.
	filterPredicate func([]string) bool

	// sort tracks the current column sort state.
	sort sortState

	// columns stores the original column definitions for sort indicator rendering.
	columns []table.Column
}

// NewResourceTable creates a ResourceTable configured with the given resource kind,
// scope, columns, and style. The table starts focused and with no rows.
func NewResourceTable(kind string, scope string, columns []table.Column, style TableStyle) ResourceTable {
	// Store a copy of columns so we can modify titles for sort indicators
	// without mutating the caller's slice.
	cols := make([]table.Column, len(columns))
	copy(cols, columns)

	t := table.New(
		table.WithColumns(cols),
		table.WithFocused(true),
		table.WithHeight(1), // will be resized by the parent layout
	)

	// Apply k9s-inspired styles using the provided palette.
	s := table.DefaultStyles()
	s.Header = s.Header.
		Foreground(style.HeaderColor).
		Bold(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(style.BorderColor)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("0")).
		Background(style.SelectedBg).
		Bold(true)
	s.Cell = s.Cell.
		Foreground(style.HeaderColor)
	t.SetStyles(s)

	return ResourceTable{
		inner:   t,
		kind:    kind,
		scope:   scope,
		style:   style,
		columns: cols,
		sort: sortState{
			colIdx:    -1,
			direction: SortNone,
		},
	}
}

// Title returns the formatted k9s-style title string, e.g. "agents(ambient-platform)[12]".
func (rt *ResourceTable) Title() string {
	count := len(rt.inner.Rows())
	return fmt.Sprintf("%s(%s)[%d]", rt.kind, rt.scope, count)
}

// SetScope updates the scope shown in the title bar.
func (rt *ResourceTable) SetScope(scope string) {
	rt.scope = scope
}

// SetKind updates the resource kind shown in the title bar.
func (rt *ResourceTable) SetKind(kind string) {
	rt.kind = kind
}

// SetRows replaces all data rows. Filtering and sorting are re-applied.
func (rt *ResourceTable) SetRows(rows []table.Row) {
	rt.allRows = make([]table.Row, len(rows))
	copy(rt.allRows, rows)
	rt.applyFilterAndSort()
}

// SetFilter sets a client-side filter predicate. Rows for which the predicate
// returns false are hidden. The predicate receives the row as a []string
// (same as table.Row's underlying type). Pass nil to clear.
func (rt *ResourceTable) SetFilter(predicate func([]string) bool) {
	rt.filterPredicate = predicate
	rt.applyFilterAndSort()
}

// ClearFilter removes any active client-side filter.
func (rt *ResourceTable) ClearFilter() {
	rt.filterPredicate = nil
	rt.applyFilterAndSort()
}

// SortByColumn toggles column sort: none -> ascending -> descending -> none.
// Calling with the same column index cycles through the states.
// Calling with a different column index resets to ascending on the new column.
func (rt *ResourceTable) SortByColumn(colIdx int) {
	if colIdx < 0 || colIdx >= len(rt.columns) {
		return
	}

	if rt.sort.colIdx == colIdx {
		// Cycle: asc -> desc -> none
		switch rt.sort.direction {
		case SortNone:
			rt.sort.direction = SortAsc
		case SortAsc:
			rt.sort.direction = SortDesc
		case SortDesc:
			rt.sort.direction = SortNone
			rt.sort.colIdx = -1
		}
	} else {
		rt.sort.colIdx = colIdx
		rt.sort.direction = SortAsc
	}

	rt.updateColumnHeaders()
	rt.applyFilterAndSort()
}

// SortDirection returns the current sort column index and direction.
// Column index is -1 when no sort is active.
func (rt *ResourceTable) SortDirection() (colIdx int, dir SortDirection) {
	return rt.sort.colIdx, rt.sort.direction
}

// SelectedRow returns the currently highlighted row, or nil if the table is empty.
func (rt *ResourceTable) SelectedRow() table.Row {
	return rt.inner.SelectedRow()
}

// Cursor returns the index of the currently selected row.
func (rt *ResourceTable) Cursor() int {
	return rt.inner.Cursor()
}

// SetHeight sets the visible height of the table (number of data rows).
func (rt *ResourceTable) SetHeight(h int) {
	rt.inner.SetHeight(h)
}

// SetWidth sets the total width available for the table and redistributes
// column widths proportionally to fill the terminal.
func (rt *ResourceTable) SetWidth(w int) {
	rt.inner.SetWidth(w)

	usable := w - 4 // 2 for border chars, 2 for padding
	if usable < 10 || len(rt.columns) == 0 {
		return
	}

	// Calculate total base width from column definitions.
	totalBase := 0
	for _, c := range rt.columns {
		totalBase += c.Width
	}
	if totalBase == 0 {
		return
	}

	// Account for cell padding: each cell has Padding(0,1) = 2 chars per cell.
	cellPadding := len(rt.columns) * 2
	distributable := usable - cellPadding
	if distributable < len(rt.columns) {
		return
	}

	// Distribute proportionally.
	cols := rt.inner.Columns()
	assigned := 0
	for i := range cols {
		if i == len(cols)-1 {
			cols[i].Width = distributable - assigned
		} else {
			cols[i].Width = rt.columns[i].Width * distributable / totalBase
			assigned += cols[i].Width
		}
	}
	rt.inner.SetColumns(cols)

	// Update selected style to span the full row width.
	s := table.DefaultStyles()
	s.Header = s.Header.
		Foreground(rt.style.HeaderColor).
		Bold(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(rt.style.BorderColor)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("0")).
		Background(rt.style.SelectedBg).
		Bold(true).
		Width(usable)
	s.Cell = s.Cell.
		Foreground(rt.style.HeaderColor)
	rt.inner.SetStyles(s)
}

// Focus gives keyboard focus to the table.
func (rt *ResourceTable) Focus() {
	rt.inner.Focus()
}

// Blur removes keyboard focus from the table.
func (rt *ResourceTable) Blur() {
	rt.inner.Blur()
}

// Focused returns whether the table currently has keyboard focus.
func (rt *ResourceTable) Focused() bool {
	return rt.inner.Focused()
}

// Rows returns the currently visible (filtered + sorted) rows.
func (rt *ResourceTable) Rows() []table.Row {
	return rt.inner.Rows()
}

// Columns returns the current column definitions.
func (rt *ResourceTable) Columns() []table.Column {
	return rt.inner.Columns()
}

// Update delegates message handling to the inner bubbles/table and adds
// scroll-wheel support. Returns the updated ResourceTable and any command.
func (rt *ResourceTable) Update(msg tea.Msg) (ResourceTable, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.MouseMsg:
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			rt.inner.MoveUp(3)
			return *rt, nil
		case tea.MouseButtonWheelDown:
			rt.inner.MoveDown(3)
			return *rt, nil
		}
	}

	var cmd tea.Cmd
	rt.inner, cmd = rt.inner.Update(msg)
	return *rt, cmd
}

// View renders the table with a k9s-style title bar.
//
// The title bar format is:
//
//	+---- kind(scope)[count] ----+
//
// using box-drawing characters and the configured border color.
func (rt *ResourceTable) View() string {
	borderStyle := lipgloss.NewStyle().Foreground(rt.style.BorderColor)
	w := rt.inner.Width()
	if w < 4 {
		w = 80
	}

	titleBar := rt.renderTitleBar()
	tableView := rt.inner.View()

	// Wrap each table line with side borders.
	tableLines := strings.Split(tableView, "\n")
	var bordered []string
	for _, line := range tableLines {
		lineWidth := lipgloss.Width(line)
		pad := max(w-lineWidth-2, 0) // 2 for side border chars
		bordered = append(bordered,
			borderStyle.Render("│")+" "+line+strings.Repeat(" ", pad)+borderStyle.Render("│"))
	}

	// Bottom border.
	bottom := borderStyle.Render("└" + strings.Repeat("─", w-2) + "┘")

	return titleBar + "\n" + strings.Join(bordered, "\n") + "\n" + bottom
}

// renderTitleBar produces the k9s-style title line with box-drawing characters.
// The title is centered: ┌──── kind(scope)[count] ────┐
// kind=cyan, scope=magenta, count=blue (matching k9s colors).
func (rt *ResourceTable) renderTitleBar() string {
	borderStyle := lipgloss.NewStyle().Foreground(rt.style.BorderColor)
	kindStyle := lipgloss.NewStyle().Foreground(rt.style.TitleColor).Bold(true)
	scopeStyle := lipgloss.NewStyle().Foreground(rt.style.ScopeColor).Bold(true)
	countStyle := lipgloss.NewStyle().Foreground(rt.style.CountColor).Bold(true)

	count := len(rt.inner.Rows())
	titleRendered := " " +
		kindStyle.Render(rt.kind) +
		scopeStyle.Render("("+rt.scope+")") +
		countStyle.Render(fmt.Sprintf("[%d]", count)) +
		" "

	titleVisualWidth := lipgloss.Width(titleRendered)
	tableWidth := rt.inner.Width()
	if tableWidth < titleVisualWidth+6 {
		return borderStyle.Render("┌────") +
			titleRendered +
			borderStyle.Render("────┐")
	}

	// Center the title between the dashes.
	remaining := tableWidth - titleVisualWidth - 2 // 2 for corner chars
	leftDashes := remaining / 2
	rightDashes := remaining - leftDashes
	leftDashes = max(leftDashes, 1)
	rightDashes = max(rightDashes, 1)

	left := borderStyle.Render("┌" + strings.Repeat("─", leftDashes))
	right := borderStyle.Render(strings.Repeat("─", rightDashes) + "┐")

	return left + titleRendered + right
}

// updateColumnHeaders updates column titles with sort direction indicators.
func (rt *ResourceTable) updateColumnHeaders() {
	cols := make([]table.Column, len(rt.columns))
	for i, c := range rt.columns {
		col := table.Column{
			Title: c.Title,
			Width: c.Width,
		}
		if i == rt.sort.colIdx {
			switch rt.sort.direction {
			case SortAsc:
				col.Title = c.Title + "↑" // up arrow
			case SortDesc:
				col.Title = c.Title + "↓" // down arrow
			}
		}
		cols[i] = col
	}
	rt.inner.SetColumns(cols)
}

// applyFilterAndSort filters allRows with the predicate, sorts the result,
// and updates the inner table's visible rows.
func (rt *ResourceTable) applyFilterAndSort() {
	rows := rt.allRows

	// Apply filter.
	if rt.filterPredicate != nil {
		filtered := make([]table.Row, 0, len(rows))
		for _, row := range rows {
			if rt.filterPredicate([]string(row)) {
				filtered = append(filtered, row)
			}
		}
		rows = filtered
	}

	// Apply sort.
	if rt.sort.colIdx >= 0 && rt.sort.direction != SortNone {
		colIdx := rt.sort.colIdx
		ascending := rt.sort.direction == SortAsc

		sorted := make([]table.Row, len(rows))
		copy(sorted, rows)
		sort.SliceStable(sorted, func(i, j int) bool {
			a := cellValue(sorted[i], colIdx)
			b := cellValue(sorted[j], colIdx)
			if ascending {
				return a < b
			}
			return a > b
		})
		rows = sorted
	}

	// Preserve cursor position within bounds.
	cursor := rt.inner.Cursor()
	rt.inner.SetRows(rows)
	if cursor >= len(rows) && len(rows) > 0 {
		rt.inner.SetCursor(len(rows) - 1)
	}
}

// cellValue safely extracts a cell value from a row, returning empty string
// if the column index is out of range.
func cellValue(row table.Row, colIdx int) string {
	if colIdx < 0 || colIdx >= len(row) {
		return ""
	}
	return row[colIdx]
}
