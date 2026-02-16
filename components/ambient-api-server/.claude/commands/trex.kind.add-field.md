# Add a Field to an Existing Kind

Add a new field to an existing Kind, updating all layers consistently.

## Instructions

Ask the user for:
1. **Kind name** (PascalCase): e.g., "Dinosaur"
2. **Field name** (snake_case): e.g., "habitat_type"
3. **Field type**: `string`, `int`, `int64`, `bool`, `float64`, `time.Time`
4. **Required or nullable**: required fields use base types (`string`), nullable use pointers (`*string`)

Derive the Go field name in PascalCase (e.g., `habitat_type` -> `HabitatType`).

Use the TodoWrite tool to track progress.

### Step 1: Update API Model (`pkg/api/{kind}.go`)

Add the field to the `{Kind}` struct:
```go
HabitatType string `json:"habitat_type"`
```

Add the field to `{Kind}PatchRequest` as a pointer:
```go
HabitatType *string `json:"habitat_type,omitempty"`
```

### Step 2: Create a New Migration File

**IMPORTANT**: Create a brand new migration file at `pkg/db/migrations/{YYYYMMDDHHMM}_add_{field}_to_{kindLowerPlural}.go`. Do NOT modify the Kind's existing migration.

```go
package migrations

import (
    "gorm.io/gorm"
    "github.com/go-gormigrate/gormigrate/v2"
)

func add{FieldPascalCase}To{KindPlural}() *gormigrate.Migration {
    type {Kind} struct {
        Model
        {FieldPascalCase} {GoType}
    }

    return &gormigrate.Migration{
        ID: "{YYYYMMDDHHMM}",
        Migrate: func(tx *gorm.DB) error {
            return tx.AutoMigrate(&{Kind}{})
        },
        Rollback: func(tx *gorm.DB) error {
            return tx.Migrator().DropColumn(&{Kind}{}, "{field_snake_case}")
        },
    }
}
```

Register it in `pkg/db/migrations/migration_structs.go` by appending to `MigrationList`.

### Step 3: Update Presenter (`pkg/api/presenters/{kind}.go`)

Add the field to both `Convert{Kind}()` and `Present{Kind}()` functions, handling the type conversion between internal and OpenAPI models.

### Step 4: Update Handler (`pkg/handlers/{kind}.go`)

In the `Patch` method, add the nil-check for the new field:
```go
if patch.{FieldPascalCase} != nil {
    found.{FieldPascalCase} = patch.{FieldPascalCase}
}
```

### Step 5: Update OpenAPI Spec (`openapi/openapi.{kindLowerPlural}.yaml`)

Add the field to three schema sections:
- `{Kind}` schema properties
- `{Kind}PatchRequest` schema properties
- If required: add to the `required` array

### Step 6: Regenerate OpenAPI Client

```bash
make generate
```

Wait for completion, then verify the new field appears in:
- `pkg/api/openapi/model_{kind_snake_case}.go`
- `pkg/api/openapi/model_{kind_snake_case}_patch_request.go`

### Step 7: Build and Test

```bash
make binary
make test-integration
```

### Common Pitfalls

- Forgetting to add the migration to `MigrationList`
- Type mismatch between Go model (`int`) and OpenAPI model (`int32`) in the presenter
- Missing the field in the PatchRequest struct
- Forgetting to handle nullable fields with pointer dereferencing in presenters

**Note**: Integration tests use testcontainers, which provisions a fresh PostgreSQL instance automatically. No manual `db/teardown` or `db/setup` is needed for tests. However, if running the server locally, use `/trex.db.setup` to recreate the database with the new migration.
