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

| Component | Path | Technology |
|-----------|------|------------|
| Backend | `components/backend/` | Go + Gin |
| Frontend | `components/frontend/` | NextJS + Shadcn |
| Operator | `components/operator/` | Go + controller-runtime |
| Runner | `components/runners/claude-code-runner/` | Python |
| Public API | `components/public-api/` | Go + Gin |

Each component has its own README with build instructions, test commands, and development tips.

---

## Contribution guidelines

See `CONTRIBUTING.md` in the repository root for the full contribution workflow -- branching strategy, pull request conventions, code standards, and commit message format.
