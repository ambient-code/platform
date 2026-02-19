# Project Init

Validate that this TRex-spawned project is in a healthy, working state. This command is idempotent — it succeeds if everything is already running and correct, and only fails if something is genuinely broken.

Use the TodoWrite tool to track each phase as you work through it.

## Phase 1: Naming Consistency Audit

Scan all `.go`, `.yaml`, and `.md` files (skip `go.sum`) for template leftovers from the TRex scaffold. Flag any of the following:

| Pattern | Why It's Wrong |
|---------|---------------|
| `"My Service"` or `"My service"` | Placeholder title never renamed |
| `"Dinosaur"` or `"dinosaur"` (outside go.mod/go.sum imports) | TRex example entity left behind |
| `"rh-trex"` in description strings or comments (import paths are fine) | Template origin leaking into runtime |
| `openshift-online-team@redhat.com` | Template default contact |
| `api.openshift.com` or `api.stage.openshift.com` in server URLs | Template default servers |
| `"template"` in code comments suggesting unfinished customization | Scaffold language not cleaned up |
| `"placeholder"` in comments | Scaffold language not cleaned up |
| Stub TODO comments (e.g., `// TODO: Add ... when needed`) in otherwise empty files | Empty scaffold files |

**Action**: Report every finding with file, line number, and the problematic text. Do NOT auto-fix — present the findings to the user and ask how they want each category handled (rename, remove, or keep).

## Phase 2: Dependency Resolution

```bash
go mod tidy
```

Verify no errors. If `go.sum` changes, that's fine — it means dependencies were stale.

## Phase 3: Build the Binary

```bash
make binary
```

This must succeed with zero errors. If it fails:
- Report the exact error output
- Diagnose the root cause (missing dependency, type error, import cycle, etc.)
- Ask the user before attempting any fix

## Phase 4: Database Setup (Idempotent)

1. Check if the PostgreSQL container is already running:
   ```bash
   docker ps --filter name=ambient-api-server-postgres --format "{{.Names}} {{.Status}}"
   ```

2. **If running and healthy**: Skip to migration. Report: "Database already running."

3. **If exists but stopped**:
   ```bash
   docker start ambient-api-server-postgres
   ```

4. **If does not exist**:
   ```bash
   make db/setup
   ```

5. Wait for readiness (up to 15 seconds):
   ```bash
   for i in $(seq 1 15); do
     docker exec ambient-api-server-postgres pg_isready -U postgres && break
     sleep 1
   done
   ```

6. Run migrations:
   ```bash
   ./ambient-api-server migrate
   ```

## Phase 5: Run Unit Tests

```bash
make test
```

All tests must pass. Report the test summary (pass count, fail count, elapsed time).

## Phase 6: Run Integration Tests

```bash
make test-integration
```

If the Makefile target says "not implemented", report that and skip — it's expected for a fresh project with no Kinds yet.

If integration tests exist and run, all must pass. Report the test summary.

## Phase 7: Verify Code Quality

Run static analysis:

```bash
go vet ./cmd/... ./pkg/...
```

Check formatting:

```bash
gofmt -l .
```

If `gofmt -l` produces output, files need formatting — report which ones.

## Phase 8: Summary Report

Present a structured report:

```
=== TRex Project Init Report ===

Project:        ambient-api-server
Module:         github.com/ambient/platform/components/ambient-api-server
Go Version:     (from go.mod)

Naming Audit:   X issues found (list categories)
Dependencies:   OK / FIXED / FAILED
Build:          OK / FAILED
Database:       RUNNING (was: already running | started | created)
Migrations:     OK / FAILED
Unit Tests:     X passed, Y failed
Integration:    X passed, Y failed / NOT CONFIGURED
Code Quality:   OK / X issues

Overall:        HEALTHY / NEEDS ATTENTION
```

If `Overall` is `NEEDS ATTENTION`, list each issue that requires user action.

## When to Use This

- After cloning/spawning a new TRex project
- After pulling major changes
- When onboarding to an existing project
- As a sanity check before starting new work
- When something feels "off" and you want a full diagnostic
