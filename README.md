# Ambient Code Platform

> Kubernetes-native AI automation platform for intelligent agentic sessions with multi-agent collaboration


## Overview

The **Ambient Code Platform** is an AI automation platform that combines Claude Code CLI with multi-agent collaboration capabilities. The platform enables teams to create and manage intelligent agentic sessions through a modern web interface.

### Key Capabilities

- **Intelligent Agentic Sessions**: AI-powered automation for analysis, research, content creation, and development tasks
- **Multi-Agent Workflows**: Specialized AI agents model realistic software team dynamics
- **Git Provider Support**: Native integration with GitHub and GitLab (SaaS and self-hosted)
- **Kubernetes Native**: Built with Custom Resources, Operators, and proper RBAC for enterprise deployment
- **Real-time Monitoring**: Live status updates and job execution tracking

## 🚀 Quick Start

**Get running locally in under 2 minutes with Kind:**

```bash
make kind-up
# Access at http://localhost:8080
```

**Full guide:** [Kind Local Development](docs/developer/local-development/kind.md)

**Alternative approaches:** [Minikube](docs/developer/local-development/minikube.md) (older) • [CRC](docs/developer/local-development/crc.md) (OpenShift-specific)

## Architecture

The platform consists of containerized microservices orchestrated via Kubernetes:

| Component | Technology | Description |
|-----------|------------|-------------|
| **Frontend** | NextJS + Shadcn | User interface for managing agentic sessions |
| **Backend API** | Go + Gin | REST API for managing Kubernetes Custom Resources |
| **Agentic Operator** | Go | Kubernetes operator that watches CRs and creates Jobs |
| **Claude Code Runner** | Python + Claude Code CLI | Pod that executes AI with multi-agent collaboration |

**Learn more:** [Architecture Documentation](docs/architecture/)

## 📚 Documentation

### For Users
- 📘 [User Guide](docs/user-guide/) - Using the platform
- 🚀 [Deployment Guide](docs/deployment/) - Production deployment

### For Developers
- 🔧 [Contributing Guide](CONTRIBUTING.md) - How to contribute
- 💻 [Developer Guide](docs/developer/) - Development setup and standards
- 🏗️ [Architecture](docs/architecture/) - Technical design and ADRs
- 🧪 [Testing](docs/testing/) - Test suite documentation

### Local Development
- ⚡ **[Kind Development](docs/developer/local-development/kind.md)** - **Recommended** (fastest, used in CI/CD)
- 🔄 **[Local Development Options](docs/developer/local-development/)** - Kind vs Minikube vs CRC
- 📦 **[Minikube Setup](docs/developer/local-development/minikube.md)** - Older approach (still supported)
- 🔴 **[CRC Setup](docs/developer/local-development/crc.md)** - For OpenShift-specific features

### Integrations
- 🔌 [GitHub Integration](docs/integrations/GITHUB_APP_SETUP.md)
- 🦊 [GitLab Integration](docs/integrations/gitlab-integration.md)
- 📁 [Google Workspace](docs/integrations/google-workspace.md)

## 🤖 Amber Automation Tool

**Amber**

- 🤖 **Auto-Fix**: Automated linting/formatting fixes
- 🔧 **Refactoring**: Automated code refactoring tasks
- 🧪 **Test Coverage**: Automated test generation

**Quick Links:**
- [5-Minute Quickstart](docs/amber-quickstart.md)
- [Complete Guide](docs/amber-automation.md)
- [Setup Instructions](AMBER_SETUP.md)

**Note:** Amber is a development tool for this repository and does NOT need to be deployed with the platform.

## 🧩 Components

Each component has its own detailed README:

- [Frontend](components/frontend/) - Next.js web application
- [Backend](components/backend/) - Go REST API
- [Operator](components/operator/) - Kubernetes controller
- [Runners](components/runners/) - AI execution pods
- [Manifests](components/manifests/) - Kubernetes deployment resources

## 🤝 Contributing

We welcome contributions! Please see:

- [CONTRIBUTING.md](CONTRIBUTING.md) - Contribution guidelines
- [CLAUDE.md](CLAUDE.md) - Development standards for AI assistants
- [Code of Conduct](CONTRIBUTING.md#code-of-conduct)

### Quick Development Workflow

```bash
# Fork and clone
git clone https://github.com/YOUR_USERNAME/vTeam.git
cd vTeam

# Create feature branch
git checkout -b feature/amazing-feature

# Make changes and test
make local-up
make test

# Submit PR
git push origin feature/amazing-feature
```

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

**Quick Links:**
[Quick Start](QUICK_START.md) • [User Guide](docs/user-guide/) • [Architecture](docs/architecture/) • [Contributing](CONTRIBUTING.md) • [API Docs](docs/api/)

**Note:** This project was formerly known as "vTeam". Technical artifacts (image names, namespaces, API groups) still use "vteam" for backward compatibility.
