# Start Server (Development Mode)

Build and start the API server with authentication and authorization **disabled** for local development.

## Instructions

1. Build the binary:
   ```bash
   make binary
   ```

2. If the build fails, diagnose and fix compilation errors before proceeding.

3. Start the server with auth disabled:
   ```bash
   make run-no-auth
   ```
   This runs `./ambient-api-server migrate` then `./ambient-api-server serve --enable-authz=false --enable-jwt=false`.

4. The server starts on `localhost:8000`. Verify it's running:
   ```bash
   curl -s http://localhost:8000/api/ambient-api-server | jq
   ```

## Prerequisites

- PostgreSQL must be running. If the server fails to connect, suggest running `/trex.db.setup` first.
- The binary must compile cleanly. If there are errors, fix them before retrying.

## What's Running

The `serve` command launches four concurrent servers:
- **API server** on `localhost:8000` (REST endpoints)
- **Metrics server** on `localhost:8080` (Prometheus metrics)
- **Health check server** on `localhost:8083`
- **Controllers server** (event-driven background processing via PostgreSQL LISTEN/NOTIFY)

## Quick Test

After the server starts, test with:
```bash
curl -s http://localhost:8000/api/ambient-api-server/v1/ | jq
```
