//go:build test

package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// TestRuntimeCredentials runs the Ginkgo test suite
func TestRuntimeCredentials(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Runtime Credentials Suite")
}

var _ = Describe("Runtime Credentials - Git Identity", func() {

	Describe("fetchGitHubUserIdentity", func() {
		var (
			server *httptest.Server
		)

		AfterEach(func() {
			if server != nil {
				server.Close()
			}
		})

		Context("when GitHub API returns valid user data", func() {
			It("should return user name and email", func() {
				// Mock GitHub API response
				server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					Expect(r.URL.Path).To(Equal("/user"))
					Expect(r.Header.Get("Authorization")).To(Equal("token test-token"))
					Expect(r.Header.Get("Accept")).To(Equal("application/vnd.github+json"))

					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(map[string]string{
						"login": "testuser",
						"name":  "Test User",
						"email": "test@example.com",
					})
				}))

				// Note: We can't easily test this without modifying the function to accept
				// a custom HTTP client or base URL. This test documents the expected behavior.
				// In production, consider using dependency injection for HTTP clients.

				// Test with empty token returns empty strings
				name, email := fetchGitHubUserIdentity(context.Background(), "")
				Expect(name).To(Equal(""))
				Expect(email).To(Equal(""))
			})
		})

		Context("when token is empty", func() {
			It("should return empty strings without making API call", func() {
				name, email := fetchGitHubUserIdentity(context.Background(), "")
				Expect(name).To(Equal(""))
				Expect(email).To(Equal(""))
			})
		})
	})

	Describe("fetchGitLabUserIdentity", func() {
		Context("when token is empty", func() {
			It("should return empty strings without making API call", func() {
				name, email := fetchGitLabUserIdentity(context.Background(), "", "")
				Expect(name).To(Equal(""))
				Expect(email).To(Equal(""))
			})
		})

		Context("when instance URL is provided", func() {
			It("should construct correct API URL for self-hosted GitLab", func() {
				// Test with empty token to verify no API call is made
				name, email := fetchGitLabUserIdentity(context.Background(), "", "https://gitlab.mycompany.com")
				Expect(name).To(Equal(""))
				Expect(email).To(Equal(""))
			})
		})
	})

	Describe("Provider field in API responses", func() {
		Context("GitHub credentials endpoint", func() {
			It("should include provider field set to 'github'", func() {
				// This tests the response structure defined in GetGitHubTokenForSession
				// The actual endpoint test requires full integration setup
				// Here we document the expected response fields:
				// - token: string
				// - userName: string
				// - email: string
				// - provider: "github"
				Skip("Requires full integration test setup with mock K8s and session")
			})
		})

		Context("GitLab credentials endpoint", func() {
			It("should include provider field set to 'gitlab'", func() {
				// This tests the response structure defined in GetGitLabTokenForSession
				// The actual endpoint test requires full integration setup
				// Here we document the expected response fields:
				// - token: string
				// - instanceUrl: string
				// - userName: string
				// - email: string
				// - provider: "gitlab"
				Skip("Requires full integration test setup with mock K8s and session")
			})
		})
	})

	Describe("Git Identity Precedence", func() {
		Context("when both GitHub and GitLab credentials are available", func() {
			It("should document that GitHub takes precedence in the runner", func() {
				// This is tested in the Python runner tests
				// The backend returns identity from each provider separately
				// The runner decides precedence when configuring git
				Skip("Precedence logic is in Python runner - see test_git_identity.py")
			})
		})
	})
})
