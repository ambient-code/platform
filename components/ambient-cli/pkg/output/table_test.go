package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestFormatCellTruncatesWithEllipsis(t *testing.T) {
	tbl := &Table{}
	got := tbl.formatCell("abcdefghij", 7)
	if got != "abcd..." {
		t.Errorf("expected 'abcd...', got %q", got)
	}
}

func TestFormatCellPadsShortValues(t *testing.T) {
	tbl := &Table{}
	got := tbl.formatCell("hi", 5)
	if got != "hi   " {
		t.Errorf("expected 'hi   ', got %q", got)
	}
}

func TestFormatCellExactWidth(t *testing.T) {
	tbl := &Table{}
	got := tbl.formatCell("hello", 5)
	if got != "hello" {
		t.Errorf("expected 'hello', got %q", got)
	}
}

func TestFormatCellZeroWidth(t *testing.T) {
	tbl := &Table{}
	got := tbl.formatCell("anything", 0)
	if got != "anything" {
		t.Errorf("expected 'anything', got %q", got)
	}
}

func TestFormatCellRuneAware(t *testing.T) {
	tbl := &Table{}
	got := tbl.formatCell("日本語テスト", 5)
	if got != "日本..." {
		t.Errorf("expected '日本...', got %q", got)
	}
}

func TestNewTableWritesHeaders(t *testing.T) {
	var buf bytes.Buffer
	cols := []Column{
		{Name: "ID", Width: 10},
		{Name: "NAME", Width: 10},
	}
	tbl := NewTable(&buf, cols)
	tbl.WriteHeaders()

	line := strings.TrimSpace(buf.String())
	if !strings.HasPrefix(line, "ID") {
		t.Errorf("expected header to start with 'ID', got %q", line)
	}
	if !strings.Contains(line, "NAME") {
		t.Errorf("expected header to contain 'NAME', got %q", line)
	}
}

func TestNewTableWritesRows(t *testing.T) {
	var buf bytes.Buffer
	cols := []Column{
		{Name: "ID", Width: 10},
		{Name: "NAME", Width: 10},
	}
	tbl := NewTable(&buf, cols)
	tbl.WriteRow("abc123", "test-name")

	line := strings.TrimSpace(buf.String())
	if !strings.Contains(line, "abc123") {
		t.Errorf("expected row to contain 'abc123', got %q", line)
	}
	if !strings.Contains(line, "test-name") {
		t.Errorf("expected row to contain 'test-name', got %q", line)
	}
}

func TestNonTerminalWriterSkipsTruncation(t *testing.T) {
	var buf bytes.Buffer
	cols := []Column{
		{Name: "A", Width: 50},
		{Name: "B", Width: 50},
		{Name: "C", Width: 50},
	}
	tbl := NewTable(&buf, cols)
	tbl.WriteRow(strings.Repeat("x", 50), strings.Repeat("y", 50), strings.Repeat("z", 50))

	line := strings.TrimSpace(buf.String())
	if len(line) < 150 {
		t.Errorf("expected non-terminal output to not be truncated, got length %d", len(line))
	}
}
