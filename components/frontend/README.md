## Ambient Agentic Runner — Frontend (Next.js)

Next.js UI for managing Agentic Sessions and Projects. In local development it proxies API calls to the backend and forwards incoming auth/context headers; it does not spoof identities.

### Prerequisites
- Node.js 20+ and npm
- Go 1.24+ (to run the backend locally)
- oc/kubectl configured to your OpenShift/Kubernetes cluster

### Backend (local) quick start
Run the backend locally while targeting your cluster.

1) Install CRDs to your cluster
```bash
oc apply -f ../manifests/crd.yaml
oc apply -f ../manifests/projectsettings-crd.yaml
```

2) Create/label a project namespace (example: my-project)
```bash
oc new-project my-project || oc project my-project
oc label namespace my-project ambient-code.io/managed=true --overwrite
oc annotate namespace my-project \
  ambient-code.io/display-name="My Project" --overwrite
```

3) Start the backend (defaults to port 8080)
```bash
cd ../backend
export KUBECONFIG="$HOME/.kube/config"   # or your kubeconfig path
go run .
# Health: curl http://localhost:8080/health
```

### Frontend (local) quick start

**Recommended: Use integrated local development environment:**
```bash
# From repository root - single command setup
make kind-up
# Access: http://localhost:8080
```

**Alternative: Standalone frontend development:**
```bash
# From this directory, install and run:
npm ci
export BACKEND_URL=http://localhost:8080/api  # Adjust for your backend
npm run dev
# Open http://localhost:3000
```

### Development Commands

```bash
cd components/frontend

# Install dependencies
npm install

# Development server
npm run dev

# Build
npm run build

# Production server
npm start

# Linting
npm run lint
```

**Pre-commit checklist**:
- Run `npm run build` - must pass with 0 errors, 0 warnings
- See `DESIGN_GUIDELINES.md` for comprehensive frontend development standards

### Authentication model

The frontend acts as a BFF (Backend-for-Frontend) OIDC confidential client. Users authenticate via Keycloak, and the frontend stores the OIDC session in an encrypted httpOnly cookie. On each API request, the frontend extracts the JWT from the session and forwards it as `Authorization: Bearer <jwt>` to the backend.

In the Kind dev cluster, Keycloak is deployed automatically with `make kind-up`. Log in with `developer` / `developer`.

Legacy mode (when `SSO_ENABLED` is not set): the frontend falls back to forwarding `X-Forwarded-*` headers from an OAuth proxy sidecar.

### Environment variables
- `BACKEND_URL` (default: `http://localhost:8080/api`) — backend API for server-side routes
- `FEEDBACK_URL` (optional) — feedback link in the masthead
- `GITHUB_APP_SLUG` (required for GitHub integration) — GitHub App slug
- `GITHUB_CALLBACK_URL` (optional) — explicit GitHub OAuth callback URL
- `SSO_ISSUER_URL` — Keycloak OIDC issuer URL (e.g. `http://keycloak-service:8080/realms/ambient-code`)
- `SSO_CLIENT_ID` — OIDC confidential client ID (e.g. `ambient-frontend`)
- `SSO_CLIENT_SECRET` — OIDC client secret
- `SSO_ENABLED` — set to `true` to enable SSO auth (disables OAuth proxy header forwarding)
- `SSO_REDIRECT_URI` — OIDC callback URL (e.g. `http://localhost:11646/api/auth/sso/callback`)
- `SESSION_SECRET` — encryption key for the session cookie (min 32 chars)
- `SSO_PUBLIC_ISSUER_URL` (Kind only) — public Keycloak URL when it differs from `SSO_ISSUER_URL`

Legacy dev helpers (when SSO is off): `OC_USER`, `OC_EMAIL`, `OC_TOKEN`, `ENABLE_OC_WHOAMI=1`

### Verifying requests
Backend directly (requires headers):
```bash
curl -i http://localhost:8080/api/projects/my-project/agentic-sessions \
  -H "X-OpenShift-Project: my-project" \
  -H "X-Forwarded-User: dev" \
  -H "X-Forwarded-Groups: ambient-project:my-project:admin"
```

Through the frontend route (forwards headers to backend):
```bash
curl -i http://localhost:3000/api/projects/my-project/agentic-sessions \
  -H "X-OpenShift-Project: my-project"
```

### Common issues
- 400 “Project is required …”
  - Use path `/api/projects/{project}/…` or include `X-OpenShift-Project`.
- 403 “Project is not managed by Ambient”
  - Ensure namespace is labeled `ambient-code.io/managed=true`.
- Missing auth header
  - In dev, provide `Authorization: Bearer <token>` (or use `OC_TOKEN` / `ENABLE_OC_WHOAMI`).

### Production notes
- Do not spoof identities. Forward real headers from your OAuth/ingress proxy.
- Provide a project selection mechanism and forward it as `X-OpenShift-Project` (or use project path in API URLs).

## RFE Workflows Frontend Implementation

### Components Implemented

#### 🔐 GitHub Integration (T009, T009a)
- **`GitHubConnection.tsx`**: GitHub App installation and fork management
  - OAuth flow for per-user GitHub App installations
  - Fork selection with visual interface
  - Automatic fork creation capability
  - Real-time connection status

#### 📁 Repository Browser (T010)
- **`RepoBrowser.tsx`**: Full repository navigation
  - File tree browsing with breadcrumb navigation
  - File content display with syntax awareness
  - Branch/ref switching support
  - Size formatting and file type detection

#### 📊 Sessions Dashboard (T011)
- **`SessionsDashboard.tsx`**: Live session management
  - Real-time WebSocket connections for session updates
  - Grouped PR display (spec repo + submodule PRs)
  - Live message streaming with partial reassembly
  - Visual status indicators for all session states
  - Multi-runner support (Claude, OpenAI, local execution)

#### 🎯 Main Application
- **`rfe-workflows.tsx`**: Complete RFE workflow interface
  - Workspace creation and management
  - Tabbed interface for different views
  - RBAC integration with access level display
  - Session creation and monitoring

### API Integration
- **Type-safe backend communication** via `apiClient`
- **WebSocket support** for real-time session updates
- **Comprehensive error handling** with user-friendly messages
- **RBAC enforcement** with access level checking

### Key Features
- **Live Session Monitoring**: WebSocket connections with automatic reconnection
- **Multi-repo PR Management**: Handle spec repo and submodule PRs separately
- **GitHub App Integration**: Streamlined per-user installation flow
- **Repository Browsing**: Full file tree navigation with content preview
- **Runner Support**: Claude Code, OpenAI, and local execution runners
- **Access Control**: Role-based permissions (view/edit/admin)

### UI/UX Design
- **Modern Interface**: Tailwind CSS with shadcn/ui components
- **Responsive Design**: Mobile-friendly responsive layout
- **Accessibility**: Full keyboard navigation and screen reader support
- **Real-time Updates**: Live status indicators and message streaming
- **Error Handling**: Comprehensive error states with recovery actions

The frontend provides a complete user interface for the RFE (Request For Enhancement) workflow system, integrating GitHub repositories, AI runners, and real-time collaboration features.

## Testing

### Unit Tests (Vitest)

466 tests across 26 files. Primary coverage metric (~74%).

```bash
# Run all tests
npx vitest run

# With coverage report
npx vitest run --coverage
open coverage/index.html

# Watch mode
npx vitest

# Single file
npx vitest run src/utils/__tests__/export-chat.test.ts
```

**Config**: `vitest.config.ts` — uses jsdom, Istanbul coverage, `@/` path alias.

### Writing Unit Tests

Place tests in `__tests__/` next to the source:

```
src/
  components/
    chat/
      ChatInputBox.tsx
      __tests__/
        ChatInputBox.test.tsx    ← test file here
  hooks/
    use-session-queue.ts
    __tests__/
      use-session-queue.test.ts  ← test file here
```

**Pure function test:**
```typescript
import { describe, it, expect } from 'vitest';
import { convertEventsToMarkdown } from '../export-chat';

describe('convertEventsToMarkdown', () => {
  it('renders text messages', () => {
    const events = [
      { type: 'TEXT_MESSAGE_START', role: 'user' },
      { type: 'TEXT_MESSAGE_CONTENT', delta: 'Hello' },
      { type: 'TEXT_MESSAGE_END' },
    ];
    const md = convertEventsToMarkdown(makeExport(events), makeSession());
    expect(md).toContain('Hello');
  });
});
```

**Component test:**
```typescript
import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { MyComponent } from '../MyComponent';

it('handles click', () => {
  const onClick = vi.fn();
  render(<MyComponent onClick={onClick} />);
  fireEvent.click(screen.getByText('Click me'));
  expect(onClick).toHaveBeenCalled();
});
```

**Hook test:**
```typescript
import { renderHook, act } from '@testing-library/react';

it('updates state', () => {
  const { result } = renderHook(() => useMyHook());
  act(() => result.current.setValue('new'));
  expect(result.current.value).toBe('new');
});
```

**Mocking:**
```typescript
// Module mock
vi.mock('@/services/api/sessions', () => ({
  createSession: vi.fn().mockResolvedValue({ name: 'test' }),
}));

// Function stub
const onSend = vi.fn();

// DOM API spy
vi.spyOn(document, 'createElement').mockReturnValue(mockEl);

// React Query wrapper
const wrapper = ({ children }) => (
  <QueryClientProvider client={new QueryClient()}>
    {children}
  </QueryClientProvider>
);
const { result } = renderHook(() => useMyQuery(), { wrapper });
```
