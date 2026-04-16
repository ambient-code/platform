//go:build test

package handlers

import (
	"context"
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

		// Enable the feature flag for tests (Unleash is not configured in test env)
		IsFeedbackLoopEnabled = func() bool { return true }

		ctx := context.Background()
		_, err := k8sUtils.K8sClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: "test-project"},
		}, metav1.CreateOptions{})
		if err != nil && !errors.IsAlreadyExists(err) {
			Expect(err).NotTo(HaveOccurred())
		}
	})

	AfterEach(func() {
		// Restore default feature flag behavior
		IsFeedbackLoopEnabled = func() bool { return FeatureEnabled(feedbackLoopFeatureFlag) }

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

			// More corrections should NOT trigger again (dedup within window)
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

			logger.Log("Deduplication prevents multiple triggers within window")
		})

		It("Should not trigger when feature flag is disabled", func() {
			IsFeedbackLoopEnabled = func() bool { return false }

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

			logger.Log("Feature flag disabled prevents triggering")
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

			// One correction for target A (workflow)
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

			// One correction for target B (repo)
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

			// Verify history has only target B
			entries, err := loadFeedbackLoopHistory(ctx, "test-project")
			Expect(err).NotTo(HaveOccurred())
			Expect(entries).To(HaveLen(1))
			Expect(entries[0].TargetRepoURL).To(Equal("https://github.com/org/other-repo"))
			Expect(entries[0].TargetType).To(Equal("repo"))
			Expect(entries[0].Source).To(Equal("event-driven"))

			logger.Log("Targets tracked independently")
		})

		It("Should record session labels correctly", func() {
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
			Expect(triggered).To(BeTrue())

			// Verify the session was created with correct labels
			gvr := GetAgenticSessionV1Alpha1Resource()
			sessions, err := k8sUtils.DynamicClient.Resource(gvr).Namespace("test-project").List(ctx, metav1.ListOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(sessions.Items).To(HaveLen(1))

			session := sessions.Items[0]
			labels := session.GetLabels()
			Expect(labels["feedback-loop"]).To(Equal("true"))
			Expect(labels["source"]).To(Equal("event-driven"))
			Expect(labels["target-type"]).To(Equal("workflow"))

			logger.Log("Session labels set correctly")
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

		It("Should handle empty URL", func() {
			name := buildSessionDisplayName("repo", "", "")
			Expect(name).To(Equal("Feedback Loop: unknown (repo)"))
		})
	})
})
