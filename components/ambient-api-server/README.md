
# Ambient API Server

REST API microservice built with the TRex.AI library, providing CRUD functionality via auto-generated Kind plugins.

## Quick Start

1. **Install dependencies:**
   ```bash
   go mod tidy
   ```

2. **Build the service:**
   ```bash
   make binary
   ```

3. **Run tests:**
   ```bash
   make test
   ```

## Generating New Kinds

This project includes the TRex.AI Kind generator to create complete CRUD functionality for new resource types.

### Important Note

To create your first Kind:

1. Ensure your project compiles and tests pass: `make test`
2. Generate your Kind and implement the needed TRex.AI interfaces

### Basic Example: Generate a HelloWorld Kind

```bash
# Generate a HelloWorld Kind with a Message attribute
go run ./scripts/generator.go --kind HelloWorld --fields "message:string:required" --project "ambient-api-server" --repo "github.com/ambient/platform/components"

```

### Field Types and Options

The generator supports these field types:
- `string` - Text data
- `int` - 32-bit integer  
- `int64` - 64-bit integer
- `bool` - Boolean true/false
- `float` or `float64` - Floating point numbers
- `time` - Timestamp (time.Time)

Field nullability options:
- `:required` - Non-nullable field (base types like `string`, `int`)
- `:optional` - Nullable field (pointer types like `*string`, `*int`) - default

### What the Generator Creates

For each Kind, the generator automatically creates:

- **API model** (`plugins/{kinds}/model.go`) - Go structs for the Kind
- **HTTP handlers** (`plugins/{kinds}/handler.go`) - REST API endpoints  
- **Service layer** (`plugins/{kinds}/service.go`) - Business logic with event handlers
- **Data access** (`plugins/{kinds}/dao.go`) - Database operations
- **Database migration** (`plugins/{kinds}/migration.go`) - Schema changes
- **OpenAPI spec** (`openapi/openapi.{kinds}.yaml`) - API documentation
- **Tests** (`plugins/{kinds}/*_test.go`) - Unit and integration tests
- **Plugin registration** (`plugins/{kinds}/plugin.go`) - Auto-wires everything together

## Database Operations

### Configuration

The service reads database credentials from the `secrets/` directory. Each file contains a single value:

| File | Description | Default |
|------|-------------|---------|
| `secrets/db.host` | Database hostname | `localhost` |
| `secrets/db.port` | Database port | `5432` |
| `secrets/db.name` | Database name | `ambient_api_server` |
| `secrets/db.user` | Database user | `postgres` |
| `secrets/db.password` | Database password | `postgres` |

These values must match the PostgreSQL instance created by `make db/setup`.

### Start PostgreSQL
```bash
make db/setup
```

Supports both Docker and Podman via the `CONTAINER_ENGINE` variable (auto-detects podman, falls back to docker):
```bash
make db/setup CONTAINER_ENGINE=docker
```

### Run Migrations
```bash
./ambient-api-server migrate
```

### Stop Database
```bash
make db/teardown
```

## Running the Service

### With Authentication
```bash
make run
```

### Without Authentication (Development)
```bash
make run-no-auth
```

The service will be available at `http://localhost:8000`.

## API Endpoints

After generating Kinds, API endpoints follow this pattern:
- `GET /api/ambient-api-server/v1/{kinds}` - List all items
- `POST /api/ambient-api-server/v1/{kinds}` - Create new item
- `GET /api/ambient-api-server/v1/{kinds}/{id}` - Get specific item
- `PATCH /api/ambient-api-server/v1/{kinds}/{id}` - Update specific item

## Development Workflow

1. **Generate new Kind**: `go run ./scripts/generator.go --kind MyKind --fields "name:string:required"`
2. **Run migrations**: `./ambient-api-server migrate`
3. **Test the API**: `make test && make test-integration`
4. **Start service**: `make run-no-auth`
5. **Test endpoints**: `curl http://localhost:8000/api/ambient-api-server/v1/my_kinds`

## Project Structure

```
├── cmd/ambient-api-server/          # Main application entry point
├── pkg/                     # Core packages
│   ├── api/                 # API types (re-exports from TRex.AI library)
│   ├── auth/                # Authentication stubs
│   ├── db/                  # Database utilities stubs
│   ├── errors/              # Error handling stubs
│   └── ...                  # Other core package stubs
├── plugins/                 # Generated Kinds (business logic)
│   └── {kinds}/             # Each Kind gets its own plugin
├── openapi/                 # OpenAPI specifications
├── scripts/                 # Code generator
├── templates/               # Generator templates
├── secrets/                 # Database credentials (see Database Operations)
└── test/                    # Test utilities
```

## Generated Plugin Architecture

Each generated Kind is a self-contained plugin with:

- **Event-driven controllers** - Process CREATE/UPDATE/DELETE events automatically
- **Idempotent handlers** - Safe to run multiple times
- **Complete CRUD** - Create, Read, Update, Delete, Search operations
- **OpenAPI integration** - Automatic API documentation generation
- **Test coverage** - Unit and integration tests included

## Next Steps

1. Generate your first Kind to see the full functionality
2. Customize the generated code for your specific business logic
3. Add custom validation in the handlers
4. Extend the service layer with additional business rules
5. Add integration tests for your specific use cases

For more information, see the [TRex.AI documentation](../../CLAUDE.md).