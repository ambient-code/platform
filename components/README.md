# Ambient Code Platform Components

This directory contains the core components of the Ambient Code Platform.

See the main [README.md](../README.md) for complete documentation, deployment instructions, and usage examples.

## 📊 Architecture Diagrams

View the platform architecture in detail:
- [Platform Architecture](../docs/platform-architecture.mmd) - Overall system architecture and data flow
- [Component Structure](../docs/component-structure.mmd) - Directory structure and development workflow
- [Agentic Session Flow](../docs/agentic-session-flow.mmd) - Detailed sequence diagram of session execution
- [Deployment Stack](../docs/deployment-stack.mmd) - Technology stack and deployment options

## Component Directory Structure

```
components/
├── frontend/                   # NextJS web interface with Shadcn UI
├── backend/                    # Go API service for Kubernetes CRD management
├── operator/                   # Kubernetes operator (Go)
├── runners/                    # AI runner services
│   └── claude-code-runner/     # Python service running Claude Code CLI with MCP
├── manifests/                  # Kubernetes deployment manifests
└── README.md                   # This documentation
```

## 🎯 Agentic Session Flow

1. **Create Session**: User creates a new agentic session via the web UI
2. **API Processing**: Backend creates an `AgenticSession` Custom Resource in Kubernetes
3. **Job Scheduling**: Operator detects the CR and creates a Kubernetes Job
4. **Execution**: Job runs a pod with AI CLI and Playwright MCP server
5. **Task Execution**: AI executes the specified task using MCP capabilities
6. **Result Storage**: Results are stored back in the Custom Resource
7. **UI Update**: Frontend displays the completed agentic session with results

## ⚡ Quick Start

### Local Development (Recommended)
```bash
# Single command to start everything (minikube)
make local-up

# Or use Kind (recommended for most development)
make kind-up
```

**Prerequisites:**
- Minikube (`brew install minikube`) or Kind (`brew install kind`) + Docker

**What you get:**
- ✅ Complete local development environment
- ✅ Frontend and backend accessible via localhost
- ✅ Backend API working with authentication
- ✅ Ready for project creation and agentic sessions

### Production Deployment
```bash
# Build and push images to your registry
export REGISTRY="your-registry.com"
make build-all push-all REGISTRY=$REGISTRY

# Deploy to OpenShift/Kubernetes
cd components/manifests
CONTAINER_REGISTRY=$REGISTRY ./deploy.sh
```

### Rebuild Components
```bash
make local-rebuild           # Rebuild and reload all components
make local-reload-backend    # Rebuild and reload backend only
make local-reload-frontend   # Rebuild and reload frontend only
make local-reload-operator   # Rebuild and reload operator only
```

## Quick Deploy

From the project root:

```bash
# Deploy with default images
make deploy

# Or deploy to custom namespace
make deploy NAMESPACE=my-namespace
```

For detailed deployment instructions, see [../docs/deployment/OPENSHIFT_DEPLOY.md](../docs/deployment/OPENSHIFT_DEPLOY.md).
