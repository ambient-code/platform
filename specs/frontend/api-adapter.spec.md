# Frontend API Adapter Specification

## Purpose

The frontend SHALL access all platform APIs through a typed adapter layer. The adapter defines the canonical types that React components consume and the contract that API implementations must satisfy. This boundary enables independent testing of the API contract without a running backend.

## Requirements

### Requirement: Typed Port Interface

The frontend SHALL define a typed port interface for each API domain that declares all operations as functions with typed inputs and outputs. Port interfaces cover three interaction patterns: request/response, streaming, and text content.

#### Scenario: Request/response operation

- GIVEN a port interface for sessions
- WHEN a consumer calls `listSessions(projectName, pagination)`
- THEN the return type is `PaginatedResult<AgenticSession>` with typed fields
- AND the consumer never sees backend-specific response shapes

#### Scenario: Streaming operation

- GIVEN a port interface for session events
- WHEN a consumer calls `connect(projectName, sessionName, options)` where options includes an optional `runId`
- THEN the port returns a handle with: `sendMessage(content, metadata)`, `interrupt()`, `disconnect()`, and a `status` field reflecting connection state (`connecting`, `connected`, `error`, `disconnected`)
- AND event callbacks are provided via the options parameter for each event type
- AND reconnection semantics (backoff, max retries) are an adapter implementation detail, but connection status changes are surfaced through the handle

#### Scenario: Text content operation

- GIVEN a port interface for workspace files
- WHEN a consumer calls `readFile(projectName, sessionName, path)`
- THEN the return type is `string` (text content)
- AND when a consumer calls `writeFile(projectName, sessionName, path, content)`
- THEN the port accepts a `string` input
- AND the consumer never constructs raw `Response` objects or sets `Content-Type` headers

#### Scenario: Port coverage

- GIVEN the complete set of API service files in `src/services/api/`
- WHEN the adapter layer is fully implemented
- THEN every API function is expressible through a port
- AND no React component or React Query hook calls a raw API function or `fetch()` directly

### Requirement: Canonical Frontend Types

The frontend SHALL define canonical types that represent platform resources as the frontend understands them. These types are the contract between the adapter layer and React components.

#### Scenario: AgenticSession type

- GIVEN the canonical `AgenticSession` type
- WHEN a component renders a session
- THEN the type preserves the existing `metadata` / `spec` / `status` structure:
  - `metadata`: `name`, `namespace`, `uid`, `creationTimestamp`, `labels`, `annotations`
  - `spec`: `initialPrompt`, `llmSettings`, `timeout`, `inactivityTimeout`, `displayName`, `project`, `userContext`, `environmentVariables`, `repos`, `activeWorkflow`, `mcpServers`
  - `status`: `phase`, `startTime`, `completionTime`, `lastActivityTime`, `agentStatus`, `stoppedReason`, `reconciledRepos`, `reconciledWorkflow`, `conditions`, `sdkSessionId`, `sdkRestartCount`, `jobName`, `runnerPodName`
  - Top-level: `autoBranch`
- AND all fields use camelCase
- AND nested structures (`llmSettings`, `repos`, `activeWorkflow`) are typed objects, never serialized strings

#### Scenario: Project type

- GIVEN the canonical `Project` type
- WHEN a component renders a project
- THEN the type includes at minimum: `name`, `displayName`, `description`, `status`, `labels`, `annotations`, `creationTimestamp`, `isOpenShift`, `namespace`, `uid`
- AND `labels` and `annotations` are `Record<string, string>`, never serialized strings

#### Scenario: Paginated results

- GIVEN the canonical `PaginatedResult<T>` type
- WHEN a consumer requests a paginated list
- THEN the result includes: `items: T[]`, `totalCount: number`, `hasMore: boolean`, `nextPage: (() => Promise<PaginatedResult<T>>) | undefined`
- AND the adapter internally manages pagination mechanics (the current backend uses numeric offset; the adapter encapsulates this so consumers call `nextPage()` without knowing the mechanism)

#### Scenario: Error type

- GIVEN the canonical `ApiError` type
- WHEN an API operation fails
- THEN the error includes: `error: string`, `code: string | undefined`, `details: Record<string, unknown> | undefined`
- AND the field name is `error` (not `message`), matching the existing `ApiClientError` class

### Requirement: API Domain Coverage

The frontend SHALL define port interfaces for all API domains. Domains are logical groupings of related operations. Multiple domains MAY map to the same underlying API service file — the port structure reflects consumer intent, not file organization.

#### Scenario: Complete domain enumeration

- GIVEN the adapter layer
- WHEN all domains are implemented
- THEN the following port interfaces exist:

| Port | Covers | Pattern | Current service file(s) |
|------|--------|---------|------------------------|
| `sessions` | Session CRUD, start, stop, clone, display name, model switch, export, pod events | Request/response | `sessions.ts` |
| `sessionEvents` | AG-UI event stream, run (send message), interrupt, feedback | Streaming + request/response | `use-agui-stream.ts` (hook, not service) |
| `sessionWorkspace` | File read, write, list, delete; git status, branches, diff, push, abandon, merge status, configure remote, create branch | Text content + request/response | `workspace.ts` |
| `sessionRepos` | Repo status, clone | Request/response | `sessions.ts` (subset) |
| `sessionMcp` | MCP server status, tool invocation | Request/response | `sessions.ts` (subset) |
| `sessionCapabilities` | Runner capabilities query | Request/response | `sessions.ts` (subset) |
| `sessionTasks` | Background task status, output, stop | Request/response | `tasks.ts` |
| `projects` | Project CRUD | Request/response | `projects.ts` |
| `projectAccess` | RBAC, permissions | Request/response | (inline fetch in hook) |
| `scheduledSessions` | Scheduled session CRUD, suspend, resume, trigger, runs | Request/response | `scheduled-sessions.ts` |
| `keys` | API key management | Request/response | `keys.ts` |
| `secrets` | Project and runner secrets | Request/response | `secrets.ts` |
| `models` | LLM model discovery | Request/response | `models.ts` |
| `runnerTypes` | Runner type discovery | Request/response | `runner-types.ts` |
| `workflows` | OOTB workflow listing, metadata | Request/response | `workflows.ts` |
| `auth` | Current user profile | Request/response | `auth.ts` |
| `github` | GitHub app install, PAT, status | Request/response | `github.ts` |
| `gitlab` | GitLab connect, disconnect, status | Request/response | `gitlab-auth.ts` |
| `google` | Google connect, disconnect, status | Request/response | `google-auth.ts` |
| `gerrit` | Gerrit instances, connect, disconnect, test, status | Request/response | `gerrit-auth.ts` |
| `jira` | Jira connect, disconnect, status | Request/response | `jira-auth.ts` |
| `coderabbit` | CodeRabbit connect, disconnect, test, status | Request/response | `coderabbit-auth.ts` |
| `mcpCredentials` | MCP credential management | Request/response | `mcp-credentials.ts` |
| `integrations` | Cross-integration status | Request/response | `integrations.ts` |
| `featureFlags` | Feature flag evaluation (workspace-scoped) | Request/response | `feature-flags-admin.ts` |
| `ldap` | User and group lookup | Request/response | `ldap.ts` |
| `repo` | Repository blob and tree browsing | Text content + request/response | `repo.ts` |
| `cluster` | Cluster info | Request/response | `cluster.ts` |
| `version` | Platform version | Request/response | `version.ts` |
| `config` | Loading tips, UI configuration | Request/response | `config.ts` |

#### Scenario: Third-party SDK exclusion

- GIVEN the Unleash SDK (`@unleash/proxy-client-react`) which uses its own HTTP client
- AND the GitHub Releases client (`github-releases.ts`) which calls `api.github.com` directly
- WHEN these make HTTP requests
- THEN these requests are excluded from the port layer
- AND the port layer covers only operations that proxy through the application's own backend

### Requirement: Adapter Implementation

The frontend SHALL provide an adapter implementation that satisfies the port interface by calling the backend API and transforming responses into canonical types.

#### Scenario: Response transformation

- GIVEN a backend response in its native format
- WHEN the adapter processes the response
- THEN the output conforms to the canonical frontend type
- AND no backend-specific field names, nesting, or serialization formats leak through

#### Scenario: Error normalization

- GIVEN a backend error response
- WHEN the adapter processes the error
- THEN the error is normalized to the canonical `ApiError` type
- AND backend-specific error formats do not leak to consumers

#### Scenario: Auth forwarding

- GIVEN an authenticated user session
- WHEN the adapter makes a backend request
- THEN the user's auth credentials are forwarded transparently via the existing Next.js route handler infrastructure
- AND the adapter does not store, log, or expose tokens
- AND the port contract does not include auth mechanics — auth is transparent infrastructure below the port

### Requirement: Testability

The adapter layer SHALL be testable in isolation without a running backend or Next.js server.

#### Scenario: Port interface testing

- GIVEN the port interface for sessions
- WHEN a test provides a fake adapter implementation
- THEN React Query hooks can be tested against the fake
- AND the test verifies the contract (input types, output types, error types) without network calls

#### Scenario: Adapter unit testing

- GIVEN a recorded backend response
- WHEN the adapter transforms it
- THEN the output matches the canonical type
- AND the test runs without a backend, verifying only the transformation logic

#### Scenario: Contract test

- GIVEN the port interface and the adapter implementation
- WHEN the contract test suite runs against a live backend
- THEN every port function returns data conforming to the canonical type
- AND pagination, error handling, streaming, and text content operations work end-to-end

### Requirement: React Query Integration

React Query hooks SHALL consume port interfaces, not raw API functions.

#### Scenario: Hook uses port

- GIVEN a React Query hook `useSessionsPaginated`
- WHEN the hook fetches data
- THEN it calls the port interface, not a raw `fetch` or API client function
- AND the hook's return type is derived from the canonical types

#### Scenario: Cache invalidation

- GIVEN a mutation hook (e.g., `useCreateSession`)
- WHEN the mutation succeeds
- THEN related query caches are invalidated using typed query keys
- AND the invalidation logic is defined alongside the port, not scattered across components

### Requirement: Incremental Adoption

The adapter layer SHALL be adoptable incrementally, one API domain at a time.

#### Scenario: Mixed state

- GIVEN that sessions have been migrated to the adapter layer
- WHEN projects have not yet been migrated
- THEN session operations use the port interface
- AND project operations continue using the existing direct API client
- AND both patterns coexist without conflict

#### Scenario: No component changes required

- GIVEN a React component consuming `useSessionsPaginated`
- WHEN the sessions API domain is migrated to the adapter layer
- THEN the component's code does not change
- AND only the hook's internal implementation changes (from direct API client to port)

#### Scenario: Legacy type coexistence

- GIVEN the existing `Session` interface in `types/index.ts` (legacy RFE workflow type)
- WHEN the adapter layer is adopted
- THEN the canonical `AgenticSession` type is used for all session adapter operations
- AND the legacy `Session` type is not modified or removed until its consumers are independently migrated
