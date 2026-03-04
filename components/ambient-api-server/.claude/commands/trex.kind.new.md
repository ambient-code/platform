# Generate a New Kind

Generate a complete CRUD entity (Kind) using the automated generator script.

## Instructions

Use the TodoWrite tool to track progress.

### Step 1: Gather Requirements

Ask the user for:
1. **Kind name** (singular, PascalCase): e.g., "Widget", "Rocket", "Customer"
2. **Fields** beyond the base Meta fields (optional):
   - Name (use snake_case): e.g., "fuel_type", "max_speed"
   - Type: `string`, `int`, `int64`, `bool`, `float64`, `time`
   - Nullability: `:required` or `:optional` (default: nullable)

### Step 2: Run the Generator

```bash
go run ./scripts/generator.go --kind {KindName}
```

With custom fields:
```bash
go run ./scripts/generator.go --kind {KindName} --fields "{field1}:{type1}:{nullability},{field2}:{type2}"
```

Example:
```bash
go run ./scripts/generator.go --kind Rocket --fields "name:string:required,fuel_type:string,max_speed:int"
```

The generator automatically:
- Creates 11 new files (plugin, model, DAO, mock, service, presenters, handlers, migration, OpenAPI spec, test factory, integration tests)
- Modifies 3 existing files (`cmd/ambient-api-server/main.go`, `pkg/db/migrations/migration_structs.go`, `openapi/openapi.yaml`)
- Runs `make generate` to regenerate the OpenAPI client
- Formats all Go files with `gofmt`

### Step 3: Build and Verify

```bash
make binary
```

Fix any compilation errors if they occur.

### Step 4: Test

```bash
make test-integration
```

Or run just the new Kind's tests:
```bash
TESTFLAGS="-run Test{Kind}" make test-integration
```

## Field Type Reference

| User Type | Go (required) | Go (nullable) | OpenAPI type | OpenAPI format | DB Type |
|-----------|--------------|---------------|-------------|---------------|---------|
| string | `string` | `*string` | string | | varchar |
| int | `int` | `*int` | integer | int32 | integer |
| int64 | `int64` | `*int64` | integer | int64 | bigint |
| bool | `bool` | `*bool` | boolean | | boolean |
| float64 | `float64` | `*float64` | number | double | float8 |
| time | `time.Time` | `*time.Time` | string | date-time | timestamptz |

## Naming Convention Reference

| Context | Convention | Example |
|---------|-----------|---------|
| Go struct | PascalCase | `ProductCategory` |
| Go variable | camelCase | `productCategory` |
| Go package | lowercase plural | `productcategorys` |
| API path | snake_case plural | `/product_categorys` |
| JSON field | snake_case | `category_name` |
| DB table | snake_case plural | `product_categorys` |
| DB column | snake_case | `category_name` |
| Plugin registry | PascalCase plural | `ProductCategorys` |

Rules:
- Plural: always append "s" (not "es" or "ies")
- snake_case: insert `_` before each uppercase letter, then lowercase everything
