# Feature Specification: CodeRabbit Integration

**Feature Branch**: `001-coderabbit-integration`
**Created**: 2026-04-14
**Status**: Draft
**Input**: User description: "CodeRabbit integration for the Ambient Code Platform following the established Jira integration pattern."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Connect CodeRabbit API Key (Priority: P1)

A platform user navigates to the Integrations page and connects their CodeRabbit account by entering their API key. The system validates the key against CodeRabbit's API, stores it securely, and displays a connected status.

**Why this priority**: Without credential storage, no other CodeRabbit functionality works. This is the foundation for runtime injection and pre-commit review.

**Independent Test**: Can be fully tested by visiting /integrations, entering an API key, and verifying the card shows "Connected" with a timestamp. Delivers secure credential management as standalone value.

**Acceptance Scenarios**:

1. **Given** a user is on the Integrations page and not connected to CodeRabbit, **When** they click "Connect CodeRabbit", enter a valid API key, and click "Save Credentials", **Then** the system validates the key, stores it, and the card shows "Connected".
2. **Given** a user enters an invalid API key, **When** they click "Save Credentials", **Then** the system shows an error message and does not store the key.
3. **Given** a user is connected to CodeRabbit, **When** they click "Disconnect", **Then** the stored credentials are removed and the card shows "Not Connected".
4. **Given** a user is connected, **When** they click "Edit", enter a new API key, and save, **Then** the stored credentials are updated.

---

### User Story 2 - CodeRabbit Credentials Injected into Sessions (Priority: P2)

When a session starts, the runner automatically fetches the user's CodeRabbit API key from the backend and injects it as an environment variable. This makes CodeRabbit CLI and pre-commit hooks work automatically inside sessions.

**Why this priority**: Runtime injection is the primary reason users connect CodeRabbit. Without this, connecting the key has no practical effect.

**Independent Test**: Can be tested by connecting a CodeRabbit API key, starting a session, and verifying the key is available in the session environment.

**Acceptance Scenarios**:

1. **Given** a user has connected CodeRabbit credentials, **When** a session starts, **Then** the runner fetches the API key and sets it in the session environment.
2. **Given** a user has not connected CodeRabbit, **When** a session starts, **Then** the runner skips CodeRabbit credential injection without errors.
3. **Given** a session is running as a different user (multi-user run), **When** credentials are fetched, **Then** the system ensures the correct user's credentials are used based on access controls.
4. **Given** a session turn completes, **When** credentials are cleared, **Then** the CodeRabbit API key is removed from the environment.

---

### User Story 3 - CodeRabbit Status in Session Settings (Priority: P3)

When a user views session settings, the integrations panel shows whether CodeRabbit is configured, consistent with how other integrations are displayed.

**Why this priority**: Provides visibility into which integrations are active. Less critical than connecting or runtime injection but important for user confidence.

**Independent Test**: Can be tested by connecting CodeRabbit, navigating to a session's settings panel, and verifying the CodeRabbit row shows a green checkmark.

**Acceptance Scenarios**:

1. **Given** a user has connected CodeRabbit, **When** they view the session settings integrations panel, **Then** CodeRabbit shows with a connected indicator and a descriptive message.
2. **Given** a user has not connected CodeRabbit, **When** they view the panel, **Then** CodeRabbit shows as not configured with a link to set it up.

---

### User Story 4 - Pre-commit CodeRabbit Review (Priority: P3)

A pre-commit hook runs CodeRabbit review on staged changes before each commit. The hook skips gracefully when the CLI is not installed, the API key is not set, or no changes are staged.

**Why this priority**: Enhances the development workflow by catching issues before commit, but is optional. The integration is useful without it.

**Independent Test**: Can be tested by staging a file, running a commit, and verifying the hook either runs a review or skips gracefully.

**Acceptance Scenarios**:

1. **Given** the CodeRabbit CLI is installed and the API key is set, **When** a user commits staged changes, **Then** the hook runs a review and shows the output.
2. **Given** the CodeRabbit CLI is not installed, **When** a user commits, **Then** the hook skips with an informational message and exits successfully.
3. **Given** the API key is not set, **When** a user commits, **Then** the hook skips with an informational message and exits successfully.
4. **Given** CodeRabbit returns a rate limit or network error, **When** the hook runs, **Then** it prints a warning and allows the commit to proceed.

---

### Edge Cases

- What happens when the CodeRabbit API is temporarily unavailable during key validation? The system returns an error indicating it could not validate the key and does not store it.
- What happens when two users store credentials concurrently? The credential storage uses retry logic with conflict detection to handle concurrent updates.
- What happens when the caller token expires mid-session? The runner falls back to the service account token with a user identity header for credential fetch.
- What happens when a user's API key becomes invalid after storage? The system does not proactively validate stored keys. The CodeRabbit CLI will fail at runtime and the user must reconnect.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST allow users to connect a CodeRabbit API key from the Integrations page
- **FR-002**: System MUST validate the API key against CodeRabbit's health endpoint before storing
- **FR-003**: System MUST store the API key securely, scoped per user
- **FR-004**: System MUST allow users to view their CodeRabbit connection status (connected/not connected, last updated timestamp)
- **FR-005**: System MUST allow users to disconnect (delete) their stored CodeRabbit credentials
- **FR-006**: System MUST allow users to update their API key by editing and re-saving
- **FR-007**: System MUST include CodeRabbit status in the unified integrations status endpoint
- **FR-008**: System MUST expose a runtime credential endpoint for session pods to fetch the API key with access control enforcement
- **FR-009**: System MUST inject the CodeRabbit API key into the session environment when credentials are available
- **FR-010**: System MUST clear the CodeRabbit API key from the environment when a session turn completes
- **FR-011**: System MUST provide a pre-commit hook that runs CodeRabbit review on staged changes
- **FR-012**: The pre-commit hook MUST skip gracefully when the CLI binary, API key, or staged changes are absent
- **FR-013**: The pre-commit hook MUST treat rate limit and network errors as non-blocking warnings
- **FR-014**: System MUST display CodeRabbit connection status in the session settings integrations panel

### Key Entities

- **CodeRabbitCredentials**: Represents a user's stored API key. Attributes: userId, apiKey, updatedAt.
- **IntegrationsStatus**: Aggregated status response including CodeRabbit alongside other integrations. CodeRabbit fields: connected (boolean), updatedAt (timestamp), valid (boolean).

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can connect their CodeRabbit API key in under 30 seconds from the Integrations page
- **SC-002**: CodeRabbit credentials are available in session pods within the normal session startup time (no added delay)
- **SC-003**: The pre-commit hook adds less than 5 seconds of overhead when skipping (no CLI or no key)
- **SC-004**: All existing integrations (GitHub, Google, GitLab, Jira) continue to function without regression
- **SC-005**: The CodeRabbit card renders consistently with the existing integration cards in layout and behavior

## Assumptions

- CodeRabbit's health endpoint is the correct validation mechanism and returns 200 for valid keys
- CodeRabbit API keys do not expire and do not require proactive refresh
- The CodeRabbit CLI binary is named `coderabbit` and supports `review --agent --base <branch>`
- The existing Jira integration pattern is the canonical pattern to follow for new integrations
