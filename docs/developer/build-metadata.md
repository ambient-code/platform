# Build Metadata System

This document explains the build metadata system that embeds git and build information into container images and logs it at runtime.

## Overview

Every container image built from this repository includes metadata about:
- **Git Commit**: Full commit hash and version
- **Git Branch**: Branch name the image was built from
- **Git Repository**: Remote repository URL
- **Git Status**: Whether there were uncommitted changes (`-dirty` suffix)
- **Build Date**: ISO 8601 timestamp of when the image was built
- **Build User**: Username and hostname of the builder

This information is logged to the console when each component starts up, making it easy to:
- Verify which version is running in production
- Track down which commit introduced a bug
- Identify if an image was built from a clean state or had local modifications
- Audit who built production images and when

## How It Works

### 1. Build Time: Makefile Captures Git Metadata

When you run `make build-all` or any build target, the Makefile captures git information:

```makefile
GIT_COMMIT := $(shell git rev-parse HEAD 2>/dev/null || echo "unknown")
GIT_BRANCH := $(shell git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")
GIT_REPO := $(shell git remote get-url origin 2>/dev/null || echo "local")
GIT_DIRTY := $(shell git diff --quiet 2>/dev/null || echo "-dirty")
GIT_VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
BUILD_USER := $(shell whoami)@$(shell hostname)
```

These values are passed to the container engine as build arguments:

```bash
podman build \
  --build-arg GIT_COMMIT=abc123... \
  --build-arg GIT_BRANCH=main \
  --build-arg GIT_REPO=https://github.com/... \
  --build-arg GIT_VERSION=v1.2.3-dirty \
  --build-arg BUILD_DATE=2025-12-15T10:30:00Z \
  --build-arg BUILD_USER=gkrumbac@MacBook \
  -t vteam-backend:latest .
```

### 2. Build Time: Dockerfiles Embed Metadata as Environment Variables

Each Dockerfile declares build arguments and sets them as environment variables:

```dockerfile
# Build arguments
ARG GIT_COMMIT=unknown
ARG GIT_BRANCH=unknown
ARG GIT_REPO=unknown
ARG GIT_VERSION=unknown
ARG BUILD_DATE=unknown
ARG BUILD_USER=unknown

# ... build stages ...

# Final stage - set as environment variables
ENV GIT_COMMIT=${GIT_COMMIT}
ENV GIT_BRANCH=${GIT_BRANCH}
ENV GIT_REPO=${GIT_REPO}
ENV GIT_VERSION=${GIT_VERSION}
ENV BUILD_DATE=${BUILD_DATE}
ENV BUILD_USER=${BUILD_USER}
```

**Note**: For multi-stage builds, you must redeclare ARG in each stage where you need to use them.

### 3. Runtime: Components Log Metadata on Startup

Each component reads these environment variables and logs them when starting:

**Backend (Go):**
```go
func logBuildInfo() {
    log.Println("==============================================")
    log.Println("Backend API - Build Information")
    log.Println("==============================================")
    log.Printf("Version:     %s", getEnvOrDefault("GIT_VERSION", "unknown"))
    log.Printf("Commit:      %s", getEnvOrDefault("GIT_COMMIT", "unknown"))
    log.Printf("Branch:      %s", getEnvOrDefault("GIT_BRANCH", "unknown"))
    log.Printf("Repository:  %s", getEnvOrDefault("GIT_REPO", "unknown"))
    log.Printf("Built:       %s", getEnvOrDefault("BUILD_DATE", "unknown"))
    log.Printf("Built by:    %s", getEnvOrDefault("BUILD_USER", "unknown"))
    log.Println("==============================================")
}
```

**Frontend (TypeScript):**
```typescript
// src/instrumentation.ts - runs once on server startup
export function register() {
  if (process.env.NEXT_RUNTIME === 'nodejs') {
    console.log('==============================================');
    console.log('Frontend - Build Information');
    console.log('==============================================');
    console.log(`Version:     ${process.env.NEXT_PUBLIC_GIT_VERSION || 'unknown'}`);
    console.log(`Commit:      ${process.env.NEXT_PUBLIC_GIT_COMMIT || 'unknown'}`);
    // ...
  }
}
```

**Runner (Python):**
```python
def log_build_info():
    """Log build metadata information."""
    logging.info("=" * 46)
    logging.info("Claude Code Runner - Build Information")
    logging.info("=" * 46)
    logging.info(f"Version:     {os.getenv('GIT_VERSION', 'unknown')}")
    logging.info(f"Commit:      {os.getenv('GIT_COMMIT', 'unknown')}")
    # ...
```

## Example Output

When you start any component, you'll see output like:

```
==============================================
Backend API - Build Information
==============================================
Version:     v1.2.3-dirty
Commit:      abc123def456789...
Branch:      feature/build-metadata
Repository:  https://github.com/ambient-code/platform.git
Built:       2025-12-15T10:30:45Z
Built by:    gkrumbac@MacBook-Pro.local
==============================================
```

The `-dirty` suffix in the version indicates there were uncommitted changes when the image was built.

## Viewing Build Metadata

### In Kubernetes/OpenShift Logs

```bash
# Backend logs
oc logs deployment/backend-api -n ambient-code | head -20

# Frontend logs
oc logs deployment/frontend -n ambient-code | head -20

# Operator logs
oc logs deployment/agentic-operator -n ambient-code | head -20

# Runner job logs
oc logs job/session-abc123 -n project-namespace | head -20
```

### Inspecting Container Environment Variables

```bash
# Using podman/docker
podman run --rm vteam-backend:latest env | grep GIT

# In Kubernetes
kubectl exec deployment/backend-api -n ambient-code -- env | grep GIT
```

### Checking Image Labels (optional enhancement)

You can also add this metadata as image labels for inspection without running the container:

```bash
podman inspect vteam-backend:latest | jq '.[0].Config.Labels'
```

## Development Workflow

### Clean Builds

To ensure no cache is used and base images are pulled fresh:

```bash
make build-all BUILD_FLAGS='--no-cache --pull'
```

Or use the VS Code task: **Build All (Podman)** which now includes these flags by default.

### Checking if Your Changes Are Reflected

After building and deploying:

1. Check the build output shows current git info:
   ```
   Building backend...
   Git: feature/my-change@abc123-dirty
   ```

2. Restart the deployment to see new logs:
   ```bash
   oc rollout restart deployment/backend-api -n ambient-code
   oc logs -f deployment/backend-api -n ambient-code
   ```

3. Verify the logged commit matches your current commit:
   ```bash
   git rev-parse --short HEAD
   ```

### Local vs Clean Builds

- **Local builds** (`-dirty` suffix): Built with uncommitted changes
- **CI builds** (clean): Built from committed code in GitHub Actions
- **Production images**: Should always be clean (no `-dirty` suffix)

## CI/CD Integration

The GitHub Actions workflow (`.github/workflows/components-build-deploy.yml`) automatically:

1. Captures git metadata from the commit being built
2. Passes build arguments to image builds
3. Pushes images to `quay.io/ambient_code` with full metadata
4. Tags images with git commit SHA for traceability

Production images are always built from clean commits, so they never have a `-dirty` suffix.

## Troubleshooting

### Build metadata shows "unknown"

**Cause**: Git commands failed during build (not in a git repository, or git not installed)

**Solution**:
- Ensure you're building from within the git repository
- Check that git is installed: `git --version`
- Verify `.git` directory exists in the project root

### Version shows "-dirty" but I committed all changes

**Cause**: There are untracked files or ignored files that were modified

**Check**:
```bash
git status
git diff --quiet && echo "clean" || echo "dirty"
```

**Solution**: Commit or stash all changes before building production images

### Frontend build metadata not showing

**Cause**: Next.js instrumentation not enabled or not using `NEXT_PUBLIC_` prefix

**Verify**:
1. `next.config.js` has `instrumentationHook: true`
2. Environment variables use `NEXT_PUBLIC_` prefix in Dockerfile
3. Frontend was rebuilt after changes

### Build metadata different between components

**Cause**: Components were built at different times or from different commits

**Solution**: Always build all components together:
```bash
make build-all
```

## Best Practices

1. **Always commit before building production images** to avoid `-dirty` suffix
2. **Use `make build-all`** to ensure all components have matching metadata
3. **Check logs after deployment** to verify correct version is running
4. **Include commit SHA in incident reports** for faster debugging
5. **Tag production releases** so version shows `v1.2.3` instead of commit hash

## Related Files

- `Makefile` - Captures git metadata and passes to builds
- `components/*/Dockerfile` - Declares ARGs and sets ENVs
- `components/backend/main.go` - Backend logging
- `components/operator/main.go` - Operator logging
- `components/frontend/src/instrumentation.ts` - Frontend logging
- `components/runners/claude-code-runner/wrapper.py` - Runner logging
- `.vscode/tasks.json` - VS Code build tasks with `--no-cache --pull`

## Future Enhancements

Potential improvements to the build metadata system:

- **Image labels**: Add metadata as OCI image labels for inspection without running
- **API endpoint**: Expose `/version` endpoint returning JSON with all metadata
- **UI display**: Show build version in frontend footer or settings page
- **Sentry integration**: Include version in error reports for better tracking
- **Metrics tags**: Tag Prometheus metrics with git version for correlation
- **Deployment annotations**: Add metadata to Kubernetes deployment annotations

