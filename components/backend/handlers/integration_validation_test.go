//go:build test

package handlers

import (
	test_constants "ambient-code-backend/tests/constants"
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Integration Validation", Label(test_constants.LabelUnit, test_constants.LabelHandlers), func() {
	// connRefusedURL returns a URL whose port is closed, triggering
	// "connection refused" from the dialer.
	connRefusedURL := func() string {
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		Expect(err).NotTo(HaveOccurred())
		addr := ln.Addr().String()
		ln.Close() // close immediately so the port is unreachable
		return "http://" + addr
	}

	Describe("networkError", func() {
		It("strips the URL and method from a *url.Error", func() {
			// Make a request to a closed port to produce a *url.Error.
			target := connRefusedURL()
			client := &http.Client{}
			req, err := http.NewRequestWithContext(context.Background(), "GET", target, nil)
			Expect(err).NotTo(HaveOccurred())

			_, doErr := client.Do(req)
			Expect(doErr).To(HaveOccurred())

			inner := networkError(doErr)
			Expect(inner.Error()).NotTo(ContainSubstring(target))
			Expect(inner.Error()).NotTo(ContainSubstring("GET"))
		})

		It("returns non-url.Error values unchanged", func() {
			original := net.UnknownNetworkError("test")
			Expect(networkError(original)).To(Equal(original))
		})
	})

	Describe("ValidateGitLabToken", func() {
		It("surfaces the network cause on connection refused", func() {
			target := connRefusedURL()
			_, err := ValidateGitLabToken(context.Background(), "glpat-secret-token", target)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("request failed:"))
			Expect(err.Error()).To(ContainSubstring("refused"))
		})

		It("does not leak the full request URL path", func() {
			target := connRefusedURL()
			_, err := ValidateGitLabToken(context.Background(), "glpat-secret-token", target)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).NotTo(ContainSubstring("/api/v4/"))
		})

		It("does not leak the token", func() {
			target := connRefusedURL()
			token := "glpat-secret-token-value"
			_, err := ValidateGitLabToken(context.Background(), token, target)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).NotTo(ContainSubstring(token))
		})

		It("returns true for a valid token", func() {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Authorization") == "Bearer valid-token" {
					w.WriteHeader(http.StatusOK)
				} else {
					w.WriteHeader(http.StatusUnauthorized)
				}
			}))
			defer ts.Close()

			valid, err := ValidateGitLabToken(context.Background(), "valid-token", ts.URL)
			Expect(err).NotTo(HaveOccurred())
			Expect(valid).To(BeTrue())
		})

		It("returns false for an invalid token", func() {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
			}))
			defer ts.Close()

			valid, err := ValidateGitLabToken(context.Background(), "bad-token", ts.URL)
			Expect(err).NotTo(HaveOccurred())
			Expect(valid).To(BeFalse())
		})
	})

	Describe("ValidateJiraToken", func() {
		It("surfaces the network cause when all endpoints fail", func() {
			target := connRefusedURL()
			_, err := ValidateJiraToken(context.Background(), target, "user@example.com", "api-token")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("request failed:"))
			Expect(err.Error()).To(ContainSubstring("refused"))
		})

		It("does not leak the full request URL path", func() {
			target := connRefusedURL()
			_, err := ValidateJiraToken(context.Background(), target, "user@example.com", "api-token")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).NotTo(ContainSubstring("/rest/api/"))
		})

		It("returns true for valid credentials", func() {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.Contains(r.URL.Path, "/rest/api/3/myself") {
					w.WriteHeader(http.StatusOK)
				} else {
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			defer ts.Close()

			valid, err := ValidateJiraToken(context.Background(), ts.URL, "user@example.com", "api-token")
			Expect(err).NotTo(HaveOccurred())
			Expect(valid).To(BeTrue())
		})
	})
})
