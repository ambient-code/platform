# Ambient Spec Development Context

**When to load:** Writing or reviewing the Ambient Data Model spec, openapi.yaml, or any document that defines *what* the platform should be.

---

## What the Spec Is

The Spec is **desired state** — the authoritative definition of what Ambient should be. It is not documentation that follows code. It is documentation that *precedes and governs* code.

**The Spec is correct. The code either matches it or is wrong.**

The reconciliation direction is always:

```
Spec → Status (code)
```

Never the reverse. If code was written that doesn't match the Spec, the code is wrong — unless the Spec is updated consciously to reflect a decision.

---

## What Belongs in the Spec

### Resource definitions (Kinds)

Every Kind (Session, Project, Agent, Role, RoleBinding, User, ProjectSettings, ...) must be fully documented:

- **What it is** — one-paragraph description of the concept
- **Fields** — name, type, required/optional, description, constraints, example values
- **Relationships** — how it relates to other Kinds (owns, references, derives from)
- **Lifecycle** — valid states and transitions (e.g. Session: `pending → running → completed | failed`)
- **Ownership** — which Kinds own which (OwnerReferences in K8s terms)

### API endpoints

Every REST endpoint must be documented in the Spec before it exists in code:

- **Method + path** — e.g. `POST /api/ambient/v1/{project}/sessions`
- **Path parameters** — name, type, description
- **Request body** — schema reference + example
- **Response** — schema reference + example + error codes
- **Authorization** — what RBAC permission is required
- **Behavior** — what happens (idempotency, side effects, ordering guarantees)

### gRPC streams

Every streaming RPC:

- **RPC name + proto service** — e.g. `Sessions.WatchSessionMessages`
- **Request fields**
- **Stream events** — each event type, when it's emitted, what it contains
- **Terminal conditions** — what ends the stream

### CLI commands

Every `acpctl` command is a first-class Spec artifact:

- **Command + subcommand** — e.g. `acpctl session messages -f`
- **Flags** — name, type, default, description
- **Behavior** — what it does, what it prints, exit codes
- **Example** — real invocation with expected output

CLI commands are the **user-facing surface of the Spec**. If a command doesn't exist in the CLI, the feature isn't shipped — regardless of what the API supports.

### SDK examples

Every resource API must have a working example in the Go SDK, Python SDK, and TypeScript SDK. These examples are:

1. **Documentation** — how to use the SDK
2. **End-to-end tests** — if the example doesn't run against a real cluster, something is broken

SDK examples belong in the Spec because they define the *intended usage contract*, not just the implementation.

---

## Spec Files in This Repository

| File | What it specifies |
|---|---|
| `components/ambient-api-server/ambient-data-model.md` | Canonical data model — all Kinds, fields, relationships |
| `components/ambient-api-server/openapi/openapi.yaml` | REST API surface — all endpoints, schemas, error codes |
| `components/ambient-api-server/openapi/openapi.*.yaml` | Per-resource OpenAPI fragments (merged into openapi.yaml) |
| `components/ambient-sdk/go-sdk/examples/` | Go SDK usage examples (spec-as-code) |
| `components/ambient-sdk/python-sdk/examples/` | Python SDK examples |
| `components/ambient-cli/README.md` | CLI command reference |
| `docs/internal/design/agent-workflow.md` | Agentic workflow Spec — how agents operate on sessions |

---

## Spec Quality Rules

A Spec entry is **complete** when:

1. The Kind/endpoint/command is fully described — no fields marked TBD
2. All relationships to other Kinds are documented
3. At least one CLI command exposes it
4. At least one SDK example exercises it
5. The openapi.yaml entry exists and is valid

A Spec entry is **incomplete** (blocks implementation) when:
- A field exists with no description or type
- A relationship is implied but not stated
- An endpoint exists in openapi.yaml but not in ambient-data-model.md (or vice versa)
- No CLI command or SDK example exists for the feature

**Incomplete Spec = implementation ambiguity = agent stops and asks.**

---

## Spec ↔ Status Relationship

```
Spec (ambient-data-model.md + openapi.yaml + CLI README + SDK examples)
    │
    Δ = Spec - Status
    │
    ├── Δ == ∅  →  no-op
    └── Δ ≠ ∅   →  Workflow(Δ) → TaskGraph → execute
```

Status is the sum of all running code: api-server plugins, SDK clients, CLI commands, frontend pages, operator logic, runner behavior.

When you change the Spec, you are creating a diff that must propagate through every layer of Status. The Workflow tells you how.

---

## Conventions

- **Edit `ambient-data-model.md` first**, then `openapi.yaml`, never the reverse
- **openapi.yaml is generated from fragments** (`openapi.*.yaml`) — edit the fragment, not the merged file directly
- **Never add a field to code that isn't in the Spec** — if you discover the code needs something not in the Spec, update the Spec first
- **SDK examples must be runnable** — test them against kind before committing
- **CLI README is part of the Spec** — if you add a command, update the README in the same PR
