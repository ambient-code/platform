# Developer Guide

Welcome to the Ambient Code Platform developer guide! This section covers everything you need to contribute to the project.

## 🏁 Getting Started

### Prerequisites
- Go 1.24+ (backend/operator)
- Node.js 20+ (frontend)
- Python 3.11+ (runners)
- Podman or Docker
- kubectl or oc CLI

### Quick Start

1. **Clone the repository:**
   ```bash
   git clone https://github.com/ambient-code/vTeam.git
   cd vTeam
   ```

2. **Set up local environment with Kind (recommended):**
   ```bash
   make kind-up
   # Access at http://localhost:8080
   ```
   
   **Full guide:** [Kind Development](local-development/kind.md)
   
   **Alternatives:** [Minikube](local-development/minikube.md) (older) • [CRC](local-development/crc.md) (OpenShift-specific) • [Comparison](local-development/)

3. **Make your changes and test:**
   ```bash
   make test
   make lint
   ```

4. **Submit a Pull Request**

## 📖 Developer Documentation

### Local Development
- **[Local Development Guide](local-development/)** - Choose your approach
  - [Kind](local-development/kind.md) - **Recommended** (fast, matches CI/CD)
  - [Minikube](local-development/minikube.md) - Older alternative (still supported)
  - [CRC](local-development/crc.md) - OpenShift-specific features only
  - [Hybrid](local-development/hybrid.md) - Run components locally for debugging

### Code Standards
- **[Code Standards](../../CLAUDE.md)** - Comprehensive development standards
  - Backend & Operator standards (Go)
  - Frontend standards (TypeScript/React)
  - Security patterns
  - Error handling

### Component Development
Each component has detailed development documentation:
- [Frontend README](../../components/frontend/README.md) - Next.js development
- [Backend README](../../components/backend/README.md) - Go API development
- [Operator README](../../components/operator/README.md) - Controller development
- [Runner README](../../components/runners/claude-code-runner/README.md) - Python runner

### Testing
- **[Testing Guide](../testing/)** - Comprehensive test documentation
  - [E2E Tests](../../e2e/README.md) - Cypress end-to-end testing
  - Backend tests - Unit, contract, integration tests
  - Frontend tests - Component and E2E testing

## 🏗️ Architecture

**[Architecture Documentation](../architecture/)**
- System design and component interactions
- [Architectural Decision Records (ADRs)](../adr/)
- [System diagrams](../architecture/diagrams/)

**Key Concepts:**
- Custom Resource Definitions (AgenticSession, ProjectSettings, RFEWorkflow)
- Operator reconciliation patterns
- Multi-tenant namespace isolation
- User token authentication

## 🔧 Development Workflow

### 1. Create Feature Branch
```bash
git checkout -b feature/your-feature-name
```

### 2. Make Changes
Follow the established patterns in [CLAUDE.md](../../CLAUDE.md)

### 3. Test Locally
```bash
# Run linters
make lint

# Run tests
make test

# Test locally
make local-up
```

### 4. Submit PR
```bash
git push origin feature/your-feature-name
# Create PR on GitHub
```

See [CONTRIBUTING.md](../../CONTRIBUTING.md) for full workflow details.

## 🛠️ Common Development Commands

### Build
```bash
make build-all              # Build all components
make build-frontend         # Build frontend only
make build-backend          # Build backend only
```

### Local Development
```bash
make local-up               # Start local environment
make local-status           # Check status
make local-logs             # View logs
make local-down             # Stop environment
```

### Testing
```bash
make test                   # Run all tests
make test-e2e               # Run E2E tests
make lint                   # Run linters
```

### Code Quality
```bash
# Go code
cd components/backend
gofmt -w .
go vet ./...
golangci-lint run

# Frontend code
cd components/frontend
npm run lint
npm run build
```

## 🎯 Where to Start

### First-Time Contributors
1. Read [CONTRIBUTING.md](../../CONTRIBUTING.md)
2. Set up local environment with [QUICK_START.md](../../QUICK_START.md)
3. Pick a "good first issue" from GitHub
4. Join the discussion in GitHub Discussions

### Experienced Developers
1. Review [Architecture Documentation](../architecture/)
2. Read [Architectural Decision Records](../adr/)
3. Choose appropriate [Local Development](local-development/) approach
4. Check out component-specific READMEs

## 📚 Additional Resources

- **[API Reference](../api/)** - REST API documentation
- **[Tools](../tools/)** - Optional development tools (Amber)
- **[Agent Personas](../agents/)** - Multi-agent collaboration agents
- **[Troubleshooting](../troubleshooting/)** - Common issues and solutions

## 🆘 Getting Help

- **Questions?** → [GitHub Discussions](https://github.com/ambient-code/vTeam/discussions)
- **Found a bug?** → [Report an Issue](https://github.com/ambient-code/vTeam/issues)
- **Want to chat?** → Check project communication channels
