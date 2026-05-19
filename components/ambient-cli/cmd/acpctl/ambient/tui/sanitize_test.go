package tui

import (
	"strings"
	"testing"
)

func TestSanitize_CSISequences(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"simple reset", "\x1b[0mhello", "hello"},
		{"color code", "\x1b[31mred\x1b[0m", "red"},
		{"bold + color", "\x1b[1;33mbold yellow\x1b[0m", "bold yellow"},
		{"cursor movement", "\x1b[2Jcleared", "cleared"},
		{"embedded CSI", "before\x1b[36mcyan\x1b[0mafter", "beforecyanafter"},
		{"multiple params", "\x1b[38;5;196mtext\x1b[0m", "text"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Sanitize(tt.input)
			if got != tt.want {
				t.Errorf("Sanitize(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSanitize_OSCSequences(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"window title BEL", "\x1b]0;My Title\x07text", "text"},
		{"window title ST", "\x1b]0;My Title\x1b\\text", "text"},
		{"hyperlink", "\x1b]8;;https://example.com\x1b\\click\x1b]8;;\x1b\\", "click"},
		{"embedded OSC", "before\x1b]2;title\x07after", "beforeafter"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Sanitize(tt.input)
			if got != tt.want {
				t.Errorf("Sanitize(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSanitize_C0ControlCharacters(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"null byte", "hel\x00lo", "hello"},
		{"bell", "alert\x07!", "alert!"},
		{"backspace", "ab\x08c", "abc"},
		{"form feed", "page\x0cbreak", "pagebreak"},
		{"carriage return", "over\rwrite", "overwrite"},
		{"escape alone", "esc\x1b here", "esc here"},
		{"mixed controls", "\x01\x02\x03text\x04\x05", "text"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Sanitize(tt.input)
			if got != tt.want {
				t.Errorf("Sanitize(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSanitize_C0PreservesTabAndNewline(t *testing.T) {
	input := "line1\n\tindented\nline3"
	got := Sanitize(input)
	if got != input {
		t.Errorf("Sanitize should preserve tabs and newlines: got %q, want %q", got, input)
	}
}

func TestSanitize_C1ControlCharacters(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"0x80 PAD", "a\x80b", "ab"},
		{"0x85 NEL", "a\x85b", "ab"},
		{"0x8E SS2", "a\x8Eb", "ab"},
		{"0x90 DCS", "a\x90b", "ab"},
		{"0x9B CSI intro", "a\x9Bb", "ab"},
		{"0x9C ST", "a\x9Cb", "ab"},
		{"0x9F APC", "a\x9Fb", "ab"},
		{"range boundary low", "a\x80b", "ab"},
		{"range boundary high", "a\x9Fb", "ab"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Sanitize(tt.input)
			if got != tt.want {
				t.Errorf("Sanitize(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSanitize_RegionTags(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"simple region", `["main"]content[""]`, "content"},
		{"named region", `before["sidebar"]middle[""]after`, "beforemiddleafter"},
		{"empty region id", `[""]text`, "text"},
		{"region with special chars", `["region-1_a"]text`, "text"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Sanitize(tt.input)
			if got != tt.want {
				t.Errorf("Sanitize(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSanitize_NormalTextPassthrough(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"plain ASCII", "Hello, World!"},
		{"numbers and punctuation", "Test 123 -- ok? Yes! @#$%^&*()"},
		{"multiline", "line 1\nline 2\nline 3"},
		{"tabs", "col1\tcol2\tcol3"},
		{"empty string", ""},
		{"spaces only", "   "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Sanitize(tt.input)
			if got != tt.input {
				t.Errorf("Sanitize(%q) = %q, want passthrough", tt.input, got)
			}
		})
	}
}

func TestSanitize_UnicodePassthrough(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"CJK characters", "你好世界"},
		{"emoji", "\U0001f680\U0001f525✨"},
		{"accented Latin", "éàüñ"},
		{"Arabic", "مرحبا"},
		{"mixed Unicode and ASCII", "Hello 世界! \U0001f44b"},
		{"right above C1 range", " ¡ÿ"}, // 0xA0, 0xA1, 0xFF are NOT C1
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Sanitize(tt.input)
			if got != tt.input {
				t.Errorf("Sanitize(%q) = %q, want passthrough", tt.input, got)
			}
		})
	}
}

func TestSanitize_EmptyString(t *testing.T) {
	got := Sanitize("")
	if got != "" {
		t.Errorf("Sanitize(\"\") = %q, want \"\"", got)
	}
}

func TestSanitize_MixedContent(t *testing.T) {
	// A realistic agent output line with ANSI colors, a region tag, and a stray control char.
	input := "\x1b[1;32m✔ Task complete\x1b[0m [\"status\"] result\x07\n"
	want := "✔ Task complete  result\n"
	got := Sanitize(input)
	if got != want {
		t.Errorf("Sanitize mixed content:\n  got  %q\n  want %q", got, want)
	}
}

func TestSanitize_OnlyControlChars(t *testing.T) {
	input := "\x00\x01\x02\x03\x04\x05\x06\x07\x08\x0b\x0c\x0d\x0e\x0f"
	got := Sanitize(input)
	if got != "" {
		t.Errorf("Sanitize(all control chars) = %q, want \"\"", got)
	}
}

func TestSanitize_BracketNotRegionTag(t *testing.T) {
	// Square brackets that don't match the region tag pattern should pass through.
	tests := []struct {
		name  string
		input string
	}{
		{"array index", "arr[0]"},
		{"no quotes", "[main]"},
		{"single quotes", "['main']"},
		{"unbalanced", `["open`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Sanitize(tt.input)
			if got != tt.input {
				t.Errorf("Sanitize(%q) = %q, want passthrough", tt.input, got)
			}
		})
	}
}

func TestSanitizeLines(t *testing.T) {
	lines := []string{
		"\x1b[31mred\x1b[0m",
		"normal text",
		"tab\there",
		"\x00null\x07bell",
		`["region"]tagged`,
	}
	got := SanitizeLines(lines)
	want := []string{
		"red",
		"normal text",
		"tab\there",
		"nullbell",
		"tagged",
	}
	if len(got) != len(want) {
		t.Fatalf("SanitizeLines returned %d lines, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("SanitizeLines[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestSanitizeLines_Empty(t *testing.T) {
	got := SanitizeLines([]string{})
	if len(got) != 0 {
		t.Errorf("SanitizeLines([]) returned %d elements, want 0", len(got))
	}
}

func TestSanitizeLines_PreservesOrder(t *testing.T) {
	lines := []string{"first", "second", "third"}
	got := SanitizeLines(lines)
	result := strings.Join(got, ",")
	if result != "first,second,third" {
		t.Errorf("SanitizeLines did not preserve order: got %q", result)
	}
}
