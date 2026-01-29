# Quick Start Guide

Get Ambient Code Platform running locally in **under 2 minutes** with Kind!

## Prerequisites

### macOS
```bash
# Install tools
brew install kind kubectl docker

# Start Docker Desktop if not running
open -a Docker
```

### Linux
```bash
# Install kubectl
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
chmod +x kubectl
sudo mv kubectl /usr/local/bin/

# Install Kind
curl -Lo ./kind https://kind.sigs.k8s.io/dl/latest/kind-linux-amd64
chmod +x ./kind
sudo mv ./kind /usr/local/bin/kind

# Install Docker (if not installed)
# Ubuntu/Debian: sudo apt-get install docker.io
# Fedora/RHEL: sudo dnf install docker
# Start Docker: sudo systemctl start docker
```

## Start Platform

```bash
# Clone the repository
git clone https://github.com/ambient-code/vTeam.git
cd vTeam

# Start everything
make kind-up
```

**That's it!** The command will:
- Create Kind cluster (~30 seconds)
- Deploy backend, frontend, and operator (~90 seconds)
- Set up ingress and networking
- Start port forwarding automatically

## Access the Application

**Frontend**: http://localhost:8080

Simple! No need to look up IPs or configure anything.

## Verify Everything Works

```bash
# Check status
kubectl get pods -n ambient-code

# Run E2E tests
make test-e2e
```

## Quick Commands

```bash
# View logs
kubectl logs -n ambient-code deployment/backend-api -f
kubectl logs -n ambient-code deployment/frontend -f

# Restart a component
kubectl rollout restart deployment/backend-api -n ambient-code

# Stop everything
make kind-down

# Restart
make kind-up
```

## Configure API Key

After accessing the UI at http://localhost:8080:

1. Create a new project
2. Navigate to Project Settings
3. Add your `ANTHROPIC_API_KEY` under API Keys
4. Create your first agentic session!

## Development Workflow

**Made code changes?**

```bash
# Rebuild and reload
make kind-down
make kind-up
```

## Alternative Local Development Options

**Need OpenShift-specific features?**
- [CRC Setup](docs/developer/local-development/crc.md) - For Routes, BuildConfigs

**Prefer Minikube?**
- [Minikube Guide](docs/developer/local-development/minikube.md) - Older, slower approach

**Need to debug with breakpoints?**
- [Hybrid Development](docs/developer/local-development/hybrid.md) - Run components locally

**Compare all options:**
- [Local Development Comparison](docs/developer/local-development/) - Which to use?

## Troubleshooting

### Docker not running?
```bash
# macOS
open -a Docker

# Linux
sudo systemctl start docker
sudo systemctl enable docker
```

### Port 8080 already in use?
```bash
# Find what's using it
lsof -i :8080

# Kill it or use different port
make kind-down
# Edit port in e2e/scripts/deploy.sh if needed
make kind-up
```

### Pods not starting?
```bash
# Check pod status
kubectl get pods -n ambient-code

# View events
kubectl get events -n ambient-code --sort-by='.lastTimestamp'

# Describe problematic pod
kubectl describe pod <pod-name> -n ambient-code
```

### Complete reset?
```bash
# Delete everything and start fresh
make kind-down
make kind-up
```

### Need help?
```bash
# Check available commands
make help

# View detailed guide
cat docs/developer/local-development/kind.md
```

## What's Next?

1. **Create a project**: Navigate to http://localhost:8080 and create your first project
2. **Configure API key**: Add your Anthropic API key in project settings
3. **Create a session**: Submit a task for AI-powered analysis
4. **Explore the docs**: Check out [docs/](docs/) for comprehensive guides
5. **Run tests**: Try `make test-e2e` to see the full test suite

## Contributing

Want to contribute? See:
- [CONTRIBUTING.md](CONTRIBUTING.md) - Contribution guidelines  
- [docs/developer/](docs/developer/) - Developer guides
- [CLAUDE.md](CLAUDE.md) - Development standards

## Why Kind?

- ⚡ **Fast**: 30-second startup vs 2-3 minutes with Minikube
- 🎯 **CI/CD Match**: Same environment as our GitHub Actions tests
- 💨 **Lightweight**: Lower memory usage
- ✅ **Official**: Used by Kubernetes project itself
- 🔄 **Quick**: Fast to create/destroy clusters for testing

---

**Full Kind guide**: [docs/developer/local-development/kind.md](docs/developer/local-development/kind.md)

**Having issues?** Open an issue on [GitHub](https://github.com/ambient-code/vTeam/issues)
