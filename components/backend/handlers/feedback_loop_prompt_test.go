//go:build test

package handlers

import (
	"strings"

	test_constants "ambient-code-backend/tests/constants"
	"ambient-code-backend/tests/logger"

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

			// The sanitized prompt should not contain the raw shell characters
			// from the agent_action and user_correction fields.
			// The prompt template itself contains markdown formatting which is fine.
			Expect(prompt).To(ContainSubstring("Used 'backticks' and VAR"))
			Expect(prompt).To(ContainSubstring("Don't use (angle) brackets"))

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
