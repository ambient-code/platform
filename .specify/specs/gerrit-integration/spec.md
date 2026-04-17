# Feature Specification: Gerrit Integration Connector

**Feature Branch**: `001-gerrit-integration`
**Created**: 2026-04-17
**Status**: Draft
**Input**: Add Gerrit as a native integration in the Ambient Code Platform, enabling users to connect one or more Gerrit instances for code review workflows in agentic sessions.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Connect a Gerrit Instance via HTTP Basic Auth (Priority: P1)

A platform user navigates to the Integrations page and connects a Gerrit instance by providing an instance name, the Gerrit server URL, and their HTTP Basic credentials (username + HTTP password). The system validates credentials against the Gerrit server before storing them. The user sees a success indicator and the instance appears in their list of connected Gerrit instances.

**Why this priority**: Core functionality — without connecting, nothing else works. HTTP Basic is the most common Gerrit auth method.

**Independent Test**: Can be fully tested by navigating to Integrations, filling out the Gerrit form with valid credentials, and verifying the instance appears as connected.

**Acceptance Scenarios**:

1. **Given** a user on the Integrations page with no Gerrit instances configured, **When** they fill in instance name "openstack", URL "https://review.opendev.org", username and HTTP token, and click Save, **Then** credentials are validated against the Gerrit server and the instance appears as connected with a green status indicator.
2. **Given** a user providing invalid credentials, **When** they click Save, **Then** the system shows an error that credentials are invalid and does not store them.
3. **Given** a user providing an HTTP (not HTTPS) URL, **When** they click Save, **Then** the system rejects the URL with an SSRF protection error.
4. **Given** a user providing a URL that resolves to a private/loopback IP, **When** they click Save, **Then** the system rejects it.

---

### User Story 2 - Connect a Gerrit Instance via Gitcookies (Priority: P1)

A platform user connects a Gerrit instance using their .gitcookies file content instead of HTTP Basic credentials. The system parses the gitcookies format, extracts the matching cookie for the target hostname (respecting subdomain flags), validates against the Gerrit server, and stores the credentials.

**Why this priority**: Gitcookies is the required auth method for many enterprise Gerrit deployments (e.g., Android, Chromium).

**Independent Test**: Can be tested by pasting gitcookies content and verifying connection succeeds.

**Acceptance Scenarios**:

1. **Given** a user with valid gitcookies content for a Gerrit host, **When** they select "Gitcookies" auth method and paste the content, **Then** the system extracts the matching cookie and validates successfully.
2. **Given** gitcookies content with subdomain flag TRUE, **When** connecting to a subdomain of the cookie's host, **Then** the cookie matches correctly.
3. **Given** a user providing both HTTP Basic fields AND gitcookies content, **When** they submit, **Then** the system rejects the request as mixed credentials.

---

### User Story 3 - Manage Multiple Gerrit Instances (Priority: P1)

A user connects multiple Gerrit instances (e.g., "openstack" and "android") and manages them independently. Each instance has its own credentials and can be connected/disconnected without affecting others. All instances are listed on the Integrations page.

**Why this priority**: Multi-instance is a core design requirement — many users work across multiple Gerrit servers.

**Independent Test**: Connect two instances, verify both appear, disconnect one, verify the other remains.

**Acceptance Scenarios**:

1. **Given** a user with one connected instance "openstack", **When** they add a second instance "android", **Then** both appear in the instances list, sorted alphabetically.
2. **Given** a user with two connected instances, **When** they disconnect "openstack", **Then** "android" remains connected and functional.
3. **Given** a user trying to add an instance with a duplicate name, **When** they submit, **Then** the existing instance credentials are updated (upsert behavior).

---

### User Story 4 - Test Credentials Without Saving (Priority: P2)

A user can test their Gerrit credentials before saving them. This validates the connection without persisting anything.

**Why this priority**: Reduces failed connection attempts and gives users confidence before committing credentials.

**Independent Test**: Click "Test" with valid/invalid credentials and verify the response without any state change.

**Acceptance Scenarios**:

1. **Given** valid credentials, **When** the user clicks Test, **Then** the system reports "Valid" without storing anything.
2. **Given** invalid credentials, **When** the user clicks Test, **Then** the system reports the credentials are invalid.
3. **Given** an unreachable Gerrit server, **When** the user clicks Test, **Then** the system reports a connection error within 15 seconds.

---

### User Story 5 - Gerrit Credentials Available in Agentic Sessions (Priority: P1)

When an agentic session starts, the runner automatically fetches the user's Gerrit credentials and generates the MCP server configuration. The Gerrit MCP server can then interact with all connected Gerrit instances during the session.

**Why this priority**: This is the purpose of the integration — making Gerrit available to agents.

**Independent Test**: Create a session with Gerrit configured, verify the MCP config file is generated with correct instance data.

**Acceptance Scenarios**:

1. **Given** a user with two connected Gerrit instances, **When** an agentic session starts, **Then** the runner generates a config file containing both instances with their auth details.
2. **Given** a user with gitcookies-based instances, **When** the session starts, **Then** a combined .gitcookies file is generated with entries from all gitcookies instances (file permissions 0o600).
3. **Given** the backend is temporarily unavailable, **When** the runner tries to fetch credentials, **Then** it preserves any existing stale config rather than clearing it.
4. **Given** a credential auth failure (PermissionError), **When** the runner handles it, **Then** existing config IS cleared and the failure is recorded.

---

### User Story 6 - View Gerrit Status in Unified Integrations Endpoint (Priority: P2)

The platform's unified integrations status endpoint includes Gerrit instance information, enabling the frontend to show connection status alongside other integrations.

**Why this priority**: Consistency with existing integration status pattern.

**Independent Test**: Call the integrations status endpoint and verify Gerrit instances are included.

**Acceptance Scenarios**:

1. **Given** a user with connected Gerrit instances, **When** the integrations status endpoint is called, **Then** the response includes a "gerrit" key with instance details.
2. **Given** a user with no Gerrit instances, **When** the status endpoint is called, **Then** the response includes "gerrit" with an empty instances array.

---

### Edge Cases

- What happens when the Gerrit server's DNS changes to a private IP after initial validation? (DNS rebinding — blocked by custom transport that re-validates at dial time)
- What happens when two users connect the same Gerrit instance? (Each user has their own Secret — no conflict)
- What happens when the K8s Secret update conflicts due to concurrent writes? (Retry up to 3 times)
- What happens when instance name is 1 character? (Rejected — minimum 2 characters)
- What happens when gitcookies content has no matching entry for the Gerrit host? (Validation fails — no cookie found)

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST support connecting multiple named Gerrit instances per user
- **FR-002**: System MUST support HTTP Basic Auth (username + HTTP token) as an authentication method
- **FR-003**: System MUST support gitcookies (tab-delimited format with subdomain flag logic) as an authentication method
- **FR-004**: System MUST reject credentials that mix fields from both auth methods (discriminated union)
- **FR-005**: System MUST validate credentials against Gerrit's `/a/accounts/self` endpoint before storing
- **FR-006**: System MUST provide a test endpoint that validates credentials without persisting them
- **FR-007**: System MUST enforce HTTPS-only URLs with SSRF protection (private IP blocking, DNS rebinding prevention)
- **FR-008**: System MUST enforce instance naming rules: lowercase alphanumeric + hyphens, 2-63 chars, regex `^[a-z0-9][a-z0-9-]{0,61}[a-z0-9]$`
- **FR-009**: System MUST store credentials in per-user Kubernetes Secrets with conflict retry handling
- **FR-010**: System MUST return instances sorted by name for deterministic API responses
- **FR-011**: System MUST enforce RBAC for session-level credential access (owner or active run user only)
- **FR-012**: System MUST include Gerrit status in the unified integrations status endpoint
- **FR-013**: System MUST generate MCP server configuration at session startup with all connected instances
- **FR-014**: System MUST handle backend failures gracefully in the runner (auth errors clear config, network errors preserve stale config)
- **FR-015**: System MUST never log credential values; use len(token) for presence checks
- **FR-016**: System MUST strip URLs and methods from error messages to prevent information leakage
- **FR-017**: System MUST gate the integration behind a feature flag
- **FR-018**: System MUST provide a frontend card with multi-instance management UI on the Integrations page
- **FR-019**: System MUST generate combined .gitcookies file from all gitcookies-based instances with 0o600 permissions

### Key Entities

- **GerritInstance**: A named connection to a Gerrit server — instanceName, URL, authMethod, credentials, updatedAt timestamp
- **GerritCredentials**: Per-user collection of GerritInstances stored in a Kubernetes Secret
- **GerritMCPConfig**: Generated configuration file consumed by the Gerrit MCP server at session runtime

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can connect a Gerrit instance and have it available in an agentic session within 30 seconds of saving credentials
- **SC-002**: All SSRF attack vectors (private IPs, DNS rebinding, HTTP downgrade) are blocked with appropriate error messages
- **SC-003**: Credential validation completes or times out within 15 seconds
- **SC-004**: Multiple Gerrit instances can be managed independently without cross-instance side effects
- **SC-005**: The integration follows the same patterns as existing integrations (Jira, CodeRabbit) so future integrations can use it as a reference
- **SC-006**: All backend handler paths are covered by Ginkgo v2 tests
- **SC-007**: Frontend build passes with zero errors and zero warnings
