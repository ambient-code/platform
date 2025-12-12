//go:build test

package handlers

import (
	test_constants "ambient-code-backend/tests/constants"
	"ambient-code-backend/tests/logger"
	"ambient-code-backend/types"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Common Types", Label(test_constants.LabelUnit, test_constants.LabelTypes, test_constants.LabelCommon), func() {
	Describe("ProviderType", func() {
		Context("When detecting provider from URL", func() {
			It("Should detect GitHub provider correctly", func() {
				testCases := []struct {
					name     string
					url      string
					expected types.ProviderType
				}{
					{
						name:     "GitHub HTTPS URL",
						url:      "https://github.com/user/repo.git",
						expected: types.ProviderGitHub,
					},
					{
						name:     "GitHub HTTPS URL without .git",
						url:      "https://github.com/user/repo",
						expected: types.ProviderGitHub,
					},
					{
						name:     "GitHub SSH URL",
						url:      "git@github.com:user/repo.git",
						expected: types.ProviderGitHub,
					},
					{
						name:     "GitHub Enterprise URL",
						url:      "https://company.github.com/user/repo.git",
						expected: types.ProviderGitHub,
					},
				}

				for _, tc := range testCases {
					By(tc.name, func() {
						// Act
						detected := types.DetectProvider(tc.url)

						// Assert
						Expect(detected).To(Equal(tc.expected),
							"URL %s should be detected as %s", tc.url, tc.expected)

						logger.Log("Detected provider %s for URL: %s", detected, tc.url)
					})
				}
			})

			It("Should detect GitLab provider correctly", func() {
				testCases := []struct {
					name     string
					url      string
					expected types.ProviderType
				}{
					{
						name:     "GitLab HTTPS URL",
						url:      "https://gitlab.com/user/repo.git",
						expected: types.ProviderGitLab,
					},
					{
						name:     "GitLab HTTPS URL without .git",
						url:      "https://gitlab.com/user/repo",
						expected: types.ProviderGitLab,
					},
					{
						name:     "GitLab SSH URL",
						url:      "git@gitlab.com:user/repo.git",
						expected: types.ProviderGitLab,
					},
					{
						name:     "Self-hosted GitLab URL",
						url:      "https://gitlab.company.com/user/repo.git",
						expected: types.ProviderGitLab,
					},
				}

				for _, tc := range testCases {
					By(tc.name, func() {
						// Act
						detected := types.DetectProvider(tc.url)

						// Assert
						Expect(detected).To(Equal(tc.expected),
							"URL %s should be detected as %s", tc.url, tc.expected)

						logger.Log("Detected provider %s for URL: %s", detected, tc.url)
					})
				}
			})

			It("Should handle unknown providers", func() {
				testCases := []string{
					"https://bitbucket.org/user/repo.git",
					"https://unknown-git.com/user/repo.git",
					"ftp://example.com/repo",
					"invalid-url",
					"",
				}

				for _, url := range testCases {
					By(fmt.Sprintf("Testing unknown URL: %s", url), func() {
						// Act
						detected := types.DetectProvider(url)

						// Assert
						Expect(detected).To(Equal(types.ProviderType("")),
							"Unknown URL should return empty provider")

						logger.Log("Unknown provider detected for URL: %s", url)
					})
				}
			})

			It("Should not have false positives", func() {
				// Test cases where URL contains provider name but isn't actually that provider
				testCases := []struct {
					url         string
					expectedNot types.ProviderType
					description string
				}{
					{
						url:         "https://github.com/gitlab/repo.git", // GitLab repo on GitHub
						expectedNot: types.ProviderGitLab,
						description: "GitLab repo hosted on GitHub",
					},
					{
						url:         "https://gitlab.com/user/github-cli.git", // GitHub-named repo on GitLab
						expectedNot: types.ProviderGitHub,
						description: "GitHub-named repo on GitLab",
					},
				}

				for _, tc := range testCases {
					By(tc.description, func() {
						// Act
						detected := types.DetectProvider(tc.url)

						// Assert
						Expect(detected).NotTo(Equal(tc.expectedNot),
							"Should not falsely detect %s for URL %s", tc.expectedNot, tc.url)

						logger.Log("Correctly avoided false positive for: %s", tc.url)
					})
				}
			})
		})

		Context("When working with provider enum values", func() {
			It("Should maintain backward compatibility", func() {
				// Verify enum values haven't changed (important for stored data)
				Expect(string(types.ProviderGitHub)).To(Equal("github"))
				Expect(string(types.ProviderGitLab)).To(Equal("gitlab"))

				logger.Log("Provider enum values are backward compatible")
			})

			It("Should handle empty provider gracefully", func() {
				emptyProvider := types.ProviderType("")

				// Should not panic
				Expect(func() {
					_ = string(emptyProvider)
				}).NotTo(Panic())

				// Should not equal valid providers
				Expect(emptyProvider).NotTo(Equal(types.ProviderGitHub))
				Expect(emptyProvider).NotTo(Equal(types.ProviderGitLab))

				logger.Log("Empty provider handled gracefully")
			})
		})
	})

	Describe("Pointer Helper Functions", func() {
		Context("StringPtr", func() {
			It("Should create pointer to string", func() {
				// Arrange
				testString := "test value"

				// Act
				ptr := types.StringPtr(testString)

				// Assert
				Expect(ptr).NotTo(BeNil())
				Expect(*ptr).To(Equal(testString))

				logger.Log("StringPtr created successfully")
			})
		})

		Context("IntPtr", func() {
			It("Should create pointer to int", func() {
				// Arrange
				testInt := 42

				// Act
				ptr := types.IntPtr(testInt)

				// Assert
				Expect(ptr).NotTo(BeNil())
				Expect(*ptr).To(Equal(testInt))

				logger.Log("IntPtr created successfully")
			})
		})

		Context("BoolPtr", func() {
			It("Should create pointer to bool", func() {
				testCases := []bool{true, false}

				for _, testBool := range testCases {
					By(fmt.Sprintf("Testing bool value: %v", testBool), func() {
						// Act
						ptr := types.BoolPtr(testBool)

						// Assert
						Expect(ptr).NotTo(BeNil())
						Expect(*ptr).To(Equal(testBool))

						logger.Log("BoolPtr created successfully for value: %v", testBool)
					})
				}
			})
		})
	})

	Describe("Error Types", func() {
		Context("When creating custom errors", func() {
			It("Should create validation errors", func() {
				// Arrange
				message := "Test validation error"

				// Act
				err := fmt.Errorf("validation error: %s", message)

				// Assert
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(message))

				logger.Log("Validation error created: %v", err)
			})

			It("Should create authentication errors", func() {
				// Arrange
				message := "Test auth error"

				// Act
				err := fmt.Errorf("authentication error: %s", message)

				// Assert
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(message))

				logger.Log("Authentication error created: %v", err)
			})
		})
	})
})
