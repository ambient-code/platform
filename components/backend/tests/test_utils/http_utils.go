// Package test_utils provides common utilities for testing HTTP handlers and API endpoints
package test_utils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"time"

	"ambient-code-backend/tests/config"

	"github.com/gin-gonic/gin"
	. "github.com/onsi/gomega"
)

// HTTPTestUtils provides utilities for testing HTTP endpoints
type HTTPTestUtils struct {
	recorder *httptest.ResponseRecorder
	context  *gin.Context
	engine   *gin.Engine
}

// NewHTTPTestUtils creates a new HTTP test utilities instance
func NewHTTPTestUtils() *HTTPTestUtils {
	gin.SetMode(gin.TestMode)
	return &HTTPTestUtils{
		recorder: httptest.NewRecorder(),
		engine:   gin.New(),
	}
}

// CreateTestGinContext creates a test Gin context with the given HTTP method, path, and body
func (h *HTTPTestUtils) CreateTestGinContext(method, path string, body interface{}) *gin.Context {
	var reqBody io.Reader
	if body != nil {
		if bodyStr, ok := body.(string); ok {
			reqBody = strings.NewReader(bodyStr)
		} else {
			jsonBody, err := json.Marshal(body)
			Expect(err).NotTo(HaveOccurred(), "Failed to marshal request body to JSON")
			reqBody = bytes.NewBuffer(jsonBody)
		}
	}

	req := httptest.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")

	h.recorder = httptest.NewRecorder()
	h.context, _ = gin.CreateTestContext(h.recorder)
	h.context.Request = req

	return h.context
}

// SetAuthHeader sets authentication header for the test context
// Also sets userID in context so getUserSubjectFromContext works
// NOTE: This sets an arbitrary token without RBAC validation.
// For tests that need RBAC validation, use SetValidTestToken instead.
func (h *HTTPTestUtils) SetAuthHeader(token string) {
	if h.context != nil && h.context.Request != nil {
		h.context.Request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
		// Set userID in context so getUserSubjectFromContext can extract it
		// Use a default test user ID if not already set
		if _, exists := h.context.Get("userID"); !exists {
			h.context.Set("userID", "test-user")
		}
	}
}

// SetValidTestToken creates a ServiceAccount with RBAC permissions and sets a valid test token.
// This ensures tests use tokens that match the RBAC security model, not just arbitrary strings.
//
// Parameters:
//   - k8sUtils: K8sTestUtils instance to create ServiceAccount and RoleBinding
//   - namespace: The namespace where resources will be created
//   - verbs: List of verbs to grant (e.g., ["get", "list", "create", "update", "delete"])
//   - resource: The resource type (e.g., "agenticsessions", "projectsettings", "*" for all)
//   - saName: Optional ServiceAccount name (auto-generated if empty)
//   - roleName: Optional pre-existing Role name (if provided, uses this Role instead of creating a new one)
//
// Example:
//
//	// Use a pre-created Role
//	token, saName, err := httpUtils.SetValidTestToken(k8sUtils, "test-project", []string{"get", "list"}, "agenticsessions", "", "read-only-role")
//	Expect(err).NotTo(HaveOccurred())
//
//	// Or create a new Role automatically
//	token, saName, err := httpUtils.SetValidTestToken(k8sUtils, "test-project", []string{"get", "list"}, "agenticsessions", "", "")
//	Expect(err).NotTo(HaveOccurred())
func (h *HTTPTestUtils) SetValidTestToken(k8sUtils *K8sTestUtils, namespace string, verbs []string, resource string, saName string, roleName string) (string, string, error) {
	if k8sUtils == nil {
		return "", "", fmt.Errorf("k8sUtils cannot be nil")
	}
	if len(verbs) == 0 {
		verbs = []string{"get", "list", "create", "update", "delete", "patch"}
	}
	if resource == "" {
		resource = "*"
	}

	ctx := context.Background()
	token, createdSAName, err := k8sUtils.CreateValidTestToken(ctx, namespace, verbs, resource, saName, roleName)
	if err != nil {
		return "", "", fmt.Errorf("failed to create valid test token: %w", err)
	}

	// Set the token in the auth header
	h.SetAuthHeader(token)

	return token, createdSAName, nil
}

// SetUserContext sets user context headers and gin context values for testing
func (h *HTTPTestUtils) SetUserContext(userID, userName, userEmail string) {
	if h.context != nil && h.context.Request != nil {
		h.context.Request.Header.Set("X-Remote-User", userID)
		h.context.Request.Header.Set("X-Remote-User-Display-Name", userName)
		h.context.Request.Header.Set("X-Remote-User-Email", userEmail)

		// Also set gin.Context values that handlers often expect
		h.context.Set("userID", userID)
		h.context.Set("user", map[string]interface{}{
			"id":    userID,
			"name":  userName,
			"email": userEmail,
		})
		h.context.Set("userEmail", userEmail)
		h.context.Set("userName", userName)
	}
}

// SetProjectContext sets project context for testing
func (h *HTTPTestUtils) SetProjectContext(projectName string) {
	if h.context != nil {
		h.context.Set("project", projectName)
	}
}

// AutoSetProjectContextFromParams automatically sets project context if projectName param exists
func (h *HTTPTestUtils) AutoSetProjectContextFromParams() {
	if h.context != nil {
		for _, param := range h.context.Params {
			if param.Key == "projectName" && param.Value != "" {
				h.SetProjectContext(param.Value)
				break
			}
		}
	}
}

// GetResponseRecorder returns the HTTP response recorder
func (h *HTTPTestUtils) GetResponseRecorder() *httptest.ResponseRecorder {
	return h.recorder
}

// GetResponseBody returns the response body as string
func (h *HTTPTestUtils) GetResponseBody() string {
	return h.recorder.Body.String()
}

// GetResponseJSON unmarshals the response body into the provided interface
// If target is a map[string]interface{}, it also adds the status code
func (h *HTTPTestUtils) GetResponseJSON(target interface{}) {
	err := json.Unmarshal(h.recorder.Body.Bytes(), target)
	Expect(err).NotTo(HaveOccurred(), "Failed to unmarshal response JSON")

	// Safely add status code if target is a map type
	if targetMap, ok := target.(*map[string]interface{}); ok && targetMap != nil {
		(*targetMap)["statusCode"] = h.recorder.Code
	}
}

// AssertHTTPStatus asserts the HTTP status code
func (h *HTTPTestUtils) AssertHTTPStatus(expectedStatus int) {
	statusCode := h.recorder.Code
	if reflect.TypeOf(statusCode).Kind() == reflect.Float64 {
		Expect(statusCode).To(Equal(float64(expectedStatus)),
			fmt.Sprintf("Expected HTTP status %d, got %d. Response body: %s",
				expectedStatus, h.recorder.Code, h.GetResponseBody()))
	} else {
		Expect(statusCode).To(Equal(expectedStatus),
			fmt.Sprintf("Expected HTTP status %d, got %d. Response body: %s",
				expectedStatus, h.recorder.Code, h.GetResponseBody()))
	}

}

// AssertHTTPSuccess asserts that the HTTP response is successful (2xx)
func (h *HTTPTestUtils) AssertHTTPSuccess() {
	Expect(h.recorder.Code).To(BeNumerically(">=", 200), "Expected successful HTTP status")
	Expect(h.recorder.Code).To(BeNumerically("<", 300), "Expected successful HTTP status")
}

// AssertHTTPError asserts that the HTTP response is an error (4xx or 5xx)
func (h *HTTPTestUtils) AssertHTTPError() {
	Expect(h.recorder.Code).To(BeNumerically(">=", 400), "Expected error HTTP status")
}

// AssertJSONContains asserts that the response JSON contains the expected key-value pairs
func (h *HTTPTestUtils) AssertJSONContains(expectedFields map[string]interface{}) {
	var responseData map[string]interface{}
	h.GetResponseJSON(&responseData)

	for key, expectedValue := range expectedFields {
		Expect(responseData).To(HaveKey(key), fmt.Sprintf("Response should contain key '%s'", key))
		Expect(responseData[key]).To(Equal(expectedValue),
			fmt.Sprintf("Expected '%s' to be '%v', got '%v'", key, expectedValue, responseData[key]))
	}
}

// AssertJSONStructure asserts that the response JSON has the expected structure
func (h *HTTPTestUtils) AssertJSONStructure(expectedKeys []string) {
	var responseData map[string]interface{}
	h.GetResponseJSON(&responseData)

	for _, key := range expectedKeys {
		Expect(responseData).To(HaveKey(key), fmt.Sprintf("Response should contain key '%s'", key))
	}
}

// AssertErrorMessage asserts that the response contains an error message
func (h *HTTPTestUtils) AssertErrorMessage(expectedMessage string) {
	var responseData map[string]interface{}
	h.GetResponseJSON(&responseData)

	Expect(responseData).To(HaveKey("error"), "Response should contain error field")
	errorMessage := responseData["error"].(string)
	Expect(errorMessage).To(ContainSubstring(expectedMessage),
		fmt.Sprintf("Expected error message to contain '%s', got '%s'", expectedMessage, errorMessage))
}

// HTTPClient represents a test HTTP client with retry capabilities
type HTTPClient struct {
	client         *http.Client
	baseURL        string
	defaultHeaders map[string]string
}

// NewHTTPClient creates a new test HTTP client
func NewHTTPClient(baseURL string) *HTTPClient {
	return &HTTPClient{
		client: &http.Client{
			Timeout: *config.APITimeout,
		},
		baseURL:        baseURL,
		defaultHeaders: make(map[string]string),
	}
}

// SetDefaultHeader sets a default header for all requests
func (c *HTTPClient) SetDefaultHeader(key, value string) {
	c.defaultHeaders[key] = value
}

// DoRequest performs an HTTP request with retry logic
func (c *HTTPClient) DoRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	url := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set default headers
	for key, value := range c.defaultHeaders {
		req.Header.Set(key, value)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// Retry logic
	var resp *http.Response
	for attempt := 1; attempt <= *config.RetryAttempts; attempt++ {
		resp, err = c.client.Do(req)
		if err == nil && resp.StatusCode < 500 {
			return resp, nil
		}

		if attempt < *config.RetryAttempts {
			delay := time.Duration(attempt) * (*config.RetryDelay)
			if delay > *config.MaxRetryDelay {
				delay = *config.MaxRetryDelay
			}
			time.Sleep(delay)
		}
	}

	return resp, err
}

// GetJSON performs a GET request and unmarshals the response to target
func (c *HTTPClient) GetJSON(ctx context.Context, path string, target interface{}) error {
	resp, err := c.DoRequest(ctx, "GET", path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP error %d: %s", resp.StatusCode, string(body))
	}

	return json.NewDecoder(resp.Body).Decode(target)
}

// PostJSON performs a POST request with JSON body
func (c *HTTPClient) PostJSON(ctx context.Context, path string, body interface{}) (*http.Response, error) {
	return c.DoRequest(ctx, "POST", path, body)
}

// PutJSON performs a PUT request with JSON body
func (c *HTTPClient) PutJSON(ctx context.Context, path string, body interface{}) (*http.Response, error) {
	return c.DoRequest(ctx, "PUT", path, body)
}

// DeleteRequest performs a DELETE request
func (c *HTTPClient) DeleteRequest(ctx context.Context, path string) (*http.Response, error) {
	return c.DoRequest(ctx, "DELETE", path, nil)
}
