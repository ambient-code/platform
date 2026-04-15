# Project Intelligence Memory — Phase 1 Implementation Plan

**Status**: Approved
**Author**: Yossi Ovadia
**Date**: 2026-04-09
**Design Doc**: [project-intelligence-memory.md](project-intelligence-memory.md)

---

## Scope

Phase 1 only. Three PostgreSQL tables, REST API, three MCP tools, auto-analysis trigger, frontend surface. No CRD changes, no operator changes, no legacy backend changes.

## Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| API location | ambient-api-server (plugin pattern) | Code generator, TSL search, RBAC, PostgreSQL native. Legacy backend is being sunset. |
| Auto-analysis | Hidden prompt to running agent | Follows existing `_trigger_repo_added_notification` pattern. Agent does the analysis naturally. |
| Route style | Flat `/api/ambient/v1/repo_intelligences` with `?project_id=X` query params | Matches api-server conventions. Project scoping via `common.ApplyProjectScope`. |
| Repo URL in queries | Query parameter `?repo_url=X` (URL-encoded) | Repo URLs contain slashes — can't be path segments. |

---

## Layer 1: Data Model (ambient-api-server)

### 1.1 `repo_intelligences` — Per-repo knowledge store

```go
// plugins/repoIntelligences/model.go
package repoIntelligences

import (
    "time"
    "gorm.io/gorm"
    api "github.com/openshift-online/rh-trex-ai/pkg/api"
)

type RepoIntelligence struct {
    api.Meta // ID, CreatedAt, UpdatedAt, DeletedAt

    // Scoping
    ProjectID string `json:"project_id" gorm:"not null;index;uniqueIndex:idx_project_repo"`
    RepoURL   string `json:"repo_url"   gorm:"not null;index;uniqueIndex:idx_project_repo"`
    RepoBranch string `json:"repo_branch" gorm:"not null;default:'main'"`

    // Content
    Summary     string  `json:"summary"      gorm:"type:text;not null"`          // Markdown overview
    Language    string  `json:"language"     gorm:"not null"`                     // Primary language
    Framework   *string `json:"framework,omitempty"`                              // e.g. "gin", "nextjs"
    BuildSystem *string `json:"build_system,omitempty"`                           // e.g. "make", "npm"
    TestStrategy *string `json:"test_strategy,omitempty" gorm:"type:text"`        // How tests are run
    Architecture *string `json:"architecture,omitempty"  gorm:"type:text"`        // Module/package layout
    Conventions  *string `json:"conventions,omitempty"   gorm:"type:text"`        // Coding patterns
    Dependencies *string `json:"dependencies,omitempty"  gorm:"type:text"`        // Key deps + versions
    Caveats      *string `json:"caveats,omitempty"       gorm:"type:text"`        // Known footguns

    // Metadata
    AnalyzedBySessionID *string    `json:"analyzed_by_session_id,omitempty" gorm:"index"`
    AnalyzedByAgentID   *string    `json:"analyzed_by_agent_id,omitempty"`
    AnalyzedAt          *time.Time `json:"analyzed_at,omitempty"`
    Confidence          *float64   `json:"confidence,omitempty"`           // 0.0–1.0
    Version             int        `json:"version"     gorm:"not null;default:1"`
}

type RepoIntelligenceList []*RepoIntelligence

func (d *RepoIntelligence) BeforeCreate(tx *gorm.DB) error {
    d.ID = api.NewID()
    if d.Version == 0 {
        d.Version = 1
    }
    return nil
}

// Patch request — all pointer fields
type RepoIntelligencePatchRequest struct {
    Summary      *string  `json:"summary,omitempty"`
    Language     *string  `json:"language,omitempty"`
    Framework    *string  `json:"framework,omitempty"`
    BuildSystem  *string  `json:"build_system,omitempty"`
    TestStrategy *string  `json:"test_strategy,omitempty"`
    Architecture *string  `json:"architecture,omitempty"`
    Conventions  *string  `json:"conventions,omitempty"`
    Dependencies *string  `json:"dependencies,omitempty"`
    Caveats      *string  `json:"caveats,omitempty"`
    Confidence   *float64 `json:"confidence,omitempty"`
}
```

### 1.2 `repo_findings` — File-scoped knowledge

```go
// plugins/repoFindings/model.go
package repoFindings

import (
    "gorm.io/gorm"
    api "github.com/openshift-online/rh-trex-ai/pkg/api"
)

type RepoFinding struct {
    api.Meta

    // Parent
    IntelligenceID string `json:"intelligence_id" gorm:"not null;index"`

    // Scope
    FilePath  string `json:"file_path"  gorm:"not null;index"` // e.g. "components/backend/handlers/sessions.go"
    Category  string `json:"category"   gorm:"not null;index"` // investigation, caveat, review, convention
    Status    string `json:"status"     gorm:"not null;default:'active';index"` // active, resolved, retracted

    // Content
    Title      string   `json:"title"      gorm:"not null"`
    Body       string   `json:"body"       gorm:"type:text;not null"` // Markdown detail
    Severity   *string  `json:"severity,omitempty"`                    // info, warning, critical
    Confidence *float64 `json:"confidence,omitempty"`

    // Provenance
    SourceType string  `json:"source_type" gorm:"not null"` // agent_analysis, investigation, review, user
    SourceRef  *string `json:"source_ref,omitempty"`         // "session:abc", "pr:456", "issue:PROJ-789"
    SessionID  *string `json:"session_id,omitempty"  gorm:"index"`
    AgentID    *string `json:"agent_id,omitempty"`

    // Resolution
    ResolvedBy    *string `json:"resolved_by,omitempty"`     // session ID or user that resolved
    ResolvedReason *string `json:"resolved_reason,omitempty"` // why it was resolved/retracted
}

type RepoFindingList []*RepoFinding

func (d *RepoFinding) BeforeCreate(tx *gorm.DB) error {
    d.ID = api.NewID()
    if d.Status == "" {
        d.Status = "active"
    }
    return nil
}

type RepoFindingPatchRequest struct {
    Status         *string `json:"status,omitempty"`
    Severity       *string `json:"severity,omitempty"`
    ResolvedBy     *string `json:"resolved_by,omitempty"`
    ResolvedReason *string `json:"resolved_reason,omitempty"`
}
```

### 1.3 `repo_events` — Audit trail (append-only)

```go
// plugins/repoEvents/model.go
package repoEvents

import (
    "gorm.io/gorm"
    api "github.com/openshift-online/rh-trex-ai/pkg/api"
)

type RepoEvent struct {
    api.Meta

    // What changed
    ResourceType string `json:"resource_type" gorm:"not null;index"` // "intelligence" or "finding"
    ResourceID   string `json:"resource_id"   gorm:"not null;index"`
    Action       string `json:"action"        gorm:"not null"`       // created, updated, resolved, retracted

    // Who
    ActorType string `json:"actor_type" gorm:"not null"` // session, agent, user, system
    ActorID   string `json:"actor_id"   gorm:"not null"`

    // Context
    ProjectID string  `json:"project_id" gorm:"not null;index"`
    Reason    *string `json:"reason,omitempty"`
    Diff      *string `json:"diff,omitempty" gorm:"type:text"` // JSON: what fields changed
}

type RepoEventList []*RepoEvent

func (d *RepoEvent) BeforeCreate(tx *gorm.DB) error {
    d.ID = api.NewID()
    return nil
}
```

### 1.4 Database Migrations

```go
// plugins/repoIntelligences/migration.go
func migration() *gormigrate.Migration {
    return &gormigrate.Migration{
        ID: "202604091200",
        Migrate: func(tx *gorm.DB) error {
            type RepoIntelligence struct { /* frozen copy */ }
            if err := tx.AutoMigrate(&RepoIntelligence{}); err != nil {
                return err
            }
            // Composite unique index: one intelligence per project+repo
            return tx.Exec(
                "CREATE UNIQUE INDEX IF NOT EXISTS idx_project_repo ON repo_intelligences(project_id, repo_url)",
            ).Error
        },
        Rollback: func(tx *gorm.DB) error {
            return tx.Migrator().DropTable("repo_intelligences")
        },
    }
}

// plugins/repoFindings/migration.go — ID: "202604091201"
// plugins/repoEvents/migration.go  — ID: "202604091202"
```

---

## Layer 2: REST API (ambient-api-server)

### 2.1 Standard Plugin Routes

Each plugin registers standard CRUD routes following the agents pattern:

```
# RepoIntelligence — per-repo knowledge
GET    /api/ambient/v1/repo_intelligences                 List (supports ?project_id=X&search=...)
POST   /api/ambient/v1/repo_intelligences                 Create or upsert
GET    /api/ambient/v1/repo_intelligences/{id}             Get by ID
PATCH  /api/ambient/v1/repo_intelligences/{id}             Update fields
DELETE /api/ambient/v1/repo_intelligences/{id}             Soft-delete

# RepoFinding — file-scoped findings
GET    /api/ambient/v1/repo_findings                       List (supports ?intelligence_id=X&file_path=Y)
POST   /api/ambient/v1/repo_findings                       Create
GET    /api/ambient/v1/repo_findings/{id}                   Get by ID
PATCH  /api/ambient/v1/repo_findings/{id}                   Update (status, resolved)
DELETE /api/ambient/v1/repo_findings/{id}                   Soft-delete

# RepoEvent — audit log (read-only)
GET    /api/ambient/v1/repo_events                         List (supports ?project_id=X&resource_id=Y)
GET    /api/ambient/v1/repo_events/{id}                     Get by ID
```

### 2.2 Custom Endpoints

```
# Lookup by project + repo URL (convenience for MCP tools)
GET    /api/ambient/v1/repo_intelligences/lookup?project_id=X&repo_url=Y
       → Returns single intelligence or 404

# Subresource: findings for an intelligence
GET    /api/ambient/v1/repo_intelligences/{id}/findings
       → Injects intelligence_id filter into search, returns RepoFindingList

# Context endpoint: condensed summary for session injection
GET    /api/ambient/v1/repo_intelligences/context?project_id=X&repo_urls=a,b,c&max_entries=20
       → Returns { intelligences: [...], findings: [...], injected_context: "markdown..." }
```

### 2.3 Handler Patterns

**Lookup handler** (new, added to `handler.go`):

```go
func (h repoIntelligenceHandler) Lookup(w http.ResponseWriter, r *http.Request) {
    cfg := &handlers.HandlerConfig{
        Action: func() (interface{}, *errors.ServiceError) {
            projectID := r.URL.Query().Get("project_id")
            repoURL := r.URL.Query().Get("repo_url")
            if projectID == "" || repoURL == "" {
                return nil, errors.BadRequest("project_id and repo_url are required")
            }
            intel, err := h.service.GetByProjectAndRepo(r.Context(), projectID, repoURL)
            if err != nil {
                return nil, err
            }
            return PresentRepoIntelligence(intel), nil
        },
        ErrorHandler: handlers.HandleError,
    }
    handlers.HandleGet(w, r, cfg)
}
```

**Findings subresource handler** (follows `agents/subresource_handler.go` pattern):

```go
func (h repoIntelligenceHandler) ListFindings(w http.ResponseWriter, r *http.Request) {
    id := mux.Vars(r)["id"]
    listArgs := services.NewListArguments(r.URL.Query())

    // Inject parent filter into search
    filter := fmt.Sprintf("intelligence_id = '%s'", id)
    if listArgs.Search != "" {
        listArgs.Search = filter + " and (" + listArgs.Search + ")"
    } else {
        listArgs.Search = filter
    }

    var findings repoFindings.RepoFindingList
    paging, err := h.findingsSvc.List(r.Context(), "id", listArgs, &findings)
    // ... present and return
}
```

**Context handler** (custom aggregation):

```go
func (h repoIntelligenceHandler) Context(w http.ResponseWriter, r *http.Request) {
    cfg := &handlers.HandlerConfig{
        Action: func() (interface{}, *errors.ServiceError) {
            projectID := r.URL.Query().Get("project_id")
            repoURLs := strings.Split(r.URL.Query().Get("repo_urls"), ",")
            maxEntries, _ := strconv.Atoi(r.URL.Query().Get("max_entries"))
            if maxEntries == 0 { maxEntries = 20 }

            ctx := r.Context()
            var allIntel []openapi.RepoIntelligence
            var allFindings []openapi.RepoFinding

            for _, repoURL := range repoURLs {
                intel, err := h.service.GetByProjectAndRepo(ctx, projectID, strings.TrimSpace(repoURL))
                if err != nil { continue } // skip repos without intelligence
                allIntel = append(allIntel, PresentRepoIntelligence(intel))

                findings, _, _ := h.findingsSvc.ListByIntelligenceID(ctx, intel.ID, maxEntries)
                for _, f := range findings {
                    allFindings = append(allFindings, repoFindings.PresentRepoFinding(f))
                }
            }

            return map[string]interface{}{
                "intelligences":   allIntel,
                "findings":        allFindings,
                "injected_context": buildInjectedContext(allIntel, allFindings),
            }, nil
        },
        ErrorHandler: handlers.HandleError,
    }
    handlers.HandleGet(w, r, cfg)
}
```

### 2.4 Plugin Registration

```go
// plugins/repoIntelligences/plugin.go
func init() {
    registry.RegisterService("RepoIntelligences", func(env *environments.Env) interface{} {
        return ServiceLocator(func() RepoIntelligenceService {
            return NewSQLRepoIntelligenceService(
                env.Database.SessionFactory,
                env.Services.Events(),
            )
        })
    })

    pkgserver.RegisterRoutes("repo_intelligences", func(
        router *mux.Router,
        services *environments.Services,
        jwt *auth.JWTMiddleware,
        authz auth.Authorization,
    ) {
        svc := Service(services)
        findingsSvc := repoFindings.Service(services)
        genericSvc := services.Generic()

        handler := NewRepoIntelligenceHandler(svc, findingsSvc, genericSvc)

        router.HandleFunc("", handler.List).Methods(http.MethodGet)
        router.HandleFunc("", handler.Create).Methods(http.MethodPost)
        router.HandleFunc("/lookup", handler.Lookup).Methods(http.MethodGet)
        router.HandleFunc("/context", handler.Context).Methods(http.MethodGet)
        router.HandleFunc("/{id}", handler.Get).Methods(http.MethodGet)
        router.HandleFunc("/{id}", handler.Patch).Methods(http.MethodPatch)
        router.HandleFunc("/{id}", handler.Delete).Methods(http.MethodDelete)
        router.HandleFunc("/{id}/findings", handler.ListFindings).Methods(http.MethodGet)

        router.Use(jwt.AuthenticateAccountJWT)
        if dbAuthz := pkgrbac.Middleware(services); dbAuthz != nil {
            router.Use(dbAuthz)
        } else {
            router.Use(authz.AuthorizeApi)
        }
    })

    pkgserver.RegisterController("RepoIntelligences", controllers.ControllerConfig{
        Source: "RepoIntelligences",
        Handlers: map[api.EventType][]controllers.ControllerHandlerFunc{
            api.CreateEventType: {func(ctx context.Context, id string) error {
                // Log audit event on create
                return Service(envSvc).OnCreate(ctx, id)
            }},
            api.UpdateEventType: {func(ctx context.Context, id string) error {
                return Service(envSvc).OnUpdate(ctx, id)
            }},
        },
    })

    presenters.RegisterPath(RepoIntelligence{}, "repo_intelligences")
    presenters.RegisterKind(RepoIntelligence{}, "RepoIntelligence")
    presenters.RegisterPath(&RepoIntelligence{}, "repo_intelligences")
    presenters.RegisterKind(&RepoIntelligence{}, "RepoIntelligence")

    db.RegisterMigration(migration())
}
```

### 2.5 Service Layer — Audit Event Creation

```go
// plugins/repoIntelligences/service.go

func (s *sqlService) Create(ctx context.Context, intel *RepoIntelligence) (*RepoIntelligence, *errors.ServiceError) {
    intel, err := s.dao.Create(ctx, intel)
    if err != nil {
        return nil, services.HandleCreateError("RepoIntelligence", err)
    }

    // Create framework event (drives controller pipeline)
    s.events.Create(ctx, &api.Event{
        Source: "RepoIntelligences", SourceID: intel.ID,
        EventType: api.CreateEventType,
    })

    // Create audit event in repo_events table
    eventSvc := repoEvents.Service(s.envServices)
    if eventSvc != nil {
        eventSvc.Create(ctx, &repoEvents.RepoEvent{
            ResourceType: "intelligence",
            ResourceID:   intel.ID,
            Action:       "created",
            ActorType:    resolveActorType(ctx),
            ActorID:      resolveActorID(ctx),
            ProjectID:    intel.ProjectID,
        })
    }

    return intel, nil
}
```

---

## Layer 3: MCP Tools (ambient-runner)

### 3.1 Intelligence API Client

```python
# ambient_runner/tools/intelligence_api.py

"""Client for the repo intelligence API (ambient-api-server)."""

import json
import logging
import os
import urllib.request
from typing import Any, Dict, List, Optional

from ambient_runner.platform.utils import get_bot_token

logger = logging.getLogger(__name__)


class IntelligenceAPIClient:
    """Client for repo intelligence endpoints on the api-server."""

    def __init__(self, api_server_url: str | None = None, project_id: str | None = None):
        self.api_server_url = (
            api_server_url or os.getenv("API_SERVER_URL", "")
        ).rstrip("/")
        self.project_id = (
            project_id
            or os.getenv("PROJECT_NAME")
            or os.getenv("AGENTIC_SESSION_NAMESPACE", "")
        ).strip()

        if not self.api_server_url:
            raise ValueError("API_SERVER_URL environment variable is required")

    def _make_request(self, method: str, path: str, data: dict | None = None) -> dict:
        url = f"{self.api_server_url}{path}"
        headers = {"Content-Type": "application/json"}
        token = get_bot_token()
        if token:
            headers["Authorization"] = f"Bearer {token}"
        body = json.dumps(data).encode("utf-8") if data else None
        req = urllib.request.Request(url, data=body, headers=headers, method=method)
        with urllib.request.urlopen(req, timeout=30) as resp:
            text = resp.read().decode("utf-8")
            return json.loads(text) if text else {}

    def lookup_intelligence(self, repo_url: str) -> dict | None:
        """Get intelligence for a repo in this project, or None if not found."""
        from urllib.parse import quote
        path = f"/api/ambient/v1/repo_intelligences/lookup?project_id={self.project_id}&repo_url={quote(repo_url, safe='')}"
        try:
            return self._make_request("GET", path)
        except urllib.error.HTTPError as e:
            if e.code == 404:
                return None
            raise

    def create_intelligence(self, data: dict) -> dict:
        data["project_id"] = self.project_id
        return self._make_request("POST", "/api/ambient/v1/repo_intelligences", data)

    def update_intelligence(self, intel_id: str, data: dict) -> dict:
        return self._make_request("PATCH", f"/api/ambient/v1/repo_intelligences/{intel_id}", data)

    def list_findings(self, intelligence_id: str, file_path: str | None = None) -> dict:
        from urllib.parse import quote
        path = f"/api/ambient/v1/repo_findings?intelligence_id={intelligence_id}"
        if file_path:
            path += f"&search=file_path%20like%20'%25{quote(file_path, safe='')}%25'"
        path += "&search=status%20%3D%20'active'"
        return self._make_request("GET", path)

    def create_finding(self, data: dict) -> dict:
        return self._make_request("POST", "/api/ambient/v1/repo_findings", data)

    def get_context(self, repo_urls: list[str], max_entries: int = 20) -> dict:
        from urllib.parse import quote
        urls_param = ",".join(quote(u, safe="") for u in repo_urls)
        path = f"/api/ambient/v1/repo_intelligences/context?project_id={self.project_id}&repo_urls={urls_param}&max_entries={max_entries}"
        return self._make_request("GET", path)

    def intelligence_exists(self, repo_url: str) -> bool:
        return self.lookup_intelligence(repo_url) is not None
```

### 3.2 MCP Tool Definitions

```python
# ambient_runner/bridges/claude/memory_tools.py

"""Memory MCP tools for storing and querying repo intelligence."""

import json
import logging
import os
from typing import Any, Callable, Optional

from ambient_runner.tools.intelligence_api import IntelligenceAPIClient

logger = logging.getLogger(__name__)


def create_memory_mcp_tools(
    sdk_tool_decorator: Callable,
    client: Optional[IntelligenceAPIClient] = None,
) -> list[Any]:
    """Create memory tools for the Claude Agent SDK."""
    api_client = client or _create_default_client()
    if api_client is None:
        logger.warning("Intelligence API client not available - memory tools will be skipped")
        return []

    tools = []
    session_id = os.getenv("AGENTIC_SESSION_NAME", "")

    def _ok(data: dict) -> dict:
        return {"content": [{"type": "text", "text": json.dumps(data, indent=2)}]}

    def _err(e: Exception) -> dict:
        return {"content": [{"type": "text", "text": json.dumps({"error": str(e)})}], "isError": True}

    # ── Tool 1: memory_query ──────────────────────────────────────────

    @sdk_tool_decorator(
        "memory_query",
        (
            "Search project memory for repo intelligence and findings. "
            "Use this when starting work on a repo or file you haven't seen before, "
            "or to check what previous sessions discovered. "
            "Returns repo-level architecture summaries and file-level findings."
        ),
        {
            "type": "object",
            "properties": {
                "repo_url": {
                    "type": "string",
                    "description": "Repository URL to query intelligence for",
                },
                "file_path": {
                    "type": "string",
                    "description": "Optional file path to filter findings (partial match)",
                },
                "category": {
                    "type": "string",
                    "description": "Filter findings by category",
                    "enum": ["investigation", "caveat", "review", "convention"],
                },
            },
            "required": ["repo_url"],
        },
    )
    async def memory_query(args: dict) -> dict:
        try:
            repo_url = args["repo_url"]
            intel = api_client.lookup_intelligence(repo_url)
            if not intel:
                return _ok({"found": False, "message": f"No intelligence stored for {repo_url}"})

            result = {"found": True, "intelligence": intel}

            # Fetch findings if requested
            file_path = args.get("file_path")
            if file_path or args.get("category"):
                findings = api_client.list_findings(intel["id"], file_path=file_path)
                result["findings"] = findings.get("items", [])

            return _ok(result)
        except Exception as e:
            return _err(e)

    tools.append(memory_query)

    # ── Tool 2: memory_store ──────────────────────────────────────────

    @sdk_tool_decorator(
        "memory_store",
        (
            "Store a finding about a file in project memory for future sessions. "
            "Use this when you discover something important: bug root causes, "
            "code caveats, conventions, or investigation findings. "
            "Future sessions touching this file will be warned."
        ),
        {
            "type": "object",
            "properties": {
                "repo_url": {
                    "type": "string",
                    "description": "Repository URL this finding belongs to",
                },
                "file_path": {
                    "type": "string",
                    "description": "File path within the repo (e.g. 'components/backend/handlers/sessions.go')",
                },
                "category": {
                    "type": "string",
                    "description": "Type of finding",
                    "enum": ["investigation", "caveat", "review", "convention"],
                },
                "title": {
                    "type": "string",
                    "description": "One-line summary of the finding",
                },
                "body": {
                    "type": "string",
                    "description": "Detailed description in markdown",
                },
                "severity": {
                    "type": "string",
                    "description": "Severity level",
                    "enum": ["info", "warning", "critical"],
                },
                "source_ref": {
                    "type": "string",
                    "description": "Reference (e.g. 'pr:456', 'issue:PROJ-789')",
                },
                "confidence": {
                    "type": "number",
                    "description": "Confidence score 0.0-1.0",
                    "minimum": 0.0,
                    "maximum": 1.0,
                },
            },
            "required": ["repo_url", "file_path", "category", "title", "body"],
        },
    )
    async def memory_store(args: dict) -> dict:
        try:
            repo_url = args["repo_url"]

            # Ensure intelligence exists for this repo
            intel = api_client.lookup_intelligence(repo_url)
            if not intel:
                return _ok({
                    "stored": False,
                    "message": f"No intelligence record for {repo_url}. Run repo analysis first.",
                })

            finding = api_client.create_finding({
                "intelligence_id": intel["id"],
                "file_path": args["file_path"],
                "category": args["category"],
                "title": args["title"],
                "body": args["body"],
                "severity": args.get("severity", "info"),
                "source_type": "agent_analysis",
                "source_ref": args.get("source_ref"),
                "confidence": args.get("confidence"),
                "session_id": session_id,
            })
            return _ok({"stored": True, "finding_id": finding.get("id")})
        except Exception as e:
            return _err(e)

    tools.append(memory_store)

    # ── Tool 3: memory_warn ───────────────────────────────────────────

    @sdk_tool_decorator(
        "memory_warn",
        (
            "Check if a file has any known findings from previous sessions. "
            "Use this before modifying a file, especially during bug fixes or reviews. "
            "Returns active warnings, investigation findings, and caveats."
        ),
        {
            "type": "object",
            "properties": {
                "file_path": {
                    "type": "string",
                    "description": "File path to check (e.g. 'components/backend/handlers/sessions.go')",
                },
                "repo_url": {
                    "type": "string",
                    "description": "Repository URL (optional — checks all repos in project if omitted)",
                },
            },
            "required": ["file_path"],
        },
    )
    async def memory_warn(args: dict) -> dict:
        try:
            file_path = args["file_path"]
            repo_url = args.get("repo_url")

            if repo_url:
                intel = api_client.lookup_intelligence(repo_url)
                if not intel:
                    return _ok({"warnings": [], "message": "No intelligence for this repo"})
                findings = api_client.list_findings(intel["id"], file_path=file_path)
            else:
                # Search across all repos in project via TSL
                findings = api_client._make_request(
                    "GET",
                    f"/api/ambient/v1/repo_findings?search=file_path%20like%20'%25{file_path}%25'%20and%20status%20%3D%20'active'",
                )

            items = findings.get("items", [])
            if not items:
                return _ok({"warnings": [], "message": f"No known findings for {file_path}"})

            return _ok({
                "warnings": items,
                "count": len(items),
                "message": f"⚠ {len(items)} known finding(s) for {file_path}",
            })
        except Exception as e:
            return _err(e)

    tools.append(memory_warn)

    return tools


def _create_default_client() -> IntelligenceAPIClient | None:
    try:
        return IntelligenceAPIClient()
    except ValueError:
        return None
```

### 3.3 MCP Server Registration

```python
# In ambient_runner/bridges/claude/mcp.py, within build_mcp_servers():

# --- ADD after backend_tools registration (line ~120) ---

from ambient_runner.bridges.claude.memory_tools import create_memory_mcp_tools

memory_tools = create_memory_mcp_tools(sdk_tool_decorator=sdk_tool)
if memory_tools:
    memory_server = create_sdk_mcp_server(
        name="memory", version="1.0.0", tools=memory_tools
    )
    mcp_servers["memory"] = memory_server
    logger.info(
        f"Added memory MCP tools ({len(memory_tools)}): "
        "memory_query, memory_store, memory_warn"
    )
```

No other registration needed — `build_allowed_tools()` automatically adds `mcp__memory__*` to the allowed tools list.

---

## Layer 4: Auto-Analysis Trigger (ambient-runner)

### 4.1 Trigger Point

In `ambient_runner/endpoints/repos.py`, after `was_newly_cloned` (line 79), add a check-and-trigger call alongside the existing notification:

```python
# In add_repo(), after line 79:
    if was_newly_cloned:
        # ... existing REPOS_JSON update and mark_dirty (lines 67-78) ...

        asyncio.create_task(_trigger_repo_added_notification(name, url, context))

        # NEW: Trigger auto-analysis if no intelligence exists
        asyncio.create_task(_trigger_auto_analysis_if_needed(name, url, context))
```

### 4.2 Analysis Trigger Function

```python
# In ambient_runner/endpoints/repos.py

async def _trigger_auto_analysis_if_needed(repo_name: str, repo_url: str, context):
    """Check if intelligence exists for this repo; if not, prompt the agent to analyze it."""
    await asyncio.sleep(3)  # Let the repo notification land first

    try:
        from ambient_runner.tools.intelligence_api import IntelligenceAPIClient

        client = IntelligenceAPIClient()
        if client.intelligence_exists(repo_url):
            logger.info(f"Intelligence already exists for {repo_name}, skipping auto-analysis")
            return

        logger.info(f"No intelligence found for {repo_name}, triggering auto-analysis")

        backend_url = os.getenv("BACKEND_API_URL", "").rstrip("/")
        project_name = os.getenv("AGENTIC_SESSION_NAMESPACE", "").strip()
        session_id = context.session_id if context else "unknown"

        if not backend_url or not project_name:
            return

        url = f"{backend_url}/projects/{project_name}/agentic-sessions/{session_id}/agui/run"
        payload = {
            "threadId": session_id,
            "runId": str(uuid.uuid4()),
            "messages": [
                {
                    "id": str(uuid.uuid4()),
                    "role": "user",
                    "content": (
                        f"The repository '{repo_name}' ({repo_url}) has no stored intelligence yet. "
                        f"Please analyze this repository and store your findings using the memory tools:\n\n"
                        f"1. Use `memory_store` to save file-level findings (bugs, caveats, conventions)\n"
                        f"2. Focus on: architecture overview, key modules, test patterns, known issues\n"
                        f"3. Be concise — future sessions will read this\n\n"
                        f"After analysis, create the repo intelligence record via the API."
                    ),
                    "metadata": {
                        "hidden": True,
                        "autoSent": True,
                        "source": "auto_analysis",
                    },
                }
            ],
        }

        bot_token = get_bot_token()
        headers = {"Content-Type": "application/json"}
        if bot_token:
            headers["Authorization"] = f"Bearer {bot_token}"

        async with aiohttp.ClientSession() as session:
            async with session.post(url, json=payload, headers=headers) as resp:
                if resp.status == 200:
                    logger.info(f"Auto-analysis triggered for {repo_name}")
                else:
                    logger.warning(f"Auto-analysis trigger failed: {resp.status}")
    except Exception as e:
        logger.debug(f"Auto-analysis trigger failed (non-critical): {e}")
```

### 4.3 Environment Variable

The runner needs `API_SERVER_URL` to reach the ambient-api-server. This is injected by the operator alongside `BACKEND_API_URL`:

```go
// In operator sessions.go, alongside BACKEND_API_URL injection:
{Name: "API_SERVER_URL", Value: fmt.Sprintf(
    "http://ambient-api-server.%s.svc.cluster.local:8000/", operatorNamespace,
)},
```

**Note**: This is the ONE operator change needed. It's a single env var addition in the existing env var block — no reconciliation logic changes, no CRD changes.

---

## Layer 5: Frontend (Context Tab)

### 5.1 API Client

```typescript
// src/services/api/intelligence.ts

import type { RepoIntelligence, RepoFinding } from "@/lib/types";

const API_BASE = process.env.NEXT_PUBLIC_API_SERVER_URL || "/api/ambient/v1";

export async function getRepoIntelligence(
  projectId: string,
  repoUrl: string,
  token: string
): Promise<RepoIntelligence | null> {
  const params = new URLSearchParams({ project_id: projectId, repo_url: repoUrl });
  const res = await fetch(`${API_BASE}/repo_intelligences/lookup?${params}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (res.status === 404) return null;
  if (!res.ok) throw new Error(`Failed to fetch intelligence: ${res.status}`);
  return res.json();
}

export async function getRepoFindings(
  intelligenceId: string,
  token: string,
  status: string = "active"
): Promise<RepoFinding[]> {
  const search = `intelligence_id = '${intelligenceId}' and status = '${status}'`;
  const params = new URLSearchParams({ search });
  const res = await fetch(`${API_BASE}/repo_findings?${params}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!res.ok) throw new Error(`Failed to fetch findings: ${res.status}`);
  const data = await res.json();
  return data.items || [];
}
```

### 5.2 React Query Hook

```typescript
// src/services/queries/use-intelligence.ts

import { useQuery } from "@tanstack/react-query";
import { getRepoIntelligence, getRepoFindings } from "@/services/api/intelligence";

export function useRepoIntelligence(projectId: string, repoUrl: string, enabled: boolean) {
  return useQuery({
    queryKey: ["repo-intelligence", projectId, repoUrl],
    queryFn: () => getRepoIntelligence(projectId, repoUrl, getToken()),
    enabled: enabled && !!projectId && !!repoUrl,
    staleTime: 60_000, // 1 minute
  });
}

export function useRepoFindings(intelligenceId: string | undefined, enabled: boolean) {
  return useQuery({
    queryKey: ["repo-findings", intelligenceId],
    queryFn: () => getRepoFindings(intelligenceId!, getToken()),
    enabled: enabled && !!intelligenceId,
    staleTime: 60_000,
  });
}
```

### 5.3 Intelligence Section Component

Rendered inside each expanded repo card in the Context tab:

```tsx
// src/app/.../explorer/intelligence-section.tsx

import { Brain, AlertTriangle, Info } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { useRepoIntelligence, useRepoFindings } from "@/services/queries/use-intelligence";

export function IntelligenceSection({ projectId, repoUrl }: { projectId: string; repoUrl: string }) {
  const { data: intel, isLoading } = useRepoIntelligence(projectId, repoUrl, true);
  const { data: findings } = useRepoFindings(intel?.id, !!intel);

  if (isLoading) return <div className="text-xs text-muted-foreground px-2 py-1">Loading intelligence...</div>;
  if (!intel) return null; // No intelligence yet — don't show anything

  const activeFindings = findings?.filter(f => f.status === "active") || [];
  const criticalCount = activeFindings.filter(f => f.severity === "critical").length;
  const warningCount = activeFindings.filter(f => f.severity === "warning").length;

  return (
    <div className="px-2 pb-2 pl-10 space-y-1 border-t border-dashed mt-1 pt-1">
      <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
        <Brain className="h-3 w-3" />
        <span className="font-medium">Intelligence</span>
        {criticalCount > 0 && (
          <Badge variant="destructive" className="text-xs px-1 py-0 h-4">
            {criticalCount} critical
          </Badge>
        )}
        {warningCount > 0 && (
          <Badge variant="outline" className="text-xs px-1 py-0 h-4 border-yellow-500 text-yellow-600">
            {warningCount} warning
          </Badge>
        )}
      </div>

      {/* Summary line */}
      <div className="text-xs text-muted-foreground truncate">
        {intel.language} · {intel.framework || "unknown framework"}
        {intel.analyzed_at && ` · analyzed ${new Date(intel.analyzed_at).toLocaleDateString()}`}
      </div>

      {/* Active findings */}
      {activeFindings.slice(0, 3).map((finding) => (
        <div key={finding.id} className="text-xs flex items-start gap-1.5 py-0.5">
          {finding.severity === "critical" ? (
            <AlertTriangle className="h-3 w-3 text-red-500 mt-0.5 flex-shrink-0" />
          ) : (
            <Info className="h-3 w-3 text-muted-foreground mt-0.5 flex-shrink-0" />
          )}
          <span className="truncate">{finding.title}</span>
        </div>
      ))}
      {activeFindings.length > 3 && (
        <div className="text-xs text-muted-foreground pl-4">
          +{activeFindings.length - 3} more findings
        </div>
      )}
    </div>
  );
}
```

### 5.4 Context Tab Integration

In `context-tab.tsx`, add the `IntelligenceSection` inside each expanded repo card, after the branches section:

```tsx
// In context-tab.tsx, after line 232 (end of branches expansion block):

{/* Intelligence section — always visible when repo is expanded */}
{isExpanded && (
  <IntelligenceSection projectId={projectId} repoUrl={repo.url} />
)}
```

Props change: `ContextTab` needs `projectId: string` added to `ContextTabProps`.

---

## Task Ordering

| # | Task | Layer | Depends On | Effort |
|---|------|-------|------------|--------|
| 1 | Run code generator for `RepoIntelligence` Kind, customize model | api-server | — | M |
| 2 | Run code generator for `RepoFinding` Kind, customize model | api-server | — | M |
| 3 | Run code generator for `RepoEvent` Kind (read-only) | api-server | — | S |
| 4 | Add DAO methods: `GetByProjectAndRepo`, `ListByIntelligenceID` | api-server | 1, 2 | S |
| 5 | Add custom handlers: `/lookup`, `/context`, `/{id}/findings` | api-server | 4 | M |
| 6 | Add audit event creation in service layer | api-server | 3, 4 | S |
| 7 | Create `IntelligenceAPIClient` | runner | 5 | S |
| 8 | Create `memory_query`, `memory_store`, `memory_warn` MCP tools | runner | 7 | M |
| 9 | Register memory MCP server in `build_mcp_servers()` | runner | 8 | S |
| 10 | Add auto-analysis trigger in `repos.py` | runner | 7 | S |
| 11 | Add `API_SERVER_URL` env var to operator pod spec | operator | 5 | S |
| 12 | Add frontend intelligence section to Context tab | frontend | 5 | M |

**Critical path**: 1 → 4 → 5 → 7 → 8 → 9

Tasks 1/2/3 can run in parallel. Tasks 7-10 can start once task 5 is done. Task 11 and 12 are independent.

---

## Testing Strategy

### api-server (Go integration tests)

```go
// plugins/repoIntelligences/integration_test.go

func TestRepoIntelligencePost(t *testing.T) {
    h, client := test.RegisterIntegration(t)
    ctx := h.NewAuthenticatedContext(h.NewRandAccount())

    // Create
    input := openapi.RepoIntelligence{
        ProjectId: "test-project",
        RepoUrl:   "https://github.com/org/repo",
        Summary:   "Go backend with Gin framework",
        Language:  "go",
    }
    resp, httpResp, err := client.DefaultAPI.ApiAmbientApiServerV1RepoIntelligencesPost(ctx).
        RepoIntelligence(input).Execute()
    require.NoError(t, err)
    require.Equal(t, http.StatusCreated, httpResp.StatusCode)
    require.NotEmpty(t, resp.Id)

    // Lookup
    lookupResp, _, err := client.DefaultAPI.ApiAmbientApiServerV1RepoIntelligencesLookupGet(ctx).
        ProjectId("test-project").RepoUrl("https://github.com/org/repo").Execute()
    require.NoError(t, err)
    require.Equal(t, *resp.Id, *lookupResp.Id)

    // Upsert (second create with same project+repo should conflict or update)
    // ...
}

func TestRepoIntelligenceProjectScoping(t *testing.T) {
    // Verify entries from project A are not visible to project B queries
}

func TestRepoFindingLifecycle(t *testing.T) {
    // Create finding → verify active → patch to resolved → verify status
}

func TestContextEndpoint(t *testing.T) {
    // Create intelligence + findings → call /context → verify aggregated response
}

func TestAuditTrail(t *testing.T) {
    // Create intelligence → verify repo_event created with action=created
    // Update intelligence → verify repo_event created with action=updated
}
```

### Runner (Python unit tests)

```python
# tests/test_memory_tools.py

def test_memory_query_no_intelligence():
    """memory_query returns found=False when no intelligence exists."""
    mock_client = MockIntelligenceClient(intelligence=None)
    tools = create_memory_mcp_tools(sdk_tool_decorator=mock_decorator, client=mock_client)
    result = asyncio.run(tools[0]({"repo_url": "https://github.com/org/repo"}))
    assert json.loads(result["content"][0]["text"])["found"] is False

def test_memory_store_creates_finding():
    """memory_store creates a finding when intelligence exists."""
    mock_client = MockIntelligenceClient(intelligence={"id": "intel-1"})
    tools = create_memory_mcp_tools(sdk_tool_decorator=mock_decorator, client=mock_client)
    result = asyncio.run(tools[1]({
        "repo_url": "https://github.com/org/repo",
        "file_path": "main.go",
        "category": "caveat",
        "title": "Not thread-safe",
        "body": "Concurrent access without mutex",
    }))
    assert json.loads(result["content"][0]["text"])["stored"] is True

def test_memory_warn_returns_findings():
    """memory_warn returns active findings for a file path."""
    # ...

def test_auto_analysis_skipped_when_intelligence_exists():
    """_trigger_auto_analysis_if_needed does nothing when intelligence exists."""
    # ...
```

### Frontend (vitest)

```typescript
// __tests__/intelligence-section.test.tsx

describe("IntelligenceSection", () => {
  it("renders nothing when no intelligence exists", async () => {
    // Mock useRepoIntelligence to return null
    // Verify component renders null
  });

  it("shows findings count badges", async () => {
    // Mock intelligence + findings with mixed severities
    // Verify critical/warning badges appear
  });

  it("truncates to 3 findings with overflow count", async () => {
    // Mock 5 findings, verify only 3 shown + "+2 more"
  });
});
```

---

## Verification Checklist

- [ ] `cd components/ambient-api-server && go test ./plugins/repoIntelligences/... ./plugins/repoFindings/... ./plugins/repoEvents/...`
- [ ] `cd components/runners/ambient-runner && python -m pytest tests/test_memory_tools.py`
- [ ] `cd components/frontend && npx vitest run --coverage`
- [ ] Manual: create session → add repo → verify intelligence auto-analysis is triggered
- [ ] Manual: restart session with same repo → verify intelligence is loaded via MCP
- [ ] Manual: use `memory_warn` MCP tool → verify findings returned
- [ ] Manual: check Context tab → verify intelligence section appears in repo card
- [ ] `make lint` passes (all components)
- [ ] No existing tests broken
