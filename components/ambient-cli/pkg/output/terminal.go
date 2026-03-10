package output

import (
	"fmt"
	"io"
	"os"
	"time"

	"golang.org/x/term"
)

type fdWriter interface {
	Fd() uintptr
}

func fileDescriptor(w io.Writer) (int, bool) {
	if f, ok := w.(fdWriter); ok {
		return int(f.Fd()), true
	}
	return 0, false
}

func TerminalWidth() int {
	return TerminalWidthFor(os.Stdout)
}

func TerminalWidthFor(w io.Writer) int {
	fd, ok := fileDescriptor(w)
	if !ok {
		return 80
	}
	width, _, err := term.GetSize(fd)
	if err != nil || width <= 0 {
		return 80
	}
	return width
}

func IsTerminal() bool {
	return IsTerminalWriter(os.Stdout)
}

func IsTerminalWriter(w io.Writer) bool {
	fd, ok := fileDescriptor(w)
	if !ok {
		return false
	}
	return term.IsTerminal(fd)
}

// FormatAge formats a duration as human-readable age string (e.g., "2h", "15m", "3d")
func FormatAge(d time.Duration) string {
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}
