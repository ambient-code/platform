# Plugin Anatomy

Each Kind is a self-contained plugin directory under `plugins/{kinds}/`. All plugins follow the same structure and registration pattern.

## File Structure

```
plugins/agents/
  plugin.go            # init() — wires everything together
  model.go             # GORM model + patch request
  handler.go           # HTTP handlers
  service.go           # Business logic + event handlers
  dao.go               # Database operations
  presenter.go         # OpenAPI ↔ model conversion
  migration.go         # Database migration
  mock_dao.go          # Mock DAO for testing
  factory_test.go      # Test factories (create test data directly in DB)
  integration_test.go  # Integration tests
  testmain_test.go     # TestMain setup
```

## Registration Flow (plugin.go)

Every plugin's `init()` registers five things:

```go
func init() {
    // 1. Service locator — lazy factory for the service
    registry.RegisterService("Agents", func(env interface{}) interface{} {
        return NewServiceLocator(env.(*environments.Env))
    })

    // 2. HTTP routes — CRUD endpoints on the mux router
    pkgserver.RegisterRoutes("agents", func(router, services, jwt, authz) {
        handler := NewAgentHandler(Service(services), generic.Service(services))
        router.HandleFunc("", handler.List).Methods("GET")
        router.HandleFunc("/{id}", handler.Get).Methods("GET")
        router.HandleFunc("", handler.Create).Methods("POST")
        router.HandleFunc("/{id}", handler.Patch).Methods("PATCH")
        router.HandleFunc("/{id}", handler.Delete).Methods("DELETE")
        router.Use(jwt.AuthenticateAccountJWT)
        router.Use(authz.AuthorizeApi)
    })

    // 3. Event controller — reacts to CREATE/UPDATE/DELETE events
    pkgserver.RegisterController("Agents", func(manager, services) {
        manager.Add(&controllers.ControllerConfig{
            Source: "Agents",
            Handlers: map[api.EventType][]controllers.ControllerHandlerFunc{
                api.CreateEventType: {service.OnUpsert},
                api.UpdateEventType: {service.OnUpsert},
                api.DeleteEventType: {service.OnDelete},
            },
        })
    })

    // 4. Presenter paths — maps model types to URL paths and Kind names
    presenters.RegisterPath(Agent{}, "agents")
    presenters.RegisterKind(Agent{}, "Agent")

    // 5. Database migration
    db.RegisterMigration(migration())
}
```

## Model Layer (model.go)

```go
type Agent struct {
    api.Meta                              // ID, CreatedAt, UpdatedAt, DeletedAt
    Name    string  `json:"name"`         // Required field (non-pointer)
    RepoUrl *string `json:"repo_url"`     // Optional field (pointer)
    Prompt  *string `json:"prompt"`       // Optional field (pointer)
}

func (d *Agent) BeforeCreate(tx *gorm.DB) error {
    d.ID = api.NewID()                    // KSUID generation
    return nil
}

type AgentPatchRequest struct {
    Name    *string `json:"name,omitempty"`    // All fields optional in patch
    RepoUrl *string `json:"repo_url,omitempty"`
    Prompt  *string `json:"prompt,omitempty"`
}
```

## Handler Layer (handler.go)

Handlers follow the `handlers.RestHandler` interface pattern:

- **Create**: Decode body → validate (ID must be empty) → service.Create → PresentX → 201
- **Get**: Extract `{id}` → service.Get → PresentX → 200
- **List**: Parse query params → generic.List → convert all → 200
- **Patch**: Extract `{id}` → service.Get → apply patch fields → service.Replace → 200
- **Delete**: Extract `{id}` → service.Delete → 204

The `List` handler uses `services.NewListArguments(r.URL.Query())` which parses `page`, `size`, `search`, `orderBy`, `fields` from query params, then delegates to `generic.List()` which handles TSL search parsing, pagination, and ordering.

## Service Layer (service.go)

Services encapsulate business logic and fire events:

```go
func (s *sqlAgentService) Create(ctx, agent) (*Agent, *errors.ServiceError) {
    agent, err := s.agentDao.Create(ctx, agent)        // Persist
    s.events.Create(ctx, &api.Event{                    // Fire event
        Source: "Agents", SourceID: agent.ID,
        EventType: api.CreateEventType,
    })
    return agent, nil
}

func (s *sqlAgentService) Replace(ctx, agent) (*Agent, *errors.ServiceError) {
    lockOwnerID := s.lockFactory.NewAdvisoryLock(...)   // Advisory lock
    defer s.lockFactory.Unlock(ctx, lockOwnerID)
    agent, err := s.agentDao.Replace(ctx, agent)        // Persist
    s.events.Create(ctx, &api.Event{...UpdateEventType}) // Fire event
    return agent, nil
}
```

## DAO Layer (dao.go)

Direct GORM operations:

```go
func (d *sqlAgentDao) Get(ctx, id) (*Agent, error) {
    g2 := (*d.sessionFactory).New(ctx)
    g2.Take(&agent, "id = ?", id)
}

func (d *sqlAgentDao) Create(ctx, agent) (*Agent, error) {
    g2 := (*d.sessionFactory).New(ctx)
    g2.Omit(clause.Associations).Create(agent)
}
```

## Presenter Layer (presenter.go)

Bidirectional conversion between GORM models and OpenAPI types:

- `ConvertX(openapi.X) → *model.X` — request body → model (for Create)
- `PresentX(*model.X) → openapi.X` — model → response body (for all responses)

Uses `presenters.PresentReference()` to populate `id`, `kind`, `href` fields.

## Testing

### Test Factories (factory_test.go)

Create test data directly in the database (bypassing HTTP):

```go
func newAgent(id string) (*agents.Agent, error) {
    agent := &agents.Agent{Name: "test-" + id, ...}
    return agent, db.Create(agent)
}
```

### Integration Tests (integration_test.go)

Full HTTP round-trip tests using the generated OpenAPI client:

```go
func TestAgentPost(t *testing.T) {
    h, client := test.RegisterIntegration(t)
    ctx := h.NewAuthenticatedContext(h.NewRandAccount())
    resp, httpResp, err := client.DefaultAPI.ApiAmbientApiServerV1AgentsPost(ctx).Agent(input).Execute()
}
```

### TestMain (testmain_test.go)

Each plugin test package has a `TestMain` that initializes the test helper (which starts the API server, metrics server, and health check server with a testcontainer PostgreSQL database).

## Adding Custom Business Logic

1. **Validation**: Add `handlers.Validate` functions in the handler's `Create` or `Patch` methods
2. **Side effects**: Implement `OnUpsert` and `OnDelete` in the service (they fire asynchronously via the event controller)
3. **Custom queries**: Add methods to the DAO interface and implementation
4. **Cross-kind logic**: Import other plugin services via their `Service()` locator function
