package middleware_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ambient-code-backend/handlers"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/rest"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// fakeK8sAPI returns an httptest.Server that responds to SelfSubjectAccessReview
// requests with the given status code and body.
func fakeK8sAPI(statusCode int, body string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		w.Write([]byte(body))
	}))
}

// ssarAllowedBody returns a JSON body for an SSAR response that allows access.
func ssarAllowedBody() string {
	return `{
		"apiVersion": "authorization.k8s.io/v1",
		"kind": "SelfSubjectAccessReview",
		"status": {"allowed": true}
	}`
}

// ssarDeniedBody returns a JSON body for an SSAR response that denies access.
func ssarDeniedBody() string {
	return `{
		"apiVersion": "authorization.k8s.io/v1",
		"kind": "SelfSubjectAccessReview",
		"status": {"allowed": false}
	}`
}

// setupRouter creates a gin router with ValidateProjectContext middleware
// and a simple OK handler behind it.
func setupRouter() *gin.Engine {
	r := gin.New()
	projectGroup := r.Group("/api/projects/:projectName")
	projectGroup.Use(handlers.ValidateProjectContext())
	projectGroup.GET("/sessions", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"project": c.GetString("project")})
	})
	return r
}

// doRequest performs a GET request against the test router with the given
// Authorization header and project name.
func doRequest(router *gin.Engine, project, authHeader string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/projects/"+project+"/sessions", nil)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	router.ServeHTTP(w, req)
	return w
}

func TestValidateProjectContext_ExpiredToken_Returns401(t *testing.T) {
	// Stand up a fake K8s API that returns 401 Unauthorized for all requests,
	// simulating an expired ServiceAccount token.
	k8s := fakeK8sAPI(http.StatusUnauthorized, `{"kind":"Status","apiVersion":"v1","status":"Failure","message":"Unauthorized","reason":"Unauthorized","code":401}`)
	defer k8s.Close()

	handlers.BaseKubeConfig = &rest.Config{Host: k8s.URL}
	defer func() { handlers.BaseKubeConfig = nil }()

	router := setupRouter()
	w := doRequest(router, "test-project", "Bearer expired-token")

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var body map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "Token expired or invalid", body["error"])
}

func TestValidateProjectContext_ServerError_Returns500(t *testing.T) {
	// Fake K8s API returns 500 — should propagate as 500, not 401.
	k8s := fakeK8sAPI(http.StatusInternalServerError, `{"kind":"Status","apiVersion":"v1","status":"Failure","message":"internal error","reason":"InternalError","code":500}`)
	defer k8s.Close()

	handlers.BaseKubeConfig = &rest.Config{Host: k8s.URL}
	defer func() { handlers.BaseKubeConfig = nil }()

	router := setupRouter()
	w := doRequest(router, "test-project", "Bearer valid-token")

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var body map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "Failed to perform access review", body["error"])
}

func TestValidateProjectContext_ValidToken_Allowed(t *testing.T) {
	k8s := fakeK8sAPI(http.StatusCreated, ssarAllowedBody())
	defer k8s.Close()

	handlers.BaseKubeConfig = &rest.Config{Host: k8s.URL}
	defer func() { handlers.BaseKubeConfig = nil }()

	router := setupRouter()
	w := doRequest(router, "test-project", "Bearer good-token")

	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "test-project", body["project"])
}

func TestValidateProjectContext_ValidToken_Denied(t *testing.T) {
	k8s := fakeK8sAPI(http.StatusCreated, ssarDeniedBody())
	defer k8s.Close()

	handlers.BaseKubeConfig = &rest.Config{Host: k8s.URL}
	defer func() { handlers.BaseKubeConfig = nil }()

	router := setupRouter()
	w := doRequest(router, "test-project", "Bearer good-token")

	assert.Equal(t, http.StatusForbidden, w.Code)

	var body map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "Unauthorized to access project", body["error"])
}

func TestValidateProjectContext_NoToken_Returns401(t *testing.T) {
	router := setupRouter()
	w := doRequest(router, "test-project", "")

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var body map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "User token required", body["error"])
}

func TestValidateProjectContext_InvalidProjectName_Returns400(t *testing.T) {
	handlers.BaseKubeConfig = &rest.Config{Host: "https://unused"}
	defer func() { handlers.BaseKubeConfig = nil }()

	router := setupRouter()
	// Kubernetes names can't contain uppercase or special chars
	w := doRequest(router, "INVALID_PROJECT", "Bearer some-token")

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var body map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "Invalid project name format", body["error"])
}

func TestValidateProjectContext_Forbidden_Returns403(t *testing.T) {
	// K8s API returns 403 Forbidden — should propagate as 403, not 401 or 500.
	k8s := fakeK8sAPI(http.StatusForbidden, `{"kind":"Status","apiVersion":"v1","status":"Failure","message":"forbidden","reason":"Forbidden","code":403}`)
	defer k8s.Close()

	handlers.BaseKubeConfig = &rest.Config{Host: k8s.URL}
	defer func() { handlers.BaseKubeConfig = nil }()

	router := setupRouter()
	w := doRequest(router, "test-project", "Bearer some-token")

	// K8s 403 on SSAR create is an API error (not an SSAR denial),
	// so it falls through to the non-unauthorized error path → 500.
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
