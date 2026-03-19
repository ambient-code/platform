# Session Sharing Audit Report

**Date**: 2026-03-19
**Investigator**: Ambient Worker (session-sharing)
**Scope**: Multi-user session access, RBAC enforcement, API key security, and message attribution

---

## Executive Summary

This audit examined how multiple users interact with shared agentic sessions in Ambient Code Platform. We identified **4 key findings** across UI transparency, RBAC enforcement, user experience, and API key security.

**Critical Findings**:
1. ✅ **Workspace-level permissions work as designed** - admin/edit/view roles are enforced via Kubernetes RBAC
2. ⚠️ **No per-session ownership controls** - users with edit/admin roles can stop/delete ANY session in the workspace, not just their own
3. ⚠️ **Creator not visible in UI** - users cannot see who created a session
4. ⚠️ **No message attribution** - messages don't show which user sent them in multi-user scenarios
5. ⚠️ **Shared API keys** - all users in a workspace share the same API keys (by design, but worth documenting)

---

## 1. EXPOSE CREATOR: Session Table UI

### Current State

**Data Model** (CRD):
- `spec.userContext` exists in the AgenticSession CRD ([agenticsessions-crd.yaml:46-60](components/manifests/base/crds/agenticsessions-crd.yaml#L46-L60))
- Contains: `userId`, `displayName`, `groups`
- Populated at session creation time ([sessions.go:814-856](components/backend/handlers/sessions.go#L814-L856))

**Backend API**:
- The `userContext` is stored in the CR spec
- Backend parses it in `parseSpec()` ([sessions.go:152-168](components/backend/handlers/sessions.go#L152-L168))
- Backend returns it in `ListSessions` and `GetSession` responses

**Frontend Types**:
- ❌ `AgenticSessionSpec` type **does NOT include `userContext`** field
  - File: `components/frontend/src/types/agentic-session.ts:51-68`
  - File: `components/frontend/src/types/api/sessions.ts:48-63`
- The `UserContext` type exists ([api/sessions.ts:6-10](components/frontend/src/types/api/sessions.ts#L6-L10)) but is not part of the session spec type

**Frontend UI**:
- Sessions table shows: Name, Status, Model, Created timestamp, Artifacts count ([sessions-section.tsx:261-272](components/frontend/src/components/workspace-sections/sessions-section.tsx#L261-L272))
- ❌ **Creator user is NOT displayed**

### Finding

**The session creator is captured in the backend but NOT exposed in the frontend UI.**

Users cannot see who created a session, which is problematic for shared workspaces where multiple team members create sessions.

### Recommendation

1. Add `userContext?: UserContext` to `AgenticSessionSpec` type in both:
   - `components/frontend/src/types/agentic-session.ts`
   - `components/frontend/src/types/api/sessions.ts`

2. Add a "Creator" column to the sessions table showing `session.spec.userContext?.displayName || session.spec.userContext?.userId`

3. Consider showing creator in the session hover card as well (currently shows model, created time, prompt)

---

## 2. RBAC AUDIT: Permission Enforcement

### Current State

**ClusterRole Definitions**:

```yaml
# ambient-project-admin (manifests/base/rbac/ambient-project-admin-clusterrole.yaml)
- verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]

# ambient-project-edit (manifests/base/rbac/ambient-project-edit-clusterrole.yaml)
- verbs: ["get", "list", "watch", "create", "update", "patch"]

# ambient-project-view (manifests/base/rbac/ambient-project-view-clusterrole.yaml)
- verbs: ["get", "list", "watch"]
```

**Backend RBAC Enforcement**:

| Operation | Handler | RBAC Check | Line |
|-----------|---------|------------|------|
| List Sessions | `ListSessions` | Via `GetK8sClientsForRequest()` (K8s RBAC) | sessions.go:392 |
| Get Session | `GetSession` | Via `GetK8sClientsForRequest()` (K8s RBAC) | sessions.go:910 |
| Create Session | `CreateSession` | Via `GetK8sClientsForRequest()` (K8s RBAC) | sessions.go:570 |
| Update Display Name | `UpdateSessionDisplayName` | ✅ **Explicit** `SelfSubjectAccessReview` | sessions.go:1244-1262 |
| Stop Session | `StopSession` | Via `GetK8sClientsForRequest()` (K8s RBAC) | sessions.go:2315 |
| Delete Session | `DeleteSession` | Via `GetK8sClientsForRequest()` (K8s RBAC) | sessions.go:2045 |
| Send Message | `HandleAGUIRunProxy` | ✅ `checkAccess(reqK8s, ..., "update")` | agui_proxy.go:229 |

**How it Works**:
- All user operations use `GetK8sClientsForRequest(c)` which builds a K8s client **with the user's token** (middleware.go:84-124)
- Kubernetes RBAC enforces permissions based on the user's RoleBindings
- No application-level checks for "you can only operate on sessions YOU created"

### Finding

**RBAC enforcement relies entirely on Kubernetes RBAC, which is workspace-scoped, not session-scoped.**

✅ **Permissions ARE enforced correctly**:
- Users with `view` role can only list/get sessions (read-only)
- Users with `edit` role can create, update, and stop sessions (but not delete)
- Users with `admin` role can delete sessions

⚠️ **No per-session ownership model**:
- A user with `edit` can **stop ANY session** in the workspace (not just their own)
- A user with `admin` can **delete ANY session** in the workspace (not just their own)
- There is no concept of "session owner" enforced by the backend

### Recommendation

**Decision Required**: Is workspace-level permission model acceptable, or do we need per-session ownership?

**If per-session ownership is desired:**

1. Add application-level checks in `StopSession`, `DeleteSession`, `UpdateSessionDisplayName`:
   ```go
   // Get current user ID from token
   currentUserID := c.GetString("userID")

   // Read session's spec.userContext.userId
   sessionCreatorID := ... // from CR spec

   // Allow if: user is creator OR user has admin role
   if currentUserID != sessionCreatorID && !hasAdminRole(reqK8s, project) {
       c.JSON(http.StatusForbidden, gin.H{"error": "Only the session creator or admin can perform this action"})
       return
   }
   ```

2. Add a helper function `hasAdminRole()` that checks if user has delete permissions on agenticsessions

**If workspace-level is acceptable:**
- Document this behavior clearly in user-facing documentation
- Consider adding a confirmation dialog in the UI: "You are about to stop a session created by [name]. Continue?"

---

## 3. MULTI-USER SESSIONS: Message Attribution

### Current State

**Message Flow**:
1. Frontend sends message via `POST /api/projects/:projectName/agentic-sessions/:sessionName/agui/run`
2. Backend proxies to runner pod (agui_proxy.go:218-268)
3. Messages are passed through as `json.RawMessage` without modification

**Message Types** (types/api/sessions.ts):
```typescript
export type UserMessage = {
  type: 'user_message';
  content: ContentBlock | string;
  timestamp: string;
  // ❌ No sender/userId field
}
```

**RBAC for Sending Messages**:
- Requires "update" permission on the session (agui_proxy.go:229)
- Users with `edit` or `admin` role can send messages

### Finding

**Messages do NOT include sender attribution.**

In a multi-user scenario:
- User A creates a session
- User B (with edit role) opens the same session
- User B sends a message
- ❌ **Neither user can see who sent which message**
- The UI doesn't distinguish between "your message" vs "someone else's message"

This creates a confusing experience where:
- You might see messages you didn't write (sent by a coworker)
- You can't tell who asked a question or provided input
- Debugging conversations is difficult ("who told the agent to do X?")

### Recommendation

1. **Add sender attribution to messages**:
   - Update `UserMessage` type to include `senderId?: string` and `senderDisplayName?: string`
   - Backend should inject these fields when proxying messages:
     ```go
     // In HandleAGUIRunProxy, after parsing input:
     userID := c.GetString("userID")
     userName := c.GetString("userName")

     // Inject into message metadata (if AG-UI protocol supports it)
     input.Metadata = map[string]string{
       "senderId": userID,
       "senderDisplayName": userName,
     }
     ```

2. **Update frontend UI to show attribution**:
   - Add sender label to user messages: `"John Doe sent a message"`
   - Highlight "your" messages vs "others'" messages with different styling
   - Add timestamp and sender to message hover card

3. **Consider session activity log**:
   - Track "User X started this session at Y"
   - Track "User A sent a message at B"
   - Display in session timeline/history

---

## 4. KEY SECURITY AUDIT: Shared API Keys

### Current State

**API Key Injection** (operator/internal/handlers/sessions.go:1224-1233):
```go
const runnerSecretsName = "ambient-runner-secrets"

if !vertexEnabled && runnerSecretsName != "" {
    sources = append(sources, corev1.EnvFromSource{
        SecretRef: &corev1.SecretEnvSource{
            LocalObjectReference: corev1.LocalObjectReference{Name: runnerSecretsName},
        },
    })
}
```

**How Keys Work**:
1. Platform admin creates `ambient-runner-secrets` Secret in workspace namespace
2. Secret contains `ANTHROPIC_API_KEY`, `OPENAI_API_KEY`, etc.
3. Operator injects ALL keys from this Secret as environment variables into EVERY runner pod
4. Keys are **workspace-scoped**, not user-scoped

**Security Model**:
- ✅ Keys are **namespace-isolated** (workspace A cannot access workspace B's keys)
- ✅ Keys are **not exposed via API** (backend doesn't return secret values)
- ⚠️ Keys are **shared by all users** in the workspace
- ⚠️ When you share a session (grant edit access), other users run agents **with the same API keys**

### Finding

**API keys are workspace-level shared secrets, not user-specific.**

**Implications**:
1. **Billing/Cost**: All users in a workspace consume API credits from the same account
2. **Attribution**: Anthropic/OpenAI logs will show all requests from the same API key (can't distinguish users)
3. **Access Control**: Granting workspace access = granting API key usage
4. **Key Rotation**: Changing keys affects ALL users and sessions simultaneously

**This is likely the intended design** (workspaces are team-scoped), but it has security/billing implications.

### Recommendation

**Short-term** (Documentation):
1. Document that API keys are workspace-level shared resources
2. Add warning when granting edit/admin access: "This user will be able to create sessions using workspace API keys"
3. Document billing implications in admin guide

**Long-term** (Optional Enhancement):
1. **User-scoped API keys** (alternative model):
   - Allow users to bring their own API keys (stored in cluster-level user secret)
   - When creating a session, use creator's keys instead of workspace keys
   - Requires significant refactoring (key injection, continuation sessions, etc.)

2. **Hybrid model**:
   - Default: workspace keys (current behavior)
   - Optional: "Use my personal API key" checkbox when creating session
   - Store user preference in session spec

3. **Key usage tracking**:
   - Inject `ANTHROPIC_USER_ID` metadata with session creator's user ID
   - Enables per-user attribution in Anthropic logs (even with shared keys)
   - See: https://docs.anthropic.com/en/api/custom-headers

---

## Summary of Findings

| # | Area | Severity | Status |
|---|------|----------|--------|
| 1 | Creator not shown in UI | Medium | ❌ Missing feature |
| 2 | No per-session RBAC | Low-Medium | ⚠️ Design decision needed |
| 3 | No message attribution | Medium | ❌ Confusing UX in multi-user scenarios |
| 4 | Shared API keys | Low | ✅ Working as designed (document) |

**Next Steps**:
1. Decide on per-session ownership model (Issue #2)
2. Implement creator display in UI (Issue #1) - quick win
3. Add message attribution (Issue #3) - improves UX significantly
4. Document API key sharing model (Issue #4)

---

## Related Files

### CRD & Backend
- `components/manifests/base/crds/agenticsessions-crd.yaml` - Session CRD definition
- `components/backend/handlers/sessions.go` - Session CRUD handlers
- `components/backend/handlers/middleware.go` - Auth and token extraction
- `components/backend/handlers/permissions.go` - RBAC role definitions
- `components/backend/websocket/agui_proxy.go` - Message proxying
- `components/manifests/base/rbac/ambient-project-*.yaml` - ClusterRole definitions

### Operator
- `components/operator/internal/handlers/sessions.go` - Pod creation and secret injection

### Frontend
- `components/frontend/src/types/agentic-session.ts` - Session TypeScript types
- `components/frontend/src/types/api/sessions.ts` - API types (UserContext defined here)
- `components/frontend/src/components/workspace-sections/sessions-section.tsx` - Sessions table UI

---

**End of Audit Report**
