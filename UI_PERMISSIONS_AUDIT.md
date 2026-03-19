# UI Permissions Audit - Role-Based Element Visibility

**Date**: 2026-03-19
**Issue**: UI elements are shown to all users regardless of their RBAC role, even though backend returns 403 errors. Need to proactively hide elements based on user's access level.

**Backend Access Endpoint**: `GET /api/projects/:projectName/access` returns:
```json
{
  "userRole": "view" | "edit" | "admin"
}
```

---

## RBAC Permission Matrix

| Operation | View | Edit | Admin |
|-----------|------|------|-------|
| **Sessions** |
| List sessions | ✅ | ✅ | ✅ |
| View session details | ✅ | ✅ | ✅ |
| Create session | ❌ | ✅ | ✅ |
| Send messages/chat | ❌ | ✅ | ✅ |
| Stop session | ❌ | ✅ | ✅ |
| Resume session | ❌ | ✅ | ✅ |
| Delete session | ❌ | ❌ | ✅ |
| Edit session name | ❌ | ✅ | ✅ |
| Clone session | ❌ | ✅ | ✅ |
| Add repository | ❌ | ✅ | ✅ |
| Upload files | ❌ | ✅ | ✅ |
| **Settings** |
| View settings | ✅ | ✅ | ✅ |
| Edit runner secrets (API keys) | ❌ | ❌ | ✅ |
| Edit integration secrets | ❌ | ❌ | ✅ |
| Manage permissions | ❌ | ❌ | ✅ |
| Manage access keys | ❌ | ❌ | ✅ |

---

## UI Elements to Hide/Disable by Page

### 1. Sessions List Page (`components/frontend/src/components/workspace-sections/sessions-section.tsx`)

**Elements requiring changes:**

| Element | Current State | Should Hide For | Notes |
|---------|---------------|-----------------|-------|
| "New Session" button | Always visible | `view` | Line ~120-130 |
| Delete icon/button (per session) | Phase-based only | `view`, `edit` | In sessions table |
| Stop action (kebab menu) | Phase-based only | `view` | Kebab menu per row |
| All write actions in kebab | Always visible | `view` | |

**Recommended implementation:**
```tsx
const { data: access } = useProjectAccess(projectName);
const canCreate = access?.userRole === 'edit' || access?.userRole === 'admin';
const canDelete = access?.userRole === 'admin';

// Hide "New Session" button for view users
{canCreate && <Button>New Session</Button>}

// Hide delete icon for non-admins
{canDelete && phase === "Stopped" && <Trash2 />}
```

---

### 2. Session Detail Page Header (`session-header.tsx`)

**Elements requiring changes:**

| Element | Current State | Should Hide For | Line |
|---------|---------------|-----------------|------|
| Stop button | Phase-based only | `view` | 282-293, 334-345 |
| Resume button | Phase-based only | `view` | 294-305, 346-357 |
| Edit name menu item | Always visible | `view` | 200-203, 371-374 |
| Clone menu item | Always visible | `view` | 376-385 |
| Delete menu item | Phase + role combined | `view`, `edit` | 226-236, 387-399 |
| Export submenu | Should be visible | All roles ✅ | 142-183 (read-only) |

**Current logic:**
```tsx
const canStop = isRunning || phase === "Creating";
const canResume = phase === "Stopped";
const canDelete = phase === "Completed" || phase === "Failed" || phase === "Stopped";
```

**Should be:**
```tsx
const { data: access } = useProjectAccess(projectName);
const userRole = access?.userRole || 'view';

const canStop = (isRunning || phase === "Creating") && userRole !== 'view';
const canResume = phase === "Stopped" && userRole !== 'view';
const canDelete = (phase === "Completed" || phase === "Failed" || phase === "Stopped") && userRole === 'admin';
const canEdit = userRole !== 'view';
const canClone = userRole !== 'view';
```

---

### 3. Chat Input Box (`components/frontend/src/components/chat/ChatInputBox.tsx`)

**Elements requiring changes:**

| Element | Current State | Should Disable For | Notes |
|---------|---------------|---------------------|-------|
| Text input field | Always enabled | `view` | Main chat input |
| Send button | Phase-based only | `view` | Message send |
| Interrupt button | Phase-based only | `view` | Stop run |
| Upload file button | Always visible | `view` | Attachment upload |
| Add repository button | Always visible | `view` | Via toolbar |

**Recommended:**
```tsx
const { data: access } = useProjectAccess(projectName);
const canInteract = access?.userRole !== 'view';

<Textarea disabled={!canInteract || isSending} />
{canInteract && <Button>Send</Button>}
{canInteract && onInterrupt && <Button>Interrupt</Button>}
```

**View users should see:** Read-only message display with a disabled input showing "You have view-only access to this session"

---

### 4. Session Settings Modal (`session-settings-modal.tsx`)

**Settings Tabs:**

| Tab | View Access | Edit Access | Admin Access |
|-----|-------------|-------------|--------------|
| Session Details | Read-only | Read-only | Read-only |
| MCP Servers | Read-only | Read-only | Read-only |
| Integrations | Read-only | Read-only | Read-only |

**No configuration changes should be editable by anyone** in session-level settings (these are workspace-level configs applied at session creation time).

Settings modal should be **read-only for all users** (current behavior is correct).

---

### 5. Explorer Panel (`components/frontend/src/app/projects/[name]/sessions/[sessionName]/components/explorer/explorer-panel.tsx`)

**Tabs & Actions:**

| Element | Current State | Should Hide For | Notes |
|---------|---------------|-----------------|-------|
| Files tab | Always visible | Keep visible for all | Read-only for `view` |
| Context tab | Always visible | Keep visible for all | Read-only for `view` |
| Add Repository button | Always visible | `view` | In context tab |
| Upload file button | Always visible | `view` | In files toolbar |
| Delete file action | Always visible | `view` | File context menu |
| Edit file action | Always visible | `view` | Opens editor - should disable |

**File/folder context menus should be read-only for view users** (no delete, rename, create).

---

### 6. Workspace Settings Page (`components/frontend/src/app/projects/[name]/permissions/page.tsx`)

**Elements requiring changes:**

| Section | Current State | Should Hide For | Notes |
|---------|---------------|-----------------|-------|
| Permissions tab | Always visible | `view`, `edit` | Admin only |
| Add user/group button | Always visible | `view`, `edit` | Admin only |
| Remove permission icon | Always visible | `view`, `edit` | Admin only |
| Keys tab | Always visible | `view`, `edit` | Admin only |
| Secrets tab | Always visible | `view`, `edit` | Admin only |

**Workspace-level pages** (`/projects/:name/permissions`, `/projects/:name/keys`) should **return 404 or redirect** for non-admin users.

---

### 7. Sessions Section (Workspace Overview)

Currently in `sessions-section.tsx`:

**Actions needing permission gating:**

| Action | Location | Hide For |
|--------|----------|----------|
| New Session button (top right) | Header toolbar | `view` |
| Stop session (table row action) | Inline button | `view` |
| Delete session (table row action) | Inline button | `view`, `edit` |
| Kebab menu (per session) | Table row | Should filter items by role |

---

## Implementation Plan

### Step 1: Create `useProjectAccess` Hook

**File**: `components/frontend/src/services/queries/use-project-access.ts`

```typescript
import { useQuery } from '@tanstack/react-query';
import type { PermissionRole } from '@/types/project';

type ProjectAccess = {
  project: string;
  allowed: boolean;
  reason?: string;
  userRole: PermissionRole; // "view" | "edit" | "admin"
};

export function useProjectAccess(projectName: string) {
  return useQuery<ProjectAccess>({
    queryKey: ['project-access', projectName],
    queryFn: async () => {
      const res = await fetch(`/api/projects/${projectName}/access`);
      if (!res.ok) throw new Error('Failed to fetch access');
      return res.json();
    },
    enabled: !!projectName,
    staleTime: 60000, // Cache for 1 minute
  });
}
```

### Step 2: Update Session Header

**File**: `session-header.tsx`

```diff
+ import { useProjectAccess } from '@/services/queries/use-project-access';

export function SessionHeader({ session, projectName, ... }: SessionHeaderProps) {
+   const { data: access } = useProjectAccess(projectName);
+   const userRole = access?.userRole || 'view';

    const phase = session.status?.phase || "Pending";
    const isRunning = phase === "Running";
-   const canStop = isRunning || phase === "Creating";
-   const canResume = phase === "Stopped";
-   const canDelete = phase === "Completed" || phase === "Failed" || phase === "Stopped";
+   const canStop = (isRunning || phase === "Creating") && userRole !== 'view';
+   const canResume = phase === "Stopped" && userRole !== 'view';
+   const canDelete = (phase === "Completed" || phase === "Failed" || phase === "Stopped") && userRole === 'admin';
+   const canEdit = userRole !== 'view';
+   const canClone = userRole !== 'view';

    // ... in menu items
-   <DropdownMenuItem onClick={() => setEditNameDialogOpen(true)}>
+   {canEdit && (
+     <DropdownMenuItem onClick={() => setEditNameDialogOpen(true)}>
        <Pencil className="w-4 h-4 mr-2" />
        Edit name
      </DropdownMenuItem>
+   )}

-   <CloneSessionDialog ... />
+   {canClone && <CloneSessionDialog ... />}
```

### Step 3: Update Chat Input

**File**: `ChatInputBox.tsx`

```diff
+ import { useProjectAccess } from '@/services/queries/use-project-access';

export const ChatInputBox: React.FC<ChatInputBoxProps> = ({ projectName, ... }) => {
+   const { data: access } = useProjectAccess(projectName);
+   const canInteract = access?.userRole !== 'view';

    return (
      <Textarea
        value={value}
        onChange={(e) => onChange(e.target.value)}
-       disabled={isSending || !sessionPhase || isTerminal}
+       disabled={!canInteract || isSending || !sessionPhase || isTerminal}
+       placeholder={!canInteract ? "You have view-only access to this session" : "Type your message..."}
      />
    );
};
```

### Step 4: Update Sessions List

**File**: `sessions-section.tsx`

```diff
+ import { useProjectAccess } from '@/services/queries/use-project-access';

export const SessionsSection: React.FC<Props> = ({ projectName }) => {
+   const { data: access } = useProjectAccess(projectName);
+   const canCreate = access?.userRole === 'edit' || access?.userRole === 'admin';
+   const canDelete = access?.userRole === 'admin';

-   <Button onClick={() => setCreateDialogOpen(true)}>
+   {canCreate && (
+     <Button onClick={() => setCreateDialogOpen(true)}>
        <Plus className="w-4 h-4 mr-2" />
        New Session
      </Button>
+   )}

    // In table rows
-   {phase === "Stopped" && (
+   {canDelete && phase === "Stopped" && (
      <Button variant="ghost" size="sm" onClick={() => handleDelete(session)}>
        <Trash2 className="w-4 h-4" />
      </Button>
    )}
```

### Step 5: Update Explorer Panel

**File**: `explorer-panel.tsx`

```diff
+ import { useProjectAccess } from '@/services/queries/use-project-access';

export const ExplorerPanel: React.FC<Props> = ({ projectName, sessionName }) => {
+   const { data: access } = useProjectAccess(projectName);
+   const canModify = access?.userRole !== 'view';

-   <Button onClick={onAddRepository}>
+   {canModify && (
+     <Button onClick={onAddRepository}>
        Add Repository
      </Button>
+   )}

-   <Button onClick={onUploadFile}>
+   {canModify && (
+     <Button onClick={onUploadFile}>
        Upload File
      </Button>
+   )}
```

### Step 6: Add Permission Warnings

For view users trying to access the session, show a banner at the top:

```tsx
{access?.userRole === 'view' && (
  <Alert variant="info">
    <Info className="h-4 w-4" />
    <AlertDescription>
      You have read-only access to this session. Contact a workspace admin to request edit permissions.
    </AlertDescription>
  </Alert>
)}
```

---

## Testing Matrix

| Scenario | View | Edit | Admin |
|----------|------|------|-------|
| See New Session button | ❌ | ✅ | ✅ |
| Create session | ❌ | ✅ | ✅ |
| Send chat message | ❌ | ✅ | ✅ |
| Stop running session | ❌ | ✅ | ✅ |
| Resume stopped session | ❌ | ✅ | ✅ |
| Delete session | ❌ | ❌ | ✅ |
| Edit session name | ❌ | ✅ | ✅ |
| Clone session | ❌ | ✅ | ✅ |
| Add repository | ❌ | ✅ | ✅ |
| Upload file | ❌ | ✅ | ✅ |
| View workspace settings | ✅ | ✅ | ✅ |
| Edit workspace secrets | ❌ | ❌ | ✅ |
| Manage permissions | ❌ | ❌ | ✅ |
| Export chat | ✅ | ✅ | ✅ |
| View session details | ✅ | ✅ | ✅ |

---

## Backend Endpoints Reference

All endpoints use `GetK8sClientsForRequest()` which enforces RBAC via user token:

| Endpoint | View | Edit | Admin |
|----------|------|------|-------|
| `GET /agentic-sessions` | ✅ | ✅ | ✅ |
| `POST /agentic-sessions` | ❌ | ✅ | ✅ |
| `DELETE /agentic-sessions/:id` | ❌ | ❌ | ✅ |
| `POST /agui/run` | ❌ | ✅ | ✅ |
| `POST /repos` | ❌ | ✅ | ✅ |
| `POST /agentic-sessions/:id/stop` | ❌ | ✅ | ✅ |
| `PUT /runner-secrets` | ❌ | ❌ | ✅ |
| `POST /permissions` | ❌ | ❌ | ✅ |

Backend already enforces these - **frontend just needs to hide the UI elements**.

---

## Summary of Required Changes

**New file**:
1. `components/frontend/src/services/queries/use-project-access.ts` - Hook to fetch user role

**Files to modify** (7 files):
1. `session-header.tsx` - Hide Stop/Resume/Delete/Edit/Clone based on role
2. `ChatInputBox.tsx` - Disable input + hide actions for view users
3. `sessions-section.tsx` - Hide New Session button for view, Delete for non-admins
4. `explorer-panel.tsx` - Hide Add Repo + Upload File for view users
5. `sessions-sidebar.tsx` - (if it has actions) - same gating
6. `file-viewer.tsx` - Disable edit/delete actions for view users
7. `MessagesTab.tsx` - Pass projectName to ChatInputBox if not already

**Total effort**: ~2-3 hours of implementation + 1 hour testing with different roles.
