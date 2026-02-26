package output

import (
	"os"

	"golang.org/x/term"
)

func TerminalWidth() int {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || width <= 0 {
		return 80
	}
	return width
}

func IsTerminal() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}
