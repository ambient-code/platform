# Documentation Map

Quick reference guide to find documentation in the Ambient Code Platform repository.

## 🗺️ Where to Find Things

### Getting Started
| What You Need | Where to Look |
|---------------|---------------|
| **First time setup** | [QUICK_START.md](../QUICK_START.md) (Kind - 2 min) |
| **Contributing** | [CONTRIBUTING.md](../CONTRIBUTING.md) |
| **Project overview** | [README.md](../README.md) |
| **Development standards** | [CLAUDE.md](../CLAUDE.md) |

### Local Development
| What You Need | Where to Look |
|---------------|---------------|
| **Choose local env** | [developer/local-development/](developer/local-development/) |
| **Minikube setup** | [developer/local-development/minikube.md](developer/local-development/minikube.md) |
| **Kind setup** | [developer/local-development/kind.md](developer/local-development/kind.md) |
| **CRC setup** | [developer/local-development/crc.md](developer/local-development/crc.md) |
| **Hybrid dev** | [developer/local-development/hybrid.md](developer/local-development/hybrid.md) |
| **Comparison guide** | [developer/local-development/README.md](developer/local-development/README.md) |

### Component Development
| Component | Documentation |
|-----------|---------------|
| **Frontend** | [components/frontend/README.md](../components/frontend/README.md) |
| **Backend** | [components/backend/README.md](../components/backend/README.md) |
| **Operator** | [components/operator/README.md](../components/operator/README.md) |
| **Runner** | [components/runners/claude-code-runner/README.md](../components/runners/claude-code-runner/README.md) |
| **Manifests** | [components/manifests/README.md](../components/manifests/README.md) |

### Testing
| Test Type | Documentation |
|-----------|---------------|
| **E2E tests** | [e2e/README.md](../e2e/README.md) |
| **Backend tests** | [components/backend/TEST_GUIDE.md](../components/backend/TEST_GUIDE.md) |
| **Testing overview** | [testing/README.md](testing/README.md) |
| **Test suite** | [tests/README.md](../tests/README.md) |

### Architecture
| Topic | Documentation |
|-------|---------------|
| **Overview** | [architecture/README.md](architecture/README.md) |
| **ADRs** | [adr/](adr/) |
| **Diagrams** | [architecture/diagrams/](architecture/diagrams/) |
| **Decisions log** | [decisions.md](decisions.md) |

### Deployment
| Topic | Documentation |
|-------|---------------|
| **Production** | [deployment/OPENSHIFT_DEPLOY.md](deployment/OPENSHIFT_DEPLOY.md) |
| **OAuth** | [deployment/OPENSHIFT_OAUTH.md](deployment/OPENSHIFT_OAUTH.md) |
| **Git Auth** | [deployment/git-authentication.md](deployment/git-authentication.md) |
| **Langfuse** | [deployment/langfuse.md](deployment/langfuse.md) |
| **MinIO** | [deployment/minio-quickstart.md](deployment/minio-quickstart.md) |
| **S3 Storage** | [deployment/s3-storage-configuration.md](deployment/s3-storage-configuration.md) |
| **Deployment Index** | [deployment/README.md](deployment/README.md) |

### Integrations
| Integration | Documentation |
|-------------|---------------|
| **GitHub** | [integrations/GITHUB_APP_SETUP.md](integrations/GITHUB_APP_SETUP.md) |
| **GitLab** | [integrations/gitlab-integration.md](integrations/gitlab-integration.md) |
| **GitLab Token Setup** | [integrations/gitlab-token-setup.md](integrations/gitlab-token-setup.md) |
| **GitLab Self-Hosted** | [integrations/gitlab-self-hosted.md](integrations/gitlab-self-hosted.md) |
| **Google Workspace** | [integrations/google-workspace.md](integrations/google-workspace.md) |
| **All integrations** | [integrations/README.md](integrations/README.md) |

### Tools
| Tool | Documentation |
|------|---------------|
| **Amber automation** | [tools/amber/README.md](tools/amber/README.md) |
| **Amber quickstart** | [amber-quickstart.md](amber-quickstart.md) |
| **Amber full guide** | [amber-automation.md](amber-automation.md) |
| **Amber setup** | [AMBER_SETUP.md](../AMBER_SETUP.md) |

### Agents
| Topic | Documentation |
|-------|---------------|
| **Agent overview** | [agents/README.md](agents/README.md) |
| **Active agents** | [agents/active/](agents/active/) |
| **Archived agents** | [agents/archived/](agents/archived/) |

### Reference
| Topic | Documentation |
|-------|---------------|
| **Glossary** | [reference/glossary.md](reference/glossary.md) |
| **Constitution** | [reference/constitution.md](reference/constitution.md) |
| **Model Pricing** | [reference/model-pricing.md](reference/model-pricing.md) |

### Observability
| Topic | Documentation |
|-------|---------------|
| **Langfuse** | [observability/observability-langfuse.md](observability/observability-langfuse.md) |
| **Operator Metrics** | [observability/operator-metrics-visualization.md](observability/operator-metrics-visualization.md) |
| **Observability Index** | [observability/README.md](observability/README.md) |

## 🎯 Common Scenarios

### "I want to run the platform locally"
→ [QUICK_START.md](../QUICK_START.md) (Kind, 2 minutes)

### "I want to write E2E tests"
→ [developer/local-development/kind.md](developer/local-development/kind.md) (Kind setup)

### "I need OpenShift-specific features"
→ [developer/local-development/crc.md](developer/local-development/crc.md) (CRC setup)

### "I want to understand the architecture"
→ [architecture/README.md](architecture/README.md)

### "I want to contribute code"
→ [CONTRIBUTING.md](../CONTRIBUTING.md) + [CLAUDE.md](../CLAUDE.md)

### "I want to deploy to production"
→ [deployment/OPENSHIFT_DEPLOY.md](deployment/OPENSHIFT_DEPLOY.md)

### "I want to use Amber automation"
→ [amber-quickstart.md](amber-quickstart.md)

### "I want to integrate with GitLab"
→ [integrations/gitlab-integration.md](integrations/gitlab-integration.md)

### "I'm debugging a component"
→ Component README in `components/<component>/README.md`

## 📂 Directory Structure

```
/
├── README.md                          # Navigation hub (111 lines)
├── QUICK_START.md                     # 2-minute Kind setup
├── CONTRIBUTING.md                    # Contribution guidelines
├── CLAUDE.md                          # AI assistant development standards
├── AMBER_SETUP.md                     # Amber configuration (for agent)
├── AGENTS.md                          # Symlink to CLAUDE.md
│
├── docs/                              # All documentation (centralized!)
│   ├── README.md                      # Documentation index
│   │
│   ├── architecture/                  # System design
│   │   ├── README.md
│   │   └── diagrams/
│   │
│   ├── developer/                     # Developer guides
│   │   ├── README.md
│   │   └── local-development/
│   │       ├── README.md (Minikube vs Kind vs CRC vs Hybrid)
│   │       ├── kind.md
│   │       ├── crc.md
│   │       └── hybrid.md
│   │
│   ├── deployment/                    # Deployment guides
│   │   ├── README.md
│   │   ├── git-authentication.md
│   │   └── langfuse.md
│   │
│   ├── testing/                       # Test documentation
│   │   └── README.md
│   │
│   ├── tools/                         # Optional tools
│   │   ├── README.md
│   │   └── amber/
│   │       └── README.md
│   │
│   ├── integrations/                  # External integrations
│   │   ├── README.md
│   │   └── google-workspace.md
│   │
│   ├── agents/                        # Agent personas
│   │   ├── README.md
│   │   ├── active/
│   │   └── archived/
│   │
│   └── archived/                      # Historical docs
│       ├── README.md
│       ├── implementation-plans/
│       └── design-docs/
│
├── components/                        # Component-specific docs ONLY
│   ├── frontend/README.md
│   ├── backend/README.md
│   ├── operator/README.md
│   ├── runners/claude-code-runner/README.md
│   └── manifests/README.md
│
└── e2e/                               # E2E test documentation
    └── README.md
```

## 🔍 Search Tips

### Finding Documentation
```bash
# Search all docs
grep -r "your search term" docs/

# Find by filename
find docs/ -name "*keyword*.md"

# List all READMEs
find docs/ -name "README.md"
```

### Navigation Pattern
1. Start at [docs/README.md](README.md)
2. Navigate to category (architecture, developer, testing, etc.)
3. Each category has a README.md with links
4. Follow links to specific guides

## 📝 Documentation Standards

When creating new documentation:
- **Improve existing docs** rather than creating new files
- **Colocate with code** when component-specific
- **Use docs/ for everything else** - No docs in components/ except component READMEs
- **Use navigation READMEs** to link related docs
- **Archive, don't delete** historical documents
- **Keep root clean** - only cross-cutting docs at root

See [CONTRIBUTING.md](../CONTRIBUTING.md#improve-documentation) for full standards.

---

**Can't find something?** Check [docs/README.md](README.md) or open a GitHub issue.
