//go:build test

package handlers

import (
	"encoding/json"
	"time"

	test_constants "ambient-code-backend/tests/constants"
	"ambient-code-backend/types"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Scheduled Sessions Helpers", Label(test_constants.LabelUnit, test_constants.LabelHandlers, test_constants.LabelSessions), func() {

	Describe("sanitizeLabelValue", func() {
		Context("When given an empty string", func() {
			It("Should return 'unknown'", func() {
				Expect(sanitizeLabelValue("")).To(Equal("unknown"))
			})
		})

		Context("When given a normal alphanumeric string", func() {
			It("Should pass through unchanged", func() {
				Expect(sanitizeLabelValue("hello123")).To(Equal("hello123"))
			})
		})

		Context("When given a string with special characters", func() {
			It("Should replace special characters with underscores", func() {
				result := sanitizeLabelValue("user@example.com")
				Expect(result).To(Equal("user_example.com"))
			})
		})

		Context("When given a string over 63 characters", func() {
			It("Should truncate to at most 63 characters", func() {
				long := "a"
				for i := 0; i < 100; i++ {
					long += "b"
				}
				result := sanitizeLabelValue(long)
				Expect(len(result)).To(BeNumerically("<=", 63))
			})
		})

		Context("When given a string with non-alphanumeric start/end", func() {
			It("Should trim non-alphanumeric characters from start and end", func() {
				result := sanitizeLabelValue("_hello_")
				Expect(result).To(Equal("hello"))
			})

			It("Should trim hyphens from start and end", func() {
				result := sanitizeLabelValue("-test-")
				Expect(result).To(Equal("test"))
			})
		})

		Context("When given a string of all special characters", func() {
			It("Should return 'unknown'", func() {
				result := sanitizeLabelValue("@#$%^&*()")
				Expect(result).To(Equal("unknown"))
			})
		})

		Context("When given valid label characters", func() {
			It("Should allow hyphens, underscores, and dots in the middle", func() {
				result := sanitizeLabelValue("a-b_c.d")
				Expect(result).To(Equal("a-b_c.d"))
			})
		})
	})

	Describe("isAlphanumeric", func() {
		Context("When given lowercase letters", func() {
			It("Should return true", func() {
				Expect(isAlphanumeric('a')).To(BeTrue())
				Expect(isAlphanumeric('z')).To(BeTrue())
				Expect(isAlphanumeric('m')).To(BeTrue())
			})
		})

		Context("When given uppercase letters", func() {
			It("Should return true", func() {
				Expect(isAlphanumeric('A')).To(BeTrue())
				Expect(isAlphanumeric('Z')).To(BeTrue())
				Expect(isAlphanumeric('M')).To(BeTrue())
			})
		})

		Context("When given digits", func() {
			It("Should return true", func() {
				Expect(isAlphanumeric('0')).To(BeTrue())
				Expect(isAlphanumeric('9')).To(BeTrue())
				Expect(isAlphanumeric('5')).To(BeTrue())
			})
		})

		Context("When given non-alphanumeric characters", func() {
			It("Should return false", func() {
				Expect(isAlphanumeric('-')).To(BeFalse())
				Expect(isAlphanumeric('_')).To(BeFalse())
				Expect(isAlphanumeric('.')).To(BeFalse())
				Expect(isAlphanumeric('@')).To(BeFalse())
				Expect(isAlphanumeric(' ')).To(BeFalse())
			})
		})
	})

	Describe("isValidCronExpression", func() {
		It("should accept standard 5-field cron expressions", func() {
			Expect(isValidCronExpression("0 * * * *")).To(BeTrue())
			Expect(isValidCronExpression("0 9 * * 1-5")).To(BeTrue())
			Expect(isValidCronExpression("*/15 * * * *")).To(BeTrue())
			Expect(isValidCronExpression("0 0 1 * *")).To(BeTrue())
		})

		It("should reject empty strings", func() {
			Expect(isValidCronExpression("")).To(BeFalse())
			Expect(isValidCronExpression("   ")).To(BeFalse())
		})

		It("should reject expressions with wrong number of fields", func() {
			Expect(isValidCronExpression("* * *")).To(BeFalse())
			Expect(isValidCronExpression("0 0 0 0 0 0")).To(BeFalse())
			Expect(isValidCronExpression("*")).To(BeFalse())
		})

		It("should reject expressions with invalid characters", func() {
			Expect(isValidCronExpression("0 9 * * ; rm -rf /")).To(BeFalse())
			Expect(isValidCronExpression("0 9 * * `cmd`")).To(BeFalse())
		})

		It("should accept named days and months", func() {
			Expect(isValidCronExpression("0 9 * * MON")).To(BeTrue())
			Expect(isValidCronExpression("0 9 1 JAN *")).To(BeTrue())
		})
	})

	Describe("cronJobToScheduledSession", func() {
		Context("When given a CronJob with all fields populated", func() {
			It("Should convert all fields correctly", func() {
				suspend := true
				lastSchedule := metav1.Now()

				sessionTemplate := types.CreateAgenticSessionRequest{
					InitialPrompt: "Run daily check",
					DisplayName:   "Daily Check",
				}
				templateJSON, err := json.Marshal(sessionTemplate)
				Expect(err).NotTo(HaveOccurred())

				cj := &batchv1.CronJob{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "schedule-123",
						Namespace:         "test-project",
						CreationTimestamp: metav1.Now(),
						Labels: map[string]string{
							labelScheduledSession: "true",
							labelCreatedBy:        "testuser",
						},
						Annotations: map[string]string{
							annotationDisplayName: "My Daily Schedule",
						},
					},
					Spec: batchv1.CronJobSpec{
						Schedule: "0 9 * * *",
						Suspend:  &suspend,
						JobTemplate: batchv1.JobTemplateSpec{
							Spec: batchv1.JobSpec{
								Template: corev1.PodTemplateSpec{
									Spec: corev1.PodSpec{
										Containers: []corev1.Container{
											{
												Name: "trigger",
												Env: []corev1.EnvVar{
													{Name: "SESSION_TEMPLATE", Value: string(templateJSON)},
													{Name: "PROJECT_NAMESPACE", Value: "test-project"},
												},
											},
										},
									},
								},
							},
						},
					},
					Status: batchv1.CronJobStatus{
						LastScheduleTime: &lastSchedule,
						Active: []corev1.ObjectReference{
							{Name: "job-1"},
						},
					},
				}

				result := cronJobToScheduledSession(cj)

				Expect(result.Name).To(Equal("schedule-123"))
				Expect(result.Namespace).To(Equal("test-project"))
				Expect(result.Schedule).To(Equal("0 9 * * *"))
				Expect(result.Suspend).To(BeTrue())
				Expect(result.DisplayName).To(Equal("My Daily Schedule"))
				Expect(result.ActiveCount).To(Equal(1))
				Expect(result.LastScheduleTime).NotTo(BeNil())
				Expect(result.Labels).To(HaveKey(labelScheduledSession))
				Expect(result.Annotations).To(HaveKey(annotationDisplayName))
				Expect(result.SessionTemplate.InitialPrompt).To(Equal("Run daily check"))
				Expect(result.SessionTemplate.DisplayName).To(Equal("Daily Check"))
			})
		})

		Context("When Suspend is nil", func() {
			It("Should default Suspend to false", func() {
				cj := &batchv1.CronJob{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "test-cj",
						Namespace:         "ns",
						CreationTimestamp: metav1.Now(),
					},
					Spec: batchv1.CronJobSpec{
						Schedule: "*/5 * * * *",
						Suspend:  nil,
						JobTemplate: batchv1.JobTemplateSpec{
							Spec: batchv1.JobSpec{
								Template: corev1.PodTemplateSpec{
									Spec: corev1.PodSpec{
										Containers: []corev1.Container{},
									},
								},
							},
						},
					},
				}

				result := cronJobToScheduledSession(cj)

				Expect(result.Suspend).To(BeFalse())
			})
		})

		Context("When LastScheduleTime is nil", func() {
			It("Should leave LastScheduleTime as nil", func() {
				cj := &batchv1.CronJob{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "test-cj",
						Namespace:         "ns",
						CreationTimestamp: metav1.Now(),
					},
					Spec: batchv1.CronJobSpec{
						Schedule: "*/5 * * * *",
						JobTemplate: batchv1.JobTemplateSpec{
							Spec: batchv1.JobSpec{
								Template: corev1.PodTemplateSpec{
									Spec: corev1.PodSpec{
										Containers: []corev1.Container{},
									},
								},
							},
						},
					},
					Status: batchv1.CronJobStatus{
						LastScheduleTime: nil,
					},
				}

				result := cronJobToScheduledSession(cj)

				Expect(result.LastScheduleTime).To(BeNil())
			})
		})

		Context("When extracting SESSION_TEMPLATE from trigger container", func() {
			It("Should parse the session template from env vars", func() {
				tmpl := types.CreateAgenticSessionRequest{
					InitialPrompt: "test prompt",
					RunnerType:    "claude",
				}
				templateJSON, err := json.Marshal(tmpl)
				Expect(err).NotTo(HaveOccurred())

				cj := &batchv1.CronJob{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "test-cj",
						Namespace:         "ns",
						CreationTimestamp: metav1.Now(),
					},
					Spec: batchv1.CronJobSpec{
						Schedule: "0 0 * * *",
						JobTemplate: batchv1.JobTemplateSpec{
							Spec: batchv1.JobSpec{
								Template: corev1.PodTemplateSpec{
									Spec: corev1.PodSpec{
										Containers: []corev1.Container{
											{
												Name: "sidecar",
												Env: []corev1.EnvVar{
													{Name: "SESSION_TEMPLATE", Value: "should-not-be-used"},
												},
											},
											{
												Name: "trigger",
												Env: []corev1.EnvVar{
													{Name: "OTHER_VAR", Value: "other"},
													{Name: "SESSION_TEMPLATE", Value: string(templateJSON)},
												},
											},
										},
									},
								},
							},
						},
					},
				}

				result := cronJobToScheduledSession(cj)

				Expect(result.SessionTemplate.InitialPrompt).To(Equal("test prompt"))
				Expect(result.SessionTemplate.RunnerType).To(Equal("claude"))
			})
		})

		Context("When annotations and labels are nil", func() {
			It("Should handle missing annotations and labels gracefully", func() {
				cj := &batchv1.CronJob{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "test-cj",
						Namespace:         "ns",
						CreationTimestamp: metav1.Now(),
						Labels:            nil,
						Annotations:       nil,
					},
					Spec: batchv1.CronJobSpec{
						Schedule: "0 0 * * *",
						JobTemplate: batchv1.JobTemplateSpec{
							Spec: batchv1.JobSpec{
								Template: corev1.PodTemplateSpec{
									Spec: corev1.PodSpec{
										Containers: []corev1.Container{},
									},
								},
							},
						},
					},
				}

				result := cronJobToScheduledSession(cj)

				Expect(result.Labels).To(BeNil())
				Expect(result.Annotations).To(BeNil())
				Expect(result.DisplayName).To(BeEmpty())
			})
		})

		Context("When CreationTimestamp is set", func() {
			It("should format CreationTimestamp as RFC3339", func() {
				fixedTime := time.Date(2026, 3, 9, 12, 0, 0, 0, time.UTC)
				cj := &batchv1.CronJob{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "test-cj",
						Namespace:         "ns",
						CreationTimestamp: metav1.NewTime(fixedTime),
					},
					Spec: batchv1.CronJobSpec{
						Schedule: "0 0 * * *",
						JobTemplate: batchv1.JobTemplateSpec{
							Spec: batchv1.JobSpec{
								Template: corev1.PodTemplateSpec{
									Spec: corev1.PodSpec{
										Containers: []corev1.Container{},
									},
								},
							},
						},
					},
				}

				result := cronJobToScheduledSession(cj)

				Expect(result.CreationTimestamp).To(Equal("2026-03-09T12:00:00Z"))
			})
		})
	})
})
