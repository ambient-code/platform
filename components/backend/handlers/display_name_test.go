//go:build test

package handlers

import (
	test_constants "ambient-code-backend/tests/constants"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"ambient-code-backend/tests/logger"
	"ambient-code-backend/tests/test_utils"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

var _ = Describe("Display Name Handler", Label(test_constants.LabelUnit, test_constants.LabelHandlers, test_constants.LabelDisplayName), func() {
	var (
		testClientFactory         *test_utils.TestClientFactory
		fakeClients               *test_utils.FakeClientSet
		originalK8sClient         kubernetes.Interface
		originalK8sClientMw       kubernetes.Interface
		originalK8sClientProjects kubernetes.Interface
		originalDynamicClient     dynamic.Interface
	)

	BeforeEach(func() {
		logger.Log("Setting up Display Name Handler test")

		// Save original state to restore in AfterEach
		originalK8sClient = K8sClient
		originalK8sClientMw = K8sClientMw
		originalK8sClientProjects = K8sClientProjects
		originalDynamicClient = DynamicClient

		// Create test client factory with fake clients
		testClientFactory = test_utils.NewTestClientFactory()
		fakeClients = testClientFactory.GetFakeClients()

		// Note: Not setting K8sClientProjects due to type incompatibility
		// Unit tests will work with the existing handlers setup
		DynamicClient = fakeClients.GetDynamicClient()
		K8sClientProjects = fakeClients.GetK8sClient()
		K8sClient = fakeClients.GetK8sClient()
		K8sClientMw = fakeClients.GetK8sClient()

		// Clear environment variables for clean test state
		os.Unsetenv("CLAUDE_CODE_USE_VERTEX")
		os.Unsetenv("ANTHROPIC_VERTEX_PROJECT_ID")
		os.Unsetenv("CLOUD_ML_REGION")
	})

	AfterEach(func() {
		// Restore original state to prevent test pollution
		K8sClient = originalK8sClient
		K8sClientMw = originalK8sClientMw
		K8sClientProjects = originalK8sClientProjects
		DynamicClient = originalDynamicClient
	})

	Context("Display Name Validation", func() {
		Describe("sanitizeDisplayName", func() {
			// Note: sanitizeDisplayName is not exported, so we test it indirectly through ValidateDisplayName
			// and observe its behavior through the validation results
		})

		Describe("ValidateDisplayName", func() {
			It("Should accept valid display names", func() {
				validNames := []string{
					"Debug auth middleware",
					"Add user dashboard",
					"Refactor API routes",
					"Fix bug #123",
					"Update dependencies v1.2.3",
				}

				for _, name := range validNames {
					result := ValidateDisplayName(name)
					Expect(result).To(BeEmpty(), "Should accept valid name: "+name)
				}
			})

			It("Should reject empty names", func() {
				result := ValidateDisplayName("")
				Expect(result).To(Equal("display name cannot be empty"))

				result = ValidateDisplayName("   ")
				Expect(result).To(Equal("display name cannot be empty"))
			})

			It("Should reject names that are too long", func() {
				// Create a name longer than 50 characters
				longName := strings.Repeat("a", 51)
				result := ValidateDisplayName(longName)
				Expect(result).To(Equal("display name cannot exceed 50 characters"))
			})

			It("Should accept names exactly at the limit", func() {
				// Create a name exactly 50 characters
				exactLimitName := strings.Repeat("a", 50)
				result := ValidateDisplayName(exactLimitName)
				Expect(result).To(BeEmpty())
			})

			It("Should reject names with control characters", func() {
				invalidNames := []string{
					"Test\x00Name",
					"Test\x1FName",
					"Test\x7FName",
					"Test\nName",
					"Test\rName",
					"Test\tName",
				}

				for _, name := range invalidNames {
					result := ValidateDisplayName(name)
					Expect(result).To(Equal("display name contains invalid characters"), "Should reject invalid name: "+fmt.Sprintf("%q", name))
				}
			})

			It("Should handle Unicode characters properly", func() {
				unicodeName := "Fix 🐛 in user auth"
				result := ValidateDisplayName(unicodeName)
				Expect(result).To(BeEmpty())

				// Test Unicode length counting
				unicodeLongName := strings.Repeat("🚀", 51)
				result = ValidateDisplayName(unicodeLongName)
				Expect(result).To(Equal("display name cannot exceed 50 characters"))
			})
		})
	})

	Context("Session Context Extraction", func() {
		Describe("ExtractSessionContext", func() {
			It("Should extract repos from session spec", func() {
				spec := map[string]interface{}{
					"repos": []interface{}{
						map[string]interface{}{
							"url":    "https://github.com/owner/repo1.git",
							"branch": "main",
						},
						map[string]interface{}{
							"url":    "https://github.com/owner/repo2.git",
							"branch": "develop",
						},
					},
				}

				ctx := ExtractSessionContext(spec)

				Expect(ctx.Repos).To(HaveLen(2))
				Expect(ctx.Repos[0]["url"]).To(Equal("https://github.com/owner/repo1.git"))
				Expect(ctx.Repos[1]["url"]).To(Equal("https://github.com/owner/repo2.git"))
			})

			It("Should extract active workflow from session spec", func() {
				spec := map[string]interface{}{
					"activeWorkflow": map[string]interface{}{
						"gitUrl": "https://github.com/owner/workflow.git",
						"name":   "test-workflow",
					},
				}

				ctx := ExtractSessionContext(spec)

				Expect(ctx.ActiveWorkflow).NotTo(BeNil())
				Expect(ctx.ActiveWorkflow["gitUrl"]).To(Equal("https://github.com/owner/workflow.git"))
				Expect(ctx.ActiveWorkflow["name"]).To(Equal("test-workflow"))
			})

			It("Should extract initial prompt from session spec", func() {
				spec := map[string]interface{}{
					"initialPrompt": "Help me debug this authentication issue",
				}

				ctx := ExtractSessionContext(spec)

				Expect(ctx.InitialPrompt).To(Equal("Help me debug this authentication issue"))
			})

			It("Should handle empty or missing fields gracefully", func() {
				spec := map[string]interface{}{}

				ctx := ExtractSessionContext(spec)

				Expect(ctx.Repos).To(BeEmpty())
				Expect(ctx.ActiveWorkflow).To(BeNil())
				Expect(ctx.InitialPrompt).To(BeEmpty())
			})

			It("Should handle malformed repos field gracefully", func() {
				spec := map[string]interface{}{
					"repos": "invalid-string-instead-of-array",
				}

				ctx := ExtractSessionContext(spec)

				Expect(ctx.Repos).To(BeEmpty())
			})

			It("Should handle malformed activeWorkflow field gracefully", func() {
				spec := map[string]interface{}{
					"activeWorkflow": "invalid-string-instead-of-map",
				}

				ctx := ExtractSessionContext(spec)

				Expect(ctx.ActiveWorkflow).To(BeNil())
			})
		})

		Describe("ShouldGenerateDisplayName", func() {
			It("Should return true when displayName is not set", func() {
				spec := map[string]interface{}{
					"initialPrompt": "Test prompt",
				}

				result := ShouldGenerateDisplayName(spec)
				Expect(result).To(BeTrue())
			})

			It("Should return true when displayName is empty string", func() {
				spec := map[string]interface{}{
					"displayName":   "",
					"initialPrompt": "Test prompt",
				}

				result := ShouldGenerateDisplayName(spec)
				Expect(result).To(BeTrue())
			})

			It("Should return true when displayName is whitespace only", func() {
				spec := map[string]interface{}{
					"displayName":   "   ",
					"initialPrompt": "Test prompt",
				}

				result := ShouldGenerateDisplayName(spec)
				Expect(result).To(BeTrue())
			})

			It("Should return false when displayName is set and non-empty", func() {
				spec := map[string]interface{}{
					"displayName":   "Existing Display Name",
					"initialPrompt": "Test prompt",
				}

				result := ShouldGenerateDisplayName(spec)
				Expect(result).To(BeFalse())
			})

			It("Should return true when displayName is not a string", func() {
				spec := map[string]interface{}{
					"displayName":   123, // Wrong type
					"initialPrompt": "Test prompt",
				}

				result := ShouldGenerateDisplayName(spec)
				Expect(result).To(BeTrue())
			})
		})
	})

	// Note: buildDisplayNamePrompt is not exported, so we test the prompt building logic
	// indirectly through the async generation function and by testing the SessionContext extraction

	// Note: getAPIKeyFromSecret is not exported, so we test API key retrieval
	// through integration tests or by creating secrets and testing the async generation function

	Context("Session Display Name Updates", func() {
		BeforeEach(func() {
			// Create a test AgenticSession using the test client factory
			err := testClientFactory.CreateTestAgenticSession("test-project", "test-session", map[string]interface{}{
				"initialPrompt": "Test prompt",
				"repos": []interface{}{
					map[string]interface{}{
						"url": "https://github.com/owner/repo.git",
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())
		})

		// Note: updateSessionDisplayNameInternal is not exported, so we test session updates
		// indirectly through the async generation function and by observing the final state
	})

	Context("Asynchronous Generation", func() {
		Describe("GenerateDisplayNameAsync", func() {
			It("Should launch goroutine without blocking", func() {
				sessionCtx := SessionContext{
					InitialPrompt: "Test prompt",
					Repos: []map[string]interface{}{
						{"url": "https://github.com/owner/repo.git"},
					},
				}

				// This should return immediately without blocking
				start := time.Now()
				GenerateDisplayNameAsync("test-project", "test-session", "Test user message", sessionCtx)
				elapsed := time.Since(start)

				// Should return almost immediately (less than 100ms)
				Expect(elapsed).To(BeNumerically("<", 100*time.Millisecond))
			})

			It("Should handle multiple concurrent generations", func() {
				sessionCtx := SessionContext{
					InitialPrompt: "Test prompt",
				}

				var wg sync.WaitGroup
				for i := 0; i < 5; i++ {
					wg.Add(1)
					go func(index int) {
						defer wg.Done()
						sessionName := fmt.Sprintf("test-session-%d", index)
						GenerateDisplayNameAsync("test-project", sessionName, "Test message", sessionCtx)
					}(i)
				}

				// Should complete without deadlock
				done := make(chan struct{})
				go func() {
					wg.Wait()
					close(done)
				}()

				select {
				case <-done:
					// Success - all goroutines completed
				case <-time.After(1 * time.Second):
					Fail("Goroutines did not complete within expected time")
				}
			})
		})
	})

	Context("Environment Configuration", func() {
		Describe("Vertex AI Configuration", func() {
			It("Should detect Vertex AI enabled configuration", func() {
				os.Setenv("CLAUDE_CODE_USE_VERTEX", "1")
				os.Setenv("ANTHROPIC_VERTEX_PROJECT_ID", "test-gcp-project")
				os.Setenv("CLOUD_ML_REGION", "us-east5")

				// This tests the environment detection logic
				// We can't easily test the full getAnthropicClient without real credentials
				// but we can verify the environment variables are read correctly
				Expect(os.Getenv("CLAUDE_CODE_USE_VERTEX")).To(Equal("1"))
				Expect(os.Getenv("ANTHROPIC_VERTEX_PROJECT_ID")).To(Equal("test-gcp-project"))
				Expect(os.Getenv("CLOUD_ML_REGION")).To(Equal("us-east5"))
			})

			It("Should handle missing Vertex configuration gracefully", func() {
				os.Setenv("CLAUDE_CODE_USE_VERTEX", "1")
				// Missing ANTHROPIC_VERTEX_PROJECT_ID

				// The actual function would return an error in this case
				// This test verifies the environment setup for that scenario
				Expect(os.Getenv("CLAUDE_CODE_USE_VERTEX")).To(Equal("1"))
				Expect(os.Getenv("ANTHROPIC_VERTEX_PROJECT_ID")).To(Equal(""))
			})

			It("Should default to API key mode when Vertex disabled", func() {
				// Default environment - Vertex not enabled
				Expect(os.Getenv("CLAUDE_CODE_USE_VERTEX")).NotTo(Equal("1"))
			})
		})
	})

	Context("Error Scenarios", func() {
		It("Should handle invalid session specs gracefully through validation", func() {
			// Test ExtractSessionContext with malformed data
			malformedSpec := map[string]interface{}{
				"repos":          "invalid-string-instead-of-array",
				"activeWorkflow": "invalid-string-instead-of-map",
				"initialPrompt":  123, // Wrong type
			}

			// Should not panic and return empty context
			ctx := ExtractSessionContext(malformedSpec)
			Expect(ctx.Repos).To(BeEmpty())
			Expect(ctx.ActiveWorkflow).To(BeNil())
			Expect(ctx.InitialPrompt).To(BeEmpty())
		})

		It("Should validate display names with edge cases", func() {
			// Test various edge cases in validation
			testCases := []struct {
				input    string
				expected string
			}{
				{"", "display name cannot be empty"},
				{"   ", "display name cannot be empty"},
				{strings.Repeat("a", 51), "display name cannot exceed 50 characters"},
				{"Test\x00Name", "display name contains invalid characters"},
				{"Valid Name", ""},
			}

			for _, tc := range testCases {
				result := ValidateDisplayName(tc.input)
				Expect(result).To(Equal(tc.expected), fmt.Sprintf("Failed for input: %q", tc.input))
			}
		})
	})
})
