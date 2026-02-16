# CLAUDE.md — ambient-api-server

REST API microservice for the Ambient Code Platform. Built on the [rh-trex-ai](https://github.com/openshift-online/rh-trex-ai) framework with auto-generated Kind plugins providing CRUD, event-driven controllers, and OpenAPI client generation.

## Quick Reference

```bash
make test              # OCM_ENV=integration_testing go test -p 1 -v ./...
make binary            # Build binary
make run               # Migrate + serve (with auth)
make run-no-auth       # Migrate + serve (no auth, dev mode)
make generate          # Regenerate OpenAPI client from specs
make db/setup          # Start PostgreSQL via Podman/Docker
make db/teardown       # Stop PostgreSQL
```

### Testing Prerequisites

- **Podman**: `systemctl --user start podman.socket`
- **DOCKER_HOST**: `export DOCKER_HOST=unix:///run/user/1000/podman/podman.sock`
- Tests use `testcontainers-go` to spin up PostgreSQL per test package
- Integration tests bind ephemeral ports (`localhost:0`) to avoid conflicts

### Lint / Format

```bash
go fmt ./...
golangci-lint run
```

## Architecture Overview

```
main.go → imports plugins (init side-effects) → registers routes, controllers, migrations
        → pkgcmd.NewServeCommand starts API server, metrics server, health check server
        → pkgcmd.NewMigrateCommand runs gormigrate migrations
```

**→ Load `docs/architecture.md` for the full request lifecycle, environment system, and framework integration.**

## Domain Model

Eight resource Kinds following the "AS agent WITH skills DO tasks" workflow pattern:

| Kind | Key Fields | Purpose |
|------|-----------|---------|
| **Agent** | name, repo_url, prompt | AI agent definition |
| **Skill** | name, repo_url, prompt | Capability a workflow can use |
| **Task** | name, repo_url, prompt | Unit of work in a workflow |
| **Workflow** | name, repo_url, prompt, agent_id | Orchestration binding agent → skills → tasks |
| **WorkflowSkill** | workflow_id, skill_id, position | Join: workflow ↔ skill (ordered) |
| **WorkflowTask** | workflow_id, task_id, position | Join: workflow ↔ task (ordered) |
| **Session** | name, repo_url, prompt, workflow_id, created_by_user_id, assigned_user_id | Execution instance of a workflow |
| **User** | username, name | Platform user |

**→ Load `docs/data-model.md` for field details, relationships, and database schema.**

## Plugin System

Each Kind is a self-contained plugin in `plugins/{kinds}/` with uniform structure:

| File | Role |
|------|------|
| `plugin.go` | `init()` — registers service, routes, controller, presenter paths, migration |
| `model.go` | Gorm model + patch request struct |
| `handler.go` | HTTP handlers (Create, Get, List, Patch, Delete) |
| `service.go` | Business logic + event handlers (OnUpsert, OnDelete) |
| `dao.go` | Data access (Get, Create, Replace, Delete, FindByIDs, All) |
| `presenter.go` | OpenAPI ↔ model conversion (ConvertX, PresentX) |
| `migration.go` | Gormigrate migration with AutoMigrate |
| `mock_dao.go` | Mock DAO for unit tests |
| `*_test.go` | Integration tests + test factories |

**→ Load `docs/plugin-anatomy.md` for the full plugin lifecycle and how to extend.**

## Code Generation

```bash
go run ./scripts/generator.go \
  --kind YourKind \
  --fields "name:string:required,description:string,priority:int" \
  --project ambient-api-server \
  --repo github.com/ambient/platform/components \
  --library github.com/openshift-online/rh-trex-ai
```

Supported field types: `string`, `int`, `int64`, `bool`, `float`, `time`
Modifiers: `:required` (non-nullable), `:optional` (nullable, default)

**→ Load `docs/code-generation.md` for generator internals, template variables, and OpenAPI auto-wiring.**

## Upstream Framework (rh-trex-ai)

The `go.mod` uses a `replace` directive pointing to the local clone:

```
replace github.com/openshift-online/rh-trex-ai => /home/mturansk/projects/src/github.com/openshift-online/rh-trex-ai
```

Key upstream packages consumed:
- `pkg/api` — Meta type, event types, ID generation
- `pkg/server` — API server, metrics, health check servers
- `pkg/environments` — Environment framework (dev, test, prod)
- `pkg/handlers` — HTTP handler patterns (Handle, HandleList, HandleGet, HandleDelete)
- `pkg/services` — GenericService (List with TSL search), EventService, ListArguments
- `pkg/db` — SessionFactory, advisory locks, migrations, SQL helpers
- `pkg/cmd` — Root/Serve/Migrate cobra commands
- `pkg/controllers` — Event-driven controller manager
- `plugins/events`, `plugins/generic` — Core upstream plugins

**→ Load `trex_comms.md` for the upstream bug fix log and communication protocol.**

## Search Query Syntax

The API uses [Tree Search Language (TSL)](https://github.com/yaacov/tree-search-language) for the `?search=` parameter. Queries must be structured expressions, NOT free-text:

```
name = 'Claude Code Assistant'
id in ('abc123', 'def456')
username like 'test%'
created_at > '2026-01-01T00:00:00Z'
name = 'API Design' and repo_url like '%github%'
```

## Project Layout

```
cmd/ambient-api-server/
  main.go                          # Entry point, plugin imports
  environments/
    environments.go                # init() → trex.Init() + env setup
    types.go                       # Type aliases from upstream
    e_development.go               # Dev env (no auth, localhost:8000)
    e_integration_testing.go       # Test env (testcontainers, ephemeral ports, mock authz)
    e_production.go                # Prod env
    e_unit_testing.go              # Unit test env
plugins/{kinds}/                   # 8 Kind plugins (see Plugin System above)
pkg/api/
  api.go                           # Re-exports from rh-trex-ai/pkg/api
  openapi_embed.go                 # GetOpenAPISpec() for server bootstrap
  openapi/                         # Generated OpenAPI Go client
openapi/                           # OpenAPI YAML specs (source of truth)
scripts/generator.go               # Kind code generator
templates/                         # Go text/template files for generator
test/
  helper.go                        # Test infrastructure (server startup, JWT, API client)
  registration.go                  # RegisterIntegration() convenience wrapper
  integration/                     # Cross-kind integration tests
secrets/                           # DB credentials (db.host, db.port, db.name, db.user, db.password)
```

## Environment System

Selected via `OCM_ENV` env var. Each environment implements `EnvironmentImpl` with:
- `Flags()` — CLI flag overrides
- `OverrideConfig()` — ApplicationConfig tweaks
- `OverrideDatabase()` — SessionFactory selection (testcontainer vs prod)
- `OverrideServices()`, `OverrideHandlers()`, `OverrideClients()`

| Environment | OCM_ENV | Database | Auth | Ports |
|-------------|---------|----------|------|-------|
| Development | `development` | External PostgreSQL | Disabled | localhost:8000 |
| Integration Testing | `integration_testing` | Testcontainer PostgreSQL | Mock | Ephemeral (localhost:0) |
| Production | `production` | External PostgreSQL | Enabled | Configured |

## API Endpoints

All routes under `/api/ambient-api-server/v1/`:

| Method | Path | Operation |
|--------|------|-----------|
| GET | `/{kinds}` | List (supports `?search=`, `?page=`, `?size=`, `?orderBy=`, `?fields=`) |
| POST | `/{kinds}` | Create |
| GET | `/{kinds}/{id}` | Get |
| PATCH | `/{kinds}/{id}` | Patch |
| DELETE | `/{kinds}/{id}` | Delete |

Kinds: `agents`, `skills`, `tasks`, `workflows`, `sessions`, `workflow_skills`, `workflow_tasks`, `users`

## Cross-Session Coordination

**→ Load `../working.md` for the shared working document between ambient-api-server and ambient-control-plane sessions.**

This file coordinates development across concurrent Claude sessions. Protocol:
- **Read before writing** — avoid clobbering the other session's updates
- **Tag entries** with `[API]` or `[CP]` to identify source
- Sections: Announcements (breaking changes), Requests (cross-session asks), Status (current work), Contracts (agreed API shapes)

### Active Contracts

| Resource | Endpoint | Change Detection |
|----------|----------|------------------|
| Sessions | `GET /api/ambient-api-server/v1/sessions` | `updated_at` diff |
| Workflows | `GET /api/ambient-api-server/v1/workflows` | `updated_at` diff |
| Tasks | `GET /api/ambient-api-server/v1/tasks` | `updated_at` diff |

Auth: `Authorization: Bearer <token>`. Pagination: `?page=1&size=100` (1-indexed).

## Conventions

- All Go code uses `go fmt`; `golangci-lint run` must pass
- No `panic()` in production code
- Table-driven tests with subtests
- OpenAPI client is generated — do not edit `pkg/api/openapi/` manually
- Plugin imports in `main.go` are side-effect imports (`_ "..."`)
- `api.Meta` provides `ID`, `CreatedAt`, `UpdatedAt`, `DeletedAt` to all models
- `BeforeCreate` gorm hook assigns `api.NewID()` (KSUID)
