# Ambient Code Platform Documentation

Welcome to the Ambient Code Platform documentation! This site provides comprehensive guides for users, developers, and operators.

## 📖 Documentation Structure

### For Users

**[User Guide](user-guide/)** - Using the Ambient Code Platform
- [Getting Started](user-guide/getting-started.md) - Installation and first session
- [Working with Amber](user-guide/working-with-amber.md) - Automation tool usage

**[Deployment](deployment/)** - Production deployment
- [OpenShift Deployment](deployment/OPENSHIFT_DEPLOY.md)
- [OAuth Configuration](deployment/OPENSHIFT_OAUTH.md)

### For Developers

**[Developer Guide](developer/)** - Contributing and development
- [Local Development](developer/local-development/) - Minikube, Kind, Hybrid approaches
- [Testing Guide](testing/) - Running tests
- [Contributing Guidelines](../CONTRIBUTING.md)

**[Architecture](architecture/)** - Technical design
- Architecture overview and component details
- [Architectural Decision Records (ADRs)](adr/) - Design decisions
- [Diagrams](architecture/diagrams/) - System diagrams

**[Code Standards](../CLAUDE.md)** - Development patterns
- Backend and Operator standards
- Frontend standards
- Security patterns

### Integrations

**[Integrations](integrations/)** - External service connections
- [GitHub Integration](integrations/GITHUB_APP_SETUP.md)
- [GitLab Integration](integrations/gitlab-integration.md)
- [Google Workspace](integrations/google-workspace.md)

### Tools & Utilities

**[Tools](tools/)** - Optional developer tools
- [Amber Automation](tools/amber/) - GitHub issue-to-PR automation

### Reference

**[Reference](reference/)** - Technical reference
- [Glossary](reference/glossary.md) - Terms and definitions
- [API Reference](api/) - REST API documentation

## 🚀 Quick Links

### Getting Started
- New to the platform? → [User Guide](user-guide/getting-started.md)
- Want to contribute? → [Contributing](../CONTRIBUTING.md)
- Need to deploy? → [Deployment Guide](OPENSHIFT_DEPLOY.md)

### Development
- Local setup → [Quick Start](../QUICK_START.md) (Kind, 2 min)
- Running tests → [Testing Guide](testing/)
- Code patterns → [CLAUDE.md](../CLAUDE.md)

### Architecture
- System design → [Architecture](architecture/)
- Design decisions → [ADRs](adr/)
- Component details → [Components](../components/)

## 🛠️ Building the Docs

This documentation is built with MkDocs:

```bash
# Install dependencies
pip install -r requirements-docs.txt

# Serve locally
mkdocs serve
# Open http://127.0.0.1:8000

# Build static site
mkdocs build

# Deploy to GitHub Pages
mkdocs gh-deploy
```

## 📝 Contributing to Documentation

See [Contributing Guidelines](../CONTRIBUTING.md#improve-documentation) for:
- Writing standards
- Preview workflow
- Content guidelines

## 🆘 Getting Help

- **Issues**: [GitHub Issues](https://github.com/ambient-code/vTeam/issues)
- **Discussions**: [GitHub Discussions](https://github.com/ambient-code/vTeam/discussions)
- **Source Code**: [GitHub Repository](https://github.com/ambient-code/vTeam)
