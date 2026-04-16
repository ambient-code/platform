package handlers

import (
	"strings"
	"testing"

	"ambient-code-backend/types"
)

func TestParseExtractionConfig(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantNil   bool
		wantModel string
		wantMax   int
		wantMin   int
	}{
		{
			name:      "fully enabled with custom settings",
			input:     `{"learning":{"enabled":true,"extraction":{"enabled":true,"model":"claude-haiku-4","maxMemoriesPerSession":3,"minTurnThreshold":10}}}`,
			wantNil:   false,
			wantModel: "claude-haiku-4",
			wantMax:   3,
			wantMin:   10,
		},
		{
			name:      "enabled with defaults",
			input:     `{"learning":{"enabled":true,"extraction":{"enabled":true}}}`,
			wantNil:   false,
			wantModel: defaultExtractionModel,
			wantMax:   defaultMaxMemoriesPerSession,
			wantMin:   defaultMinTurnThreshold,
		},
		{
			name:    "learning disabled",
			input:   `{"learning":{"enabled":false,"extraction":{"enabled":true}}}`,
			wantNil: true,
		},
		{
			name:    "extraction disabled",
			input:   `{"learning":{"enabled":true,"extraction":{"enabled":false}}}`,
			wantNil: true,
		},
		{
			name:    "no extraction key",
			input:   `{"learning":{"enabled":true}}`,
			wantNil: true,
		},
		{
			name:    "no learning key",
			input:   `{}`,
			wantNil: true,
		},
		{
			name:    "invalid JSON",
			input:   `not json`,
			wantNil: true,
		},
		{
			name:    "empty string",
			input:   ``,
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := parseExtractionConfig([]byte(tt.input))
			if tt.wantNil {
				if cfg != nil {
					t.Errorf("expected nil config, got %+v", cfg)
				}
				return
			}
			if cfg == nil {
				t.Fatal("expected non-nil config, got nil")
			}
			if cfg.Model != tt.wantModel {
				t.Errorf("model: got %q, want %q", cfg.Model, tt.wantModel)
			}
			if cfg.MaxMemoriesPerSession != tt.wantMax {
				t.Errorf("maxMemoriesPerSession: got %d, want %d", cfg.MaxMemoriesPerSession, tt.wantMax)
			}
			if cfg.MinTurnThreshold != tt.wantMin {
				t.Errorf("minTurnThreshold: got %d, want %d", cfg.MinTurnThreshold, tt.wantMin)
			}
		})
	}
}

func TestCountUserTurns(t *testing.T) {
	tests := []struct {
		name   string
		events []map[string]interface{}
		want   int
	}{
		{
			name: "messages snapshot with 3 user turns",
			events: []map[string]interface{}{
				{"type": types.EventTypeMessagesSnapshot, "messages": []interface{}{
					map[string]interface{}{"role": types.RoleUser, "content": "hello"},
					map[string]interface{}{"role": types.RoleAssistant, "content": "hi"},
					map[string]interface{}{"role": types.RoleUser, "content": "help me"},
					map[string]interface{}{"role": types.RoleAssistant, "content": "sure"},
					map[string]interface{}{"role": types.RoleUser, "content": "thanks"},
				}},
			},
			want: 3,
		},
		{
			name: "streaming events with 2 user turns",
			events: []map[string]interface{}{
				{"type": types.EventTypeTextMessageStart, "role": types.RoleUser},
				{"type": types.EventTypeTextMessageStart, "role": types.RoleAssistant},
				{"type": types.EventTypeTextMessageStart, "role": types.RoleUser},
			},
			want: 2,
		},
		{
			name:   "empty events",
			events: []map[string]interface{}{},
			want:   0,
		},
		{
			name:   "nil events",
			events: nil,
			want:   0,
		},
		{
			name: "snapshot preferred over streaming events",
			events: []map[string]interface{}{
				{"type": types.EventTypeTextMessageStart, "role": types.RoleUser},
				{"type": types.EventTypeTextMessageStart, "role": types.RoleUser},
				{"type": types.EventTypeMessagesSnapshot, "messages": []interface{}{
					map[string]interface{}{"role": types.RoleUser, "content": "hello"},
				}},
			},
			want: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := countUserTurns(tt.events)
			if got != tt.want {
				t.Errorf("countUserTurns() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestBuildTranscriptText(t *testing.T) {
	events := []map[string]interface{}{
		{"type": types.EventTypeMessagesSnapshot, "messages": []interface{}{
			map[string]interface{}{"role": types.RoleSystem, "content": "system msg"},
			map[string]interface{}{"role": types.RoleDeveloper, "content": "developer msg"},
			map[string]interface{}{"role": types.RoleUser, "content": "user msg"},
			map[string]interface{}{"role": types.RoleAssistant, "content": "assistant msg"},
		}},
	}

	text := buildTranscriptText(events)

	if strings.Contains(text, "system msg") {
		t.Error("transcript should not contain system messages")
	}
	if strings.Contains(text, "developer msg") {
		t.Error("transcript should not contain developer messages")
	}
	if !strings.Contains(text, "[user]: user msg") {
		t.Error("transcript should contain user messages")
	}
	if !strings.Contains(text, "[assistant]: assistant msg") {
		t.Error("transcript should contain assistant messages")
	}
}

func TestBuildTranscriptTextEmpty(t *testing.T) {
	text := buildTranscriptText(nil)
	if text != "" {
		t.Errorf("expected empty transcript for nil events, got %q", text)
	}
}

func TestBuildTranscriptTextTruncation(t *testing.T) {
	// Build events with very long content
	longContent := strings.Repeat("a", maxTranscriptChars+1000)
	events := []map[string]interface{}{
		{"type": types.EventTypeMessagesSnapshot, "messages": []interface{}{
			map[string]interface{}{"role": types.RoleUser, "content": longContent},
		}},
	}

	text := buildTranscriptText(events)
	if !strings.HasSuffix(text, "[transcript truncated]") {
		t.Error("long transcript should be truncated")
	}
}

func TestParseExtractionResponse(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantLen int
		wantErr bool
	}{
		{
			name:    "valid JSON array",
			input:   `[{"title":"Test","content":"Content","type":"pattern","confidence":0.8}]`,
			wantLen: 1,
			wantErr: false,
		},
		{
			name:    "with markdown fences",
			input:   "```json\n[{\"title\":\"Test\",\"content\":\"Content\",\"type\":\"correction\",\"confidence\":0.9}]\n```",
			wantLen: 1,
			wantErr: false,
		},
		{
			name:    "empty array",
			input:   `[]`,
			wantLen: 0,
			wantErr: false,
		},
		{
			name:    "invalid JSON",
			input:   `not json`,
			wantLen: 0,
			wantErr: true,
		},
		{
			name:    "missing required fields filtered out",
			input:   `[{"title":"","content":"Content","type":"pattern","confidence":0.5},{"title":"Good","content":"Content","type":"pattern","confidence":0.8}]`,
			wantLen: 1,
			wantErr: false,
		},
		{
			name:    "invalid type filtered out",
			input:   `[{"title":"Test","content":"Content","type":"invalid","confidence":0.8}]`,
			wantLen: 0,
			wantErr: false,
		},
		{
			name:    "confidence clamped to 1.0",
			input:   `[{"title":"Test","content":"Content","type":"pattern","confidence":1.5}]`,
			wantLen: 1,
			wantErr: false,
		},
		{
			name:    "negative confidence clamped to 0.0",
			input:   `[{"title":"Test","content":"Content","type":"correction","confidence":-0.5}]`,
			wantLen: 1,
			wantErr: false,
		},
		{
			name:    "multiple valid candidates",
			input:   `[{"title":"A","content":"C","type":"pattern","confidence":0.7},{"title":"B","content":"D","type":"correction","confidence":0.9}]`,
			wantLen: 2,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseExtractionResponse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseExtractionResponse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(got) != tt.wantLen {
				t.Errorf("parseExtractionResponse() returned %d candidates, want %d", len(got), tt.wantLen)
			}
		})
	}
}

func TestParseExtractionResponseConfidenceClamping(t *testing.T) {
	candidates, err := parseExtractionResponse(`[{"title":"T","content":"C","type":"pattern","confidence":1.5}]`)
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 1 {
		t.Fatal("expected 1 candidate")
	}
	if candidates[0].Confidence != 1.0 {
		t.Errorf("confidence should be clamped to 1.0, got %f", candidates[0].Confidence)
	}

	candidates, err = parseExtractionResponse(`[{"title":"T","content":"C","type":"pattern","confidence":-0.5}]`)
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 1 {
		t.Fatal("expected 1 candidate")
	}
	if candidates[0].Confidence != 0.0 {
		t.Errorf("confidence should be clamped to 0.0, got %f", candidates[0].Confidence)
	}
}

func TestRankAndCap(t *testing.T) {
	candidates := []InsightCandidate{
		{Title: "Low", Confidence: 0.3},
		{Title: "High", Confidence: 0.9},
		{Title: "Mid", Confidence: 0.6},
		{Title: "VeryHigh", Confidence: 0.95},
	}

	result := rankAndCap(candidates, 2)
	if len(result) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(result))
	}
	if result[0].Title != "VeryHigh" {
		t.Errorf("expected first candidate to be VeryHigh, got %s", result[0].Title)
	}
	if result[1].Title != "High" {
		t.Errorf("expected second candidate to be High, got %s", result[1].Title)
	}
}

func TestRankAndCapEmptySlice(t *testing.T) {
	result := rankAndCap(nil, 5)
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestRankAndCapNoTruncation(t *testing.T) {
	candidates := []InsightCandidate{
		{Title: "A", Confidence: 0.8},
		{Title: "B", Confidence: 0.5},
	}
	result := rankAndCap(candidates, 10)
	if len(result) != 2 {
		t.Errorf("expected 2 candidates (no truncation needed), got %d", len(result))
	}
}

func TestSlugify(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Hello World!", "hello-world"},
		{"use-kebab-case", "use-kebab-case"},
		{"Special @#$ Characters", "special-characters"},
		{"", "insight"},
		{"A Very Long Title That Exceeds The Maximum Length Of Sixty Characters For Slugs And More", "a-very-long-title-that-exceeds-the-maximum-length-of-sixty-c"},
		{"---dashes---", "dashes"},
		{"123 Numbers", "123-numbers"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := slugify(tt.input)
			if got != tt.want {
				t.Errorf("slugify(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestInsightFilePath(t *testing.T) {
	c := InsightCandidate{
		Title: "Use Context Managers",
		Type:  "pattern",
	}
	path := insightFilePath(c)
	if !strings.Contains(path, "docs/learned/patterns/") {
		t.Errorf("expected path to contain docs/learned/patterns/, got %s", path)
	}
	if !strings.Contains(path, "use-context-managers") {
		t.Errorf("expected path to contain use-context-managers, got %s", path)
	}

	c2 := InsightCandidate{
		Title: "Avoid Panic in Production",
		Type:  "correction",
	}
	path2 := insightFilePath(c2)
	if !strings.Contains(path2, "docs/learned/corrections/") {
		t.Errorf("expected path to contain docs/learned/corrections/, got %s", path2)
	}
}

func TestFormatInsightMarkdown(t *testing.T) {
	c := InsightCandidate{
		Title:      "Test Insight",
		Content:    "This is the content.",
		Type:       "pattern",
		Confidence: 0.85,
	}
	md := formatInsightMarkdown(c, "test-session", "test-project")

	checks := []struct {
		label    string
		contains string
	}{
		{"title", "# Test Insight"},
		{"type", "**Type:** pattern"},
		{"source", "**Source:** insight-extraction"},
		{"session ref", "test-project/test-session"},
		{"content", "This is the content."},
		{"confidence", "**Confidence:** 0.85"},
	}

	for _, check := range checks {
		if !strings.Contains(md, check.contains) {
			t.Errorf("markdown should contain %s (%q)", check.label, check.contains)
		}
	}
}

func TestParseGitHubOwnerRepo(t *testing.T) {
	tests := []struct {
		input     string
		wantOwner string
		wantRepo  string
		wantErr   bool
	}{
		{"https://github.com/owner/repo", "owner", "repo", false},
		{"https://github.com/owner/repo.git", "owner", "repo", false},
		{"git@github.com:owner/repo.git", "owner", "repo", false},
		{"https://gitlab.com/owner/repo", "", "", true},
		{"invalid-url", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			owner, repo, err := parseGitHubOwnerRepo(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if owner != tt.wantOwner {
				t.Errorf("owner = %q, want %q", owner, tt.wantOwner)
			}
			if repo != tt.wantRepo {
				t.Errorf("repo = %q, want %q", repo, tt.wantRepo)
			}
		})
	}
}
