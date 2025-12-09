package unit

import (
	test_constants "ambient-code-backend/test/constants"
	"context"
	"net/http"
	"os"
	"path/filepath"

	"ambient-code-backend/git"
	"ambient-code-backend/handlers"
	"ambient-code-backend/test/logger"
	"ambient-code-backend/test/test_utils"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Content Handler", Label(test_constants.LabelUnit, test_constants.LabelHandlers, test_constants.LabelContent), func() {
	var (
		httpUtils        *test_utils.HTTPTestUtils
		originalStateDir string
		tempStateDir     string

		// Store original git function implementations
		originalGitPushRepo           func(ctx context.Context, repoDir, commitMessage, outputRepoURL, branch, githubToken string) (string, error)
		originalGitAbandonRepo        func(ctx context.Context, repoDir string) error
		originalGitDiffRepo           func(ctx context.Context, repoDir string) (*git.DiffSummary, error)
		originalGitCheckMergeStatus   func(ctx context.Context, repoDir, branch string) (*git.MergeStatus, error)
		originalGitPullRepo           func(ctx context.Context, repoDir, branch string) error
		originalGitPushToRepo         func(ctx context.Context, repoDir, branch, commitMessage string) error
		originalGitCreateBranch       func(ctx context.Context, repoDir, branchName string) error
		originalGitListRemoteBranches func(ctx context.Context, repoDir string) ([]string, error)
	)

	BeforeEach(func() {
		logger.Log("Setting up Content Handler test")
		httpUtils = test_utils.NewHTTPTestUtils()

		// Create temporary state directory
		var err error
		tempStateDir, err = os.MkdirTemp("", "content-test-*")
		Expect(err).NotTo(HaveOccurred())

		// Store original values
		originalStateDir = handlers.StateBaseDir

		// Set test state directory
		handlers.StateBaseDir = tempStateDir

		// Store original git function implementations
		originalGitPushRepo = handlers.GitPushRepo
		originalGitAbandonRepo = handlers.GitAbandonRepo
		originalGitDiffRepo = handlers.GitDiffRepo
		originalGitCheckMergeStatus = handlers.GitCheckMergeStatus
		originalGitPullRepo = handlers.GitPullRepo
		originalGitPushToRepo = handlers.GitPushToRepo
		originalGitCreateBranch = handlers.GitCreateBranch
		originalGitListRemoteBranches = handlers.GitListRemoteBranches
	})

	AfterEach(func() {
		// Restore original values
		handlers.StateBaseDir = originalStateDir
		handlers.GitPushRepo = originalGitPushRepo
		handlers.GitAbandonRepo = originalGitAbandonRepo
		handlers.GitDiffRepo = originalGitDiffRepo
		handlers.GitCheckMergeStatus = originalGitCheckMergeStatus
		handlers.GitPullRepo = originalGitPullRepo
		handlers.GitPushToRepo = originalGitPushToRepo
		handlers.GitCreateBranch = originalGitCreateBranch
		handlers.GitListRemoteBranches = originalGitListRemoteBranches

		// Clean up temp directory
		if tempStateDir != "" {
			os.RemoveAll(tempStateDir)
		}
	})

	Context("Git Push Operations", func() {
		Describe("ContentGitPush", func() {
			It("Should push successfully with valid parameters", func() {
				// Mock successful git push
				handlers.GitPushRepo = func(ctx context.Context, repoDir, commitMessage, outputRepoURL, branch, githubToken string) (string, error) {
					Expect(repoDir).To(Equal(filepath.Join(tempStateDir, "test-repo")))
					Expect(commitMessage).To(Equal("Test commit"))
					Expect(outputRepoURL).To(Equal("https://github.com/test/repo.git"))
					Expect(branch).To(Equal("main"))
					return "Push successful", nil
				}

				requestBody := map[string]interface{}{
					"repoPath":      "test-repo",
					"commitMessage": "Test commit",
					"outputRepoUrl": "https://github.com/test/repo.git",
					"branch":        "main",
				}

				context := httpUtils.CreateTestGinContext("POST", "/content/github/push", requestBody)
				context.Request.Header.Set("X-GitHub-Token", "test-token")

				handlers.ContentGitPush(context)

				httpUtils.AssertHTTPStatus(http.StatusOK)
				httpUtils.AssertJSONContains(map[string]interface{}{
					"ok":     true,
					"stdout": "Push successful",
				})
			})

			It("Should return error when outputRepoUrl is missing", func() {
				requestBody := map[string]interface{}{
					"repoPath":      "test-repo",
					"commitMessage": "Test commit",
					"branch":        "main",
				}

				context := httpUtils.CreateTestGinContext("POST", "/content/github/push", requestBody)
				context.Request.Header.Set("X-GitHub-Token", "test-token")

				handlers.ContentGitPush(context)

				httpUtils.AssertHTTPStatus(http.StatusBadRequest)
				httpUtils.AssertErrorMessage("missing outputRepoUrl")
			})

			It("Should return error when branch is missing", func() {
				requestBody := map[string]interface{}{
					"repoPath":      "test-repo",
					"commitMessage": "Test commit",
					"outputRepoUrl": "https://github.com/test/repo.git",
				}

				context := httpUtils.CreateTestGinContext("POST", "/content/github/push", requestBody)
				context.Request.Header.Set("X-GitHub-Token", "test-token")

				handlers.ContentGitPush(context)

				httpUtils.AssertHTTPStatus(http.StatusBadRequest)
				httpUtils.AssertErrorMessage("missing branch")
			})

			It("Should reject invalid repo paths", func() {
				requestBody := map[string]interface{}{
					"repoPath":      "../../../etc/passwd",
					"commitMessage": "Test commit",
					"outputRepoUrl": "https://github.com/test/repo.git",
					"branch":        "main",
				}

				context := httpUtils.CreateTestGinContext("POST", "/content/github/push", requestBody)
				context.Request.Header.Set("X-GitHub-Token", "test-token")

				handlers.ContentGitPush(context)

				httpUtils.AssertHTTPStatus(http.StatusBadRequest)
				httpUtils.AssertErrorMessage("invalid repoPath")
			})

			It("Should handle no changes scenario", func() {
				// Mock git push that returns empty output (no changes)
				handlers.GitPushRepo = func(ctx context.Context, repoDir, commitMessage, outputRepoURL, branch, githubToken string) (string, error) {
					return "", nil
				}

				requestBody := map[string]interface{}{
					"repoPath":      "test-repo",
					"commitMessage": "Test commit",
					"outputRepoUrl": "https://github.com/test/repo.git",
					"branch":        "main",
				}

				context := httpUtils.CreateTestGinContext("POST", "/content/github/push", requestBody)

				handlers.ContentGitPush(context)

				httpUtils.AssertHTTPStatus(http.StatusOK)
				httpUtils.AssertJSONContains(map[string]interface{}{
					"ok":     true,
					"stdout": "",
				})
			})
		})

		Describe("ContentGitAbandon", func() {
			It("Should abandon repository successfully", func() {
				handlers.GitAbandonRepo = func(ctx context.Context, repoDir string) error {
					Expect(repoDir).To(Equal(filepath.Join(tempStateDir, "test-repo")))
					return nil
				}

				requestBody := map[string]interface{}{
					"repoPath": "test-repo",
				}

				context := httpUtils.CreateTestGinContext("POST", "/content/github/abandon", requestBody)

				handlers.ContentGitAbandon(context)

				httpUtils.AssertHTTPStatus(http.StatusOK)
				httpUtils.AssertJSONContains(map[string]interface{}{
					"ok": true,
				})
			})

			It("Should reject invalid repo paths", func() {
				requestBody := map[string]interface{}{
					"repoPath": "../../../etc",
				}

				context := httpUtils.CreateTestGinContext("POST", "/content/github/abandon", requestBody)
				context.Request.Header.Set("X-GitHub-Token", "test-token")

				handlers.ContentGitAbandon(context)

				httpUtils.AssertHTTPStatus(http.StatusBadRequest)
				httpUtils.AssertErrorMessage("invalid repoPath")
			})
		})
	})

	Context("Git Diff Operations", func() {
		Describe("ContentGitDiff", func() {
			It("Should return diff summary successfully", func() {
				handlers.GitDiffRepo = func(ctx context.Context, repoDir string) (*git.DiffSummary, error) {
					return &git.DiffSummary{
						FilesAdded:   3,
						FilesRemoved: 1,
						TotalAdded:   150,
						TotalRemoved: 45,
					}, nil
				}

				context := httpUtils.CreateTestGinContext("GET", "/content/github/diff?repoPath=test-repo", nil)

				handlers.ContentGitDiff(context)

				httpUtils.AssertHTTPStatus(http.StatusOK)
				httpUtils.AssertJSONContains(map[string]interface{}{
					"files": map[string]interface{}{
						"added":   float64(3),
						"removed": float64(1),
					},
					"total_added":   float64(150),
					"total_removed": float64(45),
				})
			})

			It("Should return empty diff when git operation fails", func() {
				handlers.GitDiffRepo = func(ctx context.Context, repoDir string) (*git.DiffSummary, error) {
					return nil, os.ErrNotExist
				}

				context := httpUtils.CreateTestGinContext("GET", "/content/github/diff?repoPath=test-repo", nil)

				handlers.ContentGitDiff(context)

				httpUtils.AssertHTTPStatus(http.StatusOK)
				httpUtils.AssertJSONContains(map[string]interface{}{
					"files": map[string]interface{}{
						"added":   float64(0),
						"removed": float64(0),
					},
					"total_added":   float64(0),
					"total_removed": float64(0),
				})
			})

			It("Should require repoPath parameter", func() {
				context := httpUtils.CreateTestGinContext("GET", "/content/github/diff", nil)
				context.Request.Header.Set("X-GitHub-Token", "test-token")

				handlers.ContentGitDiff(context)

				httpUtils.AssertHTTPStatus(http.StatusBadRequest)
				httpUtils.AssertErrorMessage("missing repoPath")
			})
		})

		Describe("ContentGitStatus", func() {
			It("Should return not initialized for non-existent directory", func() {
				context := httpUtils.CreateTestGinContext("GET", "/content/git-status?path=nonexistent", nil)

				handlers.ContentGitStatus(context)

				httpUtils.AssertHTTPStatus(http.StatusOK)
				httpUtils.AssertJSONContains(map[string]interface{}{
					"initialized": false,
					"hasChanges":  false,
				})
			})

			It("Should return git status for initialized repository", func() {
				// Create test directory and .git subdirectory
				testDir := filepath.Join(tempStateDir, "test-repo")
				gitDir := filepath.Join(testDir, ".git")
				err := os.MkdirAll(gitDir, 0755)
				Expect(err).NotTo(HaveOccurred())

				handlers.GitDiffRepo = func(ctx context.Context, repoDir string) (*git.DiffSummary, error) {
					return &git.DiffSummary{
						FilesAdded:   2,
						FilesRemoved: 1,
						TotalAdded:   50,
						TotalRemoved: 25,
					}, nil
				}

				context := httpUtils.CreateTestGinContext("GET", "/content/git-status?path=test-repo", nil)

				handlers.ContentGitStatus(context)

				httpUtils.AssertHTTPStatus(http.StatusOK)
				httpUtils.AssertJSONContains(map[string]interface{}{
					"initialized":      true,
					"hasChanges":       true,
					"filesAdded":       float64(2),
					"filesRemoved":     float64(1),
					"uncommittedFiles": float64(3),
					"totalAdded":       float64(50),
					"totalRemoved":     float64(25),
				})
			})

			It("Should handle paths with .. components safely", func() {
				context := httpUtils.CreateTestGinContext("GET", "/content/git-status?path=../../../etc", nil)
				context.Request.Header.Set("X-GitHub-Token", "test-token")

				handlers.ContentGitStatus(context)

				// Handler cleans path safely and returns status for non-git directory
				httpUtils.AssertHTTPStatus(http.StatusOK)
				httpUtils.AssertJSONContains(map[string]interface{}{
					"hasChanges":  false,
					"initialized": false,
				})
			})
		})
	})

	Context("File Operations", func() {
		Describe("ContentWrite", func() {
			It("Should write text content successfully", func() {
				requestBody := map[string]interface{}{
					"path":     "test/file.txt",
					"content":  "Hello World",
					"encoding": "utf8",
				}

				context := httpUtils.CreateTestGinContext("POST", "/content/write", requestBody)

				handlers.ContentWrite(context)

				httpUtils.AssertHTTPStatus(http.StatusOK)
				httpUtils.AssertJSONContains(map[string]interface{}{
					"message": "ok",
				})

				// Verify file was written
				filePath := filepath.Join(tempStateDir, "test", "file.txt")
				content, err := os.ReadFile(filePath)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(content)).To(Equal("Hello World"))
			})

			It("Should write base64 content successfully", func() {
				// "Hello World" in base64
				base64Content := "SGVsbG8gV29ybGQ="

				requestBody := map[string]interface{}{
					"path":     "test/binary.dat",
					"content":  base64Content,
					"encoding": "base64",
				}

				context := httpUtils.CreateTestGinContext("POST", "/content/write", requestBody)

				handlers.ContentWrite(context)

				httpUtils.AssertHTTPStatus(http.StatusOK)

				// Verify file was written correctly
				filePath := filepath.Join(tempStateDir, "test", "binary.dat")
				content, err := os.ReadFile(filePath)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(content)).To(Equal("Hello World"))
			})

			It("Should reject invalid base64 content", func() {
				requestBody := map[string]interface{}{
					"path":     "test/invalid.dat",
					"content":  "invalid-base64-content!@#",
					"encoding": "base64",
				}

				context := httpUtils.CreateTestGinContext("POST", "/content/write", requestBody)
				context.Request.Header.Set("X-GitHub-Token", "test-token")

				handlers.ContentWrite(context)

				httpUtils.AssertHTTPStatus(http.StatusBadRequest)
				httpUtils.AssertErrorMessage("invalid base64 content")
			})

			It("Should handle paths with .. components safely", func() {
				requestBody := map[string]interface{}{
					"path":    "../../../etc/passwd",
					"content": "test content",
				}

				context := httpUtils.CreateTestGinContext("POST", "/content/write", requestBody)
				context.Request.Header.Set("X-GitHub-Token", "test-token")

				handlers.ContentWrite(context)

				// Handler cleans path and writes safely within base directory
				httpUtils.AssertHTTPStatus(http.StatusOK)
				httpUtils.AssertJSONContains(map[string]interface{}{
					"message": "ok",
				})
			})
		})

		Describe("ContentRead", func() {
			It("Should read file content successfully", func() {
				// Create test file
				testDir := filepath.Join(tempStateDir, "test")
				err := os.MkdirAll(testDir, 0755)
				Expect(err).NotTo(HaveOccurred())

				filePath := filepath.Join(testDir, "file.txt")
				err = os.WriteFile(filePath, []byte("Test content"), 0644)
				Expect(err).NotTo(HaveOccurred())

				context := httpUtils.CreateTestGinContext("GET", "/content/file?path=test/file.txt", nil)

				handlers.ContentRead(context)

				httpUtils.AssertHTTPStatus(http.StatusOK)
				body := httpUtils.GetResponseBody()
				Expect(string(body)).To(Equal("Test content"))
			})

			It("Should return 404 for non-existent file", func() {
				context := httpUtils.CreateTestGinContext("GET", "/content/file?path=nonexistent.txt", nil)

				handlers.ContentRead(context)

				httpUtils.AssertHTTPStatus(http.StatusNotFound)
				httpUtils.AssertErrorMessage("not found")
			})

			It("Should handle paths with .. components safely", func() {
				context := httpUtils.CreateTestGinContext("GET", "/content/file?path=../../../etc/passwd", nil)
				context.Request.Header.Set("X-GitHub-Token", "test-token")

				handlers.ContentRead(context)

				// Handler cleans path safely but file doesn't exist
				httpUtils.AssertHTTPStatus(http.StatusNotFound)
				httpUtils.AssertErrorMessage("not found")
			})
		})

		Describe("ContentList", func() {
			It("Should list directory contents successfully", func() {
				// Create test directory structure
				testDir := filepath.Join(tempStateDir, "test")
				err := os.MkdirAll(testDir, 0755)
				Expect(err).NotTo(HaveOccurred())

				// Create files
				err = os.WriteFile(filepath.Join(testDir, "file1.txt"), []byte("content1"), 0644)
				Expect(err).NotTo(HaveOccurred())

				err = os.WriteFile(filepath.Join(testDir, "file2.txt"), []byte("content2"), 0644)
				Expect(err).NotTo(HaveOccurred())

				// Create subdirectory
				err = os.MkdirAll(filepath.Join(testDir, "subdir"), 0755)
				Expect(err).NotTo(HaveOccurred())

				context := httpUtils.CreateTestGinContext("GET", "/content/list?path=test", nil)

				handlers.ContentList(context)

				httpUtils.AssertHTTPStatus(http.StatusOK)

				var response map[string]interface{}
				httpUtils.GetResponseJSON(&response)

				Expect(response).To(HaveKey("items"))
				itemsInterface, exists := response["items"]
				Expect(exists).To(BeTrue(), "Response should contain 'items' field")
				items, ok := itemsInterface.([]interface{})
				Expect(ok).To(BeTrue(), "Items should be an array")
				Expect(len(items)).To(Equal(3))

				// Check that files and directories are properly identified
				itemNames := make([]string, len(items))
				for i, item := range items {
					itemMap, ok := item.(map[string]interface{})
					Expect(ok).To(BeTrue(), "Item should be a map")
					nameInterface, exists := itemMap["name"]
					Expect(exists).To(BeTrue(), "Item should contain 'name' field")
					name, ok := nameInterface.(string)
					Expect(ok).To(BeTrue(), "Item name should be a string")
					itemNames[i] = name
				}
				Expect(itemNames).To(ContainElements("file1.txt", "file2.txt", "subdir"))
			})

			It("Should handle single file metadata", func() {
				// Create test file
				testDir := filepath.Join(tempStateDir, "test")
				err := os.MkdirAll(testDir, 0755)
				Expect(err).NotTo(HaveOccurred())

				filePath := filepath.Join(testDir, "single.txt")
				err = os.WriteFile(filePath, []byte("test content"), 0644)
				Expect(err).NotTo(HaveOccurred())

				context := httpUtils.CreateTestGinContext("GET", "/content/list?path=test/single.txt", nil)

				handlers.ContentList(context)

				httpUtils.AssertHTTPStatus(http.StatusOK)

				var response map[string]interface{}
				httpUtils.GetResponseJSON(&response)

				Expect(response).To(HaveKey("items"))
				itemsInterface, exists := response["items"]
				Expect(exists).To(BeTrue(), "Response should contain 'items' field")
				items, ok := itemsInterface.([]interface{})
				Expect(ok).To(BeTrue(), "Items should be an array")
				Expect(len(items)).To(Equal(1))

				itemInterface := items[0]
				item, ok := itemInterface.(map[string]interface{})
				Expect(ok).To(BeTrue(), "Item should be a map")

				nameInterface, exists := item["name"]
				Expect(exists).To(BeTrue(), "Item should contain 'name' field")
				Expect(nameInterface).To(Equal("single.txt"))

				isDirInterface, exists := item["isDir"]
				Expect(exists).To(BeTrue(), "Item should contain 'isDir' field")
				Expect(isDirInterface).To(BeFalse())

				sizeInterface, exists := item["size"]
				Expect(exists).To(BeTrue(), "Item should contain 'size' field")
				Expect(sizeInterface).To(BeNumerically("==", 12))
			})

			It("Should return 404 for non-existent path", func() {
				context := httpUtils.CreateTestGinContext("GET", "/content/list?path=nonexistent", nil)

				handlers.ContentList(context)

				httpUtils.AssertHTTPStatus(http.StatusNotFound)
				httpUtils.AssertErrorMessage("not found")
			})

			It("Should handle paths with .. components safely", func() {
				context := httpUtils.CreateTestGinContext("GET", "/content/list?path=../../../etc", nil)
				context.Request.Header.Set("X-GitHub-Token", "test-token")

				handlers.ContentList(context)

				// Handler cleans path safely but directory doesn't exist
				httpUtils.AssertHTTPStatus(http.StatusNotFound)
				httpUtils.AssertErrorMessage("not found")
			})
		})
	})

	Context("Git Branch Operations", func() {
		Describe("ContentGitCreateBranch", func() {
			It("Should create branch successfully", func() {
				handlers.GitCreateBranch = func(ctx context.Context, repoDir, branchName string) error {
					Expect(repoDir).To(Equal(filepath.Join(tempStateDir, "test-repo")))
					Expect(branchName).To(Equal("feature-branch"))
					return nil
				}

				requestBody := map[string]interface{}{
					"path":       "test-repo",
					"branchName": "feature-branch",
				}

				context := httpUtils.CreateTestGinContext("POST", "/content/git-create-branch", requestBody)

				handlers.ContentGitCreateBranch(context)

				httpUtils.AssertHTTPStatus(http.StatusOK)
				httpUtils.AssertJSONContains(map[string]interface{}{
					"message":    "branch created",
					"branchName": "feature-branch",
				})
			})

			It("Should require branchName parameter", func() {
				requestBody := map[string]interface{}{
					"path": "test-repo",
				}

				context := httpUtils.CreateTestGinContext("POST", "/content/git-create-branch", requestBody)
				context.Request.Header.Set("X-GitHub-Token", "test-token")

				handlers.ContentGitCreateBranch(context)

				httpUtils.AssertHTTPStatus(http.StatusBadRequest)
				httpUtils.AssertErrorMessage("branchName is required")
			})
		})

		Describe("ContentGitListBranches", func() {
			It("Should list remote branches successfully", func() {
				handlers.GitListRemoteBranches = func(ctx context.Context, repoDir string) ([]string, error) {
					return []string{"main", "develop", "feature-1"}, nil
				}

				context := httpUtils.CreateTestGinContext("GET", "/content/git-list-branches?path=test-repo", nil)

				handlers.ContentGitListBranches(context)

				httpUtils.AssertHTTPStatus(http.StatusOK)
				httpUtils.AssertJSONContains(map[string]interface{}{
					"branches": []interface{}{"main", "develop", "feature-1"},
				})
			})

			It("Should handle git errors gracefully", func() {
				handlers.GitListRemoteBranches = func(ctx context.Context, repoDir string) ([]string, error) {
					return nil, os.ErrNotExist
				}

				context := httpUtils.CreateTestGinContext("GET", "/content/git-list-branches?path=test-repo", nil)

				handlers.ContentGitListBranches(context)

				httpUtils.AssertHTTPStatus(http.StatusInternalServerError)
			})
		})
	})

	Context("Git Synchronization Operations", func() {
		Describe("ContentGitPull", func() {
			It("Should pull changes successfully", func() {
				handlers.GitPullRepo = func(ctx context.Context, repoDir, branch string) error {
					Expect(repoDir).To(Equal(filepath.Join(tempStateDir, "test-repo")))
					Expect(branch).To(Equal("main"))
					return nil
				}

				requestBody := map[string]interface{}{
					"path":   "test-repo",
					"branch": "main",
				}

				context := httpUtils.CreateTestGinContext("POST", "/content/git-pull", requestBody)

				handlers.ContentGitPull(context)

				httpUtils.AssertHTTPStatus(http.StatusOK)
				httpUtils.AssertJSONContains(map[string]interface{}{
					"message": "pulled successfully",
					"branch":  "main",
				})
			})

			It("Should default to main branch when not specified", func() {
				handlers.GitPullRepo = func(ctx context.Context, repoDir, branch string) error {
					Expect(branch).To(Equal("main"))
					return nil
				}

				requestBody := map[string]interface{}{
					"path": "test-repo",
				}

				context := httpUtils.CreateTestGinContext("POST", "/content/git-pull", requestBody)

				handlers.ContentGitPull(context)

				httpUtils.AssertHTTPStatus(http.StatusOK)
			})
		})

		Describe("ContentGitPushToBranch", func() {
			It("Should push to branch successfully", func() {
				handlers.GitPushToRepo = func(ctx context.Context, repoDir, branch, commitMessage string) error {
					Expect(repoDir).To(Equal(filepath.Join(tempStateDir, "test-repo")))
					Expect(branch).To(Equal("feature"))
					Expect(commitMessage).To(Equal("Custom commit"))
					return nil
				}

				requestBody := map[string]interface{}{
					"path":    "test-repo",
					"branch":  "feature",
					"message": "Custom commit",
				}

				context := httpUtils.CreateTestGinContext("POST", "/content/git-push", requestBody)

				handlers.ContentGitPushToBranch(context)

				httpUtils.AssertHTTPStatus(http.StatusOK)
				httpUtils.AssertJSONContains(map[string]interface{}{
					"message": "pushed successfully",
					"branch":  "feature",
				})
			})

			It("Should use default values when not specified", func() {
				handlers.GitPushToRepo = func(ctx context.Context, repoDir, branch, commitMessage string) error {
					Expect(branch).To(Equal("main"))
					Expect(commitMessage).To(Equal("Session artifacts update"))
					return nil
				}

				requestBody := map[string]interface{}{
					"path": "test-repo",
				}

				context := httpUtils.CreateTestGinContext("POST", "/content/git-push", requestBody)

				handlers.ContentGitPushToBranch(context)

				httpUtils.AssertHTTPStatus(http.StatusOK)
			})
		})

		Describe("ContentGitMergeStatus", func() {
			It("Should return merge status for git repository", func() {
				// Create test git directory
				testDir := filepath.Join(tempStateDir, "test-repo")
				gitDir := filepath.Join(testDir, ".git")
				err := os.MkdirAll(gitDir, 0755)
				Expect(err).NotTo(HaveOccurred())

				handlers.GitCheckMergeStatus = func(ctx context.Context, repoDir, branch string) (*git.MergeStatus, error) {
					return &git.MergeStatus{
						CanMergeClean:      true,
						LocalChanges:       0,
						RemoteCommitsAhead: 2,
						ConflictingFiles:   []string{},
						RemoteBranchExists: true,
					}, nil
				}

				context := httpUtils.CreateTestGinContext("GET", "/content/git-merge-status?path=test-repo&branch=main", nil)

				handlers.ContentGitMergeStatus(context)

				httpUtils.AssertHTTPStatus(http.StatusOK)
				httpUtils.AssertJSONContains(map[string]interface{}{
					"canMergeClean":      true,
					"localChanges":       float64(0),
					"remoteCommitsAhead": float64(2),
					"conflictingFiles":   []interface{}{},
					"remoteBranchExists": true,
				})
			})

			It("Should return default status for non-git directory", func() {
				context := httpUtils.CreateTestGinContext("GET", "/content/git-merge-status?path=nonexistent", nil)

				handlers.ContentGitMergeStatus(context)

				httpUtils.AssertHTTPStatus(http.StatusOK)
				httpUtils.AssertJSONContains(map[string]interface{}{
					"canMergeClean":      true,
					"localChanges":       float64(0),
					"remoteCommitsAhead": float64(0),
					"conflictingFiles":   []interface{}{},
					"remoteBranchExists": false,
				})
			})

			It("Should default to main branch when not specified", func() {
				// Create test git directory
				testDir := filepath.Join(tempStateDir, "test-repo")
				gitDir := filepath.Join(testDir, ".git")
				err := os.MkdirAll(gitDir, 0755)
				Expect(err).NotTo(HaveOccurred())

				handlers.GitCheckMergeStatus = func(ctx context.Context, repoDir, branch string) (*git.MergeStatus, error) {
					Expect(branch).To(Equal("main"))
					return &git.MergeStatus{}, nil
				}

				context := httpUtils.CreateTestGinContext("GET", "/content/git-merge-status?path=test-repo", nil)

				handlers.ContentGitMergeStatus(context)

				httpUtils.AssertHTTPStatus(http.StatusOK)
			})
		})
	})

	Context("Workflow Metadata Operations", func() {
		Describe("ContentWorkflowMetadata", func() {
			It("Should return empty metadata when no workflow found", func() {
				context := httpUtils.CreateTestGinContext("GET", "/content/workflow-metadata?session=nonexistent", nil)

				handlers.ContentWorkflowMetadata(context)

				httpUtils.AssertHTTPStatus(http.StatusOK)
				httpUtils.AssertJSONContains(map[string]interface{}{
					"commands": []interface{}{},
					"agents":   []interface{}{},
					"config": map[string]interface{}{
						"artifactsDir": "artifacts",
					},
				})
			})

			It("Should require session parameter", func() {
				context := httpUtils.CreateTestGinContext("GET", "/content/workflow-metadata", nil)
				context.Request.Header.Set("X-GitHub-Token", "test-token")

				handlers.ContentWorkflowMetadata(context)

				httpUtils.AssertHTTPStatus(http.StatusBadRequest)
				httpUtils.AssertErrorMessage("missing session parameter")
			})

			It("Should parse workflow metadata when available", func() {
				// Create test workflow structure
				sessionDir := filepath.Join(tempStateDir, "sessions", "test-session", "workspace", "workflows", "test-workflow")
				claudeDir := filepath.Join(sessionDir, ".claude")
				commandsDir := filepath.Join(claudeDir, "commands")
				agentsDir := filepath.Join(claudeDir, "agents")
				ambientDir := filepath.Join(sessionDir, ".ambient")

				err := os.MkdirAll(commandsDir, 0755)
				Expect(err).NotTo(HaveOccurred())

				err = os.MkdirAll(agentsDir, 0755)
				Expect(err).NotTo(HaveOccurred())

				err = os.MkdirAll(ambientDir, 0755)
				Expect(err).NotTo(HaveOccurred())

				// Create test command file
				commandContent := `---
displayName: "Test Command"
description: "A test command"
icon: "⚡"
---
# Test Command

This is a test command.
`
				err = os.WriteFile(filepath.Join(commandsDir, "test.command.md"), []byte(commandContent), 0644)
				Expect(err).NotTo(HaveOccurred())

				// Create test agent file
				agentContent := `---
name: "Test Agent"
description: "A test agent"
tools: "bash,python"
---
# Test Agent

This is a test agent.
`
				err = os.WriteFile(filepath.Join(agentsDir, "test-agent.md"), []byte(agentContent), 0644)
				Expect(err).NotTo(HaveOccurred())

				// Create ambient.json config
				configContent := `{
  "name": "Test Workflow",
  "description": "A test workflow",
  "systemPrompt": "You are a test agent",
  "artifactsDir": "outputs"
}`
				err = os.WriteFile(filepath.Join(ambientDir, "ambient.json"), []byte(configContent), 0644)
				Expect(err).NotTo(HaveOccurred())

				context := httpUtils.CreateTestGinContext("GET", "/content/workflow-metadata?session=test-session", nil)

				handlers.ContentWorkflowMetadata(context)

				httpUtils.AssertHTTPStatus(http.StatusOK)

				var response map[string]interface{}
				httpUtils.GetResponseJSON(&response)

				// Check commands
				Expect(response).To(HaveKey("commands"))
				commandsInterface, exists := response["commands"]
				Expect(exists).To(BeTrue(), "Response should contain 'commands' field")
				commands, ok := commandsInterface.([]interface{})
				Expect(ok).To(BeTrue(), "Commands should be an array")
				Expect(len(commands)).To(Equal(1))

				commandInterface := commands[0]
				command, ok := commandInterface.(map[string]interface{})
				Expect(ok).To(BeTrue(), "Command should be a map")

				idInterface, exists := command["id"]
				Expect(exists).To(BeTrue(), "Command should contain 'id' field")
				Expect(idInterface).To(Equal("test.command"))

				nameInterface, exists := command["name"]
				Expect(exists).To(BeTrue(), "Command should contain 'name' field")
				Expect(nameInterface).To(Equal("Test Command"))

				descriptionInterface, exists := command["description"]
				Expect(exists).To(BeTrue(), "Command should contain 'description' field")
				Expect(descriptionInterface).To(Equal("A test command"))

				slashCommandInterface, exists := command["slashCommand"]
				Expect(exists).To(BeTrue(), "Command should contain 'slashCommand' field")
				Expect(slashCommandInterface).To(Equal("/command"))

				iconInterface, exists := command["icon"]
				Expect(exists).To(BeTrue(), "Command should contain 'icon' field")
				Expect(iconInterface).To(Equal("⚡"))

				// Check agents
				Expect(response).To(HaveKey("agents"))
				agentsInterface, exists := response["agents"]
				Expect(exists).To(BeTrue(), "Response should contain 'agents' field")
				agents, ok := agentsInterface.([]interface{})
				Expect(ok).To(BeTrue(), "Agents should be an array")
				Expect(len(agents)).To(Equal(1))

				agentInterface := agents[0]
				agent, ok := agentInterface.(map[string]interface{})
				Expect(ok).To(BeTrue(), "Agent should be a map")

				idInterface, exists = agent["id"]
				Expect(exists).To(BeTrue(), "Agent should contain 'id' field")
				Expect(idInterface).To(Equal("test-agent"))

				nameInterface, exists = agent["name"]
				Expect(exists).To(BeTrue(), "Agent should contain 'name' field")
				Expect(nameInterface).To(Equal("Test Agent"))

				descriptionInterface, exists = agent["description"]
				Expect(exists).To(BeTrue(), "Agent should contain 'description' field")
				Expect(descriptionInterface).To(Equal("A test agent"))

				toolsInterface, exists := agent["tools"]
				Expect(exists).To(BeTrue(), "Agent should contain 'tools' field")
				Expect(toolsInterface).To(Equal("bash,python"))

				// Check config
				Expect(response).To(HaveKey("config"))
				configInterface, exists := response["config"]
				Expect(exists).To(BeTrue(), "Response should contain 'config' field")
				config, ok := configInterface.(map[string]interface{})
				Expect(ok).To(BeTrue(), "Config should be a map")

				nameInterface, exists = config["name"]
				Expect(exists).To(BeTrue(), "Config should contain 'name' field")
				Expect(nameInterface).To(Equal("Test Workflow"))

				descriptionInterface, exists = config["description"]
				Expect(exists).To(BeTrue(), "Config should contain 'description' field")
				Expect(descriptionInterface).To(Equal("A test workflow"))

				systemPromptInterface, exists := config["systemPrompt"]
				Expect(exists).To(BeTrue(), "Config should contain 'systemPrompt' field")
				Expect(systemPromptInterface).To(Equal("You are a test agent"))

				artifactsDirInterface, exists := config["artifactsDir"]
				Expect(exists).To(BeTrue(), "Config should contain 'artifactsDir' field")
				Expect(artifactsDirInterface).To(Equal("outputs"))
			})
		})
	})

	Context("Input Validation", func() {
		It("Should handle paths with '..' components safely", func() {
			pathTestCases := []struct {
				path        string
				description string
				writeStatus int
				readStatus  int
				listStatus  int
			}{
				{"../../../etc/passwd", "path traversal attempt", http.StatusOK, http.StatusNotFound, http.StatusNotFound},
				{"test/../../../etc/passwd", "nested path traversal", http.StatusOK, http.StatusNotFound, http.StatusNotFound},
				{"test/../../..", "relative parent dirs", http.StatusBadRequest, http.StatusBadRequest, http.StatusBadRequest},
				{"../", "parent directory", http.StatusBadRequest, http.StatusBadRequest, http.StatusBadRequest},
				{"..\\..\\..\\etc", "windows-style traversal", http.StatusBadRequest, http.StatusBadRequest, http.StatusBadRequest},
			}

			for _, tc := range pathTestCases {
				// Test ContentWrite - should succeed by cleaning path
				requestBody := map[string]interface{}{
					"path":    tc.path,
					"content": "test",
				}
				context := httpUtils.CreateTestGinContext("POST", "/content/write", requestBody)
				context.Request.Header.Set("X-GitHub-Token", "test-token")
				handlers.ContentWrite(context)
				Expect(httpUtils.GetResponseRecorder().Code).To(Equal(tc.writeStatus), "ContentWrite for path: "+tc.path+" ("+tc.description+")")

				// Clean up temp directory to ensure isolation
				os.RemoveAll(tempStateDir)
				var err error
				tempStateDir, err = os.MkdirTemp("", "content-test-*")
				Expect(err).NotTo(HaveOccurred())
				handlers.StateBaseDir = tempStateDir

				// Reset recorder for read test
				httpUtils = test_utils.NewHTTPTestUtils()

				// Test ContentRead with clean environment - should return 404 since file doesn't exist in new temp dir
				context = httpUtils.CreateTestGinContext("GET", "/content/file?path="+tc.path, nil)
				context.Request.Header.Set("X-GitHub-Token", "test-token")
				handlers.ContentRead(context)
				Expect(httpUtils.GetResponseRecorder().Code).To(Equal(tc.readStatus), "ContentRead for path: "+tc.path+" ("+tc.description+")")

				// Reset recorder for list test
				httpUtils = test_utils.NewHTTPTestUtils()

				// Test ContentList with clean environment - should return 404 since directory doesn't exist in new temp dir
				context = httpUtils.CreateTestGinContext("GET", "/content/list?path="+tc.path, nil)
				context.Request.Header.Set("X-GitHub-Token", "test-token")
				handlers.ContentList(context)
				Expect(httpUtils.GetResponseRecorder().Code).To(Equal(tc.listStatus), "ContentList for path: "+tc.path+" ("+tc.description+")")

				// Reset recorder for next iteration
				httpUtils = test_utils.NewHTTPTestUtils()
			}
		})

		It("Should handle root path consistently", func() {
			// Test ContentWrite with root path
			requestBody := map[string]interface{}{
				"path":    "/",
				"content": "root content",
			}
			context := httpUtils.CreateTestGinContext("POST", "/content/write", requestBody)
			context.Request.Header.Set("X-GitHub-Token", "test-token")
			handlers.ContentWrite(context)
			httpUtils.AssertHTTPStatus(http.StatusBadRequest)

			// Reset for next test
			httpUtils = test_utils.NewHTTPTestUtils()

			// Test ContentRead with root path
			context = httpUtils.CreateTestGinContext("GET", "/content/file?path=/", nil)
			context.Request.Header.Set("X-GitHub-Token", "test-token")
			handlers.ContentRead(context)
			httpUtils.AssertHTTPStatus(http.StatusBadRequest)

			// Reset for next test
			httpUtils = test_utils.NewHTTPTestUtils()

			// Test ContentList with root path
			context = httpUtils.CreateTestGinContext("GET", "/content/list?path=/", nil)
			context.Request.Header.Set("X-GitHub-Token", "test-token")
			handlers.ContentList(context)
			httpUtils.AssertHTTPStatus(http.StatusBadRequest)
		})
	})

	Context("Error Handling", func() {
		It("Should handle JSON binding errors gracefully", func() {
			context := httpUtils.CreateTestGinContext("POST", "/content/write", "invalid-json")
			context.Request.Header.Set("X-GitHub-Token", "test-token")

			handlers.ContentWrite(context)

			httpUtils.AssertHTTPStatus(http.StatusBadRequest)
		})

		It("Should handle filesystem errors gracefully", func() {
			// Try to write to a location where directory creation will fail
			// by creating a file first, then trying to create a directory with the same name
			blockingFile := filepath.Join(tempStateDir, "blocking-file")
			err := os.WriteFile(blockingFile, []byte("blocker"), 0644)
			Expect(err).NotTo(HaveOccurred())

			requestBody := map[string]interface{}{
				"path":    "blocking-file/subfolder/file.txt",
				"content": "content",
			}

			context := httpUtils.CreateTestGinContext("POST", "/content/write", requestBody)

			handlers.ContentWrite(context)

			httpUtils.AssertHTTPStatus(http.StatusInternalServerError)
			httpUtils.AssertErrorMessage("failed to create directory")
		})
	})
})
