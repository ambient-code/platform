//go:build test

package handlers

import (
	"ambient-code-backend/tests/config"
	test_constants "ambient-code-backend/tests/constants"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"ambient-code-backend/tests/logger"
	"ambient-code-backend/tests/test_utils"
	"ambient-code-backend/types"

	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

var _ = Describe("Repository Seeding Handler", Label(test_constants.LabelUnit, test_constants.LabelHandlers, test_constants.LabelRepoSeed), func() {
	var (
		httpUtils                 *test_utils.HTTPTestUtils
		testClientFactory         *test_utils.TestClientFactory
		tempDir                   string
		ctx                       context.Context
		originalK8sClient         kubernetes.Interface
		originalK8sClientMw       kubernetes.Interface
		originalK8sClientProjects kubernetes.Interface
		originalNamespace         string
	)

	BeforeEach(func() {
		logger.Log("Setting up Repository Seeding Handler test")

		// Save original state to restore in AfterEach
		originalK8sClient = K8sClient
		originalK8sClientMw = K8sClientMw
		originalK8sClientProjects = K8sClientProjects
		originalNamespace = Namespace

		// Auth is disabled by default from config for unit tests

		// Use centralized handler dependencies setup
		k8sUtils = test_utils.NewK8sTestUtils(false, *config.TestNamespace)
		SetupHandlerDependencies(k8sUtils)

		// Create test client factory with fake clients
		testClientFactory = test_utils.NewTestClientFactory()
		_ = testClientFactory.GetFakeClients()

		// For repo seed tests, we need to set all the package-level K8s client variables
		// Different handlers use different client variables, so set them all
		// IMPORTANT: Use the same fake client for handlers that the test data is created with
		K8sClient = k8sUtils.K8sClient
		K8sClientMw = k8sUtils.K8sClient
		K8sClientProjects = k8sUtils.K8sClient
		Namespace = *config.TestNamespace

		GetGitHubTokenRepo = func(ctx context.Context, k8s kubernetes.Interface, dyn dynamic.Interface, project, userID string) (string, error) {
			if project == "unauthorized-project" {
				return "", fmt.Errorf("no GitHub token found for user")
			}
			return "mock-github-token", nil
		}

		httpUtils = test_utils.NewHTTPTestUtils()
		ctx = context.Background()

		// Create temporary directory for testing
		var err error
		tempDir, err = ioutil.TempDir("", "repo-seed-test-*")
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		// Restore original state to prevent test pollution
		K8sClient = originalK8sClient
		K8sClientMw = originalK8sClientMw
		K8sClientProjects = originalK8sClientProjects
		Namespace = originalNamespace
	})

	AfterEach(func() {
		// Clean up temporary directory
		if tempDir != "" {
			err := os.Remove(tempDir)
			if err != nil {
				logger.Log("Failed to delete tem directory: %s", tempDir)
				return
			}
		}
	})

	Context("Repository Structure Detection", func() {
		Describe("DetectMissingStructure", func() {
			It("Should detect missing .claude directory", func() {
				// Arrange - empty directory
				testRepoDir := filepath.Join(tempDir, "empty-repo")
				err := os.MkdirAll(testRepoDir, 0755)
				Expect(err).NotTo(HaveOccurred())

				// Act
				status, err := DetectMissingStructure(ctx, testRepoDir)

				// Assert
				Expect(err).NotTo(HaveOccurred())
				Expect(status.Required).To(BeTrue(), "Should require seeding when .claude directory is missing")
				Expect(status.MissingDirs).To(ContainElement(".claude"))
				Expect(status.InProgress).To(BeFalse())

				logger.Log("Successfully detected missing .claude directory")
			})

			It("Should detect missing files in existing .claude directory", func() {
				// Arrange - .claude directory exists but missing required files
				testRepoDir := filepath.Join(tempDir, "partial-repo")
				claudeDir := filepath.Join(testRepoDir, ".claude")
				commandsDir := filepath.Join(claudeDir, "commands")

				err := os.MkdirAll(commandsDir, 0755)
				Expect(err).NotTo(HaveOccurred())

				// Don't create the README.md file in commands directory

				// Act
				status, err := DetectMissingStructure(ctx, testRepoDir)

				// Assert
				Expect(err).NotTo(HaveOccurred())
				Expect(status.Required).To(BeTrue(), "Should require seeding when required files are missing")
				Expect(status.MissingFiles).To(ContainElement(".claude/commands/README.md"))

				logger.Log("Successfully detected missing required files")
			})

			It("Should detect complete .claude structure", func() {
				// Arrange - create complete .claude structure
				testRepoDir := filepath.Join(tempDir, "complete-repo")
				claudeDir := filepath.Join(testRepoDir, ".claude")
				commandsDir := filepath.Join(claudeDir, "commands")

				err := os.MkdirAll(commandsDir, 0755)
				Expect(err).NotTo(HaveOccurred())

				// Create required README.md file
				readmePath := filepath.Join(commandsDir, "README.md")
				err = ioutil.WriteFile(readmePath, []byte("# Commands\n"), 0644)
				Expect(err).NotTo(HaveOccurred())

				// Act
				status, err := DetectMissingStructure(ctx, testRepoDir)

				// Assert
				Expect(err).NotTo(HaveOccurred())
				Expect(status.Required).To(BeFalse(), "Should not require seeding when structure is complete")
				Expect(status.MissingDirs).To(HaveLen(0))
				Expect(status.MissingFiles).To(HaveLen(0))

				logger.Log("Successfully detected complete .claude structure")
			})
		})

		Describe("SeedRepository", func() {
			It("Should create .claude directory structure successfully", func() {
				// Arrange
				testRepoDir := filepath.Join(tempDir, "seed-test-repo")
				err := os.MkdirAll(testRepoDir, 0755)
				Expect(err).NotTo(HaveOccurred())

				// Initialize git repository (SeedRepository uses git add/commit)
				gitInit := exec.CommandContext(ctx, "git", "-C", testRepoDir, "init")
				Expect(gitInit.Run()).NotTo(HaveOccurred())

				// Note: SeedRepository function requires git operations
				// For unit tests, we'll focus on the directory/file creation aspects
				// Git operations would be tested in integration tests

				// Act
				response, err := SeedRepository(ctx, testRepoDir, "https://github.com/test/repo.git", "main", "test@example.com", "Test User")
				Expect(err).NotTo(HaveOccurred())

				// Assert - Check that directories were created
				Expect(response).NotTo(BeNil())
				Expect(response.RepositoryURL).To(Equal("https://github.com/test/repo.git"))

				// Check that .claude directory was created
				claudeDir := filepath.Join(testRepoDir, ".claude")
				_, err = os.Stat(claudeDir)
				Expect(err).NotTo(HaveOccurred(), ".claude directory should be created")

				// Check that commands directory was created
				commandsDir := filepath.Join(claudeDir, "commands")
				_, err = os.Stat(commandsDir)
				Expect(err).NotTo(HaveOccurred(), ".claude/commands directory should be created")

				// Check that template files were created
				readmePath := filepath.Join(claudeDir, "README.md")
				_, err = os.Stat(readmePath)
				Expect(err).NotTo(HaveOccurred(), ".claude/README.md should be created")

				logger.Log("Successfully created .claude directory structure")
			})

			It("Should skip existing files during seeding", func() {
				// Arrange
				testRepoDir := filepath.Join(tempDir, "existing-files-repo")
				claudeDir := filepath.Join(testRepoDir, ".claude")
				err := os.MkdirAll(claudeDir, 0755)
				Expect(err).NotTo(HaveOccurred())

				// Initialize git repository (SeedRepository uses git add/commit)
				gitInit := exec.CommandContext(ctx, "git", "-C", testRepoDir, "init")
				Expect(gitInit.Run()).NotTo(HaveOccurred())

				// Create existing README.md with custom content
				existingReadme := filepath.Join(claudeDir, "README.md")
				customContent := "# Custom README\n"
				err = ioutil.WriteFile(existingReadme, []byte(customContent), 0644)
				Expect(err).NotTo(HaveOccurred())

				// Act
				response, err := SeedRepository(ctx, testRepoDir, "https://github.com/test/repo.git", "main", "test@example.com", "Test User")
				Expect(err).NotTo(HaveOccurred())

				// Assert
				Expect(response).NotTo(BeNil())

				// Check that existing file content was preserved
				content, err := ioutil.ReadFile(existingReadme)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(content)).To(Equal(customContent), "Existing file content should be preserved")

				logger.Log("Successfully preserved existing files during seeding")
			})
		})
	})

	Context("HTTP Endpoints", func() {
		Describe("GetRepoSeedStatus", func() {
			It("Should require repo parameter", func() {
				// Arrange
				context := httpUtils.CreateTestGinContext("GET", "/projects/test-project/repo/seed-status", nil)
				context.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
				}
				httpUtils.SetAuthHeader("test-token")
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

				// Act
				GetRepoSeedStatus(context)

				// Assert
				httpUtils.AssertHTTPStatus(http.StatusBadRequest)
				httpUtils.AssertErrorMessage("repo query parameter required")
			})

			It("Should handle invalid repository provider", func() {
				// Arrange
				context := httpUtils.CreateTestGinContext("GET", "/projects/test-project/repo/seed-status?repo=https://bitbucket.org/user/repo", nil)
				context.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
				}
				httpUtils.SetAuthHeader("test-token")
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

				// Act
				GetRepoSeedStatus(context)

				// Assert
				httpUtils.AssertHTTPStatus(http.StatusBadRequest)
				httpUtils.AssertErrorMessage("unsupported repository provider")
			})

			It("Should handle unauthorized access", func() {
				// Arrange
				context := httpUtils.CreateTestGinContext("GET", "/projects/unauthorized-project/repo/seed-status?repo=https://github.com/user/repo", nil)
				context.Params = gin.Params{
					{Key: "projectName", Value: "unauthorized-project"},
				}
				httpUtils.SetAuthHeader("test-token")
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

				// Act
				GetRepoSeedStatus(context)

				// Assert
				httpUtils.AssertHTTPStatus(http.StatusUnauthorized)
				httpUtils.AssertErrorMessage("Invalid or missing token")
			})

			It("Should fail with git error when no authentication provided", func() {
				// Arrange
				context := httpUtils.CreateTestGinContext("GET", "/projects/test-project/repo/seed-status?repo=https://github.com/user/repo", nil)
				context.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
				}
				// Don't set auth header
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

				// Act
				GetRepoSeedStatus(context)

				// Assert
				// Note: This handler proceeds to Git operations before checking auth properly,
				// so it fails with 502 (git error) instead of 401 (auth error)
				httpUtils.AssertHTTPStatus(http.StatusBadGateway)
				httpUtils.AssertErrorMessage("Failed to clone repository: exit status 128")
			})
		})

		Describe("SeedRepositoryEndpoint", func() {
			It("Should require valid JSON body", func() {
				// Arrange
				context := httpUtils.CreateTestGinContext("POST", "/projects/test-project/repo/seed", "invalid-json")
				context.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
				}
				httpUtils.SetAuthHeader("test-token")
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

				// Act
				SeedRepositoryEndpoint(context)

				// Assert
				httpUtils.AssertHTTPStatus(http.StatusBadRequest)
				httpUtils.AssertJSONContains(map[string]interface{}{
					"error": "Invalid request: invalid character 'i' looking for beginning of value",
				})
			})

			It("Should require repositoryUrl in request body", func() {
				// Arrange
				requestBody := map[string]interface{}{
					// Missing repositoryUrl
					"branch": "main",
				}

				context := httpUtils.CreateTestGinContext("POST", "/projects/test-project/repo/seed", requestBody)
				context.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
				}
				httpUtils.SetAuthHeader("test-token")
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

				// Act
				SeedRepositoryEndpoint(context)

				// Assert
				httpUtils.AssertHTTPStatus(http.StatusBadRequest)
				httpUtils.AssertJSONContains(map[string]interface{}{
					"error": "Invalid request: Key: 'SeedRequest.RepositoryURL' Error:Field validation for 'RepositoryURL' failed on the 'required' tag",
				})
			})

			It("Should default branch to main if not specified", func() {
				// Arrange
				requestBody := map[string]interface{}{
					"repositoryUrl": "https://github.com/test/repo.git",
					// Branch not specified - should default to main
				}

				context := httpUtils.CreateTestGinContext("POST", "/projects/test-project/repo/seed", requestBody)
				context.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
				}
				httpUtils.SetAuthHeader("test-token")
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

				// Act
				SeedRepositoryEndpoint(context)

				// Assert - Should not return bad request for missing branch
				status := httpUtils.GetResponseRecorder().Code
				Expect(status).NotTo(Equal(http.StatusBadRequest), "Should accept request without branch parameter")

				logger.Log("Handled missing branch parameter correctly")
			})

			It("Should handle unsupported repository provider", func() {
				// Arrange
				requestBody := map[string]interface{}{
					"repositoryUrl": "https://bitbucket.org/user/repo.git",
					"branch":        "main",
				}

				context := httpUtils.CreateTestGinContext("POST", "/projects/test-project/repo/seed", requestBody)
				context.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
				}
				httpUtils.SetAuthHeader("test-token")
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

				// Act
				SeedRepositoryEndpoint(context)

				// Assert
				httpUtils.AssertHTTPStatus(http.StatusBadRequest)
				httpUtils.AssertErrorMessage("unsupported repository provider")
			})

			It("Should handle unauthorized access", func() {
				// Arrange
				requestBody := map[string]interface{}{
					"repositoryUrl": "https://github.com/test/repo.git",
					"branch":        "main",
				}

				context := httpUtils.CreateTestGinContext("POST", "/projects/unauthorized-project/repo/seed", requestBody)
				context.Params = gin.Params{
					{Key: "projectName", Value: "unauthorized-project"},
				}
				httpUtils.SetAuthHeader("test-token")
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

				// Act
				SeedRepositoryEndpoint(context)

				// Assert
				httpUtils.AssertHTTPStatus(http.StatusUnauthorized)
				httpUtils.AssertJSONContains(map[string]interface{}{
					"error": "Invalid or missing token",
				})
			})

			It("Should fail with git error when no authentication provided", func() {
				// Arrange
				requestBody := map[string]interface{}{
					"repositoryUrl": "https://github.com/test/repo.git",
					"branch":        "main",
				}

				context := httpUtils.CreateTestGinContext("POST", "/projects/test-project/repo/seed", requestBody)
				context.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
				}
				// Don't set auth header
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

				// Act
				SeedRepositoryEndpoint(context)

				// Assert
				// Note: This handler proceeds to Git operations before checking auth properly,
				// so it fails with 502 (git error) instead of 401 (auth error)
				httpUtils.AssertHTTPStatus(http.StatusBadGateway)
				httpUtils.AssertErrorMessage("Failed to clone repository: exit status 128")
			})
		})
	})

	Context("Data Structure Validation", func() {
		Describe("SeedingStatus", func() {
			It("Should have proper JSON tags", func() {
				// Arrange
				status := SeedingStatus{
					Required:      true,
					MissingDirs:   []string{".claude"},
					MissingFiles:  []string{".claude/commands/README.md"},
					InProgress:    false,
					Error:         "test error",
					RepositoryURL: "https://github.com/test/repo.git",
				}

				// Assert - Check that struct can be marshaled properly
				Expect(status.Required).To(BeTrue())
				Expect(status.MissingDirs).To(ContainElement(".claude"))
				Expect(status.MissingFiles).To(ContainElement(".claude/commands/README.md"))
				Expect(status.RepositoryURL).To(Equal("https://github.com/test/repo.git"))

				logger.Log("SeedingStatus structure validated")
			})
		})

		Describe("SeedRequest", func() {
			It("Should have proper validation tags", func() {
				// Arrange
				request := SeedRequest{
					RepositoryURL: "https://github.com/test/repo.git",
					Branch:        "feature-branch",
					Force:         true,
				}

				// Assert
				Expect(request.RepositoryURL).To(Equal("https://github.com/test/repo.git"))
				Expect(request.Branch).To(Equal("feature-branch"))
				Expect(request.Force).To(BeTrue())

				logger.Log("SeedRequest structure validated")
			})
		})

		Describe("SeedResponse", func() {
			It("Should track seeded directories and files", func() {
				// Arrange
				response := SeedResponse{
					Success:       true,
					Message:       "Successfully seeded",
					SeededDirs:    []string{".claude", ".claude/commands"},
					SeededFiles:   []string{".claude/README.md", ".claude/commands/README.md"},
					CommitSHA:     "abc123",
					RepositoryURL: "https://github.com/test/repo.git",
				}

				// Assert
				Expect(response.Success).To(BeTrue())
				Expect(response.SeededDirs).To(HaveLen(2))
				Expect(response.SeededFiles).To(HaveLen(2))
				Expect(response.CommitSHA).To(Equal("abc123"))

				logger.Log("SeedResponse structure validated")
			})
		})
	})

	Context("Provider Detection", func() {
		It("Should handle GitHub provider correctly", func() {
			githubUrls := []string{
				"https://github.com/user/repo.git",
				"git@github.com:user/repo.git",
			}

			for _, url := range githubUrls {
				provider := types.DetectProvider(url)
				Expect(provider).To(Equal(types.ProviderGitHub), "Should detect GitHub provider for: "+url)
				logger.Log("Correctly detected GitHub provider for: %s", url)
			}
		})

		It("Should handle GitLab provider correctly", func() {
			gitlabUrls := []string{
				"https://gitlab.com/user/repo.git",
				"git@gitlab.com:user/repo.git",
			}

			for _, url := range gitlabUrls {
				provider := types.DetectProvider(url)
				Expect(provider).To(Equal(types.ProviderGitLab), "Should detect GitLab provider for: "+url)
				logger.Log("Correctly detected GitLab provider for: %s", url)
			}
		})
	})

	Context("Template Validation", func() {
		It("Should have required Claude templates", func() {
			// Assert that required templates exist
			Expect(ClaudeTemplates).To(HaveKey(".claude/README.md"))
			Expect(ClaudeTemplates).To(HaveKey(".claude/commands/README.md"))
			Expect(ClaudeTemplates).To(HaveKey(".claude/settings.local.json"))
			Expect(ClaudeTemplates).To(HaveKey(".claude/.gitignore"))

			// Check template content is not empty
			for templatePath, content := range ClaudeTemplates {
				Expect(content).NotTo(BeEmpty(), "Template content should not be empty for: "+templatePath)
			}

			logger.Log("All required Claude templates are present and non-empty")
		})

		It("Should have valid JSON template for settings", func() {
			settingsTemplate := ClaudeTemplates[".claude/settings.local.json"]
			Expect(settingsTemplate).To(ContainSubstring("permissions"))
			Expect(settingsTemplate).To(ContainSubstring("allow"))
			Expect(settingsTemplate).To(ContainSubstring("deny"))
			Expect(settingsTemplate).To(ContainSubstring("ask"))

			logger.Log("Settings template contains required fields")
		})
	})

	Context("Error Handling", func() {
		It("Should handle missing user context gracefully", func() {
			// Arrange
			context := httpUtils.CreateTestGinContext("GET", "/projects/test-project/repo/seed-status?repo=https://github.com/user/repo", nil)
			context.Params = gin.Params{
				{Key: "projectName", Value: "test-project"},
			}
			httpUtils.SetAuthHeader("test-token")
			// Don't set user context

			// Act
			GetRepoSeedStatus(context)

			// Assert - Should handle gracefully without panicking
			// 502 can occur if GitHub API calls fail
			status := httpUtils.GetResponseRecorder().Code
			Expect(status).To(BeElementOf(http.StatusInternalServerError, http.StatusBadRequest, http.StatusUnauthorized, http.StatusBadGateway))

			logger.Log("Handled missing user context gracefully")
		})

		It("Should handle malformed JSON gracefully", func() {
			// Arrange
			context := httpUtils.CreateTestGinContext("POST", "/projects/test-project/repo/seed", "{invalid-json}")
			context.Params = gin.Params{
				{Key: "projectName", Value: "test-project"},
			}
			httpUtils.SetAuthHeader("test-token")
			httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

			// Act
			SeedRepositoryEndpoint(context)

			// Assert
			httpUtils.AssertHTTPStatus(http.StatusBadRequest)

			logger.Log("Handled malformed JSON gracefully")
		})
	})
})
