package views

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

// ANSI CSI sequences: ESC [ ... <final byte>
var viewsCsiRe = regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`)

// ANSI OSC sequences: ESC ] ... (terminated by BEL or ST)
var viewsOscRe = regexp.MustCompile(`\x1b\][^\x07\x1b]*(?:\x07|\x1b\\)`)

// lipgloss/tview region tags: ["regionid"]
var viewsRegionTagRe = regexp.MustCompile(`\["[^"]*"\]`)

// SanitizePayload strips dangerous content from agent-produced output before
// terminal rendering. It removes:
//   - ANSI CSI escape sequences (\x1b[...)
//   - ANSI OSC escape sequences (\x1b]...)
//   - C0 control characters (0x00-0x1F) except tab (0x09) and newline (0x0A)
//   - C1 control characters (0x80-0x9F)
//   - lipgloss/tview region tags (["..."])
//
// This is equivalent to the Sanitize function in the parent tui package,
// duplicated here to avoid a circular import.
func SanitizePayload(s string) string {
	s = viewsCsiRe.ReplaceAllString(s, "")
	s = viewsOscRe.ReplaceAllString(s, "")
	s = viewsRegionTagRe.ReplaceAllString(s, "")

	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		switch {
		case r == '\t' || r == '\n':
			b.WriteRune(r)
		case r <= 0x1F:
			// C0 control character — drop.
		case r >= 0x80 && r <= 0x9F:
			// C1 control character (valid 2-byte UTF-8 encoding) — drop.
		case r == utf8.RuneError && size == 1:
			if s[i] >= 0x80 && s[i] <= 0x9F {
				// C1 control byte — drop.
			} else {
				b.WriteByte(s[i])
			}
		default:
			b.WriteString(s[i : i+size])
		}
		i += size
	}
	return b.String()
}
