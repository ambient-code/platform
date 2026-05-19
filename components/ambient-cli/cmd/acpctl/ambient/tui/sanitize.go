package tui

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

// ANSI CSI sequences: ESC [ ... <final byte>
// Matches sequences like \x1b[0m, \x1b[31;1m, \x1b[2J, etc.
var csiRe = regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`)

// ANSI OSC sequences: ESC ] ... (terminated by BEL or ST)
// Matches sequences like \x1b]0;title\a, \x1b]8;;url\x1b\\, etc.
var oscRe = regexp.MustCompile(`\x1b\][^\x07\x1b]*(?:\x07|\x1b\\)`)

// lipgloss/tview region tags: ["regionid"]
var regionTagRe = regexp.MustCompile(`\["[^"]*"\]`)

// Sanitize strips dangerous content from agent-produced output before
// terminal rendering. It removes:
//   - ANSI CSI escape sequences (\x1b[...)
//   - ANSI OSC escape sequences (\x1b]...)
//   - C0 control characters (0x00-0x1F) except tab (0x09) and newline (0x0A)
//   - C1 control characters (0x80-0x9F)
//   - lipgloss/tview region tags (["..."])
func Sanitize(s string) string {
	// Strip ANSI CSI sequences.
	s = csiRe.ReplaceAllString(s, "")

	// Strip ANSI OSC sequences.
	s = oscRe.ReplaceAllString(s, "")

	// Strip region tags.
	s = regionTagRe.ReplaceAllString(s, "")

	// Strip C0 control characters (except \t and \n) and C1 control characters.
	// We use utf8.DecodeRune to properly handle multi-byte UTF-8 sequences
	// (whose continuation bytes overlap with the C1 range 0x80-0x9F).
	// Invalid single bytes in 0x80-0x9F are detected via the replacement
	// character with a width of 1.
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
			// Invalid byte; check if it falls in the C1 range.
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

// SanitizeLines applies Sanitize to each line and returns the results.
func SanitizeLines(lines []string) []string {
	out := make([]string, len(lines))
	for i, line := range lines {
		out[i] = Sanitize(line)
	}
	return out
}
