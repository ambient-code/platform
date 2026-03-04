---
title: "Contributing"
---

The Ambient Code Platform is open source. Whether you are fixing a bug, adding a feature, or improving documentation, contributions are welcome.

## Prerequisites

| Tool | Version | Purpose |
|------|---------|---------|
| **Go** | 1.24+ | Backend, operator, public API |
| **Node.js** | 20+ | Frontend |
| **Python** | 3.11+ | Runner |
| **Docker** | Latest | Container builds |
| **kubectl** | Latest | Cluster access |
| **Kind** | Latest | Local Kubernetes cluster |

---

## Local setup

```bash
# Start a local Kind cluster with all components
make kind-up
```

Once the cluster is running, access the platform at `http://localhost:8080`. Open a workspace and configure your API key in **Project Settings** before creating sessions.

---

## Components

Each component has its own README with build instructions, test commands, and development tips.

| Component | Technology | README |
|-----------|------------|--------|
| Backend | Go + Gin | [components/backend/](https://github.com/ambient-code/platform/tree/main/components/backend) |
| Frontend | NextJS + Shadcn | [components/frontend/](https://github.com/ambient-code/platform/tree/main/components/frontend) |
| Operator | Go + controller-runtime | [components/operator/](https://github.com/ambient-code/platform/tree/main/components/operator) |
| Runner | Python | [components/runners/claude-code-runner/](https://github.com/ambient-code/platform/tree/main/components/runners/claude-code-runner) |
| Public API | Go + Gin | [components/public-api/](https://github.com/ambient-code/platform/tree/main/components/public-api) |

---

## Developer docs

Internal developer documentation lives alongside these docs in [`docs/internal/`](https://github.com/ambient-code/platform/tree/main/docs/internal):

| Section | What's there |
|---------|-------------|
| [Architecture](https://github.com/ambient-code/platform/tree/main/docs/internal/architecture) | System diagrams, component structure, session lifecycle |
| [ADRs](https://github.com/ambient-code/platform/tree/main/docs/internal/adr) | Architectural Decision Records (Kubernetes-native, user token auth, language choices, etc.) |
| [Design](https://github.com/ambient-code/platform/tree/main/docs/internal/design) | Technical design docs (session reconciliation, runner-operator contract, status redesign) |
| [Deployment](https://github.com/ambient-code/platform/tree/main/docs/internal/deployment) | OpenShift deployment, OAuth, git authentication, S3 storage |
| [Integrations](https://github.com/ambient-code/platform/tree/main/docs/internal/integrations) | GitHub App, GitLab, Google Workspace setup |
| [Local dev](https://github.com/ambient-code/platform/tree/main/docs/internal/developer/local-development) | Kind, CRC, Minikube, and hybrid development setup |
| [Testing](https://github.com/ambient-code/platform/tree/main/docs/internal/testing) | E2E testing guide, test overview |
| [Observability](https://github.com/ambient-code/platform/tree/main/docs/internal/observability) | Langfuse, operator metrics, Grafana dashboards |

---

## Contribution guidelines

See [`CONTRIBUTING.md`](https://github.com/ambient-code/platform/blob/main/CONTRIBUTING.md) for the full contribution workflow -- branching strategy, pull request conventions, code standards, and commit message format.
