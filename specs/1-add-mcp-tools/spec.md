# Feature Specification: Add Context7 and DeepWiki MCP Tools + Generic MCP Credential Framework

## Status

| Field       | Value                                  |
|-------------|----------------------------------------|
| Status      | Draft                                  |
| Created     | 2026-03-06                             |
| Jira        | RHOAIENG-52212                         |
| Branch      | 1-add-mcp-tools                        |

## Overview

This feature has two parts:

1. **Immediate**: Add Context7 and DeepWiki as platform-wide MCP documentation tools available to all agentic sessions
2. **Foundational**: Build a generic MCP credential storage and delivery framework so that adding future authenticated MCP servers requires zero bespoke code per integration

Context7 provides up-to-date library and framework documentation. DeepWiki provides structured wiki access to open-source repository knowledge. Neither requires authentication today, but the platform currently has no scalable pattern for MCP server credentials — each integration (Jira, Google, GitHub, GitLab) has bespoke handlers across backend, runner, and frontend. This feature replaces that approach with a generic, reusable framework.

## Problem Statement

**Documentation access**: Agents lack access to authoritative, up-to-date library documentation and open-source project knowledge bases. They rely on training data which may be stale, leading to incorrect API patterns and inability to resolve dependency ambiguities.

**Credential scalability**: Adding a new authenticated MCP server currently requires bespoke changes across 4+ components (backend handler, runner fetch, runner auth check, frontend integration card). This cost grows linearly with each new integration and is a barrier to expanding the platform's tool ecosystem.

## Actors

- **Agent** — Uses documentation tools during sessions; consumes credentials injected for authenticated MCP servers
- **Platform Administrator** — Configures platform-level MCP servers and manages credential storage
- **End User** — Initiates sessions, configures personal credentials for MCP servers via Integrations page
- **Workflow Author** — Benefits from documentation tools being available by default across all workflows

## Functional Requirements

### Part 1: Context7 and DeepWiki

#### FR-1: Context7 Documentation Lookup

The platform must make Context7 available to all agentic sessions. Agents must be able to:

- **FR-1.1**: Resolve a library name to a unique identifier
- **FR-1.2**: Query documentation for a specific library and retrieve relevant content

#### FR-2: DeepWiki Repository Knowledge Access

The platform must make DeepWiki available to all agentic sessions. Agents must be able to:

- **FR-2.1**: Browse the structure of a repository's wiki
- **FR-2.2**: Read the full contents of specific wiki pages
- **FR-2.3**: Ask natural-language questions about a repository
- **FR-2.4**: List repositories that have wiki content available

#### FR-3: Automatic Tool Authorization

All tools from Context7 and DeepWiki must be pre-authorized. No human approval prompts — the session UI does not support interactive tool approval.

#### FR-4: Universal Availability

Both tools available in every session type (coding, bug-fix, RFE, review, etc.) without per-workflow configuration.

#### FR-5: No Disruption to Existing Tools

Adding these tools must not interfere with existing tools (web fetch, Jira, Google Workspace, or built-in platform tools).

### Part 2: Generic MCP Credential Framework

#### FR-6: Generic Credential Storage

The platform must provide a single, generic mechanism to store credentials for any MCP server. Credentials are stored per-user and identified by MCP server name. The storage must:

- **FR-6.1**: Support arbitrary credential field schemas (different servers need different fields — e.g., API key, URL, username/token pair)
- **FR-6.2**: Store credentials securely using the same mechanism as existing integrations
- **FR-6.3**: Support per-user credential isolation (one user's credentials cannot be read by another user)

#### FR-7: Generic Credential CRUD Endpoints

The backend must expose parameterized endpoints for managing MCP server credentials:

- **FR-7.1**: Store (create or update) credentials for a named MCP server
- **FR-7.2**: Check connection status for a named MCP server
- **FR-7.3**: Remove credentials for a named MCP server
- **FR-7.4**: Fetch credentials at runtime (called by runner during session execution, not by users directly)

#### FR-8: Generic Runner Credential Delivery

The runner must automatically fetch and inject credentials for MCP servers that need them:

- **FR-8.1**: For each MCP server in the configuration, attempt to fetch credentials from the generic backend endpoint
- **FR-8.2**: Map fetched credential fields to environment variables so they can be consumed by `.mcp.json` `${VAR}` expansion
- **FR-8.3**: Gracefully handle missing credentials (server either doesn't need auth or user hasn't configured it)

#### FR-9: Generic MCP Auth Status Check

The runner's MCP authentication status check must work generically:

- **FR-9.1**: For servers with credential requirements, check if the required environment variables are populated
- **FR-9.2**: Report connection status for display in the session UI's MCP status panel

#### FR-10: Frontend Credential Management

Users must be able to manage MCP server credentials through the existing Integrations page:

- **FR-10.1**: Display MCP servers that accept credentials
- **FR-10.2**: Provide a form to input credential fields (API key, etc.)
- **FR-10.3**: Show connection status (connected/disconnected)
- **FR-10.4**: Allow disconnecting (removing credentials)

## User Scenarios & Testing

### Scenario 1: Agent Looks Up Library Documentation

1. User starts a coding session
2. Agent resolves a library identifier using Context7
3. Agent queries documentation and incorporates accurate API usage

**Acceptance Criteria**: Agent retrieves documentation without errors or approval prompts.

### Scenario 2: Agent Researches an Upstream Dependency

1. User starts an RFE session
2. Agent browses a repository's wiki structure via DeepWiki
3. Agent reads wiki pages and asks targeted questions
4. Agent uses gathered context to inform the RFE

**Acceptance Criteria**: Agent navigates wiki content and receives answers without approval prompts.

### Scenario 3: Existing Workflows Unaffected

1. User starts a bug-fix session
2. Agent uses existing tools (file read/write, Jira, web fetch)
3. All existing tools continue to function

**Acceptance Criteria**: No regressions in existing tool functionality.

### Scenario 4: Administrator Adds an Authenticated MCP Server

1. Administrator adds a new MCP server entry to `.mcp.json` with an `env` block referencing credential variables
2. User navigates to Integrations page and sees the new server listed
3. User inputs their API key / credentials for the server
4. User starts a session — the agent can use the new server's tools without approval prompts
5. Credentials are fetched fresh at session start (not baked into pod)

**Acceptance Criteria**: Adding a new authenticated MCP server requires only a `.mcp.json` change — no new backend handlers, runner code, or frontend components.

### Scenario 5: User Without Credentials Starts a Session

1. An MCP server in `.mcp.json` requires credentials, but the user hasn't configured them
2. User starts a session
3. The agent's session starts normally — tools from the unconfigured server are unavailable but don't block the session
4. MCP status panel shows the server as "not configured"

**Acceptance Criteria**: Missing credentials don't block sessions or cause errors. Status is clearly communicated.

## Success Criteria

| Criterion                                                                 | Measure                          |
|---------------------------------------------------------------------------|----------------------------------|
| Context7 and DeepWiki accessible in 100% of new sessions                  | Manual or automated test pass    |
| Agents invoke documentation tools without human approval                  | Zero approval prompts observed   |
| Existing tools remain fully functional                                    | Regression test pass             |
| No increase in session startup time beyond 5 seconds                      | Timing comparison before/after   |
| Available across all workflow types without per-workflow config            | Verified on 3+ workflow types    |
| Adding a new authenticated MCP server requires only `.mcp.json` change    | Verified by adding a test entry  |
| Generic credential CRUD works for arbitrary field schemas                 | Tested with 2+ different schemas |
| MCP status panel shows correct auth state for generic servers             | Manual verification              |

## Scope

### In Scope

- Adding Context7 and DeepWiki as platform-level documentation tools
- Generic backend credential CRUD for MCP servers (single storage, parameterized endpoints)
- Generic runner credential fetching and environment variable injection
- Generic MCP authentication status checking
- Frontend integration card for generic MCP server credentials
- Ensuring tool authorization is automatic for all MCP servers

### Out of Scope

- Migrating existing bespoke integrations (Jira, Google, GitHub, GitLab) to the generic framework
- User-defined MCP server configuration through the UI (adding servers, not just credentials)
- Modifying the CRD schema for per-session MCP configuration
- OAuth flow support in the generic framework (only simple credential fields like API keys)

## Dependencies

- **Context7 service availability**: External service must be reachable from cluster
- **DeepWiki service availability**: External service must be reachable from cluster
- **Network egress**: Cluster network policies must allow outbound HTTPS to external services
- **Claude Agent SDK**: Must support HTTP-type MCP transport

## Assumptions

- Context7 and DeepWiki are free, publicly accessible services requiring no authentication today
- The generic credential framework uses K8s Secrets (consistent with the existing pattern)
- Generic credential fields are simple key-value pairs (API keys, tokens, URLs) — not OAuth flows
- The `.mcp.json` `env` block with `${VAR}` expansion is sufficient for credential injection (no new config schema needed)
- Existing bespoke integrations continue to work alongside the generic framework (no migration required)

## Risks

| Risk                                                   | Likelihood | Impact | Mitigation                                              |
|--------------------------------------------------------|------------|--------|---------------------------------------------------------|
| External service downtime blocks documentation lookups  | Medium     | Low    | Agent falls back to training data; session not blocked  |
| Network policies block outbound connections             | Medium     | High   | Document network requirements; verify in staging        |
| K8s Secret size limit (1MB) constrains credential count | Low        | Medium | One secret per server, or partition by server name      |
| Generic framework doesn't cover edge cases of future integrations | Medium | Medium | Keep bespoke path available as fallback; expand framework as needed |
