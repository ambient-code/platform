# Remove a Kind

Completely remove a generated Kind and all its artifacts from the project.

## Instructions

Ask the user for the **Kind name** (PascalCase, e.g., "Widget").

Derive the naming variants (see `/trex.kind.new` for the full table):
- `Kind` = PascalCase (e.g., `Widget`)
- `kindLowerPlural` = camelCase plural (e.g., `widgets`)
- `kind_snake_case` = snake_case (e.g., `widget`)

Use the TodoWrite tool to track progress.

### Step 1: Verify the Kind Exists

Check that the Kind's files exist before removing:
```bash
ls plugins/{kindLowerPlural}/plugin.go
ls pkg/api/{kind}.go
ls pkg/services/{kind}.go
```

If files don't exist, inform the user and stop.

### Step 2: Remove Generated Files (11 files)

Delete all files created by the generator:

```bash
rm -rf plugins/{kindLowerPlural}/
rm -f pkg/api/{kind}.go
rm -f pkg/api/presenters/{kind}.go
rm -f pkg/handlers/{kind}.go
rm -f pkg/services/{kind}.go
rm -f pkg/dao/{kind}.go
rm -f pkg/dao/mocks/{kind}.go
rm -f pkg/db/migrations/*_add_{kindLowerPlural}.go
rm -f openapi/openapi.{kindLowerPlural}.yaml
rm -f test/integration/{kindLowerPlural}_test.go
rm -f test/factories/{kindLowerPlural}.go
```

### Step 3: Remove Generated OpenAPI Client Files

```bash
rm -f pkg/api/openapi/model_{kind_snake_case}*.go
rm -f pkg/api/openapi/docs/{Kind}*.md
```

### Step 4: Unwire from Existing Files

#### 4.1 `cmd/ambient-api-server/main.go`

Remove the blank import line for the Kind's plugin.

#### 4.2 `pkg/db/migrations/migration_structs.go`

Remove from `MigrationList`:
```go
add{Kind}s(),
```

#### 4.3 `openapi/openapi.yaml`

Remove the path `$ref` entries for this Kind (both collection and item paths).
Remove the schema `$ref` entries (`{Kind}`, `{Kind}List`, `{Kind}PatchRequest`).

### Step 5: Regenerate OpenAPI Client

```bash
make generate
```

This ensures the OpenAPI client code no longer references the removed Kind.

### Step 6: Verify Clean Removal

```bash
make binary
```

Fix any remaining references. Common issues:
- Other services importing the removed Kind's plugin
- Test files referencing the removed Kind's factory
- Stale references in OpenAPI generated code

### Step 7: Confirm

List any files that still reference the Kind name:
```bash
rg "{Kind}" --type go --type yaml -l
```

If any remain, they need manual cleanup (likely cross-references from other Kinds).
