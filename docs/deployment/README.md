# Deployment Documentation

Guides for deploying the Ambient Code Platform to various environments.

## 🚀 Deployment Guides

### Production Deployment
- **[OpenShift Deployment](../OPENSHIFT_DEPLOY.md)** - Deploy to production OpenShift cluster
- **[OAuth Configuration](../OPENSHIFT_OAUTH.md)** - Set up OpenShift OAuth authentication

### Configuration
- **[Git Authentication](git-authentication.md)** - Configure Git credentials for runners
- **[GitHub App Setup](../GITHUB_APP_SETUP.md)** - GitHub App integration
- **[GitLab Integration](../gitlab-integration.md)** - GitLab configuration

### Observability
- **[Langfuse Deployment](langfuse.md)** - LLM observability and tracing
- **[Operator Metrics](../operator-metrics-visualization.md)** - Operator monitoring (if exists)

### Storage
- **[S3 Storage Configuration](../s3-storage-configuration.md)** - S3-compatible storage setup (if exists)
- **[MinIO Quickstart](../minio-quickstart.md)** - MinIO deployment (if exists)

## 📋 Deployment Checklist

### Prerequisites
- [ ] OpenShift or Kubernetes cluster with admin access
- [ ] Container registry access (or use default `quay.io/ambient_code`)
- [ ] `oc` or `kubectl` CLI configured
- [ ] Anthropic API key or Vertex AI credentials

### Basic Deployment

```bash
# 1. Prepare environment
cp components/manifests/env.example components/manifests/.env
# Edit .env and set ANTHROPIC_API_KEY

# 2. Deploy
make deploy

# 3. Verify
oc get pods -n ambient-code
oc get routes -n ambient-code
```

### Post-Deployment Configuration

1. **Configure Runner Secrets**:
   - Access web UI
   - Navigate to Settings → Runner Secrets
   - Add Anthropic API key

2. **Set Up Git Authentication** (optional):
   - See [Git Authentication Guide](git-authentication.md)
   - Configure per-project or use GitHub App

3. **Enable Observability** (optional):
   - Deploy Langfuse: [Langfuse Guide](langfuse.md)
   - Configure runner to send traces

## 🔧 Deployment Options

### Using Default Images

Fastest deployment using pre-built images from `quay.io/ambient_code`:

```bash
make deploy
```

### Building Custom Images

Build and deploy your own images:

```bash
# Build all images
make build-all CONTAINER_ENGINE=podman

# Push to registry
make push-all REGISTRY=quay.io/your-username

# Deploy with custom images
make deploy CONTAINER_REGISTRY=quay.io/your-username
```

### Custom Namespace

Deploy to a different namespace:

```bash
make deploy NAMESPACE=my-namespace
```

## 🔐 Security Configuration

### Authentication

**Production (Required):**
- OpenShift OAuth with user tokens
- Namespace-scoped RBAC
- No shared credentials

**Local Development (Insecure):**
- Authentication disabled
- Mock tokens accepted
- See [Local Development](../developer/local-development/)

### RBAC

The platform uses namespace-scoped RBAC:
- Each project maps to a Kubernetes namespace
- Users need appropriate permissions in namespace
- Backend uses user tokens (not service account)

See [ADR-0002: User Token Authentication](../adr/0002-user-token-authentication.md)

### Secrets Management

- **API Keys**: Stored in Kubernetes Secrets
- **Git Credentials**: Per-project secrets
- **OAuth Tokens**: Managed by OpenShift OAuth

## 📊 Monitoring & Observability

### Health Checks

```bash
# Backend health
curl https://backend-route/health

# Frontend accessibility
curl https://frontend-route/

# Operator status
oc get pods -n ambient-code -l app=agentic-operator
```

### Logs

```bash
# Backend logs
oc logs -n ambient-code deployment/backend-api -f

# Frontend logs
oc logs -n ambient-code deployment/frontend -f

# Operator logs
oc logs -n ambient-code deployment/agentic-operator -f

# Runner job logs (in project namespaces)
oc logs -n <project-namespace> job/<job-name>
```

### Metrics

- Prometheus-compatible metrics (if configured)
- Langfuse for LLM observability
- OpenShift monitoring integration

## 🧹 Cleanup

### Uninstall Platform

```bash
make clean
```

### Remove Namespace

```bash
oc delete namespace ambient-code
```

### Full Cleanup

```bash
# Uninstall platform
make clean

# Remove CRDs
oc delete crd agenticsessions.vteam.ambient-code
oc delete crd projectsettings.vteam.ambient-code
oc delete crd rfeworkflows.vteam.ambient-code

# Remove cluster-level RBAC
oc delete clusterrole ambient-code-operator
oc delete clusterrolebinding ambient-code-operator
```

## 🆘 Troubleshooting

### Pods Not Starting

```bash
# Check pod status
oc get pods -n ambient-code

# Describe pod for events
oc describe pod <pod-name> -n ambient-code

# View logs
oc logs <pod-name> -n ambient-code
```

### Image Pull Errors

```bash
# Check image pull secrets
oc get deployment backend-api -n ambient-code -o jsonpath='{.spec.template.spec.imagePullSecrets}'

# Verify image exists
podman pull quay.io/ambient_code/vteam_backend:latest
```

### Route Not Accessible

```bash
# Check route
oc get route frontend-route -n ambient-code

# Check service
oc get svc frontend-service -n ambient-code

# Test service directly
oc port-forward svc/frontend-service 3000:3000 -n ambient-code
```

### Operator Not Creating Jobs

```bash
# Check operator logs
oc logs -n ambient-code deployment/agentic-operator -f

# Check CRDs are installed
oc get crd agenticsessions.vteam.ambient-code

# Verify operator has permissions
oc get clusterrolebinding ambient-code-operator
```

## 📚 Related Documentation

- [Architecture Overview](../architecture/) - System design
- [Component Documentation](../../components/) - Component-specific guides
- [Local Development](../developer/local-development/) - Development environments
- [Testing](../testing/) - Test suite documentation

## 🤝 Contributing

When adding deployment features:
- Update relevant deployment guide
- Test on both OpenShift and Kubernetes
- Document any new configuration options
- Update this index

See [CONTRIBUTING.md](../../CONTRIBUTING.md) for full guidelines.
