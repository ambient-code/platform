package output

import (
	"io"
	"os"

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
