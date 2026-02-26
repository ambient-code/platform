package output

import (
	"fmt"
	"io"
	"strings"
)

type Column struct {
	Name  string
	Width int
}

type Table struct {
	writer  io.Writer
	columns []Column
	padding int
}

func NewTable(writer io.Writer, columns []Column) *Table {
	return &Table{
		writer:  writer,
		columns: columns,
		padding: 3,
	}
}

func (t *Table) WriteHeaders() {
	termWidth := TerminalWidth()
	var parts []string
	for _, col := range t.columns {
		parts = append(parts, t.formatCell(col.Name, col.Width))
	}
	line := strings.Join(parts, strings.Repeat(" ", t.padding))
	if len(line) > termWidth && IsTerminal() {
		line = line[:termWidth]
	}
	fmt.Fprintln(t.writer, line)
}

func (t *Table) WriteRow(values ...string) {
	termWidth := TerminalWidth()
	var parts []string
	for i, val := range values {
		width := 0
		if i < len(t.columns) {
			width = t.columns[i].Width
		}
		parts = append(parts, t.formatCell(val, width))
	}
	line := strings.Join(parts, strings.Repeat(" ", t.padding))
	if len(line) > termWidth && IsTerminal() {
		line = line[:termWidth]
	}
	fmt.Fprintln(t.writer, line)
}

func (t *Table) formatCell(value string, width int) string {
	if width <= 0 {
		return value
	}
	if len(value) > width {
		if width > 3 {
			return value[:width-3] + "..."
		}
		return value[:width]
	}
	return value + strings.Repeat(" ", width-len(value))
}
