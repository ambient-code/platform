# Code Generation

The `scripts/generator.go` script generates complete CRUD functionality for new resource types (Kinds).

## Usage

```bash
go run ./scripts/generator.go \
  --kind HelloWorld \
  --fields "message:string:required,priority:int,active:bool" \
  --project ambient-api-server \
  --repo github.com/ambient/platform/components \
  --library github.com/openshift-online/rh-trex-ai
```

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--kind` | `Asteroid` | PascalCase Kind name (e.g., `HelloWorld`) |
| `--fields` | `""` | Comma-separated `name:type[:required\|optional]` |
| `--project` | `ambient-api-server` | Project directory name |
| `--repo` | `github.com/ambient/platform/components` | Module path prefix |
| `--library` | `github.com/openshift-online/rh-trex-ai` | rh-trex-ai module path |
| `--plural` | auto | Override auto-pluralization |
| `--skip-generate` | false | Skip `make generate` after code gen |

## Supported Field Types

| Type | Go Type (required) | Go Type (optional) | DB Type | OpenAPI Type |
|------|-------------------|--------------------|---------|-------------|
| `string` | string | *string | text | string |
| `int` | int | *int | integer | integer (int32) |
| `int64` | int64 | *int64 | bigint | integer (int64) |
| `bool` | bool | *bool | boolean | boolean |
| `float` | float64 | *float64 | double precision | number (double) |
| `time` | time.Time | *time.Time | timestamp | string (date-time) |

## Generated Files

For a Kind named `HelloWorld` (pluralized to `helloWorlds`):

| Template | Output | Content |
|----------|--------|---------|
| `generate-api.txt` | `plugins/helloWorlds/model.go` | GORM model + patch request |
| `generate-presenters.txt` | `plugins/helloWorlds/presenter.go` | OpenAPI ↔ model conversion |
| `generate-dao.txt` | `plugins/helloWorlds/dao.go` | DAO interface + GORM impl |
| `generate-handlers.txt` | `plugins/helloWorlds/handler.go` | HTTP handlers |
| `generate-services.txt` | `plugins/helloWorlds/service.go` | Service + event handlers |
| `generate-mock.txt` | `plugins/helloWorlds/mock_dao.go` | Mock DAO |
| `generate-migration.txt` | `plugins/helloWorlds/migration.go` | DB migration |
| `generate-plugin.txt` | `plugins/helloWorlds/plugin.go` | init() registration |
| `generate-openapi-kind.txt` | `openapi/openapi.helloWorlds.yaml` | OpenAPI spec for the Kind |
| `generate-test.txt` | `plugins/helloWorlds/integration_test.go` | Integration tests |
| `generate-test-factories.txt` | `plugins/helloWorlds/factory_test.go` | Test data factories |
| `generate-testmain.txt` | `plugins/helloWorlds/testmain_test.go` | TestMain setup |

## Template Variables

| Variable | Example Value | Description |
|----------|-------------|-------------|
| `{{.Kind}}` | `HelloWorld` | PascalCase singular |
| `{{.KindPlural}}` | `HelloWorlds` | PascalCase plural |
| `{{.KindLowerPlural}}` | `helloWorlds` | camelCase plural (directory name) |
| `{{.KindLowerSingular}}` | `helloWorld` | camelCase singular |
| `{{.KindSnakeCasePlural}}` | `hello_worlds` | snake_case plural (URL path, table name) |
| `{{.Project}}` | `ambient-api-server` | Project name |
| `{{.ProjectPascalCase}}` | `AmbientApiServer` | Project in PascalCase |
| `{{.Repo}}` | `github.com/ambient/platform/components` | Repository path |
| `{{.Library}}` | `github.com/openshift-online/rh-trex-ai` | Framework library path |
| `{{.Cmd}}` | `ambient-api-server` | Command directory name |
| `{{.ID}}` | `202602141530` | Migration ID (timestamp) |
| `{{.Fields}}` | `[]Field{...}` | Parsed field definitions |

## Import Classification

| Import Target | Template Variable |
|--------------|-------------------|
| Framework packages (`pkg/api`, `pkg/server`, `pkg/db`, etc.) | `{{.Library}}` |
| Project-local packages (`pkg/api/openapi`, `plugins/...`, `cmd/...`, `test`) | `{{.Repo}}/{{.Project}}` |

## Auto-Wiring

The generator automatically:
1. Creates the plugin directory and all files
2. Injects paths and schemas into `openapi/openapi.yaml` (at `# AUTO-ADD NEW PATHS` and `# AUTO-ADD NEW SCHEMAS` markers)
3. Adds the plugin side-effect import to `cmd/ambient-api-server/main.go`
4. Runs `make generate` to regenerate the OpenAPI Go client (unless `--skip-generate`)
5. Runs `gofmt` on all generated `.go` files

## Pluralization

The generator handles common irregular plurals (registry→registries, policy→policies, etc.). Use `--plural` to override for edge cases.

## Post-Generation Steps

1. Review and customize `service.go` — add business logic to `OnUpsert`/`OnDelete`
2. Add validation in `handler.go` Create/Patch methods
3. Run `make test` to verify everything compiles and tests pass
4. Optionally add cross-kind integration tests in `test/integration/`
