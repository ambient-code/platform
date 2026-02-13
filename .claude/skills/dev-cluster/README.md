# Development Cluster Management Skill

A skill for managing Ambient Code Platform development clusters (kind and minikube) for testing local changes.

## Purpose

This skill helps developers efficiently test platform changes in local Kubernetes clusters by:
- Analyzing which components have changed
- Building only the necessary container images
- Managing cluster lifecycle (create, update, destroy)
- Deploying changes and verifying they work
- Troubleshooting deployment issues

## When to Use

Invoke this skill when working on the Ambient Code Platform and you need to:
- Test code changes in a local cluster
- Set up a development environment
- Debug deployment issues
- Iterate quickly on component changes

## Cluster Options

### Kind (Recommended for Quick Testing)
- Fast cluster creation
- Uses production Quay.io images by default
- Lightweight single-node cluster
- Aligns with CI/CD setup
- Access: http://localhost:8080

### Minikube (Recommended for Development)
- Builds images locally from source
- Hot-reload commands for quick iterations
- Full feature set for development
- Access: http://localhost:3000

## Key Features

1. **Smart Change Detection**: Analyzes git status to determine which components need rebuilding
2. **Automated Image Management**: Builds, loads, and deploys images automatically
3. **Cluster Lifecycle Management**: Handles creation, updates, and teardown
4. **Deployment Verification**: Checks pod status and logs after deployment
5. **Troubleshooting Support**: Helps diagnose and fix common issues

## Example Usage

### Quick Test in Kind
```
User: "Test this changeset in kind"
```
The skill will:
1. Detect changed components
2. Build necessary images
3. Create/update kind cluster
4. Deploy changes
5. Verify deployment
6. Provide access information

### Development with Hot Reload
```
User: "Set up minikube for frontend development"
```
The skill will:
1. Create minikube cluster
2. Build all components
3. Configure hot-reload
4. Show how to iterate quickly

### Troubleshooting
```
User: "The backend pod is crash looping"
```
The skill will:
1. Check pod status
2. Get logs
3. Analyze errors
4. Suggest fixes
5. Verify resolution

## Supported Commands

The skill knows all relevant Makefile targets:

**Kind:**
- `make kind-up` - Create cluster
- `make kind-down` - Destroy cluster

**Minikube:**
- `make local-up` - Create cluster with local builds
- `make local-rebuild` - Rebuild all and restart
- `make local-reload-backend` - Hot reload backend only
- `make local-reload-frontend` - Hot reload frontend only
- `make local-reload-operator` - Hot reload operator only
- `make local-status` - Check status
- `make local-logs-*` - View logs
- `make local-clean` - Destroy cluster

**Building:**
- `make build-all` - Build all images
- `make build-backend` - Build backend only
- `make build-frontend` - Build frontend only
- `make build-operator` - Build operator only

## Platform Components

The skill understands all platform components:

| Component | Path | Image | Purpose |
|-----------|------|-------|---------|
| Backend | `components/backend` | `vteam_backend:latest` | API server |
| Frontend | `components/frontend` | `vteam_frontend:latest` | Web UI |
| Operator | `components/operator` | `vteam_operator:latest` | K8s operator |
| Runner | `components/runners/claude-code-runner` | `vteam_claude_runner:latest` | Claude Code runner |
| State Sync | `components/runners/state-sync` | `vteam_state_sync:latest` | S3 persistence |
| Public API | `components/public-api` | `vteam_public_api:latest` | External API |

## Typical Workflow

1. **Make code changes** in one or more components
2. **Invoke the skill** with "test this in kind" or similar
3. **Skill analyzes changes** and builds necessary images
4. **Skill creates/updates cluster** and deploys changes
5. **Skill verifies deployment** and provides access info
6. **Developer tests changes** in the running cluster
7. **Iterate as needed** with hot-reload commands

## Requirements

- Access to /workspace/repos/platform repository
- kind or minikube installed
- kubectl installed
- podman or docker installed
- Make installed

## Troubleshooting

The skill handles common issues:

- **ImagePullBackOff**: Ensures images are loaded with correct pull policy
- **CrashLoopBackOff**: Analyzes logs and suggests fixes
- **Port conflicts**: Helps resolve port forwarding issues
- **Stale images**: Forces rebuild and restart

## Integration Points

This skill integrates with:
- The platform repository Makefile
- Git for change detection
- kubectl for cluster management
- kind/minikube for cluster creation
- Container runtime (podman/docker) for image management

## Best Practices

1. Use kind for quick validation
2. Use minikube for iterative development
3. Always check logs after deployment
4. Clean up clusters when done
5. Build only what changed
6. Verify image pull policy for local images

## See Also

- [Platform Repository README](/workspace/repos/platform/README.md)
- [Platform Makefile](/workspace/repos/platform/Makefile)
- [Kind Documentation](https://kind.sigs.k8s.io/)
- [Minikube Documentation](https://minikube.sigs.k8s.io/)
