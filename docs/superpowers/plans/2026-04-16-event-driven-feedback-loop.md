# Event-Driven Feedback Loop Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add event-driven threshold-based triggering of improvement sessions when corrections accumulate for a target, replacing the weekly-only batch cadence with near-real-time feedback.

**Architecture:** A ConfigMap-backed config/state store per project namespace holds feedback loop settings and deduplication state. A threshold watcher goroutine evaluates corrections asynchronously when notified. Config, history, and manual-trigger endpoints are added to the backend REST API. The improvement prompt is built using Go code ported from `scripts/feedback-loop/query_corrections.py`.

**Tech Stack:** Go (Gin HTTP handlers, K8s client-go, ConfigMaps for persistence), Ginkgo/Gomega tests

---

## File Structure

| File | Responsibility |
|------|----------------|
| **Create:** `handlers/feedback_loop_config.go` | Config GET/PUT endpoints, history endpoint, feature flag gate |
| **Create:** `handlers/feedback_loop_watcher.go` | Threshold watcher, deduplication, correction buffer, session creation |
| **Create:** `handlers/feedback_loop_prompt.go` | Improvement prompt builder (ported from Python) |
| **Create:** `handlers/feedback_loop_config_test.go` | Tests for config and history endpoints |
| **Create:** `handlers/feedback_loop_watcher_test.go` | Tests for threshold watcher and dedup |
| **Create:** `handlers/feedback_loop_prompt_test.go` | Tests for prompt builder |
| **Modify:** `routes.go` | Register feedback-loop endpoints under projectGroup |
| **Modify:** `components/manifests/base/core/flags.json` | Add `learning-agent-loop` feature flag |

All paths relative to `components/backend/`.

### Task 1: Add the `learning-agent-loop` feature flag

**Files:**
- Modify: `components/manifests/base/core/flags.json`

- [ ] **Step 1: Add the flag to flags.json**

Add the `learning-agent-loop` flag with `scope:workspace` tag to `components/manifests/base/core/flags.json`. Insert it after the last existing entry:

```json
    {
      "name": "learning-agent-loop",
      "description": "Enable the learning agent loop: corrections pipeline, feedback loop, and memory features",
      "tags": [
        {
          "type": "scope",
          "value": "workspace"
        }
      ]
    }
```

- [ ] **Step 2: Validate JSON**

Run: `python3 -c "import json; json.load(open('components/manifests/base/core/flags.json'))"`
Expected: No output (valid JSON)

- [ ] **Step 3: Commit**

```bash
git add components/manifests/base/core/flags.json
git commit -m "feat: add learning-agent-loop feature flag"
```

---

### Task 2: Improvement prompt builder

**Files:**
- Create: `components/backend/handlers/feedback_loop_prompt.go`
- Create: `components/backend/handlers/feedback_loop_prompt_test.go`

- [ ] **Step 1: Write the failing test for prompt builder**

Create `components/backend/handlers/feedback_loop_prompt_test.go`:

```go
//go:build test

package handlers

import (
	test_constants "ambient-code-backend/tests/constants"
	"ambient-code-backend/tests/logger"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Feedback Loop Prompt Builder", Label(test_constants.LabelUnit, test_constants.LabelHandlers), func() {
	Describe("buildImprovementPrompt", func() {
		It("Should build a prompt for workflow corrections", func() {
			group := correctionGroup{
				TargetType:    "workflow",
				TargetRepoURL: "https://github.com/org/repo",
				TargetBranch:  "main",
				TargetPath:    ".ambient/workflows/review",
				TotalCount:    2,
				CorrectionTypeCounts: map[string]int{
					"incorrect": 1,
					"style":     1,
				},
				SourceCounts: map[string]int{
					"human": 2,
				},
				Corrections: []correctionDetail{
					{
						CorrectionType: "incorrect",
						Source:         "human",
						AgentAction:    "Deleted the test file",
						UserCorrection: "Should have updated the test file",
						SessionName:    "session-abc",
						TraceID:        "trace-1",
					},
					{
						CorrectionType: "style",
						Source:         "human",
						AgentAction:    "Used fmt.Println for logging",
						UserCorrection: "Use log.Printf instead",
						SessionName:    "session-def",
						TraceID:        "trace-2",
					},
				},
			}

			prompt := buildImprovementPrompt(group)

			Expect(prompt).To(ContainSubstring("2 corrections"))
			Expect(prompt).To(ContainSubstring("workflow"))
			Expect(prompt).To(ContainSubstring(".ambient/workflows/review"))
			Expect(prompt).To(ContainSubstring("Deleted the test file"))
			Expect(prompt).To(ContainSubstring("Should have updated the test file"))
			Expect(prompt).To(ContainSubstring("Update workflow files"))

			logger.Log("Workflow improvement prompt built successfully")
		})

		It("Should build a prompt for repo corrections", func() {
			group := correctionGroup{
				TargetType:    "repo",
				TargetRepoURL: "https://github.com/org/repo",
				TargetBranch:  "main",
				TargetPath:    "",
				TotalCount:    3,
				CorrectionTypeCounts: map[string]int{
					"incomplete": 3,
				},
				SourceCounts: map[string]int{
					"human":  2,
					"rubric": 1,
				},
				Corrections: []correctionDetail{
					{CorrectionType: "incomplete", Source: "human", AgentAction: "a1", UserCorrection: "c1", TraceID: "t1"},
					{CorrectionType: "incomplete", Source: "human", AgentAction: "a2", UserCorrection: "c2", TraceID: "t2"},
					{CorrectionType: "incomplete", Source: "rubric", AgentAction: "a3", UserCorrection: "c3", TraceID: "t3"},
				},
			}

			prompt := buildImprovementPrompt(group)

			Expect(prompt).To(ContainSubstring("3 corrections"))
			Expect(prompt).To(ContainSubstring("repository"))
			Expect(prompt).To(ContainSubstring("Update CLAUDE.md"))
			Expect(prompt).NotTo(ContainSubstring("Update workflow files"))

			logger.Log("Repo improvement prompt built successfully")
		})

		It("Should cap corrections at 50 and summarize remainder", func() {
			corrections := make([]correctionDetail, 55)
			for i := range corrections {
				corrections[i] = correctionDetail{
					CorrectionType: "style",
					Source:         "human",
					AgentAction:    "action",
					UserCorrection: "correction",
					TraceID:        "trace",
				}
			}

			group := correctionGroup{
				TargetType:           "repo",
				TargetRepoURL:        "https://github.com/org/repo",
				TotalCount:           55,
				CorrectionTypeCounts: map[string]int{"style": 55},
				SourceCounts:         map[string]int{"human": 55},
				Corrections:          corrections,
			}

			prompt := buildImprovementPrompt(group)

			Expect(prompt).To(ContainSubstring("55 corrections"))
			Expect(prompt).To(ContainSubstring("5 additional corrections"))
			// Should only have 50 numbered correction sections
			Expect(strings.Count(prompt, "### Correction ")).To(Equal(50))

			logger.Log("Prompt correctly caps at 50 corrections")
		})

		It("Should sanitize shell-interpreted characters", func() {
			group := correctionGroup{
				TargetType:           "repo",
				TargetRepoURL:        "https://github.com/org/repo",
				TotalCount:           1,
				CorrectionTypeCounts: map[string]int{"style": 1},
				SourceCounts:         map[string]int{"human": 1},
				Corrections: []correctionDetail{
					{
						CorrectionType: "style",
						Source:         "human",
						AgentAction:    "Used `backticks` and $VAR",
						UserCorrection: "Don't use <angle> brackets",
						TraceID:        "t1",
					},
				},
			}

			prompt := buildImprovementPrompt(group)

			Expect(prompt).NotTo(ContainSubstring("`"))
			Expect(prompt).NotTo(ContainSubstring("$"))
			Expect(prompt).NotTo(ContainSubstring("<"))
			Expect(prompt).NotTo(ContainSubstring(">"))

			logger.Log("Prompt sanitizes shell characters correctly")
		})
	})

	Describe("groupCorrectionKey", func() {
		It("Should include branch for workflow targets", func() {
			key := groupCorrectionKey("workflow", "https://github.com/org/repo", "main", ".ambient/workflows/review")
			Expect(key).To(Equal("workflow|https://github.com/org/repo|main|.ambient/workflows/review"))
		})

		It("Should exclude branch for repo targets", func() {
			key := groupCorrectionKey("repo", "https://github.com/org/repo", "feature-branch", "")
			Expect(key).To(Equal("repo|https://github.com/org/repo||"))
		})
	})

	Describe("repoShortName", func() {
		It("Should extract repo name from URL", func() {
			Expect(repoShortName("https://github.com/org/my-repo.git")).To(Equal("my-repo"))
			Expect(repoShortName("https://github.com/org/my-repo")).To(Equal("my-repo"))
			Expect(repoShortName("")).To(Equal("unknown"))
		})
	})
})
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd components/backend && go test -tags test -run "Feedback Loop Prompt" ./handlers/ -v --ginkgo.v 2>&1 | tail -20`
Expected: Compilation error (types and functions not defined)

- [ ] **Step 3: Implement the prompt builder**

Create `components/backend/handlers/feedback_loop_prompt.go`:

```go
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
	TargetType           string            `json:"target_type"`
	TargetRepoURL        string            `json:"target_repo_url"`
	TargetBranch         string            `json:"target_branch"`
	TargetPath           string            `json:"target_path"`
	Corrections          []correctionDetail `json:"corrections"`
	TotalCount           int               `json:"total_count"`
	CorrectionTypeCounts map[string]int    `json:"correction_type_counts"`
	SourceCounts         map[string]int    `json:"source_counts"`
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
	targetType := group.TargetType
	targetRepoURL := group.TargetRepoURL
	targetBranch := group.TargetBranch
	targetPath := group.TargetPath
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

	// Build type breakdown
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

	// Build source breakdown
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

	// Target description and task instructions
	var targetDescription, taskInstructions string
	if targetType == "workflow" {
		branchLabel := targetBranch
		if branchLabel == "" {
			branchLabel = "default"
		}
		targetDescription = fmt.Sprintf(
			"- **Target type**: workflow\n- **Workflow path**: %s\n- **Workflow repo**: %s (branch: %s)",
			targetPath, targetRepoURL, branchLabel,
		)
		taskInstructions = fmt.Sprintf(
			"2. **Make targeted improvements**:\n"+
				"   - Update workflow files in %s (system prompt, instructions)\n"+
				"     where the workflow is guiding the agent incorrectly or incompletely\n"+
				"   - Update rubric criteria if rubric-sourced corrections indicate misaligned expectations\n"+
				"   - Update .claude/patterns/ files if the agent consistently used wrong patterns",
			targetPath,
		)
	} else {
		branchLabel := targetBranch
		if branchLabel == "" {
			branchLabel = "default"
		}
		targetDescription = fmt.Sprintf(
			"- **Target type**: repository\n- **Repository**: %s (branch: %s)",
			targetRepoURL, branchLabel,
		)
		taskInstructions = "2. **Make targeted improvements**:\n" +
			"   - Update CLAUDE.md or .claude/ context files where the agent\n" +
			"     lacked necessary knowledge about this repository\n" +
			"   - Update .claude/patterns/ files if the agent consistently used wrong patterns\n" +
			"   - Add missing documentation that would have prevented these corrections"
	}

	prompt := fmt.Sprintf(`# Feedback Loop: Improvement Session

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

	return prompt
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd components/backend && go test -tags test -run "Feedback Loop Prompt" ./handlers/ -v --ginkgo.v 2>&1 | tail -30`
Expected: All tests pass

- [ ] **Step 5: Commit**

```bash
git add components/backend/handlers/feedback_loop_prompt.go components/backend/handlers/feedback_loop_prompt_test.go
git commit -m "feat: add improvement prompt builder for feedback loop (ported from Python)"
```

---

### Task 3: Feedback loop config and history endpoints

**Files:**
- Create: `components/backend/handlers/feedback_loop_config.go`
- Create: `components/backend/handlers/feedback_loop_config_test.go`
- Modify: `components/backend/routes.go`

- [ ] **Step 1: Write the failing tests for config endpoints**

Create `components/backend/handlers/feedback_loop_config_test.go`:

```go
//go:build test

package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	test_constants "ambient-code-backend/tests/constants"
	"ambient-code-backend/tests/logger"
	"ambient-code-backend/tests/test_utils"

	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Feedback Loop Config Handler", Label(test_constants.LabelUnit, test_constants.LabelHandlers), func() {
	var (
		httpUtils   *test_utils.HTTPTestUtils
		k8sUtils    *test_utils.K8sTestUtils
		testToken   string
	)

	BeforeEach(func() {
		logger.Log("Setting up Feedback Loop Config test")

		k8sUtils = test_utils.NewK8sTestUtils(false, "test-project")
		SetupHandlerDependencies(k8sUtils)

		httpUtils = test_utils.NewHTTPTestUtils()

		ctx := context.Background()
		_, err := k8sUtils.K8sClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: "test-project"},
		}, metav1.CreateOptions{})
		if err != nil && !errors.IsAlreadyExists(err) {
			Expect(err).NotTo(HaveOccurred())
		}
		_, err = k8sUtils.CreateTestRole(ctx, "test-project", "feedback-loop-role", []string{"get", "list", "create", "update", "delete", "patch"}, "*", "")
		Expect(err).NotTo(HaveOccurred())

		token, _, err := httpUtils.SetValidTestToken(
			k8sUtils,
			"test-project",
			[]string{"get", "list", "create", "update", "delete", "patch"},
			"*",
			"",
			"feedback-loop-role",
		)
		Expect(err).NotTo(HaveOccurred())
		testToken = token
	})

	AfterEach(func() {
		if k8sUtils != nil {
			_ = k8sUtils.K8sClient.CoreV1().Namespaces().Delete(context.Background(), "test-project", metav1.DeleteOptions{})
		}
	})

	Describe("GetFeedbackLoopConfig", func() {
		It("Should return defaults when no config exists", func() {
			ginCtx := httpUtils.CreateTestGinContext("GET", "/api/projects/test-project/feedback-loop/config", nil)
			ginCtx.Params = gin.Params{{Key: "projectName", Value: "test-project"}}
			httpUtils.SetProjectContext("test-project")
			httpUtils.SetAuthHeader(testToken)

			GetFeedbackLoopConfig(ginCtx)

			httpUtils.AssertHTTPStatus(http.StatusOK)

			var resp FeedbackLoopConfig
			httpUtils.GetResponseJSON(&resp)
			Expect(resp.MinCorrections).To(Equal(defaultMinCorrections))
			Expect(resp.TimeWindowHours).To(Equal(defaultTimeWindowHours))
			Expect(resp.AutoTriggerEnabled).To(BeTrue())

			logger.Log("GetFeedbackLoopConfig returns defaults correctly")
		})

		It("Should return stored config when it exists", func() {
			ctx := context.Background()
			cm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      feedbackLoopConfigMap,
					Namespace: "test-project",
				},
				Data: map[string]string{
					"config": `{"minCorrections":5,"timeWindowHours":12,"autoTriggerEnabled":false}`,
				},
			}
			_, err := k8sUtils.K8sClient.CoreV1().ConfigMaps("test-project").Create(ctx, cm, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			ginCtx := httpUtils.CreateTestGinContext("GET", "/api/projects/test-project/feedback-loop/config", nil)
			ginCtx.Params = gin.Params{{Key: "projectName", Value: "test-project"}}
			httpUtils.SetProjectContext("test-project")
			httpUtils.SetAuthHeader(testToken)

			GetFeedbackLoopConfig(ginCtx)

			httpUtils.AssertHTTPStatus(http.StatusOK)

			var resp FeedbackLoopConfig
			httpUtils.GetResponseJSON(&resp)
			Expect(resp.MinCorrections).To(Equal(5))
			Expect(resp.TimeWindowHours).To(Equal(12))
			Expect(resp.AutoTriggerEnabled).To(BeFalse())

			logger.Log("GetFeedbackLoopConfig returns stored config")
		})

		It("Should require authentication", func() {
			ginCtx := httpUtils.CreateTestGinContext("GET", "/api/projects/test-project/feedback-loop/config", nil)
			ginCtx.Params = gin.Params{{Key: "projectName", Value: "test-project"}}
			// No auth header

			GetFeedbackLoopConfig(ginCtx)

			httpUtils.AssertHTTPStatus(http.StatusUnauthorized)

			logger.Log("GetFeedbackLoopConfig requires auth")
		})
	})

	Describe("PutFeedbackLoopConfig", func() {
		It("Should store valid config", func() {
			body := map[string]interface{}{
				"minCorrections":     3,
				"timeWindowHours":    12,
				"autoTriggerEnabled": true,
			}
			ginCtx := httpUtils.CreateTestGinContext("PUT", "/api/projects/test-project/feedback-loop/config", body)
			ginCtx.Params = gin.Params{{Key: "projectName", Value: "test-project"}}
			httpUtils.SetProjectContext("test-project")
			httpUtils.SetAuthHeader(testToken)

			PutFeedbackLoopConfig(ginCtx)

			httpUtils.AssertHTTPStatus(http.StatusOK)

			// Verify it was persisted
			ctx := context.Background()
			cm, err := k8sUtils.K8sClient.CoreV1().ConfigMaps("test-project").Get(ctx, feedbackLoopConfigMap, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(cm.Data["config"]).To(ContainSubstring(`"minCorrections":3`))

			logger.Log("PutFeedbackLoopConfig stores valid config")
		})

		It("Should reject minCorrections < 1", func() {
			body := map[string]interface{}{
				"minCorrections":     0,
				"timeWindowHours":    24,
				"autoTriggerEnabled": true,
			}
			ginCtx := httpUtils.CreateTestGinContext("PUT", "/api/projects/test-project/feedback-loop/config", body)
			ginCtx.Params = gin.Params{{Key: "projectName", Value: "test-project"}}
			httpUtils.SetProjectContext("test-project")
			httpUtils.SetAuthHeader(testToken)

			PutFeedbackLoopConfig(ginCtx)

			httpUtils.AssertHTTPStatus(http.StatusBadRequest)

			logger.Log("PutFeedbackLoopConfig rejects invalid minCorrections")
		})

		It("Should reject timeWindowHours < 1", func() {
			body := map[string]interface{}{
				"minCorrections":     2,
				"timeWindowHours":    0,
				"autoTriggerEnabled": true,
			}
			ginCtx := httpUtils.CreateTestGinContext("PUT", "/api/projects/test-project/feedback-loop/config", body)
			ginCtx.Params = gin.Params{{Key: "projectName", Value: "test-project"}}
			httpUtils.SetProjectContext("test-project")
			httpUtils.SetAuthHeader(testToken)

			PutFeedbackLoopConfig(ginCtx)

			httpUtils.AssertHTTPStatus(http.StatusBadRequest)

			logger.Log("PutFeedbackLoopConfig rejects invalid timeWindowHours")
		})

		It("Should require authentication", func() {
			body := map[string]interface{}{
				"minCorrections":     2,
				"timeWindowHours":    24,
				"autoTriggerEnabled": true,
			}
			ginCtx := httpUtils.CreateTestGinContext("PUT", "/api/projects/test-project/feedback-loop/config", body)
			ginCtx.Params = gin.Params{{Key: "projectName", Value: "test-project"}}
			// No auth header

			PutFeedbackLoopConfig(ginCtx)

			httpUtils.AssertHTTPStatus(http.StatusUnauthorized)

			logger.Log("PutFeedbackLoopConfig requires auth")
		})
	})

	Describe("GetFeedbackLoopHistory", func() {
		It("Should return empty list when no history exists", func() {
			ginCtx := httpUtils.CreateTestGinContext("GET", "/api/projects/test-project/feedback-loop/history", nil)
			ginCtx.Params = gin.Params{{Key: "projectName", Value: "test-project"}}
			httpUtils.SetProjectContext("test-project")
			httpUtils.SetAuthHeader(testToken)

			GetFeedbackLoopHistory(ginCtx)

			httpUtils.AssertHTTPStatus(http.StatusOK)

			var resp map[string]interface{}
			httpUtils.GetResponseJSON(&resp)
			sessions, ok := resp["sessions"].([]interface{})
			Expect(ok).To(BeTrue())
			Expect(sessions).To(BeEmpty())

			logger.Log("GetFeedbackLoopHistory returns empty list")
		})

		It("Should return stored history entries", func() {
			ctx := context.Background()
			entries := []FeedbackLoopHistoryEntry{
				{
					SessionName:    "session-123",
					CreatedAt:      "2026-04-16T10:00:00Z",
					Source:         "event-driven",
					TargetType:     "workflow",
					TargetRepoURL:  "https://github.com/org/repo",
					TargetBranch:   "main",
					TargetPath:     ".ambient/workflows/review",
					CorrectionIDs:  []string{"trace-1", "trace-2"},
				},
			}
			data, err := json.Marshal(entries)
			Expect(err).NotTo(HaveOccurred())

			cm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      feedbackLoopConfigMap,
					Namespace: "test-project",
				},
				Data: map[string]string{
					"history": string(data),
				},
			}
			_, err = k8sUtils.K8sClient.CoreV1().ConfigMaps("test-project").Create(ctx, cm, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			ginCtx := httpUtils.CreateTestGinContext("GET", "/api/projects/test-project/feedback-loop/history", nil)
			ginCtx.Params = gin.Params{{Key: "projectName", Value: "test-project"}}
			httpUtils.SetProjectContext("test-project")
			httpUtils.SetAuthHeader(testToken)

			GetFeedbackLoopHistory(ginCtx)

			httpUtils.AssertHTTPStatus(http.StatusOK)

			var resp map[string]json.RawMessage
			httpUtils.GetResponseJSON(&resp)
			var sessions []FeedbackLoopHistoryEntry
			err = json.Unmarshal(resp["sessions"], &sessions)
			Expect(err).NotTo(HaveOccurred())
			Expect(sessions).To(HaveLen(1))
			Expect(sessions[0].SessionName).To(Equal("session-123"))
			Expect(sessions[0].Source).To(Equal("event-driven"))

			logger.Log("GetFeedbackLoopHistory returns stored entries")
		})

		It("Should require authentication", func() {
			ginCtx := httpUtils.CreateTestGinContext("GET", "/api/projects/test-project/feedback-loop/history", nil)
			ginCtx.Params = gin.Params{{Key: "projectName", Value: "test-project"}}

			GetFeedbackLoopHistory(ginCtx)

			httpUtils.AssertHTTPStatus(http.StatusUnauthorized)

			logger.Log("GetFeedbackLoopHistory requires auth")
		})
	})
})
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd components/backend && go test -tags test -run "Feedback Loop Config" ./handlers/ -v --ginkgo.v 2>&1 | tail -10`
Expected: Compilation error (types/functions not defined)

- [ ] **Step 3: Implement the config handler**

Create `components/backend/handlers/feedback_loop_config.go`:

```go
package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// feedbackLoopConfigMap is the ConfigMap name for feedback loop config and state.
	// Stores per-project configuration, deduplication state, and trigger history.
	// NOTE: Backend restart does NOT lose state because it is persisted in this ConfigMap.
	feedbackLoopConfigMap = "feedback-loop-state"

	// feedbackLoopFeatureFlag is the Unleash flag that gates all learning agent loop features.
	feedbackLoopFeatureFlag = "learning-agent-loop"

	defaultMinCorrections  = 2
	defaultTimeWindowHours = 24
)

// FeedbackLoopConfig holds per-project feedback loop settings.
type FeedbackLoopConfig struct {
	MinCorrections     int  `json:"minCorrections"`
	TimeWindowHours    int  `json:"timeWindowHours"`
	AutoTriggerEnabled bool `json:"autoTriggerEnabled"`
}

// FeedbackLoopHistoryEntry records a triggered improvement session.
type FeedbackLoopHistoryEntry struct {
	SessionName   string   `json:"sessionName"`
	CreatedAt     string   `json:"createdAt"`
	Source        string   `json:"source"`        // "event-driven" or "github-action"
	TargetType    string   `json:"targetType"`    // "workflow" or "repo"
	TargetRepoURL string   `json:"targetRepoURL"`
	TargetBranch  string   `json:"targetBranch,omitempty"`
	TargetPath    string   `json:"targetPath,omitempty"`
	CorrectionIDs []string `json:"correctionIds"`
}

// defaultFeedbackLoopConfig returns the default configuration.
func defaultFeedbackLoopConfig() FeedbackLoopConfig {
	return FeedbackLoopConfig{
		MinCorrections:     defaultMinCorrections,
		TimeWindowHours:    defaultTimeWindowHours,
		AutoTriggerEnabled: true,
	}
}

// loadFeedbackLoopConfig reads the config from the ConfigMap. Returns defaults if not found.
func loadFeedbackLoopConfig(ctx context.Context, namespace string) (FeedbackLoopConfig, error) {
	cm, err := K8sClient.CoreV1().ConfigMaps(namespace).Get(ctx, feedbackLoopConfigMap, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		return defaultFeedbackLoopConfig(), nil
	}
	if err != nil {
		return FeedbackLoopConfig{}, err
	}

	configData, ok := cm.Data["config"]
	if !ok || configData == "" {
		return defaultFeedbackLoopConfig(), nil
	}

	var config FeedbackLoopConfig
	if err := json.Unmarshal([]byte(configData), &config); err != nil {
		log.Printf("Failed to parse feedback loop config in %s, using defaults: %v", namespace, err)
		return defaultFeedbackLoopConfig(), nil
	}

	return config, nil
}

// saveFeedbackLoopConfig persists config to the ConfigMap. Creates it if absent.
func saveFeedbackLoopConfig(ctx context.Context, namespace string, config FeedbackLoopConfig) error {
	data, err := json.Marshal(config)
	if err != nil {
		return err
	}

	cm, err := K8sClient.CoreV1().ConfigMaps(namespace).Get(ctx, feedbackLoopConfigMap, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		newCM := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      feedbackLoopConfigMap,
				Namespace: namespace,
				Labels: map[string]string{
					"app.kubernetes.io/managed-by": "ambient-code",
					"app.kubernetes.io/component":  "feedback-loop",
				},
			},
			Data: map[string]string{
				"config": string(data),
			},
		}
		_, err = K8sClient.CoreV1().ConfigMaps(namespace).Create(ctx, newCM, metav1.CreateOptions{})
		return err
	}
	if err != nil {
		return err
	}

	if cm.Data == nil {
		cm.Data = map[string]string{}
	}
	cm.Data["config"] = string(data)
	_, err = K8sClient.CoreV1().ConfigMaps(namespace).Update(ctx, cm, metav1.UpdateOptions{})
	return err
}

// loadFeedbackLoopHistory reads the history from the ConfigMap.
func loadFeedbackLoopHistory(ctx context.Context, namespace string) ([]FeedbackLoopHistoryEntry, error) {
	cm, err := K8sClient.CoreV1().ConfigMaps(namespace).Get(ctx, feedbackLoopConfigMap, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		return []FeedbackLoopHistoryEntry{}, nil
	}
	if err != nil {
		return nil, err
	}

	historyData, ok := cm.Data["history"]
	if !ok || historyData == "" {
		return []FeedbackLoopHistoryEntry{}, nil
	}

	var entries []FeedbackLoopHistoryEntry
	if err := json.Unmarshal([]byte(historyData), &entries); err != nil {
		log.Printf("Failed to parse feedback loop history in %s: %v", namespace, err)
		return []FeedbackLoopHistoryEntry{}, nil
	}

	return entries, nil
}

// appendFeedbackLoopHistory adds an entry to the history in the ConfigMap.
func appendFeedbackLoopHistory(ctx context.Context, namespace string, entry FeedbackLoopHistoryEntry) error {
	entries, err := loadFeedbackLoopHistory(ctx, namespace)
	if err != nil {
		return err
	}

	entries = append(entries, entry)
	data, err := json.Marshal(entries)
	if err != nil {
		return err
	}

	cm, err := K8sClient.CoreV1().ConfigMaps(namespace).Get(ctx, feedbackLoopConfigMap, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		newCM := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      feedbackLoopConfigMap,
				Namespace: namespace,
				Labels: map[string]string{
					"app.kubernetes.io/managed-by": "ambient-code",
					"app.kubernetes.io/component":  "feedback-loop",
				},
			},
			Data: map[string]string{
				"history": string(data),
			},
		}
		_, err = K8sClient.CoreV1().ConfigMaps(namespace).Create(ctx, newCM, metav1.CreateOptions{})
		return err
	}
	if err != nil {
		return err
	}

	if cm.Data == nil {
		cm.Data = map[string]string{}
	}
	cm.Data["history"] = string(data)
	_, err = K8sClient.CoreV1().ConfigMaps(namespace).Update(ctx, cm, metav1.UpdateOptions{})
	return err
}

// GetFeedbackLoopConfig handles GET /api/projects/:projectName/feedback-loop/config
func GetFeedbackLoopConfig(c *gin.Context) {
	namespace := c.Param("projectName")

	reqK8s, _ := GetK8sClientsForRequest(c)
	if reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User token required"})
		c.Abort()
		return
	}

	config, err := loadFeedbackLoopConfig(c.Request.Context(), namespace)
	if err != nil {
		log.Printf("Failed to load feedback loop config for %s: %v", namespace, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load feedback loop configuration"})
		return
	}

	c.JSON(http.StatusOK, config)
}

// PutFeedbackLoopConfig handles PUT /api/projects/:projectName/feedback-loop/config
func PutFeedbackLoopConfig(c *gin.Context) {
	namespace := c.Param("projectName")

	reqK8s, _ := GetK8sClientsForRequest(c)
	if reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User token required"})
		c.Abort()
		return
	}

	// Check admin permission (ability to patch ConfigMaps in namespace)
	allowed, err := checkConfigMapPermission(c.Request.Context(), reqK8s, namespace, "patch")
	if err != nil {
		log.Printf("Failed to check ConfigMap permissions for feedback loop in %s: %v", namespace, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check permissions"})
		return
	}
	if !allowed {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin permissions required to modify feedback loop configuration"})
		return
	}

	var req FeedbackLoopConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Validate
	if req.MinCorrections < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "minCorrections must be >= 1"})
		return
	}
	if req.TimeWindowHours < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "timeWindowHours must be >= 1"})
		return
	}

	if err := saveFeedbackLoopConfig(c.Request.Context(), namespace, req); err != nil {
		log.Printf("Failed to save feedback loop config for %s: %v", namespace, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save feedback loop configuration"})
		return
	}

	c.JSON(http.StatusOK, req)
}

// GetFeedbackLoopHistory handles GET /api/projects/:projectName/feedback-loop/history
func GetFeedbackLoopHistory(c *gin.Context) {
	namespace := c.Param("projectName")

	reqK8s, _ := GetK8sClientsForRequest(c)
	if reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User token required"})
		c.Abort()
		return
	}

	entries, err := loadFeedbackLoopHistory(c.Request.Context(), namespace)
	if err != nil {
		log.Printf("Failed to load feedback loop history for %s: %v", namespace, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load feedback loop history"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"sessions": entries})
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd components/backend && go test -tags test -run "Feedback Loop Config" ./handlers/ -v --ginkgo.v 2>&1 | tail -30`
Expected: All tests pass

- [ ] **Step 5: Register routes**

Add the feedback loop routes to `components/backend/routes.go` inside the `projectGroup` block, after the corrections endpoints:

```go
			// Feedback loop endpoints (gated by learning-agent-loop feature flag)
			projectGroup.GET("/feedback-loop/config", handlers.GetFeedbackLoopConfig)
			projectGroup.PUT("/feedback-loop/config", handlers.PutFeedbackLoopConfig)
			projectGroup.GET("/feedback-loop/history", handlers.GetFeedbackLoopHistory)
```

- [ ] **Step 6: Verify compilation**

Run: `cd components/backend && go build ./...`
Expected: Clean build

- [ ] **Step 7: Run all tests**

Run: `cd components/backend && go test -tags test -run "Feedback Loop" ./handlers/ -v --ginkgo.v 2>&1 | tail -30`
Expected: All tests pass

- [ ] **Step 8: Commit**

```bash
git add components/backend/handlers/feedback_loop_config.go components/backend/handlers/feedback_loop_config_test.go components/backend/routes.go
git commit -m "feat: add feedback loop config and history endpoints"
```

---

### Task 4: Threshold watcher with deduplication

**Files:**
- Create: `components/backend/handlers/feedback_loop_watcher.go`
- Create: `components/backend/handlers/feedback_loop_watcher_test.go`

- [ ] **Step 1: Write the failing tests for the watcher**

Create `components/backend/handlers/feedback_loop_watcher_test.go`:

```go
//go:build test

package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	test_constants "ambient-code-backend/tests/constants"
	"ambient-code-backend/tests/logger"
	"ambient-code-backend/tests/test_utils"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Feedback Loop Watcher", Label(test_constants.LabelUnit, test_constants.LabelHandlers), func() {
	var (
		k8sUtils *test_utils.K8sTestUtils
	)

	BeforeEach(func() {
		k8sUtils = test_utils.NewK8sTestUtils(false, "test-project")
		SetupHandlerDependencies(k8sUtils)

		ctx := context.Background()
		_, err := k8sUtils.K8sClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: "test-project"},
		}, metav1.CreateOptions{})
		if err != nil && !errors.IsAlreadyExists(err) {
			Expect(err).NotTo(HaveOccurred())
		}
	})

	AfterEach(func() {
		if k8sUtils != nil {
			_ = k8sUtils.K8sClient.CoreV1().Namespaces().Delete(context.Background(), "test-project", metav1.DeleteOptions{})
		}
	})

	Describe("NotifyCorrection", func() {
		It("Should not trigger when below threshold", func() {
			watcher := NewFeedbackLoopWatcher()

			correction := CorrectionNotification{
				Project:        "test-project",
				TargetType:     "workflow",
				TargetRepoURL:  "https://github.com/org/repo",
				TargetBranch:   "main",
				TargetPath:     ".ambient/workflows/review",
				CorrectionType: "incorrect",
				Source:         "human",
				AgentAction:    "did wrong thing",
				UserCorrection: "do right thing",
				SessionName:    "session-1",
				TraceID:        "trace-1",
				Timestamp:      time.Now(),
			}

			triggered := watcher.NotifyCorrection(context.Background(), correction)
			Expect(triggered).To(BeFalse())

			logger.Log("Single correction does not trigger threshold")
		})

		It("Should trigger when threshold is met", func() {
			watcher := NewFeedbackLoopWatcher()

			// Store config with threshold = 2
			ctx := context.Background()
			config := FeedbackLoopConfig{
				MinCorrections:     2,
				TimeWindowHours:    24,
				AutoTriggerEnabled: true,
			}
			err := saveFeedbackLoopConfig(ctx, "test-project", config)
			Expect(err).NotTo(HaveOccurred())

			base := CorrectionNotification{
				Project:        "test-project",
				TargetType:     "workflow",
				TargetRepoURL:  "https://github.com/org/repo",
				TargetBranch:   "main",
				TargetPath:     ".ambient/workflows/review",
				CorrectionType: "incorrect",
				Source:         "human",
				AgentAction:    "did wrong thing",
				UserCorrection: "do right thing",
				Timestamp:      time.Now(),
			}

			c1 := base
			c1.TraceID = "trace-1"
			c1.SessionName = "session-1"
			triggered := watcher.NotifyCorrection(ctx, c1)
			Expect(triggered).To(BeFalse())

			c2 := base
			c2.TraceID = "trace-2"
			c2.SessionName = "session-2"
			triggered = watcher.NotifyCorrection(ctx, c2)
			Expect(triggered).To(BeTrue())

			logger.Log("Threshold crossing triggers improvement session")
		})

		It("Should deduplicate within time window", func() {
			watcher := NewFeedbackLoopWatcher()

			ctx := context.Background()
			config := FeedbackLoopConfig{
				MinCorrections:     2,
				TimeWindowHours:    24,
				AutoTriggerEnabled: true,
			}
			err := saveFeedbackLoopConfig(ctx, "test-project", config)
			Expect(err).NotTo(HaveOccurred())

			base := CorrectionNotification{
				Project:        "test-project",
				TargetType:     "workflow",
				TargetRepoURL:  "https://github.com/org/repo",
				TargetBranch:   "main",
				TargetPath:     ".ambient/workflows/review",
				CorrectionType: "incorrect",
				Source:         "human",
				AgentAction:    "did wrong thing",
				UserCorrection: "do right thing",
				Timestamp:      time.Now(),
			}

			// First threshold crossing
			c1 := base
			c1.TraceID = "trace-1"
			c1.SessionName = "session-1"
			watcher.NotifyCorrection(ctx, c1)

			c2 := base
			c2.TraceID = "trace-2"
			c2.SessionName = "session-2"
			triggered := watcher.NotifyCorrection(ctx, c2)
			Expect(triggered).To(BeTrue())

			// More corrections should NOT trigger again
			c3 := base
			c3.TraceID = "trace-3"
			c3.SessionName = "session-3"
			triggered = watcher.NotifyCorrection(ctx, c3)
			Expect(triggered).To(BeFalse())

			c4 := base
			c4.TraceID = "trace-4"
			c4.SessionName = "session-4"
			triggered = watcher.NotifyCorrection(ctx, c4)
			Expect(triggered).To(BeFalse())

			logger.Log("Deduplication prevents multiple triggers")
		})

		It("Should not trigger when autoTriggerEnabled is false", func() {
			watcher := NewFeedbackLoopWatcher()

			ctx := context.Background()
			config := FeedbackLoopConfig{
				MinCorrections:     2,
				TimeWindowHours:    24,
				AutoTriggerEnabled: false,
			}
			err := saveFeedbackLoopConfig(ctx, "test-project", config)
			Expect(err).NotTo(HaveOccurred())

			base := CorrectionNotification{
				Project:        "test-project",
				TargetType:     "repo",
				TargetRepoURL:  "https://github.com/org/repo",
				CorrectionType: "style",
				Source:         "human",
				AgentAction:    "action",
				UserCorrection: "correction",
				Timestamp:      time.Now(),
			}

			c1 := base
			c1.TraceID = "trace-1"
			watcher.NotifyCorrection(ctx, c1)

			c2 := base
			c2.TraceID = "trace-2"
			triggered := watcher.NotifyCorrection(ctx, c2)
			Expect(triggered).To(BeFalse())

			logger.Log("autoTriggerEnabled=false prevents triggering")
		})

		It("Should track different targets independently", func() {
			watcher := NewFeedbackLoopWatcher()

			ctx := context.Background()
			config := FeedbackLoopConfig{
				MinCorrections:     2,
				TimeWindowHours:    24,
				AutoTriggerEnabled: true,
			}
			err := saveFeedbackLoopConfig(ctx, "test-project", config)
			Expect(err).NotTo(HaveOccurred())

			// One correction for target A
			cA := CorrectionNotification{
				Project:        "test-project",
				TargetType:     "workflow",
				TargetRepoURL:  "https://github.com/org/repo",
				TargetBranch:   "main",
				TargetPath:     ".ambient/workflows/review",
				CorrectionType: "incorrect",
				Source:         "human",
				AgentAction:    "action",
				UserCorrection: "correction",
				TraceID:        "trace-a1",
				Timestamp:      time.Now(),
			}
			watcher.NotifyCorrection(ctx, cA)

			// One correction for target B
			cB := CorrectionNotification{
				Project:        "test-project",
				TargetType:     "repo",
				TargetRepoURL:  "https://github.com/org/other-repo",
				CorrectionType: "style",
				Source:         "human",
				AgentAction:    "action",
				UserCorrection: "correction",
				TraceID:        "trace-b1",
				Timestamp:      time.Now(),
			}
			watcher.NotifyCorrection(ctx, cB)

			// Second correction for target B triggers
			cB2 := cB
			cB2.TraceID = "trace-b2"
			triggered := watcher.NotifyCorrection(ctx, cB2)
			Expect(triggered).To(BeTrue())

			// Target A still at 1 -- not triggered
			// Verify by checking history has only target B
			entries, err := loadFeedbackLoopHistory(ctx, "test-project")
			Expect(err).NotTo(HaveOccurred())
			Expect(entries).To(HaveLen(1))
			Expect(entries[0].TargetRepoURL).To(Equal("https://github.com/org/other-repo"))

			logger.Log("Targets tracked independently")
		})
	})

	Describe("buildSessionDisplayName", func() {
		It("Should format workflow display name", func() {
			name := buildSessionDisplayName("workflow", "https://github.com/org/my-repo.git", ".ambient/workflows/review")
			Expect(name).To(Equal("Feedback Loop: my-repo (review)"))
		})

		It("Should format repo display name", func() {
			name := buildSessionDisplayName("repo", "https://github.com/org/my-repo", "")
			Expect(name).To(Equal("Feedback Loop: my-repo (repo)"))
		})
	})
})
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd components/backend && go test -tags test -run "Feedback Loop Watcher" ./handlers/ -v --ginkgo.v 2>&1 | tail -10`
Expected: Compilation error

- [ ] **Step 3: Implement the watcher**

Create `components/backend/handlers/feedback_loop_watcher.go`:

```go
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CorrectionNotification is the payload passed to the watcher when a correction
// is logged via the corrections pipeline (spec 003).
type CorrectionNotification struct {
	Project        string    `json:"project"`
	TargetType     string    `json:"target_type"`     // "workflow" or "repo"
	TargetRepoURL  string    `json:"target_repo_url"`
	TargetBranch   string    `json:"target_branch"`
	TargetPath     string    `json:"target_path"`
	CorrectionType string    `json:"correction_type"` // "incomplete", "incorrect", "out_of_scope", "style"
	Source         string    `json:"source"`           // "human" or "rubric"
	AgentAction    string    `json:"agent_action"`
	UserCorrection string    `json:"user_correction"`
	SessionName    string    `json:"session_name"`
	TraceID        string    `json:"trace_id"`
	Timestamp      time.Time `json:"timestamp"`
}

// bufferedCorrection stores a correction in the per-target buffer.
type bufferedCorrection struct {
	CorrectionNotification
	ReceivedAt time.Time
}

// targetBuffer holds buffered corrections for a single target key.
type targetBuffer struct {
	Corrections []bufferedCorrection
}

// FeedbackLoopWatcher evaluates corrections against per-project thresholds
// and creates improvement sessions when thresholds are crossed.
//
// The watcher maintains an in-memory buffer of recent corrections per target
// for fast evaluation. Deduplication state and history are persisted in a
// ConfigMap so that backend restarts do not create duplicate sessions.
//
// NOTE (v1): The in-memory correction buffer is lost on backend restart.
// This means corrections logged before a restart will not count toward the
// threshold after restart. This is acceptable for v1 because:
// 1. The weekly GHA sweep catches anything the real-time path misses.
// 2. Persisting the full correction buffer in ConfigMap would add write
//    amplification on every correction log (acceptable trade-off for v2).
type FeedbackLoopWatcher struct {
	mu      sync.Mutex
	buffers map[string]*targetBuffer // key: "project|targetKey"
}

// NewFeedbackLoopWatcher creates a new watcher instance.
func NewFeedbackLoopWatcher() *FeedbackLoopWatcher {
	return &FeedbackLoopWatcher{
		buffers: make(map[string]*targetBuffer),
	}
}

// NotifyCorrection is called asynchronously when a correction is logged.
// It buffers the correction and evaluates whether the threshold has been crossed.
// Returns true if an improvement session was triggered (for testing).
func (w *FeedbackLoopWatcher) NotifyCorrection(ctx context.Context, n CorrectionNotification) bool {
	// Load project config
	config, err := loadFeedbackLoopConfig(ctx, n.Project)
	if err != nil {
		log.Printf("feedback-loop: failed to load config for %s: %v", n.Project, err)
		return false
	}

	if !config.AutoTriggerEnabled {
		return false
	}

	targetKey := groupCorrectionKey(n.TargetType, n.TargetRepoURL, n.TargetBranch, n.TargetPath)
	bufferKey := fmt.Sprintf("%s|%s", n.Project, targetKey)

	w.mu.Lock()
	defer w.mu.Unlock()

	buf, ok := w.buffers[bufferKey]
	if !ok {
		buf = &targetBuffer{}
		w.buffers[bufferKey] = buf
	}

	// Add the correction
	buf.Corrections = append(buf.Corrections, bufferedCorrection{
		CorrectionNotification: n,
		ReceivedAt:             time.Now(),
	})

	// Prune corrections outside the time window
	cutoff := time.Now().Add(-time.Duration(config.TimeWindowHours) * time.Hour)
	pruned := buf.Corrections[:0]
	for _, c := range buf.Corrections {
		if c.Timestamp.After(cutoff) {
			pruned = append(pruned, c)
		}
	}
	buf.Corrections = pruned

	// Check threshold
	if len(buf.Corrections) < config.MinCorrections {
		return false
	}

	// Check deduplication: was an improvement session already created for this target
	// within the time window?
	if w.isDuplicate(ctx, n.Project, targetKey, cutoff) {
		return false
	}

	// Threshold crossed -- create improvement session
	group := w.buildGroupFromBuffer(buf, n)
	sessionName, err := w.createImprovementSession(ctx, n.Project, group)
	if err != nil {
		log.Printf("feedback-loop: failed to create improvement session for %s in %s: %v", targetKey, n.Project, err)
		return false
	}

	// Record in history
	traceIDs := make([]string, len(buf.Corrections))
	for i, c := range buf.Corrections {
		traceIDs[i] = c.TraceID
	}

	entry := FeedbackLoopHistoryEntry{
		SessionName:   sessionName,
		CreatedAt:     time.Now().UTC().Format(time.RFC3339),
		Source:        "event-driven",
		TargetType:    n.TargetType,
		TargetRepoURL: n.TargetRepoURL,
		TargetBranch:  n.TargetBranch,
		TargetPath:    n.TargetPath,
		CorrectionIDs: traceIDs,
	}
	if err := appendFeedbackLoopHistory(ctx, n.Project, entry); err != nil {
		log.Printf("feedback-loop: failed to record history for %s: %v", n.Project, err)
	}

	// Clear the buffer for this target to prevent re-triggering
	buf.Corrections = nil

	log.Printf("feedback-loop: triggered improvement session %s for target %s in project %s", sessionName, targetKey, n.Project)
	return true
}

// isDuplicate checks whether an improvement session was already created for
// this target within the time window by reading the persisted history.
func (w *FeedbackLoopWatcher) isDuplicate(ctx context.Context, project, targetKey string, cutoff time.Time) bool {
	entries, err := loadFeedbackLoopHistory(ctx, project)
	if err != nil {
		log.Printf("feedback-loop: failed to load history for dedup check in %s: %v", project, err)
		return false // Fail open: prefer creating a possible duplicate over silently dropping
	}

	for _, entry := range entries {
		entryKey := groupCorrectionKey(entry.TargetType, entry.TargetRepoURL, entry.TargetBranch, entry.TargetPath)
		if entryKey != targetKey {
			continue
		}
		createdAt, err := time.Parse(time.RFC3339, entry.CreatedAt)
		if err != nil {
			continue
		}
		if createdAt.After(cutoff) {
			return true
		}
	}

	return false
}

// buildGroupFromBuffer constructs a correctionGroup from the buffered corrections.
func (w *FeedbackLoopWatcher) buildGroupFromBuffer(buf *targetBuffer, n CorrectionNotification) correctionGroup {
	typeCounts := map[string]int{}
	sourceCounts := map[string]int{}
	details := make([]correctionDetail, len(buf.Corrections))

	for i, c := range buf.Corrections {
		typeCounts[c.CorrectionType]++
		sourceCounts[c.Source]++
		details[i] = correctionDetail{
			CorrectionType: c.CorrectionType,
			Source:         c.Source,
			AgentAction:    c.AgentAction,
			UserCorrection: c.UserCorrection,
			SessionName:    c.SessionName,
			TraceID:        c.TraceID,
		}
	}

	return correctionGroup{
		TargetType:           n.TargetType,
		TargetRepoURL:        n.TargetRepoURL,
		TargetBranch:         n.TargetBranch,
		TargetPath:           n.TargetPath,
		Corrections:          details,
		TotalCount:           len(buf.Corrections),
		CorrectionTypeCounts: typeCounts,
		SourceCounts:         sourceCounts,
	}
}

// createImprovementSession creates a new AgenticSession CR for the improvement.
// Returns the session name.
func (w *FeedbackLoopWatcher) createImprovementSession(ctx context.Context, project string, group correctionGroup) (string, error) {
	prompt := buildImprovementPrompt(group)
	displayName := buildSessionDisplayName(group.TargetType, group.TargetRepoURL, group.TargetPath)

	labels := map[string]interface{}{
		"feedback-loop": "true",
		"source":        "event-driven",
		"target-type":   group.TargetType,
	}

	spec := map[string]interface{}{
		"initialPrompt": prompt,
		"displayName":   displayName,
		"timeout":       300,
		"llmSettings": map[string]interface{}{
			"model":       "claude-sonnet-4-6",
			"temperature": 0.7,
			"maxTokens":   4000,
		},
		"environmentVariables": map[string]interface{}{
			"LANGFUSE_MASK_MESSAGES": "false",
		},
	}

	// Add repo if available
	if group.TargetRepoURL != "" && strings.HasPrefix(group.TargetRepoURL, "http") {
		repo := map[string]interface{}{
			"url":      group.TargetRepoURL,
			"autoPush": true,
		}
		if group.TargetBranch != "" {
			repo["branch"] = group.TargetBranch
		}
		spec["repos"] = []interface{}{repo}
	}

	sessionName := fmt.Sprintf("feedback-%s", time.Now().UTC().Format("20060102-150405"))

	sessionObj := map[string]interface{}{
		"apiVersion": "vteam.ambient-code/v1alpha1",
		"kind":       "AgenticSession",
		"metadata": map[string]interface{}{
			"name":      sessionName,
			"namespace": project,
			"labels":    labels,
		},
		"spec": spec,
	}

	// Use the dynamic client to create the CR
	gvr := GetAgenticSessionV1Alpha1Resource()
	unstructuredObj := &unstructuredImport{Object: sessionObj}
	_, err := DynamicClient.Resource(gvr).Namespace(project).Create(ctx, unstructuredObj, metav1.CreateOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to create improvement session CR: %w", err)
	}

	return sessionName, nil
}

// buildSessionDisplayName constructs a human-readable display name.
func buildSessionDisplayName(targetType, repoURL, targetPath string) string {
	repoShort := repoShortName(repoURL)
	if targetType == "workflow" {
		pathShort := ""
		if targetPath != "" {
			parts := strings.Split(strings.TrimRight(targetPath, "/"), "/")
			pathShort = parts[len(parts)-1]
		}
		name := "Feedback Loop: " + repoShort
		if pathShort != "" {
			name += " (" + pathShort + ")"
		}
		return name
	}
	return "Feedback Loop: " + repoShort + " (repo)"
}

// unstructuredImport is a type alias to avoid importing the full unstructured
// package in this file — it's already imported in sessions.go.
// We use the same type that sessions.go uses.
type unstructuredImport = unstructured.Unstructured
```

Wait -- that last part won't work because `unstructured` needs to be imported. Let me adjust the approach. The file should import `k8s.io/apimachinery/pkg/apis/meta/v1/unstructured` directly. Let me fix the plan and also handle the fact that the watcher needs to use the dynamic client. I'll refine this in the actual implementation.

- [ ] **Step 4: Fix import and run tests**

The `unstructuredImport` alias at the bottom of the file should be replaced with a proper import of `"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"` at the top, and use `unstructured.Unstructured` directly.

Run: `cd components/backend && go test -tags test -run "Feedback Loop Watcher" ./handlers/ -v --ginkgo.v 2>&1 | tail -30`
Expected: All tests pass

- [ ] **Step 5: Commit**

```bash
git add components/backend/handlers/feedback_loop_watcher.go components/backend/handlers/feedback_loop_watcher_test.go
git commit -m "feat: add threshold watcher with deduplication for feedback loop"
```

---

### Task 5: Integration test and final verification

- [ ] **Step 1: Run all feedback loop tests together**

Run: `cd components/backend && go test -tags test -run "Feedback Loop" ./handlers/ -v --ginkgo.v 2>&1 | tail -40`
Expected: All tests pass

- [ ] **Step 2: Run full backend test suite to check for regressions**

Run: `cd components/backend && go test -tags test ./handlers/ -v --ginkgo.v 2>&1 | tail -40`
Expected: All tests pass

- [ ] **Step 3: Run go vet and format check**

Run: `cd components/backend && gofmt -l handlers/feedback_loop_*.go && go vet ./handlers/`
Expected: No output (clean)

- [ ] **Step 4: Commit any fixes**

If any fixes were needed from steps 1-3, commit them.

```bash
git add -A
git commit -m "fix: address feedback loop test/lint issues"
```
