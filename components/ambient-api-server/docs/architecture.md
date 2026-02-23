# Architecture

## Request Lifecycle

```
HTTP Request
  → gorilla/mux router (registered in plugin.go init())
  → JWT middleware (auth0 / mock depending on environment)
  → Authorization middleware (authz / mock)
  → Handler (handler.go)
    → Validates request body
    → Calls Service layer
    → Service calls DAO for persistence
    → Service creates Event (CREATE/UPDATE/DELETE)
    → Presenter converts model → OpenAPI response
  → handlers.Handle/HandleList/HandleGet/HandleDelete writes response
```

## Server Architecture

Three servers run concurrently, each managed by `pkg/server`:

| Server | Purpose | Default Port | Config Flag |
|--------|---------|-------------|-------------|
| API Server | REST endpoints | 8000 | `api-server-bindaddress` |
| Metrics Server | Prometheus metrics | 8080 | `metrics-server-bindaddress` |
| Health Check Server | `/health` endpoint | 8083 | `health-check-server-bindaddress` |

The API server supports `Listen()` + `Serve()` split (returns `net.Listener` for ephemeral port capture). Metrics and Health Check servers use monolithic `Start()` → `ListenAndServe()`.

## Environment Framework

```
cmd/ambient-api-server/environments/environments.go
  init()
    → trex.Init(Config{ServiceName, BasePath, ...})     // configures the framework singleton
    → pkgenv.NewEnvironment(nil)                         // creates global Env
    → env.SetEnvironmentImpls(EnvironmentImpls(env))     // registers dev/test/prod impls
```

The two-phase init pattern is critical: `trex.Init()` uses `sync.Once`, so the first call wins. The environments import in `main.go` (`_ ".../cmd/ambient-api-server/environments"`) triggers `init()` before `main()` runs.

### Environment Selection

```bash
AMBIENT_ENV=development          # DevEnvImpl — external DB, no auth
AMBIENT_ENV=integration_testing  # IntegrationTestingEnvImpl — testcontainer DB, mock auth
AMBIENT_ENV=production           # ProductionEnvImpl — external DB, full auth
```

Each impl overrides: `Flags()`, `OverrideConfig()`, `OverrideDatabase()`, `OverrideServices()`, `OverrideHandlers()`, `OverrideClients()`.

## Plugin Registration

All Kinds self-register via `init()` in `plugin.go`:

```go
func init() {
    registry.RegisterService("Agents", ...)        // Service locator
    pkgserver.RegisterRoutes("agents", ...)         // HTTP routes
    pkgserver.RegisterController("Agents", ...)     // Event controller
    presenters.RegisterPath(Agent{}, "agents")      // URL path mapping
    presenters.RegisterKind(Agent{}, "Agent")        // Kind name for responses
    db.RegisterMigration(migration())               // DB migration
}
```

These registrations are triggered by side-effect imports in `main.go` and test `TestMain` functions.

## Event-Driven Controllers

The `KindControllerManager` polls the `events` table for new events and dispatches to registered handlers:

```go
controllers.ControllerConfig{
    Source: "Agents",
    Handlers: map[api.EventType][]controllers.ControllerHandlerFunc{
        api.CreateEventType: {agentServices.OnUpsert},
        api.UpdateEventType: {agentServices.OnUpsert},
        api.DeleteEventType: {agentServices.OnDelete},
    },
}
```

Each service's `OnUpsert`/`OnDelete` methods are idempotent handlers for post-persistence logic (notifications, cascading updates, etc.).

## Database

- **ORM**: GORM v1.20.5 (with `gorm.io/gorm`)
- **Driver**: PostgreSQL via `gorm.io/driver/postgres`
- **Migrations**: `go-gormigrate/gormigrate` — each plugin registers a migration with a timestamp-based ID
- **Session Factory**: `db.SessionFactory` interface with implementations:
  - `db_session.NewProdFactory()` — connects to external PostgreSQL
  - `db_session.NewTestFactory()` — connects to external PostgreSQL for tests
  - `db_session.NewTestcontainerFactory()` — spins up PostgreSQL 14.2 in a container
- **Advisory Locks**: `db.LockFactory` provides `NewAdvisoryLock()` and `NewNonBlockingLock()` for concurrent update safety
- **Credentials**: Read from `secrets/` directory files (`db.host`, `db.port`, `db.name`, `db.user`, `db.password`)

## Upstream Dependencies

The project depends on `rh-trex-ai` (published module). Key upstream packages:

| Package | Purpose |
|---------|---------|
| `pkg/api` | `Meta` base model, `EventType`, `NewID()` (KSUID) |
| `pkg/server` | Server interfaces and factories |
| `pkg/environments` | `Env`, `EnvironmentImpl`, environment lifecycle |
| `pkg/handlers` | `HandlerConfig`, `Handle`, `HandleList`, validation helpers |
| `pkg/services` | `GenericService` (List with TSL search), `EventService`, `ListArguments` |
| `pkg/db` | `SessionFactory`, `LockFactory`, `RegisterMigration`, SQL helpers |
| `pkg/cmd` | Cobra commands (root, serve, migrate) |
| `pkg/controllers` | `KindControllerManager` for event-driven processing |
| `pkg/registry` | Service registry for dependency injection |
| `pkg/auth` | JWT and authorization middleware interfaces |
| `plugins/events` | Event persistence plugin |
| `plugins/generic` | Generic CRUD service (backs List handlers) |
