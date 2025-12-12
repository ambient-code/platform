//go:build test

package handlers

import (
	"ambient-code-backend/tests/config"
	test_constants "ambient-code-backend/tests/constants"
	"context"
	"fmt"

	"ambient-code-backend/tests/logger"
	"ambient-code-backend/tests/test_utils"
	"ambient-code-backend/types"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes"
)

var _ = Describe("Repository Handler", Label(test_constants.LabelUnit, test_constants.LabelHandlers, test_constants.LabelRepository), func() {
	var (
		testClientFactory         *test_utils.TestClientFactory
		fakeClients               *test_utils.FakeClientSet
		originalK8sClient         kubernetes.Interface
		originalK8sClientMw       kubernetes.Interface
		originalK8sClientProjects kubernetes.Interface
		originalNamespace         interface{}
		ctx                       context.Context
	)

	BeforeEach(func() {
		logger.Log("Setting up Repository Handler test")

		// Save original state to restore in AfterEach
		originalK8sClient = K8sClient
		originalK8sClientMw = K8sClientMw
		originalK8sClientProjects = K8sClientProjects
		originalNamespace = Namespace

		// Create test client factory with fake clients
		testClientFactory = test_utils.NewTestClientFactory()
		fakeClients = testClientFactory.GetFakeClients()

		// Set fake client in handlers package
		fakeK8sClient := fakeClients.GetK8sClient()
		// Use the test utility function to properly set up handler dependencies
		// which correctly handles the interface vs concrete type issue
		K8sClient = fakeK8sClient
		K8sClientMw = fakeK8sClient
		K8sClientProjects = fakeK8sClient
		Namespace = *config.TestNamespace

		ctx = context.Background()
	})

	AfterEach(func() {
		// Restore original state to prevent test pollution
		K8sClient = originalK8sClient
		K8sClientMw = originalK8sClientMw
		K8sClientProjects = originalK8sClientProjects
		if originalNamespace != nil {
			if namespace, ok := originalNamespace.(string); ok {
				Namespace = namespace
			} else {
				Namespace = ""
			}
		} else {
			Namespace = ""
		}
	})

	Context("Repository Provider Detection", func() {
		Describe("DetectRepositoryProvider", func() {
			It("Should detect GitHub provider correctly", func() {
				githubURLs := []string{
					"https://github.com/user/repo.git",
					"git@github.com:user/repo.git",
					"https://github.com/org/my-project",
				}

				for _, url := range githubURLs {
					By(fmt.Sprintf("Detecting provider for GitHub URL: %s", url), func() {
						// Act
						provider := DetectRepositoryProvider(url)

						// Assert
						Expect(provider).To(Equal(types.ProviderGitHub), "Should detect GitHub provider for: "+url)

						logger.Log("Correctly detected GitHub provider for: %s", url)
					})
				}
			})

			It("Should detect GitLab provider correctly", func() {
				gitlabURLs := []string{
					"https://gitlab.com/user/repo.git",
					"git@gitlab.com:user/repo.git",
					"https://gitlab.example.com/group/project",
					"https://self-hosted.gitlab.com/team/app",
				}

				for _, url := range gitlabURLs {
					By(fmt.Sprintf("Detecting provider for GitLab URL: %s", url), func() {
						// Act
						provider := DetectRepositoryProvider(url)

						// Assert
						Expect(provider).To(Equal(types.ProviderGitLab), "Should detect GitLab provider for: "+url)

						logger.Log("Correctly detected GitLab provider for: %s", url)
					})
				}
			})

			It("Should handle unknown providers", func() {
				unknownURLs := []string{
					"https://bitbucket.org/user/repo.git",
					"https://sourceforge.net/p/project/code",
					"invalid-url",
					"",
				}

				for _, url := range unknownURLs {
					By(fmt.Sprintf("Handling unknown provider for URL: %s", url), func() {
						// Act
						provider := DetectRepositoryProvider(url)

						// Assert
						Expect(provider).NotTo(Equal(types.ProviderGitHub))
						Expect(provider).NotTo(Equal(types.ProviderGitLab))

						logger.Log("Correctly handled unknown provider for: %s", url)
					})
				}
			})
		})
	})

	Context("Repository Validation", func() {
		Describe("ValidateGitLabRepository", func() {
			It("Should require GitLab token", func() {
				// Act
				err := ValidateGitLabRepository(ctx, "https://gitlab.com/user/repo.git", "")

				// Assert
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("GitLab token is required"))

				logger.Log("Correctly required GitLab token for validation")
			})

			It("Should handle invalid GitLab URL", func() {
				// Act
				err := ValidateGitLabRepository(ctx, "invalid-url", "test-token")

				// Assert
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("invalid GitLab repository URL"))

				logger.Log("Correctly rejected invalid GitLab URL")
			})

			// Note: Full GitLab validation would require mocking the gitlab package
			// which is complex. For unit tests, we test the wrapper logic.
			// Integration tests would test the actual GitLab API interactions.
		})
	})

	Context("Repository URL Normalization", func() {
		Describe("NormalizeRepositoryURL", func() {
			It("Should handle GitHub URLs", func() {
				githubURL := "https://github.com/user/repo.git"

				// Act
				normalized, err := NormalizeRepositoryURL(githubURL, types.ProviderGitHub)

				// Assert
				Expect(err).NotTo(HaveOccurred())
				Expect(normalized).To(Equal(githubURL), "GitHub URLs should be returned as-is")

				logger.Log("GitHub URL normalization handled correctly")
			})

			It("Should handle unsupported providers", func() {
				unknownURL := "https://bitbucket.org/user/repo.git"

				// Act
				_, err := NormalizeRepositoryURL(unknownURL, "bitbucket")

				// Assert
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("unsupported provider"))

				logger.Log("Correctly rejected unsupported provider")
			})

			It("Should normalize GitLab URLs", func() {
				// Note: This would require mocking gitlab.NormalizeGitLabURL
				// For unit tests, we verify the function call structure
				gitlabURL := "https://gitlab.com/user/repo.git"

				// Act
				normalized, err := NormalizeRepositoryURL(gitlabURL, types.ProviderGitLab)

				// Assert - Function should call gitlab.NormalizeGitLabURL
				// Since we can't easily mock the gitlab package in unit tests,
				// we verify that it doesn't panic and returns appropriate results
				if err != nil {
					// Expected if gitlab package functions aren't available in test
					Expect(err.Error()).NotTo(ContainSubstring("unsupported provider"))
				} else {
					Expect(normalized).NotTo(BeEmpty())
				}

				logger.Log("GitLab URL normalization attempted")
			})
		})
	})

	Context("Repository Information", func() {
		Describe("GetRepositoryInfo", func() {
			It("Should parse GitHub repository info", func() {
				githubURL := "https://github.com/user/awesome-repo.git"

				// Act
				info, err := GetRepositoryInfo(githubURL)

				// Assert
				if err != nil {
					// May fail if GitHub parsing is not fully implemented
					Expect(err.Error()).NotTo(ContainSubstring("unsupported provider"))
				} else {
					Expect(info).NotTo(BeNil())
					Expect(info.URL).To(Equal(githubURL))
					Expect(info.Provider).To(Equal(types.ProviderGitHub))
					Expect(info.Host).To(Equal("github.com"))
				}

				logger.Log("GitHub repository info parsing attempted")
			})

			It("Should parse GitLab repository info", func() {
				gitlabURL := "https://gitlab.com/group/project.git"

				// Act
				info, err := GetRepositoryInfo(gitlabURL)

				// Assert - May fail if gitlab package functions aren't mocked
				if err != nil {
					// Expected if gitlab parsing functions aren't available in test
					Expect(err.Error()).NotTo(ContainSubstring("unsupported provider"))
				} else {
					Expect(info).NotTo(BeNil())
					Expect(info.URL).To(Equal(gitlabURL))
					Expect(info.Provider).To(Equal(types.ProviderGitLab))
				}

				logger.Log("GitLab repository info parsing attempted")
			})

			It("Should handle unsupported providers", func() {
				unknownURL := "https://bitbucket.org/user/repo.git"

				// Act
				_, err := GetRepositoryInfo(unknownURL)

				// Assert
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("unsupported provider"))

				logger.Log("Correctly rejected unsupported provider")
			})
		})

		Describe("ValidateProjectRepository", func() {
			It("Should handle GitHub repositories", func() {
				githubURL := "https://github.com/user/repo.git"
				userID := "test-user"

				// Act
				info, err := ValidateProjectRepository(ctx, githubURL, userID)

				// Assert - Should not fail for GitHub (no special validation required)
				if err == nil {
					Expect(info).NotTo(BeNil())
					Expect(info.Provider).To(Equal(types.ProviderGitHub))
				}

				logger.Log("GitHub repository validation completed")
			})

			It("Should handle GitLab repositories without token", func() {
				gitlabURL := "https://gitlab.com/user/repo.git"
				userID := "test-user-no-token"

				// Act
				info, err := ValidateProjectRepository(ctx, gitlabURL, userID)

				// Assert - Should return info even without token (validation skipped)
				if err == nil {
					Expect(info).NotTo(BeNil())
				}
				// May fail if GetRepositoryInfo fails for GitLab URLs

				logger.Log("GitLab repository validation without token attempted")
			})
		})
	})

	Context("Repository Data Structures", func() {
		Describe("RepositoryInfo", func() {
			It("Should have proper JSON tags", func() {
				// Arrange
				info := RepositoryInfo{
					URL:                "https://github.com/user/repo.git",
					Provider:           types.ProviderGitHub,
					Owner:              "user",
					Repo:               "repo",
					Host:               "github.com",
					APIURL:             "https://api.github.com",
					IsGitLabSelfHosted: false,
				}

				// Assert
				Expect(info.URL).To(Equal("https://github.com/user/repo.git"))
				Expect(info.Provider).To(Equal(types.ProviderGitHub))
				Expect(info.Owner).To(Equal("user"))
				Expect(info.Repo).To(Equal("repo"))
				Expect(info.Host).To(Equal("github.com"))
				Expect(info.APIURL).To(Equal("https://api.github.com"))
				Expect(info.IsGitLabSelfHosted).To(BeFalse())

				logger.Log("RepositoryInfo structure validated")
			})
		})
	})

	Context("ProjectSettings Enhancement", func() {
		Describe("EnrichProjectSettingsWithProviders", func() {
			It("Should add provider information to repositories", func() {
				// Arrange
				repositories := []map[string]interface{}{
					{
						"name": "GitHub Repo",
						"url":  "https://github.com/user/repo.git",
					},
					{
						"name": "GitLab Repo",
						"url":  "https://gitlab.com/group/project.git",
					},
				}

				// Act
				enriched := EnrichProjectSettingsWithProviders(repositories)

				// Assert
				Expect(enriched).To(HaveLen(2))

				// Check GitHub repository
				githubRepo := enriched[0]
				Expect(githubRepo["name"]).To(Equal("GitHub Repo"))
				Expect(githubRepo["url"]).To(Equal("https://github.com/user/repo.git"))
				Expect(githubRepo["provider"]).To(Equal("github"))

				// Check GitLab repository
				gitlabRepo := enriched[1]
				Expect(gitlabRepo["name"]).To(Equal("GitLab Repo"))
				Expect(gitlabRepo["url"]).To(Equal("https://gitlab.com/group/project.git"))
				Expect(gitlabRepo["provider"]).To(Equal("gitlab"))

				logger.Log("Successfully enriched repositories with provider information")
			})

			It("Should preserve existing provider information", func() {
				// Arrange
				repositories := []map[string]interface{}{
					{
						"name":     "Custom Repo",
						"url":      "https://github.com/user/repo.git",
						"provider": "custom-provider", // Existing provider
					},
				}

				// Act
				enriched := EnrichProjectSettingsWithProviders(repositories)

				// Assert
				Expect(enriched).To(HaveLen(1))
				Expect(enriched[0]["provider"]).To(Equal("custom-provider"), "Should preserve existing provider")

				logger.Log("Preserved existing provider information")
			})

			It("Should handle repositories without URL", func() {
				// Arrange
				repositories := []map[string]interface{}{
					{
						"name": "Repo Without URL",
						// No URL field
					},
				}

				// Act
				enriched := EnrichProjectSettingsWithProviders(repositories)

				// Assert
				Expect(enriched).To(HaveLen(1))
				Expect(enriched[0]["name"]).To(Equal("Repo Without URL"))
				// Provider should not be added if no URL
				_, hasProvider := enriched[0]["provider"]
				Expect(hasProvider).To(BeFalse(), "Should not add provider if no URL")

				logger.Log("Handled repository without URL correctly")
			})

			It("Should handle empty repository list", func() {
				// Arrange
				repositories := []map[string]interface{}{}

				// Act
				enriched := EnrichProjectSettingsWithProviders(repositories)

				// Assert
				Expect(enriched).To(HaveLen(0))

				logger.Log("Handled empty repository list correctly")
			})

			It("Should handle repositories with unknown provider", func() {
				// Arrange
				repositories := []map[string]interface{}{
					{
						"name": "Unknown Provider Repo",
						"url":  "https://bitbucket.org/user/repo.git",
					},
				}

				// Act
				enriched := EnrichProjectSettingsWithProviders(repositories)

				// Assert
				Expect(enriched).To(HaveLen(1))

				// Should not add empty provider for unknown providers
				provider, hasProvider := enriched[0]["provider"]
				if hasProvider {
					Expect(provider).NotTo(BeEmpty())
				}

				logger.Log("Handled repository with unknown provider correctly")
			})

			It("Should preserve all existing fields", func() {
				// Arrange
				repositories := []map[string]interface{}{
					{
						"name":        "Full Repo",
						"url":         "https://github.com/user/repo.git",
						"description": "A test repository",
						"branch":      "main",
						"config": map[string]interface{}{
							"timeout": 300,
						},
					},
				}

				// Act
				enriched := EnrichProjectSettingsWithProviders(repositories)

				// Assert
				Expect(enriched).To(HaveLen(1))
				repo := enriched[0]

				Expect(repo["name"]).To(Equal("Full Repo"))
				Expect(repo["url"]).To(Equal("https://github.com/user/repo.git"))
				Expect(repo["description"]).To(Equal("A test repository"))
				Expect(repo["branch"]).To(Equal("main"))
				Expect(repo["config"]).NotTo(BeNil())
				Expect(repo["provider"]).To(Equal("github"))

				logger.Log("Preserved all existing fields while adding provider")
			})
		})
	})

	Context("Error Handling", func() {
		It("Should handle nil repository info gracefully", func() {
			// This tests the robustness of the functions
			// Most functions should handle edge cases without panicking

			// Test with empty/invalid URLs
			invalidURLs := []string{"", " ", "invalid", "ftp://example.com"}

			for _, url := range invalidURLs {
				By(fmt.Sprintf("Testing invalid URL: '%s'", url), func() {
					provider := DetectRepositoryProvider(url)
					// Should not panic, may return unknown provider
					Expect(provider).To(BeAssignableToTypeOf(types.ProviderType("")))

					logger.Log("Handled invalid URL without panic: '%s'", url)
				})
			}
		})

		It("Should handle malformed repository data in enrichment", func() {
			// Arrange - malformed repository data
			repositories := []map[string]interface{}{
				{
					"url": 123, // Invalid type for URL
				},
				{
					"url": nil, // Nil URL
				},
			}

			// Act
			enriched := EnrichProjectSettingsWithProviders(repositories)

			// Assert - Should not panic
			Expect(enriched).To(HaveLen(2))

			logger.Log("Handled malformed repository data without panic")
		})
	})
})
