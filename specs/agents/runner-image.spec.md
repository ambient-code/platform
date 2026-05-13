# Custom Runner Image Specification

**Date:** 2026-05-12
**Status:** Proposed
**Related:**
  - `runner.spec.md` — Runner runtime, AG-UI protocol, bridge layer
  - `../control-plane/control-plane.spec.md` — Pod provisioning, image selection, env var injection
  - `../api/ambient-model.spec.md` — ProjectSettings, Session data model
  - `../security/security.spec.md` — Per-session SA isolation, credential boundaries

---

## Purpose

The Ambient Runner ships a single image containing Python, git, Node.js, Go, and several CLI tools. Workspace admins who need additional tools — Terraform, kubectl, language-specific SDKs, internal CLIs — have no supported extension path short of forking the image.

This spec defines a **stable runner contract** (the set of filesystem paths, HTTP endpoints, environment variables, and security constraints that custom images must preserve), a **Dockerfile FROM extension model** (users layer tools onto a published base image), and a **ProjectSettings-driven image override** (workspace admins declare a custom image per project).

The extension model is Dockerfile FROM only. Init hooks (scripts run at pod startup) were rejected: they are non-reproducible across pods, add startup latency, require runtime network egress that conflicts with NetworkPolicy isolation, and create OpenShift SCC conflicts when installing system packages.

This spec covers only the **image boundary** — what must be true about a container image for the platform to run it as a runner. Runner internals (bridge layer, gRPC transport, credential management) are defined in `runner.spec.md`. Pod provisioning mechanics are defined in `control-plane.spec.md`.

---

## Stable Runner Contract

Everything in this section is the stable interface. Anything not listed here is internal and MAY change without notice between runner releases.

### Requirement: AG-UI HTTP Contract

A custom runner image SHALL expose the AG-UI protocol on the port specified by the `AGUI_PORT` environment variable (default `8001`).

The following endpoints are part of the stable contract:

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/` | POST | AG-UI run — execute one turn, stream SSE events |
| `/interrupt` | POST | Halt the active run for a thread |
| `/health` | GET | Liveness/readiness probe |
| `/capabilities` | GET | Declare supported features to callers |
| `/events/{thread_id}` | GET | SSE live event stream for a specific thread |

Custom images MUST NOT remove, relocate, or change the response format of these endpoints. The remaining platform endpoints (`/repos`, `/workflow`, `/feedback`, `/mcp-status`, `/content`, `/tasks`) are registered by the `ambient_runner` package and inherited automatically.

#### Scenario: Custom image passes health check

- GIVEN a custom runner image built FROM the base
- WHEN the CP creates a pod and the readiness probe calls `GET /health`
- THEN the response is `200 OK`
- AND the session transitions to `Running` phase

#### Scenario: Custom image serves AG-UI protocol

- GIVEN a custom runner image is running in a session pod
- WHEN the api-server proxies a user message to `POST /`
- THEN the runner processes the turn and streams AG-UI events via SSE
- AND the event format is identical to the standard runner

---

### Requirement: Python Runtime Contract

Custom images SHALL provide Python 3.12+ and SHALL have the `ambient_runner` package installed. The runner process MUST use the same Python major.minor version as the base image.

The base image installs packages to system Python. Custom images MAY use virtual environments for additional Python dependencies, provided the `ambient_runner` package remains importable by the runner process.

Custom tools MAY use different Python versions via explicit interpreter paths, but the runner's uvicorn process MUST run under the base image's Python.

#### Scenario: Missing ambient_runner package

- GIVEN a custom image without the `ambient_runner` package
- WHEN the pod starts
- THEN the runner process fails to start
- AND the pod exits with a non-zero exit code
- AND the CP transitions the session to `Failed`

---

### Requirement: Filesystem Contract

A custom runner image SHALL preserve the following filesystem layout:

| Path | Constraint |
|------|------------|
| `/workspace` | MUST exist; EmptyDir mounted by CP at pod creation |
| `/app` | MUST exist; writeable by UID 1001; serves as `HOME` |
| `/app/ambient-runner` | MUST contain installed `ambient_runner` package |
| `/app/vertex` | MUST tolerate read-only Secret mount by CP (when Vertex AI enabled) |
| `/tmp` | MUST be writeable |

Custom images MAY add files and directories anywhere. Custom images MUST NOT remove or relocate the paths listed above.

#### Scenario: Custom tools installed in system PATH

- GIVEN a custom image with additional system packages installed
- WHEN a session runs in a pod using this image
- THEN the additional binaries are available in the agent's PATH
- AND all AG-UI endpoints function normally

---

### Requirement: Entrypoint Contract

Custom images SHOULD NOT override CMD or ENTRYPOINT. The platform controls the runner process lifecycle through the base image's default command.

If a custom image needs pre-startup logic, it MAY use a wrapper entrypoint that performs setup and then `exec`s the original command. The runner process MUST:

- Listen on the port specified by `AGUI_PORT` (default `8001`)
- Receive SIGTERM for graceful shutdown (process must be PID 1 or a direct child of PID 1)
- Start within the pod's startup timeout

#### Scenario: Wrapper entrypoint preserves signal handling

- GIVEN a custom image with a wrapper entrypoint that execs the runner process
- WHEN the CP sends SIGTERM to the pod
- THEN the runner process receives the signal
- AND shuts down gracefully within `terminationGracePeriodSeconds`

---

### Requirement: Environment Contract

The following environment variables are injected by the CP at pod creation time. Custom images MUST NOT override these in the Dockerfile:

| Variable | Purpose |
|----------|---------|
| `SESSION_ID` | Primary session identifier |
| `PROJECT_NAME` | Project context |
| `WORKSPACE_PATH` | Workspace root (always `/workspace`) |
| `AGUI_PORT` | Runner HTTP port |
| `BACKEND_API_URL` | api-server base URL |
| `AMBIENT_GRPC_URL` | api-server gRPC address |
| `AMBIENT_GRPC_USE_TLS` | TLS flag for gRPC channel |
| `AMBIENT_CP_TOKEN_URL` | CP token endpoint |
| `AMBIENT_CP_TOKEN_PUBLIC_KEY` | RSA public key for token auth |
| `INITIAL_PROMPT` | Auto-execute prompt |
| `IS_RESUME` | Resume flag on pod restart |
| `CREDENTIAL_IDS` | JSON map of resolved credential IDs |
| `RUNNER_TYPE` | Bridge selection (from agent registry) |

The base image also sets `PYTHONUNBUFFERED=1`, `HOME=/app`, and `SHELL=/bin/bash`. Custom images SHOULD preserve these.

Custom images MAY set additional environment variables. Custom images MUST NOT unset CP-injected variables.

#### Scenario: Custom image adds environment variables

- GIVEN a custom image with additional `ENV` directives
- WHEN a session pod starts
- THEN both the custom env vars and all CP-injected env vars are present
- AND the runner starts normally

---

### Requirement: Security Contract

A custom runner image SHALL run as a non-root user with no elevated privileges.

| Constraint | Enforced by |
|------------|-------------|
| `runAsNonRoot: true` | Pod SecurityContext |
| `allowPrivilegeEscalation: false` | Pod SecurityContext |
| `drop: ["ALL"]` capabilities | Pod SecurityContext |

The base image sets `USER 1001` as the default non-root UID. Custom images SHOULD preserve this default. On OpenShift, the restricted SCC assigns an arbitrary UID from the namespace range, overriding the Dockerfile directive.

Custom images MAY use `USER 0` during build stages for installing system packages, provided the final `USER` directive sets a non-root UID. Custom images SHOULD include OpenShift arbitrary-UID compatibility (`chmod -R g=u` on writeable paths) so the image functions under any non-root UID.

#### Scenario: Custom image with system package installation

- GIVEN a custom image that installs system packages as root during build
- AND sets a non-root `USER` as the final directive
- WHEN the pod starts with `securityContext.runAsNonRoot: true`
- THEN the pod starts successfully
- AND the installed packages are executable by the runtime UID

---

## ProjectSettings Integration

### Requirement: Custom Runner Image Field

The ProjectSettings resource SHALL support a `runner_image` field (string). When set, the CP SHALL use this image instead of the default when creating session pods for that project.

The field SHALL contain a fully qualified container image reference: `registry/repository:tag` or `registry/repository@sha256:digest`. When empty or unset, the CP uses the default image.

#### Scenario: Project with custom runner image

- GIVEN a ProjectSettings with `runner_image` set to a custom image
- WHEN a session is started in that project
- THEN the CP creates the runner pod with the custom image
- AND all other pod configuration (env vars, volumes, security context) is unchanged

#### Scenario: Project without custom runner image

- GIVEN a ProjectSettings with `runner_image` unset
- WHEN a session is started
- THEN the CP uses the default runner image

---

### Requirement: Image Selection Precedence

The CP SHALL select the runner image using the following precedence (highest to lowest):

1. **ProjectSettings `runner_image`** — workspace admin override
2. **Agent registry `container.image`** — per-agent-type default (the agent registry ConfigMap defines runtime configuration — image, port, resources, sandbox — for each agent type)
3. **Operator `RUNNER_IMAGE` env var** — cluster-level default
4. **Hardcoded fallback**

`ProjectSettings.runner_image` overrides the **image** but not the **agent type configuration**. The `RUNNER_TYPE` env var, resource limits, state directory, and other agent-registry settings are still applied from the registry entry matching the session's runner type.

Custom images MUST contain the bridge implementation for every agent type that sessions in this project may use. Images built FROM the standard base inherit all bridges.

#### Scenario: Custom image with non-default runner type

- GIVEN a project with `runner_image` set to a custom image
- AND a session created with a non-default runner type
- WHEN the CP provisions the pod
- THEN the pod uses the custom image
- AND the pod env includes the `RUNNER_TYPE` from the agent registry
- AND the custom image MUST contain the matching bridge implementation

#### Scenario: No custom image — agent registry selects image

- GIVEN a project with `runner_image` unset
- AND a session with a specific runner type
- WHEN the CP provisions the pod
- THEN the pod uses the image from the agent registry entry for that runner type

When `runner_image` is unset, the agent registry provides one default image per runner type. When `runner_image` is set, it overrides the image for all runner types within the project — the custom image MUST include bridge implementations for every runner type the project uses.

---

### Requirement: Image Validation

The CP SHALL validate the `runner_image` value before creating pods.

The CP SHALL reject images where the reference is syntactically invalid (missing repository or tag/digest) or the registry host is empty.

The CP SHALL support an operator-level allowlist of permitted registries via `RUNNER_IMAGE_ALLOWED_REGISTRIES` (comma-separated hostnames). When set, images from unlisted registries SHALL be rejected and the session SHALL transition to `Failed` with a descriptive condition.

When the allowlist is unset, the CP SHALL accept any registry. Operators SHOULD configure the allowlist in production deployments.

#### Scenario: Image from disallowed registry

- GIVEN a registry allowlist that does not include `docker.io`
- AND a ProjectSettings with `runner_image` pointing to `docker.io`
- WHEN the CP validates the image reference
- THEN the session transitions to `Failed` with a condition describing the rejection

#### Scenario: No registry allowlist

- GIVEN no registry allowlist configured
- AND a ProjectSettings with `runner_image` pointing to any registry
- THEN the image is accepted

---

### Requirement: Image Pull Credentials

The ProjectSettings resource SHALL support a `runner_image_pull_secret` field (string) containing the name of a Kubernetes Secret (type `kubernetes.io/dockerconfigjson`) in the project's namespace.

When set, the CP SHALL validate that the referenced Secret exists and is of type `kubernetes.io/dockerconfigjson` before creating the pod. When the Secret does not exist or is the wrong type, the session SHALL transition to `Failed` with a descriptive condition.

When `RUNNER_IMAGE_ALLOWED_REGISTRIES` is configured, the CP SHALL verify that the `runner_image` registry is in the allowlist regardless of which pull secret is provided. The pull secret controls authentication, not trust — registry trust is governed by the allowlist.

#### Scenario: Private registry with pull secret

- GIVEN a ProjectSettings with `runner_image` and `runner_image_pull_secret` set
- AND the referenced Secret exists in the project namespace
- AND the image registry is in the operator allowlist
- WHEN the CP creates the runner pod
- THEN the pod spec includes the secret in `imagePullSecrets`

#### Scenario: Pull secret references non-existent Secret

- GIVEN a ProjectSettings with `runner_image_pull_secret` set to `my-secret`
- AND no Secret named `my-secret` exists in the project namespace
- WHEN the CP attempts to create the runner pod
- THEN the session transitions to `Failed` with a condition describing the missing Secret

---

### Requirement: Image Pull Policy

The CP SHALL set `imagePullPolicy` based on the image reference:

| Reference type | Policy |
|----------------|--------|
| `@sha256:` digest | `IfNotPresent` |
| `localhost/` prefix | `IfNotPresent` |
| All others (tags) | `Always` |

---

### Requirement: RBAC for Runner Image Configuration

Only users with `project_settings:update` permission SHALL be permitted to modify ProjectSettings, including the `runner_image` and `runner_image_pull_secret` fields. This follows the existing endpoint-level RBAC model.

#### Scenario: User without update permission

- GIVEN a user without `project_settings:update` permission
- WHEN they PATCH ProjectSettings with a `runner_image` value
- THEN the request is rejected with `403 Forbidden`

---

### Requirement: Running Sessions Unaffected

When `runner_image` changes on a ProjectSettings resource, the change SHALL apply to **new sessions only**. Running sessions continue using the image they were created with.

#### Scenario: Image change does not affect running sessions

- GIVEN running sessions in a project using image A
- WHEN the admin changes `runner_image` to image B
- THEN running sessions continue with image A
- AND the next session started uses image B

---

## Failure Modes

### Requirement: Health Check Timeout

The CP SHALL configure a readiness probe on the runner container (`GET /health` on `AGUI_PORT`). If the probe does not pass within the pod's startup timeout, the CP SHALL transition the session to `Failed`.

#### Scenario: Custom image crashes on start

- GIVEN a custom image with a broken dependency
- WHEN the pod starts and the runner process fails to initialize
- THEN the pod exits with a non-zero exit code
- AND the CP transitions the session to `Failed`

### Requirement: Bridge Mismatch

When a custom image does not contain the bridge implementation required by the session's `RUNNER_TYPE`, the runner process SHALL fail at startup. The pod logs SHALL contain an error identifying the missing bridge module.

Custom images built FROM the standard base image inherit all bridge implementations and are not affected.

#### Scenario: Custom image missing bridge for session runner type

- GIVEN a custom image that does not include the bridge for a given runner type
- AND a session is created with that runner type
- WHEN the pod starts
- THEN the runner process fails to load the bridge module
- AND the pod exits with a non-zero exit code
- AND the CP transitions the session to `Failed`

### Requirement: Image Pull Failure

When the kubelet cannot pull the custom image, the CP SHALL transition the session to `Failed` with the pull error in the session condition.

#### Scenario: Image does not exist in registry

- GIVEN `runner_image` pointing to a non-existent image
- WHEN the CP creates the pod
- THEN the kubelet enters `ImagePullBackOff`
- AND the CP transitions the session to `Failed`

---

## Security Boundary

Custom runner images run within the same security perimeter as the standard runner. The platform's security model is enforced externally — by the CP, operator, NetworkPolicy, and Kubernetes RBAC — not by the image. Custom images inherit these constraints without reimplementing them.

### Requirement: Platform-Enforced Security Inheritance

Custom runner images SHALL inherit all platform security controls. The following are enforced by the platform at pod creation time, not by the image:

| Control | Enforced by | Custom image responsibility |
|---------|-------------|-----------------------------|
| Pod SecurityContext (non-root, capabilities, privilege escalation) | CP pod spec | Set a non-root `USER` to satisfy `runAsNonRoot` |
| Per-session ServiceAccount and RBAC | Operator | None — SA is created and bound per session |
| Credential fetch and per-turn clearing | `ambient_runner` package | Preserve the package; do not override credential methods |
| Runner token authentication (AG-UI, CP, gRPC) | `ambient_runner` package | Preserve the package |
| NetworkPolicy (ingress and egress) | Cluster operator | None — pod inherits namespace policies |

Custom images MUST NOT bundle credentials, tokens, or secrets in the image layers. All credentials SHALL be fetched at runtime via cluster-internal API endpoints as defined in `../security/security.spec.md`.

#### Scenario: Custom image inherits credential isolation

- GIVEN a custom runner image with additional tools installed
- WHEN a session runs and the runner fetches credentials per turn
- THEN credentials are fetched via the same cluster-internal endpoints as the standard image
- AND credentials are cleared after each turn by the `ambient_runner` package

#### Scenario: Custom tools cannot access platform credentials directly

- GIVEN a custom image with a tool that needs external service access
- WHEN the tool attempts to read credentials from the filesystem or environment
- THEN no platform credentials are present outside the runner process's per-turn lifecycle
- AND the tool MUST obtain credentials through the runner's credential API

### Requirement: Network Isolation

Runner pods — including those using custom images — SHALL be subject to the cluster's NetworkPolicy rules. Network isolation (ingress restrictions, egress deny-by-default) is a cluster operator responsibility.

The platform's runner pods communicate only with cluster-internal services (API server, CP token endpoint, gRPC). Custom tools that require external network access (cloud provider APIs, package registries) are subject to the project namespace's egress policies, which are managed by the cluster operator.

Custom images MUST NOT require changes to the platform's NetworkPolicy configuration. Cluster operators MAY configure project-level egress rules to accommodate custom tool requirements. The platform MAY adopt additional network-level controls (egress filtering, DNS-based policies) as they become available. Such controls are additive — custom images are not affected.

#### Scenario: Custom image with external tool access

- GIVEN a custom image containing a tool that calls an external API
- AND the cluster operator has not configured egress for that destination
- WHEN the tool attempts to connect
- THEN the connection is blocked by NetworkPolicy
- AND the runner process continues to function normally

### Requirement: Image Supply Chain

Image vulnerability scanning and signing are cluster operator responsibilities, not platform runtime concerns. The platform SHALL NOT enforce image scanning at pod creation time.

Operators SHOULD configure image scanning in their CI/CD pipeline or container registry. Operators SHOULD use registry-level policies to prevent deployment of images with known critical vulnerabilities.

Custom images SHOULD be built from the base image referenced by digest (`@sha256:...`), not by mutable tag.

#### Scenario: Operator-configured image scanning

- GIVEN a cluster with registry-level vulnerability scanning enabled
- AND a custom image with a critical CVE in an installed package
- WHEN the image is pushed to the registry
- THEN the registry flags the vulnerability according to operator-configured policy
- AND the platform is not involved in the scanning decision

---

## Base Image Publishing

### Requirement: Published Base Image

The platform SHALL publish a base runner image suitable for `FROM` directives at a stable, versioned tag. The image SHALL be built from the same source as the standard runner image.

Breaking changes to the stable contract SHALL increment the major version.

### Requirement: Contract Version Label

The base image SHALL carry an OCI label indicating the contract version (e.g., `io.ambient-code.runner-contract-version`).

The CP MAY log a warning if the contract version does not match the expected version. The CP SHALL NOT block pod creation based on contract version mismatch.

#### Scenario: Contract version mismatch

- GIVEN the CP expects contract version `1`
- AND a custom image has a different contract version label
- WHEN the CP creates the pod
- THEN the CP logs a warning
- AND the pod is created normally

### Requirement: Conformance Test Suite

The platform SHALL publish a conformance test suite that validates a custom runner image against the stable contract. The test suite SHALL verify:

- AG-UI endpoints respond correctly (`/health`, `/capabilities`, `/`)
- Required filesystem paths exist and are writeable
- The runner process starts within the expected timeout
- The runner runs as a non-root user
- CP-injected environment variables are not overridden by the image

The test suite SHALL produce a pass/fail result suitable for CI/CD integration.

The test suite SHOULD include security checks: non-root user verification, no SUID binaries, and base image provenance validation. Operators MAY extend the suite with additional security scanning (vulnerability scanning, SBOM generation) using their existing tooling.

#### Scenario: Custom image passes conformance

- GIVEN a custom image built FROM the base
- AND no contract requirements violated
- WHEN the conformance test suite runs against the image
- THEN all checks pass

#### Scenario: Custom image fails conformance

- GIVEN a custom image that removed the `/workspace` directory
- WHEN the conformance test suite runs
- THEN the filesystem check fails
- AND the test suite reports the specific violation
