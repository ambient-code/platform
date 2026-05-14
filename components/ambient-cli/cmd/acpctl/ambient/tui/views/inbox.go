package views

import (
	"time"

	"github.com/charmbracelet/bubbles/table"

	sdktypes "github.com/ambient-code/platform/components/ambient-sdk/go-sdk/types"
)

// InboxColumns returns the column definitions for the inbox message list view.
// Column order matches the TUI spec: ID, FROM, BODY, READ, AGE.
func InboxColumns() []table.Column {
	return []table.Column{
		{Title: "ID", Width: 14},
		{Title: "FROM", Width: 15},
		{Title: "BODY", Width: 50},
		{Title: "READ", Width: 6},
		{Title: "AGE", Width: 8},
	}
}

// InboxRow converts an SDK InboxMessage into a table row suitable for the inbox
// list view. The now parameter is used to compute the relative AGE column.
//
// FROM displays the message's FromName, falling back to "(human)" when empty.
// READ displays "✓" for read messages and "—" for unread.
// BODY is truncated to 47 characters (50 column width minus ellipsis) to fit
// the column; the full body is available in the detail view.
func InboxRow(msg sdktypes.InboxMessage, now time.Time) table.Row {
	from := msg.FromName
	if from == "" {
		from = "(human)"
	}

	readIndicator := "—"
	if msg.Read {
		readIndicator = "✓"
	}

	age := ""
	if msg.CreatedAt != nil {
		age = FormatAge(now.Sub(*msg.CreatedAt))
	}

	body := TruncateString(msg.Body, 47)

	return table.Row{
		msg.ID,
		from,
		body,
		readIndicator,
		age,
	}
}

// NewInboxTable creates a ResourceTable configured for the inbox message list
// view. The scope parameter identifies which agent the inbox belongs to
// (e.g. "be"), matching the k9s title convention: inbox(be)[5].
func NewInboxTable(scope string, style TableStyle) ResourceTable {
	return NewResourceTable("inbox", scope, InboxColumns(), style)
}
