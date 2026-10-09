# Kueue Session Admission Specification

## Purpose

This spec defines how Ambient represents and controls session admission when runner capacity is mediated by Kubernetes scheduling and optional Kueue quotas. The user-facing contract is Ambient-native: Projects declare session admission intent, Sessions expose a first-class queued lifecycle phase, and Applications sync project intent through the Ambient API. Kueue objects remain platform scheduler infrastructure.

Scheduler admission queueing is distinct from Inbox queueing. Inbox messages are persistent Agent intent waiting for the next run. Session admission queueing is a runtime state for an already-started Session waiting for runner capacity.

## Requirements

### Requirement: First-Class Queued Session Phase
The system SHALL support `Queued` as a persistent, title-case Session phase representing a Session that has been accepted by Ambient and reconciled to a runner admission Pod, but whose runner has not yet been admitted, scheduled, and made reachable.

#### Scenario: Session Waits For Admission
- GIVEN a Session in `Pending`
- AND the control plane creates or updates the runner admission Pod
- WHEN the Pod is waiting for Kueue admission or Kubernetes scheduling capacity
- THEN the Session phase SHALL be `Queued`
- AND the Session SHALL include a condition explaining the queue or scheduling wait reason when the platform can observe one

#### Scenario: Session Starts After Admission
- GIVEN a Session in `Queued`
- WHEN the runner admission Pod is admitted, scheduled, ready, and the runner is reachable
- THEN the Session phase SHALL become `Running`
- AND `start_time` SHALL be set if it was not already set

#### Scenario: Queued Sessions Are Active
- GIVEN an Agent has a Session in `Queued`
- WHEN a user starts the same Agent again
- THEN the API SHALL return the existing active Session instead of creating a second Session

#### Scenario: Stop Queued Session
- GIVEN a Session in `Queued`
- WHEN a user stops the Session
- THEN the Session SHALL transition through `Stopping`
- AND the control plane SHALL remove the runner admission Pod
- AND the Session SHALL become `Stopped` after the Pod is removed or no longer observable

### Requirement: Session Phase Compatibility
The system SHALL preserve the existing Session phase contract while adding `Queued` to every phase validator, active-phase query, OpenAPI and gRPC schema, SDK phase helper or constant surface, CLI view, frontend view, behavioral phase allowlist, and watch/event consumer.

#### Scenario: Existing Clients See Existing Phases
- GIVEN a client that starts an Agent in an environment without scheduler admission
- WHEN the Session runs normally
- THEN the client MAY observe the existing `Pending`, `Creating`, and `Running` phases
- AND the client SHALL NOT be required to configure Kueue

#### Scenario: New Clients See Queued
- GIVEN a client lists or watches Sessions
- WHEN a Session is waiting for admission
- THEN the API, gRPC watch stream, SDKs, CLI, and frontend SHALL expose `Queued` exactly as the stored Session phase

#### Scenario: Phase Casing
- GIVEN a Session phase appears in an API response, gRPC message, SDK type, CLI output, frontend view, or example
- WHEN the phase is one of the standard phases
- THEN the value SHALL use title-case spelling: `Pending`, `Creating`, `Queued`, `Running`, `Stopping`, `Stopped`, `Completed`, or `Failed`

#### Scenario: Unknown Phase Rejected
- GIVEN a status update request sets `phase`
- WHEN the value is not one of the supported phases, including `Queued`
- THEN the API SHALL reject the request with a validation error

### Requirement: Runner Admission Pod
The system SHALL use one plain Kubernetes Pod as the v1 runner admission workload for each Session. The control plane MAY create supporting resources such as Services, Secrets, ServiceAccounts, RoleBindings, and namespaces, but admission queueing applies to the runner Pod.

#### Scenario: Runner Pod Is The Admission Unit
- GIVEN the control plane provisions a Session
- WHEN it creates the runtime Pod
- THEN it SHALL create exactly one runner admission Pod for that Session
- AND any Kueue admission labels SHALL be placed on that Pod

#### Scenario: Kueue Queue Label
- GIVEN the resolved admission profile has `kueue_enabled: true`
- WHEN the control plane creates the runner admission Pod
- THEN the Pod metadata labels SHALL include `kueue.x-k8s.io/queue-name` with value equal to the Session's resolved `admission_queue`
- AND the `admission_queue` value SHALL be the LocalQueue name resolved from the admission profile

#### Scenario: Kueue Queue Label Omitted
- GIVEN the resolved admission profile has `kueue_enabled: false`
- WHEN the control plane creates the runner admission Pod
- THEN the Pod SHALL NOT include the `kueue.x-k8s.io/queue-name` label

#### Scenario: No Raw Workload API For Tenants
- GIVEN a user or Application manifest declares desired Ambient state
- WHEN it includes raw Kueue `Workload` resources
- THEN Ambient SHALL treat those resources as unsupported scheduler infrastructure
- AND the Session admission contract SHALL remain expressed through Project and Session fields

#### Scenario: Jobs Are Out Of Scope
- GIVEN the platform later wants runner admission to use Kubernetes Jobs or another controller
- WHEN that change alters the admission unit, retry behavior, or preemption behavior
- THEN this spec SHALL be amended before implementation

### Requirement: Project Session Admission Policy
The system SHALL extend Project create, patch, read, and declarative apply surfaces with an optional `session_admission` policy that declares project-level intent for runner admission.

`session_admission` SHALL contain Ambient-owned fields, not raw scheduler object definitions:

```yaml
session_admission:
  profile: standard
```

Omitting `session_admission` or omitting `profile` SHALL inherit the platform default admission profile.

The stored value `null`, an empty object, or a `profile` value of `null` SHALL mean "inherit the platform default." In declarative apply, an omitted `session_admission` field SHALL leave the live field unmanaged and unchanged; `session_admission: null` or `session_admission: {}` SHALL clear the live value back to default inheritance.

#### Scenario: Project Uses Default Profile
- GIVEN a Project manifest does not set `session_admission`
- WHEN the Project is created or applied
- THEN the Project SHALL use the platform default admission profile
- AND existing Project manifests SHALL continue to apply without changes

#### Scenario: Project Selects Admission Profile
- GIVEN a Project manifest sets `session_admission.profile`
- WHEN the Project is created, patched, or applied
- THEN the API SHALL validate that the profile exists and is allowed for the caller
- AND new Sessions in that Project SHALL use the selected profile unless a more specific allowed override is introduced by another spec

#### Scenario: Declarative Apply Clears Admission Profile
- GIVEN an existing Project has a stored `session_admission.profile`
- WHEN `acpctl apply` or Application sync applies a Project manifest with `session_admission: null` or `session_admission: {}`
- THEN the Project SHALL clear the stored policy
- AND future Sessions SHALL inherit the platform default profile

#### Scenario: Invalid Admission Profile
- GIVEN a Project update requests an unknown `session_admission.profile`
- WHEN the API validates the update
- THEN the API SHALL reject the update
- AND no scheduler resources SHALL be changed for that Project

### Requirement: Admission Profile Catalog
The system SHALL have a platform-owned admission profile catalog used consistently by the API server, control plane, CLI, SDKs, and Application sync validation.

Each profile SHALL define:

```yaml
name: standard
default: true
tenant_selectable: true
kueue_enabled: true
cluster_queue: ambient-standard
local_queue: ambient-sessions-standard
runner_start_timeout: 10m
resource_limits:
  cpu: "2"
  memory: 4Gi
```

Exactly one profile SHALL be marked as the platform default. Profile names SHALL be stable API values suitable for Project manifests.

#### Scenario: Default Profile Resolution
- GIVEN a Project inherits the default profile
- WHEN a Session starts in that Project
- THEN the API and control plane SHALL resolve the same default profile from the platform-owned catalog

#### Scenario: Profile Catalog Changes
- GIVEN the platform profile catalog changes
- WHEN new Sessions are started
- THEN new Sessions SHALL use the current profile mapping
- AND already-created Sessions SHALL retain their resolved profile and queue snapshot

#### Scenario: Unknown Profile Rejected
- GIVEN a Project create, patch, declarative apply, or Application sync requests a profile not present in the catalog
- WHEN the API validates the request
- THEN the request SHALL be rejected before scheduler resources are changed

### Requirement: Application Sync Boundary
The system SHALL allow Applications to sync Project `session_admission` policy as part of Project resources, and SHALL NOT allow Applications to sync raw scheduler infrastructure such as Kueue `ClusterQueue`, `LocalQueue`, `ResourceFlavor`, or `Workload` resources.

#### Scenario: Application Syncs Project Admission Intent
- GIVEN an Application renders a Project manifest with `session_admission.profile`
- WHEN the Application syncs
- THEN the Project SHALL be created or patched with that admission policy only if the effective sync actor is authorized for that Project and profile
- AND Application diff, sync status, and resource status SHALL include drift in `session_admission`

#### Scenario: Local Application Does Not Bypass Admission Authorization
- GIVEN an Application targets the local Ambient instance
- WHEN the Application sync controller applies `session_admission`
- THEN authorization SHALL be evaluated against the Application's effective sync actor, not the controller's internal service bypass
- AND the sync SHALL fail for any profile the effective sync actor could not select through the normal Project API

#### Scenario: Application Renders Raw Kueue Resource
- GIVEN an Application renders a Kueue `ClusterQueue`, `LocalQueue`, `ResourceFlavor`, or `Workload`
- WHEN the Application syncs
- THEN the sync engine SHALL skip that document as unsupported infrastructure
- AND the Application `resource_status` SHALL record the skipped resource
- AND the sync operation SHALL continue for supported Ambient resources

#### Scenario: Remote Application Uses Same Contract
- GIVEN an Application targets a remote Ambient instance
- WHEN it syncs Project `session_admission`
- THEN authorization and validation SHALL be enforced by the destination Ambient API
- AND the source instance SHALL NOT infer or create scheduler resources on behalf of the destination instance

### Requirement: Kueue Admission Mode
The control plane SHALL support an optional Kueue-backed session admission mode that maps Project admission profiles to platform-managed Kueue queues.

#### Scenario: Kueue Disabled
- GIVEN Kueue admission mode is disabled
- WHEN the control plane provisions a Session
- THEN it SHALL NOT require Kueue APIs or labels
- AND it SHALL continue to provision runner Pods through the default Kubernetes scheduling path
- AND the Session SHALL remain `Queued` until the runner Pod is scheduled, ready, and reachable

#### Scenario: Kueue Enabled
- GIVEN Kueue admission mode is enabled
- AND a Project resolves to an admission profile
- WHEN a Session runner Pod is created
- THEN the Pod SHALL target the platform-managed LocalQueue for that Project and profile
- AND the Session SHALL remain `Queued` until the Pod is admitted, scheduled, ready, and the runner is reachable

#### Scenario: Runner Reachability
- GIVEN a runner admission Pod exists for a Session
- WHEN the Pod has Kubernetes `Ready=True`
- AND an HTTP `GET /health` probe to the runner on the configured runner service or Pod endpoint returns a 2xx response
- THEN the control plane SHALL treat the runner as reachable
- AND the Session MAY transition from `Queued` to `Running`

#### Scenario: Runner Reachability Timeout
- GIVEN a runner admission Pod exists for a Session
- WHEN the Pod does not become scheduled, ready, and reachable before the resolved profile's `runner_start_timeout`
- THEN the Session SHALL become `Failed`
- AND the Session SHALL include a condition with type `RunnerReachable` and a timeout reason

#### Scenario: Admission Infrastructure Missing
- GIVEN Kueue admission mode is enabled
- AND the Project's resolved queue cannot be found or created
- WHEN the control plane reconciles a pending Session
- THEN the Session SHALL become `Failed`
- AND the Session SHALL include a condition that identifies admission configuration as the failure category

### Requirement: Platform-Owned Queue Materialization
The system SHALL treat Kueue queue resources as platform-owned scheduler infrastructure materialized from platform configuration and Project admission policy.

#### Scenario: Project Namespace Has Local Queue
- GIVEN Kueue admission mode is enabled
- AND a managed Project namespace exists
- WHEN the Project resolves to an admission profile
- THEN the control plane SHALL ensure the namespace has a LocalQueue named by the resolved profile's `local_queue`
- AND the LocalQueue SHALL point to the platform-configured ClusterQueue for that profile
- AND the LocalQueue SHALL carry Ambient managed labels for project, profile, and control-plane ownership

#### Scenario: Local Queue Reconciles Drift
- GIVEN a managed LocalQueue exists in a Project namespace
- WHEN its ClusterQueue reference or managed labels drift from the resolved admission profile
- THEN the control plane SHALL update the LocalQueue rather than create a duplicate

#### Scenario: Local Queue Deletion
- GIVEN a Project is deleted
- WHEN no non-terminal Sessions reference a managed LocalQueue for that Project
- THEN the control plane SHALL delete that managed LocalQueue

#### Scenario: Admin Owns Cluster Queues
- GIVEN a platform operator configures Kueue `ClusterQueue` and `ResourceFlavor` resources
- WHEN Project owners change Project manifests
- THEN Project owners SHALL NOT be able to create, update, or delete those cluster-scoped scheduler resources through Project or Application APIs

#### Scenario: Project Profile Changes
- GIVEN a Project's `session_admission.profile` changes
- WHEN the control plane reconciles Project scheduler infrastructure
- THEN new Sessions SHALL use the new profile
- AND already-created Session runner Pods SHALL retain their originally resolved queue unless explicitly restarted

### Requirement: Resource Accounting Inputs
The control plane SHALL provide explicit resource requests for every container in a runner admission Pod, and SHALL validate any user-configurable resource overrides before they affect admission.

#### Scenario: Default Requests Applied
- GIVEN a Session has no resource overrides
- WHEN the control plane creates the runner Pod
- THEN every Pod container SHALL include platform default CPU and memory requests sufficient for scheduler accounting
- AND the Pod SHALL be eligible for pod-count quota accounting when the scheduler profile uses pod quotas

#### Scenario: Agent Resource Overrides Applied
- GIVEN an Agent defines valid `resource_overrides`
- WHEN that Agent is started
- THEN the copied Session overrides MAY affect runner Pod resource requests
- AND the effective request SHALL remain within the Project's allowed admission profile

#### Scenario: Invalid Override Rejected
- GIVEN a Session would use malformed or disallowed `resource_overrides`
- WHEN the API or control plane validates the Session
- THEN the Session SHALL fail before admission
- AND the failure SHALL be visible as a validation or status condition rather than silently falling back to different resources

### Requirement: Admission Status Observability
The system SHALL expose queue/admission state through Session phase and conditions without requiring users to read Kubernetes objects.

Sessions SHALL snapshot resolved admission state in API fields:

```yaml
admission_profile: standard
admission_queue: ambient-sessions-standard
```

Session `conditions` SHALL be a JSON array of condition objects with `type`, `status`, `reason`, `message`, and `last_transition_time`. Standard admission-related condition types SHALL include `AdmissionProfileResolved`, `AdmissionQueued`, `AdmissionConfiguration`, `PodScheduled`, `RunnerReachable`, and `Preempted`.

#### Scenario: Queue Wait Is Visible
- GIVEN a Session is `Queued`
- WHEN a user reads, lists, watches, or describes the Session
- THEN the response SHALL include the queue/admission phase
- AND SHALL include the resolved admission profile and queue identifier when resolved
- AND SHOULD include the latest wait reason when available

#### Scenario: Scheduler Objects Are For Operators
- GIVEN a platform operator has Kubernetes access
- WHEN they inspect Kueue objects directly
- THEN Kueue `Workload`, `LocalQueue`, and `ClusterQueue` status MAY be used for operational diagnosis
- AND that Kubernetes access SHALL NOT be required for normal Ambient user workflows

### Requirement: Preemption And Replacement
The control plane SHALL handle scheduler preemption explicitly and SHALL NOT silently lose a running Session's work.

#### Scenario: Queued Pod Preempted Before Runner Start
- GIVEN a Session is `Queued`
- WHEN its runner Pod is preempted or deleted before the runner starts
- AND the Session is not `Stopping`
- THEN the control plane SHALL recreate the runner Pod for the same resolved profile and queue
- AND the Session SHALL remain `Queued` with a condition recording the preemption

#### Scenario: Running Pod Preempted
- GIVEN a Session is `Running`
- WHEN its runner Pod is preempted or deleted by the scheduler or disappears without a user stop request
- THEN the Session SHALL become `Failed` unless durable resume for preempted sessions is explicitly supported
- AND the Session SHALL include a `Preempted` condition

### Requirement: RBAC
The system SHALL enforce Project and Application RBAC on admission policy changes and SHALL keep scheduler infrastructure permissions separate from tenant permissions.

#### Scenario: Project Owner Updates Admission Profile
- GIVEN a user has permission to update a Project
- AND the requested admission profile is tenant-selectable
- WHEN the user patches `session_admission.profile`
- THEN the API SHALL allow the update

#### Scenario: Privileged Profile Requires Privilege
- GIVEN an admission profile is not tenant-selectable
- WHEN a non-privileged Project user attempts to select that profile
- THEN the API SHALL reject the update

#### Scenario: Platform Admin Selects Privileged Profile
- GIVEN an admission profile is not tenant-selectable
- AND the caller has platform-admin authority through `*:*`
- WHEN the caller creates, patches, applies, or syncs a Project selecting that profile
- THEN the API SHALL allow the update

#### Scenario: Bootstrap Project Create Uses Default Admission
- GIVEN Project creation has no established Project-specific role binding yet
- WHEN the request has no platform-admin authority
- THEN the request SHALL omit `session_admission` or select only the platform default profile
- AND non-default profile selection SHALL be rejected at create time

#### Scenario: Application Sync Actor Applies Policy
- GIVEN an Application effective sync actor lacks permission to update the destination Project or select the requested profile
- WHEN the Application syncs a Project manifest changing `session_admission`
- THEN the sync SHALL fail for that Project resource
- AND the Application SHALL report the authorization failure in resource status

#### Scenario: Local Application Effective Sync Actor
- GIVEN an Application targets the local Ambient instance
- WHEN the Application is created
- THEN the API SHALL record the authenticated creator as the default effective sync actor
- AND later local syncs SHALL authorize `session_admission` changes as that actor unless the Application explicitly stores another authorized sync actor

#### Scenario: Control Plane Kueue Permissions
- GIVEN Kueue admission mode is enabled
- WHEN the control plane reconciles Project queues and Session runner Pods
- THEN the control plane SHALL have Kubernetes RBAC to get, list, watch, create, patch, update, and delete namespaced Kueue LocalQueues
- AND SHALL have Kubernetes RBAC to get, list, and watch Kueue Workloads for observed Session Pods
- AND tenant users SHALL NOT receive Kubernetes RBAC to manage Kueue ClusterQueues, ResourceFlavors, LocalQueues, or Workloads through Ambient Project or Application permissions

### Requirement: Consumer Migration
The system SHALL migrate all existing consumers of Session phase and Project declarative manifests to the new admission contract.

#### Scenario: Generated API Clients
- GIVEN OpenAPI, gRPC, Go SDK, Python SDK, TypeScript SDK, CLI, and frontend clients represent Project or Session data
- WHEN `session_admission` and `Queued` are added to the API contract
- THEN generated and hand-written clients SHALL be updated together
- AND Project create, patch, read, CLI create/update/apply, and Application sync clients SHALL carry `session_admission`
- AND Session read, list, watch, SDK, CLI, and frontend clients SHALL carry `admission_profile`, `admission_queue`, and structured `conditions`
- AND Session phase parsers, active-session checks, stop/delete gates, polling, action availability, and display helpers SHALL treat `Queued` as an active non-terminal phase

#### Scenario: Active Session Queries
- GIVEN code checks whether an Agent has an active Session
- WHEN the existing Session is `Queued`
- THEN the check SHALL treat it as active

#### Scenario: Declarative Apply
- GIVEN `acpctl apply` processes a Project manifest with `session_admission`
- WHEN the Project exists
- THEN apply SHALL patch drift in `session_admission`
- AND unchanged policies SHALL report `unchanged`

#### Scenario: Inbox Drain Timing
- GIVEN an Agent start request creates a Session that later waits in `Queued`
- WHEN unread Inbox messages existed at Agent start time
- THEN those messages SHALL be drained into that Session's start context at Agent start time
- AND Inbox messages created after Agent start while the Session is `Queued` SHALL remain unread for a future Agent start

#### Scenario: Existing Database Rows
- GIVEN existing Project rows have no admission policy
- WHEN the migration is applied
- THEN those Projects SHALL behave as if they use the platform default profile
- AND no existing Session row SHALL require phase rewriting
- AND existing Session rows SHALL receive null `admission_profile` and `admission_queue` values until reconciled
- AND existing empty Session `conditions` values SHALL migrate to an empty JSON array
- AND existing non-empty Session `conditions` values SHALL be preserved if they are valid JSON arrays, or wrapped in a `LegacyCondition` entry if they are not valid condition arrays
