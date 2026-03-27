# Feature Specification: Gerrit Integration Connector

**Feature Branch**: `001-gerrit-integration`
**Created**: 2026-03-24
**Status**: Draft
**Input**: User description: "Add a Gerrit integration connector for AmbientCodePlatform, leveraging the existing open-source Gerrit MCP server from the Gerrit community (https://gerrit.googlesource.com/gerrit-mcp-server) to support communities like OpenStack that rely exclusively on Gerrit for code reviews."

## Clarifications

### Session 2026-03-24

- Q: How should users provide their gitcookies credentials in the UI? → A: Paste gitcookies content into a text field (consistent with how other integrations accept tokens).
- Q: Should Gerrit write operations be enabled by default or require opt-in? → A: All operations (read and write) enabled by default, consistent with GitHub/Jira/GitLab integrations.
- Q: When multiple Gerrit instances are connected, should all be available in every session? → A: Yes, all connected instances are automatically available in every session.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Connect a Gerrit Instance (Priority: P1)

A platform user navigates to the Integrations page and connects their Gerrit code review instance by providing their Gerrit server URL and authentication credentials. Once connected, the integration status shows as active and the user's agentic sessions can interact with Gerrit.

**Why this priority**: Without a working connection, no Gerrit functionality is available. This is the foundation for all other Gerrit interactions.

**Independent Test**: Can be fully tested by navigating to Integrations, filling in Gerrit credentials, and verifying the connection status indicator shows "connected." Delivers the value of establishing a verified link to a Gerrit instance.

**Acceptance Scenarios**:

1. **Given** a user is on the Integrations page, **When** they select an authentication method (HTTP credentials or gitcookies), enter a valid Gerrit server URL and the corresponding credentials, and click Connect, **Then** the system validates the credentials against the Gerrit instance and displays a "Connected" status.
2. **Given** a user provides invalid Gerrit credentials, **When** they attempt to connect, **Then** the system displays a clear error message indicating the credentials are invalid.
3. **Given** a user has a connected Gerrit integration, **When** they view the integrations status, **Then** Gerrit appears as an active integration with the server URL displayed.
4. **Given** a user wants to disconnect Gerrit, **When** they click Disconnect, **Then** the integration is removed and future sessions no longer have Gerrit access.

---

### User Story 2 - Use Gerrit Tools in Agentic Sessions (Priority: P1)

A platform user with a connected Gerrit integration creates an agentic session. The session automatically has access to Gerrit MCP tools, allowing the AI agent to query changes, review code, post comments, and manage code reviews on the user's behalf.

**Why this priority**: This is the core value proposition - enabling AI-assisted code review workflows on Gerrit. Without this, the integration has no practical use.

**Independent Test**: Can be tested by creating a session after connecting Gerrit, then asking the agent to query open changes or retrieve a specific change's details. Delivers the value of AI-powered Gerrit interaction.

**Acceptance Scenarios**:

1. **Given** a user has connected Gerrit credentials, **When** they start a new agentic session, **Then** the Gerrit MCP tools are available to the AI agent within that session.
2. **Given** an active session with Gerrit tools, **When** the agent is asked to list open changes for a project, **Then** the agent uses the Gerrit query tool and returns results from the connected Gerrit instance.
3. **Given** an active session with Gerrit tools, **When** the agent is asked to review a specific change, **Then** the agent can retrieve the change details, view file diffs, and read existing comments.
4. **Given** a user has NOT connected Gerrit, **When** they start a session, **Then** Gerrit tools are not available and the agent does not attempt Gerrit operations.

---

### User Story 3 - View Gerrit Integration Status Per Session (Priority: P2)

A platform user viewing an active session can see whether the Gerrit integration is connected and available for that session. This gives visibility into which tools the agent has access to.

**Why this priority**: Provides transparency and troubleshooting capability, but sessions still function without this visibility.

**Independent Test**: Can be tested by viewing the session settings/integrations panel and confirming Gerrit status is accurately reflected. Delivers the value of integration transparency.

**Acceptance Scenarios**:

1. **Given** a user has connected Gerrit and starts a session, **When** they view the session's integrations panel, **Then** Gerrit shows as "Connected" with the server URL.
2. **Given** a user has NOT connected Gerrit, **When** they view the session's integrations panel, **Then** Gerrit shows as "Not connected" with a link to the Integrations page.

---

### User Story 4 - Connect Multiple Gerrit Instances (Priority: P3)

A platform user who contributes to multiple Gerrit-hosted projects (e.g., OpenStack on review.opendev.org and Android on android-review.googlesource.com) can connect multiple Gerrit instances and specify which one to use in their sessions.

**Why this priority**: Valuable for power users contributing to multiple communities, but the majority of users will work with a single Gerrit instance.

**Independent Test**: Can be tested by connecting two different Gerrit servers, starting a session, and directing the agent to query changes from each instance. Delivers the value of multi-community collaboration.

**Acceptance Scenarios**:

1. **Given** a user has connected two Gerrit instances, **When** they start a session, **Then** all connected instances are automatically available and the agent can interact with both without per-session selection.
2. **Given** a user has multiple Gerrit instances, **When** they view the Integrations page, **Then** each instance is listed separately with its own status and disconnect option.

---

### Edge Cases

- What happens when a user's Gerrit credentials expire or are revoked mid-session? The session should gracefully report that the Gerrit tools are unavailable and suggest re-authenticating.
- What happens when the Gerrit server is unreachable? The agent should report connectivity issues without crashing the session.
- What happens when a user connects a Gerrit instance that requires a specific authentication method not supported? The system should display a clear error about unsupported authentication.
- How does the system handle Gerrit instances behind corporate firewalls or VPNs? The system should attempt connection and report clear timeout/unreachable errors.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST allow users to connect a Gerrit instance by providing a server URL and authentication credentials from the Integrations page.
- **FR-002**: System MUST support two authentication methods: (a) HTTP credentials (username and HTTP password/access token generated from the Gerrit instance), and (b) gitcookies content pasted into a text field, which is the authentication method used by many Gerrit-based communities including OpenStack.
- **FR-003**: System MUST validate Gerrit credentials upon connection by making a test request to the Gerrit instance.
- **FR-004**: System MUST store Gerrit credentials securely, scoped to the individual user, following the same credential isolation patterns used by existing integrations.
- **FR-005**: System MUST expose Gerrit MCP tools to agentic sessions when the user has a connected Gerrit integration.
- **FR-006**: System MUST provide Gerrit integration status on both the global Integrations page and per-session integrations panel.
- **FR-007**: System MUST allow users to disconnect a Gerrit integration, removing stored credentials.
- **FR-008**: System MUST support the Gerrit MCP server's full capabilities with both read and write operations enabled by default: querying changes, viewing change details and diffs, posting review comments, managing reviewers, and change lifecycle operations (abandon, WIP, ready for review).
- **FR-009**: System MUST inject Gerrit credentials into the runner environment at session startup so the Gerrit MCP server can authenticate with the Gerrit instance.
- **FR-010**: System MUST support connecting multiple Gerrit instances per user, each identified by a unique name or server URL.

### Key Entities

- **Gerrit Integration**: A user's connection to a Gerrit code review instance, comprising a server URL, authentication credentials, and connection status.
- **Gerrit MCP Server**: The open-source MCP server from the Gerrit community that translates MCP tool calls into Gerrit REST API requests. Bundled as part of the platform's runner environment.
- **Gerrit Credentials**: User-scoped authentication data stored securely and fetched at session runtime. Comprises either (a) server URL, username, and HTTP password/access token, or (b) server URL and gitcookies file content.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can connect a Gerrit instance and see a confirmed "Connected" status within 30 seconds of providing valid credentials.
- **SC-002**: Agentic sessions with a connected Gerrit integration can successfully query changes, retrieve change details, and post review comments.
- **SC-003**: The Gerrit integration follows the same user experience patterns as existing integrations (GitHub, Jira, GitLab), requiring no additional learning curve for users already familiar with the platform.
- **SC-004**: Users can disconnect and reconnect Gerrit integrations without affecting other integrations or active sessions.
- **SC-005**: 95% of Gerrit tool invocations in sessions complete successfully when credentials are valid and the Gerrit server is reachable.

## Assumptions

- The open-source Gerrit MCP server from `gerrit.googlesource.com/gerrit-mcp-server` is suitable for bundling into the platform's runner environment.
- Two authentication methods are supported: HTTP credentials (username + HTTP password/access token) and gitcookies file content. Both are widely used across Gerrit deployments.
- The Gerrit MCP server will be run in STDIO mode within the runner pod, consistent with how other MCP servers (e.g., mcp-atlassian, google-workspace) are launched.
- The platform's existing generic MCP server credential storage pattern (`mcp-server-credentials` secret with `serverName:userID` keying) can be extended to support Gerrit.
- Users are responsible for obtaining their own credentials: either generating an HTTP password from their Gerrit instance's Settings page, or obtaining their gitcookies file content.

## Dependencies

- **Gerrit MCP Server**: The open-source project at `gerrit.googlesource.com/gerrit-mcp-server` must be compatible with the runner's Python environment.
- **Existing Integration Framework**: This feature builds on the platform's existing MCP server integration patterns (backend credential storage, frontend connection cards, runner credential injection).

## Out of Scope

- OAuth or SSO-based authentication for Gerrit (only HTTP credentials and gitcookies are supported in v1).
- Gerrit administrative operations (project creation, access control management, plugin management).
- Hosting or proxying the Gerrit MCP server centrally - it runs within each session's runner pod.
- Git push/fetch operations to Gerrit repositories (this integration covers the code review API, not Git transport).
