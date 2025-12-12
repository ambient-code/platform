//go:build test

package handlers

import (
	"context"
	"fmt"

	"ambient-code-backend/git"
	"ambient-code-backend/tests/config"
	test_constants "ambient-code-backend/tests/constants"
	"ambient-code-backend/tests/logger"
	"ambient-code-backend/tests/test_utils"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes"
)

var _ = Describe("Git Operations", Label(test_constants.LabelUnit, test_constants.LabelGit, test_constants.LabelOperations), func() {
	var (
		ctx      context.Context
		k8sUtils *test_utils.K8sTestUtils
	)

	BeforeEach(func() {
		logger.Log("Setting up Git Operations test")
		ctx = context.Background()
		k8sUtils = test_utils.NewK8sTestUtils(false, *config.TestNamespace)
	})

	Describe("Token Management", func() {
		Context("When storing and retrieving GitHub tokens", func() {

			It("Should handle missing tokens gracefully", func() {
				// Note: GetGitHubToken exists but requires valid k8s setup - testing functionality indirectly
				projectName := "test-project"
				userID := "non-existent-user"

				// Act
				k8sClient := k8sUtils.K8sClient
				clientset, _ := k8sClient.(*kubernetes.Clientset)
				token, err := git.GetGitHubToken(ctx, clientset, k8sUtils.DynamicClient, projectName, userID)

				// Assert - function should return error for missing/invalid setup
				Expect(err).To(HaveOccurred(), "Should return error for missing token/secret")
				Expect(token).To(BeEmpty(), "Should return empty token")

				logger.Log("Handled missing token gracefully")
			})

		})

		Context("When storing and retrieving GitLab tokens", func() {

			It("Should retrieve GitLab token gracefully handle missing tokens", func() {
				// Note: GetGitLabToken exists but requires valid k8s setup - testing functionality indirectly
				namespace := *config.TestNamespace
				userID := "non-existent-user"

				// Act
				token, err := git.GetGitLabToken(ctx, k8sUtils.K8sClient, namespace, userID)

				// Assert - function should return error for missing/invalid setup
				Expect(err).To(HaveOccurred(), "Should return error for missing token/secret")
				Expect(token).To(BeEmpty(), "Should return empty token")

				logger.Log("Handled missing GitLab token gracefully")
			})
		})
	})

	Describe("Branch and URL Operations", func() {
		Context("When constructing branch URLs", func() {
			It("Should construct GitHub branch URLs correctly", func() {
				testCases := []struct {
					repoURL     string
					branch      string
					expectedURL string
				}{
					{
						repoURL:     "https://github.com/user/repo.git",
						branch:      "main",
						expectedURL: "https://github.com/user/repo/tree/main",
					},
					{
						repoURL:     "https://github.com/user/repo.git",
						branch:      "feature/new-feature",
						expectedURL: "https://github.com/user/repo/tree/feature/new-feature",
					},
				}

				for _, tc := range testCases {
					By(fmt.Sprintf("Constructing branch URL for %s/%s", tc.repoURL, tc.branch), func() {
						// Act
						branchURL, err := git.ConstructGitHubBranchURL(tc.repoURL, tc.branch)

						// Assert
						Expect(err).NotTo(HaveOccurred(), "Should construct branch URL without error")
						Expect(branchURL).To(Equal(tc.expectedURL), "Branch URL should match expected")

						logger.Log("Constructed GitHub branch URL: %s", branchURL)
					})
				}
			})

			It("Should construct GitLab branch URLs correctly", func() {
				testCases := []struct {
					repoURL     string
					branch      string
					expectedURL string
				}{
					{
						repoURL:     "https://gitlab.com/user/repo.git",
						branch:      "main",
						expectedURL: "https://gitlab.com/user/repo/-/tree/main",
					},
					{
						repoURL:     "https://gitlab.com/user/repo.git",
						branch:      "feature/new-feature",
						expectedURL: "https://gitlab.com/user/repo/-/tree/feature/new-feature",
					},
				}

				for _, tc := range testCases {
					By(fmt.Sprintf("Constructing GitLab branch URL for %s/%s", tc.repoURL, tc.branch), func() {
						// Act
						branchURL, err := git.ConstructGitLabBranchURL(tc.repoURL, tc.branch)

						// Assert
						Expect(err).NotTo(HaveOccurred(), "Should construct branch URL without error")
						Expect(branchURL).To(Equal(tc.expectedURL), "Branch URL should match expected")

						logger.Log("Constructed GitLab branch URL: %s", branchURL)
					})
				}
			})
		})

		Context("When deriving folder names from URLs", func() {
			It("Should derive consistent folder names", func() {
				testCases := []struct {
					url            string
					expectedFolder string
				}{
					{
						url:            "https://github.com/user/repo.git",
						expectedFolder: "repo",
					},
					{
						url:            "https://gitlab.com/group/subgroup/project.git",
						expectedFolder: "project",
					},
					{
						url:            "git@github.com:user/my-awesome-repo.git",
						expectedFolder: "my-awesome-repo",
					},
				}

				for _, tc := range testCases {
					By(fmt.Sprintf("Deriving folder for URL: %s", tc.url), func() {
						// Act - using actual function name that exists
						folder := git.DeriveRepoFolderFromURL(tc.url)

						// Assert
						Expect(folder).To(Equal(tc.expectedFolder), "Folder name should match expected")

						logger.Log("Derived folder '%s' from URL: %s", folder, tc.url)
					})
				}
			})
		})
	})

	Describe("Error Handling", func() {
		Context("When detecting push errors", func() {
			It("Should detect authentication failures", func() {
				errorMessages := []string{
					"remote: Invalid username or password",
					"fatal: Authentication failed",
					"error: 401 Unauthorized",
				}

				for _, msg := range errorMessages {
					By(fmt.Sprintf("Detecting auth error: %s", msg), func() {
						// Act
						err := git.DetectPushError("https://github.com/user/repo.git", msg, "")

						// Assert
						Expect(err).To(HaveOccurred(), "Should detect authentication error")
						Expect(err.Error()).To(Or(ContainSubstring("authentication failed"), ContainSubstring("Invalid username or password")), "Error should mention authentication")

						logger.Log("Detected authentication error: %v", err)
					})
				}
			})

			It("Should detect permission errors", func() {
				errorMessages := []string{
					"remote: Permission denied",
					"error: 403 Forbidden",
					"remote: You don't have permission to push to this repository",
				}

				for _, msg := range errorMessages {
					By(fmt.Sprintf("Detecting permission error: %s", msg), func() {
						// Act
						err := git.DetectPushError("https://github.com/user/repo.git", msg, "")

						// Assert
						Expect(err).To(HaveOccurred(), "Should detect permission error")
						Expect(err.Error()).To(Or(ContainSubstring("Permission denied"), ContainSubstring("insufficient permissions"), ContainSubstring("You don't have permission to push to this repository")), "Error should mention permission")

						logger.Log("Detected permission error: %v", err)
					})
				}
			})

			It("Should provide helpful error messages", func() {
				// Act
				err := git.DetectPushError("https://github.com/user/repo.git", "fatal: Authentication failed", "")

				// Assert
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("GitHub"), "Should mention the provider")
				Expect(err.Error()).To(ContainSubstring("authentication failed"), "Should mention token in solution")

				logger.Log("Provided helpful error message: %v", err)
			})
		})
	})
})
