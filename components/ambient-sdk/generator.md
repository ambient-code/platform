# SDK Generator Specification

## Problem

The SDK has 8 resources that follow identical CRUD patterns. Hand-writing clients for each resource in two languages (Go, Python) produces ~3,000 lines of near-identical code that drifts from the API server's OpenAPI spec between releases. When the API server adds fields, resources, or endpoints, the SDK must be updated in lockstep — manually doing this is error-prone and slow.

## Goal

A code generator that reads the canonical `openapi.yaml` and produces idiomatic Go and Python SDK code. The generated SDKs use the **Builder pattern** for human-readable request construction while keeping response types simple and transparent. Re-running the generator on any OpenAPI change produces a correct SDK with zero manual intervention.

## OpenAPI Shape Analysis

### What the spec tells us

Every resource in the API follows the same template:

```
Resource        = ObjectReference + resource-specific fields (allOf)
ResourceList    = List + items[]Resource (allOf)
PatchRequest    = flat object with resource-specific mutable fields
```

Base schemas (shared, never change per-resource):

| Schema          | Fields                                          |
|-----------------|------------------------------------------------|
| ObjectReference | id, kind, href, created_at, updated_at         |
| List            | kind, page, size, total (+ items in each *List) |
| Error           | ObjectReference + code, reason, operation_id    |

Every list endpoint accepts 5 query params: `page`, `size`, `search`, `orderBy`, `fields`.

Every resource supports: `GET /resources` (list), `POST /resources` (create), `GET /resources/{id}` (get), `PATCH /resources/{id}` (update). User additionally supports `DELETE`.

### Current 8 resources and their specific fields

| Resource      | Specific Fields (beyond ObjectReference)             | Required        | Has Delete |
|---------------|-----------------------------------------------------|-----------------|------------|
| Session       | name, repo_url, prompt, created_by_user_id, assigned_user_id, workflow_id | name | No |
| Agent         | name, repo_url, prompt                               | name            | No         |
| Task          | name, repo_url, prompt                               | name            | No         |
| Skill         | name, repo_url, prompt                               | name            | No         |
| Workflow      | name, repo_url, prompt, agent_id                     | name            | No         |
| User          | username, name, created_at*, updated_at*             | username, name  | Yes        |
| WorkflowSkill | workflow_id, skill_id, position (int32)              | all three       | No         |
| WorkflowTask  | workflow_id, task_id, position (int32)               | all three       | No         |

*User re-declares created_at/updated_at from ObjectReference (redundant but harmless).

## Architecture

### What gets generated vs. what stays hand-written

```
ambient-sdk/
├── generator/                    # THE GENERATOR (hand-written, runs at build time)
│   ├── main.go                   # CLI entry point: reads openapi.yaml, emits Go + Python
│   ├── parser.go                 # OpenAPI YAML parser (resolves $ref, allOf)
│   ├── model.go                  # Intermediate representation (Resource, Field, Endpoint)
│   ├── templates/
│   │   ├── go/
│   │   │   ├── types.go.tmpl     # Per-resource type + builder
│   │   │   ├── client.go.tmpl    # Per-resource client methods
│   │   │   ├── base.go.tmpl      # ObjectReference, List, Error, ListOptions
│   │   │   └── iterator.go.tmpl  # Pagination iterator
│   │   └── python/
│   │       ├── types.py.tmpl     # Per-resource dataclass + builder
│   │       ├── client.py.tmpl    # Per-resource client methods
│   │       ├── base.py.tmpl      # ObjectReference, List, Error
│   │       └── iterator.py.tmpl  # Pagination iterator
│   └── generator_test.go         # Golden-file tests
│
├── go-sdk/                       # GENERATED OUTPUT (do not hand-edit)
│   ├── types/
│   │   ├── base.go               # generated: ObjectReference, ListMeta, APIError
│   │   ├── session.go            # generated: Session, SessionBuilder, SessionPatchBuilder
│   │   ├── agent.go              # generated: Agent, AgentBuilder, ...
│   │   ├── ... (one per resource)
│   │   └── list_options.go       # generated: ListOptions builder
│   ├── client/
│   │   ├── client.go             # HAND-WRITTEN: Client struct, auth, SecureToken, sanitizeLogAttrs
│   │   ├── session_api.go        # generated: Sessions() resource accessor
│   │   ├── agent_api.go          # generated: Agents() resource accessor
│   │   ├── ... (one per resource)
│   │   └── iterator.go           # generated: generic pagination iterator
│   ├── examples/main.go          # hand-written
│   ├── go.mod
│   └── README.md
│
├── python-sdk/                   # GENERATED OUTPUT (do not hand-edit)
│   ├── ambient_platform/
│   │   ├── __init__.py           # hand-written (public exports, version)
│   │   ├── _base.py              # generated: ObjectReference, ListMeta, APIError
│   │   ├── session.py            # generated: Session, SessionBuilder, SessionPatch
│   │   ├── agent.py              # generated: Agent, AgentBuilder, ...
│   │   ├── ... (one per resource)
│   │   ├── client.py             # HAND-WRITTEN: AmbientClient, auth, from_env(), context manager
│   │   ├── _session_api.py       # generated: SessionAPI mixin
│   │   ├── _agent_api.py         # generated: AgentAPI mixin
│   │   ├── ... (one per resource)
│   │   ├── _iterator.py          # generated: pagination iterator
│   │   └── exceptions.py         # hand-written
│   ├── examples/main.py
│   ├── pyproject.toml
│   └── README.md
```

### Why a custom generator instead of openapi-generator?

1. **Builder pattern.** Standard openapi-generator emits flat structs/classes with constructors. We want `NewSessionBuilder().Name("x").Prompt("y").Build()` — that requires custom templates regardless.
2. **Minimal dependencies.** Go SDK stays stdlib-only. Python SDK stays httpx-only. openapi-generator introduces runtime libraries.
3. **The spec is highly uniform.** 8 resources, same CRUD pattern, same pagination, same error schema. A custom generator is ~500 lines of Go; openapi-generator is a 200MB Java runtime generating thousands of lines of framework code we'd immediately delete.
4. **Security.** Hand-written `SecureToken`, `sanitizeLogAttrs`, and token validation stay outside the generated boundary. We don't want a generator touching auth code.

## Builder Pattern Design

### Go: Fluent Builder with Compile-Time Safety

```go
// --- CREATING ---
session, err := client.Sessions().Create(ctx,
    ambient.NewSessionBuilder().
        Name("analyze-security-report").
        Prompt("Review the latest CVE report and summarize findings").
        RepoURL("https://github.com/org/repo").
        AssignedUserID("user-123").
        Build(),
)

// --- LISTING with pagination ---
sessions, err := client.Sessions().List(ctx,
    ambient.NewListOptions().
        Page(2).
        Size(50).
        Search("name like 'security%'").
        OrderBy("created_at desc").
        Build(),
)

// --- PATCHING ---
updated, err := client.Sessions().Update(ctx, sessionID,
    ambient.NewSessionPatchBuilder().
        Name("updated-name").
        Prompt("new prompt").
        Build(),
)

// --- GET ---
session, err := client.Sessions().Get(ctx, sessionID)

// --- ITERATING all pages ---
iter := client.Sessions().ListAll(ctx,
    ambient.NewListOptions().Size(100).Build(),
)
for iter.Next() {
    session := iter.Item()
    fmt.Println(session.Name)
}
if err := iter.Err(); err != nil {
    log.Fatal(err)
}
```

### Python: Fluent Builder with Chainable Methods

```python
# --- CREATING ---
session = client.sessions.create(
    Session.builder()
        .name("analyze-security-report")
        .prompt("Review the latest CVE report and summarize findings")
        .repo_url("https://github.com/org/repo")
        .assigned_user_id("user-123")
        .build()
)

# --- LISTING with pagination ---
sessions = client.sessions.list(
    ListOptions()
        .page(2)
        .size(50)
        .search("name like 'security%'")
        .order_by("created_at desc")
)

# --- PATCHING ---
updated = client.sessions.update(session_id,
    SessionPatch()
        .name("updated-name")
        .prompt("new prompt")
)

# --- GET ---
session = client.sessions.get(session_id)

# --- ITERATING all pages ---
for session in client.sessions.list_all(size=100):
    print(session.name)
```

## Generated Type Details

### Go Types (per resource)

```go
// Session represents an Ambient Platform Session resource.
// This type is GENERATED from openapi.yaml — do not edit.
type Session struct {
    // ObjectReference fields (embedded)
    ID        string     `json:"id,omitempty"`
    Kind      string     `json:"kind,omitempty"`
    Href      string     `json:"href,omitempty"`
    CreatedAt *time.Time `json:"created_at,omitempty"`
    UpdatedAt *time.Time `json:"updated_at,omitempty"`

    // Session-specific fields
    Name             string `json:"name"`
    RepoURL          string `json:"repo_url,omitempty"`
    Prompt           string `json:"prompt,omitempty"`
    CreatedByUserID  string `json:"created_by_user_id,omitempty"`
    AssignedUserID   string `json:"assigned_user_id,omitempty"`
    WorkflowID       string `json:"workflow_id,omitempty"`
}

// SessionBuilder constructs a Session for Create requests.
type SessionBuilder struct {
    session Session
    errors  []error
}

func NewSessionBuilder() *SessionBuilder {
    return &SessionBuilder{}
}

func (b *SessionBuilder) Name(name string) *SessionBuilder {
    b.session.Name = name
    return b
}

// ... one method per mutable field ...

func (b *SessionBuilder) Build() (*Session, error) {
    if b.session.Name == "" {
        b.errors = append(b.errors, fmt.Errorf("name is required"))
    }
    if len(b.errors) > 0 {
        return nil, fmt.Errorf("validation failed: %w", errors.Join(b.errors...))
    }
    return &b.session, nil
}

// SessionPatchBuilder constructs a SessionPatchRequest.
// Uses *string to distinguish "not set" from "set to empty string".
type SessionPatchBuilder struct {
    patch map[string]any
}

func NewSessionPatchBuilder() *SessionPatchBuilder {
    return &SessionPatchBuilder{patch: make(map[string]any)}
}

func (b *SessionPatchBuilder) Name(name string) *SessionPatchBuilder {
    b.patch["name"] = name
    return b
}

func (b *SessionPatchBuilder) Build() map[string]any {
    return b.patch
}
```

### Python Types (per resource)

```python
@dataclass(frozen=True)
class Session:
    """Ambient Platform Session resource.
    GENERATED from openapi.yaml — do not edit.
    """
    # ObjectReference fields
    id: str = ""
    kind: str = ""
    href: str = ""
    created_at: Optional[datetime] = None
    updated_at: Optional[datetime] = None

    # Session-specific fields
    name: str = ""
    repo_url: str = ""
    prompt: str = ""
    created_by_user_id: str = ""
    assigned_user_id: str = ""
    workflow_id: str = ""

    @classmethod
    def from_dict(cls, data: dict) -> "Session":
        return cls(
            id=data.get("id", ""),
            kind=data.get("kind", ""),
            href=data.get("href", ""),
            created_at=_parse_datetime(data.get("created_at")),
            updated_at=_parse_datetime(data.get("updated_at")),
            name=data.get("name", ""),
            repo_url=data.get("repo_url", ""),
            prompt=data.get("prompt", ""),
            created_by_user_id=data.get("created_by_user_id", ""),
            assigned_user_id=data.get("assigned_user_id", ""),
            workflow_id=data.get("workflow_id", ""),
        )

    @classmethod
    def builder(cls) -> "SessionBuilder":
        return SessionBuilder()


class SessionBuilder:
    """Fluent builder for Session creation requests."""

    def __init__(self):
        self._data: dict[str, Any] = {}

    def name(self, value: str) -> "SessionBuilder":
        self._data["name"] = value
        return self

    def prompt(self, value: str) -> "SessionBuilder":
        self._data["prompt"] = value
        return self

    # ... one method per mutable field ...

    def build(self) -> dict:
        if "name" not in self._data:
            raise ValueError("name is required")
        return dict(self._data)


class SessionPatch:
    """Fluent builder for Session PATCH requests."""

    def __init__(self):
        self._data: dict[str, Any] = {}

    def name(self, value: str) -> "SessionPatch":
        self._data["name"] = value
        return self

    # ... one method per patchable field ...

    def to_dict(self) -> dict:
        return dict(self._data)
```

### List Envelope and Pagination

```go
// ListMeta contains pagination metadata from list responses.
type ListMeta struct {
    Kind  string `json:"kind"`
    Page  int    `json:"page"`
    Size  int    `json:"size"`
    Total int    `json:"total"`
}

// SessionList is a paginated list of Sessions.
type SessionList struct {
    ListMeta
    Items []Session `json:"items"`
}

// ListOptions configures list/search/pagination parameters.
type ListOptions struct {
    Page    int
    Size    int
    Search  string
    OrderBy string
    Fields  string
}

// ListOptionsBuilder constructs ListOptions with fluent API.
type ListOptionsBuilder struct {
    opts ListOptions
}

func NewListOptions() *ListOptionsBuilder {
    return &ListOptionsBuilder{opts: ListOptions{Page: 1, Size: 100}}
}

func (b *ListOptionsBuilder) Page(page int) *ListOptionsBuilder {
    b.opts.Page = page
    return b
}

func (b *ListOptionsBuilder) Size(size int) *ListOptionsBuilder {
    if size > 65500 { size = 65500 }
    b.opts.Size = size
    return b
}

func (b *ListOptionsBuilder) Search(search string) *ListOptionsBuilder {
    b.opts.Search = search
    return b
}

func (b *ListOptionsBuilder) OrderBy(orderBy string) *ListOptionsBuilder {
    b.opts.OrderBy = orderBy
    return b
}

func (b *ListOptionsBuilder) Fields(fields string) *ListOptionsBuilder {
    b.opts.Fields = fields
    return b
}

func (b *ListOptionsBuilder) Build() *ListOptions {
    return &b.opts
}
```

## Generated Client Methods (per resource)

### Go Resource API

```go
// SessionAPI provides CRUD operations for Sessions.
// GENERATED from openapi.yaml — do not edit.
type SessionAPI struct {
    client *Client
}

func (c *Client) Sessions() *SessionAPI {
    return &SessionAPI{client: c}
}

func (a *SessionAPI) Create(ctx context.Context, session *Session) (*Session, error) {
    body, err := json.Marshal(session)
    if err != nil {
        return nil, fmt.Errorf("marshal session: %w", err)
    }
    var result Session
    err = a.client.do(ctx, http.MethodPost, "/sessions", body, http.StatusCreated, &result)
    return &result, err
}

func (a *SessionAPI) Get(ctx context.Context, id string) (*Session, error) {
    var result Session
    err := a.client.do(ctx, http.MethodGet, "/sessions/"+id, nil, http.StatusOK, &result)
    return &result, err
}

func (a *SessionAPI) List(ctx context.Context, opts *ListOptions) (*SessionList, error) {
    var result SessionList
    err := a.client.doWithQuery(ctx, http.MethodGet, "/sessions", nil, http.StatusOK, &result, opts)
    return &result, err
}

func (a *SessionAPI) Update(ctx context.Context, id string, patch map[string]any) (*Session, error) {
    body, err := json.Marshal(patch)
    if err != nil {
        return nil, fmt.Errorf("marshal patch: %w", err)
    }
    var result Session
    err = a.client.do(ctx, http.MethodPatch, "/sessions/"+id, body, http.StatusOK, &result)
    return &result, err
}

// ListAll returns an iterator that fetches all pages.
func (a *SessionAPI) ListAll(ctx context.Context, opts *ListOptions) *Iterator[Session] {
    return NewIterator[Session](func(page int) (*SessionList, error) {
        o := *opts
        o.Page = page
        return a.List(ctx, &o)
    })
}
```

### Python Resource API (mixin pattern)

```python
class SessionAPI:
    """GENERATED from openapi.yaml — do not edit."""

    def create(self, data: dict) -> Session:
        resp = self._client._request("POST", "/sessions", json=data)
        return Session.from_dict(resp)

    def get(self, session_id: str) -> Session:
        resp = self._client._request("GET", f"/sessions/{session_id}")
        return Session.from_dict(resp)

    def list(self, opts: Optional[ListOptions] = None) -> SessionList:
        resp = self._client._request("GET", "/sessions", params=opts.to_params() if opts else None)
        return SessionList.from_dict(resp)

    def update(self, session_id: str, patch) -> Session:
        data = patch.to_dict() if hasattr(patch, 'to_dict') else patch
        resp = self._client._request("PATCH", f"/sessions/{session_id}", json=data)
        return Session.from_dict(resp)

    def list_all(self, **kwargs) -> Iterator[Session]:
        """Iterate all pages."""
        page = 1
        size = kwargs.get("size", 100)
        while True:
            result = self.list(ListOptions().page(page).size(size))
            yield from result.items
            if page * size >= result.total:
                break
            page += 1
```

## Hand-Written Boundary

The following files are **never generated** and contain security-critical, SDK-specific logic:

### Go — `client/client.go` (hand-written)

```go
type Client struct {
    baseURL    string
    basePath   string   // default: "/api/ambient-api-server/v1"
    token      SecureToken
    project    string
    httpClient *http.Client
    logger     *slog.Logger
}

// do executes an HTTP request. ALL generated *API types call this.
func (c *Client) do(ctx context.Context, method, path string, body []byte, expectedStatus int, result any) error { ... }

// doWithQuery adds query parameters (ListOptions) to the request.
func (c *Client) doWithQuery(ctx context.Context, method, path string, body []byte, expectedStatus int, result any, opts *ListOptions) error { ... }

// Functional options for client construction
func WithBasePath(path string) ClientOption { ... }
func WithTimeout(d time.Duration) ClientOption { ... }
func WithHTTPClient(c *http.Client) ClientOption { ... }
```

### Python — `client.py` (hand-written)

```python
class AmbientClient:
    def __init__(self, base_url, token, project, *, base_path="/api/ambient-api-server/v1", timeout=30.0): ...
    def _request(self, method, path, **kwargs) -> dict: ...  # ALL generated APIs call this

    @property
    def sessions(self) -> SessionAPI: ...
    @property
    def agents(self) -> AgentAPI: ...
    # ... one property per resource, lazily initialized

    @classmethod
    def from_env(cls, **kwargs) -> "AmbientClient": ...
```

## Generator Implementation

### Intermediate Representation

The generator parses OpenAPI YAML into a language-neutral IR:

```go
type Resource struct {
    Name           string     // "Session"
    Plural         string     // "sessions"
    PathSegment    string     // "sessions" (from URL path)
    Fields         []Field    // resource-specific fields (not ObjectReference)
    RequiredFields []string   // fields listed in `required`
    HasDelete      bool       // only User currently
}

type Field struct {
    Name       string    // "created_by_user_id"
    GoName     string    // "CreatedByUserID"
    PythonName string    // "created_by_user_id"
    Type       string    // "string", "integer"
    Format     string    // "date-time", "int32", ""
    GoType     string    // "string", "int32", "*time.Time"
    PythonType string    // "str", "int", "Optional[datetime]"
    Required   bool
    JSONTag    string    // `json:"created_by_user_id,omitempty"`
}
```

### Type Mapping

| OpenAPI Type + Format     | Go Type      | Python Type           |
|--------------------------|--------------|----------------------|
| `string`                  | `string`     | `str`                |
| `string` + `date-time`    | `*time.Time` | `Optional[datetime]` |
| `integer`                 | `int`        | `int`                |
| `integer` + `int32`       | `int32`      | `int`                |
| `boolean`                 | `bool`       | `bool`               |
| `array` of `string`       | `[]string`   | `list[str]`          |

### Template Execution

```
for each resource in parsed_spec:
    execute go/types.go.tmpl   → go-sdk/types/{resource_snake}.go
    execute go/client.go.tmpl  → go-sdk/client/{resource_snake}_api.go
    execute py/types.py.tmpl   → python-sdk/ambient_platform/{resource_snake}.py
    execute py/client.py.tmpl  → python-sdk/ambient_platform/_{resource_snake}_api.py

execute go/base.go.tmpl       → go-sdk/types/base.go
execute go/iterator.go.tmpl   → go-sdk/client/iterator.go
execute py/base.py.tmpl       → python-sdk/ambient_platform/_base.py
execute py/iterator.py.tmpl   → python-sdk/ambient_platform/_iterator.py
```

### Generated File Header

Every generated file starts with:

```
// Code generated by ambient-sdk-generator from openapi.yaml — DO NOT EDIT.
// Source: ../ambient-api-server/openapi/openapi.yaml
// Generated: 2026-02-14T17:00:00Z
```

This follows Go convention (`go generate` tools) and makes it obvious which files are safe to hand-edit.

## Build Integration

### Makefile target

```makefile
.PHONY: generate-sdk
generate-sdk:
	@echo "Generating SDK from OpenAPI spec..."
	cd generator && go run . \
		-spec ../ambient-api-server/openapi/openapi.yaml \
		-go-out ../go-sdk \
		-python-out ../python-sdk
	cd go-sdk && go fmt ./...
	cd python-sdk && black ambient_platform/ && isort ambient_platform/
	@echo "SDK generated successfully. Run tests to verify."

.PHONY: verify-sdk
verify-sdk: generate-sdk
	cd go-sdk && go test ./...
	cd python-sdk && pytest
```

### CI check (prevent drift)

```yaml
# In GitHub Actions
- name: Verify SDK is up to date
  run: |
    make generate-sdk
    git diff --exit-code go-sdk/ python-sdk/
    # Fails if generated output differs from committed code
```

## Error Handling

### Generated APIError type

```go
type APIError struct {
    // ObjectReference fields
    ID        string `json:"id,omitempty"`
    Kind      string `json:"kind,omitempty"`
    Href      string `json:"href,omitempty"`

    // Error-specific fields
    Code        string `json:"code"`
    Reason      string `json:"reason"`
    OperationID string `json:"operation_id,omitempty"`

    // HTTP metadata (not from JSON — set by client)
    StatusCode int `json:"-"`
}

func (e *APIError) Error() string {
    return fmt.Sprintf("ambient API error %d: %s — %s", e.StatusCode, e.Code, e.Reason)
}
```

```python
@dataclass(frozen=True)
class APIError(Exception):
    """Structured API error from the Ambient Platform."""
    status_code: int
    code: str
    reason: str
    operation_id: str = ""
    id: str = ""
    kind: str = ""
    href: str = ""

    def __str__(self) -> str:
        return f"ambient API error {self.status_code}: {self.code} — {self.reason}"
```

## Pagination Iterator

### Go (generic, requires Go 1.21+)

```go
type Listable[T any] interface {
    GetItems() []T
    GetTotal() int
    GetPage() int
    GetSize() int
}

type Iterator[T any] struct {
    fetchPage func(page int) (Listable[T], error)
    items     []T
    index     int
    page      int
    total     int
    done      bool
    err       error
}

func (it *Iterator[T]) Next() bool {
    if it.done || it.err != nil {
        return false
    }
    it.index++
    if it.index < len(it.items) {
        return true
    }
    // Fetch next page
    it.page++
    result, err := it.fetchPage(it.page)
    if err != nil {
        it.err = err
        return false
    }
    it.items = result.GetItems()
    it.index = 0
    it.total = result.GetTotal()
    if len(it.items) == 0 {
        it.done = true
        return false
    }
    return true
}

func (it *Iterator[T]) Item() T  { return it.items[it.index] }
func (it *Iterator[T]) Err() error { return it.err }
```

### Python (generator-based)

```python
def paginate(fetch_page, size=100):
    """Generic pagination iterator."""
    page = 1
    while True:
        result = fetch_page(page=page, size=size)
        yield from result.items
        if page * size >= result.total:
            break
        page += 1
```

## Decisions (resolved)

1. **Generator language: Go.** Consistent with the project. Uses `text/template` (same as K8s code generators), tested with `go test`.

2. **DELETE scope: follow the OpenAPI spec exactly.** The generator emits Delete methods only when the spec declares a DELETE operation on a resource. Currently only User has DELETE. When other resources gain DELETE in the spec, the generator picks it up automatically.

3. **Async Python client: yes.** The generator emits both sync (`httpx.Client`) and async (`httpx.AsyncClient`) variants. Async client lives in `_async_client.py` with `AsyncAmbientClient` and async resource API classes. Both share the same generated types and builders.

4. **Version pinning: yes, SHA256 hash.** The generated file header includes both the timestamp and a SHA256 hash of the concatenated OpenAPI spec files. CI can detect drift by comparing the embedded hash against the current spec hash without re-running the generator.

   ```
   // Code generated by ambient-sdk-generator from openapi.yaml — DO NOT EDIT.
   // Source: ../ambient-api-server/openapi/openapi.yaml
   // Spec SHA256: a1b2c3d4e5f6...
   // Generated: 2026-02-14T17:00:00Z
   ```

5. **Field grouping: keep flat, readOnly self-manages.** When Session expands to ~30 fields, status/runtime fields are `readOnly` in the spec so they appear only on response types, never on builders. Builders stay clean with only mutable fields. No grouping needed.

## Testing Strategy

The SDK has three testing tiers: generator tests, SDK unit tests, and end-to-end tests. The e2e tier is the most valuable — the SDK is the natural e2e harness for the entire platform because it exercises the full stack: **SDK → API server → Postgres → control plane → operator → runner pod**.

Every other component unit-tests itself (`go test ./...` in api-server, cp, operator). The SDK's e2e suite validates that the assembled system works as a whole.

### Tier 1: Generator Tests (CI, no server)

Golden-file tests that validate the generator itself produces correct output.

```
generator/
└── generator_test.go
```

| Test | What it validates |
|------|-------------------|
| `TestParseOpenAPISpec` | Parser resolves `$ref`, `allOf`, extracts all 8 resources with correct fields |
| `TestGenerateGoTypes` | Generated Go types match golden files (field names, JSON tags, types) |
| `TestGeneratePythonTypes` | Generated Python dataclasses match golden files |
| `TestGenerateGoClient` | Generated API methods match golden files (method signatures, paths) |
| `TestGeneratePythonClient` | Generated Python API methods match golden files |
| `TestBuilderValidation` | Generated builders enforce `required` fields from spec |
| `TestDeleteOnlyWhenDeclared` | Only User gets `Delete()` method; others don't |
| `TestSpecHash` | SHA256 hash in header matches actual spec content |

**How golden-file tests work:**

```bash
# Update golden files after intentional changes:
cd generator && go test ./... -update

# CI runs without -update — any diff = failure:
cd generator && go test ./...
```

### Tier 2: SDK Unit Tests (CI, no server)

Test the generated SDK code in isolation using mocked HTTP responses.

#### Go SDK

```
go-sdk/
├── types/
│   ├── base_test.go           # ObjectReference, ListMeta, APIError serialization
│   ├── session_test.go        # Session JSON round-trip, SessionBuilder, SessionPatchBuilder
│   ├── agent_test.go          # Agent JSON round-trip, AgentBuilder
│   ├── ...                    # One per resource (generated from template)
│   └── list_options_test.go   # Query param encoding, size capping at 65500
├── client/
│   ├── client_test.go         # Auth validation, token format rejection, SecureToken redaction (hand-written)
│   └── iterator_test.go       # Multi-page iteration, empty page stop, error propagation
```

#### Python SDK

```
python-sdk/
├── tests/
│   ├── test_types.py          # from_dict round-trip for all resources, datetime parsing
│   ├── test_builders.py       # Required field enforcement, Build() errors, Patch sparse output
│   ├── test_list_options.py   # to_params() encoding, size capping
│   ├── test_client.py         # Auth validation, token rejection, from_env() (hand-written)
│   ├── test_iterator.py       # Pagination generator, empty page, error propagation
│   └── test_error_parsing.py  # JSON error body → APIError with code, reason, operation_id
```

#### What unit tests cover per resource (generated from template)

| Test | Go | Python |
|------|-----|--------|
| JSON round-trip: marshal → unmarshal preserves all fields | `Session{Name:"x"} → json → Session{Name:"x"}` | `Session.from_dict(session.to_dict()) == session` |
| Builder enforces required fields | `NewSessionBuilder().Build()` → error "name is required" | `Session.builder().build()` → `ValueError` |
| Builder sets all fields | `NewSessionBuilder().Name("x").Prompt("y").Build()` → `Session{Name:"x", Prompt:"y"}` | equivalent |
| PatchBuilder emits only set fields | `NewSessionPatchBuilder().Name("x").Build()` → `{"name":"x"}` (no other keys) | equivalent |
| ObjectReference fields present on response | Unmarshal JSON with `id`, `kind`, `href`, `created_at`, `updated_at` | equivalent |
| from_dict ignores unknown fields | Extra JSON keys don't cause errors | equivalent |

#### Generated vs hand-written test split

- **Generated** (from templates, one per resource): type round-trip, builder validation, patch sparseness — identical coverage for every resource automatically
- **Hand-written** (security-critical, never generated): `client_test.go` / `test_client.py` — token format validation, placeholder detection, SecureToken log redaction, URL validation, error response sanitization

### Tier 3: End-to-End Tests (requires live server)

The SDK e2e suite is the **platform-wide integration test**. It creates real resources via the SDK, verifies they propagate through the full stack, and cleans up.

```
ambient-sdk/
├── e2e/
│   ├── e2e_test.go            # Go e2e suite
│   ├── e2e_test.py            # Python e2e suite (identical scenarios)
│   └── README.md              # Setup instructions, env vars, CI config
```

#### Environment

```bash
export AMBIENT_API_URL="https://ambient-api.apps.cluster.example.com"
export AMBIENT_TOKEN="sha256~..."
export AMBIENT_PROJECT="e2e-test-project"
```

#### E2E Test Scenarios

| # | Scenario | What it validates across the stack |
|---|----------|-----------------------------------|
| 1 | **Session lifecycle** | Create → Get → List (appears) → Patch name → Get (updated) → verify `updated_at` changed |
| 2 | **Session with all fields** | Create with name + prompt + repo_url + assigned_user_id → Get → all fields round-trip |
| 3 | **Pagination** | Create 5 sessions → List with size=2 → verify 3 pages → ListAll iterator returns all 5 |
| 4 | **Search and ordering** | Create sessions with distinct names → List with search filter → verify filtered results → OrderBy created_at desc |
| 5 | **Error: missing required field** | Create session without name → expect 400 with structured `APIError{code, reason}` |
| 6 | **Error: not found** | Get session with fake ID → expect 404 |
| 7 | **Error: bad auth** | Create client with invalid token → any request → expect 401 |
| 8 | **Project isolation** | Create session in project A → List in project B → session not visible |
| 9 | **Workflow lifecycle** | Create agent → Create workflow with agent_id → Get workflow → verify agent_id set |
| 10 | **WorkflowSkill/WorkflowTask** | Create skill → Create workflow → Create WorkflowSkill linking them → List WorkflowSkills → verify position |
| 11 | **User CRUD + DELETE** | Create user → Get → Patch → Delete → Get → 404 (only resource with DELETE) |
| 12 | **Control plane propagation** | Create session → poll until status changes from `pending` → verify CP picked it up and operator created a Job |
| 13 | **Builder validation client-side** | Verify builder rejects invalid input before HTTP call (no network round-trip for obvious errors) |
| 14 | **Concurrent creates** | 10 goroutines/threads create sessions simultaneously → all succeed → List returns all 10 |

#### E2E Test Structure (Go)

```go
func TestE2E(t *testing.T) {
    client, err := ambient.NewClientFromEnv()
    if err != nil {
        t.Skipf("e2e: skipping, no live server: %v", err)
    }

    t.Run("SessionLifecycle", func(t *testing.T) {
        ctx := context.Background()

        session, err := client.Sessions().Create(ctx,
            ambient.NewSessionBuilder().
                Name("e2e-lifecycle-"+randomSuffix()).
                Prompt("e2e test session").
                Build(),
        )
        require.NoError(t, err)
        require.NotEmpty(t, session.ID)

        t.Cleanup(func() {
            // best-effort cleanup — sessions don't have DELETE yet
        })

        got, err := client.Sessions().Get(ctx, session.ID)
        require.NoError(t, err)
        assert.Equal(t, session.Name, got.Name)

        list, err := client.Sessions().List(ctx, ambient.NewListOptions().Search("name='"+session.Name+"'").Build())
        require.NoError(t, err)
        assert.GreaterOrEqual(t, list.Total, 1)

        updated, err := client.Sessions().Update(ctx, session.ID,
            ambient.NewSessionPatchBuilder().
                Prompt("updated prompt").
                Build(),
        )
        require.NoError(t, err)
        assert.Equal(t, "updated prompt", updated.Prompt)
    })
}
```

#### E2E Test Structure (Python)

```python
@pytest.fixture
def client():
    try:
        c = AmbientClient.from_env()
    except ValueError:
        pytest.skip("e2e: no live server configured")
    with c:
        yield c

def test_session_lifecycle(client):
    name = f"e2e-lifecycle-{uuid4().hex[:8]}"

    session = client.sessions.create(
        Session.builder()
            .name(name)
            .prompt("e2e test session")
            .build()
    )
    assert session.id

    got = client.sessions.get(session.id)
    assert got.name == name

    sessions = client.sessions.list(
        ListOptions().search(f"name='{name}'")
    )
    assert sessions.total >= 1

    updated = client.sessions.update(session.id,
        SessionPatch().prompt("updated prompt")
    )
    assert updated.prompt == "updated prompt"
```

#### CI Integration

```yaml
# Unit tests (every PR)
- name: SDK unit tests
  run: |
    cd go-sdk && go test ./...
    cd python-sdk && pytest tests/ -v

# E2E tests (nightly or on-demand against staging)
- name: SDK e2e tests
  if: github.event_name == 'schedule' || contains(github.event.pull_request.labels.*.name, 'e2e')
  env:
    AMBIENT_API_URL: ${{ secrets.STAGING_API_URL }}
    AMBIENT_TOKEN: ${{ secrets.STAGING_TOKEN }}
    AMBIENT_PROJECT: e2e-test-project
  run: |
    cd e2e && go test -v -timeout 5m ./...
    cd e2e && pytest e2e_test.py -v --timeout=300
```

#### E2E as platform health check

The e2e suite doubles as a **platform health check**. Running it against a live deployment answers:

- Is the API server accepting requests?
- Is authentication working?
- Is project scoping enforced?
- Is the control plane picking up new sessions?
- Is the operator creating Jobs?
- Are all 8 resource types functioning?
- Is pagination correct?
- Are error responses structured correctly?

This makes the SDK e2e suite the **baseline smoke test** that Overlord requested — run it before and after any deployment to verify the platform is healthy.
