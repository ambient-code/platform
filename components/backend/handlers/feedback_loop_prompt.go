package handlers

import (
	"fmt"
	"sort"
	"strings"
)

// maxCorrectionsPerPrompt caps the number of corrections included in a single
// improvement prompt. Any remainder is summarized with a count.
const maxCorrectionsPerPrompt = 50

// correctionDetail holds one correction for prompt rendering.
type correctionDetail struct {
	CorrectionType string `json:"correction_type"`
	Source         string `json:"source"`
	AgentAction    string `json:"agent_action"`
	UserCorrection string `json:"user_correction"`
	SessionName    string `json:"session_name"`
	TraceID        string `json:"trace_id"`
}

// correctionGroup aggregates corrections for a single target.
type correctionGroup struct {
	TargetType           string             `json:"target_type"`
	TargetRepoURL        string             `json:"target_repo_url"`
	TargetBranch         string             `json:"target_branch"`
	TargetPath           string             `json:"target_path"`
	Corrections          []correctionDetail `json:"corrections"`
	TotalCount           int                `json:"total_count"`
	CorrectionTypeCounts map[string]int     `json:"correction_type_counts"`
	SourceCounts         map[string]int     `json:"source_counts"`
}

var correctionTypeDescriptions = map[string]string{
	"incomplete":   "missed something that should have been done",
	"incorrect":    "did the wrong thing",
	"out_of_scope": "worked on wrong files or area",
	"style":        "right result, wrong approach or pattern",
}

var correctionSourceDescriptions = map[string]string{
	"human":  "user-provided correction during a session",
	"rubric": "automatically detected from a rubric evaluation",
}

// sanitizePromptText removes shell-interpreted characters from text embedded
// in prompts. The prompt may be passed through bash eval by ambient-action,
// so backticks, $, and angle brackets must be stripped or replaced.
func sanitizePromptText(text string) string {
	r := strings.NewReplacer("`", "'", "$", "", "<", "(", ">", ")")
	return r.Replace(text)
}

// groupCorrectionKey builds a deduplication key for a correction target.
// Repo corrections exclude the branch (corrections apply regardless of branch).
// Workflow corrections include the branch since different branches may have
// different workflow instructions.
func groupCorrectionKey(targetType, repoURL, branch, path string) string {
	groupBranch := ""
	if targetType == "workflow" {
		groupBranch = branch
	}
	return fmt.Sprintf("%s|%s|%s|%s", targetType, repoURL, groupBranch, path)
}

// repoShortName extracts the short name from a repo URL.
func repoShortName(url string) string {
	if url == "" {
		return "unknown"
	}
	url = strings.TrimRight(url, "/")
	parts := strings.Split(url, "/")
	name := parts[len(parts)-1]
	return strings.TrimSuffix(name, ".git")
}

// buildImprovementPrompt constructs the prompt for an improvement session.
// This is the Go port of build_improvement_prompt() from
// scripts/feedback-loop/query_corrections.py.
func buildImprovementPrompt(group correctionGroup) string {
	total := group.TotalCount
	typeCounts := group.CorrectionTypeCounts

	// Find the most common correction type
	topType := "N/A"
	topCount := 0
	for t, count := range typeCounts {
		if count > topCount {
			topType = t
			topCount = count
		}
	}

	// Build type breakdown sorted by count descending
	type kv struct {
		Key   string
		Value int
	}
	sortedTypes := make([]kv, 0, len(typeCounts))
	for k, v := range typeCounts {
		sortedTypes = append(sortedTypes, kv{k, v})
	}
	sort.Slice(sortedTypes, func(i, j int) bool { return sortedTypes[i].Value > sortedTypes[j].Value })

	var typeBreakdown strings.Builder
	for _, item := range sortedTypes {
		desc := correctionTypeDescriptions[item.Key]
		if desc == "" {
			desc = item.Key
		}
		fmt.Fprintf(&typeBreakdown, "- **%s** (%s): %d\n", item.Key, desc, item.Value)
	}

	// Build source breakdown sorted by count descending
	sortedSources := make([]kv, 0, len(group.SourceCounts))
	for k, v := range group.SourceCounts {
		sortedSources = append(sortedSources, kv{k, v})
	}
	sort.Slice(sortedSources, func(i, j int) bool { return sortedSources[i].Value > sortedSources[j].Value })

	var sourceBreakdown strings.Builder
	for _, item := range sortedSources {
		desc := correctionSourceDescriptions[item.Key]
		if desc == "" {
			desc = item.Key
		}
		fmt.Fprintf(&sourceBreakdown, "- **%s** (%s): %d\n", item.Key, desc, item.Value)
	}

	// Build corrections detail (capped at maxCorrectionsPerPrompt)
	corrections := group.Corrections
	displayCount := len(corrections)
	if displayCount > maxCorrectionsPerPrompt {
		displayCount = maxCorrectionsPerPrompt
	}

	var correctionsDetail strings.Builder
	for i := 0; i < displayCount; i++ {
		c := corrections[i]
		sourceTag := ""
		if c.Source == "rubric" {
			sourceTag = " [rubric]"
		}
		agentAction := sanitizePromptText(c.AgentAction)
		userCorrection := sanitizePromptText(c.UserCorrection)
		fmt.Fprintf(&correctionsDetail, "### Correction %d (%s%s)\n", i+1, c.CorrectionType, sourceTag)
		fmt.Fprintf(&correctionsDetail, "- **Agent did**: %s\n", agentAction)
		fmt.Fprintf(&correctionsDetail, "- **User corrected to**: %s\n", userCorrection)
		if c.SessionName != "" {
			fmt.Fprintf(&correctionsDetail, "- **Session**: %s\n", c.SessionName)
		}
		correctionsDetail.WriteString("\n")
	}

	if len(corrections) > maxCorrectionsPerPrompt {
		remainder := len(corrections) - maxCorrectionsPerPrompt
		fmt.Fprintf(&correctionsDetail, "\n*(%d additional corrections not shown — review Langfuse for full details)*\n\n", remainder)
	}

	// Target description and task instructions differ by target type
	var targetDescription, taskInstructions string
	branchLabel := group.TargetBranch
	if branchLabel == "" {
		branchLabel = "default"
	}

	if group.TargetType == "workflow" {
		targetDescription = fmt.Sprintf(
			"- **Target type**: workflow\n- **Workflow path**: %s\n- **Workflow repo**: %s (branch: %s)",
			group.TargetPath, group.TargetRepoURL, branchLabel,
		)
		taskInstructions = fmt.Sprintf(
			"2. **Make targeted improvements**:\n"+
				"   - Update workflow files in %s (system prompt, instructions)\n"+
				"     where the workflow is guiding the agent incorrectly or incompletely\n"+
				"   - Update rubric criteria if rubric-sourced corrections indicate misaligned expectations\n"+
				"   - Update .claude/patterns/ files if the agent consistently used wrong patterns",
			group.TargetPath,
		)
	} else {
		targetDescription = fmt.Sprintf(
			"- **Target type**: repository\n- **Repository**: %s (branch: %s)",
			group.TargetRepoURL, branchLabel,
		)
		taskInstructions = "2. **Make targeted improvements**:\n" +
			"   - Update CLAUDE.md or .claude/ context files where the agent\n" +
			"     lacked necessary knowledge about this repository\n" +
			"   - Update .claude/patterns/ files if the agent consistently used wrong patterns\n" +
			"   - Add missing documentation that would have prevented these corrections"
	}

	return fmt.Sprintf(`# Feedback Loop: Improvement Session

## Context

You are analyzing %d corrections collected from Ambient Code Platform sessions.

%s
- **Most common correction type**: %s (%d occurrences)

## Correction Type Breakdown

%s
## Correction Sources

%s
## Detailed Corrections

%s## Your Task

1. **Analyze patterns**: Look for recurring themes across the corrections.
   Single incidents may be agent errors, but patterns indicate systemic gaps.

%s

3. **Use the corrections as a guide**: For each change, ask "would this correction
   have been prevented if this information existed in the context?"

4. **Be surgical**: Only update files directly related to the corrections.
   Preserve existing content. Add or modify — do not replace wholesale.

5. **Commit, push, and open a PR**: Commit your changes with a descriptive
   message, push to a feature branch, then create a pull request targeting the
   default branch. NEVER push directly to main or master.

   **Include a link to this improvement session in the PR body.** Build the URL
   by reading the environment variables AMBIENT_UI_URL, AGENTIC_SESSION_NAMESPACE,
   and AGENTIC_SESSION_NAME, then construct:
   AMBIENT_UI_URL/projects/AGENTIC_SESSION_NAMESPACE/sessions/AGENTIC_SESSION_NAME
   Add it under a "Session" heading so reviewers can trace the PR back to this session.

## Requirements

- Do NOT over-generalize from isolated incidents
- Focus on the most frequent correction types first
- Each improvement should directly address one or more specific corrections
- Keep changes minimal and focused
- Test that any modified configuration files are still valid
`,
		total,
		targetDescription,
		topType, typeCounts[topType],
		typeBreakdown.String(),
		sourceBreakdown.String(),
		correctionsDetail.String(),
		taskInstructions,
	)
}
