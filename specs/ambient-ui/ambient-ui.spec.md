# Ambient UI Specification

## Purpose

The Ambient UI is the platform's agentic SDLC operations dashboard. It serves two primary workflows: **operations monitoring** (high-frequency, low-duration — "what needs my attention?") and **agent authoring** (low-frequency, high-duration — building, testing, and iterating on agent definitions before codifying them for GitOps). The UI is organized around work outcomes (Jira tickets, PRs, reviews, incidents) rather than infrastructure primitives. Sessions and agents are the execution layer — accessible but secondary to the work they produce. It replaces the existing frontend over time but coexists functionally as a separate component during migration.

The Ambient UI interacts exclusively with the ambient-api-server API via the generated TypeScript SDK. It has no dependency on the legacy backend.

## Architecture

### Requirement: Next.js BFF with OIDC Authentication

The Ambient UI SHALL be a Next.js application acting as a Backend-for-Frontend (BFF). The BFF SHALL handle OIDC authentication as a confidential client, manage server-side sessions, and relay JWTs to the ambient-api-server. The browser SHALL never receive a raw JWT.

The BFF SHALL authenticate via Native SSO: OIDC Authorization Code Flow against a Keycloak or Red Hat SSO issuer. The BFF is the confidential client. Dev environments use a local Keycloak deployed in the Kind cluster.

#### Scenario: SSO login flow

- GIVEN a user navigates to the Ambient UI
- WHEN they are not authenticated
- THEN the BFF redirects to the OIDC authorization endpoint
- AND on callback, exchanges the code for tokens and establishes a server-side session
- AND sets an httpOnly, secure, SameSite cookie on the browser

#### Scenario: API request with JWT relay

- GIVEN an authenticated user makes an API request
- WHEN the BFF proxies the request to the ambient-api-server
- THEN the BFF extracts the JWT from the server-side session
- AND forwards it as `Authorization: Bearer <jwt>`

### Requirement: User Identity Endpoint

The BFF SHALL expose a `/api/me` endpoint that returns the authenticated user's identity extracted from JWT claims in the SSO session. The response SHALL include `username`, `name`, `email`, and `initials` (computed from the user's name).

#### Scenario: Authenticated user identity

- GIVEN a user authenticated via SSO
- WHEN the client fetches `/api/me`
- THEN the response includes `authenticated: true`, `username`, `name`, `email`, and `initials`
- AND the claims are extracted from the JWT stored in the server-side session

#### Scenario: Unauthenticated user identity

- GIVEN a user without a valid session
- WHEN the client fetches `/api/me`
- THEN the response includes `authenticated: false`

### Requirement: User Menu

The nav header SHALL display a user avatar/menu in the top-right corner showing the user's initials. Clicking the avatar SHALL open a dropdown menu displaying the user's full name, email, and a "Sign out" action that redirects to `/api/auth/sso/logout`.

#### Scenario: User menu rendering

- GIVEN an authenticated user with name "Dev User"
- WHEN the nav header renders
- THEN a circular avatar with initials "DU" appears in the top-right corner
- AND clicking it opens a dropdown with the user's name, email, and "Sign out" option

#### Scenario: Sign out

- GIVEN the user menu is open
- WHEN the user clicks "Sign out"
- THEN the browser navigates to `/api/auth/sso/logout`
- AND the SSO session is destroyed

### Requirement: Port/Adapter API Layer

The Ambient UI SHALL define domain port interfaces for each API concern. An adapter layer SHALL implement these ports by calling the generated TypeScript SDK. Components SHALL consume ports, never SDK types directly.

The port layer SHALL define canonical domain types that represent the UI's view of each resource. SDK types SHALL NOT leak into React components or hooks.

#### Scenario: Domain port for sessions

- GIVEN the sessions domain port
- WHEN a component calls `listSessions(projectId, filters)`
- THEN the port returns domain-typed `Session` objects
- AND the adapter internally calls the SDK and maps SDK types to domain types

#### Scenario: SDK type isolation

- GIVEN a React component rendering an agent
- WHEN the component reads the agent's data
- THEN the data type is a domain `Agent` type defined in the port layer
- AND no SDK-generated type appears in the component's imports

#### Scenario: Port coverage

- GIVEN the complete ambient-api-server API surface
- WHEN the port layer is fully implemented
- THEN ports exist for: Projects, Agents, Sessions, SessionMessages, SessionEvents, Credentials, RoleBindings, ScheduledSessions, Inbox

### Requirement: Domain-Oriented Observability

The Ambient UI SHALL instrument domain-significant events — not generic HTTP calls. Observability probes SHALL be expressed in domain language.

#### Scenario: Session phase change observed

- GIVEN a session transitions from Running to Failed
- WHEN the UI detects the change via SSE stream
- THEN a domain probe fires: `session.phaseChanged({ sessionId, from: 'Running', to: 'Failed', projectId })`
- AND the probe is available for logging, metrics, and alerting hooks

#### Scenario: Credential rotation observed

- GIVEN a user rotates a credential token
- WHEN the mutation succeeds
- THEN a domain probe fires: `credential.rotated({ credentialId, provider })`

---

## Navigation and Project Scoping

### Requirement: Project as Primary Context

The Ambient UI SHALL scope all operational views to a single active project. The project selector SHALL be the primary navigation pivot, positioned at the top of the sidebar.

#### Scenario: Project selection

- GIVEN a user opens the Ambient UI
- WHEN no project is selected
- THEN the main content area displays a project picker showing all accessible projects
- AND project-scoped sidebar navigation items are disabled

#### Scenario: Project-scoped views

- GIVEN a user selects project "platform"
- WHEN they navigate to Sessions, Agents, Schedules, Work, or Settings
- THEN all data displayed is scoped to the "platform" project
- AND the active project name is visible in a breadcrumb on every view

#### Scenario: Global views

- GIVEN a user navigates to Credentials
- WHEN the Credentials view renders
- THEN it displays credentials across all projects (global scope)
- AND the project selector is visually dimmed to indicate the view is not project-scoped

### Requirement: Sidebar Navigation

The Ambient UI sidebar SHALL contain three groups separated by visual dividers and group labels:

**Operate** (project-scoped, high-frequency monitoring):
- Dashboard (attention queue + active work + recent completions — default landing page)
- Work (aggregated SDLC artifacts: PRs, tickets, MRs, incidents)
- Sessions (session monitoring, route: `/sessions`)
- Schedules (cron management)

**Build** (project-scoped, agent authoring and configuration):
- Agents (agent registry, authoring workbench, test sessions)

**Configure** (cross-cutting):
- Credentials (credential management and project/agent binding)
- Settings (project configuration, permissions, API keys, feature flags)

The Dashboard sidebar item SHALL display a badge count of items requiring human attention. The group labels ("Operate", "Build", "Configure") SHALL be rendered as muted section headers, not clickable items.

#### Scenario: Navigation breadcrumbs

- GIVEN a user is viewing a session detail in the "platform" project
- WHEN the breadcrumb renders
- THEN it displays: `platform > Sessions > session-name`
- AND each segment is clickable to navigate to that level

### Requirement: Keyboard Navigation

The Ambient UI SHALL support keyboard-first navigation for power users.

#### Scenario: Table navigation

- GIVEN a user is viewing the Sessions table
- WHEN they press `j` or `k`
- THEN the selection moves down or up through table rows
- AND `Enter` opens the selected session's detail view

#### Scenario: Global search

- GIVEN a user presses `Ctrl+K` or `Cmd+K`
- WHEN the search overlay opens
- THEN they can search across session names, agent names, and registered annotation values (Jira issue keys, PR numbers)
- AND results are grouped by type and clickable

Global search SHALL be implemented client-side by querying multiple API endpoints (`GET /sessions?search=...`, `GET /projects/{id}/agents?search=...`) and aggregating results. No cross-resource search endpoint exists in the API today.

#### Scenario: Escape to go back

- GIVEN a user is in a detail view, sidebar, or modal
- WHEN they press `Escape`
- THEN the current overlay closes or the view navigates back one level

---

## Dashboard (Default Landing Page)

### Requirement: Attention-First Landing

The Dashboard SHALL be the default landing page when a user selects a project. It SHALL answer three questions in priority order: "What needs my attention?", "What is actively being worked on?", and "What completed recently?"

#### Scenario: Attention banner

- GIVEN sessions exist with `phase: Failed` or annotations `ambient-code.io/review/status: "needs-review"` or `ambient-code.io/agent/needs-input`
- WHEN the Dashboard renders
- THEN an attention banner appears at the top showing counts of items requiring human action
- AND each item is a clickable link to the relevant session or work item
- AND the sidebar "Dashboard" item displays a badge with the total attention count

#### Scenario: Nothing needs attention

- GIVEN no sessions are failed, no reviews are pending, no agents need input
- WHEN the Dashboard renders
- THEN the attention banner is hidden
- AND the sidebar badge is not displayed

#### Scenario: Active work section

- GIVEN sessions with integration annotations (`ambient-code.io/jira/issue`, `ambient-code.io/github/pr`)
- WHEN the Dashboard renders
- THEN it displays active work items as cards, grouped by integration reference
- AND each card shows the work item identifier (Jira key, PR number) as the primary heading
- AND session/agent information appears as secondary metadata within the card

#### Scenario: Sessions without integration annotations

- GIVEN sessions with no integration annotations
- WHEN the Dashboard renders
- THEN they appear as session cards with the session name as the primary identifier
- AND they are visually distinct from work-item-linked cards (lighter weight, no colored border)

#### Scenario: Recent activity feed

- GIVEN completed sessions exist
- WHEN the Dashboard renders
- THEN a compact timeline shows the last N completed work items with reference, outcome, duration, and cost

### Requirement: Needs-Input Annotation

Agents SHALL be able to signal that they are blocked on human input by writing the annotation `ambient-code.io/agent/needs-input`. The value SHALL describe the type of input needed (e.g., `"approval"`, `"clarification"`, `"credentials"`, `"review"`).

Sessions with this annotation SHALL surface in the Dashboard attention banner and SHALL display a distinct, non-muted badge in the Sessions table — at least as prominent as the Phase badge.

#### Scenario: Agent flags need for input

- GIVEN a Running session where the agent writes `ambient-code.io/agent/needs-input: "approval"`
- WHEN the Dashboard renders
- THEN the attention banner includes this session with label "Waiting for approval"
- AND the Sessions table shows an amber "Needs Input" badge on this session's row

---

## Work View (SDLC Artifacts)

### Requirement: Aggregated Work Items

The Work view SHALL aggregate all registered integration annotations (`ambient-code.io/jira/issue`, `ambient-code.io/github/pr`, `ambient-code.io/gitlab/mr`, `ambient-code.io/oncall/incident`) across sessions in the active project.

The view SHALL display work items as first-class objects, with sessions shown as subordinate detail. It SHALL support tabs for each artifact type (Pull Requests, Tickets, Merge Requests, Incidents), with counts on each tab.

#### Scenario: Work view rendering

- GIVEN sessions in a project with various integration annotations
- WHEN the Work view renders
- THEN it displays tabbed tables: Pull Requests, Tickets, Merge Requests, Incidents
- AND each tab label includes a count of items
- AND each row shows the reference, enriched details (if available), linked sessions, agent, and status

#### Scenario: Work item attention grouping

- GIVEN work items in various states
- WHEN the Work view renders
- THEN items are grouped within each tab: "Needs Attention" (needs-review, failed, changes-requested) at top, "In Progress" (linked to running sessions) in middle, "Done" (completed, merged) at bottom — collapsed by default

#### Scenario: Work item status filtering

- GIVEN the Work view with items in various statuses
- WHEN the user selects a status filter
- THEN only matching items are displayed

Status filtering requires enrichment data. When enrichment is unavailable, the status filter SHALL be hidden.

---

## Sessions View (Session Monitoring)

### Requirement: Session Table

The Sessions view SHALL display a table of all sessions in the active project. Each row SHALL show operational data answering: "Does this session need my attention?"

The table SHALL display: Phase (with activity indicator), Work Item (primary integration annotation as chip), Review Status (from `ambient-code.io/review/status`), Session Name, Agent, Duration, Last Activity, Cost, and Chat action.

The Work Item column SHALL show the first matching integration annotation in priority order: Jira issue, then GitHub PR, then GitLab MR, then Gerrit change. Sessions without integration annotations display "—". The Review Status column SHALL render as a colored badge: `needs-review` = amber, `approved` = green, `changes-requested` = red.

Phase SHALL be the single status indicator. There SHALL NOT be a separate "Status" column. When a session is Running and the agent is actively working, the Phase badge SHALL display a pulsing indicator.

Cost is annotation-driven: the UI reads the `ambient-code.io/cost/estimate` annotation value. Sessions without this annotation display "—" in the Cost column.

#### Scenario: Sessions table rendering

- GIVEN a project with 15 sessions in various phases
- WHEN the Sessions view renders
- THEN each session appears as a single-line row
- AND the Phase badge shows: Running (green), Completed (blue), Failed (red), Stopped (gray), Pending (amber)
- AND Running sessions with active agents show a pulsing dot on the Phase badge

#### Scenario: Sessions filtering

- GIVEN the Sessions table is displayed
- WHEN the user types in the search field or selects a phase filter
- THEN the table filters to matching sessions
- AND summary stats update to reflect the filtered set

#### Scenario: Empty fleet

- GIVEN a project with no sessions
- WHEN the Sessions view renders
- THEN a centered empty state is displayed with an explanation and suggested action

### Requirement: Registered Annotation Indicators

Sessions with registered annotations SHALL display compact visual indicators in the Sessions table. Indicators SHALL appear as small, muted icons to the right of the session name — never as inline chips that break the table's horizontal scan line.

Only annotations with registered keys SHALL produce indicators. Unregistered annotations SHALL NOT produce any visual element in the Sessions table or any other operational view.

Full annotation details SHALL be available on hover (as a popover) or in the session detail view.

#### Scenario: Annotation indicators in fleet

- GIVEN a session with annotations `ambient-code.io/jira/issue: "HYPERFLEET-234"` and `ambient-code.io/github/pr: "org/repo#1847"`
- WHEN the Sessions table renders
- THEN the session row shows small muted icons (Jira icon, PR icon) next to the session name
- AND hovering over the session name reveals a popover with full annotation details

#### Scenario: Unregistered annotation ignored

- GIVEN a session with annotation `ambient-code.io/desired-phase: "Running"`
- WHEN the session appears in the Sessions table or any operational view
- THEN no visual element is produced for that annotation

### Requirement: Virtual Folder Tree (ambient-code.io/ui/path)

Sessions with an `ambient-code.io/ui/path` annotation SHALL be organizable into a virtual folder hierarchy. The folder tree SHALL be toggleable — hidden by default, shown when the user activates it.

The annotation value is a forward-slash-delimited string (e.g., `"backend/auth"`). The UI SHALL parse these into a tree structure. This is a flat namespace rendered as a tree — like S3 prefixes.

#### Scenario: Folder tree activation

- GIVEN sessions with `ambient-code.io/ui/path` annotations like `"backend/auth"`, `"backend/testing"`, `"infra/networking"`
- WHEN the user toggles the folder tree
- THEN a tree panel appears showing the parsed hierarchy with session counts per node
- AND clicking a folder filters the Sessions table to sessions with that path prefix

### Requirement: Bulk Session Operations

The Sessions table SHALL support selecting multiple sessions and performing bulk operations.

#### Scenario: Bulk stop

- GIVEN the user selects 3 Running sessions via checkboxes
- WHEN they click "Stop All" in the floating action bar
- THEN a confirmation is shown
- AND on confirm, all 3 sessions are stopped via the API

### Requirement: Session Detail (Workload Inspector)

Clicking a session in the Sessions table SHALL open a detail view with tabbed content. The detail view SHALL include a sticky action bar with session lifecycle controls.

**Action bar:** Stop (when Running), Restart (when terminal), Clone, Export, Delete. Destructive actions (Stop, Delete) SHALL require confirmation.

**Tabs:** Phase, Logs, Resources, Details, Chat.

#### Scenario: Phase tab

- GIVEN a session in Running phase
- WHEN the Phase tab renders
- THEN it displays: compact phase timeline, Conditions table (with semantically correct colors), Pod Events, key metadata (session name, project, agent link, owner, timestamps)
- AND a collapsible Metrics section showing: tool call count, success/failure rate, average duration, message count, wall clock time, SDK restart count

Tool call metrics SHALL be computed client-side from SessionMessages (event types `tool_use` and `tool_result`). The API does not provide pre-computed metrics.

#### Scenario: Conditions table semantic colors

- GIVEN a condition "InactivityTimeout: False"
- WHEN the Conditions table renders
- THEN the condition displays as green (healthy — timeout has NOT fired)
- AND "Ready: True" displays as green (healthy)
- AND "Error: True" displays as red (unhealthy)

Condition colors SHALL reflect semantic health, not literal True/False values. "Problem" conditions (InactivityTimeout, Error) invert the color mapping: True = red (bad), False = green (good).

#### Scenario: Logs tab

- GIVEN a running session with AG-UI events
- WHEN the Logs tab renders
- THEN it displays a structured log view of events: timestamps, event type badges (text, tool call, error, lifecycle), content
- AND filter chips allow filtering by event type
- AND errors are visually prominent

#### Scenario: Resources tab

- GIVEN a session with attached repos and MCP servers
- WHEN the Resources tab renders
- THEN repos show: name, URL, branch, clone status, cloned timestamp
- AND MCP servers show: name, type, status (connected/disconnected/error), tool count, last heartbeat

#### Scenario: Details tab

- GIVEN a session with configuration and annotations
- WHEN the Details tab renders
- THEN it shows a Configuration section (model, temperature, max tokens, timeouts, workflow, env vars as key-value)
- AND an Annotations section with two parts:
  - Registered annotations rendered as rich cards (Jira ticket card, PR card, etc.) with enriched data when available, or raw values as fallback
  - Raw annotations table showing ALL annotations as key-value pairs (the "kubectl describe" view)

#### Scenario: Chat tab (Full AG-UI)

- GIVEN a Running session
- WHEN the Chat tab renders
- THEN it displays a full AG-UI chat interface with messages
- AND user messages are styled distinctly from assistant messages
- AND assistant messages render as Markdown (headings, lists, code blocks, tables, links)
- AND tool calls render as collapsible blocks showing tool name, arguments (when available), and result
- AND the user can send messages via an input field (Enter to send, Shift+Enter for newline)
- AND the agent's current status is displayed as a phase badge near the input
- AND the input is disabled when the session is not in Running phase

Note: the runner currently persists one `assistant` message per turn (final text only). Intermediate assistant text and individual tool call arguments are not yet persisted as separate messages — they arrive via the operational event writer with tool name and result only. Full streaming with intermediate text requires the SSE-Driven Updates requirement.

### Requirement: Draft Message Persistence

The chat input SHALL persist unsent text to `localStorage` as the user types, scoped per session ID. If the user navigates away, refreshes, or is redirected by an auth flow, the draft SHALL be restored when they return to the same session.

Drafts SHALL be cleared when the message is successfully sent. Drafts SHOULD be cleared on explicit logout (`/api/auth/sso/logout`). Drafts older than 48 hours SHOULD be silently discarded on read.

The storage key format SHALL be `ambient-draft:{sessionId}`. The stored value SHALL include the text and a timestamp.

#### Scenario: Draft survives page reload

- GIVEN a user has typed "review the PR" in the chat input for session A
- WHEN the page reloads (browser refresh or auth redirect)
- THEN the chat input for session A is pre-filled with "review the PR"

#### Scenario: Draft cleared on send

- GIVEN a user has a draft for session A
- WHEN they send the message successfully
- THEN the draft for session A is removed from localStorage

#### Scenario: Draft expires after 48 hours

- GIVEN a draft was saved 49 hours ago
- WHEN the user returns to that session
- THEN the draft is discarded and the input is empty

#### Scenario: Drafts independent per session

- GIVEN drafts exist for session A and session B
- WHEN the user views session A
- THEN only session A's draft is restored
- AND session B's draft is unaffected

### Requirement: Persistent Chat Sidebar

The Ambient UI SHALL support popping a session's chat into a persistent sidebar panel that remains visible across page navigation. The sidebar allows the user to monitor and interact with an agent while performing other tasks in the UI.

Any view that displays a session (fleet table row, session detail header, chat tab) SHALL offer a control to open the chat sidebar for that session. Only one chat sidebar MAY be open at a time; opening a different session's chat SHALL replace the current one.

#### Scenario: Open chat sidebar from Chat tab

- GIVEN the user is viewing a session's Chat tab
- WHEN the user clicks the "Pop out" button
- THEN the chat panel moves from the tab content area into a right-edge sidebar
- AND the Chat tab displays a message indicating the chat is open in the sidebar
- AND the sidebar shows the session name and phase badge

#### Scenario: Open chat sidebar from fleet table

- GIVEN the user is viewing the fleet table
- WHEN the user clicks the chat icon on a session row
- THEN the chat sidebar opens for that session
- AND the fleet table remains visible alongside the sidebar

#### Scenario: Sidebar persists across navigation

- GIVEN the chat sidebar is open for session A
- WHEN the user navigates to a different page (fleet list, another session, settings)
- THEN the sidebar remains open and continues displaying session A's chat
- AND new messages continue to appear via polling or SSE

#### Scenario: Replace sidebar session

- GIVEN the chat sidebar is open for session A
- WHEN the user opens the chat sidebar for session B
- THEN the sidebar switches to session B's chat
- AND session A's chat is no longer displayed

#### Scenario: Close sidebar

- GIVEN the chat sidebar is open
- WHEN the user clicks the close button on the sidebar
- THEN the sidebar dismisses
- AND if the user navigates back to the session's Chat tab, it renders inline again

#### Scenario: Sidebar layout

- GIVEN the chat sidebar is open
- THEN it docks to the right edge of the viewport
- AND it is resizable by dragging the left edge
- AND it is collapsible to a narrow strip showing the session name
- AND the main content area shrinks to accommodate the sidebar

### Requirement: Timestamp Toggle

The Sessions table's "Last Activity" column SHALL support toggling between relative time ("12s ago") and absolute time ("10:42:18 AM EST").

#### Scenario: Timestamp format toggle

- GIVEN the Sessions table displays relative timestamps
- WHEN the user clicks the toggle on the column header
- THEN all timestamps switch to absolute format with explicit timezone
- AND the preference persists for the session

---

## Agents View (Authoring Workbench)

The Agents view serves two personas: operators who need a quick registry glance, and agent authors who build, test, and iterate on agent definitions before codifying them for GitOps management via `acpctl apply`.

### Requirement: Agent Registry Table

The Agents view SHALL display a table of agents in the active project. The table SHALL display: Name, Source (prototype/production badge), Model, Owner, Current Session (clickable), Last Active.

Clicking an agent row SHALL navigate to the agent detail page.

The page SHALL include a "+ New Agent" button for creating agents directly in the UI.

#### Scenario: Agent table rendering

- GIVEN a project with 5 agents
- WHEN the Agents view renders
- THEN each agent appears as a row with name, source badge, model, owner, current session link, and last active timestamp
- AND prototype agents display a "Draft" badge
- AND production agents (managed via GitOps) display a "GitOps" badge

#### Scenario: Empty agents state

- GIVEN a project with no agents
- WHEN the Agents view renders
- THEN an empty state is displayed with a "Create Agent" action button

### Requirement: Agent Detail Page

Clicking an agent in the registry table SHALL navigate to a dedicated agent detail page at `/{projectId}/agents/{agentId}`. The detail page SHALL use a tabbed layout with three tabs: Overview, Sessions, Config.

The page header SHALL display the agent name, lifecycle badge (Draft/GitOps), and action buttons: "Run Test Session" and "Export YAML".

#### Scenario: Overview tab (authoring surface)

- GIVEN an agent detail page for a Draft agent
- WHEN the Overview tab renders
- THEN it displays editable fields: prompt (textarea), model (select), repository URL, description
- AND a "Save Changes" action persists edits via the API
- AND registered annotations render with icons and labels

#### Scenario: Overview tab (GitOps-managed agent)

- GIVEN an agent detail page for a GitOps-managed agent
- WHEN the Overview tab renders
- THEN fields are read-only with a banner: "This agent is managed via GitOps. Edits here will not persist."
- AND the "Run Test Session" action remains available

#### Scenario: Sessions tab (test history)

- GIVEN an agent with 10 past sessions
- WHEN the Sessions tab renders
- THEN it displays a session table filtered to sessions for this agent
- AND each row shows phase, name, duration, cost, and created timestamp
- AND clicking a session navigates to the session detail page

#### Scenario: Config tab (YAML export)

- GIVEN an agent detail page
- WHEN the Config tab renders
- THEN it displays a YAML preview of the agent definition (the format consumed by `acpctl apply`)
- AND "Copy to Clipboard" and "Download YAML" actions are available

#### Scenario: Run Test Session

- GIVEN the user clicks "Run Test Session" on an agent detail page
- WHEN the create session sheet opens
- THEN the agent is pre-selected and the form is pre-filled with the agent's model, prompt, and repos
- AND the user can override any field before submitting
- AND on success, the new session appears in the agent's Sessions tab

### Requirement: Agent Lifecycle Badge

Agents SHALL display a lifecycle badge indicating whether they are managed in the UI (prototype/draft) or via GitOps (production). The badge SHALL be derived from the `ambient-code.io/managed-by` annotation:

- `ambient-code.io/managed-by: "gitops"` → "GitOps" badge (muted, with git-branch icon)
- No annotation or any other value → "Draft" badge (default)

#### Scenario: Prototype agent

- GIVEN an agent without `ambient-code.io/managed-by` annotation
- WHEN it appears in the registry table or detail page
- THEN it displays a "Draft" badge
- AND all fields are editable in the Overview tab

#### Scenario: Production agent

- GIVEN an agent with `ambient-code.io/managed-by: "gitops"`
- WHEN it appears in the registry table or detail page
- THEN it displays a "GitOps" badge
- AND fields are read-only in the Overview tab

### Requirement: Agent CRUD

The Agents view SHALL support creating, editing, and deleting agents.

#### Scenario: Create agent

- GIVEN the user clicks "+ New Agent"
- WHEN the creation form opens
- THEN it displays fields: name (required), display name, model, prompt, repository URL, description
- AND submitting creates the agent via the API

#### Scenario: Edit agent

- GIVEN the user edits fields on a Draft agent's Overview tab
- WHEN they click "Save Changes"
- THEN the agent is updated via the API
- AND a success notification confirms the save

#### Scenario: Delete agent

- GIVEN the user clicks "Delete" on an agent
- WHEN confirmation is provided
- THEN the agent is deleted via the API
- AND the user is navigated back to the agents list

---

## Schedules View

### Requirement: Schedule Table

The Schedules view SHALL display cron schedules in the active project.

The table SHALL display: Name, Agent, Schedule (human-readable with explicit timezone), Next Run (absolute datetime with timezone), Last Run, Last Status (phase badge of most recent run), Enabled (toggle with confirmation).

Raw cron expressions SHALL be shown as hover detail, not inline text.

#### Scenario: Human-readable schedule

- GIVEN a schedule with cron `"0 9 * * 1-5"` and timezone `"America/New_York"`
- WHEN the Schedules table renders
- THEN the Schedule column shows: "9:00 AM EST, Weekdays"
- AND hovering reveals the raw cron expression

#### Scenario: Toggle confirmation

- GIVEN a user clicks the Enabled toggle for "nightly-benchmarks"
- WHEN the toggle is clicked
- THEN an inline confirmation appears: "Disable nightly-benchmarks? [Confirm] [Cancel]"
- AND the toggle does not change until confirmed

---

## Credentials View (Global)

### Requirement: Credential Registry

The Credentials view SHALL be global (not project-scoped). Credentials SHALL be grouped by category.

#### Scenario: Category grouping

- GIVEN credentials of various providers
- WHEN the Credentials view renders
- THEN credentials are grouped in collapsible sections by category:
  - Source Control (GitHub, GitLab)
  - Project Management (Jira)
  - Cloud & Infrastructure (Google Cloud, Vertex AI, Kubernetes)

### Requirement: Credential-to-Agent Binding

Credentials SHALL be bindable to **specific agents** within a project OR to **all agents** in a project. This binding is expressed via RoleBindings with `scope=credential`.

The Credentials view SHALL display bindings as compact indicators showing the project name and either "(all agents)" or specific agent names.

To display bindings for a credential, the UI SHALL query `GET /api/ambient/v1/role_bindings` filtered by `credential_id`. The `GET /credentials/{cred_id}/role_bindings` scoped endpoint is planned but not yet implemented; the generic endpoint is the interim path.

Binding a credential to a project requires the user to hold `project:owner` on that project (per the security spec). The UI SHALL only show bindable projects where the user has `project:owner`.

#### Scenario: Bind credential to all agents

- GIVEN the user manages credential "github-pat"
- AND the user holds `project:owner` on project "platform"
- WHEN they check project "platform" and select "All agents"
- THEN a RoleBinding is created with `credential_id=<cred>`, `project_id=<project>`, `agent_id=NULL`
- AND the credential row shows: "platform (all agents)"

#### Scenario: Bind credential to specific agents

- GIVEN the user manages credential "jira-cloud"
- WHEN they check project "platform" and select "Specific agents" → "pr-reviewer", "bug-fixer"
- THEN RoleBindings are created with `credential_id=<cred>`, `project_id=<project>`, `agent_id=<each agent>`
- AND the credential row shows: "platform → pr-reviewer, bug-fixer"

### Requirement: Credential CRUD with Modals

The Credentials view SHALL provide Add and Manage modals for credential lifecycle operations.

The UI SHALL NOT access credential tokens. The `credential:token-reader` role is platform-internal and granted only to runner service accounts. The UI operates with `credential:owner` (CRUD) and `credential:viewer` (metadata read) roles.

#### Scenario: Add credential

- GIVEN the user clicks "+ Add Credential"
- WHEN the modal opens
- THEN it displays: Category dropdown, Provider dropdown (filtered by category), Name field, dynamic provider-specific fields (Token, URL, Email as needed), and agent binding controls
- AND submitting creates the Credential and associated RoleBindings via the API

#### Scenario: Rotate credential

- GIVEN the user clicks "Manage" on an existing credential
- WHEN they click "Rotate Token"
- THEN a confirmation is shown before proceeding
- AND on confirm, the credential is updated via PATCH with the new token

#### Scenario: Delete credential

- GIVEN the user clicks "Delete" in the Manage modal
- WHEN confirmation is provided
- THEN the credential is soft-deleted via the API
- AND associated RoleBindings are removed

---

## Annotation System

### Requirement: Registered Annotation Keys

The Ambient UI SHALL maintain a registry of annotation keys with defined UI behavior. Only registered keys produce visual elements in operational views. Unregistered annotations are invisible in all views except the raw annotations table in the Details tab.

Annotations are general-purpose metadata — agents write arbitrary annotations for their own purposes. The UI does not render unknown annotations. The registry defines which annotations the UI understands and how it renders them.

All registered annotation keys SHALL use the `ambient-code.io/` namespace prefix, consistent with the platform's existing annotation namespace. Integration-specific annotations use path hierarchy under `ambient-code.io/` (e.g., `ambient-code.io/jira/issue`, `ambient-code.io/github/pr`). Platform-internal annotations (e.g., `ambient-code.io/desired-phase`, `ambient-code.io/session-id`) share the same namespace but are not in the UI registry and are therefore invisible in operational views.

**Registered annotation keys and their UI behavior:**

| Key | Example Value | UI Behavior |
|-----|---------------|-------------|
| `ambient-code.io/ui/path` | `"backend/auth"` | Virtual folder tree grouping in Sessions view |
| `ambient-code.io/ui/pinned` | `"true"` | Pin icon next to session name; sorts to top |
| `ambient-code.io/ui/priority` | `"high"` | Colored priority icon (red/amber/gray) left of session name |
| `ambient-code.io/ui/tag` | `"docs"` | Muted tag chip in annotation popover |
| `ambient-code.io/ui/preview-url` | `"https://app.example.com"` | Live preview panel with feedback mode |
| `ambient-code.io/ui/preview-title` | `"SSO Login v2"` | Title for the preview panel |
| `ambient-code.io/jira/issue` | `"HYPERFLEET-234"` | Jira chip (icon, key); enriched tooltip when available |
| `ambient-code.io/jira/epic` | `"HYPERFLEET-100"` | Epic reference chip; used for grouping/filtering |
| `ambient-code.io/github/pr` | `"org/repo#1847"` | PR chip (icon, number); enriched tooltip when available |
| `ambient-code.io/github/repo` | `"org/repo"` | Repository reference |
| `ambient-code.io/github/branch` | `"feat/new-auth"` | Branch reference |
| `ambient-code.io/gitlab/mr` | `"org/repo!423"` | MR chip (icon, number); enriched tooltip when available |
| `ambient-code.io/gerrit/change` | `"change/12345"` | Gerrit change link |
| `ambient-code.io/review/status` | `"needs-review"` | Status badge (amber/green/red). This is external review metadata, distinct from session phase. |
| `ambient-code.io/review/reviewer` | `"@mchen"` | Reviewer reference |
| `ambient-code.io/triggered-by` | `"schedule/nightly"` | Provenance indicator with contextual icon |
| `ambient-code.io/cost/estimate` | `"$4.12"` | Muted cost display in Sessions table |
| `ambient-code.io/oncall/incident` | `"INC-003"` | Red incident chip with alert icon |
| `ambient-code.io/parent-agent` | `"orchestrator"` | Agent delegation reference |
| `ambient-code.io/agent/needs-input` | `"approval"` | Amber attention badge; surfaces in Dashboard attention queue. Values: `approval`, `clarification`, `credentials`, `review` |
| `ambient-code.io/managed-by` | `"gitops"` | Agent lifecycle badge: "GitOps" (managed externally) vs "Draft" (UI-managed prototype) |

#### Scenario: Registered annotation rendered

- GIVEN a session with annotation `ambient-code.io/jira/issue: "HYPERFLEET-234"`
- WHEN the session appears in any view
- THEN the Jira annotation is rendered as a styled chip
- AND the annotation appears in the Details tab both as a rich card and in the raw table

#### Scenario: Unregistered annotation not rendered

- GIVEN a session with annotation `ambient-code.io/desired-phase: "Running"`
- WHEN the session appears in the Sessions table or any operational view
- THEN no visual element is produced for that annotation
- AND the annotation is visible ONLY in the raw annotations table in the Details tab

#### Scenario: Annotation key registration is explicit

- GIVEN an agent writes annotation `ambient-code.io/slack/channel: "#team-platform"`
- WHEN the Ambient UI encounters this annotation
- THEN it produces no visual element (this key is not in the registry)
- AND adding support for it requires a code change to the annotation renderer registry

### Requirement: Annotation Enrichment (Planned)

For registered annotations that reference external resources (Jira issues, GitHub PRs, GitLab MRs), the UI SHOULD display enriched data (issue title, status, assignee, PR checks) when available. Enrichment is a server-side concern — the UI SHALL NOT call external APIs directly.

**Dependency:** Annotation enrichment requires a new ambient-api-server endpoint that resolves annotation references using bound credentials. This endpoint does not exist today. Until it ships, the UI SHALL render raw annotation values as styled, clickable chips linking to the external resource. Enriched tooltips and detail cards SHALL be populated only when the enrichment API is available.

The enrichment endpoint specification is out of scope for this document and SHALL be defined in a separate API spec.

#### Scenario: Enrichment available

- GIVEN a session with annotation `ambient-code.io/jira/issue: "HYPERFLEET-234"`
- AND the enrichment API is available and the project has a Jira credential bound
- WHEN the UI requests enrichment
- THEN the API server returns enriched data (summary, status, assignee, priority)
- AND the UI renders a rich tooltip on the Jira chip

#### Scenario: Enrichment unavailable (graceful degradation)

- GIVEN a session with annotation `ambient-code.io/jira/issue: "HYPERFLEET-234"`
- AND the enrichment API is not available OR the project has no Jira credential bound
- WHEN the UI renders the annotation
- THEN it displays "HYPERFLEET-234" as a styled, clickable chip linking to the Jira instance
- AND no tooltip with enriched details is shown

---

## Issues View

**Note:** The Issues view has been superseded by the Work View (see "Work View (SDLC Artifacts)" section above). The Work view provides the same aggregated integration references with richer grouping by attention-need and tabbed artifact types. The sidebar label is "Work" rather than "Issues" to reflect the broader scope (all SDLC artifacts, not just bug-tracking items).

---

## Live Preview and Visual Feedback

### Requirement: Live Preview Mode

Sessions with an `ambient-code.io/ui/preview-url` annotation SHALL offer a live preview panel. The preview SHALL render the target URL in an iframe within a near-fullscreen overlay.

The preview iframe SHALL be hardened:
- The `sandbox` attribute SHALL be set with minimal permissions (`allow-scripts allow-same-origin allow-forms`). Top-level navigation (`allow-top-navigation`) and popups (`allow-popups`) SHALL NOT be granted.
- The UI SHALL validate the preview URL against a configurable allowlist of trusted host patterns (e.g., `*.apps.rosa.example.com`, `*.apps.cluster.local`). URLs not matching the allowlist SHALL be rejected with an error message instead of rendered.
- A Content-Security-Policy `frame-src` directive SHALL restrict the iframe to the allowlisted hosts.

#### Scenario: Preview mode activation

- GIVEN a session with `ambient-code.io/ui/preview-url: "https://app.example.com"` and `ambient-code.io/ui/preview-title: "SSO Login v2"`
- AND the URL matches the configured preview host allowlist
- WHEN the user clicks "Open Preview" in the session detail
- THEN a near-fullscreen overlay opens with the URL loaded in a sandboxed iframe
- AND the overlay header shows the preview title, device size toggles (Desktop/Tablet/Mobile), and a Comment button

#### Scenario: Preview URL rejected (untrusted host)

- GIVEN a session with `ambient-code.io/ui/preview-url: "https://evil.example.com"`
- AND the URL does not match the configured preview host allowlist
- WHEN the user clicks "Open Preview"
- THEN the preview does not render
- AND an error message is displayed: "Preview URL is not on the trusted hosts allowlist"

#### Scenario: Device size emulation

- GIVEN the preview overlay is open
- WHEN the user selects "Mobile"
- THEN the iframe width constrains to 375px, centered in the preview area

### Requirement: Visual Feedback Mode

The preview panel SHALL support a feedback mode where users can select elements or regions in the previewed app, attach comments, and batch-send feedback to the agent.

#### Scenario: Enter feedback mode

- GIVEN the preview overlay is open
- WHEN the user presses `c` or clicks "Comment"
- THEN the cursor changes to crosshair
- AND hovering over elements in the iframe highlights them with a blue outline
- AND an instruction bar appears: "Click an element or drag to select a region. Press Esc to cancel."

#### Scenario: Element selection and comment

- GIVEN the user is in feedback mode
- WHEN they click an element in the preview
- THEN the element is highlighted
- AND its `outerHTML` is captured
- AND a comment card appears anchored to the element with a textarea and "Add to Batch" button

#### Scenario: Region selection

- GIVEN the user is in feedback mode
- WHEN they click and drag to draw a rectangle
- THEN the selected region is highlighted
- AND a comment card appears with region dimensions and any contained elements

#### Scenario: Batch feedback

- GIVEN the user has added 3 comments to the batch
- WHEN they click "Send All Feedback (3)"
- THEN a confirmation is shown
- AND on confirm, all feedback is sent as a single aggregated message
- AND the message includes: each comment's text, captured HTML, and viewport metadata

### Requirement: Feedback Delivery

Feedback SHALL be delivered to the agent via the appropriate channel based on session state.

#### Scenario: Feedback to running session

- GIVEN the session is in Running phase
- WHEN feedback is sent
- THEN it is posted as a session message via `POST /api/ambient/v1/sessions/{id}/messages`
- AND the agent receives it as a user turn in the active conversation

#### Scenario: Feedback to inactive session

- GIVEN the session is in Completed or Stopped phase
- WHEN feedback is sent
- THEN it is posted to the agent's inbox via `POST /api/ambient/v1/projects/{project_id}/agents/{agent_id}/inbox`
- AND the agent receives it on next start as part of the drained inbox context

### Requirement: Feedback Panel Position

The feedback history panel SHALL be positioned as a right-side panel alongside the preview area, not below it.

#### Scenario: Feedback panel layout

- GIVEN the preview overlay is open
- WHEN the feedback panel renders
- THEN it appears as a fixed-width panel on the right side of the preview area
- AND pending feedback items appear at the top with edit/remove controls
- AND sent feedback appears below with muted styling
- AND the panel is collapsible via a toggle handle

---

## Real-Time Updates

### Requirement: SSE-Driven Updates

The Ambient UI SHALL use Server-Sent Events as the primary mechanism for real-time updates. Polling SHALL be used as a fallback for resources without SSE endpoints.

#### Scenario: Session event streaming

- GIVEN a user is viewing a Running session's Logs or Chat tab
- WHEN the agent produces new events
- THEN the UI receives them via `GET /api/ambient/v1/sessions/{id}/events` SSE stream
- AND renders them in real-time without polling

#### Scenario: Sessions table polling

- GIVEN a user is viewing the Sessions table
- WHEN a session's phase changes
- THEN the UI detects the change via periodic polling of `GET /api/ambient/v1/sessions` (5s interval)
- AND the Sessions table row updates on the next poll cycle

No list-watch endpoint exists for sessions today. Polling is the interim mechanism.

#### Scenario: SSE unavailable (runner unreachable)

- GIVEN a user is viewing a session's Logs or Chat tab
- WHEN the runner pod is unreachable (SSE returns 502)
- THEN the UI falls back to polling `GET /api/ambient/v1/sessions/{id}/messages` for historical messages
- AND displays a status indicator: "Live stream unavailable — showing cached messages"

#### Scenario: Non-streamable resource polling

- GIVEN the user is viewing the Credentials view
- WHEN credential data changes
- THEN the UI detects changes via periodic polling (30s interval)

---

## Settings View

### Requirement: Project Configuration

The Settings view SHALL provide project-scoped configuration management with tabbed sections.

**Tabs:** General (project metadata), Permissions (user/role management), API Keys (key lifecycle), Feature Flags (toggles with confirmation).

#### Scenario: Feature flag toggle confirmation

- GIVEN the user clicks a feature flag toggle
- WHEN the toggle is clicked
- THEN an inline confirmation appears before applying the change

---

## Cross-Cutting Concerns

### Requirement: Empty States

Every list view SHALL display a meaningful empty state when no data exists, including an explanation and suggested action.

### Requirement: Action Confirmation

All destructive or state-changing actions (session stop/delete, credential delete/rotate, schedule enable/disable, feature flag toggle) SHALL require explicit confirmation before executing.

### Requirement: Status Bar

The Ambient UI SHALL display a persistent status bar fixed to the bottom of the viewport. The status bar SHALL be compact (single line) and always visible regardless of scroll position or active view.

The status bar SHALL display:
- **Connection context**: The ambient-api-server URL currently targeted by the BFF
- **Connection status indicator**: A colored dot and label reflecting the ambient-api-server's reachability (moved from the top bar)

#### Scenario: Status bar rendering

- GIVEN the Ambient UI is loaded
- WHEN any view renders
- THEN a compact status bar is visible at the bottom of the viewport
- AND it displays the API server URL (e.g., `https://ambient-api-server:8000`)
- AND it displays a connection status indicator (green dot + "Connected" or red dot + "Disconnected")

#### Scenario: Cluster connected

- GIVEN the ambient-api-server is reachable
- WHEN the status bar renders
- THEN the connection indicator displays a green dot with "Connected" label

#### Scenario: Cluster disconnected

- GIVEN the ambient-api-server becomes unreachable
- WHEN the UI detects connection failure
- THEN the connection indicator changes to a red dot with "Disconnected" label
- AND a pulsing animation draws attention to the status change

### Requirement: Connection Context Switching

The status bar SHALL support switching between the default SSO-authenticated connection and a custom connection with a user-provided URL and bearer token.

The default connection uses the BFF's configured API server URL and the JWT from the user's SSO session (native-sso mode). A custom connection overrides both the URL and the authentication token.

#### Scenario: Default SSO context

- GIVEN the user has authenticated via SSO
- WHEN no custom context is active
- THEN the BFF proxies API requests to the configured API server URL
- AND uses the JWT from the SSO session as the Authorization header
- AND the status bar displays the configured URL with no override indicator

#### Scenario: Enter custom context

- GIVEN the status bar displays the default API server URL
- WHEN the user clicks the URL
- THEN the status bar expands to show two editable fields: URL and Token
- AND the URL field is pre-populated with the current URL
- AND the Token field is empty with placeholder text (e.g., "Bearer token")
- AND pressing Enter on either field confirms the change
- AND pressing Escape cancels and collapses back to the default view

#### Scenario: Custom context applied

- GIVEN the user enters a custom URL and token and confirms
- WHEN the custom context is active
- THEN the BFF proxies all API requests to the custom URL
- AND uses the user-provided token as the Authorization header (instead of the SSO JWT)
- AND the status bar displays the custom URL with a visual override indicator
- AND a "Reset" control is visible to revert to the default context

#### Scenario: Reset to default context

- GIVEN a custom context is active
- WHEN the user clicks the "Reset" control
- THEN the custom URL and token are cleared
- AND the BFF reverts to using the configured API server URL and SSO JWT
- AND the status bar returns to its default appearance

#### Scenario: Custom context with URL only (no token)

- GIVEN the user enters only a custom URL without a token
- WHEN the custom context is applied
- THEN the BFF proxies to the custom URL
- AND uses the SSO session JWT as the Authorization header (if available)
- AND falls back to no Authorization header if no SSO session exists

#### Scenario: Custom context persistence

- GIVEN the user has set a custom context
- WHEN the page is refreshed
- THEN the custom context persists (stored server-side in the BFF session)
- AND the user does not need to re-enter the URL and token

---

## Migration

### URL Routes

All existing routes SHALL remain stable. The sidebar label changes do NOT imply URL path changes:

| Sidebar Label | Route Path | Status |
|---------------|-----------|--------|
| Dashboard | `/{projectId}` (root project route) | New — becomes default landing |
| Work | `/{projectId}/work` | New |
| Sessions | `/{projectId}/sessions` | Existing — unchanged |
| Agents | `/{projectId}/agents` | Existing — unchanged |
| Agents Detail | `/{projectId}/agents/{agentId}` | New — replaces Sheet with page |
| Schedules | `/{projectId}/schedules` | New |
| Credentials | `/credentials` | New (global) |
| Settings | `/{projectId}/settings` | New |

When the Dashboard page ships, the project picker and project selector SHALL navigate to `/{projectId}` instead of `/{projectId}/sessions`. Direct navigation to `/{projectId}/sessions` SHALL continue to work.

### Session Detail Tabs

Tab URL param values SHALL remain stable: `?tab=overview`, `?tab=logs`, `?tab=resources`, `?tab=config`, `?tab=chat`. These names match the current implementation and SHALL NOT change.

### Agent Detail: Sheet to Page Migration

The existing `AgentDetailPanel` Sheet component SHALL be replaced by a full page at `/{projectId}/agents/{agentId}`. The Sheet component MAY be retained as a lightweight preview when clicking agent names in the Sessions table, but the primary agent detail surface is the page.

### Phased Rollout

New sidebar items SHALL be added incrementally as their pages are implemented. Items SHALL NOT appear in the sidebar until their page exists (no disabled "Coming soon" stubs). Recommended order:

1. Dashboard page + sidebar restructure (Operate/Build/Configure groups)
2. Agent detail page (replaces Sheet)
3. Work view
4. Schedules, Credentials, Settings

---

## API Dependencies

This section documents API endpoints and capabilities that this spec depends on but which do not yet exist. These are not requirements of this spec — they are requirements on other specs.

| Dependency | Required By | Status | Interim |
|------------|-------------|--------|---------|
| Annotation enrichment endpoint (resolve `ambient-code.io/jira/issue` etc. against bound credentials) | Annotation enrichment, Issues view status filtering | Not yet specified | Render raw annotation values as clickable chips |
| `GET /credentials/{cred_id}/role_bindings` (scoped query) | Credential binding display | Planned, not implemented | Use generic `GET /role_bindings` filtered by `credential_id` |
| Cross-resource search endpoint | Global search | Not planned | Client-side aggregation across multiple list endpoints |
| Session list-watch endpoint (`GET /sessions?watch=true`) | Sessions real-time phase updates | Not available | Poll `GET /sessions` at 5s interval |
| SSE availability guarantee (runner reachability) | Logs/Chat real-time streaming | Runner returns 502 when unreachable | Fall back to polling `GET /sessions/{id}/messages` |

## Design Decisions

| Decision | Rationale |
|----------|-----------|
| Next.js BFF (not pure SPA) | Secure OIDC confidential client. Tokens never reach the browser. Proven pattern from existing frontend. |
| Port/adapter over SDK (not SDK types directly) | Domain types decouple UI from generated code. SDK regeneration doesn't cascade into component changes. |
| `ambient-code.io/*` annotation namespace | Consistent with the platform's existing annotation namespace. UI-registered keys and platform-internal keys share the same domain; the UI registry determines which are rendered. |
| Annotation registry is a code enum (not dynamic) | Simplicity. Adding a new annotation type is a PR, not a config change. The set of annotations the UI understands should be deliberate and reviewed. |
| Enrichment as graceful degradation | UI ships without enrichment API. Raw annotation values are useful on their own (clickable links). Enriched tooltips are additive. |
| Cost as annotation, not API field | Cost is agent-computed and written as `ambient-code.io/cost/estimate`. No API-level cost computation. |
| Tool metrics computed client-side | The API stores raw SessionMessages. Aggregating tool call stats is a UI concern, not an API concern. |
| SSE for sessions, polling for rest | Sessions have real-time SSE streams. Credentials, schedules, and agents change infrequently — polling is sufficient and simpler. |
| Single interaction pattern per entity | Agent rows: navigate to detail page. Session rows: navigate to detail page. Reduces cognitive load per Krug's "Don't Make Me Think." |
| Chat sidebar is app-level, not tab-level | The sidebar lives in the root layout, not the session detail page. This enables cross-page persistence. State is managed via React context at the dashboard layout level. |
| Feedback delivery is context-dependent | Running session → session message (immediate). Stopped session → agent inbox (queued). Matches the platform's existing message model. |
| Work-centric IA, not infrastructure-centric | The UI is organized around work outcomes (PRs, tickets, reviews) rather than infrastructure primitives (sessions, agents). Sessions and agents are accessible but secondary. Sidebar groups: Operate (Dashboard, Work, Sessions, Schedules), Build (Agents), Configure (Credentials, Settings). |
| Agent detail is a page, not a sheet | The authoring workflow requires editing prompts, comparing test runs, and exporting YAML. A slide-out sheet is too narrow for sustained work. Agent detail mirrors the session detail tabbed-page pattern. |
| Agents as authoring playground | The UI serves as a prototyping workbench for agent definitions. Teams experiment in the UI, then export to YAML for GitOps management via `acpctl apply`. Draft vs GitOps lifecycle badges distinguish prototype from production agents. |
| Progressive disclosure, not mode switching | The operator and agent author share one navigation structure at different depths of engagement. No modal "Operations Mode" vs "Authoring Mode". Group labels (Operate/Build) provide wayfinding without mode complexity. |
| Dashboard as default landing | The most frequent question ("what needs my attention?") should be answered without clicking anything. The Dashboard is the project-level entry point, replacing the session list as the default. |
