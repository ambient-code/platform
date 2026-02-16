# Regenerate OpenAPI Client Code

Regenerate the Go client code from OpenAPI specifications. Run this after modifying any `openapi/*.yaml` files.

## Instructions

### Step 1: Verify Docker/Podman is Running

The generator uses a container to run the OpenAPI code generator:
```bash
podman info --format '{{.Host.RemoteSocket.Exists}}' 2>/dev/null || docker info --format '{{.ID}}' 2>/dev/null
```

If neither is running, inform the user that Docker or Podman is required.

### Step 2: Run the Generator

```bash
make generate
```

This takes 2-3 minutes. It:
1. Builds a container image from `Dockerfile.openapi`
2. Runs the OpenAPI Generator inside the container
3. Copies generated Go files back into `pkg/api/openapi/`

### Step 3: Verify Generated Files

```bash
ls -la pkg/api/openapi/model_*.go | head -20
```

Check for the expected model files for each Kind defined in the OpenAPI specs.

### Step 4: Build to Confirm

```bash
make binary
```

## When to Run This

- After creating a new Kind (part of `/trex.kind.new` workflow)
- After adding fields to an existing Kind's OpenAPI spec
- After modifying `openapi/openapi.yaml` or any `openapi/openapi.*.yaml` file
- After resolving merge conflicts in OpenAPI specs

## Troubleshooting

- **"Cannot connect to Docker daemon"**: Start Docker/Podman first
- **Build fails after generation**: Check for type mismatches between your OpenAPI schema types and the presenter code
- **Missing model files**: Verify the schema name in `openapi.yaml` matches the `$ref` target exactly
