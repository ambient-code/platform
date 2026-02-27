# Ambient Code Platform

> Kubernetes-native AI automation platform for intelligent agentic sessions

## Overview

The Ambient Code Platform combines Claude Code CLI with multi-agent collaboration capabilities. Teams create and manage intelligent agentic sessions through a modern web interface, backed by Kubernetes Custom Resources and operators.

### Key Capabilities

- **Intelligent Agentic Sessions**: AI-powered automation for analysis, research, content creation, and development tasks
- **Multi-Agent Workflows**: Specialized AI agents model realistic software team dynamics
- **Git Provider Support**: Native integration with GitHub and GitLab (SaaS and self-hosted)
- **Kubernetes Native**: Custom Resources, Operators, and proper RBAC for enterprise deployment
- **Real-time Monitoring**: Live status updates and job execution tracking

## Quick Start

See [CONTRIBUTING.md](CONTRIBUTING.md#local-development-setup) for full local development setup with Kind.

```bash
make kind-up
# Access at http://localhost:8080
```

## Architecture

The platform consists of containerized microservices orchestrated via Kubernetes:

| Component | Technology | Description |
|-----------|------------|-------------|
| **Frontend** | NextJS + Shadcn | User interface for managing agentic sessions |
| **Backend API** | Go + Gin | REST API for managing Kubernetes Custom Resources |
| **Operator** | Go | Kubernetes controller that watches CRs and creates Jobs |
| **Runner** | Python + Claude Code CLI | Pod that executes AI with multi-agent collaboration |

```
User Creates Session -> Backend Creates CR -> Operator Spawns Job ->
Pod Runs Claude CLI -> Results Stored in CR -> UI Displays Progress
```

See [docs-internal/architecture/](docs-internal/architecture/) for detailed architecture documentation.

## Documentation

- **User documentation** -- see the [documentation site](docs/) built with Astro Starlight
- **Developer/architecture docs** -- see [docs-internal/](docs-internal/)
- **Component READMEs** -- each component has its own README with development instructions

### Key Links

| Resource | Location |
|----------|----------|
| Contributing | [CONTRIBUTING.md](CONTRIBUTING.md) |
| Development standards | [CLAUDE.md](CLAUDE.md) |
| Developer bookmarks | [BOOKMARKS.md](BOOKMARKS.md) |
| Architecture decisions | [docs-internal/adr/](docs-internal/adr/) |
| Testing | [docs-internal/testing/](docs-internal/testing/) |
| Local dev setup | [docs-internal/developer/local-development/](docs-internal/developer/local-development/) |

## Components

Each component has its own detailed README:

- [Frontend](components/frontend/) -- Next.js web application
- [Backend](components/backend/) -- Go REST API
- [Operator](components/operator/) -- Kubernetes controller
- [Runner](components/runners/claude-code-runner/) -- AI execution pods
- [Public API](components/public-api/) -- Stateless HTTP gateway
- [Manifests](components/manifests/) -- Kubernetes deployment resources

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for contribution guidelines, code standards, and local development setup.

## License

This project is licensed under the MIT License -- see the [LICENSE](LICENSE) file for details.

---

**Note:** This project was formerly known as "vTeam". Technical artifacts (image names, namespaces, API groups) still use "vteam" for backward compatibility.
