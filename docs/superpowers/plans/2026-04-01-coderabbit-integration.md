# CodeRabbit Integration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add CodeRabbit as a native integration — API key storage, runtime credential injection, and frontend integration card.

**Architecture:** Backend stores API keys in a per-user K8s Secret (`coderabbit-credentials`), exposes connect/disconnect/status/test endpoints and a runtime credential fetch endpoint. Runner fetches the key at session startup and sets `CODERABBIT_API_KEY` env var. Frontend shows a connection card on the integrations page.

**Tech Stack:** Go (Gin, K8s client-go), TypeScript (Next.js, React Query, shadcn/ui), Python (runner)

---

### Task 1: Backend — CodeRabbit Auth Handlers

**Files:**
- Create: `components/backend/handlers/coderabbit_auth.go`

- [ ] **Step 1: Create the credential struct and storage functions**

Create `components/backend/handlers/coderabbit_auth.go`:

```go
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CodeRabbitCredentials represents cluster-level CodeRabbit credentials for a user
type CodeRabbitCredentials struct {
	UserID    string    `json:"userId"`
	APIKey    string    `json:"apiKey"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// storeCodeRabbitCredentials stores CodeRabbit credentials in cluster-level Secret
func storeCodeRabbitCredentials(ctx context.Context, creds *CodeRabbitCredentials) error {
	if creds == nil || creds.UserID == "" {
		return fmt.Errorf("invalid credentials payload")
	}

	const secretName = "coderabbit-credentials"

	for i := 0; i < 3; i++ {
		secret, err := K8sClient.CoreV1().Secrets(Namespace).Get(ctx, secretName, v1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				secret = &corev1.Secret{
					ObjectMeta: v1.ObjectMeta{
						Name:      secretName,
						Namespace: Namespace,
						Labels: map[string]string{
							"app":                      "ambient-code",
							"ambient-code.io/provider": "coderabbit",
						},
					},
					Type: corev1.SecretTypeOpaque,
					Data: map[string][]byte{},
				}
				if _, cerr := K8sClient.CoreV1().Secrets(Namespace).Create(ctx, secret, v1.CreateOptions{}); cerr != nil && !errors.IsAlreadyExists(cerr) {
					return fmt.Errorf("failed to create Secret: %w", cerr)
				}
				secret, err = K8sClient.CoreV1().Secrets(Namespace).Get(ctx, secretName, v1.GetOptions{})
				if err != nil {
					return fmt.Errorf("failed to fetch Secret after create: %w", err)
				}
			} else {
				return fmt.Errorf("failed to get Secret: %w", err)
			}
		}

		if secret.Data == nil {
			secret.Data = map[string][]byte{}
		}

		b, err := json.Marshal(creds)
		if err != nil {
			return fmt.Errorf("failed to marshal credentials: %w", err)
		}
		secret.Data[creds.UserID] = b

		if _, uerr := K8sClient.CoreV1().Secrets(Namespace).Update(ctx, secret, v1.UpdateOptions{}); uerr != nil {
			if errors.IsConflict(uerr) {
				continue
			}
			return fmt.Errorf("failed to update Secret: %w", uerr)
		}
		return nil
	}
	return fmt.Errorf("failed to update Secret after retries")
}

// GetCodeRabbitCredentials retrieves cluster-level CodeRabbit credentials for a user
func GetCodeRabbitCredentials(ctx context.Context, userID string) (*CodeRabbitCredentials, error) {
	if userID == "" {
		return nil, fmt.Errorf("userID is required")
	}

	const secretName = "coderabbit-credentials"

	secret, err := K8sClient.CoreV1().Secrets(Namespace).Get(ctx, secretName, v1.GetOptions{})
	if err != nil {
		return nil, err
	}

	if secret.Data == nil || len(secret.Data[userID]) == 0 {
		return nil, nil
	}

	var creds CodeRabbitCredentials
	if err := json.Unmarshal(secret.Data[userID], &creds); err != nil {
		return nil, fmt.Errorf("failed to parse credentials: %w", err)
	}

	return &creds, nil
}

// DeleteCodeRabbitCredentials removes CodeRabbit credentials for a user
func DeleteCodeRabbitCredentials(ctx context.Context, userID string) error {
	if userID == "" {
		return fmt.Errorf("userID is required")
	}

	const secretName = "coderabbit-credentials"

	for i := 0; i < 3; i++ {
		secret, err := K8sClient.CoreV1().Secrets(Namespace).Get(ctx, secretName, v1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				return nil
			}
			return fmt.Errorf("failed to get Secret: %w", err)
		}

		if secret.Data == nil || len(secret.Data[userID]) == 0 {
			return nil
		}

		delete(secret.Data, userID)

		if _, uerr := K8sClient.CoreV1().Secrets(Namespace).Update(ctx, secret, v1.UpdateOptions{}); uerr != nil {
			if errors.IsConflict(uerr) {
				continue
			}
			return fmt.Errorf("failed to update Secret: %w", uerr)
		}
		return nil
	}
	return fmt.Errorf("failed to update Secret after retries")
}
```

- [ ] **Step 2: Add the HTTP handlers**

Append to `components/backend/handlers/coderabbit_auth.go`:

```go
// ConnectCodeRabbit handles POST /api/auth/coderabbit/connect
func ConnectCodeRabbit(c *gin.Context) {
	reqK8s, _ := GetK8sClientsForRequest(c)
	if reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		return
	}

	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User authentication required"})
		return
	}
	if !isValidUserID(userID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user identifier"})
		return
	}

	var req struct {
		APIKey string `json:"apiKey" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate the API key against CodeRabbit's API
	valid, err := ValidateCodeRabbitAPIKey(c.Request.Context(), req.APIKey)
	if err != nil {
		log.Printf("Failed to validate CodeRabbit API key for user %s: %v", userID, err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to validate API key with CodeRabbit"})
		return
	}
	if !valid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid CodeRabbit API key"})
		return
	}

	creds := &CodeRabbitCredentials{
		UserID:    userID,
		APIKey:    req.APIKey,
		UpdatedAt: time.Now(),
	}

	if err := storeCodeRabbitCredentials(c.Request.Context(), creds); err != nil {
		log.Printf("Failed to store CodeRabbit credentials for user %s: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save CodeRabbit credentials"})
		return
	}

	log.Printf("✓ Stored CodeRabbit credentials for user %s", userID)
	c.JSON(http.StatusOK, gin.H{"message": "CodeRabbit connected successfully"})
}

// GetCodeRabbitStatus handles GET /api/auth/coderabbit/status
func GetCodeRabbitStatus(c *gin.Context) {
	reqK8s, _ := GetK8sClientsForRequest(c)
	if reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		return
	}

	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User authentication required"})
		return
	}

	creds, err := GetCodeRabbitCredentials(c.Request.Context(), userID)
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusOK, gin.H{"connected": false})
			return
		}
		log.Printf("Failed to get CodeRabbit credentials for user %s: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check CodeRabbit status"})
		return
	}

	if creds == nil {
		c.JSON(http.StatusOK, gin.H{"connected": false})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"connected": true,
		"updatedAt": creds.UpdatedAt.Format(time.RFC3339),
	})
}

// DisconnectCodeRabbit handles DELETE /api/auth/coderabbit/disconnect
func DisconnectCodeRabbit(c *gin.Context) {
	reqK8s, _ := GetK8sClientsForRequest(c)
	if reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		return
	}

	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User authentication required"})
		return
	}

	if err := DeleteCodeRabbitCredentials(c.Request.Context(), userID); err != nil {
		log.Printf("Failed to delete CodeRabbit credentials for user %s: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to disconnect CodeRabbit"})
		return
	}

	log.Printf("✓ Deleted CodeRabbit credentials for user %s", userID)
	c.JSON(http.StatusOK, gin.H{"message": "CodeRabbit disconnected successfully"})
}
```

- [ ] **Step 3: Verify Go code compiles**

Run: `cd components/backend && go build ./...`
Expected: Clean build, no errors.

- [ ] **Step 4: Commit**

```bash
git add components/backend/handlers/coderabbit_auth.go
git commit -m "feat(backend): add CodeRabbit auth handlers and credential storage"
```

---

### Task 2: Backend — Validation, Test Connection, and Integration Status

**Files:**
- Modify: `components/backend/handlers/integration_validation.go`
- Modify: `components/backend/handlers/integrations_status.go`

- [ ] **Step 1: Add ValidateCodeRabbitAPIKey and TestCodeRabbitConnection**

Append to `components/backend/handlers/integration_validation.go`:

```go
// ValidateCodeRabbitAPIKey checks if a CodeRabbit API key is valid
func ValidateCodeRabbitAPIKey(ctx context.Context, apiKey string) (bool, error) {
	if apiKey == "" {
		return false, fmt.Errorf("API key is empty")
	}

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.coderabbit.ai/api/v1/health", nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("request failed")
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK, nil
}

// TestCodeRabbitConnection handles POST /api/auth/coderabbit/test
func TestCodeRabbitConnection(c *gin.Context) {
	var req struct {
		APIKey string `json:"apiKey" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	valid, err := ValidateCodeRabbitAPIKey(c.Request.Context(), req.APIKey)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"valid": false, "error": err.Error()})
		return
	}

	if !valid {
		c.JSON(http.StatusOK, gin.H{"valid": false, "error": "Invalid API key"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"valid": true, "message": "CodeRabbit connection successful"})
}
```

- [ ] **Step 2: Add CodeRabbit to GetIntegrationsStatus**

In `components/backend/handlers/integrations_status.go`, add after the MCP server credentials line:

```go
// CodeRabbit status
response["coderabbit"] = getCodeRabbitStatusForUser(ctx, userID)
```

And add the helper function:

```go
func getCodeRabbitStatusForUser(ctx context.Context, userID string) gin.H {
	creds, err := GetCodeRabbitCredentials(ctx, userID)
	if err != nil || creds == nil {
		return gin.H{"connected": false}
	}

	return gin.H{
		"connected": true,
		"updatedAt": creds.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		"valid":     true,
	}
}
```

- [ ] **Step 3: Verify Go code compiles**

Run: `cd components/backend && go build ./...`
Expected: Clean build, no errors.

- [ ] **Step 4: Commit**

```bash
git add components/backend/handlers/integration_validation.go components/backend/handlers/integrations_status.go
git commit -m "feat(backend): add CodeRabbit validation, test endpoint, and integration status"
```

---

### Task 3: Backend — Runtime Credential Fetch and Route Registration

**Files:**
- Modify: `components/backend/handlers/runtime_credentials.go`
- Modify: `components/backend/routes.go`

- [ ] **Step 1: Add GetCodeRabbitCredentialsForSession**

Append to `components/backend/handlers/runtime_credentials.go`:

```go
// GetCodeRabbitCredentialsForSession handles GET /api/projects/:project/agentic-sessions/:session/credentials/coderabbit
func GetCodeRabbitCredentialsForSession(c *gin.Context) {
	project := c.Param("projectName")
	session := c.Param("sessionName")

	reqK8s, reqDyn := GetK8sClientsForRequest(c)
	if reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		return
	}

	gvr := GetAgenticSessionV1Alpha1Resource()
	obj, err := reqDyn.Resource(gvr).Namespace(project).Get(c.Request.Context(), session, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
			return
		}
		log.Printf("Failed to get session %s/%s: %v", project, session, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get session"})
		return
	}

	userID, found, err := unstructured.NestedString(obj.Object, "spec", "userContext", "userId")
	if !found || err != nil || userID == "" {
		log.Printf("Failed to extract userID from session %s/%s: found=%v, err=%v", project, session, found, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User ID not found in session"})
		return
	}

	authenticatedUserID := c.GetString("userID")
	if authenticatedUserID != "" && authenticatedUserID != userID {
		log.Printf("RBAC violation: user %s attempted to access credentials for session owned by %s", authenticatedUserID, userID)
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied: session belongs to different user"})
		return
	}

	creds, err := GetCodeRabbitCredentials(c.Request.Context(), userID)
	if err != nil {
		log.Printf("Failed to get CodeRabbit credentials for user %s: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get CodeRabbit credentials"})
		return
	}

	if creds == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "CodeRabbit credentials not configured"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"apiKey": creds.APIKey,
	})
}
```

- [ ] **Step 2: Register routes**

In `components/backend/routes.go`, add cluster-scoped routes after the GitLab test line (`api.POST("/auth/gitlab/test", ...)`):

```go
// Cluster-level CodeRabbit (user-scoped)
api.POST("/auth/coderabbit/connect", handlers.ConnectCodeRabbit)
api.GET("/auth/coderabbit/status", handlers.GetCodeRabbitStatus)
api.DELETE("/auth/coderabbit/disconnect", handlers.DisconnectCodeRabbit)
api.POST("/auth/coderabbit/test", handlers.TestCodeRabbitConnection)
```

Add the runtime credential route after the existing credentials routes in the `projectGroup` section (after the MCP credentials line):

```go
projectGroup.GET("/agentic-sessions/:sessionName/credentials/coderabbit", handlers.GetCodeRabbitCredentialsForSession)
```

- [ ] **Step 3: Verify Go code compiles**

Run: `cd components/backend && go build ./...`
Expected: Clean build, no errors.

- [ ] **Step 4: Commit**

```bash
git add components/backend/handlers/runtime_credentials.go components/backend/routes.go
git commit -m "feat(backend): add CodeRabbit runtime credential fetch and route registration"
```

---

### Task 4: Backend — Tests

**Files:**
- Create: `components/backend/handlers/coderabbit_auth_test.go`
- Modify: `components/backend/tests/constants/labels.go`

- [ ] **Step 1: Add test label constant**

In `components/backend/tests/constants/labels.go`, add to the constants block:

```go
LabelCodeRabbitAuth = "coderabbit-auth"
```

- [ ] **Step 2: Create test file**

Create `components/backend/handlers/coderabbit_auth_test.go`. Follow the Jira test patterns — use Ginkgo/Gomega with fake K8s clientset. Test cases:

1. `ConnectCodeRabbit` — stores credentials when valid API key provided
2. `ConnectCodeRabbit` — rejects empty API key
3. `ConnectCodeRabbit` — rejects unauthenticated requests
4. `GetCodeRabbitStatus` — returns `connected: true` when credentials exist
5. `GetCodeRabbitStatus` — returns `connected: false` when no credentials
6. `DisconnectCodeRabbit` — removes credentials
7. `DisconnectCodeRabbit` — succeeds when no credentials exist (idempotent)

Note: Mock the `ValidateCodeRabbitAPIKey` call — do not make real HTTP requests in tests. Use `httptest.NewServer` to stub the CodeRabbit health endpoint.

Reference the existing test setup in `components/backend/handlers/` for how other handler tests configure the fake K8s client and Gin test context.

- [ ] **Step 3: Run tests**

Run: `cd components/backend && go test ./handlers/ -run CodeRabbit -v`
Expected: All tests pass.

- [ ] **Step 4: Commit**

```bash
git add components/backend/handlers/coderabbit_auth_test.go components/backend/tests/constants/labels.go
git commit -m "test(backend): add CodeRabbit auth handler tests"
```

---

### Task 5: Frontend — API Client and React Query Hooks

**Files:**
- Create: `components/frontend/src/services/api/coderabbit-auth.ts`
- Create: `components/frontend/src/services/queries/use-coderabbit.ts`
- Modify: `components/frontend/src/services/api/integrations.ts`

- [ ] **Step 1: Create API client**

Create `components/frontend/src/services/api/coderabbit-auth.ts`:

```typescript
import { apiClient } from './client'

export type CodeRabbitStatus = {
  connected: boolean
  updatedAt?: string
}

export type CodeRabbitConnectRequest = {
  apiKey: string
}

export async function getCodeRabbitStatus(): Promise<CodeRabbitStatus> {
  return apiClient.get<CodeRabbitStatus>('/auth/coderabbit/status')
}

export async function connectCodeRabbit(data: CodeRabbitConnectRequest): Promise<void> {
  await apiClient.post<void, CodeRabbitConnectRequest>('/auth/coderabbit/connect', data)
}

export async function disconnectCodeRabbit(): Promise<void> {
  await apiClient.delete<void>('/auth/coderabbit/disconnect')
}

export async function testCodeRabbitConnection(data: CodeRabbitConnectRequest): Promise<{ valid: boolean; error?: string }> {
  return apiClient.post<{ valid: boolean; error?: string }, CodeRabbitConnectRequest>('/auth/coderabbit/test', data)
}
```

- [ ] **Step 2: Create React Query hooks**

Create `components/frontend/src/services/queries/use-coderabbit.ts`:

```typescript
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import * as coderabbitAuthApi from '../api/coderabbit-auth'

export function useCodeRabbitStatus() {
  return useQuery({
    queryKey: ['coderabbit', 'status'],
    queryFn: () => coderabbitAuthApi.getCodeRabbitStatus(),
  })
}

export function useConnectCodeRabbit() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: coderabbitAuthApi.connectCodeRabbit,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['coderabbit', 'status'] })
    },
  })
}

export function useDisconnectCodeRabbit() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: coderabbitAuthApi.disconnectCodeRabbit,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['coderabbit', 'status'] })
    },
  })
}
```

- [ ] **Step 3: Add CodeRabbit to IntegrationsStatus type**

In `components/frontend/src/services/api/integrations.ts`, add to the `IntegrationsStatus` type:

```typescript
coderabbit?: {
  connected: boolean
  updatedAt?: string
  valid?: boolean
}
```

- [ ] **Step 4: Verify frontend builds**

Run: `cd components/frontend && npx tsc --noEmit`
Expected: No type errors.

- [ ] **Step 5: Commit**

```bash
git add components/frontend/src/services/api/coderabbit-auth.ts components/frontend/src/services/queries/use-coderabbit.ts components/frontend/src/services/api/integrations.ts
git commit -m "feat(frontend): add CodeRabbit API client and React Query hooks"
```

---

### Task 6: Frontend — Next.js Proxy Routes

**Files:**
- Create: `components/frontend/src/app/api/auth/coderabbit/connect/route.ts`
- Create: `components/frontend/src/app/api/auth/coderabbit/status/route.ts`
- Create: `components/frontend/src/app/api/auth/coderabbit/disconnect/route.ts`
- Create: `components/frontend/src/app/api/auth/coderabbit/test/route.ts`

- [ ] **Step 1: Create connect proxy route**

Create `components/frontend/src/app/api/auth/coderabbit/connect/route.ts`:

```typescript
import { BACKEND_URL } from '@/lib/config'
import { buildForwardHeadersAsync } from '@/lib/auth'

export async function POST(request: Request) {
  const headers = await buildForwardHeadersAsync(request)
  const body = await request.text()

  const resp = await fetch(`${BACKEND_URL}/auth/coderabbit/connect`, {
    method: 'POST',
    headers,
    body,
  })

  const data = await resp.text()
  return new Response(data, { status: resp.status, headers: { 'Content-Type': 'application/json' } })
}
```

- [ ] **Step 2: Create status proxy route**

Create `components/frontend/src/app/api/auth/coderabbit/status/route.ts`:

```typescript
import { BACKEND_URL } from '@/lib/config'
import { buildForwardHeadersAsync } from '@/lib/auth'

export async function GET(request: Request) {
  const headers = await buildForwardHeadersAsync(request)

  const resp = await fetch(`${BACKEND_URL}/auth/coderabbit/status`, {
    method: 'GET',
    headers,
  })

  const data = await resp.text()
  return new Response(data, { status: resp.status, headers: { 'Content-Type': 'application/json' } })
}
```

- [ ] **Step 3: Create disconnect proxy route**

Create `components/frontend/src/app/api/auth/coderabbit/disconnect/route.ts`:

```typescript
import { BACKEND_URL } from '@/lib/config'
import { buildForwardHeadersAsync } from '@/lib/auth'

export async function DELETE(request: Request) {
  const headers = await buildForwardHeadersAsync(request)

  const resp = await fetch(`${BACKEND_URL}/auth/coderabbit/disconnect`, {
    method: 'DELETE',
    headers,
  })

  const data = await resp.text()
  return new Response(data, { status: resp.status, headers: { 'Content-Type': 'application/json' } })
}
```

- [ ] **Step 4: Create test proxy route**

Create `components/frontend/src/app/api/auth/coderabbit/test/route.ts`:

```typescript
import { BACKEND_URL } from '@/lib/config'
import { buildForwardHeadersAsync } from '@/lib/auth'

export async function POST(request: Request) {
  const headers = await buildForwardHeadersAsync(request)
  const body = await request.text()

  const resp = await fetch(`${BACKEND_URL}/auth/coderabbit/test`, {
    method: 'POST',
    headers,
    body,
  })

  const data = await resp.text()
  return new Response(data, { status: resp.status, headers: { 'Content-Type': 'application/json' } })
}
```

- [ ] **Step 5: Commit**

```bash
git add components/frontend/src/app/api/auth/coderabbit/
git commit -m "feat(frontend): add CodeRabbit Next.js proxy routes"
```

---

### Task 7: Frontend — Connection Card Component

**Files:**
- Create: `components/frontend/src/components/coderabbit-connection-card.tsx`

- [ ] **Step 1: Create the CodeRabbit connection card**

Create `components/frontend/src/components/coderabbit-connection-card.tsx`. Follow the `jira-connection-card.tsx` pattern exactly. Key differences from Jira:

- Single field: API key (not URL + email + token)
- No pre-population of fields
- Icon: use a code review icon (e.g., `<Search>` from lucide-react or a custom SVG)
- Description: "Connect to CodeRabbit for AI-powered code review"
- Status shows connected/not connected with `updatedAt`
- Link to CodeRabbit API key docs: `https://app.coderabbit.ai/settings/api-keys`

Props type:

```typescript
type Props = {
  status?: {
    connected: boolean
    updatedAt?: string
    valid?: boolean
  }
  onRefresh?: () => void
}
```

The card should:
1. Show a connect button when not connected
2. On click, show a form with a single password input for the API key (with show/hide toggle)
3. On submit, call `connectCodeRabbit({ apiKey })`
4. When connected, show Edit and Disconnect buttons
5. Use `useConnectCodeRabbit()` and `useDisconnectCodeRabbit()` hooks
6. Use `toast` from `sonner` for success/error notifications

- [ ] **Step 2: Verify frontend builds**

Run: `cd components/frontend && npx tsc --noEmit`
Expected: No type errors.

- [ ] **Step 3: Commit**

```bash
git add components/frontend/src/components/coderabbit-connection-card.tsx
git commit -m "feat(frontend): add CodeRabbit connection card component"
```

---

### Task 8: Frontend — Wire Up Integration Pages

**Files:**
- Modify: `components/frontend/src/app/integrations/IntegrationsClient.tsx`
- Modify: `components/frontend/src/app/projects/[name]/sessions/[sessionName]/components/settings/integrations-panel.tsx`

- [ ] **Step 1: Add CodeRabbit card to integrations page**

In `components/frontend/src/app/integrations/IntegrationsClient.tsx`:

1. Add import: `import { CodeRabbitConnectionCard } from '@/components/coderabbit-connection-card'`
2. Add the card inside the grid div, after `<JiraConnectionCard>`:

```tsx
<CodeRabbitConnectionCard
  status={integrations?.coderabbit}
  onRefresh={refetch}
/>
```

- [ ] **Step 2: Add CodeRabbit to integrations panel**

In `components/frontend/src/app/projects/[name]/sessions/[sessionName]/components/settings/integrations-panel.tsx`:

1. Add after `const jiraConfigured = ...`:

```typescript
const coderabbitConfigured = integrationsStatus?.coderabbit?.connected ?? false;
```

2. Add to the `integrations` array:

```typescript
{
  key: "coderabbit",
  name: "CodeRabbit",
  configured: coderabbitConfigured,
  configuredMessage: "Authenticated. AI code review enabled in sessions.",
},
```

- [ ] **Step 3: Verify frontend builds**

Run: `cd components/frontend && npx tsc --noEmit`
Expected: No type errors.

- [ ] **Step 4: Commit**

```bash
git add components/frontend/src/app/integrations/IntegrationsClient.tsx components/frontend/src/app/projects/[name]/sessions/[sessionName]/components/settings/integrations-panel.tsx
git commit -m "feat(frontend): wire CodeRabbit card into integrations pages"
```

---

### Task 9: Frontend — Tests

**Files:**
- Modify: `components/frontend/src/app/projects/[name]/sessions/[sessionName]/components/settings/__tests__/integrations-panel.test.tsx`

- [ ] **Step 1: Update integrations panel test**

Add CodeRabbit to the mock integrations status data and assert it renders. Find the existing mock for `useIntegrationsStatus` and add:

```typescript
coderabbit: { connected: true, updatedAt: '2026-04-01T00:00:00Z', valid: true }
```

Add a test assertion that "CodeRabbit" text appears in the rendered output.

- [ ] **Step 2: Run frontend tests**

Run: `cd components/frontend && npx vitest run --reporter=verbose -- integrations-panel`
Expected: All tests pass.

- [ ] **Step 3: Commit**

```bash
git add components/frontend/src/app/projects/[name]/sessions/[sessionName]/components/settings/__tests__/integrations-panel.test.tsx
git commit -m "test(frontend): add CodeRabbit to integrations panel tests"
```

---

### Task 10: Runner — CodeRabbit Credential Injection

**Files:**
- Modify: `components/runners/ambient-runner/ambient_runner/platform/auth.py`

- [ ] **Step 1: Add fetch function**

Add after the `fetch_gitlab_credentials` function in `auth.py`:

```python
async def fetch_coderabbit_credentials(context: RunnerContext) -> dict:
    """Fetch CodeRabbit credentials from backend API."""
    data = await _fetch_credential(context, "coderabbit")
    if data.get("apiKey"):
        logger.info("Using CodeRabbit credentials from backend")
    return data
```

- [ ] **Step 2: Add to populate_runtime_credentials**

In the `populate_runtime_credentials` function, add after the GitLab credentials block (before the GitHub credentials block that sets git identity):

```python
# CodeRabbit credentials
try:
    coderabbit_creds = await fetch_coderabbit_credentials(context)
    if coderabbit_creds.get("apiKey"):
        os.environ["CODERABBIT_API_KEY"] = coderabbit_creds["apiKey"]
        logger.info("Updated CodeRabbit API key in environment")
except Exception as e:
    logger.warning(f"Failed to refresh CodeRabbit credentials: {e}")
```

- [ ] **Step 3: Run runner tests**

Run: `cd components/runners/ambient-runner && python -m pytest tests/ -v`
Expected: All existing tests pass.

- [ ] **Step 4: Commit**

```bash
git add components/runners/ambient-runner/ambient_runner/platform/auth.py
git commit -m "feat(runner): add CodeRabbit credential injection"
```

---

### Task 11: Final Verification

- [ ] **Step 1: Run backend build and tests**

```bash
cd components/backend && go build ./... && go test ./handlers/ -run CodeRabbit -v
```

- [ ] **Step 2: Run frontend build**

```bash
cd components/frontend && npm run build
```

- [ ] **Step 3: Run frontend tests**

```bash
cd components/frontend && npx vitest run --reporter=verbose
```

- [ ] **Step 4: Run lints**

```bash
cd /path/to/worktree && pre-commit run --all-files
```

- [ ] **Step 5: Final commit if any lint fixes needed**

```bash
git add -A && git commit -m "chore: lint fixes"
```
