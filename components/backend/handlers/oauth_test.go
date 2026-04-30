//go:build test

package handlers

import (
	"ambient-code-backend/tests/config"
	test_constants "ambient-code-backend/tests/constants"
	"ambient-code-backend/tests/logger"
	"ambient-code-backend/tests/test_utils"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// buildTestHMACState constructs a signed state token using the same algorithm as the backend,
// so tests can create valid states without going through the HTTP handler.
func buildTestHMACState(stateJSON []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(stateJSON)
	sig := mac.Sum(nil)
	return base64.URLEncoding.EncodeToString(stateJSON) + "." + base64.URLEncoding.EncodeToString(sig)
}

var _ = Describe("Google OAuth Handler", Label(test_constants.LabelUnit, test_constants.LabelHandlers, test_constants.LabelGoogleAuth), func() {

	// -------------------------------------------------------------------------
	// Pure PKCE utility functions — no K8s needed
	// -------------------------------------------------------------------------

	Describe("generateCodeVerifier", func() {
		It("produces a base64url string of the correct length", func() {
			verifier, err := generateCodeVerifier()
			Expect(err).NotTo(HaveOccurred())
			// 32 raw bytes → 43 base64url chars (no padding)
			Expect(len(verifier)).To(Equal(43))
		})

		It("produces different values on each call", func() {
			v1, err := generateCodeVerifier()
			Expect(err).NotTo(HaveOccurred())
			v2, err := generateCodeVerifier()
			Expect(err).NotTo(HaveOccurred())
			Expect(v1).NotTo(Equal(v2))
		})

		It("only contains base64url-safe characters (RFC 7636 §4.1)", func() {
			for i := 0; i < 10; i++ {
				v, err := generateCodeVerifier()
				Expect(err).NotTo(HaveOccurred())
				Expect(v).To(MatchRegexp(`^[A-Za-z0-9_-]+$`))
			}
		})
	})

	Describe("generateCodeChallenge", func() {
		It("is deterministic for the same input", func() {
			verifier := "test-verifier-abc123"
			Expect(generateCodeChallenge(verifier)).To(Equal(generateCodeChallenge(verifier)))
		})

		It("matches the S256 spec: BASE64URL(SHA256(ASCII(verifier)))", func() {
			verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
			h := sha256.Sum256([]byte(verifier))
			expected := base64.RawURLEncoding.EncodeToString(h[:])
			Expect(generateCodeChallenge(verifier)).To(Equal(expected))
		})

		It("produces different challenges for different verifiers", func() {
			Expect(generateCodeChallenge("aaa")).NotTo(Equal(generateCodeChallenge("bbb")))
		})
	})

	Describe("pkceSecretKey", func() {
		It("returns a 64-char lowercase hex string", func() {
			key := pkceSecretKey("some-state-token")
			Expect(len(key)).To(Equal(64))
			Expect(key).To(MatchRegexp(`^[0-9a-f]+$`))
		})

		It("is deterministic for the same input", func() {
			Expect(pkceSecretKey("x")).To(Equal(pkceSecretKey("x")))
		})

		It("handles tokens containing characters invalid in K8s secret keys (=, +)", func() {
			// Standard base64 tokens contain =, +, / — these are valid K8s key chars only partially
			state := "abc==.def+/ghi"
			key := pkceSecretKey(state)
			Expect(key).To(MatchRegexp(`^[0-9a-f]+$`))
		})
	})

	// -------------------------------------------------------------------------
	// PKCE K8s storage — requires fake K8s client
	// -------------------------------------------------------------------------

	Describe("storePKCEVerifier / retrievePKCEVerifier", func() {
		var (
			savedK8sClient kubernetes.Interface
			savedNamespace string
			k8sTestUtils   *test_utils.K8sTestUtils
		)

		BeforeEach(func() {
			savedK8sClient = K8sClient
			savedNamespace = Namespace

			k8sTestUtils = test_utils.NewK8sTestUtils(false, *config.TestNamespace)
			SetupHandlerDependencies(k8sTestUtils)
			K8sClient = k8sTestUtils.K8sClient
			Namespace = *config.TestNamespace
		})

		AfterEach(func() {
			K8sClient = savedK8sClient
			Namespace = savedNamespace
		})

		It("round-trips: what is stored can be retrieved", func() {
			Expect(storePKCEVerifier(context.Background(), "state-abc", "verifier-abc")).To(Succeed())
			got, err := retrievePKCEVerifier(context.Background(), "state-abc")
			Expect(err).NotTo(HaveOccurred())
			Expect(got).To(Equal("verifier-abc"))
		})

		It("deletes the verifier after the first retrieval (one-time use)", func() {
			Expect(storePKCEVerifier(context.Background(), "ott-state", "ott-verifier")).To(Succeed())

			first, err := retrievePKCEVerifier(context.Background(), "ott-state")
			Expect(err).NotTo(HaveOccurred())
			Expect(first).To(Equal("ott-verifier"))

			second, err := retrievePKCEVerifier(context.Background(), "ott-state")
			Expect(err).NotTo(HaveOccurred())
			Expect(second).To(BeEmpty(), "verifier should have been consumed on first retrieval")
		})

		It("returns empty string (no error) when no verifier is stored for a state", func() {
			v, err := retrievePKCEVerifier(context.Background(), "nonexistent-state")
			Expect(err).NotTo(HaveOccurred())
			Expect(v).To(BeEmpty())
		})

		It("returns empty string (no error) when the K8s secret does not exist", func() {
			_ = K8sClient.CoreV1().Secrets(Namespace).Delete(
				context.Background(), "oauth-pkce-verifiers", metav1.DeleteOptions{})

			v, err := retrievePKCEVerifier(context.Background(), "any-state")
			Expect(err).NotTo(HaveOccurred())
			Expect(v).To(BeEmpty())
		})

		It("stores multiple states in the same secret without collisions", func() {
			pairs := map[string]string{
				"state-alpha": "verifier-alpha",
				"state-beta":  "verifier-beta",
				"state-gamma": "verifier-gamma",
			}
			for s, v := range pairs {
				Expect(storePKCEVerifier(context.Background(), s, v)).To(Succeed())
			}
			for s, expected := range pairs {
				got, err := retrievePKCEVerifier(context.Background(), s)
				Expect(err).NotTo(HaveOccurred())
				Expect(got).To(Equal(expected))
			}
		})
	})

	// -------------------------------------------------------------------------
	// exchangeOAuthCode — mock HTTP server for Google's token endpoint
	// -------------------------------------------------------------------------

	Describe("exchangeOAuthCode", func() {
		var (
			mockServer   *httptest.Server
			provider     *OAuthProvider
			capturedBody string
		)

		BeforeEach(func() {
			capturedBody = ""
			mockServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, _ := io.ReadAll(r.Body)
				capturedBody = string(body)
				w.Header().Set("Content-Type", "application/json")
				_, _ = fmt.Fprint(w, `{
					"access_token":"test-access-token",
					"refresh_token":"test-refresh-token",
					"expires_in":3600,
					"token_type":"Bearer"
				}`)
			}))

			provider = &OAuthProvider{
				Name:         "google",
				ClientID:     "cid",
				ClientSecret: "csecret",
				TokenURL:     mockServer.URL,
			}
		})

		AfterEach(func() { mockServer.Close() })

		It("includes code_verifier in the POST body when provided", func() {
			_, err := exchangeOAuthCode(context.Background(), provider,
				"auth-code", "https://cb.example.com/oauth2callback", "my-verifier")
			Expect(err).NotTo(HaveOccurred())
			Expect(capturedBody).To(ContainSubstring("code_verifier=my-verifier"))
		})

		It("omits code_verifier from the POST body when empty", func() {
			_, err := exchangeOAuthCode(context.Background(), provider,
				"auth-code", "https://cb.example.com/oauth2callback", "")
			Expect(err).NotTo(HaveOccurred())
			Expect(capturedBody).NotTo(ContainSubstring("code_verifier"))
		})

		It("includes the required OAuth grant parameters", func() {
			_, err := exchangeOAuthCode(context.Background(), provider,
				"my-code", "https://cb.example.com/oauth2callback", "")
			Expect(err).NotTo(HaveOccurred())
			Expect(capturedBody).To(ContainSubstring("code=my-code"))
			Expect(capturedBody).To(ContainSubstring("client_id=cid"))
			Expect(capturedBody).To(ContainSubstring("grant_type=authorization_code"))
			Expect(capturedBody).To(ContainSubstring("redirect_uri="))
		})

		It("returns all token fields from a successful exchange", func() {
			resp, err := exchangeOAuthCode(context.Background(), provider,
				"code", "https://cb.example.com/oauth2callback", "")
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.AccessToken).To(Equal("test-access-token"))
			Expect(resp.RefreshToken).To(Equal("test-refresh-token"))
			Expect(resp.ExpiresIn).To(Equal(int64(3600)))
			Expect(resp.TokenType).To(Equal("Bearer"))
		})

		It("returns a descriptive error when the server responds with 400", func() {
			errorServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = fmt.Fprint(w, `{"error":"invalid_grant","error_description":"Missing code verifier."}`)
			}))
			defer errorServer.Close()

			provider.TokenURL = errorServer.URL
			_, err := exchangeOAuthCode(context.Background(), provider,
				"bad-code", "https://cb.example.com/oauth2callback", "")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("token exchange failed with status 400"))
			Expect(err.Error()).To(ContainSubstring("Missing code verifier"))
		})
	})

	// -------------------------------------------------------------------------
	// Google credential K8s storage
	// -------------------------------------------------------------------------

	Describe("storeGoogleCredentials / GetGoogleCredentials", func() {
		var (
			savedK8sClient kubernetes.Interface
			savedNamespace string
			k8sTestUtils   *test_utils.K8sTestUtils
		)

		BeforeEach(func() {
			savedK8sClient = K8sClient
			savedNamespace = Namespace

			k8sTestUtils = test_utils.NewK8sTestUtils(false, *config.TestNamespace)
			SetupHandlerDependencies(k8sTestUtils)
			K8sClient = k8sTestUtils.K8sClient
			Namespace = *config.TestNamespace
		})

		AfterEach(func() {
			K8sClient = savedK8sClient
			Namespace = savedNamespace
		})

		It("round-trips credentials through store and retrieve", func() {
			expiry := time.Now().Add(1 * time.Hour).Truncate(time.Second)
			creds := &GoogleOAuthCredentials{
				UserID:       "alice",
				Email:        "alice@example.com",
				AccessToken:  "access-abc",
				RefreshToken: "refresh-def",
				Scopes:       []string{"https://www.googleapis.com/auth/drive"},
				ExpiresAt:    expiry,
				UpdatedAt:    time.Now().Truncate(time.Second),
			}
			Expect(storeGoogleCredentials(context.Background(), creds)).To(Succeed())

			got, err := GetGoogleCredentials(context.Background(), "alice")
			Expect(err).NotTo(HaveOccurred())
			Expect(got).NotTo(BeNil())
			Expect(got.UserID).To(Equal("alice"))
			Expect(got.Email).To(Equal("alice@example.com"))
			Expect(got.AccessToken).To(Equal("access-abc"))
			Expect(got.RefreshToken).To(Equal("refresh-def"))
		})

		It("returns nil (no error) when no credentials exist for a user", func() {
			got, err := GetGoogleCredentials(context.Background(), "unknown-user")
			Expect(err).NotTo(HaveOccurred())
			Expect(got).To(BeNil())
		})

		It("stores credentials for multiple users in the same secret without collisions", func() {
			users := []string{"userA", "userB", "userC"}
			for _, uid := range users {
				Expect(storeGoogleCredentials(context.Background(), &GoogleOAuthCredentials{
					UserID:      uid,
					AccessToken: "token-" + uid,
					ExpiresAt:   time.Now().Add(1 * time.Hour),
					UpdatedAt:   time.Now(),
				})).To(Succeed())
			}
			for _, uid := range users {
				got, err := GetGoogleCredentials(context.Background(), uid)
				Expect(err).NotTo(HaveOccurred())
				Expect(got.AccessToken).To(Equal("token-" + uid))
			}
		})

		It("overwrites credentials when stored again for the same user", func() {
			Expect(storeGoogleCredentials(context.Background(), &GoogleOAuthCredentials{
				UserID: "bob", AccessToken: "old", ExpiresAt: time.Now().Add(1 * time.Hour), UpdatedAt: time.Now(),
			})).To(Succeed())
			Expect(storeGoogleCredentials(context.Background(), &GoogleOAuthCredentials{
				UserID: "bob", AccessToken: "new", ExpiresAt: time.Now().Add(2 * time.Hour), UpdatedAt: time.Now(),
			})).To(Succeed())

			got, err := GetGoogleCredentials(context.Background(), "bob")
			Expect(err).NotTo(HaveOccurred())
			Expect(got.AccessToken).To(Equal("new"))
		})

		It("sanitizes OpenShift-style colon-separated userIDs", func() {
			Expect(storeGoogleCredentials(context.Background(), &GoogleOAuthCredentials{
				UserID:      "system:admin",
				AccessToken: "colon-tok",
				ExpiresAt:   time.Now().Add(1 * time.Hour),
				UpdatedAt:   time.Now(),
			})).To(Succeed())

			got, err := GetGoogleCredentials(context.Background(), "system:admin")
			Expect(err).NotTo(HaveOccurred())
			Expect(got.AccessToken).To(Equal("colon-tok"))
		})
	})

	// -------------------------------------------------------------------------
	// HTTP handlers — require fake K8s + valid auth token
	// -------------------------------------------------------------------------

	Describe("HTTP handlers", func() {
		var (
			httpTestUtils  *test_utils.HTTPTestUtils
			k8sTestUtils   *test_utils.K8sTestUtils
			savedK8sClient kubernetes.Interface
			savedNamespace string
			testToken      string
		)

		BeforeEach(func() {
			logger.Log("Setting up Google OAuth handler tests")

			savedK8sClient = K8sClient
			savedNamespace = Namespace

			k8sTestUtils = test_utils.NewK8sTestUtils(false, *config.TestNamespace)
			SetupHandlerDependencies(k8sTestUtils)
			K8sClient = k8sTestUtils.K8sClient
			Namespace = *config.TestNamespace

			httpTestUtils = test_utils.NewHTTPTestUtils()

			ctx := context.Background()
			_, err := k8sTestUtils.K8sClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{Name: *config.TestNamespace},
			}, metav1.CreateOptions{})
			if err != nil && !k8serrors.IsAlreadyExists(err) {
				Expect(err).NotTo(HaveOccurred())
			}
			_, err = k8sTestUtils.CreateTestRole(ctx, *config.TestNamespace, "google-oauth-test-role",
				[]string{"get", "list", "create", "update", "delete", "patch"}, "*", "")
			Expect(err).NotTo(HaveOccurred())

			token, _, err := httpTestUtils.SetValidTestToken(k8sTestUtils, *config.TestNamespace,
				[]string{"get", "list", "create", "update", "delete", "patch"}, "*", "",
				"google-oauth-test-role")
			Expect(err).NotTo(HaveOccurred())
			testToken = token

			os.Setenv("GOOGLE_OAUTH_CLIENT_ID", "test-google-client-id")
			os.Setenv("GOOGLE_OAUTH_CLIENT_SECRET", "test-google-client-secret")
			os.Setenv("OAUTH_STATE_SECRET", "test-hmac-secret-for-oauth")
			os.Setenv("BACKEND_URL", "https://backend.example.com")
		})

		AfterEach(func() {
			K8sClient = savedK8sClient
			Namespace = savedNamespace

			os.Unsetenv("GOOGLE_OAUTH_CLIENT_ID")
			os.Unsetenv("GOOGLE_OAUTH_CLIENT_SECRET")
			os.Unsetenv("OAUTH_STATE_SECRET")
			os.Unsetenv("BACKEND_URL")

			if k8sTestUtils != nil {
				_ = k8sTestUtils.K8sClient.CoreV1().Namespaces().Delete(
					context.Background(), *config.TestNamespace, metav1.DeleteOptions{})
			}
		})

		// ------------------------------------------------------------------
		// GetGoogleOAuthURLGlobal
		// ------------------------------------------------------------------

		Describe("GetGoogleOAuthURLGlobal", func() {
			It("returns 401 when no Authorization header is present", func() {
				c := httpTestUtils.CreateTestGinContext("POST", "/api/auth/google/connect", nil)
				httpTestUtils.SetUserContext("u", "U", "u@x.com")

				GetGoogleOAuthURLGlobal(c)

				httpTestUtils.AssertHTTPStatus(http.StatusUnauthorized)
			})

			It("returns 401 when userID is empty", func() {
				c := httpTestUtils.CreateTestGinContext("POST", "/api/auth/google/connect", nil)
				httpTestUtils.SetAuthHeader(testToken)
				c.Set("userID", "")

				GetGoogleOAuthURLGlobal(c)

				httpTestUtils.AssertHTTPStatus(http.StatusUnauthorized)
			})

			It("returns 503 when GOOGLE_OAUTH_CLIENT_ID is unset", func() {
				os.Unsetenv("GOOGLE_OAUTH_CLIENT_ID")

				c := httpTestUtils.CreateTestGinContext("POST", "/api/auth/google/connect", nil)
				httpTestUtils.SetAuthHeader(testToken)
				httpTestUtils.SetUserContext("test-user", "T", "t@x.com")

				GetGoogleOAuthURLGlobal(c)

				httpTestUtils.AssertHTTPStatus(http.StatusServiceUnavailable)
			})

			It("returns 500 when OAUTH_STATE_SECRET is unset", func() {
				os.Unsetenv("OAUTH_STATE_SECRET")

				c := httpTestUtils.CreateTestGinContext("POST", "/api/auth/google/connect", nil)
				httpTestUtils.SetAuthHeader(testToken)
				httpTestUtils.SetUserContext("test-user", "T", "t@x.com")

				GetGoogleOAuthURLGlobal(c)

				httpTestUtils.AssertHTTPStatus(http.StatusInternalServerError)
			})

			It("includes code_challenge and code_challenge_method=S256 in the returned URL", func() {
				c := httpTestUtils.CreateTestGinContext("POST", "/api/auth/google/connect", nil)
				httpTestUtils.SetAuthHeader(testToken)
				httpTestUtils.SetUserContext("test-user", "T", "t@x.com")

				GetGoogleOAuthURLGlobal(c)

				httpTestUtils.AssertHTTPStatus(http.StatusOK)
				var resp map[string]interface{}
				httpTestUtils.GetResponseJSON(&resp)

				authURL, ok := resp["url"].(string)
				Expect(ok).To(BeTrue(), "response must contain a 'url' string")
				Expect(authURL).To(ContainSubstring("code_challenge="))
				Expect(authURL).To(ContainSubstring("code_challenge_method=S256"))
			})

			It("stores a PKCE verifier that matches the challenge in the URL", func() {
				c := httpTestUtils.CreateTestGinContext("POST", "/api/auth/google/connect", nil)
				httpTestUtils.SetAuthHeader(testToken)
				httpTestUtils.SetUserContext("test-user", "T", "t@x.com")

				GetGoogleOAuthURLGlobal(c)

				httpTestUtils.AssertHTTPStatus(http.StatusOK)
				var resp map[string]interface{}
				httpTestUtils.GetResponseJSON(&resp)

				state := resp["state"].(string)
				authURL := resp["url"].(string)

				verifier, err := retrievePKCEVerifier(context.Background(), state)
				Expect(err).NotTo(HaveOccurred())
				Expect(verifier).NotTo(BeEmpty())

				// Challenge in URL must be SHA256(verifier) in base64url
				expectedChallenge := generateCodeChallenge(verifier)
				Expect(authURL).To(ContainSubstring("code_challenge=" + expectedChallenge))
			})
		})

		// ------------------------------------------------------------------
		// GetGoogleOAuthStatusGlobal
		// ------------------------------------------------------------------

		Describe("GetGoogleOAuthStatusGlobal", func() {
			It("returns connected:false when user has no stored credentials", func() {
				c := httpTestUtils.CreateTestGinContext("GET", "/api/auth/google/status", nil)
				httpTestUtils.SetAuthHeader(testToken)
				httpTestUtils.SetUserContext("fresh-user", "F", "f@x.com")

				GetGoogleOAuthStatusGlobal(c)

				httpTestUtils.AssertHTTPStatus(http.StatusOK)
				httpTestUtils.AssertJSONContains(map[string]interface{}{"connected": false})
			})

			It("returns connected:true with email when credentials are present", func() {
				Expect(storeGoogleCredentials(context.Background(), &GoogleOAuthCredentials{
					UserID:      "connected-user",
					Email:       "me@example.com",
					AccessToken: "tok",
					ExpiresAt:   time.Now().Add(1 * time.Hour),
					UpdatedAt:   time.Now(),
				})).To(Succeed())

				c := httpTestUtils.CreateTestGinContext("GET", "/api/auth/google/status", nil)
				httpTestUtils.SetAuthHeader(testToken)
				httpTestUtils.SetUserContext("connected-user", "C", "c@x.com")

				GetGoogleOAuthStatusGlobal(c)

				httpTestUtils.AssertHTTPStatus(http.StatusOK)
				httpTestUtils.AssertJSONContains(map[string]interface{}{
					"connected": true,
					"email":     "me@example.com",
				})
			})

			It("marks expired:true when the stored token is past its expiry", func() {
				Expect(storeGoogleCredentials(context.Background(), &GoogleOAuthCredentials{
					UserID:      "expired-user",
					AccessToken: "old",
					ExpiresAt:   time.Now().Add(-1 * time.Hour),
					UpdatedAt:   time.Now(),
				})).To(Succeed())

				c := httpTestUtils.CreateTestGinContext("GET", "/api/auth/google/status", nil)
				httpTestUtils.SetAuthHeader(testToken)
				httpTestUtils.SetUserContext("expired-user", "E", "e@x.com")

				GetGoogleOAuthStatusGlobal(c)

				httpTestUtils.AssertHTTPStatus(http.StatusOK)
				var resp map[string]interface{}
				httpTestUtils.GetResponseJSON(&resp)
				Expect(resp["connected"]).To(BeTrue())
				Expect(resp["expired"]).To(BeTrue())
			})

			It("returns 401 when no auth token is present", func() {
				c := httpTestUtils.CreateTestGinContext("GET", "/api/auth/google/status", nil)
				httpTestUtils.SetUserContext("u", "U", "u@x.com")

				GetGoogleOAuthStatusGlobal(c)

				httpTestUtils.AssertHTTPStatus(http.StatusUnauthorized)
			})
		})

		// ------------------------------------------------------------------
		// DisconnectGoogleOAuthGlobal
		// ------------------------------------------------------------------

		Describe("DisconnectGoogleOAuthGlobal", func() {
			It("removes credentials and reports success", func() {
				Expect(storeGoogleCredentials(context.Background(), &GoogleOAuthCredentials{
					UserID:      "to-remove",
					AccessToken: "tok",
					ExpiresAt:   time.Now().Add(1 * time.Hour),
					UpdatedAt:   time.Now(),
				})).To(Succeed())

				c := httpTestUtils.CreateTestGinContext("POST", "/api/auth/google/disconnect", nil)
				httpTestUtils.SetAuthHeader(testToken)
				httpTestUtils.SetUserContext("to-remove", "R", "r@x.com")

				DisconnectGoogleOAuthGlobal(c)

				httpTestUtils.AssertHTTPStatus(http.StatusOK)

				got, err := GetGoogleCredentials(context.Background(), "to-remove")
				Expect(err).NotTo(HaveOccurred())
				Expect(got).To(BeNil())
			})

			It("returns success even when the user was never connected", func() {
				c := httpTestUtils.CreateTestGinContext("POST", "/api/auth/google/disconnect", nil)
				httpTestUtils.SetAuthHeader(testToken)
				httpTestUtils.SetUserContext("never-connected", "N", "n@x.com")

				DisconnectGoogleOAuthGlobal(c)

				httpTestUtils.AssertHTTPStatus(http.StatusOK)
			})

			It("returns 401 when no auth token is present", func() {
				c := httpTestUtils.CreateTestGinContext("POST", "/api/auth/google/disconnect", nil)
				httpTestUtils.SetUserContext("u", "U", "u@x.com")

				DisconnectGoogleOAuthGlobal(c)

				httpTestUtils.AssertHTTPStatus(http.StatusUnauthorized)
			})
		})

		// ------------------------------------------------------------------
		// PKCE verifier is present during HandleOAuth2Callback processing
		// ------------------------------------------------------------------

		Describe("PKCE verifier lifecycle during callback", func() {
			It("a verifier stored against a signed state is retrievable before the callback consumes it", func() {
				// Build the same signed state that GetGoogleOAuthURLGlobal would produce
				stateData := map[string]interface{}{
					"provider":  "google",
					"userID":    "test-user",
					"timestamp": time.Now().Unix(),
					"cluster":   true,
				}
				stateJSON, err := json.Marshal(stateData)
				Expect(err).NotTo(HaveOccurred())

				stateToken := buildTestHMACState(stateJSON, "test-hmac-secret-for-oauth")

				Expect(storePKCEVerifier(context.Background(), stateToken, "known-verifier")).To(Succeed())

				v, err := retrievePKCEVerifier(context.Background(), stateToken)
				Expect(err).NotTo(HaveOccurred())
				Expect(v).To(Equal("known-verifier"))

				// Verifier is consumed; a second retrieval returns empty
				gone, _ := retrievePKCEVerifier(context.Background(), stateToken)
				Expect(gone).To(BeEmpty())
			})

			It("returns empty string for a plain UUID-style state (workspace-mcp generated, no stored verifier)", func() {
				plainState := "b5fbe39b07ffefa6402aa24dbeed3a94"
				v, err := retrievePKCEVerifier(context.Background(), plainState)
				Expect(err).NotTo(HaveOccurred())
				Expect(v).To(BeEmpty())
			})
		})
	})
})
