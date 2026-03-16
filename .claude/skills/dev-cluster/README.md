# Development Cluster Management Skill

A skill for managing Ambient Code Platform development clusters (kind) for testing local changes.

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

## Cluster: Kind

- Fast cluster creation (~30 seconds)
- Uses production Quay.io images by default
- `LOCAL_IMAGES=true` builds and loads from source
- Lightweight single-node cluster
- Aligns with CI/CD setup
- Access via port-forwarding (see `make kind-status` for ports)

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

- `make kind-up` - Create cluster
- `make kind-up LOCAL_IMAGES=true` - Create cluster with locally-built images
- `make kind-down` - Destroy cluster
- `make kind-rebuild` - Rebuild all, reload, restart
- `make kind-port-forward` - Port-forward services
- `make kind-status` - Show cluster status and ports
- `make build-all` - Build all images
- `make build-backend` - Build backend only
- `make build-frontend` - Build frontend only
- `make build-operator` - Build operator only
- `make local-status` - Check pod status
- `make local-logs` - View all logs

## Requirements

- kind installed
- kubectl installed
- podman or docker installed
- Make installed

## See Also

- [Kind Documentation](https://kind.sigs.k8s.io/)
